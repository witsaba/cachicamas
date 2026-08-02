# Apply Progress — `cachicamas-ai-tool-call-events` (AI-18)

> Hybrid artifact. On-disk twin of Engram `sdd/cachicamas-ai-tool-call-events/apply-progress`. First and only apply run for AI-18 — no prior progress existed.
> Worktree: `cachicamas-worktrees/ai-wave-2`, branch `feat/2026-08-01-cachicamas-ai-layer1-wave-2`. Mode: **Strict TDD**.

## Status: DONE — 25/25 tasks complete, all 4 phases + Phase 0 precondition

## Summary

Implemented the three streamed tool-call event kinds (`ToolCallStart`, `ToolCallDelta`, `ToolCallEnd`) on AI-14's event envelope, in `backend/agent/src/ai/tool_call_event.go` (291 lines) with `tool_call_event_test.go` (1341 lines, 38 scenarios S-ATC-001…038, `package ai_test`). Registered three new `EventKind` constants and registry entries in `event.go` (a shared, concurrently-edited file — see Concurrency Handling below). All phases followed RED → GREEN → REFACTOR. Final state: `make test` green (0 failures across 1069 sub-tests, both `src/ai` and `src/agenttest` packages), `make lint` clean for both of this milestone's own files (one unrelated, out-of-scope `revive` issue remains in sibling AI-17's `reasoning_event.go`).

## Design Deviation Found and Corrected During Apply

`design.md`'s "Reconciled construction model" (dated 2026-08-01, before this apply run) asserted that AI-14's landed design fixes a **bare, never-erroring `Event` constructor** (`NewXxxEvent(...) Event`), with validation deferred entirely to `CheckEmit`. This was based on reading AI-14's *design.md prose*, not AI-14/AI-15's *actual landed code* — and design.md's own Open Questions section explicitly flagged this as unverified: "AI-15's landed constructors were not independently re-verified in this pass... if AI-15's actual landed code diverges from this same bare-Event/CheckEmit pattern once it exists on disk, re-reconcile before sdd-apply."

Reading `response_start.go` and `completion.go` (AI-15's actual shipped code) at apply time showed neither takes the bare-`Event` shape: both `NewResponseStart` and `NewCompletion` validate **eagerly, inside the constructor**, and return `(Event, error)` — exactly matching `tool_call.go`'s `NewToolCall`. The spec's own scenario text (S-ATC-005, -006, -007, -011, -019) is written the same way: "when construction runs, then it fails" / "then it succeeds" — language that only makes sense if the constructor itself can fail.

**Resolution**: implemented `NewToolCallStart/Delta/End` as `(Event, error)`, validating via each payload's `validate(at Path) *Violation`, called once eagerly at construction (`at = nil`) and a second time inside `CheckEmit` (`at = Path{At("event")}`) at the true stream emission boundary — the same "two entry points, one rule set" posture `content_part.go`'s `partPayload` documents. All three payloads additionally implement the unexported `blockPayload` interface (`blockIndex() BlockIndex`) so `CheckEmit`'s generic rule 3 and `CheckStream`'s block-ordering checker work against them with zero checker-file edits, which preserves the one part of design.md's reconciliation that *was* correct (block-scoping via the shared interface). `tasks.md` was updated in place to record this re-reconciliation rather than silently deviating.

## Concurrency Handling (AI-16/AI-17 sibling agents, same worktree)

`event.go` and `event_registry_test.go` are shared, append-only registry files that AI-16 (text-events) and AI-17 (reasoning-events) were editing concurrently in this exact worktree, as flagged in advance.

- The Edit tool's stale-snapshot guard caught two genuine concurrent changes mid-session (once when AI-17 landed reasoning entries between my read and edit, once more when AI-16 landed text entries) — both times resolved by re-reading fresh and re-applying my own additive edit after the sibling's.
- Sibling AI-16 committed first (`4f63977`, "feat(ai): land the AI-16.1 text block lifecycle"), and — as that commit's own message explicitly acknowledges — incidentally captured my (AI-18's) and AI-17's already-present, non-conflicting registry entries in the same commit, since all three milestones append to one shared file in one worktree. Verified via `git show 4f63977 -- backend/agent/src/ai/event.go` that all nine kinds (3 text + 3 reasoning + 3 tool-call) are present and correct.
- Given this, my own commit (`20a483d`) is scoped to exactly my two exclusive files (`tool_call_event.go`, `tool_call_event_test.go`) via explicit `git add <path>` (never `-A`/`.`), verified via `git diff --stat` on the shared files (empty — already clean/committed) before staging.
- One transient build failure was observed and re-checked rather than treated as a defect: `reasoning_event_test.go` (AI-17) referenced constructors not yet defined in their in-progress `reasoning_event.go`; resolved itself on retry once the sibling reached their own GREEN state. Same for a `text_events_test.go` (AI-16) unused-import error. Neither required or received any edit from me.

## TDD Cycle Evidence

| Task(s) | Scenarios | RED | GREEN | REFACTOR |
|---|---|---|---|---|
| 1.1–1.8 | S-ATC-001…022 | ✅ `go vet` failed: `undefined: ai.EventKindToolCallStart` (one consolidated capture for all 8 sub-tasks, sharing one non-existent production file) | ✅ `go test -run 'ToolCall' -race -v` → `ok`, after fixing one self-inflicted test bug | ✅ Dedup of the 3× repeated block-index rule evaluated and declined (matches sibling AI-16's own choice in `text_events.go`) |
| 2.1–2.2 | S-ATC-023…028 | ✅ Written against Phase 1's already-landed code (test-only leaf, per design) | ✅ Passed on first real run | ✅ None needed in production |
| 3.1–3.2 | S-ATC-029…034 | ✅ Written with new test-local `partitionByBlock`/`reconstruct`/`callOrdinals` helpers | ✅ `-race` clean, 0 races | ✅ Extracted `deltaFragmentsOf`, removing duplication with Phase 2's inline concat |
| 4.1 | S-ATC-035…037 | ✅ Totality table + consolidated rejection sweep written; S-ATC-035 satisfied by existing unchanged guards | ✅ Full `make test` green | N/A |

## Files Changed

| File | Action | Lines |
|---|---|---|
| `backend/agent/src/ai/tool_call_event.go` | Created | 291 |
| `backend/agent/src/ai/tool_call_event_test.go` | Created | 1341 |
| `backend/agent/src/ai/event.go` | Modified (3 registry entries) | landed in sibling commit `4f63977`, not this milestone's own commit |
| `backend/agent/src/ai/event_registry_test.go` | Modified (3 witness entries) | landed in sibling commit `4f63977`, not this milestone's own commit |
| `openspec/changes/cachicamas-ai-tool-call-events/tasks.md` | Updated | all 25 tasks marked `[x]`, evidence appended |
| `openspec/changes/cachicamas-ai-tool-call-events/apply-progress.md` | Created | this file |

## Commits

- `20a483d` — `feat(ai): land the AI-18 streamed tool-call event family (AI-18)` — the two exclusive files only.

## Final Verification

```
$ go test ./src/ai/... -run 'ToolCall' -race -v
ok  	github.com/cachicamas/backend/agent/src/ai	1.560s

$ make test
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.613s
ok  	github.com/cachicamas/backend/agent/src/ai	2.864s

$ make lint
(golangci-lint: 0 issues referencing tool_call_event.go or tool_call_event_test.go;
 1 pre-existing revive issue in sibling AI-17's reasoning_event.go, out of scope)
```

## Remaining Tasks

None — 25/25 complete.

## Next Recommended

`sdd-verify`.
