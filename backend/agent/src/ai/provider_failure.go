// AI-19 — the provider error taxonomy and the terminal error event.
//
// This file implements R-AIP-001 … R-AIP-015: one closed vocabulary for
// everything that goes wrong after a valid request leaves the process,
// carried by a single concrete type reachable on both delivery paths —
// returned directly pre-stream (V-FAIL-11), and wrapped as the terminal
// error event's payload mid-stream (V-FAIL-12). It is AI-04's complement
// (R-AIE-010): a sibling caller-contract-failure vocabulary is
// validation.go's; this is the provider/transport half doc 0001 § 3.1 (C4)
// and the register (V-FAIL-05 … V-FAIL-13) already named, never redefined
// here.
//
// # Same concrete type, both paths, by construction
//
// [Failure] implements AI-14's sealed eventPayload interface directly — no
// wrapper, no second type, no converter (R-AIP-013, design.md D7). A value
// returned by [PreStreamFailure] and a value carried by [ErrorEvent] are the
// same Go type compiled once in this package, which is what makes "one
// vocabulary, two delivery paths" a structural fact an adapter cannot
// violate by accident, rather than a convention two implementations have to
// be kept in agreement with.
//
// # Two perpendicular axes, never conflated
//
// [Failure.PartialOutput] answers one question alone — did a normalized
// output event precede this failure? — and [Failure.Delivery] answers a
// second, independent one — which carrier handed this failure over? Neither
// is derivable from the other, and no accessor on this type combines them
// (R-AIP-010 … R-AIP-012, design.md D8, G8 verbatim). [PreStreamFailure]
// takes no output-flag parameter at all, which is what makes the fourth
// combination — a pre-stream failure that claims output preceded it —
// unconstructible rather than merely undocumented.

package ai

import (
	"errors"
	"time"
)

// FailureCategory is the closed vocabulary value classifying why a request
// failed after it left the process (R-AIP-004).
//
// Its zero value is not a member — [FailureCategory.Validate] rejects it,
// the same "nobody set this" distinction finish_reason.go draws for
// FinishReason(0). A value outside the vocabulary is a provider/transport
// failure's own caller-contract defect, reported through AI-04's
// [ErrNotInVocabulary].
type FailureCategory uint8

// The vocabulary. Nine members, the charter minimum (R-AIP-004), closed and
// append-only (R-AIP-005, R-AIE-003).
const (
	_ FailureCategory = iota // the zero value names no category

	// FailureCategoryAuthentication is V-FAIL-05: the provider rejected the
	// request's credentials.
	FailureCategoryAuthentication

	// FailureCategoryAuthorization is V-FAIL-05: the provider accepted the
	// credentials but refused the request they authorize.
	FailureCategoryAuthorization

	// FailureCategoryRateLimit is V-FAIL-06: the provider is rate-limiting
	// the request. See [Failure.RetryAfter] for a typed, presence-separate
	// retry hint.
	FailureCategoryRateLimit

	// FailureCategoryUnavailable is V-FAIL-06: the provider is unavailable
	// or overloaded.
	FailureCategoryUnavailable

	// FailureCategoryTimeout is V-FAIL-07: the request did not complete
	// within its deadline.
	FailureCategoryTimeout

	// FailureCategoryCancellation is a required vocabulary member, not an
	// optional one (R-AIP-004): the request was cancelled, by the caller or
	// by a deadline the caller controls, distinct from a provider-side
	// timeout.
	FailureCategoryCancellation

	// FailureCategoryMalformedResponse is V-FAIL-08: the provider's response
	// was not well-formed for its documented transport encoding.
	FailureCategoryMalformedResponse

	// FailureCategoryUnsupportedCapability is V-FAIL-08: the provider
	// rejected the request because it does not support a capability the
	// request used.
	FailureCategoryUnsupportedCapability

	// FailureCategoryUnknown is V-FAIL-06: the recorded outcome for a
	// provider failure this vocabulary does not recognise. The raw provider
	// label survives, bounded and sanitized — see [Failure.RawLabel]
	// (R-AIP-006).
	FailureCategoryUnknown

	// failureCategoryLimit is the first value outside the vocabulary. It
	// must stay last — finish_reason.go's finishReasonLimit idiom, restated:
	// appending a member moves the bound with it, with no second edit to
	// remember.
	failureCategoryLimit
)

// failureCategoryNames is the stable string form of each category.
//
// An array sized by the bound, not a map — finish_reason.go's
// finishReasonNames idiom: the size follows the vocabulary automatically, so
// an appended constant gets an empty entry and renders as the placeholder
// instead of silently inheriting a neighbour's name.
var failureCategoryNames = [failureCategoryLimit]string{
	FailureCategoryAuthentication:        "authentication",
	FailureCategoryAuthorization:         "authorization",
	FailureCategoryRateLimit:             "rate_limit",
	FailureCategoryUnavailable:           "unavailable",
	FailureCategoryTimeout:               "timeout",
	FailureCategoryCancellation:          "cancellation",
	FailureCategoryMalformedResponse:     "malformed_response",
	FailureCategoryUnsupportedCapability: "unsupported_capability",
	FailureCategoryUnknown:               "unknown",
}

// unnamedFailureCategory renders a value that is not in the vocabulary —
// finish_reason.go's unnamedFinishReason posture: a placeholder, and
// deliberately not "unknown", since "unknown" (FailureCategoryUnknown) is
// itself a recorded, valid member.
const unnamedFailureCategory = "invalid"

// String returns the stable string form of the category.
//
// A member renders as its registered name; a value outside the vocabulary —
// including the zero value — renders as the fixed placeholder and never
// panics.
func (c FailureCategory) String() string {
	if c == 0 || c >= failureCategoryLimit || failureCategoryNames[c] == "" {
		return unnamedFailureCategory
	}
	return failureCategoryNames[c]
}

// Validate reports whether the category is a member of the closed
// vocabulary, positioned at the given path (R-AIP-005).
//
// A value outside the vocabulary — including the zero value — is a
// caller-contract failure of class [ErrNotInVocabulary].
func (c FailureCategory) Validate(at ...Step) error {
	return FirstFailure(func() *Violation {
		if c == 0 || c >= failureCategoryLimit {
			return Invalid(ErrNotInVocabulary, at...)
		}
		return nil
	})
}

// FailureCategories returns the nine-member category vocabulary, in
// declaration order (R-AIP-005, groundwork for AI-23.4's enumeration).
//
// The result is a fresh slice on every call — event.go's EventKinds idiom:
// a package-level variable would be a closed vocabulary any consumer could
// rewrite.
func FailureCategories() []FailureCategory {
	out := make([]FailureCategory, 0, int(failureCategoryLimit)-1)
	for c := FailureCategory(1); c < failureCategoryLimit; c++ {
		out = append(out, c)
	}
	return out
}

// The per-category sentinels (R-AIP-014): one per charter member, so
// errors.Is(err, ai.ErrRateLimited) works without unwrapping first.
//
// There is deliberately no umbrella "any provider failure" sentinel —
// design.md D4 rejects it by name, because it would be the lazy alternative
// R-AIP-014 forbids: a consumer that wants "is this any provider failure at
// all?" already has one — errors.As(err, &f) — and a shared sentinel every
// category matched would make errors.Is(err, ai.ErrTimeout) true for a
// rate-limit failure too, defeating the whole point of a per-category
// vocabulary.
var (
	ErrAuthentication        = errors.New("provider rejected the request's credentials")
	ErrAuthorization         = errors.New("provider refused the request its credentials authorize")
	ErrRateLimited           = errors.New("provider is rate-limiting the request")
	ErrUnavailable           = errors.New("provider is unavailable or overloaded")
	ErrTimeout               = errors.New("provider request timed out")
	ErrCancelled             = errors.New("provider request was cancelled")
	ErrMalformedResponse     = errors.New("provider response was not well-formed")
	ErrUnsupportedCapability = errors.New("provider does not support a capability the request used")
	ErrUnknownFailure        = errors.New("provider failure does not match a modelled category")
)

// failureCategorySentinels maps each category to its own sentinel —
// array-pinned like failureCategoryNames, so an appended category without a
// matching sentinel is caught by exhaustiveness (provider_failure_internal_test.go)
// rather than passing unnoticed.
var failureCategorySentinels = [failureCategoryLimit]error{
	FailureCategoryAuthentication:        ErrAuthentication,
	FailureCategoryAuthorization:         ErrAuthorization,
	FailureCategoryRateLimit:             ErrRateLimited,
	FailureCategoryUnavailable:           ErrUnavailable,
	FailureCategoryTimeout:               ErrTimeout,
	FailureCategoryCancellation:          ErrCancelled,
	FailureCategoryMalformedResponse:     ErrMalformedResponse,
	FailureCategoryUnsupportedCapability: ErrUnsupportedCapability,
	FailureCategoryUnknown:               ErrUnknownFailure,
}

// failureCategorySentinel reports whether c is a member with its own
// sentinel, and that sentinel — eventRegistryEntry's bound-check idiom,
// restated, with the error-typed result last (revive's error-return rule):
// this is a data lookup, not an operation that itself fails.
func failureCategorySentinel(c FailureCategory) (bool, error) {
	if c == 0 || c >= failureCategoryLimit {
		return false, nil
	}
	return true, failureCategorySentinels[c]
}

// DeliveryPath is which carrier handed a [Failure] over (V-FAIL-11,
// V-FAIL-12): the boundary is carrier handover per AI-02.1, not the first
// event.
//
// It is deliberately a two-member enum plus a bare bool
// ([Failure.PartialOutput]) rather than a three-member shape enum — design.md
// D8 rejects the shape enum by name, because collapsing the two axes into one
// field is exactly the G8 conflation this milestone exists to prevent.
type DeliveryPath uint8

const (
	_ DeliveryPath = iota // the zero value names no delivery path

	// DeliveryPreStream is the path for a failure returned directly, before
	// any carrier was handed over. Only [PreStreamFailure] produces it, and
	// that constructor takes no output-flag parameter — the impossible
	// fourth cell of R-AIP-010's table is unconstructible, not merely
	// undocumented.
	DeliveryPreStream

	// DeliveryMidStream is the path for a failure carried as the stream's
	// terminal [ErrorEvent] payload, after a carrier was handed over —
	// including a handed-over stream that fails before emitting content
	// (AI-02.1's own rule: handover, not first event).
	DeliveryMidStream

	// deliveryPathLimit is the first value outside the vocabulary. It must
	// stay last.
	deliveryPathLimit
)

// RetryDelay is a typed, presence-separate retry-after hint (R-AIP-008).
//
// It mirrors usage.go's [TokenCount]/[Tokens]: the zero value is absent, and
// [Delay] builds a value a provider actually reported, including an explicit
// zero — so "the provider said retry immediately" and "the provider said
// nothing" stay two distinguishable facts, read back through
// [Failure.RetryAfter].
type RetryDelay struct {
	delay   time.Duration
	present bool
}

// Delay builds a retry-after hint a provider reported.
//
// Delay(0) is a reported immediate retry, and is not the same value as an
// absent hint — the zero [RetryDelay] built by declaring a variable and never
// calling Delay.
func Delay(d time.Duration) RetryDelay {
	return RetryDelay{delay: d, present: true}
}

// FailureReport is the exported-field input a caller assembles to construct a
// [Failure] (design.md D3).
//
// Its zero value means "nothing reported" — mirroring [Usage]'s own zero-value
// posture — so a caller fills in only what its transport actually told it,
// with no functional-options surface to learn and no positional-argument
// order to remember as the field set grows.
type FailureReport struct {
	// Category is the failure's classification (R-AIP-004). Required: the
	// zero value fails construction with [ErrNotInVocabulary] at "category".
	Category FailureCategory

	// Retryable is the machine-readable retry signal (R-AIP-007). It is a
	// classification, not a scheduling decision — doc 0001 § 3.1 reserves
	// the decision to retry for Layer 2 (V-OUT-11).
	Retryable bool

	// RetryAfter is a typed, presence-separate retry-after hint (R-AIP-008).
	// Its zero value is absent, distinguishable from an explicit [Delay](0).
	RetryAfter RetryDelay

	// RawLabel is the provider's own, unmodelled failure label, preserved
	// when [Category] is [FailureCategoryUnknown] (R-AIP-006). It is bounded
	// and sanitized: an over-long or control-character-bearing value is
	// dropped whole rather than truncated, and construction still succeeds
	// (design.md resolution 2).
	RawLabel string

	// StatusClass is the transport status class (0 = not reported, 1…5 =
	// informational … server error), bounded safe metadata for [Error]'s
	// rendered text. The zero value means "nothing reported".
	StatusClass int

	// RequestID is the provider's own opaque request identifier, subject to
	// the same drop-whole bound as [RawLabel].
	RequestID string

	// Cause is the underlying error this failure wraps, reachable through
	// [Failure.Unwrap] but never reproduced by [Failure.Error] (R-AIP-009).
	Cause error
}

// Failure is the one provider/transport failure type in Layer 1 (R-AIP-013):
// the same concrete value returned directly by [PreStreamFailure] and carried
// as the terminal event's payload by [MidStreamFailure] plus [ErrorEvent].
//
// Every field is unexported; reading is exported accessors and constructing
// is [PreStreamFailure] / [MidStreamFailure], the only doors in. A nil
// *Failure and the zero Failure{} are both total: every method below is safe
// to call on either and never panics (NFR-AIP-B).
type Failure struct {
	category      FailureCategory
	retryable     bool
	retryAfter    RetryDelay
	rawLabel      string
	statusClass   int
	requestID     string
	cause         error
	delivery      DeliveryPath
	partialOutput bool
}

// noProviderFailure is what a nil *Failure renders as (NFR-AIP-B) —
// validation.go's Violation.Error() nil-pointer posture, restated for this
// type.
const noProviderFailure = "no provider failure"

// Error renders the failure for a diagnostic reader (R-AIP-009).
//
// The rendering is built only from a fixed prefix and the category's own
// registered text — validation.go's Violation.Error() precedent, restated:
// it never reproduces the wrapped cause's text, and never includes
// [Failure.RawLabel], [Failure.StatusClass] or [Failure.RequestID], proven
// with a planted credential/body sentinel (S-AIP-028…031). [Unwrap] still
// exposes the cause, so causes stay inspectable while a response body or
// credential cannot reach a log line through the default string —
// redaction becomes a property of the type. A nil *Failure renders as
// [noProviderFailure]; the zero Failure{} renders with the "invalid"
// category placeholder — both total, never panicking (NFR-AIP-B).
func (f *Failure) Error() string {
	if f == nil {
		return noProviderFailure
	}
	return "provider failure: " + f.category.String()
}

// GoString renders the failure for the %#v verb (R-AIP-016).
//
// Without it, %#v on a *Failure falls back to reflection and prints every
// unexported field — including the wrapped cause, which may carry raw
// provider body text or credential-adjacent material (event.go's
// [Event.GoString], content_part.go's [Part.GoString]: the same
// delegate-don't-reflect posture, restated for a type that is itself an
// error). It delegates to the already-redacted, already-nil-safe
// [Failure.Error] rather than duplicating a second renderer that would need
// keeping in sync — including its nil-receiver case: fmt invokes GoString on
// a typed-nil *Failure because the GoStringer assertion succeeds for the
// pointer method set, and Error already returns [noProviderFailure] for that
// case, so no second nil check is needed here (NFR-AIP-B).
//
// What this covers is exactly *Failure — the form every constructor returns
// and every in-repo call site holds. A copied Failure value is outside it:
// GoString, Error, Unwrap and Is are all pointer-receiver, so a copied value
// is neither a fmt.GoStringer nor an error, and %#v on one still reflects
// over every unexported field, cause included. The receiver stays a pointer
// even so, deliberately: a value receiver would extend coverage to copies,
// but fmt would then reach GoString on a typed-nil *Failure by dereferencing
// it to make the value copy — a nil-dereference panic where Error's
// pointer-receiver nil check returns [noProviderFailure] today — trading
// NFR-AIP-B's nil totality for a form no call site uses. The comprehensive
// value-form sweep belongs to AI-36's hardening pass.
//
// Pinned by AI-36 (R-AIP-017): the scope above is now a proven, guarded
// property, not just a stated intent. TestFailure_CopiedValue_ValueShapedCause_RendersCanary
// records that a copied value form's reflective rendering does reach a
// value-shaped wrapped cause — the leak this comment always described —
// and TestNoPublishedSurfaceReturnsFailureByValue is a structural guard
// proving no exported function, method or field of this module returns or
// stores a Failure by value; either test failing means this scope
// statement no longer holds and must be re-argued in the open, exactly as
// the paragraph above already commits to.
func (f *Failure) GoString() string { return f.Error() }

// Unwrap returns the wrapped cause, reachable by errors.Is/errors.As through
// this failure (R-AIP-014). A nil *Failure unwraps to nil rather than
// panicking (NFR-AIP-B).
func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.cause
}

// Is reports whether target is this failure's own category sentinel
// (R-AIP-014, design.md D4) — errors.Is's extension point, consulted at
// every hop before Unwrap is tried, so errors.Is(err, ai.ErrRateLimited)
// works without a caller unwrapping first. It matches only the one sentinel
// [Failure.Category] maps to; every other target — including another
// category's sentinel and the wrapped cause itself — is left for Unwrap to
// reach on the next hop, which is how [Failure.Unwrap] keeps the cause's own
// sentinel chain intact rather than shadowing it here. A nil *Failure
// matches nothing (NFR-AIP-B).
func (f *Failure) Is(target error) bool {
	if f == nil {
		return false
	}
	ok, sentinel := failureCategorySentinel(f.category)
	return ok && target == sentinel
}

// kind reports this payload's registered kind, satisfying the unexported
// eventPayload interface directly — no wrapper type (R-AIP-013, design.md
// D7).
func (f *Failure) kind() EventKind { return EventKindError }

// Category returns the failure's classification (R-AIP-004).
//
// Added here rather than deferred to Phase 3's vocabulary work: it is a pure
// field accessor with no rule of its own, and R-AIP-003's terminal
// exclusivity — this file's own phase — already needs it, to prove no
// accessor of this name exists on [Completion]. A nil *Failure reports the
// zero [FailureCategory] rather than panicking (NFR-AIP-B).
func (f *Failure) Category() FailureCategory {
	if f == nil {
		return 0
	}
	return f.category
}

// RawLabel returns the provider's own, unmodelled failure label (R-AIP-006),
// or empty when none was reported or it was dropped whole by
// [sanitizeOpaqueField]'s bound. A nil *Failure reports empty rather than
// panicking (NFR-AIP-B).
func (f *Failure) RawLabel() string {
	if f == nil {
		return ""
	}
	return f.rawLabel
}

// Retryable reports the machine-readable retry signal (R-AIP-007).
//
// It is a classification only — doc 0001 § 3.1 reserves the decision to
// retry for Layer 2 (V-OUT-11); this package exports no backoff, attempt
// counter or failover identifier. A nil *Failure reports false rather than
// panicking (NFR-AIP-B).
func (f *Failure) Retryable() bool {
	if f == nil {
		return false
	}
	return f.retryable
}

// RetryAfter returns the provider's retry-after hint and whether one was
// reported (R-AIP-008).
//
// The two-result shape is usage.go's [TokenCount.Count] idiom, restated:
// absent, a reported zero and a reported non-zero duration are three
// distinguishable readings, read independently of [Failure.Retryable] —
// neither is derived from the other. A nil *Failure reports (0, false)
// rather than panicking (NFR-AIP-B).
func (f *Failure) RetryAfter() (time.Duration, bool) {
	if f == nil {
		return 0, false
	}
	return f.retryAfter.delay, f.retryAfter.present
}

// StatusClass returns the transport status class (1 = informational, 2 =
// success, 3 = redirection, 4 = client error, 5 = server error) and whether
// one was reported. The zero value (0) means "nothing reported" — the same
// presence-typed shape [Failure.RetryAfter] uses, dedicated accessors
// rather than a substring of [Error]'s rendered text (R-AIP-009). A nil
// *Failure reports (0, false) rather than panicking (NFR-AIP-B).
func (f *Failure) StatusClass() (int, bool) {
	if f == nil {
		return 0, false
	}
	if f.statusClass == 0 {
		return 0, false
	}
	return f.statusClass, true
}

// RequestID returns the provider's own opaque request identifier, or empty
// when none was reported or it was dropped whole by [sanitizeOpaqueField]'s
// bound — the same drop-whole posture as [Failure.RawLabel]. A dedicated
// accessor, never a substring of [Error]'s rendered text (R-AIP-009). A nil
// *Failure reports empty rather than panicking (NFR-AIP-B).
func (f *Failure) RequestID() string {
	if f == nil {
		return ""
	}
	return f.requestID
}

// PartialOutput answers R-AIP-010's first discriminating question alone: did
// a normalized output event precede this failure? (V-FAIL-09).
//
// It carries no delivery information by name or by value — [Failure.Delivery]
// is the separate, perpendicular axis — so "is a naive retry safe?" is
// answerable from this bool alone, never true for a [PreStreamFailure]
// value, since that constructor takes no output-flag parameter at all
// (R-AIP-011, design.md D8). A nil *Failure reports false rather than
// panicking (NFR-AIP-B).
func (f *Failure) PartialOutput() bool {
	if f == nil {
		return false
	}
	return f.partialOutput
}

// Delivery reports which carrier handed this failure over (V-FAIL-11,
// V-FAIL-12) — [PreStreamFailure] always [DeliveryPreStream],
// [MidStreamFailure] always [DeliveryMidStream]. Independent of
// [Failure.PartialOutput]: delivery alone cannot distinguish the two
// mid-stream shapes (R-AIP-010). A nil *Failure reports the zero
// [DeliveryPath] rather than panicking (NFR-AIP-B).
func (f *Failure) Delivery() DeliveryPath {
	if f == nil {
		return 0
	}
	return f.delivery
}

// validate reports the payload's first broken rule at the given position, or
// nil, in the order written (V-FAIL-04): the category is a member of the
// closed vocabulary (R-AIP-004/005), then the status class — when reported
// at all — is one of the five documented classes (R-AIP-009's bounded safe
// metadata). NFR-AIP-C: every rule below routes through an AI-04 sentinel,
// never a new failure type.
func (f *Failure) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation { return violationOf(f.category.Validate(under(at, At("category"))...)) },
		func() *Violation {
			if f.statusClass < 0 || f.statusClass > 5 {
				return Invalid(ErrOutOfRange, under(at, At("status_class"))...)
			}
			return nil
		},
	)
}

// maxOpaqueFieldLen bounds an opaque, provider-supplied field ([RawLabel],
// [RequestID]): 64 bytes (design.md resolution 2).
const maxOpaqueFieldLen = 64

// sanitizeOpaqueField applies the drop-whole bound to an opaque,
// provider-supplied field.
//
// A value over [maxOpaqueFieldLen] bytes, or carrying an ASCII control
// character, is dropped whole — the accessor reports empty — rather than
// truncated: validation.go's structuralName states the same reasoning for
// the same shape of choice, "a prefix of a secret is still a secret".
// Construction never fails because of this filter; only the accessor's
// answer changes (design.md resolution 2, S-AIP-018/019/021).
func sanitizeOpaqueField(s string) string {
	if s == "" || len(s) > maxOpaqueFieldLen {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			return ""
		}
	}
	return s
}

// newFailure builds a *Failure from a report, a delivery path and a
// partial-output fact — the one place both [PreStreamFailure] and
// [MidStreamFailure] assemble a value, so their rules are one rule set with
// two entry points (reasoning_content.go's newReasoning precedent, decision
// D7's "same concrete type" restated for construction).
func newFailure(r FailureReport, delivery DeliveryPath, partialOutput bool) (*Failure, error) {
	f := &Failure{
		category:      r.Category,
		retryable:     r.Retryable,
		retryAfter:    r.RetryAfter,
		rawLabel:      sanitizeOpaqueField(r.RawLabel),
		statusClass:   r.StatusClass,
		requestID:     sanitizeOpaqueField(r.RequestID),
		cause:         r.Cause,
		delivery:      delivery,
		partialOutput: partialOutput,
	}
	if violation := f.validate(nil); violation != nil {
		return nil, violation
	}
	return f, nil
}

// PreStreamFailure constructs a failure returned directly, before any
// carrier was handed over (R-AIP-013, V-FAIL-11).
//
// It takes no output-flag parameter: the fourth cell of R-AIP-010's table —
// a pre-stream failure that claims output preceded it — is unconstructible
// here, not merely undocumented (design.md D8). On failure the nil *Failure
// is returned, so a caller that ignores the error cannot mistake the result
// for a constructed value; every accessor on a nil *Failure is total
// (NFR-AIP-B).
func PreStreamFailure(r FailureReport) (*Failure, error) {
	return newFailure(r, DeliveryPreStream, false)
}

// MidStreamFailure constructs a failure carried as the stream's terminal
// [ErrorEvent] payload, after a carrier was handed over (R-AIP-013,
// V-FAIL-12).
//
// outputPreceded answers R-AIP-010's first discriminating question — did a
// normalized output event precede this failure? — independently of
// delivery, which this constructor already fixes to [DeliveryMidStream]. On
// failure the nil *Failure is returned.
func MidStreamFailure(r FailureReport, outputPreceded bool) (*Failure, error) {
	return newFailure(r, DeliveryMidStream, outputPreceded)
}

// ErrorEvent wraps f as the stream's terminal error event — step 3 of
// event_descriptor.go's six-step kind-adding procedure (R-AIP-001).
//
// A nil f is rejected with [ErrEmpty] at "payload", the same AI-04 sentinel
// every other Layer 1 constructor reports an absent required value with. A
// non-nil f was already validated by [PreStreamFailure] or
// [MidStreamFailure] — the only doors that produce one — so this function
// re-checks nothing about its fields; [CheckEmit] is the "again" pass every
// registered kind receives (R-AEE-010), not a second call here. On failure
// the zero [Event] is returned. The returned event is unstamped (sequence
// 0); stamping is a [Stamper]'s job, per-stream, at the producer boundary.
func ErrorEvent(f *Failure) (Event, error) {
	if f == nil {
		return Event{}, Invalid(ErrEmpty, At("payload"))
	}
	return Event{payload: f}, nil
}

// ErrorPayload returns the event's error payload, and whether the event
// carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns nil and false, and never panics — the same
// R-AEE-005-shaped typed accessor [Event.Completion] and
// [Event.ResponseStart] already give their own kind.
func (e Event) ErrorPayload() (*Failure, bool) {
	payload, ok := e.payload.(*Failure)
	return payload, ok
}
