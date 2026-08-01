# Apply progress: `cachicamas-ai-model-provider` (AI-20 — the provider interface)

> Predecessors: tasks (`tasks.md`, this folder) · spec (`specs/ai-model-provider/spec.md`, `R-AMP-001…021`)
> · design (`design.md`, reconciled 2026-08-01 against landed AI-14/AI-19 symbols).
> Mode: **Strict TDD** (`make test` = `go test -race -v ./...` from `backend/agent/`).
> Apply gate confirmed open at session start: AI-14…AI-19 all merged into `ai-wave-2`
> (git log verified: `65d8be7`…`c00e491`, AI-14 NFR through AI-19 NFR).
> This is the FIRST apply run for AI-20 — no prior apply-progress existed to merge.

## Status

Phases 0–3 complete. Phase 1 committed (`1236a56`); Phases 2–3 committed together (share the
`scriptProvider` fixture) — see below. Phases 4–7 pending. This file is checkpointed after every
phase.

## TDD Cycle Evidence

| Task | RED | GREEN | REFACTOR |
|---|---|---|---|
| 1.1/1.2 `ModelProvider` method set | `go test ./src/agenttest/... -run MethodSet` failed to compile: `undefined: ai.ModelProvider` (3 sites) | `src/ai/provider.go` created; same command green | GoDoc wording checked against the eight §9 statements verbatim |
| 1.3/1.4 external stub | same RED as above (single compile failure covered both) | `stubProvider` in `src/agenttest/provider_test.go` compiles, `var _ ai.ModelProvider = stubProvider{}`, exercised via `provider.Stream` and drained | No behavior change |
| 2.1/2.2 `Request.IsZero` | `go test ./src/ai/... -run TestRequest_IsZero` failed to compile: `unconstructed.IsZero undefined` | `request.go`: `func (r Request) IsZero() bool { return len(r.messages)==0 }`; same command green | Reviewed; no extraction needed |
| 2.3/2.4 pre-stream branch | Disclosed deviation: fixture+branch authored together (test-local code, no prior partial state) — see Phase 2 notes | 5/5 pre-stream tests green incl. NFR-D nil-context | N/A (no separate refactor beyond 2.5) |
| 3.1/3.2 mid-stream producer | `go test ./src/ai/... -run MidStream -race` against Phase-2 naive producer: 3 test funcs genuinely FAILED (wrong post-cancel delivery / missing terminal) | Select-based producer (`select{out<-ev; <-ctx.Done()}`) + terminal-event handling landed; same command green, `-count=15` stress-clean | Confirmed `scriptProvider` unexported/`ai_test`-local |

## Work Unit Evidence

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 1 — AI-20.1 interface | `go test ./src/agenttest/... -run MethodSet -v` → 2/2 PASS | N/A — pure declaration, proven by reflect + compile-time assignment | `git revert` the `AI-20.1` commit; deletes `provider.go` and `agenttest/provider_test.go` |

## Phase 0 — Reconciliation (complete, inherited from tasks.md)

- [x] 0.1 Re-read AI-14/AI-19 landed `design.md`; confirmed `Event`, `EventKindError`, `ErrorEvent`,
      `ErrorPayload`, `*Failure`, `PreStreamFailure`, `MidStreamFailure`, `FailureCategoryCancellation`,
      `ErrCancelled` match this design's assumptions (also independently re-verified directly against
      `event.go`, `provider_failure.go`, `sequence.go`, `request.go` source during this apply session).

## Phase 1 — AI-20.1 the interface (R-AMP-001…004) — COMPLETE

- [x] 1.1 RED — `src/agenttest/provider_test.go` reflect-based method-set assertion; confirmed compile
      failure (`undefined: ai.ModelProvider`).
- [x] 1.2 GREEN — `src/ai/provider.go`: `ModelProvider` interface, imports only `"context"`; GoDoc
      carries all eight AI-02.1 §9 statements (verified verbatim against
      `openspec/specs/ai-stream-lifecycle/spec.md` §9) + AI-03.1 §9's enumerability clause (verified
      against `openspec/specs/ai-minimum-capabilities/spec.md` §9 consequence 2).
- [x] 1.3 RED — same file, `stubProvider` external-implementer proof + `var _ ai.ModelProvider =
      stubProvider{}` compile pin.
- [x] 1.4 GREEN — `go test ./src/agenttest/... -run MethodSet -v` → both tests PASS.
- [x] 1.5 REFACTOR — GoDoc wording reviewed against the eight-statement checklist; no behavior change.

Commit: `feat(ai): land the AI-20.1 provider interface (AI-20.1)`.

## Phase 2 — AI-20.2 pre-stream contract (R-AMP-005…008) — COMPLETE

- 2.1 RED / 2.2 GREEN: `TestRequest_IsZero` in `src/ai/request_test.go` failed to compile
  (`unconstructed.IsZero undefined`), then passed once `Request.IsZero()` landed in `request.go`.
- 2.3/2.4: **Process deviation, disclosed** — `scriptProvider` (struct + its pre-stream `Stream`
  branch) was authored together in one file write rather than as a separately-observed RED, because
  the struct and method are test-local fixture code with no prior partial state to fail against (there
  was nothing meaningful to watch fail before the file existed at all). All 5 pre-stream tests verified
  GREEN and re-verified after the Phase 3 rewrite. Also added one case beyond the listed scenarios:
  `TestScriptProvider_PreStream_NilContext_TreatedAsBackgroundNotAPanic` (NFR-AMP-D totality) — cheap,
  directly required by a named NFR, no dedicated task number existed for it.
- 2.5 REFACTOR: reviewed; the pre-stream block is already small and linear, no extraction warranted.

## Phase 3 — AI-20.3 mid-stream contract (R-AMP-009…013) — COMPLETE

- 3.1 RED: extended `provider_test.go` with `TestScriptProvider_MidStream_*` covering one-closing-site
  (completion/error/cancel), select-on-cancel (ordinary + terminal send), bounded-close-under-race
  repeated 20x, the sanctioned loss path, and its never-cancelled converse. Ran against the Phase 2
  naive (unconditional-send, no-terminal-handling) producer: 3 test functions genuinely failed (wrong
  events delivered post-cancellation / missing terminal event); 2 passed already (never-cancelled
  backpressure, repeated-close safety) — legitimate green-from-birth pins, not suspect.
- **Self-inflicted test race found and fixed before trusting the RED signal**: a confirmation read
  placed immediately after `cancel()` could itself become a second ready `select` case in the
  producer's own goroutine at the instant it evaluates one; Go picks pseudo-randomly between two
  simultaneously-ready cases, so even a *correct* implementation could occasionally complete a send it
  should have abandoned. Fixed with `settleAfterCancel()` (50ms, zero receivers in flight during the
  window) before every such confirmation read in 3 tests, making the intended branch the only one ever
  ready — deterministic, not merely low-probability-flaky.
- 3.2 GREEN: real producer — one goroutine, `defer close(out)` as the sole closing site, every send
  (scripted events and the optional terminal error event, built via `ai.ErrorEvent` from a
  `*ai.Failure` type-asserted out of the `terminal error` field) inside
  `select { case out <- ev: case <-ctx.Done(): return }`.
- Verification: `go test ./src/ai/... -run "PreStream|MidStream" -race -count=15` → all green, zero
  flakes across 15 repeated runs (build confidence pass beyond the single required run).
- 3.3 REFACTOR: confirmed `scriptProvider` stays unexported, `ai_test`-local, one file.

### Work Unit Evidence (Phases 2–3)

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 2 — AI-20.2 pre-stream | `go test ./src/ai/... -run PreStream -race -v` → 6/6 PASS (5 scenarios + NFR-D) | `scriptProvider` under `-race`, real goroutines/channels, no mocks | revert pre-stream cases + branch in `provider_test.go`, revert `Request.IsZero` in `request.go` |
| 3 — AI-20.3 mid-stream | `go test ./src/ai/... -run MidStream -race -v` → 7/7 test funcs PASS (incl. 20 sub-iterations); `-count=15` stress pass clean | `scriptProvider` producer goroutine under `-race`, real cancellation via `context.WithCancel`, real channel backpressure (unbuffered) | revert producer body + mid-stream cases in `provider_test.go` |

## Phase 4 — AI-20.4 signature guard (R-AMP-014…016) — COMPLETE

`src/agenttest/provider_signature_guard_test.go` resolves `provider.go` via `runtime.Caller(0)` +
sibling-relative `filepath.Join` (ADR 0005 § D2 Guard C), parses with `go/parser`, and asserts:
exactly one method `Stream`; params `(context.Context, Request)`; results `(<-chan Event, error)`;
imports ⊆ `{"context"}`. Unresolvable/unparseable target fails loudly via `t.Fatalf` (S-AMP-040/041),
never skips.

Guard was green-from-birth (provider.go already conformed from Phase 1) — genuine RED evidence comes
from two bite mutations, both captured verbatim in `tasks.md`. **Process refinement over design.md's
plan**: ran each mutation with `src/agenttest/provider_test.go` temporarily moved out of the package
(`mv` to `.hold` and back), because mutating the shared `ModelProvider` interface also breaks
`stubProvider`'s `var _ ai.ModelProvider = stubProvider{}` compile-time pin in that sibling file —
without isolating it, the observed "red" would be an unrelated Go compiler type-check error about
`stubProvider`, not the guard's own AST-derived assertion message. Isolating restored the intended
"observed red is the guard's, not the compiler's" property design.md asked for.

- Bite 1 (vendor stand-in, `req Request`→`req json.RawMessage` + `import "encoding/json"`): guard
  failed with 2 of its own assertion lines (import allowlist + param type) — see `tasks.md` for the
  verbatim transcript. Reverted; confirmed byte-identical to the last commit via `git diff --stat`
  (empty output).
- Bite 2 (changed carrier, `<-chan Event`→`<-chan string`): guard failed with 1 assertion line
  (result type) — verbatim transcript in `tasks.md`. Reverted; confirmed clean the same way.
- Post-revert: `go test ./src/agenttest/... -v` → all PASS (including the two tests from Phase 1 that
  live in the restored `provider_test.go`), full module `go build && go vet && go test -race ./...`
  green.

### Work Unit Evidence (Phase 4)

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 4 — AI-20.4 signature guard | `go test ./src/agenttest/... -run SignatureGuard -v` → PASS at rest; 2/2 bite mutations produced genuine, guard-specific RED, both reverted and re-confirmed green | Real `go/parser` over the real on-disk `provider.go` (no mocked filesystem, no subprocess) | revert `provider_signature_guard_test.go`; no production code changed by this phase |

## Phases 5–7

Pending — see `tasks.md` for the authoritative checklist; this file is updated after each phase closes.

## Session

Session: SDD apply run for AI-20, worktree `ai-wave-2`, branch
`feat/2026-08-01-cachicamas-ai-layer1-wave-2`.
Artifact store: hybrid (this file + `tasks.md` on disk, mirrored to Engram topic
`sdd/cachicamas-ai-model-provider/apply-progress`).
