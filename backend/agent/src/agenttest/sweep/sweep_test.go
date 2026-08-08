// AI-36.1 (WU-1, task 1.1) — the shared sweep helper's own test surface,
// RED-first: this file references sweep.Entry, sweep.Scan and
// sweep.SelfTest, none of which exist before sweep.go lands (S-CNF-080).
package sweep_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
)

// TestScan_DetectsPlantedNeedle covers the leak-detection path: a needle
// present in the corpus is found, and the reported vector is the entry's
// own name — never the needle bytes themselves.
func TestScan_DetectsPlantedNeedle(t *testing.T) {
	t.Parallel()

	corpus := []byte("prefix noise sk-planted-canary-detect-me suffix noise")
	deny := []sweep.Entry{{Vector: "credential", Needle: []byte("sk-planted-canary-detect-me")}}

	vector, found := sweep.Scan(corpus, deny)
	if !found {
		t.Fatal("Scan() found = false, want true: the needle is present in the corpus")
	}
	if vector != "credential" {
		t.Errorf("Scan() vector = %q, want %q", vector, "credential")
	}
}

// TestScan_ReturnsFalseWhenAbsent is the paired clean-input case: a corpus
// that never carries the needle reports no leak. Without this case,
// TestScan_DetectsPlantedNeedle alone would not prove Scan can ever say
// "no" — a scan that always reports true would still pass a leak-only
// suite.
func TestScan_ReturnsFalseWhenAbsent(t *testing.T) {
	t.Parallel()

	corpus := []byte("nothing sensitive here, just prose")
	deny := []sweep.Entry{{Vector: "credential", Needle: []byte("sk-absent-canary-never-planted")}}

	vector, found := sweep.Scan(corpus, deny)
	if found {
		t.Errorf("Scan() found = true (vector=%q), want false: the needle is absent from the corpus", vector)
	}
	if vector != "" {
		t.Errorf("Scan() vector = %q, want empty on a clean corpus", vector)
	}

	t.Run("an empty corpus and an empty deny list never match", func(t *testing.T) {
		t.Parallel()

		if _, found := sweep.Scan(nil, deny); found {
			t.Error("Scan(nil, deny) found = true, want false: an empty corpus carries no needle")
		}
		if _, found := sweep.Scan(corpus, nil); found {
			t.Error("Scan(corpus, nil) found = true, want false: an empty deny list names nothing to detect")
		}
	})
}

// TestSelfTest_PassesForALiveDenyList is SelfTest's own positive path: a
// deny list whose needles are real, non-empty bytes always bites its own
// synthetic corpus, so a healthy call site never sees a spurious SelfTest
// failure.
func TestSelfTest_PassesForALiveDenyList(t *testing.T) {
	t.Parallel()

	deny := []sweep.Entry{
		{Vector: "credential", Needle: []byte("sk-selftest-live-canary-one")},
		{Vector: "content-body", Needle: []byte("prompt-selftest-live-canary-two")},
	}
	if err := sweep.SelfTest(deny); err != nil {
		t.Errorf("SelfTest() = %v, want nil: every needle here is real and must bite its own synthetic corpus", err)
	}
}

// TestSelfTest_FailsNamingVectorOnBrokenDenyList is the mandatory positive
// control for the positive control itself (AD-2): a deny-list entry
// carrying an empty needle can never be detected by Scan (Scan skips it
// rather than matching everything) — exactly the silently-vacuous shape
// this milestone's own thesis warns about. SelfTest MUST refuse to call
// such an entry proven, and MUST name the offending vector rather than the
// (empty) needle.
func TestSelfTest_FailsNamingVectorOnBrokenDenyList(t *testing.T) {
	t.Parallel()

	deny := []sweep.Entry{{Vector: "broken-vector-name", Needle: nil}}

	err := sweep.SelfTest(deny)
	if err == nil {
		t.Fatal("SelfTest() = nil, want a non-nil error: an empty-needle entry can never bite its own corpus")
	}
	if !strings.Contains(err.Error(), "broken-vector-name") {
		t.Errorf("SelfTest() error = %q, want it to name the offending vector %q", err.Error(), "broken-vector-name")
	}
}
