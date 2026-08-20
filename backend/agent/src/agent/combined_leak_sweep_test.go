// AG-21 — Phase 5: the leak sweep (R-CNH-005, R-CNH-006; charter
// AG-21.3, 0003:2030-2041; S-CNH-011..013). Design AD-5/D-B: ONE new
// non-parallel wrapper over this change's OWN drivers — never the
// pre-existing suite — passed to agenttest.RequireNoGoroutineLeak.
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
)

// TestConcurrencyHardening_PackageLeakSweep — S-CNH-011, S-CNH-013.
// Charter AG-21.3. Given one new top-level test that does NOT call
// t.Parallel(), whose scenario callable invokes this change's cell,
// pressure and cross-run drivers serially, when it is passed to the
// module's goroutine-leak helper, then the helper passes; each
// iteration fully quiesces before returning — every sink drained,
// every run result received, every gate released and every tool-
// release channel closed; no driver reads the process goroutine
// count; and no gate constructed by any driver survives its driver's
// return, proven by the cleanup release registered at construction as
// well as by the inline release (R-CNH-005).
//
// S-CNH-013: the sweep's scenario includes the pressure-2 driver,
// whose own iteration releases its deaf tool AFTER the run has
// returned; the typed detached-call report naming the tool and the
// call identity is observed on that iteration's stream; and across
// the helper's repeats the goroutine count stays within tolerance —
// the third-party goroutine is counted and shown to exit, never
// excluded from the count (R-CNH-006).
//
// NO t.Parallel() anywhere in this file, on this test or its own
// drivers: mechanically enforced by RequireNoGoroutineLeak's own
// tb.Setenv sentinel (stream_kit_leak.go:80,110) — this test, or an
// ancestor, calling it would panic. The permanently-stalled-observer
// leak (AG-20, by design) is excluded from this sweep's scenario set
// by construction: none of the drivers below gates an observing hook
// (NFR-CNH-003, adopting NFR-HKS-005 verbatim).
func TestConcurrencyHardening_PackageLeakSweep(t *testing.T) {
	scenario := func() {
		for _, tc := range cnhCells {
			runCombinedCell(t, tc.state+"/"+tc.signal, tc)
		}
		cnhDrivePressureNeverCancelled(t)
		cnhDrivePressureCancelledUnblocks(t)
		cnhDriveCrossRunNilHistory(t)
		cnhDriveCrossRunSharedHistory(t)
	}

	agenttest.RequireNoGoroutineLeak(t, scenario)
}
