# Decision — the Layer 2 readiness contract: frozen v1 surface and closing walk

> **Change**: `cachicamas-ai-layer2-handoff`
> **Milestone**: AI-40 — Publish the Layer 2 readiness contract
> **Node**: AI-40.1 — Consumer proof `[leaf]` · AI-40.2 — Capability matrix and examples `[leaf]` · AI-40.3 — Compatibility statement `[decision]`
> **Status**: decided
> **Date**: 2026-08-10
> **Project**: cachicamas (witsaba) · **Target package**: none — `[decision]` nodes ship no code
> **Closes**: doc 0002's AI-40.3 closing checklist (three items, § 6 below) and, together with AI-40.1/AI-40.2's own landed evidence, the Layer 1 completion checklist (eighteen items; item 6 excepted by design, § 3 row 6)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) § AI-40, § Layer 1 completion checklist, § Traceability spine · [AI-29's decision.md](../../../backend/agent/src/ai/openaicompat/decision.md) §§ 7, 11 (the two inherited publication duties this node discharges) · [the `ai-layer2-handoff` spec](specs/ai-layer2-handoff/spec.md) (`R-L2H-001..009`)
> **Binding predecessor**: AI-29's `decision.md` § 11 (the wire-clause and strip-reasoning duties delegated to this node) and AI-38.2's committed capability expectation (the published matrix's sole source, transcribed not regenerated)

> [!IMPORTANT]
> **This artifact enumerates a frozen surface and a closing walk, not code.** No Go type name, field name, method name, interface name or package identifier belonging to the Layer 1 contract appears here. Three identifier classes are permitted: a named landed test function, a named landed production function whose behavior is being cited as evidence, and the vendor's wire-level field names. § 2 states what Layer 2 may rely on **by capability and behavior**, never by exported identifier — the identifier-level detail lives in package documentation (`src/ai/doc.go`), which this artifact is pointed to from, never duplicated by.

---

## 1. How to use this document

**If you are AG-03's author (doc 0003's first Layer 2 milestone):** § 2 states exactly what you may build against, by capability, and marks what is still experimental. Build only against what § 2 marks frozen; nothing marked experimental is a promise.

**If you are any later doc-0003 reader:** § 5 restates the two publication duties this node inherited from AI-29, and names doc 0003's own dependency on this node's close.

**If you are reviewing this artifact:** § 3 walks all eighteen Layer 1 completion-checklist items against doc 0002's on-disk state, fresh-read in this change's own `tasks.md` Phase 0 immediately before this artifact was written. § 4 is where a missing "not reopened" clause, or an unearned test-coverage claim, would be most expensive.

---

## 2. The frozen v1 surface, by capability

Everything below is frozen as of this milestone's close unless explicitly marked **experimental**. "Frozen" means: Layer 2 may build against the described behavior; a future change to it is a breaking change requiring its own amendment, not a silent drift. Ten categories; none is empty.

1. **Request construction.** A caller assembles one normalized, provider-neutral request from a model identifier and an ordered list of role-tagged messages, each carrying one or more content parts. Construction validates once, in a fixed rule order, and rejects — rather than silently repairs — anything malformed. Nothing on this path performs network or filesystem I/O. `ExampleNewRequest` (this change) is the compiled, run demonstration.

2. **Content and tool vocabulary.** A message's content is one or more typed parts, each readable from outside the package that defines it, with no unconstructed value able to reach a request. A tool call crossing the stream carries its identity and name up front; its argument bytes arrive as an ordered sequence of fragments, and the call's closing event carries the authoritative complete bytes — byte-for-byte equal to the ordered concatenation of those fragments, never a value a consumer must reassemble and trust unverified. `ExampleModelProvider_toolCallReconstruction` (this change) demonstrates the reconstruction directly.

3. **Cache breakpoints, per-request options, and mutation-free rebuild.** Cache breakpoints are expressible on a request; a set of per-request options and a provider-specific escape hatch exist; and a request already built can be rebuilt with one change applied without mutating the original — the original and the rebuilt value remain genuinely independent.

4. **The event contract and per-stream sequencing.** Every item on a stream carries a kind derived from what it actually holds — never a separately settable field that could disagree with its own payload — and a position stamped fresh per stream, with no state shared across streams: two concurrent streams each start their own numbering at one, independently and contiguously. `ExampleModelProvider_streaming` (this change) demonstrates draining a stream in kind order.

5. **The provider interface and its pre-stream order.** One call every adapter offers: a cancellable context and a normalized request in, a receive-only event stream out. Validation runs exactly once, before the context is even consulted; only after that does the adapter check for prior cancellation. No optional capability widens this one-call contract — an optional capability, when a provider value has it, is discovered by asking the value directly, never by adding a second call.

6. **The typed failure taxonomy and the partial-output discriminator.** Every failure — whether returned directly before any request leaves the process, or carried as a stream's terminal event after a carrier was handed over — is one concrete type, classified into one of nine named categories (authentication, authorization, rate limiting, unavailability, timeout, cancellation, malformed response, unsupported capability, unknown), each independently reporting whether it is retryable and, separately, whether a retry-after hint was reported at all. A failure also independently states whether normalized output already reached the consumer before it occurred — one boolean, never folded into which delivery path carried it. `ExampleFailure_inspection` (this change) demonstrates inspecting a scripted terminal failure through this surface alone.

7. **Cancellation and bounded backpressure.** The caller owns the context passed to a stream call; a producer selects on that context on every send, including its last one, so cancellation always wins over a stalled consumer. The channel returned is unbuffered (capacity zero — measured against realistic workloads across three consumer-speed profiles, twenty runs each), with exactly one sanctioned loss path: a saturated buffer racing cancellation drops the remaining events and closes bare, with no terminal event forced through. § 4 below restates the abandoned-consumer posture this capability implies.

8. **The pre-stream retry policy.** A retryable pre-stream failure is retried automatically, up to a documented bound, honoring a provider's retry-after hint when one is reported, replaying the identical request byte-for-byte on every attempt, and draining each attempt fully before the next begins.

9. **The testing surface.** Three layers, reusable by any future adapter without reimplementing their mechanics: a scripted, deterministic fake satisfying the same one-call provider contract every real adapter does; a stream test kit built on it (a timeout-safe drain, readable bounded diffs, ordering and per-stream sequence-contiguity checks, and an opt-in goroutine-leak detector); and a conformance suite built on both (one exported entry point, a caller-supplied factory in, a nine-entry capability record out, with zero copied assertions). `TestHandoff_ConsumerProof` (this change) is this surface's own compiled, run evidence — a Layer 2 author copies its shape directly, importing nothing beyond the standard library, the neutral contract package, and the test-support package.

10. **The observability boundary.** A first, and — per this module's own dependency ADR — second-and-last authorized dependency: minimal tracing spans and attributes drawn from a fixed, twelve-key allowlist, active only by explicit injection (a caller supplying no tracer gets a no-op one, never a default that reaches out on its own), with an absolute content denylist proven by planted-and-detected absence. No metrics, no log bridge, and no vendor SDK reach this module through this boundary — only the tracing API.

### Marked experimental — not frozen

- **The live-smoke package.** An internal, environment-gated package that dispatches one bounded real request to the live vendor endpoint. It is unreachable from any composition root, skips cleanly with no credential present, and is never part of the ordinary test run — the ordinary test run's own green result never depends on it. It exists to catch drift between recorded assumptions and the live vendor, not to be depended on by a consumer, and it is explicitly not part of the frozen surface above.
- **Token counting.** A discoverable optional capability — a provider value may additionally satisfy a token-counting query, checked by asserting the value against it, never assumed present. The first landed adapter does not satisfy it. A Layer 2 author MUST NOT assume token counting is available; it MUST be discovered per provider value, the same way every other optional capability is.
- **The `R-CNF-020`…`R-CNF-026` identifier gap.** Two archived specification amendments consumed these seven identifiers for requirements never promoted into the canonical conformance specification, while two of the same numbers are already cited as binding dependencies elsewhere in the specification tree. AI-36 recorded this in place, without renumbering or backfilling it. It is a tracked defect in the specification identifier space, not a behavioral gap in the shipped adapter — a reader MUST NOT infer that any capability behind those numbers is either frozen or missing from this statement because of it, and this change does not close it.

---

## 3. The eighteen-item completion-checklist walk

Every item of [the Layer 1 completion checklist](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#layer-1-completion-checklist), in order, against the on-disk state observed fresh this session (`tasks.md` Phase 0, before this artifact was written) and the closing node(s) named by [the traceability spine](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#completion-checklist--nodes). Each row **reports** status as of the milestone that closed it; none re-verifies an already-closed item's own evidence.

| # | Item (doc 0002's own words) | Observed on 2026-08-10 | Closing node(s) | Evidence |
|---|---|---|---|---|
| 1 | The `backend/agent` module exists at the ADR-defined location, and `database_administrator/src/tools/` is untouched. | `[x]` | AI-00.1 | doc 0002 checklist line 2430 |
| 2 | **Both** import directions are mechanically guarded, and each guard has been recorded biting. | `[x]` | AI-00.3, AI-00.4 | line 2431 |
| 3 | Neutral request, message, content and tool contracts are documented and tested. | `[x]` | AI-05 … AI-10 | line 2432 |
| 4 | Every content-part variant is readable from another package, and no unconstructed value can reach a request. | `[x]` | AI-06.2, AI-06.3, AI-10.5 | line 2433 |
| 5 | Cache breakpoints are expressible; per-request options and a provider escape hatch exist; a request is rebuildable without mutation. | `[x]` | AI-11, AI-12 | line 2434 |
| 6 | Provider round-trip tokens survive byte-exact through normalization, rebuild, and the wire. | `[ ]` — **stays open by design** | Stream half closed: AI-07.3, AI-12.1, AI-17 (`R-ARE-009`/`R-ARE-010`, the `G12(b)` spine row). Wire half: AI-26.6 landed as a refusal; AI-29.2 is **struck** by AI-29 | doc 0002 lines 2435, 2458; AI-29's `decision.md` § 11; this change's `src/ai/doc.go` publication (AI-40.2) |
| 7 | Event order and stream ownership are explicit, and the sequence is per-stream and provably starts at 1 for every stream. | `[x]` | AI-14.2, AI-14.4, AI-22.3 | line 2436; Wave 2 close amendment, line 2451 |
| 8 | Every event kind that can be emitted has a payload that can actually be constructed. | `[x]` | AI-14.1, AI-19.1 | line 2437; Wave 2 close amendment, line 2452 |
| 9 | The error taxonomy is typed, safe and inspectable, with a partial-output discriminator. | `[x]` | AI-19 | line 2438; Wave 2 close amendment, line 2453 |
| 10 | The provider interface exposes no vendor type, and optional capabilities are discovered rather than required. | `[x]` | AI-20.1, AI-20.4, AI-20.5 | line 2439; Wave 2 close amendment, line 2454 |
| 11 | Cancellation cannot leak goroutines. | `[x]` (this change) | AI-33 | AI-33 close amendment, line 16 |
| 12 | Backpressure is bounded and lossless, with exactly one sanctioned loss path. | `[x]` (this change) | AI-34 | AI-34 close amendment, line 18 |
| 13 | The fake provider supports deterministic Layer 2 tests. | `[x]` | AI-21 | line 2442; Wave 3 close amendment, line 2462 |
| 14 | The conformance suite can be reused for every future adapter. | `[x]` (this change) | AI-23, AI-38.1 | AI-23, Wave 3 close amendment line 2463; AI-38.1, AI-38 close amendment line 48 (PR #140, `5bc2da4e`) |
| 15 | The first concrete adapter passes deterministic conformance. | `[x]` (this change) | AI-38 | AI-38 close amendment, line 48 |
| 16 | Secrets and sensitive bodies are absent from diagnostics by default. | `[x]` (this change) | AI-36, suite case AI-23.7 | AI-36 close amendment, line 24; suite case AI-23.7, spine line 2519 |
| 17 | The live test is optional, bounded, and unreachable from any entry point. | `[x]` (this change) | AI-39.1 | PR #142, merged `b062be74` |
| 18 | The Layer 2 handoff example compiles without vendor dependencies, and the v1 surface is declared frozen. | `[x]` (this change) | AI-40.1, AI-40.3 | `TestHandoff_ConsumerProof` (this change); this artifact (AI-40.3) |

*Line numbers above are as of this change's own close (post AI-40 amendment, doc 0002 line count +12 from the amendment block above the checklist); a later amendment appended above the checklist will shift them again by its own line count — the item text and node citations are the durable identity, not the numbers.*

**Row 6, read in full.** Item 6 stays `[ ]` — by design, not by omission. `AI-29.2` is struck by AI-29 (see AI-29's `decision.md` §§ 4–9): the pinned dialect's own schema carries no reasoning field a v1 adapter could round-trip, so the wire half has no v1 node that can close it. `src/ai/doc.go` (this change, AI-40.2) publishes that wire clause as **not exercisable in v1**, naming `AI-26.6`'s refusal and `AI-29.2`'s striking as its cause. The **stream** half — closed by AI-17 (`R-ARE-009`/`R-ARE-010`, the `G12(b)` spine row) — is named **unaffected and not reopened** in that same publication. Neither this row nor `src/ai/doc.go`'s publication ticks item 6's checkbox or restates AI-17's closure as open.

**Row 18, read in full.** `AI-40.1` supplies the compiled, run, zero-vendor-import example (`TestHandoff_ConsumerProof`, proven by the existing module-wide boundary guard rather than by inspection); `AI-40.3` — this artifact — supplies the frozen-surface declaration (§ 2). Both close together; neither alone would satisfy item 18's two clauses.

---

## 4. The never-cancelled abandoned-consumer posture, documented as contract

The caller owns the context passed to every stream call. The producer blocks until that context ends — it does not invent its own timeout, and it does not close a stream merely because a consumer stopped reading for a while. Abandoning a stream without ever cancelling its context is therefore a contract violation **by the consumer**, not a defect in the producer: nothing in this module promises to notice an abandoned, never-cancelled consumer and free its goroutine unasked.

The tested neighbour is abandoned-**then**-cancelled: a consumer that stops reading and only later has its context cancelled is covered by two landed cases — AI-23.5 (the consumer stops reading; the producer exits and the saturated-drop behavior matches the sanctioned loss path exactly) and AI-33.3 (the consumer has stopped reading and the producer is blocked mid-send; cancellation unblocks it — late events are dropped and the stream closes without a terminal event forced through). Both prove the producer's goroutine is released once cancellation eventually arrives, even though nobody was left reading by then.

The **never**-cancelled case — a consumer that stops reading and never cancels, ever — differs in kind, not only in degree: a test that never supplies the one signal the producer is documented to wait on cannot itself terminate, so no test can prove this case to termination. It is marked here, explicitly, as **untestable to termination**, and this document is deliberately the enforcement mechanism in its place: a Layer 2 author is on notice that leaking the goroutine in this shape is the abandoning caller's own defect, not a bug report against Layer 1. No claim of test coverage is made for this case, and none should be inferred from the tested neighbour above.

---

## 5. What Layer 2 inherits

Doc 0003 names AI-40 the normative entry gate: every code-bearing milestone in that document, from its `AG-03` onward, depends on this node's close. Layer 2 inherits, unconditionally:

1. **Completion-checklist item 6's wire clause** — not exercisable in v1, published in package documentation beside duty 2 (§ 3 row 6 above; AI-29's `decision.md` § 11; `src/ai/doc.go`, this change).
2. **The strip-reasoning duty** — Layer 2 MUST strip OpenRouter's `reasoning_details` field on the wire, a recorded absence (AI-29's `decision.md`), not an oversight, published beside duty 1 in the same package documentation.
3. **Doc 0003's `AG-03` gate itself** — unblocked by this change's own close (§ 6 below); `AG-03` may now proceed, consuming exactly the frozen surface § 2 enumerates and nothing marked experimental.

---

## 6. Closing-checklist verification

AI-40.3's own three closing-checklist items (doc 0002 § AI-40), walked against this artifact.

| # | Item (doc 0002's own words) | Where answered | Status |
|---|---|---|---|
| 1 | The v1 surface is enumerated and declared frozen; anything experimental is marked as such; the statement names exactly what Layer 2 may rely on. | § 2 | **answered** — ten frozen categories enumerated by capability, three items explicitly marked experimental, no category omitted |
| 2 | The completion checklist is walked item by item, each row citing the node that closed it — the spine is the template. | § 3 | **answered** — eighteen rows, in checklist order, each citing its closing node(s), matching doc 0002's own traceability spine |
| 3 | The abandoned-consumer-who-never-cancels posture is restated as documented contract... the never-cancelled case... must be documented because it cannot be tested to termination. | § 4 | **answered** — all three clauses restated, the tested neighbour named with its evidence, the never-cancelled case marked untestable to termination with no coverage claim |

**Node status.** AI-40.3 closes on merge of this artifact, together with AI-40.1's and AI-40.2's own landed evidence: `TestHandoff_ConsumerProof` (three subtests: drain, scripted error, bounded cancellation); `ExampleNewRequest`, `ExampleModelProvider_streaming`, `ExampleModelProvider_toolCallReconstruction`, `ExampleFailure_inspection`; and the published capability matrix in `src/ai/doc.go` with its read-only drift guard (`TestPublishedCapabilityMatrix_MatchesTheCommittedExpectation`). Per doc 0002's node grammar, a `[decision]` leaf produces no production code of its own and closes when the decision artifact answers every listed question and is merged.

**Unblocked by this decision:** doc 0003's `AG-03` onward (§ 5 item 3). **Not closed by this decision:** completion-checklist item 6, which stays `[ ]` by design — its wire clause is restated as not-exercisable-in-v1 (§ 3 row 6) and no artifact in this change ticks it or claims it closed.
