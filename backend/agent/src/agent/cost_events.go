// AG-06.2 — the cost family (R-APE-004, R-APE-005, VL2-EVT-07).
//
// Cost events record the model's usage: [NewCostTurn] emits per-turn
// token figures (PlacementTurn) and [NewCostSession] emits cumulative
// figures over the whole run (PlacementRun). Both payloads carry the
// same five token fields (inputTokens, outputTokens, cacheReadTokens,
// cacheWriteTokens, reasoningTokens) and a [CostLabel] discriminator
// (Estimate or Final).
//
// # Token-only shape (R-APE-004, ADR 0005 § D4 per 0003:613)
//
// The verdict on doc 0001 § 4.3 vs ADR 0005 § D4 wins: the cost
// payload is token-only. No field that could carry money, currency or
// price is admitted into the construction surface — this is
// mechanically asserted via reflection in cost_events_test.go
// (S-APE-083), not by doc-phrase check. Money joins the stream as
// Layer 3 enrichment; Layer 2's responsibility is tokens.
//
// # CostLabel discriminator (R-APE-005, doc 0001 § 7 G10)
//
// "Two dispositions worth their reasoning" — the label distinguishes
// figures emitted before the stream's final usage update
// (CostLabelEstimate) from the figure emitted on the final update
// (CostLabelFinal). Reasoning tokens only arrive on the final update;
// the label is the disambiguator for the adaptive reasoning protocol
// (`0003:662-665`).
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

// CostScope is the typed scope vocabulary a cost figure covers — the
// discriminator between per-turn and cumulative (R-APE-004,
// 0003:656-661). The two kinds `cost_turn` and `cost_session` are
// distinct, mirroring the AG-04 `RunOutcome` / AG-05 `ToolOutcome`
// pattern (typed-value-in-one-kind > kinds-by-discriminator).
type CostScope uint8

const (
	// _ CostScope = iota skips zero so it is NOT a member.
	_ CostScope = iota

	// CostScopeTurn is the scope for a per-turn figure (cost_turn
	// kind, PlacementTurn).
	CostScopeTurn

	// CostScopeSession is the scope for a cumulative figure
	// (cost_session kind, PlacementRun).
	CostScopeSession

	// costScopeLimit is the first value outside the vocabulary.
	costScopeLimit
)

// String renders the scope, or "unset"/"costscope(N)" for a non-member.
func (s CostScope) String() string {
	if s >= costScopeLimit {
		return "costscope(" + strconv.FormatUint(uint64(s), 10) + ")"
	}
	switch s {
	case CostScopeTurn:
		return "turn"
	case CostScopeSession:
		return "session"
	case 0:
		return "unset"
	default:
		return "costscope(" + strconv.FormatUint(uint64(s), 10) + ")"
	}
}

// CostLabel is the typed label vocabulary a cost figure carries —
// distinguishes figures emitted before the stream's final usage update
// (Estimate) from the figure emitted on the final update (Final),
// R-APE-005. The zero value is NOT a member; the constructor rejects
// it.
type CostLabel uint8

const (
	// _ CostLabel = iota skips zero so it is NOT a member.
	_ CostLabel = iota

	// CostLabelEstimate is the label for a cost_session figure emitted
	// as a running total while the run has not yet concluded — the
	// harness decided to run another logical turn, so more tokens may
	// follow (AG-16, R-APE-005). A cost_turn figure is never labelled
	// Estimate: per-turn usage is complete whenever it is known at
	// all, absence included (R-CST-001).
	CostLabelEstimate

	// CostLabelFinal is the label for every cost_turn figure, and for
	// the cost_session figure emitted at a run's terminal, correcting
	// every earlier estimate (AG-16, R-APE-005, R-CST-005). Reasoning
	// tokens only arrive on a turn's final usage update (doc 0001 § 7
	// G10); the run-scoped estimate/final axis is orthogonal to that
	// per-turn completeness.
	CostLabelFinal

	// costLabelLimit is the first value outside the vocabulary.
	costLabelLimit
)

// String renders the label, or "unset"/"costlabel(N)" for a
// non-member.
func (l CostLabel) String() string {
	if l >= costLabelLimit {
		return "costlabel(" + strconv.FormatUint(uint64(l), 10) + ")"
	}
	switch l {
	case CostLabelEstimate:
		return "estimate"
	case CostLabelFinal:
		return "final"
	case 0:
		return "unset"
	default:
		return "costlabel(" + strconv.FormatUint(uint64(l), 10) + ")"
	}
}

// CostFigures is the token-only payload shape every cost event carries
// (R-APE-004). Five uint64 fields, no money-suggesting field, no
// accumulated-snapshot — the mechanical reflection pin in
// cost_events_test.go (S-APE-083) asserts this surface against an
// exact field-name list and a forbidden-substring list.
//
// The shape is structurally enforced: adding a sixth field, renaming
// any field, or adding any field whose name contains a
// money-suggesting substring (money, amount, price, total, cents,
// dollar, usd, currency, fee, charge) breaks the test.
type CostFigures struct {
	// InputTokens is the count of input tokens consumed.
	InputTokens uint64

	// OutputTokens is the count of output tokens produced.
	OutputTokens uint64

	// CacheReadTokens is the count of tokens served from a
	// prompt-cache read.
	CacheReadTokens uint64

	// CacheWriteTokens is the count of tokens written to a
	// prompt-cache.
	CacheWriteTokens uint64

	// ReasoningTokens is the count of reasoning-tokens produced.
	// Reasoning tokens only arrive on the final usage update
	// (doc 0001 § 7 G10) — the [CostLabel] discriminator is the
	// disambiguator.
	ReasoningTokens uint64
}

// CostTurn is the payload carrying a per-turn cost figure (VL2-EVT-07,
// R-APE-004). It carries the [CostLabel] discriminator and the same
// five token fields as [CostSession]. cost_turn is PlacementTurn.
type CostTurn struct {
	label   CostLabel
	figures CostFigures
	// presence records, per figure, whether Layer 1 reported it
	// (AG-16, R-CST-002). Beside figures, never inside it — CostFigures
	// stays byte-unchanged (S-APE-083). Read only through the paired
	// accessors below.
	presence costPresence
}

func (CostTurn) kind() EventKind { return EventKindCostTurn }

// validate reports CostTurn's own broken rule, or nil: the label is a
// member of the closed [CostLabel] vocabulary.
func (c CostTurn) validate(at ai.Path) *ai.Violation {
	if c.label == 0 || c.label >= costLabelLimit {
		return ai.Invalid(ai.ErrNotInVocabulary, under(at, ai.At("label"))...)
	}
	return nil
}

// NewCostTurn constructs a per-turn cost event (R-APE-004). The
// label is required and MUST be a member of the closed [CostLabel]
// vocabulary. On failure the zero [Event] is returned.
func NewCostTurn(run RunID, turn TurnID, label CostLabel, figures CostFigures) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	// AG-16 (R-APE-004): a caller handing five plain figures through
	// this public constructor is asserting five figures — presence is
	// all-reported. The presence-preserving construction path is the
	// package-private sibling newCostTurnFromUsage (cost_usage.go).
	payload := CostTurn{label: label, figures: figures, presence: allReportedCostPresence}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// CostTurn returns the event's cost-turn payload, and whether the
// event carries one. Called on an event of another kind, or on an
// event that was never constructed, it returns the zero value and
// false, and never panics.
func (e Event) CostTurn() (CostTurn, bool) {
	payload, ok := e.payload.(CostTurn)
	return payload, ok
}

// Scope returns [CostScopeTurn] — every cost_turn event is per-turn.
func (c CostTurn) Scope() CostScope { return CostScopeTurn }

// Label returns the typed [CostLabel] discriminator.
func (c CostTurn) Label() CostLabel { return c.label }

// Figures returns the five-field token figures.
func (c CostTurn) Figures() CostFigures { return c.figures }

// InputTokens returns the input-token count together with whether
// Layer 1 reported it (AG-16, R-CST-002). (0, false) means absent;
// (0, true) means a reported nought — the two are never conflated.
func (c CostTurn) InputTokens() (uint64, bool) { return c.figures.InputTokens, c.presence.input }

// OutputTokens returns the output-token count together with whether
// Layer 1 reported it (AG-16, R-CST-002).
func (c CostTurn) OutputTokens() (uint64, bool) { return c.figures.OutputTokens, c.presence.output }

// CacheReadTokens returns the cache-read-token count together with
// whether Layer 1 reported it (AG-16, R-CST-002).
func (c CostTurn) CacheReadTokens() (uint64, bool) {
	return c.figures.CacheReadTokens, c.presence.cacheRead
}

// CacheWriteTokens returns the cache-write-token count together with
// whether Layer 1 reported it (AG-16, R-CST-002).
func (c CostTurn) CacheWriteTokens() (uint64, bool) {
	return c.figures.CacheWriteTokens, c.presence.cacheWrite
}

// ReasoningTokens returns the reasoning-token count together with
// whether Layer 1 reported it (AG-16, R-CST-002).
func (c CostTurn) ReasoningTokens() (uint64, bool) {
	return c.figures.ReasoningTokens, c.presence.reasoning
}

// String renders the payload for a diagnostic reader, naming the
// type, the label and the input token count — never the figures
// in full.
func (c CostTurn) String() string {
	return "cost_turn(label=" + c.label.String() + " input=" + strconv.FormatUint(c.figures.InputTokens, 10) + ")"
}

// GoString renders the payload for the %#v verb.
func (c CostTurn) GoString() string { return c.String() }

// CostSession is the payload carrying a cumulative session-scope cost
// figure (VL2-EVT-07, R-APE-004). It carries the [CostLabel]
// discriminator and the same five token fields as [CostTurn].
// cost_session is PlacementRun — a session marker spans the whole run.
type CostSession struct {
	label   CostLabel
	figures CostFigures
	// presence records, per figure, whether Layer 1 reported it
	// (AG-16, R-CST-002). Beside figures, never inside it — CostFigures
	// stays byte-unchanged (S-APE-083). Read only through the paired
	// accessors below.
	presence costPresence
}

func (CostSession) kind() EventKind { return EventKindCostSession }

// validate reports CostSession's own broken rule, or nil: the label is
// a member of the closed [CostLabel] vocabulary.
func (c CostSession) validate(at ai.Path) *ai.Violation {
	if c.label == 0 || c.label >= costLabelLimit {
		return ai.Invalid(ai.ErrNotInVocabulary, under(at, ai.At("label"))...)
	}
	return nil
}

// NewCostSession constructs a cumulative session-scope cost event
// (R-APE-004). run is required (no turn — a session marker spans the
// whole run, PlacementRun). The label is required and MUST be a
// member of the closed [CostLabel] vocabulary. On failure the zero
// [Event] is returned.
func NewCostSession(run RunID, label CostLabel, figures CostFigures) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	// AG-16 (R-APE-004): a caller handing five plain figures through
	// this public constructor is asserting five figures — presence is
	// all-reported. The presence-preserving construction path is the
	// package-private sibling newCostSessionFromTotals (cost_usage.go).
	payload := CostSession{label: label, figures: figures, presence: allReportedCostPresence}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run}, nil
}

// CostSession returns the event's cost-session payload, and whether
// the event carries one. Called on an event of another kind, or on
// an event that was never constructed, it returns the zero value and
// false, and never panics.
func (e Event) CostSession() (CostSession, bool) {
	payload, ok := e.payload.(CostSession)
	return payload, ok
}

// Scope returns [CostScopeSession] — every cost_session event is
// cumulative over the run.
func (c CostSession) Scope() CostScope { return CostScopeSession }

// Label returns the typed [CostLabel] discriminator.
func (c CostSession) Label() CostLabel { return c.label }

// Figures returns the five-field token figures.
func (c CostSession) Figures() CostFigures { return c.figures }

// InputTokens returns the cumulative input-token count together with
// whether any contributing turn reported it (AG-16, R-CST-002,
// R-CST-004). (0, false) means no contributing turn ever reported this
// figure; (0, true) means it was reported and summed to a nought.
func (c CostSession) InputTokens() (uint64, bool) { return c.figures.InputTokens, c.presence.input }

// OutputTokens returns the cumulative output-token count together
// with whether any contributing turn reported it (AG-16, R-CST-002).
func (c CostSession) OutputTokens() (uint64, bool) { return c.figures.OutputTokens, c.presence.output }

// CacheReadTokens returns the cumulative cache-read-token count
// together with whether any contributing turn reported it (AG-16,
// R-CST-002).
func (c CostSession) CacheReadTokens() (uint64, bool) {
	return c.figures.CacheReadTokens, c.presence.cacheRead
}

// CacheWriteTokens returns the cumulative cache-write-token count
// together with whether any contributing turn reported it (AG-16,
// R-CST-002).
func (c CostSession) CacheWriteTokens() (uint64, bool) {
	return c.figures.CacheWriteTokens, c.presence.cacheWrite
}

// ReasoningTokens returns the cumulative reasoning-token count
// together with whether any contributing turn reported it (AG-16,
// R-CST-002).
func (c CostSession) ReasoningTokens() (uint64, bool) {
	return c.figures.ReasoningTokens, c.presence.reasoning
}

// String renders the payload for a diagnostic reader, naming the
// type, the label and the input token count — never the figures
// in full.
func (c CostSession) String() string {
	return "cost_session(label=" + c.label.String() + " input=" + strconv.FormatUint(c.figures.InputTokens, 10) + ")"
}

// GoString renders the payload for the %#v verb.
func (c CostSession) GoString() string { return c.String() }
