// AI-06.3 item 3 — the request path, pinned instead of speculated.
//
// doc 0002 asks that the same unconstructed value offered "through the request
// path instead of the message path" be rejected there too, in the documented
// validation order. AI-10 — the request — does not exist, and this milestone has
// no charter to build one: inventing a Request type to satisfy the letter of the
// item would freeze a shape AI-10 has four leaves to decide, and would break the
// milestone rule of one contract per milestone.
//
// So the item is answered the way proposal.md records: the element rule is
// expressed as one function that AI-10 reuses, and the reuse point is pinned
// here. validateContent takes a position prefix that exists for exactly one
// caller that has not arrived. NewMessage passes none and the failure renders as
// "content[0]"; AI-10 will pass AtIndex("messages", i) and the same failure will
// render as "messages[2].content[0]". What this file asserts is that the second
// sentence is true today — that AI-10 inherits a function with a proven contract
// rather than a comment asking it to remember one.
//
// # Why this file is internal
//
// validateContent is unexported, and it stays unexported: exporting it so a test
// could reach it would widen the surface this milestone exists to narrow, and
// would hand a consumer a validator for a value it cannot construct anyway.
// Everything a consumer can observe is asserted from ai_test and agenttest_test.
// This file asserts the seam between two milestones, which is not a consumer
// concern.

package ai

import "testing"

// TestValidateContent_RequestShapedPrefix_ReportsTheDeeperPosition is the pin.
//
// It fails the day the prefix stops composing — which is the day AI-10 would
// otherwise discover, at its own boundary, that it needs a second copy of the
// element rule. That is the failure this file is here to make loud and early: a
// constructor that checks and a boundary that does not is defect C1 arriving
// from the other direction, and decision.md § 7.2 is the argument that one rule
// set with two entry points is what prevents it.
func TestValidateContent_RequestShapedPrefix_ReportsTheDeeperPosition(t *testing.T) {
	t.Parallel()

	// A part assembled without its constructor. From outside package ai this is
	// unwritable — testdata/handrolled records the six diagnostics that say so —
	// and from inside it is exactly the mistake a future variant author makes.
	skippedRules := Part{payload: textPayload{text: "   "}}

	constructed, err := NewText("model-visible text")
	if err != nil {
		t.Fatalf("NewText returned %v, want no failure", err)
	}

	cases := []struct {
		name     string
		prefix   Path
		content  []Part
		wantPath string
	}{
		{
			name:     "no prefix, an unconstructed element — what NewMessage reports today",
			prefix:   nil,
			content:  []Part{{}},
			wantPath: "content[0]",
		},
		{
			name:     "no prefix, an element deeper in the sequence",
			prefix:   nil,
			content:  []Part{constructed, constructed, {}},
			wantPath: "content[2]",
		},
		{
			name:     "a request-shaped prefix — what AI-10 will report",
			prefix:   Path{AtIndex("messages", 2)},
			content:  []Part{{}},
			wantPath: "messages[2].content[0]",
		},
		{
			name:     "a request-shaped prefix, an element deeper in the sequence",
			prefix:   Path{AtIndex("messages", 0)},
			content:  []Part{constructed, {}},
			wantPath: "messages[0].content[1]",
		},
		{
			name:     "the prefix composes with the payload's own position",
			prefix:   Path{AtIndex("messages", 7)},
			content:  []Part{skippedRules},
			wantPath: "messages[7].content[0].text",
		},
		{
			name:     "a prefix of more than one step",
			prefix:   Path{At("request"), AtIndex("messages", 1)},
			content:  []Part{{}},
			wantPath: "request.messages[1].content[0]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			violation := validateContent(tc.prefix, tc.content)
			if violation == nil {
				t.Fatal("validateContent returned nil, want a violation")
			}
			if got := violation.Path().String(); got != tc.wantPath {
				t.Errorf("validateContent(%v, …).Path() = %q, want %q", tc.prefix, got, tc.wantPath)
			}
		})
	}

	t.Run("valid content reports nothing, at either depth", func(t *testing.T) {
		t.Parallel()

		for _, prefix := range []Path{nil, {AtIndex("messages", 3)}} {
			if violation := validateContent(prefix, []Part{constructed, constructed}); violation != nil {
				t.Errorf("validateContent(%v, valid content) = %v, want nil", prefix, violation)
			}
			if violation := validateContent(prefix, nil); violation != nil {
				t.Errorf("validateContent(%v, nil) = %v, want nil — emptiness is the caller's rule", prefix, violation)
			}
		}
	})

	t.Run("the caller's prefix is not rewritten by the rule that extends it", func(t *testing.T) {
		t.Parallel()

		// The aliasing hazard, stated as a test rather than as a comment on
		// under(). A prefix with spare capacity is what an append-based
		// composition would write into, and the second call would then report a
		// position built from the first call's leftovers.
		backing := make([]Step, 1, 8)
		backing[0] = AtIndex("messages", 4)
		prefix := Path(backing)

		first := validateContent(prefix, []Part{{}})
		if first == nil {
			t.Fatal("validateContent returned nil, want a violation")
		}
		if got := first.Path().String(); got != "messages[4].content[0]" {
			t.Fatalf("the first call reported %q, want %q", got, "messages[4].content[0]")
		}

		if len(prefix) != 1 || prefix[0].String() != "messages[4]" {
			t.Errorf("the caller's prefix is now %v, want it untouched", prefix)
		}

		second := validateContent(prefix, []Part{constructed, {}})
		if second == nil {
			t.Fatal("validateContent returned nil on the second call, want a violation")
		}
		if got := second.Path().String(); got != "messages[4].content[1]" {
			t.Errorf("the second call reported %q, want %q — the first call's position leaked", got, "messages[4].content[1]")
		}
		if got := first.Path().String(); got != "messages[4].content[0]" {
			t.Errorf("the first violation now reports %q, want %q — a later call rewrote an earlier failure", got, "messages[4].content[0]")
		}
	})

	t.Run("an unregistered payload kind makes every part carrying it invalid", func(t *testing.T) {
		t.Parallel()

		// The table is load-bearing, not decoration: this is validation rule
		// 3(b), and it is what AI-06.4's third leg asserts per kind.
		unregistered := Part{payload: unregisteredPayload{}}

		violation := validateContent(Path{AtIndex("messages", 1)}, []Part{unregistered})
		if violation == nil {
			t.Fatal("validateContent accepted a payload whose kind has no table entry, want a rejection")
		}
		if got := violation.Path().String(); got != "messages[1].content[0]" {
			t.Errorf("validateContent(…).Path() = %q, want %q", got, "messages[1].content[0]")
		}
	})
}

// unregisteredPayload is a payload whose kind is past the end of the
// registration table. It exists to prove that the table decides membership: a
// payload type added to the package without a table entry is not a member of the
// vocabulary and every part carrying it is invalid.
type unregisteredPayload struct{}

func (unregisteredPayload) kind() PartKind { return PartKind(len(partKindNames) + 40) }

func (unregisteredPayload) validate(_ Path) *Violation { return nil }
