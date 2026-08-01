// AI-14.2 — the per-stream sequence.
//
// This is doc 0001's defect **C3**, fixed structurally rather than shrunk.
// The retired design kept the sequence in a process-global atomic counter:
// only the first stream in a process began at 1, and a shipped test recorded
// that gap as expected. The fix here is not a smaller counter; it is putting
// the counter where the stream is (R-AEE-008, sequence_guard_test.go).
//
// # Why no atomic and no mutex
//
// AI-02 § 4 guarantees exactly one producer goroutine per stream. A [Stamper]
// is created once per stream and touched only by that one goroutine, so its
// race-freedom under concurrent streams follows from the absence of shared
// state between streams, not from synchronizing access to shared state — two
// [Stamper] values never touch the same memory, which is a stronger
// guarantee than a lock around one counter would give, and is why
// TestStamper_SequenceState_IsPerStreamNotProcess passes `-race` with plain,
// unsynchronized field access.
//
// # Cross-stream comparison
//
// Two [Sequence] values from two different streams compare with == like any
// other integer, and the comparison is permitted — the type system cannot
// forbid it without forbidding Sequence from being comparable at all, which
// would cost every legitimate same-stream comparison too. It carries no
// meaning: two streams' sequences overlap by construction (both start at 1),
// and a consumer that orders a merged multi-stream log by [Sequence] is
// contradicting this written rule, not an unwritten assumption (R-AEE-009).
package ai

// Sequence is the per-stream position AI-14.2 assigns to an event (V-STR-13).
//
// It is 1-based and contiguous within one stream (R-AEE-007). The zero value
// is the documented sentinel for "never stamped": [CheckEmit] rejects it at
// the producer's emission boundary, the same boundary [Stamper] serves
// (R-AEE-010).
//
// Comparing sequences from two different streams is permitted and carries no
// meaning — see the file comment (R-AEE-009).
type Sequence uint64

// Stamper is the per-stream state that assigns [Sequence] values (V-STR-13).
//
// A Stamper belongs to exactly one stream: create one per stream, use it from
// that stream's single producer goroutine (AI-02 § 4), and let it go when the
// stream ends. Its zero value is ready to stamp the first event of a stream
// at 1 — there is no constructor to call and no field a caller can reach to
// seed, override or reset it (R-AEE-007, R-AEE-008).
type Stamper struct{ last Sequence }

// Stamp returns a copy of e carrying the next [Sequence] for this Stamper's
// stream: 1 for the first call, then one more than the previous call, forever
// — contiguous, total, and never erroring (R-AEE-007). e's own prior sequence
// is discarded; stamping is what assigns one, not a merge.
func (s *Stamper) Stamp(e Event) Event {
	s.last++
	e.seq = s.last
	return e
}
