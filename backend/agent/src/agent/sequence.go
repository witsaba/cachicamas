// AG-04.1 — the per-lane sequence (R-AEV-002, design AD-5).
//
// Layer 2's ordering is an independent, agent-level counter — never
// ai.Sequence reused, aliased or extended (S-AGV-021). It is stamped once
// per consumer lane, not once per process and not once per canonical
// stream: AG-01 § 5 decides that every attached consumer's lane is fed by
// its own forwarding activity, which privately owns that consumer's
// receive-only carrier (openspec/changes/archive/
// 2026-08-11-cachicamas-agent-event-delivery/decision.md:230, :267).
//
// # Why no mutex and no atomic
//
// A [LaneStamper] belongs to exactly one lane and is touched only by that
// lane's own forwarding activity — exactly one goroutine, by AG-01's own
// ownership rule. Two LaneStampers therefore never share memory, which is a
// stronger guarantee than a lock around one counter would give: their
// race-freedom under concurrent lanes follows from the absence of shared
// state between lanes, not from synchronizing access to shared state.
// Synchronizing the counter would let a two-writer lane pass `-race` while
// still interleaving stamps — masking a topology defect as safe
// concurrency, which is an amendment to AG-01's mechanism, never a
// correction this type is entitled to make silently. If a later milestone
// observes a lane fed by more than one writer, that finding amends AG-01 §
// 5 first, and only then this type (design AD-5).
//
// At AG-04 there is no producer: tests play the forwarding-activity role,
// one LaneStamper per hand-built stream.
package agent

// Sequence is the per-lane ordering position an event is stamped with
// (R-AEV-002). It is 1-based and contiguous within one consumer lane. The
// zero value is the documented sentinel for "never stamped".
//
// Comparing Sequence values from two different lanes is permitted and
// carries no meaning: two lanes' sequences overlap by construction (both
// start at 1).
type Sequence uint64

// LaneStamper is the per-lane state that assigns [Sequence] values
// (R-AEV-002, design AD-5).
//
// A LaneStamper belongs to exactly one consumer lane: create one per lane,
// use it only from that lane's own forwarding activity, and let it go when
// the lane ends. Its zero value is ready to stamp the first event of a lane
// at 1 — there is no constructor to call and no exported field a caller can
// reach to seed, override or reset it.
type LaneStamper struct{ last Sequence }

// Stamp returns a copy of e carrying the next [Sequence] for this
// LaneStamper's lane: 1 for the first call, then one more than the
// previous call, forever — contiguous, total, and never erroring. e's own
// prior sequence is discarded; stamping is what assigns one, not a merge.
func (s *LaneStamper) Stamp(e Event) Event {
	s.last++
	e.seq = s.last
	return e
}
