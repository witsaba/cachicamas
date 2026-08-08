package openaicompat

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Bound values for the client this package builds for itself when the
// caller injects none. Each bounds a distinct phase of the connection
// lifecycle, before any response body arrives — never the body read
// itself. See doc.go for why no whole-request cap exists at all.
const (
	// defaultDialTimeout bounds establishing a TCP connection only
	// (net.Dialer.Timeout).
	defaultDialTimeout = 10 * time.Second

	// defaultTLSHandshakeTimeout bounds completing the TLS handshake only,
	// once a connection exists (http.Transport.TLSHandshakeTimeout).
	defaultTLSHandshakeTimeout = 10 * time.Second

	// defaultResponseHeaderTimeout bounds time to response headers only. It
	// excludes the response body read entirely: once headers arrive, this
	// bound no longer applies (http.Transport.ResponseHeaderTimeout).
	defaultResponseHeaderTimeout = 60 * time.Second

	// defaultIdleConnTimeout bounds how long a pooled, idle connection may
	// be kept for reuse. It is a connection-pool bound, not a mid-stream
	// read-idle cutoff — no such cutoff exists in this package
	// (http.Transport.IdleConnTimeout).
	defaultIdleConnTimeout = 90 * time.Second
)

// Config configures a Client. Endpoint and Credential are required;
// HTTPClient is optional.
//
// This is the entire construction surface: no timeout, deadline or bound
// value appears among these fields (S-APC-015). Their absence is not an
// oversight — see doc.go.
type Config struct {
	// Endpoint is the base URL of the OpenAI-compatible provider or
	// gateway. It MUST be an absolute http:// or https:// URL.
	Endpoint string

	// Credential is the opaque bearer-token value attached to every
	// request this client builds. See credential.go.
	Credential Credential

	// HTTPClient is optional. When nil, New builds its own bounded client
	// (see the default* constants above); this is not a fault
	// (NFR-APC-F). When non-nil, it is used verbatim and is never
	// mutated — its bounds, including any whole-request timeout, are its
	// injector's decision (R-APC-003).
	HTTPClient *http.Client

	// TracerProvider is optional (AI-37, R-APC-016). When nil, New
	// substitutes the tracing API's own no-op provider — established once
	// at construction, so every span call site downstream is
	// unconditional and no adapter-side nil check on a tracing value
	// exists anywhere in this package (R-AOB-009). When non-nil, it is
	// used verbatim and is never wrapped, mutated or replaced by a
	// process-global value (R-APC-016). Layer 1 never reads or writes any
	// process-global telemetry registry (R-AOB-001) — this field is the
	// only door a tracer reaches Layer 1 through.
	TracerProvider trace.TracerProvider
}

// Client is a configured value for the OpenAI-compatible dialect. At AI-25
// it does nothing but hold configuration; it acquires streaming behaviour
// at AI-28. AI-37 adds tracer: derived once in New (AD-8) from the
// injected or defaulted TracerProvider, stored here so Stream never reads
// anything but this field.
type Client struct {
	base       *url.URL
	credential Credential
	httpClient *http.Client
	tracer     trace.Tracer
}

// clientRedactedLabel is the fixed, non-leaking label Client's own
// String/GoString render (AI-36, D-5, S-APC-078) — the same fixed-label
// posture openrouter's redactedProvider.String/GoString already use for
// the identical reason (wrapper.go:90-101).
const clientRedactedLabel = "<openaicompat client>"

// String renders a fixed label that names neither the underlying type nor
// any credential-shaped string (R-APC-015, D-5).
//
// Without it, Go's default formatter on a bare Client value reaches the
// unexported credential field by raw reflection: reflect.Value.CanInterface
// is false for a value read through an unexported struct field, so fmt
// cannot dispatch to Credential's own redacting String/GoString — it falls
// back to printing the field's raw underlying value instead, which is
// exactly how the bearer token would otherwise leak (proven empirically by
// TestClient_BareUnwrapped_NeverRendersCredential before this method
// existed). Value receiver, mirroring openrouter's redactedProvider
// (wrapper.go:90-101), so both the pointer and value forms redact.
func (c Client) String() string {
	return clientRedactedLabel
}

// GoString renders the same fixed label for the %#v verb — Credential's
// own GoString already redacts the token (credential.go); this layer's
// job is to stop Go's default %#v from reflecting past Client's own
// unexported fields to reach it (D-5, same reasoning as String above).
func (c Client) GoString() string {
	return clientRedactedLabel
}

// New returns a configured Client from cfg.
//
// A malformed endpoint or an empty credential fails construction with an
// AI-04 Violation before any request exists (R-APC-002). When cfg.HTTPClient
// is nil, New builds its own bounded client rather than failing or falling
// back to any process-wide default (R-APC-003, NFR-APC-F); when non-nil, it
// is used verbatim and is never mutated (R-APC-003).
func New(cfg Config) (*Client, error) {
	base, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if cfg.Credential.isEmpty() {
		return nil, ai.Invalid(ai.ErrEmpty, ai.At("credential"))
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = newDefaultHTTPClient()
	}

	tracerProvider := cfg.TracerProvider
	if tracerProvider == nil {
		tracerProvider = noop.NewTracerProvider()
	}

	return &Client{
		base:       base,
		credential: cfg.Credential,
		httpClient: httpClient,
		tracer:     tracerProvider.Tracer(tracerInstrumentationName),
	}, nil
}

// newDefaultHTTPClient builds the bounded client this package uses for
// itself when the caller injects none.
//
// Client.Timeout is deliberately left at its zero value: see doc.go for why
// no whole-request cap may exist on this path. Proxy is deliberately left
// nil: this is a fresh transport, independent of http.DefaultTransport, so
// it never resolves a proxy from the environment (R-APC-009).
func newDefaultHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: defaultDialTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ResponseHeaderTimeout: defaultResponseHeaderTimeout,
		IdleConnTimeout:       defaultIdleConnTimeout,
	}
	return &http.Client{
		Transport: transport,
	}
}

// parseEndpoint validates raw as an absolute http(s) URL and returns its
// parsed form.
//
// Every failure shape — empty, whitespace-only, schemeless, an unsupported
// scheme, unparseable, or carrying a control character — reports through
// the same rule class at the same position (S-APC-006): this package draws
// no behavioural distinction between them, only a single malformed-value
// class naming the endpoint input.
func parseEndpoint(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))
	}
	if parsed.Host == "" {
		return nil, ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))
	}
	return parsed, nil
}
