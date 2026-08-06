package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// captureTransport is a test-only http.RoundTripper substitute that
// records the first request it sees and replies with a minimal
// `[DONE]`-terminated SSE transcript so a real openaicompat.Client.Stream
// drives all the way through to its terminal event. The
// wrapper-owned attributionRoundTripper sits between openaicompat and
// this captureTransport, so the body and header this transport records
// are exactly what the wrapper handed downstream — including any
// model override and attribution-header injection.
type captureTransport struct {
	mu          sync.Mutex
	captured    *http.Request
	capturedBody []byte
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.captured != nil {
		return nil, io.EOF // a single-request capture only
	}
	c.captured = req
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		c.capturedBody = b
		req.Body = io.NopCloser(bytes.NewReader(b))
	}
	h := make(http.Header)
	h.Set("Content-Type", "text/event-stream")
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body: io.NopCloser(strings.NewReader(
			"data: {\"id\":\"x\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ok\"},\"finish_reason\":null}]}\n\n" +
				"data: {\"id\":\"x\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n")),
		Header:  h,
		Request: req,
	}, nil
}

// snapshot returns the captured request's URL, header and body verbatim.
// Fails the test if no request was captured.
func (c *captureTransport) snapshot(t *testing.T) (string, http.Header, []byte) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.captured == nil {
		t.Fatal("captureTransport: no request was captured — the wrapper never drove a stream through the injected http.Client")
	}
	return c.captured.URL.String(), c.captured.Header.Clone(), append([]byte(nil), c.capturedBody...)
}

// validWrapperRequest builds a minimal valid ai.Request for end-to-end
// Stream calls — one user message, one text part. Mirrors openaicompat's
// own validRequest helper but lives here because that one is unexported.
func validWrapperRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	req, err := ai.NewRequest("requested-model", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}
	return req
}

// drainWrapperStream drives one stream from provider to terminal and
// returns every event captured on the channel. Bounded by drainTimeout
// so a producer defect does not hang the suite.
const wrapperDrainTimeout = 5 * time.Second

func drainWrapperStream(t *testing.T, ch <-chan ai.Event) []ai.Event {
	t.Helper()
	var events []ai.Event
	deadline := time.After(wrapperDrainTimeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline:
			t.Fatalf("drainWrapperStream: timed out after %s with %d event(s) so far", wrapperDrainTimeout, len(events))
			return events
		}
	}
}

// newWrapperWithStub builds a NewProvider with a captureTransport as the
// injected HTTPClient, returning the provider and the capture so each
// test can snapshot what the wrapper actually sent downstream.
func newWrapperWithStub(t *testing.T, cfg Config) (ai.ModelProvider, *captureTransport) {
	t.Helper()
	cap := &captureTransport{}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Transport: cap}
	}
	provider, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("NewProvider() error = %v, want nil", err)
	}
	return provider, cap
}

// TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o covers R-OR-03
// sub-scenario 1 + R-OR-05 sub-scenario 1: when Config.Model is the
// zero value, the wire body's "model" field equals "openai/gpt-4o".
// This is the runtime observable that
// TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment (1.8)
// reinforces at the constant-pin level.
func TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o(t *testing.T) {
	t.Parallel()

	provider, cap := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential("default-model-token"),
		// Model deliberately omitted — zero value, defaults to
		// openrouterDefaultModel via effectiveModel.
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	_, _, body := cap.snapshot(t)
	got := bodyModel(t, body)
	if got != openrouterDefaultModel {
		t.Errorf("wire body model = %q, want %q (R-OR-03 default)", got, openrouterDefaultModel)
	}
	if openrouterDefaultModel != "openai/gpt-4o" {
		t.Errorf("openrouterDefaultModel constant drifted from the pinned value (R-OR-05): got %q, want %q", openrouterDefaultModel, "openai/gpt-4o")
	}
}

// TestNewProvider_ConfigModelOverridesDefaultOnWireBody covers R-OR-03
// sub-scenario 2: when Config.Model is non-empty, the wire body's
// "model" field equals the caller's deliberate value, never the
// default.
func TestNewProvider_ConfigModelOverridesDefaultOnWireBody(t *testing.T) {
	t.Parallel()

	const wantModel = "anthropic/claude-3-5-sonnet"
	provider, cap := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential("explicit-model-token"),
		Model:      wantModel,
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	_, _, body := cap.snapshot(t)
	got := bodyModel(t, body)
	if got != wantModel {
		t.Errorf("wire body model = %q, want %q (R-OR-03 explicit override)", got, wantModel)
	}
}

// TestNewProvider_StreamOptionsIncludeUsageIsTrue covers R-OR-04: the
// wire body carries `"stream_options":{"include_usage":true}` exactly
// as openaicompat's body.go always renders it (R-ART-017). The
// wrapper's model override must not disturb this field.
func TestNewProvider_StreamOptionsIncludeUsageIsTrue(t *testing.T) {
	t.Parallel()

	provider, cap := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential("stream-options-token"),
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	_, _, body := cap.snapshot(t)
	if !bytesContain(body, []byte(`"stream_options":{"include_usage":true}`)) {
		t.Errorf("wire body missing stream_options.include_usage=true (R-OR-04): %s", body)
	}
}

// TestNewProvider_AttributionHeadersObservedEndToEnd covers R-OR-02
// end-to-end through NewProvider (the unit-level proof is in
// transport_test.go): every configured attribution value produces its
// header on the outbound request observed by the downstream endpoint.
func TestNewProvider_AttributionHeadersObservedEndToEnd(t *testing.T) {
	t.Parallel()

	const (
		wantReferer   = "https://example.test/app"
		wantXTitle    = "cachicamas-ai-cli"
		wantXCategory = "productivity,coding"
	)
	provider, cap := newWrapperWithStub(t, Config{
		Credential:  openaicompat.NewCredential("attribution-token"),
		HTTPReferer: wantReferer,
		XTitle:      wantXTitle,
		XCategories: wantXCategory,
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	_, header, _ := cap.snapshot(t)
	if got := header.Get("HTTP-Referer"); got != wantReferer {
		t.Errorf("HTTP-Referer = %q, want %q (R-OR-02 end-to-end)", got, wantReferer)
	}
	if got := header.Get("X-OpenRouter-Title"); got != wantXTitle {
		t.Errorf("X-OpenRouter-Title = %q, want %q (R-OR-02 end-to-end)", got, wantXTitle)
	}
	if got := header.Get("X-Title"); got != wantXTitle {
		t.Errorf("X-Title = %q, want %q (R-OR-02 end-to-end)", got, wantXTitle)
	}
	if got := header.Get("X-OpenRouter-Categories"); got != wantXCategory {
		t.Errorf("X-OpenRouter-Categories = %q, want %q (R-OR-02 end-to-end)", got, wantXCategory)
	}
}

// TestNewProvider_EmptyAttributionStringsSuppressAllHeaders covers
// R-OR-02 sub-scenario 2 end-to-end: empty attribution strings
// suppress their headers, never an empty-valued header.
func TestNewProvider_EmptyAttributionStringsSuppressAllHeaders(t *testing.T) {
	t.Parallel()

	provider, cap := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential("empty-attr-token"),
		// HTTPReferer, XTitle, XCategories all zero — empty.
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	_, header, _ := cap.snapshot(t)
	for _, name := range []string{
		"HTTP-Referer",
		"X-OpenRouter-Title",
		"X-Title",
		"X-OpenRouter-Categories",
	} {
		if got, present := header[name]; present {
			t.Errorf("%s present on outbound header = %v, want absent (R-OR-02 sub-scenario 2)", name, got)
		}
	}
}

// TestNewProvider_AuthorizationHeaderCarriesCredential covers the
// composition rule (R-APC-001, R-APC-013): the wrapper's integration
// with openaicompat.New must preserve the Authorization Bearer header
// openaicompat sets, alongside the wrapper's own attribution headers.
func TestNewProvider_AuthorizationHeaderCarriesCredential(t *testing.T) {
	t.Parallel()

	const token = "auth-bearer-token-shape"
	provider, cap := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential(token),
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	_, header, _ := cap.snapshot(t)
	const wantAuth = "Bearer " + token
	if got := header.Get("Authorization"); got != wantAuth {
		t.Errorf("Authorization = %q, want %q (R-APC-001 end-to-end)", got, wantAuth)
	}
}

// TestNewProvider_RejectsEmptyCredential covers the failure shape
// NewProvider hands back when openaicompat.New itself rejects a
// malformed configuration (R-OR-01, R-APC-002). With the endpoint
// hardcoded to openrouterBaseURL, the rejected-input shape is the
// empty credential — the same path openaicompat.New uses to reject
// at construction.
func TestNewProvider_RejectsEmptyCredential(t *testing.T) {
	t.Parallel()

	cap := &captureTransport{}
	_, err := NewProvider(Config{
		Credential: openaicompat.NewCredential(""), // empty — openaicompat rejects
		HTTPClient: &http.Client{Transport: cap},
	})
	if err == nil {
		t.Fatal("NewProvider() error = nil, want a typed violation for empty credential (R-OR-01, R-APC-002)")
	}
	if !strings.Contains(err.Error(), "credential") {
		t.Errorf("error %q does not name the credential position", err.Error())
	}
}

// TestNewProvider_RequestsTheConfiguredEndpoint covers R-OR-01
// sub-scenario 2: the outbound request goes to openrouterBaseURL, the
// hardcoded endpoint — no caller-injected URL ever escapes the wrapper.
func TestNewProvider_RequestsTheConfiguredEndpoint(t *testing.T) {
	t.Parallel()

	provider, cap := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential("endpoint-token"),
	})

	ch, err := provider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	url, _, _ := cap.snapshot(t)
	if want := "https://openrouter.ai/api/v1/chat/completions"; url != want {
		t.Errorf("outbound URL = %q, want %q (R-OR-01)", url, want)
	}
}

// bodyModel extracts the "model" field value from body, which is the
// JSON request body openaicompat's Translate produces (body.go's
// appendBody). json.Unmarshal into a map[string]any handles the
// nested stream_options without a hand-rolled parser.
func bodyModel(t *testing.T, body []byte) string {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("json.Unmarshal(body) error = %v, want nil — body = %s", err, body)
	}
	model, ok := parsed["model"].(string)
	if !ok {
		t.Fatalf("body has no string \"model\" field: %s", body)
	}
	return model
}

// bytesContain reports whether haystack contains needle as a substring.
func bytesContain(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}