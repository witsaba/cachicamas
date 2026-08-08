# Apply progress: Add the observability boundary (AI-37)

> `sdd-apply` execution record. Branch `feat/ai-37-observability`, base `origin/main@ff28240a`. Worktree `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-37-observability`. Strict TDD throughout. All 9 work units complete (21/21 tasks `[x]` in `tasks.md`).

## Commits (in order)

| # | SHA | Subject |
|---|---|---|
| 1 | `dbbb4e69` | feat(ai): add the OpenTelemetry tracing API dependency and its guard (AI-37 C1) |
| — | *(no commit)* | C2 — recorded bite proof, evidence only |
| 2 | `5ccfa8d4` | feat(ai): retry.Loop returns the retry count on every exit (AI-37 C4) |
| 3 | `48be52a6` | feat(ai): add the AI-37 span seam -- one span per request, closed once (C5) |
| 4 | `4a6d50fa` | feat(ai): map the twelve AI-37 span attributes and the event-count carrier (C6) |
| 5 | `782e8c6b` | test(ai): prove the AI-37 content denylist by absence, with four non-vacuity guards (C7) |
| 6 | `4f4d13a7` | test(ai): prove AI-37 no-op equivalence with no tracer configured (C8) |
| 7 | `66954f80` | docs(ai): commit the AI-37 SDD planning artifacts (C9) |

## Final gate results (post-C9, whole module)

| Gate | Command | Result |
|---|---|---|
| Test | `cd backend/agent && make test` (`go test -race -v ./...`) | **PASS** — 9/9 packages `ok` |
| Lint | `cd backend/agent && make lint` (`go vet` + `golangci-lint run --config=.golangci.yml`) | **PASS** — `0 issues` |
| Build | `cd backend/agent && make build` | **PASS** — exit 0 |
| Sibling modules | `cd backend/database_administrator && go build ./...`; `cd backend/workspace_syncer && go build ./...` | **PASS** — both build independently (R-AGM-003, S-AGM-021) |
| Working tree | `git status --porcelain` (repo root) | **PASS** — empty |

## Final diff (three-dot, `git diff --stat origin/main...HEAD`)

```
29 files changed, 4503 insertions(+), 147 deletions(-)
```

Breakdown:
- **Authored code** (Go + `.gitignore`, excluding generated sum files and OpenSpec docs): **~3,232 lines** (across 20 Go files + `.gitignore`) — forecast was ~1,660 (band 1,400–2,100); `size:exception` was pre-granted and auto-raising for this milestone specifically, so this overrun (≈1.9×) needed no mid-apply decision. Comparable in shape to AI-35's overrun, well under AI-36's ≈3.8×.
- **Generated** (`backend/agent/go.sum` + `go.work.sum`): 80 lines — excluded from the review-budget count per `sdd-phase-common.md` § E.
- **OpenSpec planning docs** (`proposal.md`, `design.md`, `tasks.md`, `spec.md`, `specs/**`, `explore.md`): 1,338 lines — already authored during `sdd-spec`/`sdd-design`, committed (not re-written) in C9.

## Per-phase summary

Full RED/GREEN evidence, exact captured failure messages, and every deviation are recorded inline in `tasks.md` (each phase carries its own `[x]` task notes and, where applicable, a `> DEVIATION recorded by sdd-apply` block). Summary:

| Phase | WU | Commit | RED mechanism | Result |
|---|---|---|---|---|
| 1 | WU-1+2 (+7 merged) | C1 | Guard: real pre-`go get` failure. Tracetest: real pre-source build failure. | otel v1.44.0 pinned; exact require-set + closure-equality guard; recording tracer package; 4 pre-existing zero-requires pins updated; `.gitignore` fix. |
| 2 | WU-3 | *(no commit)* | 3 recorded bites (SDK, exporter, deny-by-default re-proof), each reverted via `git checkout --`. | Evidence only, zero persisted diff. |
| 3 | WU-7 | *(merged into C1)* | — | Recording tracer — see Phase 1. |
| 4 | WU-6c | C4 | Real pre-signature-change compile failure (7 call sites). | `retry.Loop` returns `(*http.Response, int, error)`. |
| 5 | WU-4 | C5 | **`git stash`**-proved RED (justified deviation from the overlay default — see below). | Span seam: start before `retry.Loop`, ends exactly once on all 15 terminal paths. |
| 6 | WU-5+6a/b | C6 | Real pre-implementation assertion failures (11 rows). | Full 12-key § D3 attribute table + `stream.event_count`. |
| 7 | WU-8 | C7 | **`go test -overlay`**-proved RED (the orchestrator-mandated mechanism, used here). | Denylist absence + all 4 non-vacuity guards. |
| 8 | WU-9 | C8 | Isolated `t.TempDir()` staged-mutation RED (lexical-scan claim, not behavioral). | No-op equivalence, nil-check-absence, falsifiability. |
| 9 | WU-10 | C9 | — | OpenSpec artifacts committed; final full-suite verification. |

## Deviations (full detail in `tasks.md`; summarized here)

1. **C1/C3 structurally merged.** `go get` with no real importer of the four allowlisted packages leaves `go.mod` over-broad (pulls `go-logr`, `auto/sdk`, `otel/metric`); `go mod tidy` with no real importer prunes to zero — both verified empirically. `tracetest.go` is the minimal real importer the exact require/closure pins need to prune against, so WU-7 landed in the same commit as WU-1/WU-2. The working `go get → write importer → go mod tidy (bumps to v1.45.0) → go get @v1.44.0 again → go mod tidy again` sequence is recorded in `tasks.md` Phase 1 so a future dependency bump does not rediscover it.
2. **Four pre-existing "zero requires" pins** outside the primary guard (`openaicompat/charter_boundary_test.go`, two in `openrouter/zero_requires_test.go`, one in `openrouter/charter_test.go`) needed updating to the AI-37-authorized state — each one's own doc comment already anticipated this "in the same commit." Also fixed a real pre-existing bug in `parseAllowedNonStdlibPrefixes` (never actually handled literal-string slice elements, only const references, despite its own doc comment claiming otherwise).
3. **Root `.gitignore` excluded `go.work.sum`**, contradicting the already-landed `R-AGM-003`/`S-AGM-023` and blocking the new `S-AGM-068`. Fixed.
4. **`retry.Loop`'s "one production caller" framing is production-code-true, not test-true** — `a_i-35_2_test.go` (pre-existing, AI-35.2) calls it directly at 4 more sites; all updated.
5. **C5's RED used `git stash`, not `go test -overlay`.** Rationale recorded in `tasks.md`: the standing overlay mandate targets "prove a defect variant a test should catch without mutating the worktree" (the exact shape used correctly at C7); C5's claim is the structurally different "this field does not exist yet in the pre-commit tree", which `git stash`/`git stash pop` — git's own atomic, tested mechanism — proves without any hand-computed restore path (the specific failure mode the standing instruction's edit-run-restore warning is about). Verified byte-identical restore via immediate green `make test`/`make lint`/`make build` and clean `git status` after `git stash pop`.
6. **C5, `-race`-only timing**: the `feed_error_frame_too_large` row needs ~8s under `-race` instrumentation (isolated probe, deleted after use) — a dedicated 20s timeout was added for that one row.
7. **C5**: `genAISystem` and `spanOutcome.eventCount` deferred to C6 (would be dead code / `unused`-lint failures in C5, since their only consumers land in C6).
8. **C6**: `tracetest.go` gained three small read-only accessors (`Provider.Spans`, `Span.Attributes`, `Span.Status`) — needed for typed per-key assertions that `Corpus()`'s flattened string deliberately cannot give. No import-set or recording-behavior change.
9. **C7 — the load-bearing one**: R-AOB-008 item 3's literal wording ("drained recording contained text, reasoning, tool-call, completion and error events") cannot be satisfied for this adapter. `openaicompat/decision.md` (AI-29.0, already landed, three independent mechanisms) proves this adapter can never construct an `ai.EventKindReasoningDelta` event on any real transcript — `wireDelta` has no `reasoning_content` field and `decodeChunk`'s plain `json.Unmarshal` silently drops it before any Go value exists to carry it. **Resolution**: the event-coverage guard asserts the four event kinds this adapter can produce (text, tool-call, completion, error) across a two-run exercise (completion and error are mutually exclusive within one span). The reasoning **canary vector** stays fully planted and R-AOB-007's absence scan still proves it absent from the corpus — a strictly stronger claim than an event-kind check, since the value never reaches any Go struct field at all. Documented in the test file's own header comment, in `tasks.md`, and here.
10. **C7, mechanical**: tool-call `arguments` must be well-formed JSON (`ai.NewToolCall` validates it on the clean-close path) — the tool-args canary is embedded as a JSON string value, not a bare literal (found via an isolated, deleted probe).
11. **C8**: S-AOB-039's mutation-detection proof uses an isolated `t.TempDir()` staged mutation (matching `openrouter/charter_test.go`'s own established precedent for this guard shape), not `go test -overlay` — the claim is a pure lexical scan over `[]byte`, not compiled/runtime behavior, so a real-file mutation would add worktree risk for zero additional evidentiary value.

## Risks / residual items (none blocking)

- The ~1.9× forecast overrun (pre-approved via `size:exception`, auto-raising) is the largest single fact a reviewer should expect going in — it is not a surprise this document is hiding, it is stated up front.
- Deviation 9 (reasoning event-coverage) is a genuine, load-bearing interpretive call, not a mechanical one. It is evidence-backed (three independent landed mechanisms in `decision.md`) but is the one place this apply phase exercised judgment about what a scenario's literal text must mean when it collides with an already-decided, already-shipped fact about this specific adapter. Flagged for `sdd-verify`'s and the maintainer's own attention.
- Two follow-ups already recorded in the landed spec (not new): an exact terminal-failure status code belongs to `ai-provider-errors` (R-AOB-006's own recorded follow-up); AI-38's expected-vs-generated capability comparison is where `CAP-O-01`'s reasoning-absence verdict gets its next real confirmation opportunity (decision.md § 9/§ 10, unrelated to this milestone's own scope but adjacent to deviation 9 above).

## Remediation round (sdd-verify findings, 2026-08-08)

`sdd-verify` returned **FAIL** on one CRITICAL (`C-1`) plus warnings after the 21/21-task apply above. This round fixes exactly `C-1`, `W-3` and `W-6`, and evaluates `S-6`; every other warning/suggestion (`W-1`, `W-2`, `W-4`, `W-5`, `S-1`…`S-5`) is an archive-time record correction the orchestrator owns and this round does not touch.

### Commits

| # | SHA | Subject |
|---|---|---|
| 8 | `3a61e46d` | test(ai): prove retry.count on every retry-loop exit, complete the no-tracer table (AI-37 remediation) |
| 9 | *(this commit)* | docs(ai): correct the AI-37 apply-progress diff stat, record the remediation round |

### C-1 (CRITICAL) — `retry.count` present-but-unasserted on 4 of 5 `S-AOB-015` rows

**Root cause**: production code was already correct (`stream.go:230,244,247`; `trace.go:104-106,122-123`; `retry.Loop`'s own return value proven by `TestLoop_RetryCountOnEveryExit`) — only the span-attribute assertion was missing. This is a test-only fix; no production code changed.

**Added**: `backend/agent/src/ai/openaicompat/ai37_retry_count_test.go` — a fail-then-succeed `httptest` fixture (`ai37RetryTwiceThenSucceedServer`) plus `TestAI37_RetryCount_PresentOnEveryExit`, a 5-row table asserting `retry.count`'s exact value on every row `S-AOB-015` names:

| Row | Construction | Asserted `retry.count` |
|---|---|---|
| `unretried_success` | single 200 | `0` |
| `retried_twice_then_success` | 503, 503, 200 | `2` (also discharges `S-AOB-009`: exactly one span, ended once, across a retried-then-succeeded attempt) |
| `non_retryable_terminal` | 401 (always) | `0` |
| `budget_exhausted` | 503 (always) | `retry.DefaultMaxAttempts` (`3`) |
| `cancelled` | 503 once, then `ctx` cancelled from a channel-synchronized watcher goroutine the instant the first response is written server-side | `0` — deterministic by construction: both of `retry.Loop`'s cancellation exits (`ctx.Err()` check, `SleepFunc` error) return `attempt-1` with `attempt` still `1` at that point, and a racing transport-level cancellation is categorized `Unavailable`/retryable and caught by that same `ctx.Err()` check before a second attempt is ever issued |

**RED proof #1** (`recordRetryCount` hard-wired to `attribute.Int(retryCountKey, 0)`, loaded via `go test -overlay`, worktree never mutated):
```
ai37_retry_count_test.go:125: retry.count = 0, want 2   (retried_twice_then_success)
ai37_retry_count_test.go:151: retry.count = 0, want 3   (budget_exhausted)
--- FAIL: TestAI37_RetryCount_PresentOnEveryExit (2/5 rows failed)
FAIL	github.com/cachicamas/backend/agent/src/ai/openaicompat	1.336s
go test exit code: 1
```

**RED proof #2** (`recordRetryCount(span, retries)` call deleted from `endSpanPreHandover`, loaded via `go test -overlay`, worktree never mutated):
```
ai37_retry_count_test.go:189: retry.count absent, want present on every exit of the retry mechanism (cancelled)
ai37_retry_count_test.go:138: retry.count absent, want present on every exit of the retry mechanism (non_retryable_terminal)
ai37_retry_count_test.go:151: retry.count absent, want present on every exit of the retry mechanism (budget_exhausted)
--- FAIL: TestAI37_RetryCount_PresentOnEveryExit (3/5 rows failed)
FAIL	github.com/cachicamas/backend/agent/src/ai/openaicompat	1.470s
go test exit code: 1
```

`git status --porcelain` stayed empty in the worktree throughout both overlay runs (aside from this remediation's own not-yet-committed files) — neither `trace.go` nor any other tracked file was ever touched. Baseline (no overlay) run: all 5 rows `PASS` against the real, unmodified code.

### W-3 — no-tracer table now covers 15 of 15 terminal shapes

Added the missing `pre_handover_cancelled_in_loop` row to `TestAI37_NoopEquivalence_NoTracerConfigured_NoPanicAcrossTerminalPaths`'s `cases` table (`ai37_noop_equivalence_test.go`), mirroring `ai37_span_lifecycle_test.go`'s own identically-named case with `mustNoTracerClient` in place of `ai37MustClient`. The file's own doc comment already claimed 15 shapes (that part was already correct); the table now actually carries 15 — confirmed empirically: 15 `--- PASS` lines, 0 `FAIL`, for this one subtest alone.

### S-6 (optional) — evaluated, skipped

`runNoPanic`'s `recover()` (`ai37_noop_equivalence_test.go`) only covers the calling goroutine, not `run`'s own producer goroutine spawned inside `Stream()` (`go run(ctx, resp, out, span)`, `stream.go`). Closing this gap would require `run` itself (production code) to install its own `recover()` plus some mechanism to report a caught panic back to the test — a real production-code behavioral change (recovering from a panic changes what the program does, not merely what a test observes), and no test-only injection point exists to wrap a goroutine that production code spawns internally. Out of scope for a test-only diagnostic improvement; skipped, not attempted, per the instruction to skip and say so when a "cheap" fix would actually perturb the paths being exercised.

### Gates (post-remediation, whole module)

| Gate | Command | Result |
|---|---|---|
| Test | `cd backend/agent && make test` (`go test -race -v ./...`) | **PASS** — 9/9 packages `ok`, 0 `FAIL` lines |
| Lint | `cd backend/agent && make lint` (`go vet` + `golangci-lint`) | **PASS** — `0 issues.` |
| Build | `cd backend/agent && make build` | **PASS** — exit 0 |
| Working tree | `git status --porcelain` | clean except this remediation's own commits |

### W-6 — this section is itself the fix

The stat this document recorded before this round (`29 files changed, 4503 insertions(+), 147 deletions(-)`) was stale — it did not count its own prior commit. Final three-dot diff, re-run **after** this remediation's own commits land (so the number left here is the actual final one, not a pre-commit estimate):

```
git diff --stat origin/main...HEAD
31 files changed, 4879 insertions(+), 147 deletions(-)
```

### Out of scope (orchestrator-owned, untouched by this round)

`W-1`, `W-2`, `W-4`, `W-5`, `S-1`, `S-2`, `S-3`, `S-4`, `S-5` — archive-time record corrections (a spec-site declination note, an evidentiary-framing correction, recorded-evidence carry-forward from verify into the archive record, an unassertable-duration scenario narrowing, spec index arithmetic, and a pre-existing `go.work` version drift owned by `agent-module-scaffold`, not by this change). Not edited here; `go.work` itself was not touched.
