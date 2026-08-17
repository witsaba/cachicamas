// AG-14 — Phase 2: the interrupt core (R-CAN-001, R-CAN-002, R-CAN-003
// groundwork, R-RUN-011 carve-out, R-HIS-007 back-annotation).
package agent_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AG-14 — S-CAN-001 (charter AG-14.1, scenario 1, first Then). Two
// independently non-vacuous halves, both required for the scenario's
// Given ("a provider stream and tools in flight"), which the harness's
// strictly sequential run loop can never make literally simultaneous
// within one turn (Schedule only runs once a turn's own provider
// stream has reached Completion) — so each half exercises its own
// production mechanism directly rather than forcing an artificial
// single flow:
//
//   - "provider cancelled mid-stream" seeds the transcript with a call
//     from a prior tool-calling turn left genuinely open (issued, no
//     result — R-HIS-007's own precondition), then holds THIS run's
//     provider stream and interrupts it: loop.go's closed-channel
//     cause check (R-CAN-001) fires, and wind-down's orphan synthesis
//     has a REAL open call to repair — a non-vacuous proof, not zero
//     entries dressed as coverage.
//   - "in-flight tool observes cancellation" dispatches a real
//     cancellation-observing tool call and interrupts while it is
//     blocked: propagation reaches the tool through the full harness
//     path (Interrupt -> runCtx -> Turn -> Schedule -> tool.Run), and
//     the run winds down through the iteration-boundary check instead
//     of loop.go's closed-channel branch — a genuinely different
//     production path from the first half.
func TestHarness_Interrupt_MidTurnEndsInterruptedWithOrphansSynthesized(t *testing.T) {
	t.Parallel()

	t.Run("provider cancelled mid-stream, a genuinely pre-existing open call is synthesized", func(t *testing.T) {
		t.Parallel()

		seedPart, err := ai.NewToolCall("call-can-001-seed", "read_can_001_seed", []byte(`{}`))
		if err != nil {
			t.Fatalf("ai.NewToolCall: %v", err)
		}
		seedMsg, err := ai.NewMessage(ai.RoleAssistant, seedPart)
		if err != nil {
			t.Fatalf("ai.NewMessage: %v", err)
		}
		seeded, err := agent.NewSeededHistory([]ai.Message{seedMsg})
		if err != nil {
			t.Fatalf("agent.NewSeededHistory: %v", err)
		}

		gate := agenttest.NewGate()
		provider := agenttest.NewProvider(heldTurnScript(t, gate))
		h := agent.Harness{Provider: provider, System: "system prompt for can-001-provider", History: seeded}

		sink := make(chan *agent.Event, 256)
		type runOutcome struct {
			msg    ai.Message
			finish ai.FinishReason
			err    error
		}
		resultCh := make(chan runOutcome, 1)
		go func() {
			msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
			resultCh <- runOutcome{msg, finish, err}
		}()

		<-gate.Reached()
		h.Interrupt()

		events := drainSink(t, sink)
		got := <-resultCh

		if got.err == nil {
			t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
		}
		if !errors.Is(got.err, agent.ErrInterrupted) {
			t.Errorf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", got.err)
		}
		if errors.Is(got.err, agent.ErrShutdown) {
			t.Error("Run error unexpectedly satisfies errors.Is against agent.ErrShutdown too — the two signals must stay distinguishable")
		}

		if len(events) == 0 {
			t.Fatal("zero events observed")
		}
		last := events[len(events)-1]
		runEnd, ok := last.RunEnd()
		if !ok {
			t.Fatalf("last event kind = %v, want run_end", last.Kind())
		}
		if runEnd.Outcome() != agent.RunOutcomeInterrupted {
			t.Errorf("run_end outcome = %v, want RunOutcomeInterrupted", runEnd.Outcome())
		}
		if runEnd.Outcome() == agent.RunOutcomeFailed {
			t.Error("run_end outcome = RunOutcomeFailed, want never Failed for an interrupted run")
		}
		if _, hasFailure := runEnd.Failure(); hasFailure {
			t.Error("run_end carries a *Failure, want none for an interrupted run")
		}

		if report := agent.CheckStream(events); report.Violation() != nil {
			t.Errorf("agent.CheckStream(emitted stream) = %v, want no violation (unmodified acceptance)", report.Violation())
		}

		entries := seeded.Entries()
		if len(entries) != 3 {
			t.Fatalf("history has %d entries, want 3 (seeded call, run's own prompt, synthesized result)", len(entries))
		}
		if entries[0].Origin() != agent.EntryOriginAppended {
			t.Errorf("entries[0].Origin() = %v, want EntryOriginAppended (committed before the signal)", entries[0].Origin())
		}
		if entries[1].Origin() != agent.EntryOriginAppended {
			t.Errorf("entries[1].Origin() = %v, want EntryOriginAppended (the run's own prompt, committed before the signal)", entries[1].Origin())
		}
		if entries[2].Origin() != agent.EntryOriginSynthesized {
			t.Errorf("entries[2].Origin() = %v, want EntryOriginSynthesized — a genuinely pre-existing open call must have been repaired, not zero entries dressed as coverage", entries[2].Origin())
		}
		resultPart, ok := entries[2].Message().Content()[0].ToolResult()
		if !ok {
			t.Fatal("synthesized entry carries no ToolResult part")
		}
		if resultPart.CallID() != "call-can-001-seed" {
			t.Errorf("synthesized ToolResult.CallID() = %q, want %q", resultPart.CallID(), "call-can-001-seed")
		}

		if err := seeded.CloseTurn(); err != nil {
			t.Errorf("history.CloseTurn() after wind-down = %v, want nil (no open call remains)", err)
		}
	})

	t.Run("in-flight tool call observes cancellation through the harness", func(t *testing.T) {
		t.Parallel()

		release := make(chan struct{}) // never closed: isolates ctx.Done() as the only path.
		observing, started, completed := CancellationObservingScriptedTool("read_can_001_tool", agent.EffectClassRead, release)
		reg := agent.NewMapRegistry(map[string]agent.Tool{"read_can_001_tool": observing})

		provider := agenttest.NewProvider(scriptToolCallResponse(t, "call-can-001-tool", "read_can_001_tool", []byte(`{}`)))
		hist := agent.NewHistory()
		h := agent.Harness{Provider: provider, System: "system prompt for can-001-tool", Turn: agent.TurnOptions{Tools: reg}, History: hist}

		sink := make(chan *agent.Event, 256)
		type runOutcome struct {
			msg    ai.Message
			finish ai.FinishReason
			err    error
		}
		resultCh := make(chan runOutcome, 1)
		go func() {
			msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
			resultCh <- runOutcome{msg, finish, err}
		}()

		<-started
		h.Interrupt()

		events := drainSink(t, sink)
		got := <-resultCh

		if got.err == nil {
			t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
		}
		if !errors.Is(got.err, agent.ErrInterrupted) {
			t.Errorf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", got.err)
		}
		if completed() {
			t.Error("completed() = true, want false — the in-flight tool must have observed ctx.Done(), not release")
		}

		if len(events) == 0 {
			t.Fatal("zero events observed")
		}
		last := events[len(events)-1]
		runEnd, ok := last.RunEnd()
		if !ok {
			t.Fatalf("last event kind = %v, want run_end", last.Kind())
		}
		if runEnd.Outcome() != agent.RunOutcomeInterrupted {
			t.Errorf("run_end outcome = %v, want RunOutcomeInterrupted", runEnd.Outcome())
		}
		if _, hasFailure := runEnd.Failure(); hasFailure {
			t.Error("run_end carries a *Failure, want none for an interrupted run")
		}

		if report := agent.CheckStream(events); report.Violation() != nil {
			t.Errorf("agent.CheckStream(emitted stream) = %v, want no violation (unmodified acceptance)", report.Violation())
		}

		// The turn's own call reached a genuine result (the scheduler's
		// existing disjoint-return-channel path, R-TLS-010) and is
		// committed normally — distinct from orphan synthesis, which
		// repairs only a call Schedule never returned a result for.
		entries := hist.Entries()
		if len(entries) == 0 {
			t.Fatal("history has zero entries, want at least the prompt and the turn's own commits")
		}
		for i, e := range entries {
			if e.Origin() != agent.EntryOriginAppended {
				t.Errorf("entries[%d].Origin() = %v, want EntryOriginAppended — this turn's own call reached a real result, it was never orphaned", i, e.Origin())
			}
		}

		// No second provider request: the iteration-boundary check
		// must wind down instead of starting a futile next turn
		// (R-RUN-011 carve-out, cross-referenced S-RUN-102).
		if got := len(provider.Requests()); got != 1 {
			t.Errorf("provider recorded %d request(s), want 1 — no request after the signal", got)
		}
	})
}

// AG-14 — S-CAN-002 (charter AG-14.1, scenario 1, second Then), jointly
// S-RUN-003. Given a harness value whose first run ended through an
// interrupt as in S-CAN-001, when a second Run is driven on that same
// value against a fresh script, then the second run emits its own
// complete run bracket, ends with the completed run outcome, returns a
// nil error, accepts a Steer call with a nil return during that second
// run whose message reaches the transcript before the next provider
// call, and CheckStream accepts the second run's stream unmodified —
// proving the steering queue reopened rather than staying closed from
// the first run.
func TestHarness_Interrupt_SameHarnessRunsNextPrompt(t *testing.T) {
	t.Parallel()

	type runOutcome struct {
		msg    ai.Message
		finish ai.FinishReason
		err    error
	}

	firstGate := agenttest.NewGate()
	secondGate := agenttest.NewGate()
	provider := agenttest.NewProvider(
		heldTurnScript(t, firstGate),               // run 1, turn 1 — interrupted mid-stream.
		heldTurnScript(t, secondGate),              // run 2, turn 1 — held so Steer is proven pre-boundary.
		scriptTextResponse(t, ai.FinishReasonStop), // run 2, turn 2 — triggered by the steered message.
	)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for can-002", History: hist}

	// First run: interrupted, exactly as S-CAN-001's own shape.
	firstSink := make(chan *agent.Event, 256)
	firstResultCh := make(chan runOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), firstSink)
		firstResultCh <- runOutcome{msg, finish, err}
	}()
	<-firstGate.Reached()
	h.Interrupt()

	firstEvents := drainSink(t, firstSink)
	firstGot := <-firstResultCh
	if !errors.Is(firstGot.err, agent.ErrInterrupted) {
		t.Fatalf("first Run error = %v, want errors.Is(err, agent.ErrInterrupted)", firstGot.err)
	}
	if len(firstEvents) == 0 {
		t.Fatal("first run: zero events observed")
	}
	firstLast := firstEvents[len(firstEvents)-1]
	if firstRunEnd, ok := firstLast.RunEnd(); !ok || firstRunEnd.Outcome() != agent.RunOutcomeInterrupted {
		t.Fatalf("first run did not end interrupted (last event kind = %v)", firstLast.Kind())
	}

	// Second run: same harness value h, fresh script, held mid-stream
	// so Steer's "before the next provider call" claim is proven by
	// the recorded request rather than inferred from ordering alone
	// (S-RUN-070's own technique).
	secondSink := make(chan *agent.Event, 256)
	secondResultCh := make(chan runOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), secondSink)
		secondResultCh <- runOutcome{msg, finish, err}
	}()
	<-secondGate.Reached()

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "steered in second run"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if serr := h.Steer(steered); serr != nil {
		t.Fatalf("Steer during the second run returned %v, want nil — the steering queue must reopen at Run entry", serr)
	}
	secondGate.Release()

	secondEvents := drainSink(t, secondSink)
	secondGot := <-secondResultCh

	if secondGot.err != nil {
		t.Fatalf("second Run returned err = %v, want nil — the same harness value must accept a new prompt after an interrupted run (R-CAN-002)", secondGot.err)
	}
	if secondGot.finish != ai.FinishReasonStop {
		t.Errorf("second Run finish = %v, want %v", secondGot.finish, ai.FinishReasonStop)
	}

	if len(secondEvents) == 0 {
		t.Fatal("second run: zero events observed")
	}
	if secondEvents[0].Kind() != agent.EventKindRunStart {
		t.Errorf("second run first event kind = %v, want run_start (its own complete run bracket)", secondEvents[0].Kind())
	}
	secondLast := secondEvents[len(secondEvents)-1]
	secondRunEnd, ok := secondLast.RunEnd()
	if !ok {
		t.Fatalf("second run last event kind = %v, want run_end", secondLast.Kind())
	}
	if secondRunEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("second run_end outcome = %v, want RunOutcomeCompleted", secondRunEnd.Outcome())
	}

	// The steered message triggered a new turn (S-RUN-072's own
	// mechanism, exercised here as evidence the queue reopened, not
	// re-tested as its own property): two turn brackets, not one.
	var turnStarts, turnEnds int
	for _, ev := range secondEvents {
		switch ev.Kind() {
		case agent.EventKindTurnStart:
			turnStarts++
		case agent.EventKindTurnEnd:
			turnEnds++
		}
	}
	if turnStarts != 2 || turnEnds != 2 {
		t.Errorf("second run turn_start=%d turn_end=%d, want 2 each — the steered message must yield a new turn, not be dropped", turnStarts, turnEnds)
	}

	if report := agent.CheckStream(secondEvents); report.Violation() != nil {
		t.Errorf("agent.CheckStream(second run's stream) = %v, want no violation (unmodified acceptance)", report.Violation())
	}

	// "Before the next provider call", proven by the recorded
	// request, not inferred from transcript order alone: requests[0]
	// is run 1's turn 1, requests[1] is run 2's turn 1 (held),
	// requests[2] is run 2's turn 2 — the one the steered message must
	// appear in.
	requests := provider.Requests()
	if len(requests) != 3 {
		t.Fatalf("provider recorded %d request(s), want 3", len(requests))
	}
	var foundInRequest bool
	for _, m := range requests[2].Messages() {
		if m.Equal(steered) {
			foundInRequest = true
		}
	}
	if !foundInRequest {
		t.Error("second run's turn-two recorded request does not contain the steered message — 'before the next provider call' is unproven")
	}

	var foundInTranscript bool
	for _, e := range hist.Entries() {
		if e.Message().Equal(steered) {
			foundInTranscript = true
		}
	}
	if !foundInTranscript {
		t.Error("steered message not found in the transcript — zero-drop guarantee violated")
	}
}
