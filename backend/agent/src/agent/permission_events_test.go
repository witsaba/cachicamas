// AG-06.1 — the permission family tests (R-APE-001, R-APE-002, R-APE-003,
// VL2-EVT-06).
//
// Permission events record the model's "may this call proceed?" loop:
// [NewPermissionDecisionRequired] opens one, [NewPermissionDecisionMade]
// answers it (with a typed [PermissionOutcome]), and
// [NewPermissionResolutionRemembered] records a fact about future calls
// the session log needs or it cannot explain why a later call was never
// asked. All three are constructible through the public surface, all three
// carry `Terminal: false`, and the remembered kind is the first exercise
// of the AG-04.3-reserved `CardinalityAtMostOne` seam
// (`event_descriptor.go:103-120`).
//
// # Strict TDD — RED-first
//
// Every scenario in this file references a type or function the
// production file (permission_events.go) does not yet declare. The
// file's compile-failure is the RED; the per-scenario assertions are the
// RED that proves the production behaviour matches the spec scenarios.

package agent_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// witnessPermissionFailure is the shared [agent.Failure] fixture used by
// the decision_made (Deny) and decision_made (ModifyInput) subtests.
// Mirrors witnessToolFailure's posture (event_registry_test.go): a
// test-scope fixture, not part of the public surface.
var witnessPermissionFailure *agent.Failure

func init() {
	inner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		panic("permission_events_test.go: ai.MidStreamFailure failed: " + err.Error())
	}
	f, err := agent.NewFailure(inner)
	if err != nil {
		panic("permission_events_test.go: agent.NewFailure failed: " + err.Error())
	}
	witnessPermissionFailure = f
}

// S-APE-001 — a permission_decision_required event carries the call
// identity, tool name, and arguments as distinct readable fields
// (R-APE-001). The kind is distinct from decision_made and
// resolution_remembered.
func TestPermission_DecisionRequired_ReadsCallIdentityToolNameArguments(t *testing.T) {
	t.Parallel()

	const (
		callID = "call-ape-001"
		tool   = "fs.read"
		args   = "{\"path\":\"/x\"}"
	)
	event, err := agent.NewPermissionDecisionRequired(
		"run-permission-witness", "turn-permission-witness", callID, tool, []byte(args),
	)
	if err != nil {
		t.Fatalf("NewPermissionDecisionRequired = %v, want nil", err)
	}

	// Kind is distinct.
	if event.Kind() != agent.EventKindPermissionDecisionRequired {
		t.Errorf("event.Kind() = %v, want %v", event.Kind(), agent.EventKindPermissionDecisionRequired)
	}

	// Payload accessor reads the three distinct fields.
	got, ok := event.PermissionDecisionRequired()
	if !ok {
		t.Fatalf("event.PermissionDecisionRequired() reported no payload on an event of its own kind")
	}
	if got.CallID() != callID {
		t.Errorf("CallID() = %q, want %q", got.CallID(), callID)
	}
	if got.Name() != tool {
		t.Errorf("Name() = %q, want %q", got.Name(), tool)
	}
	if string(got.Arguments()) != args {
		t.Errorf("Arguments() = %q, want %q", got.Arguments(), args)
	}

	// Accessor distinguishes "not this kind" from "this kind, empty".
	if _, ok := event.PermissionDecisionMade(); ok {
		t.Errorf("PermissionDecisionMade() reported a payload on a decision_required event")
	}
	if _, ok := event.PermissionResolutionRemembered(); ok {
		t.Errorf("PermissionResolutionRemembered() reported a payload on a decision_required event")
	}
}

// S-APE-010 — a permission_decision_made event reaches every typed
// [PermissionOutcome] value as a distinct typed value (R-APE-002). The
// enum is closed: zero is not a member.
func TestPermission_DecisionMade_ReachesAllFourTypedOutcomes(t *testing.T) {
	t.Parallel()

	outcomes := []agent.PermissionOutcome{
		agent.PermissionOutcomeAllowOnce,
		agent.PermissionOutcomeAllowAlways,
		agent.PermissionOutcomeDeny,
		agent.PermissionOutcomeModifyInput,
	}
	seen := make(map[agent.PermissionOutcome]bool)

	for _, want := range outcomes {
		// Per-outcome: only Deny carries a failure; only ModifyInput
		// carries modified arguments; AllowOnce/AllowAlways carry
		// neither. The constructor enforces the iff rule.
		var modifiedArgs []byte
		var failure *agent.Failure
		switch want {
		case agent.PermissionOutcomeModifyInput:
			modifiedArgs = []byte(`{"safe":true}`)
		case agent.PermissionOutcomeDeny:
			failure = witnessPermissionFailure
		}
		event, err := agent.NewPermissionDecisionMade(
			"run-permission-witness", "turn-permission-witness", "call-ape-010",
			want, modifiedArgs, failure,
		)
		if err != nil {
			t.Fatalf("NewPermissionDecisionMade(%v) = %v, want nil", want, err)
		}
		got, ok := event.PermissionDecisionMade()
		if !ok {
			t.Fatalf("PermissionDecisionMade() reported no payload on an event of its own kind")
		}
		if got.Outcome() != want {
			t.Errorf("Outcome() = %v, want %v (the typed value passed to the constructor)", got.Outcome(), want)
		}
		if seen[want] {
			t.Errorf("outcome %v appears more than once across the four-typed-value sweep", want)
		}
		seen[want] = true
	}

	// Closed enum: zero is not a member — the ctor MUST reject it.
	if _, err := agent.NewPermissionDecisionMade(
		"run-permission-witness", "turn-permission-witness", "call-ape-010",
		agent.PermissionOutcome(0), nil, nil,
	); err == nil {
		t.Errorf("NewPermissionDecisionMade(zero outcome) returned no error; the closed enum MUST reject zero (R-APE-002)")
	}
}

// S-APE-011 — a permission_decision_made event with ModifyInput carries
// the modified arguments as a distinct payload field (R-APE-002's
// ModifyInput half).
func TestPermission_DecisionMade_ModifyInputCarriesModifiedArguments(t *testing.T) {
	t.Parallel()

	const modified = "{\"path\":\"/safe\"}"
	event, err := agent.NewPermissionDecisionMade(
		"run-permission-witness", "turn-permission-witness", "call-ape-011",
		agent.PermissionOutcomeModifyInput, []byte(modified), nil,
	)
	if err != nil {
		t.Fatalf("NewPermissionDecisionMade = %v, want nil", err)
	}

	got, ok := event.PermissionDecisionMade()
	if !ok {
		t.Fatalf("PermissionDecisionMade() reported no payload on an event of its own kind")
	}
	if got.Outcome() != agent.PermissionOutcomeModifyInput {
		t.Errorf("Outcome() = %v, want ModifyInput", got.Outcome())
	}
	if string(got.ModifiedArguments()) != modified {
		t.Errorf("ModifiedArguments() = %q, want %q", got.ModifiedArguments(), modified)
	}
}

// S-APE-012 — a permission_decision_made event with Deny carries a
// typed `*agent.Failure` reachable through the typed-failure surface
// (R-APE-002's Deny half, R-AEV-008).
func TestPermission_DecisionMade_DenyCarriesTypedFailure(t *testing.T) {
	t.Parallel()

	event, err := agent.NewPermissionDecisionMade(
		"run-permission-witness", "turn-permission-witness", "call-ape-012",
		agent.PermissionOutcomeDeny, nil, witnessPermissionFailure,
	)
	if err != nil {
		t.Fatalf("NewPermissionDecisionMade = %v, want nil", err)
	}

	got, ok := event.PermissionDecisionMade()
	if !ok {
		t.Fatalf("PermissionDecisionMade() reported no payload on an event of its own kind")
	}
	if got.Outcome() != agent.PermissionOutcomeDeny {
		t.Errorf("Outcome() = %v, want Deny", got.Outcome())
	}
	f, ok := got.Failure()
	if !ok {
		t.Fatalf("Failure() reported no failure on a Deny event; the typed-failure surface is required (R-AEV-008)")
	}
	if f == nil {
		t.Errorf("Failure() reported no failure but ok=true")
	}
	if f.Category() != ai.FailureCategoryUnavailable {
		t.Errorf("Failure().Category() = %v, want %v", f.Category(), ai.FailureCategoryUnavailable)
	}
}

// S-APE-020 — a permission_resolution_remembered event is a distinct
// kind from permission_decision_made, and its payload carries the tool
// name and the remembered outcome as separate fields (R-APE-003).
func TestPermission_ResolutionRemembered_DistinctFromDecisionMade(t *testing.T) {
	t.Parallel()

	const tool = "fs.write"
	event, err := agent.NewPermissionResolutionRemembered(
		"run-permission-witness", tool, agent.PermissionOutcomeAllowAlways,
	)
	if err != nil {
		t.Fatalf("NewPermissionResolutionRemembered = %v, want nil", err)
	}

	// Distinct kind.
	if event.Kind() != agent.EventKindPermissionResolutionRemembered {
		t.Errorf("event.Kind() = %v, want %v", event.Kind(), agent.EventKindPermissionResolutionRemembered)
	}

	got, ok := event.PermissionResolutionRemembered()
	if !ok {
		t.Fatalf("PermissionResolutionRemembered() reported no payload on an event of its own kind")
	}
	if got.ToolName() != tool {
		t.Errorf("ToolName() = %q, want %q", got.ToolName(), tool)
	}
	if got.Outcome() != agent.PermissionOutcomeAllowAlways {
		t.Errorf("Outcome() = %v, want AllowAlways", got.Outcome())
	}

	// Accessor distinguishes from decision_made.
	if _, ok := event.PermissionDecisionMade(); ok {
		t.Errorf("PermissionDecisionMade() reported a payload on a resolution_remembered event")
	}
}

// S-APE-082 — **(bite)** two permission_resolution_remembered events for
// the same tool name on one stream are REJECTED with the
// CardinalityAtMostOne rule and the second offending position. RED-recorded
// before the registry ships.
//
// S-APE-082 — BITE
// RED:  scratch violation here (reverted)
// Bite: stream with two permission_resolution_remembered events for
//       the same tool name MUST be REJECTED by the validator at the
//       second event's 0-based slice index [3] with ai.ErrDuplicate.
// Asserted: CheckStream returns a non-nil violation matching the rule
//           AND the position's last index step equals 3, not the
//           offending event's sequence value 4 (W2 carry-forward).
// GREEN: the seam at stream_check.go:166-171 honors the descriptor's
//        CardinalityAtMostOne declaration; the test passes.
func TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed(t *testing.T) {
	t.Parallel()

	runID := agent.RunID("run-ape-082-cardinality")
	turnID := agent.TurnID("turn-ape-082-cardinality")

	first, err := agent.NewPermissionResolutionRemembered(runID, "fs.write", agent.PermissionOutcomeAllowAlways)
	if err != nil {
		t.Fatalf("first NewPermissionResolutionRemembered = %v, want nil", err)
	}
	second, err := agent.NewPermissionResolutionRemembered(runID, "fs.write", agent.PermissionOutcomeDeny)
	if err != nil {
		t.Fatalf("second NewPermissionResolutionRemembered = %v, want nil", err)
	}

	// Build a minimal stream: a delegated run-start (so parent is settable
	// in future families, irrelevant here), one turn, then two
	// resolution_remembered events for the same tool name.
	runStart, err := agent.NewRunStart(runID)
	if err != nil {
		t.Fatalf("NewRunStart = %v, want nil", err)
	}
	turnStart, err := agent.NewTurnStart(runID, turnID)
	if err != nil {
		t.Fatalf("NewTurnStart = %v, want nil", err)
	}
	turnEnd, err := agent.NewTurnEnd(runID, turnID, agent.TurnOutcomeFinished, nil)
	if err != nil {
		t.Fatalf("NewTurnEnd = %v, want nil", err)
	}
	runEnd, err := agent.NewRunEnd(runID, agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("NewRunEnd = %v, want nil", err)
	}

	var lane agent.LaneStamper
	stream := []agent.Event{
		lane.Stamp(runStart),
		lane.Stamp(turnStart),
		lane.Stamp(first),
		lane.Stamp(second),
		lane.Stamp(turnEnd),
		lane.Stamp(runEnd),
	}

	report := agent.CheckStream(stream)
	if report.Violation() == nil {
		t.Fatalf("agent.CheckStream accepted two permission_resolution_remembered events for the same tool name; the CardinalityAtMostOne seam MUST reject the second one (R-APE-003, S-APE-082)")
	}
	requireViolationPosition(t, report.Violation(), 3) // 0-based index of the second resolution_remembered

	// Rule class: must be ErrDuplicate (the seam's documented class).
	if !errors.Is(report.Violation(), ai.ErrDuplicate) {
		t.Errorf("rejection error = %v, want errors.Is to match ai.ErrDuplicate (CardinalityAtMostOne seam)", report.Violation())
	}
}
