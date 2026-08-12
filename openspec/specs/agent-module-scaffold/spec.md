# Spec — Agent module scaffold and boundary guards

> **Milestone**: AI-00 — Create the module and both boundary guards, of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Introduced by**: `openspec/changes/archive/2026-07-31-cachicamas-agent-module-scaffold/`, merged in PR #95 at commit `a831c06` on 2026-07-31
> **Status**: **live** — the invariants below hold for the lifetime of the module, not only at the moment AI-00 merged
> **Format**: Given/When/Then + RFC 2119 keywords per `openspec/config.yaml`
> **Identifier convention**: requirements `R-AGM-0NN`, scenarios `S-AGM-0NN`. Append-only, per doc 0002's identifier rule.
> **Dependency convention**: the module under specification carries **no dependency an ADR has not authorised**. It had zero third-party dependencies from AI-00 through AI-36; AI-37 is the first and, per ADR 0005 § D3's closing blockquote, the second-and-last milestone permitted to add one. Every "MUST NOT import" clause below is normative, not aspirational.

## Purpose

Define the contract for the `backend/agent` Go module — its identity, its build tooling, its package layout, the repo-root workspace file — and for the two executable guards that hold the module boundary in both directions.

The module contains no behavior. Its entire deliverable surface is a boundary plus the tooling that proves the boundary holds. Consequently most requirements below are structural, and the two that are not (`R-AGM-005`, `R-AGM-006`) closed only on recorded evidence that the guard **fails** against a deliberate violation before it landed green.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The module contract therefore lives here. The archived change folder at [`openspec/changes/archive/2026-07-31-cachicamas-agent-module-scaffold/`](../../changes/archive/2026-07-31-cachicamas-agent-module-scaffold/) is the historical record of how AI-00 was explored, proposed, designed, applied and verified — including the four recorded guard bites and the two doc 0002 amendments the implementation forced.

A later milestone that needs to change one of these invariants — most concretely AI-24 adding a transport allowlist entry, or AI-37 adding the OpenTelemetry API modules — amends **this file**, in the same pull request, under its own ADR gate. Reverting a guard to unblock a dependency is the exact failure [ADR 0005 § Guard A](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement) was written against.

> **Amended 2026-08-08 (AI-37)** by [`cachicamas-ai-observability`](../../changes/archive/2026-08-08-cachicamas-ai-observability/) (AI-37 — Add the observability boundary, Wave 5 — Harden). Three requirements amended in place — `R-AGM-001` (an exact, pinned require set by set equality replaces "zero requires"), `R-AGM-003` (a workspace-divergence obligation, since a sibling module already pinned a different version of the same ecosystem), `R-AGM-005` (the third-party allowlist admits exactly the OpenTelemetry **API** paths of ADR 0005 § D3 plus their forced transitive closure, still deny-by-default) — and one requirement added, `R-AGM-008` (the non-standard-library package closure pinned by **set equality**, not prefix subset, because the allowlist matcher is prefix-based). Six new scenarios: `S-AGM-068` … `S-AGM-073`. Every scenario named in the three `MODIFIED` blocks below restates its full text, including scenarios whose wording did not change, per this repo's promotion discipline; no scenario not named below changed.
>
> **Amended 2026-08-12 (AG-03)** by [`cachicamas-agent-package-scaffold`](../../changes/archive/2026-08-12-cachicamas-agent-package-scaffold/) (AG-03 — Package scaffold and boundary guards, Layer 2, Wave 1). Two requirements amended in place — `R-AGM-004` (now asserts `src/agent/`'s existence post-AG-03 per its own capability spec, and clarifies the testability relationship with its forbidden-prefix row) and `R-AGM-005` (now asserts complete Layer 1 coverage independent of the chosen mechanism for handling the Layer 2 self-reference hazard, in place of a pattern-specific assertion). Two scenarios amended: `S-AGM-035` and `S-AGM-041`. Every scenario named below restates its full text; no scenario not named changed.

## Architecture invariants

These hold for the lifetime of the module, not only at the moment AI-00 merged. A later milestone that breaks one of them is in violation of ADR 0005, not merely of this spec.

- **No dependency an ADR has not authorised.** `backend/agent/go.mod` carried zero `require` directives from AI-00 through AI-36. AI-37 landed the first: the OpenTelemetry **API** modules, pre-authorized by [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) and by nothing else, pinned by an exact require-set assertion (`R-AGM-001`) and a separate exact package-closure assertion (`R-AGM-008`). "Zero requires" was never the point; "no unauthorised dependency, provable mechanically" was — the pinned assertions are strictly stronger than the zero-count check they replace, because a zero-count check could never have distinguished an authorised path from its unauthorised transitive closure. AI-24 added no dependency (raw `net/http`, stdlib only); ADR 0005 § D3's closing blockquote makes AI-37 the second-and-last milestone permitted to add one.
- **`src/agenttest/` is a direct sibling of `src/ai/`.** AI-20.4's signature guard resolves `../ai/provider.go` from `runtime.Caller(0)`. Any other layout breaks it silently ([ADR 0005 § D2](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2), Guard C).
- **No import reaches out of the module.** Go makes a sibling-module import a build error; the forward guard covers everything Go does not — vendor SDKs, the OTel SDK, and transitive dependencies of either.
- **No import reaches back in.** Only `database_administrator/src/application/` and `…/src/cmd/server` may ever name the agent module ([ADR 0005 § D1 row 5](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2)), and neither does so in v1.
- **No CI.** `.github/workflows/` is absent. Every gate in this spec runs when a human runs `make test` inside a module directory, and "recorded green" means output pasted into the PR.

---

## Requirements

### R-AGM-001 — Module identity and its pinned dependency set

*(MODIFIED 2026-08-08 by AI-37 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. Was: "Module identity and zero dependencies", asserting `go.mod` declared no `require` and no `go.sum` existed.)*

A Go module MUST exist at `backend/agent/` with module path `github.com/cachicamas/backend/agent` and Go version `1.26.3`.

Its `go.mod` MUST declare **exactly** the require entries that an ADR authorises and no others — at AI-37 that is the OpenTelemetry **API** modules pre-authorised by [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), together with the indirect requirement their own closure forces. The require set MUST be asserted by **set equality against an in-test table of module paths and versions**, including every indirect entry, not by a count and not by a subset check. A require that no ADR authorises MUST fail the suite.

The module MUST NOT declare a `replace` directive.

A dependency-sum file MUST now exist in the module and MUST be committed. (Previously: the `go.mod` was required to declare zero requires and no dependency-sum file was permitted to exist; AI-37 is the milestone this file already named as the one that changes this, at `:20, 26` above.)

#### Scenarios

- **S-AGM-001** — Given the repository, when `backend/agent/go.mod` is read, then it contains exactly one `module` directive whose value is `github.com/cachicamas/backend/agent` AND exactly one `go` directive whose value is `1.26.3` AND a `require` set **equal** to the ADR-authorised table of module paths and versions including its indirect entries AND zero `replace` directives.
- **S-AGM-002** — Given the repository, when the module's dependency-sum file is looked for, then it exists and is tracked by version control, and no ignore rule excludes it.
- **S-AGM-003** — Given the repository with the module cache already populated, when `cd backend/agent && go build ./...` runs, then it exits 0 and neither adds, removes nor rewrites any `require` entry.
- **S-AGM-004** — Given the repository, when every non-standard-library package the module depends on, including test-only dependencies, is enumerated, then each is either a package of this module or a member of the ADR-authorised closure pinned by `R-AGM-008`. (This is `R-AGM-005`'s allowlist stated as a one-shot structural check; the guard is what keeps it true.)

### R-AGM-002 — Build tooling parity

The module MUST carry its own `Makefile`, `.golangci.yml`, `.gitignore` and `README.md`. The `Makefile` MUST define a `test` target whose body is `go test -race -v ./...` and a `lint` target that runs `golangci-lint` at the pinned version `v2.9.0`. These two targets are load-bearing for the evidence gate of every milestone in doc 0002 and MUST NOT be renamed or redefined without amending this requirement. The build configuration is a **copy** of `database_administrator`'s and SHALL NOT be shared, extracted, or symlinked between modules.

The `README.md` MUST state the module's one-paragraph purpose, its three-layer contents, the dependency rule, and the fact that `make test` must be run in this directory separately because no CI exists.

#### Scenarios

- **S-AGM-010** — Given `backend/agent/Makefile`, when the `test` target is read, then its recipe is `$(GO) test -race -v ./...` (or a literal equivalent that expands to `go test -race -v ./...`).
- **S-AGM-011** — Given `backend/agent/Makefile`, when the `lint` target is read, then it invokes `golangci-lint run --config=.golangci.yml ./...` AND the pinned version variable resolves to `v2.9.0`.
- **S-AGM-012** — Given the repository, when `cd backend/agent && make test` runs, then it exits 0 and reports no failing test.
- **S-AGM-013** — Given the repository, when `cd backend/agent && make lint` runs, then it exits 0 with no reported issue.
- **S-AGM-014** — Given `backend/agent/.golangci.yml`, when it is diffed against `backend/database_administrator/.golangci.yml`, then the two are identical.
- **S-AGM-015** — Given `backend/agent/.gitignore`, when it is read, then it ignores `bin/` AND the coverage artifacts, so that `make tools` installing `golangci-lint` into `backend/agent/bin/` never dirties the tree.
- **S-AGM-016** — Given `backend/agent/README.md`, when it is read, then it contains the module's purpose paragraph, the names of the three layers and their directories, the dependency rule with its ADR reference, and an explicit statement that `make test` must be run in this directory because no CI exists.
- **S-AGM-017** — Given the repository, when it is searched for a build or lint configuration shared between two or more backend modules, then none exists: each module owns its own `Makefile` and `.golangci.yml` file, and no module's config includes or references another's.

### R-AGM-003 — Repository workspace file

*(MODIFIED 2026-08-08 by AI-37 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. Adds the workspace-divergence obligation and `S-AGM-068`.)*

A `go.work` file MUST exist at the repository root and MUST list all three backend modules: `./backend/agent`, `./backend/database_administrator`, `./backend/workspace_syncer`. Each module MUST continue to build independently from its own directory. If the toolchain generates a `go.work.sum`, it MUST be committed. The root `.gitignore` MUST NOT exclude `go.work` or `go.work.sum`.

Because the workspace unifies the build list across all three modules, and a sibling module already pins a **different** version of an ecosystem this module now requires, the version resolved into `backend/agent/go.mod` and the version the workspace reports MAY differ. That divergence MUST fail the build rather than resolve silently: `R-AGM-001` pins the module's own require set and `R-AGM-008` pins the package closure **separately**, and both MUST hold. (Previously: the `go.work.sum` clause existed but had never been exercised, because no module in the workspace-scoped agent build carried a dependency.)

#### Scenarios

- **S-AGM-020** — Given the repository, when `go.work` is read, then its `use` block names exactly the three module directories above, and it declares `go 1.26.5`. (`go.work` is byte-unchanged by AI-37; this restates a prior milestone's version, re-verified by direct read at this round rather than asserted by any test — no guard in this module reads the `go.work` directive.)
- **S-AGM-021** — Given the repository, when `go build ./...` runs from each of the three module directories in turn, then each exits 0.
- **S-AGM-022** — Given a `go.work.sum` generated by the toolchain, when the diff of the change that generated it is inspected, then that file is present in the diff.
- **S-AGM-023** — Given the root `.gitignore`, when it is read, then it contains no pattern matching `go.work` or `go.work.sum`, AND the working tree after a full test run in all three modules reports no untracked workspace-sum file.
- **S-AGM-068** — Given this change, which is the first to make the workspace carry a dependency for this module, when the merged diff is inspected, then the generated workspace-sum file appears in it, the working tree is clean afterwards, and the version recorded in the module's own require set is asserted independently of whatever the workspace build list reports — a divergence between the two fails the suite rather than passing quietly.

### R-AGM-004 — Package and test-package layout

*(MODIFIED 2026-08-12 by AG-03 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. Previously: `src/agent/` was forbidden unconditionally alongside `src/coding/` and `src/cmd/`; doc 0003's AG-03 is the milestone this requirement already named as the one that creates it.)*

The module MUST contain `src/ai/` as its Layer 1 package. At AI-00 that package carried package documentation stating the layer boundary and the import rule **and nothing else** — no type, no constant, no function; from AI-04 onward it carries the Layer 1 contracts. The module MUST contain `src/agenttest/` as a **direct sibling** of `src/ai/`, holding an external-package test that imports `github.com/cachicamas/backend/agent/src/ai` and compiles.

From AG-03 onward the module MUST contain `src/agent/` as its Layer 2 package, created by doc 0003's AG-03 exactly as this requirement anticipated. Its contract is owned by `agent-package-scaffold`, not by this spec; this requirement asserts only its existence, its position as a direct sibling of `src/ai/` and `src/agenttest/`, and that its creation did not disturb the layout `R-AGM-004` guarantees.

The module MUST NOT contain `src/coding/` or `src/cmd/`, in any form, including as an empty directory or a directory holding only a placeholder file, until doc 0004 creates them. Creating either of them earlier would make the forward guard's forbidden-prefix list untestable, because a prefix that matches an existing package can no longer be proven forbidden **by absence**. `src/agent/` is the first prefix of that list to name a package that now exists; its row therefore MUST remain provable by a deliberate import bite instead (`R-AGM-005`, `S-AGM-041`), and MUST NOT be deleted.

#### Scenarios

- **S-AGM-030** — Given the repository at AI-00's merge, when `backend/agent/src/` is listed, then it contains exactly two entries, `ai` and `agenttest`, and both are directories.
- **S-AGM-031** — Given `backend/agent/src/ai/` at AI-00's merge, when its Go declarations are enumerated (for example with `go doc github.com/cachicamas/backend/agent/src/ai`), then the package exports nothing and declares nothing beyond the package clause and its documentation comment.
- **S-AGM-032** — Given `backend/agent/src/ai/`'s package documentation, when it is read, then it states the Layer 1 boundary and the import rule, and cites ADR 0005 § D1.
- **S-AGM-033** — Given `backend/agent/src/agenttest/`, when its test files are read, then at least one declares an external test package (`package agenttest_test`) AND imports `github.com/cachicamas/backend/agent/src/ai`.
- **S-AGM-034** — Given the repository, when `cd backend/agent && go test ./src/agenttest/...` runs, then it compiles and exits 0, proving `src/ai` is importable from outside its own package.
- **S-AGM-035** — *(AMENDED 2026-08-12 by AG-03)* Given the repository from AG-03's merge onward, when `test -e backend/agent/src/agent`, `test -e backend/agent/src/coding` and `test -e backend/agent/src/cmd` each run, then the first exits **zero** — `src/agent` exists, is a directory, and is a direct sibling of `src/ai` and `src/agenttest` — while the other two exit **non-zero**: neither `src/coding` nor `src/cmd` exists in any form, including as an empty directory or a directory holding only a placeholder file. (Before AG-03's merge all three exited non-zero; `R-AGM-004`'s prose named docs 0003 and 0004 as the milestones that change this, and AG-03 is doc 0003's.)
- **S-AGM-036** — Given `backend/agent/src/agenttest/` and `backend/agent/src/ai/`, when their parent directories are compared, then both resolve to `backend/agent/src`, so `../ai` from any file in `agenttest` resolves to the Layer 1 package directory. This is the layout invariant AI-20.4's signature guard depends on.

### R-AGM-005 — Forward guard: Layer 1 purity

*(MODIFIED 2026-08-12 by AG-03 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. `S-AGM-041` is amended because AG-03 may narrow the guard's scanned roots to resolve the self-reference hazard the guard's own source recorded at `:82-92`; the amended wording asserts complete Layer 1 coverage independently of which mechanism design selects. Previously amended 2026-08-08 by AI-37, which made the third-party allowlist group non-empty.)*

The module MUST carry an executable guard at `backend/agent/src/ai/import_boundary_test.go` that fails when any package of `backend/agent` — including its test packages and its transitive dependencies — depends on anything outside an explicit allowlist.

The guard MUST use `go list -deps -test`. Bare `go list -deps` is insufficient: it does not report test-only imports, and a direct-imports listing reports neither test imports nor transitive dependencies ([ADR 0005 § Guard A](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement), finding S6).

The guard's scanned set MUST cover **every Layer 1 package of the module** — `src/ai/…`, `src/agenttest/…` and `src/handoff` — and MUST use fully-qualified module-path patterns rather than relative patterns resolved from an assumed working directory. Because `go list` emits the scanned pattern's **own members**, and because Layer 2's package path is itself a forbidden prefix, the guard MUST distinguish a member of its own scanned set from a genuine import violation. Whether that is achieved by narrowing the scanned roots to Layer 1's packages or by exempting the pattern's own members from the prefix match is an implementation choice; the guard's source MUST record which mechanism it uses and why. Narrowing that drops coverage of any Layer 1 package is a violation of this requirement, not an implementation detail.

The allowlist MUST be **deny-by-default**: an import path that is neither a member of the Go standard library set nor a package of this module fails the guard *even when no rule names it*. The allowlist MUST be expressed as three groups — the standard library, this module's own packages, and a third-party group whose only permitted growth path is a milestone with its own ADR.

**The third-party group is no longer empty.** At AI-37 it admits exactly the OpenTelemetry **API** paths this module actually imports, plus the packages transitively required in order to import them at all. Every entry MUST carry, in the guard's own source, the ADR clause authorising it — ADR 0005 § D3 for each OpenTelemetry path. An entry that is not an OpenTelemetry module and is admitted only because an authorised path cannot be imported without it MUST record that reasoning in place: authorising an import path necessarily authorises its closure, and § D3's "any additional OTel module requires its own ADR" clause is not engaged by a non-OpenTelemetry transitive requirement. That reasoning is bounded by `R-AGM-008`'s set-equality pin, which is what stops it becoming a blank cheque.

In addition to the allowlist, the guard MUST carry a named forbidden-prefix list covering: `github.com/cachicamas/backend/database_administrator`, `github.com/cachicamas/backend/workspace_syncer`, `github.com/cachicamas/backend/agent/src/agent`, `github.com/cachicamas/backend/agent/src/coding`, `github.com/cachicamas/backend/agent/src/cmd`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters`, and `go.opentelemetry.io/contrib/bridges/otelslog`. **AI-37 adds no forbidden prefix and removes none; AG-03 adds none and removes none**; that table stays exactly as written. The OpenTelemetry **API** paths of ADR 0005 § D3 MUST NOT appear in the forbidden list.

A guard that has only ever been green is unproven. This guard closed on **bite proof**: it was shown to fail against three separate deliberate violations, with the failing output recorded, before it landed green. Any amendment to its allowlist MUST re-prove the same property — which at AI-37 is discharged by `R-AOB-002`'s two recorded bites. Any amendment to its **scanned set** MUST likewise re-prove that a genuine violation still bites — which at AG-03 is discharged by `agent-package-scaffold`'s `S-AGP-062`.

#### Scenarios

- **S-AGM-040** — Given the repository with no violation present, when `cd backend/agent && go test ./src/ai/...` runs, then the guard test passes.
- **S-AGM-041** — *(AMENDED 2026-08-12 by AG-03)* Given the guard's source, when its `go list` invocation and its scanned-root definition are read, then it passes both `-deps` and `-test`; its pattern or patterns are fully-qualified module paths, never relative; the resulting scanned set covers every Layer 1 package of the module — `src/ai/…`, `src/agenttest/…` and `src/handoff` — with none omitted; and, where the scanned set can contain the guard's own pattern members, the source states in place which mechanism prevents a member of that set from being reported as an import violation and why that mechanism was chosen. (Previously: *"then it passes both `-deps` and `-test`, and its pattern covers every package of the module."* That wording was true of `layer1Pattern = modulePath + "/..."` and is the wording AG-03 may falsify by narrowing the roots; the property it protected — complete Layer 1 coverage with no silent narrowing — is restated above in a form independent of the mechanism AG-03's design selects.)
- **S-AGM-042** — Given the guard's source, when its allowlist is read, then the standard-library group is derived from the toolchain rather than from a "path segment contains no dot" heuristic, AND the third-party group is present, non-empty, and annotated per entry with the ADR clause that authorises it and — for any entry admitted only as the closure of an authorised path — with that reasoning stated in place.
- **S-AGM-043** — Given the guard's source, when its forbidden-prefix list is read, then it names all eight prefixes above, unchanged from before this milestone, AND contains no OpenTelemetry API path (`go.opentelemetry.io/otel`, `…/otel/trace`, `…/otel/attribute`, `…/otel/codes`, `…/otel/metric`).
- **S-AGM-044** — **(bite 1)** Given a scratch file in `backend/agent/src/ai/` that imports `github.com/cachicamas/backend/database_administrator/src/domain`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. The failing output is recorded; the scratch file is then removed.
- **S-AGM-045** — **(bite 2)** Given a scratch file in `backend/agent/src/ai/` that imports `go.opentelemetry.io/otel/sdk/trace`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. Recorded, then removed. This bite MUST be re-proven at AI-37, because AI-37 is the milestone that makes the neighbouring API paths admissible and therefore the one at which an over-broad prefix would first go unnoticed.
- **S-AGM-046** — **(bite 3)** Given a scratch file in `backend/agent/src/ai/` that imports an arbitrary third-party module named by no forbidden prefix and by no allowlist entry, when `go test ./src/ai/...` runs, then the guard FAILS **on the deny-by-default allowlist rule**, not on a named prefix. This is what proves the allowlist is still deny-by-default after it stopped being empty. Recorded, then removed together with any require entry its presence forced into `go.mod`.
- **S-AGM-047** — Given the diff of any change that touches the guard, when it is inspected, then no scratch violation file appears in it AND `backend/agent/go.mod` declares no require entry that an ADR does not authorise.
- **S-AGM-048** — Given the guard's source, when its handling of `go list -deps -test` output is read, then it normalizes the synthesized entries the toolchain emits — the ` [<pkg>.test]` test-variant suffix and the `<pkg>.test` binary — so that neither is mistaken for a real import path.

### R-AGM-006 — Reverse guard: nothing reaches back

`backend/database_administrator` MUST carry an executable guard, in `src/domain/imports_test.go`, with two assertions:

1. The forbidden-prefix test over `src/domain` MUST name `github.com/cachicamas/backend/agent`. This assertion is **transitive** (`go list -deps`): `src/domain` MUST NOT reach the agent module by any path.
2. A module-scope assertion MUST verify that no package of `database_administrator` outside `src/application/` and `src/cmd/server` **names** the agent module. This assertion MUST scan **direct imports only** — `.Imports`, `.TestImports` and `.XTestImports` — and MUST NOT be transitive, because [ADR 0005 § D1 row 5](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) permits `src/application` and `src/cmd/server` to import the agent module, and a transitive scan would therefore indict every consumer of `application` the moment row 5 is exercised.

Both assertions were green from birth, because the count of such imports was zero. That is the point: the guard fails on the first accidental import rather than on the first production incident ([ADR 0005 § Guard B](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement)).

This guard also closed on **bite proof**: one deliberate violation, recorded failing, then removed.

**Evidence gate exception.** Unlike every other requirement in this spec, this requirement's gate is `make test` in `backend/database_administrator/`, not in `backend/agent/`. This is the single documented exception in doc 0002's global evidence gate.

#### Scenarios

- **S-AGM-050** — Given the repository, when `cd backend/database_administrator && go test ./src/domain/...` runs, then both reverse-guard assertions pass.
- **S-AGM-051** — Given the domain assertion's source, when its `go list` invocation is read, then it uses `-deps` over `github.com/cachicamas/backend/database_administrator/src/domain` and its forbidden set includes `github.com/cachicamas/backend/agent`.
- **S-AGM-052** — Given the module-scope assertion's source, when its `go list` invocation is read, then it requests `.Imports`, `.TestImports` and `.XTestImports` over the pattern `github.com/cachicamas/backend/database_administrator/...` AND does **not** pass `-deps`, AND a comment states that row 5 is the reason.
- **S-AGM-053** — Given the module-scope assertion's source, when its exemption set is read, then it exempts exactly the package prefixes `…/database_administrator/src/application` and `…/database_administrator/src/cmd/server`, and nothing else.
- **S-AGM-054** — Given the repository, when the module-scope assertion runs, then it inspects every package of the module (13 at AI-00's merge) and finds zero packages naming the agent module outside the exemption set.
- **S-AGM-055** — **(bite)** Given a scratch import of `github.com/cachicamas/backend/agent/src/ai` added to a file in `backend/database_administrator/src/domain/`, when `cd backend/database_administrator && go test ./src/domain/...` runs, then the guard FAILS and its message names both the offending package and the forbidden module path. The failing output is recorded; the scratch import is then removed.
- **S-AGM-056** — Given the diff of any change that touches the reverse guard, when it is inspected, then no scratch import appears in any `database_administrator` file AND `backend/database_administrator/go.mod` is unmodified.

### R-AGM-007 — Non-regression of the existing modules

Creating and maintaining `backend/agent` MUST NOT alter the behavior of `backend/database_administrator` or `backend/workspace_syncer`.

`backend/database_administrator/go.mod` MUST NOT gain a `replace` directive. [ADR 0005 § Migration](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#migration): a `replace` with no requirer is dead weight that also disguises the D1 row-5 build cost.

`backend/database_administrator/src/tools/tools.go` MUST remain byte-identical to its pre-AI-00 content. S3 is resolved by *vacating* the name `tools`, not by relocating a standard-idiom file.

A `make test` baseline MUST be captured for both existing modules **before** any file of a change like this lands, and compared afterwards. The comparison point differs per phase, and stating it precisely is what makes the requirement checkable: after the module skeleton and layout land, both modules MUST produce the same set of test names with the same outcomes as their baseline (timings excepted); after the reverse guard lands, `database_administrator`'s result set MUST differ from its baseline **only** by the added reverse-guard tests, and `workspace_syncer`'s MUST still match exactly.

#### Scenarios

- **S-AGM-060** — Given the pre-AI-00 tree, when `cd backend/database_administrator && make test` and `cd backend/workspace_syncer && make test` are run and their outputs saved, then both baselines are recorded as artifacts before any file is created. (Recorded: `database_administrator` 10 `ok` + 3 `[no test files]`; `workspace_syncer` 8 `ok`.)
- **S-AGM-061** — Given the tree after `go.work`, the new module and its packages exist but before the reverse guard is added, when `make test` is re-run in both existing modules, then each produces the same set of test names with the same pass/fail outcomes as its baseline.
- **S-AGM-062** — Given the repository, when `cd backend/workspace_syncer && make test` runs, then its result set matches the `S-AGM-060` baseline exactly.
- **S-AGM-063** — Given the repository, when `cd backend/database_administrator && make test` runs, then its result set equals the `S-AGM-060` baseline plus exactly the reverse-guard tests of `R-AGM-006`, all passing, and contains no removed or newly failing test.
- **S-AGM-064** — Given the repository, when `git diff` is taken across AI-00's range on `backend/database_administrator/src/tools/tools.go`, then it produces empty output. (Recorded: SHA-256 `79a1803a1c7e8e930d1ec34d0ead1b101004c4443ea462b96551fd27145f0cb6`, matching the pre-change hash.)
- **S-AGM-065** — Given the repository, when `backend/database_administrator/go.mod` is read, then it contains no `replace` directive.
- **S-AGM-066** — Given AI-00's merged diff, when the set of changed files under `backend/database_administrator/` is enumerated, then it contains exactly one entry: `src/domain/imports_test.go`.
- **S-AGM-067** — Given AI-00's merged diff, when the set of changed files under `backend/workspace_syncer/` is enumerated, then it is empty.

### R-AGM-008 — *(ADDED 2026-08-08 by AI-37)* The non-standard-library package closure is pinned by set equality, not by prefix subset

The guard MUST assert that the module's complete non-standard-library package closure, including test-only dependencies, **equals** an enumerated in-test set: this module's own packages plus the exact authorised third-party packages. Set **equality** is required; a subset check against the prefix allowlist MUST NOT be accepted as the proof.

The reason is mechanical and is why this is a requirement rather than a clause. The allowlist matcher is **prefix-based**, so a single ecosystem-root entry admits every package under that root. A version bump or a new import that dragged that ecosystem's metric, baggage, propagation or auto-instrumentation packages into the closure would be admitted **silently** by a prefix allowlist that never changed. Set equality is what keeps ADR 0005 § D3's "exactly the paths in the table above and nothing else" honest at *package* granularity, and it is what turns a closure-growing version bump into a build failure rather than a quiet drift.

The pin MUST distinguish a **module** requirement from a **package** import: a module may legitimately appear in the require set because a subpackage of it is imported, while the module's own root package is never imported. The require-set pin of `R-AGM-001` and the package-closure pin of this requirement are therefore separate assertions and MUST both hold.

#### Scenarios

- **S-AGM-069** — Given the repository, when the module's complete non-standard-library package closure including test-only dependencies is enumerated and compared against the enumerated in-test set, then the two sets are **equal** — neither contains a member the other lacks.
- **S-AGM-070** — Given a package that the prefix allowlist would admit but that the enumerated closure set does not name, when it enters the closure, then the pin FAILS and its message names that package — proving the pin is equality and not a prefix subset.
- **S-AGM-071** — Given the pinned closure, when it is inspected, then it contains no package whose path names an SDK, an exporter, a metric, baggage, propagation or auto-instrumentation package of the telemetry ecosystem.
- **S-AGM-072** — Given a module that appears in the require set because a subpackage of it is imported, when the closure is inspected, then that module's own root package is absent from it — the require-set pin and the closure pin disagree by design about that path, and both still pass.
- **S-AGM-073** — Given the guard's source, when this pin is read, then it states in place that it replaces a former zero-require assertion, cites the ADR clause authorising the requires it now enumerates, and explains why equality rather than a prefix subset is used.

---

## Non-functional requirements

### NFR-AGM-001 — Guard determinism

Both guards MUST be deterministic and hermetic: they shell out to the local `go` toolchain and read no network, no environment-specific path, and no file outside the repository. They MUST pass under `-race` (the evidence gate runs `go test -race -v ./...`) and MUST NOT depend on the working directory from which `make test` is invoked — package patterns are given as fully-qualified module paths rather than relative `./...` patterns resolved from an assumed cwd.

### NFR-AGM-002 — Guard legibility

Each guard's failure message MUST name the offending import path and the rule that rejected it (deny-by-default allowlist, or a specific forbidden prefix), so that a failure five milestones from now is actionable without reading the guard's source. Each guard MUST cite the ADR clause it enforces in a comment.

### NFR-AGM-003 — Review budget

Doc 0002 prefers under 250 changed lines and requires reassessment before 400. AI-00 shipped as one PR and exceeded that budget at roughly 520 changed lines. Per doc 0001 § 9 (Process), the PR description states why the change does not fit; the reasoning is in the archived `proposal.md` § "Review budget exception" and is restated in its `tasks.md`.

---

## Evidence discipline

`openspec/config.yaml` declares `apply.tdd: true`, and `openspec/AGENTS.md` states that a test written after the implementation does not satisfy this repository's discipline. Doc 0002 refines that for the node types in play here, and the refinement binds any later change to this contract as much as it bound AI-00:

- **Guard leaves (`R-AGM-005`, `R-AGM-006`)** are RED-first by construction. A guard that has only ever been green is unproven, so the closing condition is the recorded red run against a deliberate violation, followed by green. Four reds were required in total: three for the forward guard (`S-AGM-044`, `S-AGM-045`, `S-AGM-046`) and one for the reverse guard (`S-AGM-055`).
- **Mechanical leaves (`R-AGM-001` through `R-AGM-004`, `R-AGM-007`)** are exempt from red-green — there is no behavior to drive out — but are **never** exempt from their recorded Check lists. Each scenario above is a check whose recorded output (command output, file content, diff) is the evidence.

---

## Out of scope at AI-00, with the milestone that owns each

- Any dependency at all, including the OpenTelemetry API — AI-37 (landed 2026-08-08); and the transport — AI-24 (landed with zero dependency, stdlib `net/http` only). Each behind its own ADR gate.
- Any Layer 1 contract content: vocabulary (AI-01), stream lifecycle and carrier (AI-02), capability scope (AI-03), the first type (AI-04).
- `src/agent/` (Layer 2, doc 0003), `src/coding/` (Layer 3, doc 0004), `src/cmd/cachicamas/` (doc 0004) — not created, not stubbed.
- AI-20.4's signature guard. This contract guarantees only the layout it depends on.
- AI-25.2's ambient-authority scan over the adapter package.
- Exercising ADR 0005 § D1 row 5. A non-goal for all of v1; it breaks `docker compose build` and ADR 0005 prices the fix as its own change.
- Adding CI. `.github/workflows/` stays absent; ADR 0005 records this under Consequences.
- Amending ADR 0005's stale § Migration narrative. Doc 0002 records the staleness; amending the ADR is a separate change.

---

## Acceptance criteria

The contract holds when:

1. Every scenario `S-AGM-001` through `S-AGM-067` has recorded evidence.
2. `cd backend/agent && make test` and `make lint` are green.
3. `backend/agent/go.mod` declares no unauthorised `require`, and at AI-00 declared zero and had no `go.sum`.
4. All four guard bites are recorded red, and no scratch violation appears in any merged diff.
5. `cd backend/workspace_syncer && make test` matches its pre-AI-00 baseline; `cd backend/database_administrator && make test` differs from its baseline only by the added reverse-guard tests.
6. `backend/database_administrator/src/tools/tools.go` is byte-identical and its `go.mod` has no `replace`.
7. `go.work` lists all three modules and each builds independently from its own directory.
8. Any change that exceeds doc 0002's review budget states why in its PR description.

Criteria 1 through 8 were verified at AI-00's merge and recorded in the archived `verify-report.md`, which returned **PASS** on all three charter clauses with four guard bites recorded red.

**Amended 2026-08-08 (AI-37).** Scenarios `S-AGM-001`…`S-AGM-004`, `S-AGM-020`…`S-AGM-023`, `S-AGM-040`…`S-AGM-048`, `S-AGM-068`…`S-AGM-073` were re-verified or newly proven at AI-37's close and recorded in `openspec/changes/archive/2026-08-08-cachicamas-ai-observability/verify-report-final.md`: `agent-module-scaffold`'s scenarios all graded COMPLIANT except `S-AGM-020`, graded PARTIAL because the shipped `go.work` declares `go 1.26.5` while an earlier restatement of this scenario said `1.26.3` — corrected above to the true value at archive time. `S-AGM-001` (asserting `go.mod`'s own `go 1.26.3` directive, a distinct file) remains true and unaffected.
