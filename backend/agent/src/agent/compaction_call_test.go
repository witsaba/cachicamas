// AG-18.1 — Phase 3: the compaction call and its spend
// (R-CMP-002..003, S-CST-015, S-CST-016), package agent_test
// (NFR-CMP-001).
package agent_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// compactAfterNStrategy is a test-local agent.ContextStrategy that
// returns the zero verdict on its first `skip` consultations, then
// requests exactly one compaction — built by calling `request` with
// that consultation's own prompt, so a test can size Cut from the
// prompt it actually receives rather than guessing it — and the zero
// verdict on every consultation after that.
type compactAfterNStrategy struct {
	mu      sync.Mutex
	skip    int
	seen    int
	request func(prompt agent.ContextPrompt) agent.CompactionRequest
}

func (s *compactAfterNStrategy) Resolve(_ context.Context, prompt agent.ContextPrompt) agent.ContextVerdict {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen++
	if s.seen <= s.skip || s.request == nil {
		return agent.ContextVerdict{}
	}
	req := s.request(prompt)
	s.request = nil
	return agent.ContextVerdict{Compaction: &req}
}

var _ agent.ContextStrategy = (*compactAfterNStrategy)(nil)

// markedHarnessForCompaction drives one completed text-only turn
// through a fresh *agent.Harness/*agent.History pair so hist carries
// exactly one recorded turn mark (at count 2: the prompt and the
// turn's own assistant message), then returns the harness, its
// history, and its own provider — a distinct instance from whatever
// provider a test later injects into a CompactionRequest.
func markedHarnessForCompaction(t *testing.T, suffix string) (*agent.Harness, *agent.History, *agenttest.Provider) {
	t.Helper()
	hist := agent.NewHistory()
	ownProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h := &agent.Harness{Provider: ownProvider, System: "system prompt for compaction call test " + suffix, History: hist}
	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	drainSink(t, sink)
	if got := hist.Len(); got != 2 {
		t.Fatalf("fixture history carries %d entries, want 2 (prompt, assistant text)", got)
	}
	return h, hist, ownProvider
}

// TestCompaction_OwnCall_DistinctProvider — S-CMP-001, charter AG-18.1
// scenario 1, the own-call half. The compaction call goes to the
// injected provider (distinct instance B), never the harness's own
// (instance A): B's request count is exactly one and A's is unchanged.
func TestCompaction_OwnCall_DistinctProvider(t *testing.T) {
	t.Parallel()

	h, hist, ownProvider := markedHarnessForCompaction(t, "001")
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize turn one", Cut: hist.Len()}

	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err != nil {
		t.Fatalf("Compact returned %v, want nil", err)
	}
	drainSink(t, sink)

	if got := len(compactionProvider.Requests()); got != 1 {
		t.Errorf("injected compaction provider recorded %d request(s), want exactly 1", got)
	}
	if got := len(ownProvider.Requests()); got != 1 {
		t.Errorf("harness's own provider recorded %d request(s), want unchanged 1 — the compaction call must not touch it", got)
	}
}

// TestCompaction_InjectionOnly — S-CMP-002, charter AG-18.1 scenario 2.
// The captured request carries exactly the injected instruction as its
// system instruction, and its messages are exactly the compacted
// prefix, element-equal — no Layer 2-authored text anywhere.
func TestCompaction_InjectionOnly(t *testing.T) {
	t.Parallel()

	h, hist, _ := markedHarnessForCompaction(t, "002")
	preEntries := hist.Entries() // captured BEFORE Compact mutates hist.
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	const instruction = "injected-only instruction for S-CMP-002, never Layer-2-authored"
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: instruction, Cut: hist.Len()}

	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err != nil {
		t.Fatalf("Compact returned %v, want nil", err)
	}
	drainSink(t, sink)

	requests := compactionProvider.Requests()
	if len(requests) != 1 {
		t.Fatalf("compaction provider recorded %d request(s), want 1", len(requests))
	}
	sys, hasSys := requests[0].SystemInstruction()
	if !hasSys {
		t.Fatal("captured request carries no system instruction, want the injected one")
	}
	segments := sys.Segments()
	if len(segments) != 1 || segments[0].Text() != instruction {
		t.Errorf("captured system instruction segments = %+v, want exactly one segment with text %q", segments, instruction)
	}

	gotMessages := requests[0].Messages()
	if len(gotMessages) != len(preEntries) {
		t.Fatalf("captured request carries %d message(s), want %d (the compacted prefix)", len(gotMessages), len(preEntries))
	}
	for i := range preEntries {
		if !gotMessages[i].Equal(preEntries[i].Message()) {
			t.Errorf("captured message[%d] != the compacted prefix's own entry — Layer 2 must inject no runtime-authored text into any message part", i)
		}
	}
}

// TestCompaction_CancellationAbortsWithoutEndingRun — S-CMP-003,
// charter AG-18.1 scenario 1, the cancellation half. A run whose
// strategy requests a compaction against a gate-held provider: firing
// the run's own Interrupt while the call is outstanding closes the
// compaction bracket aborted with a typed failure, the run itself
// returns the interrupt sentinel, no compaction_finished appears
// anywhere on the stream, and the harness value survives and remains
// usable for a later run.
func TestCompaction_CancellationAbortsWithoutEndingRun(t *testing.T) {
	t.Parallel()

	const toolName, callID = "read_cmp_003", "call-cmp-003"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	ownProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	gate := agenttest.NewGate()
	heldProvider := agenttest.NewProvider(agenttest.Script{Steps: []agenttest.Step{agenttest.Hold(gate)}})

	strategy := &compactAfterNStrategy{
		skip: 1, // skip the pre-turn-1 consultation (no mark exists yet).
		request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{Provider: heldProvider, Instruction: "summarize for S-CMP-003", Cut: len(prompt.Transcript)}
		},
	}

	hist := agent.NewHistory()
	h := &agent.Harness{Provider: ownProvider, System: "system prompt for S-CMP-003", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy, History: hist}

	sink := make(chan *agent.Event, 256)
	type runResult struct {
		err error
	}
	resultCh := make(chan runResult, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- runResult{err: err}
	}()

	<-gate.Reached()
	h.Interrupt()

	events := drainSink(t, sink)
	got := <-resultCh

	if got.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
	}
	if !errors.Is(got.err, agent.ErrInterrupted) {
		t.Errorf("errors.Is(err, agent.ErrInterrupted) = false, want true (err = %v)", got.err)
	}

	sawCompactionFailed, sawCompactionFinished, sawAbortedTurnEnd := false, false, false
	for _, ev := range events {
		switch ev.Kind() {
		case agent.EventKindCompactionFinished:
			sawCompactionFinished = true
		case agent.EventKindCompactionFailed:
			sawCompactionFailed = true
		case agent.EventKindTurnEnd:
			if te, ok := ev.TurnEnd(); ok && te.Outcome() == agent.TurnOutcomeAborted {
				if _, hasFailure := te.Failure(); hasFailure {
					sawAbortedTurnEnd = true
				}
			}
		}
	}
	if sawCompactionFinished {
		t.Error("stream carries a compaction_finished event, want none anywhere")
	}
	if !sawCompactionFailed {
		t.Error("stream carries no compaction_failed event, want one")
	}
	if !sawAbortedTurnEnd {
		t.Error("stream carries no aborted turn_end carrying a typed failure, want one (the compaction bracket's own close)")
	}
	if len(compactionProviderRequests(heldProvider)) != 1 {
		t.Errorf("held compaction provider recorded %d request(s), want exactly 1", len(compactionProviderRequests(heldProvider)))
	}

	// The harness value survives and remains usable for a later run.
	freshSink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), freshSink); err != nil {
		t.Fatalf("a later Run on the same harness value returned %v, want nil", err)
	}
	drainSink(t, freshSink)
}

// compactionProviderRequests is a tiny indirection so
// TestCompaction_CancellationAbortsWithoutEndingRun reads its own
// assertion without repeating agenttest.Provider's own accessor name
// twice inline.
func compactionProviderRequests(p *agenttest.Provider) []ai.Request { return p.Requests() }

// TestCompaction_EmptyInstructionFailsTyped — S-CMP-005. A compaction
// request carrying an empty instruction fails typed, compaction_failed
// is emitted, no provider request is issued, and the transcript
// read-back is unchanged.
func TestCompaction_EmptyInstructionFailsTyped(t *testing.T) {
	t.Parallel()

	h, hist, _ := markedHarnessForCompaction(t, "005")
	pre := hist.Entries()
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "", Cut: hist.Len()}

	sink := make(chan *agent.Event, 64)
	err := h.Compact(contextBackground(), req, sink)
	events := drainSink(t, sink)

	if err == nil {
		t.Fatal("Compact returned nil, want a typed failure")
	}
	if got := len(compactionProvider.Requests()); got != 0 {
		t.Errorf("compaction provider recorded %d request(s), want 0 — no provider request may be issued for an empty instruction", got)
	}
	sawFailed := false
	for _, ev := range events {
		if ev.Kind() == agent.EventKindCompactionFailed {
			sawFailed = true
		}
		if ev.Kind() == agent.EventKindCompactionFinished {
			t.Error("stream carries compaction_finished, want none")
		}
	}
	if !sawFailed {
		t.Error("stream carries no compaction_failed event, want one")
	}
	post := hist.Entries()
	if len(post) != len(pre) {
		t.Fatalf("entry count after a rejected compaction = %d, want unchanged %d", len(post), len(pre))
	}
	for i := range pre {
		if !pre[i].Message().Equal(post[i].Message()) || pre[i].ID() != post[i].ID() || pre[i].Origin() != post[i].Origin() {
			t.Errorf("entry %d changed after a rejected compaction, want byte-identical", i)
		}
	}
}

// TestCompaction_SpendFoldedIntoCumulative — S-CMP-004, S-CST-015,
// S-CST-016, charter AG-18.1 scenario 1, the spend half. A cost_turn
// labelled Final appears inside the compaction bracket before its
// turn_end, carrying the compaction completion's own usage
// figure-for-figure — folded into the run's cumulative by an explicit
// accumulation at the emission site (bite S-CMP-032 proves this below).
func TestCompaction_SpendFoldedIntoCumulative(t *testing.T) {
	t.Parallel()

	h, hist, _ := markedHarnessForCompaction(t, "004")
	usage := ai.Usage{Input: ai.Tokens(111), Output: ai.Tokens(222), CacheRead: ai.Tokens(3), CacheWrite: ai.Tokens(4), Reasoning: ai.Tokens(5)}
	compactionProvider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, usage))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for S-CMP-004", Cut: hist.Len()}

	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err != nil {
		t.Fatalf("Compact returned %v, want nil", err)
	}
	events := drainSink(t, sink)

	costTurnIdx, turnEndIdx := -1, -1
	var sessionInput uint64
	sawSession := false
	for i, ev := range events {
		switch ev.Kind() {
		case agent.EventKindCostTurn:
			ct, ok := ev.CostTurn()
			if !ok {
				continue
			}
			costTurnIdx = i
			if ct.Label() != agent.CostLabelFinal {
				t.Errorf("cost_turn label = %v, want Final", ct.Label())
			}
			in, _ := ct.InputTokens()
			out, _ := ct.OutputTokens()
			cr, _ := ct.CacheReadTokens()
			cw, _ := ct.CacheWriteTokens()
			rs, _ := ct.ReasoningTokens()
			if in != 111 || out != 222 || cr != 3 || cw != 4 || rs != 5 {
				t.Errorf("cost_turn figures = (in=%d out=%d cr=%d cw=%d rs=%d), want (111 222 3 4 5)", in, out, cr, cw, rs)
			}
		case agent.EventKindTurnEnd:
			turnEndIdx = i
		case agent.EventKindCostSession:
			cs, ok := ev.CostSession()
			if !ok {
				continue
			}
			sawSession = true
			in, _ := cs.InputTokens()
			sessionInput = in
		}
	}
	if costTurnIdx < 0 {
		t.Fatal("no cost_turn event found on the stream")
	}
	if turnEndIdx < 0 || costTurnIdx >= turnEndIdx {
		t.Fatalf("cost_turn at index %d, turn_end at index %d, want cost_turn strictly before turn_end", costTurnIdx, turnEndIdx)
	}
	if !sawSession {
		t.Fatal("no cost_session event found on the stream")
	}
	if sessionInput != 111 {
		t.Errorf("cost_session input tokens = %d, want 111 (the compaction's own usage, folded)", sessionInput)
	}
}

// TestCompaction_IffHoldsOnAllThreeArms — S-CMP-006, S-CST-015's arm
// table. Three runs — completion+commit, provider error before any
// completion, completion with a non-Stop finish reason — each satisfy
// R-CST-001's iff on their own compaction bracket.
func TestCompaction_IffHoldsOnAllThreeArms(t *testing.T) {
	t.Parallel()

	t.Run("completion and commit succeed", func(t *testing.T) {
		t.Parallel()
		h, hist, _ := markedHarnessForCompaction(t, "006a")
		provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, ai.Usage{Input: ai.Tokens(9)}))
		req := agent.CompactionRequest{Provider: provider, Instruction: "summarize 006a", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err != nil {
			t.Fatalf("Compact returned %v, want nil", err)
		}
		events := drainSink(t, sink)
		outcome, hasCostTurn, hasFinished, hasFailed := compactionArmSummary(t, events)
		if outcome != agent.TurnOutcomeFinished {
			t.Errorf("bracket outcome = %v, want Finished (non-aborted)", outcome)
		}
		if !hasCostTurn {
			t.Error("no cost_turn on the bracket, want exactly one")
		}
		if !hasFinished || hasFailed {
			t.Errorf("hasFinished=%v hasFailed=%v, want finished only", hasFinished, hasFailed)
		}
	})

	t.Run("provider errors before any completion", func(t *testing.T) {
		t.Parallel()
		h, hist, _ := markedHarnessForCompaction(t, "006b")
		provider := agenttest.NewProvider() // no scripts queued: the very first call is exhausted → pre-stream error.
		req := agent.CompactionRequest{Provider: provider, Instruction: "summarize 006b", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err == nil {
			t.Fatal("Compact returned nil, want a non-nil error")
		}
		events := drainSink(t, sink)
		outcome, hasCostTurn, hasFinished, hasFailed := compactionArmSummary(t, events)
		if outcome != agent.TurnOutcomeAborted {
			t.Errorf("bracket outcome = %v, want Aborted", outcome)
		}
		if hasCostTurn {
			t.Error("cost_turn present on a no-completion arm, want none")
		}
		if hasFinished || !hasFailed {
			t.Errorf("hasFinished=%v hasFailed=%v, want failed only", hasFinished, hasFailed)
		}
	})

	t.Run("completion with a non-Stop finish reason", func(t *testing.T) {
		t.Parallel()
		h, hist, _ := markedHarnessForCompaction(t, "006c")
		provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonLength, ai.Usage{Input: ai.Tokens(4)}))
		req := agent.CompactionRequest{Provider: provider, Instruction: "summarize 006c", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err == nil {
			t.Fatal("Compact returned nil, want a non-nil error (a length-limited summary must not be admitted)")
		}
		events := drainSink(t, sink)
		outcome, hasCostTurn, hasFinished, hasFailed := compactionArmSummary(t, events)
		if outcome != agent.TurnOutcomeLengthLimited {
			t.Errorf("bracket outcome = %v, want LengthLimited (mapped from the completion's own finish reason, non-aborted)", outcome)
		}
		if !hasCostTurn {
			t.Error("no cost_turn on a completion-obtained-then-rejected arm, want one — the spend is real and must be reported")
		}
		if hasFinished || !hasFailed {
			t.Errorf("hasFinished=%v hasFailed=%v, want failed only", hasFinished, hasFailed)
		}
	})
}

// compactionArmSummary extracts the compaction bracket's own turn_end
// outcome and whether it carried a cost_turn, a compaction_finished, or
// a compaction_failed — the shape S-CMP-006's three arms are each
// judged by.
func compactionArmSummary(t *testing.T, events []agent.Event) (outcome agent.TurnOutcome, hasCostTurn, hasFinished, hasFailed bool) {
	t.Helper()
	for _, ev := range events {
		switch ev.Kind() {
		case agent.EventKindCostTurn:
			hasCostTurn = true
		case agent.EventKindCompactionFinished:
			hasFinished = true
		case agent.EventKindCompactionFailed:
			hasFailed = true
		case agent.EventKindTurnEnd:
			if te, ok := ev.TurnEnd(); ok {
				outcome = te.Outcome()
			}
		}
	}
	return outcome, hasCostTurn, hasFinished, hasFailed
}
