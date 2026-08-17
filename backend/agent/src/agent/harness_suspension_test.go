// AG-13 Phase 5 — the fourth acceptance clause: the run survives a
// permission suspension that spans a turn (R-RUN-010), jointly S-APP-015;
// and the inherited R-APP-002 parked-wait bite (S-RUN-091, jointly
// S-APP-016).
//
// The acceptance clause needs no new production code: Phase 0's injected
// *Scheduler (TurnContinuation.Scheduler, design "Decision 2") and Phase
// 2's mandatory per-turn forwarder goroutine already make WakeParked
// reachable — and permission_decision_required observable — while a turn
// is still blocked mid-Schedule. This file exercises that composition
// from AG-13's own acceptance-clause angle, exactly as design.md
// predicts, and adds the bite agent-permission-protocol/spec.md:172
// explicitly names: "the missing bite must observe the parked wait, not
// the registration."
package agent_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// deferThenAllowPolicy is Phase 5's shared fixture: Resolve's first
// invocation defers (parking the call); every later invocation is the
// re-entry after the parked wait releases (S-RUN-090's "resolution #2").
//
// The wake-issued observation is not an assertion made from inside
// Resolve: Resolve may run on a goroutine other than the test's own, and
// only t.Error/t.Errorf — never t.Fatal/t.FailNow — are safe to call off
// the test goroutine (and even those would interleave badly with a
// concurrent scheduler goroutine). Instead Resolve records what it
// observed into flagAtSecondResolve, and the test asserts on that
// recording after the run has fully completed and been synchronized back
// through a channel. This is the S-APP-015 mechanism: because the test
// sets wakeIssued immediately before calling WakeParked, a resolution
// that observes it set is possible only if the call goroutine genuinely
// waited on WakeParked's close of parkCh — registration alone (which
// happens long before decision_required is even emitted) cannot make
// this read true.
type deferThenAllowPolicy struct {
	wakeIssued *atomic.Bool

	resolveCount        atomic.Int64
	flagAtSecondResolve atomic.Bool
}

func (p *deferThenAllowPolicy) Resolve(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
	if p.resolveCount.Add(1) == 1 {
		return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
	}
	p.flagAtSecondResolve.Store(p.wakeIssued.Load())
	return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
}

func (p *deferThenAllowPolicy) Remember(context.Context, string, agent.PermissionOutcome) bool {
	return false
}

// suspendedRunOutcome carries Harness.Run's three return values across
// the goroutine boundary through a channel — never a shared variable read
// without a happens-before edge (the same posture harness_steering_test.go's
// resultCh uses).
type suspendedRunOutcome struct {
	msg    ai.Message
	finish ai.FinishReason
	err    error
}

// driveSuspendedRun is Phase 5's shared harness. It builds a tool-calling
// turn one whose one call the policy defers and a plain-text terminal
// turn two, then drives them through a Harness whose Scheduler the caller
// also holds — the live handle R-RUN-010 requires the run driver to keep.
// It reads the consumer sink event by event (synchronization by channel
// reads only, never a wall clock — NFR-RUN-002): on
// permission_decision_required for callID it proves the run has not
// returned (a non-blocking read of the run's completion channel) and the
// tool has not been invoked, sets wakeIssued, then wakes the parked call.
// It returns the complete recorded stream and the run's outcome, both
// safe to read because both crossed the goroutine boundary only through a
// channel close/receive.
func driveSuspendedRun(t *testing.T, callID, toolName string) ([]agent.Event, suspendedRunOutcome, *deferThenAllowPolicy) {
	t.Helper()

	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, callID, toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	wakeIssued := &atomic.Bool{}
	policy := &deferThenAllowPolicy{wakeIssued: wakeIssued}

	sched := &agent.Scheduler{}
	h := agent.Harness{
		Provider:  provider,
		System:    "system prompt for permission-suspension across a wake",
		Turn:      agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
		Scheduler: sched,
	}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan suspendedRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- suspendedRunOutcome{msg: msg, finish: finish, err: err}
	}()

	var events []agent.Event
	sawDecisionRequired := false
	for {
		ev, ok := <-sink
		if !ok {
			break
		}
		events = append(events, *ev)
		if sawDecisionRequired {
			continue
		}
		req, isReq := ev.PermissionDecisionRequired()
		if !isReq || req.CallID() != callID {
			continue
		}
		sawDecisionRequired = true

		// Non-blocking read of the run's completion channel: the call
		// goroutine is genuinely parked inside Schedule, so Run cannot
		// have returned yet — this is deterministic, not merely likely,
		// because nothing releases the parked select but WakeParked or
		// context cancellation, and this test does neither until here.
		select {
		case <-resultCh:
			t.Fatal("Run's completion channel already carries a result when decision_required was read — the run must not have returned yet")
		default:
		}
		if inv := tool.Invocations(); inv != 0 {
			t.Fatalf("tool invocations = %d, want 0 before the parked call is woken", inv)
		}

		wakeIssued.Store(true)
		if err := sched.WakeParked(callID); err != nil {
			t.Fatalf("WakeParked(%q) = %v, want nil", callID, err)
		}
	}
	if !sawDecisionRequired {
		t.Fatal("never observed permission_decision_required on the consumer stream")
	}

	res := <-resultCh
	return events, res, policy
}

// AG-13 — S-RUN-090, jointly S-APP-015. Fourth acceptance clause
// (0003:1302). Given a permission policy that defers the first
// resolution and allows the second, a caller-owned scheduler injected
// into the run, and a test reading the consumer sink event by event, when
// the decision-required event is read, then the run has not returned
// (checked by a non-blocking read of the run's completion channel) and
// the tool has not been invoked; and when WakeParked is called for that
// call identity, then it returns nil, the call executes, the run
// completes with the completed run outcome, the suspension's events lie
// inside turn one's bracket, and CheckStream accepts the whole stream
// unmodified. Jointly (S-APP-015): the policy's second resolution
// observes the wake-issued flag the test set immediately before
// WakeParked ran — happens-after the wake through parkCh's close, which
// is only true if the call genuinely waited.
func TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake(t *testing.T) {
	t.Parallel()

	events, res, policy := driveSuspendedRun(t, "call-run-090", "defer_tool_run_090")

	if res.err != nil {
		t.Fatalf("Run returned err = %v, want nil", res.err)
	}
	if res.finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v (turn two's real terminal, past the suspension)", res.finish, ai.FinishReasonStop)
	}
	if len(res.msg.Content()) == 0 {
		t.Error("Run returned a message with zero content parts, want turn two's text")
	}

	if !policy.flagAtSecondResolve.Load() {
		t.Error("policy's second resolution observed the wake-issued flag = false, want true — the flag read must be happens-after the wake through the parked channel's close, which is only true if the call genuinely waited (S-APP-015)")
	}

	if len(events) == 0 {
		t.Fatal("zero events observed")
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("last event kind = %v, want run_end", last.Kind())
	}
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("run_end outcome = %v, want RunOutcomeCompleted — the run must survive the suspension", runEnd.Outcome())
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Fatalf("CheckStream rejected the suspension stream: %v", report.Violation())
	}

	// The suspension's events lie inside turn one's bracket: locate the
	// first turn_start/turn_end pair (turn one's, since it is the first
	// to open and close) and confirm every permission/tool event for
	// this call falls strictly between them.
	turnStartIdx, turnEndIdx := -1, -1
	for i, ev := range events {
		if ev.Kind() == agent.EventKindTurnStart && turnStartIdx == -1 {
			turnStartIdx = i
		}
		if ev.Kind() == agent.EventKindTurnEnd && turnEndIdx == -1 {
			turnEndIdx = i
		}
	}
	if turnStartIdx == -1 || turnEndIdx == -1 || turnStartIdx >= turnEndIdx {
		t.Fatalf("turn one's bracket not found or malformed: turn_start at %d, turn_end at %d", turnStartIdx, turnEndIdx)
	}

	var sawRequired, sawMade, sawToolStart, sawToolEnd bool
	for i, ev := range events {
		inTurnOne := i > turnStartIdx && i < turnEndIdx
		if req, ok := ev.PermissionDecisionRequired(); ok && req.CallID() == "call-run-090" {
			sawRequired = true
			if !inTurnOne {
				t.Errorf("permission_decision_required at index %d, want strictly inside turn one's bracket (%d, %d)", i, turnStartIdx, turnEndIdx)
			}
		}
		if made, ok := ev.PermissionDecisionMade(); ok && made.CallID() == "call-run-090" {
			sawMade = true
			if !inTurnOne {
				t.Errorf("permission_decision_made at index %d, want strictly inside turn one's bracket (%d, %d)", i, turnStartIdx, turnEndIdx)
			}
		}
		if start, ok := ev.ToolStart(); ok && start.CallID() == "call-run-090" {
			sawToolStart = true
			if !inTurnOne {
				t.Errorf("tool_start at index %d, want strictly inside turn one's bracket (%d, %d)", i, turnStartIdx, turnEndIdx)
			}
		}
		if end, ok := ev.ToolEndSuccess(); ok && end.CallID() == "call-run-090" {
			sawToolEnd = true
			if !inTurnOne {
				t.Errorf("tool_end_success at index %d, want strictly inside turn one's bracket (%d, %d)", i, turnStartIdx, turnEndIdx)
			}
		}
	}
	if !sawRequired || !sawMade || !sawToolStart || !sawToolEnd {
		t.Fatalf("missing suspension event(s) for call-run-090: required=%v made=%v toolStart=%v toolEnd=%v", sawRequired, sawMade, sawToolStart, sawToolEnd)
	}
}

// AG-13 — S-RUN-091 (bite), jointly S-APP-016. The inherited R-APP-002
// parked-wait bite. agent-permission-protocol/spec.md:172, quoted
// verbatim by R-RUN-010: "the missing bite must observe the parked
// **wait**, not the registration." Against unmodified code this test
// passes exactly like TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake
// — its own non-vacuity is proven separately (recorded in
// apply-progress.md, not by any conditional logic in this test) by a
// scratch edit to scheduler.go's parked-wait select — replacing the
// select on parkCh with an immediate re-resolution — run under
// `go test -race -count=15`, which fails this exact test because the
// policy's second resolution then observes the wake-issued flag unset.
// That failure is the proof the property genuinely depends on the call
// having WAITED for WakeParked's close of parkCh, not merely on having
// been registered: registration happens long before decision_required is
// even emitted, so an assertion satisfiable by registration alone would
// re-encode the exact gap this bite exists to close.
func TestHarness_PermissionDefer_ParkedWaitObservedBite(t *testing.T) {
	t.Parallel()

	_, res, policy := driveSuspendedRun(t, "call-run-091", "defer_tool_run_091")

	if res.err != nil {
		t.Fatalf("Run returned err = %v, want nil", res.err)
	}
	if !policy.flagAtSecondResolve.Load() {
		t.Error("policy's second resolution observed the wake-issued flag = false, want true — under the S-RUN-091 scratch (parked wait replaced with an immediate re-resolution) this is the assertion that fails, proving the property observes the parked WAIT rather than the registration")
	}
}
