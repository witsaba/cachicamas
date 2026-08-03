package openaicompat

import (
	"net/http"
	"net/url"
	"time"

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
}

// Client is a configured value for the OpenAI-compatible dialect. At AI-25
// it does nothing but hold configuration; it acquires streaming behaviour
// at AI-26.
type Client struct {
	base       *url.URL
	credential Credential
	httpClient *http.Client
}

// New returns a configured Client from cfg.
//
// A malformed endpoint or an empty credential fails construction with an
// AI-04 Violation before any request exists (R-APC-002). The nil-HTTPClient
// default path arrives in a later phase of this milestone.
func New(cfg Config) (*Client, error) {
	base, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if cfg.Credential.isEmpty() {
		return nil, ai.Invalid(ai.ErrEmpty, ai.At("credential"))
	}

	return &Client{
		base:       base,
		credential: cfg.Credential,
		httpClient: cfg.HTTPClient,
	}, nil
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
