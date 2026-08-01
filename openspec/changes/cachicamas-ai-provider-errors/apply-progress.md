# Apply Progress — AI-19 `cachicamas-ai-provider-errors`

> Change: `cachicamas-ai-provider-errors` · Milestone: AI-19 (Wave 2 keystone)
> Mode: Strict TDD · Runner: `make test` (`go test -race -v ./...`) from `backend/agent/`
> Worktree: `cachicamas-worktrees/ai-wave-2` · Package: `backend/agent/src/ai`
> Checkpoint policy: saved after each phase (connection-stability requirement for this session)

## Safety net (baseline, before any change)

`go test ./src/ai/... -v`: 284 passing subtests, 0 failing. `make test` (both `src/agenttest` and `src/ai`): green, cached.

## Status: Phases 1–2 of 7 complete (11/38 tasks). Phases 3–7 remaining.

## Phase 1: Foundation — category and failure skeleton — COMPLETE

- [x] 1.1 RED `provider_failure_internal_test.go`: `FailureCategory`/`DeliveryPath` zero-value groundwork.
- [x] 1.2 GREEN `provider_failure.go`: `FailureCategory` iota block (9 members + `failureCategoryLimit`), `DeliveryPath` iota block, `RetryDelay`/`Delay`, `FailureReport`, `Failure` struct (unexported fields), stub `Error()`/`Unwrap()`.
- [x] 1.3 REFACTOR: gofmt/go vet clean.

## Phase 2: AI-19.1 — Terminal error event (R-AIP-001..003) — COMPLETE

- [x] 2.1 RED `provider_failure_test.go` (`ai_test`): S-AIP-001..003.
- [x] 2.2 GREEN `event.go`: `EventKindError` constant (moved `eventKindEnd`), `eventRegistry` append line. GREEN `provider_failure.go`: `PreStreamFailure`/`MidStreamFailure`, `*Failure` implements `eventPayload{kind, validate}` directly (D7), `ErrorEvent`/`ErrorPayload`.
- [x] 2.3 RED S-AIP-004..006.
- [x] 2.4 GREEN: `ErrorEvent` nil rejection (`ErrEmpty` at `At("payload")`); kind derivation.
- [x] 2.5 RED S-AIP-007..009.
- [x] 2.6 GREEN: accessor disjointness confirmed structurally.
- [x] 2.7 REFACTOR: `event.go`'s "Registered kinds" doc list gets the `error` bullet (see deviation note below); `event_registry_test.go`'s shared cross-milestone exhaustiveness guard (`eventKindWitnesses` map + `productionEventKinds` list) extended for `EventKindError`, matching the established "same commit" convention every AI-15..AI-18 milestone already followed (confirmed necessary by a full `-race` run, not skipped). gofmt/vet/`-race` clean.

### Deviations recorded so far

1. **design.md step 6 says "doc.go kind list"; the actual landed recipe (event_descriptor.go, re-verified) and every prior milestone's actual implementation put the "documented list on `[EventKind]`" inside `event.go`'s own `EventKind` doc comment, not `doc.go`.** `doc.go` already anticipates AI-15…AI-20 without naming kinds and needs no edit. Followed the landed-code precedent (event.go) over design.md's literal file name; this is the S-AIP-055 "re-verify against landed code" case design.md itself flagged.
2. **`Failure.Category()` accessor has no explicit owning task in tasks.md** (a real gap — every other accessor is scheduled under a Phase 3/4/5/6 task, this one isn't). Added in Phase 2 instead of deferred: it is a pure, final (non-fake), one-line field accessor with no validation logic, and Phase 2's own R-AIP-003 terminal-exclusivity scenario (S-AIP-008) needs it to prove `Completion` carries no accessor of the same name.
3. `Failure.validate(at Path) *Violation` is a no-op stub through Phase 2 (mirrors `export_test.go`'s `WitnessPayload.validate` precedent: "a rule is added once a test needs one to fail"). The category-vocabulary rule lands in Phase 3, the status-class bound in Phase 4 — both were literally Phase-3/4-owned in tasks.md's own text ("Construction validates via FirstFailure: category Validate(...); StatusClass...") but Go's compile-order requires `FailureCategory.Validate()` (a Phase-3 deliverable per task 3.2) to exist before `Failure.validate` can call it. Resolved by keeping Phase 2's `validate()` a stub and wiring the real rule in during Phase 3/4, not by pulling `Validate()` earlier — preserves the tasks.md phase boundary for *where a method is first proven*, at the cost of one extra internal edit later.
4. `RawLabel`/`RequestID` are stored unsanitized by Phase 2's `newFailure` (the 64-byte drop-whole bound is Phase 3's `sanitizeOpaqueField`, task 3.4) — harmless in Phase 2 since neither accessor exists yet, so no observable behavior depends on it.
5. Pre-existing, unrelated `gofmt` drift found in `completion_test.go` (extra alignment spaces before a comment, line ~524) — confirmed via `git status`/`git diff` to predate this session (not introduced by me). Left untouched per the Safety Net rule (do not fix pre-existing issues opportunistically); noting for the record, not fixing in this PR.

## Phase 3: AI-19.2 — Category vocabulary (R-AIP-004..006) — NOT STARTED
## Phase 4: AI-19.3 — Retry hints and safe metadata (R-AIP-007..009) — NOT STARTED
## Phase 5: AI-19.4 — Partial-output discriminator (R-AIP-010..012) — NOT STARTED
## Phase 6: AI-19.5 — One vocabulary, two delivery paths (R-AIP-013..015) — NOT STARTED
## Phase 7: Cross-cutting NFRs and closeout — NOT STARTED

## Files touched so far

| File | Action | Phase |
|---|---|---|
| `backend/agent/src/ai/provider_failure.go` | Created | 1, 2 |
| `backend/agent/src/ai/provider_failure_internal_test.go` | Created | 1 |
| `backend/agent/src/ai/provider_failure_test.go` | Created | 2 |
| `backend/agent/src/ai/event.go` | Modified (append-only: `EventKindError` const, doc-list bullet, registry line) | 2 |
| `backend/agent/src/ai/event_registry_test.go` | Modified (shared cross-milestone guard: witness + production-vocabulary list) | 2 |

## Verification state at this checkpoint

`gofmt -l` clean on all touched files. `go vet ./src/ai/...` clean. `go build ./...` clean. `go test -race ./src/ai/...` green. `go test ./src/agenttest/...` green (no regression in the dependent package). No commits made yet — will commit at the end of Phase 2's logical unit (AI-19.1) before continuing to Phase 3.
