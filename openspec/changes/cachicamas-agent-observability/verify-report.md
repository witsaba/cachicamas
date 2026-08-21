```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:df50cd7c9ee5f32b69aab1ba262641719fe58ec884d8d11337b1345df248e673
verdict: fail
blockers: 2
critical_findings: 2
requirements: 8/16
scenarios: 55/75
test_command: cd backend/agent && go test -race -count=1 -v ./...
test_exit_code: 0
test_output_hash: sha256:c214d59077fba71b8f5f934a73486558ba0292c7021e80865ea2e717e9b1c8f7
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-observability` (AG-22 — Add the observability boundary)
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag22-observability`, branch `feat/agent-layer2-wave6-ag22`, HEAD `fbd8acb6`, base `main@7dac9ec9`
**Mode**: Strict TDD
**Verdict**: **FAIL** — 2 CRITICAL. The suite is green, but green is not the claim: one shipped guard was rewritten so that it no longer catches what it caught on `main`, and one normative MUST is violated on a reachable path that no test drives.

---

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 49 |
| Tasks complete | 45 |
| Tasks incomplete | 4 (12.1–12.4, Phase 12 — orchestrator scope, out of this run by design) |

Phase 12 is deliberately unstarted and is **not** reported as an omission.

---

### Build & Tests Execution — every claim re-run, none trusted

| Gate | Command | Result |
|---|---|---|
| Tests | `go test -race -count=1 -v ./...` | **exit 0** — 1451 top-level PASS / 0 FAIL / 2 SKIP (uncached; never `(cached)`) |
| Build | `go build ./...`, `make build` | exit 0, clean |
| Vet | `go vet ./...` | exit 0, clean |
| Lint | `golangci-lint cache clean && make lint` | exit 0, **0 issues** |
| Vulnerabilities | `make vuln-check` | exit 0, **0 `finding` entries** |
| gofmt | `gofmt -l .` | 15 files — **all 15 independently confirmed dirty on `main` too** (`git show main:<path> \| gofmt -d`); I agree with the orchestrator |
| Freeze | `git diff --stat main...HEAD -- go.mod go.sum src/ai/` | empty — byte-unchanged |
| Diff size | `git diff --numstat main...HEAD` excl. `openspec/` | 17 files, **+2362/−51** — exactly as claimed |

The 2 skips are `TestTurn_CoverageGate` and `TestOpenRouterAdapter_LiveSmoke`, both pre-existing and unrelated.

**Apply's self-report was accurate on every number I re-ran.** The findings below are things apply's report did not surface, not things it misstated.

---

## CRITICAL

### CRITICAL-1 — The check-3 rewrite weakened enforcement; `os/exec` in Layer 2 production now passes the entire suite

**Where**: `backend/agent/src/agent/import_boundary_test.go:380-386` (the `if standard[importer] { continue }` branch), introduced in commit `07341886`.

`TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage` was rewritten from
`if slices.Contains(deps, forbidden.path) → fail` to a **direct-importer** check that exempts (a) a
per-path vetted allowlist and (b) **any standard-library importer**. Clause (b) is the hole: a denied
path reached through *any* standard-library intermediary is now invisible.

Planted, observed, reverted (worktree clean after each):

| Plant in a Layer 2 production file | On `main` | On `HEAD` |
|---|---|---|
| `import "os"` | FAIL | **FAIL** ✅ (control still bites) |
| `import "net/http"` | FAIL | **FAIL** ✅ (control still bites) |
| `import "os/exec"` | FAIL | **PASSES** ❌ |
| `import "crypto/tls"` | FAIL | **PASSES** ❌ |

Proof the original check would have caught them — with the plant in place,
`go list -deps ./...` shows `os` in the closure (via `os/exec`) and `net` in the closure (via
`crypto/tls`), so `slices.Contains(deps, …)` was true in both cases.

Worse, **no other guard covers it**: with `os/exec` planted, the *full* package suite reports
`ok github.com/cachicamas/backend/agent/src/agent 5.310s`. Check 1's deny-by-default allowlist never
sees stdlib (the toolchain filters it first), and the ambient-authority guard does not catch it either.
`os/exec` is process execution — precisely the ambient authority `NFR-AGO-003` and `R-AGP-005` exist
to deny.

Reproduce:
```
cd backend/agent/src/agent
printf 'package agent\n\nimport "os/exec"\n\nvar _ = exec.Command\n' > zz.go
go test -count=1 ./... ; rm zz.go     # → ok  (expected: FAIL)
```

**Why this is a spec violation, not a judgement call:**

- `NFR-AGO-003` (`agent-observability-boundary/spec.md:298`) — "The ten substrate files named in
  `openspec/AGENTS.md` MUST remain byte-unchanged, **including `import_boundary_test.go`'s behaviour**
  — its **data tables** are what this change amends." The change amended ~90 lines of *behaviour*
  (`productionImportGraph`, `forcedStandardLibraryImporters`, a rewritten matching loop), not data tables.
- `R-AGP-003` / `S-AGP-023` — the exact-path table MUST be "matched over a dependency listing **that
  includes the standard library**". It is now matched over a direct-importer graph with a
  standard-library exemption.
- The `agent-package-scaffold` delta's own "Not modified" table (`spec.md:32`) asserts this table is
  "**Byte-unchanged.** AG-22 adds no network or filesystem reach" — the data is byte-unchanged, but the
  semantics around it are not, and the delta claims no such amendment.
- **No spec or design text authorizes it.** `grep -rn -i 'direct-importer|import graph|forcedStandard|productionImportGraph' specs/ design.md` → **no matches**.

The underlying problem is real (`otel/trace@v1.44.0`'s `auto.go` calls `os.Getenv`) and the *vetted-importer*
half (`forcedStandardLibraryImporters`) is a sound, narrow fix. Only the blanket standard-library
exemption needs removing — `io/fs` is the sole path that relied on it, and it can take its own vetted
entry. This is the failure shape the phase brief named: a guard widened until it stops catching the class
that includes the widener's own change.

### CRITICAL-2 — `R-AGO-007`'s exactly-once span lifetime is violated on a reachable path; run and turn spans leak

**Where**: `harness.go:592` (open) vs `:393`, `:425`, `:974` (close); `loop.go:351` (open) vs 8 separate close sites; `compaction.go:300`/`:493` vs `:160`, `:432`, `:518`, `:526`.

`R-AGO-007` (`spec.md:211`): *"the closing obligation MUST be discharged **uniformly for all exits** by a
finalizer **registered at open, never at individual return sites**. Every span MUST end **exactly once**
on every terminal path without exception: normal completion, every typed failure arm, every cancellation
shape, **a panic unwinding through the frame**, and the detached arm."*

Only the **tool** family satisfies this — `scheduler.go:470` registers a real `defer`. The run span closes
at 3 individual return-site funnels, the turn span at **8** call sites, the compaction span at 2. There is
no `defer` for any of them (`grep -n 'defer func' loop.go compaction.go harness.go` finds only an unrelated
`recover()` and a channel close).

The consequence is not theoretical. A panic unwinding through `Turn`/`Harness.Run` is reachable through the
deliberately-**unrecovered** mutating `PreRequest` hook (a posture `agent-v1-scope` `R-AGS-016` records by name).
Driven with a scratch test (created, run, deleted; worktree clean):

```
ZZRESULT panic recovered by caller: zz deliberate hook panic
ZZRESULT Started=2 Ended=0
ZZRESULT LEAK: tracetest: 2 span(s) not ended exactly once:
  span 0 (name "invoke_agent") ended 0 time(s), want exactly 1;
  span 1 (name "turn")         ended 0 time(s), want exactly 1
```

Both spans leak. `R-AGO-007` says such a case "MUST fail the suite" — it does not, because
`S-AGO-051`'s run table is only "normal terminal, each typed failure arm, and cancellation"; panic appears
only in `S-AGO-052`'s **tool** table, which is the one family that happens to be correct. This is a defect
sitting exactly between two satisfied scenarios, and it is the same unbounded-lifetime shape AG-21 was
chartered to remove.

---

## MAJOR

### MAJOR-1 — The attribute vocabulary is essentially unasserted: 3 keys of 14 appear anywhere in the tests

`grep -oh 'gen_ai\.[a-z_.]*|cachicamas\.[a-z_.]*|error\.type' observability_*_test.go` yields exactly
three distinct keys: `cachicamas.run.id`, `gen_ai.tool.call.id`, `cachicamas.compaction.id`.
Consequently these scenarios have **no covering assertion at all**:

| Scenario | Claim | Status |
|---|---|---|
| `S-AGO-013` | recorded key set is a **subset** of the vocabulary; no other key appears | ❌ UNTESTED |
| `S-AGO-017` | presence-typed attrs **absent**, never `""`/`0`/`false` | ❌ UNTESTED |
| `S-AGO-018` | span status error/ok; description is the category name **only** | ❌ UNTESTED (no `SetStatus`/`codes.` assertion exists in any observability test) |
| `S-AGO-021` | `error.type` appears only on brackets with a typed failure, value = category name | ❌ UNTESTED |
| `S-AGO-023` | Layer 1 request span recorded alongside; key sets compared | ❌ UNTESTED |

`R-AGO-002`'s "no key outside this set MAY appear, under any name, on any path" is currently defended by
nothing. (I confirmed by grep that Layer 1's eleven keys are absent from Layer 2 production source — but
that is my inspection, not a shipped guard, so `S-AGO-020` is evidenced only statically.)

### MAJOR-2 — `S-AGO-014` value-equality is 3/20 rows, and it is the charter's headline acceptance

`R-AGO-002`'s table has 20 rows with a "MUST equal — exact event accessor" column; `S-AGO-014` requires the
comparison "row by row, per the table above". Three rows are asserted
(`observability_nesting_test.go:97`, `:173`, `:256`). The charter clause this discharges is
`0003:2080` — *"attributes from the decided vocabulary with values equal to the corresponding events'"* —
so the milestone's own headline acceptance is 15% covered. Status: **PARTIAL**, not compliant.

### MAJOR-3 — The denylist non-vacuity guard omits the reasoning kind the spec names explicitly

`observability_denylist_test.go:316-322` — `wantKinds` lists run/turn/tool/permission/compaction kinds and
**no reasoning kind**. `R-AGO-008` item 4 requires "every event kind the driven run structurally produces —
tool, **reasoning**, permission, compaction and failure kinds among them"; `S-AGO-072` repeats it.

The omission is load-bearing, not cosmetic: `reasoning` is one of the nine denied vectors, and the run
demonstrably produces the events. Measured with a scratch probe (deleted; tree clean):
`ZZRESULT reasoning: start=1 delta=1 end=1`. Today the reasoning-vector absence claim rests on a
precondition nothing asserts — if the reasoning deltas ever stopped flowing, that vector would pass
vacuously and no test would say so. That is exactly the vacuity `R-AGO-008` exists to prevent.

### MAJOR-4 — The parity proof compares event **kinds**, not values, contradicting `R-AGO-009`/`S-AGO-080`

`R-AGO-009` (`spec.md:253`): the two drained sequences "MUST equal, **element for element and value for
value**". `S-AGO-080` repeats "with the same values". `observability_parity_test.go:24-30,79-83` compares
only `Kind()` sequences, and its own comment (`:19-23`) concedes per-field equality "can never hold" because
the two runs mint distinct RunID/TurnID values.

The test is *right* and the requirement is unachievable as written. But the artifact ships with the
requirement claiming value equality and the scenario claiming value equality, so `S-AGO-080` archives as
false. Either `R-AGO-009`/`S-AGO-080` should be re-scoped to the kind sequence (plus the ID-independent
fields), or the comparison must project out the minted identifiers and compare the remainder.

### MAJOR-5 — The detached arm's own decided attributes are never asserted

`S-AGO-055` requires the detached tool span to end "with the error status, `cachicamas.tool.detached`
present and `true`, and `error.type` equal to that arm's typed cancellation category". The
`detached_wind_down` row (`observability_lifecycle_test.go:43`, `:222-266`) asserts only
`AssertAllEndedOnce()`, `Started() > 0`, and `Started() == Ended()`. Neither `cachicamas.tool.detached`
nor `error.type` is read anywhere in the suite. `S-AGO-056`'s "the goroutine holds no span handle" is
likewise argued in a comment (`scheduler.go:556-561`), not asserted.

The production code does set them (`scheduler.go:558-566` → `finalizeToolSpan(..., detached=true, ...)`),
so this is missing coverage rather than a defect — but D-D's decided behaviour is the one thing this
milestone chose to decide explicitly, and it is unguarded.

### MAJOR-6 — Compaction nesting, cardinality, and exit-table scenarios are untested

- `S-AGO-041` (compaction span's parent is the requesting bracket's span) — the covering subtest is named
  `compaction_span_recorded_standalone_root` (`observability_nesting_test.go:230`) and **never reads the
  parent**. ❌ UNTESTED.
- `S-AGO-019` (exactly one span covers the bracket; no turn span opened for it) — no count assertion, no
  turn-span-absence assertion. ❌ UNTESTED.
- `S-AGO-053` (a table of compaction exits — success tail and each failure arm) — the lifecycle table has
  **no compaction row** at all. ❌ UNTESTED.
- `S-AGO-054` (no span started for pre-Start-event refusals: run, turn, gated-out tool, compaction) — no
  covering test found. ❌ UNTESTED.

---

## MINOR

1. **`hksScopeFenceByteUnchangedFiles()` fully releases two files rather than filtering them.**
   `hooks_test.go:1511`/`:1512` — `import_boundary_test.go` and `scheduler.go` were **deleted** from the
   frozen list. Both releases are necessary and disclosed, but the same change demonstrates a narrower
   technique three times over (`hksFilterOutAG22TracetestFiles`, `filterOutLoopFiles`,
   `filterOutLoopHookFiles` — exact-filename suffix filters). Residual protection for `scheduler.go` is
   three narrow content guards (`TestScopeFence_S_TLS_020`, `TestScheduler_SourceGuard_PolicySlotNoTypeAssertion`,
   `TestScheduler_SourceGuard_NoErrgroupImport`), not byte identity. Note this is *how CRITICAL-1 went
   unnoticed*: `import_boundary_test.go` left the freeze in the same commit that rewrote its behaviour.
2. **`wantMinAttributeCount = 14` is a very loose floor.** `observability_denylist_test.go:300-307` sums
   **total** attributes across **all** spans of three runs and compares against the *unique-key
   cardinality* (14). The real total is far higher, so the floor cannot detect a family that stopped
   recording. A per-family or unique-key floor would bite.
3. **Stale RED-phase failure messages ship in green tests** — e.g. `observability_denylist_test.go:298`
   ("the run/turn/tool/compaction seams are not instrumented until Phases 5-8"),
   `observability_lifecycle_test.go:54`, `observability_parity_test.go:100`,
   `observability_nesting_test.go:247`. Harmless today; actively misleading when one of them next fires.

---

## SUGGESTION

1. `S-AGO-025` (values rendered through the tracing API's own value accessor, not a structure dump) is
   satisfied by `tracetest`'s corpus builder but is never asserted as its own claim; a one-line guard on
   the corpus renderer would close the AI-41 lesson explicitly.
2. `S-AGO-026`'s accessor-flow half is discharged by inspection. The `RecordError` half is genuinely
   guarded — `grep -rn RecordError src/agent/*.go` (non-test) matches only `observability.go:23`'s comment.

---

### Verified sound — audited and found no finding

| Target | Evidence |
|---|---|
| **Guard widenings are exact-filename, no wildcards** | All entries in `filterOutLoopFiles`, `filterOutLoopHookFiles`, `hksFilterOutAG22TracetestFiles` are `strings.HasSuffix(path, "/<exact>.go")`. |
| **All three git-diff substrate guards still bite** | Proved with **staged** plants (an untracked file proves nothing — `git diff <ref>` ignores it): modifying a still-frozen entry (`event.go`) → `TestHooks_ScopeFence…` FAILS at `hooks_test.go:1579`; a **non**-tracetest file under `src/agenttest/` → FAILS at `:1597` (filter does **not** over-match); a new file in `src/agent/` → both `TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched` FAIL. |
| **The widened allowlist did not go over-broad** | Planted and observed FAIL: `otel/sdk/trace` → forbidden-prefix rule, `import_boundary_test.go:213` (exactly what `S-AGP-035` claims); `database_administrator/src/application` → `:213`; `src/ai/openaicompat` → `:213`; `go.opentelemetry.io/otel` root getter → denied. |
| **`S-AGP-031`/`032`/`033` citations** | `import_boundary_test.go:147-169`: the three § D3 rows cite their rows; `otel/semconv` and `cespare/xxhash/v2` cite the **forced-closure clause plus the measurement** and cite **no** § D3 row; `semconv` is admitted as a **prefix**, with the version-bump rationale in place. Correct on every count. |
| **Closure measured independently** | `go list -deps -f '{{if not .Standard}}…' ./src/agent/...` → exactly the 10 claimed packages, no more. `S-AGO-091`/`S-AGO-092` hold. |
| **`S-AGO-090`** | Actual `import` statements (not comments) in Layer 2 production: only `otel/trace`, `otel/trace/noop`, `otel/attribute`, `otel/codes`. No root getter, no metric, no SDK, no exporter, no `otelslog`. |
| **ADR § D3 extension** | Landed at `docs/adr/0005…md:252-343`, immediately after the Layer 1 allowlist, with all four span families, the per-key accessor mapping, the eleven deferred Layer 1 keys naming AI-37, and the seven not-recorded items each with a reason. `S-AGO-010`, `S-AGO-022`, `S-AGO-030` hold. |
| **`S-AGO-011`** | The denylist paragraph appears twice and is **byte-identical** after whitespace normalization (verified programmatically). Unweakened, no carve-out. |
| **`S-AGS-068` guard is real** | `observability_scope_test.go:192-330` enumerates the shipped surface via `go/parser` **and** `reflect`, pins `Harness.TracerProvider`'s type to the OTel **API** interface and its kind to `Interface`, scans production sources for global telemetry symbols, and re-checks the `go.mod`/`go.sum`/`src/ai` freeze. It checks the shipped surface, not prose — priority target 4 is satisfied. I re-read every phrase changed by the `S-AGS-066` amendment and found **no** scenario still restating the old "no telemetry sink" claim. |
| **Denylist proof mechanics** | Needles are runtime-concatenated (`observability_denylist_test.go:55-63`), `sweep.SelfTest` runs as a positive control **before** the clean scan (`:190`), `AssertAllEndedOnce()` precedes the corpus read (`:291`), corpus-non-empty and every-needle-non-empty floors both present. Only the event-kind coverage floor is incomplete (MAJOR-3). Run A's Defer→Wake→Allow restructuring genuinely strengthened the run — `permission_decision_required` now fires and is asserted. |
| **Two "unreachable" defensive branches** | `Turn`'s `NewTurnStart`-failure gap and `emitPreStreamAbort`'s `perr != nil` fallback remain unreached, as disclosed. I did not defeat them; they are unproven-but-disclosed, not a finding. |

---

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | Full RED/GREEN table in apply-progress for every phase |
| All tasks have tests | ✅ | 45/45 in-scope tasks |
| RED confirmed (test files exist) | ✅ | All 5 observability test files present |
| GREEN confirmed (tests pass) | ✅ | 1451/1451 pass on an uncached `-count=1` run |
| Triangulation adequate | ⚠️ | Lifecycle table has 7 rows; the attribute vocabulary has 3 assertions for 14 keys (MAJOR-1/2) |
| Safety net for modified files | ✅ | Substrate guards re-proved biting after every widening |

**Assertion quality**: no tautologies, no ghost loops, no orphan-empty assertions, no mock-heavy files.
Every needle is runtime-assembled and proven to bite via `sweep.SelfTest`. The weaknesses are *missing*
assertions (MAJOR-1…6), not *trivial* ones.

### Quality Metrics
**Linter**: ✅ 0 issues · **Vet**: ✅ clean · **Vulnerabilities**: ✅ 0 findings · **Coverage**: ➖ not run (no threshold configured for this change)

---

### Verdict

**FAIL** — CRITICAL-1 (a shipped guard now passes `os/exec` and `crypto/tls` in Layer 2 production, which
`main` rejected, unauthorized by any spec text and contrary to `NFR-AGO-003`) and CRITICAL-2 (`R-AGO-007`'s
exactly-once MUST is violated on a reachable panic path, leaking both the run and turn spans) each block
archive on their own. Six MAJOR findings are scenarios whose tests do not assert what the scenarios say —
including the charter's own headline acceptance clause. Everything apply reported numerically was accurate;
these are gaps its report did not surface.
