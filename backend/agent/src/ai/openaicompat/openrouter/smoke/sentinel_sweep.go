// Package smoke implements the credential-leak sentinel sweep that
// the live OpenRouter smoke test (smoke_test.go) uses to gate its
// log buffers, error messages, and captured test arguments against
// accidental credential exposure.
//
// # What this package is
//
// A pure-function credential-leak detector for the live OpenRouter
// smoke (smoke_test.go). The scan walks a captured byte buffer
// (test log, captured stderr, environment slices captured by
// tests, or any other string field reachable from the test process
// at teardown) against a deny-list of three entries the live smoke
// configures:
//
//   1. The literal env-var name "OPENROUTER_API_KEY" — knowing
//      the env-var name is logged is the first hint that the
//      secret is reachable; flag it.
//   2. The secret's prefix (4 chars) — the live smoke feeds the
//      prefix; finding the prefix in the buffer is the strongest
//      signal that the secret was emitted into a log path.
//   3. The planted prompt bytes — the live smoke's request
//      embeds a non-secret marker; finding the marker in the
//      buffer is the third deny-list entry the spec names.
//
// A scan that finds a match returns a non-nil error naming the
// offending vector ("env-var", "secret-prefix", "planted-prompt")
// but NEVER reproducing the credential itself. The credential's
// raw form is identified by vector name alone; the scan never
// re-reads the captured bytes into the error message.
//
// # Why the needles are built at runtime
//
// The deny-list needles are constructed from byte slices assembled
// at runtime, never spelled as contiguous string literals in this
// source file. A scanner that flagged its own pattern table would
// be unusable: every test process would FAIL on the helper's own
// source. The shipped pattern in openaicompat/credential_scan_test.go
// uses the same trick for the "sk-" + 20+ chars and "Bearer " +
// 20+ chars classes; this helper follows the same convention
// (S-ART-014's design).
//
// # Why this is a pure function
//
// Scan is pure: it takes a byte buffer and a deny-list, returns a
// nil-or-error, and touches no package-level state. The deny-list
// is built per-call via BuildDenyList, so two tests cannot race on
// the same deny-list, and a regression that mutates the
// deny-list between calls is impossible by construction. The
// helper serves no I/O, no network, no filesystem — the package
// keeps the AI-00.3 forward guard green (zero requires, no
// non-stdlib imports beyond what the smoke sub-package already
// needs).
//
// # TDD posture (work unit 3.2)
//
// RED  : the assertions in sentinel_sweep_test.go reference
//        smoke.Scan, smoke.BuildDenyList, and smoke.DenyEntry —
//        none of which exist before this file lands. The
//        _test.go file does not compile and the package
//        reports "no non-test Go files", which is the observable
//        RED state.
// GREEN: this file lands with the three exports and the
//        runtime needle construction. The eight tests
//        (including the deferred R-OR-08 staged-mutation
//        bite-proof) pass; the scan's error message never
//        reproduces the credential.
package smoke

import (
	"bytes"
	"fmt"
)

// DenyEntry is one deny-list row the scan consults. Vector is the
// human-readable name the scan's error message renders when a
// Needle match is found; Needle is the byte sequence the scan
// searches for. The deny-list is built per-call via BuildDenyList
// so two tests cannot share mutations and the scan's source
// never contains its own assembled patterns as contiguous
// literals.
type DenyEntry struct {
	Vector string
	Needle []byte
}

// BuildDenyList assembles the three deny-list entries the live
// smoke's spec requires (R-OR-08): the env-var name, the secret's
// prefix (4 chars), and the planted prompt bytes. The needles
// are runtime-built strings so the source file's own raw bytes
// never include the assembled pattern — a scanner that flagged
// its own pattern table would be unusable (S-ART-014-style
// defence).
//
// The function is a constructor: it returns a fresh slice every
// call and does not retain any references to its inputs. The
// scan is concurrency-safe by construction (no shared state).
func BuildDenyList(envVarName, secretPrefix, plantedPrompt string) []DenyEntry {
	// The vector names are spelled as literals because they are
	// documentation, not needles — the scan prints them in error
	// messages, never searches for them. The needles are
	// byte-rebuilt at runtime.
	return []DenyEntry{
		{Vector: "env-var name", Needle: []byte(envVarName)},
		{Vector: "secret prefix", Needle: []byte(secretPrefix)},
		{Vector: "planted prompt", Needle: []byte(plantedPrompt)},
	}
}

// Scan walks captured for any deny-list needle and returns a
// non-nil error naming the first matching vector (but NEVER the
// credential itself). The credential is identified by vector
// name alone; the scan never re-reads the captured bytes into
// the error message.
//
// The scan iterates deny-list entries in declaration order so the
// vector a caller sees in the error message is deterministic —
// the env-var-name entry is found first if both the env-var name
// and the secret prefix appear in captured bytes, which is the
// correct ordering for a CI operator reading the skip/fail
// message.
//
// An empty captured buffer produces no leak. An empty deny-list
// produces no leak. These are the clean-input cases the test
// surface asserts above.
func Scan(captured []byte, denyList []DenyEntry) error {
	if len(captured) == 0 {
		return nil
	}
	for _, entry := range denyList {
		if len(entry.Needle) == 0 {
			continue
		}
		if bytes.Contains(captured, entry.Needle) {
			// The error names the vector, never the credential.
			// The captured bytes are not interpolated into the
			// error message; the scan does not read the needle
			// back to the caller. This is the load-bearing
			// property the deferred R-OR-08 staged-mutation
			// bite-proof enforces — a regression that
			// fmt.Errorf("... %s ...", entry.Needle) would
			// collapse the test's positive control.
			return fmt.Errorf("sentinel sweep detected credential leak: vector=%q", entry.Vector)
		}
	}
	return nil
}
