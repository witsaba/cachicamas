package openaicompat

import "github.com/cachicamas/backend/agent/src/ai"

// appendSystemMessageObject appends one wire message object rendering
// segment in this adapter's system role (R-ART-005).
//
// # Wire role: system, always — a scoped ruling, not a hedge
//
// Claim 1 (doc.go's "Wire-shape provenance" section) cites both `system`
// and `developer` as legal roles for this message kind, and explicitly
// defers which one an adapter renders to "a later slice" — this one. The
// ruling: always `system`, never `developer`. See doc.go's "System role:
// system, not developer" section for the full rationale (dialect breadth
// over model recency) and its living-graph reopen trigger.
//
// # Shape: one wire message per segment, not one message with N content parts
//
// Rendering N segments as N separate role-"system" wire messages, each
// with a plain string content — rather than one message whose content is
// an array of N text-content-part objects — needs no new wire-shape
// citation: it composes only shapes already cited. Claim 1 settles
// "messages entry, role system"; claim 4 settles that this dialect
// enforces no role-alternation constraint at all, and its own citation
// (the vendor's documented parallel-tool-call flow) already establishes
// multiple CONSECUTIVE same-role messages as a working, vendor-endorsed
// pattern general to any role, not scoped to "tool". The plain-string
// content shape is the same one appendSingleTextContent (body.go,
// AI-26.1) already uses and cites. Composing these already-cited facts
// avoids introducing an unverified claim about a system message's own
// content-array element shape.
//
// This also satisfies R-ART-005's "no segment MUST be ... merged" for
// free, structurally: each segment is its own distinct wire message
// object, which cannot be merged with another by construction — there is
// no shared container two segments could be folded into.
func appendSystemMessageObject(buf []byte, segment ai.Segment) []byte {
	buf = append(buf, `{"role":"system","content":`...)
	buf = appendJSONString(buf, segment.Text())
	return append(buf, '}')
}
