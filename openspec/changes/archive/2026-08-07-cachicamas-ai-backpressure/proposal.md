# Proposal: Lock backpressure and buffer behavior

> **Change**: `cachicamas-ai-backpressure`
> **Milestone**: AI-34 (doc 0002 lines 2041–2073)
> **Wave**: 5 — Harden
> **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal)
> **Strategy**: single-pr · strict TDD on · stdlib-only · ~600 lines aggregate

## Why

Doc 0002 chartered AI-34 to convert AI-02.1's "starting capacity of 64 events" hypothesis into a measured number, lock the backpressure invariant, and prove no unsanctioned loss path exists beyond the AI-20.3 / AI-33.3 saturated-drop exception. The code (`backend/agent/src/ai/openaicompat/stream.go:223`) has been unbuffered since AI-28.1.1; the spec (`openspec/specs/ai-stream-lifecycle/spec.md:387`) still says 64. AI-34 reconciles that drift with measurements, applies the number, and proves lossless ordering under pressure.

## Scope

### In scope
- One-line change at `backend/agent/src/ai/openaicompat/stream.go:223`: `make(chan ai.Event)` → `make(chan ai.Event, streamCarrierBuffer)`.
- New package-private constant `const streamCarrierBuffer = N` (value `[measured by AI-34.1]`; name `[proposed]` — `sdd-spec`/`sdd-design` own final naming) at `backend/agent/src/ai/openaicompat/stream.go` package scope, alongside `streamReadBufferSize` at line 123.
- NEW test file `backend/agent/src/ai/openaicompat/a_i-34_2_test.go` — capacity assertion + slow-consumer ordering + no-auxiliary-queue (RED-first `cap(ch) == N` against the real producer).
- NEW test file `backend/agent/src/ai/openaicompat/a_i-34_3_test.go` — exhaustive non-cancellation paths (slow / bursty / pause-resume × text / tool-call) × `RequireNoGoroutineLeak` × 50 repeats; serial-only via `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` (`backend/agent/src/agenttest/stream_kit_leak.go:107`).
- NEW test file `backend/agent/src/ai/openaicompat/measurement_test.go` — M1 / M2 / M3 workload host (build tag vs `-short` vs `-bench` deferred to `sdd-tasks`).
- NEW `dripFramesServer(transcript, gap)` helper (~15 lines, test-only, `package openaicompat`) — real-producer slow-consumer pattern. The only new helper in the change. Must be in the internal package so AI-34.2 can reference the package-private `streamCarrierBuffer` constant directly.
- 5-line doc-comment fix at `backend/agent/src/ai/openaicompat/a_i-33_3_test.go:28` — `stream.go:209` → `stream.go:223`.
- NEW `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` — AI-34.1 deliverable shape per explore § 4.5 (workload, numbers, tie-break rule, chosen capacity).

### Out of scope (deferred)
- `openaicompat.Config.BufferCapacity` field (Q3: stay constant until a workload demands configurability).
- `openspec/changes/2026-08-07-cachicamas-ai-backpressure/specs/ai-stream-lifecycle/spec.md` amendment (do as follow-up after AI-34.1 measures the number).
- OpenRouter wrapper Config surface (it's a one-line forwarder at `backend/agent/src/ai/openaicompat/openrouter/wrapper.go:84–86`; no buffer knob).
- Multiplexer wrappers, multimodal beyond text, embeddings, batch APIs.
- Any conformance suite amendment (R-CNF-011/012 at `backend/agent/src/agenttest/conformance_cancellation.go:46–131` prove drop physics, not capacity — they survive unchanged).
- AI-20.3 sanctioned loss path — AI-33.3 owns it, untouched.

## Scope decisions (locked at propose time)

| # | Decision | One-line rationale |
| --- | --- | --- |
| Q1 | **Implementation oversight** posture for spec 64 vs code 0 | AI-33.3 living-graph precedent; AI-34.1 records the measured number against the spec's hypothesis explicitly. |
| Q3 | **Package-private constant** (not `openaicompat.Config` field) | Mirrors `streamReadBufferSize` (line 123); matches `Config`'s "no bound value" posture (`client.go:41`); promotion deferred until a real workload demands configurability. |
| Q2 | **Workload host** (build tag vs `-short` vs bench) | Deferred to `sdd-tasks` — not blocking for propose. |

## Capabilities (contract with sdd-spec)

### New Capabilities
- **None.** AI-34 measures and applies an existing capability's quantitative parameter; no new spec is born.

### Modified Capabilities
- `ai-stream-lifecycle` — § 6 (lines 381–451) line 387 amendment: "starting capacity of 64 events" → measured value once AI-34.1 lands. The § 6 measurement tables (M1/M2/M3, tie-break, lines 422–432) stay verbatim; only the line 387 number changes. Deferred to archive phase — see "Spec amendment follow-up" below.

## Approach

Subnode execution order, all in one PR:

1. **AI-34.1 first** — measurement workload + 1-line production change. The `make(chan ai.Event, N)` must land before AI-34.2's `cap(ch) == N` assertion is RED-first against the new `N` (not the old `0`). Decision artefact `decision.md` records workload, numbers, tie-break, chosen capacity.
2. **AI-34.2 next** — capacity assertion + slow-consumer ordering + no-auxiliary-queue. The `dripFramesServer` helper lands here, not earlier.
3. **AI-34.3 last** — exhaustive non-cancellation paths × `RequireNoGoroutineLeak` × 50. Reuses AI-22.4 leak-check seam (`backend/agent/src/agenttest/stream_kit_leak.go:107`) and AI-33.3's "bare close" property.

**Strict TDD mode on** (per `openspec/AGENTS.md`): every behavior leaf's `tasks.md` will use RED-first test → GREEN implementation → REFACTOR confirmation. AI-34.2's `cap(ch) == N` is the canonical RED-first example — written against the unbuffered carrier first (RED), the 1-line buffer argument lands second (GREEN).

**Test file naming**: `a_i-34_{1,2,3}_test.go` (matches AI-33's `a_i-33_{1,2,3,4,5a,5b}_test.go` convention). Internal `package openaicompat` for 34.2 / 34.3; measurement host sits in the same package. Serial-only enforcement for AI-34.3 via `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` — no `t.Parallel()` in that file.

**Inherited decisions (closed, not reopened):** AI-02.1 carrier = channel (obs #2267); AI-20.3 sanctioned loss path; AI-22.4 leak-check seam; AI-32.3 bounded-wait terminal (`emitFailureSendBound = 5s` at `stream.go:129`); AI-33 cancellation contract.

**Why backpressure means wait, not drop**: doc 0003 AG-10.3 (lines 527–532) — the agent loop suspends calls for permission; the producer must wait on a `select`-bound send while ctx remains cancellable. Channels stay.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/openaicompat/stream.go:223` | Modified | Carrier construction: unbuffered → buffered with `streamCarrierBuffer`. |
| `backend/agent/src/ai/openaicompat/stream.go` (new const) | New | `const streamCarrierBuffer = N` at package scope (name `[proposed]`). |
| `backend/agent/src/ai/openaicompat/a_i-34_2_test.go` | New | Capacity + slow-consumer + no-auxiliary-queue leaf tests. |
| `backend/agent/src/ai/openaicompat/a_i-34_3_test.go` | New | Exhaustive non-cancellation paths × leak-check. |
| `backend/agent/src/ai/openaicompat/measurement_test.go` | New | M1 / M2 / M3 workload host. |
| `backend/agent/src/ai/openaicompat/a_i-33_3_test.go:28` | Modified | 5-line doc-comment drift fix: `stream.go:209` → `stream.go:223`. |
| `backend/agent/src/ai/openaicompat/` (new helper) | New | `dripFramesServer` ~15 lines, test-only, `package openaicompat`, real-producer slow-consumer pattern. |
| `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` | New | AI-34.1 measurement evidence + chosen capacity (sibling of `proposal.md`). |

## Risks (ported from explore § 5)

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| 5.1 — Doc-vs-code divergence (spec 64, code unbuffered since AI-28.1.1) is silently lost | Med | `decision.md` (Q1 implementation-oversight posture) names the divergence, the workload, the numbers, the tie-break rule, and the chosen capacity. |
| 5.2 — Fake `Script.Buffer` (`fake_script.go:51`) is decoupled from real producer capacity | Med | AI-34.2's `cap(ch)` assertion lives on the **real producer** (`package openaicompat`), not in the conformance suite. Conformance cases (`conformance_cancellation.go:46–131`) stay `Buffer: 0`. |
| 5.4 — Real-producer slow-consumer helper does not exist | High | Author `dripFramesServer` (~15 lines, stdlib-only: `httptest`, `io`, `time`) in AI-34.2's diff; analogous to `slowSSEServer` at `stream_test.go:106–122`. |
| 5.5 — AI-33 test doc-comment cites drifted line number | Low | 5-line edit lands in AI-34.1's diff alongside the production change. AI-33 tests do not assert `cap(ch) == 0`; no RED/GREEN impact. |
| 5.6 — Tie-break rule could land capacity at 0 (rendezvous) | Low | In scope. AI-34.1's deliverable is "with measurements, not a preference". If measurements say 0, the artefact says 0 and `decision.md` records the rationale. |
| 5.7 — `RequireNoGoroutineLeak` requires serial execution | Low | `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")`; no `t.Parallel()` in `a_i-34_3_test.go`. |
| 5.8 — Aggregate ~600 lines exceeds 400-line PR budget | Med | Single-PR strategy approved at session preflight; within 1000-line review budget. `sdd-tasks` may recommend split for review focus if test file exceeds 400 lines alone. |

## Rollback Plan

Revert the single PR. The 1-line production change is revertible in isolation; tests `a_i-34_{2,3}_test.go` and `measurement_test.go` can be dropped without code change. `decision.md` is preserved in `openspec/changes/2026-08-07-cachicamas-ai-backpressure/` as the historical record — no rollback action. Spec amendment (deferred) is independent and can be skipped without affecting code.

If the measured number turns out to be wrong in production (e.g., p99 wait > 1 ms across all consumer profiles post-merge), the rollback is to re-set the constant to 0 (rendezvous) and reopen the spec amendment — this is exactly the spec's own living-graph clause (`spec.md:381–451` preamble).

## Dependencies

- **None.** Stdlib only (`httptest`, `io`, `time`, `runtime`). ADR 0005 § D1 row 1 and § D3 are binding — no new top-level Go dependency. `dripFramesServer` is stdlib.

## Success Criteria

- [ ] `backend/agent/src/ai/openaicompat/stream.go:223` reads `make(chan ai.Event, streamCarrierBuffer)`.
- [ ] `streamCarrierBuffer` is a package-private constant at the same package scope, with the value recorded in `decision.md` (workload, numbers, tie-break).
- [ ] `a_i-34_2_test.go` asserts `cap(ch) == N` against the real producer and passes lossless ordering under 5 ms / 50 ms / pause-200 ms consumer profiles.
- [ ] `a_i-34_3_test.go` proves no unsanctioned loss path across slow / bursty / pause-resume × text / tool-call, with `RequireNoGoroutineLeak` × 50.
- [ ] `a_i-33_3_test.go:28` doc-comment cites `stream.go:223` (line number drift fixed).
- [ ] `decision.md` records the doc-vs-code divergence explicitly (Q1).
- [ ] `cd backend/agent && make test` green; no new top-level Go dependency introduced.
- [ ] Spec amendment to `openspec/specs/ai-stream-lifecycle/spec.md:387` flagged as follow-up owned by archive phase.

## Spec amendment follow-up (explicit)

`openspec/specs/ai-stream-lifecycle/spec.md:387` reads "starting capacity of 64 events" today. Once AI-34.1's measurements land, **this line is amended to the measured value** (e.g. "starting capacity of N events, per `decision.md`"). This amendment is a separate change, owned by AI-34's archive phase (`sdd-archive`). It is NOT in this PR. The `sdd-archive` step for this change carries the spec delta forward.

---

## References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2041–2073`
- **Explore artefact** — `openspec/changes/2026-08-07-cachicamas-ai-backpressure/explore.md` (~340 lines, canonical record)
- **Engram observation** — `#2629` (topic_key `sdd/cachicamas-ai-backpressure/explore`)
- **Spec § 6 (buffering)** — `openspec/specs/ai-stream-lifecycle/spec.md:381–451`
- **Conformance R-CNF-011/012** — `openspec/specs/ai-provider-conformance-suite/spec.md:163–181`; implementation `backend/agent/src/agenttest/conformance_cancellation.go:46–131`
- **Carrier construction** — `backend/agent/src/ai/openaicompat/stream.go:223`
- **Backpressure primitive** — `backend/agent/src/ai/openaicompat/stream.go:574–582`
- **Bounded-wait seam (AI-32.3)** — `backend/agent/src/ai/openaicompat/stream.go:129` and `:469–482`, `:530–540`, `:679–682`
- **Leak-check seam (AI-22.4)** — `backend/agent/src/agenttest/stream_kit_leak.go:107`
- **Drain helper** — `backend/agent/src/agenttest/stream_kit_record.go:63`
- **Real-HTTP test pattern** — `backend/agent/src/ai/openaicompat/bridge_test.go:64–92, 98–115`; `stream_test.go:50–60`
- **OpenRouter wrapper (forwarder)** — `backend/agent/src/ai/openaicompat/openrouter/wrapper.go:84–86`
- **ADR 0005 § D1, § D3** — `docs/adr/0005-promote-agent-stack-to-own-module.md`
- **Layer 2 consumer-pause-by-design** — `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:527–532` (AG-10.3)
- **AI-02.1 carrier decision** — Engram obs `#2267`; `openspec/specs/ai-stream-lifecycle/spec.md:381–438`
- **AI-33 cancellation contract** — PR #129, merged 2026-08-07
