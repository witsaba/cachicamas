# Archive report — AG-12: history and the pairing invariant

**Change:** `cachicamas-agent-history` · **Milestone:** AG-12 (Layer 2, wave 3) · **Archived:** 2026-08-16
**Branch:** `feat/agent-layer2-wave3-ag12` (base `main` `0564d162`) · **Mode:** Strict TDD · **Artifact store:** hybrid (Engram + OpenSpec)
**Closes:** R-07's boundary enforcement — *every tool call has a matching result, enforced at the history boundary.*

## Final state

AG-12 is complete and closed. `backend/agent/src/agent/history.go` holds the harness's transcript store:
append-only within a run, validated at the boundary through exactly one commit path, with orphan synthesis for
interruption. Verification returned **PASS WITH WARNINGS — 0 CRITICAL**, 9 of 9 requirements and 25 of 25
scenarios discharged, and recorded no archive blocker.

| Gate | Result |
| --- | --- |
| `go test -race -count=1 ./...` | 12 of 12 packages ok |
| `make lint` (after `golangci-lint cache clean`) | `0 issues.` |
| `make build` | exit 0 |
| `make vuln-check` | exit 0, 0 findings |

Gates were re-run independently by `sdd-verify`, not taken from the apply record.

## What shipped

### AG-12.1 — append and boundary validation

- **`History`**, an opaque struct with unexported `entries` (the ordered transcript) and `open` (unanswered
  calls, in issuance order).
- **One validating commit primitive**, `(*History).commit`, the sole writer of both fields. Every public route
  funnels through it: `Append`, `NewSeededHistory`, `CloseTurn`, `SynthesizeOrphans`. This is the C1 lesson
  applied to history — no privileged bypass for internal callers.
- **Entry envelope** carrying `EntryID`, the Layer 1 `ai.Message`, and an `EntryOrigin` discriminator. Entry
  identity is the 1-based ordinal at commit time: deterministic, stable across processes, and impossible for a
  caller to mint.
- **Read-only views** — `Entries()` returns a fresh copy; read-back yields unmodified Layer 1 values.
- **Typed rejections through Layer 1's existing rule classes**, with `ai/validation.go` untouched:
  `ai.ErrUnresolvedReference` for a result naming a call the transcript never declared, and `ai.ErrEmpty`
  positioned at the result slot for a call still open at turn close.
- **Seeded construction** takes an ordered `[]ai.Message` and nothing else, reports the position of the first
  offending entry, and re-runs the append-time rules. A seed ending in open calls is **accepted** — only an
  orphaned result rejects at seed time. Rejecting it would make an interrupted transcript unseedable on resume
  and render AG-12.2 unreachable through the exact path it exists to serve.

### AG-12.2 — orphan synthesis

- `SynthesizeOrphans() (int, error)` closes every open call with an `ai.RoleTool` message built through
  `ai.NewToolFailure`, committed with `EntryOriginSynthesized` via the same primitive.
- The discriminator is **the envelope's origin, not `ToolResult.Failed()`** — a genuinely failing tool also sets
  `Failed()`, so it cannot distinguish an interruption artifact from a real result.
- **Idempotent and total:** N orphans close on the first application returning `(N, nil)`; the second returns
  `(0, nil)` and changes nothing.

### Contract and spec closures

- **`L2C-07`** added to `doc.go` with its `expectedLayer2ContractRows` entry in the same commit, per
  `R-AGP-002`'s closed-amendment rule.
- **New capability** `openspec/specs/agent-history/spec.md` — `R-HIS-001`..`R-HIS-009`, 22 scenarios plus 3
  RED-recordable bites.
- **`agent-event-envelope` delta** — `S-AEV-122` asserted the guard table "contains 6 rows"; `L2C-07` would have
  made that silently false with no test failing. `R-AEV-014` is re-scoped to own `L2C-06`'s presence, position
  and text rather than the table's size, which fixes the drift class instead of only this instance.
- **`agent-package-scaffold` delta** — back-annotates the stale "as of AG-04: four rows" parentheticals.

## Carry-forward warnings

None blocks archive; all three are residuals in the proof machinery rather than live defects.

**W1 — the closed-route guard's surface scope.** `R-HIS-004`'s enumeration closes over the `*History` and
`Entry` method sets (reflection) and over exported package-level constructors (`go/parser`). It does **not**
catch a mutating method hung off a third exported package type holding a `*History` pointer; the verifier built
one and both guards passed. No such type ships today, so this is a future-closure gap, not a live bypass — but
AG-13 leans on this guard, so it should close before the surface widens.

**W2 — the zero value's read routes.** `S-HIS-052` says the zero value must never behave as an empty valid
transcript. Every *mutating* route on `new(History)` rejects correctly with `ai.ErrEmpty` at `history`, so the
pairing invariant is not violable through it. But `new(History).Len()` answers `0` rather than refusing, and
`history_test.go` pins that as the wanted outcome. The spec wording is stronger than the code, and the test
certifies the weaker behavior — the requirement and its test agree with each other rather than with the intent.

**W3 — `S-HIS-042` is proven partly by declaration.** The guard asserts the authored `class` field rather than
derived read-onlyness, and `reflect.TypeOf(agent.Entry{})` misses pointer-receiver methods.

## Scope held

`git diff origin/main` is empty for `backend/agent/src/ai/`, `loop.go`, `scheduler.go`, `go.mod` and `go.sum`.
AG-12 does not wire into the loop — that edge belongs to AG-13. Exactly four new files landed under
`backend/agent/src/agent/`, and all four filename suffixes are registered in both `filterOutLoopFiles` and
`filterOutLoopHookFiles`, byte-in-sync, because `S-LSK-006` diffs the whole `src/agent/` directory.

## Size

20 files, 3020 insertions, 7 deletions — **3027 changed lines** against a 1000-line review budget, under a
maintainer-pre-authorized `size:exception`. Breakdown: 1557 lines of Go (`history.go` 432, `history_test.go`
554, `history_surface_guard_test.go` 331, `history_synthesis_test.go` 240), 836 lines of SDD change-folder
markdown, 221 lines of promoted spec copies, 108 lines of apply record and doc-status bump, 47 lines of
substrate and `doc.go` touches. The runtime attempt ledger recorded attempt 1 as `passed` at 3027 lines;
`apply-progress.md`'s 2923 figure is stale, predating its own commit.

## Unblocks

Per the AG-12 charter, this milestone **blocks AG-13** (the multi-turn run driver, which consumes the frozen
seeded-construction signature and the read-only view shape) and **AG-17** (context strategy). AG-18.2 separately
re-proves the invariant post-compaction; ordinal-derived entry identity does not survive compaction, which is
recorded as an inherited constraint AG-18.2 owns rather than something solved here.

## Engram traceability

| Artifact | Topic | Id |
| --- | --- | --- |
| Exploration | `sdd/cachicamas-agent-history/explore` | 3112 |
| Proposal | `sdd/cachicamas-agent-history/proposal` | 3113 |
| Spec | `sdd/cachicamas-agent-history/spec` | 3115 |
| Design | `sdd/cachicamas-agent-history/design` | 3116 |
| Tasks | `sdd/cachicamas-agent-history/tasks` | 3123 |
| Apply progress | `sdd/cachicamas-agent-history/apply-progress` | 3150 |
| Verify report | `sdd/cachicamas-agent-history/verify-report` | 3155 |
| Archive report | `sdd/cachicamas-agent-history/archive-report` | 3157 |

All 44 tasks complete. The cycle is closed.
