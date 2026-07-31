// AI-09.1 — the tool call: a model's intent to invoke a tool.
//
// This file implements V-REQ-16 tool call and V-REQ-17 argument bytes. It is one
// file for one kind, the layout text_content.go established: the payload type,
// its rules, its constructor and its accessor live together.
//
// Layer 1 never acts on a tool call. It carries the provider-neutral transport
// representation of one — the register's first trap, which doc.go states for the
// package as a whole.

package ai

import "encoding/json"

// emptyToolArguments is the one canonical form of a call that takes no
// arguments.
//
// A no-argument tool is routine, and a parse failure on empty input is a shipped
// SDK's bug class rather than a hypothetical: every consumer that decodes
// argument bytes would need a guard clause for the zero-length case, and the
// consumer that forgets it is the defect. Canonicalizing here makes the guard
// clause unnecessary and buys an invariant worth stating — **every constructible
// tool call's argument bytes are well-formed JSON**, which is what AI-26 and
// AI-30 decode against.
//
// It is the one place this contract writes bytes a caller did not supply, and
// the scope is exactly the absent case. V-REQ-17's byte fidelity binds supplied
// bytes; absent arguments are not supplied bytes.
const emptyToolArguments = "{}"

// ToolCall is a model's request to invoke a tool: its identity, the tool's name,
// and the exact argument bytes (V-REQ-16).
//
// It is an exported payload type because it carries structure, which AI-06's
// decision.md § 6.3 permits under one condition: it is an opaque value type with
// unexported fields and a package-owned constructor. It is also the payload
// itself — the type that answers kind() — rather than a wrapper around an
// unexported twin, because the seal lives on [Part], whose field stays
// unexported, and partPayload's methods are unexported, so nothing outside this
// package can be one.
//
// The three values are held as strings, and the argument bytes are converted on
// the way in and on the way out. That keeps ToolCall comparable: [Part]
// documents that == is defined and compares payloads, and an interface value
// holding a struct with a slice field panics when compared. It also makes the
// value immutable, so copying a part cannot let two holders observe each other.
type ToolCall struct{ id, name, arguments string }

// kind reports this payload's registered kind.
func (ToolCall) kind() PartKind { return PartKindToolCall }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04:
//
//  1. the identity is not empty, else [ErrEmpty] — the identity is what makes a
//     call addressable, and a result correlates to it by string equality;
//  2. the tool name is not empty, else [ErrEmpty];
//  3. the argument bytes are syntactically well-formed JSON, else
//     [ErrMalformed] — the case [ErrMalformed]'s own GoDoc was written about.
//
// Identity before name for the reason text_content.go gives for emptiness before
// range: different facts have different fixes, and a caller repairing one at a
// time must make progress in a predictable direction.
//
// Rule 3 is syntax and never meaning. Bytes that are well-formed but satisfy no
// declared tool's schema construct successfully: this package holds no schema to
// check against and V-REQ-13 states its role as transporting them.
//
// There is no shape rule on the name. AI-08 lands one on a tool *declaration*;
// whether a call's name must satisfy it too is a cross-region question, and
// AI-10.3 owns cross-region validation. Recorded so the absence reads as a
// decision.
func (c ToolCall) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if c.id == "" {
				return Invalid(ErrEmpty, under(at, At("id"))...)
			}
			return nil
		},
		func() *Violation {
			if c.name == "" {
				return Invalid(ErrEmpty, under(at, At("name"))...)
			}
			return nil
		},
		func() *Violation {
			if !json.Valid([]byte(c.arguments)) {
				return Invalid(ErrMalformed, under(at, At("arguments"))...)
			}
			return nil
		},
	)
}

// NewToolCall constructs a tool-call content part (V-REQ-16).
//
// It is the only door in. The argument bytes are copied on the way in and
// [ToolCall.Arguments] copies them out again, so no caller and no consumer holds
// a reference into the constructed part.
//
// The bytes are stored exactly as supplied: nothing here re-marshals them,
// reorders their keys, normalizes their whitespace, escapes or re-encodes them.
// That is V-REQ-17, and AI-26's translation and AI-30's stream reassembly both
// compare bytes.
//
// Absent arguments are the one exception, and it is an exception about a value
// that was not supplied rather than about one that was: nil and a zero-length
// slice both become [emptyToolArguments], so a call that takes no arguments is
// constructible and every such call carries the identical, decodable form.
//
// The rules are [ToolCall.validate]'s, run from here and from the message
// boundary — one rule set, two entry points. On failure the zero [Part] is
// returned, so a caller that ignored the error cannot mistake the result for a
// constructed part.
func NewToolCall(id, name string, arguments []byte) (Part, error) {
	stored := string(arguments)
	if len(arguments) == 0 {
		stored = emptyToolArguments
	}
	payload := ToolCall{id: id, name: name, arguments: stored}
	if violation := payload.validate(nil); violation != nil {
		return Part{}, violation
	}
	return Part{payload: payload}, nil
}

// ToolCall returns the part's tool call and whether the part carries one.
//
// This is the accessor shape AI-06 landed and every kind inherits: a method on
// [Part], named for the kind it reads, returning the typed payload and whether
// the part is of that kind. Called on a part of another kind, or on a part that
// was never constructed, it returns the zero [ToolCall] and false, and never
// panics.
func (p Part) ToolCall() (ToolCall, bool) {
	payload, ok := p.payload.(ToolCall)
	if !ok {
		return ToolCall{}, false
	}
	return payload, true
}

// ID returns the call's identity.
//
// It is an opaque string a caller supplies rather than a handle this package
// mints, and the difference from [MessageID] is deliberate. One provider assigns
// no tool-call identifiers at all, so its adapter mints synthetic ones and keeps
// the mapping (doc 0001 § 3.3 row 7) — and that mapping must survive session
// serialisation and reload one layer up. A minted, unforgeable handle would make
// both unimplementable.
//
// Nothing here parses, rewrites or constrains the shape of the identity: an
// identity an adapter minted is textually indistinguishable from one a provider
// sent, which is the property AI-26.5 needs.
func (c ToolCall) ID() string { return c.id }

// Name returns the name of the tool the model asked to invoke.
//
// Layer 1 does not resolve it to an implementation, and holds nothing it could
// resolve it against — that is V-OUT-04 and the register's first trap.
func (c ToolCall) Name() string { return c.name }

// Arguments returns the call's argument bytes (V-REQ-17).
//
// The result is a fresh slice on every call and is byte-equal to the bytes
// [NewToolCall] was given: nothing on this path decodes, re-marshals, reorders,
// normalizes or re-encodes them. A consumer that rewrites what it received must
// not be able to rewrite the part, and two consumers must not be able to observe
// each other.
func (c ToolCall) Arguments() []byte { return []byte(c.arguments) }

// String renders the call for a diagnostic reader, naming the type and nothing
// it carries.
//
// [Part.String] documents why the default is worse than silence: fmt prints the
// unexported fields of a struct it has no String method for. That posture was
// complete while every payload was a string behind an accessor, and this is the
// first exported payload type, so the posture moves onto it. Not the identity,
// not the tool name, not the arguments, and not their length — a length is
// caller-derived and a prefix of a secret is still a secret (V-FAIL-13).
func (c ToolCall) String() string { return "toolcall" }

// GoString renders the call for the %#v verb.
//
// It exists so the posture covers every fmt verb rather than the three a reader
// thinks of: without it, %#v falls back to reflection and prints the arguments.
func (c ToolCall) GoString() string { return c.String() }
