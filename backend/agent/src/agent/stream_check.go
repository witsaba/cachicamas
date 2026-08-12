// AG-04.1/AG-04.2 — the stream-contract validator (R-AEV-002, R-AEV-004,
// R-AEV-005, R-AEV-006, design AD-1).
//
// `agent.CheckStream` mirrors `ai.CheckStream`'s architecture: kind →
// registry → [EventDescriptor] → a generic rule engine that never
// special-cases a kind by name and never reads a concrete payload type.
// The Layer 2 descriptor adds one new dimension as data — [BracketRole] —
// so the engine holds a fixed two-level scope state machine (run open → at
// most one turn open) driven purely by descriptor fields plus envelope
// identity (kind, sequence, run, turn — never payload data).
//
// Two deliberate divergences from `ai.CheckStream`, both evidence-forced
// (design AD-1):
//
//  1. Contiguity is checked here, in the validator itself, not deferred to
//     a test kit — `backend/agent/src/ai/stream_check.go:52-54` excludes it
//     and gives it to `agenttest.CheckContiguity`; AG-04.1's own scenario
//     names this validator as what proves "independent, contiguous,
//     1-based", and design AD-4 rules `agenttest` out of AG-04 entirely.
//  2. A complete run bracket is REQUIRED, not optional: an empty or nil
//     slice, a sequence with no run-start, no run-end, or an unclosed turn
//     at run-end are all rejected. Layer 2 has no "sometimes no terminal
//     at all" case at run scope, unlike Layer 1's sanctioned bare-close
//     loss path.
//
// [Placement] and the at-most-one half of [Cardinality] are AG-05's and
// AG-06's own seams: no kind AG-04 registers exercises either, and the
// engine implements them anyway so a later milestone extends the rule
// table by registering a descriptor, with no change to this file
// (R-AEV-010's structural-extensibility half, proven by S-AEV-092's
// scratch experiment).

package agent

import "github.com/cachicamas/backend/agent/src/ai"

// StreamReport is CheckStream's verdict: the first violation found, in
// stream order, or nil when the stream satisfies every rule this file
// checks — AI-04's failure value, no second failure type, no new
// sentinel (design AD-6).
type StreamReport struct{ err error }

// Violation returns the first violation [CheckStream] found, in stream
// order, or nil when the stream is valid.
func (r StreamReport) Violation() error { return r.err }

// runBracketState is CheckStream's own run-scope state: unopened, open, or
// closed. Unexported — it is validator-internal bookkeeping, never a
// public concept.
type runBracketState uint8

const (
	runBracketUnopened runBracketState = iota
	runBracketOpen
	runBracketClosed
)

// CheckStream runs AG-04's ordering invariants against events, a finite
// ordered slice offered after the fact — never a channel, and never
// mutated (R-AEV-006). It is production-exported and callable from
// another package with no test-only build tag, because it is reused
// wholesale by the Layer 3 readiness contract's kit at AG-23
// (VL2-SEAM-14).
//
// It reports the first violation in stream order, naming the offending
// event's own 0-based index within events — never its [Sequence] value,
// which may itself be the value under rejection and can point outside the
// slice entirely (post-verify correction: the sequence value and the
// slice index coincide only for an already-valid prefix). This mirrors
// `agenttest.CheckContiguity`'s own convention for the identical class of
// position (`event[0]`, `event[1]`, ... from `range`), and the dominant
// convention across Layer 1's own `ai.AtIndex` call sites. Rules, checked
// in this order for each event:
//
//  1. contiguity — the event's [Sequence] equals the running expectation,
//     1-based;
//  2. nothing follows the terminal — no event may appear once an event
//     whose [EventDescriptor.Terminal] is true has occurred;
//  3. the bracket transition its [BracketRole] implies: opening the run
//     bracket twice, closing it with a still-open turn, opening a second
//     turn before the first closes, or closing a turn that is not open
//     are all rejected;
//  4. every event other than the one opening the run bracket must occur
//     while the run bracket is open;
//  5. a [PlacementTurn] kind must occur while a turn is open (AG-05's
//     seam; unused by any kind AG-04 registers);
//  6. a [CardinalityAtMostOne] kind may not repeat (AG-06's seam; unused
//     by any kind AG-04 registers).
//
// At the end of the slice, an unclosed run bracket — including an empty
// slice, which carries none at all — is rejected.
func CheckStream(events []Event) StreamReport {
	var seqExpect Sequence = 1
	run := runBracketUnopened
	var turnOpen bool
	var openTurn TurnID
	var terminalSeen bool
	seenAtMostOnce := make(map[EventKind]bool)

	for i, e := range events {
		seq := e.Sequence()

		if seq != seqExpect {
			return violation(ai.ErrOutOfRange, ai.AtIndex("event", i))
		}
		seqExpect++

		if terminalSeen {
			return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
		}

		entry, ok := eventRegistryEntry(e.Kind())
		if !ok {
			// An unregistered or zero-kind event is CheckEmit's concern,
			// not this validator's: it carries no descriptor to read, so
			// none of the rules below apply to it.
			continue
		}
		d := entry.descriptor

		switch d.Bracket {
		case BracketRoleOpensRun:
			if run != runBracketUnopened {
				return violation(ai.ErrDuplicate, ai.AtIndex("event", i))
			}
			run = runBracketOpen

		case BracketRoleClosesRun:
			if run != runBracketOpen {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
			if turnOpen {
				return violation(ai.ErrMalformed, ai.AtIndex("event", i))
			}
			run = runBracketClosed

		case BracketRoleOpensTurn:
			if run != runBracketOpen {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
			if turnOpen {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
			turnOpen = true
			openTurn, _ = e.Turn()

		case BracketRoleClosesTurn:
			if run != runBracketOpen {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
			turnID, _ := e.Turn()
			if !turnOpen || turnID != openTurn {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
			turnOpen = false

		default: // BracketRoleNone — AG-05/AG-06's own kinds.
			if run != runBracketOpen {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
			if d.Placement == PlacementTurn && !turnOpen {
				return violation(ai.ErrMisplaced, ai.AtIndex("event", i))
			}
		}

		if d.Cardinality == CardinalityAtMostOne {
			if seenAtMostOnce[e.Kind()] {
				return violation(ai.ErrDuplicate, ai.AtIndex("event", i))
			}
			seenAtMostOnce[e.Kind()] = true
		}

		if d.Terminal {
			terminalSeen = true
		}
	}

	if run != runBracketClosed {
		// Covers "no run-end", "unclosed at end of slice", and the empty
		// slice uniformly: an empty slice never opens a run bracket, so it
		// carries none — design AD-1 divergence 2.
		return violation(ai.ErrMalformed, ai.At("stream"))
	}
	return StreamReport{}
}

// violation is CheckStream's one constructor for a failing StreamReport,
// mirroring `ai.stream_check.go`'s own `verdict` helper: one place that
// maps a rule class and position to a report, instead of a composite
// literal repeated at every call site.
func violation(rule error, at ...ai.Step) StreamReport {
	return StreamReport{err: ai.Invalid(rule, at...)}
}
