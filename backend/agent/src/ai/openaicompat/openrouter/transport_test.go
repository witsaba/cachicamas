package openrouter

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// capturingTransport is a test-only http.RoundTripper substitute that
// observes what the wrapper-owned transport hands downstream. It mirrors
// the existing openaicompat.stubTransport (client_test.go:16-32) in
// shape but lives here because openaicompat's own stub is internal to
// its package — this one needs to sit inside openrouter's tests so the
// attributionRoundTripper's own RoundTrip is the one exercised.
type capturingTransport struct {
	requests []*http.Request
}

func (c *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, req)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

// attributionProbeRequest builds a minimal outbound request whose only
// payload is the caller's pre-existing header set. The RoundTripper's
// own observation is what the test asserts on.
func attributionProbeRequest(t *testing.T, header http.Header) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.invalid/v1", nil)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v, want nil", err)
	}
	if header != nil {
		req.Header = header
	}
	return req
}

// TestAttributionRoundTripper_AllNonEmptyHeaders_AllObservedOnOutboundRequest
// covers R-OR-02 sub-scenario 1: every configured attribution value
// non-empty produces its expected header (and the X-Title alias) on the
// outbound request observed by the wrapped base RoundTripper.
func TestAttributionRoundTripper_AllNonEmptyHeaders_AllObservedOnOutboundRequest(t *testing.T) {
	t.Parallel()

	const (
		wantReferer   = "https://example.test/app"
		wantXTitle    = "cachicamas-ai-cli"
		wantXCategory = "productivity,coding"
	)

	base := &capturingTransport{}
	rt := attributionRoundTripper{
		base:        base,
		referer:     wantReferer,
		xTitle:      wantXTitle,
		xCategories: wantXCategory,
	}

	inbound := attributionProbeRequest(t, nil)
	if _, err := rt.RoundTrip(inbound); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil", err)
	}

	if got := len(base.requests); got != 1 {
		t.Fatalf("base observed %d request(s), want exactly 1", got)
	}
	outbound := base.requests[0]

	if got := outbound.Header.Get("HTTP-Referer"); got != wantReferer {
		t.Errorf("HTTP-Referer = %q, want %q (R-OR-02)", got, wantReferer)
	}
	if got := outbound.Header.Get("X-OpenRouter-Title"); got != wantXTitle {
		t.Errorf("X-OpenRouter-Title = %q, want %q (R-OR-02)", got, wantXTitle)
	}
	if got := outbound.Header.Get("X-Title"); got != wantXTitle {
		t.Errorf("X-Title (alias) = %q, want %q (R-OR-02)", got, wantXTitle)
	}
	if got := outbound.Header.Get("X-OpenRouter-Categories"); got != wantXCategory {
		t.Errorf("X-OpenRouter-Categories = %q, want %q (R-OR-02)", got, wantXCategory)
	}
}

// TestAttributionRoundTripper_AllEmptyHeaders_AllSuppressed covers
// R-OR-02 sub-scenario 2: every configured attribution value empty
// suppresses its header — never an empty-valued header, never an empty
// X-Title without its X-OpenRouter-Title sibling.
func TestAttributionRoundTripper_AllEmptyHeaders_AllSuppressed(t *testing.T) {
	t.Parallel()

	base := &capturingTransport{}
	rt := attributionRoundTripper{
		base:        base,
		referer:     "",
		xTitle:      "",
		xCategories: "",
	}

	inbound := attributionProbeRequest(t, nil)
	if _, err := rt.RoundTrip(inbound); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil", err)
	}
	outbound := base.requests[0]

	for _, name := range []string{
		"HTTP-Referer",
		"X-OpenRouter-Title",
		"X-Title",
		"X-OpenRouter-Categories",
	} {
		if got, present := outbound.Header[name]; present {
			t.Errorf("%s present on outbound header = %v, want absent (R-OR-02 sub-scenario 2)", name, got)
		}
	}
}

// TestAttributionRoundTripper_DoesNotMutateInboundRequestHeaders covers
// the http.Header.Clone() invariant: the inbound req's header must not
// see the wrapper's mutation. A wrapper that wrote to req.Header
// directly would alias the caller's map (http.Header is a map[string]
// []string) and either mutate it or race with it.
func TestAttributionRoundTripper_DoesNotMutateInboundRequestHeaders(t *testing.T) {
	t.Parallel()

	base := &capturingTransport{}
	rt := attributionRoundTripper{
		base:        base,
		referer:     "https://example.test/app",
		xTitle:      "cachicamas-ai-cli",
		xCategories: "productivity",
	}

	inbound := attributionProbeRequest(t, nil)
	if _, err := rt.RoundTrip(inbound); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil", err)
	}

	for _, name := range []string{
		"HTTP-Referer",
		"X-OpenRouter-Title",
		"X-Title",
		"X-OpenRouter-Categories",
	} {
		if got, present := inbound.Header[name]; present {
			t.Errorf("inbound request header %s = %v, want absent — RoundTrip must not alias the caller's header map (R-OR-02 mutation safety)", name, got)
		}
	}
}

// TestAttributionRoundTripper_PreservesExistingHeaders covers the
// composition rule: a header the inbound request already carries — the
// Authorization header openaicompat itself sets, or any caller-supplied
// header — survives the wrapper's mutation unchanged.
func TestAttributionRoundTripper_PreservesExistingHeaders(t *testing.T) {
	t.Parallel()

	const wantAuth = "Bearer super-secret-token"

	base := &capturingTransport{}
	rt := attributionRoundTripper{
		base:        base,
		referer:     "https://example.test/app",
		xTitle:      "cachicamas-ai-cli",
		xCategories: "productivity",
	}

	inbound := attributionProbeRequest(t, nil)
	inbound.Header.Set("Authorization", wantAuth)
	inbound.Header.Set("Content-Type", "application/json")

	if _, err := rt.RoundTrip(inbound); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil", err)
	}
	outbound := base.requests[0]

	if got := outbound.Header.Get("Authorization"); got != wantAuth {
		t.Errorf("Authorization = %q, want %q — wrapper must not overwrite openaicompat's own Authorization header", got, wantAuth)
	}
	if got := outbound.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q — wrapper must not overwrite the caller's existing Content-Type", got, "application/json")
	}
}

// TestAttributionRoundTripper_AppliesAttributionAcrossMultipleRequests
// covers the round-tripper's serial reuse: every outbound request the
// wrapper sees carries the attribution headers, never the first one
// only.
func TestAttributionRoundTripper_AppliesAttributionAcrossMultipleRequests(t *testing.T) {
	t.Parallel()

	base := &capturingTransport{}
	rt := attributionRoundTripper{
		base:        base,
		referer:     "https://example.test/app",
		xTitle:      "cachicamas-ai-cli",
		xCategories: "productivity",
	}

	for i := 0; i < 3; i++ {
		inbound := attributionProbeRequest(t, nil)
		if _, err := rt.RoundTrip(inbound); err != nil {
			t.Fatalf("RoundTrip() call %d error = %v, want nil", i+1, err)
		}
	}

	if got := len(base.requests); got != 3 {
		t.Fatalf("base observed %d request(s), want 3", got)
	}
	for i, req := range base.requests {
		if got := req.Header.Get("HTTP-Referer"); got != "https://example.test/app" {
			t.Errorf("request %d HTTP-Referer = %q, want the configured value", i+1, got)
		}
		if got := req.Header.Get("X-OpenRouter-Title"); got != "cachicamas-ai-cli" {
			t.Errorf("request %d X-OpenRouter-Title = %q, want the configured value", i+1, got)
		}
		if got := req.Header.Get("X-Title"); got != "cachicamas-ai-cli" {
			t.Errorf("request %d X-Title = %q, want the configured value", i+1, got)
		}
		if got := req.Header.Get("X-OpenRouter-Categories"); got != "productivity" {
			t.Errorf("request %d X-OpenRouter-Categories = %q, want the configured value", i+1, got)
		}
	}
}

// TestAttributionRoundTripper_PartialEmptyHeaders_OnlyNonEmptySet covers
// the partial-suppression rule: when some attribution values are empty
// and others are not, only the non-empty ones produce headers. This is
// the per-field shape of R-OR-02 sub-scenario 2 — neither all-or-nothing
// nor every-field-empty-only.
func TestAttributionRoundTripper_PartialEmptyHeaders_OnlyNonEmptySet(t *testing.T) {
	t.Parallel()

	base := &capturingTransport{}
	rt := attributionRoundTripper{
		base:        base,
		referer:     "",         // suppressed
		xTitle:      "nonempty", // set, plus alias
		xCategories: "",         // suppressed
	}

	inbound := attributionProbeRequest(t, nil)
	if _, err := rt.RoundTrip(inbound); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil", err)
	}
	outbound := base.requests[0]

	if got, present := outbound.Header["HTTP-Referer"]; present {
		t.Errorf("HTTP-Referer present = %v, want absent (empty referer)", got)
	}
	if got := outbound.Header.Get("X-OpenRouter-Title"); got != "nonempty" {
		t.Errorf("X-OpenRouter-Title = %q, want %q", got, "nonempty")
	}
	if got := outbound.Header.Get("X-Title"); got != "nonempty" {
		t.Errorf("X-Title = %q, want %q", got, "nonempty")
	}
	if got, present := outbound.Header["X-OpenRouter-Categories"]; present {
		t.Errorf("X-OpenRouter-Categories present = %v, want absent (empty categories)", got)
	}
}