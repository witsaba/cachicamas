// AI-23.2 extension — R-CNF-023: a capability-scoped conformance entry
// point. A partial adapter that has implemented, say, only streaming text
// has no legal way to run just the text cases through the unexported
// runConformanceCases or the whole-registry RunConformance — this file adds
// the exported door for exactly that: run exactly the registered cases
// keyed to one named capability, out-of-scope cases neither run nor
// reported.
//
// No plumbing refactor: runConformanceCases is already parameterized over
// an explicit case list; requireValidFactory/factoryDefect, the
// declared-absent skip machinery, the CAP-O-02 cross-check and panic
// recovery are all reused byte-identical. RunConformance stays the
// complete, unscoped gate — a scoped run yields no CapabilityRecord, so it
// can never be presented as evidence of full conformance (S-CLA-032).
//
// New file only (NFR-CNF-D): fake_*.go and stream_kit_*.go are untouched.

package agenttest

import "testing"

// RunConformanceFor runs exactly the registered conformance cases keyed to
// capability against the subject f builds, and fails t naming capability
// when it is not one this suite recognises (R-CNF-023).
//
// It returns no CapabilityRecord: a scoped run's verdict is the test
// outcome alone, and — unlike RunConformance's total record — is never
// presentable as evidence of full conformance (S-CLA-032). The factory
// construction checks of R-CNF-002 (all three optional declarations
// mandatory) and every in-scope case's own semantics and standing are
// preserved unchanged, because this delegates to the identical
// runConformanceCases RunConformance itself uses.
func RunConformanceFor(t *testing.T, f Factory, capability Capability) {
	t.Helper()
	if !capability.registered() {
		t.Fatalf("agenttest: RunConformanceFor given capability %v, outside both the CAP-R and CAP-O closed lists — want a registered capability (R-CNF-023)", capability)
	}
	runConformanceCases(t, f, casesFor(capability))
}

// casesFor returns a fresh slice of every registered case keyed to
// capability, in registration order — the same order conformanceRegistry
// itself carries, so `casesFor(c)` is a strict, order-preserving subset of
// what RunConformance would run.
func casesFor(capability Capability) []conformanceCase {
	out := make([]conformanceCase, 0, len(conformanceRegistry))
	for _, c := range conformanceRegistry {
		if c.capability == capability {
			out = append(out, c)
		}
	}
	return out
}
