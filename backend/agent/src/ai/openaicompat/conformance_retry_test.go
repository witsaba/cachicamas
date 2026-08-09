// AI-35 / AI-38 — CapRetry conformance case body (R-CNF-019, S-CNF-069..075).
//
// The case body itself now lives in the non-test package conformancetest
// (design D2, tasks.md Phase 5 WU6): it needs to be importable — a blank
// import suffices — by any conformance test binary that wants to drive
// it, including the OpenRouter conformance bridge's unscoped run, which a
// _test.go file's init() can never reach outside its own binary. See
// conformancetest/retry.go's own package doc for the full rationale.
//
// TestRetryCaseBody_RunsDirectly stays here as the thin white-box driver:
// it exists because the case body's init() registration only fires when a
// binary that imports conformancetest is built, and the agenttest test
// binary (which owns the public suite runner) cannot import the case body
// without creating an import cycle. This direct call proves every
// sub-test inside the case body holds against an in-process Factory + a
// real *openaicompat.Client speaking real HTTP.
package openaicompat_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest"
)

// TestRetryCaseBody_RunsDirectly drives conformancetest.RetryAutoRetryUpToBoundCase
// against a Factory whose Retry declaration is true. The case body is a
// t.Run block of sub-tests, so this is effectively an explicit invocation
// of the sub-tests S-CNF-069..075.
func TestRetryCaseBody_RunsDirectly(t *testing.T) {
	retryOffered := true
	factory := agenttest.Factory{
		New: func(_ testing.TB, _ ...agenttest.Script) ai.ModelProvider {
			// The case body bypasses f.New; it constructs its own
			// subject (design.md D11). Returning nil keeps the
			// Factory's contract total (New must be set) without
			// injecting a fake the case body would never use.
			return nil
		},
		Reasoning:     boolPtrFor(false),
		TokenCounting: boolPtrFor(false),
		CacheBoundary: boolPtrFor(false),
		Retry:         &retryOffered,
	}
	conformancetest.RetryAutoRetryUpToBoundCase(t, factory)
}

func boolPtrFor(b bool) *bool { return &b }
