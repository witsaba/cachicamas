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
// [NewSeededHistory], [History.CloseTurn], [History.SynthesizeOrphans],
// and AG-18's [History.ReplacePrefix] — funnels through the single
// unexported [History.commit] primitive (R-HIS-004, the C1 lesson (doc
// 0001:376) reapplied to history). Bypass is impossible by construction,
// not convention: entries, open and marks are unexported, so no external
// caller can reach them; commit is the only function whose body assigns
// them, auditable by reading this one file;
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

	// EntryOriginSummarized is the origin of the one summary entry a
	// prefix replacement commits (AG-18, R-CMP-007). It is mintable
	// ONLY by [History.ReplacePrefix]'s own commit op. Distinguishability
	// from an ordinarily-appended assistant message rides this
	// discriminator alone — never content, never a failure flag
	// (R-HIS-007's content-sentinel prohibition, carried forward).
	EntryOriginSummarized
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
	case EntryOriginSummarized:
		return "summarized"
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

// turnMark is a turn identifier together with the entry count at that
// turn's close (AG-18, R-CMP-012, design AD-9): {TurnID, count}, where
// count is the number of entries committed at the moment the turn
// closed — the same "prefix length" coordinate a compaction cut is
// denominated in. Written only through [History.commit]'s marked-close
// op, never any other way. A recorded mark is a structural guarantee: it
// can only be committed when the open-call set was empty at that
// instant (commitCloseTurnMarkedOp delegates to commitCloseTurnOp's own
// rule 4), so every prefix ending at a mark's count is pairing-closed by
// construction — a fact [resolveCut] and [History.commitReplacePrefixOp]
// both rely on to avoid a second, redundant correlation scan.
type turnMark struct {
	turnID TurnID
	count  int
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
	marks       []turnMark // recorded turn-close boundaries, count order (AG-18)
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

	// commitCloseTurnMarked runs commitCloseTurn's identical rule 4, then
	// additionally records a turn mark (AG-18, R-CMP-012). Reachable
	// only through the package-private [History.closeTurnMarked] door —
	// the exported [History.CloseTurn] keeps its pre-AG-18 signature and
	// semantics, an unmarked close.
	commitCloseTurnMarked

	// commitReplacePrefix validates and commits AG-18's one new removal
	// shape: a prefix replacement (R-HIS-001 as amended, R-CMP-005),
	// reachable only through the exported [History.ReplacePrefix].
	commitReplacePrefix
)

// commitParams carries every op's own parameters through the single
// [History.commit] dispatch point. Not every field is meaningful for
// every op — each commitXxxOp method reads only the fields its own op
// uses — mirroring the pre-AG-18 shape where commitCloseTurnOp already
// ignored the message/origin parameters it was handed.
type commitParams struct {
	message ai.Message
	origin  EntryOrigin
	turnID  TurnID // commitCloseTurnMarked
	count   int    // commitReplacePrefix
}

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
		if err := h.commit(commitAppend, commitParams{message: m, origin: EntryOriginAppended}); err != nil {
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
	return h.commit(commitAppend, commitParams{message: message, origin: EntryOriginAppended})
}

// CloseTurn validates that every tool call in the transcript has a
// matching result and, if so, closes the turn (R-HIS-003), through
// [History.commit] (R-HIS-004). "Once the turn closes", concretely: the
// turn closes when the caller invokes this method. History detects
// nothing itself — AG-13's run driver will call it when a provider turn
// ends; in AG-12 only tests exercise it. An empty open set is a no-op:
// closing an already-closed turn succeeds and changes nothing
// (idempotent).
//
// This is the UNMARKED close: it keeps its exact pre-AG-18 signature and
// semantics, and it does not record a turn mark. [History.closeTurnMarked]
// is the package-private, marked sibling AG-18 added beside it
// (R-CMP-012) — every existing caller of this exported method is
// unaffected.
func (h *History) CloseTurn() error {
	return h.commit(commitCloseTurn, commitParams{})
}

// closeTurnMarked runs CloseTurn's identical rule 4, then additionally
// records a turn mark {turnID, entry count} through the same commit
// primitive (AG-18, R-CMP-012, design AD-9). Package-private: reachable
// only from inside this package — the harness supplies turnID from the
// successful attempt's own forwarded turn bracket (S-CMP-035 asserts it
// is unreachable from package agent_test). A recorded mark is what lets
// a later compaction attribute a resolved cut to the turn identifiers
// that produced it, and what lets [resolveCut] and
// [History.commitReplacePrefixOp] both skip a redundant pairing re-scan:
// a mark can only be committed when the open-call set was empty at that
// instant, so every prefix ending at a mark's count is pairing-closed by
// construction.
func (h *History) closeTurnMarked(turnID TurnID) error {
	return h.commit(commitCloseTurnMarked, commitParams{turnID: turnID})
}

// ReplacePrefix replaces the prefix [0, count) with one summary entry
// (AG-18, R-HIS-001 as amended, R-CMP-005), through [History.commit]
// (R-HIS-004) — the sixth exported route, dispatched from the same
// single validating primitive so "commit is the ONLY function that
// writes h.entries, h.open or h.marks" stays literally true.
//
// count MUST name a recorded turn-mark boundary (R-HIS-001 item 3); a
// count that does not is rejected typed, and — because every mark
// guarantees pairing-closure of everything before it by construction —
// this one check subsumes the pairing-closure obligation without a
// second, redundant correlation re-scan. summary MUST carry a non-zero
// identity and no tool-call or tool-result part (R-CMP-005: a prefix
// replacement must not reopen the open set); its content and origin are
// otherwise taken verbatim — Layer 2 wraps, annotates or re-serialises
// nothing. On any validation failure h.entries, h.open and h.marks are
// left byte-unchanged (R-HIS-002, carried forward to this route).
//
// On success the new entry sequence is [summary, tail...], with entry
// identities renumbered ordinally from 1 (R-HIS-005) and every tail
// entry's own Layer 1 value and origin discriminator preserved
// positionally (R-CMP-006); marks fully inside the replaced prefix are
// dropped and every surviving mark's count is shifted to the new
// coordinate space.
func (h *History) ReplacePrefix(count int, summary ai.Message) error {
	return h.commit(commitReplacePrefix, commitParams{count: count, message: summary})
}

// synthesizedInterruptionContent is the fixed content every synthesized
// interruption result carries. It informs the model but is NOT the
// discriminator: a real tool can emit the identical bytes, so
// distinguishability rides the entry envelope's origin alone (R-HIS-007,
// proposal decision 4's corollary).
const synthesizedInterruptionContent = "tool call interrupted before a result arrived; this result was synthesized by the harness"

// SynthesizeOrphans completes every orphaned tool call with a result
// recorded as an interruption artifact (R-HIS-007), routed through
// [History.commit] like every other mutating route (R-HIS-004) — this is
// not a second, unaudited door.
//
// The open set is snapshotted in issuance order and every synthesized
// message is built BEFORE any commit is attempted, so a construction
// failure — impossible in practice, since a call's identity is
// non-empty by ai.NewToolCall's own invariant, but still handled rather
// than panicked — commits nothing rather than half-repairing the
// transcript.
//
// It is idempotent and total (R-HIS-008): the first application closes
// exactly N pairs and returns (N, nil); a second application finds no
// orphans, commits nothing, and returns (0, nil) with a byte-identical
// transcript. `new(History)` is rejected before anything is built or
// committed (S-HIS-030's zero-value proof for this route).
func (h *History) SynthesizeOrphans() (int, error) {
	if err := h.checkConstructed(); err != nil {
		return 0, err
	}

	orphans := make([]openCall, len(h.open))
	copy(orphans, h.open)
	if len(orphans) == 0 {
		return 0, nil
	}

	messages := make([]ai.Message, 0, len(orphans))
	for _, orphan := range orphans {
		part, err := ai.NewToolFailure(orphan.callID, synthesizedInterruptionContent)
		if err != nil {
			return 0, err
		}
		message, err := ai.NewMessage(ai.RoleTool, part)
		if err != nil {
			return 0, err
		}
		messages = append(messages, message)
	}

	for _, message := range messages {
		if err := h.commit(commitAppend, commitParams{message: message, origin: EntryOriginSynthesized}); err != nil {
			return 0, err
		}
	}
	return len(messages), nil
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

// commit is the ONLY function that writes h.entries, h.open or h.marks
// (R-HIS-004, AD-9: "commit writes everything"). Every public mutating
// route funnels through it. Rule 1 (constructed) is checked here, ahead
// of the op-specific rules, because it gates every route uniformly;
// op-specific rules are dispatched from here so a new rule always has
// exactly one place to be added.
func (h *History) commit(op commitOp, p commitParams) error {
	if err := h.checkConstructed(); err != nil {
		return err
	}
	switch op {
	case commitAppend:
		return h.commitAppendOp(p.message, p.origin)
	case commitCloseTurn:
		return h.commitCloseTurnOp()
	case commitCloseTurnMarked:
		return h.commitCloseTurnMarkedOp(p.turnID)
	case commitReplacePrefix:
		return h.commitReplacePrefixOp(p.count, p.message)
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

// commitCloseTurnMarkedOp runs commitCloseTurnOp's identical rule 4 by
// delegation, then — on success only — records a turn mark {turnID,
// len(h.entries)} (AG-18, R-CMP-012). turnID is required: an empty
// identity is rejected before the close rule even runs, since a mark
// with no identity could never be named by a later CompactionSpan.
func (h *History) commitCloseTurnMarkedOp(turnID TurnID) error {
	if turnID == "" {
		return ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	if err := h.commitCloseTurnOp(); err != nil {
		return err
	}
	h.marks = append(h.marks, turnMark{turnID: turnID, count: len(h.entries)})
	return nil
}

// markBoundaryAtOrBefore returns the greatest recorded turn-mark
// boundary <= cut, or 0 if none exists (R-CMP-004 step 1, design AD-1).
// Marks are always positive, so 0 unambiguously means "no mark at or
// before cut" — which [resolveCut] reads as "nothing to compact" at the
// end of its retraction. Read-only; never mutates h.
func (h *History) markBoundaryAtOrBefore(cut int) int {
	best := 0
	for _, m := range h.marks {
		if m.count <= cut && m.count > best {
			best = m.count
		}
	}
	return best
}

// markSpan returns the first and last recorded turn identifiers whose
// mark boundary falls within [0, cut], in mark order, and whether at
// least one such mark exists (R-CMP-012, design AD-9 span derivation).
// Every mark with count <= cut names a turn whose entries are entirely
// inside the replaced prefix — "the marks fully contained in [0, cut)".
// Read-only; never mutates h.
func (h *History) markSpan(cut int) (start, end TurnID, ok bool) {
	for _, m := range h.marks {
		if m.count > cut {
			continue
		}
		if !ok {
			start = m.turnID
			ok = true
		}
		end = m.turnID
	}
	return start, end, ok
}

// checkReplacePrefixCount reports count's own broken rule, or nil: count
// must be positive, must not exceed the current entry count, and MUST
// coincide with a recorded turn-mark boundary (R-HIS-001 item 3). The
// mark-membership check subsumes pairing-closure of [0, count) by the
// structural guarantee documented on [turnMark] — no separate
// correlation re-scan of the discarded prefix is needed here.
func checkReplacePrefixCount(marks []turnMark, count, entryCount int) *ai.Violation {
	if count <= 0 || count > entryCount {
		return ai.Invalid(ai.ErrOutOfRange, ai.At("count"))
	}
	for _, m := range marks {
		if m.count == count {
			return nil
		}
	}
	return ai.Invalid(ai.ErrOutOfRange, ai.At("count"))
}

// checkSummaryCarriesNoToolParts reports summary's own broken rule, or
// nil: a prefix-replacement summary must carry no ToolCall and no
// ToolResult part, so the replacement can never reopen the open set
// (R-CMP-005).
func checkSummaryCarriesNoToolParts(summary ai.Message) *ai.Violation {
	for partIndex, part := range summary.Content() {
		if _, ok := part.ToolCall(); ok {
			return ai.Invalid(ai.ErrMisplaced, ai.At("summary"), ai.AtIndex("content", partIndex))
		}
		if _, ok := part.ToolResult(); ok {
			return ai.Invalid(ai.ErrMisplaced, ai.At("summary"), ai.AtIndex("content", partIndex))
		}
	}
	return nil
}

// replacementOpenSet computes the open-call set the new [summary,
// tail...] sequence would carry, validating along the way that the
// sequence is pairing-closed by the SAME correlation [resolveOpenSet]
// runs for an ordinary append or a fresh seed (R-HIS-001 item 4,
// R-CMP-010 gate 3: "the same pairing rules seeded construction
// enforces"). It is read-only over its arguments — a scratch
// computation, never a partial mutation of h.
func replacementOpenSet(summary ai.Message, tail []Entry) ([]openCall, *ai.Violation) {
	var open []openCall
	var violation *ai.Violation
	open, violation = resolveOpenSet(open, summary, 0)
	if violation != nil {
		return nil, violation
	}
	for i, e := range tail {
		open, violation = resolveOpenSet(open, e.Message(), i+1)
		if violation != nil {
			return nil, violation
		}
	}
	return open, nil
}

// commitReplacePrefixOp validates and performs AG-18's one new removal
// shape (R-HIS-001 as amended, R-CMP-005, design AD-6): every check runs
// before any field is written, so a rejected replacement leaves
// h.entries, h.open and h.marks byte-unchanged (R-HIS-002, R-CMP-010).
func (h *History) commitReplacePrefixOp(count int, summary ai.Message) error {
	if err := ai.FirstFailure(
		func() *ai.Violation { return checkReplacePrefixCount(h.marks, count, len(h.entries)) },
		func() *ai.Violation {
			if summary.ID().IsZero() {
				return ai.Invalid(ai.ErrEmpty, ai.At("summary"))
			}
			return nil
		},
		func() *ai.Violation { return checkSummaryCarriesNoToolParts(summary) },
	); err != nil {
		return err
	}

	tail := h.entries[count:]
	newOpen, violation := replacementOpenSet(summary, tail)
	if violation != nil {
		return violation
	}

	newEntries := make([]Entry, 0, 1+len(tail))
	newEntries = append(newEntries, Entry{id: EntryID(1), message: summary, origin: EntryOriginSummarized})
	for i, e := range tail {
		newEntries = append(newEntries, Entry{id: EntryID(i + 2), message: e.Message(), origin: e.Origin()})
	}

	shift := count - 1
	newMarks := make([]turnMark, 0, len(h.marks))
	for _, m := range h.marks {
		if m.count <= count {
			continue
		}
		newMarks = append(newMarks, turnMark{turnID: m.turnID, count: m.count - shift})
	}

	h.entries = newEntries
	h.open = newOpen
	h.marks = newMarks
	return nil
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
