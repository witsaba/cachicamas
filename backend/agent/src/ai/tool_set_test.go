// Tests for AI-08.2 — the rules a set of tool declarations obeys.
//
// External test package, so every assertion is written against exactly the
// surface a consumer in another package sees.
package ai_test

import (
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// declare builds a valid declaration, failing the test if it cannot.
func declare(t *testing.T, name string) ai.Tool {
	t.Helper()

	tool, err := ai.NewTool(name, "", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("NewTool(%q) returned %v, want no failure", name, err)
	}
	return tool
}

// AI-08.2 — duplicate tool names in one set are rejected with an AI-04
// sentinel, positioned at the second occurrence.
//
// Positioning by index rather than by name is V-FAIL-03's posture as AI-04
// landed it: V-REQ-14 makes the set ordered and deterministically iterable, so
// an index identifies one declaration unambiguously, and the caller — which
// holds the set — resolves it to a name in one step without a caller value
// reaching a message that will be logged.
//
// The class is ErrMalformed. A duplicate is cleanly none of AI-04's five, which
// is a real finding rather than a gap: it is not a bound (ErrOutOfRange), the
// set is not a closed vocabulary (ErrNotInVocabulary), and AI-04's decision
// record is explicit that the set grows only in the pull request that needs a
// new class. "Not well-formed for what this field must be" describes a set with
// a repeated name accurately, and the position says which element broke it.
func TestNewToolSet_DuplicateNames_FailWithASentinelAtTheSecondOccurrence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		label string
		names []string
		want  int
	}{
		{"an adjacent pair", []string{"read", "read"}, 1},
		{"a separated pair", []string{"read", "write", "edit", "read"}, 3},
		{"the first of several duplicate pairs", []string{"a", "b", "b", "a"}, 2},
		{"three of the same name", []string{"x", "x", "x"}, 1},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			tools := make([]ai.Tool, 0, len(tc.names))
			for _, name := range tc.names {
				tools = append(tools, declare(t, name))
			}

			_, err := ai.NewToolSet(tools...)
			if err == nil {
				t.Fatalf("NewToolSet(%v) returned no failure, want one", tc.names)
			}
			if !errors.Is(err, ai.ErrMalformed) {
				t.Errorf("errors.Is(err, ErrMalformed) = false, want true (err = %v)", err)
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			want := "tools[" + strconv.Itoa(tc.want) + "]"
			if got := violation.Path().String(); got != want {
				t.Errorf("violation.Path() = %q, want %q", got, want)
			}
		})
	}

	t.Run("the same occurrence is reported on every run", func(t *testing.T) {
		t.Parallel()

		tools := []ai.Tool{
			declare(t, "alpha"), declare(t, "bravo"), declare(t, "bravo"),
			declare(t, "alpha"), declare(t, "charlie"), declare(t, "charlie"),
		}

		for range 64 {
			_, err := ai.NewToolSet(tools...)
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			if got := violation.Path().String(); got != "tools[2]" {
				t.Fatalf("violation.Path() = %q, want %q — the reported occurrence is not deterministic", got, "tools[2]")
			}
		}
	})

	t.Run("distinct names are accepted", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewToolSet(declare(t, "read"), declare(t, "write"), declare(t, "edit"))
		if err != nil {
			t.Errorf("NewToolSet with distinct names returned %v, want no failure", err)
		}
	})

	t.Run("the offered name never reaches the failure", func(t *testing.T) {
		t.Parallel()

		const sentinelName = "CACHICAMA_SENTINEL_NAME_c40e"

		_, err := ai.NewToolSet(declare(t, sentinelName), declare(t, sentinelName))
		if err == nil {
			t.Fatal("NewToolSet returned no failure, want one")
		}
		if got := err.Error(); got != "tools[1]: "+ai.ErrMalformed.Error() {
			t.Errorf("Error() = %q, want it composed only of the position and the rule class", got)
		}
	})
}

// AI-08.2 — an empty tool set is legal.
//
// A request with no tools is the common case, not an error. Unlike a closed
// vocabulary, whose zero value must be invalid so a field nobody set is
// rejected, here the zero value *is* a member of the legal domain: a set nobody
// built must behave as a set with nothing in it, not as an unconstructed value
// that panics or lies.
func TestNewToolSet_EmptySet_IsLegal(t *testing.T) {
	t.Parallel()

	t.Run("every spelling of empty constructs", func(t *testing.T) {
		t.Parallel()

		var nilSlice []ai.Tool

		cases := []struct {
			label string
			build func() (ai.ToolSet, error)
		}{
			{"no arguments", func() (ai.ToolSet, error) { return ai.NewToolSet() }},
			{"an empty slice", func() (ai.ToolSet, error) { return ai.NewToolSet([]ai.Tool{}...) }},
			{"a nil slice", func() (ai.ToolSet, error) { return ai.NewToolSet(nilSlice...) }},
		}

		for _, tc := range cases {
			t.Run(tc.label, func(t *testing.T) {
				t.Parallel()

				set, err := tc.build()
				if err != nil {
					t.Fatalf("NewToolSet returned %v, want no failure — an empty tool set is legal", err)
				}
				if got := set.Len(); got != 0 {
					t.Errorf("set.Len() = %d, want 0", got)
				}
				if got := set.Tools(); len(got) != 0 {
					t.Errorf("set.Tools() = %v, want an empty sequence", got)
				}
			})
		}
	})

	t.Run("the zero value is the empty set", func(t *testing.T) {
		t.Parallel()

		var set ai.ToolSet

		if got := set.Len(); got != 0 {
			t.Errorf("zero ToolSet: Len() = %d, want 0", got)
		}
		if got := set.Tools(); len(got) != 0 {
			t.Errorf("zero ToolSet: Tools() = %v, want an empty sequence", got)
		}
		if set.Declares("anything") {
			t.Error("zero ToolSet: Declares(\"anything\") = true, want false")
		}
	})

	t.Run("a non-empty set reports what it holds", func(t *testing.T) {
		t.Parallel()

		set, err := ai.NewToolSet(declare(t, "read"), declare(t, "write"))
		if err != nil {
			t.Fatalf("NewToolSet returned %v, want no failure", err)
		}
		if got := set.Len(); got != 2 {
			t.Errorf("set.Len() = %d, want 2", got)
		}
		if !set.Declares("read") || !set.Declares("write") {
			t.Error("set.Declares reported false for a name the set holds")
		}
		if set.Declares("edit") {
			t.Error("set.Declares(\"edit\") = true, want false")
		}
		if set.Declares("Read") {
			t.Error("set.Declares(\"Read\") = true, want false — matching is exact, not case-folded")
		}
	})
}

// AI-08.2 — a tool set iterated twice yields an identical order, and it is the
// order the caller supplied.
//
// This is the milestone's most expensive test to skip and its cheapest to
// write. Map-iteration order reaching the wire would silently invalidate a
// provider's cache prefix on every call: no failure, no crash, no wrong answer,
// just a tenfold input bill. V-REQ-25 puts the tool set first in the
// invalidation cascade, which is what makes the tool set specifically — rather
// than any collection — the place this matters.
//
// The assertion is against the *caller's* order and not merely against a
// previous read, because a stable-but-arbitrary order satisfies
// self-comparison and still breaks the property: a caller that builds the same
// set twice must obtain the same prefix. The set is 64 elements and is read 100
// times, so Go's map-iteration randomization would show on the first read
// rather than eventually.
func TestToolSet_Iteration_YieldsTheCallersOrderEveryTime(t *testing.T) {
	t.Parallel()

	// Names chosen so that lexical, insertion and hash orders all differ.
	const size = 64
	supplied := make([]ai.Tool, 0, size)
	wantNames := make([]string, 0, size)
	for i := range size {
		name := "tool_" + strconv.Itoa((size*7-i*13)%size) + "_" + strconv.Itoa(i)
		supplied = append(supplied, declare(t, name))
		wantNames = append(wantNames, name)
	}

	set, err := ai.NewToolSet(supplied...)
	if err != nil {
		t.Fatalf("NewToolSet returned %v, want no failure", err)
	}

	readNames := func(s ai.ToolSet) []string {
		out := make([]string, 0, s.Len())
		for _, tool := range s.Tools() {
			out = append(out, tool.Name())
		}
		return out
	}

	t.Run("every read yields the caller's order", func(t *testing.T) {
		t.Parallel()

		for attempt := range 100 {
			got := readNames(set)
			if !slices.Equal(got, wantNames) {
				t.Fatalf("read %d yielded %v, want the caller's order %v", attempt, got, wantNames)
			}
		}
	})

	t.Run("two sets built the same way are element-for-element equal", func(t *testing.T) {
		t.Parallel()

		twin, err := ai.NewToolSet(supplied...)
		if err != nil {
			t.Fatalf("NewToolSet returned %v, want no failure", err)
		}
		if !slices.Equal(readNames(twin), readNames(set)) {
			t.Error("two sets built from the same declarations in the same order read back differently")
		}
	})

	t.Run("concurrent readers each observe the caller's order", func(t *testing.T) {
		t.Parallel()

		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 25 {
					if !slices.Equal(readNames(set), wantNames) {
						t.Error("a concurrent reader observed an order other than the caller's")
						return
					}
				}
			}()
		}
		wg.Wait()
	})

	t.Run("the caller may mutate what it passed", func(t *testing.T) {
		t.Parallel()

		own := slices.Clone(supplied)
		built, err := ai.NewToolSet(own...)
		if err != nil {
			t.Fatalf("NewToolSet returned %v, want no failure", err)
		}

		slices.Reverse(own)
		own[0] = declare(t, "intruder")

		if !slices.Equal(readNames(built), wantNames) {
			t.Errorf("the set changed after the caller rewrote its own slice: %v", readNames(built))
		}
	})

	t.Run("a consumer may mutate what it read", func(t *testing.T) {
		t.Parallel()

		first := set.Tools()
		slices.Reverse(first)
		first[0] = declare(t, "intruder")

		if !slices.Equal(readNames(set), wantNames) {
			t.Errorf("the set changed after a consumer rewrote what it read: %v", readNames(set))
		}

		second := set.Tools()
		third := set.Tools()
		if len(second) > 0 && &second[0] == &third[0] {
			t.Error("two reads share a backing array; one consumer can rewrite another's view")
		}
	})
}

// AI-08.2 *(appended)* — a declaration that never passed construction is
// rejected at the collection boundary.
//
// Discovered while writing item 1 and appended to the leaf's list under doc
// 0002's living-graph clause, which appends discovered *cases* to the owning
// leaf rather than creating a node. A zero Tool has an empty name, which no
// constructed declaration can have, so emptiness is a sound detector without a
// separate "constructed" flag. Without it a value that skipped NewTool would
// reach the wire carrying a name no provider accepts — AI-06.3 item 1's shape,
// one contract over.
func TestNewToolSet_AnUnconstructedDeclaration_IsRejected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		label string
		tools []ai.Tool
		want  int
	}{
		{"the zero declaration alone", []ai.Tool{{}}, 0},
		{"a zero declaration after a valid one", []ai.Tool{declare(t, "read"), {}}, 1},
		{"a zero declaration before a valid one", []ai.Tool{{}, declare(t, "read")}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			set, err := ai.NewToolSet(tc.tools...)
			if err == nil {
				t.Fatal("NewToolSet returned no failure, want one — an unconstructed declaration must not reach the wire")
			}
			if !errors.Is(err, ai.ErrEmpty) {
				t.Errorf("errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			want := "tools[" + strconv.Itoa(tc.want) + "]"
			if got := violation.Path().String(); got != want {
				t.Errorf("violation.Path() = %q, want %q", got, want)
			}
			if set.Len() != 0 {
				t.Errorf("the returned set holds %d declarations, want the zero ToolSet", set.Len())
			}
		})
	}

	t.Run("emptiness is reported before duplication", func(t *testing.T) {
		t.Parallel()

		// Two zero declarations are both unconstructed *and* duplicates of one
		// another. The unconstructed fact is the more fundamental one and is
		// reported first, at the earlier index.
		_, err := ai.NewToolSet(ai.Tool{}, ai.Tool{})
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ErrEmpty) = false, want true (err = %v)", err)
		}
		if errors.Is(err, ai.ErrMalformed) {
			t.Errorf("errors.Is(err, ErrMalformed) = true, want false (err = %v)", err)
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
		}
		if got := violation.Path().String(); got != "tools[0]" {
			t.Errorf("violation.Path() = %q, want %q", got, "tools[0]")
		}
	})
}
