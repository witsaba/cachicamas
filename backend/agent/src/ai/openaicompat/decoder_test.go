package openaicompat_test

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// TestDecoder_SingleFrameYieldsExactlyOneFrame covers S-ASD-001 and
// S-ASD-002 (R-ASD-001): one well-formed frame fed in one chunk, followed
// by its terminating blank line, yields exactly one frame whose event
// type and data equal the input's field values byte for byte — and the
// terminating blank line produces no second, empty frame.
func TestDecoder_SingleFrameYieldsExactlyOneFrame(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("event: greeting\ndata: hello\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}

	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1 (S-ASD-002)", len(frames))
	}

	got := frames[0]
	if got.Event != "greeting" {
		t.Errorf("Event = %q, want %q (S-ASD-001)", got.Event, "greeting")
	}
	if !bytes.Equal(got.Data, []byte("hello")) {
		t.Errorf("Data = %q, want %q (S-ASD-001)", got.Data, "hello")
	}
}

// TestDecoder_MultiByteDataSurvivesByteExact covers S-ASD-003: a frame
// whose data contains non-ASCII multi-byte content decodes to a
// byte-identical payload.
func TestDecoder_MultiByteDataSurvivesByteExact(t *testing.T) {
	t.Parallel()

	// café (2-byte and 1-byte runes), a heart with a 3-byte variation
	// selector, and a 4-byte emoji — one fixture exercising every UTF-8
	// width class in a single data value.
	const payload = "café ❤️ \U0001F600"

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: " + payload + "\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want exactly 1", len(frames))
	}
	if !bytes.Equal(frames[0].Data, []byte(payload)) {
		t.Errorf("Data = %q, want %q (S-ASD-003)", frames[0].Data, payload)
	}
}

// TestDecoder_ThreeFramesInOneChunkYieldInOrder covers S-ASD-004
// (R-ASD-002): three distinct frames delivered in one input chunk are
// yielded in the same order they appear in the bytes.
func TestDecoder_ThreeFramesInOneChunkYieldInOrder(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: first\n\ndata: second\n\ndata: third\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}

	want := []string{"first", "second", "third"}
	if len(frames) != len(want) {
		t.Fatalf("Feed() yielded %d frames, want %d (S-ASD-004)", len(frames), len(want))
	}
	for i, w := range want {
		if !bytes.Equal(frames[i].Data, []byte(w)) {
			t.Errorf("frames[%d].Data = %q, want %q (S-ASD-004)", i, frames[i].Data, w)
		}
	}
}

// TestDecoder_IdenticalPayloadFramesAreNotCoalesced covers S-ASD-005
// (R-ASD-002): two frames with identical payloads are both yielded, never
// merged into one.
func TestDecoder_IdenticalPayloadFramesAreNotCoalesced(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: same\n\ndata: same\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 2 {
		t.Fatalf("Feed() yielded %d frames, want exactly 2 — identical payloads must not coalesce (S-ASD-005)", len(frames))
	}
	for i, f := range frames {
		if !bytes.Equal(f.Data, []byte("same")) {
			t.Errorf("frames[%d].Data = %q, want %q", i, f.Data, "same")
		}
	}
}

// TestDecoder_OneFramePerChunkPreservesFeedingOrder covers S-ASD-006
// (R-ASD-002): frames delivered one per input chunk are yielded in the
// same order the chunks were fed.
func TestDecoder_OneFramePerChunkPreservesFeedingOrder(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	chunks := []string{"data: alpha\n\n", "data: beta\n\n", "data: gamma\n\n"}

	var got []string
	for _, chunk := range chunks {
		frames, err := d.Feed([]byte(chunk))
		if err != nil {
			t.Fatalf("Feed(%q) error = %v, want nil", chunk, err)
		}
		for _, f := range frames {
			got = append(got, string(f.Data))
		}
	}

	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("got %d frames across chunks, want %d (S-ASD-006)", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("frame %d = %q, want %q — feeding order not preserved (S-ASD-006)", i, got[i], w)
		}
	}
}

// TestDecoder_CompletesWithNoNetworkSetup covers S-ASD-007 (R-ASD-003):
// this test itself is the proof — it performs no network setup of any
// kind (no listener, no dial, no httptest server, no goroutine) and
// decoding still completes and yields the expected frame from bytes
// alone.
func TestDecoder_CompletesWithNoNetworkSetup(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: no-network\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if err := d.Finish(); err != nil {
		t.Fatalf("Finish() error = %v, want nil", err)
	}
	if len(frames) != 1 || string(frames[0].Data) != "no-network" {
		t.Fatalf("Feed() = %+v, want one frame with Data %q (S-ASD-007)", frames, "no-network")
	}
}

// TestDecoder_StartsNoGoroutines covers S-ASD-008 (R-ASD-003): the
// runtime goroutine count sampled immediately before feeding and again
// after the end-of-input signal is unchanged — the decoder started none.
//
// Deliberately not t.Parallel(): this test's assertion is a delta between
// two runtime.NumGoroutine() samples taken around the call under test, and
// running it alongside other parallel subtests would add unrelated
// goroutine churn to both samples, not just one — noise this specific
// scenario does not need.
func TestDecoder_StartsNoGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()

	d := openaicompat.NewDecoder(0)
	if _, err := d.Feed([]byte("data: alpha\n\ndata: beta\n\n")); err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if err := d.Finish(); err != nil {
		t.Fatalf("Finish() error = %v, want nil", err)
	}

	after := runtime.NumGoroutine()
	if after != before {
		t.Errorf("goroutine count before=%d after=%d, want equal — Feed/Finish must start none (S-ASD-008)", before, after)
	}
}

// TestDecoder_TwoIndependentDecodersYieldIdenticalFrames covers S-ASD-009
// (R-ASD-003): one transcript decoded by two independent Decoder
// instances yields identical frame sequences — decoding is a pure
// function of the bytes fed, not of any shared or accumulated state.
func TestDecoder_TwoIndependentDecodersYieldIdenticalFrames(t *testing.T) {
	t.Parallel()

	const transcript = "event: a\ndata: one\n\ndata: two\n\nevent: c\ndata: three\n\n"

	d1 := openaicompat.NewDecoder(0)
	frames1, err := d1.Feed([]byte(transcript))
	if err != nil {
		t.Fatalf("d1.Feed() error = %v, want nil", err)
	}

	d2 := openaicompat.NewDecoder(0)
	frames2, err := d2.Feed([]byte(transcript))
	if err != nil {
		t.Fatalf("d2.Feed() error = %v, want nil", err)
	}

	if len(frames1) != len(frames2) {
		t.Fatalf("frame counts differ: %d vs %d (S-ASD-009)", len(frames1), len(frames2))
	}
	for i := range frames1 {
		if frames1[i].Event != frames2[i].Event || !bytes.Equal(frames1[i].Data, frames2[i].Data) {
			t.Errorf("frame %d differs: %+v vs %+v (S-ASD-009)", i, frames1[i], frames2[i])
		}
	}
}

// TestDecoder_FrameDataIsACopyUnaffectedByLaterFeed pins design.md's Frame
// value decision: Data is always a copy, so a previously returned Frame's
// Data is never observably affected by a later Feed call that reuses or
// compacts the decoder's internal accumulation buffers. Not a numbered
// spec scenario on its own — R-ASD-001 forbids altering a payload, and
// this triangulates that guarantee across a second Feed call, which the
// design's own 27.1 testing-strategy row calls out by name ("a frame's
// Data unchanged after a later Feed").
func TestDecoder_FrameDataIsACopyUnaffectedByLaterFeed(t *testing.T) {
	t.Parallel()

	d := openaicompat.NewDecoder(0)
	frames, err := d.Feed([]byte("data: original\n\n"))
	if err != nil {
		t.Fatalf("Feed() error = %v, want nil", err)
	}
	if len(frames) != 1 {
		t.Fatalf("Feed() yielded %d frames, want 1", len(frames))
	}
	first := frames[0].Data
	want := append([]byte(nil), first...)

	if _, err := d.Feed([]byte("data: second-frame-forces-buffer-reuse\n\n")); err != nil {
		t.Fatalf("second Feed() error = %v, want nil", err)
	}

	if !bytes.Equal(first, want) {
		t.Errorf("first frame's Data changed after a later Feed: got %q, want %q (copy pin)", first, want)
	}
}
