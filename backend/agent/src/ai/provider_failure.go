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

import "time"

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
// This is a stub at this milestone's foundation stage: the fixed-prefix,
// category-only rendering design.md pins is Phase 4's (task 4.4). It is
// already total — nil-safe — because NFR-AIP-B's totality guarantee is a
// property of the type from its first commit, not an afterthought bolted on
// once every field exists.
func (f *Failure) Error() string {
	if f == nil {
		return noProviderFailure
	}
	return "provider failure"
}

// Unwrap returns the wrapped cause, reachable by errors.Is/errors.As through
// this failure (R-AIP-014). A nil *Failure unwraps to nil rather than
// panicking (NFR-AIP-B).
func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.cause
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

// validate reports the payload's first broken rule at the given position, or
// nil.
//
// It carries no rule of its own at this milestone's envelope stage
// (R-AIP-001 … R-AIP-003) — export_test.go's WitnessPayload precedent: "a
// rule is added once a test needs one to fail". Phase 3 adds the
// category-vocabulary rule (R-AIP-004/005); Phase 4 adds the status-class
// bound (R-AIP-009's bounded safe metadata); NFR-AIP-C is re-verified once
// every rule this method will ever carry has landed.
func (f *Failure) validate(_ Path) *Violation { return nil }

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
		rawLabel:      r.RawLabel,
		statusClass:   r.StatusClass,
		requestID:     r.RequestID,
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
