// AG-12.1 — History and the pairing invariant: the transcript store's own
// behavior (R-HIS-001..006), exercised from an external test package
// (NFR-HIS-001). An in-package test could reach the unexported `commit`
// primitive directly and would prove nothing about the boundary R-HIS-004
// asks this capability to close.
//
// Shared helpers for this capability's three test files (history_test.go,
// history_synthesis_test.go, history_surface_guard_test.go) live here, the
// first of the three to land, mirroring agent_test_helpers_test.go's own
// "kept in one place" rule for AG-04.
package agent_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustMessage constructs a message from role and parts, failing the test on
// any construction violation — every scenario below builds well-formed
// Layer 1 input so its own assertions are never obscured by an unrelated
// construction failure.
func mustMessage(t *testing.T, role ai.Role, parts ...ai.Part) ai.Message {
	t.Helper()
	m, err := ai.NewMessage(role, parts...)
	if err != nil {
		t.Fatalf("ai.NewMessage(...) returned %v, want nil", err)
	}
	return m
}

func mustText(t *testing.T, text string) ai.Part {
	t.Helper()
	p, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText(%q) returned %v, want nil", text, err)
	}
	return p
}

func mustToolCall(t *testing.T, id, name string) ai.Part {
	t.Helper()
	p, err := ai.NewToolCall(id, name, nil)
	if err != nil {
		t.Fatalf("ai.NewToolCall(%q, %q, nil) returned %v, want nil", id, name, err)
	}
	return p
}

func mustToolResult(t *testing.T, callID, content string) ai.Part {
	t.Helper()
	p, err := ai.NewToolResult(callID, content)
	if err != nil {
		t.Fatalf("ai.NewToolResult(%q, %q) returned %v, want nil", callID, content, err)
	}
	return p
}

func mustToolFailure(t *testing.T, callID, content string) ai.Part {
	t.Helper()
	p, err := ai.NewToolFailure(callID, content)
	if err != nil {
		t.Fatalf("ai.NewToolFailure(%q, %q) returned %v, want nil", callID, content, err)
	}
	return p
}

// callMessage builds a well-formed assistant message carrying one tool
// call — an open call once it is committed and nothing has answered it yet.
func callMessage(t *testing.T, callID string) ai.Message {
	t.Helper()
	return mustMessage(t, ai.RoleAssistant, mustToolCall(t, callID, "a-tool"))
}

// resultMessage builds a well-formed tool-role message carrying one result
// answering callID.
func resultMessage(t *testing.T, callID string) ai.Message {
	t.Helper()
	return mustMessage(t, ai.RoleTool, mustToolResult(t, callID, "ok"))
}

// requireViolation fails t unless err is a typed *ai.Violation matching
// wantRule (errors.Is — which rule) and rendering at wantPosition
// (Path.String() — where), the two-axis proof AI-04's own doc comment
// states as the whole of what a Layer 1 caller-contract failure answers.
func requireViolation(t *testing.T, err error, wantRule error, wantPosition string) {
	t.Helper()
	if err == nil {
		t.Fatalf("got nil error, want an *ai.Violation matching %v at %q", wantRule, wantPosition)
	}
	if !errors.Is(err, wantRule) {
		t.Errorf("errors.Is(err, %v) = false, want true (err = %v)", wantRule, err)
	}
	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, &violation) = false, want true so the position is inspectable (err = %v)", err)
	}
	if got := violation.Path().String(); got != wantPosition {
		t.Errorf("violation position = %q, want %q (err = %v)", got, wantPosition, err)
	}
}

// TestHistory_OrderPreserved (S-HIS-001) — messages appended in
// conversation order through the public route read back in the same order,
// element-by-element, with no insertion, omission or reordering.
func TestHistory_OrderPreserved(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	msgs := []ai.Message{
		mustMessage(t, ai.RoleUser, mustText(t, "first")),
		mustMessage(t, ai.RoleAssistant, mustText(t, "second")),
		mustMessage(t, ai.RoleUser, mustText(t, "third")),
	}
	for i, m := range msgs {
		if err := h.Append(m); err != nil {
			t.Fatalf("Append(msgs[%d]) returned %v, want nil", i, err)
		}
	}

	entries := h.Entries()
	if len(entries) != len(msgs) {
		t.Fatalf("Entries() returned %d entries, want %d", len(entries), len(msgs))
	}
	for i, entry := range entries {
		if entry.Message().ID() != msgs[i].ID() {
			t.Errorf("entries[%d].Message().ID() = %v, want %v (the unmodified committed message, not a rebuilt copy)", i, entry.Message().ID(), msgs[i].ID())
		}
		if !entry.Message().Equal(msgs[i]) {
			t.Errorf("entries[%d].Message() = %v, want %v (element-by-element, same order, S-HIS-001)", i, entry.Message(), msgs[i])
		}
	}
	if h.Len() != len(msgs) {
		t.Errorf("Len() = %d, want %d", h.Len(), len(msgs))
	}
}

// TestHistory_ReadDoesNotAliasInternalStorage (S-HIS-002) — mutating a
// value a read returned must not be observable in a subsequent read.
func TestHistory_ReadDoesNotAliasInternalStorage(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	original := mustMessage(t, ai.RoleUser, mustText(t, "original"))
	if err := h.Append(original); err != nil {
		t.Fatalf("Append(original) returned %v, want nil", err)
	}

	entries := h.Entries()
	entries[0] = agent.Entry{} // mutate the caller's own view

	again := h.Entries()
	if again[0].ID() == 0 {
		t.Fatal("a mutated Entries() slice leaked into internal storage: second read's entry identity is unset")
	}
	if !again[0].Message().Equal(original) {
		t.Errorf("a mutated Entries() slice leaked into internal storage: second read = %v, want the original message unchanged (S-HIS-002)", again[0].Message())
	}
}

// TestHistory_OrphanedResult_RejectedTyped (S-HIS-010) — a tool-result
// entry naming a call the transcript does not declare is rejected typed,
// and the rejected append leaves the transcript unchanged: a failed
// commit is not a partial commit (R-HIS-002).
//
// The history holds a DIFFERENT open call ("call-other") before the
// orphan is attempted, not zero calls — S-HIS-012's bite scratch-weakens
// the check to "accept any result once >= 1 call exists, identity
// ignored"; that weakening only fails this test if the fixture has a
// call present that is not the one named, which is why the fixture is
// built this way rather than against an empty open set.
func TestHistory_OrphanedResult_RejectedTyped(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	unrelatedCall := callMessage(t, "call-other")
	if err := h.Append(unrelatedCall); err != nil {
		t.Fatalf("Append(unrelatedCall) returned %v, want nil", err)
	}
	beforeEntries := h.Entries()

	orphan := resultMessage(t, "call-missing")
	err := h.Append(orphan)
	requireViolation(t, err, ai.ErrUnresolvedReference, "messages[1].content[0]")

	after := h.Entries()
	if len(after) != len(beforeEntries) {
		t.Fatalf("Entries() after a rejected append returned %d entries, want %d (a failed commit is not a partial commit, R-HIS-002)", len(after), len(beforeEntries))
	}
	for i := range beforeEntries {
		if after[i].ID() != beforeEntries[i].ID() || !after[i].Message().Equal(beforeEntries[i].Message()) {
			t.Errorf("entries[%d] changed after a rejected append: got %v, want %v unchanged", i, after[i], beforeEntries[i])
		}
	}
}

// TestHistory_ResultAfterMatchingCall_Accepted (S-HIS-011) — a tool-result
// entry naming a call the history already holds is accepted and reads
// back after the call, in order.
func TestHistory_ResultAfterMatchingCall_Accepted(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	call := callMessage(t, "call-1")
	if err := h.Append(call); err != nil {
		t.Fatalf("Append(call) returned %v, want nil", err)
	}
	result := resultMessage(t, "call-1")
	if err := h.Append(result); err != nil {
		t.Fatalf("Append(result) returned %v, want nil", err)
	}

	entries := h.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() returned %d entries, want 2", len(entries))
	}
	if !entries[0].Message().Equal(call) {
		t.Errorf("entries[0] = %v, want the call message", entries[0].Message())
	}
	if !entries[1].Message().Equal(result) {
		t.Errorf("entries[1] = %v, want the result message read back after the call, in order (S-HIS-011)", entries[1].Message())
	}
}

// TestHistory_UnclosedCallAtTurnClose_RejectedTyped (S-HIS-020) — a tool
// call with no matching result is rejected typed when the turn closes,
// positioned at the missing result's slot, and the transcript is left
// unchanged.
func TestHistory_UnclosedCallAtTurnClose_RejectedTyped(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	call := callMessage(t, "call-open")
	if err := h.Append(call); err != nil {
		t.Fatalf("Append(call) returned %v, want nil", err)
	}
	beforeEntries := h.Entries()

	err := h.CloseTurn()
	requireViolation(t, err, ai.ErrEmpty, "messages[0].content[0].result")

	after := h.Entries()
	if len(after) != len(beforeEntries) {
		t.Fatalf("Entries() after a rejected CloseTurn returned %d entries, want %d unchanged", len(after), len(beforeEntries))
	}
}

// TestHistory_AllCallsClosed_TurnCloseSucceeds (S-HIS-021) — a turn close
// over a transcript in which every call has a matching result succeeds,
// and the transcript reads back unchanged and in order.
func TestHistory_AllCallsClosed_TurnCloseSucceeds(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	call := callMessage(t, "call-closed")
	result := resultMessage(t, "call-closed")
	if err := h.Append(call); err != nil {
		t.Fatalf("Append(call) returned %v, want nil", err)
	}
	if err := h.Append(result); err != nil {
		t.Fatalf("Append(result) returned %v, want nil", err)
	}

	if err := h.CloseTurn(); err != nil {
		t.Fatalf("CloseTurn() returned %v, want nil (every call has a matching result)", err)
	}

	entries := h.Entries()
	if len(entries) != 2 || !entries[0].Message().Equal(call) || !entries[1].Message().Equal(result) {
		t.Errorf("Entries() after a successful CloseTurn = %v, want [call, result] unchanged and in order", entries)
	}
}

// TestHistory_TwoUnclosedCalls_NamesFirstOffendingPosition (S-HIS-022) —
// when two distinct tool calls are both unclosed, the reported violation
// names the position of the FIRST offending call in transcript order, so
// a caller can act on a deterministic position.
func TestHistory_TwoUnclosedCalls_NamesFirstOffendingPosition(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	first := callMessage(t, "call-first")
	second := callMessage(t, "call-second")
	if err := h.Append(first); err != nil {
		t.Fatalf("Append(first) returned %v, want nil", err)
	}
	if err := h.Append(second); err != nil {
		t.Fatalf("Append(second) returned %v, want nil", err)
	}

	err := h.CloseTurn()
	requireViolation(t, err, ai.ErrEmpty, "messages[0].content[0].result")
}
