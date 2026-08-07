# Archive Report — `cachicamas-ai-wave2-carryovers` (AI-41)

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002 § 2233–2257) — Discharge the Wave-2 carryovers · **Wave**: 5 — Harden
> **Phase**: archive (final) · **Status**: CLOSED — fully committed, all gates green, working tree clean, Judgment Day APPROVED (see § "Execution history: the tooling gap and its resolution" and § "Judgment Day Review")
> **Date**: 2026-08-07 · **Branch**: `feat/ai-41-wave2-carryovers` · **Final HEAD**: `c833a93` (five commits: `ca2ede7`, `b82cfdf`, `b271637`, `3adbb8c`, `c833a93`)
> **Project**: cachicamas (witsaba)

---

## Executive Summary

AI-41 (Discharge the Wave-2 carryovers) is complete and fully closed: both requirements (`R-AEE-021`, `R-AIP-016`) are proven in code, `sdd-verify` returned PASS WITH WARNINGS (2/2 requirements, 4/4 scenarios, 0 CRITICAL), and verify's one recommended fix (W-1, narrowing the `S-AIP-008` exemption) was applied in commit `b271637`. The branch carries **five commits** (`ca2ede7`, `b82cfdf`, `b271637`, `3adbb8c`, `c833a93`): the first three are the apply-phase work, `3adbb8c` is the archive commit (change-folder move, spec promotions and doc 0002 amendment landed together), and `c833a93` resolves a subsequent Judgment Day review round. The final Go-code diff is **4 files, +257/−12 = 269 lines** against `origin/main@f2e460d` (production code is `provider_failure.go` alone, **+27 lines, 0 deletions**); the whole branch, including specs, doc 0002 and the archived SDD artifacts, is **18 files, +1769/−14**. All three delta specs have been promoted into their canonical specs, the `ai-stream-testkit` carryover ledger now reads discharged, and `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` has been amended with a new dated close blockquote (shipped counter 37 of 42; landed range stated non-contiguously as "AI-00 through AI-35 plus AI-41"). The working tree is clean (`git status --porcelain` empty) and `make test` / `make lint` / `make build` were all independently re-run by the orchestrator at final HEAD `c833a93` and passed.

**This archive phase originally ran with no Bash/shell tool available.** All file-content work (spec promotions, the first pass of the doc 0002 amendment, copying every change-folder artifact into the dated archive directory, and the first draft of this report) was completed with Read/Write/Edit/Glob only. The three actions that require a shell — moving (`git mv`) the change folder out of the *active* `openspec/changes/` tree, committing everything, and running `make test`/`make lint`/`make build` from `backend/agent/` — could not be executed by that phase. **That gap has since been closed in full**: a shell-capable execution completed the folder move and committed everything as `3adbb8c`, the orchestrator independently re-ran all three gates and confirmed a clean tree, a Judgment Day review round followed and its confirmed findings were fixed in `c833a93`, and the orchestrator re-ran the gates again and independently reproduced both non-vacuity proofs. See § "Execution history: the tooling gap and its resolution" below for the full sequence.

---

## Execution history: the tooling gap and its resolution

The launch prompt for the initial archive phase specified duties requiring `git mv`, `git commit`, and `make test`/`make lint`/`make build` from a shell. The tool surface actually available to that execution was: `Read`, `Edit`, `Write`, `Glob`, and the Engram/codegraph MCP tools — no Bash, no shell, no file-delete, no file-move primitive. This section records what that phase could and could not do, and how the gap it left open was subsequently closed.

**Completed with the available tools, at the initial phase:**

1. **Spec promotion** (full inline promotion, per the AI-33/AI-35 precedent) — done via `Edit`:
   - `openspec/specs/ai-event-envelope/spec.md` — `R-AEE-021` (`S-AEE-071`, `S-AEE-072`) inserted after `R-AEE-020`, before "Non-functional requirements"; the "Carried forward" section (`:322–324`) got an append-only discharge blockquote after the existing paragraph, preserved byte-for-byte.
   - `openspec/specs/ai-provider-errors/spec.md` — `R-AIP-016` (`S-AIP-056`, `S-AIP-057`) inserted after `R-AIP-015`, before "Non-functional requirements".
   - `openspec/specs/ai-stream-testkit/spec.md:39` — the carryover ledger line amended append-only to read **discharged by AI-41**, naming both `W1` and `W2`, both requirement identifiers, and this change name; the pre-existing sentences are unchanged.
2. **Doc 0002 amendment** — done via `Edit`: top `> **Status:**` line updated (36→37 of 42; landed range restated as "AI-00 through AI-35 plus AI-41" with an explanatory sentence on the non-contiguity; "Remaining" trimmed to AI-36, AI-37); a new dated `> **Amended 2026-08-07 — AI-41 close…**` blockquote appended after the AI-35 close blockquote. The initial phase's draft of this blockquote recorded both subnodes, the three apply-phase commits, and the then-current +214/−12 diff; the blockquote was **updated again after the Judgment Day round** to its final form — five commits, the +257/−12 = 269-line Go-code diff, the Judgment Day summary, and the AI-36 value-form follow-up — alongside gate evidence, verify verdict, spec promotions, W-1 resolved, W-2 recorded, and the AI-36-unblocked note.
3. **Change-folder copy** — done via `Read` (of every source file) + `Write` (of an identical copy at the archive path): `explore.md`, `proposal.md`, `design.md`, `tasks.md` (with Phase 4's `A.1`–`A.5` checkboxes marked `[x]` and annotated with what was actually done), `apply-progress.md`, `verify-report.md` (with an appended archive note recording W-1's resolution), and all three `specs/*/spec.md` delta files, all written verbatim (or, for `tasks.md`/`verify-report.md`, verbatim plus an explicitly labeled archive-time annotation) into `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/`.
4. **This report** — written to the same archive directory and (below) persisted to Engram.

**Not completed by that initial phase — required shell access:**

1. **`git mv` / removal of the original folder.**
2. **`git commit`** of the spec files, the doc 0002 amendment, and the new archive directory.
3. **Final gate re-run** (`make test`, `make lint`, `make build` from `backend/agent/`) and **`git status --porcelain`**.

**Closed by a subsequent shell-capable execution:**

1. **`git mv` / removal of the original folder — done.** `openspec/changes/cachicamas-ai-wave2-carryovers/` no longer exists at its original path; the archive directory at `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/` is the sole surviving copy, and its `tasks.md`/`verify-report.md` (carrying the archive-time annotations) are the ones committed.
2. **`git commit` — done.** Everything described in this report is committed. The spec promotions, the doc 0002 amendment, and the archive directory landed together in `3adbb8c` (`docs(agent): archive AI-41 and promote the Wave-2 carryover specs`); a subsequent Judgment Day correction round landed in `c833a93` (`fix(ai): resolve AI-41 judgment-day findings on redaction scope and rule-4 proof`). `git status --porcelain` returns empty — the working tree is clean.
3. **Final gate re-run — done, independently, at final HEAD.** The orchestrator re-ran `make test`, `make lint`, `make build`, and `git status --porcelain` directly against final HEAD `c833a93` — not sourced from any phase agent's report: `make test` — 0 `--- FAIL` across all 7 packages under `-race`; `make lint` — `0 issues.`; `make build` — exit 0; `git status --porcelain` — empty; `go.mod`/`go.sum` — byte-identical to base, untouched.

**Closing state**: this change is fully closed on disk — committed, gated green, Judgment Day approved, and the working tree clean. No further follow-up action is required for AI-41 itself.

---

## Phase Artifact Inventory

| Artifact | Status | Observation ID | Location |
| --- | --- | --- | --- |
| **Explore** | PASS | #2662 | Engram `sdd/cachicamas-ai-wave2-carryovers/explore` |
| **Proposal** | PASS | #2663 | Engram `sdd/cachicamas-ai-wave2-carryovers/proposal` |
| **Spec (delta)** | PASS | #2665 | Engram `sdd/cachicamas-ai-wave2-carryovers/spec` |
| **Design** | PASS | #2666 | Engram `sdd/cachicamas-ai-wave2-carryovers/design` |
| **Tasks** | PASS | #2668 | Engram `sdd/cachicamas-ai-wave2-carryovers/tasks` |
| **Apply-progress** | PASS (19/19 apply-phase tasks) | #2670 | Engram `sdd/cachicamas-ai-wave2-carryovers/apply-progress` |
| **Verify-report** | PASS WITH WARNINGS | #2674 | Engram `sdd/cachicamas-ai-wave2-carryovers/verify-report` |
| **Archive-report** | CLOSED | (this document) | Engram `sdd/cachicamas-ai-wave2-carryovers/archive-report` + `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/archive-report.md` |
| **Judgment Day** | APPROVED — 0 SEVERE | (see § "Judgment Day Review") | Two blind judges (`jd-judge-a`, `jd-judge-b`) against frozen HEAD `3adbb8c` |

---

## Verification Verdict (final state)

**PASS WITH WARNINGS — 2/2 requirements, 4/4 scenarios COMPLIANT, 0 CRITICAL, 1 WARNING outstanding (W-2, process gap), 1 WARNING RESOLVED (W-1)**

Per `verify-report` (Engram #2674), snapshotted at HEAD `b82cfdf`:
- **Requirements verified**: `R-AEE-021` (emission-boundary payload-validation rule, proven directly), `R-AIP-016` (provider-failure payload redaction total over the verb surface)
- **Scenarios**: `S-AEE-071`, `S-AEE-072`, `S-AIP-056`, `S-AIP-057` — all COMPLIANT
- **W-1** — the `S-AIP-008` guard exemption (`{Error, String, GoString}`) was measurably broader than the one real collision (`{GoString}`). Verify recommended narrowing it. **RESOLVED, not outstanding**: this was applied in commit `b271637`, verified by planting `func (f *Failure) String() string { return "finish reason: stop" }` and confirming it passes the broad exemption silently but fails the narrow one, naming the shared non-diagnostic accessor. This report records W-1 as resolved in `b271637` — the stale "recommended, not required" framing in the verify-report snapshot is superseded here, not echoed as current.
- **W-2** — the design phase's blast-radius method (a literal grep for `%#v`/`GoString`) cannot see a reflection-based guard such as `S-AIP-008` turned out to be. This is **recorded as a process gap for future milestones** (grep also for `reflect.TypeOf(<Type>` / `NumMethod` whenever a change adds an exported method to a type covered by a method-set guard) — it is not a defect in this change and needs no code fix here.

---

## Implementation Commits (final state)

| # | SHA | Subject | Purpose |
| --- | --- | --- | --- |
| 1 | `ca2ede7` | `feat(ai): prove CheckEmit rule 4 surfaces the payload's own violation` | AI-41.1 — `export_test.go`, `event_test.go`; additive `rejectWith *Violation` on `WitnessPayload` + rule-4 failure-path test |
| 2 | `b82cfdf` | `feat(ai): redact *Failure's Go-syntax formatting via GoString` | AI-41.2 — `provider_failure.go`, `provider_failure_test.go`; `GoString()` delegating to `Error()` + adversarial planted-canary test; also narrowed `S-AIP-008`'s exemption to 3 names as a first-pass fix |
| 3 | `b271637` | `test(ai): narrow the S-AIP-008 exemption to the one shared name` | Applies verify's W-1 recommendation: narrows the `S-AIP-008` exemption from `{Error, String, GoString}` to `{GoString}` |
| 4 | `3adbb8c` | `docs(agent): archive AI-41 and promote the Wave-2 carryover specs` | Archive phase: promotes the three delta specs into their canonical specs, amends doc 0002 with the AI-41 close blockquote, moves the change folder into the dated archive path, and commits the whole archive |
| 5 | `c833a93` | `fix(ai): resolve AI-41 judgment-day findings on redaction scope and rule-4 proof` | Judgment Day corrections: fixes four both-judge findings (corrects the `GoString` doc comment's stated scope, threads the witness's `at Path` argument so rule 4 is actually proven, replaces a tautological position assertion, removes an unreachable `defer recover()`) plus one single-judge `%q`-verb gap |

**Aggregate final diff vs `origin/main@f2e460d`**: Go code is **4 files, +257/−12 = 269 changed lines** — `event_test.go` +67/−0, `export_test.go` +49/−6, `provider_failure.go` +27/−0, `provider_failure_test.go` +114/−6 — independently re-measured by the orchestrator via `git diff --numstat`, not sourced from a prior phase's report. Production code is `provider_failure.go` alone: **+27 lines, 0 deletions**. The whole branch, including specs, doc 0002 and the archived SDD artifacts, is **18 files, +1769/−14**. Against the session's 1000-line review budget, the Go-code diff (269 lines) is comfortably inside it; no `size:exception` needed. `go.mod`/`go.sum` are byte-identical to base, untouched. The Go-code figure is well over the ~90–125-line apply-phase forecast in `tasks.md`; the overage is concentrated in test-evidence density (the D-4 canary loop, the six-assertion rule-4 test, the S-AIP-008 fix/narrowing, and the Judgment Day correction round), not in production code.

**Not pushed. No PR opened**, per the change's `single-pr` strategy and the orchestrator's explicit instruction not to push/PR/merge from this phase.

---

## Specification Amendment Summary

Three canonical specs amended at archive time, all via full inline promotion, matching the AI-33/AI-35 house style, and all honoring the no-Go-identifier rule (behaviors and contract-level language only — no Go type/method/field names in canonical spec prose).

### `openspec/specs/ai-event-envelope/spec.md`

- **`R-AEE-021`** (with `S-AEE-071`, `S-AEE-072`) inserted verbatim, immediately after `R-AEE-020`'s scenarios and before the "Non-functional requirements" heading.
- **"Carried forward" section** (`:322–324` before this edit): the existing paragraph (recording why `W1` was left open) preserved byte-for-byte; a new blockquote appended immediately after it: *"Discharged 2026-08-07 (AI-41) by `cachicamas-ai-wave2-carryovers`… This gap is no longer carried forward."*

### `openspec/specs/ai-provider-errors/spec.md`

- **`R-AIP-016`** (with `S-AIP-056`, `S-AIP-057`) inserted verbatim, immediately after `R-AIP-015`'s scenarios and before the "Non-functional requirements" heading. No new NFR added — `R-AIP-016`'s totality clause explicitly cites `NFR-AIP-B`, restated in place rather than duplicated (per the delta's own "Why no `NFR-AIP-F`" rationale).

### `openspec/specs/ai-stream-testkit/spec.md`

- **Line 39** (the carryover ledger bullet) amended append-only. Pre-existing sentences ("W1/W2, the two Wave-2 carryovers AI-21 parked… Assigned 2026-08-03 (AI-24)… AI-22 still assigns neither.") preserved byte-for-byte. Appended: *"**Discharged 2026-08-07 (AI-41)** by `cachicamas-ai-wave2-carryovers`: both are closed — the emission boundary's payload-validation rule now carries a direct, attributable failure-path proof (`R-AEE-021`), and the provider-failure payload's redaction is a property of the payload rather than of the caller's formatting verb (`R-AIP-016`). This entry now reads **discharged by AI-41**, not *assigned to AI-41*."* This satisfies the milestone's charter Acceptance line.

---

## Doc 0002 Amendment

`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` amended:

1. **Top `> **Status:**` line** — shipped counter `36 of 42` → `37 of 42`. Landed range restated as **"AI-00 through AI-35 plus AI-41 are landed and verified"** (non-contiguous — see next section for the exact wording used and why). "**Remaining:**" trimmed from `Wave 5 — Harden (AI-36, AI-37, AI-41) and Wave 6 — Hand off (AI-38..AI-40), 6 milestones` to `Wave 5 — Harden (AI-36, AI-37) and Wave 6 — Hand off (AI-38..AI-40), 5 milestones`.
2. **New dated blockquote** — `> **Amended 2026-08-07 — AI-41 close (Wave 5 — Harden, Wave-2 carryovers).**` inserted immediately after the AI-35 close blockquote (before the `> [!IMPORTANT]` authoring-constraint block), following the exact pattern of the AI-33/AI-34/AI-35 close blockquotes. It records: both subnodes (AI-41.1 → `R-AEE-021`, AI-41.2 → `R-AIP-016`); no new test file (test additions extend existing topical files `event_test.go` and `provider_failure_test.go`, plus `export_test.go` for the new constructor — test file count stays at 148, production file count stays at 85); all **five** commits (`ca2ede7`, `b82cfdf`, `b271637`, `3adbb8c`, `c833a93`); the final Go-code diff of **+257/−12 = 269 lines** across the same 4 files, production at **+27/0** in `provider_failure.go`; the spec promotions; W-1 resolved in `b271637` (with the planted-`String()` proof cited); W-2 recorded as a process gap, not a defect; the **Judgment Day review round** — two blind judges against frozen HEAD `3adbb8c`, 0 SEVERE from either, four both-judge findings plus one single-judge `%q`-verb gap fixed in `c833a93`; the **AI-36 follow-up** — the pointer-receiver scope of `GoString`/`Error`/`Unwrap`/`Is` is not fully closed by this milestone and the value-form `%#v` sweep is recorded as owned by AI-36; the Engram observation IDs for backfill; and that AI-36 is now unblocked (this milestone's charter declares `Blocks: AI-36`, and AI-36's adversarial redaction sweep can now run against the provider-failure payload's Go-syntax rendering, which did not exist before this close). "**Remaining in Wave 5:**" in the new blockquote reads `AI-36 (secret redaction), AI-37 (observability)`.

**Exact Status-line non-contiguous wording used**: *"**AI-00 through AI-35 plus AI-41** are landed and verified."* — plus one explanatory sentence: *"AI-41 lands out of dependency order — before AI-36 and AI-37 — because it discharges two Wave-2 carryovers (`W1`/`W2`) that neither blocked Wave 4 nor depended on Wave 5's other open milestones… This is why the landed range above is stated as 'AI-00 through AI-35 plus AI-41' rather than a single contiguous range."* This deliberately avoids the AI-33/34/35 precedent's contiguous-range phrasing, per the launch prompt's explicit caution.

---

## Acknowledged Deviations

### D-A1 — Pre-existing out-of-scope guard modified, then further narrowed

**Deviation**: Apply's commit `b82cfdf` modified a pre-existing, out-of-this-change's-declared-scope test (`S-AIP-008`, part of `TestErrorEvent_TerminalExclusivity_NeverBothCompletionAndError`, landed at AI-19/R-AIP-003) because the new `GoString()` method on `*ai.Failure` collided by exported-method-name with the unrelated, pre-existing `Completion.GoString()`.

**Justification**: The scenario's own spec text (`openspec/specs/ai-provider-errors/spec.md:76`, pre-amendment numbering) is a *purposive* qualifier — "no accessor name or return type is shared in a way that lets a consumer read a category off a completion or a finish reason off a failure" — not an unconditional zero-shared-names rule. `Error`/`String`/`GoString` each render fixed diagnostic text on both types; neither prohibited read opens. Apply's initial fix (3-name exemption `{Error, String, GoString}`) was faithful to the spec but broader than the one real collision.

**Resolution**: `sdd-verify` (Engram #2674, W-1) proved via non-mutating overlay that the 3-name exemption silently passes a planted `*Failure.String()` returning completion-shaped data (`"finish reason: stop"`), while a narrowed 1-name exemption (`{GoString}`) correctly fails it. Commit `b271637` applied the narrower exemption; the guard's protective power for every other accessor name (including `Category`, `FinishReason`, and anything future) is unchanged, since the exemption map is checked first and only `continue`s for the one literal name now in it.

**Acceptance**: Independently revertible (`git revert b82cfdf` isolates the whole S-AIP-008 touch; `git revert b271637` alone reverts only the narrowing, leaving the broader-but-still-spec-faithful 3-name form). Not a blocker at any point in the cycle — `verify-report` classified the original 3-name form as ACCEPTABLE WITH A NARROWER EXEMPTION, i.e. non-blocking WARNING, not CRITICAL.

**Evidence**: `apply-progress.md` § "Deviations from Design"; `verify-report.md` § 6 (the full independent-judgement audit); commit `b271637` on `feat/ai-41-wave2-carryovers`.

### D-A2 — Diff came in well over the apply-phase forecast

**Deviation**: `tasks.md`'s Review Workload Forecast estimated ~90–125 changed lines for the apply-phase PR diff. The apply-phase diff through `b82cfdf` alone landed at 220 lines (+208/−12); through `b271637` (the W-1 fix) it landed at 226 lines (+214/−12); the **final** diff, including the Judgment Day correction commit `c833a93`, lands at **269 lines (+257/−12)** — roughly **115% over** the upper end of the original forecast.

**Why the drift**: The forecast under-counted strict-TDD test-evidence density: the D-4 canary loop (three canaries × four verbs, plus a nil-receiver sub-test wrapped in `recover()` across all four verbs) and the six-distinct-assertion rule-4 test (identity, sentinel, position, two negatives, zero-regression) are both larger than a minimal happy-path test would be, by design — the assertion strength is what `sdd-verify` explicitly confirmed was load-bearing (§ 3 of the verify report), not padding. The unplanned S-AIP-008 fix/narrowing, and later the Judgment Day correction round (which rewrites the rule-4 proof to actually thread the witness's position argument), each added a further increment.

**Reason for acceptance**: 269 lines is well within the session's 1000-line exception-ok budget (731 lines of headroom); no `size:exception` request or maintainer approval was needed, unlike AI-34/AI-35's larger overruns.

**Evidence**: `apply-progress.md` § "Gate Results"; `verify-report.md` § 1 "Gates state"; `git diff --numstat origin/main c833a93` for the +257/−12 total, independently re-measured by the orchestrator.

### D-A3 — The exploration phase's OpenSpec artifact was written by the orchestrator, not the explore sub-agent

**Deviation**: `explore.md` under the (now-archived) change folder was written to disk by the orchestrator, not by the `sdd-explore` sub-agent that produced Engram observation #2662 — that sub-agent has no `Write` tool and could only persist its findings to Engram.

**Acceptance**: The content of `explore.md` is a faithful transcription of Engram #2662's content (verified by this phase — the two were compared during artifact retrieval), so no information was lost or altered in the handoff; this is a process note about *which actor* performs the filesystem write, not a content or scope deviation.

**Evidence**: Engram #2662 vs. the archived `explore.md`, both read during this phase.

### D-A4 — The initial archive execution's tooling gap (closed)

Recorded in full in "Execution history: the tooling gap and its resolution" above: no Bash/shell tool was available to the initial archive phase, so the change-folder *move* (only a *copy* was performed), the `git commit`, and the independent final-gate re-run could not be executed by that phase. **This gap is fully closed**: a subsequent shell-capable execution performed the move, committed everything as `3adbb8c`, and the orchestrator independently re-ran every gate. It is recorded here as history, not as an open item.

### D-A5 — Judgment Day's pointer-receiver finding is only partially closed by this change

**Deviation**: the Judgment Day review's first both-judge finding — `GoString`, `Error`, `Unwrap` and `Is` on `*ai.Failure` are all pointer-receiver, so a copied `ai.Failure` *value* is neither a `fmt.GoStringer` nor an `error`, and `%#v` on it still reflects every unexported field — is a **pre-existing** leak that this change narrows but does not close. Both judges classed it pre-existing rather than newly introduced.

**Why it is not fixed here**: switching any of the four methods to a value receiver would make `fmt` dereference a typed-nil `*Failure` to build the copy, turning `%#v` on a nil `*Failure` into a nil-dereference panic — a strictly worse outcome than today's `Error()`, whose nil check currently returns the no-failure label safely.

**Resolution**: the `GoString` doc comment was corrected in `c833a93` to state the true scope (pointer-receiver only), rather than implying blanket coverage. The value-form sweep is **recorded as a follow-up owned by AI-36**, whose charter is exactly "every test plants a sentinel and asserts absence" across every failure path — the natural place to decide and prove the value-form fix, if one is adopted.

**Evidence**: Judgment Day findings, both judges (`jd-judge-a`, `jd-judge-b`), frozen HEAD `3adbb8c`; commit `c833a93`'s doc-comment correction; doc 0002's AI-41 close blockquote.

---

## Judgment Day Review

A Judgment Day dual-review round followed the archive commit `3adbb8c`. Two blind, read-only judges (`jd-judge-a`, `jd-judge-b`) reviewed the frozen target at HEAD `3adbb8c` (tree `91fe53d`, base `f2e460d`), scoped to the four changed Go files (`event_test.go`, `export_test.go`, `provider_failure.go`, `provider_failure_test.go`).

**Result: 0 SEVERE from either judge.** Four findings were confirmed by *both* judges independently:

1. **Pointer-receiver scope (WARNING, both judges classed pre-existing).** `GoString`, `Error`, `Unwrap` and `Is` are all pointer-receiver, so a copied `ai.Failure` *value* is neither a `fmt.GoStringer` nor an `error`, and `%#v` on it still reflects every unexported field including `cause`. The change strictly narrows the pre-existing leak rather than opening one. Not fixable by a receiver-type change alone (see D-A5 above). Resolution: the doc comment was corrected to state the true scope, and the value-form sweep is recorded as a follow-up owned by AI-36.
2. **`WitnessPayload.validate` discarded its `at Path` argument**, so `CheckEmit`'s rule-4 positional prefix (`Path{At("event")}`, `event.go:365`) was proven by nothing.
3. **The position assertion was tautological** — `reflect.DeepEqual(violation.Path(), planted.Path())` compares a value against a clone of itself once pointer identity is already asserted, because `Violation.Path()` returns `slices.Clone(v.at)`.
4. **The `defer recover()` in the nil-receiver test was unreachable** — `fmt`'s `handleMethods` installs its own `catchPanic` around `GoString`/`Error`, so a panic renders as a `%!v(PANIC=…)` marker instead of escaping `fmt.Sprintf`.

One single-judge finding was also applied: `%q` was missing from the adversarial verb loop (the package precedent at `reasoning_content_test.go:1175` enumerates five verbs).

**All four confirmed findings plus the `%q` gap were fixed in commit `c833a93`.** Findings 2 and 3 were fixed together: the witness now positions its planted violation under the `at` argument it receives, which makes the position assertion prove that rule 4 threads the `event` prefix. This forced a design adjustment: the position is stored as `*Path`, not `Path`, because a slice field would make `WitnessPayload` non-comparable and would break the pre-existing zero-value comparison at `event_test.go:172`.

**Non-vacuity proofs, independently reproduced by the orchestrator using `go test -overlay`** (a non-mutating technique — the working tree was never modified):

- Removing `GoString` makes all three planted canaries leak: `%#v` renders `&ai.Failure{…, rawLabel:"raw-label-CANARY-77f3", …, requestID:"request-id-CANARY-88e1", cause:"cause-CANARY-99a2", …}`.
- Changing `event.go:365` to `e.payload.validate(nil)` makes the rule-4 test fail with `violation.Path() = witness_rule, want event.witness_rule` (the planted position under `CheckEmit`'s own `"event"` prefix).
- Under the earlier broad `{Error, String, GoString}` exemption, a planted `func (f *Failure) String() string { return "finish reason: stop" }` passed the `S-AIP-008` guard silently; under the narrowed `{GoString}` exemption it fails, naming the shared non-diagnostic accessor.

**Judgment Day terminal state: APPROVED.** 0 SEVERE, 0 contradictions, one discretionary correction round applied and re-verified.

---

## Spec Promotion Evidence

All three deltas promoted by full inline text insertion (per the AI-33/AI-35 precedent — canonical specs stay self-contained; the archived delta specs remain the historical record):

- **`openspec/specs/ai-event-envelope/spec.md`**: +1 requirement (`R-AEE-021`), +2 scenarios (`S-AEE-071`, `S-AEE-072`), +1 append-only discharge blockquote on "Carried forward".
- **`openspec/specs/ai-provider-errors/spec.md`**: +1 requirement (`R-AIP-016`), +2 scenarios (`S-AIP-056`, `S-AIP-057`). No NFR added (see delta's own rationale, restated above).
- **`openspec/specs/ai-stream-testkit/spec.md`**: 0 requirements added/modified/removed/renamed; the line-39 carryover ledger amended append-only. `R-STK-001`…`R-STK-013` untouched, no `S-STK-*` identifier consumed.

Numbering verified clean at spec time and re-verified at verify time: `R-AEE-020`/`S-AEE-070` and `R-AIP-015`/`S-AIP-055` were the prior maxima; no collision.

---

## Change Folder Archive

**Moved from**: `openspec/changes/cachicamas-ai-wave2-carryovers/` — removed from the active tree as part of commit `3adbb8c`; it no longer exists at that path.
**Moved to**: `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/` — committed, the sole surviving copy.

Archived folder contains:
- `explore.md` — architectural exploration; location correction for `CheckEmit`, sibling-pattern survey for `GoString`
- `proposal.md` — intent, scope, three pre-resolved design flags, risks, rollback plan
- `specs/ai-event-envelope/spec.md` — delta spec, 1 requirement, 2 scenarios, "Carried-forward discharge" blockquote text
- `specs/ai-provider-errors/spec.md` — delta spec, 1 requirement, 2 scenarios, "Why no NFR-AIP-F" rationale
- `specs/ai-stream-testkit/spec.md` — requirement-free delta fixing the exact archive-step ledger wording in advance
- `design.md` — 5 architecture decisions (D-1…D-5), redaction contract table, blast-radius analysis (later shown incomplete by W-2)
- `tasks.md` — 5 work units + 5 archive-phase tasks, all 24 checkboxes now `[x]` with inline evidence (19 apply-phase + 5 archive-phase, the latter annotated by this phase)
- `apply-progress.md` — 2 commits' TDD evidence, gate results, the S-AIP-008 deviation writeup, plus an archive-time note pointing to `b271637`
- `verify-report.md` — PASS WITH WARNINGS verdict, per-requirement audits, the full S-AIP-008 independent-judgement section, plus an archive-time note recording W-1 resolved
- `archive-report.md` — this document

---

## Recommended Next Step

No new SDD change is required for AI-41 itself — it is fully closed: committed, gated green, Judgment Day approved, and the working tree clean. No further action is outstanding for this change.

**Next milestone**: AI-36 (secret redaction) is now unblocked (doc 0002 charter `Blocks: AI-36`) and is the next recommended SDD target. It also inherits the value-form `%#v` follow-up recorded in D-A5 above.

---

## Key Learnings

1. An SDD archive phase without shell-tool access can complete every file-content duty (spec promotion, doc amendment, artifact copy) but cannot perform the git move, commit, or gate re-run the archive contract also requires; the correct response is to record that gap explicitly, not fabricate command output, and a later shell-capable execution can close it in full.
2. A method-set reflection guard like `S-AIP-008` is invisible to a literal-string blast-radius grep for verb occurrences (`%#v`, `GoString`); adding an exported method to a type also covered by such a guard needs a `reflect.TypeOf`/`NumMethod` grep, not just a usage-string grep.
3. A three-name exemption map justified by a purposive spec clause can still be measurably broader than the one real collision; proving the narrower form is safe requires a planted-method overlay probe, not just re-reading the guard's own comment.
4. Recording a carryover discharge in a spec's ledger line before its code and promoted requirement land reproduces the exact "silent pass" failure mode the discharging milestone exists to close — the target wording should be fixed in the delta spec in advance so the archive step is a transcription, not a judgement call.
5. When a milestone lands out of dependency order relative to its numeric neighbors (AI-41 before AI-36/AI-37), the landed-range status line must state the set explicitly rather than reuse the prior contiguous-range phrasing, or it becomes a false claim the moment a reader checks it.
6. A blind dual-judge review can find a test's assertion is tautological (comparing a clone of a value against itself) even when the assertion count looks strong; positional or structural proofs need a witness that actually receives and echoes the argument under test, not one wired to a fixed value.
7. Narrowing a leak (making four accessors pointer-receiver-only) is not the same as closing it; a value copy of the same type can still bypass every one of those accessors, and that residual scope belongs on the record as an explicit owned follow-up rather than left implicit.
