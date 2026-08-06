// AI-32.1 / AI-32.4 — the wire-side status mapper (R-AEM-001…009). This is
// the first construction site in this package that turns a wire failure
// (an HTTP status, its headers, its body) into an AI-19 ai.Failure. AI-26.6's
// policy.go refuse is a distinct, pre-existing, translation-time
// construction site this milestone neither invokes nor alters (spec.md
// Purpose, R-AEM-002).
//
// mapErrorResponse is the pure classification core (design.md D3): every
// input, the observation instant included, arrives as a parameter, so it
// carries no package-level clock and no hidden state. mapResponse is the
// thin *http.Response consumer wrapping it, so a later caller (AI-28.6)
// can call either without redesign.

package openaicompat

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// mapResponse turns resp into a constructed *ai.Failure, or nil for a 2xx
// response (R-AEM-001, S-AEM-010). observedAt is the caller-supplied
// observation instant (design.md D3) — never time.Now() read internally,
// so every Retry-After HTTP-date computation stays deterministic under
// test.
func mapResponse(resp *http.Response, observedAt time.Time) *ai.Failure {
	if resp.StatusCode/100 == 2 {
		return nil
	}
	captured := captureBody(resp.Body)
	return mapErrorResponse(resp.StatusCode, resp.Header, captured, observedAt)
}

// mapErrorResponse is the pure classification core (design.md D3): given a
// non-2xx status, its response header and its already-captured body,
// construct the AI-19 pre-stream failure R-AEM-001 requires. Every input
// arrives as a parameter — no clock global, no Config widening.
func mapErrorResponse(status int, header http.Header, captured capturedBody, observedAt time.Time) *ai.Failure {
	category, inRange := classifyStatus(status)
	statusClass := 0
	if inRange {
		statusClass = status / 100
	}

	label, _ := parseVendorErrorBody(captured.bytes())
	retryAfter := parseRetryAfter(header.Get("Retry-After"), observedAt)

	var cause error = captured
	if telemetry, ok := captureRateLimitTelemetry(header, captured); ok {
		cause = telemetry
	}

	failure, err := ai.PreStreamFailure(ai.FailureReport{
		Category:    category,
		Retryable:   retryableFor(category),
		RetryAfter:  retryAfter,
		RawLabel:    label,
		StatusClass: statusClass,
		RequestID:   header.Get("X-Request-Id"),
		Cause:       cause,
	})
	if err != nil {
		// category is always a member of classifyStatus's own closed range
		// of returns, all valid ai.FailureCategory vocabulary members, and
		// statusClass is always in [0, 5] — construction cannot fail for
		// an input this function always supplies itself. Reaching this
		// branch would be this package's own defect, not a caller's — the
		// same posture policy.go's refuse already takes for its own
		// PreStreamFailure call.
		panic("openaicompat: ai.PreStreamFailure construction failed unexpectedly: " + err.Error())
	}
	return failure
}

// retryableFor derives Retryable() from category alone (R-AEM-003): true
// for exactly rate-limit, unavailable and timeout, false for every other
// category. Shared, uniform derivation (design.md D6) — stage 2's
// categorizeStreamError (stream_failure.go) reuses this same function for
// mid-stream failures, so retryability is one rule, not two independently
// maintained copies.
func retryableFor(category ai.FailureCategory) bool {
	switch category {
	case ai.FailureCategoryRateLimit, ai.FailureCategoryUnavailable, ai.FailureCategoryTimeout:
		return true
	default:
		return false
	}
}

// classifyStatus maps status to its AI-19 failure category — R-AEM-002's
// dialect-conventional table first, then R-AEM-005's class fallback for
// every other status in the transport-legal 100–599 range — and reports
// whether status falls in that range at all.
//
// The second result feeds StatusClass reporting, never Retryable (which
// retryableFor derives from the category alone): a status outside 100–599
// yields (FailureCategoryUnknown, false), so mapErrorResponse leaves
// FailureReport.StatusClass at its zero value rather than reporting an
// out-of-range class (R-AEM-005, S-AEM-020).
func classifyStatus(status int) (ai.FailureCategory, bool) {
	switch status {
	case 401:
		return ai.FailureCategoryAuthentication, true
	case 403:
		return ai.FailureCategoryAuthorization, true
	case 400, 404, 422:
		return ai.FailureCategoryUnknown, true
	case 408, 504:
		return ai.FailureCategoryTimeout, true
	case 429:
		return ai.FailureCategoryRateLimit, true
	case 500, 502, 503:
		return ai.FailureCategoryUnavailable, true
	}

	if status < 100 || status > 599 {
		return ai.FailureCategoryUnknown, false
	}

	// R-AEM-005's class fallback for every status R-AEM-002's table does
	// not name: class 5 -> Unavailable, retryable; every other in-range
	// class (1, 2, 3, 4) -> Unknown, not retryable. Class 2 never reaches
	// here in production (mapResponse filters 2xx before calling this
	// function at all), but the fallback stays total and safe regardless.
	if status/100 == 5 {
		return ai.FailureCategoryUnavailable, true
	}
	return ai.FailureCategoryUnknown, true
}

// vendorError is the vendor error body's four-field shape (cited, E1):
// type, message, param, code — all pointers so a JSON null is
// distinguishable from an absent key, though this package only ever reads
// type and code (param and code are declared required-but-nullable by the
// pinned spec; param is parsed for shape-tolerance only and never read).
type vendorError struct {
	Type    *string `json:"type"`
	Message *string `json:"message"`
	Param   *string `json:"param"`
	Code    *string `json:"code"`
}

// label reports v's opaque vendor label: type, falling back to code when
// type is absent, empty or null (R-AEM-006).
func (v vendorError) label() (string, bool) {
	if v.Type != nil && *v.Type != "" {
		return *v.Type, true
	}
	if v.Code != nil && *v.Code != "" {
		return *v.Code, true
	}
	return "", false
}

// parseVendorErrorBody extracts the vendor's opaque label from body —
// tolerant of both the wrapped {"error":{…}} and bare {…} forms (cited,
// E1 + negative finding 6) — and reports whether one was found. Any parse
// failure — invalid JSON, empty input, or valid JSON of the wrong shape —
// yields ("", false) rather than an error: a body parse failure MUST NOT
// abort mapping, MUST NOT be reported as the category, and MUST NOT panic
// (R-AEM-004).
func parseVendorErrorBody(body []byte) (string, bool) {
	var wrapped struct {
		Error vendorError `json:"error"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		if label, ok := wrapped.Error.label(); ok {
			return label, true
		}
	}

	var bare vendorError
	if err := json.Unmarshal(body, &bare); err == nil {
		if label, ok := bare.label(); ok {
			return label, true
		}
	}

	return "", false
}

// parseRetryAfter parses value — a Retry-After header's raw text — per its
// two RFC 9110 § 10.2.3 forms (cited grammar; the header's presence on
// this vendor's failures is dialect-conventional, citations.md E3, pinned
// by fixture): delay-seconds, a non-negative integer count of seconds, or
// an HTTP-date. observedAt anchors the HTTP-date form's math (design.md
// D3).
//
// A malformed, negative or unparseable value yields an absent hint. An
// HTTP-date at or before observedAt yields ai.Delay(0) — reported,
// present, distinct from absent — never a negative duration (R-AEM-008).
func parseRetryAfter(value string, observedAt time.Time) ai.RetryDelay {
	if value == "" {
		return ai.RetryDelay{}
	}

	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		if seconds < 0 {
			return ai.RetryDelay{}
		}
		return ai.Delay(time.Duration(seconds) * time.Second)
	}

	if when, err := http.ParseTime(value); err == nil {
		delay := when.Sub(observedAt)
		if delay < 0 {
			delay = 0
		}
		return ai.Delay(delay)
	}

	return ai.RetryDelay{}
}
