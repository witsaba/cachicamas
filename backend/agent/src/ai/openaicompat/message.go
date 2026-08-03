// AI-26.3 — every readable content-part variant translates, message and
// intra-message part order are preserved exactly, and consecutive
// same-role messages are never merged (R-ART-007, R-ART-008, R-ART-009).
//
// AI-26.5 grows this file with the two remaining content-part kinds
// message_test.go's own partKindDispositions table (AI-26.3) hands off
// here: a tool call renders as an element of the assistant message's own
// tool_calls array (splitToolCalls, appendToolCallObject), and a tool
// result renders as its own distinct role:"tool" wire message
// (appendToolResultMessages, appendToolResultObject) — R-ART-012,
// R-ART-013, R-ART-014.
//
// This file supersedes body.go's (AI-26.1) skeleton-only
// appendMessageObject/appendSingleTextContent outright, exactly as
// body.go's own header comment said it would: this is the milestone's
// full message rendering, not a growth of the skeleton's placeholder.

package openaicompat

import "github.com/cachicamas/backend/agent/src/ai"

// appendMessageObject appends one wire message object for message.
//
// A RoleTool message renders through an entirely different shape — one
// or more distinct role:"tool" wire messages, never a "role"/"content"
// object at all (R-ART-012, appendToolResultMessages, AI-26.5) —
// dispatched first and exclusively: request.go's own rolePermittedKinds
// table restricts a RoleTool message to PartKindToolResult content
// alone, enforced by ai.NewRequest before Translate is ever reached, so
// nothing past this branch ever needs to consider a tool-role message.
//
// Every other message renders {"role":...,"content":...,"tool_calls":...}:
// "content" comes from splitToolCalls' own non-tool-call remainder,
// omitted entirely when that remainder is empty (an assistant message
// that carries only tool calls and no text) — this package's own
// "presence-flag, omit when absent" convention (option.go's generation
// options, tool.go's declaration set, body.go's own tools/tool_choice
// fields), extended here rather than reinvented, in preference to
// fabricating a "content":null this package's own citation gate never
// settled (doc.go's "Tool calls render in their own tool_calls array"
// section). "tool_calls" comes from the same split's tool-call parts,
// rendered only when at least one exists.
//
// It is called once per ai.Message, unconditionally, from
// appendMessagesField's own loop (body.go) — the same loop AI-26.1
// wrote, unedited by this slice. Nothing here reads, remembers or
// compares against a previously rendered message: this function carries
// no state across calls, so a run of consecutive ai.Message values
// sharing one role.String() renders as that many distinct wire message
// objects, never fewer (R-ART-009). See doc.go's "No merging of
// consecutive same-role messages" section for the decision and its
// reason (S-ART-034).
func appendMessageObject(buf []byte, message ai.Message) []byte {
	if message.Role() == ai.RoleTool {
		return appendToolResultMessages(buf, message)
	}

	contentParts, toolCalls := splitToolCalls(message.Content())

	buf = append(buf, `{"role":`...)
	buf = appendJSONString(buf, message.Role().String())
	if len(contentParts) > 0 {
		buf = append(buf, `,"content":`...)
		buf = appendMessageContent(buf, contentParts)
	}
	if len(toolCalls) > 0 {
		buf = append(buf, `,"tool_calls":[`...)
		for i, call := range toolCalls {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = appendToolCallObject(buf, call)
		}
		buf = append(buf, ']')
	}
	return append(buf, '}')
}

// splitToolCalls partitions content into its non-tool-call parts (rest —
// rendered as the message's own "content") and its tool-call parts
// (calls — rendered separately, as the message's own "tool_calls" array,
// AI-26.5, R-ART-012/013). Both results preserve content's own original
// relative order, computed in one pass — no reordering, no grouping, no
// map (design.md "Map discipline"), extending the same ordering
// discipline R-ART-008 already established one level up for message and
// intra-message part order.
//
// A tool call is never a member of rest: message_test.go's own hand-off
// note (partKindDispositions, PartKindToolCall, recorded at AI-26.3)
// states a tool call renders as an element of a SEPARATE top-level
// tool_calls array on the assistant message object, never as a
// content-part array element — this function is where that split
// happens, once, so appendMessageContent/appendContentPartObject never
// need to know a tool-call part can occur in their own input at all.
func splitToolCalls(content []ai.Part) (rest []ai.Part, calls []ai.ToolCall) {
	for _, part := range content {
		if call, ok := part.ToolCall(); ok {
			calls = append(calls, call)
			continue
		}
		rest = append(rest, part)
	}
	return rest, calls
}

// appendToolCallObject appends one wire tool-call object:
// {"id":...,"type":"function","function":{"name":...,"arguments":...}} —
// claim 2's own citation (doc.go's wire-shape provenance section,
// ChatCompletionMessageToolCall) reused here rather than re-derived: the
// same {"type":"function","function":{...}} wrapper tool.go's
// appendToolObject (tool declarations) and option.go's
// appendToolChoiceField (a named tool_choice) already render for this
// vendor's other function-shaped wire objects.
//
// call.ID() is spliced through appendJSONString UNCHANGED — never
// generated, derived, normalized or truncated (R-ART-013). Doc 0002's
// amended AI-26.5 charter records that this vendor assigns tool-call
// identifiers on the call's own opening delta, so the synthetic-minting
// branch has no subject here; this adapter's only remaining obligation,
// replaying a call into a later turn's history, is carrying the
// identifier it was given straight back onto the wire, byte for byte.
//
// call.Arguments() is claim 2's own subject: the bytes are already
// well-formed JSON (ai.NewToolCall's own construction invariant,
// V-REQ-17), and claim 2 cites "arguments" as a wire STRING, not a
// nested object — the opposite splice direction from tool.go's schema
// (R-ART-010, a wire OBJECT, spliced RAW): appendJSONString here
// re-encodes those bytes AS a JSON string value, escaping them, which is
// what makes claim 2's encoding actually observable on the wire rather
// than merely asserted (tool_result_test.go's own interleaved
// multi-call case is gated on this claim for exactly that reason).
func appendToolCallObject(buf []byte, call ai.ToolCall) []byte {
	buf = append(buf, `{"id":`...)
	buf = appendJSONString(buf, call.ID())
	buf = append(buf, `,"type":"function","function":{"name":`...)
	buf = appendJSONString(buf, call.Name())
	buf = append(buf, `,"arguments":`...)
	buf = appendJSONString(buf, string(call.Arguments()))
	return append(buf, '}', '}')
}

// appendToolResultMessages appends one distinct wire tool-role message
// per ai.PartKindToolResult part in message's own content — R-ART-012's
// "one distinct wire message" contract, never a content-part array
// element on some other message (message_test.go's own hand-off note,
// partKindDispositions, PartKindToolResult, recorded at AI-26.3).
//
// message.Role() == ai.RoleTool licenses reading every part in
// message.Content() as a tool result without a further per-part kind
// check here: request.go's own rolePermittedKinds table restricts a
// RoleTool message to PartKindToolResult alone (V-REQ-01,
// roleAllowsKind), enforced by ai.NewRequest before Translate is ever
// reached — this function trusts that invariant the same way
// appendToolObject (tool.go) trusts ai.NewToolSet's own duplicate/name
// rules rather than re-checking them. A part that is somehow not a tool
// result anyway panics naming its kind, via the same
// unreadableContentPartKindPanic every other unreadable-kind site in
// this file uses, rather than silently skipping it or rendering
// something wrong.
//
// The common case is exactly one part: one tool answered, one wire
// message. A RoleTool message carrying more than one ToolResult part —
// which ai.NewMessage's own rules do not forbid — still renders
// correctly: one wire object per part, comma-joined, in that message's
// own content order, never merged into one wire object naming more than
// one tool_call_id (R-ART-012's "distinct" requirement, applied per
// result rather than per ai.Message).
func appendToolResultMessages(buf []byte, message ai.Message) []byte {
	for i, part := range message.Content() {
		result, ok := part.ToolResult()
		if !ok {
			panic(unreadableContentPartKindPanic(part.Kind()))
		}
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendToolResultObject(buf, result)
	}
	return buf
}

// appendToolResultObject appends one wire tool-role message:
// {"role":"tool","tool_call_id":...,"content":...} — AI-24 §6's given
// shape (R-ART-012), not re-derived here.
//
// result.CallID() is spliced through appendJSONString UNCHANGED
// (R-ART-013, S-ART-045/046): never generated, derived, normalized or
// truncated. It is also the whole of R-ART-014's correlation contract:
// this function performs no lookup, no matching and no positional
// reasoning of its own to produce the wire tool_call_id — it is
// literally result.CallID(), which the caller supplied when it
// constructed the ToolResult (ai.NewToolResult/ai.NewToolFailure), so
// correlation survives translation simply because nothing here has an
// opportunity to substitute anything else for it.
//
// result.Content() renders unconditionally, identically whether
// result.Failed() is true or false (S-ART-044): this package defines no
// failure-disposition wire field of its own. tool_result.go's own doc
// comment states a failing tool's answer is content the model reasons
// about, not a fault this package's taxonomy names, and AI-24 §6's given
// shape carries no field for it either. "Not silently rendered as
// success" is exactly what NOT branching on Failed() — rendering the
// caller's own content verbatim either way — produces: there is no
// separate "success" code path this function could fall back to
// (tool_result_test.go's own
// TestToolResult_FailedAndSucceededRenderIdentically proves the two
// cases differ in nothing but identity and content).
func appendToolResultObject(buf []byte, result ai.ToolResult) []byte {
	buf = append(buf, `{"role":"tool","tool_call_id":`...)
	buf = appendJSONString(buf, result.CallID())
	buf = append(buf, `,"content":`...)
	buf = appendJSONString(buf, result.Content())
	return append(buf, '}')
}

// appendMessageContent appends the "content" value for content — a
// message's own content parts with any tool-call parts already removed
// by the caller (appendMessageObject, via splitToolCalls); never called
// with an empty slice, since appendMessageObject only calls it when
// len(contentParts) > 0.
//
// Claim 1.0.5 (doc.go's wire-shape provenance section) documents content
// as oneOf[string, array]: exactly one part renders as a bare JSON
// string (the shape AI-26.1's skeleton already used and every earlier
// slice's expectation already commits to, preserved here byte for byte);
// more than one part renders as an array, one typed content-part object
// per part (appendContentPartObject, below), in that same order.
//
// content is already ordered the way the caller built it: splitToolCalls
// preserves relative order while filtering, and before that,
// message.Content() is already a caller-ordered, freshly cloned slice
// (message.go's own clone-on-read contract, package ai) — so ranging
// over it here in index order introduces no map of our own (design.md
// "Map discipline"), which is what makes both the single-part and the
// multi-part shape preserve intra-message part order structurally
// (R-ART-008) rather than by a separate ordering step.
//
// A single part of a kind this phase cannot read renders through the
// same panic appendContentPartObject's default case uses, naming the
// kind — see that function's own doc comment for why a panic, and
// message_test.go's partKindDispositions table for which phase owns the
// one remaining deferred kind (Reasoning) and what it must verify.
func appendMessageContent(buf []byte, content []ai.Part) []byte {
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
// used whenever a message's non-tool-call content carries more than one
// part.
//
// PartKindText is the only readable content-part variant this array
// shape ever renders (R-ART-007): {"type":"text","text":...}.
// PartKindToolCall never reaches this function at all: appendMessageObject
// removes every tool-call part via splitToolCalls before content is
// even computed, so a tool call is never a candidate content-part array
// element (AI-26.5, discharging R-ART-012's own hand-off from AI-26.3).
// PartKindToolResult never reaches this path either — a RoleTool message
// is intercepted whole by appendMessageObject's own dispatch, before
// content or its parts are ever inspected (appendToolResultMessages,
// above). The one member of ai.PartKinds() still deferred past this
// point is PartKindReasoning: AI-26.6's own refusal door (Phase 7,
// policy.go) intercepts it earlier still, before appendBody appends
// anything at all; this function's default case remains the transitional
// safety net for it in the meantime, exactly as message_test.go's
// partKindDispositions table (TestMessage_PartKindCoverage, S-ART-027)
// records.
//
// Reaching the default case panics, naming the kind via kind.String(),
// rather than silently rendering nothing or the wrong bytes — both of
// which R-ART-007 forbids by name ("none translates by accident").
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

// unreadableContentPartKindPanic builds the one panic message every
// unreadable-content-part-kind site in this file raises for a part it
// cannot read — one message format, so no two call sites can report the
// same fact differently.
func unreadableContentPartKindPanic(kind ai.PartKind) string {
	return "openaicompat: content part kind not readable at this phase (message.go): " + kind.String()
}
