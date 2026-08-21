// Package apptest — tool.go: the scriptable agent.Tool (AG-23, design
// AD-3). Mirrors src/agent's own in-package agent_test.ScriptedTool one
// layer up, minus the wall clock: this kit performs no I/O and uses no
// wall-clock timestamp (R-L3H-002, R-L3H-004), so invocation order is
// available from the invocation count and RecordedArgs' own slice order,
// never from a time.Now() reading.
package apptest

import (
	"context"
	"sync"

	"github.com/cachicamas/backend/agent/src/agent"
)

// ScriptedTool is an importable agent.Tool whose Run delegates to a
// consumer-supplied script. It records its own invocation count and
// arguments (byte-exact copies) for inspection, and forwards the
// PolicySlot value to script untouched (Layer 2's own "never read"
// promise for that seam — this kit does not read it either).
type ScriptedTool struct {
	name   string
	effect agent.EffectClass
	script func(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error)

	mu           sync.Mutex
	invocations  int
	recordedArgs [][]byte
}

// NewScriptedTool constructs a scripted tool named name, of effect class
// effect, whose Run delegates to script.
func NewScriptedTool(name string, effect agent.EffectClass, script func(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error)) *ScriptedTool {
	return &ScriptedTool{name: name, effect: effect, script: script}
}

// Name implements agent.Tool.
func (s *ScriptedTool) Name() string { return s.name }

// EffectClass implements agent.Tool.
func (s *ScriptedTool) EffectClass() agent.EffectClass { return s.effect }

// Run implements agent.Tool. Records the invocation and a fresh copy of
// its arguments before delegating to script.
func (s *ScriptedTool) Run(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error) {
	s.mu.Lock()
	s.invocations++
	s.recordedArgs = append(s.recordedArgs, append([]byte(nil), args...))
	s.mu.Unlock()
	return s.script(ctx, args, policy)
}

// Invocations returns the number of times Run has been called.
func (s *ScriptedTool) Invocations() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.invocations
}

// RecordedArgs returns a fresh copy of every call's arguments, in
// invocation order.
func (s *ScriptedTool) RecordedArgs() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.recordedArgs))
	for i, a := range s.recordedArgs {
		out[i] = append([]byte(nil), a...)
	}
	return out
}

// Compile-time guard: ScriptedTool must satisfy agent.Tool. A
// non-conforming implementation fails to compile here, not at runtime
// under a Schedule call.
var _ agent.Tool = (*ScriptedTool)(nil)
