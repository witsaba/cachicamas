// AG-03.2 — the forward boundary guard: Layer 2 import purity over both
// closures (R-AGP-003, design AD-3).
//
// This guard enforces ADR 0005 § D1 row 2 mechanically, retargeting
// ai_test's import_boundary_test.go mechanism (AI-00.3) from Layer 1 to
// Layer 2. Structural differences from that mechanism, all AD-3:
//
//  1. SIX checks, not one merged scan: the production closure
//     ([TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault]),
//     the test closure
//     ([TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction]),
//     a network/filesystem closure scan that includes the standard
//     library
//     ([TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage],
//     AI-10.4's pattern), a direct-import family scan over Layer 2's own
//     production source files
//     ([TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport],
//     AG-22 correction, below), a fifth source-level scan over AG-23's
//     two new sibling trees, test files included
//     ([TestLayer2_SiblingTrees_NoDirectForbiddenFamilyImport], AG-23,
//     design AD-2), and a sixth, CH-01 (cachicamas-chat-package-scaffold)'s
//     own per-file, deny-by-default AST scan over the chat archetype's
//     production tree
//     ([TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault],
//     see the "CH-01 amendment" section below). Production and test are
//     asserted SEPARATELY — not L1's single `-deps -test` merge — because
//     S-AGP-026 requires proving the production closure never admits the
//     test substrate (src/agenttest) while the test closure does.
//
//     AG-23 also turns check 2's single scan pattern into a pattern SET:
//     it now sweeps {layer2Pattern} ∪ layer2TestOnlyTreePatterns, one
//     listNonStdlibDeps call PER pattern, with the zero-packages vacuity
//     floor moved INSIDE that loop so a mistyped new pattern fails BY
//     NAME rather than passing vacuously because the union as a whole is
//     still non-empty (S-AGP-070). Checks 1, 3 and 4 stay scoped to
//     layer2Pattern alone, deliberately unwidened, so Layer 2's
//     PRODUCTION closure provably never admits either new sibling tree.
//
//  2. Why check 3 exists at all: "net/http" is standard library, so
//     [listNonStdlibDeps] filters it out before checks 1 and 2 ever see
//     it — a bare-prefix "denied by name" row for it in forbiddenPrefixes
//     would be unreachable dead code. Check 3 uses
//     [listAllProductionDeps], which does NOT filter the standard
//     library, and denies by exact path instead. Production-only, for
//     AI-10.4's own recorded reason: `-test` pulls in "testing", which
//     imports "os" itself, so a test-inclusive scan could never pass.
//
//  3. Why check 4 exists (AG-22 correction, agent-package-scaffold's
//     amended R-AGP-003): check 3's own direct-import-edge mechanism
//     (below) can only ever deny a forbidden path's DIRECT importer by
//     name — a package like "crypto/tls", whose OWN name names no
//     forbidden family but whose own closure directly reaches "net" and
//     "os", is still caught there, one hop out. Check 4 is the
//     complementary, ZERO-hop guard: a source-level scan of Layer 2's
//     own production files' literal import statements, denying anything
//     whose OWN name is itself the forbidden family — "os/exec" and
//     "net/http" — the instant Layer 2 chooses to import it, without any
//     dependency-graph traversal at all.
//
// # The check-3 correction (AG-22, discovered by `sdd-verify`)
//
// Check 3 originally shipped (Phase 3 of this change) with a blanket
// exemption: "a denied path reached only through a standard-library
// importer is inert." That is false in general — "os/exec" and
// "crypto/tls" are BOTH standard library, and BOTH directly reach a
// denied path ("os", or "os"+"net") the instant Layer 2 chooses to
// import them. The blanket exemption made check 3 blind to exactly that
// case: planting "os/exec" in Layer 2 production passed the FULL
// package suite on the commit that shipped the blanket exemption, where
// `main` (the pre-AG-22 tree) correctly failed it.
//
// The fix, matched to agent-package-scaffold's amended R-AGP-003: no
// importer is exempt merely for being standard library. Every DIRECT
// importer of a forbidden path — [forcedStandardLibraryImporters], keyed
// by the exact forbidden path — is either evidence-cited as the forced
// closure of an admitted third-party path (`go.opentelemetry.io/otel/trace`,
// for "os") or evidence-cited as structurally inert (a Go-internal
// package no code outside the standard library can import directly, or
// a standard-library package — "fmt" — whose own admitted purpose does
// not expose the forbidden capability to its own callers). An importer
// on neither list fails the guard, standard library or not.
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
//
// # AG-22 amendment — the predicted first real-usage milestone has landed
//
// The paragraph above predicted "the first Layer 2 milestone with real
// OpenTelemetry usage adds its § D3 paths here, in its own PR, under its
// own justification." That milestone is AG-22 (the observability
// boundary, ADR 0005 § D3 extension): allowedProductionPrefixes below
// now carries five OTel/xxhash entries, each citing either a § D3 table
// row or, for the two entries with no row of their own, the forced-
// closure clause and a fresh go list -deps measurement (apply-progress
// for AG-22). The reasoning above (AG-03's own zero-grant decision) is
// left unstruck, not deleted: it explains why the allowlist started
// empty, and remains true as a description of that milestone's own
// evidence — it does not describe today's allowlist.
//
// # CH-01 amendment — the chat archetype's own boundary (check 6)
//
// SIX checks, not five: cachicamas-chat-package-scaffold (CH-01) adds
// [TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault],
// this module's first import-boundary check over a Layer 3 ARCHETYPE
// rather than over Layer 2 itself. It is a per-file, deny-by-default AST
// scan over backend/agent/src/chat/..., RECURSIVE (filepath.WalkDir,
// skipping directories named "testdata" and skipping "_test.go" files) —
// a stated amendment to proposal D-6's "directly inside" wording. CH-02
// (design AD-1) in fact kept the tree flat — no subpackage — but the
// recursive walk was deliberately chosen ahead of that decision so a later
// milestone that DOES add one is not silently released from the guard by a
// top-level-only scan.
//
// Every failure this check emits cites ADR 0005 § D1 row 3 (or § D3)
// TOGETHER WITH the ADR 0009 § D2 substitution ("read as any Layer 3
// archetype"), written out in full at each use — the same convention
// `chat-archetype-contract` already promoted as R-CHT-011 — and names the
// offending file by its path RELATIVE TO THE DECLARED ROOT, never by
// filepath.Base, because a base name stops identifying a file uniquely
// once a recursive walk reaches a subpackage.
//
// backend/agent/src/cmd/chat, the archetype's own composition root, is
// DELIBERATELY OUTSIDE this check's scanned set (chatArchetypeDirs): it
// is the one package ADR 0005 § D3's cmd/ column (adr:242) grants the
// OpenTelemetry SDK and exporters, so scanning it would deny the very
// import that grant exists to permit.
//
// chatArchetypeForbiddenPrefixes deliberately carries NO row for
// github.com/cachicamas/backend/database_administrator or
// github.com/cachicamas/backend/workspace_syncer, even though this
// file's own forbiddenPrefixes above names both for Layer 2: naming
// either here would make a planted violation's failure cite a named
// forbidden prefix instead of this check's deny-by-default framing,
// falsifying the message contract's third property (R-CPB-005).
//
// chatArchetypeAllowedPrefixes admits
// go.opentelemetry.io/contrib/bridges/otelslog — ADR 0005 § D3 (adr:243,
// ✅ at L3 coding), read under the ADR 0009 § D2 substitution as this
// archetype's own grant (user decision, Engram #3741). This is a
// DELIBERATE, RECORDED DEVIATION from agent-package-scaffold R-AGP-003's
// same-commit rule, and the deviation covers the WHOLE TABLE, not the
// bridge alone: at CH-01 the archetype imports nothing at all, so every
// entry is an allowlist entry without an import. R-AGP-003's rule governs
// an INCREMENTAL allowlist — an existing boundary admitting one new
// dependency; this table is a BOUNDARY DECLARATION for a new position,
// stated once so the permitted surface is legible before any code
// exercises it. Two alternatives were rejected: an empty allowlist grown
// import by import would leave the allowlist branch itself unexercised;
// a passing bite per entry would cost roughly forty further lines and
// five further recorded runs against a change already near its
// authorised review budget.
//
// The zero-hop gap this check alone leaves: at CH-01, check 6 reads each
// file's own import declarations and nothing further — no transitive
// (go list -deps) closure check ran over the archetype tree, deferred for
// a mechanical reason: the tree was doc.go-only, so a dependency listing
// returned zero non-stdlib packages and this repo's own pinned
// zero-packages vacuity floor (agent-package-scaffold S-AGP-070) would
// have failed a closure check run against it. CH-02 (R-CCP-010, R-CCP-011)
// closes that gap below with checks 7 and 8 — see the "CH-02 amendment"
// section immediately following this one.
//
// # CH-02 amendment — the archetype's transitive closure (checks 7 and 8)
//
// TWO more checks, not six: CH-02 (cachicamas-chat-conversation) adds
// [TestChatArchetype_ProductionClosure_ImportsOnlyAllowedPrefixes_DenyByDefault]
// (check 7, R-CCP-010) and
// [TestChatArchetype_TestClosure_AdmitsOnlyTheScriptedKitBeyondProduction]
// (check 8, R-CCP-011), mirroring checks 1/2's own production/test split
// (S-AGP-025, S-AGP-026, D9) rather than one merged `go list -deps -test`
// scan — so the production closure is provably, independently proven never
// to admit the scripted kit (S-CCP-101).
//
// chatArchetypeAllowedPrefixes gains two entries, both measured fresh
// against this tree rather than copied from Layer 2's own table (see the
// table's own comments): the archetype's own package path (a `go list
// -deps` self-inclusion every scanned pattern carries, the same reason
// allowedProductionPrefixes above carries modulePath+"/src/agent" itself),
// and github.com/cespare/xxhash/v2, the forced transitive closure of
// otel/attribute's own internal hashing (agent-package-scaffold
// R-AGP-003's forced-closure clause) — identically justified to Layer 2's
// own xxhash entry above.
//
// Check 8 carries its OWN forbidden table, chatArchetypeTestForbiddenPrefixes,
// deliberately distinct from chatArchetypeForbiddenPrefixes: reusing the
// production table would wrongly deny the very admission R-CCP-011
// requires (src/agenttest, src/apptest). src/layer3handoff remains denied
// to BOTH closures — see its rewritten row in chatArchetypeForbiddenPrefixes
// above, which also corrects CH-01's own forward reference on that path
// rather than silently replacing it.
package agent_test

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// layer2TestOnlyTreePatterns are AG-23's two new sibling package trees —
// the packaged test kit (src/apptest) and the consumer proof
// (src/layer3handoff) — that are neither Layer 2 nor its Layer 1
// substrate (design AD-2, agent-package-scaffold's amended R-AGP-003).
// Check 2 (the test closure) sweeps {layer2Pattern} ∪
// layer2TestOnlyTreePatterns; checks 1, 3 and 4 stay scoped to
// layer2Pattern alone, deliberately unwidened, so Layer 2's PRODUCTION
// closure provably never admits either sibling tree. Both trees are
// siblings of src/agent, never nested beneath it: allowedProductionPrefixes
// admits modulePath+"/src/agent" as a PREFIX, so nesting either tree
// under it would silently admit it into the production closure,
// destroying the separate-closure property S-AGP-025/S-AGP-026 exist to
// prove.
var layer2TestOnlyTreePatterns = []string{
	modulePath + "/src/apptest/...",
	modulePath + "/src/layer3handoff/...",
}

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
// listNonStdlibDeps). AG-22 (the observability boundary) adds the five
// OTel/xxhash entries below it, each citing either an ADR 0005 § D3
// table row or, for the two entries with no row of their own
// (otel/semconv, cespare/xxhash/v2), the forced-closure clause — see the
// package comment's "AG-22 amendment" for why this allowlist no longer
// stays OTel-free the way ai_test's own allowlist does not either.
var allowedProductionPrefixes = []string{
	modulePath + "/src/agent",
	modulePath + "/src/ai", // the Layer 1 contract package; openaicompat carved back out by the forbidden row above

	// AG-22 (design D-A/C4, ADR 0005 § D3 Layer 2 extension). Every
	// entry below is the tracing API only — never the SDK or an
	// exporter, both already denied by name in forbiddenPrefixes above.
	"go.opentelemetry.io/otel/trace",     // § D3 table row (0005:240); prefix covers the forced subpackages trace/noop, trace/embedded, trace/internal/telemetry
	"go.opentelemetry.io/otel/attribute", // § D3 table row (0005:241)
	"go.opentelemetry.io/otel/codes",     // § D3 table row (0005:241)
	// go.opentelemetry.io/otel/semconv is NOT a § D3 table row: it is the
	// forced transitive closure of otel/trace at the version otel/trace
	// pins (R-AGP-003's forced-closure clause), confirmed by
	// `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}'
	// go.opentelemetry.io/otel/trace/noop` (apply-progress for AG-22).
	// This MUST stay a prefix, never an exact path: the measured path is
	// version-suffixed (go.opentelemetry.io/otel/semconv/v1.41.0),
	// pinned by otel/trace's own go.mod requirement, and an exact entry
	// would break on an unrelated otel/trace version bump (S-AGP-032).
	"go.opentelemetry.io/otel/semconv",
	// github.com/cespare/xxhash/v2 is likewise NOT a § D3 table row: it
	// is forced by otel/attribute's own internal hashing
	// (attribute/internal/xxhash), same forced-closure clause and same
	// fresh measurement.
	"github.com/cespare/xxhash/v2",
}

// allowedTestPrefixes is check 2's allowlist: the production set plus the
// test substrate, src/agenttest — asserted as its own separate scan (AD-3)
// so the production closure can be proven, independently, to never admit
// it (S-AGP-025, S-AGP-026). The prefix match below covers agenttest's own
// subpackages (for example agenttest/sweep) without a separate entry.
//
// AG-23: the two new sibling test-only trees (layer2TestOnlyTreePatterns)
// join this same allowlist, in the SAME commit as the trees themselves
// (R-AGP-003's own "amendment lands in the same commit as the import it
// authorises" rule) — never the production allowlist above, which stays
// unwidened.
var allowedTestPrefixes = append(slices.Clone(allowedProductionPrefixes), modulePath+"/src/agenttest", modulePath+"/src/apptest", modulePath+"/src/layer3handoff")

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
//
// AG-23 (design AD-2): the scanned pattern SET is now {layer2Pattern} ∪
// layer2TestOnlyTreePatterns, one listNonStdlibDeps call PER PATTERN —
// never one call over a merged/union pattern string — because the
// zero-packages vacuity floor below MUST be per-pattern, not per-set
// (S-AGP-070). A mistyped new pattern resolves zero packages on its own;
// with a per-SET floor the union as a whole would still be non-empty
// (layer2Pattern alone already resolves plenty), so the mistyped pattern
// would pass vacuously, silently dropping a tree from the sweep while the
// guard reports green.
func TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction(t *testing.T) {
	t.Parallel()

	patterns := append([]string{layer2Pattern}, layer2TestOnlyTreePatterns...)
	for _, pattern := range patterns {
		deps, err := listNonStdlibDeps(pattern, true)
		if err != nil {
			t.Fatalf("go list failed for pattern %q: %v", pattern, err)
		}
		if len(deps) == 0 {
			t.Fatalf("go list -test returned no packages for pattern %q; the guard would pass vacuously for this pattern. "+
				"Check that the package pattern still resolves.", pattern)
		}

		for _, dep := range deps {
			if rule, forbidden := matchForbidden(dep); forbidden {
				t.Errorf("pattern %q: Layer 2's test closure must not import %q\n  rule: %s", pattern, dep, rule)
				continue
			}
			if !isAllowed(dep, allowedTestPrefixes) {
				t.Errorf("pattern %q: Layer 2's test closure must not import %q\n"+
					"  rule: deny-by-default allowlist (ADR 0005 § D1 row 2) — this path is neither "+
					"the Go standard library, a production-allowed package, nor the test substrate "+
					"(src/agenttest, src/apptest, src/layer3handoff).\n"+
					"  No forbidden prefix names it, and that is not a licence to add it.",
					pattern, dep)
			}
		}
	}
}

// forcedStandardLibraryImporters is check 3's own closed, evidence-cited
// allowlist of every DIRECT importer permitted to pull a forbidden path
// into the closure (AG-22 correction, agent-package-scaffold's amended
// R-AGP-003's "matching mechanism" paragraph). "net" and "net/http"
// carry NO entry and stay denied unconditionally the instant either
// gains ANY direct importer, exactly as before.
//
// This is NOT a "standard library is inert" exemption — the original
// AG-22 commit shipped exactly that blanket exemption, and `sdd-verify`
// found it let "os/exec" and "crypto/tls", planted directly in Layer 2
// production, reach "os" (and "os"+"net" respectively) invisibly: both
// are themselves standard library, so the blanket rule waved them
// through without ever asking whether LAYER 2 ITSELF had chosen to
// import them. Every entry below is instead named and justified
// individually, standard library or not:
//
//   - "go.opentelemetry.io/otel/trace" is the one legitimate NON-standard
//     importer of "os": trace@v1.44.0's own auto.go file makes exactly
//     one os.Getenv call (auto.go:665), gated behind OpenTelemetry's
//     SEPARATE auto-instrumentation bridge module
//     (go.opentelemetry.io/auto/sdk) — a module design D-A rejects by
//     name (S-AGP-024) and this closure never admits.
//   - "fmt" is standard library and directly imports "os" — but only for
//     os.Stdout/os.Stderr; it exposes no filesystem or process
//     capability to ITS OWN callers, unlike "os/exec" (process
//     execution) or "crypto/tls" (network dialing), and is required,
//     directly or transitively, by nearly every package in this module.
//   - "os" is itself the one legitimate importer of "io/fs" and
//     "syscall" in this closure — its own internal need, already
//     governed by the "os" row above, not a second independent choice.
//   - "internal/filepathlite", "internal/poll", "internal/syscall/unix"
//     and "internal/syscall/execenv" are Go-internal: the `internal/`
//     visibility rule the Go toolchain itself enforces means no package
//     outside the standard library — Layer 2's own code included — can
//     ever import one of them directly, so listing them here documents
//     the closure rather than trusting the reader to already know it.
//   - "time" directly imports "syscall" for OS clock/timer primitives;
//     it exposes no filesystem/network/process capability to its own
//     callers.
//
// An importer absent from a path's own list here — including a FUTURE
// Layer 2 source file of this package's own, or any other standard-
// library package not named above — fails the guard exactly as an
// unvetted third-party importer would (S-AGP-036). Re-measured with
// `go list -deps -f '{{.ImportPath}}|{{join .Imports ","}}' ./src/agent/...`
// against the clean baseline (apply-progress for this correction).
var forcedStandardLibraryImporters = map[string][]string{
	"os":      {"go.opentelemetry.io/otel/trace", "fmt"},
	"io/fs":   {"os", "internal/filepathlite"},
	"syscall": {"os", "time", "internal/poll", "internal/syscall/unix", "internal/syscall/execenv"},
}

// productionImportGraph runs `go list -deps` once over pattern's
// PRODUCTION closure (no -test, matching listAllProductionDeps' own
// invocation) and returns, for every package the closure contains, the
// exact set of packages that import it DIRECTLY — not merely
// transitively. Check 3 matches every direct importer of a forbidden
// path against that path's own closed forcedStandardLibraryImporters
// list above; there is no "is it standard library" branch left to
// short-circuit that match (the AG-22 correction this file's package
// comment records).
func productionImportGraph(pattern string) (importedBy map[string][]string, err error) {
	cmd := exec.Command("go", "list", "-deps", "-f", `{{.ImportPath}}|{{join .Imports ","}}`, pattern)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		return nil, &goListError{err: runErr, stderr: stderr.String()}
	}

	importedBy = make(map[string][]string)
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "|", 2)
		if len(fields) != 2 {
			continue
		}
		importer, ok := normalizeListedPackage(fields[0])
		if !ok {
			continue
		}
		if fields[1] == "" {
			continue
		}
		for _, imported := range strings.Split(fields[1], ",") {
			imported = strings.TrimSpace(imported)
			if imported == "" {
				continue
			}
			importedBy[imported] = append(importedBy[imported], importer)
		}
	}
	return importedBy, nil
}

// TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage is
// check 3 (AD-3, AI-10.4's pattern, ai_test's
// TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage):
// "net", "net/http", "os" and "io/fs" are standard library, so checks 1
// and 2 never see them — [listNonStdlibDeps] filters the standard library
// out before either allowlist ever runs. This is the narrower,
// complementary claim: not "nothing foreign", but "nothing that reaches a
// socket or a filesystem" THAT LAYER 2 OR AN ADMITTED DEPENDENCY REACHES
// ITSELF (AG-22's own refinement, below) — checked against the standard
// library too. Production-only, for AI-10.4's own recorded reason: -test
// pulls in "testing", which imports "os" itself, so a test-inclusive scan
// could never pass.
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

	var (
		importedBy map[string][]string
		graphErr   error
	)
	for _, forbidden := range networkOrFilesystemPackages {
		if !slices.Contains(deps, forbidden.path) {
			continue
		}
		if importedBy == nil {
			importedBy, graphErr = productionImportGraph(layer2Pattern)
			if graphErr != nil {
				t.Fatalf("go list (import graph) failed: %v", graphErr)
			}
		}

		// AG-22 correction: every DIRECT importer of forbidden.path must
		// be on ITS OWN vetted list — standard library or not. There is
		// no "importer happens to be standard library" exemption; that
		// was the defect (see forcedStandardLibraryImporters' own doc
		// comment and this file's package comment).
		vetted := forcedStandardLibraryImporters[forbidden.path]
		var unexpected []string
		for _, importer := range importedBy[forbidden.path] {
			if slices.Contains(vetted, importer) {
				continue
			}
			unexpected = append(unexpected, importer)
		}
		if len(unexpected) > 0 {
			t.Errorf("Layer 2's production closure imports %q via unexpected direct importer(s) %v (vetted: %v)\n  rule: %s",
				forbidden.path, unexpected, vetted, forbidden.rule)
		}
	}
}

// layer2ForbiddenImportFamilies is check 4's own family list (AG-22
// correction, agent-package-scaffold's amended R-AGP-003's "direct-
// import family scan" paragraph): a Layer 2 production source file's own
// import MUST NOT be exactly one of these names, nor start with one of
// them plus "/" — regardless of how many further hops that import's own
// closure would need to reach a forbidden path (check 3's own concern,
// complementary to this one). This is a source-level, ZERO-hop scan: it
// catches "os/exec" and "net/http" by NAME, the instant Layer 2 chooses
// to import them directly, without any dependency-graph traversal.
var layer2ForbiddenImportFamilies = []string{"os", "net", "syscall", "io/fs"}

// isForbiddenImportFamily reports whether importPath is exactly one of
// families or has one of them as a "/"-bounded prefix — so "os/exec"
// matches family "os" and "net/http" matches family "net", but
// "crypto/tls" and "fmt" match neither by name (check 3's
// direct-import-edge mechanism is what catches "crypto/tls" instead;
// "fmt" is vetted there, per forcedStandardLibraryImporters). Shared by
// check 4 (families: layer2ForbiddenImportFamilies) and AG-23's check 5
// (families: layer2SiblingTreeForbiddenImportFamilies, the same four
// families plus "time") — the same mechanism, applied to two different
// family lists over two different scanned trees.
func isForbiddenImportFamily(importPath string, families []string) (family string, matched bool) {
	for _, f := range families {
		if importPath == f || strings.HasPrefix(importPath, f+"/") {
			return f, true
		}
	}
	return "", false
}

// layer2AgentPackageDir resolves package agent's own directory relative
// to THIS test file's own source location — the same runtime.Caller(0)
// resolution observability_scope_test.go's agoScopePackageDir performs
// independently, per this codebase's own per-file self-containment
// convention.
func layer2AgentPackageDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve the agent package directory")
	}
	return filepath.Dir(thisFile)
}

// layer2ProductionSourceFiles returns the absolute path of every
// non-test .go file directly inside dir (package agent's own directory;
// no subpackages).
func layer2ProductionSourceFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	return files
}

// TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport is check 4
// (AG-22 correction, agent-package-scaffold's amended R-AGP-003): a
// direct, source-level scan of Layer 2's OWN production .go files for an
// import matching layer2ForbiddenImportFamilies — independent of check
// 3's transitive-closure mechanism, so a denied family is caught by NAME
// the instant Layer 2 chooses to import it, regardless of which further
// path its own closure would transitively reach.
func TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport(t *testing.T) {
	t.Parallel()

	dir := layer2AgentPackageDir(t)
	files := layer2ProductionSourceFiles(t, dir)
	if len(files) == 0 {
		t.Fatalf("no production .go file found directly inside %q; the scan would pass vacuously", dir)
	}

	fset := token.NewFileSet()
	for _, path := range files {
		file, ferr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if ferr != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v, want nil", path, ferr)
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if family, matched := isForbiddenImportFamily(importPath, layer2ForbiddenImportFamilies); matched {
				t.Errorf("%s directly imports %q, which is in the forbidden %q family\n  rule: ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no network, filesystem or process package, reached directly or through any intermediary",
					filepath.Base(path), importPath, family)
			}
		}
	}
}

// layer2SiblingTreeForbiddenImportFamilies is check 5's own family list
// (AG-23, design AD-2, agent-package-scaffold's amended R-AGP-003): the
// two new sibling trees may import none of Layer 2's own forbidden
// families PLUS "time" — the zero-I/O AND no-wall-clock property
// (R-L3H-002, R-L3H-004) denied by name, at the source level, in one
// scan. "time" carries no entry in layer2ForbiddenImportFamilies because
// check 4's own scope (Layer 2 production) has no no-wall-clock rule of
// its own; check 5 is a distinct list over a distinct pair of trees.
var layer2SiblingTreeForbiddenImportFamilies = append(slices.Clone(layer2ForbiddenImportFamilies), "time")

// layer2SiblingTreeDirs resolves AG-23's two new sibling trees' own
// directories relative to THIS test file's own source location — the
// same runtime.Caller(0) resolution layer2AgentPackageDir performs for
// Layer 2's own tree, extended one level up to reach src/apptest and
// src/layer3handoff as SIBLINGS of src/agent, never nested beneath it
// (design AD-2, the separate-closure property S-AGP-025/S-AGP-026 exist
// to keep). Keyed by the same import path layer2TestOnlyTreePatterns
// names, so a failure message below can cite one alongside the other.
func layer2SiblingTreeDirs(t *testing.T) map[string]string {
	t.Helper()
	srcDir := filepath.Dir(layer2AgentPackageDir(t))
	return map[string]string{
		modulePath + "/src/apptest":       filepath.Join(srcDir, "apptest"),
		modulePath + "/src/layer3handoff": filepath.Join(srcDir, "layer3handoff"),
	}
}

// layer2SiblingTreeSelfExcludedFiles is check 5's own, single, exact-name
// exclusion, discovered during apply (not planned at design time): the
// generic-client vocabulary guard (R-L3H-010) is co-located inside
// src/layer3handoff, one of the two trees check 5 itself scans, and it
// necessarily reads its sibling trees' own source bytes for a static
// text inspection — needing the very standard-library family this check
// denies. That need is categorically different from the kit's and the
// proof's own RUNTIME zero-I/O claim: it is verification tooling reading
// already-committed source for a build-time scan, exactly the same
// category as import_boundary_test.go's own os.ReadDir/os.ReadFile use
// scanning Layer 2's production files for check 4 — nobody reads THAT
// as a violation of Layer 2's own zero-I/O claim over its production
// surface, because the scanning mechanism is not part of the surface it
// scans. The exclusion is exact-filename, no wildcard/prefix/directory
// pattern, matching every other named exception in this module's guards.
//
// The exclusion is keyed on the (tree, filename) PAIR below, never on
// filename alone (sdd-verify round 1, W-9): a filename-only exclusion is
// a working escape hatch — a same-named file planted in the OTHER
// sibling tree (src/apptest) carries none of this reasoning and MUST
// still be scanned. Proven by planting exactly that file, with a direct
// "os" and "time" import, inside src/apptest: pre-fix the scan passed
// green; post-fix it fails naming the file and the family, identically
// to any other file in that tree.
const layer2SiblingTreeSelfExcludedFile = "generic_client_guard_test.go"

// layer2SiblingTreeSelfExcludedTreeImportPath is the ONLY import path
// layer2SiblingTreeSelfExcludedFile is ever excluded from. Together with
// layer2SiblingTreeSelfExcludedFile above, this forms the (tree,
// filename) key: a file of that exact name in any OTHER tree this check
// scans is not excluded and is scanned like every other file there.
const layer2SiblingTreeSelfExcludedTreeImportPath = modulePath + "/src/layer3handoff"

// layer2SiblingTreeAllSourceFiles returns the absolute path of every .go
// file — test files INCLUDED — directly inside dir (no subpackages),
// except layer2SiblingTreeSelfExcludedFile when, and only when, dir is
// the tree named by layer2SiblingTreeSelfExcludedTreeImportPath. Check
// 5's own listing helper: unlike check 4's own layer2ProductionSourceFiles,
// which deliberately EXCLUDES _test.go (Layer 2 production-only scope),
// check 5 must see test files too — the zero-I/O and no-wall-clock
// property this change proves covers a sibling tree's own test files
// exactly as much as its production files (R-L3H-002).
func layer2SiblingTreeAllSourceFiles(t *testing.T, importPath, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if importPath == layer2SiblingTreeSelfExcludedTreeImportPath && name == layer2SiblingTreeSelfExcludedFile {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	return files
}

// TestLayer2_SiblingTrees_NoDirectForbiddenFamilyImport is check 5 (AG-23,
// design AD-2, agent-package-scaffold's amended R-AGP-003): a direct,
// source-level scan of the two new sibling trees' OWN .go files —
// production AND test files — for an import matching
// layer2SiblingTreeForbiddenImportFamilies. Verbatim reuse of check 4's
// AST mechanism (parser.ParseFile with parser.ImportsOnly), applied to a
// different pair of trees and a different family list.
//
// This is a ZERO-HOP scan: it lists no dependencies at all, so the
// filtered-closure blind spot that affects checks 1/2 (a closure listing
// with the standard library filtered out before the allowlist ever sees
// it) does not affect THIS check's own mechanism — there is no closure
// here to filter in the first place.
//
// sdd-verify round 1, W-11 (corrected): that does NOT mean every
// standard-library-mediated reach is closed. This check denies only a
// DIRECT import of one of the five named families; a tree importing some
// OTHER standard-library package that itself reaches one of those five
// internally (an intermediary such as "text/template", which reaches
// "os") is caught by NEITHER this check (direct-imports-only; the
// intermediary's own name is not on the denied list) NOR check 2 (its
// closure listing filters the standard library out before the allowlist
// ever sees it, so a standard-library intermediary — and whatever IT
// reaches — is invisible to that check too). This is a genuine residual
// gap in the mechanism, not one closed elsewhere: R-L3H-002 requires
// denial "by name", and this scan denies the five names it lists, no
// more. A non-standard-library intermediary IS still bounded — it would
// itself need a place in check 2's own allowlist — but a
// standard-library one is not. The residual floor besides this gap is
// the test framework's own unavoidable "testing" -> "os" reach, the same
// floor check 3's own comment already records — restated here, not
// hidden.
func TestLayer2_SiblingTrees_NoDirectForbiddenFamilyImport(t *testing.T) {
	t.Parallel()

	dirs := layer2SiblingTreeDirs(t)
	fset := token.NewFileSet()
	for importPath, dir := range dirs {
		files := layer2SiblingTreeAllSourceFiles(t, importPath, dir)
		if len(files) == 0 {
			// AG-23 (S-AGP-072): the vacuity floor is PER TREE — a
			// renamed or moved tree must fail by naming ITS OWN
			// directory, not pass because some OTHER tree in this map
			// is non-empty.
			t.Fatalf("no .go file found directly inside %q (%s); the scan would pass vacuously for this tree", dir, importPath)
		}
		for _, path := range files {
			file, ferr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if ferr != nil {
				t.Fatalf("parser.ParseFile(%q) error = %v, want nil", path, ferr)
			}
			for _, imp := range file.Imports {
				imported := strings.Trim(imp.Path.Value, `"`)
				if family, matched := isForbiddenImportFamily(imported, layer2SiblingTreeForbiddenImportFamilies); matched {
					t.Errorf("%s directly imports %q, which is in the forbidden %q family\n  rule: R-L3H-002/R-L3H-004: the kit and the consumer proof perform no I/O and use no wall clock — no network, filesystem, process or time package, in production or test source",
						filepath.Base(path), imported, family)
				}
			}
		}
	}
}

// chatArchetypeForbiddenPrefixes are checked BEFORE
// chatArchetypeAllowedPrefixes, mirroring forbiddenPrefixes above and for
// the same load-bearing reason (CH-01, design AD-2): the allowlist admits
// modulePath+"/src/ai" as a PREFIX, so an allowlist-first pass would
// silently admit ".../src/ai/openaicompat" — checking this table first is
// what carves it back out (proven by S-CPB-033's bite). Every rule string
// is final text: ADR 0005 § D1 row 3 (or § D3) together with the ADR 0009
// § D2 substitution ("read as any Layer 3 archetype"), written out in
// full rather than left as an unexplained analogy, so it stays true after
// this change's own SDD artifacts move to openspec/changes/archive/.
//
// Deliberately ABSENT: "github.com/cachicamas/backend/database_administrator"
// and "github.com/cachicamas/backend/workspace_syncer", even though
// forbiddenPrefixes above names both for Layer 2. Naming either here
// would make a planted violation's failure cite A NAMED FORBIDDEN
// PREFIX instead of the archetype's deny-by-default framing — precisely
// the "missing allowlist entry" reading the message contract forbids
// (S-CPB-042, S-CPB-043; proven wrong by S-CPB-044's defeat test, which
// drafts either row in and watches the deny-by-default property fail).
// R-07 stays enforced regardless: no chatArchetypeAllowedPrefixes entry
// below reaches either module, so default denial is the mechanism.
var chatArchetypeForbiddenPrefixes = []struct {
	prefix string
	rule   string
}{
	{modulePath + "/src/ai/openaicompat", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the archetype reaches row 1 only through the src/ai contract — the vendor adapter subtree is denied by name"},
	{modulePath + "/src/agenttest", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: rows 1–2 are importable as production contract surface only — Layer 2's test substrate is not part of that surface"},
	{modulePath + "/src/apptest", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: Layer 2's packaged test kit is consumer-proof machinery, not row 2's production surface"},
	// CH-02 correction: CH-01's own comment here named this path as the one
	// a future test closure would decide on. That forward reference was
	// itself a defect: `…/src/layer3handoff` ships only `doc.go` plus two
	// `_test.go` files, so it has NO IMPORTABLE SURFACE AT ALL — admitting
	// it would grant nothing. It is therefore denied to BOTH the
	// production closure (this table) and the test closure
	// (chatArchetypeTestForbiddenPrefixes below), unconditionally. The
	// packages `0005:66` (R-10) actually requires every harness consumer
	// to test against are `…/src/apptest` and `…/src/agenttest` — named
	// nowhere in CH-01's own row — and CH-02's test closure (check 8)
	// admits both instead.
	{modulePath + "/src/layer3handoff", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: it has no importable surface at all, so it is denied to both the production and test closures — R-10's test-substrate requirement is met by src/apptest and src/agenttest instead"},
	{modulePath + "/src/cmd", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: nothing below the composition root imports src/cmd"},
}

// chatArchetypeAllowedPrefixes is check 6's deny-by-default allowlist,
// minus the standard library (classified by
// chatArchetypeStandardLibraryPackages, the toolchain's own answer —
// never a maintained list or a path-shape heuristic, honoring this
// file's own recorded warning against both). Every entry cites its
// authority in place, as allowedProductionPrefixes above requires of its
// own entries.
//
// The table ships COMPLETE at CH-01, and this is a DELIBERATE, RECORDED
// DEVIATION from agent-package-scaffold R-AGP-003's same-commit rule
// ("The allowlist amendment MUST land in the same commit as the first
// production import that requires it"), and the deviation covers the
// WHOLE TABLE, not any single entry: at CH-01 the archetype imports
// nothing at all, so EVERY entry below — modulePath+"/src/agent" and
// modulePath+"/src/ai" exactly as much as otelslog — is an allowlist
// entry without an import. R-AGP-003's same-commit rule governs an
// INCREMENTAL allowlist: an existing boundary admitting one new
// dependency, the shape S-AGP-034/S-AGP-069 each proved with a bite.
// This table is a different object: a BOUNDARY DECLARATION for a new
// position, stated once so the position's permitted surface is legible
// before any code exercises it. Two alternatives were considered and
// rejected: (1) an empty allowlist, grown import by import, would leave
// the permitted surface living only as doc.go prose, with the allowlist
// branch itself shipping unexercised — a branch no evidence reaches; (2)
// a passing bite per entry, exercising each admission, would cost
// roughly forty further lines and five further recorded runs, against a
// change already near its authorised review budget.
//
// go.opentelemetry.io/otel/sdk/... and .../exporters/... carry NO row
// here, denied by DEFAULT rather than by a forbidden-table row: no entry
// below — including otelslog's — reaches either prefix, so default
// denial is the mechanism, and a forbidden row would change the
// failure's framing in the way the message contract (R-CPB-005) forbids.
// They are permitted only inside backend/agent/src/cmd/chat, which
// chatArchetypeDirs below deliberately excludes from the scanned set.
var chatArchetypeAllowedPrefixes = []string{
	// REMEDIATION (verify #3783, CRITICAL-2): the archetype's own self-
	// inclusion (modulePath+"/src/chat") used to live here. It was
	// moved to chatArchetypeForcedClosurePrefixes, which only the
	// closure check (check 7) reads, so admitting it does not widen
	// the pre-existing per-file AST scan (R-CCP-012 forbids widening
	// any pre-existing check). See
	// TestChatArchetype_Check6AndCheck7_UseDistinctAllowlists and
	// chatArchetypeForcedClosurePrefixes' own comment for the
	// structural rationale.
	modulePath + "/src/agent",            // ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: rows 1–2 are importable
	modulePath + "/src/ai",               // same row; the vendor adapter subtree carved back out by chatArchetypeForbiddenPrefixes above
	"go.opentelemetry.io/otel/trace",     // ADR 0005 § D3 (adr:240): tracing API permitted at every layer
	"go.opentelemetry.io/otel/attribute", // ADR 0005 § D3 (adr:241)
	"go.opentelemetry.io/otel/codes",     // ADR 0005 § D3 (adr:241)
	"go.opentelemetry.io/otel/semconv",   // forced closure of otel/trace (see allowedProductionPrefixes' own semconv entry)
	// ADR 0005 § D3 (adr:243): otelslog is permitted at L3 coding, read as any
	// Layer 3 archetype under ADR 0009 § D2 — user decision, Engram #3741.
	// Authorised ahead of its first import by that decision; a deliberate,
	// recorded deviation from Layer 2's same-commit amendment convention.
	"go.opentelemetry.io/contrib/bridges/otelslog",
	// CH-02 (measured against this tree, Engram #3775, re-measured at
	// apply): forced transitive closure of otel/attribute's own internal
	// hashing (attribute/internal/xxhash), the archetype's counterpart of
	// allowedProductionPrefixes' own identically-justified xxhash entry
	// above (agent-package-scaffold R-AGP-003's forced-closure clause).
	// REMEDIATION (verify #3783, CRITICAL-2): this entry was originally
	// placed in chatArchetypeAllowedPrefixes itself, which silently
	// widened the pre-existing per-file AST scan (check 6, R-CCP-012
	// forbids widening any pre-existing check). It now lives in
	// chatArchetypeForcedClosurePrefixes, read ONLY by check 7's
	// transitive closure path. See that variable's own comment for the
	// structural rationale and
	// TestChatArchetype_Check6AndCheck7_UseDistinctAllowlists for the
	// regression guard.
	// "github.com/cespare/xxhash/v2", -- moved to chatArchetypeForcedClosurePrefixes
	// CH-03 (ADR at adr/echo-v5-in-agent-module, R-CHT-008, R-CPB-009):
	// Echo v5 is the chat archetype's HTTP framework. ADR records db_admin's
	// precedent, the rejected alternatives, and the boundary-widening row.
	"github.com/labstack/echo/v5",
	// CH-07 (ADR 0010-add-pgx-and-goose-to-backend-agent, NFR-CCS-005
	// allowlist amendment clause, R-AGP-003 same-commit rule): the chat
	// archetype admits pgx/v5/stdlib as a blank-imported driver registration
	// so chat/migrator/runner.go and chat/store_postgres.go can call
	// sql.Open("pgx", dsn). The dep is admitted in the same PR as this
	// widening (the ADR is WU-1 in the same change).
	"github.com/jackc/pgx/v5/stdlib",
	// CH-07 (ADR 0010-add-pgx-and-goose-to-backend-agent, same R-AGP-003
	// same-commit clause): the chat archetype admits pressly/goose/v3 as
	// the migration runner for the chat-owned forward-only SQL migrations.
	// chat/migrator/runner.go imports it for goose.NewProvider.
	"github.com/pressly/goose/v3",
}

// chatArchetypeForcedClosurePrefixes is a SEPARATE, closure-only allowlist
// read ONLY by check 7's transitive (go list -deps) closure scan
// (TestChatArchetype_ProductionClosure_ImportsOnlyAllowedPrefixes_DenyByDefault,
// R-CCP-010) and, where its own closure reaches the same paths, by check 8's
// test closure. Check 6's per-file AST scan
// (TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault)
// NEVER reads this list, so admitting a forced-closure path here does NOT
// widen the pre-existing guard — restoring R-CCP-012's "no pre-existing
// check may be widened" property that the shared allowlist form violated.
//
// This mirrors how Layer 2 already separates its own roles: allowedProductionPrefixes
// above carries its closure-only rows (otel/semconv, cespare/xxhash/v2)
// alongside its production-only ones because Layer 2's check 1 IS its own
// closure check, with no companion AST scan to widen. The chat archetype
// has check 6 (AST) and check 7 (closure) side by side, so the two roles
// must be carried as two distinct slices.
//
// Every entry below was measured FRESH against this tree (Engram #3775,
// re-measured at apply, not trusted from cache) and is justified under
// agent-package-scaffold R-AGP-003's forced-closure clause. Each must
// land as a PREFIX (version-suffixed paths break exact entries, S-AGP-032).
var chatArchetypeForcedClosurePrefixes = []string{
	// CH-02: `go list -deps` over chatArchetypePattern always includes
	// the queried tree's OWN package among its results — the same
	// mechanical reason allowedProductionPrefixes above carries
	// modulePath+"/src/agent" itself as its own first entry. CH-01
	// never observed this: check 6 is a per-file AST scan of import
	// STATEMENTS, and a package never imports itself, so the gap was
	// invisible until CH-02's check 7 made it visible.
	modulePath + "/src/chat",
	// CH-02: forced transitive closure of otel/attribute's own
	// internal hashing (attribute/internal/xxhash), the archetype's
	// counterpart of allowedProductionPrefixes' own identically-
	// justified xxhash entry above.
	"github.com/cespare/xxhash/v2",
	// CH-07 (ADR 0010-add-pgx-and-goose-to-backend-agent, R-AGP-003
	// forced-closure clause): pgx/v5 is admitted as a top-level dep
	// in chatArchetypeAllowedPrefixes above; its transitive closure
	// (pgxpool, pgconn, puddle, pgpassfile, pgservicefile, interpolate)
	// is admitted here in the closure-only list — check 7 reads this
	// list, check 6 does not. The pre-existing chat-archetype pattern
	// (allowed + forced-closure as two distinct slices) is preserved.
	"github.com/jackc/pgx/v5",
	"github.com/jackc/pgx/v5/pgxpool",
	"github.com/jackc/pgx/v5/pgconn",
	"github.com/jackc/puddle/v2",
	"github.com/jackc/pgpassfile",
	"github.com/jackc/pgservicefile",
	"github.com/mfridman/interpolate",
	// CH-07: pgx's transitive closure pulls in x/text and x/sync
	// (runes, bidi, secure/precis, width, semaphore, etc.). Admitted
	// here under the same forced-closure clause.
	"golang.org/x/text",
	"golang.org/x/sync",
	// CH-07: pressly/goose/v3 is admitted as a top-level dep above;
	// its transitive closure (logrus adapter internals, retry, the
	// io/fs virtual-filesystem helpers) lives here. Measured fresh
	// against the tree at WU-2.
	"github.com/pressly/goose/v3",
	"github.com/sethvargo/go-retry",
	// CH-07: goose's transitive closure pulls in go.uber.org/multierr
	// (used by goose's error-aggregation helpers). Admitted under the
	// same forced-closure clause.
	"go.uber.org/multierr",
}

// chatArchetypeMatchForbidden mirrors matchForbidden above but is its own,
// separate function over chatArchetypeForbiddenPrefixes (design AD-2's
// helper policy): matchForbidden is hardcoded to Layer 2's own
// forbiddenPrefixes, whose database_administrator/workspace_syncer rows
// are exactly what this table deliberately omits — sharing the two
// tables' matcher would re-couple them and silently reintroduce those
// rows here. The honest duplication cost is six lines; drift risk is
// bounded because the two tables are SUPPOSED to differ.
func chatArchetypeMatchForbidden(importPath string) (rule string, forbidden bool) {
	for _, f := range chatArchetypeForbiddenPrefixes {
		if importPath == f.prefix || strings.HasPrefix(importPath, f.prefix+"/") {
			return f.rule, true
		}
	}
	return "", false
}

// chatArchetypeDirs resolves check 6's declared scan roots, keyed by
// import path, mirroring layer2SiblingTreeDirs above and resolved the
// same way: relative to THIS test file's own source location
// (layer2AgentPackageDir's runtime.Caller(0)), never an assumed working
// directory (NFR-CPB-001).
//
// modulePath+"/src/cmd/chat" is DELIBERATELY ABSENT. That absence is
// itself the mechanism behind CH-01.2's second scenario clause — "the
// same import inside the composition root passes" — because
// backend/agent/src/cmd/chat is the one package of this archetype ADR
// 0005 § D3's cmd/ column (adr:242) grants the OpenTelemetry SDK and
// exporters; a scan that swept it would deny the very import the
// position exists to grant. Proven by S-CPB-050's bite and S-CPB-051's
// own negative control — a bite without its negative control does not
// prove the exclusion.
//
// This map has one entry because check 6 (CH-01) never scanned a test
// closure at all; CH-02's checks 7/8 introduce the production/test split
// this file's own Layer 2 checks already use, over chatArchetypePattern —
// a go-list-based pattern, not a second entry in this AST-scan-only
// directory map. modulePath+"/src/layer3handoff" is denied to BOTH
// closures unconditionally — see its rewritten row in
// chatArchetypeForbiddenPrefixes above, which corrects CH-01's own
// forward reference on this exact point rather than silently replacing
// it.
func chatArchetypeDirs(t *testing.T) map[string]string {
	t.Helper()
	srcDir := filepath.Dir(layer2AgentPackageDir(t))
	return map[string]string{modulePath + "/src/chat": filepath.Join(srcDir, "chat")}
}

// chatArchetypeStandardLibraryPackages runs `go list std` once per
// check-6 run and returns its output as a membership set (design AD-6,
// NFR-CPB-003): the AST scan below sees literal stdlib imports directly —
// unlike checks 1–2, whose listNonStdlibDeps filters the standard library
// out before either allowlist ever runs — so check 6 needs its own
// classification, and it is the TOOLCHAIN's own answer, never a
// maintained list or a path-shape heuristic, honoring this file's own
// recorded warning against both (see listNonStdlibDeps' package comment).
// On subprocess failure this t.Fatalf's with the command's stderr,
// matching every existing `go list` failure path in this file.
func chatArchetypeStandardLibraryPackages(t *testing.T) map[string]struct{} {
	t.Helper()
	cmd := exec.Command("go", "list", "std")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list std failed: %v", &goListError{err: err, stderr: stderr.String()})
	}

	pkgs := make(map[string]struct{})
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pkgs[line] = struct{}{}
	}
	return pkgs
}

// TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
// is check 6 (CH-01.2, cachicamas-chat-package-scaffold, design AD-1…AD-9):
// this module's first import-boundary check over a Layer 3 ARCHETYPE
// rather than over Layer 2 itself — a per-file, deny-by-default AST scan
// over chatArchetypeDirs' declared roots. Follows check 1's four-segment
// name shape (Test<Subject>_<Scope>_<Behavior>_<Expectation>) because the
// expectation — DenyByDefault — is the load-bearing property here.
//
// The walk is RECURSIVE per declared root (filepath.WalkDir, skipping
// directories literally named "testdata" and skipping "_test.go" files) —
// a STATED AMENDMENT to proposal D-6's "directly inside" wording, not a
// silent widening: checks 4 and 5 above scan flat single packages by
// design, but the archetype tree was always free to grow a subpackage
// later, and a top-level-only scan would silently release the first one
// from this guard — exactly the escape class this milestone exists to
// close. CH-02 (design AD-1) in fact kept the tree flat, so this
// recursion has not yet had a subpackage to prove itself against; it
// stays armed for whichever milestone adds the first one. A later change
// MAY overturn this back to a flat scan; doing so re-opens the subpackage
// escape and MUST assign that gap an owner.
//
// A failing import is named by its path RELATIVE TO THE DECLARED ROOT
// (filepath.Rel; a Rel error is itself fatal), NEVER by filepath.Base as
// checks 4 and 5 do: a base name stops identifying a file uniquely the
// moment a recursive walk reaches a subpackage — "chat.go" directly
// inside src/chat and "chat.go" inside a future src/chat/port would
// render identically, and a reader of a failure could not tell which
// file to open. Checks 4 and 5 may use a base name because they scan flat
// directories where base names are unique by construction; this check
// does not have that guarantee. A root-level file renders identically
// under both shapes, so nothing is lost at CH-01.
//
// Parse shape is parser.ParseFile(fset, path, nil, parser.ImportsOnly),
// verbatim from checks 4 and 5: it reads each file's own import
// declarations and RESOLVES NOTHING (NFR-CPB-001's determinism, R-CPB-009's
// go.mod byte-freeze) — the check's verdict is a function of the import
// path as WRITTEN, not of the module graph, so it neither requires a path
// to be in the build list nor is weakened when one happens to be. This is
// also this check's own ZERO-HOP boundary: it evaluates each file's
// direct imports only, never a transitive (go list -deps) closure — CH-02
// ships that closure check as checks 7 and 8 below (see the "CH-02
// amendment" section of this file's package comment), extending rather
// than replacing this check's own zero-hop scan.
//
// Evaluation order per import, per AD-6: chatArchetypeForbiddenPrefixes
// FIRST (forbidden-first is load-bearing, not cosmetic — the allowlist
// admits ".../src/ai" as a PREFIX, so an allowlist-first pass would
// silently admit ".../src/ai/openaicompat", proven wrong by S-CPB-033's
// bite), then the standard library, then chatArchetypeAllowedPrefixes,
// then deny. Forbidden and stdlib are disjoint, so this ordering is
// preserved trivially.
//
// The per-declared-root vacuity floor is FILE-COUNT based, never
// dependency-count based: a doc.go-only tree — exactly what CH-01 ships —
// has zero non-standard-library dependencies but one file, and a
// dependency-count floor would fail this guard on a tree that is exactly
// as the milestone intends. Mirrors check 4's own floor above, and
// follows agent-package-scaffold S-AGP-070's per-declared-input rule: a
// WalkDir error names the mistyped directory and its import path; zero
// collected files under a declared root names that root — both fatal,
// never a silent pass. Floor granularity is the DECLARED root, never a
// discovered subdirectory: S-AGP-070 protects declared inputs, and under
// a recursive walk a subdirectory is discovered, not declared — a
// mistyped subdirectory simply does not exist and is never visited, so a
// per-subdirectory floor could never fire on the failure S-AGP-070 exists
// to catch, while it would false-positive on a legitimately Go-free
// directory.
func TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault(t *testing.T) {
	t.Parallel()

	dirs := chatArchetypeDirs(t)
	stdlib := chatArchetypeStandardLibraryPackages(t)
	fset := token.NewFileSet()

	for archetypeImportPath, rootDir := range dirs {
		var files []string
		walkErr := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			name := d.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if walkErr != nil {
			t.Fatalf("filepath.WalkDir(%q) error = %v, want nil (chat archetype root %s)", rootDir, walkErr, archetypeImportPath)
		}
		if len(files) == 0 {
			t.Fatalf("no production .go file found under %q (%s); the scan would pass vacuously", rootDir, archetypeImportPath)
		}

		for _, path := range files {
			file, ferr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if ferr != nil {
				t.Fatalf("parser.ParseFile(%q) error = %v, want nil", path, ferr)
			}
			relPath, relErr := filepath.Rel(rootDir, path)
			if relErr != nil {
				t.Fatalf("filepath.Rel(%q, %q) error = %v, want nil", rootDir, path, relErr)
			}

			for _, imp := range file.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)

				if rule, forbidden := chatArchetypeMatchForbidden(importPath); forbidden {
					t.Errorf("%s directly imports %q\n  rule: %s", relPath, importPath, rule)
					continue
				}
				if _, isStdlib := stdlib[importPath]; isStdlib {
					continue
				}
				if matchesPrefix(importPath, chatArchetypeAllowedPrefixes) {
					continue
				}
				t.Errorf("%s directly imports %q, which the chat archetype's deny-by-default allowlist does not admit\n"+
					"  rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2) — "+
					"this path is neither the Go standard library nor a package this archetype's allowlist admits.\n"+
					"  No forbidden prefix names it, and that is not a licence to add it: adding a dependency "+
					"needs its own recorded design decision in chatArchetypeAllowedPrefixes.",
					relPath, importPath)
			}
		}
	}
}

// chatArchetypePattern scopes CH-02.4's two go-list-based checks (7 and 8)
// to the archetype's own package tree, mirroring layer2Pattern's own role
// for checks 1/2. Fully qualified, recursive (the "/..." suffix), matching
// check 6's own recursive AST walk (chatArchetypeDirs) rather than a single
// top-level package.
const chatArchetypePattern = modulePath + "/src/chat/..."

// chatArchetypeTestForbiddenPrefixes is check 8's own forbidden table
// (R-CCP-011) — deliberately DISTINCT from chatArchetypeForbiddenPrefixes
// above: that production table denies src/agenttest and src/apptest by
// name, and this requirement's whole point is admitting them to the TEST
// closure. Reusing the production table here would wrongly deny the very
// admission R-CCP-011 requires. src/layer3handoff stays denied to BOTH
// closures (it has no importable surface at all — see the rewritten row in
// chatArchetypeForbiddenPrefixes above), and src/cmd and the vendor adapter
// subtree stay denied to the test closure exactly as they are to
// production.
var chatArchetypeTestForbiddenPrefixes = []struct {
	prefix string
	rule   string
}{
	{modulePath + "/src/ai/openaicompat", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the archetype reaches row 1 only through the src/ai contract — the vendor adapter subtree is denied by name"},
	{modulePath + "/src/layer3handoff", "R-CCP-011: this path has no importable surface at all (doc.go plus two _test.go files) — admitting it would grant nothing, and it stays denied to both closures"},
	{modulePath + "/src/cmd", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: nothing below the composition root imports src/cmd"},
}

// chatArchetypeTestMatchForbidden mirrors chatArchetypeMatchForbidden but is
// its own, separate function over chatArchetypeTestForbiddenPrefixes
// (R-CCP-011): sharing one matcher across the production and test tables
// would re-couple two tables that are SUPPOSED to differ — the production
// table's agenttest/apptest denial rows are exactly what the test closure
// must NOT inherit.
func chatArchetypeTestMatchForbidden(importPath string) (rule string, forbidden bool) {
	for _, f := range chatArchetypeTestForbiddenPrefixes {
		if importPath == f.prefix || strings.HasPrefix(importPath, f.prefix+"/") {
			return f.rule, true
		}
	}
	return "", false
}

// TestChatArchetype_ProductionClosure_ImportsOnlyAllowedPrefixes_DenyByDefault
// is check 7 (CH-02.4, R-CCP-010): the archetype's PRODUCTION transitive
// dependency closure — no test files — admits only the standard library and
// chatArchetypeAllowedPrefixes. Extends check 6's zero-hop, per-file AST
// scan with the transitive closure CH-01 deferred: at CH-01 the tree was
// doc.go-only, so a dependency listing returned zero non-stdlib packages
// and agent-package-scaffold S-AGP-070's vacuity floor would have fired;
// CH-02.1's conversation now imports src/agent and src/ai, so the listing
// is non-vacuous (S-CCP-090).
func TestChatArchetype_ProductionClosure_ImportsOnlyAllowedPrefixes_DenyByDefault(t *testing.T) {
	t.Parallel()

	deps, err := listNonStdlibDeps(chatArchetypePattern, false)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list returned no packages; the guard would pass vacuously. " +
			"Check that the package pattern still resolves: " + chatArchetypePattern)
	}

	for _, dep := range deps {
		// Forbidden first (design AD-2, mirroring forbiddenPrefixes' own
		// load-bearing order above): the allowlist admits ".../src/ai" as a
		// prefix, so an allowlist-first pass would silently admit
		// ".../src/ai/openaicompat".
		if rule, forbidden := chatArchetypeMatchForbidden(dep); forbidden {
			t.Errorf("The chat archetype's production closure must not import %q\n  rule: %s", dep, rule)
			continue
		}
		// REMEDIATION (verify #3783, CRITICAL-2): the closure check
		// reads the UNION of the narrow allowlist (check 6's) and the
		// forced-closure allowlist (closure-only). The two slices were
		// merged before remediation, which silently widened check 6's
		// per-file AST scan; the structural split keeps check 6 narrow
		// (it never sees chatArchetypeForcedClosurePrefixes) while
		// letting check 7's transitive scan still admit the measured
		// forced closures.
		allowed := append(slices.Clone(chatArchetypeAllowedPrefixes), chatArchetypeForcedClosurePrefixes...)
		if !isAllowed(dep, allowed) {
			t.Errorf("The chat archetype's production closure must not import %q\n"+
				"  rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under "+
				"ADR 0009 § D2) — this path is neither the Go standard library nor a package this archetype's "+
				"production allowlist admits.\n"+
				"  No forbidden prefix names it, and that is not a licence to add it: adding a dependency needs "+
				"its own recorded design decision in chatArchetypeAllowedPrefixes or chatArchetypeForcedClosurePrefixes, measured fresh against this tree.",
				dep)
		}
	}

	// R-CCP-010: the unfiltered closure (standard library included) must
	// not carry any denied prefix either — listNonStdlibDeps' own filter
	// cannot see a stdlib-mediated boundary break (S-CCP-095).
	allDeps, allErr := listAllProductionDeps(chatArchetypePattern)
	if allErr != nil {
		t.Fatalf("go list (unfiltered) failed: %v", allErr)
	}
	deniedUnfilteredPrefixes := []string{
		modulePath + "/src/ai/openaicompat",
		modulePath + "/src/agenttest",
		modulePath + "/src/apptest",
		modulePath + "/src/layer3handoff",
		modulePath + "/src/cmd",
	}
	for _, dep := range allDeps {
		for _, denied := range deniedUnfilteredPrefixes {
			if dep == denied || strings.HasPrefix(dep, denied+"/") {
				t.Errorf("The chat archetype's UNFILTERED production closure reaches denied prefix %q via %q — a stdlib-mediated break the filtered listing above cannot see", denied, dep)
			}
		}
	}
}

// TestChatArchetype_TestClosure_AdmitsOnlyTheScriptedKitBeyondProduction is
// check 8 (CH-02.4, R-CCP-011): the archetype's TEST dependency closure,
// scanned SEPARATELY from check 7 (mirroring Layer 2's own check-1/check-2
// split, S-AGP-025/S-AGP-026, D9) so the production closure can be proven,
// independently, never to admit the scripted kit (S-CCP-101). Admits
// everything chatArchetypeAllowedPrefixes admits, plus src/agenttest and
// src/apptest as prefixes — the whole subtree deliberately, including
// agenttest/tracetest and agenttest/sweep.
func TestChatArchetype_TestClosure_AdmitsOnlyTheScriptedKitBeyondProduction(t *testing.T) {
	t.Parallel()

	deps, err := listNonStdlibDeps(chatArchetypePattern, true)
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("go list -test returned no packages; the guard would pass vacuously. " +
			"Check that the package pattern still resolves: " + chatArchetypePattern)
	}

	// REMEDIATION (verify #3783, CRITICAL-2): test closure also reads
	// the union (narrow allowlist + forced-closure allowlist + test
	// substrate). Same structural rationale as check 7 above: keep
	// check 6 narrow; let the closure-only paths through the closure
	// checks via a distinct slice.
	allowed := append(slices.Clone(chatArchetypeAllowedPrefixes), chatArchetypeForcedClosurePrefixes...)
	allowed = append(allowed, modulePath+"/src/agenttest", modulePath+"/src/apptest")

	for _, dep := range deps {
		if rule, forbidden := chatArchetypeTestMatchForbidden(dep); forbidden {
			t.Errorf("The chat archetype's test closure must not import %q\n  rule: %s", dep, rule)
			continue
		}
		if !isAllowed(dep, allowed) {
			t.Errorf("The chat archetype's test closure must not import %q\n"+
				"  rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under "+
				"ADR 0009 § D2, plus R-10's test-substrate admission) — this path is neither the Go standard "+
				"library, a production-allowed package, nor the scripted kit (src/agenttest, src/apptest).\n"+
				"  No forbidden prefix names it, and that is not a licence to add it.",
				dep)
		}
	}
}

// TestChatArchetype_Check6AndCheck7_UseDistinctAllowlists is a structural
// regression guard added during CH-02 remediation (verify #3783, CRITICAL-2):
// check 6 (per-file AST scan, R-CCP-010's zero-hop precedent) and check 7
// (transitive closure, R-CCP-010's main check) MUST NOT share a single
// allowlist slice. R-CCP-012 forbids widening any pre-existing check, and a
// shared slice would widen check 6 every time R-CCP-010's closure-only
// requirements force check 7 to admit a new transitive path (e.g. a
// stdlib-mediated reach such as otel/attribute's own internal hashing,
// forced into github.com/cespare/xxhash/v2 at the version otel/attribute
// pins). Layer 2 keeps its production and test closures apart structurally
// (allowedProductionPrefixes vs allowedTestPrefixes, see check 1 / check 2
// above); the chat archetype inherits the same separation here.
//
// Concretely:
//   - chatArchetypeAllowedPrefixes is check 6's narrow allowlist. It must
//     not contain either of the two closure-only entries this change's
//     fresh measurement identified (modulePath+"/src/chat" and
//     github.com/cespare/xxhash/v2), because admitting either to check 6
//     would silently widen that pre-existing guard.
//   - chatArchetypeForcedClosurePrefixes is the NEW, closure-only set that
//     ONLY check 7 (and, when its own closure happens to reach the same
//     paths, check 8) reads on top of chatArchetypeAllowedPrefixes. The
//     two entries above live here, mirroring how Layer 2 carries
//     modulePath+"/src/agent" and go.opentelemetry.io/otel/semconv as
//     forced-closure entries without admitting them to its own
//     direct-import AST scan.
func TestChatArchetype_Check6AndCheck7_UseDistinctAllowlists(t *testing.T) {
	t.Parallel()

	// Assert the new, closure-only allowlist exists and carries the
	// two entries this change measured as forced closure. Failure
	// here (RED) motivates the split; this is the test the structural
	// repair must satisfy.
	if !slices.Contains(chatArchetypeForcedClosurePrefixes, modulePath+"/src/chat") {
		t.Errorf("chatArchetypeForcedClosurePrefixes is missing the archetype's own self-inclusion %q (forced closure of `go list -deps` over chatArchetypePattern)",
			modulePath+"/src/chat")
	}
	if !slices.Contains(chatArchetypeForcedClosurePrefixes, "github.com/cespare/xxhash/v2") {
		t.Errorf("chatArchetypeForcedClosurePrefixes is missing %q (forced closure of otel/attribute's own internal hashing, agent-package-scaffold R-AGP-003's forced-closure clause)",
			"github.com/cespare/xxhash/v2")
	}

	// Assert the narrow check 6 allowlist stays narrow: neither
	// forced-closure entry leaked in via the shared slice. This is the
	// invariant CRITICAL-2 violated at HEAD and that Plant B1
	// (chat production file with `import _ "github.com/cespare/xxhash/v2"`,
	// run against check 6 alone) observes: at HEAD, this assertion
	// fails; after the split, it passes; at fdd41d9c (merge base, before
	// CH-02), it also passes because the two entries were never in the
	// narrow allowlist to begin with.
	if slices.Contains(chatArchetypeAllowedPrefixes, modulePath+"/src/chat") {
		t.Errorf("chatArchetypeAllowedPrefixes (check 6's narrow allowlist) MUST NOT contain the forced-closure self-inclusion %q — admitting it would silently widen the pre-existing per-file AST scan (R-CCP-012). Move it to chatArchetypeForcedClosurePrefixes, which only the closure check (check 7) reads.",
			modulePath+"/src/chat")
	}
	if slices.Contains(chatArchetypeAllowedPrefixes, "github.com/cespare/xxhash/v2") {
		t.Errorf("chatArchetypeAllowedPrefixes (check 6's narrow allowlist) MUST NOT contain %q — admitting it would silently widen the pre-existing per-file AST scan (R-CCP-012). Move it to chatArchetypeForcedClosurePrefixes, which only the closure check (check 7) reads.",
			"github.com/cespare/xxhash/v2")
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
