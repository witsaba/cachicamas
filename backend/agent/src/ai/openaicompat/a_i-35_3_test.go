// AI-35.3 — R-AIS-043 replayability and the partial-output boundary.
//
// The first three scenarios exercise the real HTTP transport and the
// adapter's pre-stream failure mapper. The fourth deliberately crosses the
// carrier handover: a semantic event is emitted before an in-band retryable
// failure, proving that the helper is no longer in the call path.
package openaicompat

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/internal/retry"
)

const ai35TrailingGarbage = "trailing-body-bytes-that-must-be-drained-before-close"

func TestAI35_3_ByteIdenticalReplay_AcrossAttempts(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		var requests atomic.Int32
		var mu sync.Mutex
		var bodies [][]byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("ReadAll(request body) error = %v", err)
			}
			mu.Lock()
			bodies = append(bodies, append([]byte(nil), body...))
			mu.Unlock()
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error"}}`)
		}))
		defer server.Close()

		client := mustClient(t, server.URL)
		request := validRequest(t)
		original, err := Translate(request)
		if err != nil {
			t.Fatalf("Translate() error = %v, want nil", err)
		}
		_, err = client.Stream(t.Context(), request)
		if err == nil {
			t.Fatal("Stream() error = nil, want exhaustion error")
		}
		if got, want := requests.Load(), int32(retry.DefaultMaxAttempts+1); got != want {
			t.Fatalf("wire requests = %d, want %d", got, want)
		}
		mu.Lock()
		deferredBodies := append([][]byte(nil), bodies...)
		mu.Unlock()
		if len(deferredBodies) != retry.DefaultMaxAttempts+1 {
			t.Fatalf("recorded bodies = %d, want %d", len(deferredBodies), retry.DefaultMaxAttempts+1)
		}
		for i, body := range deferredBodies {
			if !bytes.Equal(body, original) {
				t.Fatalf("attempt %d body = %q, want original %q", i+1, body, original)
			}
			if !bytes.Equal(body, deferredBodies[0]) {
				t.Fatalf("attempt %d body differs from attempt 1", i+1)
			}
		}
	})
}

func TestAI35_3_AttemptCountAndFinalCause_ReachableFromErrorChain(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error"}}`)
		}))
		defer server.Close()

		_, err := mustClient(t, server.URL).Stream(t.Context(), validRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want exhaustion error")
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
		if failure.Category() != ai.FailureCategoryRateLimit || failure.Delivery() != ai.DeliveryPreStream {
			t.Fatalf("final failure = category=%v delivery=%v, want rate-limit/pre-stream", failure.Category(), failure.Delivery())
		}
	})
}

func TestAI35_3_PerAttemptDrain_StatusPath(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		var requests atomic.Int32
		var connections atomic.Int32
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error"}}`+ai35TrailingGarbage)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}))
		server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
			if state == http.StateNew {
				connections.Add(1)
			}
		}
		server.Start()
		defer server.Close()

		_, err := mustClient(t, server.URL).Stream(t.Context(), validRequest(t))
		if err == nil {
			t.Fatal("Stream() error = nil, want exhaustion error")
		}
		if got, want := requests.Load(), int32(retry.DefaultMaxAttempts+1); got != want {
			t.Fatalf("wire requests = %d, want %d", got, want)
		}
		if got := connections.Load(); got != 1 {
			t.Fatalf("TCP connections = %d, want 1 after per-attempt drain", got)
		}
	})
}

func TestAI35_3_PartialOutputBoundaryMarker_AfterHandoverNoRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-ai35\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-ai35\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"error\":{\"type\":\"rate_limit_error\",\"message\":\"late\"}}\n\n")
	}))
	defer server.Close()

	ch, err := mustClient(t, server.URL).Stream(t.Context(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want carrier handover", err)
	}
	events := drainAll(t, ch)
	if got := requests.Load(); got != 1 {
		t.Fatalf("wire requests = %d, want 1 after partial output", got)
	}
	var failure *ai.Failure
	for _, event := range events {
		if candidate, ok := event.ErrorPayload(); ok {
			failure = candidate
			break
		}
	}
	if failure == nil {
		t.Fatal("stream emitted no terminal typed failure after partial output")
	}
	if !failure.PartialOutput() {
		t.Fatal("terminal failure PartialOutput() = false, want true")
	}
	if failure.Delivery() != ai.DeliveryMidStream {
		t.Fatalf("terminal failure Delivery() = %v, want mid-stream", failure.Delivery())
	}
}
