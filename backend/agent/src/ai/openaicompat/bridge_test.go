// AI-28.1.2 — R-ATS-011: AI-23.2's text conformance cases pass against real
// transport (S-ATS-040…043). AI-30.1 extends this bridge to render
// ToolCallStart / ToolCallDelta / ToolCallEnd events to SSE for
// TestConformanceBridge_ToolCalls (S-ATL-059, R-ATL-012) — cases 1
// (fragmented_interleaved) and 3 (ordinal) only; cases 2 (zero_delta) and
// 4 (mixed_text_and_tool) are gated on the suite reconciliation that
// AI-30.2 raises as its own change against ai-provider-conformance-suite.
//
// Test-only, adapter-owned bridge — lives only in this package's _test.go
// files (S-ATS-042); no file under src/agenttest is read for anything but
// its two landed, exported surfaces: agenttest.RunConformanceFor
// (R-CNF-023) and Script.Steps/Step.Event/Step.IsHold (R-CNF-024). No
// unexported agenttest state is reached anywhere in this file.
//
// # Rendering happens on the test goroutine, not the handler goroutine
//
// renderScript is called once per Script, synchronously, inside the
// factory's own New — before the httptest.Server starts — specifically so
// a Hold step's tb.Fatalf (fail-fast; gate-driven scripts are outside
// CapStreamingText scope, design.md D11) runs on the goroutine Go's testing
// package requires FailNow to run on. The HTTP handler itself only ever
// serves pre-rendered bytes; it calls no *testing.T method at all.
//
// # No block bytes on the wire (C3)
//
// TextBlockStart and TextBlockEnd steps render zero bytes: the wire itself
// carries no block-framing signal (C3, negative, load-bearing), and this
// adapter re-mints its own block boundaries on decode (R-ATS-008). Encoding
// them here would be inventing a wire signal that does not exist.
//
// # Tool-call events render to a chunk carrying delta.tool_calls (AI-30.1)
//
// ToolCallStart → element {index, id, function{name}} — wire index 0,1,2…
// minted per block in start order. ToolCallDelta → element
// {index, function{arguments:fragment}} via bridgeQuoteJSONString.
// ToolCallEnd → zero bytes (C9.6: no per-call end signal on the wire —
// encoding one would be inventing a wire signal). Scripts lacking
// ResponseStart / Completion / [DONE] (the conformance cases do, by
// design) get synthesized identity constants, a terminal chunk carrying
// finish_reason:"tool_calls", and [DONE] — otherwise replay cannot
// construct a valid stream at all.

package openaicompat

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
)

// conformanceBridgeFactory builds the agenttest.Factory this milestone's
// bridge exposes (design.md D11): New renders every given Script eagerly,
// on the caller's own goroutine, then serves the rendered transcripts, in
// queue order, from a real httptest.Server behind a real *Client. All three
// optional capabilities are declared non-nil false (W-2): this bridge
// offers none of them.
func conformanceBridgeFactory() agenttest.Factory {
	reasoningOffered, tokenCountingOffered, cacheBoundaryOffered := false, false, false
	return agenttest.Factory{
		New: func(tb testing.TB, scripts ...agenttest.Script) ai.ModelProvider {
			tb.Helper()

			transcripts := make([][]byte, len(scripts))
			for i, script := range scripts {
				transcripts[i] = renderScript(tb, script)
			}

			server := httptest.NewServer(serveTranscripts(transcripts))
			tb.Cleanup(server.Close)

			client, err := New(Config{
				Endpoint:   server.URL,
				Credential: NewCredential("conformance-bridge-token"),
				HTTPClient: server.Client(),
			})
			if err != nil {
				tb.Fatalf("bridge: New(Config) error = %v, want nil", err)
			}
			return client
		},
		Reasoning:     &reasoningOffered,
		TokenCounting: &tokenCountingOffered,
		CacheBoundary: &cacheBoundaryOffered,
	}
}

// serveTranscripts returns a handler that serves transcripts[n] to the
// (n+1)th inbound request, in arrival order. It calls no *testing.T method:
// every transcript was already rendered (and any Hold-step failure already
// raised) before this handler is ever installed.
func serveTranscripts(transcripts [][]byte) http.HandlerFunc {
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

// renderScript renders script's ordered Steps as one SSE transcript,
// reading events solely through R-CNF-024's read-only introspection
// (Step.IsHold, Step.Event) — S-ATS-041's own premise. tb.Fatalf runs on
// the caller's own goroutine (see this file's header comment).
func renderScript(tb testing.TB, script agenttest.Script) []byte {
	tb.Helper()

	var buf bytes.Buffer
	var id, model string
	// toolWireIndex maps each emitted ToolCallStart's BlockIndex to a
	// 0-based wire index, in start order. The wire index is what the
	// decoder keys per-call accumulation by, so the bridge must mint
	// one wire index per block and reuse it across that block's deltas.
	// S-ATL-059's conformance cases supply a Sequence, not a wire index,
	// so this mapping is the bridge's job.
	toolWireIndex := map[ai.BlockIndex]int{}
	nextWire := 0
	// toolCallsSeen tracks whether the script has at least one tool-call
	// event; if so, the bridge must synthesize a terminal chunk and
	// [DONE] because the conformance cases (intentionally, per their
	// own design) carry neither.
	toolCallsSeen := false

	// Pre-scan: if the script carries no ResponseStart (typical for the
	// tool-call conformance cases), synthesize identity constants BEFORE
	// rendering any chunks — otherwise the first chunk's id/model would
	// be empty and the adapter would reject it as malformed (C1 required
	// fields).
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
		id = "bridge-synthesized-id"
		model = "bridge-synthesized-model"
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
		case ai.EventKindTextBlockStart, ai.EventKindTextBlockEnd:
			// No bytes: the wire carries no block-framing signal (C3); this
			// adapter re-mints its own boundaries on decode (R-ATS-008).
		case ai.EventKindTextDelta:
			delta, _ := ev.TextDelta()
			writeDeltaChunk(&buf, id, model, delta.Delta())
		case ai.EventKindCompletion:
			completion, _ := ev.Completion()
			writeTerminalChunk(&buf, id, model, completion.FinishReason())
			buf.WriteString("data: [DONE]\n\n")
		case ai.EventKindToolCallStart:
			s, _ := ev.ToolCallStart()
			idx, ok := toolWireIndex[s.BlockIndex()]
			if !ok {
				idx = nextWire
				toolWireIndex[s.BlockIndex()] = idx
				nextWire++
			}
			writeToolStartChunk(&buf, id, model, idx, s.ID(), s.Name())
			toolCallsSeen = true
		case ai.EventKindToolCallDelta:
			d, _ := ev.ToolCallDelta()
			idx, ok := toolWireIndex[d.BlockIndex()]
			if !ok {
				idx = nextWire
				toolWireIndex[d.BlockIndex()] = idx
				nextWire++
			}
			writeToolDeltaChunk(&buf, id, model, idx, d.Fragment())
		case ai.EventKindToolCallEnd:
			// No bytes: C9.6 — no per-call end signal exists on the wire.
			// Encoding one would invent a wire signal that doesn't exist.
			_ = ev
		default:
			tb.Fatalf("bridge: renderScript: unsupported event kind %v — outside this bridge's scope", ev.Kind())
			return nil
		}
	}

	// If the script carried tool calls but no Completion / [DONE], mint
	// a terminal chunk and the sentinel — otherwise the replay never
	// produces a clean completion event (the conformance cases carry
	// neither, by their own design).
	if toolCallsSeen {
		writeTerminalChunk(&buf, id, model, ai.FinishReasonToolCalls)
		buf.WriteString("data: [DONE]\n\n")
	}

	return buf.Bytes()
}

// writeToolStartChunk appends one SSE data frame carrying a tool-call
// start element. The element shape is {index, id, function{name}} —
// arguments absent (per R-ATL-001 / C9.1's own `type` field and the
// dialect-conventional reading that identity and name arrive together).
func writeToolStartChunk(buf *bytes.Buffer, id, model string, wireIndex int, callID, name string) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":%d,\"id\":%s,\"function\":{\"name\":%s}}]},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), bridgeChunkCreated, chunkObjectDiscriminator, wireIndex, bridgeQuoteJSONString(callID), bridgeQuoteJSONString(name))
}

// writeToolDeltaChunk appends one SSE data frame carrying a tool-call
// argument fragment. The element shape is {index, function{arguments}}
// — id/name omitted (the dialect-conventional reading C9.3 names, and
// S-ATL-014 proves absence on later elements is normal).
func writeToolDeltaChunk(buf *bytes.Buffer, id, model string, wireIndex int, fragment []byte) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":%d,\"function\":{\"arguments\":%s}}]},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), bridgeChunkCreated, chunkObjectDiscriminator, wireIndex, bridgeQuoteJSONStringBytes(fragment))
}

// bridgeQuoteJSONStringBytes is the byte-slice variant of
// bridgeQuoteJSONString for raw-byte fragments that may include control
// characters.
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

// bridgeChunkCreated is the fixed "created" value every chunk this bridge
// renders carries (R-ATS-021's C1 required-field set, verify-report W1):
// nothing in this bridge or the scripts it renders ever reads it back, so
// one fixed value — matching the rest of this package's own fixture
// normalization — is enough to keep every rendered chunk a recognized,
// non-malformed one.
const bridgeChunkCreated = 1700000000

// writeDeltaChunk appends one SSE data frame carrying a choice-0 content
// delta (R-ATS-007). The object discriminator (C1) is set to the chunk
// spelling so this bridge's rendered transcripts model legal wire bytes,
// matching every other fixture in this package (R-ATS-017/D3 normalization,
// W1's own created-field normalization).
func writeDeltaChunk(buf *bytes.Buffer, id, model, content string) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), bridgeChunkCreated, chunkObjectDiscriminator, bridgeQuoteJSONString(content))
}

// writeTerminalChunk appends one SSE data frame carrying choice-0's
// terminal finish_reason with an empty delta (C2's own convention). The
// object discriminator (C1) is set the same way writeDeltaChunk sets it.
func writeTerminalChunk(buf *bytes.Buffer, id, model string, reason ai.FinishReason) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"created\":%d,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":%s}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), bridgeChunkCreated, chunkObjectDiscriminator, bridgeQuoteJSONString(reason.String()))
}

// bridgeQuoteJSONString renders s as one JSON string literal, escaping only
// the bytes JSON syntax itself requires (the quote, the backslash, and
// control characters below 0x20) and passing every other byte through
// verbatim — deliberately NOT encoding/json.Marshal, which substitutes
// U+FFFD for invalid UTF-8 and would repair exactly the split-rune bytes
// this bridge must reproduce unmodified (S-ATS-041: "the bridge itself does
// not repair encoding").
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

// TestConformanceBridge_StreamingText covers S-ATS-040/041: the R-CNF-023
// capability-scoped conformance entry point, run over CapStreamingText
// only, against this bridge's real *Client speaking real HTTP to a local
// httptest.Server — the whole-registry agenttest.RunConformance is never
// invoked. This is the slice's own exit criterion: the AI-23.2 text
// conformance cases (textOrderingCase, textEmptyCompletionCase) passing
// against real transport for the first time.
func TestConformanceBridge_StreamingText(t *testing.T) {
	t.Parallel()

	agenttest.RunConformanceFor(t, conformanceBridgeFactory(), agenttest.CapStreamingText)
}

// TestConformanceBridge_ToolCalls covers S-ATL-059 (the AI-30.2 slice's
// exit criterion, R-ATL-012): all four CapToolCalls cases run through
// the capability-scoped entry point against this bridge's real *Client
// speaking real HTTP to a local httptest.Server.
//
// Cases 2 (zero_delta_whole_call_accepted) and 4
// (mixed_text_and_tool_ends_on_tool_call_finish_reason) require
// reconstruction-based assertions rather than exact drained-kind-count:
// cachicamas-ai-conformance-tool-amendment converted those assertions
// to requireRelativeKindOrder (R-CNF-019's boundary case for
// fragmentable argument channels), so all four cases now pass under
// the unscoped RunConformanceFor (no case-restricted invocation
// needed).
func TestConformanceBridge_ToolCalls(t *testing.T) {
	t.Parallel()

	agenttest.RunConformanceFor(t, conformanceBridgeFactory(), agenttest.CapToolCalls)
}

// bridgeReconstructedCall mirrors agenttest's reconstructedCall shape
// (R-ATC-012: ordinal derived stream-side) but lives in the openaicompat
// test package — the bridge's case-1 and case-3 tests use this local
// shape rather than reading agenttest's unexported type.
type bridgeReconstructedCall struct {
	id, name         string
	arguments        []byte
	fromDeltas       []byte
	ordinal          int
	sawStart, sawEnd bool
}

// bridgeReconstructToolCalls walks events once and rebuilds every tool
// call, keyed by block index. Mirrors agenttest.reconstructToolCalls
// (R-ATC-012's ordinal derivation).
func bridgeReconstructToolCalls(events []ai.Event) map[ai.BlockIndex]*bridgeReconstructedCall {
	out := make(map[ai.BlockIndex]*bridgeReconstructedCall)
	nextOrdinal := 1
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			s, ok := ev.ToolCallStart()
			if !ok {
				continue
			}
			out[s.BlockIndex()] = &bridgeReconstructedCall{id: s.ID(), name: s.Name(), sawStart: true, ordinal: nextOrdinal}
			nextOrdinal++
		case ai.EventKindToolCallDelta:
			d, ok := ev.ToolCallDelta()
			if !ok {
				continue
			}
			if call, exists := out[d.BlockIndex()]; exists {
				call.fromDeltas = append(call.fromDeltas, d.Fragment()...)
			}
		case ai.EventKindToolCallEnd:
			e, ok := ev.ToolCallEnd()
			if !ok {
				continue
			}
			if call, exists := out[e.BlockIndex()]; exists {
				call.arguments = e.Arguments()
				call.sawEnd = true
			}
		}
	}
	return out
}

// bridgeMinimalRequest builds a minimal valid ai.Request for the
// bridge's case tests — one user message, one text part.
func bridgeMinimalRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	req, err := ai.NewRequest("model", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest: %v", err)
	}
	return req
}

// bridgeDrainAndRecord reads every event off ch until close, with a
// bounded timeout. Wraps agenttest.DrainAndRecord with the default
// timeout (2s, AI-22.1's DefaultDrainTimeout).
func bridgeDrainAndRecord(t *testing.T, ch <-chan ai.Event) agenttest.Recording {
	t.Helper()
	return agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
}

// bridgeRequireValidStream asserts the recording satisfies
// ai.CheckStream's invariants — no unterminated blocks.
func bridgeRequireValidStream(t *testing.T, rec agenttest.Recording) {
	t.Helper()
	report := ai.CheckStream(rec.Events())
	if report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil", report.Violation())
	}
}
