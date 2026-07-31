# Verify report — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-02--decide-stream-lifecycle-ownership-and-the-carrier)
> **Node**: AI-02.1 — Lifecycle, ownership and carrier `[decision]`
> **Phase**: verify
> **Status**: **PASS**
> **Date**: 2026-07-31
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Base**: `origin/main` @ `b6c59e6`
> **Change commit**: `6da8593`
> **Closes**: the concern doc 0001 and ADR 0005 track as **G13**
> **Mode**: `[decision]` leaf — no production code, no `make test` gate. Every check below is inspection; citations were re-resolved against their sources rather than taken on the artifact's word.

---

## 1. Charter acceptance

| # | Charter clause | Verdict | Evidence |
| --- | --- | --- | --- |
| 1 | A recorded decision covering **carrier, ownership, cancellation, buffering and the failure-delivery split** | **PASS** — five decisions, five sections, in checklist order | § 3 |
| 2 | AI-14 … AI-20 can be written **without reopening any of these five questions** | **PASS** | § 4 |
| 3 | AI-34's buffer measurement has a stated starting point to confirm or change | **PASS** — capacity 64, three measurements, a direction per result, and a tie-break rule | § 3.4 |
| 4 | doc 0002's note: the carrier choice is free here and only here, and **the SDD must record why it chose what it chose** | **PASS** — this is the clause most at risk, and § 3.1 is where the verification concentrates | § 3.1 |

Clause 4 is the load-bearing one. The documented default was channels, and the decision is channels. An artifact can satisfy the letter of "decide the carrier" while restating the default, and doc 0002 anticipated exactly that by deleting half of the argument that previously supported it. § 3.1 checks whether the artifact argued or restated.

---

## 2. Deliverable inventory

| Path | Present | Note |
| --- | :---: | --- |
| `decision.md` | ✅ | 515 lines; §§ 3–7 are the five checklist items in the checklist's own order |
| `proposal.md` | ✅ | |
| `specs/ai-stream-lifecycle/spec.md` | ✅ | `R-AIS-001` … `R-AIS-014`, `S-AIS-001` … `S-AIS-041` |
| `design.md` | ✅ | names the two failure modes it targets: the restated default, and the unfalsifiable constant |
| `explore.md` | ✅ | |
| `tasks.md` | ✅ | six tasks + a ten-check verification pass, all `[x]` |
| amendment to `../cachicamas-ai-contract-vocabulary/decision.md` | ✅ | append-only; § 5 |
| anything under `backend/` | ✅ absent | § 7 |

`git show --name-only 6da8593` lists **seven files**: the six above plus the register amendment. `tasks.md` acceptance criterion 5 — "adds six markdown files and amends exactly one existing file, append-only" — holds.

---

## 3. Closing-checklist closure

For a `[decision]` leaf the verification is that each item is **closed** — that AI-14 … AI-22 could be written from this artifact without reopening it — not that the topic is discussed. Each of AI-02.1's five items is checked adversarially below.

| # | Item | Verdict |
| --- | --- | --- |
| 1 | Carrier decided **with rationale**, plus the branch consequence | **CLOSED** — § 3.1 |
| 2 | Ownership: exactly once, **stated not implied**, across all three exit paths | **CLOSED** — § 3.2, with one qualification about *where* the per-path statements live |
| 3 | Cancellation, plus the abandonment statement in the package contract | **CLOSED** — § 3.3 |
| 4 | Buffering: bounded, with a **decided starting capacity**, plus the sanctioned loss path | **CLOSED** — § 3.4 |
| 5 | Failure delivery: the pre-stream versus mid-stream split | **CLOSED** — § 3.5 |

### 3.1 Item 1 — does the carrier argument argue, or restate the default?

The test applied: **strike every sentence that could have been written before reading doc 0001 and doc 0003, and see whether a decision survives.** It does. Four properties, each checked:

**(a) The opposing case is stated at full strength, and three of its four points are conceded as structural.** § 3's "First, the iterator case, at its strongest" lists four advantages and says of three of them that no amount of discipline buys them back on the channel side — the stranded-producer hazard becoming an impossibility, abandonment becoming supported, and the buffering question dissolving. `S-AIS-003` requires exactly these three; all three are present, affirmatively, before any rebuttal.

**(b) Every defeating ground cites a source, and the citations resolve.** This was checked against the sources rather than accepted:

| Ground | Citation | Re-resolved |
| --- | --- | --- |
| 1 — Layer 2 must wait on several things at once | doc 0001 § 4.1: "Suspension must not block the other concurrent calls, and must not block event delivery" | ✅ present, verbatim modulo the leading capital |
| 1 — turned into a test one layer up | doc 0003 **AG-10.3 item 1** | ✅ present at doc 0003 line 530, **verbatim**, including "message deltas already in flight keep flowing" |
| 2 — the consumer must not be the socket reader | `V-STR-08`'s "an unbounded buffer converts backpressure into memory growth" | ✅ resolves to a live register row |
| 3 — the terminal event has nowhere clean to go | `V-STR-18`, `V-STR-20`, `V-FAIL-10`, `V-FAIL-09` | ✅ all four resolve |
| 4 — the shape carries the wrong connotation | `V-PRV-10`'s wrong-physics clause | ✅ resolves |

Ground 1 is the decisive one and it is the one that is **documented rather than anticipated** — a Layer 2 test already written in doc 0003 that an iterator boundary would make unsatisfiable. That is the difference between an argument and a preference.

**(c) No ground is cost of change.** `S-AIS-004` forbids any appeal to shipped guards, merged scenarios, existing signatures or migration cost. Scanned § 3 for all four: the only mention is the explicit statement that the argument is **void** ("This decision is made as though the module were empty, because it is"), which is the single permitted form. The artifact goes further and says so in bold — *"No ground below is a cost-of-change ground."*

**(d) The cost of the chosen option is conceded, not hidden.** § 3's "What is conceded" states that the residual — a consumer who abandons and never cancels — is real, is the iterator's strongest single advantage, is not preventable by the type, and is not testable to termination. It then names the currency it is paid in: a statement in the package contract, restated at the freeze by AI-40.3. An artifact that reached the default without naming what the default costs would have failed this item; this one names it in bold.

**(e) The branch consequence is stated, and stated positively.** `R-AIS-003` requires exactly one of two branches. The channel branch is taken: iterator ergonomics delegated to AI-22.5 as a `V-STR-22` **carrier view**, "never a second contract", with AI-20.4's signature guard passing unmodified as the mechanical form of that claim. doc 0002's waves 2–5 gain **no** amendment nodes, and § 3's consequence 2 records the absence deliberately "so nobody later reads the absence as an omission."

**(f) The third option is named and priced.** "Both carriers at the boundary" is rejected as the union of both costs, with four enumerated: the conformance suite proves every contract twice, the fake implements both faithfully, AI-20.4 pins two shapes, and Layer 2 splits into two dialects. `S-AIS-006` holds.

**Verdict: the argument argues.** It reaches the documented default by a route that does not depend on the default.

### 3.2 Item 2 — ownership, and the one qualification

The item's own wording is the demanding part: what "exactly once" means across completion, error and cancellation must be **stated, not implied**.

§ 4 states it structurally in five clauses — one sending goroutine, one closing site total (not one per path), running on every exit path including an unwinding one, running after the last send attempt and never before, and nothing else closing. Clause 4 is the elegant part: with a single sender whose final act is the close, "no send after close" needs no separate rule because it is unreachable. Three misreadings are named with the defect each produces, and each misreading is a **shape** — a second closing site, several senders, a consumer-side close — which is what makes them visible in a diff rather than only in a race.

**The qualification.** `S-AIS-011` asks that the three paths "each carry their own statement of what is emitted and when the close happens." § 4 names all three (plus the unwinding exit) in a single enumerated clause; the statement of *what is emitted* on each path lives in § 6 (the cancellation branch), § 7 (completion event, terminal error event, ordinary cancellation), and the § 8 lifecycle diagram, which draws all three paths converging on one closing site.

Judged **satisfied across the artifact, not within § 4**. The checklist item asks that the meaning of "exactly once" be stated for all three paths — § 4 does that explicitly and unambiguously. The per-path *emission* statements are one section away, and the § 8 diagram makes the join visible in one picture. This is a structural observation about where the reader finds the answer, not a gap in the answer.

`S-AIS-013` also holds: the consumer, the stream test kit, Layer 2's harness and a frontend are each named as parties that do not close, and § 4's exclusions add a wrapper, a decorator, and a `V-STR-22` carrier view.

### 3.3 Item 3 — cancellation, and the statement that cannot be tested

Three obligations, each **classified** as structural or testable with the proving node named — caller-owned context (structural, seen by AI-20.4's signature guard), every send waits on both (testable, AI-20.3 item 2 and AI-33), bounded close (testable, AI-20.3 item 4). `S-AIS-014` holds.

"Bounded" is defined **by what it excludes**: after cancellation is observable the producer begins no new blocking wait on the network and none on the consumer, and a backoff waits on the signal rather than sleeping. `S-AIS-015` holds, and the sleeping-backoff case is named as the most common way a bounded close becomes unbounded.

The item's hard half is the abandonment clause, and the artifact handles it in the only way that survives:

- The **legal endings are enumerated first** — drain to close, or cancel — so that "violation" has a complement and a consumer with a legitimate need to stop early has a one-word answer rather than a prohibition. `S-AIS-017` holds, and § 5's "Why" explains that the enumeration is what makes the prohibition usable.
- The violation is stated as a pull quote, assigned to the **package contract**, justified by untestability ("no test proves a goroutine never exits, and a bounded observation that it has not exited *yet* is a strictly weaker claim"), and attributed to **AI-40.3** at the freeze. `S-AIS-016` holds on all four counts.
- § 11 rule 3 promotes the reasoning to a standing rule: *an untestable obligation is marked as such and lives in the package contract; it is never given a test that proves something weaker, because the weaker property then replaces it.* Defect **C3** is cited as what that substitution looks like when it ships.
- § 5's consequences draw the line precisely: leak assertions belong on the abandoned-**then-cancelled** path, which is testable; the abandoned-**never-cancelled** path gets no test pretending otherwise.

§ 9 statements 1–8 are the package-contract form, and AI-20.1's three documentation obligations — who closes, who owns the context, what abandoning without cancelling means — are quotable verbatim from statements 1, 2 and 4. `S-AIS-018` holds.

### 3.4 Item 4 — a decided starting capacity, and whether it is falsifiable

**One number: 64.** Not a range, not a preference, not a deferral. `S-AIS-019` holds.

The justification is checked against `S-AIS-020`'s requirement that both ends of the range be priced. § 6 gives a four-row table — zero (rendezvous), single digits, 64, hundreds or more — with what each costs. Two of those costs are non-obvious and both are stated:

- A rendezvous has maximum backpressure fidelity and **zero tolerance for a consumer that pauses at all** — and Layer 2's consumer pauses **by design**, to drive the permission protocol. The ground for rejecting the smallest option is Layer 2's documented behavior, not a preference.
- Hundreds or more makes "does the producer ever wait?" **unobservable**, which would defeat the very measurement AI-34 exists to take, and it **widens the drop window** — events lost on the sanctioned path are at most the events resident.

`S-AIS-021` holds: capacity is paid per live stream, and the concurrency sources are named — parallel tool-driven calls (**G5**), compaction calls with their own provider and cancellation, nested subagent runs (**G7**). The artifact then does something better than inflate the number's importance: it states that at 64 small events a dozen concurrent streams is still kilobytes, so **memory is explicitly not the ground the number stands on**. It stands on burst absorption, bounded by the drop window.

**Falsifiability** — `R-AIS-010`, and the second failure mode `design.md` names. The number is labelled a hypothesis; three measurements M1–M3 are specified with the direction each result implies in both directions; and a **tie-break rule** is supplied so AI-34.1 is not left arbitrating — when two capacities are indistinguishable, prefer the smaller, because observable backpressure is worth more than hidden latency and the drop window is smaller. Constant-versus-configurable is explicitly deferred to AI-34.1. § 6's exclusions close it: *"Any claim that 64 is measured. It is not."* And § 11 rule 5 makes citing 64 as settled a misreading and citing it after AI-34.1 publishes a stale citation.

`S-AIS-024`, `S-AIS-025`, `S-AIS-026` all hold: waiting stated as the full-buffer behavior and dropping stated as not being it; the sanctioned loss path stated with all three elements plus the clause that a consumer treating a missing terminal **after its own cancellation** as corruption is the party in error; and a stream that closes bare **without** having been cancelled stated as a producer defect rather than a second loss path.

### 3.5 Item 5 — the delivery split, at an observable moment

**The boundary is the handover of the carrier**, not the first event.

The item is closed by three properties, and the middle one is the one an artifact usually misses:

1. **Pre-stream is stated as what the caller observes, negatively as well as positively.** Not an empty stream, not a stream that immediately yields a failure — nothing to drain, nothing to close, no goroutine. `S-AIS-027` holds. § 7's exclusions price both wrong shapes, including that a stream-that-immediately-fails would make AI-20.2 item 1 untestable.
2. **The one case that separates the correct boundary from the intuitive one is stated explicitly:** a stream handed over that fails **before emitting any content** is mid-stream. The rejected alternative — "the first event" — is named, and the reason it fails is that it fuses the delivery axis onto the partial-output axis and rebuilds the **G8** defect. The doc 0001 § 7 **G8** quotation was re-resolved against the source and is **verbatim**: *"a stream that dies after emitting output is the most common real failure and the one naive retry logic excludes."* `S-AIS-029`, `S-AIS-030` hold.
3. **The orthogonality is drawn, not asserted.** § 7's two-axis diagram separates the DELIVERY axis (decided by handover) from the PARTIAL-OUTPUT axis (`V-FAIL-09`, "a DIFFERENT fact"). This is the second orthogonal pair in the wave — AI-01 § 6 drew owner-versus-delivery — and keeping them apart is what lets AI-19 classify and AI-35 decide retries without either reaching for the other's fact.

`S-AIS-028` holds: mid-stream failures arrive as the terminal error event and **there is no second route** — not a re-inspected return value, not a side channel, not an accessor. `S-AIS-031` holds: the same category, retryability and partial-output discriminator are reachable on both paths, which is the property that makes AI-19.5 item 3 satisfiable.

The artifact also answers a question the checklist does not ask but AI-20.2 item 2 already assumes: **what a caller observes on ordinary cancellation.** The producer offers a terminal error event **without waiting**; whether or not the offer lands, it closes. The offer must not wait because waiting would violate § 5's bounded close — which makes the sanctioned loss path the *intersection* of a bounded close and a full buffer, and nothing else. That derivation is what turns § 6's loss path from a stipulation into a consequence.

---

## 4. The acceptance criterion, checked directly

> *"AI-14 … AI-20 can be written without reopening any of these five questions; AI-34's buffer measurement has a stated starting point to confirm or change."*

doc 0002 names **four** milestones as blocked by AI-02: AI-14, AI-20, AI-21, AI-22. § 10 gives each an inheritance statement in that milestone's own terms, and AI-20 is broken down node by node.

| Blocked milestone | Inheritance stated | Node-level? |
| --- | :---: | --- |
| AI-14 — the event envelope | ✅ | carrier settled; per-stream sequence made achievable by § 4's single-sender rule; `V-STR-18` cited not restated; **what is not inherited** named (event kinds, payloads, sequence semantics, ordering invariants) |
| AI-20 — the provider interface | ✅ | AI-20.1, AI-20.2, AI-20.3 (item by item), AI-20.4 each mapped to the section that supplies it |
| AI-21 — the scripted fake | ✅ | § 9 binds the fake; AI-21.3, AI-21.4, AI-21.5 each mapped |
| AI-22 — the stream test kit | ✅ | AI-22.1, AI-22.4, AI-22.5; AI-22.5's "inverts if iterators won" note explicitly retired |

AI-15 … AI-19 are **not** directly blocked by AI-02 — doc 0002 has them depending on AI-14 — so the "AI-14 … AI-20" range is satisfied through AI-14's inheritance, and AI-19.5 additionally has its own row in § 10's downstream table ("§ 7 entire").

The second clause holds: § 10 hands AI-34.1 "§ 6's starting capacity of **64**, its three measurements, the direction each result implies, and the tie-break rule — an experiment, not an opinion."

**A stronger test than the table.** AI-03 was written from this artifact in the same wave. Its `CAP-R-04` cancellation and `CAP-R-05` typed-failure entries cite AI-02 §§ 5 and 7 for their observable shapes and **re-decide neither**. That is the acceptance criterion demonstrated rather than asserted.

---

## 5. Register integrity — the AI-02 amendment

Two nouns appended to AI-01's register in this same pull request, under AI-01 § 9 rule 2.

| Appended | Ordinal | Owner | Why the register lacked it |
| --- | --- | --- | --- |
| `V-STR-22` **carrier view** | next free in `V-STR` | AI-02 | AI-02.1 delegates iterator ergonomics to AI-22.5 and had no noun for the delegated thing. Without one every downstream restatement says "iterator view" — a phrase welded to one carrier choice |
| `V-STR-23` **backpressure** | next free in `V-STR` | AI-02 | The word **already appeared inside `V-STR-08`'s definition without being defined** — exactly the drift the register exists to prevent |

### 5.1 Measured, not claimed

Diff of `cachicamas-ai-contract-vocabulary/decision.md` from `3a48014` to `6da8593`:

```
+ dated amendment blockquote under § 4
+ (blank separator)
+ | `V-STR-22` | carrier view    | … | AI-02 | …
+ | `V-STR-23` | backpressure    | … | AI-02 | …
- | 2 | Stream-side terms: … | § 4 — `V-STR-01` … `V-STR-21`; …
+ | 2 | Stream-side terms: … | § 4 — `V-STR-01` … `V-STR-23`; …
- **Term count:** … 21 stream-side … = **109 terms**.
+ **Term count:** … 23 stream-side … = **111 terms**. *(Amended 2026-07-31: …)*
```

**Four additions and two replacements, in a 328-line file.** Both replacements are the register's own arithmetic. `S-AIS-038` holds as a measured property: **no existing row was renumbered, reworded, reordered or removed.**

### 5.2 Counts after this amendment

Register total **109 → 111**. Verified by counting rows, not by reading the claim: stream-side rows after this commit = **23** (`V-STR-01` … `V-STR-23`, contiguous, no gap, no reuse), and § 10's per-category figures and sum matched at that revision. The register today, after AI-03's later amendment, counts **114**, and every category cell in § 10 matches its actual row count.

### 5.3 Neither term is defined locally

`S-AIS-037` requires the appended nouns to appear in this change's artifacts only as citations. Both do: `V-STR-22` and `V-STR-23` are cited by identifier in `decision.md` §§ 2, 3, 4, 6, 10 and 12, and no file of this change carries a competing definition. § 1 states the discipline in the artifact's own voice: *"They are used here; they are not defined here."*

---

## 6. Spec conformance

All fourteen requirements in `specs/ai-stream-lifecycle/spec.md` verified. Listed below are the clauses where the verdict needed judgement rather than a count.

| Requirement | Verdict | Note |
| --- | --- | --- |
| `R-AIS-002` — the carrier is argued and rests on no cost-of-change ground | **PASS** | § 3.1(a)–(d). Checked by re-resolving each ground against doc 0001 and doc 0003, not by reading the artifact's summary of them |
| `R-AIS-005` / `S-AIS-011` — "exactly once" stated separately for all three paths | **PASS**, qualified | § 3.2. § 4 names all three paths plus the unwinding exit explicitly; the per-path *emission* statements live in §§ 6, 7 and the § 8 diagram |
| `R-AIS-010` — the capacity is falsifiable | **PASS**, and stronger than the requirement | The requirement asks for measurements and directions; the artifact also supplies a **tie-break rule**, which is what stops AI-34.1 arbitrating between indistinguishable results |
| `R-AIS-012` / `S-AIS-029` — the case that separates the boundaries | **PASS** | The stream-handed-over-then-fails-before-content case is stated explicitly and tied to the **G8** defect by a verbatim doc 0001 quotation |
| `R-AIS-013` — the artifact stays inside its own scope | **PASS** | Checked the four named owners individually: no event kind, payload, sequence rule or ordering invariant is decided (deferred to AI-14, and `V-STR-18` is *cited* not restated); no failure category, retryability rule or terminal payload shape is decided (AI-19); no leak-detection mechanism is chosen (AI-22.4, named); no retry rule is stated (AI-35, named). See § 7.2 for the one sentence that reaches outside this list |
| `R-AIS-014` — vocabulary discipline | **PASS** | **35 distinct `V-*` identifiers cited; all 35 resolve** to live register rows. Zero dangling citations |
| `S-AIS-040` — no Go identifiers | **PASS** | 0 camelCase and 0 PascalCase tokens across all six files, on regexes verified non-vacuous against AI-00's change. The literal token `func` appears only inside *range-over-func*, the language feature's own hyphenated name; § 3's own preamble states that language shapes are named descriptively ("the single-value iterator function shape") **on purpose**, and that discipline holds inside the argument that most tempts a reader to break it |
| `S-AIS-041` — diff hygiene | **PASS** | Seven files, all markdown under `openspec/changes/`; nothing under `backend/`; no build, module or infrastructure file |

---

## 7. Findings

Three. One is a genuine defect of hygiene; two are observations.

### 7.1 A quotation that is not verbatim — **MINOR**

§ 5 attributes to doc 0001's defect **C3**:

> "a shipped test documented the resulting gaps as expected behavior."

doc 0001's C3 row actually reads:

> A shipped test documents the resulting gaps as expected

The tense is changed (`documents` → `documented`) and the word **behavior** is added, all inside quotation marks. The substance is preserved exactly and the citation points at the right defect, so nothing downstream is misled about the fact.

It is recorded as a finding because this change's own § 11 rule 4 and AI-01 § 9's anti-paraphrase discipline are what the register exists to enforce, and a paraphrase inside quotation marks is the first step of the drift both rules name. **Not a checklist failure**; worth correcting to a verbatim quotation or an unquoted paraphrase at archive.

For contrast, the two quotations that carry real argumentative weight were both checked and are both **verbatim**: doc 0003 AG-10.3 item 1 (ground 1's decisive citation) and doc 0001 § 7 **G8** (§ 7's rejection of "the first event").

### 7.2 One sentence states a standing that AI-03 owns

§ 10's downstream table says of AI-03:

> Cancellation and typed failure delivery **are required capabilities** whose observable shape is already fixed by §§ 5 and 7, so the matrix can mark them without re-deciding them.

Assigning a capability's standing is AI-03.1's, not AI-02.1's. `R-AIS-013`'s scope list names AI-14, AI-19, AI-20, AI-22.4, AI-34 and AI-35 — not AI-03 — so this is **not a spec violation**, and doc 0002 lists AI-03 as depending on AI-02, which makes a forward-looking note legitimate.

It is recorded because the sentence is phrased as fact rather than as expectation. AI-03 independently reached the same classification through its own admission test 1, so no contradiction landed and nothing needs changing. Had AI-03 classified either differently, this row would have been stale on the day it merged.

### 7.3 A path token the sibling spec's wording reaches

`S-AIS-040` forbids "a package path" among other things. The artifact's header carries `**Target package**: `backend/agent/src/ai/` (Layer 1)`, and § 12 says "there is nothing in `backend/agent/` that this change touches". These are repository directory paths in metadata, not Go package paths (`github.com/cachicamas/backend/agent/src/ai`), and neither is a spelling of any Layer 1 surface. Doc 0002's authoring constraint — no Go identifiers, each milestone's SDD chooses spellings — is untouched. **Not a violation**; recorded because the literal reading of the scenario is tighter than the constraint behind it.

---

## 8. Cross-artifact consistency

Checked in both directions, because a wave of three coupled decision artifacts fails at the joints rather than in the middle.

| Check | Result |
| --- | --- |
| Every Layer 1 noun in AI-02 resolves to AI-01's register | ✅ 35 of 35 distinct `V-*` citations resolve |
| AI-02 defines no Layer 1 noun locally | ✅ the two it needed were appended to the register instead; § 5.3 |
| AI-02's decisions are consistent with AI-01's definitions | ✅ § 4's ownership instantiates `V-STR-05` and doc 0001 § 9's "Nothing closes a channel it does not own" (verified verbatim in doc 0001); § 6's loss path instantiates `V-STR-09`; § 7's split instantiates `V-FAIL-11`/`V-FAIL-12`, including `V-FAIL-12`'s "one vocabulary, two delivery paths" clause |
| AI-03 uses AI-02's settled semantics without re-deciding them | ✅ `CAP-R-04` cites AI-02 § 5 for cancellation (caller-owned signal, every send waits, bounded close, `V-STR-09` the only exception); `CAP-R-05` cites AI-02 § 7 for the two delivery paths |
| AI-03 contradicts nothing AI-02 settled | ✅ none found. `CAP-R-03` is scoped to "a stream that finishes **normally**", which leaves AI-02's cancellation and sanctioned-loss endings untouched; `CAP-X-03` excludes batch APIs *because* every clause of AI-02's lifecycle assumes a live cancellable stream — an exclusion derived from this decision rather than in tension with it |

---

## 9. Out-of-scope confirmations

Verified **not** done, each deliberate:

- **Nothing under `backend/`.** `git show --name-only 6da8593` touches it zero times. The ten `backend/` paths in `origin/main..HEAD` are all AI-00's — `backend/agent/**` plus `backend/database_administrator/src/domain/imports_test.go`.
- **doc 0002 not amended.** No node added, none renumbered, no stated claim corrected. § 3's consequence 2 records this as a **positive result** of the channel branch rather than an omission: the waves 2–5 amendment nodes the iterator branch would have triggered are not needed.
- **No event kind, payload, sequence rule or ordering invariant.** `V-STR-18` is cited as an assumption of §§ 4, 6 and 7 and explicitly left to AI-14 to define.
- **No failure category, retryability rule or terminal-payload shape.** § 7's exclusions name this abstention: *"This decision fixes where a failure appears, never what it says."*
- **No leak-detection mechanism** (AI-22.4 named), **no retry rule** (AI-35 named), **no constant-versus-configurable decision** on the buffer (AI-34.1 named).
- **Layer 2's carrier not decided.** § 11 rule 6: doc 0003 AG-01.1 owns it; this artifact is an input, "a recommendation with reasons, not an inheritance."
- **No claim that 64 is measured.** Stated three times — in § 6's "Why 64", in § 6's exclusions, and as § 11 rule 5.

---

## 10. Verdict

**PASS.** All five closing-checklist items are closed, each against the property that would have made it merely discussed:

- The carrier is **argued**, on four grounds whose two decisive citations were re-resolved against doc 0001 and doc 0003 and found verbatim, with the opposing case stated at full strength, the third option priced, no cost-of-change ground anywhere, and the chosen option's residual cost conceded in bold.
- Ownership is **structural** — one sender, one closing site, every exit path, after the last send attempt — with the three misreadings named as shapes visible in a diff.
- Cancellation enumerates its **legal endings** so that abandonment has a complement, and places the untestable clause in the package contract with AI-40.3 named rather than giving it a test that would prove something weaker.
- The buffer has **one number, 64**, priced at both ends of the range, and it is falsifiable: three measurements, a direction per result, a tie-break rule, and an explicit denial that the number is measured.
- The delivery split is separated at an **observable moment**, with the one case that distinguishes it from the intuitive boundary stated explicitly and tied to the **G8** defect.

The acceptance criterion holds directly — all four blocked milestones have node-level inheritance statements, AI-34.1 inherits an experiment rather than a constant — and it holds by demonstration: AI-03 was written from this artifact in the same wave and re-decided none of the five.

The register amendment is append-only, measured at four added lines and two arithmetic replacements with no existing row disturbed.

One minor finding (§ 7.1, a paraphrase inside quotation marks) and two observations, none of which blocks.

**Ready for archive** once the wave's PR merges.
