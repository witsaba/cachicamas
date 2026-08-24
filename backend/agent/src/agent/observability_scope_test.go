// AG-22 — Phase 11 (S-AGS-068, agent-v1-scope delta): the seam-not-sink
// distinction is checked against the shipped exported surface, not
// against prose.
//
// S-AGS-066 (agent-v1-scope/spec.md) asserted the package's exported
// surface declares "...and no telemetry sink". Its enforcing test
// (hooks_test.go's TestHooks_S_AGS_066_Layer3HalfCheckedAgainstShippedSurface)
// checks only hooks.go's wall clocks and zero-value Hooks inertness — it
// never asserted anything about telemetry at all. Design D-A's
// Harness.TracerProvider field falsifies S-AGS-066's literal sentence in
// letter, so the agent-v1-scope delta amends it (a *seam*, not a *sink*)
// and adds THIS scenario, S-AGS-068, as the check that keeps the amended
// claim honest against the real, compiled surface rather than against
// this file's own prose — mirroring history_surface_guard_test.go's
// closed-comparison shape: reflection for the struct field (runtime
// truth), go/parser for every OTHER exported package-level declaration
// (reflection cannot enumerate those).
package agent_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/agent"
)

// agoScopePackageDir resolves package agent's own directory relative to
// THIS test file's own source location (historyPackageDir's posture,
// restated here rather than shared, per this codebase's own per-file
// self-containment convention — docGoPath/providerGoPath/historyPackageDir
// each do the identical runtime.Caller(0) resolution independently).
func agoScopePackageDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve the agent package directory")
	}
	return filepath.Dir(thisFile)
}

// agoOtelImportAliases returns the local identifier names file's own
// import block binds to a "go.opentelemetry.io/otel" package path — for
// example {"trace": true} for a plain import, or the custom name for a
// renamed one. Every file this scenario cares about is scanned with its
// OWN import block, never a hardcoded alias table, so a renamed import
// is still caught correctly.
func agoOtelImportAliases(file *ast.File) map[string]bool {
	aliases := make(map[string]bool)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(path, "go.opentelemetry.io/otel") {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		aliases[name] = true
	}
	return aliases
}

// agoDeclMentionsOtel reports whether node's subtree contains a selector
// expression (pkg.Ident) whose package identifier is one of otelAliases
// — i.e. the declaration's own text names an OpenTelemetry-imported
// identifier anywhere in it, at any nesting depth (mirroring
// funcSignatureMentionsHistory's generic ast.Inspect walk).
func agoDeclMentionsOtel(node ast.Node, otelAliases map[string]bool) bool {
	if node == nil || len(otelAliases) == 0 {
		return false
	}
	mentions := false
	ast.Inspect(node, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && otelAliases[id.Name] {
				mentions = true
			}
		}
		return true
	})
	return mentions
}

// agoExportedOtelSurface parses every non-test .go file directly inside
// dir (package agent's own directory; no subpackages) and returns two
// disjoint lists: otherMembers names every EXPORTED, non-Harness
// top-level declaration (free function, exported method, type, var,
// const) whose own signature/declaration mentions an OpenTelemetry-
// imported identifier — this scenario expects this list EMPTY; and
// harnessFields names the Harness struct's own exported fields that do
// the same — this scenario expects exactly one, "TracerProvider". Only
// a declaration's own signature/type is scanned, never a function body:
// Harness.Run's internal tracer.Start call is implementation, not
// surface (the same distinction R-AGO-001's own injection rule draws).
func agoExportedOtelSurface(t *testing.T, dir string) (otherMembers []string, harnessFields []string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, ferr := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if ferr != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v, want nil", name, ferr)
		}
		aliases := agoOtelImportAliases(file)

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				if agoDeclMentionsOtel(d.Type, aliases) {
					otherMembers = append(otherMembers, "func:"+d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						if s.Name.Name == "Harness" {
							st, ok := s.Type.(*ast.StructType)
							if !ok {
								t.Fatalf("agent.Harness is no longer a struct type in %q", name)
							}
							for _, field := range st.Fields.List {
								if !agoDeclMentionsOtel(field.Type, aliases) {
									continue
								}
								for _, fname := range field.Names {
									if fname.IsExported() {
										harnessFields = append(harnessFields, fname.Name)
									}
								}
							}
							continue
						}
						if agoDeclMentionsOtel(s.Type, aliases) {
							otherMembers = append(otherMembers, "type:"+s.Name.Name)
						}
					case *ast.ValueSpec:
						for i, vname := range s.Names {
							if !vname.IsExported() {
								continue
							}
							var typeNode ast.Node = s.Type
							if typeNode == nil && i < len(s.Values) {
								typeNode = s.Values[i]
							}
							if agoDeclMentionsOtel(typeNode, aliases) {
								kind := "var"
								if d.Tok == token.CONST {
									kind = "const"
								}
								otherMembers = append(otherMembers, kind+":"+vname.Name)
							}
						}
					}
				}
			}
		}
	}
	return otherMembers, harnessFields
}

// TestObservability_S_AGS_068_SeamNotSinkCheckedAgainstShippedSurface is
// the agent-v1-scope delta's own scenario, discharged here rather than
// left as prose. Each subtest maps to one of S-AGS-068's own "when...
// then" clauses; the G11-orphan-check clause (S-AGS-048's row still
// naming CO-24.1/CO-24.2) is a spec-text claim about
// agent-v1-scope/spec.md's own content, not this package's code, and is
// verified by inspection (recorded in apply-progress) rather than here.
func TestObservability_S_AGS_068_SeamNotSinkCheckedAgainstShippedSurface(t *testing.T) {
	t.Parallel()

	t.Run("exported_surface_has_exactly_one_telemetry_member", func(t *testing.T) {
		t.Parallel()

		otherMembers, harnessFields := agoExportedOtelSurface(t, agoScopePackageDir(t))
		for _, m := range otherMembers {
			t.Errorf("exported package member %q mentions an OpenTelemetry-imported identifier in its own signature/declaration — the surface must declare no exporter, no SDK type, no provider-registration function and no process-global setter beyond Harness.TracerProvider", m)
		}
		if len(harnessFields) != 1 || harnessFields[0] != "TracerProvider" {
			t.Fatalf("Harness's own exported OpenTelemetry-mentioning field(s) = %v, want exactly [\"TracerProvider\"]", harnessFields)
		}

		field, ok := reflect.TypeOf(agent.Harness{}).FieldByName("TracerProvider")
		if !ok {
			t.Fatal("agent.Harness has no TracerProvider field")
		}
		wantType := reflect.TypeOf((*trace.TracerProvider)(nil)).Elem()
		if field.Type != wantType {
			t.Errorf("Harness.TracerProvider field type = %v, want %v (the OpenTelemetry API's own provider interface)", field.Type, wantType)
		}
		if field.Type.Kind() != reflect.Interface {
			t.Errorf("Harness.TracerProvider field kind = %v, want Interface — a seam, never a concrete SDK type standing in for one", field.Type.Kind())
		}
	})

	t.Run("harness_with_unset_provider_completes_normally", func(t *testing.T) {
		t.Parallel()

		// agoParityRun (observability_parity_test.go) already recovers any
		// panic and fails on a non-nil Run error — reused directly rather
		// than duplicated (R-AGO-009's own proof, restated here as this
		// scenario's own clause rather than assumed from the other file).
		events := agoParityRun(t, "ags068_scope_tool", nil)
		if len(events) == 0 {
			t.Fatal("agoParityRun with an unset TracerProvider drained zero events — want a genuine, non-trivial completed run")
		}
	})

	t.Run("production_sources_touch_no_process_global_telemetry_state", func(t *testing.T) {
		t.Parallel()

		root, err := gitTopLevel(t)
		if err != nil {
			t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
		}
		files := []string{"harness.go", "loop.go", "scheduler.go", "compaction.go", "observability.go"}
		globalSymbols := []string{"otel.SetTracerProvider", "otel.GetTracerProvider", "otel.Tracer(", "otel.SetTextMapPropagator", "otel.SetMeterProvider"}
		for _, name := range files {
			raw, rerr := os.ReadFile(root + "/backend/agent/src/agent/" + name)
			if rerr != nil {
				t.Fatalf("os.ReadFile(%s) failed: %v", name, rerr)
			}
			src := string(raw)
			// The bare root package ("go.opentelemetry.io/otel", as opposed
			// to its "/trace", "/attribute" or "/codes" subpackages) is the
			// ONLY place a process-global getter/setter lives (design D-A's
			// own rejection) — an exact, non-prefixed import match, so
			// "go.opentelemetry.io/otel/trace" does not false-positive.
			// Import lines are never inside a comment, so this loop scans
			// the raw source rather than the comment-stripped copy below.
			for _, line := range strings.Split(src, "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed == `"go.opentelemetry.io/otel"` {
					t.Errorf("%s imports the bare go.opentelemetry.io/otel root package — that is the ONLY place a process-global telemetry getter/setter lives (design D-A's own rejection)", name)
				}
			}
			// Comments stripped before the substring scans below: this
			// package's own doc comments (observability.go's package
			// comment) NAME "otel.Tracer(...)" and "RecordError" while
			// explaining they are rejected/never called — a raw substring
			// scan over the comment text would false-positive on the very
			// sentence documenting the rule (this codebase's own "a
			// regression guard is unproven until planted" lesson: strip
			// comments before matching code shapes).
			scanned := stripGoComments(src)
			for _, sym := range globalSymbols {
				if strings.Contains(scanned, sym) {
					t.Errorf("%s contains %q, want no process-global telemetry read or write anywhere in production Layer 2 code", name, sym)
				}
			}
			if strings.Contains(scanned, "RecordError") {
				t.Errorf("%s calls Span.RecordError — forbidden: it would carry a wrapped error's text into a span event (S-AGO-026)", name)
			}
		}
	})

	t.Run("go_mod_go_sum_and_src_ai_byte_unchanged", func(t *testing.T) {
		t.Parallel()

		root, err := gitTopLevel(t)
		if err != nil {
			t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
		}
		baseRef := hksResolveBaseRef(t, root)
		aiDiff, derr := gitDiff(t, root, baseRef, "backend/agent/src/ai/")
		if derr != nil {
			t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", baseRef, derr)
		}
		// CH-03 carve-out: wantGoModRequires / wantGoModRequireLines
		// is bumped from 3 to 4 entries in two ai/ test files
		// (import_boundary_test.go and zero_requires_test.go). Anything
		// beyond these two files fails.
		ch03AiCarveout := []string{
			"backend/agent/src/ai/import_boundary_test.go",
			"backend/agent/src/ai/openaicompat/openrouter/zero_requires_test.go",
		}
		if strings.Contains(aiDiff, "diff --git") {
			chunks := strings.Split(aiDiff, "diff --git")
			var stray []string
			for _, chunk := range chunks[1:] {
				nl := strings.Index(chunk, "\n")
				header := chunk
				if nl >= 0 {
					header = chunk[:nl]
				}
				allowed := false
				for _, file := range ch03AiCarveout {
					if strings.Contains(header, file) {
						allowed = true
						break
					}
				}
				if !allowed {
					stray = append(stray, chunk)
				}
			}
			if len(stray) == 0 {
				t.Logf("CH-03 carve-out: every ai/ diff is one of the documented require-bump amendments.\n%s", aiDiff)
			} else {
				t.Errorf("backend/agent/src/ai/ was edited beyond the CH-03 wantGoModRequires amendments:\n%s", aiDiff)
			}
		} else if aiDiff != "" {
			t.Errorf("backend/agent/src/ai/ diff against %s is not empty:\n%s", baseRef, aiDiff)
		}
		modDiff, merr := gitDiff(t, root, baseRef, "backend/agent/go.mod", "backend/agent/go.sum")
		if merr != nil {
			t.Fatalf("git diff %s -- go.mod go.sum failed: %v", baseRef, merr)
		}
		// CH-03 carve-out for go.mod / go.sum drift: chat
		// archetype's HTTP+SSE surface requires
		// github.com/labstack/echo/v5 v5.2.1 (ADR
		// adr/echo-v5-in-agent-module). Recorded exception;
		// every other modification fails.
		if isCH03GoModDrift(modDiff) {
			t.Logf("CH-03 carve-out: go.mod drift is the recorded Echo require; passes per D3.\n%s", modDiff)
		} else if modDiff != "" {
			t.Errorf("go.mod/go.sum diff against %s is not empty:\n%s", baseRef, modDiff)
		}
	})

	t.Run("no_hook_registered_no_new_wall_clock", func(t *testing.T) {
		t.Parallel()

		root, err := gitTopLevel(t)
		if err != nil {
			t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
		}
		forbidden := []string{"time.After", "time.Sleep", "time.NewTimer", "time.NewTicker", "context.WithTimeout", "context.WithDeadline"}

		// harness.go, loop.go, compaction.go and observability.go must
		// carry NONE of these — their only clock is the ctx cancellation
		// chain. scheduler.go is excluded from this direct scan: it
		// legitimately carries the PRE-EXISTING (AG-14) injected wind-down
		// bound's own time.NewTimer, used strictly as a ceiling
		// (NFR-CNH-002's carve-out) — checked separately below, over the
		// DIFF only, so a genuinely NEW wall-clock use would still be
		// caught.
		for _, name := range []string{"harness.go", "loop.go", "compaction.go", "observability.go"} {
			raw, rerr := os.ReadFile(root + "/backend/agent/src/agent/" + name)
			if rerr != nil {
				t.Fatalf("os.ReadFile(%s) failed: %v", name, rerr)
			}
			src := string(raw)
			for _, f := range forbidden {
				if strings.Contains(src, f) {
					t.Errorf("%s contains %q, want no wall clock of any kind — AG-22 adds no timeout, deadline, sleep or poll of its own", name, f)
				}
			}
			if strings.Contains(src, `"time"`) {
				t.Errorf(`%s imports "time", want no wall clock anywhere`, name)
			}
		}

		baseRef := hksResolveBaseRef(t, root)
		schedDiff, derr := gitDiff(t, root, baseRef, "backend/agent/src/agent/scheduler.go")
		if derr != nil {
			t.Fatalf("git diff %s -- scheduler.go failed: %v", baseRef, derr)
		}
		for _, line := range strings.Split(schedDiff, "\n") {
			if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
				continue
			}
			for _, f := range forbidden {
				if strings.Contains(line, f) {
					t.Errorf("scheduler.go's own diff against %s adds a NEW %q — the only permitted clock is the pre-existing injected wind-down bound, used as a ceiling, never a new synchronization point:\n%s", baseRef, f, line)
				}
			}
		}
	})
}
