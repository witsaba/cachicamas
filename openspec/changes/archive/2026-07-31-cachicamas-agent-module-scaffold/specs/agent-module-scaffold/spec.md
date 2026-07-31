# Spec — Agent module scaffold and boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 of [doc 0002](../../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Phase**: spec (new capability)
> **Canonical path** (after `sdd-archive`): `openspec/specs/agent-module-scaffold/spec.md`
> **Format**: Given/When/Then + RFC 2119 keywords per `openspec/config.yaml`
> **Identifier convention**: requirements `R-AGM-0NN`, scenarios `S-AGM-0NN`. Append-only, per doc 0002's identifier rule.
> **Dependency convention**: the module under specification has **zero** third-party dependencies. Every "MUST NOT import" clause below is normative, not aspirational.

## Purpose

Define the acceptance criteria for the creation of the `backend/agent` Go module — its identity, its build tooling, its package layout, the repo-root workspace file — and for the two executable guards that hold the module boundary in both directions.

The module contains no behavior. Its entire deliverable surface is a boundary plus the tooling that proves the boundary holds. Consequently most requirements below are structural, and the two that are not (R-AGM-005, R-AGM-006) close only on recorded evidence that the guard **fails** against a deliberate violation before it lands green.

## Architecture invariants

These hold for the lifetime of the module, not only at the moment this change merges. A later milestone that breaks one of them is in violation of ADR 0005, not merely of this spec.

- **Zero dependencies until an ADR grants one.** `backend/agent/go.mod` carries no `require` directive. AI-24 adds a transport behind its own ADR gate; AI-37 adds the OpenTelemetry **API** modules, pre-authorized by [ADR 0005 § D3](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) and by nothing else.
- **`src/agenttest/` is a direct sibling of `src/ai/`.** AI-20.4's signature guard resolves `../ai/provider.go` from `runtime.Caller(0)`. Any other layout breaks it silently ([ADR 0005 § D2](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2), Guard C).
- **No import reaches out of the module.** Go makes a sibling-module import a build error; the forward guard covers everything Go does not — vendor SDKs, the OTel SDK, and transitive dependencies of either.
- **No import reaches back in.** Only `database_administrator/src/application/` and `…/src/cmd/server` may ever name the agent module ([ADR 0005 § D1 row 5](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2)), and neither does so in v1.
- **No CI.** `.github/workflows/` is absent. Every gate in this spec runs when a human runs `make test` inside a module directory, and "recorded green" means output pasted into the PR.

---

## Requirements

### R-AGM-001 — Module identity and zero dependencies

A new Go module MUST exist at `backend/agent/` with module path `github.com/cachicamas/backend/agent` and Go version `1.26.3`. Its `go.mod` MUST NOT declare any `require` directive, and no `go.sum` file MAY exist in the module. The module MUST NOT declare a `replace` directive.

#### Scenarios

- **S-AGM-001** — Given the repository at the merge commit of this change, when `backend/agent/go.mod` is read, then it contains exactly one `module` directive whose value is `github.com/cachicamas/backend/agent` AND exactly one `go` directive whose value is `1.26.3` AND zero `require` directives AND zero `replace` directives.
- **S-AGM-002** — Given the merged tree, when `ls backend/agent/go.sum` runs, then no such file exists.
- **S-AGM-003** — Given the merged tree, when `cd backend/agent && go build ./...` runs, then it exits 0 and downloads nothing.
- **S-AGM-004** — Given the merged tree, when `cd backend/agent && go list -deps -test ./... | grep -v '^vendor/'` is compared against `go list std`, then every remaining import path is either a member of the `go list std` set or has the prefix `github.com/cachicamas/backend/agent/`. (This is R-AGM-005's allowlist stated as a one-shot structural check; the guard is what keeps it true.)

### R-AGM-002 — Build tooling parity

The module MUST carry its own `Makefile`, `.golangci.yml`, `.gitignore` and `README.md`. The `Makefile` MUST define a `test` target whose body is `go test -race -v ./...` and a `lint` target that runs `golangci-lint` at the pinned version `v2.9.0`. These two targets are load-bearing for the evidence gate of every milestone in doc 0002 and MUST NOT be renamed or redefined without amending this requirement. The build configuration is a **copy** of `database_administrator`'s and SHALL NOT be shared, extracted, or symlinked between modules.

The `README.md` MUST state the module's one-paragraph purpose, its three-layer contents, the dependency rule, and the fact that `make test` must be run in this directory separately because no CI exists.

#### Scenarios

- **S-AGM-010** — Given `backend/agent/Makefile`, when the `test` target is read, then its recipe is `$(GO) test -race -v ./...` (or a literal equivalent that expands to `go test -race -v ./...`).
- **S-AGM-011** — Given `backend/agent/Makefile`, when the `lint` target is read, then it invokes `golangci-lint run --config=.golangci.yml ./...` AND the pinned version variable resolves to `v2.9.0`.
- **S-AGM-012** — Given the merged tree, when `cd backend/agent && make test` runs, then it exits 0 and reports no failing test. An empty pass is a valid result at this milestone and is still evidence that the tooling is wired.
- **S-AGM-013** — Given the merged tree, when `cd backend/agent && make lint` runs, then it exits 0 with no reported issue.
- **S-AGM-014** — Given `backend/agent/.golangci.yml`, when it is diffed against `backend/database_administrator/.golangci.yml`, then the two are identical.
- **S-AGM-015** — Given `backend/agent/.gitignore`, when it is read, then it ignores `bin/` AND the coverage artifacts, so that `make tools` installing `golangci-lint` into `backend/agent/bin/` never dirties the tree.
- **S-AGM-016** — Given `backend/agent/README.md`, when it is read, then it contains the module's purpose paragraph, the names of the three layers and their directories, the dependency rule with its ADR reference, and an explicit statement that `make test` must be run in this directory because no CI exists.
- **S-AGM-017** — Given the merged tree, when the repository is searched for a build or lint configuration shared between two or more backend modules, then none exists: each module owns its own `Makefile` and `.golangci.yml` file, and no module's config includes or references another's.

### R-AGM-003 — Repository workspace file

A `go.work` file MUST exist at the repository root and MUST list all three backend modules: `./backend/agent`, `./backend/database_administrator`, `./backend/workspace_syncer`. Each module MUST continue to build independently from its own directory. If the toolchain generates a `go.work.sum`, it MUST be committed. The root `.gitignore` MUST NOT exclude `go.work` or `go.work.sum`.

#### Scenarios

- **S-AGM-020** — Given the merged tree, when `go.work` is read, then its `use` block names exactly the three module directories above, and it declares `go 1.26.3`.
- **S-AGM-021** — Given the merged tree, when `go build ./...` runs from each of the three module directories in turn, then each exits 0.
- **S-AGM-022** — Given a `go.work.sum` generated by the toolchain during any command in this change, when the PR diff is inspected, then that file is present in the diff.
- **S-AGM-023** — Given the root `.gitignore`, when it is read, then it contains no pattern matching `go.work` or `go.work.sum`, AND `git status --porcelain` after a full `make test` in all three modules reports no untracked `go.work*` file.

### R-AGM-004 — Package and test-package layout

The module MUST contain `src/ai/` as its Layer 1 package, carrying package documentation that states the layer boundary and the import rule **and nothing else** — no type, no constant, no function. The module MUST contain `src/agenttest/` as a **direct sibling** of `src/ai/`, holding an external-package test that imports `github.com/cachicamas/backend/agent/src/ai` and compiles.

The module MUST NOT contain `src/agent/`, `src/coding/`, or `src/cmd/`, in any form, including as an empty directory or a directory holding only a placeholder file. Creating any of them would make the forward guard's forbidden-prefix list untestable, because a prefix that matches an existing package can no longer be proven forbidden.

#### Scenarios

- **S-AGM-030** — Given the merged tree, when `backend/agent/src/` is listed, then it contains exactly two entries, `ai` and `agenttest`, and both are directories.
- **S-AGM-031** — Given `backend/agent/src/ai/`, when its Go declarations are enumerated (for example with `go doc github.com/cachicamas/backend/agent/src/ai`), then the package exports nothing and declares nothing beyond the package clause and its documentation comment.
- **S-AGM-032** — Given `backend/agent/src/ai/`'s package documentation, when it is read, then it states the Layer 1 boundary and the import rule, and cites ADR 0005 § D1.
- **S-AGM-033** — Given `backend/agent/src/agenttest/`, when its test files are read, then at least one declares an external test package (`package agenttest_test`) AND imports `github.com/cachicamas/backend/agent/src/ai`.
- **S-AGM-034** — Given the merged tree, when `cd backend/agent && go test ./src/agenttest/...` runs, then it compiles and exits 0, proving `src/ai` is importable from outside its own package.
- **S-AGM-035** — Given the merged tree, when `test -e backend/agent/src/agent`, `test -e backend/agent/src/coding` and `test -e backend/agent/src/cmd` each run, then each exits non-zero — none of the three exists.
- **S-AGM-036** — Given `backend/agent/src/agenttest/` and `backend/agent/src/ai/`, when their parent directories are compared, then both resolve to `backend/agent/src`, so `../ai` from any file in `agenttest` resolves to the Layer 1 package directory. This is the layout invariant AI-20.4's signature guard will depend on.

### R-AGM-005 — Forward guard: Layer 1 purity

The module MUST carry an executable guard at `backend/agent/src/ai/import_boundary_test.go` that fails when any package of `backend/agent` — including its test packages and its transitive dependencies — depends on anything outside an explicit allowlist.

The guard MUST use `go list -deps -test` over the module's own package pattern. Bare `go list -deps` is insufficient: it does not report test-only imports, and `go list -f '{{range .Imports}}'` reports neither test imports nor transitive dependencies ([ADR 0005 § Guard A](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement), finding S6).

The allowlist MUST be **deny-by-default**: an import path that is neither a member of the Go standard library set nor a package of this module fails the guard *even when no rule names it*. The allowlist MUST be expressed as three groups — the standard library, this module's own packages, and a third-party group that MUST be empty at this milestone and whose only permitted growth path is a milestone with its own ADR (AI-24 for a transport, AI-37 for the OpenTelemetry API per ADR 0005 § D3).

In addition to the allowlist, the guard MUST carry a named forbidden-prefix list covering: `github.com/cachicamas/backend/database_administrator`, `github.com/cachicamas/backend/workspace_syncer`, `github.com/cachicamas/backend/agent/src/agent`, `github.com/cachicamas/backend/agent/src/coding`, `github.com/cachicamas/backend/agent/src/cmd`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters`, and `go.opentelemetry.io/contrib/bridges/otelslog`. The OpenTelemetry **API** paths of ADR 0005 § D3 MUST NOT appear in the forbidden list.

The guard closes only on **bite proof**: it MUST be shown to fail against three separate deliberate violations, with the failing output recorded, before it lands green.

#### Scenarios

- **S-AGM-040** — Given the merged tree with no violation present, when `cd backend/agent && go test ./src/ai/...` runs, then the guard test passes.
- **S-AGM-041** — Given the guard's source, when its `go list` invocation is read, then it passes both `-deps` and `-test`, and its pattern covers every package of the module.
- **S-AGM-042** — Given the guard's source, when its allowlist is read, then the standard-library group is derived from the toolchain (for example the exact set returned by `go list std`) rather than from a "path segment contains no dot" heuristic, AND the third-party group is present, empty, and commented with the ADR-gated milestones that may add to it.
- **S-AGM-043** — Given the guard's source, when its forbidden-prefix list is read, then it names all eight prefixes above AND contains no OpenTelemetry API path (`go.opentelemetry.io/otel`, `…/otel/trace`, `…/otel/attribute`, `…/otel/codes`, `…/otel/metric`).
- **S-AGM-044** — **(bite 1)** Given a scratch file in `backend/agent/src/ai/` that imports `github.com/cachicamas/backend/database_administrator/src/domain`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. The failing output is recorded in the PR; the scratch file is then removed.
- **S-AGM-045** — **(bite 2)** Given a scratch file in `backend/agent/src/ai/` that imports `go.opentelemetry.io/otel/sdk/trace`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. Recorded, then removed.
- **S-AGM-046** — **(bite 3)** Given a scratch file in `backend/agent/src/ai/` that imports an arbitrary third-party module named by no forbidden prefix, when `go test ./src/ai/...` runs, then the guard FAILS **on the deny-by-default allowlist rule**, not on a named prefix. This is what proves the allowlist is deny-by-default and makes AI-24's transport choice a visible, ADR-gated event rather than a quiet `go get`. Recorded, then removed together with any `require` line its presence forced into `go.mod`.
- **S-AGM-047** — Given the merged PR diff, when it is inspected, then no scratch violation file appears in it AND `backend/agent/go.mod` still declares zero `require` directives.
- **S-AGM-048** — Given the guard's source, when its handling of `go list -deps -test` output is read, then it normalizes the synthesized entries the toolchain emits — the ` [<pkg>.test]` test-variant suffix and the `<pkg>.test` binary — so that neither is mistaken for a real import path.

### R-AGM-006 — Reverse guard: nothing reaches back

`backend/database_administrator` MUST carry an executable guard, in `src/domain/imports_test.go`, with two assertions:

1. The existing forbidden-prefix test over `src/domain` MUST be extended to name `github.com/cachicamas/backend/agent`. This assertion is **transitive** (`go list -deps`): `src/domain` MUST NOT reach the agent module by any path.
2. A new module-scope assertion MUST verify that no package of `database_administrator` outside `src/application/` and `src/cmd/server` **names** the agent module. This assertion MUST scan **direct imports only** — `.Imports`, `.TestImports` and `.XTestImports` — and MUST NOT be transitive, because [ADR 0005 § D1 row 5](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) permits `src/application` and `src/cmd/server` to import the agent module, and a transitive scan would therefore indict every consumer of `application` the moment row 5 is exercised.

Both assertions are green from birth, because the current count of such imports is zero. That is the point: the guard fails on the first accidental import rather than on the first production incident ([ADR 0005 § Guard B](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement)).

The guard closes only on **bite proof**: one deliberate violation, recorded failing, then removed.

**Evidence gate exception.** Unlike every other requirement in this spec, this requirement's gate is `make test` in `backend/database_administrator/`, not in `backend/agent/`. This is the single documented exception in doc 0002's global evidence gate.

#### Scenarios

- **S-AGM-050** — Given the merged tree, when `cd backend/database_administrator && go test ./src/domain/...` runs, then both reverse-guard assertions pass.
- **S-AGM-051** — Given the domain assertion's source, when its `go list` invocation is read, then it uses `-deps` over `github.com/cachicamas/backend/database_administrator/src/domain` and its forbidden set includes `github.com/cachicamas/backend/agent`.
- **S-AGM-052** — Given the module-scope assertion's source, when its `go list` invocation is read, then it requests `.Imports`, `.TestImports` and `.XTestImports` over the pattern `github.com/cachicamas/backend/database_administrator/...` AND does **not** pass `-deps`, AND a comment states that row 5 is the reason.
- **S-AGM-053** — Given the module-scope assertion's source, when its exemption set is read, then it exempts exactly the package prefixes `…/database_administrator/src/application` and `…/database_administrator/src/cmd/server`, and nothing else.
- **S-AGM-054** — Given the merged tree today, when the module-scope assertion runs, then it inspects every package of the module (13 at the time of writing) and finds zero packages naming the agent module.
- **S-AGM-055** — **(bite)** Given a scratch import of `github.com/cachicamas/backend/agent/src/ai` added to a file in `backend/database_administrator/src/domain/`, when `cd backend/database_administrator && go test ./src/domain/...` runs, then the guard FAILS and its message names both the offending package and the forbidden module path. The failing output is recorded in the PR; the scratch import is then removed.
- **S-AGM-056** — Given the merged PR diff, when it is inspected, then no scratch import appears in any `database_administrator` file AND `backend/database_administrator/go.mod` is unmodified.

### R-AGM-007 — Non-regression of the existing modules

This change MUST NOT alter the behavior of `backend/database_administrator` or `backend/workspace_syncer`.

`backend/database_administrator/go.mod` MUST NOT gain a `replace` directive. [ADR 0005 § Migration](../../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#migration): a `replace` with no requirer is dead weight that also disguises the D1 row-5 build cost.

`backend/database_administrator/src/tools/tools.go` MUST remain byte-identical. S3 is resolved by *vacating* the name `tools`, not by relocating a standard-idiom file, and the change MUST record a diff proving the file is untouched.

A `make test` baseline MUST be captured for both existing modules **before** any file of this change lands, and compared afterwards. The comparison point differs per phase, and stating it precisely is what makes the requirement checkable: after the module skeleton and layout land, both modules MUST produce the same set of test names with the same outcomes as their baseline (timings excepted); after the reverse guard lands, `database_administrator`'s result set MUST differ from its baseline **only** by the added reverse-guard tests, and `workspace_syncer`'s MUST still match exactly.

#### Scenarios

- **S-AGM-060** — Given the pre-change tree, when `cd backend/database_administrator && make test` and `cd backend/workspace_syncer && make test` are run and their outputs saved, then both baselines are recorded as artifacts of this change before any file is created.
- **S-AGM-061** — Given the tree after `go.work`, the new module and its packages exist but before the reverse guard is added, when `make test` is re-run in both existing modules, then each produces the same set of test names with the same pass/fail outcomes as its baseline.
- **S-AGM-062** — Given the merged tree, when `cd backend/workspace_syncer && make test` runs, then its result set matches the S-AGM-060 baseline exactly.
- **S-AGM-063** — Given the merged tree, when `cd backend/database_administrator && make test` runs, then its result set equals the S-AGM-060 baseline plus exactly the reverse-guard tests of R-AGM-006, all passing, and contains no removed or newly failing test.
- **S-AGM-064** — Given the merged tree, when `git diff <base>..HEAD -- backend/database_administrator/src/tools/tools.go` runs, then it produces empty output, and that output is recorded in the PR.
- **S-AGM-065** — Given the merged tree, when `backend/database_administrator/go.mod` is read, then it contains no `replace` directive.
- **S-AGM-066** — Given the merged PR diff, when the set of changed files under `backend/database_administrator/` is enumerated, then it contains exactly one entry: `src/domain/imports_test.go`.
- **S-AGM-067** — Given the merged PR diff, when the set of changed files under `backend/workspace_syncer/` is enumerated, then it is empty.

---

## Non-functional requirements

### NFR-AGM-001 — Guard determinism

Both guards MUST be deterministic and hermetic: they shell out to the local `go` toolchain and read no network, no environment-specific path, and no file outside the repository. They MUST pass under `-race` (the evidence gate runs `go test -race -v ./...`) and MUST NOT depend on the working directory from which `make test` is invoked — package patterns are given as fully-qualified module paths rather than relative `./...` patterns resolved from an assumed cwd.

### NFR-AGM-002 — Guard legibility

Each guard's failure message MUST name the offending import path and the rule that rejected it (deny-by-default allowlist, or a specific forbidden prefix), so that a failure five milestones from now is actionable without reading the guard's source. Each guard MUST cite the ADR clause it enforces in a comment.

### NFR-AGM-003 — Review budget

Doc 0002 prefers under 250 changed lines and requires reassessment before 400. This change ships as one PR and exceeds that budget at roughly 520 changed lines. Per doc 0001 § 9 (Process), the PR description MUST state why the change does not fit; the reasoning is in `proposal.md` § "Review budget exception" and is restated in `tasks.md`.

---

## Strict TDD posture

`openspec/config.yaml` declares `apply.tdd: true`, and `openspec/AGENTS.md` states that a test written after the implementation does not satisfy this repository's discipline. Doc 0002 refines that for the node types in play here:

- **Guard leaves (R-AGM-005, R-AGM-006)** are RED-first by construction. A guard that has only ever been green is unproven, so the closing condition is the recorded red run against a deliberate violation, followed by green. Four reds are required in total: three for the forward guard (S-AGM-044, S-AGM-045, S-AGM-046) and one for the reverse guard (S-AGM-055).
- **Mechanical leaves (R-AGM-001 through R-AGM-004, R-AGM-007)** are exempt from red-green — there is no behavior to drive out — but are **never** exempt from their recorded Check lists. Each scenario above is a check whose recorded output (command output, file content, diff) is the evidence.

## Out of scope (per proposal)

- Any dependency at all, including the OpenTelemetry API — AI-37; and the transport — AI-24. Each behind its own ADR gate.
- Any Layer 1 contract content: vocabulary (AI-01), stream lifecycle and carrier (AI-02), capability scope (AI-03), the first type (AI-04).
- `src/agent/` (Layer 2, doc 0003), `src/coding/` (Layer 3, doc 0004), `src/cmd/cachicamas/` (doc 0004) — not created, not stubbed.
- AI-20.4's signature guard. This change guarantees only the layout it will depend on.
- AI-25.2's ambient-authority scan over the adapter package.
- Exercising ADR 0005 § D1 row 5. A non-goal for all of v1; it breaks `docker compose build` and ADR 0005 prices the fix as its own change.
- Adding CI. `.github/workflows/` stays absent; ADR 0005 records this under Consequences.
- Amending ADR 0005's stale § Migration narrative (it assigns the module move to a milestone identifier that no longer exists). Doc 0002 already records the staleness; amending the ADR is a separate change.

## Acceptance criteria

The change is accepted when:

1. Every scenario S-AGM-001 through S-AGM-067 has recorded evidence.
2. `cd backend/agent && make test` and `make lint` are green, recorded in the PR.
3. `backend/agent/go.mod` declares zero requires and no `go.sum` exists.
4. All four guard bites are recorded red in the PR, and no scratch violation appears in the merged diff.
5. `cd backend/workspace_syncer && make test` matches its pre-change baseline; `cd backend/database_administrator && make test` differs from its baseline only by the added reverse-guard tests.
6. `backend/database_administrator/src/tools/tools.go` is byte-identical (empty diff recorded) and its `go.mod` has no `replace`.
7. `go.work` lists all three modules and each builds independently from its own directory.
8. The PR description states why the change exceeds the review budget.
