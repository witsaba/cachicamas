# Spec — The context strategy seam and token accounting (`agent-context-strategy`)

> **Change**: `cachicamas-agent-context-strategy` · **AG-17** (Layer 2, Wave 4), `0003:1600-1653`
> **NEW capability**, minted by this change (AG-17). Promoted to `openspec/specs/agent-context-strategy/spec.md` at archive.
> **Nodes**: AG-17.1 `[leaf]` (the strategy seam) · AG-17.2 `[leaf]` (token accounting)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test` with `-count=1`.
> **IDs**: `R-CTX-0NN` / `S-CTX-0NN`, **append-only**. Allocated `R-CTX-001`…`R-CTX-012` and `S-CTX-001`…`S-CTX-021`, of which `S-CTX-003`, `S-CTX-008`, `S-CTX-013` and `S-CTX-015` are the four **bites**. Prefix `CTX` verified free against the 60 prefixes in use under `openspec/specs/`. **No total is stated** — a count goes silently false the moment a later milestone appends (`S-LSK-020`).
> **Sources**: charter `0003:1600-1653`; the `cachicamas-agent-context-strategy` change's proposal (Decisions 1–5, binding), design (DD1–DD7, committed at `2ed400f6`, binding) and explore notes, archived at `openspec/changes/archive/2026-08-19-cachicamas-agent-context-strategy/`.
> **Ownership boundary**: this capability owns the seam's placement and cardinality, its prompt and verdict types, the budget type, the never-compact default, and token accounting with its provenance and its estimate. It does not own the run algorithm (`agent-run-driver`), the retry gates (`agent-retry-failover`), `Turn`'s per-path emission contract (`agent-loop-skeleton`), the transcript (`agent-history`), or **compaction** (AG-18).
> **Every `file:line` below was opened in this worktree during this phase, against `origin/main@b0de5bf6`.**

## Purpose

The harness runs turn after turn over a transcript that only grows. Nothing measures it, and nothing is asked whether it should shrink. AG-17 ships **a place to stand** and **a measurement**, and deliberately ships **no compaction**.

Two promoted specs already record this as debt owed by AG-17 by name: `agent-run-driver/spec.md:342` ("**AG-17** inserts it at AG-13's turn boundary; **AG-18** implements compaction") and `agent-history/spec.md:260` ("Context-window accounting over the transcript | AG-17"). The first row is not merely a debt marker — it is **the placement decision, already made by a promoted spec**.

## Coverage — the four charter scenarios, each mapped

| # | Charter scenario | Lines | Owning requirement | Scenario(s) |
|---|---|---|---|---|
| 1 | AG-17.1 "the strategy is consulted before every call" | `0003:1624-1627` | `R-CTX-001` (cardinality), `R-CTX-002` (inputs) | `S-CTX-001`, `S-CTX-002`, `S-CTX-004`, bite `S-CTX-003` |
| 2 | AG-17.1 "the never-compact default changes nothing (pin)" | `0003:1629-1632` | `R-CTX-003` (unconstructibility), `R-CTX-004` (inertness) | `S-CTX-005`, `S-CTX-006`, `S-CTX-007`, bite `S-CTX-008` |
| 3 | AG-17.2 "counting capability is discovered and used" | `0003:1642-1645` | `R-CTX-006` (discovery), `R-CTX-007` (three states), `R-CTX-008` (type-level distinguishability) | `S-CTX-011`, `S-CTX-012`, `S-CTX-014`, bites `S-CTX-013`, `S-CTX-015` |
| 4 | AG-17.2 "an estimate never masquerades as exact" | `0003:1647-1650` | `R-CTX-009` (method stated), `R-CTX-010` (determinism), `R-CTX-011` (pre-hook inexactness) | `S-CTX-016`, `S-CTX-017`, `S-CTX-018`, `S-CTX-019` |

Cross-cut requirements carrying no charter leaf of their own: `R-CTX-005` (the budget type, Decision 2), `R-CTX-012` (substrate and closed-sequence safety).

---

## Requirements

### R-CTX-001 — The seam is consulted exactly once per LOGICAL turn, never once per attempt

The harness MUST expose a **nil-default** context-strategy seam as a field on the caller-owned harness value, **not** on the one-turn options value. The rule is the one `failover_policy.go:19-26` already states verbatim — *"a seam belongs where its consumer is"* — and a turn boundary is a run-driver concept by the identical argument: the one-turn surface has no notion of a logical turn, because it **is** one attempt.

The seam MUST be consulted at AG-13's **turn boundary** — inside `Harness.Run`'s outer per-logical-turn loop, after the transcript for the coming turn is resolved (`harness.go:512`) and **before** the attempt loop is entered (`harness.go:562`). With the field nil, the seam MUST NOT be consulted, no accounting MUST be resolved, and the path MUST be the pre-AG-17 path.

**The count's unit is a LOGICAL TURN, and this requirement states the unit rather than leaving it to be inferred.** For a run of `N` logical turns the seam MUST be consulted **exactly `N` times**, *regardless of how many provider calls those turns issue*. A logical turn that is retried `k` times issues `k+1` `provider.Stream` calls (`loop.go:351`) inside **one** outer-loop iteration and MUST still produce **exactly one** consultation.

The charter's own wording (`0003:1627`) — *"consulted exactly N times, before each provider call"* — contains two clauses that genuinely diverge under AG-15's retries. **They are resolved here normatively, in favour of the logical turn**, for three reasons in descending strength:

1. **A promoted spec already says so.** `agent-run-driver/spec.md:342` places the check "at AG-13's turn boundary". The turn boundary is the outer loop (`harness.go:493`), not the attempt loop (`harness.go:562`).
2. **The per-attempt reading would make `R-RTY-002` unprovable.** `harness.go:550-561` documents that pin precisely: the attempt loop re-invokes the one-turn surface *"over the transcript slice built once above and reused **BY REFERENCE** across attempts — this, not an assumption, is what makes 'identical transcript' provable"*. A seam consulted **inside** the attempt loop is a seam that, once AG-18 gives it compaction teeth, could mutate the transcript between two attempts of one logical turn. That does not merely risk breaking `agent-retry-failover`'s `R-RTY-002` (`spec.md:68-76`) — it defeats the exact argument its comment relies on. Turn-boundary placement keeps that pin true **by construction** rather than by convention.
3. **The sibling seam resolves identically.** Retry timing resolves once per logical turn at `harness.go:537-543`, and its own comment gives the same structural reason.

A consultation MUST NOT be emitted as an event, MUST NOT write the transcript, and MUST NOT alter the turn's outcome, returned values or error.

#### Scenarios

- **S-CTX-001** — **Charter AG-17.1 scenario 1, clean half.** Given a recording strategy installed on the harness and a script driving a run of **two** logical turns, each completing on its first attempt, when the run completes, then the recorder holds **exactly two** consultations, each one taken before that turn's first provider call as proved by interleaving the recorder's entries against the provider's recorded requests; and the run's outcome, returned error and event stream are those the same script produces with no strategy installed.
- **S-CTX-002** — **Charter AG-17.1 scenario 1, the retry half — the reading this requirement exists to exclude.** Given a recording strategy, a provider wrapper whose **first** logical turn fails retryably with no partial output on its first attempt and then completes on its second, and a second logical turn that completes on its first attempt — **three provider attempts across two logical turns** — when the run completes, then the provider recorded **three** requests, and the recorder holds **exactly two** consultations, not three; and the two transcripts the strategy received are the two distinct per-logical-turn transcripts, never the same transcript twice.
- **S-CTX-003** — **(bite)** RED-first, **mandated**. Given a scratch tree in which the consultation is moved from the turn boundary into the attempt loop (`harness.go:562`), when `S-CTX-002` runs, then it FAILS reporting **three** consultations against an expectation of two — proving the cardinality assertion is load-bearing against the per-attempt reading rather than incidentally satisfied by a run that never retries. RED-recorded BEFORE `S-CTX-002` is GREEN, then reverted.

---

### R-CTX-002 — Every consultation carries both inputs: the current transcript and the budget

Each consultation MUST pass a typed prompt value carrying, at minimum: the **transcript** of the coming logical turn, the **budget** (`R-CTX-005`), and the **accounting** (`R-CTX-007`). The strategy MUST NOT have to reach for any of them through another route, and MUST NOT be able to reach the harness's own state.

The transcript the strategy receives MUST correspond to the messages the coming logical turn will send, and MUST be a **fresh clone** rather than the harness's own slice — `request.go:367-370`'s own argument: a consumer that rewrites what it received must not be able to rewrite the harness's slice. A strategy that mutates the slice it received MUST NOT change what the turn sends, which is what keeps `R-RTY-002`'s by-reference reuse safe in the presence of a third-party strategy.

#### Scenarios

- **S-CTX-004** — **Charter AG-17.1 scenario 1, the inputs half.** Given a recording strategy and a harness carrying a **stated** budget, when a two-logical-turn run completes, then each recorded prompt's transcript is element-equal to the messages of the request that turn's provider call actually carried (read back from the provider's recorded requests), each recorded prompt's budget reports the stated limit together with a present reading, and each recorded prompt carries an accounting value; and given a strategy that **appends to and overwrites** the slice it received, when the same run is driven, then the requests the provider recorded are byte-identical to those of the non-mutating run — the clone, not a convention, is what makes that true.

---

### R-CTX-003 — "Never compacts" is a property of the TYPE, not of the shipped implementation

The seam's verdict type MUST expose **no field, no method and no constructor** by which any implementation could request compaction. Its zero value MUST be its only constructible value.

This is deliberately stronger than "the harness ignores a compaction request". An ignored accept-flag would make the never-compacts guarantee a property of the one implementation this milestone ships; an empty verdict makes it a property of the type, **unfalsifiable by any strategy a caller could write**. `failover_policy.go:57-62` blesses exactly this extension path for the later version that gives it teeth: *"A later version adds route and re-budget fields non-breakingly; every existing implementation returning the zero verdict keeps compiling."* **AG-18 is that later version**, and until it lands, compaction is unconstructible rather than merely unimplemented.

The harness MUST receive the verdict and discard it, because nothing else exists to do with it. AG-17 MUST introduce **no** compaction branch, **no** compaction event emission, and **no** transcript surgery of any kind.

One shipped, installable never-compact default MUST exist, together with a compile-time guard that it satisfies the seam.

#### Scenarios

- **S-CTX-005** — **Charter AG-17.1 scenario 2, the type half.** Given the shipped verdict type, when it is inspected reflectively from an external test package, then it reports **zero** fields; and when the package's exported surface is inspected, then the verdict type carries no exported method and no constructor function that could produce a value distinguishable from its zero value. This scenario is what guards AG-18's extension point: it must be **deliberately** amended by AG-18, never silently outgrown.
- **S-CTX-006** — Given the shipped never-compact default, when it is installed on a harness and a run is driven, then the compile-time guard binds it to the seam, every consultation returns the zero verdict, and **no** compaction-family event appears anywhere on the recorded stream — asserted by scanning the recorded kinds, not by trusting that none is constructed.

---

### R-CTX-004 — The inertness pin: installing the default changes nothing observable

A run driven with the shipped never-compact default installed MUST be observationally indistinguishable from the identical run driven with the seam field nil.

**Indistinguishable is defined by two byte-level comparisons, both required**, because a loose comparison passes vacuously (`S-PRH-002`'s "the seam adds zero observable behavior when not installed" at `agent-pre-request-hook/spec.md:63`, `S-LSK-015` and `R-RUN-012`'s nil-default posture at `agent-run-driver/spec.md:295-302`):

1. the recorded **event streams** of the two runs MUST be byte-identical, and
2. the **transcript read back** through history's existing read-only route MUST be byte-identical between the two runs.

**No compaction event MUST be emitted on either run, and no history entry MUST be mutated, added or removed by the seam on either run.** The seam reads the transcript through the route the driver already uses (`transcriptFromHistory(hist)`, `harness.go:512`); it opens no new history route and takes no write path.

#### Scenarios

- **S-CTX-007** — **Charter AG-17.1 scenario 2 (pin).** Given two runs of the identical multi-turn script against the identical non-counting fake provider — run A with the seam field nil, run B with the shipped never-compact default installed — when both complete and both streams are recorded, then the two recorded event streams are **byte-identical**, the two history read-backs are **byte-identical**, neither stream carries any compaction-family kind, and both runs' returned values and outcomes are equal.
- **S-CTX-008** — **(bite)** RED-first. Given a scratch tree in which the consultation block emits one compaction-started event, when `S-CTX-007` runs, then it FAILS on the byte-identity comparison of the two event streams — proving the pin is a real comparison rather than a loose one that a stray emission would survive. RED-recorded BEFORE `S-CTX-007` is GREEN, then reverted.

---

### R-CTX-005 — The budget is a presence-carrying value; its zero value is ABSENT, never "zero tokens"

Layer 2 MUST define the budget's **type**; its **value** stays Layer 3's (`0003:1610`; `R-AGS-015`, `agent-v1-scope/spec.md:245`). No token-budget type exists anywhere in Layer 1 or Layer 2 today, so AG-17 mints it.

The budget MUST be a **value type carrying its own presence bit**, mirroring `ai.TokenCount` (`ai/usage.go:34-47`) whose *"zero value is **absent**"* and for which *"a count reported as nought is a different value"*. Concretely:

- its **zero value** MUST read as *"Layer 3 stated no budget"* and MUST NOT read as a budget of zero tokens — a budget of zero means *every transcript overflows*, the most dangerous default a compaction seam could carry;
- a **stated zero** MUST be constructible and MUST be a different value from absence;
- its limit MUST be reachable only through a **two-result** accessor returning the limit together with its presence, so the limit cannot be read without meeting its presence (`ai/usage.go:57-62`);
- its constructor MUST be **total** — a negative input MUST yield the **absent** zero value rather than an error, a panic, or a present clamp to zero. Minting a negative or clamped-zero limit as *present* would hand AG-18 a bound under which every transcript overflows, which is the exact defect the presence bit exists to prevent;
- it MUST be carried as **one harness-scoped field**, not per-turn: the model is fixed per harness in v1, and the one-turn options value is the wrong home by `R-CTX-001`'s own argument.

#### Scenarios

- **S-CTX-009** — Given the budget type, when its zero value is read through the two-result accessor, then it reports absence; and given a budget constructed from a stated zero, when it is read, then it reports a limit of zero **together with presence**, and the two values are not equal — absence and a stated nought are distinguishable, exactly as `ai.Tokens(0)` is distinguishable from `ai.TokenCount{}`.
- **S-CTX-010** — Given a budget constructed from a negative input, when it is read, then it reports **absence**, the construction returns no error and panics not at all, and the resulting value is equal to the zero value; and given a harness with no budget field set, when a recording strategy is consulted, then the prompt's budget reports absence rather than a limit of zero.

---

### R-CTX-006 — The counting capability is discovered by type assertion on the SHIPPED Layer 1 contract; Layer 2 declares no counting interface of its own

Accounting MUST discover the provider's optional counting capability by **type-asserting the shipped `ai.TokenCounter`** (`ai/provider.go:130-134`) on the provider value, and by no other means — no field, no flag, no registration call, no catalog entry (`ai/provider.go:106-113`). The assertion's success MUST be what "the capability is present" means.

Layer 2 MUST NOT declare a counting interface of its own. `ai.TokenCounter` already ships, is conformance-tested (`agenttest/conformance_capabilities.go:152-180`), is proven discoverable from outside its package (`agenttest/provider_test.go:148-164`), and is inside Layer 2's allowed imports (`import_boundary_test.go:127-130`). A second Layer 2 contract for one capability would violate `R-AMP-017` — *"one contract per capability, never an aggregate"* (`ai/provider.go:123-125`).

The request the accounting counts MUST be built by the **same builder the turn itself uses** (`buildLoopRequest`, `loop.go:713-726`), from the same three inputs in scope at the turn boundary, so the counted request is the same request shape the turn will send — subject to `R-CTX-011`. A failure to build that request MUST yield the *unavailable* state rather than aborting the turn: the turn's own build (`loop.go:304`) remains the single authority on aborting, and duplicating that decision would create a second failure site for one condition.

The counting-capable fixture the scenarios need MUST be an **exported** `agenttest` fixture embedding the existing exported fake by **pointer** (`Stream` is on the pointer receiver, `fake_provider.go:74`), mirroring the internal embedding shape at `agenttest/provider_test.go:133-142`. No scenario may be written as if the existing non-counting fake had been widened; it is the "without the capability" fixture and MUST stay so.

#### Scenarios

- **S-CTX-011** — **Charter AG-17.2 scenario 1, the discovery half.** Given a counting-capable exported fake provider scripted to report a specific figure, when a run is driven with a recording strategy installed, then the prompt's accounting reports **that provider's own figure** with the *reported* provenance, and the fixture recorded a counting call whose request was built by the turn's own builder from the turn's own system instruction, options and transcript; and given the existing **non-counting** exported fake and the identical script, when the same run is driven, then no counting call is made at all and the accounting reports the *estimated* provenance. Additionally: given the package's declared interfaces, when they are enumerated, then Layer 2 declares **no** token-counting interface of its own.

---

### R-CTX-007 — Three provenance states, and the estimate path is for genuine ABSENCE ONLY

Accounting MUST resolve to exactly one of **three** states, and the third exists because two would be wrong:

| State | Reached when | Figure |
|---|---|---|
| **reported** | the provider satisfies `ai.TokenCounter` **and** answers with a present count and no error | the provider's own figure |
| **estimated** | the type assertion **fails** — the capability is cleanly absent (`R-AMP-018`) | Layer 2's documented estimate (`R-CTX-009`) |
| **unavailable** | anything else, and it is the **zero value** | none |

**The estimate path MUST be reachable ONLY from a failed type assertion.** `ai/provider.go:115-119` draws the line this requirement carries through: `ok == false` is a clean absence, and *"this package supplies no substitute, estimate or default"* — which is precisely why the estimate is Layer 2's job. But *"a provider that satisfies this contract and then declines to answer is non-conformant, not absent (`R-AMP-019`) — advertising binds."*

**Two distinct shapes both resolve to *unavailable*, and naming both is the point** — a resolver that handles only the first launders the second:

1. an **advertised** counter that returns a **non-nil error**; and
2. an **advertised** counter that returns a **nil error together with an absent count** — "answered no figure" is not a report.

Falling back to the estimate on either shape would launder a non-conformant provider into a working one, and would make `R-AMP-019` unobservable from Layer 2 forever. The resolver MUST therefore label both *unavailable* and MUST NOT produce an estimate on either path. A build failure at the accounting point (`R-CTX-006`) likewise yields *unavailable*.

The counter's error MUST NOT be carried on to the strategy: the strategy is a measurement consumer, not an error handler, and `R-AMP-019` non-conformance stays observable through the *unavailable* label these scenarios assert. AG-18 MAY widen the prompt non-breakingly if diagnosis is ever needed.

#### Scenarios

- **S-CTX-012** — **Charter AG-17.2 scenario 1, the three-state table.** Given a directly-exercisable accounting resolution over four fixtures — (a) a counting fixture reporting a present figure, (b) the non-counting fake, (c) an **advertised** counting fixture returning a non-nil error, (d) an **advertised** counting fixture returning a **nil error with an absent count** — when each is resolved, then (a) yields *reported* with that figure, (b) yields *estimated* with the estimate's figure, and **both (c) and (d) yield *unavailable* with no figure and, specifically, NOT *estimated***; and the value produced for (c) and (d) is equal to the zero accounting value, so a zero value reads as "no figure" and never as "0 tokens".
- **S-CTX-013** — **(bite)** RED-first. Given a scratch tree in which the three states are collapsed to two — an advertised counter's error falling through to the estimate — when `S-CTX-012` runs, then rows (c) and (d) FAIL reporting *estimated* against an expectation of *unavailable*, proving the non-conformance distinction is enforced rather than described. RED-recorded BEFORE `S-CTX-012` is GREEN, then reverted.

---

### R-CTX-008 — The three states are distinguishable to the consuming strategy at the TYPE level

The charter's *"the two paths distinguishable to the strategy consuming them"* (`0003:1645`) is a **type-level** obligation. A doc comment does not satisfy it; neither does a bare integer beside a sibling boolean a consumer may ignore.

The accounting value MUST carry its figure in an **unexported** field and MUST expose it only through a **two-result** accessor returning the figure together with its provenance, so a consumer **physically cannot obtain the number without also obtaining the source**. This is `ai/usage.go:57-62`'s own argument, applied one level up: *"it exists so that the count cannot be read without meeting its presence. A consumer that ignores the second result gets 0, which is the uninformative value rather than a plausible one."*

The provenance MUST be a **distinct type** whose zero value is *unavailable*, so a zero accounting value reads as "no figure". Its rendering MUST distinguish the three states, for the log-line reason `ai/usage.go:72-77` gives.

**The provenance MUST NOT reuse `ai.TokenCount`'s presence bit.** That bit means *"Layer 1 **reported** it"* (`ai/usage.go:34-55`). An estimate threaded through it would masquerade as exact and could corrupt `agent-cost-events`' cumulative semantics, which are defined over reported figures (`agent-run-driver/spec.md:344`). AG-17 introduces a **separate** provenance type sharing no field with the usage record.

#### Scenarios

- **S-CTX-014** — **Charter AG-17.2 scenario 1, the distinguishability half.** Given a strategy consuming the prompt's accounting from an external test package, when it reads the figure, then the only route to it returns the provenance in the same call; the accounting type exposes **no** exported field and **no** single-result figure accessor; the three provenance values render distinctly; and the zero accounting value reports *unavailable* with a figure of zero, which is the uninformative value rather than a plausible one.
- **S-CTX-015** — **(bite)** RED-first. Given a scratch tree in which the accounting additionally exposes a bare single-result figure accessor, when `S-CTX-014` runs, then it FAILS at the "no single-result figure accessor" assertion — proving the distinguishability is mechanical rather than conventional. RED-recorded BEFORE `S-CTX-014` is GREEN, then reverted. *(Note for the implementer: if the bite is taken as a **removal** of the two-result accessor rather than an addition beside it, the compile failure is itself the RED evidence.)*

---

### R-CTX-009 — The estimate states its method, and no code path treats it as exact

The charter sets the bar in its own words (`0003:1650`): *"character-count compaction is wrong by enough to matter."* A bare divide-by-four with no stated method fails that bar by being exactly the thing it names.

The estimate MUST be computed as:

> **Estimate** = ⌈ **B** / **D** ⌉ + **M** × **K**

where **B** is the total **UTF-8 byte** length of the request's textual content, **K** is the request's message count, **D** is a stated bytes-per-token divisor, and **M** is a stated per-message structural constant. Ceiling division MUST be integer arithmetic, so the estimate needs no floating point.

**B MUST be computed over an enumerated accessor walk, stated normatively so a later Layer 1 field cannot be missed silently.** It MUST include, at minimum: the system instruction's segment text (`ai/system_instruction.go:154`, `:131`, `:48`); every message's content parts (`ai/request.go:375`), covering text, tool-call name and arguments, tool-result content, and reasoning text; and **every tool schema** (`ai/request.go:486`) — name, description and schema — because they are sent, they cost tokens, and they are the most commonly forgotten term.

Three properties MUST be stated in the estimate's own documentation as its rationale, not merely implied:

1. **Bytes, not characters or runes.** Modern tokenizers are byte-level, so token count tracks byte count far more stably across scripts. A CJK ideograph is 3 UTF-8 bytes and roughly one token: counting runes under-counts it by roughly fourfold, counting bytes by well under twofold. This is the single largest source of the error the charter warns about and it is fixed by the choice of unit alone.
2. **A per-message constant**, because chat encodings add role and delimiter tokens per message that no byte count ever sees, and that term dominates on short-message transcripts — where pure character counting is worst.
3. **Tool schemas are counted.**

**The accuracy caveat MUST be stated, not implied.** The estimate is a heuristic over a proxy with **no proven bound in either direction**: it over-counts dense ASCII prose and under-counts non-Latin scripts, base64-like content and unusual tool schemas. The estimate's documentation MUST state that it is an estimate, MUST state this formula, MUST state the unit and both constants **each with its rationale**, and MUST state the caveat naming its unbounded error.

**No code path may treat the estimate as exact.** Specifically: no path may convert an estimated figure into an `ai.TokenCount`, thread it through that type's presence bit, or route it into any cost figure. `R-CTX-008` makes this mechanically enforceable rather than aspirational — the number is unobtainable without its *estimated* label.

The estimate MUST NOT depend on a real tokenizer or a vocabulary: `L2C-01` allows the standard library and `src/ai`'s measured closure and nothing else, and `openspec/config.yaml:33` forbids a new top-level dependency without an ADR. The estimate is therefore a documented heuristic **by construction**, which is exactly why stating its method honestly is the requirement.

#### Scenarios

- **S-CTX-016** — **Charter AG-17.2 scenario 2, the "method stated" half.** Given the shipped estimate's documentation, when it is read, then it states the formula, states that the unit is **UTF-8 bytes** rather than characters or runes, names both constants with their stated values and a rationale for each, states that tool schemas are counted, and carries an explicit accuracy caveat naming the error as unbounded in both directions; and given the estimate's computation, when it is exercised over a table of requests — pure ASCII, CJK, an empty transcript, a tool-schema-bearing request, and a system-instruction-bearing request — then each row's figure equals the formula applied to that row's own byte total and message count, and the CJK row's figure exceeds the figure a rune count would produce for the same content.
- **S-CTX-017** — **Charter AG-17.2 scenario 2, the "never exact" half.** Given the shipped change's diff, when every path that produces an estimated figure is followed, then none constructs an `ai.TokenCount` from it, none routes it into a cost figure, and `cost_events.go`, `cost_usage.go` and every file under `backend/agent/src/ai/` are **byte-unchanged**; and given a consumer holding an estimated accounting value, when it reads the figure, then it necessarily receives the *estimated* provenance in the same call, so treating the figure as exact requires ignoring a value the call handed it.

---

### R-CTX-010 — The estimate is a pure function, and its determinism is asserted DIRECTLY

The estimate MUST be a **pure function** of its request value: no clock read, no environment read, no randomness, no I/O. Identical request values MUST yield identical figures, on every call and across runs.

**This MUST be asserted by a test rather than inferred from a guard, and the reason is a correction to an assumption `explore.md` § 6 carried.** `ambient_authority_test.go:73-94` forbids exactly four packages — `os`, `os/exec`, `syscall`, `io/ioutil`. **It does not forbid `time`.** So "no environment read, no I/O" is machine-checked, but "no clock" is **not**: an estimate that read a clock and drifted would pass every existing guard. Determinism is therefore a **direct assertion**, and this requirement exists so that it is written as one instead of being assumed away.

#### Scenarios

- **S-CTX-018** — Given each row of the estimate's table, when the estimate is computed **twice** over the same request value and once more over an **independently re-constructed equal** request, then all three figures are identical; and given the shipped estimate's source, when its imports and call graph are read, then it reads no clock, no environment, no randomness source and performs no I/O.

---

### R-CTX-011 — Accounting is over the PRE-hook request, and exactness is NOT claimed even on the reported path

`applyPreRequestHook` (`loop.go:326`) derives a new request **after** `buildLoopRequest` (`loop.go:304`), **inside** the one-turn surface and therefore downstream of the turn boundary where accounting runs. Accounting consequently measures the **pre-hook** request on every path.

**This MUST be stated as a semantic of the accounting, not left as a comment.** Concretely:

- the *reported* provenance MUST be documented as *"the provider reported this figure **for the pre-hook request**"* — never as "for the bytes sent";
- the prompt's accounting field MUST carry the same statement, so the inexactness lives where a consumer must read it;
- **no requirement, doc comment or test in this change may claim the accounting is exact with respect to what is sent**, on any of the three provenance paths.

The divergence MUST be **recorded by a test rather than asserted away**: a test installing a request-mutating pre-request hook MUST record that the counted request and the sent request differ, rather than asserting an equality that a hook can falsify.

#### Scenarios

- **S-CTX-019** — Given a counting-capable fixture and a harness carrying a request-mutating pre-request hook, when one logical turn is driven, then the accounting the strategy received corresponds to the **pre-hook** request the turn's own builder produced, the fixture's captured counting request and the provider's recorded sent request **differ**, and the test records that divergence rather than asserting the two are equal; and given the shipped documentation of the reported provenance and of the prompt's accounting field, when each is read, then each states that the figure is for the pre-hook request.

---

### R-CTX-012 — AG-17 emits nothing, registers nothing, and releases no substrate

AG-17 MUST introduce **no new `EventKind`**. The compaction-family kinds already exist from AG-06 (`compaction_events.go`); **AG-18 is their first caller**, and AG-17 emits none of them. `event_descriptor.go` and `event_registry_test.go` MUST be byte-unchanged and the every-kind-constructible guard MUST pass at its committed kind count.

AG-17 MUST declare **no new package-wide contract row**, and `doc.go` and `doc_contract_guard_test.go` MUST be **byte-unchanged**. The rule is the guard's own (`doc_contract_guard_test.go:19-22`): the trigger is appending a guarded paragraph to the package contract, which happens when a milestone declares a new package-wide **upward** guarantee. AG-17 adds no upward surface — it emits no event (`L2C-03`) and adds no exported history route (`L2C-07`) — and its shipped default is inert by type. **AG-15's failover seam is the decisive precedent**: structurally identical (a nil-default harness field, a one-method interface, an inert shipped default), it added **no** contract row, and `agent-retry-failover/spec.md` contains zero occurrences of that row family.

**Because AG-17 emits nothing, every enumerated closed event sequence in every promoted spec MUST remain true unamended — and this MUST be asserted rather than assumed.** The two at risk are named: `S-LSK-001`'s enumerated nil-path turn sequence (`agent-loop-skeleton/spec.md:65`), whose implementing test enforces closure by length equality (`loop_test.go:361`), and `R-CAN-002`'s enumerated wind-down order (`agent-cancellation-tree/spec.md:65`, `S-CAN-013`). AG-17's consultation sits in the run driver's outer loop, **outside the one-turn surface entirely**, and adds no event to any bracket on any path.

The substrate obligation itself — the byte-unchanged file list and the two filters' widening rule — is owned by `agent-loop-skeleton`'s `R-LSK-004` and is recorded in this change's [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) delta, not duplicated here.

#### Scenarios

- **S-CTX-020** — **Closed sequences proved safe, not assumed safe.** Given the AG-17 change merged, when the tests implementing `S-LSK-001` and `S-CAN-013` are run **unmodified and byte-unchanged**, then both pass; and when the recorded stream of a multi-logical-turn harness run with the never-compact default installed is scanned, then it carries **zero** events of any compaction-family kind and its kind sequence is equal, position for position, to that of the identical run with the seam nil (`S-CTX-007`).
- **S-CTX-021** — **No row, no kind, no release.** Given the merge base of the AG-17 branch with `origin/main`, when the diff is taken over `backend/agent/src/agent/`, over `backend/agent/src/ai/` and over `go.mod`/`go.sum`, then the contract-row table is byte-unchanged and still carries exactly the row set it carried at `origin/main@b0de5bf6` — `L2C-01` through `L2C-08` (`doc_contract_guard_test.go:62-71`) — `doc.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `compaction_events.go`, `cost_events.go`, `cost_usage.go` and `history.go` are byte-unchanged, the diff under `backend/agent/src/ai/` is **empty**, the `go.mod`/`go.sum` diff is empty, and the every-kind-constructible guard passes at its committed kind count.

---

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-CTX-001** | **External-package verifiability.** Every behavioral scenario above MUST be verifiable by `cd backend/agent && make test`, and every behavioral test MUST live in `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all. The estimate's own table and the resolver's three-state table MAY be internal — they are pure functions with no external surface, the same allowance AG-16 took for its pure converter — provided every claim about what a **strategy** observes is also asserted externally through a recording strategy. |
| **NFR-CTX-002** | **Determinism and race cleanliness.** Every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by `agenttest.Gate`, channel reads and channel closes; **no test may synchronize by sleep, timeout or wall-clock ordering**. Evidence MUST be recorded from a run with `-count=1`; a cached run is not evidence. |
| **NFR-CTX-003** | **Ambient authority and boundaries.** Production sources added by this change MUST NOT import `os`, process execution, `syscall` or legacy I/O; the ambient-authority guard and the import-boundary guard MUST pass with **zero** change. The guard's forbidden set does **not** include `time` (`ambient_authority_test.go:73-94`), so the clock obligation is discharged by `R-CTX-010`'s direct assertion, never by the guard. |
| **NFR-CTX-004** | **Substrate.** Every file named by `R-LSK-004` MUST be byte-unchanged — `doc.go`, `doc_contract_guard_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `compaction_events.go`, `cost_events.go` and `run_events.go` included — as MUST `history.go`, `cost_usage.go`, every file under `backend/agent/src/ai/`, and `go.mod`/`go.sum`. AG-17 requests **no** release. Both substrate filters MUST be widened by **exact filename suffix only**, one entry per file AG-17 introduces, with no wildcard, no prefix match and no directory-level relaxation, and MUST carry an **identical** entry set. |
| **NFR-CTX-005** | **Review budget.** `openspec/config.yaml` forecasts a 400-line review budget. This change ships as a **single** pull request under a pre-authorised `size:exception` against a 1000-line budget, forecast at 1560–2445 changed lines including SDD markdown. The pull-request description MUST state why the change does not fit the default budget. If `sdd-tasks` forecasts beyond that, the slicing boundary is the charter's own DAG: AG-17.2's types, estimate, resolver and fixture first (wired to nothing, zero behavior change), then AG-17.1's consultation. |

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-17 closes more than it does. Each row names its owning milestone.

| Not claimed | Owner and why the deferral is safe |
|---|---|
| **Compaction itself** — summarising, transcript surgery, protected recent turns, compaction event emission | **AG-18** (`0003:1655-1665`). `agent-run-driver/spec.md:342` splits the two explicitly: AG-17 inserts the check, AG-18 implements. `R-CTX-003` makes AG-17's verdict type **incapable** of asking for compaction, so the deferral is enforced by the compiler rather than by intent |
| Any budget **value**, threshold or configuration | **Layer 3.** Charter out-of-scope (`0003:1610`): "Layer 3 supplies the model's budget via options". AG-17 ships the type and the carrying field; every value is the caller's. `R-AGS-015` (`agent-v1-scope/spec.md:245`) forbids Layer 2 deciding Layer 3's content |
| **When** to compact — any threshold arithmetic over budget against accounting | **AG-18 / Layer 3** (`0003:1665`). AG-17 hands the strategy both numbers and asks nothing of them |
| A per-turn budget, or a budget on the one-turn options value | **Not this milestone.** One harness-scoped field only (`R-CTX-005`); the model is fixed per harness in v1 |
| Any new `EventKind` | **Never in AG-17** (`R-CTX-012`). AG-06 minted the compaction family; **AG-18** is its first caller |
| A new package-wide contract row | **Not this milestone** (`R-CTX-012`), on the AG-15 precedent. AG-19's child runs and AG-18's compaction may re-open the question on their own evidence |
| A real tokenizer, a BPE vocabulary, or a bounded-error estimate | **Out of scope and structurally impossible.** `L2C-01` allows the standard library and `src/ai`'s measured closure only; `openspec/config.yaml:33` forbids a new top-level dependency without an ADR. The estimate is a documented heuristic by construction (`R-CTX-009`) |
| Exactness of the accounting with respect to the bytes actually sent | **Not claimed on ANY path**, including the reported path — `R-CTX-011`. The pre-request hook derives its request downstream of this seam |
| Carrying the counter's error, or any diagnosis of non-conformance, to the strategy | **Not this milestone.** Collapsed into the *unavailable* label (`R-CTX-007`); AG-18 may widen the prompt non-breakingly if diagnosis is ever needed |
| Any edit under `backend/agent/src/ai/**`, including `ai.TokenCounter` | **Never in Layer 2** (`R-RUN-012`, `agent-run-driver/spec.md:302`). Layer 1 is consumed, never edited; the counting contract ships complete and conformance-tested |
| A second counting contract in Layer 2 | **Forbidden, not deferred** — `R-AMP-017` (`ai/provider.go:123-125`): one contract per capability (`R-CTX-006`) |
| Re-opening AG-15's retry gates or AG-16's cost accumulator | **CLOSED by their own milestones.** AG-17 reads the outer loop's state and touches neither; `R-RTY-002` is held, not amended (this change's [`../agent-retry-failover/spec.md`](../agent-retry-failover/spec.md) delta) |
| Persistence or session reload of a budget or an accounting figure | **Layer 3.** The harness holds state in memory and never touches a file (`0003:110`) |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active.

- Both charter leaves are behavior, so every behavioral scenario is **RED-first**.
- **`R-CTX-001` closes only on its bite** (`S-CTX-003`), which MUST move the consultation into the attempt loop and observe `S-CTX-002` fail at three consultations. Without it, the cardinality requirement is untested against the exact reading it exists to exclude — and that reading is what a downstream phase reconstructs from the charter Gherkin alone.
- **`R-CTX-007` closes only on its bite** (`S-CTX-013`), which MUST collapse the three states to two and observe **both** non-conformance rows fail.
- **`R-CTX-008` closes only on its bite** (`S-CTX-015`).
- **`R-CTX-004` closes only on its bite** (`S-CTX-008`); a byte-identity pin that a stray emission would survive is a vacuous pin.
- `R-CTX-010`'s determinism is asserted directly; **no guard covers it**.
- Evidence MUST come from a `-count=1` run. A cached suite result is not evidence.

## Acceptance criteria

1. Every `S-CTX-001`…`S-CTX-021` has recorded evidence; all four bites (`S-CTX-003`, `S-CTX-008`, `S-CTX-013`, `S-CTX-015`) are RED-recorded with failing output **before** their GREEN.
2. All **four** charter Gherkin scenarios (`0003:1624-1627`, `:1629-1632`, `:1642-1645`, `:1647-1650`) are mapped in the Coverage table and closed; none is reduced.
3. `cd backend/agent && make test` green under `-race` with `-count=1`; `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` all clean — `vuln-check` is **not** in `make all`.
4. `CheckStream` accepts every recorded stream **unmodified**, with `stream_check.go` byte-unchanged.
5. All five delta specs of this change are written and each cited line is re-read against the shipped change by `sdd-verify`.
