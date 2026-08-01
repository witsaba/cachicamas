# Apply progress: cachicamas-ai-response-events (AI-15)

> Mode: Strict TDD. Test runner: `make test` (`go test -race -v ./...`) from `backend/agent/`.
> Branch: `feat/2026-08-01-cachicamas-ai-layer1-wave-2` (worktree `cachicamas-worktrees/ai-wave-2`).
> Safety-net baseline (before any AI-15 edit): `go test -race ./...` → `src/agenttest` PASS, `src/ai` PASS, 205 subtests PASS / 0 FAIL in `src/ai`.

## Status: COMPLETE — all 5 phases, 25/25 tasks. `make test` and `make lint` both green. Ready for `sdd-verify`.

## Commits

- `111ae3a` `feat(ai): land the AI-15.1 response-start event (AI-15.1)`
- `eb17291` `feat(ai): land the AI-15.2 completion event (AI-15.2)`
- `75559c1` `docs(openspec): append V-STR-24/V-STR-25 to the contract vocabulary (AI-15.1)`
- `2657d9b` `test(ai): close out AI-15 NFRs - totality and a clean make lint (AI-15 NFR)`

## Phase 1: Setup — COMPLETE (2/2)

- [x] 1.1 — AI-14's code confirmed merged (4 commits: `78eb88a`, `ed7ffa8`, `5f2470f`(AI-14.3, not directly touched), `4de995a`, `65d8be7`; `event.go`, `sequence.go`, `event_descriptor.go`, `stream_check.go` all present and match the task briefing's API exactly). **Gap noted**: AI-14's OpenSpec artifact has NOT been promoted to `openspec/specs/ai-event-envelope/` — it still sits in `openspec/changes/cachicamas-ai-event-envelope/` (unarchived). This is an administrative/archive-phase gap, not a code blocker; the substantive "AI-14 merged" gate holds. Recommend a later archive pass batches AI-14's promotion (possibly alongside AI-15's own archive).
- [x] 1.2 — Register baseline confirmed: § 10 read "116 terms" / "23 stream-side" before any edit.

## Phase 2: AI-15.1 — Response start — COMPLETE (6/6)

- [x] 2.1 RED — `response_start_test.go` written (S-ARP-001…016 + redaction, see below); confirmed failing to compile (`undefined: ai.EventKindResponseStart`).
- [x] 2.2 GREEN — `response_start.go` created (`ResponseStart`, `NewResponseStart`, `ResponseID()`/`ServedModel()`, `Event.ResponseStart()`, `kind()`/`validate`); `event.go` modified to register `EventKindResponseStart` (`eventRegistry` line `{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: false}`, `eventKindFirst`/`eventKindEnd` moved, `# Registered kinds` doc updated). `go vet`/`go test -race -run ResponseStart` green.
- [x] 2.3 REFACTOR — `String()`/`GoString()` added to `response_start.go` (redacting, returns `"responsestart"`), test-first (RED: `rs.GoString undefined` → GREEN). Zero new imports in `response_start.go` (file has no import block at all).
- [x] 2.4 RED — cardinality scenarios (S-ARP-014…016) added to the same test file. No compile/run RED occurred — see deviation note below.
- [x] 2.5 GREEN — confirmed: `CardinalityAtMostOne` registered in 2.2 already satisfies S-ARP-014…016 via AI-14's unmodified `CheckStream`; no checker edit.
- [x] 2.6 REFACTOR — `grep -n "EventKindResponseStart" backend/agent/src/ai/stream_check.go` → no matches (exit 1). Confirmed via `git status --short` that no AI-14 production file (`event_descriptor.go`, `sequence.go`, `stream_check.go`) was touched.

### Deviation from design.md (Phase 2)

design.md's File Changes table lists only `response_start.go`, `response_start_test.go`, `event.go`, and the register spec as touched files. In practice, task 2.2's own instruction ("extend AI-14's exhaustiveness witness list") required also modifying `backend/agent/src/ai/event_registry_test.go`:

1. **Generalized `eventKindWitness`'s field types.** `construct` changed from the witness-only `func(ai.BlockIndex) ai.Event` to `func() (ai.Event, error)`; `read` changed from `func(ai.Event) (ai.WitnessPayload, bool)` to `func(ai.Event) (any, bool)`. This exactly mirrors the established, already-repeated precedent in `content_part_registry_test.go`'s `partKindWitness` (`constructValid func() (Part, error)`, `read func(Part) (any, bool)`), which is extended by every milestone that adds a content-part kind. Without this generalization, `EventKindResponseStart` (whose constructor/accessor shapes differ from the test-only witness) could not be added to the table at all.
2. **Replaced `TestEventKinds_ProductionVocabulary_IsEmpty`.** This test hard-asserted `len(ai.EventKinds()) == 0`, an AI-14-only-milestone fact (R-AEE-006) that becomes categorically false the moment any kind registers. Replaced with `TestEventKinds_ProductionVocabulary_IsExactlyTheRegisteredKinds`, which pins the vocabulary by hand (`productionEventKinds` variable) — mirroring `finish_reason_test.go`'s `theVocabulary` idiom (a hand-written list compared against the package's own answer, not read from it). This keeps the guard meaningful going forward instead of retiring it silently.

Both changes were necessary for `make test` to stay green (a hard success criterion) and are scoped, mechanical, and precedent-matched — not a freelance redesign. No AI-14 **production** file (`event.go` aside, which design.md explicitly assigns to AI-15) was touched; `stream_check.go`, `sequence.go`, and `event_descriptor.go` remain byte-for-byte AI-14's.

### Scenario-to-test mapping notes (Phase 2)

- **S-ARP-003** (exhaustiveness guard covers the kind; a scratch unregistered kind fails and names it) — proven by `event_registry_test.go`'s generalized `eventKindWitnesses` guard (extended above), not repeated as a separate response_start_test.go case.
- **S-ARP-009** (register resolves "served model" to `V-STR-25`, not `V-REQ-21`) and the second half of **S-ARP-016** (checker source names no kind) — treated as review-only per design.md's own Testing Strategy table ("Review-only … no-branch-in-checker (S-ARP-016, S-ARP-026) … Reviewer/diff tasks, not Go tests"). Verified by direct doc-comment review and the grep in 2.6.
- **S-ARP-014** adapted to a single-event stream (no completion) rather than "start then completion," since `Completion` is Phase 3's own leaf and using it here would break Phase 2/3 independence. The at-most-one claim under test (one occurrence never violates) does not require a second kind.

## Phase 3: AI-15.2 — Completion — COMPLETE (9/9)

- [x] 3.1 RED — `completion_test.go` written (S-ARP-017…025); confirmed failing to compile (`undefined: ai.EventKindCompletion`).
- [x] 3.2 GREEN — `completion.go` created (`Completion{reason, usage}`, `NewCompletion` delegating to `FinishReason.Validate`/`Usage.Validate` in order, accessors, `Event.Completion()`); `event.go` extended with `EventKindCompletion` (`eventRegistry` line `{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: true}`). `event_registry_test.go` witness table + production-vocabulary pin extended with the same pattern used for `EventKindResponseStart`.
- [x] 3.3 REFACTOR — `String()`/`GoString()` added test-first (RED: `c.GoString undefined` → GREEN), returns `"completion"`. S-ARP-022 (no new FinishReason/Usage declarations) proven by an AST scan of `completion.go` asserting exactly one declared type (`Completion`) and zero top-level `const` blocks. Zero new imports.
- [x] 3.4-3.5 — terminal scenarios (S-ARP-026…029): no red state occurred — `Terminal: true` was already registered in 3.2, so `CheckStream` (AI-14, unmodified) satisfied every scenario immediately. Documented as expected, same shape as 2.4/2.5.
- [x] 3.6 — `grep -n "EventKindCompletion" stream_check.go` → no matches (exit 1).
- [x] 3.7-3.8 — empty-response-stream scenarios (S-ARP-030…033): response-start→completion legal and `Terminated()==true`; a truncated stream (start only) reports `Terminated()==false` and `Violation()==nil` (informational, not an error); refusal vs normal-stop remain distinct from each other and from `FinishReasonUnknown`. No gap surfaced; no red state (same reason as 3.4/3.5).
- [x] 3.9 — confirmed via `git status --short src/ai/`: only `event.go` (AI-15-assigned by design.md), `event_registry_test.go` (documented deviation), `completion.go`/`completion_test.go` (new). `stream_check.go`, `sequence.go`, `event_descriptor.go` byte-for-byte untouched.

Commit: `eb17291`.

## Phase 4: Register amendment (`R-ARP-011`) — COMPLETE (3/3)

- [x] 4.1 — Appended `V-STR-24` (provider response identity) and `V-STR-25` (served model) rows to `openspec/specs/ai-contract-vocabulary/spec.md` § 4.2, owner AI-15, exact senses from spec.md.
- [x] 4.2 — Added dated `> **Amended 2026-08-01**` blockquote under § 4, matching the AI-02.1/AI-03.1/AI-04.1 pattern (what was appended, which milestone, why the register lacked the term).
- [x] 4.3 — Updated § 10: checklist item 2 range `V-STR-01 … V-STR-23` → `… V-STR-25`; term count `23 stream-side … 116 terms` → `25 stream-side … 118 terms`; amendment history parenthetical extended. `git diff` confirms exactly 2 new rows + 1 new blockquote + 2 count-line edits — no existing row renumbered, reworded, reordered or removed (S-ARP-034…037).

Commit: `75559c1`.

## Phase 5: Non-functional verification & cleanup — COMPLETE (5/5)

- [x] 5.1 — `TestResponseEvents_ExtremeInputs_NeverPanic` (8 cases: zero-value payloads read every way, empty/invalid construction inputs, negative usage counts, wrong-kind accessors, `CheckEmit`/`CheckStream` on zero/invalid events) added to `completion_test.go`. All PASS, no panics (S-ARP-039).
- [x] 5.2 — `go.mod` confirmed zero `require` lines; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` and `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` both PASS (S-ARP-038).
- [x] 5.3 — `grep -n "errors.New\|var Err" response_start.go completion.go` → no matches; every rejection in both files reports through AI-04's landed `ErrEmpty`/`ErrNotInVocabulary`/`ErrOutOfRange`, no new sentinel or failure type (S-ARP-040).
- [x] 5.4 — Final `make test` and `make lint` both green (see below). One lint fix applied along the way: 2 staticcheck `ST1023` findings in `completion_test.go` (redundant explicit type on `:=`-inferrable declarations), fixed, re-verified.
- [x] 5.5 — 5-item RED→GREEN→REFACTOR evidence log filled in `tasks.md` (S-ARP-041).

Commit: `2657d9b`.

### Final green output

```
$ make test    (from backend/agent/)
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.426s
ok  	github.com/cachicamas/backend/agent/src/ai	2.823s
# 227 subtests PASS, 0 FAIL, exit 0

$ make lint    (from backend/agent/)
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
# exit 0
```

## TDD Cycle Evidence (cumulative)

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 2.1–2.2 | `response_start_test.go` | Unit | ✅ 205/205 (`src/ai`) | ✅ `undefined: ai.EventKindResponseStart` | ✅ `go test -race -run ResponseStart` all PASS | ✅ 16 scenario subtests across 5 requirement groups | — |
| 2.3 | `response_start_test.go` | Unit | ✅ (2.2's green) | ✅ `rs.GoString undefined` | ✅ `go test -race ./src/ai/...` PASS | ➖ single behavior (redaction) | ✅ zero new imports |
| 2.4–2.5 | `response_start_test.go` | Unit | ✅ (2.3's green) | ➖ no red state (generic descriptor already satisfied it — expected, documented) | ✅ PASS | ✅ 2 cases (single start / two starts) | — |
| 2.6 | n/a (grep) | Review | — | — | — | — | ✅ `grep` confirms zero checker branches |
| 3.1–3.2 | `completion_test.go` | Unit | ✅ 211/211 (`src/ai`) | ✅ `undefined: ai.EventKindCompletion` | ✅ `go test -race -run Completion` all PASS | ✅ 9 scenario subtests across 3 requirement groups | — |
| 3.3 | `completion_test.go` | Unit | ✅ (3.2's green) | ✅ `c.GoString undefined` | ✅ PASS | ➖ single behavior (redaction) | ✅ zero new imports; AST scan confirms exactly one declared type |
| 3.4–3.5 | `completion_test.go` | Unit | ✅ (3.3's green) | ➖ no red state (Terminal:true already registered — expected, documented) | ✅ PASS | ✅ 4 cases (registration, ends-in-completion, event-after, two-completions) | — |
| 3.6 | n/a (grep) | Review | — | — | — | — | ✅ `grep` confirms zero checker branches |
| 3.7–3.8 | `completion_test.go` | Unit | ✅ (3.6's green) | ➖ no red state (mechanism already registered — expected, documented) | ✅ PASS | ✅ 4 cases (legal-empty, success-readable, truncated-distinguishable, refusal-vs-stop) | — |
| 3.9 | n/a (git status) | Review | — | — | — | — | ✅ confirmed no AI-14 production file touched |
| 4.1–4.3 | `ai-contract-vocabulary/spec.md` | Docs | n/a (docs-only) | n/a | n/a | n/a | ✅ `git diff` confirms append-only edit |
| 5.1 | `completion_test.go` | Unit | ✅ (Phase 4's green) | n/a (additive NFR test, no prior state to break) | ✅ 8/8 cases PASS, no panics | ✅ 8 extreme-input cases | — |
| 5.2–5.3 | n/a (grep/existing guards) | Review | — | — | ✅ both import guards PASS; zero new sentinels | — | — |
| 5.4 | full suite | Unit | ✅ | n/a | ✅ `make test` 227/227, `make lint` 0 issues | — | ✅ fixed 2 staticcheck findings, re-verified both green |

### Test Summary (final)
- Full `src/ai` suite: PASS, 0 FAIL (`go test -race ./src/ai/...`); `make test` (both `src/ai` and `src/agenttest`): 227 subtests PASS, 0 FAIL, exit 0
- `make lint`: 0 issues, exit 0
- Subtests passing: 227 (baseline 205 + 22 net new/updated across Phases 2, 3 and 5)
- Layers used: Unit (all) — no integration/E2E layer applicable to this pure-Go value-type contract
- Pure functions created: `ResponseStart.validate`, `NewResponseStart`, `Completion.validate`, `NewCompletion` (all pure — no side effects, no I/O)

## Deviations summary (for sdd-verify)

1. **`event_registry_test.go` generalization** (Phases 2 and 3) — required by task 2.2's own "extend AI-14's exhaustiveness witness list" instruction; not listed in design.md's File Changes table. Generalized `eventKindWitness.construct`/`.read` field types to match `content_part_registry_test.go`'s established per-kind-addition precedent, and replaced the now-false `TestEventKinds_ProductionVocabulary_IsEmpty` with a hand-pinned equivalent. See Phase 2's deviation note above for full detail. No AI-14 production file touched.
2. **AI-14 OpenSpec promotion gap** (Phase 1) — AI-14's code is merged and correct, but its OpenSpec artifact has not been archived/promoted from `openspec/changes/cachicamas-ai-event-envelope/` to `openspec/specs/ai-event-envelope/`. Administrative, not a code blocker; flagged for a later archive pass.
3. **Engram topic-key reuse** — `sdd/cachicamas-ai-response-events/apply-progress` upserted onto observation id `#2173`, which previously held the CLOSED apply-progress for a *prior* milestone numbering of this same change name (pre-restart AI-12, PR #76, merged and archived separately at observation `#2181`). This is correct topic_key behavior (one canonical slot per change name) and does not lose historical data — `#2181`'s archive report is untouched and remains the historical record — but is noted here for transparency since the observation id carried over from an unrelated-looking prior milestone.

None of these are design errors requiring rework; all are additive, scoped, and necessary for the stated success criteria (`make test`/`make lint` green).
