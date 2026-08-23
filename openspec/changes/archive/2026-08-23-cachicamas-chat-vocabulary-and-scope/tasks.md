# Tasks: CH-00 — Record the archetype's vocabulary, seam answers and v1 scope

> **Change**: `cachicamas-chat-vocabulary-and-scope` · **Node**: CH-00.1 `[decision]` (the milestone's only node) · **Delivery**: `single-pr`, review budget **1000** lines, extension pre-authorised by the user up front · **Store**: hybrid (Engram `sdd/cachicamas-chat-vocabulary-and-scope/tasks` + this file) · **Binding**: `decision.md` per design D-A; spec `openspec/changes/cachicamas-chat-vocabulary-and-scope/specs/chat-archetype-contract/spec.md` (13 requirements, 2 NFRs, 43 scenarios) is normative and already written — nothing below re-derives it.

## Review Workload Forecast

| File | State | Lines |
|---|---|---|
| `proposal.md` | existing | 174 |
| `design.md` | existing | 92 |
| `explore.md` | existing | 130 |
| `specs/chat-archetype-contract/spec.md` | existing | 262 |
| `tasks.md` (this file) | new | ~220 |
| `decision.md` | new (Phase 1–6) | ~380 (AG-23's comparable `decision.md` precedent is 147 lines; CH-00 carries more tables — three vocabulary tables, an 11-row seam table, a gap-findings table, a deferral register, an inconsistency register — so it runs longer) |
| `docs/architecture/milestones/0005-…md` | modify, 2 lines | ~4 counted (1 line each way × 2 edits) |
| **Total (this PR)** | | **~1259** |

`openspec/**` is **counted, not excluded** — the repo default (`openspec/AGENTS.md`, corroborated by prior-change precedent) is that `--max-changed-lines` counts the **full diff including SDD markdown**, not source-only. This estimate is judged against the **session's 1000-line budget**, not the generic 400-line default, per this session's preflight.

```text
400-line budget risk: High
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
```

**Reasoning.** ~1259 estimated lines exceeds even the session's 1000-line budget by ~26%. This is not a stop condition: the user pre-authorised extension beyond 1000 up front, at session preflight, specifically because CH-00.1 is one indivisible `[decision]` node with no leaf structure to split across PRs or commits without breaking the record's own self-sufficiency requirement (`NFR-CHT-001` — a reader must close any of the seven questions from `decision.md` alone). Splitting the record across PRs would leave an intermediate PR with a self-contradicting partial record. `Decision needed before apply` is therefore `No` — the exception is already granted, not pending. `Chain strategy` is `size-exception` (the skill's `n/a for single-pr` case resolves to this because the delivery strategy is `single-pr` and the overage is already accepted).

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Author `decision.md` §§1–11 (Phases 1–6 below) | PR 1 (single PR), commit 1 | N/A — no test exists for a `[decision]` node; verification is the manual scenario walk in `verify-report.md` against all 43 `S-CHT-*` | Manual: walk every `S-CHT-*` scenario against the merged `decision.md` | Revert `decision.md` alone — no later CH milestone has started, nothing else cites it yet |
| 2 | Milestone doc bookkeeping (Phase 7) | PR 1, commit 2 | N/A — text edit; verified by `git diff` showing exactly the checklist row and status line | N/A — no executable behavior | Revert the 2-line edit independently of `decision.md` |
| 3 | Record evidence gate output (Phase 8) | PR 1, commit 3 | `cd backend/agent && go test -race -count=1 ./...`; `golangci-lint cache clean && make lint`; `make build`; `make vuln-check` | The uncached race run is the harness — its wall-clock duration is the proof it was not a cache hit | N/A — no source changed; this commit only records pasted command output |

## Phase 1: Header block, rules block, § 1 "How to use" — `decision.md`

- [x] 1.1 Write the header block (change / milestone / node / status / sources) per design D-A, citing `0005:202-240`, ADR 0009 § D6/D7(a), ADR 0005 § D2.
- [x] 1.2 Write the `[!IMPORTANT]` rules block restating (a) the generic-client boundary (`agent-layer3-handoff/spec.md:183` — no files/shells/skills/terminals as this archetype's capabilities, chat concepts allowed) and (b) `TurnOptions.PreRequestHook` frozen-and-superseded (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:45` — kept, no deprecation marker), each as the record's own sentence with its citation.
- [x] 1.3 Write § 1 "How to use this document", modelled on AG-23 `decision.md:17-24`.
- [x] 1.4 Self-check the draft: grep for "removed"/"deprecated" near every `PreRequestHook` occurrence, and for "file"/"shell"/"skill"/"terminal" outside a quotation of the boundary rule itself. **Deviation found and fixed**: the grep found the trigger words in three additional non-quotation spots — "the terminal one" (§ 2.1 turn row), "Per-file spec discipline" (§ 7 REQ-7 row), and two "file" mentions describing `home/index.tsx` in § 7/§ 10 (F-3). All four were reworded to remove the literal words entirely (e.g. "the terminal one" → "the last one in the run"; "That file contains" → "That source carries"), since `S-CHT-051` reads as a literal whole-document scan, not scoped to "presented as a capability." Re-grepped clean after the fix: only the two sanctioned occurrences (the rules-block restatement and the verbatim quotation) remain.

**Discharges:** `R-CHT-006` (`S-CHT-050`, `S-CHT-051`, `S-CHT-052`, `S-CHT-053`)

## Phase 2: § 2 Vocabulary — `decision.md`

- [x] 2.1 Write the **mapped nouns** table (`Noun | Maps onto (exactly one VL2-* id) | Layer 2 term | Citation`) — 10 rows: run `VL2-COR-04` (`:135`), turn `VL2-COR-05` (`:136`), the loop `VL2-COR-02` (`:133`), the harness `VL2-COR-03` (`:134`), upward path `VL2-COR-09` (`:140`), transcript `VL2-COR-10` (`:141`), steering `VL2-COR-12` (`:143`), message lifecycle `VL2-EVT-04` (`:165`), run outcome `VL2-EVT-10` (`:171`), turn outcome `VL2-EVT-11` (`:172`). **Deviation found and fixed**: `S-CHT-002` reads as requiring the citation *inside* the `Maps onto` cell itself, not only in the adjacent `Citation` column design D-B's 4-column layout carries — added an inline `(:NNN)` citation to every `Maps onto` cell in addition to keeping the separate `Citation` column, satisfying both.
- [x] 2.2 Write the **coined nouns** table (no `Maps onto` column) — rows "conversation" (gap: cardinality — `VL2-COR-04` run `:135` is one harness invocation, one conversation is one-or-more runs; containment, not identity) and "participant" (gap: `VL2-COR-14` `:145` names the delegated-subagent sense, a different sense than a human participant).
- [x] 2.3 State plainly that Layer 2 defines no term for a session, no term for a human participant, no term for a "part", and quote `VL2-OUT-07` (`decision.md:252`) as an exclusion assigned to Layer 3, never as a term.
- [x] 2.4 Write the **inherited nouns** table (no `Maps onto` column; an `Inherited from` column naming exactly one Layer 1 `V-*` id) — one row, "message" → `V-REQ-02` (`openspec/specs/ai-contract-vocabulary/spec.md:111`) — stating both why it is not a Layer 2 mapping (Layer 2's only message-headed row is `VL2-EVT-04` message lifecycle, an event family already mapped under its own name in § 2.1) and why it is not a coinage (`V-REQ-02` is a promoted, citable Layer 1 row this archetype uses unchanged); note that this fourth Layer 2 absence is deliberately not listed with the three in § 2.2. **Added after verify round 1 (CRITICAL C-1)**: the noun "message" was used in the record's `[!IMPORTANT]` rules block (which sits above § 1, not inside it), in § 2's own heading, and in § 11's Q1 row, while sitting in no table (cited here by section rather than by line, because inserting § 2.3 moved every line below it), breaching `R-CHT-001`. Neither available block was truthful — Layer 1 defines the noun, so a coinage row would assert an invention that did not happen and a mapped row would assert a Layer 2 term that does not exist — so `R-CHT-001` was widened from two structurally distinct blocks to three, preserving its exhaustiveness clause ("exactly one of the three; a noun in none, or in more than one, MUST fail review") and adding `S-CHT-004` for the new table's shape. The three-block shape follows the house convention already carried by 19 promoted specs, which open with a `> **Binding vocabulary**` header listing the Layer 1 identifiers they inherit by identifier (e.g. `openspec/specs/ai-message-roles/spec.md:9`).

**Discharges:** `R-CHT-001` (`S-CHT-001`, `S-CHT-002`, `S-CHT-003`, `S-CHT-004`), `R-CHT-002` (`S-CHT-010`, `S-CHT-011`, `S-CHT-012`)

## Phase 3: § 3 Seam table, § 4 Gap findings — `decision.md`

- [x] 3.1 Write the 11-row seam table in AG-23 source order (`archive/2026-08-21-…/decision.md:31-53`), columns `# | Seam | Status | Injection point | v1 answer | Owner milestone`; `Status` cells hold the literal token `frozen` or `experimental — not frozen` (8 + 3, per the heading at `:49`).
- [x] 3.2 Fill every row's `Injection point` and `v1 answer` as direct statements — no "see source", no bare Go identifier standing alone. A deliberately empty answer is written in the `0005:116` form (what empty means operationally, its injection point, the owning milestone or "no owner and why"). Five of the eleven rows (6, 7, 9, 10, 11) resolve to "no owner in v1," each with its own stated reason.
- [x] 3.3 Write § 4 "Gap findings against Layer 2's own seam set — findings, not seam answers" immediately after § 3; opening sentence states the frozen enumeration is exactly the 11 rows above and this section extends nothing. Table `Finding | Where Layer 2 names it | AG-23 status | Disposition` (no `Injection point`/`v1 answer` columns) — rows: v2 § 6 seam 3 sandbox (`0001-…v2.md:670`, rides `tool.go:117`), seam 7 retry classification (`:674`), `Harness.RetryTiming` (doc comment `harness.go:96-97` "the injected clock and wait-function seam", field `:100`), `RetryAttempts` (`:94`), `ContextBudget` (`:120`), `System` (`:56`).
- [x] 3.4 Describe `tool.go:265-266` as a sentence that wraps and completes — "`ToolSource` port (G6) is AG-20's widening." — the word "truncated" MUST NOT appear about it. **Deviation found and fixed**: the first draft wrote "It is not truncated" — a negation, but still a literal occurrence of the banned word, which `S-CHT-042` ("the word 'truncated' is used nowhere about it") would fail on a literal grep. Reworded to state only the positive "wraps and completes" claim, with zero occurrences of "truncat*" anywhere in the file (re-verified by grep).

**Discharges:** `R-CHT-003` (`S-CHT-020`, `S-CHT-021`, `S-CHT-022`, `S-CHT-023`), `R-CHT-004` (`S-CHT-030`, `S-CHT-031`, `S-CHT-032`), `R-CHT-005` (`S-CHT-040`, `S-CHT-041`, `S-CHT-042`)

## Phase 4: § 5 Identity, § 6 Persistence, § 7 Frontend — `decision.md`

- [x] 4.1 § 5: state the archetype's name, package `backend/agent/src/chat/`, composition root `backend/agent/src/cmd/chat/`, citing `0005:8` and ADR 0005 § D2; state plainly neither path exists yet and there is no `backend/agent/src/cmd/` at all; name CH-01.1 as the owner of their creation.
- [x] 4.2 § 6: answer the database question naming the **owner** of the tables (not the intent), citing ADR 0009 § D6 (`0009:152`, the quoted sentence at `:154-155`) — "each business system owns its own tables; no archetype writes to another system's schema."
- [x] 4.3 § 7: name the attaching frontend, enumerate `frontend-chat-layer1` `REQ-1`…`REQ-7` marked inherited/not-this-document's-to-design; cite the auth guard chain at its **corrected** location `frontend/src/routes/home/layout.tsx` — `onRequest` `:30`, its three calls `:31-33`, doc comment `:21-29`, imports `:15-17`; state that no frozen-wire change is proposed and the `REQ-5` literal is retired only by a future CH-05.2 spec delta.

**Discharges:** `R-CHT-007` (`S-CHT-060`, `S-CHT-061`, `S-CHT-062`), `R-CHT-008` (`S-CHT-070`, `S-CHT-071`), `R-CHT-009` (`S-CHT-080`, `S-CHT-081`, `S-CHT-082`)

## Phase 5: § 8 Deferral register, § 9 Substitution rule, § 10 Inconsistency register — `decision.md`

- [x] 5.1 § 8: reproduce doc 0005's "Explicitly deferred until after v1" table (`0005:997-1011`) row for row, citing `0005:997` as source, every row carrying its attaching seam; cite AG-23's known-limitations register (`archive/…/decision.md:103-108`) as inherited Layer 2 input, do **not** reproduce it; confirm no `F-1`/`F-2`/`F-3` row appears here.
- [x] 5.2 § 9: list the **eight** "a Layer 3 application" occurrences by file:line (`agent-contract-vocabulary/spec.md:146,153,335`; `agent-layer3-handoff/spec.md:17,34,177,196`; `agent-v1-scope/spec.md:317`) and quote ADR 0009 § D7(a)'s substitution rule verbatim (`0009:174-176`); state the matching rule the list was derived under — a search for the phrase "Layer 3 application", not for the exact string "a Layer 3 application", because `agent-layer3-handoff/spec.md:34` carries an intervening modifier ("a **future** Layer 3 application") that an exact-string search silently misses — measured, loose returns 8 hits / 3 specs and exact returns 7 / 3, differing only at `:34`; record separately that `agent-v1-scope/spec.md:317` was missing from the first list through neither the pattern nor a mis-scoped search, but through an **unresolved open count** — `explore.md:88` asserted "plus two additional unlisted occurrences" while enumerating one, and the six-item list was carried forward past that acknowledged remainder; at every later citation of one of those artifacts, state it is read under the substitution rule.
- [x] 5.3 § 10: write the inconsistency register as the last content section before closing verification — columns `# | Side A (cited) | Side B (cited) | Disposition`, rows `F-1` (`agent-contract-vocabulary/spec.md:339` placeholder vs `R-AGV-001:36`/`S-AGV-001:40`), `F-2` (the eight substitution-rule occurrences, across three promoted specs, vs ADR 0009 § D7(a) still open), `F-3` (`frontend-chat-layer1/spec.md:9,40,134` wrong citation vs the corrected `layout.tsx` location); header line states verbatim "Recorded, not repaired. No promoted spec is modified by this change."; note `F-1`/`F-2` share `agent-contract-vocabulary/spec.md` so one future change can close both.

**Discharges:** `R-CHT-010` (`S-CHT-090`, `S-CHT-091`, `S-CHT-092`), `R-CHT-011` (`S-CHT-100`, `S-CHT-101`, `S-CHT-102`), `R-CHT-012` (`S-CHT-110`, `S-CHT-111`, `S-CHT-112`)

## Phase 6: § 11 Closing verification — `decision.md`

- [x] 6.1 Write § 11 walking the seven closing questions (`0005:219-225`) in order, naming the section where each answer lives.
- [x] 6.2 Add the acceptance statement mirroring `0005:210` verbatim: a reader asks any seam Layer 2 names, the record answers directly and names its injection point, without consulting source.
- [x] 6.3 Self-review: pick one of the 11 AG-23 seams at random, confirm the record answers both halves without opening Go source (`S-CHT-120`); confirm § 11 names where each of the seven questions' answers live (`S-CHT-121`).
- [x] 6.4 Confirm every one of the seven questions is answerable from `decision.md` alone — remove `spec.md` from hand and re-check no answer disappears (`S-CHT-130`).

**Discharges:** `R-CHT-013` partial (`S-CHT-120`, `S-CHT-121`), `NFR-CHT-001` (`S-CHT-130`)

## Phase 7: Milestone doc bookkeeping — `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md`

- [x] 7.1 Tick the CH-00.1 completion-checklist row at `:980` (`- [ ]` → `- [x]`).
- [x] 7.2 Change the status line at `:3` from "**0 of 12**" to "**1 of 12**" — no other text on that line changes. **Deviation found and fixed**: leaving "Not started" beside "1 of 12 milestones shipped" is a direct self-contradiction, and doc 0005's own sibling doc 0003 established the house convention of dropping "Not started" the moment its first (also `[decision]`, zero-code) Wave 0 milestone shipped. The orchestrator's launch prompt explicitly permits "adjust the surrounding sentence only as far as truthfulness requires" — applied that exact allowance and changed "Not started" → "In progress," touching no other word on the line.

**Discharges:** none (charter bookkeeping, not a `CHT` requirement — see proposal § "Doc 0005 bookkeeping").

## Phase 8: Evidence gate — no production code, no RED step

There is no RED step in this change. D-1 (proposal, binding) types CH-00.1 `[decision]`: it ships no production code, no new Go test, no guard — the package a guard would protect does not exist yet. Strict TDD is satisfied **vacuously**: there is no production behavior to drive red-then-green. This is not a skipped discipline; it is the correct discipline for a `[decision]` node.

- [x] 8.1 State the no-RED posture above verbatim in `verify-report.md` so a reviewer does not read its absence as an omission. **Deviation found and fixed**: this apply phase's own launch prompt scoped its output to exactly three artifacts (`decision.md`, the two milestone-doc bookkeeping edits, and Phase 8 evidence "captured into `apply-progress`") and explicitly excludes any other file change. `verify-report.md` is not among them and is the customary output of the `sdd-verify` phase, not `sdd-apply`. The no-RED posture is recorded verbatim below and in `apply-progress` instead; `verify-report.md` itself is left for `sdd-verify` to create.
- [x] 8.2 Run `cd backend/agent && go test -race -count=1 ./...`; record the wall-clock duration. A `(cached)` or sub-second result is **not** evidence — re-run until the duration is consistent with an uncached race run. **Result**: green, `go clean -testcache` run first, wall-clock **2:56.40 total** (`openaicompat` package alone 172.513s) — see `apply-progress` for the full per-package breakdown.
- [x] 8.3 Run `golangci-lint cache clean && make lint`; record 0 issues. **Result**: `bin/golangci-lint` bootstrapped via `make lint` (v2.9.0, not previously installed in this worktree), then `bin/golangci-lint cache clean && make lint` — `go vet ./...` clean, `golangci-lint run` → `0 issues.` (2.655s).
- [x] 8.4 Run `make build`; record success. **Result**: `go build -trimpath ./...` exit 0, 0.280s.
- [x] 8.5 Run `make vuln-check`; record its output. **Result**: `govulncheck ./...` → `No vulnerabilities found.`, exit 0 (JSON-mode cross-check: 0 `"finding"` entries against 170 loaded `"osv"` database entries).
- [x] 8.6 Do **not** run `make all` — its fmt step rewrites committed files and fails the substrate guards. Record this constraint, do not execute the command. **Confirmed**: `make all` was not run.
- [x] 8.7 Substrate preservation check (`openspec/AGENTS.md`, NFR-TLS-003): run `git diff --stat -- backend/agent/src/agent/`; the expected result is **empty**. Record the empty diff as evidence, together with `git diff --stat -- backend/ frontend/` showing no other hit. **Result**: both commands produced empty output — confirmed empty by direct inspection, not by absence of a stated result.

**Discharges:** `R-CHT-013` remainder (`S-CHT-122`), `NFR-CHT-002` (`S-CHT-131`)

## Coverage check — every requirement and scenario claimed exactly once

| Requirement | Phase | Scenarios |
|---|---|---|
| `R-CHT-001` | 2 | `S-CHT-001`, `S-CHT-002`, `S-CHT-003`, `S-CHT-004` |
| `R-CHT-002` | 2 | `S-CHT-010`, `S-CHT-011`, `S-CHT-012` |
| `R-CHT-003` | 3 | `S-CHT-020`, `S-CHT-021`, `S-CHT-022`, `S-CHT-023` |
| `R-CHT-004` | 3 | `S-CHT-030`, `S-CHT-031`, `S-CHT-032` |
| `R-CHT-005` | 3 | `S-CHT-040`, `S-CHT-041`, `S-CHT-042` |
| `R-CHT-006` | 1 | `S-CHT-050`, `S-CHT-051`, `S-CHT-052`, `S-CHT-053` |
| `R-CHT-007` | 4 | `S-CHT-060`, `S-CHT-061`, `S-CHT-062` |
| `R-CHT-008` | 4 | `S-CHT-070`, `S-CHT-071` |
| `R-CHT-009` | 4 | `S-CHT-080`, `S-CHT-081`, `S-CHT-082` |
| `R-CHT-010` | 5 | `S-CHT-090`, `S-CHT-091`, `S-CHT-092` |
| `R-CHT-011` | 5 | `S-CHT-100`, `S-CHT-101`, `S-CHT-102` |
| `R-CHT-012` | 5 | `S-CHT-110`, `S-CHT-111`, `S-CHT-112` |
| `R-CHT-013` | 6 + 8 | `S-CHT-120`, `S-CHT-121` (Phase 6); `S-CHT-122` (Phase 8) |
| `NFR-CHT-001` | 6 | `S-CHT-130` |
| `NFR-CHT-002` | 8 | `S-CHT-131` |

**Count check.** 13 `R-CHT-*` + 2 `NFR-CHT-*` = 15 requirement-level IDs, each assigned to exactly one phase (or, for `R-CHT-013`, exactly two phases whose scenario sets are disjoint). 43 `S-CHT-*` scenario IDs (`S-CHT-001`…`S-CHT-004`, `S-CHT-010`…`S-CHT-012`, `S-CHT-020`…`S-CHT-023`, `S-CHT-030`…`S-CHT-032`, `S-CHT-040`…`S-CHT-042`, `S-CHT-050`…`S-CHT-053`, `S-CHT-060`…`S-CHT-062`, `S-CHT-070`, `S-CHT-071`, `S-CHT-080`…`S-CHT-082`, `S-CHT-090`…`S-CHT-092`, `S-CHT-100`…`S-CHT-102`, `S-CHT-110`…`S-CHT-112`, `S-CHT-120`…`S-CHT-122`, `S-CHT-130`, `S-CHT-131` = 4+3+4+3+3+4+3+2+3+3+3+3+3+2 = 43), each claimed by exactly one phase. No ID is orphaned; no ID is claimed twice.
