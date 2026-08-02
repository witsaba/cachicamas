// AI-21.1 — the walking skeleton: a scripted text response.
//
// fake_text_test.go proves AI-21.1's four requirements from outside package
// agenttest, in the same agenttest_test package the AI-20 proof already
// uses: R-AFP-001 (the fake is importable and needs no network), R-AFP-002
// (a scripted text response drains exactly as scripted), R-AFP-003
// (concurrent streams sequence independently) and R-AFP-004 (the fake
// reproduces AI-20's stream physics, non-negotiably — it does not
// approximate them).
//
// This file also carries the small shared helpers (drainFake,
// boundedFakeTimeout, textEventSignature) the sibling fake_*_test.go files
// in this same package reuse, so they are written once.
package agenttest_test

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// fakeCompileProof fails to compile the moment agenttest.Provider stops
// satisfying ai.ModelProvider, or names anything unexported from ai
// (S-AFP-001's compile half).
var _ ai.ModelProvider = (*agenttest.Provider)(nil)

// boundedFakeTimeout is generous enough to never flake on a loaded CI
// runner under -race, and short enough that a genuinely stuck producer
// fails this suite in seconds — ai/provider_test.go's boundedTimeout,
// restated for this package.
const boundedFakeTimeout = 2 * time.Second

// drainFake drains ch until it closes or boundedFakeTimeout passes,
// returning every event received in order. It fails the test — rather than
// hanging it — when ch has not closed by the deadline.
func drainFake(t *testing.T, ch <-chan ai.Event) []ai.Event {
	t.Helper()

	var got []ai.Event
	deadline := time.After(boundedFakeTimeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("channel did not close within %s (%d event(s) received so far)", boundedFakeTimeout, len(got))
			return nil
		}
	}
}

// settleAfterFakeCancel gives a producer's goroutine a window to observe an
// already-cancelled context with no receiver of this file's own in flight —
// ai/provider_test.go's settleAfterCancel, restated for this package
// (S-AFP-054 permits a documented settling step; it is not the mechanism
// that orders scripted events, only what keeps this file's own
// confirmation read from becoming a second ready select case at exactly
// the moment the producer evaluates one).
func settleAfterFakeCancel() { time.Sleep(50 * time.Millisecond) }

// textEventSignature renders an event's kind and, when the kind carries
// text bytes, those bytes byte-for-byte — enough to prove byte-identical
// equality between two runs (S-AFP-002) and byte-exact reconstruction
// (S-AFP-005), without a general-purpose ai.Event equality this package
// does not export.
func textEventSignature(ev ai.Event) string {
	switch ev.Kind() {
	case ai.EventKindTextBlockStart:
		s, _ := ev.TextBlockStart()
		return fmt.Sprintf("text_block_start(block=%d)", s.Block())
	case ai.EventKindTextDelta:
		d, _ := ev.TextDelta()
		return fmt.Sprintf("text_delta(block=%d,%q)", d.Block(), d.Delta())
	case ai.EventKindTextBlockEnd:
		e, _ := ev.TextBlockEnd()
		return fmt.Sprintf("text_block_end(block=%d)", e.Block())
	default:
		return ev.String()
	}
}

// mustTextDeltaScript builds this file's small, deterministic script: a
// text block start, two deltas and its end — "a fully scripted stream"
// small enough to read at a glance, big enough to prove ordering
// (S-AFP-004/005).
func mustTextDeltaScript(t *testing.T) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta1, err := ai.NewTextDelta(1, "hello, ")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	delta2, err := ai.NewTextDelta(1, "world")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta1),
		agenttest.Emit(delta2),
		agenttest.Emit(end),
	}}
}

// AI-21.1 item 1 (R-AFP-001, S-AFP-001…003) — the fake compiles and
// satisfies ai.ModelProvider from outside package agenttest, needs no
// network to run a fully scripted stream to close, and repeating that run
// yields byte-identical events.
func TestProvider_ConstructedExternally_SatisfiesModelProviderAndRunsWithNoNetwork(t *testing.T) {
	t.Parallel()

	var provider ai.ModelProvider = agenttest.NewProvider(mustTextDeltaScript(t))

	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	first := drainFake(t, ch)
	if len(first) != 4 {
		t.Fatalf("drained %d event(s), want 4 (start, two deltas, end)", len(first))
	}

	// S-AFP-002 — repeating the run yields byte-identical events: a fresh
	// fake driven with the identical script produces the identical drained
	// sequence. No socket, no file handle and no wall-clock dependence is
	// involved anywhere above — every step is an in-memory ai.Event value
	// and an in-memory channel; nothing here imports "net", "os" or reads a
	// clock.
	secondProvider := agenttest.NewProvider(mustTextDeltaScript(t))
	secondCh, err := secondProvider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("second run's Stream returned %v, want no failure", err)
	}
	second := drainFake(t, secondCh)

	if len(second) != len(first) {
		t.Fatalf("second run drained %d event(s), want %d (byte-identical repeat)", len(second), len(first))
	}
	for i := range first {
		if got, want := textEventSignature(second[i]), textEventSignature(first[i]); got != want {
			t.Errorf("event %d = %s, want %s (byte-identical repeat run)", i, got, want)
		}
	}
}

// AI-21.1 item 2 (R-AFP-002, S-AFP-004/005) — a scripted text response
// drains exactly as scripted: four events, in the scripted order, sequenced
// 1, 2, 3, 4 with no gap and no repeat, and each delta's bytes equal the
// scripted fragment byte-for-byte, with no fragment merged into its
// neighbour.
func TestProvider_ScriptedTextResponse_DrainsExactlyAsScriptedAndSequencedOneToN(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(mustTextDeltaScript(t))
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	got := drainFake(t, ch)
	if len(got) != 4 {
		t.Fatalf("drained %d event(s), want 4 (start, two deltas, end)", len(got))
	}

	wantKinds := []ai.EventKind{
		ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextDelta, ai.EventKindTextBlockEnd,
	}
	for i, ev := range got {
		if ev.Kind() != wantKinds[i] {
			t.Errorf("event %d kind = %v, want %v (no event added, reordered, coalesced or omitted)", i, ev.Kind(), wantKinds[i])
		}
		if wantSeq := ai.Sequence(i + 1); ev.Sequence() != wantSeq { //nolint:gosec // i+1 is always in [1,4]
			t.Errorf("event %d sequence = %v, want %v (1…N, no gap, no repeat)", i, ev.Sequence(), wantSeq)
		}
	}

	delta1, ok := got[1].TextDelta()
	if !ok {
		t.Fatalf("event 1 carries no TextDelta payload")
	}
	if delta1.Delta() != "hello, " {
		t.Errorf("first delta = %q, want %q (byte-for-byte)", delta1.Delta(), "hello, ")
	}
	delta2, ok := got[2].TextDelta()
	if !ok {
		t.Fatalf("event 2 carries no TextDelta payload")
	}
	if delta2.Delta() != "world" {
		t.Errorf("second delta = %q, want %q (not merged with its neighbour)", delta2.Delta(), "world")
	}
}

// AI-21.1 item 2's third scenario (R-AFP-002, S-AFP-006) — a script whose
// event would violate the envelope's emission rules — here, an ai.Event
// that was never constructed — fails loudly at the fake (a panic naming
// the defect) rather than being silently delivered to the consumer as an
// invalid event.
func TestEmit_EventNeverConstructed_FailsLoudlyAtTheFakeRatherThanDeliveringAnInvalidEvent(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Emit(ai.Event{}) did not panic, want a loud failure at the fake (S-AFP-006)")
		}
		msg, ok := r.(string)
		if !ok || msg == "" {
			t.Fatalf("Emit(ai.Event{}) panicked with %#v, want a non-empty message naming the defect", r)
		}
	}()

	agenttest.Emit(ai.Event{})
	t.Fatal("unreachable: Emit should have panicked before this point")
}

// AI-21.1 item 3 (R-AFP-003, S-AFP-007/008) — two fakes streamed
// concurrently sequence independently under -race, and one fake streamed
// twice restarts its sequence at 1 rather than continuing the previous
// stream's count.
func TestProvider_ConcurrentAndRepeatedStreams_SequenceIndependently(t *testing.T) {
	t.Parallel()

	t.Run("two fakes streamed concurrently sequence independently under -race", func(t *testing.T) {
		t.Parallel()

		providerA := agenttest.NewProvider(mustTextDeltaScript(t))
		providerB := agenttest.NewProvider(mustTextDeltaScript(t))

		chA, err := providerA.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("providerA.Stream returned %v, want no failure", err)
		}
		chB, err := providerB.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("providerB.Stream returned %v, want no failure", err)
		}

		var gotA, gotB []ai.Event
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); gotA = drainFake(t, chA) }()
		go func() { defer wg.Done(); gotB = drainFake(t, chB) }()
		wg.Wait()

		if len(gotA) != 4 || len(gotB) != 4 {
			t.Fatalf("drained %d event(s) from A and %d from B, want 4 and 4", len(gotA), len(gotB))
		}
		for i, ev := range gotA {
			if want := ai.Sequence(i + 1); ev.Sequence() != want { //nolint:gosec // i+1 is always in [1,4]
				t.Errorf("providerA event %d sequence = %v, want %v (per-stream, independent of providerB)", i, ev.Sequence(), want)
			}
		}
		for i, ev := range gotB {
			if want := ai.Sequence(i + 1); ev.Sequence() != want { //nolint:gosec // i+1 is always in [1,4]
				t.Errorf("providerB event %d sequence = %v, want %v (per-stream, independent of providerA)", i, ev.Sequence(), want)
			}
		}
	})

	t.Run("one fake streamed twice restarts its sequence at 1", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(mustTextDeltaScript(t), mustTextDeltaScript(t))

		firstCh, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("first Stream returned %v, want no failure", err)
		}
		first := drainFake(t, firstCh)

		secondCh, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("second Stream returned %v, want no failure", err)
		}
		second := drainFake(t, secondCh)

		if len(first) == 0 || first[0].Sequence() != 1 {
			t.Fatalf("first stream's first event sequence = %v, want 1", first[0].Sequence())
		}
		if len(second) == 0 || second[0].Sequence() != 1 {
			t.Fatalf("second stream's first event sequence = %v, want 1 (restarts, does not continue the first stream's count)", second[0].Sequence())
		}
	})
}

// mustTextStartEvent builds a single, minimal text-block-start event this
// file's physics tests reuse where only "an event" matters, not its shape.
func mustTextStartEvent(t *testing.T) ai.Event {
	t.Helper()

	ev, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	return ev
}

// AI-21.1 item 4 (R-AFP-004, S-AFP-009…011) — the fake's mid-stream physics
// match AI-20's non-negotiably: exactly one producer goroutine and one
// closing site reached on every exit path (normal completion, the
// terminal error event, and cancellation), and a producer with no reader
// left exits on cancellation rather than blocking forever.
func TestProvider_MidStreamPhysics_OneClosingSiteAcrossCompletionErrorAndCancellation(t *testing.T) {
	t.Parallel()

	t.Run("completes normally: every scripted event arrives, then the channel closes exactly once", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(mustTextDeltaScript(t))
		ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		got := drainFake(t, ch)
		if len(got) != 4 {
			t.Fatalf("drained %d event(s), want 4", len(got))
		}
		// A second receive on an already-closed channel must report closed,
		// not panic and not block — proof the close ran exactly once.
		if _, ok := <-ch; ok {
			t.Error("receiving again after drain reported a value, want the closed zero value")
		}
	})

	t.Run("ends with the terminal error event, then the channel closes exactly once", func(t *testing.T) {
		t.Parallel()

		failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, true)
		if err != nil {
			t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
		}
		terminal, err := ai.ErrorEvent(failure)
		if err != nil {
			t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
		}
		script := agenttest.Script{Steps: []agenttest.Step{
			agenttest.Emit(mustTextStartEvent(t)),
			agenttest.Emit(terminal),
		}}

		provider := agenttest.NewProvider(script)
		ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		got := drainFake(t, ch)
		if len(got) != 2 {
			t.Fatalf("drained %d event(s), want 2 (one scripted event + the terminal error event)", len(got))
		}
		payload, ok := got[len(got)-1].ErrorPayload()
		if !ok {
			t.Fatal("last event carries no ErrorPayload, want the terminal error event")
		}
		if payload.Category() != ai.FailureCategoryUnavailable {
			t.Errorf("terminal event's category = %v, want %v", payload.Category(), ai.FailureCategoryUnavailable)
		}
	})

	t.Run("cancelling closes the channel and the producer exits rather than blocking forever with no reader left", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		provider := agenttest.NewProvider(mustTextDeltaScript(t)) // unbuffered, no reader ever

		ch, err := provider.Stream(ctx, mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		cancel() // the only way the blocked send can ever resolve
		settleAfterFakeCancel()
		got := drainFake(t, ch)
		if len(got) != 0 {
			t.Errorf("received %d event(s) with no reader ever present, want 0 (the send must select on cancellation)", len(got))
		}
	})

	// S-AFP-009/010 — repeated cancel iterations under -race, mirroring
	// AI-20.3's own 20-iteration pattern: if the fake ever closed twice, or
	// sent after close, this would panic ("close of closed channel" or
	// "send on closed channel") rather than merely fail an assertion.
	for i := range 20 {
		t.Run(fmt.Sprintf("repeated_cancel_iteration_%d", i), func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			provider := agenttest.NewProvider(mustTextDeltaScript(t))
			ch, err := provider.Stream(ctx, mustSimpleRequest(t))
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}
			cancel()
			drainFake(t, ch)
		})
	}
}

// AI-21.1 item 1's third scenario (S-AFP-003) — the fake is exported
// production code, reachable at its own package's import path, a direct
// sibling of src/ai. provider_signature_guard_test.go proves the AI-20
// signature guard itself still passes, unaffected by this file's
// existence — this assertion is this file's own share of S-AFP-003: the
// type this package exports really does live in package agenttest.
func TestProvider_Type_IsExportedFromTheAgenttestPackage(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(agenttest.NewProvider())
	const wantPath = "github.com/cachicamas/backend/agent/src/agenttest"
	if got := typ.Elem().PkgPath(); got != wantPath {
		t.Errorf("agenttest.Provider's package path = %q, want %q", got, wantPath)
	}
}
