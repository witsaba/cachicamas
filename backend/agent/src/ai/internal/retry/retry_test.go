package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestLoop_NoOpExecuteOnceReturnsResponse(t *testing.T) {
	want := &http.Response{StatusCode: http.StatusOK}
	got, err := Loop(context.Background(), []byte("body"), Config{MaxAttempts: 1}, func(context.Context, []byte) (*http.Response, error) {
		return want, nil
	})
	if err != nil {
		t.Fatalf("Loop() error = %v, want nil", err)
	}
	if got != want {
		t.Fatalf("Loop() response = %p, want %p", got, want)
	}
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
