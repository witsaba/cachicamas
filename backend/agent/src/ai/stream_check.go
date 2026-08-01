// AI-14.4 — the ordering-invariant checker.
//
// This file implements R-AEE-015 through R-AEE-019: the payload-independent
// rules a recorded stream must satisfy, read entirely from what
// event_descriptor.go declares a kind carries — never from a concrete
// payload type, never by special-casing a kind by name (R-AEE-015). A later
// milestone that registers a kind with a descriptor extends every rule
// below with no change to this file's source; that is the whole reason the
// descriptor exists (design.md D1).
package ai

// StreamReport is CheckStream's verdict: the first violation found, or nil,
// and independently, whether a terminal event was seen at all.
//
// The two are independent so that "no terminal event yet" is never folded
// into an ordering violation (R-AEE-018): AI-14 registers no terminal
// production kind, and a milestone that does asserts presence through
// Terminated() in its own tests, without this file changing.
type StreamReport struct {
	err        error
	terminated bool
}

// Violation returns the first violation CheckStream found, in stream order,
// or nil when the stream satisfies every rule this file checks. It is
// AI-04's failure value — no second failure type, no new sentinel
// (R-AEE-019).
func (r StreamReport) Violation() error { return r.err }

// Terminated reports whether a terminal-kind event (EventDescriptor.Terminal
// == true) was seen anywhere in the stream.
func (r StreamReport) Terminated() bool { return r.terminated }

// blockState is one block index's position in its start/delta/end
// lifecycle, tracked only for as long as CheckStream needs it to detect an
// out-of-order event or an unterminated block.
type blockState struct {
	// startSeq is the sequence of the event that opened this block — the
	// position an eventual "unterminated block" violation names.
	startSeq Sequence
	ended    bool
}

// CheckStream runs the AI-14.4 ordering invariants against events, a finite
// ordered slice offered after the fact — never a channel, and never mutated
// (R-AEE-019). It reads only (kind, descriptor, block index, sequence) per
// event: no concrete payload type, no kind special-cased by name
// (R-AEE-015).
//
// It reports the first violation in stream order (V-FAIL-04) and whether a
// terminal event was seen, independently (R-AEE-018). It does not check
// 1…N sequence contiguity — that is AI-22.3's charter (design.md D10). A nil
// or empty slice is safe and reports no violation and no terminal event.
// Two runs over the same slice report identical verdicts.
//
// Within one event, rules are checked in this order: whether a terminal
// event has already occurred (the most encompassing complaint — everything
// after a terminal event is wrong regardless of what it is), then block
// ordering, then at-most-one cardinality. This order is not itself spec'd
// scenario-by-scenario for the case where one event trips more than one
// rule; it is documented here so the choice is data a reader can see rather
// than behavior they have to trace.
func CheckStream(events []Event) StreamReport {
	blocks := make(map[BlockIndex]*blockState)
	// blockOrder preserves the order blocks were first opened, so an
	// eventual "unterminated block" violation names the one that has been
	// open longest — the earliest such fact in stream order.
	var blockOrder []BlockIndex
	seenAtMostOnce := make(map[EventKind]bool)
	terminalSeen := false

	for _, e := range events {
		kind := e.Kind()
		entry, ok := eventRegistryEntry(kind)
		if !ok {
			// An unregistered or zero-kind event is CheckEmit's concern
			// (R-AEE-002), not this checker's: it carries no descriptor to
			// read, so none of the rules below apply to it.
			continue
		}
		descriptor := entry.descriptor
		seq := e.Sequence()

		if terminalSeen {
			if descriptor.Terminal {
				return verdict(ErrDuplicate, true, AtIndex("event", int(seq)))
			}
			return verdict(ErrMisplaced, true, AtIndex("event", int(seq)))
		}

		if descriptor.Role != BlockRoleNone {
			if violation := checkBlockOrdering(blocks, &blockOrder, descriptor.Role, seq, blockIndexOf(e)); violation != nil {
				return StreamReport{err: violation, terminated: terminalSeen}
			}
		}

		if descriptor.Cardinality == CardinalityAtMostOne {
			if seenAtMostOnce[kind] {
				return verdict(ErrDuplicate, terminalSeen, AtIndex("event", int(seq)))
			}
			seenAtMostOnce[kind] = true
		}

		if descriptor.Terminal {
			terminalSeen = true
		}
	}

	for _, idx := range blockOrder {
		if state := blocks[idx]; !state.ended {
			return verdict(ErrMalformed, terminalSeen,
				AtIndex("event", int(state.startSeq)), AtIndex("block", int(idx)))
		}
	}

	return StreamReport{terminated: terminalSeen}
}

// verdict is CheckStream's one constructor for a failing StreamReport,
// consolidating what would otherwise be five near-identical composite
// literals into the single place that maps a rule class and position to a
// report (design.md D7's verdict mapping).
func verdict(rule error, terminated bool, at ...Step) StreamReport {
	return StreamReport{err: Invalid(rule, at...), terminated: terminated}
}

// checkBlockOrdering applies R-AEE-016 to one block-scoped event: idx opens,
// continues or closes its block in blocks (tracked in first-opened order via
// blockOrder), or the event is out of order for it. It returns the
// violation, or nil.
func checkBlockOrdering(blocks map[BlockIndex]*blockState, blockOrder *[]BlockIndex, role BlockRole, seq Sequence, idx BlockIndex) *Violation {
	state, started := blocks[idx]

	switch role {
	case BlockRoleStart:
		if started {
			// A second start for this index (design.md D7).
			return Invalid(ErrDuplicate, AtIndex("event", int(seq)), AtIndex("block", int(idx)))
		}
		blocks[idx] = &blockState{startSeq: seq}
		*blockOrder = append(*blockOrder, idx)
		return nil

	case BlockRoleDelta, BlockRoleEnd:
		if !started || state.ended {
			// Delta/end with no preceding start, or after the block's own
			// end — both are "out of order" for this index (design.md D7).
			return Invalid(ErrMisplaced, AtIndex("event", int(seq)), AtIndex("block", int(idx)))
		}
		if role == BlockRoleEnd {
			state.ended = true
		}
		return nil

	default: // BlockRoleNone never reaches here; checkBlockOrdering is only
		// called when descriptor.Role != BlockRoleNone.
		return nil
	}
}

// blockIndexOf reports e's block index through the unexported blockPayload
// interface — never a concrete payload type (R-AEE-015). A block-scoped
// kind whose payload does not implement it (a registration defect
// event_registry_test.go's exhaustiveness assertion catches separately,
// S-AEE-045) is treated as index 0, which checkBlockOrdering already
// receives as "no matching start" rather than panicking (NFR-AEE-B).
func blockIndexOf(e Event) BlockIndex {
	bp, ok := e.payload.(blockPayload)
	if !ok {
		return 0
	}
	return bp.blockIndex()
}
