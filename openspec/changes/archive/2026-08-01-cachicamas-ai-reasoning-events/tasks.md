# Tasks: Reasoning delta events (AI-17)

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~900–1300 |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR (size:exception) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Lifecycle payloads + registry | PR 1 | `go test -race -run TestReasoning -v ./...` | N/A — pure contract package, no I/O | revert `reasoning_event.go`, test file, 3 `event.go` rows |
| 2 | Token delivery | PR 1 | same, `-run Token` | N/A | additive methods, same rollback |
| 3 | Redacted/signature-only | PR 1 | same, `-run Redacted\|TokenOnly` | N/A | additive methods, same rollback |

Estimate fits the session's accepted 5000-line exception budget; units are commit boundaries inside one PR, not separate PRs.

## Phase 0: Foundation

- [x] 0.1 Create `backend/agent/src/ai/reasoning_event.go`: 3 payload structs (`blockIndex BlockIndex` field), unexported `kind()` / `blockIndex() BlockIndex` (satisfies AI-14's `blockPayload`). No validation yet.
- [x] 0.2 Modify `backend/agent/src/ai/event.go`: append 3 `EventKind` constants + `eventRegistry` rows, descriptors `{Role: BlockRoleStart/Delta/End, Cardinality: CardinalityAny, Terminal: false}`.

## Phase 1: AI-17.1 — Reasoning block lifecycle (R-ARE-001..008)

- [x] 1.1 RED — `reasoning_event_test.go` (`package ai_test`): S-ARE-001..022 — registry membership + exhaustiveness bite, family distinctness from text kinds, block-index-0 rejection, shared stream-wide index space, fragment fidelity incl. rune-split, zero-length/whitespace legality, `MaxTextLen` bound, zero-delta block, no public accumulator scan. Record red `go test` output.
- [x] 1.2 GREEN — Implement `NewReasoningBlockStart`, `NewRedactedReasoningBlockStart`, `NewReasoningDelta(start, fragment)`, accessors (`BlockIndex`, `Redacted`, `Fragment`), `Event.ReasoningBlockStart/Delta()`. Order: index 0 → `ErrOutOfRange`; fragment > `MaxTextLen` → `ErrOutOfRange`. Record green output.
- [x] 1.3 REFACTOR — Extract shared index/bound checks; confirm `make lint` clean. Record refactor note.

## Phase 2: AI-17.2 — Token delivery (R-ARE-009..011)

- [x] 2.1 RED — S-ARE-023..030: token carried on block-end only, byte-exactness over AI-07's `opaqueTokens()`, aliasing (caller-buffer + reader-copy mutation), `MaxReasoningTokenLen` bound, two-result presence accessor distinguishing absent vs zero-length. Record red output.
- [x] 2.2 GREEN — Implement `NewReasoningBlockEnd(start, token)`, `Token() ([]byte, bool)` (copy in/out, `nil` = absent). Order: index 0 → `ErrOutOfRange`; token > `MaxReasoningTokenLen` → `ErrOutOfRange`. Record green output.
- [x] 2.3 REFACTOR — Confirm no edit/import of `reasoning_content.go` internals (NFR-ARE-A). Record refactor note.

## Phase 3: AI-17.3 — Redacted and signature-only streams (R-ARE-012..014)

- [x] 3.1 RED — S-ARE-031..039: redacted signal on block-start only, no `ReasoningState` exposed on any kind, `ErrMisplaced` on a non-empty delta after a redacted start (zero-length delta still allowed), `ErrEmpty` on a redacted end without token, token-only block reconstructs to `ReasoningStateTokenOnly` and is distinguishable from a redacted block sharing the same token bytes. Record red output.
- [x] 3.2 GREEN — Add redacted-start check to `NewReasoningDelta` (`ErrMisplaced` at fragment) and to `NewReasoningBlockEnd` (`ErrEmpty` at token when redacted and no token). Test-local reconstructor feeds `(redacted, text, token)` into `NewReasoning`/`NewRedactedReasoning`. Record green output.
- [x] 3.3 REFACTOR — Extract test-local concatenator/reconstructor, kept unexported (R-ARE-008 — no public accumulator). Record refactor note.

## Phase 4: Verification and closeout

- [x] 4.1 Add totality table test (NFR-ARE-B / S-ARE-041: zero-value payloads, index 0, invalid-UTF-8/over-long fragment, nil/zero-length/over-long token — none panics) and failure-reporting assertions (NFR-ARE-C / S-ARE-042: every rejection is AI-04's `Violation`, `errors.Is` matches a landed sentinel, position names the offending field).
- [x] 4.2 Verify `go.mod` still zero requires, both AI-00 import guards pass, `reasoning_content.go`/`reasoning_content_test.go` are diff-free (NFR-ARE-A / S-ARE-040).
- [x] 4.3 Run `make test` (`go test -race -v ./...`) and `make lint` in `backend/agent/`; record the final green output plus the per-test-list red/green/refactor log in this file (NFR-ARE-D / S-ARE-043).

## Evidence log (apply, resumed session)

Resumed after a disconnect mid-Phase-3. Phases 0–2 landed pre-disconnect in
`91168d0` ("feat(ai): land the AI-17.1/AI-17.2 reasoning block lifecycle and
token events"). This session's work landed in two commits:

| # | SHA | Message | Scope |
|---|---|---|---|
| 1 | `8337573` | `feat(ai): land the AI-17.3 redacted and signature-only reasoning streams (AI-17.3)` | GREEN for S-ARE-035/036 (redacted-gated `ErrMisplaced`/`ErrEmpty` rules) against the already-present RED tests for S-ARE-031..039 |
| 2 | `4ceb77c` | `feat(ai): close out AI-17 NFRs - totality and a clean make lint (AI-17 NFR)` | S-ARE-041 totality test, S-ARE-042 failure-reporting test, one-line pre-existing lint fix |

### RED → GREEN → REFACTOR, per test-list item (this session's scope: 3.1–4.3)

| Item | RED | GREEN | REFACTOR |
|---|---|---|---|
| 3.1 (S-ARE-031..039) | ✅ Already written pre-disconnect; confirmed exactly 2 tests failing at session start (`TestReasoningBlockEnd_RedactedWithNoToken_IsRejected`, `TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected`); the other 9 in the S-ARE-031..039 range already passed against the Phase 1/2 surface | n/a (RED-only step) | n/a |
| 3.2 (redacted-gated rules) | see 3.1 | ✅ `go test -race -run 'TestReasoningBlockEnd_RedactedWithNoToken_IsRejected\|TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected' -v ./...` → both PASS | — |
| 3.3 (extract test-local helpers) | n/a | n/a | ✅ `reconstructFragments`/`reconstructReasoningState` were already unexported test-local helpers (pre-disconnect); confirmed still unexported and still the only path used, no production duplication to extract since the two redacted-gated rules use different sentinels/fields (`ErrMisplaced`/fragment vs `ErrEmpty`/token) — a shared helper would not reduce duplication. `go test -race ./...` full-package safety net: PASS, 0 FAIL |
| 4.1 (S-ARE-041/042) | ✅ Written directly against the landed surface (constructors/accessors already existed; these are closeout assertions, not new-behavior RED) | ✅ `go test -race -run 'TestReasoningEvents_Totality_NoExportedEntryPointPanics\|TestReasoningEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel' -v ./...` → both PASS, all subtests PASS | — |
| 4.2 (NFR-ARE-A) | verification only | ✅ `go.mod` = 2 lines (module + go directive), zero `require`; `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` + `TestLayer1_ModuleHasNoDependencies_ZeroRequires` both PASS; `git diff main -- reasoning_content.go reasoning_content_test.go` → empty (diff-free) | — |
| 4.3 (final gate) | — | — | ✅ `make test` (`go test -race -v ./...`) → exit 0, 0 `--- FAIL` lines, `ok` for `agenttest` and `ai`. `make lint` → exit 0, `0 issues.` (after the one-line blank-line fix below) |

### Deviation found and fixed during 4.3

`make lint` initially reported one `revive` `package-comments` finding on
`reasoning_event.go`: the file's header comment (landed in `91168d0`) was
missing the blank line every sibling file in this package (`reasoning_content.go`,
`tool_call_event.go`, `text_events.go`, `response_start.go`, `completion.go`,
`event.go`) carries between its descriptive header and `package ai`. Without
it, Go's doc-comment association treats the header as the package comment,
and revive flags it for not starting with "Package ai". Pre-existing gap from
Phase 1, not a behavior change; fixed with a single blank line in `4ceb77c`.
`make lint` was 1 issue before the fix, 0 after.

### Final verification (`backend/agent/`, after `4ceb77c`)

- `make test` → `ok  github.com/cachicamas/backend/agent/src/agenttest`, `ok  github.com/cachicamas/backend/agent/src/ai`; 0 FAIL.
- `make lint` → `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.`
- `gofmt -l` on both changed files → empty (clean).
- `git diff main --stat -- backend/agent/src/ai/reasoning_event.go backend/agent/src/ai/reasoning_event_test.go` → 1847 insertions across the full AI-17 feature (Phases 0–4, commits `91168d0`, `8337573`, `4ceb77c`).
