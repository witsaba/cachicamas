// AG-04.1 — the stream-contract validator's contiguity core (R-AEV-002,
// design AD-1 divergence 1).
//
// Layer 1 excludes 1…N contiguity from `ai.CheckStream` and gives it to
// `agenttest.CheckContiguity` (`backend/agent/src/ai/stream_check.go:52-54`).
// AG-04.1's own scenario names the validator itself as what proves
// "independent, contiguous, 1-based" (0003:445-448), and design AD-4 rules
// `agenttest` out of AG-04 entirely — so `agent.CheckStream` checks
// per-lane contiguity itself, unlike its Layer 1 namesake.
//
// This file carries only the contiguity rule (Data Flow rule 1: `seq !=
// seq_expect` violation). The two-level run/turn scope engine (rules 2-6)
// lands at AG-04.2, alongside the run and turn lifecycle families it
// brackets — there is nothing to bracket yet at AG-04.1.
package agent

import "github.com/cachicamas/backend/agent/src/ai"

// StreamReport is CheckStream's verdict: the first violation found, in
// stream order, or nil when the stream satisfies every rule this file
// checks — AI-04's failure value, no second failure type, no new sentinel.
type StreamReport struct{ err error }

// Violation returns the first violation [CheckStream] found, in stream
// order, or nil when the stream is valid.
func (r StreamReport) Violation() error { return r.err }

// CheckStream runs AG-04's ordering invariants against events, a finite
// ordered slice offered after the fact — never a channel, and never
// mutated (R-AEV-006). It is production-exported and callable from another
// package with no test-only build tag, because it is reused wholesale by
// the Layer 3 readiness contract's kit at AG-23 (VL2-SEAM-14).
//
// At AG-04.1 it checks exactly one rule: every event's [Sequence] is
// contiguous and 1-based within events. The run/turn bracket rules join it
// at AG-04.2.
func CheckStream(events []Event) StreamReport {
	var seqExpect Sequence = 1
	for _, e := range events {
		if e.Sequence() != seqExpect {
			return StreamReport{err: ai.Invalid(ai.ErrOutOfRange, ai.AtIndex("event", int(e.Sequence())))}
		}
		seqExpect++
	}
	return StreamReport{}
}
