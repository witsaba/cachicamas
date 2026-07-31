// AI-06.2 — text content, the first subject the part contract is proven against.
//
// This file implements V-REQ-08 text content: "the content-part kind carrying
// model-visible natural-language text". It is one file for one kind, which is
// the layout AI-07 and AI-09 inherit: the payload type, its rules, its
// constructor and its accessor live together, so adding a kind is one new file
// plus three lines in content_part.go.

package ai

// textPayload carries model-visible natural-language text (V-REQ-08).
//
// It is unexported and holds a string, so a text part is one interface word wide
// and carries no mutable state: copying a part copies the payload, and a string
// cannot be rewritten through the copy.
type textPayload struct{ text string }

// kind reports this payload's registered kind.
func (textPayload) kind() PartKind { return PartKindText }

// NewText constructs a text content part.
func NewText(text string) (Part, error) {
	return Part{payload: textPayload{text: text}}, nil
}

// Text returns the part's text and whether it carries any.
func (p Part) Text() (string, bool) {
	return "", false
}
