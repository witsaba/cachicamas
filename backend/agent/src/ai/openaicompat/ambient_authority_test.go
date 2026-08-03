package openaicompat

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

// This file is AI-25.2's [guard] node (R-APC-008): a mechanical call-site
// scan over this package's own non-test sources, denying by default at
// package granularity rather than admitting by a curated function list.
//
// # Why this is not AI-00.3's mechanism
//
// This repository's sibling guard (import_boundary_test.go) scans the
// transitive *import* closure. That mechanism cannot express this rule: the
// required net/http transport itself transitively imports "os", so an
// import-closure scan must either admit "os" everywhere — missing a narrow
// os.Getenv call inside this package — or forbid it everywhere, false-
// positiving on legitimate transport use. Doc 0002's AI-25.2 node records
// this correction as amendment A1. This guard instead scans *call sites*
// within this package's own files, the scope amendment A1 requires.
//
// # Why non-test sources only (R-APC-011)
//
// Scanning test sources would pull in the testing package, which itself
// imports "os" — the same reason this repository's own request-path guard
// (import_boundary_test.go's TestRequestPath_DependencyClosure...) already
// gives for excluding "-test" from its narrower scan: "-test would also
// pull in \"testing\", which imports \"os\" itself... a guard that scanned
// test imports could never pass." This package's own tests legitimately use
// the standard-library HTTP test server (timeout_test.go imports
// net/http/httptest) — excluding test sources is what makes that
// legitimate, not an oversight (S-APC-043, S-APC-044). The exclusion also
// exempts this guard's own source, ambient_authority_test.go itself,
// exactly as AI-00.3 freely uses os/exec to run `go list` to implement its
// own check (S-APC-045) — it is not a special case naming this file; see
// isAdapterSourceFile below, which excludes it by the same uniform rule
// that excludes every other test file.
//
// # Recorded limitation: a local shadow can false-positive (R-APC-012)
//
// This scan resolves an identifier to a package by its import declaration
// alone; it carries no type information. A local variable, parameter or
// struct field literally named "os" (or "exec", "syscall", "ioutil") that
// does not refer to the imported package would still be flagged as if it
// were a call into that package. Closing this gap would need a full
// type-checking pass — go/types at minimum, realistically
// golang.org/x/tools/go/analysis — and that is a non-standard-library
// dependency this milestone will not add: it would spend the
// zero-module-requires invariant AI-24 bought (NFR-APC-A). This is recorded
// here as an accepted, reversible non-requirement scoped to this change
// (S-APC-046, S-APC-047), not a permanent prohibition: a later milestone may
// revisit it by writing the ADR a new static-analysis dependency would
// need. No such shadow exists in this package's own sources today.
//
// # Scope (cross-referenced from doc.go, S-APC-068)
//
// This guard, and the guarantee it enforces, covers only this package's own
// sources and the client this package builds for itself. It does not, and
// cannot, cover an injected client's transport: see doc.go, "The
// no-ambient-authority guarantee is scoped, not absolute".

// forbiddenAmbientAuthorityPackage names one package this guard denies by
// default (R-APC-008, S-APC-034). Denial is by package, never by a curated
// function list, so a function added later to an already-forbidden package
// is caught without updating this table. Each entry records the rule it
// enforces, so a failure message can name it (S-APC-035).
type forbiddenAmbientAuthorityPackage struct {
	// importPath is the forbidden package's full import path.
	importPath string
	// defaultLocalName is the identifier an unaliased import of importPath
	// binds — the name the scan resolves a plain, non-aliased import to.
	defaultLocalName string
	// rule documents which requirement this entry enforces.
	rule string
}

// forbiddenAmbientAuthorityPackages is the guard's whole forbidden set: the
// operating-system package, the process-execution package, the low-level
// system-call package, and the deprecated legacy I/O package — broader than
// the charter's literal three verbs (environment, filesystem, process) by
// design; that breadth is intended belt-and-braces (S-APC-034).
var forbiddenAmbientAuthorityPackages = []forbiddenAmbientAuthorityPackage{
	{
		importPath:       "os",
		defaultLocalName: "os",
		rule:             "R-APC-008: no environment variable read, no filesystem path touched",
	},
	{
		importPath:       "os/exec",
		defaultLocalName: "exec",
		rule:             "R-APC-008: no process spawned",
	},
	{
		importPath:       "syscall",
		defaultLocalName: "syscall",
		rule:             "R-APC-008: no low-level system call — belt-and-braces beyond the charter's three named verbs",
	},
	{
		importPath:       "io/ioutil",
		defaultLocalName: "ioutil",
		rule:             "R-APC-008: the deprecated legacy I/O package — belt-and-braces beyond the charter's three named verbs",
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

// isAdapterSourceFile reports whether name is a file this guard scans: a
// ".go" file that is not a "_test.go" file. This is the entire file-
// selection rule (R-APC-011) — it excludes every test file uniformly,
// including this guard's own source, which is not special-cased
// (S-APC-045).
func isAdapterSourceFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// scanNonTestSourcesForAmbientAuthority scans every file isAdapterSourceFile
// admits in dir and returns one message-ready violation string per forbidden
// call site or forbidden dot-import found. Each message names its file,
// line and forbidden package (S-APC-035).
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

// scanFileForAmbientAuthority resolves file's imports to local identifiers
// (alias-aware, via file.Imports) and flags every call site whose selector
// resolves to a forbidden package, plus every dot-import of a forbidden
// package in its own right, independent of any call site.
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
			// Blank import: inert. It binds no callable identifier, and an
			// unused non-blank import fails compilation, so a blank import
			// of a forbidden package can never reach a call site.
			continue

		case imp.Name != nil && imp.Name.Name == ".":
			// A dot-import is a violation in its own right (R-APC-010 item
			// 4, S-APC-041): resolving a bare identifier back to this
			// import would need full type information (the recorded
			// limitation above), so the import itself is flagged
			// regardless of whether a call to it is found.
			pos := fset.Position(imp.Pos())
			violations = append(violations, fmt.Sprintf(
				"%s:%d: dot-import of forbidden package %q (%s)",
				fileName, pos.Line, importPath, entry.rule))
			continue

		case imp.Name != nil:
			// A named alias (S-APC-039): resolved by its local identifier,
			// not by the import path, so renaming the import does not
			// bypass the scan.
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

// TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources is AI-25.2's
// guard test (R-APC-008, S-APC-033). It is green from birth over clean,
// landed sources — a guard that only ever passes is unfalsifiable, so its
// falsifiability comes from the four bite proofs recorded in this
// milestone's evidence log (R-APC-010), each staged as a deliberate scratch
// violation in real adapter source, run red, recorded, then dropped — not
// from an ordinary TDD red phase against this test itself.
func TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources(t *testing.T) {
	t.Parallel()

	violations := scanNonTestSourcesForAmbientAuthority(t, ".")
	for _, violation := range violations {
		t.Errorf("ambient authority: %s", violation)
	}
}

// TestAmbientAuthority_ForbiddenSetIsPackageScopedDenyByDefault covers
// S-APC-034: the forbidden set names the operating-system, process-
// execution, system-call and legacy-I/O packages by package — never by a
// curated function name — and nothing else.
func TestAmbientAuthority_ForbiddenSetIsPackageScopedDenyByDefault(t *testing.T) {
	t.Parallel()

	want := []string{"io/ioutil", "os", "os/exec", "syscall"}
	got := make([]string, 0, len(forbiddenAmbientAuthorityPackages))
	for _, entry := range forbiddenAmbientAuthorityPackages {
		got = append(got, entry.importPath)
	}
	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("forbidden ambient-authority package set = %v, want %v (S-APC-034)", got, want)
	}
}

// TestAmbientAuthority_IsAdapterSourceFile covers S-APC-045's file-selection
// half: the scan's exclusion rule is a uniform suffix rule, not a list of
// named exceptions, so this guard's own source is excluded by the same rule
// that excludes every other test file — never a special case naming it.
func TestAmbientAuthority_IsAdapterSourceFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want bool
	}{
		{"a production source is included", "client.go", true},
		{"an adapter test source is excluded", "client_test.go", false},
		{"the guard's own source is excluded, not special-cased", "ambient_authority_test.go", false},
		{"a non-Go file is excluded", "README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isAdapterSourceFile(tt.file); got != tt.want {
				t.Errorf("isAdapterSourceFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

// TestAmbientAuthority_TestSourcesStayGreenEvenWithForbiddenCalls covers
// S-APC-044: a fixture "_test.go" file that itself calls a forbidden
// package — a stronger case than this package's real timeout_test.go,
// which only imports the permitted net/http/httptest — is still scanned
// clean, because test sources are excluded by construction, not because no
// test happens to need a forbidden package.
func TestAmbientAuthority_TestSourcesStayGreenEvenWithForbiddenCalls(t *testing.T) {
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
		t.Errorf("scan reported violations from a _test.go fixture, want none (S-APC-044): %v", violations)
	}
}
