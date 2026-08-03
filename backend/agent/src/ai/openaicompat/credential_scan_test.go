// AI-26.1 — the credential-sentinel scan (R-ART-004).
//
// One mechanical scan over this milestone's own expectation-bearing test
// sources, as raw bytes, so a later slice's new expectation file is
// covered automatically with no edit here (S-ART-014).
//
// Scope is this milestone's own external test package
// (`package openaicompat_test`), not every "_test.go" file in the
// directory. AI-25's already-landed, internal (`package openaicompat`)
// test files share this directory and are out of this change's scope:
// proposal.md lists "migrating AI-25's tests to the fixture convention"
// as explicitly out of scope, and credential_test.go's own redaction
// fixture deliberately embeds a key-shaped literal (to prove
// Credential.String never renders it) — a different property this scan
// must not collide with. External vs. internal test package is exactly
// the boundary the harness design already draws elsewhere (expectation
// tests are external; only the inventory/walk tests, slice 8, are
// internal), so restricting the scan to it is a natural extension of an
// existing convention, not an ad hoc exclusion list — and it still
// satisfies S-ART-014, since every later expectation file this milestone
// adds is external too.
package openaicompat_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// credentialSentinel is one recognized credential-shaped class this scan
// rejects (design.md "Credential scan").
type credentialSentinel struct {
	class   string
	pattern *regexp.Regexp
}

// credentialSentinels is the scan's pattern table. Every needle is
// assembled by string concatenation at runtime, never spelled as one
// contiguous literal, so this file's own raw source bytes never contain a
// pattern the scan searches other files' raw bytes for — a scan that
// flagged its own pattern table would be unusable (design.md "Credential
// scan").
var credentialSentinels = []credentialSentinel{
	{
		class:   "OpenAI-style secret key",
		pattern: regexp.MustCompile("s" + "k" + "-" + `[A-Za-z0-9_-]{20,}`),
	},
	{
		class:   "bearer authorization token",
		pattern: regexp.MustCompile("B" + "earer" + ` [A-Za-z0-9._-]{20,}`),
	},
}

// externalTestPackageClause matches only a line that is exactly the
// external test package's own clause, so a mention of the package name
// inside a doc comment or a string literal elsewhere in a file cannot
// match.
var externalTestPackageClause = regexp.MustCompile(`(?m)^package openaicompat_test\s*$`)

// scanCredentialSurface reads every external-package (openaicompat_test)
// "_test.go" file directly inside dir as raw bytes, and reports the first
// file name and sentinel class a match is found in, or two empty strings
// when the surface is clean.
func scanCredentialSurface(dir string) (file, class string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			return "", "", readErr
		}
		if !externalTestPackageClause.Match(raw) {
			continue
		}
		for _, sentinel := range credentialSentinels {
			if sentinel.pattern.Match(raw) {
				return name, sentinel.class, nil
			}
		}
	}
	return "", "", nil
}

// TestCredentialScan_ExpectationSurfaceIsClean is the scan itself
// (S-ART-013): this milestone's own expectation-bearing sources carry no
// credential-shaped value.
func TestCredentialScan_ExpectationSurfaceIsClean(t *testing.T) {
	file, class, err := scanCredentialSurface(".")
	if err != nil {
		t.Fatalf("scanning credential surface: %v", err)
	}
	if file != "" {
		t.Fatalf("credential-shaped value (%s) found in %s", class, file)
	}
}

// writeScannableFile writes content to name inside dir, failing the test
// on error. A test-only helper for the isolated scanCredentialSurface
// exercises below.
func writeScannableFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// TestCredentialScan_FailsOnASentinelInAnExpectationLiteral proves
// S-ART-014 and S-ART-017 against an isolated t.TempDir() surface: a
// credential-shaped value inside a new expectation-shaped literal fails
// the scan, naming the file and the sentinel class, with no edit to the
// scan itself.
func TestCredentialScan_FailsOnASentinelInAnExpectationLiteral(t *testing.T) {
	dir := t.TempDir()
	planted := "s" + "k" + "-" + "STAGEDTESTSENTINELVALUEFORFALSIFIABILITY"
	content := "package openaicompat_test\n\nconst wantBody = \"" + planted + "\"\n"
	writeScannableFile(t, dir, "planted_expectation_test.go", content)

	file, class, err := scanCredentialSurface(dir)
	if err != nil {
		t.Fatalf("scanning credential surface: %v", err)
	}
	if file != "planted_expectation_test.go" {
		t.Fatalf("file = %q, want planted_expectation_test.go", file)
	}
	if class != "OpenAI-style secret key" {
		t.Fatalf("class = %q, want OpenAI-style secret key", class)
	}
}

// TestCredentialScan_FailsOnASentinelInTestSetup proves S-ART-015: the
// scan's subject is the whole file's raw bytes, not only a literal shaped
// like an expectation — a sentinel sitting in ordinary test setup code
// still fails it.
func TestCredentialScan_FailsOnASentinelInTestSetup(t *testing.T) {
	dir := t.TempDir()
	planted := "B" + "earer" + " " + "STAGEDTESTBEARERTOKENFORFALSIFIABILITY"
	content := "package openaicompat_test\n\nfunc plantedSetup() string {\n\treturn \"" + planted + "\"\n}\n"
	writeScannableFile(t, dir, "planted_setup_test.go", content)

	file, class, err := scanCredentialSurface(dir)
	if err != nil {
		t.Fatalf("scanning credential surface: %v", err)
	}
	if file != "planted_setup_test.go" {
		t.Fatalf("file = %q, want planted_setup_test.go", file)
	}
	if class != "bearer authorization token" {
		t.Fatalf("class = %q, want bearer authorization token", class)
	}
}

// TestCredentialScan_HostsAndModelIdentifiersStayGreen proves S-ART-016:
// an endpoint host and a model identifier are legitimate content and must
// never be flagged, so the scan's sentinel classes are deliberately
// narrow rather than broad heuristics.
func TestCredentialScan_HostsAndModelIdentifiersStayGreen(t *testing.T) {
	dir := t.TempDir()
	content := "package openaicompat_test\n\n" +
		"const wantHost = \"https://api.example-provider.test/v1\"\n" +
		"const wantModel = \"gpt-4o\"\n"
	writeScannableFile(t, dir, "legitimate_content_test.go", content)

	file, class, err := scanCredentialSurface(dir)
	if err != nil {
		t.Fatalf("scanning credential surface: %v", err)
	}
	if file != "" {
		t.Fatalf("legitimate content incorrectly flagged: %s in %s", class, file)
	}
}

// TestCredentialScan_IgnoresInternalTestPackageFiles confirms the scope
// note in the file comment: an internal-package (`package
// openaicompat`) test file — AI-25's own convention, sharing this
// directory — is not swept by this milestone's scan even when it carries
// a key-shaped literal, because migrating AI-25's tests to this
// convention is out of scope (proposal.md).
func TestCredentialScan_IgnoresInternalTestPackageFiles(t *testing.T) {
	dir := t.TempDir()
	planted := "s" + "k" + "-" + "ANINTERNALPACKAGEFIXTURENOTINSCOPE"
	content := "package openaicompat\n\nconst legacyToken = \"" + planted + "\"\n"
	writeScannableFile(t, dir, "internal_fixture_test.go", content)

	file, class, err := scanCredentialSurface(dir)
	if err != nil {
		t.Fatalf("scanning credential surface: %v", err)
	}
	if file != "" {
		t.Fatalf("internal-package file incorrectly scanned: %s in %s", class, file)
	}
}
