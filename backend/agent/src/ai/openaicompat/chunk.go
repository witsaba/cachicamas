// AI-28.1.2 — chunk decode: the wire chunk's shape, and the two hand-owned
// byte-level decisions this dialect needs (design.md D2, D5).
//
// # Why json.RawMessage, not a plain string field (D2)
//
// delta.content is decoded into json.RawMessage, not a plain Go string:
// encoding/json's own string decode substitutes U+FFFD for a byte sequence
// that is not well-formed UTF-8 (validator-proved by execution, go1.26.3),
// which would repair exactly the split-rune fragments R-ATS-009 forbids
// repairing. json.RawMessage's own decode performs no such substitution —
// it captures the JSON string token's bytes verbatim — so unquoteJSONString
// (below) is what turns that raw token into the fragment's own bytes,
// preserving whatever came in, valid UTF-8 or not.
//
// # The content trichotomy (R-ATS-007, C7)
//
// C7 makes delta.content optional AND nullable — three distinguishable
// wire shapes: the key absent, the key present with JSON null, and the key
// present with a JSON string (including the empty string). contentText
// resolves this trichotomy: present is false for the first two shapes
// alike, and true — even for an empty fragment — only for the third.
//
// # Raw-string-strict finish-reason gate (R-ATS-010, D5/N-6)
//
// rawStrictFinishReason checks the wire's raw string against C2's five
// lower-case enum members byte-exactly, before any normalization: no trim,
// no case fold. A value outside the enum — including a case variant of a
// legal member, such as "STOP" — is rejected here even though
// ai.NormalizeFinishReason would happily map it; that leniency belongs to
// AI-31.1's cross-vendor table, not this dialect's own schema-declared
// constant set (R-ATS-027: accepting an unlisted spelling here would be
// recording inference as fact).

package openaicompat

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"

	"github.com/cachicamas/backend/agent/src/ai"
)

// wireChunk is this dialect's streaming chunk shape (C1): the fields
// R-ATS-007…010 need to map choice 0's content and finish reason. Unlike
// stream.go's slice-1-scoped wireChunk (superseded by this type), Choices
// carries the full per-choice delta, byte-preserving.
type wireChunk struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []wireChoice `json:"choices"`
}

// wireChoice is one choice item's minimal shape this slice reads: the
// delta object and finish_reason, via a pointer so JSON null and an absent
// key both decode to nil, matching C2's own null-until-terminal behavior.
// This milestone maps choice 0 only (spec.md's own "Definitions" section);
// no index field is read here — R-ATS-020 row 4's index validation is
// slice 6's own scope.
type wireChoice struct {
	Delta        wireDelta `json:"delta"`
	FinishReason *string   `json:"finish_reason"`
}

// wireDelta is choice 0's delta object, byte-preserving (D2): Content is
// decoded as json.RawMessage rather than a plain string, so this file — not
// encoding/json's own string decode — decides how an invalid-UTF-8
// fragment is handled.
type wireDelta struct {
	Content json.RawMessage `json:"content"`
}

// decodeChunk parses data as one wireChunk. A JSON syntax error is
// returned to the caller unmodified; stream.go maps it onto this package's
// own malformed-response cause (unchanged from slice 1's own handling —
// distinguishing a syntactically-invalid chunk from other malformed shapes
// is R-ATS-021's charter, a later slice).
func decodeChunk(data []byte) (wireChunk, error) {
	var chunk wireChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return wireChunk{}, err
	}
	return chunk, nil
}

// choice0 returns the chunk's choice-0 item, and whether one exists. An
// empty choices array (C4's usage chunk) reports false, never a zero-value
// choice mistaken for a real one.
func (c wireChunk) choice0() (wireChoice, bool) {
	if len(c.Choices) == 0 {
		return wireChoice{}, false
	}
	return c.Choices[0], true
}

// jsonNull is the literal 4-byte JSON null token, compared byte-exactly
// against a RawMessage to resolve the content trichotomy's second shape.
var jsonNull = []byte("null")

// contentText resolves R-ATS-007's content trichotomy (C7): raw is nil or
// zero-length when the "content" key was absent from the delta object;
// the 4-byte literal "null" when the key was JSON null; otherwise a quoted
// JSON string, unquoted byte-preservingly (D2, unquoteJSONString below).
// present is false for the first two shapes — a null or absent fragment is
// never read as the empty string — and true, with text possibly "", only
// for an actual JSON string value (R-ATE-008, S-ATS-027).
func contentText(raw json.RawMessage) (text string, present bool) {
	if len(raw) == 0 {
		return "", false
	}
	if bytes.Equal(raw, jsonNull) {
		return "", false
	}
	return string(unquoteJSONString(raw)), true
}

// unquoteJSONString decodes one JSON string literal's raw bytes — including
// its surrounding quotes — into the string's content bytes, preserving
// every non-escape byte verbatim and decoding only JSON's own backslash
// escapes (D2, R-ATS-009). It never validates, repairs or substitutes for
// invalid UTF-8: a byte sequence that is not well-formed UTF-8 passes
// through unchanged, which is the entire reason this function exists
// rather than a plain Go string field.
//
// raw is expected to already be one complete, syntactically valid JSON
// string token, as delivered by json.RawMessage from a successful
// json.Unmarshal — at least two bytes, starting and ending with '"'. A
// shorter or unquoted input is returned unmodified rather than panicking,
// since json.Unmarshal already rejected anything else before this function
// is ever reached.
func unquoteJSONString(raw []byte) []byte {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return append([]byte(nil), raw...)
	}
	body := raw[1 : len(raw)-1]
	out := make([]byte, 0, len(body))
	for i := 0; i < len(body); i++ {
		b := body[i]
		if b != '\\' || i+1 >= len(body) {
			out = append(out, b)
			continue
		}
		i++
		out, i = appendUnescaped(out, body, i)
	}
	return out
}

// appendUnescaped decodes the single escape sequence starting at body[i]
// (the byte immediately after the backslash) and appends its decoded bytes
// to out, returning the updated slice and the index of the escape's last
// consumed byte. An unrecognized or truncated escape passes its own literal
// bytes through rather than silently dropping them — this decoder repairs
// nothing (R-ATS-009).
func appendUnescaped(out, body []byte, i int) ([]byte, int) {
	switch esc := body[i]; esc {
	case '"', '\\', '/':
		return append(out, esc), i
	case 'b':
		return append(out, '\b'), i
	case 'f':
		return append(out, '\f'), i
	case 'n':
		return append(out, '\n'), i
	case 'r':
		return append(out, '\r'), i
	case 't':
		return append(out, '\t'), i
	case 'u':
		if i+4 < len(body) {
			if v, ok := parseHex4(body[i+1 : i+5]); ok {
				return utf8.AppendRune(out, rune(v)), i + 4
			}
		}
		return append(out, '\\', 'u'), i
	default:
		return append(out, '\\', esc), i
	}
}

// parseHex4 decodes exactly 4 hex digits into their numeric value, and
// whether every byte was a legal hex digit.
func parseHex4(b []byte) (uint16, bool) {
	var v uint16
	for _, c := range b {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= uint16(c - '0')
		case c >= 'a' && c <= 'f':
			v |= uint16(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v |= uint16(c-'A') + 10
		default:
			return 0, false
		}
	}
	return v, true
}

// finishReasonEnum is C2's five raw wire spellings, and only those five —
// matched byte-exactly by a plain map lookup, with no trim and no case
// fold (D5, N-6).
var finishReasonEnum = map[string]bool{
	"stop":           true,
	"length":         true,
	"tool_calls":     true,
	"content_filter": true,
	"function_call":  true,
}

// rawStrictFinishReason checks raw against C2's enum byte-exactly and, only
// when it is a member, normalizes it to the canonical ai.FinishReason
// (R-ATS-010, D5). ok is false for anything outside the enum — including a
// case or whitespace variant of a legal member — even though
// ai.NormalizeFinishReason alone would accept it; that leniency is
// deliberately not consulted for the accept/reject decision here, only for
// the mapping once accepted.
func rawStrictFinishReason(raw string) (ai.FinishReason, bool) {
	if !finishReasonEnum[raw] {
		return 0, false
	}
	return ai.NormalizeFinishReason(raw), true
}
