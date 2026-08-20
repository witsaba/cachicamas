# Tasks: AG-20 — Complete the hook taxonomy

> Delivery: **single-pr** (`exception-ok`, `chain_strategy: size-exception`) — SDD cycle + doc 0003 update, same PR. OpenSpec archive is `sdd-archive`'s job, not this apply step's, per items 3/4/5 below.
> Review budget: **1000 changed lines, EXCLUDING `openspec/`** (working folder). Extension pre-accepted (proposal header).
> TDD: strict, RED-first. Runner: `cd backend/agent && go test -race -count=1 ./...`, **`-count=1` mandatory** (~170s uncached).
> **Deviation from the 530-word tasks budget**: a 49-scenario, 10-delta change with 5 mandated bites and 5 orchestrator-flagged open items requires a full traceability matrix and explicit RED-staging per scenario; this overrides the generic size cap, as design.md's own 800-word deviation already established for this change.

## Review Workload Forecast

| Component | Estimate (lines, non-`openspec/`) |
|---|---|
| `hooks.go` — types, payloads, lane, snapshot, report | 260–380 |
| `loop.go` — field + chain composition + attribution | 40–70 |
| `harness.go` — field, latch, refusal, capture, 4 fire sites, lane lifecycle | 90–150 |
| `compaction.go` — splice, misplaced-options, parameter | 40–70 |
| `hooks_test.go` / `hooks_harness_test.go` / `hooks_compaction_test.go` — 49 scenarios + 5 bites | 830–1020 |
| `loop_test.go` / `loop_hook_test.go` — filter entries only | 10–20 |
| Doc 0003 status line, delivery table, checklist | 25–45 |
| **Counted total** | **1295–1755** |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | `hooks.go` types + pre-request chain | PR 1 (single) | `go test -race -count=1 -run 'TestHooks_Surface\|TestHooks_Chain' ./src/agent/...` | N/A — pure composition, no gate | Delete `hooks.go`/`hooks_test.go`; revert `loop.go`/`harness.go`(field only)/`compaction.go`(rejection only); drop 2 filter entries |
| U2 | Session-start + post-turn | PR 1 (same) | `go test -race -count=1 -run 'TestSessionStart\|TestPostTurn' ./src/agent/...` | N/A — deterministic snapshot-or-count lemma | Delete `hooks_harness_test.go`; revert latch + 4 fire sites; drop 1 filter entry |
| U3 | Pre-compact + AG-20.2 asynchrony | PR 1 (same) | `go test -race -count=1 -run 'TestPreCompact\|TestAsynchrony\|TestPanic' ./src/agent/...` | `agenttest.Gate`, mandatory (S-HKS-017/018/019/021) | Delete `hooks_compaction_test.go`; revert splice + lane snapshot; drop 1 filter entry |

## Coverage table — every scenario, exactly one task

| Scenario | Task | Scenario | Task | Scenario | Task |
|---|---|---|---|---|---|
| S-HKS-001 | 1.3 | S-HKS-019 | 3.13 | S-CMP-039 | 3.6 |
| S-HKS-002 | 1.3 | S-HKS-020 | 3.15 | S-CMP-040 | 3.3 |
| S-HKS-003 | 1.3 (+1.2, +4.6) | S-HKS-021 | 3.16 | S-CMP-041 | 3.4 |
| S-HKS-004 | 1.5 | S-HKS-022 | 1.7 | S-CMP-042 | 3.7 |
| S-HKS-005 | 1.5 | S-HKS-023 | 1.7 | S-LSK-032 | 5.3 (see 1.4/2.7/3.8) |
| S-HKS-006 | 3.1 | S-HKS-024 | 4.3 | S-AGE-030 | 5.2 |
| S-HKS-007 | 3.3 | S-HKS-025 | 4.4 | S-RUN-113 | 2.8 |
| S-HKS-008 | 3.4 | S-HKS-026 | 4.5 | S-ATT-015 | 2.9 |
| S-HKS-009 | 3.6 | S-HKS-050 (bite a) | 3.14 | S-DEL-026 | 2.10 |
| S-HKS-010 | 2.5 | S-HKS-051 (bite b) | 3.12 | S-AGS-065 | 5.4 |
| S-HKS-011 | 2.5 | S-HKS-052 (bite c) | 4.2 | S-AGS-066 | 5.4 |
| S-HKS-012 | 2.5 | S-HKS-053 (bite d) | 3.5 | S-AIV-032 | 5.5 |
| S-HKS-013 | 2.1 | S-HKS-054 (bite e) | 2.4 | | |
| S-HKS-014 | 2.1 | S-PRH-008 | 1.5 | | |
| S-HKS-015 | 2.1 | S-PRH-009 | 1.7 | | |
| S-HKS-016 | 4.1 | S-PRH-010 | 1.5 | | |
| S-HKS-017 | 3.9 | S-PRH-011 | 1.7 | | |
| S-HKS-018 | 3.11 | S-AEV-126 | 5.1 | | |
| | | S-CMP-038 | 3.3 | | |

49 scenarios (31 `S-HKS-*` incl. 5 bites, 18 across the 9 amendment deltas), each discharged once, none doubled.

## Phase 1 — U1: hooks.go core types, transport, pre-request chain

- [x] 1.1 **[predicate check]** Open `filterOutLoopFiles` (`loop_test.go:837`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`); confirm the predicate accepts exact-filename-suffix entries for `hooks.go`, `hooks_test.go`, `hooks_harness_test.go`, `hooks_compaction_test.go`; record the exact matching rule before any entry is written. (Item 1)
- [x] 1.2 **[search]** Search `backend/agent/src/agent/` for an existing compile-time-refusal fixture pattern reusable for S-HKS-003's "observing family cannot signal" assertion. If none exists, decide here — build one, or restate as a type-shape reflection assertion — and record the decision with evidence. (Item 2)
- [x] 1.3 **[RED]** `hooks_test.go` (`package agent_test`): one-surface enumeration; transport refusal (each of the 5 `Hooks` members individually); observing-family unconstructibility via 1.2's pattern. Compile-fails against nonexistent `Hooks` (accepted new-surface RED). → S-HKS-001,002,003
- [x] 1.4 **[GREEN, production]** `hooks.go`: `Hooks`, 4 hook/reporter types, 3 payloads + `ObserverStall`/`HookPoint`/`StallReason`, `isZero()`. `loop.go`: `TurnOptions.Hooks` field. `harness.go`: `Harness.Hooks` field, transport assignment beside `Continuation`, typed refusal at `Run` entry. `compaction.go`: `Hooks` joins misplaced-options rejection. Add `/hooks.go` + `/hooks_test.go` to BOTH filters, byte-in-sync, per 1.1. Makes 1.3 pass.
- [x] 1.5 **[RED]** Extend `hooks_test.go`: chain composition order (singular-first, B/C/D elements, final-output-only), AG-08 unedited-suite compatibility. → S-HKS-004,005; S-PRH-008,010
- [x] 1.6 **[GREEN, production]** Widen `loop.go`'s `applyPreRequestHook` into chain composition. Makes 1.5 pass; re-run AG-08's suite, confirm byte-unchanged diff + green.
- [x] 1.7 **[RED→GREEN]** Extend `hooks_test.go`: source-name attribution, abort-before-I/O, no-renumbering-under-insertion. → S-HKS-022,023; S-PRH-009,011

## Phase 2 — U2: session-start latch + post-turn firing sites

- [x] 2.1 **[RED]** `hooks_harness_test.go` (`package agent_test`): session-start — two serial `Run` calls, delegated child on distinct `Harness`, shutdown value. → S-HKS-013,014,015
- [x] 2.2 **[GREEN, production]** `harness.go`: `sessionStarted bool` latch beside `shutdown` (`:126-131`), guarded by `signalMu`; set after the shutdown check; enqueue placed before `NewRunStart` construction (`:480`), not merely before the send (`:484`).
- [x] 2.3 **[GREEN, production]** `harness.go`: observer lane creation (`len(PostTurn)+len(SessionStart)>0`), mutex-guarded FIFO, one drain goroutine, lock-append enqueue, registration-order dispatch, `defer recover()` per invocation → `StallPanicked`. Makes 2.1 pass together with 2.2.
- [x] 2.4 **[RED bite]** `S-HKS-054` (e): scratch-enqueue session-start per `Run`; re-run S-HKS-013; FAILS (count==2 or non-empty second-run stall report). Revert. RED before GREEN, `-count=1`.
- [x] 2.5 **[RED]** Extend `hooks_harness_test.go`: post-turn fixture over AD-7's 14 exits (6 yes-rows scripted, 8 no-rows scripted); multi-attempt cost-sum fixture (3 attempts, distinct non-zero figures). Wire only the success site first — rows 3-6 RED. → S-HKS-010,011,012
- [x] 2.6 **[GREEN, production]** `harness.go`: `turnCost costAccumulator` (run-frame local, never a `Harness` field) + `capturedOutcome` beside `capturedTurnID`/`total.add` (`:616-638`); four enqueue sites — (i) pre-`closeTurnMarked`, (ii) pre-`failRun`(`:723`), (iii) pre-`windDownRun`(`:667`), (iv) pre-`windDownRun`(`:693`). Makes 2.5 pass.
- [x] 2.7 **[test-only, filter]** Add `/hooks_harness_test.go` to both filters, byte-in-sync.
- [x] 2.8 **[RED→GREEN]** Extend `hooks_harness_test.go`: `R-RUN-013` absences — no cost-accumulator `Harness` field (source read), no timer/deadline/sleep/poll/join, no-deadline pin file-unchanged, gate-held observer returns `Run` before release, zero-value `Hooks` goroutine baseline across 6 arms. → S-RUN-113
- [x] 2.9 **[RED→GREEN, same file]** `R-ATT-010`: reported outcome equals streamed outcome value-to-value across finished/failed/interrupted/wound-down; outcome vocabulary unchanged; `turn_events.go`/`failure.go` byte-unchanged. → S-ATT-015
- [x] 2.10 **[RED→GREEN, same file]** `S-DEL-026`: reuse AG-19's delegating-tool fixture; register a post-turn observer on the CHILD harness looking up a publishing seam from its own context; assert no seam found, no attributable event on either stream, parent stream byte-identical to hookless.

## Phase 3 — U3: pre-compact splice + AG-20.2 asynchrony/snapshot

- [x] 3.1 **[RED]** `hooks_compaction_test.go` (`package agent_test`): both-doors fixture; `PreCompact` element records received plan; assert pre-provider ordering, zero `PreRequest` firings. → S-HKS-006
- [x] 3.2 **[GREEN, production]** `compaction.go`: splice `PreCompact` chain between `markSpan`(`:285-289`) and the prefix build (`:291`), `len(chain)>0` guard; hook receives resolved cut + derived span; failure → `emitCompactionFailedArm(..., TurnOutcomeAborted, err)`; `runCompaction` gains one unexported chain parameter. Makes 3.1 pass.
- [x] 3.3 **[RED→GREEN]** Unconditional post-hook re-resolution via `resolveCut`; identical-plan byte-identity; internal fixed-point table under `NFR-CMP-001`'s carve-out. → S-HKS-007; S-CMP-038,040
- [x] 3.4 **[RED→GREEN, same file]** Forward-adjustment: hook re-designates cut forward past an open pair; committed prefix pairing-closed; re-resolution-to-zero fails via the new arm call site. → S-HKS-008; S-CMP-041
- [x] 3.5 **[RED bite]** `S-HKS-053` (d): scratch-skip the post-hook re-resolution; re-run 3.4; FAILS with a split call/result pair. Revert. RED before GREEN.
- [x] 3.6 **[RED→GREEN, same file]** Chain-element-failure, both doors: zero provider requests, aborted outcome, 1 compaction-failed/0 compaction-finished, transcript byte-identical, run continues, `CheckStream` accepts. → S-HKS-009; S-CMP-039
- [x] 3.7 **[test-only, same file]** Misplaced-options total rejection for `Hooks` (member-by-member); chain-arrives-only-as-parameter confirmation. → S-CMP-042
- [x] 3.8 **[test-only, filter]** Add `/hooks_compaction_test.go` to both filters, byte-in-sync.
- [x] 3.9 **[RED]** Extend `hooks_test.go`: gated `PostTurn` observer, sink buffered to script's full event count, byte-identical stream, `Run` returns with gate held, cleanup-release backstop. → S-HKS-017
- [x] 3.10 **[GREEN, production]** `harness.go`: `defer lane.reportOutstanding()` immediately after lane creation (LIFO-first); snapshot under the lane mutex after every arm's `run_end` send and before `queue.close`/cancel-clear/`close(sink)`; three-valued `ObserverStall` reasons; reporter serialized by one dedicated mutex; reporter panics discarded. Makes 3.9 pass.
- [x] 3.11 **[RED→GREEN, same file]** Non-blocking observer captures its own stack (`runtime.Stack`); after `Run` returns, assert neither the harness run frame nor the forwarder frame appear. → S-HKS-018
- [x] 3.12 **[RED bite]** `S-HKS-051` (b): scratch-dispatch observing hooks synchronously; re-run 3.11; FAILS as an assertion (harness run frame present), not a hang. Revert.
- [x] 3.13 **[RED→GREEN, same file]** Queued-victim-beside-culprit: gate index-0, unreached index-1, ≥2 logical turns; reporter receives both `outstanding`(0) and `queued`(1), distinguishable by inspection, delivered before `Run` returns. → S-HKS-019
- [x] 3.14 **[RED bite]** `S-HKS-050` (a), anti-vacuity: observer returns immediately; assert empty report set. Scratch-report-every-observer; re-run; FAILS non-empty. RED BEFORE S-HKS-019 GREEN.
- [x] 3.15 **[RED→GREEN, same file]** Panic: 3 observers, index 1 panics, 2 logical turns; `panicked` report for index 1; indices 0/2 record full counts; process survives; stream byte-identical; report from lane goroutine. Record recovery-removed RED from an **isolated** `go test -run` (S-DEL-022 precedent — non-zero exit + panic trace, not `--- FAIL`). → S-HKS-020
- [x] 3.16 **[RED→GREEN, same file]** Nil-reporter-reports-nothing + stalling-reporter-stalls-both-observables (Run's return AND sink close), non-blocking reads, unconditional cleanup release. → S-HKS-021

## Phase 4 — Cross-cutting: ordering pin, scope fence, closed-sequence, inertness

- [x] 4.1 **[RED]** Extend `hooks_test.go`: 4 hooks at EACH of the 4 points, per-point recorder; `[0,1,2,3]` at each, session-start precedes post-turn, repeatable, `-race`-green. → S-HKS-016
- [x] 4.2 **[RED bite]** `S-HKS-052` (c): scratch-reverse dispatch at one point; re-run 4.1; FAILS `[3,2,1,0]`. Revert.
- [x] 4.3 **[RED→GREEN, same file]** Scope-fence: `git diff` over `backend/agent/` vs merge-base — every `R-HKS-010`-named file byte-unchanged, `src/ai/`+`agenttest` diffs empty, `go.mod`/`go.sum` empty, event-kind guard at committed count, no outcome/cost-label member added, `Turn`/`Run`/`Harness` surfaces unchanged. Reuse `scope_fence_test.go`'s `gitTopLevel`/`gitDiff`/`gitOutput` helpers (AG-19 precedent) — do not edit that file. Anti-vacuity floor: fail loudly on an empty diff (`S-TLS-020` precedent). → S-HKS-024
- [x] 4.4 **[test-only, same file]** Closed-sequence table check (`R-HKS-011`): each "holds" row's owning test byte-unchanged; both AMENDED rows resolve to the `agent-pre-request-hook` delta; the CLARIFIED row resolves to the `agent-compaction` delta. → S-HKS-025
- [x] 4.5 **[RED→GREEN, same file]** Inertness: 6 paired runs (success, turn failure, retry exhaustion, interrupt, shutdown, compacting-both-doors), merge-base vs zero-value-`Hooks` — byte-identical streams/history/requests; `runtime.NumGoroutine()` returns to baseline after every pair. → S-HKS-026
- [x] 4.6 **[test-only, same file]** Direct-`Turn`-caller inertness half of S-HKS-003 (observers on directly-constructed `TurnOptions.Hooks` fire nothing, byte-identical to zero-value `Hooks`) — close any gap left by 1.3.

## Phase 5 — Remaining delta scenarios (cross-reference, minimal new fixture)

- [x] 5.1 **[test-only]** `agent-event-envelope`: one assertion in `hooks_test.go` tying "no `EventKind` registered" (4.3) to "every stalled-observer path terminates in the lane's drain activity" (3.9–3.16) — the two facts `S-AEV-126` requires read together. → S-AEV-126
- [x] 5.2 **[test-only]** `agent-event-delivery`: reuse 3.13's gated-hook fixture plus 2.10's child-run seam-lookup fixture in one cross-check. → S-AGE-030
- [x] 5.3 **[verification]** `agent-loop-skeleton` full close-out (Item 1's own verification step): with all four filter entries landed (1.4, 2.7, 3.8, plus the fourth confirmed), assert the diff shows exactly `{loop.go, harness.go, compaction.go}` as changed non-test files under `src/agent`; every `R-LSK-004`-forbidden file byte-unchanged incl. `failure.go`/`doc.go`/`doc_contract_guard_test.go`; both filters carry an identical 4-entry addition, no wildcard/prefix/directory pattern; an unnamed new file fails both guards (regression probe). → S-LSK-032
- [x] 5.4 **[test-only]** `agent-v1-scope`: exported-surface deferral check (no concrete hook, no wall clock, zero-value inert) + doc-0003 R-17 traceability-spine cross-check. → S-AGS-065,066
- [x] 5.5 **[test-only]** `ai-contract-vocabulary`: enumerate `V-OUT-13`'s 4 points against shipped requirements; assert `src/ai/` diff empty; reuse 1.2/1.3's compile-time-refusal pattern. → S-AIV-032

## Phase 6 — Evidence and gates

- [x] 6.1 `gofmt -l` on every file authored/modified (`hooks*.go`, `loop.go`, `harness.go`, `compaction.go`, `loop_test.go`, `loop_hook_test.go`) — clean. **NEVER `make fmt`** (rewrites committed files, fails substrate guards).
- [x] 6.2 `go vet ./...` from `backend/agent` — clean.
- [x] 6.3 `golangci-lint cache clean && golangci-lint run ./...` (or `make lint`) — 0 issues.
- [x] 6.4 `cd backend/agent && go test -race -count=1 ./...` — record wall-clock (~170s), zero `--- FAIL`, no `(cached)` markers.
- [x] 6.5 `make build` clean; `make vuln-check` clean (not part of `make all` — do not run `make all`).
- [x] 6.6 Confirm isolated-process evidence for 3.15's panic bite (and any other process-crashing bite) is recorded outside the full-suite run.

## Phase 7 — Docs (same PR) and archive handoff

- [x] 7.1 Update `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md`: tick AG-20's row(s), append narrative sentence, bump counter to 20/24, mark Wave 5 complete.
- [ ] 7.2 **[ARCHIVE OBLIGATION — sdd-archive]** Promote `agent-hook-taxonomy` + merge 10 deltas. When promoting `agent-pre-request-hook`, replace the header's "6 charter → 7 spec + 2 bites = 9 total" (`spec.md:7`, `:11-15`, `:147`) with the allocated ID range + named bites — the `S-LSK-020` treatment. (Item 3)
- [ ] 7.3 **[ARCHIVE NOTE — sdd-archive]** `S-AGS-064` is permanently unreused (AG-19 allocated it, its promotion dropped it); this change's `agent-v1-scope` delta starts at `S-AGS-065`. Do not reuse 064. (Item 4)
- [ ] 7.4 **[VERIFY OBLIGATION — sdd-verify]** Judge on evidence the 6 ADDED-not-MODIFIED requirements (`R-CMP-015`, `R-LSK-008`, `R-RUN-013`, `R-ATT-010`, `R-DEL-011`, `R-AGS-016`, `R-AIV-014`) — confirm each targeted original requirement is genuinely not falsified (only needing back-annotation) by opening the cited text, rather than accepting the deferral claim. (Item 5)
