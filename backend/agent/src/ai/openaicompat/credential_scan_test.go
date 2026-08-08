// AI-26.1 (R-ART-004), widened by AI-36 (WU-5, D-3, R-ART-022) — the
// credential-sentinel scan.
//
// # What changed at AI-36
//
// The scan used to read only files directly inside one directory
// (os.ReadDir) belonging to the external test package (`package
// openaicompat_test`). That was a disclosed gap (stream.go:309-318): an
// *accidental* real credential in an internal-package test file, or in
// any nested directory (openrouter/, openrouter/smoke/,
// openrouter/conformance/, testdata/), was invisible.
//
// The scan now walks the whole adapter tree recursively
// (filepath.WalkDir) and sweeps every test file and every fixture file,
// regardless of package. A file MAY be exempt only by carrying an
// explicit in-file declaration — the deliberate-plant marker below — AND
// appearing on a committed, enumerated allowlist; adding a declaration
// without updating the allowlist (or vice versa) still fails the scan,
// which is what makes a new plant a reviewable diff (R-ART-022 item 3).
//
// # Why the marker and the allowlist are assembled at runtime
//
// If this file's own source contained the marker text or an allowlist
// path as one contiguous literal that just happened to also look like
// something the scan searches other files for, the scan could become
// self-referential in confusing ways. More concretely: the marker text
// itself MUST be built by concatenation so this file's own source bytes
// never contain it as a contiguous run — otherwise this file would look
// like it "carries the marker" to its own detection logic, and would
// need to explain why it is not on the allowlist (R-ART-022 item 4,
// S-ART-093).
package openaicompat_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// credentialSentinel is one recognized credential-shaped class this scan
// rejects (unchanged from AI-26.1).
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

// deliberatePlantMarker is the exact declaration text a file must carry,
// verbatim, to be considered for exemption (D-3, R-ART-022 item 2).
// Assembled at runtime — see the package doc comment for why.
var deliberatePlantMarker = "credential-scan:" + "deliberate-plant"

// deliberatePlantAllowlist is the committed, enumerated set of files
// permitted to carry the marker (R-ART-022 item 3), relative to this
// package's own directory (the scan's root). Multi-segment paths are
// built with filepath.Join rather than a hand-joined literal.
// TestCredentialScan_AllowlistFalsifiability asserts every entry here
// still corresponds to a real plant, not just a name on a list.
var deliberatePlantAllowlist = []string{
	"credential_test.go",
	"capture_proof_test.go",
	"viability_test.go",
	filepath.Join("openrouter", "credential_redaction_test.go"),
	filepath.Join("openrouter", "smoke", "smoke_test.go"),
	filepath.Join("openrouter", "smoke", "sentinel_sweep_test.go"),
}

// scanResult is one file the recursive walk found carrying a
// not-exempted credential-shaped pattern. It never carries the matched
// bytes — only the file's path (relative to the scan root) and the
// sentinel class name (R-ART-022 item 5).
type scanResult struct {
	relPath string
	class   string
}

// scannableFile reports whether d (found at path during a walk) is
// within this scan's surface: every file whose name ends in "_test.go",
// and every file inside a directory literally named "testdata" or
// "fixtures" (Go's own convention for non-buildable, fixture-only
// content) — "every test file and every fixture file under the adapter
// tree" (R-ART-022 item 1). No exemption by package kind, directory
// naming beyond this, or extension is consulted here — that is a
// property of the declaration+allowlist check, not of what gets read.
func scannableFile(path string, d fs.DirEntry) bool {
	if d.IsDir() {
		return false
	}
	if strings.HasSuffix(d.Name(), "_test.go") {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "testdata" || part == "fixtures" {
			return true
		}
	}
	return false
}

// allowlistMismatch returns every entry in marked that is absent from
// allowlist — the mismatch set TestCredentialScan_AllowlistMatchesEnumeratedList
// asserts is always empty for the real tree (S-ART-092), and which its
// own inline control proves is non-empty for a deliberately-unlisted
// marked file.
func allowlistMismatch(marked, allowlist []string) []string {
	allowSet := make(map[string]bool, len(allowlist))
	for _, a := range allowlist {
		allowSet[a] = true
	}
	var mismatch []string
	for _, m := range marked {
		if !allowSet[m] {
			mismatch = append(mismatch, m)
		}
	}
	return mismatch
}

// walkCredentialSurface recursively scans every scannableFile under root
// (D-3), reporting every file that matches a credential-shaped pattern
// and is NOT exempt. A file is exempt only when it BOTH carries the
// deliberate-plant marker AND appears in allowlist (relative to root) —
// a marker with no matching allowlist entry still fails, which is what
// makes a new plant a reviewable diff (R-ART-022 item 2/3). It never
// reproduces the matched bytes (R-ART-022 item 5).
func walkCredentialSurface(root string, allowlist []string) ([]scanResult, error) {
	var results []scanResult
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !scannableFile(path, d) {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		marked := strings.Contains(string(raw), deliberatePlantMarker)
		for _, sentinel := range credentialSentinels {
			if !sentinel.pattern.Match(raw) {
				continue
			}
			if marked && allowlistContains(allowlist, rel) {
				break // declared AND enumerated: exempt.
			}
			results = append(results, scanResult{relPath: rel, class: sentinel.class})
			break
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func allowlistContains(allowlist []string, relPath string) bool {
	for _, entry := range allowlist {
		if entry == relPath {
			return true
		}
	}
	return false
}

// markedFiles recursively finds every scannableFile under root carrying
// the deliberate-plant marker, regardless of whether it also matches a
// credential pattern, returning paths relative to root.
func markedFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !scannableFile(path, d) {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), deliberatePlantMarker) {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// verifyAllowlistFalsifiability confirms every entry in allowlist,
// relative to root, exists, carries the marker, AND still matches at
// least one credential pattern — a stale entry (the plant removed, or
// the file renamed) fails here, naming the entry (R-ART-022 item 3
// falsifiability: an allowlist that merely lists names without checking
// they still correspond to a real plant is not itself asserted).
func verifyAllowlistFalsifiability(root string, allowlist []string) error {
	for _, entry := range allowlist {
		path := filepath.Join(root, entry)
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("credential scan: allowlist entry %q: %w", entry, err)
		}
		if !strings.Contains(string(raw), deliberatePlantMarker) {
			return fmt.Errorf("credential scan: allowlist entry %q does not carry the deliberate-plant marker", entry)
		}
		matched := false
		for _, sentinel := range credentialSentinels {
			if sentinel.pattern.Match(raw) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("credential scan: allowlist entry %q no longer matches any credential-shaped pattern — the plant may have been cleaned up; remove it from the allowlist", entry)
		}
	}
	return nil
}

// scanCredentialSurface recursively scans every test file and fixture
// file under dir (D-3, R-ART-022) against the real, committed
// deliberatePlantAllowlist, and reports the first file (relative to dir)
// and sentinel class a not-exempted match is found in, or two empty
// strings when the surface is clean. Kept as a thin wrapper over
// walkCredentialSurface so the four pre-AI-36 tests below (S-ART-013…016)
// need no change.
func scanCredentialSurface(dir string) (file, class string, err error) {
	results, walkErr := walkCredentialSurface(dir, deliberatePlantAllowlist)
	if walkErr != nil {
		return "", "", walkErr
	}
	if len(results) == 0 {
		return "", "", nil
	}
	return results[0].relPath, results[0].class, nil
}

// TestCredentialScan_ExpectationSurfaceIsClean is the scan itself
// (S-ART-013): this milestone's own expectation-bearing sources — now the
// WHOLE adapter tree, recursively — carry no undeclared credential-shaped
// value.
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

// TestCredentialScan_InternalPackageFileWithoutMarker_IsSwept is AI-36's
// re-expression of S-ART-017 (design.md AD-4 item 4): the pre-AI-36 test
// of this same name proved the OPPOSITE — that an internal-package
// (`package openaicompat`) file was invisible to the scan by category.
// D-3 deliberately removes that category exemption; the identical
// construction now MUST be swept, because the obligation the old
// category-exemption satisfied (a legitimate deliberate plant does not
// break the build) is now satisfied by the declaration+allowlist rule
// instead — a real widening of coverage, not a relocation of the gap
// (S-ART-094).
func TestCredentialScan_InternalPackageFileWithoutMarker_IsSwept(t *testing.T) {
	dir := t.TempDir()
	planted := "s" + "k" + "-" + "ANINTERNALPACKAGEFIXTURENOWINSCOPE"
	content := "package openaicompat\n\nconst legacyToken = \"" + planted + "\"\n"
	writeScannableFile(t, dir, "internal_fixture_test.go", content)

	file, class, err := scanCredentialSurface(dir)
	if err != nil {
		t.Fatalf("scanning credential surface: %v", err)
	}
	if file != "internal_fixture_test.go" {
		t.Fatalf("file = %q, want internal_fixture_test.go — an internal-package file WITHOUT the marker must now be swept (D-3)", file)
	}
	if class != "OpenAI-style secret key" {
		t.Fatalf("class = %q, want OpenAI-style secret key", class)
	}
}

// TestCredentialScan_RecursiveWalk_FindsPlantInANestedDirectory proves
// S-ART-090: a credential-shaped value planted in a nested directory that
// the previous (os.ReadDir, single-directory) scan never read is now
// found — the widening proof, and its own control, since a scan that
// could never find anything nested would pass vacuously.
func TestCredentialScan_RecursiveWalk_FindsPlantInANestedDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	planted := "s" + "k" + "-" + "STAGEDNESTEDDIRECTORYPLANTFORWIDENING"
	content := "package openaicompat_test\n\nconst wantBody = \"" + planted + "\"\n"
	writeScannableFile(t, nested, "deep_plant_test.go", content)

	results, err := walkCredentialSurface(dir, nil)
	if err != nil {
		t.Fatalf("walkCredentialSurface: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("walkCredentialSurface found %d result(s), want 1 (the nested plant): %+v", len(results), results)
	}
	wantRel := filepath.Join("nested", "deeper", "deep_plant_test.go")
	if results[0].relPath != wantRel {
		t.Errorf("relPath = %q, want %q", results[0].relPath, wantRel)
	}
}

// TestCredentialScan_DeclarationRemoved_IsReported proves S-ART-091: a
// file carrying the marker AND appearing on the (synthetic, per-test)
// allowlist is not reported; the identical content with the declaration
// removed is reported again.
func TestCredentialScan_DeclarationRemoved_IsReported(t *testing.T) {
	planted := "s" + "k" + "-" + "STAGEDDECLARATIONREMOVALTEST"
	markerLine := "// " + deliberatePlantMarker + " — synthetic fixture for S-ART-091\n"

	t.Run("marker present and allowlisted: not reported", func(t *testing.T) {
		dir := t.TempDir()
		content := markerLine + "package openaicompat_test\n\nconst wantBody = \"" + planted + "\"\n"
		writeScannableFile(t, dir, "declared_test.go", content)

		results, err := walkCredentialSurface(dir, []string{"declared_test.go"})
		if err != nil {
			t.Fatalf("walkCredentialSurface: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("walkCredentialSurface found %d result(s), want 0 — a marker+allowlist entry must exempt it: %+v", len(results), results)
		}
	})

	t.Run("identical content, declaration removed: reported", func(t *testing.T) {
		dir := t.TempDir()
		content := "package openaicompat_test\n\nconst wantBody = \"" + planted + "\"\n"
		writeScannableFile(t, dir, "declared_test.go", content)

		results, err := walkCredentialSurface(dir, []string{"declared_test.go"})
		if err != nil {
			t.Fatalf("walkCredentialSurface: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("walkCredentialSurface found %d result(s), want 1 — removing the marker must make it reported again (S-ART-091)", len(results))
		}
	})
}

// TestCredentialScan_AllowlistMatchesEnumeratedList proves S-ART-092: the
// set of declaration-bearing files under the real tree matches the
// committed allowlist exactly, in both directions, with an inline control
// proving the comparison itself can fail.
func TestCredentialScan_AllowlistMatchesEnumeratedList(t *testing.T) {
	marked, err := markedFiles(".")
	if err != nil {
		t.Fatalf("markedFiles: %v", err)
	}

	if mismatch := allowlistMismatch(marked, deliberatePlantAllowlist); len(mismatch) != 0 {
		t.Errorf("marked file(s) absent from the committed allowlist, want none: %v (S-ART-092)", mismatch)
	}

	markedSet := make(map[string]bool, len(marked))
	for _, m := range marked {
		markedSet[m] = true
	}
	for _, a := range deliberatePlantAllowlist {
		if !markedSet[a] {
			t.Errorf("allowlist entry %q does not carry the deliberate-plant marker (S-ART-092)", a)
		}
	}

	t.Run("inline control: a marked file absent from the allowlist is caught", func(t *testing.T) {
		if mismatch := allowlistMismatch([]string{"newly_planted_test.go"}, nil); len(mismatch) != 1 {
			t.Fatalf("allowlistMismatch did not flag an unlisted marked file, want exactly 1 mismatch, got %v — the comparison must be falsifiable (S-ART-092)", mismatch)
		}
	})
}

// TestCredentialScan_AllowlistFalsifiability proves the mandatory
// falsifiability requirement (design.md AD-4 item 3): every real
// allowlist entry still corresponds to a live plant, and a synthetic,
// stale entry (the plant "cleaned up", pattern no longer matching) fails
// the check by name.
func TestCredentialScan_AllowlistFalsifiability(t *testing.T) {
	if err := verifyAllowlistFalsifiability(".", deliberatePlantAllowlist); err != nil {
		t.Errorf("verifyAllowlistFalsifiability(real tree) = %v, want nil — every committed allowlist entry must still correspond to a real plant", err)
	}

	t.Run("a stale entry (plant cleaned up) fails the check", func(t *testing.T) {
		dir := t.TempDir()
		content := "package openaicompat_test\n\n// " + deliberatePlantMarker + " — was a plant, now cleaned up\nconst wantBody = \"no-longer-credential-shaped\"\n"
		writeScannableFile(t, dir, "stale_test.go", content)

		if err := verifyAllowlistFalsifiability(dir, []string{"stale_test.go"}); err == nil {
			t.Fatal("verifyAllowlistFalsifiability = nil, want a non-nil error: the marked file no longer matches any credential pattern")
		}
	})

	t.Run("a missing file fails the check", func(t *testing.T) {
		dir := t.TempDir()
		if err := verifyAllowlistFalsifiability(dir, []string{"does_not_exist_test.go"}); err == nil {
			t.Fatal("verifyAllowlistFalsifiability = nil, want a non-nil error: the allowlist entry names a file that does not exist")
		}
	})
}

// TestCredentialScan_NoReprintOfMatchedValue proves S-ART-093: a scan
// report names the file and the vector only, never the matched bytes; and
// the scan's own sources, the declaration text and the committed list are
// never self-reported when the real tree is scanned.
func TestCredentialScan_NoReprintOfMatchedValue(t *testing.T) {
	dir := t.TempDir()
	planted := "s" + "k" + "-" + "STAGEDNOREPRINTTEST1234567890"
	content := "package openaicompat_test\n\nconst wantBody = \"" + planted + "\"\n"
	writeScannableFile(t, dir, "reprint_test.go", content)

	results, err := walkCredentialSurface(dir, nil)
	if err != nil {
		t.Fatalf("walkCredentialSurface: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("walkCredentialSurface found %d result(s), want 1", len(results))
	}
	if strings.Contains(results[0].relPath, planted) || strings.Contains(results[0].class, planted) {
		t.Errorf("scan result reproduces the matched value: %+v", results[0])
	}
	if results[0].relPath != "reprint_test.go" || results[0].class == "" {
		t.Errorf("scan result = %+v, want a file name and a non-empty class only", results[0])
	}

	t.Run("the scan's own sources, declaration text and committed list are never self-reported", func(t *testing.T) {
		results, err := walkCredentialSurface(".", deliberatePlantAllowlist)
		if err != nil {
			t.Fatalf("walkCredentialSurface(real tree): %v", err)
		}
		if len(results) != 0 {
			t.Errorf("walkCredentialSurface(real tree) reported %d violation(s), want 0: %+v", len(results), results)
		}
	})
}
