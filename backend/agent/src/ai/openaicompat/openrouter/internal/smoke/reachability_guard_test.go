// AI-39 work unit D (design.md D4) — the module-wide reachability guard:
// mechanical, regression-detected evidence that the live-smoke package is
// unreachable from any application entry point (R-LSM-005), following the
// AI-00.3 import_boundary_test.go pattern this module already ships
// (design.md "Verified code facts").
//
// # Three pieces
//
//  1. TestSmokePathContainsInternal — a structural pin: go list must
//     resolve the smoke package's import path (Fatal if it cannot —
//     rename protection), and the resolved path must contain an
//     "/internal/" segment.
//  2. TestSmokeReachability_DenyByDefault — a closure check, deny by
//     default, over the whole module's test-inclusive package
//     dependency graph: no package outside the
//     .../openaicompat/openrouter/ subtree may carry the smoke import
//     path in its transitive dependencies. It fails loudly — rather
//     than passing vacuously — both when its own package pattern
//     resolves to nothing, and when the smoke path cannot be found
//     anywhere at all (including the smoke package's own test
//     variants), because either shape means the enumeration itself is
//     broken and cannot prove anything.
//  3. TestSmokeUnreachableFromOutsideSubtree — a compile-negative
//     fixture: internal_import_probe (testdata/, so `./...` never
//     builds it) imports the smoke path from outside the subtree; this
//     test asserts the build fails, and fails specifically on Go's own
//     import-visibility rule, not for an unrelated reason.
//
// # Zero composition roots today (S-LSM-020)
//
// This module has no `package main` of its own anywhere yet — doc 0002
// records "No CI exists" and ADR 0005's composition root has not landed.
// A literal reachability walk starting from a composition root is
// therefore vacuous today: there is nothing to walk from. This guard's
// value is failing the day an in-module importer appears outside the
// provider-adapter subtree — a deny-by-default check over the module's
// whole package closure, not a walk from a root that does not exist yet.
// This is a deliberate, permanent limitation of what this file proves
// TODAY, stated here rather than worked around with a skipped
// placeholder test — R-LSM-005 forbids using one in this guard's place.
package smoke_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// modulePathForGuard is this module — the only prefix under which an
// importer of the smoke path is legal at all.
const modulePathForGuard = "github.com/cachicamas/backend/agent"

// smokeImportPathForGuard is the merged live-smoke package's import
// path — the single source of truth both guard pieces below check
// against, so a future rename only needs updating in one place.
const smokeImportPathForGuard = modulePathForGuard + "/src/ai/openaicompat/openrouter/internal/smoke"

// allowedSmokeImporterPrefix is the one subtree the smoke package's own
// directory placement already restricts importers to (D-placement,
// design.md): only a package at or under this prefix may legally import
// the smoke path at all — Go's own internal-package visibility rule
// enforces this at compile time (proven by
// TestSmokeUnreachableFromOutsideSubtree below); this guard restates it
// as an independent, mechanically-checked fact over the whole module.
const allowedSmokeImporterPrefix = modulePathForGuard + "/src/ai/openaicompat/openrouter"

// guardPackagePattern scopes the closure check to the whole module —
// fully qualified, matching import_boundary_test.go's own reasoning: go
// test sets the working directory to the package under test, so a
// relative ./... pattern would silently narrow the guard to this one
// package alone.
const guardPackagePattern = modulePathForGuard + "/..."

// internalImportProbePackage is the compile-negative fixture's import
// path (D.3): outside the allowed subtree, under testdata/ so `./...`
// never builds or tests it on its own.
const internalImportProbePackage = modulePathForGuard + "/src/ai/openaicompat/testdata/internal_import_probe"

// TestSmokePathContainsInternal is the structural pin (design.md D4
// piece 1, S-LSM-016): go list must resolve the merged smoke import
// path — Fatal if it cannot (rename protection) — and the resolved path
// must contain an "/internal/" segment.
func TestSmokePathContainsInternal(t *testing.T) {
	resolved, err := goListImportPath(smokeImportPathForGuard)
	if err != nil {
		t.Fatalf("go list %s: %v — the smoke package's import path no longer resolves; update smokeImportPathForGuard if it was intentionally renamed or moved (S-LSM-016)", smokeImportPathForGuard, err)
	}
	if !strings.Contains(resolved, "/internal/") {
		t.Fatalf("smoke import path %q does not contain an /internal/ segment, want one so only the provider-adapter subtree may import it (R-LSM-005, S-LSM-016)", resolved)
	}
}

// TestSmokeReachability_DenyByDefault is the closure check (design.md D4
// piece 2, S-LSM-018, S-LSM-019): over the module's whole test-inclusive
// package dependency graph, no package outside allowedSmokeImporterPrefix
// may carry the smoke path in its transitive dependencies.
func TestSmokeReachability_DenyByDefault(t *testing.T) {
	pkgs, err := goListTestJSON(guardPackagePattern)
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the module pattern still resolves: " + guardPackagePattern)
	}

	smokeSeenAnywhere := false
	for _, pkg := range pkgs {
		if !depsContainSmoke(pkg.Deps) {
			continue
		}
		smokeSeenAnywhere = true

		importer := normalizeGuardImportPath(pkg.ImportPath)
		if !allowedSmokeImporter(importer) {
			t.Errorf("%s imports the smoke path but is not under %s (R-LSM-005, S-LSM-018)", importer, allowedSmokeImporterPrefix)
		}
	}
	if !smokeSeenAnywhere {
		t.Fatal("the smoke import path was not found in any package's dependency set anywhere in the module, including the smoke package's own test variants — the guard's own enumeration is broken and cannot prove anything (anti-vacuity, S-LSM-019)")
	}
}

// TestSmokeUnreachableFromOutsideSubtree is the compile-negative fixture
// check (design.md D4 piece 3, S-LSM-017): building
// internal_import_probe — outside allowedSmokeImporterPrefix — MUST
// fail, and MUST fail specifically because of Go's own internal-package
// import-visibility rule, not for an unrelated reason (a typo, a missing
// dependency, and so on).
func TestSmokeUnreachableFromOutsideSubtree(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", os.DevNull, internalImportProbePackage)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go build %s succeeded, want it to fail on import visibility (R-LSM-005, S-LSM-017)", internalImportProbePackage)
	}
	if !strings.Contains(string(out), "use of internal package") {
		t.Fatalf("go build %s failed for the wrong reason — output: %s\nwant it to fail specifically on Go's internal-package import-visibility rule (S-LSM-017)", internalImportProbePackage, out)
	}
}

// goListImportPath resolves pattern via `go list`, returning its
// ImportPath (which, for an exact, unambiguous package pattern, is
// pattern itself echoed back) or an error naming why it did not resolve.
func goListImportPath(pattern string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", pattern)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &goListGuardError{err: err, output: stderr.String()}
	}
	return strings.TrimSpace(stdout.String()), nil
}

// smokePackageJSON is the subset of `go list -json`'s package object
// this guard reads. JSON, not a text `-f` template, is used deliberately
// (a documented, justified deviation from design.md's `-f
// '{{.ImportPath}} {{join .Deps " "}}'` sketch): go list's own
// -test-synthesized ImportPath values (e.g. `pkg_test [pkg.test]`)
// contain an embedded space, which a single-line `{{.ImportPath}}
// {{join .Deps " "}}` template cannot be split back apart from
// unambiguously — the first space no longer reliably separates the
// import path from its deps list. JSON keeps ImportPath and Deps as
// distinct fields, so this parsing ambiguity cannot arise.
type smokePackageJSON struct {
	ImportPath string
	Deps       []string
}

// goListTestJSON runs `go list -test -json` over pattern and decodes the
// resulting stream of package objects (go list -json emits one JSON
// object per matched package, back to back, not a JSON array).
func goListTestJSON(pattern string) ([]smokePackageJSON, error) {
	cmd := exec.Command("go", "list", "-test", "-json", pattern)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, &goListGuardError{err: err, output: stderr.String()}
	}

	var pkgs []smokePackageJSON
	dec := json.NewDecoder(&stdout)
	for dec.More() {
		var pkg smokePackageJSON
		if err := dec.Decode(&pkg); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

// depsContainSmoke reports whether deps carries the smoke import path,
// normalizing each entry first so a hypothetical bracket-suffixed
// dependency edge (go list's own -test synthesis shape) cannot hide a
// real hit from this check.
func depsContainSmoke(deps []string) bool {
	for _, dep := range deps {
		if normalizeGuardImportPath(dep) == smokeImportPathForGuard {
			return true
		}
	}
	return false
}

// normalizeGuardImportPath strips a `go list -test`-synthesized " [...]"
// suffix (e.g. "pkg_test [pkg.test]" -> "pkg_test"), mirroring
// import_boundary_test.go's normalizeListedPackage for the same reason:
// left unnormalized, a synthesized entry's decorated name would never
// match a plain prefix or equality check.
func normalizeGuardImportPath(importPath string) string {
	if i := strings.Index(importPath, " ["); i >= 0 {
		return importPath[:i]
	}
	return importPath
}

// allowedSmokeImporter reports whether importPath sits at or under
// allowedSmokeImporterPrefix.
func allowedSmokeImporter(importPath string) bool {
	return importPath == allowedSmokeImporterPrefix || strings.HasPrefix(importPath, allowedSmokeImporterPrefix+"/")
}

// goListGuardError pairs a go list execution error with the command's
// stderr, mirroring import_boundary_test.go's goListError so a failure
// here names the same kind of actionable detail.
type goListGuardError struct {
	err    error
	output string
}

func (e *goListGuardError) Error() string {
	if e.output == "" {
		return e.err.Error()
	}
	return e.err.Error() + ": " + strings.TrimSpace(e.output)
}
