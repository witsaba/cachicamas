// AG-23 — the kit's own determinism proof (R-L3H-004, S-L3H-019..022):
// a repeat-run structural-equality test, scoped to modulo minted
// identities, driven against a REAL agent.Harness (not a mock) — this
// IS the kit's own proof surface (WU-3's stated runtime harness).
//
// Byte-equal event sequences are NOT the claim: Run and Turn identities
// mint from process-global atomic counters (harness.go, loop.go), so two
// identical runs cannot produce byte-equal event sequences. The claim
// stated and proven here is narrower and stronger for being narrower:
// identical event-KIND sequences, identical identity-INDEPENDENT payload
// projections, and both drains reported clean — with the identity
// values themselves asserted to DIFFER, so a vacuous projection that
// never looks at identity at all would be caught rather than silently
// tolerated.
package apptest_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/apptest"
)

// detFirstMessage builds a minimal user message, mirroring src/agent's
// own loop_test.go firstMessage helper (this package cannot reach that
// unexported-to-agent_test helper, so it is rebuilt here from the same
// public ai constructors).
func detFirstMessage(t *testing.T) ai.Message {
	t.Helper()
	part, err := ai.NewText("hi")
	if err != nil {
		t.Fatalf("ai.NewText error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage error = %v, want nil", err)
	}
	return msg
}

// detScriptToolCallResponse and detScriptTextResponse mirror src/agent's
// own scriptToolCallResponse/scriptTextResponse helpers (harness_test.go,
// loop_test.go) — rebuilt here because this package lives outside
// src/agent's own test package and cannot reach those unexported-scope
// helpers, only the public ai/agenttest constructors they are built from.
func detScriptToolCallResponse(t *testing.T, callID, toolName string, args []byte) agenttest.Script {
	t.Helper()
	start, err := ai.NewToolCallStart(1, callID, toolName)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart error = %v, want nil", err)
	}
	delta, err := ai.NewToolCallDelta(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta error = %v, want nil", err)
	}
	end, err := ai.NewToolCallEnd(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd error = %v, want nil", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion error = %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

func detScriptTextResponse(t *testing.T, text string) agenttest.Script {
	t.Helper()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart error = %v, want nil", err)
	}
	delta, err := ai.NewTextDelta(1, text)
	if err != nil {
		t.Fatalf("ai.NewTextDelta error = %v, want nil", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd error = %v, want nil", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion error = %v, want nil", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// detProjection is one event's identity-independent projection: its
// kind, plus — for the kinds this scenario's claim names ("outcomes,
// tool names, message texts, verdict order", design AD-3) — the one
// payload field worth comparing. Every other kind projects to its kind
// alone, which is still meaningful: it is the "identical event-kind
// sequences" half of the claim. No projection ever reads a RunID,
// TurnID or ai.MessageID.
type detProjection struct {
	kind  agent.EventKind
	extra string
}

func detProjectEvent(ev agent.Event) detProjection {
	switch ev.Kind() {
	case agent.EventKindRunEnd:
		if re, ok := ev.RunEnd(); ok {
			return detProjection{kind: ev.Kind(), extra: re.Outcome().String()}
		}
	case agent.EventKindTurnEnd:
		if te, ok := ev.TurnEnd(); ok {
			return detProjection{kind: ev.Kind(), extra: te.Outcome().String()}
		}
	case agent.EventKindToolStart:
		if ts, ok := ev.ToolStart(); ok {
			return detProjection{kind: ev.Kind(), extra: ts.Name()}
		}
	case agent.EventKindToolEndSuccess:
		if te, ok := ev.ToolEndSuccess(); ok {
			return detProjection{kind: ev.Kind(), extra: string(te.Result())}
		}
	case agent.EventKindPermissionDecisionMade:
		if pd, ok := ev.PermissionDecisionMade(); ok {
			return detProjection{kind: ev.Kind(), extra: pd.Outcome().String()}
		}
	case agent.EventKindMessageDeltaText:
		if md, ok := ev.MessageDeltaText(); ok {
			return detProjection{kind: ev.Kind(), extra: md.Fragment()}
		}
	}
	return detProjection{kind: ev.Kind()}
}

// detRunScenario drives one full run of the kit's own proof surface: a
// tool call (scripted, mutating effect, permission-gated by an AllowOnce
// verdict) followed by a text completion. Returns the drained events,
// the clean report, and the minted run/turn identities the caller
// compares SEPARATELY from the structural projection.
func detRunScenario(t *testing.T) (events []agent.Event, report agent.StreamReport, runID agent.RunID, turnIDs []agent.TurnID) {
	t.Helper()

	toolScript := func(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
		return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: args}, nil
	}
	tool := apptest.NewScriptedTool("write_det_001", agent.EffectClassMutating, toolScript)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"write_det_001": tool})
	policy := apptest.NewScriptedPermissionPolicy(agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce})

	turnOneScript := detScriptToolCallResponse(t, "call-det-001", "write_det_001", []byte(`{"path":"x"}`))
	turnTwoScript := detScriptTextResponse(t, "done")
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{
		Provider: provider,
		System:   "system prompt for apptest determinism",
		Turn:     agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
	}
	sink := make(chan *agent.Event)
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		events, report = apptest.DrainAndCheck(sink)
	}()

	if _, _, err := h.Run(context.Background(), detFirstMessage(t), sink); err != nil {
		t.Fatalf("Run error = %v, want nil", err)
	}
	<-drainDone

	if report.Violation() != nil {
		t.Fatalf("DrainAndCheck report.Violation() = %v, want nil (clean report)", report.Violation())
	}

	seen := make(map[agent.TurnID]bool)
	for _, ev := range events {
		runID = ev.Run()
		if tid, ok := ev.Turn(); ok && !seen[tid] {
			seen[tid] = true
			turnIDs = append(turnIDs, tid)
		}
	}
	return events, report, runID, turnIDs
}

// S-L3H-019, S-L3H-020 (structural equality modulo minted identities,
// PLUS the anti-vacuity identity-difference assertion). Given one
// script, when it is driven twice in the same process and both streams
// are drained, then the two event-kind sequences are equal, the two
// identity-independent projections are equal, both reports are clean —
// and the minted run/turn identities themselves DIFFER between the two
// runs, proving the projection's exclusion of identity is deliberate
// rather than a comparison that happened never to look at it.
func TestApptestKit_RepeatRun_StructurallyEqualModuloMintedIdentities(t *testing.T) {
	t.Parallel()

	events1, _, runID1, turnIDs1 := detRunScenario(t)
	events2, _, runID2, turnIDs2 := detRunScenario(t)

	if len(events1) != len(events2) {
		t.Fatalf("run 1 produced %d event(s), run 2 produced %d — want equal event-kind sequence lengths", len(events1), len(events2))
	}
	for i := range events1 {
		p1, p2 := detProjectEvent(events1[i]), detProjectEvent(events2[i])
		if p1.kind != p2.kind {
			t.Fatalf("event[%d] kind: run 1 = %v, run 2 = %v — want identical event-kind sequences", i, p1.kind, p2.kind)
		}
		if p1.extra != p2.extra {
			t.Errorf("event[%d] (%v) identity-independent projection: run 1 = %q, run 2 = %q — want equal", i, p1.kind, p1.extra, p2.extra)
		}
	}

	// The anti-vacuity check: the identities themselves must DIFFER.
	if runID1 == runID2 {
		t.Fatalf("run 1 and run 2 minted the SAME RunID (%q) — the two Harness.Run calls did not actually run independently, or the process-global counter did not advance; the identity-exclusion above would be untestable", runID1)
	}
	if len(turnIDs1) == 0 || len(turnIDs2) == 0 {
		t.Fatal("one of the two runs produced zero turn identities — want at least one turn per run")
	}
	if len(turnIDs1) != len(turnIDs2) {
		t.Fatalf("run 1 produced %d turn identit(y/ies), run 2 produced %d — want equal turn counts (same script)", len(turnIDs1), len(turnIDs2))
	}
	for i := range turnIDs1 {
		if turnIDs1[i] == turnIDs2[i] {
			t.Fatalf("turn[%d]: run 1 and run 2 minted the SAME TurnID (%q) — want the minted identities to differ between runs", i, turnIDs1[i])
		}
	}
}

// S-L3H-021 (defeat, bite). Given the comparison widened to whole event
// STRING values (which embed the minted RunID, per Event.String's own
// documented "run=<run>" rendering) instead of the identity-independent
// projection above, when the same repeat-run scenario is compared, then
// it FAILS on the minted identities — proving the projection above is
// doing real work and the passing assertion is not vacuous. Recorded,
// then this bite is not reverted because it plants nothing in
// production or shared test code — it is a self-contained scratch
// comparison local to this one test function.
func TestApptestKit_RepeatRun_WidenedToWholeEventValues_FailsOnMintedIdentities(t *testing.T) {
	t.Parallel()

	events1, _, _, _ := detRunScenario(t)
	events2, _, _, _ := detRunScenario(t)

	if len(events1) != len(events2) {
		t.Fatalf("run 1 produced %d event(s), run 2 produced %d — want equal lengths for this bite to be meaningful", len(events1), len(events2))
	}

	sawDifference := false
	for i := range events1 {
		if events1[i].String() != events2[i].String() {
			sawDifference = true
			break
		}
	}
	if !sawDifference {
		t.Fatal("bite S-L3H-021 did not reproduce: widening the comparison to whole Event.String() values (which embed the minted RunID) found no difference across two independent runs — want at least one differing rendering, proving the narrower projection is not vacuous")
	}
}
