// AI-28.1.2 — stream_state.go's own test suite: R-ATS-008's block minting
// (S-ATS-028…032) and R-ATS-007's multi-chunk delta sequencing (S-ATS-024).
// Internal package, direct seam (design.md's Testing Strategy): every test
// here drives mapperState.applyChunk with already-decoded wireChunk values,
// with no real HTTP and no Decoder involved — chunk_test.go covers one
// chunk's own trichotomy in isolation; this file covers the mapper's state
// across a sequence of chunks.

package openaicompat

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// mustChunk decodes data via decodeChunk, failing the test on error.
func mustChunk(t *testing.T, data string) wireChunk {
	t.Helper()
	chunk, err := decodeChunk([]byte(data))
	if err != nil {
		t.Fatalf("decodeChunk(%q) error = %v, want nil", data, err)
	}
	return chunk
}

// driveChunks applies every chunk to state in order, failing the test on
// the first error, and returns every emitted event flattened into one
// ordered slice — this file's own fixture-driving helper.
func driveChunks(t *testing.T, state *mapperState, chunks ...wireChunk) []ai.Event {
	t.Helper()
	var all []ai.Event
	for i, c := range chunks {
		events, err := state.applyChunk(c)
		if err != nil {
			t.Fatalf("applyChunk(chunk %d) error = %v, want nil", i, err)
		}
		all = append(all, events...)
	}
	return all
}

// kindsOf projects events onto their kinds, for a compact sequence
// assertion.
func kindsOf(events []ai.Event) []ai.EventKind {
	kinds := make([]ai.EventKind, len(events))
	for i, ev := range events {
		kinds[i] = ev.Kind()
	}
	return kinds
}

// ---------------------------------------------------------------------
// R-ATS-007 — multi-chunk delta sequencing (S-ATS-024).
// ---------------------------------------------------------------------

// TestMapperState_ThreeContentChunks_EmitsThreeOrderedDeltas covers
// S-ATS-024: three chunks whose choice-0 content values are "Hola", ", "
// and "mundo" each become exactly one delta, in order.
func TestMapperState_ThreeContentChunks_EmitsThreeOrderedDeltas(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	events := driveChunks(t, state,
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"Hola"},"finish_reason":null}]}`),
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"content":", "},"finish_reason":null}]}`),
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"content":"mundo"},"finish_reason":null}]}`),
	)

	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}

	want := []string{"Hola", ", ", "mundo"}
	if len(deltas) != len(want) {
		t.Fatalf("emitted %d delta(s) = %q, want %d = %q (S-ATS-024)", len(deltas), deltas, len(want), want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (S-ATS-024)", i, deltas[i], want[i])
		}
	}
}

// ---------------------------------------------------------------------
// R-ATS-008 — block boundaries minted by the adapter (S-ATS-028…032).
// ---------------------------------------------------------------------

// TestMapperState_TwoContentChunksAndTerminal_EmitsSixKindsInOrder covers
// S-ATS-028: the emitted kinds are exactly ResponseStart, TextBlockStart,
// TextDelta, TextDelta, TextBlockEnd, Completion, in that order — the same
// six-kind window the landed conformance amendment's textOrderingCase
// derives (R-CNF-019).
func TestMapperState_TwoContentChunksAndTerminal_EmitsSixKindsInOrder(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	events := driveChunks(t, state,
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"content":"un"},"finish_reason":null}]}`),
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"content":" dos"},"finish_reason":null}]}`),
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`),
	)
	completion, err := state.buildCompletion()
	if err != nil {
		t.Fatalf("buildCompletion() error = %v, want nil (S-ATS-028)", err)
	}
	events = append(events, completion)

	want := []ai.EventKind{
		ai.EventKindResponseStart,
		ai.EventKindTextBlockStart,
		ai.EventKindTextDelta,
		ai.EventKindTextDelta,
		ai.EventKindTextBlockEnd,
		ai.EventKindCompletion,
	}
	got := kindsOf(events)
	if len(got) != len(want) {
		t.Fatalf("emitted kinds = %v (%d), want %v (%d) (S-ATS-028)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("kind[%d] = %v, want %v (S-ATS-028)", i, got[i], want[i])
		}
	}

	// S-ATS-029: every text-block-scoped event reports block index 1.
	blockScoped := 0
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindTextBlockStart:
			blockScoped++
			if s, _ := ev.TextBlockStart(); s.Block() != 1 {
				t.Errorf("TextBlockStart.Block() = %d, want 1 (S-ATS-029)", s.Block())
			}
		case ai.EventKindTextDelta:
			blockScoped++
			if d, _ := ev.TextDelta(); d.Block() != 1 {
				t.Errorf("TextDelta.Block() = %d, want 1 (S-ATS-029)", d.Block())
			}
		case ai.EventKindTextBlockEnd:
			blockScoped++
			if e, _ := ev.TextBlockEnd(); e.Block() != 1 {
				t.Errorf("TextBlockEnd.Block() = %d, want 1 (S-ATS-029)", e.Block())
			}
		}
	}
	if blockScoped == 0 {
		t.Fatal("no block-scoped text event found — fixture premise broken, S-ATS-029 asserts nothing")
	}
}

// TestMapperState_RoleOnlyOpeningThenContent_BlockStartBetweenThem covers
// S-ATS-030: the block start is emitted between a role-only opening chunk
// (which produces no text event at all) and the chunk carrying the first
// content string, immediately before that first delta.
func TestMapperState_RoleOnlyOpeningThenContent_BlockStartBetweenThem(t *testing.T) {
	t.Parallel()

	state := &mapperState{}

	opening, err := state.applyChunk(mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`))
	if err != nil {
		t.Fatalf("applyChunk(opening) error = %v, want nil", err)
	}
	if got := kindsOf(opening); len(got) != 1 || got[0] != ai.EventKindResponseStart {
		t.Fatalf("opening chunk emitted kinds = %v, want exactly [ResponseStart] (S-ATS-030)", got)
	}

	content, err := state.applyChunk(mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"content":"hola"},"finish_reason":null}]}`))
	if err != nil {
		t.Fatalf("applyChunk(content) error = %v, want nil", err)
	}
	want := []ai.EventKind{ai.EventKindTextBlockStart, ai.EventKindTextDelta}
	if got := kindsOf(content); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("content chunk emitted kinds = %v, want %v — block start immediately before the first delta (S-ATS-030)", got, want)
	}
}

// TestMapperState_NoContentAnywhere_NoBlockMintedAtAll covers S-ATS-031: a
// role-only opening chunk then a terminal chunk, with no content string
// anywhere, mints no TextBlockStart and no TextBlockEnd, and the resulting
// stream reports no unterminated-block violation under ai.CheckStream.
func TestMapperState_NoContentAnywhere_NoBlockMintedAtAll(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	events := driveChunks(t, state,
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`),
		mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`),
	)
	completion, err := state.buildCompletion()
	if err != nil {
		t.Fatalf("buildCompletion() error = %v, want nil (S-ATS-031)", err)
	}
	events = append(events, completion)

	for _, ev := range events {
		if ev.Kind() == ai.EventKindTextBlockStart || ev.Kind() == ai.EventKindTextBlockEnd {
			t.Errorf("emitted %v, want no text-block event at all when no content string ever arrived (S-ATS-031)", ev.Kind())
		}
	}

	// Stamp the events so ai.CheckStream (which requires a real sequence
	// per its own contract) can run over them.
	var stamper ai.Stamper
	stamped := make([]ai.Event, len(events))
	for i, ev := range events {
		stamped[i] = stamper.Stamp(ev)
	}
	if report := ai.CheckStream(stamped); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil — no unterminated-block violation for a stream that never opened one (S-ATS-031)", report.Violation())
	}
}

// TestMapperState_TerminalThenDeltaLessAndUsageChunks_ClosesCleanly covers
// S-ATS-032: once the terminal chunk closes the block, subsequent
// delta-less chunks and the usage chunk (an empty choices array, C4) in the
// terminal→sentinel window are absorbed — zero events — and the stream
// still reaches a normal completion with the block end preceding it.
func TestMapperState_TerminalThenDeltaLessAndUsageChunks_ClosesCleanly(t *testing.T) {
	t.Parallel()

	state := &mapperState{}

	// This is the very first chunk applied to a fresh mapperState, so it
	// also establishes identity (ResponseStart) alongside the block open
	// and first delta — S-ATS-030 covers the role-only-opening-chunk shape
	// separately; this test's own focus is the terminal→sentinel window.
	content, err := state.applyChunk(mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`))
	if err != nil {
		t.Fatalf("applyChunk(content) error = %v, want nil", err)
	}
	wantContentKinds := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindTextBlockStart, ai.EventKindTextDelta}
	if got := kindsOf(content); len(got) != len(wantContentKinds) || got[0] != wantContentKinds[0] || got[1] != wantContentKinds[1] || got[2] != wantContentKinds[2] {
		t.Fatalf("content chunk emitted kinds = %v, want %v", got, wantContentKinds)
	}

	terminal, err := state.applyChunk(mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(terminal) error = %v, want nil", err)
	}
	if got := kindsOf(terminal); len(got) != 1 || got[0] != ai.EventKindTextBlockEnd {
		t.Fatalf("terminal chunk emitted kinds = %v, want exactly [TextBlockEnd] (S-ATS-032)", got)
	}

	// A delta-less chunk in the terminal→sentinel window (C4's own window)
	// — carries a choice item, but no content and no (re-)finish_reason.
	deltaLess, err := state.applyChunk(mustChunk(t, `{"id":"r1","model":"m","choices":[{"index":0,"delta":{},"finish_reason":null}]}`))
	if err != nil {
		t.Fatalf("applyChunk(deltaLess) error = %v, want nil (S-ATS-032)", err)
	}
	if len(deltaLess) != 0 {
		t.Errorf("delta-less post-terminal chunk emitted %d event(s) = %v, want 0 — absorbed, never a delta (S-ATS-032)", len(deltaLess), deltaLess)
	}

	// The usage chunk itself (C4): an empty choices array.
	usage, err := state.applyChunk(mustChunk(t, `{"id":"r1","model":"m","choices":[],"usage":{"prompt_tokens":3,"completion_tokens":5,"total_tokens":8}}`))
	if err != nil {
		t.Fatalf("applyChunk(usage) error = %v, want nil (S-ATS-032)", err)
	}
	if len(usage) != 0 {
		t.Errorf("usage chunk emitted %d event(s) = %v, want 0 (S-ATS-032)", len(usage), usage)
	}

	completion, err := state.buildCompletion()
	if err != nil {
		t.Fatalf("buildCompletion() error = %v, want nil — the stream must terminate in a normal completion (S-ATS-032)", err)
	}
	if completion.Kind() != ai.EventKindCompletion {
		t.Fatalf("buildCompletion() kind = %v, want EventKindCompletion (S-ATS-032)", completion.Kind())
	}
}
