// AG-17.2 — fixture physics for the exported counting-capable fake
// (design.md DD7): report, error, absent-count-with-nil-error, and
// capture order/mutation-safety. package agenttest_test, mirroring
// provider_test.go's external posture and reusing its
// mustSimpleRequest fixture.
package agenttest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustTaggedRequest builds a valid request distinguishable from
// mustSimpleRequest's by its model identity, so a capture-order
// assertion can tell the two apart without relying on Request's
// unexported layout.
func mustTaggedRequest(t *testing.T, model string) ai.Request {
	t.Helper()

	part, err := ai.NewText("tagged request body")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	request, err := ai.NewRequest(model, []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// TestCountingProvider_ReportsScriptedResultAndCapturesRequest proves
// NewCountingProvider's CountTokens reports exactly the scripted
// result and captures the request it was called with.
func TestCountingProvider_ReportsScriptedResultAndCapturesRequest(t *testing.T) {
	t.Parallel()

	want := ai.Tokens(123)
	provider := agenttest.NewCountingProvider(want)

	req := mustSimpleRequest(t)
	got, err := provider.CountTokens(context.Background(), req)
	if err != nil {
		t.Fatalf("CountTokens returned err = %v, want nil", err)
	}
	if n, present := got.Count(); !present || n != 123 {
		t.Errorf("CountTokens() = (%d, present=%v), want (123, true)", n, present)
	}

	captured := provider.CountRequests()
	if len(captured) != 1 {
		t.Fatalf("CountRequests() returned %d request(s), want exactly 1", len(captured))
	}
	if !captured[0].Equal(req) {
		t.Error("the captured request is not equal to the request CountTokens was called with")
	}
}

// TestCountingProvider_AbsentCountWithNilError proves a
// NewCountingProvider scripted with the ZERO ai.TokenCount reports an
// absent count together with a NIL error — the "advertised counter
// answers with a nil error and an absent count" fixture S-CTX-012's
// row (d) needs, distinct from NewFailingCountingProvider's row (c).
func TestCountingProvider_AbsentCountWithNilError(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewCountingProvider(ai.TokenCount{})

	got, err := provider.CountTokens(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("CountTokens returned err = %v, want nil (the fixture answers cleanly, just with no figure)", err)
	}
	if _, present := got.Count(); present {
		t.Error("CountTokens() reported a present count, want absent (nil error, no figure)")
	}
}

// TestNewFailingCountingProvider_ReturnsScriptedErrorAndStillCaptures
// proves the ADVERTISED-but-declining fixture: CountTokens returns
// the scripted error and still captures the request — an advertised
// counter that errs is non-conformant, not silent (R-AMP-019).
func TestNewFailingCountingProvider_ReturnsScriptedErrorAndStillCaptures(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("counting provider: scripted decline")
	provider := agenttest.NewFailingCountingProvider(wantErr)

	req := mustSimpleRequest(t)
	got, err := provider.CountTokens(context.Background(), req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("CountTokens returned err = %v, want %v", err, wantErr)
	}
	if _, present := got.Count(); present {
		t.Error("CountTokens() reported a present count alongside a non-nil error, want the zero TokenCount")
	}

	captured := provider.CountRequests()
	if len(captured) != 1 {
		t.Fatalf("CountRequests() returned %d request(s), want exactly 1 -- capture happens even on a scripted decline", len(captured))
	}
	if !captured[0].Equal(req) {
		t.Error("the captured request is not equal to the request CountTokens was called with")
	}
}

// TestCountingProvider_CountRequests_CaptureOrderAndFreshSlice proves
// two properties together: CountRequests() reflects call order, and
// it returns a FRESH slice on every call (fake_provider.go:157-161's
// Requests() posture) -- a caller that mutates what it received
// cannot corrupt the fixture's own record.
func TestCountingProvider_CountRequests_CaptureOrderAndFreshSlice(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewCountingProvider(ai.Tokens(7))

	first := mustSimpleRequest(t)
	second := mustTaggedRequest(t, "second-request-model")

	if _, err := provider.CountTokens(context.Background(), first); err != nil {
		t.Fatalf("CountTokens(first) returned %v, want nil", err)
	}
	if _, err := provider.CountTokens(context.Background(), second); err != nil {
		t.Fatalf("CountTokens(second) returned %v, want nil", err)
	}

	captured := provider.CountRequests()
	if len(captured) != 2 {
		t.Fatalf("CountRequests() returned %d request(s), want exactly 2", len(captured))
	}
	if !captured[0].Equal(first) || !captured[1].Equal(second) {
		t.Error("CountRequests() did not preserve call order")
	}

	// Mutate the returned slice's backing array; a second call MUST
	// still report the original two requests, unmutated.
	captured[0] = second
	again := provider.CountRequests()
	if !again[0].Equal(first) {
		t.Error("mutating a previously returned CountRequests() slice corrupted the fixture's own record -- CountRequests() must return a fresh slice every call")
	}
}

// TestCountingProvider_StreamStillDelegatesToEmbeddedProvider proves
// the pointer-embedding satisfies the REQUIRED ai.ModelProvider
// surface too, not only the optional ai.TokenCounter one (proposal
// risk 10: Stream is on the pointer receiver, fake_provider.go:74, so
// value embedding would not compile at all -- this is the runtime
// half of that compile-time guard).
func TestCountingProvider_StreamStillDelegatesToEmbeddedProvider(t *testing.T) {
	t.Parallel()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta, err := ai.NewTextDelta(1, "hi")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}

	provider := agenttest.NewCountingProvider(ai.Tokens(1), agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}})

	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned err = %v, want nil", err)
	}
	var count int
	for range ch {
		count++
	}
	if count != 4 {
		t.Errorf("drained %d event(s) from the embedded *Provider's Stream, want 4", count)
	}

	if got := len(provider.Requests()); got != 1 {
		t.Errorf("embedded *Provider.Requests() = %d, want 1 -- Stream must still be the SAME embedded fake's method", got)
	}
}

var _ ai.ModelProvider = (*agenttest.CountingProvider)(nil)
var _ ai.TokenCounter = (*agenttest.CountingProvider)(nil)
