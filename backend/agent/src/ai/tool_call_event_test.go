// Tests for AI-18 — the streamed tool-call event family.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion is written against exactly the
// surface an adapter in another package sees — the convention every Layer 1
// test file in this package follows (event_test.go, tool_call_test.go,
// response_start_test.go). assertViolation and mustText (tool_call_test.go)
// are reused as-is: one rejection-assertion helper for the whole package,
// never a second one per file.
package ai_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------------
// AI-18.1 — Call lifecycle (R-ATC-001 … R-ATC-008)
// ---------------------------------------------------------------------------

// R-ATC-001 — task 1.1: three separately registered event kinds, each with
// its own constructible payload, kind derived (never tagged), and covered by
// the AI-14 exhaustiveness guard.
func TestToolCallEvents_ThreeKinds_RegisteredDistinctAndDerivedFromThePayload(t *testing.T) {
	t.Parallel()

	t.Run("all three kinds compile, are non-zero and distinct, and are members of the registered vocabulary (S-ATC-001)", func(t *testing.T) {
		t.Parallel()

		kinds := []ai.EventKind{ai.EventKindToolCallStart, ai.EventKindToolCallDelta, ai.EventKindToolCallEnd}
		for _, k := range kinds {
			if k == 0 {
				t.Fatalf("%v is the zero kind, want a registered non-zero member", k)
			}
		}
		if kinds[0] == kinds[1] || kinds[1] == kinds[2] || kinds[0] == kinds[2] {
			t.Fatalf("the three tool-call kinds are not pairwise distinct: %v", kinds)
		}
		for _, k := range kinds {
			found := false
			for _, registered := range ai.EventKinds() {
				if registered == k {
					found = true
				}
			}
			if !found {
				t.Errorf("ai.EventKinds() = %v, want it to contain %v", ai.EventKinds(), k)
			}
		}
	})

	t.Run("each kind is derived from its own payload and matches, with no phase field or family tag (S-ATC-002)", func(t *testing.T) {
		t.Parallel()

		start, err := ai.NewToolCallStart(1, "call-01", "read_file")
		if err != nil {
			t.Fatalf("ai.NewToolCallStart returned %v, want no failure", err)
		}
		if got := start.Kind(); got != ai.EventKindToolCallStart {
			t.Errorf("start.Kind() = %v, want %v", got, ai.EventKindToolCallStart)
		}
		delta, err := ai.NewToolCallDelta(1, []byte(`{"a":1}`))
		if err != nil {
			t.Fatalf("ai.NewToolCallDelta returned %v, want no failure", err)
		}
		if got := delta.Kind(); got != ai.EventKindToolCallDelta {
			t.Errorf("delta.Kind() = %v, want %v", got, ai.EventKindToolCallDelta)
		}
		end, err := ai.NewToolCallEnd(1, []byte(`{"a":1}`))
		if err != nil {
			t.Fatalf("ai.NewToolCallEnd returned %v, want no failure", err)
		}
		if got := end.Kind(); got != ai.EventKindToolCallEnd {
			t.Errorf("end.Kind() = %v, want %v", got, ai.EventKindToolCallEnd)
		}

		// No separate lifecycle-phase discriminator: each payload carries
		// exactly its documented fields, never a fourth "which phase" field —
		// the kind (asked of the payload) is the only discriminator.
		for _, tc := range []struct {
			payload   any
			numFields int
		}{
			{ai.ToolCallStart{}, 3}, // block index, id, name
			{ai.ToolCallDelta{}, 2}, // block index, fragment
			{ai.ToolCallEnd{}, 2},   // block index, arguments
		} {
			typ := reflect.TypeOf(tc.payload)
			if got := typ.NumField(); got != tc.numFields {
				t.Errorf("%v has %d fields, want exactly %d — no separate phase field", typ, got, tc.numFields)
			}
		}
	})

	// S-ATC-003 (exhaustiveness guard covers all three; a scratch kind fails
	// it, named) is proven by event_registry_test.go's eventKindWitnesses
	// table and productionEventKinds list, extended for these three kinds in
	// the same commit — not repeated here, per design.md's Testing Strategy.
}

// R-ATC-002 — task 1.2: call-start carries identity and tool name, both
// rejected empty at construction with ErrEmpty at a position naming the
// field.
func TestNewToolCallStart_EmptyIdentityOrName_FailsWithErrEmptyAtTheOffendingPosition(t *testing.T) {
	t.Parallel()

	t.Run("non-empty identity and name read back exactly, before any delta or end exists (S-ATC-004)", func(t *testing.T) {
		t.Parallel()

		const (
			id   = "toolu_01Start"
			name = "read_file"
		)
		// No delta or end event is constructed anywhere in this sub-test:
		// the tool name and identity are readable from the start event
		// alone.
		event, err := ai.NewToolCallStart(1, id, name)
		if err != nil {
			t.Fatalf("ai.NewToolCallStart returned %v, want no failure", err)
		}
		start, ok := event.ToolCallStart()
		if !ok {
			t.Fatal("event.ToolCallStart() reported no payload on an event of its own kind")
		}
		if got := start.ID(); got != id {
			t.Errorf("start.ID() = %q, want %q", got, id)
		}
		if got := start.Name(); got != name {
			t.Errorf("start.Name() = %q, want %q", got, name)
		}
	})

	t.Run("empty identity fails with ErrEmpty at id (S-ATC-005)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewToolCallStart(1, "", "read_file")
		if err == nil {
			t.Fatal("ai.NewToolCallStart with an empty identity = nil, want a rejection")
		}
		assertViolation(t, err, ai.ErrEmpty, "id")
		if event != (ai.Event{}) {
			t.Error("a rejected ai.NewToolCallStart returned a non-zero Event, want the zero Event")
		}
	})

	t.Run("empty tool name fails with ErrEmpty at name (S-ATC-006)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewToolCallStart(1, "toolu_02", "")
		if err == nil {
			t.Fatal("ai.NewToolCallStart with an empty tool name = nil, want a rejection")
		}
		assertViolation(t, err, ai.ErrEmpty, "name")
		if event != (ai.Event{}) {
			t.Error("a rejected ai.NewToolCallStart returned a non-zero Event, want the zero Event")
		}
	})

	t.Run("the zero-value-shaped call never yields an event carrying two empty strings (S-ATC-007)", func(t *testing.T) {
		t.Parallel()

		event, err := ai.NewToolCallStart(1, "", "")
		if err == nil {
			t.Fatal(`ai.NewToolCallStart(1, "", "") = nil error, want a rejection`)
		}
		if event != (ai.Event{}) {
			t.Error("a rejected construction returned a non-zero Event, want the zero Event — a caller that ignores the error must not observe a constructed event with two empty strings")
		}
	})
}

// R-ATC-003 — task 1.3: all three events carry a shared, stream-wide block
// index; a call block's index never collides with, and is distinguishable
// without a family tag from, another block-scoped kind's index.
func TestToolCallEvents_BlockIndex_IsSharedAcrossStartDeltaAndEnd(t *testing.T) {
	t.Run("start, delta and end for one call report the same block index (S-ATC-008)", func(t *testing.T) {
		t.Parallel()

		const block = ai.BlockIndex(5)
		start := mustToolCallStart(t, block, "call-shared", "search")
		delta := mustToolCallDelta(t, block, []byte(`{"q":`))
		end := mustToolCallEnd(t, block, []byte(`{"q":"x"}`))

		startPayload, ok := start.ToolCallStart()
		if !ok {
			t.Fatal("start.ToolCallStart() reported no payload")
		}
		deltaPayload, ok := delta.ToolCallDelta()
		if !ok {
			t.Fatal("delta.ToolCallDelta() reported no payload")
		}
		endPayload, ok := end.ToolCallEnd()
		if !ok {
			t.Fatal("end.ToolCallEnd() reported no payload")
		}

		if got := startPayload.BlockIndex(); got != block {
			t.Errorf("startPayload.BlockIndex() = %v, want %v", got, block)
		}
		if got := deltaPayload.BlockIndex(); got != block {
			t.Errorf("deltaPayload.BlockIndex() = %v, want %v", got, block)
		}
		if got := endPayload.BlockIndex(); got != block {
			t.Errorf("endPayload.BlockIndex() = %v, want %v", got, block)
		}
	})

	// No t.Parallel(): ai.RegisterTestKind mutates the shared package-level
	// registry (export_test.go's own constraint).
	t.Run("a call block's index differs from a co-open block of another kind, separated with no kind lookup (S-ATC-009)", func(t *testing.T) {
		// AI-16/AI-17's text/reasoning kinds are this milestone's siblings
		// and have not landed in this package; ai.RegisterTestKind stands in
		// for "some other block-scoped kind" — the generic mechanism AI-14
		// built for exactly this purpose (export_test.go).
		otherKind, cleanup := ai.RegisterTestKind("stand_in_other_block", ai.EventDescriptor{Role: ai.BlockRoleStart})
		t.Cleanup(cleanup)

		call := mustToolCallStart(t, 1, "call-x", "search")
		other := ai.NewTestEvent(otherKind, 2)

		callPayload, ok := call.ToolCallStart()
		if !ok {
			t.Fatal("call.ToolCallStart() reported no payload")
		}
		otherPayload, ok := other.WitnessPayload()
		if !ok {
			t.Fatal("other.WitnessPayload() reported no payload")
		}

		if callPayload.BlockIndex() == otherPayload.BlockIndex() {
			t.Fatalf("the call block's index (%v) collides with the other kind's block index (%v); they must differ", callPayload.BlockIndex(), otherPayload.BlockIndex())
		}
		// The two are separated by the index alone: neither Kind() nor a
		// family tag needs to be consulted to know they are different
		// blocks — this assertion is the index comparison above; no second
		// mechanism exists to fall back to.
	})

	t.Run("the landed surface exposes no call-scoped counter, per-family numbering space, or family tag (S-ATC-010)", func(t *testing.T) {
		t.Parallel()

		for _, typ := range []reflect.Type{
			reflect.TypeOf(ai.ToolCallStart{}),
			reflect.TypeOf(ai.ToolCallDelta{}),
			reflect.TypeOf(ai.ToolCallEnd{}),
		} {
			for i := 0; i < typ.NumMethod(); i++ {
				name := typ.Method(i).Name
				lower := strings.ToLower(name)
				if strings.Contains(lower, "family") || strings.Contains(lower, "phase") || strings.Contains(lower, "counter") {
					t.Errorf("%v exports method %q, which reads like a call-scoped counter or family tag; R-ATC-003 forbids a second index space", typ, name)
				}
			}
		}
	})
}

// R-ATC-004 — task 1.4: block index is 1-based; 0 is rejected at
// construction with AI-04's out-of-range sentinel, at a position naming the
// index field.
func TestToolCallEvents_BlockIndexZero_FailsWithErrOutOfRange(t *testing.T) {
	t.Parallel()

	t.Run("block index 0 is rejected on all three kinds (S-ATC-011)", func(t *testing.T) {
		t.Parallel()

		t.Run("call-start", func(t *testing.T) {
			t.Parallel()
			event, err := ai.NewToolCallStart(0, "call-01", "search")
			if err == nil {
				t.Fatal("ai.NewToolCallStart with block index 0 = nil, want a rejection")
			}
			assertViolation(t, err, ai.ErrOutOfRange, "block_index")
			if event != (ai.Event{}) {
				t.Error("a rejected construction returned a non-zero Event, want the zero Event")
			}
		})

		t.Run("call-delta", func(t *testing.T) {
			t.Parallel()
			event, err := ai.NewToolCallDelta(0, []byte(`{"a":1}`))
			if err == nil {
				t.Fatal("ai.NewToolCallDelta with block index 0 = nil, want a rejection")
			}
			assertViolation(t, err, ai.ErrOutOfRange, "block_index")
			if event != (ai.Event{}) {
				t.Error("a rejected construction returned a non-zero Event, want the zero Event")
			}
		})

		t.Run("call-end", func(t *testing.T) {
			t.Parallel()
			event, err := ai.NewToolCallEnd(0, []byte(`{"a":1}`))
			if err == nil {
				t.Fatal("ai.NewToolCallEnd with block index 0 = nil, want a rejection")
			}
			assertViolation(t, err, ai.ErrOutOfRange, "block_index")
			if event != (ai.Event{}) {
				t.Error("a rejected construction returned a non-zero Event, want the zero Event")
			}
		})
	})

	t.Run("the first block of a stream reads back as index 1, not 0 (S-ATC-012)", func(t *testing.T) {
		t.Parallel()

		start := mustToolCallStart(t, 1, "call-first", "search")
		payload, ok := start.ToolCallStart()
		if !ok {
			t.Fatal("start.ToolCallStart() reported no payload")
		}
		if got := payload.BlockIndex(); got != 1 {
			t.Errorf("payload.BlockIndex() = %v, want 1", got)
		}
	})
}

// R-ATC-005 — task 1.5: a delta carries only the new fragment; zero-length
// and whitespace-only fragments are accepted unaltered; no accessor returns
// accumulated content.
func TestToolCallDelta_Fragment_CarriesOnlyTheNewBytes(t *testing.T) {
	t.Parallel()

	t.Run("three deltas read back exactly their own fragments, never a growing prefix (S-ATC-013)", func(t *testing.T) {
		t.Parallel()

		fragments := [][]byte{[]byte(`{"a"`), []byte(`:1`), []byte(`}`)}
		for i, fragment := range fragments {
			delta := mustToolCallDelta(t, 1, fragment)
			payload, ok := delta.ToolCallDelta()
			if !ok {
				t.Fatalf("delta[%d].ToolCallDelta() reported no payload", i)
			}
			if got := payload.Fragment(); !bytes.Equal(got, fragment) {
				t.Errorf("delta[%d].Fragment() = %q, want %q — never an accumulated prefix", i, got, fragment)
			}
		}
	})

	t.Run("a zero-length fragment and a single-space fragment both succeed unaltered (S-ATC-014)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			what     string
			fragment []byte
		}{
			{"zero-length", []byte{}},
			{"nil", nil},
			{"a single space", []byte(" ")},
		} {
			t.Run(tc.what, func(t *testing.T) {
				t.Parallel()

				event, err := ai.NewToolCallDelta(1, tc.fragment)
				if err != nil {
					t.Fatalf("ai.NewToolCallDelta with a %s fragment returned %v, want no failure", tc.what, err)
				}
				payload, ok := event.ToolCallDelta()
				if !ok {
					t.Fatal("event.ToolCallDelta() reported no payload")
				}
				if got, want := payload.Fragment(), tc.fragment; !bytes.Equal(got, want) {
					t.Errorf("payload.Fragment() = %q, want %q unaltered", got, want)
				}
			})
		}
	})

	t.Run("exactly two field-reading accessors exist on the delta payload, neither named as an accumulator (S-ATC-015)", func(t *testing.T) {
		t.Parallel()

		typ := reflect.TypeOf(ai.ToolCallDelta{})
		var accessors []string
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			if name == "String" || name == "GoString" {
				continue
			}
			accessors = append(accessors, name)
		}
		sort.Strings(accessors)
		want := []string{"BlockIndex", "Fragment"}
		if !reflect.DeepEqual(accessors, want) {
			t.Errorf("ai.ToolCallDelta's exported accessors = %v, want exactly %v — no accumulator or snapshot accessor", accessors, want)
		}
	})
}

// canonicalizingToolCallFixture mirrors tool_call_test.go's
// canonicalizingFixture: bytes a re-marshalling implementation would rewrite
// (sorted-key order, irregular whitespace, a multi-line nested array), chosen
// so a contract that decoded and re-encoded them fails a byte comparison.
const canonicalizingToolCallFixture = `{ "zeta": 1,
  "alpha":   { "nested": [ 1,  2 , 3 ] } ,
  "beta" : "  spaced  " }`

// R-ATC-006 — task 1.6: call-end's argument bytes are byte-equal to the
// concatenated deltas, never re-marshalled, and every accessor returns a
// fresh copy.
func TestToolCallEnd_Arguments_AreByteEqualAndNeverReMarshalled(t *testing.T) {
	t.Parallel()

	t.Run("concatenating a call's deltas in arrival order equals the call-end's argument bytes (S-ATC-016)", func(t *testing.T) {
		t.Parallel()

		fragments := [][]byte{[]byte(`{"path":`), []byte(`"/etc/hosts",`), []byte(`"limit":10}`)}
		var want []byte
		for _, f := range fragments {
			want = append(want, f...)
		}

		end := mustToolCallEnd(t, 1, want)
		payload, ok := end.ToolCallEnd()
		if !ok {
			t.Fatal("end.ToolCallEnd() reported no payload")
		}
		if got := payload.Arguments(); !bytes.Equal(got, want) {
			t.Errorf("payload.Arguments() = %q, want %q (the concatenated fragments)", got, want)
		}
	})

	t.Run("whitespace, non-canonical key order and interior newlines survive unaltered (S-ATC-017)", func(t *testing.T) {
		t.Parallel()

		supplied := []byte(canonicalizingToolCallFixture)
		end := mustToolCallEnd(t, 1, supplied)
		payload, ok := end.ToolCallEnd()
		if !ok {
			t.Fatal("end.ToolCallEnd() reported no payload")
		}
		if got := payload.Arguments(); !bytes.Equal(got, supplied) {
			t.Errorf("payload.Arguments() = %q,\n                      want %q\n"+
				"  no re-marshalling, no key reordering, no whitespace normalization.", got, supplied)
		}
	})

	t.Run("a caller that mutates the returned bytes cannot affect the event, and two reads agree (S-ATC-018)", func(t *testing.T) {
		t.Parallel()

		original := []byte(canonicalizingToolCallFixture)
		end := mustToolCallEnd(t, 1, original)
		payload, ok := end.ToolCallEnd()
		if !ok {
			t.Fatal("end.ToolCallEnd() reported no payload")
		}

		first := payload.Arguments()
		for i := range first {
			first[i] = 'X'
		}

		second := payload.Arguments()
		if !bytes.Equal(second, original) {
			t.Errorf("after a caller mutated the first read, a second read returned %q, want the original %q", second, original)
		}
	})
}

// R-ATC-007 — task 1.7: call-end performs no JSON well-formedness validation
// and no empty-argument canonicalization.
func TestNewToolCallEnd_ArgumentBytes_AreNeverValidatedOrCanonicalized(t *testing.T) {
	t.Parallel()

	t.Run("bytes that are not well-formed JSON still construct and read back byte-equal (S-ATC-019)", func(t *testing.T) {
		t.Parallel()

		malformed := []byte(`{"path":`)
		event, err := ai.NewToolCallEnd(1, malformed)
		if err != nil {
			t.Fatalf("ai.NewToolCallEnd rejected malformed-but-transport-legal bytes with %v; R-ATC-007 defers well-formedness checking to AI-30", err)
		}
		payload, ok := event.ToolCallEnd()
		if !ok {
			t.Fatal("event.ToolCallEnd() reported no payload")
		}
		if got := payload.Arguments(); !bytes.Equal(got, malformed) {
			t.Errorf("payload.Arguments() = %q, want %q", got, malformed)
		}
	})

	t.Run("zero-length arguments stay zero-length, never replaced by {} (S-ATC-020)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			what string
			args []byte
		}{
			{"nil", nil},
			{"empty slice", []byte{}},
		} {
			t.Run(tc.what, func(t *testing.T) {
				t.Parallel()
				event, err := ai.NewToolCallEnd(1, tc.args)
				if err != nil {
					t.Fatalf("ai.NewToolCallEnd with %s arguments returned %v, want no failure", tc.what, err)
				}
				payload, ok := event.ToolCallEnd()
				if !ok {
					t.Fatal("event.ToolCallEnd() reported no payload")
				}
				if got := payload.Arguments(); len(got) != 0 {
					t.Errorf("payload.Arguments() = %q, want zero-length — R-ATC-007 forbids the {} canonicalization NewToolCall applies to absent arguments", got)
				}
			})
		}
	})

	t.Run("the event path calls no JSON well-formedness check, and the AI-30 deferral is stated in the contract text (S-ATC-021)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "tool_call_event.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing tool_call_event.go: %v", err)
		}

		forbidden := []string{"isWellFormedJSON", "emptyToolArguments"}
		ast.Inspect(file, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			for _, name := range forbidden {
				if ident.Name == name {
					t.Errorf("tool_call_event.go references %q; R-ATC-007 forbids a JSON well-formedness check or empty-argument canonicalization on the event path", ident.Name)
				}
			}
			return true
		})

		raw, err := readSourceFile(t, "tool_call_event.go")
		if err != nil {
			t.Fatalf("reading tool_call_event.go: %v", err)
		}
		if !strings.Contains(raw, "AI-30") {
			t.Error("tool_call_event.go does not mention AI-30; the deferral of well-formedness checking must be stated in the contract text")
		}
	})
}

// R-ATC-008 — task 1.8: identity, name, fragment and argument bytes are
// redacted from String()/GoString() on all three payloads.
func TestToolCallEvents_Rendering_NeverReproducesTheirPayload(t *testing.T) {
	t.Parallel()

	const (
		secretID   = "toolu_secret_9f8e7d"
		secretName = "read_secret_file"
	)
	secretFragment := []byte(`{"token":"sk-live-canary"}`)

	start, ok := mustToolCallStart(t, 1, secretID, secretName).ToolCallStart()
	if !ok {
		t.Fatal("event.ToolCallStart() reported no payload")
	}
	delta, ok := mustToolCallDelta(t, 1, secretFragment).ToolCallDelta()
	if !ok {
		t.Fatal("event.ToolCallDelta() reported no payload")
	}
	end, ok := mustToolCallEnd(t, 1, secretFragment).ToolCallEnd()
	if !ok {
		t.Fatal("event.ToolCallEnd() reported no payload")
	}

	cases := []struct {
		name   string
		value  fmt.Stringer
		secret []string
		want   string
	}{
		{"start", start, []string{secretID, secretName}, "toolcallstart"},
		{"delta", delta, []string{string(secretFragment)}, "toolcalldelta"},
		{"end", end, []string{string(secretFragment)}, "toolcallend"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
				rendered := fmt.Sprintf(verb, tc.value)
				if rendered != tc.want {
					t.Errorf("fmt.Sprintf(%q, %s) = %q, want %q", verb, tc.name, rendered, tc.want)
				}
				for _, secret := range tc.secret {
					if strings.Contains(rendered, secret) {
						t.Errorf("fmt.Sprintf(%q, %s) = %q, which reproduces a value it carries", verb, tc.name, rendered)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AI-18.2 — Delta optionality (R-ATC-009, R-ATC-010)
// ---------------------------------------------------------------------------

// R-ATC-009 — tasks 2.1/2.3: a call with zero deltas is legal and complete.
func TestToolCallEvents_ZeroDeltaCall_IsLegalAndComplete(t *testing.T) {
	t.Parallel()

	t.Run("a start immediately followed by its end, no delta between them, reports no violation (S-ATC-023)", func(t *testing.T) {
		t.Parallel()

		var stamper ai.Stamper
		events := []ai.Event{
			stamper.Stamp(mustToolCallStart(t, 1, "call-zero-delta", "search")),
			stamper.Stamp(mustToolCallEnd(t, 1, []byte(`{"q":"x"}`))),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil", err)
		}
	})

	t.Run("a zero-delta call and a multi-delta call both close normally, distinguished by block index alone (S-ATC-024)", func(t *testing.T) {
		t.Parallel()

		var stamper ai.Stamper
		events := []ai.Event{
			stamper.Stamp(mustToolCallStart(t, 1, "call-zero", "search")),
			stamper.Stamp(mustToolCallStart(t, 2, "call-multi", "read_file")),
			stamper.Stamp(mustToolCallDelta(t, 2, []byte(`{"p":`))),
			stamper.Stamp(mustToolCallDelta(t, 2, []byte(`"a"}`))),
			stamper.Stamp(mustToolCallEnd(t, 1, []byte(`{}`))),
			stamper.Stamp(mustToolCallEnd(t, 2, []byte(`{"p":"a"}`))),
		}
		report := ai.CheckStream(events)
		if err := report.Violation(); err != nil {
			t.Errorf("report.Violation() = %v, want nil — a zero-delta call must not be confused with an unterminated block", err)
		}
	})

	t.Run("the contract text states explicitly that no consumer may require a delta (S-ATC-025)", func(t *testing.T) {
		t.Parallel()

		raw, err := readSourceFile(t, "tool_call_event.go")
		if err != nil {
			t.Fatalf("reading tool_call_event.go: %v", err)
		}
		// Doc comments wrap at the house line width, so a multi-word phrase
		// may span a "\n// " boundary in the raw bytes; strip the leading
		// "//" line-comment marker from every field and collapse remaining
		// whitespace (newlines included) to one space before matching, so
		// the check is robust to where a comment happens to wrap.
		var words []string
		for _, field := range strings.Fields(raw) {
			if field == "//" {
				continue
			}
			words = append(words, field)
		}
		normalized := strings.Join(words, " ")
		if !strings.Contains(normalized, "zero deltas") || !strings.Contains(normalized, "is legal and complete") {
			t.Error("tool_call_event.go does not state, in its contract text, that a zero-delta call is legal and complete")
		}
		if !strings.Contains(normalized, "No consumer contract in this file requires at least one delta") {
			t.Error("tool_call_event.go does not explicitly state that no consumer contract requires a delta (R-ATC-009)")
		}
	})
}

// R-ATC-010 — task 2.2: whole and fragmented deliveries are indistinguishable
// after reconstruction.
func TestToolCallEvents_WholeAndFragmentedDeliveries_AreIndistinguishableAfterReconstruction(t *testing.T) {
	t.Parallel()

	t.Run("the same arguments delivered as zero deltas or split across five reconstruct equal (S-ATC-026)", func(t *testing.T) {
		t.Parallel()

		const (
			id   = "call-delivery-compare"
			name = "search"
		)
		arguments := []byte(`{"query":"cachicamas layer 1","limit":5,"offset":0}`)

		// Delivery A: zero deltas, the whole payload on the end.
		startA := mustToolCallStart(t, 1, id, name)
		endA := mustToolCallEnd(t, 1, arguments)

		// Delivery B: the identical bytes split across five deltas.
		startB := mustToolCallStart(t, 2, id, name)
		var deltaEvents []ai.Event
		for _, fragment := range splitInto(arguments, 5) {
			deltaEvents = append(deltaEvents, mustToolCallDelta(t, 2, fragment))
		}
		reconstructed := deltaFragmentsOf(deltaEvents)
		endB := mustToolCallEnd(t, 2, reconstructed)

		payloadStartA, ok := startA.ToolCallStart()
		if !ok {
			t.Fatal("startA.ToolCallStart() reported no payload")
		}
		payloadStartB, ok := startB.ToolCallStart()
		if !ok {
			t.Fatal("startB.ToolCallStart() reported no payload")
		}
		payloadEndA, ok := endA.ToolCallEnd()
		if !ok {
			t.Fatal("endA.ToolCallEnd() reported no payload")
		}
		payloadEndB, ok := endB.ToolCallEnd()
		if !ok {
			t.Fatal("endB.ToolCallEnd() reported no payload")
		}

		if payloadStartA.ID() != payloadStartB.ID() || payloadStartA.Name() != payloadStartB.Name() {
			t.Errorf("identity/name differ between deliveries: A=(%q,%q) B=(%q,%q)",
				payloadStartA.ID(), payloadStartA.Name(), payloadStartB.ID(), payloadStartB.Name())
		}
		if !bytes.Equal(payloadEndA.Arguments(), payloadEndB.Arguments()) {
			t.Errorf("argument bytes differ between deliveries: A=%q B=%q", payloadEndA.Arguments(), payloadEndB.Arguments())
		}
		if !bytes.Equal(payloadEndA.Arguments(), arguments) {
			t.Errorf("delivery A's arguments = %q, want the original %q", payloadEndA.Arguments(), arguments)
		}
	})

	t.Run("no accessor on start or end exposes a delta count, fragment-boundary list, or fragmentation indicator (S-ATC-027)", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			typ  reflect.Type
			want []string
		}{
			{reflect.TypeOf(ai.ToolCallStart{}), []string{"BlockIndex", "ID", "Name"}},
			{reflect.TypeOf(ai.ToolCallEnd{}), []string{"Arguments", "BlockIndex"}},
		} {
			var accessors []string
			for i := 0; i < tc.typ.NumMethod(); i++ {
				name := tc.typ.Method(i).Name
				if name == "String" || name == "GoString" {
					continue
				}
				accessors = append(accessors, name)
			}
			sort.Strings(accessors)
			sort.Strings(tc.want)
			if !reflect.DeepEqual(accessors, tc.want) {
				t.Errorf("%v's exported accessors = %v, want exactly %v — no delta-count or fragmentation signal", tc.typ, accessors, tc.want)
			}
		}
	})

	t.Run("a split dividing a multi-byte rune still decodes to the original rune, no replacement character (S-ATC-028)", func(t *testing.T) {
		t.Parallel()

		// "€" is U+20AC, encoded as the 3-byte UTF-8 sequence 0xE2 0x82 0xAC.
		rune3Byte := []byte("€")
		if len(rune3Byte) != 3 {
			t.Fatalf("test fixture error: %q encoded to %d bytes, want 3", rune3Byte, len(rune3Byte))
		}

		delta1 := mustToolCallDelta(t, 1, rune3Byte[:1])
		delta2 := mustToolCallDelta(t, 1, rune3Byte[1:])

		p1, ok := delta1.ToolCallDelta()
		if !ok {
			t.Fatal("delta1.ToolCallDelta() reported no payload")
		}
		p2, ok := delta2.ToolCallDelta()
		if !ok {
			t.Fatal("delta2.ToolCallDelta() reported no payload")
		}

		reconstructed := append(append([]byte{}, p1.Fragment()...), p2.Fragment()...)
		if !bytes.Equal(reconstructed, rune3Byte) {
			t.Fatalf("reconstructed = %x, want %x", reconstructed, rune3Byte)
		}

		r, size := utf8.DecodeRune(reconstructed)
		if r == utf8.RuneError {
			t.Errorf("utf8.DecodeRune(%x) reported RuneError, want the original rune decoded cleanly", reconstructed)
		}
		if r != '€' || size != 3 {
			t.Errorf("utf8.DecodeRune(%x) = (%q, %d), want ('€', 3)", reconstructed, r, size)
		}
	})
}

// ---------------------------------------------------------------------------
// AI-18.3 — Interleaving and the call ordinal (R-ATC-011, R-ATC-012)
// ---------------------------------------------------------------------------

// R-ATC-011 — tasks 3.1: interleaved calls reconstruct independently, by
// block index alone, with no cross-contamination.
func TestToolCallEvents_Interleaved_ReconstructIndependently(t *testing.T) {
	t.Parallel()

	var stamper ai.Stamper
	// Two calls whose start/delta/end events are deliberately interleaved,
	// never grouped by call.
	events := []ai.Event{
		stamper.Stamp(mustToolCallStart(t, 1, "call-alpha", "read_file")),
		stamper.Stamp(mustToolCallStart(t, 2, "call-beta", "write_file")),
		stamper.Stamp(mustToolCallDelta(t, 1, []byte(`{"path":`))),
		stamper.Stamp(mustToolCallDelta(t, 2, []byte(`{"path":"b",`))),
		stamper.Stamp(mustToolCallDelta(t, 1, []byte(`"a"}`))),
		stamper.Stamp(mustToolCallDelta(t, 2, []byte(`"data":"beta-payload"}`))),
		stamper.Stamp(mustToolCallEnd(t, 1, []byte(`{"path":"a"}`))),
		stamper.Stamp(mustToolCallEnd(t, 2, []byte(`{"path":"b","data":"beta-payload"}`))),
	}

	t.Run("every event partitions to its own call by block index alone (S-ATC-029)", func(t *testing.T) {
		t.Parallel()

		byBlock := partitionByBlock(t, events)
		if len(byBlock) != 2 {
			t.Fatalf("partitionByBlock returned %d blocks, want 2", len(byBlock))
		}
		if len(byBlock[1]) != 4 {
			t.Errorf("block 1 has %d events, want 4 (start, 2 deltas, end)", len(byBlock[1]))
		}
		if len(byBlock[2]) != 4 {
			t.Errorf("block 2 has %d events, want 4 (start, 2 deltas, end)", len(byBlock[2]))
		}
	})

	t.Run("no fragment of one call appears in the other's reconstruction (S-ATC-030)", func(t *testing.T) {
		t.Parallel()

		byBlock := partitionByBlock(t, events)

		alpha := reconstruct(t, byBlock[1])
		beta := reconstruct(t, byBlock[2])

		if alpha.id != "call-alpha" || alpha.name != "read_file" {
			t.Errorf("alpha reconstruction = %+v, want id=call-alpha name=read_file", alpha)
		}
		if beta.id != "call-beta" || beta.name != "write_file" {
			t.Errorf("beta reconstruction = %+v, want id=call-beta name=write_file", beta)
		}
		if !bytes.Equal(alpha.arguments, []byte(`{"path":"a"}`)) {
			t.Errorf("alpha.arguments = %q, want %q", alpha.arguments, `{"path":"a"}`)
		}
		if !bytes.Equal(beta.arguments, []byte(`{"path":"b","data":"beta-payload"}`)) {
			t.Errorf("beta.arguments = %q, want %q", beta.arguments, `{"path":"b","data":"beta-payload"}`)
		}
		if bytes.Contains(alpha.arguments, []byte("beta-payload")) {
			t.Error("alpha's reconstruction contains a byte sequence from beta's delta — cross-contamination")
		}
		if bytes.Contains(beta.arguments, []byte(`"a"`)) && string(beta.arguments) != `{"path":"b","data":"beta-payload"}` {
			t.Error("beta's reconstruction appears contaminated by alpha's fragment")
		}
	})

	// S-ATC-031 — this whole file runs under `go test -race`, per `make
	// test`'s pinned body (Makefile); the interleaving above is built from a
	// single goroutine (matching AI-02 §4's one-producer-per-stream
	// invariant), and a clean `-race` run of this test function is itself
	// the proof: there is no second mechanism that would show up
	// differently under the race detector than under a plain run.
}

// R-ATC-012 — task 3.2: the call ordinal is derived from the event
// sequence, distinct from the block index, and stored nowhere.
func TestToolCallEvents_Ordinal_IsDerivedDistinctFromBlockIndexAndStoredNowhere(t *testing.T) {
	// No t.Parallel() at the top level: the first sub-test calls
	// ai.RegisterTestKind, which export_test.go forbids running in
	// parallel with anything touching the shared registry.

	t.Run("a text block, then two call blocks: ordinals follow emission order and differ from block index (S-ATC-032)", func(t *testing.T) {
		textKind, cleanup := ai.RegisterTestKind("stand_in_text_block_for_ordinal", ai.EventDescriptor{Role: ai.BlockRoleStart})
		t.Cleanup(cleanup)

		var stamper ai.Stamper
		events := []ai.Event{
			stamper.Stamp(ai.NewTestEvent(textKind, 1)),           // block 1: not a call
			stamper.Stamp(mustToolCallStart(t, 2, "call-1", "a")), // block 2: call ordinal 1
			stamper.Stamp(mustToolCallStart(t, 3, "call-2", "b")), // block 3: call ordinal 2
		}

		ordinals := callOrdinals(events)
		if len(ordinals) != 2 {
			t.Fatalf("callOrdinals returned %d calls, want 2", len(ordinals))
		}
		if ordinals[0].id != "call-1" || ordinals[0].ordinal != 1 {
			t.Errorf("ordinals[0] = %+v, want id=call-1 ordinal=1", ordinals[0])
		}
		if ordinals[1].id != "call-2" || ordinals[1].ordinal != 2 {
			t.Errorf("ordinals[1] = %+v, want id=call-2 ordinal=2", ordinals[1])
		}
		// The ordinal (1, 2) differs from the block index (2, 3) precisely
		// because a non-call block occupies index 1.
		if int(ordinals[0].block) == ordinals[0].ordinal && int(ordinals[1].block) == ordinals[1].ordinal {
			t.Error("test fixture error: ordinal must differ from block index for this sub-test to prove anything")
		}
	})

	t.Run("no field or exported accessor on any of the three payloads carries an ordinal, index-among-calls, or sequence-number (S-ATC-033)", func(t *testing.T) {
		t.Parallel()

		forbidden := []string{"ordinal", "indexamongcalls", "callsequence", "callnumber", "position"}
		for _, typ := range []reflect.Type{
			reflect.TypeOf(ai.ToolCallStart{}),
			reflect.TypeOf(ai.ToolCallDelta{}),
			reflect.TypeOf(ai.ToolCallEnd{}),
		} {
			for i := 0; i < typ.NumField(); i++ {
				name := strings.ToLower(typ.Field(i).Name)
				for _, bad := range forbidden {
					if strings.Contains(name, bad) {
						t.Errorf("%v has field %q, which reads like an ordinal; R-ATC-012 forbids storing one", typ, typ.Field(i).Name)
					}
				}
			}
			for i := 0; i < typ.NumMethod(); i++ {
				name := strings.ToLower(typ.Method(i).Name)
				for _, bad := range forbidden {
					if strings.Contains(name, bad) {
						t.Errorf("%v exports method %q, which reads like an ordinal accessor; R-ATC-012 forbids one", typ, typ.Method(i).Name)
					}
				}
			}
		}
	})

	t.Run("interleaved calls' ordinals follow start-event order and are stable across repeated derivation (S-ATC-034)", func(t *testing.T) {
		t.Parallel()

		var stamper ai.Stamper
		events := []ai.Event{
			stamper.Stamp(mustToolCallStart(t, 5, "call-first", "a")),
			stamper.Stamp(mustToolCallStart(t, 9, "call-second", "b")),
			stamper.Stamp(mustToolCallDelta(t, 5, []byte("x"))),
			stamper.Stamp(mustToolCallDelta(t, 9, []byte("y"))),
			stamper.Stamp(mustToolCallEnd(t, 5, []byte("x"))),
			stamper.Stamp(mustToolCallEnd(t, 9, []byte("y"))),
		}

		first := callOrdinals(events)
		for range 8 {
			again := callOrdinals(events)
			if !reflect.DeepEqual(first, again) {
				t.Fatalf("callOrdinals is not stable across repeated derivation: first=%+v, again=%+v", first, again)
			}
		}
		if len(first) != 2 || first[0].id != "call-first" || first[1].id != "call-second" {
			t.Fatalf("callOrdinals(events) = %+v, want call-first at ordinal 1 and call-second at ordinal 2, in start-event order", first)
		}
	})
}

// blockEvents is one block's events, keyed by block index — test-local
// partitioning, never a helper this package ships (R-AEE-020's "no
// accumulator" posture, restated for the test's own convenience code).
type blockEvents map[ai.BlockIndex][]ai.Event

// partitionByBlock groups events by block index, reading the index through
// each event's own typed accessor (ToolCallStart/Delta/End) — never a type
// switch on an unexported payload type, which package ai does not expose.
func partitionByBlock(t *testing.T, events []ai.Event) blockEvents {
	t.Helper()
	out := make(blockEvents)
	for _, e := range events {
		var idx ai.BlockIndex
		switch {
		case e.Kind() == ai.EventKindToolCallStart:
			p, _ := e.ToolCallStart()
			idx = p.BlockIndex()
		case e.Kind() == ai.EventKindToolCallDelta:
			p, _ := e.ToolCallDelta()
			idx = p.BlockIndex()
		case e.Kind() == ai.EventKindToolCallEnd:
			p, _ := e.ToolCallEnd()
			idx = p.BlockIndex()
		default:
			continue
		}
		out[idx] = append(out[idx], e)
	}
	return out
}

// reconstructedCall is one call's test-local reconstruction: identity, name
// and the ordered concatenation of its delta fragments, cross-checked
// against its call-end's argument bytes.
type reconstructedCall struct {
	id, name  string
	arguments []byte
}

// deltaFragmentsOf concatenates every ToolCallDelta event's fragment, in
// slice order, ignoring events of any other kind. It is the one test-local
// concatenator this file uses (task 3.4's refactor: previously duplicated
// inline in the S-ATC-026 sub-test and in [reconstruct] — extracted once
// Phase 3 needed the identical loop a second time). Layer 1 itself ships no
// such helper (R-AEE-020); this one lives in the test binary only.
func deltaFragmentsOf(events []ai.Event) []byte {
	var out []byte
	for _, e := range events {
		if p, ok := e.ToolCallDelta(); ok {
			out = append(out, p.Fragment()...)
		}
	}
	return out
}

// reconstruct rebuilds one call's logical view from its own partitioned
// events (test-local; Layer 1 ships no reconstruction helper, R-AEE-020).
func reconstruct(t *testing.T, events []ai.Event) reconstructedCall {
	t.Helper()
	var out reconstructedCall
	fromDeltas := deltaFragmentsOf(events)
	for _, e := range events {
		switch e.Kind() {
		case ai.EventKindToolCallStart:
			p, _ := e.ToolCallStart()
			out.id, out.name = p.ID(), p.Name()
		case ai.EventKindToolCallEnd:
			p, _ := e.ToolCallEnd()
			out.arguments = p.Arguments()
			if !bytes.Equal(fromDeltas, p.Arguments()) {
				t.Errorf("concatenated deltas %q disagree with call-end arguments %q for call %q", fromDeltas, p.Arguments(), out.id)
			}
		}
	}
	return out
}

// callOrdinal is one call's derived ordinal — the stream-side continuation
// of ToolCalls(), reimplemented test-locally per R-ATC-012 (no such helper
// ships in package ai).
type callOrdinal struct {
	id      string
	block   ai.BlockIndex
	ordinal int
}

// callOrdinals derives each call's 1-based ordinal by filtering events to
// call-start events, in stream (emission) order, and counting — exactly
// R-ATC-012's documented derivation.
func callOrdinals(events []ai.Event) []callOrdinal {
	var out []callOrdinal
	n := 0
	for _, e := range events {
		start, ok := e.ToolCallStart()
		if !ok {
			continue
		}
		n++
		out = append(out, callOrdinal{id: start.ID(), block: start.BlockIndex(), ordinal: n})
	}
	return out
}

// ---------------------------------------------------------------------------
// Phase 4 — Non-functional, evidence, acceptance (NFR-ATC-A … NFR-ATC-D)
// ---------------------------------------------------------------------------

// NFR-ATC-A (S-ATC-035) — dependency purity: this file adds no import at
// all (mirrors response_start.go/completion.go, which also import nothing),
// so `backend/agent/go.mod`'s zero requires and both AI-00 import guards are
// unaffected by this milestone. Proven by the existing, unchanged guard
// suite (import_boundary_test.go, go.mod itself) per design.md's Testing
// Strategy table — not repeated here.

// NFR-ATC-B — task 4.1/4.2: no exported entry point of the three kinds
// panics for any extreme input, including each payload's zero value.
func TestToolCallEvents_Totality_NoExportedEntryPointPanics(t *testing.T) {
	t.Parallel()

	invalidUTF8 := []byte{0xff, 0xfe, 0x00, 0xff}
	malformedJSON := []byte(`{"unterminated`)

	run := func(t *testing.T, name string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s panicked: %v", name, r)
			}
		}()
		fn()
	}

	t.Run("NewToolCallStart", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			what     string
			block    ai.BlockIndex
			id, name string
		}{
			{"zero value inputs", 0, "", ""},
			{"block 0, valid strings", 0, "id", "name"},
			{"valid block, empty id and name", 1, "", ""},
			{"very large block index", ai.BlockIndex(1<<63 - 1), "id", "name"},
		} {
			run(t, "NewToolCallStart/"+tc.what, func() {
				event, err := ai.NewToolCallStart(tc.block, tc.id, tc.name)
				_ = err
				_ = event.Kind()
				_ = event.String()
				_ = event.GoString()
				start, _ := event.ToolCallStart()
				_ = start.BlockIndex()
				_ = start.ID()
				_ = start.Name()
				_ = start.String()
				_ = start.GoString()
			})
		}
	})

	t.Run("NewToolCallDelta", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			what     string
			block    ai.BlockIndex
			fragment []byte
		}{
			{"zero value inputs", 0, nil},
			{"valid block, nil fragment", 1, nil},
			{"valid block, empty fragment", 1, []byte{}},
			{"valid block, invalid UTF-8 fragment", 1, invalidUTF8},
			{"valid block, whitespace-only fragment", 1, []byte("   \t\n")},
		} {
			run(t, "NewToolCallDelta/"+tc.what, func() {
				event, err := ai.NewToolCallDelta(tc.block, tc.fragment)
				_ = err
				_ = event.Kind()
				delta, _ := event.ToolCallDelta()
				_ = delta.BlockIndex()
				fragment := delta.Fragment()
				if len(fragment) > 0 {
					fragment[0] = 0 // mutate the returned copy; must never panic or affect the payload
				}
				_ = delta.Fragment()
				_ = delta.String()
				_ = delta.GoString()
			})
		}
	})

	t.Run("NewToolCallEnd", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			what      string
			block     ai.BlockIndex
			arguments []byte
		}{
			{"zero value inputs", 0, nil},
			{"valid block, nil arguments", 1, nil},
			{"valid block, empty arguments", 1, []byte{}},
			{"valid block, malformed JSON", 1, malformedJSON},
			{"valid block, invalid UTF-8 arguments", 1, invalidUTF8},
		} {
			run(t, "NewToolCallEnd/"+tc.what, func() {
				event, err := ai.NewToolCallEnd(tc.block, tc.arguments)
				_ = err
				_ = event.Kind()
				end, _ := event.ToolCallEnd()
				_ = end.BlockIndex()
				args := end.Arguments()
				if len(args) > 0 {
					args[0] = 0
				}
				_ = end.Arguments()
				_ = end.String()
				_ = end.GoString()
			})
		}
	})

	t.Run("zero-value payloads read through every exported accessor", func(t *testing.T) {
		t.Parallel()
		run(t, "zero-value ToolCallStart", func() {
			var s ai.ToolCallStart
			_ = s.BlockIndex()
			_ = s.ID()
			_ = s.Name()
			_ = s.String()
			_ = s.GoString()
		})
		run(t, "zero-value ToolCallDelta", func() {
			var d ai.ToolCallDelta
			_ = d.BlockIndex()
			_ = d.Fragment()
			_ = d.String()
			_ = d.GoString()
		})
		run(t, "zero-value ToolCallEnd", func() {
			var c ai.ToolCallEnd
			_ = c.BlockIndex()
			_ = c.Arguments()
			_ = c.String()
			_ = c.GoString()
		})
		run(t, "typed accessors on the zero Event", func() {
			var e ai.Event
			if _, ok := e.ToolCallStart(); ok {
				t.Error("the zero Event reported a ToolCallStart payload")
			}
			if _, ok := e.ToolCallDelta(); ok {
				t.Error("the zero Event reported a ToolCallDelta payload")
			}
			if _, ok := e.ToolCallEnd(); ok {
				t.Error("the zero Event reported a ToolCallEnd payload")
			}
		})
	})
}

// NFR-ATC-C — S-ATC-037: every rejecting scenario in this file reports
// through AI-04's *ai.Violation, matches a landed sentinel via errors.Is,
// and names the offending field. assertViolation (tool_call_test.go)
// already performs exactly this three-part check and is called by every
// rejection sub-test above (S-ATC-005, -006, -011); this test is the
// single consolidated sweep tying the claim to one place, per kind.
func TestToolCallEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		what      string
		construct func() (ai.Event, error)
		rule      error
		at        string
	}{
		{"start: empty id", func() (ai.Event, error) { return ai.NewToolCallStart(1, "", "n") }, ai.ErrEmpty, "id"},
		{"start: empty name", func() (ai.Event, error) { return ai.NewToolCallStart(1, "i", "") }, ai.ErrEmpty, "name"},
		{"start: block 0", func() (ai.Event, error) { return ai.NewToolCallStart(0, "i", "n") }, ai.ErrOutOfRange, "block_index"},
		{"delta: block 0", func() (ai.Event, error) { return ai.NewToolCallDelta(0, []byte("f")) }, ai.ErrOutOfRange, "block_index"},
		{"end: block 0", func() (ai.Event, error) { return ai.NewToolCallEnd(0, []byte("{}")) }, ai.ErrOutOfRange, "block_index"},
	}
	for _, tc := range cases {
		t.Run(tc.what, func(t *testing.T) {
			t.Parallel()
			err := mustFailToolCallEvent(t, tc.construct)
			assertViolation(t, err, tc.rule, tc.at)
		})
	}
}

// mustFailToolCallEvent runs construct, returns its error, and fails the
// test immediately if construction unexpectedly succeeded.
func mustFailToolCallEvent(t *testing.T, construct func() (ai.Event, error)) error {
	t.Helper()
	event, err := construct()
	if err == nil {
		t.Fatalf("construction succeeded with event %v, want a rejection", event)
	}
	return err
}

// splitInto divides data into n roughly equal, contiguous, non-empty-where-
// possible fragments, in order, such that concatenating them reproduces data
// exactly. It is test-local: Layer 1 ships no fragment splitter or
// accumulator (R-ATC-012's "test-local code" posture, restated for a test
// fixture rather than a reconstruction helper).
func splitInto(data []byte, n int) [][]byte {
	if n <= 0 || len(data) == 0 {
		return [][]byte{data}
	}
	size := (len(data) + n - 1) / n
	var out [][]byte
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}
		out = append(out, data[i:end])
	}
	return out
}

// readSourceFile reads a source file in this package's directory for a
// review-only text check (S-ATC-021, S-ATC-025).
func readSourceFile(t *testing.T, name string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// mustToolCallStart constructs a call-start event or fails the test.
func mustToolCallStart(t *testing.T, block ai.BlockIndex, id, name string) ai.Event {
	t.Helper()
	event, err := ai.NewToolCallStart(block, id, name)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart(%v, %q, %q) returned %v, want no failure", block, id, name, err)
	}
	return event
}

// mustToolCallDelta constructs a call-delta event or fails the test.
func mustToolCallDelta(t *testing.T, block ai.BlockIndex, fragment []byte) ai.Event {
	t.Helper()
	event, err := ai.NewToolCallDelta(block, fragment)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(%v, %q) returned %v, want no failure", block, fragment, err)
	}
	return event
}

// mustToolCallEnd constructs a call-end event or fails the test.
func mustToolCallEnd(t *testing.T, block ai.BlockIndex, arguments []byte) ai.Event {
	t.Helper()
	event, err := ai.NewToolCallEnd(block, arguments)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd(%v, %q) returned %v, want no failure", block, arguments, err)
	}
	return event
}
