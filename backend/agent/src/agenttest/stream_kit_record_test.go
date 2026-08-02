// AI-22.1 — proof for the timeout-safe drain and reusable recording
// (R-STK-001, R-STK-002), from outside package agenttest, the same
// agenttest_test package AI-20/AI-21's own proof files use.
package agenttest_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// fakeTB is a minimal testing.TB double that captures Fatal/Fatalf calls
// instead of halting the goroutine or failing the real enclosing test, so
// this milestone's own tests can assert on a kit helper's failure message
// without crashing the suite that proves it.
//
// Embedding the testing.TB INTERFACE (never a concrete *testing.T) is what
// lets a type outside package testing satisfy testing.TB at all:
// testing.TB seals itself with an unexported method, and embedding the
// interface promotes that method (and every other one) onto fakeTB without
// fakeTB ever implementing it itself — only the methods this file overrides
// (Helper, Fatal, Fatalf, Setenv) behave differently from a real *testing.T;
// every other promoted method is never called by this milestone's helpers,
// so the embedded nil interface is never reached.
type fakeTB struct {
	testing.TB
	failed bool
	fatal  []string
	helped bool
}

func (f *fakeTB) Helper() { f.helped = true }

func (f *fakeTB) Fatalf(format string, args ...any) {
	f.failed = true
	f.fatal = append(f.fatal, fmt.Sprintf(format, args...))
}

func (f *fakeTB) Fatal(args ...any) {
	f.failed = true
	f.fatal = append(f.fatal, fmt.Sprint(args...))
}

func (f *fakeTB) Setenv(string, string) {}

// lastFatal returns the most recently captured Fatal/Fatalf message, or ""
// if none was captured.
func (f *fakeTB) lastFatal() string {
	if len(f.fatal) == 0 {
		return ""
	}
	return f.fatal[len(f.fatal)-1]
}

// AI-22.1 item 1 (R-STK-001, S-STK-001) — a producer that emits two events
// and never closes fails DrainAndRecord within a short deadline, naming the
// deadline and the two events already received.
func TestDrainAndRecord_NeverCloses_FailsNamingDeadlineAndEventsReceived(t *testing.T) {
	t.Parallel()

	ch := make(chan ai.Event, 2)
	ch <- mustTextStartEvent(t)
	ch <- mustPriorOutputEvent(t)
	// Deliberately never closed and never sent again: this channel "never
	// closes" for as long as this test cares (S-STK-001).

	fake := &fakeTB{}
	const shortDeadline = 30 * time.Millisecond
	rec := agenttest.DrainAndRecord(fake, ch, shortDeadline)

	if !fake.failed {
		t.Fatal("DrainAndRecord did not fail against a channel that never closes, want a deadline failure (S-STK-001)")
	}
	if len(fake.fatal) != 1 {
		t.Fatalf("DrainAndRecord called Fatal/Fatalf %d time(s), want exactly 1", len(fake.fatal))
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, shortDeadline.String()) {
		t.Errorf("failure message %q does not name the deadline %s", msg, shortDeadline)
	}
	if !strings.Contains(msg, "2 event") {
		t.Errorf("failure message %q does not name the 2 events received before the deadline", msg)
	}
	if got := rec.Len(); got != 2 {
		t.Errorf("Recording.Len() = %d, want 2 (the events received before the deadline, not silently discarded)", got)
	}
}

// AI-22.1 item 1 (R-STK-001, S-STK-002) — a fully scripted stream that
// closes normally drains to the complete recording well within a generous
// deadline.
func TestDrainAndRecord_FullyScriptedStream_ReturnsCompleteRecordingWithoutReachingDeadline(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(mustTextDeltaScript(t))
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	fake := &fakeTB{}
	rec := agenttest.DrainAndRecord(fake, ch, agenttest.DefaultDrainTimeout)

	if fake.failed {
		t.Fatalf("DrainAndRecord failed against a fully scripted, normally closing stream, want success: %v", fake.fatal)
	}
	if got := rec.Len(); got != 4 {
		t.Fatalf("rec.Len() = %d, want 4 (start, two deltas, end — the deadline was never reached)", got)
	}
}

// AI-22.1 item 1 (R-STK-001, S-STK-003) — a stream that closes bare with
// events undelivered (this test's stand-in for "after cancellation": a
// channel closed with fewer events than a full script would carry) is
// reported as a normal closure with the events actually delivered — a bare
// close is a close, not a deadline failure.
func TestDrainAndRecord_ClosesBareWithEventsUndelivered_ReportsNormalClosureNotDeadlineFailure(t *testing.T) {
	t.Parallel()

	ch := make(chan ai.Event, 1)
	ch <- mustTextStartEvent(t)
	close(ch) // bare close: one event delivered, nothing further ever arrives

	fake := &fakeTB{}
	rec := agenttest.DrainAndRecord(fake, ch, agenttest.DefaultDrainTimeout)

	if fake.failed {
		t.Fatalf("DrainAndRecord failed against a bare close, want it reported as normal closure with what was delivered: %v", fake.fatal)
	}
	if got := rec.Len(); got != 1 {
		t.Fatalf("rec.Len() = %d, want 1 (the one event delivered before the bare close)", got)
	}
}

// drainFourEventTextScript drains one fresh four-event text script (start,
// two deltas, end — this file's recurring fixture) into a Recording via the
// real *testing.T, failing the enclosing test if the drain itself fails, so
// a fixture defect is never mistaken for the behavior under test.
func drainFourEventTextScript(t *testing.T) agenttest.Recording {
	t.Helper()

	provider := agenttest.NewProvider(mustTextDeltaScript(t))
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	return agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
}

// AI-22.1 item 2 (R-STK-002, S-STK-004) — one recording of a four-event
// stream backs three different assertions run over it in turn, with no
// re-drain: the stream is read exactly once, through DrainAndRecord, and
// every assertion below reads only rec.Events()/rec.Len().
func TestRecording_OneRecordingBacksThreeAssertionsInTurnWithoutReDraining(t *testing.T) {
	t.Parallel()

	rec := drainFourEventTextScript(t)

	t.Run("assertion 1: event count", func(t *testing.T) {
		if got := rec.Len(); got != 4 {
			t.Errorf("rec.Len() = %d, want 4", got)
		}
	})
	t.Run("assertion 2: kind order matches the script", func(t *testing.T) {
		want := []ai.EventKind{
			ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextDelta, ai.EventKindTextBlockEnd,
		}
		events := rec.Events()
		for i, ev := range events {
			if ev.Kind() != want[i] {
				t.Errorf("event %d kind = %v, want %v", i, ev.Kind(), want[i])
			}
		}
	})
	t.Run("assertion 3: delta bytes match the script byte-for-byte", func(t *testing.T) {
		events := rec.Events()
		d1, ok := events[1].TextDelta()
		if !ok || d1.Delta() != "hello, " {
			t.Errorf("event 1's TextDelta() = %v (ok=%v), want delta %q", d1, ok, "hello, ")
		}
	})
}

// AI-22.1 item 2 (R-STK-002, S-STK-005) — reading a recording twice, then
// mutating the first read's slice, leaves the second read unaffected: proof
// that Events() returns a fresh copy on every call rather than sharing the
// recording's backing array with its caller.
func TestRecording_Events_MutatingFirstReadDoesNotAffectSecondRead(t *testing.T) {
	t.Parallel()

	rec := drainFourEventTextScript(t)

	first := rec.Events()
	wantKind, wantSeq := first[0].Kind(), first[0].Sequence()
	first[0] = ai.Event{} // mutate the caller's own copy

	second := rec.Events()
	if got := second[0].Kind(); got != wantKind {
		t.Errorf("second read's event 0 kind = %v, want %v (unaffected by mutating the first read — Events() must copy)", got, wantKind)
	}
	if got := second[0].Sequence(); got != wantSeq {
		t.Errorf("second read's event 0 sequence = %v, want %v (unaffected by mutating the first read — Events() must copy)", got, wantSeq)
	}
}

// AI-22.1 item 2 (R-STK-002, S-STK-006) — a recording of a scripted stream
// reproduces the script in order: no event added, reordered, coalesced or
// omitted.
func TestRecording_MatchesScriptedOrderExactly_NoAddReorderCoalesceOmit(t *testing.T) {
	t.Parallel()

	rec := drainFourEventTextScript(t)
	events := rec.Events()
	if len(events) != 4 {
		t.Fatalf("len(rec.Events()) = %d, want 4 (no event added or omitted)", len(events))
	}

	wantKinds := []ai.EventKind{
		ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextDelta, ai.EventKindTextBlockEnd,
	}
	for i, ev := range events {
		if ev.Kind() != wantKinds[i] {
			t.Errorf("event %d kind = %v, want %v (no event reordered)", i, ev.Kind(), wantKinds[i])
		}
		if wantSeq := ai.Sequence(i + 1); ev.Sequence() != wantSeq { //nolint:gosec // i+1 is always in [1,4]
			t.Errorf("event %d sequence = %v, want %v", i, ev.Sequence(), wantSeq)
		}
	}

	d1, ok := events[1].TextDelta()
	if !ok || d1.Delta() != "hello, " {
		t.Errorf("event 1's TextDelta() = %v (ok=%v), want delta %q (no fragment coalesced into its neighbour)", d1, ok, "hello, ")
	}
	d2, ok := events[2].TextDelta()
	if !ok || d2.Delta() != "world" {
		t.Errorf("event 2's TextDelta() = %v (ok=%v), want delta %q", d2, ok, "world")
	}
}

// AI-22.1 item 3 (NFR-STK-E, S-STK-044 partial) — a nil channel and a
// zero/negative timeout each fail DrainAndRecord with a specific,
// attributable message naming which extreme input was given, never a
// panic and never a silent hang.
func TestDrainAndRecord_ExtremeInputs_FailAttributablyNeverPanic(t *testing.T) {
	t.Parallel()

	t.Run("nil channel", func(t *testing.T) {
		t.Parallel()

		fake := &fakeTB{}
		agenttest.DrainAndRecord(fake, nil, 30*time.Millisecond)

		if !fake.failed {
			t.Fatal("DrainAndRecord(nil channel) did not fail, want an attributable failure")
		}
		if msg := fake.lastFatal(); !strings.Contains(msg, "nil channel") {
			t.Errorf("failure message %q does not specifically name a nil channel", msg)
		}
	})

	t.Run("zero timeout", func(t *testing.T) {
		t.Parallel()

		ch := make(chan ai.Event)
		fake := &fakeTB{}
		agenttest.DrainAndRecord(fake, ch, 0)

		if !fake.failed {
			t.Fatal("DrainAndRecord(zero timeout) did not fail, want an attributable failure")
		}
		if msg := fake.lastFatal(); !strings.Contains(msg, "non-positive timeout") {
			t.Errorf("failure message %q does not specifically name a non-positive timeout", msg)
		}
	})

	t.Run("negative timeout", func(t *testing.T) {
		t.Parallel()

		ch := make(chan ai.Event)
		fake := &fakeTB{}
		agenttest.DrainAndRecord(fake, ch, -1*time.Second)

		if !fake.failed {
			t.Fatal("DrainAndRecord(negative timeout) did not fail, want an attributable failure")
		}
		if msg := fake.lastFatal(); !strings.Contains(msg, "non-positive timeout") {
			t.Errorf("failure message %q does not specifically name a non-positive timeout", msg)
		}
	})
}
