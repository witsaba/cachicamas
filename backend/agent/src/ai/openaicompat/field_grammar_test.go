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
// (R-ASD-006, S-ASD-021..024) — and Phase 2b (AI-27.2 items 4-7): a
// leading byte-order mark stripped once (R-ASD-007, S-ASD-025..028), an
// empty data accumulation dispatching nothing (R-ASD-008, S-ASD-029..031),
// state reset to the default event type including an empty-valued
// "event:" line (R-ASD-009, S-ASD-032..034, S-ASD-084), and the id/retry
// ignore-disposition equivalence (R-ASD-010, S-ASD-035..038).
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

// --- R-ASD-007: one leading byte-order mark is stripped once for the whole stream ---

// bomBytes is the three-byte UTF-8 byte-order mark used by the tests below.
const bomBytes = "\xEF\xBB\xBF"

// TestFieldGrammar_SingleLeadingBOMStrippedOnce covers S-ASD-025: a
// transcript prefixed by exactly one byte-order mark decodes identically to
// the same transcript with no mark.
//
// The no-mark decoding is pinned against a hard-coded expected frame first,
// not compared only to the BOM-prefixed decoding — the same self-reference
// trap S-ASD-024 (slice 2a) was caught and fixed for: a bare comparison
// between two decodes run through the same code passes vacuously if both
// are broken identically.
func TestFieldGrammar_SingleLeadingBOMStrippedOnce(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "greet", Data: []byte("hi")}}

	plain := openaicompat.NewDecoder(0)
	plainFrames, err := plain.Feed([]byte("event: greet\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("plain Feed() error = %v, want nil", err)
	}
	if !framesEqual(plainFrames, want) {
		t.Fatalf("plain decoding = %+v, want %+v — pin the reference before comparing the BOM-prefixed case to it", plainFrames, want)
	}

	withBOM := openaicompat.NewDecoder(0)
	bomFrames, err := withBOM.Feed([]byte(bomBytes + "event: greet\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("BOM-prefixed Feed() error = %v, want nil", err)
	}
	if !framesEqual(bomFrames, want) {
		t.Errorf("BOM-prefixed decoding = %+v, want %+v — the leading mark must be stripped, not left as content (S-ASD-025)", bomFrames, want)
	}
}

// TestFieldGrammar_TwoLeadingBOMsStripsOnlyOneSecondSurvivesAsContent covers
// S-ASD-026: exactly one leading byte-order mark is stripped; a second,
// immediately following mark is ordinary content and survives as a prefix of
// the first field name, which then fails to match "event" — so the frame
// carries the default type.
func TestFieldGrammar_TwoLeadingBOMsStripsOnlyOneSecondSurvivesAsContent(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(bomBytes + bomBytes + "event: greet\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-026)", len(frames))
	}
	if frames[0].Event != "message" {
		t.Errorf("Event = %q, want %q — the second mark must survive as content, corrupting the field name so it does not match \"event\" (S-ASD-026)", frames[0].Event, "message")
	}
	if !bytes.Equal(frames[0].Data, []byte("hi")) {
		t.Errorf("Data = %q, want %q (S-ASD-026)", frames[0].Data, "hi")
	}
}

// TestFieldGrammar_BOMInsideDataValueSurvivesVerbatim covers S-ASD-027: a
// byte-order mark occurring inside a data value, not at stream start, is
// ordinary content and survives verbatim in the yielded payload.
func TestFieldGrammar_BOMInsideDataValueSurvivesVerbatim(t *testing.T) {
	t.Parallel()

	const payload = "before" + bomBytes + "after"
	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: " + payload + "\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte(payload)) {
		t.Errorf("Data = %q, want %q — a mid-value mark must survive verbatim (S-ASD-027)", frames[0].Data, payload)
	}
}

// TestFieldGrammar_LeadingBOMSplitAcrossChunksStillStrippedOnce covers
// S-ASD-028: a leading byte-order mark whose three bytes are divided across
// input chunks — at offset 1 and at offset 2, inside the mark itself — is
// still stripped exactly once.
func TestFieldGrammar_LeadingBOMSplitAcrossChunksStillStrippedOnce(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("hi")}}

	tests := []struct {
		name   string
		offset int
	}{
		{"split at offset 1", 1},
		{"split at offset 2", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := openaicompat.NewDecoder(0)
			first, err := d.Feed([]byte(bomBytes[:tt.offset]))
			if err != nil {
				t.Fatalf("first Feed() error = %v, want nil", err)
			}
			if len(first) != 0 {
				t.Fatalf("first Feed() yielded %d frames from a partial BOM prefix, want 0", len(first))
			}

			second, err := d.Feed([]byte(bomBytes[tt.offset:] + "data: hi\n\n"))
			if err != nil {
				t.Fatalf("second Feed() error = %v, want nil", err)
			}

			got := append(first, second...)
			if !framesEqual(got, want) {
				t.Errorf("split decoding = %+v, want %+v — the mark must still be stripped exactly once (S-ASD-028)", got, want)
			}
		})
	}
}

// --- R-ASD-008: an empty data accumulation dispatches nothing ---

// TestFieldGrammar_TwoBlankLinesOnlyYieldZeroFrames covers S-ASD-029: a
// transcript of two consecutive blank lines and nothing else yields zero
// frames.
func TestFieldGrammar_TwoBlankLinesOnlyYieldZeroFrames(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Errorf("Feed() yielded %d frames, want 0 — an empty data accumulation must dispatch nothing (S-ASD-029)", len(frames))
	}
}

// TestFieldGrammar_EventTypeLineWithNoDataYieldsZeroFrames covers S-ASD-030:
// a frame consisting of an event-type line followed by a blank line, with no
// data line at all, yields zero frames.
func TestFieldGrammar_EventTypeLineWithNoDataYieldsZeroFrames(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: greet\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Errorf("Feed() yielded %d frames, want 0 — an event-type line alone has an empty data accumulation and must dispatch nothing (S-ASD-030)", len(frames))
	}
}

// TestFieldGrammar_WellFormedFrameSurroundedByBlankLinesYieldsExactlyOne
// covers S-ASD-031: a well-formed frame preceded and followed by blank
// lines yields exactly one frame.
func TestFieldGrammar_WellFormedFrameSurroundedByBlankLinesYieldsExactlyOne(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("\n\ndata: hi\n\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-031)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte("hi")) {
		t.Errorf("Data = %q, want %q (S-ASD-031)", frames[0].Data, "hi")
	}
}

// --- R-ASD-009: accumulation state resets after each dispatch ---

// TestFieldGrammar_SecondTypelessFrameDefaultsNeverInheritsFirstType covers
// S-ASD-032: a frame with an explicit event-type line followed by a second
// frame with none — the second frame's type is the default type, not the
// first frame's.
func TestFieldGrammar_SecondTypelessFrameDefaultsNeverInheritsFirstType(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: greet\ndata: hi\n\ndata: bye\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 2 {
		t.Fatalf("Feed() yielded %d frames, want exactly 2", len(frames))
	}
	if frames[0].Event != "greet" {
		t.Errorf("frames[0].Event = %q, want %q", frames[0].Event, "greet")
	}
	if frames[1].Event != "message" {
		t.Errorf("frames[1].Event = %q, want %q — the second frame must default, never inherit the first frame's type (S-ASD-032)", frames[1].Event, "message")
	}
}

// TestFieldGrammar_SecondFrameDataCarriesNoneOfFirstFrameData covers
// S-ASD-033: two consecutive frames each with one data line — the second
// frame's payload contains none of the first frame's data.
func TestFieldGrammar_SecondFrameDataCarriesNoneOfFirstFrameData(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: first-frame-data\n\ndata: second\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 2 {
		t.Fatalf("Feed() yielded %d frames, want exactly 2", len(frames))
	}
	if !bytes.Equal(frames[1].Data, []byte("second")) {
		t.Errorf("frames[1].Data = %q, want %q — must carry none of the first frame's data (S-ASD-033)", frames[1].Data, "second")
	}
	if bytes.Contains(frames[1].Data, []byte("first-frame-data")) {
		t.Errorf("frames[1].Data = %q leaked content from the first frame (S-ASD-033)", frames[1].Data)
	}
}

// TestFieldGrammar_SuppressedTypeOnlyFrameLeavesNoTypeResidue covers
// S-ASD-034: an event-type-only frame that dispatches nothing (its data
// accumulation is empty, R-ASD-008), followed by a data-only frame — the
// single yielded frame carries the default type, proving the suppressed
// frame's type left no residue in the reset state.
func TestFieldGrammar_SuppressedTypeOnlyFrameLeavesNoTypeResidue(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: greet\n\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 — the type-only frame must dispatch nothing (S-ASD-034)", len(frames))
	}
	if frames[0].Event != "message" {
		t.Errorf("Event = %q, want %q — the suppressed frame's type must leave no residue (S-ASD-034)", frames[0].Event, "message")
	}
	if !bytes.Equal(frames[0].Data, []byte("hi")) {
		t.Errorf("Data = %q, want %q (S-ASD-034)", frames[0].Data, "hi")
	}
}

// TestFieldGrammar_EmptyValuedEventLineDispatchesDefaultType covers
// S-ASD-084 (R-ASD-009): an event-type line with an EMPTY value sets the
// accumulation to the empty string, which dispatch treats identically to no
// event-type line at all — the frame carries the default type, never the
// empty string.
//
// The fixture first assigns a non-empty event type ("event: custom"), then
// immediately overwrites it with an empty-valued "event:" line before the
// data line. A lone empty "event:" line would not distinguish "genuinely
// recognized and overwrote the buffer to empty" from a defective decoder
// that special-cases an empty value by leaving the buffer's prior value in
// place — both would default identically from an untouched-since-reset
// buffer. Preceding it with a non-empty assignment makes that failure mode
// observable: a defective decoder would report "custom", not "message".
func TestFieldGrammar_EmptyValuedEventLineDispatchesDefaultType(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: custom\nevent:\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-084)", len(frames))
	}
	if frames[0].Event != "message" {
		t.Errorf("Event = %q, want %q — an empty-valued event: line must dispatch with the default type, never the empty string or a stale prior value (S-ASD-084)", frames[0].Event, "message")
	}
}

// --- R-ASD-010: the identifier and retry fields are tracked by nothing ---

// TestFieldGrammar_IdentifierLineIgnoredOrdinaryValue covers S-ASD-035: a
// frame containing an identifier line with an ordinary value decodes
// identically to the same frame without the identifier line. Both sides are
// pinned against a hard-coded expectation, not only against each other.
func TestFieldGrammar_IdentifierLineIgnoredOrdinaryValue(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("hi")}}

	without := openaicompat.NewDecoder(0)
	withoutFrames, err := without.Feed([]byte("data: hi\n\n"))
	if err != nil {
		t.Fatalf("without Feed() error = %v, want nil", err)
	}
	if !framesEqual(withoutFrames, want) {
		t.Fatalf("baseline decoding (no id line) = %+v, want %+v", withoutFrames, want)
	}

	with := openaicompat.NewDecoder(0)
	withFrames, err := with.Feed([]byte("id: abc123\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("with Feed() error = %v, want nil", err)
	}
	if !framesEqual(withFrames, want) {
		t.Errorf("frames with id line = %+v, want %+v — an ordinary identifier line must be ignored (S-ASD-035)", withFrames, want)
	}
}

// TestFieldGrammar_IdentifierNULEquivalentToNonNUL covers S-ASD-036: two
// otherwise identical transcripts differing only in that one's identifier
// value contains a NUL character and the other's does not decode to
// identical frame sequences — the NUL case is not special-cased into any
// observable difference. This is deliberately an equivalence between two
// decodes, not an assertion of "unchanged state": the decoder tracks no
// identifier state to observe (R-ASD-010). Both sides are additionally
// pinned against a hard-coded expectation.
func TestFieldGrammar_IdentifierNULEquivalentToNonNUL(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("hi")}}

	withoutNUL := openaicompat.NewDecoder(0)
	framesWithoutNUL, err := withoutNUL.Feed([]byte("id: abc123\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("without-NUL Feed() error = %v, want nil", err)
	}
	if !framesEqual(framesWithoutNUL, want) {
		t.Fatalf("baseline (non-NUL id) decoding = %+v, want %+v", framesWithoutNUL, want)
	}

	withNUL := openaicompat.NewDecoder(0)
	framesWithNUL, err := withNUL.Feed([]byte("id: abc\x00123\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("with-NUL Feed() error = %v, want nil", err)
	}
	if !framesEqual(framesWithNUL, want) {
		t.Errorf("frames with NUL-bearing id = %+v, want %+v (identical to non-NUL id's frames, S-ASD-036)", framesWithNUL, want)
	}
}

// TestFieldGrammar_RetryLineIgnoredNonDigitValue covers S-ASD-037: a retry
// line whose value contains a non-digit character decodes identically to
// the same transcript without the retry line.
func TestFieldGrammar_RetryLineIgnoredNonDigitValue(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("hi")}}

	without := openaicompat.NewDecoder(0)
	withoutFrames, err := without.Feed([]byte("data: hi\n\n"))
	if err != nil {
		t.Fatalf("without Feed() error = %v, want nil", err)
	}
	if !framesEqual(withoutFrames, want) {
		t.Fatalf("baseline decoding (no retry line) = %+v, want %+v", withoutFrames, want)
	}

	with := openaicompat.NewDecoder(0)
	withFrames, err := with.Feed([]byte("retry: 3s\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("with Feed() error = %v, want nil", err)
	}
	if !framesEqual(withFrames, want) {
		t.Errorf("frames with non-digit retry line = %+v, want %+v — a non-digit retry value must be ignored (S-ASD-037)", withFrames, want)
	}
}

// TestFieldGrammar_RetryLineIgnoredAllDigitsValue covers S-ASD-038: a retry
// line whose value is all ASCII digits decodes identically to the same
// transcript without the retry line — the retry value never becomes frame
// content and never reaches a frame field.
func TestFieldGrammar_RetryLineIgnoredAllDigitsValue(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("hi")}}

	without := openaicompat.NewDecoder(0)
	withoutFrames, err := without.Feed([]byte("data: hi\n\n"))
	if err != nil {
		t.Fatalf("without Feed() error = %v, want nil", err)
	}
	if !framesEqual(withoutFrames, want) {
		t.Fatalf("baseline decoding (no retry line) = %+v, want %+v", withoutFrames, want)
	}

	with := openaicompat.NewDecoder(0)
	withFrames, err := with.Feed([]byte("retry: 3000\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("with Feed() error = %v, want nil", err)
	}
	if !framesEqual(withFrames, want) {
		t.Errorf("frames with all-digit retry line = %+v, want %+v — an all-digit retry value must still be ignored, never surfaced on a frame (S-ASD-038)", withFrames, want)
	}
}
