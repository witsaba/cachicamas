// AI-32.1 / AI-32.4 — the wire-side status mapper's RED suite (stage 1).
//
// Internal package (`package openaicompat`), matching design.md's Testing
// Strategy table for slice 1: every scenario below reaches mapResponse and
// mapErrorResponse, both unexported (design.md D4) — AI-28.6 is the only
// planned in-package caller, and this package's own established idiom
// (ambient_authority_test.go, reasoning_refusal_test.go) already tests
// unexported machinery from inside the package rather than exporting it
// only for tests to reach.
//
// Fixtures are byte-level raw HTTP responses (testdata/errormap/*.http),
// replayed via http.ReadResponse — never hand-built http.Header maps —
// so a header-name-casing claim (S-AEM-027) is proven against real wire
// bytes, not against a map this test already canonicalized by construction.
package openaicompat

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// observedAtFixed is the fixed observation instant every test below passes
// to mapResponse/mapErrorResponse (design.md D3: observedAt is a parameter,
// never a package clock) — chosen so the Retry-After HTTP-date scenarios
// (S-AEM-032/033) have a stable, human-checkable instant to compute from.
var observedAtFixed = time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)

// mustParseResponse parses raw as an HTTP response, failing the test on any
// parse error. req is always nil: per net/http's own documentation this
// only affects the "was this a HEAD request" inference, irrelevant to every
// case below.
func mustParseResponse(t *testing.T, raw string) *http.Response {
	t.Helper()
	resp, err := http.ReadResponse(bufio.NewReader(strings.NewReader(raw)), nil)
	if err != nil {
		t.Fatalf("http.ReadResponse: %v\nraw:\n%s", err, raw)
	}
	return resp
}

// mustFixtureResponse reads testdata/errormap/openai_<status>.http and
// parses it exactly like mustParseResponse (design.md D4's per-status
// fixture layout).
func mustFixtureResponse(t *testing.T, status int) *http.Response {
	t.Helper()
	path := filepath.Join("testdata", "errormap", fmt.Sprintf("openai_%d.http", status))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture %s: %v", path, err)
	}
	return mustParseResponse(t, string(raw))
}

// statusTableCase is one row of R-AEM-002's dialect-conventional table,
// pinned by its own fixture (S-AEM-003…008).
type statusTableCase struct {
	status        int
	wantCategory  ai.FailureCategory
	wantRetryable bool
	wantSentinel  error
}

// statusTableCases is every row of R-AEM-002, driven as one table test
// (S-AEM-009's own framing) so no row can be inferred from the pinned
// OpenAI specification rather than fixture-replayed (citations.md E2).
var statusTableCases = []statusTableCase{
	{401, ai.FailureCategoryAuthentication, false, ai.ErrAuthentication},
	{403, ai.FailureCategoryAuthorization, false, ai.ErrAuthorization},
	{400, ai.FailureCategoryUnknown, false, ai.ErrUnknownFailure},
	{404, ai.FailureCategoryUnknown, false, ai.ErrUnknownFailure},
	{422, ai.FailureCategoryUnknown, false, ai.ErrUnknownFailure},
	{408, ai.FailureCategoryTimeout, true, ai.ErrTimeout},
	{429, ai.FailureCategoryRateLimit, true, ai.ErrRateLimited},
	{500, ai.FailureCategoryUnavailable, true, ai.ErrUnavailable},
	{502, ai.FailureCategoryUnavailable, true, ai.ErrUnavailable},
	{503, ai.FailureCategoryUnavailable, true, ai.ErrUnavailable},
	{504, ai.FailureCategoryTimeout, true, ai.ErrTimeout},
}

// TestFailureMap_StatusCategoryTable proves R-AEM-001, R-AEM-002 and
// R-AEM-003 together (S-AEM-001…009, 011): every pinned status produces a
// pre-stream, no-partial-output failure whose StatusClass, Category and
// Retryable match the table, and whose category sentinel is reachable by
// errors.Is — never a string compare.
func TestFailureMap_StatusCategoryTable(t *testing.T) {
	for _, tc := range statusTableCases {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			resp := mustFixtureResponse(t, tc.status)
			failure := mapResponse(resp, observedAtFixed)
			if failure == nil {
				t.Fatalf("mapResponse(%d) = nil, want a constructed failure (S-AEM-001)", tc.status)
			}
			if failure.Delivery() != ai.DeliveryPreStream {
				t.Errorf("Delivery() = %v, want ai.DeliveryPreStream (R-AEM-001)", failure.Delivery())
			}
			if failure.PartialOutput() {
				t.Errorf("PartialOutput() = true, want false (R-AEM-001)")
			}
			if class, present := failure.StatusClass(); !present || class != tc.status/100 {
				t.Errorf("StatusClass() = (%d, %v), want (%d, true) (S-AEM-002)", class, present, tc.status/100)
			}
			if got := failure.Category(); got != tc.wantCategory {
				t.Errorf("Category() = %v, want %v (R-AEM-002)", got, tc.wantCategory)
			}
			if got := failure.Retryable(); got != tc.wantRetryable {
				t.Errorf("Retryable() = %v, want %v (R-AEM-003, S-AEM-011)", got, tc.wantRetryable)
			}
			if !errors.Is(failure, tc.wantSentinel) {
				t.Errorf("errors.Is(failure, %v) = false, want true (R-AEM-002)", tc.wantSentinel)
			}
		})
	}
}

// TestFailureMap_NoRowProducesUnsupportedCapabilityOrZeroCategory proves
// S-AEM-009 directly: scanning every table row, none ever names
// FailureCategoryUnsupportedCapability (AI-26.6's translation-time refusal
// alone, R-AEM-002) and none is the zero category.
func TestFailureMap_NoRowProducesUnsupportedCapabilityOrZeroCategory(t *testing.T) {
	for _, tc := range statusTableCases {
		resp := mustFixtureResponse(t, tc.status)
		failure := mapResponse(resp, observedAtFixed)
		if failure == nil {
			t.Fatalf("mapResponse(%d) = nil", tc.status)
		}
		if failure.Category() == ai.FailureCategoryUnsupportedCapability {
			t.Errorf("status %d produced FailureCategoryUnsupportedCapability, which stays AI-26.6's translation-time refusal alone (R-AEM-002)", tc.status)
		}
		if failure.Category() == 0 {
			t.Errorf("status %d produced the zero FailureCategory", tc.status)
		}
	}
}

// TestFailureMap_2xxProducesNoFailure proves S-AEM-010: a 200 response
// yields a nil failure from the mapper, never a constructed value.
func TestFailureMap_2xxProducesNoFailure(t *testing.T) {
	raw := "HTTP/1.1 200 OK\nContent-Type: application/json\n\n{\"id\":\"chatcmpl-1\"}"
	resp := mustParseResponse(t, raw)
	failure := mapResponse(resp, observedAtFixed)
	if failure != nil {
		t.Fatalf("mapResponse(200) = %v, want nil (S-AEM-010)", failure)
	}
}

// TestFailureMap_RetryableIndependentOfRetryAfterHint proves S-AEM-012 and
// S-AEM-013: Retryable() is derived from the category alone, never from
// whether a Retry-After hint was reported.
func TestFailureMap_RetryableIndependentOfRetryAfterHint(t *testing.T) {
	t.Run("429 with no Retry-After still retryable (S-AEM-012)", func(t *testing.T) {
		raw := "HTTP/1.1 429 Too Many Requests\nContent-Type: application/json\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if !failure.Retryable() {
			t.Errorf("Retryable() = false, want true")
		}
		if delay, present := failure.RetryAfter(); present {
			t.Errorf("RetryAfter() = (%v, true), want (0, false)", delay)
		}
	})

	t.Run("401 with Retry-After still terminal (S-AEM-013)", func(t *testing.T) {
		raw := "HTTP/1.1 401 Unauthorized\nRetry-After: 30\nContent-Type: application/json\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Retryable() {
			t.Errorf("Retryable() = true, want false — the hint does not set the flag")
		}
		delay, present := failure.RetryAfter()
		if !present || delay != 30*time.Second {
			t.Errorf("RetryAfter() = (%v, %v), want (30s, true)", delay, present)
		}
	})
}

// TestFailureMap_UnparseableOrAbsentBodyStillMaps proves R-AEM-004
// (S-AEM-014…016): a body that is not JSON, is empty, or is validly-shaped
// JSON of the wrong type never aborts mapping, never panics, and never lets
// a parse failure leak into the category.
func TestFailureMap_UnparseableOrAbsentBodyStillMaps(t *testing.T) {
	t.Run("non-JSON body still maps by status (S-AEM-014)", func(t *testing.T) {
		raw := "HTTP/1.1 429 Too Many Requests\nContent-Type: text/html\n\n<html>rate limited</html>"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Category() != ai.FailureCategoryRateLimit {
			t.Errorf("Category() = %v, want FailureCategoryRateLimit", failure.Category())
		}
		if !failure.Retryable() {
			t.Errorf("Retryable() = false, want true")
		}
	})

	t.Run("zero-length body still maps by status (S-AEM-015)", func(t *testing.T) {
		raw := "HTTP/1.1 503 Service Unavailable\n\n"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Category() != ai.FailureCategoryUnavailable {
			t.Errorf("Category() = %v, want FailureCategoryUnavailable", failure.Category())
		}
		if got := failure.RawLabel(); got != "" {
			t.Errorf("RawLabel() = %q, want empty", got)
		}
	})

	t.Run("right key wrong type does not panic (S-AEM-016)", func(t *testing.T) {
		raw := "HTTP/1.1 500 Internal Server Error\nContent-Type: application/json\n\n{\"error\": 12345}"
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("mapResponse panicked: %v", r)
			}
		}()
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Category() != ai.FailureCategoryUnavailable {
			t.Errorf("Category() = %v, want FailureCategoryUnavailable", failure.Category())
		}
	})
}

// TestFailureMap_UndocumentedStatusFallback proves R-AEM-005 (S-AEM-017…
// 020). 418/599/302 are valid three-digit statuses absent from R-AEM-002's
// table, so no pinning fixture is owed (R-AEM-018's obligation binds only
// the table's own dialect-conventional rows) — realistic wire bytes still
// exercise them. 0 and 999 are not realistic wire statuses at all; the
// scenario calls mapErrorResponse directly, exactly as R-AEM-005 states the
// rule ("a status outside 100–599").
func TestFailureMap_UndocumentedStatusFallback(t *testing.T) {
	t.Run("418 class 4 -> Unknown, not retryable (S-AEM-017)", func(t *testing.T) {
		raw := "HTTP/1.1 418 I'm a teapot\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Category() != ai.FailureCategoryUnknown {
			t.Errorf("Category() = %v, want FailureCategoryUnknown", failure.Category())
		}
		if failure.Retryable() {
			t.Errorf("Retryable() = true, want false")
		}
		if class, present := failure.StatusClass(); !present || class != 4 {
			t.Errorf("StatusClass() = (%d, %v), want (4, true)", class, present)
		}
	})

	t.Run("599 class 5 -> Unavailable, retryable (S-AEM-018)", func(t *testing.T) {
		raw := "HTTP/1.1 599 Network Connect Timeout Error\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Category() != ai.FailureCategoryUnavailable {
			t.Errorf("Category() = %v, want FailureCategoryUnavailable", failure.Category())
		}
		if !failure.Retryable() {
			t.Errorf("Retryable() = false, want true")
		}
		if class, present := failure.StatusClass(); !present || class != 5 {
			t.Errorf("StatusClass() = (%d, %v), want (5, true)", class, present)
		}
	})

	t.Run("302 class 3 -> Unknown (S-AEM-019)", func(t *testing.T) {
		raw := "HTTP/1.1 302 Found\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if failure.Category() != ai.FailureCategoryUnknown {
			t.Errorf("Category() = %v, want FailureCategoryUnknown", failure.Category())
		}
		if class, present := failure.StatusClass(); !present || class != 3 {
			t.Errorf("StatusClass() = (%d, %v), want (3, true)", class, present)
		}
	})

	t.Run("synthetic status 0 and 999 -> StatusClass (0, false) (S-AEM-020)", func(t *testing.T) {
		for _, status := range []int{0, 999} {
			failure := mapErrorResponse(status, http.Header{}, captureBody(io.NopCloser(strings.NewReader("{}"))), observedAtFixed)
			if failure == nil {
				t.Fatalf("mapErrorResponse(%d) = nil, want a constructed failure with no construction error", status)
			}
			if failure.Category() != ai.FailureCategoryUnknown {
				t.Errorf("status %d: Category() = %v, want FailureCategoryUnknown", status, failure.Category())
			}
			if class, present := failure.StatusClass(); present || class != 0 {
				t.Errorf("status %d: StatusClass() = (%d, %v), want (0, false)", status, class, present)
			}
		}
	})
}

// TestFailureMap_VendorBodyToleranceAndLabelOpacity proves R-AEM-006
// (S-AEM-021…025).
func TestFailureMap_VendorBodyToleranceAndLabelOpacity(t *testing.T) {
	t.Run("wrapped body opaque label survives, category unaffected (S-AEM-021)", func(t *testing.T) {
		raw := `HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{"error":{"type":"provider_specific_throttle","message":"slow down","param":null,"code":null}}`
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RawLabel(); got != "provider_specific_throttle" {
			t.Errorf("RawLabel() = %q, want %q", got, "provider_specific_throttle")
		}
		if failure.Category() != ai.FailureCategoryRateLimit {
			t.Errorf("Category() = %v, want FailureCategoryRateLimit", failure.Category())
		}
	})

	t.Run("bare body same fields, identical label (S-AEM-022)", func(t *testing.T) {
		raw := `HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{"type":"provider_specific_throttle","message":"slow down","param":null,"code":null}`
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RawLabel(); got != "provider_specific_throttle" {
			t.Errorf("RawLabel() = %q, want %q", got, "provider_specific_throttle")
		}
	})

	t.Run("empty type falls back to code (S-AEM-023)", func(t *testing.T) {
		raw := `HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":{"type":"","message":"quota","param":null,"code":"insufficient_quota"}}`
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RawLabel(); got != "insufficient_quota" {
			t.Errorf("RawLabel() = %q, want %q", got, "insufficient_quota")
		}
	})

	t.Run("different labels, identical category (S-AEM-024)", func(t *testing.T) {
		rawA := `HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":{"type":"invalid_request_error","message":"bad request","param":null,"code":null}}`
		rawB := `HTTP/1.1 400 Bad Request
Content-Type: application/json

{"error":{"type":"some_vendor_invented_type","message":"bad request","param":null,"code":null}}`
		failureA := mapResponse(mustParseResponse(t, rawA), observedAtFixed)
		failureB := mapResponse(mustParseResponse(t, rawB), observedAtFixed)
		if failureA == nil || failureB == nil {
			t.Fatal("mapResponse = nil")
		}
		if failureA.Category() != ai.FailureCategoryUnknown || failureB.Category() != ai.FailureCategoryUnknown {
			t.Fatalf("Category() = (%v, %v), want both FailureCategoryUnknown", failureA.Category(), failureB.Category())
		}
		if failureA.RawLabel() == failureB.RawLabel() {
			t.Errorf("RawLabel() identical (%q) across two different vendor types — the fixture should distinguish them", failureA.RawLabel())
		}
	})

	t.Run("message never reaches Error() (S-AEM-025)", func(t *testing.T) {
		raw := `HTTP/1.1 404 Not Found
Content-Type: application/json

{"error":{"type":"invalid_request_error","message":"the model gpt-x does not exist","param":"model","code":null}}`
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		const needle = "the model gpt-x does not exist"
		if strings.Contains(failure.Error(), needle) {
			t.Errorf("Error() = %q leaks the vendor message", failure.Error())
		}
		if strings.Contains(fmt.Sprintf("%v", failure), needle) {
			t.Errorf("%%v rendering leaks the vendor message")
		}
	})
}

// TestFailureMap_RequestIDBestEffort proves R-AEM-007 (S-AEM-026…029).
func TestFailureMap_RequestIDBestEffort(t *testing.T) {
	t.Run("lowercase header captured (S-AEM-026)", func(t *testing.T) {
		raw := "HTTP/1.1 500 Internal Server Error\nx-request-id: req_abc123\nContent-Type: application/json\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RequestID(); got != "req_abc123" {
			t.Errorf("RequestID() = %q, want %q", got, "req_abc123")
		}
	})

	t.Run("mixed-case header spelling still captured (S-AEM-027)", func(t *testing.T) {
		raw := "HTTP/1.1 500 Internal Server Error\nX-Request-Id: req_abc123\nContent-Type: application/json\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RequestID(); got != "req_abc123" {
			t.Errorf("RequestID() = %q, want %q", got, "req_abc123")
		}
	})

	t.Run("absent header tolerated (S-AEM-028)", func(t *testing.T) {
		raw := "HTTP/1.1 500 Internal Server Error\nContent-Type: application/json\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RequestID(); got != "" {
			t.Errorf("RequestID() = %q, want empty", got)
		}
	})

	t.Run("over-long value dropped whole, not truncated (S-AEM-029)", func(t *testing.T) {
		overlong := strings.Repeat("A", 64) + "Z" // 65 bytes: prefix would be all 'A's if truncated
		raw := "HTTP/1.1 500 Internal Server Error\nX-Request-Id: " + overlong + "\nContent-Type: application/json\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		if got := failure.RequestID(); got != "" {
			t.Errorf("RequestID() = %q, want exactly empty — a 64-byte prefix surviving would also be a defect", got)
		}
	})
}

// TestFailureMap_RetryAfter proves R-AEM-008 (S-AEM-030…035).
func TestFailureMap_RetryAfter(t *testing.T) {
	t.Run("delay-seconds form (S-AEM-030)", func(t *testing.T) {
		raw := "HTTP/1.1 429 Too Many Requests\nRetry-After: 30\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		delay, present := failure.RetryAfter()
		if !present || delay != 30*time.Second {
			t.Errorf("RetryAfter() = (%v, %v), want (30s, true)", delay, present)
		}
	})

	t.Run("delay-seconds zero is a reported immediate retry (S-AEM-031)", func(t *testing.T) {
		raw := "HTTP/1.1 429 Too Many Requests\nRetry-After: 0\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		delay, present := failure.RetryAfter()
		if !present || delay != 0 {
			t.Errorf("RetryAfter() = (%v, %v), want (0, true) — a reported zero, not absent", delay, present)
		}
	})

	t.Run("HTTP-date in the future (S-AEM-032)", func(t *testing.T) {
		raw := "HTTP/1.1 503 Service Unavailable\nRetry-After: Wed, 21 Oct 2026 07:28:00 GMT\n\n{}"
		observedAt := time.Date(2026, 10, 21, 7, 27, 0, 0, time.UTC)
		failure := mapResponse(mustParseResponse(t, raw), observedAt)
		delay, present := failure.RetryAfter()
		if !present || delay != 60*time.Second {
			t.Errorf("RetryAfter() = (%v, %v), want (60s, true)", delay, present)
		}
	})

	t.Run("HTTP-date at or before observedAt yields a reported zero, never negative (S-AEM-033)", func(t *testing.T) {
		raw := "HTTP/1.1 503 Service Unavailable\nRetry-After: Wed, 21 Oct 2026 07:28:00 GMT\n\n{}"
		observedAt := time.Date(2026, 10, 21, 7, 29, 0, 0, time.UTC)
		failure := mapResponse(mustParseResponse(t, raw), observedAt)
		delay, present := failure.RetryAfter()
		if !present || delay != 0 {
			t.Errorf("RetryAfter() = (%v, %v), want (0, true), never negative", delay, present)
		}
	})

	t.Run("malformed or negative values yield an absent hint (S-AEM-034)", func(t *testing.T) {
		for _, value := range []string{"soon", "-5"} {
			raw := "HTTP/1.1 429 Too Many Requests\nRetry-After: " + value + "\n\n{}"
			failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
			if failure == nil {
				t.Fatalf("Retry-After %q: mapResponse = nil", value)
			}
			if _, present := failure.RetryAfter(); present {
				t.Errorf("Retry-After %q: RetryAfter() present = true, want false", value)
			}
			if failure.Category() != ai.FailureCategoryRateLimit {
				t.Errorf("Retry-After %q: Category() = %v, want FailureCategoryRateLimit still status-derived", value, failure.Category())
			}
		}
	})

	t.Run("header entirely absent (S-AEM-035)", func(t *testing.T) {
		raw := "HTTP/1.1 429 Too Many Requests\n\n{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if _, present := failure.RetryAfter(); present {
			t.Errorf("RetryAfter() present = true, want false, distinguishable from S-AEM-031's reported zero")
		}
	})
}
