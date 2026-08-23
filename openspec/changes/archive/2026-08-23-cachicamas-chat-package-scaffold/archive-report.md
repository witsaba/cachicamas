# Archive report — `cachicamas-chat-package-scaffold` (CH-01)

> **Milestone**: CH-01 of [doc 0005](../../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md) · Wave 0 · **2 of 12** shipped
> **Closes**: R-06 (the boundary half), R-07 · **Blocks released**: CH-02, CH-03
> **Branch**: `feat/chat-archetype-wave0-ch01`, based on `origin/main` @ `419a4291`
> **Archived**: 2026-08-23

## What shipped

`backend/agent/src/chat/` (`package chat`, documentation and zero declarations) and `backend/agent/src/cmd/chat/` (`package main`, a documented empty `func main()`) — this module's first composition root of any kind. The module's single deny-by-default import guard gained **check 6**: a recursive, per-file `parser.ImportsOnly` scan over the archetype tree, evaluating a forbidden table, then the toolchain's own `go list std` answer, then the allowlist, then denying by default.

The guard was **extended, never cloned** (`agent-layer3-handoff` R-L3H-002). Four hunks in `import_boundary_test.go`: two in the package comment, two wholly-new declaration blocks after check 5. No hunk falls inside an existing check, table or shared helper, and neither new tree carries a `_test.go` file.

## Promotions

| Capability | Kind | Net | Identifier set |
| --- | --- | --- | --- |
| `chat-package-boundary` | **new** — `R-CPB-001`…`010`, `NFR-CPB-001`…`003`, 53 scenarios | +319 | promoted whole |
| `agent-package-scaffold` | modified — `R-AGP-001`, `S-AGP-003` | +8 / −2 | unchanged |
| `chat-archetype-contract` | modified — `R-CHT-007`, `S-CHT-061`, `S-CHT-062`, `R-CHT-013`, 2 prose sites | +16 / −7 | unchanged |
| `ai-observability-boundary` | modified — one out-of-scope row | +3 / −1 | unchanged |

Each promotion was written by a separate writer owning exactly one file, then audited by the orchestrator with an identifier-set diff against the pre-promotion tree. All four sets came back identical: nothing minted, nothing renumbered, nothing truncated.

## The untemporal-invariant sweep

Creating two packages falsified existence claims in promoted specs. A tree-wide grep of `openspec/specs/` found **four** sites; all four are recorded and three are repaired.

| Site | Falsified by | Disposition |
| --- | --- | --- |
| `agent-package-scaffold` R-AGP-001 / S-AGP-003 | this merge | repaired |
| `chat-archetype-contract` R-CHT-007 and five siblings | this merge | repaired |
| `ai-observability-boundary`'s out-of-scope row | this merge (2 of its 3 clauses) | repaired; the Layer 2 clause was already false since **AG-03** and is recorded as AG-03's debt, not absorbed |
| `agent-contract-vocabulary` NFR-AGV-C | **AG-04**, not this merge | recorded, deliberately not repaired |

Site 2 was found *after* apply landed — the same defect shape as site 1, one capability away, which every planning phase read past including the one that wrote site 1's repair. Sites 3 and 4 came from a deliberate tree-wide sweep rather than from any phase's own reading.

## Evidence

- `cd backend/agent && go clean -testcache && go test -race -count=1 ./...` — green, **uncached**, wall clock 2:52.95 / 2:54.76 across independent runs against this module's known ~170s baseline. A sub-second result would have been a cache hit and is not evidence.
- `make lint` clean module-wide; `gofmt -l` empty over this change's three Go files. **`make all` was never run** — its fmt step rewrites committed files.
- `git diff 419a4291` empty for `backend/agent/go.mod`, `go.sum`, `Makefile` and `.golangci.yml`. Nine shipped Layer 2 assertions require it.
- Seven bites recorded with full failing output in `apply-progress.md` and independently reproduced at verify: the forbidden module, the defeat test, `openaicompat`'s forbidden-first ordering, the OTel SDK with its passing negative control at the root, the vacuity floor, and an `os/exec` re-plant reddening the pre-existing checks 3 and 4. Every scratch file removed before merge.

## Decisions the maintainer made, with what forced each

1. **The OTel-SDK bite is a source-level scan, not a closure check** — nine assertions require the `go.mod`/`go.sum` merge-base diff to be empty, so no dependency could be added.
2. **`R-AGP-001` is repaired by a delta in this PR** — a promoted spec is never left false on `main`.
3. **ADR 0005 is not amended** — § D1 row 3 and § D3's `coding` column are read as any Layer 3 archetype under ADR 0009 § D2, the substitution CH-00 promoted as R-CHT-011.
4. **"Layer 2 internals" means the test substrate plus the vendor adapter** — `src/agent` and `src/ai` stay permitted; `src/layer3handoff` stays admissible to the archetype's test closure per R-10.
5. **`otelslog` is permitted** — correcting a denial list that had been copied from Layer 2's rows. ADR 0005 § D3 grants the bridge to Layer 3; doc 0005 forbids only the SDK.
6. **The full allowlist ships as a boundary declaration**, with R-AGP-003's same-commit rule recorded as governing incremental amendment of an existing boundary rather than the declaration of a new one.
7. **`R-CHT-007` gets a second delta**, and **`ai-observability-boundary` a third**, applying decision 2's principle to defects found later.

## Corrections made during the cycle

Three premises that later evidence falsified, each corrected on the record rather than quietly:

- **`go.work` resolution.** Both planted scratch imports build — a sibling module resolves without a `require`, and `otel/sdk` is reachable through `database_administrator`'s requirement. The design's "unresolvable import" premise was false and the spec had already frozen it into a scenario; `S-CPB-091` now forbids any evidence claiming otherwise.
- **`S-CPB-094`** asserted an unscoped `gofmt -l` prints nothing. It prints 15 pre-existing files. The operand is now pinned — this capability had written the very defect shape it exists to repair.
- **The composition root's comment** never recorded the Go language rule behind its own unimportability, which `S-CPB-010` requires. A clause had fallen between two tasks, each assuming the other carried it; both closed green.

## Known limitations, each with its owner

- **Zero-hop only.** Check 6 is a source scan; a transitive violation reached through an admitted path is uncaught. Deferred to **CH-02** for a mechanical reason, not budget: a `doc.go`-only tree returns zero non-stdlib dependencies and trips the repo's pinned per-pattern vacuity floor. Bounded, because the only admitted first-party paths are `src/agent` and `src/ai`, whose own production closures check 1 already proves.
- **`agent-contract-vocabulary` NFR-AGV-C** remains false. Stale since AG-04, unowned.
- **15 files carry pre-existing `gofmt` drift** from a toolchain version bump. Byte-unchanged against the merge base, forbidden to repair here by `R-CPB-009`, and owned by nobody.
- **`tasks.md`'s coverage table maps scenarios as ranges**, so four scenarios are named by no individual task. All four are compliant under re-execution, but the audit that cleared them could not have failed — expanding a range manufactures the references the set difference then finds. The convention belongs to the task-graph method, not to this change.
