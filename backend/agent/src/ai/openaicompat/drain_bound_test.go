// The drain-before-close is byte-bounded (AI-33.5 hardening).
//
// run()'s defer chain drains resp.Body before closing it so the transport
// can return the keep-alive connection to its pool (R-AIS-033). An
// UNBOUNDED drain turns that courtesy into a hang: a server that keeps
// writing after the terminal sentinel — on a stream whose context is
// never cancelled, because the consumer believes the stream ended
// normally — holds the producer goroutine in io.Copy indefinitely, and
// close(out) never runs. The drain therefore consumes at most
// drainBodyLimit bytes; past that the body is closed undrained and the
// transport discards the connection instead of reusing it — a bounded
// close is worth strictly more than a reused connection.
//
// The silent-stall sibling (a server that stops writing but never closes)
// is NOT closed by a byte bound and remains governed by the AI-02
// contract: the caller owns ctx and cancellation is the caller's bound.
package openaicompat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

func TestRun_ServerKeepsWritingAfterDone_CloseIsByteBoundedNotHung(t *testing.T) {
	transcript := "" +
		"data: {\"id\":\"chatcmpl-drain\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-drain\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-drain\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"

	// After the terminal sentinel the handler floods SSE keep-alive
	// comments until the client tears the connection down. An unbounded
	// drain never sees EOF here; a byte-bounded one returns quickly.
	junk := make([]byte, 4096)
	for i := range junk {
		junk[i] = ':'
	}
	junk[len(junk)-2], junk[len(junk)-1] = '\n', '\n'

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush()
		}
		for {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			if _, err := w.Write(junk); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	c := mustClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.Stream(ctx, validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	// The consumer never cancels: from its vantage the stream completed
	// normally at [DONE]. The channel must still close within the drain
	// bound — DrainAndRecord fails loudly if close(out) is hung behind
	// an unbounded body drain.
	rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout+time.Second)
	events := rec.Events()
	last := events[len(events)-1]
	if _, ok := last.Completion(); !ok {
		t.Errorf("last event kind = %v, want the Completion terminal", last.Kind())
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("ai.CheckStream violation = %v, want none", report.Violation())
	}
}
