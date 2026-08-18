// AG-16 — Phase 3+4: cumulative accumulator, the estimate/final
// protocol, and every non-happy run close (R-CST-004, R-CST-005,
// R-CST-006, R-RUN-003).
//
// Every scenario is in package agent_test (NFR-CST-001): the
// external-test posture proves every behavioral claim from outside
// the package, reading cost figures only through CostSession's paired
// accessors.
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// costSessionsIn returns every cost_session payload found in events,
// in stream order.
func costSessionsIn(events []agent.Event) []agent.CostSession {
	var out []agent.CostSession
	for _, ev := range events {
		if cs, ok := ev.CostSession(); ok {
			out = append(out, cs)
		}
	}
	return out
}

// sumUsageFromCostTurns sums every cost_turn observed in events into
// one ai.Usage, per figure independently — absence is the additive
// identity, presence is OR, mirroring costAccumulator.add's algebra.
// The test re-derives the sum from the observed events independently
// of the production accumulator, which is what proves the two agree
// rather than assuming they do (this capability's own
// Evidence-discipline rule: an equality against observed events,
// never a hand-computed literal).
func sumUsageFromCostTurns(events []agent.Event) ai.Usage {
	var input, output, cacheRead, cacheWrite, reasoning uint64
	var inputOk, outputOk, cacheReadOk, cacheWriteOk, reasoningOk bool
	for _, ct := range costTurnsIn(events) {
		if n, ok := ct.InputTokens(); ok {
			input += n
			inputOk = true
		}
		if n, ok := ct.OutputTokens(); ok {
			output += n
			outputOk = true
		}
		if n, ok := ct.CacheReadTokens(); ok {
			cacheRead += n
			cacheReadOk = true
		}
		if n, ok := ct.CacheWriteTokens(); ok {
			cacheWrite += n
			cacheWriteOk = true
		}
		if n, ok := ct.ReasoningTokens(); ok {
			reasoning += n
			reasoningOk = true
		}
	}
	u := ai.Usage{}
	if inputOk {
		u.Input = ai.Tokens(int64(input)) //nolint:gosec // test-scripted magnitudes stay well within int64
	}
	if outputOk {
		u.Output = ai.Tokens(int64(output)) //nolint:gosec // test-scripted magnitudes stay well within int64
	}
	if cacheReadOk {
		u.CacheRead = ai.Tokens(int64(cacheRead)) //nolint:gosec // test-scripted magnitudes stay well within int64
	}
	if cacheWriteOk {
		u.CacheWrite = ai.Tokens(int64(cacheWrite)) //nolint:gosec // test-scripted magnitudes stay well within int64
	}
	if reasoningOk {
		u.Reasoning = ai.Tokens(int64(reasoning)) //nolint:gosec // test-scripted magnitudes stay well within int64
	}
	return u
}

// assertCostSessionEqualsUsage asserts got's five paired accessors
// read back exactly what usage reports — magnitude and discriminator
// both.
func assertCostSessionEqualsUsage(t *testing.T, got agent.CostSession, usage ai.Usage) {
	t.Helper()

	wantInput, wantInputOk := usage.Input.Count()
	if n, ok := got.InputTokens(); uint64(wantInput) != n || wantInputOk != ok {
		t.Errorf("InputTokens() = (%d, %v), want (%d, %v)", n, ok, wantInput, wantInputOk)
	}
	wantOutput, wantOutputOk := usage.Output.Count()
	if n, ok := got.OutputTokens(); uint64(wantOutput) != n || wantOutputOk != ok {
		t.Errorf("OutputTokens() = (%d, %v), want (%d, %v)", n, ok, wantOutput, wantOutputOk)
	}
	wantCacheRead, wantCacheReadOk := usage.CacheRead.Count()
	if n, ok := got.CacheReadTokens(); uint64(wantCacheRead) != n || wantCacheReadOk != ok {
		t.Errorf("CacheReadTokens() = (%d, %v), want (%d, %v)", n, ok, wantCacheRead, wantCacheReadOk)
	}
	wantCacheWrite, wantCacheWriteOk := usage.CacheWrite.Count()
	if n, ok := got.CacheWriteTokens(); uint64(wantCacheWrite) != n || wantCacheWriteOk != ok {
		t.Errorf("CacheWriteTokens() = (%d, %v), want (%d, %v)", n, ok, wantCacheWrite, wantCacheWriteOk)
	}
	wantReasoning, wantReasoningOk := usage.Reasoning.Count()
	if n, ok := got.ReasoningTokens(); uint64(wantReasoning) != n || wantReasoningOk != ok {
		t.Errorf("ReasoningTokens() = (%d, %v), want (%d, %v)", n, ok, wantReasoning, wantReasoningOk)
	}
}

// assertCostEventsRideExistingBracketsAndLane — S-RUN-022. Walks
// events and asserts: exactly one run_start/run_end pair; every
// cost_turn lies strictly between its own turn's turn_start and
// turn_end; every cost_session lies inside the run bracket and
// outside every turn bracket; the sequence starts at 1 and increases
// by exactly 1 with no gap, repeat or restart across the whole run,
// cost events included; every event carries the run's own identity;
// and each cost_turn appears exactly once.
func assertCostEventsRideExistingBracketsAndLane(t *testing.T, events []agent.Event) {
	t.Helper()

	if len(events) == 0 {
		t.Fatal("no events to walk")
	}
	runID := events[0].Run()

	var runStarts, runEnds int
	openTurn := false
	var openTurnID agent.TurnID
	seenCostTurnInOpenTurn := false
	seenTurnIDs := map[agent.TurnID]bool{}

	for i, ev := range events {
		if ev.Run() != runID {
			t.Errorf("event[%d] (%v) run = %q, want %q", i, ev.Kind(), ev.Run(), runID)
		}
		if wantSeq := agent.Sequence(i + 1); ev.Sequence() != wantSeq { //nolint:gosec // i+1 fits within a real run's event count
			t.Errorf("event[%d] sequence = %v, want %v (1-based, contiguous, cost events included)", i, ev.Sequence(), wantSeq)
		}

		if _, ok := ev.RunStart(); ok {
			runStarts++
			continue
		}
		if _, ok := ev.RunEnd(); ok {
			runEnds++
			if openTurn {
				t.Errorf("event[%d] run_end observed while turn %q's bracket is still open", i, openTurnID)
			}
			continue
		}
		if _, ok := ev.TurnStart(); ok {
			if openTurn {
				t.Errorf("event[%d] turn_start observed while turn %q's bracket is still open", i, openTurnID)
			}
			openTurn = true
			openTurnID, _ = ev.Turn()
			seenCostTurnInOpenTurn = false
			continue
		}
		if _, ok := ev.TurnEnd(); ok {
			if !openTurn {
				t.Errorf("event[%d] turn_end observed with no open turn bracket", i)
			}
			openTurn = false
			continue
		}
		if _, ok := ev.CostTurn(); ok {
			if !openTurn {
				t.Errorf("event[%d] cost_turn observed outside any open turn bracket", i)
			}
			if seenCostTurnInOpenTurn {
				t.Errorf("event[%d] a second cost_turn observed inside turn %q's bracket", i, openTurnID)
			}
			seenCostTurnInOpenTurn = true
			turnID, _ := ev.Turn()
			if seenTurnIDs[turnID] {
				t.Errorf("cost_turn for turn %q observed more than once on the stream", turnID)
			}
			seenTurnIDs[turnID] = true
			continue
		}
		if _, ok := ev.CostSession(); ok {
			if openTurn {
				t.Errorf("event[%d] cost_session observed inside turn %q's still-open bracket, want outside every turn bracket", i, openTurnID)
			}
			if runStarts == 0 || runEnds > 0 {
				t.Errorf("event[%d] cost_session observed outside the run bracket", i)
			}
			continue
		}
	}

	if runStarts != 1 || runEnds != 1 {
		t.Errorf("run_start count = %d, run_end count = %d, want exactly 1 each", runStarts, runEnds)
	}
	if len(seenTurnIDs) == 0 {
		t.Error("no cost_turn observed on the stream")
	}
}

// TestHarness_CostSession_CumulativeEqualsEmittedCostTurns —
// S-CST-008, charter AG-16.1 scenario 2. A multi-turn run in which
// logical turn one fails retryably before any output (via the
// preStreamFailingProvider wrapper, the fixture constraint — a
// Script cannot express an arbitrary retryable pre-stream failure)
// and succeeds on its second attempt with a distinct scripted usage,
// continuing (PauseTurn) into a second logical turn with its own
// distinct usage. The run-terminal cost_session's figures equal,
// figure for figure, the sum over the cost_turn events observed on
// this same recorded stream — an equality against observed events,
// never a hand-computed literal.
func TestHarness_CostSession_CumulativeEqualsEmittedCostTurns(t *testing.T) {
	t.Parallel()

	failure, ferr := ai.PreStreamFailure(ai.FailureReport{
		Category:  ai.FailureCategoryUnavailable,
		Retryable: true,
	})
	if ferr != nil {
		t.Fatalf("ai.PreStreamFailure returned %v, want no failure", ferr)
	}

	usageTurnOneRetried := ai.Usage{
		Input:      ai.Tokens(700),
		Output:     ai.Tokens(140),
		CacheRead:  ai.Tokens(21),
		CacheWrite: ai.Tokens(7),
		Reasoning:  ai.Tokens(63),
	}
	usageTurnTwo := ai.Usage{
		Input:      ai.Tokens(19),
		Output:     ai.Tokens(23),
		CacheRead:  ai.Tokens(29),
		CacheWrite: ai.Tokens(31),
		Reasoning:  ai.Tokens(37),
	}

	turnOneScript := scriptTextResponseWithUsage(t, ai.FinishReasonPauseTurn, usageTurnOneRetried)
	turnTwoScript := scriptTextResponseWithUsage(t, ai.FinishReasonStop, usageTurnTwo)
	inner := agenttest.NewProvider(turnOneScript, turnTwoScript)
	provider := newPreStreamFailingProvider(1, failure, inner)

	h := agent.Harness{
		Provider:      provider,
		System:        "system prompt for cst-008",
		History:       agent.NewHistory(),
		RetryAttempts: 3,
		RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
	}

	sink := make(chan *agent.Event, 256)
	_, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v", finish, ai.FinishReasonStop)
	}

	events := drainSink(t, sink)
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(retry-bearing cost stream) = %v, want no violation", report.Violation())
	}

	// Exactly two cost_turn events on the stream: the retried attempt
	// that reached a Completion, and turn two — the failed first
	// attempt contributed no cost_turn at all.
	costTurns := costTurnsIn(events)
	if len(costTurns) != 2 {
		t.Fatalf("recorded %d cost_turn event(s), want 2 (retried attempt's success + turn two)", len(costTurns))
	}

	costSessions := costSessionsIn(events)
	if len(costSessions) == 0 {
		t.Fatal("recorded 0 cost_session events, want at least 1 (the run-terminal figure)")
	}
	terminal := costSessions[len(costSessions)-1]
	if got := terminal.Label(); got != agent.CostLabelFinal {
		t.Errorf("terminal cost_session label = %v, want Final", got)
	}

	wantSum := sumUsageFromCostTurns(events)
	assertCostSessionEqualsUsage(t, terminal, wantSum)

	// The equality is non-vacuous: the summed input must be non-zero,
	// proving sumUsageFromCostTurns is reading real per-attempt
	// figures rather than a coincidental all-absent record.
	if n, ok := wantSum.Input.Count(); !ok || n == 0 {
		t.Errorf("summed input = (%d, %v), want a positive reported value — the retried attempt's or turn two's usage was not actually observed", n, ok)
	}

	// Every cost_turn observed reads Final — extends S-CST-004 over
	// this retry-bearing stream (task 3.9).
	for i, ct := range costTurns {
		if got := ct.Label(); got != agent.CostLabelFinal {
			t.Errorf("cost_turn[%d] label = %v, want Final", i, got)
		}
	}

	// S-RUN-022 structural walk on this same retry-bearing stream
	// (task 3.10): brackets, lane and identity all hold with cost
	// events included.
	assertCostEventsRideExistingBracketsAndLane(t, events)
}

// costSessionFigureAtMost reports whether every one of got's reported
// figures is <= the corresponding figure in final — used to assert
// "no estimate exceeds the final one on any reported figure"
// (S-CST-009). An absent figure on got is trivially <= anything.
func costSessionFigureAtMost(got, final agent.CostSession) bool {
	check := func(gotN uint64, gotOk bool, finalN uint64) bool {
		return !gotOk || gotN <= finalN
	}
	gi, giOk := got.InputTokens()
	fi, _ := final.InputTokens()
	go2, go2Ok := got.OutputTokens()
	fo, _ := final.OutputTokens()
	gcr, gcrOk := got.CacheReadTokens()
	fcr, _ := final.CacheReadTokens()
	gcw, gcwOk := got.CacheWriteTokens()
	fcw, _ := final.CacheWriteTokens()
	gr, grOk := got.ReasoningTokens()
	fr, _ := final.ReasoningTokens()
	return check(gi, giOk, fi) && check(go2, go2Ok, fo) && check(gcr, gcrOk, fcr) &&
		check(gcw, gcwOk, fcw) && check(gr, grOk, fr)
}

// lastEventIsRunEndPrecededByCostSessionFinal asserts the stream's
// last event is run_end and the event immediately before it is a
// cost_session carrying Final — "immediately before the run-close",
// the position every non-happy close (Phase 4) also owes.
func lastEventIsRunEndPrecededByCostSessionFinal(t *testing.T, events []agent.Event) agent.CostSession {
	t.Helper()
	if len(events) < 2 {
		t.Fatalf("recorded %d event(s), want at least 2 (cost_session + run_end)", len(events))
	}
	last := events[len(events)-1]
	if _, ok := last.RunEnd(); !ok {
		t.Fatalf("last event kind = %v, want run_end", last.Kind())
	}
	beforeLast := events[len(events)-2]
	got, ok := beforeLast.CostSession()
	if !ok {
		t.Fatalf("event immediately before run_end is %v, want cost_session", beforeLast.Kind())
	}
	if got.Label() != agent.CostLabelFinal {
		t.Errorf("cost_session immediately before run_end has label %v, want Final", got.Label())
	}
	return got
}

// TestHarness_CostSession_EstimateThenFinal — S-CST-009, charter
// AG-16.1 scenario 3. A run the driver continues past its first
// logical turn (PauseTurn) carries at least one cost_session(Estimate)
// between turn brackets; the last cost_session on the stream carries
// Final, sits immediately before the run-close, equals the cumulative
// over the stream's cost_turn events, and no earlier estimate exceeds
// it on any figure.
func TestHarness_CostSession_EstimateThenFinal(t *testing.T) {
	t.Parallel()

	usageTurnOne := ai.Usage{Input: ai.Tokens(10), Output: ai.Tokens(5), Reasoning: ai.Tokens(2)}
	usageTurnTwo := ai.Usage{Input: ai.Tokens(30), Output: ai.Tokens(15), CacheRead: ai.Tokens(4)}

	turnOneScript := scriptTextResponseWithUsage(t, ai.FinishReasonPauseTurn, usageTurnOne)
	turnTwoScript := scriptTextResponseWithUsage(t, ai.FinishReasonStop, usageTurnTwo)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for cst-009", History: agent.NewHistory()}
	sink := make(chan *agent.Event, 256)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(estimate-then-final stream) = %v, want no violation", report.Violation())
	}

	costSessions := costSessionsIn(events)
	if len(costSessions) < 2 {
		t.Fatalf("recorded %d cost_session event(s), want at least 2 (>=1 Estimate + 1 Final)", len(costSessions))
	}

	final := lastEventIsRunEndPrecededByCostSessionFinal(t, events)
	assertCostSessionEqualsUsage(t, final, sumUsageFromCostTurns(events))

	var sawEstimate bool
	for _, cs := range costSessions[:len(costSessions)-1] {
		if got := cs.Label(); got != agent.CostLabelEstimate {
			t.Errorf("cost_session preceding the terminal one has label %v, want Estimate", got)
		} else {
			sawEstimate = true
		}
		if !costSessionFigureAtMost(cs, final) {
			t.Error("an estimate reports a figure exceeding the final cost_session on that figure")
		}
	}
	if !sawEstimate {
		t.Error("no cost_session carrying Estimate observed, want at least one between the two turn brackets")
	}

	// The Estimate sits strictly between the two turn brackets: after
	// turn one's turn_end, before turn two's turn_start.
	var turnEndIdx, turnStartTwoIdx, estimateIdx = -1, -1, -1
	turnStartsSeen := 0
	for i, ev := range events {
		if _, ok := ev.TurnStart(); ok {
			turnStartsSeen++
			if turnStartsSeen == 2 {
				turnStartTwoIdx = i
			}
		}
		if _, ok := ev.TurnEnd(); ok && turnEndIdx == -1 {
			turnEndIdx = i
		}
		if cs, ok := ev.CostSession(); ok && cs.Label() == agent.CostLabelEstimate && estimateIdx == -1 {
			estimateIdx = i
		}
	}
	if turnEndIdx == -1 || turnStartTwoIdx == -1 || estimateIdx == -1 {
		t.Fatalf("could not locate turn_end/second turn_start/estimate indices: %d/%d/%d", turnEndIdx, turnStartTwoIdx, estimateIdx)
	}
	if !(turnEndIdx < estimateIdx && estimateIdx < turnStartTwoIdx) {
		t.Errorf("estimate index %d is not strictly between turn_end (%d) and the second turn_start (%d)", estimateIdx, turnEndIdx, turnStartTwoIdx)
	}
}

// TestHarness_SingleTurnRun_FinalOnly — S-CST-010, the conditional
// half. A run that reaches its terminal on its first logical turn
// carries a sole cost_session, labelled Final, immediately before the
// run-close — no cost_session carrying Estimate appears anywhere: the
// conditional reading is satisfied by absence, not by a fabricated
// estimate.
func TestHarness_SingleTurnRun_FinalOnly(t *testing.T) {
	t.Parallel()

	usage := ai.Usage{Input: ai.Tokens(8), Output: ai.Tokens(6)}
	provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, usage))

	h := agent.Harness{Provider: provider, System: "system prompt for cst-010", History: agent.NewHistory()}
	sink := make(chan *agent.Event, 64)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(single-turn stream) = %v, want no violation", report.Violation())
	}

	costSessions := costSessionsIn(events)
	if len(costSessions) != 1 {
		t.Fatalf("recorded %d cost_session event(s), want exactly 1 (Final only)", len(costSessions))
	}
	if got := costSessions[0].Label(); got != agent.CostLabelFinal {
		t.Errorf("sole cost_session label = %v, want Final", got)
	}

	final := lastEventIsRunEndPrecededByCostSessionFinal(t, events)
	assertCostSessionEqualsUsage(t, final, sumUsageFromCostTurns(events))
}
