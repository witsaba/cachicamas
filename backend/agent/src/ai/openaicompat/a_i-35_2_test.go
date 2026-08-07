// AI-35.2 — R-AIS-042 bounded, injectable and context-aware backoff.
//
// These tests call the shared helper with the real adapter's executeOnce
// operation. SleepFunc and NowFunc keep every timing assertion deterministic;
// no test waits on wall-clock time.
package openaicompat

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/internal/retry"
)

func TestAI35_2_RetryAfterHint_OverridesComputedBackoff(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error"}}`))
	}))
	t.Cleanup(server.Close)

	client := mustClient(t, server.URL)
	body, err := Translate(validRequest(t))
	if err != nil {
		t.Fatalf("Translate() error = %v, want nil", err)
	}
	var delays []time.Duration
	got, err := retry.Loop(t.Context(), body, retry.Config{
		MaxAttempts: 2,
		MaxDelay:    time.Minute,
		SleepFunc: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	}, client.executeOnce)
	if got != nil || err == nil {
		t.Fatalf("Loop() = (%v, %v), want nil response and exhaustion error", got, err)
	}
	if len(delays) != 2 || delays[0] != 5*time.Second || delays[1] != 5*time.Second {
		t.Fatalf("Retry-After delays = %v, want [5s 5s]", delays)
	}
	if requests.Load() != 3 {
		t.Fatalf("wire requests = %d, want 3", requests.Load())
	}
}

func TestAI35_2_ComputedBackoff_BoundedWithSeededJitter(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error"}}`))
	}))
	t.Cleanup(server.Close)

	client := mustClient(t, server.URL)
	body, err := Translate(validRequest(t))
	if err != nil {
		t.Fatalf("Translate() error = %v, want nil", err)
	}
	fixedNow := func() time.Time { return time.Unix(1700000000, 0) }
	collect := func() []time.Duration {
		var delays []time.Duration
		_, loopErr := retry.Loop(t.Context(), body, retry.Config{
			MaxAttempts: 3,
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    200 * time.Millisecond,
			JitterSeed:  77,
			NowFunc:     fixedNow,
			SleepFunc: func(_ context.Context, delay time.Duration) error {
				delays = append(delays, delay)
				return nil
			},
		}, client.executeOnce)
		if loopErr == nil {
			t.Fatal("Loop() error = nil, want exhaustion error")
		}
		return delays
	}

	first := collect()
	second := collect()
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("computed delay counts = (%d, %d), want (3, 3)", len(first), len(second))
	}
	for i, delay := range first {
		base := 10 * time.Millisecond
		for n := 0; n < i; n++ {
			base *= 2
		}
		if delay < base || delay > 2*base {
			t.Fatalf("attempt %d delay = %s, want [%s, %s]", i+1, delay, base, 2*base)
		}
		if delay != second[i] {
			t.Fatalf("attempt %d seeded delay = %s then %s, want deterministic sequence", i+1, delay, second[i])
		}
	}
}

func TestAI35_2_Backoff_CtxCancellationAbortsImmediately(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error"}}`))
	}))
	t.Cleanup(server.Close)

	client := mustClient(t, server.URL)
	body, err := Translate(validRequest(t))
	if err != nil {
		t.Fatalf("Translate() error = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, err = retry.Loop(ctx, body, retry.Config{
		SleepFunc: func(context.Context, time.Duration) error {
			cancel()
			return context.Canceled
		},
	}, client.executeOnce)
	if err == nil {
		t.Fatal("Loop() error = nil, want last typed failure")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(%T) did not find last *ai.Failure", err)
	}
	if failure.Category() != ai.FailureCategoryRateLimit || failure.PartialOutput() {
		t.Fatalf("failure = category=%v partial=%v, want rate-limit/false", failure.Category(), failure.PartialOutput())
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("Loop() returned context.Canceled instead of the last typed failure")
	}
	if requests.Load() != 1 {
		t.Fatalf("wire requests = %d, want 1 after cancellation", requests.Load())
	}
}

func TestAI35_2_BoundedAttemptCount_ExactlyNPlusOneWireRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error"}}`))
	}))
	t.Cleanup(server.Close)

	client := mustClient(t, server.URL)
	body, err := Translate(validRequest(t))
	if err != nil {
		t.Fatalf("Translate() error = %v, want nil", err)
	}
	_, err = retry.Loop(t.Context(), body, retry.Config{
		SleepFunc: func(context.Context, time.Duration) error { return nil },
	}, client.executeOnce)
	if err == nil {
		t.Fatal("Loop() error = nil, want exhaustion error")
	}
	if got, want := requests.Load(), int32(retry.DefaultMaxAttempts+1); got != want {
		t.Fatalf("wire requests = %d, want %d", got, want)
	}
}
