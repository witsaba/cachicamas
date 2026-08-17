// AG-14 — Phase 2: the interrupt core (R-CAN-001, R-CAN-002, R-CAN-003
// groundwork, R-RUN-011 carve-out, R-HIS-007 back-annotation). Widened
// at Phase 5 (R-CAN-003 itself: interrupt during a permission
// suspension) and Phase 6 (R-CAN-004: a second signal is a no-op) —
// both land in this file per design.md's Testing Strategy table (test
// 3 and test 4 are both "same file" as test 1).
package agent_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

		// AG-14 remediation (verify-report.md MAJOR-3): R-CAN-002's
		// turn-bracket clause is normative, not merely a well-formed
		// bracket CheckStream already pins — the aborted turn's own
		// turn_end (loop.go:442-449) MUST carry TurnOutcomeAborted and a
		// *Failure of category ai.FailureCategoryCancellation. A
		// regression to a different outcome or category (e.g.
		// ai.FailureCategoryUnavailable) would otherwise stay green.
		var turnEnd agent.TurnEnd
		var turnEndFound bool
		for _, ev := range events {
			if payload, ok := ev.TurnEnd(); ok {
				turnEnd, turnEndFound = payload, true
			}
		}
		if !turnEndFound {
			t.Fatal("no turn_end event observed, want the aborted turn's own turn_end")
		}
		if turnEnd.Outcome() != agent.TurnOutcomeAborted {
			t.Errorf("turn_end outcome = %v, want agent.TurnOutcomeAborted", turnEnd.Outcome())
		}
		turnFailure, hasTurnFailure := turnEnd.Failure()
		if !hasTurnFailure {
			t.Fatal("turn_end carries no *Failure, want one for an aborted turn")
		}
		if turnFailure.Category() != ai.FailureCategoryCancellation {
			t.Errorf("turn_end failure.Category() = %v, want ai.FailureCategoryCancellation", turnFailure.Category())
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

// readUntilDecisionRequired reads events off sink, in arrival order,
// until a permission_decision_required event is observed (inclusive),
// returning every event read so far and the decision's own payload.
// AG-14 Phase 5 (R-CAN-003): the stream itself is the synchronization
// point for firing a signal mid-suspension — no agenttest.Gate, no
// wall clock beyond this helper's own defensive deadline (which never
// gates the signal itself, only guards a genuinely stuck test).
// Mirrors permission_protocol_test.go's own drainUntilDecisionRequired,
// adapted to the harness's already-stamped agent.Event values: no
// separate goroutine is needed here because sink is buffered and a
// parked call emits nothing further until a signal fires, so a direct
// sequential read cannot deadlock the caller.
func readUntilDecisionRequired(t *testing.T, sink <-chan *agent.Event) ([]agent.Event, agent.PermissionDecisionRequired) {
	t.Helper()
	var events []agent.Event
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev, ok := <-sink:
			if !ok {
				t.Fatal("readUntilDecisionRequired: sink closed before a permission_decision_required event was observed")
				return nil, agent.PermissionDecisionRequired{}
			}
			events = append(events, *ev)
			if req, ok := ev.PermissionDecisionRequired(); ok {
				return events, req
			}
		case <-deadline:
			t.Fatal("readUntilDecisionRequired: no permission_decision_required event observed within 2s")
			return nil, agent.PermissionDecisionRequired{}
		}
	}
}

// AG-14 — Phase 5: interrupt during a permission suspension aborts it
// typed (R-CAN-003, S-CAN-003; jointly S-APP-017's interrupt half,
// agent-permission-protocol delta). Given a run whose permission
// policy defers a call, and a consumer that has read the
// decision-required event off the run stream (the stream is the
// synchronization; no wall clock), when the interrupt signal fires,
// then the parked call's abort — read back through the emitted
// tool_end_execution_failure event, since the harness surface itself
// never returns []Result directly — carries a typed *Failure reporting
// category Cancellation whose unwrap chain satisfies errors.Is against
// the interrupt sentinel; a subsequent wake for that call identity
// returns ErrStrayDecision; the transcript reads back with no open
// call after the wind-down; and CheckStream accepts the stream
// unmodified.
func TestHarness_Interrupt_DuringSuspensionAbortsTyped(t *testing.T) {
	t.Parallel()

	deferring := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
		},
	}
	// Never released: the call must never reach Run — the gate aborts
	// it before executeCall ever calls tool.Run.
	tool := BlockingScriptedTool("read_can_003", agent.EffectClassRead, make(chan struct{}))
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_can_003": tool})

	provider := agenttest.NewProvider(scriptToolCallResponse(t, "call-can-003", "read_can_003", []byte(`{}`)))
	hist := agent.NewHistory()
	sched := &agent.Scheduler{}
	h := agent.Harness{
		Provider:  provider,
		System:    "system prompt for can-003-interrupt",
		Turn:      agent.TurnOptions{Tools: reg, PermissionPolicy: deferring},
		History:   hist,
		Scheduler: sched,
	}

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

	events, decision := readUntilDecisionRequired(t, sink)
	if decision.CallID() != "call-can-003" {
		t.Fatalf("permission_decision_required.CallID() = %q, want %q", decision.CallID(), "call-can-003")
	}
	h.Interrupt()

	events = append(events, drainSink(t, sink)...)
	got := <-resultCh

	if got.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
	}
	if !errors.Is(got.err, agent.ErrInterrupted) {
		t.Errorf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", got.err)
	}

	var failureEv agent.ToolEndExecutionFailure
	var found bool
	for _, ev := range events {
		if tef, ok := ev.ToolEndExecutionFailure(); ok && tef.CallID() == "call-can-003" {
			failureEv, found = tef, true
		}
	}
	if !found {
		t.Fatal("no tool_end_execution_failure observed for call-can-003")
	}
	failure, hasFailure := failureEv.Failure()
	if !hasFailure {
		t.Fatal("tool_end_execution_failure carries no *Failure")
	}
	if failure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("failure.Category() = %v, want ai.FailureCategoryCancellation", failure.Category())
	}
	if !errors.Is(failure.Unwrap(), agent.ErrInterrupted) {
		t.Error("failure does not unwrap to errors.Is-match agent.ErrInterrupted")
	}

	if err := sched.WakeParked("call-can-003"); !errors.Is(err, agent.ErrStrayDecision) {
		t.Errorf("WakeParked(%q) after wind-down = %v, want errors.Is to match agent.ErrStrayDecision", "call-can-003", err)
	}

	if err := hist.CloseTurn(); err != nil {
		t.Errorf("history.CloseTurn() after wind-down = %v, want nil (no open call remains)", err)
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(emitted stream) = %v, want no violation (unmodified acceptance)", report.Violation())
	}
}

// AG-14 — Phase 6: a second signal changes nothing and panics nothing
// (R-CAN-004, S-CAN-004, charter AG-14.1 scenario 3). Given a run
// already winding down from an interrupt, when a second interrupt
// fires from a separate goroutine concurrent with the first's
// wind-down and the suite runs under -race, then the test process
// reports no panic and no data race, the run-end outcome and the
// returned error are the same values S-CAN-001's Arm A observed, and
// the emitted event sequence is accepted by CheckStream unmodified.
// Both signal calls are launched from their own goroutines,
// synchronized only by a WaitGroup — never sequenced relative to each
// other — so either can race the other AND the Run goroutine's own
// wind-down; the composition this proves is 2.2's signalMu-guarded
// {cancelRun, shutdown} state plus Go's own documented no-op semantics
// for an already-cancelled context's cancel-cause function. No new
// production code: confirmed genuinely, not assumed, by running under
// -race -count=10 (see apply-progress.md).
func TestHarness_Interrupt_SecondInterruptIsNoOp(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	provider := agenttest.NewProvider(heldTurnScript(t, gate))
	h := agent.Harness{Provider: provider, System: "system prompt for can-004"}

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

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); h.Interrupt() }()
	go func() { defer wg.Done(); h.Interrupt() }()
	wg.Wait()

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
}

// AG-14 remediation (verify-report.md MINOR-3). harness.go's Run
// registers `defer close(sink)` BEFORE the `cancelRun`-clearing defer,
// so LIFO execution closes the sink FIRST and clears `h.cancelRun`
// SECOND. A stream-only consumer — exactly R-CAN-005's own shape, and
// S-CAN-002's serial-reuse pattern — that starts run #2 the instant it
// observes run #1's sink close can therefore have run #2 register its
// own cancelRun BEFORE run #1's still-pending defer nulls it back out,
// silently making a subsequent Interrupt() on run #2 a no-op. Both
// writes are signalMu-guarded, so -race reports nothing here: this is a
// scheduling-dependent ordering defect, not a data race.
//
// On an otherwise-idle machine the window is vanishingly narrow — run
// #1's remaining cleanup (a mutex lock, a nil-assignment, an unlock, an
// already-cancelled cancel() no-op) is structurally cheaper than run
// #2's path back into it (wake, get scheduled, run Run()'s own
// preamble). Reliably widening the window, with no sleep anywhere,
// takes two techniques together, both verified empirically
// (apply-progress.md records the calibration): (1) GOMAXPROCS
// temporarily lowered so run #1's goroutine has real scheduling
// competition instead of an idle core it can run straight through, and
// (2) parking many extra "noise" receivers on run #1's own sink AHEAD
// of a dedicated, front-of-queue real observer — sink1 is provably
// empty until Interrupt() fires below (heldTurnScript's Hold is the
// producer's first step), so every one of these calls is guaranteed to
// park rather than race a real event — which widens close(sink)'s own
// synchronous wake loop (run entirely on run #1's goroutine, before it
// can reach its next deferred call). Under the buggy order this
// combination reproduces a real, non-vacuous failure within the
// attempts budget below; against the fix the identical construction
// produced zero hits across 900+ attempts.
//
// Deliberately NOT t.Parallel(): it mutates the process-wide
// GOMAXPROCS for its own duration and must not run concurrently with
// unrelated tests.
func TestHarness_Interrupt_SinkCloseObservationDoesNotLoseRun2CancelRegistration(t *testing.T) {
	prevProcs := runtime.GOMAXPROCS(2)
	defer runtime.GOMAXPROCS(prevProcs)

	const attempts = 2000
	const noiseWaiters = 2000

	for attempt := 0; attempt < attempts; attempt++ {
		gate1 := agenttest.NewGate()
		gate2 := agenttest.NewGate()
		provider := agenttest.NewProvider(heldTurnScript(t, gate1), heldTurnScript(t, gate2))
		h := agent.Harness{Provider: provider, System: "system prompt for can-minor-3"}

		sink1 := make(chan *agent.Event, 256)
		resultCh1 := make(chan error, 1)
		go func() {
			_, _, err := h.Run(contextBackground(), firstMessage(t), sink1)
			resultCh1 <- err
		}()

		// The real observer: registered as sink1's FIRST receiver, so it
		// sits at the front of sink1's FIFO recvq and is therefore the
		// first goroutine closechan's wake loop marks runnable — it
		// starts run #2 the instant it observes run #1's sink close,
		// never reading the events for their content.
		sink2 := make(chan *agent.Event, 256)
		resultCh2 := make(chan error, 1)
		started2 := make(chan struct{})
		go func() {
			for {
				_, ok := <-sink1
				if !ok {
					break
				}
			}
			close(started2)
			_, _, err := h.Run(contextBackground(), firstMessage(t), sink2)
			resultCh2 <- err
		}()

		// Noise receivers, parked AFTER the real observer above. sink1
		// is provably empty until Interrupt() fires below, so every one
		// of these calls is guaranteed to park in recvq rather than race
		// a real event.
		var parked atomic.Int64
		for n := 0; n < noiseWaiters; n++ {
			go func() {
				parked.Add(1)
				<-sink1
			}()
		}
		for parked.Load() < noiseWaiters {
			runtime.Gosched()
		}

		<-gate1.Reached()
		h.Interrupt()

		<-started2
		<-gate2.Reached()
		h.Interrupt()

		select {
		case err2 := <-resultCh2:
			if !errors.Is(err2, agent.ErrInterrupted) {
				t.Fatalf("attempt %d: run #2 error = %v, want errors.Is(err, agent.ErrInterrupted) — run #1's still-pending cancelRun-clearing defer clobbered run #2's registration", attempt, err2)
			}
		case <-time.After(500 * time.Millisecond):
			gate2.Release()
			<-resultCh2
			t.Fatalf("attempt %d: run #2 did not end within 500ms of Interrupt() — the interrupt was silently swallowed, run #1's still-pending cancelRun-clearing defer clobbered run #2's registration", attempt)
		}

		<-resultCh1
		drainSink(t, sink2)
	}
}
