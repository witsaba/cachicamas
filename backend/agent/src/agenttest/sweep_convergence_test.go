// AI-36.1 (WU-1, task 1.5), re-anchored at AI-39 (design.md D2) —
// S-CNF-080's "both reach it" clause proven behaviorally: this package's
// own scanTextForSentinel, run over a corpus and needle, reports the same
// detection outcome as calling the shared agenttest/sweep core directly
// over the identical corpus/needle — evidence that scanTextForSentinel
// delegates to the one shared implementation (design.md AD-1) rather than
// a diverging re-implementation.
//
// AI-39 drops this file's former import of openrouter/smoke (the package
// moved to openrouter/internal/smoke, R-LSM-005, and this pin's own
// package — agenttest — must not depend on the provider-adapter subtree
// it is not part of). The smoke-side half of the same convergence proof
// now lives next to the moved package itself, in
// openaicompat/openrouter/internal/smoke/sentinel_sweep_test.go, and
// compares smoke.Scan against this same sweep.Scan core over the same
// canary-corpus construction. Both halves target sweep.Scan directly, so
// a divergence in either consumer is attributable by name, and — by
// transitivity over the same corpus construction — the two consumers
// still agree with each other (design.md D2 "Why not weaker").
//
// This is a regression pin, not RED-first: the behavior under test
// (convergent detection) is a consequence of tasks 1.2-1.4 already
// landing, not a new capability introduced by this file.
package agenttest

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
)

// TestSweepConvergence_SameCorpusAndNeedle_BothConsumersAgree is the pin.
func TestSweepConvergence_SameCorpusAndNeedle_BothConsumersAgree(t *testing.T) {
	t.Parallel()

	t.Run("a planted needle: both consumers detect it", func(t *testing.T) {
		t.Parallel()

		const needle = "SWEEP-CONVERGENCE-PLANTED-CANARY-3d8f"
		corpus := "prefix noise " + needle + " suffix noise"

		agenttestFound := scanTextForSentinel(corpus, needle)

		_, coreFound := sweep.Scan([]byte(corpus), []sweep.Entry{{Vector: "needle", Needle: []byte(needle)}})

		if agenttestFound != coreFound {
			t.Errorf("scanTextForSentinel found=%v, sweep.Scan found=%v, want identical detection outcomes (S-CNF-080)", agenttestFound, coreFound)
		}
		if !agenttestFound {
			t.Fatal("neither consumer detected the planted needle, want both to (positive control)")
		}
	})

	t.Run("no planted needle: neither consumer reports a leak", func(t *testing.T) {
		t.Parallel()

		const needle = "SWEEP-CONVERGENCE-ABSENT-CANARY-9a1c"
		corpus := "clean prose with nothing planted"

		agenttestFound := scanTextForSentinel(corpus, needle)
		_, coreFound := sweep.Scan([]byte(corpus), []sweep.Entry{{Vector: "needle", Needle: []byte(needle)}})

		if agenttestFound != coreFound {
			t.Errorf("scanTextForSentinel found=%v, sweep.Scan found=%v, want identical detection outcomes on a clean corpus", agenttestFound, coreFound)
		}
		if agenttestFound {
			t.Error("scanTextForSentinel reported a leak on a clean corpus, want none")
		}
	})
}
