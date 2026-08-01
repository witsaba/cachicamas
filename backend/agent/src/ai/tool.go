// AI-08.1 — the tool declaration.

package ai

import "slices"

// MaxToolNameLen is the documented ceiling on a tool name, in bytes.
//
// It is exported because a caller that *generates* tool names — a plugin host,
// a Layer 3 tool source — needs the ceiling before it constructs rather than
// after it fails. It is the only constant of this contract a caller could want.
const MaxToolNameLen = 64

// Tool is the provider-neutral transport representation of one tool a model may
// call (V-REQ-12): its name, its description, and its schema bytes.
//
// It is a description of a capability offered to the model. It is not an
// executable, not a resolution target and not a permission subject — the
// register's first trap, which doc.go states for the package as a whole.
type Tool struct {
	name          string
	description   string
	schema        []byte
	cacheBoundary bool // AI-11.1
}

// NewTool constructs a tool declaration from a name, a description and the
// bytes of its argument schema.
//
// The schema bytes are copied in, and [Tool.Schema] copies them out again.
// These are two mechanisms and not one: a constructor that clones and a reader
// that does not passes every construction test and fails the moment two
// consumers hold the same declaration.
// The rules are checked in the order written, per V-FAIL-04, and the order is
// part of the contract:
//
//  1. The name is not empty, else [ErrEmpty] at "name".
//  2. The name is at most [MaxToolNameLen] bytes, else [ErrOutOfRange] at
//     "name".
//  3. The name satisfies the documented character rules, else [ErrMalformed] at
//     "name".
//  4. The schema bytes are not empty, else [ErrEmpty] at "schema".
//
// Rules 2 and 3 are separate classes on purpose. V-FAIL-17 makes a rule class
// the kind of thing a rule checks rather than the field it checks it on, and
// "shorten it" and "spell it differently" are different facts with different
// fixes. Reporting an over-long name as malformed would tell a caller to change
// the spelling of a perfectly well-spelled name.
//
// The description has no rule. Every provider a v1 adapter could target treats
// it as optional — strongly recommended in at least one — and a recommendation
// is not a caller contract.
//
// The schema bytes are *not* checked for syntactic well-formedness, and their
// content never affects whether construction succeeds. doc 0002 asks for that
// check explicitly on AI-09.1's argument bytes and pointedly does not here: an
// argument payload is produced by a model and reassembled from a stream, where
// a malformed reassembly is a recurring defect class; a schema is a literal in
// the caller's own source. V-REQ-13 states this package's role as "transports
// it", and the register's trap draws the line at bytes versus meaning.
//
// On failure the zero Tool is returned, so a caller that ignored the error
// cannot mistake the result for a constructed declaration — its name is empty,
// which no constructed declaration's is.
func NewTool(name, description string, schema []byte) (Tool, error) {
	rules := append(toolNameRules(name), func() *Violation {
		if len(schema) == 0 {
			return Invalid(ErrEmpty, At("schema"))
		}
		return nil
	})
	if err := FirstFailure(rules...); err != nil {
		return Tool{}, err
	}
	return Tool{name: name, description: description, schema: slices.Clone(schema)}, nil
}

// toolNameRules is the ordered rule list every tool name is checked against.
//
// It is a function returning rules rather than a predicate so that both call
// sites — [NewTool] here and [NewNamedToolChoice] in tool_choice.go — report
// the same classes at the same position, in the same order. One name rule in
// the package, not two that could drift.
//
// # The rule, and why it is the intersection rather than the union
//
// A name is legal exactly when it is 1 to [MaxToolNameLen] bytes long, its
// first byte is an ASCII letter or underscore, and every byte is an ASCII
// letter, digit, underscore or hyphen.
//
// That is narrower than any single provider requires, and the narrowness is the
// point. A tool name is not merely sent — it comes back: the model answers with
// a tool call naming the tool, and the layer above resolves that name to an
// implementation. A name this package accepted but an adapter had to rewrite on
// the way out would have to be un-rewritten on the way in, a bidirectional
// mapping every future adapter would reimplement and every future stream
// decoder would have to invert. The alternative to rejecting the name here is
// therefore not "let the provider decide"; it is a network round trip that ends
// in a rejection, or a silent adapter-local mangling that breaks call
// correlation.
//
// The asymmetry that makes this safe to start narrow: loosening the rule later
// is additive — a name that was rejected becomes accepted, and nothing that
// worked stops working — while tightening it is breaking. Starting at the
// intersection keeps the cheap move available; starting at the union does not.
//
// Bytes rather than runes, deliberately: the alphabet is ASCII-only, so every
// byte of a multi-byte sequence fails the third rule, and a byte-wise scan
// cannot disagree with a rune-wise one about a string containing no legal
// multi-byte character. validation.go's own structuralName filter has the same
// shape for the same reason.
func toolNameRules(name string) []Rule {
	return []Rule{
		func() *Violation {
			if name == "" {
				return Invalid(ErrEmpty, At("name"))
			}
			return nil
		},
		func() *Violation {
			if len(name) > MaxToolNameLen {
				return Invalid(ErrOutOfRange, At("name"))
			}
			return nil
		},
		func() *Violation {
			if !isToolNameShaped(name) {
				return Invalid(ErrMalformed, At("name"))
			}
			return nil
		},
	}
}

// isToolNameShaped reports whether a non-empty name satisfies the documented
// character rules. It says nothing about length; that is a separate rule with a
// separate class.
func isToolNameShaped(name string) bool {
	for i := range len(name) {
		c := name[i]
		switch {
		case c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z'):
			// Legal anywhere, including first.
		case c == '-' || ('0' <= c && c <= '9'):
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return name != ""
}

// Name returns the tool's name.
func (t Tool) Name() string { return t.name }

// Description returns the tool's description, which may be empty.
func (t Tool) Description() string { return t.description }

// Schema returns the tool's schema bytes (V-REQ-13), exactly as supplied.
//
// Exactly: no re-marshalling, no key reordering, no whitespace normalization,
// no canonicalization of any kind. Every one of those is a one-line
// "improvement" that would pass a naive round-trip test — a marshal round trip
// is idempotent after the first pass — and would silently destroy the cache
// prefix V-REQ-25 describes for every caller whose schema was not already in
// the canonical form. The symptom would be no error and no wrong answer, just
// an input bill roughly ten times larger than it should be.
//
// The result is a fresh slice on every call, for the reason [NewTool] gives.
func (t Tool) Schema() []byte { return slices.Clone(t.schema) }

// MarkCacheBoundary returns a copy of t carrying a cache-boundary marker
// (V-REQ-23), indicating a point at which a provider may cache the prefix
// preceding it.
//
// It is a no-op when t never passed [NewTool]: its zero-value detector is
// [Tool.Name] reporting empty, and a marked zero value must not start
// reporting itself a cache boundary (R-ACB-002).
//
// t is a value receiver, so the marker is set on a local copy and returned;
// the value t was called on is left observably unmarked (R-ACB-001).
func (t Tool) MarkCacheBoundary() Tool {
	if t.Name() == "" {
		return t
	}
	t.cacheBoundary = true
	return t
}

// IsCacheBoundary reports whether t carries a cache-boundary marker.
//
// A declaration that was never marked reports false, and marking an
// already-marked declaration again is idempotent (R-ACB-001).
func (t Tool) IsCacheBoundary() bool { return t.cacheBoundary }
