// AG-12 — the closed-route guard: R-HIS-004's "exactly one commit path, no
// privileged bypass" enumerated in test, not merely asserted in prose
// (S-HIS-030), with a closed comparison against the real exported surface
// so the enumeration itself cannot silently drift into a stale snapshot
// (S-HIS-031). Mirrors doc_contract_guard_test.go's own closed-comparison
// shape (S-AGP-013/014), retargeted from a doc-comment table onto the
// package's real exported surface: reflection for method sets (runtime
// truth — catches a method promoted through an accidentally embedded
// field, which source scanning would miss) and go/parser for package-level
// functions (reflection cannot enumerate those).
package agent_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// routeClass discriminates what an enumerated history route is permitted
// to do.
type routeClass uint8

const (
	// routeMutating can extend or mutate the transcript; R-HIS-004
	// obligates it to reject an orphaning sequence, and the S-HIS-030
	// audit below drives one through each such row.
	routeMutating routeClass = iota + 1

	// routeReadOnly can only observe already-committed state.
	routeReadOnly

	// routeEmptyConstructor commits nothing — an empty transcript is
	// trivially valid, so NewHistory alone carries this class.
	routeEmptyConstructor
)

// historyRoute is one row of the committed expectation table: a route's
// name, the surface it was found on, and its permitted class.
type historyRoute struct {
	name  string // "Append", "NewSeededHistory", ...
	kind  string // "method:*History" | "method:Entry" | "func"
	class routeClass
}

// expectedHistoryRoutes is the committed source of truth this guard diffs
// the real exported surface against, set-equal in both directions
// (S-HIS-031: an un-diffed hand-written route list would not fail when a
// sixth mutating route landed silently — this table is worthless unless
// something checks it against reality, which is exactly what
// TestHistoryRouteGuard_SurfaceMatchesExpectedTable does).
var expectedHistoryRoutes = []historyRoute{
	{"NewHistory", "func", routeEmptyConstructor},
	{"NewSeededHistory", "func", routeMutating},
	{"Append", "method:*History", routeMutating},
	{"CloseTurn", "method:*History", routeMutating},
	{"SynthesizeOrphans", "method:*History", routeMutating},
	{"Entries", "method:*History", routeReadOnly},
	{"Len", "method:*History", routeReadOnly},
	{"ID", "method:Entry", routeReadOnly},
	{"Message", "method:Entry", routeReadOnly},
	{"Origin", "method:Entry", routeReadOnly},
	{"String", "method:Entry", routeReadOnly},
}

// surfaceRoute is the (name, kind) pair the real surface and the
// expectation table are compared over — class is authored data in the
// table, not part of the surface itself, so it is compared separately
// (S-HIS-042's check below).
type surfaceRoute struct {
	name string
	kind string
}

// historyPackageDir resolves package agent's own directory relative to
// THIS test file's own source location via runtime.Caller(0) — the doc
// guard's posture (docGoPath, providerGoPath), not an assumed working
// directory.
func historyPackageDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve the agent package directory")
	}
	return filepath.Dir(thisFile)
}

// packageLevelHistoryRoutes enumerates every exported top-level function
// in dir's non-test .go files whose signature — a parameter type or a
// result type, at any nesting depth — mentions History, by parsing with
// go/parser (stdlib), never by importing the package. Go reflection
// cannot enumerate package-level functions, so the constructors take this
// parse probe; methods keep reflection (actualHistoryRoutes).
func packageLevelHistoryRoutes(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}

	fset := token.NewFileSet()
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v, want nil", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			if funcSignatureMentionsHistory(fn.Type) {
				names = append(names, fn.Name.Name)
			}
		}
	}
	return names
}

// funcSignatureMentionsHistory reports whether any parameter or result
// type in funcType's declared signature names the identifier History, at
// any nesting depth (a pointer, a slice, ...) — a generic AST walk rather
// than a hand-enumerated set of shapes, so a future signature that wraps
// History differently is still caught.
func funcSignatureMentionsHistory(funcType *ast.FuncType) bool {
	mentions := false
	scan := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			ast.Inspect(field.Type, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name == "History" {
					mentions = true
				}
				return true
			})
		}
	}
	scan(funcType.Params)
	scan(funcType.Results)
	return mentions
}

// actualHistoryRoutes enumerates the REAL exported surface: the method
// sets of *agent.History and agent.Entry by reflection, plus
// package-level functions whose signature mentions History by parsing
// (packageLevelHistoryRoutes).
func actualHistoryRoutes(t *testing.T) []surfaceRoute {
	t.Helper()

	var routes []surfaceRoute

	historyType := reflect.TypeOf(&agent.History{})
	for i := range historyType.NumMethod() {
		routes = append(routes, surfaceRoute{name: historyType.Method(i).Name, kind: "method:*History"})
	}

	entryType := reflect.TypeOf(agent.Entry{})
	for i := range entryType.NumMethod() {
		routes = append(routes, surfaceRoute{name: entryType.Method(i).Name, kind: "method:Entry"})
	}

	for _, name := range packageLevelHistoryRoutes(t, historyPackageDir(t)) {
		routes = append(routes, surfaceRoute{name: name, kind: "func"})
	}

	return routes
}

// TestHistoryRouteGuard_SurfaceMatchesExpectedTable (S-HIS-031, S-HIS-042)
// — the real exported surface and expectedHistoryRoutes are set-equal in
// BOTH directions: a surface route with no table row fails, naming it
// (the S-HIS-031 bite target); a table row with no matching surface route
// fails too, naming the stale entry. Every method:Entry row must be
// routeReadOnly — the envelope exposes no route to set an entry identity,
// an origin discriminator, or a committed Layer 1 value (S-HIS-042).
func TestHistoryRouteGuard_SurfaceMatchesExpectedTable(t *testing.T) {
	t.Parallel()

	actual := actualHistoryRoutes(t)
	actualSet := make(map[surfaceRoute]bool, len(actual))
	for _, r := range actual {
		actualSet[r] = true
	}

	expectedSet := make(map[surfaceRoute]bool, len(expectedHistoryRoutes))
	for _, r := range expectedHistoryRoutes {
		expectedSet[surfaceRoute{name: r.name, kind: r.kind}] = true
	}

	for r := range actualSet {
		if !expectedSet[r] {
			t.Errorf("history route guard: surface route %q (%s) is not in expectedHistoryRoutes — every exported history route must be classified mutating/read-only in the same PR that adds it (R-HIS-004)", r.name, r.kind)
		}
	}
	for _, want := range expectedHistoryRoutes {
		key := surfaceRoute{name: want.name, kind: want.kind}
		if !actualSet[key] {
			t.Errorf("history route guard: expectedHistoryRoutes entry %q (%s) has no matching surface route — the committed table has drifted ahead of the exported surface (R-HIS-004)", want.name, want.kind)
		}
	}

	for _, r := range expectedHistoryRoutes {
		if r.kind == "method:Entry" && r.class != routeReadOnly {
			t.Errorf("history route guard: Entry method %q is classified %d, want routeReadOnly — the envelope must expose no route to set an entry identity, an origin discriminator, or a committed Layer 1 value (S-HIS-042)", r.name, r.class)
		}
	}
}

// requireUnresolvedReferenceAt fails t unless err is a typed
// *ai.Violation matching ai.ErrUnresolvedReference.
func requireUnresolvedReferenceAt(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("got nil error, want an *ai.Violation matching ai.ErrUnresolvedReference")
	}
	if !errors.Is(err, ai.ErrUnresolvedReference) {
		t.Errorf("errors.Is(err, ai.ErrUnresolvedReference) = false, want true (err = %v)", err)
	}
}

// requireEmptyAtHistory fails t unless err is a typed *ai.Violation
// matching ai.ErrEmpty positioned at "history" — the zero-value
// rejection every route shares (the C1 closure point).
func requireEmptyAtHistory(t *testing.T, err error) {
	t.Helper()
	requireViolation(t, err, ai.ErrEmpty, "history")
}

// TestHistory_NoBypass_EveryMutatingRouteRejectsOrphaningSequence
// (S-HIS-030) — every routeMutating row of expectedHistoryRoutes has a
// driver that constructs an orphaning sequence through that exact route
// and proves it is rejected typed with state left byte-unchanged; a
// mutating row with no driver fails, naming it, so a new mutating route
// cannot land silently unaudited. Each driver also proves new(History) is
// rejected through its own route — the zero-value door C1 exploited,
// closed at the one point every route already crosses.
func TestHistory_NoBypass_EveryMutatingRouteRejectsOrphaningSequence(t *testing.T) {
	t.Parallel()

	drivers := map[string]func(t *testing.T){
		"Append":            driveAppendOrphaningSequence,
		"NewSeededHistory":  driveNewSeededHistoryOrphaningSequence,
		"CloseTurn":         driveCloseTurnOrphaningSequence,
		"SynthesizeOrphans": driveSynthesizeOrphansOrphaningSequence,
	}

	for _, route := range expectedHistoryRoutes {
		if route.class != routeMutating {
			continue
		}
		driver, ok := drivers[route.name]
		if !ok {
			t.Errorf("history route guard: no orphaning-sequence driver for mutating route %q", route.name)
			continue
		}
		t.Run(route.name, driver)
	}
}

// driveAppendOrphaningSequence proves Append rejects an orphaned result
// typed, leaves state byte-unchanged, and rejects through the zero value.
func driveAppendOrphaningSequence(t *testing.T) {
	h := agent.NewHistory()
	orphan := resultMessage(t, "call-orphan")
	requireUnresolvedReferenceAt(t, h.Append(orphan))
	if h.Len() != 0 {
		t.Errorf("Append committed %d entries after a rejected orphan append, want 0 (state must stay byte-unchanged)", h.Len())
	}

	zero := new(agent.History)
	requireEmptyAtHistory(t, zero.Append(orphan))
}

// driveNewSeededHistoryOrphaningSequence proves NewSeededHistory rejects
// a seed containing an orphaned result typed, returning (nil, err).
func driveNewSeededHistoryOrphaningSequence(t *testing.T) {
	orphan := resultMessage(t, "call-orphan")
	h, err := agent.NewSeededHistory([]ai.Message{orphan})
	requireUnresolvedReferenceAt(t, err)
	if h != nil {
		t.Error("NewSeededHistory returned a non-nil *History alongside an error, want nil")
	}
}

// driveCloseTurnOrphaningSequence proves CloseTurn rejects an unclosed
// call typed, leaves state byte-unchanged, and rejects through the zero
// value.
func driveCloseTurnOrphaningSequence(t *testing.T) {
	h := agent.NewHistory()
	if err := h.Append(callMessage(t, "call-unclosed")); err != nil {
		t.Fatalf("Append(open call) returned %v, want nil", err)
	}
	requireViolation(t, h.CloseTurn(), ai.ErrEmpty, "messages[0].content[0].result")
	if h.Len() != 1 {
		t.Errorf("CloseTurn changed entry count to %d after rejection, want 1 unchanged", h.Len())
	}

	zero := new(agent.History)
	requireEmptyAtHistory(t, zero.CloseTurn())
}

// driveSynthesizeOrphansOrphaningSequence proves SynthesizeOrphans
// rejects through the zero value — its only bypass surface, since it
// takes no message parameter of its own to build an orphaning sequence
// from.
func driveSynthesizeOrphansOrphaningSequence(t *testing.T) {
	zero := new(agent.History)
	n, err := zero.SynthesizeOrphans()
	requireEmptyAtHistory(t, err)
	if n != 0 {
		t.Errorf("new(History).SynthesizeOrphans() returned n = %d, want 0", n)
	}
}
