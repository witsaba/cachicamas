// CH-09.1 — chat.ToolSource port + FromAgentRegistry adapter tests
// (S-CTS-001, S-CTS-002, S-CTS-003). The three scenarios are
// transcribed verbatim from the explore report #3952.
//
// All tests use the package-public chat.ToolSource / FromAgentRegistry
// surface so the test verifies the port, not the unexported adapter.

package chat_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/chat"
)

// echoTool is a stub agent.Tool used in S-CTS-003. The tool name and
// effect class are sufficient for the test — Run never gets called
// because the test only exercises the adapter's Resolve delegation.
type echoTool struct{}

func (echoTool) Name() string                              { return "echo" }
func (echoTool) EffectClass() agent.EffectClass            { return agent.EffectClassRead }
func (echoTool) Run(_ context.Context, _ []byte, _ agent.PolicySlot) (agent.Result, error) {
	return agent.Result{Outcome: agent.ToolOutcomeSuccess}, nil
}

// S-CTS-001 — Given Config{ToolSource: nil}, When NewConversation
// runs, Then it returns (*Conversation, ErrNilToolSource) typed,
// never panicking. The test asserts both halves of the contract:
// the error is non-nil AND errors.Is-compatible with ErrNilToolSource.
func TestToolSource_NilConfigReturnsErrNilToolSource(t *testing.T) {
	// Build a Config that satisfies the OTHER required fields
	// (Provider, Store, ParticipantID) so the nil-ToolSource check
	// is the documented rejection point. The store is a fresh
	// in-memory adapter; the provider is a stub from agenttest.
	cfg := chat.Config{
		// Provider is required; the nil check fires before
		// NewConversation reaches the ToolSource check, so we must
		// set Provider to a non-nil stub. The stub used here is
		// agenttest.NewProvider with no scripts — Send is never
		// called on this conversation, so the stub's empty script
		// list is fine.
		Provider:       agenttest.NewProvider(),
		Store:          chat.NewMemoryConversationStore(),
		ParticipantID:  "scn-001-nil",
		ToolSource:     nil,
	}

	conv, err := chat.NewConversation(cfg)
	if err == nil {
		t.Fatalf("NewConversation returned nil error, want ErrNilToolSource")
	}
	if !isErrNilToolSource(err) {
		t.Fatalf("NewConversation returned error %v, want errors.Is(ErrNilToolSource)", err)
	}
	if conv != nil {
		t.Fatalf("NewConversation returned non-nil *Conversation %v on error", conv)
	}
}

// S-CTS-002 — Given FromAgentRegistry(agent.NewMapRegistry(nil)), When
// Resolve("any") runs, Then it returns (nil, false) and does not
// panic. The map-registry constructor is nil-safe (agent/tool.go:280);
// the adapter's nil-safe posture is verified by passing a nil-backed
// registry through FromAgentRegistry and asserting the delegation
// passes the nil-safe behaviour through.
func TestToolSource_FromAgentRegistry_NilMapRegistry_ReturnsFalse(t *testing.T) {
	src := chat.FromAgentRegistry(agent.NewMapRegistry(nil))
	if src == nil {
		t.Fatalf("FromAgentRegistry returned nil, want non-nil ToolSource")
	}
	// Must not panic.
	tool, ok := src.Resolve("any")
	if tool != nil {
		t.Errorf("Resolve returned non-nil tool %v, want nil", tool)
	}
	if ok {
		t.Errorf("Resolve returned ok=true, want false")
	}
}

// S-CTS-003 — Given FromAgentRegistry(agent.NewMapRegistry({"echo":
// &EchoTool{}})), When Resolve("echo") runs, Then it returns
// (*EchoTool, true) byte-equal to the wrapped map's value. The
// adapter delegates byte-for-byte; the test asserts the SAME pointer
// is returned, not a copy.
func TestToolSource_FromAgentRegistry_DelegatesToWrappedRegistry(t *testing.T) {
	want := &echoTool{}
	src := chat.FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{
		"echo": want,
	}))
	if src == nil {
		t.Fatalf("FromAgentRegistry returned nil, want non-nil ToolSource")
	}
	got, ok := src.Resolve("echo")
	if !ok {
		t.Fatalf("Resolve returned ok=false, want true")
	}
	if got != want {
		// Resolve returns the agent.Tool interface value; we expect
		// the dynamic type's underlying pointer to be byte-equal to
		// `want`. agent.Tool is an interface, so the value-equal
		// comparison checks both the dynamic type and the wrapped
		// pointer.
		t.Errorf("Resolve returned %v (%T), want %v (%T)", got, got, want, want)
	}
	// Miss path: an unknown name returns (nil, false) without panic.
	if tool, ok := src.Resolve("missing"); ok || tool != nil {
		t.Errorf("Resolve(missing) returned (%v, %v), want (nil, false)", tool, ok)
	}
}

// S-CTS-001 (extra) — FromAgentRegistry(nil) is also nil-safe. The
// agent.Registry interface can be nil itself (a typed-nil), so the
// adapter must guard against it explicitly.
func TestToolSource_FromAgentRegistry_NilRegistry_ReturnsFalse(t *testing.T) {
	var nilReg agent.Registry // nil interface
	src := chat.FromAgentRegistry(nilReg)
	if src == nil {
		t.Fatalf("FromAgentRegistry(nil) returned nil, want non-nil ToolSource")
	}
	tool, ok := src.Resolve("any")
	if tool != nil || ok {
		t.Errorf("Resolve returned (%v, %v), want (nil, false)", tool, ok)
	}
}

// isErrNilToolSource reports whether err is ErrNilToolSource or wraps
// it. errors.Is is the right tool here, but we re-implement it locally
// to avoid an import for one call site; the standard errors.Is would
// do the same.
func isErrNilToolSource(err error) bool {
	if err == nil {
		return false
	}
	for e := err; e != nil; {
		if e == chat.ErrNilToolSource {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := e.(unwrapper)
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}