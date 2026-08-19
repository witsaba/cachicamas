// AG-19 — Phase 1 (U1): the seam's lifetime is exactly the hosting
// call, and revocation serializes with the publish itself (R-DEL-003,
// design AD-1).
package agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-DEL-006 — revocation on the normal path. Given a tool that
// captures its seam into a value outliving its Run frame and
// publishes an admissible event after Run returned, when that publish
// happens, then it returns the revoked sentinel, delivers nothing,
// and the parent's run completes normally with a CheckStream-valid
// stream.
func TestDelegationSeam_Revocation_NormalPath_PublishAfterReturnIsRevoked(t *testing.T) {
	t.Parallel()

	toolName := "del006_tool"
	tool := &seamCapturingTool{toolName: toolName}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "del006-call", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for del-006", Turn: agent.TurnOptions{Tools: reg}}
	sink := make(chan *agent.Event, 256)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	seam, ok := tool.Seam()
	if !ok {
		t.Fatal("tool captured no seam — cannot exercise the normal-path revocation")
	}

	msgID := mustFreshMessageID(t)
	run := agent.RunID("run-del-006")
	turn := agent.TurnID("turn-del-006")
	ev, err := agent.NewMessageStartText(run, turn, msgID)
	if err != nil {
		t.Fatalf("agent.NewMessageStartText: %v", err)
	}

	if err := seam.Publish(ev); !errors.Is(err, agent.ErrDelegationRevoked) {
		t.Errorf("Publish after Run returned = %v, want errors.Is(_, ErrDelegationRevoked)", err)
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream(parent stream) = %v, want nil (the late publish delivered nothing)", report.Violation())
	}
}

// S-DEL-008 — revocation on the re-panic path. Given a tool that
// captures its seam and then panics, when the scheduler's recovery
// reports the typed panic failure, then a later publish through the
// captured seam returns the revoked sentinel; and the parent's stream
// carries the panic's typed tool failure exactly as it does without a
// seam installed.
func TestDelegationSeam_Revocation_RePanicPath_PublishAfterRecoveredPanicIsRevoked(t *testing.T) {
	t.Parallel()

	toolName := "del008_tool"
	tool := &seamCapturingTool{toolName: toolName, panicMsg: "del-008 deliberate panic"}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: "del008-call", name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sink := make(chan *agent.Event, 16)
	sched := &agent.Scheduler{}
	results := sched.Schedule(contextBackground(), calls, reg, "run-del-008", "turn-del-008", nil, &agent.LaneStamper{}, sink)
	drainSink(t, sink)

	if len(results) != 1 {
		t.Fatalf("Schedule returned %d result(s), want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[0].Outcome = %v, want ToolOutcomeExecutionFailure (the panic recovered as a typed failure)", results[0].Outcome)
	}
	if results[0].Failure == nil {
		t.Fatal("results[0].Failure = nil, want a typed *Failure carrying the recovered panic")
	}

	seam, ok := tool.Seam()
	if !ok {
		t.Fatal("tool captured no seam before panicking — cannot exercise the re-panic-path revocation")
	}

	ev, err := agent.NewMessageStartText("run-del-008", "turn-del-008", mustFreshMessageID(t))
	if err != nil {
		t.Fatalf("agent.NewMessageStartText: %v", err)
	}
	if err := seam.Publish(ev); !errors.Is(err, agent.ErrDelegationRevoked) {
		t.Errorf("Publish after recovered panic = %v, want errors.Is(_, ErrDelegationRevoked)", err)
	}
}

// S-DEL-009 — the two refusals are distinguishable. Given a live seam
// and a revoked seam, when an inadmissible kind is published through
// the live one and an admissible kind through the revoked one, then
// the two returned errors match different sentinels by errors.Is and
// neither matches the other's.
func TestDelegationSeam_TwoRefusals_DistinguishableByErrorsIs(t *testing.T) {
	t.Parallel()

	toolName := "del009_tool"
	var liveErr error
	run := agent.RunID("run-del-009")
	turn := agent.TurnID("turn-del-009")
	tool := &seamCapturingTool{
		toolName: toolName,
		beforeReturn: func(seam agent.DelegationSeam) {
			// An inadmissible kind (run_start opens a bracket, gate 2) while the seam is still live.
			inadmissible, err := agent.NewRunStart(run)
			if err != nil {
				t.Fatalf("agent.NewRunStart: %v", err)
			}
			liveErr = seam.Publish(inadmissible)
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: "del009-call", name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sink := make(chan *agent.Event, 16)
	sched := &agent.Scheduler{}
	_ = sched.Schedule(contextBackground(), calls, reg, run, turn, nil, &agent.LaneStamper{}, sink)
	drainSink(t, sink)

	seam, ok := tool.Seam()
	if !ok {
		t.Fatal("tool captured no seam — cannot exercise the distinguishable-refusals scenario")
	}

	admissible, err := agent.NewMessageStartText(run, turn, mustFreshMessageID(t))
	if err != nil {
		t.Fatalf("agent.NewMessageStartText: %v", err)
	}
	revokedErr := seam.Publish(admissible)

	if !errors.Is(liveErr, agent.ErrDelegationInadmissible) {
		t.Errorf("live-seam publish of an inadmissible kind = %v, want errors.Is(_, ErrDelegationInadmissible)", liveErr)
	}
	if !errors.Is(revokedErr, agent.ErrDelegationRevoked) {
		t.Errorf("revoked-seam publish of an admissible kind = %v, want errors.Is(_, ErrDelegationRevoked)", revokedErr)
	}
	if errors.Is(liveErr, agent.ErrDelegationRevoked) {
		t.Error("live-seam publish's error unexpectedly also matches ErrDelegationRevoked")
	}
	if errors.Is(revokedErr, agent.ErrDelegationInadmissible) {
		t.Error("revoked-seam publish's error unexpectedly also matches ErrDelegationInadmissible")
	}
}

// detachedPublishTool ignores ctx.Done() and blocks until gate
// closes, then publishes an admissible event through its captured
// seam and reports the result on resultCh — S-DEL-007's fixture. gate
// is closed by the TEST only after Schedule has already returned, so
// the emission channel is provably already closed
// (scheduler.go:240 precedes :249) by the time the publish is
// attempted: this is what proves the detached path's revocation,
// not merely a timing coincidence.
type detachedPublishTool struct {
	toolName string
	gate     chan struct{}
	resultCh chan error
}

func (d *detachedPublishTool) Name() string                   { return d.toolName }
func (d *detachedPublishTool) EffectClass() agent.EffectClass { return agent.EffectClassRead }
func (d *detachedPublishTool) Run(ctx context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
	seam, ok := agent.DelegationSeamFrom(ctx)
	if !ok {
		d.resultCh <- errors.New("detachedPublishTool: no seam in context")
		return agent.Result{Outcome: agent.ToolOutcomeSuccess}, nil
	}
	// Deliberately ignores ctx.Done(): this is what makes the
	// scheduler's own wind-down bound abandon this call's Run
	// frame (scheduler.go:627-633) rather than ever observe a
	// timely return (R-DEL-003's "detached path").
	<-d.gate
	run, turn := seam.Parent()

	// The actual late publish happens in a GRANDCHILD goroutine Run
	// spawns and does not join — deliberately outside
	// runToolWithWindDown's own recover, which wraps only the
	// synchronous body of this Run call, never a goroutine Run
	// spawns and returns without waiting for. This is the shape
	// design.md's AD-1 names ("a tool that spawns its own goroutine
	// holding the seam and returns") and it is what makes S-DEL-022's
	// bite a genuine unrecovered panic rather than one
	// runToolWithWindDown's existing recover would silently absorb.
	go func() {
		part, err := ai.NewText("del-007-fixture")
		if err != nil {
			d.resultCh <- err
			return
		}
		msg, err := ai.NewMessage(ai.RoleAssistant, part)
		if err != nil {
			d.resultCh <- err
			return
		}
		ev, err := agent.NewMessageStartText(run, turn, msg.ID())
		if err != nil {
			d.resultCh <- err
			return
		}
		d.resultCh <- seam.Publish(ev)
	}()
	return agent.Result{Outcome: agent.ToolOutcomeSuccess}, nil
}

var _ agent.Tool = (*detachedPublishTool)(nil)

// S-DEL-007 — revocation on the detached path. Given a scheduler with
// an injected small wind-down bound and a tool that ignores its
// context and publishes only after a gate the test closes once
// Schedule has returned — so the emission channel is provably already
// closed — when that publish happens, then it returns the revoked
// sentinel, no panic occurs, and the test process survives to assert
// it.
func TestDelegationSeam_Revocation_DetachedPath_PublishAfterCloseIsRevoked(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{})
	resultCh := make(chan error, 1)
	toolName := "del007_tool"
	tool := &detachedPublishTool{toolName: toolName, gate: gate, resultCh: resultCh}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: "del007-call", name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sched := &agent.Scheduler{WindDownBound: 20 * time.Millisecond}
	sink := make(chan *agent.Event, 16)

	// runToolWithWindDown only arms the wind-down bound once ctx.Done()
	// fires (scheduler.go's detach select) — a short deadline here is
	// what makes the call detach at all, deterministically, well
	// before the tool's own <-d.gate could ever release it.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	results := sched.Schedule(ctx, calls, reg, "run-del-007", "turn-del-007", nil, &agent.LaneStamper{}, sink)
	// Schedule has returned: close(emissions) already ran
	// (scheduler.go:240), strictly before this line.
	drainSink(t, sink)

	if len(results) != 1 {
		t.Fatalf("Schedule returned %d result(s), want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeExecutionFailure {
		t.Errorf("results[0].Outcome = %v, want ToolOutcomeExecutionFailure (the call detached)", results[0].Outcome)
	}

	close(gate)
	select {
	case err := <-resultCh:
		if !errors.Is(err, agent.ErrDelegationRevoked) {
			t.Errorf("detached publish returned %v, want errors.Is(_, ErrDelegationRevoked) — no panic, no silent drop", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("detached tool's publish never completed — want ErrDelegationRevoked within 5s")
	}
}
