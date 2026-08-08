// AI-37 (WU-9, no-op equivalence) — R-AOB-009 (S-AOB-036…039), NFR-AOB-001.
// With no tracer provider injected, behavior is byte-identical to a traced
// run, nothing panics, and no adapter-side nil check on a tracing value
// exists anywhere in this package — the structural claim that distinguishes
// "the API's no-op default suffices" (doc 0002:2234) from "every call site
// was guarded". (Judgment Day remediation: this file's own claim was
// previously "anywhere in Layer 1", wider than what the scan below proves;
// narrowed to match trace.go's own package-level doc comment, which makes
// the identical claim at package scope.)
package openaicompat_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// ai37EquivalenceTranscript exercises text, tool-call and usage mapping in
// one scripted stream — a reasonably rich fixture for the equivalence
// proof, so "identical" is not trivially true of an empty stream.
const ai37EquivalenceTranscript = "" +
	"data: {\"id\":\"ai37-eq\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi there\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"ai37-eq\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_eq\",\"function\":{\"name\":\"search\",\"arguments\":\"{\\\"q\\\":\\\"x\\\"}\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"ai37-eq\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
	"data: {\"id\":\"ai37-eq\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":8,\"completion_tokens\":3,\"total_tokens\":11}}\n\n" +
	"data: [DONE]\n\n"

// TestAI37_NoopEquivalence_DrainedSequencesEqual covers S-AOB-036: the
// identical scripted stream, run once with no tracer provider configured
// and once with a recording one, produces byte-identical drained event
// sequences, element for element.
func TestAI37_NoopEquivalence_DrainedSequencesEqual(t *testing.T) {
	t.Parallel()

	serverPlain := ai37SSEServer(t, ai37EquivalenceTranscript)
	clientPlain, err := openaicompat.New(openaicompat.Config{
		Endpoint:   serverPlain.URL,
		Credential: openaicompat.NewCredential("eq-token"),
	})
	if err != nil {
		t.Fatalf("New() (no provider) error = %v, want nil", err)
	}
	chPlain, err := clientPlain.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() (no provider) error = %v, want nil", err)
	}
	eventsPlain := agenttest.DrainAndRecord(t, chPlain, agenttest.DefaultDrainTimeout).Events()

	serverTraced := ai37SSEServer(t, ai37EquivalenceTranscript)
	provider := tracetest.NewProvider()
	clientTraced, err := openaicompat.New(openaicompat.Config{
		Endpoint:       serverTraced.URL,
		Credential:     openaicompat.NewCredential("eq-token"),
		TracerProvider: provider,
	})
	if err != nil {
		t.Fatalf("New() (recording provider) error = %v, want nil", err)
	}
	chTraced, err := clientTraced.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() (recording provider) error = %v, want nil", err)
	}
	eventsTraced := agenttest.DrainAndRecord(t, chTraced, agenttest.DefaultDrainTimeout).Events()

	if len(eventsPlain) == 0 {
		t.Fatal("drained zero events, want a non-trivial scripted stream (the equality below would be vacuous)")
	}
	agenttest.RequireSameEvents(t, eventsTraced, eventsPlain)
}

// TestAI37_NoopEquivalence_NoTracerConfigured_NoPanicAcrossTerminalPaths
// covers S-AOB-037: every one of the 15 terminal-path shapes
// ai37_span_lifecycle_test.go proves span-correct WITH a tracer also
// completes, with no tracer configured, without panicking and with the
// same outcome shape (error or nil) it produces today.
func TestAI37_NoopEquivalence_NoTracerConfigured_NoPanicAcrossTerminalPaths(t *testing.T) {
	t.Parallel()

	mustNoTracerClient := func(t *testing.T, endpoint string) *openaicompat.Client {
		t.Helper()
		c, err := openaicompat.New(openaicompat.Config{
			Endpoint:   endpoint,
			Credential: openaicompat.NewCredential("no-tracer-token"),
		})
		if err != nil {
			t.Fatalf("New() error = %v, want nil", err)
		}
		return c
	}

	runNoPanic := func(t *testing.T, drive func(t *testing.T)) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panicked with no tracer configured: %v (S-AOB-037 forbids this)", r)
			}
		}()
		drive(t)
	}

	cases := []struct {
		name  string
		drive func(t *testing.T)
	}{
		{"success_done", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"completion_build_failure", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: [DONE]\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"in_band_error", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {\"error\":{\"type\":\"invalid_request_error\",\"message\":\"bad request\"}}\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"malformed_chunk_json", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {not-valid-json\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"applychunk_failure_malformed_identity", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {\"id\":\"\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"emit_race_cancellation", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n").URL)
			ctx, cancel := context.WithCancel(context.Background())
			ch, err := c.Stream(ctx, ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			cancel()
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"feed_error_frame_too_large", func(t *testing.T) {
			huge := "data: " + strings.Repeat("x", 8*1024*1024+64)
			c := mustNoTracerClient(t, ai37SSEServer(t, huge).URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, ai37HeavyDrainTimeout)
		}},
		{"eof_finish_error_truncated", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {\"id\":\"c\"").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"eof_incomplete_no_sentinel", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"mid_stream_ctx_cancellation", func(t *testing.T) {
			server, release := ai37SlowSSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
			_ = release
			c := mustNoTracerClient(t, server.URL)
			ctx, cancel := context.WithCancel(context.Background())
			ch, err := c.Stream(ctx, ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			<-ch
			cancel()
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"read_error_abrupt_close", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37AbruptCloseServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n").URL)
			ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		}},
		{"pre_handover_retry_exhausted", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37AlwaysStatusServer(t, 503).URL)
			if _, err := c.Stream(context.Background(), ai37ValidRequest(t)); err == nil {
				t.Fatal("Stream() error = nil, want a retry-exhausted failure")
			}
		}},
		{"pre_handover_non_retryable", func(t *testing.T) {
			c := mustNoTracerClient(t, ai37AlwaysStatusServer(t, 401).URL)
			if _, err := c.Stream(context.Background(), ai37ValidRequest(t)); err == nil {
				t.Fatal("Stream() error = nil, want a non-retryable failure")
			}
		}},
		{"pre_handover_cancelled_in_loop", func(t *testing.T) {
			// W-3 remediation (sdd-verify): this cancellation shape was
			// the one row missing from this table -- the file's own doc
			// comment already claimed 15, matching
			// ai37_span_lifecycle_test.go's own identically-named case,
			// which this mirrors with mustNoTracerClient in place of
			// ai37MustClient (S-AOB-037: no tracer configured).
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"error":{"type":"scripted_failure"}}`)
			}))
			t.Cleanup(server.Close)
			c := mustNoTracerClient(t, server.URL)
			ctx, cancel := context.WithCancel(context.Background())
			// The server always answers retryably; cancelling shortly
			// after Stream starts lands inside retry.Loop's own
			// ctx.Err() check between attempts, well before its real
			// backoff budget would otherwise exhaust (avoiding a
			// multi-second real sleep in this test) -- the same timing
			// rationale ai37_span_lifecycle_test.go's own
			// pre_handover_cancelled_in_loop case uses.
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()
			if _, err := c.Stream(ctx, ai37ValidRequest(t)); err == nil {
				t.Fatal("Stream() error = nil, want the last observed failure")
			}
		}},
		{"pre_handover_content_type_refusal", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, `{"ok":true}`)
			}))
			t.Cleanup(server.Close)
			c := mustNoTracerClient(t, server.URL)
			if _, err := c.Stream(context.Background(), ai37ValidRequest(t)); err == nil {
				t.Fatal("Stream() error = nil, want a non-streaming content-type refusal")
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runNoPanic(t, tc.drive)
		})
	}
}

// ai37TracingNilCheckPatterns are the exact substrings a nil-check-guarded
// recording site would contain — checked against "span" and "tracer" (the
// derived, per-request values every recording site consumes), never
// against "TracerProvider" (the Config field / construction-time
// injectable). The one-time defaulting check in client.go's New
// ("if tracerProvider == nil { tracerProvider = noop.NewTracerProvider() }",
// R-APC-016) does not match any of these patterns: "tracerProvider" is not
// "tracer" followed immediately by " == nil"/" != nil" — the identifier is
// different, not merely a superstring, so the exclusion is structural
// (a property of the patterns chosen) rather than a name it has to special-
// case.
var ai37TracingNilCheckPatterns = []string{
	"span == nil",
	"span != nil",
	"tracer == nil",
	"tracer != nil",
}

// scanTracingNilChecks returns one description per line of src whose
// trimmed text contains any of ai37TracingNilCheckPatterns — a pure,
// reusable function so both the real-file assertion (S-AOB-038) and the
// staged-mutation falsifiability control (S-AOB-039) call the identical
// scanning logic, the same discipline openrouter/charter_test.go's own
// scanFileForLiteral establishes for a comparable guard.
func scanTracingNilChecks(label string, src []byte) []string {
	var hits []string
	for i, line := range bytes.Split(src, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		for _, pattern := range ai37TracingNilCheckPatterns {
			if strings.Contains(trimmed, pattern) {
				hits = append(hits, label+":"+itoaAI37(i+1)+": "+trimmed)
			}
		}
	}
	return hits
}

// itoaAI37 formats n as decimal — a local helper avoiding a strconv import
// for one call site (mirroring openrouter/charter_test.go's own itoa).
func itoaAI37(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ai37IsPackageSourceFile reports whether name is a non-test Go source
// file this scan admits: a ".go" file that is not a "_test.go" file.
// This is the same file-selection rule
// ambient_authority_test.go's own isAdapterSourceFile establishes,
// restated here because that helper lives in package openaicompat
// (internal test) and this file is package openaicompat_test
// (external) — the two packages share no unexported symbols.
func ai37IsPackageSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// TestAI37_NoopEquivalence_NoNilCheckOnTracingValues covers S-AOB-038: no
// non-test source file in this package carries a nil check guarding a
// span or a tracer value.
//
// Judgment Day remediation: the pre-remediation scan named exactly three
// files (client.go, stream.go, trace.go) — narrower than both this
// package's own trace.go doc comment ("no adapter-side nil check on a
// tracing value exists anywhere in this package") and this file's own
// header comment (previously "anywhere in Layer 1", now narrowed to
// match). Every non-test source file in the package directory is
// scanned now, not a hardcoded list, so a file this milestone's authors
// did not anticipate is covered without a guard update.
func TestAI37_NoopEquivalence_NoNilCheckOnTracingValues(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\") error = %v, want nil", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !ai37IsPackageSourceFile(name) {
			continue
		}
		scanned++
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", name, err)
		}
		hits := scanTracingNilChecks(name, raw)
		for _, h := range hits {
			t.Errorf("nil check on a tracing value: %s (R-AOB-009/S-AOB-038 forbids this)", h)
		}
	}
	// Non-vacuity floor for the WIDENING itself (distinct from
	// scanTracingNilChecks's own falsifiability, already proven by
	// TestAI37_NoopEquivalence_NilCheckScan_DetectsStagedMutation below):
	// this package carries over twenty non-test .go files today,
	// comfortably above this conservative floor, so a directory-listing
	// regression that silently narrowed the scan back down would trip
	// this assertion rather than pass unnoticed.
	const wantMinScannedFiles = 10
	if scanned < wantMinScannedFiles {
		t.Fatalf("scanned %d non-test source file(s), want at least %d — the package-wide claim would be vacuous if the directory listing came back empty or over-filtered", scanned, wantMinScannedFiles)
	}
}

// TestAI37_NoopEquivalence_NilCheckScan_DetectsStagedMutation covers
// S-AOB-039: the scan above is falsifiable, not merely green-from-birth.
// A staged mutation — one nil-check-guarded recording site — is written to
// an isolated t.TempDir() file (never the real worktree; the same staged-
// mutation discipline openrouter/charter_test.go's own
// TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation
// establishes) and scanned with the identical function used above.
func TestAI37_NoopEquivalence_NilCheckScan_DetectsStagedMutation(t *testing.T) {
	t.Parallel()

	mutation := "package openaicompat\n\nfunc mutated(span trace.Span) {\n\tif span != nil {\n\t\tspan.SetAttributes()\n\t}\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "staged_nil_check_mutation.go")
	if err := os.WriteFile(path, []byte(mutation), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v, want nil", err)
	}
	hits := scanTracingNilChecks(path, raw)
	if len(hits) == 0 {
		t.Fatal("scanTracingNilChecks found no violation in a deliberately nil-check-guarded staged mutation — the guard would be vacuous (S-AOB-039)")
	}
	found := false
	for _, h := range hits {
		if strings.Contains(h, "span != nil") {
			found = true
		}
	}
	if !found {
		t.Errorf("hits = %v, want one naming \"span != nil\"", hits)
	}
}

// TestAI37_NoopEquivalence_CorpusFromNoTracerRunIsUntouched is a small
// structural sanity check tying S-AOB-036/037 to the recording provider
// used everywhere else in this milestone's tests: a *tracetest.Provider
// never associated with a client records nothing, regardless of how many
// requests other, untraced clients drive — the same one-way-only property
// TestAI37_Attributes_TwoAdaptersDiscriminateProviders already proves for
// two DIFFERENT providers.
func TestAI37_NoopEquivalence_CorpusFromNoTracerRunIsUntouched(t *testing.T) {
	t.Parallel()

	bystander := tracetest.NewProvider()

	server := ai37SSEServer(t, "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:   server.URL,
		Credential: openaicompat.NewCredential("bystander-token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	ch, err := client.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)

	if got := bystander.Started(); got != 0 {
		t.Errorf("an unrelated provider recorded %d span(s) from a client it was never injected into, want 0", got)
	}
	if got := len(bystander.Corpus()); got != 0 {
		t.Errorf("an unrelated provider's Corpus() = %d bytes, want 0", got)
	}
}
