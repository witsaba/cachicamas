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
// It holds exactly one entry today. One milestone may add a second, and it
// carries its own gate:
//
//   - AI-37 adds the OpenTelemetry API — and ONLY the API paths enumerated in
//     ADR 0005 § D3, which pre-authorises them and nothing else.
//
// Anything else reaching this list should be challenged in review, not merged.
var allowedNonStdlibPrefixes = []string{
	modulePath,
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

// TestLayer1_ModuleHasNoDependencies_ZeroRequires pins the property the
// allowlist is protecting, from the other side. The allowlist says what may be
// imported; this says that today nothing outside the module is, so go.mod needs
// no `require` at all.
//
// It is a pin (green from birth, exempt from red-first): AI-00's charter states
// the module is stdlib-only until AI-24 and AI-37, and this is the assertion
// that makes that statement checkable rather than aspirational.
func TestLayer1_ModuleHasNoDependencies_ZeroRequires(t *testing.T) {
	t.Parallel()

	deps, err := listNonStdlibDeps(layer1Pattern)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}

	for _, dep := range deps {
		if !strings.HasPrefix(dep, modulePath) {
			t.Errorf("the agent module is expected to carry zero dependencies, but %q is external.\n"+
				"  If a milestone with an ADR added it deliberately, update this test and "+
				"allowedNonStdlibPrefixes together.", dep)
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
