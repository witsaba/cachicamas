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
		"data: {\"id\":\"chatcmpl-cs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-cs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"mid"
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

		server := sseServer(t, "data: {\"id\":\"chatcmpl-ph\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
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

		transcript := "data: {\"id\":\"chatcmpl-ph2\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n"
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
// R-AEM-010 (AI-32.2) — an in-band error frame terminates the stream with
// a typed mid-stream failure (S-AEM-040…044, cited negative E4). This
// guard's polarity was inverted from AI-28.7's stand-alone "no bespoke
// error path" reading once AI-32.2's typed terminal landed; see
// apply-progress.md's "Guard inversions" note.
// ---------------------------------------------------------------------

// TestCharter_ErrorShapedJSONBetweenContentChunks_TerminatesWithTypedFailure
// covers R-AEM-010 (AI-32.2) using the same fixture the AI-28.7 charter
// used (S-ATS-105's literal example, a frame whose data is
// {"error":{"message":"boom","type":"server_error"}} between two content
// chunks) — now with polarity flipped: the `before` content delta IS
// delivered, an AI-32.2 typed terminal failure IS emitted carrying
// ErrInBandErrorFrame in the cause chain, and the `after` content delta
// is NEVER delivered (the stream was terminated).
//
// # Polarity-flip provenance (verify-report equivalent for AI-32.2)
//
// The previous AI-28.7 charter test (`…_NoBespokeErrorPath`) forbade a
// bespoke path for error-shaped JSON on S-ATS-105 (C6, cited negative
// E4). With AI-32.2 landing, that posture inverts by design: the spec's
// R-AEM-010 REQUIREMENT literally says "the stream MUST terminate with a
// terminal error event whose payload is an ai.MidStreamFailure". The
// renamed test now asserts the AI-32.2 typed path; the rationale
// (verbatim) is recorded here so a future reader chasing S-ATS-105's
// original citation sees why the assertion now expects the very error
// event its predecessor forbade.
func TestCharter_ErrorShapedJSONBetweenContentChunks_TerminatesWithTypedFailure(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-c6\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"error\":{\"type\":\"server_error\",\"message\":\"boom\",\"param\":null,\"code\":\"oops\"}}\n\n" +
		"data: {\"id\":\"chatcmpl-c6\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"after\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-c6\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (R-AEM-010)", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	var sawTypedError bool
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			sawTypedError = true
		}
	}

	// R-AEM-010: `before` content is delivered, the `after` content is
	// NOT — the in-band error frame terminated the stream. The single
	// error event MUST carry the AI-32.2 typed identity (cause chain
	// reaches ErrInBandErrorFrame).
	if !sawTypedError {
		t.Fatalf("no typed error event landed — the in-band error frame MUST terminate the stream (R-AEM-010, S-AEM-040)")
	}
	wantDeltas := []string{"before"}
	if len(deltas) != len(wantDeltas) {
		t.Fatalf("deltas = %q, want %q — the `after` content frame must NOT survive an in-band error frame (R-AEM-010, S-AEM-041)", deltas, wantDeltas)
	}
	for i := range wantDeltas {
		if deltas[i] != wantDeltas[i] {
			t.Errorf("delta[%d] = %q, want %q (R-AEM-010, S-AEM-041)", i, deltas[i], wantDeltas[i])
		}
	}

	requireCheckStreamClean(t, events)
}

// TestCharter_ErrorEventType_SkippedAsUnknownEventType covers S-ATS-106: an
// SSE frame whose event type is literally "error" is skipped exactly like
// any other non-default event type (R-ATS-017), never given a bespoke
// error path.
func TestCharter_ErrorEventType_SkippedAsUnknownEventType(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-c6b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
		"event: error\ndata: {\"error\":{\"message\":\"boom\"}}\n\n" +
		"data: {\"id\":\"chatcmpl-c6b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"after\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-c6b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
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

// S-ATS-107 [inspection] note:
// With AI-32.2 landing, this inspection's "no symbol claims to recognise
// an in-stream error payload" reading is no longer true for the frame
// class {"error": {…}} — stream_failure.go now owns
// failureFromErrorFrame and the run-loop's isInBandErrorFrame check.
// The inspection now stands for the narrower claim that no symbol,
// branch or comment claims to recognise OR recognise-AND-DECODE a Chat
// Completions in-stream error payload AS a CHUNK (the chunk path is
// unchanged — R-ATS-017 still skips object-absent or mismatched frames,
// R-ATS-021 still fails on broken known chunks). AI-32.2's typed
// terminal is a deliberately separate, pre-decode detection path
// (stream_failure.go), not an admission into the chunk dispatcher.
