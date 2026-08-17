// AG-13.2 — steering: queued at the boundary, arrival-ordered, never
// dropped (R-RUN-008). The production mechanism itself — steeringQueue's
// mutex-guarded FIFO, enqueue's typed post-close rejection, and the
// atomic takeOrClose terminal-decision step — already landed in
// harness.go during Phase 2 (S-RUN-002's typed-rejection test required
// it). This file exercises that same mechanism from AG-13.2's own
// charter angle: boundary timing proven by a request-recording provider,
// concurrent burst arrival order, and the final-turn atomic case.
package agent_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// heldTurnScript builds a scripted stream that blocks the producer at
// gate before emitting a short deterministic text turn ending in Stop —
// AG-13.2's shared fixture for a turn a test can hold open, steer
// during, then release.
func heldTurnScript(t *testing.T, gate *agenttest.Gate) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	delta, err := ai.NewTextDelta(1, "held turn text")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Hold(gate),
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// AG-13.2 — S-RUN-070. Charter AG-13.2 sc.1. Given a run whose turn-one
// script holds at an agenttest.Gate, when a user message is offered
// through Steer on reaching the gate and the gate is then released, then
// the in-flight turn's emitted event sequence is unchanged from the
// un-steered baseline, the steered message appears in the transcript
// between turn one's messages and turn two's, and the request the
// provider recorded for turn two contains that message — "before the
// next provider call" proven by the recorded request, not inferred from
// transcript order alone.
func TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	turnOneScript := heldTurnScript(t, gate)
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-070", History: hist}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	<-gate.Reached()

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "steered mid-turn"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(steered); err != nil {
		t.Fatalf("Steer returned %v, want nil", err)
	}

	gate.Release()

	events := drainSink(t, sink)
	if err := <-resultCh; err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}

	// Turn one's emitted kind sequence is the deterministic shape
	// heldTurnScript always produces — unaffected by the Hold or by the
	// concurrent Steer racing it (the un-steered baseline, by
	// construction: a Hold step carries no event of its own).
	var turnOneKinds []agent.EventKind
	inTurnOne := false
	for _, ev := range events {
		if ev.Kind() == agent.EventKindTurnStart && !inTurnOne {
			inTurnOne = true
		}
		if inTurnOne {
			turnOneKinds = append(turnOneKinds, ev.Kind())
		}
		if ev.Kind() == agent.EventKindTurnEnd && inTurnOne {
			break
		}
	}
	wantKinds := []agent.EventKind{
		agent.EventKindTurnStart,
		agent.EventKindMessageStartText,
		agent.EventKindMessageDeltaText,
		agent.EventKindMessageEndText,
		agent.EventKindTurnEnd,
	}
	if len(turnOneKinds) != len(wantKinds) {
		t.Fatalf("turn one emitted %d event(s), want %d: got kinds %v", len(turnOneKinds), len(wantKinds), turnOneKinds)
	}
	for i, want := range wantKinds {
		if turnOneKinds[i] != want {
			t.Errorf("turn one event[%d] kind = %v, want %v (unaffected by the Hold/concurrent Steer)", i, turnOneKinds[i], want)
		}
	}

	// The steered message sits in the transcript between turn one's
	// assistant message and turn two's.
	entries := hist.Entries()
	if len(entries) != 4 {
		t.Fatalf("history has %d entries, want 4 (prompt, turn-one assistant, steered, turn-two assistant)", len(entries))
	}
	if entries[1].Message().Role() != ai.RoleAssistant {
		t.Errorf("entries[1].Role() = %v, want RoleAssistant (turn one)", entries[1].Message().Role())
	}
	if entries[2].Message().Role() != ai.RoleUser || !entries[2].Message().Equal(steered) {
		t.Error("entries[2] is not the steered message")
	}
	if entries[3].Message().Role() != ai.RoleAssistant {
		t.Errorf("entries[3].Role() = %v, want RoleAssistant (turn two)", entries[3].Message().Role())
	}

	// Proven by the recorded request, not by transcript order alone:
	// turn two's request transcript contains the steered message.
	requests := provider.Requests()
	if len(requests) != 2 {
		t.Fatalf("provider recorded %d request(s), want 2", len(requests))
	}
	var found bool
	for _, m := range requests[1].Messages() {
		if m.Equal(steered) {
			found = true
		}
	}
	if !found {
		t.Error("turn two's recorded request does not contain the steered message — 'before the next provider call' is unproven")
	}
}

// heldToolCallScript builds a scripted stream that blocks the producer
// at gate before requesting one tool call and completing with
// FinishReasonToolCalls — a held turn that, once released, makes the
// run ITERATE (R-RUN-002's ToolCalls case) rather than take the atomic
// terminal-candidate path. Distinct from heldTurnScript so a steering
// test can choose which of Run's two boundary mechanisms — the ordinary
// per-iteration drain, or the atomic takeOrClose — it exercises.
func heldToolCallScript(t *testing.T, gate *agenttest.Gate, callID, toolName string, args []byte) agenttest.Script {
	t.Helper()

	start, err := ai.NewToolCallStart(1, callID, toolName)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	delta, err := ai.NewToolCallDelta(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	end, err := ai.NewToolCallEnd(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Hold(gate),
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// AG-13.2 — S-RUN-071. Charter AG-13.2 sc.2, first Then. Given a
// gate-held turn and N user messages offered through Steer from a second
// goroutine in a test-determined order, when the gate is released and
// the run continues, then all N messages appear in the transcript in
// that same arrival order, none is missing, and none is duplicated.
// Turn one is tool-calling (ends FinishReasonToolCalls, R-RUN-002's
// ITERATE case), so the burst is picked up by the ordinary per-boundary
// drain ahead of turn two's transcript, not by the atomic
// terminal-candidate path S-RUN-072 exercises — genuinely distinct
// coverage of Run's two message-boundary mechanisms.
func TestHarness_SteerBurst_ArrivalOrderZeroDrops(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_run_071", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_run_071": tool})

	gate := agenttest.NewGate()
	turnOneScript := heldToolCallScript(t, gate, "call-run-071", "read_run_071", []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-071", Turn: agent.TurnOptions{Tools: reg}, History: hist}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	<-gate.Reached()

	const n = 5
	wantTexts := make([]string, n)
	burstDone := make(chan struct{})
	go func() {
		defer close(burstDone)
		for i := 0; i < n; i++ {
			text := fmt.Sprintf("burst-%d", i)
			wantTexts[i] = text
			part, err := ai.NewText(text)
			if err != nil {
				t.Errorf("ai.NewText(%d): %v", i, err)
				return
			}
			msg, err := ai.NewMessage(ai.RoleUser, part)
			if err != nil {
				t.Errorf("ai.NewMessage(%d): %v", i, err)
				return
			}
			if serr := h.Steer(msg); serr != nil {
				t.Errorf("Steer(%d) returned %v, want nil", i, serr)
				return
			}
		}
	}()
	<-burstDone

	gate.Release()

	drainSink(t, sink)
	if err := <-resultCh; err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}

	entries := hist.Entries()
	// prompt, turn-one assistant(+call), turn-one tool-result, N steered,
	// turn-two assistant.
	const steerStart = 3
	wantLen := steerStart + n + 1
	if len(entries) != wantLen {
		t.Fatalf("history has %d entries, want %d", len(entries), wantLen)
	}
	if got := tool.Invocations(); got != 1 {
		t.Errorf("tool invocations = %d, want 1", got)
	}
	for i := 0; i < n; i++ {
		msg := entries[steerStart+i].Message()
		if msg.Role() != ai.RoleUser {
			t.Errorf("entries[%d].Role() = %v, want RoleUser", steerStart+i, msg.Role())
			continue
		}
		content := msg.Content()
		if len(content) != 1 {
			t.Fatalf("entries[%d] carries %d part(s), want 1", steerStart+i, len(content))
		}
		text, ok := content[0].Text()
		if !ok || text != wantTexts[i] {
			t.Errorf("entries[%d] text = %q (ok=%v), want %q — arrival order must be preserved, zero drops", steerStart+i, text, ok, wantTexts[i])
		}
	}
}

// AG-13.2 — S-RUN-072. Charter AG-13.2 sc.2, second Then. Given a
// gate-held final turn whose script would otherwise end the run, and a
// script available for one further turn, when a message is offered
// through Steer before the gate is released, then Steer returns nil, an
// additional turn bracket appears on the stream, the steered message is
// in the transcript, and the run terminates after that additional turn.
func TestHarness_FinalTurnSteer_YieldsNewTurn(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	turnOneScript := heldTurnScript(t, gate)
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-072", History: hist}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	<-gate.Reached()

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "final-turn steer"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(steered); err != nil {
		t.Fatalf("Steer returned %v, want nil (accepted before the run's terminal decision)", err)
	}

	gate.Release()

	events := drainSink(t, sink)
	if err := <-resultCh; err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}

	var turnStarts, turnEnds int
	for _, ev := range events {
		switch ev.Kind() {
		case agent.EventKindTurnStart:
			turnStarts++
		case agent.EventKindTurnEnd:
			turnEnds++
		}
	}
	if turnStarts != 2 || turnEnds != 2 {
		t.Errorf("turn_start=%d turn_end=%d, want exactly 2 each — an additional turn bracket must appear, not a dropped message", turnStarts, turnEnds)
	}

	entries := hist.Entries()
	var foundSteered bool
	for _, e := range entries {
		if e.Message().Equal(steered) {
			foundSteered = true
		}
	}
	if !foundSteered {
		t.Error("steered message not found in the transcript")
	}
}

// AG-13.2 — S-RUN-073. Given a run that has terminated, when Steer is
// called, then it returns the typed rejection of R-RUN-001 and the
// transcript is unchanged — the queue is closed, not merely empty.
// Called twice to prove the closed state persists rather than resetting
// after one rejection.
func TestHarness_SteerAfterTermination_QueueClosedTypedRejection(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-073", History: hist}

	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	lenBefore := hist.Len()

	for i := 0; i < 2; i++ {
		part, err := ai.NewText(fmt.Sprintf("too late %d", i))
		if err != nil {
			t.Fatalf("ai.NewText(%d): %v", i, err)
		}
		msg, err := ai.NewMessage(ai.RoleUser, part)
		if err != nil {
			t.Fatalf("ai.NewMessage(%d): %v", i, err)
		}
		serr := h.Steer(msg)
		if serr == nil {
			t.Fatalf("Steer call %d after termination returned nil, want a typed rejection", i)
		}
		if !errors.Is(serr, ai.ErrMisplaced) {
			t.Errorf("Steer call %d error rule class = %v, want errors.Is(err, ai.ErrMisplaced)", i, serr)
		}
	}

	if got := hist.Len(); got != lenBefore {
		t.Errorf("history.Len() after rejected Steer calls = %d, want unchanged %d", got, lenBefore)
	}
}
