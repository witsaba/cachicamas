// AI-37 Judgment Day remediation (item 2) — R-APC-016/S-APC-085.
//
// Before this fix, this wrapper's own Config had no TracerProvider
// field, and NewProvider left openaicompat.Config's own field at its
// zero value, so the only shipped concrete provider was permanently
// untraceable -- while client.go's own Config.TracerProvider doc
// comment claims that field "is the only door a tracer reaches Layer 1
// through". That door was closed for every OpenRouter caller. This file
// proves the fix: a tracer provider injected at this wrapper's own
// construction surface reaches the span the underlying
// openaicompat.Client records, exactly as it would through
// openaicompat.New directly.
package openrouter

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// TestNewProvider_InjectedTracerProviderRecordsTheSpan covers S-APC-085:
// a tracer provider injected through openrouter.Config.TracerProvider
// reaches Layer 1 exactly as client.go's own Config.TracerProvider door
// does -- the underlying openaicompat.Client's span is recorded by the
// provider this wrapper's own caller injected, not lost at the wrapper
// boundary.
func TestNewProvider_InjectedTracerProviderRecordsTheSpan(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	wrapperProvider, _ := newWrapperWithStub(t, Config{
		Credential:     openaicompat.NewCredential("tracer-door-token"),
		TracerProvider: provider,
	})

	ch, err := wrapperProvider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, ch)

	if got := provider.Started(); got != 1 {
		t.Fatalf("provider.Started() = %d, want 1 -- the injected tracer provider must record the wrapper's own span (R-APC-016/S-APC-085)", got)
	}
	if err := provider.AssertAllEndedOnce(); err != nil {
		t.Errorf("AssertAllEndedOnce() = %v, want nil", err)
	}
}

// TestNewProvider_TwoWrappersDiscriminateInjectedProviders covers the
// same S-APC-081 discrimination shape R-APC-001 requires for a
// non-transport injectable, driven through this wrapper's own
// construction surface: two wrappers built with two different recording
// tracer providers each record exactly their own span.
func TestNewProvider_TwoWrappersDiscriminateInjectedProviders(t *testing.T) {
	t.Parallel()

	providerA := tracetest.NewProvider()
	wrapperA, _ := newWrapperWithStub(t, Config{
		Credential:     openaicompat.NewCredential("tracer-door-token-a"),
		TracerProvider: providerA,
	})
	providerB := tracetest.NewProvider()
	wrapperB, _ := newWrapperWithStub(t, Config{
		Credential:     openaicompat.NewCredential("tracer-door-token-b"),
		TracerProvider: providerB,
	})

	chA, err := wrapperA.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("wrapperA.Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, chA)

	if got := providerA.Started(); got != 1 {
		t.Errorf("providerA.Started() = %d, want 1", got)
	}
	if got := providerB.Started(); got != 0 {
		t.Errorf("providerB.Started() = %d, want 0 (wrapperB has not driven a request yet -- proves no cross-adapter leakage)", got)
	}

	chB, err := wrapperB.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("wrapperB.Stream() error = %v, want nil", err)
	}
	drainWrapperStream(t, chB)

	if got := providerA.Started(); got != 1 {
		t.Errorf("providerA.Started() = %d, want 1 (unchanged by wrapperB's own request)", got)
	}
	if got := providerB.Started(); got != 1 {
		t.Errorf("providerB.Started() = %d, want 1", got)
	}
}

// TestNewProvider_NoTracerProviderInjected_UsesNoOpDefault covers the
// defaulting half: when Config.TracerProvider is left zero (today's
// common case), construction and streaming still succeed --
// openaicompat.New's own no-op substitution (R-APC-016) covers this
// wrapper's callers exactly as it covers any other openaicompat.Config
// caller, with no wrapper-side special case needed.
func TestNewProvider_NoTracerProviderInjected_UsesNoOpDefault(t *testing.T) {
	t.Parallel()

	wrapperProvider, _ := newWrapperWithStub(t, Config{
		Credential: openaicompat.NewCredential("no-tracer-door-token"),
		// TracerProvider deliberately omitted -- zero value.
	})

	ch, err := wrapperProvider.Stream(context.Background(), validWrapperRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainWrapperStream(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want a normal completion")
	}
}
