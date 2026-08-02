// AI-22.3 — ordering and gap assertions: kind/block/terminal ordering stays
// exactly where AI-14.4 already checked it; only sequence contiguity is new
// (design.md D3, honoring AI-21's own design D10).

package agenttest

import (
	"fmt"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// RequireValidStream fails tb the moment rec's events break either ordering
// invariant this kit asserts: [ai.CheckStream]'s own rules first (AI-14.4:
// kind ordering, block start/delta/end ordering, terminal exclusivity and
// placement — R-STK-005), then sequence contiguity (R-STK-006). Ordering is
// delegated to ai.CheckStream, never reimplemented: a violation it reports
// surfaces through tb's failure carrying its own verdict, unmodified, so a
// future descriptor change is absorbed in exactly one place.
func RequireValidStream(tb testing.TB, rec Recording) {
	tb.Helper()

	events := rec.Events()
	if report := ai.CheckStream(events); report.Violation() != nil {
		tb.Fatalf("agenttest: RequireValidStream: %v (R-STK-005)", report.Violation())
		return
	}
	if err := CheckContiguity(events); err != nil {
		tb.Fatalf("agenttest: RequireValidStream: %v (R-STK-006)", err)
		return
	}
}

// CheckContiguity asserts that events' sequence numbers, read in encounter
// order, start at 1 and increase by exactly one with no gap and no repeat
// (R-STK-006). This check is new — [ai.CheckStream] deliberately does not
// perform it (AI-14.4's own doc comment, honoring design.md D10). A nil or
// empty slice is safe and reports no violation, matching
// [ai.CheckStream]'s own totality posture for the same input.
//
// WHEN a gap exists, the error names the missing sequence number and the
// two events it falls between. WHEN a sequence repeats or decreases, the
// error names the offending index and both sequence values. WHEN the first
// event does not carry sequence 1, the error names that distinct violation
// rather than reporting a mid-stream gap.
func CheckContiguity(events []ai.Event) error {
	for i, ev := range events {
		seq := ev.Sequence()
		if i == 0 {
			if seq != 1 {
				return contiguityErrorf("event[0] carries seq=%d, want the stream to start at seq=1", seq)
			}
			continue
		}

		prevEvent := events[i-1]
		prev := prevEvent.Sequence()
		switch {
		case seq == prev:
			return contiguityErrorf("event[%d] repeats seq=%d, same as event[%d]", i, seq, i-1)
		case seq < prev:
			return contiguityErrorf("event[%d] carries seq=%d, less than event[%d]'s seq=%d", i, seq, i-1, prev)
		case seq != prev+1:
			return contiguityErrorf("missing seq=%d between event[%d] (%s seq=%d) and event[%d] (%s seq=%d)",
				prev+1, i-1, prevEvent.Kind(), prev, i, ev.Kind(), seq)
		}
	}
	return nil
}

// contiguityErrorf is CheckContiguity's one error constructor: the
// start-at-1, repeat/decrease and gap branches all funnel their message
// through it rather than each formatting their own near-duplicate prefix
// and requirement citation.
func contiguityErrorf(format string, args ...any) error {
	return fmt.Errorf("agenttest: CheckContiguity: "+format+" (R-STK-006)", args...)
}
