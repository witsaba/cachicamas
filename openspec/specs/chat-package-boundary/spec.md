# Spec — Chat archetype package boundary

> **Change**: `cachicamas-chat-package-scaffold` · **Milestone**: CH-01 (Wave 0) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md) (`0005:230-276`)
> **Nodes**: CH-01.1 `[mechanical]` (`0005:243-247`) · CH-01.2 `[guard]` (`0005:249-276`)
> **Closes**: R-06 (the boundary half), R-07 (`0005:232`)
> **Status**: **new capability**, promoted at archive to `openspec/specs/chat-package-boundary/spec.md`
> **Amended 2026-08-23 (CH-01 archive)**: promoted verbatim as a new capability at archive. Requirements `R-CPB-001` … `R-CPB-010`, non-functional `NFR-CPB-001` … `NFR-CPB-003` and scenarios `S-CPB-001` … `S-CPB-096` carried unchanged from `openspec/changes/cachicamas-chat-package-scaffold/specs/chat-package-boundary/spec.md`. Relative links were re-based from the change directory's depth to this one; no normative text changed.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml`. Every scenario is independently verifiable.
> **Identifier convention**: requirements `R-CPB-0NN`, scenarios `S-CPB-0NN`. Append-only. The `CPB` prefix was verified free across the whole worktree during this phase — the only occurrences outside this file are the proposal's own prefix declaration and this change's `agent-package-scaffold` delta.
> **Evidence gate**: `cd backend/agent && go test -race -count=1 ./...`, uncached, wall-clock recorded, plus `make lint`. `make all` MUST NOT be run.
> **Citation notation**: `0005:NNN` cites the milestone document; `adr:NNN` cites `docs/adr/0005-promote-agent-stack-to-own-module.md`; a bare `import_boundary_test.go:NNN` cites `backend/agent/src/agent/import_boundary_test.go` at the merge-base tree `419a4291` — the guard's new check is inserted mid-file, so verify MUST re-resolve every such line rather than trust it.

## Purpose

Define the contract for the chat archetype's two packages — `backend/agent/src/chat/` and its composition root `backend/agent/src/cmd/chat/` — at the moment they come into existence, and for the import boundary that makes the Layer 3 position's forbidden closure a failing test rather than a paragraph.

Until CH-01 merges, every architectural claim about where this archetype may reach is unfalsifiable: nothing on disk can violate a rule, so no rule is proven (`chat-archetype-contract` R-CHT-013, `spec.md:226` — *"The earliest honest mechanical binding is CH-01.2's import guard, not here."*).

The packages carry no behaviour. The guard requirements (`R-CPB-003` through `R-CPB-008`, and `R-CPB-010`) close **only** on a recorded red run against a planted violation, then green. A guard that has only ever been green is unproven.

## Requirements

### R-CPB-001 — The archetype package and its composition root exist and declare nothing

`backend/agent/src/chat/` MUST exist as a Go package named `chat`, carrying package documentation and **nothing else** — no type, no constant, no function, no variable.

`backend/agent/src/cmd/chat/` MUST exist as a Go package named `main`, carrying package documentation and a `func main()` whose body is empty. `func main()` MUST NOT be omitted: a `package main` without it does not build. It MUST NOT acquire behaviour at CH-01 — no environment read, no output, no non-zero exit — because `0005:240` puts behaviour out of scope; the honesty a fail-fast stub would carry is carried by the doc comment instead.

Both package identifiers are **derived, not chosen**. Every package under `backend/agent/src/` is named after its directory's last element — `src/agent` → `package agent`, `src/ai` → `package ai`, `src/agenttest` → `package agenttest`, `src/apptest` → `package apptest`, `src/handoff` → `package handoff`, `src/layer3handoff` → `package layer3handoff`, `src/ai/openaicompat` → `package openaicompat`, `src/ai/openaicompat/openrouter` → `package openrouter`, `src/ai/internal/retry` → `package retry`. No exception exists in this module. `package main` for the composition root is forced by the Go toolchain, not by convention.

The file shape follows the same derivation: every package under `backend/agent/src/` carries a `doc.go`, so the archetype package ships `doc.go` and nothing else, following `agent-package-scaffold` R-AGP-001's own AG-03 precedent for `src/agent` — documentation, no declarations.

#### Scenarios

- **S-CPB-001** — Given the repository after this change, when `backend/agent/src/chat/` is listed, then it exists, is a directory, and every `.go` file in it declares `package chat`.
- **S-CPB-002** — Given `backend/agent/src/chat/`, when its Go declarations are enumerated (for example `go doc github.com/cachicamas/backend/agent/src/chat`), then the package exports nothing and declares nothing beyond the package clause and its documentation comment.
- **S-CPB-003** — Given the repository after this change, when `cd backend/agent && go build ./src/cmd/chat` runs, then it exits 0; and when `backend/agent/src/cmd/chat/main.go` is read, then it declares `package main` and a `func main()` whose body contains no statement.
- **S-CPB-004** — Given `backend/agent/src/chat/doc.go`, when its comment is read, then it states that this archetype occupies the Layer 3 **position** (ADR 0009 § D2), names CH-01.1 of doc 0005 as its creator, states that it ships empty of policy until CH-02, enumerates the position's forbidden closure **and its permitted OpenTelemetry surface** — the API packages and the `otelslog` bridge (ADR 0005 § D3, `adr:243`, under the ADR 0009 § D2 substitution) — and names the guard in `backend/agent/src/agent/import_boundary_test.go` as the deny-by-default enforcement.
- **S-CPB-005** — Given `backend/agent/src/cmd/chat/main.go`, when its comment is read, then it states that this is the archetype's composition root and the only package of the archetype permitted the OTel **SDK and exporters** — ADR 0005 § D3's `cmd/` column (`adr:242`), which is generic and needs no substitution — that it is deliberately outside the guard's scanned set, and that **CH-04.1** owns its wiring, the empty `main` being a recorded decision rather than an oversight. Where the comment invokes the substitution convention it writes **ADR 0009 § D2** in full, because a bare "§ D2" following an ADR 0005 citation would read as ADR 0005's own § D2.
- **S-CPB-006** — Given the package identifiers actually written, when they are compared against the module's directory-basename convention, then each matches its directory's last element, and the derivation — not a preference — is what the spec and the source record.

### R-CPB-002 — Nothing imports the composition root, and that property is structural

Nothing in this module MUST import `github.com/cachicamas/backend/agent/src/cmd/chat` (`0005:62`, R-06: *"exactly one composition root, imported by nothing"*).

This property is enforced by the **Go language**, not by a guard: a `package main` is not importable, and an attempt is rejected by the toolchain at build time. This change MUST NOT ship a Go test asserting it. A test that cannot fail proves nothing and, worse, invites a later reader to believe the property is guarded when the guarantee actually lives in the compiler.

Independently, the archetype's forbidden table denies the `…/src/cmd` prefix (`R-CPB-004`), so a *source-level* import of any package below the composition root is denied by name before the language rule is ever consulted.

#### Scenarios

- **S-CPB-010** — Given the merged diff of this change, when it is searched for a test whose subject is "nothing imports `…/src/cmd/chat`", then none exists, and the property's mechanism — the Go language rule — is recorded in `backend/agent/src/cmd/chat/main.go`'s own comment.
- **S-CPB-011** — Given a scratch file anywhere in this module importing `github.com/cachicamas/backend/agent/src/cmd/chat`, when `go build ./...` runs, then the **toolchain** rejects it because a main package is not importable. This demonstration MAY be recorded once at apply; it MUST NOT be committed as a test.
- **S-CPB-012** — Given the archetype's forbidden table, when it is read, then it carries a `…/src/cmd` prefix row, so a source-level import of any package below the composition root fails by name rather than by default.

### R-CPB-003 — The boundary is enforced by ONE guard, extended and never cloned

The archetype's import boundary MUST be enforced by **the** module's existing deny-by-default import guard at `backend/agent/src/agent/import_boundary_test.go`, extended in place with one additional check. It MUST NOT be enforced by a second, self-contained guard living inside `backend/agent/src/chat/` or `backend/agent/src/cmd/chat/`.

This is `0005:240`'s literal instruction and `agent-layer3-handoff` R-L3H-002's requirement (`spec.md:75-84`): the zero-vendor-import property is established by *the* guard mechanism *"rather than by human inspection or by a second, duplicated allowlist living inside either new tree"*. `S-L3H-013` (`spec.md:95`) is the assertion shape this requirement reuses for the archetype's own trees.

Extending that file is permitted and is this repo's demonstrated convention: it is not byte-frozen, and its own package comment already records three prior in-place extensions (AG-03 → AG-22 → AG-23). The extension MUST update that package comment in place, appending its own amendment row.

At CH-01 the archetype ships **no** test file. The merged diff MUST therefore add zero `_test.go` files under either new tree — the second clause of CH-01.2's third scenario (`0005:272`).

#### Scenarios

- **S-CPB-020** — Given the merged tree, when `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` is inspected, then no listed file is an import guard, and the one guard at `backend/agent/src/agent/import_boundary_test.go` is what was extended.
- **S-CPB-021** — Given the merged tree, when that same listing is filtered for a `_test.go` suffix, then it is empty.
- **S-CPB-022** — Given the merged diff, when the changed files under `backend/agent/src/agent/` are enumerated, then `import_boundary_test.go` is the only guard file among them.
- **S-CPB-023** — Given `import_boundary_test.go`'s package comment after this change, when it is read, then its check enumeration includes the archetype check with its CH-01 attribution, and an appended amendment section records the new check's scope **and its recursion**, its citation rule, the composition root's exclusion from the scanned set, the deliberate absence described in `R-CPB-005`, the `otelslog` grant with its `adr:243` citation under the ADR 0009 § D2 substitution, **the whole-table deviation from `R-AGP-003`'s same-commit rule with its declaration-versus-incremental reasoning**, and the zero-hop gap with its owning milestone.

### R-CPB-004 — The archetype's forbidden closure, as ADR 0005 read under the ADR 0009 substitution

The archetype's **production** closure MUST deny, by name, each of `…/src/ai/openaicompat`, `…/src/agenttest`, `…/src/apptest`, `…/src/layer3handoff`, and `…/src/cmd`; and MUST permit `…/src/agent`, `…/src/ai`, the Go standard library, and such OpenTelemetry paths as [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) marks ✅ for this position. Anything else MUST fail **by default**, with no rule naming it.

**The OpenTelemetry grant is read off § D3's own table, never copied from Layer 2's forbidden rows.** § D3's `L3 coding` column, read as any Layer 3 archetype under ADR 0009 § D2, is what governs:

| § D3 path | At the archetype | Authority |
|---|---|---|
| `go.opentelemetry.io/otel/trace`, `/attribute`, `/codes` and the forced closure of `otel/trace` | **permitted** — allowlist entry | `adr:240-241` under ADR 0009 § D2 |
| `go.opentelemetry.io/contrib/bridges/otelslog` | **permitted** — allowlist entry | `adr:243` (❌ at L1/L2, ✅ at L3 `coding` and `cmd/`) under ADR 0009 § D2 |
| `go.opentelemetry.io/otel/sdk/…`, `…/exporters/…` | **denied**, by default rather than by name | `adr:242` (❌ at L3, ✅ only at `cmd/`) |

The `otelslog` entry MUST NOT be denied. Layer 2's own table denies it (`import_boundary_test.go:208`) because § D3 marks it ❌ at L2 — but Layer 2 is not Layer 3, and `0005:241`'s forbidden closure names only *"the OTel SDK"*, never the bridge. Its allowlist entry MUST carry `adr:243` plus the ADR 0009 § D2 substitution in place, so a later reader meets the grant as a decision rather than "correcting" it back to denied.

**The allowlist ships complete at CH-01, and the deviation this records covers every entry — not the bridge alone.** `agent-package-scaffold` R-AGP-003 (`spec.md:83`) requires that *"The allowlist amendment MUST land in the **same commit** as the first production import that requires it. A commit that adds the import without the entry, or the entry without the import, is a violation of this requirement."* At CH-01 the archetype imports nothing at all, so **every** entry in its allowlist is an entry without an import — `…/src/agent` and `…/src/ai` exactly as much as `otelslog`. Recording the deviation for the bridge alone would misstate its scope.

The full table MUST nevertheless ship now, and the deviation MUST be recorded **over the whole table**, with its reasoning stated rather than left for a reviewer to infer: R-AGP-003's same-commit rule governs an **incremental** allowlist — an existing boundary admitting one new dependency, which is the shape `S-AGP-034` and `S-AGP-069` each proved with a bite. The archetype's table is a different object: a **boundary declaration** for a new position, stated once so that the position's permitted surface is legible before any code exercises it. Both readings are defensible; this spec adopts the declaration reading deliberately.

That same statement MUST appear in the guard's own in-place comment, so a reader of `import_boundary_test.go` five milestones from now meets it as a recorded decision rather than as a table nobody justified.

Two alternatives were considered and rejected, recorded so the fork stays visible:

| Rejected | Why |
|---|---|
| An **empty allowlist**, grown import by import as R-AGP-003's incremental reading would require | The position's permitted surface would then live only as prose in `doc.go`, and the check's allowlist branch would ship **unexercised** — a branch no evidence reaches |
| A **passing bite per entry**, exercising each admission | Roughly forty lines and five further recorded runs, against a change already near its authorised review budget |

`otel/sdk/…` and `…/exporters/…` MUST be denied **by default and not by a forbidden row**: no allowlist prefix reaches either (`otelslog`'s prefix does not cover `otel/sdk`), and a forbidden row would change the failure's framing in the way `R-CPB-005` forbids. They are permitted only inside `backend/agent/src/cmd/chat/`, which is outside the scanned set (`R-CPB-006`).

Every authority citation MUST be **ADR 0005 § D1 row 3 (or § D3) together with the ADR 0009 § D2 substitution** — row 3's `…/src/coding` and § D3's `L3 coding` column are read as *"any Layer 3 archetype"*. The substitution MUST be written out where it is used, never left as an unexplained analogy, so a reader five milestones later meets it as a stated convention. This is the same read-with-substitution CH-00 promoted as `chat-archetype-contract` R-CHT-011 (`spec.md:196-200`). ADR 0005 MUST NOT be amended by this change: an ADR fixes decisions, not occupants, and a per-archetype row would have to be added again for every future archetype.

The **denial ordering is load-bearing, not cosmetic**: the forbidden table MUST be matched **before** the allowlist, because the allowlist admits `…/src/ai` as a **prefix** and an allowlist-first pass would silently admit `…/src/ai/openaicompat`. This mirrors the reason the module's existing guard pins the same ordering (`import_boundary_test.go:170-178`).

`…/src/layer3handoff` MUST remain **admissible to the archetype's test closure**, because `0005:66` (R-10) requires every harness consumer to test against the AG-23 scripted kit. At CH-01 the archetype ships no test file, so the test closure has no scanned subject; the asymmetry MUST nevertheless be recorded now, in the guard's own tables, so CH-02 **amends** it rather than inventing it.

#### Scenarios

- **S-CPB-030** — Given the guard's archetype forbidden table, when it is read, then it carries a row for each of `…/src/ai/openaicompat`, `…/src/agenttest`, `…/src/apptest`, `…/src/layer3handoff` and `…/src/cmd`, each annotated with the rule it enforces.
- **S-CPB-031** — Given the guard's archetype allowlist, when it is read, then it admits `…/src/agent` and `…/src/ai`, and every OpenTelemetry entry is a path § D3 marks ✅ for this position, or the forced closure of one, each carrying its `adr:NNN` citation in place.
- **S-CPB-032** — Given the guard's source, when any archetype-scoped citation is read, then it names ADR 0005 § D1 row 3 (or § D3) **and** the ADR 0009 § D2 substitution in the same sentence, and no citation presents row 3 as literally naming this archetype.
- **S-CPB-033** — Given a scratch file in `backend/agent/src/chat/` carrying a **blank import** (`import _ "…"`) of `…/src/ai/openaicompat/…`, when the guard runs, then it FAILS on the **forbidden-prefix** rule — proving the forbidden table is matched before an allowlist whose `…/src/ai` prefix would otherwise admit it. Recorded, then removed.
- **S-CPB-034** — Given the guard's archetype tables, when the `…/src/layer3handoff` row is read, then a comment in place records that the path is denied to the **production** closure and admissible to a future **test** closure under `0005:66`, naming CH-02 as the milestone that decides it.
- **S-CPB-035** — Given the guard's archetype tables, when they are searched for an `otel/sdk` or `…/exporters` row, then neither exists, and a comment in place states that no allowlist prefix reaches either — including `otelslog`'s — so default denial is the mechanism.
- **S-CPB-036** — Given the guard's archetype allowlist, when it is searched for `go.opentelemetry.io/contrib/bridges/otelslog`, then the entry is **present** and its comment in place cites `adr:243` together with the ADR 0009 § D2 substitution. A later change that removes this entry as though it were an oversight FAILS this scenario.
- **S-CPB-038** — Given the guard's archetype allowlist, when its recorded deviation from `R-AGP-003`'s same-commit rule is read, then the deviation is stated over the **whole table** rather than over any single entry, and it names the reason: the same-commit rule governs an incremental allowlist, whereas this table is a boundary declaration for a new position, stated once so the permitted surface is legible before any code exercises it.
- **S-CPB-039** — Given the merged tree **at CH-01's own merge state**, when each archetype allowlist entry is checked against the tree's committed imports, then **no** entry is exercised by one — which is precisely why the deviation is recorded over the whole table rather than over `otelslog` alone. The claim is scoped to this merge state deliberately: CH-02 gives the archetype its first real import, and the live successor obligation is `S-CPB-038`'s whole-table framing, which stays true whether or not an entry is later exercised.
- **S-CPB-037** — Given the guard's archetype tables read together, when `go.opentelemetry.io/contrib/bridges/otelslog` is evaluated against them in the guard's own order, then no forbidden row matches it and an allowlist entry does — so the grant is reachable, not shadowed by a row inherited from Layer 2's table.

### R-CPB-005 — The message contract: one failure names the file, names the path, and frames the denial as deny-by-default

This is the requirement CH-01.2 exists for, and it is stated over the failure's **properties**, never over its literal bytes.

A single failure emitted for an import the archetype's allowlist does not admit MUST carry all three of:

1. the **offending file**, identified by its path **relative to the declared root** of the scan — never by its base name alone;
2. the **denied import path**;
3. a rule clause framing the denial as **deny-by-default** — that the path is neither the standard library nor a package the archetype's allowlist admits, and that no forbidden prefix naming it is required for the denial to be legitimate.

All three MUST appear in **one** failure. Neither shape the existing guard already carries satisfies this alone: the closure checks name the path and frame deny-by-default but cannot name a file (a dependency listing reports package-level membership, not per-file provenance), while the source scans name the file but carry no deny-by-default framing and pass through no allowlist.

**Property 1 is tree-relative, and the reason is that this change freezes the contract.** The scan is recursive (`R-CPB-006`), so a base name stops identifying a file uniquely the moment CH-02 adds a subpackage — `chat.go` in `src/chat/` and in `src/chat/port/` would render identically, and a reader of a failure could not tell which file to open. The existing source scans may use a base name because they scan flat directories where base names are unique by construction; this check does not have that guarantee. A root-level file renders identically under both shapes, so nothing is lost at CH-01 and the ambiguity is closed before CH-02 inherits it.

**A load-bearing absence.** The archetype's forbidden table MUST NOT name `github.com/cachicamas/backend/database_administrator` or `github.com/cachicamas/backend/workspace_syncer`, even though Layer 2's own table does (`import_boundary_test.go:201-202`). Because the forbidden table is matched first (`R-CPB-004`), a row naming either module would fire before the allowlist branch and the failure would cite *a named forbidden prefix* — precisely the "missing allowlist entry" framing property 3 forbids. Their absence is what makes the required message true. R-07 (`0005:63`) remains enforced: no allowlist prefix reaches either module, so default denial is the mechanism, with ADR 0005 § D1 row 3 read under the § D2 substitution as its stated authority.

#### Scenarios

Properties 1–3 are asserted **separately**. A compound assertion would let two of three pass and read as green.

- **S-CPB-040** — **(bite, property 1)** Given a scratch file `backend/agent/src/chat/scratch_forbidden_module.go` carrying a **blank import** (`import _ "…"`) of a real Go package of another backend module, when the guard runs, then the failure identifies that file by its path **relative to the declared root** — `scratch_forbidden_module.go` at the root, and `<subdir>/<name>.go` for any file below it — and not by a base name computed independently of the root. Recorded, then removed.
- **S-CPB-041** — **(bite, property 2)** Given the same planted file and the same run, when the failure is read, then it names the **denied import path** in full.
- **S-CPB-042** — **(bite, property 3)** Given the same planted file and the same run, when the failure's rule clause is read, then it frames the denial as **deny-by-default** — stating that the path is neither the standard library nor a package this archetype's allowlist admits, and that the absence of a forbidden prefix naming it is not a licence to add it — and it does **not** cite a named forbidden prefix.
- **S-CPB-043** — Given the guard's archetype forbidden table, when it is searched for `database_administrator` and `workspace_syncer`, then neither appears, and a comment in place states why: naming either would make the failure of `S-CPB-042` cite a named forbidden prefix, falsifying property 3.
- **S-CPB-044** — Given a draft in which the archetype's forbidden table names either other backend module, when the planted import of `S-CPB-040` is run against it, then the failure's rule clause cites that named prefix and `S-CPB-042` FAILS — the absence is checked, not assumed.
- **S-CPB-045** — Given the recorded RED output of `S-CPB-040`, when it is read at verify, then all three properties are read off that one failure block, not off three different runs.
- **S-CPB-046** — Given the scan's file-naming mechanism, when it is read, then it derives the reported name relatively from the declared root, a failure to derive it is fatal rather than silently degraded to a base name, and a comment in place states why a base name is inadmissible under a recursive walk.

### R-CPB-006 — The composition root is outside the scanned set, and the negative control proves it

The archetype scan's declared roots MUST include `backend/agent/src/chat/` and MUST **not** include `backend/agent/src/cmd/chat/`. That exclusion is the mechanism behind CH-01.2's second scenario clause, *"and the same import inside the composition root passes"* (`0005:266`): the composition root is the package granted the OTel SDK and exporters (`0005:62`, R-06; ADR 0005 § D3's `cmd/` column, `adr:242`), so a scan that swept it would deny the very import the position grants.

**Within each declared root the walk MUST be recursive**, skipping directories named `testdata` and skipping `_test.go` files. Recursion is a **stated amendment** to the proposal's "directly inside" wording, not a silent widening: the archetype tree grows subpackages at CH-02, and a top-level-only scan would silently release the first subpackage from the guard — exactly the escape class this milestone exists to close. A later change MAY overturn this back to a flat scan, but doing so re-opens the subpackage escape at CH-02 and MUST assign that gap an owner.

The exclusion MUST be recorded in the scan's own source as deliberate, so a later reader does not "fix" it by adding the root.

The negative control — the identical import passing at the root — MUST be run and recorded **in its own right**. A bite without its negative control does not prove exclusion; it is equally consistent with a scan that denies nothing anywhere.

#### Scenarios

- **S-CPB-050** — **(bite)** Given a scratch file `backend/agent/src/chat/scratch_otel_sdk.go` carrying a **blank import** of a package under `go.opentelemetry.io/otel/sdk`, when the guard runs, then it FAILS naming that file and the denied path. Recorded, then removed.
- **S-CPB-051** — **(negative control)** Given a scratch file in `backend/agent/src/cmd/chat/` carrying the **identical** blank import, when the guard runs, then the archetype check PASSES with that file present. The passing run is recorded as evidence in its own right.
- **S-CPB-052** — Given the scan's declared roots, when they are read, then `…/src/cmd/chat` is absent and a comment in place states that the absence is deliberate and cites the composition root's OTel grant (`adr:242`) as the reason.
- **S-CPB-053** — Given the scan's walk, when its shape is read, then it descends into subdirectories of each declared root, skips directories named `testdata` and skips `_test.go` files, and a comment in place records the recursion as a stated amendment with its reason — a CH-02 subpackage must not silently escape the guard.

### R-CPB-007 — The vacuity floor is file-count based, per declared root, and the floor itself is proven

The archetype scan MUST fail loudly rather than pass vacuously. Per **declared root**: a walk error MUST fail naming the directory and its import path, and a root under which **zero** production `.go` files are collected MUST fail naming that root. This mirrors the module's existing per-tree floor (`import_boundary_test.go:586-588`, *"the scan would pass vacuously"*) and follows `agent-package-scaffold` `S-AGP-070`'s per-declared-input rule (`spec.md:153`).

The floor MUST be **file-count based, never dependency-count based**: a `doc.go`-only tree has zero non-standard-library dependencies but one file, and a dependency-count floor would fail the guard on a tree that is exactly as the milestone intends. This is also why the transitive closure check is deferred (see *Out of scope*).

**Floor granularity is the declared root, never the discovered subdirectory — and this is a reconciliation, not an oversight.** `S-AGP-070`'s mechanism protects **declared inputs**: a mistyped pattern or map entry must fail by name. Under the recursive walk of `R-CPB-006`, subdirectories are *discovered*, not declared — a mistyped subdirectory simply does not exist and is never visited, so a per-subdirectory floor could never fire on the failure `S-AGP-070` exists to catch, while it *would* false-positive on a legitimately Go-free directory. The declared root is therefore exactly the granularity the precedent protects; at CH-01 there is exactly one.

**The floor itself MUST be proven.** A guard that sweeps nothing while reporting green is the failure mode this requirement exists to close, and a floor that has never fired is indistinguishable from an absent one.

#### Scenarios

- **S-CPB-060** — Given the scan's source, when its floor is read, then a walk error fails naming the directory and its import path, and zero collected production files under a declared root fails naming that root.
- **S-CPB-061** — **(bite)** Given the declared root mistyped to a path that does not exist, when the guard runs, then it FATALs naming the mistyped directory. Recorded, then reverted.
- **S-CPB-062** — Given the scan's source, when the floor's basis is read, then it counts **files**, not dependencies, and a comment in place states that a `doc.go`-only tree has zero non-standard-library dependencies and would fail a dependency-count floor.

### R-CPB-008 — Creating the two packages disarms no existing guard, proven executably

Adding `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` MUST NOT silently disarm, narrow or widen any guard already shipped in this module — the CH-01 analogue of `agent-package-scaffold` R-AGP-006 (`spec.md:190-206`).

Both new trees MUST be **siblings** of `backend/agent/src/agent/`, never nested beneath it. `import_boundary_test.go:159-164` records why: the Layer 2 production allowlist admits `…/src/agent` as a **prefix**, so a tree nested under it would be silently admitted into Layer 2's production closure.

A read-only table asserting "nothing changed" is not sufficient evidence — **and neither are the executable proofs alone.** Three proofs MUST be produced; the executable ones **supplement** the table, they do not replace it:

0. **The per-guard effect table, re-verified row by row** against the post-apply tree, each row carrying the citation it rests on: Layer 1's forward guard (its scanned set is a closed, fully-qualified pattern list a new sibling tree cannot enter); the Layer 2 checks scoped to Layer 2's own pattern alone; the Layer 2 check whose pattern set is a per-pattern loop; the check that resolves its trees by directory map; and the existing `…/src/cmd` forbidden-prefix row, which this change **arms** rather than disarms.
1. **A hunk audit against the merge base.** Every hunk of `backend/agent/src/agent/import_boundary_test.go` MUST lie either in the package comment or in wholly-new archetype declarations; **zero** hunks may fall inside an existing `Test*` function, an existing table, or a shared helper. `backend/agent/src/ai/import_boundary_test.go` MUST be byte-unchanged against the merge base. This diffs guard **bodies**, not merely entry lists — a guard released from its own coverage by an edit to its logic is invisible to a table that only counts entries.
2. **A re-plant bite.** After the extension lands, a violation of an **existing** check MUST be planted as a blank import and watched to fail, then removed, under a run whose selector names the affected checks explicitly. Checks whose bodies proof 1 shows byte-identical to already-bitten shipped code rest on proofs 0 and 1.

#### Scenarios

- **S-CPB-070** — Given the merged diff of `backend/agent/src/agent/import_boundary_test.go` against the merge base, when every hunk is classified, then each lies in the package comment or in a wholly-new archetype declaration, and none lies inside an existing `Test*` function, an existing table, or a shared helper.
- **S-CPB-071** — Given the merged diff, when `backend/agent/src/ai/import_boundary_test.go` is diffed against the merge base, then it is empty — Layer 1's forward guard is untouched, and its scanned set is a closed pattern list a new sibling tree cannot enter.
- **S-CPB-072** — **(bite)** Given a scratch file planted in `backend/agent/src/agent/` carrying a **blank import** of a package an existing check denies, when a run whose selector names the affected checks explicitly is made after this change's extension has landed, then each still FAILS. Recorded, then removed.
- **S-CPB-073** — Given the repository after this change, when the locations of `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` are read, then both are siblings of `backend/agent/src/agent/` and neither is nested beneath it, so no existing production allowlist prefix reaches either.
- **S-CPB-074** — Given the module's existing `…/src/cmd` forbidden-prefix row (`import_boundary_test.go:204`), when it is read after this change, then it is unchanged and now names a tree that exists — armed for the first time rather than disarmed.
- **S-CPB-075** — Given the per-guard effect table of proof 0, when each row is re-verified against the **post-apply** tree rather than against the merge base, then each row's claim still holds and each cites the source location it rests on, re-resolved after the insertion rather than carried over from a pre-apply line number.

### R-CPB-009 — No module dependency is added

This change MUST NOT alter `backend/agent/go.mod`, `backend/agent/go.sum`, `backend/agent/Makefile`, or `backend/agent/.golangci.yml`. Each MUST be byte-unchanged against the merge base with `origin/main`, and the diff MUST add no `require` entry.

This is not a stylistic preference. Shipped Layer 2 tests assert on every branch that the merge-base diff of `backend/agent/go.mod` and `backend/agent/go.sum` is **empty** — nine such call sites were measured during this change, a count this requirement deliberately does not fix, since later milestones may add or remove call sites without changing the obligation. Adding any dependency reddens every one of them.

The obligation is satisfiable because the archetype scan reads **file bytes** with a parser in imports-only mode and **resolves nothing**: its verdict is a function of the import path as written, not of the module graph, so it neither requires a path to be in the build list nor is weakened when one happens to be. What the byte-freeze constrains is **committed** source: a real committed import would make `go mod tidy` add a `require`, drifting a frozen file. Scratch files are transient and removed before merge, so they never reach that constraint.

#### Scenarios

- **S-CPB-080** — Given the merged diff of this change, when `git diff <merge-base with origin/main> -- backend/agent/go.mod backend/agent/go.sum backend/agent/Makefile backend/agent/.golangci.yml` runs, then its output is empty for each path.
- **S-CPB-081** — Given the merged tree, when every shipped assertion that requires the `go.mod`/`go.sum` merge-base diff to be empty is run uncached, then each passes.
- **S-CPB-082** — Given the archetype scan's source, when its parse mode is read, then it parses imports only and resolves no import, and a comment in place states that its verdict is a function of the import path as written rather than of the module graph.

### R-CPB-010 — Bite proof is the closing condition, not green

CH-01.2 is a `[guard]` leaf. Its closing condition is a **recorded RED** against a planted violation, followed by green on the merged tree with **every scratch file removed**. A guard that has only ever been green is unproven.

Every bite run MUST be **scoped** — `go test -race -count=1 -run TestChatArchetype ./src/agent/`, or the equivalent selector naming the checks under test — for **focus and isolation of the check under test**, and for that reason only. The scoping MUST NOT be recorded as a necessity, because it is not one: the repository root's `go.work` uses all three backend modules, so in workspace mode a sibling module's packages are importable without a `require` in `backend/agent/go.mod`, and the build list is the union of all used modules' requirements. Both planted scratches therefore **build**, and a full `./...` run does reach the guard. A scenario that recorded a false toolchain claim as its reason would archive that claim permanently.

Every planted scratch file MUST use a **blank import** (`import _ "…"`) so it compiles under any run shape without a use site, while remaining visible to both an imports-only parse and a dependency listing. A non-blank unused import is a compile error, and the resulting build failure would masquerade as a bite.

What remains true, and MUST be recorded separately, is that the `go.mod` byte-freeze binds **committed** code: a real committed import would make `go mod tidy` add a `require`, drifting a file `R-CPB-009` freezes.

Every planted scratch file MUST be removed before merge, and the merged tree MUST report clean under both new trees.

#### Scenarios

- **S-CPB-090** — Given each bite scenario of this spec (`S-CPB-033`, `S-CPB-040`–`S-CPB-042`, `S-CPB-050`, `S-CPB-061`, `S-CPB-072`), when its evidence is read, then it contains the **full failing output** of a scoped run, not a summary or a claim that it failed.
- **S-CPB-091** — Given the recorded bite runs, when their commands and the reason recorded alongside are read, then the reason given for scoping is **focus and isolation of the check under test**, and no evidence claims that an unscoped run would fail at build or would not reach the guard.
- **S-CPB-092** — Given the merged tree **at CH-01's own merge state**, when `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` runs, then its output is exactly `backend/agent/src/chat/doc.go` and `backend/agent/src/cmd/chat/main.go` — no `_test.go` file, no second guard, nothing else — and when `git status --porcelain -- backend/agent/src/chat backend/agent/src/cmd` runs, then its output is empty, so no untracked scratch file remains. The enumeration is scoped to this merge state deliberately: CH-02 adds files to these trees by design, and the live successor obligations are `R-CPB-003`'s no-second-guard property and this requirement's scratch-removal property, both verifiable at any tree.
- **S-CPB-093** — Given the merged tree with the test cache cleared, when `cd backend/agent && go test -race -count=1 ./...` runs, then it passes and its wall-clock duration is recorded. A sub-second result is a cache hit and is not evidence.
- **S-CPB-094** — Given the merged tree, when `make lint` runs over the module, then it exits 0 with no reported issue; and when `gofmt -l` runs **over the three files this change touches** — `backend/agent/src/chat/doc.go`, `backend/agent/src/cmd/chat/main.go` and `backend/agent/src/agent/import_boundary_test.go` — then it prints nothing. `make all` is not run: its formatting step rewrites committed files, which is precisely how the drift described below would be silently absorbed instead of recorded.

  **The two operands are deliberately asymmetric, and a reader MUST NOT "helpfully" widen the second.** `make lint` is asserted **module-wide** because it is module-wide clean. `gofmt -l` is **pinned to this change's own files** because run module-wide it lists files this change never touches — each of them byte-unchanged against the merge base `419a4291`, i.e. **pre-existing formatting drift from a toolchain version change, entirely unrelated to CH-01**. `R-CPB-009` and the design's "no other Go file changes" rule forbid this change from repairing them, so a module-wide `gofmt -l` assertion would be **false at this scenario's own merge state** — the exact untemporal-invariant defect this capability's register exists to close, written fresh in the capability doing the closing. Widening the operand back to the module asserts a property over files this change is forbidden to touch. Repairing that drift needs its own change, with its own review of what a formatter version bump rewrites.
- **S-CPB-095** — Given each planted scratch file as recorded, when its source is read, then its import is **blank** (`import _ "…"`), and the evidence states that this is what lets it compile with no use site so a build failure cannot be mistaken for a bite.
- **S-CPB-096** — Given the workspace-mode claim this requirement rests on, when it is checked, then a file in this module carrying a blank import of a sibling backend module's package, and one carrying a blank import of an observability SDK package, each **build**; and the separate, still-live claim — that a real *committed* import would drift `backend/agent/go.mod` through `go mod tidy` — is recorded distinctly from it rather than conflated with it.

## Non-functional requirements

### NFR-CPB-001 — Guard determinism

The archetype scan MUST be deterministic and hermetic: it reads files inside the repository and shells out only to the local Go toolchain with fixed arguments. It MUST resolve its roots from the test's own source location rather than an assumed working directory, MUST pass under `-race`, and MUST NOT depend on the directory from which the test command is invoked. It MUST declare itself parallel-safe in the same way every existing check in that file does, so adding it does not serialise a suite that runs its guards concurrently.

### NFR-CPB-002 — Guard legibility

The scan's failure MUST name the offending file and path **and** the rule that rejected it, so a failure five milestones later is actionable without reading the guard's source. Every table entry MUST cite, in place, the ADR clause authorising or denying it, together with the ADR 0009 § D2 substitution where one is used.

### NFR-CPB-003 — Standard-library classification

The standard library MUST be classified by the Go toolchain's own answer, never by a path-shape heuristic or a maintained list, honouring the warning the module's existing guard already records against both. If that subprocess fails, the check MUST fail **fatally, carrying the command's standard error** — matching every existing toolchain-invocation failure path in that file. A classification that silently degrades would turn every standard-library import into a deny-by-default failure, or every import into an admission, depending on which way it degraded.

## Untemporal-invariant register — the complete result of a tree-wide sweep

Creating two packages that promoted specs had asserted would not exist falsifies clauses across `openspec/specs/`. **The tree was swept deliberately for the whole shape** — any promoted claim that a directory does not exist, or that a layer has no code — rather than repairing only what happened to surface. This register carries **every** site the sweep found, with its disposition: three repaired by a delta in this same pull request, one recorded and deliberately not repaired. It exists so a reader sees the complete result of the sweep rather than three repairs and an unexplained silence.

| # | Site | Falsified by | Disposition | Delta |
|---|---|---|---|---|
| 1 | `agent-package-scaffold` `R-AGP-001` (`spec.md:35`) and `S-AGP-003` (`:41`) — `…/src/coding` and `…/src/cmd` "MUST NOT exist in any form" | **This merge.** CH-01.1 creates `backend/agent/src/cmd/chat/` | **Repaired** — scoped to AG-03's own merge state; the live obligation becomes the **import** rule, and `S-CPB-074` shows this change **arms** the `…/src/cmd` forbidden row rather than disarming it | `specs/agent-package-scaffold/spec.md` |
| 2 | `chat-archetype-contract` `R-CHT-007` (`spec.md:145-155`) with `S-CHT-061` and `S-CHT-062`; `R-CHT-013`'s rationale clause (`:226`); two prose sites (`:19`, `:256`) — "neither path exists yet" | **This merge.** CH-01.1 creates both paths, making `S-CHT-062`'s `Given` unreachable | **Repaired** — scoped to CH-00's own merge state, each changed scenario naming its live successor | `specs/chat-archetype-contract/spec.md` |
| 3 | `ai-observability-boundary` *Out of scope* row (`spec.md:228`) — "Docs 0003/0004 — those directories do not exist" | **Partly already false; worsened by this merge.** `backend/agent/src/agent/` has existed since **AG-03** — that clause is AG-03's back-annotation debt, inherited already broken. CH-01.1 falsifies the Layer 3 and composition-root clauses | **Repaired, with split ownership recorded** — the reason becomes those positions' own layer documents and ADR 0005 § D3; the scope fence is unchanged, and the delta states which clause was already false and whose debt it is | `specs/ai-observability-boundary/spec.md` |
| 4 | `agent-contract-vocabulary` `NFR-AGV-C` (`spec.md:250`) — *"No citation MAY point at Layer 2 code, which does not exist."* | **Not this merge.** AG-03 created `backend/agent/src/agent/` as documentation only; **AG-04** onward gave Layer 2 code. Stale since AG-04, independently of CH-01 | **Recorded, not repaired** — see below | none |

### Row 4 — recorded, not repaired, with its cause

`agent-contract-vocabulary` `NFR-AGV-C` (`openspec/specs/agent-contract-vocabulary/spec.md:250`) states that no citation may point at Layer 2 code, *"which does not exist"*, and requires citations to resolve instead to a contract document, an ADR, the architecture reference, or the shipped Layer 1 surface. Layer 2 code exists, so the subordinate clause has been false since **AG-04** — before this change's branch point.

It is **not repaired here, by decision.** CH-01 neither causes this staleness nor worsens it: nothing in this change adds Layer 2 code, cites Layer 2 code, or touches `agent-contract-vocabulary`. Repairing it would bury an unrelated milestone's correction inside a scaffolding pull request, where no reviewer of this change is positioned to judge whether the requirement's *actual* obligation — that vocabulary citations resolve to durable documents rather than to source that moves — should also be restated while the false reason is removed. That judgement belongs to a change that owns the vocabulary capability.

It is recorded rather than left silent because **an unrepaired defect recorded with its cause is a different object from one nobody looked for.** Both sides are cited above, the milestone it went stale at is named, and the deferral is a decision rather than an omission.

### The shape

Rows 1, 2 and 3 share one mechanism: a promoted invariant stated **without temporal scope**, falsified by a later milestone that the same document had, in two of the three cases, already named as the owner of the falsifying act. Row 4 is the same shape with a different cause — a reason that expired with no single merge to point at, which is exactly why nobody had looked for it.

This capability is written so as not to become row 5: `S-CPB-039` and `S-CPB-092` carry explicit merge-state scoping, and `S-CPB-074` states the `…/src/cmd` row's arming as a property of the current tree rather than of a moment.

## Out of scope at CH-01, with the milestone that owns each

- **A transitive (`go list -deps`) closure check over `backend/agent/src/chat/...` — owned by CH-02.** The archetype check shipped here is **zero-hop**: it reads each production file's own import declarations and nothing further. The transitive check is deferred for a **mechanical** reason, not a budget one: at CH-01 the archetype is `doc.go`-only and imports nothing, so a dependency listing returns **zero** non-standard-library packages, and this repo's pinned zero-packages vacuity floor (`agent-package-scaffold` `S-AGP-070`, mechanised at `import_boundary_test.go:284-287`) would **fail** the guard. Satisfying it at CH-01 would require inventing a floor shape the repo has not pinned.
  **The exposure window is bounded and stated rather than implied**: between CH-01 and CH-02, an import reached only *transitively* through an admitted path is uncaught. The only admitted first-party paths are `…/src/agent` and `…/src/ai`, whose own production closures the module's existing check 1 already proves deny-by-default, so the uncaught set is bounded by those two closures. CH-02, which gives the archetype its first real import, is where the closure check becomes both cheap and non-vacuous. The zero-hop gap MUST also be stated in the guard's own source comment, so it is met by a reader of the code and not only by a reader of this spec.
- Any behaviour in either package — ports, projection, wiring, environment reads. The archetype's conversation is CH-02; the composition root's wiring is CH-04.1 (`0005:240`).
- An "only the composition root reads the environment" guard — CH-04.2 (`0005:304`). A distinct property with its own node and its own bite.
- A machine-checked doc-row table for `backend/agent/src/chat/doc.go` — `agent-package-scaffold` R-AGP-002 (`spec.md:45-51`) is scoped to `backend/agent/src/agent/doc.go`; CH-01 has no doc-row node.
- The archetype's **test** closure allowlist — recorded here as an asymmetry (`R-CPB-004`) so CH-02 amends it, but no test file exists at CH-01 for it to scan.
- A per-binary `build` target for `cmd/chat` in `backend/agent/Makefile` — `R-CPB-009` forbids the Makefile edit; the existing `go build ./...` already compiles a `package main`.
- An amendment to ADR 0005 adding `src/chat` or `cmd/chat` rows — settled as read-with-substitution, not amendment (`R-CPB-004`).
- The runtime-behaviour boundary — a guard proves what may be imported, not what is done with it (`0005:276`).
- CI. Every gate here runs when a human runs the test command in `backend/agent/`.

## Evidence discipline

`openspec/config.yaml` declares `apply.tdd: true`.

- **Guard requirements (`R-CPB-003` through `R-CPB-008`, `R-CPB-010`)** are RED-first by construction. The closing condition is the recorded red run against a deliberate violation, followed by green.
- **The mechanical requirements (`R-CPB-001`, `R-CPB-002`, `R-CPB-009`)** are exempt from red-green — there is no behaviour to drive out — but never exempt from their recorded check evidence.
- The final suite run MUST be **uncached**: the test cache is cleared, `go test -race -count=1 ./...` is run, and its **wall-clock duration is recorded**. A `(cached)` result is not evidence.
- `make all` MUST NOT be run: its formatting step rewrites committed files and turns a verification step into a mutation. That prohibition is load-bearing here rather than stylistic — the module carries pre-existing `gofmt` drift (see `S-CPB-094`), and `make all` would absorb it into this change's diff silently, where a scoped `gofmt -l` records it instead.
- **Every command-shaped assertion in this spec names its operand.** An assertion over an unscoped command's output is falsifiable by drift this change neither caused nor may repair; `S-CPB-094` is where that was found and fixed, and the same discipline is why `S-CPB-020`, `S-CPB-021`, `S-CPB-071`, `S-CPB-080` and `S-CPB-092` each carry explicit paths. The two deliberate exceptions are `S-CPB-093`'s full-suite run and `S-CPB-094`'s `make lint`, both of which assert a **module-wide** property that is genuinely module-wide true and is a real obligation of this change rather than an artefact of its scope.
- Bite runs are **scoped** per `R-CPB-010` for focus and isolation, never out of necessity; only the clean-tree gate runs `./...`. Every scratch import is blank.

## Acceptance criteria

The contract holds when:

1. Every scenario this spec declares, from `S-CPB-001` to `S-CPB-096`, has recorded evidence.
2. `backend/agent/src/chat/` is `package chat` declaring nothing, and `backend/agent/src/cmd/chat/` is `package main` whose binary builds.
3. One failure of the archetype check carries the file, the path, and deny-by-default framing, read off a single recorded RED block (`R-CPB-005`).
4. The composition root's exclusion is proven by both the bite and its negative control (`R-CPB-006`).
5. The vacuity floor has been watched to fire against a mistyped root (`R-CPB-007`).
6. All three proofs of `R-CPB-008` pass — the per-guard effect table re-verified row by row against the post-apply tree, the hunk audit, and the re-plant bite; no existing guard was disarmed.
7. The `otelslog` grant is present in the archetype allowlist, cited to `adr:243` under the ADR 0009 § D2 substitution, and no forbidden row shadows it; and the deviation from `R-AGP-003`'s same-commit rule is recorded over the **whole table**, in this spec and in the guard's own comment, with its declaration-versus-incremental reasoning and its two rejected alternatives (`R-CPB-004`).
8. `backend/agent/go.mod`, `go.sum`, `Makefile` and `.golangci.yml` are byte-unchanged against the merge base.
9. No second, self-contained guard exists inside either new tree, and no `_test.go` file was added under either.
10. Every scratch file used a blank import and every one is removed; the merged tree is clean and the uncached suite is green with its wall-clock recorded.
11. No recorded evidence states that an unscoped run would fail at build or would not reach the guard (`S-CPB-091`).
12. The deferred transitive closure check is recorded here and in the guard's source with its owner (CH-02), its mechanical reason, and the bound on its exposure window.
