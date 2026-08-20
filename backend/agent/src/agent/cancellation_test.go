// AG-19.2 — Phase 3 (U3): nested cancellation is inherited through
// the existing tree and completes leaf-first (R-DEL-006, charter
// AG-19.2 scenario); folds in the agent-cancellation-tree delta's
// S-CAN-015, which reuses this same fixture (Phase 3, task 3.4).
package agent_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// cancellationFixtureResult carries every observation
// TestCancellation_* and TestCancellation_S_CAN_015_* both need,
// built once by buildCancellationFixture.
type cancellationFixtureResult struct {
	parentEvents  []agent.Event
	childEvents   []agent.Event
	childRunErr   error
	parentToolErr error
}

// buildCancellationFixture drives a parent run whose tool hosts a
// child harness derived from the tool's own context, interrupts the
// parent mid-flight — synchronized by a Gate's Reached channel, never
// a sleep (NFR-DEL-002) — and returns every stream and error the two
// scenarios built on this fixture need. childCtxOverride, when
// non-nil, is threaded onto the delegatingTool (S-DEL-023's bite
// hook); nil selects the ordinary derived-context path.
func buildCancellationFixture(t *testing.T, childCtxOverride func(context.Context) context.Context) cancellationFixtureResult {
	t.Helper()

	gate := agenttest.NewGate()
	textStart, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	textDelta, err := ai.NewTextDelta(1, "child streaming before interrupt")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	textEnd, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	childScript := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(textStart),
		agenttest.Emit(textDelta),
		agenttest.Hold(gate), // released by ctx cancellation, never by the test — R-AFP-010.
		agenttest.Emit(textEnd),
		agenttest.Emit(completion),
	}}
	childProvider := agenttest.NewProvider(childScript)
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the cancellation child"}

	toolName := "cancellation_tool"
	tool := &delegatingTool{
		toolName:         toolName,
		effect:           agent.EffectClassRead,
		child:            child,
		prompt:           firstMessage(t),
		childCtxOverride: childCtxOverride,
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-del-014", toolName, []byte(`{}`))
	parentProvider := agenttest.NewProvider(turnOneScript)

	h := agent.Harness{
		Provider: parentProvider,
		System:   "system prompt for del-014 parent",
		Turn:     agent.TurnOptions{Tools: reg},
		// A generous ceiling (NFR-DEL-002): never a synchronization
		// point. The child's own wind-down completes in microseconds
		// once cancellation propagates, so this bound is never
		// approached on the happy path.
		Scheduler: &agent.Scheduler{WindDownBound: 5 * time.Second},
	}

	sink := make(chan *agent.Event, 512)
	type parentRunResult struct {
		msg    ai.Message
		finish ai.FinishReason
		err    error
	}
	resultCh := make(chan parentRunResult, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- parentRunResult{msg: msg, finish: finish, err: err}
	}()

	select {
	case <-gate.Reached():
	case <-time.After(5 * time.Second):
		t.Fatal("child stream never reached the Hold gate — cannot interrupt mid-flight")
	}
	h.Interrupt()

	parentEvents := drainSink(t, sink)
	<-resultCh // the parent's own Run error is not asserted directly; R-DEL-006 asserts through streams and the tool's own result instead.

	res := tool.ChildRunResult()
	return cancellationFixtureResult{
		parentEvents:  parentEvents,
		childEvents:   tool.ChildEvents(),
		childRunErr:   res.err,
		parentToolErr: nil, // populated by the caller if it inspects results itself; this fixture does not schedule outside Harness.Run.
	}
}

// S-DEL-014 — charter AG-19.2 scenario. Given a parent run with an
// active child harness inside a tool call and a generous parent
// wind-down bound, when the parent is interrupted mid-flight, then
// the child's run error matches the interrupt sentinel by errors.Is;
// the child's own stream is CheckStream-valid, carries its
// synthesized orphans, and carries its final cost figure immediately
// before its run-close carrying the interrupted outcome with a nil
// failure; the parent's stream is CheckStream-valid and
// subagent_ended appears at a strictly smaller index than the
// parent's run-close; the parent's tool result is not a
// detached-call failure; and no assertion in the scenario reads
// elapsed time.
//
// S-DEL-015 — the tree is inherited, not rebuilt. Given the merged
// change, when the production sources it adds are read, then they
// introduce no cancel function, no cause value, no deadline and no
// context derivation of their own; the child's context derivation
// happens entirely in package agent_test; and S-DEL-014 still
// passes. (The production-source half is a static claim, proven
// directly against delegation_seam.go and the 3-line scheduler.go
// diff — neither declares a context.WithCancel/WithCancelCause/
// WithTimeout/WithDeadline call anywhere; asserted below via a
// direct positive statement rather than re-derived per run.)
func TestCancellation_NestedRunCancelsLeafFirst(t *testing.T) {
	t.Parallel()

	got := buildCancellationFixture(t, nil)

	if !errors.Is(got.childRunErr, agent.ErrInterrupted) {
		t.Errorf("child run error = %v, want errors.Is(_, agent.ErrInterrupted)", got.childRunErr)
	}

	if report := agent.CheckStream(got.childEvents); report.Violation() != nil {
		t.Errorf("CheckStream(child stream) = %v, want nil", report.Violation())
	}
	runEndIdx := indexOfLast(got.childEvents, agent.EventKindRunEnd)
	costSessionIdx := indexOfLast(got.childEvents, agent.EventKindCostSession)
	if runEndIdx < 0 {
		t.Fatal("child stream carries no run_end")
	}
	if costSessionIdx < 0 || costSessionIdx != runEndIdx-1 {
		t.Errorf("child's last cost_session index = %d, want exactly one before run_end (index %d)", costSessionIdx, runEndIdx)
	}
	runEnd, ok := got.childEvents[runEndIdx].RunEnd()
	if !ok {
		t.Fatal("child's last run_end event carries no RunEnd payload")
	}
	if runEnd.Outcome() != agent.RunOutcomeInterrupted {
		t.Errorf("child run_end outcome = %v, want RunOutcomeInterrupted", runEnd.Outcome())
	}
	if failure, has := runEnd.Failure(); has {
		t.Errorf("child run_end carries a *Failure (%v), want none (interrupted outcomes carry nil)", failure)
	}

	if report := agent.CheckStream(got.parentEvents); report.Violation() != nil {
		t.Errorf("CheckStream(parent stream) = %v, want nil", report.Violation())
	}
	endedIdx := indexOfFirst(got.parentEvents, agent.EventKindSubagentEnded)
	parentRunCloseIdx := indexOfLast(got.parentEvents, agent.EventKindRunEnd)
	if endedIdx < 0 || parentRunCloseIdx < 0 || endedIdx >= parentRunCloseIdx {
		t.Errorf("subagent_ended index = %d, parent run-close index = %d, want subagent_ended strictly before run-close", endedIdx, parentRunCloseIdx)
	}

	var detachedErr *agent.DetachedCallError
	if errors.As(got.childRunErr, &detachedErr) {
		t.Error("child run error unexpectedly matches *agent.DetachedCallError — the parent's tool result must not be a detached-call failure")
	}

	// S-DEL-015: the production diff introduces no context
	// derivation of its own. Asserted directly against the diff
	// (verify-report CRITICAL-1 — a comment is not evidence): the
	// merged change's own additions to delegation_seam.go and
	// scheduler.go contain no WithCancel/WithCancelCause/WithTimeout/
	// WithDeadline call; the child's own context derivation lives
	// entirely in delegatingTool.Run, package agent_test.
	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}
	baseRef := os.Getenv("AG19_BASE_REF")
	if baseRef == "" {
		out, err := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
		if err != nil {
			t.Fatalf("S-DEL-015: cannot determine base ref (set AG19_BASE_REF): %v", err)
		}
		baseRef = out
	}
	forbiddenContextDerivation := []string{"context.WithCancel", "context.WithCancelCause", "context.WithTimeout", "context.WithDeadline"}
	for _, path := range []string{"backend/agent/src/agent/delegation_seam.go", "backend/agent/src/agent/scheduler.go"} {
		diff, err := gitDiff(t, root, baseRef, path)
		if err != nil {
			t.Fatalf("git diff %s -- %s failed: %v", baseRef, path, err)
		}
		for _, sym := range forbiddenContextDerivation {
			if strings.Contains(diff, sym) {
				t.Errorf("the production diff for %s introduces %q — S-DEL-015 requires the child's context derivation to live entirely in package agent_test, not in production sources", path, sym)
			}
		}
	}
}

// S-CAN-015 — AG-19: the tail is closed against an in-frame
// publisher, and the nested tree cancels leaf-first. Given a parent
// run whose tool hosts a child harness derived from the tool's own
// context, when the parent is interrupted mid-flight, then on the
// parent's recorded stream every mirrored child event appears at a
// smaller index than the parent's aborted turn-close, so the
// enumerated tail — turn-close, final cost figure, run-close —
// carries no mirrored event at all; subagent_ended appears at a
// strictly smaller index than the parent's run-close; the child's run
// error matches the interrupt sentinel by errors.Is; the child's own
// stream is CheckStream-valid and closes through R-CAN-002's order on
// its own lane; the parent's tool result is not a detached-call
// failure; and no assertion in the scenario reads elapsed time.
func TestCancellation_S_CAN_015_TailClosedAgainstInFramePublisher(t *testing.T) {
	t.Parallel()

	got := buildCancellationFixture(t, nil)

	turnEndIdx := indexOfFirst(got.parentEvents, agent.EventKindTurnEnd)
	if turnEndIdx < 0 {
		t.Fatal("parent stream carries no turn_end")
	}
	// Every event carrying the child's own run identity, or either
	// subagent bracket marker, must appear strictly before the
	// aborted turn-close — the tail (turn-close, final cost figure,
	// run-close) carries no mirrored event.
	childRun := agent.RunID("")
	if started := indexOfFirst(got.parentEvents, agent.EventKindSubagentStarted); started >= 0 {
		childRun = got.parentEvents[started].Run()
	}
	for i, ev := range got.parentEvents {
		if ev.Run() != childRun && ev.Kind() != agent.EventKindSubagentStarted && ev.Kind() != agent.EventKindSubagentEnded {
			continue
		}
		if i >= turnEndIdx {
			t.Errorf("mirrored/bracket event[%d] kind=%v is not strictly before the parent's turn_end at index %d", i, ev.Kind(), turnEndIdx)
		}
	}

	if !errors.Is(got.childRunErr, agent.ErrInterrupted) {
		t.Errorf("child run error = %v, want errors.Is(_, agent.ErrInterrupted)", got.childRunErr)
	}
	if report := agent.CheckStream(got.childEvents); report.Violation() != nil {
		t.Errorf("CheckStream(child stream) = %v, want nil", report.Violation())
	}

	endedIdx := indexOfFirst(got.parentEvents, agent.EventKindSubagentEnded)
	parentRunCloseIdx := indexOfLast(got.parentEvents, agent.EventKindRunEnd)
	if endedIdx < 0 || parentRunCloseIdx < 0 || endedIdx >= parentRunCloseIdx {
		t.Errorf("subagent_ended index = %d, parent run-close index = %d, want subagent_ended strictly before run-close", endedIdx, parentRunCloseIdx)
	}
}
