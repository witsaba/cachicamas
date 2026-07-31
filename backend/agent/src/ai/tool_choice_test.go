// Tests for AI-08.3 — the tool-choice vocabulary and its cross-validation.
//
// External test package, so every assertion is written against exactly the
// surface a consumer in another package sees.
package ai_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-08.3 — the tool-choice vocabulary is closed and each of its four members
// is constructible: automatic, none, required, and a specific named tool.
//
// Three members carry no payload and fit AI-05's pattern unchanged. The fourth
// carries a name, and design.md § 5 records how the pattern was extended for
// it: the vocabulary stays a payload-free integer, the payload lives in a
// separate value type carrying a member plus its payload, and the table row
// widens by one arity column. AI-05's design.md § 3.2 pre-authorised exactly
// that widening — "Rule 2's table may carry more than a name … that is a
// widening of the row, not a change of shape" — so the two load-bearing rules,
// iota + 1 and an enumeration over the constant space, are untouched.
func TestToolChoice_EachVocabularyMember_IsConstructibleAndReadsBack(t *testing.T) {
	t.Parallel()

	t.Run("each payload-free member constructs and reads back", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []ai.ToolChoiceMode{ai.ToolChoiceAuto, ai.ToolChoiceNone, ai.ToolChoiceRequired} {
			choice, err := ai.NewToolChoice(mode)
			if err != nil {
				t.Errorf("NewToolChoice(%v) returned %v, want no failure", mode, err)
				continue
			}
			if got := choice.Mode(); got != mode {
				t.Errorf("choice.Mode() = %v, want %v", got, mode)
			}
			if name, ok := choice.Name(); ok {
				t.Errorf("choice.Name() = (%q, true), want it to report carrying no name", name)
			}
		}
	})

	t.Run("the payload-carrying member constructs and carries its name", func(t *testing.T) {
		t.Parallel()

		choice, err := ai.NewNamedToolChoice("get_weather")
		if err != nil {
			t.Fatalf("NewNamedToolChoice returned %v, want no failure", err)
		}
		if got := choice.Mode(); got != ai.ToolChoiceSpecific {
			t.Errorf("choice.Mode() = %v, want %v", got, ai.ToolChoiceSpecific)
		}
		name, ok := choice.Name()
		if !ok {
			t.Fatal("choice.Name() reported no name, want the one supplied")
		}
		if name != "get_weather" {
			t.Errorf("choice.Name() = %q, want %q", name, "get_weather")
		}
	})

	t.Run("the payload-free constructor rejects the payload-carrying member", func(t *testing.T) {
		t.Parallel()

		// It must fail rather than quietly produce a nameless "specific"
		// choice: the member requires a name and none was supplied, which is
		// exactly what the empty rule class means.
		choice, err := ai.NewToolChoice(ai.ToolChoiceSpecific)
		if err == nil {
			t.Fatal("NewToolChoice(ToolChoiceSpecific) returned no failure, want one")
		}
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
		}
		if got := violation.Path().String(); got != "name" {
			t.Errorf("violation.Path() = %q, want %q", got, "name")
		}
		if got := choice.Mode(); got != 0 {
			t.Errorf("the returned choice has mode %v, want the zero ToolChoice", got)
		}
	})

	t.Run("a mode outside the vocabulary is rejected", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []ai.ToolChoiceMode{0, 5, 200, 255} {
			_, err := ai.NewToolChoice(mode)
			if err == nil {
				t.Errorf("NewToolChoice(%d) returned no failure, want one", uint8(mode))
				continue
			}
			if !errors.Is(err, ai.ErrNotInVocabulary) {
				t.Errorf("NewToolChoice(%d): errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", uint8(mode), err)
			}
		}
	})

	t.Run("the named constructor applies the tool-name rule", func(t *testing.T) {
		t.Parallel()

		// One name rule in the package, not two that could drift. A choice
		// naming a syntactically impossible tool can never resolve against any
		// set, so it fails at construction rather than at cross-validation.
		cases := []struct {
			label string
			name  string
			want  error
		}{
			{"empty", "", ai.ErrEmpty},
			{"over the ceiling", strings.Repeat("a", ai.MaxToolNameLen+1), ai.ErrOutOfRange},
			{"containing a dot", "github.list_prs", ai.ErrMalformed},
			{"beginning with a digit", "1st_tool", ai.ErrMalformed},
		}

		for _, tc := range cases {
			t.Run(tc.label, func(t *testing.T) {
				t.Parallel()

				choice, err := ai.NewNamedToolChoice(tc.name)
				if err == nil {
					t.Fatalf("NewNamedToolChoice(%q) returned no failure, want one", tc.name)
				}
				if !errors.Is(err, tc.want) {
					t.Errorf("errors.Is(err, %v) = false, want true (err = %v)", tc.want, err)
				}

				var violation *ai.Violation
				if !errors.As(err, &violation) {
					t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
				}
				if got := violation.Path().String(); got != "name" {
					t.Errorf("violation.Path() = %q, want %q", got, "name")
				}
				if got := choice.Mode(); got != 0 {
					t.Errorf("the returned choice has mode %v, want the zero ToolChoice", got)
				}
			})
		}
	})

	t.Run("the vocabulary enumerates in a stable order and cannot be rewritten", func(t *testing.T) {
		t.Parallel()

		modes := ai.ToolChoiceModes()
		if len(modes) != 4 {
			t.Fatalf("ToolChoiceModes() returned %d members, want 4", len(modes))
		}
		if again := ai.ToolChoiceModes(); !slices.Equal(modes, again) {
			t.Errorf("ToolChoiceModes() = %v then %v, want a stable declaration order", modes, again)
		}

		modes[0] = ai.ToolChoiceMode(200)
		if fresh := ai.ToolChoiceModes(); fresh[0] == ai.ToolChoiceMode(200) {
			t.Error("a consumer rewrote the vocabulary; ToolChoiceModes must return a fresh slice")
		}
	})

	t.Run("rendering is stable and lowercase; parsing is exact", func(t *testing.T) {
		t.Parallel()

		for _, want := range ai.ToolChoiceModes() {
			name := want.String()
			if name == "" || name != strings.ToLower(name) {
				t.Errorf("ToolChoiceMode(%d).String() = %q, want a non-empty lowercase rendering", uint8(want), name)
				continue
			}
			got, err := ai.ParseToolChoiceMode(name)
			if err != nil {
				t.Errorf("ParseToolChoiceMode(%q) returned %v, want no failure", name, err)
				continue
			}
			if got != want {
				t.Errorf("ParseToolChoiceMode(%q) = %v, want %v", name, got, want)
			}
		}

		rejected := []string{
			"Auto", "AUTO", "None", "REQUIRED", "Specific",
			" auto", "auto ", "\tauto", "auto\n",
			"any", "tool", "all", "toolChoice(5)", "5",
		}
		for _, name := range rejected {
			got, err := ai.ParseToolChoiceMode(name)
			if err == nil {
				t.Errorf("ParseToolChoiceMode(%q) = %v with no failure, want a failure", name, got)
				continue
			}
			if !errors.Is(err, ai.ErrNotInVocabulary) {
				t.Errorf("ParseToolChoiceMode(%q): errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", name, err)
			}
			if got != 0 {
				t.Errorf("ParseToolChoiceMode(%q) = %v on failure, want the zero mode", name, got)
			}
		}

		if _, err := ai.ParseToolChoiceMode(""); !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("ParseToolChoiceMode(\"\"): errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
		}

		for _, mode := range []ai.ToolChoiceMode{0, 5, 200, 255} {
			name := mode.String()
			if !strings.Contains(name, strconv.Itoa(int(mode))) {
				t.Errorf("ToolChoiceMode(%d).String() = %q, want it to identify the value", uint8(mode), name)
			}
			if _, err := ai.ParseToolChoiceMode(name); err == nil {
				t.Errorf("ParseToolChoiceMode(%q) returned no failure — a diagnostic rendering must not round-trip", name)
			}
		}
	})
}

// AI-08.3 — a tool choice naming a tool that is not in the declared set fails
// validation with an AI-04 sentinel.
//
// ErrUnresolvedReference is the class by its own definition: "a value naming
// something the request does not declare". The register's own worked example of
// a caller-contract failure is this exact case — "a tool choice naming a tool
// absent from the declared set is decidable from the request alone".
func TestToolChoice_NamingAnUndeclaredTool_FailsWithUnresolvedReference(t *testing.T) {
	t.Parallel()

	set, err := ai.NewToolSet(declare(t, "read"), declare(t, "write"), declare(t, "edit"))
	if err != nil {
		t.Fatalf("NewToolSet returned %v, want no failure", err)
	}

	t.Run("a declared tool validates", func(t *testing.T) {
		t.Parallel()

		for _, name := range []string{"read", "write", "edit"} {
			choice, err := ai.NewNamedToolChoice(name)
			if err != nil {
				t.Fatalf("NewNamedToolChoice(%q) returned %v, want no failure", name, err)
			}
			if err := choice.ValidateAgainst(set); err != nil {
				t.Errorf("choice naming %q: ValidateAgainst returned %v, want no failure", name, err)
			}
		}
	})

	t.Run("an undeclared tool fails", func(t *testing.T) {
		t.Parallel()

		// "Read" is here on purpose: resolution is exact, so a case variant is
		// as undeclared as a name nobody ever mentioned.
		for _, name := range []string{"delete", "Read", "read_file", "rea"} {
			choice, err := ai.NewNamedToolChoice(name)
			if err != nil {
				t.Fatalf("NewNamedToolChoice(%q) returned %v, want no failure", name, err)
			}

			err = choice.ValidateAgainst(set)
			if err == nil {
				t.Errorf("choice naming %q: ValidateAgainst returned no failure, want one", name)
				continue
			}
			if !errors.Is(err, ai.ErrUnresolvedReference) {
				t.Errorf("choice naming %q: errors.Is(err, ErrUnresolvedReference) = false, want true (err = %v)", name, err)
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			if got := violation.Path().String(); got != "toolChoice.name" {
				t.Errorf("violation.Path() = %q, want %q", got, "toolChoice.name")
			}
		}
	})

	t.Run("the payload-free members never consult a name", func(t *testing.T) {
		t.Parallel()

		for _, mode := range []ai.ToolChoiceMode{ai.ToolChoiceAuto, ai.ToolChoiceNone, ai.ToolChoiceRequired} {
			choice, err := ai.NewToolChoice(mode)
			if err != nil {
				t.Fatalf("NewToolChoice(%v) returned %v, want no failure", mode, err)
			}
			if err := choice.ValidateAgainst(set); err != nil {
				t.Errorf("choice %v against a non-empty set: ValidateAgainst returned %v, want no failure", mode, err)
			}
		}
	})

	t.Run("a choice that is not a vocabulary member is rejected", func(t *testing.T) {
		t.Parallel()

		var zero ai.ToolChoice

		err := zero.ValidateAgainst(set)
		if err == nil {
			t.Fatal("the zero ToolChoice validated, want a failure — a value nobody set is not a default")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", err)
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
		}
		if got := violation.Path().String(); got != "toolChoice" {
			t.Errorf("violation.Path() = %q, want %q", got, "toolChoice")
		}
	})

	t.Run("the offered name never reaches the failure", func(t *testing.T) {
		t.Parallel()

		const sentinelName = "CACHICAMA_SENTINEL_NAME_7b31"

		choice, err := ai.NewNamedToolChoice(sentinelName)
		if err != nil {
			t.Fatalf("NewNamedToolChoice returned %v, want no failure", err)
		}
		err = choice.ValidateAgainst(set)
		if err == nil {
			t.Fatal("ValidateAgainst returned no failure, want one")
		}
		if strings.Contains(err.Error(), sentinelName) {
			t.Errorf("Error() = %q, want it to carry none of the offered name", err.Error())
		}
	})
}

// AI-08.3 — a tool choice other than "none" against an empty tool set fails.
//
// This is the combination every provider rejects, caught client-side. Its value
// is the round trip it does not take: without this rule the caller learns only
// after the request has crossed the network, and learns it as a vendor error
// string rather than as a positioned caller-contract failure.
//
// "None" against an empty set is legal and is asserted here rather than merely
// permitted by omission: expressing "do not call a tool" when there are no
// tools is coherent, and a later implementation that rejected it would be a
// regression this test names.
func TestToolChoice_AgainstAnEmptyToolSet_OnlyNoneIsLegal(t *testing.T) {
	t.Parallel()

	empty, err := ai.NewToolSet()
	if err != nil {
		t.Fatalf("NewToolSet() returned %v, want no failure", err)
	}

	t.Run("every mode other than none fails", func(t *testing.T) {
		t.Parallel()

		choices := map[string]func() (ai.ToolChoice, error){
			"auto":     func() (ai.ToolChoice, error) { return ai.NewToolChoice(ai.ToolChoiceAuto) },
			"required": func() (ai.ToolChoice, error) { return ai.NewToolChoice(ai.ToolChoiceRequired) },
			"specific": func() (ai.ToolChoice, error) { return ai.NewNamedToolChoice("get_weather") },
		}

		for label, build := range choices {
			t.Run(label, func(t *testing.T) {
				t.Parallel()

				choice, err := build()
				if err != nil {
					t.Fatalf("constructing the %s choice returned %v, want no failure", label, err)
				}

				err = choice.ValidateAgainst(empty)
				if err == nil {
					t.Fatalf("%s against an empty tool set validated, want a failure", label)
				}
				if !errors.Is(err, ai.ErrEmpty) {
					t.Errorf("errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
				}

				var violation *ai.Violation
				if !errors.As(err, &violation) {
					t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
				}
				if got := violation.Path().String(); got != "tools" {
					t.Errorf("violation.Path() = %q, want %q", got, "tools")
				}
			})
		}
	})

	t.Run("none against an empty set is legal", func(t *testing.T) {
		t.Parallel()

		choice, err := ai.NewToolChoice(ai.ToolChoiceNone)
		if err != nil {
			t.Fatalf("NewToolChoice(ToolChoiceNone) returned %v, want no failure", err)
		}
		if err := choice.ValidateAgainst(empty); err != nil {
			t.Errorf("none against an empty tool set returned %v, want no failure", err)
		}
		var zeroSet ai.ToolSet
		if err := choice.ValidateAgainst(zeroSet); err != nil {
			t.Errorf("none against the zero ToolSet returned %v, want no failure", err)
		}
	})

	t.Run("the emptiness rule precedes the resolution rule, on every run", func(t *testing.T) {
		t.Parallel()

		// A specific choice against an empty set violates both rules 2 and 3.
		// Rule 2 wins: "you have declared no tools at all" is the more
		// fundamental fact and the more actionable message than "the tool you
		// named is not declared", which is true of every possible name here.
		choice, err := ai.NewNamedToolChoice("get_weather")
		if err != nil {
			t.Fatalf("NewNamedToolChoice returned %v, want no failure", err)
		}

		for range 64 {
			err := choice.ValidateAgainst(empty)
			if !errors.Is(err, ai.ErrEmpty) {
				t.Fatalf("errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
			}
			if errors.Is(err, ai.ErrUnresolvedReference) {
				t.Fatalf("errors.Is(err, ErrUnresolvedReference) = true, want false — the documented order is not stable (err = %v)", err)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
			}
			if got := violation.Path().String(); got != "tools" {
				t.Fatalf("violation.Path() = %q, want %q", got, "tools")
			}
		}
	})

	t.Run("a non-member mode is rejected before the emptiness rule", func(t *testing.T) {
		t.Parallel()

		var zero ai.ToolChoice
		err := zero.ValidateAgainst(empty)
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", err)
		}
		if errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ErrEmpty) = true, want false — the vocabulary rule is first (err = %v)", err)
		}
	})
}

// AI-08.3 *(pin)* — every declared tool-choice member is exhaustively
// tabulated.
//
// Green from birth and exempt from red-first per doc 0002's leaf anatomy. It
// bites when a member is declared without a table entry, and the reason it can
// is the rule AI-05 calls load-bearing: ToolChoiceModes enumerates the
// *constant space*, not the table. An enumeration derived from the table would
// list exactly the members that have an entry, so the omission it exists to
// catch would be invisible to it.
//
// Step 3 is this milestone's own addition, and it is what the widened table row
// buys. It asserts a *biconditional* against the arity column rather than
// against a hard-coded list of members, so a fifth member added tomorrow is
// covered by whichever branch its own row selects.
//
// The honest limit, recorded rather than discovered later: step 4 maps "needs a
// name" onto one specific constructor. With one payload-carrying member that is
// exact. If a second ever lands, the arity column stops being a boolean and
// step 4 needs a per-member constructor mapping — a widening of the same row
// and the same shape, which the milestone that meets the case should read here
// rather than re-derive.
func TestToolChoiceMode_DeclaredVocabulary_IsExhaustivelyTabulated(t *testing.T) {
	t.Parallel()

	modes := ai.ToolChoiceModes()
	if len(modes) == 0 {
		t.Fatal("ToolChoiceModes() returned nothing; the pin would pass vacuously")
	}

	for _, mode := range modes {
		name := mode.String()

		// 1. The member has a table entry with a lowercase rendering.
		if name == "" || strings.HasPrefix(name, "toolChoice(") {
			t.Errorf("ToolChoiceMode(%d) has no table entry: String() = %q", uint8(mode), name)
			continue
		}
		if name != strings.ToLower(name) {
			t.Errorf("ToolChoiceMode(%d).String() = %q, want it to be lowercase", uint8(mode), name)
		}

		// 2. The rendering parses back to the same member.
		parsed, err := ai.ParseToolChoiceMode(name)
		if err != nil {
			t.Errorf("ToolChoiceMode(%d): ParseToolChoiceMode(%q) returned %v, want it to round-trip", uint8(mode), name, err)
		} else if parsed != mode {
			t.Errorf("ToolChoiceMode(%d): ParseToolChoiceMode(%q) = %v, want %v", uint8(mode), name, parsed, mode)
		}

		// 3. The member is constructible by exactly the constructor its
		//    declared arity selects — asserted as a biconditional against the
		//    arity column, not against a list of member names.
		payloadFree, freeErr := ai.NewToolChoice(mode)
		needsName := freeErr != nil

		if needsName {
			if !errors.Is(freeErr, ai.ErrEmpty) {
				t.Errorf("ToolChoiceMode(%d) is declared but NewToolChoice rejects it for the wrong reason: %v", uint8(mode), freeErr)
			}
		} else if payloadFree.Mode() != mode {
			t.Errorf("ToolChoiceMode(%d): NewToolChoice yielded mode %v, want %v", uint8(mode), payloadFree.Mode(), mode)
		}

		// 4. A member that needs a name is reachable through the named
		//    constructor, and carries the name it was given.
		if needsName {
			named, err := ai.NewNamedToolChoice("probe_tool")
			if err != nil {
				t.Errorf("ToolChoiceMode(%d) needs a name but NewNamedToolChoice returned %v", uint8(mode), err)
				continue
			}
			if named.Mode() != mode {
				t.Errorf("ToolChoiceMode(%d) needs a name but no constructor yields it; NewNamedToolChoice yielded %v", uint8(mode), named.Mode())
			}
			if got, ok := named.Name(); !ok || got != "probe_tool" {
				t.Errorf("ToolChoiceMode(%d): Name() = (%q, %t), want (%q, true)", uint8(mode), got, ok, "probe_tool")
			}
		}
	}
}

// AI-08.3 *(appended, pin)* — no input causes a panic anywhere in the tool
// contract.
//
// Appended during the change and green from birth. Validation is total over its
// inputs: every exported entry point of this milestone returns a value or a
// caller-contract failure, never a crash. It matters here more than it looks,
// because three of these types have a zero value a caller can obtain without
// any constructor — Tool{}, ToolSet{} and ToolChoice{} — and AI-10 will hand
// them to a request that validates before any I/O. A panic there would be a
// caller-contract failure escalated into a process failure.
func TestToolContract_NoInputCausesAPanic(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("a", 1<<16)
	binary := []byte{0x00, 0xff, 0xfe, 0x00}

	var zeroTool ai.Tool
	var zeroSet ai.ToolSet
	var zeroChoice ai.ToolChoice

	probes := []struct {
		label string
		run   func()
	}{
		{"the zero declaration reads", func() {
			_, _, _ = zeroTool.Name(), zeroTool.Description(), zeroTool.Schema()
		}},
		{"a huge name and binary schema", func() {
			tool, _ := ai.NewTool(huge, huge, binary)
			_ = tool.Schema()
		}},
		{"a nil schema", func() { _, _ = ai.NewTool("probe", "", nil) }},
		{"the zero set reads", func() {
			_, _, _ = zeroSet.Tools(), zeroSet.Len(), zeroSet.Declares(huge)
		}},
		{"a set of zero declarations", func() { _, _ = ai.NewToolSet(ai.Tool{}, ai.Tool{}, ai.Tool{}) }},
		{"a set built from nothing", func() { _, _ = ai.NewToolSet() }},
		{"the zero choice reads and validates", func() {
			_ = zeroChoice.Mode()
			_, _ = zeroChoice.Name()
			_ = zeroChoice.ValidateAgainst(zeroSet)
		}},
		{"every mode value the underlying type can hold", func() {
			for m := 0; m <= 255; m++ {
				mode := ai.ToolChoiceMode(m)
				_ = mode.String()
				choice, _ := ai.NewToolChoice(mode)
				_ = choice.ValidateAgainst(zeroSet)
			}
		}},
		{"a huge and a binary name through the parser", func() {
			_, _ = ai.ParseToolChoiceMode(huge)
			_, _ = ai.ParseToolChoiceMode(string(binary))
		}},
		{"a named choice against the zero set", func() {
			choice, _ := ai.NewNamedToolChoice(huge)
			_ = choice.ValidateAgainst(zeroSet)
		}},
	}

	for _, probe := range probes {
		t.Run(probe.label, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("panicked: %v", recovered)
				}
			}()
			probe.run()
		})
	}
}
