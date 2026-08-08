// AI-37 remediation (sdd-verify C-1) — R-AOB-004/S-AOB-015, R-AOB-003/S-AOB-009.
//
// sdd-verify found retry.count recorded on every exit of the retry
// mechanism (stream.go, trace.go — correct by inspection) but effectively
// unasserted: the shipped suite asserted only the unretried-success row,
// and only that it equals 0. Two whole-module overlay defeats each left
// the pre-remediation suite green with zero failures: retry.count
// hard-wired to 0 inside recordRetryCount, and the recordRetryCount call
// deleted from endSpanPreHandover entirely (dropping the attribute from
// both of Stream's own pre-handover failure exits). This file is the
// missing falsifier: the fail-then-succeed fixture S-AOB-015 names, plus
// a five-row table asserting the exact retry count on every row — not
// merely that the key is present.
//
// This is test-only work: production code (stream.go, trace.go,
// retry.go) is unchanged by this file. The implementation was already
// correct by inspection; only its proof was missing.
//
// # Why this table also discharges S-AOB-009's outstanding row
//
// sdd-verify: "No traced retried-then-*succeeded* row exists" (R-AOB-003
// PARTIAL). retried_twice_then_success below drives a transport attempt
// retried twice before succeeding, under the recording tracer, and
// asserts exactly one span started and ended exactly once — S-AOB-009's
// own "still exactly one span exists" clause. tracetest.Span records no
// start/end timestamp (sdd-verify W-5), so the sibling "recorded duration
// encloses every attempt" clause remains unassertable with the shipped
// recorder and is not attempted here.
package openaicompat_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai/internal/retry"
)

// ai37RetryTwiceThenSucceedTranscript is a minimal well-formed completion
// — one terminal chunk (finish reason "stop"), then the sentinel — so the
// retried_twice_then_success row reaches a real completion, not merely a
// successful handshake.
const ai37RetryTwiceThenSucceedTranscript = "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

// ai37RetryTwiceThenSucceedServer answers the first two requests
// retryably (503, the same status ai37AlwaysStatusServer's own
// budget-exhausted row uses) and every request from the third on with a
// streaming success response — the fail-then-succeed fixture S-AOB-015
// names and this remediation was missing.
func ai37RetryTwiceThenSucceedServer(t *testing.T) *httptest.Server {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"type":"scripted_failure"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, ai37RetryTwiceThenSucceedTranscript)
	}))
	t.Cleanup(server.Close)
	return server
}

// ai37AssertRetryCount asserts provider recorded exactly one span whose
// retry.count attribute is present and equals want — the per-row
// assertion S-AOB-015 requires on every exit, not merely on the unretried
// row the pre-remediation suite covered.
func ai37AssertRetryCount(t *testing.T, provider *tracetest.Provider, want int64) {
	t.Helper()
	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	v, ok := ai37AttrByKey(spans[0].Attributes(), "retry.count")
	if !ok {
		t.Fatal("retry.count absent, want present on every exit of the retry mechanism (R-AOB-004, S-AOB-015)")
	}
	if v.AsInt64() != want {
		t.Errorf("retry.count = %d, want %d", v.AsInt64(), want)
	}
}

// TestAI37_RetryCount_PresentOnEveryExit drives S-AOB-015's own five named
// retry outcomes — an unretried success, a request retried twice then
// succeeding, a non-retryable terminal failure, a budget-exhausted
// failure, and a cancelled request — and asserts retry.count is present
// on every row with the exact number of retries that row performed.
func TestAI37_RetryCount_PresentOnEveryExit(t *testing.T) {
	t.Parallel()

	t.Run("unretried_success", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
		ai37AssertRetryCount(t, provider, 0)
	})

	t.Run("retried_twice_then_success", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37RetryTwiceThenSucceedServer(t)
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		// S-AOB-009: still exactly one span exists across the retried
		// transport attempt, and it ended exactly once.
		ai37AssertOneSpanEndedOnce(t, provider)
		ai37AssertRetryCount(t, provider, 2)
	})

	t.Run("non_retryable_terminal", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37AlwaysStatusServer(t, http.StatusUnauthorized)
		c := ai37MustClient(t, server.URL, provider)
		_, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want a non-retryable authentication failure")
		}
		ai37AssertOneSpanEndedOnce(t, provider)
		ai37AssertRetryCount(t, provider, 0)
	})

	t.Run("budget_exhausted", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37AlwaysStatusServer(t, http.StatusServiceUnavailable)
		c := ai37MustClient(t, server.URL, provider)
		_, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want a retry-budget-exhausted failure")
		}
		ai37AssertOneSpanEndedOnce(t, provider)
		ai37AssertRetryCount(t, provider, int64(retry.DefaultMaxAttempts))
	})

	t.Run("cancelled", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// firstAttemptDone fires the instant the first (503, retryable)
		// response has been fully written server-side. Cancelling from
		// there lands the client inside retry.Loop's own attempt-1
		// iteration — either its ctx.Err() check or its SleepFunc select
		// — before a second attempt is ever issued, so retries is
		// deterministically 0: both of those exits return attempt-1 with
		// attempt still 1. A transport-level cancellation racing the read
		// of the first response is covered too — retry.Loop's own
		// wrapTransportError path categorizes it Unavailable (retryable),
		// which then hits the very same ctx.Err() check with attempt
		// still 1.
		firstAttemptDone := make(chan struct{}, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"type":"scripted_failure"}}`)
			select {
			case firstAttemptDone <- struct{}{}:
			default:
			}
		}))
		t.Cleanup(server.Close)
		c := ai37MustClient(t, server.URL, provider)
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-firstAttemptDone
			cancel()
		}()
		_, err := c.Stream(ctx, ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want the last observed failure")
		}
		ai37AssertOneSpanEndedOnce(t, provider)
		ai37AssertRetryCount(t, provider, 0)
	})
}
