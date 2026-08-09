// AI-38 WU10 — CRITICAL-1 remediation (R-ACR-006, S-ACR-017/018,
// verify-report.md Defeat #7): openaicompat's own bridge_test.go is an
// INTERNAL test file (package openaicompat) and cannot import
// conformancetest — conformancetest imports openaicompat, and Go forbids
// an internal test file from importing anything that in turn imports its
// own package (an import cycle openaicompat_test does not share, being a
// distinct package name). This file instead reads bridge_test.go's raw
// source bytes — the same "bytes are bytes" technique
// openrouter/headers_unawareness_test.go already established in this
// module (memory #2571) — extracts the literal its retryOffered
// declaration carries, and asserts it equals conformancetest.RetryOffered,
// the single declared source of truth every conformance factory wrapping
// *openaicompat.Client MUST agree with (bridge_test.go's own
// conformanceBridgeFactory doc comment names this guard).
//
// # Why a raw-bytes scan, not an AST walk
//
// Same rationale as headers_unawareness_test.go: the two declarations
// live in separate compiled test binaries with no type-level door
// between them (the import-cycle constraint above is exactly why), so
// the most direct possible proof is reading the actual bytes a human
// would read and asserting what they say.
package openaicompat_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest"
)

// bridgeTestFilePath is the file this guard scans: openaicompat's own
// conformance bridge factory declaration (bridge_test.go, package
// openaicompat — sibling to this file in the same directory).
const bridgeTestFilePath = "bridge_test.go"

// retryOfferedLiteralPattern matches a `retryOffered := true` (or
// `:= false`) declaration line — the exact shape bridge_test.go's own
// conformanceBridgeFactory doc comment documents as this guard's target.
var retryOfferedLiteralPattern = regexp.MustCompile(`retryOffered\s*:=\s*(true|false)`)

// extractRetryOfferedLiteral reads path's raw bytes and returns the
// boolean literal its retryOffered declaration carries. Fails tb, naming
// the file, when the file cannot be read or the pattern is not found — a
// missing match would otherwise let this guard pass vacuously against a
// renamed or restructured declaration (mirrors
// TestOpenRouterAdapter_HeadersUnawarenessResolvesExpectedPath's
// non-vacuity posture).
func extractRetryOfferedLiteral(tb testing.TB, path string) bool {
	tb.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("os.ReadFile(%q) error = %v, want nil — this guard must read openaicompat's own bridge_test.go to discharge R-ACR-006", path, err)
	}
	match := retryOfferedLiteralPattern.FindSubmatch(raw)
	if match == nil {
		tb.Fatalf("%q: no `retryOffered := true|false` declaration found — this guard cannot prove R-ACR-006 parity against a declaration it cannot locate; update retryOfferedLiteralPattern if the declaration shape changed", path)
		return false
	}
	return string(match[1]) == "true"
}

// TestOpenAICompatBridgeFactory_RetryDeclaration_MatchesSharedSource
// proves R-ACR-006's parity requirement holds for openaicompat's own
// conformance bridge factory: the literal bridge_test.go declares for
// retryOffered equals conformancetest.RetryOffered, the single source
// every conformance factory wrapping *openaicompat.Client must agree
// with. A future edit that flips bridge_test.go's literal out of step
// with the canonical constant — exactly the disagreement
// verify-report.md's Defeat #7 found undetected — fails here, naming
// both the observed and the canonical value.
func TestOpenAICompatBridgeFactory_RetryDeclaration_MatchesSharedSource(t *testing.T) {
	t.Parallel()

	got := extractRetryOfferedLiteral(t, bridgeTestFilePath)
	if got != conformancetest.RetryOffered {
		t.Errorf("bridge_test.go declares retryOffered = %v, want %v (conformancetest.RetryOffered) — every conformance factory wrapping *openaicompat.Client must declare retry identically (R-ACR-006)", got, conformancetest.RetryOffered)
	}
}

// TestOpenAICompatBridgeFactory_RetryDeclaration_FailsOnStagedMutation is
// the falsifiability proof, mirroring
// TestOpenRouterAdapter_HeadersUnawarenessFailsOnStagedMutation's
// bite-proof shape: a scratch file in t.TempDir() carrying a planted
// `retryOffered := false` — the opposite of conformancetest.RetryOffered
// — is extracted as disagreeing, with no edit to the guard itself,
// proving the scan observes what it claims to observe rather than
// passing vacuously.
func TestOpenAICompatBridgeFactory_RetryDeclaration_FailsOnStagedMutation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = "package openaicompat\n\nfunc conformanceBridgeFactoryPlanted() {\n\tretryOffered := false\n\t_ = retryOffered\n}\n"
	path := filepath.Join(dir, "staged_retry_disagreement.go")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	got := extractRetryOfferedLiteral(t, path)
	if got == conformancetest.RetryOffered {
		t.Fatal("planted retryOffered := false was not detected as disagreeing with conformancetest.RetryOffered — the staged mutation proves nothing because the scan does not observe what it claims to observe")
	}
}

// TestOpenAICompatBridgeFactory_RetryDeclarationScan_ResolvesExpectedPath
// is a belt-and-braces check, mirroring
// TestOpenRouterAdapter_HeadersUnawarenessResolvesExpectedPath: the guard
// above would pass vacuously against a missing or empty file (ReadFile
// would surface a fatal error before the deny-list-style comparison ever
// runs), so this test pins bridgeTestFilePath to a real, non-empty file.
func TestOpenAICompatBridgeFactory_RetryDeclarationScan_ResolvesExpectedPath(t *testing.T) {
	t.Parallel()

	info, err := os.Stat(bridgeTestFilePath)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil — this guard must read an existing file", bridgeTestFilePath, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want a regular file", bridgeTestFilePath)
	}
	if info.Size() == 0 {
		t.Errorf("%q is empty — this guard would pass vacuously against an empty file", bridgeTestFilePath)
	}
}

// conformanceRetryTestFilePath is the THIRD retry declaration site this
// module carries (AI-38 WU11 WARN-A remediation, R-ACR-006): the driver
// factory conformance_retry_test.go builds for
// TestRetryCaseBody_RunsDirectly. Unlike bridge_test.go, this file lives in
// package openaicompat_test (external) and CAN — and now does — read
// conformancetest.RetryOffered directly (retryCaseBodyDriverFactory), so
// this guard's job differs from the two above: it does not compare a
// literal against the canonical value (there is no literal left to
// compare); it proves no local literal was silently reintroduced in its
// place, extending the parity-checking apparatus to this third site.
const conformanceRetryTestFilePath = "conformance_retry_test.go"

// TestConformanceRetryTestDriver_DoesNotReintroduceALocalLiteral proves
// conformance_retry_test.go's retryOffered declaration reads
// conformancetest.RetryOffered rather than a bare boolean literal — the
// exact regression verify-report.md's round-2 pass (WARN-A) found: a local
// `retryOffered := true` there silently converted 5 of 7 retry sub-cases
// from PASS to SKIP when mutated to false, with the overall test still
// exiting 0 and nothing named. Reuses retryOfferedLiteralPattern (this
// file's own bridge_test.go guard) as a NEGATIVE assertion here: a match
// means someone reverted to a fragile local literal, regardless of which
// boolean value it happens to carry.
func TestConformanceRetryTestDriver_DoesNotReintroduceALocalLiteral(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(conformanceRetryTestFilePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil — this guard must read conformance_retry_test.go to discharge R-ACR-006 (WARN-A)", conformanceRetryTestFilePath, err)
	}
	if match := retryOfferedLiteralPattern.FindSubmatch(raw); match != nil {
		t.Fatalf("%q declares retryOffered as a bare boolean literal (%q) — want it to read conformancetest.RetryOffered instead (R-ACR-006, AI-38 WU11 WARN-A remediation): a local literal can silently drift from the shared source with nothing failing (verify-report.md round-2 WARN-A)", conformanceRetryTestFilePath, match[1])
	}
}

// TestConformanceRetryTestDriver_LiteralReintroductionScan_FailsOnStagedMutation
// is the falsifiability proof: a scratch file in t.TempDir() carrying a
// planted `retryOffered := true` literal IS detected by
// retryOfferedLiteralPattern, proving the negative-assertion guard above
// would genuinely fire against a real regression rather than passing
// vacuously because the pattern silently stopped matching anything.
func TestConformanceRetryTestDriver_LiteralReintroductionScan_FailsOnStagedMutation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = "package openaicompat_test\n\nfunc plantedLiteralReintroduction() {\n\tretryOffered := true\n\t_ = retryOffered\n}\n"
	path := filepath.Join(dir, "staged_literal_reintroduction.go")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, err)
	}
	if match := retryOfferedLiteralPattern.FindSubmatch(raw); match == nil {
		t.Fatal("planted retryOffered := true literal was not detected — the staged mutation proves nothing because the scan does not observe what it claims to observe")
	}
}

// TestConformanceRetryTestDriverScan_ResolvesExpectedPath mirrors
// TestOpenAICompatBridgeFactory_RetryDeclarationScan_ResolvesExpectedPath's
// non-vacuity pin, for this third file.
func TestConformanceRetryTestDriverScan_ResolvesExpectedPath(t *testing.T) {
	t.Parallel()

	info, err := os.Stat(conformanceRetryTestFilePath)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil — this guard must read an existing file", conformanceRetryTestFilePath, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want a regular file", conformanceRetryTestFilePath)
	}
	if info.Size() == 0 {
		t.Errorf("%q is empty — this guard would pass vacuously against an empty file", conformanceRetryTestFilePath)
	}
}
