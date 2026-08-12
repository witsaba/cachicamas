// Tests for AG-05.2 — the tool execution family (R-AMT-005, R-AMT-006,
// R-AMT-007). package agent_test: NFR-AMT-001.
//
// The tool family is the second kind-set to exercise PlacementTurn
// (R-AEV-012, AD-2). The three end states are typed outcomes
// distinct by kind (R-AMT-006, S-AMT-050), and the ordinal that
// correlates events to call-issuance order is a payload field, not
// an envelope field (R-AMT-007, AD-3, R-13 → doc 0002 AI-30).
package agent_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-AMT-040 — a ToolStart event carries the tool call identity, the
// tool name, and the arguments as distinct fields; any ToolProgress
// for the same call carries an integer index (R-AMT-005).
func TestToolStart_CarriesIdentityNameAndArguments_ToolProgressIndexed(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-040")
	turn := agent.TurnID("turn-amt-040")

	start, err := agent.NewToolStart(run, turn, "call-amt-040", 0, "echo", []byte(`{"hi":1}`))
	if err != nil {
		t.Fatalf("agent.NewToolStart() error = %v, want nil", err)
	}
	payload, ok := start.ToolStart()
	if !ok {
		t.Fatal("start.ToolStart() reported no payload on an event of its own kind")
	}
	if got := payload.CallID(); got != "call-amt-040" {
		t.Errorf("payload.CallID() = %q, want %q", got, "call-amt-040")
	}
	if got := payload.Name(); got != "echo" {
		t.Errorf("payload.Name() = %q, want %q", got, "echo")
	}
	if got := payload.Ordinal(); got != 0 {
		t.Errorf("payload.Ordinal() = %d, want 0", got)
	}
	// Arguments carried as []byte (encoding/json is forbidden by
	// Layer 2's import boundary, ADR 0005 § D1 row 2). Their content
	// is preserved exactly.
	if got := string(payload.Arguments()); got != `{"hi":1}` {
		t.Errorf("payload.Arguments() = %q, want %q", got, `{"hi":1}`)
	}

	// ToolProgress indexed
	progress, err := agent.NewToolProgress(run, turn, "call-amt-040", 0, 7, []byte("p7"))
	if err != nil {
		t.Fatalf("agent.NewToolProgress() error = %v, want nil", err)
	}
	pPayload, ok := progress.ToolProgress()
	if !ok {
		t.Fatal("progress.ToolProgress() reported no payload on an event of its own kind")
	}
	if got := pPayload.Index(); got != 7 {
		t.Errorf("pPayload.Index() = %d, want 7", got)
	}
	if got := pPayload.CallID(); got != "call-amt-040" {
		t.Errorf("pPayload.CallID() = %q, want %q", got, "call-amt-040")
	}
}

// S-AMT-050 — tool events for one call with each of the three end
// states are reachable as typed values; no payload-convention
// inference is required to tell them apart (R-AMT-006). Each end
// kind carries its own Outcome() and is identifiable by Kind()
// without parsing text.
func TestToolEnd_ThreeOutcomes_TypedAndDistinctByKind(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-050")
	turn := agent.TurnID("turn-amt-050")

	// success
	success, err := agent.NewToolEndSuccess(run, turn, "call-amt-050-success", 0, []byte(`"ok"`))
	if err != nil {
		t.Fatalf("NewToolEndSuccess error = %v, want nil", err)
	}
	if _, ok := success.ToolEndSuccess(); !ok {
		t.Fatal("success.ToolEndSuccess() reported no payload on an event of its own kind")
	}
	// result-failure
	resultFail, err := agent.NewToolEndResultFailure(run, turn, "call-amt-050-result", 1, []byte(`"nope"`))
	if err != nil {
		t.Fatalf("NewToolEndResultFailure error = %v, want nil", err)
	}
	if _, ok := resultFail.ToolEndResultFailure(); !ok {
		t.Fatal("resultFail.ToolEndResultFailure() reported no payload on an event of its own kind")
	}
	// execution-failure
	inner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure() error = %v, want nil", err)
	}
	failWrap, err := agent.NewFailure(inner)
	if err != nil {
		t.Fatalf("agent.NewFailure() error = %v, want nil", err)
	}
	execFail, err := agent.NewToolEndExecutionFailure(run, turn, "call-amt-050-exec", 2, failWrap)
	if err != nil {
		t.Fatalf("NewToolEndExecutionFailure error = %v, want nil", err)
	}
	if _, ok := execFail.ToolEndExecutionFailure(); !ok {
		t.Fatal("execFail.ToolEndExecutionFailure() reported no payload on an event of its own kind")
	}

	// Cross-kind accessor separation: each accessor returns false for
	// events of the other two kinds, proving no payload-convention
	// inference is required.
	if _, ok := success.ToolEndResultFailure(); ok {
		t.Error("success.ToolEndResultFailure() returned ok on a success event — kinds are not distinct")
	}
	if _, ok := success.ToolEndExecutionFailure(); ok {
		t.Error("success.ToolEndExecutionFailure() returned ok on a success event")
	}
	if _, ok := resultFail.ToolEndSuccess(); ok {
		t.Error("resultFail.ToolEndSuccess() returned ok on a result-failure event")
	}
	if _, ok := resultFail.ToolEndExecutionFailure(); ok {
		t.Error("resultFail.ToolEndExecutionFailure() returned ok on a result-failure event")
	}
	if _, ok := execFail.ToolEndSuccess(); ok {
		t.Error("execFail.ToolEndSuccess() returned ok on an execution-failure event")
	}
	if _, ok := execFail.ToolEndResultFailure(); ok {
		t.Error("execFail.ToolEndResultFailure() returned ok on an execution-failure event")
	}

	// Outcome() returns the typed value — Success, ResultFailure,
	// ExecutionFailure — distinct.
	if got := outcomeOfSuccess(success); got != agent.ToolOutcomeSuccess {
		t.Errorf("success.Outcome() = %v, want ToolOutcomeSuccess", got)
	}
	if got := outcomeOfResultFailure(resultFail); got != agent.ToolOutcomeResultFailure {
		t.Errorf("resultFail.Outcome() = %v, want ToolOutcomeResultFailure", got)
	}
	if got := outcomeOfExecutionFailure(execFail); got != agent.ToolOutcomeExecutionFailure {
		t.Errorf("execFail.Outcome() = %v, want ToolOutcomeExecutionFailure", got)
	}
}

// outcomeOfSuccess, outcomeOfResultFailure, outcomeOfExecutionFailure
// are tiny typed accessors over the three end variants' Outcome()
// methods, kept here so the test file does not invent a single
// "ToolEnd" method (which does not exist — the three kinds are
// distinct by Kind, not by a shared accessor, exactly the
// R-AMT-006 typed-outcome invariant S-AMT-050 proves).
func outcomeOfSuccess(e agent.Event) agent.ToolOutcome {
	p, _ := e.ToolEndSuccess()
	return p.Outcome()
}
func outcomeOfResultFailure(e agent.Event) agent.ToolOutcome {
	p, _ := e.ToolEndResultFailure()
	return p.Outcome()
}
func outcomeOfExecutionFailure(e agent.Event) agent.ToolOutcome {
	p, _ := e.ToolEndExecutionFailure()
	return p.Outcome()
}

// S-AMT-051 — **(bite)** a tool_end_execution_failure constructed
// without a typed [Failure] is REJECTED. The typed-failure wrap
// (R-AEV-008) is required, not optional, for this outcome.
func TestToolEndExecutionFailure_WithoutFailure_Rejected(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-051")
	turn := agent.TurnID("turn-amt-051")

	_, err := agent.NewToolEndExecutionFailure(run, turn, "call-amt-051", 0, nil)
	if err == nil {
		t.Fatal("agent.NewToolEndExecutionFailure(nil) error = nil, want a rejection — the typed-failure wrap is required (R-AEV-008, S-AMT-051)")
	}
	if !errors.Is(err, ai.ErrEmpty) {
		t.Errorf("NewToolEndExecutionFailure(nil) error = %v, want errors.Is to match ai.ErrEmpty", err)
	}
}

// S-AMT-060 — a tool event's payload carries a call ordinal
// correlating to the call's issuance order, independent of completion
// order; the ordinal is a payload field, NOT an envelope field
// (R-AMT-007, AD-3, R-13 → doc 0002 AI-30).
//
// Three calls are issued in order 0, 1, 2. Their start events carry
// the matching ordinals 0, 1, 2. Even when completions arrive out
// of order (call 2 ends before call 0), each event's payload
// ordinal still correlates to issuance order.
func TestToolCallOrdinal_CorrelatesToIssuanceRegardlessOfCompletion(t *testing.T) {
	t.Parallel()

	run := agent.RunID("run-amt-060")
	turn := agent.TurnID("turn-amt-060")

	// Issue three calls in order.
	s0, err := agent.NewToolStart(run, turn, "call-0", 0, "a", []byte("{}"))
	if err != nil {
		t.Fatalf("NewToolStart(0) error = %v", err)
	}
	s1, err := agent.NewToolStart(run, turn, "call-1", 1, "b", []byte("{}"))
	if err != nil {
		t.Fatalf("NewToolStart(1) error = %v", err)
	}
	s2, err := agent.NewToolStart(run, turn, "call-2", 2, "c", []byte("{}"))
	if err != nil {
		t.Fatalf("NewToolStart(2) error = %v", err)
	}

	// Completions arrive out of order: call-2 first, then call-0,
	// then call-1.
	e2, err := agent.NewToolEndSuccess(run, turn, "call-2", 2, []byte("ok"))
	if err != nil {
		t.Fatalf("NewToolEndSuccess(2) error = %v", err)
	}
	e0, err := agent.NewToolEndSuccess(run, turn, "call-0", 0, []byte("ok"))
	if err != nil {
		t.Fatalf("NewToolEndSuccess(0) error = %v", err)
	}
	e1, err := agent.NewToolEndSuccess(run, turn, "call-1", 1, []byte("ok"))
	if err != nil {
		t.Fatalf("NewToolEndSuccess(1) error = %v", err)
	}

	// Each start carries the issuance ordinal on its payload.
	for _, pair := range []struct {
		event    agent.Event
		wantCall string
		wantOrd  uint32
	}{
		{s0, "call-0", 0},
		{s1, "call-1", 1},
		{s2, "call-2", 2},
	} {
		p, ok := pair.event.ToolStart()
		if !ok {
			t.Fatalf("ToolStart accessor failed for %s", pair.wantCall)
		}
		if got := p.CallID(); got != pair.wantCall {
			t.Errorf("CallID() = %q, want %q", got, pair.wantCall)
		}
		if got := p.Ordinal(); got != pair.wantOrd {
			t.Errorf("Ordinal() = %d, want %d", got, pair.wantOrd)
		}
	}

	// Each completion also carries the issuance ordinal on its payload
	// — completion order does not change the ordinal, which is keyed
	// to issuance, not completion.
	for _, pair := range []struct {
		event    agent.Event
		wantCall string
		wantOrd  uint32
	}{
		{e2, "call-2", 2},
		{e0, "call-0", 0},
		{e1, "call-1", 1},
	} {
		p, ok := pair.event.ToolEndSuccess()
		if !ok {
			t.Fatalf("ToolEndSuccess accessor failed for %s", pair.wantCall)
		}
		if got := p.CallID(); got != pair.wantCall {
			t.Errorf("CallID() = %q, want %q", got, pair.wantCall)
		}
		if got := p.Ordinal(); got != pair.wantOrd {
			t.Errorf("Ordinal() = %d, want %d — payload ordinal correlates to issuance order, not completion order (R-AMT-007, AD-3)", got, pair.wantOrd)
		}
	}

	// The envelope identity (Event.Run / Event.Turn) is byte-unchanged
	// from AG-04's sequence.go (the AG-05 bet). The ordinal is purely
	// a payload field — proven by the fact that no Field on the
	// envelope struct (event.go's Event struct) carries an ordinal.
	// This is asserted structurally: payload ordinals exist, and a
	// hypothetical envelope ordinal accessor does not.
	pt := reflect.TypeOf(agent.Event{})
	for i := 0; i < pt.NumField(); i++ {
		f := pt.Field(i)
		if containsFold(f.Name, "ordinal") {
			t.Errorf("Event struct has field %q — ordinal must be a payload field, not an envelope field (R-AMT-007, AD-3)", f.Name)
		}
	}

	// Sanity: the byte slice arguments survive a round trip through
	// the public surface.
	if got := string(argumentsOfStart(s0)); got != "{}" {
		t.Errorf("Arguments round-trip = %q, want %q", got, "{}")
	}

	// Tool arguments are []byte (not json.RawMessage) because the
	// encoding/json package is forbidden by Layer 2's import boundary
	// (ADR 0005 § D1 row 2). Documented in tool_event.go.
	var probe map[string]any
	if err := json.Unmarshal(argumentsOfStart(s0), &probe); err != nil {
		t.Fatalf("tool-start arguments must remain valid JSON for downstream consumers; json.Unmarshal error = %v", err)
	}
	if !reflect.DeepEqual(probe, map[string]any{}) {
		t.Errorf("arguments decoded to %v, want empty object", probe)
	}
}

// argumentsOfStart is a typed accessor over the ToolStart payload's
// Arguments() — kept in this file to avoid a one-off accessor method
// on Event just for tests.
func argumentsOfStart(e agent.Event) []byte {
	p, _ := e.ToolStart()
	return p.Arguments()
}
