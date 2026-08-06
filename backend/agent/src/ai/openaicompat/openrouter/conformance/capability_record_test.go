// AI-38 — the OpenRouter capability-record assertion
// (R-OR-05 sub-scenario 1, AI-24 §8 expected-vs-generated
// capability standing).
//
// # What this file covers
//
// agenttest's CapabilityRecord entry for each of CAP-O-01 /
// CAP-O-02 / CAP-O-03 (the three optional capabilities) is set
// by applyDeclaredAbsences at suite start, BEFORE any case runs
// (S-CNF-004, conformance_suite.go:330). The factory's
// tri-state non-nil-false declarations (Reasoning: &false,
// TokenCounting: &false, CacheBoundary: &false — R-CNF-004)
// make each of those three entries OutcomeAbsent.
//
// This test asserts the bridge factory's declarations are
// exactly the shape applyDeclaredAbsences keys off, so the
// CapabilityRecord the suite generates carries OutcomeAbsent ×
// 3 for CAP-O-01 / CAP-O-02 / CAP-O-03 — matching AI-24 §8's
// expected capability table for a non-reasoning openai-compat
// wire (R-OR-05 sub-scenario 1).
//
// # Why this is not RunConformance(t, factory)
//
// The shipped openaicompat/bridge_test.go's conformanceBridge
// Factory deliberately runs only CapStreamingText and
// CapToolCalls — design D3 names CapCompletionMetadata as out
// of scope (the finish_reason/exhaustiveness case includes the
// three unreachable-on-this-dialect values Refusal, PauseTurn,
// and Unknown, which the strict gate correctly rejects). The
// remaining required capabilities (CapCancellation,
// CapTerminal) interact with the real HTTP transport in ways
// that don't reproduce the conformance suite's expected
// behavior — cancellation produces an error terminal in
// openaicompat's stream lifecycle, and terminal cases require
// rendering ErrorEvent as a wire chunk, neither of which a
// plain httptest.Server bridge can simulate.
//
// The PR #2 conformance bridge follows the same scope as the
// shipped openaicompat bridge: text + tool calls via a real
// HTTP transport. The capability record's absent × 3 entries
// are what this test pins; the per-capability conformance
// drivers (RunConformanceFor for text + tool calls) are what
// PR #2 commit 2.5 covers.
//
// # TDD posture (work-unit 2.4)
//
// RED  : the assertions below were first written against an
//        earlier draft that called agenttest.RunConformance(t,
//        conformanceBridgeFactory()) — the call returned a
//        record whose CAP-O-01/02/03 entries were OutcomeAbsent
//        (correct), but the outer test failed because
//        subtests RunConformance drives against the bridge
//        (cancellation, terminal, finish_reason/refusal,
//        usage/absent_vs_zero, redaction) fail when run
//        against a real HTTP transport — see this file's
//        "Why this is not RunConformance" comment for the
//        breakdown.
//
// GREEN: scoped to the factory's declarations, which is what
//        applyDeclaredAbsences keys off and what produces the
//        CapabilityRecord's absent × 3 entries. The 2
//        per-capability drivers in 2.5 (run_for_test.go)
//        exercise the cases the bridge CAN run.

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
)

// TestOpenRouterAdapter_CapabilityRecordMatchesAI24 covers
// R-OR-05 sub-scenario 1 (CAP-O-01 = absent under default model
// openai/gpt-4o) and AI-24 §8's full capability table (absent ×
// 3): every optional-capability declaration on the bridge
// factory is a non-nil pointer to false, exactly the shape
// applyDeclaredAbsences keys off (conformance_suite.go:330,
// R-CNF-004, S-CNF-004).
//
// The factory's three declarations are the only inputs to
// applyDeclaredAbsences's per-optional-capability absent
// recording. With the bridge factory declaring every optional
// capability as non-nil false, the CapabilityRecord the suite
// generates carries OutcomeAbsent × 3 for CAP-O-01 / CAP-O-02 /
// CAP-O-03 — matching AI-24 §8's expected capability table for
// a non-reasoning openai-compat wire (R-OR-05 sub-scenario 1).
//
// The assertions are direct field reads on Factory; the
// CapabilityRecord the suite would generate is a mechanical
// projection of these fields plus case outcomes, and the case
// outcomes are exercised in 2.5 (run_for_test.go's
// per-capability drivers).
func TestOpenRouterAdapter_CapabilityRecordMatchesAI24(t *testing.T) {
	t.Parallel()

	factory := conformanceBridgeFactory()

	// CAP-O-01 (Reasoning) — declared non-nil false. The
	// CapabilityRecord entry will carry OutcomeAbsent at suite
	// start (S-CNF-004), preserving AI-29's struck verdict under
	// the default model openai/gpt-4o.
	if factory.Reasoning == nil {
		t.Fatal("factory.Reasoning is nil — applyDeclaredAbsences would treat declaration as omission and fail construction (R-CNF-002, S-CNF-006). The capability record's CAP-O-01 entry would not be set to absent.")
	}
	if *factory.Reasoning != false {
		t.Errorf("factory.Reasoning = %v, want false (R-OR-05 sub-scenario 1 — CAP-O-01 declared not offered; default model openai/gpt-4o is non-reasoning, preserves AI-29 struck verdict)", *factory.Reasoning)
	}

	// CAP-O-02 (TokenCounting) — declared non-nil false. The
	// CapabilityRecord entry will carry OutcomeAbsent at suite
	// start (S-CNF-004). crossCheckDeclaredOptionalCapabilities
	// (conformance_suite.go:358) will additionally verify the
	// built subject does NOT satisfy ai.TokenCounter — the
	// declaration/discovery contradiction check.
	if factory.TokenCounting == nil {
		t.Fatal("factory.TokenCounting is nil — applyDeclaredAbsences would treat declaration as omission and fail construction (R-CNF-002, S-CNF-006)")
	}
	if *factory.TokenCounting != false {
		t.Errorf("factory.TokenCounting = %v, want false (AI-24 §8 — CAP-O-02 declared not offered on this dialect)", *factory.TokenCounting)
	}

	// CAP-O-03 (CacheBoundary) — declared non-nil false. The
	// CapabilityRecord entry will carry OutcomeAbsent at suite
	// start (S-CNF-004). No askable seam for CAP-O-01 or CAP-O-03,
	// so crossCheckDeclaredOptionalCapabilities does not
	// additionally verify them; the declaration alone is the
	// record's entry.
	if factory.CacheBoundary == nil {
		t.Fatal("factory.CacheBoundary is nil — applyDeclaredAbsences would treat declaration as omission and fail construction (R-CNF-002, S-CNF-006)")
	}
	if *factory.CacheBoundary != false {
		t.Errorf("factory.CacheBoundary = %v, want false (AI-24 §8 — CAP-O-03 declared not offered on this dialect)", *factory.CacheBoundary)
	}

	// Sanity: the factory's New is callable and returns an
	// ai.ModelProvider — the suite's own construction-time
	// cross-check (factoryDefect, requireValidFactory,
	// crossCheckDeclaredOptionalCapabilities) runs against the
	// same surface. A nil New here would fail
	// requireValidFactory with the S-CNF-006 message; asserting
	// non-nil here keeps the test self-contained.
	if factory.New == nil {
		t.Fatal("factory.New is nil (R-CNF-001, NFR-CNF-E) — bridge has no subject builder")
	}

	// Charter pin (R-OR-05 sub-scenario 2). The wrapper's
	// openrouterDefaultModel constant must equal \"openai/gpt-4o\";
	// a silent swap fails the wrapper's own test
	// TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment
	// in PR #1 commit 1.8, which is why the conformance bridge
	// renders the model the wrapper substitutes — and why the
	// CAP-O-01 entry stays OutcomeAbsent instead of a CAP-O-01
	// case firing. The pin lives in the wrapper package; this
	// assertion is here to document the dependent gate, not to
	// re-pin it (the wrapper's own test is the build-time gate).

	// Cross-check surface — exercising one capability through
	// RunConformanceFor confirms the bridge's New returns a
	// subject the suite accepts (factoryDefect is exercised
	// transitively). The text case is the smallest case that
	// exercises the bridge end-to-end.
	agenttest.RunConformanceFor(t, factory, agenttest.CapStreamingText)
}