// AI-39.1 — the sentinel-sweep helper's test surface (R-OR-08, design
// D8, tasks.md § PR #3 work unit 3.2).
//
// # What this file covers
//
// The sentinel sweep lives at the boundary between the live smoke
// test (smoke_test.go) and the way the OpenRouter credential flows
// through log buffers, error messages, captured test arguments, and
// any other string field reachable from the test process. The
// helper's surface is small:
//
//   - DenyEntry{Vector, Needle} — one deny-list row.
//   - BuildDenyList(envVarName, secretPrefix, plantedPrompt) — assembles
//     the three deny-list entries R-OR-08 names, with needles built
//     at runtime so the source file never contains its own
//     contiguous literals.
//   - Scan(captured, denyList) — returns nil when captured is clean,
//     or a non-nil error naming the first matching leak vector (but
//     never the credential itself).
//
// The scan is a pure function over bytes; it has no observable
// side effects, no shared state, and no goroutine creation. The
// test package is the external smoke_test package so the helper is
// exercised through the same import path a future caller would use.
//
// # The deferred R-OR-08 staged-mutation bite-proof
//
// PR #1's verify report (WARNING #3) and PR #2's verify report
// (WARNING #2) both flag the redaction surface as lacking a
// separate staged-mutation bite-proof — the only mechanical guard
// in this PR without one. This file closes that loop with
// TestSentinelSweep_CatchesDeliberateLogfKeyMutation, which:
//
//   1. Builds a known-leaky log buffer (the literal substring
//      "Bearer " + the secret prefix).
//   2. Runs the scan over the buffer.
//   3. Asserts the scan returns a non-nil error naming the leak
//      vector.
//
// The bite-proof is structural: a regression that broke the scan
// (returning nil for any input, or only returning the vector name
// when Needle is empty) would turn the test's assertion into a
// false negative — the test would FAIL, the regression would be
// surfaced, and the guard's load-bearing nature would be observable
// in CI. The positive control TestSentinelSweep_RedactedPlaceholder_DoesNotTrigger
// is the foil: the scan must not flag the synthetic <redacted>
// placeholder, so a scan that flags everything would also fail —
// both directions are bite-proofs.
//
// # TDD posture (work unit 3.2)
//
// RED  : the assertions below reference Scan, BuildDenyList, and
//        DenyEntry, none of which exist yet — the file does not
//        compile. The seven tests are the coverable surface; the
//        bite-proof is the eighth.
// GREEN: sentinel_sweep.go lands with the three exports plus the
//        runtime needle construction. All eight tests pass. The
//        scan's error message never reproduces the credential once
//        the helper is in place.
package smoke_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/smoke"
)

// commitRedactedPlaceholder is the synthetic credential-shaped
// literal the credential redaction uses (S-APC-053 →
// openaicompat.Credential.String/GoString returns "<redacted>"). The
// positive control below asserts the scan does NOT flag this
// placeholder — a scan that flags every credential-shaped literal
// would be unusable because the credential's own redaction output
// would false-positive.
const commitRedactedPlaceholder = "<redacted>"

// plantedSecretPrefix is the test-only secret prefix the scan is
// asked to find. The value is shaped like a real OpenRouter key
// prefix (lowercase, dash, four-character run) so the deny-list
// needle has the same shape the live smoke would feed in.
const plantedSecretPrefix = "sk-aa"

// plantedEnvVarName is the env-var-name entry the deny-list carries
// on the build path. The bytes are the literal env-var name; the
// scan must find this string in captured bytes.
const plantedEnvVarName = "OPENROUTER_API_KEY"

// plantedPromptMarker is the prompt marker the smoke test plants
// in its request (smoke_test.go's liveSmokePromptMarker). The scan
// treats this as one deny-list entry because the prompt is the
// other non-secret string the live test naturally captures into
// log buffers.
const plantedPromptMarker = "live-smoke-prompt-marker-9b3a8f2c"

// TestSentinelSweep_NoMatch_ReportsNoLeak covers the clean-input
// path: captured bytes that contain none of the deny-list needles
// produce a nil error. This is the foil for every leak-detection
// test below — a regression that flags every input would surface
// here.
func TestSentinelSweep_NoMatch_ReportsNoLeak(t *testing.T) {
	t.Parallel()

	captured := []byte("hello world, no keys here — just a friendly log line")
	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	if err := smoke.Scan(captured, deny); err != nil {
		t.Errorf("Scan() = %v, want nil (R-OR-08, design D8 — clean input must report no leak)", err)
	}
}

// TestSentinelSweep_DetectsEnvVarName covers R-OR-08 deny-list
// entry 1: the literal env-var name. The scan must find the
// literal "OPENROUTER_API_KEY" in captured bytes and report the
// leak by naming the vector, not the credential.
func TestSentinelSweep_DetectsEnvVarName(t *testing.T) {
	t.Parallel()

	captured := []byte("configuration: OPENROUTER_API_KEY=REDACTED")
	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	err := smoke.Scan(captured, deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the env-var name was in the captured bytes (R-OR-08 deny-list entry 1)")
	}
	if !strings.Contains(err.Error(), "env-var") {
		t.Errorf("error = %q, want a substring naming the leak vector 'env-var'", err.Error())
	}
}

// TestSentinelSweep_DetectsSecretPrefix covers R-OR-08 deny-list
// entry 2: the secret's prefix (4 chars). The scan must find the
// planted "sk-aa" prefix in captured bytes and report the leak
// naming the vector, never the prefix itself.
func TestSentinelSweep_DetectsSecretPrefix(t *testing.T) {
	t.Parallel()

	captured := []byte("authorization: Bearer sk-aa-bb-cc-dd-ee-ff-gg")
	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	err := smoke.Scan(captured, deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the secret prefix was in the captured bytes (R-OR-08 deny-list entry 2)")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Errorf("error = %q, want a substring naming the leak vector 'secret' or 'prefix'", err.Error())
	}
}

// TestSentinelSweep_DetectsPlantedPrompt covers R-OR-08 deny-list
// entry 3: the planted prompt bytes the smoke test embeds in its
// request. The scan must find the planted prompt marker in
// captured bytes and report the leak.
func TestSentinelSweep_DetectsPlantedPrompt(t *testing.T) {
	t.Parallel()

	captured := []byte("request body: {\"messages\":[{\"role\":\"user\",\"content\":\"live-smoke-prompt-marker-9b3a8f2c — please reply OK\"}]}")
	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	err := smoke.Scan(captured, deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the planted prompt marker was in the captured bytes (R-OR-08 deny-list entry 3)")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("error = %q, want a substring naming the leak vector 'prompt'", err.Error())
	}
}

// TestSentinelSweep_RedactedPlaceholder_DoesNotTrigger is the
// positive control (R-OR-08 sub-scenario 2). The credential's own
// redaction output is "<redacted>" (S-APC-053); the scan must
// tolerate this literal in captured bytes so a test that asserts
// the credential is redacted doesn't false-positive on the
// redaction marker itself. A scan that flagged every
// credential-shaped literal would be unusable because the
// redaction output is itself a credential-shaped literal.
func TestSentinelSweep_RedactedPlaceholder_DoesNotTrigger(t *testing.T) {
	t.Parallel()

	// The redaction posture is "the credential never appears in
	// its raw form; what appears is the placeholder". Captured
	// bytes containing the placeholder are the success state.
	captured := []byte(fmt.Sprintf("credential rendered: %s", commitRedactedPlaceholder))
	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	if err := smoke.Scan(captured, deny); err != nil {
		t.Errorf("Scan() = %v, want nil (S-APC-053 — the synthetic <redacted> placeholder is the success state, not a leak)", err)
	}

	// Also assert the scan tolerates the placeholder inside the
	// "Bearer " prefix-collision case. The literal "Bearer " in
	// the credential scan's source is part of the planted secret
	// pattern; the placeholder is what the redaction actually
	// emits, and the scan must not flag it.
	capturedBearer := []byte(fmt.Sprintf("authorization: Bearer %s", commitRedactedPlaceholder))
	if err := smoke.Scan(capturedBearer, deny); err != nil {
		t.Errorf("Scan() on Bearer <redacted> = %v, want nil (positive control — the synthetic placeholder must not trigger the secret-prefix scan)", err)
	}
}

// TestSentinelSweep_ErrorDoesNotReprintCredential covers R-OR-08
// sub-scenario 1: the error message names the leak vector but
// NEVER reproduces the credential. The scan must find the planted
// secret and return an error that does not contain the secret
// bytes. A regression that included the secret in the error
// message would fail this test.
func TestSentinelSweep_ErrorDoesNotReprintCredential(t *testing.T) {
	t.Parallel()

	const fullSecret = "sk-aa-supersecret-leaked-shape-zz-9999"
	captured := []byte(fmt.Sprintf("oops: bearer %s", fullSecret))
	deny := smoke.BuildDenyList(plantedEnvVarName, fullSecret[:4], plantedPromptMarker)

	err := smoke.Scan(captured, deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the secret prefix was in the captured bytes")
	}
	if strings.Contains(err.Error(), fullSecret) {
		t.Errorf("error must not reprint the credential: %q", err.Error())
	}
	if strings.Contains(err.Error(), fullSecret[:4]) {
		t.Errorf("error must not reprint even the secret prefix: %q", err.Error())
	}
}

// TestSentinelSweep_DenyListEntriesAreDistinct covers the build
// helper's contract: when the three args are identical (a
// degenerate case), the deny-list carries the three entries as
// distinct rows. The scan would report the first match; the
// entries are kept distinct so the build is observable.
func TestSentinelSweep_DenyListEntriesAreDistinct(t *testing.T) {
	t.Parallel()

	deny := smoke.BuildDenyList("DUPLICATE", "DUPLICATE", "DUPLICATE")
	if len(deny) != 3 {
		t.Errorf("len(deny) = %d, want 3 (R-OR-08 deny-list has three entries even when the inputs collide)", len(deny))
	}
}

// TestSentinelSweep_EmptyCaptured_ReportsNoLeak covers the
// zero-input case: an empty captured buffer produces no leak. The
// scan must not panic on empty input.
func TestSentinelSweep_EmptyCaptured_ReportsNoLeak(t *testing.T) {
	t.Parallel()

	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)
	if err := smoke.Scan([]byte{}, deny); err != nil {
		t.Errorf("Scan([]byte{}) = %v, want nil (R-OR-08 — empty input is the clean input, not a leak)", err)
	}
}

// TestSentinelSweep_CatchesDeliberateLogfKeyMutation is the
// deferred R-OR-08 staged-mutation bite-proof (verify-report-pr1
// WARNING #3, verify-report-pr2 WARNING #2, engram #2583's
// RED-found-defect trail). It simulates the regression that
// PR #1's redaction surface guards against: a programmer
// accidentally calls t.Logf with the credential embedded in the
// format string. The sentinel sweep, applied to the captured
// log, catches the leak and fails the test.
//
// The bite-proof is structural:
//
//   1. Stage a known-leaky log buffer with the literal
//      "Bearer " + the credential prefix — the regression.
//   2. Run the scan over the buffer.
//   3. Assert the scan returns a non-nil error naming the
//      leak vector.
//
// A regression that disabled the scan (returned nil for any
// input) would turn the assertion into a false negative — the
// test would FAIL, the regression would be surfaced. The
// positive control (TestSentinelSweep_RedactedPlaceholder_DoesNotTrigger)
// is the foil: a scan that flagged everything would also fail
// here. Both directions are bite-proofs because the scan's
// load-bearing role is observable in both the leak-case and the
// clean-case.
func TestSentinelSweep_CatchesDeliberateLogfKeyMutation(t *testing.T) {
	t.Parallel()

	// The planted secret. The credential prefix is what the scan
	// keys on; the full secret shape mirrors what a real
	// accidental-logf regression might emit.
	const plantedSecret = "sk-deliberate-leaked-shape-bb-cc-dd-ee"

	// Build a known-leaky log buffer. This is the regression
	// surface: a caller writes the credential into a log message.
	// The byte buffer is captured before the scan runs, so the
	// test exercises the same shape the live smoke would.
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "Bearer %s", plantedSecret)

	// Stage the deny-list with the planted secret's prefix.
	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecret[:4], plantedPromptMarker)

	// Run the scan. The planted secret's prefix is in buf, so the
	// scan must catch it.
	err := smoke.Scan(buf.Bytes(), deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the planted secret prefix was in the captured log buffer (R-OR-08 — the scan must catch a deliberate leak, the regression this bite-proof guards against)")
	}

	// The error must name the vector; it must not reprint the
	// credential or even the prefix.
	if strings.Contains(err.Error(), plantedSecret) {
		t.Errorf("error must not reprint the credential: %q", err.Error())
	}
	// The vector name is "secret" or "prefix" (any sensible
	// variant); the live smoke's log buffer that captured this
	// leak would surface the vector name only.
	if !strings.Contains(err.Error(), "secret") && !strings.Contains(err.Error(), "prefix") {
		t.Errorf("error = %q, want a substring naming the leak vector (one of: secret, prefix)", err.Error())
	}
}

// TestSentinelSweep_CatchesDeliberateLogfEnvVarNameMutation is
// the second dimension of the deferred bite-proof: a regression
// that logged the env-var name by mistake. The scan catches the
// env-var-name entry, not the secret-prefix entry.
func TestSentinelSweep_CatchesDeliberateLogfEnvVarNameMutation(t *testing.T) {
	t.Parallel()

	// Planted: the env-var name in a log buffer. No secret bytes
	// present.
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "checking env: %s", plantedEnvVarName)

	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	err := smoke.Scan(buf.Bytes(), deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the env-var name was in the captured log buffer (R-OR-08 deny-list entry 1)")
	}
	if !strings.Contains(err.Error(), "env-var") {
		t.Errorf("error = %q, want a substring naming the leak vector 'env-var'", err.Error())
	}
}

// TestSentinelSweep_CatchesDeliberateLogfPromptMutation is the
// third dimension of the deferred bite-proof: a regression that
// logged the planted prompt bytes. The scan catches the
// planted-prompt entry, not the secret-prefix entry.
func TestSentinelSweep_CatchesDeliberateLogfPromptMutation(t *testing.T) {
	t.Parallel()

	// Planted: the prompt marker in a log buffer. No secret bytes
	// present.
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "request: %s", plantedPromptMarker)

	deny := smoke.BuildDenyList(plantedEnvVarName, plantedSecretPrefix, plantedPromptMarker)

	err := smoke.Scan(buf.Bytes(), deny)
	if err == nil {
		t.Fatal("Scan() = nil, want non-nil: the planted prompt marker was in the captured log buffer (R-OR-08 deny-list entry 3)")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("error = %q, want a substring naming the leak vector 'prompt'", err.Error())
	}
}
