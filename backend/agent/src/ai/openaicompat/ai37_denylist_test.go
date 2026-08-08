// AI-37 (WU-8, denylist absence + non-vacuity guards) — R-AOB-007
// (S-AOB-026…029), R-AOB-008 (S-AOB-030…035), AI-37.3.
//
// # The reasoning event-coverage clause cannot be satisfied literally — a
// documented, structural fact about this adapter, not a gap left open here
//
// R-AOB-008 item 3 (S-AOB-032) asks the drained recording to contain
// "text, reasoning, tool-call, completion and error events". This
// adapter's own decision.md (AI-29.0, already landed, three independent
// mechanisms) proves it can NEVER construct an ai.EventKindReasoningDelta
// event on any real transcript: wireDelta (chunk.go) declares no
// reasoning_content field, decodeChunk's plain json.Unmarshal (no
// DisallowUnknownFields) silently drops any such field before a single
// Go value exists to carry it forward, and reasoning_absence_test.go
// already proves exactly this drop for the identical field name. Forcing
// a fabricated reasoning EVENT here to satisfy the literal wording would
// be worse than not satisfying it: it would misrepresent what this
// adapter actually does.
//
// This file's own resolution: the event-coverage guard below asserts the
// four event kinds this adapter CAN produce — text, tool-call, completion,
// error — over the combined two-run exercise. The reasoning CANARY VECTOR
// stays fully planted, in a reasoning_content field on a scripted delta,
// exactly where decision.md says the dialect would carry it if it carried
// it at all — and R-AOB-007's absence scan still proves it never reaches
// any span field, which is in fact a STRONGER claim than "no reasoning
// EVENT exists": the value never reaches a Go struct field of any kind,
// let alone a span.
package openaicompat_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// ai37DenylistCanaries names the seven runtime-assembled markers this test
// plants — six vectors (D-4's own table), the last carrying two markers
// (header value, raw-body marker). Every value is built by concatenation,
// never as one contiguous literal, so this file's own source bytes never
// contain a needle the scan below searches other bytes for.
type ai37DenylistCanaries struct {
	prompt     string
	reasoning  string
	toolArgs   string
	toolResult string
	credential string
	header     string
	rawBody    string
}

func newAI37DenylistCanaries() ai37DenylistCanaries {
	return ai37DenylistCanaries{
		prompt:     "AI37" + "-PROMPT-" + "CANARY-9f2a",
		reasoning:  "AI37" + "-REASONING-" + "CANARY-3c7e",
		toolArgs:   "AI37" + "-TOOLARGS-" + "CANARY-5b1d",
		toolResult: "AI37" + "-TOOLRESULT-" + "CANARY-8e4f",
		credential: "AI37" + "-CREDENTIAL-" + "CANARY-2d6a",
		header:     "AI37" + "-HEADER-" + "CANARY-7a3b",
		rawBody:    "AI37" + "-RAWBODY-" + "CANARY-1f9c",
	}
}

// denyEntries builds the sweep deny-list every one of the seven markers.
func (c ai37DenylistCanaries) denyEntries() []sweep.Entry {
	return []sweep.Entry{
		{Vector: "prompt", Needle: []byte(c.prompt)},
		{Vector: "reasoning", Needle: []byte(c.reasoning)},
		{Vector: "tool-call arguments", Needle: []byte(c.toolArgs)},
		{Vector: "tool-result", Needle: []byte(c.toolResult)},
		{Vector: "credential", Needle: []byte(c.credential)},
		{Vector: "credential (bearer form)", Needle: []byte("Bearer " + c.credential)},
		{Vector: "response header", Needle: []byte(c.header)},
		{Vector: "raw body marker", Needle: []byte(c.rawBody)},
	}
}

// ai37DenylistRequest builds the request carrying the prompt and
// tool-result canaries: a user message, an assistant message whose tool
// call "call_ai37" is what the tool-result message resolves against
// (request.go's own unresolvedToolResultRule requires this correlation to
// exist somewhere in the request).
func ai37DenylistRequest(t *testing.T, c ai37DenylistCanaries) ai.Request {
	t.Helper()
	promptPart, err := ai.NewText("hello " + c.prompt)
	if err != nil {
		t.Fatalf("ai.NewText(prompt) error = %v, want nil", err)
	}
	userMsg, err := ai.NewMessage(ai.RoleUser, promptPart)
	if err != nil {
		t.Fatalf("ai.NewMessage(user) error = %v, want nil", err)
	}

	callPart, err := ai.NewToolCall("call_ai37", "search", []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall() error = %v, want nil", err)
	}
	assistantMsg, err := ai.NewMessage(ai.RoleAssistant, callPart)
	if err != nil {
		t.Fatalf("ai.NewMessage(assistant) error = %v, want nil", err)
	}

	resultPart, err := ai.NewToolResult("call_ai37", c.toolResult)
	if err != nil {
		t.Fatalf("ai.NewToolResult() error = %v, want nil", err)
	}
	toolMsg, err := ai.NewMessage(ai.RoleTool, resultPart)
	if err != nil {
		t.Fatalf("ai.NewMessage(tool) error = %v, want nil", err)
	}

	req, err := ai.NewRequest("ai37-denylist-model", []ai.Message{userMsg, assistantMsg, toolMsg},
		ai.WithMaxOutputTokens(789))
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}
	return req
}

// ai37DenylistSuccessServer answers with a scripted stream carrying: a
// custom response header (the header canary), an SSE comment line (the
// raw-body canary — SSE comments are parsed and discarded structurally,
// keepalive_test.go's own precedent), a text delta, a reasoning_content
// field on a delta (dropped structurally — see this file's header
// comment), a tool-call delta whose arguments carry the tool-args canary,
// a terminal chunk, a usage chunk exercising all four usage counts, and
// [DONE].
func ai37DenylistSuccessServer(t *testing.T, c ai37DenylistCanaries) *httptest.Server {
	t.Helper()
	transcript := "" +
		": " + c.rawBody + "\n\n" +
		"data: {\"id\":\"ai37-dl\",\"model\":\"ai37-denylist-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"thinking\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"ai37-dl\",\"model\":\"ai37-denylist-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"" + c.reasoning + "\"},\"finish_reason\":null}]}\n\n" +
		// The arguments field's JSON VALUE must itself be well-formed JSON
		// (ai.NewToolCall validates it on the clean-close path,
		// stream_state.go's applyChunk terminal-tool-close branch) — the
		// canary is embedded as a JSON string value, not a bare literal.
		"data: {\"id\":\"ai37-dl\",\"model\":\"ai37-denylist-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai37_resp\",\"function\":{\"name\":\"search\",\"arguments\":\"{\\\"query\\\":\\\"" + c.toolArgs + "\\\"}\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"ai37-dl\",\"model\":\"ai37-denylist-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: {\"id\":\"ai37-dl\",\"model\":\"ai37-denylist-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":40,\"completion_tokens\":15,\"total_tokens\":55,\"prompt_tokens_details\":{\"cached_tokens\":6,\"cache_write_tokens\":2},\"completion_tokens_details\":{\"reasoning_tokens\":3}}}\n\n" +
		"data: [DONE]\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ai37-Canary", c.header)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
	}))
	t.Cleanup(server.Close)
	return server
}

// ai37DenylistFailureServer answers with a malformed-identity transcript
// (empty first-chunk id — the same proven applyChunk-failure shape used
// throughout this milestone's own tests), producing the "error" event
// kind the success run above cannot (a completion and a terminal error
// are mutually exclusive within one stream, R-AOB-003).
func ai37DenylistFailureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return ai37SSEServer(t, "data: {\"id\":\"\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
}

// TestAI37_DenylistAbsence is R-AOB-007's absence proof and R-AOB-008's
// four-guard non-vacuity harness (a fifth guard, S-AOB-034's
// captured-field-count assertion, already lives in tracetest_test.go —
// referenced, not duplicated, here).
func TestAI37_DenylistAbsence(t *testing.T) {
	c := newAI37DenylistCanaries()
	provider := tracetest.NewProvider()

	// --- Guard 1: the positive control runs before any clean scan ---
	deny := c.denyEntries()
	if err := sweep.SelfTest(deny); err != nil {
		t.Fatalf("sweep.SelfTest(deny) = %v, want nil — every needle must bite its own synthetic corpus (R-AOB-008 item 1)", err)
	}

	// --- Run A: a successful stream carrying six of the seven markers ---
	serverA := ai37DenylistSuccessServer(t, c)
	clientA, err := openaicompat.New(openaicompat.Config{
		Endpoint:       serverA.URL,
		Credential:     openaicompat.NewCredential(c.credential),
		TracerProvider: provider,
	})
	if err != nil {
		t.Fatalf("New() clientA error = %v, want nil", err)
	}
	chA, err := clientA.Stream(context.Background(), ai37DenylistRequest(t, c))
	if err != nil {
		t.Fatalf("clientA.Stream() error = %v, want nil", err)
	}
	recA := agenttest.DrainAndRecord(t, chA, agenttest.DefaultDrainTimeout)

	// --- Run B: a failing stream, same credential, for the "error" event
	// kind a successful run cannot itself produce (R-AOB-003: a
	// completion and a terminal error are mutually exclusive within one
	// span). Shares provider so both runs feed one corpus. ---
	serverB := ai37DenylistFailureServer(t)
	clientB, err := openaicompat.New(openaicompat.Config{
		Endpoint:       serverB.URL,
		Credential:     openaicompat.NewCredential(c.credential),
		TracerProvider: provider,
	})
	if err != nil {
		t.Fatalf("New() clientB error = %v, want nil", err)
	}
	chB, err := clientB.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("clientB.Stream() error = %v, want nil", err)
	}
	recB := agenttest.DrainAndRecord(t, chB, agenttest.DefaultDrainTimeout)

	if err := provider.AssertAllEndedOnce(); err != nil {
		t.Fatalf("AssertAllEndedOnce() = %v, want nil", err)
	}

	// --- Guard 2: corpus-non-empty (R-AOB-008 item 2, S-AOB-031) ---
	corpus := provider.Corpus()
	if len(corpus) == 0 {
		t.Fatal("Corpus() is empty — the whole absence claim would pass vacuously (R-AOB-008 item 2)")
	}
	if got := provider.Started(); got < 2 {
		t.Fatalf("provider.Started() = %d, want at least 2 (one span per run)", got)
	}
	const wantMinAttributeCount = 12 // the § D3 allowlist's own cardinality
	gotAttributeCount := 0
	for _, span := range provider.Spans() {
		gotAttributeCount += len(span.Attributes())
	}
	if gotAttributeCount < wantMinAttributeCount {
		t.Errorf("recorded attribute count = %d across %d span(s), want at least %d (the § D3 allowlist cardinality, R-AOB-008 item 2)", gotAttributeCount, provider.Started(), wantMinAttributeCount)
	}

	// --- Guard 3: event-coverage, over the four event kinds this adapter
	// can produce (see this file's own header comment on "reasoning") ---
	allEvents := append(append([]ai.Event{}, recA.Events()...), recB.Events()...)
	gotKinds := make(map[ai.EventKind]bool, len(allEvents))
	for _, ev := range allEvents {
		gotKinds[ev.Kind()] = true
	}
	wantKinds := []ai.EventKind{ai.EventKindTextDelta, ai.EventKindToolCallStart, ai.EventKindCompletion, ai.EventKindError}
	for _, want := range wantKinds {
		if !gotKinds[want] {
			t.Errorf("drained recording never contained an event of kind %v — the absence claim must be proven over a run that used it (R-AOB-008 item 3)", want)
		}
	}

	// --- Guard 4: the recorded overlay RED (task 7.1) is separate,
	// build-time evidence — referenced here as a comment, not
	// re-executed inside a normal `go test` run (an overlay is a command-
	// line flag to `go test` itself, not something a test body can
	// trigger on itself). Its red/green transcript is recorded in
	// apply-progress.md and tasks.md Phase 7.

	// --- Guard 5 (S-AOB-034): tracetest_test.go's own
	// TestCorpus_CapturedFieldCountMatchesAPISurface already proves the
	// recorder's captured-field count equals the tracing API surface;
	// not re-derived here.

	// --- The absence scan itself (R-AOB-007) ---
	for _, entry := range deny {
		if _, found := sweep.Scan(corpus, []sweep.Entry{entry}); found {
			t.Errorf("denylist leak: vector %q found in the corpus (R-AOB-007) — failure names the vector only, never the matched bytes (S-AOB-029)", entry.Vector)
		}
	}

	// Sanity: every canary was at least assembled into a live value (not
	// an accidentally-empty string that would make the scan vacuous).
	for _, entry := range deny {
		if len(entry.Needle) == 0 {
			t.Fatalf("a deny entry has an empty needle — the scan above would never have been able to find it")
		}
	}
}
