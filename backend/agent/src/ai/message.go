// AI-05 — the message, and the seam it holds content through.
//
// This file implements V-REQ-02 message ("the smallest addressable unit of a
// transcript: one role plus ordered content") and V-REQ-03 message identity. It
// owns the unit and never the collection: message order across a request is
// AI-10's, and a transcript is V-OUT-02, which belongs to Layer 2.
//
// # The content seam, closed by AI-06
//
// A message holds ordered content, and AI-06 — the keystone of wave 1 — decided
// what a content part is. This file held the position open for it: an interface
// named Content, declaring one unexported method and nothing else, documented as
// "not a seal on purpose" because closing it would have taken a decision that
// was not AI-05's to take.
//
// AI-06 took it. Message content is now [Part], a concrete opaque value type, so
// the seam is closed by the type system rather than by an interface anyone could
// satisfy by embedding. What that buys is recorded on [Part] itself; what it
// costs this file is one rule, stated in [NewMessage].
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
// What this does not promise: the sequence is copied, not the parts in it. A
// part carries no mutable state, so the distinction costs nothing today — but it
// is a property of the part, not of the message, and it is [Part]'s to keep.

package ai

import (
	"slices"
	"strconv"
	"sync/atomic"
)

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
	id            MessageID
	role          Role
	content       []Part
	cacheBoundary bool // AI-11.1
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
//  3. Every element is a constructed content part, checked in index order, so
//     the first failing element is the one reported. A value that skipped its
//     constructor — the zero [Part], or one promoted out of a type that embeds
//     it — carries no payload, therefore no kind, and fails with
//     [ErrNotInVocabulary] at "content[i]". This is the seal: doc 0001's defect
//     C1 was an exported part type whose zero value passed message validation
//     and bypassed every construction rule.
//
// On failure the zero Message is returned, so a caller that ignored the error
// cannot mistake the result for a constructed message — its identity reports
// unset.
//
// The content sequence is copied, so a caller may reuse the slice it passed.
func NewMessage(role Role, content ...Part) (Message, error) {
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
		func() *Violation {
			return validateContent(nil, content)
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
func (m Message) Content() []Part { return slices.Clone(m.content) }

// Role returns the role the message is attributed to.
func (m Message) Role() Role { return m.role }

// MarkCacheBoundary returns a copy of m carrying a cache-boundary marker
// (V-REQ-23), indicating a point at which a provider may cache the prefix
// preceding it.
//
// It is a no-op when m never passed [NewMessage]: its zero-value detector is
// [MessageID.IsZero], and a marked zero value must not start reporting
// itself a cache boundary (R-ACB-002).
//
// m is a value receiver, so the marker is set on a local copy and returned;
// the value m was called on is left observably unmarked (R-ACB-001). The
// copy carries the same identity, because a copy is the same message — the
// same rule this file states for every other copy of m.
func (m Message) MarkCacheBoundary() Message {
	if m.ID().IsZero() {
		return m
	}
	m.cacheBoundary = true
	return m
}

// IsCacheBoundary reports whether m carries a cache-boundary marker.
//
// A message that was never marked reports false, and marking an
// already-marked message again is idempotent (R-ACB-001).
func (m Message) IsCacheBoundary() bool { return m.cacheBoundary }

// Equal reports whether m and other carry the same role, the same marker
// state and the same content, in the same order.
//
// Identity is excluded on purpose: this is AI-10.6's addition, landed here
// because design.md § 11.2 needs it to compose [Request.Equal]. V-REQ-03
// makes [MessageID] deliberately unforgeable and minted per construction, so
// two messages built from identical inputs are two distinct messages — but
// they are still the same *value*, and that is the question this method
// answers. Including identity here would make it unanswerable: it would
// report false for any two independently constructed messages at all, which
// is not what request-level equality needs from it.
//
// The marker comparison is AI-11.1's addition (R-ACB-004): two messages
// otherwise identical but differing in cache-boundary state describe
// different requests, because a marker expresses a request for different
// provider behavior. Unlike [Segment], Message stays non-comparable — its
// content is a slice — so this method names its fields explicitly rather
// than composing ==, and a field added here has to be named or it is
// invisible to every comparison built on this method, request equality
// included.
//
// Content is compared with slices.Equal, which compares [Part] values by ==
// — content_part.go documents that as comparing payloads, which is what
// makes every registered kind reach this method for free, without a
// kind-by-kind switch.
func (m Message) Equal(other Message) bool {
	return m.role == other.role && m.cacheBoundary == other.cacheBoundary && slices.Equal(m.content, other.content)
}

// String renders the message for the %v and %s verbs, naming structure only
// — the role and the part count — never a byte of content. fmt's printValue
// skips handleMethods for a value reached through an unexported field, so
// without this method %v on anything holding a Message falls back to
// reflection and prints the conversation verbatim (V-FAIL-13, the same
// posture every other payload-carrying type in this package keeps).
func (m Message) String() string {
	if m.role == 0 && len(m.content) == 0 {
		return "message(unset)"
	}
	return "message(" + m.role.String() + ", " + strconv.Itoa(len(m.content)) + " parts)"
}

// GoString renders the message for the %#v verb. It exists so the redaction
// posture covers every fmt verb rather than the three a reader thinks of.
func (m Message) GoString() string { return m.String() }
