# Archive Report: AG-18 — Implement compaction

> **Change**: `cachicamas-agent-compaction` · **Milestone**: AG-18 (Layer 2 Wave 4, **18 of 24**)
> **Branch**: `feat/agent-layer2-wave4-ag18` · **Base**: `origin/main@f6acc0d2` + planning commit `fbe9b410`
> **Closes**: R-18's **seam 11** (`R-11`), discharging **G3** — the verdict AG-17 explicitly did not discharge (`agent-v1-scope/spec.md` `R-AGS-007`, back-annotated by this change). Wave 4 (AG-17…AG-18) is now complete.
> **Self-verification at apply close**: full suite green under `-race -count=1` (12/12 packages `ok`, EXIT:0, 2m53s wall-clock), `make lint` 0 issues, `make build` clean, `make vuln-check` clean (0 reachable findings). **`sdd-verify` has not yet run independently** — this report records apply-phase evidence, not a separate verify pass, and does not claim one.
> **Delivery**: single PR, `size:exception` pre-authorized by the user against a 1000-line budget excluding `openspec/`, forecast 1585–2695 lines.
> **This is attempt 2.** Attempt 1 was killed by host-wide `ENOSPC` before writing any byte; disk had 24GB+ free at the start of this attempt.

## What shipped

The runtime could measure its context (AG-17) and ask whether to shrink it, but could not shrink it — AG-17's verdict type shipped no field, deliberately. AG-18 is the mechanism, and the first Layer 2 milestone that **removes** something a run committed.

- **History surgery (AG-18.2)** — `history.go`: a fourth transcript-mutating route, `ReplacePrefix`, dispatched through the single validating commit primitive shared by `Append`, `CloseTurn` and `SynthesizeOrphans`. Rejects a tool-bearing summary; renumbers surviving entries positionally while preserving Layer 1 value and origin. Turn marks (`turnID` + entry count at close) recorded through a new package-private `closeTurnMarked` door; the exported `CloseTurn` keeps its exact pre-AG-18 signature and semantics.
- **The verdict carrier** — `context_strategy.go`: `CompactionRequest` (injected provider, options, instruction, cut) and `ContextVerdict.Compaction *CompactionRequest`, the field AG-17 forecast but did not ship. The zero value re-homes the never-compact guarantee from the unconstructible type onto the zero value.
- **The compaction mechanism** — new file `compaction.go`: the call (own injected provider, injected instruction, no Layer 2-authored content on any path), cut resolution (retraction-only to the nearest recorded turn-mark boundary, pairing-closed verification, never expands forward), the prefix replacement (all-or-nothing, gated on `FinishReasonStop`), the stream record (one dedicated turn bracket, `turn_start → compaction_started → cost_turn (iff) → compaction_finished|compaction_failed → turn_end`, through the existing exported turn-event constructors — no new `EventKind`/`TurnOutcome`/`CostLabel`), the explicit cost fold into the run's cumulative, atomicity by ordering (no journal, no rollback — the commit is unreachable until the call succeeds and the replacement validates), and the on-demand door `Harness.Compact` (typed `ErrRunInFlight` refusal, never queued).
- **Harness wiring** — `harness.go`: at the context-strategy consultation site, act on a non-zero `verdict.Compaction`, rebuild the transcript from the mutated history before the attempt loop, and route to `windDownRun` if an interrupt/shutdown cause fired during compaction. Record each attempt's own turn ID during the `turnSink` drain and close through `closeTurnMarked` instead of the exported `CloseTurn`.
- **Fixtures** — `agenttest/compaction_fixtures.go`: `MisalignedCutFixture`/`NewMisalignedCutFixture`, a scripted straddling call/result pair, built `ai`-only (no `agent` import — an `agenttest`→`agent` import would create a test import cycle, discovered and fixed during this change; see Findings).
- **Substrate (Phase 8)** — `doc.go`/`doc_contract_guard_test.go`: the `L2C-07` row's two independently falsifiable clauses amended together, byte-in-sync (route enumeration re-closes at four; identity stability rescoped to a transcript generation). Both substrate filters (`filterOutLoopFiles`, `filterOutLoopHookFiles`) widened by exact filename suffix with this milestone's seven new filenames, byte-in-sync, no wildcard.
- **Four bites, RED-recorded then reverted**: `S-CMP-030` (skip cut resolution → split pair across the boundary), `S-CMP-031` (commit before the call returns → mutated transcript after a failed compaction; found and fixed a genuine fixture-timing bug while authoring this bite — see Findings), `S-CMP-032` (drop the explicit cost fold → `cost_session` short by exactly the compaction's usage), `S-CMP-033` (queue instead of refuse → the typed refusal or the deferred compaction itself fails). Full transcripts recorded in `apply-progress.md`.

## Commits

| SHA | Subject |
|---|---|
| `6e481b4a` | `feat(agent): AG-18.2 history prefix-replacement surgery` |
| `4528dee9` | `feat(agent): AG-18 context strategy compaction verdict carrier` |
| `58c9d024` | `feat(agent): AG-18 compaction mechanism -- call, surgery, atomicity, on-demand door` |
| `dcdc350c` | `test(agent): AG-18 compaction call, stream, recovery and demand-door coverage` |
| `d82737dd` | `feat(agenttest): AG-18 mis-aligned-cut compaction fixture` |
| `b5b8ffa1` | `feat(agent): AG-18 wire compaction into the run driver's turn boundary` |
| `b2b78e14` | `chore(agent): AG-18 phase 8 -- L2C-07 amendment, surface guard, filter widening` |
| `684c66a6` | `docs(0003): AG-18 status line, milestone counter, checklist` |
| `82e0b5ef` | `docs(openspec): AG-18 promote agent-compaction, back-annotate six specs` |
| *(this commit)* | `chore(openspec): AG-18 archive -- apply-progress, archive report, tasks self-check` |

(Planning artifacts — exploration, proposal, design, task breakdown, all eight delta/new spec drafts — were already committed as `fbe9b410` before this apply phase began, per the task brief's stated base.)

## Capabilities promoted

| Capability | Kind | What promoted |
|---|---|---|
| `agent-compaction` | **NEW** | `R-CMP-001`…`014`, `S-CMP-001`…`037` (incl. 4 bites `S-CMP-030`/`031`/`032`/`033`). 333 lines, body byte-identical to the draft, only 3 header lines adjusted for promotion |
| `agent-history` | MODIFIED | `R-HIS-001` (the prefix-replacement route, `S-HIS-100`/`101`), `R-HIS-004` (`S-HIS-102` bite), `R-HIS-005` (generation-scoped identity, `S-HIS-103`), `R-HIS-009` (`L2C-07` both clauses, `S-HIS-104`; `S-HIS-080`'s pre-existing count-assertion defect corrected in the same pass) |
| `agent-context-strategy` | MODIFIED | `R-CTX-003` (zero-value amendment, `S-CTX-022`…`024`), `R-CTX-004` (`S-CTX-025`) |
| `agent-cost-events` | MODIFIED | `R-CST-001`'s per-path table gains the compaction-bracket row (`S-CST-015`/`016`); the iff itself is not relaxed |
| `agent-loop-skeleton` | MODIFIED | `R-LSK-004`'s AG-18 release-scope paragraph (`doc.go` + `doc_contract_guard_test.go` only) and filter-widening paragraph (`S-LSK-029`/`030`); header range extended to `S-LSK-030` |
| `agent-run-driver` | back-annotation | Two Explicit non-requirements rows amended — the compaction-check row now CLOSED, and the History-method row records a deliberate one-third breach (the prefix-replacement route) |
| `agent-v1-scope` | back-annotation | `R-AGS-007`'s R-11/G3 discharge, `R-AGS-013`'s AG-18 inheritance-statement discharge. **`S-AGS-053` ID collision found and corrected** — renumbered to `S-AGS-061`, the true next-free identifier (see Findings) |
| `agent-protocol-events` | back-annotation | Live-emission bullet gains "CLOSED by AG-18" with three recorded facts (dedicated bracket, byte-unchanged files, exactly-one-of-finished-or-failed); compaction-strategy bullet gains the mechanism/strategy-split closure |

## Verification at close

Re-run directly by this apply phase, not inherited from an unverified claim:

- `go test -race -count=1 ./...` from `backend/agent/` — **12/12 packages `ok`** (+1 `[no test files]`), `EXIT:0`, `openaicompat` `171.572s` (genuinely uncached), **2m53s total wall-clock** (`12:39:52`→`12:42:45`)
- `golangci-lint cache clean && make lint` (pinned `v2.9.0` via `make tools`, not the machine's global) — **`0 issues.`**
- `gofmt -l backend/agent` — 15 pre-existing gofmt-dirty files, the **identical set AG-17's own archive report recorded**; none touched by this change
- `make build` clean; `make vuln-check` `EXIT:0`, verified by direct JSON inspection (`0` `"type": "finding"` entries) rather than a trusted summary line
- Import-boundary and ambient-authority guards: all 8 guard tests re-run by exact name, PASS; neither guard file appears in this change's diff; new production files' imports manually confirmed to carry no process/filesystem/environment/network facility
- Every one of the 13 archived files' content verified byte-for-byte before and after `git mv` — `git hash-object` for 11 tracked files, direct hash/checksum for the two dirty/untracked ones (`tasks.md`, `apply-progress.md`) — against the known truncation-into-placeholder risk, not a visual diff
- All four mandated bites (`S-CMP-030`, `S-CMP-031`, `S-CMP-032`, `S-CMP-033`) RED-recorded with real failing-test transcripts from a genuine scratch mutation, then reverted; each revert re-confirmed GREEN before moving on — full transcripts in `apply-progress.md`
- `agent-history`/`agent-context-strategy`'s own pre-existing bites (`S-HIS-030`/`031`, `S-CTX-003`/`008`/`013`/`015`) confirmed still passing, untouched by this change's edits to the files they exercise
- `S-AGS-053`/`S-AGS-061` renumbering re-verified after the edit: 61 scenario definitions, 15 requirement definitions, zero duplicate IDs among actual definitions in `agent-v1-scope/spec.md`
- Base checkout at `/Users/braejan/workspace/witsaba/repositories/cachicamas` never touched; all work confined to the `ag18` worktree throughout

## Findings recorded during apply (not hidden, not silently corrected)

1. **`agenttest`→`agent` import cycle, found and fixed at design-in-code time.** The first draft of `agenttest/compaction_fixtures.go` included a `DriveMarkedHistory` helper importing `package agent`. `go build ./...` succeeded (it does not link test files), but `go vet ./...` failed immediately: `agent`'s own external test binary already links `package agent_test` (which imports `agenttest`) together with `package agent` itself, so an `agenttest`→`agent` import is an import cycle in the test build specifically. Fixed by removing the helper and moving the marks-bearing fixture concept to where it structurally belongs — `package agent_test`'s own `markedHarnessForCompaction` and `package agent`'s own `appendAndMark` — with the package doc comment explaining why.
2. **A "before" snapshot taken after a synchronization point can be too late (S-CMP-031 bite).** The bite's first authored version captured `preEntries` via `<-gate.Reached()` in the test goroutine, but the scratch mutation (moved to before the provider call) runs *before* the gate is ever reached — so both sides of the byte-identity comparison were already-mutated and the bite passed incorrectly on its first run. Fixed by capturing the snapshot synchronously inside the strategy's own consultation callback, guaranteed to run before any mutation on every code-path ordering.
3. **`S-AGS-053` ID collision in the `agent-v1-scope` delta spec**, found while applying the delta's own R-AGS-007 MODIFIED block at promotion. The delta's draft allocates `S-AGS-053` for a new AG-18 discharge scenario, but that ID was already used by the shipped main spec's pre-existing `R-AGS-014` for an unrelated scenario — a genuine `sdd-spec`-phase authoring defect, never caught before promotion, ironic given `R-AGS-014` is itself the append-only-identifier rule this collision breaks. Verified the true next-free ID (`S-AGS-061`; `001`…`060` are contiguously allocated with no gap) and renumbered, recording the correction inline in the promoted text rather than propagating the collision. Full detail in `apply-progress.md`.
4. **Two commit-message drafting slips, self-caught, recorded rather than amended** (this repository's git-safety discipline forbids amending an already-made commit for a wording fix): `dcdc350c`'s message misdescribes where two bites' evidence "lands" (both are already in that same commit, not a later one); `82e0b5ef`'s subject and body enumerate six specs when the commit's own diff touches seven (`agent-protocol-events` was omitted from the prose, though its content is correct and was independently verified). Neither affects code, test or promoted-spec content — both are confined to commit-message prose. Full detail in `apply-progress.md`.
5. **Recurring fixture bug across multiple, independently-authored tests**: a turn that finishes `FinishReasonStop` with nothing steered ends the run immediately, so any fixture needing the context strategy to consult a *second* time needs turn 1 to be tool-calling (`FinishReasonToolCalls`), never plain text-stop. This bit `TestCompaction_ReconstructionNamesReplacedTurns` and, independently, three `context_strategy_test.go` tests — each caught via a genuine RED run and fixed the same way. Recorded once here as a class rather than four separate findings; full detail in `apply-progress.md`.

## Carried forward as follow-ups (non-blocking)

1. AG-19 (delegation/subagent) is next. It is not blocked by anything AG-18 leaves open.
2. Compaction **quality** (instruction content, threshold arithmetic, when to compact) stays deferred to Layer 3, named at `AG-18.1` in `agent-v1-scope/spec.md` `R-AGS-006`'s `AGS-D` entry — a delivered mechanism is not a delivered quality bar, and this change is careful throughout not to conflate the two.
3. The `harness.go` line-citation drift class (recorded by every milestone from AG-16 onward that edits this file) recurred again during this change's own citations; re-grepped at apply time per the standing lesson, not propagated forward uncorrected.
4. `sdd-verify` has not yet run independently against this change; the verification above is apply-phase self-verification only.

## State at close

Layer 2 stands at **18 of 24**. Wave 4 (AG-17…AG-18) is complete. AG-19 (delegation/subagent) is next.
