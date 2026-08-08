// AI-36.1 (WU-1, task 1.5) — S-CNF-080's "both reach it" clause proven
// behaviorally: this package's own scanTextForSentinel and
// openrouter/smoke's exported Scan, run over the identical corpus and
// needle, report the identical detection outcome — evidence that both
// delegate to the one shared agenttest/sweep implementation (design.md
// AD-1) rather than diverging re-implementations.
//
// This is a regression pin, not RED-first: the behavior under test
// (convergent detection) is a consequence of tasks 1.2-1.4 already
// landing, not a new capability introduced by this file.
package agenttest

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/smoke"
)

// TestSweepConvergence_SameCorpusAndNeedle_BothConsumersAgree is the pin.
func TestSweepConvergence_SameCorpusAndNeedle_BothConsumersAgree(t *testing.T) {
	t.Parallel()

	t.Run("a planted needle: both consumers detect it", func(t *testing.T) {
		t.Parallel()

		const needle = "SWEEP-CONVERGENCE-PLANTED-CANARY-3d8f"
		corpus := "prefix noise " + needle + " suffix noise"

		agenttestFound := scanTextForSentinel(corpus, needle)

		smokeErr := smoke.Scan([]byte(corpus), smoke.BuildDenyList("unrelated-env-var", "unrelated-secret-pfx", needle))
		smokeFound := smokeErr != nil

		if agenttestFound != smokeFound {
			t.Errorf("scanTextForSentinel found=%v, smoke.Scan found=%v, want identical detection outcomes (S-CNF-080)", agenttestFound, smokeFound)
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
		smokeErr := smoke.Scan([]byte(corpus), smoke.BuildDenyList("unrelated-env-var", "unrelated-secret-pfx", needle))
		smokeFound := smokeErr != nil

		if agenttestFound != smokeFound {
			t.Errorf("scanTextForSentinel found=%v, smoke.Scan found=%v, want identical detection outcomes on a clean corpus", agenttestFound, smokeFound)
		}
		if agenttestFound {
			t.Error("scanTextForSentinel reported a leak on a clean corpus, want none")
		}
	})
}
