package openaicompat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// stubTransport is a test-only http.RoundTripper substitute that observes
// what the adapter would send without a network (the spec's "stub
// transport"). It is how "the injected value was actually used" is proven,
// in place of reading a field (R-APC-001).
type stubTransport struct {
	requests []*http.Request
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.requests = append(s.requests, req)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}

// driveOneRequest builds and sends exactly one request through c, using
// c.httpClient directly — AI-25 ships no exported entry point that does
// this; the stub-transport observation is how every AI-25.1 scenario
// proves an injected value was used.
func driveOneRequest(t *testing.T, c *Client) {
	t.Helper()
	req, err := c.newRequest(context.Background(), "chat", "completions")
	if err != nil {
		t.Fatalf("newRequest() error = %v, want nil", err)
	}
	if _, err := c.httpClient.Do(req); err != nil {
		t.Fatalf("httpClient.Do() error = %v, want nil", err)
	}
}

// TestNew_InjectedClientObservesOutboundRequest covers S-APC-001, S-APC-002
// and S-APC-003: the injected client is the client that carries traffic,
// the observed request's scheme/host/path derive from the injected
// endpoint, and the injected credential is present on it.
func TestNew_InjectedClientObservesOutboundRequest(t *testing.T) {
	t.Parallel()

	stub := &stubTransport{}
	c, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("test-token"),
		HTTPClient: &http.Client{Transport: stub},
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	driveOneRequest(t, c)

	if len(stub.requests) != 1 {
		t.Fatalf("stub observed %d requests, want exactly 1 (S-APC-001)", len(stub.requests))
	}

	got := stub.requests[0]
	if got.URL.Scheme != "http" {
		t.Errorf("scheme = %q, want %q (S-APC-002)", got.URL.Scheme, "http")
	}
	if got.URL.Host != "example.invalid" {
		t.Errorf("host = %q, want %q (S-APC-002)", got.URL.Host, "example.invalid")
	}
	if !strings.HasPrefix(got.URL.Path, "/v1") {
		t.Errorf("path = %q, want prefix %q (S-APC-002)", got.URL.Path, "/v1")
	}

	const wantAuth = "Bearer test-token"
	if got.Header.Get("Authorization") != wantAuth {
		t.Errorf("Authorization header = %q, want %q (S-APC-003)", got.Header.Get("Authorization"), wantAuth)
	}
}

// TestNew_TwoAdaptersDoNotShareStubbedClients covers S-APC-004: two adapters
// constructed with two different stubbed clients each observe exactly their
// own request, proving no shared or cached client is substituted. It also
// stands as the mutation-detection test for S-APC-005 (Mutation proof #1):
// an implementation that stores an injected client without using it — for
// example, always driving traffic through an internally built client
// instead — leaves at least one stub with zero observed requests.
func TestNew_TwoAdaptersDoNotShareStubbedClients(t *testing.T) {
	t.Parallel()

	stub1 := &stubTransport{}
	stub2 := &stubTransport{}

	c1, err := New(Config{
		Endpoint:   "http://one.invalid/v1",
		Credential: NewCredential("token-one"),
		HTTPClient: &http.Client{Transport: stub1},
	})
	if err != nil {
		t.Fatalf("New() c1 error = %v, want nil", err)
	}
	c2, err := New(Config{
		Endpoint:   "http://two.invalid/v1",
		Credential: NewCredential("token-two"),
		HTTPClient: &http.Client{Transport: stub2},
	})
	if err != nil {
		t.Fatalf("New() c2 error = %v, want nil", err)
	}

	driveOneRequest(t, c1)
	driveOneRequest(t, c2)

	if len(stub1.requests) != 1 {
		t.Fatalf("stub1 observed %d requests, want 1 (S-APC-004/S-APC-005)", len(stub1.requests))
	}
	if len(stub2.requests) != 1 {
		t.Fatalf("stub2 observed %d requests, want 1 (S-APC-004/S-APC-005)", len(stub2.requests))
	}
	if stub1.requests[0].URL.Host != "one.invalid" {
		t.Errorf("stub1 observed host %q, want %q — cross-adapter leakage (S-APC-004)", stub1.requests[0].URL.Host, "one.invalid")
	}
	if stub2.requests[0].URL.Host != "two.invalid" {
		t.Errorf("stub2 observed host %q, want %q — cross-adapter leakage (S-APC-004)", stub2.requests[0].URL.Host, "two.invalid")
	}
	if stub1.requests[0].Header.Get("Authorization") == stub2.requests[0].Header.Get("Authorization") {
		t.Errorf("both stubs observed the same Authorization header — cross-adapter leakage (S-APC-004)")
	}
}

// TestNew_ConfigurationFaults covers R-APC-002's whole scenario set,
// S-APC-006 through S-APC-010: every malformed-endpoint shape and the
// empty-credential shape each fail construction, typed through the AI-04
// taxonomy, before any request exists, drawn from the unchanged sentinel
// set, with no usable adapter returned.
func TestNew_ConfigurationFaults(t *testing.T) {
	t.Parallel()

	malformedEndpoints := []struct {
		name     string
		endpoint string
	}{
		{"empty", ""},
		{"whitespace-only", "   "},
		{"no-scheme", "example.invalid/v1"},
		{"unsupported-scheme", "ftp://example.invalid/v1"},
		{"unparseable", "http://[::1"},
		{"control-character", "http://example.invalid/v1\x00"},
	}

	for _, tc := range malformedEndpoints {
		t.Run("endpoint/"+tc.name, func(t *testing.T) {
			t.Parallel()

			stub := &stubTransport{}
			c, err := New(Config{
				Endpoint:   tc.endpoint,
				Credential: NewCredential("token"),
				HTTPClient: &http.Client{Transport: stub},
			})
			assertConfigurationFault(t, c, err, ai.ErrMalformed, "endpoint", stub)
		})
	}

	t.Run("credential/empty", func(t *testing.T) {
		t.Parallel()

		stub := &stubTransport{}
		c, err := New(Config{
			Endpoint:   "http://example.invalid/v1",
			Credential: NewCredential(""),
			HTTPClient: &http.Client{Transport: stub},
		})
		assertConfigurationFault(t, c, err, ai.ErrEmpty, "credential", stub)
	})
}

// assertConfigurationFault asserts the shared shape every AI-25.1
// configuration fault must have: errors.Is names wantRule, errors.As yields
// a position naming wantField, zero round trips reached the stub
// (S-APC-008), no usable adapter was returned (S-APC-010), and the failure
// is drawn from exactly one member of the closed, unextended AI-04 sentinel
// set (S-APC-009).
func assertConfigurationFault(t *testing.T, c *Client, err error, wantRule error, wantField string, stub *stubTransport) {
	t.Helper()

	if err == nil {
		t.Fatal("New() error = nil, want a configuration fault")
	}
	if c != nil {
		t.Errorf("New() returned a non-nil *Client on failure, want nil — no usable adapter on failure (S-APC-010)")
	}
	if !errors.Is(err, wantRule) {
		t.Errorf("errors.Is(err, %v) = false, want true (S-APC-006/S-APC-007); err = %v", wantRule, err)
	}

	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
	}
	path := violation.Path()
	if len(path) == 0 || path[0].Name() != wantField {
		t.Errorf("violation position = %q, want first step named %q (S-APC-006/S-APC-007)", path.String(), wantField)
	}

	if len(stub.requests) != 0 {
		t.Errorf("stub observed %d requests, want 0 — the fault must be decided before any request exists (S-APC-008)", len(stub.requests))
	}

	// The closed AI-04 rule-class registry, reused unchanged (S-APC-009):
	// this package must never introduce a sentinel of its own.
	knownSentinels := []error{
		ai.ErrEmpty, ai.ErrNotInVocabulary, ai.ErrOutOfRange, ai.ErrMalformed,
		ai.ErrUnresolvedReference, ai.ErrDuplicate, ai.ErrMisplaced,
	}
	if len(knownSentinels) != 7 {
		t.Fatalf("known AI-04 sentinel count = %d, want 7 — the baseline set this package must not extend", len(knownSentinels))
	}
	matches := 0
	for _, sentinel := range knownSentinels {
		if errors.Is(err, sentinel) {
			matches++
		}
	}
	if matches != 1 {
		t.Errorf("err matched %d of the known AI-04 sentinels, want exactly 1 — zero new sentinels (S-APC-009)", matches)
	}
}
