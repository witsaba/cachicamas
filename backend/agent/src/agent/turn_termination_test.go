// AG-11.1 — turn termination: the extended TurnOutcome vocabulary
// (R-ATT-001), the exhaustive finish-reason dispatch consumed by the
// loop's terminator (R-ATT-002), the agent-level exhaustiveness pin
// (R-ATT-003), and refusal/pause divergence (R-ATT-004).
// package agent_test (NFR-ATT-001): every behavioral claim proven from
// outside the package.
//
// # Scope
//
// This file carries AG-11.1's charter leaf: 7 spec scenarios
// (S-ATT-001..006, S-ATT-012) and 3 bites (S-TTB-001..003). AG-11.2's
// typed-failure emission path lives in turn_failure_test.go;
// R-ATT-006's PartialOutput() discriminator lives in
// invariant_pin_test.go (joint closure with AG-04.3, per that file's
// own header comment).
package agent_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-ATT-001 — distinct member per finish reason. Walks the closed
// TurnOutcome range using agent.NewTurnEnd's own success/failure as the
// membership oracle (the design's exhaustiveness-pin mechanism (4)):
// turnOutcomeLimit is unexported, so an external test package cannot
// name the bound directly and instead treats "NewTurnEnd accepts it"
// as the membership criterion. Every accepted value must render a
// distinct, non-placeholder String() form, and exactly 8 members must
// exist (TurnOutcomeFinished, TurnOutcomeAborted, and the six R-ATT-001
// additions).
func TestTurnOutcome_DistinctMemberPerFinishReason(t *testing.T) {
	t.Parallel()

	failure := mustAgentFailure(t)

	seen := map[string]agent.TurnOutcome{}
	var members int
	for v := 0; v <= 255; v++ {
		outcome := agent.TurnOutcome(v)
		var f *agent.Failure
		if outcome == agent.TurnOutcomeAborted {
			f = failure
		}
		if _, err := agent.NewTurnEnd("run-att-001", "turn-att-001", outcome, f); err != nil {
			continue // not a member — NewTurnEnd is the membership oracle
		}
		members++

		s := outcome.String()
		if s == "unset" || strings.HasPrefix(s, "turnoutcome(") {
			t.Errorf("TurnOutcome(%d) is accepted by NewTurnEnd but renders the placeholder %q, want a real name", v, s)
		}
		if prior, dup := seen[s]; dup {
			t.Errorf("TurnOutcome(%d) and %v both render %q — String() forms must be distinct per member", v, prior, s)
		}
		seen[s] = outcome
	}
	if members != 8 {
		t.Errorf("agent.NewTurnEnd accepted %d TurnOutcome value(s), want exactly 8 (Finished, Aborted, LengthLimited, ToolCalls, ContentFiltered, Refused, Paused, Unknown)", members)
	}
}

// S-ATT-002 — the zero value stays a non-member and the failure rule is
// unchanged. References every one of the six R-ATT-001 additions by
// name (not merely by numeric walk), so this test alone fails to
// compile until turn_events.go grows the vocabulary (task 1.3) — the
// deliberate compile-error RED the task list expects.
func TestTurnOutcome_ZeroAndFailureRuleUnchanged(t *testing.T) {
	t.Parallel()

	if _, err := agent.NewTurnEnd("run-att-002", "turn-att-002", 0, nil); err == nil {
		t.Error("agent.NewTurnEnd(zero outcome) returned no violation, want one — the zero value must stay a non-member")
	}

	failure := mustAgentFailure(t)
	nonAbortedMembers := []agent.TurnOutcome{
		agent.TurnOutcomeFinished,
		agent.TurnOutcomeLengthLimited,
		agent.TurnOutcomeToolCalls,
		agent.TurnOutcomeContentFiltered,
		agent.TurnOutcomeRefused,
		agent.TurnOutcomePaused,
		agent.TurnOutcomeUnknown,
	}
	for _, o := range nonAbortedMembers {
		if _, err := agent.NewTurnEnd("run-att-002", "turn-att-002", o, failure); err == nil {
			t.Errorf("agent.NewTurnEnd(%v, non-nil failure) returned no violation, want one — a Failure is forbidden except for TurnOutcomeAborted", o)
		}
	}

	if _, err := agent.NewTurnEnd("run-att-002", "turn-att-002", agent.TurnOutcomeAborted, nil); err == nil {
		t.Error("agent.NewTurnEnd(TurnOutcomeAborted, nil) returned no violation, want one — a Failure is required for TurnOutcomeAborted")
	}
}

// scriptTerminationResponse builds a minimal text-only scripted stream
// ending in finishReason — this file's shared fixture for the
// finish-reason dispatch scenarios (S-ATT-003, S-ATT-004, S-ATT-005).
// Distinct from loop_test.go's scriptTextResponse only in name, kept
// local so this file's fixtures read independently of loop_test.go's
// own evolution.
func scriptTerminationResponse(t *testing.T, finishReason ai.FinishReason) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta, err := ai.NewTextDelta(1, "dispatch-fixture")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(finishReason, ai.Usage{})
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

// runTerminationTurn scripts a text-only turn ending in finishReason,
// runs agent.Turn, drains the sink, and returns the turn_end payload —
// the shared "one turn, one finish reason, one observed outcome" step
// this file's dispatch scenarios all perform.
func runTerminationTurn(t *testing.T, finishReason ai.FinishReason) agent.TurnEnd {
	t.Helper()

	provider := agenttest.NewProvider(scriptTerminationResponse(t, finishReason))
	sink := make(chan *agent.Event, 16)

	_, gotFinish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for termination dispatch",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("agent.Turn returned err = %v, want nil (finish reason %v)", err, finishReason)
	}
	if gotFinish != finishReason {
		t.Fatalf("agent.Turn returned finish = %v, want %v (the script's own FinishReason)", gotFinish, finishReason)
	}

	events := drainSink(t, sink)
	for _, ev := range events {
		if payload, ok := ev.TurnEnd(); ok {
			return payload
		}
	}
	t.Fatalf("no turn_end event observed for finish reason %v", finishReason)
	return agent.TurnEnd{}
}

// dispatchVocabulary is the hand-written table R-ATT-003's exhaustiveness
// pin (S-ATT-004) walks 0..255 against, using ai.FinishReason.Validate()
// as the membership oracle — finish_reason_test.go:277-320's idiom,
// extended to the agent-loop layer (the third instance, per R-ATT-003).
// Also reused by S-ATT-003 as the expected per-reason mapping (D4).
var dispatchVocabulary = map[ai.FinishReason]agent.TurnOutcome{
	ai.FinishReasonStop:          agent.TurnOutcomeFinished,
	ai.FinishReasonLength:        agent.TurnOutcomeLengthLimited,
	ai.FinishReasonToolCalls:     agent.TurnOutcomeToolCalls,
	ai.FinishReasonContentFilter: agent.TurnOutcomeContentFiltered,
	ai.FinishReasonRefusal:       agent.TurnOutcomeRefused,
	ai.FinishReasonPauseTurn:     agent.TurnOutcomePaused,
	ai.FinishReasonUnknown:       agent.TurnOutcomeUnknown,
}

// S-ATT-003 — each finish reason produces its own turn_end. Scripts one
// turn per dispatchVocabulary entry, asserts the observed outcome
// matches D4's table, and asserts the seven observed outcomes are
// pairwise distinct.
func TestTurn_FinishReasonDispatch_EachProducesDistinctOutcome(t *testing.T) {
	seenOutcomes := map[agent.TurnOutcome]ai.FinishReason{}
	for finish, wantOutcome := range dispatchVocabulary {
		finish, wantOutcome := finish, wantOutcome
		t.Run(finish.String(), func(t *testing.T) {
			turnEnd := runTerminationTurn(t, finish)
			if got := turnEnd.Outcome(); got != wantOutcome {
				t.Errorf("turn_end.Outcome() = %v, want %v (finish reason %v, D4's dispatch table)", got, wantOutcome, finish)
			}
		})
		if prior, dup := seenOutcomes[wantOutcome]; dup {
			t.Fatalf("dispatchVocabulary maps both %v and %v to the same outcome %v — the table itself is not one-to-one", finish, prior, wantOutcome)
		}
		seenOutcomes[wantOutcome] = finish
	}
	if len(seenOutcomes) != 7 {
		t.Errorf("observed %d distinct outcomes across the vocabulary, want 7 (one per ai.FinishReason member)", len(seenOutcomes))
	}
}

// S-ATT-012 — required observable, belonging to R-ATT-002: a provider
// stream that closes without any ai.Completion event still produces a
// turn_end, a member outcome, and the documented ai.FinishReasonStop
// fallback (design D2 — normalization moves ahead of finalize()).
func TestTurn_NoCompletionPath(t *testing.T) {
	t.Parallel()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta, err := ai.NewTextDelta(1, "no-completion-fixture")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		// deliberately no Completion — the provider just closes.
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 16)

	_, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for no-completion path",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("agent.Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v (the documented no-completion fallback)", finish, ai.FinishReasonStop)
	}

	events := drainSink(t, sink)
	var turnEnd agent.TurnEnd
	var found bool
	for _, ev := range events {
		if payload, ok := ev.TurnEnd(); ok {
			turnEnd, found = payload, true
		}
	}
	if !found {
		t.Fatal("no turn_end event observed on the no-completion path")
	}
	if turnEnd.Outcome() != dispatchVocabulary[ai.FinishReasonStop] {
		t.Errorf("turn_end.Outcome() = %v, want %v (FinishReasonStop's mapped outcome)", turnEnd.Outcome(), dispatchVocabulary[ai.FinishReasonStop])
	}
}

// S-ATT-004 — bidirectional agent-level exhaustiveness pin (R-ATT-003,
// the third instance of the idiom). outcomeForFinish is unexported, so
// this external-package pin is membership-walk + behavioral (the
// finish_reason_test.go:277-320 idiom, extended): (1) walk
// ai.FinishReason(0..255), membership by Validate() — every validating
// candidate must be named in dispatchVocabulary and the counts must
// match (an eighth upstream member fails here without touching
// unexported bounds); (2) every name in dispatchVocabulary that Layer 1
// does not validate is reported as a failure (the reverse direction);
// (3) for each validating member, actually DISPATCH it through the
// loop's own Turn (not just the test's own table) and assert the
// observed turn_end.Outcome() is member-valued and matches the table —
// proving the loop's dispatch, not merely this file's fixture.
func TestTurn_ExhaustivenessPin(t *testing.T) {
	var validating int
	for v := 0; v <= 255; v++ {
		candidate := ai.FinishReason(v)
		if err := candidate.Validate(); err != nil {
			continue
		}
		validating++
		if _, named := dispatchVocabulary[candidate]; !named {
			t.Errorf("ai.FinishReason(%d) (%v) validates but is absent from dispatchVocabulary — an eighth finish reason the dispatch does not handle", v, candidate)
		}
	}
	if validating != len(dispatchVocabulary) {
		t.Errorf("ai.FinishReason.Validate() admitted %d candidate(s), dispatchVocabulary names %d — the pin is not closed in both directions", validating, len(dispatchVocabulary))
	}
	for finish := range dispatchVocabulary {
		if err := finish.Validate(); err != nil {
			t.Errorf("dispatchVocabulary names %v, but ai.FinishReason.Validate() rejects it: %v", finish, err)
		}
	}

	// Behavioral half: dispatch every member through the real loop and
	// confirm the loop itself — not just this file's table — produces a
	// member-valued outcome equal to the table's entry.
	for finish, wantOutcome := range dispatchVocabulary {
		finish, wantOutcome := finish, wantOutcome
		t.Run("dispatch/"+finish.String(), func(t *testing.T) {
			turnEnd := runTerminationTurn(t, finish)
			got := turnEnd.Outcome()
			if got == 0 {
				t.Fatalf("turn_end.Outcome() = %v (zero value), want a member-valued outcome for finish reason %v", got, finish)
			}
			if got != wantOutcome {
				t.Errorf("turn_end.Outcome() = %v, want %v (finish reason %v) — the loop's own dispatch, not just the test's table", got, wantOutcome, finish)
			}
		})
	}
}

// S-ATT-005 — refusal, pause and finished are three values, distinguishable
// by TurnOutcome alone, with no assertion reading an ai.FinishReason to
// tell them apart.
func TestTurn_RefusalPauseFinished(t *testing.T) {
	t.Parallel()

	refused := runTerminationTurn(t, ai.FinishReasonRefusal)
	paused := runTerminationTurn(t, ai.FinishReasonPauseTurn)
	finished := runTerminationTurn(t, ai.FinishReasonStop)

	if refused.Outcome() == paused.Outcome() {
		t.Errorf("refused.Outcome() == paused.Outcome() == %v, want pairwise distinct", refused.Outcome())
	}
	if refused.Outcome() == finished.Outcome() {
		t.Errorf("refused.Outcome() == finished.Outcome() == %v, want pairwise distinct", refused.Outcome())
	}
	if paused.Outcome() == finished.Outcome() {
		t.Errorf("paused.Outcome() == finished.Outcome() == %v, want pairwise distinct", paused.Outcome())
	}
}

// S-ATT-006 — pause replays received content verbatim: interleaved
// reasoning + text deltas, a non-empty reasoning round-trip token, and a
// FinishReasonPauseTurn completion must reach the caller byte-for-byte,
// with the outcome reported as TurnOutcomePaused — not refused, not
// finished.
func TestTurn_PauseReplaysVerbatim(t *testing.T) {
	t.Parallel()

	const reasoningText = "pause-reasoning-fragment"
	const textFragment = "pause-text-fragment"
	token := []byte("pause-round-trip-token")

	reasonStart, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart returned %v, want no failure", err)
	}
	reasonStartPayload, ok := reasonStart.ReasoningBlockStart()
	if !ok {
		t.Fatal("reasoning block start event carries no ReasoningBlockStart payload")
	}
	reasonDelta, err := ai.NewReasoningDelta(reasonStartPayload, []byte(reasoningText))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	reasonEnd, err := ai.NewReasoningBlockEnd(reasonStartPayload, token)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}
	textStart, err := ai.NewTextBlockStart(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	textDelta, err := ai.NewTextDelta(2, textFragment)
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	textEnd, err := ai.NewTextBlockEnd(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonPauseTurn, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(reasonStart),
		agenttest.Emit(reasonDelta),
		agenttest.Emit(reasonEnd),
		agenttest.Emit(textStart),
		agenttest.Emit(textDelta),
		agenttest.Emit(textEnd),
		agenttest.Emit(completion),
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 16)

	msg, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for pause replay",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("agent.Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonPauseTurn {
		t.Fatalf("finish = %v, want %v", finish, ai.FinishReasonPauseTurn)
	}

	if len(msg.Content()) == 0 {
		t.Fatal("msg has no content, want reasoning + text parts replayed verbatim")
	}
	reasoningPart, ok := msg.Content()[0].Reasoning()
	if !ok {
		t.Fatalf("msg.Content()[0] is not a reasoning part, want the replayed reasoning bracket")
	}
	if reasoningPart.Text() != reasoningText {
		t.Errorf("reasoning text = %q, want %q (byte-for-byte)", reasoningPart.Text(), reasoningText)
	}
	gotToken, hasToken := reasoningPart.Token()
	if !hasToken || string(gotToken) != string(token) {
		t.Errorf("reasoning token = (%q, %v), want (%q, true) — byte-exact round-trip", gotToken, hasToken, token)
	}
	if len(msg.Content()) < 2 {
		t.Fatal("msg has no text part, want the replayed text bracket")
	}
	textPart, ok := msg.Content()[1].Text()
	if !ok || textPart != textFragment {
		t.Errorf("text content = (%q, %v), want (%q, true) — byte-for-byte", textPart, ok, textFragment)
	}

	events := drainSink(t, sink)
	var turnEnd agent.TurnEnd
	var found bool
	for _, ev := range events {
		if payload, ok := ev.TurnEnd(); ok {
			turnEnd, found = payload, true
		}
	}
	if !found {
		t.Fatal("no turn_end event observed")
	}
	if turnEnd.Outcome() != agent.TurnOutcomePaused {
		t.Errorf("turn_end.Outcome() = %v, want TurnOutcomePaused (not Refused, not Finished)", turnEnd.Outcome())
	}
}
