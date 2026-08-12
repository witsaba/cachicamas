# Design — `cachicamas-agent-package-scaffold` (AG-03)

> Inputs: `proposal.md` (this change), `explore.md`, ADR 0005 § D1–D3, `backend/agent/src/ai/import_boundary_test.go`, `backend/agent/src/ai/openaicompat/openrouter/ambient_authority_test.go`, `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go`. All new test files are `package agent_test` (mirrors `ai_test`); all guards run under `make test` (`go test -race -v ./...`).

## Technical Approach

Four workpieces, one PR: (1) `src/agent/doc.go` carrying the layer contract as machine-parseable rows plus a byte-level row scanner (AG-03.1); (2) a forward import guard running **three** closure checks (AG-03.2); (3) a call-site AST ambient-authority guard, verbatim AI-25.2 retarget (AG-03.3); (4) the L1 guard self-reference fix, sequenced before AG-03.2's first red.

## Architecture Decisions

### AD-1 — L1 self-reference fix: narrow the scanned roots (falsifies S-AGM-041)

**Choice**: replace `const layer1Pattern = modulePath + "/..."` with

```go
var layer1Patterns = []string{
	modulePath + "/src/ai/...",
	modulePath + "/src/agenttest/...",
	modulePath + "/src/handoff/...",
}
```

and make `listNonStdlibDeps` variadic (`patterns ...string`; `go list` accepts multiple patterns in one invocation). Two call sites update (`import_boundary_test.go:154`, `:323`); the vacuous-pass fatal message joins the slice; the `KNOWN HAZARD` block (lines 82–92) is rewritten to record the fix and date; the `{modulePath}/src/agent` forbidden row stays.

**Alternative rejected — exempt the pattern's own members from `matchForbidden`**: `go list -deps -test` emits a *flattened* set; `src/agent` appearing as a pattern member is byte-identical to `src/agent` appearing as a genuine dependency of `src/ai`. A member exemption would therefore also exempt the exact violation the row exists to catch. Distinguishing member from dependency needs per-package `Deps` attribution (`go list -json`) — a rewrite, not a targeted fix.

**Why narrowing still catches genuine violations**: if `src/ai` ever imports `src/agent`, `src/agent` enters `src/ai/...`'s dep closure, is emitted, and matches the forbidden row. Coverage lost is only "a future top-level `src/*` root is auto-scanned"; the module is now deliberately multi-layer and each layer ships its own guard (this change is the proof), noted in the rewritten comment.

**Spec impact (settled decision 4)**: the fix **narrows the pattern, so `S-AGM-041` ("its pattern covers every package of the module") is falsified** — amend it (full `R-AGM-005` MODIFIED restatement) to: patterns cover every **Layer 1** package (`src/ai/...`, `src/agenttest/...`, `src/handoff/...`), reason stated. `S-AGM-035` amended under `R-AGM-004`: `src/agent` exists (Layer 2, doc 0003); `src/coding`/`src/cmd` still must not. `S-AGM-043`'s eight forbidden rows are untouched. `TestLayer1_DependencySet_ExactRequiresAndClosure` is unaffected: `src/agent` contributes no external package, so `wantExternalClosure` compares equal; the require pin is module-wide (`go env GOMOD`), not pattern-scoped.

### AD-2 — OTel allowlist scope: zero entries granted to Layer 2 (settled decision 5)

**Choice**: AG-03.2 admits **no OTel path on Layer 2's own account** — no root `go.opentelemetry.io/otel`, no `otel/metric`, no § D3 grant of any kind. The five external prefixes that DO appear in the allowlist (`otel/trace`, `otel/attribute`, `otel/codes`, `otel/semconv`, `xxhash/v2`) are admitted **solely as `src/ai`'s measured forced closure** — "authorising an import path necessarily authorises its closure", L1's own recorded reasoning at `import_boundary_test.go:139-148` — and each entry's comment says so. Without them, bite 4's green half (a test file importing `agenttest`, whose closure reaches `src/ai` and thus OTel) could never pass, so they are structurally forced, not a grant.

**Rationale**: the charter ("Out of scope: any event, loop, or harness behavior") means Layer 2 has zero OTel usage at AG-03; deny-by-default admits only what is used and justified now. The milestone's "production closure admits only … the § D3-permitted OTel API paths" is an upper bound ("only") — a stricter subset satisfies it. Precedent: AI-00 shipped an empty third-party group; AI-37, the milestone that first used OTel, added L1's entries under its own justification and deliberately declined root `otel`. The first Layer-2 observability milestone adds its § D3 paths in its own PR, re-annotating the closure-forced entries then. **Alternatives rejected**: full § D3 set (admits unused paths, contradicts deny-by-default purity); L1's exact subset annotated as § D3 grants (mislabels forced transitives as authorized direct imports).

### AD-3 — Forward guard: three checks, two closures

**Choice**: `src/agent/import_boundary_test.go` reuses L1's helpers (`normalizeListedPackage` verbatim; `listNonStdlibDeps(pattern string, includeTest bool)`; `listAllProductionDeps(pattern string)`) and runs:

1. **Production deny-by-default** — `go list -deps` (no `-test`) over `layer2Pattern`, forbidden-first, then `allowedProductionPrefixes`.
2. **Test deny-by-default** — `go list -deps -test`, forbidden-first, then `allowedTestPrefixes` (= production ∪ `…/src/agenttest`).
3. **No-network/filesystem, production closure, stdlib included** — `go list -deps -f '{{.ImportPath}}'` (no `-test`), denying `net`, `net/http`, `os`, `io/fs` by exact path.

**Rationale for check 3**: `net/http` is standard library — `{{if not .Standard}}` filters it out before checks 1–2 ever see it, so the milestone's "denied by name: … `net/http`" is unreachable by the L1 mechanism alone. This is L1's own AI-10.4 pattern (`TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage`), production-only for its recorded reason: `-test` pulls `testing`, which imports `os`. `os/exec`/`syscall` need no closure row: any dep importing them drags `os` into the closure, and in-package call sites are AG-03.3's job. Two separate closure scans (not L1's single merged `-deps -test`) because bite 4 requires asserting them independently.

### AD-4 — Doc-contract row grammar: `L2C-NN`, tab-delimited, committed table

**Choice**: rows in `doc.go`'s package comment match `^//\tL2C-\d\d\t` (AI-40.2's `^//\tCAP-[RO]-\d\d\b` shape, retargeted). Parse: `strings.TrimPrefix(line, "//\t")`, `strings.SplitN(rest, "\t", 2)` → `(id, text)`; the clause text is compared **byte-exact** against a committed `expectedLayer2ContractRows` table; count equality first, then per-row in-order diff — `doc_matrix_guard_test.go`'s exact posture (fatal on unresolvable file or malformed matched row; never vacuous). `docGoPath` resolves via `runtime.Caller(0)` + same-dir join (no `../` hops needed). Later milestones append a row and a table entry in the same PR — that append discipline is the "guarded paragraph" convention AG-03.1 item 2 wanted, substituted per proposal settled decision 1.

### AD-5 — Ambient guard: verbatim AI-25.2 retarget

**Choice**: lift `openrouter/ambient_authority_test.go` wholesale — same forbidden set (`os`, `os/exec`, `syscall`, `io/ioutil`), same alias/dot/blank-import handling, same uniform `_test.go` exclusion (`isLayer2SourceFile`, renamed), same `t.TempDir()` staged-mutation falsifiability test, same recorded no-type-information limitation (a local identifier literally named `os` false-positives; fixing it costs `go/types`/`x/tools`, unauthorized). Scans `"."` (test wd is the package dir). This closes the gap AD-3 cannot: a bare `os.Getenv` call inside Layer 2 passes every import-closure check the moment `os` enters the closure legitimately — and check 3 would catch the closure, but only the AST scan names the call site.

### AD-6 — Fresh closure re-measurement is a gate, not a note (settled decision 3)

**Choice**: before AG-03.2's allowlist is committed, apply MUST run, from `backend/agent/`, and record output verbatim in evidence:

```sh
go list -deps      -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai        # (a)
go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai/...   # (b)
go list -deps      -f '{{.ImportPath}}' ./src/ai                                   # (c)
```

Expected: (a)'s external set ⊆ the ten `wantExternalClosure` entries (five allowlist prefixes); (c) contains none of `net`, `net/http`, `os`, `io/fs`; (b)'s network-reaching members confined to `…/src/ai/openaicompat/**` and `…/src/ai/internal/retry` (the 2026-08-11 claim, doc 0003:81). **Any divergence is a blocking finding reported to the orchestrator — the allowlist is not written against the stale record, and the divergence is not silently absorbed.**

## Interfaces / Contracts

**`src/agent/import_boundary_test.go`** — constants and tables (exact strings):

```go
const modulePath = "github.com/cachicamas/backend/agent"
const layer2Pattern = modulePath + "/src/agent/..."

var forbiddenPrefixes = []struct{ prefix, rule string }{ // checked BEFORE the allowlists — order is load-bearing, as in L1
	{"github.com/cachicamas/backend/database_administrator", "ADR 0005 § D1 row 2: no package of another backend module"},
	{"github.com/cachicamas/backend/workspace_syncer", "ADR 0005 § D1 row 2: no package of another backend module"},
	{modulePath + "/src/coding", "ADR 0005 § D1 row 2: Layer 2 must not import Layer 3"},
	{modulePath + "/src/cmd", "ADR 0005 § D1 row 2: Layer 2 must not import the composition root"},
	{modulePath + "/src/ai/openaicompat", "AG-03.2: the vendor adapter subtree is denied by name — Layer 2 speaks to providers only through the src/ai contract"},
	{"go.opentelemetry.io/otel/sdk", "ADR 0005 § D3: the OTel SDK belongs to a composition root"},
	{"go.opentelemetry.io/otel/exporters", "ADR 0005 § D3: OTel exporters belong to a composition root"},
	{"go.opentelemetry.io/contrib/bridges/otelslog", "ADR 0005 § D3: otelslog is permitted no lower than Layer 3"},
}

var allowedProductionPrefixes = []string{
	modulePath + "/src/agent",
	modulePath + "/src/ai", // the Layer 1 contract package; openaicompat carved back out by the forbidden row above
	// src/ai's measured forced closure (AD-2/AD-6) — NOT § D3 grants to Layer 2:
	"go.opentelemetry.io/otel/trace",
	"go.opentelemetry.io/otel/attribute",
	"go.opentelemetry.io/otel/codes",
	"go.opentelemetry.io/otel/semconv",
	"github.com/cespare/xxhash/v2",
}

var allowedTestPrefixes = append(slices.Clone(allowedProductionPrefixes), modulePath+"/src/agenttest")

var networkOrFilesystemPackages = []struct{ path, rule string }{ // check 3, exact-path match, stdlib included
	{"net", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no network package"},
	{"net/http", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no network package"},
	{"os", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no filesystem/environment package"},
	{"io/fs", "ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no filesystem package"},
}
```

Notes recorded in source: `…/src/handoff` and root `go.opentelemetry.io/otel` are absent from every list — denied by default, deliberately unnamed; `…/src/ai/internal/*` needs no row (Go `internal` visibility already forbids it outside `src/ai`).

Functions: `listNonStdlibDeps(pattern string, includeTest bool) ([]string, error)`, `listAllProductionDeps(pattern string) ([]string, error)`, `normalizeListedPackage(line string) (string, bool)` (verbatim from L1 — handles `pkg [pkg.test]`, `pkg_test [pkg.test]`, `pkg.test`), `matchForbidden(importPath string) (rule string, forbidden bool)`, `isAllowed(importPath string, allowlist []string) bool`. Tests: `TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault`, `TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction`, `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage`. Each fatals on an empty `go list` result (vacuous-pass fence, as L1 line 158).

**`src/agent/doc.go`** — `package agent`, doc comment only. Initial guarded rows (tab after `//`, tab after ID; byte-exact against the table):

```
//	L2C-01	Imports: the Go standard library, github.com/cachicamas/backend/agent/src/ai and its measured transitive closure — nothing else, deny-by-default (ADR 0005 § D1 row 2; import_boundary_test.go).
//	L2C-02	No I/O of its own: no environment read, no filesystem access, no network call, no process spawn (ADR 0005 § D1 row 2; ambient_authority_test.go and import_boundary_test.go).
//	L2C-03	The event stream is the only upward contract: Layer 2 reports exclusively through emitted events; callers observe the stream, they never reach into the loop (AG-00 vocabulary).
```

**`src/agent/doc_contract_guard_test.go`**: `layer2ContractRowPattern = regexp.MustCompile(`^//\tL2C-\d\d\t`)`; `type contractRow struct{ id, text string }`; `var expectedLayer2ContractRows []contractRow` (committed table, three entries above); `docGoPath(t *testing.T) string`; `parseLayer2ContractRows(t *testing.T, path string) []contractRow`; `TestLayer2DocContract_MatchesTheCommittedTable(t *testing.T)`.

**`src/agent/ambient_authority_test.go`**: `type forbiddenAmbientAuthorityPackage struct{ importPath, defaultLocalName, rule string }`; forbidden set exactly `{"os","os/exec","syscall","io/ioutil"}` with L2-worded rules citing ADR 0005 § D1 row 2 / AG-00 must-nevers; `isLayer2SourceFile(name string) bool`; `scanNonTestSourcesForAmbientAuthority(t *testing.T, dir string) []string`; `scanFileForAmbientAuthority(fset *token.FileSet, fileName string, file *ast.File) []string`. Tests: `…_NonTestSourcesCarryNoForbiddenCallSite` (scans `"."`), `…_ForbiddenSetIsPackageScopedDenyByDefault`, `…_FailsOnStagedMutation` (permanent TempDir bite), `…_FileSelectionIsUniform`, `…_TestSourcesStayGreenEvenWithForbiddenCalls`.

## Bite-Proof Procedures (plant → scan → assert → clean; five recorded reds)

| # | Node | Plant | Assert (red) | Clean |
| --- | --- | --- | --- | --- |
| B1 | AG-03.2 | `backend/agent/src/coding/doc.go` (`package coding`, scratch) **and** `src/agent/scratch_violation.go` with `import _ "github.com/cachicamas/backend/agent/src/coding"` (`src/coding` does not exist, so the target package is planted too; a blank import still enters the `-deps` closure) | `go test ./src/agent/...` fails on the `src/coding` **forbidden row**, naming path + rule | delete both scratch files |
| B2 | AG-03.2 | `src/agent/scratch_violation.go` with `import _ "net/http"` | check 3 fails naming `net/http` (checks 1–2 cannot see stdlib — that is the recorded point of check 3) | delete |
| B3 | AG-03.2 | `src/agent/scratch_violation.go` with `import _ "github.com/cachicamas/backend/agent/src/ai/openaicompat"` | the **deny-by-name forbidden row** fires (recorded reason is the row's rule string, not a network side effect; check 3 also firing is recorded as incidental) | delete |
| B4 | AG-03.2 | red half: `src/agent/scratch_violation.go` (production) with `import _ ".../src/agenttest"` → check 1 fails on the **deny-by-default allowlist rule** (no forbidden row names the substrate). Green half: `src/agent/scratch_widen_test.go` with the same import → checks 1 and 3 pass (production closure never sees it), check 2 passes admitting it | both runs recorded | delete both |
| B5 | AG-03.3 | `src/agent/scratch_violation.go` (`package agent`) calling `os.Getenv("SCRATCH_ONLY")` | ambient scan fails naming `scratch_violation.go:<line>: call os.Getenv …` (check 3 also firing on `os` is recorded as the two guards' designed overlap) | delete; the TempDir staged-mutation test keeps the falsifiability proof permanent |

No scratch file or `require` line may appear in the merged diff (proposal success criteria; S-AGM-047 discipline).

## File Changes

| File | Action | Node |
| --- | --- | --- |
| `backend/agent/src/agent/doc.go` | Create | AG-03.1 |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Create | AG-03.1 |
| `backend/agent/src/agent/import_boundary_test.go` | Create | AG-03.2 |
| `backend/agent/src/agent/ambient_authority_test.go` | Create | AG-03.3 |
| `backend/agent/src/ai/import_boundary_test.go` | Modify (AD-1 only) | L1 fix — must land before AG-03.2's first red, in the commit that creates `src/agent/` or earlier in the same PR, or `make test` breaks mid-stack |
| `openspec/changes/…/specs/agent-module-scaffold/spec.md` | Delta (sdd-spec) | `R-AGM-004`/`S-AGM-035`, `R-AGM-005`/`S-AGM-041` — full MODIFIED restatements |
| `backend/agent/go.mod`, `go.sum` | Byte-unchanged | verified by recorded diff |

## Testing Strategy

| Layer | What | How |
| --- | --- | --- |
| Guard (unit) | All three new guards + L1 guard post-fix | `make test` in `backend/agent/` (`go test -race -v ./...`); vacuous-pass fatals in every closure scan |
| Falsifiability | Five recorded reds (B1–B5) + permanent staged-mutation test | plant → scan → assert → clean, per table above |
| Regression | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` green with `src/agent` existing and the forbidden row intact; `TestLayer1_DependencySet_ExactRequiresAndClosure` unchanged | same `make test` run |
| Lint | `make lint` (golangci-lint v2.9.0) | pre/post recorded |

## Threat Matrix

N/A — no routing, shell, VCS/PR automation, executable-file classification, or process-integration boundary. The only subprocesses are test-only `exec.Command("go", …)` invocations with constant argv (no shell, no untrusted input), the module's existing guard pattern.

## Migration / Rollout

No migration. Rollback per proposal: revert the L1 edit + delete `src/agent/` (nothing imports it). Forward-fix preference for over-strict guards stands.

## Open Questions

None blocking. Proposal open questions 1–3 are resolved here: OTel scope (AD-2), L1 fix mechanism (AD-1), row grammar (AD-4).
