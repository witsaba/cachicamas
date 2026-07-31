// AI-06.4 — kind registration is exhaustive.
//
// Adding a content-part kind is a five-step procedure (decision.md § 8), and a
// five-step procedure handed to a later milestone is only safe if a machine
// fails when one step is missing. Go checks no sum type's exhaustiveness, which
// is precisely why this is a guard and not a comment: after AI-07 and AI-09 this
// package carries four kinds, and the failure mode of the fourth is a kind that
// can be constructed but never read, or read but never validated.
//
// # What the guard enumerates, and why it is not the name table
//
// It enumerates the **declared constant space, read from the source**, not
// PartKinds() and not partKindNames. Both of those are the package's own answer
// to the question being asked, so a guard built on either compares the package
// against itself:
//
//   - a guard over partKindNames sees exactly the kinds that have a table entry,
//     so a constant declared without one — the first way the procedure is got
//     wrong — is invisible to the assertion that exists to catch it;
//   - a guard over PartKinds() sees exactly the kinds inside partKindFirst …
//     partKindEnd, so a constant declared without moving partKindEnd — the
//     second way — is invisible too.
//
// So the constant space is parsed out of the package's own source with go/ast,
// and the three answers are then compared to each other: what is declared, what
// PartKinds() enumerates, and what partKindNames registers. Any two of them
// disagreeing is a failure, whichever direction the disagreement runs. This is
// AI-13's objection to a pin that reads the package's own accessor, applied
// here: the only input the guard trusts is the text a human wrote.
//
// # Where the witnesses live
//
// Here, not in production. Production's registry is one slice indexed by the
// constant — AI-05's roleNames shape, for AI-04's stated reason that a Layer 1
// registry is never a map — and putting four closures per kind into it so a test
// could call them would be production carrying test scaffolding. Keyed by the
// declared constant, the witness table gets the identical bite at no production
// cost: a kind declared without a witness fails the guard's first assertion.
//
// # This file is internal
//
// It needs partKindNames, needs to assemble a part that skipped its rules, and
// needs ruleClasses. None of the three is expressible from ai_test, and
// exporting any of them for a test's benefit would widen the surface AI-06
// exists to narrow.

package ai

import (
	"errors"
	"go/ast"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// partKindWitness is what a kind must supply to prove its three legs.
//
// A leg left nil is the missing step the guard exists to catch, and each field
// names the step of decision.md § 8's procedure it stands for.
type partKindWitness struct {
	// constantName is the identifier as written in the source. It is what the
	// AST scan matches against, so a kind cannot be given a witness under a
	// name it does not have.
	constantName string

	// registeredName is the entry this kind must have in partKindNames. It is
	// stated here rather than read from the table so that the table is checked
	// rather than trusted.
	registeredName string

	// constructValid is step 3's happy path: the exported constructor, called
	// with an input its rules accept.
	constructValid func() (Part, error)

	// constructInvalid is step 3's leg — a constructor *with rules*. It calls
	// the same constructor with an input the rules must reject.
	constructInvalid func() (Part, error)

	// read is step 4: the exported accessor for this kind, adapted to one
	// signature. It reports whether the part is of this kind.
	read func(Part) (any, bool)

	// skipRules is step 5's leg — a validation path. It assembles a part of
	// this kind directly from its payload, bypassing the constructor, holding a
	// value the payload's own rules reject. From outside the package this is
	// unwritable; from inside it is the mistake a future variant author makes.
	skipRules func() Part
}

// partKindWitnesses is the guard's table, keyed by the declared constant.
//
// One entry per kind, and the entry is what a kind's author writes in the same
// commit as the five steps. Everything below is a call to the exported surface
// except skipRules, which is deliberately the one thing only this package can
// write — assembling a part from a payload without going through the door.
var partKindWitnesses = map[PartKind]partKindWitness{
	PartKindText: {
		constantName:   "PartKindText",
		registeredName: "text",

		constructValid: func() (Part, error) { return NewText("model-visible text") },

		// Whitespace-only text is rejected by rule 1 of the payload's own
		// rules: V-REQ-08 says model-visible, and a part carrying only
		// separators carries nothing a model reads.
		constructInvalid: func() (Part, error) { return NewText("   \t\n ") },

		read: func(p Part) (any, bool) { return p.Text() },

		skipRules: func() Part { return Part{payload: textPayload{text: "   "}} },
	},
}

// TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath
// is the exhaustiveness guard.
//
// It fails when a kind is declared without a witness, when a witness is missing
// a leg, when the three enumerations disagree, or when a leg is wired but does
// not do what its step promises.
func TestPartKindRegistration_EveryDeclaredKind_HasConstructorAccessorAndValidationPath(t *testing.T) {
	t.Parallel()

	declared := declaredPartKindConstants(t)
	if len(declared) == 0 {
		t.Fatal("the source scan found no exported PartKind constant; the guard would pass vacuously")
	}

	t.Run("every declared constant has a witness, and every witness a constant", func(t *testing.T) {
		t.Parallel()

		byName := make(map[string]PartKind, len(partKindWitnesses))
		for kind, witness := range partKindWitnesses {
			byName[witness.constantName] = kind
		}

		for _, name := range declared {
			if _, ok := byName[name]; !ok {
				t.Errorf("%s is declared in the source but has no entry in partKindWitnesses.\n"+
					"  Adding a kind is the five-step procedure in decision.md § 8, and this guard is what fails when a step is missing.\n"+
					"  Add a witness with all three legs: a constructor with rules, a payload accessor, and a validation path.", name)
			}
		}
		for name := range byName {
			if !containsString(declared, name) {
				t.Errorf("partKindWitnesses carries a witness for %s, which is not declared as an exported PartKind constant in the source", name)
			}
		}
	})

	t.Run("the declared constants, PartKinds() and partKindNames agree", func(t *testing.T) {
		t.Parallel()

		if got, want := len(PartKinds()), len(declared); got != want {
			t.Errorf("PartKinds() enumerates %d kinds but the source declares %d (%v).\n"+
				"  A constant declared without moving partKindEnd past it is invisible to every assertion over PartKinds().", got, want, declared)
		}

		for _, kind := range PartKinds() {
			witness, ok := partKindWitnesses[kind]
			if !ok {
				t.Errorf("PartKinds() enumerates %d, which has no witness", uint8(kind))
				continue
			}
			if got := partKindName(kind); got != witness.registeredName {
				t.Errorf("partKindNames[%s] = %q, want %q.\n"+
					"  A kind declared without a table entry is not a member of the vocabulary and every part carrying it is invalid.",
					witness.constantName, got, witness.registeredName)
			}
		}

		registered := 0
		for i, name := range partKindNames {
			if name == "" {
				continue
			}
			registered++
			if !containsKind(PartKinds(), PartKind(i)) {
				t.Errorf("partKindNames registers %q at index %d, which PartKinds() does not enumerate", name, i)
			}
		}
		if registered != len(declared) {
			t.Errorf("partKindNames holds %d entries but the source declares %d constants (%v)", registered, len(declared), declared)
		}
	})

	for kind, witness := range partKindWitnesses {
		t.Run(witness.constantName, func(t *testing.T) {
			t.Parallel()

			if witness.constructValid == nil || witness.constructInvalid == nil {
				t.Fatalf("%s has no leg 1: step 3 of the procedure is an exported constructor whose rules are the payload's", witness.constantName)
			}
			if witness.read == nil {
				t.Fatalf("%s has no leg 2: step 4 of the procedure is an exported accessor func (p Part) %s() (T, bool)",
					witness.constantName, strings.TrimPrefix(witness.constantName, "PartKind"))
			}
			if witness.skipRules == nil {
				t.Fatalf("%s has no leg 3: step 5 of the procedure is a validation path the message boundary runs", witness.constantName)
			}

			t.Run("leg 1 — a constructor, with rules", func(t *testing.T) {
				t.Parallel()

				part, err := witness.constructValid()
				if err != nil {
					t.Fatalf("the valid witness for %s returned %v, want no failure", witness.constantName, err)
				}
				if part.Kind() != kind {
					t.Errorf("the valid witness for %s built a part of kind %v, want %v", witness.constantName, part.Kind(), kind)
				}

				rejected, err := witness.constructInvalid()
				if err == nil {
					t.Fatalf("the constructor for %s accepted an input its rules must reject.\n"+
						"  A constructor without rules is a door with no lock, and the message boundary would be the only thing checking.", witness.constantName)
				}
				var violation *Violation
				if !errors.As(err, &violation) {
					t.Fatalf("the constructor for %s failed with %T, want a *Violation", witness.constantName, err)
				}
				if !carriesRegisteredRuleClass(err) {
					t.Errorf("the failure for %s names no registered rule class: %v.\n"+
						"  Every caller-contract failure in Layer 1 reports through AI-04's sentinels.", witness.constantName, err)
				}
				if rejected != (Part{}) {
					t.Errorf("a rejected construction of %s returned a non-zero part, want the zero Part", witness.constantName)
				}
			})

			t.Run("leg 2 — a payload accessor", func(t *testing.T) {
				t.Parallel()

				part, err := witness.constructValid()
				if err != nil {
					t.Fatalf("the valid witness for %s returned %v, want no failure", witness.constantName, err)
				}
				if _, ok := witness.read(part); !ok {
					t.Errorf("the accessor for %s reported no payload on a part of its own kind", witness.constantName)
				}

				// The accessor must also answer no. A part of another kind, and
				// the part that was never constructed, both go through it.
				others := []struct {
					what string
					part Part
				}{{"the zero Part", Part{}}}
				for otherKind, other := range partKindWitnesses {
					if otherKind == kind || other.constructValid == nil {
						continue
					}
					built, err := other.constructValid()
					if err != nil {
						t.Fatalf("the valid witness for %s returned %v, want no failure", other.constantName, err)
					}
					others = append(others, struct {
						what string
						part Part
					}{"a part of kind " + other.constantName, built})
				}

				for _, other := range others {
					if _, ok := witness.read(other.part); ok {
						t.Errorf("the accessor for %s reported a payload on %s.\n"+
							"  The second result distinguishes \"not this kind\" from \"this kind, empty\"; an accessor that always says yes erases the distinction.",
							witness.constantName, other.what)
					}
				}
			})

			t.Run("leg 3 — a validation path", func(t *testing.T) {
				t.Parallel()

				smuggled := witness.skipRules()
				if smuggled.Kind() != kind {
					t.Fatalf("the skip-rules witness for %s built a part of kind %v, want %v — it must bypass the constructor, not the kind",
						witness.constantName, smuggled.Kind(), kind)
				}

				if violation := smuggled.validate(nil); violation == nil {
					t.Errorf("a %s part that skipped its rules validates cleanly.\n"+
						"  The constructor and the message boundary must run one rule set from two entry points (decision.md § 7.2); "+
						"a constructor that checks and a boundary that does not is defect C1 from the other direction.", witness.constantName)
				}

				if _, err := NewMessage(RoleUser, smuggled); err == nil {
					t.Errorf("NewMessage accepted a %s part that skipped its rules, want a rejection", witness.constantName)
				} else if !carriesRegisteredRuleClass(err) {
					t.Errorf("NewMessage rejected a %s part with %v, which names no registered rule class", witness.constantName, err)
				}

				// And the valid one still passes, so the leg is not satisfied by
				// a boundary that rejects everything.
				valid, err := witness.constructValid()
				if err != nil {
					t.Fatalf("the valid witness for %s returned %v, want no failure", witness.constantName, err)
				}
				if _, err := NewMessage(RoleUser, valid); err != nil {
					t.Errorf("NewMessage rejected a properly constructed %s part with %v, want no failure", witness.constantName, err)
				}
			})
		})
	}
}

// TestPartKindDocumentation_MatchesTheRegistrationTable checks the fifth step's
// other half: the documented kind list against the table.
//
// It is an AST scan and not a reading, because a documented vocabulary that
// drifts from the code is worse than an undocumented one — a consumer trusts it.
// The list lives in the GoDoc of PartKind under a fixed heading, and both
// directions are asserted: a documented kind the table lacks, and a tabulated
// kind the documentation lacks.
func TestPartKindDocumentation_MatchesTheRegistrationTable(t *testing.T) {
	t.Parallel()

	documented := documentedPartKindNames(t)
	if len(documented) == 0 {
		t.Fatal("the PartKind GoDoc lists no registered kind under its heading; the guard would pass vacuously")
	}

	tabulated := make([]string, 0, len(partKindNames))
	for _, name := range partKindNames {
		if name != "" {
			tabulated = append(tabulated, name)
		}
	}

	for _, name := range tabulated {
		if !containsString(documented, name) {
			t.Errorf("partKindNames registers %q, which the PartKind GoDoc does not list.\n"+
				"  documented: %v\n  tabulated:  %v\n"+
				"  Step 5 of decision.md § 8 is one step: the table entry and the documented line land together.", name, documented, tabulated)
		}
	}
	for _, name := range documented {
		if !containsString(tabulated, name) {
			t.Errorf("the PartKind GoDoc lists %q, which partKindNames does not register.\n"+
				"  documented: %v\n  tabulated:  %v\n"+
				"  A documented kind with no table entry is not a member of the vocabulary: every part carrying it would be invalid.", name, documented, tabulated)
		}
	}
}

// declaredPartKindConstants returns the exported PartKind constants as they are
// written in this package's own source, in declaration order.
//
// It parses rather than reflects, because reflection can only see the values the
// package computed and this guard exists to catch a constant the package's own
// enumeration does not see. Test files are excluded: a scratch kind declared in
// a test must be declared in production source to count, which is what keeps the
// bite proof honest.
//
// Within a const block a spec with neither a type nor a value repeats the
// previous spec's, which is how an iota run works — so the type carries forward
// and a spec that states its own value without a type ends the run.
func declaredPartKindConstants(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	fset := token.NewFileSet()
	out := make([]string, 0, 8)

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}

			var carried ast.Expr
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				switch {
				case value.Type != nil:
					carried = value.Type
				case len(value.Values) > 0:
					// An explicit value without a type ends the iota run's
					// carried type. partKindFirst and partKindEnd are this
					// shape, and they are not members.
					carried = nil
				}
				if !isPartKindType(carried) {
					continue
				}
				for _, ident := range value.Names {
					if ident.IsExported() {
						out = append(out, ident.Name)
					}
				}
			}
		}
	}
	return out
}

// isPartKindType reports whether an expression names the PartKind type.
func isPartKindType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "PartKind"
}

// documentedPartKindNames returns the first word of every list item under the
// "Registered kinds" heading in PartKind's GoDoc.
//
// The doc comment is parsed with go/doc/comment — the same parser that renders
// the package documentation — rather than by scanning for "  - ", so the guard
// checks what a reader sees rather than a formatting convention.
func documentedPartKindNames(t *testing.T) []string {
	t.Helper()

	const heading = "Registered kinds"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "content_part.go", nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing content_part.go: %v", err)
	}

	doc := partKindDoc(file)
	if doc == nil {
		t.Fatal("the PartKind type declaration carries no doc comment")
	}

	var parser comment.Parser
	parsed := parser.Parse(doc.Text())

	out := make([]string, 0, 8)
	inSection := false
	for _, block := range parsed.Content {
		switch typed := block.(type) {
		case *comment.Heading:
			inSection = commentPlainText(typed.Text) == heading
		case *comment.List:
			if !inSection {
				continue
			}
			for _, item := range typed.Items {
				for _, itemBlock := range item.Content {
					paragraph, ok := itemBlock.(*comment.Paragraph)
					if !ok {
						continue
					}
					fields := strings.Fields(commentPlainText(paragraph.Text))
					if len(fields) > 0 {
						out = append(out, fields[0])
					}
				}
			}
		}
	}
	return out
}

// partKindDoc finds the doc comment of the PartKind type declaration.
func partKindDoc(file *ast.File) *ast.CommentGroup {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			typed, ok := spec.(*ast.TypeSpec)
			if !ok || typed.Name.Name != "PartKind" {
				continue
			}
			if typed.Doc != nil {
				return typed.Doc
			}
			return gen.Doc
		}
	}
	return nil
}

// commentPlainText flattens the text of a doc comment element.
func commentPlainText(text []comment.Text) string {
	var out strings.Builder
	for _, part := range text {
		switch typed := part.(type) {
		case comment.Plain:
			out.WriteString(string(typed))
		case comment.Italic:
			out.WriteString(string(typed))
		case *comment.Link:
			out.WriteString(commentPlainText(typed.Text))
		case *comment.DocLink:
			out.WriteString(commentPlainText(typed.Text))
		}
	}
	return out.String()
}

// carriesRegisteredRuleClass reports whether the failure names one of AI-04's
// registered rule classes. A Layer 1 caller-contract failure that names none is
// a sentinel someone invented in passing.
func carriesRegisteredRuleClass(err error) bool {
	for _, class := range ruleClasses {
		if errors.Is(err, class) {
			return true
		}
	}
	return false
}

func containsString(haystack []string, want string) bool {
	for _, got := range haystack {
		if got == want {
			return true
		}
	}
	return false
}

func containsKind(haystack []PartKind, want PartKind) bool {
	for _, got := range haystack {
		if got == want {
			return true
		}
	}
	return false
}
