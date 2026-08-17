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

### R-HIS-001 — The transcript is ordered, append-only within a run, and read back in order

History MUST hold an ordered sequence of Layer 1 values across a run. Entries MUST be appended in conversation order and MUST be read back in that same order. Within a run, history MUST NOT support removal, reordering, or in-place mutation of an already-committed entry; the only admitted extension is an append through the commit path of `R-HIS-004`.

A read MUST NOT alias internal storage: mutating anything a read returns MUST NOT be observable in a subsequent read.

#### Scenarios

- **S-HIS-001** — Given a sequence of Layer 1 messages appended in conversation order through the public append route, when the transcript is read back, then the read-back sequence is equal to the appended sequence, element-by-element and in the same order, with no insertion, omission or reordering.
- **S-HIS-002** — Given a populated history, when a caller in an external test package takes a read-back view and mutates the value it holds (including any slice it exposes), then a subsequent read of the same history returns the original values unchanged — no read aliases internal storage.

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

Every route that can extend or mutate the transcript — append, seeded construction, and orphan synthesis alike — MUST funnel through a single validating commit primitive. The package MUST NOT expose, and MUST NOT internally retain, a second route that reaches committed storage without that validation. This is `VL2-HAR-01`'s "exactly one commit path, with no privileged bypass for internal callers" and the C1 lesson (`0001:376`) reapplied to history.

The obligation MUST be discharged **by enumeration in test, not by assertion in prose**: the external test surface MUST enumerate every public route that can extend or mutate the transcript, drive an orphaning sequence through each, and assert rejection for each. The enumeration MUST be closed — a public mutating route that the enumeration does not name MUST be observable as a failure of the suite, never as an unaudited door.

Because the tests must exercise only what a caller outside the package can reach, they MUST live in an external test package (`package agent_test`): an in-package test could reach the private primitive directly and would prove nothing about the boundary.

#### Scenarios

- **S-HIS-030** — Given the enumerated set of every public route that can extend or mutate the transcript, when an external test constructs an orphaning sequence through each route in turn, then every route rejects it with the rule class `R-HIS-002` or `R-HIS-003` assigns to that shape, and no route accepts it.
- **S-HIS-031** — **(bite)** Given a scratch edit that adds a public mutating route reaching committed storage without the validating primitive, when the closed route enumeration of `S-HIS-030` runs, then it FAILS naming the unenumerated route — proving the audit is closed rather than a snapshot of the routes that happened to exist. Recorded, then reverted. RED-recordable.

### R-HIS-005 — Read-only views for the loop and for a future session, with stable entry identity

History MUST expose read-only views serving two consumers: the loop, which reads the transcript, and a future session, which reads entry identities. The transcript MUST arrive as **unmodified Layer 1 values** — the same `ai.Message` / `ai.ToolResult` values that were committed, neither re-serialised nor rewritten — carried in an opaque entry envelope with unexported fields and read-only accessors, per the reconciliation above.

Entry identity MUST be **derived deterministically from ordinal position** in the transcript. No caller may supply, mint, or overwrite an entry identity — that would be the C1 back door. Identity MUST be stable: the same transcript yields the same identities on every read and across processes.

No read may return a handle through which committed state can be changed.

#### Scenarios

- **S-HIS-040** — Given a populated history, when the loop-facing view is read from an external test package, then it yields the committed Layer 1 values unmodified and in order, and each value is equal to the one committed without re-serialisation or field rewriting.
- **S-HIS-041** — Given a populated history, when the same transcript is read twice, and when a second history is constructed over the same ordered seed, then each entry's identity is equal across both reads and across both histories — identity is a function of ordinal position alone.
- **S-HIS-042** — Given the public surface of the entry envelope, when it is enumerated from an external test package, then it exposes no route to set or replace an entry identity, an origin discriminator, or a committed Layer 1 value.

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

#### Scenarios

- **S-HIS-060** — Given a transcript interrupted after tool calls were issued but before their results arrived, when synthesis runs, then every orphaned call has a matching result in the transcript, each such entry reports origin `synthesized`, and every pre-existing entry still reports origin `appended`.
- **S-HIS-061** — Given a synthesized result and a real appended result constructed with byte-identical content and with `Failed()` set identically, when each entry's origin is read, then the two are distinguished correctly, and the assertion reads neither content nor `Failed()`.
- **S-HIS-062** — Given a history over which synthesis has run, when the turn is closed, then the close succeeds — synthesis produced a transcript the pairing invariant of `R-HIS-003` accepts, through the same commit path.

### R-HIS-008 — Synthesis is idempotent and total

Over a transcript holding N orphaned calls, synthesis MUST close **exactly N** pairs on the first application. A second application MUST change nothing: the transcript after the second application MUST be identical to the transcript after the first, entry-for-entry, including entry identities and origin discriminators.

Synthesis MUST NOT touch a call that already has a matching result, whether that result was appended or synthesized.

#### Scenarios

- **S-HIS-070** — Given a transcript containing N orphaned calls (with N ≥ 2) interleaved with already-matched calls, when synthesis is applied once, then exactly N new results are present, each matching a distinct orphaned call, and no already-matched call gained a second result.
- **S-HIS-071** — Given the transcript produced by the first application, when synthesis is applied a second time, then the read-back transcript is identical to the first-application transcript entry-for-entry — same order, same Layer 1 values, same entry identities, same origin discriminators — and no new entry was committed.

### R-HIS-009 — The `L2C-07` doc-contract row and its bidirectional guard

History is a **second upward surface** — a transcript read by the loop and by a future session — that no existing doc-contract row covers, and "exactly one commit path, no privileged bypass" is the machine-checkable, package-wide shape `L2C-01`..`L2C-04` are written in. The package documentation MUST therefore carry an `L2C-07` row stating that guarantee, and its `expectedLayer2ContractRows` entry MUST land in the **same pull request**, per `R-AGP-002`'s closed-amendment rule. The guardian is the same `doc_contract_guard_test.go` that guards `L2C-01`..`L2C-06`.

The per-behavior semantics of history belong in `doc.go` prose, not in the guarded row.

*Criterion to overturn in `sdd-design` (`proposal.md` decision 1)*: if the guarantee turns out to be expressible only as behavior of one type rather than as a constraint on the package, the row is dropped and the drop is stated explicitly.

#### Scenarios

- **S-HIS-080** — Given the `expectedLayer2ContractRows` table in `doc_contract_guard_test.go`, when it is read, then it contains 7 rows (`L2C-01`..`L2C-07`) — the new `L2C-07` row present, in order, with row text referencing history's single validated commit path as a package-wide guarantee — and the guard passes against `doc.go`.
- **S-HIS-081** — **(bite)** Given a scratch edit that appends an `L2C-07` row to `doc.go` without adding its entry to `expectedLayer2ContractRows`, when the doc-contract guard runs, then it FAILS naming the unexpected row — the closed-amendment rule is observed, not bypassed. Recorded, then reverted. RED-recordable.

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
| The invariant holds after compaction removes entries | AG-18.2, which re-proves it — ordinal-derived identity is a known, named constraint AG-18 inherits |
| Cancellation semantics that *produce* an interruption | AG-14. AG-12 repairs the transcript an interruption left; it neither detects nor causes one. *(Still open at AG-13: the run driver propagates its context unmodified and defines no cancellation vocabulary.)* |
| Context-window accounting over the transcript | AG-17 |
| Steering-message queueing at turn boundaries | AG-13.2. **CLOSED by AG-13**: the queue, its arrival ordering, its zero-drop guarantee and its typed post-terminal rejection are owned by `R-RUN-008`. History needed no change to accept it — `commitAppendOp` carries no role-alternation check, so consecutive same-role entries were already legal |
| A new rule class in `ai/validation.go` | Not this milestone, and forbidden under this change. *(Still true at AG-13: the continuation's typed rejections and `Steer`'s typed rejection reuse existing `ai` rule classes.)* |
