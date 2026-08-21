// Package apptest — permission.go: the scriptable agent.PermissionPolicy
// (AG-23, design AD-3). Resolves queued verdicts strictly FIFO under a
// mutex; never re-implements the permission gate itself — it only scripts
// the decision surface a real Layer 3 policy would supply (R-L3H-003).
package apptest

import (
	"context"
	"sync"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ScriptedPermissionPolicy is an importable, scriptable
// agent.PermissionPolicy. A consumer queues verdicts up front at
// construction; Resolve returns them in queue order, one per call,
// under a mutex.
//
// An exhausted queue MUST NOT wedge a run (R-L3H-003): once every queued
// verdict has been consumed, Resolve falls back to the stated default —
// PermissionOutcomeAllowOnce — and latches Exhausted, so a consumer
// asserts full consumption rather than discovering starvation as a hang.
type ScriptedPermissionPolicy struct {
	// RememberReturns is the value Remember reports on every call —
	// whether this policy "committed" an AllowAlways rule to a
	// persistent store. Zero value (false): the scripted policy commits
	// nothing by default, matching a policy with no persistent memory.
	RememberReturns bool

	mu        sync.Mutex
	verdicts  []agent.PermissionVerdict
	next      int
	exhausted bool
	resolved  []ai.ToolCall
}

// NewScriptedPermissionPolicy constructs a policy that resolves verdicts,
// in the order given, one per Resolve call.
func NewScriptedPermissionPolicy(verdicts ...agent.PermissionVerdict) *ScriptedPermissionPolicy {
	queued := make([]agent.PermissionVerdict, len(verdicts))
	copy(queued, verdicts)
	return &ScriptedPermissionPolicy{verdicts: queued}
}

// Resolve implements agent.PermissionPolicy (permission_protocol.go's
// interface pin). It records call, then resolves the next queued verdict
// in FIFO order. Once the queue is exhausted it returns AllowOnce and
// latches Exhausted rather than blocking, panicking, or wedging the run.
func (p *ScriptedPermissionPolicy) Resolve(ctx context.Context, call ai.ToolCall) agent.PermissionVerdict {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resolved = append(p.resolved, call)
	if p.next >= len(p.verdicts) {
		p.exhausted = true
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
	}
	v := p.verdicts[p.next]
	p.next++
	return v
}

// Remember implements agent.PermissionPolicy. It returns RememberReturns
// unconditionally — the scripted policy never decides for itself whether
// a rule was committed; the consumer configures the answer up front.
func (p *ScriptedPermissionPolicy) Remember(ctx context.Context, toolName string, outcome agent.PermissionOutcome) bool {
	return p.RememberReturns
}

// Resolved returns every call this policy's Resolve has been asked to
// decide, in the order asked — a fresh copy, safe from further mutation
// by the caller.
func (p *ScriptedPermissionPolicy) Resolved() []ai.ToolCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ai.ToolCall, len(p.resolved))
	copy(out, p.resolved)
	return out
}

// Exhausted reports whether Resolve has ever been asked more times than
// verdicts were queued at construction.
func (p *ScriptedPermissionPolicy) Exhausted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exhausted
}

// Compile-time guard: ScriptedPermissionPolicy must satisfy
// agent.PermissionPolicy. A non-conforming implementation fails to
// compile here, not at runtime under a Schedule call.
var _ agent.PermissionPolicy = (*ScriptedPermissionPolicy)(nil)
