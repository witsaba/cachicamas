// AG-23 (D-5, R-RUN-014, design AD-1) — the RED for the forwarder's
// bounded exit on the panic-unwind path (S-RUN-114). Lives in
// `package agent_test` per harness_test.go's own guard
// (TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard), which
// requires every `harness*_test.go` file to declare it.
//
// Watched failing before the production fix existed (apply-progress
// records the exact crash trace): a `PreRequestHook` panicking through
// the deliberately-unrecovered seam (R-AGS-016) left the per-attempt
// forwarder goroutine (harness.go) permanently parked forwarding
// turn_start to an abandoned consumer sink, and Run's own `close(sink)`
// defer then raced it — "send on closed channel", an unrecovered crash
// on a goroutine the caller does not own.
package agent_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// harnessForwarderPanicSentinel is a runtime-minted panic value: a fresh
// pointer per iteration, never a shared package constant. The test
// asserts the recovered value is the EXACT pointer this iteration
// panicked with — identity, not shape — so a production change that
// discarded the original panic and re-panicked with a same-shaped
// substitute would be caught: R-RUN-014 itself forbids the harness from
// recovering, wrapping, or re-typing the panic.
type harnessForwarderPanicSentinel struct {
	iteration int
}

// TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink is R-RUN-014's own
// RED (S-RUN-114). 100 iterations, a fresh Harness each: PreRequestHook
// panics with this iteration's own uniquely-minted sentinel; the
// consumer drains EXACTLY the run-open event (the run_start Harness.Run
// itself emits) and then abandons the sink — no further read ever
// happens on it. That abandonment is what parks the per-attempt
// forwarder in its own forwarding send for as long as the unwind takes
// to reach it, which is exactly the window this requirement's fix must
// close safely.
func TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink(t *testing.T) {
	t.Parallel()

	for i := 0; i < 100; i++ {
		sentinel := &harnessForwarderPanicSentinel{iteration: i}

		hook := func(_ context.Context, _ ai.Request) (ai.Request, error) {
			panic(sentinel)
		}

		h := agent.Harness{
			// Never consumed: the hook panics before Turn ever reaches
			// provider.Stream, so an empty script queue is sufficient.
			Provider: agenttest.NewProvider(),
			System:   fmt.Sprintf("system prompt for run-014 iteration %d", i),
			Turn:     agent.TurnOptions{PreRequestHook: hook},
		}
		sink := make(chan *agent.Event)

		runDone := make(chan struct{})
		var recovered any
		go func() {
			defer close(runDone)
			defer func() {
				recovered = recover()
			}()
			_, _, _ = h.Run(contextBackground(), firstMessage(t), sink)
		}()

		// Drain EXACTLY the run-open event, then abandon the sink.
		ev, ok := <-sink
		if !ok {
			t.Fatalf("iteration %d: sink closed before the run-open event was ever sent", i)
		}
		if ev.Kind() != agent.EventKindRunStart {
			t.Fatalf("iteration %d: first event kind = %v, want EventKindRunStart (the run-open event)", i, ev.Kind())
		}
		// No further read on sink from this point on — the consumer has
		// abandoned it, exactly as S-RUN-114 requires.

		<-runDone

		recoveredSentinel, ok := recovered.(*harnessForwarderPanicSentinel)
		if !ok {
			t.Fatalf("iteration %d: recovered value = %#v (%T), want *harnessForwarderPanicSentinel — the panic MUST propagate uncontained, never recovered, wrapped, or re-typed by the harness (R-RUN-014)", i, recovered, recovered)
		}
		if recoveredSentinel != sentinel {
			t.Fatalf("iteration %d: recovered sentinel is a different pointer than the one this iteration panicked with — want the EXACT value by identity, not merely one that looks the same", i)
		}
	}
}
