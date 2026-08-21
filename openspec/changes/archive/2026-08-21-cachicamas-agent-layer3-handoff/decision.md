# Decision — the Layer 2 readiness contract: frozen v1 surface and closing walk

> **Change**: `cachicamas-agent-layer3-handoff`
> **Milestone**: AG-23 — Publish the Layer 3 readiness contract (Wave 6, 24 of 24 — **Layer 2's exit**)
> **Node**: AG-23.1 — Consumer proof `[leaf]` · AG-23.2 — Packaged kit and examples `[leaf]` · AG-23.3 — Compatibility statement `[decision]`
> **Status**: decided
> **Project**: cachicamas (witsaba) · **Target package**: none — `[decision]` nodes ship no code
> **Closes**: doc 0003's AG-23.3 closing checklist (three items, § 6 below) and, together with AG-23.1's and AG-23.2's own landed evidence, the Layer 2 completion checklist (twenty-one rows; none stays open)
> **Sources**: [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) § AG-23, § Layer 2 completion checklist, § Traceability spine · [the `agent-layer3-handoff` spec](specs/agent-layer3-handoff/spec.md) (`R-L3H-001..011`) · [AI-40's own `decision.md`](../archive/2026-08-10-cachicamas-ai-layer2-handoff/decision.md) (the identical milestone shape, one layer down)
> **Binding predecessor**: `agent-hook-taxonomy`'s `R-HKS-010` (the W-6 scope-fence disposition this artifact records), `agent-run-driver`'s `R-RUN-014` (the forwarder fix this artifact certifies as fixed, never registered as a limitation), `agent-v1-scope`'s `R-AGS-016` (the declined pre-request-hook removal this artifact records as frozen-and-superseded)

> [!IMPORTANT]
> **This artifact enumerates a frozen surface and a closing walk, not code.** No Go type name, field name, method name, interface name or package identifier belonging to the Layer 2 contract appears here. Three identifier classes are permitted: a named landed test function, a named landed production function whose behavior is being cited as evidence, and a package's own import path when naming where something ships. § 2 states what a Layer 3 application may rely on **by capability and behavior**, never by exported identifier — the identifier-level detail lives in package documentation (`src/agent/doc.go`), which this artifact is pointed to from, never duplicated by. This statement is written throughout without reference to any concept a coding-focused application would need and a different kind of application would not (`0003:2152`).

---

## 1. How to use this document

**If you are a Layer 3 application author:** § 2 states exactly what you may build against, by capability, and marks what is still experimental. Build only against what § 2 marks frozen; nothing marked experimental is a promise. The importable scripting kit and the third-party consumer proof this milestone ships are your own starting shape — copy their pattern directly.

**If you are doc 0004's reader:** § 5 restates what Layer 3 inherits and names doc 0004's own dependency on this node's close.

**If you are reviewing this artifact:** § 3 walks all twenty-one Layer 2 completion-checklist rows against doc 0003's on-disk state, re-read fresh in this change's own working session immediately before this artifact was written. § 4 is where an unearned evidence claim, or a defect quietly relabelled as a design limitation, would be most expensive.

---

## 2. The frozen v1 surface, by capability

Everything below is frozen as of this milestone's close unless explicitly marked **experimental**. "Frozen" means: a Layer 3 application may build against the described behavior; a future change to it is a breaking change requiring its own amendment, not a silent drift. Nine categories; none is empty.

1. **Building and driving a run.** A caller assembles one value representing a multi-turn conversation from a model provider, a system instruction, and a set of per-turn options, then drives it to a final decision with one call taking a context, a prompt, and a receive-only channel it owns. The value requires no constructor and no required field; every optional seam below defaults to an inert, byte-identical-to-baseline behavior when left unset. **Seam — the run's model provider.** Injection point: a required field on the run value. v1 default: none — a caller must supply one; the packaged kit's own scripted provider (via the Layer 1 substrate it builds on) is the deterministic stand-in a test uses. `ExampleHarness` (this change) is the compiled, run demonstration.

2. **Steering, interruption, and shutdown.** A caller may offer an additional prompt to an in-flight run, which the driver folds in at the next turn boundary with zero drops; a caller may signal an interrupt or a shutdown from outside the run's own call, distinguishable from ordinary completion on the resulting stream. Wind-down after either signal is bounded — no unbounded wait, no invented delay. `ExampleHarness_Run` and this change's own consumer proof both drive a run past an interrupt and a resumed prompt on the same value.

3. **The permission protocol and its packaged scripting surface.** Every scheduled call consults an injectable decision port before it executes; four typed decisions plus a suspension are possible; a suspension parks only that one call while its siblings proceed and is released by an explicit, externally-reachable wake, never by a wall clock. **Seam — the decision port.** Injection point: an optional per-run field. v1 default: unset bypasses the port entirely — every call is treated as immediately allowed, byte-identical to a build with no decision port at all. **Seam — the caller-owned wake handle.** Injection point: an optional field on the run value. v1 default: unset constructs one internally; supplying one's own value is what makes the external wake that releases a suspension reachable from outside the run's own call. The packaged kit ships an importable, scriptable implementation of this port — queued decisions resolved strictly in order, an exhausted queue falling back to a stated default rather than wedging. `ExampleHarness_suspension` (this change) and `TestLayer3Handoff_ConsumerProof` (this change) both drive a real suspension and its resolution through this exact seam.

4. **Tool scheduling.** A registry resolves a call's name to an implementation; read-class calls run concurrently up to a bound, mutating and execute-class calls run serialized in call order; the rejoin preserves call order regardless of completion order. A tool receives an opaque per-call value it must not interpret — only forward — the confinement decision belongs one layer above. **Seam — the tool registry.** Injection point: a per-run field resolving a scheduled call's name to an implementation. v1 default: unset yields a typed unresolved-name failure result for every scheduled call rather than a crash — one bad name does not abort the turn. The packaged kit ships an importable, scriptable tool implementation recording its own invocation count and arguments for a consumer's own inspection.

5. **The event stream and its own validator.** Every item on the stream carries a kind derived from what it actually holds, a position stamped fresh per run, and belongs to a documented bracket discipline (a run bracket, nested turn brackets, and same-turn-scoped tool and permission events). An already-exported validator accepts or rejects a whole drained stream in one call; the packaged kit's own drain helper delegates to it wholesale, never re-implementing any rule. `ExampleHarness_events` (this change) demonstrates consuming the stream by kind name.

6. **Transcript continuation across values.** A completed run's committed transcript is readable through public accessors; a second, independent run value can be seeded from exactly those entries, with no channel other than that public surface ever touched to move it. **Seam — the transcript.** Injection point: an optional field on the run value. v1 default: unset constructs an empty transcript internally; supplying one seeds a run's continuation, including through the seeded-entries route this milestone's own consumer proof exercises below. This change's own consumer proof builds a second run value this way and completes a further conversation on it.

7. **Cost accounting.** Every turn emits a cost figure; a running total accumulates across retries and any mid-run context reduction, labelled as an estimate until the run's own final figure supersedes it. No money or currency value is ever carried — token counts only.

8. **The pre-request seam and the hook taxonomy.** One singular, exported pre-request seam is composed first inside a wider, typed taxonomy of four hook families (a pre-request family, a pre-reduction family, a post-turn family, and a session-start family); the two mutating families take a payload and return one with an error, the two observing families have no result parameters at all, so a hook that could signal a mutation is unconstructible by the type system, not by convention. Observing hooks run on their own lane and cannot stall the stream even when one of them never returns. **Seam — the wider hook-family registration surface.** Injection point: one registration value on the run, composed once per turn alongside the singular pre-request field. v1 default: the zero value is inert — no lane, no goroutine, no queue, and every arm behaves byte-identically to a build carrying no hook registration at all. **Seam — the singular pre-request field.** Injection point: a per-run field, composed as element zero of the wider taxonomy's own chain. v1 default: unset changes nothing. **Frozen-and-superseded**: this field is kept, unamended in behavior, carries no deprecation marker, and is published here as frozen with a post-v1 removal path (§ 4) — it is not removed and MUST NOT be described as removed or deprecated anywhere in this document.

9. **The observability boundary.** An optional, injected tracing-API value records a small set of spans and attributes drawn from a fixed allowlist; left unset, a run is inert and behaviourally identical to a build carrying no observability capability at all. **Seam — the tracing-API provider.** Injection point: an optional field on the run value. v1 default: unset resolves internally to the tracing API's own no-op provider — a run stays inert and behaviourally identical to a build carrying no observability capability at all. No vendor software-development kit, no exporter, and no process-global telemetry state is reachable through this boundary — the tracing application-programming interface only.

### Marked experimental — not frozen

- **A production coordination tool for nested runs.** The structural properties a delegation tool would stand on — nested identity, cost, and permission scope — are proven; no such tool ships in v1, and none is configured by default. **Seam — the in-frame delegation door.** Injection point: installed internally, per scheduled call, onto that call's own frame — never a field the caller sets directly on the run value. v1 default: no coordination tool is shipped or configured, so nothing exercises the door in v1. Its post-v1 path is named in § 4.
- **A failover policy that actually switches providers.** **Seam — the failover policy.** Injection point: an optional field on the run value, consulted exactly once at the retry bound's exhaustion. v1 default: unset is never called and behaves as an unconditional decline — the one shipped, installable implementation has the identical behavior. A real policy re-opens token budgets and cached-prefix assumptions and needs its own design; its post-v1 path is named in § 4.
- **A default context-reduction policy.** **Seam — the context-reduction policy.** Injection point: an optional field on the run value, consulted once per logical turn. v1 default: unset means no reduction is ever attempted and the run behaves exactly as it did before the seam existed — the one shipped, installable implementation has the identical behavior. What triggers a reduction, and the instruction driving it, are both Layer 3's to supply; its post-v1 path is named in § 4.

---

## 3. The twenty-one-row completion-checklist walk

Every row of [the Layer 2 completion checklist](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#layer-2-completion-checklist), in order, against the on-disk state re-read fresh this session, and the closing node(s) [the traceability spine](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#requirements--closing-nodes) names. Each row **reports** status as of the milestone that closed it; **none re-verifies an already-closed row's own evidence, and no row but the last cites this change.**

| # | Row (doc 0003's own words) | Status | Closing node(s) | Evidence |
|---|---|---|---|---|
| 1 | The package exists at the ADR 0005 § D2 location with both boundary guards biting. | `[x]` | AG-03 | doc 0003 line 2161's own pre-existing annotation — "closed by AG-03." — present, byte-identical, at the merge base (`git diff main -- docs/…/0003-…md` touches only this row's checkbox character, never its annotation text); a pre-existing documentation gap, closed long before this milestone; see § 3 note below |
| 2 | Every event family of v2 § 4.3 — all eight — is constructible, validated, and guarded. | `[x]` | AG-04, AG-05, AG-06 | doc 0003 line 2162's own pre-existing annotation — "closed by AG-04, AG-05, AG-06." — present at the merge base |
| 3 | All four envelope invariants hold by test. | `[x]` | AG-04.3, AG-19.1, AG-20.2, AG-11.2 | doc 0003 line 2163 (already checked) |
| 4 | One turn runs end to end against the fake provider with events in contract order. | `[x]` | AG-07 | doc 0003 line 2164's own pre-existing annotation — "closed by AG-07." — present at the merge base |
| 5 | The pre-request hook can rebuild the outgoing request; the identity default changes nothing. | `[x]` | AG-08 | doc 0003 line 2165's own pre-existing annotation — "closed by AG-08." — present at the merge base |
| 6 | Tools execute with read-parallel/write-serial policy, bounded fan-out, and call-ordered rejoin. | `[x]` | AG-09 | doc 0003 line 2166's own pre-existing annotation — "closed by AG-09." — present at the merge base |
| 7 | Permission is a suspension on the stream with all four outcomes; suspension blocks nothing else. | `[x]` | AG-10 | doc 0003 line 2167's own pre-existing annotation — "closed by AG-10." — present at the merge base |
| 8 | Refusal, pause, and unknown finish reasons produce three distinct behaviors. | `[x]` | AG-11.1 | doc 0003 line 2168 (already checked) |
| 9 | The loop never retries and never decides policy; the guards and tests that prove it stay green. | `[x]` | AG-11.2, AG-10.1, AG-03.3 | doc 0003 line 2169's own pre-existing annotation — "closed by AG-11.2, AG-10.1, AG-03.3." — present at the merge base |
| 10 | History cannot orphan a tool call; interruption synthesizes results; enforcement is at the boundary. | `[x]` | AG-12 | doc 0003 line 2170 (already checked) |
| 11 | A multi-turn run completes with steering, pause resumption, and a complete event story. | `[x]` | AG-13 | doc 0003 line 2171 (already checked) |
| 12 | Interrupt and shutdown are distinguishable end to end; wind-down is bounded. | `[x]` | AG-14 | doc 0003 line 2172 (already checked) |
| 13 | Partial-output failures are never silently retried; retry attempts are visible events. | `[x]` | AG-15.1 | doc 0003 line 2173 (already checked) |
| 14 | Every turn emits a cost event; cumulative figures include retries and compaction; estimates are labelled. | `[x]` | AG-16.1, AG-18.1 | doc 0003 line 2174 (already checked) |
| 15 | The context strategy is consulted before every call; token counting is capability-discovered with a labelled fallback. | `[x]` | AG-17 | doc 0003 line 2175 (already checked) |
| 16 | Compaction protects recent turns, preserves the pairing invariant, is recorded on the stream, recovers from interruption, and is invocable on demand. | `[x]` | AG-18 | doc 0003 line 2176 (already checked) |
| 17 | The harness is re-entrant: nested runs with nested cancellation, cost, and permission scope, parent-identified. | `[x]` | AG-19 | doc 0003 line 2177 (already checked) |
| 18 | All four hook points fire; observers cannot stall the stream. | `[x]` | AG-20 | doc 0003 line 2178 (already checked) |
| 19 | The package is race-clean and leak-free under the combined-scenario matrix. | `[x]` | AG-21 | doc 0003 line 2179 (already checked) |
| 20 | Telemetry stays inside the § D3 boundary with the denylist proven by absence. | `[x]` | AG-22 | doc 0003 line 2180 (already checked) |
| 21 | The Layer 3 readiness contract is published with the consumer proof and the scripted-harness kit. | `[x]` (this change) | AG-23 | `TestLayer3Handoff_ConsumerProof`, the four `Example*` functions, and this artifact (this change) |

**Row 1, 2, 4, 5, 6, 7, 9 — the flip is a documentation-defect closure, not new evidence.** Doc 0003 shipped these seven rows unchecked long after the milestones that actually closed them merged — a pre-existing gap in the on-disk document, not a re-opened or re-decided claim. Each row cites the same node the traceability spine already names for it; this change corrects the checkbox to match evidence that has been merged since AG-03 through AG-11.2 respectively, and asserts nothing new about their own behavior.

**Row 21, read in full.** `AG-23.1` supplies the compiled, run, zero-vendor-import consumer proof; `AG-23.2` supplies the packaged kit and the four runnable examples; `AG-23.3` — this artifact — supplies the frozen-surface declaration (§ 2), the checklist walk (§ 3, this section), and the known-limitations register (§ 4). All three close together; none alone would satisfy this row's own clause.

---

## 4. Documented contracts and the known-limitations register

### 4.1 The abandoned-consumer contract, inherited from Layer 1

The caller owns the context passed to a run. A consumer that stops draining the returned stream and never cancels that context is a **contract violation by the consumer**, not a defect in the driver — nothing here promises to notice an abandoned, never-cancelled consumer and release its own goroutine unasked. This posture is inherited unchanged from Layer 1's own decision (`cachicamas-ai-layer2-handoff`'s `decision.md` § 4): the never-cancelled case is untestable to termination, and is documented here for the same reason it was documented there — a test that never supplies the one signal a producer waits on cannot itself terminate.

**This is distinct from, and must not be confused with, this change's own forwarder fix (§ 4.3 below).** The fix concerns a narrower hazard — a crash reachable through an unrecovered mutating-hook panic, on a path that terminates regardless of consumer behavior. It does not, and could not, make an abandoned-and-never-cancelled consumer's own goroutine leak disappear: that posture remains the caller's own responsibility, exactly as stated above.

### 4.2 The known-limitations register

Each entry is **restated, not re-litigated**: it reports a decision already taken elsewhere and does not reopen, weaken, or re-decide it.

| Limitation | Seam it attaches to | Post-v1 path |
|---|---|---|
| **No production coordination tool for nested runs ships in v1.** The proven re-entrancy substrate (nested identity, cost, and permission scope) is the seam a future tool would stand on; none is configured by default and none ships. | The re-entrancy substrate this change's own predecessor milestone proved | A production tool and its depth limit are decided by a future proposal against this now-proven substrate — doc 0003's own deferred-capability table names the same disposition |
| **Failover declines every time in v1.** The seam is consulted exactly once, at the retry bound's exhaustion; the only shipped implementation always declines. | The failover seam, consulted once per logical turn's own exhaustion | A real policy needs its own design — it re-opens token-budget and cached-prefix assumptions a v1 answer cannot safely presume |
| **No default context-reduction (compaction) policy ships in v1.** Left unset, the seam is never consulted for a decision and no reduction is ever attempted — a run behaves exactly as it did before the seam existed. | The context-reduction seam, consulted once per logical turn | What triggers a reduction, and the instruction driving it, are Layer 3's to supply and remain iterative/configurable rather than a v1 default |
| **The abandoned-consumer contract, inherited from Layer 1** (§ 4.1). A consumer that stops draining a run's stream and never cancels its context leaks a goroutine by its own defect, untestable to termination. | The run's own owned context, threaded to every downstream call | No path — this is a caller obligation, not a capability gap, and is not expected to change |

### 4.3 A defect is not a limitation, and this register states so explicitly

A crash reachable through the deliberately unrecovered pre-request-hook seam — a per-attempt event forwarder left parked forwarding to a sink an abandoned-but-not-yet-cancelled consumer would never again read, racing the run's own close of that sink during the panic's unwind — was found and **fixed** this milestone (`agent-run-driver`'s `R-RUN-014`), RED-first — the RED watched failing before the fix existed, and both defeat directions planted and watched failing after the fix landed, each reverting one half of it before being reverted in turn. It is a crash-class defect reachable through a seam this milestone freezes, not a design limitation with a post-v1 path, and it appears **nowhere** in the register above. Laundering it as a limitation would make this statement false on the day it lands. The unrecovered-panic posture itself is unchanged and unweakened by the fix: a panicking mutating hook still propagates uncontained, recovered by nothing this driver owns; what changed is only that the driver's own forwarder can no longer crash a goroutine the caller does not own while that propagation happens.

### 4.4 The scope-fence's two prior releases, finally disposed of

Two source units were released from a shipped scope fence at an earlier milestone, pending disposition once a later milestone either needed the same source again or confirmed it did not. Both are disposed of here, since this is Layer 2's last milestone and neither disposition has anywhere left to be forwarded to:

- **One source unit is restored to the fence.** This milestone ships no change to it; the earlier release expired with the milestone that took it.
- **One source unit's release is made PERMANENT, and the reasoning is recorded here as well as beside the list in the guard's own source.** That unit is the layer's designed **extension point** for its own import boundary: every milestone that admits a new dependency or a new sibling package must edit it, by construction. Two independent milestones needing the same extension point is evidence the freeze **entry** was wrong, not that either milestone was — freezing a designed extension point is a category error. Restoring it would make the restoring milestone's own branch fail its own restored guard.

---

## 5. What Layer 3 inherits

Doc 0003 names AG-23 the normative entry gate for doc 0004: every node in that document depends on this node's close. Layer 3 inherits, unconditionally:

1. **The frozen v1 surface (§ 2)** — every capability, its injection point, and its v1 default, to build against without inference.
2. **The packaged scripting kit** — an importable substrate for testing against the surface above, without ever reaching into a Layer 2 test-only compilation unit.
3. **The known-limitations register (§ 4.2)**, each with a post-v1 path, and the explicit statement that a defect is never entered there (§ 4.3).
4. **Doc 0003's own `AG-23` gate itself** — unblocked by this change's own close (§ 6 below); doc 0004's nodes may now proceed, consuming exactly the frozen surface § 2 enumerates and nothing marked experimental.

---

## 6. Closing-checklist verification

AG-23.3's own three closing-checklist items (doc 0003 § AG-23), walked against this artifact.

| # | Item (doc 0003's own words) | Where answered | Status |
|---|---|---|---|
| 1 | The v1 surface is enumerated and frozen; experimental corners are marked; the statement names what a Layer 3 application may rely on, including every seam's injection point and its v1 default. It is written without reference to files, shells, skills or terminals. | § 2 | **answered** — nine frozen categories enumerated by capability, three items explicitly marked experimental, no category omitted, and this artifact carries no reference to any of the four named concepts anywhere |
| 2 | The Layer 2 completion checklist is walked item by item, each citing its closing node. | § 3 | **answered** — twenty-one rows, in checklist order, each citing its closing node(s), matching doc 0003's own traceability spine |
| 3 | The known-limitations register is stated: no subagent tool, failover declines, never-compact default, and the abandoned-consumer contract inherited from Layer 1 — each with its post-v1 path. | § 4 | **answered** — all four named limitations present with their seams and post-v1 paths, and the forwarder defect is recorded as fixed, appearing in the register nowhere |

**Node status.** AG-23.3 closes on merge of this artifact, together with AG-23.1's and AG-23.2's own landed evidence: `TestLayer3Handoff_ConsumerProof` (the seven-capability sequential driver), the vocabulary-boundary guard, and the four `Example*` functions (`ExampleHarness`, `ExampleHarness_Run`, `ExampleHarness_events`, `ExampleHarness_suspension`). Per doc 0003's own node grammar, a `[decision]` leaf produces no production code of its own and closes when the decision artifact answers every listed question and is merged.

**Unblocked by this decision:** doc 0004's own nodes (§ 5, item 4). **Layer 2 is complete**: no row of its completion checklist remains open after this change merges.
