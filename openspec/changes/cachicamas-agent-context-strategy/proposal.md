# Proposal: AG-17 — Add the context strategy seam and token accounting

> **Change**: `cachicamas-agent-context-strategy` · **Milestone**: AG-17 (Layer 2 Wave 4, milestone 17 of 24; doc `0003:1600-1653`)
> **Branch**: `feat/agent-layer2-wave4-ag17` · base `origin/main@b0de5bf6` · **Worktree**: `cachicamas-worktrees/ag17`
> **Artifact store**: hybrid (Engram + filesystem) · **Delivery**: single PR, `size:exception` pre-authorised against the 1000-line budget
> **TDD**: strict, RED-first (`cd backend/agent && make test`)
> **Closes**: R-18 **seam 5** (AG-17.1) and **seam 6** (AG-17.2), per `agent-v1-scope/spec.md:145` (`S-AGS-023`). Contributes the pre-condition for R-11 (AG-18).
> **Depends on**: AG-02, AG-12, AG-13 (all archived); AG-15, AG-16 (merged) · **Blocks**: AG-18
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-context-strategy/explore`
> **Bite-ID prefix**: `R-CTX-` / `S-CTX-` — verified free against the 60 prefixes in `openspec/specs/`

---

## Intent

The harness runs turn after turn over a transcript that only grows. Nothing measures it, and nothing is asked whether it should shrink. Layer 2 today has **no budget type at all** — a repo-wide search finds only "retry budget" and "attempt bound" prose — and it discards Layer 1's shipped counting capability without ever asking for it.

AG-17 closes that with the smallest possible thing: a place to stand. It ships the **seam** and the **measurement**, and deliberately ships **no compaction**.

**Two promoted specs already record this as debt owed by AG-17 by name.** Both rows retire with this change:

| Spec row | Text | Retires as |
|---|---|---|
| `agent-run-driver/spec.md:342` | "A compaction check between turns \| **AG-17** inserts it at AG-13's turn boundary; **AG-18** implements compaction." | **CLOSED by AG-17** — the check ships; compaction stays AG-18's |
| `agent-history/spec.md:260` | "Context-window accounting over the transcript \| AG-17" | **CLOSED by AG-17** |

The first row is not merely a debt marker — it is **the placement decision, already made by a promoted spec**. Design does not get to choose the seam's position; it inherits it.

**What ticks**: seams 5 and 6 of R-18; `agent-v1-scope`'s `AGS-S` rows for both gain their back-annotation; AG-18 becomes startable.

---

## Scope

### In

- **AG-17.1** — `ContextStrategy`, a one-method seam interface with typed prompt and verdict structs, mirroring `FailoverPolicy` (`failover_policy.go:27-79`) exactly.
- **AG-17.1** — a nil-default `Harness.ContextStrategy` field, consulted **once per logical turn** in `Harness.Run`'s outer loop, between `transcript` (`harness.go:498`) and the attempt loop (`harness.go:530`). See Decision 1.
- **AG-17.1** — `NoOpContextStrategy`, the shipped installable never-compact default, plus the inertness pin: installing it changes nothing observable versus leaving the field nil.
- **AG-17.1** — `ContextBudget`, the budget's **type**; and the `Harness` field carrying it. Its **value** stays Layer 3's. See Decision 2.
- **AG-17.2** — token accounting by type assertion on the shipped `ai.TokenCounter` (`ai/provider.go:130-134`), yielding a `TokenAccounting` value whose provenance is unreadable-around at the **type** level. See Decision 3.
- **AG-17.2** — the estimate path, with a stated method and a stated accuracy caveat. See Decision 4.
- **AG-17.2** — a new **exported** counting-capable fake in `agenttest`, mirroring `stubProviderWithTokenCounter`'s embedding shape (`agenttest/provider_test.go:133-142`).
- Substrate-guard filter widening in `loop_test.go` and `loop_hook_test.go` by exact filename suffix, byte-in-sync — the AG-11/AG-13/AG-14/AG-15/AG-16 discipline.

### Out — deferred, with the owner named

| Deferred | Owner and why deferral is safe |
|---|---|
| **Compaction itself** — summarising, transcript surgery, protected recent turns, compaction events | **AG-18** (`0003:1655-1665`). `agent-run-driver/spec.md:342` splits the two explicitly: AG-17 inserts the check, AG-18 implements. Decision 1 makes AG-17's verdict type *incapable* of asking for compaction, so this is enforced by the compiler, not by intent. |
| **Any budget value, threshold, or configuration** | **Layer 3.** Charter out-of-scope (`0003:1610`): "Layer 3 supplies the model's budget via options". AG-17 ships the type and the carrying field; every value is the caller's. `R-AGS-015` (`agent-v1-scope/spec.md:245`) forbids Layer 2 deciding Layer 3's content. |
| **When to compact** — any threshold arithmetic over budget vs. accounting | **AG-18 / Layer 3** (`0003:1665`). AG-17 hands the strategy both numbers and asks nothing of them. |
| **Any new `EventKind`** | **Never in AG-17.** `compaction_events.go` already defines `CompactionStarted`/`Finished`/`Failed` from AG-06; **AG-18** is their first caller. AG-17 emits nothing, so `event_descriptor.go` and `event_registry_test.go` stay untouched at their committed kind count. |
| **Any edit under `backend/agent/src/ai/**`** | **Never in Layer 2** (`R-RUN-012`). Layer 1 is consumed, never edited — including `ai.TokenCounter`, which ships complete and conformance-tested (`agenttest/conformance_capabilities.go:152-180`). |
| **A second counting contract in Layer 2** | **Forbidden**, not deferred — `R-AMP-017` (`ai/provider.go:123-125`): one contract per capability. See Decision 3. |
| **A real tokenizer or BPE vocabulary** | **Out of scope and structurally impossible.** `L2C-01` allows the standard library and `src/ai`'s measured closure, nothing else; `openspec/config.yaml:33` forbids new top-level deps without an ADR. The estimate is therefore a documented heuristic by construction, which is exactly why Decision 4 must state its method and its error honestly. |
| **Re-opening AG-15's retry gates or AG-16's accumulator** | **CLOSED by their own milestones.** AG-17 reads the outer loop's state and touches neither. |

---

## Decision 1 — seam placement and the "N times" ambiguity **(DECIDED — a `Harness` field, consulted once per logical turn)**

### 1a. Where the seam lives

`Harness.ContextStrategy`, a nil-default field on the caller-owned harness — **not** on `TurnOptions`.

`failover_policy.go:19-26` states the governing rule verbatim: *"a seam belongs where its consumer is"*, and keeps `FailoverPolicy` off `TurnOptions` because *"exhaustion is a run-driver concept — `Turn` knows nothing of attempts"*. A turn boundary is a run-driver concept by the identical argument: `Turn` has no notion of a logical turn either, because it **is** one attempt.

AG-08's `TurnOptions.PreRequestHook` (`loop.go:326`) is the wrong model to copy for exactly this reason — it fires once per *attempt*.

| Option | Verdict |
|---|---|
| **(a)** `Harness.ContextStrategy`, consulted in the outer loop | **RECOMMENDED.** The only placement a promoted spec authorises. |
| **(b)** A `TurnOptions` field beside `PreRequestHook` | **Rejected — contradicts a promoted spec.** `agent-run-driver/spec.md:342` places the check "at AG-13's turn boundary". It also fires per attempt, which breaks the count under any retry, and (see 1b) would falsify `R-RTY-002` the moment AG-18 gives the seam teeth. |

### 1b. What "N times" counts — the change's central ambiguity, resolved

The AG-17.1 Gherkin (`0003:1627`) reads: *"the strategy was consulted exactly N times, before each provider call"*. Under AG-15's retries these two clauses genuinely diverge — a retried logical turn issues **more than one** `provider.Stream` call (`loop.go:351`) inside **one** outer-loop iteration.

**RESOLVED: `N` counts logical turns, not attempts.** Three reasons, descending in strength:

1. **A promoted spec already says so.** `agent-run-driver/spec.md:342` — "at AG-13's turn boundary". The turn boundary is the outer loop (`harness.go:479`), not the attempt loop (`harness.go:530`).
2. **The per-attempt reading would falsify AG-15's retry pin.** `harness.go:518-529` documents `R-RTY-002` precisely: the attempt loop re-invokes `Turn` *"over the transcript slice built once above and reused **BY REFERENCE** across attempts — this, not an assumption, is what makes 'identical transcript' provable"*. A seam consulted **inside** the attempt loop is a seam that, once AG-18 makes it capable of compacting, could mutate the transcript between two attempts of one logical turn. That does not merely risk breaking the pin — it makes the pin unprovable by the exact argument the comment relies on. Turn-boundary placement keeps `R-RTY-002` true **by construction** rather than by convention.
3. **The sibling seam resolves identically.** Retry timing resolves once per logical turn at `harness.go:505-511`, and its comment gives the same structural reason: *"the harness's analogous unit is one logical turn's own inner attempt loop below, not the whole multi-turn run"*.

**This resolution MUST be stated normatively in the spec, not left implicit.** A downstream phase reading only the Gherkin will otherwise reconstruct the per-attempt reading, and the charter sentence will read as authorising it.

### 1c. The verdict type makes "never compacts" unconstructible

`ContextVerdict struct{}` — the zero value and the only value. v1 ships **no** field by which an implementation could ask for compaction.

This is `FailoverVerdict`'s posture, and `failover_policy.go:57-62` blesses the extension path explicitly: *"A later version adds route and re-budget fields non-breakingly; every existing implementation returning the zero verdict keeps compiling."* AG-18 is that later version. The payoff is that AG-17.1's second scenario — *"the never-compact default changes nothing"* — becomes **unfalsifiable by any implementation a caller could write**, rather than a property of the one implementation we shipped.

---

## Decision 2 — the budget's type **(DECIDED — a presence-carrying value type; zero value is *absent*, never *zero tokens*)**

No token-budget type exists anywhere in Layer 1 or Layer 2 today. AG-17 must mint it. Its value stays Layer 3's (`0003:1610`).

**The whole decision is what the zero value means.** A plain `int64` field makes the unset budget `0`, and a budget of `0` means *every transcript overflows* — the most dangerous default a compaction seam could have. Layer 1 already answered this exact question: `ai.TokenCount` is `{count int64; present bool}` whose *"zero value is **absent**"*, and *"a count reported as nought is a different value"* (`ai/usage.go:34-47`).

**Recommendation — mirror it:**

```
type ContextBudget struct { limit int64; present bool }   // unexported fields
func ContextBudgetOf(n int64) ContextBudget
func (b ContextBudget) Limit() (int64, bool)
```

carried as `Harness.ContextBudget ContextBudget`, whose zero value means *"Layer 3 stated no budget"*.

| Option | Verdict |
|---|---|
| **(a)** Plain `int64` field | **Rejected.** Unset is indistinguishable from "budget zero", the worst possible conflation. |
| **(b)** `*int64` | **Rejected.** Two call sites can share one, and it is dereferenceable without meeting its absence — the exact defect `ai/usage.go:41-43` cites for choosing a value type. |
| **(c)** Presence-carrying value type mirroring `ai.TokenCount` | **RECOMMENDED.** House idiom, already proven, and the two-result accessor means the limit cannot be read without meeting its presence (`ai/usage.go:57-62`). |

**`sdd-design` MUST close**: the constructor's name; whether a negative `n` is rejected or clamped; and whether the budget is additionally reachable per-turn (recommendation: **no** — one harness-scoped field, since the model is fixed per harness in v1).

---

## Decision 3 — token accounting **(DECIDED — type-assert `ai.TokenCounter`; provenance is a type, not a comment)**

### 3a. No new interface

`ai.TokenCounter` (`ai/provider.go:130-134`) already ships, is conformance-tested (`agenttest/conformance_capabilities.go:152-180`), is proven discoverable from outside its package (`agenttest/provider_test.go:148-164`), and is inside Layer 2's allowed imports (`import_boundary_test.go:127-130`). Declaring a Layer 2 counting interface would violate `R-AMP-017` — *"one contract per capability, never an aggregate"* (`ai/provider.go:123-125`).

Discovery is `counter, ok := provider.(ai.TokenCounter)` and nothing else — no field, no flag, no registry (`ai/provider.go:106-113`).

### 3b. Three states, not two — and the distinction that is easiest to get wrong

`ai/provider.go:115-119` draws a line AG-17 **must** carry through:

- `ok == false` is a **clean absence** (`R-AMP-018`), and *"this package supplies no substitute, estimate or default"*. That sentence is the reason the estimate is Layer 2's job. **This is the only path that reaches the estimate.**
- A provider that satisfies the contract and then **declines to answer is non-conformant, not absent** (`R-AMP-019`) — *"advertising binds"*. **Falling back to the estimate here would launder a non-conformant provider into a working one**, and would make `R-AMP-019` unobservable from Layer 2 forever.

So the accounting has three outcomes, and they must be three type-level states.

### 3c. How the paths stay "distinguishable to the strategy consuming them"

AG-17.2's scenario wording (`0003:1645`) is a **type-level** obligation. A comment does not satisfy it; neither does a bare `int64` plus a sibling `bool` a consumer may ignore.

**Recommendation:**

```
type TokenSource int
const (
    TokenSourceUnavailable TokenSource = iota   // zero value
    TokenSourceReported
    TokenSourceEstimated
)

type TokenAccounting struct { tokens int64; source TokenSource }   // unexported fields
func (a TokenAccounting) Tokens() (int64, TokenSource)
```

carried to the strategy as `ContextPrompt.Accounting`.

Why this shape:

- **The number is unreadable without its provenance.** Same argument `ai/usage.go:57-62` gives for `TokenCount.Count() (int64, bool)`: *"it exists so that the count cannot be read without meeting its presence. A consumer that ignores the second result gets 0, which is the uninformative value rather than a plausible one."*
- **`Unavailable` is the zero value.** A zero `TokenAccounting{}` therefore reads as *"no figure"*, never as *"0 tokens"* — the same absence-is-not-zero discipline as Decision 2.
- **It does not reuse AG-16's presence bit.** `ai.TokenCount`'s bit means *"Layer 1 **reported** it"* (`ai/usage.go:34-55`). An estimate threaded through it would masquerade as exact and could corrupt `agent-cost-events`' cumulative semantics, which are defined over reported figures (`agent-run-driver/spec.md:344`). AG-17 introduces a **separate** provenance type and touches `cost_*` not at all.
- **It needs no new exported helper.** AG-17.2's scenario ("*when accounting runs against each*") is satisfied by running the harness with each provider and reading `ContextPrompt.Accounting` from a recording strategy. Zero extra exported surface.

**The counted request.** `ai.TokenCounter.CountTokens` takes an `ai.Request`, not a transcript. `buildLoopRequest(opts, system, transcript)` (`loop.go:713-726`) is package-private and every one of its three inputs is in scope at the turn boundary (`h.Turn`, `h.System`, `transcript`) — so the accounting counts **the same request shape the turn will send**, by reusing the same builder rather than a parallel one. See risk 5 for the pre-request-hook caveat.

**`sdd-design` MUST close**: whether the counter's error is additionally carried on the prompt or only collapsed into `TokenSourceUnavailable`; and whether a build failure in `buildLoopRequest` at the accounting point aborts the turn or yields `Unavailable` (recommendation: **yields `Unavailable`** — the turn's own build at `loop.go:304` remains the single authority on aborting, and duplicating that decision would create a second failure site for one condition).

---

## Decision 4 — the estimate's method **(DECIDED — UTF-8 bytes per token plus a per-message constant, stated as approximate)**

The charter's own words set the bar (`0003:1650`): *"character-count compaction is wrong by enough to matter."* A bare `len(s)/4` with no stated method fails that bar by being exactly the thing it names.

**Method (recommended, to be pinned normatively in the spec):**

> **Estimate** = ⌈ **B** / **D** ⌉ + **M** × **K**
> where **B** is the total UTF-8 **byte** length of the request's textual content — the system instruction (`ai/system_instruction.go:154`), every message (`ai/request.go:375`), and every tool schema (`ai/request.go:486`) — **K** is the message count, **D** is a stated bytes-per-token divisor, and **M** is a stated per-message structural constant.

Three properties make it defensible rather than arbitrary:

1. **Bytes, not characters or runes.** Modern tokenizers are byte-level BPE, so token count tracks *byte* count far more stably across scripts than rune count. A CJK ideograph is 3 UTF-8 bytes and roughly one token; counting runes under-counts it by ~4×, counting bytes by well under 2×. This is the single largest source of the error the charter warns about, and it is fixed by the choice of unit alone.
2. **A per-message constant.** Chat encodings add role and delimiter tokens per message that no character count ever sees. On short messages this term dominates — a 20-message transcript of one-line turns is where pure character counting is worst.
3. **Tool schemas are counted.** They are sent, they cost tokens, and they are the most commonly forgotten term.

**Accuracy caveat — stated, not implied.** The estimate is a heuristic over a proxy. It has **no proven bound in either direction**: it will over-count dense ASCII prose and under-count non-Latin scripts, base64-like content, and unusual tool schemas. The spec MUST state that it is an estimate, MUST state this method, and MUST forbid any code path from treating it as exact. Decision 3c makes that last clause **mechanically enforceable** rather than aspirational: a consumer physically cannot obtain the number without also obtaining `TokenSourceEstimated`.

**Purity.** The estimate MUST be a pure function of its `ai.Request`: no clock, no environment, no randomness, no I/O — so it is table-driven testable and identical across runs.

> **Correction to `explore.md` § 6**: `ambient_authority_test.go:73-94` forbids exactly four packages — `os`, `os/exec`, `syscall`, `io/ioutil`. It does **not** forbid `time`. Explore's "no clock, no env read" is right as an *obligation* but wrong as a *guard claim*: the env half is machine-checked, the clock half is not. The spec MUST therefore assert determinism directly (identical input ⇒ identical output, asserted in a test), rather than leaning on a guard that would not catch it.

**`sdd-design` MUST close**: the exact values of **D** and **M**, each with its stated rationale in the doc comment; and the exact accessor set walked to compute **B**, so the estimate cannot silently miss a request field a later Layer 1 version adds.

---

## Decision 5 — the `L2C-NN` doc-contract row **(RESOLVED — NO new row; `doc.go` and `doc_contract_guard_test.go` stay byte-unchanged)**

`explore.md` § 6 left this open. It is resolved here, with evidence, and is not passed downstream.

**The rule**, from the guard's own package comment (`doc_contract_guard_test.go:19-22`): *"A later milestone that appends a guarded paragraph to `doc.go`'s contract MUST add its row here, to `expectedLayer2ContractRows`, in the SAME pull request."* The trigger is **appending a paragraph to `doc.go`**, which happens when a milestone declares a **new package-wide upward guarantee**.

**The test**, stated verbatim by the run driver's own precedent (`agent-run-driver/spec.md:178`): *"This is `L2C-03` … at run scope, **not a new package-wide guarantee**; no `L2C-08` doc-contract row lands, and `doc.go` and `doc_contract_guard_test.go` stay byte-unchanged."*

**AG-17 fails that test in every direction:**

| Question | Answer |
|---|---|
| Does AG-17 add a new upward surface? | **No.** The two upward surfaces are the event stream (`L2C-03`) and history (`L2C-07`). AG-17 emits no event and adds no exported `History` method. |
| Does AG-17 change anything a caller observes by default? | **No.** The seam is nil-default and its shipped default is inert by *type* (Decision 1c). |
| Does AG-17 register a new `EventKind`? | **No.** `compaction_events.go` was minted by AG-06; AG-18 is its first caller. |
| Is there a precedent for exactly this shape? | **Yes, decisive.** **AG-15's failover seam is structurally identical** — a nil-default `Harness` field, a one-method interface, an inert shipped default. It added **no** `L2C` row: `agent-retry-failover/spec.md` contains **zero** occurrences of `L2C`, and its `NFR-RTY-004` (`:203`) instead requires *"every file named by `R-LSK-004` MUST be byte-unchanged"* — a list that includes `doc.go` and `doc_contract_guard_test.go` (`agent-loop-skeleton/spec.md:103`). AG-16 likewise added none (`agent-run-driver/spec.md:352`). |

The committed table therefore stays at its **eight** rows, `L2C-01`…`L2C-08` (`doc_contract_guard_test.go:62-71`), and AG-17 carries the same byte-unchanged substrate obligation AG-15 and AG-16 carried.

*(This is a scope-reducing resolution, so getting it wrong in the safe direction is cheap; getting it wrong in the unsafe direction — adding a row — would append an unamendable paragraph to a byte-pinned contract for a seam nobody can yet observe.)*

---

## Capabilities

> Contract with `sdd-spec`. Existing names taken verbatim from `openspec/specs/`.

### New

- **`agent-context-strategy`** — the turn-boundary strategy seam, its budget type, the never-compact default, and token accounting with type-level provenance. IDs `R-CTX-0NN` / `S-CTX-0NN`, bites `S-CTX-0NN`. Prefix `CTX` verified free against all 60 prefixes in use. Becomes `openspec/specs/agent-context-strategy/spec.md` at archive.

### Modified — deltas required

| Capability | What changes | Mandatory? |
|---|---|---|
| `agent-run-driver` | `:342`'s non-requirement row back-annotated **CLOSED by AG-17** for the *check* half, with AG-18 still owning compaction. The consultation site and its once-per-logical-turn cardinality recorded against `R-RUN-003`'s bracket/lane rule (unchanged — AG-17 emits nothing). `R-RUN-012`'s substrate posture restated: no new `EventKind`, no `ai/` edit. | **Yes — blocking.** |
| `agent-history` | `:260`'s row ("Context-window accounting over the transcript \| AG-17") back-annotated **CLOSED**, recording that history needed **no change**: the strategy reads through the existing `transcriptFromHistory(hist)` route (`harness.go:498`) and `history.go` is byte-unchanged. | **Yes — blocking.** |
| `agent-v1-scope` | Back-annotation of the `AGS-S` rows for **seam 5 → AG-17.1** and **seam 6 → AG-17.2** (`S-AGS-023`, `agent-v1-scope/spec.md:145`), following `R-AGS-014`'s standing amendment rules (`:232-234`) and AG-15's executed precedent at `:130`. `S-AGS-010` requires each `AGS-S` entry to name its trivial implementation as a concrete behavior — and cites *"that the shipped default never compacts"* as its own example, which Decision 1c now makes literally true at the type level. | **Yes.** Spec hygiene enforced by **no Go test** — the easiest delta in this change to skip. |
| `agent-retry-failover` | Back-annotation only: `R-RTY-002`'s byte-identical-transcript pin confirmed **held, not amended**, because Decision 1b places the seam outside the attempt loop. | Yes — annotation. |
| `agent-loop-skeleton` | `R-LSK-004` records that AG-17 requests **no** substrate release (`doc.go`, `doc_contract_guard_test.go`, `stream_check.go`, `event_descriptor.go`, `event_registry_test.go` all byte-unchanged, per Decision 5), and its filter-widening rule extends to AG-17's new filenames, byte-in-sync across both filters. | **Yes.** |

*(`agent-retry-failover` and `agent-loop-skeleton` are additions to `explore.md` § 3's list, found by reading `NFR-RTY-004` at `agent-retry-failover/spec.md:203` and `R-LSK-004`'s file list at `agent-loop-skeleton/spec.md:103`.)*

---

## Approach

Ordered so each step is independently green and the risky ones land last.

1. **The types first, wired to nothing.** `ContextBudget`, `TokenSource`, `TokenAccounting`, `ContextPrompt`, `ContextVerdict`, `ContextStrategy`, `NoOpContextStrategy`, and the compile-time guard `var _ ContextStrategy = NoOpContextStrategy{}`. Pure declarations, zero behavior change.
2. **The estimate, in isolation.** A pure function over `ai.Request`, table-driven across ASCII, CJK, empty transcript, tool-schema-bearing and system-instruction-bearing requests, plus a determinism assertion. This is the highest-value unit test in the change and it needs no harness.
3. **The accounting resolver.** Type-assert; on `ok` call `CountTokens` and label `Reported`; on `!ok` estimate and label `Estimated`; on an advertised counter's error label `Unavailable` and **never** estimate (Decision 3b). Table-driven against three fixture providers.
4. **The exported `agenttest` fixture**, embedding `*agenttest.Provider` and adding `CountTokens`, mirroring `stubProviderWithTokenCounter` (`agenttest/provider_test.go:133-142`). Note `agenttest.Provider`'s methods are on a **pointer** receiver (`fake_provider.go:74`), so the embedding must be `*Provider`.
5. **The seam consultation.** Insert between `harness.go:498` and `:530`, guarded by `h.ContextStrategy != nil`, with a comment stating Decision 1b's resolution in the same voice as the `R-RTY-002` block at `:518-529`.
6. **The pins.** Run identical scripts with the default strategy present and absent; assert **byte-identical event streams** and **byte-identical `hist.Entries()` read-backs** — the `S-PRH-002` / `S-LSK-015` / `R-RUN-012` nil-default precedent.

---

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/context_strategy.go` | **New** | The seam interface, `ContextPrompt`, `ContextVerdict`, `NoOpContextStrategy`, `ContextBudget` |
| `backend/agent/src/agent/token_accounting.go` | **New** | `TokenSource`, `TokenAccounting`, the resolver, the estimate |
| `backend/agent/src/agent/harness.go` | **Modified** | `ContextStrategy` and `ContextBudget` fields; consultation between `:498` and `:530` |
| `backend/agent/src/agenttest/` (new file) | **New** | Exported counting-capable fake embedding `*Provider` |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | AG-17.1's and AG-17.2's 4 scenarios, the estimate table, the inertness pins, bites |
| `loop_test.go`, `loop_hook_test.go` | **Modified** | Substrate filter widening, byte-in-sync, exact filename suffixes |
| `openspec/specs/{agent-run-driver, agent-history, agent-v1-scope, agent-retry-failover, agent-loop-skeleton}` | **Delta** | Five deltas — three normative, two back-annotation |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-17 checklist tick, R-18 seams 5/6 back-annotation, counter to 17/24 |
| `doc.go`, `doc_contract_guard_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `compaction_events.go`, `cost_events.go`, `history.go`, `go.mod`, `go.sum`, **all of `backend/agent/src/ai/**`** | **NOT TOUCHED** | Decision 5; no new event kind; no Layer 1 edit; no new dependency |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | A downstream phase reads only the Gherkin and rebuilds the **per-attempt** seam | **High** | The single most likely defect in this change. Decision 1b is normative spec text, not a proposal note. A **bite** must move the consultation inside the attempt loop and prove the once-per-logical-turn scenario FAILS under a scripted retry — otherwise the requirement is untested against the reading it exists to exclude. |
| 2 | The estimate is used as a fallback when an **advertised** counter errors, laundering an `R-AMP-019` non-conformant provider | **Med-High** | Three type-level states (Decision 3c), `Unavailable` as the zero value, and a scenario scripting a counter that returns an error, asserting the source is `Unavailable` and **not** `Estimated`. Bite: collapse to two states → that scenario fails. |
| 3 | The estimate is threaded through `ai.TokenCount`'s presence bit and masquerades as reported | Med | A separate provenance type that shares no field with `ai.Usage`. `cost_events.go` is **byte-unchanged** and `agent-cost-events`' cumulative rule (`agent-run-driver/spec.md:344`) is untouched — verified by `sdd-verify` reading the shipped file, not by assertion. |
| 4 | "Distinguishable to the strategy" ships as a doc comment or an ignorable sibling `bool` | Med | The two-result accessor makes the count unobtainable without its source (`ai/usage.go:57-62`'s own argument). Bite: expose a bare `Tokens() int64` → the distinguishability scenario fails. |
| 5 | The counted request differs from the sent request, because AG-08's pre-request hook derives a new `ai.Request` **after** the accounting point (`loop.go:326`, after `buildLoopRequest` at `:304`) | **Med — newly identified, not in `explore.md`** | Real and unavoidable at this seam: the hook runs inside `Turn`, downstream of the turn boundary. The spec MUST state that accounting is over the **pre-hook** request, and MUST NOT claim exactness even on the `Reported` path. A test with a request-mutating hook installed must record the divergence rather than assert equality. |
| 6 | `agent-v1-scope`'s seam-5/seam-6 back-annotation is skipped | **Med-High** | Enforced by **no Go test** — this repo's known "un-back-annotated merge" staleness shape. `R-AGS-014` (`:232-234`) sets append-only, no-renumbering, struck-through-supersession rules; AG-15's executed annotation at `:130` is the template. `sdd-verify` MUST open `S-AGS-023` (`:145`) and confirm both rows carry it. |
| 7 | The nil-vs-default inertness pin passes **vacuously** because the recorded streams are compared loosely | Med | Assert **byte-identical** event streams *and* byte-identical `hist.Entries()` read-backs across the present/absent pair, per `S-PRH-002` / `S-LSK-015`. A bite that makes the default emit one event must break it. |
| 8 | An `L2C-09` row is added on the assumption that a new seam is a new guarantee | Low (now) | Decision 5 resolves it with the AG-15 precedent. `doc.go` and `doc_contract_guard_test.go` are named byte-unchanged in `NFR`, so the guard's own count-equality check catches a stray row. |
| 9 | The estimate quietly reads a clock or drifts, and `ambient_authority_test.go` does not catch it | Low-Med | The guard forbids only `os`, `os/exec`, `syscall`, `io/ioutil` (`:73-94`) — **not** `time`. Determinism is therefore asserted directly by test, per Decision 4's correction. |
| 10 | The `agenttest` fixture embeds `Provider` by value and fails to satisfy `ai.ModelProvider` | Low | `Stream` is on `*Provider` (`fake_provider.go:74`). Approach step 4 names the pointer embedding; a `var _ ai.ModelProvider` / `var _ ai.TokenCounter` guard pair catches it at compile time. |
| 11 | Five spec deltas; one gets missed | Med | Enumerated with file and line in Capabilities. Two of the five were **not** in `explore.md` and were found by reading. `sdd-verify` re-reads each cited line against the shipped change. |
| 12 | Review budget exceeds the raised 1000-line bar | **High** | `size:exception` pre-accepted. Slicing plan follows the numbered Approach steps, already ordered by dependency. |

---

## Rollback Plan

Single revert of the AG-17 merge commit.

`context_strategy.go`, `token_accounting.go`, the new `agenttest` fixture and all new test files are deleted; `harness.go` returns to going straight from `transcript` (`:498`) into the attempt loop (`:530`) with no `ContextStrategy` or `ContextBudget` field; both substrate filters return to their pre-AG-17 filename lists; the five spec deltas are dropped; doc 0003's AG-17 line un-ticks, seams 5 and 6 lose their back-annotation, and the counter returns to 16/24.

**The revert is unusually clean, by design.** Nothing persists, no data migrates, no `go.mod`/`go.sum` change, nothing outside `backend/agent`, no Layer 1 file touched, no `EventKind` added or removed, and `doc.go`/`doc_contract_guard_test.go`/`stream_check.go`/`history.go`/`cost_events.go` were never modified. Because the shipped default is inert **by type** (Decision 1c), no observable behavior existed to lose: an event stream recorded before the change and one recorded after are byte-identical, so a revert cannot change any caller's observations either.

The only externally visible removal is the new exported surface — `ContextStrategy`, `ContextBudget`, `TokenAccounting`, `TokenSource` and the `agenttest` fixture. Layer 3 does not exist yet (`0003:110`), so no live consumer is orphaned.

**Forward-looking cost**: reverting re-opens `agent-run-driver:342`'s check half and `agent-history:260`, re-opens R-18 seams 5 and 6, and **blocks AG-18 entirely** — AG-18 depends on AG-17 by the charter (`0003:1664`) and has no seam to implement without it. Scheduling consequence, not correctness.

---

## Review-workload forecast

| Component | Estimate (authored, additions + deletions) |
|---|---|
| `context_strategy.go` — interface, prompt, verdict, budget, no-op, guards | 130–200 |
| `token_accounting.go` — provenance type, resolver, estimate | 120–190 |
| `harness.go` — two fields, the consultation, its comment | 40–70 |
| `agenttest` fixture | 30–60 |
| Test files — 4 charter scenarios + estimate table + inertness pins + bites | 450–700 |
| Filter widening | 20–40 |
| Doc 0003 tick + seam back-annotation + traceability | 20–35 |
| **Go subtotal** | **810–1295** |
| SDD markdown — proposal, spec, **5 deltas**, design, tasks, apply-progress, verify-report | **750–1150** |
| **Total** | **1560–2445** |

`Decision needed before apply: No` — `size:exception` pre-accepted at 1000 review lines, recorded here, one PR.
`Chained PRs recommended: No` — but if `sdd-tasks` forecasts above ~2800, slice along the Approach steps: **U1** (types + estimate + accounting resolver + `agenttest` fixture — AG-17.2 complete, wired to nothing, zero behavior change) → **U2** (the seam consultation + AG-17.1's two scenarios). U1 is independently deliverable and changes no observable behavior at all.
`400-line budget risk: High`

The SDD markdown counts toward the attempt budget. `sdd-tasks` MUST forecast against the **full** diff, not the Go diff.

---

## Dependencies

- **AG-13** (archived) — `Harness`, `Run`'s outer per-logical-turn loop (`harness.go:479-670`), `transcriptFromHistory` (`:498`).
- **AG-15** (merged) — the attempt loop (`harness.go:530-608`), `R-RTY-002`'s byte-identical-transcript pin (`:513-529`), and `FailoverPolicy`'s seam convention (`failover_policy.go:19-79`) — the shape this change mirrors.
- **AG-12** (archived) — `History`, `Entries()`, and the read-only route the strategy uses.
- **AG-02** (archived) — `agent-v1-scope`'s verdict list and `R-AGS-014`'s amendment rules.
- **AG-08** (archived) — `TurnOptions.PreRequestHook` (`loop.go:326`), relevant to risk 5.
- **Layer 1 (AI-20.5)** — `ai.TokenCounter` (`ai/provider.go:102-134`), `ai.TokenCount` and its presence bit (`ai/usage.go:34-77`), `ai.Request`'s accessors (`request.go:375`, `:486`; `system_instruction.go:154`).
- **`agenttest`** — `Provider` (`fake_provider.go:55-117`, pointer receivers), `Script`, `Requests()`; `stubProviderWithTokenCounter` (`provider_test.go:133-142`) as the embedding template.
- **doc 0003:1600-1653** — the AG-17 charter and its two Gherkin leaves; **doc 0003:1655-1665** — AG-18's charter, which consumes this seam.

---

## Success Criteria — restated as verifiable checks

- [ ] `cd backend/agent && make test` green under `-race` with `-count=1` (never a cached run); all four AG-17 scenarios closed with recorded evidence
- [ ] **AG-17.1 / cardinality** — a run of N logical turns consults the strategy **exactly N times**, and a run in which one logical turn is **retried** still consults it exactly N times, not N+retries (Decision 1b, risk 1)
- [ ] **AG-17.1 / inputs** — every consultation carries the current transcript **and** the budget; the transcript the strategy sees is the same slice the turn will send
- [ ] **AG-17.1 / inertness** — a run with the default strategy and a run with a nil field produce **byte-identical** event streams and **byte-identical** `hist.Entries()` read-backs; **no compaction event** appears on either
- [ ] **AG-17.1 / unconstructibility** — `ContextVerdict` exposes no field by which any implementation could request compaction, verified by reading the shipped type
- [ ] **AG-17.2 / discovery** — with the counting-capable fake the accounting reports `TokenSourceReported` and the provider's own figure; with `agenttest.Provider` it reports `TokenSourceEstimated`; the two are distinguishable **through the type**, not a comment
- [ ] **AG-17.2 / non-conformance** — a provider that advertises `ai.TokenCounter` and returns an error yields `TokenSourceUnavailable`, **never** `TokenSourceEstimated` (`R-AMP-019`, risk 2)
- [ ] **AG-17.2 / no masquerade** — the estimate is unreachable without its `TokenSourceEstimated` label; no path converts it into an `ai.TokenCount`; `cost_events.go` is **byte-unchanged**
- [ ] **AG-17.2 / method stated** — the estimate's doc comment states the formula, the unit (UTF-8 bytes), both constants with their rationale, and an explicit accuracy caveat naming its unbounded error
- [ ] **AG-17.2 / determinism** — the estimate is a pure function: identical `ai.Request` ⇒ identical figure, asserted directly (the ambient guard does not cover the clock, risk 9)
- [ ] **Bites, RED-recorded before GREEN**: (a) move the consultation into the attempt loop → the cardinality scenario fails under a scripted retry; (b) collapse `TokenSource` to two states → the non-conformance scenario fails; (c) expose a bare `Tokens() int64` → the distinguishability scenario fails; (d) make the default emit one event → the inertness pin fails
- [ ] **No new `EventKind`** — `event_descriptor.go` and `event_registry_test.go` untouched; the every-kind-constructible guard passes at its committed count
- [ ] **`doc.go` and `doc_contract_guard_test.go` byte-unchanged**; `expectedLayer2ContractRows` still carries exactly **eight** rows (Decision 5)
- [ ] `stream_check.go`, `history.go`, `compaction_events.go` byte-unchanged; zero files under `backend/agent/src/ai/` differ; `go.mod`/`go.sum` unchanged
- [ ] Import and ambient-authority guards pass with **zero** changes
- [ ] Both substrate filters carry an identical exact-filename entry set, one entry per file AG-17 introduces, no wildcard or prefix pattern
- [ ] All five spec deltas written; each cited line re-read against the shipped change by `sdd-verify`
- [ ] **`agent-v1-scope` seams 5 and 6 back-annotated** per `S-AGS-023` and `R-AGS-014`, following AG-15's template at `:130` (risk 6 — no Go test enforces this)
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean — `vuln-check` is **not** in `make all`
- [ ] doc 0003's AG-17 checklist ticked, R-18 seams 5/6 back-annotated, milestone counter bumped to 17/24
