// AG-06.3 — the delegation family tests (R-APE-006, VL2-EVT-08).
//
// Delegation events record a subagent's lifecycle inside a parent run:
// [NewSubagentStarted] opens the subagent's stream (parent-linked via
// the envelope's [Event.Parent]) and [NewSubagentEnded] closes it. The
// parent field is the same one AG-04.1 reserved on the envelope —
// AG-06.3 is the FIRST non-[NewDelegatedRunStart] consumer of it
// (`event.go:362-366`, R-AEV-003 direction-2).
//
// # Strict TDD — RED-first
//
// Every scenario in this file references a type or function the
// production file (delegation_events.go) does not yet declare. The
// file's compile-failure is the RED; the per-scenario assertions are
// the RED that proves the production behaviour matches the spec
// scenarios.

package agent_test

import (
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// S-APE-050 — a subagent_started event for a delegated run reports its
// parent's RunID through [Event.Parent] (R-APE-006). The parent
// identifier is set on the envelope, NOT a payload field — same shape
// as AG-04.1's [NewDelegatedRunStart].
func TestDelegation_SubagentStarted_ParentIsReadable(t *testing.T) {
	t.Parallel()

	const (
		subagentRun = "run-delegation-subagent" // the subagent's own run identity
		parent      = "run-delegation-parent"    // the delegating run's identity
		subagentID  = "sub-witness-050"
	)
	event, err := agent.NewSubagentStarted(subagentRun, parent, "turn-delegation-witness", subagentID)
	if err != nil {
		t.Fatalf("NewSubagentStarted = %v, want nil", err)
	}

	if event.Kind() != agent.EventKindSubagentStarted {
		t.Errorf("event.Kind() = %v, want %v", event.Kind(), agent.EventKindSubagentStarted)
	}

	// Parent identifier is set on the envelope — read via [Event.Parent].
	gotParent, hasParent := event.Parent()
	if !hasParent {
		t.Fatalf("event.Parent() reported no parent for a subagent_started event; the parent identifier MUST be set on the envelope (R-APE-006, R-AEV-003 direction-2)")
	}
	if gotParent != agent.RunID(parent) {
		t.Errorf("event.Parent() = %q, want %q", gotParent, parent)
	}

	// SubagentID is on the payload.
	got, ok := event.SubagentStarted()
	if !ok {
		t.Fatalf("SubagentStarted() reported no payload on an event of its own kind")
	}
	if got.SubagentID() != subagentID {
		t.Errorf("SubagentID() = %q, want %q", got.SubagentID(), subagentID)
	}
}

// S-APE-051 — a subagent_ended event for the same delegated run
// reports the same parent identifier as subagent_started (R-APE-006).
// The pair is parent-linked consistently.
func TestDelegation_SubagentStartedAndEnded_ShareParent(t *testing.T) {
	t.Parallel()

	const (
		subagentRun = "run-delegation-subagent-051"
		parent      = "run-delegation-parent-051"
		subagentID  = "sub-witness-051"
	)
	started, err := agent.NewSubagentStarted(subagentRun, parent, "turn-delegation-witness", subagentID)
	if err != nil {
		t.Fatalf("NewSubagentStarted = %v, want nil", err)
	}
	ended, err := agent.NewSubagentEnded(subagentRun, parent, "turn-delegation-witness", subagentID)
	if err != nil {
		t.Fatalf("NewSubagentEnded = %v, want nil", err)
	}

	startedParent, startedHas := started.Parent()
	if !startedHas {
		t.Fatalf("started.Parent() reported no parent")
	}
	endedParent, endedHas := ended.Parent()
	if !endedHas {
		t.Fatalf("ended.Parent() reported no parent")
	}
	if startedParent != endedParent {
		t.Errorf("started.Parent() = %q, ended.Parent() = %q — subagent_started and subagent_ended MUST share the same parent identifier (R-APE-006)", startedParent, endedParent)
	}
	if startedParent != agent.RunID(parent) {
		t.Errorf("started.Parent() = %q, want %q", startedParent, parent)
	}
}

// S-APE-052 — **(bite)** a non-delegated event from AG-04 or AG-05
// reports [Event.Parent] as (RunID(""), false) — the "no parent" state
// is unambiguous, not a zero-value trick (R-AEV-003, R-APE-006's
// negative half).
//
// S-APE-052 — BITE
// RED:  scratch violation here (reverted) — assigning a parent on
//       any AG-04 or AG-05 event would flip this test to RED.
// Bite: the parent identifier exists ONLY on events constructed by
//       the delegation doors. Non-delegated events report (zero
//       RunID, false) — the absence is the rule (R-AEV-003).
// Asserted: for each of the 18 AG-04 / AG-05 / AG-06.1 / AG-06.2
//           non-delegation kinds, Event.Parent() returns
//           (RunID(""), false).
// GREEN: only the delegation ctors set hasParent; the test passes.
func TestDelegation_NonDelegatedEvent_HasNoParent(t *testing.T) {
	t.Parallel()

	// Hand-build a representative non-delegated event of each kind
	// from AG-04, AG-05, AG-06.1 and AG-06.2. The constructors
	// themselves never set parent — the only doors that do are
	// NewDelegatedRunStart, NewSubagentStarted, NewSubagentEnded.
	nonDelegated := map[string]agent.Event{
		"run_start":            mustBuildEvent(t, "run-start-witness", func() (agent.Event, error) { return agent.NewRunStart("run-start-witness") }),
		"run_end":              mustBuildEvent(t, "run-end-witness", func() (agent.Event, error) { return agent.NewRunEnd("run-end-witness", agent.RunOutcomeCompleted, nil) }),
		"turn_start":           mustBuildEvent(t, "run-turn-witness", func() (agent.Event, error) { return agent.NewTurnStart("run-turn-witness", "turn-witness") }),
		"turn_end":             mustBuildEvent(t, "run-turn-end-witness", func() (agent.Event, error) { return agent.NewTurnEnd("run-turn-end-witness", "turn-end-witness", agent.TurnOutcomeFinished, nil) }),
		"message_start_text":   mustBuildEvent(t, "run-msg-witness", func() (agent.Event, error) { return agent.NewMessageStartText("run-msg-witness", "turn-msg-witness", witnessMessageID) }),
		"message_delta_text":   mustBuildEvent(t, "run-msg-witness", func() (agent.Event, error) { return agent.NewMessageDeltaText("run-msg-witness", "turn-msg-witness", witnessMessageID, 0, "x") }),
		"message_end_text":     mustBuildEvent(t, "run-msg-witness", func() (agent.Event, error) { return agent.NewMessageEndText("run-msg-witness", "turn-msg-witness", witnessMessageID) }),
		"message_start_reasoning": mustBuildEvent(t, "run-msg-witness", func() (agent.Event, error) { return agent.NewMessageStartReasoning("run-msg-witness", "turn-msg-witness", witnessMessageID) }),
		"message_delta_reasoning": mustBuildEvent(t, "run-msg-witness", func() (agent.Event, error) { return agent.NewMessageDeltaReasoning("run-msg-witness", "turn-msg-witness", witnessMessageID, 0, "x") }),
		"message_end_reasoning":   mustBuildEvent(t, "run-msg-witness", func() (agent.Event, error) { return agent.NewMessageEndReasoning("run-msg-witness", "turn-msg-witness", witnessMessageID) }),
		"tool_start":               mustBuildEvent(t, "run-tool-witness", func() (agent.Event, error) { return agent.NewToolStart("run-tool-witness", "turn-tool-witness", "call-witness", 0, "echo", []byte("{}")) }),
		"tool_progress":            mustBuildEvent(t, "run-tool-witness", func() (agent.Event, error) { return agent.NewToolProgress("run-tool-witness", "turn-tool-witness", "call-witness", 0, 0, []byte("{}")) }),
		"tool_end_success":         mustBuildEvent(t, "run-tool-witness", func() (agent.Event, error) { return agent.NewToolEndSuccess("run-tool-witness", "turn-tool-witness", "call-witness", 0, []byte("ok")) }),
		"tool_end_result_failure":  mustBuildEvent(t, "run-tool-witness", func() (agent.Event, error) { return agent.NewToolEndResultFailure("run-tool-witness", "turn-tool-witness", "call-witness", 0, []byte("nope")) }),
		"tool_end_execution_failure": mustBuildEvent(t, "run-tool-witness", func() (agent.Event, error) { return agent.NewToolEndExecutionFailure("run-tool-witness", "turn-tool-witness", "call-witness", 0, witnessToolFailure) }),
		"permission_decision_required": mustBuildEvent(t, "run-perm-witness", func() (agent.Event, error) { return agent.NewPermissionDecisionRequired("run-perm-witness", "turn-perm-witness", "call-witness", "fs.read", []byte("{}")) }),
		"permission_decision_made":     mustBuildEvent(t, "run-perm-witness", func() (agent.Event, error) { return agent.NewPermissionDecisionMade("run-perm-witness", "turn-perm-witness", "call-witness", agent.PermissionOutcomeAllowOnce, nil, nil) }),
		"permission_resolution_remembered": mustBuildEvent(t, "run-perm-witness", func() (agent.Event, error) { return agent.NewPermissionResolutionRemembered("run-perm-witness", "fs.read", agent.PermissionOutcomeAllowAlways) }),
		"cost_turn":   mustBuildEvent(t, "run-cost-witness", func() (agent.Event, error) { return agent.NewCostTurn("run-cost-witness", "turn-cost-witness", agent.CostLabelEstimate, agent.CostFigures{InputTokens: 1}) }),
		"cost_session": mustBuildEvent(t, "run-cost-witness", func() (agent.Event, error) { return agent.NewCostSession("run-cost-witness", agent.CostLabelFinal, agent.CostFigures{InputTokens: 1}) }),
	}

	for name, event := range nonDelegated {
		parentID, hasParent := event.Parent()
		if hasParent {
			t.Errorf("%s.Parent() = (%q, true), want (RunID(\"\"), false) — non-delegated events MUST NOT carry a parent identifier (R-AEV-003, R-APE-006 negative half, S-APE-052)",
				name, parentID)
		}
		if parentID != agent.RunID("") {
			t.Errorf("%s.Parent() returned non-empty RunID %q with hasParent=false; the zero RunID and false must travel together (R-AEV-003)",
				name, parentID)
		}
	}

	// Positive: only the delegation doors set parent. Confirm via
	// the same sweep on the two delegation kinds.
	delegated := map[string]agent.Event{
		"subagent_started": mustBuildEvent(t, "run-delegation-subagent", func() (agent.Event, error) { return agent.NewSubagentStarted("run-delegation-subagent", "run-delegation-parent", "turn-delegation-witness", "sub-witness") }),
		"subagent_ended":   mustBuildEvent(t, "run-delegation-subagent", func() (agent.Event, error) { return agent.NewSubagentEnded("run-delegation-subagent", "run-delegation-parent", "turn-delegation-witness", "sub-witness") }),
		// AG-04.1's door, the only other place that sets parent.
		"delegated_run_start": mustBuildEvent(t, "run-delegated-child", func() (agent.Event, error) { return agent.NewDelegatedRunStart("run-delegated-child", "run-delegated-parent") }),
	}
	for name, event := range delegated {
		parentID, hasParent := event.Parent()
		if !hasParent {
			t.Errorf("%s.Parent() = (RunID(\"\"), false), want (non-empty, true) — delegation doors MUST set the parent identifier (R-APE-006)",
				name)
		}
		if parentID == agent.RunID("") {
			t.Errorf("%s.Parent() returned empty RunID with hasParent=true; the zero RunID and false must travel together (R-AEV-003)",
				name)
		}
	}
}

// S-APE-052-bis — the parent identifier is typed RunID on the envelope,
// not a proxy under a different name. The reflection pin verifies
// field name AND field type (AG-05 S2 carry-forward).
func TestDelegation_ParentField_ReflectionPin_NameAndType(t *testing.T) {
	t.Parallel()

	eventType := reflect.TypeOf(agent.Event{})

	// Find the unexported "parent" field — its name and type are
	// the structural contract. A reflection-pin check protects the
	// parent identifier from being silently moved or retyped.
	parentField, ok := eventType.FieldByName("parent")
	if !ok {
		t.Fatalf("agent.Event has no unexported field %q — the parent identifier MUST live on the envelope (R-AEV-003, R-APE-006)", "parent")
	}
	if parentField.Type != reflect.TypeOf(agent.RunID("")) {
		t.Errorf("agent.Event.parent has type %v, want agent.RunID — the parent identifier MUST be typed RunID, not a proxy under a different name (R-AEV-003, AG-05 S2 carry-forward)",
			parentField.Type)
	}
}

// mustBuildEvent is the small shared helper that turns a constructor
// closure into an event or fails the test loudly — never silently
// returns a zero event.
func mustBuildEvent(t *testing.T, _ string, build func() (agent.Event, error)) agent.Event {
	t.Helper()
	event, err := build()
	if err != nil {
		t.Fatalf("constructor closure = %v, want nil", err)
	}
	return event
}
