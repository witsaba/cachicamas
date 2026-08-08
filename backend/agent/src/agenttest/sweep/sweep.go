// Package sweep is the one credential/content-sentinel sweep every
// redaction proof in this module reaches (AI-36.1, WU-1, S-CNF-080,
// design.md AD-1).
//
// # Why this is its own subpackage, not part of agenttest itself
//
// agenttest imports "testing" package-wide (conformance_redaction.go), and
// openrouter/smoke/sentinel_sweep.go is a non-test file: importing
// agenttest directly from it would pull testing's flag registration into a
// non-test build. This package imports only "bytes" and "fmt", so it is
// safe to import from a non-test file, a _test.go file, or any package
// under the module — production and test code alike (design.md AD-1).
//
// # Detection facts, not formatted errors (AD-1)
//
// Scan reports WHAT matched, never a formatted message: each existing call
// site (openrouter/smoke's Scan, agenttest's scanForSentinel/
// scanTextForSentinel) keeps its own message text verbatim rather than
// adopting a new one, so migrating to this shared core is a behavior-
// preserving refactor, not a message-format change.
//
// # The positive control is part of the package (AD-2)
//
// SelfTest is this milestone's answer to its own stated characteristic
// failure mode: an absence assertion that cannot fail is worse than no
// assertion at all. Every sweep call site runs SelfTest(deny) before
// trusting a clean Scan result over the real corpus.
package sweep

import (
	"bytes"
	"fmt"
)

// Entry is one deny-list row: Vector names the leak class a caller's own
// report may name, and Needle is the byte sequence Scan searches for.
// Needle is never reproduced by Scan's return value or by SelfTest's
// error — only Vector is (R-CNF-027 item 6, no sweep reprints a match).
type Entry struct {
	Vector string
	Needle []byte
}

// Scan reports the first Entry in deny whose Needle occurs in corpus, and
// true — or ("", false) when none does. Scan returns detection facts
// only; it builds no formatted message, so each call site's own message
// text (and its own no-reprint discipline) stays exactly as written.
//
// Entries are consulted in declaration order, so the vector a caller
// reports is deterministic when more than one needle matches. An entry
// with an empty Needle is skipped — bytes.Contains would otherwise treat
// an empty subslice as present in every corpus, turning a degenerate
// entry into a false positive on every scan. SelfTest is the
// complementary check that a skipped, empty-needle entry is never
// silently trusted as a proof (see SelfTest below).
func Scan(corpus []byte, deny []Entry) (vector string, found bool) {
	if len(corpus) == 0 {
		return "", false
	}
	for _, entry := range deny {
		if len(entry.Needle) == 0 {
			continue
		}
		if bytes.Contains(corpus, entry.Needle) {
			return entry.Vector, true
		}
	}
	return "", false
}

// SelfTest is the positive control every Scan call site runs before
// trusting a clean result (AD-2, R-CNF-027 item 5): for each entry, it
// builds an in-memory corpus of benign padding plus that entry's own
// Needle, scans for it in isolation, and reports an error naming the
// first vector that failed to bite. Nothing is written to a fixture — the
// corpus lives only on this call's stack, and teardown is ordinary
// garbage collection.
//
// An entry carrying an empty Needle can never bite (Scan skips it by
// construction, above) — SelfTest deliberately does NOT special-case that
// away: a deny-list entry that can never be detected is exactly the
// silently-vacuous shape this milestone exists to catch, so SelfTest
// reports it as broken, naming the vector, rather than passing it through
// unproven.
func SelfTest(deny []Entry) error {
	for _, entry := range deny {
		corpus := append([]byte("sweep self-test padding — "), entry.Needle...)
		if _, found := Scan(corpus, []Entry{entry}); !found {
			return fmt.Errorf("agenttest/sweep: SelfTest: vector %q did not bite its own needle in a synthetic corpus — the deny-list entry (or the sweep mechanism) is broken", entry.Vector)
		}
	}
	return nil
}
