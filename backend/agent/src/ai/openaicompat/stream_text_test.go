// AI-28.1.2 — the httptest.Server-driven seam for R-ATS-007/008/010
// (design.md's Testing Strategy: fixtures driven "both through
// httptest.Server and directly through Decoder.Feed+mapper"). chunk_test.go
// and stream_state_test.go cover the direct seam; this file proves the same
// mapping wired end to end through a real Client.Stream call, real HTTP,
// and the real Decoder — the runtime harness this milestone's Work Unit
// Evidence table requires alongside the unit-level tests.
//
// This file does not re-test every S-ATS-0NN number at this layer — that
// would duplicate chunk_test.go/stream_state_test.go's own coverage without
// adding confidence. It proves the wiring itself: multi-chunk delta
// ordering, the six-kind block window, byte-exact reconstruction across a
// split rune, and the finish-reason gate, each end to end exactly once.

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestStream_ThreeContentChunks_DrainsThreeOrderedDeltas covers S-ATS-024
// end to end: a real Client.Stream call over a real httptest.Server
// produces three ordered deltas "Hola", ", ", "mundo".
func TestStream_ThreeContentChunks_DrainsThreeOrderedDeltas(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-a\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hola\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-a\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\", \"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-a\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"mundo\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-a\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-024)", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	want := []string{"Hola", ", ", "mundo"}
	if len(deltas) != len(want) {
		t.Fatalf("drained %d delta(s) = %q, want %d = %q (S-ATS-024)", len(deltas), deltas, len(want), want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-024)", i, deltas[i], want[i])
		}
	}

	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil", report.Violation())
	}
}

// TestStream_TwoContentChunksAndTerminal_EmitsSixEventKinds covers
// S-ATS-028/029 end to end: the drained kinds are exactly ResponseStart,
// TextBlockStart, TextDelta, TextDelta, TextBlockEnd, Completion, and every
// text-block-scoped event reports block index 1.
func TestStream_TwoContentChunksAndTerminal_EmitsSixEventKinds(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"un\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" dos\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-028)", err)
	}
	events := drainAll(t, ch)

	want := []ai.EventKind{
		ai.EventKindResponseStart,
		ai.EventKindTextBlockStart,
		ai.EventKindTextDelta,
		ai.EventKindTextDelta,
		ai.EventKindTextBlockEnd,
		ai.EventKindCompletion,
	}
	if len(events) != len(want) {
		t.Fatalf("drained %d event(s), want exactly %d (S-ATS-028): %+v", len(events), len(want), events)
	}
	for i, ev := range events {
		if ev.Kind() != want[i] {
			t.Errorf("events[%d].Kind() = %v, want %v (S-ATS-028)", i, ev.Kind(), want[i])
		}
	}

	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindTextBlockStart:
			if s, _ := ev.TextBlockStart(); s.Block() != 1 {
				t.Errorf("TextBlockStart.Block() = %d, want 1 (S-ATS-029)", s.Block())
			}
		case ai.EventKindTextDelta:
			if d, _ := ev.TextDelta(); d.Block() != 1 {
				t.Errorf("TextDelta.Block() = %d, want 1 (S-ATS-029)", d.Block())
			}
		case ai.EventKindTextBlockEnd:
			if e, _ := ev.TextBlockEnd(); e.Block() != 1 {
				t.Errorf("TextBlockEnd.Block() = %d, want 1 (S-ATS-029)", e.Block())
			}
		}
	}

	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil", report.Violation())
	}
}

// TestStream_SplitRuneAcrossTwoChunks_ReconstructsByteExact covers
// S-ATS-033/034 end to end: raw invalid-UTF-8 bytes split across two SSE
// data lines survive Client.Stream, real HTTP and the real Decoder
// unmodified.
func TestStream_SplitRuneAcrossTwoChunks_ReconstructsByteExact(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"caf\xc3\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\xa9 shop, abierto\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-033)", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	if len(deltas) != 2 {
		t.Fatalf("drained %d delta(s) = %q, want exactly 2 (S-ATS-033)", len(deltas), deltas)
	}
	if deltas[0] != "caf\xc3" {
		t.Errorf("first delta = %q, want %q unmodified — no replacement character (S-ATS-034)", deltas[0], "caf\xc3")
	}
	const want = "caf\xc3\xa9 shop, abierto"
	if got := deltas[0] + deltas[1]; got != want {
		t.Errorf("concatenated = %q, want %q byte-exact (S-ATS-033)", got, want)
	}
}

// TestStream_ContentThenUnrecognizedFinishReason_TerminatesMalformedWithPartialOutputTrue
// covers R-ATS-010's gate (S-ATS-039) AND proves stream.go's own tracked
// outputPreceded fact (design.md D7, R-AIP-010): once a text-block-scoped
// event has been emitted, a later mid-stream failure reports
// PartialOutput() == true — the deltas already drained are preserved
// (R-ATS-022) rather than discarded because a later chunk failed.
func TestStream_ContentThenUnrecognizedFinishReason_TerminatesMalformedWithPartialOutputTrue(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-d\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-d\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"quota_burned\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-039)", err)
	}
	events := drainAll(t, ch)

	if len(events) == 0 {
		t.Fatal("drained zero events, want at least the preceding delta and a terminal failure")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload, want a malformed-response terminal (S-ATS-039); got kinds: %v", kindsOf(events))
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-039)", failure.Category())
	}
	if !failure.PartialOutput() {
		t.Error("PartialOutput() = false, want true — a text delta already preceded this failure (design.md D7, R-AIP-010)")
	}

	sawDelta := false
	for _, ev := range events[:len(events)-1] {
		if d, ok := ev.TextDelta(); ok {
			sawDelta = true
			if d.Delta() != "partial" {
				t.Errorf("preceding delta = %q, want %q — preserved byte-exact ahead of the terminal failure (R-ATS-022)", d.Delta(), "partial")
			}
		}
	}
	if !sawDelta {
		t.Fatal("no TextDelta event preceded the terminal failure — test premise broken")
	}
}
