// Tests for AI-09.1 and AI-09.2 — the tool call and its ordinal.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
// surface an adapter in another package sees. That is not a stylistic choice
// here: doc 0002 asks for an external-package read on both leaves, because the
// consumer this contract exists for is AI-26's translator, which lives in a
// vendor package and can reach nothing else.
package ai_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-09.1 item 1 — the walking skeleton.
//
// Construct a tool call, place it in a message, read it back out from another
// package, and recover all three values the register names: the call's
// identity, the tool's name, and the argument bytes (V-REQ-16).
func TestToolCall_IdentityNameAndArguments_ReadBackFromAMessageInAnExternalPackage(t *testing.T) {
	t.Parallel()

	const (
		id   = "toolu_01A09Kf7"
		name = "read_file"
	)
	arguments := []byte(`{"path":"/etc/hosts","limit":10}`)

	part, err := ai.NewToolCall(id, name, arguments)
	if err != nil {
		t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
	}

	message, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}

	content := message.Content()
	if len(content) != 1 {
		t.Fatalf("message.Content() has %d elements, want 1", len(content))
	}

	call, ok := content[0].ToolCall()
	if !ok {
		t.Fatalf("content[0].ToolCall() reported no tool call on a part built by ai.NewToolCall")
	}

	if got := call.ID(); got != id {
		t.Errorf("call.ID() = %q, want %q", got, id)
	}
	if got := call.Name(); got != name {
		t.Errorf("call.Name() = %q, want %q", got, name)
	}
	if got := call.Arguments(); !bytes.Equal(got, arguments) {
		t.Errorf("call.Arguments() = %q, want %q", got, arguments)
	}
}

// canonicalizingFixture is argument bytes chosen so that a re-marshalling
// implementation fails rather than passes for the wrong reason.
//
// Three properties, each of which a JSON round trip rewrites: the object keys
// are in an order encoding/json would sort, the interior whitespace is irregular
// and would be removed, and the nested array is spread over lines an encoder
// would collapse. A contract that decoded and re-encoded these bytes would
// return something a byte comparison rejects, which is the whole point of
// V-REQ-17.
const canonicalizingFixture = `{ "zeta": 1,
  "alpha":   { "nested": [ 1,  2 , 3 ] } ,
  "beta" : "  spaced  " }`

// AI-09.1 item 2 — argument bytes survive unmodified.
//
// Byte-equality is the contract (V-REQ-17), and AI-26's request translation and
// AI-30's stream reassembly both depend on it. The two aliasing sub-tests are
// the second half: a constructor that clones and a reader that does not passes
// every construction test and fails the moment two consumers hold one part.
func TestToolCall_ArgumentBytes_PassThroughByteIdentically(t *testing.T) {
	t.Parallel()

	t.Run("bytes a canonicalizing encoder would rewrite survive a message round trip", func(t *testing.T) {
		t.Parallel()

		supplied := []byte(canonicalizingFixture)

		call := mustReadToolCallBack(t, ai.RoleAssistant, mustToolCall(t, "call-1", "search", supplied))

		if got := call.Arguments(); !bytes.Equal(got, supplied) {
			t.Errorf("call.Arguments() = %q,\n                want %q\n"+
				"  V-REQ-17: no re-marshalling, no key reordering, no whitespace normalization.", got, supplied)
		}
	})

	t.Run("the caller may reuse the slice it passed", func(t *testing.T) {
		t.Parallel()

		supplied := []byte(canonicalizingFixture)
		part := mustToolCall(t, "call-2", "search", supplied)

		for i := range supplied {
			supplied[i] = 'X'
		}

		call, ok := part.ToolCall()
		if !ok {
			t.Fatal("part.ToolCall() reported no tool call")
		}
		if got := call.Arguments(); !bytes.Equal(got, []byte(canonicalizingFixture)) {
			t.Errorf("after the caller overwrote its own buffer, call.Arguments() = %q, want the bytes supplied at construction", got)
		}
	})

	t.Run("a consumer that rewrites what it received cannot rewrite the part", func(t *testing.T) {
		t.Parallel()

		part := mustToolCall(t, "call-3", "search", []byte(canonicalizingFixture))
		call, ok := part.ToolCall()
		if !ok {
			t.Fatal("part.ToolCall() reported no tool call")
		}

		first := call.Arguments()
		for i := range first {
			first[i] = 'X'
		}

		if got := call.Arguments(); !bytes.Equal(got, []byte(canonicalizingFixture)) {
			t.Errorf("after a consumer overwrote the slice it received, a second read returned %q, want the constructed bytes", got)
		}

		second, ok := part.ToolCall()
		if !ok {
			t.Fatal("part.ToolCall() reported no tool call on a second read")
		}
		if got := second.Arguments(); !bytes.Equal(got, []byte(canonicalizingFixture)) {
			t.Errorf("a second consumer observed %q, want the constructed bytes — two holders of one part must not observe each other", got)
		}
	})
}

// AI-09.1 item 3 — the construction rules, and the sentinels they report.
//
// Three rules, in the documented order: a non-empty identity, a non-empty tool
// name, and argument bytes that are syntactically well-formed for the documented
// encoding. All three report through AI-04's landed classes; none of them needed
// a new one, and design.md § 4 records why the syntax check lands here while
// AI-08 declined it for schema bytes.
func TestNewToolCall_BrokenConstructionRules_FailWithTheDocumentedSentinels(t *testing.T) {
	t.Parallel()

	wellFormed := []byte(`{"path":"/etc/hosts"}`)

	cases := []struct {
		what      string
		id        string
		name      string
		arguments []byte
		rule      error
		at        string
	}{
		{"an empty identity", "", "read_file", wellFormed, ai.ErrEmpty, "id"},
		{"an empty tool name", "call-1", "", wellFormed, ai.ErrEmpty, "name"},
		{"argument bytes that are a truncated object", "call-1", "read_file", []byte(`{"path":`), ai.ErrMalformed, "arguments"},
		{"argument bytes that are whitespace only", "call-1", "read_file", []byte("  \t\n "), ai.ErrMalformed, "arguments"},
		{"argument bytes that are not the documented encoding at all", "call-1", "read_file", []byte("path=/etc/hosts"), ai.ErrMalformed, "arguments"},
		{"argument bytes carrying two values rather than one", "call-1", "read_file", []byte(`{"a":1}{"b":2}`), ai.ErrMalformed, "arguments"},
	}

	for _, testCase := range cases {
		t.Run(testCase.what, func(t *testing.T) {
			t.Parallel()

			part, err := ai.NewToolCall(testCase.id, testCase.name, testCase.arguments)
			if err == nil {
				t.Fatalf("ai.NewToolCall accepted %s, want a caller-contract failure", testCase.what)
			}
			assertViolation(t, err, testCase.rule, testCase.at)
			if part != (ai.Part{}) {
				t.Errorf("a rejected construction returned a non-zero part, want the zero ai.Part")
			}
		})
	}

	t.Run("more than one rule broken reports the first in the documented order", func(t *testing.T) {
		t.Parallel()

		// Every rule is broken at once. The identity's is first, and stays
		// first across repeated runs — AI-04.3's determinism property, at this
		// contract's scope.
		for range 32 {
			_, err := ai.NewToolCall("", "", []byte("{"))
			assertViolation(t, err, ai.ErrEmpty, "id")
		}

		// With the identity supplied, the name's rule wins over the arguments'.
		_, err := ai.NewToolCall("call-1", "", []byte("{"))
		assertViolation(t, err, ai.ErrEmpty, "name")
	})

	t.Run("the failure never reproduces the values it was given", func(t *testing.T) {
		t.Parallel()

		const secret = "sk-live-4f2a9c" //nolint:gosec // a fixture, not a credential

		_, err := ai.NewToolCall("", secret, []byte(`{"token":"`+secret+`"}`))
		if err == nil {
			t.Fatal("ai.NewToolCall accepted an empty identity, want a caller-contract failure")
		}
		if rendered := err.Error(); strings.Contains(rendered, secret) {
			t.Errorf("the rendered failure %q reproduces a value it was given.\n"+
				"  V-FAIL-13 puts the redaction posture on the type: a rendered message is built only from text this package wrote.", rendered)
		}
	})

	t.Run("well-formed bytes no declared tool would accept still construct", func(t *testing.T) {
		t.Parallel()

		// This is the register's first trap, drawn at bytes versus meaning.
		// Layer 1 checks the encoding and never the schema — ErrMalformed's own
		// GoDoc says so, and nothing in this package holds a schema to check
		// against at construction time.
		if _, err := ai.NewToolCall("call-1", "read_file", []byte(`{"wildly":"wrong","for":["every","tool"]}`)); err != nil {
			t.Errorf("ai.NewToolCall rejected well-formed bytes with %v; validating arguments against a tool's schema is out of scope for Layer 1", err)
		}
		if _, err := ai.NewToolCall("call-1", "read_file", []byte(`"a bare string is a whole JSON value"`)); err != nil {
			t.Errorf("ai.NewToolCall rejected a well-formed non-object value with %v; the rule is syntax, not shape", err)
		}
	})
}

// AI-09.1 item 4 — a call with no arguments.
//
// A no-argument tool is routine, so the call must be constructible; and a parse
// failure on empty input is a shipped-SDK bug class, so the absent case
// normalizes to one canonical form that decodes. The normalization is scoped to
// the absent case alone: supplied bytes are never rewritten, which is the
// sentence that keeps this compatible with V-REQ-17.
func TestNewToolCall_AbsentArguments_NormalizeToOneCanonicalDecodableForm(t *testing.T) {
	t.Parallel()

	fromNil, err := ai.NewToolCall("call-1", "now", nil)
	if err != nil {
		t.Fatalf("ai.NewToolCall with nil arguments returned %v; a tool that takes no arguments is routine", err)
	}
	fromEmpty, err := ai.NewToolCall("call-1", "now", []byte{})
	if err != nil {
		t.Fatalf("ai.NewToolCall with zero-length arguments returned %v", err)
	}

	nilCall, ok := fromNil.ToolCall()
	if !ok {
		t.Fatal("the part built from nil arguments carries no tool call")
	}
	emptyCall, ok := fromEmpty.ToolCall()
	if !ok {
		t.Fatal("the part built from zero-length arguments carries no tool call")
	}

	t.Run("absent and zero-length agree on one canonical form", func(t *testing.T) {
		if got, want := nilCall.Arguments(), emptyCall.Arguments(); !bytes.Equal(got, want) {
			t.Errorf("nil arguments read back as %q and zero-length as %q; there must be exactly one canonical empty form", got, want)
		}
		if fromNil != fromEmpty {
			t.Errorf("two calls built from equivalent absent arguments are not ==; ai.Part documents equality as defined and comparing payloads")
		}
	})

	t.Run("the canonical form decodes without a special case", func(t *testing.T) {
		var decoded map[string]any
		if err := json.Unmarshal(nilCall.Arguments(), &decoded); err != nil {
			t.Errorf("decoding the canonical empty arguments %q failed with %v.\n"+
				"  A consumer decoding a call's arguments must never meet an input that parses only by special case.", nilCall.Arguments(), err)
		}
		if len(decoded) != 0 {
			t.Errorf("the canonical empty arguments decoded to %v, want no arguments", decoded)
		}
	})

	t.Run("supplied bytes are never replaced by the canonical form", func(t *testing.T) {
		// Semantically an empty argument object, written differently. A
		// contract that normalized supplied bytes would return the canonical
		// form here, and V-REQ-17 forbids exactly that.
		supplied := []byte("{\n  \n}")
		call := mustReadToolCallBack(t, ai.RoleAssistant, mustToolCall(t, "call-2", "now", supplied))
		if got := call.Arguments(); !bytes.Equal(got, supplied) {
			t.Errorf("call.Arguments() = %q, want the supplied %q; the normalization is for absent arguments, not for supplied ones", got, supplied)
		}
	})
}

// AI-09.1 item 5 — appended during implementation.
//
// AI-06 put the redaction posture on [ai.Part] itself, which was complete while
// every payload was a string returned by an accessor. This milestone is the
// first to export a payload *type*, and that reopens it: a consumer holding an
// ai.ToolCall and writing "%v" would print its unexported fields through
// reflection, and those fields carry argument bytes a model produced from a
// user's prompt. V-FAIL-13 makes redaction a property of the type rather than of
// anyone's discipline, so the type gets renderers.
func TestToolCall_Rendering_NeverReproducesItsPayload(t *testing.T) {
	t.Parallel()

	const secret = "sk-live-4f2a9c" //nolint:gosec // a fixture, not a credential

	part := mustToolCall(t, "call-"+secret, "read_"+secret, []byte(`{"token":"`+secret+`"}`))
	call, ok := part.ToolCall()
	if !ok {
		t.Fatal("part.ToolCall() reported no tool call")
	}

	for _, rendering := range []struct {
		verb string
		got  string
	}{
		{"%v", fmt.Sprintf("%v", call)},
		{"%s", call.String()},
		{"%#v", fmt.Sprintf("%#v", call)},
		{"%v on the part", fmt.Sprintf("%v", part)},
		{"%#v on the part", fmt.Sprintf("%#v", part)},
	} {
		if strings.Contains(rendering.got, secret) {
			t.Errorf("%s rendered %q, which reproduces the payload it carries.\n"+
				"  A consumer that wants the payload calls the accessor, which is what an accessor is for.", rendering.verb, rendering.got)
		}
	}

	if got := fmt.Sprintf("%v", part); got != "part(tool_call)" {
		t.Errorf("%%v on the part = %q, want %q — the rendering names the kind and nothing else", got, "part(tool_call)")
	}
}

// AI-09.2 item 1 — the call ordinal is observable and stable.
//
// V-STR-21 defines the ordinal as "the observable position of a tool call among
// the calls of one response". It is derived from position rather than stored on
// the call, for AI-06's own reason one level up: a stored ordinal and the
// position a message actually holds are two facts that can disagree, and a part
// is placed in a message after it is built. design.md § 5 records the choice and
// both of its consequences.
//
// Layer 2's parallel-execution re-join is ordered by this (doc 0001 § 7 G5), and
// several providers reject tool results that do not correspond positionally to
// their calls.
func TestToolCalls_InterleavedContent_AreObservableInOrderWithStableOrdinals(t *testing.T) {
	t.Parallel()

	// Interleaved with another kind on purpose: the ordinal counts calls, not
	// content elements, so a message whose calls are not at content indices
	// 0, 1, 2 is the case that separates the two.
	message, err := ai.NewMessage(
		ai.RoleAssistant,
		mustText(t, "I will read three files."),
		mustToolCall(t, "call-alpha", "read_file", []byte(`{"path":"a"}`)),
		mustText(t, "and then"),
		mustToolCall(t, "call-beta", "read_file", []byte(`{"path":"b"}`)),
		mustToolCall(t, "call-gamma", "read_file", []byte(`{"path":"c"}`)),
	)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}

	want := []string{"call-alpha", "call-beta", "call-gamma"}

	t.Run("the calls appear in content order and nothing else does", func(t *testing.T) {
		t.Parallel()

		assertOrdinals(t, ai.ToolCalls(message.Content()), want)
	})

	t.Run("every read yields the same ordinals", func(t *testing.T) {
		t.Parallel()

		for range 16 {
			assertOrdinals(t, ai.ToolCalls(message.Content()), want)
		}
	})
}

// AI-09.2 item 2 — the ordinal survives message copy and readback.
//
// The middle sub-test is the one that proves the ordinal is *derived* rather
// than stored: the calls are read out of one message and rebuilt into a second
// in the opposite order, and the ordinals follow the position the second message
// holds. An implementation that recorded an ordinal at construction, or that
// imposed an order of its own, disagrees here and agrees everywhere else.
func TestToolCalls_MessageCopyAndReadback_PreserveEveryOrdinal(t *testing.T) {
	t.Parallel()

	built := []ai.Part{
		mustToolCall(t, "call-alpha", "read_file", []byte(`{"path":"a"}`)),
		mustToolCall(t, "call-beta", "read_file", []byte(`{"path":"b"}`)),
		mustToolCall(t, "call-gamma", "read_file", []byte(`{"path":"c"}`)),
	}
	message, err := ai.NewMessage(ai.RoleAssistant, built...)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}

	t.Run("a value copy of the message observes the same ordinals", func(t *testing.T) {
		t.Parallel()

		copied := message
		assertOrdinals(t, ai.ToolCalls(copied.Content()), []string{"call-alpha", "call-beta", "call-gamma"})
	})

	t.Run("the ordinal follows the position a message holds, not the order the calls were built in", func(t *testing.T) {
		t.Parallel()

		read := ai.ToolCalls(message.Content())
		rebuilt := make([]ai.Part, 0, len(read))
		for i := len(read) - 1; i >= 0; i-- {
			rebuilt = append(rebuilt, mustToolCall(t, read[i].ID(), read[i].Name(), read[i].Arguments()))
		}

		reversed, err := ai.NewMessage(ai.RoleAssistant, rebuilt...)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		assertOrdinals(t, ai.ToolCalls(reversed.Content()), []string{"call-gamma", "call-beta", "call-alpha"})
		// And the original is untouched by the second message existing.
		assertOrdinals(t, ai.ToolCalls(message.Content()), []string{"call-alpha", "call-beta", "call-gamma"})
	})

	t.Run("a consumer that rewrites the sequence it received cannot change the ordinals", func(t *testing.T) {
		t.Parallel()

		first := ai.ToolCalls(message.Content())
		first[0], first[2] = first[2], first[0]

		assertOrdinals(t, ai.ToolCalls(message.Content()), []string{"call-alpha", "call-beta", "call-gamma"})
	})

	t.Run("the derivation is total", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []struct {
			what    string
			content []ai.Part
			want    []string
		}{
			{"a nil sequence", nil, nil},
			{"an empty sequence", []ai.Part{}, nil},
			{"a sequence holding no tool call", []ai.Part{mustText(t, "no calls here")}, nil},
			{"a sequence holding a part that was never constructed", []ai.Part{{}, built[0], {}}, []string{"call-alpha"}},
		} {
			t.Run(testCase.what, func(t *testing.T) {
				t.Parallel()

				got := ai.ToolCalls(testCase.content)
				if len(got) != len(testCase.want) {
					t.Fatalf("ai.ToolCalls(%s) returned %d calls, want %d", testCase.what, len(got), len(testCase.want))
				}
				assertOrdinals(t, got, testCase.want)
			})
		}
	})
}

// assertOrdinals checks that a derived call sequence carries the expected
// identities at the expected ordinals.
func assertOrdinals(t *testing.T, calls []ai.ToolCall, want []string) {
	t.Helper()

	if len(calls) != len(want) {
		t.Fatalf("ai.ToolCalls returned %d calls, want %d — a part of another kind is skipped, not counted", len(calls), len(want))
	}
	for ordinal, call := range calls {
		if got := call.ID(); got != want[ordinal] {
			t.Errorf("the call at ordinal %d is %q, want %q", ordinal, got, want[ordinal])
		}
	}
}

// mustText constructs a text part or fails the test.
func mustText(t *testing.T, text string) ai.Part {
	t.Helper()

	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	return part
}

// assertViolation checks that a failure carries the expected rule class at the
// expected position, and no other registered class.
func assertViolation(t *testing.T, err error, rule error, at string) {
	t.Helper()

	if err == nil {
		t.Fatalf("want a failure carrying %v, got none", rule)
	}
	if !errors.Is(err, rule) {
		t.Errorf("errors.Is(err, %v) = false for %v", rule, err)
	}
	for _, other := range []error{ai.ErrEmpty, ai.ErrNotInVocabulary, ai.ErrOutOfRange, ai.ErrMalformed, ai.ErrUnresolvedReference} {
		if other == rule {
			continue
		}
		if errors.Is(err, other) {
			t.Errorf("%v also matches %v; one rule class per failure, or the two axes AI-04 defines collapse into one", err, other)
		}
	}

	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As reported no *ai.Violation for %T", err)
	}
	if got := violation.Path().String(); got != at {
		t.Errorf("violation.Path() = %q, want %q", got, at)
	}
}

// mustToolCall constructs a tool call or fails the test.
func mustToolCall(t *testing.T, id, name string, arguments []byte) ai.Part {
	t.Helper()

	part, err := ai.NewToolCall(id, name, arguments)
	if err != nil {
		t.Fatalf("ai.NewToolCall(%q, %q, …) returned %v, want no failure", id, name, err)
	}
	return part
}

// mustReadToolCallBack places a part in a message and reads the tool call back
// out of it, which is the path an adapter in another package walks.
func mustReadToolCallBack(t *testing.T, role ai.Role, part ai.Part) ai.ToolCall {
	t.Helper()

	message, err := ai.NewMessage(role, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	content := message.Content()
	if len(content) != 1 {
		t.Fatalf("message.Content() has %d elements, want 1", len(content))
	}
	call, ok := content[0].ToolCall()
	if !ok {
		t.Fatal("content[0].ToolCall() reported no tool call on a part built by ai.NewToolCall")
	}
	return call
}
