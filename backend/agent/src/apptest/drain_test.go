// AG-23 — strict-TDD tests for DrainAndCheck (R-L3H-003, S-L3H-017).
package apptest_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/apptest"
)

// Given the drain helper's source exercised at runtime: it blocks until
// the sink closes, preserves event order, and returns the exported
// validator's own report unmodified. Synchronization is by channel send
// and channel close only — no sleep, timeout, deadline or poll (R-L3H-004):
// each send on the unbuffered sink is itself the synchronization point
// that proves DrainAndCheck has not returned early.
func TestDrainAndCheck_BlocksUntilClose_PreservesOrder(t *testing.T) {
	t.Parallel()

	sink := make(chan *agent.Event) // unbuffered: every send synchronizes with DrainAndCheck's own receive.
	done := make(chan struct{})
	var events []agent.Event
	var report agent.StreamReport
	go func() {
		defer close(done)
		events, report = apptest.DrainAndCheck(sink)
	}()

	stamper := &agent.LaneStamper{}

	runStart, err := agent.NewRunStart("run-drain-001")
	if err != nil {
		t.Fatalf("agent.NewRunStart error = %v, want nil", err)
	}
	stampedStart := stamper.Stamp(runStart)
	sink <- &stampedStart

	// The send above only returns once DrainAndCheck's range has
	// received it — DrainAndCheck is therefore now blocked on its NEXT
	// receive, and cannot have returned already.
	select {
	case <-done:
		t.Fatal("DrainAndCheck returned before the sink was closed (after 1 event) — want it still blocked on the next receive")
	default:
	}

	runEnd, err := agent.NewRunEnd("run-drain-001", agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd error = %v, want nil", err)
	}
	stampedEnd := stamper.Stamp(runEnd)
	sink <- &stampedEnd

	select {
	case <-done:
		t.Fatal("DrainAndCheck returned before the sink was closed (after 2 events) — want it still blocked on the next receive")
	default:
	}

	close(sink)
	<-done // Go's channel-range semantics guarantee this returns; no timeout is needed or used.

	if len(events) != 2 {
		t.Fatalf("DrainAndCheck returned %d event(s), want 2", len(events))
	}
	if events[0].Kind() != agent.EventKindRunStart {
		t.Errorf("events[0].Kind() = %v, want EventKindRunStart (order preserved)", events[0].Kind())
	}
	if events[1].Kind() != agent.EventKindRunEnd {
		t.Errorf("events[1].Kind() = %v, want EventKindRunEnd (order preserved)", events[1].Kind())
	}
	if v := report.Violation(); v != nil {
		t.Errorf("report.Violation() = %v, want nil for a valid run bracket", v)
	}
}

// Given a stream DrainAndCheck itself would never construct incorrectly
// but the exported validator DOES reject — a run-close event with no
// matching run-open — when DrainAndCheck runs, then its returned report
// carries that violation UNMODIFIED, proving delegation is real rather
// than a wrapper that always reports clean.
func TestDrainAndCheck_DelegatesRealViolationsWholesale(t *testing.T) {
	t.Parallel()

	sink := make(chan *agent.Event, 1)
	stamper := &agent.LaneStamper{}
	runEnd, err := agent.NewRunEnd("run-drain-002", agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd error = %v, want nil", err)
	}
	stamped := stamper.Stamp(runEnd)
	sink <- &stamped
	close(sink)

	events, report := apptest.DrainAndCheck(sink)
	if len(events) != 1 {
		t.Fatalf("DrainAndCheck returned %d event(s), want 1", len(events))
	}
	if report.Violation() == nil {
		t.Fatal("report.Violation() = nil for a run_end with no preceding run_start — want the exported validator's own rejection, unmodified")
	}
}
