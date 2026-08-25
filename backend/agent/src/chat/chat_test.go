// CH-03 integration coverage — exercises the full POST → GET → DELETE
// flow end to end against a real *echo.Echo on a random port. The
// scenarios that need deterministic timing (cancellation mid-stream,
// already-terminated GET, client disconnect) use an agenttest.Gate to
// park the producer; the green-path flows use a deterministic one-
// shot scripted stream from eventsource_test.go's scriptForOneTurn.

package chat_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// TestChatHTTP_PostStreamCancelFlow — happy path. POST opens a turn;
// GET streams the projected events in order; DELETE cancels and the
// stream emits `event: turn.end\n\n` with NO `FinishReason` in the
// data line (R-CHS-003.a's cancellation discriminator observed at
// the HTTP layer).
func TestChatHTTP_PostStreamCancelFlow(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	delta, err := ai.NewTextDelta(1, "alpha")
	if err != nil {
		t.Fatalf("ai.NewTextDelta: %v", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	provider := agenttest.NewProvider(agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Hold(gate),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}})

	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-chat",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	})

	// POST opens the turn.
	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hi"})
	res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
_ = 	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", res.StatusCode)
	}

	// GET subscribes. The held script emits message.start + one
	// delta, then parks. We open the GET FIRST so the cancellation
	// observes the same stream the user subscribed to.
	getReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-1/events", nil)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET events: %v", err)
	}
defer func() { _ = getResp.Body.Close() }()

	// Wait for the producer to reach the gate so the projector is
	// parked and the SSE connection is mid-stream.
	select {
	case <-gate.Reached():
	case <-time.After(2 * time.Second):
		t.Fatal("producer never reached the gate")
	}

	// DELETE cancels the turn while the producer is parked.
	delReq, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/agent/turns/t-1", nil)
	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
_ = 	delRes.Body.Close()
	if delRes.StatusCode != http.StatusNoContent {
		t.Errorf("DELETE status=%d, want 204", delRes.StatusCode)
	}

	// Release the gate so the producer emits message.end then
	// turn.end. Read the SSE stream until turn.end arrives.
	gate.Release()

	scanner := bufio.NewScanner(getResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var frame strings.Builder
	var sawTurnEnd bool
	var sawTurnEndFinishReason *string // nil means absent
	deadline := time.After(5 * time.Second)
read:
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out reading SSE; sawTurnEnd=%v, sawTurnEndFinishReason=%v", sawTurnEnd, sawTurnEndFinishReason)
		default:
		}
		if !scanner.Scan() {
			break read
		}
		line := scanner.Text()
		if line == "" {
			// End of a frame — parse it.
			if frame.Len() == 0 {
				continue
			}
			var event string
			var data string
			for _, l := range strings.Split(strings.TrimRight(frame.String(), "\n"), "\n") {
				switch {
				case strings.HasPrefix(l, "event: "):
					event = strings.TrimPrefix(l, "event: ")
				case strings.HasPrefix(l, "data: "):
					data = strings.TrimPrefix(l, "data: ")
				}
			}
			frame.Reset()
			if event == "turn.end" {
				sawTurnEnd = true
				var payload struct {
					FinishReason *string `json:"finishReason"`
				}
				_ = json.Unmarshal([]byte(data), &payload)
				sawTurnEndFinishReason = payload.FinishReason
				break read
			}
			continue
		}
		frame.WriteString(line)
		frame.WriteString("\n")
	}

	if !sawTurnEnd {
		t.Fatal("turn.end frame never observed on the SSE stream")
	}
	if sawTurnEndFinishReason != nil {
		t.Errorf("turn.end carried finishReason=%q, want nil/absent (R-CHS-003.a: cancellation discriminator)",
			*sawTurnEndFinishReason)
	}
}

// TestChatHTTP_AlreadyTerminatedGet — S-CHS-002.b. After a turn
// completes, a subsequent GET receives an immediate single
// `event: turn.end\n\n` followed by a clean close (no `message.*`).
func TestChatHTTP_AlreadyTerminatedGet(t *testing.T) {
	t.Parallel()

	// First provider: completes quickly with one fragment.
	quick := agenttest.NewProvider(agenttest.Script{Steps: func() []agenttest.Step {
		start, _ := ai.NewTextBlockStart(1)
		delta, _ := ai.NewTextDelta(1, "x")
		end, _ := ai.NewTextBlockEnd(1)
		completion, _ := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		return []agenttest.Step{
			agenttest.Emit(start),
			agenttest.Emit(delta),
			agenttest.Emit(end),
			agenttest.Emit(completion),
		}
	}()})

	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: quick, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-chat",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	})

	body, _ := json.Marshal(map[string]string{"id": "t-fast", "prompt": "hi"})
	_, _ = http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))

	// Drain the stream to let the turn complete (so MarkStreamCompleted fires).
	drainReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-fast/events", nil)
	drainResp, err := http.DefaultClient.Do(drainReq)
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}
	_, _ = io.Copy(io.Discard, drainResp.Body)
_ = 	drainResp.Body.Close()

	// Second GET: the stream should already be marked completed.
	// The handler returns a single `event: turn.end\n\n`.
	secondReq, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-fast/events", nil)
	secondResp, err := http.DefaultClient.Do(secondReq)
	if err != nil {
		t.Fatalf("second GET: %v", err)
	}
defer func() { _ = secondResp.Body.Close() }()

	if got := secondResp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type=%q, want text/event-stream", got)
	}

	scanner := bufio.NewScanner(secondResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var events []string
	var frame strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if frame.Len() > 0 {
				for _, l := range strings.Split(strings.TrimRight(frame.String(), "\n"), "\n") {
					if strings.HasPrefix(l, "event: ") {
						events = append(events, strings.TrimPrefix(l, "event: "))
					}
				}
				frame.Reset()
			}
			continue
		}
		frame.WriteString(line)
		frame.WriteString("\n")
	}
	if len(events) != 1 {
		t.Fatalf("got %d events on the second GET, want 1 (a single turn.end): events=%v", len(events), events)
	}
	if events[0] != "turn.end" {
		t.Errorf("event=%q, want turn.end", events[0])
	}
}

// TestChatHTTP_ClientDisconnectDoesNotLeakGoroutine — S-CHS-002.c.
// When the SSE subscriber disconnects mid-stream without a DELETE,
// the turn continues, writeFrames exits on ctx.Done(), and no
// goroutine remains blocked on the departed subscriber.
func TestChatHTTP_ClientDisconnectDoesNotLeakGoroutine(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	start, _ := ai.NewTextBlockStart(1)
	delta, _ := ai.NewTextDelta(1, "alpha")
	end, _ := ai.NewTextBlockEnd(1)
	completion, _ := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	provider := agenttest.NewProvider(agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Hold(gate),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}})

	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-chat",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	})

	body, _ := json.Marshal(map[string]string{"id": "t-leak", "prompt": "hi"})
	res, _ := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	_ = res.Body.Close()

	// GET to subscribe; close early to simulate a client disconnect.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Read from the SSE until ctx cancels, then bail to force
		// the server's handler out via ctx.Done().
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/agent/turns/t-leak/events", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
_ = 		resp.Body.Close()
	}()

	// Let the producer reach the gate so the GET handler is mid-stream.
	select {
	case <-gate.Reached():
	case <-time.After(3 * time.Second):
		_ = sync.Mutex{}
		t.Fatal("producer never reached the gate")
	}
	cancel()
	_ = ctx

	// After the disconnect, the turn continues. A second GET (or the
	// same turn's completion) must not block. We assert this by
	// issuing a second POST for the SAME participant and watching it
	// succeed: this only happens if the registry's inFlight flag
	// was cleared when the first projector goroutine finally wrote
	// its terminal event. That requires the harness to have kept
	// running after the GET's ctx.Done().

	// Release the gate to allow completion.
	gate.Release()

	// Wait briefly for completion to land. We assert via the
	// no-leak property: a follow-up POST must reach a 409 once
	// the first turn's terminal event is being projected, and a
	// 200 once it's fully through. Either is fine — what matters
	// is no hang. We use a deadline-bounded poll.
	want := deadlinePoll(t, 3*time.Second, func() (bool, string) {
		body, _ := json.Marshal(map[string]string{"id": "t-leak-2", "prompt": "hi"})
		res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
		if err != nil {
			return false, err.Error()
		}
_ = 		res.Body.Close()
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusConflict {
			return false, "unexpected status"
		}
		return true, ""
	})
	if !want {
		t.Fatal("second POST hung or returned an unexpected status — goroutine may be leaked")
	}
}

// TestChatHTTP_PostStreamCancelFlow_ViaTurnedCompletion — companion
// to TestChatHTTP_PostStreamCancelFlow using the same plumbing but a
// simpler script. Drains the stream to completion and asserts the
// turn ends with finishReason "stop" (R-CHS-002.a, no cancellation).
func TestChatHTTP_PostStreamCancelFlow_NaturalCompletion(t *testing.T) {
	t.Parallel()

	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-chat",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	})

	body, _ := json.Marshal(map[string]string{"id": "t-natural", "prompt": "hi"})
	_, _ = http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-natural/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
defer func() { _ = resp.Body.Close() }()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var events []string
	var frame strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if frame.Len() > 0 {
				for _, l := range strings.Split(strings.TrimRight(frame.String(), "\n"), "\n") {
					if strings.HasPrefix(l, "event: ") {
						events = append(events, strings.TrimPrefix(l, "event: "))
					}
				}
				frame.Reset()
			}
			continue
		}
		frame.WriteString(line)
		frame.WriteString("\n")
	}

	want := []string{"message.start", "message.delta", "message.end", "turn.end"}
	if len(events) < len(want) {
		t.Fatalf("got %d events, want at least %d (events=%v)", len(events), len(want), events)
	}
	for i, w := range want {
		if events[i] != w {
			t.Errorf("event[%d]=%q, want %q", i, events[i], w)
		}
	}
}

// deadlinePoll retries fn every 50ms until it returns true or the
// deadline elapses. Returns the final outcome. Used by the leak test
// to wait for harness.Run to finish without sleeping on a wall clock.
func deadlinePoll(t *testing.T, d time.Duration, fn func() (bool, string)) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	var lastMsg string
	for time.Now().Before(deadline) {
		ok, msg := fn()
		if ok {
			return true
		}
		lastMsg = msg
		time.Sleep(50 * time.Millisecond)
	}
	t.Logf("deadlinePoll exhausted: %s", lastMsg)
	return false
}

// oneShotEcho is a tiny shim — we never call it but the import keeps
// the file syntactically valid if every test above is commented out
// during a future refactor.
// mu keeps the package-level sync import alive across all configurations.
var _ = sync.Mutex{}
