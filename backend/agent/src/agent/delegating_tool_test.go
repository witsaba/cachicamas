// AG-19 — Phase 2 (U2): the delegating test tool. Everything that
// knows what a "subagent" is lives here, in package agent_test, which
// production code cannot import (R-DEL-001, R-DEL-009) — the
// scripted_tool_test.go precedent restated for this milestone's own
// fixture.
//
// delegatingTool hosts a child agent.Harness inside its Run frame:
// the child runs on a context derived from the tool's own ctx (unless
// overridden, the bite S-DEL-023 uses), its events are mirrored onto
// the parent's lane through the installed DelegationSeam, and its own
// captured stream is kept for the test to validate separately
// (R-DEL-005's split-then-validate rule).
package agent_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// nextDelegationSubagentID mints a fresh, deterministic per-process
// subagent identity — a package-level counter rather than reading
// PolicySlot as a call identity, so this fixture stays decoupled from
// that AG-09-era convention (tool.go's own doc names it interim,
// pending a Layer 3 sandbox).
var delegationSubagentIDCounter atomic.Uint64

func nextDelegationSubagentID() string {
	return "subagent-" + itoa(int(delegationSubagentIDCounter.Add(1)))
}

// childRunResult carries a hosted child Harness.Run's three return
// values across the goroutine boundary, mirroring
// harness_suspension_test.go's suspendedRunOutcome.
type childRunResult struct {
	msg    ai.Message
	finish ai.FinishReason
	err    error
}

// delegatingTool hosts one child agent.Harness per Run call. Two
// distinct delegatingTool VALUES give AG-19.1's siblings scenario two
// distinct Harness values (R-DEL-005, R-RUN-001's one-run-at-a-time
// clause stays out of scope: this fixture never drives two Run calls
// on the SAME Harness value).
type delegatingTool struct {
	toolName         string
	effect           agent.EffectClass
	child            *agent.Harness
	prompt           ai.Message
	childCtxOverride func(parentCtx context.Context) context.Context

	mu          sync.Mutex
	childRun    agent.RunID
	childEvents []agent.Event
	childResult childRunResult
	publishErrs []error
}

func (d *delegatingTool) Name() string                   { return d.toolName }
func (d *delegatingTool) EffectClass() agent.EffectClass { return d.effect }

// Run implements agent.Tool. It reaches the installed DelegationSeam,
// runs the child Harness on a goroutine (so the child's own emission
// sends never deadlock against this method's own mirroring drain),
// mirrors every child event through the seam in order — publishing
// exactly one subagent_started before the first mirrored attempt and
// exactly one subagent_ended only after the child's own sink has
// closed (R-DEL-004, R-AGE-010's ordering) — and returns a typed
// Result summarizing the hosted run.
func (d *delegatingTool) Run(ctx context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
	seam, ok := agent.DelegationSeamFrom(ctx)
	if !ok {
		return agent.Result{}, errDelegatingToolNoSeam
	}
	parentRun, parentTurn := seam.Parent()

	childCtx := ctx
	if d.childCtxOverride != nil {
		childCtx = d.childCtxOverride(ctx)
	}

	childSink := make(chan *agent.Event, 256)
	resultCh := make(chan childRunResult, 1)
	go func() {
		msg, finish, err := d.child.Run(childCtx, d.prompt, childSink)
		resultCh <- childRunResult{msg: msg, finish: finish, err: err}
	}()

	subagentID := nextDelegationSubagentID()
	var childRun agent.RunID
	var captured []agent.Event
	var pubErrs []error
	first := true
	for ev := range childSink {
		captured = append(captured, *ev)
		if first {
			childRun = ev.Run()
			startedEv, serr := agent.NewSubagentStarted(childRun, parentRun, parentTurn, subagentID)
			if serr != nil {
				pubErrs = append(pubErrs, serr)
			} else if perr := seam.Publish(startedEv); perr != nil {
				pubErrs = append(pubErrs, perr)
			}
			first = false
		}
		if perr := seam.Publish(*ev); perr != nil {
			pubErrs = append(pubErrs, perr)
		}
	}

	// subagent_ended is published only AFTER the loop above observed
	// childSink close — which happens only after the child's own
	// run-close was sent (harness.go:446) — so this ordering
	// structurally proves the child's stream is complete before the
	// parent learns it ended (R-AGE-010, R-DEL-006 assertion 3).
	endedEv, eerr := agent.NewSubagentEnded(childRun, parentRun, parentTurn, subagentID)
	if eerr != nil {
		pubErrs = append(pubErrs, eerr)
	} else if perr := seam.Publish(endedEv); perr != nil {
		pubErrs = append(pubErrs, perr)
	}

	res := <-resultCh

	d.mu.Lock()
	d.childRun = childRun
	d.childEvents = captured
	d.childResult = res
	d.publishErrs = pubErrs
	d.mu.Unlock()

	if res.err != nil {
		return agent.Result{}, res.err
	}
	return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("delegated:" + string(childRun))}, nil
}

// ChildRun returns the child's own run identity, valid after Run has
// returned.
func (d *delegatingTool) ChildRun() agent.RunID {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.childRun
}

// ChildEvents returns a copy of the child's own captured stream — the
// "child's own stream" R-DEL-005 requires be validated separately
// from the parent's.
func (d *delegatingTool) ChildEvents() []agent.Event {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]agent.Event, len(d.childEvents))
	copy(out, d.childEvents)
	return out
}

// ChildRunResult returns the hosted child Harness.Run's own three
// return values.
func (d *delegatingTool) ChildRunResult() childRunResult {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.childResult
}

// PublishErrors returns every non-nil error the mirroring loop
// observed from seam.Publish (expected: one per refused kind — the
// child's own run_start, run_end, turn_start, turn_end and any
// cost_turn/permission_resolution_remembered it happens to emit).
func (d *delegatingTool) PublishErrors() []error {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]error, len(d.publishErrs))
	copy(out, d.publishErrs)
	return out
}

var _ agent.Tool = (*delegatingTool)(nil)

// errDelegatingToolNoSeam is delegatingTool's own typed fixture
// defect: this tool must only ever be scheduled through the ordinary
// path that installs a seam. A wrong-context invocation should fail
// loudly rather than run without mirroring anything.
type errDelegatingToolNoSeamType struct{}

func (errDelegatingToolNoSeamType) Error() string {
	return "delegatingTool: no DelegationSeam installed on this call's context"
}

var errDelegatingToolNoSeam = errDelegatingToolNoSeamType{}

// --- Two-stream reconstruction helper (R-DEL-004, R-DEL-005) --------

// validateStreamsSeparately asserts parent and child are each
// CheckStream-valid, checked as two independent slices — never
// concatenated (R-DEL-005: CheckStream's own bracket/terminal/
// at-most-once state is global over the validated slice, so a
// concatenation would present two run brackets to a validator that
// has no way to tell them apart).
func validateStreamsSeparately(t *testing.T, parent, child []agent.Event) {
	t.Helper()
	if report := agent.CheckStream(parent); report.Violation() != nil {
		t.Errorf("CheckStream(parent stream) = %v, want nil", report.Violation())
	}
	if report := agent.CheckStream(child); report.Violation() != nil {
		t.Errorf("CheckStream(child stream) = %v, want nil", report.Violation())
	}
}

// resolveChildToParent performs R-DEL-004's one-hop walk: it finds
// the single subagent_started on parent whose Run() equals childRun,
// and returns the parent identity its Parent() reports. No per-event
// parent field is read or assumed to exist on any other event.
func resolveChildToParent(parent []agent.Event, childRun agent.RunID) (parentRun agent.RunID, found bool) {
	for _, ev := range parent {
		started, ok := ev.SubagentStarted()
		if !ok {
			continue
		}
		if ev.Run() != childRun {
			continue
		}
		p, hasParent := ev.Parent()
		if !hasParent {
			return "", false
		}
		_ = started
		return p, true
	}
	return "", false
}

// countKind returns how many events in events carry kind.
func countKind(events []agent.Event, kind agent.EventKind) int {
	n := 0
	for _, ev := range events {
		if ev.Kind() == kind {
			n++
		}
	}
	return n
}

// indexOfFirst returns the 0-based index of the first event in events
// carrying kind, or -1.
func indexOfFirst(events []agent.Event, kind agent.EventKind) int {
	for i, ev := range events {
		if ev.Kind() == kind {
			return i
		}
	}
	return -1
}

// indexOfLast returns the 0-based index of the last event in events
// carrying kind, or -1.
func indexOfLast(events []agent.Event, kind agent.EventKind) int {
	idx := -1
	for i, ev := range events {
		if ev.Kind() == kind {
			idx = i
		}
	}
	return idx
}
