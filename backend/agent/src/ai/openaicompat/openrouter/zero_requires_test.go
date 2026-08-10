// R-OR-09's defense-in-depth regression test (design §8, AI-00.3
// invariant): the AI-00.3 forward-guard's allowlist,
// `allowedNonStdlibPrefixes` in backend/agent/src/ai/import_boundary_test.go,
// holds exactly the entries AI-37 (ADR 0005 § D3, D-6) authorises: this
// module's own path plus the five OpenTelemetry tracing-API prefixes.
// Before AI-37 this test pinned exactly one entry; AI-37 is the
// documented, ADR-gated exception, landed in the same commit as this
// file's update — any FURTHER entry, beyond the ones AI-37 adds, is
// either a later ADR-gated addition (which must update this file
// alongside it) or an unauthorized new top-level dependency.
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
//  1. backend/agent/go.mod declares exactly wantGoModRequireLines
//     required modules (3, as of AI-37: the block form's two direct
//     OpenTelemetry entries, plus the single-line indirect xxhash
//     entry). This pin exists because go.mod is the source of truth
//     for what the module declares as a dependency; the guard against
//     an unauthorized require is the module's own invariant, not the
//     test's.
//
//  2. allowedNonStdlibPrefixes in import_boundary_test.go holds
//     exactly wantAllowedNonStdlibPrefixes' entries, in order. A
//     future change that grows the list beyond what AI-37 landed,
//     without updating this file in the same commit, is a
//     regression on the same surface as an unreviewed go.mod
//     require, and this test fails it before CI runs the deeper
//     guard.
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
	"slices"
	"strings"
	"testing"
)

// openrouterModulePath is this module's own import path — always the
// allowlist's first entry.
const openrouterModulePath = "github.com/cachicamas/backend/agent"

// goModPath is the relative path from this sub-package's directory
// to the module's go.mod. The test reads this file as raw bytes
// and counts required modules.
const goModPath = "../../../../go.mod"

// wantGoModRequireLines is the exact required-module count AI-37 (ADR
// 0005 § D3, D-6) authorises: the two direct OpenTelemetry entries
// inside the "require ( ... )" block, plus the single-line indirect
// xxhash require — 3 total, counted by countGoModRequireLines's own
// module-entry walk (Judgment Day remediation: this counts entries, not
// "require"-prefixed lines, so an addition inside the block is no
// longer invisible). Any further required module is either a later
// ADR-gated addition — which must update this constant in the same
// commit — or an unauthorized dependency.
const wantGoModRequireLines = 3

// wantAllowedNonStdlibPrefixes is the exact, ordered
// allowedNonStdlibPrefixes AI-37 authorises: this module's own path, plus
// the five OpenTelemetry tracing-API prefixes import_boundary_test.go
// adds, each citing ADR 0005 § D3.
var wantAllowedNonStdlibPrefixes = []string{
	openrouterModulePath,
	"go.opentelemetry.io/otel/trace",
	"go.opentelemetry.io/otel/attribute",
	"go.opentelemetry.io/otel/codes",
	"go.opentelemetry.io/otel/semconv",
	"github.com/cespare/xxhash/v2",
}

// importBoundaryTestPath is the relative path from this
// sub-package's directory to the AI-00.3 forward-guard test. The
// test parses this file with go/parser and walks the AST to
// locate the allowedNonStdlibPrefixes variable.
const importBoundaryTestPath = "../../import_boundary_test.go"

// TestOpenRouterAdapter_RequireLinesMatchAI37Authorization asserts
// go.mod carries exactly wantGoModRequireLines `require` lines — no
// more (R-OR-09). Before AI-37 this test pinned zero; AI-37 (ADR 0005
// § D3) is the one, ADR-gated milestone permitted to change that. The
// AI-00.3 forward guard's primary assertion
// (TestLayer1_DependencySet_ExactRequiresAndClosure,
// import_boundary_test.go) carries the same property through the
// dependency-closure walk; this test pins it at the source-of-truth
// file for faster, more focused feedback.
func TestOpenRouterAdapter_RequireLinesMatchAI37Authorization(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil — go.mod must be readable for this guard to discharge R-OR-09", goModPath, err)
	}

	requireLines := countGoModRequireLines(raw)
	if requireLines != wantGoModRequireLines {
		t.Errorf("backend/agent/go.mod declares %d require line(s), want exactly %d (R-OR-09, AI-37 D-6, NFR-APC-A):\n%s", requireLines, wantGoModRequireLines, raw)
	}
}

// TestOpenRouterAdapter_AllowedNonStdlibPrefixesMatchesAI37Authorization
// asserts the AI-00.3 allowlist holds exactly wantAllowedNonStdlibPrefixes'
// entries, in order (R-OR-09, defense-in-depth copy of
// import_boundary_test.go's load-bearing guard). Before AI-37 this test
// pinned exactly one entry (this module's own path); AI-37 is the
// documented, ADR-gated exception. Any entry beyond what AI-37 landed —
// even a benign-looking one — fails this test, by design: a further
// entry requires an ADR and an import_boundary_test.go header comment
// update alongside it, not a silent addition.
//
// The parse uses go/parser + go/ast: the import_boundary_test.go
// file is parsed as Go, the package-level variable declaration
// `allowedNonStdlibPrefixes` is located, and the slice literal's
// ElementList is walked. Each element is resolved to its string
// value by following const declarations in the same file (the
// canonical pattern this package's own const modulePath uses) or,
// for a literal string, by reading the literal directly.
func TestOpenRouterAdapter_AllowedNonStdlibPrefixesMatchesAI37Authorization(t *testing.T) {
	t.Parallel()

	entries, err := parseAllowedNonStdlibPrefixes(importBoundaryTestPath, openrouterModulePath)
	if err != nil {
		t.Fatalf("parseAllowedNonStdlibPrefixes(%q) error = %v, want nil", importBoundaryTestPath, err)
	}

	if !slices.Equal(entries, wantAllowedNonStdlibPrefixes) {
		t.Errorf("allowedNonStdlibPrefixes = %v, want %v, in order (R-OR-09, AI-37 D-6)", entries, wantAllowedNonStdlibPrefixes)
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
						switch e := elt.(type) {
						case *ast.Ident:
							if v, ok := constValues[e.Name]; ok {
								entries = append(entries, v)
								continue
							}
							// Unknown identifier — it could be a
							// const declared in another file (not
							// visible to this parse). Surface that
							// as an empty entry so the assertion
							// above fails with a clear mismatch
							// message rather than passing silently.
							entries = append(entries, "")
						case *ast.BasicLit:
							// AI-37's entries are literal strings
							// (import_boundary_test.go's own table,
							// each carrying its own ADR-citing
							// comment) — the literal-string half
							// this function's own doc comment
							// already promised, and the only half
							// that was actually implemented before
							// AI-37 exercised it.
							if e.Kind == token.STRING {
								if v, unquoteErr := unquoteStringLiteral(e.Value); unquoteErr == nil {
									entries = append(entries, v)
									continue
								}
							}
							entries = append(entries, "")
						default:
							entries = append(entries, "")
						}
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

// countGoModRequireLines counts the MODULE ENTRIES a go.mod's require
// directives declare (R-OR-09, AI-00.3, Judgment Day remediation): every
// non-empty, non-comment line inside a `require ( ... )` block, plus
// every single-line `require <path> <version>` directive. go.mod syntax
// has two require-directive shapes:
//
//   - `require ( ... )`  — a block form; individual entries inside
//     the block are indented, one module per line.
//   - `require <path> <version>` — a single-line form.
//
// This is deliberately a count of MODULE ENTRIES, not a count of how
// many times the literal word "require" opens a directive. An earlier
// version of this function counted only lines whose trimmed text
// started with "require", so a dependency appended INSIDE an existing
// require( ... ) block introduced no new "require"-prefixed line and
// left the count unchanged -- an unauthorized addition inside the block
// was invisible to this guard (Judgment Day round 1, finding 6;
// TestCountGoModRequireLines_DetectsInBlockAddition, below, is the
// falsifier). Comment lines (`//`), blank lines and the `module` / `go`
// directives are not counted.
func countGoModRequireLines(raw []byte) int {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	count := 0
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			count++
			continue
		}
		if line == "require (" {
			inBlock = true
			continue
		}
		if strings.HasPrefix(line, "require ") {
			count++
			continue
		}
	}
	return count
}

// ai37AuthorizedGoMod is a byte-identical shape to the AI-37-authorized
// backend/agent/go.mod (block form, two direct requires, one single-line
// indirect require) -- the baseline this test's mutation is compared
// against. Kept in-memory so this proof never touches the real go.mod
// file (Judgment Day remediation: no overlay is needed for a pure
// []byte -> int function).
var ai37AuthorizedGoMod = []byte(`module github.com/cachicamas/backend/agent

go 1.26.3

require (
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
)

require github.com/cespare/xxhash/v2 v2.3.0 // indirect
`)

// ai37GoModWithInBlockAddition is the exact same go.mod, with one
// additional, unauthorized module appended INSIDE the existing
// require( ... ) block -- the precise shape Judgment Day round 1 named:
// a dependency added inside the block introduces no new line starting
// with the literal word "require".
var ai37GoModWithInBlockAddition = []byte(`module github.com/cachicamas/backend/agent

go 1.26.3

require (
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	github.com/evil/unauthorized v1.0.0
)

require github.com/cespare/xxhash/v2 v2.3.0 // indirect
`)

// TestCountGoModRequireLines_DetectsInBlockAddition is the falsifier for
// R-OR-09's own guard function: a dependency appended INSIDE the
// existing require( ... ) block MUST change what this function reports,
// so the guard actually bites an addition there instead of passing it
// through silently. Before the Judgment Day fix, both go.mod shapes
// above counted identically (2: the "require (" line and the
// single-line indirect require), because neither counting pass ever
// looks at the block's own interior lines.
func TestCountGoModRequireLines_DetectsInBlockAddition(t *testing.T) {
	t.Parallel()

	authorized := countGoModRequireLines(ai37AuthorizedGoMod)
	withAddition := countGoModRequireLines(ai37GoModWithInBlockAddition)

	if withAddition == authorized {
		t.Errorf("countGoModRequireLines() = %d for both the authorized go.mod and one with an unauthorized module appended INSIDE the require( ... ) block, want different counts -- an in-block addition must not be invisible to this guard (Judgment Day round 1, finding 6)", authorized)
	}
	if withAddition != authorized+1 {
		t.Errorf("countGoModRequireLines(with one extra in-block module) = %d, want exactly %d (authorized %d + 1)", withAddition, authorized+1, authorized)
	}
}
