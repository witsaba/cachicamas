// Tests for the Layer 2 audit fix (branch
// fix/agent-layer2-audit-findings) — behavioral coverage for the
// stream-contract validator's documented rejection branches that
// previously had zero test executions anywhere in the module
// (R-AEV-004, R-AEV-005, S-AEV-110). A separate file rather than an
// edit to stream_check_test.go, deliberately: S-LSK-031's filter guard
// pins stream_check_test.go as never released from the substrate
// freeze, and a file this change CREATES needs no release. package
// agent_test: NFR-AEV-001.
package agent_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// R-AEV-004 — a turn-end whose run bracket was never opened is REJECTED,
// naming the escaped close. This exercises the BracketRoleClosesTurn
// arm's own run-open gate (stream_check.go), distinct from S-AEV-042's
// turn-START escapes: after run-end the nothing-follows-the-terminal
// rule fires first, so the only sequence that reaches this gate is a
// turn-end BEFORE run-start.
func TestCheckStream_TurnEndBeforeRunStart_RejectedNamingEscapedClose(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	turnEnd, err := agent.NewTurnEnd("run-148", "turn-1", agent.TurnOutcomeFinished, nil)
	if err != nil {
		t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(turnEnd)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(turn-end before run-start) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrMisplaced) {
		t.Errorf("agent.CheckStream(turn-end before run-start) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
	}
	requireViolationPosition(t, report.Violation(), 0)
}

// R-AEV-005 — a turn-end inside an open run bracket is REJECTED when no
// turn is open at all, and separately when its TurnID names a turn other
// than the one that is open: closing a turn that is not open, under
// either mechanism, is a misplaced close.
func TestCheckStream_TurnEndClosingNoOpenTurn_RejectedNamingMisplacedClose(t *testing.T) {
	t.Parallel()

	t.Run("no turn open", func(t *testing.T) {
		t.Parallel()
		var lane agent.LaneStamper
		run := agent.RunID("run-152a")
		start, err := agent.NewRunStart(run)
		if err != nil {
			t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
		}
		turnEnd, err := agent.NewTurnEnd(run, "turn-1", agent.TurnOutcomeFinished, nil)
		if err != nil {
			t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
		}
		events := []agent.Event{lane.Stamp(start), lane.Stamp(turnEnd)}

		report := agent.CheckStream(events)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(turn-end with no open turn) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("agent.CheckStream(turn-end with no open turn) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
		}
		requireViolationPosition(t, report.Violation(), 1)
	})

	t.Run("mismatched turn identity", func(t *testing.T) {
		t.Parallel()
		var lane agent.LaneStamper
		run := agent.RunID("run-152b")
		start, err := agent.NewRunStart(run)
		if err != nil {
			t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
		}
		turnStart, err := agent.NewTurnStart(run, "turn-1")
		if err != nil {
			t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
		}
		turnEnd, err := agent.NewTurnEnd(run, "turn-2", agent.TurnOutcomeFinished, nil)
		if err != nil {
			t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
		}
		events := []agent.Event{lane.Stamp(start), lane.Stamp(turnStart), lane.Stamp(turnEnd)}

		report := agent.CheckStream(events)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(turn-end naming the wrong turn) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("agent.CheckStream(turn-end naming the wrong turn) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
		}
		requireViolationPosition(t, report.Violation(), 2)
	})
}

// R-AEV-004 — a BracketRoleNone kind before run-start is REJECTED: every
// event other than the one opening the run bracket must occur while the
// run bracket is open. Both placements walk the same gate — a
// PlacementTurn kind (message_start_text) and the one PlacementRun kind
// (cost_session) — proving the rejection is the run-open rule, which is
// checked before any turn-placement rule.
func TestCheckStream_BracketRoleNoneBeforeRunStart_RejectedNamingEscape(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-158")

	t.Run("message_start_text (PlacementTurn)", func(t *testing.T) {
		t.Parallel()
		var lane agent.LaneStamper
		start, err := agent.NewMessageStartText(run, "turn-1", mustMessageID(t))
		if err != nil {
			t.Fatalf("agent.NewMessageStartText() error = %v, want nil", err)
		}
		events := []agent.Event{lane.Stamp(start)}

		report := agent.CheckStream(events)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(message_start_text before run-start) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("agent.CheckStream(message_start_text before run-start) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
		}
		requireViolationPosition(t, report.Violation(), 0)
	})

	t.Run("cost_session (PlacementRun)", func(t *testing.T) {
		t.Parallel()
		var lane agent.LaneStamper
		session, err := agent.NewCostSession(run, agent.CostLabelFinal, agent.CostFigures{})
		if err != nil {
			t.Fatalf("agent.NewCostSession() error = %v, want nil", err)
		}
		events := []agent.Event{lane.Stamp(session)}

		report := agent.CheckStream(events)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(cost_session before run-start) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("agent.CheckStream(cost_session before run-start) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
		}
		requireViolationPosition(t, report.Violation(), 0)
	})
}

// S-AEV-110 — a hand-built sequence containing a message_start_text
// outside an open turn is REJECTED naming the PlacementTurn rule
// (R-AEV-012, R-AMT-009). Extended over ALL 11 AG-05 message/tool kinds:
// each is placed between turn-end and the next turn-start — inside the
// open run bracket, so the only rule it can break is placement — and
// each must be rejected. This is the behavioral half of
// TestEventKinds_AG05AllRegisterPlacementTurn (event_registry_test.go),
// which pins only the registered NAMES: a kind whose registry row
// dropped Placement: PlacementTurn would pass the name scan there and
// fail here.
func TestCheckStream_AG05KindBetweenTurns_RejectedNamingPlacement(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-110")
	turn := agent.TurnID("turn-1")
	msgID := mustMessageID(t)
	failure := mustAgentFailure(t)

	cases := []struct {
		wantKind agent.EventKind
		build    func() (agent.Event, error)
	}{
		{agent.EventKindMessageStartText, func() (agent.Event, error) {
			return agent.NewMessageStartText(run, turn, msgID)
		}},
		{agent.EventKindMessageDeltaText, func() (agent.Event, error) {
			return agent.NewMessageDeltaText(run, turn, msgID, 0, "fragment")
		}},
		{agent.EventKindMessageEndText, func() (agent.Event, error) {
			return agent.NewMessageEndText(run, turn, msgID)
		}},
		{agent.EventKindMessageStartReasoning, func() (agent.Event, error) {
			return agent.NewMessageStartReasoning(run, turn, msgID)
		}},
		{agent.EventKindMessageDeltaReasoning, func() (agent.Event, error) {
			return agent.NewMessageDeltaReasoning(run, turn, msgID, 0, "fragment")
		}},
		{agent.EventKindMessageEndReasoning, func() (agent.Event, error) {
			return agent.NewMessageEndReasoning(run, turn, msgID)
		}},
		{agent.EventKindToolStart, func() (agent.Event, error) {
			return agent.NewToolStart(run, turn, "call-110", 0, "echo", []byte(`{"hi":1}`))
		}},
		{agent.EventKindToolProgress, func() (agent.Event, error) {
			return agent.NewToolProgress(run, turn, "call-110", 0, 0, []byte("p0"))
		}},
		{agent.EventKindToolEndSuccess, func() (agent.Event, error) {
			return agent.NewToolEndSuccess(run, turn, "call-110", 0, []byte(`"ok"`))
		}},
		{agent.EventKindToolEndResultFailure, func() (agent.Event, error) {
			return agent.NewToolEndResultFailure(run, turn, "call-110", 0, []byte(`"nope"`))
		}},
		{agent.EventKindToolEndExecutionFailure, func() (agent.Event, error) {
			return agent.NewToolEndExecutionFailure(run, turn, "call-110", 0, failure)
		}},
	}
	if want := 11; len(cases) != want {
		t.Fatalf("the case table lists %d AG-05 kinds, want all %d message/tool kinds", len(cases), want)
	}

	for _, tc := range cases {
		t.Run(tc.wantKind.String(), func(t *testing.T) {
			t.Parallel()

			misplaced, err := tc.build()
			if err != nil {
				t.Fatalf("constructing the %v event: error = %v, want nil", tc.wantKind, err)
			}
			if got := misplaced.Kind(); got != tc.wantKind {
				t.Fatalf("built event's Kind() = %v, want %v — the table row and the constructor disagree", got, tc.wantKind)
			}

			var lane agent.LaneStamper
			start, err := agent.NewRunStart(run)
			if err != nil {
				t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
			}
			turnStart, err := agent.NewTurnStart(run, turn)
			if err != nil {
				t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
			}
			turnEnd, err := agent.NewTurnEnd(run, turn, agent.TurnOutcomeFinished, nil)
			if err != nil {
				t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
			}
			events := []agent.Event{
				lane.Stamp(start), lane.Stamp(turnStart), lane.Stamp(turnEnd),
				lane.Stamp(misplaced), // between turn-end and any next turn-start
			}

			report := agent.CheckStream(events)
			if report.Violation() == nil {
				t.Fatalf("agent.CheckStream(%v between turns) = no violation, want a PlacementTurn rejection", tc.wantKind)
			}
			if !errors.Is(report.Violation(), ai.ErrMisplaced) {
				t.Errorf("agent.CheckStream(%v between turns) = %v, want errors.Is to match ai.ErrMisplaced", tc.wantKind, report.Violation())
			}
			requireViolationPosition(t, report.Violation(), 3)
		})
	}
}

// mustMessageID mints a valid ai.MessageID off a minimal assistant
// message, for tests that need a message identity and are not themselves
// testing the message surface (that is message_text_test.go's job,
// AG-05.1).
func mustMessageID(t *testing.T) ai.MessageID {
	t.Helper()
	part, err := ai.NewText("stream-check-fixture")
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
	return msgID
}
