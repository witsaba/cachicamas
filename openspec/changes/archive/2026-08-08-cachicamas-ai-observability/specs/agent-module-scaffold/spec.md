# Delta for `agent-module-scaffold` — AI-37.1 the dependency and its guard

> **Change**: `cachicamas-ai-observability` · **Milestone**: AI-37 (doc 0002:2188–2235) · **Node**: AI-37.1 · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/agent-module-scaffold/spec.md`](../../../../specs/agent-module-scaffold/spec.md): three `MODIFIED` requirements and one `ADDED` requirement
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-AGM-*` / `S-AGM-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 4 D-1, D-6 · § 7 risk 7.2 · [explore.md](../../explore.md) § 9

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `agent-module-scaffold` |
| **Type** | Delta — `MODIFIED` `R-AGM-001`, `R-AGM-003`, `R-AGM-005`; `ADDED` `R-AGM-008` |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical, active and archived alike: maxima `R-AGM-007` / `S-AGM-067`. `R-AGM-008` and `S-AGM-068` onward are free. |
| **New scenarios** | `S-AGM-068` (under `R-AGM-003`), `S-AGM-069` … `S-AGM-073` (under `R-AGM-008`) |
| **Pre-authorisation** | The canonical spec names this milestone as the one that amends it, twice — `agent-module-scaffold/spec.md:20` and `:26`. This is the documented path, not an exception. |

### Why this is a `MODIFIED` and not an `ADDED`

The module's dependency invariant is stated today as **zero** requires and **no** dependency-sum file. AI-37 makes both false. Restating the requirement is the only honest amendment: appending a second, contradictory requirement would leave the canonical file asserting two incompatible invariants and would let a future reader cite the older one. Each `MODIFIED` block below restates its **entire** requirement, including every unchanged scenario, so the archive step loses nothing.

### The invariant is narrowed, not dropped

"Zero requires" was never the point; **"no unauthorised dependency, provable mechanically"** was. Replacing a zero-count assertion with an exact, pinned require set plus a set-equality closure pin is strictly stronger than what it replaces, because the zero-count assertion could never have distinguished an authorised path from its unauthorised transitive closure. `R-AGM-008` carries that closure pin as its own requirement precisely because it is the load-bearing half: the allowlist matcher is prefix-based, so a bare ecosystem-root entry would silently admit that ecosystem's metric, baggage and propagation packages the moment a version bump pulled them in.

---

## MODIFIED Requirements

### R-AGM-001 — Module identity and its pinned dependency set

A Go module MUST exist at `backend/agent/` with module path `github.com/cachicamas/backend/agent` and Go version `1.26.3`.

Its `go.mod` MUST declare **exactly** the require entries that an ADR authorises and no others — at AI-37 that is the OpenTelemetry **API** modules pre-authorised by [ADR 0005 § D3](../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), together with the indirect requirement their own closure forces. The require set MUST be asserted by **set equality against an in-test table of module paths and versions**, including every indirect entry, not by a count and not by a subset check. A require that no ADR authorises MUST fail the suite.

The module MUST NOT declare a `replace` directive.

A dependency-sum file MUST now exist in the module and MUST be committed. (Previously: the `go.mod` was required to declare zero requires and no dependency-sum file was permitted to exist; AI-37 is the milestone the canonical spec already named as the one that changes this, at `agent-module-scaffold/spec.md:20, 26`.)

#### Scenarios

- **S-AGM-001** — Given the repository, when `backend/agent/go.mod` is read, then it contains exactly one `module` directive whose value is `github.com/cachicamas/backend/agent` AND exactly one `go` directive whose value is `1.26.3` AND a `require` set **equal** to the ADR-authorised table of module paths and versions including its indirect entries AND zero `replace` directives.
- **S-AGM-002** — Given the repository, when the module's dependency-sum file is looked for, then it exists and is tracked by version control, and no ignore rule excludes it.
- **S-AGM-003** — Given the repository with the module cache already populated, when `cd backend/agent && go build ./...` runs, then it exits 0 and neither adds, removes nor rewrites any `require` entry.
- **S-AGM-004** — Given the repository, when every non-standard-library package the module depends on, including test-only dependencies, is enumerated, then each is either a package of this module or a member of the ADR-authorised closure pinned by `R-AGM-008`. (This is `R-AGM-005`'s allowlist stated as a one-shot structural check; the guard is what keeps it true.)

### R-AGM-003 — Repository workspace file

A `go.work` file MUST exist at the repository root and MUST list all three backend modules: `./backend/agent`, `./backend/database_administrator`, `./backend/workspace_syncer`. Each module MUST continue to build independently from its own directory. If the toolchain generates a `go.work.sum`, it MUST be committed. The root `.gitignore` MUST NOT exclude `go.work` or `go.work.sum`.

Because the workspace unifies the build list across all three modules, and a sibling module already pins a **different** version of an ecosystem this module now requires, the version resolved into `backend/agent/go.mod` and the version the workspace reports MAY differ. That divergence MUST fail the build rather than resolve silently: `R-AGM-001` pins the module's own require set and `R-AGM-008` pins the package closure **separately**, and both MUST hold. (Previously: the `go.work.sum` clause existed but had never been exercised, because no module in the workspace-scoped agent build carried a dependency.)

#### Scenarios

- **S-AGM-020** — Given the repository, when `go.work` is read, then its `use` block names exactly the three module directories above, and it declares `go 1.26.5`. (`go.work` is byte-unchanged by AI-37; this restates a prior milestone's version, re-verified by direct read at this round rather than asserted by any test — no guard in this module reads the `go.work` directive.)
- **S-AGM-021** — Given the repository, when `go build ./...` runs from each of the three module directories in turn, then each exits 0.
- **S-AGM-022** — Given a `go.work.sum` generated by the toolchain, when the diff of the change that generated it is inspected, then that file is present in the diff.
- **S-AGM-023** — Given the root `.gitignore`, when it is read, then it contains no pattern matching `go.work` or `go.work.sum`, AND the working tree after a full test run in all three modules reports no untracked workspace-sum file.
- **S-AGM-068** — Given this change, which is the first to make the workspace carry a dependency for this module, when the merged diff is inspected, then the generated workspace-sum file appears in it, the working tree is clean afterwards, and the version recorded in the module's own require set is asserted independently of whatever the workspace build list reports — a divergence between the two fails the suite rather than passing quietly.

### R-AGM-005 — Forward guard: Layer 1 purity

The module MUST carry an executable guard at `backend/agent/src/ai/import_boundary_test.go` that fails when any package of `backend/agent` — including its test packages and its transitive dependencies — depends on anything outside an explicit allowlist.

The guard MUST use `go list -deps -test` over the module's own package pattern. Bare `go list -deps` is insufficient: it does not report test-only imports, and a direct-imports listing reports neither test imports nor transitive dependencies ([ADR 0005 § Guard A](../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement), finding S6).

The allowlist MUST be **deny-by-default**: an import path that is neither a member of the Go standard library set nor a package of this module fails the guard *even when no rule names it*. The allowlist MUST be expressed as three groups — the standard library, this module's own packages, and a third-party group whose only permitted growth path is a milestone with its own ADR.

**The third-party group is no longer empty.** At AI-37 it admits exactly the OpenTelemetry **API** paths this module actually imports, plus the packages transitively required in order to import them at all. Every entry MUST carry, in the guard's own source, the ADR clause authorising it — ADR 0005 § D3 for each OpenTelemetry path. An entry that is not an OpenTelemetry module and is admitted only because an authorised path cannot be imported without it MUST record that reasoning in place: authorising an import path necessarily authorises its closure, and § D3's "any additional OTel module requires its own ADR" clause is not engaged by a non-OpenTelemetry transitive requirement. That reasoning is bounded by `R-AGM-008`'s set-equality pin, which is what stops it becoming a blank cheque. (Previously: the third-party group was empty and was described as empty.)

In addition to the allowlist, the guard MUST carry a named forbidden-prefix list covering: `github.com/cachicamas/backend/database_administrator`, `github.com/cachicamas/backend/workspace_syncer`, `github.com/cachicamas/backend/agent/src/agent`, `github.com/cachicamas/backend/agent/src/coding`, `github.com/cachicamas/backend/agent/src/cmd`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters`, and `go.opentelemetry.io/contrib/bridges/otelslog`. **AI-37 adds no forbidden prefix and removes none**; that table stays exactly as written. The OpenTelemetry **API** paths of ADR 0005 § D3 MUST NOT appear in the forbidden list.

A guard that has only ever been green is unproven. This guard closed on **bite proof**: it was shown to fail against three separate deliberate violations, with the failing output recorded, before it landed green. Any amendment to its allowlist MUST re-prove the same property — which at AI-37 is discharged by `R-AOB-002`'s two recorded bites.

#### Scenarios

- **S-AGM-040** — Given the repository with no violation present, when `cd backend/agent && go test ./src/ai/...` runs, then the guard test passes.
- **S-AGM-041** — Given the guard's source, when its `go list` invocation is read, then it passes both `-deps` and `-test`, and its pattern covers every package of the module.
- **S-AGM-042** — Given the guard's source, when its allowlist is read, then the standard-library group is derived from the toolchain rather than from a "path segment contains no dot" heuristic, AND the third-party group is present, non-empty, and annotated per entry with the ADR clause that authorises it and — for any entry admitted only as the closure of an authorised path — with that reasoning stated in place.
- **S-AGM-043** — Given the guard's source, when its forbidden-prefix list is read, then it names all eight prefixes above, unchanged from before this milestone, AND contains no OpenTelemetry API path (`go.opentelemetry.io/otel`, `…/otel/trace`, `…/otel/attribute`, `…/otel/codes`, `…/otel/metric`).
- **S-AGM-044** — **(bite 1)** Given a scratch file in `backend/agent/src/ai/` that imports `github.com/cachicamas/backend/database_administrator/src/domain`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. The failing output is recorded; the scratch file is then removed.
- **S-AGM-045** — **(bite 2)** Given a scratch file in `backend/agent/src/ai/` that imports `go.opentelemetry.io/otel/sdk/trace`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. Recorded, then removed. This bite MUST be re-proven at AI-37, because AI-37 is the milestone that makes the neighbouring API paths admissible and therefore the one at which an over-broad prefix would first go unnoticed.
- **S-AGM-046** — **(bite 3)** Given a scratch file in `backend/agent/src/ai/` that imports an arbitrary third-party module named by no forbidden prefix and by no allowlist entry, when `go test ./src/ai/...` runs, then the guard FAILS **on the deny-by-default allowlist rule**, not on a named prefix. This is what proves the allowlist is still deny-by-default after it stopped being empty. Recorded, then removed together with any require entry its presence forced into `go.mod`.
- **S-AGM-047** — Given the diff of any change that touches the guard, when it is inspected, then no scratch violation file appears in it AND `backend/agent/go.mod` declares no require entry that an ADR does not authorise.
- **S-AGM-048** — Given the guard's source, when its handling of `go list -deps -test` output is read, then it normalizes the synthesized entries the toolchain emits — the ` [<pkg>.test]` test-variant suffix and the `<pkg>.test` binary — so that neither is mistaken for a real import path.

---

## ADDED Requirements

### R-AGM-008 — The non-standard-library package closure is pinned by set equality, not by prefix subset

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

## Pins / regressions

| Behaviour leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| Dependency set is exactly what an ADR authorised | ADR 0005 § D3 table | Set equality over require paths **and versions**, incl. indirect |
| No `replace` | Landed `S-AGM-001` clause | Unchanged assertion |
| Closure cannot grow silently under a prefix | — | Set equality over the package closure |
| Module require ≠ package import | — | Root package absent from the closure while its module is required |
| Guard still denies by default | Landed bite 3 | Re-proven against an arbitrary unauthorised module |
| Workspace divergence surfaces | `R-AGM-003` | Require set asserted independently of the workspace build list |

## Out of scope

| Item | Owner |
| --- | --- |
| Which telemetry paths Layer 1 chooses to import, and why the global getter is declined | `ai-observability-boundary` — `R-AOB-001` |
| The two recorded bites for the telemetry boundary | `ai-observability-boundary` — `R-AOB-002` |
| The measured versions, the exact require literals and the enumerated closure members | **Design phase** — re-measured under the workspace after the dependency lands; this spec pins the *shape* of the assertion, not its literals |
| Any transport dependency | **AI-24**, behind its own ADR gate |
| Adding continuous integration | Non-goal — unchanged from AI-00 |
