// AG-09 — the scripted tool: an in-test `agent.Tool` whose `Run` is
// configurable per-instance, recording policy byte-for-byte.
//
// The tool lives in `package agent_test` (external posture,
// NFR-TLS-001). Layer 1's `agenttest` package cannot import the
// `agent` package (ADR 0005 § D1 row 1 — Layer 1 must not import
// Layer 2), so the scripted tool is a test-only file in the
// agent package's external test surface.
package agent_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
)

// ScriptedTool is an in-memory `agent.Tool` whose `Run` is
// configurable per-instance. The test sets `Script` to a function
// that returns the desired `Result` (or panics, for the
// panic-containment test). The tool records its invocations and
// the `PolicySlot` value the loop / scheduler forwarded to it
// (byte-exact, R-TLS-002).
type ScriptedTool struct {
	Name_  string
	Effect agent.EffectClass
	Script func(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error)

	// Recorded state (test inspection surface):
	mu             sync.Mutex
	invocations    int64
	recordedArgs   [][]byte
	recordedPolicy []agent.PolicySlot
	recordedResult []agent.Result
	startAt        []int64
}

// NewScriptedTool constructs a fresh scripted tool whose `Run`
// returns the given `Result` and `nil` error.
func NewScriptedTool(name string, effect agent.EffectClass, result agent.Result) *ScriptedTool {
	return &ScriptedTool{
		Name_:  name,
		Effect: effect,
		Script: func(_ context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
			return result, nil
		},
	}
}

// EchoScriptedTool constructs a scripted tool that echoes its
// arguments as the result's `Content`. Used by the end-to-end
// dispatch test where the loop's transcript reconstruction
// asserts the tool result reaches the message parts.
func EchoScriptedTool(name string, effect agent.EffectClass) *ScriptedTool {
	return &ScriptedTool{
		Name_:  name,
		Effect: effect,
		Script: func(_ context.Context, args []byte, _ agent.PolicySlot) (agent.Result, error) {
			return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: args}, nil
		},
	}
}

// BlockingScriptedTool constructs a scripted tool that blocks
// until `release` is closed. Used by the end-to-end test to
// ensure the scheduler's goroutines don't race with the loop's
// rejoin.
func BlockingScriptedTool(name string, effect agent.EffectClass, release <-chan struct{}) *ScriptedTool {
	return &ScriptedTool{
		Name_:  name,
		Effect: effect,
		Script: func(_ context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
			<-release
			return agent.Result{Outcome: agent.ToolOutcomeSuccess, Content: []byte("released")}, nil
		},
	}
}

// PanickingScriptedTool constructs a scripted tool whose `Run`
// panics. Used by the panic-containment test (S-TLS-011).
func PanickingScriptedTool(name string, effect agent.EffectClass, msg string) *ScriptedTool {
	return &ScriptedTool{
		Name_:  name,
		Effect: effect,
		Script: func(_ context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
			panic(msg)
		},
	}
}

// Name implements `agent.Tool`.
func (s *ScriptedTool) Name() string { return s.Name_ }

// EffectClass implements `agent.Tool`.
func (s *ScriptedTool) EffectClass() agent.EffectClass { return s.Effect }

// Run implements `agent.Tool`. Records the call's arguments and
// policy value byte-for-byte, then delegates to `Script`.
func (s *ScriptedTool) Run(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error) {
	atomic.AddInt64(&s.invocations, 1)
	s.mu.Lock()
	s.startAt = append(s.startAt, time.Now().UnixNano())
	s.recordedArgs = append(s.recordedArgs, append([]byte(nil), args...))
	s.recordedPolicy = append(s.recordedPolicy, policy)
	s.mu.Unlock()
	res, err := s.Script(ctx, args, policy)
	s.mu.Lock()
	s.recordedResult = append(s.recordedResult, res)
	s.mu.Unlock()
	return res, err
}

// Invocations returns the number of times `Run` was called.
func (s *ScriptedTool) Invocations() int64 { return atomic.LoadInt64(&s.invocations) }

// RecordedArgs returns a fresh copy of the call's arguments, in
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

// RecordedPolicies returns the policy values, in invocation
// order. The slice contains the values byte-equal to what the
// scheduler forwarded.
func (s *ScriptedTool) RecordedPolicies() []agent.PolicySlot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]agent.PolicySlot, len(s.recordedPolicy))
	copy(out, s.recordedPolicy)
	return out
}

// StartTimes returns the per-invocation `time.Now().UnixNano()`
// values, in invocation order. Used by the S-TLS-006 start-order
// tests.
func (s *ScriptedTool) StartTimes() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]int64, len(s.startAt))
	copy(out, s.startAt)
	return out
}

// Compile-time guard: ScriptedTool must satisfy agent.Tool. A
// non-conforming implementation would fail to compile here, not
// at runtime under a Schedule call.
var _ agent.Tool = (*ScriptedTool)(nil)

// (unused parameter — `_ = t` is a no-op marker used by some
// helper functions; tests import it for completeness.)
var _ = testing.T{}
