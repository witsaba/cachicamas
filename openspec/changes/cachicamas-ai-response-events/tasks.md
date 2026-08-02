# Tasks: Response lifecycle events (AI-15)

> **Reconciliation note (read first).** AI-14's landed design (`openspec/changes/cachicamas-ai-event-envelope/design.md`) was diffed against this change's `design.md` before writing these tasks. Symbol names (`Event`, `EventKind`, `Event.Kind()`, `Event.Sequence()`) matched verbatim — no rename needed. Three refinements were applied to `design.md`: (1) the "registration table, and descriptor table" was corrected to ONE combined `eventRegistry []eventRegistration` table in `event.go`, embedding an inline `EventDescriptor{Role, Cardinality, Terminal}` literal per new kind; (2) the Data Flow diagram was corrected to name `CheckStream`/`StreamReport` and to add the previously-missing `CheckEmit(e Event) error` step between `Stamp` and recording; (3) the "no terminal event" open question resolved to `StreamReport.Terminated() == false` with `Violation() == nil`. No AI-14 file changes are required.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~700–1300 (2 payload files + 2 large scenario test files + `event.go` append + register spec rows) |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR, `size:exception` (5000-line budget pre-accepted this session) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | AI-15.1 response start: payload, `ErrEmpty`, at-most-one | PR 1 (size:exception) | `cd backend/agent && go test -race -run ResponseStart ./...` | `make test` (full suite) | `response_start.go`, `response_start_test.go`, its `event.go` const+registry line — revertable without touching completion |
| 2 | AI-15.2 completion: payload, absence, terminal, empty-response legality | PR 1 (size:exception) | `cd backend/agent && go test -race -run Completion ./...` | `make test` (full suite) | `completion.go`, `completion_test.go`, its `event.go` const+registry line — independent of unit 1 |
| 3 | Register amendment `R-ARP-011` + NFR verification | PR 1 (size:exception) | N/A — docs diff, reviewed not tested | N/A (no runtime behavior; text-only spec append) | `openspec/specs/ai-contract-vocabulary/spec.md` rows/blockquote/counts — independent text revert |

## Phase 1: Setup

- [x] 1.1 Confirm `openspec/specs/ai-event-envelope/` is promoted (AI-14 merged) and re-read `backend/agent/src/ai/event.go`, `event_descriptor.go` to find exact insertion points (kind const block, `eventKindEnd`, `eventRegistry`, kind-doc list). **Note**: AI-14's code is merged (4 landed commits, `event.go`/`sequence.go`/`event_descriptor.go`/`stream_check.go` all present, `make test`/`make lint` clean at baseline) — the substantive gate. Its OpenSpec artifact is NOT yet promoted to `openspec/specs/ai-event-envelope/` (still sits in `openspec/changes/cachicamas-ai-event-envelope/`) — an administrative archive-phase gap, not a code blocker. Insertion points confirmed by direct read.
- [x] 1.2 Confirm current register counts in `openspec/specs/ai-contract-vocabulary/spec.md` § 10 read 116 total / 23 stream-side before any edit (baseline for `R-ARP-011`). **Confirmed**: § 10 read "116 terms" / "23 stream-side" before any edit.

## Phase 2: AI-15.1 — Response start (leaf)

- [x] 2.1 RED — `backend/agent/src/ai/response_start_test.go` (`package ai_test`): registered/distinct kind (S-ARP-001…003), byte-exact accessors incl. punctuation/mixed-case (S-ARP-004…006), served-model independence from request model (S-ARP-007…009), `ErrEmpty` on empty responseID/servedModel/both/zero-value (S-ARP-010…013). Record red output.
- [x] 2.2 GREEN — create `response_start.go`: `ResponseStart` struct, `NewResponseStart`, `ResponseID()`/`ServedModel()`, `Event.ResponseStart()`, `kind()`/`validate(at Path) *Violation`; append `EventKindResponseStart` const (move `eventKindEnd`) + `eventRegistry` line `EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: false}` in `event.go`; extend AI-14's exhaustiveness witness list. Record green output (`make test`). **Deviation**: extending the witness list required generalizing `event_registry_test.go`'s `eventKindWitness.construct`/`read` field types (from witness-only signatures to `func() (ai.Event, error)` / `func(ai.Event) (any, bool)`, mirroring `content_part_registry_test.go`'s established per-kind-addition pattern) and replacing `TestEventKinds_ProductionVocabulary_IsEmpty` (an AI-14-only invariant, false once any kind registers) with a hand-pinned `TestEventKinds_ProductionVocabulary_IsExactlyTheRegisteredKinds`. Not listed in design.md's File Changes table; required for `make test` to stay green. See apply-progress deviation note.
- [x] 2.3 REFACTOR — redacting `String()`/`GoString()` (`V-FAIL-13`), confirm zero new imports, record refactor note. **Confirmed**: `response_start.go` has zero import statements.
- [x] 2.4 RED — extend scenario tests with two-response-start-per-stream cardinality case via `CheckStream` (S-ARP-014…016). Record red output. **Note**: this step did not produce a compile/run RED — the generic `CardinalityAtMostOne` registration from 2.2 already satisfied it the moment the test was written (expected; matches "no checker edit" and registry-driven extensibility).
- [x] 2.5 GREEN — confirm generic `CardinalityAtMostOne` registration alone (2.2) satisfies S-ARP-014…016, no checker edit. Record green output.
- [x] 2.6 REFACTOR — grep review: zero checker branches name `EventKindResponseStart`; record note. **Confirmed**: `grep -n "EventKindResponseStart" src/ai/stream_check.go` → no matches (exit 1).

## Phase 3: AI-15.2 — Completion (leaf)

- [x] 3.1 RED — `backend/agent/src/ai/completion_test.go` (`package ai_test`): registered/distinct kind (S-ARP-017…018), embeds `FinishReason`/`Usage` unchanged (S-ARP-019), invalid finish reason → `ErrNotInVocabulary` (S-ARP-020), negative usage count → `ErrOutOfRange` (S-ARP-021), no new declarations (S-ARP-022), absence survives incl. `Tokens(0)` vs absent (S-ARP-023…025). Record red output. **Red**: `undefined: ai.EventKindCompletion`.
- [x] 3.2 GREEN — create `completion.go`: `Completion{reason FinishReason; usage Usage}`, `NewCompletion` delegating to `FinishReason.Validate`/`Usage.Validate` in `V-FAIL-04` order, accessors, `Event.Completion()`; append `EventKindCompletion` const + `eventRegistry` line `EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: true}`. Record green output. **Green**: `go test -race ./src/ai/...` PASS. Same `event_registry_test.go` witness-table + production-vocabulary-pin extension pattern as 2.2.
- [x] 3.3 REFACTOR — redacting renders, confirm no new `FinishReason`/`Usage` field or parallel type declared, record note. **Confirmed**: `String()`/`GoString()` added test-first (RED: `c.GoString undefined` → GREEN); S-ARP-022's AST-scan test confirms `completion.go` declares exactly one type (`Completion`), zero top-level consts; zero new imports.
- [x] 3.4 RED — terminal scenarios: completion is stream terminal (S-ARP-026…027), event after completion → violation (S-ARP-028), two completions → violation (S-ARP-029). Record red output. **Note**: no compile/run RED occurred — `Terminal: true` was already registered in 3.2; test passed immediately (expected, matches "no checker edit").
- [x] 3.5 GREEN — confirm `Terminal: true` (3.2) satisfies S-ARP-026…029 with no checker edit. Record green output. **Confirmed**: `go test -race -run TestCompletion_IsTerminal` PASS.
- [x] 3.6 REFACTOR — grep review: zero checker branches name `EventKindCompletion`; record note. **Confirmed**: `grep -n "EventKindCompletion" src/ai/stream_check.go` → no matches (exit 1).
- [x] 3.7 RED — empty-response-stream scenarios: response-start→completion with no content is legal (S-ARP-030), success readable from terminal event alone (S-ARP-031), distinguishable from no-terminal stream via `StreamReport.Terminated()` (S-ARP-032), refusal vs normal-stop remain distinguishable (S-ARP-033). Record red output. **Note**: no compile/run RED — same reason as 3.4 (mechanism already registered).
- [x] 3.8 GREEN — confirm `CheckStream`/`StreamReport.Terminated()` (AI-14, unmodified) satisfies this; if a gap surfaces, fix only in AI-15's own files, never `stream_check.go`. Record green output. **Confirmed**: no gap surfaced; `go test -race -run TestEmptyResponseStream` PASS.
- [x] 3.9 REFACTOR — record note; confirm no AI-14 file was touched. **Confirmed**: `git status --short src/ai/` shows only `event.go` (AI-15-assigned), `event_registry_test.go` (documented deviation), `completion.go`/`completion_test.go` (new) — `stream_check.go`, `sequence.go`, `event_descriptor.go` untouched.

## Phase 4: Register amendment (`R-ARP-011`)

- [x] 4.1 Append `V-STR-24` (provider response identity) and `V-STR-25` (served model) rows to `openspec/specs/ai-contract-vocabulary/spec.md` § 4.2, owner AI-15, exact senses from the spec table.
- [x] 4.2 Add dated § 4 amendment blockquote (what appended, which milestone, why the register lacked the term), matching the AI-02.1/AI-03.1/AI-04.1 pattern. Dated 2026-08-01.
- [x] 4.3 Update § 10 counts 116→118, stream-side 23→25; diff § 3–§ 8 to confirm no existing row renumbered/reworded/reordered/removed (S-ARP-034…037). **Confirmed** via `git diff`: exactly 2 new rows, 1 new blockquote, 2 count-line edits — no existing row touched.

## Phase 5: Non-functional verification & cleanup

- [x] 5.1 Totality table test: zero-value payloads, empty strings, invalid finish reason, fully absent usage, negative count — assert no panic on any exported entry point (S-ARP-039). **Green**: `TestResponseEvents_ExtremeInputs_NeverPanic` (8 cases) added to `completion_test.go`, `go test -race -run TestResponseEvents_ExtremeInputs_NeverPanic` PASS.
- [x] 5.2 Verify `backend/agent/go.mod` still zero requires and both AI-00 import guards pass (S-ARP-038). **Confirmed**: `go.mod` has zero `require` lines; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` and `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` both PASS.
- [x] 5.3 Review: every rejecting scenario's failure is AI-04's value with a landed sentinel, no new sentinel/type (S-ARP-040). **Confirmed**: `grep -n "errors.New\|var Err" response_start.go completion.go` → no matches; both files reference only `ErrEmpty`, `ErrNotInVocabulary`, `ErrOutOfRange` (AI-04's landed sentinels, the latter two via delegation to `FinishReason.Validate`/`Usage.Validate`).
- [x] 5.4 Run `make test` (`go test -race -v ./...`) and `make lint` in `backend/agent/`; record final green output. **Green** (see below). One lint fix applied: `completion_test.go` had 2 staticcheck `ST1023` issues (redundant explicit type on `:=`-inferrable declarations) — simplified to `:=`, re-verified `make test`/`make lint` both clean.
- [x] 5.5 Fill the 5-item RED→GREEN→REFACTOR evidence log (S-ARP-041): AI-15.1 items 1–2, AI-15.2 items 1–3, each with recorded red, green, refactor note. **See table below.**

### 5-item RED → GREEN → REFACTOR evidence log (S-ARP-041)

| # | Item | Tasks | RED | GREEN | REFACTOR |
|---|------|-------|-----|-------|----------|
| 1 | AI-15.1 — fields, byte-exactness, `ErrEmpty` | 2.1–2.3 | `undefined: ai.EventKindResponseStart` (compile failure) | `go test -race -run ResponseStart` all PASS | `String()`/`GoString()` added test-first (RED: `rs.GoString undefined` → GREEN); zero new imports |
| 2 | AI-15.1 — `at-most-one` via checker | 2.4–2.6 | No red state — `CardinalityAtMostOne` (item 1's registration) already satisfied S-ARP-014…016 the moment the test was written; documented as expected (registry-driven extensibility, no checker edit) | `go test -race -run ResponseStart` all PASS | grep confirms zero `stream_check.go` branches name `EventKindResponseStart` |
| 3 | AI-15.2 — embed unchanged + absence | 3.1–3.3 | `undefined: ai.EventKindCompletion` (compile failure) | `go test -race -run Completion` all PASS | `String()`/`GoString()` added test-first (RED: `c.GoString undefined` → GREEN); AST scan confirms exactly one declared type, zero new imports |
| 4 | AI-15.2 — terminal | 3.4–3.6 | No red state — `Terminal: true` (item 3's registration) already satisfied S-ARP-026…029; documented as expected | `go test -race -run TestCompletion_IsTerminal` PASS | grep confirms zero `stream_check.go` branches name `EventKindCompletion` |
| 5 | AI-15.2 — empty response legal vs. no-terminal | 3.7–3.9 | No red state — same mechanism, already registered; documented as expected | `go test -race -run TestEmptyResponseStream` PASS | confirmed via `git status` that no AI-14 file (`stream_check.go`/`sequence.go`/`event_descriptor.go`) was touched across the whole change |

### Final green output (`backend/agent/`)

```
$ make test
...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.426s
ok  	github.com/cachicamas/backend/agent/src/ai	2.823s
# 227 subtests PASS, 0 FAIL, exit 0

$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
# exit 0
```

> **Deviation note**: exceeds the sdd-tasks 530-word budget for the same reason the spec and design recorded it — five separate RED/GREEN/REFACTOR test-list items across two leaves plus a register amendment, each needing its scenario-ID traceability preserved per `NFR-ARP-D`/`S-ARP-041`.
