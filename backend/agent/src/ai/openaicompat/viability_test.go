// credential-scan:deliberate-plant — plants "Bearer viability-proxy-token"
// (a 21-char bearer-class literal) to assert the proxy Authorization header
// still carries the real credential despite a poisoned proxy environment
// (S-APC-052). Declared here per AI-36's recursive credential scan (D-3,
// R-ART-022); enumerated in credential_scan_test.go's deliberatePlantAllowlist.
package openaicompat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// This file is AI-25.3's [leaf] node (R-APC-013): the milestone's only
// sanctioned request. It proves that construction wired the injected
// endpoint and credential into a real, network-visible HTTP request against
// a local test server — nothing more.
//
// # Scope: attachment only, not a wire-level secrecy guarantee (R-APC-014)
//
// Nothing in this file asserts wire-level redaction, provider-returned-
// content non-disclosure, or log safety of the credential. The bound on
// what a wire error body may carry is AI-32.5's; the exhaustive sentinel
// sweep across every failure surface is AI-36.1's. This file proves only
// that the credential a caller injects reaches the wire in the dialect's
// expected header shape — what happens to it after that is out of scope
// here. The credential's type-shape opacity (never rendered by fmt or
// json.Marshal) is asserted once already, in credential_test.go's
// TestCredential_NeverRendersRawToken (S-APC-053); this file does not
// duplicate or extend that assertion (S-APC-054, S-APC-055).
//
// # Why the probe uses the adapter-built client (Config.HTTPClient left nil)
//
// http.ProxyFromEnvironment's localhost exemption matches only the literal
// host name "localhost". httptest.NewServer's URL is a 127.0.0.1 loopback
// address, which IS proxy-eligible under that rule — a client whose
// transport consults the proxy environment (http.DefaultTransport among
// them) could silently misroute this probe on any developer machine with
// HTTP_PROXY set. Every construction in this file therefore leaves
// Config.HTTPClient nil, so New builds its own client, whose transport
// leaves Proxy unset (R-APC-009) — proxy-immune by construction, which is
// what TestViability_AdapterBuiltClientIgnoresProxyEnvironment below
// exercises directly.
//
// # The header shape, resolved against merged fact
//
// AI-24's merged decision.md § 4's credential-handling-boundary axis and
// § 11's credential boundary both state the same shape: "a bearer token is
// attached as a single Authorization: Bearer <token> header". That is also
// exactly what request.go already implements (Credential.bearer() returns
// "Bearer "+token, attached via req.Header.Set("Authorization", ...)) —
// merged fact and landed code agree, so this file asserts the literal
// "Bearer <token>" shape rather than a placeholder.

// capturedRequest is what the local test server observed about one
// request. The handler reports one of these per request on a channel
// rather than writing to a shared variable directly: a channel send/
// receive establishes the happens-before edge the race detector requires
// between the server's handler goroutine and this file's assertions, which
// a bare shared field would not.
type capturedRequest struct {
	authorization string
	path          string
}

// newViabilityServer returns a local test server whose handler reports
// each request's Authorization header and URL path on ch, then responds
// 200 OK. Callers must defer server.Close().
func newViabilityServer(ch chan<- capturedRequest) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- capturedRequest{
			authorization: r.Header.Get("Authorization"),
			path:          r.URL.Path,
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// driveOneViabilityRequest builds and sends exactly one request through c,
// closing the response body once observed. Unlike client_test.go's
// driveOneRequest (which drives a stub transport that touches no real
// socket), every request in this file crosses a real local network
// connection, so the body is drained and closed here rather than left for
// the deferred server.Close() to clean up.
func driveOneViabilityRequest(t *testing.T, c *Client, segments ...string) {
	t.Helper()
	req, err := c.newRequest(context.Background(), nil, segments...)
	if err != nil {
		t.Fatalf("newRequest() error = %v, want nil", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		t.Fatalf("httpClient.Do() error = %v, want nil", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

// receiveCapturedRequest waits for the local test server to report one
// request on ch, failing the test if none arrives within a generous bound
// — a local loopback round trip taking anywhere near this long indicates
// something is already wrong, not a slow but healthy system.
func receiveCapturedRequest(t *testing.T, ch <-chan capturedRequest) capturedRequest {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("local test server did not observe a request within 5s (S-APC-048)")
		return capturedRequest{}
	}
}

// assertNoExtraRequest fails the test if the local test server reported
// more than the one request already consumed from ch — the exactly-one
// half of S-APC-048.
func assertNoExtraRequest(t *testing.T, ch <-chan capturedRequest) {
	t.Helper()
	select {
	case extra := <-ch:
		t.Fatalf("local test server observed an unexpected extra request, want exactly one (S-APC-048): %+v", extra)
	default:
	}
}

// TestViability_RequestReachesLocalServerWithCredentialAttached covers
// S-APC-048 through S-APC-051: the adapter-built client path, driven
// against a local test server, delivers exactly one request per
// construction, carrying that construction's own injected credential in
// the dialect's Authorization: Bearer <token> header shape, at the
// R-APC-006 joined path, with no cross-construction leakage.
func TestViability_RequestReachesLocalServerWithCredentialAttached(t *testing.T) {
	t.Parallel()

	ch := make(chan capturedRequest, 1)
	server := newViabilityServer(ch)
	defer server.Close()

	t.Run("first construction, first credential", func(t *testing.T) {
		c, err := New(Config{
			Endpoint:   server.URL,
			Credential: NewCredential("viability-token-one"),
		})
		if err != nil {
			t.Fatalf("New() error = %v, want nil", err)
		}

		driveOneViabilityRequest(t, c, "chat", "completions")

		got := receiveCapturedRequest(t, ch)
		assertNoExtraRequest(t, ch)

		const wantAuth = "Bearer viability-token-one"
		if got.authorization != wantAuth {
			t.Errorf("Authorization header = %q, want %q (S-APC-049)", got.authorization, wantAuth)
		}
		const wantPath = "/chat/completions"
		if got.path != wantPath {
			t.Errorf("path = %q, want %q (S-APC-050)", got.path, wantPath)
		}
	})

	t.Run("second construction, different credential carries the second value", func(t *testing.T) {
		c, err := New(Config{
			Endpoint:   server.URL,
			Credential: NewCredential("viability-token-two"),
		})
		if err != nil {
			t.Fatalf("New() error = %v, want nil", err)
		}

		driveOneViabilityRequest(t, c, "chat", "completions")

		got := receiveCapturedRequest(t, ch)
		assertNoExtraRequest(t, ch)

		const wantAuth = "Bearer viability-token-two"
		if got.authorization != wantAuth {
			t.Errorf("Authorization header = %q, want %q — attachment derives from the injected credential, not a hard-coded value (S-APC-051)", got.authorization, wantAuth)
		}
	})
}

// TestViability_AdapterBuiltClientIgnoresProxyEnvironment covers S-APC-052:
// with a proxy environment variable set in this test process to an address
// that would fail, the probe still reaches the local test server — the
// adapter-built client's transport leaves its proxy resolver unset
// (R-APC-009), so it never consults HTTP_PROXY at all.
//
// This test is deliberately SERIAL. Do NOT add t.Parallel() anywhere in
// this test: t.Setenv panics if called from a parallel test. This package
// already has one other serial test for the same reason (timeout_test.go's
// paired comparison). Go runs a package's non-parallel tests strictly
// sequentially, so the two never overlap, and t.Setenv's automatic cleanup
// restores the environment before the next serial test runs.
func TestViability_AdapterBuiltClientIgnoresProxyEnvironment(t *testing.T) {
	ch := make(chan capturedRequest, 1)
	server := newViabilityServer(ch)
	defer server.Close()

	// A dead address: nothing listens here. If the adapter-built client
	// consulted this at all, every request would fail to reach the real
	// local server instead of arriving directly.
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")

	c, err := New(Config{
		Endpoint:   server.URL,
		Credential: NewCredential("viability-proxy-token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	driveOneViabilityRequest(t, c, "chat", "completions")

	got := receiveCapturedRequest(t, ch)
	assertNoExtraRequest(t, ch)

	const wantAuth = "Bearer viability-proxy-token"
	if got.authorization != wantAuth {
		t.Errorf("Authorization header = %q, want %q — the request must still carry the credential despite the poisoned proxy environment (S-APC-052)", got.authorization, wantAuth)
	}
}
