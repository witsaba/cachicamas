package openaicompat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

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

// defaultTransportSnapshot captures the observable fields of
// http.DefaultTransport that this package must never mutate. It is a plain
// value of comparable fields so two snapshots compare with ==.
type defaultTransportSnapshot struct {
	dialContextSet        bool
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	idleConnTimeout       time.Duration
	proxySet              bool
	maxIdleConns          int
}

func snapshotDefaultTransport(t *testing.T) defaultTransportSnapshot {
	t.Helper()
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatalf("http.DefaultTransport is %T, want *http.Transport", http.DefaultTransport)
	}
	return defaultTransportSnapshot{
		dialContextSet:        tr.DialContext != nil,
		tlsHandshakeTimeout:   tr.TLSHandshakeTimeout,
		responseHeaderTimeout: tr.ResponseHeaderTimeout,
		idleConnTimeout:       tr.IdleConnTimeout,
		proxySet:              tr.Proxy != nil,
		maxIdleConns:          tr.MaxIdleConns,
	}
}

// defaultClientSnapshot captures the observable fields of http.DefaultClient
// that this package must never mutate.
type defaultClientSnapshot struct {
	timeout        time.Duration
	transportIsNil bool
}

func snapshotDefaultClient(t *testing.T) defaultClientSnapshot {
	t.Helper()
	return defaultClientSnapshot{
		timeout:        http.DefaultClient.Timeout,
		transportIsNil: http.DefaultClient.Transport == nil,
	}
}

// TestNew_DoesNotMutateProcessWideDefaults covers S-APC-012: the observable
// configuration of http.DefaultClient and http.DefaultTransport is
// unchanged after construction on both the injected-client path and the
// adapter-built (nil-client) path. It is also the mutation-detection test
// for S-APC-014 (Mutation proof #2): staging a deliberate assignment onto
// http.DefaultTransport's fields must make this test fail.
func TestNew_DoesNotMutateProcessWideDefaults(t *testing.T) {
	t.Parallel()

	beforeTransport := snapshotDefaultTransport(t)
	beforeClient := snapshotDefaultClient(t)

	stub := &stubTransport{}
	if _, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("token"),
		HTTPClient: &http.Client{Transport: stub},
	}); err != nil {
		t.Fatalf("New() injected path error = %v, want nil", err)
	}

	if _, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("token"),
	}); err != nil {
		t.Fatalf("New() adapter-built path error = %v, want nil", err)
	}

	afterTransport := snapshotDefaultTransport(t)
	afterClient := snapshotDefaultClient(t)

	if beforeTransport != afterTransport {
		t.Errorf("http.DefaultTransport observably changed: before=%+v after=%+v (S-APC-012/S-APC-014)", beforeTransport, afterTransport)
	}
	if beforeClient != afterClient {
		t.Errorf("http.DefaultClient observably changed: before=%+v after=%+v (S-APC-012/S-APC-014)", beforeClient, afterClient)
	}
}

// TestNew_SharedInjectedClientNotMutatedAcrossTwoConstructions covers
// S-APC-013: one HTTP client value used to construct two adapters is left
// exactly as it was, and the first adapter's outbound behaviour is
// unaffected by the second construction.
func TestNew_SharedInjectedClientNotMutatedAcrossTwoConstructions(t *testing.T) {
	t.Parallel()

	stub := &stubTransport{}
	shared := &http.Client{Transport: stub}

	c1, err := New(Config{
		Endpoint:   "http://one.invalid/v1",
		Credential: NewCredential("token-one"),
		HTTPClient: shared,
	})
	if err != nil {
		t.Fatalf("New() c1 error = %v, want nil", err)
	}

	beforeTimeout := shared.Timeout
	beforeTransport := shared.Transport

	if _, err := New(Config{
		Endpoint:   "http://two.invalid/v1",
		Credential: NewCredential("token-two"),
		HTTPClient: shared,
	}); err != nil {
		t.Fatalf("New() c2 error = %v, want nil", err)
	}

	if shared.Timeout != beforeTimeout {
		t.Errorf("shared.Timeout changed after second construction: before=%v after=%v (S-APC-013)", beforeTimeout, shared.Timeout)
	}
	if shared.Transport != beforeTransport {
		t.Error("shared.Transport changed identity after second construction (S-APC-013)")
	}

	driveOneRequest(t, c1)
	if len(stub.requests) != 1 {
		t.Fatalf("stub observed %d requests after driving c1, want 1 (S-APC-013)", len(stub.requests))
	}
	if stub.requests[0].URL.Host != "one.invalid" {
		t.Errorf("c1 request host = %q, want %q — c1 unaffected by c2's construction (S-APC-013)", stub.requests[0].URL.Host, "one.invalid")
	}
}

// TestConfig_SurfaceIsEndpointCredentialAndOptionalClient covers S-APC-015:
// the construction surface is exactly the endpoint, the credential and an
// optional HTTP client — no timeout, deadline or bound value among them.
func TestConfig_SurfaceIsEndpointCredentialAndOptionalClient(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(Config{})
	if typ.NumField() != 3 {
		t.Fatalf("Config has %d fields, want exactly 3 (S-APC-015)", typ.NumField())
	}

	wantKinds := map[string]reflect.Kind{
		"Endpoint":   reflect.String,
		"Credential": reflect.Struct,
		"HTTPClient": reflect.Ptr,
	}
	for name, wantKind := range wantKinds {
		field, ok := typ.FieldByName(name)
		if !ok {
			t.Errorf("Config has no field %q (S-APC-015)", name)
			continue
		}
		if field.Type.Kind() != wantKind {
			t.Errorf("Config.%s kind = %v, want %v (S-APC-015)", name, field.Type.Kind(), wantKind)
		}
	}

	forbiddenNameFragments := []string{"Timeout", "Deadline", "Bound", "Duration"}
	for i := range typ.NumField() {
		name := typ.Field(i).Name
		for _, fragment := range forbiddenNameFragments {
			if strings.Contains(name, fragment) {
				t.Errorf("Config.%s looks like a timeout/deadline/bound field, which must not be caller-injectable (S-APC-015)", name)
			}
		}
	}
}

// TestNew_AdapterBuiltClientIsNeitherDefaultClientNorDefaultTransport
// covers S-APC-016: constructing with no client supplied builds a fresh
// client and transport rather than adopting the shared, process-wide
// defaults it could not bound without mutating. It is also part of the
// mutation-detection coverage for S-APC-037 (Mutation proof #5): routing
// the adapter-built client through http.DefaultTransport must make this
// test fail.
func TestNew_AdapterBuiltClientIsNeitherDefaultClientNorDefaultTransport(t *testing.T) {
	t.Parallel()

	c, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	if c.httpClient == http.DefaultClient {
		t.Error("adapter-built client is http.DefaultClient itself (S-APC-016)")
	}
	if c.httpClient.Transport == http.DefaultTransport {
		t.Error("adapter-built client's transport is http.DefaultTransport itself (S-APC-016/S-APC-037)")
	}
}

// TestNew_AdapterBuiltTransportLeavesProxyUnset covers S-APC-036
// (R-APC-009): the adapter-built client's transport never resolves a proxy
// from the environment. It is also part of the mutation-detection coverage
// for S-APC-037 (Mutation proof #5).
func TestNew_AdapterBuiltTransportLeavesProxyUnset(t *testing.T) {
	t.Parallel()

	c, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("adapter-built client's Transport is %T, want *http.Transport", c.httpClient.Transport)
	}
	if tr.Proxy != nil {
		t.Error("adapter-built transport's Proxy resolver is set, want nil — no environment-derived proxy (S-APC-036)")
	}
}

// TestNew_AdapterBuiltTransportBoundsExist covers S-APC-021 and S-APC-022
// (R-APC-005): the adapter-built client's connect, TLS-handshake,
// time-to-headers and idle bounds are each non-zero and equal the design's
// recorded safe values. This is a deliberately structural assertion, not a
// behavioural one — see R-APC-005's own reasoning on why a probe would
// either wait a real bound out or produce a vacuous green. It is also the
// mutation-detection test for S-APC-023 (Mutation proof #4).
func TestNew_AdapterBuiltTransportBoundsExist(t *testing.T) {
	t.Parallel()

	c, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("adapter-built client's Transport is %T, want *http.Transport", c.httpClient.Transport)
	}

	if tr.DialContext == nil {
		t.Error("DialContext is nil, want non-nil — a connect-phase bound must exist (S-APC-021)")
	}
	if tr.TLSHandshakeTimeout != defaultTLSHandshakeTimeout {
		t.Errorf("TLSHandshakeTimeout = %v, want %v (S-APC-021)", tr.TLSHandshakeTimeout, defaultTLSHandshakeTimeout)
	}
	if tr.ResponseHeaderTimeout != defaultResponseHeaderTimeout {
		t.Errorf("ResponseHeaderTimeout = %v, want %v (S-APC-022)", tr.ResponseHeaderTimeout, defaultResponseHeaderTimeout)
	}
	if tr.IdleConnTimeout != defaultIdleConnTimeout {
		t.Errorf("IdleConnTimeout = %v, want %v (S-APC-022)", tr.IdleConnTimeout, defaultIdleConnTimeout)
	}

	for name, d := range map[string]time.Duration{
		"defaultDialTimeout":           defaultDialTimeout,
		"defaultTLSHandshakeTimeout":   defaultTLSHandshakeTimeout,
		"defaultResponseHeaderTimeout": defaultResponseHeaderTimeout,
		"defaultIdleConnTimeout":       defaultIdleConnTimeout,
	} {
		if d <= 0 {
			t.Errorf("%s = %v, want > 0 — bounds are asserted, not assumed (S-APC-021/S-APC-022)", name, d)
		}
	}
}

// TestNew_TotalityAcrossExtremeInputs covers S-APC-072 (NFR-APC-F): every
// extreme input to New — empty endpoint, empty credential, whitespace-only
// endpoint, both empty — returns a typed AI-04 failure naming the
// offending input's position, and none panics.
func TestNew_TotalityAcrossExtremeInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		endpoint   string
		credential string
		wantRule   error
		wantField  string
	}{
		{"empty endpoint", "", "token", ai.ErrMalformed, "endpoint"},
		{"empty credential", "http://example.invalid/v1", "", ai.ErrEmpty, "credential"},
		{"whitespace-only endpoint", "   ", "token", ai.ErrMalformed, "endpoint"},
		{"both endpoint and credential empty", "", "", ai.ErrMalformed, "endpoint"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				c   *Client
				err error
			)
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("New() panicked: %v, want a typed failure instead (S-APC-072)", r)
					}
				}()
				c, err = New(Config{
					Endpoint:   tc.endpoint,
					Credential: NewCredential(tc.credential),
				})
			}()

			assertConfigurationFault(t, c, err, tc.wantRule, tc.wantField, &stubTransport{})
		})
	}
}

// TestNew_SucceedsWithNoHTTPClientSupplied covers S-APC-073 (NFR-APC-F): a
// valid endpoint and credential with no HTTP client supplied succeeds,
// returns a usable adapter, and returns no error — an absent client is the
// documented selector for the adapter-built path, never a fault.
func TestNew_SucceedsWithNoHTTPClientSupplied(t *testing.T) {
	t.Parallel()

	c, err := New(Config{
		Endpoint:   "http://example.invalid/v1",
		Credential: NewCredential("token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil (S-APC-073)", err)
	}
	if c == nil {
		t.Fatal("New() returned a nil *Client, want a usable adapter (S-APC-073)")
	}
	if c.httpClient == nil {
		t.Error("c.httpClient is nil, want the adapter-built client (S-APC-073)")
	}
}
