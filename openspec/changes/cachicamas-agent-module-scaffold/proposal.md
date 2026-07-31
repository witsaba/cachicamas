# Proposal — Create the `backend/agent` module and both boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 (Wave 0 — Found) of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Status**: proposed
> **Created**: 2026-07-31
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Scope**: new `backend/agent` Go module + repo-root `go.work` + one modified test file in `backend/database_administrator`
> **Predecessor artifact**: `openspec/changes/cachicamas-agent-module-scaffold/explore.md`
> **Decisions implemented**: [ADR 0005 § D2](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2), [§ D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), [§ Enforcement](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement)
> **Stack**: Go 1.26.3. **Zero third-party dependencies** — that is the point of the change, not an omission.

---

## Intent

Doc 0002 plans Layer 1 of the cachicamas agent stack as forty-one milestones built from nothing. It opens with a blunt statement of the situation: neither the `backend/agent` module nor a single line of Layer 1 exists on disk. AI-00 is the first milestone because there is nowhere to put a test until it merges.

This change creates that module — `backend/agent/`, module path `github.com/cachicamas/backend/agent`, `go 1.26.3`, zero requires — together with its build tooling, its one Layer 1 package, the external test package that ADR 0005 pins as structurally load-bearing, a repo-root `go.work` listing all three backend modules, and both halves of the module boundary as executable guards.

**Why the guards land with the module rather than after it.** ADR 0005 promotes the agent stack out of `database_administrator` specifically so that the compiler enforces half of the dependency rule for free: Go modules cannot form an import cycle. The other half is not free. `backend/agent` importing a sibling backend module is a build error, but `backend/agent` importing an arbitrary vendor SDK, or the OpenTelemetry SDK, or a transitive cachicamas package pulled in through a vendor dependency, is not. Neither is `database_administrator` reaching *forward* into the agent module from a layer that ADR 0005 § D1 forbids. Both directions need a test, and a test written after the first violation is a cleanup, not a guard.

**Why the module is empty.** The retired plan built seventeen Layer 1 milestones inside `database_administrator/src/tools/agent/` and then had to unbuild them; four of doc 0002's disappearing milestones existed only because a contract had been built the wrong way and needed correcting. Starting from a correct, empty, guarded module is the cheapest moment in the whole programme to get the shape right. Every deliverable here is either a boundary or the tooling that proves a boundary holds.

---

## Locked decisions

These are settled by ADR 0005, doc 0002's AI-00 charter, and the user's delivery decision. The later phases document and implement them; they do not re-litigate them.

1. **Module identity.** `backend/agent/` → `module github.com/cachicamas/backend/agent`, `go 1.26.3`, **zero `require` directives**. No `go.sum`. No dependency arrives in this change — not OpenTelemetry, not a transport, not a test helper library. The OTel API arrives at AI-37 (pre-authorized by ADR 0005 § D3 and by nothing else); the transport arrives at AI-24 behind its own ADR gate.
2. **Layout.** `src/ai/` is the Layer 1 package. `src/agenttest/` is its **direct sibling** — non-negotiable, because AI-20.4's signature guard will resolve `../ai/provider.go` from `runtime.Caller(0)` (ADR 0005 § D2, Guard C).
3. **No placeholder siblings.** `src/agent/`, `src/coding/` and `src/cmd/` are **not** created, not even as empty directories. They belong to docs 0003 and 0004, and their absence is what makes the forward guard's forbidden-prefix list testable.
4. **No `replace` directive** in `backend/database_administrator/go.mod`. ADR 0005 § Migration: a `replace` with no requirer is dead weight that also disguises the D1 row-5 build cost.
5. **`backend/database_administrator/src/tools/tools.go` stays byte-identical.** S3 is resolved by vacating the name `tools`, not by relocating a standard-idiom file. The change records a diff proving it.
6. **Copied build config, not shared build config.** `Makefile` and `.golangci.yml` are copies of `database_administrator`'s. A shared config would be a fourth thing to own and would couple two modules that must not be coupled. The drift risk is recorded by ADR 0005 § D2 and accepted; the mitigation is that the load-bearing targets are named in the spec — `test` is `go test -race -v ./...` and `lint` runs `golangci-lint` at the pinned `v2.9.0` — so a later divergence is a spec violation rather than a silent edit.
7. **Forward guard mechanism.** `go list -deps -test` with a **deny-by-default allowlist** (stdlib as the exact set from `go list std`, plus this module's own packages, plus an explicitly empty vendor group). Not `go list -f '{{range .Imports}}'`, which sees neither test imports nor transitive dependencies (ADR 0005 § Guard A, finding S6). The `-test` flag is required: `-deps` alone does not cover test imports, verified on this worktree.
8. **Reverse guard mechanism.** Direct imports only for the module-scope half — ADR 0005 § D1 row 5 permits `src/application` and `src/cmd/server` to import the agent module, so a transitive scan would indict every consumer of `application` the moment row 5 is exercised. The `src/domain` half stays transitive (`-deps`), because domain must not reach the agent module by any path.
9. **Bite proof is the closing condition for both guards.** Three scratch violations for the forward guard (an import of `database_administrator`, an import of the OTel SDK, an import of an arbitrary third-party module), one for the reverse guard (an import of `backend/agent` from `src/domain`). Each must make its guard FAIL, the red output is recorded in the PR, then the violation is dropped.
10. **One PR for the whole of Wave 0's first milestone**, by explicit user decision, exceeding doc 0002's review budget. See [Review budget exception](#review-budget-exception).

---

## Scope

### In scope

- **New Go module `backend/agent/`**:
  - `go.mod` — `module github.com/cachicamas/backend/agent`, `go 1.26.3`, zero requires.
  - `Makefile` — copied from `database_administrator`, reduced: `help`, `tools`, `tidy`, `build`, `test`, `test/cover`, `fmt`, `vet`, `lint`, `clean`, `all`. Same `LOCALBIN := bin`, same `GOLANGCI_LINT_VERSION := v2.9.0`.
  - `.golangci.yml` — verbatim copy of the 30-line v2 config (`govet`, `errcheck`, `staticcheck`, `unused`, `revive`; `run.tests: true`).
  - `.gitignore` — verbatim copy (`bin/`, coverage artifacts, editor noise).
  - `README.md` — the module's one-paragraph purpose, its three-layer contents, the dependency rule, and the fact that `make test` must be run here separately because no CI exists.
  - `src/ai/doc.go` — package documentation only: the layer boundary and the import rule, nothing else.
  - `src/ai/import_boundary_test.go` — the forward guard.
  - `src/agenttest/` — external test package proving `src/ai` is importable from outside the package.
- **Repo-root `go.work`** listing `./backend/agent`, `./backend/database_administrator`, `./backend/workspace_syncer`, plus `go.work.sum` if the toolchain generates one.
- **Reverse guard** in `backend/database_administrator/src/domain/imports_test.go`: the existing forbidden-prefix test extended to name the agent module, plus a new module-scope test asserting that no package outside `src/application/` and `src/cmd/server` names it.
- **Recorded evidence**: pre-change `make test` baselines for both existing modules; post-change runs for all three; the four recorded guard-bite red outputs; the `git diff` proving `src/tools/tools.go` is byte-identical.

### Out of scope

**Deferred but related** — each has a named owner, so none of these is a gap:

- **Any dependency whatsoever.** The transport / vendor SDK is AI-24 (its own ADR gate). The OpenTelemetry API is AI-37 (pre-authorized by ADR 0005 § D3). Neither may be added here to "save a step".
- **Any Layer 1 contract content.** The vocabulary is AI-01, the stream lifecycle and carrier decision is AI-02, the capability scope is AI-03, and the first type is AI-04. `src/ai/doc.go` states the boundary and stops.
- **`src/agent/` (Layer 2), `src/coding/` (Layer 3), `src/cmd/cachicamas/`** — docs 0003 and 0004. Not created, not stubbed, not an empty directory (locked decision 3).
- **AI-20.4's signature guard.** This change only guarantees the *layout* that guard depends on. The guard itself belongs to AI-20.
- **AI-25.2's ambient-authority scan** over the adapter package. Different guard, different milestone.
- **Exercising ADR 0005 § D1 row 5** (`database_administrator` actually importing the agent module). A non-goal for all of v1. It breaks `docker compose build` — the Dockerfile copies only `./backend/database_administrator` — and costs a `replace`, a compose build-context move, and a rewrite of every `COPY` path. ADR 0005 prices it as its own change with its own review.
- **A `replace` directive** anywhere (locked decision 4).
- **Moving or renaming `database_administrator/src/tools/tools.go`** (locked decision 5).
- **CI.** `.github/workflows/` is absent and this change does not add it. Three modules now means three places a human runs `make test`. ADR 0005 records this under Consequences rather than solving it, and doc 0002 repeats the caveat once, globally.
- **A shared build configuration** for the three modules. Rejected in locked decision 6, with the drift risk accepted and mitigated by naming the load-bearing targets.
- **Docker packaging of the new module.** No `Dockerfile`, no `.dockerignore`, no compose service. Nothing runs yet; there is no binary until doc 0004.

---

## Affected areas

| Area | Kind | Path | Change |
| --- | --- | --- | --- |
| Module | new | `backend/agent/go.mod` | `module github.com/cachicamas/backend/agent`; `go 1.26.3`; zero requires |
| Build | new | `backend/agent/Makefile` | Copy of `database_administrator/Makefile`, minus `run`, `install`, and `test/integration` |
| Lint | new | `backend/agent/.golangci.yml` | Verbatim copy of the v2.9.0-era config |
| VCS | new | `backend/agent/.gitignore` | Verbatim copy |
| Docs | new | `backend/agent/README.md` | Purpose, three-layer contents, dependency rule, no-CI note |
| Layer 1 | new | `backend/agent/src/ai/doc.go` | `package ai` doc comment: boundary + import rule. No declarations |
| Guard A | new | `backend/agent/src/ai/import_boundary_test.go` | Forward guard: `go list -deps -test`, deny-by-default allowlist, named forbidden prefixes |
| Consumer proof | new | `backend/agent/src/agenttest/doc.go` | `package agenttest` doc comment stating why this directory must stay a sibling of `src/ai` |
| Consumer proof | new | `backend/agent/src/agenttest/import_compile_test.go` | `package agenttest_test`, imports `…/src/ai`, compiles |
| Workspace | new | `go.work` | Lists all three backend modules |
| Workspace | new (conditional) | `go.work.sum` | Committed only if the toolchain generates one |
| Guard B | modified | `backend/database_administrator/src/domain/imports_test.go` | Adds the agent-module forbidden prefix to the domain test; adds the module-scope direct-import test |
| — | **unchanged** | `backend/database_administrator/go.mod` | No `replace`. Asserted, not assumed |
| — | **unchanged** | `backend/database_administrator/src/tools/tools.go` | Byte-identical. Diff recorded |
| — | **unchanged** | `backend/workspace_syncer/**` | Listed in `go.work`; no file edited |

---

## Approach

```
cachicamas/
├── go.work                                 # NEW — lists the three modules
├── backend/
│   ├── agent/                              # NEW MODULE
│   │   ├── go.mod                          #   module …/backend/agent, go 1.26.3, zero requires
│   │   ├── Makefile                        #   test = go test -race -v ./...  ·  lint = golangci-lint v2.9.0
│   │   ├── .golangci.yml
│   │   ├── .gitignore
│   │   ├── README.md
│   │   └── src/
│   │       ├── ai/                         #   Layer 1
│   │       │   ├── doc.go                  #     package doc only
│   │       │   └── import_boundary_test.go #     Guard A (forward)
│   │       └── agenttest/                  #   DIRECT SIBLING of ai/ — ADR 0005 § D2, Guard C
│   │           ├── doc.go
│   │           └── import_compile_test.go  #     external-package import of …/src/ai
│   ├── database_administrator/
│   │   ├── go.mod                          #   UNCHANGED — no replace
│   │   └── src/
│   │       ├── domain/imports_test.go      #   MODIFIED — Guard B, both halves
│   │       └── tools/tools.go              #   UNCHANGED — byte-identical
│   └── workspace_syncer/                   #   UNCHANGED
```

`src/agent/`, `src/coding/` and `src/cmd/` are absent from the tree above **on purpose**. Their absence is a chartered property, not an oversight.

The technical mechanics of both guards — the exact `go list` invocations, the normalization of synthesized test packages, the three allowlist groups, and the forbidden-prefix table — are in `design.md`.

---

## Review budget exception

Doc 0002's milestone rules say *prefer under 250 changed lines; stop and reassess before 400*. Wave 0's first milestone ships as **one PR** by explicit user decision, and the forecast exceeds that budget: roughly 520 changed lines, of which the two guards and their tables account for about 250 and the copied `Makefile` for about 120.

Doc 0001 § 9 (Process) requires the PR description to say *why* a change does not fit the budget. The reason is stated here so the tasks phase and the PR description carry the same words:

> AI-00 is not decomposable into shippable slices. Its four leaves have a strict dependency chain — the module must exist before a package can go in it, and a package must exist before a guard can scan it — and none of the three intermediate states is mergeable on its own. A PR containing `go.mod` and a `Makefile` with no package fails its own acceptance ("`make test` runs and passes inside the new module") vacuously; a PR containing the package but not the guards merges an unguarded module, which is precisely the failure mode ADR 0005 exists to prevent. The chain is also *short*: 520 lines of which about 60 % is copied build configuration reviewed by diffing it against its source, and about 40 % is two test files. The review cost is far below what the line count suggests.

Two of the four leaves — AI-00.3 and AI-00.4 — are explicitly parallel and touch disjoint modules, so they are separate commits within the PR and can be reviewed independently.

---

## Rollback plan

Required by `openspec/config.yaml`. Three levels, all cheap, because the change is almost purely additive:

1. **Revert the reverse-guard commit alone.** `backend/database_administrator/src/domain/imports_test.go` returns to its previous content. Nothing else in that module referenced the new tests. `make test` there returns to the recorded pre-change baseline. This is the only file this change modifies outside the new module, so it is the only revert that can affect existing behavior — and it cannot, because a test file has no production callers.
2. **Revert the `go.work` commit.** The two existing modules leave workspace mode and resolve dependencies exactly as they do today from their own `go.mod`/`go.sum`. `backend/agent` still builds independently — a module needs no workspace. This is the only part of the change with any chance of perturbing the existing modules' build lists, and it is a single-file deletion.
3. **Delete `backend/agent/` entirely.** Nothing imports it: Go enforces that no other module can, and the reverse guard proves that no package in `database_administrator` names it. Removing the directory leaves the repository exactly as it is today.

There is no migration, no persisted state, no running process, no container, and no published artifact. A full revert is `git revert` of at most four commits, in any order, with no coordination.

**Forward-fix preference.** If a guard turns out to be over-strict after merge — for example if AI-24's transport genuinely needs an allowlist entry — the correct move is to amend the allowlist in that milestone's own change, with its own ADR, not to revert the guard. Reverting a guard to unblock a dependency is the exact failure ADR 0005 § Guard A was written against.

---

## Dependencies

- **No new top-level Go dependency in any module.** `backend/agent/go.mod` has zero requires; `database_administrator/go.mod` and `workspace_syncer/go.mod` are not edited. `openspec/AGENTS.md` rule 5 (new dependency ⇒ ADR first) is therefore not triggered by this change — and the two future dependencies it defers are each already gated: AI-24 by its own ADR, AI-37 by ADR 0005 § D3.
- **Toolchain**: Go 1.26.3 (matches the local toolchain and both existing modules). `golangci-lint v2.9.0`, auto-installed into `backend/agent/bin/` by `make tools`, ignored by `.gitignore`.
- **Upstream artifacts**: ADR 0005 and ADR 0006 are merged. Doc 0002's AI-00 charter is the normative scope.

---

## Risks

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| 1 | `go.work` perturbs the two existing modules' build lists | low | Pre-change `make test` baselines captured for both modules; re-run and compared after `go.work` lands. `go.work.sum` committed if generated. Rollback level 2 is a single-file deletion |
| 2 | The forward guard passes vacuously — a doc-only package has almost no dependency graph | high | The leaf does not close on green. It closes on three recorded reds against scratch violations, then green |
| 3 | A scratch violation is committed by accident | low | The PR diff is the check: no scratch file and no `require` line may appear in it. `go.mod` with zero requires and a clean `git status` are recorded evidence |
| 4 | Build-config drift between the three copied `Makefile`s over time | medium (certain, eventually) | Accepted by ADR 0005 § D2. Mitigated by naming `test` and `lint` in the spec so divergence is a spec violation, not an unnoticed edit |
| 5 | A future contributor creates `src/agent/` empty "to be ready" | medium | The reason is stated in the spec and in `backend/agent/README.md`; the forward guard's forbidden-prefix coverage is what breaks |
| 6 | The reverse guard's module-scope test is read as transitive and later "fixed" to use `-deps` | medium | The design records why direct-imports-only is correct (row 5), and the test carries the reason in a comment naming ADR 0005 § D1 row 5 |
| 7 | The one-PR decision makes review superficial | medium | Four separate commits, one per leaf; two of them touch disjoint modules and are independently reviewable. The budget exception states the reasoning in the PR description per doc 0001 § 9 |

---

## Success criteria

The change is shippable when:

- [ ] `cd backend/agent && make test` is green (recorded output in the PR).
- [ ] `cd backend/agent && make lint` is clean (recorded output in the PR).
- [ ] `cd backend/agent && go build ./...` succeeds; same from the other two module directories.
- [ ] `backend/agent/go.mod` contains exactly a `module` line and a `go 1.26.3` line — zero `require` directives, and no `go.sum` exists.
- [ ] `backend/agent/src/agenttest/` is a direct sibling of `backend/agent/src/ai/`.
- [ ] `backend/agent/src/agent/`, `src/coding/` and `src/cmd/` do not exist.
- [ ] `go.work` lists all three modules; each module builds from its own directory.
- [ ] `cd backend/database_administrator && make test` differs from the recorded pre-change baseline **only** by the added reverse-guard tests, all passing.
- [ ] `cd backend/workspace_syncer && make test` matches its recorded pre-change baseline.
- [ ] `git diff` on `backend/database_administrator/src/tools/tools.go` is empty (recorded).
- [ ] `backend/database_administrator/go.mod` contains no `replace` directive (recorded).
- [ ] Four recorded guard-bite reds are in the PR description: three for the forward guard, one for the reverse guard. No scratch violation appears in the merged diff.
- [ ] `backend/agent/README.md` states the purpose, the three-layer contents, the dependency rule, and the run-`make test`-here-yourself note.
- [ ] The PR description states why the change exceeds the review budget (doc 0001 § 9, Process).

---

## Known stale cross-reference (do not fix here)

[ADR 0005 § Migration](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#migration) assigns the module move to milestone **AI-39** (`cachicamas-agent-module-promotion`) and says it must land before AI-17. Neither identifier means what the ADR meant any more: doc 0002 renumbered Layer 1 from zero and **folded the module promotion into AI-00**, which is this change. AI-39 in the current plan is a Wave 6 hand-off milestone, and the ADR's link into doc 0002 points at an anchor that no longer exists.

Doc 0002 § "What changed from the retired plan" already records this: *"ADR 0005's Context and Migration sections are now stale. They describe seventeen shipped milestones and assign the move to AI-39. The decisions themselves (D1–D4, Enforcement) stand unchanged and this document implements them; the narrative around them describes a history that was undone. Amending the ADR is out of this document's scope and worth doing separately."*

This change implements D2, D3 and Enforcement, which are the parts that stand. It **does not** amend the ADR's narrative — that is a separate change with its own review, and silently editing a merged ADR from inside an implementation PR is the wrong shape. The reference is recorded here so a reviewer following the ADR's link does not conclude that AI-00 is doing something unplanned.

---

## Notes for the `sdd-spec` phase

One spec, `specs/agent-module-scaffold/spec.md`, with `R-AGM-0NN` requirement IDs and `S-AGM-0NN` scenario IDs. RFC 2119 keywords and Given/When/Then per `openspec/config.yaml`. Every scenario must be independently verifiable by one recorded command. Requirements must cover, at minimum: module identity and zero dependencies; build-tooling parity with the named load-bearing targets; the workspace file; the package and test-package layout including the three directories that must *not* exist; the forward guard including its bite proof and the deny-by-default property; the reverse guard including its bite proof and its direct-imports-only property; and non-regression of the two existing modules (baseline comparison, byte-identical `tools.go`, no `replace`).

## Notes for the `sdd-design` phase

`design.md` must carry: the exact file tree with every file's role; the forward guard algorithm as pseudocode including the `-test` flag, the synthesized-package normalization, the three allowlist groups and the forbidden-prefix table; the reverse guard algorithm including why it is direct-imports-only; the `go.work` layout and its build-list risk; the rationale for a copied rather than shared build config together with the drift risk ADR 0005 records; and the recorded-evidence protocol for the four guard bites.

## Notes for the `sdd-tasks` phase

The four leaves of AI-00's graph become the four phases: AI-00.1 module skeleton `[mechanical]`, AI-00.2 package and test-package layout `[mechanical]`, AI-00.3 forward guard `[guard]`, AI-00.4 reverse guard `[guard]`. AI-00.3 and AI-00.4 are explicitly parallel and touch disjoint modules. Each task states its RED step, its GREEN step and its evidence command. The evidence gate is `make test` in `backend/agent/`, with AI-00.4 as the one documented exception — its gate is `make test` in `backend/database_administrator/`. The mechanical leaves are exempt from red-green but not from their recorded Check lists. `tasks.md` must restate the review budget exception.
