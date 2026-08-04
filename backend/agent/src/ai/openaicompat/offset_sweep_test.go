package openaicompat_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// This file covers Phase 3 (AI-27.3, chunk-boundary re-entrancy):
// R-ASD-012 (S-ASD-041..044), R-ASD-013 (S-ASD-046..048), and R-ASD-014
// (S-ASD-049, the structural size guard; S-ASD-051 is a reviewer-trace
// obligation discharged in the evidence log, not by a test here).
//
// Test names start with TestOffsetSweep so the slice's focused command
// (`go test -race -run TestOffsetSweep ./src/ai/openaicompat/...`) selects
// exactly this file's cases — the same naming convention
// field_grammar_test.go established for TestFieldGrammar.

// TestOffsetSweep_FixturesWithinSweepBound is the harness's first act
// (design.md's Offset-Sweep Harness section): every registered transcript
// MUST be at most maxSweepFixtureBytes, inclusive (S-ASD-049) — this is
// what keeps the multi-megabyte fixture (Slice 5's own scope, R-ASD-014)
// structurally out of this O(N^2) sweep, rather than relying on
// convention or reviewer diligence.
//
// This proves S-ASD-049's POSITIVE clause only (every real fixture is
// within bound). checkSweepFixtureBound below factors the identical check
// into a pure function callable over any candidate set, so the scenario's
// NEGATIVE clause — "a deliberately overweight fixture trips the guard" —
// can be proven permanently too, without staging a violating fixture in
// this real registry. See TestOffsetSweep_SyntheticOverweightFixture_Fails.
func TestOffsetSweep_FixturesWithinSweepBound(t *testing.T) {
	t.Parallel()

	for _, v := range checkSweepFixtureBound(sweepTranscripts) {
		t.Error(v)
	}
}

// TestOffsetSweep_SyntheticOverweightFixture_Fails is S-ASD-049's second
// clause — "a deliberately overweight fixture trips the guard" — made
// PERMANENTLY runnable (post-verify durability fix, W1 in
// verify-report.md), replacing a proof that previously existed only once,
// transiently: Slice 3 staged a deliberately overweight fixture directly in
// sweepTranscripts, ran the guard, recorded it failing loudly, then removed
// the fixture (see this file's own history and tasks.md's Slice-3 evidence
// log for the exact recorded failure line). That proof was real but left
// nothing in the shipped suite to re-establish it — a future refactor that
// silently defanged the size comparison would go unnoticed.
//
// checkSweepFixtureBound is exercised directly against a synthetic
// candidate constructed in-test; sweepTranscripts (the real registry) is
// never touched. This is exactly src/ai/sequence_guard_test.go's
// TestSequenceGuard_*_Fails idiom (e.g.
// TestSequenceGuard_ScratchPackageLevelInteger_Fails): construct a
// deliberately violating input in-test, assert the guard rejects it and
// names the offender, leave the real subject clean.
//
// Falsifiable by construction: if checkSweepFixtureBound's size comparison
// were removed or inverted, this test fails (see this task's own scratch
// break-and-revert record in tasks.md).
func TestOffsetSweep_SyntheticOverweightFixture_Fails(t *testing.T) {
	t.Parallel()

	const oversizedName = "SYNTHETIC-OVERWEIGHT-FOR-GUARD-PROOF"
	oversized := sweepTranscript{
		name:       oversizedName,
		transcript: bytes.Repeat([]byte("x"), maxSweepFixtureBytes+1), // 1025 bytes: one over the inclusive bound
	}

	violations := checkSweepFixtureBound([]sweepTranscript{oversized})
	if len(violations) != 1 {
		t.Fatalf("checkSweepFixtureBound(1025-byte synthetic fixture) = %d violations, want exactly 1", len(violations))
	}
	if violations[0].name != oversizedName {
		t.Errorf("violation names transcript %q, want %q — the guard must name the offending transcript", violations[0].name, oversizedName)
	}
	if violations[0].size != maxSweepFixtureBytes+1 {
		t.Errorf("violation reports size %d, want %d — the guard must name the offending transcript's exact size", violations[0].size, maxSweepFixtureBytes+1)
	}

	// The bound is inclusive (S-ASD-049, design.md Rev 3): exactly
	// maxSweepFixtureBytes MUST NOT trip the guard — never silently
	// flipped to a strict `<`.
	atBound := sweepTranscript{name: "AT-BOUND", transcript: bytes.Repeat([]byte("x"), maxSweepFixtureBytes)}
	if got := checkSweepFixtureBound([]sweepTranscript{atBound}); len(got) != 0 {
		t.Errorf("checkSweepFixtureBound(exactly-at-bound fixture) = %v, want no violations — the bound is inclusive, exactly %d bytes is admissible", got, maxSweepFixtureBytes)
	}
}

// TestOffsetSweep_CanonicalDecodeMatchesHardCodedFrames anchors every
// registry entry that sets want against a literal expected frame sequence,
// not only against itself. Comparing every split decode only to a
// same-code canonical decode is vacuous if the canonical decode is itself
// wrong — the exact trap Slice 2a's first S-ASD-024 draft fell into (see
// field_grammar_test.go's doc comment on
// TestFieldGrammar_CRLFSplitAcrossChunksReconstructsNoBlankLine). Without
// this test, TestOffsetSweep_EveryOffsetMatchesCanonicalDecode below could
// pass in full while every canonical decode was uniformly broken.
func TestOffsetSweep_CanonicalDecodeMatchesHardCodedFrames(t *testing.T) {
	t.Parallel()

	anchored := 0
	for _, tr := range sweepTranscripts {
		if tr.want == nil {
			continue
		}
		anchored++
		got := decodeCanonical(t, tr.transcript)
		if !framesEqual(got, tr.want) {
			t.Errorf("transcript %q canonical decode = %+v, want %+v (hard-coded anchor)", tr.name, got, tr.want)
		}
	}
	if anchored == 0 {
		t.Fatal("no registry entry sets want — the sweep would compare every split decode only against a same-code canonical decode, which cannot detect a uniformly wrong decoder")
	}
}

// TestOffsetSweep_EveryOffsetMatchesCanonicalDecode covers S-ASD-041 (every
// offset, mechanically, not sampled), S-ASD-042 (a split landing inside a
// field name — "event" in every non-BOM fixture below), S-ASD-043 (a split
// landing inside a multi-byte data character — the "multi-line-data"
// fixture's café/heart/emoji content), and S-ASD-044 (the "bom-prefixed"
// fixture split at offsets 1 and 2, inside the three-byte mark itself).
//
// All four are discharged by the same exhaustive, mechanical per-offset
// proof (R-ASD-012), not by separate hand-picked cases: a UTF-8
// continuation byte is never 0x0A or 0x0D, so no offset landing inside a
// multi-byte rune can ever be mistaken for a line terminator, and a split
// inside "event" or inside the BOM is simply one of the
// len(transcript)+1 offsets this loop already covers.
func TestOffsetSweep_EveryOffsetMatchesCanonicalDecode(t *testing.T) {
	for _, tr := range sweepTranscripts {
		t.Run(tr.name, func(t *testing.T) {
			t.Parallel()

			canonical := decodeCanonical(t, tr.transcript)

			for offset := 0; offset <= len(tr.transcript); offset++ {
				got := decodeSplitAtOffset(t, tr.transcript, offset)
				if msg, mismatched := sweepMismatch(tr.name, offset, got, canonical); mismatched {
					t.Error(msg)
				}
			}

			gotByteAtATime := decodeByteAtATime(t, tr.transcript)
			if !framesEqual(gotByteAtATime, canonical) {
				t.Errorf("transcript %q byte-at-a-time replay: frames = %+v, want %+v (canonical decode)", tr.name, gotByteAtATime, canonical)
			}
		})
	}
}

// --- R-ASD-013: a split between CR and LF must not inject a phantom blank line ---

// TestOffsetSweep_CRLFSplitBetweenCRAndLFContentLine covers S-ASD-046: a
// transcript containing a CRLF terminator, split at the exact offset
// between that CR and its LF, decodes identically to the unsplit
// reference with no extra frame boundary — the single offset an
// example-based test essentially never selects, so it is pinned here in
// addition to being covered by the exhaustive sweep above.
//
// Anchored against a hard-coded frame, not only against a same-code
// reference decode (the same self-reference trap noted above).
//
// The fixture deliberately carries a SECOND data line after the split
// point, not just the terminating blank line: a first draft of this test
// used only "data: x\r" | "\n\n" and passed vacuously even when
// resolveCR's deferred-CR wait was deliberately broken (guessing a lone CR
// instead of waiting for one more byte) — with nothing after the phantom
// blank line's premature dispatch, the wrongly-early reset had nothing
// observable left to corrupt, and the net result coincidentally matched
// the correct one. Only once a second data line follows does the
// premature reset manifest as an extra, wrongly-split frame instead of one
// joined frame — the fixture below is what actually falsifies the break
// (see this slice's evidence log for the recorded break-and-revert cycle).
func TestOffsetSweep_CRLFSplitBetweenCRAndLFContentLine(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("first\nsecond")}}

	d := openaicompat.NewDecoder(0)
	first, err := d.Feed([]byte("data: first\r"))
	if err != nil {
		t.Fatalf("first Feed() error = %v, want nil", err)
	}
	if len(first) != 0 {
		t.Fatalf("first Feed() yielded %d frames before the CR/LF split resolved, want 0 (S-ASD-046)", len(first))
	}

	second, err := d.Feed([]byte("\ndata: second\n\n"))
	if err != nil {
		t.Fatalf("second Feed() error = %v, want nil", err)
	}

	got := append(first, second...)
	if !framesEqual(got, want) {
		t.Errorf("split decoding = %+v, want %+v — no extra frame boundary must appear at the CR/LF split (S-ASD-046)", got, want)
	}
}

// TestOffsetSweep_CRLFSplitBetweenCRAndLFAtFrameTerminatingBlankLine covers
// S-ASD-047: the same CR/LF split, but landing in the CRLF that
// terminates the frame (the blank line) rather than a content line's own
// terminator — exactly one dispatch must occur at that point, never two.
// A decoder that resolved the CR alone as its own terminator before the LF
// arrived would dispatch once too early on the (still non-empty) data
// accumulation, then dispatch again — or inject a phantom empty frame —
// when the LF arrived and was read as a second, independent blank line.
func TestOffsetSweep_CRLFSplitBetweenCRAndLFAtFrameTerminatingBlankLine(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("x")}}

	d := openaicompat.NewDecoder(0)
	// "data: x\r\n" (one complete, CRLF-terminated data line) + "\r" (the
	// blank line's own CR, its LF withheld until the next Feed call).
	first, err := d.Feed([]byte("data: x\r\n\r"))
	if err != nil {
		t.Fatalf("first Feed() error = %v, want nil", err)
	}
	if len(first) != 0 {
		t.Fatalf("first Feed() yielded %d frames before the frame-terminating CR/LF resolved, want 0 (S-ASD-047)", len(first))
	}

	second, err := d.Feed([]byte("\n"))
	if err != nil {
		t.Fatalf("second Feed() error = %v, want nil", err)
	}
	if len(second) != 1 {
		t.Fatalf("second Feed() yielded %d frames, want exactly 1 — exactly one dispatch must occur at the split, not two (S-ASD-047)", len(second))
	}

	got := append(first, second...)
	if !framesEqual(got, want) {
		t.Errorf("split decoding = %+v, want %+v (S-ASD-047)", got, want)
	}
}

// TestOffsetSweep_LoneCRChunkFinalNoFollowingLF covers S-ASD-048: a lone
// carriage return that is the final byte of an input chunk, and is NOT
// followed by a line feed in the next chunk, terminates its line exactly
// once — joining with the next line's own content rather than producing a
// duplicate dispatch or a lost terminator.
func TestOffsetSweep_LoneCRChunkFinalNoFollowingLF(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("a\nb")}}

	d := openaicompat.NewDecoder(0)
	first, err := d.Feed([]byte("data: a\r"))
	if err != nil {
		t.Fatalf("first Feed() error = %v, want nil", err)
	}
	if len(first) != 0 {
		t.Fatalf("first Feed() yielded %d frames before the lone CR resolved, want 0", len(first))
	}

	// The next chunk's first byte is 'd', not '\n' — the CR above must
	// resolve as a lone (one-byte) terminator, not stay pending forever
	// and not be misread as half of a CRLF.
	second, err := d.Feed([]byte("data: b\n\n"))
	if err != nil {
		t.Fatalf("second Feed() error = %v, want nil", err)
	}

	got := append(first, second...)
	if !framesEqual(got, want) {
		t.Errorf("split decoding = %+v, want %+v — the lone CR must terminate its line exactly once (S-ASD-048)", got, want)
	}
}

// --- S-ASD-045 upgrade: the mismatch path names the offending byte offset ---

// TestOffsetSweep_MismatchMessageNamesOffendingByteOffset covers the
// S-ASD-045 upgrade path recorded in spec.md: because sweepMismatch below
// is a separately callable pure function, this scenario is now runnable
// and failable (a [test], not [inspection]) — it exercises the mismatch
// path's message construction directly, without needing a real decoder
// defect to reach it. A bare boolean mismatch could not locate the defect
// on its own; this pins that the message names both the transcript and the
// exact byte offset.
func TestOffsetSweep_MismatchMessageNamesOffendingByteOffset(t *testing.T) {
	t.Parallel()

	got := []openaicompat.Frame{{Event: "message", Data: []byte("a")}}
	want := []openaicompat.Frame{{Event: "message", Data: []byte("b")}}

	msg, mismatched := sweepMismatch("some-transcript", 42, got, want)
	if !mismatched {
		t.Fatalf("sweepMismatch(...) reported mismatched = false, want true for differing frame sequences")
	}
	if !strings.Contains(msg, "42") {
		t.Errorf("mismatch message = %q, does not name the offending byte offset 42 — a bare boolean cannot locate the defect", msg)
	}
	if !strings.Contains(msg, "some-transcript") {
		t.Errorf("mismatch message = %q, does not name the transcript", msg)
	}

	if msg, mismatched := sweepMismatch("irrelevant", 0, got, got); mismatched {
		t.Errorf("sweepMismatch(...) reported mismatched = true for identical frame sequences, message = %q", msg)
	}
}

// --- Harness helpers, extracted for reuse by Slice 5's representative-offset stand-in ---

// decodeCanonical feeds transcript to a fresh Decoder in one Feed call and
// returns the frames it yields — the "canonical single-Feed parse" every
// split replay below is compared against (design.md's Offset-Sweep Harness
// section).
func decodeCanonical(t *testing.T, transcript []byte) []openaicompat.Frame {
	t.Helper()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed(transcript)
	if err != nil {
		t.Fatalf("canonical Feed() error = %v, want nil", err)
	}
	return frames
}

// decodeSplitAtOffset feeds transcript to a fresh Decoder in exactly two
// chunks split at offset, and returns the combined frame sequence in
// arrival order. Reused by Slice 5's representative-offset, non-exhaustive
// check on the multi-megabyte fixture that this sweep structurally
// excludes (R-ASD-014, S-ASD-050) — the same split-and-compare shape,
// applied to a fixture too large for the exhaustive loop above.
func decodeSplitAtOffset(t *testing.T, transcript []byte, offset int) []openaicompat.Frame {
	t.Helper()

	d := openaicompat.NewDecoder(0)
	first, err := d.Feed(transcript[:offset])
	if err != nil {
		t.Fatalf("first Feed() error = %v, want nil (offset %d)", err, offset)
	}
	second, err := d.Feed(transcript[offset:])
	if err != nil {
		t.Fatalf("second Feed() error = %v, want nil (offset %d)", err, offset)
	}
	return append(first, second...)
}

// decodeByteAtATime feeds transcript to a fresh Decoder one byte per Feed
// call. It is design.md's "byte-at-a-time replay (O(N), catches
// multi-split)" — a single decode that exercises every possible split
// simultaneously, complementing the exhaustive two-chunk sweep above,
// which only ever splits into exactly two chunks per replay.
func decodeByteAtATime(t *testing.T, transcript []byte) []openaicompat.Frame {
	t.Helper()

	d := openaicompat.NewDecoder(0)
	var frames []openaicompat.Frame
	for i := range transcript {
		got, err := d.Feed(transcript[i : i+1])
		if err != nil {
			t.Fatalf("byte-at-a-time Feed() error = %v, want nil (byte %d)", err, i)
		}
		frames = append(frames, got...)
	}
	return frames
}

// sweepMismatch reports whether got and want frame sequences differ and,
// if so, a message identifying both the transcript and the byte offset the
// caller was replaying when the mismatch was found (S-ASD-045): once a
// mismatch can occur at any of len(transcript)+1 offsets, a bare boolean
// equality check cannot locate the defect on its own.
//
// Factoring this into a separately callable pure function is what turns
// S-ASD-045 from an inspection-only obligation into a runnable, failable
// [test] scenario — see TestOffsetSweep_MismatchMessageNamesOffendingByteOffset
// above, and the spec's recorded upgrade condition.
func sweepMismatch(transcriptName string, offset int, got, want []openaicompat.Frame) (msg string, mismatched bool) {
	if framesEqual(got, want) {
		return "", false
	}
	return fmt.Sprintf("transcript %q split at byte offset %d: frames = %+v, want %+v (canonical decode)", transcriptName, offset, got, want), true
}

// --- S-ASD-049 durability (post-verify W1): the size guard as a pure function ---

// sweepFixtureBoundViolation names one candidate transcript whose size
// exceeds maxSweepFixtureBytes (S-ASD-049): the transcript's name and its
// exact byte size, so a caller can report which fixture is over the bound
// and by how much.
type sweepFixtureBoundViolation struct {
	name string
	size int
}

// String renders a violation exactly as the guard has always reported it,
// so TestOffsetSweep_FixturesWithinSweepBound's failure text is unchanged
// by this refactor.
func (v sweepFixtureBoundViolation) String() string {
	return fmt.Sprintf("transcript %q is %d bytes, want <= %d (S-ASD-049)", v.name, v.size, maxSweepFixtureBytes)
}

// checkSweepFixtureBound is R-ASD-014/S-ASD-049's structural size guard,
// factored into a pure function over any candidate transcript set — never
// only the real sweepTranscripts registry — so both of S-ASD-049's clauses
// stay permanently provable: the positive clause by
// TestOffsetSweep_FixturesWithinSweepBound (the real registry, want zero
// violations) and the negative clause by
// TestOffsetSweep_SyntheticOverweightFixture_Fails (a synthetic candidate
// built in-test, want exactly one named violation) — without ever staging
// a violating fixture inside the real registry the way Slice 3's original,
// transient proof did.
//
// The bound is inclusive: len(tr.transcript) == maxSweepFixtureBytes does
// NOT violate; only strictly greater does (design.md Rev 3) — this
// comparison must never be silently flipped to a strict `<`.
func checkSweepFixtureBound(candidates []sweepTranscript) []sweepFixtureBoundViolation {
	var violations []sweepFixtureBoundViolation
	for _, tr := range candidates {
		if len(tr.transcript) > maxSweepFixtureBytes {
			violations = append(violations, sweepFixtureBoundViolation{name: tr.name, size: len(tr.transcript)})
		}
	}
	return violations
}
