// AG-12.2 — orphan synthesis: every orphaned call is closed with a result
// recorded as an interruption artifact, distinguishable from a real one by
// the entry envelope's origin alone (R-HIS-007, R-HIS-008). External
// package agent_test for the same reason history_test.go is: the "any
// public route" audit is meaningful only against the exported surface
// (NFR-HIS-001). Helpers shared across this capability's three test files
// live in history_test.go, the first of the three to land.
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestHistory_SynthesizeOrphans_ClosesEveryOrphanDistinguishableByOrigin
// (S-HIS-060) — a transcript interrupted after tool calls were issued but
// before their results arrived: after synthesis, every orphaned call has
// a matching result, each such entry reports origin synthesized, and
// every pre-existing entry still reports origin appended.
func TestHistory_SynthesizeOrphans_ClosesEveryOrphanDistinguishableByOrigin(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	closedCall := callMessage(t, "call-closed")
	closedResult := resultMessage(t, "call-closed")
	orphan := callMessage(t, "call-orphan")
	for _, m := range []ai.Message{closedCall, closedResult, orphan} {
		if err := h.Append(m); err != nil {
			t.Fatalf("Append(...) returned %v, want nil", err)
		}
	}

	n, err := h.SynthesizeOrphans()
	if err != nil {
		t.Fatalf("SynthesizeOrphans() returned err = %v, want nil", err)
	}
	if n != 1 {
		t.Fatalf("SynthesizeOrphans() returned n = %d, want 1", n)
	}

	entries := h.Entries()
	if len(entries) != 4 {
		t.Fatalf("Entries() returned %d entries, want 4 (3 pre-existing + 1 synthesized)", len(entries))
	}
	wantOrigins := []agent.EntryOrigin{agent.EntryOriginAppended, agent.EntryOriginAppended, agent.EntryOriginAppended, agent.EntryOriginSynthesized}
	for i, want := range wantOrigins {
		if entries[i].Origin() != want {
			t.Errorf("entries[%d].Origin() = %v, want %v", i, entries[i].Origin(), want)
		}
	}

	synthesizedResult, ok := entries[3].Message().Content()[0].ToolResult()
	if !ok {
		t.Fatal("the synthesized entry's message does not carry a ToolResult part")
	}
	if synthesizedResult.CallID() != "call-orphan" {
		t.Errorf("synthesized result CallID() = %q, want %q", synthesizedResult.CallID(), "call-orphan")
	}
}

// TestHistory_SynthesizedVsReal_DistinguishableByOriginOnly (S-HIS-061) —
// a synthesized result and a real appended result constructed with
// byte-identical content and an identical Failed() outcome are still
// distinguished correctly, and the distinguishing assertion reads neither
// content nor Failed() — only Origin(). Content/Failed() equality is
// itself proven (not assumed) before the distinguishing assertion, so the
// test cannot pass by accident.
func TestHistory_SynthesizedVsReal_DistinguishableByOriginOnly(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	if err := h.Append(callMessage(t, "call-orphan")); err != nil {
		t.Fatalf("Append(orphan call) returned %v, want nil", err)
	}
	if _, err := h.SynthesizeOrphans(); err != nil {
		t.Fatalf("SynthesizeOrphans() returned %v, want nil", err)
	}

	entries := h.Entries()
	synthesizedEntry := entries[len(entries)-1]
	synthesizedResult, ok := synthesizedEntry.Message().Content()[0].ToolResult()
	if !ok {
		t.Fatal("synthesized entry does not carry a ToolResult part")
	}

	// Build a REAL result answering a DIFFERENT call, with content read
	// back from the synthesized result — byte-identical by construction
	// — and the same Failed() outcome.
	if err := h.Append(callMessage(t, "call-real")); err != nil {
		t.Fatalf("Append(real call) returned %v, want nil", err)
	}
	realPart, err := ai.NewToolFailure("call-real", synthesizedResult.Content())
	if err != nil {
		t.Fatalf("ai.NewToolFailure(...) returned %v, want nil", err)
	}
	if err := h.Append(mustMessage(t, ai.RoleTool, realPart)); err != nil {
		t.Fatalf("Append(real result) returned %v, want nil", err)
	}

	entries = h.Entries()
	realEntry := entries[len(entries)-1]
	realResult, ok := realEntry.Message().Content()[0].ToolResult()
	if !ok {
		t.Fatal("real entry does not carry a ToolResult part")
	}

	if realResult.Content() != synthesizedResult.Content() {
		t.Fatalf("test setup invalid: content differs (%q vs %q), want byte-identical", realResult.Content(), synthesizedResult.Content())
	}
	if realResult.Failed() != synthesizedResult.Failed() {
		t.Fatalf("test setup invalid: Failed() differs (%v vs %v), want identical — otherwise this test could distinguish them by Failed() instead of Origin()", realResult.Failed(), synthesizedResult.Failed())
	}

	if synthesizedEntry.Origin() != agent.EntryOriginSynthesized {
		t.Errorf("synthesized entry Origin() = %v, want %v", synthesizedEntry.Origin(), agent.EntryOriginSynthesized)
	}
	if realEntry.Origin() != agent.EntryOriginAppended {
		t.Errorf("real entry Origin() = %v, want %v", realEntry.Origin(), agent.EntryOriginAppended)
	}
}

// TestHistory_SynthesizeOrphans_TurnClosesAfterSynthesis (S-HIS-062) —
// once synthesis has run, closing the turn succeeds: synthesis produced a
// transcript the pairing invariant accepts, through the same commit path.
func TestHistory_SynthesizeOrphans_TurnClosesAfterSynthesis(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	if err := h.Append(callMessage(t, "call-orphan")); err != nil {
		t.Fatalf("Append(orphan) returned %v, want nil", err)
	}

	if _, err := h.SynthesizeOrphans(); err != nil {
		t.Fatalf("SynthesizeOrphans() returned %v, want nil", err)
	}

	if err := h.CloseTurn(); err != nil {
		t.Fatalf("CloseTurn() after synthesis returned %v, want nil", err)
	}
}

// TestHistory_SynthesizeOrphans_ClosesExactlyN (S-HIS-070) — over a
// transcript holding N (>= 2) orphaned calls interleaved with
// already-matched calls, synthesis closes exactly N pairs on the first
// application; no already-matched call gains a second result. Pinned
// explicitly even though Phase 6's open-set logic already establishes
// this shape, per the coverage table.
func TestHistory_SynthesizeOrphans_ClosesExactlyN(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	messages := []ai.Message{
		callMessage(t, "call-matched-1"),
		resultMessage(t, "call-matched-1"),
		callMessage(t, "call-orphan-1"),
		callMessage(t, "call-matched-2"),
		resultMessage(t, "call-matched-2"),
		callMessage(t, "call-orphan-2"),
	}
	for _, m := range messages {
		if err := h.Append(m); err != nil {
			t.Fatalf("Append(...) returned %v, want nil", err)
		}
	}

	n, err := h.SynthesizeOrphans()
	if err != nil {
		t.Fatalf("SynthesizeOrphans() returned err = %v, want nil", err)
	}
	if n != 2 {
		t.Fatalf("SynthesizeOrphans() returned n = %d, want 2 (exactly the orphan count)", n)
	}

	entries := h.Entries()
	if len(entries) != len(messages)+2 {
		t.Fatalf("Entries() returned %d entries, want %d (%d pre-existing + 2 synthesized)", len(entries), len(messages)+2, len(messages))
	}

	// Closing the turn must now succeed: no unclosed call remains.
	if err := h.CloseTurn(); err != nil {
		t.Fatalf("CloseTurn() after synthesis returned %v, want nil", err)
	}

	resultCount := map[string]int{}
	for _, entry := range entries {
		result, ok := entry.Message().Content()[0].ToolResult()
		if !ok {
			continue
		}
		resultCount[result.CallID()]++
	}
	for _, callID := range []string{"call-matched-1", "call-matched-2", "call-orphan-1", "call-orphan-2"} {
		if resultCount[callID] != 1 {
			t.Errorf("resultCount[%q] = %d, want exactly 1 (no already-matched call gained a second result)", callID, resultCount[callID])
		}
	}
}

// TestHistory_SynthesizeOrphans_SecondApplicationNoOp (S-HIS-071) — a
// second application after the first changes nothing: the read-back
// transcript is identical to the first-application transcript,
// entry-for-entry, including identities and origins, and no new entry is
// committed.
func TestHistory_SynthesizeOrphans_SecondApplicationNoOp(t *testing.T) {
	t.Parallel()

	h := agent.NewHistory()
	if err := h.Append(callMessage(t, "call-orphan")); err != nil {
		t.Fatalf("Append(...) returned %v, want nil", err)
	}

	firstN, err := h.SynthesizeOrphans()
	if err != nil {
		t.Fatalf("first SynthesizeOrphans() returned %v, want nil", err)
	}
	if firstN != 1 {
		t.Fatalf("first SynthesizeOrphans() returned n = %d, want 1", firstN)
	}
	afterFirst := h.Entries()

	secondN, err := h.SynthesizeOrphans()
	if err != nil {
		t.Fatalf("second SynthesizeOrphans() returned %v, want nil", err)
	}
	if secondN != 0 {
		t.Errorf("second SynthesizeOrphans() returned n = %d, want 0 (no orphans remain)", secondN)
	}

	afterSecond := h.Entries()
	if len(afterSecond) != len(afterFirst) {
		t.Fatalf("Entries() after the second application returned %d entries, want %d unchanged", len(afterSecond), len(afterFirst))
	}
	for i := range afterFirst {
		changed := afterSecond[i].ID() != afterFirst[i].ID() ||
			afterSecond[i].Origin() != afterFirst[i].Origin() ||
			!afterSecond[i].Message().Equal(afterFirst[i].Message())
		if changed {
			t.Errorf("entries[%d] changed after a no-op second application: got %v, want %v", i, afterSecond[i], afterFirst[i])
		}
	}
}
