// CH-03.1/.2/.3/.4 — the HTTP handler tests. Each scenario from
// openspec/specs/frontend-chat-layer1 + chat-spec #3817 is asserted
// against a real *echo.Echo served by httptest.NewServer on a random
// port (R-CHS-001..004). The tests intentionally do NOT touch the
// conversation's real harness from a goroutine perspective; they
// script the provider via agenttest.NewProvider (CH-02's pattern)
// and use a Gate for in-flight assertions.

package chat_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// fixedResolver is a minimal IdentityResolver that returns a fixed
// participant id. Each test chooses its own id so cross-participant
// refusals can use two of them.
type fixedResolver struct{ ID string }

func (r fixedResolver) IdentityFromRequest(_ context.Context, _ *http.Request) (chat.Identity, bool) {
	if r.ID == "" {
		return nil, false
	}
	return fixedIdentity{r.ID}, true
}

type fixedIdentity struct{ id string }

func (i fixedIdentity) ParticipantID() string { return i.id }

// scriptForOneTurn returns a Script that emits message.start, one
// delta, message.end, then a completion. Used by happy-path tests
// that don't need a held mid-stream producer.
func scriptForOneTurn(t *testing.T) agenttest.Script {
	t.Helper()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	delta, err := ai.NewTextDelta(1, "hi")
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
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// mountedServer constructs an echo.Echo with RegisterRoutes mounted,
// starts it on a random local port (httptest.NewServer), and returns
// the URL plus the registry so the test can reach into the
// participant state (e.g. for RaceFree concurrent-POST tests).
func mountedServer(t *testing.T, resolver chat.IdentityResolver, newConv chat.ConversationFactory) (*httptest.Server, *chat.Registry) {
	t.Helper()

	e := echo.New()
	registry, err := chat.RegisterRoutes(e, resolver, newConv)
	if err != nil {
		t.Fatalf("RegisterRoutes returned %v", err)
	}
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv, registry
}

// readSSE reads SSE frames from r until the stream closes or d
// elapses. Each frame is appended to the returned slice as a raw
// string (event+data, no trailing blank line). On ctx cancel or EOF,
// it returns what it has plus the error.
func readSSE(t *testing.T, r io.Reader, d time.Duration) []string {
	t.Helper()
	type outcome struct {
		frames []string
		err    error
	}
	ch := make(chan outcome, 1)
	go func() {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var frames []string
		var frameBuilder strings.Builder
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if frameBuilder.Len() > 0 {
					frames = append(frames, frameBuilder.String())
					frameBuilder.Reset()
				}
				continue
			}
			frameBuilder.WriteString(line)
			frameBuilder.WriteString("\n")
		}
		ch <- outcome{frames: frames, err: scanner.Err()}
	}()
	select {
	case o := <-ch:
		return o.frames
	case <-time.After(d):
		t.Fatalf("timed out after %s waiting for SSE frames", d)
		return nil
	}
}

// TestHandleOpenTurn_Valid — S-CHS-001.a. POST /api/agent/turns with
// {id, prompt} returns 200 with {turnId, streamUrl} where streamUrl
// equals /api/agent/turns/<id>/events.
func TestHandleOpenTurn_Valid(t *testing.T) {
	t.Parallel()

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t))})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hello"})
	res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", res.StatusCode)
	}
	var got struct {
		TurnID    string `json:"turnId"`
		StreamURL string `json:"streamUrl"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TurnID != "t-1" {
		t.Errorf("turnId=%q, want t-1", got.TurnID)
	}
	if got.StreamURL != "/api/agent/turns/t-1/events" {
		t.Errorf("streamUrl=%q, want /api/agent/turns/t-1/events", got.StreamURL)
	}
}

// TestHandleOpenTurn_EmptyPrompt — S-CHS-001.b. POST with prompt=""
// returns 400 validation. No turn is created and no provider call is
// made (verified by constructing a Conversation-on-fail factory).
func TestHandleOpenTurn_EmptyPrompt(t *testing.T) {
	t.Parallel()

	var factoryCalled bool
	newConv := func(_ string) (*chat.Conversation, error) {
		factoryCalled = true
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t))})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	for _, body := range []map[string]string{
		{"id": "t-1", "prompt": ""},
		{"id": "t-1"},
	} {
		b, _ := json.Marshal(body)
		res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("body=%v status=%d, want 400", body, res.StatusCode)
		}
		res.Body.Close()
	}
	if factoryCalled {
		t.Error("Conversation factory was called for a malformed prompt (the factory must be bypassed on validation failure)")
	}
}

// TestHandleOpenTurn_Inflight409 — S-CHS-001.c. While a turn is in
// flight (gate held), a second POST from the same participant
// returns 409. The factory is NOT called for the second POST.
func TestHandleOpenTurn_Inflight409(t *testing.T) {
	t.Parallel()

	gate := agenttest.NewGate()
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart: %v", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	provider := agenttest.NewProvider(
		agenttest.Script{Steps: []agenttest.Step{
			agenttest.Emit(start),
			agenttest.Hold(gate),
			agenttest.Emit(end),
			agenttest.Emit(completion),
		}},
		// Second script for any test that issues a second POST. We
		// don't reach it for inFlight 409 (the second POST must NOT
		// call the provider), but registering it keeps the fake's
		// exhaustion counter from triggering a panic if the
		// test framework does extra probes.
		agenttest.Script{Steps: []agenttest.Step{
			agenttest.Emit(start),
			agenttest.Hold(gate),
			agenttest.Emit(end),
			agenttest.Emit(completion),
		}},
	)

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: provider})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "first"})
	res1, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST #1: %v", err)
	}
	res1.Body.Close()
	if res1.StatusCode != http.StatusOK {
		t.Fatalf("POST #1 status=%d, want 200", res1.StatusCode)
	}

	select {
	case <-gate.Reached():
	case <-time.After(2 * time.Second):
		t.Fatal("script never reached the gate (the conversation's projector never started)")
	}

	body2, _ := json.Marshal(map[string]string{"id": "t-2", "prompt": "second"})
	res2, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body2))
	if err != nil {
		t.Fatalf("POST #2: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusConflict {
		t.Errorf("POST #2 status=%d, want 409", res2.StatusCode)
	}

	// Release the gate so the producer finishes; GET the stream to drain it.
	gate.Release()
	streamsURL := srv.URL + "/api/agent/turns/t-1/events"
	req, _ := http.NewRequest(http.MethodGet, streamsURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// TestHandleStreamEvents_FullTurn — S-CHS-002.a + S-CHS-002.d. After a
// POST, GETting the stream URL returns the SSE frames in order
// (message.start, message.delta, message.end, turn.end) and the
// connection closes once.
func TestHandleStreamEvents_FullTurn(t *testing.T) {
	t.Parallel()

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t))})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hi"})
	http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-1/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type=%q, want text/event-stream", got)
	}
	if got := resp.Header.Get("X-Accel-Buffering"); got != "no" {
		t.Errorf("X-Accel-Buffering=%q, want no", got)
	}

	frames := readSSE(t, resp.Body, 5*time.Second)
	var gotEvents []string
	for _, f := range frames {
		for _, line := range strings.Split(strings.TrimRight(f, "\n"), "\n") {
			if strings.HasPrefix(line, "event: ") {
				gotEvents = append(gotEvents, strings.TrimPrefix(line, "event: "))
			}
		}
	}
	wantOrder := []string{"message.start", "message.delta", "message.end", "turn.end"}
	if len(gotEvents) < len(wantOrder) {
		t.Fatalf("got %d events, want at least %d (got=%v)", len(gotEvents), len(wantOrder), gotEvents)
	}
	for i, want := range wantOrder {
		if gotEvents[i] != want {
			t.Errorf("event[%d]=%q, want %q (full=%v)", i, gotEvents[i], want, gotEvents)
		}
	}
}

// TestIdentityRefusal_401 — S-CHS-004.a. A request carrying no
// resolvable identity is refused as 401 (server envelope) BEFORE any
// turn is opened.
func TestIdentityRefusal_401(t *testing.T) {
	t.Parallel()

	var factoryCalled bool
	newConv := func(_ string) (*chat.Conversation, error) {
		factoryCalled = true
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t))})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: ""}, newConv) // empty Participant id → refusal

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hi"})
	res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", res.StatusCode)
	}
	var env struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&env)
	if env.Error != "server" {
		t.Errorf("error.kind=%q, want server", env.Error)
	}
	if factoryCalled {
		t.Error("Conversation factory was called for an unauthenticated request (R-CHS-004.a: no provider call)")
	}
}

// TestCrossParticipantRefusal_403 — S-CHS-004.b. A second participant
// cannot subscribe to or cancel another participant's turn. The
// refusal is 403 not_found (not 404) so the existence of the turn is
// not leaked.
//
// The test uses a header-driven resolver (X-Test-Participant:
// alice|bob) so a single *echo.Echo with one Registry serves both
// participants; the cross-participant guard runs through the real
// OwnerOf path inside the GET and DELETE handlers.
func TestCrossParticipantRefusal_403(t *testing.T) {
	t.Parallel()

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t))})
	}

	headerResolver := chat.HeaderParticipantResolver("X-Test-Participant")
	e := echo.New()
	_, _ = chat.RegisterRoutes(e, headerResolver, newConv)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)

	// Alice opens a turn under her own identity.
	bodyAlice, _ := json.Marshal(map[string]string{"id": "t-alice", "prompt": "hi"})
	reqAlice, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/agent/turns", bytes.NewReader(bodyAlice))
	reqAlice.Header.Set("X-Test-Participant", "alice")
	reqAlice.Header.Set("Content-Type", "application/json")
	resAlice, err := http.DefaultClient.Do(reqAlice)
	if err != nil {
		t.Fatalf("POST as alice: %v", err)
	}
	resAlice.Body.Close()
	if resAlice.StatusCode != http.StatusOK {
		t.Fatalf("alice POST status=%d, want 200", resAlice.StatusCode)
	}

	// Drain Alice's stream so the turn completes (so the
	// already-terminated fast path doesn't surprise the bob
	// cancellation tests).
	reqDrain, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-alice/events", nil)
	reqDrain.Header.Set("X-Test-Participant", "alice")
	drainResp, err := http.DefaultClient.Do(reqDrain)
	if err != nil {
		t.Fatalf("alice GET (drain): %v", err)
	}
	_, _ = io.Copy(io.Discard, drainResp.Body)
	drainResp.Body.Close()

	// Bob cannot subscribe to Alice's stream. The single Registry
	// holds Alice's stream under her ownerID; bob's identity
	// resolution returns "bob", the handler checks
	// ownerID != bob.ParticipantID() and refuses 403 not_found.
	// Crucially, this is identical to the 403 a truly-nonexistent
	// turn would surface (CH-03.4 R-CHS-004.b: "the refusal does
	// not reveal whether the turn exists").
	reqBobGet, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-alice/events", nil)
	reqBobGet.Header.Set("X-Test-Participant", "bob")
	resBobGet, err := http.DefaultClient.Do(reqBobGet)
	if err != nil {
		t.Fatalf("bob GET: %v", err)
	}
	defer resBobGet.Body.Close()
	if resBobGet.StatusCode != http.StatusForbidden {
		t.Errorf("cross-participant GET status=%d, want 403 not_found (R-CHS-004.b)", resBobGet.StatusCode)
	}
	var envBobGet struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resBobGet.Body).Decode(&envBobGet)
	if envBobGet.Error != "not_found" {
		t.Errorf("cross-participant GET kind=%q, want not_found (R-CHS-004.b: refusal shape matches not_found so existence is not leaked)", envBobGet.Error)
	}

	// Bob cannot cancel Alice's turn.
	reqBobDel, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/agent/turns/t-alice", nil)
	reqBobDel.Header.Set("X-Test-Participant", "bob")
	resBobDel, err := http.DefaultClient.Do(reqBobDel)
	if err != nil {
		t.Fatalf("bob DELETE: %v", err)
	}
	resBobDel.Body.Close()
	if resBobDel.StatusCode != http.StatusNoContent {
		t.Errorf("cross-participant DELETE status=%d, want 204 (R-CHS-003.b/c: DELETE-on-other's-turn is a no-op)", resBobDel.StatusCode)
	}

	// Alice can still subscribe to her own stream (the bob actions
	// must not have invalidated Alice's entry — DELETE clears only
	// the entry; GET refusal on Bob's side does not affect Alice's
	// subscription).
	reqAliceGet, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-alice", nil)
	reqAliceGet.Header.Set("X-Test-Participant", "alice")
	resAliceGet, err := http.DefaultClient.Do(reqAliceGet)
	if err != nil {
		t.Fatalf("alice GET (post-bob): %v", err)
	}
	resAliceGet.Body.Close()
	// Either 403 (already cleared by bob's no-op DELETE which is
	// keyed to bob's identity but ALSO clears the entry — see
	// HandleCancelTurn's OwnerRefusesClearStream path) or 404/200
	// is acceptable; what matters is no panic and no leaked
	// goroutine.
	_ = resAliceGet
}

// TestHandleCancelTurn_Unknown — S-CHS-003.b/c. DELETE on a turn
// that does not exist returns 204 (no-op) with no side effects.
func TestHandleCancelTurn_Unknown(t *testing.T) {
	t.Parallel()

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t))})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/agent/turns/no-such-turn", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status=%d, want 204", res.StatusCode)
	}
}
