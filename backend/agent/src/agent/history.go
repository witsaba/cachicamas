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

	// commitCloseTurn validates that no call is left unanswered (rule 4).
	commitCloseTurn
)

// NewHistory constructs an empty, usable history. An empty transcript is
// trivially valid, so this constructor commits nothing and cannot fail.
func NewHistory() *History {
	return &History{constructed: true}
}

// NewSeededHistory constructs a history over a pre-existing transcript
// (R-HIS-006), re-running the SAME append-time rules [History.Append]
// runs over the whole seed, in order (decision 2, frozen at the AG-12
// handoff — session resume and next-run model switching stand on this
// seam). It accepts an ordered sequence of Layer 1 messages and nothing
// else: no parameter accepts a caller-supplied entry identity or origin
// discriminator (S-HIS-053) — the C1 back door stays closed here too.
//
// It ACCEPTS a seed that ends in one or more open calls. Seeding re-runs
// only the append-time rules of [History.Append] (R-HIS-002); it never
// applies the turn-close rule of [History.CloseTurn] (R-HIS-003). A
// seeded history is constructed with its turn open, exactly as an
// appended one is — this is what makes an interrupted transcript
// reconstructable on resume, and it is the precondition
// [History.SynthesizeOrphans] depends on (S-HIS-054).
//
// It rejects only a seed containing an orphaned RESULT — a tool-result
// entry naming a call the seed does not declare at that point — with the
// same rule class the equivalent append would produce, positioned at the
// first offending entry via [History.commit]'s own would-be-index
// positioning (S-HIS-051).
//
// On rejection it returns (nil, err): the zero history is never usable,
// so a caller cannot mistake a rejected construction for an empty valid
// transcript (S-HIS-052).
func NewSeededHistory(messages []ai.Message) (*History, error) {
	h := NewHistory()
	for _, m := range messages {
		if err := h.commit(commitAppend, m, EntryOriginAppended); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// Append validates and commits message as the next transcript entry
// (R-HIS-001), through [History.commit] (R-HIS-004). On rejection the
// transcript is left byte-unchanged — a failed commit is not a partial
// commit.
func (h *History) Append(message ai.Message) error {
	return h.commit(commitAppend, message, EntryOriginAppended)
}

// CloseTurn validates that every tool call in the transcript has a
// matching result and, if so, closes the turn (R-HIS-003), through
// [History.commit] (R-HIS-004). "Once the turn closes", concretely: the
// turn closes when the caller invokes this method. History detects
// nothing itself — AG-13's run driver will call it when a provider turn
// ends; in AG-12 only tests exercise it. An empty open set is a no-op:
// closing an already-closed turn succeeds and changes nothing
// (idempotent).
func (h *History) CloseTurn() error {
	return h.commit(commitCloseTurn, ai.Message{}, 0)
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
	case commitCloseTurn:
		return h.commitCloseTurnOp()
	default:
		return nil
	}
}

// commitCloseTurnOp runs rule 4: the open set must be empty, else the
// first-issued open call's result slot is missing (R-HIS-003, S-HIS-022's
// determinism). commitCloseTurn never writes h.entries or h.open — closing
// a turn commits no new entry, it only validates the ones already there.
func (h *History) commitCloseTurnOp() error {
	if len(h.open) == 0 {
		return nil
	}
	first := h.open[0]
	return ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", first.entryIndex), ai.AtIndex("content", first.partIndex), ai.At("result"))
}

// commitAppendOp runs rule 2 (the message was built through
// `ai.NewMessage`) and rule 3 (every `ToolResult` part resolves to an open
// call) then commits. Composed with `ai.FirstFailure` so the documented
// order — which rule wins — is data a reviewer reads rather than control
// flow to trace, the convention this package's other constructors already
// use (run_events.go, event.go).
//
// Rule 3's open-set delta is computed by [resolveOpenSet] into a local
// variable and only assigned to h.open after every rule has passed: state
// is written only once the whole message is proven valid (all-or-nothing —
// a violation on part 3 of 5 must not half-commit parts 0-2's pairing
// effects either).
func (h *History) commitAppendOp(message ai.Message, origin EntryOrigin) error {
	entryIndex := len(h.entries)
	var nextOpen []openCall

	if err := ai.FirstFailure(
		func() *ai.Violation {
			if message.ID().IsZero() {
				return ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", entryIndex))
			}
			return nil
		},
		func() *ai.Violation {
			resolved, violation := resolveOpenSet(h.open, message, entryIndex)
			nextOpen = resolved
			return violation
		},
	); err != nil {
		return err
	}

	h.entries = append(h.entries, Entry{id: EntryID(entryIndex + 1), message: message, origin: origin})
	h.open = nextOpen
	return nil
}

// resolveOpenSet computes the open set that would result from committing
// message at entryIndex, without mutating current (R-HIS-002, R-HIS-004:
// the caller commits the result only after every rule has passed). It
// walks message's content in order, per V-FAIL-04:
//
//   - a ToolResult part must name a call current (or an earlier part of
//     this same message) declares open, else the first such part fails
//     with ai.ErrUnresolvedReference at its own position, and the call is
//     removed from the returned set — "the open set is what the
//     transcript declares", so a second result for the same call is no
//     longer declared and fails by the same rule (no ErrDuplicate, no
//     third rule);
//   - a ToolCall part joins the returned set, in issuance order.
func resolveOpenSet(current []openCall, message ai.Message, entryIndex int) ([]openCall, *ai.Violation) {
	next := make([]openCall, len(current))
	copy(next, current)

	for partIndex, part := range message.Content() {
		if result, ok := part.ToolResult(); ok {
			answered := -1
			for i, oc := range next {
				if oc.callID == result.CallID() {
					answered = i
					break
				}
			}
			if answered < 0 {
				return nil, ai.Invalid(ai.ErrUnresolvedReference, ai.AtIndex("messages", entryIndex), ai.AtIndex("content", partIndex))
			}
			next = append(next[:answered], next[answered+1:]...)
			continue
		}
		if call, ok := part.ToolCall(); ok {
			next = append(next, openCall{callID: call.ID(), entryIndex: entryIndex, partIndex: partIndex})
		}
	}
	return next, nil
}
