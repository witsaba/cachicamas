// AI-28.1.2 — R-ATS-011: AI-23.2's text conformance cases pass against real
// transport (S-ATS-040…043). Test-only, adapter-owned bridge — lives only
// in this package's _test.go files (S-ATS-042); no file under src/agenttest
// is read for anything but its two landed, exported surfaces:
// agenttest.RunConformanceFor (R-CNF-023) and Script.Steps/Step.Event/
// Step.IsHold (R-CNF-024). No unexported agenttest state is reached
// anywhere in this file.
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
		default:
			tb.Fatalf("bridge: renderScript: unsupported event kind %v — outside CapStreamingText scope", ev.Kind())
			return nil
		}
	}

	return buf.Bytes()
}

// writeDeltaChunk appends one SSE data frame carrying a choice-0 content
// delta (R-ATS-007). The object discriminator (C1) is set to the chunk
// spelling so this bridge's rendered transcripts model legal wire bytes,
// matching every other fixture in this package (R-ATS-017/D3 normalization).
func writeDeltaChunk(buf *bytes.Buffer, id, model, content string) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), chunkObjectDiscriminator, bridgeQuoteJSONString(content))
}

// writeTerminalChunk appends one SSE data frame carrying choice-0's
// terminal finish_reason with an empty delta (C2's own convention). The
// object discriminator (C1) is set the same way writeDeltaChunk sets it.
func writeTerminalChunk(buf *bytes.Buffer, id, model string, reason ai.FinishReason) {
	fmt.Fprintf(buf, "data: {\"id\":%s,\"model\":%s,\"object\":%q,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":%s}]}\n\n",
		bridgeQuoteJSONString(id), bridgeQuoteJSONString(model), chunkObjectDiscriminator, bridgeQuoteJSONString(reason.String()))
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
