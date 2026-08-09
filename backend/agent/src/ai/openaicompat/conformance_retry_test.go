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

// retryCaseBodyDriverFactory builds the Factory TestRetryCaseBody_RunsDirectly
// drives the case body with. Retry is declared from
// conformancetest.RetryOffered (AI-38 WU11 WARN-A remediation, R-ACR-006):
// this file lives in the external package openaicompat_test, which already
// imports conformancetest (line above) with no import-cycle obstacle — unlike
// bridge_test.go's internal test file, this declaration site CAN read the
// shared constant directly, so it does. Before this fix, a THIRD,
// independent retry declaration here was a bare local boolean, hardcoded
// true, that verify-report.md's round-2 pass (WARN-A) found unguarded:
// flipping that local value to false silently converted 5 of the 7
// sub-cases below from PASS to SKIP
// (they're gated behind `if !retryOffered { t.Skipf(...) }` inside
// RetryAutoRetryUpToBoundCase) with the overall test still exiting 0 and
// nothing named. Reading conformancetest.RetryOffered instead makes that
// drift structurally impossible, and
// TestRetryCaseBodyDriverFactory_RetryDeclaration_MatchesSharedSource below
// plus retry_parity_test.go's own extended guard both catch a future
// reintroduction of a local literal here.
func retryCaseBodyDriverFactory() agenttest.Factory {
	retryOffered := conformancetest.RetryOffered
	return agenttest.Factory{
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
}

// TestRetryCaseBody_RunsDirectly drives conformancetest.RetryAutoRetryUpToBoundCase
// against a Factory whose Retry declaration is true. The case body is a
// t.Run block of sub-tests, so this is effectively an explicit invocation
// of the sub-tests S-CNF-069..075.
func TestRetryCaseBody_RunsDirectly(t *testing.T) {
	conformancetest.RetryAutoRetryUpToBoundCase(t, retryCaseBodyDriverFactory())
}

// TestRetryCaseBodyDriverFactory_RetryDeclaration_MatchesSharedSource proves
// R-ACR-006's parity requirement holds for this third declaration site too
// (AI-38 WU11 WARN-A remediation): its constructed Retry pointer
// dereferences to exactly conformancetest.RetryOffered, the single declared
// source every conformance factory wrapping *openaicompat.Client must agree
// with — mirrors
// TestConformanceBridgeFactory_RetryDeclaration_MatchesSharedSource
// (openrouter/conformance/bridge_test.go) exactly.
func TestRetryCaseBodyDriverFactory_RetryDeclaration_MatchesSharedSource(t *testing.T) {
	t.Parallel()

	factory := retryCaseBodyDriverFactory()

	if factory.Retry == nil {
		t.Fatal("retryCaseBodyDriverFactory().Retry is nil (R-CNF-002, S-CNF-006) — declaration is by omission")
	}
	if *factory.Retry != conformancetest.RetryOffered {
		t.Errorf("retryCaseBodyDriverFactory().Retry = %v, want %v (conformancetest.RetryOffered) — every conformance factory wrapping *openaicompat.Client must declare retry identically (R-ACR-006)", *factory.Retry, conformancetest.RetryOffered)
	}
}

func boolPtrFor(b bool) *bool { return &b }
