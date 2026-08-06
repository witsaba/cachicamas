// AI-32 — the charter boundary (R-AEM-017, S-AEM-064…066). This milestone
// exports no backoff policy, attempt counter or failover identifier
// (retryability is a flag; AI-35 owns acting on it), does not widen
// ai.FailureCategory past nine members, and keeps backend/agent/go.mod at
// zero require lines. Internal package, matching design.md's Testing
// Strategy table for slice 1.
package openaicompat

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// forbiddenCharterSubstrings names the concepts R-AEM-017 forbids this
// package from exporting. Matched case-insensitively against every
// exported top-level identifier's name.
var forbiddenCharterSubstrings = []string{
	"backoff",
	"attempt",
	"failover",
}

// checkCharterIdent appends file:ident to flagged when ident is exported
// and its lowercased form contains one of forbiddenCharterSubstrings.
func checkCharterIdent(flagged *[]string, file, ident string) {
	if !ast.IsExported(ident) {
		return
	}
	lower := strings.ToLower(ident)
	for _, forbidden := range forbiddenCharterSubstrings {
		if strings.Contains(lower, forbidden) {
			*flagged = append(*flagged, file+":"+ident+" (contains \""+forbidden+"\")")
		}
	}
}

// TestCharterBoundary_NoBackoffAttemptOrFailoverExported proves S-AEM-064:
// this package's own non-test sources, parsed with go/ast, export no
// identifier naming a backoff, attempt-count or failover concept.
//
// Top-level declarations only — function/method names, type names, and
// var/const names — mirroring reasoning_refusal_test.go's own
// TestPolicy_NoNewSentinelsExported scan shape (var/const), extended here
// to funcs and types. A struct field is not a top-level identifier in this
// sense, the same scope reasoning_refusal_test.go's own sentinel scan
// already draws.
func TestCharterBoundary_NoBackoffAttemptOrFailoverExported(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\"): %v", err)
	}

	fset := token.NewFileSet()
	var flagged []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%s): %v", name, parseErr)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				checkCharterIdent(&flagged, name, d.Name.Name)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						checkCharterIdent(&flagged, name, s.Name.Name)
					case *ast.ValueSpec:
						for _, ident := range s.Names {
							checkCharterIdent(&flagged, name, ident.Name)
						}
					}
				}
			}
		}
	}

	if len(flagged) != 0 {
		t.Fatalf("exported identifiers naming a forbidden charter concept (R-AEM-017): %v", flagged)
	}
}

// TestCharterBoundary_FailureCategoriesHasNineMembers proves S-AEM-065:
// this milestone does not widen ai.FailureCategory — the vocabulary stays
// at nine members.
func TestCharterBoundary_FailureCategoriesHasNineMembers(t *testing.T) {
	if got := len(ai.FailureCategories()); got != 9 {
		t.Fatalf("len(ai.FailureCategories()) = %d, want 9 (R-AEM-017: the vocabulary stays closed)", got)
	}
}

// TestCharterBoundary_GoModHasZeroRequireLines proves S-AEM-066:
// backend/agent/go.mod carries zero lines beginning "require" — this
// milestone adds no dependency. Three directories up from this package
// (src/ai/openaicompat -> src/ai -> src -> agent).
func TestCharterBoundary_GoModHasZeroRequireLines(t *testing.T) {
	path := filepath.Join("..", "..", "..", "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "require") {
			count++
		}
	}
	if count != 0 {
		t.Fatalf("%s has %d line(s) beginning \"require\", want 0 (R-AEM-017)", path, count)
	}
}
