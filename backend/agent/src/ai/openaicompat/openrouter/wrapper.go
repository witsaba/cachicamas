package openrouter

import (
	"net/http"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// Config is the wrapper's full injection surface (R-OR-01). Every field is
// caller-supplied; the wrapper reads nothing from the environment, the
// filesystem or a process. Credential and HTTPClient are not optional in
// the sense that callers may leave them zero-valued — Credential must
// carry a non-empty bearer (openaicompat.New rejects an empty credential
// with ai.Invalid(ai.ErrEmpty, ai.At("credential")), and HTTPClient nil
// selects openaicompat's own bounded default client (R-APC-003,
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
	// NewProvider builds one through openaicompat.New — whose own
	// newDefaultHTTPClient is bounded (R-APC-009: no DefaultTransport,
	// no ProxyFromEnvironment) and never sets a whole-request timeout
	// (R-APC-003). When non-nil, it is used verbatim by openaicompat.New.
	HTTPClient *http.Client

	// HTTPReferer sets the HTTP-Referer header on every outbound request
	// when non-empty. An empty string suppresses the header (R-OR-02
	// sub-scenario 2).
	HTTPReferer string

	// XTitle sets both X-OpenRouter-Title and its alias X-Title on every
	// outbound request when non-empty. An empty string suppresses both
	// headers (R-OR-02 sub-scenario 2).
	XTitle string

	// XCategories sets X-OpenRouter-Categories on every outbound
	// request when non-empty. An empty string suppresses the header
	// (R-OR-02 sub-scenario 2).
	XCategories string

	// Model overrides the model identifier carried on the wire body's
	// "model" field when non-empty. A zero value is replaced by
	// openrouterDefaultModel at construction so the wire never sees a
	// zero-value model — see endpoint.go's effectiveModel (R-OR-03).
	Model string
}

// NewProvider returns an ai.ModelProvider that speaks the OpenAI-compatible
// Chat Completions dialect to OpenRouter, composing (not re-implementing)
// the shipped openaicompat package. The returned value is an
// *openaicompat.Client whose outbound transport is wrapped by a
// wrapper-owned http.RoundTripper (transport.go) that injects the three
// attribution headers and overrides the wire body's model identifier.
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
// adapter satisfies (AI-20). The underlying *openaicompat.Client is the
// returned value; NewProvider is a factory (function), not a new type
// (design § 6, design § 13).
func NewProvider(cfg Config) (ai.ModelProvider, error) {
	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:   openrouterBaseURL,
		Credential: cfg.Credential,
		HTTPClient: cfg.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}