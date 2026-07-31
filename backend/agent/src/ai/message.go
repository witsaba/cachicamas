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
//
// # Copy on construct, copy on read
//
// A message copies the content sequence in and copies it out again. These are
// two mechanisms and not one: a constructor that clones and a reader that does
// not passes every construction test and fails the moment two consumers hold
// the same message.
//
// The reason the first mechanism is easy to miss is that Go hides it. A
// variadic parameter called with a slice spread does not copy, so
// NewMessage(role, parts...) would alias the caller's backing array, and the
// symptom would appear later as content that changed with nobody writing to the
// message. doc 0002 calls its absence "the most confusing class of test failure
// in a streaming package".
//
// What this does not promise: the sequence is copied, not the parts in it.
// Whether a part's own payload can be mutated after construction is a property
// of the part, and the part is AI-06's.

package ai

import (
	"slices"
	"strconv"
	"sync/atomic"
)

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

// MessageID is the stable handle by which one message is distinguished from
// another (V-REQ-03).
//
// It is minted by [NewMessage] and cannot be supplied, forged or recomputed by
// a caller. That is what makes "distinguished from" a property of the type
// rather than a rule someone else has to write: two messages built from an
// identical role and identical content are two messages, and a caller-supplied
// handle could not tell them apart. It is equally why the identity is not
// derived from the content.
//
// It is comparable, so it works as a map key and with ==. It is opaque: a
// uint64 or a string would be comparable too, and would also be forgeable,
// reusable and arithmetic — a caller could write id + 1 and name another
// message. There is deliberately no parser, because an identity a consumer can
// reconstruct from text is a supplied identity wearing a rendering.
//
// The zero value is the identity of a message that was never constructed.
type MessageID struct{ n uint64 }

// IsZero reports whether the identity was never minted.
//
// It is the one way a consumer can tell a constructed message from a zero
// value, which is the question AI-10 asks when it validates a request it was
// handed.
func (id MessageID) IsZero() bool { return id.n == 0 }

// String renders the identity for a diagnostic reader. It has no parser.
func (id MessageID) String() string {
	if id.IsZero() {
		return "msg-unset"
	}
	return "msg-" + strconv.FormatUint(id.n, 10)
}

// lastMessageID is the source of minted identities.
//
// A package-level counter is what defect C3 was, so the difference is worth
// stating rather than assuming. C3's contract was a statement about the
// counter's value — "every stream's first event carries 1, every stream is
// independently contiguous" — which a process-global counter cannot satisfy for
// the second stream in a process, and V-STR-13 records the fix as "putting the
// counter where the stream is". V-REQ-03 states no property of the value at
// all: it asks that two messages be distinguishable, which a monotonic
// process-wide counter gives for every message in the process rather than for
// the first request's alone. Nothing observable would change if this were
// replaced by random bytes tomorrow.
//
// It is atomic rather than mutex-guarded because messages are constructed
// concurrently the moment Layer 2 exists, and the property is proven under the
// race detector rather than by inspection.
var lastMessageID atomic.Uint64

// mintMessageID returns an identity no other message in this process holds.
func mintMessageID() MessageID { return MessageID{n: lastMessageID.Add(1)} }

// Message is the smallest addressable unit of a transcript: one role plus
// ordered content.
//
// It is a value, and copying one is safe: every read returns a fresh sequence,
// so two copies cannot observe each other's content. A copy carries the same
// identity, because a copy is the same message.
//
// A message is Layer 1's unit of attribution and ordering *within* a request.
// It is not a turn, not a transcript and not a history — V-OUT-01 and V-OUT-02
// place all three above this layer.
type Message struct {
	id      MessageID
	role    Role
	content []Content
}

// NewMessage constructs a message from a role and its ordered content.
//
// The rules are checked in the order written, per V-FAIL-04, and the order is
// part of the contract:
//
//  1. The role is a member of the closed vocabulary, else [ErrNotInVocabulary]
//     at "role".
//  2. The content is not empty, else [ErrEmpty] at "content". No arguments, an
//     empty sequence and a nil sequence are one fact and report one failure.
//
// On failure the zero Message is returned, so a caller that ignored the error
// cannot mistake the result for a constructed message — its identity reports
// unset.
//
// The content sequence is copied, so a caller may reuse the slice it passed.
// The elements are not inspected: a nil element is accepted, because rejecting
// an unconstructed content part is AI-06.3's rule and it needs AI-06.1's
// strategy to know what one is.
func NewMessage(role Role, content ...Content) (Message, error) {
	if err := FirstFailure(
		func() *Violation {
			if roleName(role) == "" {
				return Invalid(ErrNotInVocabulary, At("role"))
			}
			return nil
		},
		func() *Violation {
			if len(content) == 0 {
				return Invalid(ErrEmpty, At("content"))
			}
			return nil
		},
	); err != nil {
		return Message{}, err
	}
	return Message{id: mintMessageID(), role: role, content: slices.Clone(content)}, nil
}

// ID returns the message's identity, minted at construction.
func (m Message) ID() MessageID { return m.id }

// Content returns the message's ordered content, in the order it was
// constructed with, repetitions included.
//
// The result is a fresh slice on every call: a consumer that rewrites what it
// received must not be able to rewrite the message, and two consumers must not
// be able to observe each other. The elements themselves are returned as they
// are held — what a content part is, and whether its payload can be read or
// mutated, is AI-06's.
func (m Message) Content() []Content { return slices.Clone(m.content) }

// Role returns the role the message is attributed to.
func (m Message) Role() Role { return m.role }
