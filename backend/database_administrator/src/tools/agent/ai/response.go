// Package-level note for the AI-12 response start/complete payloads (response.go).
//
// This file implements two concrete eventPayload implementations that
// satisfy the AI-11 sealed interface (see event.go): ResponseStartPayload
// (response ID + model identifier) and ResponseCompletePayload
// (response ID + FinishReason + Usage). The unexported aiPayload()
// marker seals each type into the interface — only this package can
// implement eventPayload.
//
// Ordering invariants (AI-11) are documented but NOT enforced at runtime
// here: AI-20 (stream testkit) and AI-21 (conformance suite) own
// per-producer stream validation. The payload types validate only their
// own field-level invariants. See AI-12 design #1 (payload scope) and
// #5 (ordering scope).
package ai

import (
	"errors"
)

// MaxResponseIDLength is the inclusive byte ceiling for the ResponseID
// field on every AI-12 response payload (ResponseStartPayload and
// ResponseCompletePayload). The bound is forward-safe at 256 bytes —
// long enough for vendor prefixes and ULID-shaped identifiers, short
// enough that an accidental user-input echo cannot blow the field
// across a stream frame. Per AI-12 design #2.
const MaxResponseIDLength = 256

// MaxResponseModelLength is the inclusive byte ceiling for the Model
// field on ResponseStartPayload. Same rationale as MaxResponseIDLength:
// forward-safe for current vendor identifiers, short enough to refuse
// pathological echoes. Per AI-12 design #2.
const MaxResponseModelLength = 256

// ErrEmptyResponseID is returned when the ResponseID field on a
// ResponseStartPayload / ResponseCompletePayload is empty. The sentinel
// is distinct from AI-09's ErrEmptyModel (Request.Model) via errors.Is
// pairwise distinctness across the package.
var ErrEmptyResponseID = errors.New("ai: empty response id")

// ErrWhitespaceResponseID is returned when the ResponseID is non-empty
// but every rune reports IsSpace. Per AI-12 spec § "Requirement:
// Whitespace validation".
var ErrWhitespaceResponseID = errors.New("ai: whitespace-only response id")

// ErrResponseIDTooLong is returned when len(ResponseID) exceeds
// MaxResponseIDLength (inclusive). The length check runs AFTER the
// empty / whitespace checks so a pathological zero value cannot silently
// trip the boundary instead of the semantic check.
var ErrResponseIDTooLong = errors.New("ai: response id exceeds MaxResponseIDLength bytes")

// ErrEmptyResponseModel is returned when the Model field on
// ResponseStartPayload is empty. Named to avoid collision with AI-09's
// ErrEmptyModel in request.go (which validates Request.Model). The two
// sentinels are pairwise distinct via errors.Is while sharing the
// "ai: ..." message prefix per Phase B convention. Per AI-12 discovery
// #2171 (sentinel collision) — the suffix disambiguates the two
// contexts while keeping the package vocabulary predictable.
var ErrEmptyResponseModel = errors.New("ai: empty response model identifier")

// ErrWhitespaceResponseModel is returned when the Model field is
// non-empty but every rune reports IsSpace. Distinct from AI-09's
// ErrWhitespaceModel via errors.Is.
var ErrWhitespaceResponseModel = errors.New("ai: whitespace-only response model identifier")

// ErrResponseModelTooLong is returned when len(Model) exceeds
// MaxResponseModelLength (inclusive). Distinct from any request-side
// length sentinel.
var ErrResponseModelTooLong = errors.New("ai: response model identifier exceeds MaxResponseModelLength bytes")

// ResponseStartPayload is the AI-12 event payload for response.start
// events. It carries the response ID (the cross-stream correlation
// handle) and the model identifier that produced the stream. The
// struct is value-typed with unexported fields; NewResponseStartPayload
// is the sanctioned constructor. The unexported aiPayload() marker
// seals the type into the eventPayload interface — only this package
// can satisfy that interface.
//
// Per AI-12 design #1 and #6: payload scope is per-event only; ordering
// invariants are NOT validated here. AI-20 / AI-21 own stream-level
// conformance.
type ResponseStartPayload struct {
	responseID string
	model      string
}

// aiPayload is the unexported marker method that seals
// ResponseStartPayload into the eventPayload interface. External
// packages cannot satisfy this method, which is the seal that makes
// (Kind, Payload) parity unrepresentable outside this package. See
// event.go for the full contract.
func (ResponseStartPayload) aiPayload() {}

// Kind returns the canonical EventKind for this payload. Every
// ResponseStartPayload returns EventKindResponseStart verbatim,
// regardless of the field values. The seal guarantees that
// Event.Kind (derived from this in NewEvent) cannot disagree with
// the payload for events constructed via the sanctioned entry
// points (NewEvent, NewResponseStartEvent).
func (ResponseStartPayload) Kind() EventKind { return EventKindResponseStart }

// ResponseID returns the verbatim response ID supplied at construction.
// NewResponseStartPayload rejects empty / whitespace / oversize input
// before storing; Validate re-runs the same checks against the stored
// value.
func (p ResponseStartPayload) ResponseID() string { return p.responseID }

// Model returns the verbatim model identifier supplied at construction.
// NewResponseStartPayload rejects empty / whitespace / oversize input
// before storing; Validate re-runs the same checks against the stored
// value.
func (p ResponseStartPayload) Model() string { return p.model }

// Validate re-runs the construction-time invariants in fixed order:
// response ID (empty → whitespace → length), then model
// (empty → whitespace → length). The order is pinned by
// TestNewResponseStartPayload_ConstructorValidationMatrix — the test
// pins both which sentinel fires and that earlier checks win over
// later ones (e.g., an empty ResponseID fires ErrEmptyResponseID before
// any model check). Idempotent and pure.
func (p ResponseStartPayload) Validate() error {
	if p.responseID == "" {
		return ErrEmptyResponseID
	}
	if isAllWhitespace(p.responseID) {
		return ErrWhitespaceResponseID
	}
	if len(p.responseID) > MaxResponseIDLength {
		return ErrResponseIDTooLong
	}
	if p.model == "" {
		return ErrEmptyResponseModel
	}
	if isAllWhitespace(p.model) {
		return ErrWhitespaceResponseModel
	}
	if len(p.model) > MaxResponseModelLength {
		return ErrResponseModelTooLong
	}
	return nil
}

// NewResponseStartPayload constructs a ResponseStartPayload after
// validating responseID and model in fixed order
// (responseID: empty → whitespace → length; then model:
// empty → whitespace → length). The validation order is fixed by the
// spec and pinned by the matrix test.
//
// Returns the zero value ResponseStartPayload{} with the first
// validation error — never a half-constructed payload.
func NewResponseStartPayload(responseID, model string) (ResponseStartPayload, error) {
	p := ResponseStartPayload{responseID: responseID, model: model}
	if err := p.Validate(); err != nil {
		return ResponseStartPayload{}, err
	}
	return p, nil
}

// NewResponseStartEvent is the producer-side event constructor for
// response.start. It validates first (via NewResponseStartPayload)
// then defers to NewEvent so the package-private atomic counter
// stamps the next Sequence and the sealed eventPayload interface
// stores the payload verbatim. Returns the zero Event with the
// first validation error. Per AI-12 spec § "Scenario: Construct a
// response start event".
func NewResponseStartEvent(responseID, model string) (Event, error) {
	p, err := NewResponseStartPayload(responseID, model)
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// ResponseCompletePayload is the AI-12 event payload for response.complete
// events. It carries the response ID (cross-stream correlation handle),
// the finish reason (canonical 6-value enum from AI-10), and the Usage
// (token counts from AI-10). The struct is value-typed with unexported
// fields; NewResponseCompletePayload is the sanctioned constructor.
// The unexported aiPayload() marker seals the type into the eventPayload
// interface.
//
// Per AI-12 design #1 and #6: payload scope is per-event only; ordering
// invariants (at-most-one-complete-per-producer, mutex with error) are
// NOT validated here. AI-20 / AI-21 own stream-level conformance.
type ResponseCompletePayload struct {
	responseID string
	finish     FinishReason
	usage      Usage
}

// aiPayload is the unexported marker method that seals
// ResponseCompletePayload into the eventPayload interface. See event.go
// for the full contract.
func (ResponseCompletePayload) aiPayload() {}

// Kind returns the canonical EventKind for this payload. Every
// ResponseCompletePayload returns EventKindResponseComplete verbatim,
// regardless of the field values. The seal guarantees that Event.Kind
// (derived from this in NewEvent) cannot disagree with the payload for
// events constructed via the sanctioned entry points.
func (ResponseCompletePayload) Kind() EventKind { return EventKindResponseComplete }

// ResponseID returns the verbatim response ID supplied at construction.
func (p ResponseCompletePayload) ResponseID() string { return p.responseID }

// FinishReason returns the verbatim finish reason supplied at construction.
// NewResponseCompletePayload rejects non-canonical values via
// FinishReason.IsValid before storing; Validate re-runs the same check.
func (p ResponseCompletePayload) FinishReason() FinishReason { return p.finish }

// Usage returns the verbatim usage value supplied at construction.
// NewResponseCompletePayload delegates to Usage.Validate before storing;
// Validate re-runs the same check.
func (p ResponseCompletePayload) Usage() Usage { return p.usage }

// Validate re-runs the construction-time invariants in fixed order:
// response ID (empty → whitespace → length), then finish reason
// (canonical 6-value check via FinishReason.IsValid), then usage
// (delegated to Usage.Validate). The order is pinned by
// TestNewResponseCompletePayload_ConstructorValidationMatrix —
// earlier checks win over later ones so a bad ResponseID cannot be
// masked by a bad finish reason, and a bad finish reason cannot be
// masked by a bad usage. Idempotent and pure.
func (p ResponseCompletePayload) Validate() error {
	if p.responseID == "" {
		return ErrEmptyResponseID
	}
	if isAllWhitespace(p.responseID) {
		return ErrWhitespaceResponseID
	}
	if len(p.responseID) > MaxResponseIDLength {
		return ErrResponseIDTooLong
	}
	if !p.finish.IsValid() {
		return ErrInvalidFinishReason
	}
	if err := p.usage.Validate(); err != nil {
		return err
	}
	return nil
}

// NewResponseCompletePayload constructs a ResponseCompletePayload after
// validating responseID (empty → whitespace → length), finish
// (canonical 6-value check via FinishReason.IsValid), and usage
// (delegated to Usage.Validate). The validation order is fixed by the
// spec and pinned by the matrix test.
//
// Returns the zero value ResponseCompletePayload{} with the first
// validation error — never a half-constructed payload.
func NewResponseCompletePayload(responseID string, finish FinishReason, usage Usage) (ResponseCompletePayload, error) {
	p := ResponseCompletePayload{responseID: responseID, finish: finish, usage: usage}
	if err := p.Validate(); err != nil {
		return ResponseCompletePayload{}, err
	}
	return p, nil
}

// NewResponseCompleteEvent is the producer-side event constructor for
// response.complete. It validates first (via NewResponseCompletePayload)
// then defers to NewEvent so the package-private atomic counter stamps
// the next Sequence and the sealed eventPayload interface stores the
// payload verbatim. Returns the zero Event with the first validation
// error. Per AI-12 spec § "Scenario: Construct a response complete event".
func NewResponseCompleteEvent(responseID string, finish FinishReason, usage Usage) (Event, error) {
	p, err := NewResponseCompletePayload(responseID, finish, usage)
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// IsTerminalKind reports whether k is one of the canonical terminal
// EventKind values: EventKindResponseComplete (successful terminal) or
// EventKindError (failure terminal). The zero value, every other
// reserved kind, and any unregistered string all return false. Consumers
// use this helper to decide whether an event closes a stream.
//
// The helper does NOT validate that a stream has exactly one terminal —
// that ordering invariant is owned by AI-20 (stream testkit) and AI-21
// (conformance suite), not by this package. Per AI-12 design #5
// (ordering scope is documented but not runtime-enforced here).
func IsTerminalKind(k EventKind) bool {
	switch k {
	case EventKindResponseComplete, EventKindError:
		return true
	default:
		return false
	}
}

// AsResponseStart returns the payload of ev as a ResponseStartPayload
// when ev.Kind matches EventKindResponseStart AND the dynamic payload
// type is ResponseStartPayload. The kind check guards the type
// assertion: if a caller (or a future regression) bypasses NewEvent by
// direct struct construction with a mismatched Kind, the helper returns
// (zero, false) rather than panicking on the assertion. Returns
// (zero ResponseStartPayload, false) when either check fails.
//
// Per AI-12 design #3: helpers verify Event.Kind parity BEFORE the type
// assertion so a same-kind-different-payload Event cannot leak through.
// The kind parity is normally guaranteed by NewEvent (which derives Kind
// from payload.Kind()); this helper is the defensive mirror for callers
// that receive Events from outside the producer (e.g., from a vendor
// adapter or test fixture).
func AsResponseStart(ev Event) (ResponseStartPayload, bool) {
	if ev.Kind != EventKindResponseStart {
		return ResponseStartPayload{}, false
	}
	p, ok := ev.Payload.(ResponseStartPayload)
	if !ok {
		return ResponseStartPayload{}, false
	}
	return p, true
}

// AsResponseComplete returns the payload of ev as a ResponseCompletePayload
// when ev.Kind matches EventKindResponseComplete AND the dynamic payload
// type is ResponseCompletePayload. Same parity check as AsResponseStart.
func AsResponseComplete(ev Event) (ResponseCompletePayload, bool) {
	if ev.Kind != EventKindResponseComplete {
		return ResponseCompletePayload{}, false
	}
	p, ok := ev.Payload.(ResponseCompletePayload)
	if !ok {
		return ResponseCompletePayload{}, false
	}
	return p, true
}
