// AI-26.3 — every readable content-part variant translates, message and
// intra-message part order are preserved exactly, and consecutive
// same-role messages are never merged (R-ART-007, R-ART-008, R-ART-009).
//
// This file supersedes body.go's (AI-26.1) skeleton-only
// appendMessageObject/appendSingleTextContent outright, exactly as
// body.go's own header comment said it would: this is the milestone's
// full message rendering, not a growth of the skeleton's placeholder.

package openaicompat

import "github.com/cachicamas/backend/agent/src/ai"

// appendMessageObject appends one wire message object for message:
// {"role":...,"content":...}.
//
// It is called once per ai.Message, unconditionally, from
// appendMessagesField's own loop (body.go) — the same loop AI-26.1 wrote,
// unedited by this slice. Nothing here reads, remembers or compares
// against a previously rendered message: this function carries no state
// across calls, so a run of consecutive ai.Message values sharing one
// role.String() renders as that many distinct wire message objects, never
// fewer (R-ART-009). See doc.go's "No merging of consecutive same-role
// messages" section for the decision and its reason (S-ART-034).
func appendMessageObject(buf []byte, message ai.Message) []byte {
	buf = append(buf, `{"role":`...)
	buf = appendJSONString(buf, message.Role().String())
	buf = append(buf, `,"content":`...)
	buf = appendMessageContent(buf, message)
	return append(buf, '}')
}

// appendMessageContent appends message's "content" value.
//
// Claim 1.0.5 (doc.go's wire-shape provenance section) documents content
// as oneOf[string, array]: a message carrying exactly one part renders
// its content as a bare JSON string (the shape AI-26.1's skeleton already
// used and every earlier slice's expectation already commits to,
// preserved here byte for byte); a message carrying more than one part
// renders content as an array, one typed content-part object per part
// (appendContentPartObject, below), in that message's own order.
//
// message.Content() is already a caller-ordered, freshly cloned slice
// (message.go's own clone-on-read contract, package ai) — ranging over it
// here in index order introduces no map of our own (design.md "Map
// discipline"), which is what makes both the single-part and the
// multi-part shape preserve intra-message part order structurally
// (R-ART-008) rather than by a separate ordering step.
//
// A single part of a kind this phase cannot read renders through the same
// panic appendContentPartObject's default case uses, naming the kind —
// see that function's own doc comment for why a panic, and
// message_test.go's partKindDispositions table for which later phase owns
// each deferred kind and what it must verify.
func appendMessageContent(buf []byte, message ai.Message) []byte {
	content := message.Content()
	if len(content) == 1 {
		part := content[0]
		if text, ok := part.Text(); ok {
			return appendJSONString(buf, text)
		}
		panic(unreadableContentPartKindPanic(part.Kind()))
	}

	buf = append(buf, '[')
	for i, part := range content {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendContentPartObject(buf, part)
	}
	return append(buf, ']')
}

// appendContentPartObject appends one typed content-part object — the
// array shape's own per-element rendering (appendMessageContent, above),
// used whenever a message carries more than one part.
//
// PartKindText is this phase's only readable content-part variant
// (R-ART-007): {"type":"text","text":...}. Every other member of
// ai.PartKinds() — PartKindReasoning, PartKindToolCall, PartKindToolResult
// — is deferred to a later phase/node, each named precisely in
// message_test.go's partKindDispositions table (TestMessage_PartKindCoverage,
// S-ART-027), which walks ai.PartKinds() itself so a vocabulary member
// added later with no entry there fails naming it, rather than silently
// falling through this function's own default case unnoticed.
//
// Reaching the default case panics, naming the kind via kind.String(),
// rather than silently rendering nothing (a shorter body than the caller
// authored, with no error) or silently rendering the wrong bytes (some
// other kind's shape) — both of which R-ART-007 forbids by name ("none
// translates by accident"). This is a transitional safety net, not the
// deferred kind's eventual replacement site: AI-26.6's reasoning refusal
// (Phase 7) and AI-26.8.2's exhaustive policy walk (Phase 8) intercept
// their kinds earlier, before appendBody appends anything at all, through
// policy.go's shared refusal door — see message_test.go's own hand-off
// notes for exactly what each later phase must verify.
func appendContentPartObject(buf []byte, part ai.Part) []byte {
	switch part.Kind() {
	case ai.PartKindText:
		text, _ := part.Text()
		buf = append(buf, `{"type":"text","text":`...)
		buf = appendJSONString(buf, text)
		return append(buf, '}')
	default:
		panic(unreadableContentPartKindPanic(part.Kind()))
	}
}

// unreadableContentPartKindPanic builds the one panic message both
// appendMessageContent's single-part branch and appendContentPartObject's
// default case raise for a content-part kind this phase cannot read — one
// message format, so the two call sites cannot drift and report the kind
// differently.
func unreadableContentPartKindPanic(kind ai.PartKind) string {
	return "openaicompat: content part kind not readable at this phase (message.go): " + kind.String()
}
