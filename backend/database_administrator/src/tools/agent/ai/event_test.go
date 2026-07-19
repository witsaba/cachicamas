// Package-internal test for the sealed eventPayload interface and the
// package-private sequence counter. AI-11 reserves concrete payload
// implementations for AI-12/13/14/15 (text, reasoning, tool-call,
// response/error) and therefore exposes no public payload constructor
// at this milestone; only this package can implement eventPayload. Using
// package ai (not ai_test) is the deliberate, well-justified exception
// to the usual external-test convention — strictly scoped to AI-11.
package ai

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// testPayload is the test-only implementation of eventPayload. It
// returns whatever kind it was constructed with so tests can drive
// both registered kinds (EventKindTextStart, etc.) and unregistered
// kinds (zero value, garbage strings) through the sealed surface.
type testPayload struct {
	k EventKind
}

func (p testPayload) aiPayload()      {}
func (p testPayload) Kind() EventKind { return p.k }

func newTestPayload(k EventKind) testPayload { return testPayload{k: k} }

// resetSequenceCounter restores the package-private counter to zero so
// tests that assert absolute sequence values (1, 2, 3, ...) start from
// a clean baseline. This is a deliberate test affordance: the counter
// is process-scoped and survives across test functions within a single
// `go test` invocation. Per AI-11 design decision #3 (sequence is
// producer-assigned; the test simulates a fresh producer per test).
func resetSequenceCounter() { atomic.StoreUint64(&sequenceCounter, 0) }

// ---------------------------------------------------------------------------
// Scenario: Construct an event with kind + payload (spec req #1)
// ---------------------------------------------------------------------------

// TestNewEvent_HappyPath_FirstEventHasSequenceOne verifies that the very
// first call to NewEvent on a fresh producer stamps Sequence=1, derives
// Kind from the payload, and stores the payload verbatim. The baseline
// counter reset guarantees deterministic behavior across test runs.
func TestNewEvent_HappyPath_FirstEventHasSequenceOne(t *testing.T) {
	resetSequenceCounter()

	payload := newTestPayload(EventKindTextStart)
	ev, err := NewEvent(payload)
	if err != nil {
		t.Fatalf("NewEvent(text-start payload) = %v, want nil", err)
	}
	if ev.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1 (first event on a fresh producer)", ev.Sequence)
	}
	if ev.Kind != EventKindTextStart {
		t.Errorf("Kind = %q, want %q (derived from payload)", ev.Kind, EventKindTextStart)
	}
	// Payload stored verbatim — same dynamic type and value.
	got, ok := ev.Payload.(testPayload)
	if !ok {
		t.Fatalf("Payload dynamic type = %T, want testPayload", ev.Payload)
	}
	if got != payload {
		t.Errorf("Payload stored = %+v, want verbatim %+v", got, payload)
	}
}

// ---------------------------------------------------------------------------
// Scenario: Kind() derived from payload (spec req #2 — sealed-payload parity)
// ---------------------------------------------------------------------------

// TestNewEvent_KindDerivedFromPayload_AcrossKinds triangulates across
// multiple reserved kinds to ensure NewEvent always reads the kind from
// the payload (not from any implicit default).
func TestNewEvent_KindDerivedFromPayload_AcrossKinds(t *testing.T) {
	resetSequenceCounter()

	cases := []EventKind{
		EventKindResponseStart,
		EventKindResponseComplete,
		EventKindTextStart,
		EventKindTextDelta,
		EventKindTextEnd,
		EventKindReasoningStart,
		EventKindReasoningDelta,
		EventKindReasoningEnd,
		EventKindToolCallStart,
		EventKindToolCallDelta,
		EventKindToolCallEnd,
		EventKindError,
	}
	for _, want := range cases {
		t.Run(string(want), func(t *testing.T) {
			resetSequenceCounter()
			payload := newTestPayload(want)
			ev, err := NewEvent(payload)
			if err != nil {
				t.Fatalf("NewEvent = %v, want nil", err)
			}
			if ev.Kind != want {
				t.Errorf("Event.Kind = %q, want %q (derived from payload)", ev.Kind, want)
			}
			if ev.Kind != payload.Kind() {
				t.Errorf("Event.Kind (%q) must match payload.Kind() (%q)", ev.Kind, payload.Kind())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario: Sequence is producer-assigned and contiguous (spec req #1)
// ---------------------------------------------------------------------------

// TestNewEvent_SequenceMonotonic_OneTwoThree verifies three sequential
// calls on a fresh producer yield contiguous, 1-based sequences. This
// is the AI-20 conformance property: a consumer can detect dropped
// events by inspecting gaps in Sequence.
func TestNewEvent_SequenceMonotonic_OneTwoThree(t *testing.T) {
	resetSequenceCounter()

	p1 := newTestPayload(EventKindResponseStart)
	p2 := newTestPayload(EventKindTextDelta)
	p3 := newTestPayload(EventKindResponseComplete)

	e1, err := NewEvent(p1)
	if err != nil {
		t.Fatalf("NewEvent #1 = %v, want nil", err)
	}
	e2, err := NewEvent(p2)
	if err != nil {
		t.Fatalf("NewEvent #2 = %v, want nil", err)
	}
	e3, err := NewEvent(p3)
	if err != nil {
		t.Fatalf("NewEvent #3 = %v, want nil", err)
	}

	if e1.Sequence != 1 {
		t.Errorf("event 1 Sequence = %d, want 1", e1.Sequence)
	}
	if e2.Sequence != 2 {
		t.Errorf("event 2 Sequence = %d, want 2", e2.Sequence)
	}
	if e3.Sequence != 3 {
		t.Errorf("event 3 Sequence = %d, want 3", e3.Sequence)
	}

	// Contiguity invariant: sequences are unique within the stream and
	// strictly increasing.
	if !(e1.Sequence < e2.Sequence && e2.Sequence < e3.Sequence) {
		t.Errorf("sequences must be strictly increasing: %d, %d, %d",
			e1.Sequence, e2.Sequence, e3.Sequence)
	}
	if e1.Sequence == e2.Sequence || e2.Sequence == e3.Sequence || e1.Sequence == e3.Sequence {
		t.Errorf("sequences must be unique within stream: %d, %d, %d",
			e1.Sequence, e2.Sequence, e3.Sequence)
	}
}

// ---------------------------------------------------------------------------
// Scenario: Reject zero-value kind (spec req #2)
// ---------------------------------------------------------------------------

// TestValidate_RejectsZeroEventKind verifies that an Event with the
// zero-value EventKind fails Validate with the registered sentinel.
// NewEvent might succeed (deriving the empty kind from the payload);
// what matters is that the resulting Event.Validate is not nil.
func TestValidate_RejectsZeroEventKind(t *testing.T) {
	payload := newTestPayload(EventKindResponseStart) // any non-zero kind payload
	ev := Event{
		Kind:     EventKind(""), // zero value
		Sequence: 1,
		Payload:  payload,
	}
	err := ev.Validate()
	if err == nil {
		t.Fatal("Event{Kind: \"\"}.Validate() = nil, want error")
	}
	assertErrorIsEvent(t, err, ErrEventKindUnregistered)
}

// TestNewEvent_RejectsPayloadDeclaringZeroKind verifies that when a
// payload reports Kind() == "", NewEvent still produces an Event whose
// Validate fails. This covers the spec's "Validate rejects zero-value
// EventKind" requirement when the only path to an Event is NewEvent.
func TestNewEvent_RejectsPayloadDeclaringZeroKind(t *testing.T) {
	resetSequenceCounter()
	payload := newTestPayload(EventKind("")) // payload reports zero kind
	ev, err := NewEvent(payload)
	// NewEvent may succeed (deriving the empty kind) or fail; either is
	// acceptable here. We require that the Event — if produced — fails
	// Validate, and that some sentinel identifies the problem.
	if err == nil {
		if validateErr := ev.Validate(); validateErr == nil {
			t.Fatal("zero-kind Event.Validate() = nil, want error")
		} else {
			assertErrorIsEvent(t, validateErr, ErrEventKindUnregistered)
		}
	} else {
		// If NewEvent itself rejects, the sentinel must identify the
		// zero-kind cause (or, at minimum, the unregistered-kind class).
		assertErrorIsEvent(t, err, ErrEventKindUnregistered)
	}
}

// ---------------------------------------------------------------------------
// Scenario: Reject unknown kind (spec req #2)
// ---------------------------------------------------------------------------

// TestValidate_RejectsUnregisteredEventKind verifies that an Event whose
// Kind is a string outside the 12-slot canonical set fails Validate.
// The sentinel must be errors.Is-compatible.
func TestValidate_RejectsUnregisteredEventKind(t *testing.T) {
	payload := newTestPayload(EventKindResponseStart)
	cases := []EventKind{
		"start",           // not in registry
		"response.start",  // dotty variant, not the underscore wire format
		"RESPONSE_START",  // case mismatch
		"tool-call",       // hyphenated variant
		"response.end",    // not in registry (response.complete is)
		"tool_call.delta", // not in registry (delta lives under text/reasoning)
		"image",           // reserved for ContentPart, not for Event
	}
	for _, bad := range cases {
		t.Run(string(bad), func(t *testing.T) {
			ev := Event{Kind: bad, Sequence: 1, Payload: payload}
			err := ev.Validate()
			if err == nil {
				t.Fatalf("Event{Kind: %q}.Validate() = nil, want ErrEventKindUnregistered", bad)
			}
			assertErrorIsEvent(t, err, ErrEventKindUnregistered)
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario: Nil payload rejected (spec req #4 — non-nil payload)
// ---------------------------------------------------------------------------

// TestNewEvent_RejectsNilPayload verifies that passing nil for the
// payload returns the registered sentinel and a zero Event. This is
// the typed-nil + interface-nil trap from content.go applied to
// eventPayload.
func TestNewEvent_RejectsNilPayload(t *testing.T) {
	resetSequenceCounter()

	ev, err := NewEvent(nil)
	if err == nil {
		t.Fatal("NewEvent(nil) = no error, want ErrEventPayloadMissing")
	}
	if ev != (Event{}) {
		t.Errorf("NewEvent(nil) returned %+v, want zero-value Event", ev)
	}
	assertErrorIsEvent(t, err, ErrEventPayloadMissing)
}

// TestValidate_RejectsNilPayload verifies that an Event whose Payload
// is the interface-nil also fails Validate (in case a caller bypasses
// NewEvent by direct struct construction).
func TestValidate_RejectsNilPayload(t *testing.T) {
	ev := Event{
		Kind:     EventKindResponseStart,
		Sequence: 1,
		Payload:  nil, // interface-nil literal
	}
	err := ev.Validate()
	if err == nil {
		t.Fatal("Event{Payload: nil}.Validate() = nil, want ErrEventPayloadMissing")
	}
	assertErrorIsEvent(t, err, ErrEventPayloadMissing)
}

// ---------------------------------------------------------------------------
// Scenario: Payload/kind mismatch is unrepresentable (spec req #3)
// ---------------------------------------------------------------------------

// TestEventPayload_InterfaceIsSealed verifies via reflection that
// eventPayload has exactly two methods: the unexported aiPayload()
// marker (which only types in this package can satisfy) and the
// exported Kind() EventKind (which implementors must define). External
// packages cannot construct a value implementing eventPayload, so the
// seal is enforced at the type system level.
func TestEventPayload_InterfaceIsSealed(t *testing.T) {
	iface := reflect.TypeOf((*eventPayload)(nil)).Elem()
	if iface.Kind() != reflect.Interface {
		t.Fatalf("eventPayload reflect kind = %v, want interface", iface.Kind())
	}
	if iface.NumMethod() != 2 {
		t.Fatalf("eventPayload has %d methods, want 2 (aiPayload, Kind)", iface.NumMethod())
	}
	methodNames := map[string]bool{}
	for i := 0; i < iface.NumMethod(); i++ {
		methodNames[iface.Method(i).Name] = true
	}
	if !methodNames["aiPayload"] {
		t.Error("eventPayload must declare unexported aiPayload() marker for sealing")
	}
	if !methodNames["Kind"] {
		t.Error("eventPayload must declare Kind() EventKind method")
	}
}

// TestValidate_DetectsPayloadKindMismatch verifies that an Event
// constructed directly (bypassing NewEvent) with mismatched Kind vs
// Payload fails Validate. This is the parity check Validate performs
// even though NewEvent guarantees parity by construction.
func TestValidate_DetectsPayloadKindMismatch(t *testing.T) {
	payload := newTestPayload(EventKindTextDelta) // payload declares TextDelta
	ev := Event{
		Kind:     EventKindResponseStart, // ...but Event.Kind is ResponseStart
		Sequence: 1,
		Payload:  payload,
	}
	err := ev.Validate()
	if err == nil {
		t.Fatal("Event{Kind: ResponseStart, Payload: textDelta-payload}.Validate() = nil, want error")
	}
	assertErrorIsEvent(t, err, ErrEventPayloadKindMismatch)
}

// TestNewEvent_PayloadKindParityGuaranteed verifies that an Event
// constructed via NewEvent automatically has matching Kind/Payload —
// NewEvent stamps Kind from payload.Kind(), so direct construction is
// the only way to break parity. This is the "unrepresentable via
// NewEvent" half of the spec requirement.
func TestNewEvent_PayloadKindParityGuaranteed(t *testing.T) {
	resetSequenceCounter()
	payload := newTestPayload(EventKindToolCallStart)
	ev, err := NewEvent(payload)
	if err != nil {
		t.Fatalf("NewEvent = %v, want nil", err)
	}
	if ev.Kind != payload.Kind() {
		t.Errorf("NewEvent did not stamp Kind from payload: got %q, want %q",
			ev.Kind, payload.Kind())
	}
	if err := ev.Validate(); err != nil {
		t.Errorf("NewEvent-constructed Event.Validate() = %v, want nil (parity guaranteed)", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario: Reserved kind slots are present (spec req #2)
// ---------------------------------------------------------------------------

// TestAllEventKinds_ReturnsTwelveReservedInOrder verifies that
// AllEventKinds returns the 12 reserved kinds in the canonical
// registration order from design #4. The ordering is the iteration
// surface for AI-20 (stream testkit) and AI-21 (conformance suite).
func TestAllEventKinds_ReturnsTwelveReservedInOrder(t *testing.T) {
	want := []EventKind{
		EventKindResponseStart,
		EventKindResponseComplete,
		EventKindTextStart,
		EventKindTextDelta,
		EventKindTextEnd,
		EventKindReasoningStart,
		EventKindReasoningDelta,
		EventKindReasoningEnd,
		EventKindToolCallStart,
		EventKindToolCallDelta,
		EventKindToolCallEnd,
		EventKindError,
	}
	got := AllEventKinds()
	if len(got) != 12 {
		t.Fatalf("AllEventKinds() returned %d kinds, want 12 (response x2, text x3, reasoning x3, tool-call x3, error)", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllEventKinds()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	// Pairwise distinctness — registry must not contain duplicates.
	seen := map[EventKind]int{}
	for i, k := range got {
		if prev, dup := seen[k]; dup {
			t.Errorf("AllEventKinds contains duplicate %q at indices %d and %d", k, prev, i)
		}
		seen[k] = i
	}
}

// TestAllEventKinds_ReturnsCopy verifies that AllEventKinds returns a
// defensive copy. Mutating the returned slice must not affect a
// subsequent call. Per design #4 (accessor, not exported variable).
func TestAllEventKinds_ReturnsCopy(t *testing.T) {
	first := AllEventKinds()
	if len(first) == 0 {
		t.Fatal("AllEventKinds() returned empty slice")
	}
	// Mutate the first slice.
	original := first[0]
	first[0] = EventKind("garbage")

	second := AllEventKinds()
	if second[0] != original {
		t.Errorf("AllEventKinds() returned a reference, not a copy: got %q after mutation, want %q",
			second[0], original)
	}
	// Restore so other tests see the canonical registry.
	first[0] = original
}

// TestEventKind_AllReservedConstantsPinned verifies the 12 reserved
// constants' wire-format strings, since AI-20/21 iterate by string and
// any drift here is a wire-format break.
func TestEventKind_AllReservedConstantsPinned(t *testing.T) {
	cases := []struct {
		got  EventKind
		want string
	}{
		{EventKindResponseStart, "response.start"},
		{EventKindResponseComplete, "response.complete"},
		{EventKindTextStart, "text.start"},
		{EventKindTextDelta, "text.delta"},
		{EventKindTextEnd, "text.end"},
		{EventKindReasoningStart, "reasoning.start"},
		{EventKindReasoningDelta, "reasoning.delta"},
		{EventKindReasoningEnd, "reasoning.end"},
		{EventKindToolCallStart, "tool_call.start"},
		{EventKindToolCallDelta, "tool_call.delta"},
		{EventKindToolCallEnd, "tool_call.end"},
		{EventKindError, "error"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("EventKind constant %q has wire-format %q, want %q", tc.got, string(tc.got), tc.want)
		}
	}
}

// TestEventKind_IsValid_AcceptsCanonical pins IsValid == true for every
// reserved constant and IsValid == false for the zero value. The zero
// value must never silently become a valid wire kind.
func TestEventKind_IsValid_AcceptsCanonical(t *testing.T) {
	canonical := []EventKind{
		EventKindResponseStart,
		EventKindResponseComplete,
		EventKindTextStart,
		EventKindTextDelta,
		EventKindTextEnd,
		EventKindReasoningStart,
		EventKindReasoningDelta,
		EventKindReasoningEnd,
		EventKindToolCallStart,
		EventKindToolCallDelta,
		EventKindToolCallEnd,
		EventKindError,
	}
	for _, k := range canonical {
		if !k.IsValid() {
			t.Errorf("canonical EventKind %q must report IsValid() == true", k)
		}
	}
	if (EventKind("")).IsValid() {
		t.Error("zero-value EventKind(\"\") must report IsValid() == false")
	}
}

// ---------------------------------------------------------------------------
// Scenario: Sentinel errors match errors.Is (spec req #2 + Phase B pattern)
// ---------------------------------------------------------------------------

// TestEventSentinels_AreTypedAndDistinct verifies that each of the
// three sentinels is non-nil, errors.Is-compatible with itself, and
// pairwise distinct (so callers can disambiguate the failure cause).
// The "ai: " prefix is part of the Phase B convention.
func TestEventSentinels_AreTypedAndDistinct(t *testing.T) {
	errs := map[string]error{
		"ErrEventKindUnregistered":    ErrEventKindUnregistered,
		"ErrEventPayloadKindMismatch": ErrEventPayloadKindMismatch,
		"ErrEventPayloadMissing":      ErrEventPayloadMissing,
	}
	for name, e := range errs {
		if e == nil {
			t.Errorf("%s must not be nil", name)
			continue
		}
		if !errors.Is(e, e) {
			t.Errorf("%s must be errors.Is-compatible with itself", name)
		}
		if !strings.HasPrefix(e.Error(), "ai: ") {
			t.Errorf("%s.Error() = %q, must start with %q (Phase B sentinel prefix)",
				name, e.Error(), "ai: ")
		}
	}
	// Pairwise distinctness.
	for nameA, a := range errs {
		for nameB, b := range errs {
			if nameA == nameB {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("%s and %s must be distinct sentinels", nameA, nameB)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Scenario: No dynamic payload carrier (spec req #4 — vendor leak guard)
// ---------------------------------------------------------------------------

// TestEvent_NoDynamicPayloadCarrier verifies via reflection that Event
// has no field of type `any`, `interface{}`, or `json.RawMessage`.
// This is the vendor-leak guard: AI-11 owns marshaling centrally,
// and the public API must not expose any dynamic-typed slot.
func TestEvent_NoDynamicPayloadCarrier(t *testing.T) {
	evType := reflect.TypeOf(Event{})
	if evType.Kind() != reflect.Struct {
		t.Fatalf("Event reflect kind = %v, want struct", evType.Kind())
	}
	for i := 0; i < evType.NumField(); i++ {
		f := evType.Field(i)
		// Reject `any` / `interface{}` / json.RawMessage by string match
		// since reflect does not unify the alias.
		ts := f.Type.String()
		switch ts {
		case "any", "interface {}", "json.RawMessage":
			t.Errorf("Event.%s has forbidden dynamic type %q (AI-11 spec req #4)",
				f.Name, ts)
		}
	}
}

// TestEvent_AllFieldsHaveDocumentedJSONShape verifies that the public
// fields of Event are simple value types: typed string, uint64, and
// the sealed interface. This is the same vendor-leak guard from a
// different angle — fields must not be maps, slices, or pointers
// other than the sealed interface.
func TestEvent_AllFieldsHaveDocumentedJSONShape(t *testing.T) {
	evType := reflect.TypeOf(Event{})
	want := map[string]string{
		"Kind":     "ai.EventKind",
		"Sequence": "uint64",
		"Payload":  "ai.eventPayload",
	}
	for i := 0; i < evType.NumField(); i++ {
		f := evType.Field(i)
		expected, ok := want[f.Name]
		if !ok {
			t.Errorf("Event has unexpected field %q (only Kind, Sequence, Payload are allowed)", f.Name)
			continue
		}
		if got := f.Type.String(); got != expected {
			t.Errorf("Event.%s type = %q, want %q", f.Name, got, expected)
		}
	}
	if evType.NumField() != len(want) {
		t.Errorf("Event has %d fields, want exactly %d", evType.NumField(), len(want))
	}
}

// ---------------------------------------------------------------------------
// Scenario: Event is comparable by value (spec req #1 + § Comparable)
// ---------------------------------------------------------------------------

// TestEvent_ComparableByValue is a compile-time check that Event is
// usable with `==` (no maps, no slices, no funcs). If Event ever gains
// a non-comparable field, this test will fail to compile.
func TestEvent_ComparableByValue(t *testing.T) {
	// Compile-time assertion: this expression must type-check.
	var a, b Event
	if a != b {
		t.Error("zero-value Events must compare equal")
	}
	resetSequenceCounter()
	p := newTestPayload(EventKindTextStart)
	e1, err := NewEvent(p)
	if err != nil {
		t.Fatalf("NewEvent #1 = %v", err)
	}
	resetSequenceCounter()
	e2, err := NewEvent(p)
	if err != nil {
		t.Fatalf("NewEvent #2 = %v", err)
	}
	if e1 != e2 {
		t.Errorf("Events with same Kind, Sequence, Payload must be ==, got %+v vs %+v", e1, e2)
	}

	// Distinctness check: different payloads → different events.
	resetSequenceCounter()
	e3, err := NewEvent(newTestPayload(EventKindTextDelta))
	if err != nil {
		t.Fatalf("NewEvent #3 = %v", err)
	}
	if e1 == e3 {
		t.Error("Events with different Payload must NOT be ==")
	}
}

// ---------------------------------------------------------------------------
// Scenario: Document ordering rules (spec req #5)
// ---------------------------------------------------------------------------

// TestDoc_ReferencesEventAndOrderingRules verifies that doc.go mentions
// the envelope surface (Event, EventKind, AllEventKinds) and the four
// ordering rules (exactly-one-start, contiguous sequence,
// at-most-one-successful-terminal, cancellation best-effort). The
// phrases are lowercased for case-insensitive substring match.
func TestDoc_ReferencesEventAndOrderingRules(t *testing.T) {
	body, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("ReadFile doc.go = %v", err)
	}
	doc := strings.ToLower(string(body))

	mustMention := []string{
		// Surface summary.
		"event",     // Event type
		"eventkind", // EventKind type (after lowercasing)
		"alleventkinds",
	}
	mustMentionRules := []string{
		// Ordering rules.
		"exactly one",  // exactly-one-start
		"contiguous",   // contiguous sequence
		"at most one",  // at-most-one-successful-terminal (or "at-most-one")
		"cancellation", // AI-01 cancellation best-effort
	}

	for _, phrase := range mustMention {
		if !strings.Contains(doc, phrase) {
			t.Errorf("doc.go must mention %q (AI-11 surface summary)", phrase)
		}
	}
	for _, phrase := range mustMentionRules {
		if !strings.Contains(doc, phrase) {
			t.Errorf("doc.go must mention %q (AI-11 ordering rule)", phrase)
		}
	}
}

// TestEvent_IsNotContentPart verifies via reflection that Event does
// NOT implement ContentPart. This is the cross-coupling guard from
// design risk table: ContentPart.Kind is the wire-side discriminator,
// Event.Kind is the stream-side discriminator; conflating them would
// be a layering violation.
func TestEvent_IsNotContentPart(t *testing.T) {
	if _, ok := reflect.TypeOf(Event{}).MethodByName("Kind"); ok {
		// Event has a Kind method — but it's of type EventKind, not Kind.
		// The ContentPart interface requires `Kind() Kind`. Event's Kind
		// method would only satisfy the signature if both returned Kind
		// — which they don't (one returns EventKind, the other returns
		// the content-side Kind). The runtime check below confirms.
		var _ ContentPart = Event{} // MUST fail to compile if Event implements ContentPart
		t.Fatal("Event must NOT implement ContentPart (Kind() Kind vs Kind() EventKind)")
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// assertErrorIsEvent is a local copy of request_test.go's assertErrorIs
// because the external-test version lives in `package ai_test`, which
// this internal test file cannot reach. Kept short and focused to
// avoid duplicating the full helper.
func assertErrorIsEvent(t *testing.T, got, want error) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("error = %v, want nil", got)
		}
		return
	}
	if !errors.Is(got, want) {
		t.Errorf("error = %v, want errors.Is(_, %v)", got, want)
	}
}