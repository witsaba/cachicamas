// Shared test helpers for package agent_test, used across AG-04's scenario
// files. Kept in one place so a helper added for one phase's scenarios is
// available to a later phase's without duplication.
package agent_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
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

// funcDeclSignature returns the *ast.FuncType of the top-level function
// named funcName declared in file (a package-agent source file resolved
// relative to the test binary's own working directory, mirroring
// exportedPackageAgentNames), or nil if either does not exist. Used to
// prove a declaration's shape — parameter types, receiver, exportedness —
// directly from source, never by importing the package.
func funcDeclSignature(t *testing.T, file, funcName string) *ast.FuncType {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile(%q) error = %v, want nil", file, err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != funcName {
			continue
		}
		return fn.Type
	}
	return nil
}

// packageFileDoc returns file's own leading comment group, collapsed to
// single-spaced text — sequence_test.go's own
// TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule idiom, restated
// here for reuse across AG-04's documentation-reading assertions.
func packageFileDoc(t *testing.T, file string) string {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile(%q) error = %v, want nil", file, err)
	}
	if len(f.Comments) == 0 {
		t.Fatalf("%s carries no leading comment; the guard would pass vacuously", file)
	}
	return strings.Join(strings.Fields(f.Comments[0].Text()), " ")
}

// requireViolationPosition fails t unless violation is an *ai.Violation
// whose position's last step carries an index equal to wantIndex —
// R-AEV-006's "MUST name the offending position" obligation (post-verify
// correction, W1/W2), pinned exactly rather than merely checked for
// presence: a report that omits the index, or names the wrong one (for
// example the offending event's sequence value instead of its slice
// index), fails here.
func requireViolationPosition(t *testing.T, violation error, wantIndex int) {
	t.Helper()

	var v *ai.Violation
	if !errors.As(violation, &v) {
		t.Fatalf("violation = %T, want errors.As to reach *ai.Violation so its position is inspectable", violation)
	}
	path := v.Path()
	if len(path) == 0 {
		t.Fatal("violation carries an empty position; R-AEV-006 requires the report to name the offending position")
	}
	last := path[len(path)-1]
	idx, indexed := last.Index()
	if !indexed {
		t.Fatalf("violation's last position step %q carries no index; want the offending event's slice index", last.Name())
	}
	if idx != wantIndex {
		t.Errorf("violation position index = %d, want %d (the offending event's own 0-based index within the checked slice, not its sequence value)", idx, wantIndex)
	}
}

// allFileComments returns every comment group in file, concatenated and
// collapsed to single-spaced text — used where the text being asserted on
// may live in a specific declaration's own doc comment rather than the
// file's leading package-doc comment (packageFileDoc's narrower scope).
func allFileComments(t *testing.T, file string) string {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile(%q) error = %v, want nil", file, err)
	}
	if len(f.Comments) == 0 {
		t.Fatalf("%s carries no comments; the guard would pass vacuously", file)
	}
	var b strings.Builder
	for _, group := range f.Comments {
		b.WriteString(group.Text())
		b.WriteByte(' ')
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
