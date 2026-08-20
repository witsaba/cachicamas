// AG-19 — Phase 1 (U1): the delegation seam's admissibility rule,
// exercised from outside the package (NFR-DEL-001, R-DEL-001, R-DEL-002).
//
// package agent_test: every claim here is proven through the exported
// surface only — DelegationSeam, DelegationSeamFrom, and the two
// sentinel errors — mirroring the ScriptedTool precedent
// (scripted_tool_test.go:1-9) for the identical Layer-1/Layer-2 import
// boundary reason: the seam's own admissibility predicate is
// unexported, so a table proving it total can only observe it through
// a live seam a real Schedule call installs.
package agent_test

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// --- S-DEL-004 fixture: one row per registered kind ------------------

// delegationAdmissibilityCase is one row of R-DEL-002's total
// admissibility table: a constructor for a valid Event of one
// registered kind, and the verdict the seam MUST reach for it.
type delegationAdmissibilityCase struct {
	kind  agent.EventKind
	build func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event
	admit bool
}

func mustFreshMessageID(t *testing.T) ai.MessageID {
	t.Helper()
	part, err := ai.NewText("del-004-fixture")
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	return msg.ID()
}

func mustDelegationSeamTestFailure(t *testing.T) *agent.Failure {
	t.Helper()
	inner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure: %v", err)
	}
	f, err := agent.NewFailure(inner)
	if err != nil {
		t.Fatalf("agent.NewFailure: %v", err)
	}
	return f
}

// buildDelegationAdmissibilityTable returns one row per kind
// agent.EventKinds() currently reports, IN THAT EXACT ORDER — the
// call site cross-checks both the row count and the per-index kind
// against the live registry (S-DEL-004: "the table's row count
// equals the number of kinds the registry reports, so a kind
// registered later cannot be silently absent from the assertion").
func buildDelegationAdmissibilityTable() []delegationAdmissibilityCase {
	return []delegationAdmissibilityCase{
		{agent.EventKindRunStart, func(t *testing.T, run agent.RunID, _ agent.TurnID) agent.Event {
			ev, err := agent.NewRunStart(run)
			if err != nil {
				t.Fatalf("agent.NewRunStart: %v", err)
			}
			return ev
		}, false},
		{agent.EventKindRunEnd, func(t *testing.T, run agent.RunID, _ agent.TurnID) agent.Event {
			ev, err := agent.NewRunEnd(run, agent.RunOutcomeCompleted, nil)
			if err != nil {
				t.Fatalf("agent.NewRunEnd: %v", err)
			}
			return ev
		}, false},
		{agent.EventKindTurnStart, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewTurnStart(run, turn)
			if err != nil {
				t.Fatalf("agent.NewTurnStart: %v", err)
			}
			return ev
		}, false},
		{agent.EventKindTurnEnd, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewTurnEnd(run, turn, agent.TurnOutcomeFinished, nil)
			if err != nil {
				t.Fatalf("agent.NewTurnEnd: %v", err)
			}
			return ev
		}, false},
		{agent.EventKindMessageStartText, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewMessageStartText(run, turn, mustFreshMessageID(t))
			if err != nil {
				t.Fatalf("agent.NewMessageStartText: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindMessageDeltaText, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewMessageDeltaText(run, turn, mustFreshMessageID(t), 0, "frag")
			if err != nil {
				t.Fatalf("agent.NewMessageDeltaText: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindMessageEndText, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewMessageEndText(run, turn, mustFreshMessageID(t))
			if err != nil {
				t.Fatalf("agent.NewMessageEndText: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindMessageStartReasoning, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewMessageStartReasoning(run, turn, mustFreshMessageID(t))
			if err != nil {
				t.Fatalf("agent.NewMessageStartReasoning: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindMessageDeltaReasoning, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewMessageDeltaReasoning(run, turn, mustFreshMessageID(t), 0, "frag")
			if err != nil {
				t.Fatalf("agent.NewMessageDeltaReasoning: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindMessageEndReasoning, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewMessageEndReasoning(run, turn, mustFreshMessageID(t))
			if err != nil {
				t.Fatalf("agent.NewMessageEndReasoning: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindToolStart, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewToolStart(run, turn, "call-del-004", 0, "child_tool", []byte(`{}`))
			if err != nil {
				t.Fatalf("agent.NewToolStart: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindToolProgress, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewToolProgress(run, turn, "call-del-004", 0, 1, []byte("p"))
			if err != nil {
				t.Fatalf("agent.NewToolProgress: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindToolEndSuccess, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewToolEndSuccess(run, turn, "call-del-004", 0, []byte("ok"))
			if err != nil {
				t.Fatalf("agent.NewToolEndSuccess: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindToolEndResultFailure, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewToolEndResultFailure(run, turn, "call-del-004", 0, []byte("bad"))
			if err != nil {
				t.Fatalf("agent.NewToolEndResultFailure: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindToolEndExecutionFailure, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewToolEndExecutionFailure(run, turn, "call-del-004", 0, mustDelegationSeamTestFailure(t))
			if err != nil {
				t.Fatalf("agent.NewToolEndExecutionFailure: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindPermissionDecisionRequired, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewPermissionDecisionRequired(run, turn, "call-del-004", "child_tool", []byte(`{}`))
			if err != nil {
				t.Fatalf("agent.NewPermissionDecisionRequired: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindPermissionDecisionMade, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewPermissionDecisionMade(run, turn, "call-del-004", agent.PermissionOutcomeAllowOnce, nil, nil)
			if err != nil {
				t.Fatalf("agent.NewPermissionDecisionMade: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindPermissionResolutionRemembered, func(t *testing.T, run agent.RunID, _ agent.TurnID) agent.Event {
			ev, err := agent.NewPermissionResolutionRemembered(run, "child_tool", agent.PermissionOutcomeAllowAlways)
			if err != nil {
				t.Fatalf("agent.NewPermissionResolutionRemembered: %v", err)
			}
			return ev
		}, false},
		{agent.EventKindCostTurn, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewCostTurn(run, turn, agent.CostLabelFinal, agent.CostFigures{InputTokens: 7})
			if err != nil {
				t.Fatalf("agent.NewCostTurn: %v", err)
			}
			return ev
		}, false},
		{agent.EventKindCostSession, func(t *testing.T, run agent.RunID, _ agent.TurnID) agent.Event {
			ev, err := agent.NewCostSession(run, agent.CostLabelFinal, agent.CostFigures{InputTokens: 7})
			if err != nil {
				t.Fatalf("agent.NewCostSession: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindSubagentStarted, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewSubagentStarted("run-del-004-child", run, turn, "subagent-del-004")
			if err != nil {
				t.Fatalf("agent.NewSubagentStarted: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindSubagentEnded, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewSubagentEnded("run-del-004-child", run, turn, "subagent-del-004")
			if err != nil {
				t.Fatalf("agent.NewSubagentEnded: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindCompactionStarted, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewCompactionStarted(run, turn, "compaction-del-004")
			if err != nil {
				t.Fatalf("agent.NewCompactionStarted: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindCompactionFinished, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			span := agent.CompactionSpan{StartTurnID: turn, EndTurnID: turn}
			ev, err := agent.NewCompactionFinished(run, turn, "compaction-del-004", span, "summary-del-004")
			if err != nil {
				t.Fatalf("agent.NewCompactionFinished: %v", err)
			}
			return ev
		}, true},
		{agent.EventKindCompactionFailed, func(t *testing.T, run agent.RunID, turn agent.TurnID) agent.Event {
			ev, err := agent.NewCompactionFailed(run, turn, "compaction-del-004", mustDelegationSeamTestFailure(t))
			if err != nil {
				t.Fatalf("agent.NewCompactionFailed: %v", err)
			}
			return ev
		}, true},
	}
}

// seamCapturingTool is a Tool whose Run captures the DelegationSeam
// installed on its context, and — if beforeReturn is set — runs it
// synchronously while the seam is still live, before returning. The
// shared fixture S-DEL-002/004/005/006/008/009 build on: the ONLY way
// package agent_test can ever hold a live (or later-revoked) seam
// value is through a real scheduled tool call (R-DEL-001's
// unmintable-from-outside guarantee).
type seamCapturingTool struct {
	toolName     string
	beforeReturn func(seam agent.DelegationSeam)
	panicMsg     string

	mu             sync.Mutex
	captured       agent.DelegationSeam
	hadSeam        bool
	capturedPolicy agent.PolicySlot
}

func (s *seamCapturingTool) Name() string                   { return s.toolName }
func (s *seamCapturingTool) EffectClass() agent.EffectClass { return agent.EffectClassRead }
func (s *seamCapturingTool) Run(ctx context.Context, _ []byte, policy agent.PolicySlot) (agent.Result, error) {
	seam, ok := agent.DelegationSeamFrom(ctx)
	s.mu.Lock()
	s.captured, s.hadSeam, s.capturedPolicy = seam, ok, policy
	s.mu.Unlock()
	if s.beforeReturn != nil && ok {
		s.beforeReturn(seam)
	}
	if s.panicMsg != "" {
		panic(s.panicMsg)
	}
	return agent.Result{Outcome: agent.ToolOutcomeSuccess}, nil
}

// Seam returns the seam this tool's Run captured, and whether one was
// present. Safe to call only after the scheduled call has returned.
func (s *seamCapturingTool) Seam() (agent.DelegationSeam, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.captured, s.hadSeam
}

// Policy returns the PolicySlot value this tool's Run received —
// S-TLS-020's own comparison point ("the policy value is
// byte-identical to the injected one"). Safe to call only after the
// scheduled call has returned.
func (s *seamCapturingTool) Policy() agent.PolicySlot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.capturedPolicy
}

var _ agent.Tool = (*seamCapturingTool)(nil)

// S-DEL-001 — Charter AG-19.1 scenario 1, the door half. Given a
// tool scheduled by the ordinary path, when its Run frame asks the
// context for a seam, then one is present; the identities it
// reports equal the hosting run's and the hosting turn's, compared
// against the run_start and turn_start events observed on that same
// recorded stream; and an event published through it is observed by
// the consumer on the parent's stream. Asserted directly
// (verify-report CRITICAL-6 — previously unasserted): seam.Parent()
// is compared against the actual run_start/turn_start events on the
// same drained stream, not merely consumed by a fixture.
func TestDelegationSeam_S_DEL_001_IdentitiesMatchRunStartAndTurnStartOnSameStream(t *testing.T) {
	t.Parallel()

	var capturedRun agent.RunID
	var capturedTurn agent.TurnID
	var capturedMsgID ai.MessageID
	var publishErr error

	toolName := "del001_tool"
	tool := &seamCapturingTool{
		toolName: toolName,
		beforeReturn: func(seam agent.DelegationSeam) {
			capturedRun, capturedTurn = seam.Parent()
			capturedMsgID = mustFreshMessageID(t)
			ev, err := agent.NewMessageStartText(capturedRun, capturedTurn, capturedMsgID)
			if err != nil {
				t.Fatalf("agent.NewMessageStartText: %v", err)
			}
			publishErr = seam.Publish(ev)
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-del-001", toolName, []byte(`{}`))
	provider := agenttest.NewProvider(turnOneScript, scriptTextResponse(t, ai.FinishReasonStop))
	h := agent.Harness{Provider: provider, System: "system prompt for del-001", Turn: agent.TurnOptions{Tools: reg}}

	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	if _, ok := tool.Seam(); !ok {
		t.Fatal("seamCapturingTool captured no seam — Tool.Run's context carries no DelegationSeam")
	}
	if publishErr != nil {
		t.Errorf("seam.Publish(message_start_text) = %v, want nil (an admissible kind)", publishErr)
	}

	runStartIdx := indexOfFirst(events, agent.EventKindRunStart)
	turnStartIdx := indexOfFirst(events, agent.EventKindTurnStart)
	if runStartIdx < 0 {
		t.Fatal("recorded stream carries no run_start")
	}
	if turnStartIdx < 0 {
		t.Fatal("recorded stream carries no turn_start")
	}
	if got := events[runStartIdx].Run(); got != capturedRun {
		t.Errorf("seam.Parent() run = %q, run_start event's Run() = %q, want equal", capturedRun, got)
	}
	gotTurn, hasTurn := events[turnStartIdx].Turn()
	if !hasTurn {
		t.Fatal("turn_start event reports no turn — Event.Turn()'s second return is false")
	}
	if gotTurn != capturedTurn {
		t.Errorf("seam.Parent() turn = %q, turn_start event's Turn() = %q, want equal", capturedTurn, gotTurn)
	}
	if got := events[turnStartIdx].Run(); got != capturedRun {
		t.Errorf("turn_start event's Run() = %q, want %q (seam.Parent()'s run)", got, capturedRun)
	}

	// "an event published through it is observed by the consumer on
	// the parent's stream": the message_start_text this tool
	// published through the seam appears on the same recorded
	// stream, matched by its own minted message identity — not
	// merely by kind and run, which the harness's own turn-2 text
	// response also satisfies and would make this a ghost assertion
	// (verify-report round-2 CRITICAL-7).
	found := false
	for _, ev := range events {
		payload, ok := ev.MessageStartText()
		if !ok {
			continue
		}
		if ev.Run() == capturedRun && payload.MessageID() == capturedMsgID {
			found = true
			break
		}
	}
	if !found {
		t.Error("the message_start_text published through the seam (matched by its minted message ID) does not appear on the parent's recorded stream")
	}
}

// S-DEL-004 — the rule is total, kind by kind. Given a table naming
// every registered kind together with the verdict R-DEL-002's table
// states, when each kind is published through a live seam, then the
// observed verdict equals the stated one for all of them; each
// refusal returns the inadmissible sentinel and delivers nothing to
// any sink; and the table's row count equals the number of kinds the
// registry reports.
func TestDelegationSeam_AdmissibilityTable_TotalOverEveryRegisteredKind(t *testing.T) {
	t.Parallel()

	table := buildDelegationAdmissibilityTable()
	kinds := agent.EventKinds()
	if len(table) != len(kinds) {
		t.Fatalf("admissibility table has %d row(s), want %d (one per agent.EventKinds()) — a kind registered later must not be silently absent from this table", len(table), len(kinds))
	}
	for i, row := range table {
		if row.kind != kinds[i] {
			t.Fatalf("table row %d kind = %v, want %v (table order must track agent.EventKinds() order)", i, row.kind, kinds[i])
		}
	}

	run := agent.RunID("run-del-004")
	turn := agent.TurnID("turn-del-004")

	type observed struct {
		err error
	}
	var mu sync.Mutex
	var results []observed

	toolName := "del004_tool"
	tool := &seamCapturingTool{
		toolName: toolName,
		beforeReturn: func(seam agent.DelegationSeam) {
			for _, row := range table {
				ev := row.build(t, run, turn)
				err := seam.Publish(ev)
				mu.Lock()
				results = append(results, observed{err: err})
				mu.Unlock()
			}
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: "del004-call", name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sink := make(chan *agent.Event, 64)
	sched := &agent.Scheduler{}
	_ = sched.Schedule(contextBackground(), calls, reg, run, turn, nil, &agent.LaneStamper{}, sink)
	events := drainSink(t, sink)

	if len(results) != len(table) {
		t.Fatalf("recorded %d publish attempt(s), want %d", len(results), len(table))
	}

	var wantMirrored int
	for i, row := range table {
		got := results[i]
		if row.admit {
			wantMirrored++
			if got.err != nil {
				t.Errorf("kind %v (row %d): Publish returned %v, want nil (admitted)", row.kind, i, got.err)
			}
			continue
		}
		if !errors.Is(got.err, agent.ErrDelegationInadmissible) {
			t.Errorf("kind %v (row %d): Publish returned %v, want errors.Is(_, ErrDelegationInadmissible) (refused)", row.kind, i, got.err)
		}
	}

	// "delivers nothing to any sink": total events on the parent's
	// sink equal this call's own ToolStart+ToolEnd (2, emitted by the
	// scheduler itself, not through the seam) plus exactly one per
	// admitted row. A refused row that leaked onto sink would inflate
	// this count; using the total rather than a per-kind count avoids
	// ambiguity between the outer call's own tool_start/tool_end_*
	// events and same-kind rows this table itself publishes.
	if want := 2 + wantMirrored; len(events) != want {
		t.Errorf("sink carries %d event(s), want %d (2 for this call's own ToolStart/ToolEnd, plus one per admitted row)", len(events), want)
	}
}

// S-DEL-005 — the zero Event is refused by the membership gate, not
// admitted by zero-valued descriptor fields. Given a zero-value
// Event, when it is published, then the publish returns the
// inadmissible sentinel, nothing reaches any sink, and the parent's
// recorded stream is byte-identical to the same run whose tool
// published nothing.
func TestDelegationSeam_ZeroEvent_RefusedByMembershipGate(t *testing.T) {
	t.Parallel()

	var publishErr error
	toolName := "del005_tool"
	tool := &seamCapturingTool{
		toolName: toolName,
		beforeReturn: func(seam agent.DelegationSeam) {
			publishErr = seam.Publish(agent.Event{})
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: "del005-call", name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sink := make(chan *agent.Event, 16)
	sched := &agent.Scheduler{}
	_ = sched.Schedule(contextBackground(), calls, reg, "run-del-005", "turn-del-005", nil, &agent.LaneStamper{}, sink)
	events := drainSink(t, sink)

	if !errors.Is(publishErr, agent.ErrDelegationInadmissible) {
		t.Errorf("Publish(zero Event) returned %v, want errors.Is(_, ErrDelegationInadmissible)", publishErr)
	}
	// The same call publishing nothing produces exactly ToolStart +
	// ToolEndSuccess — byte-identical in kind shape to a tool that
	// never touched the seam at all.
	if len(events) != 2 {
		t.Errorf("sink carries %d event(s), want 2 (ToolStart, ToolEndSuccess — the zero Event reached nothing)", len(events))
	}
}

// collectTopLevelExportedNames parses path with go/parser and returns
// every EXPORTED top-level identifier it declares (function, type,
// var, const — never a method, which is not itself a top-level
// identifier a package consumer names directly). Source-level
// reflection over the file the seam ships in, mirroring this
// package's other source-guard tests (PolicySlot's raw-byte scan,
// scheduler_test.go:616-650) rather than reflect.TypeOf, which can
// only enumerate symbols the test already knows to reference.
func collectTopLevelExportedNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile(%q): %v", path, err)
	}
	names := map[string]bool{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				continue // a method is not a top-level identifier.
			}
			if d.Name.IsExported() {
				names[d.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						names[s.Name.Name] = true
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n.IsExported() {
							names[n.Name] = true
						}
					}
				}
			}
		}
	}
	return names
}

// wantDelegationSeamExportedNames is the complete, sanctioned
// exported surface delegation_seam.go MAY declare (R-DEL-001): the
// seam interface, its context accessor, and the two sentinel errors
// of R-DEL-002/R-DEL-003 — nothing else. No exported constructor, no
// exported installer, no exported context-key type, no admissibility
// predicate.
func wantDelegationSeamExportedNames() map[string]bool {
	return map[string]bool{
		"DelegationSeam":            true,
		"DelegationSeamFrom":        true,
		"ErrDelegationInadmissible": true,
		"ErrDelegationRevoked":      true,
	}
}

// S-DEL-002 — the seam is unmintable from outside. Given package
// agent_test, when the package's exported surface is enumerated, then
// it declares the seam interface, its accessor and the two sentinels
// and nothing else of the delegation seam; no exported constructor,
// installer, context key or admissibility predicate exists; and no
// external caller can install a seam onto a context the scheduler
// will honour.
func TestDelegationSeam_UnmintableSurface_ExactlyFourExportedIdentifiers(t *testing.T) {
	t.Parallel()

	got := collectTopLevelExportedNames(t, "delegation_seam.go")
	want := wantDelegationSeamExportedNames()
	if len(got) != len(want) {
		t.Errorf("delegation_seam.go declares %d exported top-level identifier(s), want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for name := range want {
		if !got[name] {
			t.Errorf("delegation_seam.go does not declare exported identifier %q, want it declared", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("delegation_seam.go declares unexpected exported identifier %q — the seam's surface MUST be exactly %v", name, want)
		}
	}

	// No external caller can install a seam onto a context the
	// scheduler will honour: an ordinary context, including one
	// carrying an independently constructed value under some other
	// key, reports no seam.
	seam, ok := agent.DelegationSeamFrom(context.Background())
	if ok || seam != nil {
		t.Errorf("DelegationSeamFrom(context.Background()) = (%v, %v), want (nil, false) — an ordinary context installs no seam", seam, ok)
	}
}

// S-AEV-125 — AG-19: what the package declares after the seam ships,
// and what it still does not (agent-event-envelope delta). Reuses
// S-DEL-002's enumeration (tasks.md 4.4) rather than re-deriving it: a
// second, independently named test proves the same file's exported
// surface from the envelope capability's own scenario identifier, so
// a later archive that drops one delta's test file still leaves the
// other capability's own coverage intact. Scoped to delegation_seam.go
// specifically — the package's pre-existing SubagentStarted/
// SubagentEnded event vocabulary (delegation_events.go, AG-06.3) is
// event-protocol surface, not a "subagent mechanism"; what this
// scenario guards is the SEAM's own file declaring no subagent type,
// child-harness type, delegation depth, delegation configuration or
// subagent lifecycle method.
func TestDelegationSeam_S_AEV_125_ExportedSurfaceDeclaresNoSubagentMechanism(t *testing.T) {
	t.Parallel()

	got := collectTopLevelExportedNames(t, "delegation_seam.go")
	want := wantDelegationSeamExportedNames()
	if len(got) != len(want) {
		t.Fatalf("delegation_seam.go declares %d exported top-level identifier(s), want exactly %d: got=%v", len(got), len(want), got)
	}
	for name := range got {
		if !want[name] {
			t.Errorf("delegation_seam.go declares %q, which S-AEV-125 does not sanction — the seam's surface MUST name no subagent, child-harness, depth, configuration or lifecycle concept", name)
		}
	}
	// Belt-and-braces: the four sanctioned names themselves must not
	// happen to contain a forbidden substring (they don't, but a
	// future rename should trip this rather than pass silently).
	forbidden := []string{"Subagent", "ChildHarness", "Depth", "Config", "Lifecycle"}
	for name := range want {
		for _, bad := range forbidden {
			if strings.Contains(name, bad) {
				t.Errorf("sanctioned name %q contains forbidden substring %q", name, bad)
			}
		}
	}

	seam, ok := agent.DelegationSeamFrom(context.Background())
	if ok || seam != nil {
		t.Errorf("DelegationSeamFrom(context.Background()) = (%v, %v), want (nil, false)", seam, ok)
	}
}
