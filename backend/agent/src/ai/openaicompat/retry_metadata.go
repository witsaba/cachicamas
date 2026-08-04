// AI-32.4 — the rate-limit telemetry carrier (R-AEM-009).
//
// The three headers below are undocumented by the pinned OpenAI
// specification (dialect-conventional, cited negative citations.md E3:
// x-ratelimit -> 0 hits) and MUST be pinned by fixture — retry_metadata_test.go
// and failure_map_test.go replay them from raw HTTP bytes, never from a
// hand-built http.Header map.

package openaicompat

import "net/http"

// rateLimitTelemetryText is RateLimitTelemetry's fixed Error() text: no
// header name or value is ever rendered through it (S-AEM-038/039).
const rateLimitTelemetryText = "openaicompat: rate-limit telemetry attached"

// rateLimitHeaderAllowlist is the fixed, exhaustive set of headers
// captureRateLimitTelemetry reads (design.md D2): exactly the three names
// S-AEM-036/037 pin. A header outside this set is never captured, so a
// credential-bearing header cannot enter the carrier by accident
// (R-AEM-009) — allowlist, not a blocklist.
const (
	headerRateLimitLimitRequests     = "X-Ratelimit-Limit-Requests"
	headerRateLimitRemainingRequests = "X-Ratelimit-Remaining-Requests"
	headerRateLimitResetRequests     = "X-Ratelimit-Reset-Requests"
)

// RateLimitTelemetry is the adapter-local, exported carrier a mapped
// failure's cause chain exposes rate-limit telemetry through, reachable
// with errors.As (R-AEM-009). Its three fields are individually
// addressable strings — never one concatenated blob — and an empty field
// means that header was absent; any subset may be empty.
//
// It is adapter-local, never a widening of ai.FailureReport or ai.Failure:
// package ai's own closed vocabulary stays exactly as AI-19 shipped it
// (design.md D2).
type RateLimitTelemetry struct {
	// LimitRequests is the x-ratelimit-limit-requests header's raw value,
	// or empty when the header was absent.
	LimitRequests string
	// RemainingRequests is the x-ratelimit-remaining-requests header's raw
	// value, or empty when the header was absent.
	RemainingRequests string
	// ResetRequests is the x-ratelimit-reset-requests header's raw value,
	// or empty when the header was absent.
	ResetRequests string

	cause error
}

// Error returns RateLimitTelemetry's fixed diagnostic text. It never
// reproduces a captured header's name or value (S-AEM-038/039).
func (t *RateLimitTelemetry) Error() string { return rateLimitTelemetryText }

// Unwrap returns the next cause in the chain — capturedBody, wrapping the
// retained error-body diagnostic (design.md D7's cause chain: Failure ->
// RateLimitTelemetry -> capturedBody).
func (t *RateLimitTelemetry) Unwrap() error { return t.cause }

// captureRateLimitTelemetry reads header's allowlisted rate-limit fields
// and reports whether at least one was present. cause becomes the returned
// carrier's own Unwrap() target (design.md D7's cause chain), so the
// caller supplies the capturedBody the telemetry wraps.
//
// header.Get is net/http's own canonicalized lookup (textproto's MIME
// header canonicalization), which is what makes header-name matching
// case-insensitive without this function doing anything special (R-AEM-009
// shares this posture with R-AEM-007's request-id capture).
func captureRateLimitTelemetry(header http.Header, cause error) (*RateLimitTelemetry, bool) {
	limit := header.Get(headerRateLimitLimitRequests)
	remaining := header.Get(headerRateLimitRemainingRequests)
	reset := header.Get(headerRateLimitResetRequests)

	if limit == "" && remaining == "" && reset == "" {
		return nil, false
	}

	return &RateLimitTelemetry{
		LimitRequests:     limit,
		RemainingRequests: remaining,
		ResetRequests:     reset,
		cause:             cause,
	}, true
}
