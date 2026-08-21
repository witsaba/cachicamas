// AG-23 — strict-TDD tests for ScriptedTool (R-L3H-003, S-L3H-014,
// S-L3H-015).
package apptest_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/apptest"
)

// Given a scripted tool, when Run is called twice with distinct
// arguments, then Invocations reports 2, RecordedArgs reports both
// argument sets in call order, and each call's return value is exactly
// what the script produced for it.
func TestScriptedTool_RecordsInvocationsAndArgsInOrder(t *testing.T) {
	t.Parallel()

	var seenArgs [][]byte
	script := func(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
		seenArgs = append(seenArgs, args)
		return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: args}, nil
	}
	tool := apptest.NewScriptedTool("read_tool", agent.EffectClassRead, script)

	res1, err := tool.Run(context.Background(), []byte(`{"n":1}`), nil)
	if err != nil {
		t.Fatalf("Run #1 error = %v, want nil", err)
	}
	if string(res1.Content) != `{"n":1}` {
		t.Errorf("Run #1 Content = %q, want %q", res1.Content, `{"n":1}`)
	}

	res2, err := tool.Run(context.Background(), []byte(`{"n":2}`), nil)
	if err != nil {
		t.Fatalf("Run #2 error = %v, want nil", err)
	}
	if string(res2.Content) != `{"n":2}` {
		t.Errorf("Run #2 Content = %q, want %q", res2.Content, `{"n":2}`)
	}

	if got := tool.Invocations(); got != 2 {
		t.Fatalf("Invocations() = %d, want 2", got)
	}

	recorded := tool.RecordedArgs()
	if len(recorded) != 2 {
		t.Fatalf("RecordedArgs() returned %d entr(y/ies), want 2", len(recorded))
	}
	if string(recorded[0]) != `{"n":1}` || string(recorded[1]) != `{"n":2}` {
		t.Errorf("RecordedArgs() = %q, want [%q %q] in call order", recorded, `{"n":1}`, `{"n":2}`)
	}

	// RecordedArgs returns a copy: mutating the returned slice must not
	// affect a later call's own reading.
	recorded[0][0] = 'X'
	if got := tool.RecordedArgs()[0][0]; got == 'X' {
		t.Error("mutating RecordedArgs()' returned slice mutated the tool's own recorded state — want an independent copy")
	}
}

// Name() and EffectClass() report exactly the constructor's arguments.
func TestScriptedTool_NameAndEffectClass(t *testing.T) {
	t.Parallel()

	tool := apptest.NewScriptedTool("write_tool", agent.EffectClassMutating, func(context.Context, []byte, agent.PolicySlot) (agent.Result, error) {
		return agent.Result{}, nil
	})
	if got := tool.Name(); got != "write_tool" {
		t.Errorf("Name() = %q, want %q", got, "write_tool")
	}
	if got := tool.EffectClass(); got != agent.EffectClassMutating {
		t.Errorf("EffectClass() = %v, want EffectClassMutating", got)
	}
}

// The PolicySlot value is forwarded byte-for-byte (by identity) to the
// script, mirroring the "Layer 2 never reads it" seam agent_test's own
// ScriptedTool already proves one layer down.
func TestScriptedTool_ForwardsPolicySlotByIdentity(t *testing.T) {
	t.Parallel()

	type marker struct{ n int }
	want := &marker{n: 7}

	var got agent.PolicySlot
	tool := apptest.NewScriptedTool("policy_tool", agent.EffectClassRead, func(_ context.Context, _ []byte, policy agent.PolicySlot) (agent.Result, error) {
		got = policy
		return agent.Result{Outcome: agent.ToolOutcomeSuccess}, nil
	})
	if _, err := tool.Run(context.Background(), nil, want); err != nil {
		t.Fatalf("Run error = %v, want nil", err)
	}
	if gotPtr, ok := got.(*marker); !ok || gotPtr != want {
		t.Errorf("script observed policy = %#v, want the exact %#v pointer forwarded untouched", got, want)
	}
}

// Compile-time interface pin, re-asserted at the test's own call site:
// ScriptedTool satisfies agent.Tool.
var _ agent.Tool = (*apptest.ScriptedTool)(nil)
