// AI-32.5's capture layer, slice-1 trace (task 1.9): R-AEM-004's own
// sentence — "the bytes actually read MUST be retained as a bounded
// diagnostic reachable through the error chain and MUST NOT appear in
// Failure.Error()" — exercised through S-AEM-014…016's three bodies, at
// the capture layer specifically. failure_map_test.go already proves the
// CATEGORY/RETRYABLE/RAWLABEL outcomes for these same three bodies; this
// file proves the RETENTION itself: the raw bytes mapResponse actually read
// are recoverable by unwrapping the constructed failure.
//
// The exhaustive truncation-boundary and multi-megabyte drain proofs
// (S-AEM-056…059) are slice 2c's own RED-first responsibility
// (capture_proof_test.go) — capture.go's slice-1 implementation is
// deliberately minimal (design.md D1/D7, tasks.md 1.13), so this file
// stays within what that minimal implementation actually guarantees.
package openaicompat

import (
	"errors"
	"strings"
	"testing"
)

// captureRetentionCase pairs a full raw HTTP response with a substring
// that MUST be recoverable from the constructed failure's retained
// diagnostic.
type captureRetentionCase struct {
	name         string
	raw          string
	wantContains string
}

var captureRetentionCases = []captureRetentionCase{
	{
		name:         "non-JSON body retained verbatim (S-AEM-014)",
		raw:          "HTTP/1.1 429 Too Many Requests\nContent-Type: text/html\n\n<html>rate limited</html>",
		wantContains: "<html>rate limited</html>",
	},
	{
		name:         "wrong-shape JSON body retained verbatim (S-AEM-016)",
		raw:          "HTTP/1.1 500 Internal Server Error\nContent-Type: application/json\n\n{\"error\": 12345}",
		wantContains: `{"error": 12345}`,
	},
}

// TestCapture_RetainedBytesReachableThroughErrorChain proves R-AEM-004's
// retention sentence for the two non-empty bodies: errors.As reaches a
// capturedBody carrying the exact bytes mapResponse read, and that text is
// absent from Failure.Error().
func TestCapture_RetainedBytesReachableThroughErrorChain(t *testing.T) {
	for _, tc := range captureRetentionCases {
		t.Run(tc.name, func(t *testing.T) {
			failure := mapResponse(mustParseResponse(t, tc.raw), observedAtFixed)
			if failure == nil {
				t.Fatal("mapResponse = nil, want a constructed failure")
			}

			var cb capturedBody
			if !errors.As(failure, &cb) {
				t.Fatalf("errors.As(failure, &capturedBody): no capturedBody reachable through the error chain")
			}
			if got := string(cb.bytes()); !strings.Contains(got, tc.wantContains) {
				t.Errorf("captured bytes = %q, want substring %q — the body was not genuinely retained", got, tc.wantContains)
			}

			if rendered := failure.Error(); strings.Contains(rendered, tc.wantContains) {
				t.Errorf("Failure.Error() = %q leaks the retained body content %q", rendered, tc.wantContains)
			}
		})
	}
}

// TestCapture_EmptyBodyRetainsExactlyZeroBytes proves S-AEM-015's own
// capture-layer half: a zero-length body input reads back as a zero-length
// retained diagnostic — not an unread/absent capture, a genuinely empty
// one. TestFailureMap_UnparseableOrAbsentBodyStillMaps (failure_map_test.go)
// already proves the companion non-empty cases produce non-empty capture,
// so this assertion is not vacuous under the Empty Collection Rule.
func TestCapture_EmptyBodyRetainsExactlyZeroBytes(t *testing.T) {
	raw := "HTTP/1.1 503 Service Unavailable\n\n"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil, want a constructed failure")
	}

	var cb capturedBody
	if !errors.As(failure, &cb) {
		t.Fatalf("errors.As(failure, &capturedBody): no capturedBody reachable through the error chain")
	}
	if got := cb.bytes(); len(got) != 0 {
		t.Errorf("captured bytes = %q (%d bytes), want exactly zero", got, len(got))
	}
}

// TestCapture_TruncationMarkerIsPinned locks captureLimit and
// truncationMarker's literal values (design.md D1) so slice 2c's
// exhaustive truncation proofs (capture_proof_test.go, gated on 2a+2b
// merging) build against a fixed target rather than a value this slice
// could still drift. This does NOT exercise the truncation mechanism
// itself — captureBody does not yet append the marker on overflow
// (capture.go's own file comment) — that remains capture.go's D7
// finalization, slice 2c's own RED-first proof (S-AEM-056…059).
func TestCapture_TruncationMarkerIsPinned(t *testing.T) {
	if captureLimit != 8<<10 {
		t.Errorf("captureLimit = %d, want %d (design.md D1)", captureLimit, 8<<10)
	}
	if truncationMarker != "...(truncated)" {
		t.Errorf("truncationMarker = %q, want %q (design.md D1)", truncationMarker, "...(truncated)")
	}
}
