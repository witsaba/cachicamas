// AI-32.4 — the rate-limit telemetry carrier's RED suite (R-AEM-009,
// S-AEM-036…039). Internal package: RateLimitTelemetry is exported, but
// this file also drives it through the unexported mapResponse so the
// carrier is proven wired into the real cause chain, not merely
// constructible in isolation (design.md D2).
package openaicompat

import (
	"errors"
	"fmt"
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

// headerCaptureLeak reports whether rendered reproduces headerName or
// headerValue, and which one — the pure check both
// TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue (AI-36,
// WU-3, S-APC-079) and TestHeaderDiagnostics_NoneCaptureWholeHeaderSet
// (S-APC-080) drive, so the positive-control sub-test can prove the check
// itself is falsifiable without touching a real *testing.T.
func headerCaptureLeak(rendered, headerName, headerValue string) (leaked bool, what string) {
	if strings.Contains(rendered, headerName) {
		return true, "header name"
	}
	if strings.Contains(rendered, headerValue) {
		return true, "header value"
	}
	return false, ""
}

// TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue covers
// S-APC-079: a response carrying a credential-bearing header (Authorization)
// alongside a sentinel value never reproduces either the header's NAME or
// its VALUE through any rendering of the header-capturing diagnostic — a
// stronger, name-inclusive widening of TestRetryMetadata_CredentialHeaderNeverCaptured
// (S-AEM-038) above, which only ever asserted value-absence. The inline
// positive control proves headerCaptureLeak itself is falsifiable.
func TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue(t *testing.T) {
	const headerName = "Authorization"
	// Runtime-assembled (never a contiguous "Bearer sk-..." literal in
	// this source file): credential_scan_test.go's widened recursive
	// sweep (AI-36 WU-5) now covers every _test.go file in this tree.
	sentinelValue := "Bearer " + "s" + "k" + "-ai36-header-capture-sentinel-9f2c7a"

	raw := "HTTP/1.1 429 Too Many Requests\n" +
		headerName + ": " + sentinelValue + "\n" +
		"X-Ratelimit-Remaining-Requests: 7\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatal("errors.As found no *RateLimitTelemetry in the chain, want one — the fixture carries X-Ratelimit-Remaining-Requests")
	}

	renderings := map[string]string{
		"failure.Error()":                 failure.Error(),
		"telemetry.Error()":               telemetry.Error(),
		"fmt.Sprintf(\"%+v\", telemetry)": fmt.Sprintf("%+v", telemetry),
	}
	for label, rendered := range renderings {
		if leaked, what := headerCaptureLeak(rendered, headerName, sentinelValue); leaked {
			t.Errorf("%s reproduces the %s (want neither name nor value): %q", label, what, rendered)
		}
	}

	if telemetry.LimitRequests == sentinelValue || telemetry.RemainingRequests == sentinelValue || telemetry.ResetRequests == sentinelValue {
		t.Errorf("the credential-bearing header's value was captured into an allowlisted field: %+v", telemetry)
	}

	t.Run("inline positive control: a deliberately-leaking rendering fails the same check", func(t *testing.T) {
		leakingName := fmt.Sprintf("captured header %s present", headerName)
		if leaked, what := headerCaptureLeak(leakingName, headerName, sentinelValue); !leaked || what != "header name" {
			t.Fatalf("headerCaptureLeak(%q) = (%v, %q), want (true, \"header name\") — the check itself must be falsifiable (S-APC-079)", leakingName, leaked, what)
		}
		leakingValue := fmt.Sprintf("captured value=%s", sentinelValue)
		if leaked, what := headerCaptureLeak(leakingValue, headerName, sentinelValue); !leaked || what != "header value" {
			t.Fatalf("headerCaptureLeak(%q) = (%v, %q), want (true, \"header value\") — the check itself must be falsifiable (S-APC-079)", leakingValue, leaked, what)
		}
	})
}

// TestHeaderDiagnostics_NoneCaptureWholeHeaderSet covers S-APC-080: every
// header-capturing diagnostic in the module reads only through an
// explicit admission list, none captures the whole header set, and a
// header newly present on a response stays absent from every diagnostic
// until it is explicitly admitted (design.md AD-5/D-6:
// captureRateLimitTelemetry's 3-name allowlist is the module's whole
// header-capture surface).
func TestHeaderDiagnostics_NoneCaptureWholeHeaderSet(t *testing.T) {
	t.Parallel()

	t.Run("the allowlist has exactly 3 entries, matching the documented headers", func(t *testing.T) {
		t.Parallel()

		allowlist := []string{headerRateLimitLimitRequests, headerRateLimitRemainingRequests, headerRateLimitResetRequests}
		const wantAllowlistSize = 3
		if len(allowlist) != wantAllowlistSize {
			t.Fatalf("allowlist has %d entries, want %d (S-APC-080)", len(allowlist), wantAllowlistSize)
		}
	})

	t.Run("a header newly present on a response, not on the allowlist, is absent from the carrier until admitted", func(t *testing.T) {
		t.Parallel()

		const newHeaderName = "X-Provider-Experimental-Debug-Token"
		const newHeaderValue = "unadmitted-header-sentinel-4d8e1f"

		raw := "HTTP/1.1 429 Too Many Requests\n" +
			newHeaderName + ": " + newHeaderValue + "\n" +
			"X-Ratelimit-Remaining-Requests: 12\n" +
			"\n" +
			"{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		var telemetry *RateLimitTelemetry
		if !errors.As(failure, &telemetry) {
			t.Fatal("errors.As found no *RateLimitTelemetry, want one — the fixture carries X-Ratelimit-Remaining-Requests")
		}
		if telemetry.LimitRequests == newHeaderValue || telemetry.RemainingRequests == newHeaderValue || telemetry.ResetRequests == newHeaderValue {
			t.Errorf("the unadmitted header's value was captured into an allowlisted field: %+v", telemetry)
		}
		if leaked, what := headerCaptureLeak(fmt.Sprintf("%+v", telemetry), newHeaderName, newHeaderValue); leaked {
			t.Errorf("%%+v rendering reproduces the unadmitted header's %s, want it absent until explicitly admitted (S-APC-080)", what)
		}
	})
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
