// AG-18.3/AG-18.5 — Phase 4: reconstruction from the recorded stream
// (R-CMP-009, R-CMP-012, R-CMP-014), package agent_test (NFR-CMP-001).
// reconstruction_test.go itself is byte-unchanged (design AD-12) — this
// file is deliberately new.
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestCompaction_ReconstructionNamesReplacedTurns — S-CMP-018, charter
// AG-18.3 scenario, the reconstruction half. A reconstruction driven
// only from compaction_finished's own payload names exactly the two
// earlier turn brackets its replaced prefix covered, and locates the
// summary entry in the post-compaction read-back by message identity
// alone.
func TestCompaction_ReconstructionNamesReplacedTurns(t *testing.T) {
	t.Parallel()

	const tool1, call1 = "read_cmp_018a", "call-cmp-018a"
	const tool2, call2 = "read_cmp_018b", "call-cmp-018b"
	reg := agent.NewMapRegistry(map[string]agent.Tool{
		tool1: EchoScriptedTool(tool1, agent.EffectClassRead),
		tool2: EchoScriptedTool(tool2, agent.EffectClassRead),
	})

	ownProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, call1, tool1, []byte(`{"x":1}`)),
		scriptToolCallResponse(t, call2, tool2, []byte(`{"y":2}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	strategy := &compactAfterNStrategy{
		skip: 2, // skip pre-turn-1 and pre-turn-2 (fewer than 2 marks exist yet).
		request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for S-CMP-018", Cut: len(prompt.Transcript)}
		},
	}
	hist := agent.NewHistory()
	h := &agent.Harness{Provider: ownProvider, System: "system prompt for S-CMP-018", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy, History: hist}

	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	events := drainSink(t, sink)

	// Every turn_start's own TurnID, in first-appearance order —
	// turn1, turn2, the compaction bracket's own, then turn3.
	var turnStartOrder []agent.TurnID
	seen := map[agent.TurnID]bool{}
	var finished agent.CompactionFinished
	foundFinished := false
	for _, ev := range events {
		if ev.Kind() == agent.EventKindTurnStart {
			turn, _ := ev.Turn()
			if !seen[turn] {
				seen[turn] = true
				turnStartOrder = append(turnStartOrder, turn)
			}
		}
		if ev.Kind() == agent.EventKindCompactionFinished {
			finished, foundFinished = ev.CompactionFinished()
		}
	}
	if !foundFinished {
		t.Fatal("no compaction_finished event found")
	}
	if len(turnStartOrder) < 3 {
		t.Fatalf("fixture defect: observed %d distinct turn bracket(s), want at least 3 (turn1, turn2, the compaction bracket)", len(turnStartOrder))
	}
	wantStart, wantEnd := turnStartOrder[0], turnStartOrder[1]

	span := finished.Span()
	if span.StartTurnID != wantStart || span.EndTurnID != wantEnd {
		t.Errorf("Span() = {%v, %v}, want {%v, %v} (turn1, turn2 — the two turn brackets the replaced prefix covered)", span.StartTurnID, span.EndTurnID, wantStart, wantEnd)
	}
	for _, want := range []agent.TurnID{span.StartTurnID, span.EndTurnID} {
		found := false
		for _, id := range turnStartOrder {
			if id == want {
				found = true
			}
		}
		if !found {
			t.Errorf("span turn %v does not appear as an earlier turn bracket on the stream", want)
		}
	}

	readBack := hist.Entries()
	matches := 0
	for _, e := range readBack {
		if e.Message().ID().String() == finished.SummaryID() {
			matches++
			if e.Origin() != agent.EntryOriginSummarized {
				t.Errorf("the entry matching SummaryID() has origin %v, want EntryOriginSummarized", e.Origin())
			}
		}
	}
	if matches != 1 {
		t.Fatalf("%d entries match SummaryID() %q, want exactly 1", matches, finished.SummaryID())
	}
}

// TestCompaction_FinishedXorFailed — S-CMP-019. Across a successful and
// a failed compaction, no operation produces both a finished and a
// failed event, and every started event is followed by exactly one of
// the two inside the same bracket.
func TestCompaction_FinishedXorFailed(t *testing.T) {
	t.Parallel()

	check := func(t *testing.T, events []agent.Event) {
		t.Helper()
		startedTurns := map[agent.TurnID]bool{}
		finishedTurns := map[agent.TurnID]bool{}
		failedTurns := map[agent.TurnID]bool{}
		for _, ev := range events {
			turn, _ := ev.Turn()
			switch ev.Kind() {
			case agent.EventKindCompactionStarted:
				startedTurns[turn] = true
			case agent.EventKindCompactionFinished:
				finishedTurns[turn] = true
			case agent.EventKindCompactionFailed:
				failedTurns[turn] = true
			}
		}
		if len(startedTurns) == 0 {
			t.Fatal("fixture defect: no compaction_started event observed")
		}
		for turn := range startedTurns {
			f, fl := finishedTurns[turn], failedTurns[turn]
			if f == fl {
				t.Errorf("turn %v: finished=%v failed=%v, want exactly one", turn, f, fl)
			}
		}
	}

	t.Run("successful compaction", func(t *testing.T) {
		t.Parallel()
		h, hist, _ := markedHarnessForCompaction(t, "019a")
		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		req := agent.CompactionRequest{Provider: provider, Instruction: "summarize 019a", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err != nil {
			t.Fatalf("Compact returned %v, want nil", err)
		}
		check(t, drainSink(t, sink))
	})

	t.Run("failed compaction", func(t *testing.T) {
		t.Parallel()
		h, hist, _ := markedHarnessForCompaction(t, "019b")
		provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonLength, ai.Usage{}))
		req := agent.CompactionRequest{Provider: provider, Instruction: "summarize 019b", Cut: hist.Len()}
		sink := make(chan *agent.Event, 64)
		if err := h.Compact(contextBackground(), req, sink); err == nil {
			t.Fatal("Compact returned nil, want a non-nil error")
		}
		check(t, drainSink(t, sink))
	})
}

// TestCompaction_MarkedSpanFromTwoRuns — S-CMP-029. A history driven
// through two complete runs, then compacted across the boundary
// between them: the span names the first run's own turn and the second
// run's own turn, and both appear as turn brackets earlier on the
// recorded stream.
func TestCompaction_MarkedSpanFromTwoRuns(t *testing.T) {
	t.Parallel()

	hist := agent.NewHistory()

	provider1 := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h1 := &agent.Harness{Provider: provider1, System: "system prompt for S-CMP-029 run 1", History: hist}
	sink1 := make(chan *agent.Event, 64)
	if _, _, err := h1.Run(contextBackground(), firstMessage(t), sink1); err != nil {
		t.Fatalf("run 1 returned %v, want nil", err)
	}
	events1 := drainSink(t, sink1)
	var run1TurnID agent.TurnID
	for _, ev := range events1 {
		if ev.Kind() == agent.EventKindTurnStart {
			run1TurnID, _ = ev.Turn()
		}
	}

	provider2 := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	h2 := &agent.Harness{Provider: provider2, System: "system prompt for S-CMP-029 run 2", History: hist}
	sink2 := make(chan *agent.Event, 64)
	if _, _, err := h2.Run(contextBackground(), firstMessage(t), sink2); err != nil {
		t.Fatalf("run 2 returned %v, want nil", err)
	}
	events2 := drainSink(t, sink2)
	var run2TurnID agent.TurnID
	for _, ev := range events2 {
		if ev.Kind() == agent.EventKindTurnStart {
			run2TurnID, _ = ev.Turn()
		}
	}
	if run1TurnID == "" || run2TurnID == "" || run1TurnID == run2TurnID {
		t.Fatalf("fixture defect: run1TurnID=%q run2TurnID=%q, want two distinct non-empty identities", run1TurnID, run2TurnID)
	}

	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize across both runs", Cut: hist.Len()}
	sink3 := make(chan *agent.Event, 64)
	if err := h2.Compact(contextBackground(), req, sink3); err != nil {
		t.Fatalf("Compact returned %v, want nil", err)
	}
	events3 := drainSink(t, sink3)

	var finished agent.CompactionFinished
	found := false
	for _, ev := range events3 {
		if ev.Kind() == agent.EventKindCompactionFinished {
			finished, found = ev.CompactionFinished()
		}
	}
	if !found {
		t.Fatal("no compaction_finished event found")
	}
	span := finished.Span()
	if span.StartTurnID != run1TurnID {
		t.Errorf("Span().StartTurnID = %v, want %v (run 1's own turn)", span.StartTurnID, run1TurnID)
	}
	if span.EndTurnID != run2TurnID {
		t.Errorf("Span().EndTurnID = %v, want %v (run 2's own turn)", span.EndTurnID, run2TurnID)
	}

	allEarlierIDs := map[agent.TurnID]bool{}
	for _, ev := range events1 {
		if turn, ok := ev.Turn(); ok {
			allEarlierIDs[turn] = true
		}
	}
	for _, ev := range events2 {
		if turn, ok := ev.Turn(); ok {
			allEarlierIDs[turn] = true
		}
	}
	if !allEarlierIDs[span.StartTurnID] || !allEarlierIDs[span.EndTurnID] {
		t.Error("span identifiers do not both appear as turn brackets earlier on the recorded stream")
	}
}

// TestCompaction_UnmarkedPrefixFailsTyped — S-CMP-034, the named v1
// limitation asserted rather than assumed. A history constructed from
// a seed no run ever drove carries no turn mark, so a compaction over
// any prefix of it fails typed before any provider call, with no
// identifier invented and the transcript unchanged.
func TestCompaction_UnmarkedPrefixFailsTyped(t *testing.T) {
	t.Parallel()

	seedPart, err := ai.NewText("seeded, never driven by a run")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want nil", err)
	}
	seedMsg, err := ai.NewMessage(ai.RoleAssistant, seedPart)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want nil", err)
	}
	hist, err := agent.NewSeededHistory([]ai.Message{seedMsg})
	if err != nil {
		t.Fatalf("NewSeededHistory returned %v, want nil", err)
	}
	pre := hist.Entries()

	h := &agent.Harness{Provider: agenttest.NewProvider(), History: hist}
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	req := agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize the unmarked seed", Cut: hist.Len()}

	sink := make(chan *agent.Event, 64)
	err = h.Compact(contextBackground(), req, sink)
	events := drainSink(t, sink)

	if err == nil {
		t.Fatal("Compact returned nil, want a typed failure (no recorded turn mark exists)")
	}
	if got := len(compactionProvider.Requests()); got != 0 {
		t.Errorf("compaction provider recorded %d request(s), want 0 — an unattributable cut must fail before any provider call", got)
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
		if !pre[i].Message().Equal(post[i].Message()) || pre[i].ID() != post[i].ID() {
			t.Errorf("entry %d changed after a rejected compaction, want byte-identical", i)
		}
	}
}

// TestCompaction_InertUnlessRequested — S-CMP-037. Two runs of an
// identical multi-turn script — one with no strategy installed, one
// with a strategy that never requests compaction on any consultation —
// produce byte-identical event streams and byte-identical history
// read-backs, and neither carries any compaction-family kind.
func TestCompaction_InertUnlessRequested(t *testing.T) {
	t.Parallel()

	run := func(t *testing.T, strategy agent.ContextStrategy) ([]agent.Event, []agent.Entry) {
		t.Helper()
		const toolName, callID = "read_cmp_037", "call-cmp-037"
		tool := EchoScriptedTool(toolName, agent.EffectClassRead)
		reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
		provider := agenttest.NewProvider(
			scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
			scriptTextResponse(t, ai.FinishReasonStop),
		)
		hist := agent.NewHistory()
		h := &agent.Harness{Provider: provider, System: "system prompt for S-CMP-037", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy, History: hist}
		sink := make(chan *agent.Event, 256)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned %v, want nil", err)
		}
		return drainSink(t, sink), hist.Entries()
	}

	nilEvents, nilEntries := run(t, nil)
	neverStrategy := &compactAfterNStrategy{skip: 1 << 30} // never fires: every consultation is within skip.
	neverEvents, neverEntries := run(t, neverStrategy)

	if len(nilEvents) != len(neverEvents) {
		t.Fatalf("event count: nil-strategy = %d, never-compacting strategy = %d, want equal", len(nilEvents), len(neverEvents))
	}
	for i := range nilEvents {
		if nilEvents[i].Kind() != neverEvents[i].Kind() {
			t.Errorf("event[%d].Kind(): nil-strategy = %v, never-compacting = %v, want equal", i, nilEvents[i].Kind(), neverEvents[i].Kind())
		}
		if isCompactionKind(nilEvents[i].Kind()) || isCompactionKind(neverEvents[i].Kind()) {
			t.Errorf("event[%d] is compaction-family, want none anywhere on either stream", i)
		}
	}

	if len(nilEntries) != len(neverEntries) {
		t.Fatalf("history entry count: nil-strategy = %d, never-compacting = %d, want equal", len(nilEntries), len(neverEntries))
	}
	for i := range nilEntries {
		if !nilEntries[i].Message().Equal(neverEntries[i].Message()) || nilEntries[i].ID() != neverEntries[i].ID() || nilEntries[i].Origin() != neverEntries[i].Origin() {
			t.Errorf("history entry %d differs between the nil-strategy and never-compacting runs, want byte-identical", i)
		}
	}
}
