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
