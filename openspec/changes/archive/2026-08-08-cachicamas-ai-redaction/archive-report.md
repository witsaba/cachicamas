# Archive Report — `cachicamas-ai-redaction` (AI-36)

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002 § 2150–2184; amendment :2161; AI-41 close :22) — Enforce secret redaction · **Wave**: 5 — Harden
> **Phase**: archive (final) · **Status**: **CLOSED with two maintainer-owned open items** (see § "Open items at close" — neither blocks the archive, both are recorded rather than resolved)
> **Date**: 2026-08-08 · **Branch**: `feat/ai-36-redaction` · **Base**: `origin/main@1103769` · **Final HEAD**: `8e7caa31` (ten commits)
> **Project**: cachicamas (witsaba) · **Worktree**: `cachicamas-worktrees/ai-36-redaction`
> **Artifact store**: hybrid — OpenSpec files **and** Engram observations

---

## Executive summary

AI-36 turns redaction from **a property of which surface a caller reached for** into **a property of the module**, and it discharges the value-form follow-up AI-41 recorded against itself at its own close. Six requirements landed across six capability specs, all promoted at archive time. `sdd-verify` returned **PASS WITH WARNINGS — 6/6 requirements, 23/23 scenarios, 0 CRITICAL**, having empirically defeated and restored six separate redaction guards to prove no absence assertion is vacuous. Judgment Day ran **two rounds and reached terminal state APPROVED**. All four verification gates were re-run independently by the orchestrator at final HEAD.

The milestone's two pre-designed empirical unknowns both came back **BITES, not clean** — which is the substantive result. A hostile provider echoing the caller's authorization header into a non-streaming response leaked that credential verbatim through the refusal's bounded excerpt; and the bare configured client leaked its raw token under every verb in both pointer and value form. Neither was hypothetical, both were observed before any production change existed, and both are fixed.

Two items are **open and owned by the maintainer**: a blocked Gentle AI attempt ledger, and W-3 — the prompt-body echo residual, which is spec-anticipated and deliberately unfixed but narrows the charter's Goal prose and therefore awaits explicit sign-off.

---

## Final-state authority — what supersedes what

Per the archive skill's Final-State Authority hierarchy, this report describes the state **at close**, not at any earlier point. Three intermediate claims are superseded and are **not** echoed here as current:

| Intermediate claim | Source, and when it was true | State at close |
|---|---|---|
| `status: partial`, size overrun handed back as an open decision | `apply-progress` (#2687), at HEAD `8e7fdc97` | **Resolved.** `size:exception` was granted by the maintainer up front, with automatic raising authorized for this milestone. Recorded as an accepted, non-blocking deviation. |
| **W-1** — tautological header-admission guard that also missed a second header-reading site | `verify-report` (#2688), at HEAD `8e7fdc97` | **RESOLVED** in commit `9e2c149d`, which replaced the literal-restating sub-test with a falsifiable structural guard — exactly W-1's own recommended fix. |
| **W-2** — dead unwrap-chain guard | `verify-report` (#2688), same snapshot | **RESOLVED** in the same commit `9e2c149d`. |
| Test count **2462 PASS** / 13 SKIP | `verify-report`, same snapshot | Superseded: **2472 PASS / 13 SKIP / 0 FAIL** at final HEAD, re-run independently by the orchestrator. |
| Code diff **2045 changed lines**, 6 commits | `apply-progress`, same snapshot | Superseded: **36 files, +4997/−130** across **ten** commits, driven by two review rounds and the archive artifacts. |

**W-3 is the one verify warning that survives to close**, and it survives as an open item, not as a resolved one. No contradiction between sources was left unrankable; nothing required silent resolution.

---

## Phase artifact inventory

| Artifact | Verdict | Observation | Archived file |
|---|---|---|---|
| **Explore** | PASS | Engram **#2679** | `explore.md` |
| **Proposal** | PASS | Engram **#2681** | `proposal.md` |
| **Spec (index + 6 deltas)** | PASS | Engram **#2682** | `spec.md`, `specs/*/spec.md` |
| **Design** | PASS | Engram **#2684** | `design.md` |
| **Tasks** | 43/43 complete | Engram **#2685** | `tasks.md` |
| **Apply-progress** | 43/43, 4 gates green | Engram **#2687** | `apply-progress.md` |
| **Verify-report** | PASS WITH WARNINGS, 0 CRITICAL | Engram **#2688** | `verify-report.md` |
| **Judgment Day round 1 findings** | 2 CRITICAL, both fixed | Engram **#2690** | recorded in this report |
| **Archive-report** | CLOSED | Engram **#2692** | this document |

**Task Completion Gate: PASSED.** All 43 implementation tasks were already `[x]` in the persisted `tasks.md` when this phase read it. **No checkbox was reconciled by the archive phase**, and no exceptional repair was performed.

---

## Verification verdict at close

**PASS WITH WARNINGS — 6/6 requirements, 23/23 scenarios COMPLIANT, 0 CRITICAL, 2 warnings resolved, 1 warning open (W-3).**

Gates, re-run **independently by the orchestrator at final HEAD `8e7caa31`** — not sourced from any phase agent's report:

| Gate | Result |
|---|---|
| `make test` | exit 0 — **0 `--- FAIL`, 2472 PASS, 13 SKIP** across 8 packages under `-race` |
| `make lint` | `0 issues.` |
| `make build` | exit 0 |
| `git status --porcelain` | empty (clean tree) |
| `go.mod` / `go.sum` | byte-identical to base — no new dependency |

The 13 SKIPs are pre-existing credential-gated live-smoke tests, unrelated to this change.

### Why the absence claims are trustworthy

The characteristic failure mode of a redaction milestone is a sweep that passes because the sentinel never reached the surface under test. `sdd-verify` refused to accept any absence claim on the strength of a passing test and instead **defeated six guards, observed a genuine RED, and restored byte-identical state** (five via `go test -overlay`, never touching the tree; one on-disk edit restored and SHA-256-verified):

| # | Guard defeated | Proves |
|---|---|---|
| E-1 | Credential removal in the refusal path made a no-op | The hostile-server leak is real and the guard bites |
| E-2 | The configured client's rendering methods removed | The bare-client leak is real, all four verbs, both forms |
| E-3a | The canary made pointer-shaped | The value-shaped placement is **load-bearing**, not decorative |
| E-3b | The vacuity-contrast pin made value-shaped | The contrast pin is itself falsifiable |
| E-3c | The by-value structural detector defeated | An escape guard that can never fail is not a proof |
| E-4 | A real planted sentinel neutralized below threshold | The allowlist is an asserted artifact, not decoration |
| E-5 | The bounded-summary fix reverted | The four unbounded identity fields were a real finding |
| E-6 | **The shared sweep itself blinded** | The milestone's central claim: a blinded sweep does **not** silently turn every absence assertion green — the self-test fires at every call site before the real corpus is trusted |

---

## The two empirical branches — both bit

Both unknowns were designed with **both** outcomes pre-specified, so neither could be resolved by assumption at apply time. Both resolved against the optimistic branch.

### D-4 — the non-streaming excerpt reproduced the caller's credential

A hostile server was driven to echo the request's authorization header and body into a 200 response with an unexpected content type, forcing the non-streaming refusal path. **Before any production fix**, the sentinel credential surfaced verbatim, byte for byte, in the rendered failure at unwrap depth 1 under all four verbs. That is the "credential IS reproduced" row of the pre-declared trigger table, so **`R-AEM-019` triggered and is binding** — promoted to the canonical spec as landed, not conditional. The removal step landed in the capture path, is applied strictly after the size bound so the bound is unchanged, and the same test re-run green is now the regression pin for both scenarios.

### D-5 — the bare configured client leaked its raw token

The bare, unwrapped configured client leaked the sentinel under all four verbs in **both** pointer and value form. The cause is structural, not incidental: a value reached through a field the package does not publish cannot dispatch to the redacting renderings its own type would otherwise supply, so the fallback walks its internal state and reproduces the credential. Fixed with value-receiver rendering methods mirroring the wrapping provider's landed precedent. `S-APC-078`'s required recording therefore reads **behavior newly established**, not behavior already holding — and that disposition is now written into the canonical spec so no future reader has to re-derive which branch fired.

---

## Judgment Day — two rounds, terminal state APPROVED

### Round 1 (target `9e2c149d`) — two judges, two *different* CRITICALs

1. **The refusal rendered two response-derived fields but redacted only one.** A credential echoed into a content-type parameter surfaced verbatim at unwrap depth 1. The orchestrator reproduced this empirically.
2. **Whole-needle replacement could not match a bisected credential.** The capture layer's fixed truncation offset can cut a credential in half, leaving an attacker-chosen prefix that whole-occurrence removal never matches.

Three further warnings, from both or a single judge: case-sensitive matching evadable by a case-folding provider — **with the guard sharing the same blind spot**; a placeholder documented as 11 bytes that is actually 10, behind an unenforced growth invariant; and a header-read pin whose comment claimed to over-match while it in fact under-matched.

**All fixed in `dfcc1663`.**

### Round 2 (target `dfcc1663`) — no SEVERE from either judge

Three findings, all fixed in `8e7caa31`:

1. The re-cap truncated without marking itself, so a cut excerpt read as complete.
2. The truncation-edge heuristic, applied to the content-type header, could erase the very media type the refusal is required to name.
3. The header pin's comment still overclaimed, while missing struct-field-qualified reads and multi-hop aliases.

**Terminal state: APPROVED.**

---

## Specification amendment summary

Six canonical specs amended, all by **full inline promotion** per the AI-33/AI-35/AI-41 precedent: requirements and every scenario placed inline, in ID order, in the correct section. Each promotion applied the four-part transform — no delta header carried across, cross-references resolved for the canonical file's own depth, the amendment's provenance stated in place, body otherwise unchanged. **The no-Go-identifier rule was honored throughout**: every added line is behavior-level, naming rendering verbs in prose and never a Go symbol, type, method, package or format verb. `R-AIP-016` is the reference register.

| Canonical spec | Added | Scenarios | Notes |
|---|---|---|---|
| `openspec/specs/ai-provider-conformance-suite/spec.md` | `R-CNF-027` | `S-CNF-077`…`S-CNF-081` | Plus the identifier note below, and the `S-CNF-081` named-residual recording |
| `openspec/specs/ai-provider-client/spec.md` | `R-APC-015` | `S-APC-077`…`S-APC-080` | Plus `S-APC-078`'s required empirical recording |
| `openspec/specs/ai-request-translation/spec.md` | `R-ART-022` | `S-ART-090`…`S-ART-094` | `R-ART-004` amended **append-only**, not edited |
| `openspec/specs/ai-stream-testkit/spec.md` | `R-STK-014` | `S-STK-047`…`S-STK-049` | Plus the item-2 outcome recording |
| `openspec/specs/ai-provider-errors/spec.md` | `R-AIP-017` | `S-AIP-058`…`S-AIP-061` | Plus item 3's recording, and an append-only `W2` discharge note |
| `openspec/specs/ai-provider-error-mapping/spec.md` | `R-AEM-019` | `S-AEM-071`, `S-AEM-072` | **Was contingent; it triggered — promoted as LANDED and binding** |

**Zero MODIFIED, zero REMOVED, zero RENAMED requirements.** Every requirement not named above was preserved untouched.

### Consequential count corrections, made because this change falsified them

Three prose counts became false the moment the new requirements landed. Each was corrected minimally, with its provenance stated in place so the correction is auditable:

- `ai-stream-testkit` canonical-home section: "thirteen requirements" → **fourteen**.
- `ai-provider-conformance-suite` canonical-home section: "eighteen requirements" and "eight-entry record" → **twenty** and **nine-entry**. *Both were already stale before AI-36* — AI-35 landed `R-CNF-019` and grew the record to nine entries without updating that sentence.
- `ai-provider-error-mapping` count line: 18 requirements / 70 scenarios → **19 / 72**, with AI-32's own figures preserved as a statement about AI-32.

### `R-AEM-019` — a contingent requirement resolved, not dropped

The contingent row was the one place this cycle could have silently lost a requirement. It did not: the delta file carried both dispositions and their trigger table from spec time, apply recorded the fired branch **in the delta file itself** with the empirical evidence, and archive promoted it as binding. The not-triggered disposition is preserved in the archived delta for completeness, explicitly marked as not this run's outcome.

---

## Doc 0002 amendment

`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md`, amended **append-only** — no existing amendment was rewritten:

1. **Top `> **Status:**` line** — shipped counter `37 of 42` → **`38 of 42`**; landed range → **"AI-00 through AI-36 plus AI-41"**; production/test file counts `85`/`148` → **`86`/`152`**; the "Remaining" clause trimmed from `Wave 5 (AI-36, AI-37) + Wave 6 (AI-38..AI-40), 5 milestones` to **`Wave 5 (AI-37) + Wave 6 (AI-38..AI-40), 4 milestones`**; and the AI-41 non-contiguity sentence's quoted range updated so it no longer quotes a stale string.
2. **New dated blockquote** — `> **Amended 2026-08-08 — AI-36 close…**`, appended immediately after the AI-41 close blockquote and before the authoring-constraint block, following the AI-33/AI-34/AI-35/AI-41 pattern exactly. It records: the six work units; both empirical branches biting; all ten commits and the final HEAD; the aggregate diff and the up-front `size:exception`; the six spec promotions; the deliberately-open `R-CNF` identifier gap; the independently re-run gates; the verify verdict with W-1/W-2 resolved and W-3 open; both Judgment Day rounds; the two maintainer-owned open items; the four informational non-fixes; the Engram ids; and that **AI-37 is now unblocked**.

**File-count provenance, stated honestly.** The `86`/`152` figures are **derived** from this change's own recorded file list — one created production file, four created test files, no deletion — not from a fresh `find` measurement, because this phase had no shell. The amendment says so in place and asks the next milestone to confirm them.

---

## Open items at close — maintainer-owned, NOT resolved

### O-1 — Gentle AI attempt ledger blocked

The runtime attempt ledger is **blocked with `reason: maintainer_decision`** because the final diff exceeds the 1600-line cap declared at acquire time. The attempt itself recorded `outcome: passed`. This is the same class of tooling friction recorded at AI-34 and AI-35 — a ledger-state problem, not a verification finding. It does not contradict any gate result, all of which were re-run independently.

### O-2 — W-3, the prompt-body residual, awaiting explicit sign-off

The prompt-body echo through the non-streaming excerpt is **left unfixed, deliberately**.

- **It is spec-anticipated, not discovered-and-excused.** `S-CNF-081` item 2, `R-AEM-019` items 2 and 4, and `S-AEM-072` all require it to be *recorded in writing as a named residual*. It now is, in three canonical specs.
- **The engineering reason is a genuine conflict between two landed requirements.** Suppressing a provider's replay of the caller's own content would defeat the landed obligation that the excerpt stay diagnostically readable. An excerpt scrubbed of everything the caller sent is not diagnostic.
- **The credential is a different matter and is provably absent** — that guard was empirically defeated and restored (E-1).
- **But it narrows the charter's Goal prose**, which names "sensitive prompt bodies" alongside credentials. `sdd-verify` rated it WARNING rather than CRITICAL, and explicitly recommended the maintainer acknowledge it at archive rather than let it pass silently. **That acknowledgement has not been given.** It is recorded here as an accepted, maintainer-visible residual awaiting sign-off.

---

## Informational — deliberately not fixed, recorded for AI-37 or later

None of these is an open defect; each is a bounded, known coverage limit recorded so a later author does not have to rediscover it.

1. **Interior non-tail credential prefixes** left by non-overlapping replacement.
2. **The deliberate-plant allowlist is per-file, not per-plant.** The judges disagreed here — one flagged it, the other verified both falsifiability directions hold. It is the granularity the requirement itself specifies; narrowing it is available future work.
3. **A conformance sweep lacks a corpus-non-empty guard.**
4. **The by-value structural guard does not cover composite or package-level variable shapes.** Both are absent from the shipped tree; neither is asserted against by a control.

---

## Standing repository defects — pre-existing, out of AI-36's scope

Recorded so they are not rediscovered as new. **Neither was introduced by this change and neither was repaired by it.**

### SD-1 — Seven `R-CNF` requirements were never promoted, and the identifiers are live

`R-CNF-020`…`R-CNF-026` are consumed by two archived changes — `2026-08-05-cachicamas-ai-conformance-lifecycle-amendment` (020–024) and `2026-08-05-cachicamas-ai-conformance-tool-amendment` (025–026) — whose requirements were **never promoted** into the canonical conformance spec, which still jumps from `R-CNF-019` to AI-36's `R-CNF-027`. The identifiers are nevertheless **live**: `R-CNF-023` and `R-CNF-024` are cited as binding dependencies by `openspec/specs/ai-provider-text-stream/spec.md` and `openspec/specs/ai-provider-completion/spec.md`.

AI-36 **took `R-CNF-027`, the first genuinely free identifier, and neither renumbered nor backfilled the gap** — that was the explicit scope ruling. An identifier note now records the gap in place, at the foot of the canonical conformance spec, so the next author reads it before re-deriving the wrong maximum from a canonical-only grep. **Closing it needs its own change.**

### SD-2 — Three canonical specs are verbatim delta copies, with unresolvable links

Three canonical specs still carry their delta headers rather than promoted ones — the exact defect `openspec/changes/archive/WAVE-1-ARCHIVE.md` § 2 documents:

| Canonical spec | Symptom |
|---|---|
| `openspec/specs/ai-provider-client/spec.md` | `Introduced by: openspec/changes/cachicamas-ai-provider-client/` (the *active* path, which no longer exists); `Status: **change-folder delta** — promoted to … at archive`; links use `../../../../specs/…`, which does not resolve from `openspec/specs/<cap>/spec.md` |
| `openspec/specs/ai-request-translation/spec.md` | Same three symptoms |
| `openspec/specs/ai-provider-error-mapping/spec.md` | Worse — it carries the self-referential line `**Canonical spec**: openspec/specs/ai-provider-error-mapping/spec.md — created by sdd-archive from this delta` *inside that very file*, plus a link into `../../cachicamas-ai-stream-decoder/…` that resolves nowhere |

Their archived change folders **do** exist (`2026-08-05-cachicamas-ai-provider-client`, `-ai-request-translation`, `-ai-provider-error-mapping`), so the repair is mechanical: rewrite the header to `Introduced by` the archive path and `Status: live`, re-resolve every relative link to the canonical depth, and add a canonical-home section. **This phase deliberately did not perform it** — it is a substantive edit to specs owned by AI-25/AI-26/AI-32, it would require asserting merge commits and PR numbers this phase could not verify without a shell, and it is outside AI-36's scope. AI-36's own six promotions were written to the correct depth and carry correct provenance.

---

## Change folder archive

**Copied from**: `openspec/changes/cachicamas-ai-redaction/`
**Copied to**: `openspec/changes/archive/2026-08-08-cachicamas-ai-redaction/`

This phase had **no Bash tool**, so the move was performed as Read-then-Write of every file. **All 13 files were copied; the folder is not half-moved.** The orchestrator must `git rm` the 13 source paths listed in the return envelope to complete the move.

| Archived file | Treatment |
|---|---|
| `explore.md` | Verbatim |
| `proposal.md` | Verbatim |
| `design.md` | Verbatim |
| `spec.md` | Verbatim + archive annotation (contingent resolved to LANDED; totals 6/23, not 5/21) |
| `tasks.md` | Verbatim + archive annotation (43/43 already complete, nothing reconciled; counts superseded) |
| `apply-progress.md` | Verbatim + archive annotation (`status: partial` resolved; 4 later commits; final diff) |
| `verify-report.md` | Verbatim, **admitted bytes untouched** + an appended archive annotation (W-1/W-2 resolved, W-3 open, counts superseded) |
| `specs/ai-provider-conformance-suite/spec.md` | Verbatim + archive note; links re-resolved 4→5 levels |
| `specs/ai-provider-client/spec.md` | Same |
| `specs/ai-request-translation/spec.md` | Same |
| `specs/ai-stream-testkit/spec.md` | Same |
| `specs/ai-provider-errors/spec.md` | Same |
| `specs/ai-provider-error-mapping/spec.md` | Same |
| `archive-report.md` | This document |

**Why the delta links changed depth.** A delta spec's relative links are written for the *active* location `openspec/changes/<name>/specs/<cap>/spec.md`, which is four levels below `openspec/`. After the move to `archive/2026-08-08-<name>/…` the correct depth is **five**. This is the second consequence WAVE-1-ARCHIVE § 6 records; it affects the archived audit trail only, never the live specs.

---

## Recommended next step

**No new SDD change is required for AI-36 itself.** The implementation is complete, gated green, and Judgment Day approved.

**Next milestone: AI-37 (observability boundary), now unblocked.** AI-36's charter declares `Blocks: AI-37`, and AI-37 may not emit a span until the redaction sweep it depends on exists. It now does. AI-37 also inherits the four informational items above, and is the natural owner of a per-plant allowlist narrowing and a corpus-non-empty guard should it want them.

**Two decisions the maintainer owes before this cycle is fully quiet**: sign-off on W-3 (O-2), and whatever action they choose on the blocked attempt ledger (O-1).

---

## Lineage

| Artifact | Reference |
|---|---|
| Phase observations | Engram `sdd/cachicamas-ai-redaction/{explore,proposal,spec,design,tasks,apply-progress,verify-report}` — ids **#2679, #2681, #2682, #2684, #2685, #2687, #2688** |
| Judgment Day round 1 | Engram **#2690** |
| This report | Engram **#2692** (`sdd/cachicamas-ai-redaction/archive-report`) + this file |
| Immediate predecessor | `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/` (AI-41 — the milestone whose follow-up this discharges) |
| Promotion-transform precedent | `openspec/changes/archive/WAVE-1-ARCHIVE.md` § 2 (the four-part transform, and the verbatim-copy defect it documents) |
| Charter | `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` §§ 2150–2184, amendment `:2161`, AI-41 close `:22` |
