// AG-23.1 — the consumer proof (R-L3H-001): the future Layer 3 in
// miniature. This test package is neither Layer 2's own package
// (src/agent) nor its Layer 1 substrate (src/agenttest) — it is an
// external, genuinely third-party package one level up, built entirely
// from the public surface: agent.Harness, agenttest's scripted provider,
// and apptest's scripted policy, scripted tool and drain helper.
//
// Every stage below is named for the acceptance capability it
// discharges (S-L3H-002), runs SEQUENTIALLY over shared state — never in
// parallel, because stage 6's hand-off from the first harness's own
// transcript is ordered and a parallel stage would leave that ordering
// unobserved rather than proven — and every drain is validated through
// apptest.DrainAndCheck or agent.CheckStream directly, never
// re-implemented.
package layer3handoff_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/apptest"
)

// l3hFirstMessage builds a minimal user message through the public ai
// constructors only.
func l3hFirstMessage(t *testing.T) ai.Message {
	t.Helper()
	part, err := ai.NewText("hello from the consumer proof")
	if err != nil {
		t.Fatalf("ai.NewText error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage error = %v, want nil", err)
	}
	return msg
}

// l3hResumeMessage builds the stage-5 resumed prompt — a distinct
// message from l3hFirstMessage's, so a reader can tell the two turns
// apart without inference.
func l3hResumeMessage(t *testing.T) ai.Message {
	t.Helper()
	part, err := ai.NewText("please continue")
	if err != nil {
		t.Fatalf("ai.NewText error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage error = %v, want nil", err)
	}
	return msg
}

// l3hToolCallScript scripts one provider turn requesting a tool call.
func l3hToolCallScript(t *testing.T, callID, toolName string, args []byte) agenttest.Script {
	t.Helper()
	start, err := ai.NewToolCallStart(1, callID, toolName)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart error = %v, want nil", err)
	}
	delta, err := ai.NewToolCallDelta(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta error = %v, want nil", err)
	}
	end, err := ai.NewToolCallEnd(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd error = %v, want nil", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion error = %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// l3hTextScript scripts one provider turn returning a short completed
// text response.
func l3hTextScript(t *testing.T, text string) agenttest.Script {
	t.Helper()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart error = %v, want nil", err)
	}
	delta, err := ai.NewTextDelta(1, text)
	if err != nil {
		t.Fatalf("ai.NewTextDelta error = %v, want nil", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd error = %v, want nil", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion error = %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// l3hHoldScript scripts one provider turn that blocks at gate instead of
// emitting anything — the fixture the interrupt stage drives.
func l3hHoldScript(gate *agenttest.Gate) agenttest.Script {
	return agenttest.Script{Steps: []agenttest.Step{agenttest.Hold(gate)}}
}

// TestLayer3Handoff_ConsumerProof is R-L3H-001's own driver: one ordered
// run of every acceptance capability the charter names, over shared
// state, never parallel.
func TestLayer3Handoff_ConsumerProof(t *testing.T) {
	const toolCallID = "call-l3h-001"
	const toolName = "write_l3h_001"

	hist := agent.NewHistory()
	sched := &agent.Scheduler{}
	policy := apptest.NewScriptedPermissionPolicy(
		agent.PermissionVerdict{Outcome: agent.PermissionDefer},
		agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce},
	)
	toolScript := func(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
		return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: args}, nil
	}
	tool := apptest.NewScriptedTool(toolName, agent.EffectClassMutating, toolScript)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	gate := agenttest.NewGate()

	provider := agenttest.NewProvider(
		l3hToolCallScript(t, toolCallID, toolName, []byte(`{"n":1}`)), // Run #1, turn 1
		l3hTextScript(t, "turn one done"),                             // Run #1, turn 2
		l3hHoldScript(gate),                                           // Run #2 (interrupted before any event)
		l3hTextScript(t, "resumed and done"),                          // Run #3
	)

	h := agent.Harness{
		Provider:  provider,
		System:    "system prompt for the layer3handoff consumer proof",
		Turn:      agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
		Scheduler: sched,
		History:   hist,
	}

	var run1Events []agent.Event
	var run1Msg ai.Message
	var run1Finish ai.FinishReason
	var run2Events []agent.Event
	var run2Report agent.StreamReport
	var run3Events []agent.Event
	var run3Report agent.StreamReport
	var seededEvents []agent.Event
	var seededReport agent.StreamReport

	// Stage 1 — construction from injected fakes, through the public
	// surface only. No stage below reaches an unexported Layer 2
	// identifier: every value above is built from agent's, ai's,
	// agenttest's and apptest's own exported constructors (S-L3H-001,
	// S-L3H-008).
	t.Run("01_construction_from_injected_fakes", func(t *testing.T) {
		if h.Provider == nil {
			t.Fatal("Harness.Provider is nil, want the injected scripted provider")
		}
		if h.Turn.Tools == nil {
			t.Fatal("Harness.Turn.Tools is nil, want the injected registry over the scripted tool")
		}
		if h.Turn.PermissionPolicy == nil {
			t.Fatal("Harness.Turn.PermissionPolicy is nil, want the injected scripted policy")
		}
		if h.Scheduler == nil {
			t.Fatal("Harness.Scheduler is nil, want the caller-owned scheduler handle")
		}
		if h.History == nil {
			t.Fatal("Harness.History is nil, want the caller-owned transcript")
		}
	})

	// Stage 2 + Stage 3 (one Run, two capabilities) — a multi-turn
	// conversation whose first turn's tool call is deferred by the
	// scripted policy and resolved by its own second queued verdict,
	// mirroring src/agent's own harness_suspension_test.go driveSuspendedRun
	// mechanism one layer up: read the consumer sink event by event
	// (synchronization by channel reads only, R-L3H-004's no-wall-clock
	// rule), and on permission_decision_required call Scheduler.WakeParked
	// for the call identity. This stage's own read loop performs exactly
	// what apptest.DrainAndCheck does internally — accumulate to close,
	// then delegate to agent.CheckStream — but inlined, because it must
	// also intercept mid-stream to call WakeParked; every later stage's
	// drain calls apptest.DrainAndCheck directly.
	t.Run("02_multiturn_and_scripted_permission_suspension", func(t *testing.T) {
		sink := make(chan *agent.Event)
		resultCh := make(chan struct {
			msg    ai.Message
			finish ai.FinishReason
			err    error
		}, 1)
		go func() {
			msg, finish, err := h.Run(context.Background(), l3hFirstMessage(t), sink)
			resultCh <- struct {
				msg    ai.Message
				finish ai.FinishReason
				err    error
			}{msg, finish, err}
		}()

		sawRequired := false
		for {
			ev, ok := <-sink
			if !ok {
				break
			}
			run1Events = append(run1Events, *ev)
			if sawRequired {
				continue
			}
			req, isReq := ev.PermissionDecisionRequired()
			if !isReq || req.CallID() != toolCallID {
				continue
			}
			sawRequired = true

			select {
			case <-resultCh:
				t.Fatal("Run #1's completion channel already carries a result when decision_required was read — want the run still suspended")
			default:
			}
			if inv := tool.Invocations(); inv != 0 {
				t.Fatalf("tool invocations = %d before the parked call is woken, want 0", inv)
			}
			if err := sched.WakeParked(toolCallID); err != nil {
				t.Fatalf("WakeParked(%q) = %v, want nil", toolCallID, err)
			}
		}
		if !sawRequired {
			t.Fatal("never observed permission_decision_required for the deferred call")
		}

		res := <-resultCh
		run1Msg, run1Finish = res.msg, res.finish
		if res.err != nil {
			t.Fatalf("Run #1 returned err = %v, want nil", res.err)
		}
		if run1Finish != ai.FinishReasonStop {
			t.Errorf("Run #1 finish = %v, want %v (turn two's own completed finish reason)", run1Finish, ai.FinishReasonStop)
		}
		if len(run1Msg.Content()) == 0 {
			t.Error("Run #1 returned a message with zero content parts, want turn two's text")
		}
		if policy.Exhausted() {
			t.Error("policy.Exhausted() = true after exactly 2 Resolve calls against a 2-entry queue, want false (exhaustion latches only on a call BEYOND the queued count)")
		}
		if got := len(policy.Resolved()); got != 2 {
			t.Errorf("policy.Resolved() returned %d call(s), want 2 (the deferred call, then its own re-entry)", got)
		}
		if got := tool.Invocations(); got != 1 {
			t.Errorf("tool.Invocations() = %d, want 1", got)
		}

		turnBrackets := 0
		for _, ev := range run1Events {
			if ev.Kind() == agent.EventKindTurnStart {
				turnBrackets++
			}
		}
		if turnBrackets < 2 {
			t.Errorf("Run #1 carried %d turn_start event(s), want at least 2 (multi-turn)", turnBrackets)
		}

		if report := agent.CheckStream(run1Events); report.Violation() != nil {
			t.Fatalf("agent.CheckStream rejected Run #1's stream: %v", report.Violation())
		}
	})

	// Stage 4 — an interrupt. Run #2's single turn blocks at gate before
	// emitting anything; the test waits for gate.Reached() (a channel
	// read, never a wall clock) and then calls Harness.Interrupt() —
	// exactly the mechanism the run driver's own interrupt scenarios use.
	// The consumer sink is drained through apptest.DrainAndCheck directly:
	// nothing here needs to intercept mid-stream, because the
	// synchronization point is gate.Reached(), a channel entirely
	// separate from the event sink.
	t.Run("03_interrupt", func(t *testing.T) {
		sink := make(chan *agent.Event)
		drainDone := make(chan struct{})
		go func() {
			defer close(drainDone)
			run2Events, run2Report = apptest.DrainAndCheck(sink)
		}()

		runDone := make(chan struct{})
		var runErr error
		go func() {
			defer close(runDone)
			_, _, runErr = h.Run(context.Background(), l3hFirstMessage(t), sink)
		}()

		<-gate.Reached()
		h.Interrupt()

		<-runDone
		<-drainDone

		if runErr == nil {
			t.Fatal("Run #2 returned err = nil, want the interrupt cause")
		}
		if report := run2Report; report.Violation() != nil {
			t.Fatalf("apptest.DrainAndCheck rejected Run #2's (interrupted) stream: %v", report.Violation())
		}
		if len(run2Events) == 0 {
			t.Fatal("Run #2 produced zero events, want at least run_start and run_end")
		}
		last := run2Events[len(run2Events)-1]
		runEnd, ok := last.RunEnd()
		if !ok {
			t.Fatalf("Run #2's last event kind = %v, want run_end", last.Kind())
		}
		if runEnd.Outcome() != agent.RunOutcomeInterrupted {
			t.Errorf("Run #2's run_end outcome = %v, want RunOutcomeInterrupted", runEnd.Outcome())
		}
	})

	// Stage 5 — a resumed prompt. Run #3 drives a further prompt on the
	// SAME harness value after the interrupt: R-RUN-001's own contract
	// that an interrupted-then-reused Harness accepts a new run.
	t.Run("04_resumed_prompt", func(t *testing.T) {
		sink := make(chan *agent.Event)
		drainDone := make(chan struct{})
		go func() {
			defer close(drainDone)
			run3Events, run3Report = apptest.DrainAndCheck(sink)
		}()

		_, finish, err := h.Run(context.Background(), l3hResumeMessage(t), sink)
		<-drainDone

		if err != nil {
			t.Fatalf("Run #3 returned err = %v, want nil", err)
		}
		if finish != ai.FinishReasonStop {
			t.Errorf("Run #3 finish = %v, want %v", finish, ai.FinishReasonStop)
		}
		if report := run3Report; report.Violation() != nil {
			t.Fatalf("apptest.DrainAndCheck rejected Run #3's stream: %v", report.Violation())
		}
		last := run3Events[len(run3Events)-1]
		runEnd, ok := last.RunEnd()
		if !ok {
			t.Fatalf("Run #3's last event kind = %v, want run_end", last.Kind())
		}
		if runEnd.Outcome() != agent.RunOutcomeCompleted {
			t.Errorf("Run #3's run_end outcome = %v, want RunOutcomeCompleted", runEnd.Outcome())
		}
	})

	// Stage 6 — a second harness constructed over the first's transcript,
	// through the public accessors only: History.Entries() ->
	// Entry.Message() -> agent.NewSeededHistory(msgs) -> a fresh
	// Harness{History: seeded}. No Layer 2 internal was touched to move
	// the transcript.
	t.Run("05_second_harness_over_seeded_transcript", func(t *testing.T) {
		entries := hist.Entries()
		if len(entries) == 0 {
			t.Fatal("first harness's History.Entries() returned zero entries after three runs, want at least one")
		}
		msgs := make([]ai.Message, len(entries))
		for i, e := range entries {
			msgs[i] = e.Message()
		}

		seeded, err := agent.NewSeededHistory(msgs)
		if err != nil {
			t.Fatalf("agent.NewSeededHistory error = %v, want nil", err)
		}

		secondProvider := agenttest.NewProvider(l3hTextScript(t, "second harness done"))
		h2 := agent.Harness{
			Provider: secondProvider,
			System:   "system prompt for the second, seeded harness",
			History:  seeded,
		}

		sink := make(chan *agent.Event)
		drainDone := make(chan struct{})
		go func() {
			defer close(drainDone)
			seededEvents, seededReport = apptest.DrainAndCheck(sink)
		}()

		_, finish, err := h2.Run(context.Background(), l3hResumeMessage(t), sink)
		<-drainDone

		if err != nil {
			t.Fatalf("second harness Run returned err = %v, want nil", err)
		}
		if finish != ai.FinishReasonStop {
			t.Errorf("second harness finish = %v, want %v", finish, ai.FinishReasonStop)
		}
		if report := seededReport; report.Violation() != nil {
			t.Fatalf("apptest.DrainAndCheck rejected the second harness's stream: %v", report.Violation())
		}
		if len(seededEvents) == 0 {
			t.Fatal("second harness produced zero events, want at least run_start and run_end")
		}
	})

	// Stage 7 — the closing assertion (R-L3H-001, S-L3H-002..008): every
	// run this whole sequence drove is, independently, already validated
	// clean by the exported stream checker — re-confirmed here as one
	// explicit closing gate rather than left implicit in each stage
	// above. No stage above ever referenced a helper declared in a Layer
	// 2 test-only compilation unit: doing so would not compile, which is
	// what makes this sufficiency claim falsifiable rather than merely
	// asserted (S-L3H-008).
	t.Run("06_closing_validation_every_drain_clean", func(t *testing.T) {
		if report := agent.CheckStream(run1Events); report.Violation() != nil {
			t.Errorf("Run #1's stream: %v", report.Violation())
		}
		if run2Report.Violation() != nil {
			t.Errorf("Run #2's (interrupted) stream: %v", run2Report.Violation())
		}
		if run3Report.Violation() != nil {
			t.Errorf("Run #3's (resumed) stream: %v", run3Report.Violation())
		}
		if seededReport.Violation() != nil {
			t.Errorf("the second, seeded harness's stream: %v", seededReport.Violation())
		}
	})
}
