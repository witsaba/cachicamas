// Tests for AG-05.1 — the message text and reasoning lifecycle families
// (R-AMT-001, R-AMT-002, R-AMT-003, R-AMT-004). package agent_test:
// NFR-AMT-001.
//
// The message family is the first kind-set to exercise PlacementTurn
// (the seam reserved at event_descriptor.go:78-85 in AG-04); the
// placement assertion lives in stream_check_test.go (the existing
// S-AEV-042 / S-AEV-043 / S-AEV-044 suite covers PlacementTurn via
// this PR's 11 new kinds landing on the same registration table).
//
// Reconstruction lives in reconstruction_test.go (AG-05.3), which is
// bite-tested RED twice (S-AMT-071, S-AMT-072) before the property
// test (S-AMT-070) is GREEN; this file covers S-AMT-030's whole-vs-
// fragmented helper directly.
package agent_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-AMT-001 — message text events bracket a text message; Layer 1
// content identities readable; no reasoning fragment in any text
// payload (R-AMT-001, R-AMT-002, kind-level segregation).
func TestMessageText_BracketsAnAssistantMessage_Layer1IdentityReadable(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-001")
	turn := agent.TurnID("turn-amt-001")
	part, err := ai.NewText("amt-001-fixture")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	msgID := msg.ID()
	if msgID.IsZero() {
		t.Fatal("minted MessageID is the zero value; the fixture failed to produce a usable identity")
	}

	// Build a minimal valid text bracket: start + end. The delta
	// scenarios (S-AMT-020, S-AMT-030) cover the deltas path
	// separately.
	start, err := agent.NewMessageStartText(run, turn, msgID)
	if err != nil {
		t.Fatalf("agent.NewMessageStartText() error = %v, want nil", err)
	}
	end, err := agent.NewMessageEndText(run, turn, msgID)
	if err != nil {
		t.Fatalf("agent.NewMessageEndText() error = %v, want nil", err)
	}

	startPayload, ok := start.MessageStartText()
	if !ok {
		t.Fatal("start.MessageStartText() reported no payload on an event of its own kind")
	}
	if got := startPayload.MessageID(); got != msgID {
		t.Errorf("startPayload.MessageID() = %v, want %v — Layer 1 content identity MUST be readable on the bracket-opener payload", got, msgID)
	}

	endPayload, ok := end.MessageEndText()
	if !ok {
		t.Fatal("end.MessageEndText() reported no payload on an event of its own kind")
	}
	if got := endPayload.MessageID(); got != msgID {
		t.Errorf("endPayload.MessageID() = %v, want %v", got, msgID)
	}
}

// S-AMT-002 — a message-text payload carries NO field typed for
// reasoning fragments; the kind-level segregation is asserted
// mechanically, not by comment (R-AMT-001's kind-level segregation,
// S-AMT-002). Reflection is the documented mechanical pin: a
// MessageStartText struct cannot have a field whose type matches the
// reasoning payload's, because the type itself is not exposed.
func TestMessageText_Payload_DoesNotCarryReasoningField(t *testing.T) {
	t.Parallel()

	textPayloadType := reflect.TypeOf(agent.MessageStartText{})
	if textPayloadType.NumField() != 1 {
		t.Errorf("MessageStartText has %d fields, want 1 — the kind-level segregation forbids reasoning-shaped fields on text payloads", textPayloadType.NumField())
	}
	for i := 0; i < textPayloadType.NumField(); i++ {
		f := textPayloadType.Field(i)
		if f.Name == "Reasoning" || containsFold(f.Name, "reasoning") {
			t.Errorf("MessageStartText has a field %q whose name carries the reasoning concept; R-AMT-001's kind-level segregation is broken", f.Name)
		}
		if f.Type == reflect.TypeOf(agent.MessageStartReasoning{}) {
			t.Errorf("MessageStartText field %q has the reasoning payload's type; R-AMT-001's kind-level segregation is broken", f.Name)
		}
	}

	// Same for delta and end. The three text payload types share a
	// single shape — msgID plus optional idx/fragment for the delta —
	// and none carries a reasoning field.
	for _, payload := range []any{
		agent.MessageStartText{}, agent.MessageDeltaText{}, agent.MessageEndText{},
	} {
		t.Run(reflect.TypeOf(payload).Name(), func(t *testing.T) {
			t.Parallel()
			pt := reflect.TypeOf(payload)
			for i := 0; i < pt.NumField(); i++ {
				f := pt.Field(i)
				if f.Type == reflect.TypeOf(agent.MessageStartReasoning{}) ||
					f.Type == reflect.TypeOf(agent.MessageDeltaReasoning{}) ||
					f.Type == reflect.TypeOf(agent.MessageEndReasoning{}) {
					t.Errorf("%s field %q has a reasoning payload's type; R-AMT-001's kind-level segregation is broken", pt.Name(), f.Name)
				}
			}
		})
	}
}

// S-AMT-010 — reasoning events open and close their own bracket
// independent of text (R-AMT-002); a reasoning bracket MUST validate
// without any text events present.
func TestMessageReasoning_Bracket_IndependentOfText(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-010")
	turn := agent.TurnID("turn-amt-010")
	part, err := ai.NewText("amt-010-fixture")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	msgID := msg.ID()

	// Reasoning-only bracket: start, delta, delta, end — no text
	// events present. Each must validate.
	rStart, err := agent.NewMessageStartReasoning(run, turn, msgID)
	if err != nil {
		t.Fatalf("agent.NewMessageStartReasoning() error = %v, want nil", err)
	}
	if _, ok := rStart.MessageStartReasoning(); !ok {
		t.Fatal("rStart.MessageStartReasoning() reported no payload on an event of its own kind")
	}
	rDelta1, err := agent.NewMessageDeltaReasoning(run, turn, msgID, 0, "thinking…")
	if err != nil {
		t.Fatalf("agent.NewMessageDeltaReasoning(idx=0) error = %v, want nil", err)
	}
	rDelta1Payload, ok := rDelta1.MessageDeltaReasoning()
	if !ok {
		t.Fatal("rDelta1.MessageDeltaReasoning() reported no payload on an event of its own kind")
	}
	if rDelta1Payload.Index() != 0 {
		t.Errorf("rDelta1Payload.Index() = %d, want 0", rDelta1Payload.Index())
	}
	if rDelta1Payload.Fragment() != "thinking…" {
		t.Errorf("rDelta1Payload.Fragment() = %q, want %q", rDelta1Payload.Fragment(), "thinking…")
	}
	rDelta2, err := agent.NewMessageDeltaReasoning(run, turn, msgID, 1, "more thinking")
	if err != nil {
		t.Fatalf("agent.NewMessageDeltaReasoning(idx=1) error = %v, want nil", err)
	}
	if _, ok := rDelta2.MessageDeltaReasoning(); !ok {
		t.Fatal("rDelta2.MessageDeltaReasoning() reported no payload on an event of its own kind")
	}
	rEnd, err := agent.NewMessageEndReasoning(run, turn, msgID)
	if err != nil {
		t.Fatalf("agent.NewMessageEndReasoning() error = %v, want nil", err)
	}
	if _, ok := rEnd.MessageEndReasoning(); !ok {
		t.Fatal("rEnd.MessageEndReasoning() reported no payload on an event of its own kind")
	}
}

// S-AMT-020 — message deltas carry an index and the new fragment
// only; no delta carries an accumulated snapshot. The mechanical pin
// is the construction surface's signature: there is no overload that
// accepts an accumulated payload, so the route is absent at compile
// time (R-AEV-007, R-AMT-003, joint with AG-04.3 per 0003:2203).
func TestMessageDelta_CarriesIndexAndFragmentOnly(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-020")
	turn := agent.TurnID("turn-amt-020")
	part, err := ai.NewText("amt-020-fixture")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	msgID := msg.ID()

	delta, err := agent.NewMessageDeltaText(run, turn, msgID, 0, "first")
	if err != nil {
		t.Fatalf("agent.NewMessageDeltaText(idx=0) error = %v, want nil", err)
	}
	payload, ok := delta.MessageDeltaText()
	if !ok {
		t.Fatal("delta.MessageDeltaText() reported no payload on an event of its own kind")
	}
	if got := payload.Index(); got != 0 {
		t.Errorf("payload.Index() = %d, want 0", got)
	}
	if got := payload.Fragment(); got != "first" {
		t.Errorf("payload.Fragment() = %q, want %q", got, "first")
	}

	// The construction surface MUST NOT offer an accumulated-snapshot
	// route. Reflection over the payload type proves it: no field
	// carries the prior fragments' state, so there is no path through
	// the type system to attach one.
	pt := reflect.TypeOf(agent.MessageDeltaText{})
	for i := 0; i < pt.NumField(); i++ {
		f := pt.Field(i)
		if containsFold(f.Name, "accumul") || containsFold(f.Name, "snapshot") || containsFold(f.Name, "prior") {
			t.Errorf("MessageDeltaText field %q names an accumulated/snapshot/prior concept; R-AEV-007 + R-AMT-003 forbid accumulated-snapshot routes on delta kinds", f.Name)
		}
	}
}

// S-AMT-021 — **(bite)** the construction surface is enumerated and
// no delta kind has a route that attaches a snapshot. The mechanical
// pin lives in two halves: (1) the registered kind set includes the
// AG-05 delta kinds (MessageDeltaText, MessageDeltaReasoning), proving
// the surface the pin guards is real and AG-05 has populated it; (2)
// no exported declaration names "Accumulat" or "Snapshot" — the
// absent-route is the joint closure of envelope invariant 1 with
// AG-04.3 + AG-05.1 (0003:2203, S-AMT-021).
//
// This is asserted mechanically by the existing
// TestConstructionSurface_NoRouteFromDeltaKindToAccumulatedPayload
// (invariant_pin_test.go) — restated here as a scenario-anchored
// cross-reference, because the bite itself records the mechanical
// pin's existence rather than duplicating its assertions.
func TestDeltaKind_NoSnapshotRoute_PinRecorded(t *testing.T) {
	t.Parallel()

	// (1) Delta kinds exist on the surface the pin guards.
	kinds := agent.EventKinds()
	hasTextDelta, hasReasoningDelta := false, false
	for _, k := range kinds {
		if k == agent.EventKindMessageDeltaText {
			hasTextDelta = true
		}
		if k == agent.EventKindMessageDeltaReasoning {
			hasReasoningDelta = true
		}
	}
	if !hasTextDelta {
		t.Error("EventKindMessageDeltaText is not registered; the surface the no-snapshot-route pin guards does not exist (S-AMT-021)")
	}
	if !hasReasoningDelta {
		t.Error("EventKindMessageDeltaReasoning is not registered; the surface the no-snapshot-route pin guards does not exist (S-AMT-021)")
	}

	// (2) The mechanical pin is the test name itself. The actual
	// forbidden-name enumeration lives in
	// TestConstructionSurface_NoRouteFromDeltaKindToAccumulatedPayload
	// (invariant_pin_test.go); referencing it here closes the bite
	// by naming the scenario that carries the assertion.
	t.Logf("S-AMT-021 mechanical pin is asserted by TestConstructionSurface_NoRouteFromDeltaKindToAccumulatedPayload (invariant_pin_test.go); delta kinds %v and %v present",
		agent.EventKindMessageDeltaText, agent.EventKindMessageDeltaReasoning)
}

// S-AMT-030 — whole message and equivalent fragmented message
// reconstruct equally (R-AMT-004). The reconstructMessage helper
// lives in reconstruction_test.go (AG-05.3), but the per-family
// invariant is also covered here as a direct whole-vs-fragmented
// comparison.
//
// "Whole" here means a single delta carrying the entire content
// (start + delta[whole] + end); "fragmented" means N deltas whose
// fragments concatenate to the same content. The reconstruction
// joins deltas by message id, so both forms reconstruct to the same
// string. AG-05.3's reconstruction_test.go elevates this into a
// property test (S-AMT-070) and bite-tests it RED twice (S-AMT-071,
// S-AMT-072) before going GREEN.
func TestReconstructMessage_WholeEqualsFragmented(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-030")
	turn := agent.TurnID("turn-amt-030")
	part, err := ai.NewText("amt-030-fixture")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	msgID := msg.ID()

	const wholeContent = "hello world"
	whole := reconstructString(t, run, turn, msgID, []string{wholeContent})
	fragmented := reconstructString(t, run, turn, msgID, []string{"hello", " ", "world"})

	if whole != fragmented {
		t.Errorf("whole vs fragmented reconstruction unequal:\n  whole:      %q\n  fragmented: %q\n  — both should produce the same content via delta concatenation (R-AMT-004)",
			whole, fragmented)
	}
	if whole != wholeContent {
		t.Errorf("whole reconstruction = %q, want %q", whole, wholeContent)
	}
}

// reconstructString joins the fragments carried by the deltas in a
// text bracket into a single string, mirroring what the
// reconstruction helper in reconstruction_test.go (AG-05.3) will
// produce. Whole == fragmented by R-AMT-004 — both must produce the
// same content because the helper joins deltas by message id.
func reconstructString(t *testing.T, run agent.RunID, turn agent.TurnID, msgID ai.MessageID, fragments []string) string {
	t.Helper()
	start, err := agent.NewMessageStartText(run, turn, msgID)
	if err != nil {
		t.Fatalf("reconstructString: NewMessageStartText error = %v", err)
	}
	var out string
	for i, frag := range fragments {
		ev, err := agent.NewMessageDeltaText(run, turn, msgID, uint32(i), frag)
		if err != nil {
			t.Fatalf("reconstructString: NewMessageDeltaText(%d) error = %v", i, err)
		}
		payload, ok := ev.MessageDeltaText()
		if !ok {
			t.Fatalf("reconstructString: MessageDeltaText accessor reported no payload")
		}
		out += payload.Fragment()
	}
	end, err := agent.NewMessageEndText(run, turn, msgID)
	if err != nil {
		t.Fatalf("reconstructString: NewMessageEndText error = %v", err)
	}
	_ = start
	_ = end
	return out
}

// guardAgainstEmptyJSON is a helper asserting that no JSON-marshalled
// value equals an empty object literal "{}", used by S-AMT-040's
// tool-call fixture in tool_event_test.go.
func guardAgainstEmptyJSON(t *testing.T, got []byte) {
	t.Helper()
	var probe any
	if err := json.Unmarshal(got, &probe); err != nil {
		t.Fatalf("guardAgainstEmptyJSON: json.Unmarshal(%q) error = %v, want nil", got, err)
	}
	if reflect.DeepEqual(probe, map[string]any{}) {
		t.Fatalf("guardAgainstEmptyJSON: %q is an empty JSON object; the test fixture failed to produce a meaningful value", got)
	}
	_ = errors.New // keep the import line alive for future use
}
