// AI-23.2 extension — R-CNF-023: the capability-scoped conformance entry
// point. White-box (package agenttest), matching conformance_suite_test.go's
// own precedent for this directory's runner-internal tests: casesFor is
// unexported, and this file's own registry-level checks need direct access
// to conformanceRegistry's real registration order.

package agenttest

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestCasesFor_StreamingTextScope_ReturnsExactlyTheTwoTextCaseNames proves
// S-CLA-029's registry-level observable: casesFor(CapStreamingText) returns
// exactly the two cases conformance_text.go's own init() registered, in
// registration order, and no other registered case.
func TestCasesFor_StreamingTextScope_ReturnsExactlyTheTwoTextCaseNames(t *testing.T) {
	cases := casesFor(CapStreamingText)
	wantNames := []string{"text/order_contiguity_byte_exact_reconstruction", "text/empty_completion_is_legal"}
	if len(cases) != len(wantNames) {
		t.Fatalf("casesFor(CapStreamingText) returned %d case(s), want %d: %v (S-CLA-029)", len(cases), len(wantNames), cases)
	}
	for i, c := range cases {
		if c.name != wantNames[i] {
			t.Errorf("casesFor(CapStreamingText)[%d].name = %q, want %q (S-CLA-029)", i, c.name, wantNames[i])
		}
		if c.capability != CapStreamingText {
			t.Errorf("casesFor(CapStreamingText)[%d].capability = %v, want %v", i, c.capability, CapStreamingText)
		}
	}
}

// TestCasesFor_FreshSliceEachCall proves casesFor never aliases the
// registry: a caller mutating the returned slice cannot corrupt what a
// later call returns.
func TestCasesFor_FreshSliceEachCall(t *testing.T) {
	first := casesFor(CapToolCalls)
	if len(first) == 0 {
		t.Fatal("casesFor(CapToolCalls) returned no cases, want at least one to mutate")
	}
	first[0].name = "mutated"

	second := casesFor(CapToolCalls)
	if second[0].name == "mutated" {
		t.Error("casesFor returned an aliased slice: mutating the first result corrupted the second (want a fresh slice each call)")
	}
}

// TestRunConformanceFor_StreamingTextScope_PassesEndToEnd proves S-CLA-029's
// other half: a scoped run named for CapStreamingText executes and passes
// green end to end against FakeFactory — the exported door compiles and
// runs, delegating to the unmodified runConformanceCases (D8: "plumbing
// refactor needed: none").
func TestRunConformanceFor_StreamingTextScope_PassesEndToEnd(t *testing.T) {
	RunConformanceFor(t, FakeFactory(), CapStreamingText)
}

// TestRunConformanceFor_UndeclaredOptionalCapability_ReachesTheIdenticalConstructionFailure
// proves S-CLA-031: a factory leaving one optional capability undeclared
// still fails at construction naming it, exactly as R-CNF-002's S-CNF-006
// already requires of the unscoped runner. RunConformanceFor's body is
// exactly runConformanceCases(t, f, casesFor(capability)) — reused
// byte-identical, no plumbing refactor (D8) — whose own first statement is
// requireValidFactory(t, f); this test proves that shared precedent
// directly (the pure factoryDefect decision, plus the identical
// testing.TB-substitutable wiring conformance_suite_test.go's
// TestConformanceSkeleton_UndeclaredOptionalCapability_FailsConstructionNamingIt
// already proves for the unscoped door) rather than driving
// RunConformanceFor itself with an undeclared factory: Go's testing package
// propagates a failed subtest to every ancestor *testing.T unconditionally
// (conformance_suite_test.go:622), which would make this file's own
// `make test` red by construction rather than by defect.
func TestRunConformanceFor_UndeclaredOptionalCapability_ReachesTheIdenticalConstructionFailure(t *testing.T) {
	f := Factory{
		New:           func(_ testing.TB, scripts ...Script) ai.ModelProvider { return NewProvider(scripts...) },
		Reasoning:     nil, // undeclared — the scoped door's own construction check
		TokenCounting: boolPtr(false),
		CacheBoundary: boolPtr(false),
	}

	if msg := factoryDefect(f); msg == "" || !contains(msg, CapReasoningContent.String()) {
		t.Fatalf("factoryDefect(undeclared Reasoning) = %q, want a defect naming %v (S-CLA-031)", msg, CapReasoningContent)
	}

	probe := &probeTB{}
	requireValidFactory(probe, f)
	if !probe.failed {
		t.Error("requireValidFactory did not fail against an undeclared capability, want it to (S-CLA-031)")
	}
}

// TestRunConformanceFor_FailingInScopeCase_ReachesTheIdenticalPropagationSeam
// proves S-CLA-030's structural seam: a case that lies within a named scope
// is genuinely reached by RunConformanceFor — casesFor's own inclusion,
// confirmed here — and would be executed through the identical
// runConformanceCases -> runRegisteredCase -> runOneCase (t.Run) delegation
// the unscoped RunConformance already uses (no plumbing refactor, D8), so a
// failing in-scope case fails the scoped run and names it exactly as it
// would for the unscoped runner.
//
// The live end-to-end failure itself was scratch-verified once rather than
// committed as a permanently-run failing test, for the identical
// propagation reason stated above: tasks.md records the captured
// `--- FAIL: .../text/order_contiguity_byte_exact_reconstruction` output
// from a factory whose subject emits nothing at all.
func TestRunConformanceFor_FailingInScopeCase_ReachesTheIdenticalPropagationSeam(t *testing.T) {
	cases := casesFor(CapStreamingText)
	var found bool
	for _, c := range cases {
		if c.name == "text/order_contiguity_byte_exact_reconstruction" {
			found = true
		}
	}
	if !found {
		t.Fatal("casesFor(CapStreamingText) does not include text/order_contiguity_byte_exact_reconstruction, want it reachable so a failing subject would genuinely fail the scoped run (S-CLA-030)")
	}
}
