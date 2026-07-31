// Tests for AI-06.2 and AI-06.3 — the content part, readable and sealed.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
// surface a consumer in another package sees. The cross-package round trip that
// AI-06.2 item 1 asks for lives one directory over, in src/agenttest, because
// that package's own documentation reserves it.
package ai_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-06.2 item 2 — the discriminator and the payload agree.
//
// They agree because there is only one of them: [ai.Part] stores no kind field
// and derives it from the payload, so the state a naive design has to test for
// — a part labelled text carrying something else, or labelled text carrying
// nothing — is not a state the type can be in. This test is the pin over that
// property rather than the proof of it: it fails the day someone reintroduces a
// kind field and lets the two drift.
func TestPart_KindAndPayload_Agree(t *testing.T) {
	t.Parallel()

	t.Run("every registered kind reports itself and yields its payload", func(t *testing.T) {
		t.Parallel()

		for _, kind := range ai.PartKinds() {
			switch kind {
			case ai.PartKindText:
				part, err := ai.NewText("model-visible text")
				if err != nil {
					t.Fatalf("ai.NewText returned %v, want no failure", err)
				}
				if got := part.Kind(); got != kind {
					t.Errorf("part.Kind() = %v, want %v", got, kind)
				}
				text, ok := part.Text()
				if !ok {
					t.Errorf("part.Text() reported no text on a part of kind %v", kind)
				}
				if text == "" {
					t.Errorf("part.Text() = %q on a part of kind %v, want the payload it was built from", text, kind)
				}
			case ai.PartKindToolCall:
				part, err := ai.NewToolCall("toolu_01A09Kf7", "read_file", []byte(`{"path":"/etc/hosts"}`))
				if err != nil {
					t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
				}
				if got := part.Kind(); got != kind {
					t.Errorf("part.Kind() = %v, want %v", got, kind)
				}
				call, ok := part.ToolCall()
				if !ok {
					t.Errorf("part.ToolCall() reported no tool call on a part of kind %v", kind)
				}
				if call.ID() == "" {
					t.Errorf("part.ToolCall().ID() is empty on a part of kind %v, want the payload it was built from", kind)
				}
			case ai.PartKindToolResult:
				part, err := ai.NewToolResult("toolu_01A09Kf7", "127.0.0.1\tlocalhost\n")
				if err != nil {
					t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
				}
				if got := part.Kind(); got != kind {
					t.Errorf("part.Kind() = %v, want %v", got, kind)
				}
				result, ok := part.ToolResult()
				if !ok {
					t.Errorf("part.ToolResult() reported no tool result on a part of kind %v", kind)
				}
				if result.CallID() == "" {
					t.Errorf("part.ToolResult().CallID() is empty on a part of kind %v, want the payload it was built from", kind)
				}
			default:
				t.Errorf("kind %v is registered but this test does not exercise it — AI-06.4 mechanizes what this default catches by hand", kind)
			}
		}
	})

	t.Run("a part that carries no payload reports no kind", func(t *testing.T) {
		t.Parallel()

		var unconstructed ai.Part

		if got := unconstructed.Kind(); got != 0 {
			t.Errorf("the zero Part reports kind %v, want the zero kind", got)
		}
		if slicesContains(ai.PartKinds(), unconstructed.Kind()) {
			t.Errorf("the zero Part's kind %v is a member of the vocabulary, want a non-member", unconstructed.Kind())
		}
		if text, ok := unconstructed.Text(); ok || text != "" {
			t.Errorf("zeroPart.Text() = (%q, %t), want (%q, false)", text, ok, "")
		}
	})

	t.Run("the kind survives copying, because it is derived and not stored", func(t *testing.T) {
		t.Parallel()

		part, err := ai.NewText("model-visible text")
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}

		copied := part
		if copied.Kind() != part.Kind() {
			t.Errorf("copied.Kind() = %v, want %v", copied.Kind(), part.Kind())
		}
		if copied != part {
			t.Errorf("a copied part is not equal to its original")
		}

		original, _ := part.Text()
		duplicate, _ := copied.Text()
		if original != duplicate {
			t.Errorf("copied.Text() = %q, want %q", duplicate, original)
		}
	})
}

// AI-06.2 item 3 — the construction rules fail through AI-04's sentinels.
//
// Three rule violations, two rule classes, one position. The order is contract
// per V-FAIL-04 and is asserted by the last subtest: a string that is both
// whitespace-only and over the bound reports emptiness, because "you gave me
// nothing" and "you gave me too much" are different facts and a caller fixing
// one at a time must make progress in a predictable direction.
func TestNewText_RuleViolations_FailWithTheDocumentedSentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want error
	}{
		{"an empty string", "", ai.ErrEmpty},
		{"a single space", " ", ai.ErrEmpty},
		{"tabs and newlines", "\t\n\r\n\t", ai.ErrEmpty},
		{"non-ASCII whitespace", "  　", ai.ErrEmpty},
		{"one byte over the documented bound", strings.Repeat("x", ai.MaxTextLen+1), ai.ErrOutOfRange},
		{"far over the documented bound", strings.Repeat("x", ai.MaxTextLen*2), ai.ErrOutOfRange},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			part, err := ai.NewText(tc.text)
			if err == nil {
				t.Fatalf("ai.NewText returned no failure, want one")
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("errors.Is(err, %v) = false, want true (err = %v)", tc.want, err)
			}
			for _, other := range []error{ai.ErrEmpty, ai.ErrNotInVocabulary, ai.ErrOutOfRange, ai.ErrMalformed, ai.ErrUnresolvedReference} {
				if other == tc.want {
					continue
				}
				if errors.Is(err, other) {
					t.Errorf("errors.Is(err, %v) = true, want false — a failure names one rule class (err = %v)", other, err)
				}
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
			}
			if path := violation.Path().String(); path != "text" {
				t.Errorf("violation.Path() = %q, want %q", path, "text")
			}

			// A failed construction yields nothing usable: the returned part is
			// the zero value, so a caller that ignored the failure cannot
			// mistake it for a constructed part. That such a part is also
			// refused by a message is the seal, and it is AI-06.3's — see
			// TestMessage_UnconstructedContentElement_IsRejected.
			if part.Kind() != 0 {
				t.Errorf("a failed ai.NewText returned a part of kind %v, want the zero part", part.Kind())
			}
			if _, ok := part.Text(); ok {
				t.Error("a failed ai.NewText returned a part that yields text, want the zero part")
			}
		})
	}

	t.Run("the documented bound is exact and exported", func(t *testing.T) {
		t.Parallel()

		if _, err := ai.NewText(strings.Repeat("x", ai.MaxTextLen)); err != nil {
			t.Errorf("ai.NewText at exactly MaxTextLen returned %v, want no failure", err)
		}
		if _, err := ai.NewText(strings.Repeat("x", ai.MaxTextLen-1)); err != nil {
			t.Errorf("ai.NewText one byte under MaxTextLen returned %v, want no failure", err)
		}
	})

	t.Run("both rules violated reports the first in the documented order", func(t *testing.T) {
		t.Parallel()

		both := strings.Repeat(" ", ai.MaxTextLen+1)

		var first string
		for run := range 128 {
			_, err := ai.NewText(both)
			if err == nil {
				t.Fatalf("run %d: ai.NewText returned no failure, want one", run)
			}
			if run == 0 {
				first = err.Error()
				if !errors.Is(err, ai.ErrEmpty) {
					t.Fatalf("errors.Is(err, ErrEmpty) = false, want true — emptiness is the first rule (err = %v)", err)
				}
				continue
			}
			if err.Error() != first {
				t.Fatalf("run %d reported %q, want %q on every run", run, err.Error(), first)
			}
		}
	})

	t.Run("the failure never reproduces the text it was given", func(t *testing.T) {
		t.Parallel()

		const distinctive = "sk-live-9f3a-CANARY-do-not-log"

		_, err := ai.NewText(strings.Repeat(distinctive, ai.MaxTextLen/len(distinctive)+1))
		if err == nil {
			t.Fatal("ai.NewText returned no failure, want one")
		}
		if strings.Contains(err.Error(), "CANARY") {
			t.Errorf("the rendered failure reproduces the offered text: %q", err.Error())
		}
	})
}

// AI-06.2 item 4 — valid text survives construction unaltered.
//
// The rule that rejects whitespace-only text inspects the text; it must not
// rewrite it. An implementation that stored strings.TrimSpace(text) would pass
// every assertion of item 3 and fail the first subtest here, which is why the
// case exists.
func TestNewText_ValidText_SurvivesConstructionUnaltered(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
	}{
		{"surrounding whitespace around content", "  \t\nhello\n\t  "},
		{"embedded newlines and carriage returns", "one\ntwo\r\nthree\r"},
		{"astral-plane runes", "\U0001F9EA\U0001F30D\U000E0021"},
		{"combining marks", "ȩ́ ä"},
		// The bidi controls are written as escapes rather than as literal
		// characters: a source file whose visible order differs from its byte
		// order is the Trojan Source hazard, and a test about bytes must not
		// rely on a reader trusting what the editor renders.
		{"right-to-left text", "\u202bمرحبا بالعالم\u202c"},
		{"a lone surrogate byte sequence that is not valid UTF-8", "valid \xed\xa0\x80 tail"},
		{"an embedded NUL", "before\x00after"},
		{"content that looks like markup", "<part kind=\"text\">&amp;</part>"},
		{"content that looks like a rendered violation", "messages[2].content[0]: required value is empty"},
		{"every byte value from 1 to 255", allBytesFrom(1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			part, err := ai.NewText(tc.text)
			if err != nil {
				t.Fatalf("ai.NewText returned %v, want no failure", err)
			}

			got, ok := part.Text()
			if !ok {
				t.Fatalf("part.Text() reported no text")
			}
			if got != tc.text {
				t.Errorf("part.Text() = %q, want %q", got, tc.text)
			}
			if len(got) != len(tc.text) {
				t.Errorf("len(part.Text()) = %d, want %d — the payload was re-encoded", len(got), len(tc.text))
			}

			// And again through a message, because the round trip is where an
			// accidental normalization would actually bite.
			msg, err := ai.NewMessage(ai.RoleAssistant, part)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v, want no failure", err)
			}
			viaMessage, ok := msg.Content()[0].Text()
			if !ok || viaMessage != tc.text {
				t.Errorf("msg.Content()[0].Text() = (%q, %t), want (%q, true)", viaMessage, ok, tc.text)
			}
		})
	}
}

// AI-06.2 *(appended)* — the kind vocabulary is closed, stable and immutable.
//
// Appended to AI-06.2's list during implementation, per doc 0002's rule that a
// discovered case is appended to the owning leaf's list rather than chased ad
// hoc. AI-05 landed this shape for roles and role.go calls it "the pattern every
// later closed vocabulary in this package reuses"; the kind vocabulary is its
// second instance, and the pattern is only inherited if it is asserted.
func TestPartKinds_Vocabulary_IsClosedStableAndImmutable(t *testing.T) {
	t.Parallel()

	t.Run("the enumeration is stable across calls", func(t *testing.T) {
		t.Parallel()

		first, second := ai.PartKinds(), ai.PartKinds()
		if len(first) == 0 {
			t.Fatal("ai.PartKinds() returned nothing")
		}
		if !reflect.DeepEqual(first, second) {
			t.Errorf("ai.PartKinds() = %v then %v, want an identical order on every call", first, second)
		}
	})

	t.Run("a consumer cannot rewrite the vocabulary", func(t *testing.T) {
		t.Parallel()

		kinds := ai.PartKinds()
		original := kinds[0]
		kinds[0] = ai.PartKind(200)

		if again := ai.PartKinds(); again[0] != original {
			t.Errorf("ai.PartKinds()[0] = %v after a consumer rewrote its copy, want %v", again[0], original)
		}
	})

	t.Run("every member renders as a stable lowercase name", func(t *testing.T) {
		t.Parallel()

		for _, kind := range ai.PartKinds() {
			name := kind.String()
			if name == "" {
				t.Errorf("kind %d renders as the empty string", uint8(kind))
			}
			if name != strings.ToLower(name) {
				t.Errorf("kind %d renders as %q, want a lowercase name", uint8(kind), name)
			}
			if strings.ContainsAny(name, "() ") {
				t.Errorf("kind %d renders as %q, which is the shape reserved for a non-member", uint8(kind), name)
			}
		}
	})

	t.Run("a non-member renders as a non-member", func(t *testing.T) {
		t.Parallel()

		if got := ai.PartKind(0).String(); got != "unset" {
			t.Errorf("ai.PartKind(0).String() = %q, want %q", got, "unset")
		}
		for _, k := range []ai.PartKind{200, 255} {
			got := k.String()
			if !strings.HasPrefix(got, "partkind(") {
				t.Errorf("ai.PartKind(%d).String() = %q, want a diagnostic rendering", uint8(k), got)
			}
			if slicesContains(ai.PartKinds(), k) {
				t.Errorf("ai.PartKind(%d) is a member of the vocabulary, want a non-member", uint8(k))
			}
		}
	})
}

// AI-06.2 *(appended)* — a part's diagnostic rendering carries no payload.
//
// validation.go records that a validation failure is "the first thing in this
// package that formats caller data". A content part is the second, and its
// default is worse: fmt prints the unexported fields of a struct it has no
// String method for. This is V-FAIL-13's redaction posture extended to the first
// payload-carrying value type in the package, and it was appended to AI-06.2's
// list when writing the item-1 test showed %v printing the payload.
func TestPart_String_CarriesNoPayload(t *testing.T) {
	t.Parallel()

	const canary = "sk-live-9f3a-CANARY-do-not-log"

	part, err := ai.NewText(canary)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}

	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		rendered := fmt.Sprintf(verb, part)
		if strings.Contains(rendered, "CANARY") {
			t.Errorf("fmt.Sprintf(%q, part) = %q, which reproduces the payload", verb, rendered)
		}
	}

	if got := fmt.Sprintf("%v", part); got != "part(text)" {
		t.Errorf("fmt.Sprintf(\"%%v\", part) = %q, want %q", got, "part(text)")
	}

	var unconstructed ai.Part
	if got := fmt.Sprintf("%v", unconstructed); got != "part(unset)" {
		t.Errorf("fmt.Sprintf(\"%%v\", the zero Part) = %q, want %q", got, "part(unset)")
	}
}

// allBytesFrom builds a string holding every byte value from low to 255.
//
// It is deliberately not valid UTF-8: V-REQ-08 says text, and a contract that
// only survives well-formed input is one that re-encodes.
func allBytesFrom(low byte) string {
	out := make([]byte, 0, 256-int(low))
	for b := int(low); b < 256; b++ {
		out = append(out, byte(b))
	}
	return string(out)
}

// slicesContains reports whether the slice holds the value. It exists so the
// assertions above read as sentences; slices.Contains would do, and the import
// is not worth the one call site it would serve here.
func slicesContains(kinds []ai.PartKind, want ai.PartKind) bool {
	for _, k := range kinds {
		if k == want {
			return true
		}
	}
	return false
}

// AI-06.3 item 1 — a value that skipped construction cannot reach a message.
//
// This is doc 0001's defect C1, asserted before a second variant exists to
// inherit it. In the retired Layer 1 "the exported text value type satisfies the
// part interface directly, so its zero value is a valid part that passes message
// validation and bypasses every construction rule". Here the zero part has no
// payload, therefore no kind, therefore no membership of the closed kind
// vocabulary — which is why the failure is [ai.ErrNotInVocabulary] and not a
// sentinel of its own.
func TestMessage_UnconstructedContentElement_IsRejected(t *testing.T) {
	t.Parallel()

	valid := func(t *testing.T, text string) ai.Part {
		t.Helper()
		part, err := ai.NewText(text)
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}
		return part
	}

	t.Run("the zero part is rejected, positioned by ordinal", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name     string
			content  func(t *testing.T) []ai.Part
			wantPath string
		}{
			{"the only element", func(*testing.T) []ai.Part {
				return []ai.Part{{}}
			}, "content[0]"},
			{"the third of four", func(t *testing.T) []ai.Part {
				return []ai.Part{valid(t, "a"), valid(t, "b"), {}, valid(t, "d")}
			}, "content[2]"},
			{"the last element", func(t *testing.T) []ai.Part {
				return []ai.Part{valid(t, "a"), {}}
			}, "content[1]"},
			{"several unconstructed elements report the first", func(t *testing.T) []ai.Part {
				return []ai.Part{valid(t, "a"), {}, {}, {}}
			}, "content[1]"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				msg, err := ai.NewMessage(ai.RoleUser, tc.content(t)...)
				if err == nil {
					t.Fatal("ai.NewMessage returned no failure, want one")
				}
				if !errors.Is(err, ai.ErrNotInVocabulary) {
					t.Errorf("errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", err)
				}
				if errors.Is(err, ai.ErrEmpty) {
					t.Errorf("errors.Is(err, ErrEmpty) = true, want false (err = %v)", err)
				}

				var violation *ai.Violation
				if !errors.As(err, &violation) {
					t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
				}
				if path := violation.Path().String(); path != tc.wantPath {
					t.Errorf("violation.Path() = %q, want %q", path, tc.wantPath)
				}
				if !msg.ID().IsZero() {
					t.Errorf("the rejected construction returned a message with identity %v, want none", msg.ID())
				}
			})
		}
	})

	t.Run("a part promoted out of an embedding type is the zero part", func(t *testing.T) {
		t.Parallel()

		// This is the shape AI-05's own tests used, and the one message.go's
		// documentation recorded as the open door. It no longer satisfies
		// anything — embedded is now a distinct type that ai.NewMessage will not
		// take — so the only thing an author of it can still offer is the
		// promoted field, which is the zero part.
		type embedded struct {
			ai.Part

			label string
		}

		smuggled := embedded{label: "looks like a part"}

		if smuggled.Kind() != 0 {
			t.Errorf("the promoted part reports kind %v, want the zero kind", smuggled.Kind())
		}
		if _, err := ai.NewMessage(ai.RoleUser, smuggled.Part); err == nil {
			t.Fatal("ai.NewMessage accepted a promoted zero part, want a rejection")
		}
	})

	t.Run("the role rule still wins over the element rule", func(t *testing.T) {
		t.Parallel()

		// V-FAIL-04: the documented order is role, then emptiness, then the
		// elements. A construction breaking the first and the third reports the
		// first, identically on every run.
		for run := range 64 {
			_, err := ai.NewMessage(ai.Role(0), ai.Part{})
			if err == nil {
				t.Fatalf("run %d: ai.NewMessage returned no failure, want one", run)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("run %d: errors.As(err, *ai.Violation) = false, want true", run)
			}
			if path := violation.Path().String(); path != "role" {
				t.Fatalf("run %d: violation.Path() = %q, want %q — the role rule is first", run, path, "role")
			}
		}
	})

	t.Run("a message that was constructed holds only valid parts", func(t *testing.T) {
		t.Parallel()

		msg, err := ai.NewMessage(ai.RoleUser, valid(t, "a"), valid(t, "b"))
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}
		for i, part := range msg.Content() {
			if part.Kind() == 0 {
				t.Errorf("msg.Content()[%d] carries no kind, want a constructed part", i)
			}
		}
	})
}

// AI-06.3 item 2 — an external package cannot implement the part contract, and
// the proof is the compiler's.
//
// The seal is two mechanisms with a clean boundary between them, and this is the
// half that no in-process assertion can reach. decision.md § 3.3 works eight
// bypasses; five of them are answered by the language rather than by a check
// this package wrote, and a language rule is observable only by asking the
// toolchain to compile something and watching it refuse. AI-00's import guard
// established the precedent for shelling out — a property of the build graph is
// not visible from inside the test binary — and this is the same argument one
// level down, about the type graph.
//
// The other half — the one value external code can still produce, the zero
// Part — is rejected at runtime and is proven by
// TestMessage_UnconstructedContentElement_IsRejected.
//
// # Why there are two programs
//
// A test that asserts "this does not compile" is satisfied by every reason a
// build can fail: a misspelled import path, an unresolvable module, a broken
// package ai, an absent toolchain. testdata/constructed is the control. It is
// written against the same package through the same import path from the same
// directory and it must build, so a hand-rolled failure means the seal and not
// the weather.
//
// # Why testdata
//
// The go tool and golangci-lint both exclude a directory named testdata from
// package patterns, so a program that must not compile is not part of
// `go build ./...`, `make test`'s package set, or `make lint`.
func TestPart_HandRolledFromAnotherPackage_DoesNotCompile(t *testing.T) {
	t.Parallel()

	// Each entry is one row of decision.md § 3.3, named by its bypass number,
	// with the diagnostic the toolchain must produce for it. The substrings are
	// matched, not the line numbers: the fixture is allowed to move.
	bypasses := []struct {
		bypass     int
		attempt    string
		diagnostic string
	}{
		{2, "a named composite literal", "cannot refer to unexported field payload in struct literal of type ai.Part"},
		{3, "a positional composite literal", "implicit assignment to unexported field payload in struct literal of type ai.Part"},
		{4, "offering a type that embeds Part as message content", "as ai.Part value in argument to ai.NewMessage"},
		{6, "naming the payload contract", "name partPayload not exported by package ai"},
		{6, "implementing the payload contract", "does not implement ai.partPayload (unexported method kind)"},
		{7, "assigning the field on a Part value", "p.payload undefined (cannot refer to unexported field payload)"},
	}

	t.Run("the control program builds", func(t *testing.T) {
		t.Parallel()

		if out, err := buildTestdataProgram(t, "constructed"); err != nil {
			t.Fatalf("testdata/constructed must build, but the toolchain refused it.\n"+
				"Until it does, a failure from testdata/handrolled proves nothing about the seal.\n%v\n%s", err, out)
		}
	})

	t.Run("the hand-rolled program does not build", func(t *testing.T) {
		t.Parallel()

		out, err := buildTestdataProgram(t, "handrolled")
		if err == nil {
			t.Fatalf("testdata/handrolled compiled, want a build failure.\n"+
				"Every bypass in decision.md § 3.3 that the compiler is supposed to answer is now open.\n%s", out)
		}

		for _, b := range bypasses {
			if !strings.Contains(out, b.diagnostic) {
				t.Errorf("bypass %d (%s): the build output does not carry the expected diagnostic.\n"+
					"  want substring: %s\n  got:\n%s", b.bypass, b.attempt, b.diagnostic, out)
			}
		}
	})
}

// AI-06.3 item 2 *(appended)* — reflection is not a way in either.
//
// Bypass 7 of decision.md § 3.3 is answered by the language and not by a check
// this package wrote, and unlike bypasses 2, 3, 4 and 6 it is answerable from
// inside the test binary: reflect refuses to Set an unexported field, and refuses
// to read one into an interface. Appended to AI-06.3's list when writing the
// compile proof made it obvious that the runtime half of bypass 7 had no home.
//
// The exported-field count is the assertion that fails the day someone widens
// the struct, which is the only edit that would make every other assertion here
// vacuous.
func TestPart_ExportedSurface_ExposesNoConstructibleState(t *testing.T) {
	t.Parallel()

	partType := reflect.TypeOf(ai.Part{})

	t.Run("the struct declares exactly one field, and it is unexported", func(t *testing.T) {
		t.Parallel()

		if got := partType.NumField(); got != 1 {
			t.Fatalf("reflect.TypeOf(ai.Part{}).NumField() = %d, want 1", got)
		}
		if field := partType.Field(0); field.IsExported() {
			t.Errorf("ai.Part declares the exported field %q; the seal is a property of the field being unexported", field.Name)
		}
	})

	t.Run("reflection cannot write the payload", func(t *testing.T) {
		t.Parallel()

		valid, err := ai.NewText("constructed the only way in")
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}

		target := valid
		field := reflect.ValueOf(&target).Elem().Field(0)
		if field.CanSet() {
			t.Fatal("reflect reports the payload field as settable from another package, want CanSet() = false")
		}
		if field.CanInterface() {
			t.Error("reflect reports the payload field as readable into an interface, want CanInterface() = false")
		}

		// The part is unchanged, and still the one NewText built.
		if target != valid {
			t.Error("the part changed under a reflection attempt, want it untouched")
		}
	})

	t.Run("a Part value carries no exported method that sets anything", func(t *testing.T) {
		t.Parallel()

		// Every exported method must be a reader: no arguments, and results
		// only. A setter would be construction wearing a method name.
		for i := range partType.NumMethod() {
			method := partType.Method(i)
			if method.Type.NumIn() != 1 {
				t.Errorf("ai.Part.%s takes an argument; every exported method on a sealed value type is a reader", method.Name)
			}
			if method.Type.NumOut() == 0 {
				t.Errorf("ai.Part.%s returns nothing, so it exists to mutate", method.Name)
			}
		}
	})
}

// buildTestdataProgram compiles one program under testdata and returns the
// toolchain's combined output.
//
// The output is discarded with -o os.DevNull: what is under test is whether the
// build succeeds and what it says, never the binary. `go test` runs with the
// working directory set to the package under test, so the relative path resolves
// against src/ai — the same assumption AI-00's import guard documents, from the
// other direction.
func buildTestdataProgram(t *testing.T, name string) (string, error) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", os.DevNull, "./testdata/"+name)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
