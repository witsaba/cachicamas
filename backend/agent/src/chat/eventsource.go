// CH-03.2 — the SSE wire-format writer. Every projected WireEvent is
// rendered to bytes that match `frontend-chat-layer1`'s frozen contract
// (frontend/src/lib/chat-types.ts:158-178). The function takes a Go
// channel and an http.ResponseWriter with its associated http.Flusher;
// it writes one frame per event and flushes after each, breaks on
// <-ctx.Done(), and never writes a partial frame.
//
// Why this lives in chat/eventsource.go and not in database_administrator:
// the wire shape belongs to the chat archetype. db_admin's
// sync_stream_handler.go is a copy-paste with renames inspiration; we
// adopt the channel-driven `fmt.Fprintf` + per-frame `Flush()` pattern
// (ADR 0005 § D1 row 3 substitutability — same archetype position).

package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// wireFrameNames is the sealed mapping from each WireEvent variant to
// its frozen `event:` field value. CH-02's wire.go declares the closed
// WireEvent union; this switch is exhaustive over its members by
// construction (the compiler enforces it once each variant has a case).
//
// Adding a new WireEvent variant triggers a compile error in this
// switch's default branch — a future contributor MUST update the wire
// contract here AND in frontend/src/lib/chat-types.ts together.
func wireFrameName(ev WireEvent) string {
	switch ev.(type) {
	case MessageStart:
		return "message.start"
	case MessageDelta:
		return "message.delta"
	case MessageEnd:
		return "message.end"
	case TurnEnd:
		return "turn.end"
	case Error:
		return "error"
	// CH-09 (R-CTS-004, D-3, D-6) — four new wire frame names for
	// tool-call projection. Lowercase JSON keys on the payload
	// (chat.WriteFrameForTest serialises via encoding/json); the
	// frontend mirror at frontend/src/lib/chat-types.ts:212-279
	// parses the same field names. ToolCallDelta and ToolCallEnd
	// cases exist for forward-compatibility (NFR-CTS-002: a future
	// long-running MCP tool can land here without a wire shape
	// change); v1's projector never emits them.
	case ToolCallStart:
		return "tool.call.start"
	case ToolCallDelta:
		return "tool.call.delta"
	case ToolCallEnd:
		return "tool.call.end"
	case ToolResult:
		return "tool.result"
	// CH-10 (R-CPM-003, D-3) — two new wire frame names for the
	// permission event family. The chat wire's CLOSED 2-value
	// Outcome vocabulary "allow_once" | "deny" (D-12 collapse)
	// carries here; Layer 2's 4-value outcome collapses to 2 at
	// the projector arm in chat/projection.go (D-12). The
	// frontend mirror at frontend/src/lib/chat-types.ts:284-360
	// parses the same field names (REQ-12, REQ-13).
	case PermissionDecisionRequired:
		return "permission.decision.required"
	case PermissionDecisionMade:
		return "permission.decision.made"
	default:
		// Unreachable while WireEvent remains a closed interface; kept
		// to make the exhaustiveness intentional rather than implicit.
		panic(fmt.Sprintf("chat: unknown WireEvent variant %T", ev))
	}
}

// writeSSEHeaders sets the four headers the frozen wire contract
// requires: Content-Type:text/event-stream;charset=utf-8,
// Cache-Control:no-cache, Connection:keep-alive, X-Accel-Buffering:no
// (the last defeats nginx proxy buffering — R-CHS-002.d). The
// charset parameter stops DevTools' Response viewer from
// mis-decoding multi-byte UTF-8 frames as windows-1252 when a chat
// turn carries non-ASCII text (Fix D).
func writeSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
}

// writeFrame serialises one WireEvent as `event: <name>\ndata: <json>\n\n`
// and flushes. Returns any encoding error so the caller can decide
// (typically: log + return from the goroutine; the connection will be
// closed by Echo when the handler returns).
func writeFrame(w http.ResponseWriter, flusher http.Flusher, ev WireEvent) error {
	name := wireFrameName(ev)
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("chat: marshal %s: %w", name, err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// writeFrames ranges the events channel and writes one frame per
// WireEvent until the channel closes or ctx is cancelled. The function
// returns nil on a clean close (the producer sent a terminal event and
// then closed the channel — R-CHS-002.a) and a non-nil error if the
// client disconnected mid-stream (R-CHS-002.c — the goroutine exits
// cleanly so no leak).
//
// The function flushes after every frame so a streaming assistant
// response becomes visible immediately — important for v1's
// "assistant text streams in as it's produced" UX (frontend-chat-layer1
// REQ-1 S-1.a, S-CHS-002.a).
func writeFrames(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, events <-chan WireEvent) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			if err := writeFrame(w, flusher, ev); err != nil {
				return err
			}
		}
	}
}

// WriteFrameForTest exposes writeFrame to the chat_test package so
// per-variant byte-exact assertions stay in the test tree. Production
// callers go through writeFrames, which batches channel drain + frame
// write + flush.
func WriteFrameForTest(w http.ResponseWriter, flusher http.Flusher, ev WireEvent) error {
	return writeFrame(w, flusher, ev)
}

// WriteSSEHeadersForTest exposes writeSSEHeaders to the chat_test
// package so the Content-Type charset assertion (Fix D) lives in the
// test tree without making the production helper exported. The HTTP
// handlers in http.go call writeSSEHeaders before driveStream;
// production callers go through that path unchanged.
func WriteSSEHeadersForTest(w http.ResponseWriter) {
	writeSSEHeaders(w)
}

// WriteFramesForTest exposes writeFrames to the chat_test package so
// multi-event drain assertions stay in the test tree.
func WriteFramesForTest(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, events <-chan WireEvent) error {
	return writeFrames(ctx, w, flusher, events)
}
