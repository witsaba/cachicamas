# Exploration — AG-17: Add the context strategy seam and token accounting

> Phase: `sdd-explore` · Change: `cachicamas-agent-context-strategy`
> Milestone AG-17 (Layer 2, Wave 4, 17 of 24) — doc `0003` § AG-17
> Worktree `cachicamas-worktrees/ag17`, branch `feat/agent-layer2-wave4-ag17`, base `origin/main@b0de5bf6`
> Artifact store: hybrid — mirrored to Engram as `sdd/cachicamas-agent-context-strategy/explore`
>
> Every claim below was opened and read at the cited `file:line`. The five
> load-bearing citations were independently re-verified by the orchestrator
> before this file was written.

## 1. Current state

### 1.1 The provider call site and the two candidate seam positions

`provider.Stream(ctx, req)` fires at `backend/agent/src/agent/loop.go:351`, inside `Turn(...)`.

`Harness.Run` (`harness.go:371-684`) calls `Turn` once per **retry attempt**, inside AG-15's
inner attempt loop (`harness.go:530-608`, `for attempt := 1; ; attempt++`). The outer
per-**logical-turn** loop (`harness.go:479-670`) builds `transcript := transcriptFromHistory(hist)`
once at `harness.go:498`, *before* entering the attempt loop.

Therefore a retried logical turn issues **multiple** `provider.Stream` calls over one
transcript, within **one** outer-loop iteration. This is the change's central ambiguity
and it is resolved in § 2.

### 1.2 Decisive placement evidence

`openspec/specs/agent-run-driver/spec.md:342`, in the *Explicit non-requirements* table:

> | A compaction check between turns | **AG-17** inserts it at AG-13's turn boundary; **AG-18** implements compaction. Nothing here anticipates its shape. *(Still unclaimed at AG-16, which inserts a cost emission at the turn boundary and no compaction check; the two are independent insertions and AG-16 pre-empts nothing about AG-17's shape)* |

`openspec/specs/agent-history/spec.md:260`, same table shape:

> | Context-window accounting over the transcript | AG-17 |

The seam's position is therefore **already specified by a promoted spec**, not open for
design to choose: AG-13's turn boundary — the outer per-logical-turn loop, at the site
where `transcript` (`harness.go:498`) and `timing` (`harness.go:510`) already resolve once
per logical turn, before the attempt loop at `harness.go:530`.

### 1.3 The nil-safe injection convention to reuse

From `failover_policy.go` (AG-15.3), the established Layer 2 seam shape:

- `Harness.Failover FailoverPolicy` — a nil-default field on `Harness`, deliberately **not**
  on `TurnOptions`. `failover_policy.go:19-26` states the rule: *"a seam belongs where its
  consumer is"* — exhaustion and attempts are run-driver concepts, so the seam lives on the
  run driver. A turn-boundary strategy is a run-driver concept by the identical argument.
- A one-method interface (`Resolve(ctx, FailoverPrompt) FailoverVerdict`, line 38) with typed
  exported-field prompt and verdict structs.
- Nil is never called and behaves exactly as the shipped default — the *inertness pin*.
- A shipped, installable no-op implementation (`NoOpFailoverPolicy{}`, lines 65-76) plus a
  compile-time guard `var _ FailoverPolicy = NoOpFailoverPolicy{}` (line 79).

AG-08's pre-request hook (`TurnOptions.PreRequestHook`, `loop.go:71-90`, invoked at
`loop.go:326`) is the **wrong** model to copy: it lives on `TurnOptions` and therefore fires
once per *attempt*, since `Turn` runs once per attempt.

### 1.4 Layer 1 already ships the counting capability

`backend/agent/src/ai/provider.go:102-134`:

```go
type TokenCounter interface {
    CountTokens(ctx context.Context, req Request) (TokenCount, error)
}
```

Its doc comment settles three things AG-17.2 would otherwise have to invent:

- It is advertised **only** by satisfying it — `counter, ok := provider.(TokenCounter)` — and
  by no field, flag, registration call or catalog entry (`ai-minimum-capabilities` § 9).
- `ok == false` is a **clean absence** (`R-AMP-018`): not an error, not a zero standing in for
  one, and *"this package supplies no substitute, estimate or default."* That sentence is
  precisely why the documented estimate is Layer 2's responsibility and not Layer 1's.
- A provider that satisfies the contract and then declines to answer is **non-conformant, not
  absent** (`R-AMP-019`) — advertising binds. The estimate path is for absence only, never a
  fallback for an advertised-but-failing counter.

It is already conformance-tested (`agenttest/conformance_capabilities.go:152-180`) and already
proven discoverable from outside the package (`agenttest/provider_test.go:126-183`).
`ai.TokenCounter` is inside Layer 2's allowed imports (`import_boundary_test.go:127-130`).

**AG-17.2 must not declare a new interface.** Doing so would violate `R-AMP-017`
(one contract per capability, never an aggregate).

### 1.5 Fake-provider gap

`agenttest.Provider` (`agenttest/fake_provider.go:55-117`) — the one exported fake that
`src/agent` tests use — has **no** `CountTokens` method. It is the "without the capability"
fixture as it stands. The "counting-capable fake" the AG-17.2 scenario requires needs a new
**exported** fixture in `agenttest`, mirroring the internal `stubProviderWithTokenCounter`
(`agenttest/provider_test.go:133-143`) embedding shape.

### 1.6 The usage idiom to compose with, never duplicate

AG-16's `costFromUsage` (`cost_usage.go:31-60`) and `ai.TokenCount`'s presence bit
(`ai/usage.go:34-77`, *"absence is not zero"*) already encode one fact: **Layer 1 reported it.**

An AG-17.2 estimate must never be threaded through that presence bit. That bit means
*reported*, not *estimated*; conflating them would silently corrupt `agent-cost-events`'
cumulative accounting semantics and would be the exact failure the AG-17.2 scenario
*"an estimate never masquerades as exact"* forbids.

### 1.7 No budget type exists

A repo-wide search finds no token-budget type in Layer 1 or Layer 2 — only "retry budget"
and "attempt bound" prose. AG-17 must define the budget's **type and shape**; its **value**
stays Layer 3's, per the charter's out-of-scope line.

### 1.8 Compaction events are AG-18's to call

`compaction_events.go` defines `CompactionStarted` / `CompactionFinished` / `CompactionFailed`.
These are AG-18's callers, not AG-17's. AG-17 introduces **no new `EventKind`** and appears in
no closed-sequence assertion in any promoted spec, nor in `event_registry_test.go`'s witness
table.

### 1.9 History surface stays closed

`history.go`'s single commit path is untouched; the strategy reads through the same
`transcriptFromHistory(hist)` route the driver already uses. *"No history was mutated"* is
provable by this codebase's existing nil-default byte-stability precedent (`S-PRH-002`,
`S-LSK-015`, `R-RUN-012`): run identical scripts with the default strategy present and absent,
then assert byte-identical event streams and byte-identical `hist.Entries()` read-backs.

## 2. The resolved ambiguity — "N times, before each provider call"

The AG-17.1 scenario says the strategy is *"consulted exactly N times, before each provider
call."* Under AG-15 retries these two clauses genuinely diverge: a logical turn that retries
issues more than one provider call.

**Resolution: N counts logical turns, not attempts.** Three independent reasons, in
descending strength:

1. **A promoted spec already says so.** `agent-run-driver/spec.md:342` places the check at
   *AG-13's turn boundary*. The turn boundary is the outer loop.
2. **The alternative would break AG-15's retry pin.** `R-RTY-002` and the comment block at
   `harness.go:513-529` guarantee a retried attempt runs over a **byte-identical transcript,
   reused by reference**. A seam consulted inside the attempt loop is a seam that — once AG-18
   makes it capable of compacting — could mutate the transcript between two attempts of one
   logical turn, falsifying that pin. Placing it at the turn boundary keeps AG-15's guarantee
   true by construction rather than by convention.
3. **Consistency with the sibling seam.** Retry timing resolves once per logical turn at
   `harness.go:510` for the same structural reason, and says so in its own comment.

This resolution MUST be stated explicitly in the spec, not left implicit — a downstream phase
that reads only the Gherkin will otherwise reconstruct the per-attempt reading.

## 3. Affected areas

| Area | Change |
|---|---|
| `backend/agent/src/agent/harness.go` | New `Harness.ContextStrategy` field; consultation between `:498` and `:530` |
| `backend/agent/src/agent/` (new file) | The context strategy interface, its prompt/verdict types, the never-compact default |
| `backend/agent/src/agent/` (new file) | Token-accounting helper: `ai.TokenCounter` type assertion + labelled estimate |
| `backend/agent/src/agenttest/` | New exported counting-capable fake fixture |
| `openspec/specs/agent-run-driver/spec.md` | Delta — the `:342` non-requirement row closes |
| `openspec/specs/agent-history/spec.md` | Delta — the `:260` non-requirement row closes |
| `openspec/specs/agent-v1-scope/spec.md` | Delta — seams 5 and 6 back-annotation (`R-AGS-014`, `S-AGS-023`) |

## 4. Approaches considered

### 4.1 The seam

| # | Approach | Verdict |
|---|---|---|
| 1 | **`Harness.ContextStrategy` field**, nil-default, consulted once per logical turn, mirroring `FailoverPolicy` exactly | **Recommended.** Matches the only authoritative placement evidence; reuses an established, tested convention; "N times for N logical turns" holds trivially under retry |
| 2 | `TurnOptions` field beside `PreRequestHook`, consulted inside `Turn` | **Rejected.** Contradicts `agent-run-driver/spec.md:342`; fires per attempt, breaking the count under any retry; `Turn` has no logical-turn concept — the same argument `failover_policy.go:19-26` uses to keep failover off `TurnOptions` |

### 4.2 Token accounting

| # | Approach | Verdict |
|---|---|---|
| 1 | **Type-assert the existing `ai.TokenCounter`** | **Recommended.** The capability already ships and is conformance-tested |
| 2 | Declare a new Layer 2 counting interface | **Rejected.** Duplicates a shipped Layer 1 contract and violates `R-AMP-017` |

## 5. Recommendation

Approach 1 for the seam plus reuse of `ai.TokenCounter` for accounting, composing with — never
duplicating — AG-16's presence-preserving idiom, so the estimate stays distinguishable from a
reported figure at the type level rather than by convention.

Proposed bite-ID prefix: **`R-CTX-` / `S-CTX-`**. Verified free: the 60 prefixes currently in
use across `openspec/specs/` do not include `CTX`.

One spec covers both AG-17.1 and AG-17.2 under the single change
`cachicamas-agent-context-strategy`.

## 6. Guards enumerated

| Guard | Risk to AG-17 |
|---|---|
| `import_boundary_test.go` | **Safe** — the `ai.TokenCounter` assertion stays inside the already-allowed `src/ai` |
| `ambient_authority_test.go` | **Watch** — the estimate must stay pure: no clock, no env read |
| `doc_contract_guard_test.go` | **Open question** — byte-pinned `L2C-NN` table; AG-17 adds no new `EventKind` and no new upward-visible fact, so "no new row" is likely, but propose must state this rather than assume it |
| `event_registry_test.go` | **Safe** — no new event kind |
| `history_surface_guard_test.go` | **Safe** — read-only via `Entries()` |
| `stream_check.go` byte-unchanged pin | **Safe** — no new bracket |
| Closed-sequence specs (`S-LSK-001`, `R-CAN-002`) | **Safe by construction** — the default emits nothing; must be asserted, not assumed |

## 7. Risks carried into propose

1. **Placement ambiguity** — if propose does not cite `agent-run-driver/spec.md:342`, a
   `TurnOptions`-based design will fight the retry loop's per-attempt semantics.
2. **Fake-provider gap** — apply must add an *exported* counting fixture matching the
   `stubProviderWithTokenCounter` embedding shape, not an ad hoc one.
3. **Estimate/report conflation** — threading an estimate through `costPresence`'s
   "Layer 1 reported it" bit would violate the never-masquerades requirement and could
   corrupt `agent-cost-events`' cumulative semantics.
4. **`agent-v1-scope` back-annotation** — `R-AGS-014`'s amendment rule and the AG-15 precedent
   (`agent-v1-scope/spec.md:130`) require the seam-5/seam-6 `AGS-S` rows to be back-annotated.
   This is spec hygiene enforced by no Go test, so it is easy for `sdd-archive` to skip.
5. **Unresolved `L2C-NN` question** — see the guard table above.

## 8. Ready for proposal

Yes. The placement decision (a `Harness` field, per `agent-run-driver/spec.md:342`) and the
`ai.TokenCounter` reuse decision are settled inputs, not open questions.
