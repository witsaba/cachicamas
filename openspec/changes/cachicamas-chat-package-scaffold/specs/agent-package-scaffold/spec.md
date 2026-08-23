# Delta for `agent-package-scaffold`

> **Change**: `cachicamas-chat-package-scaffold` · **Milestone**: CH-01 (Wave 0) of doc 0005 · proposal **D-2**
> **Target**: `openspec/specs/agent-package-scaffold/spec.md`
> **Authority to amend**: that spec's own rule at `spec.md:21` — *"A later milestone that needs to change one of these invariants … amends **this file**, in the same pull request"*. Its status line at `spec.md:6` calls its invariants **live**, which is exactly why this is a repair and not a historical footnote.

## Why this delta exists

`R-AGP-001` (`spec.md:35`) states that `backend/agent/src/coding/` and `backend/agent/src/cmd/` *"MUST NOT exist in any form, including as an empty directory"*, and `S-AGP-003` (`spec.md:41`) requires `test -e backend/agent/src/cmd` to exit non-zero. CH-01.1 creates `backend/agent/src/cmd/chat/`.

That is not a future tension. It is a promoted, live invariant that becomes **literally false** on the day this change merges unless the same pull request repairs it. Recording it in an inconsistency register would not do: a register records a defect nobody's merge forces, and this merge forces it.

**Verified during this change**: no live Go test enforces `S-AGP-003` — it is spec text only. This delta therefore has no code consequence. The module's own `…/src/cmd` forbidden-prefix row (`backend/agent/src/agent/import_boundary_test.go:204`) is untouched and is **armed** rather than disarmed by the arrival of `src/cmd/chat`: it previously named a tree that did not exist.

## Scope of this delta, pinned

**Changed** — and nothing else:

1. `R-AGP-001`'s second sentence, the `backend/agent/src/coding/` and `backend/agent/src/cmd/` non-existence clause: scoped to AG-03's own merge state instead of standing as a live invariant.
2. `S-AGP-003`: scoped to the same merge state, with the live successor obligation stated.

**Deliberately untouched**, so a reviewer can diff the claim:

- `R-AGP-001`'s first sentence — `backend/agent/src/agent/` exists as a Go package named `agent` carrying documentation and **nothing else** — survives **byte-identical**. It is not weakened, narrowed, or re-scoped.
- `R-AGP-001`'s third sentence — the `go.mod`/`go.sum`/`Makefile`/`.golangci.yml` clause — survives byte-identical. It is likewise AG-03-scoped in fact, but this change does not rely on that: proposal D-1 independently forbids touching all four, and `chat-package-boundary` `R-CPB-009` asserts them byte-unchanged.
- `S-AGP-001`, `S-AGP-002`, `S-AGP-004`, `S-AGP-005` survive byte-identical.
- No other requirement of this spec changes. `R-AGP-003`'s guard contract is **extended in practice** by CH-01's new check, but its normative text needs no edit: it is already stated over a per-check pattern set (`spec.md:93-102`) rather than over a fixed list of checks.
- No scenario is renumbered. No identifier is minted by this delta.

## MODIFIED Requirements

### R-AGP-001 — The Layer 2 package exists and declares nothing

`backend/agent/src/agent/` MUST exist as a Go package named `agent`, carrying package documentation and **nothing else** — no type, no constant, no function, no variable. **At AG-03's own merge state**, `backend/agent/src/coding/` and `backend/agent/src/cmd/` MUST NOT have existed in any form, including as an empty directory or a directory holding only a placeholder; that clause records what AG-03 shipped and is **not** a live invariant over the module's lifetime. A later milestone MAY create either tree, and doc 0005's CH-01.1 does exactly that for `backend/agent/src/cmd/chat/`. What remains live is the **import** rule, not the non-existence: `…/src/coding` and `…/src/cmd` MUST remain denied-by-name rows in the module's forbidden-prefix table under `R-AGP-003`, so Layer 2 provably never imports Layer 3 or the composition root — and a tree that exists is what makes those rows reachable rather than aspirational. The change MUST NOT alter `backend/agent/go.mod`, `go.sum`, `Makefile`, or `.golangci.yml`.

(Previously: the `backend/agent/src/coding/` and `backend/agent/src/cmd/` non-existence clause was stated as a live invariant with no temporal scope; doc 0005's CH-01.1 creates `backend/agent/src/cmd/chat/`, which would have falsified the promoted requirement on the day that change merged.)

#### Scenarios

- **S-AGP-001** — Given the repository after this change, when `backend/agent/src/agent/` is listed, then it exists, is a directory, and every `.go` file in it declares `package agent` or `package agent_test`.
- **S-AGP-002** — Given `backend/agent/src/agent/`, when its Go declarations are enumerated (for example `go doc github.com/cachicamas/backend/agent/src/agent`), then the package exports nothing and declares nothing beyond the package clause and its documentation comment.
- **S-AGP-003** — Given the repository **at AG-03's merge state**, when `test -e backend/agent/src/coding` and `test -e backend/agent/src/cmd` each run, then each exits non-zero. This scenario is verified against AG-03's merged tree and is **not** re-asserted against later trees: from doc 0005's CH-01.1 onward `backend/agent/src/cmd/chat/` exists by design. The live successor obligation is that `…/src/coding` and `…/src/cmd` remain forbidden-prefix rows under `R-AGP-003`, verifiable at any tree by reading the guard's table — `S-AGP-023` is where that is asserted.

  (Previously: stated over "the repository after this change" with no temporal scope, so it read as a standing invariant that CH-01.1's composition root falsifies.)
- **S-AGP-004** — Given the repository after this change, when `cd backend/agent && make test` and `make lint` run, then each exits 0 with no failing test and no reported issue.
- **S-AGP-005** — Given the merged diff of this change, when `backend/agent/go.mod`, `go.sum`, `Makefile` and `.golangci.yml` are inspected, then each is byte-unchanged and the diff adds no `require` entry.

## Amendment note to append at promotion

The archive executor MUST append the following blockquote to the target spec's amendment list (after the AG-23 row at `spec.md:29`), matching that list's existing convention:

> **Amended 2026-08-23 (CH-01, `cachicamas-chat-package-scaffold`)** by the archive executor: `R-AGP-001` and `S-AGP-003` are MODIFIED. Their `backend/agent/src/coding/` and `backend/agent/src/cmd/` non-existence clause is scoped to **AG-03's own merge state** rather than stated as a live invariant, because doc 0005's CH-01.1 creates `backend/agent/src/cmd/chat/` and would otherwise falsify a promoted requirement on the day it merged. The live obligation is restated as the **import** rule: `…/src/coding` and `…/src/cmd` remain denied-by-name rows in `R-AGP-003`'s forbidden-prefix table, and `backend/agent/src/cmd/chat`'s arrival arms that row rather than disarming it. `R-AGP-001`'s `backend/agent/src/agent/` "exists, declares nothing" clause and its `go.mod`/`go.sum`/`Makefile`/`.golangci.yml` clause are **byte-unchanged**; `S-AGP-001`, `S-AGP-002`, `S-AGP-004` and `S-AGP-005` are byte-unchanged; no other requirement is touched and no scenario is renumbered. No live Go test enforced `S-AGP-003`; this is a spec repair with no code consequence.

## Verification of this delta

- **V-1** — Given the promoted target after archive, when `R-AGP-001`'s first and third sentences are diffed against their pre-promotion bytes, then each is byte-identical.
- **V-2** — Given the promoted target after archive, when `S-AGP-001`, `S-AGP-002`, `S-AGP-004` and `S-AGP-005` are diffed against their pre-promotion bytes, then each is byte-identical.
- **V-3** — Given the promoted target after archive, when it is searched for a clause asserting that `backend/agent/src/cmd` does not exist at the current tree, then none remains.
- **V-4** — Given the promoted target after archive, when the set of requirement and scenario identifiers is diffed against the pre-promotion set, then the two sets are equal — this delta mints nothing and renumbers nothing.
