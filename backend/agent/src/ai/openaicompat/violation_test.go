// AI-28.5 — protocol-order violations: R-ATS-020 (the five-row structural
// violation table, S-ATS-074…079), R-ATS-021 (malformed payload of a known
// type, distinguished from an unknown type, S-ATS-080…082) and R-ATS-022
// (no panic on any violating input, partial output always preserved,
// S-ATS-083…085).
//
// # The object-discriminator three-way rule (slice-6 coordinator ruling)
//
// Slice 5 implemented isChunk() pragmatically as "PRESENT and MISMATCHED
// only" — absence was accepted as a chunk — disclosed as a deviation from
// design.md D3's literal "≠ chat.completion.chunk (or absent) → skip" text,
// because none of this package's 176 pre-slice-5 fixtures carried the
// object field at all, and the literal rule would have skipped every one
// of them. Slice 6's own precondition commit normalized every existing
// fixture to carry the field, discharging that safety-net conflict, so
// this slice implements the FINAL three-way rule design.md always
// intended: object ABSENT → not a chunk at all → skipped (this file's own
// TestProtocolViolation_AbsentObjectDiscriminator_SkippedBetweenTwoContentChunks,
// citing R-ATS-017's own family — the spec's Definitions section makes the
// discriminator constitutive of "a chunk" in the first place, and
// R-ATS-017's own text — "whose top-level object is not chat.completion.chunk"
// — is trivially true of an absent value too); object PRESENT and
// MISMATCHED → skipped (S-ATS-066, unchanged, landed in slice 5); object
// PRESENT and CORRECT but a C1 required top-level field missing →
// malformed terminal (R-ATS-021, S-ATS-081, this file's own
// TestProtocolViolation_MissingRequiredModelField_MalformedTerminal).
package openaicompat

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// Object-discriminator three-way rule, corrective for slice 5 (R-ATS-017
// family; R-ATS-021, S-ATS-081).
// ---------------------------------------------------------------------

// TestProtocolViolation_AbsentObjectDiscriminator_SkippedBetweenTwoContentChunks
// proves the three-way rule's first branch: a frame whose JSON object omits
// "object" entirely — carrying an adversarial content string of its own —
// is skipped exactly like a present-and-mismatched one (S-ATS-066), never
// applied. The adversarial "INTRUDER" payload matches this package's own
// established non-vacuity discipline (S-ATS-063/066): a naive empty-choices
// no-op fixture would pass whether or not the skip rule fires at all.
func TestProtocolViolation_AbsentObjectDiscriminator_SkippedBetweenTwoContentChunks(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-abs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"alpha\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-abs\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"INTRUDER\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-abs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"omega\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-abs\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
		if ev.Kind() == ai.EventKindError {
			t.Fatalf("unexpected error event, want the object-absent frame silently skipped: %+v", ev)
		}
	}
	want := []string{"alpha", "omega"}
	if len(deltas) != len(want) {
		t.Fatalf("deltas = %q, want %q — an object-absent frame must be skipped, not applied", deltas, want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q", i, deltas[i], want[i])
		}
	}
}

// TestProtocolViolation_MissingRequiredModelField_MalformedTerminal covers
// S-ATS-081: a chunk carrying a correct object discriminator and choices,
// but omitting the required model (C1), on a SECOND chunk — after a first
// chunk already established identity and emitted one delta — yields a
// malformed-response terminal, with the preceding delta preserved
// byte-exact. The offending chunk is deliberately NOT the first one: the
// first chunk's own identity path (ai.NewResponseStart) already rejects an
// empty/absent model there (S-ATS-015) — this scenario's own gap is a
// LATER chunk, which no existing code path re-reads model on at all.
//
// The offending chunk deliberately carries the TERMINAL finish_reason
// ("stop"), not a null one: an earlier draft of this fixture left
// finish_reason null on the offending chunk, which meant no terminal chunk
// was ever observed before the sentinel arrived — a transcript shape D9
// (design.md) already turns into a DIFFERENT malformed terminal ("sentinel
// arrived with no terminal chunk ever observed", via ai.NewCompletion
// rejecting a zero-value FinishReason). That made the test pass for the
// wrong reason — a pre-existing, unrelated mechanism, not the missing
// model field — so the offending chunk here is shaped to BE the terminal
// chunk, closing the request's block cleanly, isolating the missing-model
// concern from D9's own territory.
func TestProtocolViolation_MissingRequiredModelField_MalformedTerminal(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-req\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-req\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-081)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want the preceding delta and a terminal failure (S-ATS-081)")
	}

	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload, want a malformed-response terminal (S-ATS-081): %+v", last)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-081)", failure.Category())
	}
	if !errors.Is(failure, ai.ErrMalformedResponse) {
		t.Error("errors.Is(failure, ai.ErrMalformedResponse) = false, want true (S-ATS-081)")
	}

	var deltas []string
	for _, ev := range events[:len(events)-1] {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	if len(deltas) != 1 || deltas[0] != "before" {
		t.Errorf("preceding deltas = %q, want exactly [\"before\"] preserved byte-exact (S-ATS-081)", deltas)
	}
}
