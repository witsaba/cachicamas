package openaicompat

import (
	"encoding/json"

	"github.com/cachicamas/backend/agent/src/ai"
)

// appendBody hand-assembles req's wire body into one contiguous byte
// slice, in one fixed source-code order: model, messages, tools (this
// slice's own field, AI-26.4, present only when the request declares at
// least one — appendToolsField, tool.go), [tool_choice, generation
// options — later slices splice these in here], stream/stream_options,
// [provider-extension members — later slices splice these in last]
// (design.md "Data Flow").
//
// This is deliberately NOT struct-marshalled through encoding/json — see
// doc.go's wire-shape provenance section, claim 3: json.Marshal pipes
// every json.Marshaler result, including json.RawMessage, through
// encoding/json's own internal compaction and HTML-escaping
// (encoding/json/encode.go:483-488, indent.go:51), which would silently
// rewrite the verbatim tool-schema bytes R-ART-010 requires to pass
// through unmodified. Every string LEAF here still goes through
// appendJSONString (json.Marshal's deterministic escaping); only a
// would-be json.RawMessage-shaped value is ever spliced raw, and this
// slice introduces none.
//
// Field order is source order, never iteration order: ranging over a map
// anywhere on this path would make cross-run determinism probabilistic
// rather than structural (R-ART-003, design.md "Map discipline"), and
// nothing below ranges over one.
func appendBody(req ai.Request) []byte {
	buf := make([]byte, 0, 256)
	buf = append(buf, '{')
	buf = appendModelField(buf, req)
	buf = append(buf, ',')
	buf = appendMessagesField(buf, req)
	if tools, hasTools := req.Tools(); hasTools {
		buf = append(buf, ',')
		buf = appendToolsField(buf, tools)
	}
	// Later slices splice tool_choice and generation options (26.7) in
	// here, between tools and stream, in that fixed order.
	buf = append(buf, ',')
	buf = appendStreamFields(buf)
	// Later slices splice this adapter's own provider-extension namespace
	// members in here, last (26.7).
	buf = append(buf, '}')
	return buf
}

// appendModelField appends `"model":"<value>"`, the model identity taken
// verbatim from req (R-ART-002).
func appendModelField(buf []byte, req ai.Request) []byte {
	buf = append(buf, `"model":`...)
	return appendJSONString(buf, req.Model())
}

// appendMessagesField appends `"messages":[...]`. Each of req's
// system-instruction segments renders first, one wire message per
// segment (system.go's appendSystemMessageObject, AI-26.2, R-ART-005),
// followed by one rendered object per req.Messages(), in caller order —
// both SystemInstruction.Segments() and Messages() are already
// caller-ordered slices, so this introduces no map (design.md "Map
// discipline"). A request with no system instruction (S-ART-020)
// contributes no system message at all — wrote tracks whether anything
// has been appended yet, rather than assuming messages is the sole
// source of entries, so the join works whether or not a system
// instruction is present.
//
// This skeleton renders exactly the shape R-ART-002's minimal request
// needs: one role name plus one text-part content string. Every other
// role and every other content-part variant is AI-26.3's (message.go,
// slice 5), which replaces appendMessageObject and
// appendSingleTextContent outright; this is not the milestone's full
// message rendering.
func appendMessagesField(buf []byte, req ai.Request) []byte {
	buf = append(buf, `"messages":[`...)
	wrote := false
	if system, hasSystem := req.SystemInstruction(); hasSystem {
		for _, segment := range system.Segments() {
			if wrote {
				buf = append(buf, ',')
			}
			buf = appendSystemMessageObject(buf, segment)
			wrote = true
		}
	}
	for _, message := range req.Messages() {
		if wrote {
			buf = append(buf, ',')
		}
		buf = appendMessageObject(buf, message)
		wrote = true
	}
	buf = append(buf, ']')
	return buf
}

// appendMessageObject appends one wire message object for message.
func appendMessageObject(buf []byte, message ai.Message) []byte {
	buf = append(buf, `{"role":`...)
	buf = appendJSONString(buf, message.Role().String())
	buf = append(buf, `,"content":`...)
	buf = appendSingleTextContent(buf, message)
	return append(buf, '}')
}

// appendSingleTextContent appends message's content as a JSON string —
// one of the two shapes the vendor's "Create chat completion" reference
// documents for a message's content field, the other being an array of
// typed content parts (doc.go's wire-shape provenance section, task
// 1.0.5). A plain string is what this skeleton's one-text-part case
// renders.
//
// Slice 1's own scenarios construct only a single text part per message;
// every other content shape is AI-26.3's (message.go, slice 5), which
// replaces this function outright rather than growing a case list here.
func appendSingleTextContent(buf []byte, message ai.Message) []byte {
	content := message.Content()
	if len(content) == 1 {
		if text, ok := content[0].Text(); ok {
			return appendJSONString(buf, text)
		}
	}
	panic("openaicompat: content shape not supported before AI-26.3 (message.go)")
}

// appendStreamFields appends
// `"stream":true,"stream_options":{"include_usage":true}`.
//
// Both are unconditional from this skeleton onward (R-ART-017). AI-24
// §§8, 13.1 assign the total positive assertion to AI-26.7, but the bytes
// themselves are emitted here so no later slice rewrites every earlier
// expectation literal (design.md "Usage opt-in placement"). Neither value
// is caller-controlled, so neither needs appendJSONString: both are fixed
// boolean literals, not text that could need escaping.
func appendStreamFields(buf []byte) []byte {
	return append(buf, `"stream":true,"stream_options":{"include_usage":true}`...)
}

// appendJSONString appends s, JSON-string-encoded via json.Marshal's
// deterministic escaping (design.md "Wire-body representation").
//
// The error return is unreachable for a string: json.Marshal only fails
// for an unsupported Go type (a channel, a function, a cyclic pointer, or
// a broken Marshaler), none of which a string value can be.
func appendJSONString(buf []byte, s string) []byte {
	encoded, err := json.Marshal(s)
	if err != nil {
		panic("openaicompat: json.Marshal(string) failed unexpectedly: " + err.Error())
	}
	return append(buf, encoded...)
}
