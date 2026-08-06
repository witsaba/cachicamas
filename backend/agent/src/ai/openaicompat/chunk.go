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
// carries the full per-choice delta, byte-preserving. Usage is a pointer
// (AI-28.3, D10) so an absent key and an explicit JSON null both decode to
// nil identically (C4: "usage: null contributes nothing" makes no
// distinction between the two at the chunk level) while a populated usage
// object decodes to a non-nil *wireUsage.
type wireChunk struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Created *int64       `json:"created"`
	Choices []wireChoice `json:"choices"`
	Usage   *wireUsage   `json:"usage"`
	Object  string       `json:"object"`
}

// chunkObjectDiscriminator is C1's required object-field spelling for a
// recognized chunk (R-ATS-017, D3): "the chunk discriminator". A frame
// whose decoded object value is PRESENT and does not match this spelling
// is not recognized as a chunk at all and MUST be skipped (S-ATS-066),
// never treated as a malformed one — that distinction (a recognized
// shape that is broken) is R-ATS-021's charter, a later slice.
const chunkObjectDiscriminator = "chat.completion.chunk"

// isChunk reports whether c should be treated as a recognized streaming
// chunk (R-ATS-017, D3, final three-way rule): true only when Object is
// PRESENT and matches chunkObjectDiscriminator exactly. Object ABSENT is
// no longer treated as a chunk — the spec's own Definitions section makes
// the discriminator constitutive of "a chunk" in the first place, and
// R-ATS-017's own text ("whose top-level object is not chat.completion.chunk")
// is trivially true of an absent value too — so an object-absent frame is
// skipped exactly like a present-and-mismatched one (S-ATS-066).
//
// # Corrective: supersedes slice 5's own pragmatic deviation
//
// Slice 5 deliberately implemented this method as "absent OR matching" —
// disclosed as narrower than design.md D3's own literal "≠
// chat.completion.chunk (or absent) → skip" text — because none of this
// package's 176 pre-slice-5 fixtures set the object field at all, and the
// literal rule would have skipped every one of them, a safety-net
// regression. Slice 6's own precondition commit normalized every existing
// fixture to carry the field (C1 requires it: those fixtures were the
// defect, not the rule), discharging that conflict, so this method now
// implements D3's literal three-way rule in full: absent → skip
// (R-ATS-017 family); present-and-mismatched → skip (S-ATS-066); present
// and correct but some OTHER C1 required field missing is NOT this
// method's concern — see wireChunk.hasRequiredFields, R-ATS-021,
// S-ATS-081 — that is a recognized-but-broken chunk, never a skip.
func (c wireChunk) isChunk() bool {
	return c.Object == chunkObjectDiscriminator
}

// hasRequiredFields reports whether c, already recognized as a chunk by
// isChunk(), nonetheless carries every C1 required top-level field
// (choices, created, id, model, object — R-ATS-021, S-ATS-081, verify-
// report W1). The five-field set is discharged five different ways, not
// five identical checks: Model and Created are genuinely re-validated
// here directly, because no other code path re-reads either one past the
// first chunk; id, object and choices are each covered by a DIFFERENT
// mechanism, documented below, so repeating them here would be redundant
// with a rule that already closes the same gap.
//
// Model is checked because R-ATS-004's own identity path
// (ai.NewResponseStart) only ever runs once, on the chunk that
// establishes identity, so a LATER chunk's empty or absent model would
// otherwise pass through unchecked. Created is checked for the identical
// reason: no code path anywhere in this package reads it at all, so
// nothing else could ever notice its absence — wireChunk declared no
// Created field until this check needed one.
//
// ID is deliberately NOT re-checked here: an empty/absent id on the FIRST
// chunk is already rejected by ai.NewResponseStart's own constructor
// (S-ATS-015, errMalformedIdentity); on a LATER chunk, R-ATS-020 row 3's
// own established-identity comparison (stream_state.go) already rejects an
// id that differs from the one already established — including the empty
// string, which can never equal a non-empty established id — so a second,
// independent id-presence check here would be redundant with that rule,
// not a gap it leaves open.
//
// Object is deliberately NOT re-checked here either: isChunk() — the
// caller's own precondition for ever calling this method — already
// requires Object to be present and correct (R-ATS-017/D3); by the time
// this method runs, Object has already been proven both present and
// exactly chunkObjectDiscriminator, so a second check here could never
// fail.
//
// Choices' own required-key presence is deliberately not probed here: an
// empty choices array is C4's own legitimate usage-chunk shape, and Go's
// zero-value decode cannot distinguish "the key was absent" from "the key
// was present as []" without an unwarranted raw-presence probe no scenario
// in this node asks for.
func (c wireChunk) hasRequiredFields() bool {
	return c.Model != "" && c.Created != nil
}

// wireUsage is the streaming usage object's field set this milestone maps
// (AI-28.3, C8, D10): prompt_tokens → Usage.Input, completion_tokens →
// Usage.Output. Both are pointers so a present-but-explicit-0 value
// (Tokens(0), a reported zero) is distinguishable from an absent key
// (TokenCount's zero value, R-ATS-015/AI-13.3) — the same absent/null/
// present-string trichotomy contentText resolves for delta.content, here
// resolved by json.Unmarshal's own pointer-nil-on-absence behavior instead
// of a hand-rolled decoder, since C8's prompt_tokens/completion_tokens are
// plain JSON numbers with no byte-preservation concern like content has.
//
// # AI-31.2 — Two new typed detail structs (R-ACP-005, design D-A)
//
// C8's nested detail objects (prompt_tokens_details with cached_tokens
// and cache_write_tokens; completion_tokens_details with reasoning_tokens)
// join wireUsage as POINTER FIELDS, so an absent key on the parent
// (no detail object at all) and a present parent whose leaf key is
// absent both decode to nil on the leaf — the same absent/null/present
// trichotomy the landed prompt_tokens/completion_tokens use, extended
// one level deep. Leaf fields stay *int64 for the same absent-vs-zero
// reason (S-ACP-013's CacheRead discrimination extends naturally).
//
// total_tokens (C8's third required field) is deliberately not decoded
// here: ai.Usage has no corresponding field, and AI-13.4's cost formula
// does not want one. See the un-attested-exclusivity record on
// usageFromWire below (R-ACP-006, D-C).
type wireUsage struct {
	PromptTokens            *int64                       `json:"prompt_tokens"`
	CompletionTokens        *int64                       `json:"completion_tokens"`
	PromptTokensDetails     *wirePromptTokensDetails     `json:"prompt_tokens_details"`
	CompletionTokensDetails *wireCompletionTokensDetails `json:"completion_tokens_details"`
}

// wirePromptTokensDetails is the C8 nested object on the prompt side.
// Every leaf is a pointer so absent-key (including absent parent) and
// present-but-explicit-0 stay distinguishable (S-ACP-013, S-ACP-021).
type wirePromptTokensDetails struct {
	CachedTokens     *int64 `json:"cached_tokens"`
	CacheWriteTokens *int64 `json:"cache_write_tokens"`
}

// wireCompletionTokensDetails is the C8 nested object on the completion
// side. reasoning_tokens is a pointer for the same absent-vs-zero
// discipline (S-ACP-013-style).
type wireCompletionTokensDetails struct {
	ReasoningTokens *int64 `json:"reasoning_tokens"`
}

// usageFromWire maps w's fields onto an ai.Usage record (R-ATS-015/016,
// R-ACP-005…007, D10): present() only when the wire key was present —
// including an explicit 0 — else the field stays TokenCount's absent zero
// value.
//
// # AI-31.2 — Three new raw mappings (R-ACP-005, design D-D)
//
//   - cached_tokens → CacheRead       (raw, no arithmetic)
//   - cache_write_tokens → CacheWrite (raw, no arithmetic)
//   - reasoning_tokens → Reasoning    (raw, no arithmetic)
//
// Every new mapping is RAW: the vendor's number is copied into the
// neutral field without adjustment. The landed prompt_tokens → Input /
// completion_tokens → Output mappings are a SEPARATE, FROZEN fact
// (R-ACP-009, byte-identical tests) — this change touches neither.
//
// # AI-31.2 — Un-attested exclusivity record (R-ACP-006, S-ACP-016, design D-C)
//
// AI-13.4's operative sentence, quoted verbatim from
// backend/agent/src/ai/usage.go (~L112):
//
//	"Input is the tokens of the request the provider processed fresh,
//	 excluding everything counted in CacheRead and CacheWrite."
//
// and the immediately-following clause: "This record is exclusive, so an
// adapter for an inclusive vendor subtracts before it fills this field
// in."
//
// U1 and U2 (this change's citations.md) are SILENT on whether the
// wire's prompt_tokens is inclusive of cached_tokens and cache_write_tokens.
// The chat schema's only usage-arithmetic sentence (the
// rejected_prediction_tokens description, U3) settles completion_tokens ⊇
// reasoning_tokens and nothing else; no chat-scope prose establishes
// prompt_tokens arithmetic. Subtracting on documented silence would be
// inference recorded as fact (R-ATS-027), so this adapter:
//
//  1. maps prompt_tokens → Usage.Input as the vendor's RAW value,
//  2. maps cached_tokens → CacheRead and cache_write_tokens → CacheWrite
//     as the vendor's RAW values,
//  3. performs NO arithmetic — in particular no subtraction of
//     cached_tokens or cache_write_tokens from Input, and
//  4. records here that THIS ADAPTER CANNOT YET ATTEST AI-13.4's
//     exclusivity relation for Input on this dialect: a consumer
//     summing Input + CacheRead + CacheWrite may therefore double-count
//     on this dialect, and that limitation is named in code (this
//     comment) and in spec (R-ACP-006) rather than papered over.
//
// The discharge obligation — pinning the relation against a real
// cache-hit transcript — is routed to AI-38.2's expected-vs-generated
// capability report, where declared-capability standing is decided.
// S-ACP-017's impossible-arithmetic probe (prompt_tokens: 500 with
// cached_tokens: 800 → Input = 500, CacheRead = 800 raw, no error or
// clamp) proves the stronger claim that NO consistency arithmetic is
// enforced anywhere on the path.
func usageFromWire(w wireUsage) ai.Usage {
	var u ai.Usage
	if w.PromptTokens != nil {
		u.Input = ai.Tokens(*w.PromptTokens)
	}
	if w.CompletionTokens != nil {
		u.Output = ai.Tokens(*w.CompletionTokens)
	}
	if d := w.PromptTokensDetails; d != nil {
		if d.CachedTokens != nil {
			u.CacheRead = ai.Tokens(*d.CachedTokens)
		}
		if d.CacheWriteTokens != nil {
			u.CacheWrite = ai.Tokens(*d.CacheWriteTokens)
		}
	}
	if d := w.CompletionTokensDetails; d != nil {
		if d.ReasoningTokens != nil {
			u.Reasoning = ai.Tokens(*d.ReasoningTokens)
		}
	}
	return u
}

// wireChoice is one choice item's minimal shape this slice reads: the
// delta object, finish_reason (via a pointer so JSON null and an absent
// key both decode to nil, matching C2's own null-until-terminal behavior),
// and Index (C2: index is a per-choice required field alongside delta and
// finish_reason; R-ATS-020 row 4, slice 6): also a pointer, so an absent
// index key is distinguishable from an explicit 0 — the same absent/present
// pattern wireUsage's own fields already use. This milestone maps choice 0
// only (spec.md's own "Definitions" section).
type wireChoice struct {
	Delta        wireDelta `json:"delta"`
	FinishReason *string   `json:"finish_reason"`
	Index        *int      `json:"index"`
}

// hasValidIndex reports whether c carries a present, non-negative index
// (C2's own required per-choice field; R-ATS-020 row 4: "negative or
// absent" both violate). Only consulted when c also carries a choice-0
// content string — a role-only or delta-less choice item's own index is
// not this row's concern, per the row's own wire-shape text ("...while
// carrying content").
func (c wireChoice) hasValidIndex() bool {
	return c.Index != nil && *c.Index >= 0
}

// wireDelta is choice 0's delta object, byte-preserving (D2): Content is
// decoded as json.RawMessage rather than a plain string, so this file — not
// encoding/json's own string decode — decides how an invalid-UTF-8
// fragment is handled.
//
// ToolCalls (AI-30.1, R-ATL-001, C9.1) is the streaming tool-call element
// array. Its element is ChatCompletionMessageToolCallChunk: index (the
// ONLY required field, C9.1), id, and function{name, arguments}. type is
// the single-member enum "function" (C9.4) and carries no mapping decision,
// so wireDelta declares no Type field — the spec's own "decoded to nothing
// else" reading of C9.1 (design D2).
//
// The deprecated streaming function_call delta (C9.5, C7) is a recorded
// skip (R-ATL-013): no FunctionCall field is declared here, so
// encoding/json's own unknown-field tolerance (R-ATS-017) silently drops
// it; the disposition is "tolerated, ignored, never mapped" — never a
// silent defect. Reopen trigger (R-ATL-013 / S-ATL-064): if a real backend
// is proven to emit function_call exclusively with no tool_calls array,
// this disposition is reopened as its own change, never patched in
// silently here.
type wireDelta struct {
	Content   json.RawMessage       `json:"content"`
	ToolCalls []wireToolCallElement `json:"tool_calls"`
}

// wireToolCallElement is one item of choice 0's tool_calls array
// (R-ATL-001, C9.1: ChatCompletionMessageToolCallChunk). Index is a
// pointer so an absent key decodes to nil (a C9.1 required-list violation,
// S-ATL-011) — distinguishable from an explicit zero, which the spec
// treats as a legal wire value (S-ATL-008).
//
// ID, Function.Name and Function.Arguments are all json.RawMessage
// rather than plain Go strings: every byte a non-empty wire value carries
// must round-trip through NewToolCallStart and the per-call byte buffer
// byte-exact (R-ATL-001, R-ATL-003), and encoding/json's own string decode
// substitutes U+FFFD for non-UTF-8 sequences — the same reasoning D2
// already applied to delta.content. unquoteJSONString (below) is the one
// decoder — reused, never a second (S-ATL-005).
type wireToolCallElement struct {
	Index    *int                  `json:"index"`
	ID       json.RawMessage       `json:"id"`
	Function *wireToolCallFunction `json:"function"`
}

// wireToolCallFunction is the nested {"function":{...}} half of a tool-call
// element (C9.1). Both Name and Arguments are json.RawMessage so neither
// is touched by encoding/json's own string decode — D2's byte preservation
// applies here too.
type wireToolCallFunction struct {
	Name      json.RawMessage `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
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
//
// # AI-31.1 — Unreachable neutral values on this dialect (R-ACP-002, S-ACP-004)
//
// Three members of ai.FinishReason's seven-value closed vocabulary cannot
// be produced from any wire finish_reason on this dialect. They are
// enumerated here at the strict gate (the single decision point that
// decides what reaches a completion) so a reviewer finds the unreachability
// where the gate actually rejects, not in a doc far from the code:
//
//	FinishReasonRefusal  — U5 NEGATIVE: no chat finish_reason member spells
//	                       refusal; delta.refusal (C7) is the only
//	                       refusal-shaped channel and its companion
//	                       finish_reason is undocumented. Reopens when the
//	                       pinned dialect gains a refusal finish member, or
//	                       AI-38 pins the companion value from a real
//	                       transcript (D2 ruling (a)).
//	FinishReasonPauseTurn — U5 NEGATIVE: no pause channel exists in chat
//	                       scope (pause/paused hits are all fine-tuning
//	                       endpoints). AI-31.1's pause-resume lossiness
//	                       Note is vacuous for this adapter. Reopens when
//	                       the pinned dialect gains a pause finish member.
//	FinishReasonUnknown  — Unreachable by design, not by omission: the
//	                       strict gate below rejects an out-of-enum value
//	                       as a typed malformed response (S-ATS-039), so
//	                       an unrecognised stop value never becomes a
//	                       completion at all. Reopens only with a
//	                       deliberate reversal of D1.
//
// See spec R-ACP-002 / S-ACP-004 in
// openspec/changes/cachicamas-ai-provider-completion/specs/ai-provider-completion/spec.md
// for the citation-anchored table each row above summarizes.
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
