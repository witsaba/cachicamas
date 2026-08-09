// AI-38 — the OpenRouter capability-record assertion (R-OR-05, R-OR-06,
// R-ACR-004, design D7, tasks.md Phase 6 WU7).
//
// # What this file covers
//
// expectedOpenRouterRecord is the committed nine-entry expectation
// TestOpenRouterAdapter_FullConformance (run_for_test.go) asserts the
// REAL unscoped run's GENERATED record against, via
// agenttest.CompareCapabilityRecords — not a read of the factory's
// declared pointers (R-ACR-004: "asserting the factory's declared
// capability pointers MUST NOT substitute for that comparison"). Every
// required capability (CAP-R-01…05) is expected Satisfied; CAP-O-01
// (reasoning), CAP-O-02 (token counting) and CAP-O-03 (cache boundary)
// are expected Absent, preserving AI-29's struck verdict under the
// default non-reasoning model openai/gpt-4o (R-OR-05); CAP-O-04 (retry)
// is expected Satisfied (locked decision 2, WU6's declaration + blank
// import of conformancetest).
//
// TestOpenRouterAdapter_CapabilityRecordMatchesAI24 below shrinks to
// factory-shape sanity (design D7): it still pins the three
// declared-absent optional capabilities and New's non-nil contract, but
// no longer stands in for the generated-record comparison — that
// comparison is what run_for_test.go's requireOpenRouterCapabilityRecord
// now performs against a REAL run's output.

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
)

// expectedOpenRouterRecord returns the committed nine-entry capability
// expectation (design D7): CAP-R-01…05 Satisfied, CAP-O-01/02/03 Absent,
// CAP-O-04 Satisfied. Built via NewCapabilityRecordForTest +
// SetOutcomeForTest — the same exported test-support surface every
// entry's outcome is asserted through, so this expectation is provably
// nine entries totalling the two closed lists (R-CNF-017), never a
// partial or hand-rolled struct literal.
func expectedOpenRouterRecord() agenttest.CapabilityRecord {
	record := agenttest.NewCapabilityRecordForTest("openrouter-conformance-bridge-expectation")
	record.SetOutcomeForTest(agenttest.CapStreamingText, agenttest.OutcomeSatisfied)
	record.SetOutcomeForTest(agenttest.CapToolCalls, agenttest.OutcomeSatisfied)
	record.SetOutcomeForTest(agenttest.CapCompletionMetadata, agenttest.OutcomeSatisfied)
	record.SetOutcomeForTest(agenttest.CapCancellation, agenttest.OutcomeSatisfied)
	record.SetOutcomeForTest(agenttest.CapTypedFailures, agenttest.OutcomeSatisfied)
	record.SetOutcomeForTest(agenttest.CapReasoningContent, agenttest.OutcomeAbsent)
	record.SetOutcomeForTest(agenttest.CapTokenCounting, agenttest.OutcomeAbsent)
	record.SetOutcomeForTest(agenttest.CapCacheBoundary, agenttest.OutcomeAbsent)
	record.SetOutcomeForTest(agenttest.CapRetry, agenttest.OutcomeSatisfied)
	return record
}

// requireOpenRouterCapabilityRecord asserts record — a REAL unscoped
// run's generated CapabilityRecord — has nil diffs against
// expectedOpenRouterRecord() AND a Verdict() of VerdictPass (design D7,
// R-ACR-004). A generated CAP-O-01(reasoning_content) = Satisfied gets a
// dedicated message naming the AI-29 reopen trigger explicitly (T4,
// S-ACR-013): the committed expectation is never amended to accommodate
// it, and this is a hard stop, not a silent pass.
func requireOpenRouterCapabilityRecord(tb testing.TB, record agenttest.CapabilityRecord) {
	tb.Helper()

	for _, diff := range agenttest.CompareCapabilityRecords(record, expectedOpenRouterRecord()) {
		if diff.Capability == agenttest.CapReasoningContent && diff.Got == agenttest.OutcomeSatisfied {
			tb.Errorf("capability record: CAP-O-01(reasoning_content) generated as %v under the default model, want %v — this IS AI-29 reopen trigger #1 (R-OR-05, R-ACR-004): a reasoning-capable default model needs an explicit ADR and spec amendment BEFORE this entry can read satisfied; the committed expectation is never amended to accommodate an observed satisfied outcome", diff.Got, diff.Want)
			continue
		}
		tb.Errorf("capability record mismatch: %v", diff)
	}
	if verdict := record.Verdict(); verdict != agenttest.VerdictPass {
		tb.Errorf("record.Verdict() = %v, want %v (R-ACR-004)", verdict, agenttest.VerdictPass)
	}
}

// TestOpenRouterAdapter_CapabilityRecordMatchesAI24 is factory-shape
// sanity (design D7): it pins the three declared-absent optional
// capabilities' non-nil-false shape (what applyDeclaredAbsences keys
// off, conformance_suite.go, R-CNF-002/S-CNF-006) and that New builds a
// usable subject. It no longer stands in for the generated-record
// comparison — TestOpenRouterAdapter_FullConformance's
// requireOpenRouterCapabilityRecord call (run_for_test.go) is what
// R-ACR-004 requires: the record a REAL run emits, not a read of these
// declared pointers.
func TestOpenRouterAdapter_CapabilityRecordMatchesAI24(t *testing.T) {
	t.Parallel()

	factory := conformanceBridgeFactory()

	if factory.Reasoning == nil {
		t.Fatal("factory.Reasoning is nil — applyDeclaredAbsences would treat declaration as omission and fail construction (R-CNF-002, S-CNF-006)")
	}
	if *factory.Reasoning != false {
		t.Errorf("factory.Reasoning = %v, want false (R-OR-05 — CAP-O-01 declared not offered; default model openai/gpt-4o is non-reasoning, preserves AI-29 struck verdict)", *factory.Reasoning)
	}
	if factory.TokenCounting == nil {
		t.Fatal("factory.TokenCounting is nil — applyDeclaredAbsences would treat declaration as omission and fail construction (R-CNF-002, S-CNF-006)")
	}
	if *factory.TokenCounting != false {
		t.Errorf("factory.TokenCounting = %v, want false (CAP-O-02 declared not offered on this dialect)", *factory.TokenCounting)
	}
	if factory.CacheBoundary == nil {
		t.Fatal("factory.CacheBoundary is nil — applyDeclaredAbsences would treat declaration as omission and fail construction (R-CNF-002, S-CNF-006)")
	}
	if *factory.CacheBoundary != false {
		t.Errorf("factory.CacheBoundary = %v, want false (CAP-O-03 declared not offered on this dialect)", *factory.CacheBoundary)
	}
	if factory.New == nil {
		t.Fatal("factory.New is nil (R-CNF-001, NFR-CNF-E) — bridge has no subject builder")
	}
}