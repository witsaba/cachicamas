// AI-28.5 (cross-slice) — R-ATS-025…028's charter/citation scenarios: AI-28
// owns failure construction while the decoder only names a category
// (S-ATS-094…097), the charter boundary holds (S-ATS-098…101, all
// [inspection]), every wire-shape claim resolves to a citation
// (S-ATS-102…104, all [inspection]), and no mid-stream error frame is
// recognised or specified (S-ATS-105…107).
//
// Per design's own "Requirement homes" table, R-ATS-025…028 are
// cross-slice: asserted in slice 1 (construction/charter groundwork
// already exercised throughout every prior test file's own use of
// ai.MidStreamFailure/ai.PreStreamFailure) and slice 6 (this file, the
// scenarios that specifically need this node's own protocol-violation and
// object-discriminator machinery to construct a meaningful fixture).
package openaicompat

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// R-ATS-025 — AI-28 owns failure construction; the decoder only names a
// category (S-ATS-094…097).
// ---------------------------------------------------------------------

// TestCharter_TruncatedTranscript_CategoryMatchesOpenAICompatCategory
// covers S-ATS-094: a GENUINELY truncated transcript — cut mid data-line,
// leaving a pending partial frame at EOF, so Decoder.Finish() returns
// ErrTruncated for real (matching truncation_test.go's own S-ATS-051
// shape, not S-ATS-048's "clean SSE framing, sentinel never observed"
// shape, which takes a DIFFERENT cause, errIncompleteStream) — proves
// errors.Is(failure, ai.ErrMalformedResponse) holds AND the failure's own
// Category() equals exactly what Category(ErrTruncated) reports, deriving
// the expected value from the same function under test rather than a
// hardcoded literal.
func TestCharter_TruncatedTranscript_CategoryMatchesOpenAICompatCategory(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-cs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-cs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"mid"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-094)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want a terminal failure (S-ATS-094)")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload, want a malformed-response terminal (S-ATS-094): %+v", last)
	}
	if !errors.Is(failure, ai.ErrMalformedResponse) {
		t.Error("errors.Is(failure, ai.ErrMalformedResponse) = false, want true (S-ATS-094)")
	}
	wantCategory, ok := Category(ErrTruncated)
	if !ok {
		t.Fatal("test premise broken: Category(ErrTruncated) ok = false, want true")
	}
	if failure.Category() != wantCategory {
		t.Errorf("failure.Category() = %v, want %v — the same value Category(ErrTruncated) reports (S-ATS-094)", failure.Category(), wantCategory)
	}
	requireCheckStreamClean(t, events)
}

// TestCharter_PreAndPostHandoverFailures_DeliveryPathsDiffer covers
// S-ATS-095: a pre-handover failure (an already-cancelled context, before
// the carrier is ever returned) and a post-handover failure (a malformed
// transcript, after the carrier is returned) in one table, asserting the
// first reports ai.DeliveryPreStream and the second ai.DeliveryMidStream.
func TestCharter_PreAndPostHandoverFailures_DeliveryPathsDiffer(t *testing.T) {
	t.Parallel()

	t.Run("pre-handover: DeliveryPreStream", func(t *testing.T) {
		t.Parallel()

		server := sseServer(t, "data: {\"id\":\"chatcmpl-ph\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		defer server.Close()
		c := mustClient(t, server.URL)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.Stream(ctx, validRequest(t))
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-ATS-095)", err, err)
		}
		if failure.Delivery() != ai.DeliveryPreStream {
			t.Errorf("Delivery() = %v, want DeliveryPreStream (S-ATS-095)", failure.Delivery())
		}
	})

	t.Run("post-handover: DeliveryMidStream", func(t *testing.T) {
		t.Parallel()

		transcript := "data: {\"id\":\"chatcmpl-ph2\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n"
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ATS-095)", err)
		}
		events := drainAll(t, ch)
		if len(events) == 0 {
			t.Fatal("drained zero events (S-ATS-095)")
		}
		failure, ok := events[len(events)-1].ErrorPayload()
		if !ok {
			t.Fatalf("last event carries no ErrorPayload (S-ATS-095): %+v", events[len(events)-1])
		}
		if failure.Delivery() != ai.DeliveryMidStream {
			t.Errorf("Delivery() = %v, want DeliveryMidStream (S-ATS-095)", failure.Delivery())
		}
		requireCheckStreamClean(t, events)
	})
}

// S-ATS-096 ("a post-handover failure that emitted no content event
// reports Delivery()==DeliveryMidStream and PartialOutput()==false — the
// two facts are independent") is not a new test in this file: it is
// truncation_test.go's own pre-existing
// TestTruncation_ResponseStartOnlyNoSentinel_PartialOutputFalse (R-ATS-013,
// S-ATS-050), which already asserts exactly these two facts on exactly
// this shape (a response-start-only transcript, connection closes with no
// content and no sentinel) — see that test for the assertions; duplicating
// it here would only restate the same fixture and the same two checks.

// S-ATS-097 is [inspection], not [test]: discharged by reading
// ai.FailureCategories()'s own doc comment (provider_failure.go) — "the
// nine-member category vocabulary" — and confirming this milestone's own
// source (chunk.go, stream.go, stream_state.go) never references a
// FailureCategory value beyond the nine already declared, and never calls
// anything that would append a tenth. See tasks.md's own evidence log.

// ---------------------------------------------------------------------
// R-ATS-028 — no mid-stream error frame is recognised or specified
// (S-ATS-105…107, C6).
// ---------------------------------------------------------------------

// TestCharter_ErrorShapedJSONBetweenContentChunks_NoBespokeErrorPath
// covers S-ATS-105, using the spec's own literal example fixture — a frame
// whose data is {"error":{"message":"boom","type":"server_error"}} —
// between two content chunks. That JSON carries no "object" key at all,
// so under this slice's own three-way rule it is skipped as an
// object-absent frame (R-ATS-017 family): exactly what R-ATS-017
// prescribes for the shape, never a bespoke error path.
func TestCharter_ErrorShapedJSONBetweenContentChunks_NoBespokeErrorPath(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-c6\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"error\":{\"message\":\"boom\",\"type\":\"server_error\"}}\n\n" +
		"data: {\"id\":\"chatcmpl-c6\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"after\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-c6\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-105)", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event — no bespoke error path may derive from an error-shaped JSON payload (S-ATS-105, C6): %+v", ev)
		}
	}
	want := []string{"before", "after"}
	if len(deltas) != len(want) {
		t.Fatalf("deltas = %q, want %q (S-ATS-105)", deltas, want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-105)", i, deltas[i], want[i])
		}
	}
}

// TestCharter_ErrorEventType_SkippedAsUnknownEventType covers S-ATS-106: an
// SSE frame whose event type is literally "error" is skipped exactly like
// any other non-default event type (R-ATS-017), never given a bespoke
// error path.
func TestCharter_ErrorEventType_SkippedAsUnknownEventType(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-c6b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
		"event: error\ndata: {\"error\":{\"message\":\"boom\"}}\n\n" +
		"data: {\"id\":\"chatcmpl-c6b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"after\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-c6b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-106)", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event — an \"error\"-typed SSE frame must be skipped as an unknown event type, not given a bespoke path (S-ATS-106): %+v", ev)
		}
	}
	want := []string{"before", "after"}
	if len(deltas) != len(want) {
		t.Fatalf("deltas = %q, want %q (S-ATS-106)", deltas, want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-106)", i, deltas[i], want[i])
		}
	}
}

// S-ATS-107 is [inspection], not [test]: discharged by reading this
// package's shipped source — no symbol, branch or comment in stream.go,
// chunk.go or stream_state.go claims to parse, recognise or map a
// Chat Completions in-stream error payload; every frame that merely looks
// like one is handled entirely by R-ATS-017's skip rule or R-ATS-021's
// fail rule, per this file's own two tests above. See tasks.md's own
// evidence log.
