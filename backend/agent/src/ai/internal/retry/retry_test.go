package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

func TestLoop_NoOpExecuteOnceReturnsResponse(t *testing.T) {
	want := &http.Response{StatusCode: http.StatusOK}
	got, retries, err := Loop(context.Background(), []byte("body"), Config{MaxAttempts: 1}, func(context.Context, []byte) (*http.Response, error) {
		return want, nil
	})
	if err != nil {
		t.Fatalf("Loop() error = %v, want nil", err)
	}
	if got != want {
		t.Fatalf("Loop() response = %p, want %p", got, want)
	}
	if retries != 0 {
		t.Fatalf("Loop() retries = %d, want 0 (AI-37, AD-6: an unretried success reports 0)", retries)
	}
}

// noopSleep never actually sleeps and never errors — used by every
// AI-37/AD-6 subtest below that does not itself exercise the sleep-error
// exit.
func noopSleep(context.Context, time.Duration) error { return nil }

// mustRetryableFailure builds a retryable *ai.Failure for the executeOnce
// stubs below — AI-37/AD-6's retry-count table needs full control over
// Retryable() per call, which a plain wrapped error (retryableFor's
// default) does not give.
func mustRetryableFailure(t *testing.T) error {
	t.Helper()
	f, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, Retryable: true})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure(retryable) error = %v, want nil", err)
	}
	return f
}

// mustNonRetryableFailure builds a non-retryable *ai.Failure.
func mustNonRetryableFailure(t *testing.T) error {
	t.Helper()
	f, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryAuthentication, Retryable: false})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure(non-retryable) error = %v, want nil", err)
	}
	return f
}

// TestLoop_RetryCountOnEveryExit covers AI-37's D-3c/AD-6: Loop returns the
// number of retries performed (attempt-1 against the 1-based attempt
// counter) on every one of its six return sites, not only on success —
// retry.go's five other exits carried no count at all before this
// milestone (proposal D-3c).
func TestLoop_RetryCountOnEveryExit(t *testing.T) {
	t.Run("unretried_success", func(t *testing.T) {
		want := &http.Response{StatusCode: http.StatusOK}
		calls := 0
		resp, retries, err := Loop(context.Background(), nil, Config{MaxAttempts: 3, SleepFunc: noopSleep}, func(context.Context, []byte) (*http.Response, error) {
			calls++
			return want, nil
		})
		if err != nil {
			t.Fatalf("Loop() error = %v, want nil", err)
		}
		if resp != want {
			t.Fatalf("Loop() response = %p, want %p", resp, want)
		}
		if retries != 0 {
			t.Errorf("retries = %d, want 0 (unretried success)", retries)
		}
		if calls != 1 {
			t.Errorf("executeOnce called %d time(s), want 1", calls)
		}
	})

	t.Run("success_after_two_retries", func(t *testing.T) {
		want := &http.Response{StatusCode: http.StatusOK}
		calls := 0
		resp, retries, err := Loop(context.Background(), nil, Config{MaxAttempts: 3, SleepFunc: noopSleep}, func(context.Context, []byte) (*http.Response, error) {
			calls++
			if calls <= 2 {
				return nil, mustRetryableFailure(t)
			}
			return want, nil
		})
		if err != nil {
			t.Fatalf("Loop() error = %v, want nil", err)
		}
		if resp != want {
			t.Fatalf("Loop() response = %p, want %p", resp, want)
		}
		if retries != 2 {
			t.Errorf("retries = %d, want 2 (two retryable failures before success)", retries)
		}
		if calls != 3 {
			t.Errorf("executeOnce called %d time(s), want 3", calls)
		}
	})

	t.Run("non_retryable_terminal", func(t *testing.T) {
		calls := 0
		_, retries, err := Loop(context.Background(), nil, Config{MaxAttempts: 3, SleepFunc: noopSleep}, func(context.Context, []byte) (*http.Response, error) {
			calls++
			return nil, mustNonRetryableFailure(t)
		})
		if err == nil {
			t.Fatal("Loop() error = nil, want a non-retryable terminal failure")
		}
		if retries != 0 {
			t.Errorf("retries = %d, want 0 (non-retryable terminal on the first attempt)", retries)
		}
		if calls != 1 {
			t.Errorf("executeOnce called %d time(s), want 1 — a non-retryable failure must not retry", calls)
		}
	})

	t.Run("budget_exhausted", func(t *testing.T) {
		const maxAttempts = 3
		calls := 0
		_, retries, err := Loop(context.Background(), nil, Config{MaxAttempts: maxAttempts, SleepFunc: noopSleep}, func(context.Context, []byte) (*http.Response, error) {
			calls++
			return nil, mustRetryableFailure(t)
		})
		if err == nil {
			t.Fatal("Loop() error = nil, want a budget-exhausted *AttemptReport")
		}
		var report *AttemptReport
		if !errors.As(err, &report) {
			t.Fatalf("errors.As(err, *AttemptReport) = false; err = %v", err)
		}
		if retries != maxAttempts {
			t.Errorf("retries = %d, want %d (= MaxAttempts, every retry exhausted)", retries, maxAttempts)
		}
		if calls != maxAttempts+1 {
			t.Errorf("executeOnce called %d time(s), want %d (the initial attempt plus every retry)", calls, maxAttempts+1)
		}
	})

	t.Run("ctx_cancelled_before_sleep", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		calls := 0
		_, retries, err := Loop(ctx, nil, Config{MaxAttempts: 5, SleepFunc: noopSleep}, func(context.Context, []byte) (*http.Response, error) {
			calls++
			if calls == 2 {
				// Cancel from inside the second attempt so Loop
				// observes ctx.Err() != nil immediately after that
				// attempt's own failure, before it would otherwise
				// sleep and retry a third time.
				cancel()
			}
			return nil, mustRetryableFailure(t)
		})
		if err == nil {
			t.Fatal("Loop() error = nil, want the last observed failure")
		}
		if retries != 1 {
			t.Errorf("retries = %d, want 1 (one retry — attempt 2 — before cancellation was observed)", retries)
		}
		if calls != 2 {
			t.Errorf("executeOnce called %d time(s), want 2 — cancellation must stop a third attempt", calls)
		}
	})

	t.Run("sleep_interrupted", func(t *testing.T) {
		sleepErr := errors.New("sleep interrupted")
		calls := 0
		sleeps := 0
		_, retries, err := Loop(context.Background(), nil, Config{MaxAttempts: 5, SleepFunc: func(context.Context, time.Duration) error {
			sleeps++
			return sleepErr
		}}, func(context.Context, []byte) (*http.Response, error) {
			calls++
			return nil, mustRetryableFailure(t)
		})
		if err == nil {
			t.Fatal("Loop() error = nil, want the last observed failure")
		}
		if retries != 0 {
			t.Errorf("retries = %d, want 0 (the first attempt's own retryable failure, interrupted before it could sleep and retry)", retries)
		}
		if calls != 1 {
			t.Errorf("executeOnce called %d time(s), want 1 — a sleep interruption must stop a second attempt", calls)
		}
		if sleeps != 1 {
			t.Errorf("SleepFunc called %d time(s), want 1", sleeps)
		}
	})
}

func TestAttemptReport_ErrorAndUnwrap(t *testing.T) {
	cause := errors.New("final failure")
	report := &AttemptReport{Attempts: 4, FinalCause: cause}
	if report.Error() == "" {
		t.Fatal("AttemptReport.Error() returned an empty message")
	}
	if !errors.Is(report, cause) {
		t.Fatal("AttemptReport does not unwrap its final cause")
	}
}

func TestApplyDefaults_SubstitutesZeroValues(t *testing.T) {
	cfg := applyDefaults(Config{})
	if cfg.MaxAttempts != DefaultMaxAttempts {
		t.Fatalf("MaxAttempts = %d, want %d", cfg.MaxAttempts, DefaultMaxAttempts)
	}
	if cfg.NowFunc == nil || cfg.SleepFunc == nil || cfg.AfterReader == nil {
		t.Fatal("applyDefaults left a timing or Retry-After function nil")
	}
	if cfg.BaseDelay <= 0 || cfg.MaxDelay <= cfg.BaseDelay {
		t.Fatalf("delay defaults = (%s, %s), want positive bounded delays", cfg.BaseDelay, cfg.MaxDelay)
	}
}

func TestComputeBackoff_BoundedExponentialWithSeededJitter(t *testing.T) {
	cfg := Config{BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}
	first := newJitter(42, time.Unix(0, 0))
	second := newJitter(42, time.Unix(0, 0))
	for attempt := 1; attempt <= 4; attempt++ {
		got := computeBackoff(attempt, cfg, first)
		want := computeBackoff(attempt, cfg, second)
		if got != want {
			t.Fatalf("attempt %d: seeded delays differ: got %s, want %s", attempt, got, want)
		}
		lower := cfg.BaseDelay
		for i := 1; i < attempt; i++ {
			lower *= 2
		}
		if lower > cfg.MaxDelay {
			lower = cfg.MaxDelay
		}
		if got < lower || got > 2*lower {
			t.Fatalf("attempt %d: delay %s outside [%s, %s]", attempt, got, lower, 2*lower)
		}
	}
}
