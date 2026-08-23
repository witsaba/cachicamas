```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:fa83a86b0de823e9fc01e88f3a641f6eea1da042e3e873c398340ba7a8051e7e
verdict: fail
blockers: 1
critical_findings: 1
requirements: 15/15
scenarios: 43/43
test_command: "cd backend/agent && go test -race -count=1 ./..."
test_exit_code: 0
test_output_hash: sha256:cd0cc35365f9492908cdb4583c7a5b7c4b3763c9f86a8edf4639c4988e83e056
build_command: "cd backend/agent && make build"
build_exit_code: 0
build_output_hash: sha256:032160733ad8ca369b2abf479f3d46d8c771d95813d8345ca1b3bc67599e7235
```

> **Envelope note — read before the hashes.** `evidence_revision` is this round's own value, recomputed
> by the § 5 recipe over the current bytes. **`test_output_hash` and `build_output_hash` are round 2's
> values, stated here as INHERITED and not re-measured this round** — the byte-identity premise was
> proven first (§ 5) and that premise is the entire warrant for the carry-forward.
>
> **`requirements: 15/15` and `scenarios: 43/43` beside `fail`, and that is not a contradiction.** For
> the fourth time in six rounds the blocker is a **false claim in a durable record that no scenario
> asserts**. Rounds 2, 3 and 5 all failed at full counts on exactly this shape. This round's is worse
> than round 5's in one respect that decides its severity: it sits in `specs/chat-archetype-contract/spec.md`,
> the artifact promoted **verbatim** to `openspec/specs/` at archive, and one of the two sites is inside
> a normative MUST clause of `R-CHT-004`.
>
> Repository HEAD at verification: `d83c23d3e962e4f8c0d8560c873bd9a3c550a819` (= `origin/main`).

# Verify report — round 6 — CH-00 `cachicamas-chat-vocabulary-and-scope`

**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ch00-chat-vocabulary-scope`
**Branch**: `feat/chat-archetype-wave0-ch00` · **HEAD**: `d83c23d3`
**R1**: `fail` (3C) · **R2**: `fail` (2C/7W/7S) · **R3**: `fail` (1C/2W/5S) · **R4**: `fail` (1C/3W/4S) · **R5**: `fail` (1C/0W/5S)
**Mode**: full spec verification · **Store**: hybrid · **Strict TDD**: vacuous and correctly so

Every claim in the brief was re-measured in the bytes. No fix was accepted on its description. Every
sweep below is this report's own. **Round 5's CRITICAL and all five of its SUGGESTIONs are closed.**

---

## 1. Round-5 findings — every one re-measured

| # | Round-5 finding | Status | Command / file:line that settled it |
|---|---|---|---|
| **C-1** | `decision.md:234` and `proposal.md:106` claimed the F-1/F-2/F-3 union was **three** promoted specs; it is four | **CLOSED** | Both sites now say **four** *and* enumerate. Union re-derived from the register's own rows, not from the brief — see § 2. The enumeration is element-for-element identical to my derivation at both sites |
| **S-1** | "the two `agent-*` specs … returns only seven" re-derives to 8 under the literal phrase | **CLOSED** | `decision.md:212` now reads "the loose pattern over **the two specs the cited six live in** returns only seven". Measured: `grep -rn 'Layer 3 application'` over `agent-contract-vocabulary` + `agent-layer3-handoff` → **7**; over `openspec/specs/agent-*/spec.md` → **8**. The new phrase re-derives to 7 and the old one to 8, so the fix is correct in the direction claimed |
| **S-2** | bare `0009:152` at five sites | **CLOSED** | All **nine** `:152` sites now carry `:154-155`: `decision.md:9`, `:132`; `spec.md:6`, `:158`, `:164`; `proposal.md:113`, `:160`; `tasks.md:67`; `explore.md:117`. `grep -rn ':152' \| grep -v '154-155'` → **empty**. The brief's count of nine matches mine exactly |
| **S-3** | `proposal.md:24` two-way framing | **CLOSED** | `:24` now: "distinguish what **maps** onto Layer 2, what is **coined** here, and what is **inherited** from Layer 1 — three provenances, not two". Three-way with the inherited provenance named |
| **S-4** | lint cache artifact | carried, informational | not re-run (premise held, § 5) |
| **S-5** | `decision.md:142` "the fourth `CUSTOM` consumer … named by §2.2" | **CLOSED — and the node list independently verified at source** | § 3 below. I did **not** take my own round-5 reading on trust; `0001:203-291` was re-opened |

---

## 2. The CRITICAL, re-derived from the register's own rows

Derived from `decision.md` § 10's three rows, not from the brief and not from the corrected sentence.

| Defect | Promoted specs it is cited against | Source row |
|---|---|---|
| F-1 | `agent-contract-vocabulary` (`:339`, `:36`, `:40`) | `decision.md:230` |
| F-2 | `agent-contract-vocabulary`, `agent-layer3-handoff`, `agent-v1-scope` | `decision.md:231` — the eight occurrences |
| F-3 | `frontend-chat-layer1` (`:9`, `:40`, `:134`) | `decision.md:232` |

Union = **four**: `agent-contract-vocabulary`, `agent-layer3-handoff`, `agent-v1-scope`,
`frontend-chat-layer1`.

| Site | Now reads | Verdict |
|---|---|---|
| `decision.md:234` | "cited against **four** promoted specs — `agent-contract-vocabulary` (F-1 and F-2), `agent-layer3-handoff` (F-2), `agent-v1-scope` (F-2) and `frontend-chat-layer1` (F-3)" | **TRUE.** Count and enumeration both correct; each spec's defect attribution matches its register row |
| `proposal.md:106` | "recorded as defects against **four** promoted specs — `agent-contract-vocabulary` (F-1, F-2), `agent-layer3-handoff` (F-2), `agent-v1-scope` (F-2), `frontend-chat-layer1` (F-3)" | **TRUE.** Same set, same attributions |
| `decision.md:231`, `proposal.md:93`, `proposal.md:163`, `spec.md:197` | "three specs" about **F-2 alone** | **TRUE and correctly untouched** — F-2 alone does span exactly three. Re-checked all four; none was disturbed by the fix |

**Round 5's C-1 is genuinely closed.** The trailing meta-sentence the fix added is a separate matter —
see **W-1**.

---

## 3. Every claim this round's fixes introduced, checked

The brief named two as most likely wrong. Both were re-derived from the repository.

| Introduced claim | Measurement | Verdict |
|---|---|---|
| `decision.md:234` — the four-spec enumeration | § 2, derived from the register rows | **TRUE** |
| `proposal.md:106` — the same enumeration | § 2 | **TRUE** |
| **`decision.md:142` — "§2.2's frontend subgraph names three nodes, `TUI`, `Print mode` and `CUSTOM["Future: IDE / RPC"]`"** | **`docs/architecture/0001-cachicamas-agent-stack-v2.md:203-291` re-opened at source.** § 2.2 is "Layer view"; its `subgraph FE["Frontends — consume events only"]` at `:207-211` holds **exactly three** nodes: `TUI["TUI"]` (`:208`), `PRINT["Print mode"]` (`:209`), `CUSTOM["Future: IDE / RPC"]` (`:210`) | **TRUE.** Exactly three, exactly those labels |
| `decision.md:142` — "does not itself name this frontend" | none of the three nodes is a web chat; the only in-edges to the FE subgraph are `CSE --> TUI/PRINT/CUSTOM` (`:251-253`) | **TRUE** |
| `decision.md:142` — "the first real occupant of the `CUSTOM` consumer slot that §2.2 reserves" | no prior occupant exists; `CUSTOM`'s label is a forward reservation | **TRUE**, and the ambiguity round 5 flagged is gone — the sentence now discloses what §2.2 does *not* say |
| `decision.md:212` — "the two specs the cited six live in returns only seven" | acv (3) + alh (4) = **7**; measured | **TRUE** |
| `decision.md:212` — the quoted fragment *"exactly the cited set, plus two additional unlisted occurrences that don't contradict"* | **verbatim** at `explore.md:88` | **TRUE** |
| `proposal.md:24` — three provenances | `:24` re-read | **TRUE** |
| the nine `:154-155` sites | `grep -rn ':152' \| grep -v '154-155'` → empty | **TRUE** |
| **the meta-claim `decision.md:234` appended to the fix** — "The four are enumerated rather than counted, so that adding a target to any row **cannot** leave this sentence quietly false" | the sentence carries the numeral **four** three times; a fifth target falsifies it exactly as "three" was falsified | **OVERCLAIMED — W-1** |

**No fix introduced a false repository claim.** The one defect a fix introduced is a claim about the
fix's own robustness, not about the repository.

---

## 4. My own sweeps

Run independently of the brief's, over the whole change folder, `verify-report.md` excluded.

### 4.1 Boundary sweep — `decision.md`, whole document, run LAST

    grep -niE '\b(file|files|shell|shells|skill|skills|terminal|terminals|repository|repositories)\b' decision.md
    → 15, 79

`:15` is the rules-block restatement; `:79` is the verbatim quotation of `agent-layer3-handoff/spec.md:183`,
re-opened at source this round: "The statement MUST be written **without reference to files, shells,
skills or terminals**" — verbatim. **No third hit.** Run after all reading, per the brief.
**`S-CHT-051` PASS**, second round running.

### 4.2 `PreRequestHook` sweep

`grep -n 'PreRequestHook' decision.md` → **`17, 91, 199`**.
`grep -niE '\bremov(e|ed|al|es)\b|deprecat' decision.md` → **the same three lines and no others.**
`:17`'s own scope claim ("§ 3 (seam row 7) and § 9 … the two sections outside this block that name it")
is **exhaustive in both directions**: `:91` is § 3 row 7, `:199` is § 9. Archive `:45` re-opened —
"this field is kept, unamended in behavior, carries no deprecation marker … MUST NOT be described as
removed or deprecated". `agent-v1-scope/spec.md:317` re-opened — "frozen-and-superseded with a post-v1
removal path", the phrase `:199` quotes, verbatim. `S-CHT-052`, `S-CHT-053` **PASS**.

### 4.3 Aggregate-count sweep — the shape the brief asked for

Every number-word/digit followed by a countable noun, across the whole change folder — **147 hits**,
each resolved. The ones that could go stale, re-derived against their source rather than against
another artifact in the change:

| Claim | Site(s) | Measured this round | Verdict |
|---|---|---|---|
| eleven seams, 8 frozen + 3 experimental | `decision.md:81`, `:103`; `spec.md:84`; `design.md:85`; `proposal.md:33`, `:60` | `sed -n '31,53p' archive/…/decision.md \| grep -oE '\*\*Seam — [^*]*\*\*'` → **11**, listed in the exact order § 3 uses; `Injection point` → **11**; experimental heading at `:49` | **TRUE** |
| AG-23's known-limitations register has **four** rows | `decision.md:180`, `design.md:35` | header `:103`, separator `:104`, data rows `:105-108` → **4**, and the four names match `decision.md:180`'s gloss one for one | **TRUE** |
| `V-REQ-02` cited by **exactly four** other promoted specs | `decision.md:69`, `:71` | `grep -rl 'V-REQ-02' openspec/specs/` → **5 files**, minus the definer (`ai-contract-vocabulary`) = **4**: `ai-message-roles`, `ai-content-parts`, `ai-tool-messages`, `ai-cache-breakpoints`. Named set identical | **TRUE** |
| "five promoted specs contain the identifier, four excluding the one that defines it" | `decision.md:69` | 5 / 4 | **TRUE** |
| eight occurrences / three specs (F-2) | `decision.md:192-199`, `:204-205`, `:220`, `:231`; `proposal.md:93`, `:163`; `tasks.md:75`; `explore.md:88`, `:108` | loose **8 hits / 3 files**, exact **7 / 3**, differing only at `alh:34`. Every one of the nine sites agrees element for element | **TRUE** |
| 19 promoted specs open with `> **Binding vocabulary**` | `tasks.md:51`, `design.md:21` | `grep -rl` → **19** | **TRUE** |
| 11 deferral rows reproduced from `0005:997-1011` | `decision.md:164-178`, `spec.md:183` | source rows `:1001-1011` → **11**; § 8 carries 11 in identical order | **TRUE** |
| "86 ROWS IN 6 CATEGORIES" | `decision.md:230` | verbatim at `agent-contract-vocabulary/spec.md:339` | **TRUE** |
| `frontend-chat-layer1` `REQ-1`…`REQ-7` | `decision.md:142-152` | `grep -oE 'REQ-[0-9]+' \| sort -u` → exactly REQ-1…REQ-7 | **TRUE** |
| **three nodes in `0001` §2.2** | `decision.md:142` | § 3 | **TRUE** |
| **the union of F-1/F-2/F-3 is four** | `decision.md:234`, `proposal.md:106` | § 2 | **TRUE** |
| 43 scenarios / 15 requirements / 262 spec lines | `tasks.md:3`, `:12`, `:130` | 43 / 15 / 262 | **TRUE** |

**No surviving false aggregate count.** Round 5's blocker class is clean.

### 4.4 Open-count sweep

    grep -rniE 'among others|and others|etc\.|plus [a-z0-9]+ (additional|more|further|other)|and more|\bothers\b|\bat least\b|\bseveral\b|\bvarious\b|\ba few\b|\bnot limited to\b|\bamong them\b|\band so on\b|\bor so\b|\bapproximately\b|~[0-9]'

Every hit opened. `explore.md:88`, `decision.md:212`/`:214` quote the closed defect to name it;
`tasks.md:13-16`/`:27` are line-count *estimates*, correctly marked `~`; `decision.md:88` ("at least one
tool") is a lower bound on CH-09's deliverable; `spec.md:197` ("at least one occurrence carries an
intervening modifier") is a true lower bound where the exact figure is one; `proposal.md:86` is rhetoric.
**No surviving unclosed open count.**

### 4.5 Exhaustive-binary sweep

    grep -rnoiE '\bthe two (specs|tables|blocks|kinds|categories|classes|things|halves|ways|sections|constraints|lower layers|patterns|sanctioned)[a-z ]*'

Seven hits, all opened, all correct: `decision.md:17` (verified exhaustive in § 4.2), `:201` ("the two
patterns differ by exactly one occurrence" — measured true), `:212` (S-1's fix, measured true),
`tasks.md:42`, `proposal.md:130`, `spec.md:86`, `spec.md:88`. **No surviving two-way claim over a
three-way population.** S-3's residue is gone.

### 4.6 Self-referential absolute-claim sweep

`grep -nic 'truncat' decision.md` → **0** (`S-CHT-042` PASS). `tasks.md:92`'s "no other text on that
line changes" and `tasks.md:106`'s "no other hit" both re-measured against the live tree: doc 0005 now
carries **0** occurrences of "Not started" and **0** of "0 of 12"; `git diff -- backend/ frontend/` is
empty. `decision.md:15`'s "named nowhere below except…" is § 4.1. **All hold.**

### 4.7 Citation-resolution sweep — **this is the sweep that found the blocker**

Every `file:line` citation in the change was opened at its target and read against the claim it
carries. This is the sweep no earlier round ran exhaustively, and it is the one the brief's fourth
request ("anything the spec's requirements do not reach but `spec.md:121` does") points at.

**Resolved and correct** — all 89 `0005:NNN` citations except one target (below); ADR `0009:152`,
`:154-155`, `:169-176`, `:174-176`; ADR `0005:198-218`; `agent-layer3-handoff/spec.md:183`, `:195`,
`:196`; `agent-contract-vocabulary/spec.md:36`, `:40`, `:146`, `:153`, `:335`, `:339`;
`agent-v1-scope/spec.md:317`; `ai-contract-vocabulary/spec.md:111`, `:313`; `frontend-chat-layer1/spec.md:9`,
`:40`, `:62`, `:134`; `frontend/src/routes/home/layout.tsx:15-17`, `:21-29`, `:30`, `:31-33`;
`home/index.tsx:5-6`; AG-14 archive `:133`, `:134`, `:135`, `:136`, `:140`, `:141`, `:143`, `:145`,
`:165`, `:171`, `:172`, `:252`; AG-23 archive `:31-53`, `:45`, `:49`, `:103-108`, `:110`;
`0001:658-683`, `:670`, `:674`; `harness.go:53`, `:56`, `:74`, `:81`, `:85`, `:94`, `:96-97`, `:100`,
`:107`, `:115`, `:120`, `:131`; `loop.go:93`, `:112`, `:130`; `tool.go:117`, `:265-266`, `:267`;
`delegation_seam.go:18-19`, `:46`. **Every one resolves to exactly what the citing sentence claims.**

**Two do not — see C-1.** `0005:117` (4 sites, one of them a normative MUST in the promoted spec) and
`agent-concurrency-hardening/spec.md:13` (1 site, the promoted spec's header).

---

## 5. Evidence gate — premise proven first, carry-forward stated explicitly

| Check | Command | Result |
|---|---|---|
| HEAD identity | `git rev-parse HEAD` / `origin/main` | both `d83c23d3e962e4f8c0d8560c873bd9a3c550a819` |
| **Source byte-identity** | `git diff --numstat d83c23d3 -- backend/ frontend/ openspec/specs/` | **empty** |
| Whole-worktree name-status | `git status --porcelain -uall` | one `M docs/…0005-…md`; the rest is the untracked change folder (7 files). **Nothing else** |
| `docs/` diff | `git diff --numstat d83c23d3 -- docs/` | **`2 2`** — one file, 2 insertions, 2 deletions. Diff read in full: `:3` `Not started`/`0 of 12` → `In progress`/`1 of 12`; `:980` `- [ ]` → `- [x]`. **No third line** |
| No line-number drift into doc 0005 | both edits are in-place substitutions | every `0005:NNN` citation resolves at the same line in the base blob — confirmed for `:115`, `:116`, `:117` via `git show d83c23d3:docs/…` |

**The premise holds: only markdown inside the change folder moved. Round 2's green gate is therefore
INHERITED, and this report states that explicitly rather than reusing it silently.** The suite, lint,
build and vulnerability gates were **not re-executed this round**, because nothing they measure moved —
none of the changed bytes is compiled, linted or scanned.

| Gate | Command | Result (round 2, **inherited**) |
|---|---|---|
| Race suite | `cd backend/agent && go test -race -count=1 ./...` | exit **0**; **2:56.40** wall clock (`go clean -testcache` first); 0 `(cached)`; 0 `FAIL` |
| Build | `cd backend/agent && make build` | `go build -trimpath ./...` exit **0** |
| Lint | `bin/golangci-lint cache clean && make lint` | `go vet` clean; **`0 issues.`**, exit **0** |
| Vulnerabilities | `cd backend/agent && make vuln-check` | exit **0**; 0 `"finding"` against 170 `"osv"` entries |
| `make all` | — | **NOT RUN** — its fmt step rewrites committed files |

Evidence-revision recipe (this round's own value, over current bytes):

    EX=openspec/changes/cachicamas-chat-vocabulary-and-scope/verify-report.md
    { git diff HEAD --no-color -- . ":(exclude)$EX";
      git ls-files --others --exclude-standard -- openspec/changes/cachicamas-chat-vocabulary-and-scope \
      | grep -v verify-report.md | sort \
      | while read f; do git diff --no-index --no-color /dev/null "$f" || true; done; } | shasum -a 256

### TDD compliance (Strict TDD active)

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ✅ | `apply-progress` (Engram #3705) carries the vacuous-TDD posture |
| RED/GREEN | ➖ vacuous | CH-00.1 is `[decision]`; no production behaviour exists to drive red-then-green (D-1) |
| Test files created | 0 — and correctly 0 | `R-CHT-007` proves the package a guard would protect does not exist (re-checked on disk) |
| Assertion quality / layer distribution | ➖ N/A | No test file created or modified |

---

## 6. Per-requirement findings — all 15 re-walked, all 43 scenarios

### `R-CHT-001` — three structurally distinct vocabulary blocks — **PASS**

§ 2.1 carries `Maps onto` with exactly one `VL2-*` id plus citation in all 10 rows; every one of the
ten was re-resolved at the AG-14 archive this round (`:133`, `:134`, `:135`, `:136`, `:140`, `:141`,
`:143`, `:165`, `:171`, `:172`) and each term cell matches the archived row's term exactly
(`S-CHT-002` PASS). § 2.2 has no `Maps onto` column; both rows cite their `VL2-*` id as evidence of
absence and state the absence (`S-CHT-003` PASS). § 2.3 carries `Inherited from` with exactly one `V-*`
id and its citation, no `Maps onto` column, and states both negative arguments (`S-CHT-004` PASS —
`ai-contract-vocabulary/spec.md:111` re-opened, a register row defining **message**). `V-OUT-02`'s
quoted clause at `:313` — "Layer 1 owns the unit, never the collection, its ordering across turns, or
its repair" — **verbatim**. Noun exhaustiveness re-probed on current bytes: no new archetype noun
introduced. `S-CHT-001` PASS.

### `R-CHT-002` — coinages and the three coinage-forcing absences — **PASS**

`decision.md:63` answers "no" in its own sentence for a session, a human participant and a "part",
quoting `VL2-OUT-07` as an exclusion assigned to Layer 3 (`S-CHT-012` PASS). Conversation states the
1-to-N cardinality against `VL2-COR-04` (`S-CHT-010` PASS); participant states `VL2-COR-14` is the
delegated-subagent sense (`S-CHT-011` PASS). All three `VL2-*` citations re-opened at the archive:
`:135` **run**, `:145` **subagent** ("The delegated participant"), `:252` **session persistence** —
all exact.

### `R-CHT-003` — one row per AG-23 seam, both halves, explicit status — **PASS**

Re-derived from the frozen source by occurrence: `sed -n '31,53p'` → **exactly 11** `**Seam — …**`
markers and **exactly 11** `Injection point` occurrences. The eleven names, in source order, are
identical to § 3's rows 1–11 (`S-CHT-020` PASS). Every `Injection point` and `v1 answer` cell non-empty
(`S-CHT-021` PASS). Status split **8 frozen / 3 experimental — not frozen**, matching the archived
heading at `:49` (`S-CHT-022` PASS). `S-CHT-023`'s planted-omission mechanism — numbering plus source
order — intact. PASS.

### `R-CHT-004` — every answer stated directly; empty answers written as answers — **PASS on the record; see C-1 on the requirement's own citation**

Whole `v1 answer` column re-scanned: no cell is "none", "n/a", "-", empty, a bare path or a bare Go
identifier (`S-CHT-031` PASS). Rows 6, 7, 9, 10, 11 each state what empty means operationally, where it
is injected, and that no milestone owns it with the reason (`S-CHT-032` PASS).

**`S-CHT-030` re-tested on a seam no earlier round used — row 8, the tracing-API provider**, reading
`decision.md` only: injection point `Harness.TracerProvider`, an optional field on the run value, nil
resolving to the tracing API's own no-op provider; v1 answer, CH-04.1 is the only package permitted to
install the OpenTelemetry SDK and wires a real provider into that field, everything below the
composition root passing it through unmodified. Both halves, no Go source opened. PASS.

**The record satisfies every MUST of this requirement.** The requirement's *own text* cites `0005:117`
for the mandated form, and `0005:117` is a different statement — **C-1**. No scenario asserts the
citation, so `S-CHT-030`/`031`/`032` all pass and the requirement scores PASS; the defect is in the
normative text that will be promoted.

### `R-CHT-005` — gap findings structurally incapable of reading as seam answers — **PASS**

Heading, opening sentence and column set compliant — § 4's columns are `Finding | Where Layer 2 names
it | AG-23 status | Disposition`, with no `Injection point` and no `v1 answer` (`S-CHT-040`,
`S-CHT-041` PASS). Every gap-finding citation re-resolved this round: `0001:670` is v2 § 6 seam 3
**Sandbox policy**, `:674` is seam 7 **Retry classification**, `tool.go:117` is `type PolicySlot any`,
`harness.go:96-97` is the doc comment calling it "the injected clock and wait-function seam" with the
field at `:100`, and `:94`/`:120`/`:56` are `RetryAttempts`/`ContextBudget`/`System`. All exact.
`grep -nic 'truncat' decision.md` → **0**; `tool.go:265-266` re-opened — the sentence wraps across the
two lines and completes with "AG-20's widening", exactly as `:114` describes it. `S-CHT-042` PASS.

### `R-CHT-006` — the two inherited form constraints — **PASS**

**(a)** Boundary grep (§ 4.1) → `15, 79` only, run last. `S-CHT-050` PASS — the rules block restates
both constraints in the record's own text, each with its citation. `S-CHT-051` PASS.
**(b)** § 4.2: `S-CHT-052` PASS on both conjuncts; `S-CHT-053` PASS — the prohibition is checked rather
than assumed. **Second consecutive round at PASS**, after three rounds of unrelated edits breaking it.

### `R-CHT-007` — name, package path, composition root, and their non-existence — **PASS**

`decision.md:122` names `cachicamas_chat`, `backend/agent/src/chat/`, `backend/agent/src/cmd/chat/`,
citing `0005:8` (re-opened — the **Target packages** header line, exact) and ADR 0005 § D2
(`S-CHT-060` PASS). `:124` states neither exists and there is no `backend/agent/src/cmd/` of any kind,
naming CH-01.1 (`S-CHT-061` PASS). **Re-checked on disk**: `backend/agent/src/` = `agent agenttest ai
apptest handoff layer3handoff`; no `chat`, no `cmd`. `S-CHT-062` PASS.

### `R-CHT-008` — persistence names the owner — **PASS**

`:132` answers both halves and names the owner. ADR 0009 re-opened at source this round: `:152` is the
heading `### D6 — Each business system owns its own tables`; the quoted sentence spans `:154-155`
("**Data ownership is per system…** Each business system owns its own" / "tables; no archetype writes
to another system's schema."). `S-CHT-070`, `S-CHT-071` PASS. **S-2 is closed** — all nine citing sites
now carry the sub-range.

### `R-CHT-009` — attaching frontend and the frozen wire — **PASS**

`REQ-1`…`REQ-7` enumerated and marked `Inherited`; the promoted spec's ID set re-measured as exactly
REQ-1…REQ-7 (`S-CHT-080` PASS). Auth chain re-resolved at source: `layout.tsx:15-17` are the three
imports, `:21-29` the doc comment ("The canonical guard chain, unchanged since it was written"), `:30`
`export const onRequest: RequestHandler = async (event) => {`, `:31-33` `setSsrCookieHeader` →
`requireAuthRedirect` → `await requireOwnboarding`. Every cited line exact. F-3's other side
re-confirmed: `frontend-chat-layer1/spec.md:9`, `:40`, `:134` all name `home/index.tsx:39-51`, and
`home/index.tsx` has **0** occurrences of the three guard names, its own `:5-6` saying "The guard chain
lives in this section's layout." `S-CHT-081` PASS. `:156` proposes no frozen-wire change and pins
`REQ-5`'s retirement to CH-05.2, with `frontend-chat-layer1/spec.md:62` re-opened as the mandating
line (`S-CHT-082` PASS). **S-5 closed** — the `CUSTOM`-slot phrasing is now correct and self-disclosing
(§ 3).

### `R-CHT-010` — deferrals carry their seam; no defect laundered — **PASS**

`0005:997-1011` re-read: heading `:997`, header `:999`, separator `:1000`, **11 capability rows**
`:1001-1011`; § 8 carries 11 in identical order (`S-CHT-090` PASS). Every row carries a `Seam where it
attaches later` cell (`S-CHT-091` PASS). AG-23's known-limitations register — re-opened, header `:103`,
**4 rows** `:105-108` — is cited at `:180`, not reproduced, and its four names match `:180`'s gloss one
for one; no `F-*` row appears in the deferral register (`S-CHT-092` PASS). The § 4.3 rule quotation at
`:182` re-checked against the archive: the heading is at `:110` as cited, and the quoted text at `:112`
is **verbatim**.

### `R-CHT-011` — the substitution rule and its artifact list — **PASS**

Independently re-derived: `grep -rn 'Layer 3 application' openspec/specs/` → **8 hits / 3 files**,
element-for-element identical to `:192-199`'s list (`S-CHT-100` PASS). The exact string → **7 / 3**,
differing only at `alh:34`, exactly as `:201-208` states. ADR 0009 `:174-176` re-opened — the quotation
at `:218` is **verbatim**. `:220` states every citation of one of the eight is read under the
substitution rule (`S-CHT-101` PASS). `git diff d83c23d3 -- openspec/specs/` is **empty**
(`S-CHT-102` PASS). **S-1 closed** — `:212`'s phrase now re-derives to 7 rather than 8.

### `R-CHT-012` — the inconsistency register — **PASS**

F-1/F-2/F-3 each cite both sides with a disposition, and every cited line was opened this round:
`agent-contract-vocabulary/spec.md:339` is the placeholder verbatim, `:36` is `R-AGV-001`, `:40` is
`S-AGV-001` (both forbidding the archive as a citation target) (`S-CHT-110` PASS). The header at `:224`
states verbatim "Inconsistency register — recorded, not repaired. No promoted spec is modified by this
change." and `proposal.md:106` opens "**None.**" (`S-CHT-111` PASS). F-1/F-2 name the shared landing
spot with "closing F-2 completely also requires the other two"; F-3 carries the `layout.tsx` location
(`S-CHT-112` PASS). **Round 5's C-1 is closed here** (§ 2). The meta-sentence appended to the fix is
**W-1**.

### `R-CHT-013` — the record answers any seam question directly — **PASS**

`decision.md:255` mirrors `0005:210` word for word — both re-opened and compared token by token.
`S-CHT-120` re-tested above on a fresh seam (row 8). `S-CHT-121` PASS — § 11's table names the
answering section for all seven questions plus Q2b, and `0005:219-225` was re-opened to confirm the
seven questions are the ones the table walks. `S-CHT-122` PASS — see § 5.

### `NFR-CHT-001` — the record is self-sufficient — **PASS**

Tested with `spec.md` set aside and no source opened. Q1 resolves for all four nouns with provenance:
conversation → coined (cardinality vs `VL2-COR-04`); turn → mapped (`VL2-COR-05`); message → inherited
(`V-REQ-02`); participant → coined (`VL2-COR-14` is a different sense). Q2 through Q7 close from
§§ 3–9 alone. `S-CHT-130` PASS.

### `NFR-CHT-002` — the evidence gate — **PASS**

`S-CHT-131` PASS — see § 5. The recorded evidence is an uncached `-count=1` run at 2:56.40 wall clock
with zero `(cached)` tokens, `go clean -testcache` run first, and `make all` not run.

---

## 7. Issues

### CRITICAL

#### C-1 — two citations in the **promoted** spec resolve to lines that do not carry the claim, and one of them is a normative MUST

Found by § 4.7. Neither is a drift: both were wrong when written, proven against the base blob.

**(a) `0005:117` — cited 4 times, once inside a MUST that will be promoted verbatim.**

`specs/chat-archetype-contract/spec.md:103`, the second paragraph of `R-CHT-004`:

> "A seam whose v1 answer is deliberately empty MUST be written as a **full sentence in the `0005:117`
> form** stating (i) what empty means operationally for this archetype, (ii) where it is injected, and
> (iii) which milestone ends the emptiness…"

`docs/architecture/milestones/0005-…md:117` reads, in full:

> "- **"It works in the browser" is not the exit condition of wave 1.** Wave 1 exits when a *real* turn
> streams from a *real* provider through the archetype into the page. A green frontend suite against a
> fake proves the client, which `frontend-chat-layer1` already proved."

It is the **third** of doc 0005's three wording traps and says nothing about the form of an empty seam
answer. The content the MUST describes is at **`0005:116`** — *"**"Simple" does not license an unfilled
seam.** A chat with no tools still has a tool-source seam, and its v1 answer is *an empty source,
stated*"* — and the three-part exemplar is at **`0005:213`**, which the same clause already cites
correctly one sentence later.

Confirmed un-drifted: `git show d83c23d3:docs/…0005-…md` gives the identical `:115`/`:116`/`:117`, and
this branch's only `docs/` edit is two in-place substitutions at `:3` and `:980` (§ 5), so no line moved.

The same wrong target appears at **`design.md:25`** and **`tasks.md:58`** (both for the empty-answer
form), and at **`design.md:18`** for a *different* claim — the phrase "the archetype's model of one",
which is at **`0005:115`**, not `:117`. `decision.md:60` cites that same phrase to `:115` **correctly**,
so the change contradicts itself on where that quotation lives.

**(b) `agent-concurrency-hardening/spec.md:13` — cited for a convention that line contradicts.**

`specs/chat-archetype-contract/spec.md:9`:

> "The header states **ranges and never totals** — a total is defended by no test and goes silently
> false on the next append (`agent-observability-boundary/spec.md:9`, `agent-concurrency-hardening/spec.md:13`)."

`agent-concurrency-hardening/spec.md:13` is the archive amendment line, and it **states totals**:
"8 requirements (`R-CNH-001`…`R-CNH-008`), 17 scenarios (`S-CNH-001`…`S-CNH-017`) and 5 non-functional
requirements". It is an instance of the thing the sentence says the precedent avoids.

The precedent this header was copied from settles the correct target by itself.
`agent-observability-boundary/spec.md:9` — the other citation, and the one that *is* correct — carries
the identical sentence and cites **`agent-concurrency-hardening/spec.md:8`**, the "Allocated IDs" line.
So this change transcribed the precedent and shifted its citation from `:8` to `:13`.

**Why CRITICAL rather than a warning, stated honestly:**

1. **It is false against the repository, and it is in the artifact that becomes normative.**
   `specs/chat-archetype-contract/spec.md:5` states this file "is promoted verbatim to
   `openspec/specs/chat-archetype-contract/spec.md` at archive". After archive it is immutable and, per
   `proposal.md:142`, correcting it requires an amending SDD change rather than a revert.
2. **One site is a MUST that is unresolvable at its own cited target.** A later CH milestone author
   discharging `R-CHT-004` opens `0005:117` looking for "the form" and finds a statement about wave-1
   exit conditions. The requirement instructs, and the instruction does not resolve.
3. **`spec.md:121` states this change's governing principle** — *"a false claim MUST NOT enter a durable
   record."* Round 5 applied that principle to `decision.md`; it applies with more force to the spec.
4. **This change's own verification precedent.** Rounds 2, 3 and 5 all failed at full requirement and
   scenario counts on a false claim no scenario asserted. This is the same shape, one artifact further
   downstream.
5. **It is the shape the change itself diagnoses.** `decision.md` § 9 is a case study in a citation
   carried forward without being re-resolved. Four of the five wrong sites are copies of one another.

It does **not** break a MUST and does **not** fail a scenario — requirements are **15/15** and scenarios
**43/43**. Stated plainly rather than inflated: no `S-CHT-*` asserts where a citation points.

**Required change — five sites:**

| File:line | Change |
|---|---|
| `specs/chat-archetype-contract/spec.md:103` | "`0005:117` form" → **`0005:116`** (or drop the first citation and rely on `0005:213`, already correct in the same clause). **Promoted — this is the blocking one** |
| `specs/chat-archetype-contract/spec.md:9` | `agent-concurrency-hardening/spec.md:13` → **`:8`**, matching `agent-observability-boundary/spec.md:9`'s own citation. **Promoted** |
| `design.md:25` | `0005:117` → **`0005:116`** |
| `tasks.md:58` | `0005:117` → **`0005:116`** |
| `design.md:18` | `0005:117` → **`0005:115`**, agreeing with `decision.md:60` |

Then re-run the § 4.1 boundary grep — four consecutive rounds have seen an unrelated edit break
`S-CHT-051`, and two of these sites are prose.

**Do not** touch `decision.md` for this finding: **`decision.md` carries no instance.** Every one of its
citations resolved (§ 4.7). The record is clean; the spec, design and tasks are not.

### WARNING

#### W-1 — the C-1 fix's own closing sentence overclaims what enumeration buys

`decision.md:234`, appended by this round's fix:

> "The four are **enumerated rather than counted**, so that adding a target to any row **cannot** leave
> this sentence quietly false: that is exactly how it went false once…"

Two problems, neither about the repository:

1. The sentence *is* counted — the numeral "four" appears in it three times. It is enumerated **and**
   counted, not "rather than".
2. "cannot leave this sentence quietly false" is stronger than what enumeration delivers. A fifth
   target falsifies "four" exactly as `agent-v1-scope` falsified "three". What the enumeration buys is
   that the shortfall is **visible against the rows three lines above**, not that it cannot happen. And
   § 9 of this same record is the change's own case study in an *enumeration* going short silently.

The factual content of `:234` is correct (§ 2) and a reader planning the repair gets the right four
specs, so this blocks nothing on its own. But `spec.md:9`'s convention line — "a total is defended by no
test and goes silently false on the next append" — is this change's own rule, and `:234` states a total
while claiming immunity from it.

**Suggested wording:** "The four are enumerated as well as counted, so a later reader can check the list
against the rows above rather than trusting the numeral — which is how the count went false once, when
`agent-v1-scope` joined F-2 and nothing re-derived the union."

### SUGGESTION

- **S-4** (carried, rounds 2–5) — the round-1 `revive var-naming` lint finding did not reproduce on
  round 2's cold run. Recorded so a later reader does not rediscover it as new. Not re-run; the § 5
  premise makes a re-run vacuous.
- **S-6** (new) — **surfaced by the S-5 fix.** `decision.md:142` now correctly states that `0001` §2.2
  "does not itself name this frontend". The promoted `frontend-chat-layer1/spec.md:40` says the opposite:
  *"Doc 0001 §2.2 places the chat as an authed consumer."* §2.2 names no chat and mentions no auth; its
  only upward arrow is a permission decision from `TUI` (`0001:255`). This is a fourth promoted-spec
  inaccuracy, in the same file as F-3. `R-CHT-012` binds the register to exactly F-1/F-2/F-3 "as
  enumerated in this change's `proposal.md` follow-up table", so adding an F-4 would require amending
  `proposal.md` and `spec.md:209` too — disproportionate for a rationale sentence. **Recommend
  forwarding it to CH-05, which already owns a `frontend-chat-layer1` delta (`REQ-5`, CH-05.2), rather
  than opening the register here.**
- **S-7** (new) — `proposal.md:22` presents *"The tool source is empty in v1, **injected at the registry
  seam on the run value**, and CH-09 is where it stops being empty"* in quotation marks immediately after
  citing `0005:213`. The charter's actual words are "injected at **session construction**". The
  substitution is an improvement — AG-23's seam is `TurnOptions.Tools`, a per-run field (`loop.go:112`),
  not session construction — but it is an unmarked correction of a quoted source, in a change whose
  whole discipline is that a correction is annotated. `spec.md:103` handles the same exemplar safely, by
  quoting only the first half and paraphrasing the second. Fix: mark it as corrected, in the
  `explore.md:88` / `proposal.md:93` house style.
- **S-8** (new, minor) — `decision.md:8` closes "doc 0005's own inconsistency-register rows **3, 5, 6**
  (`0005:96-99`)". That range is `:96`–`:99` = rows **3, 4, 5, 6**; row 4 is included by the range and
  excluded by the claim. Precise form: `0005:96`, `:98`, `:99`.

---

## 8. Task completion and ID hygiene

| Check | Result |
|---|---|
| Unchecked tasks | **0** — `grep -c "^- \[ \]" tasks.md` → 0 |
| Checked tasks | **31** |
| Scenario IDs defined in spec | **43** |
| Requirement IDs defined in spec | **15** — `grep -cE '^### (R\|NFR)-CHT-'` → 15 (13 + 2) |
| `spec.md` line count | **262** — matches `tasks.md:12` |
| `tasks.md:130` arithmetic | `4+3+4+3+3+4+3+2+3+3+3+3+3+2` = **43** ✓; ID enumeration matches the spec's set exactly |
| In spec, not discharged by a ticked task | **none** (`comm` returns only the header's range tokens `S-CHT-0`, `S-CHT-199`) |
| In tasks, not defined in spec | **none** |
| Double-claimed IDs | **none** — `R-CHT-013` splits across Phases 6 and 8 over disjoint scenario sets |
| `CHT` prefix collision | **0** hits across `openspec/specs/` and all other `openspec/changes/` — re-measured |
| `tasks.md:42` end-state claim | **TRUE** — "only the two sanctioned occurrences … remain" matches § 4.1 |
| `tasks.md:51` "19 promoted specs" | **TRUE** — re-measured → 19 |
| `tasks.md:92`, `:106` absolute claims | **TRUE** — re-measured against the live tree (§ 4.6) |
| Coverage-table hygiene | **CLEAN** |
| **`tasks.md:58` content** | ticked, and its own text carries the `0005:117` error — **C-1** |

---

## 9. Verdict

**FAIL.** 1 CRITICAL, 1 WARNING, 4 SUGGESTION. Requirements **15/15**; scenarios **43/43**.

**Round 5's blocker is genuinely closed, and so is every one of its five SUGGESTIONs.** The union is
four, enumerated, and each spec's defect attribution matches its register row — re-derived from the rows
rather than from the brief. The two claims the brief flagged as most likely wrong were both checked at
source and are both **true**: `0001` §2.2's frontend subgraph holds exactly three nodes, `TUI`,
`Print mode` and `CUSTOM["Future: IDE / RPC"]`, none of them this frontend; and "the two specs the cited
six live in" re-derives to seven where the old phrase re-derived to eight. The nine `:154-155` sites are
complete, `proposal.md:24` is three-way, and the boundary grep over `decision.md` measures exactly
`15, 79`, run last after all reading.

It is not archivable for one reason, and it is again a new one, found by a sweep no earlier round ran to
exhaustion:

**Five citations point at lines that do not carry the claim, and two of them are in the spec that gets
promoted verbatim.** `0005:117` is doc 0005's *wave-1 exit condition* trap, cited four times as the
source of the mandated form for a deliberately empty seam answer — including inside `R-CHT-004`'s own
MUST at `spec.md:103`. The content is at `0005:116`, and the three-part exemplar at `0005:213` is
already cited correctly one sentence later. Separately, `spec.md:9` cites
`agent-concurrency-hardening/spec.md:13` — a line that **states totals** — as precedent for stating
"ranges and never totals"; the precedent this header was copied from,
`agent-observability-boundary/spec.md:9`, cites `:8` for exactly that claim, so the citation was shifted
in transcription. Neither is drift: the base blob carries the identical lines.

`decision.md` itself is clean — every one of its citations resolved. That is worth saying plainly,
because five rounds of pressure have been applied to the record and this round's defect is in the three
artifacts around it.

**Exactly what must change before archive:**

| File:line | Change | Severity |
|---|---|---|
| `specs/chat-archetype-contract/spec.md:103` | `0005:117` → `0005:116` — **inside a promoted MUST** | **CRITICAL** |
| `specs/chat-archetype-contract/spec.md:9` | `agent-concurrency-hardening/spec.md:13` → `:8` — **promoted** | **CRITICAL** |
| `design.md:25`, `tasks.md:58` | `0005:117` → `0005:116` | **CRITICAL** (same claim) |
| `design.md:18` | `0005:117` → `0005:115`, agreeing with `decision.md:60` | **CRITICAL** (same claim) |
| `decision.md:234` | "enumerated rather than counted … cannot" → "as well as counted … so a later reader can check the list against the rows above" | WARNING |
| — | forward `frontend-chat-layer1/spec.md:40` ("Doc 0001 §2.2 places the chat as an authed consumer") to CH-05 rather than opening the register | SUGGESTION |
| `proposal.md:22` | annotate the corrected exemplar instead of presenting it as a quotation of `0005:213` | SUGGESTION |
| `decision.md:8` | `0005:96-99` → `0005:96`, `:98`, `:99` | SUGGESTION |

None of these touches `backend/`, `frontend/`, or any promoted spec, so the substrate guarantees and the
inherited evidence gate hold unchanged once the artifacts are corrected and re-verified. **The CRITICAL
is a line-number edit at five sites, two of them in the promoted spec.** After it, re-run the § 4.1
boundary grep — four consecutive rounds have seen an unrelated edit break `S-CHT-051`.

**One method note for round 7.** § 4.7 — opening every cited target and reading it against its claim —
is the sweep that found this, and it had not been run to exhaustion before. It should be run again after
the fix, because a citation fix is exactly the kind of edit that introduces a new wrong citation.
