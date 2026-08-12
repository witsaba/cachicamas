// AG-03.2 — the forward boundary guard: Layer 2 import purity over both
// closures (R-AGP-003, design AD-3).
//
// This guard enforces ADR 0005 § D1 row 2 mechanically, retargeting
// ai_test's import_boundary_test.go mechanism (AI-00.3) from Layer 1 to
// Layer 2. Two structural differences from that mechanism, both AD-3:
//
//  1. THREE checks, not one merged scan: the production closure
//     ([TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault]),
//     the test closure
//     ([TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction]),
//     and a network/filesystem closure scan that includes the standard
//     library
//     ([TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage],
//     AI-10.4's pattern). Production and test are asserted SEPARATELY —
//     not L1's single `-deps -test` merge — because S-AGP-026 requires
//     proving the production closure never admits the test substrate
//     (src/agenttest) while the test closure does.
//  2. Why check 3 exists at all: "net/http" is standard library, so
//     [listNonStdlibDeps] filters it out before checks 1 and 2 ever see
//     it — a bare-prefix "denied by name" row for it in forbiddenPrefixes
//     would be unreachable dead code. Check 3 uses
//     [listAllProductionDeps], which does NOT filter the standard
//     library, and denies by exact path instead. Production-only, for
//     AI-10.4's own recorded reason: `-test` pulls in "testing", which
//     imports "os" itself, so a test-inclusive scan could never pass.
//
// # Zero OpenTelemetry grant (AD-2, settled decision 5) — and why
// allowedProductionPrefixes has no OTel entry at all
//
// AG-03 has zero event, loop or harness behavior (the milestone's own
// charter), so Layer 2 has zero OpenTelemetry usage to admit on its own
// account: no root go.opentelemetry.io/otel, no otel/metric, no § D3
// grant of any kind.
//
// design.md's own Interfaces/Contracts table additionally proposed five
// OTel/xxhash entries, reasoned as "src/ai's measured forced closure" —
// on the theory that bite 4's green half (a test file importing
// src/agenttest) reaches src/ai and therefore OpenTelemetry. A fresh
// measurement taken during apply (recorded in apply-progress.md,
// R-AGP-004) disproves that theory:
//
//	go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai
//
// (the bare src/ai package, no -test) returns the EMPTY set — src/ai's
// own production files import only the standard library.
// OpenTelemetry is imported exclusively by src/ai/openaicompat/** and
// its openrouter subpackage — a subtree this guard forbids BY NAME,
// below — never by src/ai itself. src/agenttest's own bare closure
// (also freshly measured) is {src/agenttest, src/agenttest/sweep,
// src/ai} — equally OTel-free. So neither this guard's production
// closure (src/agent, src/ai) nor its widened test closure (+
// src/agenttest) ever actually reaches OpenTelemetry.
//
// Per this milestone's explicit instruction — an OTel-prefixed entry
// belongs in this allowlist ONLY if the fresh measurement shows it
// forced transitively, never as a speculative grant — none appear here.
// This is a deliberate, evidence-driven deviation from design.md's
// literal table, reported in apply-progress.md rather than silently
// applied. The first Layer 2 milestone with real OpenTelemetry usage
// adds its § D3 paths here, in its own PR, under its own justification —
// exactly the precedent AI-37 set for Layer 1's own empty-at-birth
// third-party group.
package agent_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// modulePath is this module, and the only non-stdlib prefix the allowlist
// admits by way of its own two entries below.
const modulePath = "github.com/cachicamas/backend/agent"

// layer2Pattern scopes the scan to Layer 2's own package tree. Fully
// qualified on purpose — see ai_test's import_boundary_test.go for why a
// relative ./... pattern is unsafe.
const layer2Pattern = modulePath + "/src/agent/..."

// forbiddenPrefixes are checked BEFORE the allowlist, mirroring
// ai_test's import_boundary_test.go — and, as there, the order is
// load-bearing for one of the two reasons the guard needs a forbidden
// list at all:
//
//  1. The vendor adapter subtree, ".../src/ai/openaicompat", begins with
//     the allowed Layer 1 prefix ".../src/ai" (allowedProductionPrefixes,
//     below) — an allowlist-first pass would admit it. Checking this
//     table first is what carves it back out (S-AGP-029).
//  2. "net/http" is standard library, so it never reaches THIS table's
//     match at all — [listNonStdlibDeps] filters the standard library
//     out before either check 1 or check 2 runs. A "denied by name" row
//     for it here would be unreachable dead code; check 3
//     ([TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage])
//     is the actual, reachable mechanism that denies it, using
//     networkOrFilesystemPackages below instead of this table.
//
// Each entry carries the rule it enforces, because a guard failure
// should tell the reader which decision they are contradicting, not
// merely that they are.
//
// Deliberately absent, denied by default rather than named: ".../src/handoff"
// (Layer 1's own consumer proof — nothing in Layer 2 has any reason to
// import it) and root "go.opentelemetry.io/otel" (no OTel grant at all,
// see the package comment). ".../src/ai/internal/*" needs no row: Go's
// own `internal` visibility rule already forbids importing it from
// outside src/ai, before this guard ever runs.
var forbiddenPrefixes = []struct {
	prefix string
	rule   string
}{
	{"github.com/cachicamas/backend/database_administrator", "ADR 0005 § D1 row 2: no package of another backend module"},
	{"github.com/cachicamas/backend/workspace_syncer", "ADR 0005 § D1 row 2: no package of another backend module"},
	{modulePath + "/src/coding", "ADR 0005 § D1 row 2: Layer 2 must not import Layer 3"},
	{modulePath + "/src/cmd", "ADR 0005 § D1 row 2: Layer 2 must not import the composition root"},
	{modulePath + "/src/ai/openaicompat", "AG-03.2: the vendor adapter subtree is denied by name — Layer 2 speaks to providers only through the src/ai contract"},
	{"go.opentelemetry.io/otel/sdk", "ADR 0005 § D3: the OTel SDK belongs to a composition root"},
	{"go.opentelemetry.io/otel/exporters", "ADR 0005 § D3: OTel exporters belong to a composition root"},
	{"go.opentelemetry.io/contrib/bridges/otelslog", "ADR 0005 § D3: otelslog is permitted no lower than Layer 3"},
}

// allowedProductionPrefixes is check 1's deny-by-default allowlist, minus
// the standard library (filtered by the toolchain itself — see
// listNonStdlibDeps). See the package comment for why no OpenTelemetry or
// xxhash entry appears here, unlike ai_test's own allowlist.
var allowedProductionPrefixes = []string{
	modulePath + "/src/agent",
	modulePath + "/src/ai", // the Layer 1 contract package; openaicompat carved back out by the forbidden row above
}

// allowedTestPrefixes is check 2's allowlist: the production set plus the
// test substrate, src/agenttest — asserted as its own separate scan (AD-3)
// so the production closure can be proven, independently, to never admit
// it (S-AGP-025, S-AGP-026). The prefix match below covers agenttest's own
// subpackages (for example agenttest/sweep) without a separate entry.
var allowedTestPrefixes = append(slices.Clone(allowedProductionPrefixes), modulePath+"/src/agenttest")

// networkOrFilesystemPackages is check 3's own list — mirroring
// ai_test's TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage
// (AI-10.4). Exact-path match, over a closure that includes the standard
// library (see [listAllProductionDeps]) — the opposite filtering choice
// from checks 1 and 2, and the reason this table exists at all.
var networkOrFilesystemPackages = []struct {
	path string
	rule string
}{
	{"net", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no network package"},
	{"net/http", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no network package"},
	{"os", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no filesystem/environment package"},
	{"io/fs", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no filesystem package"},
}

// TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault is
// check 1 (AD-3): Layer 2's PRODUCTION dependency closure — no test files —
// admits only the standard library and allowedProductionPrefixes.
func TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault(t *testing.T) {
	t.Parallel()

	deps, err := listNonStdlibDeps(layer2Pattern, false)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the package pattern still resolves: " + layer2Pattern)
	}

	for _, dep := range deps {
		// Forbidden first. See forbiddenPrefixes' comment: an
		// allowlist-first pass would admit the vendor adapter subtree.
		if rule, forbidden := matchForbidden(dep); forbidden {
			t.Errorf("Layer 2's production closure must not import %q\n  rule: %s", dep, rule)
			continue
		}
		if !isAllowed(dep, allowedProductionPrefixes) {
			t.Errorf("Layer 2's production closure must not import %q\n"+
				"  rule: deny-by-default allowlist (ADR 0005 § D1 row 2) — this path is neither "+
				"the Go standard library nor a package this guard's production allowlist admits.\n"+
				"  No forbidden prefix names it, and that is not a licence to add it: adding a "+
				"dependency needs its own recorded design decision in allowedProductionPrefixes.",
				dep)
		}
	}
}

// TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction is
// check 2 (AD-3): Layer 2's TEST dependency closure admits everything the
// production closure admits, plus src/agenttest — scanned separately from
// check 1 so the production closure can be proven never to see it
// (S-AGP-025, S-AGP-026).
func TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction(t *testing.T) {
	t.Parallel()

	deps, err := listNonStdlibDeps(layer2Pattern, true)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list -test returned no packages; the guard would pass vacuously. " +
			"Check that the package pattern still resolves: " + layer2Pattern)
	}

	for _, dep := range deps {
		if rule, forbidden := matchForbidden(dep); forbidden {
			t.Errorf("Layer 2's test closure must not import %q\n  rule: %s", dep, rule)
			continue
		}
		if !isAllowed(dep, allowedTestPrefixes) {
			t.Errorf("Layer 2's test closure must not import %q\n"+
				"  rule: deny-by-default allowlist (ADR 0005 § D1 row 2) — this path is neither "+
				"the Go standard library, a production-allowed package, nor the test substrate "+
				"(src/agenttest).\n"+
				"  No forbidden prefix names it, and that is not a licence to add it.",
				dep)
		}
	}
}

// TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage is
// check 3 (AD-3, AI-10.4's pattern, ai_test's
// TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage):
// "net", "net/http", "os" and "io/fs" are standard library, so checks 1
// and 2 never see them — [listNonStdlibDeps] filters the standard library
// out before either allowlist ever runs. This is the narrower,
// complementary claim: not "nothing foreign", but "nothing that reaches a
// socket or a filesystem", checked against the standard library too.
// Production-only, for AI-10.4's own recorded reason: -test pulls in
// "testing", which imports "os" itself, so a test-inclusive scan could
// never pass.
func TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage(t *testing.T) {
	t.Parallel()

	deps, err := listAllProductionDeps(layer2Pattern)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the package pattern still resolves: " + layer2Pattern)
	}

	for _, forbidden := range networkOrFilesystemPackages {
		if slices.Contains(deps, forbidden.path) {
			t.Errorf("Layer 2's production closure imports %q\n  rule: %s", forbidden.path, forbidden.rule)
		}
	}
}

func matchForbidden(importPath string) (rule string, forbidden bool) {
	for _, f := range forbiddenPrefixes {
		if importPath == f.prefix || strings.HasPrefix(importPath, f.prefix+"/") {
			return f.rule, true
		}
	}
	return "", false
}

// isAllowed reports whether importPath is admitted by allowlist. A
// trailing "_test" is trimmed and retried as a second attempt: unlike
// ai_test's import_boundary_test.go, whose single bare-modulePath
// allowlist entry incidentally covers everything module-wide, this
// guard's allowlist is curated per layer, so the "<pkg>_test" shape
// go list -deps -test synthesizes for a scanned package's own EXTERNAL
// test files (never a [pkg.test]-bracketed form — normalizeListedPackage
// has already stripped that) would otherwise be measured as a foreign
// import. It never is one: it can only ever be produced for a package
// actually inside layer2Pattern's own tree, so a package already admitted
// by allowlist has its own external test package admitted too, without a
// separate entry per package the scanned pattern reaches — including a
// future subpackage of src/agent this milestone does not yet have.
func isAllowed(importPath string, allowlist []string) bool {
	if matchesPrefix(importPath, allowlist) {
		return true
	}
	if base, ok := strings.CutSuffix(importPath, "_test"); ok {
		return matchesPrefix(base, allowlist)
	}
	return false
}

func matchesPrefix(importPath string, allowlist []string) bool {
	for _, prefix := range allowlist {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

// listNonStdlibDeps returns the deduplicated non-stdlib transitive
// dependency set of pattern. includeTest selects between the production
// closure (bare `go list -deps`) and the test closure (`go list -deps
// -test`) — two SEPARATE checks (AD-3), not ai_test's single merged
// `-deps -test` scan, because S-AGP-026 requires the production and test
// postures to be provably independent.
//
// The standard library is filtered by the toolchain via `.Standard`
// rather than a list maintained here — ai_test's import_boundary_test.go
// package comment records why the obvious heuristic is wrong.
func listNonStdlibDeps(pattern string, includeTest bool) ([]string, error) {
	args := []string{"list", "-deps"}
	if includeTest {
		args = append(args, "-test")
	}
	args = append(args, "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", pattern)

	cmd := exec.Command("go", args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, &goListError{err: err, stderr: stderr.String()}
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, 16)
	for _, line := range strings.Split(stdout.String(), "\n") {
		path, ok := normalizeListedPackage(line)
		if !ok {
			continue
		}
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out, nil
}

// listAllProductionDeps returns every import path in pattern's PRODUCTION
// dependency closure, the standard library included and test imports
// excluded — check 3's own input (AD-3): "net/http" is standard library,
// so it must NOT be filtered the way listNonStdlibDeps filters it, and
// production-only because `-test` would pull in "testing", which imports
// "os" itself.
func listAllProductionDeps(pattern string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", pattern)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, &goListError{err: err, stderr: stderr.String()}
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, 32)
	for _, line := range strings.Split(stdout.String(), "\n") {
		path, ok := normalizeListedPackage(line)
		if !ok {
			continue
		}
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out, nil
}

// normalizeListedPackage cleans one line of `go list -deps [-test]`
// output. Ported verbatim from ai_test's import_boundary_test.go (AD-3):
// the -test flag makes go list emit three synthesized shapes that are
// artifacts of how it models test binaries, not real dependencies. Left
// unnormalized, each would be measured against the allowlist and the
// guard would fail on its own module:
//
//	"pkg [pkg.test]"       the package as compiled into a test binary
//	"pkg_test [pkg.test]"  an external test package
//	"pkg.test"             the synthesized test-binary main package
//
// The first two are stripped back to the real import path. The third is
// dropped: it names no importable package.
func normalizeListedPackage(line string) (string, bool) {
	path := strings.TrimSpace(line)
	if path == "" {
		return "", false
	}
	if i := strings.Index(path, " ["); i >= 0 {
		path = path[:i]
	}
	if strings.HasSuffix(path, ".test") {
		return "", false
	}
	return path, true
}

type goListError struct {
	err    error
	stderr string
}

func (e *goListError) Error() string {
	if e.stderr == "" {
		return e.err.Error()
	}
	return e.err.Error() + ": " + strings.TrimSpace(e.stderr)
}
