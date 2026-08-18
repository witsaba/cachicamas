// Package agent is Layer 2 of the cachicamas agent stack. This file
// (cost_usage.go) hosts AG-16 — usage capture, the presence-preserving
// conversion, and the run-scoped cumulative accumulator
// (R-CST-001..006).
//
// This file is package-private end to end: no Layer 3 exists yet
// (0003:110), external tests read cost events through the paired
// accessors on cost_events.go, and a smaller construction surface
// keeps the rollback clean (design DD1). It never touches
// CostFigures, cost_events_test.go, or anything under src/ai/ — every
// figure this file produces is read out of ai.Usage through its own
// public TokenCount.Count() idiom (usage.go:63-65).
package agent

import "github.com/cachicamas/backend/agent/src/ai"

// costPresence records, per figure, whether Layer 1 reported it
// (R-CST-002). Unexported: consumers read presence only through the
// paired accessors on CostTurn/CostSession, so a count is never
// readable without its bit.
type costPresence struct {
	input, output, cacheRead, cacheWrite, reasoning bool
}

// allReportedCostPresence is every figure present — what NewCostTurn
// and NewCostSession set internally (R-APE-004): a caller handing
// five plain figures through those public constructors is asserting
// five figures.
var allReportedCostPresence = costPresence{input: true, output: true, cacheRead: true, cacheWrite: true, reasoning: true}

// costFromUsage is the ai.Usage → payload conversion (R-CST-003):
// total (defined for every ai.Usage value, the zero value included),
// pure (no I/O, no clock read, no emission), and presence-preserving
// in both directions. ai.Tokens(0) maps to a reported nought and is
// never conflated with absence; int64 → uint64 is safe because a
// completion's usage is validated non-negative before it ever reaches
// this function (ai.Completion.validate, usage.go:151-169).
func costFromUsage(u ai.Usage) (CostFigures, costPresence) {
	input, inputOk := u.Input.Count()
	output, outputOk := u.Output.Count()
	cacheRead, cacheReadOk := u.CacheRead.Count()
	cacheWrite, cacheWriteOk := u.CacheWrite.Count()
	reasoning, reasoningOk := u.Reasoning.Count()

	figures := CostFigures{
		InputTokens:      uint64(input),
		OutputTokens:     uint64(output),
		CacheReadTokens:  uint64(cacheRead),
		CacheWriteTokens: uint64(cacheWrite),
		ReasoningTokens:  uint64(reasoning),
	}
	presence := costPresence{
		input:      inputOk,
		output:     outputOk,
		cacheRead:  cacheReadOk,
		cacheWrite: cacheWriteOk,
		reasoning:  reasoningOk,
	}
	return figures, presence
}

// newCostTurnFromUsage constructs a per-turn cost event carrying
// presence taken from u (R-CST-001, R-CST-003) — the AG-14
// typedCancellationFailure sibling precedent: a package-private door
// that bypasses the public, all-reported NewCostTurn so a caller
// holding a real ai.Usage can construct a payload whose presence is
// not invented. run and turn identity are validated exactly as
// NewCostTurn validates them; label is validated by the shared
// CostTurn.validate.
func newCostTurnFromUsage(run RunID, turn TurnID, label CostLabel, u ai.Usage) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	figures, presence := costFromUsage(u)
	payload := CostTurn{label: label, figures: figures, presence: presence}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// newCostSessionFromTotals constructs a cumulative session-scope cost
// event carrying the given figures and presence directly (R-CST-004,
// R-CST-005) — the package-private door a costAccumulator's own
// sessionEvent method uses, so the harness's run-scoped cumulative is
// never funneled through the public, all-reported NewCostSession.
func newCostSessionFromTotals(run RunID, label CostLabel, f CostFigures, p costPresence) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	payload := CostSession{label: label, figures: f, presence: p}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run}, nil
}

// costAccumulator is the run-scoped cumulative (R-CST-004): a local in
// Harness.Run's own stack frame, never a Harness field (R-CAN-002) —
// Harness is value-form, serially reused, and carries no cross-run
// state.
type costAccumulator struct {
	figures  CostFigures
	presence costPresence
}

// add folds one observed cost_turn into the running total, per figure
// independently (R-CST-004's algebra): absence is the additive
// identity in value, and presence is OR — a reported nought survives
// (absent + reported 0 → reported 0), and a figure no turn ever
// reported stays absent on the cumulative, never a fabricated 0.
// Plain uint64 addition, no saturation: each contribution is bounded
// by int64 max (Layer 1 non-negativity), so overflow is unreachable
// in any real run.
func (a *costAccumulator) add(c CostTurn) {
	a.figures.InputTokens += c.figures.InputTokens
	a.figures.OutputTokens += c.figures.OutputTokens
	a.figures.CacheReadTokens += c.figures.CacheReadTokens
	a.figures.CacheWriteTokens += c.figures.CacheWriteTokens
	a.figures.ReasoningTokens += c.figures.ReasoningTokens

	a.presence.input = a.presence.input || c.presence.input
	a.presence.output = a.presence.output || c.presence.output
	a.presence.cacheRead = a.presence.cacheRead || c.presence.cacheRead
	a.presence.cacheWrite = a.presence.cacheWrite || c.presence.cacheWrite
	a.presence.reasoning = a.presence.reasoning || c.presence.reasoning
}

// sessionEvent constructs a cost_session event carrying a's current
// totals under label (R-CST-005, R-CST-006) — Estimate for a running
// total between turns, Final for the run-terminal figure.
func (a *costAccumulator) sessionEvent(run RunID, label CostLabel) (Event, error) {
	return newCostSessionFromTotals(run, label, a.figures, a.presence)
}
