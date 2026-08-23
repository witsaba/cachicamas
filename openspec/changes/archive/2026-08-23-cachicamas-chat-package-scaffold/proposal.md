# Proposal: CH-01 — Scaffold the archetype package and make its import boundary bite

| | |
|---|---|
| **Change** | `cachicamas-chat-package-scaffold` |
| **Milestone** | CH-01 (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:230-276`) · Wave 0 · **2 of 12** once merged |
| **Closes** | R-06 (the boundary half), R-07 (`0005:232`) |
| **Nodes** | CH-01.1 `[mechanical]` (`0005:243-247`) · CH-01.2 `[guard]` (`0005:249-276`) |
| **Depends on** | CH-00 — merged (PR #190, `419a4291`). **Blocks** CH-02, CH-03 (`0005:239`) |
| **Worktree / base** | `cachicamas-worktrees/feat-chat-archetype-wave0-ch01`, branch `feat/chat-archetype-wave0-ch01`, based on `origin/main` @ `419a4291` |
| **Artifact store** | hybrid (`openspec/changes/…` + Engram `sdd/cachicamas-chat-package-scaffold/*`) |
| **Delivery** | single-pr · review budget 1000 counted lines (extension pre-authorised) |
| **Evidence runner** | `cd backend/agent && go test -race -count=1 ./...`, wall-clock recorded, plus `make lint`. **A `(cached)` result is not evidence.** `make all` MUST NOT be run — its fmt step rewrites committed files |
| **Prefix** | **`CPB`**, capability `chat-package-boundary` — verified free this phase: 0 hits for `-CPB-` across the whole worktree |
| **Exploration** | Engram `sdd/cachicamas-chat-package-scaffold/explore` (#3724) · `openspec/changes/cachicamas-chat-package-scaffold/explore.md` |
| **Binding decisions** | Engram #3725 (four user decisions, 2026-08-23) — settled scope, reproduced as D-1…D-4 below |

> Every `file:line` below was re-resolved by reading the file in **this worktree** during this phase. Where the exploration and this phase disagree, this phase's reading governs.

## Intent

The chat archetype is, today, a paragraph. `backend/agent/src/chat/` does not exist, and `backend/agent/src/cmd/` does not exist **at all** — this module has never had a composition root of any kind (`chat-archetype-contract/spec.md:149`, restating `0005:228`). Every architectural claim about where the archetype may and may not reach is therefore currently unfalsifiable: nothing on disk can violate a rule, so no rule is proven.

Two things break the moment that changes, and CH-01 exists to make sure they break loudly rather than silently:

1. **The archetype's position has a forbidden closure that nothing enforces.** `0005:241` names it: the OTel SDK anywhere but the root; any `src/cmd/…` package from below the root; any Go package of `database_administrator` or `workspace_syncer` (network only, R-07); and Layer 2 internals. Layer 2's own guard (`backend/agent/src/agent/import_boundary_test.go`) scans `src/agent/...` and two sibling test trees (`:150`, `:165-168`) — **no pattern in it touches `src/chat` or `src/cmd/chat`, and neither string appears anywhere in the file.** CH-02 is the first milestone that gives the archetype real imports; if it arrives before the boundary bites, the first violation lands unopposed.
2. **A promoted requirement becomes false on merge.** `agent-package-scaffold/spec.md:35` (R-AGP-001) states `backend/agent/src/coding/` and `backend/agent/src/cmd/` "MUST NOT exist in any form, including as an empty directory", and `S-AGP-003` (`:41`) requires `test -e backend/agent/src/cmd` to exit non-zero. CH-01.1 creates `src/cmd/chat/`. That is not a future tension: it is a promoted spec that is wrong the day this PR merges unless the same PR repairs it.

CH-01 is also the milestone where CH-00 stops being paper. `chat-archetype-contract` R-CHT-013 (`spec.md:226`) says it in its own words: *"The earliest honest mechanical binding is CH-01.2's import guard, not here."*

## Scope

### In scope

1. **`backend/agent/src/chat/doc.go`** — the archetype package, `package chat`, documentation and **nothing else**: no type, no constant, no function, no variable (D-5).
2. **`backend/agent/src/cmd/chat/main.go`** — the composition root, `package main`, a documented no-op `func main()` and nothing else (D-5).
3. **One extension to the single existing guard** at `backend/agent/src/agent/import_boundary_test.go`: a sixth check — a per-file, deny-by-default AST scan over the archetype tree — plus its allowlist, its forbidden table, its vacuity floor, and the package comment's own amendment row (D-6).
4. **A spec delta on `agent-package-scaffold`** amending R-AGP-001 and S-AGP-003 to scope their `src/cmd`/`src/coding` prohibition to AG-03's own merge state (D-2).
5. **A new capability `chat-package-boundary`**, prefix `CPB`, promoted at archive: the two packages' identity, the archetype's forbidden closure, the guard's extension-not-clone property, the message contract, and the vacuity floor.
6. **Recorded bite evidence** for all three CH-01.2 scenarios, with every scratch file removed before merge (D-7).
7. **Doc 0005 bookkeeping** in the same PR: tick the checklist row at `0005:981`, move the status line at `0005:3` to **2 of 12**.

### Out of scope

| Excluded | Owner / reason |
|---|---|
| Any behaviour in either package — ports, projection, wiring, env reads | `0005:240` "the packages ship empty of policy"; the archetype's conversation is CH-02, the root's wiring is CH-04.1 |
| A second, self-contained guard inside the archetype tree | `0005:240` and `agent-layer3-handoff` R-L3H-002 (`spec.md:73-95`) — the existing guard is *extended*, never cloned. `S-L3H-013` is the assertion |
| Any change to `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | D-1 — **nine** shipped Layer 2 assertions require the merge-base diff of `go.mod`/`go.sum` to be empty; see D-1 for all nine citations |
| An amendment to ADR 0005 § D1 / § D3 adding `src/chat` or `cmd/chat` rows | D-3 — read-with-substitution under ADR 0009 § D2, the rule CH-00 already promoted as `chat-archetype-contract` R-CHT-011 (`spec.md:196-200`) |
| A **transitive** (`go list -deps`) closure check over `src/chat/...` | **Deferred to CH-02** — stated with its cost in D-6 "The deferred second mechanism". Not silently omitted |
| An "only the root reads the environment" guard | CH-04.2 (`0005:304`) — a distinct property with its own node and its own bite |
| A machine-checked doc-row table for `src/chat/doc.go` | `agent-package-scaffold` R-AGP-002 (`spec.md:45-51`) is scoped to `backend/agent/src/agent/doc.go`. CH-01 has no doc-row node |
| A per-binary `build` target for `cmd/chat` in `backend/agent/Makefile` | D-1 forbids the Makefile edit; the existing `go build ./...` already compiles a `package main`. The stale Makefile comment naming `cmd/cachicamas` is recorded as F-1, not repaired |
| Retiring the frontend's offline literal | CH-05.2 (`0005:989`) — needs a recorded spec delta, never a silent deletion |

## Resolved decisions

### D-1 — The OTel-SDK half of the bite is a source-level scan; no dependency is added

**Decided** (user, Engram #3725 item 1). The guard reads the archetype tree's **file bytes** and denies `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters` and `go.opentelemetry.io/contrib/bridges/otelslog` textually — the same AST-scan shape checks 4 (`import_boundary_test.go:581-604`) and 5 already use. `backend/agent/src/cmd/chat` is simply **outside the scanned set**, which is exactly what CH-01.2 scenario 2's *"the same import inside the composition root passes"* means.

**Why no `go.mod` change is permitted.** Shipped Layer 2 tests assert that `git diff <merge-base with origin/main> -- backend/agent/go.mod backend/agent/go.sum` is **empty**, and they run on every branch. **Corrected this phase: there are nine such call sites, not six** — the binding decision record named six; grepping the exact argument pair `"backend/agent/go.mod", "backend/agent/go.sum"` across `backend/agent/` finds `hooks_test.go:1650`, `hooks_test.go:2512`, `loop_test.go:1438`, `observability_scope_test.go:295`, `compaction_stream_test.go:166`, `context_strategy_test.go:1220`, `scope_fence_test.go:78`, `cost_turn_emission_test.go:590`, `loop_hook_test.go:990`. The correction strengthens the decision rather than altering it. The base ref resolves through `hksResolveBaseRef` (`hooks_test.go:1735`, falling back to `git merge-base HEAD origin/main`). Adding any dependency turns the agent suite red in nine places. `go.opentelemetry.io/otel/sdk` is absent from `backend/agent/go.mod`, `go.sum` and the root `go.work.sum`.

**Consequence.** A scratch file importing the SDK will not compile as part of a real build (the module has no such dependency), but the AST scan reads bytes with `go/parser` in `parser.ImportsOnly` mode and never resolves the import — the same reason check 4 can deny a family without the family being present. The bite is therefore reachable without touching the module graph.

**Rejected — install the SDK and use a closure check.** It reddens nine unrelated assertions for a property a byte scan already proves.

### D-2 — The R-AGP-001 contradiction is repaired by a spec delta inside this change

**Decided** (user, Engram #3725 item 2). This change carries a delta on capability `agent-package-scaffold` amending **R-AGP-001** (`spec.md:35`) and **S-AGP-003** (`spec.md:41`) so their `src/coding`/`src/cmd` non-existence clause is scoped to AG-03's own merge state rather than stated as a live invariant.

**Why a delta and not a recorded-not-repaired register entry.** CH-00's F-1/F-2/F-3 were inconsistencies between artifacts with no forcing merge. This is different in kind: `S-AGP-003` becomes literally false the moment `src/cmd/chat/` lands. The spec file's own header (`spec.md:6`) calls its invariants **live** — "the invariants below hold for the lifetime of Layer 2, not only at the moment AG-03 merged" — and `spec.md:21` states the amendment rule this delta follows: a later milestone that needs to change one of these invariants "amends **this file**, in the same pull request".

**Verified this phase.** No live Go test enforces `S-AGP-003`; it is spec text only. This is a spec repair, not a code change.

**Scope of the delta, pinned.** R-AGP-001's `backend/agent/src/agent/` clause ("exists, declares nothing") is **not** weakened. R-AGP-001's own `go.mod`/`go.sum`/`Makefile`/`.golangci.yml` clause is likewise AG-03-scoped, but D-1 independently forbids that half for CH-01 anyway, so this change does not rely on the delta for it.

### D-3 — ADR 0005 is not amended; the guard cites row 3 plus the substitution

**Decided** (user, Engram #3725 item 3). ADR 0005 § D1 row 3 (`…/src/coding`, Layer 3) and § D3's `L3 coding` column are read as **"any Layer 3 archetype"** under ADR 0009 § D2 (Layer 3 is a position, not an occupant) — the same read-with-substitution CH-00 already promoted as `chat-archetype-contract` R-CHT-011 (`spec.md:196-200`).

Every failure message the new check emits cites **ADR 0005 § D1 row 3 (or § D3) *plus* the substitution**, so a reader five milestones later meets the substitution as a stated convention rather than as an unexplained analogy. No ADR delta; no per-archetype rows added to ADR 0005.

**Rejected — amend ADR 0005 to add `src/chat`/`cmd/chat` rows.** An ADR fixes decisions, not occupants; a per-archetype row would have to be added again for every future archetype, and ADR 0009 § D2 already generalises the position.

### D-4 — "Layer 2 internals" means the test substrate plus the vendor adapter

**Decided** (user, Engram #3725 item 4). The fourth item of `0005:241`'s forbidden closure resolves as follows:

| Path | Archetype **production** closure | Archetype **test** closure |
|---|---|---|
| `…/src/agent` | **permitted** (ADR 0005 § D1 row 3, read with substitution — row 3 permits rows 1–2) | permitted |
| `…/src/ai` | **permitted** (same) | permitted |
| `…/src/ai/openaicompat` | **denied by name** — vendor adapter, mirrors `import_boundary_test.go:205` | denied |
| `…/src/agenttest` | **denied by name** — Layer 2's test substrate | *(not needed at CH-01; CH-02 decides)* |
| `…/src/apptest` | **denied by name** | *(not needed at CH-01)* |
| `…/src/layer3handoff` | **denied by name** | **admissible** — R-10 (`0005:66`) requires every harness consumer to test against the AG-23 scripted kit |

At CH-01 the archetype ships no test file, so only the production column has a scanned subject. The test column is stated here because the guard's own table must record the asymmetry now rather than have CH-02 invent it.

### D-5 — Package identifiers and file shape, derived from repo convention

**`backend/agent/src/chat/` → `package chat`.** Derivation, verified by reading: this module names every package after its directory's last element — `src/agent/doc.go:74` `package agent`, `src/ai/doc.go:127` `package ai`, `src/agenttest/doc.go:87` `package agenttest`, `src/apptest/doc.go:24` `package apptest`, `src/handoff/doc.go:6` `package handoff`, `src/layer3handoff/doc.go:7` `package layer3handoff`, `src/ai/openaicompat/doc.go:692` `package openaicompat`, `src/ai/openaicompat/openrouter/doc.go:56` `package openrouter`, `src/ai/internal/retry/doc.go:12` `package retry`. **Nine for nine — no exception exists in this module.**

**`backend/agent/src/cmd/chat/` → `package main`,** forced by Go: a buildable command is a main package, matching the two sibling modules' own roots (`backend/workspace_syncer/src/cmd/server/main.go:9`).

**File shape.** `doc.go` is this module's universal convention — **every** package under `backend/agent/src/` has one (glob-verified: nine `doc.go` files across nine packages, listed above). The archetype package therefore ships `doc.go` and nothing else, following R-AGP-001's own AG-03 precedent for `src/agent`: documentation, no declarations.

The root ships `main.go` with a package doc comment, `func main()` with an **empty body**, and a comment naming CH-04.1 as the owner of its wiring. `func main()` cannot be omitted — a `package main` without it does not build.

**Rejected — a fail-fast stub that writes "not wired yet" to stderr and exits non-zero.** It is more honest to a human who runs the binary, but it is behaviour, and `0005:240` puts behaviour out of scope. The doc comment carries the honesty instead. Recorded so a reviewer can overturn it deliberately.

### D-6 — One new check: a per-file, deny-by-default AST scan over the archetype tree

**Decided.** CH-01.2 adds **check 6** to `import_boundary_test.go` — never a new file, never a new guard. The file is not byte-frozen (`scope_fence_test.go:20-33`'s `del024ByteUnchangedFiles()` names only `event_descriptor.go`, `delegation_events.go`, `cost_events.go` — `:29-31`), and its own package comment records three prior in-place extensions (AG-03 → AG-22 → AG-23), so extending it is this repo's demonstrated convention and R-L3H-002's literal requirement.

**Mechanism.** For every `.go` file directly inside the archetype tree, parse with `go/parser` in `parser.ImportsOnly` and evaluate each import path in this order:

1. **Forbidden table first**, mirroring the ordering `import_boundary_test.go:170-176` pins as load-bearing: `…/src/ai/openaicompat`, `…/src/agenttest`, `…/src/apptest`, `…/src/layer3handoff` (D-4), plus `…/src/cmd` (`0005:241` — no `src/cmd/…` package from below the root). Forbidden-first is not cosmetic: the allowlist admits `…/src/ai` as a **prefix**, so an allowlist-first pass would silently admit `…/src/ai/openaicompat`.
2. **Deny-by-default allowlist**: the standard library (skipped by the toolchain's own classification, as `listNonStdlibDeps` does for checks 1–2), `…/src/agent`, `…/src/ai`, and the OTel **API** packages ADR 0005 § D3 authorises at Layer 3 — `otel/trace`, `otel/attribute`, `otel/codes`, `otel/semconv` — each carrying its § D3 citation in place, exactly as `allowedProductionPrefixes:219-244` requires of every entry.
3. Anything else **fails by default**.

**The message contract — how CH-01.2 scenario 1 is met.** The existing guard splits its message shapes: closure checks 1–3 name the denied **path** and use deny-by-default framing (`:296-303`) but cannot name a file, because `go list -deps` reports package-level closure membership, not per-file provenance; AST checks 4–5 name the **file** and the family (`:598-601`) but carry no deny-by-default framing and pass through no allowlist. Neither shape alone satisfies scenario 1, which demands **file AND path AND deny-by-default framing in one failure**.

Check 6 is the shape that carries all three, and it is a shape this file does not yet have: an AST scan (so it knows the file) filtered through a deny-by-default allowlist (so its rule text is deny-by-default rather than a named forbidden prefix). Its failure names `filepath.Base(path)`, the denied import path, and cites *"deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2)"* together with the clause the existing guard already uses at `:300-301`: adding a dependency needs its own recorded design decision, not an allowlist entry.

**One consequence, deliberate and load-bearing:** the archetype's forbidden table **must not name `database_administrator` or `workspace_syncer`**, even though Layer 2's own table does (`:201-202`). If it named them, scenario 1's planted import would fail citing *a named forbidden prefix* — precisely the "missing allowlist entry" framing the scenario forbids. Denying them by default is what makes the required message true. R-07 is still enforced; the rule text cites ADR 0005 § D1 row 3 as the deny-by-default authority.

**The deferred second mechanism, stated with its cost.** A transitive `go list -deps` closure check over `src/chat/...` would add non-zero-hop coverage that check 6 (zero-hop) does not have. It is **deferred to CH-02**, for a concrete mechanical reason, not for budget: at CH-01 the archetype is `doc.go`-only and imports nothing, so `listNonStdlibDeps` returns **zero** non-stdlib packages and the repo's own vacuity floor (`if len(deps) == 0 { t.Fatal }`, `:284-287`, pinned per-pattern by `S-AGP-070`) would **fail the guard**. Satisfying it would require inventing a different floor shape than the one the repo has pinned. Cost of deferring: between CH-01 and CH-02, an import reached only transitively through an admitted path is uncaught — bounded, because the only admitted first-party paths are `…/src/agent` and `…/src/ai`, whose own production closures check 1 already proves deny-by-default. Cost of including it now: ~35 lines plus a new vacuity-floor mechanism and its own bite. **CH-02, which gives the archetype its first real import, is where the closure check becomes both cheap and non-vacuous.**

### D-7 — The R-AGP-006 analogue: creating two packages disarms nothing

`agent-package-scaffold` R-AGP-006 (`spec.md:190-207`) requires that creating a package must not silently disarm an existing guard, and that the fix be **proven** not to have silenced it. The CH-01 analogue is discharged as follows, each claim verified by reading:

| Guard | Effect of `src/chat` + `src/cmd/chat` arriving | Evidence |
|---|---|---|
| Layer 1's forward guard (`src/ai/import_boundary_test.go`) | **None.** Its scanned set is an explicit, fully qualified pattern list — `src/ai/...`, `src/agenttest/...`, `src/handoff/...` (`:59-61`), narrowed from a module-wide `modulePath + "/..."` by its own AD-1. A new sibling tree cannot enter it | read this phase |
| Layer 2 checks 1, 3, 4 | **None.** Scoped to `layer2Pattern` = `…/src/agent/...` alone (`:150`) | read this phase |
| Layer 2 check 2 | **None.** `{layer2Pattern} ∪ layer2TestOnlyTreePatterns` (`:165-168`), one call per pattern | read this phase |
| Layer 2 check 5 | **None.** Scans `src/apptest` and `src/layer3handoff` by directory (`:627-630`) | read this phase |
| `forbiddenPrefixes` row `modulePath + "/src/cmd"` (`:204`) | **Armed, not disarmed.** The row previously named a tree that did not exist; `src/cmd/chat` makes it reachable for the first time | read this phase |

**Both new trees are siblings of `src/agent`, never nested beneath it.** The comment at `:159-164` records why this matters: `allowedProductionPrefixes` admits `modulePath+"/src/agent"` as a **prefix**, so nesting a tree under it silently admits it into Layer 2's production closure. `src/chat` and `src/cmd/chat` sit at `src/`, so no existing allowlist prefix reaches them.

**Vacuity floor for check 6.** The scan fatals when the archetype tree yields **zero** `.go` files, mirroring check 4's own floor (`:586-588`, *"the scan would pass vacuously"*). A `doc.go`-only package satisfies this floor with one file — which is exactly why the floor is file-count-based and not dependency-count-based (see D-6's deferral). If the scan walks subdirectories, the floor is **per directory**, following `S-AGP-070`'s per-pattern rule (`:313-321`): a mistyped path must fail loudly, never sweep nothing while reporting green.

## Bite proof — CH-01.2, transcribed verbatim from `0005:253-273`

```gherkin
Feature: the archetype's import boundary

Scenario: the guard bites on a forbidden module import
  Given a scratch file in the archetype package importing a Go package of another backend module
  When the import guard runs
  Then it fails naming the scratch file and the denied path
  And the failure cites the deny-by-default rule rather than a missing allowlist entry

Scenario: the guard bites on the observability SDK below the root
  Given a scratch file in the archetype package importing the observability SDK
  When the import guard runs
  Then it fails naming the file
  And the same import inside the composition root passes

Scenario: the guard is green on the merged tree
  Given the merged change with every scratch file removed
  When the import guard runs
  Then it passes
  And the merged diff contains no second, self-contained guard inside the archetype tree
```

**How each is proven, and what is recorded:**

| # | Scratch file planted | Expected RED | Recorded |
|---|---|---|---|
| 1 | `backend/agent/src/chat/scratch_forbidden_module.go` importing `github.com/cachicamas/backend/workspace_syncer/...` | Check 6 fails naming `scratch_forbidden_module.go`, the denied path, and the deny-by-default rule (never a named forbidden prefix — D-6) | full `go test` output pasted into `apply-progress.md`; file deleted immediately after |
| 2a | `backend/agent/src/chat/scratch_otel_sdk.go` importing `go.opentelemetry.io/otel/sdk/trace` | Check 6 fails naming `scratch_otel_sdk.go` and the denied path | same |
| 2b | `backend/agent/src/cmd/chat/scratch_otel_sdk.go`, **identical import** | Check 6 **passes** — the root is outside the scanned set (D-1) | the *passing* run is recorded too; a bite is only a bite if the negative control is shown |
| 3 | none | Whole suite green, uncached, wall-clock recorded | `git status --porcelain backend/agent/src/chat backend/agent/src/cmd` clean; `git diff --stat` shows the guard extension inside `src/agent/import_boundary_test.go` only, and **zero** `_test.go` files added under `src/chat` or `src/cmd/chat` |

**Every scratch file is removed before merge.** The merged diff is asserted to add no test file inside the archetype tree — that assertion is scenario 3's second clause and a `CPB` scenario in its own right.

## Capabilities

### New capabilities

- `chat-package-boundary` — prefix **`CPB`** (verified free). Covers: the two packages' paths, package identifiers and declares-nothing property; the archetype's forbidden closure as resolved by D-3 and D-4; the guard-extended-never-cloned property and the no-second-guard assertion; check 6's message contract (file **and** path **and** deny-by-default framing); its vacuity floor; and the composition root's exclusion from the scanned set.

### Modified capabilities

- `agent-package-scaffold` — **R-AGP-001** and **S-AGP-003** amended so the `backend/agent/src/coding/` and `backend/agent/src/cmd/` non-existence clause is scoped to AG-03's own merge state (D-2). No other requirement of that spec changes; `R-AGP-003`'s guard contract is **extended in practice** by check 6 but its normative text needs no edit — the requirement is already stated over "whatever the committed table holds" for its sibling R-AGP-002 and over a per-check pattern set for itself (`spec.md:29`).

## Approach

1. Write `src/chat/doc.go` (`package chat`, documentation only) and `src/cmd/chat/main.go` (`package main`, documented no-op `func main()`).
2. Prove CH-01.1's check evidence: `go build ./...` compiles both; the root's binary builds; nothing imports the root (structurally guaranteed — Go forbids importing a main package — and asserted in `CPB` rather than restated as a test that cannot fail).
3. Extend `import_boundary_test.go` **in place**: an amendment row in its package comment (following the AG-22 and AG-23 rows already there), the archetype tree resolver via `runtime.Caller(0)` as `layer2SiblingTreeDirs` does (`:624-631`), the forbidden table, the allowlist with per-entry § D3 citations, the file-count vacuity floor, and check 6.
4. Drive it TDD: check 6 written first against a planted violation (RED, recorded), then the tables that make it green.
5. Plant, record and remove each scratch file of the bite table above, including the root's negative control.
6. Write the `agent-package-scaffold` delta (D-2) and the `chat-package-boundary` spec.
7. Doc 0005 bookkeeping: tick `0005:981`, status line `0005:3` → **2 of 12**.
8. Final gate: cache cleared, `go test -race -count=1 ./...` green with wall-clock recorded, `make lint` clean, `gofmt -l` empty (**never** `make all` — its fmt step rewrites committed files).

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/chat/doc.go` | **New** | `package chat`, documentation only |
| `backend/agent/src/cmd/chat/main.go` | **New** | `package main`, documented no-op `func main()` |
| `backend/agent/src/agent/import_boundary_test.go` | **Modified** | Check 6 + tables + floor + package-comment amendment row. The only guard file touched |
| `openspec/specs/agent-package-scaffold/spec.md` | **Modified at archive** | R-AGP-001, S-AGP-003 scoped to AG-03's merge state |
| `openspec/specs/chat-package-boundary/spec.md` | **New at archive** | `CPB` capability |
| `openspec/changes/cachicamas-chat-package-scaffold/` | **New** | proposal, design, spec deltas, tasks, apply-progress, verify-report |
| `docs/architecture/milestones/0005-…md` | **Modified** | `:981` ticked, `:3` → 2 of 12 |
| `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | **Unchanged** | D-1. Asserted byte-identical against the merge base |
| `frontend/**`, any other `backend/*` module | **Unchanged** | — |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| A dependency creeps into `go.mod`, reddening nine unrelated Layer 2 assertions | Med | D-1's source-level scan needs none. Verify gate re-runs `git diff <merge-base> -- backend/agent/go.mod backend/agent/go.sum` and asserts empty |
| Check 6's failure message is written with named-forbidden-prefix framing, silently failing scenario 1's second `Then` | **High** | D-6 pins the other backend modules **out** of the forbidden table on purpose. The `CPB` spec asserts the message's three properties separately, and the recorded RED output is read for all three at verify |
| A second guard file is created inside `src/chat` "for locality" | Med | `0005:240`, R-L3H-002, `S-L3H-013`. Scenario 3's second clause is a spec-level assertion over the merged diff, checked with `git diff --stat` |
| Check 6 passes vacuously — wrong directory, mistyped path, or an empty tree | Med | File-count floor mirroring `:586-588`, per-directory if subdirectories are walked (`S-AGP-070` precedent). Proven by pointing the resolver at a nonexistent path once and recording the fatal |
| An existing guard is silently widened while adding check 6 | Med | D-7's table is re-verified at verify by diffing the **guard body** against the merge base, not merely the entry list — the failure mode recorded in `releasing-a-file-from-its-freeze-removes-its-reviewer` |
| The zero-hop scan is read as proving transitive purity | Med | D-6 states the gap, its bound, and its owner (CH-02) explicitly, in the proposal and in the guard's own source comment |
| `func main() {}` trips a linter | Low | `make lint` runs before merge; if it fires, the fix is a lint directive with its reason, never a behavioural body |
| The `agent-package-scaffold` delta over-corrects and weakens R-AGP-001's "declares nothing" clause for `src/agent` | Med | D-2 pins the delta's scope to the `src/coding`/`src/cmd` sentence and S-AGP-003. Verify diffs the promoted requirement's other clauses for byte-identity |
| Scenario 2's negative control is skipped because it is a *passing* case | Med | The bite table records the passing run as evidence in its own right; a bite without its negative control does not prove the root is excluded |

## Recorded, not repaired

| # | Finding | Evidence | Disposition |
|---|---|---|---|
| **F-1** | `backend/agent/Makefile:96-98` states *"this module has no `main` package yet. The composition root arrives with doc 0004 (src/cmd/cachicamas)."* Doc 0005's chat root arrives first, and this change gives the module its first `main` package | read this phase | Cosmetic. D-1 forbids the Makefile edit in this change; recorded so a reviewer does not read the stale comment as a planning contradiction |
| **F-2** | `backend/agent/go.mod` declares `go 1.26.6`; ADR 0005 § D2 states `go 1.26.3` | exploration | Pre-existing drift, unrelated to CH-01. Not repaired here |

## Rollback plan

**Before merge:** discard the branch. The worktree touches no shared state.

**After merge, if the guard is found wrong:** revert the single PR. The change adds two Go files, extends one test file, adds one `openspec/` directory, amends one promoted spec and edits two lines of doc 0005. It alters **no** `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, no schema, no wire, no configuration, and no runtime behaviour — nothing is deployed, nothing is migrated, and no consumer exists yet (CH-02 and CH-03 are blocked on this milestone). The revert is complete by construction.

**Partial rollback is available and preferable if only the guard is wrong:** reverting check 6 and its tables alone leaves the two packages in place; CH-01.1 and CH-01.2 are separate nodes with separate closing conditions. Reverting the packages alone is **not** available — check 6's vacuity floor requires the archetype tree to exist.

**After CH-02 merges, revert is unavailable.** The correction path is then an amending SDD change against `chat-package-boundary`, recording what changed and why, exactly as this repo requires a breaking change to be an amendment rather than a silent drift.

## Dependencies

- **CH-00 merged** — PR #190, `419a4291`. `chat-archetype-contract` is promoted and is this change's citation target for the archetype's identity (`spec.md:145-149`) and the substitution rule (`spec.md:196-200`).
- **AG-23 merged**, Layer 2 complete at 24 of 24; the guard this change extends is its shipped artifact.
- Read-only: ADR 0005, ADR 0009, `agent-package-scaffold`, `agent-layer3-handoff`.
- **There is no `openspec` CLI, no `.github/`, and no CI in this repo.** Every gate is human-run.

## Success criteria

| # | Criterion | Evidence at verify |
|---|---|---|
| 1 | `backend/agent/src/chat/` exists, is `package chat`, and declares nothing beyond its package clause and doc comment | `go doc` output; `CPB` scenario |
| 2 | `backend/agent/src/cmd/chat/` exists, is `package main`, and its binary builds | `go build ./...`; `go build ./src/cmd/chat` |
| 3 | Nothing imports the composition root | Structural (Go forbids importing `package main`); asserted in `CPB` |
| 4 | Scenario 1 bites: one failure naming the scratch **file**, the denied **path**, and the **deny-by-default** rule | recorded RED output in `apply-progress.md`, read for all three properties |
| 5 | Scenario 2 bites in `src/chat` and **passes** in `src/cmd/chat` for the identical import | both runs recorded — the RED and the negative control |
| 6 | Scenario 3: merged tree green, every scratch file removed, no second guard inside the archetype tree | `git status --porcelain`; `git diff --stat`; zero `_test.go` under `src/chat`/`src/cmd/chat` |
| 7 | No existing guard was disarmed | D-7's five-row table re-verified; guard **bodies** diffed against the merge base |
| 8 | `go.mod`, `go.sum`, `Makefile`, `.golangci.yml` byte-unchanged | `git diff <merge-base with origin/main> -- …` empty for each |
| 9 | Suite green, **uncached** | cache cleared, `go test -race -count=1 ./...`, wall-clock recorded. A sub-second result is a cache hit and is not evidence |
| 10 | `agent-package-scaffold` no longer carries a false clause | R-AGP-001 / S-AGP-003 delta promoted; the rest of the requirement byte-identical |
| 11 | Bookkeeping | `0005:981` ticked; `0005:3` reads **2 of 12** |

## Review Workload Forecast

| Component | Est. counted lines |
|---|---|
| `src/chat/doc.go` | 25 |
| `src/cmd/chat/main.go` | 20 |
| `import_boundary_test.go` extension (check 6, tables, floor, comment row) | 135 |
| `agent-package-scaffold` delta | 45 |
| `chat-package-boundary` spec | 185 |
| SDD artifacts (design, tasks, apply-progress, verify-report) | 550 |
| doc 0005 bookkeeping | 4 |
| **Total** | **~965** |

Authored Go under `backend/` is ~180 lines — comfortably inside the milestone's own sizing rule (`0005:130`: prefer under 250 changed lines, reassess before 400). The overage is SDD markdown, which `sdd-attempt-budget-counts-docs` records is counted against the budget.

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High
```

The 400-line default is exceeded and knowingly so: the session's cached delivery strategy is `single-pr` with a **1000-line budget pre-approved by the user**, and `size:exception` is pre-authorised. Splitting is possible in principle (CH-01.1 then CH-01.2) but rejected: check 6's vacuity floor requires the archetype tree to exist, so a CH-01.1-only PR would ship two packages with **no** guard over them — precisely the unguarded window this milestone exists to close. The two nodes are one deliverable.

## Proposal question round (auto mode — recorded, not asked)

Execution mode is `auto` and the four blocking product questions were already answered by the user (Engram #3725). These three remaining forks were decided from the charter rather than asked, and are stated so a reviewer can overturn each deliberately:

1. **Does the composition root's stub say anything when run?** Decided **no** (D-5) — an empty `func main()` with a doc comment. `0005:240` puts behaviour out of scope. If overturned, the stub gains a stderr line and a non-zero exit, and the change gains its first behaviour.
2. **Does CH-01 also ship a transitive closure check?** Decided **no** (D-6) — the repo's pinned vacuity floor cannot be satisfied by a tree with zero non-stdlib dependencies, and CH-02 is where the check becomes non-vacuous. If overturned, CH-01 must also invent a new floor shape, and the exposure window between CH-01 and CH-02 closes.
3. **Is the `agent-package-scaffold` repair a delta or a follow-up?** Decided **delta, in this PR** (D-2, user decision #2). If overturned, a promoted spec stays false on `main` from the day this merges.
