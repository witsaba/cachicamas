// Tests for AI-14.4 — the ordering-invariant checker (R-AEE-015 … R-AEE-019).
//
// Most tests in this file call ai.RegisterTestKind to give a stream a kind
// with a chosen descriptor, and therefore do NOT call t.Parallel() —
// export_test.go's own file comment states why: eventRegistry is one shared
// package-level slice, and a concurrent truncation from another parallel
// test would corrupt it. Only tests that touch no registry state at all
// (a pure AST scan, or a nil/empty-slice call) are marked parallel.
package ai_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// streamKind registers a fresh test-only kind with descriptor d and a
// t.Cleanup that unregisters it, and is this file's one entry point for
// getting a kind with a chosen shape onto a stream. Every caller inherits
// the "no t.Parallel" constraint above.
func streamKind(t *testing.T, name string, d ai.EventDescriptor) ai.EventKind {
	t.Helper()
	k, cleanup := ai.RegisterTestKind(name, d)
	t.Cleanup(cleanup)
	return k
}

// R-AEE-015, S-AEE-046 — the checker's own source names no concrete payload
// type and no individual kind constant. A non-test file cannot reference a
// _test.go-only identifier at all — WitnessPayload and KindTestWitness would
// not compile into stream_check.go — and AI-14 ships zero production kind
// constants (R-AEE-006), so this also confirms nothing production-specific
// was smuggled in. No registry interaction: safe to parallelize.
func TestCheckStream_Source_NamesNoConcretePayloadTypeOrKindConstant(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "stream_check.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing stream_check.go: %v", err)
	}

	forbidden := []string{"WitnessPayload", "KindTestWitness"}
	ast.Inspect(file, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		for _, name := range forbidden {
			if ident.Name == name {
				t.Errorf("stream_check.go references %q, a test-only identifier; the checker must read only "+
					"(kind, descriptor, block index, sequence), never a concrete payload type", name)
			}
		}
		return true
	})
}

// R-AEE-015, S-AEE-047 — a second, independently registered test-only
// witness kind is constrained correctly, with no edit to the checker: both
// kinds below are registered side by side, each carrying a rule the other
// does not, and each is enforced on its own stream with no interference.
func TestCheckStream_TwoIndependentlyRegisteredKinds_EachConstrainedCorrectly(t *testing.T) {
	onceKind := streamKind(t, "once_kind", ai.EventDescriptor{Cardinality: ai.CardinalityAtMostOne})
	terminalKind := streamKind(t, "terminal_kind", ai.EventDescriptor{Terminal: true})

	t.Run("the at-most-one kind is enforced", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(onceKind, 0)),
			s.Stamp(ai.NewTestEvent(onceKind, 0)),
		}
		report := ai.CheckStream(events)
		if !errors.Is(report.Violation(), ai.ErrDuplicate) {
			t.Errorf("report.Violation() = %v, want errors.Is to match ErrDuplicate", report.Violation())
		}
	})

	t.Run("the terminal kind is enforced, with the at-most-one kind also registered", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(terminalKind, 0)),
			s.Stamp(ai.NewTestEvent(onceKind, 0)), // one event after the terminal one
		}
		report := ai.CheckStream(events)
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("report.Violation() = %v, want errors.Is to match ErrMisplaced (an event after the terminal one)", report.Violation())
		}
		if !report.Terminated() {
			t.Error("report.Terminated() = false, want true")
		}
	})
}

// R-AEE-016 — block ordering: start ≺ deltas ≺ end, per block index.
func TestCheckStream_BlockOrdering(t *testing.T) {
	start := streamKind(t, "block_start", ai.EventDescriptor{Role: ai.BlockRoleStart})
	delta := streamKind(t, "block_delta", ai.EventDescriptor{Role: ai.BlockRoleDelta})
	end := streamKind(t, "block_end", ai.EventDescriptor{Role: ai.BlockRoleEnd})

	// S-AEE-049 — one block's start, two deltas and end in order: no violation.
	t.Run("start, two deltas, end: no violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(start, 1)),
			s.Stamp(ai.NewTestEvent(delta, 1)),
			s.Stamp(ai.NewTestEvent(delta, 1)),
			s.Stamp(ai.NewTestEvent(end, 1)),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})

	// S-AEE-050 — a delta for block 2 before block 2's start: a violation
	// naming index 2 and the offending sequence.
	t.Run("delta before its block's start: violation naming the block and the offending sequence", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(delta, 2)),
		}
		report := ai.CheckStream(events)
		err := report.Violation()
		if !errors.Is(err, ai.ErrMisplaced) {
			t.Fatalf("report.Violation() = %v, want errors.Is to match ErrMisplaced", err)
		}
		if got := err.Error(); !strings.Contains(got, "block[2]") || !strings.Contains(got, "event[1]") {
			t.Errorf("report.Violation().Error() = %q, want it to name block[2] and event[1]", got)
		}
	})

	// S-AEE-051 — two blocks whose events interleave: no violation.
	t.Run("two blocks interleaved: no violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(start, 1)),
			s.Stamp(ai.NewTestEvent(start, 2)),
			s.Stamp(ai.NewTestEvent(delta, 1)),
			s.Stamp(ai.NewTestEvent(delta, 2)),
			s.Stamp(ai.NewTestEvent(end, 1)),
			s.Stamp(ai.NewTestEvent(end, 2)),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})

	// S-AEE-052 — block 1's start with no matching end: an unterminated-block
	// report, distinguishable from an ordering violation (a different
	// sentinel: ErrMalformed, not ErrMisplaced/ErrDuplicate).
	t.Run("start with no matching end: unterminated, distinguishable from an ordering violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(start, 1)),
		}
		report := ai.CheckStream(events)
		err := report.Violation()
		if !errors.Is(err, ai.ErrMalformed) {
			t.Fatalf("report.Violation() = %v, want errors.Is to match ErrMalformed", err)
		}
		if errors.Is(err, ai.ErrMisplaced) || errors.Is(err, ai.ErrDuplicate) {
			t.Errorf("report.Violation() = %v, also matches an ordering sentinel; want it distinguishable", err)
		}
	})

	// S-AEE-053 — a start immediately followed by its end with zero deltas:
	// no violation (V-STR-16 permits a delta-free block).
	t.Run("start immediately followed by end, zero deltas: no violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(start, 1)),
			s.Stamp(ai.NewTestEvent(end, 1)),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})

	// A second start for the same block index is also a violation
	// (R-AEE-016's "second start" case from design.md D7), naming ErrDuplicate.
	t.Run("a second start for the same block index: ErrDuplicate", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(start, 1)),
			s.Stamp(ai.NewTestEvent(start, 1)),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); !errors.Is(err, ai.ErrDuplicate) {
			t.Errorf("report.Violation() = %v, want errors.Is to match ErrDuplicate", err)
		}
	})
}

// R-AEE-017 — at-most-one is a reusable descriptor-driven primitive.
func TestCheckStream_AtMostOneCardinality(t *testing.T) {
	onceKind := streamKind(t, "at_most_once", ai.EventDescriptor{Cardinality: ai.CardinalityAtMostOne})
	anyKind := streamKind(t, "any_cardinality", ai.EventDescriptor{Cardinality: ai.CardinalityAny})

	// S-AEE-054 — an at-most-one test kind appearing twice: a violation
	// naming the second event's sequence.
	t.Run("appearing twice: violation naming the second event's sequence", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(onceKind, 0)),
			s.Stamp(ai.NewTestEvent(onceKind, 0)),
		}
		report := ai.CheckStream(events)
		err := report.Violation()
		if !errors.Is(err, ai.ErrDuplicate) {
			t.Fatalf("report.Violation() = %v, want errors.Is to match ErrDuplicate", err)
		}
		if got := err.Error(); !strings.Contains(got, "event[2]") {
			t.Errorf("report.Violation().Error() = %q, want it to name event[2] (the second event)", got)
		}
	})

	// S-AEE-055 — appearing once, or not at all: no cardinality violation.
	t.Run("appearing once: no violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{s.Stamp(ai.NewTestEvent(onceKind, 0))}
		if err := ai.CheckStream(events).Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})
	t.Run("not appearing at all: no violation", func(t *testing.T) {
		if err := ai.CheckStream(nil).Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})

	// S-AEE-056 — an any-cardinality kind appearing many times: no violation.
	t.Run("an any-cardinality kind appearing many times: no violation", func(t *testing.T) {
		var s ai.Stamper
		var events []ai.Event
		for range 5 {
			events = append(events, s.Stamp(ai.NewTestEvent(anyKind, 0)))
		}
		if err := ai.CheckStream(events).Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})
}

// R-AEE-018 — terminal is a reusable descriptor-driven primitive.
func TestCheckStream_Terminal(t *testing.T) {
	terminalKind := streamKind(t, "terminal", ai.EventDescriptor{Terminal: true})
	plainKind := streamKind(t, "plain", ai.EventDescriptor{})

	// S-AEE-057 — a terminal-kind event with one event after it: a violation
	// naming the following event's sequence.
	t.Run("one event after terminal: violation naming the following event's sequence", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(terminalKind, 0)),
			s.Stamp(ai.NewTestEvent(plainKind, 0)),
		}
		report := ai.CheckStream(events)
		err := report.Violation()
		if !errors.Is(err, ai.ErrMisplaced) {
			t.Fatalf("report.Violation() = %v, want errors.Is to match ErrMisplaced", err)
		}
		if got := err.Error(); !strings.Contains(got, "event[2]") {
			t.Errorf("report.Violation().Error() = %q, want it to name event[2] (the following event)", got)
		}
		if !report.Terminated() {
			t.Error("report.Terminated() = false, want true")
		}
	})

	// S-AEE-058 — two terminal-kind events: a violation naming the second.
	t.Run("two terminal events: violation naming the second", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(terminalKind, 0)),
			s.Stamp(ai.NewTestEvent(terminalKind, 0)),
		}
		report := ai.CheckStream(events)
		err := report.Violation()
		if !errors.Is(err, ai.ErrDuplicate) {
			t.Fatalf("report.Violation() = %v, want errors.Is to match ErrDuplicate", err)
		}
		if got := err.Error(); !strings.Contains(got, "event[2]") {
			t.Errorf("report.Violation().Error() = %q, want it to name event[2] (the second terminal event)", got)
		}
	})

	// S-AEE-059 — no terminal event at all: reported distinguishably and as
	// informational, not as an ordering violation — Terminated() == false,
	// Violation() == nil.
	t.Run("no terminal event: informational, not a violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(plainKind, 0)),
			s.Stamp(ai.NewTestEvent(plainKind, 0)),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil — absence of a terminal event is not itself a violation", err)
		}
		if report.Terminated() {
			t.Error("report.Terminated() = true, want false")
		}
	})

	// S-AEE-060 — a stream whose last event is its single terminal event: no
	// violation.
	t.Run("terminal event as the stream's last event: no violation", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(plainKind, 0)),
			s.Stamp(ai.NewTestEvent(terminalKind, 0)),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
		if !report.Terminated() {
			t.Error("report.Terminated() = false, want true")
		}
	})
}

// R-AEE-019 — runs against a recorded stream, reports through AI-04.
func TestCheckStream_FirstViolationInStreamOrder_NoMutation_Idempotent(t *testing.T) {
	onceKind := streamKind(t, "first_violation_once", ai.EventDescriptor{Cardinality: ai.CardinalityAtMostOne})
	terminalKind := streamKind(t, "first_violation_terminal", ai.EventDescriptor{Terminal: true})

	// S-AEE-061/062 — given two violations, the one with the lower sequence
	// (earlier in stream order) is reported, as AI-04's failure value.
	t.Run("the earlier violation wins", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(onceKind, 0)),     // 1: fine
			s.Stamp(ai.NewTestEvent(onceKind, 0)),     // 2: at-most-one violated here
			s.Stamp(ai.NewTestEvent(terminalKind, 0)), // 3: terminal, fine on its own
			s.Stamp(ai.NewTestEvent(onceKind, 0)),     // 4: also after terminal
		}
		report := ai.CheckStream(events)
		err := report.Violation()
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("report.Violation() = %v (%T), want it to be (or wrap) *ai.Violation — AI-04's failure value, no second failure type", err, err)
		}
		if !errors.Is(err, ai.ErrDuplicate) {
			t.Fatalf("report.Violation() = %v, want errors.Is to match ErrDuplicate (the earlier, at-most-one violation at event 2)", err)
		}
		if got := err.Error(); !strings.Contains(got, "event[2]") {
			t.Errorf("report.Violation().Error() = %q, want it to name event[2] (the earlier violation), not event[4]", got)
		}
	})

	// S-AEE-063 — two runs over the same slice: identical verdicts and an
	// unchanged slice.
	t.Run("idempotent, and the input slice is not mutated", func(t *testing.T) {
		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(ai.NewTestEvent(onceKind, 0)),
			s.Stamp(ai.NewTestEvent(onceKind, 0)),
		}
		before := append([]ai.Event(nil), events...)

		first := ai.CheckStream(events)
		second := ai.CheckStream(events)

		if (first.Violation() == nil) != (second.Violation() == nil) {
			t.Fatalf("two runs disagree on whether there is a violation: first=%v second=%v", first.Violation(), second.Violation())
		}
		if first.Violation() != nil && first.Violation().Error() != second.Violation().Error() {
			t.Errorf("two runs report different violations: first=%q second=%q", first.Violation().Error(), second.Violation().Error())
		}
		if first.Terminated() != second.Terminated() {
			t.Errorf("two runs disagree on Terminated(): first=%v second=%v", first.Terminated(), second.Terminated())
		}
		for i := range events {
			if events[i].Kind() != before[i].Kind() || events[i].Sequence() != before[i].Sequence() {
				t.Errorf("events[%d] changed after CheckStream: before kind=%v seq=%v, after kind=%v seq=%v",
					i, before[i].Kind(), before[i].Sequence(), events[i].Kind(), events[i].Sequence())
			}
		}
	})
}

// NFR-AEE-B totality, exercised early for CheckStream specifically (the
// fuller totality table lands in Phase 6): a nil slice and an empty slice
// must not panic and must report no violation and no terminal event. No
// registry interaction: safe to parallelize.
func TestCheckStream_NilAndEmptySlice_SafeAndReportNoViolation(t *testing.T) {
	t.Parallel()

	for name, events := range map[string][]ai.Event{"nil slice": nil, "empty slice": {}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			report := ai.CheckStream(events)
			if err := report.Violation(); err != nil {
				t.Errorf("report.Violation() = %v, want nil", err)
			}
			if report.Terminated() {
				t.Error("report.Terminated() = true, want false")
			}
		})
	}
}
