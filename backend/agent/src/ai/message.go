// AI-05 — the message, and the seam it holds content through.
//
// This file implements V-REQ-02 message ("the smallest addressable unit of a
// transcript: one role plus ordered content") and V-REQ-03 message identity. It
// owns the unit and never the collection: message order across a request is
// AI-10's, and a transcript is V-OUT-02, which belongs to Layer 2.
//
// # The content seam, and what it deliberately does not decide
//
// A message holds ordered content, and AI-06 — the keystone of wave 1 — decides
// what a content part is. Those two milestones are in that order on purpose, so
// this file names the position a content part occupies and settles nothing
// about the part itself. [Content] declares one unexported method and nothing
// else: no payload, no kind, no accessor, no constructor, no rendering.
//
// Both of AI-06.1's properties therefore remain open. V-REQ-06 content-part
// readability is open because the seam exposes nothing to read; V-REQ-07
// content-part sealing is open because the seam validates nothing, and this
// milestone rejects no content element — not even a nil one. AI-06.3 item 1 is
// the assertion that rejects an unconstructed part, and it has to be able to
// fail before it passes.
package ai

// Content is the seam through which a message holds one element of its ordered
// content. AI-06 decides what a content part is.
//
// # This is not a seal
//
// The method is unexported, so a type in another package cannot implement
// Content — but it can satisfy it by embedding the interface, which promotes
// the method:
//
//	type part struct {
//		ai.Content
//		label string
//	}
//
// That door is open on purpose. AI-06.3 item 2 says the seal may be "the
// compiler or validation, whichever the AI-06.1 strategy chose", and closing
// the compile-time half here would take a decision that is not this milestone's
// to take. It is also what lets an external test hold content at all, one
// milestone before a content part exists.
//
// A part built that way carries a nil embedded interface, so any method AI-06
// later calls on it panics. That is precisely why AI-06.3 item 1 must reject an
// unconstructed part at validation rather than trusting the compiler.
type Content interface {
	isContent()
}

// Message is the smallest addressable unit of a transcript: one role plus
// ordered content.
//
// It is a value. Copying a Message is safe: every read returns a fresh
// sequence, so two copies cannot observe each other's content.
type Message struct {
	role Role
}

// NewMessage constructs a message from a role and its ordered content.
//
// The rules are checked in the order written, per V-FAIL-04: the role must be a
// member of the vocabulary. On failure the zero Message is returned, so a
// caller that ignored the error cannot mistake the result for a constructed
// message.
func NewMessage(role Role, content ...Content) (Message, error) {
	if err := FirstFailure(
		func() *Violation {
			if roleName(role) == "" {
				return Invalid(ErrNotInVocabulary, At("role"))
			}
			return nil
		},
	); err != nil {
		return Message{}, err
	}
	return Message{role: role}, nil
}

// Role returns the role the message is attributed to.
func (m Message) Role() Role { return m.role }
