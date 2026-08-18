// AG-15.2 — strict-TDD tests for the timing seam, backoff, retry-after
// override, interrupt-during-backoff and the composed ceiling
// (R-RTY-005..009, S-RTY-005..009 + bite S-RTY-012).
//
// package agent_test throughout (NFR-RTY-001): every scenario here
// drives only Harness.Run or the package's exported RetryTiming /
// DefaultRetrySleep surface, never retryDecision or computeRetryBackoff
// directly. Zero time.Sleep and zero wall-clock synchronization
// (NFR-RTY-002): every wait is either a recording SleepFunc that
// returns nil immediately, or a channel signal a custom SleepFunc
// sends before blocking on ctx.Done().
package agent_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// instantSleep is a RetryTiming.SleepFunc that never actually waits —
// the "recording sleep function (returns nil without waiting)"
// convention retry_policy_test.go's own TestHarness_RetryVisibleAttempts
// comment already names, made concrete here for every Phase 2 test that
// only cares about attempt counts, not the delay values themselves.
func instantSleep(context.Context, time.Duration) error { return nil }

// recordingSleep is a RetryTiming.SleepFunc that captures every
// requested delay, in order, then returns nil immediately — the
// deterministic observation point every S-RTY-006/007 assertion below
// reads instead of a wall clock.
type recordingSleep struct {
	mu     sync.Mutex
	delays []time.Duration
}

func (r *recordingSleep) sleep(_ context.Context, d time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.delays = append(r.delays, d)
	return nil
}

func (r *recordingSleep) Delays() []time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]time.Duration, len(r.delays))
	copy(out, r.delays)
	return out
}

// fixedNow is a RetryTiming.NowFunc that always reports the same
// instant — jitter reproducibility (S-RTY-006) requires the seed input
// to be pinned, not merely the seed field itself.
func fixedNow() time.Time { return time.Unix(1700000000, 0) }

// mustPreStreamRetryableFailure builds a retryable, no-partial-output
// pre-stream *ai.Failure, optionally carrying a retry-after hint —
// every S-RTY-005/006/007/008 fixture's evidence shape.
func mustPreStreamRetryableFailure(t *testing.T, retryAfter ai.RetryDelay) *ai.Failure {
	t.Helper()
	f, err := ai.PreStreamFailure(ai.FailureReport{
		Category:   ai.FailureCategoryUnavailable,
		Retryable:  true,
		RetryAfter: retryAfter,
	})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure returned %v, want no failure", err)
	}
	return f
}

// TestHarness_BoundHoldsAboveH — S-RTY-005, charter AG-15.2 scenario 2,
// first Then. Given a provider wrapper failing retryably pre-stream on
// strictly more calls than H, when the run is driven to its terminal,
// then the wrapper recorded exactly H requests and the inner provider
// recorded none; re-run on a harness whose bound is a different
// positive value H' records exactly H' requests — the bound is the
// field's value, not a hardcoded constant.
func TestHarness_BoundHoldsAboveH(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bound int
	}{
		{name: "default-shaped bound", bound: 3},
		{name: "a different positive bound", bound: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
			inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
			// Fails strictly more than the largest bound this table
			// tests, so the harness's own budget -- not the wrapper's
			// script length -- is what stops the attempts.
			provider := newPreStreamFailingProvider(10, failure, inner)

			hist := agent.NewHistory()
			h := agent.Harness{
				Provider:      provider,
				System:        "system prompt for bound holds above H",
				History:       hist,
				RetryAttempts: tt.bound,
				RetryTiming:   agent.RetryTiming{SleepFunc: instantSleep},
			}

			sink := make(chan *agent.Event, 128)
			_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
			if err == nil {
				t.Fatal("Run returned err = nil, want the exhausted-retry failure")
			}
			_ = drainSink(t, sink)

			if n := len(provider.Requests()); n != tt.bound {
				t.Errorf("wrapper recorded %d request(s), want exactly %d (H)", n, tt.bound)
			}
			if n := len(inner.Requests()); n != 0 {
				t.Errorf("inner provider recorded %d request(s), want 0 (never reached -- the bound exhausted first)", n)
			}
		})
	}
}

// TestRetryTiming_BackoffJitterAndClamp — S-RTY-006's behavioral half.
// Given a harness whose timing seam pins NowFunc and JitterSeed, when
// the retry backoff is exercised across successive attempts via a
// recording SleepFunc, then the delays are reproducible for that seed
// across two independent runs of the identical fixture, strictly
// increasing while unclamped, and never exceed MaxDelay once clamped.
func TestRetryTiming_BackoffJitterAndClamp(t *testing.T) {
	t.Parallel()

	const failCount = 5 // 5 retryable pre-stream failures, then success -- H = 6.
	const baseDelay = 50 * time.Millisecond
	const maxDelay = 500 * time.Millisecond
	const jitterSeed = int64(424242)

	runOnce := func(t *testing.T) []time.Duration {
		t.Helper()
		failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(failCount, failure, inner)

		rec := &recordingSleep{}
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for backoff jitter and clamp",
			History:       agent.NewHistory(),
			RetryAttempts: failCount + 1,
			RetryTiming: agent.RetryTiming{
				SleepFunc:  rec.sleep,
				NowFunc:    fixedNow,
				BaseDelay:  baseDelay,
				MaxDelay:   maxDelay,
				JitterSeed: jitterSeed,
			},
		}
		sink := make(chan *agent.Event, 128)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil (retry succeeds within the bound)", err)
		}
		_ = drainSink(t, sink)
		return rec.Delays()
	}

	first := runOnce(t)
	if len(first) != failCount {
		t.Fatalf("recorded %d delay(s), want exactly %d (one per failed attempt)", len(first), failCount)
	}
	for i, d := range first {
		if d <= 0 {
			t.Errorf("delay[%d] = %v, want a positive duration", i, d)
		}
		if d > maxDelay {
			t.Errorf("delay[%d] = %v, want <= MaxDelay (%v)", i, d, maxDelay)
		}
		if i > 0 && d < first[i-1] {
			t.Errorf("delay[%d] = %v < delay[%d] = %v, want non-decreasing growth (exponential backoff)", i, d, i-1, first[i-1])
		}
	}
	if first[len(first)-1] != maxDelay {
		t.Errorf("last delay = %v, want exactly MaxDelay (%v) -- growth should have clamped by attempt %d", first[len(first)-1], maxDelay, failCount)
	}

	second := runOnce(t)
	if len(second) != len(first) {
		t.Fatalf("second run recorded %d delay(s), want %d (reproducibility for the same seed)", len(second), len(first))
	}
	for i := range first {
		if second[i] != first[i] {
			t.Errorf("delay[%d]: first run = %v, second run = %v, want identical (JitterSeed pins the sequence)", i, first[i], second[i])
		}
	}
}

// TestHarness_RetryAfterOverridesBackoff — S-RTY-007, charter AG-15.2
// scenario 1, first Then. A reported retry-after value wins over the
// computed backoff, clamped to MaxDelay; an absent value falls back to
// the computed backoff.
func TestHarness_RetryAfterOverridesBackoff(t *testing.T) {
	t.Parallel()

	t.Run("retry-after present overrides computed backoff", func(t *testing.T) {
		t.Parallel()

		const retryAfter = 7500 * time.Millisecond
		failure := mustPreStreamRetryableFailure(t, ai.Delay(retryAfter))
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(1, failure, inner)

		rec := &recordingSleep{}
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for retry-after override",
			History:       agent.NewHistory(),
			RetryAttempts: 2,
			RetryTiming: agent.RetryTiming{
				SleepFunc: rec.sleep,
				NowFunc:   fixedNow,
				BaseDelay: 100 * time.Millisecond, // computed backoff for attempt 1 would be in [100ms,200ms)
				MaxDelay:  30 * time.Second,
			},
		}
		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		_ = drainSink(t, sink)

		delays := rec.Delays()
		if len(delays) != 1 {
			t.Fatalf("recorded %d delay(s), want exactly 1", len(delays))
		}
		if delays[0] != retryAfter {
			t.Errorf("requested delay = %v, want exactly the reported retry-after value %v", delays[0], retryAfter)
		}
		if delays[0] >= 100*time.Millisecond && delays[0] < 200*time.Millisecond {
			t.Errorf("requested delay = %v falls inside the computed-backoff range for attempt 1, want it to differ (retry-after must win)", delays[0])
		}
	})

	t.Run("retry-after absent falls back to computed backoff", func(t *testing.T) {
		t.Parallel()

		failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
		inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		provider := newPreStreamFailingProvider(1, failure, inner)

		rec := &recordingSleep{}
		h := agent.Harness{
			Provider:      provider,
			System:        "system prompt for retry-after absent",
			History:       agent.NewHistory(),
			RetryAttempts: 2,
			RetryTiming: agent.RetryTiming{
				SleepFunc: rec.sleep,
				NowFunc:   fixedNow,
				BaseDelay: 100 * time.Millisecond,
				MaxDelay:  30 * time.Second,
			},
		}
		sink := make(chan *agent.Event, 64)
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		if err != nil {
			t.Fatalf("Run returned err = %v, want nil", err)
		}
		_ = drainSink(t, sink)

		delays := rec.Delays()
		if len(delays) != 1 {
			t.Fatalf("recorded %d delay(s), want exactly 1", len(delays))
		}
		if delays[0] < 100*time.Millisecond || delays[0] >= 200*time.Millisecond {
			t.Errorf("requested delay = %v, want in [BaseDelay, 2*BaseDelay) = [100ms,200ms) -- an absent retry-after must fall back to the computed backoff for attempt 1", delays[0])
		}
	})
}

// TestHarness_InterruptAbortsBackoff — S-RTY-008, charter AG-15.2
// scenario 1, second Then. The backoff wait selects on the run's own
// context; an interrupt fired while the harness is waiting aborts the
// wait and winds the run down, never surfacing a failed run-close.
func TestHarness_InterruptAbortsBackoff(t *testing.T) {
	t.Parallel()

	failure := mustPreStreamRetryableFailure(t, ai.RetryDelay{})
	inner := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	provider := newPreStreamFailingProvider(2, failure, inner)

	sleeping := make(chan struct{})
	var sleepOnce sync.Once
	blockingSleep := func(ctx context.Context, _ time.Duration) error {
		sleepOnce.Do(func() { close(sleeping) })
		<-ctx.Done()
		return ctx.Err()
	}

	h := agent.Harness{
		Provider:      provider,
		System:        "system prompt for interrupt aborts backoff",
		History:       agent.NewHistory(),
		RetryAttempts: 3,
		RetryTiming:   agent.RetryTiming{SleepFunc: blockingSleep},
	}

	sink := make(chan *agent.Event, 64)
	type runOutcome struct {
		err error
	}
	resultCh := make(chan runOutcome, 1)
	go func() {
		_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- runOutcome{err: err}
	}()

	<-sleeping
	h.Interrupt()

	got := drainSink(t, sink)
	result := <-resultCh

	if result.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
	}
	if !errors.Is(result.err, agent.ErrInterrupted) {
		t.Errorf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", result.err)
	}

	if n := len(provider.Requests()); n != 1 {
		t.Errorf("provider recorded %d request(s), want exactly 1 (no request after the interrupt signal)", n)
	}

	var runEnds, interruptedEnds, failedEnds int
	for _, ev := range got {
		if re, ok := ev.RunEnd(); ok {
			runEnds++
			switch re.Outcome() {
			case agent.RunOutcomeInterrupted:
				interruptedEnds++
				if _, hasFailure := re.Failure(); hasFailure {
					t.Error("run_end(Interrupted) carries a *Failure, want nil (RunEnd.validate's failure-iff-Failed rule)")
				}
			case agent.RunOutcomeFailed:
				failedEnds++
			}
		}
	}
	if runEnds != 1 {
		t.Errorf("run_end count = %d, want exactly 1", runEnds)
	}
	if interruptedEnds != 1 {
		t.Errorf("run_end(Interrupted) count = %d, want exactly 1", interruptedEnds)
	}
	if failedEnds != 0 {
		t.Errorf("run_end(Failed) count = %d, want 0 -- no failed run-close may appear anywhere on the stream", failedEnds)
	}

	report := agent.CheckStream(got)
	if v := report.Violation(); v != nil {
		t.Errorf("CheckStream rejected the interrupted-during-backoff stream: %v, want it accepted unmodified", v)
	}
}

// TestDefaultRetrySleep_PreCancelledContextReturnsImmediately —
// S-RTY-008's direct unit test of the production default sleep
// function: called with an already-cancelled context, it returns
// immediately with a non-nil error and creates no observable delay.
// The elapsed-time bound below is a diagnostic ceiling proving no real
// timer was armed, not a synchronization mechanism (NFR-RTY-002 governs
// synchronization; this assertion never coordinates two goroutines).
func TestDefaultRetrySleep_PreCancelledContextReturnsImmediately(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := agent.DefaultRetrySleep(ctx, 5*time.Second)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("DefaultRetrySleep returned err = nil for a pre-cancelled context, want a non-nil error")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("DefaultRetrySleep took %v to return for a pre-cancelled context, want an immediate return with no observable delay", elapsed)
	}
}

// --- S-RTY-009 / bite S-RTY-012 -------------------------------------

// wantLayer1RetryDocSentences are the exact Layer 1 sentences this
// capability's own documentation must stay consistent with (R-RTY-009).
// Bite S-RTY-012 perturbs a LOCAL copy of this slice -- never the real
// ai/internal/retry/doc.go file on disk, which stays byte-unchanged
// per R-RUN-012 -- proving the comparison below is genuinely
// load-bearing rather than a comment nobody re-reads.
var wantLayer1RetryDocSentences = []string{
	"DefaultMaxAttempts = 3",
	"N+1 = 4 wire requests",
}

// wantRetryPolicyDocSubstrings are the composed-ceiling phrases
// retry_policy.go's own package documentation must state (task 2.4):
// the formula in this layer's wording, the concrete value at the
// shipped defaults, and the total-attempts convention (R-RTY-005).
var wantRetryPolicyDocSubstrings = []string{
	"H × 4 = 12",
	"TOTAL attempts",
	"ai/internal/retry/doc.go",
}

// checkComposedCeilingWording reports every sentence from want that is
// missing from content, verbatim. An empty return means every cited
// sentence is present.
func checkComposedCeilingWording(content string, want []string) []string {
	var missing []string
	for _, w := range want {
		if !strings.Contains(content, w) {
			missing = append(missing, w)
		}
	}
	return missing
}

// readAgentPackageFile reads name from this test's own package
// directory (backend/agent/src/agent/), a plain file read -- not an
// import -- exactly as design AD-8 records: "reading a file is not
// importing a package". go test's working directory is always the
// package's own source directory, so a relative path needs no git
// shell-out and no repo-root resolution.
func readAgentPackageFile(t *testing.T, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned %v, want no error", relPath, err)
	}
	return string(data)
}

// TestComposedCeiling_MatchesLayer1Wording — S-RTY-009, charter
// AG-15.2 scenario 2, second Then. The Layer 1 retry helper's package
// documentation, read as a file, states its own retry-budget sentences
// verbatim; this capability's own policy documentation states the
// composed formula in the same wording together with the value the
// shipped defaults evaluate to and the total-attempts convention.
func TestComposedCeiling_MatchesLayer1Wording(t *testing.T) {
	t.Parallel()

	layer1Doc := readAgentPackageFile(t, "../ai/internal/retry/doc.go")
	if missing := checkComposedCeilingWording(layer1Doc, wantLayer1RetryDocSentences); len(missing) != 0 {
		t.Errorf("ai/internal/retry/doc.go is missing the cited sentence(s) %v -- this capability's ceiling documentation cites it verbatim and has drifted", missing)
	}

	layer2Doc := readAgentPackageFile(t, "retry_policy.go")
	if missing := checkComposedCeilingWording(layer2Doc, wantRetryPolicyDocSubstrings); len(missing) != 0 {
		t.Errorf("retry_policy.go's package documentation is missing the composed-ceiling substring(s) %v (R-RTY-009, task 2.4)", missing)
	}
}
