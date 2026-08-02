// AI-22.3 — proof for delegated ordering and new contiguity assertions
// (R-STK-005, R-STK-006), from outside package agenttest, the same
// agenttest_test package this milestone's sibling proof files use.
package agenttest_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// recordingOf builds a Recording from events via a real drain — Recording
// has no exported constructor besides DrainAndRecord (R-STK-002's own
// immutability posture), so this file's fixtures route through a small
// buffered channel exactly like any real producer's output would.
func recordingOf(t *testing.T, events []ai.Event) agenttest.Recording {
	t.Helper()

	ch := make(chan ai.Event, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
}

func mustNoOrderingErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("fixture construction returned %v, want no failure", err)
	}
}

// AI-22.3 item 1 (R-STK-005, S-STK-013) — a recording carrying a terminal
// event followed by a text delta fails RequireValidStream, and the failure
// carries ai.CheckStream's own verdict for that violation, unmodified.
func TestRequireValidStream_TerminalFollowedByDelta_CarriesCheckStreamsOwnVerdictUnmodified(t *testing.T) {
	t.Parallel()

	prior := mustPriorOutputEvent(t)
	failure, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout})
	mustNoOrderingErr(t, err)
	terminal, err := ai.ErrorEvent(failure)
	mustNoOrderingErr(t, err)
	late, err := ai.NewTextDelta(1, "too late")
	mustNoOrderingErr(t, err)

	events := stampN([]ai.Event{prior, terminal, late})
	wantReport := ai.CheckStream(events)
	if wantReport.Violation() == nil {
		t.Fatal("ai.CheckStream(events) reported no violation for this fixture (terminal followed by a delta) — fixture is broken, want ErrMisplaced")
	}

	rec := recordingOf(t, events)
	fake := &fakeTB{}
	agenttest.RequireValidStream(fake, rec)

	if !fake.failed {
		t.Fatal("RequireValidStream did not fail for a terminal event followed by a text delta, want ai.CheckStream's violation surfaced")
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, wantReport.Violation().Error()) {
		t.Errorf("failure message %q does not carry ai.CheckStream's own verdict %q, unmodified", msg, wantReport.Violation().Error())
	}
}

// AI-22.3 item 1 (R-STK-005, S-STK-014) — a well-formed recording passes
// RequireValidStream and reports no violation.
func TestRequireValidStream_WellFormedRecording_Passes(t *testing.T) {
	t.Parallel()

	start, err := ai.NewTextBlockStart(1)
	mustNoOrderingErr(t, err)
	delta, err := ai.NewTextDelta(1, "hi")
	mustNoOrderingErr(t, err)
	end, err := ai.NewTextBlockEnd(1)
	mustNoOrderingErr(t, err)

	rec := recordingOf(t, stampN([]ai.Event{start, delta, end}))
	fake := &fakeTB{}
	agenttest.RequireValidStream(fake, rec)

	if fake.failed {
		t.Fatalf("RequireValidStream failed for a well-formed recording, want success: %v", fake.fatal)
	}
}

// stampSequences returns len(seqs) events, each independently stamped to
// carry exactly the requested sequence — including repeats, decreases and
// gaps, which a single shared ai.Stamper could never produce (it only ever
// counts upward from 1). Every slot uses its own fresh ai.Stamper, called
// seq times, keeping only the last result. The underlying event kind is
// irrelevant to CheckContiguity, which reads only Sequence().
func stampSequences(t *testing.T, seqs []int) []ai.Event {
	t.Helper()

	template, err := ai.NewTextBlockStart(1)
	mustNoOrderingErr(t, err)

	out := make([]ai.Event, len(seqs))
	for i, want := range seqs {
		if want < 1 {
			t.Fatalf("stampSequences: requested sequence %d, want >= 1", want)
		}
		var s ai.Stamper
		var ev ai.Event
		for range want {
			ev = s.Stamp(template)
		}
		out[i] = ev
	}
	return out
}

// AI-22.3 item 2 (R-STK-006, S-STK-016) — a recording sequenced 1, 2, 3, 4
// passes the contiguity assertion.
func TestCheckContiguity_Sequenced1234_Passes(t *testing.T) {
	t.Parallel()

	events := stampSequences(t, []int{1, 2, 3, 4})
	if err := agenttest.CheckContiguity(events); err != nil {
		t.Errorf("CheckContiguity(1,2,3,4) = %v, want nil", err)
	}
}

// AI-22.3 item 2 (R-STK-006, S-STK-017) — a recording sequenced 1, 2, 4
// fails, naming the missing sequence 3 and the two neighbouring events
// (indices 1 and 2, carrying 2 and 4).
func TestCheckContiguity_Sequenced124_NamesMissingSeq3AndNeighbours(t *testing.T) {
	t.Parallel()

	events := stampSequences(t, []int{1, 2, 4})
	err := agenttest.CheckContiguity(events)
	if err == nil {
		t.Fatal("CheckContiguity(1,2,4) = nil, want a gap violation naming missing sequence 3")
	}
	msg := err.Error()
	if !strings.Contains(msg, "3") {
		t.Errorf("error %q does not name the missing sequence 3", msg)
	}
	if !strings.Contains(msg, "seq=2") || !strings.Contains(msg, "seq=4") {
		t.Errorf("error %q does not name both neighbouring sequence values (2 and 4)", msg)
	}
	if !strings.Contains(msg, "event[1]") || !strings.Contains(msg, "event[2]") {
		t.Errorf("error %q does not name both neighbouring indices (1 and 2)", msg)
	}
}

// AI-22.3 item 2 (R-STK-006, S-STK-018) — a recording whose first event
// carries sequence 2 fails naming the start-at-1 violation, not a
// mid-stream gap.
func TestCheckContiguity_FirstEventCarriesSequence2_NamesStartAt1NotMidStreamGap(t *testing.T) {
	t.Parallel()

	events := stampSequences(t, []int{2, 3})
	err := agenttest.CheckContiguity(events)
	if err == nil {
		t.Fatal("CheckContiguity([2,3]) = nil, want a start-at-1 violation")
	}
	msg := err.Error()
	if !strings.Contains(msg, "seq=1") {
		t.Errorf("error %q does not name the required starting sequence 1", msg)
	}
	if strings.Contains(msg, "missing seq") {
		t.Errorf("error %q reports a mid-stream gap, want the distinct start-at-1 violation", msg)
	}
}

// AI-22.3 item 2 (R-STK-006, S-STK-019) — a recording sequenced 1, 2, 2
// fails naming the repeated sequence and its index.
func TestCheckContiguity_Sequenced122_NamesRepeatedSequenceAndIndex(t *testing.T) {
	t.Parallel()

	events := stampSequences(t, []int{1, 2, 2})
	err := agenttest.CheckContiguity(events)
	if err == nil {
		t.Fatal("CheckContiguity(1,2,2) = nil, want a repeated-sequence violation")
	}
	msg := err.Error()
	if !strings.Contains(msg, "event[2]") {
		t.Errorf("error %q does not name the offending index (2)", msg)
	}
	if !strings.Contains(msg, "seq=2") {
		t.Errorf("error %q does not name the repeated sequence value (2)", msg)
	}
}

// AI-22.3 item 3 (NFR-STK-E, S-STK-044 partial) — CheckContiguity and
// RequireValidStream never panic on an empty or single-element recording:
// empty is valid, and a single element must start at sequence 1.
func TestCheckContiguityAndRequireValidStream_ExtremeInputs_NeverPanic(t *testing.T) {
	t.Parallel()

	t.Run("CheckContiguity: empty is valid", func(t *testing.T) {
		t.Parallel()

		if err := agenttest.CheckContiguity(nil); err != nil {
			t.Errorf("CheckContiguity(nil) = %v, want nil (empty is valid)", err)
		}
		if err := agenttest.CheckContiguity([]ai.Event{}); err != nil {
			t.Errorf("CheckContiguity([]ai.Event{}) = %v, want nil (empty is valid)", err)
		}
	})

	t.Run("CheckContiguity: single element at seq=1 is valid", func(t *testing.T) {
		t.Parallel()

		events := stampSequences(t, []int{1})
		if err := agenttest.CheckContiguity(events); err != nil {
			t.Errorf("CheckContiguity([seq=1]) = %v, want nil", err)
		}
	})

	t.Run("CheckContiguity: single element not at seq=1 fails attributably", func(t *testing.T) {
		t.Parallel()

		events := stampSequences(t, []int{5})
		err := agenttest.CheckContiguity(events)
		if err == nil {
			t.Fatal("CheckContiguity([seq=5]) = nil, want a start-at-1 violation")
		}
		if !strings.Contains(err.Error(), "event[0]") {
			t.Errorf("error %q does not name event[0]", err.Error())
		}
	})

	t.Run("RequireValidStream: empty recording is valid", func(t *testing.T) {
		t.Parallel()

		fake := &fakeTB{}
		agenttest.RequireValidStream(fake, recordingOf(t, nil))
		if fake.failed {
			t.Fatalf("RequireValidStream failed for an empty recording, want success: %v", fake.fatal)
		}
	})
}
