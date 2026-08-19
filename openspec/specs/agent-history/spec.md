# Spec — History and the pairing invariant (`agent-history`)

> **Change**: `cachicamas-agent-history` · **AG-12** (Layer 2, Wave 3, milestone 12 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-12--implement-history-and-the-pairing-invariant), `0003:1227-1292`
> **Nodes**: AG-12.1 `[leaf]` (append + boundary validation, `0003:1247-1274`) · AG-12.2 `[leaf]` (orphan synthesis, `0003:1276-1292`)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test`.
> **Identifier convention**: requirements `R-HIS-0NN`, scenarios `S-HIS-0NN` (bites carry the same `S-HIS-` prefix and are marked **(bite)**). Append-only. Distinct from `R-AEV-`/`R-AGE-`/`R-AGP-`/`R-AGM-`/`R-AGV-`/`R-ATT-`/`R-TLS-`/`R-APE-`/`R-PRH-`/`R-LSK-`/`R-AMT-`; the `HIS` prefix was verified collision-free repo-wide before minting.
> **Evidence gate**: `cd backend/agent && make test` (`go test -race -v ./...`), plus `make lint`, `make build` and `make vuln-check`. No CI exists.
> **Authoring constraint**: this spec states obligations and observable behavior. It names **no** Layer 2 Go type, field or function; `sdd-design` owns the Go shape (doc 0003's authoring constraint). Layer 1 identifiers already shipped (`ai.Message`, `ai.ToolCall`, `ai.ToolResult`, `ai.Violation`, `ai.ErrUnresolvedReference`, `ai.ErrEmpty`) are cited as consumed contracts and alignment targets, never as Layer 2's own vocabulary.
> **Inherited decisions (CLOSED, not re-openable here)**: the `L2C-07` doc-contract row lands (`proposal.md` decision 1); typed pairing rejections reuse `ai`'s existing violation machinery and existing rule classes, and `ai/validation.go` MUST NOT be extended under this change (decision 2); the seeded-construction obligations are frozen (decision 3); synthesized-vs-real distinguishability rides an opaque entry envelope carrying an origin discriminator, never `ToolResult.Failed()` and never a content sentinel (decision 4). From `sdd-design`: a seed MAY end with open calls, and only an unresolved **result** rejects at seed time (`design.md:214`) — carried below as the open-call reconciliation.

## Coverage

| Charter leaf | Requirements | Spec scenarios | Bites |
|---|---|---|---|
| AG-12.1 (`0003:1247-1274`) | `R-HIS-001`…`006` | 16 | 2 |
| AG-12.2 (`0003:1276-1292`) | `R-HIS-007`…`008` | 5 | 0 |
| Cross-cut (doc contract) | `R-HIS-009` | 1 | 1 |

Charter → spec: `0003:1252-1256` → `R-HIS-001`/`002`/`003`; `0003:1258-1261` → `R-HIS-004`; `0003:1263-1266` → `R-HIS-005`; `0003:1268-1271` → `R-HIS-006`; `0003:1281-1284` → `R-HIS-007`; `0003:1286-1289` → `R-HIS-008`.

## Purpose

The harness has no transcript. Layer 1 declines the pairing job on purpose — `ai/tool_result.go:135-136` states verbatim that "the invariant that every call in a transcript has a matching result is V-OUT-02", Layer 2's job. Today any caller can assemble a sequence containing a tool call with no result and hand it to a provider: Layer 1 validates each part and each message, and nothing validates the sequence.

AG-12 introduces the transcript store, enforces the pairing invariant **at the boundary and not at call sites** (`VL2-COR-11`), and repairs an interrupted transcript by synthesizing results for orphaned calls. It wires into nothing: `loop.go` and `scheduler.go` stay byte-unchanged; the run driver's consumption of this surface is AG-13's.

## Reconciliation — the entry envelope and "history exposes Layer 1 values"

Two charter scenarios read as if they conflict, and this spec resolves the tension explicitly rather than leaving it to implementation.

`0003:1263-1266` requires that "the transcript arrives as Layer 1 values with stable entry identity". A bare `ai.Message` cannot carry that identity — its `MessageID` belongs to Layer 1 and history does not mint it — so an **envelope** is required by that scenario alone, before AG-12.2 is considered. `0003:1281-1284` then requires a synthesized result to be "distinguishable from a real result"; `ai.ToolResult` is frozen and its content is stored byte-for-byte, so a content sentinel is forgeable by any real tool and is rejected outright.

The single envelope satisfies both: it carries the **unmodified** Layer 1 value, the entry identity, and an origin discriminator. "History exposes Layer 1 values" holds because read-back yields the Layer 1 value unmodified and unaliased; the discriminator rides the envelope that stable entry identity already forced into existence.

## Reconciliation — an open call is not an orphaned result

The pairing invariant has two directions, and this spec keeps them apart everywhere it is stated. Collapsing them would make AG-12.2 unreachable, so the distinction is normative, not editorial.

- An **orphaned result** — a tool-result entry naming a call the transcript does not declare — is rejected **at commit time**, through every route including seeded construction (`R-HIS-002`). It can never enter a transcript.
- An **open call** — a tool call issued with no result yet — is **legal while the turn is open**, through every route including seeded construction. It is rejected only when the turn is closed (`R-HIS-003`), and that rejection is identical on a seeded and on an appended history.

A seed MUST therefore be accepted when it ends in one or more open calls. AG-12.2's charter scenario is a turn "interrupted after calls were issued but before results arrived", repaired "before the next turn" (`0003:1281-1284`), and seeded construction is named the seam session resume stands on (`0003:1271`). If a seed carrying open calls were rejected, an interrupted transcript could never be reconstructed on resume and orphan synthesis would be unreachable through the exact path it exists to serve — AG-12.2 would be dead code.

"Seeded construction validates like appends do" (`0003:1268`) therefore means the seed re-runs the **append-time** rules over every entry in order. It does **not** mean the seed is additionally turn-closed. Wherever this spec says a seed with an "unmatched pair" is rejected, it means an orphaned result and nothing else.

(Forwarded from `design.md:214` as a normative-prose obligation, extending proposal risk 4, so no later reader can re-derive the stricter reading.)

## Requirements

### R-HIS-001 — The transcript is ordered, append-only within a run, and read back in order — with exactly one removal route, opened by AG-18

History MUST hold an ordered sequence of Layer 1 values across a run. Entries MUST be appended in conversation order and MUST be read back in that same order.

Within a run, history MUST NOT support **reordering** or **in-place mutation** of an already-committed entry, on **any** route, compaction's included. Those two prohibitions are unchanged and unconditional.

**Removal is admitted in exactly one shape and nowhere else.** Beside the append of `R-HIS-004`, the transcript admits one further extension: a **prefix replacement**, in which a contiguous prefix `[0, cut)` is replaced by a single summary entry. It is subject to every one of the following, each of which is the replacement for what the old blanket prohibition provided:

1. **It funnels through the same single validating commit primitive of `R-HIS-004`.** It is not a second door, and no privileged internal bypass is created.
2. **It is prefix-shaped, not span-shaped.** Only a prefix beginning at index 0 may be replaced. A mid-transcript span replacement MUST NOT exist, because no consumer needs one and it would relax this requirement further than the change requires.
3. **Its boundary MUST be pairing-closed.** The replaced prefix MUST NOT split a tool-call/tool-result pair in either direction, and the boundary MUST coincide with a recorded turn-mark boundary.
4. **The post-replacement transcript MUST satisfy the same append-time pairing rules seeded construction enforces (`R-HIS-006`).** The invariant is not weakened; it is re-proved over a shorter transcript.
5. **Entries at or after the boundary MUST survive value-identical** — their Layer 1 values and their origin discriminators MUST compare equal before and after, positionally. Their entry **identities** do not survive, by `R-HIS-005`'s own ordinal rule; see that requirement.
6. **Nothing may be permuted, edited or partially rewritten.** The replacement replaces a prefix and touches nothing else.
7. **A failed replacement is not a partial replacement.** On any validation failure the transcript MUST be left byte-identical to its pre-attempt state (`R-HIS-002`).

Nothing outside this shape is admitted. An ordinary caller's ability to remove, reorder or rewrite an entry is **unchanged — still forbidden**.

A read MUST NOT alias internal storage: mutating anything a read returns MUST NOT be observable in a subsequent read.

(Previously: "Within a run, history MUST NOT support removal, reordering, or in-place mutation of an already-committed entry; the only admitted extension is an append through the commit path of `R-HIS-004`." AG-18.2 replaces a prefix of entries with one summary entry, which is removal, so the sentence and the shipped code would otherwise have contradicted each other in the same pull request.)

#### Scenarios

- **S-HIS-001** — Given a sequence of Layer 1 messages appended in conversation order through the public append route, when the transcript is read back, then the read-back sequence is equal to the appended sequence, element-by-element and in the same order, with no insertion, omission or reordering.
- **S-HIS-002** — Given a populated history, when a caller in an external test package takes a read-back view and mutates the value it holds (including any slice it exposes), then a subsequent read of the same history returns the original values unchanged — no read aliases internal storage.
- **S-HIS-100** — **AG-18: the one removal route, fenced.** Given a populated history whose prefix `[0, cut)` is pairing-closed and mark-aligned, when the prefix replacement commits, then the read-back holds one summary entry at position 0 followed by exactly the entries formerly at indices ≥ `cut` in their original relative order; when a fresh history is constructed over that read-back through seeded construction, then construction **succeeds**; and when the same replacement is attempted with a boundary that splits a pair, then it is rejected typed and a subsequent read returns the transcript byte-identical to its pre-attempt state.
- **S-HIS-101** — **AG-18: nothing else became removable.** Given the exported surface of the transcript store after this change, when it is enumerated from an external test package, then the only route that removes a committed entry is the prefix replacement; no route removes a mid-transcript span, no route reorders entries, no route rewrites a surviving entry in place, and each such attempt is either absent from the surface or rejected typed.

### R-HIS-002 — Appending an orphaned tool result is rejected typed

Appending a tool-result entry whose call identity has no prior tool call in the transcript MUST be rejected with an `ai.Violation` whose rule class is `ai.ErrUnresolvedReference` (`ai/validation.go:93-96` — "a value naming something the request does not declare"), positioned so the violation names the offending entry. The rejected append MUST leave the transcript **unchanged**: a failed commit is not a partial commit.

Correlation MUST be by the Layer 1 identity pair `ai.ToolCall.ID()` / `ai.ToolResult.CallID()` and by nothing else. No assertion in this capability's tests may distinguish an admitted result from a rejected one by inspecting result content.

This requirement covers the **result** direction only. A tool call that has no result yet is not an orphan and MUST NOT be rejected here; that direction belongs exclusively to `R-HIS-003`, at turn close.

#### Scenarios

- **S-HIS-010** — Given a history holding no tool call with identity `c`, when a tool-result entry whose call identity is `c` is appended through any public route, then the append fails with an `ai.Violation` matching `ai.ErrUnresolvedReference` at a position naming the offending entry, and a subsequent read returns the transcript byte-identical to its state before the attempt.
- **S-HIS-011** — Given a history holding a tool call with identity `c` and no result for it, when a tool-result entry whose call identity is `c` is appended, then the append succeeds and the result is read back after the call, in order.
- **S-HIS-012** — **(bite)** Given the correlation check weakened to admit any tool result once at least one call exists (identity ignored), when `S-HIS-010` runs, then it FAILS reporting the accepted orphan — proving the identity comparison, not mere presence of a call, is what rejects. RED-recorded BEFORE `S-HIS-010` is GREEN.

### R-HIS-003 — A call still unclosed at turn close is rejected typed

Committing a state in which a tool call carries no matching result **once the turn closes** MUST be rejected with an `ai.Violation` whose rule class is `ai.ErrEmpty`, positioned at the missing result's slot — the required value is absent, the position carries the specificity and the class carries the kind. `sdd-design` MAY substitute a different **existing** `ai` rule class if it finds a better fit; it MUST NOT extend `ai/validation.go` under this change.

A turn close over a transcript in which every call has a matching result MUST succeed. Detecting or causing the interruption that produces an unclosed call is **not** this capability's (`AG-14`); this requirement covers only the rejection at the boundary.

An open call is legal until the turn closes: it MUST NOT be rejected at append time and MUST NOT be rejected at seed time (see the open-call reconciliation above). This requirement is the **only** place the unclosed-call rejection lives, and it applies identically to a seeded and to an appended history.

#### Scenarios

- **S-HIS-020** — Given a history in which a tool call with identity `c` has no matching result, when the turn is closed through the public route, then it fails with an `ai.Violation` matching the declared rule class, positioned at the missing result slot for `c`, and the transcript is left unchanged.
- **S-HIS-021** — Given a history in which every tool call has a matching result, when the turn is closed, then it succeeds and the transcript reads back unchanged and in order.
- **S-HIS-022** — Given a history in which two distinct tool calls are both unclosed, when the turn is closed, then the reported violation names the position of the **first** offending call in transcript order, so a caller can act on a deterministic position rather than an arbitrary one.

### R-HIS-004 — Exactly one commit path, with no privileged bypass for internal callers

Every route that can extend or mutate the transcript — append, seeded construction, orphan synthesis and **prefix replacement** alike — MUST funnel through a single validating commit primitive. The package MUST NOT expose, and MUST NOT internally retain, a second route that reaches committed storage without that validation. This is `VL2-HAR-01`'s "exactly one commit path, with no privileged bypass for internal callers" and the C1 lesson (`0001:376`) reapplied to history.

**AG-18 adds the fourth route and satisfies this requirement rather than relaxing it.** The prefix replacement of `R-HIS-001` is dispatched from the same primitive — verified in this worktree at `history.go:327-332`, whose own comment reads "commit is the ONLY function that writes h.entries or h.open (R-HIS-004)" — so that sentence stays literally true after the change. The **turn marks** of `R-CMP-012` are written through the same primitive by the same rule, via a package-private door beside the exported turn-close route; a package-private validated door is not a bypass, because it is unreachable from outside the package and it does not skip validation.

The obligation MUST be discharged **by enumeration in test, not by assertion in prose**: the external test surface MUST enumerate every public route that can extend or mutate the transcript, drive an orphaning sequence through each, and assert rejection for each. The enumeration MUST be closed — a public mutating route that the enumeration does not name MUST be observable as a failure of the suite, never as an unaudited door. **The enumeration MUST gain the prefix-replacement route in the same commit that introduces the route**, and the closed exported-surface guard MUST be updated deliberately rather than incidentally: before this change the exported surface is exactly `Append`, `CloseTurn`, `SynthesizeOrphans`, `Entries` and `Len` (verified against `history.go` in this worktree), and the replacement route is the sixth member.

Because the tests must exercise only what a caller outside the package can reach, they MUST live in an external test package (`package agent_test`): an in-package test could reach the private primitive directly and would prove nothing about the boundary.

(Previously: the route enumeration named three routes — append, seeded construction and orphan synthesis — and the requirement made no statement about a fourth or about the exported-surface guard's committed member set.)

#### Scenarios

- **S-HIS-030** — Given the enumerated set of every public route that can extend or mutate the transcript — append, seeded construction, orphan synthesis **and prefix replacement** — when an external test constructs an orphaning sequence through each route in turn, then every route rejects it with the rule class `R-HIS-002` or `R-HIS-003` assigns to that shape, and no route accepts it.
- **S-HIS-031** — **(bite)** Given a scratch edit that adds a public mutating route reaching committed storage without the validating primitive, when the closed route enumeration of `S-HIS-030` runs, then it FAILS naming the unenumerated route — proving the audit is closed rather than a snapshot of the routes that happened to exist. Recorded, then reverted. RED-recordable.
- **S-HIS-102** — **(bite) AG-18: the fourth route must be named.** Given a scratch tree in which the prefix-replacement route is dropped from the enumeration of `S-HIS-030` while the route itself remains exported, when the closure guard runs, then it FAILS naming the unenumerated route — proving the enumeration closed over the widened surface rather than over the surface it happened to have at AG-12. RED-recorded BEFORE the route's own scenarios are GREEN, then reverted.

### R-HIS-005 — Read-only views for the loop and for a future session, with ordinal entry identity stable within a transcript generation

History MUST expose read-only views serving two consumers: the loop, which reads the transcript, and a future session, which reads entry identities. The transcript MUST arrive as **unmodified Layer 1 values** — the same `ai.Message` / `ai.ToolResult` values that were committed, neither re-serialised nor rewritten — carried in an opaque entry envelope with unexported fields and read-only accessors, per the reconciliation above.

Entry identity MUST be **derived deterministically from ordinal position** in the transcript. No caller may supply, mint, or overwrite an entry identity — that would be the C1 back door. **That prohibition is unchanged and unconditional**, and it is what forces the amendment below rather than permitting it.

**Stability is scoped to a transcript generation.** Identity MUST be stable within a generation: the same transcript yields the same identities on every read and across processes. A **compaction begins a new generation**, and identities are re-derived over the new sequence. This is not a concession: preserving pre-compaction identities across a shortened transcript would require minting non-ordinal identities, which the paragraph above forbids outright — so renumbering is *forced by this requirement's own rule* rather than chosen by AG-18.

**The replacement for cross-generation stability is named rather than left missing**: the durable handle across a compaction is the **turn identifier**, not the entry identity. That is why the compaction span was built out of turn identifiers twelve milestones before compaction existed (`R-APE-007`). A consumer MUST NOT store an entry identity across a compaction.

**A consequence a reader would otherwise infer wrongly from the charter's word "byte-identical"**: protected entries are byte-identical in their **Layer 1 values and origin discriminators**, and are **NOT** identical in their entry identities. Any assertion written as whole-entry structural equality including identity encodes a false claim — it either fails spuriously or, on a fixture that happens not to shift ordinals, passes while asserting something this requirement makes impossible. Such an assertion MUST NOT be written (`R-CMP-006`).

No read may return a handle through which committed state can be changed.

(Previously: "Identity MUST be stable: the same transcript yields the same identities on every read and across processes" — stated with no generation qualifier, so a shortened transcript would have falsified it, and with no statement of what survives a compaction in its place.)

#### Scenarios

- **S-HIS-040** — Given a populated history, when the loop-facing view is read from an external test package, then it yields the committed Layer 1 values unmodified and in order, and each value is equal to the one committed without re-serialisation or field rewriting.
- **S-HIS-041** — Given a populated history, when the same transcript is read twice, and when a second history is constructed over the same ordered seed, then each entry's identity is equal across both reads and across both histories — identity is a function of ordinal position alone.
- **S-HIS-042** — Given the public surface of the entry envelope, when it is enumerated from an external test package, then it exposes no route to set or replace an entry identity, an origin discriminator, or a committed Layer 1 value.
- **S-HIS-103** — **AG-18: the generation boundary, asserted in both directions.** Given a history whose replaced prefix holds at least two entries and whose protected tail holds at least two entries, when a compaction commits, then every protected entry's Layer 1 value and origin discriminator compare **equal** to their pre-compaction values positionally; at least one protected entry's identity compares **different**; the post-compaction identities are exactly the ordinal-derived identities of the new sequence; and no assertion in this scenario compares whole entry values structurally.

### R-HIS-006 — Seeded construction validates by the same rule as appends

Construction over a pre-existing transcript MUST re-run the **same** validation the append path runs — the **append-time** rules of `R-HIS-002`, entry by entry in seed order — over the whole seed. Its obligations are frozen at this handoff (`proposal.md` decision 3) because session resume and next-run model switching stand on this seam:

- It MUST accept an ordered sequence of Layer 1 messages **and nothing else**. It MUST NOT accept a caller-supplied entry identity or origin discriminator.
- It MUST return a constructed history **and an error**. A zero-valued history MUST NOT be usable as a history.
- It MUST reject a seed containing an **orphaned result** — a tool-result entry naming a call the seed does not declare at that point — with the rule class the equivalent append produces.
- It MUST **accept** a seed that ends in one or more **open calls**. Seeded construction MUST NOT apply the turn-close rule of `R-HIS-003`; a seeded history is constructed with its turn open, exactly as an appended one is. This is what makes an interrupted transcript reconstructable on resume, and it is the precondition orphan synthesis (`R-HIS-007`) depends on.
- On rejection it MUST report the **position of the first offending entry**, so a resume can name what broke.
- Identities over an accepted seed MUST be the ordinal-derived identities of `R-HIS-005`.

#### Scenarios

- **S-HIS-050** — Given a pre-existing transcript in which every tool call has a matching result, when a history is constructed over it, then construction succeeds, the transcript reads back equal to the seed and in order, and entry identities are the ordinal-derived identities.
- **S-HIS-051** — Given a pre-existing transcript containing an **orphaned result** — a tool-result entry whose call identity has no prior tool call in the seed — when a history is constructed over it, then construction fails with the same `ai.Violation` rule class the equivalent append would produce (`ai.ErrUnresolvedReference`, `R-HIS-002`), and the violation's position names the **first** offending entry in seed order. This scenario asserts the result direction only; it makes no claim about a seed that merely ends in an open call.
- **S-HIS-052** — Given a rejected seeded construction, when the returned history value is used through any public read or append route, then it is not usable as a history — the zero value never behaves as an empty valid transcript.
- **S-HIS-053** — Given the seeded constructor's public signature, when it is inspected from an external test package, then it accepts an ordered sequence of Layer 1 messages and no other caller-supplied input, and offers no parameter through which an entry identity or an origin discriminator could be provided.
- **S-HIS-054** — Given a pre-existing transcript that ends in one or more **open calls** — tool calls issued with no result, the shape an interrupted turn leaves — when a history is constructed over it, then construction **succeeds**, the transcript reads back equal to the seed and in order; and when the turn is then closed on that seeded history, then it fails with the same rule class and the same positional shape the appended path produces in `S-HIS-020` — the unclosed-call rejection is `R-HIS-003`'s alone, on seeded and appended histories alike.

### R-HIS-007 — Orphan synthesis closes every orphaned pair with an interruption artifact

Synthesis MUST complete **every** orphaned tool call in the transcript with a result recorded as an interruption artifact. The artifact MUST be distinguishable from a real result **by the entry envelope's origin discriminator** (`appended` | `synthesized`), read through a read-only accessor.

Distinguishability MUST NOT depend on result content and MUST NOT depend on `ai.ToolResult.Failed()`. A synthesized result MAY set `Failed()`, but so does a real failing tool, so `Failed()` is not the discriminator; and a real tool can emit any content bytes, so a content sentinel is forgeable and is prohibited. No test in this capability may distinguish origin by inspecting content or `Failed()`.

Synthesis MUST route through the commit path of `R-HIS-004`; it is not a second door.

**AG-14 back-annotation: synthesis now has a production caller.** The harness's cancellation wind-down invokes it as its first step, before closing the turn and before emitting the run-close (`R-CAN-002`). Three consequences, recorded so no reader has to re-derive them:

1. **No behavior of this requirement changes.** The production caller uses the existing exported surface; `history.go` is byte-unchanged and no exported route is added.
2. **Idempotency is load-bearing, not merely nice.** `R-HIS-008` is what makes the wind-down safe to run on every cancellation path regardless of how far the turn got: a turn whose results were already committed synthesizes nothing, so the wind-down needs no special case for "were we mid-turn or between turns".
3. **The interruption this artifact was designed for is now real.** From AG-14 onward, a transcript carrying `synthesized`-origin entries is the ordinary observable outcome of an interrupted or shut-down run, not only a test fixture, and the run-driver carve-out of `R-RUN-011` is what routes to it.

(Previously: the requirement described synthesis with no statement of who calls it in production; the AG-12/AG-13 reading was that nothing did.)

#### Scenarios

- **S-HIS-060** — Given a transcript interrupted after tool calls were issued but before their results arrived, when synthesis runs, then every orphaned call has a matching result in the transcript, each such entry reports origin `synthesized`, and every pre-existing entry still reports origin `appended`.
- **S-HIS-061** — Given a synthesized result and a real appended result constructed with byte-identical content and with `Failed()` set identically, when each entry's origin is read, then the two are distinguished correctly, and the assertion reads neither content nor `Failed()`.
- **S-HIS-062** — Given a history over which synthesis has run, when the turn is closed, then the close succeeds — synthesis produced a transcript the pairing invariant of `R-HIS-003` accepts, through the same commit path.
- **S-HIS-097** — **AG-14: the production caller.** Given a harness-driven run whose transcript carries a tool call left **open** by an earlier turn, when an interrupt or shutdown signal fires and the run has returned, then the transcript read back through the existing surface holds a matching result for that open call, its entry reports origin `synthesized`, every entry committed before the signal still reports origin `appended`, and the turn is closed — no test invoked synthesis directly. Cross-referenced to `S-CAN-001` Arm A.

  **The Given says *open*, not *in flight*, and the distinction is load-bearing.** A call that reached the scheduler is always rejoined — the scheduler fills every ordinal slot, and the wind-down bound guarantees it does so even against a cancellation-deaf tool (`R-CAN-006`) — and `finishContinuationTurn` then commits those results synchronously (`loop.go:493-513`), so an in-flight call's entry origin is `appended`: it was resolved, not orphaned. Synthesis exists for calls that never obtained a result at all. A scenario worded "interrupted while a tool call is in flight" would demand an observation that path cannot produce, and could be made to pass only by breaking the rejoin.
- **S-HIS-098** — Given the AG-14 branch, when the suite runs and `git diff` is taken against the merge base, then `history.go` is **byte-unchanged**, `history_surface_guard_test.go` is byte-unchanged and passes, and its enumerated exported route set is equal in both directions — AG-14 added no route and removed none.

### R-HIS-008 — Synthesis is idempotent and total

Over a transcript holding N orphaned calls, synthesis MUST close **exactly N** pairs on the first application. A second application MUST change nothing: the transcript after the second application MUST be identical to the transcript after the first, entry-for-entry, including entry identities and origin discriminators.

Synthesis MUST NOT touch a call that already has a matching result, whether that result was appended or synthesized.

#### Scenarios

- **S-HIS-070** — Given a transcript containing N orphaned calls (with N ≥ 2) interleaved with already-matched calls, when synthesis is applied once, then exactly N new results are present, each matching a distinct orphaned call, and no already-matched call gained a second result.
- **S-HIS-071** — Given the transcript produced by the first application, when synthesis is applied a second time, then the read-back transcript is identical to the first-application transcript entry-for-entry — same order, same Layer 1 values, same entry identities, same origin discriminators — and no new entry was committed.

### R-HIS-009 — The `L2C-07` doc-contract row and its bidirectional guard

History is a **second upward surface** — a transcript read by the loop and by a future session — that no existing doc-contract row covers, and "exactly one commit path, no privileged bypass" is the machine-checkable, package-wide shape `L2C-01`..`L2C-04` are written in. The package documentation MUST therefore carry an `L2C-07` row stating that guarantee, and its `expectedLayer2ContractRows` entry MUST land in the **same pull request**, per `R-AGP-002`'s closed-amendment rule. The guardian is the same `doc_contract_guard_test.go` that guards `L2C-01`..`L2C-06`.

The per-behavior semantics of history belong in `doc.go` prose, not in the guarded row.

**AG-18 amends the `L2C-07` row, and BOTH of its independently falsifiable clauses must change together.** The row as shipped (`doc_contract_guard_test.go:69`, read verbatim in this worktree) asserts (a) "stable **ordinal entry identity**" and (b) a **closed three-route enumeration**, "every route that can extend it — append, seeded construction, orphan synthesis — funnels through one validating commit primitive". AG-18 falsifies each independently: `R-HIS-005` scopes identity stability to a transcript generation, and `R-HIS-004` admits a fourth route. **Amending only one leaves the row false**, so both MUST be amended, byte-in-sync between `doc.go` and `expectedLayer2ContractRows`, in the same pull request. The row's remaining clauses — the pairing invariant at the boundary, no privileged bypass, and origin-only distinguishability of a synthesized result — stay true and MUST be preserved.

*Criterion to overturn in `sdd-design` (`proposal.md` decision 1)*: if the guarantee turns out to be expressible only as behavior of one type rather than as a constraint on the package, the row is dropped and the drop is stated explicitly.

(Previously: the requirement recorded the row's introduction with no statement about amending it; a milestone widening the route enumeration or the identity clause had no written obligation to touch the other.)

#### Scenarios

- **S-HIS-080** — Given the `expectedLayer2ContractRows` table in `doc_contract_guard_test.go`, when it is read, then it contains the `L2C-07` row, present and in its committed position, with row text referencing history's single validated commit path as a package-wide guarantee, and the guard passes against `doc.go`. **The claim is deliberately scoped off the literal row count.** *(Correction of a PRE-EXISTING defect, not an AG-18 regression: this scenario previously asserted the table "contains 7 rows (`L2C-01`..`L2C-07`)". It was already false before this change — AG-14 appended `L2C-08`, and the shipped table carries **eight** rows at `origin/main@f6acc0d2`, re-verified in this worktree at `doc_contract_guard_test.go:62-71`. AG-17 recorded the drift and correctly declined to repair it, having not touched the table. AG-18 does touch the table, so it repairs it here rather than compounding it, and repairs it by re-scoping the claim off the count — this repository's count-assertion drift class — rather than by re-fixing the number, which would go false again on the next append.)*
- **S-HIS-081** — **(bite)** Given a scratch edit that appends an `L2C-07` row to `doc.go` without adding its entry to `expectedLayer2ContractRows`, when the doc-contract guard runs, then it FAILS naming the unexpected row — the closed-amendment rule is observed, not bypassed. Recorded, then reverted. RED-recordable.
- **S-HIS-104** — **AG-18: both clauses moved, and the guard proves they moved together.** Given the amended `L2C-07` row, when its text is read in `doc.go` and in `expectedLayer2ContractRows`, then the two are byte-identical to each other; the route clause names **four** routes including the prefix replacement; the identity clause states stability within a transcript generation rather than unqualified stability; the pairing-invariant, no-bypass and origin-distinguishability clauses are preserved; and the doc-contract guard passes. And given a scratch edit amending only the route clause, when this scenario runs, then it FAILS on the identity clause — the row cannot be half-amended.

### R-HIS-010 — On the continuation path, `Turn` commits the turn's own messages

When `Turn` runs with a non-nil continuation carrying a transcript store (`R-LSK-001`), it MUST commit that turn's own messages to that store before returning, and it MUST commit **only** those. Specifically, after finalize and after the rejoin:

1. It MUST append the turn's assistant message — including the turn's tool-call parts, with provider-exact bytes — so that a pure-tool turn yields a constructible assistant message whose calls the pairing invariant of `R-HIS-003` can match. A turn that produced no content at all MUST append nothing rather than appending an empty message.
2. It MUST then append **one tool-result message per rejoin result, in call order**, mapping each result's outcome onto the Layer 1 tool-result value that already carries it — the success form for a successful outcome and the failure form for both failure outcomes. This is the one-message-per-result shape orphan synthesis already established.
3. An append failure MUST be returned as a typed error from `Turn`. It MUST NOT be swallowed, and a turn whose append failed MUST NOT be reported to the caller as successful.
4. `Turn` MUST NOT call `CloseTurn`. Closing the turn is the run driver's, at the turn boundary (`R-RUN-005`).

**The appender split is normative**, because a message written twice and a message written nowhere are the same defect wearing different clothes:

| Message class | Sole appender |
|---|---|
| The run's initial prompt | the run driver, at run entry (`R-RUN-005`) |
| Each steered user message | the run driver, at a turn boundary (`R-RUN-008`) |
| The turn's assistant message | `Turn`, on the continuation path (this requirement) |
| Each tool-result message | `Turn`, on the continuation path (this requirement) |

Because `Turn` appends calls first and results second, the open-call set is empty by the time the driver closes the turn, so the boundary `CloseTurn` succeeds under `R-HIS-003` without any special case.

With a **nil** continuation, `Turn` MUST commit nothing anywhere: there is no transcript store to commit to, and every pre-AG-13 path stays byte-stable.

Consecutive same-role entries MUST remain legal. `commitAppendOp` carries no role-alternation check, so this requires no History change and none is made.

#### Scenarios

- **S-HIS-090** — Given a continuation carrying a transcript store and a provider scripted with a turn that emits assistant text and one tool call whose tool succeeds, when `Turn` runs, then the store holds, in order, the turn's assistant message carrying the tool call with provider-exact argument bytes, followed by one tool-result message correlated to that call; and when the caller then closes the turn, then the close succeeds because no call is open.
- **S-HIS-091** — Given a continuation and a provider scripted with a turn producing **no** content and no tool call, when `Turn` runs, then nothing is appended to the store, a subsequent read returns the store unchanged, and `Turn` returns without error.
- **S-HIS-092** — Given a continuation and a turn whose scheduled calls resolve to a mix of success and both failure outcomes, when `Turn` runs, then exactly one tool-result message is appended per rejoin result, in call order, each correlated to its own call identity, and each failure outcome is carried by the Layer 1 failure form rather than by a content sentinel.
- **S-HIS-093** — Given a continuation whose transcript store rejects the append, when `Turn` runs, then it returns a non-nil typed error carrying that rejection, and the caller's run terminates through the failure path of `R-RUN-011` rather than reporting a successful turn.
- **S-HIS-094** — Given a `TurnOptions` with a **nil** continuation, when `Turn` runs any scripted shape, then no transcript store is touched, and the closed-route history surface guard passes with its source byte-unchanged.

## Non-functional requirements

### NFR-HIS-001 — External-package verifiability

Every scenario above MUST be verifiable by `cd backend/agent && make test`. Every behavioral test MUST live in an external test package (`package agent_test`), so a behavior reachable only from inside the package is, for the purposes of this spec, not reachable at all. `R-HIS-004` in particular is meaningless from inside the package.

### NFR-HIS-002 — Determinism and race cleanliness

Every test added by this change MUST be deterministic and hermetic — no network, no filesystem outside `t.TempDir()`, no environment dependence — and MUST pass under `-race`, which `make test` applies to the whole module.

### NFR-HIS-003 — No Layer 1 edit, no new dependency; the AG-12 wiring freeze, and what AG-13 released

`loop.go` and `scheduler.go` MUST be byte-unchanged **under AG-12**: wiring history into `Turn`/`Schedule` is AG-13's. That was a freeze scoped to the milestone that wrote it — its own second clause names its successor — and it is **not** a standing prohibition on every later milestone.

**AG-13 did the wiring**, as forecast. It modifies `loop.go` (the continuation seam and the commits of `R-HIS-010`), `scheduler.go` (sink ownership) and `tool.go` (the sink-ownership flag), and it does so without touching `history.go` and without adding any exported `History` route. From AG-13 onward, this requirement's byte-unchanged claim covers `history.go` and Layer 1 only; a later milestone that wants to change `loop.go` or `scheduler.go` needs no release from this requirement, but does need whatever release `R-LSK-004` requires.

Every file under `backend/agent/src/ai/` MUST be byte-unchanged, including `ai/validation.go`. `go.mod` and `go.sum` MUST be unchanged. Layer 1 is consumed, never edited. **`history.go` MUST be byte-unchanged and no exported `History` method may be added**: `history_surface_guard_test.go` is a closed-route guard, set-equal in both directions, and it MUST pass with its own source unchanged.

(Previously: "`loop.go` and `scheduler.go` MUST be byte-unchanged: wiring history into `Turn`/`Schedule` is AG-13's" — stated without a milestone qualifier, so a later reader could take it as a standing prohibition that AG-13 violates rather than as the AG-12 freeze it was, and it made no claim about `history.go` or the exported-route surface.)

#### Scenarios

- **S-HIS-095** — Given the merge base of the AG-13 branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then `history.go` is byte-unchanged, every file under `backend/agent/src/ai/` is byte-unchanged, `go.mod` and `go.sum` are byte-unchanged, and the only Layer 2 non-test files that differ are `loop.go`, `scheduler.go`, `tool.go` and the run driver's new file.
- **S-HIS-096** — Given `history_surface_guard_test.go` and its committed expected-route set, when the suite runs against this change, then the guard passes with its source byte-unchanged and the enumerated exported route set is equal in both directions — AG-13 added no route and removed none.

### NFR-HIS-004 — Substrate filters widened by exact filename

`filterOutLoopFiles` and `filterOutLoopHookFiles` MUST be widened by **exact filename** for the new files and kept byte-in-sync with each other — no wildcard (the AG-11 recorded lesson).

### NFR-HIS-005 — Review budget

`openspec/config.yaml` forecasts a 400-line review budget; this change ships as a single pull request under a pre-authorised `size:exception` against a 1000-line budget, forecast at 1100–1600 changed lines including SDD markdown. The pull request description MUST state why the change does not fit the default budget.

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-12 closes more than it does.

| Not claimed | Owner |
|---|---|
| History is wired into `Turn` / `Schedule` | AG-13 (run driver). **CLOSED by AG-13**: `Turn` commits the turn's assistant message and its tool-result messages on the continuation path (`R-HIS-010`); the run driver commits the user-side messages and closes each turn (`R-RUN-005`). `Schedule` itself still commits nothing — it returns the rejoin and `Turn` commits from it |
| A transcript is persisted or reloaded across processes | Layer 3 |
| The invariant holds after compaction removes entries | AG-18.2, which re-proves it — ordinal-derived identity is a known, named constraint AG-18 inherits. *(Still AG-18's at AG-17: AG-17 ships the compaction **check** and no compaction, and its verdict type carries no field by which any strategy could request one, so no entry can be removed by anything this milestone ships)* **CLOSED by AG-18, and the closure is a re-proof rather than a waiver.** Compaction removes a **prefix** and nothing else, through the single commit primitive, at a boundary that is both mark-aligned and pairing-closed; the post-compaction transcript is then re-validated against the **same** append-time pairing rules seeded construction enforces, so the invariant holds on a shorter transcript rather than being suspended on the old one (`R-HIS-001` as amended, `R-CMP-004`, `R-CMP-005`). **The inherited constraint landed exactly as this row forecast**: ordinal-derived identity made renumbering unavoidable, so `R-HIS-005`'s stability clause is now scoped to a transcript generation and the durable cross-generation handle is the **turn identifier**, which is why `CompactionSpan` was built on that axis at AG-06 (`R-APE-007`). Two further facts are recorded here because a later reader will ask them of history rather than of compaction: *(1)* the summary entry is discriminated by a **third origin member on the envelope** and by nothing else — a content sentinel stays prohibited by `R-HIS-007`'s own argument, which does not weaken for a model-authored summary; *(2)* turn attribution is recorded **in the transcript store**, at turn close, through the same commit primitive and a package-private door, because the compactable region crosses prior run invocations and no run-scoped structure can see them. **Not closed here**: attribution of a seeded prefix no run ever drove, which fails typed as a named v1 limitation (`R-CMP-012`) |
| Cancellation semantics that *produce* an interruption | AG-14. AG-12 repairs the transcript an interruption left; it neither detects nor causes one. **CLOSED by AG-14**: interrupt and shutdown ship in the `agent-cancellation-tree` capability, and the harness's wind-down is orphan synthesis's first production caller (`R-HIS-007` back-annotation, `R-CAN-002`). History itself needed **no change** to accept it — no new route, no `history.go` edit. The **deadline** signal remains unclaimed by any milestone |
| Context-window accounting over the transcript | AG-17. **CLOSED by AG-17 — and history needed NO change to accept it, which is the substance of the closure rather than a footnote to it.** AG-17 measures the transcript of each coming logical turn at AG-13's turn boundary and hands the figure to an injected strategy. It reaches the transcript through the **existing** `transcriptFromHistory(hist)` route the run driver already uses (`harness.go:512`) — no new route, no `history.go` edit, no new exported `History` method — and it passes the strategy a **fresh clone**, so a strategy that rewrites what it received cannot rewrite the harness's slice (`request.go:367-370`'s own argument), which is also what keeps `R-RTY-002`'s by-reference reuse safe in the presence of a third-party strategy. **Three facts about the figure are recorded here because a later reader will ask them of history rather than of the seam.** *(1)* The figure is **not** a Layer 1 usage record: it carries its own three-state provenance — reported, estimated, unavailable — sharing no field with `ai.TokenCount`, whose presence bit means "Layer 1 reported it" (`ai/usage.go:34-55`). *(2)* The estimate path is reachable **only** from a genuinely absent counting capability (`R-AMP-018`); an advertised counter that errors, and an advertised counter that answers with an absent count and a nil error, both resolve to *unavailable* and **never** to an estimate, because `R-AMP-019` makes those non-conformance rather than absence. *(3)* Nothing is written back: no entry is appended, removed, re-ordered or re-identified by the accounting on any path, proved by a **byte-identical `Entries()` read-back** between a run with the seam installed and the same run with it nil. Owned by `agent-context-strategy` (`R-CTX-002`, `R-CTX-004`, `R-CTX-007`, `R-CTX-009`). **Not closed here**: acting on the figure — any threshold arithmetic, and compaction itself — which is AG-18's and Layer 3's *(compaction's half is now CLOSED by AG-18, per the row above; the threshold arithmetic remains Layer 3's)* |
| Steering-message queueing at turn boundaries | AG-13.2. **CLOSED by AG-13**: the queue, its arrival ordering, its zero-drop guarantee and its typed post-terminal rejection are owned by `R-RUN-008`. History needed no change to accept it — `commitAppendOp` carries no role-alternation check, so consecutive same-role entries were already legal |
| A new rule class in `ai/validation.go` | Not this milestone, and forbidden under this change. *(Still true at AG-14: the cancellation sentinels are Go error values, not `ai` rule classes, and the typed aborts reuse existing ones. Still true at AG-17, which adds no rule class and edits nothing under `backend/agent/src/ai/` at all. Still true at AG-18: the prefix replacement's rejections reuse the existing rule classes `R-HIS-002` and `R-HIS-003` already assign, the compaction refusal is a Go sentinel error, and the diff under `backend/agent/src/ai/` is empty)* |
| A general mid-transcript span replacement | **Not this milestone, and deliberately so.** AG-18 opens a **prefix**-shaped route only (`R-HIS-001`), because every charter scenario compacts the oldest region and protects the recent tail. A mid-span route has no consumer and would relax `R-HIS-001` further than the change requires |
| A compaction journal, undo, or pre-compaction snapshot | **Not this milestone.** Compacted entries are discarded irreversibly from memory; this is recorded as a named consequence (`R-CMP-013`), not as a migration concern, and is acceptable for v1 only because nothing persists across processes |
