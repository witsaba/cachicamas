// AI-38 WU10 — CRITICAL-1 remediation (R-ACR-006, S-ACR-017/018,
// verify-report.md Defeat #7): pins the single declared source of truth
// every conformance factory wrapping *openaicompat.Client must read for
// its Factory.Retry field. See RetryOffered's own doc comment in
// retry.go for the full rationale.
package conformancetest

import "testing"

// TestRetryOffered_DeclaresTrue proves the shared declaration itself
// reads true (design D2 locked decision 2: openaicompat's client
// auto-retries per AI-35). A future accidental flip of the ONE place
// capable of declaring this value is caught here by name, before it can
// propagate — silently or otherwise — to any consuming factory.
func TestRetryOffered_DeclaresTrue(t *testing.T) {
	if !RetryOffered {
		t.Fatalf("conformancetest.RetryOffered = false, want true (R-ACR-006: openaicompat's client auto-retries per AI-35, design D2 locked decision 2)")
	}
}
