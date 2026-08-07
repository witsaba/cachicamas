# Archive Report — `cachicamas-ai-backpressure`

> **Status**: **CLOSED** · **Verdict**: **PASS** · **Milestone AI-34: GREEN, ready for PR merge**
> **Date**: 2026-08-07
> **Archived to**: `openspec/changes/archive/2026-08-07-cachicamas-ai-backpressure/`
> **Artifact store**: hybrid (filesystem + Engram `sdd/cachicamas-ai-backpressure/archive-report`)

## Outcome

AI-34 locks Layer 1's backpressure invariant by measuring the carrier buffer against realistic workloads (text + tool-call transcripts × 5ms / 50ms / pause-200ms consumer profiles, 20 runs each) and applying the doc 0002:432 tie-break. Chosen capacity: **`N=0` (rendezvous)**, recorded in `decision.md` against the spec's "starting capacity of 64 events" hypothesis.

Three subnodes closed: AI-34.1 (measured decision), AI-34.2 (lossless ordering under pressure), AI-34.3 (no unsanctioned loss path across 6-case × 50 leak-check matrix). Three new requirements added (R-AIS-039, R-AIS-040); one amended (R-AIS-031, with the canonical spec.md line 387 wording now reflecting the measured `N`). Production change is +15/-1 on `stream.go` — function-equivalent at `N=0` but contract-binding.

The canonical `openspec/specs/ai-stream-lifecycle/spec.md` line 387 amendment is applied **at archive time** (deliberately deferred from the PR per the proposal and the AI-34 doc 0002 amendment). R-AIS-031 S-1 ("spec wording matches the decision artefact") is now satisfied by this amendment.

## Identity

| Field | Value |
|---|---|
| Change | `cachicamas-ai-backpressure` |
| Milestone | AI-34 — Lock backpressure and buffer behavior (doc 0002 § 2041–2073) |
| Wave | 5 — Harden (second milestone) |
| Branch | `feat/ai-34-backpressure` |
| Worktree | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-34` |
| Base | `main` @ `a78a941` |
| Module | `backend/agent` — layered, NOT hexagonal (ADR 0005 § D1) |
| Test runner | `cd backend/agent && make test` (`go test -race -v ./...`); measurement host: `go test -tags=measurement ./src/ai/openaicompat/...` |
| Capability amended | `ai-stream-lifecycle` |

## Commits

Six implementation commits + one metadata commit + one doc-amendment cherry-pick (8 total on the branch; the doc cherry-pick is duplicated on main). Subnode order (34.1 → 34.2 → 34.3) follows the proposal.

| SHA | Type | Subnode | Subject |
|---|---|---|---|
| `a110e84` | docs | 34.1 | `docs(ai): AI-34.1 decision.md skeleton` |
| `719489e` | test | 34.1 | `test(ai): measurement workload M1/M2/M3 + decision.md numbers` |
| `3b7deb4` | feat | 34.1 | `feat(ai): apply streamCarrierBuffer = 0` |
| `e621a84` | docs | 34.1 | `5-line doc-fix on a_i-33_3_test.go:28` |
| `9151234` | test | 34.2 | `dripFramesServer + lossless ordering under pressure` |
| `db768c7` | test | 34.3 | `exhaustive non-cancellation leak-check × 50` |
| `85cd936` | docs | — | `AI-34 apply-progress (mirrors Engram obs #2635)` |
| `8e69e4b` | docs | — | `doc 0002 amend with AI-34 close (Wave 5 — Harden)` *(also on `main` as `c0cda47`)* |

## Subnode summary

- **AI-34.1 — Measured buffer decision** (`decision`). M1/M2/M3 measured against realistic workloads. Tie-break lands at `N=0`. `const streamCarrierBuffer = 0` declared near `stream.go:123` alongside `streamReadBufferSize`. `make(chan ai.Event)` → `make(chan ai.Event, streamCarrierBuffer)` at `stream.go:223`. Documented in `decision.md` (96 lines).
- **AI-34.2 — Lossless ordering under pressure** (`leaf`). `cap(ch) == streamCarrierBuffer` direct assertion (RED-first at authoring; GREEN at commit time). Slow / bursty / pause-resume consumer profiles via `dripFramesServer` helper at `drip_frames_test.go`. No-auxiliary-queue observation.
- **AI-34.3 — No unsanctioned loss path** (`leaf`). 6-case matrix (slow/bursty/pause-resume × text/tool-call) × 50 repeats wrapped in `agenttest.RequireNoGoroutineLeak`. Serial-only via `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` (R-STK-008).

## Test gates

- `cd backend/agent && make test` → **PASS** (all 6 packages; 54s includes the leak-check × 50 × 6 cases)
- `cd backend/agent && make lint` → **0 issues** (`go vet` + `golangci-lint` v2.9.0)
- `cd backend/agent && go test -tags=measurement -v -run TestMeasurement_Workload_M1M2M3 ./src/ai/openaicompat/...` → **PASS** in 27.57s

## Files archived (this folder)

- `explore.md` — AI-34 explore phase artefact (Engram obs #2629)
- `proposal.md` — AI-34 proposal (Engram obs #2630)
- `specs/ai-stream-lifecycle/spec.md` — AI-34 delta spec (Engram obs #2632); introduces R-AIS-039 + R-AIS-040, modifies R-AIS-031
- `design.md` — AI-34 design (Engram obs #2631); D1-D5 settled (const location, helper signature, measurement mechanics, subnode order, decision.md shape)
- `tasks.md` — AI-34 task plan (Engram obs #2633); 22 tasks across 3 subnodes, Q2 answered (`//go:build measurement`)
- `decision.md` — AI-34.1 evidence artefact (chosen `N=0`, M1/M2/M3 numbers, tie-break reasoning, constant-vs-configurable)
- `apply-progress.md` — AI-34 apply-phase record (Engram obs #2635)
- `verify-report.md` — AI-34 verify report (this archive; documents spec delta coverage + test results + decision rationale + tooling deviations)
- `archive-report.md` — **this file** (Engram obs `<see session_summary>`)

## Spec amendment applied at archive

**Canonical** `openspec/specs/ai-stream-lifecycle/spec.md:387` amended from:

> The buffer between producer and consumer (`V-STR-08`) is **bounded**, with a **starting capacity of 64 events**.

to:

> The buffer between producer and consumer (`V-STR-08`) is **bounded**, with a **starting capacity of 0 events** *(measured by AI-34.1 against the workload recorded in `openspec/changes/archive/2026-08-07-cachicamas-ai-backpressure/decision.md`; tie-break per doc 0002:432 applied — "prefer the smaller; backpressure observable is worth more than latency that was hidden")*. Runtime behaviour at `N=0` is identical to the prior unbuffered carrier (`make(chan T, 0) == make(chan T)`) but the named constant + the explicit buffer argument are now part of the contract that R-AIS-031 and R-AIS-039 lock down.

This satisfies R-AIS-031 S-1 ("spec wording matches the decision artefact"). R-AIS-031 S-2 ("capacity re-confirmed by `sdd-verify` at archive time") is satisfied by `verify-report.md`'s PASS verdict.

## Spec delta reference

`openspec/changes/2026-08-07-cachicamas-ai-backpressure/specs/ai-stream-lifecycle/spec.md` documents:
- ADDED: R-AIS-039 (capacity equals measured N); R-AIS-040 (no unsanctioned loss path × 6-case × 50 leak-check matrix)
- MODIFIED: R-AIS-031 (line 387 wording change, `(Previously: ...)` annotation)
- Unchanged sections: § 6 corollaries (395–399), § 6 measurement tables (422–430), § 6 tie-break rule (432)
- 8 scenarios in total (3 + 3 + 2)

## Conformance suite — unchanged (correctly)

`backend/agent/src/agenttest/conformance_cancellation.go:46-131` (R-CNF-011 + R-CNF-012) still uses `Buffer: 0` in the fake provider's `Script`. Its drop-physics proof is unaffected — saturated-buffer cancel-during-blocked-send remains the only sanctioned loss path (AI-20.3). The buffered real producer satisfies R-CNF-011/012 unchanged. **No conformance amendment is required.**

## Dependency surface — unchanged

- `backend/agent/go.mod` is unchanged.
- No new dependencies. Stdlib-only per ADR 0005 § D1 row 1.
- Layer 1 imports nothing outside the Go standard library, `net/http`, and the OpenTelemetry API (per ADR 0005 § D1 / § D3 — AI-37's OTel API gate not yet opened).

## Doc 0002 amendment

`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 3 and 18 amended at AI-34 close (commit `c0cda47` on main, `8e69e4b` cherry-picked to the branch):
- Top status: shipped counter 34 → **35 of 42**; landed AI range AI-00..AI-33 → **AI-00..AI-34**; file count 85 prod / 139 test → **85 prod / 143 test**; "Remaining" 8 → **7 milestones**.
- New amendment blockquote dated 2026-08-07: full AI-34 close record (subnodes, production change, commits, evidence, three acknowledged deviations).

## Deviation record

Three acknowledged deviations documented in `apply-progress.md`:

1. **Size deviation** — aggregate diff +1241/-2 lines vs ~600 forecast (~107% over); 90% of overage is test code (the measurement + capacity-assertion + exhaustive-path evidence). `size:exception` accepted by maintainer (Engram obs #2634). PR #130 also exceeds the 1000-line review budget by ~241 lines.
2. **Doc-vs-code divergence reconciled on the record** — spec § 6 line 387 said "starting capacity of 64 events" as a hypothesis (AI-02.1); implementation has been unbuffered since AI-28.1.1 with no commit ever introducing a buffer argument. AI-34.1 records the divergence explicitly and lands the first measured number against a real workload, applying the AI-33 living-graph precedent (spec amendment applied at this archive).
3. **Formal verify/archive cycle had tooling obstacles** — Gentle AI runtime ledger returns `state: complete` after the apply settle (`passed`) and blocks subsequent `sdd-attempt acquire` for the same change; `gentle-ai sdd-status` dispatcher requires a bounded review transaction (the reviewer sub-agent rejected the `GENTLE_AI_REVIEW_BINDING` prompts — binding format strict); user disabled receipt-driven development (`gentle-ai review mode disable`) to bypass; PR opened manually; tooling discoveries saved to Engram obs #2636. **This archive-report and `verify-report.md` are the manual substitutes for the formal `sdd-verify` and `sdd-archive` sub-agent runs**; the orchestrator did the archive work directly because the formal sub-agent path was unreachable.

## Per-record evidence

- **Branch**: `feat/ai-34-backpressure` @ `8e69e4b` (pushed to origin)
- **PR**: #130 (https://github.com/witsaba/cachicamas/pull/130) — OPEN, ready for review and merge
- **Doc 0002 amendment**: `c0cda47` on main (also cherry-picked as `8e69e4b` on the feature branch)
- **Engram observations**: `#2629` (explore), `#2630` (proposal), `#2631` (design), `#2632` (spec), `#2633` (tasks), `#2634` (decisions/size:exception), `#2635` (apply-progress), `#2636` (tooling-discoveries)

## Status

**22/22 tasks complete. 8 scenarios satisfied. 0 CRITICAL. 0 FAIL. Decision `N=0` recorded. Spec amended. Doc 0002 amended. PR ready to merge.**

---

**Subsequent Wave 5 milestones** (per doc 0002 § 220 and the AI-33 / AI-34 amendment "Remaining in Wave 5"):
- AI-35 — Define retry and idempotency policy
- AI-36 — Enforce secret redaction
- AI-37 — Add the observability boundary (the OTel API gate per ADR 0005 § D3)
- AI-41 — Discharge the Wave-2 carryovers

And Wave 6 (AI-38 full deterministic adapter conformance, AI-39 opt-in live smoke, AI-40 Layer 2 readiness contract).
