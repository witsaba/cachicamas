// AI-09.3 — the tool result: the answer to a tool call.
//
// This file implements V-REQ-18 tool result. It is one file for one kind, the
// layout text_content.go established.
//
// Providers place a result differently — a block inside a user-role message on
// one, a distinct role on another, a nested response object on a third (doc 0001
// § 3.3 row 4). The register's verdict on that row is adapter-local: the
// normalized part models the answer, and each adapter maps it on the way out.
// Which role may carry one is AI-10.3's rule, not this file's.

package ai

// ToolResult is the answer to a tool call: its correlation to the originating
// call, its content, and whether the tool reported failure (V-REQ-18).
//
// It is an exported payload type under AI-06 decision.md § 6.3 — opaque, with
// unexported fields and a package-owned constructor — and it is the payload
// itself, for the reason [ToolCall] records. Its fields are a string, a string
// and a bool, so it stays comparable and immutable, and [Part] equality keeps
// working.
type ToolResult struct {
	callID, content string
	failed          bool
}

// kind reports this payload's registered kind.
func (ToolResult) kind() PartKind { return PartKindToolResult }

// validate reports the payload's first broken rule at the given position, or
// nil. There is one rule:
//
//  1. the correlation is not empty, else [ErrEmpty] — a result that names no
//     call is an answer to nothing, and the correlation is the only thing that
//     makes it placeable.
//
// The content has no rule, and an empty one is legal. A tool that produced no
// output — a search with no matches, a write that printed nothing — is routine,
// and rejecting it here would force every caller to invent a placeholder. Which
// placeholder, and whether a given provider needs one, is an application
// decision: V-OUT-04 puts tool execution above this layer and doc.go puts
// application behavior outside it. This is the one place a tool result
// deliberately differs from text content, whose emptiness rule exists because
// V-REQ-08 says model-visible natural-language text and a run of separators is
// none.
//
// Nothing constrains the *shape* of the correlation, deliberately: an identity
// an adapter minted must be indistinguishable from one a provider sent.
//
// Whether two results in one collection may correlate to the same call is a
// collection rule and is not checked here; AI-10.3 owns cross-region validation.
func (r ToolResult) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if r.callID == "" {
				return Invalid(ErrEmpty, under(at, At("call"))...)
			}
			return nil
		},
	)
}

// NewToolResult constructs a tool-result content part reporting that the tool
// answered (V-REQ-18).
//
// The correlation is the identity of the [ToolCall] this answers, compared by
// ordinary string equality. It is supplied rather than minted, and nothing here
// parses or rewrites it: an adapter for a provider that assigns no tool-call
// identifiers mints synthetic ones and keeps the mapping (doc 0001 § 3.3 row 7),
// and a session one layer up reloads it.
//
// The content is stored byte for byte. Nothing trims, normalizes, escapes or
// re-encodes it.
//
// On failure the zero [Part] is returned, so a caller that ignored the error
// cannot mistake the result for a constructed part.
func NewToolResult(callID, content string) (Part, error) {
	return newToolResult(ToolResult{callID: callID, content: content, failed: false})
}

// NewToolFailure constructs a tool-result content part reporting that the tool
// failed (V-REQ-18).
//
// # This is not an error, and that is the point
//
// A failing tool is a normal outcome the model must see and reason about. The
// register says so without hedging: a result that reports failure "is not a
// V-FAIL-01 and not a V-FAIL-05; it is ordinary content". It is neither a
// caller-contract failure — nobody broke a construction rule — nor a
// provider/transport failure, so nothing about it reaches AI-04's taxonomy. This
// constructor returns a nil error on its happy path exactly like its sibling,
// and the outcome is a state on the value, read back with [ToolResult.Failed].
//
// # Why two constructors rather than one with a flag
//
// A bare boolean at a call site — NewToolResult(id, out, true) — reads as
// nothing and is the classic place a caller inverts a meaning silently. Two
// constructors make the call site name the outcome. A two-step build, or a
// mutating setter, would put a valid-looking half-built value into play, which
// is the shape AI-06 spent a milestone removing.
//
// The content is the tool's own failure output. This package defines no failure
// *reason* vocabulary: it cannot know why a tool failed, and a taxonomy here
// would be Layer 2's policy wearing a Layer 1 type.
func NewToolFailure(callID, content string) (Part, error) {
	return newToolResult(ToolResult{callID: callID, content: content, failed: true})
}

// newToolResult runs the payload's rules and wraps it, so both constructors run
// one rule set rather than two that could drift.
func newToolResult(payload ToolResult) (Part, error) {
	if violation := payload.validate(nil); violation != nil {
		return Part{}, violation
	}
	return Part{payload: payload}, nil
}

// ToolResult returns the part's tool result and whether the part carries one.
//
// The accessor shape AI-06 landed. Called on a part of another kind, or on a
// part that was never constructed, it returns the zero [ToolResult] and false,
// and never panics.
func (p Part) ToolResult() (ToolResult, bool) {
	payload, ok := p.payload.(ToolResult)
	if !ok {
		return ToolResult{}, false
	}
	return payload, true
}

// CallID returns the identity of the tool call this result answers.
//
// It compares to [ToolCall.ID] with ==, so a consumer pairing calls to results
// needs no function of this package's. Pairing them is Layer 2's job: the
// invariant that every call in a transcript has a matching result is V-OUT-02,
// and the request-level fragment of it is AI-10.3's.
func (r ToolResult) CallID() string { return r.callID }

// Content returns the tool's answer, byte for byte as it was supplied.
func (r ToolResult) Content() string { return r.content }

// Failed reports whether the tool reported failure.
//
// It is not an error and must not be turned into one on the way up: the model
// sees this result and reasons about it, which is why a failing tool is content
// rather than a fault (V-REQ-18).
func (r ToolResult) Failed() bool { return r.failed }

// String renders the result for a diagnostic reader, naming the type and the
// outcome and nothing the tool produced.
//
// The outcome is rendered because it is a two-valued state this package defines
// rather than caller data, and it is the one thing a reader of a log actually
// needs. The correlation and the content are omitted for [ToolCall.String]'s
// reason: not the payload, and not its length, because a length is
// caller-derived and a prefix of a secret is still a secret (V-FAIL-13).
//
// A tool's output is the most sensitive payload this package carries — it is
// whatever a tool read off a disk or a network — so the posture AI-06 put on
// [Part] moves onto this exported payload type too.
func (r ToolResult) String() string {
	if r.failed {
		return "toolresult(failed)"
	}
	return "toolresult(ok)"
}

// GoString renders the result for the %#v verb, so the posture covers every fmt
// verb rather than the three a reader thinks of: without it, %#v falls back to
// reflection and prints the content.
func (r ToolResult) GoString() string { return r.String() }
