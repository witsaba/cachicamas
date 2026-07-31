// Package domain_test — architectural invariant tests.
//
// Spec S-PR-X5: the domain package MUST NOT import github.com/jackc/pgx.
// pgx stays inside infrastructure/postgres/. This test shells out to
// `go list -deps` and asserts the dependency graph is clean.
//
// AI-00.4 adds the reverse boundary guard — the half of ADR 0005 § D1 that Go
// cannot enforce for free. Go makes a module cycle a build error, so
// backend/agent importing this module is already impossible; nothing stops this
// module from importing backend/agent, which is what these two tests cover.
//
// # Why the module-scope test lives in the domain package
//
// TestModule_OnlyApplicationAndCmdServerMayImportAgentModule is wider in scope
// than the package that hosts it: it inspects every package of the module, not
// just domain. It is here because architectural invariant tests already live in
// this file, and because a dedicated src/architecture/ package would need a
// non-test Go file purely to keep `go build ./...` working — a file with no
// reason to exist beyond satisfying the toolchain. Moving it later is a
// mechanical rename with no behavioral consequence.
package domain_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// agentModulePath is the module ADR 0005 promotes the agent stack into. No
// package of this module imports it today, so both guards below are green from
// birth — and that is precisely the point. They fail on the first accidental
// import rather than on the first production incident.
const agentModulePath = "github.com/cachicamas/backend/agent"

// dbaModulePath scopes the module-wide scan.
const dbaModulePath = "github.com/cachicamas/backend/database_administrator"

// agentImportersAllowlist is the exact set of package prefixes that ADR 0005
// § D1 row 5 permits to import the agent module. It is two entries, and it must
// stay two entries — see the comment in
// TestModule_OnlyApplicationAndCmdServerMayImportAgentModule for why widening it
// is the wrong response to a failure.
//
// Row 5 is a permitted direction, not an exercised one: no package imports the
// agent module in v1, and doing so breaks `docker compose build` until the
// compose build context and Dockerfile COPY paths are reworked. ADR 0005
// § "Row 5 is the honest answer to S1" prices that as its own change.
var agentImportersAllowlist = []string{
	dbaModulePath + "/src/application",
	dbaModulePath + "/src/cmd/server",
}

func TestDomainLayer_DoesNotImportPgx(t *testing.T) {
	t.Parallel()
	out, err := runGoListDeps("github.com/cachicamas/backend/database_administrator/src/domain")
	if err != nil {
		t.Fatalf("go list -deps failed: %v", err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "github.com/jackc/pgx") {
			t.Errorf("domain package must not import pgx; found: %q", line)
		}
	}
}

// TestDomainLayer_DoesNotImportAgentModule is the first half of ADR 0005
// § Guard B, and the narrower one.
//
// ADR 0005 § D1 row 6: the domain package must not reach the agent module by any
// path. The scan is TRANSITIVE (`go list -deps`) because domain is the bottom of
// this module's dependency graph — nothing imports it downward — so a transitive
// hit here is always a real violation and never collateral from someone else's
// permitted import.
//
// This is the opposite of the module-scope test below, and the difference is
// deliberate. See its comment.
func TestDomainLayer_DoesNotImportAgentModule(t *testing.T) {
	t.Parallel()
	out, err := runGoListDeps(dbaModulePath + "/src/domain")
	if err != nil {
		t.Fatalf("go list -deps failed: %v", err)
	}
	for _, line := range strings.Split(out, "\n") {
		dep := strings.TrimSpace(line)
		if dep == agentModulePath || strings.HasPrefix(dep, agentModulePath+"/") {
			t.Errorf("the domain package must not import the agent module, by any path; found: %q\n"+
				"  rule: ADR 0005 § D1 row 6 — domain's permitted imports are unchanged by the "+
				"agent module's existence.", dep)
		}
	}
}

// TestModule_OnlyApplicationAndCmdServerMayImportAgentModule is the second half
// of ADR 0005 § Guard B: the module-scope assertion that never existed before.
//
// # This scan is DIRECT-IMPORTS-ONLY, and that is not an oversight
//
// It deliberately does not pass -deps, and a future maintainer who sees this
// test fail and reaches for -deps must read this first.
//
// ADR 0005 § D1 row 5 permits src/application and src/cmd/server to import the
// agent module. The moment row 5 is exercised, every package that imports
// application inherits the agent module transitively — so a transitive scan
// would report interfaces/http, infrastructure/... and others as violators when
// they did nothing wrong. The path of least resistance at that point is to add
// each one to the allowlist, and a few iterations later the allowlist names
// every package in the module and the guard means nothing.
//
// Direct imports keep the assertion honest: it asks "does this package name the
// agent module in its own source", which is exactly the question row 5 answers,
// and its answer does not change when a permitted importer starts importing.
//
// TestImports and XTestImports are included because a test file is source too;
// omitting them is one of the two blind spots ADR 0005 § Guard A records against
// the approach this replaces.
func TestModule_OnlyApplicationAndCmdServerMayImportAgentModule(t *testing.T) {
	t.Parallel()

	const format = "{{.ImportPath}}|{{join .Imports \" \"}}|{{join .TestImports \" \"}}|{{join .XTestImports \" \"}}"
	out, err := runGoList("-f", format, dbaModulePath+"/...")
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	inspected := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) != 4 {
			t.Fatalf("unexpected go list output shape: %q", line)
		}
		pkg := fields[0]
		inspected++

		if isPermittedAgentImporter(pkg) {
			continue
		}

		for _, group := range fields[1:] {
			for _, imp := range strings.Fields(group) {
				if imp == agentModulePath || strings.HasPrefix(imp, agentModulePath+"/") {
					t.Errorf("package %q must not import the agent module; found: %q\n"+
						"  rule: ADR 0005 § D1 row 5 — only %v may import %s.\n"+
						"  If this import is intentional, the fix is NOT to add this package to "+
						"the allowlist: route the dependency through src/application instead. "+
						"Widening the allowlist is how this guard stops meaning anything.\n"+
						"  Note also that exercising row 5 at all breaks `docker compose build` "+
						"until the compose build context and Dockerfile COPY paths are reworked "+
						"(ADR 0005 prices this as its own change).",
						pkg, imp, agentImportersAllowlist, agentModulePath)
				}
			}
		}
	}

	if inspected == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the module pattern still resolves: " + dbaModulePath + "/...")
	}
	t.Logf("inspected %d packages of %s for direct imports of %s", inspected, dbaModulePath, agentModulePath)
}

func isPermittedAgentImporter(pkg string) bool {
	for _, allowed := range agentImportersAllowlist {
		if pkg == allowed || strings.HasPrefix(pkg, allowed+"/") {
			return true
		}
	}
	return false
}

// runGoListDeps runs `go list -deps` on the given import path and
// returns stdout. The helper is in the test file (not in a regular
// .go file) because no production code path should depend on it.
func runGoListDeps(importPath string) (string, error) {
	return runGoList("-deps", importPath)
}

// runGoList runs `go list` with the given arguments and returns stdout.
// Stderr is folded into the returned error, because `go list` reports the
// interesting part of a failure there and a bare exit status is not actionable.
func runGoList(args ...string) (string, error) {
	cmd := exec.Command("go", append([]string{"list"}, args...)...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if s := strings.TrimSpace(stderr.String()); s != "" {
			return "", fmt.Errorf("%w: %s", err, s)
		}
		return "", err
	}
	return stdout.String(), nil
}
