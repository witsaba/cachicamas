package openrouter

import "net/http"

// applyAttribution mutates header by setting each of the three
// OpenRouter attribution headers when its corresponding value is
// non-empty (R-OR-02). Empty strings suppress their headers — never
// an empty-valued header, never an empty X-Title without its
// X-OpenRouter-Title sibling.
//
// The function is a pure helper extracted from RoundTrip so the
// attribution rule lives at one source-code location: a future change
// to the header set, the alias relationship, or the empty-string
// suppression rule applies here and is exercised by transport_test.go
// without re-reading RoundTrip's body.
//
// The headers and their alias:
//
//	HTTP-Referer           <- referer (one header)
//	X-OpenRouter-Title     <- xTitle (one header)
//	X-Title                <- xTitle (one alias header)
//	X-OpenRouter-Categories<- xCategories (one header)
func applyAttribution(header http.Header, referer, xTitle, xCategories string) {
	if referer != "" {
		header.Set("HTTP-Referer", referer)
	}
	if xTitle != "" {
		header.Set("X-OpenRouter-Title", xTitle)
		header.Set("X-Title", xTitle)
	}
	if xCategories != "" {
		header.Set("X-OpenRouter-Categories", xCategories)
	}
}