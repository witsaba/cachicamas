// AG-15 — strict-TDD tests for the retry policy and the failover seam
// (R-RTY-001..012, S-RTY-001..015 + bites S-RTY-010, S-RTY-011,
// S-RTY-012).
//
// Every scenario is in package agent_test (NFR-RTY-001): the
// external-test posture proves every behavioral claim from outside
// the package. The one exception is the predicate's own directly-
// callable table-driven unit test (S-RTY-001), which NFR-RTY-001
// itself permits to exercise the predicate "through whatever surface
// sdd-design fixed" — design AD-2 fixes `retryDecision` and
// `retryVerdict` as unexported, so that one test lives in a separate,
// package-internal file (retry_decision_internal_test.go), not here.
// Every OTHER behavioral claim this spec makes is independently
// provable through this file's tests, driving only the exported
// surface (Turn, Harness).
//
// # Phase 0 (D1) — the pre-stream bracket gap
//
// D1 closes the gap Turn's three pre-stream failure paths left open
// since AG-07: buildLoopRequest (loop.go:304-308), the pre-request
// hook (loop.go:317-328), and provider.Stream (loop.go:332-338) each
// left the turn bracket open on failure, so CheckStream rejected any
// second turn_start on the same lane (stream_check.go:141-143) — the
// exact shape a harness retry over a pre-output failure would
// produce. Without D1, no AG-15.1 retry scenario can pass
// (agent-loop-skeleton's R-LSK-001 delta, S-LSK-021).
package agent_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// preStreamFailingProvider is a test-local ai.ModelProvider wrapper —
// the fixture constraint every AG-15.1 scenario needing a scripted
// retryable pre-stream failure MUST use (agent-retry-failover
// spec.md's "Fixture constraint" section, the errorProvider
// precedent at loop_test.go:1408-1421): agenttest.Script cannot
// express one on its own (fake_provider.go:74-117 fails pre-stream
// only on a zero request, an already-cancelled context, or script
// exhaustion). It fails its first failCount Stream calls with a
// scripted *ai.Failure, captures its own requests independently of
// the inner provider, then delegates every later call to inner.
// Test-local only — agenttest itself is never widened by this
// change.
type preStreamFailingProvider struct {
	mu        sync.Mutex
	failCount int
	failure   *ai.Failure
	calls     int
	requests  []ai.Request
	inner     ai.ModelProvider
}

// newPreStreamFailingProvider constructs a wrapper that fails its
// first failCount Stream calls with failure, then delegates to inner.
func newPreStreamFailingProvider(failCount int, failure *ai.Failure, inner ai.ModelProvider) *preStreamFailingProvider {
	return &preStreamFailingProvider{failCount: failCount, failure: failure, inner: inner}
}

// Stream implements ai.ModelProvider.
func (p *preStreamFailingProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.requests = append(p.requests, req)
	p.mu.Unlock()

	if call <= p.failCount {
		return nil, p.failure
	}
	return p.inner.Stream(ctx, req)
}

// Requests returns every request this wrapper itself captured, in
// call order — independent of the inner provider's own Requests(),
// so a test can prove the wrapper's own call count is exactly H
// regardless of how many (if any) calls reached inner.
func (p *preStreamFailingProvider) Requests() []ai.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ai.Request, len(p.requests))
	copy(out, p.requests)
	return out
}

// eventKindsOf renders events' kinds in order, for a failed
// assertion's diagnostic message.
func eventKindsOf(events []agent.Event) []agent.EventKind {
	kinds := make([]agent.EventKind, len(events))
	for i, e := range events {
		kinds[i] = e.Kind()
	}
	return kinds
}

// assertPreStreamAbortSequence asserts the AG-15 D1 event sequence
// for one of Turn's three pre-stream failure paths (S-LSK-021):
// run_start (nil-continuation path only), turn_start,
// turn_end(Aborted) carrying a non-nil *Failure, and (nil-
// continuation path only) run_end(Failed) carrying a non-nil
// *Failure — in that order, and nothing else on the stream.
func assertPreStreamAbortSequence(t *testing.T, got []agent.Event, nilContinuation bool) {
	t.Helper()

	wantKinds := []agent.EventKind{agent.EventKindTurnStart, agent.EventKindTurnEnd}
	if nilContinuation {
		wantKinds = []agent.EventKind{
			agent.EventKindRunStart,
			agent.EventKindTurnStart,
			agent.EventKindTurnEnd,
			agent.EventKindRunEnd,
		}
	}

	if len(got) != len(wantKinds) {
		t.Fatalf("event sequence = %v, want kinds %v (nilContinuation=%v)", eventKindsOf(got), wantKinds, nilContinuation)
	}
	for i, wantKind := range wantKinds {
		if got[i].Kind() != wantKind {
			t.Fatalf("event[%d].Kind() = %v, want %v (full sequence: %v)", i, got[i].Kind(), wantKind, eventKindsOf(got))
		}
	}

	turnEndIdx := 1
	if nilContinuation {
		turnEndIdx = 2
	}
	turnEnd, ok := got[turnEndIdx].TurnEnd()
	if !ok {
		t.Fatalf("event[%d] does not carry a TurnEnd payload", turnEndIdx)
	}
	if turnEnd.Outcome() != agent.TurnOutcomeAborted {
		t.Errorf("turn_end outcome = %v, want %v", turnEnd.Outcome(), agent.TurnOutcomeAborted)
	}
	if failure, hasFailure := turnEnd.Failure(); !hasFailure || failure == nil {
		t.Error("turn_end carries no *Failure, want a non-nil one (AG-15 D1, R-LSK-001)")
	}

	if !nilContinuation {
		return
	}

	runEnd, ok := got[3].RunEnd()
	if !ok {
		t.Fatal("event[3] does not carry a RunEnd payload")
	}
	if runEnd.Outcome() != agent.RunOutcomeFailed {
		t.Errorf("run_end outcome = %v, want %v", runEnd.Outcome(), agent.RunOutcomeFailed)
	}
	if failure, hasFailure := runEnd.Failure(); !hasFailure || failure == nil {
		t.Error("run_end carries no *Failure, want a non-nil one (AG-15 D1, R-LSK-001)")
	}
}

// fullTurnContinuation builds a fully-configured TurnContinuation —
// the harness_test.go:38-45 fixture shape, reused here so a
// continuation-path pre-stream failure can be driven directly
// through Turn without a Harness.
func fullTurnContinuation(run agent.RunID) *agent.TurnContinuation {
	return &agent.TurnContinuation{
		Run:       run,
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{},
		History:   agent.NewHistory(),
	}
}

// TestTurn_PreStreamBuildErrorEmitsTurnEnd — AG-15 D1, first of three
// pre-stream paths (loop.go:304-308). An empty system string makes
// buildLoopRequest fail before any I/O (ai.NewSystemText rejects it,
// system_instruction.go:37) — the "None" cell of design.md's
// existing-test enumeration table: no test today drives this path.
func TestTurn_PreStreamBuildErrorEmitsTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("nil continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider() // never consulted: build fails first
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"", // empty system: buildLoopRequest fails (R-LSK-001)
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the request-build error")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, true)
	})

	t.Run("continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider()
		sink := make(chan *agent.Event, 16)
		cont := fullTurnContinuation("run-rty-d1-build")

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{Continuation: cont},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the request-build error")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, false)
	})
}

// TestTurn_PreStreamHookErrorEmitsTurnEnd — AG-15 D1, second of three
// pre-stream paths (loop.go:317-328). Reuses hookBoomAlwaysErrors
// (loop_hook_test.go:107), the S-PRH-003 fixture already proving the
// hook aborts before I/O; this test additionally proves the bracket
// now closes.
func TestTurn_PreStreamHookErrorEmitsTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("nil continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 hook error",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{PreRequestHook: hookBoomAlwaysErrors()},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the hook's typed pre-stream failure")
		}
		if len(provider.Requests()) != 0 {
			t.Errorf("provider captured %d request(s), want 0 (hook failure aborts BEFORE provider.Stream)", len(provider.Requests()))
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, true)
	})

	t.Run("continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)
		cont := fullTurnContinuation("run-rty-d1-hook")

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 hook error continuation",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{PreRequestHook: hookBoomAlwaysErrors(), Continuation: cont},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the hook's typed pre-stream failure")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, false)
	})
}

// TestTurn_PreStreamProviderErrorEmitsTurnEnd — AG-15 D1, third of
// three pre-stream paths (loop.go:332-338). Reuses errorProvider
// (loop_test.go:1408-1421), the S-LSK-011-adjacent fixture
// TestTurn_ProviderPreStreamFailureSurfacesOnReturn already drives
// (loop_test.go:1436-1461, which stays green byte-unchanged per
// design's enumeration table); this test additionally proves the
// bracket now closes.
func TestTurn_PreStreamProviderErrorEmitsTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("nil continuation", func(t *testing.T) {
		t.Parallel()

		provider := errorProvider{}
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 provider error",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the provider's typed pre-stream failure")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, true)
	})

	t.Run("continuation", func(t *testing.T) {
		t.Parallel()

		provider := errorProvider{}
		sink := make(chan *agent.Event, 16)
		cont := fullTurnContinuation("run-rty-d1-provider")

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 provider error continuation",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{Continuation: cont},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the provider's typed pre-stream failure")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, false)
	})
}

// TestHarness_RetryVisibleAttempts — S-RTY-002, charter AG-15.1
// scenario 1. Given a provider wrapper scripted to fail its first
// H-1 calls with a retryable pre-stream *ai.Failure reporting no
// partial output, backed by an inner script that then completes the
// turn normally, when the run is driven, then the wrapper recorded
// exactly H requests; every recorded request compares equal to every
// other as an ai.Request; the emitted stream carries H turn brackets
// for that logical turn — the first H-1 closing aborted, the last
// closing normally — with pairwise distinct turn identities on one
// contiguous 1-based lane inside a single run bracket; the run-close
// carries the completed run outcome; and CheckStream accepts the
// recorded stream unmodified.
func TestHarness_RetryVisibleAttempts(t *testing.T) {
	t.Parallel()

	const bound = 3 // H

	failure, ferr := ai.PreStreamFailure(ai.FailureReport{
		Category:  ai.FailureCategoryUnavailable,
		Retryable: true,
	})
	if ferr != nil {
		t.Fatalf("ai.PreStreamFailure returned %v, want no failure", ferr)
	}

	inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	provider := newPreStreamFailingProvider(bound-1, failure, inner)

	hist := agent.NewHistory()
	h := agent.Harness{
		Provider:      provider,
		System:        "system prompt for retry visible attempts",
		History:       hist,
		RetryAttempts: bound,
	}

	sink := make(chan *agent.Event, 64)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil (the retry succeeds within the bound)", err)
	}

	got := drainSink(t, sink)

	// The wrapper recorded exactly H requests -- the count is the
	// contract -- and every attempt's request is byte-identical
	// (R-RTY-002).
	requests := provider.Requests()
	if len(requests) != bound {
		t.Fatalf("wrapper recorded %d request(s), want exactly %d (H)", len(requests), bound)
	}
	for i := 1; i < len(requests); i++ {
		if !requests[i].Equal(requests[0]) {
			t.Errorf("request[%d] != request[0]; attempts of one logical turn must be byte-identical (R-RTY-002)", i)
		}
	}
	if got := len(inner.Requests()); got != 1 {
		t.Errorf("inner provider recorded %d request(s), want exactly 1 (only the final, succeeding attempt reaches it)", got)
	}

	var runStarts, runEnds, turnStarts, turnEnds int
	var abortedCount, finishedCount int
	turnIDs := map[agent.TurnID]bool{}
	for i, ev := range got {
		wantSeq := agent.Sequence(i + 1)
		if ev.Sequence() != wantSeq {
			t.Errorf("event[%d].Sequence() = %d, want %d (one contiguous 1-based lane)", i, ev.Sequence(), wantSeq)
		}

		switch ev.Kind() {
		case agent.EventKindRunStart:
			runStarts++
		case agent.EventKindRunEnd:
			runEnds++
			if runEnd, ok := ev.RunEnd(); ok && runEnd.Outcome() != agent.RunOutcomeCompleted {
				t.Errorf("run_end outcome = %v, want %v", runEnd.Outcome(), agent.RunOutcomeCompleted)
			}
		case agent.EventKindTurnStart:
			turnStarts++
			if turnID, ok := ev.Turn(); ok {
				turnIDs[turnID] = true
			}
		case agent.EventKindTurnEnd:
			turnEnds++
			if turnEnd, ok := ev.TurnEnd(); ok {
				switch turnEnd.Outcome() {
				case agent.TurnOutcomeAborted:
					abortedCount++
					if _, hasFailure := turnEnd.Failure(); !hasFailure {
						t.Error("turn_end(Aborted) carries no *Failure, want a non-nil one")
					}
				case agent.TurnOutcomeFinished:
					finishedCount++
				}
			}
		}
	}

	if runStarts != 1 || runEnds != 1 {
		t.Errorf("run brackets: %d run_start, %d run_end, want exactly 1 each (single run bracket)", runStarts, runEnds)
	}
	if turnStarts != bound || turnEnds != bound {
		t.Errorf("turn brackets: %d turn_start, %d turn_end, want exactly %d each (H -- one bracket per attempt)", turnStarts, turnEnds, bound)
	}
	if abortedCount != bound-1 {
		t.Errorf("aborted turn_end count = %d, want %d (H-1)", abortedCount, bound-1)
	}
	if finishedCount != 1 {
		t.Errorf("finished turn_end count = %d, want 1 (only the last attempt)", finishedCount)
	}
	if len(turnIDs) != bound {
		t.Errorf("distinct turn identities observed = %d, want %d (H, pairwise distinct per attempt)", len(turnIDs), bound)
	}

	report := agent.CheckStream(got)
	if v := report.Violation(); v != nil {
		t.Errorf("CheckStream rejected the recorded multi-attempt stream: %v, want it accepted unmodified", v)
	}
}

// scriptTextThenMidStreamFailure builds a script that emits a short
// text bracket (only when outputPreceded) then a terminal mid-stream
// *ai.Failure carrying retryable and outputPreceded exactly as given
// — generalizing loop_test.go:1251's scriptTextThenMidStreamError
// precedent over the two discriminators this file's S-RTY-003/004
// scenarios need to vary independently.
func scriptTextThenMidStreamFailure(t *testing.T, retryable, outputPreceded bool) agenttest.Script {
	t.Helper()

	steps := []agenttest.Step{}
	if outputPreceded {
		start, err := ai.NewTextBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
		}
		delta, err := ai.NewTextDelta(1, "partial-before-failure")
		if err != nil {
			t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
		}
		end, err := ai.NewTextBlockEnd(1)
		if err != nil {
			t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
		}
		steps = append(steps, agenttest.Emit(start), agenttest.Emit(delta), agenttest.Emit(end))
	}

	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category:  ai.FailureCategoryUnavailable,
		Retryable: retryable,
	}, outputPreceded)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	steps = append(steps, agenttest.Emit(terminal))
	return agenttest.Script{Steps: steps}
}

// TestHarness_RetryPartialOutputNotRetried — S-RTY-003, charter
// AG-15.1 scenario 2 (the G8 sentence, now at the harness, R-RTY-003).
// A failure reporting itself retryable but carrying already-emitted
// output MUST NOT be retried: gate G3 fires ahead of G4. A further
// script is queued so a retry would be observable rather than an
// error.
func TestHarness_RetryPartialOutputNotRetried(t *testing.T) {
	t.Parallel()

	failingScript := scriptTextThenMidStreamFailure(t, true, true)
	neverConsumedScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(failingScript, neverConsumedScript)

	hist := agent.NewHistory()
	h := agent.Harness{
		Provider:      provider,
		System:        "system prompt for partial output not retried",
		History:       hist,
		RetryAttempts: 3,
	}

	sink := make(chan *agent.Event, 64)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err == nil {
		t.Fatal("Run returned err = nil, want the mid-stream failure (G3 forbids retry; the run must end failed)")
	}

	got := drainSink(t, sink)

	if n := len(provider.Requests()); n != 1 {
		t.Fatalf("provider recorded %d request(s), want exactly 1 (G3 forbids retry after emitted output — a second request is the exact defect this requirement prevents)", n)
	}

	var sawAbortedWithPartial bool
	var runOutcome agent.RunOutcome
	for _, ev := range got {
		if te, ok := ev.TurnEnd(); ok && te.Outcome() == agent.TurnOutcomeAborted {
			if failure, hasFailure := te.Failure(); hasFailure && failure.PartialOutput() {
				sawAbortedWithPartial = true
			}
		}
		if re, ok := ev.RunEnd(); ok {
			runOutcome = re.Outcome()
		}
	}
	if !sawAbortedWithPartial {
		t.Error("no turn_end(Aborted) carrying a *Failure with PartialOutput() == true observed, want the partial-content failure surfaced")
	}
	if runOutcome != agent.RunOutcomeFailed {
		t.Errorf("run_end outcome = %v, want %v", runOutcome, agent.RunOutcomeFailed)
	}
}

// TestHarness_RetryNonRetryableSurfacesImmediately — S-RTY-004,
// charter AG-15.1 scenario 3. A non-retryable failure surfaces
// immediately regardless of delivery position: gate G2 fires ahead of
// G3/G4/G5 whenever Retryable() == false. Both positions from the
// spec's Given clause — pre-stream (via the wrapper) and mid-stream
// after emitted output (via a script) — each with a further script
// queued so a retry would be observable.
func TestHarness_RetryNonRetryableSurfacesImmediately(t *testing.T) {
	t.Parallel()

	t.Run("pre-stream position", func(t *testing.T) {
		t.Parallel()

		failure, ferr := ai.PreStreamFailure(ai.FailureReport{
			Category:  ai.FailureCategoryUnavailable,
			Retryable: false,
		})
		if ferr != nil {
			t.Fatalf("ai.PreStreamFailure returned %v, want no failure", ferr)
		}
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(1, failure, inner)

		hist := agent.NewHistory()
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for non-retryable pre-stream",
			History:       hist,
			RetryAttempts: 3,
		}

		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err == nil {
			t.Fatal("Run returned err = nil, want the non-retryable pre-stream failure")
		}
		_ = drainSink(t, sink)

		if n := len(provider.Requests()); n != 1 {
			t.Errorf("wrapper recorded %d request(s), want exactly 1 (non-retryable surfaces immediately at G2)", n)
		}
		if n := len(inner.Requests()); n != 0 {
			t.Errorf("inner provider recorded %d request(s), want 0 (never reached — no retry)", n)
		}
	})

	t.Run("mid-stream position after emitted output", func(t *testing.T) {
		t.Parallel()

		failingScript := scriptTextThenMidStreamFailure(t, false, true)
		neverConsumedScript := scriptTextResponse(t, ai.FinishReasonStop)
		provider := agenttest.NewProvider(failingScript, neverConsumedScript)

		hist := agent.NewHistory()
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for non-retryable mid-stream",
			History:       hist,
			RetryAttempts: 3,
		}

		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err == nil {
			t.Fatal("Run returned err = nil, want the non-retryable mid-stream failure")
		}
		_ = drainSink(t, sink)

		if n := len(provider.Requests()); n != 1 {
			t.Errorf("provider recorded %d request(s), want exactly 1 (non-retryable surfaces immediately at G2, regardless of delivery position)", n)
		}
	})
}

// --- Phase 4 (Decision 4, R-RTY-012) --------------------------------

// findRunEnd scans got for its one RunEnd payload, failing the test if
// none is found.
func findRunEnd(t *testing.T, got []agent.Event) agent.RunEnd {
	t.Helper()
	for _, ev := range got {
		if re, ok := ev.RunEnd(); ok {
			return re
		}
	}
	t.Fatal("no run_end event found on the stream")
	return agent.RunEnd{}
}

// TestHarness_ExhaustedRetryPreservesTrueCategory — S-RTY-015,
// proposal Decision 4, first Given/Then. A run exhausting its
// attempts against a provider wrapper whose final pre-stream failure
// reports a category other than unavailable, a true retryability
// flag, a retry-after value, no partial output and pre-stream
// delivery: the run-close event's *Failure MUST report that same
// evidence, whole -- not the Unavailable category
// wrapHarnessFailure hardcodes -- and unwrapping it MUST reach the
// identical *ai.Failure value the final attempt failed with.
func TestHarness_ExhaustedRetryPreservesTrueCategory(t *testing.T) {
	t.Parallel()

	const bound = 2
	const retryAfter = 2500 * time.Millisecond
	failure, ferr := ai.PreStreamFailure(ai.FailureReport{
		Category:   ai.FailureCategoryRateLimit,
		Retryable:  true,
		RetryAfter: ai.Delay(retryAfter),
	})
	if ferr != nil {
		t.Fatalf("ai.PreStreamFailure returned %v, want no failure", ferr)
	}

	inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	provider := newPreStreamFailingProvider(bound+5, failure, inner)

	h := agent.Harness{
		Provider:      provider,
		System:        "system prompt for exhausted retry true category",
		History:       agent.NewHistory(),
		RetryAttempts: bound,
		RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
	}

	sink := make(chan *agent.Event, 128)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err == nil {
		t.Fatal("Run returned err = nil, want the exhausted-retry failure")
	}
	got := drainSink(t, sink)

	runEnd := findRunEnd(t, got)
	if runEnd.Outcome() != agent.RunOutcomeFailed {
		t.Fatalf("run_end outcome = %v, want %v", runEnd.Outcome(), agent.RunOutcomeFailed)
	}
	gotFailure, hasFailure := runEnd.Failure()
	if !hasFailure || gotFailure == nil {
		t.Fatal("run_end carries no *Failure, want a non-nil one")
	}

	if gotFailure.Category() != ai.FailureCategoryRateLimit {
		t.Errorf("run_end failure Category() = %v, want %v (the true category, not the hardcoded Unavailable)", gotFailure.Category(), ai.FailureCategoryRateLimit)
	}
	if !gotFailure.Retryable() {
		t.Error("run_end failure Retryable() = false, want true (preserved from the final attempt)")
	}
	if gotFailure.PartialOutput() {
		t.Error("run_end failure PartialOutput() = true, want false")
	}
	if gotFailure.Delivery() != ai.DeliveryPreStream {
		t.Errorf("run_end failure Delivery() = %v, want %v", gotFailure.Delivery(), ai.DeliveryPreStream)
	}

	unwrapped := gotFailure.Unwrap()
	asAIFailure, isAIFailure := unwrapped.(*ai.Failure)
	if !isAIFailure {
		t.Fatalf("run_end failure Unwrap() = %T, want *ai.Failure", unwrapped)
	}
	if asAIFailure != failure {
		t.Error("run_end failure does not unwrap to the identical *ai.Failure the final attempt failed with -- it was reconstructed, not wrapped")
	}
	if gotAfter, ok := asAIFailure.RetryAfter(); !ok || gotAfter != retryAfter {
		t.Errorf("unwrapped failure RetryAfter() = (%v, %v), want (%v, true)", gotAfter, ok, retryAfter)
	}
}

// TestHarness_ExhaustedRetryPlainErrorStaysUnavailable — S-RTY-015,
// second Given/Then. A run failed by a plain Go error cause (not an
// *ai.Failure) reports the Unavailable category exactly as it does
// today, through the existing, byte-unchanged wrapHarnessFailure.
// The empty-system fixture makes buildLoopRequest fail inside Turn
// with a plain validation error (D1's own "None" fixture, loop.go's
// buildLoopRequest error path returns err unwrapped) -- an error that
// does not satisfy errors.As against *ai.Failure, so retryDecision's
// G1 gate surfaces it immediately, exercising the exact same failRun
// call site the exhausted-retry path uses.
func TestHarness_ExhaustedRetryPlainErrorStaysUnavailable(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider() // never consulted: build fails first
	h := agent.Harness{
		Provider:      provider,
		System:        "", // empty system: buildLoopRequest fails with a plain error
		History:       agent.NewHistory(),
		RetryAttempts: 3,
		RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
	}

	sink := make(chan *agent.Event, 64)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err == nil {
		t.Fatal("Run returned err = nil, want the request-build error")
	}
	var asAIFailure *ai.Failure
	if errors.As(err, &asAIFailure) {
		t.Fatalf("Run's returned error unexpectedly reaches an *ai.Failure via errors.As (%v) -- this fixture must stay a plain error for G1 to fire", asAIFailure)
	}
	got := drainSink(t, sink)

	runEnd := findRunEnd(t, got)
	if runEnd.Outcome() != agent.RunOutcomeFailed {
		t.Fatalf("run_end outcome = %v, want %v", runEnd.Outcome(), agent.RunOutcomeFailed)
	}
	gotFailure, hasFailure := runEnd.Failure()
	if !hasFailure || gotFailure == nil {
		t.Fatal("run_end carries no *Failure, want a non-nil one")
	}
	if gotFailure.Category() != ai.FailureCategoryUnavailable {
		t.Errorf("run_end failure Category() = %v, want %v (today's unchanged behavior for a plain-error cause)", gotFailure.Category(), ai.FailureCategoryUnavailable)
	}
}
