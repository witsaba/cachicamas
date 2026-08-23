# Tasks: CH-01 — Scaffold the archetype package and make its import boundary bite

Change `cachicamas-chat-package-scaffold`. Executes design AD-1…AD-9 plus its two stated amendments (the recursive walk over proposal D-6; the `otelslog` allowlist grant, Engram #3741) and the whole-table `R-AGP-003` deviation (Engram #3745) — no D-row/AD-row reopened. All `file:line` cites merge base `419a4291`. Citation notation: `0005:NNN` cites the milestone doc, `adr:NNN` cites `docs/adr/0005-promote-agent-stack-to-own-module.md`. Two leaves per `0005:230-276`: **CH-01.1** `[mechanical]` (Phase 1) is exempt from red-green but not from check evidence; **CH-01.2** `[guard]` (Phases 2–4) closes only on recorded RED against a planted violation, then green, per R-CPB-010.

**Evidence artifacts**: `apply-progress.md` records every RED/GREEN transcript, the negative control, the probe-build evidence, and the final-gate run (Phases 2–4) as apply executes them. `verify-report.md` is `sdd-verify`'s own downstream artifact: it re-confirms/cites that same evidence and the requirement/scenario coverage; it is not produced by this task list.

**Scoping rule, corrected (R-CPB-010, S-CPB-091, Engram #3739)**: every bite run in Phase 2 uses `go test -race -count=1 -run TestChatArchetype -v ./src/agent/` (from `backend/agent`) for **focus and isolation of the check under test, and for that reason only**. The repo root's `go.work` (`go.work:3-7`) uses all three backend modules, so in workspace mode a sibling module's packages import without a `require` in `backend/agent/go.mod`, and the build list is the union of all used modules' requirements — both planted scratches **build**, and an unscoped `./...` run would reach the guard too. **No recorded evidence may claim otherwise** — a false toolchain claim archived as the reason would violate S-CPB-091. What remains true and must be recorded *separately*: the `go.mod`/`go.sum` byte-freeze (R-CPB-009) binds **committed** code — a real committed import would make `go mod tidy` add a `require`; scratch files are transient and never reach that constraint (S-CPB-096). Every planted scratch file MUST use a **blank import** (`import _ "…"`) so it compiles with no use site while staying visible to `parser.ImportsOnly` and `go list -deps` (S-CPB-095) — this governs every bite task below (2.1–2.3, 2.5–2.7, 2.10).

## Review Workload Forecast

The proposal's original ~965-line estimate is superseded — restating it here would be quietly wrong by roughly 2×.

| Field | Value |
|---|---|
| Estimated changed lines | **~1900–2000**, not ~965 — see basis below |
| Review budget named at session start | 1000 counted lines, with extension/`size:exception` pre-approved by the user |
| Budget vs. the 1000-line figure | **Exceeds it substantially.** Pre-authorised (`size:exception`) — not a stop-and-ask — but the magnitude is recorded plainly, not reassuringly |
| 400-line default budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

**Basis.** The planning artifacts alone already total ~1080 lines: `proposal.md` 301, `specs/chat-package-boundary/spec.md` 290, `design.md` 189, `explore.md` 155, `specs/agent-package-scaffold/spec.md` (the delta) 59, this `tasks.md` file. That is before `apply-progress.md`, `verify-report.md`, the ~180–205 lines of authored Go (the guard extension alone is now ~140–160 per design, vs. the proposal's 135), and the two-line doc 0005 edit. CH-00's comparable PR — same shape, two Go files plus one guard extension plus SDD artifacts — landed at 1915 insertions. Roughly **1900–2000 counted lines** is the realistic projection.

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

Splitting is rejected on a mechanical ground, not re-opened here: check 6's vacuity floor (AD-5) requires the archetype tree to exist, so a CH-01.1-only PR would ship two packages with no guard over them. The two nodes ship as one deliverable.

### Suggested Work Units

| Unit | Goal | PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | CH-01.1 + CH-01.2 complete: two packages, check 6 (full allowlist incl. otelslog), both spec deltas, doc 0005 bookkeeping | PR 1 (single) | `cd backend/agent && go test -race -count=1 -run TestChatArchetype -v ./src/agent/` (bite proofs); `go test -race -count=1 ./...` (final gate) | N/A — backend unit-test guard, no compose/browser harness applies | Pre-merge: discard branch. Post-merge: single-PR revert; the change alters no `go.mod`/`go.sum`/`Makefile`/`.golangci.yml`, no schema, no wire, no runtime behavior |

## Phase 1: CH-01.1 — Package scaffold (mechanical; check evidence, not red-green)

- [ ] 1.1 Create `backend/agent/src/chat/doc.go` — `package chat`, doc comment only (AD-8: Layer 3 position per ADR 0009 § D2, created by CH-01.1, empty of policy until CH-02, enumerates the forbidden closure AND the permitted OTel surface — the API packages plus the `otelslog` bridge, `adr:243`, under the ADR 0009 § D2 substitution — and names check 6 as enforcement). → R-CPB-001, S-CPB-001, S-CPB-002, S-CPB-004, S-CPB-006.
- [ ] 1.2 Create `backend/agent/src/cmd/chat/main.go` — `package main`, doc comment, `func main() {}` empty body (AD-8: composition root, the only package of the archetype permitted the OTel **SDK and exporters** — ADR 0005 § D3's `cmd/` column, `adr:242`, which is generic and needs no substitution — outside check 6's scanned set, CH-04.1 owns wiring. Where the comment invokes the substitution convention, write "ADR 0009 § D2" in full — a bare "§ D2" after an ADR 0005 citation misreads as ADR 0005's own section). → R-CPB-001, S-CPB-003, S-CPB-005, S-CPB-006.
- [ ] 1.3 Check evidence: `go build ./...` and `go build ./src/cmd/chat` exit 0; record build output in `apply-progress.md` (design step A). Confirm both new trees sit as siblings of `src/agent`, never nested. → S-CPB-003, S-CPB-073.
- [ ] 1.4 Check evidence for R-CPB-002: confirm no test asserting "nothing imports `…/src/cmd/chat`" is written or committed (the property is a Go-language guarantee, stated in 1.2's comment, never a test). Demonstration MAY be recorded once — attempt a throwaway build importing `…/src/cmd/chat`, observe toolchain rejection, record output, discard the file, do not commit it. → S-CPB-010, S-CPB-011.

## Phase 2: CH-01.2 — Guard TDD sequence (scoped runs for focus only; design steps B–G plus required proofs not in that table)

- [ ] 2.1 (design step B, RED-1) Plant `backend/agent/src/chat/scratch_forbidden_module.go` with `import _ "github.com/cachicamas/backend/workspace_syncer/src/domain"` (a real package of that module). Run `go build ./...` once and record exit 0 — the first half of S-CPB-096's probe evidence, distinct from the separate go.mod-drift claim. Write check 6 `TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault` calling `t.Parallel()`; `chatArchetypeDirs` resolver; the per-declared-root **recursive** walk (`filepath.WalkDir`, skip `testdata`, skip `_test.go`) as a stated amendment to D-6 (S-CPB-053); the tree-relative file-naming helper (`filepath.Rel`, `Rel` error → `t.Fatalf`, never `filepath.Base`, S-CPB-046); `chatArchetypeForbiddenPrefixes` + `chatArchetypeMatchForbidden` (AD-2, forbidden-first); the **full, complete** `chatArchetypeAllowedPrefixes` **including `go.opentelemetry.io/contrib/bridges/otelslog`** (`adr:243` under the ADR 0009 § D2 substitution); the `go list std` helper that `t.Fatalf`s with the subprocess's stderr on failure (AD-6, NFR-CPB-003); the per-declared-root file-count floor (AD-5). Append AD-7's package-comment amendment row (enumeration "FIVE"→"SIX"; the CH-01 amendment section recording recursion, the citation rule, `src/cmd/chat`'s exclusion, the `database_administrator`/`workspace_syncer` absence and why, the otelslog grant, **the whole-table deviation from `agent-package-scaffold` R-AGP-003's same-commit rule with its declaration-vs-incremental reasoning and its two rejected alternatives** (S-CPB-038), and the zero-hop gap). Scoped run → FAIL with AD-3's tree-relative message. Record full output, labelled with the corrected scoping rationale (focus/isolation only — see rule above). Track the running line count added to `import_boundary_test.go` against the ~140–160-line estimate; note it in `apply-progress.md`. → R-CPB-004, R-CPB-005 (properties 1–3), S-CPB-012, S-CPB-023, S-CPB-030, S-CPB-031, S-CPB-032, S-CPB-034, S-CPB-035, S-CPB-036, S-CPB-037, S-CPB-038, S-CPB-039, S-CPB-040, S-CPB-041, S-CPB-042, S-CPB-043, S-CPB-045, S-CPB-046, S-CPB-052, S-CPB-053, S-CPB-060, S-CPB-062, S-CPB-082, S-CPB-091 (governs 2.1–2.10), S-CPB-096 (half 1).
- [ ] 2.2 (defeat test, S-CPB-044) With 2.1's scratch file still present, temporarily add `database_administrator`/`workspace_syncer` rows to `chatArchetypeForbiddenPrefixes`. Rerun the scoped test: the failure now cites a *named* forbidden prefix, so property 3 (deny-by-default framing) is lost — S-CPB-042 FAILS under this drafted table. Record both the drafted-table run and its failing property. Revert the table to its recorded shape (no other-backend-module rows). → S-CPB-044.
- [ ] 2.3 (openaicompat ordering bite) Plant `backend/agent/src/chat/scratch_openaicompat.go` with `import _ "github.com/cachicamas/backend/agent/src/ai/openaicompat"` (or a real subpackage). Scoped run → FAILS on the forbidden-prefix rule, not the allowlist — proving forbidden-first ordering carves the vendor adapter out of the `…/src/ai` allowlist prefix. Record output. Delete the scratch file. → S-CPB-033.
- [ ] 2.4 (design step C) Delete `scratch_forbidden_module.go`. Scoped run → PASS. Record output.
- [ ] 2.5 (design step D, RED-2a) Plant `backend/agent/src/chat/scratch_otel_sdk.go` with `import _ "go.opentelemetry.io/otel/sdk/trace"`. Run `go build ./...` once and record exit 0 — the second half of S-CPB-096's probe evidence; record it distinctly from the separate, still-live go.mod-drift claim (a *committed* import would trigger `go mod tidy`, S-CPB-096's second clause). Scoped run → FAIL naming the file (tree-relative) and the path. Record output. → S-CPB-050, S-CPB-096 (half 2).
- [ ] 2.6 (design step E, 2b negative control) Plant the identical blank import in `backend/agent/src/cmd/chat/scratch_otel_sdk.go` (root outside the scanned set, AD-1). Scoped run → check 6 PASSES with both scratch files present. Record the *passing* run as evidence in its own right, then delete both otel scratch files. → S-CPB-051.
- [ ] 2.7 (design step F, vacuity floor) With scratches removed, mistype `chatArchetypeDirs`' map entry to a nonexistent path (e.g. `…/chatx`). Scoped run → FATAL naming the mistyped directory and import path. Record output, then revert the entry to the correct path. → S-CPB-061.
- [ ] 2.8 (R-CPB-008 proof 0, S-CPB-075) After 2.1–2.7 land, re-verify D-7's per-guard effect table **row by row against the post-apply tree**, re-resolving every citation line number rather than trusting the pre-insertion numbers (check 6's insertion shifts lines below it): Layer 1's forward guard's closed pattern list; checks 1/3/4 scoped to `layer2Pattern` alone; check 2's pattern set; check 5's directory map; the `…/src/cmd` forbidden row (now armed, not disarmed). Record each row's re-verified citation. → S-CPB-075.
- [ ] 2.9 (R-CPB-008 proof 1 / AD-9.1 hunk audit) `git diff 419a4291 -- backend/agent/src/agent/import_boundary_test.go`; classify every hunk — each must lie in the package comment or a wholly-new `chatArchetype*`/check-6 declaration; zero hunks inside an existing `Test*` func, table, or shared helper. `git diff 419a4291 -- backend/agent/src/ai/import_boundary_test.go` → empty. Confirm the existing `…/src/cmd` forbidden-prefix row (`:204`) is unchanged. Record both diffs. → S-CPB-070, S-CPB-071, S-CPB-073, S-CPB-074.
- [ ] 2.10 (R-CPB-008 proof 2 / design step G / AD-9.2 re-plant bite) Plant `backend/agent/src/agent/scratch_os_exec.go` with `import _ "os/exec"`. Run `go test -race -count=1 -run 'TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage|TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport' -v ./src/agent/` — both RED, recorded. Delete the scratch file. Checks 1, 2, 5's armed-ness rests on proofs 0–1 (2.8–2.9: bodies byte-identical to shipped, already-bitten code). → S-CPB-072.

## Phase 3: Non-code invariant

- [ ] 3.1 Assert `git diff 419a4291 -- backend/agent/go.mod backend/agent/go.sum backend/agent/Makefile backend/agent/.golangci.yml` is empty for all four paths. Record output. → R-CPB-009, S-CPB-080.

## Phase 4: Final gate (design step H — never `make all`, its fmt step rewrites committed files)

- [ ] 4.1 Confirm zero scratch files remain. `go clean -testcache`; `cd backend/agent && go test -race -count=1 ./...` → green, wall-clock recorded (a `(cached)` or sub-second result is not evidence). This run also exercises the nine shipped `go.mod`/`go.sum` merge-base-diff-empty assertions. → S-CPB-081, S-CPB-093, R-CPB-010.
- [ ] 4.2 `make lint` → 0 issues; `gofmt -l` → empty output. `make all` is NOT run. → S-CPB-094.
- [ ] 4.3 Scenario 3's second clause, concrete: `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` outputs **exactly** `backend/agent/src/chat/doc.go` and `backend/agent/src/cmd/chat/main.go` (no `_test.go`, no second guard); `git status --porcelain -- backend/agent/src/chat backend/agent/src/cmd` → empty. Since those are the only two files and neither imports anything, this listing also confirms no archetype allowlist entry is exercised by a committed import at CH-01's own merge state (S-CPB-039). `git diff --stat 419a4291` confirms `import_boundary_test.go` is the only changed file under `src/agent`. → R-CPB-003, R-CPB-010, S-CPB-020, S-CPB-021, S-CPB-022, S-CPB-039, S-CPB-092.
- [ ] 4.4 (budget gate) Sum counted changed lines (additions + deletions) across every touched file in this PR; record the total against the 1000-line figure named at session start. Pre-authorised via `size:exception` — not a stop condition — but the real total is expected to land near the ~1900–2000 projection above and must be recorded plainly in `apply-progress.md`, not narrowed to look closer to 1000.

## Phase 5: Doc 0005 bookkeeping

- [ ] 5.1 Re-resolve, then tick the checklist row at `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:981` (`- [ ]` → `- [x]`, "closed by CH-01.2" already present).
- [ ] 5.2 Re-resolve, then update the status line at `0005:3` from "1 of 12" to "2 of 12".

## Coverage — every `S-CPB` scenario mapped to a closing task

Re-derived from `specs/chat-package-boundary/spec.md` as currently written (not from any prior list of ids).

| Requirement | Scenarios | Closing task(s) |
|---|---|---|
| R-CPB-001 | S-CPB-001…006 | 1.1, 1.2, 1.3 |
| R-CPB-002 | S-CPB-010…012 | 1.4, 2.1 |
| R-CPB-003 | S-CPB-020…023 | 2.1, 4.3 |
| R-CPB-004 | S-CPB-030…039 | 2.1, 2.3 |
| R-CPB-005 | S-CPB-040…046 | 2.1, 2.2 |
| R-CPB-006 | S-CPB-050…053 | 2.1, 2.5, 2.6 |
| R-CPB-007 | S-CPB-060…062 | 2.1, 2.7 |
| R-CPB-008 | S-CPB-070…075 | 1.3, 2.8, 2.9, 2.10 |
| R-CPB-009 | S-CPB-080…082 | 2.1, 3.1, 4.1 |
| R-CPB-010 | S-CPB-090…096 | 2.1–2.10 (per-task recording), 4.1, 4.2, 4.3 |

Reverse check: every task above (1.1–5.2) carries an inline `→` mapping to at least one `R-CPB`/`S-CPB` id, a design AD-id, or a proposal D-id; Phase 5's two tasks close proposal item 7 and success criterion 11 (doc 0005 bookkeeping) rather than a `CPB` scenario; Phase 4's 4.4 is a budget-discipline task with no closing scenario, by design.
