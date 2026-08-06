// AI-38 — the OpenRouter attribution-header fixture.
//
// # Attribution headers — present variant
//
// The OpenRouter HTTP gateway expects three attribution headers on
// every outbound request (R-OR-02 sub-scenario 1, design C3 +
// Engram #2571):
//
//   - HTTP-Referer
//   - X-OpenRouter-Title  (and its alias X-Title)
//   - X-OpenRouter-Categories
//
// openrouterAttributionHeaderNames is the fixed name list the
// bridge-local http.RoundTripper writes — the same set
// openrouter/transport.go's own (unexported) attributionRoundTripper
// injects, mirrored here because PR #2's hard rule forbids
// modifying PR #1 code. Any future change to the attribution-header
// set (additions, removals, aliasing) must move both copies
// together — tracked at the verify phase's spec-compliance matrix
// and at the openrouter package's own headers-unawareness test
// (headers_unawareness_test.go), which reads openaicompat/request.go
// as raw bytes and fails on any of these three literals appearing
// in openaicompat's own header surface.
//
// # Attribution headers — absent variant
//
// An empty attribution string suppresses its header (R-OR-02
// sub-scenario 2). openrouterAttributionAbsentCount is the expected
// number of OpenRouter-attribution-named headers in the outbound
// header when every attribution field of the bridge's Config is
// the zero value: zero. A request built with all-empty attribution
// strings MUST NOT carry any of the four names above — the
// suppression is what makes "absence of attribution" a meaningful
// configuration shape, not a misconfiguration that silently emits
// empty-valued headers.
//
// # Single source of truth
//
// Both the openrouter package's own tests and this conformance
// bridge's tests iterate openrouterAttributionHeaderNames (a Go
// var, exported) so a future change to the header set has exactly
// one place to update. The alias relationship between
// X-OpenRouter-Title and X-Title is fixed by OpenRouter's API
// documentation; both names are listed, in that order, so an
// iteration that expects "X-Title absent" when xTitle is empty
// matches the documented behavior.

package fixtures

// openrouterAttributionHeaderNames is the fixed list of HTTP header
// names the conformance bridge injects on every outbound request
// (R-OR-02 sub-scenario 1).
//
// The list is ordered: HTTP-Referer, then the title pair (canonical
// name followed by alias), then the categories. Order is the only
// stable invariant a caller can rely on; the conformance cases
// iterate, not index, the slice.
var openrouterAttributionHeaderNames = []string{
	"HTTP-Referer",
	"X-OpenRouter-Title",
	"X-Title",
	"X-OpenRouter-Categories",
}

// openrouterAttributionPresentValues is the expected outbound header
// map a request built with all non-empty attribution strings
// produces — every name in openrouterAttributionHeaderNames is
// present with the configured value, and X-OpenRouter-Title and
// X-Title both carry the same configured title value (R-OR-02
// sub-scenario 1, "both names are aliases of each other").
var openrouterAttributionPresentValues = map[string]string{
	"HTTP-Referer":            "https://app.cachicamas.test/openrouter-conformance",
	"X-OpenRouter-Title":      "cachicamas-conformance-bridge",
	"X-Title":                 "cachicamas-conformance-bridge",
	"X-OpenRouter-Categories": "test,conformance",
}

// openrouterAttributionAbsentCount is the expected count of
// attribution-named headers in the outbound header when every
// attribution field is the zero value: zero. The constant exists
// so a future change to the set is forced to update the count, not
// silently leave a stale assertion in place.
const openrouterAttributionAbsentCount = 0

// openrouterAttributionPresentCount is the expected count of
// attribution-named headers in the outbound header when every
// attribution field is non-empty: the full set (HTTP-Referer +
// X-OpenRouter-Title + X-Title alias + X-OpenRouter-Categories).
const openrouterAttributionPresentCount = 4