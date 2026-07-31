# Verify report — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-03--decide-the-v1-capability-set-and-optional-capability-discovery)
> **Node**: AI-03.1 — The capability matrix `[decision]`
> **Phase**: verify
> **Status**: **PASS**
> **Date**: 2026-07-31
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Base**: `origin/main` @ `b6c59e6`
> **Change commit**: `f701e58` — the last node of wave 0
> **Closes**: the Layer 1 half of the concern doc 0001 and ADR 0005 track as **G3**
> **Mode**: `[decision]` leaf — no production code, no `make test` gate. Every check below is inspection; citations were re-resolved against their sources rather than taken on the artifact's word.

---

## 1. Charter acceptance

| # | Charter clause | Verdict | Evidence |
| --- | --- | --- | --- |
| 1 | A recorded capability matrix with a **required/optional column** and a **discovery mechanism** | **PASS** — three closed lists (5 required, 3 optional, 4 excluded) with stable identifiers, plus § 9's mechanism | § 3 |
| 2 | **AI-23's suite can mark each case required or optional from this list alone** | **PASS**, and passed by a route the checklist does not name | § 4 |
| 3 | A provider lacking an optional capability is **fully conformant** and records "absent" rather than skipping silently | **PASS** — `absent` is one of four closed outcome values, and the verdict rule makes it a pass while making `not exercised` inconclusive | § 3.5 |
| 4 | doc 0002's note: token counting is optional and discovered by assertion on the provider value, **not part of the provider interface** | **PASS** — § 9 keeps the core contract unwidened and names AI-20.5 item 3's pin as the mechanical form | § 3.4 |

Clause 2 is the one that determines whether this artifact works. § 4 shows that it is **not satisfiable from the required list**, that the artifact found this itself, and what it supplied instead.

---

## 2. Deliverable inventory

| Path | Present | Note |
| --- | :---: | --- |
| `decision.md` | ✅ | 581 lines; §§ 5–10 follow the closing checklist's order, with §§ 3, 4, 8 and 11 sitting outside the spine and each explaining why |
| `proposal.md` | ✅ | |
| `specs/ai-minimum-capabilities/spec.md` | ✅ | `R-AIC-001` … `R-AIC-015`, `S-AIC-001` … `S-AIC-059` |
| `design.md` | ✅ | names the three failure modes it targets |
| `explore.md` | ✅ | |
| `tasks.md` | ✅ | six tasks + a twelve-check verification pass, all `[x]` |
| amendment to `../cachicamas-ai-contract-vocabulary/decision.md` | ✅ | append-only; § 5 |
| anything under `backend/` | ✅ absent | § 9 |

`git show --name-only f701e58` lists **seven files**: the six above plus the register amendment. `tasks.md` acceptance criterion 5 holds.

---

## 3. Closing-checklist closure

The verification is that each item is **closed** — that AI-20.5, AI-23 and AI-24 could be written from this artifact without reopening it — not that the topic is covered.

| # | Item | Verdict |
| --- | --- | --- |
| 1 | Required capabilities enumerated | **CLOSED** — § 3.1 |
| 2 | Optional capabilities enumerated, **each with its reason** | **CLOSED** — § 3.2 |
| 3 | Excluded for v1, **with the reason** | **CLOSED** — § 3.3 |
| 4 | The discovery mechanism decided, with **advertise AND ask** both stated | **CLOSED** — § 3.4 |
| 5 | "Absent" is a **recorded outcome**, and the record's shape is sketched | **CLOSED** — § 3.5 |

### 3.1 Item 1 — the required list, and the defect a required list actually has

All five named capabilities are present with stable identifiers: `CAP-R-01` streaming text, `CAP-R-02` tool calls, `CAP-R-03` completion metadata, `CAP-R-04` cancellation, `CAP-R-05` typed failures with the partial-output distinction.

The item is closed rather than merely populated because each entry carries **what it does not oblige** — and the artifact states why that negative clause is structural rather than editorial: *the expensive defect in a required list is not an omission, it is an entry that forces an honest adapter either to fail conformance for something that is not a defect, or to fabricate.* Two such readings hide inside `CAP-R-03` alone, and both are closed explicitly:

1. **Not every finish-reason value must be emitted.** The obligation is that the vocabulary is *reachable*; `V-MET-08` **unknown finish reason** is a conformant mapping. Requiring emission of every value would require inventing stop conditions. (`S-AIC-010`)
2. **Not every token count must be populated.** `V-MET-10` makes each count independently present or absent, and `V-MET-11` makes "not reported" and "reported as nought" different facts. *"Requiring a populated count is requiring a fabricated one"* — the same defect as a mandatory token count, one level down, hidden inside a capability everyone agrees is required. (`S-AIC-009`)

Two more negative clauses close the cases most likely to be got wrong downstream (`S-AIC-011`): a block delivered **whole with zero deltas** is conformant, and an adapter that **mints identifiers** for a vendor that assigns none is *satisfying* `CAP-R-02` rather than lacking it.

`CAP-R-04`'s standing is handled honestly rather than uniformly. It is required **by construction** — AI-02 put the cancellation signal on the call that creates the stream and AI-20.4's guard sees it, so "absent" is not expressible — and the artifact says so, keeping it on the list only because the list must be total for `CAP-R-05`'s sake. `S-AIC-012` holds: `CAP-R-04` and `CAP-R-05` **cite** AI-02 §§ 5 and 7 for their observable shapes rather than re-deciding them.

**The closing note is what makes the list defensible.** § 5 records two behaviors that are required of every adapter and are deliberately **not** capabilities — the stream lifecycle (`V-PRV-05`, proven by AI-20.3) and request-translation totality (AI-26.8) — because listing them would make the required list a duplicate of the contract, "and a duplicated contract drifts." That note is what forces § 11 to exist.

### 3.2 Item 2 — three optional entries, each with its reason, and the argument that carries the milestone

All three named capabilities are present: `CAP-O-01` reasoning content, `CAP-O-02` token counting, `CAP-O-03` honoring cache-boundary markers. Each carries the reason it is optional **rather than required** (`S-AIC-016`), and each carries **what a consumer does on a recorded absence** (`S-AIC-017`) — which is the clause that makes optionality survivable rather than merely permitted.

**`CAP-O-02` token counting is the load-bearing entry, and it is argued rather than asserted.** The check applied: does the artifact concede the opposing reading, and does it lose on the right ground?

- **The opposing case is stated at its strongest and conceded as correct as far as it goes.** Admission test 1(a) — loop-necessity — is satisfied *"and not marginally"*: Layer 2's compaction needs a real number, and doc 0001 § 6 seam 6 says so directly. That quotation was re-resolved against the source: *"Compaction that estimates by character count is wrong by enough to matter"* is **present verbatim** in doc 0001's seam table. An artifact that did not concede this would be weaker than it looks, and `tasks.md`'s own review focus says so.
- **It loses on test 1(b), universality without a lie** — not on effort, not on convenience. `S-AIC-013` requires the reason to be a consequence for a consumer, and it is: a required count leaves an adapter whose vendor has none exactly two options, and the second one's cost is stated precisely rather than as an epithet — *"a fabricated count is worse than an absent one, because the absent one degrades visibly and the fabricated one corrupts invisibly."* Given an honest absence, compaction estimates and **knows it is estimating**; given a fabricated count, compaction believes the figure.
- **The corollary is turned on Layer 1 itself** (`S-AIC-015`, § 13 rule 4): Layer 1 must never supply a fallback estimate, because *"a default that estimates is a fabrication with better provenance"* — the consumer cannot tell it from a real count, which is precisely the harm.

**`CAP-O-03` is the arguable entry, and the artifact argues against itself first** (`S-AIC-022`). The case for calling it an adapter-local mapping obligation is stated at full strength: markers are advisory by contract (`V-REQ-23`), and AI-11.3 proves ignoring every one of them leaves the request fully translatable and semantically unchanged. If both behaviors are conformant and neither changes what the consumer receives, it looks exactly like a non-capability.

The answer is `V-REQ-24`, **the breakpoint cap** (`S-AIC-023`): a provider that honors markers enforces a small hard cap whose breach is a caller-contract failure raised before any I/O, and a provider that caches automatically has none. The two therefore differ in **what a consumer may legally construct** — and Layer 2 places markers programmatically under **G4**. That is a consumer-visible consequence, which is the exact showing § 8's divergence rule demands.

`S-AIC-024` also holds, and its mirror for `CAP-O-01`: what is discovered is **honoring**, never the markers; what is optional is **emission**, never the neutral reasoning shape. Both rows state that a reader who concludes AI-11 or AI-07 is conditional has inverted them.

**"And anything else v1 admits" resolves to nothing, with the work shown** (§ 6.4). Five candidates were run through the tests and each is recorded with the clause it fails — reasoning-signature round-tripping (not separate, inside `CAP-O-01`), cache token counts (a usage field inside `CAP-R-03`), honoring tool choice (a translation failure at AI-26.8.2, not a standing), the escape hatch (nothing to ask), server-supplied request identity (safe metadata on a failure). `S-AIC-005` holds several times over.

Closure is **structural, not tidy**: every optional capability costs AI-23 a suite case, AI-38 a record entry and AI-24.1 a recorded expectation — and § 10's totality property is only available over a closed list. The amendment route is stated (`S-AIC-018`): by amendment to this artifact, in the pull request that needs it, having applied test 2 and § 8's divergence rule.

`CAP-O-01` also carries a **non-subdivision clause** worth recording: an advertised capability is advertised whole, so `V-REQ-11`'s byte-exact round trip comes with it. Splitting it would license an adapter that emits reasoning and drops signatures, and doc 0001 § 3.2 records what that breaks — multi-turn extended thinking with tool use, because at least one provider signs reasoning cryptographically.

### 3.3 Item 3 — four exclusions, each with a reason, under a stated rule

All four named exclusions are present with identifiers: `CAP-X-01` multimodal beyond text, `CAP-X-02` embeddings, `CAP-X-03` batch APIs, `CAP-X-04` server-side tool execution.

The item is closed by a property beyond enumeration: each entry carries **"why not optional"** as well as "why excluded", under one separating rule (`S-AIC-048`):

> **An optional capability has a defined absence. An excluded one has no defined presence.**

That rule is what makes the list extensible by someone who was not in the room. Applied:

- **`CAP-X-01`** — "supports images" is not one question but a family (which modalities, direction, formats, size limits, transcoding), and that family *is* the per-provider capability-detection model doc 0001 § 8 says v1 does not have. The doc 0001 quotation was re-resolved and is present: image and audio "exist in the content vocabulary and no constructor produces them", consistent with `V-REQ-05`. (`S-AIC-046`)
- **`CAP-X-02`** — an embeddings call returns a vector, not a normalized event stream (`V-PRV-01`). The artifact makes the sharp point that the *mechanism would permit it* through an optional contract, "and that is precisely why the exclusion has to be explicit rather than left to the mechanism's flexibility."
- **`CAP-X-03`** — not a behavior added to a stream but the **absence of a stream**: nothing to cancel, nothing to close, no terminal event. Derived from AI-02's lifecycle rather than asserted.
- **`CAP-X-04`** — principled rather than scope-based (`S-AIC-047`): it is **trap 1** directly, it places execution below Layer 1 where it is invisible to the permission protocol (**G1**) and the sandbox (**G2**), and doc 0001 § 6 seam 2's consequence was re-resolved and is verbatim — *"the event stream stops being a complete description of the session. Every frontend then reimplements it, differently."* The closing clause is the strongest sentence in § 7: *"There is no version of 'absent' that makes its presence safe."*

§ 7's coda states that an exclusion is recorded so its absence "reads as a decision rather than an oversight", and that none of the four is admitted by this decision's amendment route — which is reserved for optional capabilities the vocabulary already models.

### 3.4 Item 4 — the mechanism, both halves, and the honesty about what was inherited

**The inherited part is declared as inherited.** doc 0002's checklist item 4 states the mechanism *family* as a requirement, and doc 0001 § 6 seam 6, § 7 **G3** and ADR 0005 § D4 row G3 all say the same thing. § 9 says so explicitly: *"This decision does not argue that choice; it would be dishonest to present an inherited constraint as an argued conclusion."* That is the correct handling and it is the opposite of AI-02's carrier, where the choice genuinely was free — the wave gets both cases right, in opposite directions.

**Both halves are stated** (`S-AIC-027`):

- **Advertise** — by satisfying the additional contract **and by no other means**. Not a flag, not a registration call, not a declared list, not a configuration entry, not a constructor argument. An adapter that lacks the capability **declares nothing at all** — no field, no negative answer, no unsupported entry. The artifact names this asymmetry as the mechanism's central virtue: *"absence costs an adapter zero lines"*, which is what removes every incentive to fabricate. (`S-AIC-028`)
- **Ask** — of the **provider value**, at the point of use, never of a model identity, a configuration entry or a catalog (`V-OUT-14`, and ADR 0005 § D1 row 4's rule that Layer 1 reads no configuration). The result is the capability or a **clean absence** — AI-20.5 item 1's own words, "not an error and not a zero", re-resolved against doc 0002 and present verbatim. (`S-AIC-029`, `S-AIC-032`)

Three advertising rules and three asking rules bind each side. **Advertising binds** (`S-AIC-031`) is the one that closes the mechanism's back door: a provider that satisfies an optional contract and then declines to answer is **non-conformant**, not absent — without it an adapter could satisfy every optional contract and refuse from inside each one, reproducing the widened interface one level down and harder to see.

**Six alternatives are named and rejected** — `S-AIC-030` requires four; the artifact supplies six, and the two extra ones are the informative ones. A **single aggregate optional contract** is rejected as all-or-none *and* as ungrowable: adding a fourth capability would change a contract existing adapters already satisfy, breaking the very thing the mechanism promised would never widen. **Per-consumer contracts at each call site** are rejected because discovery would still work but nothing could **enumerate** — AI-23 could not iterate and AI-38's record could not be total.

**The wrapping rule is the artifact's own contribution** (`R-AIC-011`). It appears in no source document, and it is a direct consequence of choosing silent absence: a wrapper that satisfies the core contract and forwards nothing **silently removes every optional capability of the value it wraps**, and the removal is invisible *precisely because* absence is legitimately silent. The rule — forward, or document which you remove and why — is stated as a pull quote, and **AI-37 is named as the first milestone it binds**, before AI-37 writes its first wrapper. (`S-AIC-034`, `S-AIC-035`)

### 3.5 Item 5 — the record, and the distinction made structural

`V-PRV-09` states the requirement: *"'Absent' is a recorded outcome, not an unrun test."* The artifact opens § 10 with the observation that doc 0002 restates this clause **four times** — AI-23.1 item 2, AI-23.8 item 1, AI-23.6 item 2, AI-38.2 — *"which is a reliable signal that prose is not enough"*, because in an ordinary test report the two are typographically identical: a case with no result.

**So the distinction is made structural rather than editorial**, and this is what closes the item:

| Outcome | Meaning | Legal for |
| --- | --- | --- |
| **satisfied** | exercised and met | required, optional |
| **absent** | asked, does not advertise, cases **deliberately** not exercised — a conclusion, and a conformant one | **optional only** |
| **failed** | exercised and not met | required, optional |
| **not exercised** | the cases did not run — the **absence** of a conclusion | neither, as a result |

Three consequences are stated, and each closes a way a reader goes wrong: `absent` is **illegal for a required capability** (a required capability not satisfied is `failed`; there are no waivers, and AI-38.1 item 2's "that is what 'required' means" was re-resolved in doc 0002 and is present); an advertised optional capability that fails is `failed`, not `absent` (advertising binds, in the record); and `not exercised` is the record saying it does not know.

**The verdict rule is what makes the four-value set survive contact with a verdict** (`S-AIC-042`):

> A record containing any `not exercised` entry is **inconclusive** — neither a pass nor a failure.

Without that clause a four-value set collapses to three the moment a verdict is computed and a skipped case reads as a pass again — *"the failure `V-PRV-09` exists to prevent, defeated at the first use of the thing built to prevent it."* This is the sharpest check in the artifact and it holds.

**Totality** (`S-AIC-036`): one entry per capability in the closed lists, in every record; a capability with no entry is a **defect in the run**, not an absence. The artifact ties this back correctly — totality is only available because §§ 5 and 6 are closed, which is the second structural reason § 6.4 closes the optional list.

**Standing comes from the decision, never from the run** (`S-AIC-038`), so a run cannot demote a required capability by recording it optional and make the suite's verdict negotiable. **Records are comparable entry by entry**, because AI-24.1 records an expectation before an adapter exists and AI-38.2 asserts against it — with a difference in *either* direction being a finding, including an unexpected `satisfied`, which means the adapter grew a capability nobody reviewed.

**What the record must not carry** (`S-AIC-039`): no capability-specific detail, and no model content, credential or raw provider text — because AI-40.2 publishes it into package documentation, so `V-FAIL-13`'s safe-metadata posture and `V-FAIL-14`'s redaction discipline bind it. *"A record is published by construction; it is not a debugging artifact."*

§ 10.4 separates the absent optional capability from AI-19.2's **unsupported capability** failure category across five axes (`S-AIC-057`). They share a word and nothing else: one is about the provider and known before any request is built; the other is about one request's features and arises at translation time. Confusing them would make a consumer treat absence as a failure — which AI-20.5 item 1 forbids — and hide a real defect behind a conformant-looking record.

---

## 4. The acceptance criterion, and the gap the artifact found in it

> *"AI-23's suite can mark each case required or optional **from this list alone**."*

This is the criterion the milestone can most easily appear to satisfy while failing. The artifact does not appear to satisfy it — it **identifies why the obvious reading is unsatisfiable and supplies what is**.

§ 11 states the problem plainly: the required list is five capability-shaped entries, while the contract has many more required cases — every ordering invariant, every ownership rule, the pre-stream contract, redaction, translation totality. A suite marking cases by lookup against the required list would leave all of those **unmarked**, *"and the natural default for an unmarked case is the dangerous one."*

The rule supplied instead:

> A conformance case is **optional if and only if** the capability it exercises appears in this decision's optional list (`CAP-O-01` … `CAP-O-03`). **Every other case is required.**

Three properties make it work, and all three were checked:

1. **It is total.** No case is unmarkable, because the biconditional's negative branch is a default rather than a lookup. `S-AIC-025` — an ordering invariant or the pre-stream contract — is marked required "unambiguously and without consulting any other document". Confirmed against § 11's worked cases: AI-23.4 item 1 (exactly one terminal per stream) and AI-23.7 (a planted secret appears nowhere) are both marked required **by the default, not by an entry**, and § 12 records that as the rule working as intended.
2. **Its failure mode is the safe one.** A capability nobody has classified defaults to **required**, so an unclassified case fails loudly rather than being skipped silently — § 10's posture applied one level up, in the suite instead of in the record.
3. **It is stated over the capability a case exercises, not over the suite's node structure.** This is the rule's sharpest edge and the artifact tests itself on it: AI-23.8 item 3 (absent-versus-zero in usage) **lives inside AI-23.8's optional-capability node** but exercises `CAP-R-03`, and is therefore **required**. *"The node it lives in does not decide its standing; the capability it exercises does."* Seven worked cases in total, and this is the one that proves the biconditional was written carefully.

`S-AIC-026` holds on both halves: the rule is a biconditional over the optional list, and the artifact states **why** the required list cannot be the marking source.

**Verdict on clause 2 of the charter: PASS.** AI-23's suite can mark every case from this decision alone, and the artifact says which part of itself does the marking.

The second half of the acceptance criterion — a provider lacking an optional capability is fully conformant and records "absent" rather than skipping silently — is § 6's opening clause (*"A provider that lacks every one of these is fully conformant… an adapter implementing only the required surface passes, completely, with three recorded absences"*) plus § 10's outcome set. Both hold.

---

## 5. Register integrity — the AI-03 amendment

Three nouns appended to AI-01's register in this same pull request, under AI-01 § 9 rule 2.

| Appended | Ordinal | Owner | Why the register lacked it |
| --- | --- | --- | --- |
| `V-PRV-16` **capability** | next free in `V-PRV` | AI-03 | Closes a gap **AI-01 identified in its own § 7 preamble**: that preamble names five terms AI-03's charter is not writable without, and the table delivered four |
| `V-PRV-17` **token counting** | next free in `V-PRV` | AI-03 | The phrase **already appeared inside `V-OUT-06`'s definition undefined**, where it silently collapses into `V-MET-09` **usage** — a report about an output standing in for a question about an input |
| `V-PRV-18` **capability outcome** | next free in `V-PRV` | AI-03 | The word "outcome" **already appeared inside `V-PRV-09` undefined**, and the distinction that row exists to protect is a distinction *between outcome values* |

`V-PRV-17` and `V-PRV-18` are the same shape as AI-02's `V-STR-23` **backpressure**: a word used inside a definition without being one. Three of the five nouns appended across this wave are that shape, which is a useful signal about where a register leaks.

### 5.1 Measured, not claimed

Diff of `cachicamas-ai-contract-vocabulary/decision.md` from `6da8593` to `f701e58`:

```
+ dated amendment blockquote under § 7
+ (blank separator)
+ | `V-PRV-16` | capability         | … | AI-03 | …
+ | `V-PRV-17` | token counting     | … | AI-03 | …
+ | `V-PRV-18` | capability outcome | … | AI-03 | …
- **Term count:** … 15 provider-surface … = **111 terms**. *(Amended … V-STR-22, V-STR-23 …)*
+ **Term count:** … 18 provider-surface … = **114 terms**. *(Amended … V-STR-22, V-STR-23 … Amended … V-PRV-16, V-PRV-17, V-PRV-18 …)*
```

**Five additions and one replacement**, and the replacement is the register's own arithmetic. `S-AIC-054` holds as a measured property: **no existing row was renumbered, reworded, reordered or removed.**

### 5.2 Counts, measured against the register's own claims

Row counts extracted by matching `^\| \`V-[A-Z]+-\d+\`` across the whole file, after this commit:

| Category | § 10 claims | Actually counted | Match |
| --- | ---: | ---: | :---: |
| `V-REQ` | 29 | **29** | ✅ |
| `V-STR` | 23 | **23** | ✅ |
| `V-MET` | 12 | **12** | ✅ |
| `V-FAIL` | 15 | **15** | ✅ |
| `V-PRV` | 18 | **18** | ✅ |
| `V-OUT` | 17 | **17** | ✅ |
| **Total** | **114** | **114** | ✅ |

Register history: **109 → 111 → 114**, consistent with two amendments of two and three rows. All 114 identifiers are distinct, all 114 term names are distinct, and `V-PRV` is contiguous 01–18 with no gap and no reuse.

### 5.3 Each appended definition **defers** rather than decides

This is the clause `tasks.md` calls "easy to miss", and it is the one that keeps AI-01 § 9 rule 5 intact — a register row stating which outcome values exist would be AI-01 deciding AI-03's matrix retroactively. Checked row by row (`S-AIC-055`):

| Row | Deferral clause |
| --- | --- |
| `V-PRV-16` | *"Which behaviors are capabilities, and which standing each takes, is AI-03's."* |
| `V-PRV-17` | *"Its standing, and what a consumer observes when it is absent, are AI-03's."* |
| `V-PRV-18` | *"Which values the set contains, and which of them are conformant, are AI-03's."* |

All three defer explicitly, by name, in the pattern `V-PRV-08` already established for the discovery mechanism. **PASS.**

### 5.4 None of the three is defined locally

`S-AIC-053` requires the appended nouns to appear in this change's artifacts only as citations. § 1 states the discipline — *"They are used here; they are not defined here."* The closest call is § 3, which opens with a block quotation of `V-PRV-16`; see § 7.1.

---

## 6. Spec conformance

All fifteen requirements in `specs/ai-minimum-capabilities/spec.md` verified. Listed below are the clauses where the verdict needed judgement rather than a count.

| Requirement | Verdict | Note |
| --- | --- | --- |
| `R-AIC-002` — classification governed by stated tests, not assertion | **PASS**, and the tests are part of the deliverable rather than scaffolding | § 4's four tests are applied in order and named per entry. Test 1(b) is identified as the clause that does the real filtering, with `CAP-O-02` as the case that proves it — satisfies 1(a) and fails 1(b) |
| `R-AIC-003` / `S-AIC-007` — a capability distinguished from its three near-neighbours | **PASS** | All three named with a sourced example. The third — a contract property optional for **everyone** — is the one doc 0001 § 3.3 row 1's use of the word *optional* invites a reader to get wrong, and § 3 says so: making tool-call deltas discoverable would license a consumer to demand a delta from a provider that advertised them, "the exact assumption row 1 exists to forbid" |
| `R-AIC-005` — token counting optional, argument recorded in full | **PASS** | § 3.2. Loses on universality, not on effort; the opposing reading is conceded as correct as far as it goes |
| `R-AIC-007` — the nine-row leakage register cross-checked row by row | **PASS**, and it is the strongest guard in the artifact | § 8 gives a verdict per row. **The finding is the value: nine documented divergences produce zero optional capabilities on their own.** Two rows have a capability *adjacent* to a contract item, and in both the capability is the emitting or honoring half, never the neutral shape. Promoted to § 13 rule 3, the divergence rule |
| `R-AIC-009` — the marking rule makes the acceptance criterion satisfiable | **PASS**, by identifying that the obvious reading is not satisfiable | § 4 |
| `R-AIC-011` — wrapping addressed | **PASS**, and it is the artifact's own contribution | The rule appears in no source document; AI-37 named before it writes a wrapper |
| `R-AIC-013` — `absent` and `not exercised` distinct | **PASS** | § 3.5. The verdict rule is what stops the four-value set collapsing to three |
| `R-AIC-015` — scope discipline | **PASS** | Checked each named owner individually: no failure category (AI-19), no finish-reason value definition (AI-13), no marker cap number (AI-11), no contract declaration (AI-20), no vendor named or guessed (AI-24.1 — *"This decision names none and guesses at none"*), no answer on whether the first adapter emits reasoning (AI-29.0, whose decision this artifact's only contribution is to make **both** answers legal). `S-AIC-049` … `S-AIC-051` hold |
| `S-AIC-052` — every Layer 1 noun resolves to a register row | **PASS** | **60 distinct `V-*` identifiers cited; all 60 resolve.** Zero dangling citations |
| `S-AIC-058` — no Go identifiers | **PASS** | 0 camelCase and 0 PascalCase tokens across all six files, on regexes verified non-vacuous against AI-00's change. § 9 is where the temptation is highest — doc 0001 § 6 seam 6 itself says "discovered by type assertion" — and the artifact renders it descriptively as "an additional, separately-asserted contract on the provider value" throughout |
| `S-AIC-059` — diff hygiene | **PASS** | Seven files, all markdown under `openspec/changes/`; nothing under `backend/`; no build, module or infrastructure file |

---

## 7. Findings

Three. One is a real navigability defect; two are observations.

### 7.1 § 3 quotes the register row it must not re-define — and the quotation holds

`S-AIC-053` forbids a definition of an appended noun appearing in this change's artifacts "other than as a citation". § 3 opens:

> **A capability is a behavior a consumer can ask a provider for, and whose presence or absence the consumer can observe.**

Compared word for word against `V-PRV-16`'s definition text in the register: **identical**. It is a quotation of the owning row, and § 3 frames it as one (*"`V-PRV-16` names the unit. Its content, for this decision's purposes"*). `CAP-O-02`'s opening does the same for `V-PRV-17`.

**Holds.** Recorded because it is the closest call in the change and it is the exact shape a genuine violation would take — a downstream artifact carrying the definitive wording of a term it does not own. Verifying it required a text comparison rather than a reading.

### 7.2 Seven cross-references point at unnumbered headings — **MINOR**

`decision.md` cites its own subsections `§ 6.1`, `§ 6.2` and `§ 6.3` seven times — for instance *"§ 6.2's corollary"*, *"honoring is `CAP-O-03` (§ 6.3)"*, *"`CAP-O-01` carries `V-REQ-11`'s byte-exact round trip with it (§ 6.1)"*, and § 1's *"§ 6.2 is where the argument is load-bearing"*.

No heading in § 6 carries those numbers. The three subsections are headed `### \`CAP-O-01\` — reasoning content`, `### \`CAP-O-02\` — token counting` and `### \`CAP-O-03\` — honoring cache-boundary markers`, while the fourth is headed `### 6.4 Why the list is exactly three`. § 10 has the same asymmetry: § 10.4 is numbered and its five siblings are not.

Every reference is **correct by position** — § 6.1 is the first subsection of § 6, and so on — so no reader is misdirected to the wrong content. But § 1's reading guide sends a reviewer to "§ 6.2", which cannot be found by searching for that string. **Not a checklist failure and not a spec violation** — no requirement constrains heading numbering — but a real navigability defect in an artifact whose § 1 is explicitly a "read only the section you need" guide. Worth numbering the three headings, or renumbering § 6.4, at archive.

### 7.3 A new identifier namespace, and why it does not belong in the register

`CAP-R-01` … `CAP-X-04` are twelve identifiers this decision introduces and AI-01's register does not carry. They are **entry identifiers for lists this decision owns**, not Layer 1 nouns, and they are used only inside this change — confirmed: `grep -rl 'CAP-[ROX]-'` across `openspec/changes/` matches only this change's five files.

AI-01 § 9 rule 2 binds Layer 1 *nouns*, and the nouns these entries are made of (`V-PRV-06` required capability, `V-PRV-07` optional capability, `V-PRV-16` capability) are all in the register and all cited. **Not a violation.** Recorded because a downstream milestone citing `CAP-O-02` is citing a row in *this* artifact, not in the register, and this report is where that should be written down once — AI-23, AI-24 and AI-38 will all do it.

---

## 8. Cross-artifact consistency

Checked in both directions. A wave of three coupled decision artifacts fails at the joints.

| Check | Result |
| --- | --- |
| Every Layer 1 noun in AI-03 resolves to AI-01's register | ✅ **60 of 60** distinct `V-*` citations resolve |
| AI-03 defines no Layer 1 noun locally | ✅ the three it needed were appended to the register; § 5.4 and § 7.1 |
| AI-03 uses **AI-02's exact terms** | ✅ `CAP-R-04` cites AI-02 § 5 for cancellation and `CAP-R-05` cites AI-02 § 7 for the two delivery paths, both by section and both without restating the rule as its own |
| **AI-03's required capabilities contradict nothing AI-02 settled about cancellation** | ✅ `CAP-R-04` reproduces AI-02 § 5's three obligations exactly — caller-owned signal (`V-STR-06`), every send waits, bounded close — and names `V-STR-09`'s sanctioned loss path as the **only** exception, which is AI-02 § 6's own wording. It adds the observation that the standing is *by construction* (an adapter lacking cancellation cannot be built), which is a consequence of AI-02's placement of the signal on the creating call, not a new obligation |
| **AI-03's required capabilities contradict nothing AI-02 settled about failure delivery** | ✅ `CAP-R-05` requires classification through one vocabulary **on both delivery paths** — `V-FAIL-11` before handover, `V-FAIL-12` after — which is AI-02 § 7's boundary verbatim in substance. Its "one precision" (a provider whose mid-stream failures arrive as an untyped closure **fails** this capability rather than being partially conformant) is a strengthening consistent with `V-FAIL-12`'s "one vocabulary, two delivery paths", not a departure |
| `CAP-R-03` versus AI-02's non-normal endings | ✅ scoped to "a stream that finishes **normally**", which leaves AI-02's cancellation ending and its sanctioned bare close untouched |
| `CAP-X-03` versus AI-02's lifecycle | ✅ the exclusion is **derived from** AI-02 — a batch job has nothing to cancel and nothing to close, so every clause of AI-02's lifecycle fails on it |
| AI-02's forward reference to AI-03's standings | ⚠️ AI-02 § 10 states that cancellation and typed failure delivery *"are required capabilities"* — a standing only AI-03.1 may assign. AI-03 reached the same classification independently through admission test 1, so **no contradiction landed**; recorded in AI-02's verify-report § 7.2 as an observation |
| Doc-0002 acceptance clauses quoted by AI-03 | ✅ four quotations re-resolved against doc 0002 and present verbatim: AI-20.5 item 1's "not an error and not a zero"; AI-29.0's "or documents a capability absence"; AI-38.1 item 2's "There are no waivers; that is what…"; the wave-0 exit condition |
| doc 0001 quotations carrying argumentative weight | ✅ re-resolved and present verbatim: § 6 seam 6's "wrong by enough to matter"; § 6 seam 2's "the event stream stops being a complete description of the session"; § 8's "no constructor produces them" |

### 8.1 One discrepancy in a source, recorded by the artifact rather than inherited

§ 8 records that doc 0001 § 3.3's preamble says "Three require a contract change; the rest are absorbed inside an adapter", while its own table marks rows 8 and 9 as carrying a Layer 1 contract half. The artifact resolves it: the preamble counts **G12**'s three rows only, while rows 8 and 9's contract halves belong to **G4** and are scheduled at AI-10 and AI-11. It then states the consequence of getting it wrong — *"Reading 'the rest are adapter-local' as covering rows 8 and 9 entirely would delete AI-11 from the plan."*

That is a source defect found and neutralised rather than propagated, and it is the kind of finding a nine-row cross-check exists to produce.

---

## 9. Out-of-scope confirmations

Verified **not** done, each deliberate:

- **Nothing under `backend/`.** `git show --name-only f701e58` touches it zero times. The ten `backend/` paths in `origin/main..HEAD` are all AI-00's — `backend/agent/**` plus `backend/database_administrator/src/domain/imports_test.go`.
- **doc 0002 not amended.** No node added, none renumbered, no stated claim corrected — so there is no living-graph amendment to record.
- **No vendor named or guessed.** § 12: *"Not inherited, and deliberately not pre-empted: which vendor. This decision names none and guesses at none."* Confirmed by inspection — no vendor name appears anywhere in the change.
- **AI-29.0 not pre-empted.** The artifact's only contribution is making **both** answers legal; § 13 rule 7 states the abstention as a rule (*"a sentence here that answered either would delete a decision node"*), and `tasks.md`'s review focus 4 tells the reviewer to hunt for a violation of it.
- **No contract declared.** § 12: *"Not inherited, because it is AI-20's: every declaration — the provider contract, the optional contracts, and their spellings."*
- **No conformance assertion written.** § 12: *"Not inherited: every assertion. This decision supplies the list and the marking rule, not the cases."*
- **No failure category, no finish-reason value, no marker cap number.** AI-19, AI-13 and AI-11 named respectively.
- **No Layer 1 default implementation of an optional capability.** § 9's exclusions and § 13 rule 4 — the abstention that follows from `CAP-O-02`'s own argument.

---

## 10. Verdict

**PASS.** All five closing-checklist items are closed, each against the property that would have made it merely covered:

- The **required list** carries a negative clause per entry, because the expensive defect in a required list is an entry that forces an honest adapter to fabricate — and the two readings that would do so are both inside `CAP-R-03` and both closed.
- The **optional list** is three entries, each with its reason and with what a consumer does on a recorded absence; "anything else v1 admits" resolves to nothing with five candidates and their failing clauses recorded; and `CAP-O-02` loses on universality-without-a-lie after conceding the opposing reading in full.
- The **excluded list** carries "why not optional" as well as "why excluded", under one rule that a later reader can apply: an optional capability has a defined absence, an excluded one has no defined presence.
- The **discovery mechanism** states both halves, declares the inherited part as inherited rather than dressing it as an argued conclusion, binds each side with three rules, rejects six alternatives, and contributes the wrapper forwarding rule that appears in no source document.
- The **capability record** makes "absent is a recorded outcome" structural rather than editorial — a closed four-value set in which `not exercised` is a different value, with a verdict rule that stops the set collapsing to three at the first verdict.

The acceptance criterion is met by the route § 11 supplies, not by the one the criterion's wording suggests: the artifact found that the required list cannot be the marking source, said so, and gave a biconditional over the optional list with a **required** default that is total over cases and fails loudly rather than silently.

Cross-artifact consistency holds in both directions: 60 of 60 register citations resolve, AI-02's settled semantics for cancellation and failure delivery are cited rather than re-decided, and no contradiction with either predecessor was found.

The register amendment is append-only — five added lines, one arithmetic replacement, no existing row disturbed — the register counts **114** as claimed in every category and in the sum, and all three appended definitions defer their substance to AI-03 by name.

One minor finding (§ 7.2, seven cross-references to unnumbered headings) and two observations, none of which blocks.

**Wave 0 closes here.** doc 0002's exit condition — the module exists, both import directions bite, and vocabulary, stream lifecycle, carrier and capability scope are recorded decisions — holds on this merge.

**Ready for archive** once the wave's PR merges.
