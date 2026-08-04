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
	requireCheckStreamClean(t, events)
}

// ---------------------------------------------------------------------
// R-ATS-020 — the five-row structural violation table (S-ATS-074…079).
// ---------------------------------------------------------------------

// TestProtocolViolation_StructuralViolationTable covers S-ATS-074…079: all
// five rows of R-ATS-020's table, driven from one table-driven test
// (S-ATS-079's own literal requirement), each asserting the common outcome
// (a malformed-response terminal, never a normal completion) plus its own
// row-specific behavior. Row 3's own preceding-delta assertion doubles as
// S-ATS-084's own proof (partial output preserved byte-exact ahead of a
// terminal failure) — a dedicated, separate test would only duplicate it.
//
// Row 5 ("close without open") needs no new production code: its own
// scenario text — "a terminal chunk that carries no usable response
// identity" — is the same empty/absent-identity shape R-ATS-004's
// existing constructor rejection (S-ATS-015, errMalformedIdentity) already
// covers; this row's own subtest proves that coverage extends to a chunk
// that is simultaneously the first AND the terminal one.
func TestProtocolViolation_StructuralViolationTable(t *testing.T) {
	t.Parallel()

	type row struct {
		name       string
		transcript string
		check      func(t *testing.T, events []ai.Event, failure *ai.Failure)
	}

	rows := []row{
		{
			name: "row 1: delta after close (S-ATS-074)",
			transcript: "" +
				"data: {\"id\":\"chatcmpl-r1\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"one\"},\"finish_reason\":null}]}\n\n" +
				"data: {\"id\":\"chatcmpl-r1\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: {\"id\":\"chatcmpl-r1\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"two\"},\"finish_reason\":null}]}\n\n" +
				"data: [DONE]\n\n",
			check: func(t *testing.T, events []ai.Event, _ *ai.Failure) {
				t.Helper()
				var deltas []string
				for _, ev := range events {
					if d, ok := ev.TextDelta(); ok {
						deltas = append(deltas, d.Delta())
					}
				}
				want := []string{"one"}
				if len(deltas) != len(want) || deltas[0] != want[0] {
					t.Errorf("deltas = %q, want exactly %q — no delta carries \"two\" (S-ATS-074)", deltas, want)
				}
			},
		},
		{
			name: "row 2: duplicate close (S-ATS-075)",
			transcript: "" +
				"data: {\"id\":\"chatcmpl-r2\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
				"data: {\"id\":\"chatcmpl-r2\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: {\"id\":\"chatcmpl-r2\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"length\"}]}\n\n" +
				"data: [DONE]\n\n",
			check: func(t *testing.T, events []ai.Event, _ *ai.Failure) {
				t.Helper()
				completions := 0
				for _, ev := range events {
					if ev.Kind() == ai.EventKindCompletion {
						completions++
					}
				}
				if completions > 1 {
					t.Errorf("completion count = %d, want zero or one, never two (S-ATS-075)", completions)
				}
			},
		},
		{
			name: "row 3: second response start (S-ATS-076)",
			transcript: "" +
				"data: {\"id\":\"chatcmpl-A\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"one\"},\"finish_reason\":null}]}\n\n" +
				"data: {\"id\":\"chatcmpl-A\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"two\"},\"finish_reason\":null}]}\n\n" +
				"data: {\"id\":\"chatcmpl-B\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"three\"},\"finish_reason\":null}]}\n\n" +
				"data: [DONE]\n\n",
			check: func(t *testing.T, events []ai.Event, _ *ai.Failure) {
				t.Helper()
				responseStarts := 0
				var deltas []string
				for _, ev := range events {
					if ev.Kind() == ai.EventKindResponseStart {
						responseStarts++
					}
					if d, ok := ev.TextDelta(); ok {
						deltas = append(deltas, d.Delta())
					}
				}
				if responseStarts != 1 {
					t.Errorf("response-start count = %d, want exactly 1 (S-ATS-076)", responseStarts)
				}
				want := []string{"one", "two"}
				if len(deltas) != len(want) {
					t.Fatalf("preceding deltas = %q, want %q byte-exact (S-ATS-076, S-ATS-084)", deltas, want)
				}
				for i := range want {
					if deltas[i] != want[i] {
						t.Errorf("delta[%d] = %q, want %q (S-ATS-076, S-ATS-084)", i, deltas[i], want[i])
					}
				}
			},
		},
		{
			name: "row 4: delta with no open block, absent index (S-ATS-077)",
			transcript: "" +
				"data: {\"id\":\"chatcmpl-r4\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"one\"},\"finish_reason\":null}]}\n\n" +
				"data: {\"id\":\"chatcmpl-r4\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"delta\":{\"content\":\"two\"},\"finish_reason\":null}]}\n\n" +
				"data: [DONE]\n\n",
			check: func(t *testing.T, events []ai.Event, _ *ai.Failure) {
				t.Helper()
				var deltas []string
				for _, ev := range events {
					if d, ok := ev.TextDelta(); ok {
						deltas = append(deltas, d.Delta())
					}
				}
				want := []string{"one"}
				if len(deltas) != len(want) || deltas[0] != want[0] {
					t.Errorf("deltas = %q, want exactly %q — no delta derives from the index-less choice item (S-ATS-077)", deltas, want)
				}
			},
		},
		{
			name:       "row 5: close without open, no usable identity (S-ATS-078)",
			transcript: "data: {\"id\":\"\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n",
			check: func(t *testing.T, _ []ai.Event, failure *ai.Failure) {
				t.Helper()
				if failure.PartialOutput() {
					t.Error("PartialOutput() = true, want false — no output ever preceded this failure (S-ATS-078)")
				}
			},
		},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()

			server := sseServer(t, r.transcript)
			defer server.Close()
			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), validRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			events := drainAll(t, ch)
			if len(events) == 0 {
				t.Fatal("drained zero events, want at least a terminal failure")
			}

			last := events[len(events)-1]
			failure, ok := last.ErrorPayload()
			if !ok {
				// S-ATS-079: every row's assertion must fail if the producer
				// instead emitted a normal completion.
				t.Fatalf("last event carries no ErrorPayload — want a malformed-response terminal, got kind %v (want failure, not a normal completion) (S-ATS-079)", last.Kind())
			}
			if failure.Category() != ai.FailureCategoryMalformedResponse {
				t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-079)", failure.Category())
			}
			requireCheckStreamClean(t, events)

			r.check(t, events, failure)
		})
	}
}

// TestProtocolViolation_NegativeIndexWithContent_MalformedTerminal covers
// R-ATS-020 row 4's OTHER named shape — a negative index, not merely an
// absent one (S-ATS-077's own scenario names only "omits index"; the
// requirement text names both "negative or absent"). A dedicated test
// keeps the five-row table's own row count matching S-ATS-079's literal
// "all five rows" wording while still covering the requirement's full text.
func TestProtocolViolation_NegativeIndexWithContent_MalformedTerminal(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-neg\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"one\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-neg\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":-1,\"delta\":{\"content\":\"two\"},\"finish_reason\":null}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want at least a terminal failure")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload, want a malformed-response terminal for a negative index: %+v", last)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse", failure.Category())
	}
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	if len(deltas) != 1 || deltas[0] != "one" {
		t.Errorf("deltas = %q, want exactly [\"one\"] — no delta derives from the negative-index choice item", deltas)
	}
	requireCheckStreamClean(t, events)
}

// ---------------------------------------------------------------------
// R-ATS-021 — malformed payload of a known type, distinguished from an
// unknown type (S-ATS-080…082).
// ---------------------------------------------------------------------

// TestProtocolViolation_InvalidJSON_MalformedTerminal covers S-ATS-080,
// using the spec's own literal example fixture: truncated, syntactically
// invalid JSON.
func TestProtocolViolation_InvalidJSON_MalformedTerminal(t *testing.T) {
	t.Parallel()

	transcript := "data: {\"object\":\"chat.completion.chunk\",\"choices\":\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-080)", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want a terminal failure (S-ATS-080)")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event carries no ErrorPayload, want a malformed-response terminal (S-ATS-080): %+v", last)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-080)", failure.Category())
	}
	requireCheckStreamClean(t, events)
}

// TestProtocolViolation_UnknownTypeVsBrokenKnownType_DifferentOutcomes
// covers S-ATS-082: an unknown-type frame (object mismatched, R-ATS-017's
// own skip rule) and a broken-known-type frame (object correct, invalid
// JSON, R-ATS-021's own fail rule) are asserted in the SAME test and
// produce DIFFERENT outcomes — the unknown one completes cleanly with its
// adjacent deltas intact, the broken one terminates in a failure.
func TestProtocolViolation_UnknownTypeVsBrokenKnownType_DifferentOutcomes(t *testing.T) {
	t.Parallel()

	t.Run("unknown type: object mismatched, skipped, deltas intact", func(t *testing.T) {
		t.Parallel()

		transcript := "" +
			"data: {\"id\":\"chatcmpl-uk\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"object\":\"chat.completion\",\"id\":\"chatcmpl-uk\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"INTRUDER\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-uk\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"after\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-uk\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n"
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ATS-082)", err)
		}
		events := drainAll(t, ch)

		var deltas []string
		for _, ev := range events {
			if d, ok := ev.TextDelta(); ok {
				deltas = append(deltas, d.Delta())
			}
			if ev.Kind() == ai.EventKindError {
				t.Fatalf("unexpected error event — the unknown-type case must complete cleanly (S-ATS-082): %+v", ev)
			}
		}
		want := []string{"before", "after"}
		if len(deltas) != len(want) {
			t.Fatalf("deltas = %q, want %q (S-ATS-082)", deltas, want)
		}
		last := events[len(events)-1]
		if last.Kind() != ai.EventKindCompletion {
			t.Errorf("last event kind = %v, want EventKindCompletion — the unknown-type case must complete cleanly (S-ATS-082)", last.Kind())
		}
	})

	t.Run("broken known type: object correct, invalid JSON, terminal failure", func(t *testing.T) {
		t.Parallel()

		transcript := "" +
			"data: {\"id\":\"chatcmpl-bk\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"before\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-bk\",\"object\":\"chat.completion.chunk\",\"choices\":\n\n" +
			"data: [DONE]\n\n"
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ATS-082)", err)
		}
		events := drainAll(t, ch)
		if len(events) == 0 {
			t.Fatal("drained zero events, want a terminal failure (S-ATS-082)")
		}
		last := events[len(events)-1]
		if _, ok := last.ErrorPayload(); !ok {
			t.Errorf("last event kind = %v, want a terminal failure — the broken-known-type case must fail, not complete (S-ATS-082)", last.Kind())
		}

		// design.md D8 (verify-report C1): this is the invalid-JSON-mid-
		// block probe shape — the "before" delta mints and holds an open
		// block, then the next frame's broken JSON fails decodeChunk. The
		// producer must close that block before the terminal error.
		requireCheckStreamClean(t, events)
		requireBlockClosedBeforeError(t, events)
	})
}

// ---------------------------------------------------------------------
// R-ATS-022 — no panic on any input, partial output always preserved
// (S-ATS-083…085; S-ATS-084 is covered by the five-row table's own row 3
// assertion above, which already proves preceding deltas survive
// byte-exact ahead of a terminal failure — a separate test would only
// duplicate it).
// ---------------------------------------------------------------------

// TestProtocolViolation_NoPanicAcrossEveryViolatingRow covers S-ATS-083: a
// table of every violating transcript in this node, driven through
// mapperState.applyChunk directly — stream_state_test.go's own established
// "direct seam" (mustChunk + a fresh mapperState) — so a real recover()
// can catch a real panic. A probe wrapping the full Stream() pipeline
// instead would only ever observe the CALLING goroutine: run() executes on
// its own goroutine (go run(...)), and an uncaught panic there crashes the
// whole test binary before any caller-side recover() could ever run — not
// a meaningful safety net.
func TestProtocolViolation_NoPanicAcrossEveryViolatingRow(t *testing.T) {
	t.Parallel()

	rows := []struct {
		name   string
		chunks []string
	}{
		{"row 1: delta after close", []string{
			`{"id":"r1","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"one"},"finish_reason":null}]}`,
			`{"id":"r1","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`{"id":"r1","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"two"},"finish_reason":null}]}`,
		}},
		{"row 2: duplicate close", []string{
			`{"id":"r2","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`{"id":"r2","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"length"}]}`,
		}},
		{"row 3: second response start", []string{
			`{"id":"A","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"one"},"finish_reason":null}]}`,
			`{"id":"B","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"two"},"finish_reason":null}]}`,
		}},
		{"row 4: absent index with content", []string{
			`{"id":"r4","model":"m","object":"chat.completion.chunk","choices":[{"delta":{"content":"one"},"finish_reason":null}]}`,
		}},
		{"row 4b: negative index with content", []string{
			`{"id":"r4b","model":"m","object":"chat.completion.chunk","choices":[{"index":-1,"delta":{"content":"one"},"finish_reason":null}]}`,
		}},
		{"row 5: close without open, no usable identity", []string{
			`{"id":"","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}},
		{"required field missing", []string{
			`{"id":"r6","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		}},
		{"unrecognised finish_reason", []string{
			`{"id":"r7","model":"m","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"quota_burned"}]}`,
		}},
		{"empty frame: no choices, no usage", []string{
			`{"id":"r8","model":"m","object":"chat.completion.chunk","choices":[]}`,
		}},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()

			func() {
				defer func() {
					if rec := recover(); rec != nil {
						t.Errorf("applyChunk panicked: %v (S-ATS-083)", rec)
					}
				}()
				state := &mapperState{}
				for _, c := range r.chunks {
					// The error return, if any, is expected — already
					// proven correct by the table-driven tests above. This
					// probe asserts only the absence of a panic.
					_, _ = state.applyChunk(mustChunk(t, c))
				}
			}()
		})
	}
}
