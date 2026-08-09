// AI-23.8 — optional-capability conformance cases: CAP-O-01 reasoning,
// CAP-O-02 token counting, CAP-O-03 cache-boundary honoring, and this
// node's own two REQUIRED cases — exhaustive finish reasons and
// absent-vs-zero usage, both exercising CAP-R-03 despite living here
// (R-CNF-014…R-CNF-016, AI-03 §11's worked example).
//
// # The finish-reason hand-list (design D3, NFR-CNF-B)
//
// ai exports no finish-reason enumerator, and this change MUST NOT add
// one (NFR-CNF-B forbids any src/ai edit). The seven values are therefore
// hand-listed below, behind finishReasonDriftGuard: a behavioral probe of
// ai.FinishReason(n).String() walking upward from 1 until it first renders
// "invalid" (the package's own not-a-member placeholder) — the same
// outside-the-package exhaustiveness technique AI-22.2's R-STK-004 already
// established for this codebase. If ai ever gains or loses a member, the
// probed count and this hand-list's length diverge and the drift guard
// fails, naming the discrepancy in either direction.

package agenttest

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

func init() {
	registerConformanceCase("reasoning/whole_blocks_never_leak_into_text", CapReasoningContent, reasoningWholeBlocksCase)
	registerConformanceCase("token_counting/asked_of_the_provider_value", CapTokenCounting, tokenCountingCase)
	registerConformanceCase("cache_boundary/honoring_is_consumer_visible", CapCacheBoundary, cacheBoundaryHonoringCase)
	registerConformanceCase("finish_reason/all_seven_values_reachable_drift_guarded", CapCompletionMetadata, finishReasonExhaustivenessCase)
	registerConformanceCase("usage/absent_vs_zero_distinguishable", CapCompletionMetadata, usageAbsentVsZeroCase)
}

// --- CAP-O-01 reasoning (R-CNF-014) ---

// reasoningWholeBlocksCase proves R-CNF-014. It is reached only when
// Factory.Reasoning is declared true — the runner's own generic
// declared-absent skip (AI-23.1, R-CNF-004) is what produces S-CNF-038's
// "skipped with a report, entry absent" half, and is not reimplemented
// here.
func reasoningWholeBlocksCase(t *testing.T, f Factory) {
	t.Helper()

	t.Run("plain_reasoning_never_leaks_into_text_signature_round_trips", func(t *testing.T) {
		reasoningStart, err := ai.NewReasoningBlockStart(1)
		requireConstructed(t, err, "ai.NewReasoningBlockStart")
		reasoningPayload, ok := reasoningStart.ReasoningBlockStart()
		if !ok {
			t.Fatal("reasoningStart carries no ReasoningBlockStart payload")
		}
		const reasoningMarker = "REASONING_ONLY_MARKER_v1"
		reasoningDelta, err := ai.NewReasoningDelta(reasoningPayload, []byte(reasoningMarker))
		requireConstructed(t, err, "ai.NewReasoningDelta")
		const signature = "signature-bytes-\xff\xfe-round-trip"
		reasoningEnd, err := ai.NewReasoningBlockEnd(reasoningPayload, []byte(signature))
		requireConstructed(t, err, "ai.NewReasoningBlockEnd")

		textStart, err := ai.NewTextBlockStart(2)
		requireConstructed(t, err, "ai.NewTextBlockStart")
		const textMarker = "TEXT_ONLY_MARKER_v1"
		textDelta, err := ai.NewTextDelta(2, textMarker)
		requireConstructed(t, err, "ai.NewTextDelta")
		textEnd, err := ai.NewTextBlockEnd(2)
		requireConstructed(t, err, "ai.NewTextBlockEnd")

		script := Script{Steps: []Step{
			Emit(reasoningStart), Emit(textStart),
			Emit(reasoningDelta), Emit(textDelta),
			Emit(reasoningEnd), Emit(textEnd),
		}}
		subject := f.New(t, script)
		ch, err := subject.Stream(t.Context(), minimalRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure (R-CNF-014)", err)
		}
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		RequireValidStream(t, rec)

		var sawSignature bool
		for _, ev := range rec.Events() {
			switch ev.Kind() {
			case ai.EventKindTextDelta:
				d, _ := ev.TextDelta()
				if strings.Contains(d.Delta(), reasoningMarker) {
					t.Error("a text event's delta contains the reasoning marker, want reasoning walled off from text (S-CNF-035)")
				}
			case ai.EventKindReasoningDelta:
				d, _ := ev.ReasoningDelta()
				if strings.Contains(string(d.Fragment()), textMarker) {
					t.Error("a reasoning event's fragment contains the text marker, want text walled off from reasoning")
				}
			case ai.EventKindReasoningBlockEnd:
				e, _ := ev.ReasoningBlockEnd()
				token, hasToken := e.Token()
				if !hasToken || string(token) != signature {
					t.Errorf("round-tripped signature = (%q, hasToken=%v), want (%q, true) byte-identical (S-CNF-036)", token, hasToken, signature)
				}
				sawSignature = true
			}
		}
		if !sawSignature {
			t.Fatal("no ReasoningBlockEnd event drained, want the signature round-trip to be observable")
		}
	})

	t.Run("redacted_bit_propagates_structurally_through_delta_and_end", func(t *testing.T) {
		redactedStart, err := ai.NewRedactedReasoningBlockStart(1)
		requireConstructed(t, err, "ai.NewRedactedReasoningBlockStart")
		redactedPayload, ok := redactedStart.ReasoningBlockStart()
		if !ok {
			t.Fatal("redactedStart carries no ReasoningBlockStart payload")
		}
		redactedEnd, err := ai.NewReasoningBlockEnd(redactedPayload, []byte("redacted-token"))
		requireConstructed(t, err, "ai.NewReasoningBlockEnd")
		responseStart, err := ai.NewResponseStart("resp-redacted-reasoning", "model-redacted-reasoning")
		requireConstructed(t, err, "ai.NewResponseStart")
		completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		requireConstructed(t, err, "ai.NewCompletion")

		script := Script{Steps: []Step{Emit(responseStart), Emit(redactedStart), Emit(redactedEnd), Emit(completion)}}
		subject := f.New(t, script)
		ch, err := subject.Stream(t.Context(), minimalRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		RequireValidStream(t, rec)

		events := rec.Events()
		requireDrainedKinds(t, events, []ai.EventKind{ai.EventKindResponseStart, ai.EventKindReasoningBlockStart, ai.EventKindReasoningBlockEnd, ai.EventKindCompletion}) // S-CLA-011: exactly four, behind the lifecycle prefix and terminal
		start, ok := events[1].ReasoningBlockStart()
		if !ok || !start.Redacted() {
			t.Errorf("start.Redacted() = %v (ok=%v), want true (S-CNF-037)", start.Redacted(), ok)
		}

		// The redacted bit's propagation is proven structurally, not just at
		// block start: ai's own NewReasoningDelta inherits it from the typed
		// start payload and rejects a non-empty fragment built from a
		// redacted block (R-ARE-013) — this IS the propagation mechanism,
		// asserted per block rather than re-derived (this item's own
		// REFACTOR confirmation).
		if _, err := ai.NewReasoningDelta(redactedPayload, []byte("visible fragment")); err == nil {
			t.Error("ai.NewReasoningDelta on a redacted block's payload accepted a non-empty fragment, want the redacted bit to reject it structurally (S-CNF-037)")
		}
	})
}

// --- CAP-O-02 token counting (R-CNF-015) ---

// tokenCountingCase proves R-CNF-015. It is reached only when
// Factory.TokenCounting is declared true; AI-23.1's own construction-time
// cross-check (crossCheckDeclaredOptionalCapabilities) already fails the
// entry if the subject does not satisfy ai.TokenCounter at all — this
// case additionally ASKS it, through tokenCounterOf (the one shared
// type-assertion site, this item's own REFACTOR confirmation), which a
// type assertion alone cannot prove: an advertised counter that declines
// to answer is failed, not absent (R-CNF-015, advertising binds).
func tokenCountingCase(t *testing.T, f Factory) {
	t.Helper()

	counter, ok := tokenCounterOf(t, f)
	if !ok {
		t.Fatal("agenttest: token_counting case reached without a subject satisfying ai.TokenCounter — AI-23.1's cross-check should have failed the entry first (R-CNF-002)")
	}
	count, err := counter.CountTokens(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: CountTokens returned %v — an advertised counter that declines to answer is failed, not absent (R-CNF-015, S-CNF-041)", err)
	}
	got, present := count.Count()
	if !present {
		t.Error("CountTokens returned a TokenCount reporting absent, want a genuine count from an advertised counter (S-CNF-039)")
	}
	if got < 0 {
		t.Errorf("CountTokens returned a negative count %d, want a non-negative token count", got)
	}
}

// --- CAP-O-03 cache-boundary honoring (R-CNF-016) ---

// cacheBoundaryHonoringCase proves R-CNF-016's optional half. It is
// reached only when Factory.CacheBoundary is declared true (the
// declared-absent half — "recorded absent with a reported skip",
// S-CNF-042 — is the runner's own generic mechanism, not reimplemented
// here). Honoring has no askable seam of its own (only CAP-O-02 is
// askable, R-AMP-017): a provider that honors a request's cache-boundary
// markers reports that as consumer-visible behaviour in its usage
// record's CacheRead/CacheWrite counts (usage.go's own vocabulary) —
// the one place honoring becomes observable through the neutral event
// stream at all.
func cacheBoundaryHonoringCase(t *testing.T, f Factory) {
	t.Helper()

	responseStart, err := ai.NewResponseStart("resp-cache-boundary", "model-cache-boundary")
	requireConstructed(t, err, "ai.NewResponseStart")
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{CacheRead: ai.Tokens(128)})
	requireConstructed(t, err, "ai.NewCompletion")
	script := Script{Steps: []Step{Emit(responseStart), Emit(completion)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure (R-CNF-016)", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	events := rec.Events()
	requireDrainedKinds(t, events, []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion}) // S-CLA-012: exactly two, behind the lifecycle prefix
	comp, ok := events[1].Completion()
	if !ok {
		t.Fatal("event 1 is not a Completion")
	}
	count, present := comp.Usage().CacheRead.Count()
	if !present {
		t.Error("Usage().CacheRead reports absent, want a consumer-visible cache-read count when cache-boundary honoring is declared offered (R-CNF-016, S-CNF-042)")
	}
	if count != 128 {
		t.Errorf("Usage().CacheRead count = %d, want 128", count)
	}
}

// --- Required cases living in this optional-capability node (S-CNF-046) ---

// handListedFinishReasons is the seven-value hand-list this suite carries
// behind finishReasonDriftGuard, in ai.FinishReason's own declaration
// order — the ONLY place this count lives in the package (this item's own
// REFACTOR confirmation).
var handListedFinishReasons = []ai.FinishReason{
	ai.FinishReasonStop,
	ai.FinishReasonLength,
	ai.FinishReasonToolCalls,
	ai.FinishReasonContentFilter,
	ai.FinishReasonRefusal,
	ai.FinishReasonPauseTurn,
	ai.FinishReasonUnknown,
}

// finishReasonDriftGuardAgainst probes ai.FinishReason(n).String() upward
// from 1 until it first renders "invalid" — the package's own not-a-member
// placeholder — and reports that probed count alongside whether it matches
// len(handList). Parameterized over handList (rather than reading the
// package-level var directly) so this package's own tests can prove the
// guard fires in both directions (S-CNF-044) against an artificially
// shrunk or grown list, since a real eighth ai.FinishReason value cannot
// be constructed — the vocabulary is closed.
func finishReasonDriftGuardAgainst(handList []ai.FinishReason) (probedCount int, matches bool) {
	for n := 1; n < 256; n++ {
		if ai.FinishReason(n).String() == "invalid" {
			probedCount = n - 1
			break
		}
	}
	return probedCount, probedCount == len(handList)
}

// finishReasonExhaustivenessCase proves R-CNF-016's first required case:
// all seven finish reasons are reachable on a normally-finished stream and
// each is a closed-vocabulary value (S-CNF-043), with the drift guard
// confirming the hand-list still matches ai's own vocabulary (S-CNF-044).
// Standing is required (S-CNF-046, CAP-R-03) despite living in this
// optional-capability node — proven directly by
// TestConformanceSkeleton_CompletionMetadataStanding_IsRequired (AI-23.1)
// against the same CapCompletionMetadata key this case registers under.
func finishReasonExhaustivenessCase(t *testing.T, f Factory) {
	t.Helper()

	for _, reason := range handListedFinishReasons {
		t.Run(reason.String(), func(t *testing.T) {
			if dialectFinishReasonUnreachable(f, reason) {
				requireFinishReasonUnreachable(t, f, reason)
				return
			}

			responseStart, err := ai.NewResponseStart("resp-finish-reason", "model-finish-reason")
			requireConstructed(t, err, "ai.NewResponseStart")
			completion, err := ai.NewCompletion(reason, ai.Usage{})
			requireConstructed(t, err, "ai.NewCompletion")
			script := Script{Steps: []Step{Emit(responseStart), Emit(completion)}}
			subject := f.New(t, script)
			ch, err := subject.Stream(t.Context(), minimalRequest(t))
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}
			rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
			events := rec.Events()
			requireDrainedKinds(t, events, []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion}) // S-CLA-013: exactly two per subtest, behind the lifecycle prefix
			comp, ok := events[1].Completion()
			if !ok {
				t.Fatal("event 1 is not a Completion")
			}
			if comp.FinishReason() != reason {
				t.Errorf("FinishReason() = %v, want %v (S-CNF-043)", comp.FinishReason(), reason)
			}
			if comp.FinishReason().String() == "invalid" {
				t.Errorf("finish reason %v renders as %q, want a closed-vocabulary value", reason, "invalid")
			}
		})
	}

	if probed, matches := finishReasonDriftGuardAgainst(handListedFinishReasons); !matches {
		t.Errorf("agenttest: finish-reason drift guard: ai package now exposes %d finish-reason value(s), this suite's hand-list carries %d — update handListedFinishReasons to match (R-CNF-016, S-CNF-044)", probed, len(handListedFinishReasons))
	}
}

// dialectFinishReasonUnreachable reports whether f's declared Dialect marks
// reason unreachable on this subject's wire dialect (design D5). A nil
// Dialect declares nothing unreachable — the suite's fully-expressive
// default, unaffected by this seam's addition.
func dialectFinishReasonUnreachable(f Factory, reason ai.FinishReason) bool {
	if f.Dialect == nil {
		return false
	}
	return slices.Contains(f.Dialect.UnreachableFinishReasons, reason)
}

// requireFinishReasonUnreachable proves R-CNF-016's dialect-aware-absence
// half (S-CNF-043, S-CNF-084): scripting reason — declared unreachable on
// f's wire dialect — MUST yield exactly one typed failure terminal and no
// Completion. Unsatisfiable and unviolated, proven rather than skipped
// (AI-29/AI-31 precedent): a subject that DOES manage to produce reason,
// or any other shape besides exactly one typed terminal, still fails here
// — the dialect-aware absence is not available as a general escape.
func requireFinishReasonUnreachable(tb testing.TB, f Factory, reason ai.FinishReason) {
	tb.Helper()

	responseStart, err := ai.NewResponseStart("resp-finish-reason-unreachable", "model-finish-reason-unreachable")
	requireConstructed(tb, err, "ai.NewResponseStart")
	completion, err := ai.NewCompletion(reason, ai.Usage{})
	requireConstructed(tb, err, "ai.NewCompletion")
	script := Script{Steps: []Step{Emit(responseStart), Emit(completion)}}
	subject := f.New(tb, script)
	ch, err := subject.Stream(context.Background(), minimalRequest(tb))
	if err != nil {
		tb.Fatalf("Stream returned %v, want no failure constructing the scenario", err)
	}
	rec := DrainAndRecord(tb, ch, DefaultDrainTimeout)
	events := rec.Events()

	errorCount := 0
	for _, ev := range events {
		if _, ok := ev.Completion(); ok {
			tb.Errorf("finish reason %v — declared unreachable on this dialect — produced a Completion, want the strict gate to reject it as a typed failure instead (R-CNF-016, S-CNF-084: the dialect-aware absence is not available as a general escape)", reason)
		}
		if _, ok := ev.ErrorPayload(); ok {
			errorCount++
		}
	}
	if errorCount != 1 {
		tb.Errorf("finish reason %v: %d typed failure terminal(s) observed, want exactly 1 (R-CNF-016)", reason, errorCount)
		return
	}
	last := events[len(events)-1]
	if _, ok := last.ErrorPayload(); !ok {
		tb.Errorf("last drained event kind = %v, want the typed failure terminal in final position (R-CNF-016)", last.Kind())
	}
}

// usageAbsentVsZeroCase proves R-CNF-016's second required case: an absent
// count and a count reported as zero are distinguishable and neither is
// coerced into the other (S-CNF-045). Standing is required (S-CNF-046,
// CAP-R-03), same as finishReasonExhaustivenessCase above.
func usageAbsentVsZeroCase(t *testing.T, f Factory) {
	t.Helper()

	usage := ai.Usage{Output: ai.Tokens(0)} // Input left at its zero value: absent, never set
	responseStart, err := ai.NewResponseStart("resp-usage-absent-vs-zero", "model-usage-absent-vs-zero")
	requireConstructed(t, err, "ai.NewResponseStart")
	completion, err := ai.NewCompletion(ai.FinishReasonStop, usage)
	requireConstructed(t, err, "ai.NewCompletion")
	script := Script{Steps: []Step{Emit(responseStart), Emit(completion)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure (R-CNF-016)", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	events := rec.Events()
	requireDrainedKinds(t, events, []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion}) // S-CLA-014: exactly two, behind the lifecycle prefix
	comp, ok := events[1].Completion()
	if !ok {
		t.Fatal("event 1 is not a Completion")
	}

	if _, present := comp.Usage().Input.Count(); present {
		t.Error("Usage().Input reports present, want absent — nothing was ever set for it (S-CNF-045)")
	}
	outputCount, outputPresent := comp.Usage().Output.Count()
	if !outputPresent {
		t.Error("Usage().Output reports absent, want present — an explicit Tokens(0) was scripted (S-CNF-045)")
	}
	if outputCount != 0 {
		t.Errorf("Usage().Output count = %d, want 0", outputCount)
	}
}
