// Tests for the AI-14 descriptor skeleton — groundwork for R-AEE-014.
//
// The package under test is imported by its module path from the external
// test package ai_test, matching every other Layer 1 test file's convention.
package ai_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Groundwork for R-AEE-014 — the descriptor's zero value is "no block role,
// no cardinality restriction, not terminal", the least restrictive reading,
// so a kind registered without stating one does not accidentally join block
// ordering, get capped at one, or end the stream.
func TestEventDescriptor_ZeroValue_RoundTrips(t *testing.T) {
	t.Parallel()

	t.Run("BlockRoleNone and CardinalityAny are the zero values", func(t *testing.T) {
		t.Parallel()

		var role ai.BlockRole
		if role != ai.BlockRoleNone {
			t.Errorf("the zero BlockRole = %v, want BlockRoleNone", role)
		}

		var cardinality ai.Cardinality
		if cardinality != ai.CardinalityAny {
			t.Errorf("the zero Cardinality = %v, want CardinalityAny", cardinality)
		}
	})

	t.Run("the zero EventDescriptor round-trips", func(t *testing.T) {
		t.Parallel()

		var d ai.EventDescriptor
		if d.Role != ai.BlockRoleNone {
			t.Errorf("the zero EventDescriptor.Role = %v, want BlockRoleNone", d.Role)
		}
		if d.Cardinality != ai.CardinalityAny {
			t.Errorf("the zero EventDescriptor.Cardinality = %v, want CardinalityAny", d.Cardinality)
		}
		if d.Terminal {
			t.Error("the zero EventDescriptor.Terminal = true, want false")
		}

		// An explicit literal built from the zero-valued components must equal
		// the zero value: the three fields are the whole of the type's
		// comparable state, so nothing hides an unstated fourth thing.
		explicit := ai.EventDescriptor{Role: ai.BlockRoleNone, Cardinality: ai.CardinalityAny, Terminal: false}
		if explicit != d {
			t.Errorf("an explicit zero-valued EventDescriptor %+v != the zero value %+v", explicit, d)
		}
	})
}

// R-AEE-020, S-AEE-064 — none of the exported identifiers accumulates, joins
// or reconstructs a block's deltas: doc 0001 §4.3 invariant 1 reserves
// accumulation for the consumer.
func TestExportedSurface_NoIdentifier_AccumulatesJoinsOrReconstructsDeltas(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading .: %v", err)
	}

	forbidden := []string{"accumulat", "reconstruct", "rebuild", "transcript", "reduce"}
	fset := token.NewFileSet()
	found := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			for _, ident := range exportedTopLevelIdents(decl) {
				found++
				lower := strings.ToLower(ident)
				for _, bad := range forbidden {
					if strings.Contains(lower, bad) {
						t.Errorf("%s exports %q, whose name suggests accumulating/joining/reconstructing "+
							"a block's deltas — R-AEE-020 forbids that", name, ident)
					}
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no exported top-level identifier found across the package; the guard would pass vacuously")
	}
}

// exportedTopLevelIdents returns every exported top-level identifier a
// declaration introduces: a func's (or method's) name, or a var/const/type
// spec's names.
func exportedTopLevelIdents(decl ast.Decl) []string {
	var names []string
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if ast.IsExported(d.Name.Name) {
			names = append(names, d.Name.Name)
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if ast.IsExported(n.Name) {
						names = append(names, n.Name)
					}
				}
			case *ast.TypeSpec:
				if ast.IsExported(s.Name.Name) {
					names = append(names, s.Name.Name)
				}
			}
		}
	}
	return names
}

// R-AEE-020, S-AEE-065 — the documented steps for registering a delta kind
// state fragment-only and forbid a snapshot.
//
// Reads file.Comments[0] rather than file.Doc: revive's package-comments
// rule (make lint) requires exactly one file per package — doc.go — to
// carry the comment attached to the package clause, so this file's own
// header comment is deliberately separated from "package ai" by a blank
// line and is no longer file.Doc, though it is still file.Comments[0].
func TestEventDescriptorGoFile_Doc_StatesDeltaIsFragmentOnlyAndForbidsSnapshot(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "event_descriptor.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing event_descriptor.go: %v", err)
	}
	if len(file.Comments) == 0 {
		t.Fatal("event_descriptor.go carries no leading comment; the guard would pass vacuously")
	}
	doc := strings.Join(strings.Fields(file.Comments[0].Text()), " ")

	if !strings.Contains(doc, "fragment") {
		t.Errorf("event_descriptor.go's doc does not state \"fragment\":\n%s", doc)
	}
	if !strings.Contains(doc, "snapshot") {
		t.Errorf("event_descriptor.go's doc does not state \"snapshot\" (forbidding one):\n%s", doc)
	}
}
