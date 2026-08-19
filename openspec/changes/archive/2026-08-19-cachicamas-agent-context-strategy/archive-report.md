# Archive Report: AG-17 — Add the context strategy seam and token accounting

> **Change**: `cachicamas-agent-context-strategy` · **Milestone**: AG-17 (Layer 2 Wave 4, **17 of 24**)
> **Branch**: `feat/agent-layer2-wave4-ag17` · **Base**: `origin/main@b0de5bf6`
> **Closes**: R-18's **seam 5** (AG-17.1, the turn-boundary check) and **seam 6** (AG-17.2, the optional counting capability), per `agent-v1-scope/spec.md` `S-AGS-023`. Contributes the pre-condition for R-11 (AG-18).
> **Self-verification at apply close**: full suite green under `-race -count=1` (12/12 packages, EXIT:0), `make lint` 0 issues, `make build` clean, `make vuln-check` clean. **`sdd-verify` has not yet run independently** — this report records apply-phase evidence, not a separate verify pass, and does not claim one.
> **Delivery**: single PR, `size:exception` pre-authorized by the user against a 1000-line budget
> **Receipt-driven development**: off for this session (decided by clone_local); not invoked

## What shipped

The harness now has a place to stand and a measurement, and deliberately ships no compaction.

- **The context strategy seam (AG-17.1)** — `context_strategy.go`: `ContextStrategy`, a one-method interface mirroring `FailoverPolicy` symbol for symbol; `ContextPrompt` (exported `Transcript`/`Budget`/`Accounting` fields); `ContextVerdict struct{}` — the zero value is the ONLY constructible value, so a verdict requesting compaction is unconstructible by any implementation a caller could write; `NoOpContextStrategy{}`, the one shipped never-compact default, plus its compile-time guard. `Harness.ContextStrategy`/`ContextBudget` (nil/zero-default) fields, consulted **exactly once per LOGICAL turn** — immediately after `transcript := transcriptFromHistory(hist)` and before the retry-bound block, strictly outside the attempt loop — never per retry attempt.
- **The budget type** — `ContextBudget{limit int64; present bool}`, mirroring `ai.TokenCount`'s absence discipline: the zero value is absent ("Layer 3 stated no budget"), never a budget of zero tokens; `ContextBudgetOf` is total, mapping every negative input to the same absent zero value with no error and no panic.
- **Token accounting (AG-17.2)** — `token_accounting.go`: `TokenSource` (`Unavailable`/`Reported`/`Estimated`, zero value `Unavailable`) and `TokenAccounting` (unexported fields, one two-result accessor `Tokens() (int64, TokenSource)` — mechanically unreadable without its provenance). `resolveTokenAccounting` discovers the capability by type-asserting the shipped `ai.TokenCounter` and by no other means: an advertised counter that errors, and one that answers a nil error with an absent count, both resolve to `Unavailable` — never `Estimated` — because advertising binds (`R-AMP-019`); a clean type-assertion failure resolves to `Estimated` via the byte-based `estimateTokens` (`R-AMP-018`).
- **The estimate** — `⌈B/D⌉ + M×K` over UTF-8 bytes (`D=4`), with a per-message structural constant (`M=4`), walking system-instruction segments, every message's text/tool-call/tool-result/reasoning content, and every declared tool's name+description+schema. Pure — no clock, no environment, no randomness, no I/O — asserted by a direct determinism test, since the ambient-authority guard does not forbid `time`.
- **The pre-hook semantic, stated not implied** — accounting measures the pre-hook request on every path (`applyPreRequestHook` derives a new request downstream of the turn boundary); both the `Reported` provenance's doc comment and `ContextPrompt.Accounting`'s state this explicitly, and a dedicated test records a mutating hook's divergence rather than asserting a false equality.
- **`agenttest.CountingProvider`** — the exported, scriptable counting-capable fixture, embedding `*Provider` by pointer (`Stream` is on the pointer receiver) with two constructors so a `(result, err)` pair can never be half-stated.
- **Four bites, RED-recorded then reverted**: `S-CTX-003` (moving the consultation into the attempt loop → 3 consultations instead of 2), `S-CTX-008` (emitting one `CompactionStarted` from the guarded block → 19 events instead of 17 on the NoOp run), `S-CTX-013` (collapsing the non-conformance branch into the estimate → both rows report `estimated`), `S-CTX-015` (a bare single-result accessor beside `Tokens()` → 2 exported methods instead of 1).
- **No substrate release taken** — `harness.go` is the only pre-existing `src/agent` file this change modifies; both substrate filters widened by exact filename suffix for the two new production files and their two test files only, byte-in-sync, no pre-existing filename added (the AG-15 shape, not AG-16's).

## Commits

| SHA | Subject |
|---|---|
| `c33d8bf8` | `docs(openspec): AG-17 exploration — context strategy seam and token accounting` |
| `9863c4f2` | `docs(openspec): AG-17 proposal — context strategy seam and token accounting` |
| `2ed400f6` | `docs(openspec): AG-17 design — context strategy seam and token accounting` |
| `eef41df2` | `docs(openspec): AG-17 delta specs — six files, one new capability` |
| `85efa23b` | `docs(openspec): AG-17 task breakdown — 70 tasks across 13 phases` |
| `d5ecc37e` | `feat(agent): AG-17.2 phase 1 — context strategy and token accounting types` |
| `4fc28ba6` | `feat(agent): AG-17.2 phase 2 — the byte-based token estimate` |
| `b4a211f1` | `feat(agenttest): AG-17.2 phase 4 — exported counting-capable fixture` |
| `89fe7b12` | `feat(agent): AG-17.2 phase 3 — the accounting resolver` |
| `a7f630de` | `feat(agent): AG-17.1 phase 5 — wire the context strategy seam into Harness.Run` |
| `33edd658` | `test(agent): AG-17.2 phase 6 — externally-observed discovery and pre-hook divergence` |
| `57637aa0` | `test(agent): AG-17 phase 7 — closed-sequence proof and substrate verification` |
| `9287bf24` | `chore(agent): AG-17 phase 8 — widen substrate filters, no release taken` |
| `1f9a61ea` | `docs(openspec): AG-17 phase 9 — five spec-delta back-annotations` |
| `db232e0e` | `docs(0003): AG-17 status line, milestone counter, checklist` |
| `90cbaa66` | `docs(openspec): AG-17 phase 11.1 — promote the agent-context-strategy capability` |
| `d2ee37c4` | `fix(agent): AG-17 phase 12 — reorder closure return to satisfy staticcheck ST1008` (also carries the archive `git mv`) |
| *(this commit)* | `chore(openspec): AG-17 archive — apply-progress, archive report, tasks self-check` |

## Capabilities promoted

| Capability | Kind | What promoted |
|---|---|---|
| `agent-context-strategy` | **NEW** | `R-CTX-001`…`012`, `S-CTX-001`…`021` (incl. 4 bites `S-CTX-003`/`008`/`013`/`015`). 310 lines |
| `agent-run-driver` | back-annotation | Row `:342` closed for the check half (AG-18 keeps compaction); five "still true at AG-17" parentheticals across the Explicit non-requirements table |
| `agent-history` | back-annotation | Row `:260` closed — history needed **no** change; two related parentheticals |
| `agent-v1-scope` | back-annotation | `R-AGS-007`/`S-AGS-023..026` — seams 5 and 6's shipped mechanism recorded. **Enforced by no Go test**, applied deliberately (proposal risk 6) |
| `agent-retry-failover` | back-annotation | `R-RTY-002` HELD, not amended — the seam's turn-boundary placement is what keeps the byte-identical-transcript pin provable; `S-RTY-002` gains an AG-17 note |
| `agent-loop-skeleton` | MODIFIED | Header range extended to `S-LSK-028`; `R-LSK-004`'s "AG-17 requests NO release" paragraph and filter-widening paragraph; `S-LSK-027`/`S-LSK-028` new |

## Verification at close

Re-run directly by this apply phase, not inherited from an unverified claim:

- `go test -race -count=1 ./...` from `backend/agent/` — **12/12 packages `ok`**, EXIT:0, `openaicompat` 173.467s (genuinely uncached — never cited a cached, sub-10s run as evidence throughout this change)
- `golangci-lint cache clean && make lint` (pinned `v2.9.0`, installed via `make tools`, not the machine's global `2.12.2`) — **`0 issues.`** after one real fix (ST1008, ordering the local test closure's return values)
- `gofmt -l` empty on every file this change touches; 15 pre-existing gofmt-dirty files elsewhere in the package confirmed untouched by this change (several are substrate files it is forbidden to edit)
- `govulncheck ./...` — **"No vulnerabilities found."**; `make vuln-check`'s own `-json` run confirms 0 findings with a reachable trace, EXIT:0
- Byte-unchanged vs `origin/main@b0de5bf6`: `doc.go`, `doc_contract_guard_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `compaction_events.go`, `cost_events.go`, `cost_usage.go`, `history.go`, and **all of `backend/agent/src/ai/`** and `go.mod`/`go.sum` — confirmed by empty `git diff`, not by assertion
- The pre-existing non-test-file diff against `origin/main` under `backend/agent/src/agent/` is exactly `{harness.go}` — matches `S-LSK-027` exactly; the two new production files (`context_strategy.go`, `token_accounting.go`) are additions, not releases
- Both substrate filters (`filterOutLoopFiles`, `filterOutLoopHookFiles`) carry an identical entry set after widening; `cost_events_test.go`/`stream_check_test.go` confirmed absent from both; no file under `src/agenttest/` added to either
- `TestTurn_SubstrateUntouched`/`TestTurn_PreRequestHook_SubstrateUntouched` PASS at the final state (they legitimately FAILED between Phase 5 and Phase 8, resolved by the filter widening — recorded in `apply-progress.md`, not hidden)
- Every archived file's `git hash-object` blob hash confirmed identical before and after `git mv` — cryptographic proof against the known truncation-into-placeholder risk, not a visual diff
- All four bites (`S-CTX-003`, `S-CTX-008`, `S-CTX-013`, `S-CTX-015`) RED-recorded with failing output, then reverted; each revert re-confirmed GREEN before moving on
- Base checkout at `/Users/braejan/workspace/witsaba/repositories/cachicamas` never touched; all work confined to the `ag17` worktree throughout

## Findings recorded during apply (not hidden, not silently corrected)

1. **Two genuine test-assertion defects, found and fixed via the test, never the production code**: `TestTokenAccounting_UnreadableWithoutSource`'s field-count check originally used `reflect.Type.NumField()` (counts unexported fields too) where "exposes no **exported** field" needed `f.IsExported()`; the pre-hook doc-comment check used a case-sensitive substring match against a doc comment that states the phrase in uppercase for emphasis. Both are documented in `apply-progress.md` at the phase where they were found.
2. **A staticcheck ST1008 finding** (error not the last return value) in one test's local closure, found by the final `make lint` gate and fixed with no behavior change.
3. **`harness.go`'s design-time line citations have drifted** from their pre-edit values (`:498`→`:512`, `:530`→`:562`, `:91`→`:92`, `:518-529`→`:550`+), fully accounted for by the new import, the two new fields, and the consultation block. Recorded in `apply-progress.md` task 9.6 rather than rewritten across the five citing specs, per this repo's own established convention (AG-11 through AG-16's promoted specs carry the identical class of post-landing drift for the same reason).
4. **A commit-staging defect caught before it could ship silently**: the first archive commit's `git commit` picked up a staged rename but not an unstaged `tasks.md` checkbox edit made afterward, because only one other file had been explicitly `git add`ed. Found by checking `git show HEAD --stat` immediately rather than trusting the commit, corrected in the next commit.

## Carried forward as follow-ups (non-blocking)

1. AG-18 (compaction itself — summarising, transcript surgery, protected recent turns, the compaction event family's first emission) is now startable: it consumes this milestone's seam, budget, and accounting, and its verdict-type extension path is the one `failover_policy.go:57-62`/`R-CTX-003` already names.
2. `S-HIS-080` (`agent-history/spec.md:182`) is a **pre-existing** count-assertion defect this change neither caused nor touches (recorded by this change's own `agent-history` delta, not repaired by it — repairing it belongs to a change that owns that table).
3. The `harness.go` line-citation drift named above (finding 3) is a recurring, structural pattern across every milestone that edits this file; a future change could consider whether design.md citations should be phrased as symbol/content anchors rather than line numbers, to reduce the number of times this same finding recurs.

## State at close

Layer 2 stands at **17 of 24**. AG-18 (compaction) is next.
