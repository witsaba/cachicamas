// AI-35.1 — R-AIS-041 retry predicate at the pre-stream boundary.
//
// These tests drive the real OpenAI-compatible producer through a scripted
// HTTP server. The request count is the observable boundary: retryable
// pre-stream failures may consume the Layer 1 budget, while terminal typed
// failures must return after the first wire request.
package openaicompat

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/internal/retry"
)

func TestAI35_1_RetryablePreStream_RetriesUpToBound(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"slow"}}`))
	}))
	t.Cleanup(server.Close)

	_, err := mustClient(t, server.URL).Stream(t.Context(), validRequest(t))
	if err == nil {
		t.Fatal("Stream() error = nil, want exhausted retry failure")
	}
	if got, want := requests.Load(), int32(retry.DefaultMaxAttempts+1); got != want {
		t.Fatalf("wire requests = %d, want %d", got, want)
	}
}

func TestAI35_1_TerminalCategory_NeverRetries(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_api_key","message":"bad"}}`))
	}))
	t.Cleanup(server.Close)

	_, err := mustClient(t, server.URL).Stream(t.Context(), validRequest(t))
	if err == nil {
		t.Fatal("Stream() error = nil, want authentication failure")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("wire requests = %d, want 1", got)
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(%T) did not find *ai.Failure", err)
	}
	if failure.Category() != ai.FailureCategoryAuthentication || failure.Retryable() {
		t.Fatalf("failure = (%v, retryable=%v), want authentication/non-retryable", failure.Category(), failure.Retryable())
	}
}

func TestAI35_1_ExhaustedBudget_ReturnsLastFailureWrappedWithAttemptCount(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"still slow"}}`))
	}))
	t.Cleanup(server.Close)

	_, err := mustClient(t, server.URL).Stream(t.Context(), validRequest(t))
	if err == nil {
		t.Fatal("Stream() error = nil, want exhausted retry failure")
	}
	var report *retry.AttemptReport
	if !errors.As(err, &report) {
		t.Fatalf("errors.As(%T) did not find *retry.AttemptReport", err)
	}
	if report.Attempts != retry.DefaultMaxAttempts+1 {
		t.Fatalf("attempts = %d, want %d", report.Attempts, retry.DefaultMaxAttempts+1)
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(%T) did not find final *ai.Failure", err)
	}
	if failure.Category() != ai.FailureCategoryRateLimit || failure.Delivery() != ai.DeliveryPreStream || failure.PartialOutput() {
		t.Fatalf("final failure = category=%v delivery=%v partial=%v, want rate-limit/pre-stream/false", failure.Category(), failure.Delivery(), failure.PartialOutput())
	}
	if got, want := requests.Load(), int32(retry.DefaultMaxAttempts+1); got != want {
		t.Fatalf("wire requests = %d, want %d", got, want)
	}
}
