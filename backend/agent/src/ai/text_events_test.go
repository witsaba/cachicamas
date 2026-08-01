// Tests for AI-16 — the text-block event family: start, delta, end.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion is written against exactly the
// surface an adapter in another package sees — the convention every Layer 1
// test file in this package follows (event_test.go, response_start_test.go,
// completion_test.go).
package ai_test

import (
	"errors"
	"reflect"
	"sort"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// R-ATE-001 — the family is three separately registered event kinds, each
// with its own constructible payload, the kind derived from the payload with
// no internal phase discriminator. S-ATE-003's exhaustiveness-guard half (the
// guard covers all three, and a scratch unregistered kind still fails it by
// name) is proven generically by event_registry_test.go's eventKindWitnesses
// table, extended for these three kinds in the same commit — not repeated
// here.
func TestTextEvents_ThreeKinds_DistinctRegisteredAndDerivedFromThePayload(t *testing.T) {
	t.Parallel()

	kinds := []struct {
		name  string
		kind  ai.EventKind
		build func() (ai.Event, error)
	}{
		{"text block start", ai.EventKindTextBlockStart, func() (ai.Event, error) { return ai.NewTextBlockStart(1) }},
		{"text delta", ai.EventKindTextDelta, func() (ai.Event, error) { return ai.NewTextDelta(1, "fragment") }},
		{"text block end", ai.EventKindTextBlockEnd, func() (ai.Event, error) { return ai.NewTextBlockEnd(1) }},
	}

	t.Run("each kind compiles, is non-zero, and is a member of the registered vocabulary (S-ATE-001)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range kinds {
			if tc.kind == 0 {
				t.Errorf("%s: kind is the zero kind, want a registered non-zero member", tc.name)
			}
			found := false
			for _, k := range ai.EventKinds() {
				if k == tc.kind {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: ai.EventKinds() = %v, want it to contain %v", tc.name, ai.EventKinds(), tc.kind)
			}
		}
	})

	t.Run("the three kinds are pairwise distinct, and distinct from every other registered kind (S-ATE-001)", func(t *testing.T) {
		t.Parallel()

		seen := map[ai.EventKind]string{
			ai.EventKindResponseStart: "response start",
			ai.EventKindCompletion:    "completion",
		}
		for _, tc := range kinds {
			if name, dup := seen[tc.kind]; dup {
				t.Fatalf("%s shares its kind with %s, want every registered kind distinct", tc.name, name)
			}
			seen[tc.kind] = tc.name
		}
	})

	t.Run("the kind is derived from the payload and matches, with no separate phase field (S-ATE-002)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range kinds {
			event, err := tc.build()
			if err != nil {
				t.Fatalf("%s: build() returned %v, want no failure", tc.name, err)
			}
			if got := event.Kind(); got != tc.kind {
				t.Errorf("%s: event.Kind() = %v, want %v", tc.name, got, tc.kind)
			}
		}

		// No separate lifecycle-phase field on any of the three: each carries
		// exactly its block index (TextDelta additionally its fragment), and
		// the kind asked of the payload is the only discriminator.
		if got := reflect.TypeOf(ai.TextBlockStart{}).NumField(); got != 1 {
			t.Errorf("ai.TextBlockStart{} has %d fields, want exactly 1 (block index), no separate phase field", got)
		}
		if got := reflect.TypeOf(ai.TextDelta{}).NumField(); got != 2 {
			t.Errorf("ai.TextDelta{} has %d fields, want exactly 2 (block index, fragment), no separate phase field", got)
		}
		if got := reflect.TypeOf(ai.TextBlockEnd{}).NumField(); got != 1 {
			t.Errorf("ai.TextBlockEnd{} has %d fields, want exactly 1 (block index), no separate phase field", got)
		}
	})
}

// R-ATE-002 — every text block start, delta and end carries a
// producer-stamped block index, readable from an external package without
// parsing text.
func TestTextEvents_BlockIndex_StampedReadableAndOrderIndependent(t *testing.T) {
	t.Parallel()

	t.Run("start, two deltas and end for one block all report the same index (S-ATE-004)", func(t *testing.T) {
		t.Parallel()

		const block = ai.BlockIndex(3)
		start := mustTextBlockStart(t, block)
		first := mustTextDelta(t, block, "a")
		second := mustTextDelta(t, block, "b")
		end := mustTextBlockEnd(t, block)

		s, _ := start.TextBlockStart()
		d1, _ := first.TextDelta()
		d2, _ := second.TextDelta()
		e, _ := end.TextBlockEnd()

		for name, got := range map[string]ai.BlockIndex{
			"start": s.Block(), "delta 1": d1.Block(), "delta 2": d2.Block(), "end": e.Block(),
		} {
			if got != block {
				t.Errorf("%s.Block() = %v, want %v", name, got, block)
			}
		}
	})

	t.Run("the value read is the value stamped, unaffected by re-read order (S-ATE-005)", func(t *testing.T) {
		t.Parallel()

		const block = ai.BlockIndex(7)
		event, err := ai.NewTextBlockStart(block)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
		}

		first, ok := event.TextBlockStart()
		if !ok {
			t.Fatal("event.TextBlockStart() reported no payload on an event of its own kind")
		}
		second, ok := event.TextBlockStart()
		if !ok {
			t.Fatal("event.TextBlockStart() reported no payload on an event of its own kind")
		}
		if first.Block() != block || second.Block() != block {
			t.Errorf("re-reading the event reported %v then %v, want %v both times", first.Block(), second.Block(), block)
		}
	})
}

// R-ATE-003 — the block index is 1-based; 0 is rejected at construction with
// AI-04's out-of-range sentinel, naming the block-index field.
func TestTextEvents_ZeroBlockIndex_RejectedAtConstructionWithErrOutOfRange(t *testing.T) {
	t.Parallel()

	constructors := []struct {
		name  string
		build func(ai.BlockIndex) error
	}{
		{"text block start", func(b ai.BlockIndex) error { _, err := ai.NewTextBlockStart(b); return err }},
		{"text delta", func(b ai.BlockIndex) error { _, err := ai.NewTextDelta(b, "fragment"); return err }},
		{"text block end", func(b ai.BlockIndex) error { _, err := ai.NewTextBlockEnd(b); return err }},
	}

	t.Run("block index 0 fails with ErrOutOfRange at block (S-ATE-006)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range constructors {
			err := tc.build(0)
			if err == nil {
				t.Fatalf("%s: constructing with block index 0 = nil, want a rejection", tc.name)
			}
			if !errors.Is(err, ai.ErrOutOfRange) {
				t.Errorf("%s: errors.Is(err, ai.ErrOutOfRange) = false, want true; err = %v", tc.name, err)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("%s: errors.As(err, &violation) = false, want true; err = %v", tc.name, err)
			}
			if got, want := violation.Path().String(), "block"; got != want {
				t.Errorf("%s: violation position = %q, want %q", tc.name, got, want)
			}
		}
	})

	t.Run("a stream's first block carries index 1, not 0 (S-ATE-007)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewTextBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart(1) returned %v, want no failure", err)
		}
		start, ok := event.TextBlockStart()
		if !ok {
			t.Fatal("event.TextBlockStart() reported no payload on an event of its own kind")
		}
		if got := start.Block(); got != 1 {
			t.Errorf("start.Block() = %v, want 1", got)
		}
	})

	t.Run("a zero-value payload never constructs, so a caller that ignored the error cannot observe an event carrying block index 0 (S-ATE-008)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range constructors {
			if err := tc.build(0); err == nil {
				t.Fatalf("%s: zero-value block index construction = nil, want a rejection", tc.name)
			}
		}
	})
}

// R-ATE-004 — one block-index space per stream, shared across content
// families: a text block's index never collides with another family's, and
// the index alone disambiguates, with no family tag.
func TestTextEvents_SharedBlockIndexSpace_DisambiguatesByIndexAlone(t *testing.T) {
	t.Run("two interleaved text blocks are partitioned correctly by index alone, arrival order notwithstanding (S-ATE-009)", func(t *testing.T) {
		t.Parallel()

		// Arrival order deliberately interleaves block 1 and block 2, and
		// block 2's start arrives before block 1's first delta.
		block2Start := mustTextBlockStart(t, 2)
		block1Start := mustTextBlockStart(t, 1)
		block1Delta := mustTextDelta(t, 1, "one-")
		block2Delta := mustTextDelta(t, 2, "two-")
		block2End := mustTextBlockEnd(t, 2)
		block1End := mustTextBlockEnd(t, 1)

		stream := []ai.Event{block2Start, block1Start, block1Delta, block2Delta, block2End, block1End}

		partitioned := make(map[ai.BlockIndex]int)
		for _, e := range stream {
			partitioned[blockIndexOfTextEvent(t, e)]++
		}
		if got := partitioned[1]; got != 3 {
			t.Errorf("block 1 received %d events, want 3 (start, delta, end)", got)
		}
		if got := partitioned[2]; got != 3 {
			t.Errorf("block 2 received %d events, want 3 (start, delta, end)", got)
		}
	})

	// No t.Parallel() on this subtest: it registers a test kind via
	// ai.RegisterTestKind, which export_test.go's file comment forbids
	// running in parallel with another test holding a registration
	// (eventRegistry is one shared package-level slice).
	t.Run("a text block and a non-text block open at once carry different indices, separable without consulting kind (S-ATE-010)", func(t *testing.T) {
		k, cleanup := ai.RegisterTestKind("shared_index_space_non_text", ai.EventDescriptor{Role: ai.BlockRoleStart})
		t.Cleanup(cleanup)

		textEvent := mustTextBlockStart(t, 1)
		nonTextEvent := ai.NewTestEvent(k, 2)

		textIdx := blockIndexOfTextEvent(t, textEvent)
		nonTextPayload, ok := nonTextEvent.WitnessPayload()
		if !ok {
			t.Fatal("nonTextEvent.WitnessPayload() reported no payload on an event of its own kind")
		}
		nonTextIdx := nonTextPayload.BlockIndex()

		if textIdx == nonTextIdx {
			t.Fatalf("text block index %v equals the non-text block index %v, want them distinguishable", textIdx, nonTextIdx)
		}
		// Separated by index alone: neither side's kind was consulted above.
	})

	t.Run("neither struct declares a family tag or a second field required to disambiguate an index (S-ATE-011)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name string
			typ  reflect.Type
			want []string
		}{
			{"TextBlockStart", reflect.TypeOf(ai.TextBlockStart{}), []string{"block"}},
			{"TextDelta", reflect.TypeOf(ai.TextDelta{}), []string{"block", "delta"}},
			{"TextBlockEnd", reflect.TypeOf(ai.TextBlockEnd{}), []string{"block"}},
		} {
			var fields []string
			for i := 0; i < tc.typ.NumField(); i++ {
				fields = append(fields, tc.typ.Field(i).Name)
			}
			if !reflect.DeepEqual(fields, tc.want) {
				t.Errorf("ai.%s's fields = %v, want exactly %v — no family tag or second disambiguating field", tc.name, fields, tc.want)
			}
		}
	})
}

// blockIndexOfTextEvent reads e's block index through whichever of the three
// text-block accessors matches, for tests that partition a stream by index
// alone (S-ATE-009) — a small test-local helper reading one event's index, not
// a production reconstructor (R-ATE-011 is about accumulating a block's
// *content*, not reading its index).
func blockIndexOfTextEvent(t *testing.T, e ai.Event) ai.BlockIndex {
	t.Helper()
	if s, ok := e.TextBlockStart(); ok {
		return s.Block()
	}
	if d, ok := e.TextDelta(); ok {
		return d.Block()
	}
	if b, ok := e.TextBlockEnd(); ok {
		return b.Block()
	}
	t.Fatalf("event %v carries no text-block payload", e)
	return 0
}

// R-ATE-005 — a delta carries only the new fragment, never accumulated
// content; no accessor on any of the three kinds returns accumulated block
// content.
func TestTextDelta_CarriesOnlyTheFragment_NeverAccumulatedContent(t *testing.T) {
	t.Parallel()

	t.Run("each delta's fragment is read back as-is, never accumulated (S-ATE-012)", func(t *testing.T) {
		t.Parallel()

		fragments := []string{"a", "b", "c"}
		var got []string
		for _, f := range fragments {
			event, err := ai.NewTextDelta(1, f)
			if err != nil {
				t.Fatalf("ai.NewTextDelta(1, %q) returned %v, want no failure", f, err)
			}
			d, ok := event.TextDelta()
			if !ok {
				t.Fatal("event.TextDelta() reported no payload on an event of its own kind")
			}
			got = append(got, d.Delta())
		}
		if !reflect.DeepEqual(got, fragments) {
			t.Errorf("read fragments = %v, want exactly %v — never \"a\", \"ab\", \"abc\"", got, fragments)
		}
	})

	t.Run("no exported accessor of any of the three kinds returns accumulated or snapshot content (S-ATE-013)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name string
			typ  reflect.Type
			want []string
		}{
			{"TextBlockStart", reflect.TypeOf(ai.TextBlockStart{}), []string{"Block"}},
			{"TextDelta", reflect.TypeOf(ai.TextDelta{}), []string{"Block", "Delta"}},
			{"TextBlockEnd", reflect.TypeOf(ai.TextBlockEnd{}), []string{"Block"}},
		} {
			var accessors []string
			for i := 0; i < tc.typ.NumMethod(); i++ {
				name := tc.typ.Method(i).Name
				if name == "String" || name == "GoString" {
					continue // diagnostic rendering, not a field accessor (V-FAIL-13)
				}
				accessors = append(accessors, name)
			}
			sort.Strings(accessors)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			if !reflect.DeepEqual(accessors, want) {
				t.Errorf("ai.%s's exported accessors = %v, want exactly %v — no accumulator or snapshot accessor", tc.name, accessors, want)
			}
		}
	})
}

// mustTextBlockStart, mustTextDelta and mustTextBlockEnd construct a valid
// event of their kind or fail the test — response_start_test.go's
// mustResponseStart shape, restated for the three text-event kinds so later
// phases (byte fidelity, zero-delta) build fixtures without repeating the
// same three-line error check at every call site (AI-16.1 REFACTOR).
func mustTextBlockStart(t *testing.T, block ai.BlockIndex) ai.Event {
	t.Helper()
	event, err := ai.NewTextBlockStart(block)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart(%v) returned %v, want no failure", block, err)
	}
	return event
}

func mustTextDelta(t *testing.T, block ai.BlockIndex, delta string) ai.Event {
	t.Helper()
	event, err := ai.NewTextDelta(block, delta)
	if err != nil {
		t.Fatalf("ai.NewTextDelta(%v, %q) returned %v, want no failure", block, delta, err)
	}
	return event
}

func mustTextBlockEnd(t *testing.T, block ai.BlockIndex) ai.Event {
	t.Helper()
	event, err := ai.NewTextBlockEnd(block)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd(%v) returned %v, want no failure", block, err)
	}
	return event
}
