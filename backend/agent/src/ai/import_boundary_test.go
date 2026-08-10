// AI-00.3 — the forward boundary guard: Layer 1 import purity.
//
// This guard enforces ADR 0005 § D1 row 1 and § D3 mechanically, so that the
// import rule is a build failure rather than a review convention.
//
// # Why it is deny-by-default
//
// The allowlist admits the Go standard library and this module's own packages,
// and nothing else. A dependency that is neither — one that no forbidden prefix
// happens to name — still fails. That is the whole design: it makes the
// transport choice at AI-24 a visible, ADR-gated event rather than a quiet
// `go get`, and it means this guard does not need to be updated every time a new
// undesirable dependency is invented.
//
// The forbidden-prefix table is therefore belt-and-braces, not the mechanism. It
// exists to produce a specific, actionable failure message for the violations we
// can predict, and to pin the § D3 split — OpenTelemetry API allowed, SDK and
// exporters and otelslog forbidden — where a reader will find it.
//
// # Two implementation details that are load-bearing
//
// `go list -deps -test`, not bare `-deps`. ADR 0005 § Guard A requires this guard
// to close the two blind spots of the `.Imports`-only approach it replaces: test
// imports and transitive dependencies. Bare `-deps` closes only the second.
// Measured on this repository, `go list -deps` over a package with an external
// test package reports 2 non-stdlib entries where `go list -deps -test` reports
// 5. A Layer 1 *test file* importing a sibling backend module would pass a bare
// `-deps` guard — precisely the failure this guard exists to prevent.
//
// `{{if not .Standard}}`, not a "first path segment contains no dot" heuristic.
// `go list std` contains vendored paths such as vendor/golang.org/x/crypto/...
// which appear verbatim in real dependency output; the heuristic misclassifies
// every one of them as third-party. `.Standard` is the toolchain's own
// classification and is correct by construction.
package ai_test

import (
	"encoding/json"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// modulePath is this module, and the only non-stdlib prefix the allowlist admits.
const modulePath = "github.com/cachicamas/backend/agent"

// layer1Pattern scopes the scan. It is fully qualified on purpose: `go test` runs
// with the working directory set to the package under test, so a relative
// pattern such as ./... would silently narrow the guard to src/ai alone and stop
// covering src/agenttest.
const layer1Pattern = modulePath + "/..."

// requestPackagePath scopes AI-10.4 item 3's guard, below. It names the one
// package request.go lives in rather than layer1Pattern, because the claim
// under test — "validation performs no I/O" — is about the request path
// specifically, not about Layer 1 as a whole.
const requestPackagePath = modulePath + "/src/ai"

// forbiddenPrefixes are checked BEFORE the allowlist, and the order matters.
// ".../agent/src/agent" begins with the allowed own-module prefix
// ".../agent", so an allowlist-first pass would admit Layer 2 and this guard
// would enforce nothing about the sibling layers.
//
// Each entry carries the rule it enforces, because a guard failure should tell
// the reader which decision they are contradicting, not merely that they are.
var forbiddenPrefixes = []struct {
	prefix string
	rule   string
}{
	// ADR 0005 § D1 row 1 — the agent module imports no other backend module,
	// in any package. Go enforces this for a cycle; these two catch the
	// one-way import that is not a cycle.
	{"github.com/cachicamas/backend/database_administrator", "ADR 0005 § D1 row 1: no package of another backend module"},
	{"github.com/cachicamas/backend/workspace_syncer", "ADR 0005 § D1 row 1: no package of another backend module"},

	// ADR 0005 § D1 row 1 — the sibling layers. Dependencies flow
	// coding -> agent -> ai and never upward. These three directories do not
	// exist yet; naming them now is what makes the rule testable on the day
	// the first one is created.
	//
	// KNOWN HAZARD for that day (recorded 2026-08-10, before doc 0003's
	// AG milestones create src/agent): layer1Pattern scans the WHOLE
	// module, and `go list -deps -test` emits the pattern's own member
	// packages — so merely CREATING backend/agent/src/agent makes the
	// new package appear in the scanned set and match this forbidden
	// prefix, failing the guard with zero actual import violations. The
	// deliberate fix when it fires is to narrow the scanned roots to
	// Layer 1's own packages (src/ai/..., src/agenttest/..., src/handoff)
	// or to exempt the pattern's own members from the prefix match —
	// decided then, in the change that creates the directory, never by
	// deleting the prefix row.
	{modulePath + "/src/agent", "ADR 0005 § D1 row 1: Layer 1 must not import Layer 2"},
	{modulePath + "/src/coding", "ADR 0005 § D1 row 1: Layer 1 must not import Layer 3"},
	{modulePath + "/src/cmd", "ADR 0005 § D1 row 1: Layer 1 must not import the composition root"},

	// ADR 0005 § D3 — the observability boundary is the standard
	// library-instrumentation split: API yes, SDK no. The OTel API paths are
	// deliberately ABSENT from this table; forbidding them would make AI-37
	// impossible without weakening the guard.
	{"go.opentelemetry.io/otel/sdk", "ADR 0005 § D3: the OTel SDK belongs to a composition root"},
	{"go.opentelemetry.io/otel/exporters", "ADR 0005 § D3: OTel exporters belong to a composition root"},
	{"go.opentelemetry.io/contrib/bridges/otelslog", "ADR 0005 § D3: otelslog is permitted no lower than Layer 3"},
}

// allowedNonStdlibPrefixes is the deny-by-default allowlist, minus the standard
// library (which is filtered by the toolchain itself — see listNonStdlibDeps).
//
// AI-37 (ADR 0005 § D3, D-1/D-6) adds five entries — the OpenTelemetry
// tracing API and its own transitive closure, and NOTHING else from that
// ecosystem. Deliberately absent: the root `go.opentelemetry.io/otel`
// global-getter package — § D3's import table permits it, and D-1 declines
// it for two recorded reasons: Layer 1 takes an injected TracerProvider
// instead of the global getter, AND the root package's own measured
// package closure contains `go.opentelemetry.io/auto/sdk`, a path
// literally named for the SDK, inside the one boundary whose acceptance
// clause is "the OTel API and nothing else from that ecosystem"
// (S-AOB-003's owed source-comment, added 2026-08-10) — `otel/metric`, `otel/baggage`,
// `otel/propagation`, and every `sdk`/`exporters`/`auto/sdk`-named path
// (already forbidden above). Each entry below is bounded by the exact
// package-closure pin (TestLayer1_DependencySet_ExactRequiresAndClosure,
// R-AGM-008): the matcher below is prefix-based, so the closure pin — not
// this list — is what stops a future version bump from silently admitting
// a sibling package under an already-allowed prefix.
//
// Anything else reaching this list should be challenged in review, not merged.
var allowedNonStdlibPrefixes = []string{
	modulePath,

	// ADR 0005 § D3: the tracing interfaces (Tracer/Span/TracerProvider),
	// the API's own no-op provider (trace/noop) and the forward-compat
	// marker types (trace/embedded) — all subpackages of this prefix.
	"go.opentelemetry.io/otel/trace",
	// ADR 0005 § D3: the attribute.KeyValue/Value type Layer 1 spans carry.
	"go.opentelemetry.io/otel/attribute",
	// ADR 0005 § D3: the span-status code type (codes.Ok/codes.Error).
	"go.opentelemetry.io/otel/codes",
	// Not itself a § D3 table entry: otel/trace's own vendored semantic-
	// convention package, pulled in transitively by importing otel/trace at
	// all (trace@v1.44.0/auto.go). Authorising an import path necessarily
	// authorises its closure; not an OTel module of its own, so § D3's
	// "any additional OTel module requires its own ADR" clause is not
	// engaged. Bounded by the closure pin (R-AGM-008), not a blank cheque.
	"go.opentelemetry.io/otel/semconv",
	// Not an OTel module: enters transitively through otel/attribute's
	// hashing (attribute/internal/xxhash). Same reasoning as semconv above
	// — bounded by the closure pin, not a blank cheque.
	"github.com/cespare/xxhash/v2",
}

func TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault(t *testing.T) {
	t.Parallel()

	deps, err := listNonStdlibDeps(layer1Pattern)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the module pattern still resolves: " + layer1Pattern)
	}

	for _, dep := range deps {
		// Forbidden first. See the comment on forbiddenPrefixes: an
		// allowlist-first pass would admit the sibling layers.
		if rule, forbidden := matchForbidden(dep); forbidden {
			t.Errorf("Layer 1 must not import %q\n  rule: %s", dep, rule)
			continue
		}
		if !isAllowed(dep) {
			t.Errorf("Layer 1 must not import %q\n"+
				"  rule: deny-by-default allowlist (ADR 0005 § D1 row 1) — this path is neither "+
				"the Go standard library nor a package of %s.\n"+
				"  No forbidden prefix names it, and that is not a licence to add it: adding a "+
				"top-level dependency needs its own ADR (openspec/AGENTS.md rule 5). If this is "+
				"the AI-37 OpenTelemetry API, extend "+
				"allowedNonStdlibPrefixes in the same commit as the ADR.",
				dep, modulePath)
		}
	}
}

// wantGoModRequires is the exact require set ADR 0005 § D3 authorises at
// AI-37 — measured under the repo-root go.work, where a sibling module
// (backend/database_administrator) already pins the OpenTelemetry API at
// v1.44.0 and the unified workspace build list resolves the same version
// here (design.md AD-1). `go get go.opentelemetry.io/otel/trace@v1.44.0
// go.opentelemetry.io/otel@v1.44.0` is expected to land exactly this table.
//
// Update procedure (AD-2, S-AGM-073): on a deliberate, ADR-gated dependency
// bump, re-run `go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}'
// github.com/cachicamas/backend/agent/...`, diff it against
// wantExternalClosure below, verify no `go.opentelemetry.io/otel/sdk`- or
// `go.opentelemetry.io/auto/sdk`-named path appears, and update both this
// table and wantExternalClosure in the same commit.
var wantGoModRequires = []requireEntry{
	{Path: "go.opentelemetry.io/otel", Version: "v1.44.0", Indirect: false},
	{Path: "go.opentelemetry.io/otel/trace", Version: "v1.44.0", Indirect: false},
	{Path: "github.com/cespare/xxhash/v2", Version: "v2.3.0", Indirect: true},
}

// wantExternalClosure is the exact non-stdlib package closure AI-37 pins by
// SET EQUALITY, not by prefix subset (R-AGM-008) — the load-bearing half,
// because allowedNonStdlibPrefixes' matcher (isAllowed, below) is
// prefix-based and would silently admit a future otel/metric,
// otel/baggage or otel/propagation import under the same "otel/trace" or
// "otel/attribute" entries. A version bump that grows this set — the
// vendored semconv path in particular, which is otel/trace's own
// version-numbered subpackage — is a build failure here, not a quiet
// drift (AD-2).
var wantExternalClosure = []string{
	"github.com/cespare/xxhash/v2",
	"go.opentelemetry.io/otel/attribute",
	"go.opentelemetry.io/otel/attribute/internal",
	"go.opentelemetry.io/otel/attribute/internal/xxhash",
	"go.opentelemetry.io/otel/codes",
	"go.opentelemetry.io/otel/semconv/v1.41.0",
	"go.opentelemetry.io/otel/trace",
	"go.opentelemetry.io/otel/trace/embedded",
	"go.opentelemetry.io/otel/trace/internal/telemetry",
	"go.opentelemetry.io/otel/trace/noop",
}

// requireEntry mirrors one element of `go mod edit -json`'s "Require" array
// — Path, Version and whether the requirement is transitive-only
// (`// indirect`).
type requireEntry struct {
	Path     string
	Version  string
	Indirect bool
}

// goModEditJSON is the subset of `go mod edit -json`'s output shape this
// guard reads: the require table and whether any replace directive exists.
// Unmarshaling ignores every other field the command emits (Module, Go,
// Exclude, Retract, …) — Go's default, not a DisallowUnknownFields opt-in.
type goModEditJSON struct {
	Require []requireEntry
	Replace []struct {
		Old struct{ Path, Version string }
		New struct{ Path, Version string }
	}
}

// goModPath returns the absolute path of the go.mod file governing the
// current package — `go env GOMOD`, not a hand-built relative path, so this
// guard keeps working regardless of which package directory `go test` sets
// as its working directory. Verified to resolve to backend/agent/go.mod
// even when invoked from a nested package directory under the repo-root
// go.work (workspace mode does not change which module's go.mod a package
// belongs to).
func goModPath() (string, error) {
	cmd := exec.Command("go", "env", "GOMOD")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &goListError{err: err, stderr: stderr.String()}
	}
	return strings.TrimSpace(stdout.String()), nil
}

// readGoModRequires parses backend/agent/go.mod via `go mod edit -json`
// (AD-2: workspace-immune — it reports go.mod as written, unlike `go list`,
// which reports the unified `go.work` build list). encoding/json and
// os/exec are the guard's existing pattern; this adds no new dependency.
func readGoModRequires() (requires []requireEntry, hasReplace bool, err error) {
	modPath, err := goModPath()
	if err != nil {
		return nil, false, err
	}

	cmd := exec.Command("go", "mod", "edit", "-json", modPath)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, false, &goListError{err: err, stderr: stderr.String()}
	}

	var parsed goModEditJSON
	if err := json.Unmarshal([]byte(stdout.String()), &parsed); err != nil {
		return nil, false, err
	}
	return parsed.Require, len(parsed.Replace) > 0, nil
}

// TestLayer1_DependencySet_ExactRequiresAndClosure replaces
// TestLayer1_ModuleHasNoDependencies_ZeroRequires (AI-37, D-6, AD-2): the
// module's dependency invariant is no longer "zero requires" — AI-37 is the
// milestone `agent-module-scaffold/spec.md` already names as the one that
// amends it. It is strictly stronger than what it replaces, asserting
// THREE things a zero-count assertion could never distinguish:
//
//  1. The exact `require` set (path, version, indirect), by set equality,
//     with zero `replace` directives (R-AGM-001).
//  2. The exact non-stdlib package closure, by set equality — not a prefix
//     subset (R-AGM-008): the allowlist matcher below is prefix-based, so a
//     bare "otel/trace" or "otel/attribute" entry would silently admit any
//     future subpackage a version bump pulled in.
//  3. The forbidden-prefix table (forbiddenPrefixes, above) is unchanged —
//     AI-37 adds no forbidden row and removes none.
func TestLayer1_DependencySet_ExactRequiresAndClosure(t *testing.T) {
	t.Parallel()

	gotRequires, hasReplace, err := readGoModRequires()
	if err != nil {
		t.Fatalf("reading go.mod requires via `go mod edit -json`: %v", err)
	}
	if hasReplace {
		t.Errorf("go.mod declares a replace directive, want zero (R-AGM-001, S-AGM-001)")
	}

	gotRequiresSorted := slices.Clone(gotRequires)
	slices.SortFunc(gotRequiresSorted, func(a, b requireEntry) int { return strings.Compare(a.Path, b.Path) })
	wantRequiresSorted := slices.Clone(wantGoModRequires)
	slices.SortFunc(wantRequiresSorted, func(a, b requireEntry) int { return strings.Compare(a.Path, b.Path) })
	if !slices.Equal(gotRequiresSorted, wantRequiresSorted) {
		t.Errorf("require set mismatch: want %d entries %+v, got %d entries %+v (R-AGM-001, AD-2)",
			len(wantRequiresSorted), wantRequiresSorted, len(gotRequiresSorted), gotRequiresSorted)
	}

	deps, err := listNonStdlibDeps(layer1Pattern)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	var gotClosure []string
	for _, dep := range deps {
		if !strings.HasPrefix(dep, modulePath) {
			gotClosure = append(gotClosure, dep)
		}
	}
	slices.Sort(gotClosure)
	wantClosureSorted := slices.Clone(wantExternalClosure)
	slices.Sort(wantClosureSorted)
	if !slices.Equal(gotClosure, wantClosureSorted) {
		t.Errorf("closure mismatch: want %d entries %v, got %d non-stdlib entries %v (R-AGM-008, AD-2)",
			len(wantClosureSorted), wantClosureSorted, len(gotClosure), gotClosure)
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

func isAllowed(importPath string) bool {
	for _, prefix := range allowedNonStdlibPrefixes {
		if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}

// listNonStdlibDeps returns the deduplicated non-stdlib transitive dependency set
// of pattern, test imports included.
//
// The standard library is filtered by the toolchain via `.Standard` rather than
// by a list maintained here — see the package comment for why the obvious
// heuristic is wrong.
func listNonStdlibDeps(pattern string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-test",
		"-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", pattern)

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

// normalizeListedPackage cleans one line of `go list -deps -test` output.
//
// The -test flag makes go list emit three synthesized shapes that are artifacts
// of how it models test binaries, not real dependencies. Left unnormalized, each
// would be measured against the allowlist and the guard would fail on its own
// module:
//
//	"pkg [pkg.test]"       the package as compiled into a test binary
//	"pkg_test [pkg.test]"  an external test package
//	"pkg.test"             the synthesized test-binary main package
//
// The first two are stripped back to the real import path. The third is dropped:
// it names no importable package.
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

// AI-10.4 item 3 — validation performs no I/O, asserted mechanically in the
// AI-00.3 style: the request path's dependency closure contains no network
// and no filesystem package (R-AMR-013, design.md § 9.3).
//
// # Why this is not the guard above
//
// TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault only screens
// non-stdlib dependencies — [listNonStdlibDeps] filters the standard library
// out before the allowlist ever runs. "net", "net/http", "os" and "io/fs" are
// standard library, so that guard would pass them silently. This is the
// narrower, complementary claim design.md § 9.3 names: not "nothing foreign",
// but "nothing that reaches a socket or a filesystem", checked against the
// standard library too.
//
// # Why -deps without -test
//
// This is a claim about the production dependency closure that runs when a
// request is validated, not about what a test file imports to build one. A
// test elsewhere in this package using t.TempDir (which pulls in "os")
// would be a false positive under a claim this narrow, and -test would also
// pull in "testing", which imports "os" itself — a guard that scanned test
// imports could never pass.
func TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage(t *testing.T) {
	t.Parallel()

	deps, err := listAllDeps(requestPackagePath)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the package pattern still resolves: " + requestPackagePath)
	}

	for _, forbidden := range networkOrFilesystemPackages {
		if slices.Contains(deps, forbidden.path) {
			t.Errorf("the request path's dependency closure imports %q\n  rule: %s", forbidden.path, forbidden.rule)
		}
	}
}

// networkOrFilesystemPackages is S-AMR-051's own list: the four standard-
// library packages design.md § 9.3 names as the shape "an I/O package" takes
// for this guard's purpose. Each entry carries the rule it enforces, matching
// forbiddenPrefixes above.
var networkOrFilesystemPackages = []struct {
	path string
	rule string
}{
	{"net", "V-REQ-22: validation runs entirely at construction and performs no I/O of its own — no network package"},
	{"net/http", "V-REQ-22: validation runs entirely at construction and performs no I/O of its own — no network package"},
	{"os", "V-REQ-22: validation runs entirely at construction and performs no I/O of its own — no filesystem package"},
	{"io/fs", "V-REQ-22: validation runs entirely at construction and performs no I/O of its own — no filesystem package"},
}

// listAllDeps returns every import path in pattern's dependency closure, the
// standard library included and test imports excluded — the opposite two
// choices from [listNonStdlibDeps], for the reasons on the test above.
func listAllDeps(pattern string) ([]string, error) {
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
