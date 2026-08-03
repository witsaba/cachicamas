package openaicompat_test

import (
	"bytes"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// This file covers Phase 4 (AI-27.4, keep-alives and unknowns): R-ASD-015
// (comment lines ignored without disturbing accumulation, S-ASD-052..053),
// R-ASD-016 (unknown field names ignored, unknown event-type names
// yielded, S-ASD-054..055), R-ASD-017 (an explicit event-type line in the
// data-only dialect has defined, tested behavior, S-ASD-057), and
// R-ASD-025's data-only half of the charter boundary — the dialect's
// terminal sentinel is framing-invisible (S-ASD-079..081).
//
// Test names start with TestKeepalive so the slice's focused command
// (`go test -race -run TestKeepalive ./src/ai/openaicompat/...`) selects
// exactly this file's cases (see tasks.md's Suggested Work Units table).

// --- R-ASD-015: comment lines are ignored without disturbing accumulation ---

// TestKeepalive_CommentLinesInterleavedLeavePayloadUnchanged covers
// S-ASD-052: a frame whose data lines are interleaved with comment lines —
// one before the first data line, one between the two data lines, and one
// right before the terminating blank line — decodes to the same payload as
// the same frame with every comment line removed.
//
// The comment BETWEEN "first" and "second" is this scenario's load-bearing
// case: a comment must not merely fail to yield a frame of its own, it must
// not perturb the in-progress data accumulator either. A decoder that
// misclassified a comment line as blank (dispatching early) would split
// this into two frames instead of one; a decoder that misrouted a
// comment's "value" into the data accumulator would corrupt the joined
// payload's content instead. Both failure modes are exercised below — the
// frame-count assertion catches the first, the byte-content assertion
// catches the second.
func TestKeepalive_CommentLinesInterleavedLeavePayloadUnchanged(t *testing.T) {
	t.Parallel()

	const withComments = ": keep-alive before\ndata: first\n: keep-alive between\ndata: second\n: keep-alive after\n\n"
	const withoutComments = "data: first\ndata: second\n\n"
	const want = "first\nsecond"

	baseline := openaicompat.NewDecoder(0)
	baselineFrames, err := baseline.Feed([]byte(withoutComments))
	if err != nil {
		t.Fatalf("baseline Feed() error = %v, want nil", err)
	}
	if len(baselineFrames) != 1 || !bytes.Equal(baselineFrames[0].Data, []byte(want)) {
		t.Fatalf("baseline (no comments) decoding = %+v, want data %q — pin the reference before comparing the comment-interleaved case to it", baselineFrames, want)
	}

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(withComments))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 — a comment must not trigger a dispatch of its own (S-ASD-052)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte(want)) {
		t.Errorf("Data = %q, want %q — a mid-frame comment must not perturb the data accumulator (S-ASD-052)", frames[0].Data, want)
	}
}

// TestKeepalive_CommentOnlyStreamEndingAtLineBoundaryYieldsZeroFrames
// covers S-ASD-053: a stream consisting solely of comment lines, each
// properly terminated (ending at a line boundary, never mid-line), yields
// zero frames and reports no error — from Feed, and from the end-of-input
// signal.
//
// This scenario's own reach is narrower than S-ASD-052's: no blank line
// exists anywhere in this fixture, so Feed cannot dispatch under ANY
// comment-handling implementation, correct or not — this test alone
// cannot distinguish "comments correctly ignored" from "comments
// misrouted elsewhere but never assembled into an observable frame". That
// distinguishing job belongs to S-ASD-052 above; the two scenarios are
// complementary, not individually self-sufficient — the same pattern this
// milestone already used for the BOM pair S-ASD-025/026 (see slice 2b's
// evidence log). This scenario's own job is the boundary-cleanliness fact:
// a comment-only stream that ends cleanly must not be mistaken for
// something requiring an error.
//
// Finish()'s "no error" here is a genuinely live assertion, not a stub
// artifact: AI-27.6/slice 6 implemented Finish()'s truncation detection
// (decoder.go, Finish around lines 282-299), which reports ErrTruncated
// whenever d.buf, d.data or d.eventType is still non-empty at end-of-input.
// This test proves a comment-only stream ending cleanly at a line boundary
// leaves all three empty, so that check does NOT fire — a Finish()
// implementation that mistakenly treated a comment-only clean ending as
// truncation would fail this assertion.
func TestKeepalive_CommentOnlyStreamEndingAtLineBoundaryYieldsZeroFrames(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(": one\n: two\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 0 {
		t.Errorf("Feed() yielded %d frames, want 0 (S-ASD-053)", len(frames))
	}

	if err := d.Finish(); err != nil {
		t.Errorf("Finish() error = %v, want nil — a comment-only stream ending at a line boundary must finish cleanly (S-ASD-053)", err)
	}
}

// --- R-ASD-016: unknown field names ignored; unknown event names yielded ---

// TestKeepalive_InventedFieldNameIgnoredWithoutResidue covers S-ASD-054: a
// field with an invented name alongside a valid data line decodes
// identically to the same frame without the invented field.
func TestKeepalive_InventedFieldNameIgnoredWithoutResidue(t *testing.T) {
	t.Parallel()

	want := []openaicompat.Frame{{Event: "message", Data: []byte("hi")}}

	without := openaicompat.NewDecoder(0)
	withoutFrames, err := without.Feed([]byte("data: hi\n\n"))
	if err != nil {
		t.Fatalf("without Feed() error = %v, want nil", err)
	}
	if !framesEqual(withoutFrames, want) {
		t.Fatalf("baseline decoding (no invented field) = %+v, want %+v", withoutFrames, want)
	}

	with := openaicompat.NewDecoder(0)
	withFrames, err := with.Feed([]byte("totally-invented-field: whatever\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("with Feed() error = %v, want nil", err)
	}
	if !framesEqual(withFrames, want) {
		t.Errorf("frames with invented field = %+v, want %+v — an unrecognized field name must be ignored without residue (S-ASD-054)", withFrames, want)
	}
}

// TestKeepalive_UnrecognizedEventTypeNameYieldedVerbatim covers S-ASD-055:
// an event-type line naming a type this repository has never seen is
// yielded on the frame verbatim rather than dropped, replaced or
// defaulted. No registry or allowlist exists to consult (R-ASD-016,
// inspected by S-ASD-056) — faithful compliance with § 9.2's dispatch step
// already produces this outcome.
func TestKeepalive_UnrecognizedEventTypeNameYieldedVerbatim(t *testing.T) {
	t.Parallel()

	const invented = "a-type-nobody-registered-xyz"
	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: " + invented + "\ndata: hi\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if frames[0].Event != invented {
		t.Errorf("Event = %q, want %q — an unrecognized event-type name must be yielded verbatim, not dropped or defaulted (S-ASD-055)", frames[0].Event, invented)
	}
}

// --- R-ASD-017: an unexpected event-type line has defined, tested behavior ---

// TestKeepalive_ExplicitEventTypeLineInDataOnlyTranscriptSetsOnlyThatFrame
// covers S-ASD-057: a data-only transcript into which one explicit
// event-type line is inserted sets that frame's type only — the frame
// before it and the frame after it both carry the default type, proving
// the inserted type neither leaks backward (structurally impossible, since
// the first frame already dispatched before the event-type line is even
// parsed) nor forward into the third.
func TestKeepalive_ExplicitEventTypeLineInDataOnlyTranscriptSetsOnlyThatFrame(t *testing.T) {
	t.Parallel()

	const transcript = "data: before\n\nevent: custom\ndata: middle\n\ndata: after\n\n"
	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(transcript))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil — an event-type line in a data-only dialect must not be treated as an error (S-ASD-057)", err)
	}
	if len(frames) != 3 {
		t.Fatalf("Feed() yielded %d frames, want exactly 3", len(frames))
	}
	if frames[0].Event != "message" || !bytes.Equal(frames[0].Data, []byte("before")) {
		t.Errorf("frames[0] = %+v, want Event %q Data %q", frames[0], "message", "before")
	}
	if frames[1].Event != "custom" || !bytes.Equal(frames[1].Data, []byte("middle")) {
		t.Errorf("frames[1] = %+v, want Event %q Data %q — the inserted event-type line must set this frame's type (S-ASD-057)", frames[1], "custom", "middle")
	}
	if frames[2].Event != "message" || !bytes.Equal(frames[2].Data, []byte("after")) {
		t.Errorf("frames[2] = %+v, want Event %q Data %q — the inserted type must not leak into the surrounding frame (S-ASD-057)", frames[2], "message", "after")
	}
}

// --- R-ASD-025 (partial): the [DONE] boundary — framing is not meaning ---

// doneSentinel is the OpenAI-compatible dialect's terminal sentinel
// string. It is a TEST-ONLY literal: R-ASD-025 forbids decoder.go and
// frame.go from containing this string at all, in any form. That
// inspection (S-ASD-082) formally closes at slice 6's milestone-close
// pass, but it is checked here too — see this slice's evidence log — since
// this slice is precisely the one most plausibly tempted to introduce it.
const doneSentinel = "[DONE]"

// TestKeepalive_DoneSentinelYieldedVerbatimNoSpecialHandling covers
// S-ASD-079: a frame whose data value is the dialect's terminal sentinel
// is yielded as an ordinary frame carrying that exact string as its
// payload — no special status, no early finish, no error. Finish()
// afterward still reports nil, proving the sentinel triggered no early
// completion and left the decoder's state exactly as an ordinary dispatch
// would.
func TestKeepalive_DoneSentinelYieldedVerbatimNoSpecialHandling(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: " + doneSentinel + "\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil — the sentinel must never produce an error (S-ASD-079)", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 — the sentinel frame must not be dropped (S-ASD-079)", len(frames))
	}
	if frames[0].Event != "message" {
		t.Errorf("Event = %q, want %q — the sentinel frame carries the ordinary default type, no special status (S-ASD-079)", frames[0].Event, "message")
	}
	if !bytes.Equal(frames[0].Data, []byte(doneSentinel)) {
		t.Errorf("Data = %q, want %q — the sentinel must be yielded verbatim (S-ASD-079)", frames[0].Data, doneSentinel)
	}

	if err := d.Finish(); err != nil {
		t.Errorf("Finish() error = %v, want nil — the sentinel must not trigger an early finish (S-ASD-079)", err)
	}
}

// TestKeepalive_FramesFollowingDoneSentinelStillYieldedNormally covers
// S-ASD-080 — the load-bearing lock against the AI-28.2-item-3 leak named
// in design.md's AI-27/AI-28 seam decision: frames arriving after a
// [DONE] frame in the same stream are still yielded, normally, with their
// actual content intact. The decoder suppresses nothing after the
// sentinel.
//
// Two frames follow the sentinel, not one, and both are asserted on their
// full content (Data), not merely counted: a fixture with no observable
// content after the interesting position would pass vacuously even under
// a "suppress everything after [DONE]" break — exactly the trap this
// milestone's evidence discipline calls out. Asserting the literal
// payloads is what makes the suppression failure mode observable.
func TestKeepalive_FramesFollowingDoneSentinelStillYieldedNormally(t *testing.T) {
	t.Parallel()

	const transcript = "data: " + doneSentinel + "\n\ndata: alpha\n\ndata: beta\n\n"
	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte(transcript))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 3 {
		t.Fatalf("Feed() yielded %d frames, want exactly 3 — nothing after the sentinel must be suppressed (S-ASD-080)", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte(doneSentinel)) {
		t.Errorf("frames[0].Data = %q, want %q", frames[0].Data, doneSentinel)
	}
	if !bytes.Equal(frames[1].Data, []byte("alpha")) {
		t.Errorf("frames[1].Data = %q, want %q — a frame following the sentinel must still be yielded normally (S-ASD-080)", frames[1].Data, "alpha")
	}
	if !bytes.Equal(frames[2].Data, []byte("beta")) {
		t.Errorf("frames[2].Data = %q, want %q — a second frame following the sentinel must also still be yielded (S-ASD-080)", frames[2].Data, "beta")
	}
}

// TestKeepalive_FramesDifferingOnlyInPayloadContentDecodeIdentically
// covers S-ASD-081: two frames differing only in their data content — one
// the terminal sentinel, one ordinary text — decode with identical
// framing outcomes. No part of the framing decision (frame count, event
// type, error-ness) depends on what the payload says.
func TestKeepalive_FramesDifferingOnlyInPayloadContentDecodeIdentically(t *testing.T) {
	t.Parallel()

	sentinelDecoder := openaicompat.NewDecoder(0)
	sentinelFrames, err := sentinelDecoder.Feed([]byte("data: " + doneSentinel + "\n\n"))
	if err != nil {
		t.Fatalf("sentinel Feed() error = %v, want nil", err)
	}

	ordinaryDecoder := openaicompat.NewDecoder(0)
	ordinaryFrames, err := ordinaryDecoder.Feed([]byte("data: ordinary-payload\n\n"))
	if err != nil {
		t.Fatalf("ordinary Feed() error = %v, want nil", err)
	}

	if len(sentinelFrames) != len(ordinaryFrames) {
		t.Fatalf("frame counts differ: sentinel = %d, ordinary = %d, want equal (S-ASD-081)", len(sentinelFrames), len(ordinaryFrames))
	}
	if sentinelFrames[0].Event != ordinaryFrames[0].Event {
		t.Errorf("event types differ: sentinel = %q, ordinary = %q, want equal — no framing decision may depend on payload content (S-ASD-081)", sentinelFrames[0].Event, ordinaryFrames[0].Event)
	}
	if bytes.Equal(sentinelFrames[0].Data, ordinaryFrames[0].Data) {
		t.Fatalf("fixture bug: sentinel and ordinary payloads must differ for this equivalence to mean anything")
	}
}
