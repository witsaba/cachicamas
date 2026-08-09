# Proposal — AI-38: Run full deterministic adapter conformance

> **Change**: `cachicamas-ai-adapter-conformance`
> **Milestone**: AI-38 (doc 0002, lines 2289–2327) · **Wave**: 6 — Hand off
> **Artifact store**: hybrid (file + Engram) · **Explore**: [`explore.md`](./explore.md) · Engram `#2761`
> **Depends on**: AI-23, AI-29 … AI-37 · **Blocks**: AI-39, AI-40

## Intent

The AI-23 conformance suite has never been run, unscoped, against a real adapter. Its only
unscoped subject is AI-21's in-process fake. The OpenRouter bridge that shipped with
`add-openrouter-first-provider` runs two of five required capabilities through the **scoped**
`agenttest.RunConformanceFor`, whose own doc states a scoped verdict "is never presentable as
evidence of full conformance". The three remaining required capabilities are `t.Skip`'d.

So today the project cannot answer the question Wave 6 exists to answer: *does the shipped
adapter satisfy the contract every Layer-2 consumer will be told it satisfies?* Until it can,
AI-39's live smoke has nothing to compare against and AI-40 cannot publish a readiness claim.

Success: `agenttest.RunConformance` runs unscoped against the OpenRouter adapter over replayed
transcripts, every required case passes with no skip and no waiver, the generated nine-entry
capability record is asserted against a committed expectation, and every transcript is
regenerable from a real captured stream rather than hand-typed.

## Scope

### In scope

1. **Unscoped conformance run** — `agenttest.RunConformance(t, openRouterBridgeFactory())` becomes
   the acceptance gate. Every `t.Skip` in `openrouter/conformance/run_for_test.go` is removed.
2. **Defect resolution, both directions** — each failing case is resolved as a defect in the suite
   case *or* in the adapter, with the side that moved and the reason recorded. No waivers.
3. **Recording helper** — captures a real SSE stream into the exact fixture byte format the replay
   harness consumes; today's hand-typed fixtures are regenerated through it.
4. **Generated capability record** — a real suite run's `CapabilityRecord`, compared with
   `CompareCapabilityRecords` against a committed nine-entry expectation.
5. **`CapRetry` declaration parity** — the two adapter factories stop disagreeing.
6. **Integration-level boundary-replay matrix** — every conformance transcript replayed split at
   adversarial byte offsets with an identical suite verdict at every split point.
7. **Delta specs in this change** for any requirement that must move (see *Capabilities*).

### Out of scope

- Real vendor network calls in `make test` — AI-39's, and gated. The recording helper's capture
  mode is credential-gated and never runs in the default suite.
- `R-OR-07` / `R-OR-08` (live-smoke gating and sentinel sweep) and the orphan
  `openspec/changes/add-openrouter-first-provider/` directory carrying the revised `R-OR-07`.
  AI-39 reconciles and archives it.
- A second vendor, a reasoning-capable default model, or any Anthropic path.
- Editing `openspec/specs/**` directly. Canonical specs change only through this change's delta
  specs, promoted at archive time.

## Capabilities

### New capabilities

- `ai-adapter-conformance-run`: the deterministic end-to-end matrix itself — unscoped-run
  obligation, transcript regenerability, the committed capability-record expectation, and the
  integration-level boundary-replay sweep.

### Modified capabilities

- `ai-provider-conformance-suite`: cancellation-case admission shape (T2 below) and the
  scoped-vs-unscoped evidence rule made an explicit requirement rather than a doc comment.
- `ai-openrouter-first-provider`: `R-OR-05` / `R-OR-06` restated for a nine-entry record
  (`CAP-O-04` post-dates "absent × 3") and for regenerable transcripts.
- `ai-provider-error-mapping` **or** `ai-stream-lifecycle` — only if T2 resolves adapter-side.
- `ai-provider-completion` — only if T3 requires widening `R-ACP-002`'s unreachability set.

The last two are conditional on `sdd-design`'s verdict and MUST NOT be assumed by `sdd-spec`.

## Approach

Approach 1 from exploration (widen-and-reconcile), sequenced so that evidence precedes design
commitment:

| Step | Work | Milestone |
|---|---|---|
| A | Run the unscoped suite against the bridge; capture the real failure set as the RED baseline | AI-38.1 |
| B | Build the recording helper; regenerate every fixture from captured wire bytes | AI-38.1 |
| C | Resolve each failing case as a suite-side or adapter-side defect, with a recorded rationale | AI-38.1 |
| D | Generate the nine-entry capability record; assert it against a committed expectation | AI-38.2 |
| E | Boundary-replay sweep at the integration level, anchored so it cannot pass vacuously | AI-38.3 |

Step A's output is the binding input to `sdd-design`. The exact failure set is not knowable from
reading code alone — `capability_record_test.go`'s own comment records that an earlier unscoped
draft failed on five case families (cancellation, terminal, `finish_reason/refusal`,
`usage/absent_vs_zero`, redaction), which is two more than the three skipped drivers suggest.

## Tension resolutions

| # | Tension | Position |
|---|---|---|
| T1 | No-waivers charter vs. scoped bridge | Unscoped `RunConformance` is the gate. `RunConformanceFor` survives only as a debugging affordance and is never cited as acceptance evidence. Zero `t.Skip` in conformance drivers at close. |
| T2 | Cancellation semantics | Verified real conflict between two promoted specs: `R-CNF-011` / `R-CNF-012` assert a bare close with no invented terminal; `R-AEM-014` / `S-AEM-051…055` (AI-32.3, which amended AI-20.3) require a typed terminal on cancel/deadline. **Preferred direction: suite-side.** The suite over-specifies "bare close" as the *only* conformant cancellation shape; it should admit either a bare close or exactly one typed cancellation-category terminal and nothing else. Falsifier: if step A shows the adapter emits a terminal of the wrong category, or more than one, the adapter moves instead. `sdd-design` owns the final call against step A evidence. |
| T3 | Regenerable transcripts | A capture helper serialises a real stream into the fixture format, credential-gated exactly like AI-39's smoke. Committed fixtures MUST be byte-identical to the helper's output for the same captured wire bytes, proven by a drift guard, so a hand edit fails the suite. |
| T4 | Capability record | AI-38.2 asserts a **generated** record from an unscoped run, not declared factory bools. The expectation is restated over all nine entries: "absent × 3" predates `CAP-O-04`. `CapRetry` is declared once, consistently, and the value chosen is justified in the expectation table. A generated `CAP-O-01 = satisfied` **blocks the change** and escalates as AI-29 reopen trigger #1 — never a silent expectation update. |
| T5 | Boundary replay at integration level | New sweep asserting an identical `CapabilityRecord` (not merely identical decoded frames) at every split offset, plus a canonical anchor run pinned to an expected record so a uniformly-broken harness cannot pass vacuously. Reuse `checkSweepFixtureBound`'s size-bound shape; if the full offset cross-product exceeds the runtime budget, bound fixture size first and only then sample offsets, recording the bound. |
| T6 | Scope control | AI-38 owns `agenttest`, the two bridges, and the fixtures. Every canonical-behaviour change is an ADDED/MODIFIED delta requirement under `openspec/changes/cachicamas-ai-adapter-conformance/specs/`. Nothing under `openspec/specs/**` is edited in place. |

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/conformance/` | Modified | Skips removed; unscoped run; regenerated fixtures; boundary sweep |
| `backend/agent/src/ai/openaicompat/bridge_test.go` | Modified | `CapRetry` parity; whatever step C resolves adapter-side |
| `backend/agent/src/agenttest/conformance_*.go` | Modified | Cancellation-case admission (T2); any suite-side defect fix |
| `backend/agent/src/ai/openaicompat/` (non-test) | Conditional | Only if step C resolves a defect adapter-side |
| `openspec/changes/cachicamas-ai-adapter-conformance/specs/` | New | Delta specs for every requirement that moves |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| The real failure set is wider than the three skipped drivers | High | Step A runs before any design commitment; `sdd-design` is gated on its output |
| Resolving T2/T3 reopens promoted spec surfaces (error-mapping, completion, lifecycle) | Med | Confined to delta specs in this change; each reopen carries an explicit rationale row |
| 1000-line budget exceeded | High | See forecast below; `sdd-tasks` MUST re-forecast and may need work-unit slicing inside the single PR |
| Pre-existing scaffold does not compile or pass | Med | `sdd-apply` runs `make test` first and treats any failure as a defect in found code |
| Generated `CAP-O-01 = satisfied` | Low | Hard stop and escalate as AI-29 reopen trigger #1 |
| Boundary sweep runtime blows up the `-race` suite | Med | Fixture-size bound first, offset sampling second, both recorded |

## Size forecast (against the 1000-line budget)

| Work unit | Authored lines (est.) |
|---|---|
| Recording helper + drift guard | 150–300 |
| Suite-side cancellation admission + case updates | 80–150 |
| Terminal / in-band failure resolution | 120–250 |
| Finish-reason resolution | 60–150 |
| Capability-record generation + nine-entry expectation | 80–150 |
| Boundary-replay sweep | 150–300 |
| `CapRetry` parity | 20–40 |
| Delta specs (markdown) | 150–350 |

**Total 810–1690 authored lines. 1000-line budget risk: High.** Regenerated fixtures are generated
goldens, excluded from authored risk but included in snapshot identity. The pre-approved
AI-38-scoped `size:exception` covers the single-PR decision; `sdd-tasks` MUST still forecast
explicitly and MAY propose internal work-unit boundaries so review stays tractable.

## Rollback plan

The change is confined to the worktree branch. Revert the branch and the pre-existing 2026-08-06
scaffold is restored unchanged — it is committed on `main`, so nothing is lost by discarding this
work. Delta specs never touch `openspec/specs/**` before archive, so an abandoned change leaves
the promoted spec tree untouched. If only the boundary sweep proves unaffordable, it is
independently revertible: AI-38.1 and AI-38.2 do not depend on AI-38.3.

## Dependencies

- AI-23 (suite), AI-29 (reasoning struck), AI-35 (`CapRetry` / `CAP-O-04`), AI-27.3 (offset-sweep
  mechanism), AI-32.3 (typed terminal on cancel/deadline) — all shipped.
- Credentials for the recording helper's capture mode are supplied out of band; no credential is
  written inside the repository and `make test` never depends on one.

## Success criteria

- [ ] `agenttest.RunConformance` runs unscoped against the OpenRouter adapter and every required
      capability passes.
- [ ] Zero `t.Skip` remains in the conformance drivers; every resolved defect names which side
      moved and why.
- [ ] Every conformance fixture is produced by the recording helper, and a hand edit fails the
      drift guard.
- [ ] The generated nine-entry capability record equals the committed expectation via
      `CompareCapabilityRecords`; optional-capability outcomes are recorded, never unrun.
- [ ] Both adapter factories declare `CapRetry` identically.
- [ ] Every transcript replays split at adversarial boundaries with an identical record, and the
      canonical anchor run pins that record.
- [ ] `make test` green with `-race`; `make lint` clean; `go.mod` still declares zero `require`.
