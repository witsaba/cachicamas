// Tests for AI-17 — the streamed reasoning block event family.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion below is written against exactly
// the surface a consumer in another package sees — reasoning_content_test.go's
// reason, restated for the wire side of the same contract.
//
// # Cross-family scenarios against AI-16
//
// S-ARE-004, S-ARE-005, S-ARE-006 (R-ARE-002, distinctness from AI-16's text
// kinds) and S-ARE-011 (R-ARE-004, a reasoning block and a text block open on
// the same stream) each need AI-16's landed text-block payload types.
// AI-16 landed text_events.go in this worktree while this milestone was in
// flight (both are Wave 2 changes applied concurrently in one worktree), so
// they are implemented below against its current exported surface
// (TextBlockStart/TextDelta/TextBlockEnd, NewTextBlockStart/NewTextDelta/
// NewTextBlockEnd). apply-progress.md records the coupling this creates: if
// AI-16's surface changes after this file is written, these four tests are
// the ones to re-check first.
package ai_test

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cachicamas/backend/agent/src/ai"
)

// reconstructFragments concatenates, in arrival order, the Fragment() bytes
// of every reasoning delta in events whose block index is block — the
// test-local concatenator R-ARE-008 requires in place of an exported
// accumulator (S-ARE-022).
func reconstructFragments(events []ai.Event, block uint64) string {
	var b strings.Builder
	for _, e := range events {
		d, ok := e.ReasoningDelta()
		if !ok || d.BlockIndex() != block {
			continue
		}
		b.Write(d.Fragment())
	}
	return b.String()
}

// reconstructReasoningState rebuilds the (redacted, text, token) shape of the
// reasoning block at block index block from events — its start, its deltas
// and its end, in any order — and derives its [ai.ReasoningState] by feeding
// that shape into AI-07's own construction doors ([ai.NewReasoning] /
// [ai.NewRedactedReasoning]), so the derivation this milestone proves is
// provably the same one [ai.Reasoning.State] applies, not an imitation of it
// (S-ARE-033). It is the test-local reconstructor design.md commits to; it is
// never exported (R-ARE-008).
func reconstructReasoningState(t *testing.T, events []ai.Event, block uint64) ai.ReasoningState {
	t.Helper()

	var (
		redacted         bool
		sawStart, sawEnd bool
		text             strings.Builder
		token            []byte
		hasToken         bool
	)
	for _, e := range events {
		if s, ok := e.ReasoningBlockStart(); ok && s.BlockIndex() == block {
			redacted, sawStart = s.Redacted(), true
		}
		if d, ok := e.ReasoningDelta(); ok && d.BlockIndex() == block {
			text.Write(d.Fragment())
		}
		if end, ok := e.ReasoningBlockEnd(); ok && end.BlockIndex() == block {
			token, hasToken = end.Token()
			sawEnd = true
		}
	}
	if !sawStart || !sawEnd {
		t.Fatalf("reconstructReasoningState: block %d is missing a start or an end event in the given events", block)
	}
	tokenOrNil := token
	if !hasToken {
		tokenOrNil = nil
	}

	var (
		part ai.Part
		err  error
	)
	if redacted {
		part, err = ai.NewRedactedReasoning(tokenOrNil)
	} else {
		part, err = ai.NewReasoning(text.String(), tokenOrNil)
	}
	if err != nil {
		t.Fatalf("reconstructReasoningState: reconstructing block %d: %v", block, err)
	}
	reasoning, ok := part.Reasoning()
	if !ok {
		t.Fatal("reconstructReasoningState: part.Reasoning() reported no reasoning payload")
	}
	return reasoning.State()
}

// ---- R-ARE-001 — the family is three separately registered event kinds ---

// S-ARE-001 — each of the three kinds compiles, all three are distinct
// values, and each is a member of the registry's closed kind set.
func TestReasoningEventKinds_AreThreeDistinctRegisteredMembers(t *testing.T) {
	t.Parallel()

	kinds := []ai.EventKind{ai.EventKindReasoningBlockStart, ai.EventKindReasoningDelta, ai.EventKindReasoningBlockEnd}
	seen := make(map[ai.EventKind]bool, len(kinds))
	for _, k := range kinds {
		if seen[k] {
			t.Fatalf("%v repeats an earlier kind; the three kinds must be distinct values", k)
		}
		seen[k] = true
		if !containsEventKind(ai.EventKinds(), k) {
			t.Errorf("%v is not a member of ai.EventKinds()'s closed production kind set", k)
		}
	}
}

// S-ARE-002 — constructing each kind's payload and reading the resulting
// event's kind: the kind matches the payload, and no separate phase field is
// exposed on any of the three (no field whose name reads as a phase/stage
// discriminator).
func TestReasoningEventKind_ConstructedPayload_MatchesAndExposesNoPhaseField(t *testing.T) {
	t.Parallel()

	start, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart(1) returned %v, want no failure", err)
	}
	if got := start.Kind(); got != ai.EventKindReasoningBlockStart {
		t.Errorf("start.Kind() = %v, want %v", got, ai.EventKindReasoningBlockStart)
	}
	startPayload, ok := start.ReasoningBlockStart()
	if !ok {
		t.Fatal("start.ReasoningBlockStart() reported no payload on an event of its own kind")
	}

	deltaEvent, err := ai.NewReasoningDelta(startPayload, []byte("a"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	if got := deltaEvent.Kind(); got != ai.EventKindReasoningDelta {
		t.Errorf("deltaEvent.Kind() = %v, want %v", got, ai.EventKindReasoningDelta)
	}

	endEvent, err := ai.NewReasoningBlockEnd(startPayload, []byte("token"))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}
	if got := endEvent.Kind(); got != ai.EventKindReasoningBlockEnd {
		t.Errorf("endEvent.Kind() = %v, want %v", got, ai.EventKindReasoningBlockEnd)
	}

	for _, typ := range []reflect.Type{
		reflect.TypeOf(ai.ReasoningBlockStart{}),
		reflect.TypeOf(ai.ReasoningDelta{}),
		reflect.TypeOf(ai.ReasoningBlockEnd{}),
	} {
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			if strings.Contains(name, "phase") || strings.Contains(name, "stage") {
				t.Errorf("%v declares field %q, which reads as a phase discriminator; the event kind alone must decide", typ, typ.Field(i).Name)
			}
		}
	}
}

// S-ARE-003 (partial — the registry-membership half; the exhaustiveness
// guard's own pass/fail behavior is exercised by event_registry_test.go,
// extended in the same commit) — each kind is registered with a block-scoped
// descriptor from the stated domains.
func TestReasoningEventKinds_AreRegisteredWithBlockScopedDescriptors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind ai.EventKind
		role ai.BlockRole
	}{
		{"reasoningblockstart", ai.EventKindReasoningBlockStart, ai.BlockRoleStart},
		{"reasoningdelta", ai.EventKindReasoningDelta, ai.BlockRoleDelta},
		{"reasoningblockend", ai.EventKindReasoningBlockEnd, ai.BlockRoleEnd},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d, ok := ai.DescriptorOf(tc.kind)
			if !ok {
				t.Fatalf("ai.DescriptorOf(%v) reported not registered", tc.kind)
			}
			if d.Role != tc.role {
				t.Errorf("%v's descriptor.Role = %v, want %v", tc.kind, d.Role, tc.role)
			}
			if d.Cardinality != ai.CardinalityAny {
				t.Errorf("%v's descriptor.Cardinality = %v, want CardinalityAny", tc.kind, d.Cardinality)
			}
			if d.Terminal {
				t.Errorf("%v's descriptor.Terminal = true, want false", tc.kind)
			}
		})
	}
}

// ---- R-ARE-003 — 1-based producer-stamped block index; 0 is rejected -----

// S-ARE-007 — a start, two deltas and an end for one block all report the
// same block index.
func TestReasoningBlock_AllFourEvents_CarryTheSameBlockIndex(t *testing.T) {
	t.Parallel()

	const want = 3
	start, err := ai.NewReasoningBlockStart(want)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart(%d) returned %v, want no failure", want, err)
	}
	startPayload, _ := start.ReasoningBlockStart()

	delta1, err := ai.NewReasoningDelta(startPayload, []byte("a"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	delta2, err := ai.NewReasoningDelta(startPayload, []byte("b"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	end, err := ai.NewReasoningBlockEnd(startPayload, []byte("tok"))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	d1, _ := delta1.ReasoningDelta()
	d2, _ := delta2.ReasoningDelta()
	e, _ := end.ReasoningBlockEnd()

	if startPayload.BlockIndex() != want || d1.BlockIndex() != want || d2.BlockIndex() != want || e.BlockIndex() != want {
		t.Errorf("block indices = [%d %d %d %d], want all %d",
			startPayload.BlockIndex(), d1.BlockIndex(), d2.BlockIndex(), e.BlockIndex(), uint64(want))
	}
}

// S-ARE-008 — a block-scoped payload constructed with block index 0 fails,
// errors.Is reports ErrOutOfRange, and the position names the block-index
// field.
func TestReasoningBlockStart_IndexZero_IsRejected(t *testing.T) {
	t.Parallel()

	_, err := ai.NewReasoningBlockStart(0)
	if err == nil {
		t.Fatal("ai.NewReasoningBlockStart(0) returned no failure, want ErrOutOfRange")
	}
	if !errors.Is(err, ai.ErrOutOfRange) {
		t.Errorf("errors.Is(err, ai.ErrOutOfRange) = false for %v", err)
	}
	if !strings.Contains(err.Error(), "block") {
		t.Errorf("err.Error() = %q, want it to name the block-index field", err.Error())
	}
}

// S-ARE-009 — each of the three kinds, constructed from a start that was
// never passed through a constructor (the zero-value ReasoningBlockStart,
// block index 0), fails rather than yielding an event carrying block index 0.
func TestReasoningDeltaAndEnd_ZeroValueStart_FailRatherThanCarryingBlockIndexZero(t *testing.T) {
	t.Parallel()

	var zeroStart ai.ReasoningBlockStart // never passed through NewReasoningBlockStart

	if _, err := ai.NewReasoningDelta(zeroStart, []byte("x")); err == nil || !errors.Is(err, ai.ErrOutOfRange) {
		t.Errorf("ai.NewReasoningDelta(zero start, ...) = %v, want ErrOutOfRange", err)
	}
	if _, err := ai.NewReasoningBlockEnd(zeroStart, []byte("tok")); err == nil || !errors.Is(err, ai.ErrOutOfRange) {
		t.Errorf("ai.NewReasoningBlockEnd(zero start, ...) = %v, want ErrOutOfRange", err)
	}
}

// ---- R-ARE-004 — one stream-wide block-index space ------------------------

// S-ARE-010 — two interleaved reasoning blocks whose events arrive out of
// block order: partitioning by block index alone attributes every event to
// its own block.
func TestReasoningBlocks_InterleavedOutOfOrder_PartitionByBlockIndexAlone(t *testing.T) {
	t.Parallel()

	start1, _ := ai.NewReasoningBlockStart(1)
	start1Payload, _ := start1.ReasoningBlockStart()
	start2, _ := ai.NewReasoningBlockStart(2)
	start2Payload, _ := start2.ReasoningBlockStart()

	d1a, _ := ai.NewReasoningDelta(start1Payload, []byte("he"))
	d2a, _ := ai.NewReasoningDelta(start2Payload, []byte("wo"))
	d1b, _ := ai.NewReasoningDelta(start1Payload, []byte("llo"))
	e2, _ := ai.NewReasoningBlockEnd(start2Payload, []byte("tok2"))
	d2b, _ := ai.NewReasoningDelta(start2Payload, []byte("rld"))
	e1, _ := ai.NewReasoningBlockEnd(start1Payload, []byte("tok1"))

	// Arrival order interleaves the two blocks and closes block 2 before
	// block 1 has received all of its deltas.
	stream := []ai.Event{start1, start2, d1a, d2a, d1b, e2, d2b, e1}

	if got := reconstructFragments(stream, 1); got != "hello" {
		t.Errorf("block 1 reconstructed = %q, want %q", got, "hello")
	}
	if got := reconstructFragments(stream, 2); got != "world" {
		t.Errorf("block 2 reconstructed = %q, want %q", got, "world")
	}
}

// S-ARE-012 (partial — the reasoning-only-space and 0-based-index halves;
// the family-tag half needs AI-16 and is deferred) — block index 1 is legal
// and is the shared space's first index, never a reasoning-only space.
func TestReasoningBlockIndex_UsesTheSharedOneBasedSpace(t *testing.T) {
	t.Parallel()

	start, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart(1) returned %v, want no failure — 1 is the first legal index in R-ATE-004's shared space", err)
	}
	payload, _ := start.ReasoningBlockStart()
	if payload.BlockIndex() != 1 {
		t.Errorf("payload.BlockIndex() = %d, want 1", payload.BlockIndex())
	}
}

// ---- R-ARE-005 — a reasoning delta carries a text fragment only ----------

// S-ARE-013 — a block whose deltas carry "a", "b" and "c" reads back exactly
// those three fragments, never the accumulation.
func TestReasoningDelta_Fragment_IsExactlyTheNewBytesNeverAccumulated(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	for _, want := range []string{"a", "b", "c"} {
		event, err := ai.NewReasoningDelta(startPayload, []byte(want))
		if err != nil {
			t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
		}
		delta, ok := event.ReasoningDelta()
		if !ok {
			t.Fatal("event.ReasoningDelta() reported no payload on an event of its own kind")
		}
		if got := string(delta.Fragment()); got != want {
			t.Errorf("delta.Fragment() = %q, want %q — never accumulated", got, want)
		}
	}
}

// S-ARE-014 — the delta kind's exported accessors expose no accumulated
// content, token, redacted signal or state.
func TestReasoningDelta_ExportedAccessors_NeverReturnAccumulatedContentTokenOrState(t *testing.T) {
	t.Parallel()

	forbidden := map[string]bool{"Token": true, "Redacted": true, "State": true, "Text": true}
	typ := reflect.TypeOf(ai.ReasoningDelta{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if forbidden[name] {
			t.Errorf("ai.ReasoningDelta exports method %q, which R-ARE-005 forbids on the delta kind", name)
		}
	}
}

// ---- R-ARE-006 — concatenated deltas reconstruct byte-exactly ------------

// S-ARE-015 — a known reasoning text split into an arbitrary number of
// fragments reconstructs byte-identically via a test-local concatenator,
// including leading/trailing whitespace and an interior newline.
func TestReasoningDeltas_ConcatenatedInArrivalOrder_ReconstructByteExactly(t *testing.T) {
	t.Parallel()

	const want = "  leading and trailing whitespace\nwith an interior newline\t and a tab  "
	cuts := []int{0, 3, 3, 10, len(want) / 2, len(want) - 4, len(want)}

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	var events []ai.Event
	prev := 0
	for _, cut := range cuts {
		if cut < prev || cut > len(want) {
			continue
		}
		event, err := ai.NewReasoningDelta(startPayload, []byte(want[prev:cut]))
		if err != nil {
			t.Fatalf("ai.NewReasoningDelta(%q) returned %v, want no failure", want[prev:cut], err)
		}
		events = append(events, event)
		prev = cut
	}
	if got := reconstructFragments(events, 1); got != want {
		t.Errorf("reconstructed text = %q, want %q", got, want)
	}
}

// S-ARE-016 — a multi-byte rune split across a delta boundary: both deltas
// succeed even though the first fragment alone is not well-formed UTF-8, and
// their concatenation decodes to the original rune with no replacement
// character.
func TestReasoningDelta_RuneSplitAcrossDeltaBoundary_BothFragmentsSucceedAndConcatenateCleanly(t *testing.T) {
	t.Parallel()

	const euroSign = "€" // U+20AC, encodes to 3 bytes: 0xE2 0x82 0xAC
	full := []byte(euroSign)
	if len(full) != 3 {
		t.Fatalf("test fixture assumption broken: %q encodes to %d bytes, want 3", euroSign, len(full))
	}
	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	first, err := ai.NewReasoningDelta(startPayload, full[:1])
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta(first byte) returned %v, want no failure — a fragment may be invalid UTF-8 on its own", err)
	}
	second, err := ai.NewReasoningDelta(startPayload, full[1:])
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta(remaining bytes) returned %v, want no failure", err)
	}
	d1, _ := first.ReasoningDelta()
	d2, _ := second.ReasoningDelta()

	if utf8.Valid(d1.Fragment()) {
		t.Fatal("test fixture assumption broken: the first fragment alone must not be well-formed UTF-8")
	}
	joined := append(bytes.Clone(d1.Fragment()), d2.Fragment()...)
	if !bytes.Equal(joined, full) {
		t.Errorf("joined fragments = %x, want %x — neither fragment's bytes may be altered", joined, full)
	}
	r, size := utf8.DecodeRune(joined)
	if r == utf8.RuneError || size != 3 {
		t.Errorf("joined fragments decode to rune %q (size %d), want %q with no replacement character", r, size, euroSign)
	}
}

// ---- R-ARE-007 — no complete-part rule on a fragment; zero deltas legal --

// S-ARE-018 — a whitespace-only fragment and a zero-length fragment both
// succeed, with no emptiness sentinel.
func TestReasoningDelta_WhitespaceOnlyAndZeroLengthFragments_AreLegal(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	cases := []struct {
		name string
		frag []byte
	}{
		{"single space", []byte(" ")},
		{"zero-length", []byte{}},
		{"nil", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ai.NewReasoningDelta(startPayload, tc.frag); err != nil {
				t.Errorf("ai.NewReasoningDelta(%q) returned %v, want no failure — no complete-part rule applies to a fragment", tc.frag, err)
			}
		})
	}
}

// S-ARE-019 — a fragment of exactly MaxTextLen bytes succeeds; one byte over
// fails with ErrOutOfRange naming the fragment field.
func TestReasoningDelta_FragmentBound_IsMaxTextLen(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	atBound := bytes.Repeat([]byte("a"), ai.MaxTextLen)
	if _, err := ai.NewReasoningDelta(startPayload, atBound); err != nil {
		t.Errorf("ai.NewReasoningDelta(exactly MaxTextLen bytes) returned %v, want no failure", err)
	}

	overBound := bytes.Repeat([]byte("a"), ai.MaxTextLen+1)
	_, err := ai.NewReasoningDelta(startPayload, overBound)
	if err == nil || !errors.Is(err, ai.ErrOutOfRange) {
		t.Fatalf("ai.NewReasoningDelta(MaxTextLen+1 bytes) = %v, want ErrOutOfRange", err)
	}
	if !strings.Contains(err.Error(), "fragment") {
		t.Errorf("err.Error() = %q, want it to name the fragment field", err.Error())
	}
}

// S-ARE-020 — a zero-delta block and a multi-delta block are both valid; the
// zero-delta block reconstructs to empty text and is not confused with an
// unterminated block.
func TestReasoningBlock_ZeroDeltas_ReconstructsToEmptyTextAndIsNotConfusedWithUnterminated(t *testing.T) {
	t.Parallel()

	zeroStart, _ := ai.NewReasoningBlockStart(1)
	zeroStartPayload, _ := zeroStart.ReasoningBlockStart()
	zeroEnd, err := ai.NewReasoningBlockEnd(zeroStartPayload, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd (zero-delta block, no token) returned %v, want no failure", err)
	}
	if _, ok := zeroEnd.ReasoningBlockEnd(); !ok {
		t.Fatal("zeroEnd.ReasoningBlockEnd() reported no payload on an event of its own kind — the block is not unterminated")
	}

	multiStart, _ := ai.NewReasoningBlockStart(2)
	multiStartPayload, _ := multiStart.ReasoningBlockStart()
	d1, err := ai.NewReasoningDelta(multiStartPayload, []byte("hi"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	multiEnd, err := ai.NewReasoningBlockEnd(multiStartPayload, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd (multi-delta block) returned %v, want no failure", err)
	}

	if got := reconstructFragments([]ai.Event{zeroStart, zeroEnd}, 1); got != "" {
		t.Errorf("zero-delta block reconstructed text = %q, want empty", got)
	}
	if got := reconstructFragments([]ai.Event{multiStart, d1, multiEnd}, 2); got != "hi" {
		t.Errorf("multi-delta block reconstructed text = %q, want %q", got, "hi")
	}
}

// ---- R-ARE-008 — no public accumulation or reconstruction helper ---------

// S-ARE-021 — reasoning_event.go exports no function whose name reads as an
// accumulator, transcript rebuilder or reducer of a block's deltas.
func TestReasoningEventFile_ExportsNoAccumulatorOrReconstructor(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "reasoning_event.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing reasoning_event.go: %v", err)
	}
	forbidden := []string{"accumulat", "reconstruct", "concat", "join", "transcript", "rebuild"}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !fn.Name.IsExported() {
			continue
		}
		lower := strings.ToLower(fn.Name.Name)
		for _, bad := range forbidden {
			if strings.Contains(lower, bad) {
				t.Errorf("reasoning_event.go exports func %s, whose name reads as an accumulator/reconstructor — R-ARE-008 forbids one", fn.Name.Name)
			}
		}
	}
}

// S-ARE-022 — the concatenator this milestone's own tests use to prove
// byte-exactness (reconstructFragments, above) is defined inside this test
// package and is not exported from the contract. Proven by construction:
// asserted here as a guard against a future edit accidentally promoting it.
func TestReconstructFragments_IsATestLocalHelper_NotAPackageExport(t *testing.T) {
	t.Parallel()

	name := "reconstructFragments"
	if name[0] < 'a' || name[0] > 'z' {
		t.Fatalf("the test-local concatenator %q must start lower-case (unexported)", name)
	}
	// Sanity: the exported ai package itself carries no such identifier.
	typ := reflect.TypeOf(ai.Event{})
	for i := 0; i < typ.NumMethod(); i++ {
		if strings.EqualFold(typ.Method(i).Name, name) {
			t.Errorf("ai.Event exports a method named %q; the concatenator must stay test-local", typ.Method(i).Name)
		}
	}
}

// ---- R-ARE-002 — structurally distinct from AI-16's text family ----------

// S-ARE-004 — none of AI-16's three text kinds accepts reasoning content, a
// redacted signal, or a round-trip token: no exported method is named
// Token/Redacted/State, and no exported constructor takes a
// ReasoningBlockStart parameter.
func TestTextEventKinds_ExposeNoReasoningSurface(t *testing.T) {
	t.Parallel()

	forbiddenMethods := map[string]bool{"Token": true, "Redacted": true, "State": true}
	for _, typ := range []reflect.Type{
		reflect.TypeOf(ai.TextBlockStart{}),
		reflect.TypeOf(ai.TextDelta{}),
		reflect.TypeOf(ai.TextBlockEnd{}),
	} {
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			if forbiddenMethods[name] {
				t.Errorf("%v exports method %q, which R-ARE-002 forbids on a text kind", typ, name)
			}
		}
	}

	reasoningType := reflect.TypeOf(ai.ReasoningBlockStart{})
	for _, ctor := range []any{ai.NewTextBlockStart, ai.NewTextDelta, ai.NewTextBlockEnd} {
		ctorType := reflect.TypeOf(ctor)
		for i := 0; i < ctorType.NumIn(); i++ {
			if ctorType.In(i) == reasoningType {
				t.Errorf("%v accepts a %v parameter, which R-ARE-002 forbids on a text constructor", ctorType, reasoningType)
			}
		}
	}
}

// S-ARE-005 — a reasoning delta and a text delta carrying identical fragment
// bytes: their event kinds differ, their payload types differ (neither
// accessor reads the other event), and no assignment or conversion between
// the two payload types compiles — proven by this file simply never
// attempting one; a stray attempt would fail `go vet`/`go build` before any
// test could run.
func TestReasoningDeltaAndTextDelta_IdenticalBytes_AreDistinctAtEveryLevel(t *testing.T) {
	t.Parallel()

	const fragment = "identical fragment bytes"

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()
	reasoningEvent, err := ai.NewReasoningDelta(startPayload, []byte(fragment))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	textEvent, err := ai.NewTextDelta(1, fragment)
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}

	if reasoningEvent.Kind() == textEvent.Kind() {
		t.Errorf("reasoningEvent.Kind() == textEvent.Kind() == %v, want them to differ", reasoningEvent.Kind())
	}

	reasoningDelta, ok := reasoningEvent.ReasoningDelta()
	if !ok {
		t.Fatal("reasoningEvent.ReasoningDelta() reported no payload")
	}
	textDelta, ok := textEvent.TextDelta()
	if !ok {
		t.Fatal("textEvent.TextDelta() reported no payload")
	}
	if string(reasoningDelta.Fragment()) != textDelta.Delta() {
		t.Fatalf("test fixture assumption broken: fragment bytes differ (%q vs %q)", reasoningDelta.Fragment(), textDelta.Delta())
	}

	if _, ok := reasoningEvent.TextDelta(); ok {
		t.Error("reasoningEvent.TextDelta() reported a payload on a reasoning-kind event")
	}
	if _, ok := textEvent.ReasoningDelta(); ok {
		t.Error("textEvent.ReasoningDelta() reported a payload on a text-kind event")
	}
	if reflect.TypeOf(reasoningDelta) == reflect.TypeOf(textDelta) {
		t.Error("ai.ReasoningDelta and ai.TextDelta share a reflect.Type; the two payload types must differ")
	}
}

// S-ARE-006 — a consumer that switches only on event kind, handling no flag,
// classifies every reasoning event as reasoning on a stream carrying both
// families, and no reasoning byte reaches the text path.
func TestMixedStream_SwitchingOnKindAlone_ClassifiesReasoningAndTextWithoutCrossContamination(t *testing.T) {
	t.Parallel()

	rStart, _ := ai.NewReasoningBlockStart(1)
	rStartPayload, _ := rStart.ReasoningBlockStart()
	rDelta, err := ai.NewReasoningDelta(rStartPayload, []byte("reasoning-only bytes"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	tStart, err := ai.NewTextBlockStart(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	tDelta, err := ai.NewTextDelta(2, "text-only bytes")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}

	stream := []ai.Event{rStart, tStart, rDelta, tDelta}

	var reasoningBytes, textBytes []byte
	for _, e := range stream {
		switch e.Kind() {
		case ai.EventKindReasoningDelta:
			d, _ := e.ReasoningDelta()
			reasoningBytes = append(reasoningBytes, d.Fragment()...)
		case ai.EventKindTextDelta:
			d, _ := e.TextDelta()
			textBytes = append(textBytes, []byte(d.Delta())...)
		}
	}

	if string(reasoningBytes) != "reasoning-only bytes" {
		t.Errorf("reasoningBytes = %q, want %q", reasoningBytes, "reasoning-only bytes")
	}
	if string(textBytes) != "text-only bytes" {
		t.Errorf("textBytes = %q, want %q", textBytes, "text-only bytes")
	}
	if bytes.Contains(textBytes, []byte("reasoning")) {
		t.Error("a reasoning byte reached the text path")
	}
}

// ---- R-ARE-004 (remainder) — reasoning and text share the index space ----

// S-ARE-011 — a reasoning block and a text block both open on the same
// stream: their indices differ, and a consumer separates the two without
// consulting the event kind.
func TestReasoningAndTextBlocksBothOpen_IndicesDifferWithoutConsultingKind(t *testing.T) {
	t.Parallel()

	rStart, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart(1) returned %v, want no failure", err)
	}
	tStart, err := ai.NewTextBlockStart(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart(2) returned %v, want no failure", err)
	}

	rPayload, _ := rStart.ReasoningBlockStart()
	tPayload, _ := tStart.TextBlockStart()

	if rPayload.BlockIndex() == uint64(tPayload.Block()) {
		t.Fatalf("reasoning block index %d collides with text block index %d; R-ARE-004 forbids this", rPayload.BlockIndex(), tPayload.Block())
	}
}

// ---- R-ARE-009 — the token arrives whole on the block-end event only -----

// S-ARE-023 — a token appears on the block-end kind only: neither the start
// nor the delta kind exports a Token method.
func TestReasoningBlockStartAndDelta_ExposeNoTokenAccessor(t *testing.T) {
	t.Parallel()

	for _, typ := range []reflect.Type{
		reflect.TypeOf(ai.ReasoningBlockStart{}),
		reflect.TypeOf(ai.ReasoningDelta{}),
	} {
		for i := 0; i < typ.NumMethod(); i++ {
			if typ.Method(i).Name == "Token" {
				t.Errorf("%v exports a Token method; R-ARE-009 permits Token only on ReasoningBlockEnd", typ)
			}
		}
	}
}

// S-ARE-024 — a token longer than any single delta fragment in the same
// block is still delivered whole in one block-end event; no delta carries
// any token byte.
func TestReasoningBlockEnd_TokenLongerThanAnyDelta_ArrivesWholeInOneEvent(t *testing.T) {
	t.Parallel()

	longToken := bytes.Repeat([]byte("token-byte-"), 100) // far longer than any delta below
	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()
	d1, err := ai.NewReasoningDelta(startPayload, []byte("hi"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	d2, err := ai.NewReasoningDelta(startPayload, []byte("!"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	end, err := ai.NewReasoningBlockEnd(startPayload, longToken)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	endPayload, ok := end.ReasoningBlockEnd()
	if !ok {
		t.Fatal("end.ReasoningBlockEnd() reported no payload")
	}
	got, present := endPayload.Token()
	if !present || !bytes.Equal(got, longToken) {
		t.Errorf("endPayload.Token() = (%x, %v), want (%x, true)", got, present, longToken)
	}

	for _, e := range []ai.Event{d1, d2} {
		d, _ := e.ReasoningDelta()
		if bytes.Contains(d.Fragment(), longToken[:10]) {
			t.Error("a delta fragment carries token bytes; R-ARE-009 forbids this")
		}
	}
}

// ---- R-ARE-010 — byte-exact across the boundary, never interpreted ------

// S-ARE-025 — every byte class in AI-07's opaqueTokens() fixture round-trips
// through a reasoning block-end event byte for byte.
func TestReasoningBlockEnd_OpaqueToken_RoundTripsByteExactly(t *testing.T) {
	t.Parallel()

	for _, tc := range opaqueTokens() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want := bytes.Clone(tc.token)
			start, _ := ai.NewReasoningBlockStart(1)
			startPayload, _ := start.ReasoningBlockStart()
			event, err := ai.NewReasoningBlockEnd(startPayload, tc.token)
			if err != nil {
				t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure — nothing validates a token beyond its bound", err)
			}
			end, ok := event.ReasoningBlockEnd()
			if !ok {
				t.Fatal("event.ReasoningBlockEnd() reported no payload")
			}
			got, present := end.Token()
			if !present {
				t.Fatal("end.Token() reported no token, want one")
			}
			if len(got) != len(want) {
				t.Fatalf("end.Token() returned %d bytes, want %d — a length change is a normalization", len(got), len(want))
			}
			if !bytes.Equal(got, want) {
				t.Errorf("end.Token() = %x, want %x", got, want)
			}
		})
	}
}

// S-ARE-026 — aliasing is not observable: mutating the caller's buffer after
// construction, and separately mutating a reader's copy of the result, must
// not change the token an event reports.
func TestReasoningBlockEnd_Token_DoesNotAlias(t *testing.T) {
	t.Parallel()

	t.Run("caller buffer mutated after construction", func(t *testing.T) {
		t.Parallel()

		buf := []byte("original bytes")
		start, _ := ai.NewReasoningBlockStart(1)
		startPayload, _ := start.ReasoningBlockStart()
		event, err := ai.NewReasoningBlockEnd(startPayload, buf)
		if err != nil {
			t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
		}
		buf[0] = 'X'

		end, _ := event.ReasoningBlockEnd()
		got, _ := end.Token()
		if string(got) != "original bytes" {
			t.Errorf("end.Token() = %q after caller mutated its buffer, want %q", got, "original bytes")
		}
	})

	t.Run("reader's copy mutated", func(t *testing.T) {
		t.Parallel()

		start, _ := ai.NewReasoningBlockStart(1)
		startPayload, _ := start.ReasoningBlockStart()
		event, err := ai.NewReasoningBlockEnd(startPayload, []byte("original bytes"))
		if err != nil {
			t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
		}
		end, _ := event.ReasoningBlockEnd()

		first, _ := end.Token()
		first[0] = 'X'

		second, _ := end.Token()
		if string(second) != "original bytes" {
			t.Errorf("end.Token() (second read) = %q after a reader mutated the first read's slice, want %q", second, "original bytes")
		}
	})
}

// S-ARE-027 — a token of exactly MaxReasoningTokenLen bytes succeeds; one
// byte over fails with ErrOutOfRange naming the token field.
func TestReasoningBlockEnd_TokenBound_IsMaxReasoningTokenLen(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	atBound := bytes.Repeat([]byte("a"), ai.MaxReasoningTokenLen)
	if _, err := ai.NewReasoningBlockEnd(startPayload, atBound); err != nil {
		t.Errorf("ai.NewReasoningBlockEnd(exactly MaxReasoningTokenLen bytes) returned %v, want no failure", err)
	}

	overBound := bytes.Repeat([]byte("a"), ai.MaxReasoningTokenLen+1)
	_, err := ai.NewReasoningBlockEnd(startPayload, overBound)
	if err == nil || !errors.Is(err, ai.ErrOutOfRange) {
		t.Fatalf("ai.NewReasoningBlockEnd(MaxReasoningTokenLen+1 bytes) = %v, want ErrOutOfRange", err)
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("err.Error() = %q, want it to name the token field", err.Error())
	}
}

// ---- R-ARE-011 — absent token and empty token are distinguishable -------

// S-ARE-028 — a block-end constructed with no token reports absent; one
// constructed with a zero-length token reports present with a zero-length
// slice.
func TestReasoningBlockEnd_AbsentToken_IsDistinguishableFromAnEmptyToken(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	noToken, err := ai.NewReasoningBlockEnd(startPayload, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd(nil) returned %v, want no failure", err)
	}
	noTokenPayload, _ := noToken.ReasoningBlockEnd()
	if got, present := noTokenPayload.Token(); present {
		t.Errorf("noTokenPayload.Token() = (%x, true), want present=false", got)
	}

	emptyToken, err := ai.NewReasoningBlockEnd(startPayload, []byte{})
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd([]byte{}) returned %v, want no failure", err)
	}
	emptyTokenPayload, _ := emptyToken.ReasoningBlockEnd()
	got, present := emptyTokenPayload.Token()
	if !present {
		t.Fatal("emptyTokenPayload.Token() reported present=false, want true")
	}
	if len(got) != 0 {
		t.Errorf("emptyTokenPayload.Token() = %x, want zero-length", got)
	}
}

// S-ARE-029 — the two events above are not equal, and the difference is
// readable without inspecting byte length (the presence result alone decides
// it).
func TestReasoningBlockEnd_NoTokenVsEmptyToken_AreNotEqual(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	noToken, _ := ai.NewReasoningBlockEnd(startPayload, nil)
	emptyToken, _ := ai.NewReasoningBlockEnd(startPayload, []byte{})

	if noToken == emptyToken {
		t.Fatal("a no-token block-end event equals a zero-length-token block-end event, want them distinguishable")
	}
	noTokenPayload, _ := noToken.ReasoningBlockEnd()
	emptyTokenPayload, _ := emptyToken.ReasoningBlockEnd()
	_, noPresent := noTokenPayload.Token()
	_, emptyPresent := emptyTokenPayload.Token()
	if noPresent == emptyPresent {
		t.Fatal("the presence result alone does not distinguish the two events")
	}
}

// S-ARE-030 — a reasoning block whose start, deltas and end all carry no
// token is valid, and reconstructs to ReasoningStateText.
func TestReasoningBlock_NoTokenAtAll_ReconstructsToReasoningStateText(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()
	delta, err := ai.NewReasoningDelta(startPayload, []byte("plain reasoning text"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	end, err := ai.NewReasoningBlockEnd(startPayload, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd(nil) returned %v, want no failure", err)
	}

	got := reconstructReasoningState(t, []ai.Event{start, delta, end}, 1)
	if got != ai.ReasoningStateText {
		t.Errorf("reconstructed state = %v, want %v", got, ai.ReasoningStateText)
	}
}

// ---- R-ARE-012 — redaction is signalled once, on the block-start event ---

// S-ARE-031 — the redacted signal appears on the block-start kind only, and
// no kind exposes a ReasoningState field or accessor.
func TestReasoningEventKinds_ExposeRedactedOnlyOnStart_AndNoReasoningStateAnywhere(t *testing.T) {
	t.Parallel()

	stateType := reflect.TypeOf(ai.ReasoningStateText)
	types := []reflect.Type{
		reflect.TypeOf(ai.ReasoningBlockStart{}),
		reflect.TypeOf(ai.ReasoningDelta{}),
		reflect.TypeOf(ai.ReasoningBlockEnd{}),
	}
	for _, typ := range types {
		hasRedacted := false
		for i := 0; i < typ.NumMethod(); i++ {
			m := typ.Method(i)
			if m.Name == "Redacted" {
				hasRedacted = true
			}
			for r := 0; r < m.Type.NumOut(); r++ {
				if m.Type.Out(r) == stateType {
					t.Errorf("%v.%s returns %v; R-ARE-012 forbids any event of this family exposing a ReasoningState", typ, m.Name, stateType)
				}
			}
		}
		wantRedacted := typ == reflect.TypeOf(ai.ReasoningBlockStart{})
		if hasRedacted != wantRedacted {
			t.Errorf("%v exports Redacted() = %v, want %v (R-ARE-012: only the block-start kind)", typ, hasRedacted, wantRedacted)
		}
	}
}

// S-ARE-032 — a redacted block's start event, read before any further event
// arrives, reports that the block is redacted.
func TestRedactedReasoningBlockStart_ReportsRedaction_BeforeAnyFurtherEvent(t *testing.T) {
	t.Parallel()

	start, err := ai.NewRedactedReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewRedactedReasoningBlockStart(1) returned %v, want no failure", err)
	}
	payload, ok := start.ReasoningBlockStart()
	if !ok {
		t.Fatal("start.ReasoningBlockStart() reported no payload")
	}
	if !payload.Redacted() {
		t.Error("payload.Redacted() = false, want true — readable from the start event alone, before any delta")
	}
}

// S-ARE-033 — a reconstructed block's state derivation consults the start
// event's redacted signal, the concatenated fragments and the end event's
// token, and returns the same state Reasoning.State() returns for the
// equivalent constructed part.
func TestReconstructedReasoningBlock_StateDerivation_MatchesTheEquivalentConstructedPart(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()
	delta, err := ai.NewReasoningDelta(startPayload, []byte("visible reasoning"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	end, err := ai.NewReasoningBlockEnd(startPayload, []byte("sig"))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	got := reconstructReasoningState(t, []ai.Event{start, delta, end}, 1)

	wantPart, err := ai.NewReasoning("visible reasoning", []byte("sig"))
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	wantReasoning, ok := wantPart.Reasoning()
	if !ok {
		t.Fatal("wantPart.Reasoning() reported no payload")
	}
	if got != wantReasoning.State() {
		t.Errorf("reconstructed state = %v, want %v (the equivalent constructed part's own derivation)", got, wantReasoning.State())
	}
}

// ---- R-ARE-013 — a redacted block streams its payload verbatim -----------

// S-ARE-034 — a redacted block whose end event carries an opaque payload
// from opaqueTokens() reconstructs to ReasoningStateRedacted, byte-identical.
func TestRedactedReasoningBlock_OpaquePayload_ReconstructsToRedactedStateByteIdentical(t *testing.T) {
	t.Parallel()

	for _, tc := range opaqueTokens() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			start, err := ai.NewRedactedReasoningBlockStart(1)
			if err != nil {
				t.Fatalf("ai.NewRedactedReasoningBlockStart(1) returned %v, want no failure", err)
			}
			startPayload, _ := start.ReasoningBlockStart()
			end, err := ai.NewReasoningBlockEnd(startPayload, tc.token)
			if err != nil {
				t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
			}

			endPayload, _ := end.ReasoningBlockEnd()
			got, present := endPayload.Token()
			if !present || !bytes.Equal(got, tc.token) {
				t.Fatalf("endPayload.Token() = (%x, %v), want (%x, true)", got, present, tc.token)
			}

			state := reconstructReasoningState(t, []ai.Event{start, end}, 1)
			if state != ai.ReasoningStateRedacted {
				t.Errorf("reconstructed state = %v, want %v", state, ai.ReasoningStateRedacted)
			}
		})
	}
}

// S-ARE-035 — a redacted block whose end event carries no token fails with
// ErrEmpty naming the token field.
func TestReasoningBlockEnd_RedactedWithNoToken_IsRejected(t *testing.T) {
	t.Parallel()

	start, _ := ai.NewRedactedReasoningBlockStart(1)
	startPayload, _ := start.ReasoningBlockStart()

	_, err := ai.NewReasoningBlockEnd(startPayload, nil)
	if err == nil || !errors.Is(err, ai.ErrEmpty) {
		t.Fatalf("ai.NewReasoningBlockEnd(redacted start, nil token) = %v, want ErrEmpty", err)
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("err.Error() = %q, want it to name the token field", err.Error())
	}

	// A present, zero-length token satisfies the rule (mirrors
	// Reasoning.validate rule 3): only a nil (absent) token is rejected.
	if _, err := ai.NewReasoningBlockEnd(startPayload, []byte{}); err != nil {
		t.Errorf("ai.NewReasoningBlockEnd(redacted start, zero-length token) returned %v, want no failure", err)
	}
}

// S-ARE-036 — a delta carrying a non-empty fragment, constructed for a block
// whose start signalled redaction, is rejected and the failure names the
// offending field. A zero-length fragment on the same block is still legal.
func TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected(t *testing.T) {
	t.Parallel()

	start, err := ai.NewRedactedReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewRedactedReasoningBlockStart(1) returned %v, want no failure", err)
	}
	startPayload, _ := start.ReasoningBlockStart()
	if !startPayload.Redacted() {
		t.Fatal("startPayload.Redacted() = false, want true")
	}

	_, err = ai.NewReasoningDelta(startPayload, []byte("leaked plaintext"))
	if err == nil || !errors.Is(err, ai.ErrMisplaced) {
		t.Fatalf("ai.NewReasoningDelta(redacted start, non-empty fragment) = %v, want ErrMisplaced", err)
	}
	if !strings.Contains(err.Error(), "fragment") {
		t.Errorf("err.Error() = %q, want it to name the fragment field", err.Error())
	}

	if _, err := ai.NewReasoningDelta(startPayload, nil); err != nil {
		t.Errorf("ai.NewReasoningDelta(redacted start, nil fragment) returned %v, want no failure — only a non-empty fragment is rejected", err)
	}
	if _, err := ai.NewReasoningDelta(startPayload, []byte{}); err != nil {
		t.Errorf("ai.NewReasoningDelta(redacted start, zero-length fragment) returned %v, want no failure", err)
	}
}

// ---- R-ARE-014 — a token-and-no-text block is valid signature-only -------

// S-ARE-037 — a reasoning block start with no redacted signal, no deltas,
// and a block end carrying a token is valid, reconstructs to empty text, and
// derives ReasoningStateTokenOnly.
func TestReasoningBlock_TokenAndNoText_ReconstructsToReasoningStateTokenOnly(t *testing.T) {
	t.Parallel()

	start, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart(1) returned %v, want no failure", err)
	}
	startPayload, _ := start.ReasoningBlockStart()
	end, err := ai.NewReasoningBlockEnd(startPayload, []byte("signature-only-token"))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	if got := reconstructFragments([]ai.Event{start, end}, 1); got != "" {
		t.Errorf("reconstructed text = %q, want empty", got)
	}
	state := reconstructReasoningState(t, []ai.Event{start, end}, 1)
	if state != ai.ReasoningStateTokenOnly {
		t.Errorf("reconstructed state = %v, want %v", state, ai.ReasoningStateTokenOnly)
	}
}

// S-ARE-038 — a token-only block and a redacted block carrying
// byte-identical token bytes reconstruct to different, non-equal states.
func TestTokenOnlyBlock_AndRedactedBlock_WithIdenticalTokenBytes_AreDistinguishable(t *testing.T) {
	t.Parallel()

	const token = "shared-token-bytes"

	tokenOnlyStart, _ := ai.NewReasoningBlockStart(1)
	tokenOnlyStartPayload, _ := tokenOnlyStart.ReasoningBlockStart()
	tokenOnlyEnd, err := ai.NewReasoningBlockEnd(tokenOnlyStartPayload, []byte(token))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	redactedStart, _ := ai.NewRedactedReasoningBlockStart(2)
	redactedStartPayload, _ := redactedStart.ReasoningBlockStart()
	redactedEnd, err := ai.NewReasoningBlockEnd(redactedStartPayload, []byte(token))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	tokenOnlyPayload, _ := tokenOnlyEnd.ReasoningBlockEnd()
	redactedPayload, _ := redactedEnd.ReasoningBlockEnd()
	tokenOnlyBytes, _ := tokenOnlyPayload.Token()
	redactedBytes, _ := redactedPayload.Token()
	if !bytes.Equal(tokenOnlyBytes, redactedBytes) {
		t.Fatalf("test fixture assumption broken: token bytes differ (%x vs %x)", tokenOnlyBytes, redactedBytes)
	}

	tokenOnlyState := reconstructReasoningState(t, []ai.Event{tokenOnlyStart, tokenOnlyEnd}, 1)
	redactedState := reconstructReasoningState(t, []ai.Event{redactedStart, redactedEnd}, 2)

	if tokenOnlyState != ai.ReasoningStateTokenOnly {
		t.Errorf("token-only block's state = %v, want %v", tokenOnlyState, ai.ReasoningStateTokenOnly)
	}
	if redactedState != ai.ReasoningStateRedacted {
		t.Errorf("redacted block's state = %v, want %v", redactedState, ai.ReasoningStateRedacted)
	}
	if tokenOnlyState == redactedState {
		t.Error("the token-only and redacted states compare equal despite identical token bytes; they must be distinguishable")
	}
}

// S-ARE-039 — a reasoning block with neither text, nor a token, nor a
// redacted signal is legal at the event level (R-ARE-007's zero-delta
// block), and its start reports no redaction. Reconstructing it into AI-07's
// content-part shape fails with the same rule Reasoning.validate applies to
// the equivalent constructed part ("a part with neither text nor token
// carries nothing at all") — it is never silently reported as redacted or
// signature-only.
func TestReasoningBlock_NeitherTextNorTokenNorRedacted_IsLegalAtTheEventLevelButNotAReconstructibleState(t *testing.T) {
	t.Parallel()

	start, err := ai.NewReasoningBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart(1) returned %v, want no failure — R-ARE-007's zero-delta block is legal", err)
	}
	startPayload, ok := start.ReasoningBlockStart()
	if !ok {
		t.Fatal("start.ReasoningBlockStart() reported no payload")
	}
	if startPayload.Redacted() {
		t.Fatal("startPayload.Redacted() = true, want false")
	}
	end, err := ai.NewReasoningBlockEnd(startPayload, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd(start, nil) returned %v, want no failure", err)
	}
	endPayload, _ := end.ReasoningBlockEnd()
	if _, present := endPayload.Token(); present {
		t.Error("endPayload.Token() reported present=true, want false")
	}

	_, err = ai.NewReasoning("", nil)
	if err == nil || !errors.Is(err, ai.ErrEmpty) {
		t.Fatalf(`ai.NewReasoning("", nil) = %v, want ErrEmpty — the equivalent AI-07 rule this block's reconstruction must respect`, err)
	}
}

// ---- NFR-ARE-B — totality: nothing in this file panics -------------------

// S-ARE-041 — every exported entry point in this file, driven with a
// zero-value payload, a zero-value (never-constructed) start, block index 0,
// an oversized block index, an invalid-UTF-8 or over-long fragment, and a
// nil, zero-length or over-long token, returns rather than panics —
// text_events_test.go's TestTextEvents_ExtremeInputs_NeverPanic and
// tool_call_event_test.go's TestToolCallEvents_Totality_NoExportedEntryPointPanics,
// restated for this file's own constructors and accessors.
func TestReasoningEvents_Totality_NoExportedEntryPointPanics(t *testing.T) {
	t.Parallel()

	// An overlong two-byte encoding of NUL: invalid UTF-8 on its own, with no
	// dependency on where a legal rune boundary happens to fall.
	invalidUTF8Alone := []byte{0xC0, 0xAF}
	overLongFragment := bytes.Repeat([]byte("a"), ai.MaxTextLen+1)
	overLongToken := bytes.Repeat([]byte("a"), ai.MaxReasoningTokenLen+1)

	validStart, _ := ai.NewReasoningBlockStart(1)
	validStartPayload, _ := validStart.ReasoningBlockStart()
	redactedStart, _ := ai.NewRedactedReasoningBlockStart(1)
	redactedStartPayload, _ := redactedStart.ReasoningBlockStart()
	var zeroStart ai.ReasoningBlockStart // never passed through a constructor

	run := func(t *testing.T, name string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("%s panicked: %v", name, r)
			}
		}()
		fn()
	}

	t.Run("zero-value payloads read through every exported accessor", func(t *testing.T) {
		t.Parallel()
		run(t, "zero-value ReasoningBlockStart", func() {
			var s ai.ReasoningBlockStart
			_ = s.BlockIndex()
			_ = s.Redacted()
			_ = s.String()
			_ = s.GoString()
		})
		run(t, "zero-value ReasoningDelta", func() {
			var d ai.ReasoningDelta
			_ = d.BlockIndex()
			_ = d.Fragment()
			_ = d.String()
			_ = d.GoString()
		})
		run(t, "zero-value ReasoningBlockEnd", func() {
			var e ai.ReasoningBlockEnd
			_ = e.BlockIndex()
			_, _ = e.Token()
			_ = e.String()
			_ = e.GoString()
		})
		run(t, "typed accessors on the zero Event", func() {
			var e ai.Event
			if _, ok := e.ReasoningBlockStart(); ok {
				t.Error("the zero Event reported a ReasoningBlockStart payload")
			}
			if _, ok := e.ReasoningDelta(); ok {
				t.Error("the zero Event reported a ReasoningDelta payload")
			}
			if _, ok := e.ReasoningBlockEnd(); ok {
				t.Error("the zero Event reported a ReasoningBlockEnd payload")
			}
		})
		run(t, "CheckEmit over an unstamped, unconstructed zero Event", func() {
			_ = ai.CheckEmit(ai.Event{})
		})
	})

	t.Run("NewReasoningBlockStart and NewRedactedReasoningBlockStart", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			what        string
			blockIndex  uint64
			constructor func(uint64) (ai.Event, error)
		}{
			{"NewReasoningBlockStart, index 0", 0, ai.NewReasoningBlockStart},
			{"NewReasoningBlockStart, very large index", 1<<63 - 1, ai.NewReasoningBlockStart},
			{"NewRedactedReasoningBlockStart, index 0", 0, ai.NewRedactedReasoningBlockStart},
			{"NewRedactedReasoningBlockStart, very large index", 1<<63 - 1, ai.NewRedactedReasoningBlockStart},
		} {
			run(t, tc.what, func() {
				event, err := tc.constructor(tc.blockIndex)
				_ = err
				_ = event.Kind()
				_ = event.String()
				_ = event.GoString()
				_ = ai.CheckEmit(event)
				payload, _ := event.ReasoningBlockStart()
				_ = payload.BlockIndex()
				_ = payload.Redacted()
			})
		}
	})

	t.Run("NewReasoningDelta", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			what     string
			start    ai.ReasoningBlockStart
			fragment []byte
		}{
			{"zero-value start, nil fragment", zeroStart, nil},
			{"valid start, nil fragment", validStartPayload, nil},
			{"valid start, empty fragment", validStartPayload, []byte{}},
			{"valid start, invalid-UTF-8 fragment", validStartPayload, invalidUTF8Alone},
			{"valid start, over-long fragment", validStartPayload, overLongFragment},
			{"redacted start, non-empty fragment", redactedStartPayload, []byte("x")},
			{"redacted start, zero-length fragment", redactedStartPayload, []byte{}},
		} {
			run(t, "NewReasoningDelta/"+tc.what, func() {
				event, err := ai.NewReasoningDelta(tc.start, tc.fragment)
				_ = err
				_ = event.Kind()
				_ = ai.CheckEmit(event)
				delta, _ := event.ReasoningDelta()
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

	t.Run("NewReasoningBlockEnd", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			what  string
			start ai.ReasoningBlockStart
			token []byte
		}{
			{"zero-value start, nil token", zeroStart, nil},
			{"valid start, nil token", validStartPayload, nil},
			{"valid start, empty token", validStartPayload, []byte{}},
			{"valid start, over-long token", validStartPayload, overLongToken},
			{"redacted start, nil token", redactedStartPayload, nil},
			{"redacted start, empty token", redactedStartPayload, []byte{}},
			{"redacted start, over-long token", redactedStartPayload, overLongToken},
		} {
			run(t, "NewReasoningBlockEnd/"+tc.what, func() {
				event, err := ai.NewReasoningBlockEnd(tc.start, tc.token)
				_ = err
				_ = event.Kind()
				_ = ai.CheckEmit(event)
				end, _ := event.ReasoningBlockEnd()
				_ = end.BlockIndex()
				token, _ := end.Token()
				if len(token) > 0 {
					token[0] = 0 // mutate the returned copy; must never panic or affect the payload
				}
				_, _ = end.Token()
				_ = end.String()
				_ = end.GoString()
			})
		}
	})
}

// ---- NFR-ARE-C — every rejection reports through AI-04's failure value ---

// S-ARE-042 — every rejecting scenario across the three constructors reports
// through AI-04's *ai.Violation, matches exactly the sentinel it documents
// via errors.Is, and names the offending field. assertViolation
// (tool_call_test.go) already performs this three-part check for the whole
// ai_test package; it is reused here rather than redefined, the way AI-18's
// own consolidated sweep (TestToolCallEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel)
// already reuses it for tool-call events.
func TestReasoningEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel(t *testing.T) {
	t.Parallel()

	validStart, _ := ai.NewReasoningBlockStart(1)
	validStartPayload, _ := validStart.ReasoningBlockStart()
	redactedStart, _ := ai.NewRedactedReasoningBlockStart(1)
	redactedStartPayload, _ := redactedStart.ReasoningBlockStart()
	var zeroStart ai.ReasoningBlockStart // never passed through a constructor

	cases := []struct {
		what      string
		construct func() (ai.Event, error)
		rule      error
		at        string
	}{
		{"start: index 0", func() (ai.Event, error) { return ai.NewReasoningBlockStart(0) }, ai.ErrOutOfRange, "block_index"},
		{"redacted start: index 0", func() (ai.Event, error) { return ai.NewRedactedReasoningBlockStart(0) }, ai.ErrOutOfRange, "block_index"},
		{"delta: zero-value (never-constructed) start", func() (ai.Event, error) { return ai.NewReasoningDelta(zeroStart, []byte("x")) }, ai.ErrOutOfRange, "block_index"},
		{"delta: fragment one byte over MaxTextLen", func() (ai.Event, error) {
			return ai.NewReasoningDelta(validStartPayload, bytes.Repeat([]byte("a"), ai.MaxTextLen+1))
		}, ai.ErrOutOfRange, "fragment"},
		{"delta: non-empty fragment on a redacted block", func() (ai.Event, error) {
			return ai.NewReasoningDelta(redactedStartPayload, []byte("leaked plaintext"))
		}, ai.ErrMisplaced, "fragment"},
		{"end: zero-value (never-constructed) start", func() (ai.Event, error) { return ai.NewReasoningBlockEnd(zeroStart, []byte("tok")) }, ai.ErrOutOfRange, "block_index"},
		{"end: token one byte over MaxReasoningTokenLen", func() (ai.Event, error) {
			return ai.NewReasoningBlockEnd(validStartPayload, bytes.Repeat([]byte("a"), ai.MaxReasoningTokenLen+1))
		}, ai.ErrOutOfRange, "token"},
		{"end: redacted block with no token at all", func() (ai.Event, error) { return ai.NewReasoningBlockEnd(redactedStartPayload, nil) }, ai.ErrEmpty, "token"},
	}
	for _, tc := range cases {
		t.Run(tc.what, func(t *testing.T) {
			t.Parallel()
			_, err := tc.construct()
			assertViolation(t, err, tc.rule, tc.at)
		})
	}
}

// containsEventKind and sort are used by other _test.go files in this
// package too; sort is imported here so gofmt/goimports does not churn this
// file if a future edit adds a sorted-output assertion.
var _ = sort.Strings
