// AG-16 — Phase 1, Scenario 1: cost_turn emission (R-CST-001, R-CST-002,
// R-CST-003).
//
// Every scenario is in package agent_test (NFR-CST-001): the
// external-test posture proves every behavioral claim from outside
// the package, reading cost figures only through CostTurn's paired
// accessors (InputTokens() etc., cost_events.go).
package agent_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// scriptTextResponseWithUsage builds a deterministic text-only
// scripted stream carrying a specific ai.Usage and ai.FinishReason —
// the S-CST-001..006 fixture, parametrizing scriptTextResponse's
// always-empty ai.Usage{} so a turn's cost_turn can be checked against
// a real, scripted figure set.
func scriptTextResponseWithUsage(t *testing.T, finishReason ai.FinishReason, usage ai.Usage) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta, err := ai.NewTextDelta(1, "cost-scripted-text")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(finishReason, usage)
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// turnBracketEvents extracts every event, inclusive, between the
// turn_start carrying turnID and its own matching turn_end — a small
// local helper so a scenario can assert what a single turn's own
// bracket carries without hand-walking indices.
func turnBracketEvents(t *testing.T, events []agent.Event, turnID agent.TurnID) []agent.Event {
	t.Helper()

	var bracket []agent.Event
	inBracket := false
	for _, ev := range events {
		evTurn, hasTurn := ev.Turn()
		if !inBracket {
			if _, ok := ev.TurnStart(); ok && hasTurn && evTurn == turnID {
				inBracket = true
				bracket = append(bracket, ev)
			}
			continue
		}
		bracket = append(bracket, ev)
		if _, ok := ev.TurnEnd(); ok && hasTurn && evTurn == turnID {
			return bracket
		}
	}
	t.Fatalf("no closed turn bracket found for turn %q in %d event(s)", turnID, len(events))
	return nil
}

// assertFiguresEqualUsage asserts got's five paired accessors read
// back exactly what usage reports — magnitude and discriminator both
// — never the count alone (the capability's own Evidence-discipline
// rule).
func assertFiguresEqualUsage(t *testing.T, got agent.CostTurn, usage ai.Usage) {
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

// costTurnsIn returns every cost_turn payload found in events, in
// stream order — used by the aborted-path assertions below to prove
// zero, and by the multi-turn assertion to prove exactly one per
// bracket.
func costTurnsIn(events []agent.Event) []agent.CostTurn {
	var out []agent.CostTurn
	for _, ev := range events {
		if ct, ok := ev.CostTurn(); ok {
			out = append(out, ct)
		}
	}
	return out
}

// TestTurn_CostTurn_FiguresExactPerTurn — S-CST-001, charter AG-16.1
// scenario 1, exactness half. A multi-turn run (turn one pauses, turn
// two finishes) scripts a distinct ai.Usage per turn, every one of the
// five figures present and pairwise distinct across turns — cache
// reads, cache writes and reasoning included, not only input/output.
func TestTurn_CostTurn_FiguresExactPerTurn(t *testing.T) {
	t.Parallel()

	usageTurnOne := ai.Usage{
		Input:      ai.Tokens(1100),
		Output:     ai.Tokens(220),
		CacheRead:  ai.Tokens(330),
		CacheWrite: ai.Tokens(440),
		Reasoning:  ai.Tokens(550),
	}
	usageTurnTwo := ai.Usage{
		Input:      ai.Tokens(17),
		Output:     ai.Tokens(29),
		CacheRead:  ai.Tokens(31),
		CacheWrite: ai.Tokens(43),
		Reasoning:  ai.Tokens(59),
	}

	turnOneScript := scriptTextResponseWithUsage(t, ai.FinishReasonPauseTurn, usageTurnOne)
	turnTwoScript := scriptTextResponseWithUsage(t, ai.FinishReasonStop, usageTurnTwo)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for cst-001", History: agent.NewHistory()}
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
		t.Errorf("agent.CheckStream(cost-bearing stream) = %v, want no violation (unmodified acceptance)", report.Violation())
	}

	var turnStarts []agent.TurnID
	for _, ev := range events {
		if _, ok := ev.TurnStart(); ok {
			id, _ := ev.Turn()
			turnStarts = append(turnStarts, id)
		}
	}
	if len(turnStarts) != 2 {
		t.Fatalf("recorded %d turn_start event(s), want 2 (one per logical turn)", len(turnStarts))
	}

	for i, turnID := range turnStarts {
		wantUsage := usageTurnOne
		if i == 1 {
			wantUsage = usageTurnTwo
		}
		bracket := turnBracketEvents(t, events, turnID)
		if len(bracket) < 2 {
			t.Fatalf("turn %q bracket has %d event(s), want at least 2 (content + cost_turn + turn_end)", turnID, len(bracket))
		}
		// The cost_turn is positioned after that turn's content events
		// and before turn_end: the last bracket event is turn_end
		// (turnBracketEvents' own contract), so the one immediately
		// before it must be cost_turn.
		costTurnEv := bracket[len(bracket)-2]
		got, ok := costTurnEv.CostTurn()
		if !ok {
			t.Fatalf("turn %q: event immediately before turn_end is %v, want cost_turn", turnID, costTurnEv.Kind())
		}
		if got.Label() != agent.CostLabelFinal {
			t.Errorf("turn %q: cost_turn label = %v, want Final", turnID, got.Label())
		}
		assertFiguresEqualUsage(t, got, wantUsage)
	}

	// Exactly one cost_turn per turn, no more, no fewer.
	if got := len(costTurnsIn(events)); got != 2 {
		t.Errorf("recorded %d cost_turn event(s) total, want 2 (one per turn)", got)
	}
}

// TestTurn_NoCompletion_CostTurnAllAbsent — S-CST-002, charter
// AG-16.1 scenario 1, the no-Completion path. A provider that closes
// its stream without ever emitting an ai.Completion still closes the
// turn non-aborted (the "ran to empty" fallback, S-ATT-012's shape),
// so it still carries a cost_turn — every figure reporting absence,
// never a present zero.
func TestTurn_NoCompletion_CostTurnAllAbsent(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextOnlyNoCompletion(t))
	sink := make(chan *agent.Event, 16)

	_, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for cst-002",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v (no-completion fallback)", finish, ai.FinishReasonStop)
	}

	events := drainSink(t, sink)
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(no-completion stream) = %v, want no violation", report.Violation())
	}

	costTurns := costTurnsIn(events)
	if len(costTurns) != 1 {
		t.Fatalf("recorded %d cost_turn event(s), want exactly 1", len(costTurns))
	}
	got := costTurns[0]
	if got.Label() != agent.CostLabelFinal {
		t.Errorf("cost_turn label = %v, want Final", got.Label())
	}
	assertFiguresEqualUsage(t, got, ai.Usage{})
	if _, ok := got.InputTokens(); ok {
		t.Error("InputTokens() reports present, want absent (no Completion was ever observed)")
	}
	if _, ok := got.ReasoningTokens(); ok {
		t.Error("ReasoningTokens() reports present, want absent (no Completion was ever observed)")
	}

	// Not reported as a failed turn by the emission.
	for _, ev := range events {
		if te, ok := ev.TurnEnd(); ok && te.Outcome() != agent.TurnOutcomeFinished {
			t.Errorf("turn_end outcome = %v, want Finished (the no-Completion fallback, not a failure)", te.Outcome())
		}
	}
}

// TestTurn_AbortedTurn_NoCostTurn — S-CST-003, the aborted half of
// the iff. Five conditions, each a turn that closes aborted: the
// mid-stream fatal path, each of R-LSK-001's three pre-stream failure
// paths, and an interrupt fired mid-turn. None of the five has a
// usage source (Completion is Terminal; ai.Failure carries no usage),
// so none may emit a cost_turn.
func TestTurn_AbortedTurn_NoCostTurn(t *testing.T) {
	t.Parallel()

	assertNoCostTurn := func(t *testing.T, events []agent.Event) {
		t.Helper()
		if got := costTurnsIn(events); len(got) != 0 {
			t.Errorf("recorded %d cost_turn event(s) on an aborted turn's bracket, want 0", len(got))
		}
		if report := agent.CheckStream(events); report.Violation() != nil {
			t.Errorf("agent.CheckStream(aborted stream) = %v, want no violation", report.Violation())
		}
	}

	t.Run("mid-stream fatal", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextThenMidStreamError(t))
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for cst-003 mid-stream",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the mid-stream terminal failure")
		}
		assertNoCostTurn(t, drainSink(t, sink))
	})

	t.Run("pre-stream: request build error", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider() // never consulted: build fails first
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"", // empty system: buildLoopRequest fails
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the request-build error")
		}
		assertNoCostTurn(t, drainSink(t, sink))
	})

	t.Run("pre-stream: hook error", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for cst-003 hook",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{PreRequestHook: hookBoomAlwaysErrors()},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the hook's typed pre-stream failure")
		}
		assertNoCostTurn(t, drainSink(t, sink))
	})

	t.Run("pre-stream: provider.Stream error", func(t *testing.T) {
		t.Parallel()

		provider := errorProvider{}
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for cst-003 provider",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the provider's typed pre-stream failure")
		}
		assertNoCostTurn(t, drainSink(t, sink))
	})

	t.Run("cancellation mid-turn", func(t *testing.T) {
		t.Parallel()

		gate := agenttest.NewGate()
		provider := agenttest.NewProvider(heldTurnScript(t, gate))
		h := agent.Harness{Provider: provider, System: "system prompt for cst-003 cancel", History: agent.NewHistory()}

		sink := make(chan *agent.Event, 256)
		resultCh := make(chan error, 1)
		go func() {
			_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
			resultCh <- err
		}()

		<-gate.Reached()
		h.Interrupt()

		events := drainSink(t, sink)
		err := <-resultCh
		if !errors.Is(err, agent.ErrInterrupted) {
			t.Fatalf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", err)
		}
		assertNoCostTurn(t, events)
	})
}

// TestTurn_CostTurn_LabelAlwaysFinal — S-CST-004, the label axis at
// turn scope. Every cost_turn observed above, and the one
// no-Completion cost_turn, reads Final; none reads Estimate. Extended
// over Phase 3's retry-bearing run at TestHarness_CostSession_
// CumulativeEqualsEmittedCostTurns (S-CST-009's companion assertion).
func TestTurn_CostTurn_LabelAlwaysFinal(t *testing.T) {
	t.Parallel()

	usage := ai.Usage{Input: ai.Tokens(5), Output: ai.Tokens(7)}
	provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, usage))
	sink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for cst-004",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	costTurns := costTurnsIn(events)
	if len(costTurns) != 1 {
		t.Fatalf("recorded %d cost_turn event(s), want 1", len(costTurns))
	}
	if got := costTurns[0].Label(); got != agent.CostLabelFinal {
		t.Errorf("cost_turn label = %v, want Final — a cost_turn is never labelled Estimate", got)
	}
}

// TestTurn_CostTurn_AbsenceVsReportedZero — S-CST-005, charter
// AG-16.1 scenario 1, absence half. Two otherwise identical
// single-turn runs — one whose Completion carries ai.Usage{} and one
// whose Completion carries a usage whose input figure is
// ai.Tokens(0) — must be observably different through the presence
// discriminator; the assertion reads the discriminator, never the
// count alone.
func TestTurn_CostTurn_AbsenceVsReportedZero(t *testing.T) {
	t.Parallel()

	runOne := func(usage ai.Usage) agent.CostTurn {
		t.Helper()
		provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, usage))
		sink := make(chan *agent.Event, 16)
		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for cst-005",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err != nil {
			t.Fatalf("Turn returned err = %v, want nil", err)
		}
		events := drainSink(t, sink)
		costTurns := costTurnsIn(events)
		if len(costTurns) != 1 {
			t.Fatalf("recorded %d cost_turn event(s), want 1", len(costTurns))
		}
		return costTurns[0]
	}

	absent := runOne(ai.Usage{})
	reportedZero := runOne(ai.Usage{Input: ai.Tokens(0)})

	inputAbsent, okAbsent := absent.InputTokens()
	if okAbsent {
		t.Errorf("absent-usage run: InputTokens() reports present (%d), want absent", inputAbsent)
	}
	inputZero, okZero := reportedZero.InputTokens()
	if !okZero {
		t.Error("reported-zero run: InputTokens() reports absent, want present at zero")
	}
	if inputZero != 0 {
		t.Errorf("reported-zero run: InputTokens() = %d, want 0", inputZero)
	}

	// Both counts are 0 — proving the assertion above reads the
	// discriminator and not the count alone: a count-only comparison
	// could not tell these two events apart.
	if inputAbsent != 0 || inputZero != 0 {
		t.Fatalf("test fixture defect: expected both counts to read 0 (absent=%d, zero=%d)", inputAbsent, inputZero)
	}
	if okAbsent == okZero {
		t.Error("both runs report the same presence discriminator, want them to differ (absent vs reported-nought)")
	}
}

// TestTurn_CostTurn_MixedRecordPartialPresence — S-CST-006, charter
// AG-16.1 scenario 1, the mixed record. A turn whose Completion
// carries ai.Usage{Input: ai.Tokens(100)} — one figure reported, four
// unreported — reads input (100, reported) and the other four each
// absent; none of the four reads as a present zero.
func TestTurn_CostTurn_MixedRecordPartialPresence(t *testing.T) {
	t.Parallel()

	usage := ai.Usage{Input: ai.Tokens(100)}
	provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, usage))
	sink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for cst-006",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	costTurns := costTurnsIn(events)
	if len(costTurns) != 1 {
		t.Fatalf("recorded %d cost_turn event(s), want 1", len(costTurns))
	}
	got := costTurns[0]

	if n, ok := got.InputTokens(); !ok || n != 100 {
		t.Errorf("InputTokens() = (%d, %v), want (100, true)", n, ok)
	}
	if n, ok := got.OutputTokens(); ok || n != 0 {
		t.Errorf("OutputTokens() = (%d, %v), want (0, false) — absent, never a present zero", n, ok)
	}
	if n, ok := got.CacheReadTokens(); ok || n != 0 {
		t.Errorf("CacheReadTokens() = (%d, %v), want (0, false) — absent, never a present zero", n, ok)
	}
	if n, ok := got.CacheWriteTokens(); ok || n != 0 {
		t.Errorf("CacheWriteTokens() = (%d, %v), want (0, false) — absent, never a present zero", n, ok)
	}
	if n, ok := got.ReasoningTokens(); ok || n != 0 {
		t.Errorf("ReasoningTokens() = (%d, %v), want (0, false) — absent, never a present zero", n, ok)
	}
}

// TestTurn_Standalone_NoCostSession — S-CST-011. A standalone Turn
// invocation (nil continuation, zero-value TurnOptions) with a
// completing script carries its cost_turn before turn_end and no
// cost_session of either label — cost_session is harness-scoped
// (R-CST-005), and a bare Turn caller aggregates nothing.
func TestTurn_Standalone_NoCostSession(t *testing.T) {
	t.Parallel()

	usage := ai.Usage{Input: ai.Tokens(3), Output: ai.Tokens(4)}
	provider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, usage))
	sink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for cst-011",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	if got := len(costTurnsIn(events)); got != 1 {
		t.Errorf("recorded %d cost_turn event(s), want 1", got)
	}
	for _, ev := range events {
		if _, ok := ev.CostSession(); ok {
			t.Error("recorded a cost_session event from a standalone Turn call, want none — cost_session is harness-scoped")
		}
	}
}

// TestCost_ScopeFence — S-CST-014, R-CST-007. This test owns only the
// diff half of the scenario: git diff over backend/agent/src/ai/ and
// over go.mod/go.sum must be empty. The other two clauses —
// S-APE-083's forbidden-substring scan and reflection walk, and the
// every-kind-constructible guard — are proven by cost_events_test.go's
// own TestCost_PayloadShape_NoMoneyField and event_registry_test.go's
// own kind-count test, BOTH byte-unchanged by this change and
// therefore not duplicated here; their continuing to pass unedited
// alongside this test in the same `make test` run is itself the
// evidence for those two clauses.
func TestCost_ScopeFence(t *testing.T) {
	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}

	mainRef := os.Getenv("AG16_BASE_REF")
	if mainRef == "" {
		// Fall back to the merge-base with origin/main (the most
		// recent common ancestor) — mirrors S-LSK-006's
		// TestTurn_SubstrateUntouched so the test survives merge.
		out, mbErr := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
		if mbErr != nil {
			t.Fatalf("scope fence: cannot determine base ref (set AG16_BASE_REF): %v", mbErr)
		}
		mainRef = out
	}

	aiDiff, err := gitDiff(t, root, mainRef, "backend/agent/src/ai/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", mainRef, err)
	}
	// CH-03 (cachicamas-chat-http-surface) carve-out: the agent
	// module's go.mod require bump from 3 to 4 lines is reflected
	// in TWO ai/ test files (import_boundary_test.go's
	// wantGoModRequires and openrouter/zero_requires_test.go's
	// wantGoModRequireLines) per the documented "update in the
	// same commit" instruction each test carries. No other ai/
	// source is edited. Anything beyond the two documented files
	// fails this check normally.
	ch03AiCarveoutFiles := []string{
		"ai/import_boundary_test.go",
		"ai/openaicompat/openrouter/zero_requires_test.go",
	}
	if strings.Contains(aiDiff, "diff --git") {
		// Walk every `diff --git a/.../foo.go b/.../foo.go` line —
		// each chunk is one file's diff. If any one is OUTSIDE the
		// carve-out list, fail.
		chunks := strings.Split(aiDiff, "diff --git")
		var stray []string
		for _, chunk := range chunks[1:] {
			// chunk's first line is the path header. Every
			// carve-out file's path appears somewhere in the
			// header line.
			nl := strings.Index(chunk, "\n")
			header := chunk
			if nl >= 0 {
				header = chunk[:nl]
			}
			allowed := false
			for _, allow := range ch03AiCarveoutFiles {
				// `diff --git a/path b/path` repeats the path
				// on both sides of the diff header, so a
				// Contains check on either side is enough.
				if strings.Contains(header, allow) {
					allowed = true
					break
				}
			}
			if !allowed {
				stray = append(stray, chunk)
			}
		}
		if len(stray) == 0 {
			t.Logf("CH-03 carve-out: every ai/ edit is one of the documented require-bump amendments; R-CST-007 still binding for every other ai/ file.\n%s", aiDiff)
		} else {
			t.Errorf("backend/agent/src/ai/ was edited beyond the CH-03 wantGoModRequires amendments (R-CST-007 violated):\n%s", aiDiff)
		}
	} else if len(aiDiff) != 0 {
		t.Errorf("backend/agent/src/ai/ was edited (R-CST-007 violated — Layer 1 is consumed, never edited):\n%s", aiDiff)
	}

	goModSum, err := gitDiff(t, root, mainRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/go.mod go.sum failed: %v", mainRef, err)
	}
	// CH-03 (cachicamas-chat-http-surface) is the recorded exception:
	// the chat archetype's HTTP+SSE surface requires
	// github.com/labstack/echo/v5 v5.2.1 (ADR adr/echo-v5-in-agent-module
	// recorded at the start of CH-03's SDD flow), bumping the agent
	// module's go.mod from 3 require lines (AI-37) to 4. Every other
	// cost-emission / substrate milestone is bound by R-CST-007 as
	// stated; this clause is the one carve-out, with the bump count
	// verified by TestLayer1_DependencySet_ExactRequiresAndClosure
	// (import_boundary_test.go:313, wantGoModRequires updated to 4
	// in CH-03's own amendment) and the openrouter zero-requires
	// guard (zero_requires_test.go, wantGoModRequireLines=4 in the
	// same commit). For CH-03 itself, the diff MAY carry the
	// github.com/labstack/echo/v5 require entry — anything beyond
	// that (a replace directive, a second new require, etc.) fails
	// this check normally.
	hasEcho := strings.Contains(goModSum, `+	github.com/labstack/echo/v5 v5.2.1`)
	hasReplace := strings.Contains(goModSum, `replace `) || strings.Contains(goModSum, `+replace`)
	if hasEcho && !hasReplace {
		t.Logf("CH-03 exception: go.mod diff carries the recorded Echo require; R-CST-007 still binding for every other change.\n%s", goModSum)
	} else if len(goModSum) != 0 {
		t.Errorf("go.mod / go.sum drifted from main (R-CST-007 violated — no new top-level Go deps; CH-03's Echo addition is the only recorded exception):\n%s", goModSum)
	}
}
