# Spec — Agent package scaffold and boundary guards

> **Change**: `cachicamas-agent-package-scaffold` · **Milestone**: AG-03 (Layer 2, Wave 1) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-03--package-scaffold-and-boundary-guards)
> **Nodes**: AG-03.1 `[mechanical]` · AG-03.2 `[guard]` · AG-03.3 `[guard]`
> **Introduced by**: `openspec/changes/archive/2026-08-12-cachicamas-agent-package-scaffold/`, opened as a PR against `main` on 2026-08-12 (PR number and merge commit to be back-filled once merged, per this repo's convention on already-merged specs).
> **Status**: **live** — the invariants below hold for the lifetime of Layer 2, not only at the moment AG-03 merged
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml`. Every scenario is independently verifiable.
> **Identifier convention**: requirements `R-AGP-0NN`, scenarios `S-AGP-0NN`. Append-only.
> **Evidence gate**: `cd backend/agent && make test` (`go test -race -v ./...`), plus `make lint`. No CI exists.

## Purpose

Define the contract for `backend/agent/src/agent/` — Layer 2's package — at the moment it comes into existence: that it exists and declares nothing; that its layer contract is machine-checked rather than remembered; that its import boundary and its I/O boundary are executable guards proven to bite; and that creating it does not silently disarm Layer 1's shipped forward guard.

The package contains no behavior. Both guard requirements (`R-AGP-003`, `R-AGP-005`, `R-AGP-006`) close **only** on a recorded red run against a deliberate violation, then green. A guard that has only ever been green is unproven.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The module contract therefore lives here. The archived change folder at [`openspec/changes/archive/2026-08-12-cachicamas-agent-package-scaffold/`](../../changes/archive/2026-08-12-cachicamas-agent-package-scaffold/) is the historical record of how AG-03 was explored, proposed, designed, applied and verified — including the seven recorded guard bites and any doc 0003 amendments the implementation forced.

A later milestone that needs to change one of these invariants — for example AG-04 giving Layer 2 behavior, or a later node correcting a guard — amends **this file**, in the same pull request, under its own ADR gate.

> **Amended 2026-08-12 (AG-03 archive phase)** by the archive executor: `R-AGP-003`, `S-AGP-022`, and `S-AGP-023` were clarified to more explicitly describe the two-table mechanism the shipped guard implements — a forbidden-prefix table for non-standard-library denials matched before the allowlist, and a separate exact-path table for standard-library network/filesystem denial — and to scop the before-the-allowlist ordering to the vendor subtree alone. The W1 finding from verify-report.md was addressed prior to promotion.

> **Amended 2026-08-12 (AG-04, `cachicamas-agent-event-envelope`)** by the archive executor: `R-AGP-002` and its scenarios are restated in full below to record that the committed expectation table in `doc_contract_guard_test.go`'s `expectedLayer2ContractRows` grew from three rows (`L2C-01`..`L2C-03`, AG-03) to four rows (`L2C-01`..`L2C-04`, AG-04), adding the stream-membership criterion (`L2C-04`, design AD-3, landed in commit `e10831d3` in the same commit as its corresponding row in `doc.go`). No requirement text or scenario wording changed; the amendment records the committed table's expanded content. `R-AGP-002`'s scenarios were re-verified against the four-row baseline during AG-04's apply phase (see AG-04's `apply-progress.md`, Deviation #4).

## Requirements

### R-AGP-001 — The Layer 2 package exists and declares nothing

`backend/agent/src/agent/` MUST exist as a Go package named `agent`, carrying package documentation and **nothing else** — no type, no constant, no function, no variable. `backend/agent/src/coding/` and `backend/agent/src/cmd/` MUST NOT exist in any form, including as an empty directory or a directory holding only a placeholder. The change MUST NOT alter `backend/agent/go.mod`, `go.sum`, `Makefile`, or `.golangci.yml`.

#### Scenarios

- **S-AGP-001** — Given the repository after this change, when `backend/agent/src/agent/` is listed, then it exists, is a directory, and every `.go` file in it declares `package agent` or `package agent_test`.
- **S-AGP-002** — Given `backend/agent/src/agent/`, when its Go declarations are enumerated (for example `go doc github.com/cachicamas/backend/agent/src/agent`), then the package exports nothing and declares nothing beyond the package clause and its documentation comment.
- **S-AGP-003** — Given the repository after this change, when `test -e backend/agent/src/coding` and `test -e backend/agent/src/cmd` each run, then each exits non-zero.
- **S-AGP-004** — Given the repository after this change, when `cd backend/agent && make test` and `make lint` run, then each exits 0 with no failing test and no reported issue.
- **S-AGP-005** — Given the merged diff of this change, when `backend/agent/go.mod`, `go.sum`, `Makefile` and `.golangci.yml` are inspected, then each is byte-unchanged and the diff adds no `require` entry.

### R-AGP-002 — The layer contract is a machine-checked doc-row table

The package documentation MUST state the Layer 2 contract in machine-parseable rows covering, at minimum: the permitted-imports rule ([ADR 0005 § D1](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) row 2), the no-I/O-of-its-own rule, and the event stream as the only upward contract.

A guard in the same package MUST read `backend/agent/src/agent/doc.go`'s **raw bytes**, match the pinned row grammar with a line pattern, and compare the parsed rows **entry-by-entry and in order** against a **closed, committed expectation table declared in the test source**. The comparison MUST be equality over the full row set, not a subset or containment check: a row present in the file but absent from the table MUST fail, and a table entry absent from the file MUST fail.

This mechanism is a recorded **substitution** for AG-03.1's cited "doc-guard byte-suffix convention", which exists nowhere in doc 0002 or doc 0003 as a worked mechanism. The substituted precedent is `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go` (AI-40.2). The substitution MUST be stated in the guard's own source with its reason. Any later milestone that appends a guarded paragraph to this doc comment MUST amend the expectation table in the **same** pull request; the guard is what makes omitting that amendment a failing test rather than drift.

The requirement is stated over **whatever the committed table holds**; it never fixes a row count. The row baseline has grown `L2C-01`..`L2C-03` (AG-03) → `L2C-04` (AG-04) → `L2C-05` (AG-05) → `L2C-06` (AG-06) → `L2C-07` (AG-12, history's single validated commit path), each row landing in the same pull request as its expectation-table entry.

(Previously: identical normative text; `S-AGP-012` and `S-AGP-014` carried "as of AG-04 … four rows" parentheticals that no longer described the committed baseline, and the requirement did not state that the count is deliberately unfixed.)

#### Scenarios

- **S-AGP-010** — Given `backend/agent/src/agent/doc.go`, when its documentation comment is read, then it carries rows stating the permitted-imports rule citing ADR 0005 § D1, the no-I/O rule, and the event stream as the only upward contract.
- **S-AGP-011** — Given the doc-row guard's source, when its file access is read, then it resolves `doc.go` from the test's own location rather than an assumed working directory, reads its raw bytes, and parses rows with a pinned line pattern — not by importing the package or reading `go/doc` output.
- **S-AGP-012** — Given the repository with no divergence, when `cd backend/agent && go test ./src/agent/...` runs, then the doc-row guard passes and the parsed row set equals the committed table in the same order. *(As of AG-12: the committed table has seven rows, `L2C-01`..`L2C-07`; the scenario's own wording is unaffected.)*
- **S-AGP-013** — **(bite)** Given a scratch edit to `doc.go` that changes one contract row's text without amending the expectation table, when the doc-row guard runs, then it FAILS and its message names the divergent row and both the expected and the found text. The failing output is recorded; the scratch edit is then reverted.
- **S-AGP-014** — **(bite)** Given a scratch edit that appends a new contract row to `doc.go` without adding it to the expectation table, when the doc-row guard runs, then it FAILS naming the unexpected row — proving the comparison is closed and not a containment check. Recorded, then reverted. *(Re-proven at AG-12 against the seven-row baseline by `S-HIS-081`, which withholds `L2C-07`'s own table entry — the same shape of bite, three rows later.)*
- **S-AGP-015** — Given the doc-row guard's source, when its header comment is read, then it states that this mechanism substitutes for doc 0003's cited "byte-suffix convention", names `doc_matrix_guard_test.go` as the precedent it copies, and states that a later appended paragraph must amend the table in the same pull request.

### R-AGP-003 — Forward import guard: Layer 2 import purity over both closures

`backend/agent/src/agent/` MUST carry an executable guard that fails when any package of Layer 2 — including its test packages and their transitive dependencies — depends on anything outside an explicit allowlist.

The guard MUST enumerate dependencies via the Go toolchain's own dependency listing with test dependencies included, MUST derive the standard-library set from the toolchain's own classification rather than a path-shape heuristic, and MUST normalize the synthesized test-variant entries the toolchain emits so that neither a test-variant suffix nor a test-binary path is mistaken for a real import. The guard MUST fail loudly rather than pass vacuously when the listing returns zero packages.

The allowlist MUST be **deny-by-default**: a path that is neither standard library nor an explicitly admitted entry fails **even when no rule names it**.

The **production** closure MUST admit only: the standard library; `github.com/cachicamas/backend/agent/src/ai` and its transitive closure as freshly measured under `R-AGP-004`; and such observability API paths as [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) authorises for Layer 2 **and** design explicitly selects — no others. Every third-party entry MUST carry, in the guard's own source, the ADR clause authorising it; an entry admitted only as the forced closure of an authorised path MUST record that reasoning in place; and an entry that the fresh measurement does not show in the actual closure MUST record why an as-yet-unused path is admitted. The **test** closure MUST additionally admit `github.com/cachicamas/backend/agent/src/agenttest` and its subpackages, and MUST be asserted **separately** from the production closure.

The guard MUST maintain **two separate enforcement tables**. The first is a **forbidden-prefix table** for non-standard-library denials, matched **before** the allowlist, covering at minimum: the vendor adapter subtree `github.com/cachicamas/backend/agent/src/ai/openaicompat`; `…/agent/src/coding`; `…/agent/src/cmd`; `github.com/cachicamas/backend/database_administrator`; `github.com/cachicamas/backend/workspace_syncer`; `go.opentelemetry.io/otel/sdk`; `go.opentelemetry.io/otel/exporters`; `go.opentelemetry.io/contrib/bridges/otelslog`. The before-the-allowlist ordering of this table MUST be stated in source with its reason: the vendor subtree lives under an otherwise-admitted Layer 1 prefix, so an allowlist-first pass would admit it. This ordering applies only to the forbidden-prefix table.

Standard-library network/filesystem denial (`net`, `net/http`, `os`, `io/fs`) MUST NOT be attempted via the forbidden-prefix table — the dependency lister filters the standard library out before either the forbidden-prefix table or the allowlist runs, so a row there is unreachable dead code. The guard MUST instead carry a **separate, exact-path table** matched over a dependency listing that includes the standard library, covering exactly `net`, `net/http`, `os`, and `io/fs`. Each entry in this exact-path table MUST be annotated with the rule it enforces, mirroring `ai_test`'s `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` (AI-10.4).

This guard closes on **bite proof**, not on green.

#### Scenarios

- **S-AGP-020** — Given the repository with no violation present, when `cd backend/agent && go test ./src/agent/...` runs, then the forward guard passes.
- **S-AGP-021** — Given the guard's source, when its dependency listing is read, then it requests both transitive dependencies and test dependencies, uses a fully-qualified package pattern rather than a relative one, filters the standard library by the toolchain's own classification, normalizes the synthesized test-variant and test-binary entries, and fails the test when the listing returns zero packages.
- **S-AGP-022** — Given the guard's source, when the matching order and its scope are read, then the forbidden-prefix table is checked before the allowlist AND a comment states that this ordering applies only to the forbidden-prefix table because the vendor adapter subtree lives under an otherwise-admitted Layer 1 prefix, so an allowlist-first pass would admit it.
- **S-AGP-023** — Given the guard's source, when its two denial tables are read, then the forbidden-prefix table names every non-standard-library prefix listed in this requirement, each annotated with the rule it enforces, AND the separate exact-path table covers exactly `net`, `net/http`, `os`, and `io/fs`, matched over a dependency listing that includes the standard library, each entry annotated with the rule it enforces.
- **S-AGP-024** — Given the guard's source, when its allowlist is read, then every non-standard-library, non-own-module entry carries the ADR clause authorising it for Layer 2, no entry names an observability SDK, exporter or `otelslog` path, and any entry absent from the fresh `R-AGP-004` measurement carries its own recorded reason for being admitted before use.
- **S-AGP-025** — Given the guard, when the production closure and the test closure are asserted, then they are two separate assertions with separate allowlists, and the production assertion does not admit `…/src/agenttest` or its subpackages.
- **S-AGP-026** — Given a test file in `backend/agent/src/agent/` importing `…/src/agenttest` and no production file importing it, when the guard runs over both closures, then the production closure passes without admitting the substrate AND the test closure passes admitting it.
- **S-AGP-027** — **(bite 1)** Given a scratch file in `backend/agent/src/agent/` importing an application-layer path (`…/agent/src/coding/…`), when the forward guard runs, then it FAILS and its message names the violating import path and the rule. Recorded, then removed.
- **S-AGP-028** — **(bite 2)** Given a scratch file in `backend/agent/src/agent/` importing `net/http`, when the forward guard runs, then it FAILS and its message names the violating import path. Recorded, then removed.
- **S-AGP-029** — **(bite 3)** Given a scratch file in `backend/agent/src/agent/` importing the vendor adapter subtree `…/src/ai/openaicompat/…`, when the forward guard runs, then it FAILS naming the violating import path AND the recorded reason is the deny-by-name prefix rule, not a network side effect nor the deny-by-default rule.
- **S-AGP-030** — Given the merged diff of this change, when it is inspected, then no scratch violation file appears in it, `git status` reports a clean tree, and `backend/agent/go.mod` gained no `require` entry.

### R-AGP-004 — The Layer 1 closure is re-measured fresh, never inherited

The allowlist of `R-AGP-003` MUST be written against a **fresh** enumeration of `backend/agent/src/ai/...`'s production and test dependency closures, taken during this change and recorded with its date and exact commands as evidence. The recorded 2026-08-11 measurement in doc 0003's research digest MUST NOT be treated as still true; it is prose, guarded by nothing.

If the fresh measurement shows a path reaching the network (for example `net`, `net/http`, or any transport package) inside the **production** closure that the allowlist would admit, that divergence MUST be reported as a **blocking finding** to the change's driver before the allowlist is written. It MUST NOT be quietly absorbed by widening the allowlist, and it MUST NOT be recorded as an unremarkable transitive entry.

#### Scenarios

- **S-AGP-040** — Given this change's recorded evidence, when it is read, then it contains the exact dependency-listing commands run over `backend/agent/src/ai/...` for both the production and the test closure, their full output, and the date they were taken during this change.
- **S-AGP-041** — Given the recorded fresh measurement and the allowlist actually written, when they are compared, then every admitted Layer 1 closure entry appears in the measurement and the allowlist admits no measured entry it does not name.
- **S-AGP-042** — Given a fresh measurement that shows a network-reaching path in the production closure, when the change proceeds, then the finding is reported as blocking and recorded before the allowlist is written, and no allowlist entry is added that admits that path without an explicit recorded decision.
- **S-AGP-043** — Given the fresh measurement, when it is compared against doc 0003's recorded 2026-08-11 claim, then any divergence is stated explicitly in the change's evidence rather than reconciled silently.

### R-AGP-005 — No-ambient-authority guard: Layer 2 performs no I/O of its own

`backend/agent/src/agent/` MUST carry an executable guard that fails when any non-test source file of the package **calls** a function of an ambient-authority package. The forbidden set MUST cover at minimum `os`, `os/exec`, `syscall`, and `io/ioutil`.

The guard MUST be a **call-site** scan over the package's parsed syntax, not an import-closure scan. The reason MUST be stated in source: `os` is standard library, so a bare `os.Getenv` call inside Layer 2 passes `R-AGP-003` silently; and an import-closure scan cannot distinguish a narrow environment read from a legitimate transitive dependency.

The guard MUST resolve import declarations to their local identifiers, handling aliased and dot imports, MUST select only non-test `.go` files in the package directory, and MUST uniformly exclude its own source file. Its known limitation — no type information, so a local identifier shadowing a forbidden package name would false-positive — MUST be recorded in the guard's source together with the reason it is accepted: removing it would require a dependency no milestone authorises.

This guard closes on **bite proof**, not on green.

#### Scenarios

- **S-AGP-050** — Given the repository with no violation present, when `cd backend/agent && go test ./src/agent/...` runs, then the ambient-authority guard passes and reports having inspected at least one source file.
- **S-AGP-051** — Given the guard's source, when its mechanism is read, then it parses each candidate file's syntax and inspects call expressions whose selector base resolves to a forbidden package — rather than matching import paths or raw text.
- **S-AGP-052** — Given the guard's source, when its forbidden set is read, then it names `os`, `os/exec`, `syscall` and `io/ioutil`, and a comment states why an import-closure scan cannot express this rule.
- **S-AGP-053** — Given the guard's source, when its import resolution is read, then an aliased import and a dot import are both resolved to the forbidden package they name.
- **S-AGP-054** — Given the guard's source, when its file selection is read, then it selects only non-`_test.go` files in the package directory and excludes the guard's own file by a uniform rule rather than a hard-coded name check on one file.
- **S-AGP-055** — **(bite)** Given a scratch source file planted in a temporary directory containing a real ambient environment read, when the scan runs over that directory, then it reports at least one violation naming the file, the line, and the offending package. Recorded, then removed by the test's own cleanup.
- **S-AGP-056** — Given the guard's source, when its recorded limitation is read, then it states that the scan carries no type information, gives the shadowing false-positive example, and states that removing the limitation would require an unauthorised dependency.

### R-AGP-006 — Creating Layer 2 MUST NOT disarm Layer 1's forward guard

Creating `backend/agent/src/agent/` makes the new package appear in the scanned set of `backend/agent/src/ai/import_boundary_test.go`, matching its `…/agent/src/agent` forbidden-prefix row with **zero real import violations**. That hazard is recorded in that guard's own source (lines 82–92) and its fix belongs to this change.

This change MUST fix it so that `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` passes with `src/agent` present. The `…/agent/src/agent` forbidden-prefix row MUST NOT be deleted, weakened, or commented out. The fix MUST NOT reduce the guard's coverage of any Layer 1 package. The chosen mechanism and its reason MUST be recorded in the guard's own source, replacing the hazard note.

The fix MUST be proven not to have silenced the guard: after the fix, a genuine Layer 1 violation MUST still fail it.

#### Scenarios

- **S-AGP-060** — Given the repository after this change, when `cd backend/agent && go test ./src/ai/...` runs, then `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` passes.
- **S-AGP-061** — Given the Layer 1 guard's source after this change, when its forbidden-prefix table is read, then the `…/agent/src/agent` row is still present, still carries its ADR 0005 § D1 row 1 rule string, and the `…/src/coding` and `…/src/cmd` rows are unchanged.
- **S-AGP-062** — **(bite)** Given a scratch file in `backend/agent/src/ai/` that genuinely imports `github.com/cachicamas/backend/agent/src/agent`, when `cd backend/agent && go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path and the Layer-2 forbidden-prefix rule. Recorded, then removed. This is what distinguishes a fix from a silencing.
- **S-AGP-063** — Given the Layer 1 guard after this change, when the set of Layer 1 packages it inspects is determined, then it still covers every package of `src/ai/…`, `src/agenttest/…` and `src/handoff`, and the count of inspected non-standard-library dependencies is not zero.
- **S-AGP-064** — Given the repository after this change, when `TestLayer1_DependencySet_ExactRequiresAndClosure` runs, then it passes with its `wantGoModRequires` and `wantExternalClosure` tables byte-unchanged.
- **S-AGP-065** — Given the Layer 1 guard's source after this change, when the location of the former known-hazard note is read, then it states which mechanism was chosen, why, and that the forbidden-prefix row was preserved rather than deleted.
- **S-AGP-066** — Given the merged diff of this change, when the set of changed files under `backend/agent/src/ai/` is enumerated, then it contains exactly one entry, `import_boundary_test.go`, and that file's change adds no `require` and removes no forbidden-prefix row.

## Non-functional requirements

### NFR-AGP-001 — Guard determinism

Every guard added by this change MUST be deterministic and hermetic: it shells out to the local Go toolchain or reads files inside the repository, and reads no network and no environment-specific path. Each MUST pass under `-race` and MUST NOT depend on the working directory from which `make test` is invoked — package patterns are fully qualified and file paths are resolved from the test's own location.

### NFR-AGP-002 — Guard legibility

Each guard's failure message MUST name the offending path or row **and** the rule that rejected it (deny-by-default allowlist, a specific forbidden prefix, a call-site violation, or a doc-row divergence), so a failure five milestones later is actionable without reading the guard's source. Each guard MUST cite the ADR clause or milestone node it enforces in a comment.

### NFR-AGP-003 — Review budget

`openspec/config.yaml` forecasts a 400-line review budget. This change ships three guards plus a cross-cutting Layer 1 edit as a single pull request under a pre-authorised `size:exception` against a 1000-line budget; the pull request description MUST state why the change does not fit the default budget.

## Evidence discipline

`openspec/config.yaml` declares `apply.tdd: true`.

- **Guard leaves (`R-AGP-002`, `R-AGP-003`, `R-AGP-005`, `R-AGP-006`)** are RED-first by construction. The closing condition is the recorded red run against a deliberate violation, followed by green. Seven reds are required: two doc-row (`S-AGP-013`, `S-AGP-014`), three forward-guard (`S-AGP-027`, `S-AGP-028`, `S-AGP-029`), one ambient-authority (`S-AGP-055`), and one Layer 1 re-bite (`S-AGP-062`).
- **The mechanical leaf (`R-AGP-001`)** is exempt from red-green — there is no behavior to drive out — but is never exempt from its recorded Check list.

## Out of scope at AG-03, with the milestone that owns each

- Any event, loop, or harness behavior — no type, no constant, no function. AG-04 onward.
- Any dependency, including the observability API modules. No `go.mod` or `go.sum` edit.
- The reverse direction (nothing reaches into Layer 2) — owned by Layer 1's forward guard and the hexagon's guards.
- `src/coding/`, `src/cmd/` — doc 0004. Their absence is what keeps their forbidden-prefix rows testable.
- Amending doc 0003's "byte-suffix convention" wording — a separate documentation change under doc 0003's living-graph clause.
- Extending the ambient-authority scan with type information — no milestone authorises that dependency.
- CI. Every gate here runs when a human runs `make test` in `backend/agent/`.

## Acceptance criteria

The contract holds when:

1. Every scenario `S-AGP-001` through `S-AGP-066` has recorded evidence.
2. `cd backend/agent && make test` and `make lint` are green, recorded in the pull request.
3. `backend/agent/go.mod` and `go.sum` are byte-unchanged.
4. All seven guard bites are recorded red, and no scratch violation appears in the merged diff.
5. The fresh closure measurement of `R-AGP-004` is recorded, dated, and matches the allowlist actually written.
6. Layer 1's forward guard passes with its `…/src/agent` forbidden-prefix row still present and still able to bite.
7. The `agent-module-scaffold` delta amends `S-AGM-035` and `S-AGM-041` per that spec's promotion discipline.

**All acceptance criteria were verified at AG-03's archive and recorded in `openspec/changes/archive/2026-08-12-cachicamas-agent-package-scaffold/verify-report.md`: `agent-package-scaffold`'s all scenarios graded COMPLIANT except `S-AGP-022`, `S-AGP-023`, and `S-AGP-040`, `S-AGP-043`, graded PARTIAL (see verify-report.md for W2–W3 details and resolution). Zero CRITICAL findings; all WARNINGS were non-blocking or resolved at archive time per Final-State Authority. At AG-04's close, `R-AGP-002`'s scenarios were re-verified against the four-row baseline (commit `e10831d3`), and all bites reproduced as specified (see `sdd/cachicamas-agent-event-envelope/apply-progress.md`, Deviation #4).**
