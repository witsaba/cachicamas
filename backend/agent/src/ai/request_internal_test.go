// Internal tests for AI-10.1 — the content-validation seam, consumed.
//
// Internal by necessity rather than by preference, and the necessity is the
// point. A constructed message cannot hold an invalid content part: NewMessage
// validates its content, and Part is sealed, so no consumer in another package
// can assemble the value this test needs. That is exactly why the request's own
// content rule is worth landing — it is the rule that keeps AI-10 free of
// per-kind logic on the day a path that is not NewMessage produces a message.
//
// content_part_internal_test.go is the precedent for reaching inside to pin a
// seam whose failure mode is unreachable from outside.

package ai

import (
	"errors"
	"testing"
)

// AI-10.1 item 6 (appended) — the request path calls AI-06's content validator
// with the request-shaped prefix.
//
// AI-06.3 item 3 pinned the prefix "messages[2].content[0]" before a request
// existed, and content_part.go names this caller by name: "AI-10 holds a
// request, whose content sits one level deeper, and will pass
// AtIndex(\"messages\", i)". The decision record states the principle as one
// rule set, two callers, and names the failure of the alternative — a
// constructor that checks and a boundary that does not.
//
// The assertion is on the *composed position*, not merely on the class,
// because the position is the only observable proof the prefix was passed
// rather than dropped.
func TestNewRequest_InvalidContentPart_ReportsAI06sRuleAtTheComposedPosition(t *testing.T) {
	t.Parallel()

	text, err := NewText("hello")
	if err != nil {
		t.Fatalf("NewText returned %v, want no failure", err)
	}

	// Assembled inside the package: a message that passed construction cannot
	// carry the zero Part, which is the whole of AI-06's seal.
	good := Message{id: mintMessageID(), role: RoleUser, content: []Part{text}}
	bad := Message{id: mintMessageID(), role: RoleUser, content: []Part{{}}}

	_, err = NewRequest("m", []Message{good, bad})
	if err == nil {
		t.Fatalf("NewRequest returned no failure, want ErrNotInVocabulary at messages[1].content[0]")
	}
	if !errors.Is(err, ErrNotInVocabulary) {
		t.Errorf("errors.Is(err, ErrNotInVocabulary) = false on %v", err)
	}

	var violation *Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, *Violation) = false on %v", err)
	}
	if got, want := violation.Path().String(), "messages[1].content[0]"; got != want {
		t.Errorf("violation.Path() = %q, want %q", got, want)
	}
}

// AI-10.1 item 6 (appended) — the deeper positions compose too.
//
// A text part exceeding the documented bound reports at
// "messages[0].content[0].text": three levels, of which AI-10 supplies one and
// AI-06 supplies two. A request that reimplemented the text rule would have to
// know the third level exists, and this is the assertion that says it does not.
func TestNewRequest_OverlongTextPart_ReportsAI06sDeepPositionBeneathTheRequestPrefix(t *testing.T) {
	t.Parallel()

	overlong := Part{payload: textPayload{text: string(make([]byte, MaxTextLen+1))}}
	message := Message{id: mintMessageID(), role: RoleUser, content: []Part{overlong}}

	_, err := NewRequest("m", []Message{message})
	if err == nil {
		t.Fatalf("NewRequest returned no failure, want a bound failure at messages[0].content[0].text")
	}

	var violation *Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, *Violation) = false on %v", err)
	}
	if got, want := violation.Path().String(), "messages[0].content[0].text"; got != want {
		t.Errorf("violation.Path() = %q, want %q", got, want)
	}
}
