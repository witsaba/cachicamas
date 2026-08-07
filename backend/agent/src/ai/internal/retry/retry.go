// Package retry implements bounded, context-aware retries for Layer 1
// pre-stream adapter operations.
package retry

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// DefaultMaxAttempts is the number of retries permitted after the initial
// wire request. A logical call therefore issues at most N+1 requests when
// every attempt fails retryably.
const DefaultMaxAttempts = 3

const (
	defaultBaseDelay = 100 * time.Millisecond
	defaultMaxDelay  = 30 * time.Second
)

// NowFunc supplies the clock used to seed computed backoff jitter.
type NowFunc func() time.Time

// SleepFunc waits for a computed or Retry-After delay. Implementations MUST
// return promptly when ctx is cancelled.
type SleepFunc func(ctx context.Context, delay time.Duration) error

// AfterReader reads a presence-typed Retry-After hint from a failure.
type AfterReader func(*ai.Failure) (time.Duration, bool)

// Config controls one Loop invocation. MaxAttempts is the retry budget, not
// the initial request; zero values select the package defaults.
type Config struct {
	MaxAttempts      int
	NowFunc          NowFunc
	SleepFunc        SleepFunc
	AfterReader AfterReader
	BaseDelay        time.Duration
	MaxDelay         time.Duration
	JitterSeed       int64
}

// AttemptReport is the typed outer cause returned when the retry budget is
// exhausted. Unwrap exposes FinalCause so errors.As can reach both this
// report and the final *ai.Failure in one traversal.
type AttemptReport struct {
	Attempts   int
	FinalCause error
}

const attemptReportText = "retry: attempt budget exhausted"

// Error returns a stable diagnostic without reproducing provider content.
func (r *AttemptReport) Error() string { return attemptReportText }

// Unwrap exposes the final typed failure and its own cause chain.
func (r *AttemptReport) Unwrap() error { return r.FinalCause }

// Loop executes one pre-stream operation, retrying retryable typed failures
// up to cfg.MaxAttempts retries. The executeOnce callback rebuilds any
// request body for each attempt, so callers can provide a fresh reader over
// the same immutable byte slice.
func Loop(
	ctx context.Context,
	body []byte,
	cfg Config,
	executeOnce func(ctx context.Context, body []byte) (*http.Response, error),
) (*http.Response, error) {
	cfg = applyDefaults(cfg)
	jitter := newJitter(cfg.JitterSeed, cfg.NowFunc())
	var lastFailure *ai.Failure
	totalAttempts := cfg.MaxAttempts + 1

	for attempt := 1; attempt <= totalAttempts; attempt++ {
		resp, err := executeOnce(ctx, body)
		if err == nil && resp != nil {
			return resp, nil
		}

		failure := failureFromResult(err)
		lastFailure = failure
		if !failure.Retryable() {
			return nil, failure
		}
		if attempt == totalAttempts {
			return nil, &AttemptReport{Attempts: attempt, FinalCause: failure}
		}
		if ctx.Err() != nil {
			return nil, lastFailure
		}

		delay := computeBackoff(attempt, cfg, jitter)
		if hinted, ok := cfg.AfterReader(failure); ok {
			if hinted < 0 {
				hinted = 0
			}
			if hinted > cfg.MaxDelay {
				hinted = cfg.MaxDelay
			}
			delay = hinted
		}
		if err := cfg.SleepFunc(ctx, delay); err != nil {
			return nil, lastFailure
		}
	}

	return nil, &AttemptReport{Attempts: totalAttempts, FinalCause: lastFailure}
}

func failureFromResult(err error) *ai.Failure {
	var failure *ai.Failure
	if errors.As(err, &failure) && failure != nil {
		return failure
	}
	return wrapTransportError(err)
}

func wrapTransportError(err error) *ai.Failure {
	failure, constructErr := ai.PreStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    err,
	})
	if constructErr != nil {
		return &ai.Failure{}
	}
	return failure
}

func applyDefaults(cfg Config) Config {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = DefaultMaxAttempts
	}
	if cfg.NowFunc == nil {
		cfg.NowFunc = time.Now
	}
	if cfg.SleepFunc == nil {
		cfg.SleepFunc = defaultSleep
	}
	if cfg.AfterReader == nil {
		cfg.AfterReader = defaultAfterReader
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = defaultBaseDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = defaultMaxDelay
	}
	return cfg
}

func newJitter(seed int64, now time.Time) *rand.Rand {
	if seed == 0 {
		seed = now.UnixNano()
	}
	return rand.New(rand.NewPCG(uint64(seed), uint64(now.UnixNano())))
}

// computeBackoff returns bounded exponential backoff with deterministic
// multiplicative jitter in [base, 2*base), capped at MaxDelay.
func computeBackoff(attempt int, cfg Config, jitter *rand.Rand) time.Duration {
	base := cfg.BaseDelay
	for i := 1; i < attempt && base < cfg.MaxDelay; i++ {
		if base > cfg.MaxDelay/2 {
			base = cfg.MaxDelay
			break
		}
		base *= 2
	}
	if base >= cfg.MaxDelay {
		return cfg.MaxDelay
	}
	span := int64(base)
	if span <= 0 {
		return 0
	}
	delay := base + time.Duration(jitter.Int64N(span))
	if delay > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return delay
}

func defaultSleep(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func defaultAfterReader(failure *ai.Failure) (time.Duration, bool) {
	return failure.RetryAfter()
}
