# Design: AG-18 — Implement compaction (`cachicamas-agent-compaction`)

> Phase: `sdd-design`, run **before** `sdd-spec` deliberately and in series — the decisions below are the input `sdd-spec` writes requirements against.
> Worktree: `cachicamas-worktrees/ag18` off `origin/main@f6acc0d2`. Every file:line below was opened and re-verified against **this** tree during this phase; none is carried forward unresolved.
> Inherits the proposal's Deliverable 0 (A1–A8), delivery (`exception-ok`, single PR, 1000 counted lines excluding `openspec/`), and strict RED-first TDD with `go test -race -count=1`.

## Technical Approach

Compaction is one internal operation, `runCompaction`, invoked from two doors: the captured `ContextVerdict` at the existing seam (`harness.go:524-530`, verdict currently discarded), and a new on-demand `Harness` method that wraps the same operation in its own minimal run bracket. The operation runs in the loop's verified no-turn-open gap (previous `CloseTurn` at `harness.go:668`; next `turn_start` only inside the next `Turn`), inside a **dedicated compaction turn bracket** built from the existing exported `NewTurnStart`/`NewTurnEnd` constructors — so the compaction family's `PlacementTurn` rule (`stream_check.go:161`, verified verbatim: `if d.Placement == PlacementTurn && !turnOpen`) is satisfied with `stream_check.go` byte-unchanged. History gains exactly one new exported mutating route — a **prefix replacement** through the single commit primitive (`history.go:332`) — plus internal, commit-funneled **turn marks** that make `CompactionSpan`'s TurnIDs derivable. Atomicity is ordering: nothing commits until the model call succeeded and the replacement passed the same pairing validation `NewSeededHistory` runs.

## Architecture Decisions

Each decision carries choice, rejected alternatives, rationale, and the criterion that would overturn it. AD-1…AD-8 settle the proposal's D1–D7+A8; AD-9…AD-12 are new, forced by evidence found in this phase.

### AD-1 (settles D1, **modifies the proposal's recommendation**) — Cut resolution is retraction to a marked turn boundary, never forward expansion

**Choice**: The verdict names a naive cut as an index into the transcript it received (`transcriptFromHistory` is 1:1 with `Entries()`, `harness.go:270-277`). Resolution retracts the cut **backward only**, to the greatest recorded turn-mark boundary ≤ the naive cut (AD-9), then verifies pairing-closure over `[0, cut)` by replaying `resolveOpenSet`-style CallID correlation (`history.go:408-432`) read-only over `Entries()` — a pure helper, one new file. If verification finds a straddling pair (possible only on unmarked/adversarial input), it iterates: retract to the earliest open call's entry index and re-verify. Termination: the cut is a non-negative integer that strictly decreases each iteration and is bounded below by 0; at 0 the span is empty and compaction reports failure typed (nothing to compact).
**Rejected**: (a) The proposal's **two-directional expansion to a fixed point** — expanding the end forward compacts entries at/after the strategy's stated boundary, which are exactly the entries designated protected; the charter's protection clause is MUST-level and beats "compact as much as asked". Retraction makes protection monotone by construction: the protected tail can only grow. The charter's "the boundary moved to include the whole pair" is satisfied — the pair ends up whole on the protected side. (b) Exposing `History`'s private `openCall` index — widens the guarded exported surface (`S-HIS-030`/`S-HIS-042`) for information fully recomputable from public `Entries()` (correlation is by `ai.ToolCall.ID()`/`ai.ToolResult.CallID()`, both on the message parts).
**Overturn if**: `sdd-spec` finds a charter scenario requiring the compacted span to grow past the requested boundary. None exists in `0003:1682-1760`.

### AD-2 (settles D2) — The summary entry is discriminated by a third `EntryOrigin` member

**Choice**: `EntryOriginSummarized`, beside `EntryOriginAppended`/`EntryOriginSynthesized` (`history.go`, minted at `commitAppendOp`'s `Entry{..., origin: origin}` construction, `:390`), mintable **only** by the new replacement commit op. The summary entry's message is the compaction completion's returned assistant message **unchanged** — Layer 2 authors nothing, not even a wrapper; distinguishability lives entirely in the envelope.
**Rejected**: a content sentinel — prohibited by name at `agent-history/spec.md:139` ("a real tool can emit any content bytes, so a content sentinel is forgeable and is prohibited"); the argument does not weaken for compaction. A new entry field/type — heavier than the discriminator that exists for exactly this purpose.
**Overturn if**: the third value proves unmintable without a second commit door — it does not; it rides the same primitive (AD-6).

### AD-3 (settles D3 + A8 — the load-bearing one) — The compaction bracket stands, and `R-CST-001`'s iff is satisfied by construction, not scoped away

**Choice**: One compaction operation is one dedicated turn bracket: fresh `TurnID` (own mint, e.g. `turn-cmp-N`, colliding with `mintLoopTurnID`'s `turn-lsk-N` namespace never), opened via `NewTurnStart`, closed via `NewTurnEnd` (`turn_events.go:42`, `:178`), emitted with `sendStamped` on the harness's own lane. Inside it: `compaction_started` → (`cost_turn` iff a completion was obtained) → `compaction_finished`|`compaction_failed` → `turn_end`. `R-CST-001`'s iff (`agent-cost-events/spec.md:42`) holds on every arm:

| Arm | Bracket close | `cost_turn`? | Iff status |
|---|---|---|---|
| Completion obtained, commit succeeds | `TurnOutcomeFinished` (non-aborted) | Yes, `Final`, from the completion's `ai.Usage` via `newCostTurnFromUsage` (`cost_usage.go:70`) | satisfied |
| No completion (provider error, `runCtx` cancellation) | `TurnOutcomeAborted` + typed `*Failure` (required by `NewTurnEnd`'s validate, `turn_events.go:166`) | No | satisfied — aborted close never emits |
| Completion obtained, then rejected (non-`Stop` finish, or commit validation failure) | outcome mapped from the completion's own finish reason (non-aborted) | Yes — the spend is real and is reported | satisfied |

Only a `FinishReasonStop` completion admits its summary; any other finish reason (a length-limited summary is silently lossy) takes the failed arm with the spend still counted.
**Consequence for A8**: `R-CST-001` needs a `MODIFIED` delta that is **additive, not a relaxation** — one new row in its enumerated per-path table (`:46-54`) naming the compaction bracket, because the table claims to enumerate every path and would otherwise be silently incomplete. The iff sentence itself ships **unchanged**. This overrides the proposal's either/or framing (scope the iff vs. re-decide D3): the design satisfies the sentence literally, so nothing is relaxed and nothing is put in its place because nothing is taken away.
**Rejected**: (a) scoping "turn" to model turns — relaxes a MUST and creates a replacement-invariant debt for no benefit; (b) emitting the compaction events inside the *next* model turn's bracket — misattributes them to a turn that did not perform the operation, and the on-demand path has no next model turn at all; (c) no bracket — `PlacementTurn` events would be rejected at `stream_check.go:161` in the no-turn-open gap, and `stream_check.go` is frozen (`R-LSK-004:103`, verified).
**Overturn if**: any promoted scenario is found asserting a closed run-level bracket count or length-equality over `Harness.Run` streams that the new bracket breaks — `sdd-tasks` MUST grep `harness_test.go` for enumerated sequences before apply (proposal risk 8; `S-LSK-001` verified not at risk — it asserts a direct `Turn()` call).

### AD-4 (settles D4, **overturns the proposal's recommendation**) — The on-demand gate is run-in-flight, reusing `signalMu`/`cancelRun`; no new state

**Choice**: `Harness.Compact` refuses **whenever a run is in flight**, detected exactly as `Interrupt`/`Shutdown` do: `h.signalMu` + `h.cancelRun != nil` (`harness.go:117`, `:124`, `:143-146`, set `:435`, cleared `:450`). The refusal is a typed sentinel error (`errors.Is`-distinguishable, the `ErrInterrupted`/`ErrShutdown` house pattern), returned synchronously; nothing is queued in any form (proposal Q4's answer carried forward).
**Rationale (the overturning evidence)**: the proposal's D4 recommended a new `turnInFlight` flag, noting the granularity gap. But `History` is unsynchronized by design — `commit` is plain field mutation with no lock — and the run loop reads/writes it between turns too (`:506-512`, `:668`). A concurrent on-demand mutation is therefore a data race **whether or not a turn is open**; "between turns of a live run" is not a safe window, so a turn-granular flag would gate the wrong predicate. Run-in-flight subsumes turn-in-flight: the charter's "Given a turn in flight" fixture is refused because its run is. With no run in flight, no turn is open anywhere — the strongest form of "compaction happens only at turn boundaries".
**Rejected**: steering-queue routing — inherently queues; the charter forbids queueing. New `turnInFlight` flag — gates an unsafe window as safe.
**Overturn if**: `sdd-spec` finds a charter clause requiring on-demand compaction *during* a live run — none exists; doc 0004's `/compact` arrives between prompts, i.e. between `Run` calls.

### AD-5 (settles D5, with one correction) — Compaction spend reuses `cost_turn`/`CostLabelFinal`; the fold site is explicit

**Choice**: `newCostTurnFromUsage(runID, compactionTurnID, CostLabelFinal, usage)` — no new label, no new kind, `cost_events.go`/`cost_usage.go` byte-unchanged. Distinguishability comes from the bracket's own `TurnID`, correlatable through `CompactionFinished.Span()`. **Correction to the proposal's wording**: the "existing forwarder fold" (`harness.go:562-580`) reads only `turnSink` during a `Turn` call; compaction events are emitted at harness level via `sendStamped` and never pass through it. The emission site therefore calls `total.add(ct)` itself, immediately before forwarding — same accumulator (`costAccumulator.add`, `cost_usage.go:118`), different fold site. The run's `cost_session` figures then include compaction spend on every close path unchanged.
**Overturn if**: A8 had removed the bracket — it did not (AD-3).

### AD-6 (settles the route) — One new exported `History` route: prefix replacement through the single commit primitive

**Choice**: `History.ReplacePrefix(count int, summary ai.Message) error` (name final at spec/tasks), implemented as a new `commitOp` dispatched from `commit` (`history.go:332`) — "the ONLY function that writes h.entries or h.open" stays literally true. The op: validates `count` against a recorded mark boundary and pairing-closure; validates the summary message (non-zero ID, no tool parts — a prefix replacement must not reopen the open set); then performs the replacement as **one** rebuild — new entries slice `[summary, tail...]` with identities renumbered `EntryID(i+1)` (the existing ordinal rule at `:390`), origins preserved for the tail, `EntryOriginSummarized` for the summary, marks rebuilt (in-span marks dropped, tail marks shifted). On any validation error, `h.entries` is untouched — R-HIS-002's "a failed commit is not a partial commit" carried forward verbatim.
**Prefix-only, not general span**: every charter scenario compacts the oldest region and protects the recent tail; a mid-span route has no consumer and would force a wider R-HIS-001 relaxation than needed. The A1 delta should be worded prefix-shaped. **Overturn if**: a charter scenario needs a non-prefix span — none does (`0003:1682-1760` re-read this phase).
**Spec consequence**: this route breaches `agent-run-driver`'s non-requirements row `:351` ("a new exported `History` method — not this milestone, and forbidden under this change", whose AG-17 annotation names AG-18 as the compaction family's first caller) — the row gains its AG-18 back-annotation recording the breach as forecast, under this change's own delta. `S-HIS-030`'s enumeration gains the route in the same commit as the route (A2).

### AD-7 (settles D6) — `ContextVerdict` grows one pointer field; the request is a struct both doors share

**Choice**:

```go
type CompactionRequest struct {
    Provider    ai.ModelProvider // required; may differ from h.Provider
    Options     /* the existing turn-option surface, tools/continuation inert */
    Instruction string           // required; Layer 3-authored, Layer 2 never writes one
    Cut         int              // naive boundary: compact transcript[0:Cut)
}
type ContextVerdict struct {
    Compaction *CompactionRequest // nil requests nothing
}
```

The zero verdict (`Compaction == nil`) requests nothing — the never-compact guarantee moves from a property of the type to a property of the **zero value**, exactly the deliberate amendment `S-CTX-005:81` instructs ("it must be deliberately amended by AG-18, never silently outgrown"). `NoOpContextStrategy` keeps compiling untouched (`context_strategy.go:58-63`). `Cut` is denominated in the transcript the prompt carried — the only coordinate system the strategy can name (it sees `[]ai.Message`, never TurnIDs or harness state), so `R-CTX-002` is preserved. `Resolve` stays consulted exactly once per logical turn (`R-CTX-001`); the harness captures the verdict at `:524-530` instead of discarding it, acts on it in the same gap, at most once per boundary (proposal Q5's answer), and **rebuilds `transcript` from the mutated history** before the attempt loop — the slice built at `:512` is stale after a successful compaction, and `R-RTY-002`'s by-reference argument requires the attempt loop to receive the rebuilt slice once, before any attempt.
**Rejected**: a second seam method — two consultations per boundary, contradicting `R-CTX-001`. A boundary hint in TurnIDs — the strategy cannot name them.
**Overturn if**: the option surface proves to need narrowing that `TurnOptions` cannot express non-confusingly — `sdd-spec`/`sdd-tasks` finalize the exact type after reading `TurnOptions`; the carrier shape is settled regardless.

### AD-8 (settles D7) — What "protected turns are byte-identical" means, and the assertion shape

**Choice**: value-identity over Layer 1 values **and** origin discriminators, positionally from the tail: for each protected pre-compaction entry `pre[cut+i]` and post-compaction entry `post[1+i]`, assert `pre.Message().Equal(post.Message())` (Layer 1's own `ai.Message.Equal`) **and** `pre.Origin() == post.Origin()`. Entry identity is **excluded and asserted to have changed**: on a fixture whose replaced prefix has length ≥ 2 (so tail ordinals provably shift), additionally assert `pre.ID() != post.ID()` for at least one protected entry — the renumbering is proven, not left untested. **`reflect.DeepEqual` over whole `Entry` values is forbidden in every protection assertion**: `Entry` embeds the ordinal `EntryID`, so `DeepEqual` either fails spuriously or — on a fixture that happens not to shift ordinals — encodes the false claim that identities survive compaction, which `R-HIS-005:106`'s own ordinal rule makes impossible (preserving old IDs would require minting non-ordinal identities, the C1 back door). Generation semantics per A3: identities are stable within a transcript generation; a compaction begins a new one; `TurnID` is the durable handle (`CompactionSpan` chose that axis at AG-06).
**Overturn if**: a way to preserve `EntryID`s without minting appears — `R-HIS-005` forbids the only mechanism, so effectively closed.

### AD-9 (**new — the material divergence from the proposal**) — Turn attribution lives in `History` as commit-funneled turn marks

**Problem the proposal missed**: `CompactionFinished` requires a valid `CompactionSpan{StartTurnID, EndTurnID}` (constructor validates it; `compaction_events.go` is frozen by `R-LSK-004:103`, so Span cannot become optional). But the compactable span crosses **prior `Run` calls** — `h.History` is caller-owned and serially reused (`harness_test.go:1013` pin, re-verified this phase), so a run-frame structure (the `costAccumulator` pattern) can never attribute earlier runs' entries to their TurnIDs, and `History` today records no turn structure at all.
**Choice**: `History` records a **turn mark** — `{TurnID, entry count}` — at turn close, written through the single commit primitive (the close op extended, still "commit writes everything"). The harness supplies the successful attempt's `TurnID`, captured from its own forwarded turn bracket events (safe: read after `<-forwarderDone`, `harness.go:591`), through a **package-private marked-close door** (`closeTurnMarked(TurnID)`) on the `newCostTurnFromUsage` precedent (`cost_usage.go:62-69` — a validated package-private door beside the public constructor is the blessed shape). Exported `CloseTurn()` keeps its exact signature and semantics (unmarked close), so no external caller breaks. Span derivation: the marks fully contained in `[0, cut)` name `StartTurnID` (first) and `EndTurnID` (last); AD-1's cut always lands on a mark boundary, which is what makes Span well-defined in whole turns. A span containing **unmarked** entries (a seeded prefix never driven by a run) is unattributable — compaction fails typed (`compaction_failed`), a named v1 limitation `sdd-spec` records rather than a silent fabrication of TurnIDs.
**Rejected**: (a) run-frame marks — cannot see prior runs (dead on the evidence above); (b) `Harness`-field marks — duplicates state that describes the History onto a value-form type, and desynchronizes the moment the caller hands the same `History` to another harness; (c) caller-supplied Span on the on-demand method — lets a caller lie about attribution and breaks the one-path symmetry; (d) making Span optional — requires editing frozen `compaction_events.go`.
**Overturn if**: `sdd-spec` finds marks violate a promoted History pin — `R-HIS-004` is satisfied (the door funnels through commit; validation is not bypassed), reads stay read-only, and no exported surface changes except AD-6's one route.

### AD-10 (new) — The on-demand door is a minimal run bracket, and equivalence is scoped to the compaction bracket

**Choice**: `Harness.Compact(ctx, req CompactionRequest, sink chan<- *Event)` (exact signature at spec/tasks) wraps `runCompaction` in its own minimal, independently `CheckStream`-valid stream: `run_start` → [compaction bracket per AD-3] → `cost_session(Final)` (every run close emits the cumulative — `R-CST-005/006`, and the compaction's own spend is the cumulative here) → `run_end(Completed)` (or `Failed` per existing run rules if the operation errored — final wording at spec). AG-18.5's equivalence scenario compares the **compaction turn bracket sub-sequence** (`turn_start` … `turn_end`) of the two streams, `Kind()`-for-`Kind()`, excluding fresh `RunID`/`TurnID` (the `context_strategy_test.go:222-224` exclusion) — the maximal unit one path produces in both modes, since the strategy-triggered stream necessarily also carries its prompt turns. `sdd-spec` records this scoping explicitly so the scenario is falsifiable as written.
**Rejected**: emitting a bare compaction bracket with no run bracket — `CheckStream` rejects any stream without a complete run bracket (`stream_check.go:178-183`, design AD-1 divergence 2, verified); comparing whole streams — structurally impossible to make equal, invites a vacuous or rigged fixture.

### AD-11 (settles atomicity, D3's second half) — Atomic-or-absent by ordering, precisely

**Choice**: build-then-commit. The commit (`ReplacePrefix`) is reachable **only** through this sequence: (1) cut resolved and mark-aligned; (2) the provider call returned an `ai.Completion` with `FinishReasonStop`; (3) the full replacement message sequence (summary + protected tail) passed the same pairing rules `NewSeededHistory` (`history.go:216-224`) enforces, on a scratch value. An interruption — `runCtx` cancellation, provider error, non-Stop finish — surfaces as an error from step (2) or a rejection at (3), and both branch to the failed arm **before any statement that touches `h.History`'s pointee**. There is no partial state to roll back because no write precedes the single commit call, and the commit op itself is all-or-nothing (AD-6). No journal, no snapshot, no new cancellation primitive (charter: compaction participates in the existing tree). A run-level `Interrupt` during compaction closes the bracket `Aborted`, and the run then winds down at the next iteration-boundary cause check (`harness.go:502`) — `windDownRun` is **never** entered from inside compaction, preserving `R-CAN-002`'s closed order. AG-18.4's "the next turn proceeds" fixture uses a compaction-local failure (failing compaction provider), not a run-level interrupt — the exploration's reading, confirmed.

### AD-12 (new) — `reconstruction_test.go` is NOT touched; the `R-LSK-004` release is exactly two files

**Choice**: AG-18.3's reconstruction assertion lives in a **new** test file; `reconstruction_test.go` stays byte-unchanged, narrowing the release the proposal's A6 left open ("may touch the third"). The recorded release, in the AG-11/AG-14/AG-16 form for the `agent-loop-skeleton` delta:

> **AG-18's release scope — `doc.go` and `doc_contract_guard_test.go` only, and the reason is structural rather than convenient.** Both are released **for AG-18 only**. The `L2C-07` row's text is declared in `doc.go` and mirrored byte-for-byte in `expectedLayer2ContractRows` (`doc_contract_guard_test.go:62-71`), and `R-AGP-002`'s closed-amendment rule requires the row and its table entry to land in the same pull request (`R-HIS-009`). AG-18 falsifies two of that row's clauses independently — the closed three-route enumeration ("append, seeded construction, orphan synthesis"; AD-6 adds a fourth) and "stable ordinal entry identity" (AD-8 scopes stability to a transcript generation) — and row text is unrelocatable: it is the guarded artifact itself. AG-18's edits to these two files MUST be confined to the `L2C-07` row's two clauses, byte-in-sync between the two files. `event.go`, `event_descriptor.go`, `stream_check.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `failure.go`, `compaction_events.go`, `cost_events.go`, `event_registry_test.go`, `reconstruction_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `go.mod` and `go.sum` all stay byte-unchanged — AG-18 registers **no** `EventKind` (AG-06 minted the compaction family; AG-18 is its first caller, closing the AG-17 annotation's forecast), adds no `TurnOutcome` and no `CostLabel`, and every file under `backend/agent/src/ai/` is untouched. This release does not extend to any milestone after AG-18 without its own recorded delta.

## Relaxation ledger (decision 8's table — every relaxed MUST names its replacement)

| Relaxed | Replacement invariant |
|---|---|
| `R-HIS-001:52` "MUST NOT support removal" | Exactly one removal shape exists: a **prefix** replacement, reachable only through the commit primitive of `R-HIS-004`; the replaced prefix MUST end at a recorded turn-mark boundary and be pairing-closed; the post-replacement transcript MUST satisfy the same append-time pairing rules `NewSeededHistory` enforces. Reordering and in-place mutation of a surviving entry remain MUST NOT on every route. Protected tail entries MUST compare value-identical in Layer 1 values and origins (AD-8). |
| `R-HIS-005:106` "Identity MUST be stable" | Stability scoped to a transcript generation; a compaction begins a new generation; identities remain ordinal-derived and caller-unmintable; `TurnID` is the durable cross-generation handle. |
| `R-CTX-003:71` verdict unconstructible | The never-compact guarantee attaches to the **zero value**: a zero/nil-`Compaction` verdict requests nothing on any path. `R-CTX-004:95`'s "opens no new history route and takes no write path" re-scoped to every no-request path, re-proved by the existing byte-identical two-run comparison. |
| `R-CST-001:46-54` per-path table completeness | Additive row for the compaction bracket (AD-3); the iff sentence unchanged — nothing relaxed, so nothing owed. |
| `L2C-07` row (both clauses) | Route enumeration re-closed at four (compaction named); "stable ordinal entry identity" becomes generation-stable; `S-HIS-031`'s closure bite re-proved over the widened enumeration. `S-HIS-080:182`'s stale "7 rows" re-scoped **off the literal count** (the repo's count-assertion drift class) in the same delta. |
| `agent-run-driver:351` forbidden row | Back-annotated, not silently breached: AG-18 adds the one exported route the AG-17 annotation forecast, under this change's recorded delta. No new `EventKind`, no new `TurnOutcome` — those two-thirds of the row stay true and are restated. |

## Data Flow — the compaction operation (both doors)

```
Harness.Run outer loop (no turn open; prev CloseTurn done, harness.go:668)   |   Harness.Compact (no run in flight, AD-4 gate)
  transcript := transcriptFromHistory(hist)                    :512          |     mint RunID; emit run_start
  verdict := h.ContextStrategy.Resolve(...)                    :524-530      |
  verdict.Compaction == nil ──────────────────────► next Turn (unchanged)    |
  else: runCompaction(runCtx, hist, req, runID, stamper, sink, total)  ◄─────┘  (same operation, both doors)
    ├─ emit turn_start (fresh compaction TurnID)          [bracket opens — PlacementTurn satisfiable from here]
    ├─ emit compaction_started(compactionID)
    ├─ resolve cut: greatest turn mark ≤ req.Cut; pairing verified (AD-1) ── error ──────────────┐
    ├─ provider call: req.Provider + req.Instruction, on runCtx ───────────── error/cancelled ───┤   ◄── INTERRUPTION ESCAPE:
    ├─ build replacement [summary | protected tail] in memory;                                   │       every failure branches here
    │  scratch pairing validation (NewSeededHistory rules)  ── invalid / non-Stop finish ────────┤       BEFORE the commit point;
    ├─ emit cost_turn(Final, completion usage) + total.add(ct)   (iff a completion exists)       │       h.History untouched by
    ├─ hist.ReplacePrefix(cut, summaryMsg)   ◄════ THE COMMIT POINT (single all-or-nothing op)   │       construction
    ├─ emit compaction_finished(Span from marks, SummaryID = summary message ID)                 │
    └─ emit turn_end(TurnOutcomeFinished)                 [bracket closes]                       │
    failed arm ◄─────────────────────────────────────────────────────────────────────────────────┘
    ├─ (cost_turn already emitted iff a completion existed)
    ├─ emit compaction_failed(typed *Failure)                    — never windDownRun (R-CAN-002 untouched)
    └─ emit turn_end(Aborted+failure | mapped non-aborted outcome iff a completion existed)  — R-CST-001 iff holds (AD-3)
  rebuild transcript from hist; continue to next Turn            |   emit cost_session(Final); emit run_end
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/compaction.go` | **Create** | `runCompaction`, cut resolution (AD-1), Span derivation (AD-9), bracket + three event call sites (AD-3), `Harness.Compact` + refusal gate (AD-4, AD-10), ID mints |
| `backend/agent/src/agent/history.go` | Modify | `EntryOriginSummarized` (AD-2); `ReplacePrefix` commit op (AD-6); turn marks + package-private `closeTurnMarked` (AD-9); mark rebuild on replacement |
| `backend/agent/src/agent/context_strategy.go` | Modify | `CompactionRequest`; `ContextVerdict.Compaction *CompactionRequest` (AD-7) |
| `backend/agent/src/agent/harness.go` | Modify | Capture verdict at `:524-530`; invoke `runCompaction` in the gap; rebuild transcript after success; capture successful attempt's `TurnID` and marked close at `:668` |
| `backend/agent/src/agent/doc.go`, `doc_contract_guard_test.go` | Modify (**released**, AD-12) | `L2C-07`'s two clauses, byte-in-sync, same PR |
| `backend/agent/src/agent/history_surface_guard_test.go` | Modify | Enumeration gains `ReplacePrefix` in the same commit as the route |
| Substrate byte-unchanged filters (exact files located by `sdd-tasks` — the AG-11…AG-17 widening discipline) | Modify | Exact-filename widening, byte-in-sync |
| New `package agent_test` files (compaction call/surgery/stream/recovery/on-demand + reconstruction) | **Create** | Five leaves + four bites; `reconstruction_test.go` untouched (AD-12) |
| `backend/agent/src/agenttest` | Modify/Create | Mis-aligned-cut fixture helper; second scripted provider is an existing-`NewProvider` instance, no new type |
| `openspec/changes/.../specs/` | Create | New `agent-compaction` spec (`R-CMP-`/`S-CMP-`) + deltas: `agent-history` (A1–A4), `agent-context-strategy` (A5), `agent-loop-skeleton` (A6/AD-12), `agent-cost-events` (AD-3's additive row), back-annotations `agent-protocol-events:154`, `agent-run-driver:342/:351`, `agent-v1-scope` (R-11/G3) |
| `docs/architecture/milestones/0003-…md` | Modify | AG-18 status, checklist, counter 18/24 |
| `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `compaction_events.go`, `cost_events.go`, `cost_usage.go`, `reconstruction_test.go`, `loop.go`, `go.mod`, `go.sum`, all `src/ai/**` | **NOT touched** | No new kind/outcome/label; 25-kind pin passes; first-caller-only for the compaction family |

## Testing Strategy (per leaf — the RED assertion each charter scenario needs)

All in `package agent_test`, `-race -count=1`, `agenttest.Gate` for synchronization (no sleeps), `Provider.Requests()` for captured-bytes assertions.

| Leaf / scenario | RED assertion (fails before the mechanism exists) |
|---|---|
| 18.1 own call + spend | Distinct compaction provider instance proven called (its `Requests()` non-empty, `h.Provider`'s count unchanged); the run's `cost_session(Final)` figures include the compaction completion's scripted usage; run-level interrupt mid-compaction-call → bracket `Aborted`, harness survives (interrupt semantics), `windDownRun` order intact |
| 18.1 injection | Captured request carries exactly the injected instruction; assert **no runtime-authored bytes** over captured request content, not over comments |
| 18.2 no split pair | Fixture: naive `Cut` between a `ToolCall` and its `ToolResult`; assert the resolved cut retracted to the mark boundary below, both pair halves whole on the protected side; post-transcript passes `NewSeededHistory`'s rules |
| 18.2 protection + typed summary | AD-8's exact assertion shape, on a fixture whose prefix length ≥ 2 (ordinals provably shift); summary distinguished via `Origin()` only — content and `Failed()` never read (R-HIS-007 discipline) |
| 18.3 stream | `CheckStream` accepts the recorded run unmodified; a reconstruction given only `Span()`+`SummaryID()` names the replaced TurnIDs against the stream's own earlier brackets and locates the summary entry by message ID — new test file |
| 18.4 atomic-or-absent | Failing compaction provider (Gate-held then errored): `hist.Entries()` byte-identical pre/post attempt, `compaction_failed` (never `finished`), next turn proceeds on uncompacted transcript, `windDownRun` never entered |
| 18.5 one path | AD-10's bracket sub-sequence `Kind()`-for-`Kind()` equality, excluding fresh `RunID`/`TurnID` |
| 18.5 typed refusal | Gate-blocked mid-turn run + concurrent `Compact` → typed sentinel via `errors.Is`, zero events emitted on the compact sink, in-flight turn completes unaffected |
| Bites (RED before GREEN, all four) | (a) skip mark-alignment/pairing verification → split-pair scenario FAILS; (b) move the commit before the provider call returns → 18.4 FAILS under a failed call; (c) drop `ReplacePrefix` from `S-HIS-030`'s enumeration → closure guard FAILS naming it; (d) queue the on-demand demand instead of refusing → typed-refusal scenario FAILS |
| New substrate needed | Mis-aligned-cut transcript builder; a marks-aware fixture (turns driven through real runs so marks exist); verdict-construction helper (possible only once AD-7 lands) |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Pure in-process library change under `backend/agent`.

## Migration / Rollout

No migration. Nothing persists across processes; single-revert rollback per the proposal. The one irreversible runtime effect (compacted entries discarded from memory) is a named spec consequence, not a migration concern (proposal Q1's recorded answer).

## Review-workload forecast

Inherits the proposal's counting rule (excludes `openspec/`) and delivery: `Decision needed before apply: No` · `Chained PRs recommended: No` · `400-line budget risk: High`. AD-9 adds marks work to `history.go`/`harness.go` (~+80–150 counted lines over the proposal's estimate); still inside the reserve-slice threshold.

## Open Questions (none blocking)

- [ ] Exact `Options` type for `CompactionRequest` (reuse `TurnOptions` vs. a narrowed subset) — `sdd-spec` decides after reading `TurnOptions`; the carrier is settled (AD-7).
- [ ] `Compact` after `Shutdown`: recommend refusal under the existing terminal-shutdown semantics; `sdd-spec` words it.
- [ ] The exact substrate-filter file names (the byte-unchanged guard tests widened at AG-11…AG-17) — `sdd-tasks` locates them by grep before apply; failing to widen them fails the suite loudly, so the risk is a blocked apply, not silent drift.
