package openrouter

import "net/http"

// attributionRoundTripper is the wrapper-owned http.RoundTripper that
// injects the three OpenRouter attribution headers on every outbound
// request (R-OR-02, design D2). It wraps whatever transport the
// openaicompat.Client itself built (default-bounded when no
// Config.HTTPClient is injected, or the caller-injected client's
// transport verbatim) and is the only seam through which the wrapper
// can attach headers without widening openaicompat's own "Authorization
// + Content-Type only" rule (request.go:35).
//
// Header set:
//
//   - HTTP-Referer                 — when referer != ""
//   - X-OpenRouter-Title            — when xTitle != ""
//   - X-Title                       — when xTitle != "" (alias of
//                                     X-OpenRouter-Title)
//   - X-OpenRouter-Categories       — when xCategories != ""
//
// An empty attribution string suppresses its header (R-OR-02
// sub-scenario 2). No header is set with an empty value, ever.
//
// Concurrency note. http.Header.Clone is documented thread-safe and
// returns a deep copy; req.Clone(ctx) is documented thread-safe and
// returns a shallow copy with the body and header both cloned. The
// outbound request handed to rt.base is therefore a fresh value with no
// aliasing of the caller's req. RoundTrip is serial per request (http
// transport serializes per-request state); the only mutable shared
// state this type carries is the configured attribution strings, which
// are set once at construction and never mutated.
type attributionRoundTripper struct {
	base        http.RoundTripper
	referer     string
	xTitle      string
	xCategories string
}

// RoundTrip clones the request's headers, applies the three attribution
// headers when their values are non-empty, clones the request itself so
// the outbound request is a fresh value with no aliasing of the caller's
// req, and delegates to the wrapped base RoundTripper.
//
// Header mutation goes through req.Header.Clone() — the header field is
// a map[string][]string; a shallow assignment would alias the caller's
// map and either mutate it (wrong) or be mutated by it (race).
// req.Clone(ctx) carries the cloned header into a fresh request value.
func (rt attributionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedHeader := req.Header.Clone()
	applyAttribution(clonedHeader, rt.referer, rt.xTitle, rt.xCategories)
	clonedReq := req.Clone(req.Context())
	clonedReq.Header = clonedHeader
	return rt.base.RoundTrip(clonedReq)
}