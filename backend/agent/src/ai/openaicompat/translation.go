package openaicompat

import "github.com/cachicamas/backend/agent/src/ai"

// Translate renders req as this adapter's wire body: a neutral request in,
// wire bytes out, taking no client, performing no I/O and mutating nothing
// req holds (NFR-ART-C).
//
// A request carrying a reasoning content part refuses (R-ART-015,
// policy.go's refuseReasoning/refuse): nil bytes and an
// [ai.PreStreamFailure] error, checked BEFORE appendBody runs at all, so
// no wire body is ever produced for a refused request — not even a
// partial one covering the messages that precede the reasoning part.
// AI-26.8's own exhaustive-walk refusals (R-ART-021, a later slice, for
// every other refuse-disposition feature) are still to come, added
// without changing this function's signature: refuseReasoning is one
// call among what will be more than one, not a special case wired
// directly into this function forever.
//
// The body is hand-assembled by body.go's appendBody, in one fixed
// source-code order, never struct-marshalled — see doc.go's wire-shape
// provenance section (claim 3) for why encoding/json's own compaction and
// HTML-escaping disqualify it for this milestone (R-ART-010).
func Translate(req ai.Request) ([]byte, error) {
	if err := refuseReasoning(req); err != nil {
		return nil, err
	}
	return appendBody(req), nil
}
