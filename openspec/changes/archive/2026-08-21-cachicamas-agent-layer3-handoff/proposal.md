# Proposal: AG-23 — Publish the Layer 3 readiness contract

| | |
|---|---|
| **Change** | `cachicamas-agent-layer3-handoff` |
| **Milestone** | AG-23 (`docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:2095-2155`) · Wave 6 · **24 of 24 — Layer 2's exit; the surface freezes here** |
| **Closes** | R-21 |
| **Depends on** | AG-21 (merged), AG-22 (merged, PR #184, `9c55eeda`) |
| **Worktree / base** | `cachicamas-worktrees/ag23-layer3-handoff` off `main@9c55eeda` |
| **Artifact store** | hybrid (`openspec/changes/…` + Engram `sdd/cachicamas-agent-layer3-handoff/*`) |
| **Delivery** | single-pr |
| **Review budget** | 1000 counted lines (`openspec/**` excluded; production + tests + `docs/**` counted) — forecast **≈ 917**, see § Budget |
| **TDD runner** | `cd backend/agent && go clean -testcache && make test` (`make test` is `go test -race -v ./...` — the shipped Makefile carries no `-count=1` of its own; a testcache clean is what proves an uncached run). Focused/single-package runs use `go test -race -count=1 ./...` directly. **A `(cached)` result is not evidence** either way |
| **Prefix** | **`L3H`** — verified free this phase: `rg 'L3H\|R-L3H\|S-L3H' openspec/` → **0 occurrences across 0 files** (whole tree, including `changes/` and `changes/archive/`) |
| **Exploration** | Engram `sdd/cachicamas-agent-layer3-handoff/explore` (#3562) |

> Every `file:line` below was opened in **this worktree, this phase**. Citations inherited from the exploration were re-resolved before use.

## Intent

Layer 2 is feature-complete and unconsumed. Every helper a Layer 3 application would need to test against the harness — `scriptedPermissionPolicy` (`permission_protocol_test.go:57`), `NoOpPermissionPolicy` (`permission_policy_helpers_test.go:49`), `ScriptedTool` (`scripted_tool_test.go:28`) — is declared in a `_test.go` file under `package agent_test` (verified: all three, this phase). Go makes those **unimportable from any other package, at all**. So today the claim "Layer 3 can build on this" is unfalsifiable: nothing outside `src/agent` has ever built a harness.

AG-23 makes it falsifiable. It freezes the v1 surface, proves sufficiency by **consuming** it from a genuinely third-party package, and ships the deterministic substrate that consumption needs. It also closes the two follow-ups AG-22 handed forward (W-5, W-6), because there is no milestone after this one to hand them to.

## Scope

### In scope

1. **AG-23.2 — the packaged test kit** at a new sibling `backend/agent/src/apptest/`: an importable, non-`_test.go` scriptable `agent.PermissionPolicy`, a scriptable `agent.Tool`, and thin drain/assert helpers over the existing `agent.CheckStream`.
2. **AG-23.2 — runnable examples** in `backend/agent/src/agent/example_test.go` (`package agent_test`), each with a mandatory `// Output:` block: building a harness, driving a run, consuming events, handling a suspension.
3. **AG-23.1 — the consumer proof** at a new sibling `backend/agent/src/layer3handoff/` (`package layer3handoff_test`), exercising the seven acceptance capabilities in one run.
4. **AG-23.3 — the compatibility statement**: `decision.md` in this change folder + a pointer GoDoc section in `src/agent/doc.go`.
5. **The Layer 2 completion checklist walk**, including flipping the stale `[ ]` rows in doc 0003.
6. **W-6 disposition** — restore `scheduler.go` to the freeze list; permanently record the `import_boundary_test.go` release with its reason (D-6).
7. **W-5 disposition** — fix the `Harness.Run` forwarder panic race, RED-first (D-5).

### Out of scope

| Excluded | Reason |
|---|---|
| Implementing **anything** in Layer 3 | Charter, `0003:2105` |
| A miniature **coding** agent as the consumer proof | `0003:2101` and `0003:112`: a test that only makes sense for a coding agent is a boundary violation of the same weight as an import violation. Mechanical check: `D-1`'s guard sweep plus a source-level scan proving the proof package imports **only** `src/agent`, `src/apptest`, `src/agenttest` and stdlib — no file, shell, terminal, editor or repository concept can enter through any of them |
| Any new module dependency | Both scope fences assert `go.mod`/`go.sum` byte-unchanged |
| Re-litigating the deferred capabilities | `0003:2185-2192` — AG-23.3 **states** them with post-v1 paths; it does not decide them |

## Resolved decisions

### D-1 — The consumer proof is a new sibling package `backend/agent/src/layer3handoff/`

**Decided.** `package layer3handoff_test`, sibling to `src/agent`, `src/ai`, `src/agenttest`, `src/handoff` under `backend/agent/src/`.

**Evidence.** AG-23.1's scenario names "the AI-40 consumer-proof discipline, one layer up" (`0003:2126`). AI-40 placed Layer 1's proof at `backend/agent/src/handoff/`, a brand-new sibling, and its guard comment records why a nested or in-package home was rejected: a genuinely third-party consumer package is the whole point. `src/handoff` is taken by Layer 1, so the name disambiguates by the **receiving** layer, matching this change's own capability name `agent-layer3-handoff`.

**Rejected — `package agent_test` inside `src/agent/`.** Cheapest (already swept by check 2, zero guard edit) but it is what AI-40 explicitly refused: the library testing itself. It also cannot prove the surface is reachable from outside the layer, which is the only thing this milestone exists to prove.

**Rejected — a subpackage `src/agent/layer3handoff/`.** Tempting: `layer2Pattern = modulePath + "/src/agent/..."` (`import_boundary_test.go:136`) sweeps subpackages automatically, so it needs no guard edit and would let W-6 restore `import_boundary_test.go`. Refused on two grounds. (a) `allowedProductionPrefixes` admits `modulePath + "/src/agent"` **as a prefix** (`:188`), so a kit or proof nested there would be silently admitted into Layer 2's *production* closure — destroying the separate-scan property AD-3 built deliberately (`:214-219`, `S-AGP-025`/`S-AGP-026`: the production closure is proven never to see the test substrate). (b) A Layer 3 application living inside Layer 2's own tree is a position error; Layer 3 is above Layer 2, not inside it.

**Rejected — `apphandoff`** as the directory name: symmetric with `apptest` but loses the direct link to the milestone, the change name and the spec capability.

### D-2 — The kit is a new sibling `backend/agent/src/apptest/`, not an extension of `src/agenttest`

**Decided.** New package `apptest`, named by the same discipline as `agenttest` — for the layer that *consumes* it. `agenttest`'s own doc.go declares it "Layer 1's external-consumer proof, and … an importable testing library" for the agent layer (`agenttest/doc.go:1-3`); `apptest` is the same role one layer up, for the application layer.

**Exported surface (shape; sdd-design owns the exact signatures).** A scriptable `PermissionPolicy` resolving queued verdicts in order; a scriptable `Tool`; a drain helper delegating validation to the already-exported `agent.CheckStream`. Nothing else — every other capability the proof needs already exists on Layer 2's public surface (`NewMapRegistry`, `NewSeededHistory`, `Entries()`/`Entry.Message()`, `Run`/`Steer`/`Interrupt`/`Shutdown`/`Compact`) or on Layer 1's (`agenttest.NewProvider`/`Script`/`Step`).

**Why Layer 1's kit cannot be reused — verified, not inherited.** `permission_policy_helpers_test.go:6-17` records the mechanism, and `src/ai/import_boundary_test.go:58-62` + its `forbiddenPrefixes` confirm it this phase: `layer1Patterns` sweeps `src/ai/...`, `src/agenttest/...` and `src/handoff/...` together, and Layer 1's forbidden list denies `.../src/agent` by name. Any `agent.PermissionPolicy` implementation necessarily references Layer 2 types, so it **cannot compile inside `agenttest` without failing Layer 1's own guard**. Independently, `hksScopeFenceByteUnchangedFiles`'s sibling assertion requires `backend/agent/src/agenttest/` to be byte-unchanged against the merge base (`hooks_test.go:1591-1598`), filtered only for AG-22's two `tracetest` files — so extending `agenttest` would fail a second, unrelated guard.

**Why `agent.CheckStream` needs no promotion.** It is already production-exported at `stream_check.go:92`, and its doc comment already says it "is reused wholesale by the Layer 3 readiness contract's kit at AG-23 (VL2-SEAM-14)". Cite it; ship nothing.

**"Deterministic, with no wall clocks" is proven, not asserted.** Two mechanical checks, not a claim in a comment: (a) a source-level import scan over `src/apptest`'s and `src/layer3handoff`'s own `.go` and `_test.go` files denying `time` as a *direct* import alongside the `os`/`net`/`syscall`/`io/fs` families (the check-4 shape at `import_boundary_test.go:526-549`, extended to the two new trees and to test files); (b) a repeat-run equality assertion — the same script drained twice yields byte-equal event sequences under `agent.CheckStream`. Determinism that only holds on one run is not determinism.

### D-3 — The zero-vendor-import proof extends the existing guard; it does not clone it

**Decided.** `import_boundary_test.go`'s single `const layer2Pattern` (`:136`) becomes a pattern **set**, mirroring Layer 1's `layer1Patterns` slice, but split by check so the production closure keeps its curated purity:

- **checks 1, 3, 4** keep scanning `layer2Pattern` alone → `allowedProductionPrefixes` is **not** widened, and Layer 2 production still provably cannot import `apptest`.
- **check 2** (test closure) scans `{layer2Pattern, …/src/apptest/…, …/src/layer3handoff/…}` against `allowedTestPrefixes` extended by the two new prefixes. This is what proves zero vendor imports for the proof, because `go list -deps -test` over the proof's tree yields its whole closure including its external test files.
- a **new check 5**, the check-4 source scan applied to the two new trees including `_test.go` files, carries the zero-I/O and no-wall-clock half (check 3 is production-only by construction: `-test` pulls in `testing`, which imports `os` — `:409-411`).

**Rejected — a second, self-contained guard inside each new package.** It would leave `import_boundary_test.go` byte-unchanged and let W-6 restore it (see D-6), but it duplicates the allowlist, creating exactly the drift surface a single deny-by-default list exists to prevent. AG-23.1's scenario says "**the** guard mechanism", definite article, and AI-40 wrote no new guard.

### D-4 — Requirement prefix `L3H`; new capability `agent-layer3-handoff`

**Decided.** Directory `openspec/specs/agent-layer3-handoff/`, prefix `L3H` — the exact analogue of Layer 1's `ai-layer2-handoff` / `L2H`, one layer up.

**Proof it is free, run this phase:** `rg 'L3H|R-L3H|S-L3H' openspec/` → `No matches found. Found 0 total occurrences across 0 files.` None of the 24 taken prefixes (AGE, AGO, AGM, AEV, CNH, APE, AGP, ATT, DEL, AGS, HKS, HIS, AGV, CST, AMT, CAN, RUN, RTY, CMP, CTX, LSK, APP, TLS, PRH) collide.

### D-5 — W-5: the `Harness.Run` forwarder panic race is **fixed here**, RED-first

**Re-verified on this branch, this phase.** `harness.go:781-817` starts a per-attempt forwarder goroutine ranging over the **unbuffered** `turnSink` and sending each event on `sink` (`:815`). It is joined at `<-forwarderDone` (`:834`) only after `Turn` returns normally at `:833`. `Run` registers `defer close(sink)` first in its own stack (`:533`), so it executes **last** during unwind. If `Turn` panics — reachable through the deliberately-unrecovered `PreRequestHook` seam, R-AGS-016 — line `:834` is never reached, and `close(sink)` runs concurrently with a forwarder blocked in `sink <- ev`: send on a closed channel, an unrecovered crash in a goroutine no consumer owns.

**Decided: fix it.** Four reasons. (1) It is a **crash-class defect reachable through a documented v1 seam that this milestone freezes** — shipping it inside a "compatibility statement" would make that statement false on the day it lands. (2) The completion checklist row "the package is race-clean and leak-free" is already `[x]`, closed by AG-21 (`0003:2179`); AG-23.3 must walk that row citing its closing node, and cannot honestly do so while the race stands. (3) The known-limitations register (`0003:2154`) is for **design** limitations with post-v1 paths — no subagent tool, failover declines, never-compact default, the abandoned-consumer contract. A send-on-closed-channel bug is not a design limitation and must not be laundered as one. (4) AG-23 is the exit milestone: a carry-forward has nowhere to go, and "Layer 3 will find it" is the leak this milestone exists to prevent.

**The design constraint that makes this non-trivial — named, not discovered later.** The naive fix (move the join into a `defer`) converts a crash into a **hang**: if the consumer has abandoned `sink`, the forwarder is blocked forever and the join never returns, deadlocking the panic unwind. The fix must therefore both (a) guarantee the forwarder terminates — its send selecting on the run context, whose `cancel(nil)` at `:539` already runs *before* `close(sink)` under LIFO — and (b) establish happens-before between forwarder exit and `close(sink)`. Trading a crash for a hang is not a fix, and the RED must fail for the send-on-closed reason specifically, verified by watching it fail before the production edit exists.

**Abort condition.** If sdd-design finds the correct fix cannot be bounded inside this budget, it stops and reports rather than shipping a partial one; the fallback is a budget extension request to ~1150, **not** a demotion to the limitations register.

**Rejected — carry forward again.** Matches AG-22's own precedent for W-1, and AI-40's zero-production-code posture. Refused: those carry-forwards had a successor milestone. This one does not.

### D-6 — W-6: restore `scheduler.go`; close the `import_boundary_test.go` release permanently

**The mechanism, read before deciding.** `TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind` (`hooks_test.go:1563`) resolves a base ref via `hksResolveBaseRef` (`:1674-1684`) — `AG20_BASE_REF` if set, otherwise `git merge-base HEAD origin/main` — and asserts `git diff <baseRef> -- backend/agent/src/agent/<name>` is **empty** for every listed file (`:1572-1581`). It is a **per-branch scope fence against the moving merge base**, not a recorded hash. Since AG-22 merged, the baseline is now `main@9c55eeda`, so "restore an entry" means exactly: *this branch must not touch that file*. The two files therefore decide independently.

**`scheduler.go` — restored.** AG-23 ships no scheduler change: the kit, the proof and the examples are new packages and new test files, and the D-5 fix is confined to `harness.go`. Restoration costs nothing and re-arms a guard AG-22 released for a reason that expired with AG-22.

**`import_boundary_test.go` — NOT restored; the release is made permanent and recorded.** D-3 requires editing it. Restoring it would make AG-23's branch fail its own restored guard — the identical trap AG-22 hit. This is **not** a silent carry-forward: it is a closure, on the merits. That file's allowlists and pattern set are the layer's **designed extension point** — every milestone that admits a dependency or adds a package sibling must edit it, by construction. Two consecutive, independent milestones (AG-22 for the OTel allowlist, AG-23 for the pattern set) needing the same file proves the freeze **entry** was wrong, not that the milestones were. Freezing an extension point is a category error. AG-23 therefore records the permanent exclusion with this reasoning at the list itself (`hooks_test.go`, beside AG-22's own comment) and in the compatibility statement, and the follow-up closes. Nothing is handed to Layer 3.

### D-7 — The compatibility statement lives in **both** places

**Decided**, mirroring AI-40's resolution exactly: a `decision.md` in this change folder carrying the structure a GoDoc comment cannot — §1 how-to-use per audience, §2 the frozen surface **by capability, never by Go identifier**, with experimental corners marked and every seam's injection point and v1 default, §3 the completion-checklist walk table (status / closing node / evidence per row), §4 documented contracts, §5 what Layer 3 inherits, §6 closing-checklist verification — plus one short pointer paragraph in `src/agent/doc.go` as a **new GoDoc section**, not a new machine-checked `L2C` row.

**Why no new `L2C` row.** `doc_contract_guard_test.go` parses `doc.go`'s rows against `expectedLayer2ContractRows` (`:83-90`) and pins `L2C-07` at index 6 (`TestDocContract_RowCountReScoped:292`). A pointer paragraph is not a falsifiable package-wide contract clause and must not be dressed as one. AI-40 made the same call.

**The `0003:2152` leak rule is a hard constraint on §2's prose**: the statement is written without reference to files, shells, skills or terminals. Anything a *coding* agent needs and a different application would not is a leak, and naming it here is cheaper than discovering it at the second application.

### D-8 — doc 0003's completion checklist is updated in this PR

**Decided.** Nine rows are stale `[ ]` today — `0003:2161` (AG-03), `:2162` (AG-04/05/06), `:2164` (AG-07), `:2165` (AG-08), `:2166` (AG-09), `:2167` (AG-10), `:2169` (AG-11.2/AG-10.1/AG-03.3) — every one closed by a milestone that shipped long before Wave 6. **Each is a documentation defect this milestone closes**, and AG-23.3's item-by-item walk (`0003:2153`) is exactly the audit that finds them. Row `:2181` (AG-23 itself) flips when this change lands. The walk table in `decision.md` is the evidence; the checkbox flip is its consequence. A walk that leaves the boxes wrong has not been walked.

## Capabilities

### New capabilities

- `agent-layer3-handoff` — the frozen v1 surface, the consumer proof, the packaged kit, the runnable examples, the known-limitations register. Prefix **`L3H`** (verified free above).

### Modified capabilities

- `agent-package-scaffold` — `R-AGP-003`'s guard shape changes: one pattern becomes a per-check pattern set, `allowedTestPrefixes` gains two entries, and a fifth check lands. The delta pins that the two new trees are swept, that the **production** closure still never admits either, and that check 5 bites on a planted `time`/`os` import in each new tree.
- `agent-hook-taxonomy` — `R-HKS-010`'s byte-unchanged list gains `scheduler.go` back and records the permanent `import_boundary_test.go` exclusion (D-6).
- `agent-run-driver` — the `Harness.Run` forwarder gains a joined, cancellation-terminated exit on the panic-unwind path (D-5).
- `agent-v1-scope` — `R-AGS-016`'s unrecovered-panic seam keeps its semantics (the panic still propagates uncontained) while no longer being able to crash a goroutine the caller does not own.

## Every existing guard, and how it stays green

| Guard | Where | How AG-23 keeps it green |
|---|---|---|
| Import-boundary checks 1/3/4 | `import_boundary_test.go:239,412,526` | `layer2Pattern` and `allowedProductionPrefixes` are **untouched**; the two new trees enter only checks 2 and 5 |
| Import-boundary check 2 | `:274` | New prefixes added to `allowedTestPrefixes` in the **same commit** as the packages they admit |
| No-ambient-authority | `ambient_authority_test.go` (frozen entry) | Not edited; new packages perform zero I/O, proven by check 5 |
| Doc-contract rows | `doc_contract_guard_test.go` | `doc.go` gains a GoDoc **section**, no `L2C` row; row table and index-6 pin unchanged (D-7) |
| `hksScopeFenceByteUnchangedFiles()` | `hooks_test.go:1495` | 16 entries untouched, `scheduler.go` restored (17th); `harness.go` and `import_boundary_test.go` are not on the list |
| `src/agenttest/` sibling freeze | `hooks_test.go:1591-1598` | Kit ships as `src/apptest`, **not** inside `agenttest` (D-2) |
| `del024ByteUnchangedFiles()` | `scope_fence_test.go:20-30` | All 7 entries (`event.go` … `cost_events.go`) untouched |
| `src/ai/` byte-freeze | both fences | Layer 1's guard needs no edit — `layer1Patterns` does not sweep the new siblings, and Layer 1's forbidden list already denies `src/agent` |
| `go.mod` / `go.sum` freeze | both fences | Zero new dependencies |
| Exported-surface pins | `hooks_test.go:1623-1644` | 25 event kinds, 5 `Harness` methods, `Turn`/`Run` signatures all unchanged — D-5 changes `Run`'s internals only |

## Budget forecast

| Bucket | Files | Forecast |
|---|---|---|
| Production | `src/apptest/{doc,permission,tool}.go`, `src/layer3handoff/doc.go`, `src/agent/doc.go` (+section), `src/agent/harness.go` (D-5 fix) | **≈ 245** |
| Test | `src/layer3handoff/layer3handoff_test.go` (~260), `src/agent/example_test.go` (~140), `src/apptest/*_test.go` (~120), `import_boundary_test.go` (+45), `hooks_test.go` (+12), the D-5 RED (~80) | **≈ 657** |
| `docs/**` | `0003-…md` — 9 checkbox flips + AG-23 status | **≈ 15** |
| **Counted total** | | **≈ 917 / 1000 — fits** |
| `openspec/**` (excluded) | proposal, design, spec + 4 deltas, tasks, decision.md, verify-report | ~1400, not counted |

**The marginal item is D-5.** Its fix plus RED is ≈ 95 counted lines. If the cancellation-safe shape needs more than forecast, it is the one part that pushes the milestone over — the response is a budget extension to ~1150 with the reason recorded, never a silent demotion of the race to the limitations register.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| D-5's fix converts the crash into a deadlock on an abandoned consumer | Med | Named as the design constraint up front; the RED must fail for the send-on-closed reason *specifically*, and a second test must prove an abandoned consumer still unwinds |
| The consumer proof drifts toward a coding agent | Med | Check 5's import scan makes it mechanical: no file/shell/process import can enter the proof at all |
| `apptest`'s scripted policy re-implements the gate instead of scripting it | Med | It resolves queued verdicts only; every assertion delegates to `agent.CheckStream` |
| Check 2's widened pattern set passes vacuously | Low | The existing `len(deps) == 0` fatal (`:281-284`) is per-pattern; design must keep it per-pattern, not per-set |
| The checklist walk certifies itself (a row and its evidence edited together) | Med | Each row cites a **closing node and its merged evidence**, resolved against the archive, not against this change |
| The `import_boundary_test.go` exclusion reads as a dropped follow-up | Low | Recorded twice with its reasoning — at the list in `hooks_test.go` and in `decision.md` §4 |

## Rollback

Revert the single PR. The kit and proof are new directories (`src/apptest/`, `src/layer3handoff/`) that nothing else imports; `example_test.go` is new; the `doc.go` section, the `import_boundary_test.go` pattern split, the `hooks_test.go` entry, the `harness.go` fix and the doc 0003 checkboxes are additive hunks with no data or wire consequences. No migration, no dependency, no schema.

## Dependencies

- AG-21 and AG-22 merged (both are; `main@9c55eeda`).
- `agent.CheckStream` exported and stable (`stream_check.go:92`) — already true.
- `agenttest.NewProvider`/`Script`/`Step` importable from Layer 1 — already true.

## Success criteria — traceability

| # | Criterion | Node | Evidence at verify |
|---|---|---|---|
| 1 | An external-package test builds a harness from injected fakes through the public surface only | AG-23.1 | `src/layer3handoff/layer3handoff_test.go` passes under `-count=1 -race` |
| 2 | It runs a multi-turn conversation with tool execution, a scripted permission suspension, an interrupt and a resumed prompt | AG-23.1 | same test, one run |
| 3 | A second harness is constructed over the first's transcript via `NewSeededHistory` | AG-23.1 | same test |
| 4 | The full event stream drains and validates | AG-23.1 | `agent.CheckStream` report clean |
| 5 | Zero vendor imports, zero I/O, proven by the guard mechanism | AG-23.1 / D-3 | checks 2 and 5 green, and **shown to bite** on a planted import |
| 6 | The kit is importable from an external package and deterministic with no wall clocks | AG-23.2 | check 5 denies `time`; repeat-run byte equality |
| 7 | Every example compiles and runs under the normal test run | AG-23.2 | `// Output:` blocks in `example_test.go` |
| 8 | The v1 surface is enumerated and frozen, by capability, with seams and v1 defaults, free of coding-agent concepts | AG-23.3 | `decision.md` §2 |
| 9 | The completion checklist is walked item by item, each citing its closing node | AG-23.3 / D-8 | `decision.md` §3 + 10 checkbox flips in `0003` |
| 10 | The known-limitations register is stated with post-v1 paths | AG-23.3 | `decision.md` §4 |
| 11 | W-6 closed: `scheduler.go` restored, `import_boundary_test.go` exclusion recorded | D-6 | `hooks_test.go` 17-entry list + comment |
| 12 | W-5 closed: the forwarder cannot send on a closed sink during panic unwind | D-5 | RED watched failing, then green under `-race -count=1` |
| 13 | Layer 2's checklist row `0003:2181` flips | AG-23 | doc 0003 |

## Proposal question round (auto mode — recorded, not asked)

Execution mode is `auto`, so these product forks were decided from evidence rather than asked. Each is stated so a reviewer can overturn it deliberately:

1. **Is a known crash race acceptable in a frozen v1 surface?** Decided *no* (D-5). If the maintainer prefers a documented limitation, D-5 inverts and ≈ 95 counted lines leave the budget.
2. **Does "closing a follow-up" permit deciding the freeze entry was wrong?** Decided *yes* for `import_boundary_test.go` (D-6), on the extension-point argument. If refused, AG-23 must adopt D-3's rejected per-package-guard alternative and accept the duplicated allowlist.
3. **Do stale checkboxes in doc 0003 belong to AG-23?** Decided *yes* (D-8) — the walk is the audit that finds them, and a walk that leaves them wrong has not been walked.
