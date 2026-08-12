// Shared test helpers for package agent_test, used across AG-04's scenario
// files. Kept in one place so a helper added for one phase's scenarios is
// available to a later phase's without duplication.
package agent_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// exportedPackageAgentNames enumerates every exported top-level
// declaration (type, function, method, const, var) across package agent's
// own non-test source files, by parsing their syntax directly — never by
// importing the package or reading `go/doc` output, mirroring
// doc_contract_guard_test.go's own raw-file-read posture.
//
// `go test` sets the working directory to the package under test's own
// directory, shared between package agent and this external package
// agent_test, so "." resolves it without an assumed absolute path.
func exportedPackageAgentNames(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\") error = %v, want nil", err)
	}

	fset := token.NewFileSet()
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v, want nil", name, err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							names = append(names, s.Name.Name)
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								names = append(names, n.Name)
							}
						}
					}
				}
			case *ast.FuncDecl:
				if d.Name.IsExported() {
					names = append(names, d.Name.Name)
				}
			}
		}
	}
	return names
}

// containsFold reports whether s contains substr, ASCII-case-insensitively
// — enough for the plain identifier-shaped names this scan compares.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
