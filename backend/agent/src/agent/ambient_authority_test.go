// AG-03.3 — the no-ambient-authority guard: Layer 2 performs no I/O of
// its own (R-AGP-005, design AD-5).
//
// This file lifts openaicompat/openrouter/ambient_authority_test.go
// (AI-25.2's "guard" node) WHOLESALE, retargeted from the openrouter
// adapter sub-package to this package: same forbidden set, same
// alias/dot/blank-import handling, same uniform _test.go exclusion, same
// t.TempDir() staged-mutation falsifiability test, same recorded
// no-type-information limitation.
//
// # Why this is not import_boundary_test.go's forward guard
//
// "os" is standard library, so a bare os.Getenv call inside Layer 2 would
// pass the forward guard's deny-by-default allowlist silently —
// listNonStdlibDeps (import_boundary_test.go) filters the standard
// library out before either of its allowlists ever run. An import-closure
// scan also cannot distinguish a narrow environment read from a
// legitimate transitive dependency that happens to reach "os" for an
// unrelated reason. AI-25.2's correction (doc 0002 amendment A1) is to
// scan CALL SITES within the package's own files instead of its import
// closure. This file is that scan, retargeted to Layer 2.
//
// # Why non-test sources only
//
// Excluding test sources is what lets this file's own os.ReadDir call
// (in scanNonTestSourcesForAmbientAuthority, below) not false-positive
// the guard. The exclusion is uniform — every _test.go file is excluded,
// including this guard's own source, by the same suffix rule, never a
// name check naming this file specifically.
//
// # Recorded limitation (inherited from openaicompat's own guard)
//
// This scan resolves an identifier to a package by its import
// declaration alone; it carries no type information. A local variable,
// parameter or struct field literally named "os" that did not refer to
// the imported package would still be flagged as if it were a call into
// that package. Closing this gap would need go/types at minimum,
// realistically golang.org/x/tools/go/analysis — a non-standard-library
// dependency no milestone has authorised. No such shadow exists in this
// package's own sources today.
package agent_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// forbiddenAmbientAuthorityPackage names one package this guard denies by
// default (R-AGP-005). Denial is by package, never by a curated function
// list — a function added later to an already-forbidden package is
// caught without updating this table. Each entry records the rule it
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
// spawn" (doc.go's L2C-02) by design; that breadth is intended
// belt-and-braces, mirroring openaicompat's own guard.
var forbiddenAmbientAuthorityPackages = []forbiddenAmbientAuthorityPackage{
	{
		importPath:       "os",
		defaultLocalName: "os",
		rule:             "ADR 0005 § D1 row 2 / AG-00: Layer 2 performs no I/O of its own — no environment variable read, no filesystem path touched",
	},
	{
		importPath:       "os/exec",
		defaultLocalName: "exec",
		rule:             "ADR 0005 § D1 row 2 / AG-00: Layer 2 performs no I/O of its own — no process spawned",
	},
	{
		importPath:       "syscall",
		defaultLocalName: "syscall",
		rule:             "ADR 0005 § D1 row 2 / AG-00: no low-level system call — belt-and-braces beyond the charter's three named verbs",
	},
	{
		importPath:       "io/ioutil",
		defaultLocalName: "ioutil",
		rule:             "ADR 0005 § D1 row 2 / AG-00: the deprecated legacy I/O package — belt-and-braces beyond the charter's three named verbs",
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

// isLayer2SourceFile reports whether name is a file this guard scans: a
// ".go" file that is not a "_test.go" file. This is the entire
// file-selection rule (S-AGP-054) — it excludes every test file
// uniformly, including this guard's own source.
func isLayer2SourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// scanNonTestSourcesForAmbientAuthority scans every file
// isLayer2SourceFile admits in dir and returns one message-ready
// violation string per forbidden call site or forbidden dot-import
// found. Each message names its file, line and forbidden package.
func scanNonTestSourcesForAmbientAuthority(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}

	fset := token.NewFileSet()
	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isLayer2SourceFile(name) {
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v, want nil", path, err)
		}

		violations = append(violations, scanFileForAmbientAuthority(fset, name, file)...)
	}
	return violations
}

// scanFileForAmbientAuthority resolves file's imports to local
// identifiers (alias-aware, via file.Imports) and flags every call site
// whose selector resolves to a forbidden package, plus every dot-import
// of a forbidden package in its own right, independent of any call site.
func scanFileForAmbientAuthority(fset *token.FileSet, fileName string, file *ast.File) []string {
	localNameToPackage := make(map[string]forbiddenAmbientAuthorityPackage)
	var violations []string

	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			importPath = strings.Trim(imp.Path.Value, `"`)
		}
		entry, forbidden := findForbiddenAmbientAuthorityPackage(importPath)
		if !forbidden {
			continue
		}

		switch {
		case imp.Name != nil && imp.Name.Name == "_":
			// Blank import: inert. It binds no callable identifier, and
			// an unused non-blank import fails compilation, so a blank
			// import of a forbidden package can never reach a call
			// site.
			continue

		case imp.Name != nil && imp.Name.Name == ".":
			// A dot-import is a violation in its own right: resolving a
			// bare identifier back to this import would need full type
			// information (the recorded limitation above), so the
			// import itself is flagged regardless of whether a call to
			// it is found.
			pos := fset.Position(imp.Pos())
			violations = append(violations, fmt.Sprintf(
				"%s:%d: dot-import of forbidden package %q (%s)",
				fileName, pos.Line, importPath, entry.rule))
			continue

		case imp.Name != nil:
			// A named alias: resolved by its local identifier, not by
			// the import path, so renaming the import does not bypass
			// the scan.
			localNameToPackage[imp.Name.Name] = entry

		default:
			localNameToPackage[entry.defaultLocalName] = entry
		}
	}

	if len(localNameToPackage) == 0 {
		return violations
	}

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		entry, forbidden := localNameToPackage[ident.Name]
		if !forbidden {
			return true
		}

		pos := fset.Position(call.Pos())
		violations = append(violations, fmt.Sprintf(
			"%s:%d: call %s.%s reaches forbidden package %q (%s)",
			fileName, pos.Line, ident.Name, sel.Sel.Name, entry.importPath, entry.rule))
		return true
	})

	return violations
}

// TestLayer2Agent_NonTestSourcesCarryNoForbiddenCallSite is the guard's
// own assertion (R-AGP-005, S-AGP-050): this package's non-test sources
// carry no forbidden-package call site. The guard is green from birth
// over clean, landed sources — the falsifiability proof below (the
// staged-mutation bite-proof) is what makes the assertion non-vacuous.
//
// It also fences the vacuous pass explicitly (S-AGP-050's "reports having
// inspected at least one source file"): scanNonTestSourcesForAmbientAuthority
// itself reports zero violations for a directory with no matching
// source file, indistinguishable from a clean pass, so this test counts
// isLayer2SourceFile matches independently and fatals if the count is
// zero — the same vacuous-pass posture import_boundary_test.go's `go
// list` checks use.
func TestLayer2Agent_NonTestSourcesCarryNoForbiddenCallSite(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\") error = %v, want nil", err)
	}
	var inspected int
	for _, entry := range entries {
		if !entry.IsDir() && isLayer2SourceFile(entry.Name()) {
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

// TestLayer2Agent_ForbiddenSetIsPackageScopedDenyByDefault covers the
// deny-by-default posture: the forbidden set names the operating-system,
// process-execution, system-call and legacy-I/O packages by package —
// never by a curated function name — and nothing else.
func TestLayer2Agent_ForbiddenSetIsPackageScopedDenyByDefault(t *testing.T) {
	t.Parallel()

	want := []string{"io/ioutil", "os", "os/exec", "syscall"}
	got := make([]string, 0, len(forbiddenAmbientAuthorityPackages))
	for _, entry := range forbiddenAmbientAuthorityPackages {
		got = append(got, entry.importPath)
	}
	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("forbidden ambient-authority package set = %v, want %v (R-AGP-005 deny-by-default)", got, want)
	}
}

// TestLayer2Agent_FailsOnStagedMutation is the falsifiability proof: a
// planted scratch file with a forbidden call site in a t.TempDir() fails
// the scan, naming the file, line, and forbidden package — with no edit
// to the scan itself. This is the bite-proof the green-from-birth
// assertion above needs: a guard that only ever passes is unfalsifiable,
// and this staged mutation proves the scan actually catches what it
// claims to catch (S-AGP-055, kept permanent so the falsifiability proof
// does not depend on a manually-planted-then-removed scratch file). The
// planted file is then removed by t.Cleanup so it does not pollute the
// working tree.
func TestLayer2Agent_FailsOnStagedMutation(t *testing.T) {
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
		t.Fatal("scanNonTestSourcesForAmbientAuthority returned zero violations for a planted os.Getenv call site, want at least one (R-AGP-005 bite-proof)")
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

// TestLayer2Agent_FileSelectionIsUniform covers the file-selection half:
// the scan's exclusion rule is a uniform suffix rule, not a list of
// named exceptions, so this guard's own source is excluded by the same
// rule that excludes every other test file — never a special case
// naming it (S-AGP-054).
func TestLayer2Agent_FileSelectionIsUniform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want bool
	}{
		{"a production source is included", "doc.go", true},
		{"a doc-contract-guard test source is excluded", "doc_contract_guard_test.go", false},
		{"the guard's own source is excluded, not special-cased", "ambient_authority_test.go", false},
		{"a non-Go file is excluded", "README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isLayer2SourceFile(tt.file); got != tt.want {
				t.Errorf("isLayer2SourceFile(%q) = %v, want %v (R-AGP-005 file-selection uniformity)", tt.file, got, tt.want)
			}
		})
	}
}

// TestLayer2Agent_TestSourcesStayGreenEvenWithForbiddenCalls covers the
// test-source exclusion: a fixture "_test.go" file that itself calls a
// forbidden package is still scanned clean, because test sources are
// excluded by construction.
func TestLayer2Agent_TestSourcesStayGreenEvenWithForbiddenCalls(t *testing.T) {
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
