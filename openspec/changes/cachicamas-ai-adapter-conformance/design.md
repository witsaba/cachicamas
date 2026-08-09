# Design — AI-38: Run full deterministic adapter conformance

> **Change**: `cachicamas-ai-adapter-conformance` · **Milestone**: AI-38 (Wave 6)
> **Inputs**: proposal.md (Engram #2762), explore.md (#2761), locked maintainer decisions (#2763)
> **Store**: hybrid · Engram `sdd/cachicamas-ai-adapter-conformance/design`

## Technical Approach

Approach 1 (widen-and-reconcile), constrained by the four locked decisions. The unscoped
`agenttest.RunConformance` run against the OpenRouter conformance bridge becomes the acceptance
gate; each of the five recorded failing families is resolved on a named side (suite or bridge —
the adapter itself moves nowhere). Fixtures migrate to raw `testdata/*.sse` goldens produced by a
byte-transparent recording helper; the capability record is generated, not declared; the AI-27.3
offset sweep is lifted to full-stack replay with a bounded cost model.

**Verified code facts this design rests on** (worktree, 2026-08-09):
- openaicompat already decodes mid-stream in-band error frames (`stream_failure.go`,
  `isInBandErrorFrame`/`failureFromErrorFrame`, R-AEM-010) — but ALWAYS as
  `FailureCategoryUnknown` + preserved `RawLabel`. Per-category mid-stream failures are not
  wire-expressible on this dialect.
- ctx-cancel mid-stream emits an optional block-closing `TextBlockEnd` then exactly one typed
  cancellation terminal (`stream.go:749-771`, R-AEM-014).
- `renderScript` (both bridge copies) drops `Usage` on the terminal chunk and `tb.Fatalf`s on
  `EventKindError` — the direct cause of the usage and terminal/redaction family failures.
- The CapRetry case body lives in `package openaicompat_test` and registers only in the
  openaicompat test binary; the `openrouter_conformance` binary never sees it, so an unscoped run
  there with `Retry=&true` would end `NotExercised` → `VerdictInconclusive`.

## Architecture Decisions

### D1 — Unscoped harness entry lives in `openrouter_conformance` (AI-35 precedent)

**Choice**: One serial driver `TestOpenRouterAdapter_FullConformance` in
`openrouter/conformance/run_for_test.go`: `record := agenttest.RunConformance(t,
conformanceBridgeFactory())`, then `CompareCapabilityRecords(record, expectedOpenRouterRecord())`
must return nil diffs. All three `t.Skip` drivers are deleted; the per-capability
`RunConformanceFor` drivers may remain as debugging affordances but are never acceptance
evidence (T1). The driver MUST NOT call `t.Parallel()` — cancellation cases use
`RequireNoGoroutineLeak` (`tb.Setenv` panics under a parallel ancestor).
**Alternatives**: drive from `openaicompat`'s own bridge (rejected: R-OR-06 names the OpenRouter
adapter); a new package (rejected: the bridge factory and its case bodies already live here).
**Rationale**: case body in the package it exercises; the factory seam is already the suite's one
door (R-CNF-001).

### D2 — CapRetry case body extracted to an importable test-support package

**Choice**: Move `retryAutoRetryUpToBoundCase` (+ helpers) from
`openaicompat/conformance_retry_test.go` into a new non-test package
`openaicompat/conformancetest` that registers via `agenttest.RegisterConformanceCase` in its
`init()`. `openaicompat_test` keeps a thin `TestRetryCaseBody_RunsDirectly` importing it;
`openrouter_conformance` imports it (blank import suffices) so the unscoped run drives CAP-O-04.
Both bridge factories flip `Retry` to `&true` (locked decision 2 — honest: the client auto-retries
per AI-35).
**Alternatives**: duplicate the ~250-line body in `openrouter_conformance` (rejected: two copies
of one contract); leave `Retry=&false` (rejected by locked decision 2: the record must report
SATISFIED).
**Rationale**: without registration in the OpenRouter binary, CAP-O-04 is structurally
`NotExercised` and the verdict inconclusive. Precedent for non-test test-kit packages: `agenttest`
itself, `openrouter/smoke`. `import_boundary_test.go` allowlist admissibility is verified at
apply.

### D3 — Recording helper: raw-byte capture + `testdata/*.sse` goldens + round-trip drift guard

**Choice**: Fixtures migrate from hand-typed Go string literals (`fixtures/*.go`) to raw SSE byte
files `openrouter/conformance/testdata/*.sse`, exposed through the existing `fixtures` package
accessors via `go:embed`. The helper is test-only code in `openrouter_conformance`:
`recordTranscript(tb, endpointURL, credential, requestBody) []byte` — one real HTTP POST, raw
`io.ReadAll` of the SSE body, zero parsing, zero normalization (byte-transparent by
construction). Regeneration test: render each canonical script with the bridge renderer, serve it
from `httptest.Server`, capture through `recordTranscript` over real HTTP, and (a) with
`UPDATE_CONFORMANCE_FIXTURES=1`, write `testdata/*.sse`; (b) otherwise assert byte-identity with
the committed file — this assertion IS the drift guard: any hand edit fails `make test` (T3).
Real-vendor capture is only the same function pointed at a credential-gated endpoint — wired by
AI-39, never in this change (locked decision 3: local endpoint only, zero vendor network).
**Alternatives**: generate Go source literals (rejected: a source generator is more authored code
than an embed loader and fights the repo's golden-file idiom); SSE-parsing/normalizing recorder
(rejected: any transform reintroduces the drift the helper exists to kill).
**Rationale**: raw bytes make "committed fixture == helper output for the same wire bytes"
provable with `bytes.Equal`; goldens are excluded from authored risk but stay in snapshot
identity (§E).

### D4 — Cancellation: suite-side amendment of R-CNF-011/R-CNF-012 (LOCKED)

**Choice**: A new shared helper `requireCancellationTail(t, rec)` in
`conformance_cancellation.go` replaces the assertions at :79-81 and :121-128. Admissible drained
tail after cancel: **either** a bare close **or** exactly one `ai.ErrorEvent` whose
`Category() == ai.FailureCategoryCancellation`, in final position, optionally preceded by the
block-closing end events for blocks opened before cancellation (the adapter's mandated
valid-stream closure, S-AEM-051…055). Still rejected, unchanged: any `Completion`, more than one
error terminal, a terminal of any other category, or any event after the terminal. Nothing else
is relaxed. AI-21's fake closes bare → tail `[]` → still green; the suite's own
`conformance_suite_test.go` subjects are unaffected.
**Falsifier (T2)**: if the RED baseline shows the adapter emitting a wrong-category or second
terminal, the adapter moves instead — hard stop, escalate.
**Rationale**: R-CNF-011/012 vs R-AEM-014 is a real promoted-spec conflict; the maintainer locked
the suite side. Delta spec: MODIFIED `ai-provider-conformance-suite` R-CNF-011/R-CNF-012.

### D5 — Dialect-aware admission for finish reasons and mid-stream categories (LOCKED for
finish reasons; same mechanism generalized)

**Choice**: `agenttest.Factory` gains one optional field:

```go
// Dialect declares wire-dialect expressiveness limits. nil = fully
// expressive (the fake); declarations are proven, never trusted.
Dialect *DialectConstraints
type DialectConstraints struct {
    UnreachableFinishReasons  []ai.FinishReason   // e.g. Refusal, PauseTurn, Unknown
    MidStreamCategoryCollapse *ai.FailureCategory // e.g. Unknown for label-preserving in-band dialects
}
```

- `finishReasonExhaustivenessCase`: reachable reasons keep today's assertions byte-identical.
  Each declared-unreachable reason asserts the **negative**: scripting it yields exactly one
  typed failure terminal and no Completion (the strict gate rejects it as typed malformed) —
  "unsatisfiable-and-unviolated" (AI-29/AI-31 precedent), proven, not skipped. Drift guard and
  hand-list untouched; declared set must be a subset of the hand-list.
- `terminalFailureCategoryExhaustivenessCase` mid-stream half: with a declared collapse, every
  category's scripted terminal must still arrive as exactly one typed terminal, with
  `Category() == *MidStreamCategoryCollapse`; the pre-stream half (pure construction) is
  unchanged. `requireFailureCategoryCoverage` counts collapse-exercised categories as covered.
- OpenRouter bridge declares `{Refusal,PauseTurn,Unknown}` unreachable and collapse `Unknown`
  (matches `failureFromErrorFrame`'s shipped hardcoded category). No R-ACP-002 reopen (locked
  decision 4).
**Alternatives**: nested `t.Skip` per unreachable value (rejected: waiver-shaped, asserts
nothing); widening the adapter's strict gate (rejected: reopens R-ACP-002, explicitly excluded);
per-case bespoke Factory fields (rejected: two fields today, N tomorrow — one declaration struct
scales). **Rationale**: one seam serves both amended cases; `nil` keeps every existing factory
and the fake bit-for-bit green. Delta spec: MODIFIED `ai-provider-conformance-suite`
(R-CNF-010 mid-stream scenario, R-CNF-016 finish-reason scenario). The `Dialect` Factory field
needs no R-CNF-002 amendment: promoted R-CNF-002 scopes only the capability declaration, and repo
precedent (R-CNF-013's `Sentinel`, R-CNF-019 "extending R-CNF-002") adds Factory fields through
the requirement owning the behavior, never by modifying the seam's own requirement.

### D6 — Terminal, usage and redaction families resolve bridge-side

**Choice**: Extend the OpenRouter bridge's `renderScript` only (openaicompat's copy stays
scoped to text+tools):
- `EventKindError` → one in-band error frame `data: {"error":{"type":<fixed-label-per-category>,
  "message":<unwrapped cause text + request id>}}` and stop rendering (terminal). Sentinels
  scripted into `Cause`/`RequestID` land ON the wire; the adapter's own paths must keep them out
  of every rendering — the redaction case now tests the adapter, not the fake. `RawLabel` is
  sanctioned; the sentinel never goes into `type`.
- `EventKindCompletion` → `writeTerminalChunk` gains a `usage` object rendering **present fields
  only** (`ai.Tokens` absent ⇒ key omitted; zero ⇒ `0` emitted), resolving
  `usage/absent_vs_zero_distinguishable`.
This makes `terminal/exactly_one` (kinds-only), `terminal/partial_output_discriminator`
(PartialOutput-only) and `redaction/*` (leak-scan-only) pass against the real adapter with no
adapter change; the category-asserting exhaustiveness case is covered by D5.
**Alternatives**: adapter-side in-band category vocabulary (rejected: reopens AI-32's
label-preserving design and R-AEM-010); suite-side weakening of kinds/discriminator assertions
(rejected: nothing is wrong with them — the bridge simply couldn't render the scripts).

### D7 — Capability record: generated, nine entries, committed expectation

**Choice**: `expectedOpenRouterRecord()` built via `NewCapabilityRecordForTest` +
`SetOutcomeForTest`: CAP-R-01…05 `Satisfied`; CAP-O-01/02/03 `Absent`; CAP-O-04 `Satisfied`.
The driver asserts nil diffs from `CompareCapabilityRecords` AND `record.Verdict() ==
VerdictPass`, replacing `capability_record_test.go`'s pointer-only declaration checks (those
shrink to factory-shape sanity). A generated `CAP-O-01 = Satisfied` produces a diff whose failure
message names AI-29 reopen trigger #1 and hard-stops the change — never an expectation edit
(T4). "absent × 3" phrasing is superseded by the nine-entry table (locked decision 2). Delta
spec: MODIFIED `ai-openrouter-first-provider` R-OR-05/R-OR-06.

### D8 — Boundary replay at integration level: exhaustive event-level + sampled record-level

**Choice**: `openrouter/conformance/boundary_sweep_test.go`, two tiers over every committed
`testdata/*.sse` transcript:
1. **Event-level, every offset** (0…len, inclusive): one shared `httptest.Server` per transcript
   whose handler writes `bytes[:k]`, `Flush()`, `bytes[k:]`; a real `*openaicompat.Client`
   drains; drained events must equal the canonical unsplit replay. At least one transcript's
   canonical replay is anchored against a hard-coded expected event list (the AI-27.3 vacuity
   guard, lifted).
2. **Record-level, sampled offsets** `{1, len/2, len-1}`: a split-serving Factory variant runs
   the full `RunConformance`; the returned record must equal `expectedOpenRouterRecord()` at
   every sampled offset.
Fixture size bound: every swept transcript ≤ 1024 bytes, inclusive, enforced by a local
`checkConformanceSweepBound` (same shape as `checkSweepFixtureBound`, which is unexported and
test-local in `openaicompat_test` — a bounded, recorded duplication). Runtime budget: target
< 30 s added under `-race`; if exceeded, offsets are sampled for transcripts > 512 B and the
bound is recorded in the delta spec (T5's authorized fallback: bound size first, then sample).
**Alternatives**: full suite × every offset (rejected: ~10³ suite runs, unaffordable under
`-race`); decoder-level only (rejected: T5 demands suite-verdict equivalence, and AI-27.3
already owns the decoder level).

## Data Flow

```
Script ──renderScript──► SSE bytes ──httptest.Server──► real *openaicompat.Client
   ▲                        │  ▲                              │
   │              recordTranscript (raw io.ReadAll)      ai.Events ──► suite cases
canonical scripts           ▼  │                              │
                      testdata/*.sse ◄──UPDATE=1──┘     CapabilityRecord ──CompareCapabilityRecords──► committed expectation
                      (drift guard: bytes.Equal)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `backend/agent/src/agenttest/conformance_cancellation.go` | Modify | D4 `requireCancellationTail`; amended R-CNF-011/012 assertions |
| `backend/agent/src/agenttest/conformance_suite.go` | Modify | D5 `Dialect *DialectConstraints` field + type |
| `backend/agent/src/agenttest/conformance_capabilities.go` | Modify | D5 finish-reason dialect branch |
| `backend/agent/src/agenttest/conformance_terminal.go` | Modify | D5 category-collapse branch + coverage accounting |
| `backend/agent/src/agenttest/conformance_suite_test.go` (+leaf tests) | Modify | Suite self-tests for D4/D5 branches; fake stays green |
| `backend/agent/src/ai/openaicompat/conformancetest/` (new pkg) | Create | D2 extracted CapRetry case body + init registration |
| `backend/agent/src/ai/openaicompat/conformance_retry_test.go` | Modify | Shrinks to thin driver importing `conformancetest` |
| `backend/agent/src/ai/openaicompat/bridge_test.go` | Modify | `Retry` → `&true` (D2) |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go` | Modify | `Retry` → `&true`; D5 dialect declaration; D6 error-frame + usage rendering |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/run_for_test.go` | Modify | D1 unscoped serial driver; three `t.Skip` drivers deleted |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/capability_record_test.go` | Modify | D7 generated-record assertion replaces pointer checks |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/recorder_test.go` | Create | D3 `recordTranscript` + regenerate/drift-guard test |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/testdata/*.sse` | Create | Recorded goldens (generated; excluded from authored count) |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures/*.go` | Modify | String literals → `go:embed` of testdata; accessors kept |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/boundary_sweep_test.go` | Create | D8 two-tier sweep + size bound + anchor |
| `openspec/changes/cachicamas-ai-adapter-conformance/specs/**` | Create | Delta specs (D4, D5, D7, new `ai-adapter-conformance-run`) |

## Testing Strategy (strict TDD — RED first)

| Step | RED | GREEN |
|------|-----|-------|
| A (baseline) | Commit the unscoped driver first; run `make test`; record the exact five-family failure set as evidence | — (this RED is the design's binding input; deviations from the predicted set escalate before any resolution lands) |
| D4 | bounded_close/abandoned fail against bridge with typed terminal | suite amendment; fake re-run proves no regression |
| D5 | finish_reason refusal/pause_turn/unknown + category mid-stream fail | dialect declarations + amended cases |
| D6 | usage/absent_vs_zero + terminal kinds + redaction fail (`renderScript` Fatalf) | error-frame + usage rendering |
| D3 | drift guard fails against a deliberately hand-edited byte | recorded goldens committed |
| D8 | anchor test with empty expected list fails (vacuity guard proven) | sweep green at every offset |
| Suite self-tests | new D4/D5 branches proven against hand-built leaking/violating doubles (pure helpers, no real subtest) | — |

Runner: `make test` (repo root, `-race`); `make lint`; `go.mod` stays zero-`require`.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or
process-integration boundary. All new code is test-only plus one test-support package.

## Work Units / Size Forecast (1000-line budget; `size:exception` pre-approved)

| WU | Content | Authored est. |
|----|---------|---------------|
| 1 | Unscoped driver + RED baseline evidence | 60–100 |
| 2 | D3 recorder + testdata migration + drift guard | 200–300 |
| 3 | D4 cancellation amendment + suite self-tests | 120–180 |
| 4 | D5 dialect seam + two case amendments + self-tests | 180–280 |
| 5 | D6 bridge rendering (error frame + usage) | 120–200 |
| 6 | D2 retry extraction (mostly moved lines) + parity flips | 80–150 moved+new |
| 7 | D7 record assertion + expectation | 80–130 |
| 8 | D8 boundary sweep | 180–280 |
| 9 | Delta specs (markdown) | 150–350 |

Total ≈ **1170–1970 authored lines** (goldens excluded). Budget risk: **High** — the AI-38-scoped
`size:exception` and auto-raise apply; `sdd-tasks` must re-forecast and slice work units so each
is independently revertible (WU8 is fully detachable; WU2's testdata migration is detachable from
WU1's gate only until the drift guard lands).

## Open Questions

- [ ] RED baseline may show in-flight scripted deltas racing past cancel on real transport; if
  so, D4's tail rule needs a recorded decision (escalate — do not silently widen).
- [ ] `usage` absent-vs-zero preservation through `chunk.go`'s decoder is assumed from AI-31;
  the RED baseline proves it before D6 lands.
- [ ] `conformancetest` package vs `import_boundary_test.go` allowlist — verified at apply; if
  denied, fallback is registration-by-duplication with a drift test pinning both bodies.
