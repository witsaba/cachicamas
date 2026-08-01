# Tasks: Streamed tool-call events (AI-18)

> **Apply-time re-reconciliation (2026-08-01, supersedes the note below):** `design.md`'s "reconciled construction model" assumed AI-14's landed *design.md* prose (a bare, never-erroring `Event` constructor deferring validation to `CheckEmit`). Reading AI-15's actual *landed code* (`response_start.go`, `completion.go`) at apply time showed neither takes that shape: both validate eagerly inside the constructor and return `(Event, error)`, exactly like `tool_call.go`'s `NewToolCall`. The spec's own scenario text ("when construction runs, then it fails" — S-ATC-005/-006/-007/-011/-019) already assumed this eager-validation shape. `tool_call_event.go` follows the landed precedent: `NewToolCallStart/Delta/End` return `(Event, error)` and validate via each payload's `validate(at Path) *Violation`, called both at construction (`at = nil`) and again inside `CheckEmit` (`at = Path{At("event")}`) at the true stream emission boundary. All three payloads are block-scoped (`BlockRoleStart/Delta/End`) and additionally implement the unexported `blockPayload` interface (`blockIndex() BlockIndex`) so `CheckEmit`'s generic rule 3 and `CheckStream`'s block-ordering checker work with no checker edit.
>
> ~~Reconciled against AI-14's landed design (obs #2354): constructors return bare `Event` (never errors); validation runs via `CheckEmit` calling each payload's `validate(at Path)`. See `design.md` "Reconciled construction model".~~ — superseded by the paragraph above.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~850–1000 (prod ~180, registry append ~20, tests ~650–800 for 38 scenarios) |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR (size exception) |
| Delivery strategy | exception-ok (~5000-line budget pre-accepted) |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

**Actual changed lines (apply time):** `tool_call_event.go` 291 lines (new), `tool_call_event_test.go` 1341 lines (new) = 1632 authored lines, both committed in `20a483d`. `event.go`/`event_registry_test.go`'s 3-entry append each is real but landed inside sibling AI-16's commit `4f63977` (see Phase 1 note) — zero additional authored lines charged to this commit. Confirms the High-risk forecast; `exception-ok` delivery strategy was pre-accepted per the Review Workload Forecast above, consistent with `size:exception`.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | AI-18.1 lifecycle (kinds, start/delta/end, redaction) | PR 1 | `go test ./src/ai/... -run ToolCallEvent -race -v` | N/A — pure unit, no external harness | `tool_call_event.go` + registry append revertable independently |
| 2 | AI-18.2 delta optionality | PR 1 | `go test ./src/ai/... -run ToolCallDelta -race -v` | N/A — reuses Unit 1's constructors | Test-file-only; no prod diff expected |
| 3 | AI-18.3 interleaving + ordinal | PR 1 | `go test ./src/ai/... -run Interleav -race -v` | N/A — recorded-stream, no live stream harness | Test-file-only |

## Phase 0: Pre-condition

- [x] 0.1 Verify AI-14's `event.go`/`event_descriptor.go`/`sequence.go`/`stream_check.go` exist on this branch/`main`; if absent, RED tasks may proceed, GREEN tasks block until landed.
      **Verified**: all four files present and read in full at apply start (AI-14 + AI-15 landed: `event.go`, `event_descriptor.go`, `sequence.go`, `stream_check.go`, `response_start.go`, `completion.go`, `event_registry_test.go`, `export_test.go`). Precondition satisfied; GREEN tasks unblocked from the start.

## Phase 1: AI-18.1 — Call lifecycle (R-ATC-001…008)

- [x] 1.1 RED `tool_call_event_test.go`: S-ATC-001…003 (three distinct registered kinds, kind derived not tagged, exhaustiveness guard bites scratch kind).
- [x] 1.2 RED: S-ATC-004…007 — `R-ATC-002` empty id/name via construction → `ErrEmpty` (see re-reconciliation note: asserted on the constructor's own returned error, not `CheckEmit`'s).
- [x] 1.3 RED: S-ATC-008…010 — `R-ATC-003` shared block index across start/delta/end, distinct from a text block's index (simulated via `ai.RegisterTestKind`, since AI-16's real text-block kind was not yet guaranteed present at RED time).
- [x] 1.4 RED: S-ATC-011…012 — `R-ATC-004` blockIndex 0 rejected via construction → `ErrOutOfRange`.
- [x] 1.5 RED: S-ATC-013…015 — `R-ATC-005` fragment-only deltas, zero-length/whitespace accepted, no accumulator accessor.
- [x] 1.6 RED: S-ATC-016…018 — `R-ATC-006` byte-equal reconstruction, no re-marshal, fresh-copy accessors.
- [x] 1.7 RED: S-ATC-019…021 — `R-ATC-007` no JSON well-formedness check on call-end path.
- [x] 1.8 RED: S-ATC-022 — `R-ATC-008` redacted `String()`/`GoString()`.
- [x] 1.9 GREEN: create `tool_call_event.go` — `ToolCallStart/Delta/End` structs, `(Event, error)` constructors (re-reconciled shape), exported accessors, unexported `kind()`/`blockIndex()`/`validate(at Path)`.
- [x] 1.10 GREEN: append AI-14 registry entries (constant + table + descriptor + name) for the three kinds.
      **Concurrency note**: `event.go` and `event_registry_test.go` are shared, append-only files also being edited live by sibling AI-16 (text-events) and AI-17 (reasoning-events) in this same worktree. My additions were appended after each sibling's, re-reading the file fresh before every edit (the Edit tool's stale-snapshot guard caught two genuine concurrent changes mid-session). AI-16 committed first (`4f63977`), and — as that commit's own message states — incidentally captured AI-17's and this milestone's already-present, non-conflicting registry entries in the same commit, since all three milestones append to one shared file. Verified via `git show 4f63977 -- backend/agent/src/ai/event.go` that all nine kinds (3 reasoning + 3 text + 3 tool-call) are present and correct. No further edit to either file was needed or made in this milestone's own commit.
- [x] 1.11 GREEN: run `make test`; record green output in this file. **See "Final Evidence" below.**
- [x] 1.12 REFACTOR: dedupe shared rule-1 (`blockIndex >= 1`) validation if repeated 3×; record refactor note.
      **Refactor note**: the rule is repeated 3× verbatim (`if x.block == 0 { return Invalid(ErrOutOfRange, under(at, At("block_index"))...) }`). Evaluated extraction into a shared helper; **declined**, matching sibling AI-16's `text_events.go` (grepped: it keeps the identical inline repetition across its own three payloads rather than extracting one). This matches the package's established posture (`firstViolation`'s own doc: "keeping the order as data rather than as control flow is the reason it is worth three lines: a reviewer reads which rule wins instead of tracing it") — each payload's validation order stays self-contained and locally readable rather than routed through a shared function. Tests re-run green after the decision (no code changed).

## Phase 2: AI-18.2 — Delta optionality (R-ATC-009…010)

- [x] 2.1 RED: S-ATC-023…025 — zero-delta call legal/complete, no truncation signal.
- [x] 2.2 RED: S-ATC-026…028 — whole vs. fragmented indistinguishable after reconstruction, multi-byte-rune split safe.
- [x] 2.3 GREEN: confirm Phase 1 code already satisfies both (test-only leaf); run `make test`; record green.
      **As predicted**: Phase 1's `tool_call_event.go` needed zero changes; all Phase 2 tests passed on first run once written (after one self-inflicted test-side bug fixed, see Issues Found). This is expected — Phase 2 exercises Phase 1's existing code from a new angle, not new production behavior.
- [x] 2.4 REFACTOR: record refactor note (expected: none needed).
      **Refactor note**: none needed in production code, as predicted. One test-local refactor performed as part of Phase 3 (task 3.4) retroactively also simplified Phase 2's S-ATC-026 sub-test (see 3.4's note) — recorded there to avoid duplicating the note.

## Phase 3: AI-18.3 — Interleaving and ordinal (R-ATC-011…012)

- [x] 3.1 RED: S-ATC-029…031 — two interleaved calls partition by block index alone, green under `-race`.
- [x] 3.2 RED: S-ATC-032…034 — ordinal derived by filtering call-start order, distinct from block index, stable.
- [x] 3.3 GREEN: add test-local partition/ordinal helpers (`package ai_test`); run `go test -race -v ./...`; record green.
- [x] 3.4 REFACTOR: extract shared test-local concatenator/ordinal helper if duplicated across phases.
      **Refactor note**: yes, duplication found and removed. Phase 2's S-ATC-026 sub-test and Phase 3's `reconstruct()` helper both looped over `[]ai.Event`, filtered to `ToolCallDelta` payloads and concatenated `Fragment()`. Extracted `deltaFragmentsOf(events []ai.Event) []byte` and rewired both call sites to use it. Re-ran `go test ./src/ai/... -run ToolCall -race -v` after the refactor — still green, all 38 scenarios passing.

## Phase 4: Non-functional, evidence, acceptance

- [x] 4.1 RED: S-ATC-035 (go.mod zero requires + import guards), S-ATC-036 (totality table: zero value, blockIndex 0, empty id/name, zero-length/invalid-UTF-8 fragment, malformed JSON — no panic), S-ATC-037 (every rejection is AI-04's value with landed sentinel).
      S-ATC-035 is satisfied by the **existing, unchanged** guard suite (`import_boundary_test.go`, `go.mod`) per `design.md`'s Testing Strategy table — `tool_call_event.go` imports nothing at all (matching `response_start.go`/`completion.go`), so no new test was needed or written. S-ATC-036 and S-ATC-037 got new tests: `TestToolCallEvents_Totality_NoExportedEntryPointPanics` and `TestToolCallEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel`.
- [x] 4.2 GREEN: fix any gap found; run `make test` + `make lint`; record green + clean lint.
      Gap found and fixed: `gofmt` alignment (2 spots) and a `revive` `package-comments` violation (missing blank line between the file header comment and `package ai`, which made `revive` treat the header as a malformed package doc). Both fixed; see "Final Evidence".
- [x] 4.3 Record red/green/refactor evidence per test-list item (S-ATC-038, NFR-ATC-D) in this file. **Done — see "TDD Cycle Evidence" and "Final Evidence" below.**
- [x] 4.4 Walk acceptance criteria 1–10 against landed code; final `make test`/`make lint` sweep. **Done — see "Acceptance Criteria Walk" below.**

---

## TDD Cycle Evidence

| Task | Scenarios | RED | GREEN | REFACTOR |
|------|-----------|-----|-------|----------|
| 1.1–1.8 | S-ATC-001…022 | ✅ Written; `go vet ./src/ai/...` failed with `undefined: ai.EventKindToolCallStart` (production file did not exist yet) — one consolidated compile-failure capture for all 8 RED sub-tasks, since they share one non-existent production file as their common cause | ✅ `go build ./...` clean, then `go test ./src/ai/... -run 'ToolCall' -race -v` → `ok` after fixing one self-inflicted test bug (S-ATC-025, see Issues Found) | ✅ Dedup evaluated and declined (task 1.12 note); tests re-run green |
| 2.1–2.2 | S-ATC-023…028 | ✅ Written against Phase 1's already-landed code (test-only leaf per design) | ✅ Passed on first real run (after the same S-ATC-025 fix, which lives in this file too) | ✅ None needed in production; test-local dedup recorded under 3.4 |
| 3.1–3.2 | S-ATC-029…034 | ✅ Written; test-local `partitionByBlock`/`reconstruct`/`callOrdinals` helpers authored alongside | ✅ `go test ./src/ai/... -run 'ToolCall' -race -v` → `ok`, 0 races | ✅ Extracted `deltaFragmentsOf`; re-verified green |
| 4.1 | S-ATC-035…037 | ✅ Written (S-ATC-036 totality table, S-ATC-037 consolidated rejection sweep); S-ATC-035 satisfied by existing unchanged guards | ✅ `make test` full suite green (1069 sub-tests, 0 failures) | N/A — non-functional sweep, no dedicated refactor pass beyond 1.12/3.4 |

## Final Evidence

**`go vet ./src/ai/...` (RED, before `tool_call_event.go` existed):**
```
vet: src/ai/tool_call_event_test.go:40:30: undefined: ai.EventKindToolCallStart
```

**`go test ./src/ai/... -run 'ToolCall' -race -v` (GREEN, final run):**
```
ok  	github.com/cachicamas/backend/agent/src/ai	1.560s
```
All `TestToolCallEvents_*` / `TestNewToolCallStart_*` / `TestNewToolCallEnd_*` / `TestToolCallDelta_*` top-level tests and their sub-tests pass (68 nested `--- PASS` lines, 0 `--- FAIL`).

**`make test` (full suite, from `backend/agent/`):**
```
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.613s
ok  	github.com/cachicamas/backend/agent/src/ai	2.864s
```
0 `--- FAIL` lines across 1069 `=== RUN` entries.

**`make lint` (from `backend/agent/`):**
`go vet ./...` clean. `golangci-lint run --config=.golangci.yml ./...`: **zero issues attributable to `tool_call_event.go` or `tool_call_event_test.go`** (confirmed via `grep -c tool_call_event` on the full lint output = 0). One outstanding `revive: package-comments` issue remains, but it names `reasoning_event.go` — sibling AI-17's file, out of this milestone's scope per the explicit "only touch files this milestone's tasks.md names" instruction; not fixed here. Re-checked twice a few minutes apart; still present at the time of this record, so `make lint`'s process exit code is non-zero for reasons entirely outside this milestone's own files.

## Issues Found (during apply, not deviations from spec)

1. **`gofmt` alignment** — two spots in `tool_call_event_test.go` (a trailing-comment column and a struct-tag column) were mis-aligned after edits; fixed with `gofmt -w`.
2. **`revive: package-comments`** — `tool_call_event.go`'s header comment lacked the blank line before `package ai` that every other landed file in this package (`event.go`, `response_start.go`, `completion.go`) uses; without it, `revive` treats the header as a malformed package doc comment. Fixed by adding the blank line.
3. **Own test bug (S-ATC-025)** — the doc-text presence check used `strings.Contains` against the raw file, which fails when the target phrase wraps across a `// `-prefixed comment line. Fixed by normalizing whitespace and stripping `//` tokens before matching, rather than reformatting the production doc comment's natural line wrap to dodge the test.
4. **Test helper name collision** — a `mustFail` helper I first wrote for the Phase 4 consolidated-rejection test collided with an existing unrelated `mustFail(t, name, schema string) error` in `tool_test.go` (same `ai_test` package, shared namespace). Renamed mine to `mustFailToolCallEvent` and restructured its signature to accept a `func() (ai.Event, error)` (Go's multi-value-return-in-call-context rule does not allow a leading `t` argument before a multi-value call).

## Acceptance Criteria Walk (spec.md)

1. Start event exposes identity and tool name before any argument byte — ✅ S-ATC-004 (no delta/end constructed in that sub-test at all).
2. Call-end argument bytes byte-equal to streamed bytes, never re-marshalled — ✅ S-ATC-016/017/018.
3. Zero-delta call and fragmented equivalent reconstruct identically — ✅ S-ATC-026.
4. Two interleaved calls reconstruct independently, no cross-contamination, green under `-race` — ✅ S-ATC-029/030/031; whole suite (including this test) runs under `-race` via `make test`'s pinned body.
5. Each call's ordinal observable from events, stored nowhere — ✅ S-ATC-032/033/034; `callOrdinals` is test-local, not shipped.
6. `make test` green from `backend/agent/` — ✅ confirmed, see Final Evidence.
7. Call-end performs no JSON well-formedness check, no empty-argument canonicalization; AI-30 deferral stated in contract text — ✅ S-ATC-019/020/021.
8. Deltas carry index + fragment only; no accessor returns accumulated arguments — ✅ S-ATC-013/014/015.
9. All three events carry shared block index; 1-based, 0 rejected via `ErrOutOfRange` — ✅ S-ATC-008…012.
10. Three separately registered kinds, exhaustiveness guard covers them, bites a scratch kind — ✅ via `event_registry_test.go`'s extended witness table (landed in `4f63977`); guard mechanism itself unchanged and pre-proven by AI-14/AI-15.

**25/25 tasks complete. All 38 scenarios (S-ATC-001…038) covered. `make test` green, `make lint` clean for this milestone's own files.**
