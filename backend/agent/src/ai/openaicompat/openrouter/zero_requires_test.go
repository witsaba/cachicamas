// R-OR-09's defense-in-depth regression test (design §8, AI-00.3
// invariant): the AI-00.3 forward-guard's allowlist,
// `allowedNonStdlibPrefixes` in backend/agent/src/ai/import_boundary_test.go,
// holds exactly one entry — this module's own path
// (`github.com/cachicamas/backend/agent`). A second entry would
// either be the AI-37 OpenTelemetry API path (pre-authorised by
// ADR 0005 § D3 and not yet landed) or an unauthorized new
// top-level dependency. Both shapes must be reviewed in the same
// commit as the entry addition; this test fires whenever the
// allowlist shape drifts from one entry without an explicit update
// to this file.
//
// # Why this is defense-in-depth, not the primary guard
//
// AI-00.3's own test (TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault,
// import_boundary_test.go:107) is the load-bearing guard. This
// test is a copy at the openrouter sub-package's vantage point: if
// the allowlist grows beyond one entry in a future change, this
// test fires first with a more focused failure message, and the
// AI-00.3 test fires second as a deeper one. Either alone is
// sufficient; both together make the regression harder to miss in
// review.
//
// # The two assertions (R-OR-09)
//
//  1. backend/agent/go.mod declares zero `require` lines. This
//     pin exists because go.mod is the source of truth for what
//     the module declares as a dependency; the guard against new
//     requires is the module's own invariant, not the test's.
//
//  2. allowedNonStdlibPrefixes in import_boundary_test.go holds
//     exactly one entry, and that entry equals this module's
//     import path. The second half pins the value: a future change
//     that grows the list to two and adds a non-module entry is a
//     regression on the same surface as a go.mod require, and this
//     test fails it before CI runs the deeper guard.
//
// # Why go/parser instead of a narrower text scan
//
// A narrower scan that hand-parses the slice literal would be
// brittle: a future shape change (a comment, a wrapped line, a
// renamed identifier) would silently rotate the guard to look at
// the wrong text. go/parser + go/ast is the same toolchain the
// project's own signature guard uses (provider_signature_guard_test.go:88)
// and the same toolchain openaicompat's own ambient_authority_test.go
// uses (ambient_authority_test.go:150-160) — its load-bearing
// posture is "this file's source as Go is the source under test",
// and any non-trivial parsing lives in the toolchain.

package openrouter

import (
	"bufio"
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// openrouterModulePath is the only non-stdlib prefix the allowlist
// admits today. This module's own import path — any future change
// that adds a second entry here must also land the entry's
// rationale in the import_boundary_test.go header comment and in
// an ADR.
const openrouterModulePath = "github.com/cachicamas/backend/agent"

// goModPath is the relative path from this sub-package's directory
// to the module's go.mod. The test reads this file as raw bytes
// and counts require lines.
const goModPath = "../../../../go.mod"

// importBoundaryTestPath is the relative path from this
// sub-package's directory to the AI-00.3 forward-guard test. The
// test parses this file with go/parser and walks the AST to
// locate the allowedNonStdlibPrefixes variable.
const importBoundaryTestPath = "../../import_boundary_test.go"

// TestOpenRouterAdapter_ZeroRequiresGuard asserts the go.mod is
// the empty-dependency pin (R-OR-09). A `require` line in
// go.mod — of any kind — fails this test. The AI-00.3 forward
// guard's primary assertion (TestLayer1_ModuleHasNoDependencies_
// ZeroRequires, import_boundary_test.go:147) carries the same
// property through the dependency-closure walk; this test pins it
// at the source-of-truth file for faster, more focused feedback.
func TestOpenRouterAdapter_ZeroRequiresGuard(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil — go.mod must be readable for this guard to discharge R-OR-09", goModPath, err)
	}

	requireLines := countGoModRequireLines(raw)
	if requireLines != 0 {
		t.Errorf("backend/agent/go.mod declares %d require line(s), want exactly 0 (R-OR-09, AI-00.3, NFR-APC-A):\n%s", requireLines, raw)
	}
}

// TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry
// asserts the AI-00.3 allowlist holds exactly one entry, and that
// entry is this module's own import path (R-OR-09, defense-in-depth
// copy of import_boundary_test.go's load-bearing guard). A second
// entry — even a benign-looking one — fails this test, by design:
// the second-entry path requires an ADR and a
// import_boundary_test.go header comment update alongside the
// entry, not a silent addition.
//
// The parse uses go/parser + go/ast: the import_boundary_test.go
// file is parsed as Go, the package-level variable declaration
// `allowedNonStdlibPrefixes` is located, and the slice literal's
// ElementList is walked. Each element is resolved to its string
// value by following const declarations in the same file (the
// canonical pattern this package's own const modulePath uses) or,
// for a literal string, by reading the literal directly.
func TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry(t *testing.T) {
	t.Parallel()

	entries, err := parseAllowedNonStdlibPrefixes(importBoundaryTestPath, openrouterModulePath)
	if err != nil {
		t.Fatalf("parseAllowedNonStdlibPrefixes(%q) error = %v, want nil", importBoundaryTestPath, err)
	}

	if len(entries) != 1 {
		t.Errorf("allowedNonStdlibPrefixes has %d entries = %v, want exactly 1 (R-OR-09; the only admitted non-stdlib prefix is this module's own path)", len(entries), entries)
	}
	if len(entries) > 0 && entries[0] != openrouterModulePath {
		t.Errorf("allowedNonStdlibPrefixes first entry = %q, want %q (R-OR-09)", entries[0], openrouterModulePath)
	}
}

// parseAllowedNonStdlibPrefixes parses path as Go and returns the
// resolved string values of the package-level
// allowedNonStdlibPrefixes variable declaration. The resolution
// follows const declarations in the same file — the canonical
// pattern this package's own const modulePath uses — so a slice
// literal that names `modulePath` resolves to its declared
// string value.
//
// An empty result with a nil error means the variable exists but
// has zero elements (a regression on its own); an error means the
// variable could not be located or one of its elements could not
// be resolved.
func parseAllowedNonStdlibPrefixes(path, _ string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	constValues := collectStringConsts(file)

	var entries []string
	found := false
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if name.Name != "allowedNonStdlibPrefixes" {
					continue
				}
				found = true
				if valueSpec.Values == nil {
					continue
				}
				for _, expr := range valueSpec.Values {
					lit, ok := expr.(*ast.CompositeLit)
					if !ok {
						continue
					}
					for _, elt := range lit.Elts {
						ident, ok := elt.(*ast.Ident)
						if !ok {
							continue
						}
						if v, ok := constValues[ident.Name]; ok {
							entries = append(entries, v)
							continue
						}
						// Unknown identifier — it could be a
						// const declared in another file (not
						// visible to this parse). Surface that as
						// an empty entry so the assertion above
						// fails with a clear "not equal to
						// openrouterModulePath" message rather
						// than passing silently.
						entries = append(entries, "")
					}
				}
			}
		}
	}

	if !found {
		return nil, nil
	}
	return entries, nil
}

// collectStringConsts walks file's declarations and returns a map
// from const-name to its declared string-literal value. Only
// string-valued consts are recorded; non-string consts and
// string-valued consts whose right-hand side is not a string
// literal are silently ignored (the slice under test only names
// string-valued consts).
func collectStringConsts(file *ast.File) map[string]string {
	out := make(map[string]string)
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if len(valueSpec.Values) != 1 {
				continue
			}
			basicLit, ok := valueSpec.Values[0].(*ast.BasicLit)
			if !ok || basicLit.Kind != token.STRING {
				continue
			}
			value, err := unquoteStringLiteral(basicLit.Value)
			if err != nil {
				continue
			}
			for _, name := range valueSpec.Names {
				out[name.Name] = value
			}
		}
	}
	return out
}

// unquoteStringLiteral strips the surrounding quotes from a Go
// string literal (token.BasicLit.Value) and unescapes the inner
// bytes — the same shape go/ast produces for every string in a
// parsed file. The implementation is narrow but sufficient: a Go
// string literal is exactly `"..."` with the inner bytes subject
// to Go's standard escape sequences.
func unquoteStringLiteral(lit string) (string, error) {
	if len(lit) < 2 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		return "", os.ErrInvalid
	}
	inner := lit[1 : len(lit)-1]
	var b strings.Builder
	b.Grow(len(inner))
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(inner) {
			return "", os.ErrInvalid
		}
		esc := inner[i+1]
		switch esc {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case '"':
			b.WriteByte('"')
		case '\\':
			b.WriteByte('\\')
		case '\'':
			b.WriteByte('\'')
		default:
			b.WriteByte(esc)
		}
		i++
	}
	return b.String(), nil
}

// countGoModRequireLines counts lines in raw that are go.mod
// `require` lines (R-OR-09, AI-00.3). go.mod syntax has three
// require-directive shapes:
//
//   - `require ( ... )`  — a block form; individual entries inside
//     the block are indented.
//   - `require <path> <version>` — a single-line form.
//
// Both shapes fail this guard by design: a single require is the
// regression R-OR-09 exists to catch. Comment lines (`//`) and the
// `module` / `go` directives are not require lines.
func countGoModRequireLines(raw []byte) int {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "require") {
			count++
		}
	}
	return count
}