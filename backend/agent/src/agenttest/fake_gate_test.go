// AI-21.4 — delays and the blocked stream: a synchronization point a
// scripted Hold step blocks the producer on, and deterministic buffer
// saturation.
//
// fake_gate_test.go proves R-AFP-010/011 from outside package agenttest.
// Every wait below is on Gate.Reached() or a bounded test deadline — never
// a sleep used as the mechanism ordering events (S-AFP-024, NFR-AFP-C).
package agenttest_test

import (
	"context"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-21.4 item 1 (R-AFP-010, S-AFP-022/023) — a script that emits one event
// and then holds blocks a second receive until the test releases the hold;
// releasing then drains the remaining scripted events, in order, and the
// stream completes and closes.
func TestProvider_HeldStream_BlocksUntilReleasedThenDrainsRemainingEventsInOrder(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(mustTextStartEvent(t)),
		agenttest.Hold(gate),
		agenttest.Emit(end),
	}}
	provider := agenttest.NewProvider(script)

	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	first := mustReceive(t, ch)
	if first.Kind() != ai.EventKindTextBlockStart {
		t.Fatalf("first event kind = %v, want %v", first.Kind(), ai.EventKindTextBlockStart)
	}

	// gate.Reached() closing is the whole proof the stream is open and
	// held: the producer runs sequentially in its own goroutine, so it
	// cannot have sent the scripted end yet — no sleep required to know
	// that (R-AFP-010).
	select {
	case <-gate.Reached():
	case <-time.After(boundedFakeTimeout):
		t.Fatal("gate.Reached() did not close within the bounded timeout — the producer never arrived at its Hold")
	}

	gate.Release()

	second := mustReceive(t, ch)
	if second.Kind() != ai.EventKindTextBlockEnd {
		t.Errorf("second event kind = %v, want %v", second.Kind(), ai.EventKindTextBlockEnd)
	}
	if _, ok := <-ch; ok {
		t.Error("received a third event, want the stream closed after its two scripted events")
	}
}

// AI-21.4 item 1's third scenario (S-AFP-024) is a repository-wide property
// of every test this package adds, checked in Phase 9 (NFR-AFP-C,
// S-AFP-054) rather than by one dedicated test here: it is a fact about
// which mechanism these files use, not a runtime behaviour of the fake.

// AI-21.4 item 2 (R-AFP-011, S-AFP-025/026) — a script with more events
// than the stream's buffer admits saturates deterministically when the
// consumer reads nothing (design.md's saturation recipe: n emits, Hold,
// then a late emit, with Buffer: n); a slow-but-resumed consumer that never
// cancels still receives every scripted event afterwards, in order, none
// dropped.
func TestProvider_UnreadConsumer_DeterministicallySaturatesBuffer_ThenResumedConsumerDropsNothing(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	e1 := mustTextStartEvent(t)
	e2, err := ai.NewTextDelta(1, "a")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	late, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	script := agenttest.Script{
		Buffer: 2,
		Steps: []agenttest.Step{
			agenttest.Emit(e1),
			agenttest.Emit(e2),
			agenttest.Hold(gate),
			agenttest.Emit(late),
		},
	}
	provider := agenttest.NewProvider(script)

	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	if got := cap(ch); got != 2 {
		t.Fatalf("cap(ch) = %d, want 2 (the scripted buffer)", got)
	}

	// The consumer reads nothing at all above: gate.Reached() closing
	// proves the producer placed exactly 2 events in the 2-capacity
	// buffer and is now blocked at its Hold — saturated deterministically,
	// reached without any wall-clock assumption (S-AFP-025).
	select {
	case <-gate.Reached():
	case <-time.After(boundedFakeTimeout):
		t.Fatal("gate.Reached() did not close within the bounded timeout")
	}

	// S-AFP-026 — a slow-but-resumed, non-cancelling consumer still
	// receives every event, none dropped: drain the two buffered events
	// (catching up), release, then confirm the held event still arrives.
	first := mustReceive(t, ch)
	if first.Kind() != ai.EventKindTextBlockStart {
		t.Fatalf("first drained event kind = %v, want %v", first.Kind(), ai.EventKindTextBlockStart)
	}
	second := mustReceive(t, ch)
	if second.Kind() != ai.EventKindTextDelta {
		t.Fatalf("second drained event kind = %v, want %v", second.Kind(), ai.EventKindTextDelta)
	}

	gate.Release()

	rest := drainFake(t, ch)
	if len(rest) != 1 {
		t.Fatalf("remaining event(s) after release = %d, want 1 (backpressure waited, it did not drop)", len(rest))
	}
	if rest[0].Kind() != ai.EventKindTextBlockEnd {
		t.Errorf("remaining event kind = %v, want %v", rest[0].Kind(), ai.EventKindTextBlockEnd)
	}
}
