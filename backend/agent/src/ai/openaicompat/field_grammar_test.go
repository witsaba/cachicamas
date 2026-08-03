package openaicompat_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// This file covers Phase 2a (AI-27.2 items 1-3): field splitting
// (R-ASD-004, S-ASD-012..016), multi-line data joining (R-ASD-005,
// S-ASD-017..020), and the three line terminators with CRLF-as-one
// (R-ASD-006, S-ASD-021..024).
//
// Test names start with TestFieldGrammar so the slice's focused command
// (`go test -race -run TestFieldGrammar ./src/ai/openaicompat/...`) selects
// exactly this file's cases (see tasks.md's Suggested Work Units table).

// framesEqual reports whether two frame sequences carry the same event
// types and byte-identical data, in the same order.
func framesEqual(a, b []openaicompat.Frame) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Event != b[i].Event || !bytes.Equal(a[i].Data, b[i].Data) {
			return false
		}
	}
	return true
}

// --- R-ASD-004: field lines split at the first colon, one leading space stripped ---

// TestFieldGrammar_ColonInValueSplitsAtFirstColonOnly covers S-ASD-012: a
// data value containing colons of its own is split only at the field
// line's first colon; the remaining colons survive inside the value.
func TestFieldGrammar_ColonInValueSplitsAtFirstColonOnly(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: a:b:c\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("a:b:c")) {
		t.Errorf("Data = %q, want %q — split must occur at the first colon only (S-ASD-012)", frames[0].Data, "a:b:c")
	}
}

// TestFieldGrammar_TwoLeadingSpacesRetainsSecondSpace covers S-ASD-013: when
// the value is preceded by two spaces, exactly one is removed and the
// second survives in the payload.
func TestFieldGrammar_TwoLeadingSpacesRetainsSecondSpace(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data:  indented\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte(" indented")) {
		t.Errorf("Data = %q, want %q — exactly one leading space must be stripped (S-ASD-013)", frames[0].Data, " indented")
	}
}

// TestFieldGrammar_NoSpaceAfterColonValueVerbatim covers S-ASD-014: a value
// with no space after the colon is taken verbatim, nothing removed.
func TestFieldGrammar_NoSpaceAfterColonValueVerbatim(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data:no-space\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("no-space")) {
		t.Errorf("Data = %q, want %q — no leading space means nothing is removed (S-ASD-014)", frames[0].Data, "no-space")
	}
}

// TestFieldGrammar_ColonlessDataLineIsEmptyValueDataField covers S-ASD-015:
// the bare word "data" with no colon is treated as the data field with an
// empty value, per § 9.2's colonless-line rule.
//
// The fixture pairs the colonless "data" line with a second, ordinary data
// line rather than using it alone: with dispatch still unconditional in
// this slice (R-ASD-008's "empty accumulation dispatches nothing" is
// Phase 2b's rule, not yet implemented), a bare "data\n\n" transcript would
// yield an empty-Data frame whether or not the colonless line is correctly
// routed to the data accumulator — that assertion alone cannot distinguish
// "recognized, contributed an empty value" from "not recognized at all".
// Following it with "data: second" makes the join's leading separator
// observable only when the first line was genuinely recognized.
func TestFieldGrammar_ColonlessDataLineIsEmptyValueDataField(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data\ndata: second\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-015)", len(frames))
	}
	const want = "\nsecond"
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q — colonless \"data\" must join as the data field with an empty value, leaving its separator before \"second\" (S-ASD-015)", frames[0].Data, want)
	}
}

// TestFieldGrammar_ColonlessUnrecognizedNameIgnored covers S-ASD-016: a
// colonless line whose name is not a recognized field is ignored and
// disturbs no accumulation.
func TestFieldGrammar_ColonlessUnrecognizedNameIgnored(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: keep\nunrecognizedline\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("keep")) {
		t.Errorf("Data = %q, want %q — unrecognized colonless line must not disturb accumulation (S-ASD-016)", frames[0].Data, "keep")
	}
}

// --- R-ASD-005: multi-line data joins with LF, loses the trailing one at dispatch ---

// TestFieldGrammar_ThreeDataLinesJoinWithSingleLF covers S-ASD-017: three
// data lines join with exactly one line feed each and no trailing one.
func TestFieldGrammar_ThreeDataLinesJoinWithSingleLF(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: first\ndata: second\ndata: third\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	const want = "first\nsecond\nthird"
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q (S-ASD-017)", frames[0].Data, want)
	}
}

// TestFieldGrammar_SingleDataLineNoTrailingLF covers S-ASD-018: one data
// line's payload carries no trailing line feed.
func TestFieldGrammar_SingleDataLineNoTrailingLF(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: only\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if bytes.HasSuffix(frames[0].Data, []byte("\n")) {
		t.Errorf("Data = %q, must not carry a trailing line feed (S-ASD-018)", frames[0].Data)
	}
	if !bytes.Equal(frames[0].Data, []byte("only")) {
		t.Errorf("Data = %q, want %q (S-ASD-018)", frames[0].Data, "only")
	}
}

// TestFieldGrammar_EmptyLastDataLinePreservesInteriorLF covers S-ASD-019:
// when the last data line has an empty value, the separator that preceded
// it survives — only the final line feed is removed at dispatch.
func TestFieldGrammar_EmptyLastDataLinePreservesInteriorLF(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: a\ndata:\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	const want = "a\n"
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q — the separator before the empty line must survive, only the final one removed (S-ASD-019)", frames[0].Data, want)
	}
}

// TestFieldGrammar_ContentEmbeddedLFSurvivesDispatchTrim covers S-ASD-020:
// when the accumulated data itself ends in a line feed that is content
// (not the dispatch-time separator), only the dispatch-time separator is
// removed and the content line feed survives.
func TestFieldGrammar_ContentEmbeddedLFSurvivesDispatchTrim(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: a\ndata: b\ndata:\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	const want = "a\nb\n"
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q — only the dispatch-time separator is removed (S-ASD-020)", frames[0].Data, want)
	}
}

// --- R-ASD-006: three line terminators, CRLF is one terminator ---

// grammarTemplate is one logical two-frame transcript using "\n" as a
// placeholder terminator. Each terminator-style test below substitutes it
// for CRLF, LF or lone CR to render the same logical transcript three ways.
const grammarTemplate = "event: greet\ndata: hi\n\ndata: bye\n\n"

// wantGrammarFrames is grammarTemplate's canonical decoding, independent of
// which terminator style rendered it.
func wantGrammarFrames() []openaicompat.Frame {
	return []openaicompat.Frame{
		{Event: "greet", Data: []byte("hi")},
		{Event: "message", Data: []byte("bye")},
	}
}

// TestFieldGrammar_ThreeTerminatorStylesDecodeIdentically covers S-ASD-021:
// the same logical transcript rendered with CRLF, LF and lone-CR
// terminators decodes to identical frames in all three cases.
func TestFieldGrammar_ThreeTerminatorStylesDecodeIdentically(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		transcript string
	}{
		{"LF", grammarTemplate},
		{"CRLF", strings.ReplaceAll(grammarTemplate, "\n", "\r\n")},
		// The lone-CR rendering ends with an extra padding CR: Feed alone
		// never resolves a buffer-final CR (design.md — resolving it as a
		// lone terminator is Finish's job, AI-27.6/slice 6, not yet
		// implemented in this slice). The pad byte gives the final blank
		// line's CR a following byte so Feed can resolve it without
		// invoking Finish; the pad itself is never asserted on.
		{"lone CR", strings.ReplaceAll(grammarTemplate, "\n", "\r") + "\r"},
	}

	want := wantGrammarFrames()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := openaicompat.NewDecoder(0)
			frames, err := d.Feed([]byte(tt.transcript))
			if err != nil {
				t.Fatalf("Feed() error = %v, want nil", err)
			}
			if !framesEqual(frames, want) {
				t.Errorf("%s rendering: frames = %+v, want %+v (S-ASD-021)", tt.name, frames, want)
			}
		})
	}
}

// TestFieldGrammar_CRLFOnlyProducesNoEmptyFrame covers S-ASD-022: a
// transcript terminated exclusively by CRLF produces no empty frame and no
// empty data line anywhere in the output.
func TestFieldGrammar_CRLFOnlyProducesNoEmptyFrame(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: alpha\r\ndata: beta\r\n\r\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 — no phantom frame from CRLF (S-ASD-022)", len(frames))
	}
	const want = "alpha\nbeta"
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q — no empty data line must appear (S-ASD-022)", frames[0].Data, want)
	}
}

// TestFieldGrammar_MixedTerminatorsAccumulateUniformly covers S-ASD-023: a
// frame whose terminators mix CRLF, LF and lone CR accumulates its data
// lines exactly as if every terminator were uniform.
func TestFieldGrammar_MixedTerminatorsAccumulateUniformly(t *testing.T) {
	t.Parallel()

	// "one" ends in CRLF, "two" ends in a lone LF, "three" ends in a lone
	// CR. The trailing blank line is CRLF-terminated (both bytes present
	// in this single Feed call, so it self-resolves — no pad byte needed).
	const transcript = "data: one\r\ndata: two\ndata: three\r\r\n"

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(transcript))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	const want = "one\ntwo\nthree"
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q — mixed terminators must accumulate uniformly (S-ASD-023)", frames[0].Data, want)
	}
}

// TestFieldGrammar_CRLFSplitAcrossChunksReconstructsNoBlankLine covers
// S-ASD-024: a CRLF pair divided so the CR is the last byte of one input
// chunk and the LF is the first byte of the next reconstructs identically
// to the unsplit decoding, with no blank line injected at the split point.
func TestFieldGrammar_CRLFSplitAcrossChunksReconstructsNoBlankLine(t *testing.T) {
	t.Parallel()

	const unsplit = "data: x\r\n\r\n"
	// Hard-coded, not derived from a reference decode: comparing the split
	// decoding only to a same-code "reference" decode is vacuous if both
	// sides are broken identically (as they were pre-fix here — the CRLF
	// blank line was misparsed as a one-byte "\r" content line by both, so
	// both produced zero frames and a bare equality check passed for the
	// wrong reason). Anchoring to the literal expected value is what makes
	// this scenario actually falsifiable.
	wantFrames := []openaicompat.Frame{{Event: "message", Data: []byte("x")}}

	reference := openaicompat.NewDecoder(0)
	refFrames, err := reference.Feed([]byte(unsplit))
	if err != nil {
		t.Fatalf("reference Feed() error = %v, want nil", err)
	}
	if !framesEqual(refFrames, wantFrames) {
		t.Fatalf("unsplit reference decoding = %+v, want %+v — the fixture itself must decode correctly before the split comparison below means anything", refFrames, wantFrames)
	}

	d := openaicompat.NewDecoder(0)
	first, err := d.Feed([]byte("data: x\r"))
	if err != nil {
		t.Fatalf("first Feed() error = %v, want nil", err)
	}
	if len(first) != 0 {
		t.Fatalf("first Feed() yielded %d frames before the split resolved, want 0 — the buffer-final CR must not be guessed at (S-ASD-024)", len(first))
	}

	second, err := d.Feed([]byte("\n\r\n"))
	if err != nil {
		t.Fatalf("second Feed() error = %v, want nil", err)
	}

	got := append(first, second...)
	if !framesEqual(got, wantFrames) {
		t.Errorf("split decoding = %+v, want %+v (identical to unsplit decoding, S-ASD-024)", got, wantFrames)
	}
}
