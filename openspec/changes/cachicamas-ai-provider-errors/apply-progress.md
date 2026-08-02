# Apply Progress — AI-19 `cachicamas-ai-provider-errors`

> Change: `cachicamas-ai-provider-errors` · Milestone: AI-19 (Wave 2 keystone)
> Mode: Strict TDD · Runner: `make test` (`go test -race -v ./...`) from `backend/agent/`
> Worktree: `cachicamas-worktrees/ai-wave-2` · Package: `backend/agent/src/ai`

## STATUS: COMPLETE — 38/38 tasks, all 7 phases. `make test` and `make lint` both green/clean.

## Safety net (baseline, before any change)

`go test ./src/ai/... -v`: 284 passing (top-level-anchored count), 0 failing. `make test` (both `src/agenttest` and `src/ai`): green, cached.

## Final state

- `go test ./src/ai/... -v -count=1` (fresh, not cached): **1210 passing subtests (including all nested `t.Run` cases), 0 failing.**
- `make test` (`go test -race -v ./...` from `backend/agent/`): both `src/agenttest` and `src/ai` — **ok**.
- `make lint` (`go vet ./...` + `golangci-lint run --config=.golangci.yml ./...`): **0 issues.**
- `gofmt -l` on every file this milestone touched: clean.
- `go build ./...`: clean.

## Commits (6, one per logical unit)

1. `6a01487` feat(ai): land the AI-19.1 terminal error event (AI-19.1) — Phases 1+2.
2. `9584135` feat(ai): land the AI-19.2 provider failure category vocabulary (AI-19.2) — Phase 3.
3. `d3b1d05` feat(ai): land the AI-19.3 retry hints and safe metadata (AI-19.3) — Phase 4.
4. `a2aaa66` feat(ai): land the AI-19.4 partial-output discriminator (AI-19.4) — Phase 5.
5. `2e60dac` feat(ai): land the AI-19.5 one-vocabulary-two-paths closeout (AI-19.5) — Phase 6.
6. (pending) feat(ai): close out AI-19 NFRs — totality and a clean make lint (AI-19 NFR) — Phase 7, this commit.

## Files touched (final)

| File | Action |
|---|---|
| `backend/agent/src/ai/provider_failure.go` | Created (~330 lines: 5 types, 9 sentinels, 2 constructors, 11 accessors, `ErrorEvent`/`ErrorPayload`) |
| `backend/agent/src/ai/provider_failure_test.go` | Created (`ai_test`, ~30 top-level test functions, external/black-box surface) |
| `backend/agent/src/ai/provider_failure_internal_test.go` | Created (`ai`, internal exhaustiveness pins) |
| `backend/agent/src/ai/event.go` | Modified, append-only (`EventKindError` const, doc-list bullet, registry line) |
| `backend/agent/src/ai/event_registry_test.go` | Modified (shared cross-milestone exhaustiveness guard extended) |
| `openspec/changes/cachicamas-ai-provider-errors/tasks.md` | `[x]` marks + per-task evidence notes |
| `openspec/changes/cachicamas-ai-provider-errors/apply-progress.md` | This file |

## Per-phase summary (all 7 complete)

**Phase 1 — Foundation.** `FailureCategory`/`DeliveryPath` iota blocks, `RetryDelay`/`Delay`, `FailureReport`, `Failure` struct skeleton, nil-safe stub `Error()`/`Unwrap()`.

**Phase 2 — AI-19.1 terminal error event (R-AIP-001..003).** `EventKindError` wired into AI-14's registry (`Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: true`); `PreStreamFailure`/`MidStreamFailure` constructors; `*Failure` implements the sealed `eventPayload` interface directly (no wrapper); `ErrorEvent`/`ErrorPayload`; terminal exclusivity against `Completion` proven by exhaustive method-set comparison.

**Phase 3 — AI-19.2 category vocabulary (R-AIP-004..006).** `String()`/`Validate()`/`FailureCategories()` on the 9-member vocabulary; category rule wired into construction; `RawLabel()` + `sanitizeOpaqueField` (64-byte drop-whole bound); AST-proof no cross-vendor normalizer exists.

**Phase 4 — AI-19.3 retry hints + safe metadata (R-AIP-007..009).** `Retryable()`/`RetryAfter()` (presence-typed, `usage.go`'s `TokenCount` idiom); `Error()` redaction (fixed prefix + category text only, planted-sentinel proof); `StatusClass()`/`RequestID()` dedicated accessors; status-class 0..5 bound.

**Phase 5 — AI-19.4 partial-output discriminator (R-AIP-010..012).** `PartialOutput()`/`Delivery()`, the two perpendicular axes; fourth cell (pre-stream + output) proven unconstructible by a compile-time signature pin; naive-retry-safety predicate proven independent of category and retryability.

**Phase 6 — AI-19.5 one vocabulary, two paths (R-AIP-013..015).** Per-category sentinels + `Is()`, no umbrella sentinel (N×N cross-check); `errors.As` reaches the full failure through a wrap; cause's own sentinel chain survives `Unwrap()` alongside the category sentinel; both-paths accessor parity swept across all 9 categories.

**Phase 7 — NFR closeout.** Totality sweep (11 extreme-input cases, 0 panics); dependency purity confirmed (zero `go.mod` requires, both AI-00 import guards pass); NFR-AIP-C closing table (all 4 rejection paths route through `*ai.Violation`); NFR-AIP-E re-confirmed against landed AI-14/15 source; **found and fixed 3 real `make lint` issues** (see below); final `make test` + `make lint` both clean.

## `make lint` issues found and fixed (Phase 7)

`make lint` was not run until Phase 7 (only `go vet`/`gofmt` per-phase) — this surfaced 3 real issues the per-phase checks didn't catch:

1. `package-comments` (revive): `provider_failure.go`'s file-top doc comment had no blank line before `package ai`, making `golangci-lint` read it as an attempted (malformed) package comment. Every other file in this package has that blank line (`doc.go` alone owns the real package comment). Fixed: added the blank line.
2. `error-return` (revive): `failureCategorySentinel` returned `(error, bool)` — error not last. Fixed: swapped to `(bool, error)`, matching `eventRegistryEntry`'s own `(value, bool)` idiom in this codebase (this is a data lookup, not an operation that itself fails).
3. `QF1011` (staticcheck): a deliberate compile-time signature-pin (`var _ func(ai.FailureReport) (*ai.Failure, error) = ai.PreStreamFailure`, proving S-AIP-032's "no output-flag parameter" claim) was flagged as a redundant type annotation. It is not redundant — inferring the type would make the check vacuous. Fixed with `//nolint:staticcheck` + a reason comment, matching this codebase's existing suppression convention (`tool_call_test.go`'s `//nolint:gosec // a fixture, not a credential`).

All three are now fixed; `make lint` = "0 issues".

## Deviations (7 total — all disclosed at the time they occurred, none hidden)

1. **design.md step 6 literally says "doc.go kind list"; the actual landed six-step recipe (`event_descriptor.go`) and every prior milestone's real implementation put the "documented list on `[EventKind]`" inside `event.go`'s own `EventKind` doc comment, not `doc.go`.** `doc.go` already anticipates AI-15…AI-20 without naming kinds and needs zero edits. Followed landed-code precedent over design.md's literal file name — the S-AIP-055 "re-verify against landed code" case design.md itself flagged.
2. **`Failure.Category()` accessor has no explicit owning task anywhere in tasks.md** (a real gap — every other accessor is scheduled under a Phase 3/4/5/6 task, this one isn't). Added in Phase 2: a pure, final (non-fake), one-line field accessor with no validation logic; Phase 2's own R-AIP-003 scenario (S-AIP-008) needed it to prove `Completion` carries no accessor of the same name.
3. `Failure.validate(at Path) *Violation` was a no-op stub through Phase 2 (`export_test.go`'s `WitnessPayload.validate` precedent), then gained the category rule in Phase 3 and the status-class bound in Phase 4. Go's compile order requires the callee (`FailureCategory.Validate()`, a Phase-3 deliverable per task 3.2) to exist before the caller compiles; tasks.md's own prose describes the constructors' final validation behavior without sequencing which phase first proves which sub-rule. Resolved by staging, not by moving `Validate()` earlier.
4. `RawLabel`/`RequestID` were stored unsanitized by Phase 2's `newFailure` (harmless — no accessor existed yet); Phase 3 added `sanitizeOpaqueField` and wired both fields through it in the same edit, though `RequestID()` itself doesn't land until Phase 4.
5. Pre-existing, unrelated `gofmt` drift in `completion_test.go` (extra alignment spaces before a comment, ~line 524) — confirmed via `git status`/`git diff` to predate this session. Left untouched per the Safety Net rule; noted, not fixed.
6. **Genuine strict-TDD ordering slip**: the status-class 0..5 bound was written directly into `Failure.validate()` before its own dedicated test existed (every other rule in this milestone was correctly test-driven first). Disclosed rather than hidden. Remediated same-session: added `TestNewFailure_OutOfRangeStatusClass_FailsWithErrOutOfRangeAtStatusClass` immediately after, then **actually verified** it was load-bearing — not just asserted — by temporarily deleting the bound check, re-running to confirm the test genuinely FAILED, then restoring the check and confirming PASS again.
7. `Is(target error) bool` (task 6.2) and the per-category sentinels (task 6.4) were implemented together in one RED/GREEN cycle rather than tasks.md's literal 6.2-then-6.4 split: `Is()` has nothing real to compare against until the sentinels exist, so a separate empty-stub cycle for 6.2 alone would have been low-value theater. Same class of issue as deviation 3, same resolution style (disclosed, not hidden).

## TDD Cycle Evidence

Strict TDD Mode confirmed active throughout. RED/GREEN pairs were executed and verified per phase (sometimes per sub-step within a phase, batched where tightly coupled — see deviations 3/7 above for the specific, disclosed cases). Every RED was a genuine compile failure or a genuine assertion failure (verified by running `go test`/`go vet`, never assumed); every GREEN was a genuine passing `go test` run.

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `provider_failure_internal_test.go` | Unit | N/A (new) | ✅ Written, confirmed compile-fail | ✅ Passed | ➖ 2 types, exhaustive over all named members | ✅ gofmt/vet clean |
| 1.2 | `provider_failure.go` | Unit | — | (see 1.1) | ✅ Confirmed by `go test -run` | ➖ Structural | ✅ |
| 1.3 | — | — | — | — | — | — | ✅ gofmt/vet clean |
| 2.1/2.3/2.5 | `provider_failure_test.go` | Unit | N/A (new) | ✅ Written, confirmed `vet` compile-fail (`undefined: ai.PreStreamFailure`) | ✅ Passed (9 subtests) | ✅ S-AIP-001..009, 9 scenarios | ✅ |
| 2.2/2.4/2.6 | `event.go`, `provider_failure.go` | Unit | ✅ 284/284 pre-existing tests (baseline) | (see above) | ✅ | ✅ | ✅ |
| 2.7 | `event_registry_test.go` | Unit | ✅ | Guard caught missing witness entry (real failure, not simulated) | ✅ Fixed, `-race` full run green | ➖ | ✅ gofmt/vet/-race clean |
| 3.1 | `provider_failure_test.go` + internal | Unit | ✅ | ✅ Written, confirmed compile-fail (`c.Validate undefined`) | ✅ Passed | ✅ 9 categories + 256-value sweep | ✅ |
| 3.2 | `provider_failure.go` | Unit | — | (see 3.1) + added test for construction-time rejection before writing that wiring | ✅ | ✅ | ✅ |
| 3.3 | `provider_failure_test.go` | Unit | ✅ | ✅ Written, confirmed compile-fail (`f.RawLabel undefined`) | ✅ Passed (6 subtests) | ✅ over-long/exact-bound/control-char/empty/wrap-survival | ✅ |
| 3.4 | `provider_failure.go` | Unit | — | (see 3.3) | ✅ | ✅ | ✅ |
| 3.5 | `provider_failure_internal_test.go` | Unit | ✅ | — (pin, not scenario-driven) | ✅ Passed | ➖ | ✅ (sentinel half deferred to 6.4, disclosed) |
| 4.1 | `provider_failure_test.go` | Unit | ✅ | ✅ Written, confirmed compile-fail (`f.Retryable undefined`) | ✅ Passed | ✅ all 9 categories × 2 retryable states | ✅ |
| 4.2 | `provider_failure.go` | Unit | — | (see 4.1) | ✅ | ✅ | ✅ |
| 4.3 | `provider_failure_test.go` | Unit | ✅ | ✅ Written, confirmed compile-fail (`f.StatusClass undefined`) | ✅ Passed (planted-sentinel proof) | ✅ 4 subtests + companion absent-case | ✅ |
| 4.4 | `provider_failure.go` | Unit | — | ⚠️ **Ordering slip on the status-class bound alone** (deviation 6) — disclosed, remediated with an actual remove/re-run/restore verification | ✅ | ✅ | ✅ |
| 4.5 | — | — | — | — | — | — | ✅ doc comment + structural confirmation |
| 5.1/5.3 | `provider_failure_test.go` | Unit | ✅ | ✅ Written, confirmed compile-fail (`f.PartialOutput undefined`) | ✅ Passed | ✅ 3 shapes × comparison × unsafety-across-9-categories | ✅ |
| 5.2/5.4 | `provider_failure.go` | Unit | — | (see 5.1) | ✅ | ✅ | ✅ |
| 5.5 | `provider_failure_test.go` | Unit | ✅ | — (refactor only) | ✅ Still passing after table consolidation | ➖ | ✅ table-driven |
| 6.1 | `provider_failure_test.go` | Unit | ✅ | ✅ Written; passed immediately (genuinely structural, no new code — task 6.2's own description) | ✅ | ✅ 4 subtests | ✅ |
| 6.2/6.3/6.4 | `provider_failure.go`, `provider_failure_test.go` | Unit | ✅ | ✅ Written, confirmed compile-fail (`undefined: ai.ErrAuthentication`) | ✅ Passed | ✅ 9×9 cross-check + wrap/cause-chain proof | ✅ |
| 6.5/6.6 | `provider_failure_test.go` | Unit | ✅ | ✅ Written (parity table) | ✅ Passed immediately | ✅ swept all 9 categories | ✅ |
| 7.1/7.2 | `provider_failure_test.go` | Unit | ✅ | ✅ Written (11-case totality table) | ✅ Passed, 0 panics | ✅ 11 cases | ✅ |
| 7.3 | — (verification) | — | — | — | ✅ Both AI-00 guards pass | — | — |
| 7.4 | `provider_failure_test.go` | Unit | ✅ | ✅ Written (4-row closing table) | ✅ Passed | ✅ 4 rejection paths | ✅ |
| 7.5 | — (verification) | — | — | — | ✅ Full suite green with `EventKindError` integrated | — | — |
| 7.6 | This file | — | — | — | — | — | — |
| 7.7 | — | — | — | — | ✅ `make test` + `make lint` both clean (3 real lint issues found and fixed) | — | ✅ |

### Test Summary

- **Total tests written this milestone**: ~34 top-level test functions (2 internal, ~32 external), well over 100 subtests/table rows covering all 55 scenarios (S-AIP-001..055) plus 5 NFRs.
- **Total tests passing**: 1210/1210 subtests in the full `ai` package (fresh run, `-count=1`), 0 failing.
- **Layers used**: Unit only (no integration/E2E boundary at Layer 1 — matches design.md's own "Testing" section: no routing/shell/subprocess/VCS boundary, threat matrix N/A).
- **Approval tests**: None — no refactoring-of-existing-behavior tasks in this milestone (only additive).
- **Pure functions created**: `sanitizeOpaqueField`, `failureCategorySentinel`, all `FailureCategory`/`DeliveryPath` methods, `newFailure` — all deterministic, no side effects.

## Work Unit Evidence (all-modes hard gate)

| Evidence | Value |
|---|---|
| Focused test command and result | `go test -run TestFailure -race ./backend/agent/src/ai/...` → PASS (all `TestFailure*`/`TestNewFailure*`/`TestErrorEvent*`/`TestFailureCategor*`/`TestProviderFailure*` cases) |
| Runtime harness | N/A — pure library, no provider/runtime integration at Layer 1 (design.md's own Testing Strategy note; AI-20 is the provider interface, out of scope here) |
| Rollback boundary | `git revert` the 6 commits (or drop the branch): additive only — `provider_failure.go` + its two test files are new; `event.go`/`event_registry_test.go` changes are append-only lines. Nothing else in the module imports this milestone's exports yet (AI-20 has not landed), matching the proposal's own rollback plan. |
