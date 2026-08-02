# Tasks: Indexed text block events (AI-16)

> Reconciled against AI-14's landed `design.md` on 2026-08-01 — see `design.md` reconciliation register (A1–A5, all resolved). `BlockIndex` (not raw `uint64`) and `blockPayload.blockIndex()` are now binding.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~260–320 (`event.go` +~15, `event_registry_test.go` +~25, `text_events.go` ~140, `text_events_test.go` ~150) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | AI-16.1 lifecycle (3 kinds, block index, interleave) | PR 1 (single) | `go test -race -run TestText -v ./src/ai/...` | N/A — pure value construction, no I/O; `go test -race -v ./...` is the harness | `text_events.go`/`text_events_test.go` + `event.go` registry rows revert cleanly, no downstream imports yet |
| 2 | AI-16.2 byte fidelity + AI-16.3 zero-delta | PR 1 (same, appended) | `go test -race -run TestTextDelta -v ./src/ai/...` | N/A — same reason | Same files; struct/function additions only |

## Phase 0: Blocking Precondition

- [x] 0.1 Confirm `backend/agent/src/ai/event.go` and `event_registry_test.go` exist with landed `Event`, `EventKind`, `eventRegistry`, `BlockIndex`, `blockPayload`, `CheckEmit`, `CheckStream` (AI-14 applied). **STOP if absent.** — confirmed present at apply start.
- [x] 0.2 Confirm `backend/agent/src/ai/response_start.go`/`completion.go` exist (AI-15 applied — dependency per milestone header). **STOP if absent.** — confirmed present at apply start.
- [x] 0.3 Re-diff design.md's Interfaces block against the actual landed `event.go`/`event_descriptor.go` source (not just AI-14's design doc) — adjust field/type names only if code diverged from its own design. — read both files directly; design.md's reconciliation register (A1–A5) already matches the landed source verbatim (`BlockIndex`, `blockPayload`, `EventDescriptor{Role,Cardinality,Terminal}`). No further adjustment needed.

## Phase 1: AI-16.1 — Text block lifecycle (R-ATE-001…005)

- [x] 1.1 RED — `text_events_test.go`: `S-ATE-001..003` three kinds distinct, registered, exhaustiveness guard covers all three + scratch-kind bite (`event_registry_test.go`). — confirmed RED: `go vet ./src/ai/...` failed with `undefined: ai.EventKindTextBlockStart`.
- [x] 1.2 GREEN — `event.go`: add `EventKindTextBlockStart/TextDelta/TextBlockEnd` constants + `eventRegistry` rows (`BlockRoleStart/Delta/End`, `CardinalityAny`, `Terminal:false`).
- [x] 1.3 RED — `S-ATE-004..005` block index stamped, readable, order-independent; `S-ATE-006..008` index 0 rejected at construction (`ErrOutOfRange` at `block`), first block = 1. — written together with 1.1 in the same test file batch (Go whole-package compilation makes finer RED granularity impractical; see apply-progress).
- [x] 1.4 GREEN — `text_events.go`: `TextBlockStart`/`TextDelta`/`TextBlockEnd` structs (`block BlockIndex`), `New…` constructors, `validate`, `Block()`, `blockIndex()`, `kind()`, `Event` accessors.
- [x] 1.5 RED — `S-ATE-009..011` shared one index space, interleaved text/non-text blocks disambiguated by index alone, no family tag/per-family space exists.
- [x] 1.6 GREEN — confirm via test only (no production code change expected; index space is inherently shared by using one `BlockIndex` type). — confirmed: no production code needed beyond the shared `BlockIndex` type already in place.
- [x] 1.7 RED — `S-ATE-012..013` delta carries only the new fragment; no accumulated-content accessor exists anywhere on the three kinds.
- [x] 1.8 GREEN — `TextDelta.delta string` field + `Delta()` getter only; no accumulator method added.
- [x] 1.9 REFACTOR — dedupe table-driven construction tests across the three kinds; record red/green output per item in this file's checklist notes. — introduced `mustTextBlockStart`/`mustTextDelta`/`mustTextBlockEnd` helpers (mirroring `response_start_test.go`'s `mustResponseStart`), retrofit S-ATE-004/009/010 subtests to use them. Verified GREEN in an isolated scratch copy (sibling AI-18's `reasoning_event_test.go` was transiently mid-GREEN in the real worktree at this point — see apply-progress); all `TestText*` tests pass.

## Phase 2: AI-16.2 — Byte fidelity (R-ATE-006…009)

- [x] 2.1 RED — `S-ATE-014..015` arbitrary fragment split reconstructs byte-exactly incl. leading/trailing whitespace and interior newlines, via test-local concatenator.
- [x] 2.2 GREEN — add test-local concatenator helper inside `text_events_test.go` (not exported).
- [x] 2.3 RED — `S-ATE-016..018` multi-byte rune split across delta boundary: both deltas construct, no byte altered, reconstruction decodes correctly, no replacement char, GoDoc uses "byte" wording.
- [x] 2.4 GREEN — confirm `NewTextDelta` never validates/repairs UTF-8; add GoDoc sentence on `TextDelta`. — GoDoc sentence was already written in phase 1's field comment ("raw bytes... not... well-formed UTF-8"); S-ATE-014/015/016/017/018/019/020/021/023 all passed on first run without any production change (Phase 1's implementation already satisfies these "absence of restriction" properties) — only S-ATE-022 was genuinely RED.
- [x] 2.5 RED — `S-ATE-019..021` whitespace-only and zero-length fragments accepted, no `ErrEmpty`.
- [x] 2.6 GREEN — confirm `validate` never calls the AI-06 text-emptiness rule.
- [x] 2.7 RED — `S-ATE-022..023` fragment over `MaxTextLen` rejected (`ErrOutOfRange` at `delta`); exactly `MaxTextLen` succeeds. — confirmed RED: S-ATE-022 failed (`ai.NewTextDelta` with a `MaxTextLen+1`-byte fragment wrongly succeeded); S-ATE-023 passed trivially (no upper bound existed yet to violate).
- [x] 2.8 GREEN — `validate`: `len(delta) <= MaxTextLen` check, ordered after the block-index check (V-FAIL-04).
- [x] 2.9 REFACTOR — extract shared `MaxTextLen`-boundary table cases if duplicated with `text_content_test.go` patterns; record red/green output. — note: `text_content_test.go` does not exist in this repo; the actual MaxTextLen-boundary precedent lives in `content_part_test.go` (`TestNewText_RuleViolations_FailWithTheDocumentedSentinels`). Followed its idiom (over-bound / exact-bound cases) rather than sharing code, since `TextDelta` and `textPayload` are distinct unexported types with no shared test-helper surface. Also fixed a pre-existing `gofmt` alignment drift in the shared `event_registry_test.go` (mechanical whitespace only, caused by a concurrent sibling's long line elsewhere in the same var block).

## Phase 3: AI-16.3 — Zero-delta blocks (R-ATE-010…011)

- [x] 3.1 RED — `S-ATE-024..026` start immediately followed by end (no delta) is legal, reconstructs to empty, not confused with an unterminated block when mixed with a multi-delta block.
- [x] 3.2 GREEN — confirm no production change needed (zero-delta legality is emergent per design); if `CheckStream`/`CheckEmit` interaction needs a scenario-level integration test, add it here. — added the integration test against AI-14's `CheckStream`; passed immediately with zero production changes, exactly as design.md predicted.
- [x] 3.3 RED — `S-ATE-027..028` export-enumeration test: no accumulator/reconstructor/transcript-rebuilder is exported from the package; concatenator lives only in the test package.
- [x] 3.4 GREEN — confirm via AST/reflect export-scan test only; no production code added (R-ATE-011). — the exact-surface AST enumeration passed on first write, confirming the exported surface built in phases 1–2 contains nothing beyond the documented constructors/accessors/getters.
- [x] 3.5 REFACTOR — finalize `text_events.go` GoDoc (family comment), record red/green output for 3.1–3.4. — added a "block with no deltas is not a special case" section to the file-level doc comment referencing R-ATE-010 and CheckStream; re-verified all `TestText*` green and `gofmt`-clean after the doc-only change.

## Phase 4: Non-functional + wiring

- [x] 4.1 RED — `S-ATE-029` `go.mod` still zero requires; both AI-00 import guards pass. — no new test: this is already covered generically by `import_boundary_test.go`'s existing `TestLayer1_ModuleHasNoDependencies_ZeroRequires` and `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, which apply to the whole package regardless of what AI-16 adds.
- [x] 4.2 GREEN — no dependency added; run `go list -m all` / existing guard tests to confirm. — confirmed: `backend/agent/go.mod` reads `module github.com/cachicamas/backend/agent` + `go 1.26.3` only, no `require` block.
- [x] 4.3 RED — `S-ATE-030` totality table: zero-value payloads, block index 0, zero-length/invalid-UTF-8/over-long fragment through every exported entry point — none panics.
- [x] 4.4 GREEN — add table-driven totality test in `text_events_test.go`; fix any panic found (expected: none, per design). — passed immediately on first run (confirmatory, no panic found, matching design's "expected: none").
- [x] 4.5 RED — `S-ATE-031` every rejecting scenario resolves through AI-04's `Violation`/sentinels, position names the offending field.
- [x] 4.6 GREEN — confirm via `errors.Is` + `Path` assertions already written in Phases 1–2; no new sentinel introduced. — confirmed: every rejection path (S-ATE-006, S-ATE-022) asserts `errors.Is(err, ai.ErrOutOfRange)` + `violation.Path().String()`; `ai.ErrOutOfRange` is AI-04's existing landed sentinel, no new one introduced.
- [x] 4.7 `backend/agent/src/ai/doc.go`: append one sentence noting the text-event family, if not already covered by AI-14's/AI-15's edits. — confirmed already covered: doc.go's "What comes back" section already states generically "AI-15 … AI-20 add those [event kinds] without editing this package's AI-14-owned contracts" — no per-milestone edit needed there; the actual six-step-procedure "documented list" requirement (step 6) was satisfied in `event.go`'s `EventKind` GoDoc bullet list instead (task 1.2/AI-16.1 commit).

## Phase 5: Verification

- [x] 5.1 Run `make test` (`go test -race -v ./...`) in `backend/agent/` — record full green output in this file. — `make test` exit 0; `ok github.com/cachicamas/backend/agent/src/agenttest` and `ok github.com/cachicamas/backend/agent/src/ai`; all 13 `TestText*` test functions pass, including a cross-milestone check from AI-18's `reasoning_event_test.go` (`TestTextEventKinds_ExposeNoReasoningSurface`) that also passes against this milestone's exported surface.
- [x] 5.2 Run `make lint` — record clean output. — **not clean**: `make lint` exits 2 with exactly one `revive` `package-comments` finding, entirely in `backend/agent/src/ai/reasoning_event.go` (AI-18's file — its header comment lacks the blank-line separation from `package ai` that AI-14's own NFR commit established as the house fix for this exact finding class). `gofmt -l` and `go vet` are clean for every file this milestone (AI-16) touches (`text_events.go`, `text_events_test.go`, `event.go`, `event_registry_test.go`). Per the explicit scope boundary (do not touch AI-17/AI-18's files), `reasoning_event.go` was left untouched — this is AI-18's own NFR close-out to make, not AI-16's.
- [x] 5.3 Walk `S-ATE-001..032` against `tasks.md` — confirm each carries recorded red output, green output, and a refactor note (`NFR-ATE-D`, `S-ATE-032`). — walked: all 32 scenarios (R-ATE-001 through NFR-ATE-D) are covered by a named test/subtest above with recorded RED/GREEN/REFACTOR notes in Phases 1–4; several scenarios (S-ATE-014/015/016/017/018/019/020/021/023/024/025/026/027/028/030) passed on first write with no production change, honestly recorded as confirmatory rather than behavioral RED, matching design.md's own "no production change needed" expectation for those legs.
- [x] 5.4 Cross-check `R-ATE-004` binding note: confirm no per-family index space was introduced (reviewer scan of `text_events.go`). — confirmed via `grep`: the only index-typed field anywhere in `text_events.go` is `block BlockIndex` (AI-14's shared type) on all three structs; no `TextBlockIndex` or other per-family type exists.

## Rollback boundary

Revert the commit range touching `text_events.go`, `text_events_test.go`, and the three registry rows in `event.go`/`event_registry_test.go`. Nothing outside `backend/agent/src/ai` changes; no consumer imports the new kinds yet (AI-28 is future).
