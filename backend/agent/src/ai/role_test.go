// Tests for AI-05.1 — the closed role vocabulary.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
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

// newPart builds one element of a message's ordered content, from outside
// package ai, for tests that need content and do not care what it holds.
//
// Until AI-06 it built a struct that embedded the ai.Content interface, which
// was the only route an external package had and was the bypass message.go's own
// documentation recorded. AI-06 closed the seam: message content is now a
// concrete opaque type, embedding it yields a different type that cannot be
// offered at all, and the only door in is a constructor. So the helper calls
// one.
//
// It panics rather than returning an error because every caller below passes a
// label that satisfies the text rules; a panic here means the helper was
// misused, not that a test's subject failed.
func newPart(label string) ai.Part {
	p, err := ai.NewText(label)
	if err != nil {
		panic("newPart(" + label + "): " + err.Error())
	}
	return p
}

// AI-05.1 — a message is constructible with each vocabulary role, and the role
// reads back exactly.
//
// This is the milestone's walking skeleton: the thinnest end-to-end path
// through the public surface. Every later leaf widens it.
func TestMessage_EachVocabularyRole_ConstructsAndReadsBackExactly(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role ai.Role
	}{
		{"user", ai.RoleUser},
		{"assistant", ai.RoleAssistant},
		{"tool", ai.RoleTool},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			msg, err := ai.NewMessage(tc.role, newPart("hello"))
			if err != nil {
				t.Fatalf("NewMessage(%v) returned %v, want no failure", tc.role, err)
			}
			if got := msg.Role(); got != tc.role {
				t.Errorf("msg.Role() = %v, want %v", got, tc.role)
			}
		})
	}
}

// AI-05.1 — a role outside the vocabulary fails construction with an AI-04
// sentinel.
//
// V-REQ-01 makes the vocabulary closed, not advisory: "a value outside the
// vocabulary is a caller-contract failure, not an unknown passed through". The
// zero role is in the table on purpose — a Role field nobody set must be
// rejected exactly like a wild one, which is why the vocabulary starts at
// iota + 1.
func TestMessage_RoleOutsideTheVocabulary_FailsWithTheClosedVocabularySentinel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		role ai.Role
	}{
		{"the zero role", ai.Role(0)},
		{"one past the last declared role", ai.Role(4)},
		{"far outside the declared range", ai.Role(200)},
		{"the maximum of the underlying type", ai.Role(255)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			msg, err := ai.NewMessage(tc.role, newPart("hello"))
			if err == nil {
				t.Fatalf("NewMessage(%v) returned no failure, want one", tc.role)
			}
			if !errors.Is(err, ai.ErrNotInVocabulary) {
				t.Errorf("errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", err)
			}
			if errors.Is(err, ai.ErrEmpty) {
				t.Errorf("errors.Is(err, ErrEmpty) = true, want false (err = %v)", err)
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			path := violation.Path()
			if len(path) != 1 || path[0].Name() != "role" {
				t.Errorf("violation.Path() = %q, want a single step named %q", path.String(), "role")
			}

			if got := msg.Role(); got != 0 {
				t.Errorf("the returned message has role %v, want the zero message", got)
			}
		})
	}
}

// AI-05.1 — each role's string form is stable, lowercase, and round-trips
// through parse-and-render.
//
// Parsing is exact. V-REQ-01's last clause is "Not a provider's own role string
// — an adapter maps between them", so a case variant, a padded form or a
// vendor's own spelling is rejected here and mapped there. The empty string is
// a different fact from an unrecognised one and reports a different sentinel,
// in AI-04's documented order.
func TestRole_StringForm_RoundTripsThroughParseAndRender(t *testing.T) {
	t.Parallel()

	t.Run("every member renders lowercase and parses back", func(t *testing.T) {
		t.Parallel()

		for _, want := range []ai.Role{ai.RoleUser, ai.RoleAssistant, ai.RoleTool} {
			name := want.String()
			if name == "" {
				t.Errorf("Role(%d).String() = %q, want a non-empty rendering", uint8(want), name)
				continue
			}
			if name != strings.ToLower(name) {
				t.Errorf("Role(%d).String() = %q, want it to be lowercase", uint8(want), name)
			}

			got, err := ai.ParseRole(name)
			if err != nil {
				t.Errorf("ParseRole(%q) returned %v, want no failure", name, err)
				continue
			}
			if got != want {
				t.Errorf("ParseRole(%q) = %v, want %v", name, got, want)
			}
		}
	})

	t.Run("only the exact rendering parses", func(t *testing.T) {
		t.Parallel()

		rejected := []string{
			"User", "USER", "Assistant", "TOOL",
			" user", "user ", "\tuser", "user\n",
			"system", "developer", "human", "model",
			"role(4)", "role(0)", "1", "user,assistant",
		}

		for _, name := range rejected {
			got, err := ai.ParseRole(name)
			if err == nil {
				t.Errorf("ParseRole(%q) = %v with no failure, want a failure", name, got)
				continue
			}
			if !errors.Is(err, ai.ErrNotInVocabulary) {
				t.Errorf("ParseRole(%q): errors.Is(err, ErrNotInVocabulary) = false, want true (err = %v)", name, err)
			}
			if got != 0 {
				t.Errorf("ParseRole(%q) = %v on failure, want the zero role", name, got)
			}
		}
	})

	t.Run("the empty string is empty, not unrecognised", func(t *testing.T) {
		t.Parallel()

		_, err := ai.ParseRole("")
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("ParseRole(\"\"): errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
		}
		if errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("ParseRole(\"\"): errors.Is(err, ErrNotInVocabulary) = true, want false (err = %v)", err)
		}
	})

	t.Run("a non-member renders diagnostically and does not parse back", func(t *testing.T) {
		t.Parallel()

		for _, r := range []ai.Role{0, 4, 200, 255} {
			name := r.String()
			if !strings.Contains(name, strconv.Itoa(int(r))) {
				t.Errorf("Role(%d).String() = %q, want it to identify the value", uint8(r), name)
			}
			if _, err := ai.ParseRole(name); err == nil {
				t.Errorf("ParseRole(%q) returned no failure, want one — a diagnostic rendering must not round-trip", name)
			}
		}
	})

	t.Run("the offered string never reaches the failure", func(t *testing.T) {
		t.Parallel()

		const sentinelBody = "CACHICAMA-SENTINEL-BODY-a71c"

		_, err := ai.ParseRole(sentinelBody)
		if err == nil {
			t.Fatalf("ParseRole(%q) returned no failure, want one", sentinelBody)
		}
		if strings.Contains(err.Error(), sentinelBody) {
			t.Errorf("Error() = %q, want it to carry none of the offered string", err.Error())
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, *ai.Violation) = false, want true")
		}
		if path := violation.Path().String(); path != "role" {
			t.Errorf("violation.Path() = %q, want %q", path, "role")
		}
	})
}

// AI-05.1 *(pin)* — every declared role is exhaustively tabulated.
//
// Green from birth and exempt from red-first per doc 0002's leaf anatomy. It
// bites when a member is declared without a table entry, and the reason it can
// is design.md § 3 rule 3: Roles enumerates the *constant space*, not the name
// table. An enumeration derived from the table would list exactly the members
// that have an entry, so the omission it exists to catch would be invisible to
// it.
//
// This is the pattern every later closed vocabulary in this package reuses —
// AI-07 reasoning state, AI-08 tool choice, AI-13 finish reasons.
func TestRole_DeclaredVocabulary_IsExhaustivelyTabulated(t *testing.T) {
	t.Parallel()

	roles := ai.Roles()
	if len(roles) == 0 {
		t.Fatal("Roles() returned nothing; the pin would pass vacuously")
	}

	if again := ai.Roles(); !slices.Equal(roles, again) {
		t.Errorf("Roles() = %v then %v, want a stable declaration order", roles, again)
	}

	for _, r := range roles {
		name := r.String()

		if name == "" || strings.HasPrefix(name, "role(") {
			t.Errorf("Role(%d) has no table entry: String() = %q", uint8(r), name)
		}
		if name != strings.ToLower(name) {
			t.Errorf("Role(%d).String() = %q, want it to be lowercase", uint8(r), name)
		}

		parsed, err := ai.ParseRole(name)
		if err != nil {
			t.Errorf("Role(%d): ParseRole(%q) returned %v, want it to round-trip", uint8(r), name, err)
		} else if parsed != r {
			t.Errorf("Role(%d): ParseRole(%q) = %v, want %v", uint8(r), name, parsed, r)
		}

		if _, err := ai.NewMessage(r, newPart("hello")); err != nil {
			t.Errorf("Role(%d) is declared but NewMessage rejects it: %v", uint8(r), err)
		}
	}
}
