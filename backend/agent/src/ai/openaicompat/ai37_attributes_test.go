// AI-37 (WU-5+WU-6a/b, attribute mapping + carriers) — R-AOB-004
// (S-AOB-012…018), R-AOB-005 usage-equality half (S-AOB-020, 021),
// R-AOB-006 (S-AOB-022…025), NFR-AOB-002, R-APC-016 (S-APC-082…084),
// R-APC-001 (S-APC-081). One row per § D3 key, driven through a real
// scripted stream against tracetest.Provider, plus the two-adapter
// discrimination proof and the terminal-failure omission proof.
package openaicompat_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// ai37FullTranscript carries a content delta, a terminal chunk (finish
// reason "stop"), and a trailing usage-only chunk exercising all four
// usage sub-counts — the one scripted stream every row in
// TestAI37_Attributes_PerKeyTable reads from.
const ai37FullTranscript = "" +
	"data: {\"id\":\"ai37-attr\",\"model\":\"ai37-full-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"ai37-attr\",\"model\":\"ai37-full-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: {\"id\":\"ai37-attr\",\"model\":\"ai37-full-model\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150,\"prompt_tokens_details\":{\"cached_tokens\":20,\"cache_write_tokens\":5},\"completion_tokens_details\":{\"reasoning_tokens\":10}}}\n\n" +
	"data: [DONE]\n\n"

// ai37AttrRequest builds a valid request naming model and, when
// maxTokens is present, WithMaxOutputTokens(maxTokens).
func ai37AttrRequest(t *testing.T, model string, maxTokens int, hasMaxTokens bool) ai.Request {
	t.Helper()
	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	var opts []ai.RequestOption
	if hasMaxTokens {
		opts = append(opts, ai.WithMaxOutputTokens(maxTokens))
	}
	req, err := ai.NewRequest(model, []ai.Message{msg}, opts...)
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}
	return req
}

// ai37AttrByKey returns the value of the first attribute in attrs whose
// key matches want, and whether it was found — the presence-typed read
// every per-key row needs (S-AOB-016).
func ai37AttrByKey(attrs []attribute.KeyValue, want string) (attribute.Value, bool) {
	for _, kv := range attrs {
		if string(kv.Key) == want {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

// ai37DrainAllPlain drains ch to close.
func ai37DrainAllPlain(t *testing.T, ch <-chan ai.Event) []ai.Event {
	t.Helper()
	return ai37DrainAll(t, ch)
}

// TestAI37_Attributes_PerKeyTable drives ai37FullTranscript once and
// asserts every one of the twelve § D3 keys this run allowlists, spelled
// exactly, with the value the normalized result carries (S-AOB-012…018,
// S-AOB-020, S-AOB-021).
func TestAI37_Attributes_PerKeyTable(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	server := ai37SSEServer(t, ai37FullTranscript)
	c := ai37MustClient(t, server.URL, provider)

	req := ai37AttrRequest(t, "ai37-full-model", 456, true)
	ch, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := ai37DrainAllPlain(t, ch)
	if err := provider.AssertAllEndedOnce(); err != nil {
		t.Fatalf("AssertAllEndedOnce() = %v, want nil", err)
	}
	if provider.Started() != 1 {
		t.Fatalf("Started() = %d, want 1", provider.Started())
	}

	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	attrs := spans[0].Attributes()

	// The twelve-key allowlist MUST be a subset with no thirteenth key
	// (S-AOB-012). This success run is expected to carry exactly eleven
	// of the twelve (every key except error.type, which only ever
	// appears on a failure path — S-AOB-025's own complementary case).
	wantKeys := map[string]bool{
		"gen_ai.system":                    true,
		"gen_ai.request.model":             true,
		"gen_ai.request.max_tokens":        true,
		"gen_ai.response.finish_reasons":   true,
		"gen_ai.usage.input_tokens":        true,
		"gen_ai.usage.output_tokens":       true,
		"gen_ai.usage.cache_read_tokens":   true,
		"gen_ai.usage.cache_write_tokens":  true,
		"http.response.status_code":        true,
		"retry.count":                      true,
		"stream.event_count":               true,
		"error.type":                       true,
	}
	for _, kv := range attrs {
		if !wantKeys[string(kv.Key)] {
			t.Errorf("recorded attribute key %q is not in the twelve-key § D3 allowlist (S-AOB-012)", kv.Key)
		}
	}

	t.Run("gen_ai.system", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "gen_ai.system")
		if !ok {
			t.Fatal("gen_ai.system absent, want present")
		}
		if v.AsString() != "openai_compat" {
			t.Errorf("gen_ai.system = %q, want %q (AD-7)", v.AsString(), "openai_compat")
		}
	})

	t.Run("gen_ai.request.model", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "gen_ai.request.model")
		if !ok {
			t.Fatal("gen_ai.request.model absent, want present")
		}
		if v.AsString() != "ai37-full-model" {
			t.Errorf("gen_ai.request.model = %q, want %q", v.AsString(), "ai37-full-model")
		}
	})

	t.Run("gen_ai.request.max_tokens", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "gen_ai.request.max_tokens")
		if !ok {
			t.Fatal("gen_ai.request.max_tokens absent, want present")
		}
		if v.AsInt64() != 456 {
			t.Errorf("gen_ai.request.max_tokens = %d, want 456", v.AsInt64())
		}
	})

	t.Run("gen_ai.response.finish_reasons", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "gen_ai.response.finish_reasons")
		if !ok {
			t.Fatal("gen_ai.response.finish_reasons absent, want present")
		}
		got := v.AsStringSlice()
		if len(got) != 1 || got[0] != "stop" {
			t.Errorf("gen_ai.response.finish_reasons = %v, want [stop]", got)
		}
	})

	usageCases := []struct {
		key  string
		want int64
	}{
		{"gen_ai.usage.input_tokens", 100},
		{"gen_ai.usage.output_tokens", 50},
		{"gen_ai.usage.cache_read_tokens", 20},
		{"gen_ai.usage.cache_write_tokens", 5},
	}
	for _, tc := range usageCases {
		t.Run(tc.key, func(t *testing.T) {
			v, ok := ai37AttrByKey(attrs, tc.key)
			if !ok {
				t.Fatalf("%s absent, want present", tc.key)
			}
			if v.AsInt64() != tc.want {
				t.Errorf("%s = %d, want %d", tc.key, v.AsInt64(), tc.want)
			}
		})
	}

	t.Run("http.response.status_code", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "http.response.status_code")
		if !ok {
			t.Fatal("http.response.status_code absent, want present (success path)")
		}
		if v.AsInt64() != http.StatusOK {
			t.Errorf("http.response.status_code = %d, want %d", v.AsInt64(), http.StatusOK)
		}
	})

	t.Run("retry.count", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "retry.count")
		if !ok {
			t.Fatal("retry.count absent, want present on every exit")
		}
		if v.AsInt64() != 0 {
			t.Errorf("retry.count = %d, want 0 (unretried)", v.AsInt64())
		}
	})

	t.Run("stream.event_count", func(t *testing.T) {
		v, ok := ai37AttrByKey(attrs, "stream.event_count")
		if !ok {
			t.Fatal("stream.event_count absent, want present")
		}
		if int(v.AsInt64()) != len(events) {
			t.Errorf("stream.event_count = %d, want %d (the number of events actually drained, S-AOB-021)", v.AsInt64(), len(events))
		}
		if len(events) == 0 {
			t.Fatal("drained zero events — the equality above would be vacuous")
		}
	})

	t.Run("error.type absent on success", func(t *testing.T) {
		if _, ok := ai37AttrByKey(attrs, "error.type"); ok {
			t.Error("error.type present on a successful terminal, want absent")
		}
	})
}

// TestAI37_Attributes_PresenceTyped covers S-AOB-016: a request carrying
// no max-output-token bound, and a completion carrying no cache-read
// count, report both keys ABSENT — never recorded as zero.
func TestAI37_Attributes_PresenceTyped(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	transcript := "" +
		"data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
		"data: [DONE]\n\n"
	server := ai37SSEServer(t, transcript)
	c := ai37MustClient(t, server.URL, provider)

	req := ai37AttrRequest(t, "m", 0, false) // no WithMaxOutputTokens applied
	ch, err := c.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	ai37DrainAllPlain(t, ch)

	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	attrs := spans[0].Attributes()

	if _, ok := ai37AttrByKey(attrs, "gen_ai.request.max_tokens"); ok {
		t.Error("gen_ai.request.max_tokens present, want absent (no bound was applied, S-AOB-016)")
	}
	if _, ok := ai37AttrByKey(attrs, "gen_ai.usage.cache_read_tokens"); ok {
		t.Error("gen_ai.usage.cache_read_tokens present, want absent (no cached_tokens in the usage chunk, S-AOB-016)")
	}
	// The unconditional counterpart still present, proving absence above
	// is not an artifact of nothing being recorded at all.
	if _, ok := ai37AttrByKey(attrs, "gen_ai.usage.input_tokens"); !ok {
		t.Error("gen_ai.usage.input_tokens absent, want present (S-AOB-016's own control)")
	}
}

// TestAI37_Attributes_TerminalFailureOmitsStatusCode covers R-AOB-006
// (S-AOB-022…025): on a terminal provider failure, http.response.status_code
// is absent, no status class appears under any allowlisted key, and
// error.type is present with a member of the closed nine-category
// vocabulary.
func TestAI37_Attributes_TerminalFailureOmitsStatusCode(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"type":"invalid_api_key"}}`)
	}))
	t.Cleanup(server.Close)
	c := ai37MustClient(t, server.URL, provider)

	_, err := c.Stream(context.Background(), ai37ValidRequest(t))
	if err == nil {
		t.Fatal("Stream() error = nil, want an authentication failure")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false; err = %v", err)
	}

	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	attrs := spans[0].Attributes()

	if _, ok := ai37AttrByKey(attrs, "http.response.status_code"); ok {
		t.Error("http.response.status_code present on a terminal failure, want absent (R-AOB-006)")
	}
	// 401's status CLASS is 4 (status/100): S-AOB-023 forbids that class
	// value appearing under ANY allowlisted key, not only the exact-code
	// one — this is the precise leak shape R-AOB-006 exists to forbid
	// (failure_map.go computes status/100 and discards the code; nothing
	// in this package may re-surface the class under a different name).
	const status401Class = 4
	for _, kv := range attrs {
		if kv.Value.Type() == attribute.INT64 && kv.Value.AsInt64() == status401Class {
			t.Errorf("attribute %q = %d — a bare status CLASS value must never appear under any allowlisted key (R-AOB-006, S-AOB-023)", kv.Key, kv.Value.AsInt64())
		}
	}

	v, ok := ai37AttrByKey(attrs, "error.type")
	if !ok {
		t.Fatal("error.type absent on a terminal failure, want present")
	}
	if v.AsString() != failure.Category().String() {
		t.Errorf("error.type = %q, want %q (the closed category vocabulary's own name)", v.AsString(), failure.Category().String())
	}
}

// TestAI37_Attributes_ContentTypeRefusalRecordsExactStatusCode covers the
// content-type-refusal half of R-AOB-006's split (D-3b, Judgment Day
// remediation, S-AOB-040). Unlike
// TestAI37_Attributes_TerminalFailureOmitsStatusCode above (whose
// retry.Loop-error exit never obtains a *http.Response at all), a
// response IS in hand on this exit -- stream.go's own content-type
// refusal branch has resp.StatusCode before it ever calls
// endSpanPreHandover -- so the exact code MUST be recorded, exactly as
// the success path records it, not omitted. Before this remediation no
// test asserted this row; the spec text itself claimed the opposite
// (blanket omission on every terminal-failure path), which is the
// accuracy defect this round also corrects.
func TestAI37_Attributes_ContentTypeRefusalRecordsExactStatusCode(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	// A 2xx status is required to reach the content-type check at all:
	// retry.Loop's own status mapping (failure_map.go's mapResponse)
	// hands back a *http.Response with a nil error only for a 2xx
	// response -- any other status is classified and returned as an
	// error from retry.Loop itself, taking the OTHER pre-handover exit
	// (the one TestAI37_Attributes_TerminalFailureOmitsStatusCode
	// covers). http.StatusAccepted is deliberately distinct from the
	// http.StatusOK used elsewhere in this milestone's own fixtures, so
	// this assertion cannot be satisfied by a coincidental default.
	const refusalStatus = http.StatusAccepted
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(refusalStatus)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(server.Close)
	c := ai37MustClient(t, server.URL, provider)

	_, err := c.Stream(context.Background(), ai37ValidRequest(t))
	if err == nil {
		t.Fatal("Stream() error = nil, want a non-streaming content-type refusal")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false; err = %v", err)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse", failure.Category())
	}

	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	attrs := spans[0].Attributes()

	v, ok := ai37AttrByKey(attrs, "http.response.status_code")
	if !ok {
		t.Fatal("http.response.status_code absent on the content-type-refusal exit, want present -- a response WAS in hand on this exit (R-AOB-006 D-3b split, S-AOB-040)")
	}
	if v.AsInt64() != refusalStatus {
		t.Errorf("http.response.status_code = %d, want %d (the exact refusal response's own code)", v.AsInt64(), refusalStatus)
	}
}

// TestAI37_Attributes_MidStreamFailureRecordsExactStatusCode is Judgment
// Day round 2's own falsifier for R-AOB-006 bullet 2's general clause
// (S-AOB-041). A POST-handover (mid-stream) terminal failure has the
// exact same "a response WAS obtained before the failure was recognized"
// shape the content-type-refusal exit above already covers -- run is
// only ever launched after a successful, content-type-accepted response
// (finalizeSpan's own doc comment, trace.go, cites resp.StatusCode as
// always in hand there), so http.response.status_code MUST be recorded
// on this exit too, exactly as the success path and the content-type
// refusal already record it -- not omitted the way the pre-handover
// retry.Loop-error exit correctly omits it (TestAI37_Attributes_
// TerminalFailureOmitsStatusCode above, a structurally different exit
// with no *http.Response at all). Before this round, finalizeSpan's
// failure branch returned before ever recording it, and no scenario
// asserted either its presence or its absence here -- which is why the
// divergence between the landed spec text and the shipped code went
// unnoticed.
func TestAI37_Attributes_MidStreamFailureRecordsExactStatusCode(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	// The in-band-error-frame shape: a real HTTP 200 response is obtained
	// and its Content-Type accepted (run is launched), and the stream
	// itself then terminates on a provider error frame -- a
	// deterministic, non-racy mid-stream failure that needs no
	// cancellation and is independent of the round-2 CRITICAL fix above
	// (this fixture opens no text block, so blockOpen stays false).
	server := ai37SSEServer(t, "data: {\"error\":{\"type\":\"invalid_request_error\",\"message\":\"bad request\"}}\n\n")
	c := ai37MustClient(t, server.URL, provider)

	ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	ai37DrainAllPlain(t, ch)

	spans := provider.Spans()
	if len(spans) != 1 {
		t.Fatalf("provider recorded %d span(s), want 1", len(spans))
	}
	attrs := spans[0].Attributes()

	if _, ok := ai37AttrByKey(attrs, "error.type"); !ok {
		t.Fatal("error.type absent on a mid-stream failure, want present -- this row's own control that the failure branch actually ran")
	}

	v, ok := ai37AttrByKey(attrs, "http.response.status_code")
	if !ok {
		t.Fatal("http.response.status_code absent on a post-handover (mid-stream) terminal failure, want present -- a response WAS obtained before the in-band error frame terminated the stream (R-AOB-006 bullet 2, S-AOB-041)")
	}
	if v.AsInt64() != http.StatusOK {
		t.Errorf("http.response.status_code = %d, want %d (the exact response run was launched with)", v.AsInt64(), http.StatusOK)
	}
}

// TestAI37_Attributes_CrossVendorSystemEquality covers S-AOB-018: two
// adapters constructed against two different vendor endpoints record the
// same gen_ai.system value — the dialect, not a vendor-derived one.
func TestAI37_Attributes_CrossVendorSystemEquality(t *testing.T) {
	t.Parallel()

	const doneOnly = "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

	providerA := tracetest.NewProvider()
	serverA := ai37SSEServer(t, doneOnly)
	clientA := ai37MustClient(t, serverA.URL, providerA)

	providerB := tracetest.NewProvider()
	serverB := ai37SSEServer(t, doneOnly)
	clientB := ai37MustClient(t, serverB.URL, providerB)

	chA, err := clientA.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("clientA.Stream() error = %v, want nil", err)
	}
	ai37DrainAllPlain(t, chA)

	chB, err := clientB.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("clientB.Stream() error = %v, want nil", err)
	}
	ai37DrainAllPlain(t, chB)

	spansA, spansB := providerA.Spans(), providerB.Spans()
	if len(spansA) != 1 || len(spansB) != 1 {
		t.Fatalf("recorded %d/%d span(s), want 1/1", len(spansA), len(spansB))
	}
	vA, okA := ai37AttrByKey(spansA[0].Attributes(), "gen_ai.system")
	vB, okB := ai37AttrByKey(spansB[0].Attributes(), "gen_ai.system")
	if !okA || !okB {
		t.Fatalf("gen_ai.system present = (%v, %v), want (true, true)", okA, okB)
	}
	if vA.AsString() != vB.AsString() {
		t.Errorf("gen_ai.system differs across vendor endpoints: %q vs %q, want equal (S-AOB-018)", vA.AsString(), vB.AsString())
	}
}

// TestAI37_Attributes_TwoAdaptersDiscriminateProviders covers S-APC-081:
// two adapters constructed with two different recording tracer providers
// each record exactly their own span, never the other's — the
// non-transport-injectable discrimination proof R-APC-001 requires in
// place of a round-trip-boundary observation.
func TestAI37_Attributes_TwoAdaptersDiscriminateProviders(t *testing.T) {
	t.Parallel()

	const doneOnly = "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

	providerA := tracetest.NewProvider()
	serverA := ai37SSEServer(t, doneOnly)
	clientA := ai37MustClient(t, serverA.URL, providerA)

	providerB := tracetest.NewProvider()
	serverB := ai37SSEServer(t, doneOnly)
	clientB := ai37MustClient(t, serverB.URL, providerB)

	chA, err := clientA.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("clientA.Stream() error = %v, want nil", err)
	}
	ai37DrainAllPlain(t, chA)

	if got := providerA.Started(); got != 1 {
		t.Errorf("providerA.Started() = %d, want 1 (clientB has not driven a request yet)", got)
	}
	if got := providerB.Started(); got != 0 {
		t.Errorf("providerB.Started() = %d, want 0 (clientB has not driven a request yet — proves no cross-adapter leakage)", got)
	}

	chB, err := clientB.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("clientB.Stream() error = %v, want nil", err)
	}
	ai37DrainAllPlain(t, chB)

	if got := providerA.Started(); got != 1 {
		t.Errorf("providerA.Started() = %d, want 1 (unchanged by clientB's own request)", got)
	}
	if got := providerB.Started(); got != 1 {
		t.Errorf("providerB.Started() = %d, want 1", got)
	}
}

// TestAI37_Attributes_NoProviderInjected_UsesNoOpDefault covers
// S-APC-082/S-APC-083 (R-APC-016): with no TracerProvider supplied,
// construction succeeds, the request completes normally, and — because
// there is no process-global telemetry registry in this package at all
// (R-AOB-001) — there is nothing for a caller to inspect except that
// behavior is unaffected. This is a construction-and-drive smoke proof;
// the fuller no-tracer/tracer behavioral-equivalence proof is R-AOB-009,
// covered in Phase 8.
func TestAI37_Attributes_NoProviderInjected_UsesNoOpDefault(t *testing.T) {
	t.Parallel()

	const doneOnly = "data: {\"id\":\"c\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	server := ai37SSEServer(t, doneOnly)
	c, err := openaicompat.New(openaicompat.Config{
		Endpoint:   server.URL,
		Credential: openaicompat.NewCredential("no-provider-token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil (S-APC-082)", err)
	}
	ch, err := c.Stream(context.Background(), ai37ValidRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := ai37DrainAllPlain(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want a normal completion")
	}
}
