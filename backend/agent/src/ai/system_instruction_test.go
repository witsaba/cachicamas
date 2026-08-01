// Tests for AI-10.2 — the segmented system instruction.
//
// External package, for AI-10.1's reason: readability from outside is
// constitutive of the request rather than incidental to it.

package ai_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// segment builds a segment, failing the test rather than the contract when the
// text is one this milestone considers legal.
func segment(t *testing.T, text string) ai.Segment {
	t.Helper()

	s, err := ai.NewSegment(text)
	if err != nil {
		t.Fatalf("ai.NewSegment(%q) returned %v, want no failure", text, err)
	}
	return s
}

// AI-10.2 item 1 — ordered segments round-trip exactly.
//
// V-REQ-19: one ordered, individually markable piece of the system
// instruction. doc 0001 § 3.2 lists the flat system instruction first among the
// things that must change before an adapter exists, because a flat string has
// nowhere to put a cache boundary — and retrofitting segments was a breaking
// change in the retired plan.
func TestRequest_SystemInstruction_RoundTripsSegmentOrderAndContentExactly(t *testing.T) {
	t.Parallel()

	texts := []string{
		"You are a terse assistant.",
		"  Preserve the caller's own separators.  ",
		"Répondez en français. 中文も。\U0001F600",
	}

	segments := make([]ai.Segment, 0, len(texts))
	for _, text := range texts {
		segments = append(segments, segment(t, text))
	}

	instruction, err := ai.NewSystemInstruction(segments...)
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "hello")},
		ai.WithSystemInstruction(instruction),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	read, ok := request.SystemInstruction()
	if !ok {
		t.Fatalf("request.SystemInstruction() reported absent, want present")
	}
	if got := read.Len(); got != len(texts) {
		t.Fatalf("read.Len() = %d, want %d", got, len(texts))
	}

	for i, want := range texts {
		if got := read.Segments()[i].Text(); got != want {
			t.Errorf("read.Segments()[%d].Text() = %q, want %q", i, got, want)
		}
	}
}

// AI-10.2 item 2 — the single-segment convenience path is indistinguishable.
//
// doc 0002 prices segments at zero on exactly this condition: "a single-segment
// request is the common case, and the ergonomic constructor for it is part of
// this milestone's deliverable". Anthropic's system field takes either a string
// or an ordered array of blocks, which is why the ergonomic path must produce
// the one-block array rather than a parallel shape beside it.
func TestNewSystemText_SingleSegmentPath_IsIndistinguishableFromTheSegmentBySegmentBuild(t *testing.T) {
	t.Parallel()

	const text = "You are a terse assistant."

	fromText, err := ai.NewSystemText(text)
	if err != nil {
		t.Fatalf("ai.NewSystemText returned %v, want no failure", err)
	}
	fromSegment, err := ai.NewSystemInstruction(segment(t, text))
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}

	if fromText.Len() != fromSegment.Len() {
		t.Fatalf("text-built instruction has %d segments, segment-built has %d", fromText.Len(), fromSegment.Len())
	}
	if !slices.Equal(fromText.Segments(), fromSegment.Segments()) {
		t.Errorf("ai.NewSystemText(%q).Segments() = %v, want it to equal the segment-by-segment build", text, fromText.Segments())
	}
}

// AI-10.2 item 3 — an absent system instruction is legal, and is
// distinguishable from one empty segment.
//
// The trap a flat string sets is not only the missing cache boundary: "" means
// both absent and empty, so a request that meant to carry no instruction and one
// that carries an empty one are the same value. Segments dissolve both. Absence
// is the option not being applied; "one empty segment" is not merely equal to
// absence, it is unrepresentable, because neither an empty segment nor a
// zero-segment instruction can be constructed.
func TestRequest_AbsentSystemInstruction_IsLegalAndUnrepresentableAsAnEmptySegment(t *testing.T) {
	t.Parallel()

	messages := []ai.Message{userTextMessage(t, "hello")}

	bare, err := ai.NewRequest("m", messages)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	read, ok := bare.SystemInstruction()
	if ok {
		t.Errorf("bare.SystemInstruction() reported present on a request that applied no instruction")
	}
	if got := read.Len(); got != 0 {
		t.Errorf("read.Len() = %d on an absent instruction, want 0", got)
	}

	t.Run("a zero-segment instruction cannot be constructed", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewSystemInstruction()
		requireViolation(t, err, ai.ErrEmpty, "system")
	})

	t.Run("an instruction that skipped its constructor is rejected by the request", func(t *testing.T) {
		t.Parallel()

		_, err := ai.NewRequest("m", messages, ai.WithSystemInstruction(ai.SystemInstruction{}))
		requireViolation(t, err, ai.ErrEmpty, "system")
	})

	t.Run("one real segment is present and distinguishable from absence", func(t *testing.T) {
		t.Parallel()

		instruction, err := ai.NewSystemText("be terse")
		if err != nil {
			t.Fatalf("ai.NewSystemText returned %v, want no failure", err)
		}
		request, err := ai.NewRequest("m", messages, ai.WithSystemInstruction(instruction))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		if _, ok := request.SystemInstruction(); !ok {
			t.Errorf("request.SystemInstruction() reported absent on a request that applied one")
		}
	})
}

// AI-10.2 item 4 — segment construction rules report through AI-04's
// vocabulary.
//
// Whitespace-only is rejected with emptiness for a reason V-REQ-19 supplies: a
// segment of spaces places no text into the instruction while occupying an
// ordinal that is individually markable, so accepting it would let a caller
// mark a cache boundary on nothing.
func TestNewSegment_EmptyOrWhitespaceOnlyText_FailsWithErrEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		what string
		text string
	}{
		{"empty text", ""},
		{"spaces only", "   "},
		{"tabs and newlines only", "\t\n\r\n\t"},
		{"a non-breaking space only", " "},
		{"an ideographic space only", "　"},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			t.Parallel()

			_, err := ai.NewSegment(c.text)
			requireViolation(t, err, ai.ErrEmpty, "text")
		})
	}
}

// AI-10.2 item 4, the other direction — whitespace-only is a rejection
// criterion and never a normalization.
//
// Without this pin a later "helpful" trim would pass every other test in this
// file and silently rewrite callers' prompts. Prompt text is the one thing in
// this package that must survive byte-exact, and an adapter concatenating
// segments needs the caller's own separators.
func TestNewSegment_TextWithSurroundingWhitespace_IsNotTrimmed(t *testing.T) {
	t.Parallel()

	const text = "\n  Be terse.  \n"

	if got := segment(t, text).Text(); got != text {
		t.Errorf("segment.Text() = %q, want %q", got, text)
	}
}

// AI-10.2 item 2, deferred to here — the convenience path inherits the
// segment's rules.
//
// It was written during item 2 and parked until this item landed the rule it
// asserts, so that it would go green with no further edit to NewSystemText.
// That is the observable proof the two construction paths share one rule set: a
// second path that built the value directly would still be passing item 2's
// indistinguishability test and failing this one.
func TestNewSystemText_TextThatNoSegmentMayCarry_FailsWithTheSegmentsOwnRule(t *testing.T) {
	t.Parallel()

	_, err := ai.NewSystemText("   ")
	requireViolation(t, err, ai.ErrEmpty, "text")
}

// AI-10.2 item 5 (appended) — the system region renders no text through any
// verb.
//
// The system instruction is the region most likely to carry a proprietary
// prompt, and AI-10.1's leak table could not cover it because the region did
// not exist yet. Segment and SystemInstruction each need both renderings for
// AI-10.1's reason: without GoString, %#v falls back to reflection and prints
// the unexported text.
func TestSystemInstruction_Formatting_RendersNoSegmentTextThroughAnyVerb(t *testing.T) {
	t.Parallel()

	const secret = "SECRET-SYSTEM-PROMPT"

	seg := segment(t, secret)
	instruction, err := ai.NewSystemInstruction(seg, segment(t, "second"))
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}
	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "hello")},
		ai.WithSystemInstruction(instruction),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	subjects := map[string]any{
		"the segment":            seg,
		"the system instruction": instruction,
		"the request":            request,
	}

	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		for name, subject := range subjects {
			rendered := fmt.Sprintf(verb, subject)
			if strings.Contains(rendered, secret) {
				t.Errorf("fmt.Sprintf(%q, %s) leaked the segment text: %q", verb, name, rendered)
			}
		}
	}
}

// AI-10.2 item 5 (appended), the other direction — the renderings still name
// the shape.
//
// A request must say that it carries a system instruction and how many segments
// it holds, because that is the fact a reader debugging a cache boundary needs
// and it is not caller data.
func TestSystemInstruction_Formatting_NamesTheSegmentCountAndTheRequestsSystemRegion(t *testing.T) {
	t.Parallel()

	instruction, err := ai.NewSystemInstruction(segment(t, "a"), segment(t, "b"))
	if err != nil {
		t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
	}
	if got, want := instruction.String(), "system(2 segments)"; got != want {
		t.Errorf("instruction.String() = %q, want %q", got, want)
	}
	if got, want := (ai.Segment{}).String(), "segment(unset)"; got != want {
		t.Errorf("ai.Segment{}.String() = %q, want %q", got, want)
	}
	if got, want := segment(t, "a").String(), "segment"; got != want {
		t.Errorf("segment.String() = %q, want %q", got, want)
	}

	request, err := ai.NewRequest(
		"m", []ai.Message{userTextMessage(t, "hello")},
		ai.WithSystemInstruction(instruction),
	)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	if got, want := request.String(), "system(2 segments)"; !strings.Contains(got, want) {
		t.Errorf("request.String() = %q, want it to contain %q", got, want)
	}
}
