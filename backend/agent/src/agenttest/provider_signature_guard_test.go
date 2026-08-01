// AI-20.4 — the signature guard: ModelProvider's carrier and import
// allowlist, pinned as a build-time failure rather than a review
// convention (R-AMP-014).
//
// This guard resolves src/ai/provider.go relative to ITS OWN source file
// via runtime.Caller(0) (ADR 0005 § D2, Guard C): src/agenttest must remain
// a direct sibling of src/ai, or this resolution breaks — loudly, per
// R-AMP-015, never by skipping or passing vacuously.
//
// It parses with go/parser and asserts syntactically (no go/types, no
// subprocess, no build context): the interface exists with exactly one
// method, Stream's parameter and result types are the neutral ones, and
// the file's imports are a subset of {"context"}. "Exactly one method"
// also mechanizes half of R-AMP-021's pin — the other half is
// provider_test.go's compile proof that a Stream-only stub satisfies
// ModelProvider from outside package ai.
package agenttest_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// modelProviderAllowedImports is R-AMP-002's whole allowlist: "context" and
// nothing else. A vendor SDK type, or a stand-in for one, cannot reach the
// declaration without also reaching an import outside this set.
var modelProviderAllowedImports = map[string]bool{
	"context": true,
}

// providerGoPath resolves src/ai/provider.go relative to THIS test file's
// own source location (ADR 0005 Guard C, R-AMP-015).
func providerGoPath(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve provider.go")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "ai", "provider.go")
}

// AI-20.4 (R-AMP-014…016) — the signature guard itself. Bite-proof evidence
// for the two mutations this test must fail against lives in tasks.md
// (Phase 4), pasted verbatim from manual apply/run/revert cycles against
// this exact test.
func TestModelProviderInterface_SignatureGuard(t *testing.T) {
	t.Parallel()

	path := providerGoPath(t)
	file, err := resolveAndParseGoFile(path)
	if err != nil {
		// R-AMP-015: fails loudly, never skips, never passes vacuously.
		// resolveAndParseGoFile's own error already names the path.
		t.Fatal(err)
	}

	assertImportAllowlist(t, file)

	iface := findInterface(file, "ModelProvider")
	if iface == nil {
		t.Fatalf("provider.go at %q declares no interface named ModelProvider", path)
	}
	assertExactlyOneStreamMethod(t, iface)
}

// resolveAndParseGoFile resolves and parses one Go source file, returning a
// single, path-naming error on either failure (R-AMP-015: "fails loudly,
// never skips, never passes vacuously"). It is a plain function rather than
// a *testing.T-bound helper so the two failure branches below are directly
// testable without engineering a real missing or corrupt provider.go.
func resolveAndParseGoFile(path string) (*ast.File, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("target is not resolvable at %q — src/agenttest must stay a direct "+
			"sibling of src/ai (ADR 0005 § D2, Guard C): %w", path, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("target at %q is not parseable: %w", path, err)
	}
	return file, nil
}

// AI-20.4 (R-AMP-015, S-AMP-040/041) — an unresolvable target is reported
// as an error naming the path, never silently skipped.
func TestResolveAndParseGoFile_UnresolvableTarget_ReportsAnErrorNamingThePath(t *testing.T) {
	t.Parallel()

	const missing = "/nonexistent/path/does-not-exist/provider.go"
	_, err := resolveAndParseGoFile(missing)
	if err == nil {
		t.Fatal("resolveAndParseGoFile returned no error for a path that does not exist, want a loud failure (R-AMP-015)")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q does not name the unresolvable path %q", err.Error(), missing)
	}
}

// AI-20.4 (R-AMP-015, S-AMP-040/041) — an unparseable target is reported as
// an error naming the path, never silently skipped.
func TestResolveAndParseGoFile_UnparseableTarget_ReportsAnErrorNamingThePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "not_valid_go.go")
	if err := os.WriteFile(path, []byte("this is not valid Go source {{{"), 0o600); err != nil {
		t.Fatalf("os.WriteFile returned %v, want no failure", err)
	}

	_, err := resolveAndParseGoFile(path)
	if err == nil {
		t.Fatal("resolveAndParseGoFile returned no error for unparseable source, want a loud failure (R-AMP-015)")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the unparseable path %q", err.Error(), path)
	}
}

// assertImportAllowlist is R-AMP-002/R-AMP-014's mechanical form: every
// import in provider.go must be "context", or this fails by name.
func assertImportAllowlist(t *testing.T, file *ast.File) {
	t.Helper()

	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			importPath = imp.Path.Value
		}
		if !modelProviderAllowedImports[importPath] {
			t.Errorf("provider.go imports %q; only \"context\" is allowed on ModelProvider's declaring "+
				"file (R-AMP-002)", importPath)
		}
	}
}

// findInterface returns the top-level interface type declaration named
// name, or nil if provider.go declares none by that name.
func findInterface(file *ast.File, name string) *ast.InterfaceType {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}
			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				return nil
			}
			return iface
		}
	}
	return nil
}

// assertExactlyOneStreamMethod is R-AMP-001's syntactic form, and half of
// R-AMP-021's pin: ModelProvider declares exactly one method, named Stream,
// with the neutral context.Context/Request parameters and the
// <-chan Event/error results.
func assertExactlyOneStreamMethod(t *testing.T, iface *ast.InterfaceType) {
	t.Helper()

	if iface.Methods == nil || len(iface.Methods.List) != 1 {
		var names []string
		if iface.Methods != nil {
			for _, m := range iface.Methods.List {
				for _, n := range m.Names {
					names = append(names, n.Name)
				}
			}
		}
		t.Fatalf("ModelProvider declares %d method(s) %v, want exactly 1 (R-AMP-001; mechanizes half "+
			"of R-AMP-021's widening pin)", len(iface.Methods.List), names)
	}

	method := iface.Methods.List[0]
	if len(method.Names) != 1 || method.Names[0].Name != "Stream" {
		t.Fatalf("ModelProvider's one method is not named Stream: %v", method.Names)
	}

	funcType, ok := method.Type.(*ast.FuncType)
	if !ok {
		t.Fatalf("ModelProvider.Stream's declared type is %T, want a function type", method.Type)
	}

	assertParams(t, funcType)
	assertResults(t, funcType)
}

func assertParams(t *testing.T, funcType *ast.FuncType) {
	t.Helper()

	var params []ast.Expr
	if funcType.Params != nil {
		params = flattenFieldTypes(funcType.Params.List)
	}
	if len(params) != 2 {
		t.Fatalf("Stream declares %d parameter(s), want exactly 2 (ctx, req)", len(params))
	}
	if !isSelector(params[0], "context", "Context") {
		t.Errorf("Stream's first parameter is %s, want context.Context", exprString(params[0]))
	}
	if !isIdent(params[1], "Request") {
		t.Errorf("Stream's second parameter is %s, want Request (Layer 1's own normalized type, not a vendor type)", exprString(params[1]))
	}
}

func assertResults(t *testing.T, funcType *ast.FuncType) {
	t.Helper()

	var results []ast.Expr
	if funcType.Results != nil {
		results = flattenFieldTypes(funcType.Results.List)
	}
	if len(results) != 2 {
		t.Fatalf("Stream declares %d result(s), want exactly 2 (event stream, error)", len(results))
	}
	if !isRecvChanOf(results[0], "Event") {
		t.Errorf("Stream's first result is %s, want <-chan Event (V-STR-04's receive-only carrier)", exprString(results[0]))
	}
	if !isIdent(results[1], "error") {
		t.Errorf("Stream's second result is %s, want error", exprString(results[1]))
	}
}

// flattenFieldTypes expands a field list into one type expression per
// parameter or result: a field naming several identifiers of the same type
// (e.g. "a, b int") expands to one entry per name, and an unnamed field
// (the ordinary shape for a result) is one entry.
func flattenFieldTypes(fields []*ast.Field) []ast.Expr {
	var out []ast.Expr
	for _, f := range fields {
		if len(f.Names) == 0 {
			out = append(out, f.Type)
			continue
		}
		for range f.Names {
			out = append(out, f.Type)
		}
	}
	return out
}

func isSelector(expr ast.Expr, pkgName, sel string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == pkgName && selector.Sel.Name == sel
}

func isIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// isRecvChanOf reports whether expr is a receive-only channel of the named
// element type. Go's go/ast field for a channel's element type is Value,
// not Elt (Elt names ArrayType/MapType's element field instead).
func isRecvChanOf(expr ast.Expr, elemName string) bool {
	chanType, ok := expr.(*ast.ChanType)
	if !ok {
		return false
	}
	if chanType.Dir != ast.RECV {
		return false
	}
	return isIdent(chanType.Value, elemName)
}

// exprString renders an AST type expression for a failure message. It is a
// diagnostic aid only — this guard's assertions above never depend on its
// output.
func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return "<selector>." + e.Sel.Name
	case *ast.ChanType:
		switch e.Dir {
		case ast.RECV:
			return "<-chan " + exprString(e.Value)
		case ast.SEND:
			return "chan<- " + exprString(e.Value)
		default:
			return "chan " + exprString(e.Value)
		}
	default:
		return fmt.Sprintf("%T", expr)
	}
}
