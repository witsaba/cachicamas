// AG-23 — strict-TDD tests for ScriptedPermissionPolicy (R-L3H-003,
// S-L3H-014..016).
package apptest_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/apptest"
)

// toolCallForTest builds an ai.ToolCall value through the public
// constructor route (ai.NewToolCall -> Part.ToolCall()), mirroring
// src/agent's own scheduler_test.go callsToAICalls helper.
func toolCallForTest(t *testing.T, id, name string) ai.ToolCall {
	t.Helper()
	part, err := ai.NewToolCall(id, name, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall(%q, %q) error = %v, want nil", id, name, err)
	}
	call, ok := part.ToolCall()
	if !ok {
		t.Fatalf("ai.NewToolCall(%q, %q) returned a non-tool-call part", id, name)
	}
	return call
}

// S-L3H-016 (FIFO order). Given a verdict queue of two entries, when
// three permission decisions are requested, then the first two resolve
// in queue order, the third resolves to the stated default, the
// exhaustion flag reads true, and the run does not wedge.
func TestScriptedPermissionPolicy_FIFOOrder_ThenExhaustedDefaultNeverWedges(t *testing.T) {
	t.Parallel()

	deny := agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny}
	allowAlways := agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowAlways}
	policy := apptest.NewScriptedPermissionPolicy(deny, allowAlways)

	if policy.Exhausted() {
		t.Fatal("Exhausted() = true before any Resolve call, want false")
	}

	call1 := toolCallForTest(t, "call-1", "read_one")
	got1 := policy.Resolve(context.Background(), call1)
	if got1.Outcome != agent.PermissionOutcomeDeny {
		t.Errorf("Resolve #1 outcome = %v, want Deny (queue order)", got1.Outcome)
	}

	call2 := toolCallForTest(t, "call-2", "read_two")
	got2 := policy.Resolve(context.Background(), call2)
	if got2.Outcome != agent.PermissionOutcomeAllowAlways {
		t.Errorf("Resolve #2 outcome = %v, want AllowAlways (queue order)", got2.Outcome)
	}

	if policy.Exhausted() {
		t.Fatal("Exhausted() = true after exactly 2 calls against a 2-entry queue, want false")
	}

	// The third call — the queue is now exhausted. This call MUST NOT
	// block or panic: it must fall back to the stated default.
	call3 := toolCallForTest(t, "call-3", "read_three")
	got3 := policy.Resolve(context.Background(), call3)
	if got3.Outcome != agent.PermissionOutcomeAllowOnce {
		t.Errorf("Resolve #3 (exhausted) outcome = %v, want AllowOnce (the stated default)", got3.Outcome)
	}
	if !policy.Exhausted() {
		t.Error("Exhausted() = false after a 3rd call against a 2-entry queue, want true (latched)")
	}

	// Resolved() reports every call asked, in order — full consumption
	// is assertable rather than merely inferred from a hang that never
	// happened.
	resolved := policy.Resolved()
	if len(resolved) != 3 {
		t.Fatalf("Resolved() returned %d call(s), want 3", len(resolved))
	}
	wantNames := []string{"read_one", "read_two", "read_three"}
	for i, want := range wantNames {
		if got := resolved[i].Name(); got != want {
			t.Errorf("Resolved()[%d].Name() = %q, want %q", i, got, want)
		}
	}
}

// S-L3H-015 (compile-time interface pin, exercised at runtime too).
// Given a verdict queue of a single AllowOnce entry, when one decision
// is requested, then it resolves without touching the exhaustion latch.
func TestScriptedPermissionPolicy_SingleEntry_ResolvesUnexhausted(t *testing.T) {
	t.Parallel()

	policy := apptest.NewScriptedPermissionPolicy(agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce})
	call := toolCallForTest(t, "call-solo", "read_solo")
	got := policy.Resolve(context.Background(), call)
	if got.Outcome != agent.PermissionOutcomeAllowOnce {
		t.Errorf("Resolve outcome = %v, want AllowOnce", got.Outcome)
	}
	if policy.Exhausted() {
		t.Error("Exhausted() = true after consuming exactly the queued entry count, want false")
	}
}

// Given RememberReturns set true, when Remember is called, then it
// returns true; given the zero value (false, the default), when Remember
// is called, then it returns false — the scripted policy commits nothing
// by default, matching a policy with no persistent memory.
func TestScriptedPermissionPolicy_Remember_ReturnsConfiguredValue(t *testing.T) {
	t.Parallel()

	t.Run("default false", func(t *testing.T) {
		t.Parallel()
		policy := apptest.NewScriptedPermissionPolicy()
		if got := policy.Remember(context.Background(), "read_one", agent.PermissionOutcomeAllowAlways); got {
			t.Error("Remember() = true with RememberReturns at its zero value, want false")
		}
	})

	t.Run("configured true", func(t *testing.T) {
		t.Parallel()
		policy := apptest.NewScriptedPermissionPolicy()
		policy.RememberReturns = true
		if got := policy.Remember(context.Background(), "read_one", agent.PermissionOutcomeAllowAlways); !got {
			t.Error("Remember() = false with RememberReturns = true, want true")
		}
	})
}

// Compile-time interface pin, re-asserted at the test's own call site:
// ScriptedPermissionPolicy satisfies agent.PermissionPolicy.
var _ agent.PermissionPolicy = (*apptest.ScriptedPermissionPolicy)(nil)
