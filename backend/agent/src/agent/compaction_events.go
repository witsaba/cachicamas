// AG-06.4 — the compaction family (R-APE-007, R-APE-008, VL2-EVT-09).
//
// Compaction events record a context-compaction lifecycle:
// [NewCompactionStarted] opens it, [NewCompactionFinished] closes it
// carrying the [startTurnID, endTurnID] turn bracket and a
// [summaryID], and [NewCompactionFailed] declares `Terminal: false`
// (not the zero value — explicit per AG-05 S1 carry-forward) and
// carries a typed [Failure] (R-AEV-008). The engine accepts follow-on
// events after a failed compaction but does NOT synthesize recovery
// — recovery requires a new [NewCompactionStarted] (S-APE-084).
//
// # Turn bracket identity (R-APE-007, doc 0001 § 7 G3)
//
// Turns are the protected unit. The span identity is
// `[startTurnID TurnID, endTurnID TurnID]` — sequence-only fails for
// multi-lane streams (per-lane `Sequence` is meaningless across
// lanes), ordinal-only fails for the same reason. Turns are the only
// per-run identity the validator already tracks (`event.go:354-366`,
// `S-AEV-004`).
//
// # Recovery after failure (R-APE-008)
//
// `compaction_failed` declares `Terminal: false` — honoring `doc 0001
// § 7 G3` "survive interruption". The engine accepts but does NOT
// synthesize recovery: a follow-on `compaction_started` is required
// (S-APE-084). The distinct kind from `compaction_finished` keeps the
// session log honest about which compactions succeeded.
//
// # Per-family file split (AD-6, AG-05 precedent)
//
// Payload + ctors + kind constants + EventDescriptor rows are
// co-located, mirroring message_text.go and tool_event.go. The same
// package owns the EventKind block (event.go) and the eventRegistry
// rows — this file's constants are referenced by event.go's iota
// continuation; a new kind declared here is wired through by the same
// commit.

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// CompactionSpan is the bracket identity a compaction_finished event
// carries (R-APE-007). Turns are the protected unit — startTurnID
// and endTurnID are typed TurnID, NOT sequence values (sequence-only
// fails for multi-lane streams) and NOT ordinals (the same).
type CompactionSpan struct {
	StartTurnID TurnID
	EndTurnID   TurnID
}

// validate reports CompactionSpan's own broken rule, or nil: the
// bracket is non-empty (`EndTurnID >= StartTurnID`).
func (s CompactionSpan) validate(at ai.Path) *ai.Violation {
	if s.StartTurnID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("start_turn_id"))...)
	}
	if s.EndTurnID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("end_turn_id"))...)
	}
	if s.EndTurnID < s.StartTurnID {
		return ai.Invalid(ai.ErrOutOfRange, under(at, ai.At("end_turn_id"))...)
	}
	return nil
}

// CompactionStarted is the payload opening a context-compaction
// bracket (VL2-EVT-09, R-APE-007). It carries the compaction identity
// — enough for the session log to correlate the compaction's start
// and end events.
type CompactionStarted struct {
	compactionID string
}

func (CompactionStarted) kind() EventKind { return EventKindCompactionStarted }

// validate reports CompactionStarted's own broken rule, or nil: the
// compaction identity is required.
func (c CompactionStarted) validate(at ai.Path) *ai.Violation {
	if c.compactionID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("compaction_id"))...)
	}
	return nil
}

// NewCompactionStarted constructs the event opening a context-
// compaction bracket (R-APE-007). run is required; turn is required
// (compaction events are PlacementTurn, not PlacementRun — a
// compaction is a turn-scoped operation). compactionID is required.
// On failure the zero [Event] is returned.
func NewCompactionStarted(run RunID, turn TurnID, compactionID string) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	if compactionID == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("compaction_id"))
	}
	payload := CompactionStarted{compactionID: compactionID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// CompactionStarted returns the event's compaction-started payload,
// and whether the event carries one. Called on an event of another
// kind, or on an event that was never constructed, it returns the
// zero value and false, and never panics.
func (e Event) CompactionStarted() (CompactionStarted, bool) {
	payload, ok := e.payload.(CompactionStarted)
	return payload, ok
}

// CompactionID returns the compaction's identity the event opens.
func (c CompactionStarted) CompactionID() string { return c.compactionID }

// String renders the payload for a diagnostic reader, naming the type
// and the compaction identity.
func (c CompactionStarted) String() string {
	return "compaction_started(" + strconv.Quote(c.compactionID) + ")"
}

// GoString renders the payload for the %#v verb.
func (c CompactionStarted) GoString() string { return c.String() }

// CompactionFinished is the payload closing a context-compaction
// bracket (VL2-EVT-09, R-APE-007). It carries the compaction identity
// (matching its opener), the [startTurnID, endTurnID] turn bracket
// (turns are the protected unit), and a summaryID identifying the
// persisted summary — enough for a session log to record the
// operation.
type CompactionFinished struct {
	compactionID string
	span         CompactionSpan
	summaryID    string
}

func (CompactionFinished) kind() EventKind { return EventKindCompactionFinished }

// validate reports CompactionFinished's own broken rule, or nil: the
// compaction identity is required, the summary identity is required
// (distinct from run identity — the spec scenario requires this), and
// the span bracket is non-empty.
func (c CompactionFinished) validate(at ai.Path) *ai.Violation {
	if c.compactionID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("compaction_id"))...)
	}
	if c.summaryID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("summary_id"))...)
	}
	if violation := c.span.validate(under(at, ai.At("span"))); violation != nil {
		return violation
	}
	return nil
}

// NewCompactionFinished constructs the event closing a context-
// compaction bracket (R-APE-007). run and turn are required;
// compactionID, span, and summaryID are all required. On failure the
// zero [Event] is returned.
func NewCompactionFinished(run RunID, turn TurnID, compactionID string, span CompactionSpan, summaryID string) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := CompactionFinished{
		compactionID: compactionID,
		span:         span,
		summaryID:    summaryID,
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// CompactionFinished returns the event's compaction-finished payload,
// and whether the event carries one. Called on an event of another
// kind, or on an event that was never constructed, it returns the
// zero value and false, and never panics.
func (e Event) CompactionFinished() (CompactionFinished, bool) {
	payload, ok := e.payload.(CompactionFinished)
	return payload, ok
}

// CompactionID returns the compaction's identity the event closes.
func (c CompactionFinished) CompactionID() string { return c.compactionID }

// Span returns the turn-bracket identity [startTurnID, endTurnID].
func (c CompactionFinished) Span() CompactionSpan { return c.span }

// SummaryID returns the persisted summary's identity — distinct from
// the run identity, so a session log can persist the summary
// separately from the run record.
func (c CompactionFinished) SummaryID() string { return c.summaryID }

// String renders the payload for a diagnostic reader, naming the
// type, the compaction identity, the bracket, and the summary
// identity.
func (c CompactionFinished) String() string {
	return "compaction_finished(" + strconv.Quote(c.compactionID) + " span=" + string(c.span.StartTurnID) + ".." + string(c.span.EndTurnID) + " summary=" + strconv.Quote(c.summaryID) + ")"
}

// GoString renders the payload for the %#v verb.
func (c CompactionFinished) GoString() string { return c.String() }

// CompactionFailed is the payload closing a context-compaction
// bracket with a typed failure (VL2-EVT-09, R-APE-008). It is a
// distinct kind from [CompactionFinished] — the session log MUST be
// able to tell successful compactions from failed ones by kind, not
// by payload convention. It declares `Terminal: false` (explicit, not
// the zero value — AG-05 S1 carry-forward): the engine accepts
// follow-on events (a subsequent `compaction_started` may re-emit) but
// does NOT synthesize recovery.
type CompactionFailed struct {
	compactionID string
	failure      *Failure
}

func (CompactionFailed) kind() EventKind { return EventKindCompactionFailed }

// validate reports CompactionFailed's own broken rule, or nil: the
// compaction identity is required and the typed failure is required
// (R-AEV-008).
func (c CompactionFailed) validate(at ai.Path) *ai.Violation {
	if c.compactionID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("compaction_id"))...)
	}
	if c.failure == nil {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("failure"))...)
	}
	return nil
}

// NewCompactionFailed constructs the event closing a context-
// compaction bracket with a typed failure (R-APE-008). run and turn
// are required; compactionID is required; failure is required
// (R-AEV-008). On failure the zero [Event] is returned.
func NewCompactionFailed(run RunID, turn TurnID, compactionID string, failure *Failure) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := CompactionFailed{
		compactionID: compactionID,
		failure:      failure,
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// CompactionFailed returns the event's compaction-failed payload,
// and whether the event carries one. Called on an event of another
// kind, or on an event that was never constructed, it returns the
// zero value and false, and never panics.
func (e Event) CompactionFailed() (CompactionFailed, bool) {
	payload, ok := e.payload.(CompactionFailed)
	return payload, ok
}

// CompactionID returns the compaction's identity the event closes.
func (c CompactionFailed) CompactionID() string { return c.compactionID }

// Failure returns the typed [Failure] that caused the compaction to
// fail (R-AEV-008).
func (c CompactionFailed) Failure() (*Failure, bool) { return c.failure, c.failure != nil }

// String renders the payload for a diagnostic reader, naming the
// type, the compaction identity and the failure — never the
// failure's own content.
func (c CompactionFailed) String() string {
	return "compaction_failed(" + strconv.Quote(c.compactionID) + ")"
}

// GoString renders the payload for the %#v verb.
func (c CompactionFailed) GoString() string { return c.String() }
