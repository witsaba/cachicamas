// AI-22.1 — timeout-safe drain and record: the select loop
// ai/provider_test.go's requireClosedWithin and this package's own
// fake_text_test.go's drainFake already carry twice, under two different
// names, promoted verbatim into one importable helper (design.md D1) so
// AI-23 does not need to write it a third time. requireClosedWithin and
// drainFake themselves stay exactly where they are (NFR-STK-C) — this file
// is additive tooling, not a migration of either one.

package agenttest

import (
	"slices"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// DefaultDrainTimeout is a deadline generous enough not to flake a loaded CI
// runner under -race, and short enough that a genuinely stuck producer fails
// a suite using it in seconds, not minutes — ai/provider_test.go's
// boundedTimeout and fake_text_test.go's boundedFakeTimeout, restated as one
// exported constant (design.md D1).
const DefaultDrainTimeout = 2 * time.Second

// Recording is the finite, ordered record one drain produced (R-STK-002):
// immutable once produced, and reusable across every kit assertion without
// re-draining the stream that built it.
type Recording struct {
	events []ai.Event
}

// Events returns a fresh copy of the recorded events, in receive order
// (R-STK-002). Every call allocates a new slice, so a caller that mutates
// what it received cannot alter the recording, and two reads never observe
// each other's edits.
func (r Recording) Events() []ai.Event {
	return slices.Clone(r.events)
}

// Len reports the number of events the recording holds (R-STK-002).
func (r Recording) Len() int {
	return len(r.events)
}

// DrainAndRecord receives from ch until it closes or timeout elapses,
// whichever happens first, and returns everything received as a
// [Recording] (R-STK-001, R-STK-002).
//
// On deadline expiry it fails tb with a message naming the deadline and how
// many events arrived first — DrainAndRecord never hangs, never blocks past
// timeout, and never returns a partial recording as though the stream had
// closed on its own: a bare close (the channel closing with fewer events
// than a caller might have expected) is reported as an ordinary, complete
// recording, never as a deadline failure.
//
// A nil ch or a non-positive timeout each fail tb immediately with a
// message naming which extreme input was given, rather than falling through
// to the generic deadline message or panicking (NFR-STK-E): a nil channel
// blocks forever on every receive, and a non-positive timeout would
// otherwise report a technically-true but unhelpful "did not close within
// 0s" for a caller who almost certainly passed the wrong value by mistake.
func DrainAndRecord(tb testing.TB, ch <-chan ai.Event, timeout time.Duration) Recording {
	tb.Helper()

	if ch == nil {
		tb.Fatalf("agenttest: DrainAndRecord given a nil channel — want a live stream to drain (R-STK-001)")
		return Recording{}
	}
	if timeout <= 0 {
		tb.Fatalf("agenttest: DrainAndRecord given a non-positive timeout %s — want a positive deadline (R-STK-001)", timeout)
		return Recording{}
	}

	var got []ai.Event
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return Recording{events: got}
			}
			got = append(got, ev)
		case <-deadline:
			tb.Fatalf("agenttest: DrainAndRecord did not close within %s (%d event(s) received so far) — want a bounded close (R-STK-001)", timeout, len(got))
			return Recording{events: got}
		}
	}
}
