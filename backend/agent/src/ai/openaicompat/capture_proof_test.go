// AI-32.5 — bounded, sanitized capture proofs (R-AEM-015, R-AEM-016,
// S-AEM-056…063).
//
// This file is the RED-first proof surface for capture.go's D7
// finalization. Slice 1's capture.go is deliberately minimal
// (capture.go's own doc comment, capture_test.go's own file comment),
// so the assertions below fail by construction until this slice lands
// the LimitReader+1 probe, the truncation marker, the drain + single
// Close, and the cause-chain Unwrap to the inner error.
//
// # package openaicompat is load-bearing (S-AEM-060)
//
// S-AEM-060's sentinel `sk-AEM060-planted-in-body-only` matches
// credential_scan_test.go's raw regex `sk-[A-Za-z0-9_-]{20,}`. This
// file — and every file asserting the S-AEM-060/062 planted sentinel —
// MUST declare `package openaicompat` (internal). package
// openaicompat_test (external) would fail the credential-scan guard's
// build. The credential is planted deliberately so the guard's regex
// can never see it; that posture is load-bearing and called out in
// the S-ART-054 allowlist's own comment (stream_failure.go's
// ErrInBandErrorFrame allowlist entry cites the same posture).
//
// credential-scan:deliberate-plant — plants sAEM060Sentinel
// ("sk-AEM060-planted-in-body-only", S-AEM-060) to prove a body-only
// credential-shaped sentinel never reaches a typed error's rendered text.
// Declared here per AI-36's recursive credential scan (D-3, R-ART-022);
// enumerated in credential_scan_test.go's deliberatePlantAllowlist.

package openaicompat

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// S-AEM-056…058 — truncation invariant
// ---------------------------------------------------------------------

// captureProofSeededBody seeds a body of totalSize bytes, all 'X',
// returning a ReadCloser over a bytes.Reader. The reader is the
// simplest possible single-buffer source — it gives us exact byte
// counts without any pipelining or partial-read surprises, so the
// truncation arithmetic below is deterministic.
func captureProofSeededBody(t *testing.T, totalSize int) io.ReadCloser {
	t.Helper()
	buf := make([]byte, totalSize)
	for i := range buf {
		buf[i] = 'X'
	}
	return io.NopCloser(strings.NewReader(string(buf)))
}

// TestCapture_StopsExactlyAtCaptureLimit covers S-AEM-056: a body
// whose total length exceeds captureLimit produces a capturedBody
// whose bytes() length is exactly captureLimit + len(truncationMarker).
// Nothing more — captureBody MUST NOT retain beyond captureLimit + the
// marker. The +1 sentinel-byte probe in capture.go (D7 finalization) is
// what proves this from the read side; this test asserts the shape
// from the caller side.
func TestCapture_StopsExactlyAtCaptureLimit(t *testing.T) {
	// 4× the capture limit, so a runaway reader would obviously exceed
	// the cap if no bound were enforced.
	rc := captureProofSeededBody(t, 4*captureLimit)
	got := captureBody(rc)

	wantLen := captureLimit + len(truncationMarker)
	if len(got.bytes()) != wantLen {
		t.Errorf("len(captured bytes) = %d, want exactly %d (captureLimit + len(truncationMarker)) (S-AEM-056)", len(got.bytes()), wantLen)
	}
}

// TestCapture_TruncationMarkerPresentIffTruncated covers S-AEM-057:
// a body that exceeds captureLimit ends with truncationMarker; a body
// that fits within captureLimit does not. Both halves are required for
// the invariant to hold — neither is enough alone.
func TestCapture_TruncationMarkerPresentIffTruncated(t *testing.T) {
	t.Run("truncated body carries the marker", func(t *testing.T) {
		rc := captureProofSeededBody(t, 4*captureLimit)
		got := captureBody(rc)
		if !strings.HasSuffix(string(got.bytes()), truncationMarker) {
			t.Errorf("truncated capture bytes = %q, want suffix %q (S-AEM-057)", got.bytes(), truncationMarker)
		}
	})

	t.Run("non-truncated body has no marker", func(t *testing.T) {
		// 1 KiB body — well under captureLimit.
		rc := captureProofSeededBody(t, 1024)
		got := captureBody(rc)
		if strings.Contains(string(got.bytes()), truncationMarker) {
			t.Errorf("non-truncated capture bytes = %q, must NOT contain %q (S-AEM-057)", got.bytes(), truncationMarker)
		}
		// And the bytes are byte-for-byte the body itself.
		if len(got.bytes()) != 1024 {
			t.Errorf("len(captured bytes) = %d, want 1024 (full body retained) (S-AEM-057)", len(got.bytes()))
		}
	})
}

// TestCapture_TruncatedBodyPrefixIsTheFirstCaptureLimitBytes covers
// S-AEM-058: the truncated capture's first captureLimit bytes are the
// body's first captureLimit bytes (the marker is appended after, not
// replacing). This is the "truncate-back" mechanic the design D7
// promises.
func TestCapture_TruncatedBodyPrefixIsTheFirstCaptureLimitBytes(t *testing.T) {
	rc := captureProofSeededBody(t, 4*captureLimit)
	got := captureBody(rc)
	captured := got.bytes()

	if len(captured) != captureLimit+len(truncationMarker) {
		t.Fatalf("len = %d, want exactly %d (S-AEM-056)", len(captured), captureLimit+len(truncationMarker))
	}
	prefix := captured[:captureLimit]
	for i, b := range prefix {
		if b != 'X' {
			t.Errorf("prefix[%d] = %q, want 'X' — the captured prefix must be the body's first captureLimit bytes (S-AEM-058)", i, b)
			break
		}
	}
	if suffix := string(captured[captureLimit:]); suffix != truncationMarker {
		t.Errorf("suffix = %q, want %q (S-AEM-058)", suffix, truncationMarker)
	}
}

// ---------------------------------------------------------------------
// S-AEM-059 — multi-megabyte body drain, exactly one Close
// ---------------------------------------------------------------------

// captureProbeReadCloser records the order of its Read and Close
// calls and counts them, so the S-AEM-059 proof below can assert the
// invariant "Read past the captured prefix + one drain read, then
// Close exactly once" without trusting a mock-library's assertions
// about itself. closingCount is read at the end of each test after the
// captureBody call has returned, so the defer in captureBody has
// already fired.
type captureProbeReadCloser struct {
	data         []byte
	cursor       int
	reads        int
	closingCount int
	closed       bool
}

func (p *captureProbeReadCloser) Read(dst []byte) (int, error) {
	p.reads++
	if p.cursor >= len(p.data) {
		return 0, io.EOF
	}
	n := copy(dst, p.data[p.cursor:])
	p.cursor += n
	return n, nil
}

func (p *captureProbeReadCloser) Close() error {
	p.closingCount++
	p.closed = true
	return nil
}

// TestCapture_MultiMegabyteBodyIsDrainedAndClosedExactlyOnce covers
// S-AEM-059: a body many times the capture limit (4 MiB) is fully
// drained — the producer doesn't leave a slow-loris-shaped unread
// remainder on the wire — and is closed exactly once. Reads may
// exceed the captured-prefix read by exactly one (the +1 probe) plus
// whatever the drain itself takes; the precise number depends on
// Read's chunking, so the assertion is "Read ≥ 2" rather than an
// exact count.
func TestCapture_MultiMegabyteBodyIsDrainedAndClosedExactlyOnce(t *testing.T) {
	const bodySize = 4 << 20 // 4 MiB — many multiples of captureLimit.
	probe := &captureProbeReadCloser{data: make([]byte, bodySize)}

	got := captureBody(probe)

	// Closed exactly once (S-AEM-059: "closed exactly once").
	if probe.closingCount != 1 {
		t.Errorf("Close call count = %d, want exactly 1 (S-AEM-059)", probe.closingCount)
	}
	if !probe.closed {
		t.Error("Close never invoked, want one invocation (S-AEM-059)")
	}
	// Drained fully — cursor reached EOF.
	if probe.cursor != bodySize {
		t.Errorf("drained cursor = %d, want %d — the unread remainder MUST be drained (S-AEM-059)", probe.cursor, bodySize)
	}
	// Capture is bounded to captureLimit + marker.
	if len(got.bytes()) != captureLimit+len(truncationMarker) {
		t.Errorf("len(captured bytes) = %d, want exactly %d (S-AEM-059)", len(got.bytes()), captureLimit+len(truncationMarker))
	}
	// Reads cover at least the probe + drain. We assert ≥ 2 because the
	// +1 probe read is one and the drain read is at least one; the
	// exact split depends on Read's chunking for this size, but the
	// "drained fully" assertion above already proves the drain
	// completed regardless of the chunk count.
	if probe.reads < 2 {
		t.Errorf("Read calls = %d, want ≥ 2 (probe + at least one drain read) (S-AEM-059)", probe.reads)
	}
}

// ---------------------------------------------------------------------
// S-AEM-060…062 — sentinel credential never reaches typed error text
// ---------------------------------------------------------------------

// sAEM060Sentinel is the credential-like string planted ONLY in this
// test's body fixture (S-AEM-060). It matches the credential-scan
// guard's raw regex `sk-[A-Za-z0-9_-]{20,}` — the guard cannot see it
// because this file is internal-package by design (S-ATS-089 rev 4,
// credential_scan_test.go's scope). The sentinel is intentionally
// distinct from any real credential format to keep the deliberate
// plant obvious in code review.
const sAEM060Sentinel = "sk-AEM060-planted-in-body-only"

// TestCapture_SentinelCredentialNeverReachesTypedErrorText covers
// S-AEM-060/061/062: a body that contains a credential-like sentinel
// produces a capturedBody whose Error(), every cause in its
// Unwrap()-reachable chain's Error(), and every %v/%+v rendering of
// the failure built around it NEVER reproduce the sentinel. The
// guarantee is structural — capturedBody.Error() is a fixed string,
// and no upstream error in the chain renders the captured bytes —
// so the assertion is the S-AEM-060/061/062 spec line, not a
// textual fingerprint.
func TestCapture_SentinelCredentialNeverReachesTypedErrorText(t *testing.T) {
	body := fmt.Sprintf(
		"HTTP/1.1 500 Internal Server Error\nContent-Type: application/json\n\n"+
			"{\"error\":{\"message\":\"something went wrong: %s\"}}\n",
		sAEM060Sentinel,
	)
	failure := mapResponse(mustParseResponse(t, body), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil, want a constructed failure")
	}

	// S-AEM-060: Failure.Error() never contains the sentinel.
	if rendered := failure.Error(); strings.Contains(rendered, sAEM060Sentinel) {
		t.Errorf("Failure.Error() = %q contains the planted sentinel %q (S-AEM-060)", rendered, sAEM060Sentinel)
	}

	// S-AEM-061: every Unwrap()-reachable cause's Error() never
	// contains the sentinel. The chain here is
	//   Failure -> RateLimitTelemetry|nil -> capturedBody -> nil
	// and capturedBody.Error() is the fixed capturedBodyText, so this
	// assertion is the chain-shape guarantee (the captured bytes
	// travel on capturedBody, never on its Error()).
	for cause := error(failure); cause != nil; cause = errors.Unwrap(cause) {
		if rendered := cause.Error(); strings.Contains(rendered, sAEM060Sentinel) {
			t.Errorf("cause.Error() = %q contains the planted sentinel %q (S-AEM-061)", rendered, sAEM060Sentinel)
		}
	}

	// S-AEM-062: fmt's %v and %+v renderings of the failure never
	// contain the sentinel either — they go through Error(), which
	// never renders captured bytes (R-AEM-016, design D7).
	if rendered := fmt.Sprintf("%v", failure); strings.Contains(rendered, sAEM060Sentinel) {
		t.Errorf("fmt.Sprintf(%%v) = %q contains the planted sentinel %q (S-AEM-062)", rendered, sAEM060Sentinel)
	}
	if rendered := fmt.Sprintf("%+v", failure); strings.Contains(rendered, sAEM060Sentinel) {
		t.Errorf("fmt.Sprintf(%%+v) = %q contains the planted sentinel %q (S-AEM-062)", rendered, sAEM060Sentinel)
	}
}

// ---------------------------------------------------------------------
// S-AEM-063 — sentinel removed → surrounding body still captured
// ---------------------------------------------------------------------

// TestCapture_SentinelRemovedBodyStillRetainsSurroundingText covers
// S-AEM-063: a body whose credential sentinel is NOT in the body
// (the "sentinel-removed fixture" shape — opposite of the S-AEM-060
// planting) still retains the surrounding body text in the captured
// diagnostic. This proves the capture is genuine (the body is really
// being read up to captureLimit), NOT a vacuous pass — if the
// redaction alone (S-AEM-060) were the only guarantee, the test
// suite could pass trivially by simply not reading any body. The
// surrounding-text-still-present assertion is the triangulation case
// the design D7 redacted-capture property requires.
func TestCapture_SentinelRemovedBodyStillRetainsSurroundingText(t *testing.T) {
	const surrounding = "boundary-text-before-the-sentinel-region-and-after"
	// Fixture WITHOUT the sentinel — the surrounding text alone. This
	// is the "sentinel-removed" shape the spec describes.
	body := fmt.Sprintf(
		"HTTP/1.1 500 Internal Server Error\nContent-Type: application/json\n\n"+
			"{\"error\":{\"message\":\"%s middle %s\"}}\n",
		surrounding, surrounding,
	)
	failure := mapResponse(mustParseResponse(t, body), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil, want a constructed failure")
	}

	var cb capturedBody
	if !errors.As(failure, &cb) {
		t.Fatalf("errors.As(failure, &capturedBody): no capturedBody reachable")
	}
	if got := string(cb.bytes()); !strings.Contains(got, surrounding) {
		t.Errorf("captured bytes do not contain surrounding text %q — the capture is vacuous; the redaction alone is not a real capture (S-AEM-063)\n  captured: %q", surrounding, got)
	}
	// And — independent assertion — the body itself never had the
	// sentinel, so this is the positive-case triangulation for
	// S-AEM-060's negative case.
	if got := string(cb.bytes()); strings.Contains(got, sAEM060Sentinel) {
		t.Errorf("captured bytes = %q contains the planted sentinel %q — body had no sentinel; this is a leak (S-AEM-063)", got, sAEM060Sentinel)
	}
}

// ---------------------------------------------------------------------
// Sanity check on the failure shape used by the credential tests
// ---------------------------------------------------------------------

// TestCapture_SentinelCredentialFailureIsTheRightShape is a tiny
// proof that the failure mapResponse returns for the S-AEM-060 body
// is the failure shape the credential-scan guard's surrounding code
// expects (a *ai.Failure reachable through errors.As). Without it,
// the credential tests above might pass vacuously if mapResponse
// returned nil or a non-typed error.
func TestCapture_SentinelCredentialFailureIsTheRightShape(t *testing.T) {
	body := fmt.Sprintf(
		"HTTP/1.1 500 Internal Server Error\nContent-Type: application/json\n\n"+
			"{\"error\":{\"message\":\"x %s\"}}\n",
		sAEM060Sentinel,
	)
	failure := mapResponse(mustParseResponse(t, body), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil, want a constructed *ai.Failure")
	}
	var asFailure *ai.Failure
	if !errors.As(error(failure), &asFailure) {
		t.Fatalf("errors.As(error(failure), &*ai.Failure) = false — the S-AEM-060 sentinel tests' shape is wrong")
	}
}
