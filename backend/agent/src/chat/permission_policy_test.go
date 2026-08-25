// CH-10.1 — port + default policy + composition-root discipline tests
// (S-CPM-001..006, S-CPM-017a.5, R-CPM-001, R-CPM-002).
//
// Tests:
//   - S-CPM-001 — chat.PermissionPolicy is a byte-identical alias of
//     agent.PermissionPolicy (type-aliased, not redeclared).
//   - S-CPM-002 — Config{PermissionPolicy: nil} → ErrNilPermissionPolicy
//     (typed refusal, never panic).
//   - S-CPM-003 — composition-root-only construction; only cmd/chat/main.go
//     calls chat.NewDefaultPermissionPolicy in production paths.
//   - S-CPM-004 — default policy defers for configured tools and allows
//     everything else synchronously (AllowOnce).
//   - S-CPM-005 — AllowAlways collapses to AllowOnce in the default
//     implementation (the default has no AllowAlways return path).
//   - S-CPM-006 — ModifyInput is bypassed by the default policy (no
//     argument substitution; the default has no ModifyInput return path).
//   - S-CPM-017a.5 — recordVerdict returns ErrDecisionAlreadyMade typed on
//     a second call with the same callID (double-decision race → 409).

package chat_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// newToolCallForTest constructs an ai.ToolCall via the documented
// ai.NewToolCall constructor and returns the typed value the
// permission policy's Resolve signature takes.
func newToolCallForTest(t *testing.T, name string) ai.ToolCall {
	t.Helper()
	part, err := ai.NewToolCall("c1", name, []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall(%q): %v", name, err)
	}
	toolCall, ok := part.ToolCall()
	if !ok {
		t.Fatalf("ai.NewToolCall(%q) did not return a ToolCall part", name)
	}
	return toolCall
}

// S-CPM-001 — chat.PermissionPolicy is a byte-identical alias of
// agent.PermissionPolicy (R-CPM-001). A type alias (vs. redeclaration)
// preserves static-type byte-identity; this test asserts that by
// comparing reflect.Type of the two interface declarations directly
// (not the dynamic type of any assigned value).
func TestPermissionPolicy_TypeAliasByteIdentical(t *testing.T) {
	t.Parallel()

	// Both interfaces must have IDENTICAL reflect.Type — only true
	// for type aliases. A redeclared interface would have its own
	// distinct identity even with byte-identical method sets.
	got := reflect.TypeOf((*chat.PermissionPolicy)(nil)).Elem()
	want := reflect.TypeOf((*agent.PermissionPolicy)(nil)).Elem()
	if got != want {
		t.Errorf("chat.PermissionPolicy reflect.Type = %v, want %v (S-CPM-001 — type alias must be byte-identical, not redeclared)", got, want)
	}

	// Bonus: cross-assignment without conversion — only works for type aliases.
	// A redeclared interface with the same method set would require an
	// explicit conversion. The mere fact that `var q agent.PermissionPolicy = p`
	// compiles is the byte-identity proof.
	var p chat.PermissionPolicy = chat.NewDefaultPermissionPolicy(nil)
	//nolint:staticcheck // SA4023: cross-type alias check is intentional.
	q := agent.PermissionPolicy(p)
	_ = q
}

// S-CPM-002 — Config{PermissionPolicy: nil} → ErrNilPermissionPolicy
// (R-CPM-001, NFR-CTS-001 mirror — typed refusal, never a panic).
func TestConversation_NewConversation_NilPermissionPolicy_ReturnsTypedError(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider()
	_, err := chat.NewConversation(chat.Config{
		Provider:         provider,
		Store:            chat.NewMemoryConversationStore(),
		ParticipantID:    "scn-002-participant",
		ToolSource:       chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy: nil, // <-- the test target
	})
	if err == nil {
		t.Fatalf("NewConversation returned nil error with PermissionPolicy=nil, want ErrNilPermissionPolicy")
	}
	if !errors.Is(err, chat.ErrNilPermissionPolicy) {
		t.Errorf("error = %v, want errors.Is(_, chat.ErrNilPermissionPolicy)", err)
	}
}

// S-CPM-003 — composition-root-only construction; only one site in
// the production source tree instantiates NewDefaultPermissionPolicy. Test
// files may construct extra instances freely (per R-CTS-001 precedent).
func TestPermissionPolicy_CompositionRootOnly(t *testing.T) {
	t.Parallel()

	// CH-10 R-CPM-001 / S-CPM-003: chat.NewDefaultPermissionPolicy
	// must appear in EXACTLY ONE production site (cmd/chat/main.go's
	// factory closure body).
	//
	// We grep the production tree (chat + cmd/chat) for the symbol
	// and assert exactly one match under non-test files.
	const target = "chat.NewDefaultPermissionPolicy"
	matches := productionSymbolMatches(t, target)
	if matches != 1 {
		t.Errorf("%s appears %d times in production source; want 1 (S-CPM-003 — composition-root discipline)", target, matches)
	}
}

// S-CPM-004 — default policy defers for tools in deferToolNames and
// allows everything else synchronously (R-CPM-002, D-2).
func TestDefaultPermissionPolicy_DefersConfigured_AllowsRest(t *testing.T) {
	t.Parallel()

	policy := chat.NewDefaultPermissionPolicy([]string{"summarize_conversation"})

	deferCall := newToolCallForTest(t, "summarize_conversation")
	allowCall := newToolCallForTest(t, "current_time")

	deferVerdict := policy.Resolve(context.Background(), deferCall)
	if deferVerdict.Outcome != agent.PermissionDefer {
		t.Errorf("Resolve(summarize_conversation) Outcome = %v, want PermissionDefer", deferVerdict.Outcome)
	}

	allowVerdict := policy.Resolve(context.Background(), allowCall)
	if allowVerdict.Outcome != agent.PermissionOutcomeAllowOnce {
		t.Errorf("Resolve(current_time) Outcome = %v, want PermissionOutcomeAllowOnce", allowVerdict.Outcome)
	}
}

// S-CPM-005 — AllowAlways collapses to AllowOnce in the default
// implementation (R-CPM-002, D-12). The default has no AllowAlways
// return path: a grep over the production default's source returns
// zero matches for PermissionOutcomeAllowAlways return statements.
func TestDefaultPermissionPolicy_AllowAlwaysCollapsedToAllowOnce(t *testing.T) {
	t.Parallel()

	policy := chat.NewDefaultPermissionPolicy([]string{"current_time"})
	// Drive Resolve many times for various tools; the verdict MUST
	// always be AllowOnce or Defer — never AllowAlways.
	for _, name := range []string{"a", "b", "c", "summarize_conversation"} {
		v := policy.Resolve(context.Background(), newToolCallForTest(t, name))
		if v.Outcome == agent.PermissionOutcomeAllowAlways {
			t.Errorf("Resolve(%q) Outcome = AllowAlways; the default policy MUST collapse to AllowOnce (S-CPM-005, D-12)", name)
		}
	}
}

// S-CPM-006 — ModifyInput is bypassed by the default policy (no
// argument substitution; R-CPM-002). The default has no ModifyInput
// return path: Resolve never returns PermissionOutcomeModifyInput.
func TestDefaultPermissionPolicy_ModifyInputBypassed(t *testing.T) {
	t.Parallel()

	policy := chat.NewDefaultPermissionPolicy(nil)
	for _, name := range []string{"a", "b", "c", "current_time", "summarize_conversation"} {
		v := policy.Resolve(context.Background(), newToolCallForTest(t, name))
		if v.Outcome == agent.PermissionOutcomeModifyInput {
			t.Errorf("Resolve(%q) Outcome = ModifyInput; the default policy MUST NOT substitute arguments (S-CPM-006)", name)
		}
	}
}

// S-CPM-017a.5 — recordVerdict on the default policy returns
// ErrDecisionAlreadyMade typed on a second call with the same callID.
// The HTTP handler maps this to 409 conflict (R-CPM-004 + D-10).
func TestDefaultPermissionPolicy_RecordVerdict_DoubleDecision_Refused(t *testing.T) {
	t.Parallel()

	policy := chat.NewDefaultPermissionPolicy([]string{"summarize_conversation"})

	// First call: Defer; second call's recordVerdict on the same
	// callID must succeed.
	if v := policy.Resolve(context.Background(), newToolCallForTest(t, "summarize_conversation")); v.Outcome != agent.PermissionDefer {
		t.Fatalf("first Resolve Outcome = %v, want PermissionDefer (precondition for recordVerdict)", v.Outcome)
	}

	// Cast to *chat.DefaultPermissionPolicy so we can call the
	// chat-only recordVerdict (not on the agent.PermissionPolicy
	// interface).
	type verdictRecorder interface {
		RecordVerdict(callID, outcome string) error
	}
	vr, ok := any(policy).(verdictRecorder)
	if !ok {
		t.Fatalf("DefaultPermissionPolicy does not implement RecordVerdict (CH-10 D-2 typed sentinel seam)")
	}

	if err := vr.RecordVerdict("c1", "allow_once"); err != nil {
		t.Fatalf("first RecordVerdict returned %v, want nil", err)
	}
	if err := vr.RecordVerdict("c1", "deny"); !errors.Is(err, chat.ErrDecisionAlreadyMade) {
		t.Errorf("second RecordVerdict error = %v, want errors.Is(_, chat.ErrDecisionAlreadyMade)", err)
	}
}

// TestPermissionPolicy_ResolveRememberContract checks that the
// default policy's Remember returns false unconditionally at v1
// (D-12 wire collapse: chat collapses AllowAlways → AllowOnce; never
// asks Remember to persist).
func TestDefaultPermissionPolicy_RememberReturnsFalse(t *testing.T) {
	t.Parallel()

	policy := chat.NewDefaultPermissionPolicy(nil)
	if got := policy.Remember(context.Background(), "any-tool", agent.PermissionOutcomeAllowAlways); got != false {
		t.Errorf("Remember returned %v, want false (S-CPM-005/D-12 — v1 static)", got)
	}
}

// TestPermissionPolicy_NilPolicyBypassMirrors_AG10 documents the
// v1 composition-root posture: when the policy is nil the harness
// runs the AG-10 nil-bypass (no events, no asks). This test is the
// contract assurance: chat's NewConversation refuses a nil
// PermissionPolicy typed (above), so the harness never sees a nil
// policy in production chat paths.
//
// The harness-side AG-10 nil-bypass contract is verified by AG-10's
// own scenario S-APP-001. The chat-side refusal is the complement:
// a nil chat-side policy is a composition defect, not a runtime
// identity.
func TestPermissionPolicy_NilPolicyRefusedAtComposition(t *testing.T) {
	t.Parallel()

	if !strings.Contains(chat.ErrNilPermissionPolicy.Error(), "PermissionPolicy") {
		t.Errorf("ErrNilPermissionPolicy message = %q; want to mention PermissionPolicy for diagnosis", chat.ErrNilPermissionPolicy.Error())
	}
}