// AI-36 judgment-day round 2 — regression pins for the confirmed findings
// against redactCredential's re-cap and against the content-type redaction
// path (stream.go). Internal package by the same load-bearing necessity
// a_i-36_jd1_test.go states: both cases drive unexported entry points
// (redactCredential, redactCredentialOccurrences,
// refuseNonStreamContentType) against the REAL captureLimit and
// truncationMarker, neither of which is reachable from package
// openaicompat_test without duplicating the constants they must stay in
// lockstep with.
package openaicompat

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestRedactCredential_GrowthRecapMarksItsOwnTruncation is the round-2
// finding-A pin. capture.go documents truncationMarker as "present on no
// capture that did not hit captureLimit", so a reader is entitled to treat
// "no marker" as "complete". redactCredential's own re-cap can cut a
// payload that arrived complete — substitution grows a payload whose
// credential is shorter than credentialRedactedPlaceholder — and the
// pre-fix code re-attached the marker only when the ORIGINAL excerpt
// already carried one, rendering a truncated excerpt indistinguishable
// from a complete one.
//
// The fixture is the smallest shape that reaches it: a 2-byte credential
// (client.go's New enforces non-empty and nothing more) whose 2000-byte
// echo is comfortably under captureLimit, so captureBody appends no
// marker, and whose substitution to 10 bytes per occurrence grows the
// payload to 10000 bytes — past captureLimit.
func TestRedactCredential_GrowthRecapMarksItsOwnTruncation(t *testing.T) {
	t.Parallel()

	cred := NewCredential("ab")
	body := bytes.Repeat([]byte("ab"), 1000)
	if len(body) >= captureLimit {
		t.Fatalf("fixture body is %d bytes, want under captureLimit (%d) so captureBody would append no marker — the whole point of the case", len(body), captureLimit)
	}

	got := redactCredential(body, cred)

	if maxLen := captureLimit + len(truncationMarker); len(got) > maxLen {
		t.Errorf("redactCredential returned %d bytes, want at most %d (captureLimit + len(truncationMarker)) — the documented largest capture this package can emit", len(got), maxLen)
	}
	if !bytes.HasSuffix(got, []byte(truncationMarker)) {
		t.Fatalf("redactCredential truncated a complete payload without the truncation marker (result %d bytes, input %d) — capture.go documents the marker as present on no capture that did not hit the limit, so a missing marker must keep meaning \"complete\"", len(got), len(body))
	}

	payload := got[:len(got)-len(truncationMarker)]
	if bytes.Contains(payload, []byte("ab")) {
		t.Errorf("redacted payload still contains the credential bytes")
	}
	if len(payload)%len(credentialRedactedPlaceholder) != 0 {
		t.Errorf("redacted payload is %d bytes, not a whole number of %d-byte placeholders — the cut left a partial placeholder, which reads as content", len(payload), len(credentialRedactedPlaceholder))
	}
	whole := bytes.Repeat([]byte(credentialRedactedPlaceholder), len(payload)/len(credentialRedactedPlaceholder))
	if !bytes.Equal(payload, whole) {
		t.Errorf("redacted payload is not a clean run of placeholders; tail = %q", payload[max(0, len(payload)-32):])
	}
}

// TestRedactCredential_CompleteCaptureStaysUnmarked is the control for the
// pin above: the marker must be added only when the re-cap actually cuts,
// so an excerpt that fits keeps meaning "complete".
func TestRedactCredential_CompleteCaptureStaysUnmarked(t *testing.T) {
	t.Parallel()

	cred := NewCredential("ab")
	got := redactCredential([]byte("prefix ab suffix"), cred)
	if bytes.Contains(got, []byte(truncationMarker)) {
		t.Errorf("redactCredential(%q) = %q, want no truncation marker on a capture that never hit the limit", "prefix ab suffix", got)
	}
	if want := "prefix " + credentialRedactedPlaceholder + " suffix"; string(got) != want {
		t.Errorf("redactCredential = %q, want %q", got, want)
	}
}

// TestRefuseNonStreamContentType_TailCollisionKeepsMediaType is the
// round-2 finding-B pin. redactCredentialTailFragment exists for BODIES
// cut by captureLimit; a header value is never truncated by the capture
// layer, so routing the Content-Type through it applies a
// truncation-edge heuristic where no truncation edge exists. A credential
// whose first minCredentialFragment bytes equal the media type's own tail
// makes the heuristic eat the media type R-ATS-023's refusal is required
// to name.
//
// Both halves are asserted together, because the fix must remove only the
// body-specific steps: the bare media type survives intact, AND a
// credential genuinely echoed into a Content-Type parameter is still
// redacted.
func TestRefuseNonStreamContentType_TailCollisionKeepsMediaType(t *testing.T) {
	t.Parallel()

	// 19 bytes, never spelled after a scheme keyword in this file: short
	// enough that credential_scan_test.go's sentinel patterns (20+ byte
	// runs) cannot match it. Its first four bytes collide with the tail of
	// the media type below.
	const collidingToken = "json-web-tok-9f2e"
	cred := NewCredential(collidingToken)

	tests := []struct {
		name        string
		contentType string
		wantEcho    bool
	}{
		{
			name:        "bare media type whose tail collides with the credential prefix",
			contentType: "application/json",
		},
		{
			name:        "media type whose parameter echoes the credential",
			contentType: `application/json; echo="` + collidingToken + `"`,
			wantEcho:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			}
			resp.Header.Set("Content-Type", tt.contentType)

			// The stored content type lives on the refusal's CAUSE, so the
			// whole unwrap chain is rendered — the same walk
			// a_i-36_1_test.go's own content-type pin performs.
			renderings := jdRenderEveryVerbAndUnwrapChain(refuseNonStreamContentType(resp, cred))

			sawMediaType := false
			sawPlaceholder := false
			for label, rendered := range renderings {
				if strings.Contains(rendered, "application/json") {
					sawMediaType = true
				}
				if strings.Contains(rendered, credentialRedactedPlaceholder) {
					sawPlaceholder = true
				}
				if strings.Contains(rendered, collidingToken) {
					t.Errorf("rendering %q reproduces the caller's own credential, want every whole occurrence redacted (D-4, R-AEM-019)", label)
				}
			}
			if !sawMediaType {
				t.Errorf("no rendering names the offending media type application/json (renderings: %v) — the truncation-edge heuristic is body-only and must never erase the media type R-ATS-023's refusal has to report", renderings)
			}
			if tt.wantEcho && !sawPlaceholder {
				t.Errorf("no rendering carries %s (renderings: %v), want the echoed credential replaced in place", credentialRedactedPlaceholder, renderings)
			}
		})
	}
}
