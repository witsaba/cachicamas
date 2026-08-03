package openaicompat_test

import "github.com/cachicamas/backend/agent/src/ai/openaicompat"

// This file declares the offset sweep's registry of golden transcripts and
// the structural size bound (R-ASD-012...014) that keeps the multi-megabyte
// fixture (Slice 5's own scope) out of this package's O(N^2) exhaustive
// sweep. The bound is declared here, beside the registry, so the assertion
// and the requirement read the same number (design.md); the guard itself —
// the assertion that every registered transcript actually satisfies the
// bound — executes in offset_sweep_test.go, as that harness's first act.

// maxSweepFixtureBytes is the sweep fixture bound (S-ASD-049): inclusive,
// so a transcript of exactly this size is admissible and one byte more is
// not. R-ASD-014 requires the multi-megabyte fixture's exclusion be
// enforced structurally, by a guard over this registry, never by
// convention or reviewer diligence alone.
const maxSweepFixtureBytes = 1024

// sweepTranscript is one golden transcript admitted to the offset sweep.
//
// want, when non-nil, anchors this transcript's canonical decoding to a
// literal expected frame sequence rather than only to a same-code
// reference decode. At least one registry entry MUST set want: a sweep
// that only ever compares a split decode to a canonical decode produced by
// the same decoder is the exact vacuous-pass shape this milestone has
// already been caught by once — Slice 2a's first draft of
// TestFieldGrammar_CRLFSplitAcrossChunksReconstructsNoBlankLine (S-ASD-024)
// compared a split decode only to a "reference" decode running the same
// code, and passed vacuously while both sides were identically broken (see
// field_grammar_test.go's doc comment on that test). want breaks that
// shape by pinning the correct answer independently of the decoder under
// test, for every terminator-style rendering below.
type sweepTranscript struct {
	name       string
	transcript []byte
	want       []openaicompat.Frame
}

// sweepTemplateFrames is the canonical decoding shared by every
// terminator-style rendering of the same logical two-frame transcript
// below ("lf-only", "crlf", "lone-cr", "mixed-terminators") — one
// hard-coded literal anchoring all four entries, mirroring
// field_grammar_test.go's wantGrammarFrames pattern for the same reason: a
// fresh slice per call, so no test can observably mutate a shared literal
// out from under another.
func sweepTemplateFrames() []openaicompat.Frame {
	return []openaicompat.Frame{
		{Event: "greet", Data: []byte("hello")},
		{Event: "message", Data: []byte("second frame")},
	}
}

// sweepTranscripts is the only input to the offset sweep (design.md). It
// carries one representative transcript per terminator style plus the BOM
// and multi-line-data cases the sweep must prove re-entrant, per
// tasks.md 3.1: LF-only, CRLF, lone-CR, mixed, BOM-prefixed,
// multi-line-data. Every entry sets want — see sweepTranscript's doc
// comment for why that matters beyond the one entry the spec requires as a
// minimum.
var sweepTranscripts = []sweepTranscript{
	{
		name:       "lf-only",
		transcript: []byte("event: greet\ndata: hello\n\ndata: second frame\n\n"),
		want:       sweepTemplateFrames(),
	},
	{
		name:       "crlf",
		transcript: []byte("event: greet\r\ndata: hello\r\n\r\ndata: second frame\r\n\r\n"),
		want:       sweepTemplateFrames(),
	},
	{
		// A trailing pad byte (a third consecutive CR) gives the
		// frame-terminating blank line's own CR a following byte, so a
		// single Feed call can resolve it as a lone (one-byte) terminator
		// — Feed alone never resolves a buffer-final CR (design.md; doing
		// so is Finish's job, AI-27.6/slice 6, not yet implemented). The
		// pad byte is never asserted on and is simply left retained,
		// unresolved, in the decoder's buffer after this Feed call — the
		// same technique field_grammar_test.go's grammarTemplate rendering
		// uses for its own "lone CR" case.
		name:       "lone-cr",
		transcript: []byte("event: greet\rdata: hello\r\rdata: second frame\r\r\r"),
		want:       sweepTemplateFrames(),
	},
	{
		// Mixes all three terminator styles within one transcript: CRLF
		// after "event: greet", a lone LF after "data: hello" and again
		// for the blank line dispatching frame 1, a lone CR after "data:
		// second frame", then CRLF for the final blank line — which, like
		// "crlf" and "lf-only" above, means this rendering resolves
		// cleanly within a single Feed call with no pad byte needed.
		name:       "mixed-terminators",
		transcript: []byte("event: greet\r\ndata: hello\n\ndata: second frame\r\r\n"),
		want:       sweepTemplateFrames(),
	},
	{
		// bomBytes is field_grammar_test.go's own package-level constant
		// (same test package); reused rather than redeclared.
		name:       "bom-prefixed",
		transcript: []byte(bomBytes + "event: greet\ndata: hello\n\n"),
		want: []openaicompat.Frame{
			{Event: "greet", Data: []byte("hello")},
		},
	},
	{
		// café/heart/emoji is decoder_test.go's own multi-byte-width
		// payload (TestDecoder_MultiByteDataSurvivesByteExact), reused here
		// so the offset sweep's exhaustive per-offset loop necessarily
		// lands inside a multi-byte rune at some offset, exercising
		// S-ASD-043 without a hand-picked split point.
		name:       "multi-line-data",
		transcript: []byte("data: line one\ndata: line two\ndata: line three\n\ndata: café ❤️ \U0001F600\n\n"),
		want: []openaicompat.Frame{
			{Event: "message", Data: []byte("line one\nline two\nline three")},
			{Event: "message", Data: []byte("café ❤️ \U0001F600")},
		},
	},
}
