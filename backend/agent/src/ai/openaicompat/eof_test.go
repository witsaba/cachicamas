package openaicompat_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// This file covers Phase 6 (AI-27.6, EOF discipline): R-ASD-022 (a clean
// end at a frame boundary finishes without error, S-ASD-070..072),
// R-ASD-023 (an end mid-frame is a typed truncation error and the
// partial frame is discarded, S-ASD-073..076), and R-ASD-024
// (end-of-input is a decoder signal, not a transport condition,
// S-ASD-077). S-ASD-078 (no io.EOF/io.ErrUnexpectedEOF anywhere in
// decoder source) and the milestone-close pair S-ASD-082/083 are
// [inspection] scenarios, discharged by trace in this slice's evidence
// log rather than by a test function here.
//
// Test names start with TestEOF so the slice's focused command
// (`go test -race -run TestEOF ./src/ai/openaicompat/...`) selects
// exactly this file's cases (see tasks.md's Suggested Work Units table).

// --- R-ASD-022: a clean end at a frame boundary finishes without error ---

// TestEOF_CleanEndAtFrameBoundaryFinishesWithoutError covers S-ASD-070: a
// transcript ending with complete frames and their terminating blank
// lines reports no error from Finish, and the expected frames were
// already yielded by Feed.
//
// Deliberately TWO frames, not one — real, non-trivial content, per this
// milestone's own evidence discipline: a clean-end test built only on an
// empty stream (S-ASD-071, its own separate scenario below) could not
// catch an over-aggressive truncation check that wrongly flags leftover
// state a correct implementation must have already reset after an
// earlier dispatch. Two dispatches before Finish is what actually
// exercises that.
func TestEOF_CleanEndAtFrameBoundaryFinishesWithoutError(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: greet\ndata: hello\n\ndata: world\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 2 {
		t.Fatalf("Feed() yielded %d frames, want exactly 2", len(frames))
	}
	if frames[0].Event != "greet" || !bytes.Equal(frames[0].Data, []byte("hello")) {
		t.Errorf("frames[0] = %+v, want Event %q Data %q", frames[0], "greet", "hello")
	}
	if frames[1].Event != "message" || !bytes.Equal(frames[1].Data, []byte("world")) {
		t.Errorf("frames[1] = %+v, want Event %q Data %q", frames[1], "message", "world")
	}

	if err := d.Finish(); err != nil {
		t.Errorf("Finish() error = %v, want nil — a clean end at a frame boundary must not report truncation (S-ASD-070)", err)
	}
}

// TestEOF_EmptyStreamFinishesCleanWithZeroFrames covers S-ASD-071: a
// decoder that never received any bytes at all finishes cleanly, with no
// error.
func TestEOF_EmptyStreamFinishesCleanWithZeroFrames(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	if err := d.Finish(); err != nil {
		t.Errorf("Finish() error = %v, want nil — an empty stream must finish cleanly (S-ASD-071)", err)
	}
}

// TestEOF_CommentOnlyStreamEndingAtLineBoundaryFinishesClean covers
// S-ASD-072: a stream of comment lines only, ending at a line boundary,
// reports no error from Finish — a comment line leaves no accumulator
// state of its own for Finish to (wrongly) flag.
func TestEOF_CommentOnlyStreamEndingAtLineBoundaryFinishesClean(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(": keep-alive one\n: keep-alive two\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Fatalf("Feed() yielded %d frames, want 0", len(frames))
	}

	if err := d.Finish(); err != nil {
		t.Errorf("Finish() error = %v, want nil — a comment-only stream ending at a line boundary must finish cleanly (S-ASD-072)", err)
	}
}

// --- R-ASD-023: an end mid-frame is a typed truncation error, partial frame discarded ---

// TestEOF_CutMidDataValueReportsTruncationNoPartialFrame covers
// S-ASD-073: a transcript cut in the middle of a data value — no line
// terminator anywhere, so the line itself never resolved into the data
// accumulator at all — reports a truncation error from Finish, and Feed
// itself never yielded a frame for the unfinished content.
func TestEOF_CutMidDataValueReportsTruncationNoPartialFrame(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: this value is cut off with no terminat"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil — the cut is only detectable at Finish, not Feed (R-ASD-024)", err)
	}
	if len(frames) != 0 {
		t.Fatalf("Feed() yielded %d frames, want 0 — nothing can dispatch before the line even terminates", len(frames))
	}

	err = d.Finish()
	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated) (S-ASD-073)", err)
	}
	if errors.Is(err, openaicompat.ErrFrameTooLarge) {
		t.Error("Finish() error also matches ErrFrameTooLarge — the two identities must stay distinct (R-ASD-020)")
	}
}

// TestEOF_CompleteDataLinesNoTerminatingBlankLineReportsTruncation covers
// S-ASD-074: the final frame's data lines are each individually
// well-formed and fully resolved (LF-terminated, already joined into the
// data accumulator) — the ONLY thing missing is the terminating blank
// line that would dispatch them. Finish must still report truncation and
// that frame must never dispatch.
//
// Two data lines, not one, so the accumulated payload has real,
// checkable multi-line content (R-ASD-005's join rule) rather than a
// single trivial value.
func TestEOF_CompleteDataLinesNoTerminatingBlankLineReportsTruncation(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: first\ndata: second\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Fatalf("Feed() yielded %d frames, want 0 — the terminating blank line never arrived, so nothing may dispatch (S-ASD-074)", len(frames))
	}

	err = d.Finish()
	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated) — complete data lines with no terminating blank line must still report truncation (S-ASD-074)", err)
	}
}

// TestEOF_PendingEventTypeWithNoDataLineReportsTruncation covers a gap
// none of R-ASD-023's four numbered scenarios isolates on its own: a
// frame with only an event-type line — no data line at all, so the data
// accumulation stays empty — followed directly by end-of-input, with no
// blank line ever arriving. Every other truncation fixture in this file
// leaves d.data non-empty; this is the one case where ONLY the
// event-type accumulation is pending. Without a fixture like this,
// Finish's non-empty-eventType check (design.md's "or non-empty
// data/eventType accumulation") could be silently dropped and nothing in
// this file would notice.
func TestEOF_PendingEventTypeWithNoDataLineReportsTruncation(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: custom\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Fatalf("Feed() yielded %d frames, want 0 — no data line and no blank line means nothing can dispatch", len(frames))
	}

	err = d.Finish()
	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated) — a pending event-type accumulation with no data line is still a pending, undispatched frame", err)
	}
}

// TestEOF_TwoCompleteFramesThenPartialThirdYieldsTwoAndReportsTruncation
// covers S-ASD-075: two complete, well-formed frames dispatch normally
// from Feed, and a partial third — never reaching its own terminator —
// is absent from the yielded frames and reported as truncated by Finish.
//
// Both complete frames' exact content is asserted, not merely their
// count: a fixture asserting only "2 frames" could pass even if the
// decoder silently merged or corrupted content while still landing on
// the right count. The partial third's own distinct payload
// ("third-never-arrives") never appears anywhere in the assertions below
// specifically so a regression that somehow surfaced it would be caught.
func TestEOF_TwoCompleteFramesThenPartialThirdYieldsTwoAndReportsTruncation(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: one\n\ndata: two\n\ndata: third-never-arrives"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 2 {
		t.Fatalf("Feed() yielded %d frames, want exactly 2 (S-ASD-075)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("one")) {
		t.Errorf("frames[0].Data = %q, want %q", frames[0].Data, "one")
	}
	if !bytes.Equal(frames[1].Data, []byte("two")) {
		t.Errorf("frames[1].Data = %q, want %q", frames[1].Data, "two")
	}

	err = d.Finish()
	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated) — the partial third frame must be reported as truncated (S-ASD-075)", err)
	}
}

// --- R-ASD-023's category, and its distinction from cap-exceeded (task 6.3) ---

// TestEOF_TruncationErrorCategoryIsMalformedResponseDistinguishableFromCapExceeded
// covers S-ASD-076: the error Finish actually returns for a live
// truncation — not merely the bare ErrTruncated sentinel Slice 5 already
// checked in isolation — names the malformed-response category and stays
// distinguishable from cap-exceeded (R-ASD-020). This is the genuinely
// new fact this slice adds: that Finish's own return value carries the
// exact sentinel identity Category and errors.Is expect, not a
// re-wrapped or approximated one.
func TestEOF_TruncationErrorCategoryIsMalformedResponseDistinguishableFromCapExceeded(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	if _, err := d.Feed([]byte("data: incomplete")); err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	err := d.Finish()

	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated)", err)
	}
	if errors.Is(err, openaicompat.ErrFrameTooLarge) {
		t.Error("Finish() error also matches ErrFrameTooLarge — the two identities must stay distinct (R-ASD-020, S-ASD-076)")
	}
	if !errors.Is(err, ai.ErrMalformedResponse) {
		t.Error("Finish() error does not match ai.ErrMalformedResponse — the repo-native AI-19 matching idiom must hold (S-ASD-076)")
	}
	gotCategory, ok := openaicompat.Category(err)
	if !ok || gotCategory != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category(err) = (%v, %v), want (%v, true) (S-ASD-076)", gotCategory, ok, ai.FailureCategoryMalformedResponse)
	}
}

// --- R-ASD-024: end-of-input is a decoder signal, not a transport condition (task 6.4) ---

// TestEOF_FramesFedButNotFinishedExposeOnlyDispatchedFramesNoError covers
// S-ASD-077: a decoder that has been fed a complete frame plus a partial
// second, with Finish never called, exposes only the frame whose
// terminating blank line already arrived — and reports no error. Feed
// itself never infers or guesses that the trailing bytes might be
// truncated; that fact is only ever surfaced by an explicit Finish call
// (R-ASD-024), which this test deliberately never makes.
func TestEOF_FramesFedButNotFinishedExposeOnlyDispatchedFramesNoError(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: complete\n\ndata: still-arriving"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil — Feed alone must never report truncation, only Finish does (R-ASD-024, S-ASD-077)", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 — only the already-dispatched frame, nothing guessed about the partial second (S-ASD-077)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("complete")) {
		t.Errorf("frames[0].Data = %q, want %q", frames[0].Data, "complete")
	}
}

// --- Finish's CR resolution (design.md's "Finish semantics" decision; tasks.md's testing-strategy row for 27.6) ---

// TestEOF_TrailingBufferFinalCRResolvesToTruncationWhenContentPending
// pins design.md's own Testing Strategy row for AI-27.6: a transcript
// ending in "data: a\r" — an unresolved, buffer-final CR that Feed could
// not yet resolve as lone-CR-vs-CRLF-pending — must, once Finish resolves
// it (no further byte is ever coming), still report truncation: the CR
// terminates a REAL, non-empty field line ("data: a"), which Finish
// accumulates and then correctly reports as pending, never dispatched.
//
// This fixture alone cannot prove Finish genuinely RESOLVES the CR rather
// than merely noticing the buffer is non-empty — see the companion test
// immediately below, which is the one that actually distinguishes the
// two.
func TestEOF_TrailingBufferFinalCRResolvesToTruncationWhenContentPending(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: a\r"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Fatalf("Feed() yielded %d frames before the trailing CR resolved, want 0", len(frames))
	}

	err = d.Finish()
	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated) — resolving the trailing CR must not launder a still-pending field value into a clean finish", err)
	}
}

// TestEOF_TrailingBufferFinalCRWithNothingPendingFinishesClean is this
// slice's own falsifiability-critical pin, the direct counterpart to the
// test above: a stream that ends immediately after a fully-dispatched
// frame, with one extra, spurious lone CR trailing it and nothing else
// pending, must finish CLEAN — not truncated.
//
// This is deliberately the discriminating case design.md's "resolve a
// pending buffer-final CR... first" clause exists for. A naive Finish
// that skips CR resolution and just checks "is the retained buffer
// non-empty" would see this exact one leftover CR byte and WRONGLY report
// ErrTruncated — while the test above (a real pending field value behind
// the CR) would, by coincidence, report ErrTruncated correctly either
// way, CR-aware or not. Only this test's fixture is where a naive and a
// correct implementation diverge: this is what makes the CR-resolution
// step itself falsifiable rather than a dead branch that happens to agree
// with the naive shortcut on every other fixture this slice exercises
// (see this slice's evidence log for the scratch break-and-revert proof).
func TestEOF_TrailingBufferFinalCRWithNothingPendingFinishesClean(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: only\n\n\r"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (the frame dispatched by the transcript's own terminating blank line, before the trailing spurious CR)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("only")) {
		t.Errorf("frames[0].Data = %q, want %q", frames[0].Data, "only")
	}

	if err := d.Finish(); err != nil {
		t.Errorf("Finish() error = %v, want nil — a trailing CR terminating an already-empty, nothing-pending line must not be misreported as truncation", err)
	}
}

// --- Slice 2b's forward note: a pending partial BOM prefix at Finish is truncation too ---

// TestEOF_PendingPartialBOMPrefixAtFinishReportsTruncation pins the
// forward note slice 2b's apply run recorded for this slice (tasks.md,
// Phase 6): a stream consisting of only two of the three-byte
// byte-order mark's bytes — never confirmed as a full mark, never ruled
// out as one either — is exactly the "any retained line bytes"
// truncation condition (R-ASD-023) once Finish is called; no
// BOM-specific code exists in Finish, because the general retained-bytes
// check already covers it, and this fixture is what pins that inference
// by test rather than leaving it unverified.
func TestEOF_PendingPartialBOMPrefixAtFinishReportsTruncation(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte{0xEF, 0xBB}) // two of the mark's three bytes
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Fatalf("Feed() yielded %d frames, want 0", len(frames))
	}

	err = d.Finish()
	if !errors.Is(err, openaicompat.ErrTruncated) {
		t.Fatalf("Finish() error = %v, want errors.Is(err, ErrTruncated) — a pending partial BOM prefix is retained, unresolved bytes", err)
	}
}
