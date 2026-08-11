```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:7bf9fc9dfcdaf0e1e782672be8c3a504aac728bbf00828fad944440c4ec24804
verdict: fail
blockers: 0
critical_findings: 0
requirements: 11/15
scenarios: 44/48
test_command: python3 test_cmd.py — register-integrity and amended-surface-form conformance inspection over specs/agent-contract-vocabulary/spec.md (positive control first, then 86-row parse, per-category counts, duplicate scan, ordinal contiguity, R-AGV-015/S-AGV-046 surface-form test, provenance-kind audit, requirement/scenario census)
test_exit_code: 0
test_output_hash: sha256:5f0fdaf613a736517b5dc2465097b20a335b880e3aa60c5f8a635ea445bb8158
build_command: bash build_cmd.sh — build-inertness proof (git porcelain, tracked-diff check, backend/ go.mod go.sum go.work Makefile Dockerfile status, change-folder inventory, Go-identifier scan plus mixed-case and underscore blind-spot scans across the six artifacts)
build_exit_code: 0
build_output_hash: sha256:aa2c49c609945ffd55196a8e5a72fae54755da1bc59da1684ee135ca7065a453
```

# Verify report — Layer 2 contract vocabulary (pass 3, post-C3-remediation)

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 of doc 0003 — Layer 2 milestones and task graph
> **Node**: AG-00.1 — The vocabulary decision `[decision]`
> **Phase**: verify (third pass, after the orchestrator applied C3's fix directly)
> **Status**: **0 CRITICAL, 4 WARNING, 6 SUGGESTION** — all three original CRITICALs closed. Human-readable verdict: **PASS WITH WARNINGS**. Strict-envelope verdict: **`fail`**, and § 1.1 explains exactly why that word does not mean a blocker here.
> **Date**: 2026-08-11
> **Worktree**: `cachicamas-worktrees/agent-layer2-wave0` · **Branch**: `feat/agent-layer2-wave0` · **HEAD**: `b9b4e662`
> **Baselines**: pass 1 — FAIL, 3 CRITICAL, 8/15 requirements, 41 PASS scenarios. Pass 2 — FAIL, 1 CRITICAL (C3), 10/15 requirements.
> **Mode**: `[decision]` leaf — no production code, no `make test` gate. Every check is document inspection, executed mechanically wherever a count or a comparison can settle it.
> **Method**: nothing was carried forward. Every pass-2 verdict, including the ones pass 2 marked clean, was re-derived by command against the current bytes. Both scanning regexes were fired against a planted positive control **before** any clean result was accepted.

---

## 1. Method, and one correction to pass 2's own arithmetic

Zero executable code ships, so there is no runtime evidence and no covering test for any scenario. The evidence gate for a `[decision]` leaf is document inspection; two named inspection commands stand in for the test and build commands, and both digests above are real output of the scripts named.

**Pass 2's scenario total was wrong by one, and this report does not repeat it.** Pass 2's own § 13 ledger records PARTIAL for `S-AGV-001`, `S-AGV-004`, `S-AGV-034` and `S-AGV-036` — four rows — but its summary line and its envelope both said "45 PASS, 3 PARTIAL". The correct reading of that ledger is 44 PASS, 4 PARTIAL. This pass derives 44/48 independently and reports 44, not 45. No scenario regressed to produce the difference; pass 2 simply miscounted its own table.

### 1.1 Why the envelope says `fail` while the prose says PASS WITH WARNINGS

These are not in conflict, and the difference is worth stating plainly so nobody reads `fail` as "C3 is still open".

`gentle-ai sdd-verify-validate` binds the schema's `verdict` field to evidence *completeness*, not to blocker count. A `verdict: pass` is admitted only when `requirements` and `scenarios` are both complete over their totals. This pass derives four PARTIAL scenarios — `S-AGV-001`, `S-AGV-004`, `S-AGV-034`, `S-AGV-036` — so the honest counts are 11/15 and 44/48, and the validator denies `pass` against them:

```text
$ gentle-ai sdd-verify-validate --input report.md --requirements 15 --scenarios 48
Error: verify report admission denied: passing verdict contradicts failing or incomplete evidence   (exit 1)

$ # same bytes, verdict: fail
{"valid": true, "verdict": "fail", "evidence_revision": "sha256:7bf9fc9d..."}   (exit 0)
```

The only way to obtain `verdict: pass` here would be to declare 15/15 and 48/48, which this report's own § 12 ledger contradicts. That would be inflating the counts to buy a word, so it was not done.

So the envelope reads `verdict: fail` with `blockers: 0` and `critical_findings: 0` — the schema's shape for "no blocker, evidence not complete". The substantive result is in those two zeroes and in § 2: **C1, C2 and C3 are all closed**, and the four PARTIALs are the accepted warnings W1′, W3, W4, W5 plus the structurally-unresolvable `S-AGV-001`.

**One limit worth restating.** The change folder is still **untracked** (`?? openspec/changes/cachicamas-agent-contract-vocabulary/`), so there is no git baseline of the pre-remediation bytes. "X was not changed" cannot be proven by diff here; it is proven by re-deriving X from the current file and cross-checking against the two sibling changes that consume it.

---

## 2. Disposition of the three original CRITICALs

| # | Original finding | Disposition | Decisive evidence |
| --- | --- | --- | --- |
| **C1** | AG-21/AG-22 zero register coverage; `decision.md` claimed the opposite | **CLOSED** (re-derived) | `VL2-HAR-10` → AG-21; `VL2-SEAM-15`, `VL2-SEAM-16` → AG-22. Zero milestones absent from the register half. Deferral note at spec line 462, under the live `VL2-SEAM` heading. § 4.1 |
| **C2** | doc 0003 `R-19` and `R-20` absent from provenance | **CLOSED** (re-derived) | All of `R-01`…`R-21` present in the provenance column. `R-19` → `VL2-SEAM-15`, `VL2-SEAM-16`; `R-20` → `VL2-COR-01`. `G9` still correctly absent. § 4.2 |
| **C3** | `R-AGV-015`'s term-name clause excluded 12 of 86 rows and contradicted `S-AGV-046` | **CLOSED** | 0 of 86 term names fail the amended surface-form test, control fired 6/6 first. Requirement and scenario now carry the same four disqualifiers in the same order, both anchored on "surface form". The false justification is explicitly withdrawn. Zero rows renamed. § 3 |

---

## 3. C3 — the amended `R-AGV-015`, re-derived from zero

### 3.1 The positive control fired before any clean result was accepted

```text
POSITIVE CONTROL: {'AgentHarness': ['camel', 'leading-cap'], 'toolCall': ['camel'],
                   'snake_case_name': ['underscore'], 'OTel': ['leading-cap', 'consec-caps'],
                   'runtime.Gosched': ['code-styled'], 'Run()': ['leading-cap']}
control fired 6/6 -> surface-form test trusted
```

The control deliberately includes `OTel`, the consecutive-capital shape pass 2 identified as the blind spot of a naive camel-case regex, and `runtime.Gosched`, a dotted selector. Both fire. A benign two-word control, `agent runtime`, returns clean, so the test is not simply flagging everything.

### 3.2 The amended rule applied to all 86 term names

```text
rows: 86
per-category: {'COR': 23, 'EVT': 19, 'LOOP': 9, 'HAR': 10, 'SEAM': 16, 'OUT': 9}
R-AGV-015/S-AGV-046 surface-form FAILURES: 0 []
single-token terms: 13 ['run', 'turn', 'attempt', 'transcript', 'subagent', 'delegation',
                        'carrier', 'history', 'compaction', 're-entrancy', 'observer',
                        'frontends', 'catalogs']
```

**Zero failures of 86.** The thirteen single-token names are the identical thirteen pass 1 and pass 2 listed, in the same rows — `VL2-COR-04`, `-05`, `-07`, `-10`, `-14`, `-15`, `VL2-EVT-17`, `VL2-HAR-01`, `VL2-SEAM-03`, `-09`, `-11`, `VL2-OUT-08`, `-09`. **No register row was renamed**; the fix was made entirely in requirement and scenario text, which is what the amendment claims.

Pass 2 measured 12 of these 13 failing the first amendment's clause. All 13 now clear, including the 12 that did not, and `re-entrancy` clears for a principled reason rather than an accident of its hyphen.

### 3.3 The requirement and its scenario now agree

Both were extracted and compared, not read impressionistically.

```text
R-AGV-015 (spec line 220):
  "Terms MUST be expressed as conceptual noun phrases written as ordinary English —
   never camel-case, never Pascal-case, never underscore-joined, and never styled as
   a code identifier in any other way."

S-AGV-046 (spec line 229):
  "... then its surface form is ordinary English rather than a code identifier —
   not camel-case, not Pascal-case, not underscore-joined, not otherwise styled as code —"

disqualifier count      : 4 vs 4
predicates and order    : camel-case, Pascal-case, underscore-joined, catch-all — identical
both anchored on "surface form": True / True
```

Four disqualifiers each, same three named forms in the same order, and a semantically equivalent catch-all fourth. Both sentences state the same test and both name **surface form** as the thing being tested. The pass-2 defect — a requirement testing compilability and a scenario testing capitalization — is gone. Applying either text to the 86 names yields the same answer: all clear.

### 3.4 The false justification is explicitly withdrawn

Spec line 224, the second amendment blockquote, states verbatim:

> The claim that the thirteen rows "were always conceptual noun phrases and never Go identifiers" is withdrawn: as a syntactic claim it is false, and it is replaced by the surface-form scope above.

That is an explicit withdrawal naming the withdrawn sentence, conceding it was false, and pointing at what replaces it. It is not a quiet rewording. Its three supporting factual claims were each checked against the pass-2 baseline, which is the only record of the superseded text:

| Claim in the second amendment | Verified against | Holds |
| --- | --- | --- |
| The first amendment's clause was "never a single token that could itself compile as a type, field, method, or package name" | pass-2 report § 5.3, which quotes spec line 220 as it then stood | yes |
| "twelve of the thirteen single-token rows still failed" | pass-2 report § 5.3's classification table: 12 legal-Go-identifier=True, 1 False | yes |
| "only *re-entrancy* was cleared, by its hyphen" | same table — `VL2-SEAM-09` is the single `legal-Go-identifier=False` row | yes |

### 3.5 The amendment convention was followed

```text
both amendments dated 2026-08-11        : True / True
explicit withdrawal present             : True
superseded phrasing 1 quoted verbatim   : True  ("containing spaces.")
superseded phrasing 2 quoted verbatim   : True  ("never a single token that could itself compile ...")
```

Two dated blockquotes sit consecutively under `R-AGV-015`, the second appended below the first rather than replacing it. Neither edit is silent: each names what it changed and why, and each quotes the exact wording it supersedes, so the full revision history of this requirement is readable in place. `R-AGV-013` rule 4's strikethrough form is specified for *a superseded definition* — a register row — and does not govern requirement prose; the quotation-in-blockquote form used here preserves the same property, which is that superseded text stays visible.

---

## 4. Was the no-Go-identifier constraint preserved?

The rule has two jobs. The rewrite touched only the second.

### 4.0 The first sentence and `S-AGV-045` are intact

```text
first sentence intact       : True
  "No artifact of this change MAY contain a Go type name, field name, method name,
   interface name, or package identifier belonging to the future Layer 2 surface."
module-change clause intact : True
  "The change MUST NOT create, modify, or delete any file under `backend/`, any
   `go.mod`, `go.sum`, `Makefile`, or any build or container configuration."

S-AGV-045 (spec line 228) — unchanged in shape:
  "Given any artifact of this change, when a reviewer scans for camel-case or
   Pascal-case single-token names, struct or interface declarations, or field lists,
   then none is present."
```

The artifact-wide prohibition and the build-inertness clause are both untouched. Only the sentence governing **term names** was rewritten.

### 4.1 Nothing in the rewrite lets an actual Go identifier through

The supplementary scans, run because a clean result from one pattern is never proof:

```text
camel/Pascal hits across the six artifacts                    : 0
mixed-case blind-spot tokens (the "OTel" shape)               : {'OTel': 4, 'SDDs': 2, 'ADRs': 2, 'PRs': 4, 'IDs': 3}
underscore tokens                                             : cachicamas_coding ×7, finish_reason ×2, tool_call ×2
type/func/package declarations, struct/interface literals      : 0 in every file
go code fences                                                 : 0 in every file
literal "camelCase" / "PascalCase" strings                     : 0
```

Every mixed-case hit is an abbreviation or an abbreviation plural — `OTel` is the OpenTelemetry brand, `SDDs`/`ADRs`/`PRs`/`IDs` are ordinary plurals. None is a Layer 2 surface identifier. Every underscore token is legitimate: `cachicamas_coding` is doc 0001's Layer 3 application name, appearing inside trap 4's verbatim quote and in prohibitive corrections; `tool_call` and `finish_reason` are shipped Layer 1 file paths, which the register's own row-shape note explicitly sanctions as grounding.

**`S-AGV-045` passes.**

### 4.2 The amendment landed exactly at its stated precedent — not past it, not short of it

This was the specific over-correction to look for after two failed attempts, so it was measured rather than assumed. Layer 1's normative sentence (`openspec/specs/ai-contract-vocabulary/spec.md` line 456) constrains term names only to "conceptual noun phrases" and carries **no** identifier clause at all. The amended Layer 2 rule adds four surface-form disqualifiers on top of that. So Layer 2 is now strictly *stronger* than the precedent it cites and no longer stronger in the way that excluded its own rows. Pass 2's complaint — that the first amendment "went past its stated precedent" — no longer applies, and the opposite failure (weakening below Layer 1) did not occur either.

---

## 5. No renumbering — the wave's highest-risk regression, re-resolved

Both siblings were remediated after the pass-2 run, so their citation sets were re-resolved in full against the current register, not sampled and not carried forward.

```text
register defined ids: 86
cachicamas-agent-event-delivery/decision.md (AG-01): 29 distinct VL2-* ids, unresolved=0
cachicamas-agent-v1-scope/decision.md        (AG-02): 27 distinct VL2-* ids, unresolved=0
```

29 and 27, unchanged. Resolution was then checked **semantically**: every citation's surrounding sentence in the sibling was compared against the register's term and definition for that id. A first automated pass flagged 14 citations whose 360-character window did not echo the term; all 14 were then read manually and all 14 are correct — the flags came from several ids packed into one sentence, or from paraphrase. Representative:

```text
VL2-LOOP-04 effect class          REG "The scheduler's concurrency-policy discriminator on a tool ...
                                       Reads may run concurrently, up to a documented bounded ..."
                                  SIB "a concurrency policy (`VL2-LOOP-04`) — reads may run concurrently
                                       up to a documented bounded fan-out"
VL2-EVT-12..15 envelope invariants REG EVT-12 = "Envelope invariant 1", -13 = "2", -14 = "3", -15 = "4"
                                  SIB "envelope invariant 2 (`VL2-EVT-13`)", "envelope invariant 3
                                       (`VL2-EVT-14`)", "the invariants are `VL2-EVT-12` … `VL2-EVT-15`"
VL2-COR-11 pairing invariant      SIB "never orphaning a call/result pair (`VL2-COR-11`)"
VL2-SEAM-07 on-demand entry point SIB "invocable on demand at a turn boundary (`VL2-SEAM-07`)"
VL2-LOOP-08 finish-reason dispatch SIB "pause-resumption is model-initiated — a finish reason
                                       `VL2-LOOP-08` dispatches"
```

The invariant numbering is the sharpest test available, because an off-by-one renumber in `VL2-EVT` would show up as an invariant mismatch. It does not: the sibling's invariants 1–4 map to `VL2-EVT-12`…`VL2-EVT-15` exactly as the register defines them.

Ordinals are contiguous and unique in every category, with no gap, no repeat and no superseded row:

```text
ordinals contiguous: {'COR': True, 'EVT': True, 'LOOP': True, 'HAR': True, 'SEAM': True, 'OUT': True}
COR n=23 01..23 · EVT n=19 01..19 · LOOP n=9 01..09 · HAR n=10 01..10 · SEAM n=16 01..16 · OUT n=9 01..09
```

Neither sibling cites any of the three appended rows, and the highest ordinal either sibling cites in the touched categories is `VL2-SEAM-13` and `VL2-HAR-07` — both below the appended `VL2-SEAM-15`, `VL2-SEAM-16`, `VL2-HAR-10`. The sibling citation space lies entirely inside the pre-existing identifier space.

**Zero renumbering. Both siblings remain intact after their own remediation.**

---

## 6. C1 and C2, re-derived

### 6.1 C1 — AG-21 and AG-22 coverage

```text
AG-21 owned rows: [('VL2-HAR-10', 'hardening suite')]
AG-22 owned rows: [('VL2-SEAM-15', 'the observability boundary'), ('VL2-SEAM-16', 'content denylist')]
milestones ABSENT from the register half: none
AG-21 mentions: 3 | AG-22 mentions: 11
```

All 24 milestones `AG-00`…`AG-23` now appear in the register half; pass 1 measured `AG-21` and `AG-22` at zero. All three rows carry five non-empty cells and observable definitions.

The deferral note is in the **live** register, which was the whole point of the pass-1 finding, so it was checked by location:

```text
spec.md:462  > The exact attribute vocabulary — span names and the decided per-span attribute list
             for run, turn, tool-execution and compaction spans — is deliberately deferred to
             `AG-22.1`'s own `[decision]` node ...
             under heading: ## 5. Cross-cutting seams — `VL2-SEAM` (16 terms)
canonical-path declarations in this file: lines 6, 36, 40, 269
```

The note sits under the live `VL2-SEAM` category heading, inside the file `sdd-archive` promotes to `openspec/specs/agent-contract-vocabulary/spec.md`. **Correct location.**

### 6.2 C2 — provenance traceability

Parsed from the fifth column only, never from the whole row.

```text
R-01: 2 rows (VL2-COR-01, VL2-SEAM-15)   R-08: 8    R-15: 5
R-02: 2 rows (VL2-COR-01, VL2-COR-18)    R-09: 5    R-16: 4
R-03: 1 row  (VL2-COR-01)                R-10: 1    R-17: 3
R-04: 11                                 R-11: 7    R-18: 6
R-05: 8                                  R-12: 2    R-19: 2 rows (VL2-SEAM-15, VL2-SEAM-16)
R-06: 9                                  R-13: 4    R-20: 1 row  (VL2-COR-01)
R-07: 7                                  R-14: 5    R-21: 1 row  (VL2-SEAM-14)
R-01..R-21 present: True
G1..G8, G10, G11, G12: present            G9: ABSENT
provenance cells citing a line number: 0
provenance cells citing an unmerged change-folder path: 0
```

`G9`'s absence is still correct and was re-derived, not assumed: doc 0001 assigns `G9` to **L1 / AI-12** ("Per-request options plus a typed-but-opaque provider pass-through | L1 | ... | **in v1 — AI-12**"), so it carries no Layer 2 obligation and `NFR-AGV-B` does not require it.

**C2 is closed. `NFR-AGV-B` passes.**

---

## 7. Every count, recounted

```text
per-category measured: COR=23 EVT=19 LOOP=9 HAR=10 SEAM=16 OUT=9   total=86
duplicate ids: 0   duplicate term names: 0   duplicate definition texts: 0
rows whose cell count != 5 or with any empty cell: 0 / 86
```

Every location in the file that states a count, enumerated by parse rather than by memory:

| Line | Location | Stated | Measured | Agrees |
| --- | --- | --- | --- | --- |
| 273 | intro sentence — "86 terms in six categories" | 86 | 86 | yes |
| 359 | `## 1. Core identity — VL2-COR (23 terms)` | 23 | 23 | yes |
| 391 | `## 2. The event envelope — VL2-EVT (19 terms)` | 19 | 19 | yes |
| 421 | `## 3. Loop mechanics — VL2-LOOP (9 terms)` | 9 | 9 | yes |
| 439 | `## 4. Harness mechanics — VL2-HAR (10 terms)` | 10 | 10 | yes |
| 458 | `## 5. Cross-cutting seams — VL2-SEAM (16 terms)` | 16 | 16 | yes |
| 485 | `## 6. Excluded — VL2-OUT (9 terms)` | 9 | 9 | yes |
| 505 | `## Term count` per-category figures and sum | 23·19·9·10·16·9 = 86 | same | yes |

**Eight locations, eight agreements.** Line numbers are two higher than pass 2 recorded them, which is exactly the two lines the second amendment blockquote added — an independent confirmation that the C3 fix touched only `R-AGV-015`'s block and `S-AGV-046`'s line, and inserted nothing in the register itself.

Amendment rule 5 (spec line 293) covers all eight: intro sentence (1) + own category heading (1 of 6) + `## Term count` per-category figures **and** sum (2).

### `decision.md` snapshot parity, re-derived after the C3 edits

```text
register rows=86  decision snapshot rows=86
register  sha256 0a8d91a3d7bacde0878e281842794a017793f62901b7b8d4931deb2ed5ef430b
snapshot  sha256 0a8d91a3d7bacde0878e281842794a017793f62901b7b8d4931deb2ed5ef430b
BYTE-IDENTICAL: True
```

Parity survived a second remediation, which is the expected result given the C3 fix touched no register row.

---

## 8. The four wording traps, re-diffed from source

```text
source: doc 0003 lines 109, 110, 111, 112, leading "- " stripped
register: spec.md lines 315, 321, 327, 333, leading "> " stripped
byte lengths  src: 323 282 549 583      reg: 319 278 545 579      delta: -4 each
after removing ** emphasis delimiters:
  src_nb sha256: 56c820e5505939f436f2878d2fdc65045e41e52f163a8956b976b8f6ce4f9d79
  reg_nb sha256: 56c820e5505939f436f2878d2fdc65045e41e52f163a8956b976b8f6ce4f9d79
  IDENTICAL AFTER EMPHASIS STRIP: True
```

Zero prose drift. The `**` emphasis delimiters around each trap's leading sentence remain dropped, exactly 4 bytes per trap (W3), and trap 2's corrected phrasing still carries the file-touching clause only by citation to `VL2-COR-18` (W4). Neither was in the C3 fix's scope.

---

## 9. Provenance targets, all resolved

```text
doc 0001 sections cited: 2.2 2.3 3.2 4 4.1 4.2 4.3 5.1 5.2 6 7   -> 11/11 resolve to a real heading
ADR 0005 sections cited: D3, D4                                  -> 2/2 resolve
doc 0003 node references (AG-NN.M): 51 distinct                  -> 51/51 present in doc 0003
Layer 1 V-* ids: 21 direct + 2 cited ranges (V-MET-01…08, V-FAIL-05…10)
                 -> 29 distinct after expansion, 29/29 resolve in the live L1 register
live L1 register rows: 118
files under openspec/specs/ containing "VL2-": 0   (correct — promotion happens at archive)
```

Pass 2 reported 30 distinct `V-*` ids; the derived figure here is 29 (21 direct plus 8 new members from expanding the two ranges). The difference is a counting detail, not a resolution failure: every cited id and every interior range member resolves.

---

## 10. Ownership and category discipline

```text
non-excluded rows: 77 — every owner cell holds exactly one AG-NN; no range, no list
excluded rows: 9 — every owner is a layer, a port, or the composition root; zero AG-NN
distinct owners: 21 (AG-00 01 04 05 06 08 09 10 11 12 13 14 15 16 17 18 19 20 21 22 23)
non-excluded owners not matching ^AG-\d\d$: none
per-owner: AG-00:23 AG-01:6 AG-04:10 AG-05:2 AG-06:4 AG-08:2 AG-09:4 AG-10:1 AG-11:2
           AG-12:2 AG-13:1 AG-14:2 AG-15:3 AG-16:1 AG-17:2 AG-18:5 AG-19:2 AG-20:1
           AG-21:1 AG-22:2 AG-23:1
excluded owners: Layer 3 — permission-policy port / sandbox port / tool-source port /
                 injected summarization instruction / session persistence ×2 / price-table port;
                 above Layer 3 — consumes events only; the composition root
```

AG-00 measures **23** again, the same value pass 2 derived. Two consecutive passes now agree; only pass 1's 24 dissents, and with the folder untracked there is no baseline to settle which was the slip. On the merits the current assignment is right — `VL2-COR-09` *upward path* is owned by AG-01, whose charter Goal names "the **upward path** (permission decisions, steering input, interrupts)" verbatim. Carried as SUGGESTION 2, downgraded from pass 2 because the value is now stable across passes.

---

## 11. Requirement-by-requirement verdict

| Requirement | Scenarios | Verdict | Decisive evidence | vs pass 2 |
| --- | --- | --- | --- | --- |
| `R-AGV-001` register live, canonical, singular | S-001 PARTIAL, S-002 PASS | **PARTIAL** | Canonical path declared at spec lines 6, 36, 40, 269; `decision.md` labels itself historical and non-citable; promotion has not happened yet, so S-001 is only fully checkable after archive. | unchanged |
| `R-AGV-002` fixed row shape | S-003 PASS, S-004 PARTIAL | **PARTIAL** | 86/86 rows five non-empty cells; zero line numbers; zero unmerged paths; all doc/ADR/node/`V-*` targets resolve; five rows cite an SDD decision artifact (W1′). | unchanged |
| `R-AGV-003` exactly one owning milestone | S-005/006/007 PASS | **PASS** | 77/77 single `AG-NN`, all resolving to a doc 0003 heading; AG-21's Deliverable names "hardening suite" verbatim, AG-22's heading is "Add the observability boundary". | unchanged |
| `R-AGV-004` identifier discipline | S-008/009/010 PASS | **PASS** | Ordinals contiguous and unique in all six categories; appends took the next free ordinal; zero `VL2-` under `openspec/specs/`. | unchanged |
| `R-AGV-005` six categories | S-011/012 PASS | **PASS** | 6 headings, 23+19+9+10+16+9 = 86 = measured rows; 0 category-code mismatches. | unchanged |
| `R-AGV-006` one observable definition + boundary cases | S-013…S-018 PASS | **PASS** | 86 distinct definitions; boundary cases 1–4 each an explicit citable row (`VL2-COR-05`, `VL2-SEAM-06`, `VL2-COR-12`, `VL2-SEAM-08`); `VL2-COR-08` reconciles turn/provider call/attempt citing both sides. | unchanged |
| `R-AGV-007` reuse-versus-wrap | S-019/020/021 PASS | **PASS** | Reused table cites `V-*` only; the ordering entry says "explicitly **not** the Layer 1 per-stream counter". | unchanged |
| `R-AGV-008` must-nevers as citable obligations | S-022/023/024 PASS | **PASS** | `VL2-COR-17`…`22` six loop obligations + `VL2-COR-23` the harness one, each naming its guard. | unchanged |
| `R-AGV-009` name fixation | S-025…S-028 PASS | **PASS** | `VL2-COR-01` carries both negative clauses; the retired "the portable brain" framing is named with its reason; all 7 "coding agent" occurrences are prohibitive or inside trap 4's verbatim quote. | unchanged |
| `R-AGV-010` delegation vocabulary | S-029/030 PASS | **PASS** | `VL2-COR-14` *subagent* canonical, `VL2-COR-15` *delegation* canonical, two admissible synonyms marked as synonyms with distinct senses. | unchanged |
| `R-AGV-011` exclusions named, attributed, never defined | S-031/032/033 PASS | **PARTIAL** | 9/9 exclusion owners are a layer, a port or the composition root; no Layer 2 definitions; but the exclusion **category** still omits concepts named out-of-scope in charters (W5). | unchanged |
| `R-AGV-012` traps verbatim | S-034 PARTIAL, S-035 PASS, S-036 PARTIAL | **PARTIAL** | Prose byte-identical (matching sha256); `**` emphasis dropped (W3); trap 2's file-touching clause by citation only (W4). | unchanged |
| `R-AGV-013` growth by amendment | S-037/038/039/040 PASS | **PASS** | Rules 1–6 operationally complete; rule 5 covers all eight count locations; rule 3's "why the register lacked the term" is a required element and `S-AGV-038` tests its absence. | unchanged |
| `R-AGV-014` downstream charters expressible | S-041/042/043/044 PASS | **PASS** | AG-01's 29 and AG-02's 27 citations all resolve, identifier-wise and semantically; AG-21/AG-22 nouns resolve to the three appended rows. | unchanged |
| `R-AGV-015` no Go identifiers, no code | S-045/046/047/048 PASS | **PASS** | 0/86 term names fail the amended surface-form test (control fired 6/6 first); requirement and scenario congruent; artifact-wide prohibition and build-inertness clause intact; zero camel/Pascal tokens, zero declarations, zero `go` fences; markdown-only diff. | **FAIL → PASS** |

**Requirements clean: 11/15** (pass 2: 10/15; pass 1: 8/15). PARTIAL: `R-AGV-001`, `R-AGV-002`, `R-AGV-011`, `R-AGV-012`. FAIL: none.

### 11.1 Non-functional requirements

| NFR | Verdict | Evidence | vs pass 2 |
| --- | --- | --- | --- |
| `NFR-AGV-A` reviewability | **PASS** | Uniform five-column shape across all 86 rows; no row requires opening doc 0001 to be understood. | unchanged |
| `NFR-AGV-B` traceability | **PASS** | `R-01`…`R-21` all present in the provenance column; `G1`…`G8`, `G10`, `G11` present; `G9` correctly absent (doc 0001 assigns it to L1/AI-12); zero line numbers. | unchanged |
| `NFR-AGV-C` durability | **PARTIAL** | Zero citations point at Layer 2 code or an unmerged path; five rows still cite an SDD change's `decision.md`, which is none of the four admissible targets (W1′). | unchanged |

---

## 12. Scenario-by-scenario ledger — all 48 re-derived

| Scenario | Verdict | What decided it |
| --- | --- | --- |
| S-AGV-001 | PARTIAL | Promotion to the canonical path happens at archive; no `VL2-*` citation points into `openspec/changes/archive/` today. Structural, not fixable pre-archive. |
| S-AGV-002 | PASS | `decision.md` labels itself historical and non-citable; the register holds the normative rows. |
| S-AGV-003 | PASS | 86/86 rows, five non-empty cells each, measured. |
| S-AGV-004 | PARTIAL | Zero line numbers; 11 doc 0001 §, 2 ADR §, 51 doc 0003 nodes, 29 `V-*` ids all resolve; five rows cite an SDD decision artifact (W1′). |
| S-AGV-005 | PASS | 77/77 single `AG-NN`, all resolving to doc 0003 headings. |
| S-AGV-006 | PASS | `VL2-COR-01` owner = AG-00, the defining milestone. **Discriminating scenario — passes.** |
| S-AGV-007 | PASS | AG-21's Deliverable names "hardening suite" verbatim; AG-22's heading is "Add the observability boundary"; its Acceptance names the absolute content denylist. |
| S-AGV-008 | PASS | Ordinals 01..NN contiguous in all six categories; no superseded rows. |
| S-AGV-009 | PASS | Literal `V-<CAT>-nn` occurs in zero `VL2-` ids; zero `VL2-` under `openspec/specs/`. |
| S-AGV-010 | PASS (as stated) | Appends took the next free ordinal; no existing id changed value or position, proven by resolving all 56 sibling citations. |
| S-AGV-011 | PASS | Six headings, six counts, sum 86 = measured rows. |
| S-AGV-012 | PASS | 0/86 category-code mismatches; zero term names under two headings. |
| S-AGV-013 | PASS | 86 distinct term names, 86 distinct definition texts, measured. |
| S-AGV-014 | PASS | `VL2-COR-05`: "A turn with zero tool calls is still a complete turn — the terminal one". |
| S-AGV-015 | PASS | `VL2-SEAM-06`: "a transcript entry, not metadata beside history", typed and distinguishable from a model message. |
| S-AGV-016 | PASS | `VL2-COR-12`: belongs to the **next** turn; "Edge case: a steering message queued during the final turn yields a new turn rather than being dropped." |
| S-AGV-017 | PASS | `VL2-SEAM-08`: "a provider call ... but not a turn", with the definitional reason stated. |
| S-AGV-018 | PASS | `VL2-COR-08`: "a turn spans one or more provider calls; the count exceeds one only via harness retry", both sides cited. |
| S-AGV-019 | PASS | Reused table entries cite `V-*` only, with no independent definition sentence. |
| S-AGV-020 | PASS | Stream-scoped vs run/turn-scoped, parent nesting with no L1 analog, four families. |
| S-AGV-021 | PASS | Ordering entry: "explicitly **not** the Layer 1 per-stream counter". **Discriminating scenario — passes.** |
| S-AGV-022 | PASS | Exactly six loop must-nevers (`VL2-COR-17`…`22`), each naming its guard. |
| S-AGV-023 | PASS | `VL2-COR-23`: may vary *when* and *how often*, "may never reach inside a single loop invocation". |
| S-AGV-024 | PASS | AG-03.2/AG-03.3 named inside `VL2-COR-18`/`19`/`22`. |
| S-AGV-025 | PASS | `VL2-COR-01`: loop-plus-harness "and nothing else — not a third thing wrapping them, not a synonym for either alone". |
| S-AGV-026 | PASS | The retired "the portable brain" framing is named with its reason. **Discriminating scenario — passes.** |
| S-AGV-027 | PASS | Stated operationally as the phrase to write. |
| S-AGV-028 | PASS | "a Layer 3 application" throughout; all 7 "coding agent" occurrences read individually — 5 prohibitive, 2 inside trap 4's verbatim quote. |
| S-AGV-029 | PASS | `VL2-COR-14` canonical for the participant; two admissible synonyms listed as synonyms. |
| S-AGV-030 | PASS | Scope rule binds event kinds, scenario ids, test names, acceptance criteria. |
| S-AGV-031 | PASS | 9/9 exclusion owners are a layer, a port, or the composition root; zero `AG-NN`. |
| S-AGV-032 | PASS | No Layer 2 definition in any `VL2-OUT` row; each names the confusable Layer 2 concept. |
| S-AGV-033 | PASS | `VL2-COR-16` token-only, "no field that could hold money — money is Layer 3 enrichment"; `VL2-OUT-06` cites the other side. |
| S-AGV-034 | PARTIAL | Prose sha256-identical; `**` emphasis delimiters dropped, exactly 4 bytes per trap (W3). |
| S-AGV-035 | PASS | Trap 1's correction states schedules-against-contract, never executes / decides permission / decides confinement. |
| S-AGV-036 | PARTIAL | Trap 2's file-touching clause carried by citation to `VL2-COR-18`, not restated (W4). |
| S-AGV-037 | PASS (as stated) | Amendment rules 1, 2, 3, 6 together specify it. |
| S-AGV-038 | PASS | Rule 3 requires *why the register lacked the term*, and the scenario tests that its absence fails. **Discriminating scenario — passes.** |
| S-AGV-039 | PASS (as stated) | Rule 4 states struck-through supersession with the identifier retained. |
| S-AGV-040 | PASS | Rule 5 enumerates intro sentence + own category heading + `Term count` per-category and sum — all eight locations. |
| S-AGV-041 | PASS | AG-01's nouns resolve; its 29 `VL2-*` citations all resolve, semantically checked. |
| S-AGV-042 | PASS | AG-02's nouns resolve; its 27 `VL2-*` citations all resolve, semantically checked. |
| S-AGV-043 | PASS | AG-21's and AG-22's domain nouns resolve; the defect was recorded in `decision.md` § 6 and closed by amendment in this pull request. |
| S-AGV-044 | PASS | `VL2-COR-08`, `VL2-COR-14`, `VL2-COR-16`, `VL2-OUT-06` each record the disposition with both sides cited. |
| S-AGV-045 | PASS | Zero camel/Pascal tokens, zero declarations, zero `go` fences across the six artifacts; regex proven live against a 6/6 control; two supplementary scans found no missed shape. |
| S-AGV-046 | PASS | 0 of 86 term names fail the amended surface-form test, control fired first; the scenario and `R-AGV-015` now carry the same four disqualifiers. **FAIL → PASS was the pass-2 state; the requirement now agrees.** |
| S-AGV-047 | PASS | 7 files, all `.md`, all under the change folder; zero tracked modifications. |
| S-AGV-048 | PASS | No non-markdown path created, modified or deleted; `backend/`, `go.mod`, `go.sum`, `go.work`, `Makefile`, `Dockerfile` all clean. |

**Scenarios: 44 PASS, 4 PARTIAL, 0 FAIL of 48.** (Pass 2's ledger, correctly totalled: 44/4/0. Pass 1: 41/5/2.)

---

## 13. Diff scope and build inertness

```text
$ git status --porcelain
?? openspec/changes/cachicamas-agent-contract-vocabulary/
?? openspec/changes/cachicamas-agent-event-delivery/
?? openspec/changes/cachicamas-agent-v1-scope/
$ git diff --name-only HEAD                                                    -> (empty)
$ git status --porcelain -uall -- backend/ go.mod go.sum go.work Makefile Dockerfile  -> (empty)
change-folder inventory: 7 files, non-.md count: 0
  decision.md  design.md  explore.md  proposal.md  tasks.md
  specs/agent-contract-vocabulary/spec.md  verify-report.md
```

Nothing outside this change's own folder was touched. The two sibling untracked folders are the other wave-0 changes, not this one's diff. Build provably inert.

---

## 14. Completeness

| Metric | Value |
| --- | --- |
| Tasks total | 23 (`T-AGV-1` … `T-AGV-14`, `V-1` … `V-9`) |
| Tasks complete | 23 |
| Tasks incomplete | 0 |
| Requirements | 15 (`R-AGV-001` … `R-AGV-015`, contiguous — verified) |
| Scenarios | 48 (`S-AGV-001` … `S-AGV-048`, contiguous — verified) |
| NFRs | 3 (`NFR-AGV-A`, `-B`, `-C`) |

```text
tasks checked  : 23     tasks unchecked: 0
requirements   : 15     contiguous: True
scenarios      : 48     contiguous: True
```

`tasks.md` was not modified by either remediation pass and still matches the artifact state: all 23 items map to content that exists, and the C3 fix landed inside `T-AGV-11`/`T-AGV-13`'s requirement-authoring scope rather than requiring a new task. Task state and artifact state agree.

---

## 15. Findings

### CRITICAL

**None.** All three original CRITICALs are closed and independently re-derived — C1 and C2 in § 6, C3 in § 3.

### WARNING — carried forward, not re-escalated

All four warnings below were open at pass 2, were not in the C3 fix's scope, and remain open. **The orchestrator has accepted them for this pull request.** They are recorded here at their honest severity so the record is complete; none of them blocks archive, and this report does not treat any of them as a blocker.

**W1′ — five rows cite an SDD change's `decision.md` as provenance (`R-AGV-002`, `NFR-AGV-C`, `S-AGV-004`).**

```text
SDD-decision-artifact provenance targets:
  VL2-EVT-10, VL2-EVT-18, VL2-EVT-19, VL2-SEAM-11, VL2-SEAM-12
```

`R-AGV-002` enumerates the admissible kinds as "doc 0001 §, doc 0003 requirement `R-01`…`R-21` or forward-requirement `G1`…`G11`, an ADR, or a shipped Layer 1 register row". `NFR-AGV-C` lists four admissible targets. *"AG-01.1's decision"* is none of them; a reader following it lands in an archived change folder. The archive-time path fragility pass 1 found is genuinely gone — zero `openspec/changes/…` paths remain — so what is left is admissibility, not breakage.

Refining pass 2's phrasing, which was slightly loose on one point: of the two rows pre-added on AG-01's behalf, `VL2-EVT-19` does carry a doc-level co-citation (`doc 0003 AG-14.3`), but only for "the time bound this posture depends on", not for the loss posture itself; `VL2-EVT-18` co-cites `Layer 1 V-STR-09`, which **is** an enumerated admissible kind, but again only for "the rule this posture reuses unchanged". So for the loss-posture content specifically, both rows rest on the SDD decision artifact alone.

**W3 — the four trap quotations drop the source's `**` emphasis delimiters (`S-AGV-034`).** Unchanged. Prose is sha256-identical; each quotation is exactly 4 bytes shorter. Under the scenario's literal "character for character" this is a miss; under a prose reading it is clean.

**W4 — trap 2's corrected phrasing omits the file-touching clause (`S-AGV-036`).** Unchanged. The scenario's second required clause ("a harness which touches a file has crossed the boundary") is carried by citation to `VL2-COR-18`, not restated.

**W5 — the exclusion category still omits deliberately-unowned concepts (`R-AGV-011`).** Unchanged and measured again:

```text
grep -ic "budget configuration"   -> 0     (doc 0003 AG-17 Out of scope)
grep -ic "when to compact"        -> 0     (doc 0003 AG-18 Out of scope)
grep -ic "threshold"              -> 0     (doc 0003 AG-18 Out of scope)
VL2-OUT rows: still 9
```

`R-AGV-011` asks the exclusion category to list *every* concept Layer 2 deliberately does not own. The telemetry surface is recorded in the live register, but inside `VL2-SEAM-15`'s definition and the `VL2-SEAM` category note rather than as `VL2-OUT` rows. `VL2-OUT-09` does exactly this job for provider catalogs and credentials, so the pattern exists and the gap is a completeness gap, not a design gap.

### SUGGESTION

**S1 — the trap-3 "one mention" claim is still off by one.** Spec line 329 calls trap 3's quoted block "this register's one unavoidable mention of Go's `runtime` package *inside a quoted block*". The block contains two (`sed -n '327p' | grep -o "Go's \`runtime\`" | wc -l -> 2`). The rule the sentence states is correct; only the count inside it is wrong.

**S2 — pass 1 and passes 2–3 disagree on AG-00's owned-row count (24 vs 23).** Now measured at 23 in two consecutive independent passes, and correct on the merits (`VL2-COR-09` → AG-01, matching AG-01's charter Goal verbatim). Downgraded from pass 2: the value is stable and the assignment is right; only pass 1's figure dissents and there is no baseline to settle it.

**S3 — the artifacts still disagree on how many reconciled conflicts there are.** `proposal.md` line 198 names **two** (cost payload; the two "too broad" readings); `tasks.md` line 120 writes "both reconciled conflicts" but then names **three** (pairing invariant, steering, cost); spec line 260 says "both" without naming. `VL2-COR-11` and `VL2-COR-12` carry no both-sides citation, consistent with the two-conflict reading.

**S4 — `decision.md`'s byte-identical 86-row snapshot goes stale silently at the first post-merge amendment.** Intentional and sanctioned by `S-AGV-002`; the file labels itself historical in three places. Noted only because the snapshot becomes indistinguishable from the live register by inspection once they diverge.

**S5 — two nouns still resolve only weakly.** AG-13's "pause resumption" resolves via a clause inside `VL2-LOOP-08` rather than a row of its own; AG-23's "compatibility statement" is absent but is ordinary English.

**S6 (new) — two minor scope looseness items surfaced by this pass, neither a defect.**

1. `R-AGV-015`'s amended sentence says an ordinary lowercase English word "satisfies **this requirement**", where the antecedent is the term-name clause, not the whole requirement. The first sentence still forbids any Layer 2 surface identifier in any artifact, and `R-AGV-015` itself defers each term's *spelling* to the owning milestone's SDD, so the two sentences cannot conflict for the current register. Tightening "this requirement" to "this clause" would remove the ambiguity for a later reader.
2. `R-AGV-002` and `NFR-AGV-B` enumerate the forward-requirement range as `G1`…`G11`, while doc 0001 defines `G1`…`G13`. One row, `VL2-LOOP-08`, cites `G12(c)`; that row also cites `doc 0001 § 3.2, § 3.3`, so it satisfies `R-AGV-002` through the doc-section kind and nothing is broken. `G12` and `G13` are both delivered at Layer 1 (`in v1 — AI-07, AI-13, AI-18` and `decision only — AI-02`), so neither carries a Layer 2 obligation and `NFR-AGV-B`'s narrower range is defensible as written.

---

## 16. What was verified clean, stated positively

Re-derived independently in this pass and holding exactly:

- **All 86 term names satisfy `R-AGV-015` as amended** — zero failures, with the surface-form test proven live against a 6/6 positive control that includes the consecutive-capital and dotted-selector shapes a naive camel-case regex misses.
- **`R-AGV-015` and `S-AGV-046` state the same test** — four disqualifiers each, same predicates, same order, both anchored on "surface form".
- **The no-Go-identifier constraint survived the rewrite** — the artifact-wide first sentence, the build-inertness clause, and `S-AGV-045` are all intact, and the amended rule is strictly stronger than the Layer 1 precedent it cites.
- **The false justification is explicitly withdrawn**, and both amendments are dated blockquotes that quote the exact wording they supersede.
- 86 terms in six categories — `VL2-COR` 23 · `VL2-EVT` 19 · `VL2-LOOP` 9 · `VL2-HAR` 10 · `VL2-SEAM` 16 · `VL2-OUT` 9 — agreeing with all eight locations that state a count.
- Zero duplicate identifiers, term names or definition texts; all 86 rows carry exactly five non-empty cells under the heading matching their category code.
- **Zero renumbering**, proven by re-resolving AG-01's 29 and AG-02's 27 citations in full after both siblings were themselves remediated, identifier-wise and semantically, including the envelope-invariant numbering that would expose an off-by-one.
- Every non-excluded owner is a single `AG-NN` resolving to a real doc 0003 heading; every excluded owner is a layer, a port, or the composition root.
- 11 doc 0001 sections, 2 ADR 0005 sections, 51 doc 0003 nodes and 29 Layer 1 `V-*` ids all resolve; zero line numbers; zero unmerged-path citations.
- `R-01`…`R-21` all present in the provenance column; `G9`'s absence re-proved correct from doc 0001's L1/AI-12 assignment.
- Zero Go identifiers, declarations or code fences across the six artifacts, with two supplementary blind-spot scans finding no missed shape.
- Zero prose drift in the four wording traps — matching sha256 after removing markdown emphasis.
- `decision.md`'s 86-row snapshot is byte-identical to the register after two rounds of remediation.
- The diff is markdown under this change's own folder and nothing else; zero tracked modifications; build provably inert.
- All four discriminating scenarios the spec phase wrote to catch a plausible-but-defective register (`S-AGV-006`, `S-AGV-021`, `S-AGV-026`, `S-AGV-038`) pass on substance.

---

## 17. Verdict

**PASS WITH WARNINGS on the merits; `verdict: fail` in the strict envelope for evidence incompleteness only** — 0 CRITICAL, 0 blockers, 4 WARNING, 6 SUGGESTION. Requirements clean 11/15; scenarios 44 PASS / 4 PARTIAL / 0 FAIL of 48. See § 1.1: the envelope word tracks completeness, not blockers.

C3 is closed, and it is closed for the right reason rather than by a third re-encoding of the same prohibition. The first amendment failed because it swapped a spelling test for a compilability test, which is the same net for a lowercase English word. The second amendment moves the test to the surface form a reader sees, which is what `design.md` § 3 argued for all along, and it fixes the requirement and its scenario together — pass 2's finding was that they disagreed, and disagreement is how the defect survived the first attempt. All 86 term names now clear, the thirteen single-token rows included, with no row renamed and the artifact-wide Go-identifier prohibition untouched. The false supporting claim is withdrawn in writing rather than quietly dropped.

The remaining four PARTIAL requirements are the same four pass 2 carried. `R-AGV-001` cannot resolve before archive by construction. `R-AGV-002`/`NFR-AGV-C`, `R-AGV-011` and `R-AGV-012` rest on W1′, W5, W3 and W4 respectively — all four are known, measured, unchanged in scope, and accepted by the orchestrator for this pull request. None is a MUST that the register violates in a way a downstream consumer would trip over: the provenance citations resolve to real artifacts, the trap prose is byte-identical, and the exclusion gaps are omissions from a completeness list rather than wrong statements.

The regression that would have broken the wave — renumbering — did not happen, and was re-proved by resolving all 56 sibling citations after both siblings were independently remediated.

**Next**: `sdd-archive` on the merits — no CRITICAL remains and nothing found in this pass needs code or register change.

One routing caveat, stated rather than buried: if the archive gate keys on the strict envelope's `verdict` field rather than on `blockers`/`critical_findings`, it will read `fail` and refuse. Two honest ways forward, in order of cost: (a) record W1′, W3, W4 and W5 as explicitly accepted and let archive proceed on `blockers: 0` / `critical_findings: 0`; or (b) run one short `sdd-apply` pass to close W3 and W4, which are two literal text edits — restore the four `**` emphasis pairs in the trap quotations, and restate trap 2's file-touching clause instead of citing it — which would move `S-AGV-034` and `S-AGV-036` to PASS and lift the scenarios to 46/48. Neither W1′ nor W5 nor `S-AGV-001` can be closed by a text edit: W1′ needs a provenance decision, W5 needs new `VL2-OUT` rows, and `S-AGV-001` resolves only once archive itself promotes the file.

S1, S3 and S6 are one-line text fixes an amendment can carry whenever the register is next touched.
