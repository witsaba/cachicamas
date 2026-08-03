package openaicompat

import "github.com/cachicamas/backend/agent/src/ai"

// Translate renders req as this adapter's wire body: a neutral request in,
// wire bytes out, taking no client, performing no I/O and mutating nothing
// req holds (NFR-ART-C). Once a later slice's refusal path exists
// (R-ART-015, R-ART-021), a refusal returns nil bytes and a
// [ai.PreStreamFailure] error; this slice never refuses, so it always
// returns a nil error.
//
// The body is hand-assembled by body.go's appendBody, in one fixed
// source-code order, never struct-marshalled — see doc.go's wire-shape
// provenance section (claim 3) for why encoding/json's own compaction and
// HTML-escaping disqualify it for this milestone (R-ART-010).
//
// R-ART-002 is this slice's whole scope: model identity, one rendered
// message and the unconditional stream/stream_options.include_usage
// fields (R-ART-017; the bytes are emitted here so no later slice rewrites
// every earlier expectation literal — design.md "Usage opt-in
// placement"). System segments, tools, generation options, every
// content-part variant, tool results, reasoning refusal and the
// exhaustive feature policy are later slices', added without changing
// this function's signature.
func Translate(req ai.Request) ([]byte, error) {
	return appendBody(req), nil
}
