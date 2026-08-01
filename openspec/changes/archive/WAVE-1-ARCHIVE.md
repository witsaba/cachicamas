# Wave 1 Archive Report — Layer 1 model adapter (AI-04 … AI-13)

> **SDD archive status**: **closed** — ten changes verified, their contracts promoted into `openspec/specs/`, doc 0002 amended.
> **Change set**: `cachicamas-ai-layer-1` Wave 1 — ten SDD changes closed as one deliverable.
> **Worktree**: `cachicamas-worktrees/ai-wave-1-archive` · **Branch**: `docs/2026-08-01-cachicamas-ai-wave-1-archive`, based at `15408a8`.
> **Verify report**: `openspec/changes/archive/WAVE-1-VERIFY.md` · Engram `sdd/cachicamas-ai-layer-1/wave-1-verify-report` (observation **#2323**).
> **Archive date**: 2026-08-01.

---

## 1. Delivery state — read this before citing anything below

| Fact | State |
| --- | --- |
| SDD archive | **closed** |
| PR #101 | **OPEN, not merged.** The merge is owed by the user/reviewer, not by this archive. |
| `main` | `efdedc4` — **does not contain Wave 1**. Wave 0 (AI-00 … AI-03) is the newest Layer 1 code on `main`. |
| Wave 1 implementation head | `15408a8` (74 commits from `efdedc4`) |
| doc 0002 shipped counter | **14 of 41** |

Every promoted spec's `> **Introduced by**` line states PR #101 as open. No promoted spec claims `main` carries this code. When PR #101 merges, the honest edit is to append the merge commit to those lines; nothing else about the specs changes.

---

## 2. What this run corrected

An earlier archive attempt exhausted its budget mid-run and left the tree in a state its own report (Engram observation **#2330**) described inaccurately. This report supersedes #2330. The corrections:

| #2330 claimed | Actually true |
| --- | --- |
| "All change folders moved" to `openspec/changes/archive/2026-08-01-*` | **False at the time it was written and still false.** No folder had been moved. See § 6 — the moves remain outstanding. |
| Four capability specs "fully merged" (`ai-validation-errors`, `ai-message-roles`, `ai-content-parts`, `ai-reasoning-content`) | **False.** All four were **verbatim copies of the delta file**. Each still carried its delta header — including the self-referential line *"Canonical spec: openspec/specs/…/spec.md — created by sdd-archive from this delta"* inside that very file — and each carried `../../../../specs/ai-contract-vocabulary/spec.md`, which does not resolve from `openspec/specs/<capability>/spec.md`. All four have been rewritten in this run. |
| Six specs "not yet merged … manual copy required" | Superseded. All six were authored in this run as promoted specs, not copied. |
| doc 0002 amendment 3 (traceability spine) complete | **Two rows were wrong.** `G4` and leakage row 9 were marked fully **Closed** when only their Layer 1 half is closed. Both corrected. |
| doc 0002 amendment 4 complete | **One identifier was wrong.** The recurrence list read `AI-10.5`; the recurring failure is `AI-10.6`. Corrected, and the obligation's text expanded. |
| doc 0002 amendment 5 complete | **Incomplete.** It omitted the depth-5 000 agreement, the reachability path `tool_call.go:91`, the generator's file:line, and — the substantive omission — the **false doc-comment claim at `json_syntax_differential_internal_test.go:28`**, which the brief and § 5.3 of the verify report both make part of the obligation. Rewritten. |

**The promotion defect, stated once so it is not repeated.** Promoting a delta is not a copy. The delta header is replaced by the live-spec header; every relative link is re-resolved from `openspec/specs/<capability>/spec.md`; a *Status — this file is the canonical home of the contract* section is added; and paths naming the change folder are rewritten to the archive path. Wave 0's four promoted specs are the reference for all four transformations.

---

## 3. Per-milestone table

| Milestone | Change | Capability spec | Verdict | Closes (doc 0002 spine) |
| --- | --- | --- | --- | --- |
| AI-04 | `cachicamas-ai-validation-errors` | `openspec/specs/ai-validation-errors/spec.md` | PASS | no tracked gap row; the taxonomy every later Wave 1 milestone reports through |
| AI-05 | `cachicamas-ai-message-roles` | `openspec/specs/ai-message-roles/spec.md` | PASS | completion-checklist item 3 (with AI-06 … AI-10) |
| AI-06 | `cachicamas-ai-content-parts` | `openspec/specs/ai-content-parts/spec.md` | PASS | **C1** in full; Layer 1 half of **C2** |
| AI-07 | `cachicamas-ai-reasoning-content` | `openspec/specs/ai-reasoning-content/spec.md` | PASS | Layer 1 half of **G12(b)** |
| AI-08 | `cachicamas-ai-tool-declarations` | `openspec/specs/ai-tool-declarations/spec.md` | PASS | no tracked gap row; forced `ErrDuplicate` + `S-AIE-034`/`S-AIE-035` into AI-04's contract |
| AI-09 | `cachicamas-ai-tool-messages` | `openspec/specs/ai-tool-messages/spec.md` | PASS | Layer 1 half of **G5** |
| AI-10 | `cachicamas-ai-model-request` | `openspec/specs/ai-model-request/spec.md` | PASS | Layer 1 half of leakage **row 8**; last node of Layer 1 half of **C2** |
| AI-11 | `cachicamas-ai-cache-breakpoints` | `openspec/specs/ai-cache-breakpoints/spec.md` | PASS | Layer 1 half of **G4** and of leakage **row 9** |
| AI-12 | `cachicamas-ai-request-extension-points` | `openspec/specs/ai-request-extension-points/spec.md` | PASS | **G9**; the mechanism recorded as meeting **G11** |
| AI-13 | `cachicamas-ai-completion-metadata` | `openspec/specs/ai-completion-metadata/spec.md` | PASS | **G12(c)**; the mechanism recorded as meeting **G10** |

Ten of ten PASS. 116/116 requirements, 420/420 scenarios, 345/345 task checkboxes complete — per the verify report at its verification time, with no later work reopening any of them.

---

## 4. Evidence gate

Recorded by `sdd-verify` at head `e76bab8`, from `backend/agent/`:

| Gate | Result |
| --- | --- |
| `make test` (`go test -race -v ./...`) | exit 0 — 176 top-level tests, 745 PASS lines including subtests, 0 FAIL, 0 SKIP |
| `make lint` | exit 0 — 0 issues |
| `backend/agent/go.mod` | zero `require` lines |
| AI-00 forward guard | passes — `src/ai` closure is stdlib plus the module's own packages only |
| AI-00 reverse guard | passes — run from `backend/database_administrator/` |
| AI-10.4 dependency-closure guard | passes — no `net`, `net/http`, `os`, `io/fs`; `fmt` and `encoding/json` excluded transitively |
| `evidence_revision` | `sha256:9c00f7ebdbea26cf1894ad8880d8aa3d20e90c604596d883e66a05b6ad49d79d` |

This archive run changed **nothing under `backend/`**. Only `docs/architecture/milestones/0002-…md` and files under `openspec/` were touched, so the evidence above still describes the tree.

---

## 5. The four verify warnings, at close

| # | Warning | State at close |
| --- | --- | --- |
| **W1** | `isWellFormedJSON` (`json_syntax.go:55`) has no nesting-depth cap: agrees with `encoding/json.Valid` at depth 5 000, disagrees at 10 001, `fatal error: stack overflow` at 20 000 000. Reachable from `tool_call.go:91`. Its differential generator caps nesting at ~5 (`json_syntax_differential_internal_test.go:76`) and that test's doc comment at line 28 falsely claims coverage of "deeply nested structures". | **Deferred to Wave 2**, recorded in doc 0002's traceability spine by this run. No untrusted input path exists until AI-24. |
| **W2** | `S-AMR-046` prose said "four permitted cells"; the requirement's own table, `rolePermittedKinds` and the test all say five. | **Closed by `c54f01c`.** The delta spec now reads "five permitted cells", and the promoted `ai-model-request` spec carries the corrected prose. |
| **W3** | `R-AMR-`/`S-AMR-` prefix collision between AI-05 and AI-10. | **Closed by `15408a8`.** AI-05 was re-prefixed to `R-AMSG-`/`S-AMSG-`; AI-10 keeps `R-AMR-`/`S-AMR-`. No number was renumbered — `R-AMR-011` (totality) became `R-AMSG-011`. The promoted `ai-message-roles` spec carries the new prefix and an identifier note pointing a stale `R-AMR-` citation at `ai-model-request`. |
| **W4** | doc 0002 fourteen milestones stale; five amendments owed. | **Closed by this run** — see § 7. |

---

## 6. Archive operations — what landed, and what is still owed

### Landed in this run

**Ten capability specs promoted into `openspec/specs/`.** Four were rewritten from the prior agent's broken copies; six were authored from their deltas. Every one carries the live-spec header, resolving relative links, and a canonical-home statement.

**`openspec/specs/ai-contract-vocabulary/spec.md` was not touched.** Not edited, not moved, not frozen, not marked closed. AI-04.1's amendment appending `V-FAIL-16` (**validation rule**) and `V-FAIL-17` (**rule class**), and moving the term count 114 → 116, stands exactly as it landed. This is the AI-01 archive lesson from Wave 0, and it held.

**Five doc 0002 amendments verified and, where wrong, corrected** — § 7.

### Still owed — this run could not perform it

The ten change folders are **still at `openspec/changes/<change-name>/`**. They must move to `openspec/changes/archive/2026-08-01-<change-name>/`, matching Wave 0's convention:

```
openspec/changes/cachicamas-ai-validation-errors        → openspec/changes/archive/2026-08-01-cachicamas-ai-validation-errors
openspec/changes/cachicamas-ai-message-roles            → openspec/changes/archive/2026-08-01-cachicamas-ai-message-roles
openspec/changes/cachicamas-ai-content-parts            → openspec/changes/archive/2026-08-01-cachicamas-ai-content-parts
openspec/changes/cachicamas-ai-reasoning-content        → openspec/changes/archive/2026-08-01-cachicamas-ai-reasoning-content
openspec/changes/cachicamas-ai-tool-declarations        → openspec/changes/archive/2026-08-01-cachicamas-ai-tool-declarations
openspec/changes/cachicamas-ai-tool-messages            → openspec/changes/archive/2026-08-01-cachicamas-ai-tool-messages
openspec/changes/cachicamas-ai-model-request            → openspec/changes/archive/2026-08-01-cachicamas-ai-model-request
openspec/changes/cachicamas-ai-cache-breakpoints        → openspec/changes/archive/2026-08-01-cachicamas-ai-cache-breakpoints
openspec/changes/cachicamas-ai-request-extension-points → openspec/changes/archive/2026-08-01-cachicamas-ai-request-extension-points
openspec/changes/cachicamas-ai-completion-metadata      → openspec/changes/archive/2026-08-01-cachicamas-ai-completion-metadata
openspec/changes/WAVE-1-VERIFY.md                       → openspec/changes/archive/WAVE-1-VERIFY.md
```

**Ordering matters.** Each promoted spec links to `../../changes/archive/2026-08-01-<change-name>/`. Those links resolve only after the moves. The promotion and the move are one closure; until the moves land, ten promoted specs carry a link to a directory that does not yet exist.

**A second consequence of the move.** Each delta spec's own links are written for the *active* change location — for example `../../../../specs/ai-contract-vocabulary/spec.md` from `openspec/changes/<name>/specs/<cap>/spec.md`. After the move to `archive/2026-08-01-<name>/` the correct depth is five, not four. Wave 0's archived deltas use archive-relative depths, so this is a real divergence, not a matter of taste. It affects the archived audit trail only, never the live specs.

---

## 7. doc 0002 amendments — all five, verified against the file

File: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md`. Amendments are appended or corrected in place; **no milestone or node identifier was renumbered**.

**1 — Shipped counter and the two false sentences (line 3, line 13).** Verified correct as the prior agent left it. Line 3 now reads *"Wave 0 + Wave 1 complete — **14 of 41** milestones shipped. **AI-00 through AI-13 are landed and verified.** The `backend/agent` module exists at `backend/agent/` with 19 production files and 27 test files."* The "Neither the `backend/agent` module nor a single line of Layer 1 exists on disk" sentence is gone. Line 13's authoring constraint now reads *"From Wave 1 forward, this document cites shipped Layer 1 code as evidence for completed milestones, never for unshipped ones."*

> **Recorded contradiction.** The launch brief for this run describes the transition as **4/41 → 14/41**. The verify report § 7.1 quotes the pre-amendment text verbatim as *"Not started — **0 of 41** milestones shipped"* and states the counter *"was already stale before Wave 1: Wave 0 was archived on `main` in `ead6ac8` and the counter was never moved."* The two accounts disagree about the pre-image. The verify report's is corroborated by the quoted text; the brief's describes the value the counter *should* have held after Wave 0. Recorded rather than resolved, because this run has no read of the pre-amendment file.

**2 — Layer 1 completion checklist (lines 2275–2292).** Verified correct. Items 1 and 2 checked (Wave 0). Items **3** (line 2277, neutral contracts documented and tested), **4** (line 2278, every content-part variant readable and sealed) and **5** (line 2279, cache breakpoints, options, rebuild) checked. Item **6** (line 2280, round-trip tokens byte-exact through the wire) left **open** — its wire half is AI-26.6 / AI-29.2, Wave 2+. Items 7–18 remain open.

**3 — Traceability spine (lines 2333–2345). Two rows corrected by this run.**

| Row | Line | State | This run |
| --- | --- | --- | --- |
| **C1** | 2333 | **Closed.** | verified |
| **C2** | 2334 | **Closed for Layer 1.** | verified |
| **G4** | 2337 | **Layer 1 half closed.** Remaining: wire rendering by AI-26.2 | **corrected** — the prior agent wrote a bare "**Closed.**", which contradicts the trailing Wave 2 clause and diverges from how G5 and G12(b) render the identical shape |
| **G5** | 2338 | **Layer 1 half closed.** | verified |
| **G9** | 2340 | **Closed.** | verified |
| **G12(b)** | 2342 | **Layer 1 half closed.** | verified |
| **G12(c)** | 2343 | **Closed.** | verified |
| leakage row 8 | 2345 | **Layer 1 half closed** by AI-10.2 | verified |
| leakage row 9 | 2345 | **Layer 1 half closed** by AI-11.3 (wire rendering AI-26.2, Wave 2) | **corrected** — the prior agent wrote "**Closed**", which claims more than AI-11.3 delivers |

**4 — Wave 2 obligation: region-level exhaustiveness guard (line 2346). Corrected.** The landed tripwire `request_test.go:1745` asserts `len(regions) != 11`. It catches a **deleted table row**; it does **not** catch a **field added to `requestDraft`**, which is precisely the failure that recurred across **AI-10.6, AI-11.1 and AI-12.3**. The prior agent's text named `AI-10.5`; the recurrence is `AI-10.6`.

**5 — Wave 2 obligation: nesting-depth cap on `isWellFormedJSON` (line 2347). Rewritten.** The row now records, in full: no depth cap at `json_syntax.go:55`; reachable from caller input through `tool_call.go:91`; **agrees** with `encoding/json.Valid` at depth 5 000, **disagrees** at 10 001, `fatal error: stack overflow` at 20 000 000 (unrecoverable — `recover()` cannot catch it); the differential generator caps nesting at ~5 at `json_syntax_differential_internal_test.go:76`; and **the same test's doc comment at line 28 claims coverage of "deeply nested structures", which is not true as written**. The false coverage claim is part of the obligation and must be corrected in the same change. Fix: cap nesting at 10 000, add a corpus case at 10 001, raise the generator's nesting bound above the cap, and correct the doc comment.

---

## 8. Lineage

| Artifact | Reference |
| --- | --- |
| Wave 1 verify report | Engram **#2323** · `openspec/changes/archive/WAVE-1-VERIFY.md` (after the move) |
| Wave 1 implementation record | Engram **#2322** |
| Prior, superseded archive report | Engram **#2330** — corrected by this document (§ 2); the Engram topic `sdd/cachicamas-ai-layer-1/wave-1-archive-report` now holds this report |
| Per-change archive notes | Engram `sdd/{change-name}/archive-report`, one per milestone AI-04 … AI-13 |
| Wave 0 precedent | `openspec/changes/archive/2026-07-31-cachicamas-{agent-module-scaffold,ai-contract-vocabulary,ai-stream-lifecycle,ai-minimum-capabilities}/` |
| Charter | `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 361–850 |

---

## 9. Risks carried out of this archive

1. **The folder moves and the commits are outstanding** (§ 6). Until they land, ten promoted specs carry an archive link that does not resolve, and ten completed changes sit in the active changes directory. This is the single largest open item.
2. **W1 is a real, reproduced defect** deferred to Wave 2, with an unrecoverable crash mode at extreme nesting depth. It is unreachable from untrusted input until AI-24; that is a scoping argument, not a fix.
3. **The region-exhaustiveness guard is one-directional** and cannot trip on the failure it exists to guard against. Three occurrences were each caught by a human noticing, not by a control.
4. **Intra-leaf TDD ordering is unauditable from git** for the whole wave — one commit per leaf. The verify report corroborated AI-12's transcripts against verbatim format strings and found nothing contradicting them, but the structural gap stands for all ten changes.
5. **`main` does not contain Wave 1.** Any reader who assumes otherwise from the doc 0002 counter — which now says 14 of 41 — will be wrong until PR #101 merges. The counter describes the milestone plan, not `main`.
