# Spec — The chat archetype's contract (`chat-archetype-contract`)

> **Change**: `cachicamas-chat-vocabulary-and-scope` · **CH-00** (Layer 3, Wave 0) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-00--record-the-archetypes-vocabulary-seam-answers-and-v1-scope), charter `0005:202-240`
> **Nodes**: CH-00.1 `[decision]` — answer every question the record must close (`0005:215-228`), the milestone's only node.
> **Status**: **new capability**. This file is the normative text; per the AG-14 / AG-19 / AG-20 / AG-21 / AG-22 precedent it is promoted verbatim to `openspec/specs/chat-archetype-contract/spec.md` at archive. No promoted spec's requirements are modified by this change.
> **Amended 2026-08-23 (CH-00 archive)**: promoted verbatim as a new capability at archive. Requirements `R-CHT-001` … `R-CHT-013`, non-functional `NFR-CHT-001` … `NFR-CHT-002` and scenarios `S-CHT-001` … `S-CHT-131` carried unchanged from `openspec/changes/archive/2026-08-23-cachicamas-chat-vocabulary-and-scope/specs/chat-archetype-contract/spec.md`. Relative links were re-based from the change directory's depth to this one; no normative text changed.
> **Governing ADR**: [ADR 0009](../../../docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md) — § D6 table ownership (`0009:152`, the quoted sentence at `:154-155`) and § D7(a) the read-with-substitution rule (`0009:174-176`). ADR 0005 § D2 fixes the target package positions cited by `0005:8`.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable **by reading the record** — this milestone is typed `[decision]` and ships no executable guard (proposal D-1), so no scenario here is discharged by a test and none is marked **(bite)**: there is no guard to deliberately violate.
> **IDs**: requirements `R-CHT-0NN`, scenarios `S-CHT-0NN`, non-functional `NFR-CHT-0NN`. **Append-only.**
> **Allocated ranges**: `R-CHT-001` … `R-CHT-099`; `S-CHT-001` … `S-CHT-199`; `NFR-CHT-001` … `NFR-CHT-099`. The range is reserved so the eleven later CH milestones append without collision. The header states **ranges and never totals** — a total is defended by no test and goes silently false on the next append (`agent-observability-boundary/spec.md:9`, `agent-concurrency-hardening/spec.md:8`).
> **Prefix verification**: `CHT` was verified collision-free across `openspec/specs/`, `openspec/changes/` **and** `openspec/changes/archive/` — 0 pre-existing hits for `R-CHT-`/`S-CHT-`/`NFR-CHT-`. `ARC` is **not** free and was rejected on that ground before the slug was chosen (proposal D-4).
> **Evidence gate**: the promoted spec plus `cd backend/agent && go test -race -count=1 ./...` with the wall-clock duration recorded (a `(cached)` run is not evidence), plus `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check`. `make all` MUST NOT be run — its fmt step rewrites committed files. There is no CI and no `openspec` CLI in this repository; every gate is human-run.
> **Sources**: charter `0005:202-240`; this change's `proposal.md` (decisions **D-1 … D-4**), `design.md` (decisions **D-A … D-F** and its closing work order `design.md:78-89`) and `explore.md`, all binding.

> **Note on length.** The `sdd-spec` skill sets a 650-word budget. This spec exceeds it deliberately, on the recorded precedent of `agent-observability-boundary/spec.md:14` and `agent-v1-scope/spec.md:348-350`: a seven-question closing checklist, an eleven-seam both-halves rule, three structurally separated vocabulary blocks and three absence rules do not compress without dropping content `openspec/config.yaml` requires be independently verifiable.

## Purpose

The chat archetype does not exist yet — there is no `backend/agent/src/chat/`, and no `backend/agent/src/cmd/` at all. Every statement about it is currently a sentence in a milestone doc, which leaves each of the eleven later CH milestones free to invent its own answer to the same question with nothing to disagree.

Layer 2 froze its surface at AG-23 and published, per seam, an injection point and a v1 default, precisely so its first consumer would have to **state** its answers rather than discover them. This capability makes CH-00's record checkable: it asserts the **shape** the record must have so that a reviewer can hold the record against it and find it wanting. It does not restate the answers — those live completely in `decision.md`, which a reader can close any of the seven questions from without this file and without source (`S-L3H-029` posture, design D-A).

**Two boundaries govern every requirement below.** (a) The generic-client boundary: `agent-layer3-handoff/spec.md:183` requires Layer 2's compatibility statement be written "without reference to files, shells, skills or terminals". CH-00's record is a *chat* archetype's record and may name chat concepts; it may name no coding-archetype concept. (b) `TurnOptions.PreRequestHook` is **frozen-and-superseded** (`openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:45`): it is kept, carries no deprecation marker, and MUST NOT be described as removed or deprecated anywhere in this change's artifacts.

## Coverage — the charter's own acceptance and closing checklist, traced

| Charter clause | Requirements | Scenarios |
|---|---|---|
| Acceptance (`0005:210`) — a reader asks any seam's answer, the record answers directly and names the injection point, without consulting source | `R-CHT-013` | `S-CHT-120`, `S-CHT-121`, `S-CHT-122` |
| Q1 (`0005:219`) — conversation / turn / message / participant, and for each the Layer 2 term it maps onto, the Layer 1 term it inherits, or the gap that forced a coinage | `R-CHT-001`, `R-CHT-002` | `S-CHT-001`…`S-CHT-004`, `S-CHT-010`…`S-CHT-012` |
| Q2 (`0005:220`) — every seam of the frozen AG-23 enumeration, its v1 answer and its injection point; deliberately empty answers listed, never omitted | `R-CHT-003`, `R-CHT-004` | `S-CHT-020`…`S-CHT-023`, `S-CHT-030`…`S-CHT-032` |
| Q2b (design D-2 / D-D disclosure) — gap findings against Layer 2's own seam set, separately labelled | `R-CHT-005` | `S-CHT-040`…`S-CHT-042` |
| Q3 (`0005:221`) — archetype name, package path, composition root | `R-CHT-007` | `S-CHT-060`…`S-CHT-062` |
| Q4 (`0005:222`) — database and whose tables; the answer names the owner, not the intent | `R-CHT-008` | `S-CHT-070`, `S-CHT-071` |
| Q5 (`0005:223`) — which frontend attaches, and what of its wire is already frozen | `R-CHT-009` | `S-CHT-080`…`S-CHT-082` |
| Q6 (`0005:224`) — what is out of v1, and the seam each deferral attaches to | `R-CHT-010` | `S-CHT-090`…`S-CHT-092` |
| Q7 (`0005:225`) — artifacts still saying "a Layer 3 application", and the substitution rule | `R-CHT-011` | `S-CHT-100`…`S-CHT-102` |
| Design D-A rules block — the two inherited form constraints | `R-CHT-006` | `S-CHT-050`…`S-CHT-053` |
| Design D-F — the inconsistency register, recorded not repaired | `R-CHT-012` | `S-CHT-110`…`S-CHT-112` |
| Proposal D-1 — evidence gate, and the record's self-sufficiency | `R-CHT-013`, `NFR-CHT-001`, `NFR-CHT-002` | `S-CHT-122`, `S-CHT-130`, `S-CHT-131` |

The columns name IDs rather than counts, for the reason the **Allocated ranges** header line records.

---

## Requirements

### R-CHT-001 — Every archetype noun sits in exactly one of three structurally distinct vocabulary blocks

The record MUST present this archetype's nouns in **three tables under three headings**, and the separation MUST be structural rather than typographic (design D-B — typography is not mechanically assertable and survives copy-paste badly).

The **mapped nouns** table MUST carry a `Maps onto` column, and each of its rows MUST name **exactly one** `VL2-*` identifier there, with its citation. The **coined nouns** table MUST NOT carry a `Maps onto` column at all, so that no coinage can hold a `VL2-*` identifier in a mapping position; where a `VL2-*` identifier appears in the coined table it MUST appear as evidence of **absence**, and the row MUST say which absence it evidences. The **inherited nouns** table MUST carry an `Inherited from` column naming **exactly one** Layer 1 `V-*` identifier with its citation, MUST NOT carry a `Maps onto` column either, and each of its rows MUST state both why the noun is not a Layer 2 mapping and why it is not a coinage of this archetype's own.

Three blocks and not two, because this archetype sits above **both** lower layers. Layer 1's own register draws the boundary this record must record on both sides of: `V-OUT-02` (`openspec/specs/ai-contract-vocabulary/spec.md:313`) assigns *transcript* to Layer 2 while keeping *message* in Layer 1 — "Layer 1 owns the unit, never the collection, its ordering across turns, or its repair." A two-block vocabulary can represent one side of that sentence and not the other. An inherited noun recorded as a coinage asserts an invention that did not happen; recorded as a mapping it asserts a Layer 2 term that does not exist.

Every noun the record uses for this archetype MUST appear in exactly one of the three tables. A noun in none of them, or in more than one, MUST fail review.

Every coined row MUST name **the gap that forces the coinage** — the row is incomplete without it, exactly as a seam row is incomplete without both halves under `R-CHT-003`.

#### Scenarios

- **S-CHT-001** — Given the record's vocabulary section, when each noun this archetype uses is looked up, then it is found in exactly one of the mapped table, the coined table and the inherited table, and never in more than one.
- **S-CHT-002** — Given the mapped nouns table, when each row's `Maps onto` cell is read, then it contains exactly one `VL2-*` identifier together with its citation, and no cell contains two identifiers, a prose hedge, or an empty value.
- **S-CHT-003** — Given the coined nouns table, when its column headers are read, then there is no `Maps onto` column; and when each row is read, then it names the gap that forces the coinage, and any `VL2-*` identifier it cites is presented as evidence of absence with the absence stated.
- **S-CHT-004** — Given the inherited nouns table, when its column headers are read, then there is no `Maps onto` column; and when each row is read, then its `Inherited from` cell contains exactly one Layer 1 `V-*` identifier with its citation, and the row states both why the noun is not a Layer 2 mapping and why it is not a coinage of this archetype's own.

### R-CHT-002 — "Conversation" and "participant" are recorded as coinages, and the three coinage-forcing Layer 2 absences are named

The record MUST place **"conversation"** and **"participant"** in the coined table and MUST NOT present either as a mapping onto any `VL2-*` row.

For "conversation" the record MUST state the cardinality argument that decides it: `VL2-COR-04` **run** (`openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:135`) is one harness invocation, whereas one conversation is realized by **one or more runs** — continued across turns and process restarts through the frozen transcript seam — so the relation is **containment, not identity**, and a mapping must preserve identity. `VL2-COR-05` **turn** (`:136`) is a mapping and MUST appear in the mapped table.

For "participant" the record MUST state that `VL2-COR-14` "the delegated participant" is a **different sense** (the subagent) and is therefore not the mapping target for a human participant.

The record MUST state plainly that Layer 2 defines **no term for a session**, **no term for a human participant**, and **no Layer 2 "part"**; and that `VL2-OUT-07` session persistence is an **exclusion assigned to Layer 3**, quoted as an exclusion and never used as a term.

#### Scenarios

- **S-CHT-010** — Given the record's vocabulary section, when "conversation" is looked up, then it is a coined row, its gap cell states the one-conversation-to-many-runs cardinality against `VL2-COR-04` (`decision.md:135`), and no cell of the record claims "conversation" maps onto `VL2-COR-04`.
- **S-CHT-011** — Given the record, when "participant" is looked up, then it is a coined row whose gap cell states that `VL2-COR-14` names the delegated-subagent sense and not a human participant.
- **S-CHT-012** — Given the record, when a reader asks whether Layer 2 supplies a term for a session, for a human participant, or for a "part", then the record answers no in each case in its own sentence, and cites `VL2-OUT-07` as an exclusion assigned to Layer 3 rather than as a term this archetype inherits.

### R-CHT-003 — One seam row per seam of AG-23's frozen enumeration, each carrying both halves and an explicit frozen/experimental status

The record MUST carry **one table row per seam named by AG-23's frozen enumeration**, in that source's own order (`openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:31-53`). That archived enumeration is immutable and fixes the population at **eleven seams — eight frozen and three marked experimental** under the heading at `:49`. Any count stated by the record MUST be attributed to that frozen archived source and MUST NOT be stated as a property of an open-ended set (`design.md:83`).

Every row MUST carry a non-empty **injection point** and a non-empty **v1 answer**. No seam may be listed with only one of the two halves — this is `S-L3H-028` (`openspec/specs/agent-layer3-handoff/spec.md:195`) applied to the first consumer.

Every row MUST carry a **status** cell holding the literal token `frozen` or `experimental — not frozen`. The status MUST be a column rather than a footnote, so the two classes cannot be mistaken for one another and neither can be counted by inference.

No seam named by that frozen enumeration may be omitted, including a seam whose v1 answer is deliberately empty (`0005:220`).

#### Scenarios

- **S-CHT-020** — Given the record's seam table, when its rows are read against `archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:31-53` in that source's order, then every seam the archived enumeration names has exactly one row, no row names a seam absent from it, and the row order matches.
- **S-CHT-021** — Given the seam table, when every row's `Injection point` and `v1 answer` cells are read, then both are non-empty for every row, and no row is present with one half filled and the other blank.
- **S-CHT-022** — Given the seam table, when the `Status` column is read, then every row holds the literal token `frozen` or `experimental — not frozen`, the three rows marked experimental are the three the archived heading at `:49` marks, and the remaining rows are marked frozen.
- **S-CHT-023** — Given a reviewer who deletes one seam row from the record and re-reads it against the archived enumeration, when the walk reaches that seam's position, then the omission is detected at that position — the record's ordering makes a dropped seam visible rather than silent.

### R-CHT-004 — Every seam answer is stated directly, and a deliberately empty answer is written as an answer

Each seam row's `v1 answer` MUST be a direct statement of this archetype's answer. It MUST NOT defer to source: a bare file path, a bare Go identifier, "see source", "as Layer 2 does" or an unaccompanied citation is not an answer (`S-L3H-029`, `agent-layer3-handoff/spec.md:196`). A citation MAY accompany an answer; it may never replace one.

A seam whose v1 answer is deliberately empty MUST be written as a **full sentence in the `0005:116` form** stating (i) what empty means operationally for this archetype, (ii) where it is injected, and (iii) which milestone ends the emptiness — or, where none does, that there is no owner and why. The tokens "none", "n/a", "-" and an empty cell MUST NOT appear as a v1 answer. The charter's own example is normative: *"this archetype has no tools"* is not a seam answer; the empty-tool-source answer naming its injection point and its later milestone is (`0005:213`).

#### Scenarios

- **S-CHT-030** — Given the record and any seam AG-23 names, when a reader asks what this archetype's v1 answer is, then the record answers in its own words and names the injection point, and the reader consults no Go source to obtain either half.
- **S-CHT-031** — Given the seam table, when every `v1 answer` cell is read, then none is the token "none", "n/a", "-", an empty cell, a bare file path, or a bare Go identifier standing alone as the answer.
- **S-CHT-032** — Given a seam whose v1 answer is deliberately empty, when its row is read, then the row states what empty means operationally, where the empty value is injected, and the milestone at which the emptiness ends — or states that no milestone owns it and why.

### R-CHT-005 — The gap-findings section is structurally incapable of reading as a seam answer

The record MUST carry a **separately headed** gap-findings section, placed immediately after the seam table, whose title marks it as findings rather than seam answers.

Its opening sentence MUST state that the frozen enumeration is exactly the seam rows above and that this section **extends nothing**.

Its table MUST NOT carry an `Injection point` column and MUST NOT carry a `v1 answer` column, so that a finding row cannot be read as a seam answer — the same structural device `R-CHT-001` uses for coinages. A finding MUST NOT be appended to the seam table under any marking, because appending to it *is* extending the frozen enumeration.

The section MUST record the gaps found against Layer 2's own seam table (`docs/architecture/0001-cachicamas-agent-stack-v2.md:658-683`): v2 § 6 seam 3 sandbox policy (`:670`), v2 § 6 seam 7 retry classification (`:674`), `Harness.RetryTiming` — cited as its **doc comment** at `backend/agent/src/agent/harness.go:96-97`, which calls it "the injected clock and wait-function seam", with the field declared at `:100` — and the settable but unenumerated `RetryAttempts` (`:94`), `ContextBudget` (`:120`) and `System` (`:56`).

Where the record notes `backend/agent/src/agent/tool.go:265-266`, it MUST describe the comment as a sentence that **wraps and completes** — "`ToolSource` port (G6) is AG-20's widening." — and MUST NOT describe it as truncated. That claim was checked and is false, and a false claim MUST NOT enter a durable record.

#### Scenarios

- **S-CHT-040** — Given the record, when the section following the seam table is read, then its heading marks it as findings rather than seam answers, and its opening sentence states that the frozen enumeration is exactly the seam rows above and that the section extends nothing.
- **S-CHT-041** — Given the gap-findings table, when its column headers are read, then there is no `Injection point` column and no `v1 answer` column; and when the seam table is re-read, then no gap finding appears among its rows under any marking.
- **S-CHT-042** — Given the gap-findings section, when the `tool.go:265-266` note is read, then it presents the comment as wrapping and completing with "AG-20's widening", and the word "truncated" is used nowhere about it.

### R-CHT-006 — The record carries the two inherited form constraints as rules, and neither is violated anywhere in it

The record MUST open with a rules block restating both inherited constraints in its own text, so a later reader of the record alone can apply them.

**(a) The generic-client boundary.** The record MUST name no coding-archetype concept — files, shells, skills, terminals or repositories presented as capabilities of this archetype (`agent-layer3-handoff/spec.md:183`). Chat concepts are permitted; this archetype is a chat archetype. A coding-archetype concept MAY be named only where the record is quoting or citing the boundary rule itself.

**(b) The frozen-and-superseded field.** The record MUST NOT describe `TurnOptions.PreRequestHook` as removed or as deprecated, anywhere, in any section, including footnotes and register rows. It is kept and carries no deprecation marker (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:45`). Where the record names it, it MUST name it as frozen-and-superseded.

#### Scenarios

- **S-CHT-050** — Given the record, when its opening rules block is read, then it restates the generic-client boundary and the frozen-and-superseded prohibition in the record's own text, each with its citation.
- **S-CHT-051** — Given the record, when it is read end to end, then no file, shell, skill, terminal or repository is presented as a capability of this archetype, and any such word appears only inside a quotation or citation of the boundary rule.
- **S-CHT-052** — Given the record, when every occurrence of `PreRequestHook` is read, then none describes it as removed or deprecated, and each presents it as frozen-and-superseded.
- **S-CHT-053** — Given a draft of the record in which one seam row describes `PreRequestHook` as deprecated, when the record is reviewed against `S-CHT-052`, then the review fails and names that row — the prohibition is checked, not assumed.

### R-CHT-007 — The archetype's name, package path and composition root are stated, together with the fact that neither path exists yet

The record MUST state the archetype's name, its package path `backend/agent/src/chat/` and its composition root `backend/agent/src/cmd/chat/`, citing `0005:8` and ADR 0005 § D2.

It MUST state plainly that **neither path exists yet**, and that there is no `backend/agent/src/cmd/` directory at all. Stating the paths as though they existed would make the record false on the day it lands, and CH-01.1 owns their creation (`0005:228`).

#### Scenarios

- **S-CHT-060** — Given the record, when a reader asks where this archetype and its composition root live, then the record names `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` and cites `0005:8`.
- **S-CHT-061** — Given the record, when the identity section is read, then it states that neither path exists yet and that there is no `backend/agent/src/cmd/` at all, and it names CH-01.1 as the owner of their creation.
- **S-CHT-062** — Given a reader who checks the repository against the identity section, when the two paths are not found, then the record has already said so and no statement in it is falsified by their absence.

### R-CHT-008 — The persistence answer names the owner of the tables, not the intent

The record MUST answer whether this archetype writes to a database, and MUST **name the owner** of the tables it writes (`0005:222` — "the answer must name the owner, not just the intent"), citing ADR 0009 § D6 (`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md:152`, whose quoted sentence spans `:154-155`): each business system owns its own tables, and no archetype writes to another system's schema.

An answer expressing only intent — that the archetype "will persist", "needs storage", or "stores conversations" without naming whose tables — MUST fail review.

#### Scenarios

- **S-CHT-070** — Given the record, when a reader asks whether this archetype writes to a database and to whose tables, then the record answers both halves and names the owning system, citing ADR 0009 § D6 (`0009:152`, its quoted sentence at `:154-155`).
- **S-CHT-071** — Given a draft whose persistence section states only that the archetype persists conversations server-side, when it is reviewed against `S-CHT-070`, then the review fails because no owner is named.

### R-CHT-009 — The attaching frontend and the frozen part of its wire are enumerated as inherited, not designed here

The record MUST name the attaching frontend and MUST enumerate what of its wire is already frozen by the promoted `frontend-chat-layer1` capability, `REQ-1` through `REQ-7`, marked explicitly as **inherited and not this document's to design** (`0005:223`).

Where the record cites the auth guard chain, it MUST cite the **corrected** location: `frontend/src/routes/home/layout.tsx` — the `onRequest` handler at `:30`, its three calls at `:31-33`, its doc comment at `:21-29` and its imports at `:15-17`. It MUST NOT reproduce the promoted spec's incorrect `home/index.tsx` citation as though it were correct; that defect is carried in the inconsistency register under `R-CHT-012`.

The record MUST NOT propose retiring, altering or silently dropping any frozen wire element, including the literal string `REQ-5` mandates; retiring it needs a recorded spec delta at CH-05.2.

#### Scenarios

- **S-CHT-080** — Given the record, when a reader asks which frontend attaches and what of its wire is frozen, then the record names the frontend and enumerates `frontend-chat-layer1` `REQ-1`…`REQ-7`, each marked as inherited rather than decided here.
- **S-CHT-081** — Given the record's frontend section, when the auth guard chain citation is read, then it points at `frontend/src/routes/home/layout.tsx` with `onRequest` at `:30` and its three calls at `:31-33`, and the record nowhere presents `home/index.tsx:39-51` as the correct location.
- **S-CHT-082** — Given the record, when its frontend section is read for changes to the frozen wire, then it proposes none, and it states that the `REQ-5` literal is retired only by a recorded spec delta at CH-05.2 and never silently.

### R-CHT-010 — Every deferral carries its attaching seam, and no defect is laundered into the deferral register

The record MUST reproduce doc 0005's "Explicitly deferred until after v1" table (`0005:997-1011`) row for row, citing `0005:997` as its source, and **every row MUST carry its attaching seam**. A bare exclusion — a row saying only that something is not in v1 — MUST fail review (`0005:224`).

AG-23's own known-limitations register (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:103-108`) MUST be **cited as inherited Layer 2 input and MUST NOT be reproduced** here; reproducing it would create a second count-bearing copy that drifts against its source.

No `F-*` defect may appear as a deferral row. The record adopts AG-23 § 4.3's rule (heading `:110`) for its own register, quoting the archived text verbatim: a defect "appears **nowhere** in the register above. Laundering it as a limitation would make this statement false on the day it lands." `F-1`, `F-2` and `F-3` therefore live only in the inconsistency register of `R-CHT-012`.

#### Scenarios

- **S-CHT-090** — Given the record's deferral register, when its rows are read against `0005:997-1011`, then every row of that source table is present and cites it as the source.
- **S-CHT-091** — Given the deferral register, when each row is read, then it names the seam the deferral attaches to, and no row is a bare statement that something is out of v1.
- **S-CHT-092** — Given the deferral register, when it is searched for `F-1`, `F-2` and `F-3`, then none is present as a row; and when AG-23's known-limitations register is looked for, then it is cited as inherited input and its rows are not reproduced.

### R-CHT-011 — The substitution rule is quoted, and the artifacts it applies to are listed

The record MUST list the cited Layer 2 artifacts that still say "a Layer 3 application" — the occurrences in `openspec/specs/agent-contract-vocabulary/spec.md`, `openspec/specs/agent-layer3-handoff/spec.md` and `openspec/specs/agent-v1-scope/spec.md` — and MUST state the matching rule the list was derived under, namely a search for the phrase "Layer 3 application" rather than for the exact string "a Layer 3 application", because at least one occurrence carries an intervening modifier. The record MUST quote ADR 0009 § D7(a)'s substitution rule (`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md:174-176`), under which "a Layer 3 application" in shipped Layer 2 artifacts is read as "a Layer 3 archetype" until the renaming change lands.

Wherever the record cites one of those artifacts, it MUST say that it is reading it under the substitution rule, so no reader concludes the rename already happened. The record MUST NOT perform the rename in any promoted spec — that is `0005:212`'s scope fence and ADR 0009 § D7(a)'s own change owns it.

#### Scenarios

- **S-CHT-100** — Given the record, when the substitution section is read, then it lists the occurrences by file and line and quotes ADR 0009 § D7(a) with its citation (`0009:174-176`).
- **S-CHT-101** — Given the record, when any citation of a listed artifact is read, then the record states that the artifact is being read under the substitution rule.
- **S-CHT-102** — Given the repository after this change merges, when the listed promoted specs are diffed, then none is modified by this change and every "a Layer 3 application" occurrence is unchanged.

### R-CHT-012 — The inconsistency register carries each found defect with both sides cited and a disposition, recorded and not repaired

The record MUST carry an inconsistency register as its last content section before the closing verification, holding a row for each defect this change found — `F-1`, `F-2` and `F-3` as enumerated in this change's `proposal.md` follow-up table — with columns naming **Side A (cited)**, **Side B (cited)** and a **Disposition**. A row citing only one side, or carrying no disposition, MUST fail review.

The register's header line MUST state verbatim that the defects are **recorded, not repaired**, and that no promoted spec is modified by this change.

`F-1` and `F-2` both land on `openspec/specs/agent-contract-vocabulary/spec.md`; the register MUST say so, so one future change can close both. `F-3`'s row MUST carry the corrected guard-chain location per `R-CHT-009`.

#### Scenarios

- **S-CHT-110** — Given the record's inconsistency register, when each of `F-1`, `F-2` and `F-3` is looked up, then each has a row citing both sides of the inconsistency and stating a disposition.
- **S-CHT-111** — Given the register, when its header line is read, then it states that the defects are recorded and not repaired and that no promoted spec is modified by this change; and when this change's "Modified capabilities" is read, then it says None.
- **S-CHT-112** — Given the register, when `F-1` and `F-2` are read, then both name `agent-contract-vocabulary/spec.md` as their shared landing spot and the register states that one future change can close both; and when `F-3` is read, then it carries the `home/layout.tsx` location rather than the promoted spec's incorrect one.

### R-CHT-013 — The record answers any seam question directly, and that is this milestone's acceptance

The record MUST satisfy the charter's acceptance verbatim (`0005:210`): given the record, when a reader asks what this archetype's answer is to any seam Layer 2 names, then the record answers it directly and names that seam's injection point, without the reader consulting source.

This is a property of the **record**, discharged by reading it. This milestone is typed `[decision]` (`0005:215`) and MUST NOT be discharged by production code, by a new Go test, or by a guard: the package such a guard would protect does not exist (`R-CHT-007`, proposal D-1). The earliest honest mechanical binding is CH-01.2's import guard, not here.

#### Scenarios

- **S-CHT-120** — Given the record and any seam of AG-23's frozen enumeration chosen at random by a reviewer, when the reviewer asks that seam's v1 answer and injection point, then the record supplies both directly and the reviewer opens no Go source file to obtain either.
- **S-CHT-121** — Given the record and the seven questions at `0005:219-225`, when each question is asked in turn, then the record carries a written answer to each, and the closing-verification section of the record names where each answer lives.
- **S-CHT-122** — Given this change as merged, when its diff is inspected, then it adds no Go file, no test file, no guard, and no change under `backend/` or `frontend/`; and `cd backend/agent && go test -race -count=1 ./...` run uncached is green with its wall-clock duration recorded.

---

## Non-functional requirements

### NFR-CHT-001 — The record is self-sufficient; the spec asserts shape, never answers

A reader MUST be able to close any of the seven questions from `decision.md` alone, without this spec and without source. This spec MUST carry only shape assertions about the record and the contract claims later CH milestones cite; **no answer may live only in the spec** (design D-A).

- **S-CHT-130** — Given this spec and the record, when any of the seven questions is answered, then the answer is found in the record, and removing this spec from the reader's hands changes no answer.

### NFR-CHT-002 — The evidence gate is the promoted spec plus a green uncached race run

Closing evidence for this change is the promoted spec plus `cd backend/agent && go test -race -count=1 ./...` run **uncached** with its wall-clock duration recorded, together with `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check`. `make all` MUST NOT be run. A `(cached)` or sub-second result is a cache hit and is not evidence.

- **S-CHT-131** — Given the verification record for this change, when the test evidence is read, then it shows an uncached invocation with `-count=1` and a recorded wall-clock duration consistent with a real race run, and no line of evidence is a `(cached)` result.

---

## Explicit non-requirements

| Not required here | Why, and who owns it |
|---|---|
| Any Go file, test, guard or production code | Proposal D-1 — CH-00.1 is `[decision]`; the package a guard would protect does not exist. CH-01.2 owns the first mechanical binding |
| Implementing any seam answer | Charter `0005:212` — owned by every later CH milestone |
| Creating `backend/agent/src/chat/` or `backend/agent/src/cmd/chat/` | `0005:228` — owned by CH-01.1 |
| Renaming "application" to "archetype" in Layer 2's promoted specs | `0005:212` and ADR 0009 § D7(a)'s own SDD change |
| Repairing `F-1`, `F-2` or `F-3` | Proposal D-3 — record, do not repair. `R-CHT-012` requires the register, not the fix |
| Re-litigating the deferred capabilities | `0005:997-1011` states them with their attaching seams; it does not re-decide them |
| Retiring the `frontend-chat-layer1` `REQ-5` literal | Mandated by a promoted spec; needs a recorded spec delta at CH-05.2 (`R-CHT-009`) |
| An `openspec validate` run or any CI gate | Neither exists in this repository; every gate is human-run |
