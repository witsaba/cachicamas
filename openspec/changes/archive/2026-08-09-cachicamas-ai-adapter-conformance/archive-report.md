# Archive Report — `cachicamas-ai-adapter-conformance` (AI-38)

> **Change**: `cachicamas-ai-adapter-conformance` · **Milestone**: AI-38 — Run full deterministic adapter conformance (doc 0002, lines 2289–2327) · **Wave**: 6 — Hand off, **first milestone**
> **Phase**: archive (final) · **Status**: SDD cycle **CLOSED**
> **Date**: 2026-08-09 · **Branch**: `feat/ai-38-adapter-conformance` · **Base**: `origin/main@033baa67` · **Final HEAD**: `135b5105` (16 commits)
> **Project**: cachicamas (witsaba) · **Worktree**: `cachicamas-worktrees/ai-38-adapter-conformance`
> **Artifact store**: hybrid — OpenSpec files **and** Engram observations

---

## Delivery state — read this before citing anything below

| Fact | State |
| --- | --- |
| SDD cycle | **closed** — planned, implemented, verified PASS, specs promoted, change archived |
| Issue | **#139** |
| PR | **#140** — **OPEN, not merged.** https://github.com/witsaba/cachicamas/pull/140. The merge is owed by the maintainer, not by this archive. |
| `main` | `origin/main@033baa67` — **does not contain AI-38**. AI-00 through AI-37 plus AI-41 remain the newest Layer 1 code on `main`. |
| AI-38 branch head | `135b5105` (16 commits from `origin/main@033baa67`) |
| doc 0002 SDD-closed counter | **40 of 42** (39 landed-on-`main`, AI-38 SDD-closed pending merge) |

Every promoted spec touched by this change either carries an `Introduced by` line naming PR #140 as open (the new `ai-adapter-conformance-run/spec.md`) or an in-place `> **Amended 2026-08-09 (AI-38)**` blockquote naming the change and its archive path (the two amended files). When PR #140 merges, the honest edit is to append the merge commit to those lines; nothing else about the specs changes. This follows the exact precedent `WAVE-1-ARCHIVE.md` recorded for PR #101.

---

## Executive summary

AI-38 runs the AI-23 deterministic conformance suite's **unscoped** entry point, `agenttest.RunConformance`, against a real, shipped adapter — the OpenRouter bridge over the real `openaicompat` client — for the first time. Before this change, the suite's only unscoped subject was AI-21's in-process fake; the OpenRouter bridge passed only 2 of 5 required capabilities through the **scoped** `RunConformanceFor`, whose own doc states a scoped verdict "is never presentable as evidence of full conformance," with the remaining three capabilities `t.Skip`'d.

The RED baseline (`TestOpenRouterAdapter_FullConformance`, replacing the three `t.Skip` drivers) failed **five** case families — cancellation, terminal, `finish_reason`/refusal, `usage`/absent-vs-zero, and redaction — two more than the three skipped drivers suggested, confirming the proposal's own prediction that the real failure set was wider than what static reading of the skipped code could show. Each was resolved on a named side, with zero cases removed and zero waivers: two suite-side spec amendments (cancellation admits either a bare close or exactly one cancellation-category terminal, resolving a real conflict with the promoted `ai-provider-error-mapping` `R-AEM-014` typed-terminal obligation from AI-32.3; finish-reason reachability and mid-stream failure-category classification each narrowed to a **dialect-aware** absence/collapse, each paired with an anti-escape scenario) and one bridge-side rendering fix (in-band error frames for mid-stream failures, present-fields-only usage rendering). A byte-transparent recording helper with a drift guard replaced hand-typed SSE fixtures; a generated nine-entry capability record is asserted against a committed expectation with a hard block wired for a generated `CAP-O-01 = satisfied`; and a two-tier boundary-replay sweep proves the verdict survives adversarial stream fragmentation.

Verification took three rounds: R1 found 3 CRITICAL by defeat-testing spec obligations gates alone could not see; R2 found 0 CRITICAL but a completeness-gate FAIL (3 PARTIAL scenarios pending an honest scope narrowing); R3, after the WU11 narrowing and closing the last two gaps, reached **PASS — 14/14 requirements, 47/47 scenarios, 0 CRITICAL**, with `gentle-ai sdd-verify-validate` granting the verdict. Receipt-driven review (lineage `review-d5bd9a2a0cd4e4e0`) returned **APPROVED** with 0 severe findings.

---

## Final-state authority — what supersedes what

Per the archive skill's Final-State Authority hierarchy, this report describes the state **at close**. The following intermediate claims are superseded and are **not** echoed here as current facts:

| Intermediate claim | Source, and when it was true | State at close |
|---|---|---|
| `verify-report.md` **Round 1** verdict FAIL, 3 CRITICAL, 6/14 requirements, 37/47 scenarios | Round 1, HEAD `93ae4109`, report commit `ae5b7600` | **Resolved.** All 3 CRITICAL closed by the WU10 remediation round (commits `92acdf8b`, `ef3359f7`) — retry parity now mechanically enforced at all declaration sites, `NFR-ACR-A`/`S-ACR-020` reworded to the true byte-identity-to-base obligation, the scoped-run-is-never-evidence rule now has a structural guard. |
| `verify-report.md` **Round 2** verdict FAIL — completeness gate only, 0 CRITICAL, 11/14 requirements, 44/47 scenarios, 3 PARTIAL | Round 2, HEAD `ef3359f7`, report commit `6f5e3c45` | **Resolved.** The WU11 polish round (`f93b800a` fix, `a619dc69` docs) closed the third retry-declaration site, added the scoped-run citation guard, and honestly narrowed `R-ACR-003`/`R-OR-06` to "every transcript **with a neutral-event preimage**" — audited for honesty (bounded by a structural property, names the excluded fixture, adds a positive obligation) rather than accepted as a convenient weakening. |
| `tasks.md`'s own original Phase 8.1 text — "One `--- SKIP: TestOpenRouterAdapter_LiveSmoke`" | Original apply evidence, before WU10 | **Corrected within the same artifact** (WU10 task 9.6): the true count, at any subtest nesting level, is **26 `--- SKIP`**, all benign (25 nested declared-absent-optional-capability/retry-absent subtests plus the one genuinely top-level, AI-39-gated live-smoke skip). `verify-report.md` Round 3 carries the corrected figure (2841 PASS / 0 FAIL / 26 SKIP) as final. |
| Design D4's literal cancellation-admission text — "bare close OR exactly one terminal, optionally preceded ONLY by block-closing end events" | `design.md`, written before Phase 2/WU3's safety-net discovery | **Refined, recorded as a design refinement not a silent deviation** (`tasks.md` Phase 2 evidence): the admission shape now also tolerates an ordinary non-terminal prefix raced by Go `select`'s no-case-priority — proven necessary against both AI-21's fake and the real adapter. Every rejection guarantee (two terminals, wrong category, post-terminal event, invented Completion) is unchanged and independently self-tested. |
| `verify-report.md`'s WARN-B — `R-OR-05`'s "default-model swap" scenario names the capability-record test as the enforcement mechanism | Round 3, still open at report-close time, explicitly flagged for promotion-time correction | **Closed at this promotion.** `openspec/specs/ai-openrouter-first-provider/spec.md` `R-OR-05`'s scenario now names the actual mechanism — `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` plus the `openrouterDefaultModel` pin — and states explicitly that the capability-record test is model-insensitive. |

No contradiction above was left unrankable; every superseded claim is cited with its source and the commit or mechanism that resolved it. Explicit final-state facts supplied at launch (diff stat, review lineage, PR number, gate results) are consistent with `verify-report.md`'s own Round 3 recorded state — no conflict to record between the launch prompt and the artifact.

---

## Phase artifact inventory

| Artifact | Verdict | Observation | Archived file |
|---|---|---|---|
| **Explore** | evidence, 4 corrections recorded post-hoc | Engram **#2761** | `explore.md` |
| **Proposal** | scoped, 5 tension resolutions | Engram **#2762** | `proposal.md` |
| **Design** | 8 architecture decisions (D1–D8), 9 work units | Engram **#2765** | `design.md` |
| **Spec (3 deltas)** | full capability + 2 modified-in-place | Engram **#2764** | `specs/ai-adapter-conformance-run/spec.md`, `specs/ai-provider-conformance-suite/spec.md`, `specs/ai-openrouter-first-provider/spec.md` |
| **Tasks** | 25/25 original + 6/6 WU10 remediation, all `[x]` | Engram **#2768** | `tasks.md` |
| **Verify-report (3 rounds, in-place history)** | R1 FAIL → R2 FAIL (completeness) → **R3 PASS** | Engram **#2773** | `verify-report.md` |
| **Archive-report** | CLOSED | *(this document, saved after)* | this document |

**Task Completion Gate: PASSED.** All 25 original implementation tasks plus all 6 WU10 remediation sub-tasks were already `[x]` in the persisted `tasks.md` when this phase read it. No checkbox was reconciled by the archive phase, and no exceptional repair was performed.

**Native Review Receipt Gate: satisfied via explicit final-state facts supplied at launch.** This session's Engram search for `sdd/cachicamas-ai-adapter-conformance/review/{transaction,ledger,receipt,gate-context}` topics returned no observations — the structured review-gate status this skill's engram-mode instructions describe reading was not persisted to Engram for this change under that topic-key convention. The orchestrator's launch prompt instead supplied the operative facts directly, ranked as "explicit final-state facts in the orchestrator's launch prompt" per the Final-State Authority hierarchy: receipt-driven review lineage `review-d5bd9a2a0cd4e4e0` returned **APPROVED** (reliability lens, full 33-path inspection, 0 severe findings, 2 WARNING + 2 SUGGESTION recorded as non-blocking follow-ups — carried forward into `ai-adapter-conformance-run/spec.md`'s own "Out of scope" table as `R3-1`…`R3-4`), and pre-commit/pre-push/pre-pr gates all returned `allow`. This report treats those facts as the operative review evidence for archive purposes, attributed to their source rather than restated as independently re-verified.

---

## Verification verdict at close

**Round 3 (final)**: **PASS — 14/14 requirements, 47/47 scenarios, 0 CRITICAL, 0 blockers, 0 PARTIAL**, 3 WARNING (all deliberately scoped out except WARN-B, closed at this promotion), 2 SUGGESTION. `gentle-ai sdd-verify-validate --requirements 14 --scenarios 47` returned `{"valid":true,"verdict":"pass"}`, evidence_revision `sha256:63037b8f2fe481028ebcad2564188dddbdf6301a09171bdee96471e0e6f4f106`.

**10 of 10** guard-defeat probes across all three rounds bit as designed; zero guards were accepted on inspection alone. Round 3's own two new defeats: Defeat F/F′ (WARN-A re-probe — a real edit reintroducing a bare `false` literal at the third retry-declaration site, caught by two independent mechanisms, including a layering probe proving the direct parity self-test catches what a textual scan structurally cannot) and Defeat G (S-CNF-086 citation guard — a real edit replacing the citation phrase, caught by name, with two bite-proofs for the co-location half).

**Gates at final HEAD `135b5105`**, as recorded in `verify-report.md`'s own Round 3 gate execution (`a619dc69`, the round-3-verified commit; `135b5105` is the report-commit HEAD one commit later) — this archive session had no shell tool available to independently re-run them, so they are cited from the artifact, not re-executed:

| Gate | Result |
|---|---|
| `make test` (`go test -race -v ./...`) | exit 0 — **2841 `--- PASS`, 0 `--- FAIL`, 26 `--- SKIP`** (all benign — 25 nested declared-absent-optional-capability/retry-absent subtests, 1 genuinely top-level AI-39-gated live-smoke skip) |
| `make lint` | exit 0 — `0 issues.` |
| `make build` | exit 0 |
| Acceptance gate — `TestOpenRouterAdapter_FullConformance -race -v -count=1` | exit 0 — nine-entry record unchanged (`CAP-R-01…05 satisfied`, `CAP-O-01/02/03 absent`, `CAP-O-04 satisfied`, `Verdict()==VerdictPass`) |
| Determinism — `-count=3 -race` on conformance + agenttest packages | exit 0 |
| `git status --porcelain` | empty at every checkpoint across all three rounds |

### Why the "no defect vs. narrowed spec" distinction is trustworthy here

The three former PARTIAL scenarios were closed by narrowing spec text, which is the failure mode that most resembles silently weakening a requirement to match what shipped. `verify-report.md`'s own Round 3 explicitly audited each narrowing for honesty rather than convenience: each is bounded by a structural property (no `ai.Event` preimage exists for a hand-authored vendor wire-extension field, not "whatever the code currently does"), names the excluded fixture and exact vendor fields, adds a **new positive obligation** (the exclusion must be named in the recorder test) rather than only deleting one, preserves the excluded fixture's own dedicated coverage, and carries a `(Previously: …)` clause citing the report's own SUG-1. This archive step re-confirmed the same property while merging the delta into the two canonical specs (see below).

---

## Specification amendment summary

Three delta specs promoted at this archive step, using the four-part transformation (header rewrite to `Introduced by`/`Status: live`, cross-reference re-resolution for the canonical path depth, an added `## Status` canonical-home section, body otherwise unchanged for the new capability; in-place `> **Amended 2026-08-09 (AI-38)**` blockquotes for the two modified-in-place canonical files) — following the `WAVE-1-ARCHIVE.md` precedent this session read before promoting anything.

| Canonical spec | Action | Requirements | Scenarios | Notes |
|---|---|---|---|---|
| `openspec/specs/ai-adapter-conformance-run/spec.md` | **Created** | `R-ACR-001`…`R-ACR-006`, `NFR-ACR-A` (7 total) | `S-ACR-001`…`S-ACR-021` (21) | New capability, full four-part promotion. `Introduced by` line explicitly caveats PR #140 as open/not merged. |
| `openspec/specs/ai-provider-conformance-suite/spec.md` | `R-CNF-010`, `R-CNF-011`, `R-CNF-012`, `R-CNF-016` **MODIFIED** in place; `R-CNF-028` **ADDED** | 5 touched | `S-CNF-082`…`S-CNF-087` new (6); `S-CNF-024`, `S-CNF-026`…`028`, `S-CNF-029`…`031`, `S-CNF-042`…`046` restated (13) | Requirement count **twenty → twenty-one**; scenario count **eighty → eighty-six** — both counted directly from the file before and after this promotion, not estimated. Every restated scenario re-read and confirmed present after promotion — no landed proof lost. |
| `openspec/specs/ai-openrouter-first-provider/spec.md` | `R-OR-05`, `R-OR-06` **MODIFIED** in place | 2 touched | 1 new scenario (the `CAP-O-01=satisfied`-blocks scenario under `R-OR-05`); 6 restated (3 under `R-OR-05`, 3 under `R-OR-06`, counting the corrected default-model-swap scenario as restated-and-corrected, not new) | `R-OR-05`'s "default-model swap" scenario corrected at this promotion — see WARN-B closure below. |

**Totals**: 8 requirements touched (7 new — 6 `R-ACR-*` + `NFR-ACR-A` — + 1 added `R-CNF-028` + 6 modified in place across two files). 28 scenarios newly added or newly restated across the three files (21 new-capability + 6 suite-new + 1 openrouter-new). **Zero REMOVED, zero RENAMED requirements.**

**Verification against the "verbatim copy corrupted an earlier archive" precedent** (`WAVE-1-ARCHIVE.md` § 2). After each edit, this phase re-read the full file:
- `ai-provider-conformance-suite/spec.md`: confirmed `R-CNF-001` through `R-CNF-019`, `R-CNF-027`, `R-CNF-028` all present in order, with `NFR-CNF-A` through `NFR-CNF-F` untouched, and every scenario `S-CNF-001` through `S-CNF-087` present with no gap beyond the pre-existing, documented `S-CNF-068` numbering skip.
- `ai-openrouter-first-provider/spec.md`: confirmed `R-OR-01` through `R-OR-10` all present in order, `R-OR-07`/`R-OR-08` (AI-39's) and `R-OR-09`/`R-OR-10` unmodified, the Decisions table and Traceability section untouched.
- `ai-adapter-conformance-run/spec.md`: full body carried over unchanged from the delta except the header/Status transform, verified by direct comparison against the delta source read earlier in this session.

### `R-OR-05`'s WARN-B closed at this promotion

`verify-report.md`'s WARN-B named `R-OR-05`'s "default-model swap" scenario as stating the wrong enforcement mechanism — it claimed "the capability-record test fails," but that test is model-insensitive (`factory.Reasoning` is hard-coded `false`). This promotion corrects the scenario to name the actual mechanism: `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment`, pinned against the `openrouterDefaultModel` constant, with an explicit clause stating the capability-record test's own insensitivity to the swap. This was the last spec-text inaccuracy `verify-report.md` carried into promotion; it is now closed.

### A pre-existing defect this phase did NOT fix, deliberately

`openspec/specs/ai-openrouter-first-provider/spec.md`'s own header still reads `> **Status**: DRAFT`, `> **Change**: add-openrouter-first-provider`, and relative links written for the active-change path depth — it never went through the four-part promotion transform at all, unlike every other canonical spec this session touched. This change's own `explore.md` (correction #4, confirmed independently this session by reading the live file) traces this to an **orphan post-archive revision** of `add-openrouter-first-provider`: the change was archived on 2026-08-06, but a later, separate revision re-created the active change-folder carrying only a revised `R-OR-07` (workflow-dispatch gating), which the promoted `openspec/specs/` copy does not reflect — the repository has no `.github/` directory at all. This change's own proposal explicitly fences `R-OR-07`/`R-OR-08` and the orphan directory reconciliation out of scope, naming it **AI-39's**. This phase applied its two MODIFIED requirements (`R-OR-05`, `R-OR-06`) with correct content but did not attempt the header/status rewrite, since that is a substantive edit outside this milestone's charter — following the exact precedent AI-37's own archive report set for the structurally identical situation in `ai-provider-client/spec.md`'s SD-2 defect. Recorded here so a future reader does not have to rediscover it.

### A pre-existing stale count this phase did NOT fix, deliberately

`ai-provider-conformance-suite/spec.md`'s own "Acceptance criteria" item 10 still reads "The record carries exactly eight entries" — stale since AI-35 grew the record to nine entries (`R-CNF-017`, amended 2026-08-07) and never corrected in that section. This predates AI-38 and is outside the four requirements this change modified; recorded here, not fixed, following the same "don't silently sweep in an unrelated correction" discipline as the two items above.

---

## Doc 0002 amendment

`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md`, amended in two places, following the AI-33…AI-37 blockquote pattern exactly — no existing amendment was rewritten:

1. **Top `> **Status:**` line** — SDD-closed counter `39 of 42` → **`40 of 42`**; explicit new clause distinguishing "landed and verified on `main`" (unchanged at AI-00 through AI-37 plus AI-41) from AI-38's own state (SDD-closed, PR #140 open and not yet merged); the "Remaining" clause corrected from "Wave 6 — Hand off (AI-38..AI-40), 3 milestones" to **"Wave 6 — Hand off — AI-39 and AI-40 not yet started, 2 milestones — plus AI-38's own merge to `main`, owed by the maintainer, not by this archive."**
2. **New dated blockquote** — `> **Amended 2026-08-09 — AI-38 close (Wave 6 — Hand off, first milestone).**`, appended immediately after the AI-37 close blockquote and before the `[!IMPORTANT]` authoring-constraint block, following the established pattern. It records: the PR-open caveat and the `WAVE-1-ARCHIVE.md` precedent it follows; what AI-38 delivered (unscoped run, RED baseline's five-family failure set, the two suite-side and one bridge-side resolutions, the recording helper, the generated capability record and its AI-29 hard-block, the boundary sweep, retry-parity enforcement); the three-round verify trajectory and the 10/10 guard-defeat record; the receipt-driven-review APPROVED verdict and its lineage; the 16-commit / 33-file / +4312/-705 final diff and the pre-approved `size:exception`; the four maintainer decisions (Engram #2763); four acknowledged deviations plus one reconciliation, all traced to `tasks.md`'s own evidence; the spec-amendment summary with exact before/after counts; an explicit note that **production/test file-count reconciliation was not performed** at this close (no shell tool available in this archive session) — left as AI-37's own last fresh measurement rather than silently repeated as current, following this document's own established pattern of flagging an unreconciled figure instead of guessing one; the gate results cited from `verify-report.md`'s own Round 3 execution; and that AI-39/AI-40 are now unblocked on the SDD-plan axis, with an explicit note that AI-39's own scope includes `R-OR-07`/`R-OR-08` and the orphan `add-openrouter-first-provider` reconciliation.

**A deliberate gap, stated honestly rather than filled with a guess.** Every prior Wave 5 close amendment (AI-33 through AI-37) states an orchestrator-`find`-measured production/test `.go` file-count delta against the base commit. This archive session's tool set has no shell access (Read/Write/Edit/Glob only), so it cannot run `find` against both `origin/main@033baa67` and final HEAD `135b5105` to state an honest number. Rather than repeat AI-37's `93`/`162` figures as if they were still current — design's own Threat Matrix records AI-38 as adding a new non-test package (`openaicompat/conformancetest/`), so the true figures have almost certainly moved — this amendment explicitly flags the gap and defers the reconciliation to the next milestone's close with shell access, following this document's own established pattern (AI-36's self-flagged 86/152 estimate, corrected at AI-37's close) for exactly this situation.

---

## Change folder archive

**Copied from**: `openspec/changes/cachicamas-ai-adapter-conformance/`
**Copied to**: `openspec/changes/archive/2026-08-09-cachicamas-ai-adapter-conformance/`

This phase had **no Bash tool**, so the move was performed as Read-then-Write of every file, matching the AI-36/AI-37 archive precedent. **All 8 files are copied by this archive step; the folder is not half-moved.** The orchestrator must delete the 8 source paths listed below (and the now-empty directory tree, including the `specs/` subdirectory) to complete the move.

| Archived file | Treatment |
|---|---|
| `explore.md` | Verbatim |
| `proposal.md` | Verbatim |
| `design.md` | Verbatim |
| `tasks.md` | Verbatim |
| `verify-report.md` | Verbatim (the admitted, PASS-granting Round 3 bytes untouched; full 3-round history preserved in place, as the file itself already carries) |
| `specs/ai-adapter-conformance-run/spec.md` | Verbatim delta (this is also the source the canonical file was promoted from) |
| `specs/ai-provider-conformance-suite/spec.md` | Verbatim delta |
| `specs/ai-openrouter-first-provider/spec.md` | Verbatim delta |
| `archive-report.md` | This document |

**Old paths the orchestrator must delete** (all under `openspec/changes/cachicamas-ai-adapter-conformance/`):
```
openspec/changes/cachicamas-ai-adapter-conformance/explore.md
openspec/changes/cachicamas-ai-adapter-conformance/proposal.md
openspec/changes/cachicamas-ai-adapter-conformance/design.md
openspec/changes/cachicamas-ai-adapter-conformance/tasks.md
openspec/changes/cachicamas-ai-adapter-conformance/verify-report.md
openspec/changes/cachicamas-ai-adapter-conformance/specs/ai-adapter-conformance-run/spec.md
openspec/changes/cachicamas-ai-adapter-conformance/specs/ai-provider-conformance-suite/spec.md
openspec/changes/cachicamas-ai-adapter-conformance/specs/ai-openrouter-first-provider/spec.md
```
After deletion, the empty `openspec/changes/cachicamas-ai-adapter-conformance/` directory tree (including the now-empty `specs/` subdirectory) should also be removed.

**Why the delta links inside the archived `specs/*/spec.md` files were left at their active-change depth.** Following the exact precedent the AI-36/AI-37 archives recorded for the identical situation: a delta spec's relative links are written for the *active* location `openspec/changes/<name>/specs/<cap>/spec.md`. After the move to `archive/2026-08-09-<name>/…` the correct depth would be one level deeper. This phase did not re-resolve those links inside the archived delta copies — this affects only the archived audit trail, never the live specs (which this phase promoted separately, with correct canonical-depth links, into `openspec/specs/`).

---

## Final diff and gates, at close

**Final diff** (`git diff --stat origin/main...HEAD` at final HEAD `135b5105`, per the orchestrator's explicit launch-time facts — this archive session had no Bash tool to independently re-run it): **33 files changed, 4312 insertions(+), 705 deletions(-)**. Commit count: **16**.

**Gates**, cited from `verify-report.md`'s own Round 3 recorded execution at `a619dc69` (the round-3-verified commit; `135b5105` is one further report-only commit on top) — not independently re-run by this archive session (no Bash tool): `make test` exit 0 (2841 PASS / 0 FAIL / 26 SKIP, under `-race`); `make lint` exit 0 (`0 issues.`); `make build` exit 0; `git status --porcelain` empty. No commit on this branch carries `Co-Authored-By` or any other AI attribution (per repository convention; not independently re-verified across all 16 commits by this archive session, again for lack of Bash access — recorded as a gap, not silently assumed).

---

## Open items at close — none blocking, all recorded by name and owner

Four follow-ups, recorded in `ai-adapter-conformance-run/spec.md`'s own "Out of scope" table at this promotion step, none of which blocks any requirement or scenario above:

1. **R3-1** — the post-cancel-close event bound is not tightened beyond what `R-CNF-011`/`R-CNF-012` already admit for a specific class of late-arriving frames. Owner: a later round.
2. **R3-2** — the retry-declaration-parity guard's third call site relies on a raw-bytes literal scan (import-cycle-blocked from the shared source); a negative-literal-mutation scan of that specific site was not independently re-verified beyond the two guards already defeated in Round 3. Owner: a later round.
3. **R3-3** — `DialectConstraints.UnreachableFinishReasons` is asserted as a superset check, not strict subset-enforcement against the closed seven-value vocabulary. Owner: a later round.
4. **R3-4** — the retry-literal pattern guard (`retryOfferedLiteralPattern`) uses `FindSubmatch` (first match only) rather than `FindAllSubmatch` with an all-must-agree rule, narrowing its reach at the one call site where the import cycle leaves it as the sole mechanism. Owner: a later round.

Two carried WARNINGs from `verify-report.md`, deliberately scoped out and unchanged by this promotion: **WARN-C** (record-level boundary-sweep sampling was extrapolated, not measured — design D8 mandates sampling unconditionally, so following design was correct) and **WARN-D** (`design.md` lists a stale fixture path; `design.md` is not promoted, so cosmetic). Two carried SUGGESTIONs, unchanged: **SUG-2** (`TestAI33_1_RaceCancelMidDo`, a pre-existing timing flake untouched by this branch, never fired in any of three rounds) and **SUG-3** (`retryOfferedLiteralPattern`'s first-match-only scan, materially reduced in severity by Round 3's direct parity self-tests at two of three sites).

One pre-existing repository defect, not introduced and not repaired by this change (see § "A pre-existing defect this phase did NOT fix, deliberately" above): `openspec/specs/ai-openrouter-first-provider/spec.md`'s own header never went through the four-part promotion transform, tracing to an orphan post-archive revision of `add-openrouter-first-provider` that AI-39's own scope names as its own to reconcile.

One deliberate reporting gap: production/test file-count reconciliation was not performed at this close, for lack of shell tooling in this archive session — see the doc 0002 amendment section above.

---

## Recommended next step

**No new SDD change is required for AI-38 itself.** The implementation is complete, verified PASS, and specs are promoted.

**Immediate**: the maintainer merges PR #140. Until then, `main` does not contain AI-38, and any reader who assumes otherwise from the doc 0002 counter — which now says 40 of 42 — will be wrong about what `main` carries, though right about what the SDD plan has closed. The counter describes the milestone plan, not `main`, per the exact caveat `WAVE-1-ARCHIVE.md` recorded for the identical situation.

**Next milestones: AI-39 (opt-in live smoke) and AI-40 (Layer 2 readiness handoff), now unblocked on the SDD-plan axis.** AI-38's charter declares `Blocks: AI-39, AI-40`; both now have a real, generated conformance artifact to compare against or certify. AI-39's own scope explicitly includes reconciling `R-OR-07`/`R-OR-08` and the orphan `add-openrouter-first-provider` change-folder — named as AI-39's, not AI-38's, by this change's own proposal.

---

## Lineage

| Artifact | Reference |
|---|---|
| Phase observations | Engram `sdd/cachicamas-ai-adapter-conformance/{explore,proposal,spec,design,tasks,verify-report}` — ids **#2761, #2762, #2764, #2765, #2768, #2773** |
| Maintainer decisions | Engram `#2763` (`sdd/cachicamas-ai-adapter-conformance/decisions`) |
| This report | `sdd/cachicamas-ai-adapter-conformance/archive-report` (Engram id assigned on save) + this file |
| Immediate predecessor | `openspec/changes/archive/2026-08-08-cachicamas-ai-observability/` (AI-37 — the milestone that unblocked this one) |
| Promotion-transform precedent | `openspec/changes/archive/WAVE-1-ARCHIVE.md` § 2 (the four-part transform, and the verbatim-copy corruption it documents) and its § 1/9 (the open-PR counter caveat this report follows for PR #140) |
| Charter | `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 2289–2327 |
| Delivery | Issue #139 · PR #140 — https://github.com/witsaba/cachicamas/pull/140 (**OPEN, not merged**) |
