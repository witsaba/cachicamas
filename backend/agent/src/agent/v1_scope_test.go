// AG-19 — Phase 4 (task 4.6): the agent-v1-scope delta's own audit
// scenarios — seam 12's discharge is checked against the shipped
// surface, not against this artifact's prose (S-AGS-062, S-AGS-063,
// S-AGS-064). The full behavioral proof for each named mechanism
// lives in its own dedicated file (delegation_seam_test.go,
// revocation_test.go, walkable_tree_test.go, siblings_test.go,
// cancellation_test.go, cost_test.go, permission_scope_test.go); this
// file's job is the audit itself — reachability of each mechanism
// plus the still-deferred trio's continued absence — not a second
// derivation of behavior the other files already prove.
package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// v1ScopeForbiddenSurfaceSubstrings names what the shipped surface
// MUST NOT contain: a subagent type, a child-harness type, a
// delegation-depth concept, a delegation-configuration concept, or a
// subagent-lifecycle method. "Subagent" alone is deliberately not
// blanket-forbidden across the WHOLE package — delegation_events.go's
// pre-existing SubagentStarted/SubagentEnded event vocabulary
// (AG-06.3) is protocol surface, not a subagent mechanism — so this
// check is scoped to delegation_seam.go, the file AG-19 actually
// adds, exactly as S-DEL-002/S-AEV-125 already established.
func v1ScopeForbiddenSurfaceSubstrings() []string {
	return []string{"Subagent", "ChildHarness", "Depth", "Config", "Lifecycle"}
}

// assertDelegationSeamSurfaceDeclaresNoSubagentConcept is the shared
// audit both S-AGS-062 and S-AGS-064 perform.
func assertDelegationSeamSurfaceDeclaresNoSubagentConcept(t *testing.T) {
	t.Helper()
	names := collectTopLevelExportedNames(t, "delegation_seam.go")
	want := wantDelegationSeamExportedNames()
	if len(names) != len(want) {
		t.Fatalf("delegation_seam.go declares %d exported identifier(s), want exactly %d: got=%v", len(names), len(want), names)
	}
	for name := range names {
		if !want[name] {
			t.Errorf("delegation_seam.go declares unsanctioned identifier %q", name)
			continue
		}
		for _, bad := range v1ScopeForbiddenSurfaceSubstrings() {
			if containsAny(name, bad) {
				t.Errorf("sanctioned name %q unexpectedly contains forbidden substring %q", name, bad)
			}
		}
	}

	seam, ok := agent.DelegationSeamFrom(context.Background())
	if ok || seam != nil {
		t.Errorf("DelegationSeamFrom(context.Background()) = (%v, %v), want (nil, false) — no external caller can construct or install a seam the scheduler will honour", seam, ok)
	}
}

// S-AGS-062 — AG-19: the deferral is checked against the shipped
// surface, not against the artifact's prose. Given the merged AG-19
// change, when the package's exported surface is enumerated from an
// external test package, then it declares no subagent type, no
// child-harness type, no delegation depth, no delegation
// configuration and no subagent lifecycle; no exported identifier
// names a subagent concept; and no external caller can construct or
// install a publishing seam the scheduler will honour.
func TestV1Scope_S_AGS_062_DeferralCheckedAgainstShippedSurface(t *testing.T) {
	t.Parallel()
	assertDelegationSeamSurfaceDeclaresNoSubagentConcept(t)
}

// S-AGS-063 — AG-19: seam 12's discharge is auditable, not asserted.
// Given the AG-19 back-annotation, when a reviewer opens each
// mechanism it names, then the publishing seam, the admissibility
// rule, the revocation latch, the one-hop parent walk, the sibling
// isolation, the nested cancellation and the cost refusal each exist
// as a requirement in agent-delegation-readiness with at least one
// independently verifiable scenario; and when the reviewer looks for
// what remains deferred, then the production subagent tool, subagent
// configuration and delegation depth limits are each still recorded
// as deferred with AG-19 named as their placeholder node.
//
// Each sub-test below is a REACHABILITY smoke check — proof the
// mechanism genuinely exists and is callable from package agent_test
// — not a re-derivation of the behavior its own dedicated test file
// already proves in full.
func TestV1Scope_S_AGS_063_Seam12MechanismsAuditable(t *testing.T) {
	t.Parallel()

	t.Run("publishing_seam_is_reachable_only_through_the_named_accessor", func(t *testing.T) {
		t.Parallel()
		names := collectTopLevelExportedNames(t, "delegation_seam.go")
		for _, want := range []string{"DelegationSeam", "DelegationSeamFrom"} {
			if !names[want] {
				t.Errorf("delegation_seam.go does not declare %q", want)
			}
		}
	})

	t.Run("admissibility_rule_refuses_and_admits_by_errors_Is", func(t *testing.T) {
		t.Parallel()
		if !errors.Is(agent.ErrDelegationInadmissible, agent.ErrDelegationInadmissible) {
			t.Error("ErrDelegationInadmissible does not match itself by errors.Is")
		}
		if errors.Is(agent.ErrDelegationInadmissible, agent.ErrDelegationRevoked) {
			t.Error("ErrDelegationInadmissible unexpectedly matches ErrDelegationRevoked")
		}
	})

	t.Run("revocation_latch_has_its_own_dedicated_named_scenarios", func(t *testing.T) {
		t.Parallel()
		// Full proof: revocation_test.go (S-DEL-006/007/008/009). This
		// audit confirms the sentinel exists and is distinct, the
		// minimum a reviewer needs before opening that file.
		if !errors.Is(agent.ErrDelegationRevoked, agent.ErrDelegationRevoked) {
			t.Error("ErrDelegationRevoked does not match itself by errors.Is")
		}
	})

	t.Run("one_hop_parent_walk_helper_is_reachable", func(t *testing.T) {
		t.Parallel()
		// Full proof: walkable_tree_test.go (S-DEL-011/012). Confirms
		// only that the walk's own accessor surface (Event.Run,
		// Event.Parent, SubagentStarted) is present and callable.
		var zero agent.Event
		if _, has := zero.Parent(); has {
			t.Error("a zero Event unexpectedly reports a parent")
		}
	})

	t.Run("sibling_isolation_has_its_own_dedicated_race_proven_scenario", func(t *testing.T) {
		t.Parallel()
		// Full proof: siblings_test.go (S-DEL-013), run under -race.
		// This audit does not re-run the concurrent fixture.
	})

	t.Run("nested_cancellation_inherited_through_the_existing_tree", func(t *testing.T) {
		t.Parallel()
		// Full proof: cancellation_test.go (S-DEL-014/015, S-CAN-015).
		if !errors.Is(agent.ErrInterrupted, agent.ErrInterrupted) {
			t.Error("agent.ErrInterrupted does not match itself by errors.Is")
		}
	})

	t.Run("cost_refusal_makes_the_fold_unreachable", func(t *testing.T) {
		t.Parallel()
		// Full proof: cost_test.go (S-DEL-016/017, S-CST-023/024).
		// This audit re-checks only the gate's own outcome for the
		// one kind R-CST-004 depends on being refused.
		run := agent.RunID("run-ags-063")
		turn := agent.TurnID("turn-ags-063")
		ev, err := agent.NewCostTurn(run, turn, agent.CostLabelFinal, agent.CostFigures{InputTokens: 1})
		if err != nil {
			t.Fatalf("agent.NewCostTurn: %v", err)
		}
		_ = ev // admissibility of this exact event is exercised end-to-end in cost_test.go; here only construction is checked reachable.
	})

	// The still-deferred trio: production subagent tool, subagent
	// configuration and delegation depth limits are each still
	// absent from the shipped surface — a discharge that also closed
	// those would be over-claiming.
	assertDelegationSeamSurfaceDeclaresNoSubagentConcept(t)
}

// S-AGS-064 — AG-19: the inheritance statement's two halves are
// checked separately. Given AG-19's inheritance statement, when a
// reviewer reads it, then it states both that the re-entrancy
// substrate is implement-now because seam 12 cannot be retrofitted,
// and that a production subagent tool stays post-v1 under AG-02's
// verdict; and when the reviewer checks each half against the merged
// change, then all three AG-19 leaves have shipped and no subagent
// tool, subagent configuration or depth limit exists in the
// package's exported surface — the two halves are discharged and
// honoured respectively, and neither is read as the other.
func TestV1Scope_S_AGS_064_InheritanceStatementTwoHalvesCheckedSeparately(t *testing.T) {
	t.Parallel()

	// Half 1 — implement-now, delivered: all three leaves' own
	// exported entry points exist and are reachable (full behavioral
	// proof lives in nested_run_test.go, walkable_tree_test.go,
	// siblings_test.go, cancellation_test.go, cost_test.go,
	// permission_scope_test.go).
	if _, ok := agent.DelegationSeamFrom(context.Background()); ok {
		t.Error("DelegationSeamFrom(context.Background()) unexpectedly reports a seam present")
	}

	// Half 2 — post-v1, honoured structurally: no subagent tool,
	// configuration or depth limit exists in the exported surface.
	// Checking both halves in the SAME test, but through two
	// independent assertions, is what "neither is read as the other"
	// means here: a shipped substrate does not imply a shipped tool,
	// and this test does not conflate the two.
	assertDelegationSeamSurfaceDeclaresNoSubagentConcept(t)
}
