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
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
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

// R-ATE-006 — concatenated deltas reconstruct a text block byte-exactly, with
// no normalization, trimming or re-encoding.
func TestTextDelta_ConcatenatedFragments_ReconstructByteExactly(t *testing.T) {
	t.Parallel()

	t.Run("an arbitrary split reassembles to the original text (S-ATE-014)", func(t *testing.T) {
		t.Parallel()

		const original = "The quick brown fox jumps over the lazy dog."
		fragments := []string{"The quick ", "brown fox ", "jumps over ", "the lazy dog."}

		var events []ai.Event
		for _, f := range fragments {
			events = append(events, mustTextDelta(t, 1, f))
		}
		if got := concatenateTextDeltas(events); got != original {
			t.Errorf("concatenateTextDeltas(events) = %q, want %q", got, original)
		}
	})

	t.Run("leading whitespace, trailing whitespace and interior newlines survive with no trimming (S-ATE-015)", func(t *testing.T) {
		t.Parallel()

		const original = "  leading and trailing  \nwith\ninterior\nnewlines\n  "
		fragments := []string{"  leading and trailing  \n", "with\ninterior\n", "newlines\n  "}

		var events []ai.Event
		for _, f := range fragments {
			events = append(events, mustTextDelta(t, 1, f))
		}
		if got := concatenateTextDeltas(events); got != original {
			t.Errorf("concatenateTextDeltas(events) = %q, want %q — no trimming of any whitespace byte", got, original)
		}
	})
}

// R-ATE-007 — a fragment is raw bytes, not validated text: an individual
// fragment may be invalid UTF-8 on its own, and construction never repairs,
// replaces or re-encodes a byte of it.
func TestTextDelta_MultiByteRuneSplitAcrossADeltaBoundary_PreservesEveryByte(t *testing.T) {
	t.Parallel()

	// "€" (U+20AC) encodes to the three bytes 0xE2 0x82 0xAC. Split after the
	// first byte, so the first fragment ends mid-rune and the second begins
	// mid-rune.
	const euro = "€"
	if len(euro) != 3 {
		t.Fatalf("test fixture error: %q must encode to 3 bytes, got %d", euro, len(euro))
	}
	first := euro[:1]
	second := euro[1:]

	t.Run("both deltas construct, and neither fragment's bytes are altered (S-ATE-016)", func(t *testing.T) {
		t.Parallel()

		firstEvent, err := ai.NewTextDelta(1, first)
		if err != nil {
			t.Fatalf("ai.NewTextDelta(1, %q) returned %v, want no failure", first, err)
		}
		secondEvent, err := ai.NewTextDelta(1, second)
		if err != nil {
			t.Fatalf("ai.NewTextDelta(1, %q) returned %v, want no failure", second, err)
		}

		d1, _ := firstEvent.TextDelta()
		d2, _ := secondEvent.TextDelta()
		if d1.Delta() != first {
			t.Errorf("d1.Delta() = %q (% x), want %q (% x) unaltered", d1.Delta(), d1.Delta(), first, first)
		}
		if d2.Delta() != second {
			t.Errorf("d2.Delta() = %q (% x), want %q (% x) unaltered", d2.Delta(), d2.Delta(), second, second)
		}
	})

	t.Run("concatenating the pair decodes to the original rune, with no replacement character (S-ATE-017)", func(t *testing.T) {
		t.Parallel()

		events := []ai.Event{mustTextDelta(t, 1, first), mustTextDelta(t, 1, second)}
		got := concatenateTextDeltas(events)
		if got != euro {
			t.Errorf("concatenateTextDeltas(events) = %q, want %q", got, euro)
		}
		if strings.ContainsRune(got, '�') {
			t.Errorf("concatenateTextDeltas(events) = %q contains the replacement character, want none", got)
		}
	})

	t.Run("the fragment's declaration uses the word byte and states a single fragment may not be well-formed UTF-8 (S-ATE-018)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "text_events.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing text_events.go: %v", err)
		}

		doc := fieldDoc(file, "TextDelta", "delta")
		if doc == "" {
			t.Fatal(`text_events.go's "TextDelta" struct declares no doc comment on its "delta" field`)
		}
		if !strings.Contains(strings.ToLower(doc), "byte") {
			t.Errorf("TextDelta.delta's doc comment = %q, want it to use the word \"byte\"", doc)
		}
		if !strings.Contains(strings.ToLower(doc), "utf-8") {
			t.Errorf("TextDelta.delta's doc comment = %q, want it to record that a fragment may not be well-formed UTF-8 alone", doc)
		}
	})
}

// R-ATE-008 — no text-content emptiness rule applies to a fragment: a
// whitespace-only or zero-length fragment is legal, and construction never
// returns ErrEmpty for either.
func TestTextDelta_WhitespaceOnlyAndZeroLengthFragments_AreLegal(t *testing.T) {
	t.Parallel()

	t.Run("a single-space fragment succeeds and reads back exactly one space (S-ATE-019)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewTextDelta(1, " ")
		if err != nil {
			t.Fatalf("ai.NewTextDelta(1, \" \") returned %v, want no failure", err)
		}
		d, ok := event.TextDelta()
		if !ok {
			t.Fatal("event.TextDelta() reported no payload on an event of its own kind")
		}
		if got := d.Delta(); got != " " {
			t.Errorf("d.Delta() = %q, want a single space", got)
		}
	})

	t.Run("a zero-length fragment succeeds with no emptiness sentinel (S-ATE-020)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewTextDelta(1, "")
		if err != nil {
			t.Fatalf("ai.NewTextDelta(1, \"\") returned %v, want no failure", err)
		}
		if errors.Is(err, ai.ErrEmpty) {
			t.Error("errors.Is(err, ai.ErrEmpty) = true, want false — a zero-length fragment is legal")
		}
		d, ok := event.TextDelta()
		if !ok {
			t.Fatal("event.TextDelta() reported no payload on an event of its own kind")
		}
		if got := d.Delta(); got != "" {
			t.Errorf("d.Delta() = %q, want empty", got)
		}
	})

	t.Run("a whitespace-only fragment among non-empty ones lands at its original position on reconstruction (S-ATE-021)", func(t *testing.T) {
		t.Parallel()

		events := []ai.Event{
			mustTextDelta(t, 1, "before"),
			mustTextDelta(t, 1, "   "),
			mustTextDelta(t, 1, "after"),
		}
		want := "before   after"
		if got := concatenateTextDeltas(events); got != want {
			t.Errorf("concatenateTextDeltas(events) = %q, want %q", got, want)
		}
	})
}

// R-ATE-009 — a fragment is bounded by the existing MaxTextLen ceiling: a
// fragment exceeding it is rejected at construction, and a fragment of
// exactly the bound succeeds.
func TestTextDelta_FragmentBoundByMaxTextLen(t *testing.T) {
	t.Parallel()

	t.Run("a fragment exceeding MaxTextLen fails with ErrOutOfRange at delta (S-ATE-022)", func(t *testing.T) {
		t.Parallel()

		oversized := strings.Repeat("a", ai.MaxTextLen+1)
		_, err := ai.NewTextDelta(1, oversized)
		if err == nil {
			t.Fatal("ai.NewTextDelta with an over-long fragment = nil, want a rejection")
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("errors.Is(err, ai.ErrOutOfRange) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "delta"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("a fragment of exactly MaxTextLen bytes succeeds (S-ATE-023)", func(t *testing.T) {
		t.Parallel()

		exact := strings.Repeat("a", ai.MaxTextLen)
		event, err := ai.NewTextDelta(1, exact)
		if err != nil {
			t.Fatalf("ai.NewTextDelta with a %d-byte fragment returned %v, want no failure", ai.MaxTextLen, err)
		}
		d, ok := event.TextDelta()
		if !ok {
			t.Fatal("event.TextDelta() reported no payload on an event of its own kind")
		}
		if got := len(d.Delta()); got != ai.MaxTextLen {
			t.Errorf("len(d.Delta()) = %d, want %d", got, ai.MaxTextLen)
		}
	})
}

// concatenateTextDeltas is a test-local reconstructor proving R-ATE-006's and
// R-ATE-010's byte-exactness. It is not exported: R-ATE-011 forbids Layer 1
// from shipping one, and this is this milestone's own proof that a consumer
// can write one in a few lines without Layer 1's help (S-ATE-028). An empty
// slice — the zero-delta case — returns the empty string with no error
// (R-ATE-010).
func concatenateTextDeltas(events []ai.Event) string {
	var b strings.Builder
	for _, e := range events {
		if d, ok := e.TextDelta(); ok {
			b.WriteString(d.Delta())
		}
	}
	return b.String()
}

// fieldDoc returns the doc comment text of a named field within a named
// struct type declared in file, or "" if either does not exist or the field
// has no doc comment.
func fieldDoc(file *ast.File, structName, fieldName string) string {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typed, ok := spec.(*ast.TypeSpec)
			if !ok || typed.Name.Name != structName {
				continue
			}
			st, ok := typed.Type.(*ast.StructType)
			if !ok {
				return ""
			}
			for _, f := range st.Fields.List {
				for _, name := range f.Names {
					if name.Name == fieldName && f.Doc != nil {
						return f.Doc.Text()
					}
				}
			}
		}
	}
	return ""
}

// R-ATE-010 — a text block with zero deltas is legal and reconstructs to
// empty; it is not confused with an unterminated block.
func TestTextBlock_ZeroDeltas_IsLegalAndReconstructsToEmpty(t *testing.T) {
	t.Parallel()

	t.Run("a start immediately followed by its end validates with no violation (S-ATE-024)", func(t *testing.T) {
		t.Parallel()

		start := mustTextBlockStart(t, 1)
		end := mustTextBlockEnd(t, 1)

		var s ai.Stamper
		report := ai.CheckStream([]ai.Event{s.Stamp(start), s.Stamp(end)})
		if got := report.Violation(); got != nil {
			t.Errorf("report.Violation() = %v, want nil — a zero-delta block is legal", got)
		}
	})

	t.Run("a test-local concatenator reconstructs it as empty, with no error from the empty join (S-ATE-025)", func(t *testing.T) {
		t.Parallel()

		if got := concatenateTextDeltas(nil); got != "" {
			t.Errorf("concatenateTextDeltas(nil) = %q, want empty", got)
		}
		// Also from a real (empty) delta slice extracted between a start and
		// an end carrying no delta in between.
		start := mustTextBlockStart(t, 1)
		end := mustTextBlockEnd(t, 1)
		if got := concatenateTextDeltas([]ai.Event{start, end}); got != "" {
			t.Errorf("concatenateTextDeltas([start, end]) = %q, want empty — neither event is a delta", got)
		}
	})

	t.Run("a zero-delta block and a multi-delta block both close normally, and the zero-delta block is not mistaken for unterminated (S-ATE-026)", func(t *testing.T) {
		t.Parallel()

		zeroDeltaStart := mustTextBlockStart(t, 1)
		zeroDeltaEnd := mustTextBlockEnd(t, 1)
		multiStart := mustTextBlockStart(t, 2)
		multiDelta := mustTextDelta(t, 2, "content")
		multiEnd := mustTextBlockEnd(t, 2)

		var s ai.Stamper
		events := []ai.Event{
			s.Stamp(zeroDeltaStart), s.Stamp(multiStart), s.Stamp(multiDelta),
			s.Stamp(zeroDeltaEnd), s.Stamp(multiEnd),
		}
		report := ai.CheckStream(events)
		if got := report.Violation(); got != nil {
			t.Errorf("report.Violation() = %v, want nil — neither block is unterminated", got)
		}
	})
}

// R-ATE-011 — Layer 1 ships no public accumulator, transcript rebuilder or
// reducer of a block's deltas; byte-exactness is proven by a test-local
// concatenator instead.
func TestTextEvents_ExportedSurface_ShipsNoAccumulatorOrReconstructor(t *testing.T) {
	t.Parallel()

	t.Run("the exported declarations of text_events.go are exactly the constructors, accessors and getters design.md documents (S-ATE-027)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "text_events.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing text_events.go: %v", err)
		}

		got := exportedTopLevelNames(file)
		sort.Strings(got)

		want := []string{
			// constructors
			"NewTextBlockEnd", "NewTextBlockStart", "NewTextDelta",
			// types
			"TextBlockEnd", "TextBlockStart", "TextDelta",
			// Event accessors
			"Event.TextBlockEnd", "Event.TextBlockStart", "Event.TextDelta",
			// field getters and diagnostic rendering
			"TextBlockEnd.Block", "TextBlockEnd.GoString", "TextBlockEnd.String",
			"TextBlockStart.Block", "TextBlockStart.GoString", "TextBlockStart.String",
			"TextDelta.Block", "TextDelta.Delta", "TextDelta.GoString", "TextDelta.String",
		}
		sort.Strings(want)

		if !reflect.DeepEqual(got, want) {
			t.Errorf("text_events.go's exported declarations = %v, want exactly %v — none of them accumulates, joins or reconstructs a block's deltas", got, want)
		}
	})

	t.Run("the concatenator used to prove byte-exactness is a package-level unexported func in the test package, not exported from the contract (S-ATE-028)", func(t *testing.T) {
		t.Parallel()

		// concatenateTextDeltas is declared in this ai_test package file, and
		// the preceding subtest's exact-surface assertion over text_events.go
		// structurally proves no exported counterpart exists there. This
		// assertion documents where it does live: an unexported func, only
		// reachable from within ai_test.
		if got := reflect.TypeOf(concatenateTextDeltas).Kind(); got != reflect.Func {
			t.Fatalf("concatenateTextDeltas is a %v, want a func", got)
		}
	})
}

// exportedTopLevelNames returns every exported top-level function, method and
// type declared in file, methods qualified as "Receiver.Method" — the
// complete exported surface a reviewer would enumerate to look for an
// accumulator (S-ATE-027).
func exportedTopLevelNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			if d.Recv == nil || len(d.Recv.List) == 0 {
				names = append(names, d.Name.Name)
				continue
			}
			names = append(names, receiverTypeName(d.Recv.List[0].Type)+"."+d.Name.Name)
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				if typed, ok := spec.(*ast.TypeSpec); ok && typed.Name.IsExported() {
					names = append(names, typed.Name.Name)
				}
			}
		}
	}
	return names
}

// receiverTypeName returns the bare type name of a method receiver
// expression, stripping a pointer star when present.
func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "?"
}

// NFR-ATE-B, S-ATE-030 — totality: no exported function or method of the
// three text-event kinds panics for any input, including the zero value of
// each payload, a zero block index, a zero-length fragment, a fragment that
// is invalid UTF-8 alone, and an over-long fragment. Mirrors
// completion_test.go's TestResponseEvents_ExtremeInputs_NeverPanic shape:
// each case recovers its own panic, so one panicking case does not hide the
// rest.
func TestTextEvents_ExtremeInputs_NeverPanic(t *testing.T) {
	t.Parallel()

	// An overlong two-byte encoding of NUL: invalid UTF-8 on its own, with no
	// dependency on where a legal rune boundary happens to fall.
	invalidUTF8Alone := string([]byte{0xC0, 0xAF})

	cases := []struct {
		name string
		act  func()
	}{
		{"the zero TextBlockStart, read every way", func() {
			var zero ai.TextBlockStart
			_ = zero.Block()
			_ = zero.String()
			_ = zero.GoString()
		}},
		{"NewTextBlockStart with block index 0", func() {
			_, _ = ai.NewTextBlockStart(0)
		}},
		{"the zero TextDelta, read every way", func() {
			var zero ai.TextDelta
			_ = zero.Block()
			_ = zero.Delta()
			_ = zero.String()
			_ = zero.GoString()
		}},
		{"NewTextDelta with block index 0 and a zero-length fragment", func() {
			_, _ = ai.NewTextDelta(0, "")
		}},
		{"NewTextDelta with a fragment that is invalid UTF-8 alone", func() {
			_, _ = ai.NewTextDelta(1, invalidUTF8Alone)
		}},
		{"NewTextDelta with a fragment one byte over MaxTextLen", func() {
			_, _ = ai.NewTextDelta(1, strings.Repeat("a", ai.MaxTextLen+1))
		}},
		{"the zero TextBlockEnd, read every way", func() {
			var zero ai.TextBlockEnd
			_ = zero.Block()
			_ = zero.String()
			_ = zero.GoString()
		}},
		{"NewTextBlockEnd with block index 0", func() {
			_, _ = ai.NewTextBlockEnd(0)
		}},
		{"wrong-kind accessors on the zero Event", func() {
			_, _ = ai.Event{}.TextBlockStart()
			_, _ = ai.Event{}.TextDelta()
			_, _ = ai.Event{}.TextBlockEnd()
		}},
		{"CheckEmit over an unstamped, unconstructed zero Event", func() {
			_ = ai.CheckEmit(ai.Event{})
		}},
		{"CheckStream over a stream mixing all three kinds, unstamped and invalid on failure", func() {
			start, _ := ai.NewTextBlockStart(0) // deliberately invalid/zero on failure
			delta, _ := ai.NewTextDelta(0, invalidUTF8Alone)
			end, _ := ai.NewTextBlockEnd(0)
			report := ai.CheckStream([]ai.Event{start, delta, end})
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
