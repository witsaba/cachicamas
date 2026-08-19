// AG-18.4 — Phase 5: atomicity and recovery (R-CMP-010), package
// agent_test (NFR-CMP-001).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// runInterruptibleCompactionFixture drives a run whose strategy
// requests one compaction, gated then made to fail with a mid-stream
// *ai.Failure once released — the shared S-CMP-020/S-CMP-021 fixture.
// It returns the run's own result, the recorded stream, the history's
// entries captured WHILE the compaction call was held (before its own
// release — nothing else touches hist at that instant, since Run's own
// goroutine is itself blocked reading the held channel), and the
// harness's own provider (so a caller can inspect the SECOND turn's
// captured request).
func runInterruptibleCompactionFixture(t *testing.T, suffix string) (events []agent.Event, preEntries []agent.Entry, ownProvider *agenttest.Provider, runErr error) {
	t.Helper()

	toolName, callID := "read_cmp_"+suffix, "call-cmp-"+suffix
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	ownProvider = agenttest.NewProvider(
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)

	gate := agenttest.NewGate()
	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, Retryable: false}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want nil", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want nil", err)
	}
	failingProvider := agenttest.NewProvider(agenttest.Script{Steps: []agenttest.Step{agenttest.Hold(gate), agenttest.Emit(terminal)}})

	hist := agent.NewHistory()
	strategy := &compactAfterNStrategy{
		skip: 1, // skip the pre-turn-1 consultation (no mark exists yet).
		request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			// Captured HERE, synchronously inside the consultation that
			// triggers the compaction — the true pre-attempt state,
			// before runCompaction (and any scratch mutation inside
			// it) ever runs. Capturing this later (e.g. after
			// gate.Reached()) would be too late: a premature commit,
			// were one ever introduced, runs BEFORE the provider call
			// that the gate blocks inside, so a snapshot taken after
			// the gate is reached would already observe the mutated
			// state on both sides of the comparison and the bite this
			// fixture exists to prove would go undetected.
			preEntries = hist.Entries()
			return agent.CompactionRequest{Provider: failingProvider, Instruction: "summarize for S-CMP-020/021", Cut: len(prompt.Transcript)}
		},
	}
	h := &agent.Harness{Provider: ownProvider, System: "system prompt for S-CMP-020/021 " + suffix, Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy, History: hist}

	sink := make(chan *agent.Event, 256)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	<-gate.Reached()
	gate.Release()

	events = drainSink(t, sink)
	runErr = <-resultCh
	return events, preEntries, ownProvider, runErr
}

// TestCompaction_AtomicOrAbsent — S-CMP-020, charter AG-18.4 scenario.
// A run whose compaction provider is gated then made to fail: the
// history read-back after the attempt is byte-identical to the
// read-back captured before it; compaction_failed appears and
// compaction_finished does not; the next model turn's request carries
// the UNCOMPACTED transcript; and the run reaches its ordinary close.
func TestCompaction_AtomicOrAbsent(t *testing.T) {
	t.Parallel()

	events, preEntries, ownProvider, runErr := runInterruptibleCompactionFixture(t, "020")
	if runErr != nil {
		t.Fatalf("Run returned err = %v, want nil — a failed compaction must not end the run", runErr)
	}

	sawFailed, sawFinished := false, false
	for _, ev := range events {
		if ev.Kind() == agent.EventKindCompactionFailed {
			sawFailed = true
		}
		if ev.Kind() == agent.EventKindCompactionFinished {
			sawFinished = true
		}
	}
	if !sawFailed {
		t.Error("stream carries no compaction_failed event, want one")
	}
	if sawFinished {
		t.Error("stream carries compaction_finished, want none")
	}

	requests := ownProvider.Requests()
	if len(requests) != 2 {
		t.Fatalf("own provider recorded %d request(s), want 2 (turn 1, then turn 2 over the uncompacted transcript)", len(requests))
	}
	turn2Messages := requests[1].Messages()
	if len(turn2Messages) != len(preEntries) {
		t.Fatalf("turn 2's request carries %d message(s), want %d (the uncompacted transcript, unchanged by the failed compaction)", len(turn2Messages), len(preEntries))
	}
	for i := range preEntries {
		if !turn2Messages[i].Equal(preEntries[i].Message()) {
			t.Errorf("turn 2's request message[%d] differs from the pre-attempt transcript, want byte-identical", i)
		}
	}
}

// TestCompaction_WindDownNeverEntered — S-CMP-021. The failed
// compaction's own stream carries no orphan-synthesis-driven wind-down
// signature: the run's own close is the ordinary RunOutcomeCompleted,
// never a cancellation outcome, and the run performed a further
// (second) turn after the compaction failure.
func TestCompaction_WindDownNeverEntered(t *testing.T) {
	t.Parallel()

	events, _, ownProvider, runErr := runInterruptibleCompactionFixture(t, "021")
	if runErr != nil {
		t.Fatalf("Run returned err = %v, want nil", runErr)
	}

	var runEndOutcome agent.RunOutcome
	foundRunEnd := false
	turnStarts := map[agent.TurnID]bool{}
	for _, ev := range events {
		if ev.Kind() == agent.EventKindRunEnd {
			if re, ok := ev.RunEnd(); ok {
				runEndOutcome = re.Outcome()
				foundRunEnd = true
			}
		}
		if ev.Kind() == agent.EventKindTurnStart {
			turn, _ := ev.Turn()
			turnStarts[turn] = true
		}
	}
	if !foundRunEnd {
		t.Fatal("no run_end event found")
	}
	if runEndOutcome != agent.RunOutcomeCompleted {
		t.Errorf("run_end outcome = %v, want RunOutcomeCompleted (the ordinary close, never a cancellation outcome — the wind-down order was never entered)", runEndOutcome)
	}
	// Two model turns (turn 1 and the post-failure turn 2) plus the
	// compaction's own bracket: at least 3 distinct turn brackets.
	if len(turnStarts) < 3 {
		t.Errorf("observed %d distinct turn bracket(s), want at least 3 (turn 1, the compaction bracket, and a further turn after the failure)", len(turnStarts))
	}
	if got := len(ownProvider.Requests()); got != 2 {
		t.Errorf("own provider recorded %d request(s), want 2 — the run performed at least one further turn after the compaction failure", got)
	}
}
