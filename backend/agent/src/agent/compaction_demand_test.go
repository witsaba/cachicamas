// AG-18.5 — Phase 6: the on-demand entry point (R-CMP-001, R-CMP-011),
// package agent_test (NFR-CMP-001).
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

// heldThenCompletesScript builds a script that blocks at gate, then —
// once released — completes a plain text-only turn: the shared
// "a run held mid-turn" fixture for S-CMP-024.
func heldThenCompletesScript(t *testing.T, gate *agenttest.Gate) agenttest.Script {
	t.Helper()
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Hold(gate),
		agenttest.Emit(completion),
	}}
}

// TestHarness_Compact_TypedRefusalMidTurn — S-CMP-024, charter AG-18.5
// scenario 2. A run held mid-turn by a test gate: a concurrent Compact
// call returns synchronously, typed, errors.Is-distinguishable; zero
// events reach the caller's own sink; the in-flight turn completes
// unaffected; and the run's own recorded stream is byte-identical to
// the same run driven with no concurrent demand.
func TestHarness_Compact_TypedRefusalMidTurn(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	heldProvider := agenttest.NewProvider(heldThenCompletesScript(t, gate))
	hist := agent.NewHistory()
	h := &agent.Harness{Provider: heldProvider, System: "system prompt for S-CMP-024", History: hist}

	sink := make(chan *agent.Event, 64)
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- err
	}()

	<-gate.Reached()

	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize mid-turn", Cut: 1}
	compactSink := make(chan *agent.Event, 16)
	compactErr := h.Compact(contextBackground(), req, compactSink)

	if compactErr == nil {
		t.Fatal("Compact returned nil, want a typed refusal")
	}
	if !errors.Is(compactErr, agent.ErrRunInFlight) {
		t.Errorf("errors.Is(err, agent.ErrRunInFlight) = false, want true (err = %v)", compactErr)
	}
	select {
	case ev, ok := <-compactSink:
		t.Fatalf("Compact's own sink carries an event/close (ev=%v ok=%v), want zero events delivered", ev, ok)
	default:
	}
	if got := len(compactionProvider.Requests()); got != 0 {
		t.Errorf("the refused compaction's own provider recorded %d request(s), want 0", got)
	}

	gate.Release()
	runErr := <-resultCh
	if runErr != nil {
		t.Fatalf("in-flight Run returned err = %v, want nil (unaffected by the concurrent refused demand)", runErr)
	}
	events := drainSink(t, sink)

	// Byte-identical to the same run with no concurrent demand: the
	// baseline reuses the EXACT same script shape (Hold-then-complete),
	// with its own gate pre-released so it never actually blocks —
	// otherwise a baseline built from a differently-shaped script (e.g.
	// scriptTextResponse's richer text bracket) would differ in event
	// count for a reason having nothing to do with the concurrent
	// demand this scenario is about.
	baselineGate := agenttest.NewGate()
	baselineGate.Release()
	baselineProvider := agenttest.NewProvider(heldThenCompletesScript(t, baselineGate))
	baselineH := &agent.Harness{Provider: baselineProvider, System: "system prompt for S-CMP-024"}
	baselineSink := make(chan *agent.Event, 64)
	if _, _, err := baselineH.Run(contextBackground(), firstMessage(t), baselineSink); err != nil {
		t.Fatalf("baseline Run returned %v, want nil", err)
	}
	baselineEvents := drainSink(t, baselineSink)
	if len(events) != len(baselineEvents) {
		t.Fatalf("event count: with-demand=%d baseline=%d, want equal", len(events), len(baselineEvents))
	}
	for i := range events {
		if events[i].Kind() != baselineEvents[i].Kind() {
			t.Errorf("event[%d].Kind(): with-demand=%v baseline=%v, want equal", i, events[i].Kind(), baselineEvents[i].Kind())
		}
	}
}

// TestHarness_Compact_OnDemandUnmodifiedStream — S-CMP-023. No run in
// flight: the emitted stream is accepted by CheckStream unmodified,
// carries exactly one run bracket enclosing exactly one compaction
// bracket, and carries the run's cumulative cost event before the run
// close.
func TestHarness_Compact_OnDemandUnmodifiedStream(t *testing.T) {
	t.Parallel()

	h, hist, _ := markedHarnessForCompaction(t, "023")
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for S-CMP-023", Cut: hist.Len()}

	sink := make(chan *agent.Event, 64)
	if err := h.Compact(contextBackground(), req, sink); err != nil {
		t.Fatalf("Compact returned %v, want nil", err)
	}
	events := drainSink(t, sink)

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Fatalf("CheckStream rejected the on-demand stream: %v", report.Violation())
	}

	runStarts, runEnds, compactionStarts := 0, 0, 0
	costSessionIdx, runEndIdx := -1, -1
	for i, ev := range events {
		switch ev.Kind() {
		case agent.EventKindRunStart:
			runStarts++
		case agent.EventKindRunEnd:
			runEnds++
			runEndIdx = i
		case agent.EventKindCompactionStarted:
			compactionStarts++
		case agent.EventKindCostSession:
			costSessionIdx = i
		}
	}
	if runStarts != 1 || runEnds != 1 {
		t.Errorf("run brackets: %d start(s), %d end(s), want exactly 1 each", runStarts, runEnds)
	}
	if compactionStarts != 1 {
		t.Errorf("compaction brackets: %d compaction_started event(s), want exactly 1", compactionStarts)
	}
	if costSessionIdx < 0 || runEndIdx < 0 || costSessionIdx >= runEndIdx {
		t.Errorf("cost_session at index %d, run_end at index %d, want cost_session strictly before run_end", costSessionIdx, runEndIdx)
	}
}

// TestHarness_Compact_RefusesAfterShutdown — S-CMP-025. After a
// terminal shutdown, the on-demand door refuses typed on the existing
// terminal-shutdown route, emits no event, and mutates no transcript.
func TestHarness_Compact_RefusesAfterShutdown(t *testing.T) {
	t.Parallel()

	hist := agent.NewHistory()
	h := &agent.Harness{Provider: agenttest.NewProvider(), History: hist}
	h.Shutdown()

	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize", Cut: 1}
	sink := make(chan *agent.Event, 16)
	err := h.Compact(contextBackground(), req, sink)

	if err == nil {
		t.Fatal("Compact returned nil, want a typed refusal")
	}
	if !errors.Is(err, agent.ErrPromptAfterShutdown) {
		t.Errorf("errors.Is(err, agent.ErrPromptAfterShutdown) = false, want true (err = %v) -- want the same route as a post-shutdown prompt", err)
	}
	select {
	case ev, ok := <-sink:
		t.Fatalf("sink carries an event/close (ev=%v ok=%v), want zero events delivered", ev, ok)
	default:
	}
	if got := len(compactionProvider.Requests()); got != 0 {
		t.Errorf("compaction provider recorded %d request(s), want 0", got)
	}
	if got := hist.Len(); got != 0 {
		t.Errorf("history carries %d entries after a refused Compact, want 0 unchanged", got)
	}
}

// compactionBracketKinds extracts the Kind()-for-Kind() sub-sequence of
// events belonging to the compaction turn bracket (turn_start through
// its matching turn_end), located by its own compaction_started event
// — the shape S-CMP-022 compares, excluding the fresh RunID/TurnID
// values themselves (mirroring context_strategy_test.go's own
// RunID/TurnID exclusion in assertEventStreamsStructurallyIdentical).
func compactionBracketKinds(t *testing.T, events []agent.Event) []agent.EventKind {
	t.Helper()
	var compactionTurnID agent.TurnID
	for _, ev := range events {
		if ev.Kind() == agent.EventKindCompactionStarted {
			compactionTurnID, _ = ev.Turn()
		}
	}
	var kinds []agent.EventKind
	for _, ev := range events {
		if turn, ok := ev.Turn(); ok && turn == compactionTurnID {
			kinds = append(kinds, ev.Kind())
		}
	}
	return kinds
}

// TestCompaction_OneDemandSharedMechanics — S-CMP-022, charter AG-18.5
// scenario 1, the shared-mechanics half. A strategy-triggered
// compaction and an on-demand one over an equivalent transcript reduce
// to Kind()-for-Kind()-equal compaction bracket sub-sequences, and both
// histories read back with the same entry count and the same origin
// sequence.
func TestCompaction_OneDemandSharedMechanics(t *testing.T) {
	t.Parallel()

	const toolName, callID = "read_cmp_022", "call-cmp-022"
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: EchoScriptedTool(toolName, agent.EffectClassRead)})

	// Strategy-triggered path: turn 1 (tool call) completes and marks;
	// the boundary consultation requests compaction, which commits
	// before the run attempts turn 2. Deliberately only ONE script is
	// queued — turn 2's own attempt exhausts the queue and fails
	// pre-stream, so Run returns a (tolerated, expected) non-nil error
	// WITHOUT turn 2 ever appending anything to history (only the
	// completion arm of Turn commits on the continuation path). This
	// is what lets triggeredHist's read-back be compared directly
	// against the on-demand door's own read-back below, with neither
	// side carrying an unrelated extra turn the comparison would
	// otherwise have to account for.
	ownProvider := agenttest.NewProvider(scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)))
	triggeredCompactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	strategy := &compactAfterNStrategy{
		skip: 1,
		request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{Provider: triggeredCompactionProvider, Instruction: "summarize", Cut: len(prompt.Transcript)}
		},
	}
	triggeredHist := agent.NewHistory()
	th := &agent.Harness{Provider: ownProvider, System: "system prompt for S-CMP-022 triggered", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy, History: triggeredHist}
	triggeredSink := make(chan *agent.Event, 256)
	if _, _, err := th.Run(contextBackground(), firstMessage(t), triggeredSink); err == nil {
		t.Fatal("triggered Run returned nil, want the deliberately-scripted turn-2 exhaustion error (fixture design, not a compaction failure)")
	}
	triggeredEvents := drainSink(t, triggeredSink)

	// On-demand path over an equivalent, independently-driven,
	// one-turn-marked transcript (markedHarnessForCompaction's own
	// fixture — reused here rather than re-deriving a tool-calling
	// drive, since S-CMP-022 compares the compaction bracket's OWN
	// Kind()-for-Kind() sub-sequence, not the driving turn's shape).
	dh, demandHist, _ := markedHarnessForCompaction(t, "022demand")

	demandCompactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	demandReq := agent.CompactionRequest{Provider: demandCompactionProvider, Instruction: "summarize", Cut: demandHist.Len()}
	demandSink := make(chan *agent.Event, 64)
	if err := dh.Compact(contextBackground(), demandReq, demandSink); err != nil {
		t.Fatalf("Compact returned %v, want nil", err)
	}
	demandEvents := drainSink(t, demandSink)

	triggeredBracket := compactionBracketKinds(t, triggeredEvents)
	demandBracket := compactionBracketKinds(t, demandEvents)
	if len(triggeredBracket) != len(demandBracket) {
		t.Fatalf("compaction bracket length: triggered=%d demand=%d, want equal", len(triggeredBracket), len(demandBracket))
	}
	for i := range triggeredBracket {
		if triggeredBracket[i] != demandBracket[i] {
			t.Errorf("bracket[%d]: triggered=%v demand=%v, want equal", i, triggeredBracket[i], demandBracket[i])
		}
	}

	triggeredReadBack := triggeredHist.Entries()
	demandReadBack := demandHist.Entries()
	if len(triggeredReadBack) != len(demandReadBack) {
		t.Fatalf("read-back entry count: triggered=%d demand=%d, want equal", len(triggeredReadBack), len(demandReadBack))
	}
	for i := range triggeredReadBack {
		if triggeredReadBack[i].Origin() != demandReadBack[i].Origin() {
			t.Errorf("read-back[%d] origin: triggered=%v demand=%v, want equal", i, triggeredReadBack[i].Origin(), demandReadBack[i].Origin())
		}
	}
}

// alwaysCompactingStrategy requests a compaction on EVERY consultation
// — a fresh provider each time, since a Provider's scripts are
// single-use — and counts how many times it was consulted, for
// S-CMP-026's "one bracket per boundary" comparison.
type alwaysCompactingStrategy struct {
	mu          sync.Mutex
	seen        int
	newProvider func() *agenttest.Provider
}

func (s *alwaysCompactingStrategy) Resolve(_ context.Context, prompt agent.ContextPrompt) agent.ContextVerdict {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen++
	return agent.ContextVerdict{Compaction: &agent.CompactionRequest{
		Provider:    s.newProvider(),
		Instruction: "summarize every boundary",
		Cut:         len(prompt.Transcript),
	}}
}

func (s *alwaysCompactingStrategy) consultations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seen
}

var _ agent.ContextStrategy = (*alwaysCompactingStrategy)(nil)

// TestCompaction_AtMostOnePerBoundary — S-CMP-026. A strategy
// requesting compaction on every consultation produces exactly one
// compaction bracket per consultation (equivalently, per logical
// turn boundary) — never more.
func TestCompaction_AtMostOnePerBoundary(t *testing.T) {
	t.Parallel()

	const tool1, call1 = "read_cmp_026a", "call-cmp-026a"
	const tool2, call2 = "read_cmp_026b", "call-cmp-026b"
	reg := agent.NewMapRegistry(map[string]agent.Tool{
		tool1: EchoScriptedTool(tool1, agent.EffectClassRead),
		tool2: EchoScriptedTool(tool2, agent.EffectClassRead),
	})
	ownProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, call1, tool1, []byte(`{"x":1}`)),
		scriptToolCallResponse(t, call2, tool2, []byte(`{"y":2}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)

	strategy := &alwaysCompactingStrategy{
		newProvider: func() *agenttest.Provider { return agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop)) },
	}
	hist := agent.NewHistory()
	h := &agent.Harness{Provider: ownProvider, System: "system prompt for S-CMP-026", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy, History: hist}
	sink := make(chan *agent.Event, 512)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	events := drainSink(t, sink)

	compactionBrackets := 0
	for _, ev := range events {
		if ev.Kind() == agent.EventKindCompactionStarted {
			compactionBrackets++
		}
	}
	want := strategy.consultations()
	if want == 0 {
		t.Fatal("fixture defect: the strategy was never consulted")
	}
	if compactionBrackets != want {
		t.Errorf("compaction brackets = %d, want exactly %d (one per logical-turn boundary, the number of consultations)", compactionBrackets, want)
	}
}
