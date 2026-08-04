// AI-32.5 — bounded, sanitized body capture (R-AEM-015, R-AEM-016).
//
// capturedBody is the structural guarantee behind R-AEM-016: a credential
// echoed only inside an error body can never reach Failure.Error(), because
// no error text in this package's cause chain is ever built FROM the body's
// bytes — capturedBody.Error() is a fixed string, and the bytes themselves
// are reachable only through the unexported bytes() accessor, never
// rendered by any Error()/%v/%+v path (design.md D7).

package openaicompat

import "io"

// captureLimit bounds how many body bytes captureBody retains as a
// diagnostic (R-AEM-015): the spec's own cap is 64 KiB, and this package
// chooses 8 KiB — vendor error bodies are small JSON, well under either
// bound (design.md D1).
const captureLimit = 8 << 10

// truncationMarker is the fixed ASCII suffix a capture that hit
// captureLimit carries, present on no capture that did not (R-AEM-015).
// [Slice 1 does not yet append this marker to any capture — every fixture
// this stage exercises is far under captureLimit by construction, so the
// exact truncation invariant (S-AEM-056…059) is deliberately unexercised
// here. capture.go's D7 finalization, gated on slices 2a+2b merging, is
// what proves and completes the truncate-back mechanics; see
// capture_test.go's own file comment.]
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

// Unwrap returns the next cause in the chain — nil in slice 1, since
// ErrInBandErrorFrame (the frame-path cause slice 2a adds, design.md D7)
// does not exist yet.
func (c capturedBody) Unwrap() error { return c.cause }

// bytes returns the retained diagnostic bytes. Unexported and never
// rendered by any Error()/%v/%+v path — the structural half of R-AEM-016's
// guarantee (design.md D7).
func (c capturedBody) bytes() []byte { return c.data }

// captureBody reads rc bounded to captureLimit bytes and closes it,
// returning a capturedBody carrying whatever was actually read.
//
// [Minimal for slice 1 (S-AEM-014…016 via capture_test.go, task 1.13):
// bounded by captureLimit via io.LimitReader, so it structurally cannot
// retain more than the limit, but it does not yet drain an unread
// remainder beyond the limit or append truncationMarker on overflow — that
// finalization (D7) is slice 2c's own RED-first proof
// (capture_proof_test.go, S-AEM-056…059), gated on 2a+2b merging.]
func captureBody(rc io.ReadCloser) capturedBody {
	defer func() { _ = rc.Close() }()
	data, _ := io.ReadAll(io.LimitReader(rc, captureLimit))
	return capturedBody{data: data}
}
