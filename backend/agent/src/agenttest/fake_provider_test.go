// CH-02 — fake provider's script-queue visibility for "scripts queued but
// not yet consumed" (R-CCP-009 / S-CCP-082). The harness's own retry may
// observe a third script as available and choose not to take it; without
// ScriptsRemaining the no-archetype-retry assertion is unfalsifiable when
// the fixture has exactly two scripts, because the provider refuses to
// record an exhausted-queue Stream call (R-AFP-020, ErrScriptsExhausted
// never appends to p.requests). This file pairs with the
// (a) hksFilterOutCH02FakeProviderFiles release in hooks_test.go and
// (b) TestFailure_NoArchetypeRetry's third-script fixture in
// failure_test.go — both edits land together.
package agenttest_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustMinimalRequest builds a request the provider's pre-stream
// validation will accept (R-AFP-013, fake_provider.go:74-77). The test
// cares about the queue counter, not about the request's contents.
func mustMinimalRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("hi")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	req, err := ai.NewRequest("test-model", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return req
}

// TestProvider_ScriptsRemaining — R-CCP-009 / S-CCP-082. Given a freshly
// constructed Provider with three scripted Scripts, when ScriptsRemaining
// is read before any Stream call, then it returns 3; and given a single
// Stream call has consumed the first script, when ScriptsRemaining is
// read again, then it returns 2. The two readings force the method to
// reflect consumed scripts, not merely the queue length — the
// unfalsifiable-fixture shape that motivated this method.
func TestProvider_ScriptsRemaining(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(
		agenttest.Script{},
		agenttest.Script{},
		agenttest.Script{},
	)

	if got := provider.ScriptsRemaining(); got != 3 {
		t.Errorf("pre-Stream ScriptsRemaining = %d, want 3", got)
	}

	out, err := provider.Stream(context.Background(), mustMinimalRequest(t))
	if err != nil {
		t.Fatalf("first Stream returned %v, want no failure", err)
	}
	drained := 0
	for range out {
		drained++
	}
	if drained != 0 {
		t.Errorf("drained %d events from an empty Script, want 0", drained)
	}

	if got := provider.ScriptsRemaining(); got != 2 {
		t.Errorf("post-first-Stream ScriptsRemaining = %d, want 2", got)
	}
}
