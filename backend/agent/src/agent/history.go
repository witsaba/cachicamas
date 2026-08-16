// AG-12.1/AG-12.2 — History and the pairing invariant (R-HIS-001..009).
//
// The harness has no transcript before this file. `ai/tool_result.go`
// states its own boundary verbatim: "Pairing them is Layer 2's job: the
// invariant that every call in a transcript has a matching result is
// V-OUT-02." `ai.ToolCall.ID()` and `ai.ToolResult.CallID()` are the
// correlation key; nothing above Layer 1 checked it until now.
//
// # One opaque store, one validating commit path
//
// [History] is an opaque struct with unexported storage. Every public
// route that can extend or mutate the transcript — [History.Append],
// [NewSeededHistory], [History.CloseTurn], [History.SynthesizeOrphans] —
// funnels through the single unexported [History.commit] primitive
// (R-HIS-004, the C1 lesson (doc 0001:376) reapplied to history). Bypass is
// impossible by construction, not convention: entries and open are
// unexported, so no external caller can reach them; commit is the only
// function whose body assigns them, auditable by reading this one file;
// [EntryOrigin] is a commit parameter no public signature accepts, so only
// [History.SynthesizeOrphans] can mint [EntryOriginSynthesized]; and the
// zero-value door defect C1 exploited is closed at the same single point —
// commit rejects a history that never passed [NewHistory] or
// [NewSeededHistory], so `new(History)` is unusable through every route.
// history_surface_guard_test.go closes the enumeration itself: the route
// list this file's own doc could drift from is never trusted — it is
// diffed, set-equal, against the exported surface (S-HIS-030/S-HIS-031).
//
// # Two directions, kept apart everywhere
//
// The pairing invariant has two directions, and collapsing them would make
// AG-12.2 unreachable, so this file keeps them apart in every rule and
// every doc comment, not only in the spec's prose:
//
//   - An orphaned result — a tool-result entry naming a call the transcript
//     does not declare — is rejected at commit time, through every route
//     including seeded construction (R-HIS-002). It can never enter a
//     transcript.
//   - An open call — a tool call issued with no result yet — is legal
//     while the turn is open, through every route including seeded
//     construction. It is rejected only when the turn is closed
//     (R-HIS-003, [History.CloseTurn]), identically on a seeded and an
//     appended history.
//
// A seed ending in one or more open calls is therefore accepted: rejecting
// it would make an interrupted transcript unseedable on resume, and
// [History.SynthesizeOrphans] would be unreachable through the exact path
// it exists to serve.
//
// # The entry envelope
//
// Every committed value is stored in an opaque [Entry] envelope carrying
// the unmodified `ai.Message`, an ordinal-derived [EntryID] (R-HIS-005),
// and an [EntryOrigin] discriminator (R-HIS-007). "History exposes Layer 1
// values" holds because read-back yields the Layer 1 value unmodified and
// unaliased ([History.Entries] copies on every call); the discriminator
// rides the envelope that stable entry identity already forced into
// existence, so the two obligations were never in tension.
//
// # What this file does not do
//
// It does not wire into `loop.go` or `scheduler.go` — the run driver's
// consumption of this surface is AG-13's (NFR-HIS-003). It does not detect
// or cause an interruption — that is AG-14's; this file only repairs the
// transcript an interruption already left. It adds no rule class to
// `ai/validation.go` — every typed rejection reuses `ai.Invalid` /
// `ai.At` / `ai.AtIndex` / `ai.FirstFailure` with the existing
// `ai.ErrUnresolvedReference` and `ai.ErrEmpty` classes.
package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// EntryOrigin discriminates how an entry entered the transcript
// (R-HIS-007). The zero value is unset — the [ai.Role] idiom: the first
// member starts at iota + 1, so a field nobody set is rejected exactly
// like a wild one, never silently read as [EntryOriginAppended].
type EntryOrigin uint8

const (
	// EntryOriginAppended is the origin of an entry committed through
	// [History.Append] or through [NewSeededHistory].
	EntryOriginAppended EntryOrigin = iota + 1

	// EntryOriginSynthesized is the origin of an entry committed by
	// [History.SynthesizeOrphans] — an interruption artifact, not a real
	// tool answer. It is the only route that can mint this origin.
	EntryOriginSynthesized
)

// String renders the origin, or "entryorigin(N)" for a non-member,
// deliberately a shape no parser accepts, mirroring [ai.Role.String]'s
// posture.
func (o EntryOrigin) String() string {
	switch o {
	case EntryOriginAppended:
		return "appended"
	case EntryOriginSynthesized:
		return "synthesized"
	default:
		return "entryorigin(" + strconv.Itoa(int(o)) + ")"
	}
}

// EntryID is an entry's stable identity: its 1-based ordinal position in
// the transcript at commit time (R-HIS-005). It is deterministic — the
// same seed yields the same identities in any process — and no caller can
// mint one: [Entry] has no constructor, so the C1 back door stays closed.
// The zero value is the identity of an entry that was never committed.
// uint32 matches R-AMT-007's ordinal convention.
type EntryID uint32

// Entry is the opaque envelope every committed value travels in
// (R-HIS-005). Its fields are unexported; every accessor is read-only, and
// history_surface_guard_test.go asserts the exported method set carries no
// route to set an entry identity, an origin discriminator, or a committed
// Layer 1 value (S-HIS-042).
type Entry struct {
	id      EntryID
	message ai.Message
	origin  EntryOrigin
}

// ID returns the entry's stable identity. It is 0 on the zero [Entry].
func (e Entry) ID() EntryID { return e.id }

// Message returns the unmodified Layer 1 value this entry carries —
// neither re-serialised nor rewritten (R-HIS-005). `ai.Message` is itself
// copy-on-read, so this accessor never aliases internal storage on its
// own.
func (e Entry) Message() ai.Message { return e.message }

// Origin returns the entry's origin discriminator (R-HIS-007):
// [EntryOriginAppended] for a real transcript entry, [EntryOriginSynthesized]
// for an interruption artifact [History.SynthesizeOrphans] committed.
func (e Entry) Origin() EntryOrigin { return e.origin }

// String renders the entry for a diagnostic reader, naming its identity
// and origin and never a byte of its message content (V-FAIL-13's
// redaction posture, carried into Layer 2).
func (e Entry) String() string {
	if e.id == 0 {
		return "entry(unset)"
	}
	return "entry(" + strconv.Itoa(int(e.id)) + " " + e.origin.String() + ")"
}

// openCall is the unexported pairing-index record for one unanswered tool
// call, in issuance order. entryIndex and partIndex locate the call's
// position in the transcript so a rejection at turn close (R-HIS-003) can
// name the first-issued offender's exact position.
type openCall struct {
	callID     string
	entryIndex int // the call's message position in entries (0-based)
	partIndex  int // the call's part position within that message's content
}

// History is the ordered, append-only transcript store for one run
// (R-HIS-001). Its storage is unexported; every route that can extend or
// mutate it funnels through [History.commit] (R-HIS-004). The zero value
// is not usable as a history — construct one through [NewHistory] or
// [NewSeededHistory].
type History struct {
	constructed bool // set only by the two constructors; commit's first gate
	entries     []Entry
	open        []openCall // unanswered calls, issuance order
}

// commitOp discriminates which validated mutation [History.commit]
// performs. Unexported: no public signature accepts one, so a caller
// cannot select a commit path history.go itself did not offer.
type commitOp uint8

const (
	// commitAppend validates and appends one message (rules 1-3).
	commitAppend commitOp = iota + 1
)

// NewHistory constructs an empty, usable history. An empty transcript is
// trivially valid, so this constructor commits nothing and cannot fail.
func NewHistory() *History {
	return &History{constructed: true}
}

// Append validates and commits message as the next transcript entry
// (R-HIS-001), through [History.commit] (R-HIS-004). On rejection the
// transcript is left byte-unchanged — a failed commit is not a partial
// commit.
func (h *History) Append(message ai.Message) error {
	return h.commit(commitAppend, message, EntryOriginAppended)
}

// Entries returns a freshly allocated snapshot of the transcript, in
// order. Mutating the result cannot touch internal storage (S-HIS-002):
// the slice is copied, and [Entry]'s own fields are unexported value
// types.
func (h *History) Entries() []Entry {
	out := make([]Entry, len(h.entries))
	copy(out, h.entries)
	return out
}

// Len returns the number of committed entries.
func (h *History) Len() int { return len(h.entries) }

// checkConstructed reports whether h passed a constructor. It is read-only
// — commit stays the only function that writes h.entries or h.open — and
// is shared by every route that must reject the zero value before doing
// anything else, including a route like [History.SynthesizeOrphans] whose
// happy path might otherwise commit nothing and never reach commit at all.
func (h *History) checkConstructed() error {
	if !h.constructed {
		return ai.Invalid(ai.ErrEmpty, ai.At("history"))
	}
	return nil
}

// commit is the ONLY function that writes h.entries or h.open (R-HIS-004).
// Every public mutating route funnels through it. Rule 1 (constructed) is
// checked here, ahead of the op-specific rules, because it gates every
// route uniformly; op-specific rules are dispatched from here so a new
// rule always has exactly one place to be added.
func (h *History) commit(op commitOp, message ai.Message, origin EntryOrigin) error {
	if err := h.checkConstructed(); err != nil {
		return err
	}
	switch op {
	case commitAppend:
		return h.commitAppendOp(message, origin)
	default:
		return nil
	}
}

// commitAppendOp runs rule 2 (the message was built through
// `ai.NewMessage`) then commits. Composed with `ai.FirstFailure` so the
// documented order — which rule wins — is data a reviewer reads rather
// than control flow to trace, the convention this package's other
// constructors already use (run_events.go, event.go).
func (h *History) commitAppendOp(message ai.Message, origin EntryOrigin) error {
	entryIndex := len(h.entries)

	if err := ai.FirstFailure(
		func() *ai.Violation {
			if message.ID().IsZero() {
				return ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", entryIndex))
			}
			return nil
		},
	); err != nil {
		return err
	}

	h.entries = append(h.entries, Entry{id: EntryID(entryIndex + 1), message: message, origin: origin})
	return nil
}
