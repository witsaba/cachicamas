// CH-04.2 — the no-ambient-authority guard for the chat archetype
// (R-06, design AD-5). The chat package performs no I/O of its own:
// no environment read, no filesystem touch, no process spawn.
//
// This file lifts backend/agent/src/agent/ambient_authority_test.go
// (AG-03.3's "guard" node) WHOLESALE, retargeted from `package agent`
// to `package chat_test`. Same forbidden set, same alias/dot/blank-
// import handling, same uniform _test.go exclusion, same t.TempDir()
// staged-mutation falsifiability test, same recorded no-type-
// information limitation.
//
// # Why this is not import_boundary_test.go's forward guard
//
// "os" is standard library, so a bare os.Getenv call inside the chat
// archetype would pass the forward guard's deny-by-default allowlist
// silently — listNonStdlibDeps (import_boundary_test.go) filters the
// standard library out before either of its allowlists ever runs.
// An import-closure scan also cannot distinguish a narrow environment
// read from a legitimate transitive dependency that happens to reach
// "os" for an unrelated reason. AG-03.3's correction (doc 0002
// amendment A1) is to scan CALL SITES within the package's own files
// instead of its import closure. This file is that scan, retargeted
// to the chat archetype.
//
// # Why non-test sources only
//
// Excluding test sources is what lets this file's own os.ReadDir call
// (in scanNonTestSourcesForAmbientAuthority, below) not false-positive
// the guard. The exclusion is uniform — every _test.go file is excluded,
// including this guard's own source, by the same suffix rule, never a
// name check naming this file specifically.
//
// # Recorded limitation (inherited from AG-03.3)
//
// This scan resolves an identifier to a package by its import
// declaration alone; it carries no type information. A local variable,
// parameter or struct field literally named "os" that did not refer to
// the imported package would still be flagged as if it were a call into
// that package. Closing this gap would need go/types at minimum,
// realistically golang.org/x/tools/go/analysis — a non-standard-library
// dependency no milestone has authorised. No such shadow exists in this
// package's own sources today.
package chat_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// forbiddenAmbientAuthorityPackage names one package this guard denies by
// default (R-06, design AD-5). Denial is by package, never by a curated
// function list — a function added later to an already-forbidden package
// is caught without updating this table. Each entry records the rule it
// enforces, so a failure message can name it.
type forbiddenAmbientAuthorityPackage struct {
	importPath       string
	defaultLocalName string
	rule             string
}

// forbiddenAmbientAuthorityPackages is the guard's whole forbidden set:
// the operating-system package, the process-execution package, the
// low-level system-call package, and the deprecated legacy I/O package —
// broader than "no environment read, no filesystem access, no process
// spawn" by design; that breadth is intended belt-and-braces, mirroring
// AG-03.3's own guard.
var forbiddenAmbientAuthorityPackages = []forbiddenAmbientAuthorityPackage{
	{
		importPath:       "os",
		defaultLocalName: "os",
		rule:             "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the chat archetype performs no I/O of its own — no environment variable read, no filesystem path touched",
	},
	{
		importPath:       "os/exec",
		defaultLocalName: "exec",
		rule:             "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the chat archetype performs no I/O of its own — no process spawned",
	},
	{
		importPath:       "syscall",
		defaultLocalName: "syscall",
		rule:             "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: no low-level system call — belt-and-braces beyond the charter's three named verbs",
	},
	{
		importPath:       "io/ioutil",
		defaultLocalName: "ioutil",
		rule:             "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the deprecated legacy I/O package — belt-and-braces beyond the charter's three named verbs",
	},
}

// findForbiddenAmbientAuthorityPackage looks importPath up in
// forbiddenAmbientAuthorityPackages.
func findForbiddenAmbientAuthorityPackage(importPath string) (forbiddenAmbientAuthorityPackage, bool) {
	for _, entry := range forbiddenAmbientAuthorityPackages {
		if entry.importPath == importPath {
			return entry, true
		}
	}
	return forbiddenAmbientAuthorityPackage{}, false
}

// isChatArchetypeSourceFile reports whether name is a file this guard
// scans: a ".go" file that is not a "_test.go" file. This is the entire
// file-selection rule — it excludes every test file uniformly, including
// this guard's own source.
func isChatArchetypeSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// scanNonTestSourcesForAmbientAuthority scans every file
// isChatArchetypeSourceFile admits in dir and returns one message-ready
// violation string per forbidden call site or forbidden dot-import
// found. Each message names its file, line and forbidden package.
//
// RED-stage stub (CH-04.2 work unit 1.1): the scan returns an empty
// slice. The bite-proof test (TestChatArchetype_FailsOnStagedMutation)
// plants an os.Getenv fixture and expects ≥1 violation; with this stub
// the test fails — proving the bite-proof is wired and the test is not
// vacuous. GREEN (work unit 1.2) replaces this stub with the real
// AST walk that flags each forbidden call site.
func scanNonTestSourcesForAmbientAuthority(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}

	_ = entries // RED stub: directory listing is read but no AST walk runs yet.
	return nil
}

// TestChatArchetype_NonTestSourcesCarryNoForbiddenCallSite is the guard's
// own assertion (R-06, design AD-5): this package's non-test sources
// carry no forbidden-package call site. The guard is green from birth
// over clean, landed sources — the falsifiability proof below (the
// staged-mutation bite-proof) is what makes the assertion non-vacuous.
//
// It also fences the vacuous pass explicitly: scanNonTestSourcesForAmbientAuthority
// itself reports zero violations for a directory with no matching
// source file, indistinguishable from a clean pass, so this test counts
// isChatArchetypeSourceFile matches independently and fatals if the count is
// zero — the same vacuous-pass posture import_boundary_test.go's `go
// list` checks use.
func TestChatArchetype_NonTestSourcesCarryNoForbiddenCallSite(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\") error = %v, want nil", err)
	}
	var inspected int
	for _, entry := range entries {
		if !entry.IsDir() && isChatArchetypeSourceFile(entry.Name()) {
			inspected++
		}
	}
	if inspected == 0 {
		t.Fatal("no non-test source file found in \".\"; the guard would pass vacuously")
	}
	t.Logf("ambient-authority scan inspected %d non-test source file(s)", inspected)

	violations := scanNonTestSourcesForAmbientAuthority(t, ".")
	for _, violation := range violations {
		t.Errorf("ambient authority: %s", violation)
	}
}

// TestChatArchetype_ForbiddenSetIsPackageScopedDenyByDefault covers the
// deny-by-default posture: the forbidden set names the operating-system,
// process-execution, system-call and legacy-I/O packages by package —
// never by a curated function name — and nothing else.
func TestChatArchetype_ForbiddenSetIsPackageScopedDenyByDefault(t *testing.T) {
	t.Parallel()

	want := []string{"io/ioutil", "os", "os/exec", "syscall"}
	got := make([]string, 0, len(forbiddenAmbientAuthorityPackages))
	for _, entry := range forbiddenAmbientAuthorityPackages {
		got = append(got, entry.importPath)
	}
	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("forbidden ambient-authority package set = %v, want %v (R-06 deny-by-default)", got, want)
	}
}

// TestChatArchetype_FailsOnStagedMutation is the falsifiability proof: a
// planted scratch file with a forbidden call site in a t.TempDir() fails
// the scan, naming the file, line, and forbidden package — with no edit
// to the scan itself. This is the bite-proof the green-from-birth
// assertion above needs: a guard that only ever passes is unfalsifiable,
// and this staged mutation proves the scan actually catches what it
// claims to catch; the planted file is then removed by t.Cleanup so it
// does not pollute the working tree.
func TestChatArchetype_FailsOnStagedMutation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = `package fixture

import "os"

func readSomethingForbidden() string {
	return os.Getenv("SCRATCH_ONLY")
}
`
	path := filepath.Join(dir, "staged_ambient_authority_violation.go")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	violations := scanNonTestSourcesForAmbientAuthority(t, dir)
	if len(violations) == 0 {
		t.Fatal("scanNonTestSourcesForAmbientAuthority returned zero violations for a planted os.Getenv call site, want at least one (R-06 bite-proof)")
	}

	var sawOsGetenv bool
	for _, violation := range violations {
		if strings.Contains(violation, "os.Getenv") && strings.Contains(violation, "staged_ambient_authority_violation.go") {
			sawOsGetenv = true
			break
		}
	}
	if !sawOsGetenv {
		t.Errorf("scan missed the staged os.Getenv call site: violations = %v", violations)
	}
}

// TestChatArchetype_FileSelectionIsUniform covers the file-selection half:
// the scan's exclusion rule is a uniform suffix rule, not a list of
// named exceptions, so this guard's own source is excluded by the same
// rule that excludes every other test file — never a special case
// naming it.
func TestChatArchetype_FileSelectionIsUniform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want bool
	}{
		{"a production source is included", "doc.go", true},
		{"a chat-surface test source is excluded", "http_test.go", false},
		{"the guard's own source is excluded, not special-cased", "ambient_authority_test.go", false},
		{"a non-Go file is excluded", "README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isChatArchetypeSourceFile(tt.file); got != tt.want {
				t.Errorf("isChatArchetypeSourceFile(%q) = %v, want %v (R-06 file-selection uniformity)", tt.file, got, tt.want)
			}
		})
	}
}

// TestChatArchetype_TestSourcesStayGreenEvenWithForbiddenCalls covers the
// test-source exclusion: a fixture "_test.go" file that itself calls a
// forbidden package is still scanned clean, because test sources are
// excluded by construction.
func TestChatArchetype_TestSourcesStayGreenEvenWithForbiddenCalls(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = `package fixture

import "os"

func readSomethingATestMightNeed() string {
	return os.Getenv("SCRATCH_ONLY")
}
`
	if err := os.WriteFile(filepath.Join(dir, "fixture_test.go"), []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	violations := scanNonTestSourcesForAmbientAuthority(t, dir)
	if len(violations) != 0 {
		t.Errorf("scan reported violations from a _test.go fixture, want none: %v", violations)
	}
}