// Tests for AI-14.2 — the per-stream sequence.
package ai_test

import (
	"errors"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// R-AEE-007 — the sequence is 1-based and contiguous within one stream.
func TestStamper_Stamp_IsOneBasedAndContiguous(t *testing.T) {
	t.Parallel()

	t.Run("N events stamped in order read 1..N with no gap or repeat", func(t *testing.T) {
		t.Parallel()

		var stamper ai.Stamper
		const n = 5
		for i := 1; i <= n; i++ {
			stamped := stamper.Stamp(ai.NewWitnessEvent(1))
			if got, want := stamped.Sequence(), ai.Sequence(i); got != want {
				t.Fatalf("event %d: stamped.Sequence() = %v, want %v", i, got, want)
			}
		}
	})

	t.Run("Stamper holds no exported field: no way to supply a sequence from outside", func(t *testing.T) {
		t.Parallel()

		typ := reflect.TypeOf(ai.Stamper{})
		for i := 0; i < typ.NumField(); i++ {
			if field := typ.Field(i); field.IsExported() {
				t.Errorf("ai.Stamper field %q is exported; a caller could supply a sequence directly", field.Name)
			}
		}
	})

	t.Run("using the stamper again after N events continues at N+1: no reachable reset", func(t *testing.T) {
		t.Parallel()

		var stamper ai.Stamper
		for range 3 {
			stamper.Stamp(ai.NewWitnessEvent(1))
		}
		if got, want := stamper.Stamp(ai.NewWitnessEvent(1)).Sequence(), ai.Sequence(4); got != want {
			t.Errorf("the 4th stamp = %v, want %v (no gap, no reset)", got, want)
		}
	})
}

// R-AEE-008 — sequence state belongs to the stream, not the process.
func TestStamper_SequenceState_IsPerStreamNotProcess(t *testing.T) {
	t.Parallel()

	t.Run("two streams stamped concurrently each start at 1 and stay independently contiguous", func(t *testing.T) {
		t.Parallel()

		const perStream = 50
		firstResults := make([]ai.Sequence, perStream)
		secondResults := make([]ai.Sequence, perStream)

		done := make(chan struct{}, 2)
		go func() {
			var s ai.Stamper
			for i := range perStream {
				firstResults[i] = s.Stamp(ai.NewWitnessEvent(1)).Sequence()
			}
			done <- struct{}{}
		}()
		go func() {
			var s ai.Stamper
			for i := range perStream {
				secondResults[i] = s.Stamp(ai.NewWitnessEvent(1)).Sequence()
			}
			done <- struct{}{}
		}()
		<-done
		<-done

		for i := range perStream {
			if want := ai.Sequence(i + 1); firstResults[i] != want {
				t.Errorf("first stream event %d: sequence = %v, want %v", i, firstResults[i], want)
			}
			if want := ai.Sequence(i + 1); secondResults[i] != want {
				t.Errorf("second stream event %d: sequence = %v, want %v", i, secondResults[i], want)
			}
		}
	})

	t.Run("a third stream started after two others finish also begins at 1", func(t *testing.T) {
		t.Parallel()

		var first ai.Stamper
		for range 10 {
			first.Stamp(ai.NewWitnessEvent(1))
		}
		var second ai.Stamper
		for range 10 {
			second.Stamp(ai.NewWitnessEvent(1))
		}

		var third ai.Stamper
		if got, want := third.Stamp(ai.NewWitnessEvent(1)).Sequence(), ai.Sequence(1); got != want {
			t.Errorf("third stream's first stamp = %v, want %v — no residual process state to inherit", got, want)
		}
	})
}

// R-AEE-009 — cross-stream comparison is permitted and meaningless, and the
// contract says so.
func TestSequence_CrossStreamComparison_IsPermittedAndMeaningless(t *testing.T) {
	t.Parallel()

	var streamA, streamB ai.Stamper
	a := streamA.Stamp(ai.NewWitnessEvent(1)).Sequence()
	b := streamB.Stamp(ai.NewWitnessEvent(1)).Sequence()

	// Permitted by the type system: comparing two Sequence values from two
	// independent streams compiles and returns a bool, violating nothing.
	if a != b {
		t.Errorf("streamA's first sequence %v != streamB's first sequence %v, want both streams to independently start at 1", a, b)
	}
}

// R-AEE-009 (doc half) — the package documentation states the cross-stream
// rule explicitly, so a consumer who merges streams by sequence contradicts a
// written rule rather than an unwritten assumption.
//
// This mirrors content_part_registry_test.go's documentedPartKindNames
// idiom: read what a reader would read, with go/doc-adjacent source parsing,
// rather than trust that the prose exists.
func TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sequence.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing sequence.go: %v", err)
	}
	if file.Doc == nil {
		t.Fatal("sequence.go carries no package doc comment; the guard would pass vacuously")
	}
	// GoDoc line-wraps prose at the source's own line breaks, which are a
	// formatting artifact rather than a semantic one — collapse them before
	// matching, so a rewrap that moves "no" and "meaning" onto different
	// source lines does not make an honest doc comment fail this guard.
	doc := strings.Join(strings.Fields(file.Doc.Text()), " ")

	if !strings.Contains(doc, "overlap") {
		t.Errorf("sequence.go's package doc does not state that streams' sequences overlap:\n%s", doc)
	}
	if !strings.Contains(doc, "no meaning") && !strings.Contains(doc, "carries no meaning") {
		t.Errorf("sequence.go's package doc does not state that cross-stream ordering carries no meaning:\n%s", doc)
	}
}

// R-AEE-010 — the never-stamped sequence has a documented sentinel value,
// rejected at a stated boundary.
func TestCheckEmit_UnstampedSentinel_RejectedWithErrOutOfRange(t *testing.T) {
	t.Parallel()

	t.Run("an event that was never stamped is rejected, and the position names the sequence", func(t *testing.T) {
		t.Parallel()

		neverStamped := ai.NewWitnessEvent(1)
		if got := neverStamped.Sequence(); got != 0 {
			t.Fatalf("ai.NewWitnessEvent's own sequence = %v before stamping, want the sentinel 0", got)
		}

		err := ai.CheckEmit(neverStamped)
		if err == nil {
			t.Fatal("ai.CheckEmit(neverStamped) = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("ai.CheckEmit(neverStamped) = %v, want errors.Is to match ai.ErrOutOfRange", err)
		}
	})

	t.Run("a stamped event carrying sequence 1 is accepted at that same boundary", func(t *testing.T) {
		t.Parallel()

		var stamper ai.Stamper
		stamped := stamper.Stamp(ai.NewWitnessEvent(1))
		if err := ai.CheckEmit(stamped); err != nil {
			t.Errorf("ai.CheckEmit(stamped) = %v, want no failure — the rejection rule must not reject the first legal event of a stream", err)
		}
	})
}
