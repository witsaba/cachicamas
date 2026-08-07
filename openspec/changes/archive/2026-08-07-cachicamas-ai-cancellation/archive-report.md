# Archive Report — `cachicamas-ai-cancellation`

> **Status**: **CLOSED** · **Verdict**: **PASS** · **Milestone AI-33: GREEN, ready for PR review**
> **Date**: 2026-08-07
> **Archived to**: `openspec/changes/archive/2026-08-07-cachicamas-ai-cancellation/`
> **Artifact store**: hybrid (filesystem + Engram `sdd/cachicamas-ai-cancellation/archive-report`)

## Outcome

AI-33 proved the Layer 1 cancellation contract over a **real HTTP transport** at all four cancellation moments plus the resource-discipline clause. 6/6 requirements, 22/22 scenarios, 0 FAIL, 0 lint issues, 0 CRITICAL. The entire milestone cost **one line of production logic**. Ready for the PR phase.

**One gap blocks nothing here but needs an orchestrator decision**: the source-of-truth spec was **not** synced — see [Spec sync — NOT performed](#spec-sync--not-performed-orchestrator-decision-required). This is a deliberate hold, not an omission.

## Identity

| Field | Value |
|---|---|
| Change | `cachicamas-ai-cancellation` |
| Milestone | AI-33 — Prove cancellation and goroutine cleanup (doc 0002 § 1987–2037) |
| Wave | 5 — Harden |
| Branch | `feat/ai-33-cancellation` |
| Worktree | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-33` |
| Base | `main` @ `e9a8054` |
| Module | `backend/agent` — **layered, NOT hexagonal** (ADR 0005 § D1) |
| Test runner | `cd backend/agent && make test` (`go test -race -v ./...`) |
| Capability amended | `ai-stream-lifecycle` |

## Commits

Six commits, in landing order. Subnode order (33.1 → 33.4 → 33.2 → 33.3 → 33.5) follows the proposal, not numeric order.

| # | SHA | Subject | Subnode | Requirement |
|---|---|---|---|---|
| 1 | `07e4d0c` | `test(openaicompat): add AI-33.1 real-HTTP cancel-before-headers proof` | AI-33.1 | R-AIS-034 |
| 2 | `e6eb3a1` | `test(openaicompat): add AI-33.4 real-HTTP cancel-after-completion proof` | AI-33.4 | R-AIS-037 |
| 3 | `4ff662f` | `test(openaicompat): add AI-33.2 real-HTTP cancel-between-frames proof` | AI-33.2 | R-AIS-035 |
| 4 | `665fa3e` | `test(openaicompat): add AI-33.3 real-HTTP truly-abandoned-then-cancelled proof` | AI-33.3 | R-AIS-036 (+ `R-CNF-012` verbatim) |
| 5 | `99fef5a` | `fix(openaicompat): add drain-before-close to run() defer chain (R-AIS-033)` | AI-33.5a | **R-AIS-033 — the only production change** |
| 6 | `83eef31` | `test(openaicompat): add AI-33.5 full-package leak check over every exit path (R-AIS-038)` | AI-33.5b | R-AIS-038 |

**Diff vs base**: 7 files, 1,918 insertions, 1 deletion.

## Production scope — confirmed

Exactly **one** production file changed across the whole milestone.

| File | Diff | Production logic | Header doc | Refactored defer |
|---|---|---|---|---|
| `backend/agent/src/ai/openaicompat/stream.go` | +18/−1 | **+1 line** | +14 lines (91–104) | +3 lines (359–362, replaces 1) |

The change, verified in the archived tree at `stream.go:362`:

```go
defer close(out)
// AI-33.5 (R-AIS-033): drain-before-close — see header. Silent
// (network errors ignored), runs inside the producer goroutine's
// existing defer chain (R-ATS-003 preserved: no second goroutine).
defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
```

`import "io"` was already present at `stream.go:111` — no import added. The drain mirrors the shipped idiom at `capture.go:117–122`.

**Untouched, confirmed**: `backend/agent/src/agenttest/*` (conformance suite + stream kit), `backend/agent/src/ai/openaicompat/openrouter/*`, `backend/agent/go.mod`, `backend/agent/go.sum`.

**Invariants preserved**: `R-ATS-003` (single-producer, no second goroutine) · `R-CNF-009` (single closing site, the unique `defer close(out)` at `stream.go:358`) · `R-STK-009` (stdlib-only; `go.mod` stays 3 lines / 0 `require`).

## Requirements covered

| Requirement | Scenarios | Test file(s) | Verdict |
|---|---|---|---|
| `R-AIS-033` — drain-before-close on every exit path | 7/7 | `a_i-33_5a_test.go`, `a_i-33_5b_test.go` | ✅ PASS |
| `R-AIS-034` — cancel before headers, no stream produced | 3/3 | `a_i-33_1_test.go` | ✅ PASS |
| `R-AIS-035` — cancel between frames, bounded close, connection freed | 3/3 | `a_i-33_2_test.go` | ✅ PASS |
| `R-AIS-036` — truly-abandoned + cancelled drops clean, no terminal invented | 3/3 | `a_i-33_3_test.go` | ✅ PASS |
| `R-AIS-037` — cancel after completion is a no-op, close exactly once | 3/3 | `a_i-33_4_test.go` | ✅ PASS |
| `R-AIS-038` — full-package leak check, every exit path, both stream kinds | 3/3 | `a_i-33_5b_test.go` | ✅ PASS |
| **Total** | **22/22** | **6 new test files** | **✅ PASS** |

**Evidence** (per `verify-report.md`, SHA-256 `5be4a404edfa2bd2860195b405df98c5392205d56d08a2f2252a3c86593e294a`):

- `go test -race -count=1 ./...` → exit `0`, 6 packages, 0 FAIL
- `make lint` (`go vet` + golangci-lint v2.9.0) → exit `0`, `0 issues`
- Every scenario runs under `RequireNoGoroutineLeak` × 50; no `t.Parallel()` in any AI-33 file (`R-STK-008`)
- Tasks: 13/13 complete, 0 unchecked. Task 33.2.3 is a **recorded skip** (33.2.2 closed the decision without forcing a production change), not an incompletion.

## Deviations acknowledged

All 8 carried from `verify-report.md` § "Deviations acknowledged". None are blockers; all are documented at source.

| # | Deviation | Disposition |
|---|---|---|
| 1 | AI-33.2 size overage — 578 changed lines vs 400 budget | User-approved `size:exception`, Engram **#2621** (`decision/ai-33-size-exception-ai-33-2`) |
| 2 | AI-33.2 used `serveTranscripts` (same-package) instead of `bridgeServeTranscripts` (other package, unexported) | Behavior identical; `serveTranscripts` is the helper `conformanceBridgeFactory` itself uses. See follow-up #2 |
| 3 | AI-33.2 race assertion refined to `terminalKind ∈ {Completion, Error}` | Spec S-3 wording admits both; both are AI-32.3's documented behavior. Engram **#2619** |
| 4 | AI-33.3 truly-abandoned timing — full-suite leak check uses cancel-then-drain, not truly-abandoned | Truly-abandoned costs 5s/repeat × 50 × 9 = 37m30s, past `go test`'s 10-min timeout. Cancel-then-drain is the conformance case's own ordering and preserves the same physics. Both properties proven separately |
| 5 | AI-33.5 split into 33.5a + 33.5b | Per `tasks.md` line 173 chain plan (file ≥ 400 lines). Commits `99fef5a` + `83eef31` |
| 6 | AI-33.5 production diff +18/−1 | +1 production line; +14 reviewer-facing header doc; +3 refactored defer body |
| 7 | Settle record cosmetic damage, AI-33.3 ordinal 4 | `diagnosis`/`cleanup_evidence`/`process_evidence` carry placeholders; critical fields correct. **Correction of record**: `.claude/evidence/ai-33-3-settle-correction.md`. Reset was deliberately not run (it mutates the ledger AI-33.5 depended on) |
| 8 | `openspec/AGENTS.md` skill reference drift | `test-driven-development/SKILL.md` referenced but does not exist. Pre-existing, unrelated to AI-33. See follow-up #1 |

## Cumulative size:exception

Recorded in Engram **#2624** (`decision/ai-33-size-exception-cumulative`).

| Field | Value |
|---|---|
| User preflight budget | `milestone-controlled` |
| Delivery strategy | `single-pr` |
| Explicit approval | `size:exception` for the AI-33.2 overage (Engram **#2621**) |
| Cumulative disposition | Natural extension of the AI-33.2 approval to the milestone total |
| Cumulative changed lines | 899 per Engram #2624 (`git diff --shortstat main...HEAD` reports 1,918 insertions / 1 deletion for the final tree) |

> **Note on the two counts.** Engram #2624 records 899 cumulative changed lines; the final-tree diff reports 1,919. The two are not reconciled in any source available at archive time — #2624 appears to count a per-subnode aggregate rather than the final tree. Both are recorded here rather than silently picking one. This does not affect the exception's validity: the approval is `milestone-controlled`, so it is not bounded by a line count.

## Spec sync — NOT performed (orchestrator decision required)

**Status**: `openspec/specs/ai-stream-lifecycle/spec.md` does **NOT** contain `R-AIS-033`–`R-AIS-038`. It carries `R-AIS-001`–`R-AIS-014` only. The delta at `specs/ai-stream-lifecycle/spec.md` in this archive remains change-bound, not live.

The sync was held deliberately. A verbatim merge would **violate the target spec's own normative constraints**:

| Conflict | Source-of-truth clause | Delta reality |
|---|---|---|
| **No Go identifiers permitted** | `R-AIS-014`: "No Go type, field, method, interface or package identifier MAY appear in this file." Also the `[!IMPORTANT]` callout (line 14), scenario `S-AIS-040`, and acceptance criterion 3 | Delta contains **59** Go-identifier occurrences — `io.Copy`, `io.Discard`, `resp.Body`, `httpClient.Do`, `httptest.NewServer`, `stream.go:362`, `bridgeServeTranscripts`, … |
| **Amendments are dated blockquotes under the touched section**, not appends | "How to amend this contract" rules 2–3 (lines 37–38): dated blockquote under the touched heading; superseded text struck through, never deleted | Delta is shaped as an `## ADDED Requirements` append block |
| **Section/ordinal continuity** | SOT ordinals run `R-AIS-001`–`014`; scenario ids `S-AIS-001`–`041` already used | Delta jumps to `R-AIS-033`–`038` (milestone-aligned), leaving `015`–`032` unallocated, and uses a different scenario scheme (`R-AIS-033 / S-1`) |

The delta's own header (line 4) states it amends **§ 4 and § 5** of the contract "per the contract's amendment rule (line 30–41)" — i.e. the intended shape is a **dated amendment blockquote in §§ 4–5**, not a new requirements block appended to § Requirements.

**Held per**: the orchestrator's explicit launch instruction ("if the source-of-truth file does NOT yet include R-AIS-033..038, note this as an open item; the orchestrator will handle the sync"), the `sdd-archive` rule "If the merge would be destructive … WARN the orchestrator and ask for confirmation", and `openspec/config.yaml` `rules.archive: Warn before merging destructive deltas`.

**Recommended resolution** — one of:

1. **Restate the six requirements without Go identifiers** as a dated amendment blockquote in §§ 4–5 of the SOT, keeping the code-level detail in the archived delta. Preserves `R-AIS-014`. *Recommended.*
2. **Relocate the requirements to an implementation-level capability spec** (e.g. `openspec/specs/ai-provider-text-stream/` or a new `ai-openaicompat-cancellation`), where Go identifiers are permitted, and amend `ai-stream-lifecycle` §§ 4–5 with a pointer only.
3. **Amend `R-AIS-014` first** to scope the no-Go-identifier rule, then merge. Highest blast radius — `R-AIS-014` is cited by acceptance criterion 3 and `S-AIS-040`.

## Open items for follow-up

| # | Item | Detail | Owner |
|---|---|---|---|
| 1 | `AGENTS.md` skill reference drift | `openspec/AGENTS.md` names `/Users/braejan/.claude/skills/test-driven-development/SKILL.md`, which does not exist; `sdd-apply` dispatches to `sdd-apply/strict-tdd.md` instead. Pre-existing, unrelated to AI-33 | Separate tidy PR |
| 2 | `serveTranscripts` vs `bridgeServeTranscripts` promotion | Two near-identical transcript helpers in different packages. Resolve by promoting `serveTranscripts` to the conformance package or adding an exported wrapper. Raised as SUGGESTION in `verify-report.md` § "Issues found" | Future cleanup change |
| 3 | **Spec sync unresolved** | See section above. Blocks `R-AIS-033`–`038` from being live contract | **Orchestrator, before or with the PR** |
| 4 | SDD artifacts + evidence are untracked | `openspec/changes/archive/2026-08-07-cachicamas-ai-cancellation/` and `.claude/evidence/` are untracked in git (`git status` shows `??`). They are **not** in any of the 6 commits | **Orchestrator, at PR time** — commit them or they will not ship |
| 5 | AI-33.3 settle ordinal 4 placeholders | `.claude/evidence/ai-33-3-settle-correction.md` is the correction of record. No reset performed (deliberate) | Accept as-is |

## Final-state handoff

Authoritative facts at close, for any future session. These outrank intermediate snapshots.

- **Verdict**: PASS. 6/6 requirements, 22/22 scenarios, 0 FAIL, 0 lint issues, 0 CRITICAL, 0 WARNING.
- **Branch**: `feat/ai-33-cancellation` @ `83eef31`, base `main` @ `e9a8054`, 6 commits, clean working tree except untracked SDD artifacts + evidence (open item #4).
- **Production surface**: `backend/agent/src/ai/openaicompat/stream.go` only. One line of logic at `stream.go:362`. Nothing else in `backend/` changed.
- **`go.mod` / `go.sum`**: unchanged. `R-STK-009` stdlib-only posture intact.
- **Conformance suite**: untouched and green.
- **Tasks**: 13/13 complete; 33.2.3 recorded-skip.
- **Spec sync**: **NOT DONE** — `R-AIS-033`–`038` are change-bound, not live. See open item #3.
- **No PR opened, nothing pushed.** The orchestrator owns the PR phase.

### Gate results at archive time

| Gate | Result | Basis |
|---|---|---|
| Native Review Receipt | **Unmanaged** — proceed | `gentle-ai review status --cwd .` → `status: clean`, `entries: []`, `locks: []`. No review governs this change; no failed or pending review artifact exists |
| Task Completion | **PASS** | 0 unchecked implementation tasks in the persisted `tasks.md`; no archive-time reconciliation performed |
| CRITICAL findings | **PASS** | 0 CRITICAL in `verify-report.md` |
| Action Context | **PASS** | Not `workspace-planning`; all writes inside the `ai-33` worktree |

## Archive references

### Engram observations

| Phase | Artifact | Engram |
|---|---|---|
| Explore | `explore.md` | **#2611** |
| Propose | `proposal.md` | **#2612** |
| Spec | `specs/ai-stream-lifecycle/spec.md` (delta) | **#2613** |
| Design | `design.md` | **#2614** |
| Tasks | `tasks.md` | **#2616** |
| Apply | AI-33.1 apply progress | **#2617** |
| Apply | AI-33.4 landed (`e6eb3a1`) | **#2620** |
| Apply | AI-33.4 race terminal-kind refinement | **#2619** |
| Apply | AI-33.5 apply progress (drain + leak check) | **#2622** |
| Decision | AI-33.2 `size:exception` approved | **#2621** (`decision/ai-33-size-exception-ai-33-2`) |
| Decision | AI-33 shipped — `size:exception` cumulative | **#2624** (`decision/ai-33-size-exception-cumulative`) |
| Verify | `verify-report.md` | **#2625** |
| Archive | `archive-report.md` (this file) | `sdd/cachicamas-ai-cancellation/archive-report` |

### Archived artifacts

`openspec/changes/archive/2026-08-07-cachicamas-ai-cancellation/` — `explore.md`, `proposal.md`, `specs/ai-stream-lifecycle/spec.md`, `design.md`, `tasks.md`, `verify-report.md`, `archive-report.md`.

### Apply evidence

`.claude/evidence/` — `ai-33.1-apply-1.txt`, `ai-33-2-apply-3.txt`, `ai-33-3-apply.md`, `ai-33-3-settle-correction.md`, `ai-33-4-apply-2-evidence.txt`, `ai-33-5/`.

### Source code

| Path | Role |
|---|---|
| `backend/agent/src/ai/openaicompat/stream.go` | Production change (`:362` drain, `:91–104` header doc) |
| `backend/agent/src/ai/openaicompat/a_i-33_1_test.go` | R-AIS-034 (226 lines) |
| `backend/agent/src/ai/openaicompat/a_i-33_2_test.go` | R-AIS-035 (390 lines) |
| `backend/agent/src/ai/openaicompat/a_i-33_3_test.go` | R-AIS-036 (279 lines) |
| `backend/agent/src/ai/openaicompat/a_i-33_4_test.go` | R-AIS-037 (322 lines) |
| `backend/agent/src/ai/openaicompat/a_i-33_5a_test.go` | R-AIS-033 drain (423 lines) |
| `backend/agent/src/ai/openaicompat/a_i-33_5b_test.go` | R-AIS-038 full-package leak check (260 lines) |

### Charter

`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` § 1987–2037 (AI-33, Wave 5 — Harden).

## Result contract

```yaml
status: success
change: cachicamas-ai-cancellation
milestone: AI-33
wave: 5 — Harden
closed_at: 2026-08-07
archived_to: openspec/changes/archive/2026-08-07-cachicamas-ai-cancellation/
verdict: PASS
requirements: 6/6
scenarios: 22/22
critical_findings: 0
warnings: 0
deviations_acknowledged: 8
production_files_changed: 1  # stream.go, +1 line of logic
go_mod_state: unchanged (3 lines, 0 require)
conformance_suite: untouched, green
spec_sync: NOT PERFORMED — normative conflict with R-AIS-014 (no Go identifiers); orchestrator decision required
branches_pushed: 0
prs_created: 0
next_recommended: orchestrator PR phase (resolve spec sync + commit untracked SDD artifacts first)
```
