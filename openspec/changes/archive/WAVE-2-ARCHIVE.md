# Wave 2 Archive Report — Layer 1 "Stream" (AI-14 … AI-20)

> **SDD archive status**: **closed** — seven changes verified, their contracts promoted into `openspec/specs/`, doc 0002 and doc 0001 amended.
> **Change set**: `cachicamas-ai-layer-1` Wave 2 — seven SDD changes closed as one deliverable.
> **Worktree**: `cachicamas-worktrees/ai-wave-2-archive` · **Branch**: `docs/2026-08-01-cachicamas-ai-wave-2-archive`, based at `37898c7`.
> **Verify report**: [`openspec/changes/archive/WAVE-2-VERIFY.md`](./WAVE-2-VERIFY.md) · Engram `sdd/cachicamas-ai-layer-1/wave-2-verify-report`.
> **Archive date**: 2026-08-01.

---

## 1. Delivery state — read this before citing anything below

| Fact | State |
| --- | --- |
| SDD archive | **closed** |
| PR #104 | **MERGED.** Nothing about this wave is owed to a reviewer. |
| `main` | `37898c7` — **contains Wave 2 in full.** AI-00 … AI-20 are the Layer 1 code on `main`. |
| Wave 2 implementation | merged at `37898c7`; this archive branch is based on it |
| doc 0002 shipped counter | **21 of 41** |
| `backend/agent` file counts | **33 production `.go` files, 44 test `.go` files** — re-counted against the tree, not copied from the verify report |

**This is the difference from Wave 1, and it matters.** Wave 1's archive ran against an *open* PR #101, so every promoted spec's `> **Introduced by**` line had to say so, and the Wave 1 report carried a standing caveat that `main` did not contain the code. Neither applies here. Every Wave 2 promoted spec states `merged to main in PR #104 (commit 37898c7)` as a fact, and a reader who assumes `main` carries this code is correct.

**One inherited caveat is now discharged.** Wave 1's report closed with "`main` does not contain Wave 1." It does now — PR #101 merged (`dda2432`), and PR #104 merged on top of it.

---

## 2. Per-milestone table

| Milestone | Change | Capability spec | Verdict | Closes (doc 0002 spine) |
| --- | --- | --- | --- | --- |
| AI-14 | `cachicamas-ai-event-envelope` | `openspec/specs/ai-event-envelope/spec.md` | PASS | **C3** in full — the process-global sequence counter, now structurally unwritable. Also lands the mechanism recorded for **C4**'s guard and the Layer 1 half of completion-checklist item 7 |
| AI-15 | `cachicamas-ai-response-events` | `openspec/specs/ai-response-events/spec.md` | PASS | the completion half of **G12(c)** and **G10** — finish reason and usage now cross the event boundary by value, so cost and finish are first-class stream facts |
| AI-16 | `cachicamas-ai-text-events` | `openspec/specs/ai-text-events/spec.md` | PASS | no tracked gap row of its own; `R-ATE-003`/`R-ATE-004` fix the shared 1-based block-index space that binds AI-17 and AI-18 |
| AI-17 | `cachicamas-ai-reasoning-events` | `openspec/specs/ai-reasoning-events/spec.md` | PASS | the stream half of **G12(b)** — the reasoning round-trip token survives the event boundary byte-exactly, whole on block-end |
| AI-18 | `cachicamas-ai-tool-call-events` | `openspec/specs/ai-tool-call-events/spec.md` | PASS | the Layer 1 half of **G5** (AI-18.3, the call ordinal on the stream side) and **G12(a)** (delta-optional tool calls) |
| AI-19 | `cachicamas-ai-provider-errors` | `openspec/specs/ai-provider-errors/spec.md` | PASS | **C4** by construction, and the Layer 1 half of **G8** — the typed taxonomy with a partial-output discriminator |
| AI-20 | `cachicamas-ai-model-provider` | `openspec/specs/ai-model-provider/spec.md` | PASS | **G13** — the stream carrier, decided by AI-02.1 and now pinned mechanically. Also the mechanism recorded for **G3**, and completion-checklist item 10 |

Seven of seven PASS. **223/223 task checkboxes complete, 104 requirements landed, 345 scenarios covered** — per the verify report, with no later work reopening any of them.

---

## 3. What the verify report blocked on, and why this archive could proceed

The Wave 2 verify report returned **FAIL** with 1 CRITICAL. That verdict is history, not current state, and the distinction is worth stating precisely because a future reader will find a `verdict: fail` header on the archived report.

**C1 — 26 OpenSpec artifact files, 3 595 lines, untracked.** At verification time, all five `spec.md` files for AI-16 through AI-20 (plus three complete change folders) existed on disk but were absent from version control. The consequences the report named were real: the PR contained no specification for five of seven milestones, and `sdd-archive` would have had nothing to promote for five capabilities.

**It was fixed before PR #104 merged.** The artifacts were staged and committed; all seven change folders are in the repository, and this archive read every one of them from tracked source. The `blockers: 1` / `critical_findings: 1` header on the archived verify report describes the tree at verification time. No CRITICAL was carried into this archive, and none was overridden.

---

## 4. Evidence gate

Recorded by `sdd-verify` at head `b4cbab4`, from `backend/agent/`:

| Gate | Result |
| --- | --- |
| `make test` (`go test -race -v ./...`) | exit 0 — **336** top-level tests, **1 271** subtest PASS lines, 0 FAIL, 0 SKIP (Wave 1 closed at 176 top-level; Wave 2 added 160) |
| `make lint` | exit 0 — 0 issues |
| `backend/agent/go.mod` | zero `require` lines. AI-19 added the wave's only new stdlib dependency, `time` |
| AI-00 forward guard | passes — `src/ai`'s closure is stdlib plus the module's own packages only |
| AI-00 reverse guard | passes |
| AI-10.4 dependency-closure guard | passes |
| AI-14.3 sequence-state guard | passes package-wide at wave end, allowlist pinned to one reasoned entry |
| AI-20.4 signature guard | passes; both bite mutations recorded and reverted |
| `evidence_revision` | `sha256:61d455db3fb59784385c032444af3c3bb474df1667063b19296a7dff27199a63` |

**This archive run changed nothing under `backend/`.** Only `docs/architecture/0001-…md`, `docs/architecture/milestones/0002-…md` and files under `openspec/` were touched, so the evidence above still describes the tree.

---

## 5. The three cross-milestone properties, at close

These are the reason a wave-level archive exists rather than seven independent ones. Each was checked against landed code by `sdd-verify`, and each is now the load-bearing claim of a promoted spec.

**The block-index space is shared at the type level, not by convention.** Exactly one declaration exists — `type BlockIndex uint64`, `event_descriptor.go:102` — and `git log -S` returns exactly one commit for it: `297f08d`, **AI-14's own foundation commit**, landed before any content family existed. One unexported `blockPayload` interface is implemented by exactly nine payloads across three families. The `0` sentinel is rejected in two places for all three families: at construction and again at the emission boundary, where `CheckEmit` rule 3 reads the shared interface and never a concrete type. Three independent index spaces are not merely absent — they are unwritable without introducing a second type the checker would not read.

**`R-AEE-015`'s extensibility claim is provable from git.** Five milestones registered twelve event kinds, and `git log --oneline main..HEAD -- src/ai/stream_check.go src/ai/event_descriptor.go src/ai/sequence.go` shows **no commit after AI-14's own NFR close-out** (`65d8be7`) touched the checker, the descriptor types or the sequence. `grep -E "EventKind[A-Z]|ResponseStart|Completion|TextBlock|Reasoning|ToolCall|Failure" stream_check.go` returns nothing. This is the strongest single result in the wave, and it is why `ai-event-envelope`'s promoted spec states the property as settled rather than aspirational.

**One `*Failure` type reaches both delivery paths.** It satisfies AI-14's sealed `eventPayload` directly, with no wrapper. Pre-stream and mid-stream both funnel through one assembler, `newFailure` (`provider_failure.go:542`), so `R-AIP-013`'s "same concrete type on both paths" is a structural fact rather than two implementations kept in sync.

---

## 6. The six fixes this archive made

The verify report assigned four WARNINGs to `sdd-archive` (**W3**, **W4**, **W5**, **W6**) and deferred two to Wave 3. All four archive-owned warnings are closed. They resolve into six discrete edits.

### Fix 1 — `R-ATE-009`'s live `[provisional]` marker (W3, first half)

**Was**: the requirement's own **title** read `### R-ATE-009 — A fragment is bounded by the existing text ceiling *(provisional)*`, and its body said *"This requirement is `[provisional]`: `design.md` MAY drop or replace it … if it is dropped, `S-ATE-022` is struck."* Repeated in the Open Items list.

**Why it mattered**: AI-16's `design.md` resolved this to **KEPT** and the cap ships and is enforced at `text_events.go:145`. Promoting the delta verbatim would have frozen a main spec telling a future reader that a landed, tested rule is still an open option and a passing scenario may yet be struck.

**Fixed in two files**:
- `openspec/specs/ai-text-events/spec.md` — `R-ATE-009` written as settled, `SHOULD` raised to `MUST` (the rule is enforced, and `S-ATE-022` asserts the failure), with the design rationale kept as a *"Why this bound and not another"* block and the recorded asymmetry (only the single fragment is bounded, never the block's reconstructed total) preserved.
- `openspec/changes/archive/2026-08-01-cachicamas-ai-text-events/specs/ai-text-events/spec.md` — same requirement corrected in the archived delta, with a dated `> **Resolved 2026-08-01 at Wave 2 archive**` blockquote quoting the struck text. Open item 1 struck through and marked closed.

**Source lines, as the change folder read before this pass**: `specs/ai-text-events/spec.md:120` (title), `:122` (body), `:197` (Open Items).

### Fix 2 — `R-ARE-013`'s live `[provisional]` sentence (W3, second half)

**Was**: `R-ARE-013`'s last sentence carried *"(This last sentence is `[provisional]`: … `design.md` MAY replace it with a carry-and-report reading … if it is dropped, `S-ARE-036` is struck.)"* Repeated in the Open Items list.

**Why it mattered**: identical class to Fix 1. AI-17's `design.md` resolved it to **KEPT** — a redacted block rejects a non-empty delta with `ErrMisplaced` at `"fragment"`, implemented at `reasoning_event.go:189`.

**Fixed in two files**:
- `openspec/specs/ai-reasoning-events/spec.md` — the rule stated as settled, made precise about the **non-empty** qualifier (a zero-length fragment on a redacted block is accepted, because it carries nothing), with the three-part design rationale retained: AI-07 parity, redaction means plaintext withheld, and construction-time decidability.
- `openspec/changes/archive/2026-08-01-cachicamas-ai-reasoning-events/specs/ai-reasoning-events/spec.md` — same, with a dated resolution blockquote quoting the struck text. Open item 1 struck through and marked closed.

**Source lines, as the change folder read before this pass**: `specs/ai-reasoning-events/spec.md:169` (requirement), `:234` (Open Items).

### Fix 3 — doc 0002's shipped counter, completion checklist and traceability spine (W4, first half)

File: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md`.

**Line 3, the status line.** Was *"Wave 0 + Wave 1 complete — **14 of 41** milestones shipped. **AI-00 through AI-13 are landed and verified.** … 19 production files and 27 test files."* Now reads *"Wave 0 + Wave 1 + Wave 2 complete — **21 of 41** milestones shipped. **AI-00 through AI-20 are landed and verified.** … 33 production files and 44 test files."*

> **The two file counts were re-derived, not copied.** The verify report's § 1 Completeness table states 33 and 44; this archive counted the actual tree rather than trusting it. **44 test files** matches exactly (38 in `src/ai` + 6 in `src/agenttest`). **33 production files** is 30 non-test `.go` files in `src/ai`, plus `src/agenttest/doc.go`, plus the two `src/ai/testdata/{constructed,handrolled}/main.go` fixtures — 31 if the `testdata` pair is excluded. The verify report's figure includes them. Recorded here so the arithmetic is reproducible rather than asserted.

**Completion checklist (lines 2275–2292).** Four items moved to checked, and an `> **Amended 2026-08-01 — Wave 2 close**` blockquote follows the list stating the reason for each, so a reader can re-derive the boxes rather than trust them:

| Item | Line (pre-edit) | Now | Closed by |
| --- | --- | --- | --- |
| 7 — order, ownership, per-stream sequence | 2281 | **checked** | AI-14.2, AI-14.3, AI-14.4, AI-20.1. The spine also lists AI-22.3, which *packages* these assertions as reusable helpers; it does not add the property, so the item closes on the property and AI-22.3 stays owed as ergonomics — stated explicitly in the amendment |
| 8 — every emittable kind has a constructible payload | 2282 | **checked** | AI-14.1 + AI-19.1 — twelve kinds, twelve witnesses, a bidirectional exhaustiveness assertion. **C4** made mechanically impossible |
| 9 — typed, safe, inspectable taxonomy with the discriminator | 2283 | **checked** | AI-19 in full |
| 10 — no vendor type, optional capabilities discovered | 2284 | **checked** | AI-20.1, AI-20.4, AI-20.5 |

**Item 12 (bounded, lossless backpressure) deliberately stays open**, and the amendment says why: AI-20's `R-AMP-012` *states* the contract, but the spine maps this item to **AI-34**, which locks buffer sizing and the behaviour. Stating a rule is not locking it. **Item 6 also stays open**, unchanged — its own text names the wire, which is AI-26.6 / AI-29.2.

**Traceability spine (lines 2333–2345, pre-edit).** Seven rows rewritten:

| Row | Change |
| --- | --- |
| **C3** | bare node reference → **Closed**, naming AI-14.2 and AI-14.3's one-entry allowlist |
| **C4** | bare node reference → **Closed**, naming AI-19.1's external-package construction proof and recording that the ordering obligation held (AI-19 shipped before AI-20) |
| **G4** | `(Wave 2)` → `(Wave 4)` on AI-26.2 — a stale wave label that would now read as an unmet Wave 2 obligation |
| **G5** | **AI-18.3 moved from "Remaining" to landed.** Now: Layer 1 half closed by AI-09.2 (request side) *and* AI-18.3 (stream side); remaining is AI-30.5, the wire, Wave 4 |
| **G8** | bare node list → **Layer 1 half closed**, naming the discriminator and `R-AIP-012`'s two-axis prohibition; remaining AI-32.2, AI-32.3, AI-35.1 |
| **G12(a)** | bare node reference → **Layer 1 half closed** by AI-18.2 |
| **G12(b)** | **AI-17.2 added** — the token now crosses the event boundary byte-exactly; wire proof still AI-29.2/AI-26.6, wave label corrected to Wave 4 |
| **G12(c)** | **AI-15.2 added** — the seven finish reasons now reach a consumer as a terminal completion event, not only as a constructible part; AI-31.1's wave label corrected to Wave 4 |
| **G13** | bare node reference → **Closed**, naming AI-20.4's AST walk, its `{"context"}` import allowlist and its two proven bites |

**Forward-requirements register (lines 2314–2321, pre-edit).** Three rows updated: **G3** now records the landed clean-absence semantics; **G5** distinguishes the landed request and stream halves from the wire half; **G10** records that AI-15.2's completion event carries the usage record **by value**, so `TokenCount`'s presence bit crosses the boundary untouched, and restates that Layer 1's obligation ends there.

### Fix 4 — doc 0001 § 9's streams-and-concurrency block (W4, second half)

File: `docs/architecture/0001-cachicamas-agent-stack-v2.md`, § 9 Review checklist, lines 675–681.

**The five checkboxes stay unticked, on purpose.** This block is a **per-milestone reviewer template**, re-run against each change — not a project progress bar. Ticking it globally would be the wrong edit. What changed is that it can now be *answered at all*: Wave 1 marked all five N/A five ways because it shipped request-side contracts and no stream existed.

An `> **Amended 2026-08-01 — this block became applicable at Wave 2**` blockquote follows the list with a per-item disposition table:

| Item | Disposition |
| --- | --- |
| Documented exit path; every send selects on cancellation | **Satisfied** — `R-AMP-009`, `R-AMP-010`, proven with AI-20's `scriptProvider`, `-count=15` stress-clean |
| Nothing closes a channel it does not own | **Satisfied** — receive-only at the boundary, single closing site |
| Cancellation is a context, not a polled flag; backoff waits on it | **Half satisfied, half not yet applicable** — the context half is closed; the backoff half has no subject, because `R-AIP-007` deliberately forbids a backoff schedule on the failure type. Becomes answerable at AI-35 |
| Delta events carry an index, never a snapshot | **Satisfied, structurally** — fragment-only deltas in all three families, and no accumulator ships anywhere, so a snapshot is unrepresentable through the public surface |
| No goroutine leak on early abandon; run under the race detector | **Half satisfied** — race detector closed (336 tests, 0 races); goroutine accounting exists on the pre-stream path; the early-abandon path is AI-33's and stays open |

Two **Contracts**-block items are also noted as first-time-answerable in the same wave: *every event kind constructible* and *tool-call deltas remain optional*.

### Fix 5 — AI-14's missing `apply-progress.md`, synthesized (W5)

**Was**: `openspec/changes/cachicamas-ai-event-envelope/` held proposal, explore, design, spec and tasks — no `apply-progress.md`. AI-14 was the only one of the seven without it. The *substance* was present and unusually good: `tasks.md` carries a full phase-by-phase evidence log with verbatim RED transcripts, a lint-gap discovery with its fix, and two honest deviation disclosures.

**Fixed**: `openspec/changes/archive/2026-08-01-cachicamas-ai-event-envelope/apply-progress.md`, synthesized from `tasks.md`'s Evidence Log in the same structure as the other six milestones' files.

**The provenance is stated at the top of the file, not buried.** It records that the file was written at archive time, why (the original apply session never wrote a separate one), that `tasks.md` remains the primary record and wins any disagreement, and that **every claim is traceable to `tasks.md` with nothing inferred**. Where `tasks.md` records an uncertainty, the synthesis preserves it rather than smoothing it over — most visibly:

- **Phases 1–3's RED is retroactive, not observed.** The apply ran across two sessions; the first disconnected before checkpointing. The synthesized file says so in a dedicated "Session shape" section rather than presenting six uniform RED/GREEN cycles.
- **The AI-14.3 guard's commit is not identified.** `tasks.md` names five commits but none for Phase 4, and the verify report's git log was filtered to three files that a `sequence_guard_test.go`-only commit would not touch. The file records this as an explicit uncertainty instead of guessing.
- **All four deviations are reproduced without softening** — including D1, the `CheckEmit` rule-4 coverage gap, with an archive-time follow-up noting it was never discharged and why (see § 7).

### Fix 6 — AI-15's `design.md` asserting an archive that never happened (W6)

**Was**, in `cachicamas-ai-response-events/design.md` line 23: *"`sdd-apply` remains gated behind AI-14's merge, which is satisfied (**AI-14 archived, `openspec/specs/ai-event-envelope/` promoted per doc 0002 milestone tracking**)."*

**That claim was false when written.** `openspec/specs/ai-event-envelope/` did not exist. AI-14's change folder was still unarchived. AI-15's own `apply-progress.md` deviation #2 records the discrepancy **correctly** — *"AI-14's code is merged and correct, but its OpenSpec artifact has not been archived/promoted … flagged for a later archive pass"* — so the milestone knew, and it was the design.md text that was stale relative to its own apply record. The verify report caught the contradiction and escalated it as **W6** / **D8**, the wave's only factually false disclosure.

**Fixed** in the copy that was archived, before it moved. The claim is struck through and replaced with the accurate sequence, in a dated `> **Corrected 2026-08-01 at Wave 2 archive**` blockquote:

- AI-15's `sdd-apply` correctly proceeded once AI-14's **code** merged — that is the gate that actually blocks implementation, and `event.go`, `event_descriptor.go`, `sequence.go` and `stream_check.go` were all present.
- AI-14's OpenSpec **promotion** happened later — in this Wave 2 archive pass, which also archived AI-15.
- `openspec/specs/ai-event-envelope/spec.md` exists as of this pass.
- Nothing about the pinned symbols or the design changes; only the archive-state claim was wrong.

A matching archive-time note was appended to AI-15's `apply-progress.md` recording that its flagged follow-up landed, and that its own account was the correct one of the two.

---

## 7. What this archive deliberately did **not** fix

Two verify warnings and five suggestions are Wave 3's, and are recorded in the promoted specs rather than silently dropped. Naming them here is the point: a future reader should find them from the archive, not rediscover them.

| # | Carried forward | Where it is recorded |
| --- | --- | --- |
| **W1** | `CheckEmit`'s rule 4 has no failure-path test. AI-14 deferred the coverage to "AI-15+"; AI-18's design gate — working correctly — then replaced the construction model that deferral depended on, so the receiver ceased to exist and nobody connected the two decisions. Not blocking (rule 4 is defensive depth behind a constructor that already validated), but the branch is untested at wave close | `ai-event-envelope` promoted spec, *Carried forward* section; AI-14's synthesized `apply-progress.md`; AI-18's `design.md` correction notice |
| **W2** | `*Failure` is the only Wave 2 payload without `GoString()`, so `%#v` falls back to reflection and reproduces the sanitized, bounded `rawLabel` and `requestID`. The wrapped cause's text does **not** leak, so `R-AIP-009`'s literal claim holds; the wave's own four-verb convention does not. Fix is one method plus one canary test | `ai-provider-errors` promoted spec, under `R-AIP-009`; AI-19's archived `design.md` and `apply-progress.md` |
| **S1** | Registered kind names mix two conventions — six unseparated (`responsestart`, `reasoningdelta`), six snake_case (`text_block_start`, `tool_call_start`), split by which agent landed first. These are `EventKind.String()`'s output, a public rendering surface | verify report § 6 |
| **S2** | AI-17 is the only block family whose tests never drive `CheckStream` — reasoning block ordering works by construction, not by runtime evidence | AI-17's archived `apply-progress.md` |
| **S3** | No test drives one realistic full stream end to end. Cross-family evidence is pairwise at best | verify report § 6 |
| **S4** | Text-event payloads have no dedicated rendering canary, despite `TextDelta` carrying raw model output | verify report § 6 |
| **S5** | Three requirements have no ID citation, and `R-AMP-004` has **no mechanical guard at all** — deleting one of the eight GoDoc ownership statements makes nothing fail. Roughly fifteen lines of test would close it | `ai-model-provider` promoted spec, under `R-AMP-004`; AI-20's archived `design.md` and `apply-progress.md` |

Two further facts are recorded in the archived artifacts rather than corrected, because correcting them would rewrite the audit trail:

- **AI-18's `design.md` asserted a construction model the code does not implement** (bare `Event` constructors, validation deferred to `CheckEmit`). Apply caught it and re-reconciled against AI-15's landed source. The design is preserved **verbatim and uncorrected**, behind a prominent archive-time correction notice, because the gate catching it is the point — the verify report calls it "the gate working" and the only case in the wave where a reconciliation gate actually bit.
- **AI-16's `apply-progress.md` claims its result was "within the forecast's ~260–320 estimate"**. It was not: 1 171 authored lines, 3.7× the forecast, and AI-16 was the only milestone declaring `400-line budget risk: Low`. The wave-level `size:exception` covers it, so nothing was blocked, but the estimate should not be cited as precedent. Corrected in an appended archive-time note rather than by editing the claim.

---

## 8. Archive operations — what landed

### Seven capability specs promoted into `openspec/specs/`

Every one was **transformed, not copied**. The Wave 1 archive report records in detail what happens when a delta is copied verbatim: the delta header survives (including a self-referential *"Canonical spec: openspec/specs/…/spec.md — created by sdd-archive from this delta"* line inside the very file that claims to be that canonical spec), and every relative link is wrong by two levels. Four Wave 1 specs shipped that way before it was caught. This pass applied all four transformations to all seven:

1. **Header replaced.** The delta's `> **Change**` / `> **Capability**: … Promoted to … at archive` framing is gone. Each promoted spec carries `> **Introduced by**: openspec/changes/archive/2026-08-01-<name>/, merged to main in PR #104 (commit 37898c7)`, `> **Status**: **live**`, a `> **Project**` line, a `> **Closes**` line naming what it closes in the doc 0002 spine, and `> **Sources**` with resolving doc links.
2. **Every relative link re-resolved** from the new depth. A delta at `openspec/changes/<name>/specs/<cap>/spec.md` is four directories from `openspec/`; a promoted spec at `openspec/specs/<cap>/spec.md` is two. `../../../../specs/ai-contract-vocabulary/spec.md` became `../ai-contract-vocabulary/spec.md`; doc references became `../../../docs/…`. Cross-capability citations now point at sibling promoted specs (for example `ai-tool-call-events` cites `../ai-text-events/spec.md`, not a change folder).
3. **A `## Status — this file is the canonical home of the contract` section added**, pointing at the archived change folder as the historical record, and stating which requirement in that file is the one most likely to be eroded by a later change.
4. **Every requirement and scenario body preserved verbatim**, except `R-ATE-009` and `R-ARE-013` (Fixes 1 and 2), which the verify report explicitly assigned to this phase.

### Seven change folders moved to `openspec/changes/archive/2026-08-01-<name>/`

All 41 source artifacts plus AI-14's synthesized `apply-progress.md` — 42 files — are written under the archive paths, following Wave 0's and Wave 1's convention: `proposal.md`, `explore.md`, `design.md`, `specs/<capability>/spec.md`, `tasks.md`, `apply-progress.md`.

**Wave 1's recorded link-depth divergence is fixed here rather than inherited.** Wave 1's report § 6 noted that an archived delta's own links are written for the *active* change location and become wrong by one level after the move to `archive/2026-08-01-<name>/`, and that Wave 0's archived deltas use archive-relative depths — a real divergence, not a matter of taste. Every moved file in this wave had its relative links re-resolved for the archive depth:

- delta specs: `../../../../specs/…` → `../../../../../specs/…`
- top-level change files: `../../../docs/…` → `../../../../docs/…`, `../../specs/…` → `../../../specs/…`
- cross-change citations: `../../../cachicamas-ai-text-events/…` → `../../../2026-08-01-cachicamas-ai-text-events/…`

Links relative *within* a change folder (`./proposal.md`, `../../proposal.md`, `./specs/<cap>/spec.md`) are unchanged, because the folder's internal shape did not change.

Each archived delta spec additionally carries a one-line banner naming its promoted live counterpart, so a reader who lands in the archive first is pointed at the current contract immediately.

### `openspec/specs/ai-contract-vocabulary/spec.md` was not touched

Not edited, not moved, not frozen, not marked closed. AI-15.1's amendment appending `V-STR-24` (**provider response identity**) and `V-STR-25` (**served model**), and moving the term count 116 → 118 with 23 → 25 stream-side, stands exactly as it landed in `75559c1`. The verify report § 3.5 confirmed that commit was the register's **only** touch in the whole wave, append-only, with no existing row renumbered, reworded, reordered or removed, and arithmetic re-checked. This is the AI-01 archive lesson from Wave 0, and it held for a second consecutive wave.

---

## 9. Lineage

| Artifact | Reference |
| --- | --- |
| Wave 2 verify report | [`openspec/changes/archive/WAVE-2-VERIFY.md`](./WAVE-2-VERIFY.md) · Engram `sdd/cachicamas-ai-layer-1/wave-2-verify-report` |
| Wave 2 archive report | this document · Engram `sdd/cachicamas-ai-layer-1/wave-2-archive-report` |
| Per-change artifacts | `openspec/changes/archive/2026-08-01-cachicamas-ai-{event-envelope,response-events,text-events,reasoning-events,tool-call-events,provider-errors,model-provider}/` |
| Per-change Engram topics | `sdd/{change-name}/{explore,proposal,spec,design,tasks,apply-progress}`, one set per milestone AI-14 … AI-20 |
| Wave 1 archive | [`openspec/changes/archive/WAVE-1-ARCHIVE.md`](./WAVE-1-ARCHIVE.md) · [`WAVE-1-VERIFY.md`](./WAVE-1-VERIFY.md) |
| Wave 0 precedent | `openspec/changes/archive/2026-07-31-cachicamas-{agent-module-scaffold,ai-contract-vocabulary,ai-stream-lifecycle,ai-minimum-capabilities}/` |
| Charter | `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 851–1163 |

---

## 10. Risks carried out of this archive

1. **`W1` is a real, undischarged coverage gap** whose owner was designed away. `CheckEmit` rule 4 is unreachable to a failure through the public API because every Wave 2 constructor validates eagerly. Two individually sound decisions invalidated each other, and only a wave-level verify found it. Wave 3 must either give `WitnessPayload` a controllable failure mode or record rule 4 as deliberately unreachable defence — the one thing it must not do is leave the deferral pointing at a receiver that no longer exists.
2. **`W2` is a reproduced, narrow leak.** `*Failure` under `%#v` prints two sanitized, bounded provider-supplied fields. `R-AIP-009`'s literal claim survives and the blast radius is small, but the wave's own stated rendering posture does not survive, and the inconsistency is exactly the kind that a later refactor turns into a real leak.
3. **`R-AMP-004` has no mechanical guard.** Nine documented ownership statements are load-bearing for every future adapter, and deleting any one of them fails no test. This is the largest "proven by prose" surface in Layer 1.
4. **Intra-leaf TDD ordering remains unauditable from git** for the whole wave — one commit per leaf, as in Wave 1. The verify report corroborated the transcripts against verbatim compile-failure strings and found nothing contradicting them, and three TDD violations were volunteered rather than concealed (one remediated with a delete/re-run/restore mutation proof stronger than the process it broke). The structural gap nonetheless stands for all seven changes.
5. **Two shared files were edited concurrently by three sibling agents in one worktree.** `event.go` and `event_registry_test.go` carry AI-16's, AI-17's and AI-18's registry entries, and AI-16's commit `4f63977` incidentally captured all nine. Verification confirmed all twelve rows present and correct and no sibling content lost, but the coordination mechanism was "re-read before every edit and hope" — it worked, and it is not a control.
6. **Wave 2's forecasting was good in aggregate and wrong where it mattered most.** Five of seven forecasts landed within ±30 %; the one milestone that declared the budget *safe* (AI-16, `400-line budget risk: Low`) was off by 3.7×. The wave-level exception absorbed it. A future wave without that exception would not have been so lucky.
