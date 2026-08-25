// CH-09.1 — the chat archetype's tool-source port (D-1, D-2).
//
// Chat owns the port the same way it owns chat.ConversationStore
// (CH-06/07/08 precedent, R-CCS-010). The chat package depends on
// Layer 2's agent.Registry by interface (backend/agent/src/agent/tool.go:267-269),
// never by import-of-internals — the import-boundary check
// (import_boundary_test.go) admits the chat package importing agent,
// CH-09 does NOT widen the allowlist.
//
// chat.ToolSource is the inverse of chat.ConversationStore at the
// adapter level: for ConversationStore, chat owns the port and
// adapters (Memory, Postgres) sit behind it INSIDE chat; for
// ToolSource, chat owns the port and the chat package internally
// adapts it BACK to Layer 2's agent.Registry for harness.Tools
// (loop.go:112). The composition root provides a chat.ToolSource
// whose internal field holds an agent.Registry (via
// chat.FromAgentRegistry(agent.NewMapRegistry(...))).
//
// Wrapper-direction rationale: chat.ToolSource contains
// agent.Registry (composition root owns the registry, chat owns the
// port). Different from ConversationStore where the chat port is the
// typed abstraction and Memory/Postgres are its adapters.
//
// R-CTS-001 — ToolSource is the chat package's tool-source port.
// R-CTS-002 — FromAgentRegistry adapts Layer 2's Registry to chat's
//              port byte-for-byte, nil-safe.
// NFR-CTS-001 — chat.ToolSource wraps agent.Registry by interface.

package chat

import (
	"errors"

	"github.com/cachicamas/backend/agent/src/agent"
)

// ToolSource is the chat package's tool-source port (R-CTS-001).
// Resolve(name) returns the named tool or `(nil, false)` on miss.
// The signature is byte-identical to agent.Registry.Resolve so the
// composition root can hand the chat port an agent.Registry value
// through FromAgentRegistry; the two interfaces are structurally
// identical but semantically distinct (chat's is the chat-owned port;
// agent's is Layer 2's executor-side seam).
type ToolSource interface {
	Resolve(name string) (agent.Tool, bool)
}

// ErrNilToolSource is returned by NewConversation when cfg.ToolSource
// is nil (R-CTS-001). The error is typed (errors.Is) and the check
// never panics, so a misconfigured composition root surfaces a
// distinguishable rejection rather than crashing on a nil-deref.
var ErrNilToolSource = errors.New("chat: Config.ToolSource is required")

// FromAgentRegistry adapts an agent.Registry value to a chat.ToolSource
// (R-CTS-002). The adapter is nil-safe: FromAgentRegistry(nil) returns
// a ToolSource whose Resolve always returns `(nil, false)` without
// panic. A non-nil registry is delegated to byte-for-byte.
//
// The adapter is the wrapper-direction port described in D-1:
// chat.ToolSource contains an agent.Registry internally; the
// conversation-side consumer (Config.ToolSource) reaches the agent
// surface through this boundary, never through direct import of
// agent.mapRegistry (an unexported type — chat cannot reach it via
// import-of-internals even if it tried).
func FromAgentRegistry(r agent.Registry) ToolSource {
	return &agentRegistryAdapter{r: r}
}

// agentRegistryAdapter is the FromAgentRegistry adapter's concrete
// type. It is unexported so callers reach it through the ToolSource
// interface (mirrors the agent.Registry/mapRegistry pair's posture).
type agentRegistryAdapter struct {
	r agent.Registry
}

// Resolve implements ToolSource. A nil receiver OR a nil wrapped
// registry returns `(nil, false)` without panic — the same nil-safe
// posture as agent.NewMapRegistry(nil).
func (a *agentRegistryAdapter) Resolve(name string) (agent.Tool, bool) {
	if a == nil || a.r == nil {
		return nil, false
	}
	return a.r.Resolve(name)
}