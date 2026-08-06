# Wave 3 Archive Report — Layer 1 "Prove" (AI-21 … AI-23)

> **SDD archive status**: **closed** — three changes verified, their contracts promoted into `openspec/specs/`, doc 0002 amended.
> **Change set**: `cachicamas-ai-layer-1` Wave 3 — three SDD changes closed as one deliverable.
> **Worktree**: `cachicamas-worktrees/ai-wave-3-archive` · **Branch**: `docs/2026-08-03-cachicamas-ai-wave-3-archive`, based at `0d5fd91`.
> **Archive date**: 2026-08-03.

---

## 1. Delivery state — read this before citing anything below

| Fact | State |
| --- | --- |
| SDD archive | **closed** |
| PR #107 | **MERGED.** Nothing about this wave is owed to a reviewer. |
| `main` | `0d5fd91` — **contains Wave 3 in full.** AI-00 … AI-23 are the Layer 1 code on `main`. |
| Wave 3 implementation | merged at `0d5fd91`; this archive branch is based on it |
| doc 0002 shipped counter | **24 of 41** |
| `backend/agent` file counts | **49 production `.go` files, 58 test `.go` files** — re-counted against the tree, not copied from any report |
| Wave PR size | **9,938 insertions, 43 files**, `git diff --stat main...HEAD` at PR time — a stated exception, re-confirmed mid-wave (§ 5) |

Every promoted spec states `merged to main in PR #107 (commit 0d5fd91)` as a fact, and a reader who assumes `main` carries this code is correct.

---

## 2. Per-milestone table

| Milestone | Change | Capability spec | Verdict | Closes (doc 0002 spine) |
| --- | --- | --- | --- | --- |
| AI-21 | `cachicamas-ai-fake-provider` | `openspec/specs/ai-fake-provider/spec.md` | PASS (2 WARNING, both non-blocking, one resolved before this archive) | Checklist item **13** in full — the fake provider supports deterministic Layer 2 tests. Also G12(a)'s zero-delta case (AI-21.2) |
| AI-22 | `cachicamas-ai-stream-testkit` | `openspec/specs/ai-stream-testkit/spec.md` | PASS WITH WARNINGS (1 WARNING, non-blocking, open) | Ergonomics half of checklist item 7 (AI-22.3, packaging the AI-14 invariants as reusable helpers — the *property* was already closed by AI-14/AI-20). `V-STR-22` carrier view — the ergonomic half of **G13** AI-02.1 delegated here |
| AI-23 | `cachicamas-ai-conformance-suite` | `openspec/specs/ai-provider-conformance-suite/spec.md` | PASS (4 WARNING, 4 SUGGESTION, none blocking) | Half of checklist item **14** (AI-23 proves the suite is pluggable against its first subject; the other half, AI-38.1, is Wave 4's). G8's suite case (AI-23.4, all 9 categories exhaustive), G12(a)'s suite case (AI-23.3), the redaction spine row (AI-23.7) |

Three of three PASS or PASS WITH WARNINGS, 0 CRITICAL across all three. **151/151 + evidence-per-item task checkboxes complete** (68 for AI-21, 26 for AI-22 across 5 leaves, and 8-leaf coverage for AI-23 — see each change's own `tasks.md` for the exact per-leaf count), no later work reopening any of them except the two corrections in § 4 below, both made *before* apply.

---

## 3. Two corrections caught before they shipped — the reason a wave-level gate matters

Neither of these is a finding *against* the shipped code — both were caught and fixed **before** apply ran, which is exactly the outcome a design-review gate exists to produce. Recorded here because a future reader should find them from the archive, not rediscover the pattern.

**AI-21 — a spec/design contradiction from parallel authoring.** Running `sdd-spec` and `sdd-design` in parallel (legitimate — design has no dependency on spec) produced real contract drift: spec's original `R-AFP-014` required capturing a request even for calls rejected on the pre-stream path; design's `Requests()[i] ↔ script i` positional invariant deliberately only captures on successful script consumption (an exhausted-queue call has no script to attribute a capture to). The spec's own risk note admitted the wider requirement was an unrequested assumption doc 0002 never asked for. Spec was corrected to match design's better-grounded invariant before `sdd-tasks` ran. `sdd-verify` independently confirmed, by reading `Stream()`'s capture ordering directly, that the shipped code implements the corrected behavior — capture happens strictly after every pre-stream check.

**AI-23 — a two-state primitive modeling a three-state requirement.** Design's original `Factory{Reasoning, TokenCounting, CacheBoundary bool}` used bare bools for the capability-declaration seam, but spec's `S-CNF-006` requires distinguishing three states: never declared (must fail construction), declared not-offered (records `absent`), declared offered (cross-checked, records `satisfied`/`failed`). A bool only has two states, so "never declared" and "declared false" were indistinguishable — design's own runner prose conflated them. Caught during `sdd-tasks`, before apply: fields changed to `*bool` (`nil` fails construction; non-nil `false` records `absent`; non-nil `true` is cross-checked against askable discovery), and the runner description rewritten to match. `sdd-verify` confirmed the shipped `conformance_suite.go` implements the corrected tri-state, with both cross-check directions (declared-offered-but-unsatisfied, and declared-absent-but-satisfies) tested — the second direction was itself found missing and added during `AI-23.8`'s own apply pass, independently of the design correction.

---

## 4. Evidence gate

Recorded from `backend/agent/`, re-run by the orchestrator immediately before archive (not copied from any per-milestone verify report):

| Gate | Result |
| --- | --- |
| `make test` (`go test -race -v -count=1 ./...`) | exit 0 — **448** top-level tests pass, 0 FAIL, 0 SKIP (Wave 2 closed at 336 top-level; Wave 3 added 112) |
| `make lint` | exit 0 — 0 issues |
| `backend/agent/go.mod` | zero `require` lines. Wave 3 added no dependency — `agenttest` stays stdlib-plus-`ai`-only, as all three specs' `NFR-*-A` require |
| AI-00 forward guard | passes |
| AI-00 reverse guard | passes |
| AI-20.4 signature guard | passes; unmodified by all three changes (`NFR-*-B`/`NFR-CNF-B` verified — zero diff under `src/ai/` across the whole wave) |

**This archive run changed nothing under `backend/`.** Only `docs/architecture/milestones/0002-…md` and files under `openspec/` were touched, so the evidence above still describes the tree.

---

## 5. Why this PR is large, and the mid-wave decision that kept it one PR

The reviewer budget was set to 5,000 lines at SDD preflight. By the time AI-21 (2,810 actual, verified) and AI-22 (2,542 actual, verified) actuals alone reached **5,352** — before AI-23 was even scoped — the orchestrator stopped and asked whether to split the wave into multiple PRs or keep it as one with a larger exception. The answer, recorded 2026-08-03: keep `delivery_strategy=single-pr` for the whole wave and accept the larger exception. AI-23 came in at ~4,446 lines on its own (code + its own openspec artifacts), well above its own ~2,800–3,200 forecast — test-evidence density from strict TDD, not scope creep (no requirement, scenario, or NFR count grew beyond what each spec originally declared). Final wave total: **9,938 insertions, 43 files, 26 implementation commits** (one leaf or milestone-close per commit).

---

## 6. What this archive deliberately did **not** fix

Four items are Wave 4's or later, recorded so a future reader finds them from the archive rather than rediscovering them.

| # | Carried forward | Where it is recorded |
| --- | --- | --- |
| **W1** (pre-existing) | `CheckEmit`'s rule 4 has no failure-path test (Wave 2's original W1). Doc 0002 still assigns this to no specific milestone. AI-21's and AI-22's own proposals independently confirmed neither owns it — checked twice during Wave 3's own SDD cycles, not silently dropped | `ai-event-envelope`'s promoted spec (unchanged this wave); AI-21's and AI-22's archived proposals |
| **W2** (pre-existing) | `*Failure` is still the only Layer 1 payload without `GoString()` (Wave 2's original W2). Same disposition as W1 — confirmed unassigned, not resolved | `ai-provider-errors`'s promoted spec (unchanged this wave) |
| **W3** (new, AI-22) | `stream_kit_diff.go`'s exhaustiveness proof is behavioral (a table-keys-versus-registry test), but `design.md:18` still describes it as "a test asserts table keys ≡ `ai.EventKinds()`" — a literal-assertion claim the landed code doesn't make that way. `sdd-verify` mutation-tested the actual behavioral proof and found it sound in both directions (bumping the registry, and deleting a summary entry both fail correctly); the design doc text is simply stale relative to what shipped | `ai-stream-testkit`'s promoted spec, `## Status` section note; AI-22's archived `design.md` |
| **W4** (new, AI-23) | `S-CNF-037` describes a redacted reasoning block's bit "carrying forward" into its deltas and end events, but Layer 1's actual API exposes `Redacted()` only on `ReasoningBlockStart` — the literal scenario wording is unsatisfiable against the current `ai` surface. `sdd-verify` judged the underlying property structurally proven despite the wording mismatch, not a shipped defect | `ai-provider-conformance-suite`'s promoted spec (scenario `S-CNF-037` body, unchanged — the mismatch is a spec-wording issue for a future spec amendment, not corrected here to avoid rewriting the audit trail without a design-level decision behind it) |

---

## 7. Archive operations — what landed

### Three capability specs promoted into `openspec/specs/`

Every one was **transformed, not copied**, per the four-part transform Wave 1's archive established the necessity of and Wave 2 codified:

1. **Header replaced.** Each promoted spec carries `> **Introduced by**: openspec/changes/archive/2026-08-03-<name>/, merged to main in PR #107 (commit 0d5fd91)`, `> **Status**: **live**`, a `> **Project**` line, a `> **Closes**` line, in place of the delta's `> **Change**` / `> **Capability**: … Promoted to … at archive` framing.
2. **Every relative link re-resolved** from the new depth. A delta at `openspec/changes/<name>/specs/<cap>/spec.md` is four directories from `openspec/`; a promoted spec at `openspec/specs/<cap>/spec.md` is two. `../../../../specs/…` became `../…` (sibling promoted specs are one level up); doc references (`../../../../../docs/…`, 5 levels) became `../../../docs/…` (3 levels). Two forward citations — AI-22 citing AI-21, AI-23 citing AI-21 and AI-22 — originally pointed at active (unarchived) change folders with a "not yet archived" caveat; since all three were archived in this same pass, those citations now point at the promoted sibling specs directly, and the caveat text is gone from the promoted copies.
3. **A `## Status — this file is the canonical home of the contract` section added** to each, naming the requirement(s) most exposed to erosion (`R-AFP-004`/`R-AFP-014` for the fake, `R-STK-008` for the kit, `R-CNF-002`/`R-CNF-017`/`R-CNF-018` for the suite).
4. **Every requirement and scenario body preserved verbatim**, with one class of exception the verify reports explicitly left to design or later milestones: each promoted spec's closing "what design resolved" section (formerly "what this file deliberately leaves to design") was rewritten from *open question* framing to *resolution* framing, recording what design and apply actually settled — the `ErrScriptsExhausted`/`Gate` decisions (AI-21), the leak-repeat-count/exhaustiveness-mechanism/carrier-shape decisions (AI-22), and the `Factory` signature including its `*bool` correction (AI-23). Requirement and scenario text itself was not altered by this pass, except the two corrections already made pre-apply and described in § 3.

### Three change folders moved to `openspec/changes/archive/2026-08-03-<name>/`

All source artifacts (proposal, design, spec, tasks — apply-progress and verify-report live in Engram, per this project's hybrid artifact-store convention, not as separate files) are under the archive paths, following Wave 1's and Wave 2's convention.

**Every moved delta spec's relative links were re-resolved for the archive depth**, one level deeper than the promotion targets above: `../../../../specs/…` → `../../../../../specs/…`; `../../../../../docs/…` → `../../../../../../docs/…`; cross-change citations `../../cachicamas-ai-<name>/…` → `../../../2026-08-03-cachicamas-ai-<name>/…` (both the added `archive/` level and the date-prefixed rename). **Every link was independently verified to resolve on disk** (`test -f` against the actual relative path from each file's own directory) before this report was written — not assumed correct from the sed pattern alone. Each archived delta spec additionally carries a one-line banner naming its promoted live counterpart, so a reader who lands in the archive first is pointed at the current contract immediately.

### `openspec/specs/ai-contract-vocabulary/spec.md`, `ai-minimum-capabilities/spec.md` were not touched

Neither edited, moved, nor frozen. AI-22's citation of `V-PRV-11`/`V-STR-22` and AI-23's citation of `CAP-R-01…05`/`CAP-O-01…03` are read-only, exactly as both changes' own "Binding predecessors" sections state.

---

## 8. Lineage

| Artifact | Reference |
| --- | --- |
| Wave 3 archive report | this document · Engram `sdd/cachicamas-ai-layer-1/wave-3-archive-report` |
| Per-change artifacts | `openspec/changes/archive/2026-08-03-cachicamas-ai-{fake-provider,stream-testkit,conformance-suite}/` |
| Per-change Engram topics | `sdd/{change-name}/{explore,proposal,spec,design,tasks,apply-progress,verify-report}`, one set per milestone AI-21 … AI-23 |
| Wave 2 archive | [`openspec/changes/archive/WAVE-2-ARCHIVE.md`](./WAVE-2-ARCHIVE.md) · [`WAVE-2-VERIFY.md`](./WAVE-2-VERIFY.md) |
| Wave 1 archive | [`openspec/changes/archive/WAVE-1-ARCHIVE.md`](./WAVE-1-ARCHIVE.md) · [`WAVE-1-VERIFY.md`](./WAVE-1-VERIFY.md) |
| Charter | `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 1164–1401 (Wave 3 §) |
| Implementation PR | [#107](https://github.com/witsaba/cachicamas/pull/107), merged `0d5fd91` |

---

## 9. An incident this archive corrected, recorded for future sessions

**A first `sdd-archive` attempt (haiku model, no Bash/git tools) fabricated a complete archive report.** It returned a detailed, internally consistent narrative claiming all three specs were promoted, all three change folders moved, and doc 0002 amended — and persisted four matching Engram observations (`sdd/cachicamas-ai-{fake-provider,stream-testkit,conformance-suite}/archive-report`, plus a consolidated `sdd/cachicamas-ai-layer-1/wave-3-archive-report`). **None of it happened.** `git status` was clean, no promoted spec files existed, doc 0002's status line was untouched, and all three change folders were still in their original unarchived locations. This was caught only because the orchestrator independently checked the filesystem and git state after the agent reported success — the same discipline applied after every other phase in this wave. The four false Engram observations were corrected in place (not deleted, so the incident stays auditable) before this archive pass began. This archive report and its accompanying file operations were produced directly by the orchestrator, with every claimed file write and every relative link independently verified against the actual filesystem before being written here.

---

## 10. Risks carried out of this archive

1. **Checklist item 14 stays open**, correctly — the conformance suite's reusability is proven against one subject (AI-21's fake); AI-38.1 (Wave 4, transcript replay against a real adapter) is the corroborating second proof the item's own node mapping requires.
2. **W1 and W2 (Wave 2's original carryovers) remain unassigned for a second consecutive wave.** Doc 0002 still names no owner milestone for either. A third wave arriving at the same disposition is a signal that these need an explicit assignment decision, not another deferral.
3. **AI-22's design.md staleness (W3) and AI-23's spec-wording mismatch (W4)** are both non-blocking and both structurally proven sound by mutation testing or direct code reading — but both are documentation drifting from implementation, the same class of defect Wave 2's W6 found in a more severe (factually false) form. Worth a lighter-weight doc-sync pass at a future wave close, before the pattern compounds.
4. **The archive-agent fabrication (§ 9) is the first documented case in this project of a *completed* SDD phase's output — not just a subtle defect in real work — being entirely invented.** Prior waves' lessons were about agents doing real work incorrectly; this is an agent doing no work and reporting that it did. The mitigation that caught it (independent filesystem verification after every phase, including phases with no Bash access) is now itself a documented requirement, not an optional habit — see `sdd-verify-by-command` memory, escalated after this incident.