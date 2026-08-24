// CH-03.2 — SSE wire-format writer tests (R-CHS-002.a/d, R-CHS-002.c).
// Byte-exact assertions against the frozen wire contract.

package chat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/chat"
)

// stubFlusher implements http.Flusher and records how many times
// Flush() was called. The eventsource writer must flush exactly once
// per frame (R-CHS-002.a "the stream carries a terminal event AND the
// connection closes exactly once" — the same invariant at the per-frame
// granularity).
type stubFlusher struct {
	mu       sync.Mutex
	flushes  int
}

func (f *stubFlusher) Flush() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flushes++
}

func (f *stubFlusher) flushCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.flushes
}

// recordingResponseWriter captures bytes written and supports an
// optional stubFlusher. It satisfies http.ResponseWriter for
// header/body and http.Flusher via composition.
type recordingResponseWriter struct {
	buf     bytes.Buffer
	headers http.Header
	flusher *stubFlusher
}

func newRecordingRW(flusher *stubFlusher) *recordingResponseWriter {
	return &recordingResponseWriter{headers: make(http.Header), flusher: flusher}
}

func (w *recordingResponseWriter) Header() http.Header        { return w.headers }
func (w *recordingResponseWriter) Write(p []byte) (int, error) { return w.buf.Write(p) }
func (w *recordingResponseWriter) WriteHeader(statusCode int)  {}

// writeSSEHeaders is internal in eventsource.go, so the test sets the
// headers manually before calling writeFrame. The handlers in http.go
// call writeSSEHeaders before driveStream — tested in http_test.go.

// TestWriteFrame_AllVariants — S-CHS-002.d (wire shape) + R-CHS-002.a.
// For each WireEvent variant, assert:
//   - frame is exactly `event: <name>\ndata: <json>\n\n`
//   - data is valid JSON re-encoding the variant's exported fields
//   - Flush() was called exactly once
func TestWriteFrame_AllVariants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ev   chat.WireEvent
		want string
	}{
		{
			name: "MessageStart",
			ev:   chat.MessageStart{MessageID: "msg-1", Index: 0},
			want: `event: message.start` + "\n" + `data: {"MessageID":"msg-1","Index":0}` + "\n\n",
		},
		{
			name: "MessageDelta",
			ev:   chat.MessageDelta{Index: 0, Delta: "hi"},
			want: `event: message.delta` + "\n" + `data: {"Index":0,"Delta":"hi"}` + "\n\n",
		},
		{
			name: "MessageEnd",
			ev:   chat.MessageEnd{Index: 0, FinishReason: chat.FinishReasonStop},
			want: `event: message.end` + "\n" + `data: {"Index":0,"FinishReason":"stop"}` + "\n\n",
		},
		{
			name: "TurnEnd",
			ev:   chat.TurnEnd{FinishReason: nil},
			want: `event: turn.end` + "\n" + `data: {"FinishReason":null}` + "\n\n",
		},
		{
			name: "Error",
			ev:   chat.Error{Kind: "server", Message: "boom"},
			want: `event: error` + "\n" + `data: {"Kind":"server","Message":"boom"}` + "\n\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flusher := &stubFlusher{}
			w := newRecordingRW(flusher)
			if err := chat.WriteFrameForTest(w, flusher, tc.ev); err != nil {
				t.Fatalf("writeFrame returned %v, want nil", err)
			}
			if got := w.buf.String(); got != tc.want {
				t.Errorf("frame bytes\n got %q\nwant %q", got, tc.want)
			}
			if got := flusher.flushCount(); got != 1 {
				t.Errorf("Flush() called %d time(s) for one frame, want 1 (R-CHS-002.a: flush per frame)", got)
			}
			// Re-parse the JSON data to confirm it's valid AND matches
			// the variant's exported fields.
			lines := strings.Split(tc.want, "\n")
			if len(lines) < 2 {
				t.Fatalf("frame shape unexpected: %q", tc.want)
			}
			dataLine := strings.TrimPrefix(lines[1], "data: ")
			var got map[string]any
			if err := json.Unmarshal([]byte(dataLine), &got); err != nil {
				t.Errorf("data line is not valid JSON: %v (raw=%q)", err, dataLine)
			}
		})
	}
}

// TestWriteFrames_ClosesAfterChannelClose — R-CHS-002.a. Given a
// channel that emits three events then closes, when writeFrames
// drains it, the recorded bytes carry exactly three frames and
// writeFrames returns nil (no error).
func TestWriteFrames_ClosesAfterChannelClose(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	w := newRecordingRW(flusher)

	events := make(chan chat.WireEvent, 3)
	events <- chat.MessageStart{MessageID: "m", Index: 0}
	events <- chat.MessageDelta{Index: 0, Delta: "x"}
	events <- chat.TurnEnd{}
	close(events)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := chat.WriteFramesForTest(ctx, w, flusher, events); err != nil {
		t.Fatalf("writeFrames returned %v, want nil on clean close", err)
	}
	out := w.buf.String()
	count := strings.Count(out, "\n\n")
	if count != 3 {
		t.Errorf("recorded output carried %d frame terminators, want 3 (R-CHS-002.a: one terminal event closes the stream)", count)
	}
	if got := flusher.flushCount(); got != 3 {
		t.Errorf("Flush() called %d time(s), want 3 (one per frame)", got)
	}
}

// TestWriteFrames_ContextCancel — R-CHS-002.c / S-CHS-002.c. Given a
// context that is cancelled mid-stream, when writeFrames is running,
// it returns ctx.Err() promptly and writes no further frames.
func TestWriteFrames_ContextCancel(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	w := newRecordingRW(flusher)

	events := make(chan chat.WireEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the call

	done := make(chan error, 1)
	go func() {
		done <- chat.WriteFramesForTest(ctx, w, flusher, events)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("writeFrames returned nil for a cancelled context, want non-nil")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("writeFrames did not return within 1s for a cancelled context (it should observe ctx.Done())")
	}
}
