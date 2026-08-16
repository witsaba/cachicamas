// AG-10 remediation (verify-report S4) — the no-op pass-through
// PermissionPolicy design.md:84 promises for tests that need a
// non-nil policy but do not want any of the gate's suspend / deny /
// modify behavior.
//
// design.md's own wording places this decorator "in agenttest". It
// cannot actually live there: `agenttest` (backend/agent/src/agenttest)
// is Layer 1's own external-consumer proof and importable testing
// library — its doc.go states, under "Dependency-free (R-STK-009)",
// that the package "imports only the standard library and this
// module's own src/ai", enforced as a build failure by
// src/ai/import_boundary_test.go's Layer 1 closure guard (which scans
// src/ai, src/agenttest and src/handoff together and denies any
// import of a sibling layer). A PermissionPolicy implementation
// necessarily references agent.PermissionPolicy, agent.PermissionVerdict
// and agent.PermissionOutcome — Layer 2 types — so it cannot compile
// inside agenttest without breaking that guard.
//
// This is not a new problem: scripted_tool_test.go's ScriptedTool hit
// the identical constraint for AG-09 and resolved it the same way —
// "Layer 1's agenttest package cannot import the agent package (ADR
// 0005 § D1 row 1 — Layer 1 must not import Layer 2), so the scripted
// tool is a test-only file in the agent package's external test
// surface." NoOpPermissionPolicy follows the same precedent: it lives
// here, in package agent_test, exported so any test file in this
// package can use it without writing its own trivial stub.
package agent_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// NoOpPermissionPolicy is a non-nil agent.PermissionPolicy that never
// gates anything: Resolve always returns AllowOnce (synchronous,
// zero decision_required events) and Remember always returns false
// (never records a rule, zero resolution_remembered events). It is
// NOT the same as a nil policy — a nil policy bypasses the gate
// entirely (zero permission_decision_made events either); this type
// exercises the gate's synchronous AllowOnce path for real, emitting
// permission_decision_made for every call, while still requiring
// nothing from the caller beyond "a policy exists".
//
// Reach for this whenever a test needs SOME PermissionPolicy but has
// no interest in the gate's suspend / deny / modify behavior.
type NoOpPermissionPolicy struct{}

// Resolve always returns AllowOnce.
func (NoOpPermissionPolicy) Resolve(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
	return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
}

// Remember always returns false — it never asks the caller to
// silence subsequent identical calls.
func (NoOpPermissionPolicy) Remember(_ context.Context, _ string, _ agent.PermissionOutcome) bool {
	return false
}

// Compile-time guard: NoOpPermissionPolicy must satisfy
// agent.PermissionPolicy.
var _ agent.PermissionPolicy = NoOpPermissionPolicy{}

// TestNoOpPermissionPolicy_AllowsEverySynchronously proves the
// no-op decorator is genuinely wired through a real Schedule call,
// not merely a compiling stub: zero decision_required events (never
// defers), the call executes exactly once, and — the property that
// distinguishes it from a nil policy — a real
// permission_decision_made{AllowOnce} event IS on the stream, since
// a non-nil policy always goes through the gate even when every
// verdict is the identity outcome.
func TestNoOpPermissionPolicy_AllowsEverySynchronously(t *testing.T) {
	t.Parallel()

	tool := EchoScriptedTool("noop_tool", agent.EffectClassRead)
	reg := newMapRegistry(map[string]agent.Tool{"noop_tool": tool})
	calls := []scheduledPermissionCall{{id: "noop_0", name: "noop_tool", args: []byte(`{}`)}}

	results, events := runPermissionSchedulerAndCollect(
		&agent.Scheduler{MaxConcurrentReads: 4}, calls, reg, NoOpPermissionPolicy{},
	)

	if got := countByKind(events)[agent.EventKindPermissionDecisionRequired]; got != 0 {
		t.Errorf("decision_required count = %d, want 0 (NoOpPermissionPolicy never defers)", got)
	}
	madeAllowOnce := 0
	for _, ev := range events {
		if made, ok := ev.PermissionDecisionMade(); ok && made.Outcome() == agent.PermissionOutcomeAllowOnce {
			madeAllowOnce++
		}
	}
	if madeAllowOnce != 1 {
		t.Errorf("decision_made{AllowOnce} count = %d, want 1 (a non-nil policy drives the gate for real, unlike a nil policy's total bypass)", madeAllowOnce)
	}
	if len(results) != 1 {
		t.Fatalf("Schedule returned %d results, want 1", len(results))
	}
	if results[0].Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("results[0].Outcome = %v, want Success", results[0].Outcome)
	}
	if inv := tool.Invocations(); inv != 1 {
		t.Errorf("noop_tool invocations = %d, want 1", inv)
	}
}
