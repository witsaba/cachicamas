// AI-28.2 — R-ATS-013: a stream closing without the terminal sentinel is a
// typed terminal error with partial output preserved and flagged
// (S-ATS-048…051). This is the EOF/no-sentinel half of terminal
// discipline; terminal_test.go carries R-ATS-012/014 (the sentinel's own
// recognition and post-sentinel ignoring).
//
// # Coverage of already-implemented behavior, not a new feature
//
// Slice 1's own stream.go already built errIncompleteStream and threaded
// outputPreceded through emitFailure, with its own doc comment naming
// R-ATS-007 landing in AI-28.1.2 as one trigger and this milestone's own
// task list naming this slice as the trigger to give the EOF/no-sentinel
// fallback its real scenario coverage. Every test below passed on first
// execution against the unmodified slice-1/2 code — non-vacuity is proven
// by a staged, reverted mutation instead of a compile-fail RED (this
// milestone's apply-progress record carries the mutation and its output),
// the same disclosed pattern slice 2 used for its own already-satisfied
// cases (bridge_test.go's reverted-encoder mutation).

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestTruncation_TwoContentChunksNoSentinel_ErrorEventWithPrecedingDeltas
// covers S-ATS-048 and S-ATS-049: a connection that closes after two
// content chunks with no sentinel ends in a malformed-response error
// event, preceded by the two byte-exact deltas, with PartialOutput() true
// and Delivery() DeliveryMidStream.
func TestTruncation_TwoContentChunksNoSentinel_ErrorEventWithPrecedingDeltas(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-tr\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"uno\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-tr\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"dos\"},\"finish_reason\":null}]}\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-048)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want the preceding deltas and a terminal failure (S-ATS-048)")
	}

	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload, want a malformed-response terminal (S-ATS-048): %+v", last)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-048)", failure.Category())
	}

	var deltas []string
	for _, ev := range events[:len(events)-1] {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	want := []string{"uno", "dos"}
	if len(deltas) != len(want) {
		t.Fatalf("preceding deltas = %q, want %q byte-exact (S-ATS-048)", deltas, want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-048)", i, deltas[i], want[i])
		}
	}

	// S-ATS-049
	if !failure.PartialOutput() {
		t.Error("PartialOutput() = false, want true (S-ATS-049)")
	}
	if failure.Delivery() != ai.DeliveryMidStream {
		t.Errorf("Delivery() = %v, want DeliveryMidStream (S-ATS-049)", failure.Delivery())
	}

	// design.md D8 (verify-report C1): this is the truncation-mid-block
	// probe shape — two content deltas mint and hold an open block, then
	// the connection ends with no sentinel. The producer must close that
	// block before the terminal error, so a recorded failure stream still
	// satisfies ai.CheckStream's no-unterminated-block invariant.
	requireCheckStreamClean(t, events)
	requireBlockClosedBeforeError(t, events)
}

// TestTruncation_ResponseStartOnlyNoSentinel_PartialOutputFalse covers
// S-ATS-050: a connection that closes after the response start but before
// any content chunk reports PartialOutput() == false while Delivery()
// remains DeliveryMidStream — handover, not first content, decides the
// delivery path.
func TestTruncation_ResponseStartOnlyNoSentinel_PartialOutputFalse(t *testing.T) {
	t.Parallel()

	transcript := "data: {\"id\":\"chatcmpl-ro\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-050)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-050)")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload (S-ATS-050): %+v", last)
	}
	if failure.PartialOutput() {
		t.Error("PartialOutput() = true, want false — ResponseStart alone does not count as output (S-ATS-050)")
	}
	if failure.Delivery() != ai.DeliveryMidStream {
		t.Errorf("Delivery() = %v, want DeliveryMidStream (S-ATS-050)", failure.Delivery())
	}
	requireCheckStreamClean(t, events)
}

// TestTruncation_CutMidDataLine_NoEventFromPendingPartialFrame covers
// S-ATS-051: a connection cut in the middle of a data: line never
// dispatches the pending partial frame as a complete chunk — no event
// derives from it, only the eventual terminal failure.
func TestTruncation_CutMidDataLine_NoEventFromPendingPartialFrame(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-cut\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-cut\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"mid"
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-051)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events (S-ATS-051)")
	}

	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			t.Errorf("unexpected TextDelta %q derived from a never-dispatched partial frame (S-ATS-051)", d.Delta())
		}
	}
	last := events[len(events)-1]
	if _, ok := last.ErrorPayload(); !ok {
		t.Fatalf("last event carries no ErrorPayload, want a terminal failure for the cut connection (S-ATS-051): %+v", last)
	}
	requireCheckStreamClean(t, events)
}
