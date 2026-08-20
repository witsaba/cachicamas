# Delta for `ai-contract-vocabulary` — `V-OUT-13`'s four hook points and its "observers never synchronous" clause are REALISED, with no Layer 1 edit

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5), `0003:1864-1918`
> **Modifies**: `ai-contract-vocabulary` ([`../../../../specs/ai-contract-vocabulary/spec.md`](../../../../specs/ai-contract-vocabulary/spec.md)) — **by APPENDING one requirement**, `R-AIV-014`, alongside `R-AIV-007` (the excluded-term register, whose `V-OUT-13` row sits at `spec.md:331`). **`R-AIV-007` and the `V-OUT-13` row itself are byte-unchanged.**
> **SPEC TEXT ONLY — no Layer 1 code is touched.** `R-RUN-012` forbids Layer 2 editing `backend/agent/src/ai/**`, and AG-20's diff under that tree is **empty** (`R-HKS-010` / `S-HKS-024`). This delta records that an exclusion the register wrote has now been honoured by the layer it was assigned to; it does not move a term, re-own a row, or add one.
> **Why an ADDED requirement rather than a MODIFIED `R-AIV-007` or an edited row.** `R-AIV-004` makes term identifiers **stable and append-only** and requires any addition to arrive by a dated amendment blockquote; `V-OUT-13`'s definition, owner and note are all still exactly right, so there is nothing in the row to change and changing it would be churn against a stability rule. `R-AIV-007`'s own requirement text is likewise unaffected. `R-AIV-014` is the next free requirement identifier (`R-AIV-001`…`R-AIV-013` are allocated); `S-AIV-032` is the next free scenario identifier (`S-AIV-001`…`S-AIV-031` are allocated, verified in this phase).
> **Ownership**: the taxonomy is owned by [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md). This delta owns only the register's audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| The `V-OUT-13` row itself — its term, owner, note and provenance (`spec.md:331`) | **Byte-unchanged, and every cell is still correct.** The term is *hook*; the owner is *Layer 2 and Layer 3*; the note states that Layer 1's obligation is the request rebuild, *"which is the mechanism the pre-request hook stands on"*, and that the hook itself is not Layer 1's. AG-20 confirms all four cells by delivery. Editing a correct row would violate `R-AIV-004`'s stability rule for no gain |
| `R-AIV-007` — excluded terms carry a named owner | **Untouched.** AG-20 excludes no term and re-owns none |
| `R-AIV-004` — stable, append-only term identifiers | **Honoured.** No identifier is added, reused or renumbered |
| `V-REQ-29` — request rebuild | **Untouched and confirmed as the mechanism it was named to be.** The pre-request chain composes over it: each element receives a value and returns a value, and the copy-on-write rebuild is what makes composition safe rather than mutating |
| Everything under `backend/agent/src/ai/` | **Byte-unchanged.** `R-RUN-012` holds; the diff under that tree is empty |

## ADDED Requirements

### R-AIV-014 — `V-OUT-13`'s exclusion is HONOURED: Layer 2 built all four hook points and the asynchrony clause, and Layer 1 was not touched

**`V-OUT-13` excluded *hook* from Layer 1 and assigned it to Layer 2 and Layer 3, naming four points and one discipline.** That row was written before any of them existed. AG-20 realises the Layer 2 half in full, and the realisation MUST be recorded here rather than left for a reader to infer from a milestone document, because the register's own value is that a reader can resolve a term without leaving it.

**Each of the row's four named points now exists in Layer 2:**

| `V-OUT-13`'s point | Realised by | Family |
|---|---|---|
| pre-request | AG-08's shipped callable, **composed into a chain** by AG-20 | mutating |
| pre-compact | AG-20, spliced inside the one compaction operation before the provider call | mutating |
| post-turn | AG-20, once per logical turn over an enumerated set of exits | observing |
| session-start | AG-20, once per harness **value** | observing |

**The row's discipline clause — *"observers never synchronous on the streaming path"* — is realised MECHANICALLY, and the distinction matters to this register.** The clause is a vocabulary statement, and a vocabulary statement is satisfied by a mechanism or it is satisfied by nothing. AG-20 satisfies it by type and by structure rather than by convention: the two observing families' function types have **no result parameters**, so an observer cannot signal a mutation or a failure back to the runtime; dispatch happens on a per-run lane whose enqueue never blocks and whose goroutine is neither the run's nor any event-delivery goroutine; and the observers' invocation context is value-stripped, so the one context-carried route back onto a run's event lane is unreachable from an observer. `R-AGE-008`'s standard — *"A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement"* — is the bar, and it is met.

**Two boundary facts MUST be stated so the row is not later misread:**

1. **Layer 1 was not touched, and the row's note explains why it did not need to be.** Layer 1's obligation is `V-REQ-29`, the request rebuild, *"which is the mechanism the pre-request hook stands on"*. AG-20 composes over that rebuild and adds nothing to it: each chain element receives a request value and returns a request value. The diff under `backend/agent/src/ai/` is **empty**, so `R-RUN-012` holds and the row's Layer 1 exclusion is honoured in the strongest available sense.
2. **The row's *Layer 3* half is NOT discharged, and it is the half most at risk of being wrongly closed.** `V-OUT-13` names *"Layer 2 **and Layer 3**"* as owners. Layer 2 owns the **taxonomy**; Layer 3 owns the **concrete hooks** — cache-breakpoint placement, compaction policy, telemetry — at doc 0004 CO-24.1 / CO-24.2, which `S-AGS-048` already maps. AG-20 ships **zero** concrete hooks, verbatim from its charter: *"Any concrete hook implementation (Layer 3 wires them)"* (`0003:1874`). **A taxonomy is not a wiring**, and the row's two owners MUST NOT be collapsed into one.

**This requirement adds no obligation to Layer 1 and no term to the register.** It records a discharge, so that a later reader asking "who built the hook, and did Layer 1 have to change?" gets both answers from the register itself.

#### Scenarios

- **S-AIV-032** — **AG-20: `V-OUT-13`'s exclusion is checked against the shipped code, not against this register's prose.** Given the merged AG-20 change, when a reviewer opens each of the row's four named hook points, then each exists in Layer 2 with at least one independently verifiable scenario in `agent-hook-taxonomy`, and each is classified mutating or observing by its **function type** rather than by a doc comment — an observing type's function signature declares **no result parameters**, asserted by a compile-time refusal rather than by a runtime check. When the reviewer looks for a synchronous observer dispatch on the streaming path, then there is none: a run carrying a deliberately stalled observing hook produces an event stream byte-identical to the same script with no hooks installed, `CheckStream` accepts it unmodified, and the observing hook's recorded stack contains neither the run driver's frame nor the event forwarder's. When the reviewer takes `git diff` over `backend/agent/src/ai/` against the merge base, then it is **empty**, and `go.mod`/`go.sum` are byte-unchanged. And when the reviewer reads the row's Layer 3 half, then the concrete hooks are still assigned to doc 0004 CO-24.1 / CO-24.2 and AG-20 ships none — a discharge that also closed those would be over-claiming. Cross-referenced to `R-HKS-001` / `S-HKS-003`, `R-HKS-007` / `S-HKS-017` / `S-HKS-018`, `R-HKS-010` / `S-HKS-024` and `R-AGS-016` / `S-AGS-066`.
