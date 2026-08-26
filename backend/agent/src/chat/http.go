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
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/archetype"
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

		// Fix C (observability): the POST returns 200 even when the
		// projected stream's terminal frame is a failure — the
		// failure is in the stream, not the POST envelope. The
		// "chat turn opened" line is the only log entry on the
		// open path that pairs with the projector's "chat turn
		// failed" line on the failure path; without it an operator
		// who reads only the open path cannot correlate a
		// subsequent failure log back to the originating turn.
		slog.InfoContext(turnCtx, "chat turn opened",
			slog.String("participant_id", ident.ParticipantID()),
			slog.String("turn_id", req.ID),
		)

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
//
// Path-param decoding (chat-stack-wiring fix):
//   Echo v5.2.1's c.Param("id") does NOT percent-decode the matched
//   path segment — verified against the running build by sending
//   /conversations/braejan%40proton.me and observing the handler
//   received "braejan%40proton.me" literally, which never matched
//   the cookie-derived ident.ParticipantID() ("braejan@proton.me")
//   and produced a spurious 403 not_found. Pre-CH-08.2 the wire's
//   :id was a UUID with no reserved characters so this was
//   invisible. After the participant_id switched to email
//   (chat/auth_shim.go) the @ in the email becomes a percent-encoded
//   %40 on the wire — every reload would 403 until we decode at the
//   boundary. Same fix is needed for any future reserved character
//   in an email localpart (+, ., internationalised), so we apply
//   url.PathUnescape unconditionally rather than special-casing @.
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
		rawID := strings.TrimSpace(c.Param("id"))
		if rawID == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"id is required", map[string]string{"id": "required"})
		}
		// Decode percent-escapes that the router left intact. A
		// malformed %-sequence returns an error — we treat that as
		// 400 validation rather than silently coercing the malformed
		// string into a participant id (which would 403 anyway).
		requestedID, err := url.PathUnescape(rawID)
		if err != nil {
			return writeError(c, http.StatusBadRequest, "validation",
				"id is malformed", map[string]string{"id": "malformed"})
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

// permissionDecisionRequest is the POST body for the CH-10.1
// reverse-channel endpoint. Body shape is the chat wire's CLOSED
// 2-value vocabulary (R-CPM-003, D-12 collapse):
// `{"outcome": "allow_once" | "deny"}`. Out-of-vocab values are
// refused typed (S-CPM-013).
type permissionDecisionRequest struct {
	Outcome string `json:"outcome"`
}

// permissionDecisionResponse is the 200 OK envelope on the happy
// path (S-CPM-010). The body carries the recorded verdict's
// outcome for client-side confirmation.
type permissionDecisionResponse struct {
	CallID  string `json:"callID"`
	Outcome string `json:"outcome"`
}

// HandlePermissionDecision is the CH-10.1 reverse-channel POST
// handler (R-CPM-004, S-CPM-010..013, S-CPM-017a.5). It receives
// the participant's click on a parked tool's hold row, writes the
// verdict into the chat-owned policy state, and wakes the gate.
//
// Routes: `POST /api/agent/turns/:id/permissions/:callID`
//
// Responses:
//   - 200 OK — verdict recorded; WakeParked returned nil.
//   - 403 not_found — cross-participant (R-CHS-004.b shape).
//   - 404 not_found — unknown callID; WakeParked returned
//     ErrStrayDecision (caller-side race: the parked entry was
//     already woken or never parked).
//   - 409 conflict — second click on the same callID; RecordVerdict
//     returned ErrDecisionAlreadyMade (S-CPM-017a.5).
//   - 422 validation — body outside the closed 2-value vocabulary.
func HandlePermissionDecision(registry *Registry) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ident, _ := getIdentity(c)
		if ident == nil {
			return writeError(c, http.StatusUnauthorized, "server",
				"identity not resolved", nil)
		}

		turnID := c.Param("id")
		callID := strings.TrimSpace(c.Param("callID"))
		if turnID == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"turn id is required", map[string]string{"id": "required"})
		}
		if callID == "" {
			return writeError(c, http.StatusBadRequest, "validation",
				"call id is required", map[string]string{"callID": "required"})
		}

		// Cross-participant guard (R-CHS-004.b shape, mirror of
		// HandleStreamEvents). The turn belongs to the participant
		// who opened it; a foreign click is refused as 403
		// not_found so existence is not leaked.
		ownerID, ok := registry.OwnerOf(turnID)
		if !ok {
			// Unknown turn: surface as 403 not_found so the
			// existence of the turn is not leaked (R-CHS-004.b).
			return writeError(c, http.StatusForbidden, "not_found",
				"turn not found", nil)
		}
		if ownerID != ident.ParticipantID() {
			return writeError(c, http.StatusForbidden, "not_found",
				"turn not found", nil)
		}

		// Parse body. The handler accepts ONLY the closed 2-value
		// vocabulary; out-of-vocab yields 422 validation (S-CPM-013).
		var req permissionDecisionRequest
		if err := c.Bind(&req); err != nil {
			return writeError(c, http.StatusBadRequest, "validation",
				"malformed request body", map[string]string{"body": "invalid JSON"})
		}
		if req.Outcome != "allow_once" && req.Outcome != "deny" {
			return writeError(c, http.StatusUnprocessableEntity, "validation",
				"outcome must be \"allow_once\" or \"deny\"",
				map[string]string{"outcome": req.Outcome})
		}

		// Locate the conversation for this participant.
		target, found := registry.ConversationByTurnID(ident.ParticipantID(), turnID)
		if !found || target == nil {
			//nolint:revive // closed registry surface mirrors CH-03 HandleStreamEvents.
			return writeError(c, http.StatusForbidden, "not_found",
				"conversation not found", nil)
		}

		// Write the verdict into the chat-owned policy state.
		// The policy is *chat.DefaultPermissionPolicy (D-2) at
		// v1; the type assertion reaches the chat-only
		// RecordVerdict method (not on the Layer 2 interface).
		if err := recordVerdict(target, callID, req.Outcome); err != nil {
			if errors.Is(err, ErrDecisionAlreadyMade) {
				return writeError(c, http.StatusConflict, "conflict",
					"decision already made for this call",
					map[string]string{"callID": callID})
			}
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to record verdict", nil)
		}

		// Wake the parked gate. ErrStrayDecision → 404 not_found
		// (caller-side race; the parked entry was already cleared
		// or never existed). Any other error → 500 server.
		if err := target.Scheduler().WakeParked(callID); err != nil {
			if errors.Is(err, agent.ErrStrayDecision) {
				return writeError(c, http.StatusNotFound, "not_found",
					"callID not parked", map[string]string{"callID": callID})
			}
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to wake parked call", nil)
		}

		return c.JSON(http.StatusOK, permissionDecisionResponse{
			CallID:  callID,
			Outcome: req.Outcome,
		})
	}
}

// recordVerdict is the type-assertion wrapper that reaches the
// chat-only RecordVerdict method on *chat.DefaultPermissionPolicy.
// S-CPM-017a.5: the typed sentinel ErrDecisionAlreadyMade surfaces
// here so the handler can map it to 409 conflict.
func recordVerdict(conv *Conversation, callID, outcome string) error {
	policy, ok := conv.harness.Turn.PermissionPolicy.(*DefaultPermissionPolicy)
	if !ok {
		return errors.New("chat: handler requires *chat.DefaultPermissionPolicy; got another implementation")
	}
	return policy.RecordVerdict(callID, outcome)
}

// RegisterPermissionRoutes wires the CH-10.1 reverse-channel route
// onto e. The route rides the existing `identityMiddleware` (no
// second middleware instance) — the identity path is identical
// to the CH-03 / CH-08 surface.
//
// The handler reads the conversation from the Registry (OwnerOf +
// ConversationByTurnID). The production composition root passes
// the same Registry RegisterRoutes returned; tests construct their
// own via chat.NewRegistry(testFactory) and seed it the same way
// (StoreStream + GetOrCreate).
//
// Production wire (from cmd/chat/main.go):
//
//   registry, err := chat.RegisterRoutes(e, resolver, factory)
//   chat.RegisterPermissionRoutes(e, resolver, registry)
//
// Test wire (mountPermissionRoutes helper):
//
//   reg := chat.NewRegistry(testFactory)
//   reg.conversations[pid] = conv
//   reg.StoreStream(pid, "T1", nil) // or similar
//   chat.RegisterPermissionRoutes(e, resolver, reg)
func RegisterPermissionRoutes(e *echo.Echo, resolver IdentityResolver, registry *Registry) error {
	if e == nil {
		return errors.New("chat: RegisterPermissionRoutes requires a non-nil *echo.Echo")
	}
	if resolver == nil {
		return errors.New("chat: RegisterPermissionRoutes requires a non-nil IdentityResolver")
	}
	if registry == nil {
		return errors.New("chat: RegisterPermissionRoutes requires a non-nil *Registry")
	}

	api := e.Group("/api/agent", identityMiddleware(resolver))
	api.POST("/turns/:id/permissions/:callID", HandlePermissionDecision(registry))
	return nil
}

// HandleGetAssistantConfig is the CH-12.1 GET handler (REQ-CACAPI-001).
// Reads the org-scoped AssistantConfig via the supplied archetype.Loader
// and returns the JSON-encoded config. The handler lives in the chat
// package because the URL namespace (/api/chat/...) is chat-specific,
// but the storage is generic — the chat archetype is one consumer of
// the Layer 3 archetype package, not its owner.
//
// v1 scope-key simplification: chat.Identity exposes only ParticipantID;
// for v1 the ArchetypeConfig row is keyed by participantID (a
// single-user workspace maps participant 1:1 to org). The mapping seam
// `participantID → orgID` is deferred until a multi-org workspace lands;
// see tasks.md Open disagreements for the carry-over note.
//
// Status codes (REQ-CACAPI-001):
//
//	200 — config JSON in body (auto-seeded defaults if no row yet)
//	403 — no identity resolved (anonymous / cookie missing)
//	500 — Loader returned a non-NoRows error (response body is the
//	      generic server-error envelope; underlying error string is
//	      logged, not echoed)
//
// The handler does NOT use the package's shared identityMiddleware
// because the spec mandates 403 (not the existing 401). It calls the
// resolver directly so the refusal shape is exact.
func HandleGetAssistantConfig(resolver IdentityResolver, loader archetype.Loader) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if resolver == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"assistant config resolver not wired", nil)
		}
		if loader == nil {
			return writeError(c, http.StatusInternalServerError, "server",
				"assistant config loader not wired", nil)
		}
		ident, resolved := resolver.IdentityFromRequest(c.Request().Context(), c.Request())
		if !resolved || ident == nil {
			return writeError(c, http.StatusForbidden, "server",
				"identity not resolved", nil)
		}
		participantID := ident.ParticipantID()
		if participantID == "" {
			return writeError(c, http.StatusForbidden, "server",
				"identity missing participant id", nil)
		}
    
		cfg, _, lerr := loader.LoadByKindAndOrg(c.Request().Context(), archetype.KindChat, participantID)
		if lerr != nil {
			// Do not echo the underlying error string in the body
			// (information leak). The error is logged via the
			// default slog handler at higher layers; here we
			// surface a generic envelope.
			return writeError(c, http.StatusInternalServerError, "server",
				"failed to load assistant config", nil)
		}
		return c.JSON(http.StatusOK, cfg)
	}
}

// RegisterAssistantConfigRoutes wires the AssistantConfig surface
// (REQ-CACAPI-001) onto e. The GET handler lives at
// /api/chat/assistant/config (the spec's locked URL — note this is a
// separate namespace from the existing /api/agent/... tree, by design:
// the AssistantConfig surface is configuration, not chat wire).
//
// The resolver and loader are closure-captured by the handler; the
// registration function only owns the route mounting.
func RegisterAssistantConfigRoutes(e *echo.Echo, resolver IdentityResolver, loader archetype.Loader) error {
	if e == nil {
		return errors.New("chat: RegisterAssistantConfigRoutes requires a non-nil *echo.Echo")
	}
	if resolver == nil {
		return errors.New("chat: RegisterAssistantConfigRoutes requires a non-nil IdentityResolver")
	}
	if loader == nil {
		return errors.New("chat: RegisterAssistantConfigRoutes requires a non-nil archetype.Loader")
	}
    
	g := e.Group("/api/chat")
	g.GET("/assistant/config", HandleGetAssistantConfig(resolver, loader))
	return nil
}
