// AI-26.7 — every neutral generation option maps to its vendor field
// (R-ART-016); the usage opt-in's bytes were already emitted at slice 1
// (translation.go/body.go) — this node owns their total, positive
// assertion (R-ART-017, option_test.go); the output-token limit's
// mandatory-default branch is a deliberate no-op for this vendor
// (R-ART-018, doc.go).
//
// The escape hatch (R-ART-019) is NOT implemented in this file — see
// doc.go's "Escape hatch: reserved namespace not yet defined upstream"
// section and tasks.md's Phase 4 evidence log.

package openaicompat

import (
	"encoding/json"

	"github.com/cachicamas/backend/agent/src/ai"
)

// appendGenerationOptionFields appends this request's max_tokens,
// temperature, top_p and stop fields, each independently gated on its own
// presence flag (R-ART-016): an unset option renders no field at all,
// never a zero value, and a deliberately zero-valued option still
// renders (S-ART-058, option_test.go's
// TestOption_ZeroValueVersusUnset_TranslatesDifferently).
//
// Field order here is fixed source order, matching body.go's own
// declared order (model, messages, tools, tool_choice, THIS group,
// stream/stream_options, extension members) — a fourth, independent
// instance of the same map-discipline reasoning translation.go and
// body.go already cite: nothing below ranges over anything, so there is
// no order to leak.
//
// max_tokens, temperature, top_p and stop are this vendor's own standard
// Chat Completions fields. Unlike the four wire-shape claims doc.go's
// provenance section cites under R-ART-001's gate, none of these four
// field names carries a comparable, previously-uncited ambiguity:
// R-ART-001 scopes its citation gate to exactly those four claims, and
// "max_tokens" specifically is already named directly in this vendor's
// own decision record (cachicamas-ai-first-provider-decision/design.md,
// row A4: "the mandatory-output-limit branch is a deliberate no-op ...
// (max_tokens optional)"). temperature/top_p/stop need no citation beyond
// that same decision's general dialect selection: they are the dialect's
// unchanged, standard field names, carrying none of the placement/role/
// encoding/alternation ambiguity each of the four gated claims turned on.
func appendGenerationOptionFields(buf []byte, req ai.Request) []byte {
	if tokens, has := req.MaxOutputTokens(); has {
		buf = append(buf, ',')
		buf = append(buf, `"max_tokens":`...)
		buf = appendJSONInt(buf, tokens)
	}
	if temperature, has := req.Temperature(); has {
		buf = append(buf, ',')
		buf = append(buf, `"temperature":`...)
		buf = appendJSONFloat64(buf, temperature)
	}
	if topP, has := req.TopP(); has {
		buf = append(buf, ',')
		buf = append(buf, `"top_p":`...)
		buf = appendJSONFloat64(buf, topP)
	}
	if sequences, has := req.StopSequences(); has {
		buf = append(buf, ',')
		buf = appendStopField(buf, sequences)
	}
	return buf
}

// appendToolChoiceField appends "tool_choice":<value>.
//
// A mode carrying no name — Auto, None or Required — renders as that
// mode's own bare wire string: [ai.ToolChoiceMode.String] already returns
// exactly the three vendor-accepted values "auto", "none" and "required"
// (tool_choice.go's own registered table), so this needs no separate
// switch of its own. A mode carrying a name (Specific) renders as
// {"type":"function","function":{"name":...}} — the same
// {"type":"function","function":{...}} wrapper tool.go's
// appendToolObject already renders tool declarations with, reused here
// rather than re-derived, for the vendor's one documented way to name a
// function object.
func appendToolChoiceField(buf []byte, choice ai.ToolChoice) []byte {
	buf = append(buf, `"tool_choice":`...)
	if name, ok := choice.Name(); ok {
		buf = append(buf, `{"type":"function","function":{"name":`...)
		buf = appendJSONString(buf, name)
		return append(buf, '}', '}')
	}
	return appendJSONString(buf, choice.Mode().String())
}

// appendStopField appends "stop":[...], one JSON string per sequence, in
// caller order. sequences is already a caller-ordered, freshly cloned
// slice (Request.StopSequences' own clone-on-read contract), so ranging
// over it here introduces no map (design.md "Map discipline").
func appendStopField(buf []byte, sequences []string) []byte {
	buf = append(buf, `"stop":[`...)
	for i, s := range sequences {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendJSONString(buf, s)
	}
	return append(buf, ']')
}

// appendJSONInt appends n as a JSON number via json.Marshal. Like
// appendJSONString (body.go), the error return is unreachable by
// construction: no int value can make json.Marshal fail.
func appendJSONInt(buf []byte, n int) []byte {
	encoded, err := json.Marshal(n)
	if err != nil {
		panic("openaicompat: json.Marshal(int) failed unexpectedly: " + err.Error())
	}
	return append(buf, encoded...)
}

// appendJSONFloat64 appends f as a JSON number via json.Marshal — the
// same deterministic, shortest round-trippable formatting every Go
// encoder uses. The error return is reachable only for NaN or an
// infinite value; neither is rejected by request.go's own boundsRule
// (which bounds temperature and top_p by ordinary numeric comparison, not
// by finiteness), so this panic is a documented, unexercised safety net
// rather than a verified-unreachable path the way appendJSONInt's is —
// consistent with this package's existing posture of not duplicating
// AI-10's own construction-time validation.
func appendJSONFloat64(buf []byte, f float64) []byte {
	encoded, err := json.Marshal(f)
	if err != nil {
		panic("openaicompat: json.Marshal(float64) failed unexpectedly: " + err.Error())
	}
	return append(buf, encoded...)
}
