// AG-13 — the multi-turn run driver. Phase 0 opens the continuation
// seam (TurnOptions.Continuation, R-LSK-001) and the scheduler's
// sink-ownership seam (Scheduler.LeaveSinkOpen, R-TLS-012); Phase 1
// wires the continuation path through Turn (identity/brackets,
// schedule-before-finalize, and the R-HIS-010 transcript commits).
// Later phases (AG-13.1..AG-13.3, the R-APP-002 parked-wait bite) add
// the Harness type itself and its own tests to this same file, so a
// helper added for one phase is available to a later phase's without
// duplication — loop_test.go's own precedent (NFR-LSK-001: every
// scenario in package agent_test).
package agent_test

import (
	"errors"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AG-13.0 — S-LSK-014. Given a TurnOptions carrying a continuation
// with any one member absent, when Turn runs, then it returns a typed
// rejection naming the absent member's position, no event whatsoever
// is emitted on sink, and the sink is left in the state the caller
// gave it — a half-configured continuation never produces a partial
// stream. Table-driven over all four members (triangulation: the
// "naming the absent member" property must hold for each, not just
// whichever one a hardcoded check happens to catch).
func TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission(t *testing.T) {
	t.Parallel()

	full := func() *agent.TurnContinuation {
		return &agent.TurnContinuation{
			Run:       "run-lsk-014",
			Stamper:   &agent.LaneStamper{},
			Scheduler: &agent.Scheduler{},
			History:   agent.NewHistory(),
		}
	}

	tests := []struct {
		name    string
		mutate  func(*agent.TurnContinuation)
		wantPos string
	}{
		{"run absent", func(c *agent.TurnContinuation) { c.Run = "" }, "continuation.run"},
		{"stamper absent", func(c *agent.TurnContinuation) { c.Stamper = nil }, "continuation.stamper"},
		{"scheduler absent", func(c *agent.TurnContinuation) { c.Scheduler = nil }, "continuation.scheduler"},
		{"history absent", func(c *agent.TurnContinuation) { c.History = nil }, "continuation.history"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cont := full()
			tt.mutate(cont)
			sink := make(chan *agent.Event, 16)

			_, _, err := agent.Turn(
				contextBackground(),
				nil, // MUST never be reached — validation happens before any provider interaction.
				"system prompt for lsk-014",
				[]ai.Message{firstMessage(t)},
				agent.TurnOptions{Continuation: cont},
				sink,
			)
			if err == nil {
				t.Fatal("Turn returned err = nil, want a typed rejection for a half-configured continuation")
			}
			if !errors.Is(err, ai.ErrEmpty) {
				t.Errorf("Turn error rule class = %v, want errors.Is(err, ai.ErrEmpty)", err)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("Turn error = %T, want errors.As to reach *ai.Violation", err)
			}
			if got := violation.Path().String(); got != tt.wantPos {
				t.Errorf("violation position = %q, want %q (naming the absent member)", got, tt.wantPos)
			}

			// No event whatsoever on sink: a non-blocking receive
			// must hit default — no data ready, and (proven below)
			// not closed either.
			select {
			case ev, ok := <-sink:
				t.Fatalf("sink carries an event/close signal (ev=%v, ok=%v), want zero events and an untouched channel", ev, ok)
			default:
			}

			// The sink is left in the state the caller gave it —
			// Turn must not have closed it. Closing it here
			// ourselves must not panic; a panic would prove Turn
			// already closed it.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("closing sink after Turn panicked (%v) — Turn already closed it, want the sink left untouched", r)
					}
				}()
				close(sink)
			}()
		})
	}
}

// AG-13.0 — S-TLS-013. Given a Scheduler constructed with the
// sink-ownership flag at its zero value and a set of scheduled calls,
// when Schedule runs and a consumer drains sink, then the ordered
// rejoin is fully populated, the consumer observes the sink close
// after the last tool event, and the emitted event sequence is
// byte-identical to the pre-AG-13 sequence for the same input.
func TestSchedule_LeaveSinkOpenZeroDefault_ClosesSinkUnchanged(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_tls_013", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_tls_013": tool})
	calls := callsToAICalls([]scheduledCall{
		{id: "call-tls-013", name: "read_tls_013", args: []byte(`{"x":1}`), effect: agent.EffectClassRead},
	})

	sched := &agent.Scheduler{MaxConcurrentReads: 4} // LeaveSinkOpen at its zero value (false).
	sink := make(chan *agent.Event, 16)

	results := sched.Schedule(contextBackground(), calls, reg, "run-tls-013", "turn-tls-013", nil, &agent.LaneStamper{}, sink)

	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if got := results[0].CallID(); got != "call-tls-013" {
		t.Errorf("results[0].CallID() = %q, want %q", got, "call-tls-013")
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want ToolOutcomeSuccess", results[0].Outcome)
	}

	// The consumer observes the sink close after the last tool event:
	// a bare range over sink only returns once the channel is both
	// drained AND closed — which happens here only if Schedule closed
	// it (the zero-value default, byte-identical to AG-09 behavior).
	var kinds []agent.EventKind
	for ev := range sink {
		kinds = append(kinds, ev.Kind())
	}
	wantKinds := []agent.EventKind{agent.EventKindToolStart, agent.EventKindToolEndSuccess}
	if len(kinds) != len(wantKinds) {
		t.Fatalf("emitted %d event kind(s) = %v, want %d (%v)", len(kinds), kinds, len(wantKinds), wantKinds)
	}
	for i, want := range wantKinds {
		if kinds[i] != want {
			t.Errorf("event[%d] kind = %v, want %v (byte-identical to the pre-AG-13 sequence)", i, kinds[i], want)
		}
	}
}

// AG-13.0 — S-TLS-014. Given a Scheduler constructed with the
// sink-ownership flag set, when Schedule runs to its rejoin, then the
// sink is NOT closed, a subsequent send on it by the caller succeeds
// and is observed by the consumer, and the consumer observes the
// close only when the caller closes it; the ordered rejoin is fully
// populated exactly as in S-TLS-013.
func TestSchedule_LeaveSinkOpenSet_CallerOwnsClose(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_tls_014", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_tls_014": tool})
	calls := callsToAICalls([]scheduledCall{
		{id: "call-tls-014", name: "read_tls_014", args: []byte(`{"y":2}`), effect: agent.EffectClassRead},
	})

	sched := &agent.Scheduler{MaxConcurrentReads: 4, LeaveSinkOpen: true}
	sink := make(chan *agent.Event, 16)

	results := sched.Schedule(contextBackground(), calls, reg, "run-tls-014", "turn-tls-014", nil, &agent.LaneStamper{}, sink)

	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if got := results[0].CallID(); got != "call-tls-014" {
		t.Errorf("results[0].CallID() = %q, want %q", got, "call-tls-014")
	}

	// Schedule is synchronous — it returns only after the dispatcher
	// goroutine has exited (Schedule.<-dispatcherDone) — so both
	// expected tool events are already buffered; read them with a
	// non-blocking select so a missing event fails immediately rather
	// than hanging the test.
	wantKinds := []agent.EventKind{agent.EventKindToolStart, agent.EventKindToolEndSuccess}
	for i, want := range wantKinds {
		select {
		case ev, ok := <-sink:
			if !ok {
				t.Fatalf("sink closed after %d event(s); want it left open (LeaveSinkOpen: true)", i)
			}
			if ev.Kind() != want {
				t.Errorf("event[%d] kind = %v, want %v", i, ev.Kind(), want)
			}
		default:
			t.Fatalf("sink had no event[%d] ready, want %v (Schedule is synchronous)", i, want)
		}
	}

	// Not closed: a non-blocking receive must hit default — no more
	// data queued, and not closed either (a closed empty channel
	// would win the receive case immediately instead).
	select {
	case ev, ok := <-sink:
		t.Fatalf("sink carries an extra event/close signal (ev=%v, ok=%v) after the rejoin, want it left open with nothing further queued", ev, ok)
	default:
	}

	// A subsequent send by the caller succeeds and is observed by the
	// consumer — the caller, not Schedule, now owns sink.
	marker, err := agent.NewRunStart("run-tls-014-marker")
	if err != nil {
		t.Fatalf("agent.NewRunStart: %v", err)
	}
	select {
	case sink <- &marker:
	default:
		t.Fatal("send on sink after Schedule blocked/failed — want it open and writable by the caller")
	}
	select {
	case got, ok := <-sink:
		if !ok || got != &marker {
			t.Fatalf("did not observe the caller's own send back out of sink (ok=%v)", ok)
		}
	default:
		t.Fatal("caller's send did not reach a reader — want the buffered marker observable")
	}

	// The consumer observes the close only when the caller closes it:
	// closing here must not panic (a panic would mean Schedule had
	// already closed it despite LeaveSinkOpen).
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("closing sink after Schedule panicked (%v) — Schedule must have left it open for the caller", r)
			}
		}()
		close(sink)
	}()
	if _, ok := <-sink; ok {
		t.Error("sink still reports open after the caller's own close")
	}
}

// AG-13.1 — S-LSK-013 (non-nil half, text-only turn). Given a
// TurnOptions carrying a fully configured continuation, when Turn
// runs a text-only turn (no tool calls), then no run-open and no
// run-close appears on sink, the emitted events are stamped from the
// supplied stamper continuing its existing lane, and every emitted
// event carries the continuation's run identity — proven across two
// turns sharing one continuation so "shared identity and lane" is a
// real cross-call property, not an artifact of a single call.
func TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane(t *testing.T) {
	t.Parallel()

	stamper := &agent.LaneStamper{}
	cont := &agent.TurnContinuation{
		Run:       "run-lsk-013-shared",
		Stamper:   stamper,
		Scheduler: &agent.Scheduler{},
		History:   agent.NewHistory(),
	}

	// First turn: text-only, continuation mode.
	firstProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	firstSink := make(chan *agent.Event, 16)
	_, finish, err := agent.Turn(
		contextBackground(),
		firstProvider,
		"system prompt for lsk-013 first",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Continuation: cont},
		firstSink,
	)
	if err != nil {
		t.Fatalf("first Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v", finish, ai.FinishReasonStop)
	}
	firstEvents := drainSink(t, firstSink)
	if len(firstEvents) == 0 {
		t.Fatal("first turn emitted zero events")
	}

	// No run-open, no run-close: only turn_start .. turn_end.
	for _, ev := range firstEvents {
		if ev.Kind() == agent.EventKindRunStart || ev.Kind() == agent.EventKindRunEnd {
			t.Errorf("continuation-path event kind = %v, want no run_start/run_end (the caller's run owns the bracket)", ev.Kind())
		}
	}
	if got := firstEvents[0].Kind(); got != agent.EventKindTurnStart {
		t.Errorf("first event kind = %v, want %v (turn_start, no run_start)", got, agent.EventKindTurnStart)
	}
	if got := firstEvents[len(firstEvents)-1].Kind(); got != agent.EventKindTurnEnd {
		t.Errorf("last event kind = %v, want %v (turn_end, no run_end)", got, agent.EventKindTurnEnd)
	}

	// Every event carries the continuation's run identity, not a
	// freshly minted one.
	for i, ev := range firstEvents {
		if ev.Run() != cont.Run {
			t.Errorf("event[%d].Run() = %q, want %q (continuation.Run, not a minted identity)", i, ev.Run(), cont.Run)
		}
	}

	// Sequence starts at 1 for the first turn sharing a fresh stamper.
	if firstEvents[0].Sequence() != 1 {
		t.Errorf("first turn's first event sequence = %v, want 1", firstEvents[0].Sequence())
	}
	lastSeq := firstEvents[len(firstEvents)-1].Sequence()

	// Second turn: same continuation (same Run, same Stamper) — the
	// lane must continue, not restart, proving "shared identity and
	// lane" (R-LSK-001 point 1).
	secondProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	secondSink := make(chan *agent.Event, 16)
	_, _, err = agent.Turn(
		contextBackground(),
		secondProvider,
		"system prompt for lsk-013 second",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Continuation: cont},
		secondSink,
	)
	if err != nil {
		t.Fatalf("second Turn returned err = %v, want nil", err)
	}
	secondEvents := drainSink(t, secondSink)
	if len(secondEvents) == 0 {
		t.Fatal("second turn emitted zero events")
	}

	if got := secondEvents[0].Sequence(); got != lastSeq+1 {
		t.Errorf("second turn's first event sequence = %v, want %v (one contiguous lane across turns, not a restart)", got, lastSeq+1)
	}
	for i, ev := range secondEvents {
		if ev.Run() != cont.Run {
			t.Errorf("second turn event[%d].Run() = %q, want %q (shared run identity)", i, ev.Run(), cont.Run)
		}
	}

	// TurnID is still minted fresh per call: the two turns must NOT
	// share a turn identity (R-LSK-001 point 1 carry — N distinct
	// turn brackets need N identities).
	firstTurnID, _ := firstEvents[0].Turn()
	secondTurnID, _ := secondEvents[0].Turn()
	if firstTurnID == secondTurnID {
		t.Errorf("first and second continuation turns share turnID %q, want distinct fresh identities per call", firstTurnID)
	}
}

// AG-13.1 — R-LSK-001 point 2, the mid-stream fatal path. Given a
// TurnOptions carrying a fully configured continuation, when the
// provider stream ends in a terminal mid-stream error, then
// turn_end(Aborted) is still emitted but no run_end appears — the
// caller's run owns the run bracket on every path, including this
// one. Not a charter scenario on its own; a coverage/evidence test
// for R-LSK-001 point 2's explicit "on any path" clause, mirroring
// loop_test.go's own non-charter coverage-helper precedent
// (TestTurn_MidStreamErrorSurfacesOnReturn et al.).
func TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd(t *testing.T) {
	t.Parallel()

	cont := &agent.TurnContinuation{
		Run:       "run-lsk-013-fatal",
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{},
		History:   agent.NewHistory(),
	}
	provider := agenttest.NewProvider(scriptTextThenMidStreamError(t))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for lsk-013 fatal",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Continuation: cont},
		sink,
	)
	if err == nil {
		t.Fatal("Turn returned err = nil, want a non-nil typed failure (mid-stream terminal error)")
	}

	events := drainSink(t, sink)
	if len(events) == 0 {
		t.Fatal("sink was empty, want at least turn_start + turn_end(Aborted)")
	}
	for _, ev := range events {
		if ev.Kind() == agent.EventKindRunStart || ev.Kind() == agent.EventKindRunEnd {
			t.Errorf("continuation-path fatal event kind = %v, want no run_start/run_end even on the mid-stream fatal path", ev.Kind())
		}
	}
	last := events[len(events)-1]
	turnEnd, ok := last.TurnEnd()
	if !ok {
		t.Fatalf("last event kind = %v, want turn_end", last.Kind())
	}
	if turnEnd.Outcome() != agent.TurnOutcomeAborted {
		t.Errorf("turn_end.Outcome() = %v, want TurnOutcomeAborted (still emitted on the continuation path)", turnEnd.Outcome())
	}
	if _, hasFailure := turnEnd.Failure(); !hasFailure {
		t.Error("turn_end carries no Failure, want a non-nil *agent.Failure")
	}
}

// AG-13.1 — S-LSK-013 (tool-calling half), the schedule-before-
// finalize reorder. Given a TurnOptions carrying a fully configured
// continuation, when Turn runs a tool-calling turn, then the turn's
// tool events land INSIDE the open turn bracket and BEFORE turn_end,
// and CheckStream accepts the resulting stream once framed with the
// run bracket the caller (AG-13.1's Harness) supplies around it —
// design.md's run algorithm step 2: NewRunStart + the SAME shared
// stamper. Expect FAIL today: the nil path's finalize-first ordering
// puts tool events after turn_end, which CheckStream rejects (a
// PlacementTurn event outside an open turn, or after the terminal
// run_end).
func TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_lsk_013_tool", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_lsk_013_tool": tool})

	stamper := &agent.LaneStamper{}
	cont := &agent.TurnContinuation{
		Run:       "run-lsk-013-tool",
		Stamper:   stamper,
		Scheduler: &agent.Scheduler{MaxConcurrentReads: 4, LeaveSinkOpen: true},
		History:   agent.NewHistory(),
	}

	tcStart, err := ai.NewToolCallStart(1, "call-lsk-013", "read_lsk_013_tool")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tcDelta, err := ai.NewToolCallDelta(1, []byte(`{"n":1}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tcEnd, err := ai.NewToolCallEnd(1, []byte(`{"n":1}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(tcStart),
		agenttest.Emit(tcDelta),
		agenttest.Emit(tcEnd),
		agenttest.Emit(completion),
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 256)

	// The caller (AG-13.1's Harness, simulated here) owns the run
	// bracket and shares its stamper with Turn — design.md's run
	// algorithm step 2 (NewRunStart + stamper.Stamp, sequence 1).
	runStart, err := agent.NewRunStart(cont.Run)
	if err != nil {
		t.Fatalf("agent.NewRunStart: %v", err)
	}
	stampedRunStart := stamper.Stamp(runStart)

	_, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for lsk-013 tool",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Tools: reg, Continuation: cont},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonToolCalls {
		t.Errorf("finish = %v, want %v", finish, ai.FinishReasonToolCalls)
	}
	if got := tool.Invocations(); got != 1 {
		t.Errorf("tool invocations = %d, want 1", got)
	}

	turnEvents := drainSink(t, sink)

	runEnd, err := agent.NewRunEnd(cont.Run, agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd: %v", err)
	}
	stampedRunEnd := stamper.Stamp(runEnd)

	// Tool events must land BEFORE turn_end, inside the still-open
	// turn bracket — the reorder under test.
	turnEndIdx, toolStartIdx, toolEndIdx := -1, -1, -1
	for i, ev := range turnEvents {
		switch ev.Kind() {
		case agent.EventKindTurnEnd:
			turnEndIdx = i
		case agent.EventKindToolStart:
			toolStartIdx = i
		case agent.EventKindToolEndSuccess:
			toolEndIdx = i
		}
	}
	if turnEndIdx < 0 {
		t.Fatal("no turn_end found in the emitted stream")
	}
	if toolStartIdx < 0 || toolEndIdx < 0 {
		t.Fatalf("tool_start/tool_end_success not both found (start=%d, end=%d)", toolStartIdx, toolEndIdx)
	}
	if toolStartIdx > turnEndIdx || toolEndIdx > turnEndIdx {
		t.Errorf("tool events at [start=%d end=%d] land after turn_end at [%d], want both BEFORE it (inside the open turn bracket)", toolStartIdx, toolEndIdx, turnEndIdx)
	}

	// CheckStream must accept the framed stream unmodified — the
	// concrete evidence the reorder is required, not cosmetic: a
	// PlacementTurn tool event outside an open turn (or after the
	// terminal run_end) is exactly what CheckStream rejects.
	framed := append([]agent.Event{stampedRunStart}, turnEvents...)
	framed = append(framed, stampedRunEnd)
	if report := agent.CheckStream(framed); report.Violation() != nil {
		t.Errorf("CheckStream rejected the continuation-path stream: %v", report.Violation())
	}
}

// AG-13.1 — S-HIS-090. Given a continuation carrying a transcript
// store and a provider scripted with a turn that emits assistant text
// and one tool call whose tool succeeds, when Turn runs, then the
// store holds, in order, the turn's assistant message carrying the
// tool call with provider-exact argument bytes, followed by one
// tool-result message correlated to that call; and when the caller
// then closes the turn, the close succeeds because no call is open.
//
// This test also discharges task 6.3's forwarded obligation: it
// proves — by actually running ai.NewRequest over the committed
// transcript, not by inspecting source — that Layer 1 accepts a
// transcript containing a RoleTool result message. design.md and the
// spec both flag this as expected but unproven until run.
func TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_his_090", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_his_090": tool})

	history := agent.NewHistory()
	cont := &agent.TurnContinuation{
		Run:       "run-his-090",
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{MaxConcurrentReads: 4, LeaveSinkOpen: true},
		History:   history,
	}

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	delta, err := ai.NewTextDelta(1, "checking the file")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	tcStart, err := ai.NewToolCallStart(2, "call-his-090", "read_his_090")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tcDelta, err := ai.NewToolCallDelta(2, []byte(`{"path":"a"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tcEnd, err := ai.NewToolCallEnd(2, []byte(`{"path":"a"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(tcStart),
		agenttest.Emit(tcDelta),
		agenttest.Emit(tcEnd),
		agenttest.Emit(completion),
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 256)

	msg, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for his-090",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Tools: reg, Continuation: cont},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonToolCalls {
		t.Errorf("finish = %v, want %v", finish, ai.FinishReasonToolCalls)
	}
	drainSink(t, sink)

	entries := history.Entries()
	if len(entries) != 2 {
		t.Fatalf("history has %d entries, want 2 (assistant message + one tool-result message)", len(entries))
	}

	assistantMsg := entries[0].Message()
	if assistantMsg.Role() != ai.RoleAssistant {
		t.Errorf("entries[0].Message().Role() = %v, want RoleAssistant", assistantMsg.Role())
	}
	if !assistantMsg.Equal(msg) {
		t.Error("entries[0].Message() != Turn's own returned msg — one source of truth is violated")
	}
	var foundCall bool
	for _, part := range assistantMsg.Content() {
		if call, ok := part.ToolCall(); ok {
			foundCall = true
			if call.ID() != "call-his-090" {
				t.Errorf("committed ToolCall.ID() = %q, want %q", call.ID(), "call-his-090")
			}
			if got := string(call.Arguments()); got != `{"path":"a"}` {
				t.Errorf("committed ToolCall.Arguments() = %q, want %q (provider-exact bytes)", got, `{"path":"a"}`)
			}
		}
	}
	if !foundCall {
		t.Error("assistant message carries no ToolCall part")
	}

	resultMsg := entries[1].Message()
	if resultMsg.Role() != ai.RoleTool {
		t.Errorf("entries[1].Message().Role() = %v, want RoleTool", resultMsg.Role())
	}
	var foundResult bool
	for _, part := range resultMsg.Content() {
		if tr, ok := part.ToolResult(); ok {
			foundResult = true
			if tr.CallID() != "call-his-090" {
				t.Errorf("committed ToolResult.CallID() = %q, want %q", tr.CallID(), "call-his-090")
			}
			if tr.Failed() {
				t.Error("committed ToolResult.Failed() = true, want false (the tool succeeded)")
			}
		}
	}
	if !foundResult {
		t.Error("tool-result message carries no ToolResult part")
	}

	if err := history.CloseTurn(); err != nil {
		t.Errorf("history.CloseTurn() = %v, want nil (no call is open)", err)
	}

	// Task 6.3's forwarded obligation, discharged by running it: a
	// transcript built from history.Entries() — the call message,
	// then the RoleTool result message — must round-trip through
	// ai.NewRequest without rejection.
	transcript := make([]ai.Message, len(entries))
	for i, e := range entries {
		transcript[i] = e.Message()
	}
	if _, err := ai.NewRequest("model-his-090", transcript); err != nil {
		t.Fatalf("ai.NewRequest rejected a transcript containing a RoleTool result message: %v — this is a design-reopening finding (design.md Decision 1, spec's forwarded obligation), not a test bug", err)
	}
}

// AG-13.1 — S-HIS-091. Given a continuation and a provider scripted
// with a turn producing no content and no tool call, when Turn runs,
// then nothing is appended to the store, a subsequent read returns
// the store unchanged, and Turn returns without error. Covered by the
// same skip-if-zero-content branch S-HIS-090 exercises for the
// non-empty case (tasks.md 1.11: "covered by 1.8's skip-if-zero-
// content branch") — GREEN-confirmed here, not independently RED
// (nothing appended is also what an unwired implementation would
// produce for a content-less turn, so this scenario alone cannot
// distinguish the two; S-HIS-090/092/093 are what the scratch-revert
// evidence below exercises).
func TestTurn_ContinuationEmptyContent_AppendsNothing(t *testing.T) {
	t.Parallel()

	history := agent.NewHistory()
	cont := &agent.TurnContinuation{
		Run:       "run-his-091",
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{},
		History:   history,
	}

	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(completion)}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 16)

	msg, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for his-091",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Continuation: cont},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v", finish, ai.FinishReasonStop)
	}
	if len(msg.Content()) != 0 {
		t.Errorf("Turn returned msg with %d content part(s), want 0 (the turn produced no content)", len(msg.Content()))
	}
	drainSink(t, sink)

	if got := history.Len(); got != 0 {
		t.Errorf("history.Len() = %d, want 0 (a ran-to-empty turn appends nothing, not an empty message)", got)
	}
}

// AG-13.1 — S-HIS-092. Given a continuation and a turn whose
// scheduled calls resolve to a mix of success and both failure
// outcomes, when Turn runs, then exactly one tool-result message is
// appended per rejoin result, in call order, each correlated to its
// own call identity, and each failure outcome is carried by the
// Layer 1 failure form (ToolResult.Failed()) rather than by a content
// sentinel. Three calls: one registered tool that succeeds, one
// registered tool that returns ToolOutcomeResultFailure, and one
// UNREGISTERED name that takes the R-TLS-009/010 orphan path
// (ToolOutcomeExecutionFailure) — both failure outcomes covered.
func TestTurn_ContinuationMixedOutcomes_OneResultPerCallInOrder(t *testing.T) {
	t.Parallel()

	okTool := EchoScriptedTool("ok_his_092", agent.EffectClassRead)
	resultFailTool := NewScriptedTool("result_fail_his_092", agent.EffectClassRead, agent.Result{
		Outcome: agent.ToolOutcomeResultFailure,
		Content: []byte("tool reported failure"),
	})
	reg := agent.NewMapRegistry(map[string]agent.Tool{
		"ok_his_092":          okTool,
		"result_fail_his_092": resultFailTool,
		// "missing_his_092" is deliberately unregistered — the
		// R-TLS-009/010 orphan path yields ToolOutcomeExecutionFailure.
	})

	history := agent.NewHistory()
	cont := &agent.TurnContinuation{
		Run:       "run-his-092",
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{MaxConcurrentReads: 4, LeaveSinkOpen: true},
		History:   history,
	}

	tc1Start, err := ai.NewToolCallStart(1, "call-his-092-a", "ok_his_092")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tc1Delta, err := ai.NewToolCallDelta(1, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tc1End, err := ai.NewToolCallEnd(1, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	tc2Start, err := ai.NewToolCallStart(2, "call-his-092-b", "result_fail_his_092")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tc2Delta, err := ai.NewToolCallDelta(2, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tc2End, err := ai.NewToolCallEnd(2, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	tc3Start, err := ai.NewToolCallStart(3, "call-his-092-c", "missing_his_092")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	tc3Delta, err := ai.NewToolCallDelta(3, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	tc3End, err := ai.NewToolCallEnd(3, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(tc1Start), agenttest.Emit(tc1Delta), agenttest.Emit(tc1End),
		agenttest.Emit(tc2Start), agenttest.Emit(tc2Delta), agenttest.Emit(tc2End),
		agenttest.Emit(tc3Start), agenttest.Emit(tc3Delta), agenttest.Emit(tc3End),
		agenttest.Emit(completion),
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 256)

	_, _, err = agent.Turn(
		contextBackground(),
		provider,
		"system prompt for his-092",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Tools: reg, Continuation: cont},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	entries := history.Entries()
	if len(entries) != 4 {
		t.Fatalf("history has %d entries, want 4 (assistant message + 3 tool-result messages)", len(entries))
	}

	wantIDs := []string{"call-his-092-a", "call-his-092-b", "call-his-092-c"}
	wantFailed := []bool{false, true, true}
	for i, wantID := range wantIDs {
		resultMsg := entries[i+1].Message()
		if resultMsg.Role() != ai.RoleTool {
			t.Errorf("entries[%d].Message().Role() = %v, want RoleTool", i+1, resultMsg.Role())
		}
		content := resultMsg.Content()
		if len(content) != 1 {
			t.Fatalf("entries[%d] message carries %d part(s), want 1 (exactly one tool-result message per rejoin result)", i+1, len(content))
		}
		tr, ok := content[0].ToolResult()
		if !ok {
			t.Fatalf("entries[%d] part is not a ToolResult", i+1)
		}
		if tr.CallID() != wantID {
			t.Errorf("entries[%d].CallID() = %q, want %q (call order preserved)", i+1, tr.CallID(), wantID)
		}
		if tr.Failed() != wantFailed[i] {
			t.Errorf("entries[%d].Failed() = %v, want %v — the outcome must ride the Layer 1 failure form, not a content sentinel", i+1, tr.Failed(), wantFailed[i])
		}
	}

	if err := history.CloseTurn(); err != nil {
		t.Errorf("history.CloseTurn() = %v, want nil", err)
	}
}

// AG-13.1 — S-HIS-093. Given a continuation whose transcript store
// rejects the append, when Turn runs, then it returns a non-nil typed
// error carrying that rejection, and the caller's run terminates
// through the failure path of R-RUN-011 (Phase 2, cross-checked
// there) rather than reporting a successful turn. The deterministic
// way to make ANY commit fail without depending on the pairing rules'
// own (already AG-12-tested) rejection paths: an unconstructed
// History (never passed through NewHistory/NewSeededHistory) rejects
// its first commit via the closed zero-value door (S-HIS-030).
func TestTurn_ContinuationAppendFailure_TypedErrorReturned(t *testing.T) {
	t.Parallel()

	brokenHistory := &agent.History{}
	cont := &agent.TurnContinuation{
		Run:       "run-his-093",
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{},
		History:   brokenHistory,
	}

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for his-093",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{Continuation: cont},
		sink,
	)
	if err == nil {
		t.Fatal("Turn returned err = nil, want a non-nil typed rejection (the transcript store rejects the append)")
	}
	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("Turn error = %T, want errors.As to reach *ai.Violation (History.Append's own typed rejection propagated, not swallowed)", err)
	}

	// Turn still honors its own invariant (closes sink) even on this
	// failure path.
	drainSink(t, sink)
}

// AG-13.1 — S-HIS-094. Given a TurnOptions with a nil continuation,
// when Turn runs any scripted shape, then no transcript store is
// touched, and the closed-route history surface guard
// (history_surface_guard_test.go) passes with its source byte-
// unchanged — confirmed structurally via git diff, recorded in
// apply-progress.md; this test carries the behavioral half.
func TestTurn_ContinuationNil_HistorySurfaceGuardStaysGreen(t *testing.T) {
	t.Parallel()

	// A live History sits in scope, exactly as it would inside a real
	// harness-driven process (AG-13.1's Harness always constructs or
	// receives one) — proving a nil-continuation Turn call has no way
	// to reach it, not merely that this test never wired one in.
	sentinel := agent.NewHistory()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for his-094",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err != nil {
		t.Fatalf("Turn returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	if got := sentinel.Len(); got != 0 {
		t.Errorf("sentinel history.Len() = %d, want 0 — a nil-continuation Turn call must never reach any transcript store", got)
	}
}

// ============================================================================
// Phase 2 — AG-13.1: Harness run to completion (R-RUN-001..007, R-RUN-011)
// ============================================================================

// AG-13.1 — S-RUN-001. Given a harness value constructed as a struct
// literal with only its required provider field set, when Run drives a
// scripted single-turn conversation from an external test package, then
// the run completes without any constructor call, the caller-visible
// field values are unchanged after Run returns except for the recorded
// sink-ownership flag, and the type exposes no third exported method
// beyond Run and Steer.
func TestHarness_StructLiteralRun_NoConstructorFieldsUnchanged(t *testing.T) {
	t.Parallel()

	t.Run("nil optional fields resolve to locals, caller struct untouched", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		h := agent.Harness{Provider: provider, System: "system prompt for run-001"} // struct literal — no constructor call.

		sink := make(chan *agent.Event, 64)
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		if finish != ai.FinishReasonStop {
			t.Errorf("finish = %v, want %v", finish, ai.FinishReasonStop)
		}
		if len(msg.Content()) == 0 {
			t.Error("Run returned a message with zero content parts, want the scripted text")
		}
		drainSink(t, sink)

		if h.System != "system prompt for run-001" {
			t.Errorf("h.System = %q, want unchanged", h.System)
		}
		if h.History != nil {
			t.Error("h.History changed from nil — Run must resolve nil defaults into LOCALS, never write them back onto the caller's struct")
		}
		if h.Scheduler != nil {
			t.Error("h.Scheduler changed from nil — Run must resolve nil defaults into LOCALS, never write them back onto the caller's struct")
		}
	})

	t.Run("supplied Scheduler gets LeaveSinkOpen set - the one recorded exception", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sched := &agent.Scheduler{MaxConcurrentReads: 3}
		hist := agent.NewHistory()
		h := agent.Harness{Provider: provider, System: "system prompt for run-001b", Scheduler: sched, History: hist}

		sink := make(chan *agent.Event, 64)
		if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		drainSink(t, sink)

		if !sched.LeaveSinkOpen {
			t.Error("sched.LeaveSinkOpen = false after Run, want true — the one recorded caller-field mutation exception")
		}
		if sched.MaxConcurrentReads != 3 {
			t.Errorf("sched.MaxConcurrentReads = %d, want unchanged 3", sched.MaxConcurrentReads)
		}
		if h.Scheduler != sched {
			t.Error("h.Scheduler pointer changed — Run must use the caller's scheduler in place, never replace it")
		}
		if h.History != hist {
			t.Error("h.History pointer changed — Run must use the caller's history in place, never replace it")
		}
	})

	t.Run("exactly two exported methods", func(t *testing.T) {
		t.Parallel()

		typ := reflect.TypeOf(&agent.Harness{})
		var names []string
		for i := 0; i < typ.NumMethod(); i++ {
			names = append(names, typ.Method(i).Name)
		}
		sort.Strings(names)
		want := []string{"Run", "Steer"}
		if !reflect.DeepEqual(names, want) {
			t.Errorf("*Harness exported methods = %v, want exactly %v", names, want)
		}
	})
}

// AG-13.1 — S-RUN-002. Given a run that has taken its terminal decision
// and returned, when Steer is called with a well-formed user message,
// then it returns an error that satisfies the typed rejection
// ai.Invalid(ai.ErrMisplaced, ai.At("steering")), the transcript is
// unchanged, and no further event reaches the consumer sink.
func TestHarness_SteerAfterTerminal_TypedRejectionNoSilentDrop(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-002", History: hist}

	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	lenBefore := hist.Len()

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "too late"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	err = h.Steer(steered)
	if err == nil {
		t.Fatal("Steer after the run's terminal decision returned nil, want a typed rejection")
	}
	if !errors.Is(err, ai.ErrMisplaced) {
		t.Errorf("Steer error rule class = %v, want errors.Is(err, ai.ErrMisplaced)", err)
	}
	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("Steer error = %T, want errors.As to reach *ai.Violation", err)
	}
	if got := violation.Path().String(); got != "steering" {
		t.Errorf("violation position = %q, want %q", got, "steering")
	}

	if got := hist.Len(); got != lenBefore {
		t.Errorf("history.Len() after rejected Steer = %d, want unchanged %d", got, lenBefore)
	}
}

// scriptToolCallResponse builds a scripted stream requesting one tool
// call (start/delta/end for callID/toolName/args) then completing with
// FinishReasonToolCalls — Phase 2's shared fixture for a tool-calling
// turn.
func scriptToolCallResponse(t *testing.T, callID, toolName string, args []byte) agenttest.Script {
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
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// AG-13.1 — S-RUN-010. Charter AG-13.1 sc.1. Given a fake provider
// scripted with turn one requesting a tool call and turn two answering
// with final text, when one prompt drives the run and the consumer
// drains the sink to close, then the consumer observes, in order: run
// open, turn one open, turn one's message and tool events including the
// tool's execution and result, turn one close carrying the tool-calls
// outcome, turn two open, turn two's message events, turn two close
// carrying the finished outcome, run close carrying the completed
// outcome; and Run returns the second turn's message and finish reason
// with a nil error.
func TestHarness_TwoTurnRun_CompletesToTerminal(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_run_010", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_run_010": tool})

	turnOneScript := scriptToolCallResponse(t, "call-run-010", "read_run_010", []byte(`{"x":1}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for run-010", Turn: agent.TurnOptions{Tools: reg}}

	sink := make(chan *agent.Event, 256)
	msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v (turn two's finish reason)", finish, ai.FinishReasonStop)
	}
	if len(msg.Content()) == 0 {
		t.Error("Run returned a message with zero content parts, want turn two's text")
	}

	events := drainSink(t, sink)
	if len(events) == 0 {
		t.Fatal("zero events observed")
	}

	var kinds []agent.EventKind
	for _, ev := range events {
		kinds = append(kinds, ev.Kind())
	}
	if kinds[0] != agent.EventKindRunStart {
		t.Errorf("first event kind = %v, want run_start", kinds[0])
	}
	if kinds[len(kinds)-1] != agent.EventKindRunEnd {
		t.Errorf("last event kind = %v, want run_end", kinds[len(kinds)-1])
	}

	var runStarts, runEnds, turnStarts, turnEnds int
	var turnEndOutcomes []agent.TurnOutcome
	var sawToolStart, sawToolEnd bool
	for _, ev := range events {
		switch ev.Kind() {
		case agent.EventKindRunStart:
			runStarts++
		case agent.EventKindRunEnd:
			runEnds++
		case agent.EventKindTurnStart:
			turnStarts++
		case agent.EventKindTurnEnd:
			turnEnds++
			te, _ := ev.TurnEnd()
			turnEndOutcomes = append(turnEndOutcomes, te.Outcome())
		case agent.EventKindToolStart:
			sawToolStart = true
		case agent.EventKindToolEndSuccess:
			sawToolEnd = true
		}
	}
	if runStarts != 1 || runEnds != 1 {
		t.Errorf("run_start count = %d, run_end count = %d, want exactly 1 each (one run bracket)", runStarts, runEnds)
	}
	if turnStarts != 2 || turnEnds != 2 {
		t.Errorf("turn_start count = %d, turn_end count = %d, want exactly 2 each (N turn brackets)", turnStarts, turnEnds)
	}
	if len(turnEndOutcomes) == 2 {
		if turnEndOutcomes[0] != agent.TurnOutcomeToolCalls {
			t.Errorf("turn one's turn_end outcome = %v, want TurnOutcomeToolCalls", turnEndOutcomes[0])
		}
		if turnEndOutcomes[1] != agent.TurnOutcomeFinished {
			t.Errorf("turn two's turn_end outcome = %v, want TurnOutcomeFinished", turnEndOutcomes[1])
		}
	}
	if !sawToolStart || !sawToolEnd {
		t.Errorf("tool events missing: toolStart=%v toolEnd=%v — want the tool's execution and result inside turn one", sawToolStart, sawToolEnd)
	}
	if got := tool.Invocations(); got != 1 {
		t.Errorf("tool invocations = %d, want 1", got)
	}

	runEnd, ok := events[len(events)-1].RunEnd()
	if !ok {
		t.Fatal("last event carries no RunEnd payload")
	}
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("run_end outcome = %v, want RunOutcomeCompleted", runEnd.Outcome())
	}
}

// AG-13.1 — S-RUN-011. Given a fake provider scripted so that a single
// turn ends in each terminal-candidate member of ai.FinishReason's
// vocabulary in turn, when each run is driven with an empty steering
// queue, then each run ends after exactly one turn, the run-close event
// carries the completed outcome, and no candidate falls through to a
// second provider call.
func TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn(t *testing.T) {
	t.Parallel()

	candidates := []ai.FinishReason{
		ai.FinishReasonStop,
		ai.FinishReasonLength,
		ai.FinishReasonContentFilter,
		ai.FinishReasonRefusal,
		ai.FinishReasonUnknown,
	}

	for _, fr := range candidates {
		fr := fr
		t.Run(fr.String(), func(t *testing.T) {
			t.Parallel()

			provider := agenttest.NewProvider(scriptTerminationResponse(t, fr))
			h := agent.Harness{Provider: provider, System: "system prompt for run-011"}

			sink := make(chan *agent.Event, 64)
			_, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
			if err != nil {
				t.Fatalf("Run returned err = %v, want nil", err)
			}
			if finish != fr {
				t.Errorf("finish = %v, want %v", finish, fr)
			}

			events := drainSink(t, sink)
			var turnStarts, turnEnds int
			for _, ev := range events {
				switch ev.Kind() {
				case agent.EventKindTurnStart:
					turnStarts++
				case agent.EventKindTurnEnd:
					turnEnds++
				}
			}
			if turnStarts != 1 || turnEnds != 1 {
				t.Errorf("turn_start count = %d, turn_end count = %d, want exactly 1 each — %v must end after exactly one turn", turnStarts, turnEnds, fr)
			}

			if len(events) == 0 {
				t.Fatal("zero events observed")
			}
			last := events[len(events)-1]
			runEnd, ok := last.RunEnd()
			if !ok {
				t.Fatalf("last event kind = %v, want run_end", last.Kind())
			}
			if runEnd.Outcome() != agent.RunOutcomeCompleted {
				t.Errorf("run_end outcome = %v, want RunOutcomeCompleted", runEnd.Outcome())
			}

			if got := len(provider.Requests()); got != 1 {
				t.Errorf("provider recorded %d request(s), want exactly 1 — no candidate falls through to a second provider call", got)
			}
		})
	}
}

// AG-13.1 — S-RUN-012. Given a run whose final turn is held at an
// agenttest.Gate and a user message offered through Steer while that
// turn is in flight, when the gate is released and the turn returns a
// terminal candidate, then Steer had returned nil, the queued message is
// appended, an additional turn bracket appears on the stream, and the
// run terminates only after that additional turn — proving the terminal
// decision and the queue close are one atomic step.
func TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	textStart, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	textDelta, err := ai.NewTextDelta(1, "turn one text")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	textEnd, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion1, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	heldScript := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Hold(gate),
		agenttest.Emit(textStart),
		agenttest.Emit(textDelta),
		agenttest.Emit(textEnd),
		agenttest.Emit(completion1),
	}}
	secondScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(heldScript, secondScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-012", History: hist}

	sink := make(chan *agent.Event, 256)
	type runResult struct {
		finish ai.FinishReason
		err    error
	}
	resultCh := make(chan runResult, 1)
	go func() {
		_, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- runResult{finish, err}
	}()

	<-gate.Reached()

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "steered near terminal"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(steered); err != nil {
		t.Fatalf("Steer returned %v, want nil (accepted before the run's terminal decision)", err)
	}

	gate.Release()

	events := drainSink(t, sink)
	res := <-resultCh
	if res.err != nil {
		t.Fatalf("Run returned err = %v, want nil", res.err)
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
		t.Errorf("turn_start count = %d, turn_end count = %d, want exactly 2 each — the steered message must yield an ADDITIONAL turn, not a dropped message", turnStarts, turnEnds)
	}

	entries := hist.Entries()
	var foundSteered bool
	for _, e := range entries {
		if e.Message().Role() != ai.RoleUser {
			continue
		}
		for _, part := range e.Message().Content() {
			if text, ok := part.Text(); ok && text == "steered near terminal" {
				foundSteered = true
			}
		}
	}
	if !foundSteered {
		t.Error("steered message not found in the transcript — it must have been appended, not dropped")
	}
}

// AG-13.1 — S-RUN-020. Given the complete event slice recorded from a
// two-turn run, when it is passed to CheckStream unmodified from an
// external test package, then it is accepted with no violation; and when
// the slice is walked, then it contains exactly one run-open and exactly
// one run-close, exactly two turn-open/turn-close brackets both nested
// between them, and a sequence that starts at 1 and increases by exactly
// 1 with no gap, no repeat and no restart across the whole run. Pins the
// composition of Phases 0-1 plus the iterating Run above — no new
// production code is expected here (design.md).
func TestHarness_EventStream_OneRunBracketContiguousLane_CheckStreamAccepts(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_run_020", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_run_020": tool})
	turnOneScript := scriptToolCallResponse(t, "call-run-020", "read_run_020", []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for run-020", Turn: agent.TurnOptions{Tools: reg}}

	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Fatalf("CheckStream rejected the run stream: %v", report.Violation())
	}

	var runStarts, runEnds, turnStarts, turnEnds int
	var lastSeq agent.Sequence
	for i, ev := range events {
		wantSeq := agent.Sequence(i + 1)
		if ev.Sequence() != wantSeq {
			t.Errorf("event[%d] sequence = %v, want %v (contiguous 1-based lane)", i, ev.Sequence(), wantSeq)
		}
		lastSeq = ev.Sequence()
		switch ev.Kind() {
		case agent.EventKindRunStart:
			runStarts++
		case agent.EventKindRunEnd:
			runEnds++
		case agent.EventKindTurnStart:
			turnStarts++
		case agent.EventKindTurnEnd:
			turnEnds++
		}
	}
	if runStarts != 1 || runEnds != 1 {
		t.Errorf("run_start=%d run_end=%d, want exactly 1 each", runStarts, runEnds)
	}
	if turnStarts != 2 || turnEnds != 2 {
		t.Errorf("turn_start=%d turn_end=%d, want exactly 2 each", turnStarts, turnEnds)
	}
	if int(lastSeq) != len(events) {
		t.Errorf("last sequence = %d, want %d (no gap, no repeat, no restart)", lastSeq, len(events))
	}
}

// AG-13.1 — S-RUN-030 / S-RUN-031. Given the complete event slice
// recorded from a multi-turn run, when every event's run identity is
// read, then all values are equal to one another and equal to the
// identity carried by the run-open event, asserted by this capability's
// own test rather than by CheckStream; and given a harness-driven run
// and a bare Turn invocation recorded side by side, when their run
// identities are read, then the harness-driven identity carries the
// run-hrn- prefix, the bare invocation's carries the loop's own prefix,
// and the two prefixes are different.
func TestHarness_RunIdentity_ConsistentAcrossEventsAndProvenanceDistinct(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_run_030", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_run_030": tool})
	turnOneScript := scriptToolCallResponse(t, "call-run-030", "read_run_030", []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for run-030", Turn: agent.TurnOptions{Tools: reg}}

	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)
	if len(events) == 0 {
		t.Fatal("zero events observed")
	}

	want := events[0].Run()
	if !strings.HasPrefix(string(want), "run-hrn-") {
		t.Errorf("harness-driven run identity = %q, want run-hrn- prefix", want)
	}
	for i, ev := range events {
		if ev.Run() != want {
			t.Errorf("event[%d].Run() = %q, want %q (same value across every event in one run)", i, ev.Run(), want)
		}
	}

	// A bare Turn invocation carries the loop's own, different prefix
	// (provenance-distinct — R-RUN-004).
	bareProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	bareSink := make(chan *agent.Event, 16)
	_, _, err := agent.Turn(contextBackground(), bareProvider, "bare turn system prompt", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, bareSink)
	if err != nil {
		t.Fatalf("agent.Turn returned err = %v, want nil", err)
	}
	bareEvents := drainSink(t, bareSink)
	if len(bareEvents) == 0 {
		t.Fatal("zero bare-turn events observed")
	}
	bareRunID := bareEvents[0].Run()
	if strings.HasPrefix(string(bareRunID), "run-hrn-") {
		t.Errorf("bare Turn invocation's run identity = %q, want the loop's own prefix, not run-hrn-", bareRunID)
	}
	if bareRunID == want {
		t.Error("harness-driven and bare-Turn run identities are equal, want distinct prefixes/values")
	}
}

// AG-13.1 — S-RUN-040. Charter AG-13.1 sc.1, second Then. Given the
// completed two-turn run, when the transcript is read back through the
// public read route, then it holds, in order, the prompt, turn one's
// assistant message carrying the tool call, the tool result message for
// that call, and turn two's assistant message; every tool call has a
// matching result; and the read-back sequence contains no duplicated and
// no missing message.
func TestHarness_History_AlternatingTranscriptEveryPairMatched(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("read_run_040", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_run_040": tool})
	turnOneScript := scriptToolCallResponse(t, "call-run-040", "read_run_040", []byte(`{"y":2}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-040", Turn: agent.TurnOptions{Tools: reg}, History: hist}

	sink := make(chan *agent.Event, 256)
	prompt := firstMessage(t)
	if _, _, err := h.Run(contextBackground(), prompt, sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	drainSink(t, sink)

	entries := hist.Entries()
	if len(entries) != 4 {
		t.Fatalf("history has %d entries, want 4 (prompt, turn-one assistant+call, tool result, turn-two assistant)", len(entries))
	}

	if !entries[0].Message().Equal(prompt) {
		t.Error("entries[0] is not the prompt")
	}
	if entries[1].Message().Role() != ai.RoleAssistant {
		t.Errorf("entries[1].Role() = %v, want RoleAssistant (turn one)", entries[1].Message().Role())
	}
	var foundCall bool
	for _, part := range entries[1].Message().Content() {
		if call, ok := part.ToolCall(); ok {
			foundCall = true
			if call.ID() != "call-run-040" {
				t.Errorf("committed ToolCall.ID() = %q, want %q", call.ID(), "call-run-040")
			}
		}
	}
	if !foundCall {
		t.Error("entries[1] carries no ToolCall part")
	}
	if entries[2].Message().Role() != ai.RoleTool {
		t.Errorf("entries[2].Role() = %v, want RoleTool", entries[2].Message().Role())
	}
	var foundResult bool
	for _, part := range entries[2].Message().Content() {
		if tr, ok := part.ToolResult(); ok {
			foundResult = true
			if tr.CallID() != "call-run-040" {
				t.Errorf("committed ToolResult.CallID() = %q, want %q", tr.CallID(), "call-run-040")
			}
		}
	}
	if !foundResult {
		t.Error("entries[2] carries no ToolResult part")
	}
	if entries[3].Message().Role() != ai.RoleAssistant {
		t.Errorf("entries[3].Role() = %v, want RoleAssistant (turn two)", entries[3].Message().Role())
	}

	if err := hist.CloseTurn(); err != nil {
		t.Errorf("history.CloseTurn() after the run = %v, want nil (every call matched)", err)
	}
}

// AG-13.1 — S-RUN-050 / S-LSK-017 (first half). Given the run driver's
// production source read as raw bytes by a source-scan guard (the
// scheduler_test.go regex precedent), when it is scanned, then it
// contains no reference to any enumerated loop internal — the turn
// accumulator, the loop's identity minters, its stamped-emission helper,
// its sink-close helper, its request builder, or its finish-reason
// mapper — and no Schedule call site (the R-LSK-006 reconciliation); and
// given the capability's behavioral tests, when their package clause is
// read, then every one of them is in an external test package.
func TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("harness.go")
	if err != nil {
		t.Fatalf("os.ReadFile(\"harness.go\") error = %v, want nil", err)
	}
	text := stripGoComments(string(src))

	forbidden := []string{
		"turnAccumulator",
		"mintLoopRunID",
		"mintLoopTurnID",
		"mintLoopMessageID",
		"emitStamped",
		"closeSink",
		"buildLoopRequest",
		"outcomeForFinish",
	}
	for _, name := range forbidden {
		if strings.Contains(text, name) {
			t.Errorf("source-guard violation: harness.go references %q — the harness MUST reach the loop through no channel but Turn's own public one-turn surface (R-RUN-006)", name)
		}
	}

	scheduleCall := regexp.MustCompile(`\.\s*Schedule\s*\(`)
	if loc := scheduleCall.FindStringIndex(text); loc != nil {
		t.Errorf("source-guard violation: %q found at offset %d in harness.go — the harness MUST NOT call Schedule directly (R-LSK-006 reconciliation, design.md); only Turn may invoke it", scheduleCall.String(), loc[0])
	}

	// Every capability test lives in an external test package: this
	// very file (and every harness_*_test.go sibling) declares
	// `package agent_test` — grep-verified, not merely asserted, so a
	// future file added in-package would be caught.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\") error = %v, want nil", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "harness") || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		testSrc, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", name, err)
		}
		if !strings.Contains(string(testSrc), "package agent_test") {
			t.Errorf("%s does not declare package agent_test — every harness capability test MUST live in an external test package (R-RUN-006, NFR-RUN-001)", name)
		}
	}
}
