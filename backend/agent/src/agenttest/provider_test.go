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
