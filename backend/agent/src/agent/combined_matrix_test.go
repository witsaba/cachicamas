// AG-21 — Phase 2: the twelve combined-state x signal cells
// (R-CNH-001, R-CNH-002; S-CNH-001..005). Charter AG-21.1
// (0003:1983-2010): "the interactions no single-milestone test
// exercises". Every cell is driven through runCombinedCell
// (combined_state_fixtures_test.go), the one shared assertion core, so
// no cell's assertions can be weakened alone (design AD-1).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// cnhCells is the twelve-row table (design AD-1): one row per state x
// signal pair. Ten cells carry the uniform Then; the compaction x
// provider-failure and child-harness-active x provider-failure cells
// carry R-CNH-001's recorded, evidence-cited adjusted Then (sdd-verify
// round-2, MAJOR-2: suspension x provider-failure is uniform in
// outcome, adjusted only in the timing S-CNH-005 checks — it is NOT
// one of these two); the three child-harness cells additionally carry
// S-CNH-002's containment check.
var cnhCells = []cellCase{
	// State 1 — a suspension pending (a permission-deferred call parked
	// on the gate).
	{state: "suspension_pending", signal: "interrupt", sig: cnhInterrupt, arrange: arrangeSuspensionPending},
	{state: "suspension_pending", signal: "shutdown", sig: cnhShutdown, arrange: arrangeSuspensionPending},
	{state: "suspension_pending", signal: "provider_failure", sig: cnhProviderFailure, arrange: arrangeSuspensionPending, adjustedThen: adjustedThenSuspensionFailure},

	// State 2 — steering queued (a message accepted with a nil return
	// while the turn's provider stream is held).
	{state: "steering_queued", signal: "interrupt", sig: cnhInterrupt, arrange: arrangeSteeringQueued},
	{state: "steering_queued", signal: "shutdown", sig: cnhShutdown, arrange: arrangeSteeringQueued},
	{state: "steering_queued", signal: "provider_failure", sig: cnhProviderFailure, arrange: arrangeSteeringQueued},

	// State 3 — compaction mid-run (a compaction call in flight).
	{state: "compaction_mid_run", signal: "interrupt", sig: cnhInterrupt, arrange: arrangeCompactionMidRun},
	{state: "compaction_mid_run", signal: "shutdown", sig: cnhShutdown, arrange: arrangeCompactionMidRun},
	{state: "compaction_mid_run", signal: "provider_failure", sig: cnhProviderFailure, arrange: arrangeCompactionMidRun, adjustedThen: adjustedThenCompactionFailure},

	// State 4 — a child harness active (hosted inside a tool call, its
	// own provider stream held). Every one of these three additionally
	// carries S-CNH-002's containment/leaf-first check.
	{state: "child_harness_active", signal: "interrupt", sig: cnhInterrupt, arrange: arrangeChildHarnessActive, containment: containmentLeafFirst},
	{state: "child_harness_active", signal: "shutdown", sig: cnhShutdown, arrange: arrangeChildHarnessActive, containment: containmentLeafFirst},
	{state: "child_harness_active", signal: "provider_failure", sig: cnhProviderFailure, arrange: arrangeChildHarnessActive, adjustedThen: adjustedThenChildFailure, containment: containmentChildFailure},
}

// TestCombinedMatrix — S-CNH-001. Charter AG-21.1, the twelve cells.
// Given a run driven to each of the four pending states, where each
// pending state is proven by a happens-before edge the production code
// itself provides and never by elapsed time or goroutine scheduling,
// when the cell's signal fires strictly after that edge, then all four
// (or adjusted) assertions of R-CNH-001 hold for that cell; and each
// cell runs under -race with its own harness, provider, sink and
// gates, sharing nothing with any sibling cell (R-CNH-002).
func TestCombinedMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range cnhCells {
		tc := tc
		t.Run(tc.state+"/"+tc.signal, func(t *testing.T) {
			t.Parallel()
			runCombinedCell(t, tc.state+"/"+tc.signal, tc)
		})
	}
}

// adjustedThenCompactionFailure — S-CNH-004. The compaction x
// provider-failure cell, at its R-CNH-001-recorded adjusted Then.
// Given a run whose compaction call is in flight against an injected
// provider held at a gate, when that provider's script fires a
// terminal failure, then the compaction_failed event is emitted and no
// compaction_finished is (R-CMP-010), the run does NOT wind down -- it
// continues and reaches its own terminal -- and a run-close carrying
// the interrupted or shutdown outcome anywhere on this cell's stream
// FAILS the scenario.
func adjustedThenCompactionFailure(t *testing.T, _ *cnhArrangement, got cnhRunOutcome, events []agent.Event) {
	t.Helper()

	if got := countKind(events, agent.EventKindCompactionFinished); got != 0 {
		t.Errorf("compaction_mid_run/provider_failure: compaction_finished count = %d, want 0 (R-CMP-010: a failed compaction never commits)", got)
	}
	if got := countKind(events, agent.EventKindCompactionFailed); got != 1 {
		t.Errorf("compaction_mid_run/provider_failure: compaction_failed count = %d, want exactly 1", got)
	}

	if len(events) == 0 {
		t.Fatal("compaction_mid_run/provider_failure: zero events observed")
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("compaction_mid_run/provider_failure: last event kind = %v, want run_end", last.Kind())
	}
	// sdd-verify round-2, SUGGESTION-1 (adopted): assert the positive
	// outcome directly rather than only excluding the two wrong ones --
	// strictly stronger, and this exact strengthening is what would
	// have surfaced MAJOR-2 at authoring time.
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("compaction_mid_run/provider_failure: run_end outcome = %v, want RunOutcomeCompleted -- the run continues past the compaction failure to its OWN terminal, never interrupted/shutdown/failed (R-CMP-010's own heading: a failed compaction never winds the run down)", runEnd.Outcome())
	}
	if got.err != nil {
		t.Errorf("compaction_mid_run/provider_failure: Run returned err = %v, want nil -- the compaction failure is stream-visible only and never fails the run itself (R-CMP-013)", got.err)
	}
}

// adjustedThenSuspensionFailure — S-CNH-005. The suspension x
// provider-failure cell. Its Then is NOT adjusted: R-CNH-001's two
// recorded exceptions are compaction x failure and child-harness-
// active x failure, and this cell is neither. What is bounded here is
// the firing POINT, not the outcome -- no main-provider stream is
// live while a call is parked, so the failure fires at the next
// reachable provider boundary and the uniform Then then holds.
// Given a run whose call is parked on the permission gate, when the
// park is resolved and the next reachable provider boundary fires a
// terminal failure, then the uniform provider-failure Then holds at
// that terminal; and the scenario asserts that no provider request was
// recorded between the park and its resolution -- the evidence that no
// main-provider stream was live to fail while the call was parked.
func adjustedThenSuspensionFailure(t *testing.T, arr *cnhArrangement, got cnhRunOutcome, events []agent.Event) {
	t.Helper()

	if arr.requestsAtPark != 1 {
		t.Errorf("suspension_pending/provider_failure: %d provider request(s) recorded at the moment the park was proven, want exactly 1 (turn one's own) -- no main-provider stream may be live while a call is parked", arr.requestsAtPark)
	}
	assertUniformOutcome(t, "suspension_pending/provider_failure", cnhProviderFailure, got, events)
}

// adjustedThenChildFailure is the child harness's own provider-failure
// Then, following the same disposition R-CNH-001 records for
// compaction: the requirement's uniform prose describes the state
// whose OWN driving provider fails (suspension, steering); here it is
// the CHILD's provider that fails, and the failure is CONTAINED
// (S-CNH-002, R-DEL-002) rather than propagated -- the true mechanism,
// corrected here (sdd-verify round-2, MINOR-1; the prior wording --
// "Schedule's own return value is discarded by the loop" -- was false
// on this path and carried a dangling R-LSK-... placeholder): Schedule
// (tool.go) has NO error return at all, only []Result. On the
// continuation path Harness.Run always takes, finishContinuationTurn
// (loop.go:525-566) captures that slice at :542 and converts each
// Result via toolResultMessage (:578-598) into a tool-result or
// tool-failure message appended to history -- never a Turn-level
// error. The one place Schedule's return IS discarded (`_ =
// sched.Schedule(...)`, loop.go:411) is the nil-continuation path,
// which Harness.Run never takes. So a tool-execution-level failure
// never surfaces as the PARENT's own Turn-level error, and the parent
// falls through to its own next turn exactly as the compaction cell
// does. Verified directly against the shipped scheduler/loop behavior
// in this worktree (not assumed): the parent reaches its own terminal
// with a nil error, never RunOutcomeFailed, and never
// interrupted/shutdown -- the containment closure
// (containmentChildFailure) carries the scenario's remaining, more
// specific assertions.
func adjustedThenChildFailure(t *testing.T, _ *cnhArrangement, got cnhRunOutcome, events []agent.Event) {
	t.Helper()

	if len(events) == 0 {
		t.Fatal("child_harness_active/provider_failure: zero events observed")
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("child_harness_active/provider_failure: last event kind = %v, want run_end", last.Kind())
	}
	// sdd-verify round-2, SUGGESTION-1 (adopted): assert the positive
	// outcome directly rather than only excluding the two wrong ones.
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("child_harness_active/provider_failure: run_end outcome = %v, want RunOutcomeCompleted -- the parent reaches its own terminal, never interrupted/shutdown/failed -- the parent itself was never signalled", runEnd.Outcome())
	}
	if got.err != nil {
		t.Errorf("child_harness_active/provider_failure: Run returned err = %v, want nil -- the child's failure is contained on its own subagent bracket and never fails the parent run (R-DEL-002, S-CNH-002)", got.err)
	}
}

// containmentLeafFirst — S-CNH-002's second half. Given the child-
// harness-active arrangement signalled by interrupt or by shutdown,
// then the child winds down before the parent, asserted by observed
// event index, never by elapsed time (R-DEL-006, S-CAN-015).
func containmentLeafFirst(t *testing.T, arr *cnhArrangement, events []agent.Event) {
	t.Helper()

	childEvents := arr.childTool.ChildEvents()
	if report := agent.CheckStream(childEvents); report.Violation() != nil {
		t.Errorf("child_harness_active: CheckStream(child stream) = %v, want nil", report.Violation())
	}
	subagentEndedIdx := indexOfFirst(events, agent.EventKindSubagentEnded)
	runEndIdx := indexOfLast(events, agent.EventKindRunEnd)
	if subagentEndedIdx < 0 || runEndIdx < 0 || subagentEndedIdx >= runEndIdx {
		t.Errorf("child_harness_active: subagent_ended index = %d, parent run_end index = %d, want subagent_ended strictly before run_end (leaf-first, R-DEL-006)", subagentEndedIdx, runEndIdx)
	}
}

// containmentChildFailure — S-CNH-002's first half. Given a parent run
// whose tool hosts a child harness with its own held provider stream,
// when the child's stream fires a terminal provider failure, then the
// parent's stream carries a closed subagent bracket with only
// R-DEL-002-admitted kinds between its markers, and the parent's
// stream and the child's stream are each CheckStream-valid, validated
// separately, never concatenated (R-DEL-005).
func containmentChildFailure(t *testing.T, arr *cnhArrangement, events []agent.Event) {
	t.Helper()

	childEvents := arr.childTool.ChildEvents()
	validateStreamsSeparately(t, events, childEvents)

	startedIdx := indexOfFirst(events, agent.EventKindSubagentStarted)
	endedIdx := indexOfFirst(events, agent.EventKindSubagentEnded)
	if startedIdx < 0 || endedIdx < 0 || startedIdx >= endedIdx {
		t.Fatalf("child_harness_active/provider_failure: subagent_started index = %d, subagent_ended index = %d, want started strictly before ended, both present", startedIdx, endedIdx)
	}
	for i := startedIdx + 1; i < endedIdx; i++ {
		switch events[i].Kind() {
		case agent.EventKindRunStart, agent.EventKindRunEnd, agent.EventKindTurnStart, agent.EventKindTurnEnd,
			agent.EventKindPermissionResolutionRemembered, agent.EventKindCostTurn:
			t.Errorf("child_harness_active/provider_failure: event %d between the subagent bracket carries refused kind %v -- it must never have been mirrored", i, events[i].Kind())
		}
	}
}
