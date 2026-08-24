// CH-03.1/.2/.3/.4 — the chat archetype's HTTP+SSE surface, served
// but not wired by this package. CH-04's composition root calls
// RegisterRoutes(e *echo.Echo, resolver IdentityResolver, newConv
// ConversationFactory) *Registry with the production IdentityResolver
// and a factory backed by the composition-root-resolved provider.
//
// Four wire endpoints, each a leaf in doc 0005:
//
//	POST   /api/agent/turns                 → handleOpenTurn        (CH-03.1)
//	GET    /api/agent/turns/:id/events      → handleStreamEvents    (CH-03.2)
//	DELETE /api/agent/turns/:id             → handleCancelTurn      (CH-03.3)
//	(any refusal)                           → identityMiddleware    (CH-03.4)
//
// Identity middleware runs BEFORE every handler. Cross-participant
// guard (R-CHS-004.b) runs inside the stream and cancel handlers — a
// turn-id's owner is the participant that opened it, looked up via the
// Registry's (participantID, turnID) → *Conversation mapping.
//
// Errors use the same five-kind envelope the rest of the backend uses
// (validation / conflict / not_found / server). The frontend's
// envelopeToResult (frontend/src/lib/chat-api.ts:118-166) parses them
// without per-kind special-casing.

package chat

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// openTurnRequest is the POST body the client sends. Mirrors
// frontend/src/lib/chat-types.ts:ChatTurnRequest { id, prompt,
// systemHint? }. Only `id` and `prompt` are required for v1;
// systemHint is a reserved seam — accepted, ignored.
type openTurnRequest struct {
	ID         string `json:"id"`
	Prompt     string `json:"prompt"`
	SystemHint string `json:"systemHint,omitempty"`
}

// openTurnResponse mirrors ChatTurnResponse { turnId, streamUrl } —
// the frontend's submitTurn awaits this shape (chat-api.ts:206-211).
type openTurnResponse struct {
	TurnID    string `json:"turnId"`
	StreamURL string `json:"streamUrl"`
}

// errorEnvelope is the canonical error shape (mirrors
// frontend/src/lib/api.ts:96-110 ApiResult's error half + the
// lib/chat-api.envelopeToResult envelope). Fields is omitted when
// empty so the wire stays tidy.
type errorEnvelope struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeError(c *echo.Context, status int, kind, message string, fields map[string]string) error {
	return c.JSON(status, errorEnvelope{
		Error:   kind,
		Message: message,
		Fields:  fields,
	})
}

// identityMiddleware applies the CH-03.4 refusal path. It runs on
// every route this package registers. If the resolver returns
// (nil, _) or (_, false), the middleware writes a 401 with kind=server
// (R-CHS-004.a) and aborts the chain — every downstream handler
// can therefore assume Identity is non-nil.
//
// The Identity is stashed in echo.Context.Set so handlers read it as
// `getIdentity(c)`. Storing on the request context (rather than the
// echo.Context) would also work but breaks Echo's typed Get API and
// the value would need a key — echo.Context.Set/Get is idiomatic.
func identityMiddleware(resolver IdentityResolver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ident, ok := resolver.IdentityFromRequest(c.Request().Context(), c.Request())
			if !ok || ident == nil {
				return writeError(c, http.StatusUnauthorized, "server",
					"identity not resolved", nil)
			}
			c.Set(identityContextKey, ident)
			return next(c)
		}
	}
}

// identityContextKey is the unexported key under which identityMiddleware
// stores the resolved Identity in the echo.Context. Unexported so no
// other package can read or set it — the only reader is getIdentity
// in this file.
const identityContextKey = "chat.identity"

// getIdentity extracts the Identity stored by identityMiddleware.
// Returns (nil, false) if the middleware was bypassed (e.g. a test
// that calls a handler directly). Every handler in this package
// MUST treat (nil, false) as a 401 — defence in depth against a
// future routing mistake.
func getIdentity(c *echo.Context) (Identity, bool) {
	v := c.Get(identityContextKey)
	if v == nil {
		return nil, false
	}
	ident, ok := v.(Identity)
	return ident, ok
}

// HandleOpenTurn is the CH-03.1 POST handler. S-CHS-001.a/b/c.
//
// On a malformed prompt (empty), returns 400 validation WITHOUT
// allocating a Conversation or invoking the provider (S-CHS-001.b).
//
// On a successful POST, calls registry.GetOrCreate(participantID) —
// which can return (nil, true) signaling a 409 inFlight conflict
// (S-CHS-001.c). Drives the turn on the returned Conversation; the
// handler returns the {turnId, streamUrl} pair only after the
// stream has been established.
//
// Why we drive Send before responding: the spec requires "the turn
// is driving before the response is written" so a fast subsequent
// GET to the stream URL already sees the message.start frame.
func HandleOpenTurn(registry *Registry) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ident, _ := getIdentity(c)

		var req openTurnRequest
		if err := c.Bind(&req); err != nil {
			return writeError(c, http.StatusBadRequest, "validation",
				"malformed request body", map[string]string{"body": "invalid JSON"})
		}
		if strings.TrimSpace(req.ID) == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"id is required", map[string]string{"id": "required"})
		}
		if strings.TrimSpace(req.Prompt) == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"prompt is required", map[string]string{"prompt": "required"})
		}

		conv, busy := registry.GetOrCreate(ident.ParticipantID())
		if busy {
			return writeError(c, http.StatusConflict, "conflict",
				"another turn is already in flight for this participant", nil)
		}

		// Send runs a background goroutine that drives the harness to
		// completion — that goroutine's lifetime must outlast the
		// POST request, otherwise the request context cancels when
		// the 200 response is written and the harness reports the
		// run as cancelled. context.WithoutCancel (Go 1.21+) keeps
		// the values (request id, slog attrs) but discards the
		// deadline/cancel signal. The DELETE handler drives the
		// cancellation explicitly via Conversation.Cancel; SSE
		// disconnects are handled by the GET handler's own ctx.
		turnCtx := context.WithoutCancel(c.Request().Context())
		stream, err := conv.Send(turnCtx, req.Prompt)
		if err != nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to start turn", nil)
		}

		// Stash the stream on the registry so the GET and DELETE
		// handlers can find it. We use a per-participant single-slot
		// map protected by the registry's mutex — one in-flight turn
		// per participant, enforced both here and in GetOrCreate's
		// inFlight check above.
		registry.StoreStream(ident.ParticipantID(), req.ID, stream)

		return c.JSON(http.StatusOK, openTurnResponse{
			TurnID:    req.ID,
			StreamURL: streamURLFor(req.ID),
		})
	}
}

// HandleStreamEvents is the CH-03.2 GET handler. S-CHS-002.a/b/c/d.
//
// Behaviour:
//   - 403 not_found if the turn belongs to another participant
//     (R-CHS-004.b)
//   - Already-terminated turns get an immediate `event: turn.end\n\n`
//     followed by a clean close (S-CHS-002.b)
//   - Active turns stream the existing projected WireEvent channel
//     until it closes or the client disconnects (S-CHS-002.c: goroutine
//     exits cleanly on ctx.Done())
//
// Headers: text/event-stream, no-cache, keep-alive, X-Accel-Buffering:no
// (R-CHS-002.d). All four are set via writeSSEHeaders.
func HandleStreamEvents(registry *Registry) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ident, _ := getIdentity(c)
		turnID := c.Param("id")
		if strings.TrimSpace(turnID) == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"turn id is required", map[string]string{"id": "required"})
		}

		stream, ownerID, ok := registry.LoadStream(turnID)
		if !ok {
			// Unknown turn. Surface as 403 not_found — same shape as
			// cross-participant refusal so the existence of the turn
			// is not leaked (R-CHS-004.b: "the refusal does not
			// reveal whether the turn exists").
			return writeError(c, http.StatusForbidden, "not_found",
				"turn not found", nil)
		}
		if ownerID != ident.ParticipantID() {
			// Cross-participant — same 403.
			return writeError(c, http.StatusForbidden, "not_found",
				"turn not found", nil)
		}

		w := c.Response()
		writeSSEHeaders(w)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return writeError(c, http.StatusInternalServerError, "server",
				"streaming unsupported on this transport", nil)
		}

		if stream == nil {
			// Already-terminal: write a single turn.end frame.
			_ = writeFrame(w, flusher, TurnEnd{})
			return nil
		}

		// Live stream: writeFrames drains it and exits on
		// c.Request().Context().Done() — S-CHS-002.c. The handler
		// returns when the stream closes or the client disconnects.
		err := writeFrames(c.Request().Context(), w, flusher, stream)
		// Once we exit (clean close OR client disconnect), mark the
		// turn completed so a subsequent GET can take the
		// already-terminated fast path. We don't ClearStream here so
		// the DELETE handler can still find the entry; DELETE itself
		// clears it.
		registry.MarkStreamCompleted(turnID)
		return err
	}
}

// HandleCancelTurn is the CH-03.3 DELETE handler. S-CHS-003.a/b/c.
//
// On an in-flight turn, calls Conversation.Cancel(); the SSE stream's
// projector (chat/projection.go) will then emit turn.end with
// finishReason absent (R-CHS-002a's cancellation discriminator).
// On an unknown or finished turn, returns 204 (no-op) — the spec
// calls for 200/204; we use 204 because there's no body to send
// (S-CHS-003.b/c).
func HandleCancelTurn(registry *Registry) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ident, _ := getIdentity(c)
		turnID := c.Param("id")
		if strings.TrimSpace(turnID) == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"turn id is required", map[string]string{"id": "required"})
		}

		ownerID, ok := registry.OwnerOf(turnID)
		if !ok || ownerID != ident.ParticipantID() {
			// Unknown or cross-participant — same 204 (R-CHS-004.b:
			// don't leak existence). DELETE is idempotent so a no-op
			// response is honest.
			return c.NoContent(http.StatusNoContent)
		}

		registry.CancelByTurnID(ident.ParticipantID(), turnID)
		registry.ClearStream(turnID)
		return c.NoContent(http.StatusNoContent)
	}
}

// streamURLFor returns the relative path the GET handler mounts.
// Echo receives this via c.Param, so the path includes `:id`.
func streamURLFor(turnID string) string {
	return "/api/agent/turns/" + turnID + "/events"
}

// RegisterRoutes wires every route this file owns onto e. The
// identityMiddleware runs on every route. CH-04 calls this from the
// composition root with the production IdentityResolver and a
// factory that constructs *Conversation from a real provider.
//
// Returns the Registry so the caller (composition root) can hold it
// for graceful shutdown if a future milestone adds one.
func RegisterRoutes(e *echo.Echo, resolver IdentityResolver, newConv ConversationFactory) (*Registry, error) {
	if e == nil {
		return nil, errors.New("chat: RegisterRoutes requires a non-nil *echo.Echo")
	}
	if resolver == nil {
		return nil, errors.New("chat: RegisterRoutes requires a non-nil IdentityResolver")
	}
	if newConv == nil {
		return nil, errors.New("chat: RegisterRoutes requires a non-nil ConversationFactory")
	}
	registry := NewRegistry(newConv)

	api := e.Group("/api/agent", identityMiddleware(resolver))

	api.POST("/turns", HandleOpenTurn(registry))
	api.GET("/turns/:id/events", HandleStreamEvents(registry))
	api.DELETE("/turns/:id", HandleCancelTurn(registry))

	return registry, nil
}
