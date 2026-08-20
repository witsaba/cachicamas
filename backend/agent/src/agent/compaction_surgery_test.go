// AG-18.2 — Phase 1: pure cut resolution and invariant-safe surgery
// (R-CMP-004..007, R-HIS-005, R-HIS-103).
//
// resolveCut is an unexported pure helper with no external caller yet in
// this package's own scope at the point this file lands (NFR-CMP-001's
// own carve-out: "Pure helpers with no external surface MAY be tested
// internally, provided every claim about what a caller observes is also
// asserted externally"). It is tested here directly, package agent,
// mirroring retry_decision_internal_test.go's precedent for an
// unexported predicate (NFR-RTY-001's own carve-out). Every claim this
// file makes about what an external caller observes is reproved from
// package agent_test once the full compaction pipeline exists
// (compaction_stream_test.go, compaction_reconstruction_test.go).
//
// Building a marks-bearing History from this file is direct — this
// package has access to the unexported closeTurnMarked door — unlike an
// external test, which can only get one by driving a real Harness.Run
// (S-CMP-035: closeTurnMarked is not reachable from package agent_test).
package agent

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// mustPlainMessage builds a well-formed assistant text message, failing
// t on any construction violation.
func mustPlainMessage(t *testing.T, text string) ai.Message {
	t.Helper()
	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText(%q) returned %v, want nil", text, err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want nil", err)
	}
	return msg
}

// mustCallMessage builds a well-formed assistant message carrying one
// tool call.
func mustCallMessage(t *testing.T, callID string) ai.Message {
	t.Helper()
	part, err := ai.NewToolCall(callID, "a-tool", nil)
	if err != nil {
		t.Fatalf("ai.NewToolCall(%q, ...) returned %v, want nil", callID, err)
	}
	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want nil", err)
	}
	return msg
}

// mustResultMessage builds a well-formed tool-role message answering
// callID.
func mustResultMessage(t *testing.T, callID string) ai.Message {
	t.Helper()
	part, err := ai.NewToolResult(callID, "ok")
	if err != nil {
		t.Fatalf("ai.NewToolResult(%q, ...) returned %v, want nil", callID, err)
	}
	msg, err := ai.NewMessage(ai.RoleTool, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want nil", err)
	}
	return msg
}

// appendAndMark appends every message in msgs to h, in order, then
// records a turn mark under turnID — this file's own minimal stand-in
// for "a turn driven through Harness.Run and closed", built directly
// against the package-private surface rather than a live harness.
func appendAndMark(t *testing.T, h *History, turnID TurnID, msgs ...ai.Message) {
	t.Helper()
	for _, m := range msgs {
		if err := h.Append(m); err != nil {
			t.Fatalf("Append returned %v, want nil", err)
		}
	}
	if err := h.closeTurnMarked(turnID); err != nil {
		t.Fatalf("closeTurnMarked(%q) returned %v, want nil", turnID, err)
	}
}

// twoTurnStraddleFixture builds the S-CMP-010 fixture: turn "turn-a"
// commits one plain entry (index 0) and closes marked; turn "turn-b"
// commits a tool call at index 1 (i) and its matching result at index 2
// (j), then closes marked. It returns the history and the pair's
// indices, so callers can pick a naive cut strictly between them.
func twoTurnStraddleFixture(t *testing.T) (hist *History, callIndex, resultIndex int) {
	t.Helper()
	hist = NewHistory()
	appendAndMark(t, hist, "turn-a", mustPlainMessage(t, "turn a reply"))
	appendAndMark(t, hist, "turn-b", mustCallMessage(t, "call-straddle"), mustResultMessage(t, "call-straddle"))
	return hist, 1, 2
}

// TestResolveCut_RetractsToMarkBoundary — S-CMP-010, charter AG-18.2
// scenario 1. A naive cut falling strictly between a tool call and its
// matching result retracts to the greatest mark boundary at or before
// the call's own turn — never splitting the pair.
func TestResolveCut_RetractsToMarkBoundary(t *testing.T) {
	hist, callIndex, resultIndex := twoTurnStraddleFixture(t)
	naiveCut := resultIndex // entries[:resultIndex] includes the call, excludes the result.

	got := resolveCut(hist, naiveCut)

	if got > callIndex {
		t.Fatalf("resolveCut(hist, %d) = %d, want <= %d (the call's own entry index) — the boundary must not split the call/result pair", naiveCut, got, callIndex)
	}
	if got != 1 {
		t.Fatalf("resolveCut(hist, %d) = %d, want 1 (turn-a's own mark boundary)", naiveCut, got)
	}

	entries := hist.Entries()
	tail := entries[got:]
	if len(tail) != 2 {
		t.Fatalf("protected tail has %d entr(y/ies), want 2 (both halves of the straddling pair)", len(tail))
	}
	callPart, ok := tail[0].Message().Content()[0].ToolCall()
	if !ok || callPart.ID() != "call-straddle" {
		t.Errorf("tail[0] is not the whole call, want the call-straddle tool call")
	}
	resultPart, ok := tail[1].Message().Content()[0].ToolResult()
	if !ok || resultPart.CallID() != "call-straddle" {
		t.Errorf("tail[1] is not the whole result, want the result answering call-straddle")
	}
}

// TestResolveCut_SeededValidation — S-CMP-011, the validation half of
// AG-18.2 scenario 1. The transcript resolveCut, then ReplacePrefix,
// produce passes the same append-time pairing rules a fresh seed
// enforces.
func TestResolveCut_SeededValidation(t *testing.T) {
	hist, _, resultIndex := twoTurnStraddleFixture(t)
	cut := resolveCut(hist, resultIndex)
	if cut == 0 {
		t.Fatal("resolveCut returned 0, want a positive resolved cut")
	}

	summary := mustPlainMessage(t, "a summary")
	if err := hist.ReplacePrefix(cut, summary); err != nil {
		t.Fatalf("ReplacePrefix(%d, summary) returned %v, want nil", cut, err)
	}

	readBack := hist.Entries()
	messages := make([]ai.Message, len(readBack))
	for i, e := range readBack {
		messages[i] = e.Message()
	}
	if _, err := NewSeededHistory(messages); err != nil {
		t.Errorf("NewSeededHistory(post-compaction read-back) returned %v, want nil — the post-replacement transcript must satisfy the same rules a fresh seed enforces", err)
	}
}

// TestResolveCut_MonotoneNeverExpandsForward — S-CMP-012. Retraction is
// monotone: the resolved cut never exceeds the naive cut, across every
// fixture this file drives, and the protected tail after compaction is
// a superset of the entries at or after the naive cut.
func TestResolveCut_MonotoneNeverExpandsForward(t *testing.T) {
	tests := []struct {
		name  string
		naive int
	}{
		{"exact mark boundary", 1},
		{"straddling the pair", 2},
		{"at the very end", 3},
		{"beyond the end", 99},
		{"zero", 0},
		{"negative", -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hist, _, _ := twoTurnStraddleFixture(t)
			naive := tt.naive
			resolved := resolveCut(hist, naive)

			if naive >= 0 && resolved > naive {
				t.Errorf("resolveCut(hist, %d) = %d, want <= %d (retraction never expands forward)", naive, resolved, naive)
			}
			if naive < 0 && resolved != 0 {
				t.Errorf("resolveCut(hist, %d) = %d, want 0 (a negative naive cut has nothing to compact)", naive, resolved)
			}

			// The protected tail at the RESOLVED cut is always a
			// superset of the entries the NAIVE cut alone would have
			// protected (index >= naive, clamped to the transcript
			// length) — S-CMP-012's superset half.
			total := hist.Len()
			naiveClamped := naive
			if naiveClamped < 0 {
				naiveClamped = 0
			}
			if naiveClamped > total {
				naiveClamped = total
			}
			if resolved > naiveClamped {
				t.Errorf("resolved cut %d protects fewer entries than the naive cut %d would have — not a superset", resolved, naiveClamped)
			}
		})
	}
}

// nestedStraddleFixture builds S-CMP-013's adversarial input: one
// marked turn, then a SECOND marked turn carrying two interleaved open
// calls (issued back to back) each closed by its own result, so a
// naive cut landing inside it can straddle either pair depending on
// where it falls.
func nestedStraddleFixture(t *testing.T) *History {
	t.Helper()
	hist := NewHistory()
	appendAndMark(t, hist, "turn-a", mustPlainMessage(t, "turn a reply"))
	appendAndMark(t, hist, "turn-b",
		mustCallMessage(t, "call-nest-1"),
		mustCallMessage(t, "call-nest-2"),
		mustResultMessage(t, "call-nest-1"),
		mustResultMessage(t, "call-nest-2"),
	)
	return hist
}

// TestResolveCut_TerminatesOnAdversarialInput — S-CMP-013. A naive cut
// splitting more than one pair still resolves: termination is
// structural (strictly decreasing, bounded below by 0), the resolved
// cut is strictly less than the naive cut, and the resolved prefix is
// pairing-closed. A transcript whose every entry belongs to one
// straddling pair (no earlier mark at all) resolves to 0.
func TestResolveCut_TerminatesOnAdversarialInput(t *testing.T) {
	hist := nestedStraddleFixture(t)
	// entries: 0=turn-a reply, 1=call-nest-1, 2=call-nest-2,
	// 3=result-nest-1, 4=result-nest-2. A naive cut of 4 splits
	// call-nest-2/result-nest-2 (issued at 2, answered at 4).
	naiveCut := 4
	resolved := resolveCut(hist, naiveCut)
	if resolved >= naiveCut {
		t.Fatalf("resolveCut(hist, %d) = %d, want strictly < %d", naiveCut, resolved, naiveCut)
	}
	if resolved != 1 {
		t.Fatalf("resolveCut(hist, %d) = %d, want 1 (turn-a's own mark, the only boundary at or before both open calls)", naiveCut, resolved)
	}

	// A history whose every entry belongs to one straddling pair, with
	// no mark at or before it at all, resolves to 0 (nothing to
	// compact) — the other half of S-CMP-013.
	unmarked := NewHistory()
	if err := unmarked.Append(mustCallMessage(t, "call-unmarked")); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}
	if err := unmarked.Append(mustResultMessage(t, "call-unmarked")); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}
	if got := resolveCut(unmarked, 2); got != 0 {
		t.Errorf("resolveCut(unmarked, 2) = %d, want 0 (no recorded mark exists at or before the naive cut)", got)
	}
}

// TestCompaction_ProtectionValueIdentical — S-CMP-014, S-HIS-103,
// charter AG-18.2 scenario 2 (protection half). Every protected entry's
// Layer 1 value and origin discriminator compare equal before and
// after, positionally — never compared as whole Entry structs.
//
// reflect.DeepEqual over whole Entry values is FORBIDDEN in this
// assertion (R-CMP-006, design AD-8): Entry embeds the ordinal EntryID,
// so DeepEqual either fails spuriously or, on a fixture that does not
// shift ordinals, encodes the false claim that identities survive
// compaction. This file contains no such call — grepped at review.
func TestCompaction_ProtectionValueIdentical(t *testing.T) {
	hist := NewHistory()
	appendAndMark(t, hist, "turn-a", mustPlainMessage(t, "reply a-1"), mustPlainMessage(t, "reply a-2"))
	appendAndMark(t, hist, "turn-b", mustPlainMessage(t, "reply b-1"), mustPlainMessage(t, "reply b-2"))

	pre := hist.Entries()
	if len(pre) != 4 {
		t.Fatalf("fixture carries %d entries, want 4", len(pre))
	}

	cut := 2 // replace turn-a's two entries; protect turn-b's two.
	summary := mustPlainMessage(t, "summary of turn a")
	if err := hist.ReplacePrefix(cut, summary); err != nil {
		t.Fatalf("ReplacePrefix(%d, summary) returned %v, want nil", cut, err)
	}

	post := hist.Entries()
	if len(post) != 3 {
		t.Fatalf("post-compaction entry count = %d, want 3 (1 summary + 2 protected)", len(post))
	}

	for i := 0; i < 2; i++ {
		preEntry := pre[cut+i]
		postEntry := post[1+i]
		if !preEntry.Message().Equal(postEntry.Message()) {
			t.Errorf("protected entry %d: post message does not equal pre message", i)
		}
		if preEntry.Origin() != postEntry.Origin() {
			t.Errorf("protected entry %d: origin changed from %v to %v, want unchanged", i, preEntry.Origin(), postEntry.Origin())
		}
	}
}

// TestCompaction_IdentityRenumbers — S-CMP-015, the identity half,
// asserted rather than assumed. At least one protected entry's ID()
// differs pre/post; post-compaction identities are exactly the new
// ordinal sequence — so no reader takes the charter's "byte-identical"
// to mean identity survived.
func TestCompaction_IdentityRenumbers(t *testing.T) {
	hist := NewHistory()
	appendAndMark(t, hist, "turn-a", mustPlainMessage(t, "reply a-1"), mustPlainMessage(t, "reply a-2"))
	appendAndMark(t, hist, "turn-b", mustPlainMessage(t, "reply b-1"), mustPlainMessage(t, "reply b-2"))

	pre := hist.Entries()
	cut := 2
	summary := mustPlainMessage(t, "summary of turn a")
	if err := hist.ReplacePrefix(cut, summary); err != nil {
		t.Fatalf("ReplacePrefix(%d, summary) returned %v, want nil", cut, err)
	}
	post := hist.Entries()

	changed := false
	for i := 0; i < 2; i++ {
		if pre[cut+i].ID() != post[1+i].ID() {
			changed = true
		}
	}
	if !changed {
		t.Error("no protected entry's identity changed, want at least one to differ pre/post")
	}
	for i, e := range post {
		if want := EntryID(i + 1); e.ID() != want {
			t.Errorf("post[%d].ID() = %v, want %v (the new ordinal sequence)", i, e.ID(), want)
		}
	}
}

// TestCompaction_SummaryTypedByOriginOnly — S-CMP-016, charter AG-18.2
// scenario 2 (typing half). The summary entry is distinguished from a
// byte-identical-content, ordinarily-appended control message by its
// origin alone — content is never read to tell them apart.
func TestCompaction_SummaryTypedByOriginOnly(t *testing.T) {
	const sameContent = "identical bytes either way"

	hist := NewHistory()
	appendAndMark(t, hist, "turn-a", mustPlainMessage(t, "reply a-1"))
	if err := hist.ReplacePrefix(1, mustPlainMessage(t, sameContent)); err != nil {
		t.Fatalf("ReplacePrefix returned %v, want nil", err)
	}
	summaryEntry := hist.Entries()[0]

	control := NewHistory()
	if err := control.Append(mustPlainMessage(t, sameContent)); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}
	controlEntry := control.Entries()[0]

	if !summaryEntry.Message().Equal(controlEntry.Message()) {
		t.Fatal("fixture defect: summary and control messages are not content-equal")
	}
	if summaryEntry.Origin() != EntryOriginSummarized {
		t.Errorf("summaryEntry.Origin() = %v, want EntryOriginSummarized", summaryEntry.Origin())
	}
	if controlEntry.Origin() != EntryOriginAppended {
		t.Errorf("controlEntry.Origin() = %v, want EntryOriginAppended", controlEntry.Origin())
	}
	if summaryEntry.Origin() == controlEntry.Origin() {
		t.Error("summary and control origins are equal, want distinguishable")
	}
}

// TestResolveCut_FixedPointOnItsOwnOutput — AG-20, S-CMP-040 (design
// AD-8's idempotence proof, exercised under NFR-CMP-001's pure-helper
// carve-out: resolveCut already has an external caller in this
// package's own scope by the time AG-20 lands, and this table asserts
// its fixed-point property directly rather than assuming it). The
// carve-out's own condition — every claim about what a CALLER observes
// is also asserted externally — is satisfied by
// hooks_compaction_test.go's TestHooksCompaction_Idempotence_
// IdenticalPlanByteIdenticalToNoHook (S-HKS-007 / S-CMP-038), which
// asserts the OBSERVABLE consequence (a hook returning its input plan
// unchanged produces a byte-identical stream and history read-back to
// the hookless run) from package agent_test.
//
// Over the same class of naive cuts this file's own suite already
// drives — zero, on a mark, mid-mark (straddling a pair), straddling a
// nested open pair, and beyond the transcript length — resolving a
// cut's own output a second time MUST return that output unchanged:
// resolveCut(hist, resolveCut(hist, n)) == resolveCut(hist, n).
func TestResolveCut_FixedPointOnItsOwnOutput(t *testing.T) {
	tests := []struct {
		name  string
		build func(t *testing.T) (*History, int)
	}{
		{"zero", func(t *testing.T) (*History, int) {
			hist, _, _ := twoTurnStraddleFixture(t)
			return hist, 0
		}},
		{"on a mark", func(t *testing.T) (*History, int) {
			hist, _, _ := twoTurnStraddleFixture(t)
			return hist, 1
		}},
		{"mid-mark, straddling a pair", func(t *testing.T) (*History, int) {
			hist, _, resultIndex := twoTurnStraddleFixture(t)
			return hist, resultIndex
		}},
		{"straddling a nested open pair", func(t *testing.T) (*History, int) {
			hist := nestedStraddleFixture(t)
			return hist, 4
		}},
		{"beyond the transcript length", func(t *testing.T) (*History, int) {
			hist, _, _ := twoTurnStraddleFixture(t)
			return hist, 99
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hist, naive := tt.build(t)
			first := resolveCut(hist, naive)
			second := resolveCut(hist, first)
			if second != first {
				t.Errorf("resolveCut(hist, resolveCut(hist, %d)) = %d, want %d (resolution is a fixed point on its own output; naive cut was %d)", naive, second, first, naive)
			}
		})
	}
}
