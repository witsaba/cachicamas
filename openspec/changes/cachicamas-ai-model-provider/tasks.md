# Tasks: The model provider interface (AI-20)

> Predecessors: [`proposal.md`](proposal.md) · [`specs/ai-model-provider/spec.md`](specs/ai-model-provider/spec.md) (`R-AMP-001…021`) · [`design.md`](design.md) (reconciled 2026-08-01 — see design.md's updated reconciliation-gate note).
> **Apply gate**: this milestone is the wave's join point. `sdd-apply` for AI-20 MUST run strictly
> **after** AI-14, AI-15, AI-16, AI-17, AI-18 and AI-19 have all merged/landed in this worktree
> (`ai-wave-2`), not merely designed. Do not start Phase 1 implementation before that gate opens.
> Threat matrix: N/A per `design.md` — no rows applicable, none omitted.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~650–900 (interface+GoDoc ~90, `request.go` +test ~30, `provider_test.go` ~250, `agenttest/provider_test.go` ~150, AST guard test ~150, `doc.go` ~15) |
| 400-line budget risk | High |
| Chained PRs recommended | No — session-cached `exception-ok` already accepts the 5000-line budget |
| Suggested split | single PR |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units (informational — one PR)

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | AI-20.1 interface + 8-statement doc | PR 1 | `go test ./src/agenttest/... -run MethodSet` | N/A — pure declaration | revert `provider.go` |
| 2 | AI-20.2 pre-stream (`scriptProvider`) | PR 1 | `go test ./src/ai/... -run PreStream -race` | `scriptProvider` under `-race` | revert pre-stream cases + branch |
| 3 | AI-20.3 mid-stream (`scriptProvider` producer) | PR 1 | `go test ./src/ai/... -run MidStream -race` | `scriptProvider` goroutine under `-race` | revert producer + cases |
| 4 | AI-20.4 signature guard + bite proof | PR 1 | `go test ./src/agenttest/... -run SignatureGuard` | 2 scratch mutations, applied/reverted | revert guard file |
| 5 | AI-20.5 `TokenCounter` discovery | PR 1 | `go test ./src/agenttest/... -run TokenCounter` | stub with/without capability | revert `TokenCounter` + tests |

## Phase 0 — Reconciliation (complete)

- [x] 0.1 Re-read AI-14/AI-19 landed `design.md`; confirmed `Event`, `EventKindError`, `ErrorEvent`,
      `ErrorPayload`, `*Failure`, `PreStreamFailure`, `MidStreamFailure`, `FailureCategoryCancellation`,
      `ErrCancelled` match this design's assumptions; updated `design.md` in place with exact spellings.

## Phase 1 — AI-20.1 the interface (R-AMP-001…004)

- [x] 1.1 RED — `src/agenttest/provider_test.go`: assert `ModelProvider` has exactly one method
      `Stream(context.Context, Request) (<-chan Event, error)` (S-AMP-001/002/003); fails, type absent.
- [x] 1.2 GREEN — create `src/ai/provider.go`: `ModelProvider` interface, import `"context"` only;
      GoDoc carries all eight AI-02.1 §9 statements + AI-03.1 §9 enumerability (R-AMP-004).
- [x] 1.3 RED — `src/agenttest/provider_test.go`: external stub type implements `ModelProvider`,
      compiles, is exercised through it (S-AMP-007/008); assert no unexported method/embed (S-AMP-009).
- [x] 1.4 GREEN — stub compiles and passes from outside package `ai`; `go test ./src/agenttest/...`.
- [x] 1.5 REFACTOR — tidy GoDoc wording against the eight-statement checklist; no behavior change.

## Phase 2 — AI-20.2 pre-stream contract (R-AMP-005…008)

- [x] 2.1 RED — `src/ai/request_test.go`: `TestRequest_IsZero` (empty messages ⇒ true); fails, no method.
- [x] 2.2 GREEN — `src/ai/request.go`: add `func (r Request) IsZero() bool { return len(r.messages)==0 }`.
- [x] 2.3 RED — `src/ai/provider_test.go`: define unexported `scriptProvider{events []Event, terminal
      error, buffer int}`; cases for zero-request (S-AMP-017), invalid-request (S-AMP-013/014/015),
      validation-before-cancellation order (S-AMP-016/018), already-cancelled+valid → `PreStreamFailure`
      cancellation category (S-AMP-019/020/021), no observable effect pre-validation (S-AMP-022/023).
      **Process note**: fixture (scriptProvider) and its pre-stream branch were authored together in
      one file rather than as a separate observed-red step, because the struct and its Stream method
      are test-local fixture code with no prior partial state to fail against — see apply-progress.md
      Deviations. Also added an NFR-AMP-D nil-context case beyond the listed scenarios (cheap,
      directly required by the NFR, no dedicated task number).
- [x] 2.4 GREEN — implement `scriptProvider.Stream` pre-stream branch: `req.IsZero()` → `*Violation`;
      else `ctx.Err()` → `ai.PreStreamFailure({Category: FailureCategoryCancellation})`; else handover
      (Phase 2's handover is a naive unconditional send loop — Phase 3 hardens it under genuine RED).
- [x] 2.5 REFACTOR — reviewed; pre-stream checks are already one small, linear block — no extraction
      needed. `-race` run clean (`go test ./src/ai/... -run PreStream -race -v`).

## Phase 3 — AI-20.3 mid-stream contract (R-AMP-009…013) — COMPLETE

- [x] 3.1 RED — extended `scriptProvider` tests against the Phase 2 naive (unconditional-send, no
      terminal handling) producer: one closing site across completion/error/cancel (S-AMP-024/025/026),
      every send selects on cancellation incl. terminal (S-AMP-027/028), bounded close under `-race`
      no send-after-close (S-AMP-029/030/031), sanctioned bare-close-on-saturated-cancel vs
      never-cancelled-defect (S-AMP-032…035). Confirmed genuine RED (`go test ./src/ai/... -run
      MidStream -v -race`): 3 test functions failed for the right reason (naive impl delivered events
      after cancellation instead of dropping them); 2 passed pre-existing-true properties
      (never-cancelled backpressure, repeated bounded-close-under-race safety) — not everything needs
      to start red. **Found and fixed a self-inflicted race in the RED tests themselves**: a
      confirmation read placed immediately after `cancel()` could become a second ready `select` case
      in the producer's own goroutine at the exact moment it evaluates one, and Go resolves two
      simultaneously-ready cases pseudo-randomly — so even a *correct* implementation could
      occasionally "win" a send. Fixed with a `settleAfterCancel()` 50ms window (no receiver in flight
      during it) before every such confirmation read, making the intended branch the only ready one.
- [x] 3.2 GREEN — implemented `scriptProvider`'s producer goroutine: one goroutine, `defer close(out)`,
      every send (scripted events AND the optional terminal error event) inside
      `select{out<-ev; <-ctx.Done()}`, terminal built via `ai.ErrorEvent(*ai.Failure)` type-asserted
      from the `terminal error` field. `go test ./src/ai/... -run "PreStream|MidStream" -race -count=15`
      → all green, no flakiness across 15 repeated runs.
- [x] 3.3 REFACTOR — confirmed `scriptProvider` stays unexported and `ai_test`-local (S-AMP-036/037):
      one file, lower-case type, package `ai_test`, no exported alternative added.

## Phase 4 — AI-20.4 signature guard (R-AMP-014…016)

- [ ] 4.1 RED — `src/agenttest/provider_signature_guard_test.go`: resolve `provider.go` via
      `runtime.Caller(0)`; parse with `go/parser`; assert method set, param/result types, import
      allowlist `{"context"}`; run after Phase 1 lands so the target exists.
- [ ] 4.2 GREEN — guard passes (S-AMP-038/039); unresolvable/unparseable target → `t.Fatalf` naming
      the path, never skip (S-AMP-040/041); document sibling-layout dependency (S-AMP-042).
- [ ] 4.3 Bite mutation 1 (vendor stand-in): `req Request`→`req json.RawMessage` +
      `import "encoding/json"` on scratch copy; run
      `go test ./src/agenttest/ -run TestModelProviderInterface_SignatureGuard`; **paste red output
      here verbatim**; revert; confirm green.
- [ ] 4.4 Bite mutation 2 (changed carrier): `<-chan Event`→`<-chan string`; same guard test; **paste
      red output here verbatim**; revert; confirm green (S-AMP-043…045).
- [ ] 4.5 REFACTOR — tidy guard assertions/failure messages; no behavior change.

## Phase 5 — AI-20.5 optional capabilities (R-AMP-017…021)

- [ ] 5.1 RED — `src/agenttest/provider_test.go`: discovery via `provider.(ai.TokenCounter)`
      (S-AMP-049); non-advertising stub → clean absence, no error/zero/fallback (S-AMP-050/051); no
      catalog/config lookup exists (S-AMP-052); required-only stub fully conformant (S-AMP-055/056).
- [ ] 5.2 GREEN — `src/ai/provider.go`: add `TokenCounter` interface,
      `CountTokens(ctx context.Context, req Request) (TokenCount, error)`.
- [ ] 5.3 RED — method-set pin test: scratch-mutate `CountTokens` into `ModelProvider` itself; guard
      fails and names the added method (S-AMP-057/058); revert.
- [ ] 5.4 GREEN — confirm pin passes unmodified (R-AMP-021); record revert.
- [ ] 5.5 REFACTOR — confirm exactly one optional contract, no aggregate (S-AMP-046…048).

## Phase 6 — Wiring & docs

- [ ] 6.1 Modify `src/ai/doc.go`: one paragraph naming `ModelProvider`/`TokenCounter`.
- [ ] 6.2 Confirm `backend/agent/go.mod` zero requires; both AI-00 import guards pass (S-AMP-060).

## Phase 7 — Verification & closeout

- [ ] 7.1 Confirm apply gate: AI-14…AI-19 merged/landed in `ai-wave-2` before this phase starts.
- [ ] 7.2 Run `make test` (`go test -race -v ./...`) in `backend/agent/`; record green output.
- [ ] 7.3 Run `make lint`; record clean output.
- [ ] 7.4 Walk acceptance criteria 1–11 against landed code; record pass/fail per item.

> **Deviation note**: exceeds the 530-word tasks budget — house convention for this change
> (`spec.md`, `design.md`) already carries deviation notes at this density for the same reason:
> 5 leaves × RED/GREEN/REFACTOR, a guard with 2 recorded bite proofs, and an explicit apply-order
> gate across 6 sibling milestones cannot compress further without losing scenario traceability.
