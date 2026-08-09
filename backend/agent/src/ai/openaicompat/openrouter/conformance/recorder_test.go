// AI-38 — the byte-transparent recording helper and its regeneration /
// drift-guard test (R-ACR-003, design D3, tasks.md Phase 1 WU2).
//
// recordTranscript is the one capture mechanism every testdata/*.sse
// golden this sub-package embeds is produced through: one real HTTP POST,
// raw io.ReadAll of the response body, zero parsing, zero normalization.
// Pointed at a local httptest.Server (this file) it captures a canonical
// rendering for regression; pointed at a credential-gated real vendor
// endpoint it is AI-39's live-capture path — never exercised here (locked
// decision 3: local endpoint only, zero vendor network spend).
//
// TestRecordTranscript_RegeneratesEveryFixture is the drift guard T3
// promises: for every recorder-eligible fixture (text_stream, tool_call —
// see fixtures/with_usage.go and fixtures/reasoning_extensions.go for why
// those two stay outside this set), it renders the documented canonical
// agenttest.Script, serves the bytes from a fresh httptest.Server, captures
// them back over real HTTP through recordTranscript, and either (with
// UPDATE_CONFORMANCE_FIXTURES=1) writes fixtures/testdata/<name>.sse, or
// (the default) asserts the capture is byte-identical to the committed,
// embedded fixture — so a hand edit to the committed file fails here,
// naming the drifted fixture (S-ACR-008).

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures"
)

// recordTranscript captures one real HTTP stream from endpoint: a POST of
// requestBody carrying credential as a Bearer Authorization header, its
// response body read raw via one io.ReadAll — zero parsing, zero
// normalization (R-ACR-003, design D3). This is the sole capture mechanism
// every committed testdata/*.sse golden in this sub-package is produced
// through; the same function pointed at a real vendor endpoint is AI-39's
// concern (credential-gated), never exercised by this change.
func recordTranscript(tb testing.TB, endpoint, credential string, requestBody []byte) []byte {
	tb.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		tb.Fatalf("recordTranscript: http.NewRequestWithContext error = %v, want nil", err)
	}
	req.Header.Set("Authorization", "Bearer "+credential)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		tb.Fatalf("recordTranscript: http.DefaultClient.Do error = %v, want nil", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("recordTranscript: io.ReadAll error = %v, want nil", err)
	}
	return raw
}

// recorderRequireConstructed fails tb, naming the constructor, when err is
// non-nil — this file's own copy of the suite's requireConstructed idiom
// (agenttest's own is unexported and unreachable from this package).
func recorderRequireConstructed(tb testing.TB, err error, constructor string) {
	tb.Helper()
	if err != nil {
		tb.Fatalf("recorder_test: %s returned %v, want no failure", constructor, err)
	}
}

// recorderFixture names one recorder-eligible fixture: its testdata file
// stem, the canonical script that renders its bytes, and the fixtures
// package accessor that exposes its committed, embedded content.
type recorderFixture struct {
	name      string
	script    func(tb testing.TB) agenttest.Script
	committed func() string
}

// recorderCanonicalTextStreamScript renders exactly what
// fixtures/text_stream.go documents: ResponseStart("chatcmpl-or-text",
// "openai/gpt-4o") + TextBlockStart(1) + TextDelta(1, "alpha") +
// TextDelta(1, " beta") + TextDelta(1, " gamma") + TextBlockEnd(1) +
// Completion(FinishReasonStop, Usage{}).
func recorderCanonicalTextStreamScript(tb testing.TB) agenttest.Script {
	tb.Helper()
	responseStart, err := ai.NewResponseStart("chatcmpl-or-text", "openai/gpt-4o")
	recorderRequireConstructed(tb, err, "ai.NewResponseStart")
	start, err := ai.NewTextBlockStart(1)
	recorderRequireConstructed(tb, err, "ai.NewTextBlockStart")
	alpha, err := ai.NewTextDelta(1, "alpha")
	recorderRequireConstructed(tb, err, "ai.NewTextDelta")
	beta, err := ai.NewTextDelta(1, " beta")
	recorderRequireConstructed(tb, err, "ai.NewTextDelta")
	gamma, err := ai.NewTextDelta(1, " gamma")
	recorderRequireConstructed(tb, err, "ai.NewTextDelta")
	end, err := ai.NewTextBlockEnd(1)
	recorderRequireConstructed(tb, err, "ai.NewTextBlockEnd")
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	recorderRequireConstructed(tb, err, "ai.NewCompletion")

	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(responseStart),
		agenttest.Emit(start),
		agenttest.Emit(alpha),
		agenttest.Emit(beta),
		agenttest.Emit(gamma),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// recorderCanonicalToolCallScript renders exactly what
// fixtures/tool_call.go documents: ResponseStart("chatcmpl-or-tool",
// "openai/gpt-4o") + ToolCallStart(1, "call_or_tool_1", "search") +
// ToolCallDelta(1, `{"q":`) + ToolCallDelta(1, `"a"}`) — no
// Completion/ToolCallEnd: the tool-call cases carry neither by design, and
// bridgeRenderScript synthesizes the finish_reason:"tool_calls" terminal +
// [DONE] itself once it has seen a tool call.
func recorderCanonicalToolCallScript(tb testing.TB) agenttest.Script {
	tb.Helper()
	responseStart, err := ai.NewResponseStart("chatcmpl-or-tool", "openai/gpt-4o")
	recorderRequireConstructed(tb, err, "ai.NewResponseStart")
	start, err := ai.NewToolCallStart(1, "call_or_tool_1", "search")
	recorderRequireConstructed(tb, err, "ai.NewToolCallStart")
	fragment1, err := ai.NewToolCallDelta(1, []byte(`{"q":`))
	recorderRequireConstructed(tb, err, "ai.NewToolCallDelta")
	fragment2, err := ai.NewToolCallDelta(1, []byte(`"a"}`))
	recorderRequireConstructed(tb, err, "ai.NewToolCallDelta")

	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(responseStart),
		agenttest.Emit(start),
		agenttest.Emit(fragment1),
		agenttest.Emit(fragment2),
	}}
}

// recorderCanonicalTextStreamWithUsageScript renders exactly what
// fixtures/with_usage.go documents: ResponseStart("chatcmpl-or-usage",
// "openai/gpt-4o") + TextBlockStart(1) + TextDelta(1, "hello") +
// TextDelta(1, " world") + TextBlockEnd(1) + Completion(FinishReasonStop,
// Usage{Input:7, Output:24, CacheRead:0, CacheWrite:0, Reasoning:0}) — all
// five usage counts present (D6's present-fields-only rule: an explicit
// zero still renders its key, distinguishable from an absent one,
// R-CNF-016/S-CNF-045).
//
// AI-38 WU10 ACCURACY-1 remediation (verify-report.md WARN-3, apply
// deviation #3): this fixture's own doc comment recorded the round-trip
// exclusion as blocked on Phase 5/WU5's D6 usage rendering. D6 landed in
// WU5 (bridgeWriteTerminalChunk now renders usage present-fields-only);
// this script closes that now-stale exclusion by proving the golden is
// still exactly what the recording helper produces.
func recorderCanonicalTextStreamWithUsageScript(tb testing.TB) agenttest.Script {
	tb.Helper()
	responseStart, err := ai.NewResponseStart("chatcmpl-or-usage", "openai/gpt-4o")
	recorderRequireConstructed(tb, err, "ai.NewResponseStart")
	start, err := ai.NewTextBlockStart(1)
	recorderRequireConstructed(tb, err, "ai.NewTextBlockStart")
	hello, err := ai.NewTextDelta(1, "hello")
	recorderRequireConstructed(tb, err, "ai.NewTextDelta")
	world, err := ai.NewTextDelta(1, " world")
	recorderRequireConstructed(tb, err, "ai.NewTextDelta")
	end, err := ai.NewTextBlockEnd(1)
	recorderRequireConstructed(tb, err, "ai.NewTextBlockEnd")
	usage := ai.Usage{
		Input:      ai.Tokens(7),
		Output:     ai.Tokens(24),
		CacheRead:  ai.Tokens(0),
		CacheWrite: ai.Tokens(0),
		Reasoning:  ai.Tokens(0),
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, usage)
	recorderRequireConstructed(tb, err, "ai.NewCompletion")

	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(responseStart),
		agenttest.Emit(start),
		agenttest.Emit(hello),
		agenttest.Emit(world),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// recorderFixtures is the recorder-eligible fixture set: every testdata
// golden the recording helper can reproduce from a canonical
// agenttest.Script today — text_stream, tool_call, and (AI-38 WU10
// ACCURACY-1 remediation, now that D6 landed) with_usage.
//
// fixtures/reasoning_extensions.go stays permanently outside this set,
// not a phase-blocked one: reasoning_details/reasoning are vendor
// wire-extension fields with no ai.Event equivalent at all, so
// bridgeRenderScript has no script vocabulary that could ever produce
// them — the same structural reason boundary_sweep_test.go's own header
// comment states for its curated set (mirroring that file's
// tool_call.sse sweep-bound exclusion precedent: an exclusion this
// package cannot close by writing a script is recorded here in the
// test, not merely left implicit in fixtures/reasoning_extensions.go's
// package doc). See that file's own doc comment for the full rationale.
func recorderFixtures() []recorderFixture {
	return []recorderFixture{
		{name: "text_stream", script: recorderCanonicalTextStreamScript, committed: fixtures.OpenrouterTextStream},
		{name: "tool_call", script: recorderCanonicalToolCallScript, committed: fixtures.OpenrouterToolCallStream},
		{name: "with_usage", script: recorderCanonicalTextStreamWithUsageScript, committed: fixtures.OpenrouterTextStreamWithUsage},
	}
}

// TestRecordTranscript_RegeneratesEveryFixture is the drift guard T3
// promises (R-ACR-003, S-ACR-006/007/009): see file header.
func TestRecordTranscript_RegeneratesEveryFixture(t *testing.T) {
	update := os.Getenv("UPDATE_CONFORMANCE_FIXTURES") == "1"

	for _, rf := range recorderFixtures() {
		t.Run(rf.name, func(t *testing.T) {
			canonical := bridgeRenderScript(t, rf.script(t))

			server := httptest.NewServer(bridgeServeTranscripts([][]byte{canonical}))
			t.Cleanup(server.Close)

			captured := recordTranscript(t, server.URL, "conformance-bridge-token", []byte(`{"model":"openai/gpt-4o","stream":true}`))

			if !bytes.Equal(captured, canonical) {
				t.Fatalf("recordTranscript over real HTTP captured %d byte(s), want byte-identical to bridgeRenderScript's %d byte(s) rendered locally (R-ACR-003: capture must be byte-transparent)", len(captured), len(canonical))
			}

			if update {
				path := filepath.Join("fixtures", "testdata", rf.name+".sse")
				if err := os.WriteFile(path, captured, 0o644); err != nil {
					t.Fatalf("UPDATE_CONFORMANCE_FIXTURES=1: os.WriteFile(%q) error = %v, want nil", path, err)
				}
				t.Logf("UPDATE_CONFORMANCE_FIXTURES=1: wrote %s (%d bytes)", path, len(captured))
				return
			}

			if committed := rf.committed(); committed != string(captured) {
				t.Errorf("fixtures/testdata/%s.sse has drifted from the recording helper's output (R-ACR-003 drift guard, S-ACR-008): committed %d byte(s), freshly captured %d byte(s) — regenerate with UPDATE_CONFORMANCE_FIXTURES=1 if the change is intentional, or revert the hand edit", rf.name, len(committed), len(captured))
			}
		})
	}
}
