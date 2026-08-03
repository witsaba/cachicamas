// AI-26.4 — tool declarations translate byte-faithfully, in the caller's
// declaration order (R-ART-010, R-ART-011).

package openaicompat

import "github.com/cachicamas/backend/agent/src/ai"

// appendToolsField appends `"tools":[...]`, one wire tool object per
// declared ai.Tool, in the caller's declaration order (R-ART-011).
//
// ai.ToolSet.Tools() is already a caller-ordered slice — tool_set.go's own
// "no map" design (V-REQ-14) — so ranging over it here and appending in
// that same order introduces no map of our own either (design.md "Map
// discipline"). This is the only place this function ever decides an
// order: nothing here sorts, groups, keys or otherwise reorders by name.
// A map-keyed intermediate anywhere on this path would turn the wire
// order from structural into merely probabilistic across independent
// process runs — see tool_test.go's fresh-process proof and its own
// staged map-mutation red for why that failure mode produces no error and
// no wrong answer, only a silently invalidated vendor cache prefix on
// every call.
func appendToolsField(buf []byte, tools ai.ToolSet) []byte {
	buf = append(buf, `"tools":[`...)
	for i, tool := range tools.Tools() {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendToolObject(buf, tool)
	}
	return append(buf, ']')
}

// appendToolObject appends one wire tool declaration:
// {"type":"function","function":{"name":...,"description":...,"parameters":<schema>}}.
//
// The {"type":"function","function":{...}} wrapper and the "parameters"
// field name both come from claim 3's own citation (doc.go's wire-shape
// provenance section: "ChatCompletionTool.function resolves to
// FunctionObject" — the same schema object whose "parameters" field this
// splices), so this needs no separate citation — exactly as system.go's
// message shape reused claim 1 and claim 4 without a new one.
//
// tool.Schema() bytes are appended RAW: never through appendJSONString and
// never through any decode/re-encode round trip (R-ART-010). Claim 3's own
// text — FunctionParameters is "type: object", "additionalProperties:
// true", "described as a JSON Schema object" — licenses passing a neutral
// schema value straight through, with no vendor-owned narrowing to marshal
// it through. Decoding it first (into a map, in particular) and
// re-marshalling would not merely lose whitespace: encoding/json.Marshal
// sorts a map's keys alphabetically on the way back out, silently
// reordering the schema's own keys too — body.go's citation of
// encoding/json's internal appendCompact (encode.go:483-488, indent.go:51)
// is the general finding; this is its concrete cost for a tool's schema
// specifically.
//
// tool.Description() is rendered unconditionally, even when empty. Unlike
// a generation option (option.go, AI-26.7), which tracks a separate
// "has*" flag to distinguish "the caller never set this" from "the caller
// set it to the zero value", ai.Tool carries no such flag for its
// description: Description() "may be empty" is the whole contract (its
// own doc comment, package ai) — there is no caller intent beyond the
// string value itself to preserve, so always rendering it keeps this
// function total, with no branch keyed on string emptiness.
func appendToolObject(buf []byte, tool ai.Tool) []byte {
	buf = append(buf, `{"type":"function","function":{"name":`...)
	buf = appendJSONString(buf, tool.Name())
	buf = append(buf, `,"description":`...)
	buf = appendJSONString(buf, tool.Description())
	buf = append(buf, `,"parameters":`...)
	buf = append(buf, tool.Schema()...)
	return append(buf, '}', '}')
}
