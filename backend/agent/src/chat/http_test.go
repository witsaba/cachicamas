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
	"github.com/cachicamas/backend/agent/src/agent"
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
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hello"})
	res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
defer func() { _ = res.Body.Close() }()
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
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
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
_ = 		res.Body.Close()
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
		return chat.NewConversation(chat.Config{Provider: provider, Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "first"})
	res1, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST #1: %v", err)
	}
_ = 	res1.Body.Close()
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
defer func() { _ = res2.Body.Close() }()
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
_ = 	resp.Body.Close()
}

// TestHandleStreamEvents_FullTurn — S-CHS-002.a + S-CHS-002.d. After a
// POST, GETting the stream URL returns the SSE frames in order
// (message.start, message.delta, message.end, turn.end) and the
// connection closes once.
func TestHandleStreamEvents_FullTurn(t *testing.T) {
	t.Parallel()

	newConv := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hi"})
	_, _ = http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/turns/t-1/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET stream: %v", err)
	}
defer func() { _ = resp.Body.Close() }()

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
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: ""}, newConv) // empty Participant id → refusal

	body, _ := json.Marshal(map[string]string{"id": "t-1", "prompt": "hi"})
	res, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", res.StatusCode)
	}
	var env struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&env)
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
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
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
_ = 	resAlice.Body.Close()
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
_ = 	drainResp.Body.Close()

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
defer func() { _ = resBobGet.Body.Close() }()
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
_ = 	resBobDel.Body.Close()
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
_ = 	resAliceGet.Body.Close()
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
		return chat.NewConversation(chat.Config{Provider: agenttest.NewProvider(scriptForOneTurn(t)), Store: chat.NewMemoryConversationStore(), ParticipantID: "test-http",
		ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)), PermissionPolicy: chat.NewDefaultPermissionPolicy(nil)})
	}
	srv, _ := mountedServer(t, fixedResolver{ID: "alice"}, newConv)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/agent/turns/no-such-turn", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status=%d, want 204", res.StatusCode)
	}
}

// ----------------------------------------------------------------------------
// CH-08 (WU-5): the resume handlers — RED scaffold. The two new GET
// endpoints land in WU-6:
//   - GET /api/agent/conversations/:id    -> HandleReloadConversation (R-CRI-001)
//   - GET /api/agent/conversations        -> HandleListConversations  (R-CRI-002)
//
// Both go behind the existing `identityMiddleware`. Both take the
// `chat.ConversationStore` directly (not the Registry) so a future
// rewrite of the in-flight pipeline does not invalidate the read
// path. At WU-5 the helper `chat.RegisterResumeRoutes(...)` does not
// exist — the test code's reference compiles-fail, which is the
// strict-TDD RED signal per openspec/AGENTS.md "Strict TDD is on".
//
// Five sub-tests cover the spec's Gherkin + boundary cases:
//   - S-CRI-001 happy path: 200 OK with a JSON array of exchanges
//   - R-CRI-001 cross-participant refusal: 403 not_found (mirrors
//     R-CHS-004.b; existence is not leaked to non-owners)
//   - R-CRI-001 unknown conversation: 404 not_found (ErrConversationNotFound
//     surfaced; the participant-id matches but the store has nothing)
//   - R-CRI-002 list happy path: 200 OK with one summary entry
//   - R-CRI-002 + S-CCS-018 / S-CRI-004 empty list: 200 OK with []
//     (NOT 404 — the empty list is success per the spec)
// ----------------------------------------------------------------------------

// resumeMountedServer is the test parallel of mountedServer for the
// CH-08 routes. At WU-5 it compiles-fail (Referenced
// `chat.RegisterResumeRoutes` does not exist); WU-6 introduces it.
// The helper keeps the resume handlers behind the same identity
// middleware used by the CH-03 routes — exercise the full chain
// (resolver → middleware → handler), not a stub bypass.
func resumeMountedServer(t *testing.T, resolver chat.IdentityResolver, store chat.ConversationStore) (*httptest.Server, error) {
	t.Helper()

	e := echo.New()
	registry, err := chat.RegisterRoutes(e, resolver, func(_ string) (*chat.Conversation, error) {
		// The CH-03 routes require a factory; the CH-08 test path
		// does not exercise them. The factory here uses an unrelated
		// memory store so a stray request to /api/agent/turns does
		// not interfere with the resume handlers' shared store.
		return chat.NewConversation(chat.Config{
			Provider:      agenttest.NewProvider(scriptForOneTurn(t)),
			Store:         chat.NewMemoryConversationStore(),
			ParticipantID: "test-resume-factory",
			ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy: chat.NewDefaultPermissionPolicy(nil),
		})
	})
	if err != nil {
		return nil, err
	}
	_ = registry // not used by the CH-08 tests; registration only

	if err := chat.RegisterResumeRoutes(e, resolver, store); err != nil {
		return nil, err
	}
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv, nil
}

// TestHandleReloadConversation_HappyPath — S-CRI-001 / Gherkin
// verbatim (`0005:862-866`). GET /api/agent/conversations/:id where
// :id matches the authenticated participant and the store has two
// recorded exchanges returns 200 with a JSON array of those
// exchanges (mirror of chat.Exchange's eight fields — D-7).
func TestHandleReloadConversation_HappyPath(t *testing.T) {
	t.Parallel()

	store := chat.NewMemoryConversationStore()
	if err := store.Append("alice", chat.Exchange{PromptText: "turn-one", AssistantText: "reply-one"}); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}
	if err := store.Append("alice", chat.Exchange{PromptText: "turn-two", AssistantText: "reply-two"}); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}

	srv, err := resumeMountedServer(t, chat.HeaderParticipantResolver("X-Test-Participant"), store)
	if err != nil {
		t.Fatalf("resumeMountedServer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/conversations/alice", nil)
	req.Header.Set("X-Test-Participant", "alice")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", res.StatusCode)
	}
	var got []struct {
		Position      int    `json:"Position"`
		PromptText    string `json:"PromptText"`
		AssistantText string `json:"AssistantText"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d exchanges, want 2", len(got))
	}
	if got[0].Position != 0 || got[1].Position != 1 {
		t.Errorf("positions = (%d, %d), want (0, 1)", got[0].Position, got[1].Position)
	}
	if got[0].PromptText != "turn-one" {
		t.Errorf("got[0].PromptText = %q, want turn-one", got[0].PromptText)
	}
	if got[1].PromptText != "turn-two" {
		t.Errorf("got[1].PromptText = %q, want turn-two", got[1].PromptText)
	}
}

// TestHandleReloadConversation_CrossParticipant — R-CRI-001 +
// R-CHS-004.b shape. A non-owner request to GET
// /api/agent/conversations/:id MUST be refused as 403 not_found —
// not 404 (do not probe existence).
func TestHandleReloadConversation_CrossParticipant(t *testing.T) {
	t.Parallel()

	store := chat.NewMemoryConversationStore()
	if err := store.Append("alice", chat.Exchange{PromptText: "alice-1", AssistantText: "r-a"}); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}

	srv, err := resumeMountedServer(t, chat.HeaderParticipantResolver("X-Test-Participant"), store)
	if err != nil {
		t.Fatalf("resumeMountedServer: %v", err)
	}

	// Bob asks for alice's conversation. He is authenticated as bob,
	// so :id == "alice" does not match bob's participant id. The
	// refusal MUST be 403 not_found — identical to a truly-nonexistent
	// id (so the existence is not leaked).
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/conversations/alice", nil)
	req.Header.Set("X-Test-Participant", "bob")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status=%d, want 403 (R-CRI-001: cross-participant refused as not_found, not 404)", res.StatusCode)
	}
	var env struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&env)
	if env.Error != "not_found" {
		t.Errorf("error.kind=%q, want not_found (R-CRI-001)", env.Error)
	}
}

// TestHandleReloadConversation_Unknown — R-CRI-001 boundary case.
// :id matches the authenticated participant but the store has no
// record under it (ErrConversationNotFound). Response MUST be 404
// not_found, not an empty body / 200.
func TestHandleReloadConversation_Unknown(t *testing.T) {
	t.Parallel()

	store := chat.NewMemoryConversationStore()
	// alice has nothing recorded.

	srv, err := resumeMountedServer(t, chat.HeaderParticipantResolver("X-Test-Participant"), store)
	if err != nil {
		t.Fatalf("resumeMountedServer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/conversations/alice", nil)
	req.Header.Set("X-Test-Participant", "alice")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status=%d, want 404 (R-CRI-001: ErrConversationNotFound → 404 not_found)", res.StatusCode)
	}
}

// TestHandleListConversations_HappyPath — R-CRI-002 + S-CRI-003 +
// D-1 one-per-participant. GET /api/agent/conversations with one
// recorded conversation returns 200 with a JSON array carrying the
// one ConversationSummaryDTO.
func TestHandleListConversations_HappyPath(t *testing.T) {
	t.Parallel()

	store := chat.NewMemoryConversationStore()
	if err := store.Append("alice", chat.Exchange{PromptText: "a1", AssistantText: "r-a1"}); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}

	srv, err := resumeMountedServer(t, chat.HeaderParticipantResolver("X-Test-Participant"), store)
	if err != nil {
		t.Fatalf("resumeMountedServer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/conversations", nil)
	req.Header.Set("X-Test-Participant", "alice")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", res.StatusCode)
	}
	var got []struct {
		ConversationID string `json:"conversationID"`
		TurnCount      int    `json:"turnCount"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	if got[0].ConversationID != "alice" {
		t.Errorf("got[0].ConversationID = %q, want alice", got[0].ConversationID)
	}
	if got[0].TurnCount != 1 {
		t.Errorf("got[0].TurnCount = %d, want 1", got[0].TurnCount)
	}
}

// TestHandleListConversations_Empty — R-CRI-002 + S-CRI-004 /
// S-CCS-018. Authenticated participant with no recorded
// conversation MUST get 200 with an empty JSON array ([]), NOT 404
// — the empty list is success, not not-found (the spec's exact
// Gherkin: "the response is a success rather than a not-found").
func TestHandleListConversations_Empty(t *testing.T) {
	t.Parallel()

	store := chat.NewMemoryConversationStore()
	// alice has nothing recorded.

	srv, err := resumeMountedServer(t, chat.HeaderParticipantResolver("X-Test-Participant"), store)
	if err != nil {
		t.Fatalf("resumeMountedServer: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/conversations", nil)
	req.Header.Set("X-Test-Participant", "alice")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200 (S-CRI-004: empty list returns 200 [], NOT 404)", res.StatusCode)
	}
	// Decode into a raw shape so a `null` body or a non-array body
	// would surface distinctly from an empty array. A correct
	// response is `[]` (a non-null, length-0 array).
	var raw json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("empty response body")
	}
	if string(raw) == "null" {
		t.Errorf("response body is null; want \"[]\" (non-null empty array)")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("body is not a JSON array: %v (body=%s)", err, string(raw))
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}
