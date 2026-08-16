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
	"fmt"
	"go/ast"
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

// fieldsOrNil returns fields.List, or nil when fields itself is nil — an
// empty parameter/result list parses as a nil *ast.FieldList rather than
// an empty one.
func fieldsOrNil(fields *ast.FieldList) []*ast.Field {
	if fields == nil {
		return nil
	}
	return fields.List
}

func isIdentNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func isQualifiedIdent(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == pkg && sel.Sel.Name == name
}

// signatureTypeString renders a type expression for a failure message —
// diagnostic only, mirroring agenttest_test's exprString.
func signatureTypeString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + signatureTypeString(e.X)
	case *ast.ArrayType:
		return "[]" + signatureTypeString(e.Elt)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return "<selector>." + e.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// TestNewSeededHistory_ValidSeed_Accepted (S-HIS-050) — a pre-existing
// transcript in which every tool call has a matching result constructs
// successfully, reads back equal to the seed and in order, with
// ordinal-derived entry identities.
func TestNewSeededHistory_ValidSeed_Accepted(t *testing.T) {
	t.Parallel()

	seed := []ai.Message{
		mustMessage(t, ai.RoleUser, mustText(t, "hi")),
		callMessage(t, "call-seed"),
		resultMessage(t, "call-seed"),
	}
	h, err := agent.NewSeededHistory(seed)
	if err != nil {
		t.Fatalf("NewSeededHistory(seed) returned %v, want nil", err)
	}
	entries := h.Entries()
	if len(entries) != len(seed) {
		t.Fatalf("Entries() returned %d entries, want %d", len(entries), len(seed))
	}
	for i, entry := range entries {
		if entry.ID() != agent.EntryID(i+1) {
			t.Errorf("entries[%d].ID() = %v, want %v (1-based ordinal)", i, entry.ID(), agent.EntryID(i+1))
		}
		if !entry.Message().Equal(seed[i]) {
			t.Errorf("entries[%d].Message() = %v, want %v", i, entry.Message(), seed[i])
		}
	}
}

// TestNewSeededHistory_OrphanedResult_RejectedFirstOffendingPosition
// (S-HIS-051, result direction only) — a seed containing an orphaned
// result fails with the same rule class the equivalent append would
// produce, positioned at the first offending entry in seed order.
func TestNewSeededHistory_OrphanedResult_RejectedFirstOffendingPosition(t *testing.T) {
	t.Parallel()

	seed := []ai.Message{
		mustMessage(t, ai.RoleUser, mustText(t, "hi")),
		resultMessage(t, "call-missing"), // index 1: no prior call in the seed
	}
	h, err := agent.NewSeededHistory(seed)
	requireViolation(t, err, ai.ErrUnresolvedReference, "messages[1].content[0]")
	if h != nil {
		t.Error("NewSeededHistory returned a non-nil *History alongside an error, want nil")
	}
}

// TestNewSeededHistory_RejectedSeed_ZeroValueUnusable (S-HIS-052) — a
// rejected seeded construction's returned history is not usable as a
// history through any public read or append route; the zero value never
// behaves as an empty valid transcript.
func TestNewSeededHistory_RejectedSeed_ZeroValueUnusable(t *testing.T) {
	t.Parallel()

	seed := []ai.Message{resultMessage(t, "call-missing")}
	h, err := agent.NewSeededHistory(seed)
	if err == nil {
		t.Fatal("NewSeededHistory(seed) returned nil error, want a rejection")
	}
	if h != nil {
		t.Fatalf("NewSeededHistory returned %v, want nil on rejection", h)
	}

	zero := new(agent.History)
	if got := zero.Len(); got != 0 {
		t.Errorf("new(History).Len() = %d, want 0", got)
	}
	if err := zero.Append(mustMessage(t, ai.RoleUser, mustText(t, "x"))); err == nil {
		t.Error("new(History).Append(...) returned nil error, want a rejection — the zero value is not a usable history")
	} else {
		requireViolation(t, err, ai.ErrEmpty, "history")
	}
}

// TestNewSeededHistory_SignatureAcceptsOnlyMessages (S-HIS-053) — the
// seeded constructor's public signature accepts an ordered sequence of
// Layer 1 messages and no other caller-supplied input: no parameter
// through which an entry identity or an origin discriminator could be
// provided.
func TestNewSeededHistory_SignatureAcceptsOnlyMessages(t *testing.T) {
	t.Parallel()

	funcType := funcDeclSignature(t, "history.go", "NewSeededHistory")
	if funcType == nil {
		t.Fatal("history.go declares no top-level function named NewSeededHistory")
	}

	if len(fieldsOrNil(funcType.Params)) != 1 {
		t.Fatalf("NewSeededHistory declares %d parameter field(s), want exactly 1 (an ordered sequence of Layer 1 messages and nothing else, S-HIS-053)", len(fieldsOrNil(funcType.Params)))
	}
	paramType := funcType.Params.List[0].Type
	arrayType, ok := paramType.(*ast.ArrayType)
	if !ok || arrayType.Len != nil {
		t.Fatalf("NewSeededHistory's parameter type is %s, want a slice ([]ai.Message)", signatureTypeString(paramType))
	}
	if !isQualifiedIdent(arrayType.Elt, "ai", "Message") {
		t.Errorf("NewSeededHistory's parameter element type is %s, want ai.Message", signatureTypeString(arrayType.Elt))
	}

	if len(fieldsOrNil(funcType.Results)) != 2 {
		t.Fatalf("NewSeededHistory declares %d result field(s), want exactly 2 (a constructed history and an error)", len(fieldsOrNil(funcType.Results)))
	}
	first, second := funcType.Results.List[0].Type, funcType.Results.List[1].Type
	star, ok := first.(*ast.StarExpr)
	if !ok || !isIdentNamed(star.X, "History") {
		t.Errorf("NewSeededHistory's first result type is %s, want *History", signatureTypeString(first))
	}
	if !isIdentNamed(second, "error") {
		t.Errorf("NewSeededHistory's second result type is %s, want error", signatureTypeString(second))
	}
}

// TestNewSeededHistory_EndsInOpenCall_AcceptedThenCloseTurnRejects
// (S-HIS-054) — a seed ending in one or more open calls is ACCEPTED;
// CloseTurn on that seeded history then rejects with the identical rule
// class and positional shape S-HIS-020 produces on an appended history.
// The unclosed-call rejection belongs to CloseTurn alone, on seeded and
// appended histories alike — a seed is not additionally turn-closed.
func TestNewSeededHistory_EndsInOpenCall_AcceptedThenCloseTurnRejects(t *testing.T) {
	t.Parallel()

	seed := []ai.Message{
		mustMessage(t, ai.RoleUser, mustText(t, "hi")),
		callMessage(t, "call-open"),
	}
	h, err := agent.NewSeededHistory(seed)
	if err != nil {
		t.Fatalf("NewSeededHistory(seed ending in an open call) returned %v, want nil (an open call is legal in a seed, R-HIS-006)", err)
	}
	entries := h.Entries()
	if len(entries) != len(seed) {
		t.Fatalf("Entries() returned %d entries, want %d — the seed reads back equal to the input", len(entries), len(seed))
	}

	err = h.CloseTurn()
	requireViolation(t, err, ai.ErrEmpty, "messages[1].content[0].result")
}

// TestHistory_ReadBack_UnmodifiedValuesInOrder (S-HIS-040) — the
// loop-facing view yields the committed Layer 1 values unmodified and in
// order; each value equals the one committed, without re-serialisation or
// field rewriting. Pinned explicitly even though Phase 1's GREEN already
// established this shape, per the coverage table.
func TestHistory_ReadBack_UnmodifiedValuesInOrder(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	committed := []ai.Message{
		mustMessage(t, ai.RoleUser, mustText(t, "one")),
		mustMessage(t, ai.RoleAssistant, mustText(t, "two")),
	}
	for _, m := range committed {
		if err := h.Append(m); err != nil {
			t.Fatalf("Append(...) returned %v, want nil", err)
		}
	}

	entries := h.Entries()
	for i, entry := range entries {
		if entry.Message().ID() != committed[i].ID() {
			t.Errorf("entries[%d].Message().ID() = %v, want %v (the same committed value, not re-serialised)", i, entry.Message().ID(), committed[i].ID())
		}
		if entry.Message().Role() != committed[i].Role() {
			t.Errorf("entries[%d].Message().Role() = %v, want %v", i, entry.Message().Role(), committed[i].Role())
		}
		if !entry.Message().Equal(committed[i]) {
			t.Errorf("entries[%d].Message() = %v, want %v", i, entry.Message(), committed[i])
		}
	}
}

// TestHistory_EntryIdentity_StableAcrossReadsAndSeed (S-HIS-041) — an
// entry's identity is equal across two reads of the same history, and
// across a second history constructed via NewSeededHistory over the same
// ordered seed — identity is a function of ordinal position alone.
func TestHistory_EntryIdentity_StableAcrossReadsAndSeed(t *testing.T) {
	t.Parallel()

	seed := []ai.Message{
		mustMessage(t, ai.RoleUser, mustText(t, "one")),
		mustMessage(t, ai.RoleAssistant, mustText(t, "two")),
	}
	appended := agent.NewHistory()
	for _, m := range seed {
		if err := appended.Append(m); err != nil {
			t.Fatalf("Append(...) returned %v, want nil", err)
		}
	}

	firstRead := appended.Entries()
	secondRead := appended.Entries()
	for i := range firstRead {
		if firstRead[i].ID() != secondRead[i].ID() {
			t.Errorf("entries[%d].ID() = %v on first read, %v on second — identity must be stable across reads", i, firstRead[i].ID(), secondRead[i].ID())
		}
	}

	seeded, err := agent.NewSeededHistory(seed)
	if err != nil {
		t.Fatalf("NewSeededHistory(seed) returned %v, want nil", err)
	}
	seededEntries := seeded.Entries()
	if len(seededEntries) != len(firstRead) {
		t.Fatalf("seeded Entries() returned %d entries, want %d", len(seededEntries), len(firstRead))
	}
	for i := range firstRead {
		if seededEntries[i].ID() != firstRead[i].ID() {
			t.Errorf("seeded entries[%d].ID() = %v, want %v (the appended original's identity — a function of ordinal position alone)", i, seededEntries[i].ID(), firstRead[i].ID())
		}
	}
}
