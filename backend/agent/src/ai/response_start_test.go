// Tests for AI-15.1 — the response-start event.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion is written against exactly the
// surface an adapter in another package sees — the convention every Layer 1
// test file in this package follows (event_test.go, tool_call_test.go).
package ai_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// R-ARP-001 — response start is one separately registered event kind with
// its own payload, and its kind is derived from the payload, never stored
// beside it. S-ARP-003 (the exhaustiveness guard covers it, and a scratch
// unregistered kind fails the guard by name) is proven by
// event_registry_test.go's eventKindWitnesses table, extended for this kind
// in the same commit — not repeated here.
func TestResponseStart_Kind_IsRegisteredDistinctAndDerivedFromThePayload(t *testing.T) {
	t.Parallel()

	t.Run("the kind compiles, is non-zero, and is a member of the registered vocabulary (S-ARP-001)", func(t *testing.T) {
		t.Parallel()

		if ai.EventKindResponseStart == 0 {
			t.Fatal("ai.EventKindResponseStart is the zero kind, want a registered non-zero member")
		}
		found := false
		for _, k := range ai.EventKinds() {
			if k == ai.EventKindResponseStart {
				found = true
			}
		}
		if !found {
			t.Errorf("ai.EventKinds() = %v, want it to contain ai.EventKindResponseStart", ai.EventKinds())
		}
	})

	t.Run("distinct from every other registered kind (S-ARP-001)", func(t *testing.T) {
		t.Parallel()

		witness := ai.NewWitnessEvent(1)
		if ai.EventKindResponseStart == witness.Kind() {
			t.Fatal("ai.EventKindResponseStart must not equal the test-witness kind")
		}
	})

	t.Run("the kind is derived from the payload and matches, with no separate lifecycle-phase field (S-ARP-002)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewResponseStart("resp_01AbC", "gpt-test")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		if got := event.Kind(); got != ai.EventKindResponseStart {
			t.Errorf("event.Kind() = %v, want %v", got, ai.EventKindResponseStart)
		}

		// ResponseStart carries exactly its two fields and nothing else — in
		// particular no separate lifecycle-phase discriminator: the kind
		// (asked of the payload) is the only discriminator, never
		// duplicated onto the payload itself.
		typ := reflect.TypeOf(ai.ResponseStart{})
		if got := typ.NumField(); got != 2 {
			t.Fatalf("ai.ResponseStart{} has %d fields, want exactly 2 (response identity, served model), no separate phase field", got)
		}
	})
}

// R-ARP-002 — response start carries provider response identity and served
// model, both externally readable and byte-exact.
func TestResponseStart_ResponseIDAndServedModel_ReadBackByteExactFromAnExternalPackage(t *testing.T) {
	t.Parallel()

	t.Run("both fields read back exactly the bytes supplied, no type switch over unexported types (S-ARP-004)", func(t *testing.T) {
		t.Parallel()

		const (
			responseID  = "resp_01AbC"
			servedModel = "claude-served-x"
		)
		event, err := ai.NewResponseStart(responseID, servedModel)
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}

		rs, ok := event.ResponseStart()
		if !ok {
			t.Fatal("event.ResponseStart() reported no payload on an event of its own kind")
		}
		if got := rs.ResponseID(); got != responseID {
			t.Errorf("rs.ResponseID() = %q, want %q", got, responseID)
		}
		if got := rs.ServedModel(); got != servedModel {
			t.Errorf("rs.ServedModel() = %q, want %q", got, servedModel)
		}
	})

	t.Run("punctuation, mixed case and a vendor prefix survive byte-identically (S-ARP-005)", func(t *testing.T) {
		t.Parallel()

		const (
			responseID  = "msg_01AbC-XyZ_9Q!"
			servedModel = "Provider/Model-v2.1"
		)
		event, err := ai.NewResponseStart(responseID, servedModel)
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		rs, ok := event.ResponseStart()
		if !ok {
			t.Fatal("event.ResponseStart() reported no payload on an event of its own kind")
		}
		if got := rs.ResponseID(); got != responseID {
			t.Errorf("rs.ResponseID() = %q, want %q — not trimmed, not lower-cased, not re-formatted", got, responseID)
		}
		if got := rs.ServedModel(); got != servedModel {
			t.Errorf("rs.ServedModel() = %q, want %q", got, servedModel)
		}
	})

	t.Run("exactly two field-reading accessors exist, no minting/parsing/catalog-lookup entry point (S-ARP-006)", func(t *testing.T) {
		t.Parallel()

		typ := reflect.TypeOf(ai.ResponseStart{})
		var accessors []string
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			if name == "String" || name == "GoString" {
				continue // diagnostic rendering, not a field accessor (V-FAIL-13)
			}
			accessors = append(accessors, name)
		}
		sort.Strings(accessors)
		want := []string{"ResponseID", "ServedModel"}
		if !reflect.DeepEqual(accessors, want) {
			t.Errorf("ai.ResponseStart's exported field accessors = %v, want exactly %v — no minting, parsing or catalog-lookup entry point", accessors, want)
		}
	})
}

// R-ARP-003 — the served model is a distinct concept from the requested
// model; Layer 1 never compares, asserts or couples the two.
func TestResponseStart_ServedModel_IsIndependentOfTheRequestedModel(t *testing.T) {
	t.Parallel()

	t.Run("a served model differing from a hypothetical requested model still succeeds and reports unchanged (S-ARP-007)", func(t *testing.T) {
		t.Parallel()

		const requestedModel = "gpt-5-requested"
		const servedModel = "gpt-5-served-variant" // deliberately different

		event, err := ai.NewResponseStart("resp_02", servedModel)
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure — a served model differing from the requested model is a passing case", err)
		}
		rs, ok := event.ResponseStart()
		if !ok {
			t.Fatal("event.ResponseStart() reported no payload")
		}
		if got := rs.ServedModel(); got != servedModel {
			t.Errorf("rs.ServedModel() = %q, want %q unchanged", got, servedModel)
		}
		if rs.ServedModel() == requestedModel {
			t.Fatal("test fixture error: servedModel must differ from requestedModel to prove independence")
		}
	})

	t.Run("response_start.go contains no reference to the request-side model identity type (S-ARP-008)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "response_start.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing response_start.go: %v", err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if ok && ident.Name == "Request" {
				t.Errorf("response_start.go references identifier %q; R-ARP-003 forbids any comparison or coupling with the request-side model identity", ident.Name)
			}
			return true
		})
	})
}

// R-ARP-004 — both fields are required and non-empty, reported through
// AI-04's ErrEmpty sentinel at a position naming the offending field.
func TestNewResponseStart_EmptyFields_FailWithErrEmptyAtTheOffendingPosition(t *testing.T) {
	t.Parallel()

	t.Run("empty response identity fails with ErrEmpty at response_id (S-ARP-010)", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewResponseStart("", "served-model")
		if err == nil {
			t.Fatal("ai.NewResponseStart with an empty response identity = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ai.ErrEmpty) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "response_id"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("empty served model fails with ErrEmpty at served_model (S-ARP-011)", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewResponseStart("resp_03", "")
		if err == nil {
			t.Fatal("ai.NewResponseStart with an empty served model = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ai.ErrEmpty) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "served_model"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("both fields empty reports exactly one failure, naming the first offending field (S-ARP-012)", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewResponseStart("", "")
		if err == nil {
			t.Fatal("ai.NewResponseStart with both fields empty = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ai.ErrEmpty) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "response_id"; got != want {
			t.Errorf("violation position = %q, want %q (the first field in validation order)", got, want)
		}
	})

	t.Run("the zero value never yields a constructed event carrying two empty strings (S-ARP-013)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewResponseStart("", "")
		if err == nil {
			t.Fatal(`ai.NewResponseStart("", "") = nil error, want a rejection`)
		}
		if event != (ai.Event{}) {
			t.Error("a rejected ai.NewResponseStart returned a non-zero Event, want the zero Event — a caller that ignores the error must not observe a constructed event with two empty strings")
		}
	})
}

// R-ARP-005 — at most one response start per stream, via AI-14's generic
// at-most-one descriptor. S-ARP-016's "no branch names the kind" half is a
// review-only check (design.md's Testing Strategy table), verified by grep
// against stream_check.go and recorded in tasks.md, not repeated here.
func TestResponseStart_AtMostOnePerStream_EnforcedByTheGenericDescriptor(t *testing.T) {
	t.Parallel()

	// completion.go lands in Phase 3 (AI-15.2); a start-then-completion
	// stream is proven legal there (S-ARP-030, empty-response scenario).
	// Here, a single response start is proof enough that one occurrence of
	// an at-most-one kind never trips the rule — Phase 2 stays independent
	// of Phase 3's payload.
	t.Run("a single response start reports no violation (S-ARP-014)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		start, err := ai.NewResponseStart("resp_04", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(start)})
		if got := report.Violation(); got != nil {
			t.Errorf("report.Violation() = %v, want nil", got)
		}
	})

	t.Run("two response-start events on one stream report a violation (S-ARP-015)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		first, err := ai.NewResponseStart("resp_05", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		second, err := ai.NewResponseStart("resp_06", "model-y")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(first), s.Stamp(second)})
		got := report.Violation()
		if got == nil {
			t.Fatal("report.Violation() = nil, want a violation for a second response-start event")
		}
		if !errors.Is(got, ai.ErrDuplicate) {
			t.Errorf("errors.Is(err, ai.ErrDuplicate) = false, want true; err = %v", got)
		}
	})

	t.Run("the registration's cardinality is CardinalityAtMostOne (S-ARP-016)", func(t *testing.T) {
		t.Parallel()

		d, ok := ai.DescriptorOf(ai.EventKindResponseStart)
		if !ok {
			t.Fatal("ai.DescriptorOf(ai.EventKindResponseStart) reported not registered")
		}
		if d.Cardinality != ai.CardinalityAtMostOne {
			t.Errorf("descriptor.Cardinality = %v, want ai.CardinalityAtMostOne", d.Cardinality)
		}
	})
}

// NFR-ARP-D / V-FAIL-13 (task 2.3) — the response start never leaks its
// fields through a diagnostic rendering. Mirrors event_test.go's
// TestEvent_StringAndGoString_RenderOnlyKindAndSequence and tool_call.go's
// ToolCall.String() posture: the first exported event payload this package
// registers moves the same posture onto it.
func TestResponseStart_StringAndGoString_NameTheTypeAndCarryNoField(t *testing.T) {
	t.Parallel()

	rs, ok := mustResponseStart(t, "resp_canary_9f8e7d", "served-canary-model").ResponseStart()
	if !ok {
		t.Fatal("event.ResponseStart() reported no payload on an event of its own kind")
	}

	for _, verb := range []string{"%v", "%s", "%+v"} {
		rendered := fmt.Sprintf(verb, rs)
		if rendered != "responsestart" {
			t.Errorf("fmt.Sprintf(%q, rs) = %q, want %q", verb, rendered, "responsestart")
		}
	}
	if got := fmt.Sprintf("%#v", rs); got != "responsestart" {
		t.Errorf(`fmt.Sprintf("%%#v", rs) = %q, want %q`, got, "responsestart")
	}
	if got := rs.GoString(); got != rs.String() {
		t.Errorf("rs.GoString() = %q, want it to equal rs.String() = %q", got, rs.String())
	}
	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		rendered := fmt.Sprintf(verb, rs)
		if strings.Contains(rendered, "resp_canary_9f8e7d") || strings.Contains(rendered, "served-canary-model") {
			t.Errorf("fmt.Sprintf(%q, rs) = %q, which reproduces a field value", verb, rendered)
		}
	}
}

// mustResponseStart constructs a response-start event or fails the test.
func mustResponseStart(t *testing.T, responseID, servedModel string) ai.Event {
	t.Helper()
	event, err := ai.NewResponseStart(responseID, servedModel)
	if err != nil {
		t.Fatalf("ai.NewResponseStart(%q, %q) returned %v, want no failure", responseID, servedModel, err)
	}
	return event
}
