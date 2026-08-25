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
	"time"

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

// exchangeDTO mirrors chat.Exchange's ten fields (D-7, CH-09
// R-CCS-015 widening) and is the transport projection the GET
// /api/agent/conversations/:id handler emits (R-CRI-001, R-CRI-004).
// The DTO lives in the transport layer (this file) and is a pure
// projection of the port type — the DTO must NOT invent fields
// beyond Exchange (REQ-7 closed-union enforcement on the wire).
// Mirrors frontend/src/lib/chat-types.ts:ExchangeDTO verbatim (the
// TS spec owns the JSON surface; the Go DTO aligns field-for-field).
type exchangeDTO struct {
	Position        int      `json:"position"`
	PromptText      string   `json:"promptText"`
	AssistantText   string   `json:"assistantText"`
	Partial         bool     `json:"partial"`
	TerminalKind    string   `json:"terminalKind"`
	FailureCategory string   `json:"failureCategory"`
	FinishReason    *string  `json:"finishReason,omitempty"`
	MessageIDs      []string `json:"messageIDs"`
	// CH-09 — two new optional fields widen the DTO additively.
		// The frontend mirror (chat-types.ts) gains the same two
		// fields; their omission from the wire (a tool-free turn)
		// is JSON-encoded as missing keys, never null slices — the
		// optional `omitempty` keeps the wire tidy.
	ToolCalls   []toolCallDTO   `json:"toolCalls,omitempty"`
	ToolResults []toolResultDTO `json:"toolResults,omitempty"`
}

// toolCallDTO is the wire projection of chat.ToolCallRecord
// (R-CCS-015, R-CTS-006). JSON keys lowercase per the closed
// ExchangeDTO precedent. Field names match the wire's event
// payload (chat.ToolCallStart.WireCallID / Tool / Arguments) so a
// reload-side replay and a live-stream event agree byte-for-byte.
type toolCallDTO struct {
	WireCallID string `json:"wireCallId"`
	Tool       string `json:"tool"`
	Arguments  string `json:"arguments"`
}

// toolResultDTO is the wire projection of chat.ToolResultRecord
// (R-CCS-015, R-CTS-006). Outcome is the closed three-value
// vocabulary (\"success\" | \"result_failure\" | \"execution_failure\").
// FailureCategory is non-empty ONLY when Outcome == \"execution_failure\"
// (R-CCP-008 / D6 mirror — no provider text).
type toolResultDTO struct {
	WireCallID      string `json:"wireCallId"`
	Tool            string `json:"tool"`
	Outcome         string `json:"outcome"`
	Content         string `json:"content"`
	FailureCategory string `json:"failureCategory"`
}

// conversationSummaryDTO mirrors chat.ConversationSummary's three
// fields (R-CCS-014, R-CRI-004). `lastActivityAt` is rendered as
// RFC3339 — the spec's "ISO8601" form. `turnCount` is the count of
// exchanges recorded under the participant (one conversation per
// participant at v1 — D-1).
type conversationSummaryDTO struct {
	ConversationID string `json:"conversationID"`
	LastActivityAt string `json:"lastActivityAt"`
	TurnCount      int    `json:"turnCount"`
}

// exchangeToDTO projects a recorded Exchange to the wire form.
// finishReason is omitempty via a nil pointer (matches the wire's
// FinishReason ABSENCE for cancelled/failed turns per D-7 / R-CCS-004).
//
// CH-09 (R-CCS-015): ToolCalls and ToolResults are projected with
// the same JSON-key surface as the wire's live ToolCallStart /
// ToolResult events; an empty slice is omitted (omitempty) so a
// tool-free turn's reload surface stays byte-equal to the
// pre-CH-09 shape.
func exchangeToDTO(ex Exchange) exchangeDTO {
	var fr *string
	if ex.FinishReason != nil {
		s := ex.FinishReason.String()
		fr = &s
	}
	dto := exchangeDTO{
		Position:        ex.Position,
		PromptText:      ex.PromptText,
		AssistantText:   ex.AssistantText,
		Partial:         ex.Partial,
		TerminalKind:    ex.TerminalKind.String(),
		FailureCategory: ex.FailureCategory.String(),
		FinishReason:    fr,
		MessageIDs:      ex.MessageIDs,
	}
	if len(ex.ToolCalls) > 0 {
		dto.ToolCalls = make([]toolCallDTO, len(ex.ToolCalls))
		for i, tc := range ex.ToolCalls {
			dto.ToolCalls[i] = toolCallDTO(tc)
		}
	}
	if len(ex.ToolResults) > 0 {
		dto.ToolResults = make([]toolResultDTO, len(ex.ToolResults))
		for i, tr := range ex.ToolResults {
			dto.ToolResults[i] = toolResultDTO(tr)
		}
	}
	return dto
}

// conversationSummaryToDTO projects a ConversationSummary to the wire
// form. `lastActivityAt` becomes RFC3339 so the frontend's
// relative-time helper can consume the timestamp directly.
func conversationSummaryToDTO(s ConversationSummary) conversationSummaryDTO {
	return conversationSummaryDTO{
		ConversationID: s.ConversationID,
		LastActivityAt: s.LastActivityAt.UTC().Format(time.RFC3339),
		TurnCount:      s.TurnCount,
	}
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

// reloadConversationResponse is the wire body of GET
// /api/agent/conversations/:id on the happy path (200 OK). The
// shape is exactly the JSON array of ExchangeDTO — wrapped here as
// a named slice so the JSON encoder emits `[]`, not `null`, on the
// empty case (an authenticated participant with zero recorded
// exchanges never hits this path: that case is unknown → 404, not
// empty → 200; the empty 200 only ever happens with zero rows on
// the list endpoint).
type reloadConversationResponse = []exchangeDTO

// listConversationsResponse is the wire body of GET
// /api/agent/conversations on the happy path (200 OK). A non-nil
// empty slice ensures the JSON encoder emits `[]` not `null` — the
// S-CRI-004 / S-CCS-018 contract ("the list is empty, AND the
// response is a success rather than a not-found").
type listConversationsResponse = []conversationSummaryDTO

// HandleReloadConversation is the CH-08.1 GET handler. R-CRI-001 +
// S-CRI-001 / S-CRI-002.
//
// Behaviour:
//   - 403 not_found if :id does not match the authenticated
//     participant (R-CHS-004.b shape; refuse rather than probe
//     existence)
//   - 404 not_found if :id matches but the store returns
//     ErrConversationNotFound (the participant's first GET before
//     any Append has happened)
//   - 200 []ExchangeDTO on hit; the slice is a fresh projection
//     (no caller-side mutation can corrupt the store — NFR-CCS-004
//     carries forward)
//
// The handler does NOT mutate store state. Reads only; the page's
// Reload button is offline-only at v1 per the deferred register
// (doct 0005's "Resumable mid-turn reconnect" row).
func HandleReloadConversation(store ConversationStore) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ident, _ := getIdentity(c)
		if ident == nil {
			// Defence in depth: the identity middleware aborts the
			// chain with 401 if Identity is missing, so a non-nil
			// ident is the contract. This branch is unreachable in
			// production; a violation here means the routing was
			// changed without also fixing this handler.
			return writeError(c, http.StatusUnauthorized, "server",
				"identity not resolved", nil)
		}
		requestedID := strings.TrimSpace(c.Param("id"))
		if requestedID == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"id is required", map[string]string{"id": "required"})
		}

		// Cross-participant guard (R-CRI-001 + R-CHS-004.b shape).
		// The participant reads ONLY their own conversation. The
		// refusal is 403 not_found — identical to a truly-
		// nonexistent :id, so existence is not leaked.
		if requestedID != ident.ParticipantID() {
			return writeError(c, http.StatusForbidden, "not_found",
				"conversation not found", nil)
		}

		// Same participant — call the store.
		exchanges, err := store.Load(requestedID)
		if err != nil {
			if errors.Is(err, ErrConversationNotFound) {
				return writeError(c, http.StatusNotFound, "not_found",
					"conversation not found", nil)
			}
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to load conversation", nil)
		}

		dtos := make(reloadConversationResponse, len(exchanges))
		for i, ex := range exchanges {
			dtos[i] = exchangeToDTO(ex)
		}
		return c.JSON(http.StatusOK, dtos)
	}
}

// HandleListConversations is the CH-08.2 GET handler. R-CRI-002 +
// S-CRI-003 / S-CRI-004.
//
// Behaviour:
//   - 401 not_found if the identity middleware has no resolved
//     identity (defence in depth; the middleware aborts the chain
//     before this handler runs)
//   - 200 []ConversationSummaryDTO on hit, sorted most-recent-first
//     by the adapter (D-3 — chat_conversations.updated_at DESC)
//   - The empty list is 200 [], NEVER 404 (S-CRI-004 / S-CCS-018
//     verdict: "the response is a success rather than a not-found")
//
// The handler does NOT mutate store state.
func HandleListConversations(store ConversationStore) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ident, _ := getIdentity(c)
		if ident == nil {
			return writeError(c, http.StatusUnauthorized, "server",
				"identity not resolved", nil)
		}

		summaries, err := store.List(ident.ParticipantID())
		if err != nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to list conversations", nil)
		}

		// Defensive copy: store.List already returns a defensive
		// copy (NFR-CCS-004), but projecting to DTOs produces a
		// fresh slice regardless. Empty input → empty output
		// (non-nil) so the JSON encoder emits `[]` not `null`.
		dtos := make(listConversationsResponse, 0, len(summaries))
		for _, s := range summaries {
			dtos = append(dtos, conversationSummaryToDTO(s))
		}
		return c.JSON(http.StatusOK, dtos)
	}
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

// RegisterResumeRoutes wires the CH-08 read surface onto e. Both
// routes go behind the existing `identityMiddleware`; the same one
// the CH-03 routes ride (no second middleware instance). The store
// is the same `ConversationStore` the composition root already holds
// — the resume handlers are read-only over the same adapter.
//
// The CH-03 signature is preserved byte-unchanged (R-CCS-010 closed
// surface); the resume surface is additive (R-CCS-013 + the
// extension-namespace the spec reserves for additive widens). The
// composition root (cmd/chat/main.go) is the single place that
// calls both RegisterRoutes and RegisterResumeRoutes.
func RegisterResumeRoutes(e *echo.Echo, resolver IdentityResolver, store ConversationStore) error {
	if e == nil {
		return errors.New("chat: RegisterResumeRoutes requires a non-nil *echo.Echo")
	}
	if resolver == nil {
		return errors.New("chat: RegisterResumeRoutes requires a non-nil IdentityResolver")
	}
	if store == nil {
		return errors.New("chat: RegisterResumeRoutes requires a non-nil ConversationStore")
	}

	api := e.Group("/api/agent", identityMiddleware(resolver))

	api.GET("/conversations/:id", HandleReloadConversation(store))
	api.GET("/conversations", HandleListConversations(store))

	return nil
}
