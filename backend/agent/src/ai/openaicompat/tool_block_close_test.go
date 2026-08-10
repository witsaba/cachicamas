// Tool-call blocks must be closed on EVERY terminal path, not only the
// ones that route through emitFailure (AI-30.1, design.md D7/D8).
//
// emitFailure already closes still-open tool-call blocks via
// state.truncateOpenCalls before sending its terminal ErrorEvent, so the
// decode/feed/EOF failure shapes satisfy ai.CheckStream's
// no-unterminated-block invariant. The in-band error frame path and the
// three inline cancellation recovery paths in run() predate that
// discipline: they closed only the open TEXT block, so a stream that
// failed or was cancelled while a tool-call block was open terminated
// with that block unterminated — an ai.CheckStream violation on a
// stream this package itself produced.
//
// Closure is asserted from the CARRIER's vantage: only a block whose
// ToolCallStart was actually delivered may receive a ToolCallEnd, which
// is why run() tracks confirmed-open tool blocks itself (the same
// reasoning its blockOpen mirror documents) rather than trusting the
// mapper state that can run ahead of the carrier on a lost send.
//
// Scaffolding reuses a_i-33_2_test.go's stall handler and drain helpers
// and stream_test.go's mustClient/validToolCallRequest — no new deps.
package openaicompat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// toolBlockFrames is the two-frame prefix both tests share: an identity
// chunk (ResponseStart + TextBlockStart) followed by a tool-call-start
// chunk with complete identity and one argument fragment, leaving one
// tool-call block open on the carrier.
var toolBlockFrames = []string{
	"data: {\"id\":\"chatcmpl-toolblock\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n",
	"data: {\"id\":\"chatcmpl-toolblock\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_toolblock\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"{\\\"city\"}}]},\"finish_reason\":null}]}\n\n",
}

// requireClosedToolBlockStream asserts the full recorded stream (a) has a
// ToolCallEnd for the block the ToolCallStart opened, (b) passes
// ai.CheckStream with no violation — the no-unterminated-block invariant
// — and (c) terminates with exactly the expected failure category.
func requireClosedToolBlockStream(t *testing.T, events []ai.Event, wantCategory ai.FailureCategory) {
	t.Helper()

	var startBlock, endBlock ai.BlockIndex
	sawStart, sawEnd := false, false
	var failure *ai.Failure
	for _, ev := range events {
		if payload, ok := ev.ToolCallStart(); ok {
			startBlock, sawStart = payload.BlockIndex(), true
		}
		if payload, ok := ev.ToolCallEnd(); ok {
			endBlock, sawEnd = payload.BlockIndex(), true
		}
		if payload, ok := ev.ErrorPayload(); ok {
			failure = payload
		}
	}
	if !sawStart {
		t.Fatalf("no ToolCallStart on the recorded stream — the fixture did not open a tool-call block; kinds = %v", kindsOf(events))
	}
	if !sawEnd {
		t.Errorf("no ToolCallEnd on the recorded stream — the open tool-call block was never closed on this terminal path (AI-30.1, D7/D8); kinds = %v", kindsOf(events))
	} else if endBlock != startBlock {
		t.Errorf("ToolCallEnd block = %d, want %d (the block ToolCallStart opened)", endBlock, startBlock)
	}

	report := ai.CheckStream(events)
	if v := report.Violation(); v != nil {
		t.Errorf("ai.CheckStream violation = %v, want none — this package produced a stream its own checker rejects; kinds = %v", v, kindsOf(events))
	}
	if !report.Terminated() {
		t.Errorf("stream carries no terminal event, want exactly one ErrorEvent")
	}
	if failure == nil {
		t.Fatalf("no ErrorEvent payload on the recorded stream; kinds = %v", kindsOf(events))
	}
	if got := failure.Category(); got != wantCategory {
		t.Errorf("terminal failure category = %v, want %v", got, wantCategory)
	}
}

// TestRun_InBandErrorFrame_ClosesOpenToolCallBlock: a provider that
// reports an in-band error frame while a tool-call block is open. The
// consumer drains normally throughout, so every close send can succeed —
// there is no race here, only the close discipline itself.
func TestRun_InBandErrorFrame_ClosesOpenToolCallBlock(t *testing.T) {
	transcript := toolBlockFrames[0] + toolBlockFrames[1] +
		"data: {\"error\":{\"type\":\"server_error\",\"message\":\"boom\"}}\n\n"

	// Plain write-and-return handler: the response body closes with the
	// handler, so the producer's drain-before-close is not pinned by a
	// held connection (unlike ai33StallHandler, which stalls on purpose).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
	}))
	defer server.Close()

	c := mustClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.Stream(ctx, validToolCallRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout+ai33StallBound)

	requireClosedToolBlockStream(t, rec.Events(), ai.FailureCategoryUnknown)
}

// TestRun_CancelMidStream_ClosesOpenToolCallBlock: the consumer receives
// the three prefix events (ResponseStart, TextBlockStart, ToolCallStart),
// cancels, and KEEPS DRAINING — the S-AEM-052 caller posture — so the
// recovery path's bounded sends can all land. The full stream must still
// satisfy ai.CheckStream: tool-call close, text close, one cancellation
// terminal.
func TestRun_CancelMidStream_ClosesOpenToolCallBlock(t *testing.T) {
	server := httptest.NewServer(ai33StallHandler(toolBlockFrames...))
	defer server.Close()

	c := mustClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.Stream(ctx, validToolCallRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	// Receive the three prefix events (ResponseStart, ToolCallStart,
	// ToolCallDelta); the producer is then blocked in Body.Read on a
	// frame the stalled handler never sends.
	var events []ai.Event
	sawToolCallStart := false
	for i := 0; i < 3; i++ {
		ev := ai33FirstEvent(t, ch, 2*time.Second)
		if ev.Kind() == ai.EventKindToolCallStart {
			sawToolCallStart = true
		}
		events = append(events, ev)
	}
	if !sawToolCallStart {
		t.Fatalf("no ToolCallStart among the first three events, kinds = %v", kindsOf(events))
	}

	cancel()

	rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout+ai33StallBound)
	events = append(events, rec.Events()...)

	requireClosedToolBlockStream(t, events, ai.FailureCategoryCancellation)
}
