// AI-38 WU11 — WARN-E / S-CNF-086 remediation: a raw-bytes guard proving
// run_for_test.go — the acceptance artifact R-ACR-001/R-CNF-028 govern —
// still carries the citation stating its scoped debugging-affordance
// drivers are never conformance evidence. Mirrors retry_parity_test.go's
// own "bytes are bytes" idiom (itself mirroring
// openrouter/headers_unawareness_test.go): the citation is prose, not a
// typed symbol, so the most direct possible proof reads the file's actual
// bytes and asserts what they say, the same way a human reviewer would.
//
// # Why this closes S-CNF-086, not just R-CNF-028's structural half
//
// WU10's TestRunConformanceFor_ReturnsNoVerdictOrRecord_StructuralGuard
// (agenttest/conformance_scoped_test.go) proves a scoped run structurally
// CANNOT produce a verdict or record a caller could mistake for total
// evidence — S-CNF-085's compile-time half. S-CNF-086 is the companion
// obligation: the citation stating this MUST remain present at the exact
// artifact site the spec names, so a future edit that quietly drops the
// qualifying language while keeping the scoped drivers is caught by name,
// not merely by review.

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// scopedRunCitationFilePath is the acceptance artifact R-ACR-001/R-CNF-028
// govern: the file carrying both the unscoped acceptance gate and the two
// scoped debugging-affordance drivers.
const scopedRunCitationFilePath = "run_for_test.go"

// scopedRunCitationPhrase is the exact non-evidential citation
// run_for_test.go's header comment carries for its scoped drivers
// (R-CNF-028, S-CNF-086).
const scopedRunCitationPhrase = "never cited as conformance evidence"

// scopedRunCitationRequiredSubstrings names every byte sequence
// scopedRunCitationFilePath MUST carry for S-CNF-086 to hold: the
// requirement ID under obligation, the exact non-evidential phrase, and
// the three test function signatures the citation is about (the unscoped
// gate plus both scoped drivers — belt-and-braces against a wholesale
// removal, not only a citation removal).
var scopedRunCitationRequiredSubstrings = []string{
	"R-CNF-028",
	scopedRunCitationPhrase,
	"func TestOpenRouterAdapter_FullConformance",
	"func TestOpenRouterAdapter_StreamingText",
	"func TestOpenRouterAdapter_ToolCalls",
}

// scopedRunDriverFuncPattern locates both scoped debugging-affordance
// driver function definitions the citation must precede (in byte order)
// — proving co-location, not merely presence somewhere in the file.
var scopedRunDriverFuncPattern = regexp.MustCompile(`func Test(OpenRouterAdapter_StreamingText|OpenRouterAdapter_ToolCalls)\(`)

// checkScopedRunCitation reads path's raw bytes and returns every
// S-CNF-086 problem found: a missing required substring, or the
// non-evidential citation phrase failing to precede either scoped driver
// function it must govern. A read error is reported as its own single
// problem rather than panicking, so this stays a pure, testing.TB-free
// function the bite-proofs below can drive against an arbitrary scratch
// path exactly the way the real guard drives it against the real file.
func checkScopedRunCitation(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("os.ReadFile(%q) error = %v", path, err)}
	}
	content := string(raw)

	var problems []string
	for _, want := range scopedRunCitationRequiredSubstrings {
		if !strings.Contains(content, want) {
			problems = append(problems, fmt.Sprintf("%q does not contain %q — the R-CNF-028/S-CNF-086 citation (or the driver it governs) is missing", path, want))
		}
	}

	citationIdx := strings.Index(content, scopedRunCitationPhrase)
	if citationIdx < 0 {
		// Already reported above (the phrase is itself one of the
		// required substrings) — nothing further to co-locate.
		return problems
	}

	for _, m := range scopedRunDriverFuncPattern.FindAllStringIndex(content, -1) {
		if citationIdx >= m[0] {
			problems = append(problems, fmt.Sprintf("%q: the %q citation (byte offset %d) does not precede the scoped driver at byte offset %d — S-CNF-086 requires the non-evidential citation to govern the driver it sits above, not follow it", path, scopedRunCitationPhrase, citationIdx, m[0]))
		}
	}
	return problems
}

// TestScopedRunCitation_RemainsPresentAndPrecedesTheDrivers proves
// S-CNF-086: run_for_test.go still carries every required citation
// substring, and the non-evidential phrase's byte offset is strictly
// before both scoped driver function definitions it qualifies.
func TestScopedRunCitation_RemainsPresentAndPrecedesTheDrivers(t *testing.T) {
	t.Parallel()

	for _, problem := range checkScopedRunCitation(scopedRunCitationFilePath) {
		t.Error(problem)
	}
}

// TestScopedRunCitation_FailsOnStagedRemoval is the falsifiability proof
// for the required-substring half: a scratch file in t.TempDir() carrying
// both scoped driver function signatures but dropping the non-evidential
// citation phrase entirely is detected as missing it, with no edit to
// checkScopedRunCitation itself.
func TestScopedRunCitation_FailsOnStagedRemoval(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = `package openrouter_conformance

func TestOpenRouterAdapter_FullConformance(t *testing.T) {}

func TestOpenRouterAdapter_StreamingText(t *testing.T) {}

func TestOpenRouterAdapter_ToolCalls(t *testing.T) {}
`
	path := filepath.Join(dir, "staged_citation_removed.go")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	if problems := checkScopedRunCitation(path); len(problems) == 0 {
		t.Fatal("planted fixture carrying the scoped drivers but no citation was not detected as missing anything — the staged mutation proves nothing because the check does not observe what it claims to observe")
	}
}

// TestScopedRunCitation_FailsOnStagedMisplacement is the falsifiability
// proof for the co-location half: a scratch file carrying the citation
// phrase AFTER both scoped drivers (instead of governing them from
// above, as run_for_test.go's real header comment does) is detected as
// misplaced.
func TestScopedRunCitation_FailsOnStagedMisplacement(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = `package openrouter_conformance

func TestOpenRouterAdapter_FullConformance(t *testing.T) {}

func TestOpenRouterAdapter_StreamingText(t *testing.T) {}

func TestOpenRouterAdapter_ToolCalls(t *testing.T) {}

// R-CNF-028: their verdict is never cited as conformance evidence.
`
	path := filepath.Join(dir, "staged_citation_misplaced.go")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	if problems := checkScopedRunCitation(path); len(problems) == 0 {
		t.Fatal("planted fixture with the citation AFTER both scoped drivers was not detected as misplaced — the staged mutation proves nothing because the co-location check does not observe what it claims to observe")
	}
}

// TestScopedRunCitationScan_ResolvesExpectedPath is a belt-and-braces
// check, mirroring TestOpenAICompatBridgeFactory_RetryDeclarationScan_ResolvesExpectedPath:
// the guard above would pass vacuously against a missing or empty file,
// so this test pins scopedRunCitationFilePath to a real, non-empty file.
func TestScopedRunCitationScan_ResolvesExpectedPath(t *testing.T) {
	t.Parallel()

	info, err := os.Stat(scopedRunCitationFilePath)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil — this guard must read an existing file", scopedRunCitationFilePath, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want a regular file", scopedRunCitationFilePath)
	}
	if info.Size() == 0 {
		t.Errorf("%q is empty — this guard would pass vacuously against an empty file", scopedRunCitationFilePath)
	}
}
