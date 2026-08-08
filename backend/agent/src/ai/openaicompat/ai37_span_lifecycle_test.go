// AI-37 (WU-4, span seam) — R-AOB-003: one span per logical request,
// started before the first transport attempt, ended exactly once on every
// terminal path. This file drives the 15-path terminal table design.md's
// AD-5 names — 11 post-handover (run's own terminal returns) plus 4
// pre-handover (Stream's own two failure exits, further split by cause) —
// against tracetest.Provider, asserting exactly one span started and
// AssertAllEndedOnce() reports nil for every row.
package openaicompat_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/internal/retry"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// ai37DrainTimeout bounds every drain in this file except the
// frame-too-large row (below), which needs its own, longer bound.
const ai37DrainTimeout = 5 * time.Second

// ai37HeavyDrainTimeout bounds the frame_error_frame_too_large row: that
// fixture accumulates over DefaultMaxFrameBytes (8 MiB) through 32 KiB
// reads before the decoder trips its cap, and under -race the byte
// tracking overhead alone measured ~8s (isolated probe, this milestone's
// own apply-progress evidence) — comfortably under 5s without -race, not
// under it.
const ai37HeavyDrainTimeout = 20 * time.Second

// ai37MustClient builds a *Client whose Config.TracerProvider is provider
// — the RED-defining call for this file: Config carries no such field
// before this work unit lands.
func ai37MustClient(t *testing.T, endpoint string, provider *tracetest.Provider) *openaicompat.Client {
	t.Helper()
	c, err := openaicompat.New(openaicompat.Config{
		Endpoint:       endpoint,
		Credential:     openaicompat.NewCredential("ai37-span-lifecycle-token"),
		TracerProvider: provider,
	})
	if err != nil {
		t.Fatalf("openaicompat.New() error = %v, want nil", err)
	}
	return c
}

// ai37ValidRequest builds a minimal valid ai.Request.
func ai37ValidRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	req, err := ai.NewRequest("ai37-model", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}
	return req
}

// ai37DrainAll drains ch to close, discarding events, bounded by
// ai37DrainTimeout.
func ai37DrainAll(t *testing.T, ch <-chan ai.Event) []ai.Event {
	t.Helper()
	return ai37DrainAllWithTimeout(t, ch, ai37DrainTimeout)
}

// ai37DrainAllWithTimeout is ai37DrainAll with a caller-supplied bound —
// the frame_error_frame_too_large row needs one longer than every other
// row in this file's table.
func ai37DrainAllWithTimeout(t *testing.T, ch <-chan ai.Event, timeout time.Duration) []ai.Event {
	t.Helper()
	var events []ai.Event
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline:
			t.Fatalf("ai37DrainAllWithTimeout: timed out after %s with %d event(s) so far", timeout, len(events))
			return events
		}
	}
}

// ai37AssertOneSpanEndedOnce is the shared post-condition every row in
// this file's 15-path table asserts: exactly one span was started
// (R-AOB-003, and the ordering proof — a pre-handover failure still
// starts a span before retry.Loop ever runs), and it ended exactly once.
func ai37AssertOneSpanEndedOnce(t *testing.T, provider *tracetest.Provider) {
	t.Helper()
	if got := provider.Started(); got != 1 {
		t.Errorf("provider.Started() = %d, want exactly 1 (R-AOB-003)", got)
	}
	if err := provider.AssertAllEndedOnce(); err != nil {
		t.Errorf("AssertAllEndedOnce() = %v, want nil", err)
	}
}

// ai37SSEServer returns an httptest.Server that writes transcript verbatim
// as the whole SSE response body.
func ai37SSEServer(t *testing.T, transcript string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
	}))
	t.Cleanup(server.Close)
	return server
}

// ai37SlowSSEServer writes first, flushes, then blocks on release or
// request-context-done — the mid-stream-cancellation fixture shape
// (stream_test.go's own slowSSEServer, restated here to keep this file
// self-contained).
func ai37SlowSSEServer(t *testing.T, first string) (server *httptest.Server, release chan struct{}) {
	t.Helper()
	release = make(chan struct{})
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, first)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(release); server.Close() })
	return server, release
}

// ai37AbruptCloseServer writes first, flushes, then panics with
// http.ErrAbortHandler — net/http's own documented sentinel for
// aborting a response without a stack-trace log, which the client
// observes as a read error that is neither io.EOF nor a context
// cancellation (the generic "read error" terminal path, stream.go's
// final emitFailure call).
func ai37AbruptCloseServer(t *testing.T, first string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, first)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		panic(http.ErrAbortHandler)
	}))
	t.Cleanup(server.Close)
	return server
}

// ai37AlwaysStatusServer answers every request with status, forever.
func ai37AlwaysStatusServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"error":{"type":"scripted_failure"}}`)
	}))
	t.Cleanup(server.Close)
	return server
}

// TestAI37_SpanLifecycle drives the 15-path terminal table (design.md
// AD-5): 11 post-handover paths through run's own terminal returns, plus
// 4 pre-handover paths through Stream's own two failure exits. Every row
// asserts ai37AssertOneSpanEndedOnce.
func TestAI37_SpanLifecycle(t *testing.T) {
	t.Parallel()

	// --- Post-handover (run's own 11 terminal paths) ---

	t.Run("success_done", func(t *testing.T) {
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
	})

	t.Run("completion_build_failure", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// No prior chunk ever set a finish reason: buildCompletion fails
		// at the sentinel (design.md D9).
		server := ai37SSEServer(t, "data: [DONE]\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("in_band_error", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37SSEServer(t, "data: {\"error\":{\"type\":\"invalid_request_error\",\"message\":\"bad request\"}}\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("malformed_chunk_json", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37SSEServer(t, "data: {not-valid-json\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("applychunk_failure_malformed_identity", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// Empty id on the first chunk: errMalformedIdentity via
		// applyChunk (stream_test.go's own proven fixture shape).
		server := ai37SSEServer(t, "data: {\"id\":\"\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("emit_race_cancellation", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := c.Stream(ctx, ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		// Never drain yet: the carrier is unbuffered (streamCarrierBuffer
		// == 0), so run's first emit() blocks with nobody reading. A
		// short delay lets the producer reach that blocking send before
		// cancellation — the AI-32.3 emit-race path (stream.go's own
		// !emit(...) branch inside the events loop) — without depending
		// on any exported synchronization point.
		time.Sleep(20 * time.Millisecond)
		cancel()
		// Now drain: the bounded-wait recovery sends inside the
		// emit-race path need a reader to succeed quickly.
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("feed_error_frame_too_large", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// An in-progress data line exceeding DefaultMaxFrameBytes (8 MiB)
		// with no terminating blank line: Feed itself returns
		// ErrFrameTooLarge while still reading, not at Finish.
		huge := "data: " + strings.Repeat("x", 8*1024*1024+64)
		server := ai37SSEServer(t, huge)
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAllWithTimeout(t, ch, ai37HeavyDrainTimeout)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("eof_finish_error_truncated", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// An incomplete data line, no closing blank line, then the
		// connection closes cleanly: decoder.Finish() reports
		// ErrTruncated at transport EOF.
		server := ai37SSEServer(t, "data: {\"id\":\"c\"")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("eof_incomplete_no_sentinel", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// Clean, complete SSE framing, but [DONE] is never observed
		// before the connection closes.
		server := ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("mid_stream_ctx_cancellation", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server, release := ai37SlowSSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
		_ = release
		c := ai37MustClient(t, server.URL, provider)
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := c.Stream(ctx, ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		<-ch // the one event the slow server delivers before blocking
		cancel()
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("read_error_abrupt_close", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37AbruptCloseServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
		c := ai37MustClient(t, server.URL, provider)
		ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		ai37DrainAll(t, ch)
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	// --- Pre-handover (Stream's own two failure exits, four causes) ---

	t.Run("pre_handover_retry_exhausted", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37AlwaysStatusServer(t, http.StatusServiceUnavailable)
		c := ai37MustClient(t, server.URL, provider)
		_, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want a retry-budget-exhausted failure")
		}
		var report *retry.AttemptReport
		if !errors.As(err, &report) {
			t.Fatalf("errors.As(err, *retry.AttemptReport) = false; err = %v", err)
		}
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("pre_handover_non_retryable", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := ai37AlwaysStatusServer(t, http.StatusUnauthorized)
		c := ai37MustClient(t, server.URL, provider)
		_, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want a non-retryable authentication failure")
		}
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("errors.As(err, *ai.Failure) = false; err = %v", err)
		}
		if failure.Category() != ai.FailureCategoryAuthentication {
			t.Errorf("Category() = %v, want FailureCategoryAuthentication", failure.Category())
		}
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("pre_handover_cancelled_in_loop", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		// Judgment Day remediation: this row previously kept an
		// unsynchronized `var hits int` incremented inside the handler
		// (potentially served across retry attempts by different
		// goroutines) and never read anywhere -- both racy under -race
		// and dead. Deleted rather than made atomic: this row's own
		// sibling in ai37_noop_equivalence_test.go's no-tracer table
		// never carried a counter for the identical fixture shape, and
		// nothing in this row's assertions needs a request count.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"type":"scripted_failure"}}`)
		}))
		t.Cleanup(server.Close)
		c := ai37MustClient(t, server.URL, provider)
		ctx, cancel := context.WithCancel(context.Background())
		// The server always answers retryably; cancelling shortly after
		// Stream starts lands inside retry.Loop's own ctx.Err() check
		// between attempts, well before its real backoff budget would
		// otherwise exhaust (avoiding a multi-second real sleep in this
		// test).
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		_, err := c.Stream(ctx, ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want the last observed failure")
		}
		ai37AssertOneSpanEndedOnce(t, provider)
	})

	t.Run("pre_handover_content_type_refusal", func(t *testing.T) {
		t.Parallel()
		provider := tracetest.NewProvider()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"ok":true}`)
		}))
		t.Cleanup(server.Close)
		c := ai37MustClient(t, server.URL, provider)
		_, err := c.Stream(context.Background(), ai37ValidRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want a non-streaming content-type refusal")
		}
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("errors.As(err, *ai.Failure) = false; err = %v", err)
		}
		if failure.Category() != ai.FailureCategoryMalformedResponse {
			t.Errorf("Category() = %v, want FailureCategoryMalformedResponse", failure.Category())
		}
		ai37AssertOneSpanEndedOnce(t, provider)
	})
}

// TestAI37_TerminalEvents_AlwaysStamped is a Judgment Day remediation for
// a defect this milestone did not introduce but did carry forward
// unchanged (confirmed against origin/main): the mid-stream
// ctx-cancellation exit's own TextBlockEnd/ErrorEvent recovery sends the
// raw ai.Event value out <- end / out <- errEv directly onto the
// carrier, bypassing stamper.Stamp -- so a terminal event that follows
// earlier stamped events (sequence 1..n) itself carries sequence 0,
// breaking R-AEE-007/R-AEE-008's contiguous per-stream sequence. Every
// sibling terminal-send path in this file stamps first (the events
// loop's own recovery; emitFailure, whose own comment says the sequence
// assignment "is not bypassed, ever (R-AEE-008)"). This reproduces the
// same mid-stream-cancellation shape the "mid_stream_ctx_cancellation"
// row above drives, and additionally asserts every drained event's
// Sequence() is contiguous from 1.
func TestAI37_TerminalEvents_AlwaysStamped(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	server, release := ai37SlowSSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
	_ = release
	c := ai37MustClient(t, server.URL, provider)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Stream(ctx, ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	first := <-ch // the one event the slow server delivers before blocking
	cancel()
	rest := ai37DrainAll(t, ch)

	events := append([]ai.Event{first}, rest...)
	if len(events) < 2 {
		t.Fatalf("drained %d event(s), want at least 2 (the pre-cancellation event plus a terminal error) -- the sequence check below needs a stamped predecessor to compare against", len(events))
	}
	for i, ev := range events {
		want := ai.Sequence(i + 1)
		if ev.Sequence() != want {
			t.Errorf("event %d (kind %v) has Sequence() = %d, want %d -- every send on this carrier MUST go through the per-stream Stamper (R-AEE-007/R-AEE-008)", i, ev.Kind(), ev.Sequence(), want)
		}
	}
}
