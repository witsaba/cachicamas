// Tests for AI-15.2 — the completion event.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion is written against exactly the
// surface an adapter in another package sees — the convention every Layer 1
// test file in this package follows (event_test.go, response_start_test.go).
package ai_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// R-ARP-006 — completion is one separately registered event kind with its
// own payload, kind derived, distinct from response start and every other
// kind. S-ARP-018 (the exhaustiveness guard covers it alongside
// response-start) is proven by event_registry_test.go's eventKindWitnesses
// table, extended for this kind in the same commit — not repeated here.
func TestCompletion_Kind_IsRegisteredDistinctAndDerivedFromThePayload(t *testing.T) {
	t.Parallel()

	t.Run("the kind compiles, is non-zero, and is a member of the registered vocabulary (S-ARP-017)", func(t *testing.T) {
		t.Parallel()

		if ai.EventKindCompletion == 0 {
			t.Fatal("ai.EventKindCompletion is the zero kind, want a registered non-zero member")
		}
		found := false
		for _, k := range ai.EventKinds() {
			if k == ai.EventKindCompletion {
				found = true
			}
		}
		if !found {
			t.Errorf("ai.EventKinds() = %v, want it to contain ai.EventKindCompletion", ai.EventKinds())
		}
	})

	t.Run("distinct from response start and every other registered kind (S-ARP-017)", func(t *testing.T) {
		t.Parallel()

		if ai.EventKindCompletion == ai.EventKindResponseStart {
			t.Fatal("ai.EventKindCompletion must not equal ai.EventKindResponseStart")
		}
		witness := ai.NewWitnessEvent(1)
		if ai.EventKindCompletion == witness.Kind() {
			t.Fatal("ai.EventKindCompletion must not equal the test-witness kind")
		}
	})

	t.Run("the kind is derived from the payload and matches (S-ARP-017)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		if got := event.Kind(); got != ai.EventKindCompletion {
			t.Errorf("event.Kind() = %v, want %v", got, ai.EventKindCompletion)
		}
	})
}

// R-ARP-007 — completion embeds AI-13's finish reason and usage unchanged,
// with no new value, field, parallel type, or re-encoding.
func TestCompletion_FinishReasonAndUsage_EmbedAI13Unchanged(t *testing.T) {
	t.Parallel()

	t.Run("both values read back exactly what was supplied, same AI-13 types (S-ARP-019)", func(t *testing.T) {
		t.Parallel()

		reason := ai.FinishReasonToolCalls
		usage := ai.Usage{Input: ai.Tokens(120), Output: ai.Tokens(45)}

		event, err := ai.NewCompletion(reason, usage)
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		c, ok := event.Completion()
		if !ok {
			t.Fatal("event.Completion() reported no payload on an event of its own kind")
		}

		// gotReason/gotUsage are the same AI-13 types by construction: this
		// line would not compile against a parallel FinishReason or Usage
		// type, since FinishReason()/Usage() are statically typed to
		// return AI-13's own ai.FinishReason / ai.Usage.
		gotReason := c.FinishReason()
		if gotReason != reason {
			t.Errorf("c.FinishReason() = %v, want %v", gotReason, reason)
		}
		gotUsage := c.Usage()
		if gotUsage != usage {
			t.Errorf("c.Usage() = %+v, want %+v", gotUsage, usage)
		}
	})

	t.Run("an invalid finish reason fails with ErrNotInVocabulary at finish_reason (S-ARP-020)", func(t *testing.T) {
		t.Parallel()

		var zero ai.FinishReason // the zero value is outside the vocabulary
		_, err := ai.NewCompletion(zero, ai.Usage{})
		if err == nil {
			t.Fatal("ai.NewCompletion with the zero-value finish reason = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("errors.Is(err, ai.ErrNotInVocabulary) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "finish_reason"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("a negative token count fails with ErrOutOfRange naming the offending count (S-ARP-021)", func(t *testing.T) {
		t.Parallel()

		usage := ai.Usage{Output: ai.Tokens(-5)}
		_, err := ai.NewCompletion(ai.FinishReasonStop, usage)
		if err == nil {
			t.Fatal("ai.NewCompletion with a negative token count = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("errors.Is(err, ai.ErrOutOfRange) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "usage.output"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("completion.go declares no new finish-reason/usage member, field or parallel type (S-ARP-022)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "completion.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing completion.go: %v", err)
		}

		var declaredTypes []string
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			switch gen.Tok {
			case token.TYPE:
				for _, spec := range gen.Specs {
					if typed, ok := spec.(*ast.TypeSpec); ok {
						declaredTypes = append(declaredTypes, typed.Name.Name)
					}
				}
			case token.CONST:
				t.Errorf("completion.go declares a top-level const block; R-ARP-007 forbids a new finish-reason vocabulary member here (AI-13 owns that closed vocabulary)")
			}
		}
		want := []string{"Completion"}
		if !reflect.DeepEqual(declaredTypes, want) {
			t.Errorf("completion.go declares types %v, want exactly %v — no parallel FinishReason/Usage type", declaredTypes, want)
		}
	})
}

// R-ARP-008 — AI-13.3's absence-versus-zero property survives the event
// boundary unchanged.
func TestCompletion_Usage_AbsenceSurvivesTheEventBoundary(t *testing.T) {
	t.Parallel()

	t.Run("every count absent stays absent, none reports present-zero (S-ARP-023)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		c, ok := event.Completion()
		if !ok {
			t.Fatal("event.Completion() reported no payload")
		}
		usage := c.Usage()
		for name, count := range map[string]ai.TokenCount{
			"input": usage.Input, "output": usage.Output,
			"cache_read": usage.CacheRead, "cache_write": usage.CacheWrite, "reasoning": usage.Reasoning,
		} {
			if _, present := count.Count(); present {
				t.Errorf("usage.%s reports present, want absent", name)
			}
		}
	})

	t.Run("Tokens(0) present stays distinguishable from an absent count (S-ARP-024)", func(t *testing.T) {
		t.Parallel()

		usage := ai.Usage{Input: ai.Tokens(0)} // explicit reported-nought input, absent output
		event, err := ai.NewCompletion(ai.FinishReasonStop, usage)
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		c, ok := event.Completion()
		if !ok {
			t.Fatal("event.Completion() reported no payload")
		}
		got := c.Usage()

		inputCount, inputPresent := got.Input.Count()
		if !inputPresent {
			t.Error("got.Input reports absent, want present (Tokens(0) was supplied)")
		}
		if inputCount != 0 {
			t.Errorf("got.Input's count = %d, want 0", inputCount)
		}
		if _, outputPresent := got.Output.Count(); outputPresent {
			t.Error("got.Output reports present, want absent — it was never supplied")
		}
	})

	t.Run("an entirely absent usage record validates successfully (S-ARP-025)", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Errorf("ai.NewCompletion with an all-absent usage returned %v, want no failure — presence is never required", err)
		}
	})
}

// R-ARP-009 — completion is terminal via AI-14's generic descriptor-driven
// terminal property. S-ARP-026's "no branch names the kind" half is a
// review-only check (design.md's Testing Strategy table), verified by grep
// against stream_check.go and recorded in tasks.md, not repeated here.
func TestCompletion_IsTerminal_EnforcedByTheGenericDescriptor(t *testing.T) {
	t.Parallel()

	t.Run("the registration's terminal property is set (S-ARP-026)", func(t *testing.T) {
		t.Parallel()

		d, ok := ai.DescriptorOf(ai.EventKindCompletion)
		if !ok {
			t.Fatal("ai.DescriptorOf(ai.EventKindCompletion) reported not registered")
		}
		if !d.Terminal {
			t.Error("descriptor.Terminal = false, want true")
		}
	})

	t.Run("a stream ending in a completion reports no violation and is identified as terminated (S-ARP-027)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		start, err := ai.NewResponseStart("resp_10", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		done, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(start), s.Stamp(done)})
		if got := report.Violation(); got != nil {
			t.Errorf("report.Violation() = %v, want nil", got)
		}
		if !report.Terminated() {
			t.Error("report.Terminated() = false, want true — the stream ends in a completion")
		}
	})

	t.Run("an event of any kind after a completion reports a violation (S-ARP-028)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		done, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		after, err := ai.NewResponseStart("resp_11", "model-y")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(done), s.Stamp(after)})
		got := report.Violation()
		if got == nil {
			t.Fatal("report.Violation() = nil, want a violation for an event following a completion")
		}
		if !errors.Is(got, ai.ErrMisplaced) {
			t.Errorf("errors.Is(err, ai.ErrMisplaced) = false, want true; err = %v", got)
		}
	})

	t.Run("two completion events on one stream report a violation (S-ARP-029)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		first, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		second, err := ai.NewCompletion(ai.FinishReasonLength, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(first), s.Stamp(second)})
		got := report.Violation()
		if got == nil {
			t.Fatal("report.Violation() = nil, want a violation for a second completion event")
		}
		if !errors.Is(got, ai.ErrDuplicate) {
			t.Errorf("errors.Is(err, ai.ErrDuplicate) = false, want true; err = %v", got)
		}
	})
}

// R-ARP-010 — an empty response (response start, then completion, no
// content between them) is legal and distinguishable from every failure
// shape available before AI-19.
func TestEmptyResponseStream_ResponseStartThenCompletion_IsLegalAndDistinguishable(t *testing.T) {
	t.Parallel()

	t.Run("no violation, and the stream is reported as normally terminated (S-ARP-030)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		start, err := ai.NewResponseStart("resp_12", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		done, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(start), s.Stamp(done)})
		if got := report.Violation(); got != nil {
			t.Errorf("report.Violation() = %v, want nil — an empty-but-complete response is legal", got)
		}
		if !report.Terminated() {
			t.Error("report.Terminated() = false, want true")
		}
	})

	t.Run("success is readable from the terminal event's presence and kind alone (S-ARP-031)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		start, err := ai.NewResponseStart("resp_13", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		done, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		events := []ai.Event{s.Stamp(start), s.Stamp(done)}

		// The success verdict below reads only report.Violation() and
		// report.Terminated() — never events, content or usage.
		report := ai.CheckStream(events)
		succeeded := report.Violation() == nil && report.Terminated()
		if !succeeded {
			t.Error("succeeded = false, want true, reading only Violation() and Terminated()")
		}
	})

	t.Run("distinguishable from a stream with no terminal event at all (S-ARP-032)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		start, err := ai.NewResponseStart("resp_14", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}

		// A stream that never received a completion — truncated, not empty.
		truncated := ai.CheckStream([]ai.Event{s.Stamp(start)})
		if truncated.Terminated() {
			t.Error("truncated.Terminated() = true, want false — no completion was ever recorded")
		}
		if got := truncated.Violation(); got != nil {
			t.Errorf("truncated.Violation() = %v, want nil — an absent terminal is informational (Terminated()), never an ordering violation", got)
		}

		done, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		var s2 ai.Stamper
		start2, err := ai.NewResponseStart("resp_15", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}
		complete := ai.CheckStream([]ai.Event{s2.Stamp(start2), s2.Stamp(done)})

		if truncated.Terminated() == complete.Terminated() {
			t.Error("truncated and complete report the same Terminated() value, want them distinguishable")
		}
	})

	t.Run("a refusal and a normal stop remain distinguishable legal completions (S-ARP-033)", func(t *testing.T) {
		t.Parallel()

		refusal, err := ai.NewCompletion(ai.FinishReasonRefusal, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		normal, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}

		refusalCompletion, _ := refusal.Completion()
		normalCompletion, _ := normal.Completion()

		if refusalCompletion.FinishReason() == normalCompletion.FinishReason() {
			t.Fatal("refusal and normal-stop report the same finish reason, want them distinct")
		}
		for _, fr := range []ai.FinishReason{refusalCompletion.FinishReason(), normalCompletion.FinishReason()} {
			if fr == ai.FinishReasonUnknown {
				t.Errorf("finish reason %v equals FinishReasonUnknown, want both legal completions distinct from unknown", fr)
			}
		}
	})
}

// NFR-ARP-D / V-FAIL-13 (task 3.3) — the completion never leaks its finish
// reason or usage through a diagnostic rendering. Mirrors
// response_start_test.go's TestResponseStart_StringAndGoString_NameTheTypeAndCarryNoField.
func TestCompletion_StringAndGoString_NameTheTypeAndCarryNoField(t *testing.T) {
	t.Parallel()

	event, err := ai.NewCompletion(ai.FinishReasonContentFilter, ai.Usage{Output: ai.Tokens(999999)})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	c, ok := event.Completion()
	if !ok {
		t.Fatal("event.Completion() reported no payload on an event of its own kind")
	}

	for _, verb := range []string{"%v", "%s", "%+v"} {
		rendered := fmt.Sprintf(verb, c)
		if rendered != "completion" {
			t.Errorf("fmt.Sprintf(%q, c) = %q, want %q", verb, rendered, "completion")
		}
	}
	if got := fmt.Sprintf("%#v", c); got != "completion" {
		t.Errorf(`fmt.Sprintf("%%#v", c) = %q, want %q`, got, "completion")
	}
	if got := c.GoString(); got != c.String() {
		t.Errorf("c.GoString() = %q, want it to equal c.String() = %q", got, c.String())
	}
	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		rendered := fmt.Sprintf(verb, c)
		if strings.Contains(rendered, "999999") || strings.Contains(rendered, "content_filter") {
			t.Errorf("fmt.Sprintf(%q, c) = %q, which reproduces a field value", verb, rendered)
		}
	}
}

// NFR-ARP-B, S-ARP-039 (task 5.1) — totality: no exported entry point of
// either AI-15 kind panics for extreme inputs, mirroring event_test.go's
// TestEvent_ExtremeInputs_NeverPanics and message_test.go's
// TestMessage_ExtremeInputs_NeverPanics shape: each case recovers its own
// panic, so one panicking case does not hide the rest.
func TestResponseEvents_ExtremeInputs_NeverPanic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		act  func()
	}{
		{"the zero ResponseStart, read every way", func() {
			var zero ai.ResponseStart
			_ = zero.ResponseID()
			_ = zero.ServedModel()
			_ = zero.String()
			_ = zero.GoString()
		}},
		{"NewResponseStart with both fields empty", func() {
			_, _ = ai.NewResponseStart("", "")
		}},
		{"the zero Completion, read every way", func() {
			var zero ai.Completion
			_ = zero.FinishReason()
			_ = zero.Usage()
			_ = zero.String()
			_ = zero.GoString()
		}},
		{"NewCompletion with the zero-value (invalid) finish reason and a fully absent usage", func() {
			var zero ai.FinishReason
			_, _ = ai.NewCompletion(zero, ai.Usage{})
		}},
		{"NewCompletion with a negative token count in every usage field", func() {
			usage := ai.Usage{
				Input: ai.Tokens(-1), Output: ai.Tokens(-1), CacheRead: ai.Tokens(-1),
				CacheWrite: ai.Tokens(-1), Reasoning: ai.Tokens(-1),
			}
			_, _ = ai.NewCompletion(ai.FinishReasonStop, usage)
		}},
		{"a wrong-kind accessor on the zero Event", func() {
			_, _ = ai.Event{}.ResponseStart()
			_, _ = ai.Event{}.Completion()
		}},
		{"CheckEmit on the zero (unconstructed) Event", func() {
			_ = ai.CheckEmit(ai.Event{})
		}},
		{"CheckStream over a stream mixing both new kinds, unstamped", func() {
			start, _ := ai.NewResponseStart("", "")   // deliberately invalid/zero on failure
			done, _ := ai.NewCompletion(ai.FinishReason(0), ai.Usage{})
			report := ai.CheckStream([]ai.Event{start, done})
			_ = report.Violation()
			_ = report.Terminated()
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("panicked: %v", recovered)
				}
			}()
			tc.act()
		})
	}
}
