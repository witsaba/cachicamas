// AI-32.5 — bounded, sanitized body capture (R-AEM-015, R-AEM-016).
//
// capturedBody is the structural guarantee behind R-AEM-016: a credential
// echoed only inside an error body can never reach Failure.Error(), because
// no error text in this package's cause chain is ever built FROM the body's
// bytes — capturedBody.Error() is a fixed string, and the bytes themselves
// are reachable only through the unexported bytes() accessor, never
// rendered by any Error()/%v/%+v path (design.md D7).
//
// # D7 finalization (slice 2c, S-AEM-056…063)
//
// captureBody reads at most captureLimit+1 bytes via io.LimitReader. If
// the read returned exactly captureLimit+1 bytes, the body exceeded the
// limit and captureBody retains the first captureLimit bytes plus the
// fixed truncationMarker; if it returned fewer, captureBody retains
// whatever was read with no marker. Either way the remainder is
// drained (io.Copy to io.Discard) before the single Close — the
// slow-loris guard (S-AEM-059). The +1 probe is the cheap detection
// mechanism: a body exactly at the limit reads captureLimit bytes (no
// overflow), a body over the limit reads captureLimit+1 (overflow →
// marker), a body under the limit reads < captureLimit (under).

package openaicompat

import "io"

// captureLimit bounds how many body bytes captureBody retains as a
// diagnostic (R-AEM-015): the spec's own cap is 64 KiB, and this package
// chooses 8 KiB — vendor error bodies are small JSON, well under either
// bound (design.md D1).
const captureLimit = 8 << 10

// truncationMarker is the fixed ASCII suffix a capture that hit
// captureLimit carries, present on no capture that did not (R-AEM-015).
// Pinned to its literal value by TestCapture_TruncationMarkerIsPinned
// (capture_test.go, slice 1's test for the value alone; slice 2c's
// capture_proof_test.go proves the marker is appended iff truncated,
// S-AEM-057).
const truncationMarker = "...(truncated)"

// capturedBodyText is capturedBody's fixed Error() text — never built from
// the captured bytes, so no captured content can reach it (R-AEM-016).
const capturedBodyText = "openaicompat: provider error body captured"

// capturedBody is the unexported cause a mapped failure's error chain
// carries the retained body diagnostic through (design.md D7). Its bytes
// are reachable only via the unexported bytes() accessor; nothing renders
// them.
type capturedBody struct {
	data  []byte
	cause error
}

// Error returns capturedBody's fixed diagnostic text (R-AEM-016): it never
// reproduces the captured bytes.
func (c capturedBody) Error() string { return capturedBodyText }

// Unwrap returns the next cause in the chain (R-AEM-014, design D7):
// the rate-limit telemetry carrier (retry_metadata.go) when the
// status path attaches telemetry ahead of the capturedBody. For the
// status-without-telemetry and the non-stream-content-type paths,
// Unwrap returns nil — capturedBody is the chain's terminal link.
// errors.As traverses the chain through this link, so callers reach
// *ai.Failure, then *RateLimitTelemetry if any, then *capturedBody,
// in that order. The link is what the S-AEM-060/061/062
// sentinel-leak-proofs traverse when they assert every chain link's
// Error() is sentinel-free.
func (c capturedBody) Unwrap() error { return c.cause }

// bytes returns the retained diagnostic bytes. Unexported and never
// rendered by any Error()/%v/%+v path — the structural half of R-AEM-016's
// guarantee (design.md D7).
func (c capturedBody) bytes() []byte { return c.data }

// captureBody reads rc bounded to captureLimit+1 bytes, retains at most
// the first captureLimit bytes plus a fixed truncation marker on
// overflow, drains the unread remainder, and closes rc exactly once
// (S-AEM-056…059, design D7 finalization). The returned capturedBody's
// Unwrap() returns nil — the cause chain ends at the capturedBody for
// the status path (failure_map.go wraps the body directly when no
// telemetry is present), and the rate-limit telemetry path
// (retry_metadata.go) wraps capturedBody rather than the other way
// around. errors.As on the resulting *ai.Failure reaches the
// capturedBody via that link, and the S-AEM-060/061/062 sentinel-leak
// proofs traverse it.
//
// The truncation probe is a single io.LimitReader(rc, captureLimit+1)
// ReadAll: a body that exceeds the limit returns captureLimit+1 bytes
// (overflow → marker); a body within the limit returns < captureLimit+1
// bytes (no overflow → no marker). The slice 1 implementation used
// io.LimitReader(rc, captureLimit) and dropped the unread remainder;
// slice 2c's finalization drains that remainder into io.Discard so
// the connection can be reused / freed by the transport.
func captureBody(rc io.ReadCloser) capturedBody {
	defer func() { _ = rc.Close() }()

	// Probe read: captureLimit+1 bytes — one byte past the cap so a
	// body that exactly hits the cap reads captureLimit bytes (no
	// overflow) while a body one byte over reads captureLimit+1
	// (overflow → marker).
	probe, _ := io.ReadAll(io.LimitReader(rc, captureLimit+1))

	var data []byte
	if len(probe) > captureLimit {
		// Overflow: retain the first captureLimit bytes, append the
		// fixed truncation marker. The marker carries no payload bytes
		// (R-AEM-016 structural redaction; the marker is its own
		// constant).
		data = make([]byte, 0, captureLimit+len(truncationMarker))
		data = append(data, probe[:captureLimit]...)
		data = append(data, truncationMarker...)
	} else {
		// Within limit: retain exactly what was read. No marker.
		data = probe
	}

	// Drain the unread remainder (slow-loris guard, S-AEM-059) before
	// the deferred Close fires. The drain is silent — any error from
	// io.Copy is the network's problem, not a capture-level concern;
	// the structural invariants we care about (closed exactly once,
	// drained to EOF) are observable independently.
	_, _ = io.Copy(io.Discard, rc)

	return capturedBody{data: data}
}
