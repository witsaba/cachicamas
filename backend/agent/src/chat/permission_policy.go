// CH-10.1 — chat-owned permission policy port (R-CPM-001, R-CPM-002,
// D-1, D-2).
//
// chat.PermissionPolicy is a TYPE ALIAS of agent.PermissionPolicy
// (AG-10, `permission_protocol.go:80-94`). The alias preserves byte-
// identity with Layer 2's protocol interface — a value of one
// dynamic type satisfies the other, and the S-CPM-001 reflect-based
// test asserts this byte-equality.
//
// The default implementation (D-2, G7) defers for any tool name in
// deferToolNames and returns AllowOnce for everything else. State is
// held in a per-Conversation pendingDecision map keyed by callID; the
// HTTP handler (T-03b) writes the verdict via RecordVerdict and the
// gate's re-entry on WakeParked reads it back.
//
// Layer 2 owns the protocol; chat owns the policy content (R-APP-011).
// The chat package imports Layer 2's agent.PermissionPolicy by
// interface (no import-of-internals); the 10-file substrate list
// stays byte-clean.

package chat

import (
	"context"
	"errors"
	"sync"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// PermissionPolicy is the chat-owned port for permission policy
// (R-CPM-001, D-1). Type-aliased to agent.PermissionPolicy so the
// dynamic type is byte-identical (S-CPM-001). The chat package
// depends on Layer 2's interface, never on a concrete policy type.
type PermissionPolicy = agent.PermissionPolicy

// ErrNilPermissionPolicy is returned by NewConversation when
// cfg.PermissionPolicy is nil (R-CPM-001). Typed (errors.Is), never
// panic — a misconfigured composition root surfaces a distinguishable
// rejection at composition time, not a nil-deref at first call.
var ErrNilPermissionPolicy = errors.New("chat: Config.PermissionPolicy is required")

// ErrDecisionAlreadyMade is returned by RecordVerdict on a second
// call with the same callID (R-CPM-002, D-10 race). The HTTP
// handler maps this to a 409 conflict envelope (R-CPM-004). A
// typed sentinel so the chat package's chat-specific policy
// methods (RecordVerdict is NOT on the agent.PermissionPolicy
// interface — chat-owned state, chat-owned seam) can carry a
// typed refusal without polluting the Layer 2 surface.
var ErrDecisionAlreadyMade = errors.New("chat: decision already made for this callID")

// pendingDecision is the per-call verdict state the HTTP handler
// writes via RecordVerdict and the gate's re-entry on WakeParked
// reads back. The Outcome field carries the chat wire's CLOSED
// 2-value vocabulary "allow_once" | "deny" (D-12 collapse).
type pendingDecision struct {
	WireCallID string
	Outcome    string
}

// DefaultPermissionPolicy is the chat-owned v1 policy (D-2, G7,
// R-CPM-002). Defers for tools in deferToolNames; allows everything
// else synchronously with AllowOnce. The pendingDecision map carries
// state across the ask → re-enter boundary; it lives one
// Conversation (factory closure owns lifecycle).
//
// Exported because the constructor returns it (and golangci-lint's
// revive rule complains about unexported returns). Callers always
// reach the type via the chat.PermissionPolicy interface (which
// is type-aliased to agent.PermissionPolicy, byte-identical,
// S-CPM-001).
//
// # Concurrency
//
// The pendingDecision map is guarded by sync.Mutex; Resolve is
// called from the gate goroutine and RecordVerdict is called from
// the HTTP handler goroutine. Both paths acquire the lock briefly.
type DefaultPermissionPolicy struct {
	mu              sync.Mutex
	deferSet        map[string]bool
	pendingDecision map[string]pendingDecision
}

// NewDefaultPermissionPolicy constructs the chat-owned v1 policy
// (R-CPM-002, D-2). deferToolNames is the list of tool names that
// the policy defers; an empty/nil list means "never defer" (all
// calls run synchronously with AllowOnce).
//
// Returns the chat-owned struct so the chat-only RecordVerdict
// method is reachable through a type assertion. The HTTP handler
// performs the assertion (S-CPM-017a.5). The unexported return
// type is intentional — callers always go through the
// chat.PermissionPolicy interface (which is type-aliased to
// agent.PermissionPolicy, byte-identical, S-CPM-001).
func NewDefaultPermissionPolicy(deferToolNames []string) *DefaultPermissionPolicy {
	deferSet := make(map[string]bool, len(deferToolNames))
	for _, name := range deferToolNames {
		deferSet[name] = true
	}
	return &DefaultPermissionPolicy{
		deferSet:        deferSet,
		pendingDecision: make(map[string]pendingDecision),
	}
}

// Resolve asks the policy for a verdict on one call (R-APP-001).
//
//   - If a pendingDecision is recorded for this call's id, return it
//     verbatim (the gate's re-entry after WakeParked). WireOutcome
//     "allow_once" → PermissionOutcomeAllowOnce; "deny" →
//     PermissionOutcomeDeny. The wire-outcome → typed-outcome
//     mapping lives here (D-12).
//
//   - Otherwise, if the tool name is in deferSet, return
//     PermissionDefer.
//
//   - Otherwise, return AllowOnce (G7 — synchronous verdict for every
//     non-deferred tool).
//
// Note the wire-outcome vocabulary lives in the recorded verdict; the
// default policy's Resolve never produces AllowAlways (S-CPM-005,
// D-12 collapse) or ModifyInput (S-CPM-006, D-12 bypass).
func (p *DefaultPermissionPolicy) Resolve(_ context.Context, call ai.ToolCall) agent.PermissionVerdict {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pd, ok := p.pendingDecision[call.ID()]; ok {
		return pd.toVerdict()
	}

	if p.deferSet[call.Name()] {
		return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
	}
	return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
}

// Remember is the policy's storage hook for AllowAlways rules
// (R-APP-007). The default v1 returns false unconditionally: chat
// collapses AllowAlways → AllowOnce on the wire (D-12), so the
// scheduler never invokes Remember on the default policy. The hook
// is implemented (typed to satisfy the interface) but the value is
// always false (S-CPM-005 / D-12 static posture).
func (p *DefaultPermissionPolicy) Remember(_ context.Context, _ string, _ agent.PermissionOutcome) bool {
	return false
}

// RecordVerdict writes the user's HTTP click into the pendingDecision
// map (R-CPM-002, D-10 race). outcome must be one of the chat wire's
// closed 2-value vocabulary "allow_once" | "deny" — out-of-vocab
// values are refused typed (the HTTP handler maps to 422).
//
// Returns ErrDecisionAlreadyMade typed when an entry is already
// present for the same callID (S-CPM-017a.5); the HTTP handler maps
// this to 409 conflict. The double-decision race surfaces here, not
// as silent overwriting.
//
// The method is chat-specific — Layer 2's agent.PermissionPolicy
// interface does NOT expose it. Callers (HTTP handler) reach it via
// a type assertion on the *defaultPermissionPolicy returned by
// NewDefaultPermissionPolicy.
func (p *DefaultPermissionPolicy) RecordVerdict(callID, outcome string) error {
	if outcome != "allow_once" && outcome != "deny" {
		return errors.New("chat: RecordVerdict outcome must be \"allow_once\" or \"deny\"")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.pendingDecision[callID]; exists {
		return ErrDecisionAlreadyMade
	}
	p.pendingDecision[callID] = pendingDecision{
		WireCallID: callID,
		Outcome:    outcome,
	}
	return nil
}

// toVerdict maps a recorded pendingDecision to the typed
// PermissionVerdict (R-APP-001 + D-12 collapse). The recorded
// Outcome field is the chat wire's 2-value vocabulary; this is the
// single point of translation.
func (pd pendingDecision) toVerdict() agent.PermissionVerdict {
	switch pd.Outcome {
	case "allow_once":
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
	case "deny":
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeDeny}
	default:
		// Defensive only: RecordVerdict's validation prevents
		// out-of-vocab values from reaching the map.
		return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
	}
}