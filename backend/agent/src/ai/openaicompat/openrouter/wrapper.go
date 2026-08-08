package openrouter

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// Config is the wrapper's full injection surface (R-OR-01). Every field is
// caller-supplied; the wrapper reads nothing from the environment, the
// filesystem or a process. Credential and HTTPClient are not optional in
// the sense that callers may leave them zero-valued — Credential must
// carry a non-empty bearer (openaicompat.New rejects an empty credential
// with ai.Invalid(ai.ErrEmpty, ai.At("credential")), and HTTPClient nil
// selects this wrapper's own bounded default client (R-APC-003,
// NFR-APC-F). The three attribution strings and Model default to no-op
// values (empty strings suppress their headers; a zero Model is replaced
// by openrouterDefaultModel at construction).
type Config struct {
	// Credential is the opaque bearer token attached to every outbound
	// request. Its raw value is never rendered by String(), GoString(),
	// json.Marshal, default formatting, or any path that reaches an
	// error message — see credential.go in the openaicompat package for
	// the type-shape opacity property.
	Credential openaicompat.Credential

	// HTTPClient is the *http.Client carrying outbound traffic. When nil,
	// NewProvider builds one with a fresh http.Transport whose bounds
	// mirror openaicompat's own newDefaultHTTPClient (R-APC-003,
	// R-APC-009: no DefaultTransport, no ProxyFromEnvironment). When
	// non-nil, it is used verbatim — its Transport is wrapped by
	// attributionRoundTripper; its bounds, including any whole-request
	// timeout, are its injector's decision (R-APC-003).
	HTTPClient *http.Client

	// HTTPReferer sets the HTTP-Referer header on every outbound request
	// when non-empty. An empty string suppresses the header (R-OR-02
	// sub-scenario 2).
	HTTPReferer string

	// XTitle sets both X-OpenRouter-Title and its alias X-Title on every
	// outbound request when non-empty. An empty string suppresses both
	// headers (R-OR-02 sub-scenario 2).
	XTitle string

	// XCategories sets X-OpenRouter-Categories on every outbound request
	// when non-empty. An empty string suppresses the header (R-OR-02
	// sub-scenario 2).
	XCategories string

	// Model overrides the model identifier carried on the wire body's
	// "model" field when non-empty. A zero value is replaced by
	// openrouterDefaultModel at construction so the wire never sees a
	// zero-value model — see endpoint.go's effectiveModel (R-OR-03).
	Model string

	// TracerProvider is optional (AI-37 Judgment Day remediation,
	// R-APC-016/S-APC-085). When nil, the underlying openaicompat.Client
	// substitutes the tracing API's own no-op provider (R-APC-016) — this
	// wrapper adds no default of its own. When non-nil, it is threaded to
	// openaicompat.New verbatim, exactly as every other injectable on
	// this Config already is: this field is this wrapper's own door onto
	// the same injection-only path client.go's own Config.TracerProvider
	// documents as "the only door a tracer reaches Layer 1 through" — a
	// caller composing this wrapper (the only shipped concrete provider)
	// had no way to open that door before this field existed.
	TracerProvider trace.TracerProvider
}

// redactedProvider is the wrapper-private type NewProvider returns
// (R-OR-08, R-APC-014). It satisfies ai.ModelProvider by delegating to
// an embedded *openaicompat.Client, and it carries its own String and
// GoString methods that render a fixed non-leaking label — without them,
// Go's default formatter on the underlying *openaicompat.Client would
// expose the credential's unexported token field through %+v and %#v
// (Credential.String() and Credential.GoString() already redact, but
// Client itself has no custom formatter, so default formatting
// reaches the field). The embedding is a deliberate Go-idiomatic
// forwarder for the Stream method, not a documented exported type —
// callers see ai.ModelProvider, not this struct.
//
// redactedProvider is intentionally unexported (lowercase) so the
// wrapper exposes no new public type — design § 6's "the wrapper is a
// factory (function), not a new type" guidance reads as "no new
// exported type", not "no internal helper type".
type redactedProvider struct {
	*openaicompat.Client
}

// Stream forwards to the embedded openaicompat.Client (AI-20, R-AMP-001
// — the interface signature is preserved verbatim).
func (p redactedProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	return p.Client.Stream(ctx, req)
}

// String renders a fixed label that names neither the underlying type
// nor any credential-shaped string (R-OR-08, R-APC-014, S-APC-053).
func (p redactedProvider) String() string {
	return "<openrouter provider>"
}

// GoString renders the same fixed label — Go-syntax representation
// that never reproduces the bearer token. Credential.GoString() already
// renders "<redacted>" (S-APC-053); this layer's role is to prevent Go's
// default %#v from descending into the embedded *openaicompat.Client's
// unexported fields.
func (p redactedProvider) GoString() string {
	return "<openrouter provider>"
}

// NewProvider returns an ai.ModelProvider that speaks the OpenAI-compatible
// Chat Completions dialect to OpenRouter, composing (not re-implementing)
// the shipped openaicompat package. The returned value is a
// redactedProvider wrapping an *openaicompat.Client whose outbound
// transport is wrapped by a wrapper-owned http.RoundTripper
// (transport.go) that injects the three attribution headers and
// overrides the wire body's model identifier.
//
// Construction is injection-only: NewProvider reads no environment
// variable, touches no filesystem path and spawns no process
// (R-OR-01, AI-25.2 invariant). Every input reaches the adapter through
// cfg, and the call to openaicompat.New validates the endpoint and the
// credential before any outbound request exists (R-APC-002).
//
// A malformed endpoint is rejected with ai.Invalid(ai.ErrMalformed,
// ai.At("endpoint")) — never an outbound request — mirroring the
// rejection shape openaicompat.New itself uses for the same failure
// (R-APC-002). An empty credential is rejected with ai.Invalid(
// ai.ErrEmpty, ai.At("credential")) for the same reason.
//
// NewProvider returns ai.ModelProvider, the one interface every concrete
// adapter satisfies (AI-20). The wrapper is a factory (function), not
// a new exported type (design § 6, design § 13); redactedProvider is an
// internal helper type whose only purpose is the redaction posture.
func NewProvider(cfg Config) (ai.ModelProvider, error) {
	transport := attributionRoundTripper{
		base:          baseTransport(cfg.HTTPClient),
		referer:       cfg.HTTPReferer,
		xTitle:        cfg.XTitle,
		xCategories:   cfg.XCategories,
		modelOverride: effectiveModel(cfg),
	}
	httpClient := &http.Client{Transport: transport}

	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:       openrouterBaseURL,
		Credential:     cfg.Credential,
		HTTPClient:     httpClient,
		TracerProvider: cfg.TracerProvider,
	})
	if err != nil {
		return nil, err
	}
	return redactedProvider{Client: client}, nil
}

// baseTransport returns the http.RoundTripper attributionRoundTripper
// wraps. When cfg.HTTPClient is nil, this wrapper builds a fresh
// http.Transport whose bounds mirror openaicompat's own
// newDefaultHTTPClient (R-APC-009: Proxy nil, no DefaultTransport, no
// ProxyFromEnvironment) — see backend/agent/src/ai/openaicompat/client.go
// for the canonical bound values. The duplication is documented here
// because openaicompat's helper is unexported and this wrapper must
// stay compositionally distinct from the package it composes.
//
// When cfg.HTTPClient is non-nil, attributionRoundTripper wraps the
// injected client's existing Transport verbatim — its bounds are its
// injector's decision (R-APC-003), not this wrapper's. When the
// injected client's Transport is itself nil (the http.Client default
// is http.DefaultTransport, which would silently adopt a proxy from
// the environment), this wrapper substitutes the same bounded
// default it would build for the nil-HTTPClient case — R-APC-009
// applies regardless of who owns the http.Client.
func baseTransport(cfgHTTPClient *http.Client) http.RoundTripper {
	if cfgHTTPClient != nil {
		if t := cfgHTTPClient.Transport; t != nil {
			return t
		}
	}
	return defaultBoundedTransport()
}

// defaultBoundedTransport returns a fresh *http.Transport with the
// canonical openaicompat-mirroring bounds (R-APC-009). Extracted so
// baseTransport and any future call site share one source-code
// location for the bound values — drift between two copies of these
// numbers is the maintenance hazard the extraction closes.
func defaultBoundedTransport() *http.Transport {
	return &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
}