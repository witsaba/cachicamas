// AI-32.4 — the rate-limit telemetry carrier's RED suite (R-AEM-009,
// S-AEM-036…039). Internal package: RateLimitTelemetry is exported, but
// this file also drives it through the unexported mapResponse so the
// carrier is proven wired into the real cause chain, not merely
// constructible in isolation (design.md D2).
package openaicompat

import (
	"errors"
	"strings"
	"testing"
)

// TestRetryMetadata_AllThreeHeadersIndividuallyAddressable proves
// S-AEM-036: errors.As retrieves a carrier reporting all three values
// individually, not as one concatenated string.
func TestRetryMetadata_AllThreeHeadersIndividuallyAddressable(t *testing.T) {
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"Content-Type: application/json\n" +
		"X-Ratelimit-Limit-Requests: 3000\n" +
		"X-Ratelimit-Remaining-Requests: 2999\n" +
		"X-Ratelimit-Reset-Requests: 20ms\n" +
		"\n" +
		`{"error":{"type":"rate_limit_error","message":"slow down","param":null,"code":null}}`
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatalf("errors.As(failure, &telemetry) = false, want a *RateLimitTelemetry reachable through the cause chain")
	}
	if telemetry.LimitRequests != "3000" {
		t.Errorf("LimitRequests = %q, want %q", telemetry.LimitRequests, "3000")
	}
	if telemetry.RemainingRequests != "2999" {
		t.Errorf("RemainingRequests = %q, want %q", telemetry.RemainingRequests, "2999")
	}
	if telemetry.ResetRequests != "20ms" {
		t.Errorf("ResetRequests = %q, want %q", telemetry.ResetRequests, "20ms")
	}
}

// TestRetryMetadata_PartialSubsetTolerated proves S-AEM-037: only one
// header present still yields a retrievable carrier, the other two fields
// empty (absent, not an error).
func TestRetryMetadata_PartialSubsetTolerated(t *testing.T) {
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"X-Ratelimit-Remaining-Requests: 0\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatalf("errors.As(failure, &telemetry) = false, want a carrier even with only one header present")
	}
	if telemetry.RemainingRequests != "0" {
		t.Errorf("RemainingRequests = %q, want %q", telemetry.RemainingRequests, "0")
	}
	if telemetry.LimitRequests != "" {
		t.Errorf("LimitRequests = %q, want empty (absent)", telemetry.LimitRequests)
	}
	if telemetry.ResetRequests != "" {
		t.Errorf("ResetRequests = %q, want empty (absent)", telemetry.ResetRequests)
	}
}

// TestRetryMetadata_CredentialHeaderNeverCaptured proves S-AEM-038: a
// credential-bearing header alongside the telemetry headers never reaches
// any rendering of the carrier or the failure — the allowlist, not a
// blocklist, is what keeps it out.
//
// The planted value is deliberately short enough (17 bytes after "sk-")
// that it does NOT match credential_scan_test.go's own
// sk-[A-Za-z0-9_-]{20,} pattern — this file is already internal package
// openaicompat, outside that guard's external-package scope, but the
// literal is kept sub-threshold anyway so a future refactor that widens
// the guard's scope cannot make this file collateral damage.
func TestRetryMetadata_CredentialHeaderNeverCaptured(t *testing.T) {
	const planted = "sk-planted-AEM038"
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"Authorization: Bearer " + planted + "\n" +
		"X-Ratelimit-Remaining-Requests: 5\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	if strings.Contains(failure.Error(), planted) {
		t.Fatalf("Failure.Error() = %q leaks the planted credential", failure.Error())
	}

	// The fixture carries a telemetry header, so the carrier MUST be
	// reachable — a bare `if errors.As` here passed vacuously when a
	// verify-phase mutation probe disabled telemetry capture entirely
	// (verify finding W2). The premise is asserted, then the leak checks.
	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatal("errors.As found no *RateLimitTelemetry in the chain, want one — the fixture carries X-Ratelimit-Remaining-Requests")
	}
	if strings.Contains(telemetry.Error(), planted) {
		t.Errorf("RateLimitTelemetry.Error() = %q leaks the planted credential", telemetry.Error())
	}
	if telemetry.LimitRequests == planted || telemetry.RemainingRequests == planted || telemetry.ResetRequests == planted {
		t.Errorf("planted credential captured into an allowlisted field: %+v", telemetry)
	}
}

// TestRetryMetadata_FailureErrorTextIsFixedAndClean proves S-AEM-039:
// with full telemetry attached, Failure.Error() equals exactly
// "provider failure: rate_limit" — no header name or value included. Exact
// equality is the strongest form of "contains no header name or value":
// any leaked byte would already break it.
func TestRetryMetadata_FailureErrorTextIsFixedAndClean(t *testing.T) {
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"X-Ratelimit-Limit-Requests: 3000\n" +
		"X-Ratelimit-Remaining-Requests: 0\n" +
		"X-Ratelimit-Reset-Requests: 20s\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}
	const want = "provider failure: rate_limit"
	if got := failure.Error(); got != want {
		t.Fatalf("Failure.Error() = %q, want exactly %q", got, want)
	}
}
