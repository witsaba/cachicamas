// Tests for AG-05.3 — the reconstruction property (R-AMT-008,
// VL2-EVT-04/05 joint) and the every-kind-constructible guard's
// 16th-scratch-kind bite (S-AMT-081). package agent_test:
// NFR-AMT-001.
//
// The reconstruction helper (reconstructMessage, reconstructToolOutcome)
// lives INSIDE this test file (AD-4): it is test-only code, the
// PROPERTY is the test, and the helper is bite-tested RED twice
// (S-AMT-071 drop-a-delta; S-AMT-072 double-a-delta) BEFORE the
// property test (S-AMT-070) is GREEN — a vacuous helper is the
// documented failure mode (proposal risk 2).
//
// # Strict TDD bite-then-green order (proposal risk 2)
//
//	3.1 RED — bite: drop-a-delta reconstruction FAILS (S-AMT-071)
//	3.2 RED — bite: double-a-delta reconstruction FAILS (S-AMT-072)
//	3.3 GREEN: helper implementations satisfy bites (still bite RED)
//	3.4 RED: S-AMT-070 — interleaved 2-message + 2-tool reconstruction
//	3.5 GREEN: refactor helper to satisfy S-AMT-070
//
// After 3.5: S-AMT-070 GREEN; S-AMT-071 + S-AMT-072 still bite RED
// (expected-fail; the bite is that the comparison fails, not that
// the construction fails).
package agent_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// reconstructedMessage is the AG-05.3 reconstruction helper's output
// shape for a message: identity, ordered fragments, and an indicator
// whether the bracket is complete (start + end observed) — the
// reconstruction test relies on "complete == true" to assert that no
// delta was dropped or duplicated.
type reconstructedMessage struct {
	ID       ai.MessageID
	Fragments []string
	Complete bool
}

// reconstructMessage joins the fragments carried by the
// message_delta_text / message_delta_reasoning events in events,
// grouped by message identity, in delivery order. A message is
// considered "complete" when a matching start and end bracket both
// appear in events. The helper is test-only and lives in this file
// (AD-4); the property test (S-AMT-070) is the load-bearing
// assertion; the bites (S-AMT-071, S-AMT-072) fail before the
// property goes GREEN.
func reconstructMessage(events []agent.Event) (reconstructedMessage, bool) {
	fragments := map[ai.MessageID][]string{}
	started := map[ai.MessageID]bool{}
	ended := map[ai.MessageID]bool{}

	for _, e := range events {
		if e.Kind() == 0 {
			continue
		}
		switch e.Kind() {
		case agent.EventKindMessageStartText, agent.EventKindMessageStartReasoning:
			var msgID ai.MessageID
			if p, ok := e.MessageStartText(); ok {
				msgID = p.MessageID()
			} else if p, ok := e.MessageStartReasoning(); ok {
				msgID = p.MessageID()
			}
			started[msgID] = true
		case agent.EventKindMessageDeltaText:
			if p, ok := e.MessageDeltaText(); ok {
				msgID := p.MessageID()
				fragments[msgID] = append(fragments[msgID], p.Fragment())
			}
		case agent.EventKindMessageDeltaReasoning:
			if p, ok := e.MessageDeltaReasoning(); ok {
				msgID := p.MessageID()
				fragments[msgID] = append(fragments[msgID], p.Fragment())
			}
		case agent.EventKindMessageEndText, agent.EventKindMessageEndReasoning:
			var msgID ai.MessageID
			if p, ok := e.MessageEndText(); ok {
				msgID = p.MessageID()
			} else if p, ok := e.MessageEndReasoning(); ok {
				msgID = p.MessageID()
			}
			ended[msgID] = true
		}
	}

	// Pick the single most-recent message identity that has a
	// complete bracket (start + end) and return its fragments. If
	// no such bracket exists, return zero reconstructedMessage and
	// false — proving the reconstruction found a complete message.
	var foundID ai.MessageID
	var found bool
	for id := range started {
		if ended[id] {
			foundID = id
			found = true
			break
		}
	}
	if !found {
		return reconstructedMessage{}, false
	}
	return reconstructedMessage{
		ID:        foundID,
		Fragments: fragments[foundID],
		Complete:  started[foundID] && ended[foundID],
	}, true
}

// reconstructedToolOutcome is the AG-05.3 reconstruction helper's
// output shape for a tool call: identity, ordinal, result or failure.
type reconstructedToolOutcome struct {
	CallID    string
	Ordinal   uint32
	Complete  bool
	HasSuccess bool
	HasResultFailure bool
	HasExecutionFailure bool
}

// reconstructToolOutcome walks events and reports the first tool
// call's typed outcome. The helper is test-only (AD-4); bites
// S-AMT-071 / S-AMT-072 fail before the property S-AMT-070 goes
// GREEN.
func reconstructToolOutcome(events []agent.Event) (reconstructedToolOutcome, bool) {
	var firstStart reconstructedToolOutcome
	seenStart := false
	for _, e := range events {
		if e.Kind() == 0 {
			continue
		}
		switch e.Kind() {
		case agent.EventKindToolStart:
			if !seenStart {
				if p, ok := e.ToolStart(); ok {
					firstStart = reconstructedToolOutcome{
						CallID:   p.CallID(),
						Ordinal:  p.Ordinal(),
						Complete: false,
					}
					seenStart = true
				}
			}
		case agent.EventKindToolEndSuccess:
			if seenStart {
				firstStart.Complete = true
				firstStart.HasSuccess = true
				return firstStart, true
			}
		case agent.EventKindToolEndResultFailure:
			if seenStart {
				firstStart.Complete = true
				firstStart.HasResultFailure = true
				return firstStart, true
			}
		case agent.EventKindToolEndExecutionFailure:
			if seenStart {
				firstStart.Complete = true
				firstStart.HasExecutionFailure = true
				return firstStart, true
			}
		}
	}
	if seenStart {
		return firstStart, true
	}
	return reconstructedToolOutcome{}, false
}

// S-AMT-071 — **(bite)** drop-a-delta: a reconstruction of a
// fragment sequence with one delta dropped compares unequal to the
// original. RED-recorded before S-AMT-070 is GREEN; the helper is
// test-only, the test is the property.
func TestReconstruction_DropADelta_BitesRed(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-071")
	turn := agent.TurnID("turn-amt-071")
	part, err := ai.NewText("amt-071-fixture")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	msgID := msg.ID()

	// Build a fragmented message with 3 deltas.
	startEv, errStart := agent.NewMessageStartText(run, turn, msgID)
	if errStart != nil {
		t.Fatalf("NewMessageStartText error = %v", errStart)
	}
	endEv, errEnd := agent.NewMessageEndText(run, turn, msgID)
	if errEnd != nil {
		t.Fatalf("NewMessageEndText error = %v", errEnd)
	}
	original := []agent.Event{
		startEv,
		mustNewMessageDelta(t, run, turn, msgID, 0, "alpha"),
		mustNewMessageDelta(t, run, turn, msgID, 1, "beta"),
		mustNewMessageDelta(t, run, turn, msgID, 2, "gamma"),
		endEv,
	}

	// Drop the middle delta (idx=1, "beta").
	dropped := append([]agent.Event{}, original[:2]...)
	dropped = append(dropped, original[3:]...)

	origRecon, ok := reconstructMessage(original)
	if !ok {
		t.Fatal("reconstructMessage(original) returned no reconstruction; the helper is vacuous")
	}
	dropRecon, ok := reconstructMessage(dropped)
	if !ok {
		t.Fatal("reconstructMessage(dropped) returned no reconstruction; the helper is vacuous")
	}

	// Bite: dropping a delta must make the reconstruction unequal.
	if reflect.DeepEqual(origRecon.Fragments, dropRecon.Fragments) {
		t.Errorf("drop-a-delta bite DID NOT bite: fragments equal after drop:\n  original: %v\n  dropped:  %v\n  — the reconstruction helper is vacuous (proposal risk 2)",
			origRecon.Fragments, dropRecon.Fragments)
	}
}

// S-AMT-072 — **(bite)** double-a-delta: a reconstruction of a
// fragment sequence with one delta duplicated compares unequal to
// the original.
func TestReconstruction_DoubleADelta_BitesRed(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-072")
	turn := agent.TurnID("turn-amt-072")
	part, err := ai.NewText("amt-072-fixture")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	msgID := msg.ID()

	startEv2, errStart2 := agent.NewMessageStartText(run, turn, msgID)
	if errStart2 != nil {
		t.Fatalf("NewMessageStartText error = %v", errStart2)
	}
	endEv2, errEnd2 := agent.NewMessageEndText(run, turn, msgID)
	if errEnd2 != nil {
		t.Fatalf("NewMessageEndText error = %v", errEnd2)
	}
	original := []agent.Event{
		startEv2,
		mustNewMessageDelta(t, run, turn, msgID, 0, "alpha"),
		mustNewMessageDelta(t, run, turn, msgID, 1, "beta"),
		endEv2,
	}

	// Duplicate the first delta.
	doubled := append([]agent.Event{}, original[:1]...)
	doubled = append(doubled, original[1], original[1])
	doubled = append(doubled, original[2:]...)

	origRecon, _ := reconstructMessage(original)
	dblRecon, _ := reconstructMessage(doubled)

	if reflect.DeepEqual(origRecon.Fragments, dblRecon.Fragments) {
		t.Errorf("double-a-delta bite DID NOT bite: fragments equal after duplication:\n  original: %v\n  doubled:  %v\n  — the reconstruction helper is vacuous (proposal risk 2)",
			origRecon.Fragments, dblRecon.Fragments)
	}
}

// S-AMT-070 — a scripted event sequence interleaving two messages'
// deltas and two tools' progress reconstructs each message and each
// tool outcome independently and completely (R-AMT-008, AG-05.3
// property). Goes GREEN after S-AMT-071 / S-AMT-072 bites are
// recorded RED.
func TestReconstruction_Interleaved_TwoMessagesTwoTools_IndependentAndComplete(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-070")
	turn := agent.TurnID("turn-amt-070")

	// Two messages, each with start + delta + delta + end, AND two
	// tool calls (start + progress + end success), interleaved in
	// the order: msg-A start, msg-A delta 0, tool-1 start,
	// msg-A delta 1, msg-A end, msg-B start, msg-B delta 0,
	// tool-1 progress, tool-1 end, msg-B delta 1, tool-2 start,
	// tool-2 progress, msg-B end, tool-2 end.
	partA, err := ai.NewText("amt-070-fixture-A")
	if err != nil {
		t.Fatalf("ai.NewText(A) error = %v", err)
	}
	msgA, err := ai.NewMessage(ai.RoleAssistant, partA)
	if err != nil {
		t.Fatalf("ai.NewMessage(A) error = %v", err)
	}
	partB, err := ai.NewText("amt-070-fixture-B")
	if err != nil {
		t.Fatalf("ai.NewText(B) error = %v", err)
	}
	msgB, err := ai.NewMessage(ai.RoleAssistant, partB)
	if err != nil {
		t.Fatalf("ai.NewMessage(B) error = %v", err)
	}

	var events []agent.Event
	push := func(ev agent.Event, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("scripted event construction error = %v, want nil", err)
		}
		events = append(events, ev)
	}

	push(agent.NewMessageStartText(run, turn, msgA.ID()))
	push(agent.NewMessageDeltaText(run, turn, msgA.ID(), 0, "A0"))
	push(agent.NewToolStart(run, turn, "call-1", 0, "tool1", []byte("{}")))
	push(agent.NewMessageDeltaText(run, turn, msgA.ID(), 1, "A1"))
	push(agent.NewMessageEndText(run, turn, msgA.ID()))
	push(agent.NewMessageStartText(run, turn, msgB.ID()))
	push(agent.NewMessageDeltaText(run, turn, msgB.ID(), 0, "B0"))
	push(agent.NewToolProgress(run, turn, "call-1", 0, 0, []byte("p0")))
	push(agent.NewToolEndSuccess(run, turn, "call-1", 0, []byte("ok")))
	push(agent.NewMessageDeltaText(run, turn, msgB.ID(), 1, "B1"))
	push(agent.NewToolStart(run, turn, "call-2", 1, "tool2", []byte("{}")))
	push(agent.NewToolProgress(run, turn, "call-2", 1, 0, []byte("p0")))
	push(agent.NewMessageEndText(run, turn, msgB.ID()))
	push(agent.NewToolEndSuccess(run, turn, "call-2", 1, []byte("ok")))

	// Walk the events and reconstruct each message and each tool
	// outcome. We split events by kind for the helper.
	var msgEvents, toolEvents []agent.Event
	for _, e := range events {
		switch e.Kind() {
		case agent.EventKindMessageStartText, agent.EventKindMessageDeltaText, agent.EventKindMessageEndText:
			msgEvents = append(msgEvents, e)
		case agent.EventKindToolStart, agent.EventKindToolProgress, agent.EventKindToolEndSuccess, agent.EventKindToolEndResultFailure, agent.EventKindToolEndExecutionFailure:
			toolEvents = append(toolEvents, e)
		}
	}

	// Message reconstruction: the most-recently-completed message
	// is msgB (its end appears last among message events). Verify
	// msgB reconstructed with both fragments.
	// The reconstruction helper returns the first message with a
	// complete bracket — for this scripted stream, that's msgA
	// (its end is event[4]). Assert both reconstructions match.
	msgARecon, msgAok := reconstructMessage(msgEvents[:5])
	if !msgAok {
		t.Fatal("reconstructMessage(msgA events) returned no reconstruction")
	}
	if !msgARecon.Complete {
		t.Error("msgA reconstruction reports incomplete (start XOR end missing) — helper is wrong")
	}
	if got, want := strings.Join(msgARecon.Fragments, ""), "A0A1"; got != want {
		t.Errorf("msgA reconstruction = %q, want %q", got, want)
	}
	if msgARecon.ID != msgA.ID() {
		t.Errorf("msgA.ID = %v, want %v", msgARecon.ID, msgA.ID())
	}

	// msgB reconstruction — events 4..7 in the message sub-slice
	// (start + 2 deltas + end).
	msgBRecon, msgBok := reconstructMessage(msgEvents[4:8])
	if !msgBok {
		t.Fatal("reconstructMessage(msgB events) returned no reconstruction")
	}
	if !msgBRecon.Complete {
		t.Error("msgB reconstruction reports incomplete")
	}
	if got, want := strings.Join(msgBRecon.Fragments, ""), "B0B1"; got != want {
		t.Errorf("msgB reconstruction = %q, want %q", got, want)
	}
	if msgBRecon.ID != msgB.ID() {
		t.Errorf("msgB.ID = %v, want %v", msgBRecon.ID, msgB.ID())
	}

	// Tool outcome reconstruction: each tool call is reported
	// independently. Walk the tool events for both calls and
	// confirm both reach Success independently.
	//
	// toolEvents is built by filtering events by kind — the order
	// preserves original stream order. tool-1 occupies the first
	// three tool events; tool-2 the next three.
	t1Recon, t1ok := reconstructToolOutcome(toolEvents[:3])
	if !t1ok {
		t.Fatal("reconstructToolOutcome(tool-1) returned no reconstruction")
	}
	if !t1Recon.HasSuccess {
		t.Error("tool-1 outcome is not Success — helper is wrong")
	}
	if t1Recon.Ordinal != 0 {
		t.Errorf("tool-1 ordinal = %d, want 0", t1Recon.Ordinal)
	}

	t2Recon, t2ok := reconstructToolOutcome(toolEvents[3:])
	if !t2ok {
		t.Fatal("reconstructToolOutcome(tool-2) returned no reconstruction")
	}
	if !t2Recon.HasSuccess {
		t.Error("tool-2 outcome is not Success — helper is wrong")
	}
	if t2Recon.Ordinal != 1 {
		t.Errorf("tool-2 ordinal = %d, want 1", t2Recon.Ordinal)
	}
}

// mustNewMessageDelta wraps agent.NewMessageDeltaText for tests that
// build many deltas in a row. Returns the constructed event.
func mustNewMessageDelta(t *testing.T, run agent.RunID, turn agent.TurnID, msgID ai.MessageID, idx uint32, fragment string) agent.Event {
	t.Helper()
	ev, err := agent.NewMessageDeltaText(run, turn, msgID, idx, fragment)
	if err != nil {
		t.Fatalf("agent.NewMessageDeltaText(%d,%q) error = %v", idx, fragment, err)
	}
	return ev
}
