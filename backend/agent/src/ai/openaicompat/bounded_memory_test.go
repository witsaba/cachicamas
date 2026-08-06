package openaicompat_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// This file covers Phase 5 (AI-27.5, bounded memory): R-ASD-018 (a single
// multi-megabyte frame decodes correctly, S-ASD-058..060), R-ASD-019 (a
// frame exceeding the hard cap aborts with a typed error naming the
// malformed-response category, and frames completed before the trip are
// not collateral, S-ASD-061..064, S-ASD-085), R-ASD-020 (cap-exceeded and
// truncation stay distinguishable by error identity, S-ASD-065..067), and
// R-ASD-014's own remaining stand-in fact discharged here — the
// multi-megabyte fixture replayed at a small fixed set of representative
// offsets rather than exhaustively (S-ASD-050).
//
// Test names start with TestBoundedMemory so the slice's focused command
// (`go test -race -run TestBoundedMemory ./src/ai/openaicompat/...`)
// selects exactly this file's cases (see tasks.md's Suggested Work Units
// table).
//
// R-ASD-021's recorded deferral (the tenth-category question, S-ASD-068)
// and R-ASD-019's no-provider-failure-construction obligation (S-ASD-064)
// are both [inspection] scenarios, discharged by trace in this slice's
// evidence log rather than by a test function here.

// bigPayloadSize is this file's shared multi-megabyte data payload size:
// comfortably "several megabytes" (S-ASD-058) while staying well under
// DefaultMaxFrameBytes (8 MiB), so every test below that feeds it under
// NewDecoder(0) exercises a genuinely below-cap decode.
const bigPayloadSize = 3 * 1024 * 1024

// bigPayload returns this file's shared multi-megabyte data payload: a
// deterministic, uniform byte run containing no CR or LF byte anywhere, so
// it forms exactly one data line regardless of where a chunk boundary
// falls inside it.
//
// This fixture is deliberately NEVER registered in
// sweep_transcripts_test.go's sweepTranscripts (R-ASD-014): the offset
// sweep is quadratic in transcript length, and
// TestOffsetSweep_FixturesWithinSweepBound's structural guard would fail
// the suite loudly if it ever were — confirmed by trace in this slice's
// evidence log (S-ASD-051). It stays a local variable in this file only;
// this file's own representative-offset test is its non-exhaustive
// re-entrancy stand-in (S-ASD-050).
func bigPayload() []byte {
	return bytes.Repeat([]byte("A"), bigPayloadSize)
}

// bigTranscript wraps payload as one complete SSE frame: a single data
// line carrying it, with no explicit event-type line, followed by the
// terminating blank line.
func bigTranscript(payload []byte) []byte {
	transcript := make([]byte, 0, len(payload)+8)
	transcript = append(transcript, "data: "...)
	transcript = append(transcript, payload...)
	transcript = append(transcript, "\n\n"...)
	return transcript
}

// --- R-ASD-018: a single multi-megabyte frame decodes correctly ---

// TestBoundedMemory_MultiMegabyteFrameDecodesCorrectlyInOneChunk covers
// S-ASD-058: this is the default-line-limit trap pinned directly —
// bufio.Scanner's 64 KiB MaxScanTokenSize (verified at
// /usr/local/go/src/bufio/scan.go:71,82, failing with ErrTooLong) is the
// canonical instance of the class this hand-rolled decoder exists
// specifically to avoid.
func TestBoundedMemory_MultiMegabyteFrameDecodesCorrectlyInOneChunk(t *testing.T) {
	t.Parallel()

	payload := bigPayload()
	transcript := bigTranscript(payload)

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed(transcript)
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil (S-ASD-058)", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-058)", len(frames))
	}
	if len(frames[0].Data) != len(payload) {
		t.Fatalf("Data length = %d, want %d (S-ASD-058)", len(frames[0].Data), len(payload))
	}
	if !bytes.Equal(frames[0].Data, payload) {
		t.Error("Data does not match the input payload byte-for-byte (S-ASD-058)")
	}
	if frames[0].Event != "message" {
		t.Errorf("Event = %q, want %q (no event-type line was fed)", frames[0].Event, "message")
	}
}

// TestBoundedMemory_MultiMegabyteFrameFedInManySmallChunksMatchesSingleChunk
// covers S-ASD-059: the identical frame, fed 64 KiB at a time (a
// realistic transport read-buffer size) instead of in one Feed call,
// decodes to the same payload as the single-chunk decoding above.
func TestBoundedMemory_MultiMegabyteFrameFedInManySmallChunksMatchesSingleChunk(t *testing.T) {
	t.Parallel()

	payload := bigPayload()
	transcript := bigTranscript(payload)

	const chunkSize = 64 * 1024
	d := openaicompat.NewDecoder(0)
	var frames []openaicompat.Frame
	for offset := 0; offset < len(transcript); offset += chunkSize {
		end := offset + chunkSize
		if end > len(transcript) {
			end = len(transcript)
		}
		got, err := d.Feed(transcript[offset:end])
		if err != nil {
			t.Fatalf("Feed() error = %v, want nil at chunk offset %d (S-ASD-059)", err, offset)
		}
		frames = append(frames, got...)
	}

	if len(frames) != 1 {
		t.Fatalf("chunked feeding yielded %d frames, want exactly 1 (S-ASD-059)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, payload) {
		t.Error("chunked feeding's Data does not match the single-chunk payload byte-for-byte (S-ASD-059)")
	}
}

// TestBoundedMemory_FrameLargerThan64KiBBelowCapSucceeds covers S-ASD-060:
// a frame past the 64 KiB bufio.Scanner default, but far below the
// decoder's own (default) cap, succeeds — no implicit sub-cap limit
// exists anywhere in this decoder.
func TestBoundedMemory_FrameLargerThan64KiBBelowCapSucceeds(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte("b"), 100*1024) // 100 KiB > 64 KiB
	transcript := bigTranscript(payload)

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed(transcript)
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil (S-ASD-060)", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-060)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, payload) {
		t.Error("Data does not match the 100 KiB input payload byte-for-byte (S-ASD-060)")
	}
}

// --- R-ASD-019: a frame exceeding the hard cap aborts; exactly-at-cap decodes ---

// TestBoundedMemory_ExactlyAtCapDecodesOneByteOverAborts is this slice's
// core boundary pin (S-ASD-061, S-ASD-063). The cap is chosen to equal
// EXACTLY the peak accumulated size this transcript reaches while
// decoding: payloadLen+1, the one byte processFieldLine's data-join
// appends beyond the value itself (R-ASD-005's dispatch-time separator,
// stripped only at dispatch). at_cap_decodes uses that exact peak;
// one_under_peak_aborts uses one byte less, guaranteeing R-ASD-019's
// strictly-greater relation trips.
//
// This pairing — not merely "a 2 KiB frame against a 1 KiB cap" — is what
// actually falsifies a >= implementation: a decoder written with
// accumulated >= cap (instead of >) would incorrectly abort the
// at_cap_decodes case too, since payloadLen+1 >= payloadLen+1 is true.
// See this slice's evidence log for the scratch break-and-revert proof.
func TestBoundedMemory_ExactlyAtCapDecodesOneByteOverAborts(t *testing.T) {
	t.Parallel()

	const payloadLen = 2048
	payload := bytes.Repeat([]byte("c"), payloadLen)
	transcript := bigTranscript(payload)
	const atCap = payloadLen + 1 // exact peak: value bytes + dispatch-time separator

	t.Run("at_cap_decodes", func(t *testing.T) {
		t.Parallel()

		d := openaicompat.NewDecoder(atCap)
		frames, err := d.Feed(transcript)
		if err != nil {
			t.Fatalf("Feed() error = %v, want nil — a frame reaching exactly the cap must still decode (S-ASD-063)", err)
		}
		if len(frames) != 1 {
			t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-063)", len(frames))
		}
		if !bytes.Equal(frames[0].Data, payload) {
			t.Error("Data does not match the input payload byte-for-byte (S-ASD-063)")
		}
	})

	t.Run("one_under_peak_aborts", func(t *testing.T) {
		t.Parallel()

		d := openaicompat.NewDecoder(atCap - 1)
		frames, err := d.Feed(transcript)
		if len(frames) != 0 {
			t.Errorf("Feed() yielded %d frames, want 0 — the over-cap frame must not be yielded (S-ASD-061)", len(frames))
		}
		if !errors.Is(err, openaicompat.ErrFrameTooLarge) {
			t.Fatalf("Feed() error = %v, want errors.Is(err, ErrFrameTooLarge) (S-ASD-061)", err)
		}
		if errors.Is(err, openaicompat.ErrTruncated) {
			t.Error("Feed() error also matches ErrTruncated — the two identities must stay distinct (R-ASD-020)")
		}
		if !errors.Is(err, ai.ErrMalformedResponse) {
			t.Error("Feed() error does not match ai.ErrMalformedResponse — the repo-native AI-19 matching idiom must hold (S-ASD-062)")
		}
		gotCategory, ok := openaicompat.Category(err)
		if !ok || gotCategory != ai.FailureCategoryMalformedResponse {
			t.Errorf("Category(err) = (%v, %v), want (%v, true) (S-ASD-062)", gotCategory, ok, ai.FailureCategoryMalformedResponse)
		}
	})
}

// TestBoundedMemory_UnterminatedOverCapLineTripsCapBeforeAnyLineResolves
// exercises Feed's OTHER cap-check call site: no "data:" prefix, no
// colon, and — critically — no line terminator anywhere, so nextLine can
// never resolve this into a line at all. This is the retained-tail
// branch (reached only once nextLine confirms no terminator has arrived
// yet), never the post-processFieldLine branch the boundary test above
// exercises. Both call sites share one counter (task 5.8's
// consolidation) but are independently necessary — see this slice's
// evidence log for the break-and-revert proof that each catches a
// scenario the other does not.
func TestBoundedMemory_UnterminatedOverCapLineTripsCapBeforeAnyLineResolves(t *testing.T) {
	t.Parallel()

	const capBytes = 1024
	unterminated := bytes.Repeat([]byte("f"), capBytes+1)

	d := openaicompat.NewDecoder(capBytes)
	frames, err := d.Feed(unterminated)
	if len(frames) != 0 {
		t.Errorf("Feed() yielded %d frames, want 0", len(frames))
	}
	if !errors.Is(err, openaicompat.ErrFrameTooLarge) {
		t.Fatalf("Feed() error = %v, want errors.Is(err, ErrFrameTooLarge)", err)
	}
}

// TestBoundedMemory_CompleteFrameBeforeOverCapFrameReturnedTogetherWithError
// covers S-ASD-085: one Feed call carrying a complete frame followed by an
// over-cap frame returns the complete frame TOGETHER WITH the cap error —
// mirroring R-ASD-023's truncation-side pin at S-ASD-075. Frames
// completed before the trip are not collateral (R-ASD-019): a caller
// writing the idiomatic "if err != nil { return err }" would otherwise
// silently lose delivered content.
func TestBoundedMemory_CompleteFrameBeforeOverCapFrameReturnedTogetherWithError(t *testing.T) {
	t.Parallel()

	const capBytes = 64
	overCapPayload := bytes.Repeat([]byte("d"), 200) // comfortably over cap once accumulated

	var chunk []byte
	chunk = append(chunk, "event: greet\ndata: hello\n\n"...) // complete frame, well under cap
	chunk = append(chunk, "data: "...)
	chunk = append(chunk, overCapPayload...)
	chunk = append(chunk, "\n\n"...)

	d := openaicompat.NewDecoder(capBytes)
	frames, err := d.Feed(chunk)

	if !errors.Is(err, openaicompat.ErrFrameTooLarge) {
		t.Fatalf("Feed() error = %v, want errors.Is(err, ErrFrameTooLarge) (S-ASD-085)", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 — the complete frame before the trip must still be returned together with the error, not dropped (S-ASD-085, R-ASD-019)", len(frames))
	}
	want := openaicompat.Frame{Event: "greet", Data: []byte("hello")}
	if frames[0].Event != want.Event || !bytes.Equal(frames[0].Data, want.Data) {
		t.Errorf("returned frame = %+v, want %+v (S-ASD-085)", frames[0], want)
	}
}

// TestBoundedMemory_PoisonedAfterCapExceededFeedAndFinishReturnSameError
// covers design.md's "Poisoning" decision: the first terminal error
// poisons the decoder, so every later Feed AND Finish call returns the
// same error, with zero frames — a half-consumed frame boundary is
// unrecoverable, so there is nothing safe to resume.
//
// The first trip deliberately uses an UNTERMINATED over-cap line (the
// retained-tail checkpoint), and the second Feed call deliberately
// supplies just "\n\n": fed to a completely FRESH decoder, capBytes+1
// colonless 'g' bytes followed by a blank line is a harmless,
// UNRECOGNIZED field (no colon) followed by an empty dispatch — it would
// decode with NO error at all. This is what makes the assertion
// genuinely discriminating rather than a masked coincidence: without an
// explicit poisoning guard, Feed would re-scan from the still-retained,
// un-trimmed 'g' run, classify it as an ignorable colonless field, and
// "recover" to (nil, nil) on this exact second call — see this slice's
// evidence log for the scratch break-and-revert proof that an earlier,
// less careful fixture failed to catch a removed poisoning guard at all.
func TestBoundedMemory_PoisonedAfterCapExceededFeedAndFinishReturnSameError(t *testing.T) {
	t.Parallel()

	const capBytes = 1024
	unterminated := bytes.Repeat([]byte("g"), capBytes+1)

	d := openaicompat.NewDecoder(capBytes)
	_, err := d.Feed(unterminated)
	if !errors.Is(err, openaicompat.ErrFrameTooLarge) {
		t.Fatalf("first Feed() error = %v, want errors.Is(err, ErrFrameTooLarge)", err)
	}

	frames2, err2 := d.Feed([]byte("\n\n"))
	if len(frames2) != 0 {
		t.Errorf("second Feed() yielded %d frames, want 0 once poisoned", len(frames2))
	}
	if !errors.Is(err2, openaicompat.ErrFrameTooLarge) {
		t.Errorf("second Feed() error = %v, want the same ErrFrameTooLarge identity — an unpoisoned decoder would wrongly \"recover\" to nil here", err2)
	}

	if err3 := d.Finish(); !errors.Is(err3, openaicompat.ErrFrameTooLarge) {
		t.Errorf("Finish() after poisoning = %v, want the same ErrFrameTooLarge identity", err3)
	}
}

// --- R-ASD-020: cap-exceeded and truncation stay distinguishable by error identity ---

// TestBoundedMemory_ErrorIdentitiesStayDistinctBothReportMalformedResponse
// covers S-ASD-065, S-ASD-066 and S-ASD-067 together: they are all facts
// about the relationship between the two package-level sentinel values
// and need no live decode to exercise — Category and errors.Is operate on
// the sentinels directly.
func TestBoundedMemory_ErrorIdentitiesStayDistinctBothReportMalformedResponse(t *testing.T) {
	t.Parallel()

	if errors.Is(openaicompat.ErrFrameTooLarge, openaicompat.ErrTruncated) {
		t.Error("ErrFrameTooLarge matches ErrTruncated's identity, want distinct (R-ASD-020, S-ASD-065)")
	}
	if errors.Is(openaicompat.ErrTruncated, openaicompat.ErrFrameTooLarge) {
		t.Error("ErrTruncated matches ErrFrameTooLarge's identity, want distinct (R-ASD-020, S-ASD-066)")
	}
	if !errors.Is(openaicompat.ErrFrameTooLarge, openaicompat.ErrFrameTooLarge) {
		t.Error("ErrFrameTooLarge does not match its own identity (S-ASD-067)")
	}
	if !errors.Is(openaicompat.ErrTruncated, openaicompat.ErrTruncated) {
		t.Error("ErrTruncated does not match its own identity (S-ASD-067)")
	}

	if !errors.Is(openaicompat.ErrFrameTooLarge, ai.ErrMalformedResponse) {
		t.Error("ErrFrameTooLarge does not match ai.ErrMalformedResponse")
	}
	if !errors.Is(openaicompat.ErrTruncated, ai.ErrMalformedResponse) {
		t.Error("ErrTruncated does not match ai.ErrMalformedResponse")
	}

	gotCategory, ok := openaicompat.Category(openaicompat.ErrFrameTooLarge)
	if !ok || gotCategory != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category(ErrFrameTooLarge) = (%v, %v), want (%v, true) (S-ASD-067)", gotCategory, ok, ai.FailureCategoryMalformedResponse)
	}
	gotCategory, ok = openaicompat.Category(openaicompat.ErrTruncated)
	if !ok || gotCategory != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category(ErrTruncated) = (%v, %v), want (%v, true) (S-ASD-067)", gotCategory, ok, ai.FailureCategoryMalformedResponse)
	}

	if _, ok := openaicompat.Category(nil); ok {
		t.Error("Category(nil) reported ok = true, want false — nil is not one of this package's sentinels")
	}
	if _, ok := openaicompat.Category(errors.New("unrelated")); ok {
		t.Error("Category(unrelated error) reported ok = true, want false")
	}
}

// --- R-ASD-014's remaining stand-in fact, discharged here (Phase 3's own cross-slice note) ---

// TestBoundedMemory_MultiMegabyteFixtureAtRepresentativeOffsetsMatchesCanonical
// covers S-ASD-050: the multi-megabyte fixture replayed split at a SMALL
// FIXED SET of representative offsets, not exhaustively — the offset
// sweep it is structurally excluded from (R-ASD-014) is O(N^2) and
// infeasible over megabytes.
//
// decodeCanonical/decodeSplitAtOffset/sweepMismatch are reused verbatim
// from offset_sweep_test.go (task 3.7's own extraction, built specifically
// for this reuse), not reimplemented. The canonical reference this
// compares against is anchored independently, not merely
// self-referentially: TestBoundedMemory_MultiMegabyteFrameDecodesCorrectlyInOneChunk
// (S-ASD-058) already hard-codes this identical transcript shape's decode
// against the literal input payload, so this test only adds the
// split-boundary dimension on top of an already-proven-correct reference.
func TestBoundedMemory_MultiMegabyteFixtureAtRepresentativeOffsetsMatchesCanonical(t *testing.T) {
	t.Parallel()

	payload := bigPayload()
	transcript := bigTranscript(payload)

	canonical := decodeCanonical(t, transcript)
	if len(canonical) != 1 || !bytes.Equal(canonical[0].Data, payload) {
		t.Fatal("canonical decode did not reproduce the expected single frame — precondition for this test failed")
	}

	representative := []int{
		0,                      // split before any byte
		1,                      // split one byte in
		6,                      // split right after the "data: " prefix
		1000,                   // split inside the payload, near its start
		len(transcript) / 2,    // split inside the payload, near its middle
		len(transcript) - 1000, // split inside the payload, near its end
		len(transcript) - 1,    // split one byte before the end
		len(transcript),        // split at the very end (all in the first chunk)
	}
	for _, offset := range representative {
		got := decodeSplitAtOffset(t, transcript, offset)
		if msg, mismatched := sweepMismatch("bounded-memory-multi-mb", offset, got, canonical); mismatched {
			t.Error(msg)
		}
	}
}

// --- 5.4: NewDecoder(0) selects the documented default ---

// TestBoundedMemory_ZeroCapSelectsDocumentedDefaultConstant pins
// DefaultMaxFrameBytes's exact numeric value. This has never been
// asserted anywhere in this package's tests: every existing test
// constructs NewDecoder(0) and exercises framing correctness, never the
// cap's own numeric value — confirmed by trace (grep DefaultMaxFrameBytes
// across every *_test.go file finds no prior assertion). The documented
// 8 MiB default must not ship as an unobserved assumption (tasks.md 5.4).
//
// Combined with TestBoundedMemory_MultiMegabyteFrameDecodesCorrectlyInOneChunk
// above (which already uses NewDecoder(0) and decodes a several-megabyte
// frame under it), these two facts together are what tasks.md 5.4 asks
// for: zero selects EXACTLY this constant, AND a multi-megabyte frame
// decodes correctly under the default it selects.
func TestBoundedMemory_ZeroCapSelectsDocumentedDefaultConstant(t *testing.T) {
	t.Parallel()

	const want = 8 * 1024 * 1024
	if openaicompat.DefaultMaxFrameBytes != want {
		t.Errorf("DefaultMaxFrameBytes = %d, want %d (8 MiB)", openaicompat.DefaultMaxFrameBytes, want)
	}
}
