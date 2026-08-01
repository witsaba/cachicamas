// AI-14.3 — the no-process-global-sequence-state guard (R-AEE-011,
// R-AEE-012, R-AEE-013, design.md D8/D9).
//
// # What C3 was, and why this is a guard rather than a review note
//
// The retired design kept doc 0001's stream sequence in a package-level
// atomic counter: one process-wide `atomic.Uint64`, incremented for every
// event across every stream. Register C3 recorded the resulting defect —
// only the first stream in a process began at 1 — and sequence.go's own
// comment restates the fix in one sentence: it is not a smaller counter; it
// is putting the counter where the stream is. A [Stamper] belongs to one
// stream and holds a plain uint64, no atomic, no mutex, no package scope.
//
// A review note is not a mechanism: nothing stops a later milestone from
// reaching for a familiar package-level `atomic.Uint64` "just for this one
// case" and reintroducing exactly the process-global counter this milestone
// exists to retire. This file is R-AEE-011's answer — a mechanical AST scan
// of every non-test source file this package ships, failing the build on any
// package-level state shaped like the retired counter, or any function that
// resets one, unless R-AEE-012's allowlist admits it by name and reasoned
// rationale.
package ai_test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// packageDir is package ai's own directory. `go test` sets the working
// directory to the package under test, so "." names it without hardcoding an
// absolute path that would not survive a checkout elsewhere.
const packageDir = "."

// R-AEE-011, S-AEE-031 — the landed package passes: its one qualifying var,
// message.go's lastMessageID, is admitted by R-AEE-012's allowlist.
func TestSequenceGuard_LandedPackage_Passes(t *testing.T) {
	t.Parallel()

	violations, err := scanSequenceStateGuard(packageDir, sequenceStateAllowlist)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", packageDir, err)
	}
	if len(violations) != 0 {
		t.Errorf("scanSequenceStateGuard(%q) = %v, want no violations", packageDir, violations)
	}
}

// R-AEE-011, S-AEE-032 — a scratch package-level integer var fails, naming
// the offending file and identifier.
func TestSequenceGuard_ScratchPackageLevelInteger_Fails(t *testing.T) {
	t.Parallel()

	dir := writeScratchPackage(t, map[string]string{
		"scratch.go": "package ai\n\nvar scratchSeq uint64\n",
	})

	violations, err := scanSequenceStateGuard(dir, nil)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", dir, err)
	}
	assertNamesViolation(t, violations, "scratch.go", "scratchSeq")
}

// R-AEE-011, S-AEE-033 — a scratch var of an unexported struct whose only
// field is a qualifying integer still fails: "recursively contains" reaches
// struct fields at any depth, not only a bare declaration.
func TestSequenceGuard_ScratchStructWrappedInteger_Fails(t *testing.T) {
	t.Parallel()

	dir := writeScratchPackage(t, map[string]string{
		"scratch.go": "package ai\n\ntype scratchState struct{ n uint64 }\n\nvar scratchCounter scratchState\n",
	})

	violations, err := scanSequenceStateGuard(dir, nil)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", dir, err)
	}
	assertNamesViolation(t, violations, "scratch.go", "scratchCounter")
}

// R-AEE-011, S-AEE-034 — a scratch package-level reset function for
// sequence-shaped state fails, independently of the var-declaration
// violation above.
func TestSequenceGuard_ScratchResetFunction_Fails(t *testing.T) {
	t.Parallel()

	dir := writeScratchPackage(t, map[string]string{
		"scratch.go": "package ai\n\nvar scratchSeq uint64\n\nfunc resetScratchSeq() { scratchSeq = 0 }\n",
	})

	violations, err := scanSequenceStateGuard(dir, nil)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", dir, err)
	}
	assertNamesViolation(t, violations, "scratch.go", "scratchSeq")

	found := false
	for _, v := range violations {
		if strings.Contains(v.reason, "reset") {
			found = true
		}
	}
	if !found {
		t.Errorf("violations = %v, want at least one whose reason names the reset", violations)
	}
}

// R-AEE-011, S-AEE-035 — a package-level counter declared in a _test.go file
// does not fail the guard: its subject is the shipped, non-test contract.
func TestSequenceGuard_TestFileCounter_Passes(t *testing.T) {
	t.Parallel()

	dir := writeScratchPackage(t, map[string]string{
		"scratch.go":      "package ai\n",
		"scratch_test.go": "package ai\n\nvar scratchSeq uint64\n",
	})

	violations, err := scanSequenceStateGuard(dir, nil)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", dir, err)
	}
	if len(violations) != 0 {
		t.Errorf("scanSequenceStateGuard(%q) = %v, want no violations — the counter is in a _test.go file", dir, violations)
	}
}

// R-AEE-012, S-AEE-036 — the landed allowlist has exactly one entry, naming
// message.go/lastMessageID with a non-empty rationale naming C3. Like
// TestLayer1_ModuleHasNoDependencies_ZeroRequires (import_boundary_test.go),
// this is a pin — green from birth, exempt from red-first — because it
// asserts a property of already-declared data rather than driving new
// guard logic.
func TestSequenceStateAllowlist_HasExactlyOneReasonedEntry(t *testing.T) {
	t.Parallel()

	if len(sequenceStateAllowlist) != 1 {
		t.Fatalf("len(sequenceStateAllowlist) = %d, want exactly 1", len(sequenceStateAllowlist))
	}
	entry := sequenceStateAllowlist[0]
	if entry.file != "message.go" {
		t.Errorf("sequenceStateAllowlist[0].file = %q, want %q", entry.file, "message.go")
	}
	if entry.identifier != "lastMessageID" {
		t.Errorf("sequenceStateAllowlist[0].identifier = %q, want %q", entry.identifier, "lastMessageID")
	}
	if entry.rationale == "" {
		t.Error("sequenceStateAllowlist[0].rationale is empty, want a non-empty rationale naming C3")
	}
	if !strings.Contains(entry.rationale, "C3") {
		t.Errorf("sequenceStateAllowlist[0].rationale = %q, want it to name C3", entry.rationale)
	}
}

// R-AEE-012, S-AEE-037 — with the allowlist entry removed, the guard fails
// naming lastMessageID: Pass 1's admission check already requires an exact
// (file, identifier) match, so a missing entry is indistinguishable from one
// that was never written. Green from birth against the 4.2 implementation —
// recorded here as the scenario, not as new drive-out logic.
func TestSequenceGuard_AllowlistEntryRemoved_FailsNamingLastMessageID(t *testing.T) {
	t.Parallel()

	violations, err := scanSequenceStateGuard(packageDir, nil)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", packageDir, err)
	}
	assertNamesViolation(t, violations, "message.go", "lastMessageID")
}

// R-AEE-012, S-AEE-038 — an allowlist entry naming an identifier that does
// not exist fails as stale, naming the entry.
func TestSequenceGuard_AllowlistEntryNamesNonexistentIdentifier_FailsAsStale(t *testing.T) {
	t.Parallel()

	stale := []allowlistEntry{
		{file: "message.go", identifier: "doesNotExist", rationale: "placeholder rationale for a test that names a nonexistent identifier"},
	}
	violations, err := scanSequenceStateGuard(packageDir, stale)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", packageDir, err)
	}

	found := false
	for _, v := range violations {
		if v.file == "message.go" && v.identifier == "doesNotExist" && strings.Contains(v.reason, "stale") {
			found = true
		}
	}
	if !found {
		t.Errorf("violations = %v, want one naming message.go/doesNotExist as a stale allowlist entry", violations)
	}
}

// R-AEE-012, S-AEE-039 — an allowlist entry with an empty rationale fails,
// naming it, even though its (file, identifier) exactly matches a real
// qualifying var.
func TestSequenceGuard_AllowlistEntryHasEmptyRationale_Fails(t *testing.T) {
	t.Parallel()

	unreasoned := []allowlistEntry{
		{file: "message.go", identifier: "lastMessageID", rationale: ""},
	}
	violations, err := scanSequenceStateGuard(packageDir, unreasoned)
	if err != nil {
		t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", packageDir, err)
	}

	found := false
	for _, v := range violations {
		if v.file == "message.go" && v.identifier == "lastMessageID" && strings.Contains(v.reason, "rationale") {
			found = true
		}
	}
	if !found {
		t.Errorf("violations = %v, want one naming message.go/lastMessageID's empty rationale", violations)
	}
}

// R-AEE-013, S-AEE-041 — this file's own doc comment names C3, the retired
// process-global counter, and states the fix: not a smaller counter, the
// counter where the stream is. Mirrors sequence_test.go's
// TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule: read what a reader
// would read, rather than trust that the prose exists.
func TestSequenceGuardGoFile_PackageDoc_NamesC3AndTheFix(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sequence_guard_test.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing sequence_guard_test.go: %v", err)
	}
	if file.Doc == nil {
		t.Fatal("sequence_guard_test.go carries no package doc comment; the guard would pass vacuously")
	}
	doc := strings.Join(strings.Fields(file.Doc.Text()), " ")

	if !strings.Contains(doc, "C3") {
		t.Errorf("sequence_guard_test.go's doc does not name C3:\n%s", doc)
	}
	if !strings.Contains(doc, "process-global") && !strings.Contains(doc, "process-wide") {
		t.Errorf("sequence_guard_test.go's doc does not name the retired process-global/process-wide counter:\n%s", doc)
	}
	if !strings.Contains(doc, "not a smaller counter") {
		t.Errorf("sequence_guard_test.go's doc does not state the fix (\"not a smaller counter\"):\n%s", doc)
	}
	if !strings.Contains(doc, "counter where the stream is") {
		t.Errorf("sequence_guard_test.go's doc does not state the fix (\"counter where the stream is\"):\n%s", doc)
	}
}

// R-AEE-013, S-AEE-042 — the guard's failure message always names the
// offending file and identifier (guardViolation.String()), and the two
// allowlist-admission failure reasons point the reader at sequenceStateAllowlist
// and its rationale field by name.
func TestSequenceGuard_FailureMessage_NamesFileIdentifierAndPointsAtRationale(t *testing.T) {
	t.Parallel()

	t.Run("String() always renders file and identifier", func(t *testing.T) {
		t.Parallel()

		v := guardViolation{file: "scratch.go", identifier: "scratchSeq", reason: "some reason"}
		got := v.String()
		if !strings.Contains(got, "scratch.go") || !strings.Contains(got, "scratchSeq") {
			t.Errorf("guardViolation.String() = %q, want it to contain both the file and the identifier", got)
		}
	})

	t.Run("an unadmitted var's reason points at sequenceStateAllowlist and rationale", func(t *testing.T) {
		t.Parallel()

		violations, err := scanSequenceStateGuard(packageDir, nil)
		if err != nil {
			t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", packageDir, err)
		}
		v := findViolation(violations, "message.go", "lastMessageID")
		if v == nil {
			t.Fatal("no violation naming message.go/lastMessageID")
		}
		if !strings.Contains(v.reason, "sequenceStateAllowlist") || !strings.Contains(v.reason, "rationale") {
			t.Errorf("reason = %q, want it to name sequenceStateAllowlist and rationale", v.reason)
		}
	})

	t.Run("an empty-rationale entry's reason points at rationale", func(t *testing.T) {
		t.Parallel()

		unreasoned := []allowlistEntry{{file: "message.go", identifier: "lastMessageID", rationale: ""}}
		violations, err := scanSequenceStateGuard(packageDir, unreasoned)
		if err != nil {
			t.Fatalf("scanSequenceStateGuard(%q) returned an error: %v", packageDir, err)
		}
		v := findViolation(violations, "message.go", "lastMessageID")
		if v == nil {
			t.Fatal("no violation naming message.go/lastMessageID")
		}
		if !strings.Contains(v.reason, "rationale") {
			t.Errorf("reason = %q, want it to name rationale", v.reason)
		}
	})
}

// findViolation returns the first violation naming exactly file and
// identifier, or nil.
func findViolation(violations []guardViolation, file, identifier string) *guardViolation {
	for i := range violations {
		if violations[i].file == file && violations[i].identifier == identifier {
			return &violations[i]
		}
	}
	return nil
}

// writeScratchPackage materializes files (base name -> source text) into a
// fresh t.TempDir and returns its path. Every scratch-subject test in this
// file uses it instead of touching a real source file in this package,
// so the guard's own detection logic is proven and re-proven on every test
// run rather than by a one-time manual edit-and-revert.
func writeScratchPackage(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatalf("writing scratch file %s: %v", name, err)
		}
	}
	return dir
}

// assertNamesViolation fails t unless violations contains one naming exactly
// file and identifier.
func assertNamesViolation(t *testing.T, violations []guardViolation, file, identifier string) {
	t.Helper()

	for _, v := range violations {
		if v.file == file && v.identifier == identifier {
			return
		}
	}
	t.Errorf("violations = %v, want one naming file %q identifier %q", violations, file, identifier)
}

// guardViolation is one thing scanSequenceStateGuard found wrong: a
// package-level var shaped like process-global sequence state with no
// admitting allowlist entry, an allowlist entry that does not hold up, or a
// function that resets a qualifying var.
type guardViolation struct {
	file       string
	identifier string
	reason     string
}

func (v guardViolation) String() string {
	return v.file + ": " + v.identifier + ": " + v.reason
}

// allowlistEntry is R-AEE-012's admission record: a file, the identifier it
// names, and why that identifier is not V-STR-13 sequence state.
type allowlistEntry struct {
	file       string
	identifier string
	rationale  string
}

// qualifyingVar is where in the scanned source a qualifying package-level
// var was declared.
type qualifyingVar struct {
	file  string
	ident string
}

// scanSequenceStateGuard is R-AEE-011's mechanical AST scan applied to dir:
// every non-test ".go" file directly inside it — never recursing, so a
// fixture directory such as testdata is never in scope, matching how Go
// itself treats "the package". It fails on a package-level var whose type
// is, or recursively contains, an integer type or a sync/atomic type
// (qualifies, below), unless allowlist admits it, and on a package-level
// function that resets one.
//
// It returns every violation found, not just the first, and a nil error
// paired with a nil/empty slice means dir passes clean.
func scanSequenceStateGuard(dir string, allowlist []allowlistEntry) ([]guardViolation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	fileOf := make(map[*ast.File]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		files = append(files, f)
		fileOf[f] = name
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s carries no non-test .go file; the guard would scan nothing", dir)
	}

	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	if _, err := conf.Check(files[0].Name.Name, fset, files, info); err != nil {
		return nil, fmt.Errorf("type-checking %s: %w", dir, err)
	}

	allowed := make(map[[2]string]allowlistEntry, len(allowlist))
	for _, e := range allowlist {
		allowed[[2]string{e.file, e.identifier}] = e
	}

	qualifying := make(map[types.Object]qualifyingVar)
	var violations []guardViolation

	// Pass 1 — every qualifying package-level var, flagged unless the
	// allowlist admits it by exact (file, identifier).
	for _, f := range files {
		fname := fileOf[f]
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					if name.Name == "_" {
						continue
					}
					obj, ok := info.Defs[name]
					if !ok {
						continue
					}
					v, ok := obj.(*types.Var)
					if !ok || !qualifies(v.Type()) {
						continue
					}
					qualifying[obj] = qualifyingVar{file: fname, ident: name.Name}

					entry, admitted := allowed[[2]string{fname, name.Name}]
					switch {
					case admitted && entry.rationale != "":
						continue
					case admitted:
						violations = append(violations, guardViolation{
							file:       fname,
							identifier: name.Name,
							reason: "its entry in sequenceStateAllowlist carries an empty rationale — " +
								"R-AEE-012 requires a non-empty one",
						})
					default:
						violations = append(violations, guardViolation{
							file:       fname,
							identifier: name.Name,
							reason: "package-level var holds sequence-shaped state (an integer or sync/atomic type) with " +
								"no admitting entry in sequenceStateAllowlist — R-AEE-012 requires one, with a rationale",
						})
					}
				}
			}
		}
	}

	// Staleness — every allowlist entry with a rationale must name a var
	// Pass 1 actually found qualifying, in the exact file it names. An entry
	// naming a real var is validated above (admitted / empty-rationale); an
	// entry that matches nothing found is stale.
	qualifyingByLocation := make(map[[2]string]bool, len(qualifying))
	for _, qv := range qualifying {
		qualifyingByLocation[[2]string{qv.file, qv.ident}] = true
	}
	for _, e := range allowlist {
		if e.rationale == "" {
			continue // already reported above if the var exists at all
		}
		if qualifyingByLocation[[2]string{e.file, e.identifier}] {
			continue
		}
		violations = append(violations, guardViolation{
			file:       e.file,
			identifier: e.identifier,
			reason: "allowlist entry is stale: no qualifying package-level var of that name exists in " +
				"that file — R-AEE-012",
		})
	}

	// Pass 2 — any package-level function that resets a qualifying var, by
	// plain assignment, increment/decrement, or Store/Swap/CompareAndSwap.
	// Add stays legal: it is how message.go's mintMessageID mints an identity,
	// not a reset.
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch stmt := n.(type) {
				case *ast.AssignStmt:
					for _, lhs := range stmt.Lhs {
						if qv, ok := resolveQualifying(lhs, info, qualifying); ok {
							violations = append(violations, resetViolation(qv, fd.Name.Name, "assignment"))
						}
					}
				case *ast.IncDecStmt:
					if qv, ok := resolveQualifying(stmt.X, info, qualifying); ok {
						violations = append(violations, resetViolation(qv, fd.Name.Name, "increment/decrement"))
					}
				case *ast.CallExpr:
					sel, ok := stmt.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					switch sel.Sel.Name {
					case "Store", "Swap", "CompareAndSwap":
						if qv, ok := resolveQualifying(sel.X, info, qualifying); ok {
							violations = append(violations, resetViolation(qv, fd.Name.Name, "."+sel.Sel.Name+"(...)"))
						}
					}
				}
				return true
			})
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].file != violations[j].file {
			return violations[i].file < violations[j].file
		}
		if violations[i].identifier != violations[j].identifier {
			return violations[i].identifier < violations[j].identifier
		}
		return violations[i].reason < violations[j].reason
	})
	return violations, nil
}

// resetViolation is Pass 2's shared constructor, naming both the reset
// qualifying var and, in its reason, the function that resets it and the
// mechanism — and restating R-AEE-013's fix in the reader's path, not just
// this file's header: not a smaller counter, putting the counter where the
// stream is.
func resetViolation(qv qualifyingVar, funcName, mechanism string) guardViolation {
	return guardViolation{
		file:       qv.file,
		identifier: qv.ident,
		reason: fmt.Sprintf(
			"%s resets it (%s) — R-AEE-011: the fix for process-global sequence state is not a smaller counter, it is putting the counter where the stream is",
			funcName, mechanism,
		),
	}
}

// resolveQualifying reports the qualifying var e's root identifier resolves
// to via info.Uses, and whether it does — e.g. scratchSeq in "scratchSeq =
// 0", or scratchCounter in "scratchCounter.n = 0" (any depth: a
// struct-wrapped counter's reset must be caught the same way a bare one's
// is).
func resolveQualifying(e ast.Expr, info *types.Info, qualifying map[types.Object]qualifyingVar) (qualifyingVar, bool) {
	ident := rootIdent(e)
	if ident == nil {
		return qualifyingVar{}, false
	}
	obj, ok := info.Uses[ident]
	if !ok {
		return qualifyingVar{}, false
	}
	qv, ok := qualifying[obj]
	return qv, ok
}

// rootIdent unwraps selector, index, pointer-dereference and parenthesized
// expressions down to the base identifier, or reports none for anything
// else (e.g. a function call result has no root identifier to resolve).
func rootIdent(e ast.Expr) *ast.Ident {
	for {
		switch v := e.(type) {
		case *ast.Ident:
			return v
		case *ast.SelectorExpr:
			e = v.X
		case *ast.IndexExpr:
			e = v.X
		case *ast.StarExpr:
			e = v.X
		case *ast.ParenExpr:
			e = v.X
		default:
			return nil
		}
	}
}

// qualifies reports whether t is, or recursively contains, an integer type
// (int, sized int*/uint*, uintptr) or a named sync/atomic type — R-AEE-011's
// "recursively contains" floor: a named type via its underlying type, a
// struct's fields at any depth, an array's element. There is no depth cap:
// Go forbids a by-value type cycle, so this recursion always terminates
// structurally (design.md D8).
//
// Slices, maps, pointers, channels and funcs are deliberately NOT recursed.
// design.md D8 records why: covering slices would flag eventRegistry itself
// (a slice whose element's descriptor field holds only uint8 enums),
// forcing a second allowlist entry R-AEE-012's "exactly one" forbids.
func qualifies(t types.Type) bool {
	t = types.Unalias(t)
	switch u := t.(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
			types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
			types.Uintptr:
			return true
		default:
			return false
		}
	case *types.Named:
		if isSyncAtomicType(u) {
			return true
		}
		return qualifies(u.Underlying())
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if qualifies(u.Field(i).Type()) {
				return true
			}
		}
		return false
	case *types.Array:
		return qualifies(u.Elem())
	case *types.TypeParam:
		// Fail-closed: a bare type parameter's qualification cannot be
		// decided at declaration time. Unreachable at package-var scope —
		// only a function or a generic type carries one — but coded rather
		// than assumed (design.md D8).
		return true
	default:
		// Slice, Map, Pointer, Chan, Signature and everything else:
		// deliberately not recursed. See the func comment.
		return false
	}
}

// isSyncAtomicType reports whether n is a named type declared in package
// sync/atomic — atomic.Uint64, atomic.Int64 and every sibling — matched by
// declaring package rather than by name, so a type sync/atomic adds later
// qualifies with no edit here.
func isSyncAtomicType(n *types.Named) bool {
	obj := n.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == "sync/atomic"
}

// sequenceStateAllowlist is R-AEE-012's allowlist: file, identifier and a
// non-empty rationale, admitted only when both match exactly and the
// rationale is not empty. Exactly one entry at this milestone.
var sequenceStateAllowlist = []allowlistEntry{
	{
		file:       "message.go",
		identifier: "lastMessageID",
		rationale: "C3 was a process-global atomic sequence counter; the fix (R-AEE-008, " +
			"sequence.go) is not a smaller counter, it is putting the counter where the " +
			"stream is. lastMessageID is a different property: V-REQ-03 message identity " +
			"needs only that two messages be distinguishable, which any process-wide " +
			"monotonic counter gives for every message in the process. V-STR-13 stream " +
			"sequencing needs that every stream's first event carries 1 and is " +
			"independently contiguous, which a process-wide counter structurally cannot " +
			"give — that gap is exactly C3. message.go's own comment on lastMessageID " +
			"draws the same line.",
	},
}
