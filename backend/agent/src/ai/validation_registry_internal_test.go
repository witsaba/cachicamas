// AI-04 *(guard)* — the rule-class registry is enforced, not mirrored.
//
// This guard makes two properties of the closed rule-class set mechanical
// failures rather than review conventions:
//
//  1. Every sentinel declared in validation.go is listed in ruleClasses.
//  2. The mirror in validation_test.go is the registry, exactly and in order.
//
// # Why it exists
//
// AI-08 appended ErrDuplicate. The append was correct, but it revealed that
// nothing checked it. ruleClasses is unexported on purpose — a consumer never
// iterates the set, and exporting it to satisfy a test would widen the surface
// AI-40 freezes — so the external test package ai_test carries a hand-written
// copy of the list. Nothing compared the two. A class appended to the package
// registry stayed outside the "exactly one class matches" pin until somebody
// remembered to edit the copy, and a sentinel declared in validation.go but
// never added to ruleClasses was caught by nothing at all: it renders as "value
// violates an unregistered rule" and is pinned by no test.
//
// doc 0002 defines a guard leaf as a mechanical check that must keep failing
// forever when violated, closed by showing it bite. A hand-written mirror cannot
// fail, so it was never that. Both directions below are recorded biting against
// deliberate scratch violations in tasks.md.
//
// # Why it is internal, and why it reads source
//
// The file is package ai rather than ai_test so it can read ruleClasses
// directly, which is what lets the registry stay unexported. It scans source
// with go/ast rather than trusting a list, in the idiom of AI-00's
// import_boundary_test.go and AI-13's vocabulary pin: a list obtained from the
// thing under test would agree with itself no matter what was added to it.
//
// The AST is bound to the running slice at the end of the first test, by
// comparing each ruleClasses element's own text against the string literal its
// declaration was given. Without that step the guard would prove two source
// files agree with each other and nothing about the registry the package
// actually iterates in ruleText.
//
// The sentinel scan is deliberately wider than the rule-class var block: every
// top-level `errors.New` in validation.go counts. validation.go declares rule
// classes and nothing else, so a sentinel of another kind landing here must
// either join the registry or move to the file that owns it — and this guard is
// where that choice gets made rather than skipped.

package ai

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The two sources this guard reads, and the name both registries are bound to.
//
// The paths are relative because `go test` runs with the working directory set
// to the package under test. The mirror carries the same name as the registry
// on purpose: a mirror renamed out from under this guard fails it rather than
// silently escaping it.
const (
	validationSource     = "validation.go"
	validationTestSource = "validation_test.go"
	registryVarName      = "ruleClasses"
)

// sentinelDeclaration is one rule-class sentinel as its source declares it: the
// identifier it binds and the fixed text errors.New was given.
type sentinelDeclaration struct {
	name    string
	message string
}

// registryElement is one element of a registry composite literal: the sentinel
// it names, and how the source spells it. The two differ in the mirror, where
// every element is qualified — `ai.ErrEmpty` names `ErrEmpty`.
type registryElement struct {
	name     string
	rendered string
}

// TestRuleClasses_EverySentinelDeclaredInTheSource_IsInTheRegistry closes the
// declared-but-unregistered hole, in both directions, and then binds what it
// read to the slice the package runs.
func TestRuleClasses_EverySentinelDeclaredInTheSource_IsInTheRegistry(t *testing.T) {
	t.Parallel()

	source := parseGoSource(t, validationSource)

	declared := declaredSentinels(t, source, validationSource)
	if len(declared) == 0 {
		t.Fatalf("%s declares no `errors.New` sentinel, so this guard would pass vacuously. "+
			"Check that the rule classes are still declared here.", validationSource)
	}

	registered := registryElements(t, source, validationSource)
	if len(registered) == 0 {
		t.Fatalf("%s lists no element in %s, so this guard would pass vacuously.",
			validationSource, registryVarName)
	}

	isRegistered := make(map[string]bool, len(registered))
	for _, element := range registered {
		isRegistered[element.name] = true
	}
	messageOf := make(map[string]string, len(declared))
	for _, sentinel := range declared {
		messageOf[sentinel.name] = sentinel.message
	}

	for _, sentinel := range declared {
		if !isRegistered[sentinel.name] {
			t.Errorf("%s declares the sentinel %s, which %s does not list.\n"+
				"  rule: the rule-class set is closed, and a class is appended in the pull request "+
				"that needs it (design.md § 3.1) — declaring the sentinel is half of that append.\n"+
				"  A sentinel outside %s renders as \"value violates an unregistered rule\", and no "+
				"test of this package covers it.",
				validationSource, sentinel.name, registryVarName, registryVarName)
		}
	}

	for _, element := range registered {
		if _, isSentinel := messageOf[element.name]; !isSentinel {
			t.Errorf("%s lists %s, which %s does not declare as an `errors.New` sentinel.\n"+
				"  rule: the registry enumerates rule classes, and a rule class is a sentinel. "+
				"An element that is anything else cannot carry the class text ruleText renders.",
				registryVarName, element.rendered, validationSource)
		}
	}

	// Bind the source to the running slice. Everything above is a claim about
	// two lists in one file; this is what makes it a claim about the registry
	// ruleText iterates.
	if len(ruleClasses) != len(registered) {
		t.Fatalf("the running %s holds %d classes, but %s lists %d — this guard is not reading "+
			"the slice the package uses.", registryVarName, len(ruleClasses), validationSource, len(registered))
	}
	for i, element := range registered {
		want, isSentinel := messageOf[element.name]
		if !isSentinel {
			continue // Already reported above.
		}
		if got := ruleClasses[i].Error(); got != want {
			t.Errorf("%s[%d] carries the text %q, but %s was declared with %q.\n"+
				"  The source order and the running order have diverged, so every assertion this "+
				"guard makes about the source says nothing about the registry.",
				registryVarName, i, got, element.name, want)
		}
	}
}

// TestRuleClasses_TheExternalTestMirror_MatchesTheRegistryExactly closes the
// drift hole AI-08 exposed: the mirror in ai_test must be the registry, with
// the same members in the same order.
//
// The order is asserted and not only the membership. ruleClasses is a slice and
// not a map because nothing in this package may let an unordered iteration
// decide anything; a mirror free to reorder would be a second, disagreeing
// answer to "which class is first".
func TestRuleClasses_TheExternalTestMirror_MatchesTheRegistryExactly(t *testing.T) {
	t.Parallel()

	registered := registryElements(t, parseGoSource(t, validationSource), validationSource)
	mirrored := registryElements(t, parseGoSource(t, validationTestSource), validationTestSource)

	registryNames := elementNames(registered)
	mirrorNames := elementNames(mirrored)

	for _, name := range registryNames {
		if !slices.Contains(mirrorNames, name) {
			t.Errorf("%s registers the rule class %s, which the mirror in %s omits.\n"+
				"  rule: the mirror is what the \"a failure matches exactly one rule class\" pin "+
				"iterates, so a class missing from it is a class that pin does not cover.\n"+
				"  Append it to %s in %s, at the same position it holds here.",
				validationSource, name, validationTestSource, registryVarName, validationTestSource)
		}
	}

	for _, name := range mirrorNames {
		if !slices.Contains(registryNames, name) {
			t.Errorf("the mirror in %s lists %s, which %s does not register.\n"+
				"  rule: the mirror names the closed set, and a name outside it is either a "+
				"sentinel that was never appended to %s or one that was removed from it.",
				validationTestSource, name, validationSource, registryVarName)
		}
	}

	if !slices.Equal(registryNames, mirrorNames) {
		t.Errorf("the mirror is not the registry.\n"+
			"  %s: %s\n"+
			"  %s: %s\n"+
			"  rule: same members, same order. %s is ordered on purpose, and a mirror free to "+
			"reorder it is a second answer to which class comes first.",
			validationSource, strings.Join(registryNames, ", "),
			validationTestSource, strings.Join(mirrorNames, ", "),
			registryVarName)
	}
}

// parseGoSource parses one source file of this package, comments included.
func parseGoSource(t *testing.T, fileName string) *ast.File {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), fileName, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing %s: %v", fileName, err)
	}
	return file
}

// declaredSentinels reports every top-level `var X = errors.New("…")` of a file,
// in source order.
//
// It matches the call by its written form — the identifier `errors`, the
// selector `New`, one string literal — rather than by resolving types, because
// go/types is not needed to read a declaration this package wrote and a guard
// that needed type information would be a guard nobody could follow.
func declaredSentinels(t *testing.T, file *ast.File, fileName string) []sentinelDeclaration {
	t.Helper()

	var declared []sentinelDeclaration
	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || general.Tok != token.VAR {
			continue
		}
		for _, spec := range general.Specs {
			value, isValue := spec.(*ast.ValueSpec)
			if !isValue || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			message, isSentinel := errorsNewText(t, fileName, value.Values[0])
			if !isSentinel {
				continue
			}
			declared = append(declared, sentinelDeclaration{name: value.Names[0].Name, message: message})
		}
	}
	return declared
}

// errorsNewText reports the fixed text of an `errors.New("…")` call, and whether
// the expression is one.
func errorsNewText(t *testing.T, fileName string, expr ast.Expr) (string, bool) {
	t.Helper()

	call, isCall := expr.(*ast.CallExpr)
	if !isCall || len(call.Args) != 1 {
		return "", false
	}
	selector, isSelector := call.Fun.(*ast.SelectorExpr)
	if !isSelector || selector.Sel.Name != "New" {
		return "", false
	}
	pkg, isIdent := selector.X.(*ast.Ident)
	if !isIdent || pkg.Name != "errors" {
		return "", false
	}
	literal, isLiteral := call.Args[0].(*ast.BasicLit)
	if !isLiteral || literal.Kind != token.STRING {
		t.Fatalf("%s: a sentinel is declared from something other than a string literal, which "+
			"this guard cannot read and V-FAIL-13 does not allow — a class's text is fixed.", fileName)
	}
	text, err := strconv.Unquote(literal.Value)
	if err != nil {
		t.Fatalf("%s: unquoting the text of a sentinel declaration: %v", fileName, err)
	}
	return text, true
}

// registryElements reports the elements of the file's top-level registry
// composite literal, in source order.
//
// A file with no such declaration is a failure and not an empty result: this
// guard reads two files by name, and "the thing I was pointed at is gone" must
// not be indistinguishable from "it agrees".
func registryElements(t *testing.T, file *ast.File, fileName string) []registryElement {
	t.Helper()

	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || general.Tok != token.VAR {
			continue
		}
		for _, spec := range general.Specs {
			value, isValue := spec.(*ast.ValueSpec)
			if !isValue || len(value.Names) != 1 || value.Names[0].Name != registryVarName {
				continue
			}
			if len(value.Values) != 1 {
				t.Fatalf("%s: %s is declared without a single composite literal, which this guard "+
					"reads it from.", fileName, registryVarName)
			}
			composite, isComposite := value.Values[0].(*ast.CompositeLit)
			if !isComposite {
				t.Fatalf("%s: %s is not a composite literal, which this guard reads it from.",
					fileName, registryVarName)
			}

			elements := make([]registryElement, 0, len(composite.Elts))
			for _, element := range composite.Elts {
				elements = append(elements, sentinelReference(t, fileName, element))
			}
			return elements
		}
	}

	t.Fatalf("%s declares no top-level var named %s. Either it was renamed — in which case rename "+
		"it back or update this guard in the same commit — or the registry this guard exists to "+
		"enforce is gone.", fileName, registryVarName)
	return nil
}

// sentinelReference reads one registry element as a reference to a sentinel,
// qualified or not.
func sentinelReference(t *testing.T, fileName string, expr ast.Expr) registryElement {
	t.Helper()

	switch reference := expr.(type) {
	case *ast.Ident:
		return registryElement{name: reference.Name, rendered: reference.Name}
	case *ast.SelectorExpr:
		if qualifier, isIdent := reference.X.(*ast.Ident); isIdent {
			return registryElement{
				name:     reference.Sel.Name,
				rendered: qualifier.Name + "." + reference.Sel.Name,
			}
		}
	}

	t.Fatalf("%s: an element of %s is not a plain reference to a sentinel. The registry enumerates "+
		"the closed set by name; an expression computing a member would make the set open.",
		fileName, registryVarName)
	return registryElement{}
}

// elementNames projects registry elements onto the sentinel names they refer to,
// which is the form the two registries are comparable in.
func elementNames(elements []registryElement) []string {
	names := make([]string, 0, len(elements))
	for _, element := range elements {
		names = append(names, element.name)
	}
	return names
}
