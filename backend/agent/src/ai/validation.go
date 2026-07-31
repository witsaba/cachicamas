// AI-04 — the validation error taxonomy.
//
// This file is the one way every Layer 1 contract reports that a caller broke a
// rule. It implements four terms the Layer 1 vocabulary register already
// defines: V-FAIL-01 caller-contract failure, V-FAIL-02 validation sentinel,
// V-FAIL-03 positional context, and V-FAIL-04 first-failure ordering. It does
// not invent a taxonomy; the register recorded one before the first validating
// contract existed, which is why this milestone is the first of wave 1.
//
// # What belongs here, and what does not
//
// A caller-contract failure is a violation of a construction or composition rule
// by the caller — the caller's bug, decidable from the request alone, without
// contacting a provider. Everything that can only be learned by asking a
// provider is a provider/transport failure and belongs to AI-19: transport,
// protocol, authentication, rate limiting, provider-side errors, deadlines and
// disconnects. Two clauses of that boundary do the work and are the ones dropped
// on recall. "From the request alone" excludes facts that are true of the call
// but not of the request value: a context whose deadline has already passed is
// decidable with no I/O at all and is still not a caller-contract failure. "Not
// who is to blame" excludes the intuition that a caller's mistakes are
// caller-contract failures: a revoked credential is the caller's mistake and is
// a provider/transport failure, because this package cannot know it without
// asking.
//
// # Two axes, two queries
//
// A failure answers exactly two questions, and each has exactly one mechanism:
//
//	errors.Is(err, ErrEmpty)  // which rule class was violated
//	errors.As(err, &v)        // where, structurally
//
// Sentinels name a rule class and are reusable across every contract in this
// package, so "a required value is empty" is one thing everywhere. Which value
// is empty is the position's job, not a second sentinel's. There is exactly one
// concrete failure type, so a consumer never has to try a second errors.As
// target — that would be the second, parallel failure vocabulary V-FAIL-03
// forbids.
//
// # Redaction starts here
//
// A validation failure is the first thing in this package that formats caller
// data, and V-FAIL-13 puts the redaction posture at that point rather than at
// the hardening milestone. It is a property of the type rather than of anyone's
// discipline: a rendered message is built only from a registered rule class's
// own text, filtered structural names, decimal indices and fixed punctuation.
// Neither of the two dynamic inputs can carry a caller's value into a message.

package ai

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

// The rule classes.
//
// A rule class is the kind of thing a rule checks, independent of what it checks
// it on (V-FAIL-17). One sentinel per class, reusable across types, is AI-04.1's
// decision: a sentinel per rule per type would grow an exported surface without
// bound and would make the common question — "did some required value come back
// empty?" — answerable only by enumerating every sentinel in the package.
//
// The set is closed and is extended by appending a class in the pull request
// that needs it, never by a milestone defining a sentinel of its own. Each class
// below has a citable case in the register or in the Layer 1 task graph; none is
// landed in anticipation of a milestone that has not arrived.
var (
	// ErrEmpty is the class for a required value that is empty. Register § 6.3:
	// an empty model identity is decidable without I/O and is a caller-contract
	// failure.
	ErrEmpty = errors.New("required value is empty")

	// ErrNotInVocabulary is the class for a value outside a closed vocabulary.
	// V-REQ-01: a value outside the role vocabulary is a caller-contract
	// failure, not an unknown passed through.
	ErrNotInVocabulary = errors.New("value is outside a closed vocabulary")

	// ErrOutOfRange is the class for a value outside a documented bound.
	// V-REQ-24: a request exceeding the documented breakpoint cap is a
	// caller-contract failure rather than a silent truncation.
	ErrOutOfRange = errors.New("value is outside a documented bound")

	// ErrMalformed is the class for a value that is not well-formed for its
	// documented encoding. Register § 6.3: argument bytes that are not
	// well-formed are decidable from the request alone. Bytes that are
	// well-formed but do not satisfy a tool's schema are neither this nor a
	// provider failure — this package does not validate against schemas.
	ErrMalformed = errors.New("value is not well-formed for its documented encoding")

	// ErrUnresolvedReference is the class for a value naming something the
	// request does not declare. Register § 6.3: a tool choice naming a tool
	// absent from the declared set is decidable from the request alone.
	ErrUnresolvedReference = errors.New("value names something the request does not declare")

	// ErrDuplicate is the class for a value that repeats another the same
	// collection already carries, where the rule requires them to differ.
	// AI-08.2 item 1: two tool declarations with the same name in one tool set
	// are decidable from the request alone.
	//
	// It is not [ErrMalformed], and the difference is the whole reason it
	// exists. Each repeated value is perfectly well-formed on its own; what
	// fails is uniqueness across the collection, which no single value can be
	// inspected for. The fix a consumer is being told to make differs
	// accordingly — "remove one of them", not "spell it differently" — and
	// errors.Is is the only place a consumer can read that difference.
	ErrDuplicate = errors.New("value repeats another the collection already carries")
)

// ruleClasses is the closed, ordered registry of rule classes.
//
// It is a slice and not a map on purpose. Nothing in this package may let an
// unordered iteration decide anything, and a registry is where that temptation
// is strongest.
var ruleClasses = []error{
	ErrEmpty,
	ErrNotInVocabulary,
	ErrOutOfRange,
	ErrMalformed,
	ErrUnresolvedReference,
	ErrDuplicate,
}

// Step is one segment of a position: a named field of a Layer 1 contract, and
// optionally which element of it.
//
// A step is built by [At] or [AtIndex] and carries no caller-supplied value. Its
// zero value is usable and renders as an unnamed segment.
type Step struct {
	name    string
	index   int
	indexed bool
}

// At names a field of a Layer 1 contract, with no index.
//
// The name is a constant written by this package — "model", "content",
// "schema". It is not caller data, and a value that is not shaped like a field
// name is replaced whole rather than rendered.
func At(name string) Step {
	return Step{name: structuralName(name)}
}

// AtIndex names one element of a field of a Layer 1 contract.
//
// This is how a position says which message, which content part, or which tool:
// by ordinal, not by name. V-REQ-14 makes the tool set ordered and
// deterministically iterable, so an index identifies one tool unambiguously, and
// the caller — which holds the request — can resolve it to a name in one step.
// Rendering the name here instead would put caller data in a message that will
// be logged.
//
// An index below zero names no element and is treated as if [At] had been used:
// rendering "[-1]" would be a segment a reader has to decode.
func AtIndex(name string, index int) Step {
	if index < 0 {
		return At(name)
	}
	return Step{name: structuralName(name), index: index, indexed: true}
}

// Name returns the step's structural name. A step with no name — the zero value
// — reports the same placeholder a rejected name does.
func (s Step) Name() string {
	if s.name == "" {
		return redactedName
	}
	return s.name
}

// Index returns the step's index and whether it has one.
//
// The second result is what distinguishes a field from an element, so that no
// caller has to know a sentinel integer means "not indexed".
func (s Step) Index() (int, bool) {
	return s.index, s.indexed
}

// String renders the step as one segment of a position.
func (s Step) String() string {
	if !s.indexed {
		return s.Name()
	}
	return s.Name() + "[" + strconv.Itoa(s.index) + "]"
}

// Path is the position of a violation: an ordered sequence of steps, outermost
// first. An empty path is a violation with no meaningful position, which is a
// value rather than an absence.
type Path []Step

// String renders the position, for example "messages[2].content[0]".
//
// Rendering is iterative, so a deeply nested position costs memory and never
// stack.
func (p Path) String() string {
	var b strings.Builder
	for i, step := range p {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(step.String())
	}
	return b.String()
}

// Violation is the one caller-contract failure type in Layer 1.
//
// Every rule violation in every Layer 1 contract reports through it, so a
// consumer writes exactly one errors.As target for the whole package. It carries
// the rule class it violated, reachable by errors.Is through any number of
// wrappers, and the position of the violation, which may be empty.
type Violation struct {
	rule error
	at   Path
}

// Invalid builds a caller-contract failure for a violated rule class at a
// position.
//
// The rule should be one of this package's rule classes, or an error wrapping
// one; a rule that wraps a class stays fully matchable and contributes nothing
// to the rendered message. The position is copied, so a caller may reuse the
// slice it passed.
func Invalid(rule error, at ...Step) *Violation {
	return &Violation{rule: rule, at: slices.Clone(Path(at))}
}

// Error renders the failure as "position: rule class", or as the rule class
// alone when the position is empty.
//
// The rendering is composed only of a registered rule class's own text, filtered
// structural names, decimal integers and fixed punctuation. It never reproduces
// the text of the error the failure was built from, so a wrapper carrying a
// content body, an argument payload or a credential cannot leak through it.
func (v *Violation) Error() string {
	if v == nil {
		return "no violation"
	}
	if len(v.at) == 0 {
		return ruleText(v.rule)
	}
	return v.at.String() + ": " + ruleText(v.rule)
}

// Unwrap returns the rule class, which is what errors.Is traverses to.
//
// It makes the class the failure's cause, so a failure stays matchable through
// any number of intermediate wrappers — V-FAIL-02's own requirement.
func (v *Violation) Unwrap() error {
	if v == nil {
		return nil
	}
	return v.rule
}

// Path returns the position of the violation.
//
// The result is a copy: a failure is passed upward and logged, and a consumer
// that appends to the position it received must not be able to rewrite the
// failure's own.
func (v *Violation) Path() Path {
	if v == nil {
		return nil
	}
	return slices.Clone(v.at)
}

// ruleText renders the text of the registered rule class a rule belongs to.
//
// It never renders rule.Error(). The rendering is built only from text this
// package wrote, which is what makes the redaction posture a property of the
// type rather than of everyone's discipline.
func ruleText(rule error) string {
	if rule == nil {
		return "value violates an unnamed rule"
	}
	for _, class := range ruleClasses {
		if errors.Is(rule, class) {
			return class.Error()
		}
	}
	return "value violates an unregistered rule"
}

// maxStructuralNameLen bounds a structural name. Field names of Layer 1
// contracts are short; anything longer is not one.
const maxStructuralNameLen = 32

// redactedName replaces a name that is not structural, and renders a step that
// has none.
const redactedName = "?"

// structuralName filters a position's name down to the shape of a Layer 1 field
// name and replaces anything else whole.
//
// Whole, never truncated: a prefix of a secret is still a secret. Digits are
// excluded as well as punctuation, because a structural field name never needs
// one and their absence removes the shape most credentials and opaque
// identifiers take.
//
// The filter is a backstop, not the contract. The contract is that a position is
// built from names this package wrote; the filter is what makes a value that is
// not one — a body, an argument payload, a credential — unable to render.
func structuralName(name string) string {
	if name == "" || len(name) > maxStructuralNameLen {
		return redactedName
	}
	for i := range len(name) {
		c := name[i]
		if c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') {
			continue
		}
		return redactedName
	}
	return name
}

// Rule is one check a Layer 1 contract makes against a value it is validating.
//
// It is a function rather than an already-built violation so that a rule after
// the first failure is never evaluated: eager construction would allocate on the
// success path and would be aggregation wearing a first-failure hat.
type Rule func() *Violation

// FirstFailure evaluates rules in order and reports the first violation, or a
// nil error when every rule passes. A nil rule is skipped.
//
// The order of the slice is the documented order (V-FAIL-04). Making the order
// data rather than control flow is the point: a reviewer reads which rule wins
// instead of tracing it, and there is nowhere for an unordered iteration to
// decide it.
//
// It returns error and not *Violation deliberately. A nil *Violation placed in
// an error interface is not a nil error, and that trap would otherwise be
// reproduced at every call site in every contract of this package. Converting
// once, here, makes "if err := FirstFailure(...); err != nil" correct
// everywhere.
func FirstFailure(rules ...Rule) error {
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if violation := rule(); violation != nil {
			return violation
		}
	}
	return nil
}
