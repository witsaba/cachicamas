// AI-20.1, AI-20.5 — the provider interface, proven from another package.
//
// Lives here rather than in ai_test, for request_test.go's reason restated:
// the milestone's acceptance is implementability and discovery from another
// package, and agenttest is the package that already proves external
// readability for AI-06 and AI-10. mustSimpleRequest, stubProvider and
// stubProviderWithTokenCounter are this file's own fixtures, shared with
// provider_signature_guard_test.go in the same package.
package agenttest_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// mustSimpleRequest builds the smallest valid request this file's tests
// need: one model identity, one user text message (V-REQ-20, V-REQ-21).
func mustSimpleRequest(t *testing.T) ai.Request {
	t.Helper()

	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	request, err := ai.NewRequest("cachicamas-neutral-model-1", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return request
}

// modelProviderType is ai.ModelProvider's reflect.Type, resolved once so
// every assertion in this file reads the identical value.
func modelProviderType() reflect.Type {
	return reflect.TypeOf((*ai.ModelProvider)(nil)).Elem()
}

// AI-20.1 item 1 (R-AMP-001, the R-AMP-021 pin's mechanical half) —
// ModelProvider declares exactly one method, with the neutral parameter and
// result types (S-AMP-001…003).
func TestModelProviderInterface_MethodSet_ExactlyOneStreamMethod(t *testing.T) {
	t.Parallel()

	typ := modelProviderType()
	if typ.NumMethod() != 1 {
		t.Fatalf("ai.ModelProvider declares %d methods, want exactly 1 (R-AMP-001; R-AMP-021's pin requires no widening)", typ.NumMethod())
	}

	method := typ.Method(0)
	if method.Name != "Stream" {
		t.Fatalf("ai.ModelProvider's one method is named %q, want %q", method.Name, "Stream")
	}

	sig := method.Type
	if sig.NumIn() != 2 {
		t.Fatalf("Stream declares %d parameters, want exactly 2 (ctx, req)", sig.NumIn())
	}
	if wantCtx := reflect.TypeOf((*context.Context)(nil)).Elem(); sig.In(0) != wantCtx {
		t.Errorf("Stream's first parameter is %v, want %v", sig.In(0), wantCtx)
	}
	if wantReq := reflect.TypeOf(ai.Request{}); sig.In(1) != wantReq {
		t.Errorf("Stream's second parameter is %v, want %v", sig.In(1), wantReq)
	}

	if sig.NumOut() != 2 {
		t.Fatalf("Stream declares %d results, want exactly 2 (event stream, error)", sig.NumOut())
	}
	if wantChan := reflect.ChanOf(reflect.RecvDir, reflect.TypeOf(ai.Event{})); sig.Out(0) != wantChan {
		t.Errorf("Stream's first result is %v, want %v (a receive-only chan of ai.Event, V-STR-04)", sig.Out(0), wantChan)
	}
	if wantErr := reflect.TypeOf((*error)(nil)).Elem(); sig.Out(1) != wantErr {
		t.Errorf("Stream's second result is %v, want %v", sig.Out(1), wantErr)
	}
}

// stubProvider is the external-consumer proof that ai.ModelProvider is
// implementable from outside package ai (R-AMP-003, S-AMP-007…009): every
// method is exported and every type this stub names is exported and
// constructible, or this file would fail to compile. It advertises no
// optional capability, which AI-20.5 also uses to prove a required-only
// provider is fully conformant (R-AMP-020, S-AMP-055/056).
type stubProvider struct{}

func (stubProvider) Stream(_ context.Context, _ ai.Request) (<-chan ai.Event, error) {
	out := make(chan ai.Event)
	close(out)
	return out, nil
}

// modelProviderCompileProof fails to compile the moment ai.ModelProvider
// grows an unexported method or embeds an unexported interface: a struct
// declared here, outside package ai, could satisfy neither.
var _ ai.ModelProvider = stubProvider{}

// AI-20.1 items 2/3 (R-AMP-003, S-AMP-007/008) — a non-ai package
// implements, compiles, and is exercised through the interface.
func TestModelProviderInterface_MethodSet_ExternalStubImplementsCompilesAndIsExercised(t *testing.T) {
	t.Parallel()

	var provider ai.ModelProvider = stubProvider{}

	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("stubProvider.Stream returned %v, want no failure", err)
	}
	if ch == nil {
		t.Fatal("stubProvider.Stream returned a nil channel with a nil error")
	}

	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("drained %d events from stubProvider, want 0 (it advertises none)", count)
	}
}

// --- AI-20.5 — optional capabilities: TokenCounter (R-AMP-017…021) --------

// stubProviderWithTokenCounter advertises the one v1 optional capability by
// satisfying its contract, and by no other means (ai-minimum-capabilities
// §9): no field, no flag, no registration call. Embedding stubProvider
// keeps this file's one Stream implementation the single source of the
// required surface.
type stubProviderWithTokenCounter struct {
	stubProvider
}

func (stubProviderWithTokenCounter) CountTokens(_ context.Context, _ ai.Request) (ai.TokenCount, error) {
	return ai.Tokens(42), nil
}

var _ ai.ModelProvider = stubProviderWithTokenCounter{}
var _ ai.TokenCounter = stubProviderWithTokenCounter{}

// AI-20.5 item 1 (R-AMP-017/018, S-AMP-046…049) — discovery is asked of the
// provider value, through a plain type assertion, and nothing else: a
// provider that satisfies TokenCounter is found this way, usable
// immediately.
func TestModelProviderInterface_TokenCounter_DiscoveredWhenTheProviderValueSatisfiesIt(t *testing.T) {
	t.Parallel()

	var provider ai.ModelProvider = stubProviderWithTokenCounter{}

	counter, ok := provider.(ai.TokenCounter)
	if !ok {
		t.Fatal("type assertion to ai.TokenCounter reported false on a provider that implements it")
	}
	count, err := counter.CountTokens(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("CountTokens returned %v, want no failure", err)
	}
	if n, present := count.Count(); !present || n != 42 {
		t.Errorf("CountTokens() = (%d, present=%v), want (42, true)", n, present)
	}
}

// AI-20.5 item 1's converse, and items 2/3 (R-AMP-018/020, S-AMP-050/051/
// 055/056) — a provider that does not implement TokenCounter reports a
// clean absence through the same type assertion: not an error, not a zero
// value standing in for one. The same provider value — stubProvider, Phase
// 1's required-only stub — is otherwise fully conformant, already proven by
// TestModelProviderInterface_MethodSet_ExternalStubImplementsCompilesAndIsExercised.
func TestModelProviderInterface_TokenCounter_CleanAbsenceWhenTheProviderDoesNotAdvertiseIt(t *testing.T) {
	t.Parallel()

	var provider ai.ModelProvider = stubProvider{}

	counter, ok := provider.(ai.TokenCounter)
	if ok {
		t.Fatal("type assertion to ai.TokenCounter reported true on stubProvider, want false — it advertises nothing optional")
	}
	if counter != nil {
		t.Errorf("a false type assertion returned %#v, want nil (a clean absence, not a zero value standing in for one)", counter)
	}

	// S-AMP-052 — this test method IS the whole discovery mechanism: there
	// is no second, catalog- or configuration-driven way to ask. Nothing in
	// this file names a model identity, reads a configuration entry, or
	// consults a capability table anywhere; the type assertion on the
	// provider value, above, is the only door
	// (ai-minimum-capabilities/spec.md §9). That absence is structural — a
	// symbol that does not exist cannot be called — rather than provable by
	// a further runtime assertion here.
}
