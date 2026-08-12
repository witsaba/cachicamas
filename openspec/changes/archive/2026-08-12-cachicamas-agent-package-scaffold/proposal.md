# Proposal — Create `backend/agent/src/agent/` with both boundary guards

> **Change**: `cachicamas-agent-package-scaffold`
> **Milestone**: AG-03 — Package scaffold and boundary guards (Layer 2, Wave 1) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-03--package-scaffold-and-boundary-guards)
> **Nodes**: AG-03.1 scaffold `[mechanical]` · AG-03.2 forward import guard `[guard]` · AG-03.3 no-ambient-authority guard `[guard]`
> **Status**: proposed · **Date**: 2026-08-12 · **Driver**: braejan · **Project**: cachicamas (witsaba)
> **Branch**: `feat/agent-layer2-wave1-ag03` · **Predecessor artifact**: `explore.md` (this change)
> **Closes**: doc 0003 R-01, R-02, R-03 mechanically · **Blocks**: AG-04, AG-12 and everything code-bearing
> **Scope**: one new Go package under `backend/agent/src/agent/` + one targeted edit to Layer 1's shipped forward guard. **Zero new dependencies. Zero `go.mod` change. Zero behavior.**
> **Delivery**: single PR, 1000-line budget, `size:exception` pre-authorized by the user

---

## Intent

Layer 2 has no code and no directory. AG-03 is the AI-00 of this layer: it creates `backend/agent/src/agent/` and, in the same PR, makes its import and I/O boundaries mechanically enforced, so "Layer 2 performs no I/O of its own" ([ADR 0005 § D1](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2)) is a failing test rather than a reviewer's memory.

**Why the guards land with the package, not after it.** Go's module system enforces nothing here: `src/agent` importing `src/coding`, `net/http`, or the vendor adapter subtree all compile. And `os.Getenv` inside Layer 2 compiles *and* passes an import-closure guard, because `os` is stdlib. A guard written after the first violation is a cleanup, not a guard.

**Why this change is not purely additive.** Creating the directory falsifies three properties that Layer 1's shipped spec and shipped guard currently assert. That was known and recorded in advance — `backend/agent/src/ai/import_boundary_test.go:82-92` states the fix belongs *"in the change that creates the directory, never by deleting the prefix row"* — and this is that change.

---

## Settled decisions (inputs, not open questions)

These are resolved upstream by the user. Later phases implement and document them; they do not re-litigate them.

### 1. The "doc-guard byte-suffix convention" is substituted, not invented

AG-03.1 item 2 cites a *"doc-guard byte-suffix convention doc 0002's guards establish."* No such worked mechanism exists anywhere in doc 0002 or doc 0003 — the only repo-wide hit is one parenthetical example in a node-grammar table cell (`0002:155`), and the same wording already sat in doc 0003 v1, so it predates the v2 restructure.

**Substituted mechanism — the repo's real working precedent:** `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go` (AI-40.2, R-L2H-004). It resolves a doc file via `runtime.Caller(0)`, reads its **raw bytes** with `os.ReadFile`, matches a pinned tab-indented doc-comment row grammar with a `regexp`, and diffs the parsed rows entry-by-entry and in order against a **committed in-test expectation table**. `backend/agent/src/agent/doc.go` adopts that shape: the layer-contract paragraphs (imports row, no-I/O rule, event stream as the only upward contract) are written as machine-parseable rows with a pinned prefix, and a test in the same package diffs them against a committed table. Later milestones append guarded paragraphs by appending rows and table entries in the same PR.

This is recorded as a **substitution**, not a match: doc 0003:355's wording is flagged for a separate amendment (see Out of scope). The proposal does not pretend the cited convention existed.

### 2. The Layer 1 guard self-reference fix is in scope for AG-03

`layer1Pattern = modulePath + "/..."` scans the whole module, and `go list -deps -test` emits the pattern's own members — so the mere existence of `src/agent` makes it appear in Layer 1's scanned set and match the `{modulePath}/src/agent` forbidden-prefix row, failing `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` with **zero real import violations**. The fix is either narrowing the scanned roots to Layer 1's own packages or exempting the pattern's own members from the prefix match — design decides which. The forbidden-prefix row is never deleted.

**Bounded blast radius, measured:** `TestLayer1_DependencySet_ExactRequiresAndClosure` (R-AGM-008) filters `modulePath`-prefixed entries out of its closure before comparing, so it is immune either way as long as `src/agent` adds no external dependency — which it does not.

### 3. The 2026-08-11 network-access measurement is re-run, not trusted

AG-03.2's production allowlist admits `src/ai`'s transitive closure on the strength of *"measured free of direct network access, 2026-08-11."* That claim traces to doc 0003's own Phase-1 research digest (`0003:81`) — a point-in-time manual measurement in prose, guarded by nothing, and present in neither AG-00's nor AG-01's `decision.md`. Design and apply MUST re-run `go list -deps` and `go list -deps -test` over `backend/agent/src/ai/...` **fresh** and record the output as evidence. If the closure has changed, the allowlist is written against the measured reality and the divergence is reported, not smoothed over.

---

## Scope

### In scope

| # | Deliverable | Node |
| --- | --- | --- |
| 1 | `backend/agent/src/agent/doc.go` — `package agent`, doc comment only, no declarations. Carries the layer contract in the AI-40.2 row grammar (settled decision 1) | AG-03.1 |
| 2 | The doc-row scan: committed expectation table + byte-level parse of `doc.go`, mirroring `doc_matrix_guard_test.go` | AG-03.1 |
| 3 | Forward import guard in `backend/agent/src/agent/` — `go list -deps -test`, deny-by-default allowlist, forbidden prefixes matched **before** the allowlist, production and test closures asserted separately. Four recorded bites (doc 0003's four Gherkin scenarios) | AG-03.2 |
| 4 | No-ambient-authority guard in `backend/agent/src/agent/` — `go/parser` + `ast.Inspect` call-site scan over non-`_test.go` files, forbidding `os`, `os/exec`, `syscall`, `io/ioutil` call sites, alias- and dot-import-aware. One recorded bite via a planted `t.TempDir()` scratch file | AG-03.3 |
| 5 | Targeted fix to `backend/agent/src/ai/import_boundary_test.go` for the self-reference hazard (settled decision 2) | AG-03.2 |
| 6 | Delta spec amending `agent-module-scaffold` — `S-AGM-035` (asserts `src/agent` does not exist) and, if the pattern narrows, `S-AGM-041` (asserts the guard's pattern covers every package of the module) | AG-03.2 |
| 7 | Recorded evidence: pre-change and post-change `make test` / `make lint` in `backend/agent/`; the fresh `go list` closure measurement; five guard-bite reds | all |

### Out of scope

**Deferred but related — each has a named owner, so none is a gap:**

- **Any event, loop, or harness behavior** — the milestone's own charter clause. No type, no constant, no function. AG-04 onward.
- **Any dependency, including the OTel API modules.** No `go.mod` or `go.sum` edit. Admitting an OTel *path* to the allowlist is not the same as adding a require, and this change adds no require either way.
- **The reverse direction** (nothing reaches into Layer 2) — owned by Layer 1's forward guard and the hexagon's guards, per AG-03.2's own out-of-scope field.
- **`src/coding/`, `src/cmd/`** — doc 0004. Not created, not stubbed, not empty directories. Their absence is what keeps their forbidden-prefix rows testable.
- **Amending doc 0003:355's "byte-suffix convention" wording** — a separate documentation change under doc 0003's living-graph clause. Silently editing a merged plan from inside an implementation PR is the wrong shape.
- **Extending the ambient-authority scan with type information** (`go/types`, `x/tools/go/analysis`) to remove the known local-identifier false-positive — no milestone authorizes that dependency; the limitation is recorded in the guard's source, exactly as AI-25.2 does.
- **`src/handoff`** — nothing imports it; it is Layer 1's own consumer proof and stays that way.
- **CI.** Still absent. Every gate here runs when a human runs `make test` in `backend/agent/`.

---

## Capabilities

### New capabilities

- `agent-package-scaffold`: the Layer 2 package's existence, its documented layer contract and doc-row convention, its forward import guard over both closures, and its no-ambient-authority guard — with bite proof as the closing condition for both guards. Requirement ids `R-AGP-0NN`, scenario ids `S-AGP-0NN` (verified unused repo-wide).

### Modified capabilities

- `agent-module-scaffold`: **`R-AGM-004`** — `S-AGM-035` asserts unconditionally that `backend/agent/src/agent` does not exist. The requirement prose already authorizes this change (*"MUST NOT contain `src/agent/` … until docs 0003 and 0004 create them"*), so the scenario is amended to state the post-AG-03 invariant while `src/coding/` and `src/cmd/` remain forbidden. **`R-AGM-005`** — `S-AGM-041` asserts the guard's pattern *"covers every package of the module"*; if design narrows `layer1Pattern`, that scenario is amended to state the Layer-1-scoped roots and the reason. Both amendments follow this spec's promotion discipline: `MODIFIED` blocks restating the full requirement including unchanged scenarios.

---

## Approach

```
backend/agent/
├── src/
│   ├── ai/
│   │   └── import_boundary_test.go   # MODIFIED — self-reference fix only (decision 2)
│   ├── agenttest/                    # UNCHANGED — direct sibling of ai/ (ADR 0005 § D2, Guard C)
│   └── agent/                        # NEW — Layer 2
│       ├── doc.go                    #   layer contract in the AI-40.2 row grammar
│       ├── doc_contract_guard_test.go#   AG-03.1 — byte-level row scan vs committed table
│       ├── import_boundary_test.go   #   AG-03.2 — go list -deps -test, deny-by-default
│       └── ambient_authority_test.go #   AG-03.3 — go/ast call-site scan
```

1. **AG-03.1** creates the package and its doc contract; `make test` and `make lint` stay green. Mechanical leaf: exempt from red-green, never from its recorded Check list.
2. **AG-03.2** re-measures `src/ai`'s two closures fresh (decision 3), writes the allowlist against the measured reality, fixes the L1 self-reference (decision 2), and closes on four recorded reds.
3. **AG-03.3** retargets AI-25.2's call-site AST scan — *not* AI-00's import-closure scan, which cannot express ambient authority — and closes on one recorded red.

Every guard mechanism, allowlist table, and forbidden-prefix row is specified in `design.md`; this proposal names the mechanisms and their sources, not their code.

---

## Affected areas

| Area | Impact | Path | Change |
| --- | --- | --- | --- |
| Layer 2 | New | `backend/agent/src/agent/doc.go` | Package doc only; layer contract as parseable rows |
| Guard | New | `backend/agent/src/agent/doc_contract_guard_test.go` | AG-03.1 row scan vs committed table |
| Guard | New | `backend/agent/src/agent/import_boundary_test.go` | AG-03.2 forward guard, both closures |
| Guard | New | `backend/agent/src/agent/ambient_authority_test.go` | AG-03.3 call-site AST scan |
| Layer 1 | **Modified** | `backend/agent/src/ai/import_boundary_test.go` | Self-reference fix; forbidden-prefix rows preserved |
| Specs | Modified | `openspec/specs/agent-module-scaffold/spec.md` | Delta at archive: `R-AGM-004`, `R-AGM-005` |
| — | **Unchanged** | `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | No dependency, no tooling change |
| — | **Unchanged** | `backend/agent/src/agenttest/**`, `src/handoff/**` | Named by the guards; not edited |
| — | **Unchanged** | `backend/database_administrator/**`, `backend/workspace_syncer/**`, `docs/**` | Cited, never edited |

---

## Risks

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| 1 | The forward guard passes vacuously — a doc-only package has almost no dependency graph | High | The leaf does not close on green. It closes on four recorded reds, then green. The guard also fatals when `go list` returns zero packages, as Layer 1's does |
| 2 | The L1 self-reference fix over-narrows and silently stops covering a Layer 1 package | Medium | Design records the chosen mechanism and its reason in the guard's source; the `S-AGM-041` delta states the new scanned roots explicitly, so a later narrowing is a spec violation rather than an edit |
| 3 | The 2026-08-11 closure measurement is stale and the allowlist is written against fiction | Medium | Decision 3: re-measured fresh at design and apply, output recorded as evidence |
| 4 | A scratch bite violation is committed by accident | Low | Five bites, all scratch-file based, all removed. The PR diff is the check: no scratch file and no `require` line may appear in it |
| 5 | The AST scan false-positives on a local identifier named `os` | Low | Recorded limitation, inherited verbatim from AI-25.2 with its reasoning; removing it costs an unauthorized dependency |
| 6 | The substituted doc-guard convention diverges from whatever doc 0003 later means by "byte-suffix" | Medium | The substitution is recorded here, in `design.md`, and in the PR description; the doc 0003 amendment is filed as its own change |
| 7 | Review budget — three guards plus a cross-cutting L1 edit | Medium | `size:exception` pre-authorized. Four commits, one per node plus the L1 fix, each independently reviewable; the L1 fix touches one file and deletes no rule |

---

## Rollback plan

Required by `openspec/config.yaml`. Three levels, all cheap, because nothing depends on Layer 2 yet.

1. **Revert the L1 guard fix alone.** `backend/agent/src/ai/import_boundary_test.go` returns to its previous content. This is the only file the change modifies, and it is a test file with no production callers — but note it only makes sense together with level 3, because the fix exists *because* `src/agent` exists.
2. **Delete `backend/agent/src/agent/` entirely.** Nothing imports it: no Go file anywhere names it, and Layer 1's own forbidden-prefix row proves Layer 1 never will. `make test` in all three modules returns to the recorded pre-change baseline.
3. **Revert the merge commit.** Levels 1 and 2 together. No migration, no persisted state, no running process, no container, no published artifact, no dependency to unwind.

**Forward-fix preference.** If a guard turns out over-strict after merge — AG-04's event envelope needing an allowlist entry, say — the correct move is to amend the allowlist in *that* milestone's change with its own justification, in the same PR, never to revert the guard. Reverting a guard to unblock a dependency is the exact failure ADR 0005 § Guard A was written against.

---

## Dependencies

- **AI-40** — merged `7326a813`, 2026-08-10. Layer 1 closed 42 of 42; its contract surface is frozen.
- **AG-00** — `agent-contract-vocabulary`, merged as PR #159. The layer contract in `doc.go` uses its terms.
- **ADR 0005 § D1 / § D2 / § D3** — the normative import table, the location mapping, and the observability boundary.
- **Toolchain**: Go 1.26.3, `golangci-lint v2.9.0`. **No new dependency in any module**, so `openspec/AGENTS.md` rule 5 is not triggered.

---

## Success criteria

- [ ] `cd backend/agent && make test` green and `make lint` clean, recorded in the PR.
- [ ] `backend/agent/go.mod` and `go.sum` are byte-unchanged (recorded diff).
- [ ] `backend/agent/src/agent/` exists, exports nothing, declares nothing beyond the package clause and its doc comment.
- [ ] `backend/agent/src/coding/` and `backend/agent/src/cmd/` still do not exist.
- [ ] The doc-row scan parses `doc.go`'s raw bytes and diffs every contract row against a committed table, in order.
- [ ] Five recorded guard-bite reds in the PR: four for the forward guard (application-layer import, `net/http`, vendor adapter subtree denied by name, test-substrate widening the test closure only), one for the ambient-authority guard (`os.Getenv`).
- [ ] The forward guard's forbidden-prefix table is matched **before** its allowlist, with the reason stated in source.
- [ ] The fresh `go list -deps` / `go list -deps -test` measurement over `backend/agent/src/ai/...` is recorded, dated, and matches the allowlist actually written.
- [ ] `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` passes with the `{modulePath}/src/agent` forbidden-prefix row still present and still able to bite.
- [ ] `TestLayer1_DependencySet_ExactRequiresAndClosure` passes unchanged.
- [ ] The `agent-module-scaffold` delta amends `S-AGM-035` (and `S-AGM-041` if the pattern narrowed), restating each full requirement per that spec's promotion discipline.
- [ ] No scratch violation file appears in the merged diff; `git status` is clean.

---

## Open questions carried to design

| # | Question | Status |
| --- | --- | --- |
| 1 | **Which OTel API paths does AG-03.2's allowlist admit?** Three candidates: ADR 0005 § D3's full permitted set (`otel`, `/trace`, `/attribute`, `/codes`, `/metric`); Layer 1's narrower actually-shipped subset (`/trace`, `/attribute`, `/codes` + forced transitives, deliberately excluding root `otel`); or **zero** OTel entries, since AG-03's charter excludes all event, loop, and harness behavior and the package therefore has no OTel usage to admit. Each has a real tradeoff between deny-by-default purity and churn at the first observability milestone | **Open — design decides.** Not pre-decided here |
| 2 | Which mechanism fixes the L1 self-reference — narrowing `layer1Pattern` to Layer 1's roots, or exempting the pattern's own members from the prefix match | Open — design decides; decision 2 fixes only that it *is* fixed here and the row is never deleted |
| 3 | Exact row grammar and prefix for the doc-contract convention (AI-40.2 uses `^//\tCAP-[RO]-\d\d\b`) | Open — design decides the Layer 2 spelling |

---

## Notes for the following phases

- **`sdd-spec`**: one new spec `specs/agent-package-scaffold/spec.md` (`R-AGP-0NN` / `S-AGP-0NN`) plus a delta for `agent-module-scaffold`. Guard requirements close on recorded bites, never on green. Doc 0003's four AG-03.2 Gherkin scenarios and one AG-03.3 scenario are the normative starting set — restate them as verifiable scenarios, do not reduce them.
- **`sdd-design`**: must carry the three open questions' resolutions with rationale; the exact `go list` invocation with `-test` and its synthesized-package normalization; the allowlist groups and forbidden-prefix table against the **fresh** closure measurement; the AST scan's import-resolution rules with its recorded type-information limitation; the L1 self-reference fix with a before/after diff rationale; and the doc-row grammar.
- **`sdd-tasks`**: four phases — AG-03.1, AG-03.2, AG-03.3, and the L1 fix (sequenced with AG-03.2 since it fires the moment the directory exists). AG-03.2 and AG-03.3 are parallel after AG-03.1. Evidence gate is `make test` in `backend/agent/` for all four. Single PR; `size:exception` pre-authorized against the 1000-line budget; the tasks file must restate why.
