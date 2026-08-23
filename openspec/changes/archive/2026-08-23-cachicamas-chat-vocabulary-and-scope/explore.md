# Explore — `cachicamas-chat-vocabulary-and-scope` (CH-00)

## Milestone charter (verified: `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:202-240`)

CH-00 is `[decision]`. Deliverable: a decision record in the change folder, cited by every later CH milestone. Acceptance: a reader asks what this archetype's answer is to any Layer 2 seam, and the record answers directly with the injection point, without reading source. Out of scope: implementing any answer (owned by later milestones); renaming "application"→"archetype" in Layer 2's promoted spec (owned by ADR 0009 § D7(a)'s own SDD change). Closes: R-02, R-03, register rows 3, 5, 6.

Closing checklist — seven questions (`:219-225`), each verified traceable below.

## Binding decisions (do not re-litigate)
- **D-1 Evidence gate**: record + delta spec only. No new Go test/guard test. Closing evidence: promoted spec + green uncached `cd backend/agent && go test -race -count=1 ./...`.
- **D-2 Seam breadth**: answer exactly the 11 AG-23 seams (8 frozen + 3 experimental) PLUS a separately labelled gap-findings section against Layer 2's seam set.
- **D-3 Citation defect**: cite the archive where a row only exists there; state the violation in an inconsistency register; forward the repair as a follow-up alongside ADR 0009 § D7(a). Do not attempt repair here.
- **D-4 Spec shape**: ONE capability spec, slug `chat-archetype-contract`, ID prefix `CHT` (collision-free, 0 hits, verified by prior pass — not re-checked this pass; low risk, prefix is novel).

## Question-by-question source map (all citations spot-verified this pass)

### Q1 — Vocabulary mapping
Source: `openspec/specs/agent-contract-vocabulary/spec.md` (promoted, live) — but the 86 `VL2-*` rows are a **placeholder** at `:339`: "[REGISTER CONTINUES WITH 86 ROWS IN 6 CATEGORIES — SEE ARCHIVED DECISION.MD FOR THE COMPLETE SNAPSHOT]". The real rows live ONLY at `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md`. `R-AGV-001` (spec.md:36) and `S-AGV-001` (spec.md:40) explicitly forbid the archive as a citation target — this is D-3's citation defect, confirmed structurally present.

Verified rows (line numbers confirmed against `decision.md`):
- `VL2-COR-02` the loop — line 133
- `VL2-COR-03` the harness — line 134
- `VL2-COR-04` run — **line 135** (NOT :136 as an earlier pass cited — off-by-one, corrected here)
- `VL2-COR-05` turn — **line 136** (NOT :137 as an earlier pass cited — off-by-one, corrected here)
- `VL2-COR-09` upward path — line 140
- `VL2-COR-10` transcript — line 141
- `VL2-COR-12` steering — line 143
- `VL2-COR-14` subagent "the delegated participant" — line 145
- `VL2-EVT-04` message lifecycle — line 165
- `VL2-EVT-10` run outcome — line 171
- `VL2-EVT-11` turn outcome — line 172

**Critical gap, confirmed by grep**: Layer 2 has NO term for "session" as its own concept — `VL2-OUT-07` (`decision.md:252`) explicitly assigns "session persistence" to Layer 3 ("Append-only session records with parent chains, under a Layer 3 application's own storage. Layer 2 exposes a transcript... it never writes one itself"). No general "participant" term exists (`VL2-COR-14` only defines "the delegated participant" = subagent, a different sense). No "part" term (that's Layer 1's `ai-content-parts`, not searched this pass but consistent with prior finding). **The archetype's "conversation" and "participant" vocabulary are coinages, not Layer 2 mappings — the record must say so explicitly.**

### Q2 — Seam-by-seam v1 answers (D-2's 11 seams)
Source: `openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md` § 2, lines 27-53. Framing line 29: "Nine categories; none is empty." Confirmed structure: categories 1,3,4,6,8,9 carry named "Seam —" labels (8 total, since category 3 and 8 each carry two); categories 2,5,7 (steering/interruption, event stream, cost accounting) carry NO seam label. The "### Marked experimental — not frozen" heading is at line 49 exactly.

All 8 frozen + 3 experimental seam citations (injection point, v1 default, line number) verified exact against source text — line numbers 31/35/35/37/41/45/45/47 (frozen) and 51/52/53 (experimental) all confirmed byte-accurate. Go identifiers cross-verified against source:
- `Harness.Provider` — `backend/agent/src/agent/harness.go:53` ✓
- `Harness.Scheduler` — `harness.go:81` ✓
- `Harness.History` — `harness.go:85` ✓
- `Harness.Hooks` — `harness.go:74` ✓
- `Harness.TracerProvider` — `harness.go:131` ✓
- `Harness.Failover` — `harness.go:107` ✓
- `Harness.ContextStrategy` — `harness.go:115` ✓
- `TurnOptions.PreRequestHook` — `backend/agent/src/agent/loop.go:93` ✓ (constraint at decision.md:45 — "Frozen-and-superseded ... it is not removed and MUST NOT be described as removed or deprecated anywhere in this document" — verbatim confirmed)
- `TurnOptions.Tools` — `loop.go:112`; `Registry` iface at `tool.go:267` ✓
- `TurnOptions.PermissionPolicy` — `loop.go:130` ✓
- `DelegationSeam` — `delegation_seam.go:46`, unconstructible per `:18-22` ✓ ("no code outside this package can construct or install a seam")

Spec scenarios binding the record's form (`openspec/specs/agent-layer3-handoff/spec.md`): `S-L3H-028:195` ("every seam names its injection point and its v1 default, and no seam is listed without both"), `S-L3H-029:196` ("the answer is stated directly and does not require reading source"), `:183` ("without reference to files, shells, skills or terminals"). All confirmed verbatim.

**Gap-findings section (D-2's disclosed gaps against Layer 2's own seam set)** — confirmed against `docs/architecture/0001-cachicamas-agent-stack-v2.md:658-683` ("## 6. The twelve seams that must exist now"):
- Seam 3 **Sandbox policy** (`:670`) — no AG-23 counterpart. Rides `PolicySlot any` at `backend/agent/src/agent/tool.go:117`, documented at `:105-106` as "seam 3 of v2 § 6" verbatim.
- Seam 7 **Retry classification** (`:674`) — no AG-23 counterpart.
- `Harness.RetryTiming` (`harness.go:96`) is literally called "the injected clock and wait-function seam" in its own doc comment but is unenumerated in AG-23.
- `Harness.RetryAttempts` (`:94`), `Harness.ContextBudget` (`:120`), `Harness.System` (`:56`) are settable, behaviour-affecting, unenumerated fields.
- `tool.go:265` — the only `ToolSource` mention in the module. **Corrected during the design phase; this exploration note was wrong.** The comment is *not* truncated: the sentence simply wraps, `:265` reading "`ToolSource` port (G6) is AG-20's" and `:266` completing it with "widening." `R-CHT-005` now forbids the record from describing it as truncated, and `spec.md:121` records the claim as checked and false.

### Q3 — Package identity
Header of doc 0005 (`:8`, confirmed verbatim): "**Target packages:** `backend/agent/src/chat/` (the archetype) and `backend/agent/src/cmd/chat/` (its composition root), per [ADR 0005 § D2]." Confirmed by directory listing: neither `backend/agent/src/chat/` nor `backend/agent/src/cmd/` exists yet (0 files matched both globs). Existing sibling packages (confirmed via `doc.go` presence): `agent/`, `agenttest/`, `ai/`, `apptest/`, `handoff/`, `layer3handoff/` — no `chat/`, no `cmd/` anywhere under `src/`.

### Q4 — Database ownership
`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md` § D6, line 152 (confirmed verbatim): "Each business system owns its own tables; no archetype writes to another system's schema." This archetype persists server-side per participant in tables **it owns**; CH-06/CH-07 own the implementation. Doc 0005 register row 3 (per `:711` "Closes: R-04, R-16, register row 3") reconciles doc 0004's home-directory sessions as that archetype's own answer, not a layer rule — consistent cross-reference, not independently re-derived this pass.

### Q5 — Frontend attachment and frozen wire
`openspec/specs/frontend-chat-layer1/spec.md`, REQ-1..REQ-7, confirmed. Frozen wire: `POST /api/agent/turns` → `{turn_id, stream_url}`; `EventSource(stream_url, {withCredentials:true})`; event names `message.start`, `message.delta`, `message.end`, `turn.end`, `error`; cancel `DELETE /api/agent/turns/:id`; error envelope `ApiResult<T>` confirmed verbatim at `frontend/src/lib/api.ts:96-110` (kinds: `validation|conflict|not_found|server|offline`, confirmed at `:90-94`); sanitizer `renderSanitizedMarkdown` confirmed verbatim at `frontend/src/lib/markdown.ts:65-67`. REQ-5 mandates the literal `"backend not wired — see PR for backend wire"` (spec.md:62); doc 0005 register row 4 (per `:626,635` "register row 4... the literal is mandated by a promoted spec") says retiring it needs a recorded spec delta at CH-05.2, never silent deletion.

**CITATION DEFECT FOUND (not previously flagged)**: the auth-guard chain citation `frontend/src/routes/home/index.tsx:39-51` is **WRONG** — repeated three times in the promoted spec itself (`frontend-chat-layer1/spec.md:9, :40, :134`). The actual guard chain (`setSsrCookieHeader → requireAuthRedirect → requireOwnboarding`, `onRequest` handler) lives at **`frontend/src/routes/home/layout.tsx:21-34`** (function body `:30-34`). `home/index.tsx` itself states in its own file-header comment (`:5-6`): "The guard chain lives in this section's layout." This is a pre-existing defect in the promoted `frontend-chat-layer1` spec, inherited (not introduced) by this exploration. CH-00's record should cite the corrected location and flag the spec defect as a follow-up, parallel to D-3's archive-citation defect.

### Q6 — Explicitly deferred capabilities
Doc 0005's "## Explicitly deferred until after v1" table at line **997** (not exactly "line 990" as an earlier pass estimated, but the same table — 7-line discrepancy, non-material). Confirmed 10 rows, each paired with its attaching seam:
- Resumable mid-turn reconnect → the turn's event stream (needs durable ordered log)
- Multi-device/multi-tab delivery → same stream/log
- MCP tool sources → CH-09's tool-source port, ADR 0009 § D4, register row 5
- Sandboxing of tool execution → v2 § 6 seam 3 (v1 answer: "none", stated at CH-00.1)
- Remembered permission decisions → CH-10's policy port
- Provider/model selection by participant → CH-04.1
- Cost and usage display → runtime's accounting seam
- Conversation branching/renaming/deletion/search → CH-06/CH-08
- Websocket transport → CH-03 (research finding 2: one-directional output doesn't justify it)
- Promoting any part into shared Layer 3 code → deferred, no seam yet
- Cross-archetype coordination → ADR 0009 § D3 (a layer above 3, deliberately undesigned)

AG-23's own known-limitations register at `archive/.../decision.md:103-108` confirmed present (4 rows: no coordination tool, failover declines, no compaction policy, abandoned-consumer contract). Hard-rule framing at `:110-112` — **note**: the earlier pass's quoted text "'The register is for design limitations only. A defect … MUST NOT be entered in it.'" attributed to line `:112` is a **paraphrase, not a verbatim quote**. Line 112's actual text: "...it is a crash-class defect reachable through a seam this milestone freezes, not a design limitation with a post-v1 path, and it appears **nowhere** in the register above. Laundering it as a limitation would make this statement false on the day it lands." The substance (a defect must never be entered as a limitation) is correct; the quotation marks around a non-verbatim string should be dropped in the final record.

### Q7 — Substitution rule for "a Layer 3 application"
`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md` § D7(a), lines 169-176, confirmed verbatim: "Until that change lands, 'a Layer 3 application' in shipped Layer 2 artifacts is read as 'a Layer 3 archetype', exactly as ADR 0004's amendment established read-with-substitution before it." Occurrences in promoted specs confirmed by direct grep (matches exactly the cited set, plus two additional unlisted occurrences that don't contradict): `agent-contract-vocabulary/spec.md:146, :153, :335`; `agent-layer3-handoff/spec.md:17, :177, :196` (the two additional occurrences are `agent-layer3-handoff/spec.md:34` — "a **future** Layer 3 application", which an exact-string search misses — and `agent-v1-scope/spec.md:317`. **Both are named here as a correction:** this note originally enumerated only `:34` while asserting "two additional", leaving an open count that the downstream six-item list was then carried forward past. Ground truth is eight occurrences across three promoted specs). Go source uses bare "Layer 3" — not independently re-grepped this pass, consistent with prior finding, low risk.

## Verification log

All citations below were opened at the cited file:line and checked against the quoted/paraphrased text.

| # | Citation | Result |
|---|---|---|
| 1 | Milestone doc 0005:202-240 (charter, 7 questions) | MATCH — verbatim |
| 2 | archive/2026-08-21.../decision.md:27-53 (11-seam enumeration, all line refs) | MATCH — verbatim, all 8 frozen + 3 experimental line numbers exact |
| 3 | decision.md:29 "Nine categories; none is empty" | MATCH — verbatim |
| 4 | decision.md:49 "### Marked experimental — not frozen" | MATCH — verbatim heading |
| 5 | decision.md:45 pre-request-hook "Frozen-and-superseded...MUST NOT be described as removed or deprecated" | MATCH — verbatim |
| 6 | decision.md:103-108 known-limitations register (4 rows) | MATCH — table content confirmed |
| 7 | decision.md:112 "hard rule" quote | **MISMATCH** — paraphrase, not verbatim; substance correct, quotation marks misleading |
| 8 | harness.go:53,56,74,81,85,94,96,107,115,120,131 (all named fields) | MATCH — all 11 field/line pairs exact |
| 9 | loop.go:93,112,130 (PreRequestHook, Tools, PermissionPolicy) | MATCH — exact |
| 10 | tool.go:117 (PolicySlot any), :105-106 (seam 3 of v2 §6), :267 (Registry iface), :265 (ToolSource comment — this row's original "truncated" reading was checked and disproven; see the corrected note above) | MATCH on the line references; the "truncated" characterisation did NOT survive checking |
| 11 | delegation_seam.go:46 (DelegationSeam iface), :18-22 (unconstructible) | MATCH — exact |
| 12 | agent-layer3-handoff/spec.md:17, :177, :183, :195 (S-L3H-028), :196 (S-L3H-029) | MATCH — all verbatim |
| 13 | "Layer 3 application" occurrences: agent-contract-vocabulary/spec.md:146,153,335; agent-layer3-handoff/spec.md:17,177,196 | MATCH on the six cited — but the row's "plus 2 unlisted extra hits" was an **open count, corrected here**: the two are `agent-layer3-handoff/spec.md:34` and `agent-v1-scope/spec.md:317`, making eight across three specs. Leaving them unenumerated is what let `:317` fall out of every downstream list |
| 14 | agent-contract-vocabulary/spec.md:36 (R-AGV-001), :40 (S-AGV-001), :339 (placeholder) | MATCH — verbatim |
| 15 | decision.md (VL2 register) VL2-COR-02:133, COR-03:134, COR-09:140, COR-10:141, COR-12:143, COR-14:145, EVT-04:165, EVT-10:171, EVT-11:172 | MATCH — exact |
| 16 | decision.md VL2-COR-04 run **:136**, VL2-COR-05 turn **:137** | **MISMATCH** — actual lines are 135 and 136 respectively (off by one each); corrected in this record |
| 17 | decision.md:252 VL2-OUT-07 session persistence; no term for "session"/"participant" | MATCH — confirmed by direct grep, gap is real |
| 18 | frontend-chat-layer1/spec.md REQ-1..REQ-7, full wire | MATCH — verbatim |
| 19 | frontend/src/lib/api.ts:96-110 (ApiResult<T>), :90-94 (kinds) | MATCH — exact |
| 20 | frontend/src/lib/markdown.ts:65-67 (renderSanitizedMarkdown) | MATCH — exact |
| 21 | frontend/src/routes/home/index.tsx:39-51 (auth guard chain) | **MISMATCH** — wrong file; chain is not present anywhere in `home/index.tsx` (zero grep hits for all 3 function names). Actual location: `frontend/src/routes/home/layout.tsx:21-34`. This defect is pre-existing in the promoted `frontend-chat-layer1/spec.md` itself (3 occurrences), not introduced by prior exploration passes. |
| 22 | docs/adr/0009-...md:152 (§ D6, table ownership) | MATCH — verbatim. Note for later citers: `:152` is D6's heading; the quoted sentence spans `:154-155` |
| 23 | docs/adr/0009-...md:174-176 (§ D7(a), substitution rule) | MATCH — verbatim |
| 24 | docs/architecture/0001-cachicamas-agent-stack-v2.md:658-683 (12-seam table), :670 (Sandbox), :674 (Retry classification) | MATCH — exact |
| 25 | doc 0005:8 (target packages, ADR 0005 § D2) | MATCH — verbatim |
| 26 | Package directory listing (no `src/chat/`, no `src/cmd/`; existing: agent/agenttest/ai/apptest/handoff/layer3handoff) | MATCH — confirmed by glob, zero hits for chat/cmd |
| 27 | doc 0005 deferred table "near line 990" | MATCH (approximate) — actual heading is at line 997, table content 999-1011; non-material 7-line estimate drift |

## Risks
- Two genuine citation defects found this pass, both material to CH-00's record: (a) VL2-COR-04/COR-05 line numbers off by one each (cosmetic, easy fix); (b) the frontend auth-guard-chain file path is wrong throughout the promoted `frontend-chat-layer1` spec (`home/index.tsx` → should be `home/layout.tsx`) — this is a pre-existing spec defect that predates CH-00 and should be named as a follow-up alongside D-3's archive-citation defect, not silently corrected in place per D-3's own posture (record, don't repair).
- The line-112 "hard rule" quotation in a prior research pass was a paraphrase dressed as a verbatim quote — the record must not carry it in quotation marks.
- Register rows 3/5/6 (doc 0005's own cross-reference numbering) were corroborated via doc 0005's internal cross-references (register row 3 at :711, row 4 at :626/:635, row 5 at :1003/:1027) but their canonical source table was not independently opened this pass — low risk, doc-internal consistency is already strong.

## Ready for proposal
Yes, with two corrections folded into the CH-00 record: (1) fix the two off-by-one VL2 line citations, (2) cite the frontend auth guard chain at `home/layout.tsx:21-34` (not `home/index.tsx:39-51`) and record the promoted-spec defect as a named follow-up, not a silent fix.
