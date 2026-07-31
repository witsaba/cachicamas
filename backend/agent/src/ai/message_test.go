// Tests for AI-05.2 and AI-05.3 — message identity, ordered content, and copy
// semantics.
//
// The package under test is imported by its module path from the external test
// package ai_test. The content helper these tests build messages from — part,
// in role_test.go — satisfies the seam by embedding it, which is the only route
// an external package has and is the one the seam's own documentation records.
package ai_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-05.2 — a message carries a stable identity that is comparable and does not
// change across reads.
//
// V-REQ-03 calls identity "the stable handle by which one message is
// distinguished from another". Distinguished is the clause the last two
// subtests are about: two messages built from identical inputs are two
// messages, so an identity derived from content would satisfy "comparable" and
// answer the wrong question.
func TestMessage_Identity_IsStableComparableAndDistinct(t *testing.T) {
	t.Parallel()

	t.Run("repeated reads agree", func(t *testing.T) {
		t.Parallel()

		msg, err := ai.NewMessage(ai.RoleUser, newPart("hello"))
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}

		first := msg.ID()
		if first.IsZero() {
			t.Fatalf("msg.ID() = %v, want a minted identity", first)
		}
		for range 8 {
			if got := msg.ID(); got != first {
				t.Errorf("msg.ID() = %v on a later read, want %v", got, first)
			}
		}
	})

	t.Run("identical messages have different identities", func(t *testing.T) {
		t.Parallel()

		shared := newPart("hello")

		first, err := ai.NewMessage(ai.RoleUser, shared)
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}
		second, err := ai.NewMessage(ai.RoleUser, shared)
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}

		if first.ID() == second.ID() {
			t.Errorf("two messages share the identity %v, want distinct identities", first.ID())
		}
	})

	t.Run("an unconstructed message has no identity", func(t *testing.T) {
		t.Parallel()

		var zero ai.Message
		if !zero.ID().IsZero() {
			t.Errorf("the zero Message has identity %v, want an unset one", zero.ID())
		}

		failed, err := ai.NewMessage(ai.Role(0), newPart("hello"))
		if err == nil {
			t.Fatal("NewMessage(Role(0)) returned no failure, want one")
		}
		if !failed.ID().IsZero() {
			t.Errorf("a failed construction returned identity %v, want an unset one", failed.ID())
		}
	})

	t.Run("concurrent construction mints distinct identities", func(t *testing.T) {
		t.Parallel()

		const goroutines = 64

		ids := make([]ai.MessageID, goroutines)
		var wg sync.WaitGroup
		for i := range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				msg, err := ai.NewMessage(ai.RoleAssistant, newPart("hello"))
				if err != nil {
					t.Errorf("goroutine %d: NewMessage returned %v, want no failure", i, err)
					return
				}
				ids[i] = msg.ID()
			}()
		}
		wg.Wait()

		seen := make(map[ai.MessageID]int, goroutines)
		for i, id := range ids {
			if id.IsZero() {
				t.Errorf("goroutine %d minted an unset identity", i)
				continue
			}
			if previous, duplicate := seen[id]; duplicate {
				t.Errorf("goroutines %d and %d both minted %v", previous, i, id)
			}
			seen[id] = i
		}
	})

	t.Run("identity renders diagnostically", func(t *testing.T) {
		t.Parallel()

		msg, err := ai.NewMessage(ai.RoleUser, newPart("hello"))
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}

		if rendered := msg.ID().String(); !strings.HasPrefix(rendered, "msg") {
			t.Errorf("MessageID.String() = %q, want a diagnostic rendering", rendered)
		}
		var zero ai.MessageID
		if rendered := zero.String(); !strings.HasPrefix(rendered, "msg") {
			t.Errorf("the zero MessageID renders as %q, want a diagnostic rendering", rendered)
		}
	})
}

// AI-05.2 — a message carries ordered content, and the order reads back exactly
// as constructed.
//
// The repeated-element case is the one that matters. A message that stored its
// content in a set, a map keyed by the element, or anything that deduplicates
// passes the distinct-elements case and fails this one, and doing so silently
// is how a repeated tool call or two identical text parts disappear from a
// request.
func TestMessage_ContentOrder_RoundTripsExactly(t *testing.T) {
	t.Parallel()

	a, b, c := newPart("a"), newPart("b"), newPart("c")

	cases := []struct {
		name string
		want []ai.Content
	}{
		{"one element", []ai.Content{a}},
		{"three distinct elements", []ai.Content{a, b, c}},
		{"the same element repeated", []ai.Content{a, a, a}},
		{"repetitions interleaved with distinct elements", []ai.Content{a, b, a, a, c, b}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			msg, err := ai.NewMessage(ai.RoleUser, tc.want...)
			if err != nil {
				t.Fatalf("NewMessage returned %v, want no failure", err)
			}

			got := msg.Content()
			if len(got) != len(tc.want) {
				t.Fatalf("len(msg.Content()) = %d, want %d", len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("msg.Content()[%d] = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// AI-05.2 — a message constructed with no content fails with an AI-04 sentinel.
//
// The three ways of saying "nothing" — no arguments, an empty slice, a nil
// slice — are one fact and report one failure. The last subtest pins V-FAIL-04
// at this contract: a construction that breaks both rules reports the first in
// the documented order, on every run, so a caller fixing one failure at a time
// makes progress in a predictable direction.
func TestMessage_NoContent_FailsWithTheEmptySentinel(t *testing.T) {
	t.Parallel()

	t.Run("the three ways of saying nothing fail identically", func(t *testing.T) {
		t.Parallel()

		var nilContent []ai.Content

		cases := []struct {
			name    string
			content []ai.Content
		}{
			{"no arguments", nil},
			{"an empty slice", []ai.Content{}},
			{"a nil slice", nilContent},
		}

		for _, tc := range cases {
			_, err := ai.NewMessage(ai.RoleUser, tc.content...)
			if err == nil {
				t.Errorf("%s: NewMessage returned no failure, want one", tc.name)
				continue
			}
			if !errors.Is(err, ai.ErrEmpty) {
				t.Errorf("%s: errors.Is(err, ErrEmpty) = false, want true (err = %v)", tc.name, err)
			}
			if errors.Is(err, ai.ErrNotInVocabulary) {
				t.Errorf("%s: errors.Is(err, ErrNotInVocabulary) = true, want false (err = %v)", tc.name, err)
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Errorf("%s: errors.As(err, *ai.Violation) = false, want true", tc.name)
				continue
			}
			if path := violation.Path().String(); path != "content" {
				t.Errorf("%s: violation.Path() = %q, want %q", tc.name, path, "content")
			}
		}
	})

	t.Run("both rules violated reports the first in the documented order", func(t *testing.T) {
		t.Parallel()

		var first string
		for run := range 256 {
			_, err := ai.NewMessage(ai.Role(0))
			if err == nil {
				t.Fatalf("run %d: NewMessage returned no failure, want one", run)
			}
			if run == 0 {
				first = err.Error()
				if !errors.Is(err, ai.ErrNotInVocabulary) {
					t.Fatalf("errors.Is(err, ErrNotInVocabulary) = false, want true — role is the first rule (err = %v)", err)
				}
				continue
			}
			if err.Error() != first {
				t.Fatalf("run %d reported %q, want %q on every run", run, err.Error(), first)
			}
		}
	})
}

// AI-05.3 — mutating the slice a caller passed leaves the constructed message
// unchanged.
//
// The hazard is Go's, not the design's: a variadic parameter called with a
// slice spread does not copy, so NewMessage(role, parts...) aliases the
// caller's backing array unless the constructor clones. Nothing in the language
// warns about it, every happy-path test passes, and the symptom appears later
// as content that changed with nobody writing to the message.
//
// Both subtests mutate through the caller's own handle. Neither appends beyond
// capacity, because a reallocating append hides the defect rather than exposing
// it.
func TestMessage_CallerMutatesWhatItPassed_MessageIsUnchanged(t *testing.T) {
	t.Parallel()

	a, b, c, d := newPart("a"), newPart("b"), newPart("c"), newPart("d")

	t.Run("replacing an element in place", func(t *testing.T) {
		t.Parallel()

		parts := []ai.Content{a, b}
		msg, err := ai.NewMessage(ai.RoleUser, parts...)
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}

		parts[0] = c
		parts[1] = d

		assertContent(t, msg, []ai.Content{a, b})
	})

	t.Run("reusing the backing array for a second message", func(t *testing.T) {
		t.Parallel()

		buf := make([]ai.Content, 0, 4)
		buf = append(buf, a, b)

		first, err := ai.NewMessage(ai.RoleUser, buf...)
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}

		buf = buf[:0]
		buf = append(buf, c, d)

		second, err := ai.NewMessage(ai.RoleAssistant, buf...)
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}

		assertContent(t, first, []ai.Content{a, b})
		assertContent(t, second, []ai.Content{c, d})
	})
}

// assertContent reports every way msg's content differs from want.
func assertContent(t *testing.T, msg ai.Message, want []ai.Content) {
	t.Helper()

	got := msg.Content()
	if len(got) != len(want) {
		t.Fatalf("len(msg.Content()) = %d, want %d (got %v, want %v)", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("msg.Content()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// AI-05.3 — mutating what a read returned leaves the message unchanged.
//
// This is the other half of the copy contract and it is a separate mechanism: a
// constructor that clones and a reader that does not passes every construction
// test and fails the moment two consumers hold the same message. AI-21.6's
// request capture is the first consumer that would meet it, and doc 0002 notes
// this property is what lets that capture assert on history without defensive
// copying at the call site.
//
// Roles is covered here for the same reason: an enumeration of a closed
// vocabulary that a consumer can rewrite is not closed.
func TestMessage_CallerMutatesWhatItRead_MessageIsUnchanged(t *testing.T) {
	t.Parallel()

	a, b, c := newPart("a"), newPart("b"), newPart("c")

	newFixture := func(t *testing.T) ai.Message {
		t.Helper()
		msg, err := ai.NewMessage(ai.RoleUser, a, b)
		if err != nil {
			t.Fatalf("NewMessage returned %v, want no failure", err)
		}
		return msg
	}

	t.Run("a reader cannot rewrite the message", func(t *testing.T) {
		t.Parallel()

		msg := newFixture(t)

		read := msg.Content()
		read[0] = c

		assertContent(t, msg, []ai.Content{a, b})
	})

	t.Run("two readers are independent", func(t *testing.T) {
		t.Parallel()

		msg := newFixture(t)

		first, second := msg.Content(), msg.Content()
		first[0] = c

		if second[0] != a {
			t.Errorf("the second reader saw %v at index 0, want %v", second[0], a)
		}
		assertContent(t, msg, []ai.Content{a, b})
	})

	t.Run("a message copied by assignment is independent", func(t *testing.T) {
		t.Parallel()

		msg := newFixture(t)
		copied := msg

		copied.Content()[0] = c

		assertContent(t, msg, []ai.Content{a, b})
		assertContent(t, copied, []ai.Content{a, b})
		if msg.ID() != copied.ID() {
			t.Errorf("a copy has identity %v, want the original's %v — a copy is the same message", copied.ID(), msg.ID())
		}
	})

	t.Run("the role vocabulary cannot be rewritten", func(t *testing.T) {
		t.Parallel()

		roles := ai.Roles()
		if len(roles) == 0 {
			t.Fatal("Roles() returned nothing")
		}
		first := roles[0]
		roles[0] = ai.Role(200)

		if again := ai.Roles(); again[0] != first {
			t.Errorf("Roles()[0] = %v after a consumer rewrote its copy, want %v", again[0], first)
		}
	})
}

// AI-05.3 *(appended, pin)* — no input causes a panic.
//
// Appended to AI-05.3's test list during the spec phase, per doc 0002's rule
// that a discovered case is appended to the owning leaf's list rather than
// chased ad hoc. It is green from birth and is a pin rather than a red-first
// item: there is no guard to remove, because the shapes below are total by
// construction today.
//
// Its value is forward-looking and specific. AI-06.3 adds a rule that rejects
// an unconstructed content part, and AI-10 adds rules at request scope; both
// touch this constructor, and both are written against inputs — a nil element,
// a very large sequence, a role at the end of its range — that a rule is easy
// to write without considering.
func TestMessage_ExtremeInputs_NeverPanics(t *testing.T) {
	t.Parallel()

	const huge = 10_000

	many := make([]ai.Content, huge)
	for i := range many {
		many[i] = newPart("x")
	}

	cases := []struct {
		name string
		act  func()
	}{
		{"a nil content element", func() {
			msg, _ := ai.NewMessage(ai.RoleUser, nil)
			_ = msg.Content()
		}},
		{"a mixture of nil and constructed elements", func() {
			msg, _ := ai.NewMessage(ai.RoleUser, nil, newPart("a"), nil)
			_ = msg.Content()
		}},
		{"ten thousand elements", func() {
			msg, err := ai.NewMessage(ai.RoleAssistant, many...)
			if err != nil {
				t.Errorf("NewMessage returned %v, want no failure", err)
			}
			if got := len(msg.Content()); got != huge {
				t.Errorf("len(msg.Content()) = %d, want %d", got, huge)
			}
		}},
		{"every value of the role's underlying type", func() {
			for r := range 256 {
				role := ai.Role(r)
				_ = role.String()
				_, _ = ai.ParseRole(role.String())
				_, _ = ai.NewMessage(role, newPart("a"))
			}
		}},
		{"the zero message, read every way", func() {
			var zero ai.Message
			_ = zero.ID()
			_ = zero.ID().String()
			_ = zero.Role()
			_ = zero.Role().String()
			if got := zero.Content(); len(got) != 0 {
				t.Errorf("the zero Message has %d content elements, want none", len(got))
			}
		}},
		{"an empty and a very long name parsed as a role", func() {
			_, _ = ai.ParseRole("")
			_, _ = ai.ParseRole(strings.Repeat("user", huge))
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
