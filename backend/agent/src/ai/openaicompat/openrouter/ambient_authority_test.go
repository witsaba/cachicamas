// R-OR-01's mechanical guard for the openrouter sub-package: a
// call-site scan over this package's own non-test sources, denying by
// default at package granularity rather than admitting by a curated
// function list. The mechanism is openaicompat's own — AI-25.2's
// "guard" node, verbatim — adapted to scan this sub-package's
// directory rather than openaicompat's.
//
// # Why this is not AI-00.3's mechanism
//
// This repository's sibling guard (import_boundary_test.go in package
// ai) scans the transitive *import* closure. That mechanism cannot
// express this rule: the required net/http transport itself
// transitively imports "os", so an import-closure scan must either
// admit "os" everywhere — missing a narrow os.Getenv call inside
// this package — or forbid it everywhere, false-positiving on
// legitimate transport use. AI-25.2's correction (doc 0002 amendment
// A1) is to scan *call sites* within this package's own files. This
// file lifts that mechanism — same forbidden-package table, same
// file-selection rule, same AST walk.
//
// # Why non-test sources only (R-APC-011 inheritance)
//
// Excluding test sources is what makes the legitimate use of
// net/http/httptest and os (for os.ReadDir, the scan's own walk) in
// this file not false-positive the guard. The exclusion is uniform —
// every _test.go file is excluded, including this guard's own source
// (S-APC-045). AI-00.3's own _test.go file freely uses os/exec to run
// `go list` to implement its check, and that pattern is mirrored
// here: this file's `forbiddenAmbientAuthorityPackages` walk uses
// os.ReadDir, which would fail its own scan if test sources were not
// excluded.
//
// # Recorded limitation (inherited from openaicompat/ambient_authority_test.go)
//
// This scan resolves an identifier to a package by its import
// declaration alone; it carries no type information. A local variable,
// parameter or struct field literally named "os" that did not refer
// to the imported package would still be flagged as if it were a
// call into that package. Closing this gap would need go/types at
// minimum, realistically golang.org/x/tools/go/analysis — and that is
// a non-standard-library dependency this milestone will not add
// (NFR-APC-A). The recorded limitation is the same shape
// openaicompat's own guard documents (R-APC-012). No such shadow
// exists in this package's own sources today.
//
// # Scope (cross-referenced from doc.go)
//
// This guard, and the guarantee it enforces, covers only this
// sub-package's own sources and the client this sub-package builds
// through NewProvider. It does not, and cannot, cover an injected
// client's transport: see openaicompat/doc.go, "The no-ambient-
// authority guarantee is scoped, not absolute".

package openrouter

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

// forbiddenAmbientAuthorityPackage names one package this guard
// denies by default (R-OR-01, R-APC-008 inheritance). Denial is by
// package, never by a curated function list — a function added later
// to an already-forbidden package is caught without updating this
// table. Each entry records the rule it enforces, so a failure
// message can name it (S-APC-035 inheritance).
type forbiddenAmbientAuthorityPackage struct {
	importPath       string
	defaultLocalName string
	rule             string
}

// forbiddenAmbientAuthorityPackages is the guard's whole forbidden
// set: the operating-system package, the process-execution package,
// the low-level system-call package, and the deprecated legacy I/O
// package — broader than the charter's literal three verbs
// (environment, filesystem, process) by design; that breadth is
// intended belt-and-braces (S-APC-034 inheritance).
var forbiddenAmbientAuthorityPackages = []forbiddenAmbientAuthorityPackage{
	{
		importPath:       "os",
		defaultLocalName: "os",
		rule:             "R-OR-01: no environment variable read, no filesystem path touched",
	},
	{
		importPath:       "os/exec",
		defaultLocalName: "exec",
		rule:             "R-OR-01: no process spawned",
	},
	{
		importPath:       "syscall",
		defaultLocalName: "syscall",
		rule:             "R-OR-01: no low-level system call — belt-and-braces beyond the charter's three named verbs",
	},
	{
		importPath:       "io/ioutil",
		defaultLocalName: "ioutil",
		rule:             "R-OR-01: the deprecated legacy I/O package — belt-and-braces beyond the charter's three named verbs",
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

// isAdapterSourceFile reports whether name is a file this guard
// scans: a ".go" file that is not a "_test.go" file. This is the
// entire file-selection rule (R-APC-011 inheritance) — it excludes
// every test file uniformly, including this guard's own source.
func isAdapterSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// scanNonTestSourcesForAmbientAuthority scans every file
// isAdapterSourceFile admits in dir and returns one message-ready
// violation string per forbidden call site or forbidden dot-import
// found. Each message names its file, line and forbidden package
// (S-APC-035 inheritance).
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
		if entry.IsDir() || !isAdapterSourceFile(name) {
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
// identifiers (alias-aware, via file.Imports) and flags every call
// site whose selector resolves to a forbidden package, plus every
// dot-import of a forbidden package in its own right, independent of
// any call site.
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
			// Blank import: inert. It binds no callable identifier,
			// and an unused non-blank import fails compilation, so a
			// blank import of a forbidden package can never reach a
			// call site.
			continue

		case imp.Name != nil && imp.Name.Name == ".":
			// A dot-import is a violation in its own right
			// (R-APC-010 item 4, S-APC-041 inheritance): resolving a
			// bare identifier back to this import would need full
			// type information (the recorded limitation above), so
			// the import itself is flagged regardless of whether a
			// call to it is found.
			pos := fset.Position(imp.Pos())
			violations = append(violations, fmt.Sprintf(
				"%s:%d: dot-import of forbidden package %q (%s)",
				fileName, pos.Line, importPath, entry.rule))
			continue

		case imp.Name != nil:
			// A named alias (S-APC-039 inheritance): resolved by its
			// local identifier, not by the import path, so renaming
			// the import does not bypass the scan.
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

// TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage is
// the guard's own assertion (R-OR-01, S-OR-01 sub-scenario 1): this
// sub-package's non-test sources carry no forbidden-package call
// site. The guard is green from birth over clean, landed sources —
// the falsifiability proof below (the staged-mutation bit-proof) is
// what makes the assertion non-vacuous.
func TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage(t *testing.T) {
	t.Parallel()

	violations := scanNonTestSourcesForAmbientAuthority(t, ".")
	for _, violation := range violations {
		t.Errorf("ambient authority: %s", violation)
	}
}

// TestOpenRouterAdapter_ForbiddenSetIsPackageScopedDenyByDefault
// covers the deny-by-default posture: the forbidden set names the
// operating-system, process-execution, system-call and legacy-I/O
// packages by package — never by a curated function name — and
// nothing else. Belt-and-braces beyond the charter's literal three
// named verbs (S-APC-034 inheritance).
func TestOpenRouterAdapter_ForbiddenSetIsPackageScopedDenyByDefault(t *testing.T) {
	t.Parallel()

	want := []string{"io/ioutil", "os", "os/exec", "syscall"}
	got := make([]string, 0, len(forbiddenAmbientAuthorityPackages))
	for _, entry := range forbiddenAmbientAuthorityPackages {
		got = append(got, entry.importPath)
	}
	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("forbidden ambient-authority package set = %v, want %v (R-OR-01 deny-by-default)", got, want)
	}
}

// TestOpenRouterAdapter_AmbientAuthorityFailsOnStagedMutation is the
// falsifiability proof: a planted scratch file with a forbidden
// call site in a t.TempDir() fails the scan, naming the file, line,
// and forbidden package — with no edit to the scan itself. This is
// the bite-proof the green-from-birth assertion above needs: a
// guard that only ever passes is unfalsifiable, and this staged
// mutation proves the scan actually catches what it claims to
// catch. The planted file is then removed by t.Cleanup so it does
// not pollute the working tree.
func TestOpenRouterAdapter_AmbientAuthorityFailsOnStagedMutation(t *testing.T) {
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
		t.Fatal("scanNonTestSourcesForAmbientAuthority returned zero violations for a planted os.Getenv call site, want at least one (R-OR-01 bite-proof)")
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

// TestOpenRouterAdapter_AmbientAuthorityIsAdapterSourceFile covers
// the file-selection half: the scan's exclusion rule is a uniform
// suffix rule, not a list of named exceptions, so this guard's own
// source is excluded by the same rule that excludes every other
// test file — never a special case naming it (S-APC-045
// inheritance).
func TestOpenRouterAdapter_AmbientAuthorityIsAdapterSourceFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want bool
	}{
		{"a production source is included", "wrapper.go", true},
		{"a wrapper test source is excluded", "wrapper_test.go", false},
		{"the guard's own source is excluded, not special-cased", "ambient_authority_test.go", false},
		{"a non-Go file is excluded", "README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isAdapterSourceFile(tt.file); got != tt.want {
				t.Errorf("isAdapterSourceFile(%q) = %v, want %v (R-OR-01 file-selection uniformity)", tt.file, got, tt.want)
			}
		})
	}
}

// TestOpenRouterAdapter_AmbientAuthorityTestSourcesStayGreenEvenWithForbiddenCalls
// covers the test-source exclusion: a fixture "_test.go" file that
// itself calls a forbidden package is still scanned clean, because
// test sources are excluded by construction (S-APC-044 inheritance).
func TestOpenRouterAdapter_AmbientAuthorityTestSourcesStayGreenEvenWithForbiddenCalls(t *testing.T) {
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
		t.Errorf("scan reported violations from a _test.go fixture, want none (S-APC-044 inheritance): %v", violations)
	}
}
