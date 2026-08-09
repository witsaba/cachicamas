// AI-38 — the OpenRouter conformance bridge: the first concrete adapter
// for agenttest.RunConformance's whole case table (R-OR-06, design D5,
// tasks.md § PR #2 work unit 2.1).

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance
//
// # Bridge factory — the one seam RunConformance consumes
//
// conformanceBridgeFactory is the package's single agenttest.Factory
// (R-CNF-001, R-CNF-002): New renders every given Script eagerly, on the
// caller's own goroutine, then serves the rendered transcripts in queue
// order from a real httptest.Server behind a real *openaicompat.Client.
// All three optional capabilities are declared non-nil false (R-CNF-004,
// W-2): this bridge offers none of them. The factory's tri-state shape
// — non-nil false vs nil — is what applyDeclaredAbsences keys off, so
// every capability the conformance suite enumerates receives its
// absent entry at suite start, before any case runs (S-CNF-004).
//
// # Header injection at the server seam, not the wire-decoder seam
//
// The OpenRouter-specific attribution headers (HTTP-Referer,
// X-OpenRouter-Title + alias X-Title, X-OpenRouter-Categories) live on
// an http.RoundTripper this bridge injects between the openaicompat-
// built transport and the test server. openaicompat's own request.go
// never names those three headers (R-OR-02 sub-scenario 3, design D2
// and headers_unawareness_test.go's raw-bytes scan over openaicompat/
// request.go), so the bridge is the only place they appear — and the
// header injection happens transparently over real HTTP, not as a
// stub-transport probe.
//
// # Why this RoundTripper is duplicated, not imported
//
// The openrouter package's own attributionRoundTripper (transport.go)
// is unexported by design (PR #1 commit 1.2, design D2): the wrapper's
// surface stays the factory function openrouter.NewProvider, with no
// second exported type for callers to wire up by hand. PR #2's hard
// rule — "Do NOT modify PR #1 code" — forbids adding an exported
// factory function to that package, so this bridge carries its own
// minimal duplicate of the round-tripper. The duplication is bounded:
// the type is a struct with five fields and one RoundTrip method that
// clones the inbound header, sets three non-empty attribution values,
// and delegates to the wrapped base. Any future change to the
// attribution-header set or to the aliasing rule will require both
// copies to move together — recorded in this file's header comment
// and tracked at the verify phase's spec-compliance matrix.
//
// # Test-only — no production Go
//
// Per R-OR-09 and the module's ADR-0005 § D1 row 1, this sub-package's
// production Go is exactly zero: every file in this directory is a
// _test.go file. go.mod is unchanged at three lines / zero requires
// (the AI-00.3 forward guard stays green), and the new sub-package's
// own imports are stdlib + the openrouter wrapper + openaicompat +
// agenttest + ai — all reachable through the module path the
// import_boundary_test.go allowlist already admits.
//
// # TDD posture (work-unit 2.1)
//
// The conformance suite this bridge drives is shipped code; this
// commit's RED was "the factory function this PR introduces does not
// yet exist" (compile error referencing conformanceBridgeFactory from
// the assertions below). GREEN is the factory + renderScript +
// serveTranscripts + the bridge-local attributionRoundTripper below
// — the assertions exercise the suite's own construction-time checks
// (requireValidFactory, crossCheckDeclaredOptionalCapabilities) via
// the only exported door that runs them.

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// conformanceBridgeChunkCreated and conformanceBridgeObjectDiscriminator
// are the fixed values every chunk this bridge renders carries
// (openaicompat's bridgeChunkCreated and chunkObjectDiscriminator,
// re-declared here because those shipped names are unexported).
// Keeping them aligned with the shipped package is what makes every
// rendered chunk a recognized, non-malformed one — the openaicompat
// chunk decoder (chunk.go) rejects frames whose object discriminator
// is not "chat.completion.chunk" (R-ATS-017 / D3) and treats a zero
// created as a malformed identity chunk (R-ATS-021 / S-ATS-081).
const (
	conformanceBridgeChunkCreated      = 1700000000
	conformanceBridgeObjectDiscriminator = "chat.completion.chunk"
)

// bridgeAttributionRoundTripper is the bridge-local mirror of the
// openrouter package's attributionRoundTripper (transport.go:48-91).
// It injects the three OpenRouter-shaped attribution headers on every
// outbound request — see this file's header comment for why the
// duplication is intentional under the PR #2 "Do NOT modify PR #1
// code" rule.
//
// An empty attribution string suppresses its header, matching
// openrouter's own suppression rule (R-OR-02 sub-scenario 2). No
// header is ever set with an empty value.
type bridgeAttributionRoundTripper struct {
	base                            http.RoundTripper
	referer, xTitle, xCategories    string
}

// RoundTrip clones the request's headers, applies the three
// attribution headers when their values are non-empty, clones the
// request so the outbound request is a fresh value with no aliasing
// of the caller's req, and delegates to the wrapped base RoundTripper.
// http.Header.Clone is documented thread-safe; req.Clone(ctx) is
// documented thread-safe; RoundTrip is serial per request (http
// transport serializes per-request state).
func (rt bridgeAttributionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clonedHeader := req.Header.Clone()
	if rt.referer != "" {
		clonedHeader.Set("HTTP-Referer", rt.referer)
	}
	if rt.xTitle != "" {
		clonedHeader.Set("X-OpenRouter-Title", rt.xTitle)
		clonedHeader.Set("X-Title", rt.xTitle)
	}
	if rt.xCategories != "" {
		clonedHeader.Set("X-OpenRouter-Categories", rt.xCategories)
	}
	clonedReq := req.Clone(req.Context())
	clonedReq.Header = clonedHeader
	return rt.base.RoundTrip(clonedReq)
}

// bridgeServeTranscripts returns a handler that serves transcripts[n]
// to the (n+1)th inbound request, in arrival order. It calls no
// *testing.T method: every transcript was already rendered (and any
// Hold-step failure already raised) before this handler is ever
// installed.
func bridgeServeTranscripts(transcripts [][]byte) http.HandlerFunc {
	var mu sync.Mutex
	next := 0
	return func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		idx := next
		next++
		mu.Unlock()

		if idx >= len(transcripts) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(transcripts[idx])
	}
}

// bridgeQuoteJSONString renders s as one JSON string literal, escaping
// only the bytes JSON syntax itself requires (the quote, the backslash
// and control characters below 0x20) and passing every other byte
// through verbatim — deliberately NOT encoding/json.Marshal, which
// substitutes U+FFFD for invalid UTF-8 and would repair exactly the
// split-rune bytes the suite's text-ordering case requires this
// bridge to reproduce unmodified.
func bridgeQuoteJSONString(s string) string {
	var out strings.Builder
	out.Grow(len(s) + 2)
	out.WriteByte('"')
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch {
		case b == '"':
			out.WriteString(`\"`)
		case b == '\\':
			out.WriteString(`\\`)
		case b < 0x20:
			fmt.Fprintf(&out, `\u%04x`, b)
		default:
			out.WriteByte(b)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// bridgeQuoteJSONStringBytes is the byte-slice variant of
// bridgeQuoteJSONString for raw-byte fragments that may include
// control characters.
func bridgeQuoteJSONStringBytes(b []byte) string {
	var out strings.Builder
	out.Grow(len(b) + 2)
	out.WriteByte('"')
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == '"':
			out.WriteString(`\"`)
		case c == '\\':
			out.WriteString(`\\`)
		case c < 0x20:
			fmt.Fprintf(&out, `\u%04x`, c)
		default:
			out.WriteByte(c)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// bridgeWriteResponseStartChunk appends one SSE data frame carrying
// the chunk-level identity fields (id, model, created, object) plus
// choice 0's role-only delta — the lifecycle prefix every started-
// response conformance case opens its drained window with
// (R-CNF-019, R-CNF-020).
func bridgeWriteResponseStartChunk(buf *bytes.Buffer, id, model string) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), conformanceBridgeChunkCreated, conformanceBridgeObjectDiscriminator)
}

// bridgeWriteTextDeltaChunk appends one SSE data frame carrying choice
// 0's content delta (R-ATS-007, R-CNF-005). The object discriminator
// is set to the chunk spelling so this bridge's rendered transcripts
// model legal wire bytes (R-ATS-017, D3).
func bridgeWriteTextDeltaChunk(buf *bytes.Buffer, id, model, content string) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), conformanceBridgeChunkCreated, conformanceBridgeObjectDiscriminator, bridgeQuoteJSONString(content))
}

// bridgeWriteTerminalChunk appends one SSE data frame carrying choice
// 0's terminal finish_reason with an empty delta (C2's own
// convention) and, when usage carries at least one present count, a
// top-level "usage" member rendering PRESENT FIELDS ONLY (design D6):
// an absent ai.TokenCount omits its wire key entirely; a count
// reported as zero still emits 0, distinguishable from absence by the
// key's mere presence — resolving usage/absent_vs_zero_distinguishable
// (R-CNF-016, S-CNF-045). Every existing case scripts ai.Usage{} (all
// five counts absent) and keeps rendering with no "usage" key at all —
// byte-identical to before this change. The object discriminator is
// set the same way bridgeWriteTextDeltaChunk sets it.
func bridgeWriteTerminalChunk(buf *bytes.Buffer, id, model string, reason ai.FinishReason, usage ai.Usage) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":%s}]%s}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), conformanceBridgeChunkCreated, conformanceBridgeObjectDiscriminator, bridgeQuoteJSONString(reason.String()), bridgeUsageSuffix(usage))
}

// bridgeUsageSuffix renders a leading-comma "usage" member — matching
// openaicompat/chunk.go's wireUsage field names exactly (prompt_tokens,
// completion_tokens, prompt_tokens_details{cached_tokens,
// cache_write_tokens}, completion_tokens_details{reasoning_tokens}) — for
// usage's present fields only, or "" when every count is absent (design
// D6's present-fields-only rule).
func bridgeUsageSuffix(usage ai.Usage) string {
	var fields []string
	if n, present := usage.Input.Count(); present {
		fields = append(fields, fmt.Sprintf("\"prompt_tokens\":%d", n))
	}
	if n, present := usage.Output.Count(); present {
		fields = append(fields, fmt.Sprintf("\"completion_tokens\":%d", n))
	}
	if detail := bridgePromptTokensDetails(usage); detail != "" {
		fields = append(fields, "\"prompt_tokens_details\":{"+detail+"}")
	}
	if n, present := usage.Reasoning.Count(); present {
		fields = append(fields, fmt.Sprintf("\"completion_tokens_details\":{\"reasoning_tokens\":%d}", n))
	}
	if len(fields) == 0 {
		return ""
	}
	return ",\"usage\":{" + strings.Join(fields, ",") + "}"
}

// bridgePromptTokensDetails renders usage's CacheRead/CacheWrite present
// fields as prompt_tokens_details' inner members, or "" when both are
// absent — the same present-fields-only rule bridgeUsageSuffix applies at
// the top level.
func bridgePromptTokensDetails(usage ai.Usage) string {
	var fields []string
	if n, present := usage.CacheRead.Count(); present {
		fields = append(fields, fmt.Sprintf("\"cached_tokens\":%d", n))
	}
	if n, present := usage.CacheWrite.Count(); present {
		fields = append(fields, fmt.Sprintf("\"cache_write_tokens\":%d", n))
	}
	return strings.Join(fields, ",")
}

// bridgeErrorFrameLabel returns the fixed, category-derived label this
// bridge renders into an in-band error frame's "type" field (design D6).
// Deterministic and safe by construction: it depends only on the
// scripted ai.Failure's OWN Category() — never on anything a script could
// plant a sentinel into (Cause, RequestID) — so every one of the nine
// categories gets a distinct, non-secret label (conformance_redaction.go's
// own file header: RawLabel is the sanctioned rendering channel, not
// Cause/RequestID).
func bridgeErrorFrameLabel(category ai.FailureCategory) string {
	return "conformance_" + category.String() + "_error"
}

// bridgeWriteErrorFrame appends one in-band error frame — the wire shape
// isInBandErrorFrame/failureFromErrorFrame (openaicompat/stream_failure.go)
// decode: {"error":{"type":<fixed-label-per-category>,"message":<...>}}
// (design D6). The message carries payload's UNWRAPPED cause text,
// followed by its request id when present — verbatim, deliberately:
// whatever a script planted there (including a redaction sentinel) lands
// on the wire on purpose, because the redaction case exists to test the
// ADAPTER's own redaction paths against real leaking input, not this test
// bridge (R-CNF-013, R-CNF-027). "type" never carries either — only the
// fixed, category-derived label above.
func bridgeWriteErrorFrame(buf *bytes.Buffer, payload *ai.Failure) {
	var message strings.Builder
	if cause := payload.Unwrap(); cause != nil {
		message.WriteString(cause.Error())
	}
	if reqID := payload.RequestID(); reqID != "" {
		if message.Len() > 0 {
			message.WriteByte(' ')
		}
		message.WriteString("request_id=")
		message.WriteString(reqID)
	}
	fmt.Fprintf(buf, "data: {\"error\":{\"type\":%s,\"message\":%s}}\n\n",
		bridgeQuoteJSONString(bridgeErrorFrameLabel(payload.Category())), bridgeQuoteJSONString(message.String()))
}

// bridgeWriteToolStartChunk appends one SSE data frame carrying a
// tool-call start element. The element shape is {index, id,
// function{name}} — arguments absent (R-ATL-001 / C9.1).
func bridgeWriteToolStartChunk(buf *bytes.Buffer, id, model string, wireIndex int, callID, name string) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":%d,\"id\":%s,\"function\":{\"name\":%s}}]},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), conformanceBridgeChunkCreated, conformanceBridgeObjectDiscriminator, wireIndex, bridgeQuoteJSONString(callID), bridgeQuoteJSONString(name))
}

// bridgeWriteToolDeltaChunk appends one SSE data frame carrying a
// tool-call argument fragment. The element shape is {index,
// function{arguments}} — id/name omitted (C9.3).
func bridgeWriteToolDeltaChunk(buf *bytes.Buffer, id, model string, wireIndex int, fragment []byte) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":%d,\"function\":{\"arguments\":%s}}]},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), conformanceBridgeChunkCreated, conformanceBridgeObjectDiscriminator, wireIndex, bridgeQuoteJSONStringBytes(fragment))
}

// bridgeRenderScript renders script's ordered Steps as one SSE
// transcript, reading events solely through R-CNF-024's read-only
// introspection (Step.IsHold, Step.Event). tb.Fatalf runs on the
// caller's own goroutine — see this file's header comment for the
// rule that drives that.
//
// Scripts lacking ResponseStart / Completion / [DONE] (the
// conformance cases do, by design) get synthesized identity constants
// and a terminal chunk carrying finish_reason:"tool_calls" + [DONE] if
// the script carried at least one tool-call event — otherwise the
// replay never produces a clean completion event (the conformance
// cases carry neither, by their own design, and the synthesis is the
// only way a fully-empty script can still drain to a valid terminal).
func bridgeRenderScript(tb testing.TB, script agenttest.Script) []byte {
	tb.Helper()

	var buf bytes.Buffer
	var id, model string

	// toolWireIndex maps each emitted ToolCallStart's BlockIndex to a
	// 0-based wire index, in start order. The wire index is what the
	// decoder keys per-call accumulation by, so the bridge must mint
	// one wire index per block and reuse it across that block's
	// deltas. The conformance cases supply a Sequence, not a wire
	// index, so this mapping is the bridge's job.
	toolWireIndex := map[ai.BlockIndex]int{}
	nextWire := 0
	// toolCallsSeen tracks whether the script has at least one
	// tool-call event; if so, the bridge must synthesize a terminal
	// chunk and [DONE] because the conformance cases (intentionally,
	// per their own design) carry neither.
	toolCallsSeen := false

	// Pre-scan: if the script carries no ResponseStart, synthesize
	// identity constants BEFORE rendering any chunks — otherwise the
	// first chunk's id/model would be empty and the adapter would
	// reject it as malformed (C1 required fields).
	for _, step := range script.Steps {
		if step.IsHold() {
			continue
		}
		if ev, ok := step.Event(); ok && ev.Kind() == ai.EventKindResponseStart {
			start, _ := ev.ResponseStart()
			id, model = start.ResponseID(), start.ServedModel()
			break
		}
	}
	if id == "" {
		id = "openrouter-conformance-bridge-id"
		model = "openai/gpt-4o"
	}

	for _, step := range script.Steps {
		if step.IsHold() {
			tb.Fatalf("bridge: renderScript: script carries a Hold step — gate-driven scripts are outside CapStreamingText scope (R-CNF-024, design.md D11)")
			return nil
		}
		ev, ok := step.Event()
		if !ok {
			tb.Fatalf("bridge: renderScript: a non-hold step carries no event — script-authoring defect")
			return nil
		}

		switch ev.Kind() {
		case ai.EventKindResponseStart:
			start, _ := ev.ResponseStart()
			id, model = start.ResponseID(), start.ServedModel()
			bridgeWriteResponseStartChunk(&buf, id, model)
		case ai.EventKindTextBlockStart, ai.EventKindTextBlockEnd:
			// No bytes: the wire carries no block-framing signal
			// (C3); the adapter re-mints its own boundaries on
			// decode (R-ATS-008).
		case ai.EventKindTextDelta:
			delta, _ := ev.TextDelta()
			bridgeWriteTextDeltaChunk(&buf, id, model, delta.Delta())
		case ai.EventKindCompletion:
			completion, _ := ev.Completion()
			bridgeWriteTerminalChunk(&buf, id, model, completion.FinishReason(), completion.Usage())
			buf.WriteString("data: [DONE]\n\n")
		case ai.EventKindError:
			// D6: an in-band error frame is this dialect's own terminal
			// (R-AEM-010) — render it and stop, matching the real
			// adapter's contract that nothing follows a terminal.
			payload, _ := ev.ErrorPayload()
			bridgeWriteErrorFrame(&buf, payload)
			return buf.Bytes()
		case ai.EventKindToolCallStart:
			s, _ := ev.ToolCallStart()
			idx, ok := toolWireIndex[s.BlockIndex()]
			if !ok {
				idx = nextWire
				toolWireIndex[s.BlockIndex()] = idx
				nextWire++
			}
			bridgeWriteToolStartChunk(&buf, id, model, idx, s.ID(), s.Name())
			toolCallsSeen = true
		case ai.EventKindToolCallDelta:
			d, _ := ev.ToolCallDelta()
			idx, ok := toolWireIndex[d.BlockIndex()]
			if !ok {
				idx = nextWire
				toolWireIndex[d.BlockIndex()] = idx
				nextWire++
			}
			bridgeWriteToolDeltaChunk(&buf, id, model, idx, d.Fragment())
		case ai.EventKindToolCallEnd:
			// No bytes: C9.6 — no per-call end signal exists on the
			// wire. Encoding one would invent a wire signal that
			// doesn't exist.
			_ = ev
		default:
			tb.Fatalf("bridge: renderScript: unsupported event kind %v — outside this bridge's scope", ev.Kind())
			return nil
		}
	}

	// If the script carried tool calls but no Completion / [DONE],
	// mint a terminal chunk and the sentinel — otherwise the replay
	// never produces a clean completion event (the conformance cases
	// carry neither, by their own design).
	if toolCallsSeen {
		bridgeWriteTerminalChunk(&buf, id, model, ai.FinishReasonToolCalls, ai.Usage{})
		buf.WriteString("data: [DONE]\n\n")
	}

	return buf.Bytes()
}

// conformanceBridgeFactory builds the agenttest.Factory this
// milestone's bridge exposes (R-CNF-001, R-CNF-002, design D5):
//
//   - New renders every given Script eagerly, on the caller's own
//     goroutine, then serves the rendered transcripts, in queue order,
//     from a real httptest.Server behind a real *openaicompat.Client.
//     The HTTP client carries the bridge's local attribution
//     round-tripper (this file's bridgeAttributionRoundTripper),
//     injecting the three OpenRouter-shaped attribution headers on
//     every outbound request — exactly the seam the openrouter
//     wrapper's own round-tripper would inject, mirrored here because
//     the wrapper's own round-tripper is unexported and PR #2 cannot
//     modify PR #1.
//   - Reasoning, TokenCounting and CacheBoundary are all declared
//     non-nil false (R-CNF-004, W-2): this bridge offers none of
//     them. The tri-state shape — non-nil false vs nil — is what
//     applyDeclaredAbsences keys off, so every optional capability
//     receives its absent entry at suite start, before any case runs
//     (S-CNF-004).
//
// # Prior-server eager close (AI-38 WU3 discovery)
//
// Every New call spins up a fresh, real httptest.Server (a real TCP
// listener plus its Serve goroutine). agenttest.RequireNoGoroutineLeak
// calls a scenario — which calls New — up to 50 times against the SAME
// tb, and tb.Cleanup callbacks all run at the end of that one test, not
// incrementally: relying on tb.Cleanup alone would leave up to 50
// concurrently-open servers alive at RequireNoGoroutineLeak's own
// goroutine-count measurement, which is a test-harness artifact, not a
// production leak (AI-21's in-process FakeFactory has no such cost, so
// this was never observable before AI-38 exercises a real HTTP factory
// through the same helper). New instead closes the PREVIOUS server it
// built, under this closure's own mutex, before returning the new one —
// safe because every conformance case's scenario is fully synchronous
// (construct, stream, drain) before the next New call is ever made — and
// registers exactly one tb.Cleanup, closing whichever server is current
// when the whole test ends.
func conformanceBridgeFactory() agenttest.Factory {
	reasoningOffered, tokenCountingOffered, cacheBoundaryOffered, retryOffered := false, false, false, false

	var mu sync.Mutex
	var current *httptest.Server
	cleanupRegistered := false

	return agenttest.Factory{
		New: func(tb testing.TB, scripts ...agenttest.Script) ai.ModelProvider {
			tb.Helper()

			transcripts := make([][]byte, len(scripts))
			for i, script := range scripts {
				transcripts[i] = bridgeRenderScript(tb, script)
			}

			server := httptest.NewServer(bridgeServeTranscripts(transcripts))

			mu.Lock()
			previous := current
			current = server
			needsCleanup := !cleanupRegistered
			cleanupRegistered = true
			mu.Unlock()

			if previous != nil {
				previous.Close()
			}
			if needsCleanup {
				tb.Cleanup(func() {
					mu.Lock()
					defer mu.Unlock()
					if current != nil {
						current.Close()
					}
				})
			}

			transport := bridgeAttributionRoundTripper{
				base:         server.Client().Transport,
				referer:      "https://app.cachicamas.test/openrouter-conformance",
				xTitle:       "cachicamas-conformance-bridge",
				xCategories:  "test,conformance",
			}
			httpClient := &http.Client{Transport: transport}

			client, err := openaicompat.New(openaicompat.Config{
				Endpoint:   server.URL,
				Credential: openaicompat.NewCredential("conformance-bridge-token"),
				HTTPClient: httpClient,
			})
			if err != nil {
				tb.Fatalf("bridge: openaicompat.New(Config) error = %v, want nil", err)
			}
			return client
		},
		Reasoning:     &reasoningOffered,
		TokenCounting: &tokenCountingOffered,
		CacheBoundary: &cacheBoundaryOffered,
		Retry:         &retryOffered,
		Dialect:       conformanceBridgeDialect(),
	}
}

// conformanceBridgeMidStreamCollapse is the single category
// failureFromErrorFrame (openaicompat/stream_failure.go) hardcodes for
// every in-band error frame, regardless of the vendor's real category —
// the dialect carries no discriminator that could preserve it, only the
// vendor's own RawLabel. conformanceBridgeDialect declares this openly
// (design D5) rather than leaving the mid-stream half of
// terminalFailureCategoryExhaustivenessCase to fail against it.
var conformanceBridgeMidStreamCollapse = ai.FailureCategoryUnknown

// conformanceBridgeDialect declares this bridge's wire-dialect
// expressiveness limits (design D5, AI-38, locked decision 4 — no
// R-ACP-002 reopen): Refusal, PauseTurn and Unknown are unreachable finish
// reasons on a normally-finished stream (openaicompat's chunk.go strict
// gate rejects them as typed malformed responses, R-ACP-002, S-ACP-004),
// and every mid-stream failure category collapses to Unknown
// (conformanceBridgeMidStreamCollapse, matching failureFromErrorFrame's
// shipped hardcoded category exactly).
func conformanceBridgeDialect() *agenttest.DialectConstraints {
	return &agenttest.DialectConstraints{
		UnreachableFinishReasons: []ai.FinishReason{
			ai.FinishReasonRefusal,
			ai.FinishReasonPauseTurn,
			ai.FinishReasonUnknown,
		},
		MidStreamCategoryCollapse: &conformanceBridgeMidStreamCollapse,
	}
}

// TestConformanceBridgeFactory_DeclaresAllThreeOptionalCapabilities
// covers R-CNF-002 and S-CNF-006 for the OpenRouter conformance
// bridge: every optional-capability declaration on the factory is a
// non-nil pointer to false — distinguishable from a nil pointer
// (declaration by omission), which is what factoryDefect rejects at
// construction. With all three declarations non-nil false,
// applyDeclaredAbsences records each of CAP-O-01 / CAP-O-02 / CAP-O-03
// as OutcomeAbsent at suite start, before any case runs (S-CNF-004)
// — the exact "absent × 3" shape R-OR-05 and AI-24 §8's capability
// table expect.
func TestConformanceBridgeFactory_DeclaresAllThreeOptionalCapabilities(t *testing.T) {
	t.Parallel()

	factory := conformanceBridgeFactory()

	if factory.New == nil {
		t.Fatal("factory.New is nil (R-CNF-001, NFR-CNF-E) — bridge has no subject builder")
	}
	if factory.Reasoning == nil {
		t.Fatal("factory.Reasoning is nil (R-CNF-002, S-CNF-006) — declaration is by omission; declaration and omission are not distinguishable")
	}
	if *factory.Reasoning != false {
		t.Errorf("factory.Reasoning = %v, want false (R-CNF-004 absent declaration — bridge declares CAP-O-01 not offered)", *factory.Reasoning)
	}
	if factory.TokenCounting == nil {
		t.Fatal("factory.TokenCounting is nil (R-CNF-002, S-CNF-006) — declaration is by omission")
	}
	if *factory.TokenCounting != false {
		t.Errorf("factory.TokenCounting = %v, want false (R-CNF-004 absent declaration — bridge declares CAP-O-02 not offered)", *factory.TokenCounting)
	}
	if factory.CacheBoundary == nil {
		t.Fatal("factory.CacheBoundary is nil (R-CNF-002, S-CNF-006) — declaration is by omission")
	}
	if *factory.CacheBoundary != false {
		t.Errorf("factory.CacheBoundary = %v, want false (R-CNF-004 absent declaration — bridge declares CAP-O-03 not offered)", *factory.CacheBoundary)
	}
}