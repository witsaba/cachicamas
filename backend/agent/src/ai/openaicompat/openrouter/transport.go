package openrouter

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// attributionRoundTripper is the wrapper-owned http.RoundTripper that
// injects the three OpenRouter attribution headers on every outbound
// request (R-OR-02, design D2) and overrides the wire body's "model"
// field with the effective model (R-OR-03, design D6). It wraps
// whatever transport the openaicompat.Client itself built (default-
// bounded when no Config.HTTPClient is injected, or the caller-injected
// client's transport verbatim) and is the only seam through which the
// wrapper can attach headers without widening openaicompat's own
// "Authorization + Content-Type only" rule (request.go:35), and the
// only seam through which the wrapper can substitute the wire body's
// model without widening openaicompat's Translate (translation.go) to
// take a second input.
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
// Model override (R-OR-03): when modelOverride is non-empty, the wire
// body's "model" field is replaced with modelOverride on every
// outbound request — exactly the substitution effect OpenRouter's API
// requires for a deliberate model field that does not depend on the
// caller setting req.Model() at construction time.
//
// Concurrency note. http.Header.Clone is documented thread-safe and
// returns a deep copy; req.Clone(ctx) is documented thread-safe and
// returns a shallow copy with the body and header both cloned. The
// outbound request handed to rt.base is therefore a fresh value with no
// aliasing of the caller's req. RoundTrip is serial per request (http
// transport serializes per-request state); the only mutable shared
// state this type carries is the configured attribution strings and
// model override, set once at construction and never mutated.
type attributionRoundTripper struct {
	base          http.RoundTripper
	referer       string
	xTitle        string
	xCategories   string
	modelOverride string
}

// RoundTrip clones the request's headers, applies the three attribution
// headers when their values are non-empty, applies the model override to
// the request body when configured, clones the request itself so the
// outbound request is a fresh value with no aliasing of the caller's
// req, and delegates to the wrapped base RoundTripper.
//
// Header mutation goes through req.Header.Clone() — the header field is
// a map[string][]string; a shallow assignment would alias the caller's
// map and either mutate it (wrong) or be mutated by it (race).
// req.Clone(ctx) carries the cloned header into a fresh request value.
//
// Body mutation goes through bytes.NewReader on the modified bytes —
// the inbound req.Body is replaced with a fresh reader, so the caller
// cannot observe the override, and req.ContentLength is set to the
// modified length so openaicompat's own http.NewRequestWithContext
// (request.go:31) does not have to re-chunk the body.
func (rt attributionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedHeader := req.Header.Clone()
	applyAttribution(clonedHeader, rt.referer, rt.xTitle, rt.xCategories)

	clonedReq := req.Clone(req.Context())
	clonedReq.Header = clonedHeader

	if rt.modelOverride != "" && clonedReq.Body != nil {
		body, err := io.ReadAll(clonedReq.Body)
		if err != nil {
			return nil, err
		}
		_ = clonedReq.Body.Close()
		body = overrideBodyModel(body, rt.modelOverride)
		clonedReq.Body = io.NopCloser(bytes.NewReader(body))
		clonedReq.ContentLength = int64(len(body))
	}

	return rt.base.RoundTrip(clonedReq)
}

// overrideBodyModel replaces the top-level "model" field in body with
// override, returning the modified bytes. The body is the JSON openaicompat
// renders from an ai.Request (body.go's appendBody), whose field order is
// fixed at source: "model" is the first field, so its byte-offset span
// is precisely `"model":"<value>"` followed by `,`.
//
// The replacement uses json.Marshal on override — its output is
// `"<value>"` with the outer quotes json.Marshal always renders, so the
// surrounding structure of body stays intact. The substitute value is
// not escaped further because json.Marshal's deterministic escaping is
// the same one openaicompat's appendJSONString (body.go) already uses —
// a model identifier like "openai/gpt-4o" has no bytes below 0x20, no
// quotes and no backslashes, so the two paths are byte-identical for any
// realistic model identifier.
func overrideBodyModel(body []byte, override string) []byte {
	const key = `"model":`
	keyIdx := bytes.Index(body, []byte(key))
	if keyIdx < 0 {
		// The wire body never had a model field — openaicompat always
		// appends one, so reaching this branch is a defensive fallback
		// that returns the body unchanged rather than fabricate bytes.
		return body
	}
	valStart := keyIdx + len(key)
	for valStart < len(body) && (body[valStart] == ' ' || body[valStart] == '\t' || body[valStart] == '\n' || body[valStart] == '\r') {
		valStart++
	}
	if valStart >= len(body) || body[valStart] != '"' {
		return body
	}
	valEnd := valStart + 1
	for valEnd < len(body) {
		if body[valEnd] == '\\' {
			valEnd += 2
			continue
		}
		if body[valEnd] == '"' {
			break
		}
		valEnd++
	}
	if valEnd >= len(body) {
		return body
	}

	encoded, err := json.Marshal(override)
	if err != nil {
		return body
	}

	out := make([]byte, 0, len(body)-(valEnd-valStart-1)+len(encoded))
	out = append(out, body[:valStart]...)
	out = append(out, encoded...)
	out = append(out, body[valEnd+1:]...)
	return out
}