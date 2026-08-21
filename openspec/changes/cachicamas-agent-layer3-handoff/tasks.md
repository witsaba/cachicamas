# Tasks: AG-23 — Publish the Layer 3 readiness contract

Terminal milestone of Layer 2 (24 of 24). Single PR: SDD cycle + code + `docs/**` + OpenSpec archive.

## Review Workload Forecast

Counted = Go source/tests + `docs/architecture/milestones/0003-...md`. Uncounted = `openspec/**` (delta specs, `decision.md`, back-annotation deltas, archive move — per session config).

| File | Action | Est. lines (counted) |
|---|---|---|
| `backend/agent/src/agent/harness.go` | `forwarderAbort` chan + hoisted `forwarderDone` var + defer + double-select forwarder (WU-1) | 25–35 |
| `backend/agent/src/agent/harness_forwarder_panic_test.go` (new) | 100-iter RED test + both defeat-direction plant/revert notes (WU-1) | 75–95 |
| `backend/agent/src/agent/import_boundary_test.go` | `layer2TestOnlyTreePatterns` set, per-pattern check-2 loop, new check 5 (AST, +`time` family), `allowedTestPrefixes` +2 (WU-2) | 70–100 |
| `backend/agent/src/apptest/doc.go`, `permission.go`, `tool.go`, `drain.go` (new) | Scripted permission policy, scripted tool, `DrainAndCheck` (WU-3) | 130–170 |
| `backend/agent/src/apptest/permission_test.go`, `tool_test.go`, `drain_test.go` (new) | Kit unit tests + repeat-run structural-equality-modulo-identity proof (WU-3) | 130–160 |
| `backend/agent/src/layer3handoff/doc.go` (new, empty) | AI-40-shaped package doc pointer (WU-4) | 5–10 |
| `backend/agent/src/layer3handoff/layer3handoff_test.go` (new) | Sequential 7-stage consumer proof, `package layer3handoff_test` (WU-4) | 220–260 |
| `backend/agent/src/layer3handoff/generic_client_guard_test.go` (new) | Vocabulary scan (comments included) + non-self-tripping pin (WU-4) | 60–80 |
| `backend/agent/src/agent/example_test.go` (new) | 4 `Example*` funcs, mandatory `// Output:` (WU-5) | 120–160 |
| `backend/agent/src/agent/doc.go` | One GoDoc pointer section citing the change by name — **no new L2C row** (WU-6) | 8–14 |
| `backend/agent/src/agent/hooks_test.go` | Restore `scheduler.go` to `hksScopeFenceByteUnchangedFiles()`; rewrite `import_boundary_test.go` release comment as permanent (WU-7) | 3–8 |
| `docs/architecture/milestones/0003-...md` | Flip 8 checklist rows (re-counted at edit time) + AG-23 Wave-6 sentence + "24 of 24" + Layer 2 complete (WU-8) | 15–25 |
| **Counted total** | | **≈ 861–1002** |

`openspec/changes/cachicamas-agent-layer3-handoff/decision.md` (new, six AI-40-shaped sections), three back-annotation delta specs (WU-9), and the archive move (WU-10) are `openspec/**` and excluded per session config.

Midpoint ≈ 931, matching design AD-8's own estimate. **Pre-authorized extension to ~1150** is reserved specifically for WU-1 (D-5, the forwarder fix) if the 100-iteration RED test or its double-select mechanism grows beyond estimate — never for demoting a discharged obligation to the known-limitations register.

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1–10 (whole change) | Forwarder race fix, guard extension, apptest kit, consumer proof, examples, compat statement, W-6 closure, docs, back-annotations, archive | PR 1 (single, `size:exception`) | `cd backend/agent && make test` | Real: full suite `-race -count=1 ./...` incl. `src/apptest`, `src/layer3handoff`, `example_test.go` execution | Revert the merge commit — restores 2-entry `layer2TestOnlyTreePatterns` absence, removes `src/apptest`/`src/layer3handoff`, restores `hksScopeFenceByteUnchangedFiles()` to its pre-AG-23 16 entries, reverts doc 0003 to "23 of 24" |

---

## WU-1 — AD-1 forwarder-race fix (`harness.go`, RED-first)

- [x] 1.1 Enumerate every goroutine in `Run` that sends on `turnSink`/`sink`, confirming all live on `Turn`'s own goroutine (no stranded producer on abort) — apply obligation from design AD-1.
- [x] 1.2 **RED**: create `harness_forwarder_panic_test.go`, `TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink` — 100 iterations, fresh `Harness` each, `PreRequestHook` panics with a runtime-minted unique sentinel, consumer drains only the run-open portion then abandons the sink, recover asserts sentinel IDENTITY, then drains to close. Run `go test -race -count=1 ./src/agent/... -run TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink -v`. **Watch it fail for the send-on-closed-channel reason specifically**: confirm the failure output is a `panic: send on closed channel` crash trace (not a timeout, not a race detector report, not a compile error) pointing at the forwarder's `sink <- ev` line — reject the RED if the message differs. Discharges the RED half of `R-L3H-011`'s non-panic-behavior baseline and `R-RUN-014`.
- [x] 1.3 **GREEN**: add dedicated `forwarderAbort` channel, hoisted run-scoped `forwarderDone` var, and a defer registered immediately after the sink-close defer's registration point (LIFO: it runs BEFORE `close(sink)`) executing `close(forwarderAbort); if forwarderDone != nil { <-forwarderDone }`. Forwarder's receive AND send both `select` on abort (double select). Re-run 1.2 `-count=1`; confirm PASS across all 100 iterations.
- [x] 1.4 **Defeat direction A — cancellation-termination removed**: plant a change that replaces `forwarderAbort`/double-select with the rejected `runCtx.Done()` terminator (or removes the abort select entirely, reverting to a bare send). Re-run 1.2's exact command `-count=1`, **watch it FAIL** (crash on iteration 1, or dropped events on the interrupt path), record the failure output, then `git checkout --` the plant and confirm `git status --short` is clean. Discharges `S-RUN-115`.
- [x] 1.5 **Defeat direction B — happens-before join removed**: plant a change that deletes the `<-forwarderDone` join (keeps both selects but removes the wait), leaving `close(forwarderAbort)` and `close(sink)` back-to-back so both select cases race. Re-run 1.2's exact command `-count=1`, **watch it FAIL** (uniform random pick crashes roughly half the iterations; confirm at least one crash across the 100-iteration loop, or re-run once more if the ~2⁻¹⁰⁰ miss window is hit), record the failure output, then revert and confirm clean. Discharges `S-RUN-116`.
- [x] 1.6 Run the full existing `src/agent` suite `-race -count=1` to confirm stream identity on non-panic paths is unchanged (mechanism argument, not re-tested per-event). Discharges `R-RUN-014`, `S-RUN-114`, `S-RUN-117`.

**Focused test**: `go test -race -count=1 ./src/agent/... -run TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink -v` plus full package suite.
**Rollback boundary**: revert `harness.go` and delete `harness_forwarder_panic_test.go` — no other file depends on the abort channel.

## WU-2 — AD-2 import-boundary guard extension

- [x] 2.1 Read `import_boundary_test.go` checks 1–4 to confirm current line anchors (checks 1/3/4 stay untouched; check 2's per-invocation vacuity floor and check 4's AST `parser.ParseFile(..., parser.ImportsOnly)` mechanism are the two things check 5 reuses).
- [x] 2.2 Add `layer2TestOnlyTreePatterns` (the `src/apptest/…` and `src/layer3handoff/…` patterns) as a new const set, keeping `layer2Pattern` untouched and `allowedProductionPrefixes` untouched.
- [x] 2.3 Change check 2 to loop over `{layer2Pattern} ∪ layer2TestOnlyTreePatterns`, calling `listNonStdlibDeps(p, true)` **once per pattern**, with the `len(deps)==0` fatal **inside the loop, naming the pattern** — this is the per-pattern vacuity floor (a mistyped pattern must fail by name, not pass because the unioned set is non-empty). Add the 2 new entries to `allowedTestPrefixes` in the same commit.
- [x] 2.4 **RED-first bite for the per-pattern floor**: plant a bogus/mistyped pattern in `layer2TestOnlyTreePatterns`, run the check-2 test, **watch it FAIL naming the bogus pattern**, revert, confirm clean.
- [x] 2.5 Add check 5: verbatim reuse of check 4's AST mechanism over every `.go` file (including `_test.go`) in both new trees, denying families `{os, net, syscall, io/fs}` **plus `time`**, with a per-tree zero-files floor. State explicitly in the task's evidence: this is a **zero-hop AST source scan**, so it is NOT blinded by a `go list -deps` closure filtered with `{{if not .Standard}}` — that filtered form must never be offered as evidence for the zero-I/O claim (a stdlib-mediated import reaches `os`/`net` invisibly to a `.Standard`-filtered listing but is still caught here because the scan reads import declarations directly, not the resolved dependency graph).
- [x] 2.6 **RED-first bite for check 5**: plant a direct `time` import and a direct `os` import (separately, one at a time) inside `src/apptest` or `src/layer3handoff`, run check 5, **watch each FAIL naming the family and the file**, revert, confirm clean after each.
- [x] 2.7 Confirm behavioral half of the `time` denial: `apptest.ScriptedTool` records tool-call ordering via gate channels, never `time.After`/wall-clock timestamps (AD-2's stated consequence — no `startAt` wall-clock field).

**Focused test**: `go test -race -count=1 ./src/agent/... -run TestLayer2` (confirm exact check function names by reading `import_boundary_test.go` at edit time).
**Runtime harness**: N/A — AST-only guard, no runtime scenario beyond the Go test itself.
**Rollback boundary**: revert `import_boundary_test.go` only — checks 1/3/4 and `allowedProductionPrefixes` are untouched, so a revert cannot affect production closure.
Discharges: `R-L3H-002` (guard mechanism half).

## WU-3 — AD-3 `src/apptest` kit

- [x] 3.1 Create `src/apptest/doc.go` (package doc — importable production source, per `R-L3H-003`).
- [x] 3.2 Create `permission.go`: `NewScriptedPermissionPolicy(verdicts ...agent.PermissionVerdict)` — FIFO under mutex; `Resolve`/`Remember` satisfying `permission_protocol.go`'s interface; `RememberReturns` field; `Resolved() []ai.ToolCall`; `Exhausted() bool`. Exhausted-queue default returns `AllowOnce` and **latches** `Exhausted()` — never wedges.
- [x] 3.3 Create `tool.go`: `NewScriptedTool(name, effect, script func(ctx, args []byte, policy agent.PolicySlot) (agent.Result, error))` implementing `Name`/`EffectClass`/`Run`; `Invocations()`/`RecordedArgs()` mutex-guarded; no wall clock.
- [x] 3.4 Create `drain.go`: `DrainAndCheck(sink <-chan *agent.Event) ([]agent.Event, agent.StreamReport)` — drains to close, derefs preserving order, delegates to `agent.CheckStream` **wholesale** (never re-implemented).
- [x] 3.5 Compile pins: `var _ agent.PermissionPolicy = (*ScriptedPermissionPolicy)(nil)`, `var _ agent.Tool = (*ScriptedTool)(nil)`.
- [x] 3.6 Write `permission_test.go`, `tool_test.go`, `drain_test.go` covering FIFO order, exhaustion latch, invocation recording, and `DrainAndCheck` delegation.
- [x] 3.7 **Determinism proof, scoped to structural equality modulo minted identities**: two repeat runs of the same scripted scenario cannot be byte-equal (RunID/TurnID mint from `harness.go`'s/`loop.go`'s process-global atomic counters). Write a repeat-run test asserting: (a) identical event-kind sequences, (b) identity-independent payload projections are equal, AND (c) **the identity values themselves are asserted to differ** between the two runs — this is the anti-vacuity check; a projection that never looks at the identity field would pass even if both runs shared one Harness and the counters never advanced.
- [x] 3.8 Both drains in 3.7 must be `apptest.CheckStream`-clean.

**Focused test**: `go test -race -count=1 ./src/apptest/...`.
**Runtime harness**: real scripted permission/tool scenario against a live `agent.Harness` (not a mock) — this IS the kit's own proof surface.
**Rollback boundary**: delete `src/apptest/` — no production package outside `src/apptest` imports it (enforced by WU-2's guard).
Discharges: `R-L3H-003`, `R-L3H-004` (kit half).

## WU-4 — AD-4 `src/layer3handoff` consumer proof

- [x] 4.1 Create `layer3handoff/doc.go` — package `layer3handoff`, intentionally empty (AI-40 shape, mirrors `backend/agent/src/handoff/doc.go`'s precedent), pointing at `decision.md` for the frozen v1 surface.
- [x] 4.2 Create `layer3handoff_test.go`, package `layer3handoff_test`. One driver `TestLayer3Handoff_ConsumerProof` with **sequential** (never parallel) `t.Run` stages covering all seven acceptance capabilities in order:
  1. Build a harness from fakes (`agenttest` provider, `NewMapRegistry` over an `apptest` tool, `apptest` policy, public `Harness` fields).
  2. Drive Run #1: multi-turn + tool call.
  3. Scripted `PermissionDefer` suspension + resolution via drained permission events.
  4. Drive Run #2 over an `agenttest` hold gate + `Interrupt()`.
  5. Drive Run #3: resumed prompt.
  6. Take Run #1's `History.Entries()` → `Entry.Message()` → `NewSeededHistory` → build Harness #2 → run it.
  7. Every drain routed through `apptest.DrainAndCheck`; every report clean.
- [x] 4.3 End the sequence with `agent.CheckStream` over the fully drained stream (`R-L3H-001`'s explicit closing assertion).
- [x] 4.4 Create `generic_client_guard_test.go`: scan both trees' `.go` bytes (comments **included**) for runtime-**concatenated** word-boundary needles (`file`, `shell`, `terminal`, `editor`, `git`, `repository`, `filesystem`, `directory` — never the bare substring `path`).
- [x] 4.5 **Non-self-tripping pin**: run the guard against the tree's own prose and confirm it does not flag its own needle-construction code (needles must be built by runtime concatenation, never a literal contiguous match of the denied word inside the guard file itself). Bite: plant a fixture tool named `read_file` in either tree, run the guard, **watch it FAIL naming the literal**, revert, confirm clean.
- [x] 4.6 Confirm capability-leak coverage is mechanical, not prose-only: file/shell/process reach requires `os`/`io/fs`/`net`/`syscall`, already denied by name in WU-2's check 5 — cross-reference, do not re-implement.

**Focused test**: `go test -race -count=1 ./src/layer3handoff/...`.
**Runtime harness**: real, full 7-stage `agent.Harness` run sequence — this test IS the runtime harness.
**Rollback boundary**: delete `src/layer3handoff/` — no production code depends on it.
Discharges: `R-L3H-001`, `R-L3H-004` (identity-difference proof), `R-L3H-010`.

## WU-5 — AD-5 runnable examples

- [x] 5.1 Create `src/agent/example_test.go`, package `agent_test` (already swept by WU-2 check 2). Four `Example*` functions, each with a **mandatory** `// Output:` comment block: build harness, drive run, consume events (kind names only), handle suspension (outcome sequence).
- [x] 5.2 Confirm each example **compiles and runs** under `go test` (Go's example-output mechanism, not merely `gofmt`/`go vet`) — run `go test -race -count=1 ./src/agent/... -run Example -v` and confirm each prints its declared output and PASSes.
- [x] 5.3 Confirm no example prints a minted ID (RunID/TurnID) — global counters make them order-dependent within the test process, which would make `// Output:` non-deterministic across runs.

**Focused test**: `go test -race -count=1 ./src/agent/... -run Example -v`.
**Runtime harness**: the examples themselves execute a live `Harness` — no separate harness needed.
**Rollback boundary**: delete `example_test.go` only.
Discharges: `R-L3H-005`.

## WU-6 — AD-6 compatibility statement

- [x] 6.1 Write `openspec/changes/cachicamas-agent-layer3-handoff/decision.md` mirroring AI-40's six-section shape: §1 how-to-use per audience; §2 frozen v1 surface **by capability, never Go identifier** (boxed callout), seams + injection points + v1 defaults, experimental features marked, no file/shell/skill/terminal reference (doc 0003:2152 constraint); §3 checklist walk table (row / status / closing node / evidence); §4 documented contracts + known-limitations register with post-v1 paths, **including the permanent `import_boundary_test.go` release record** (never entering it as a defect, per `R-L3H-008`); §5 what Layer 3 inherits; §6 closing-checklist verification.
- [x] 6.2 §3's checklist walk MUST be scoped to the **property** ("no unchecked row remains; every row cites a closing node with MERGED evidence"), never phrased as a row count — cite each row's evidence from the **archive and merged PRs**, never from this change's own unmerged work, except AG-23's own row.
- [x] 6.3 §2 records the `TurnOptions.PreRequestHook` ruling explicitly: **DECLINED for removal, frozen into v1, post-v1 removal path recorded** — do not describe it as removed or deprecated anywhere in this document.
- [x] 6.4 Add one GoDoc **section** (not a new `expectedLayer2ContractRows` entry) to `src/agent/doc.go`, citing this change by name and pointing at `decision.md`'s archive path. Confirm `doc_contract_guard_test.go`'s row-count check (`expectedLayer2ContractRows`) still passes unchanged — a prose section is invisible to the row parser by construction.

**Focused test**: `go test -race -count=1 ./src/agent/... -run TestDocContract` (confirm exact name at edit time).
**Runtime harness**: N/A — documentation and a doc-comment addition; no runtime behavior.
**Rollback boundary**: revert `doc.go`'s new section and delete `decision.md` independently of any code work unit.
Discharges: `R-L3H-006`, `R-L3H-007`, `R-L3H-008`, `R-L3H-009`.

## WU-7 — W-6 closure (guard restoration)

- [x] 7.1 In `hooks_test.go`, add `"scheduler.go"` back to `hksScopeFenceByteUnchangedFiles()`'s list (17th entry; AG-23 does not touch it). Bite: confirm the fence still bites by planting a one-line scratch comment in `scheduler.go`, watch the fence FAIL naming it, revert, confirm clean.
- [x] 7.2 Rewrite the comment at the freeze-list site recording `import_boundary_test.go`'s release from the fence as **PERMANENT** (not a dropped follow-up): state the category-error reasoning — freezing a designed extension point (the guard itself, which must grow with every new sibling tree) is a category error, not an oversight. Record this reasoning in **both** the guard source comment and `decision.md` §4's known-limitations register (cross-reference WU-6).
- [x] 7.3 Confirm `harness.go` was never on either freeze list to begin with (`hksScopeFenceByteUnchangedFiles()`'s 16 pre-AG-23 entries, `del024ByteUnchangedFiles()`'s 7) — WU-1's edit needs no widening.
- [x] 7.4 Re-run `hksScopeFenceByteUnchangedFiles()`'s test and `del024ByteUnchangedFiles()`'s test (`scope_fence_test.go`) `-count=1`, confirm both green with `scheduler.go` restored and `import_boundary_test.go` still released.

**Focused test**: `go test -race -count=1 ./src/agent/... -run TestHooks_ScopeFence` and the `scope_fence_test.go` suite.
**Runtime harness**: N/A — guard-list edit, structural check only.
**Rollback boundary**: revert `hooks_test.go`'s two edits independently — no other file depends on the freeze-list contents.
Discharges: `R-L3H-011` (guard-list portion).

## WU-8 — Docs: doc 0003 completion

- [x] 8.1 Run `rg '^- \[ \]' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` **at edit time** — do not trust any cached count (verified at planning time as exactly 8: lines 2161, 2162, 2164, 2165, 2166, 2167, 2169, 2181). Re-verify before editing since prior sessions may have shifted line numbers.
- [x] 8.2 Flip the 7 stale rows (2161, 2162, 2164, 2165, 2166, 2167, 2169) from `- [ ]` to `- [x]` — each already cites its closing milestone (AG-03…AG-11.2); no new evidence text needed, only the checkbox.
- [x] 8.3 Flip AG-23's own row (2181) to `- [x]` **last**, after every code/test/doc work unit above is committed — this is the self-closing row and must not flip before the work it certifies exists.
- [x] 8.4 Update line 3's status header from "**23 of 24**" to "**24 of 24**", derived from re-reading the document's own Wave 0–6 table of contents (never restated from memory), and append an AG-23 sentence to the Wave-6 run-on narrative in the house style of the AG-21/AG-22 entries: name the forwarder-race fix (D-5), the guard's fifth check, the `src/apptest` kit, the `src/layer3handoff` consumer proof, the four runnable examples, the compatibility statement, and mark **Wave 6 complete, Layer 2 complete**.
- [x] 8.5 Confirm the appended sentence does not misstate the `TurnOptions.PreRequestHook` ruling (declined, frozen, not removed) or the test-wrapper relocation (delivered in `src/apptest`, not `agenttest`).

**Focused test**: N/A (markdown-only); verification is `rg '^- \[ \]'` returning zero matches in the 2159–2181 range post-edit.
**Runtime harness**: N/A — documentation.
**Rollback boundary**: revert the single doc 0003 commit; no code depends on its prose.
Discharges: doc-contract completion obligation (not tied to a `R-L3H-`/`S-L3H-` ID).

## WU-9 — Three back-annotation deltas (falsified-requirement closure)

- [x] 9.1 Add a MODIFIED-requirement delta to `openspec/changes/cachicamas-agent-layer3-handoff/specs/agent-loop-skeleton/spec.md` amending the requirement/scenario at `openspec/specs/agent-loop-skeleton/spec.md:294` that defers test-convenience wrappers to AG-23: record **DELIVERED, RELOCATED to `src/apptest`** (not `src/agenttest`, which stays byte-frozen), with the reason (a genuinely third-party consumer package cannot live inside `agenttest`, which Layer 1's own import guard sweeps under Layer 1's rules).
- [x] 9.2 Add the matching delta to `.../specs/agent-protocol-events/spec.md` amending `openspec/specs/agent-protocol-events/spec.md:163`, same DELIVERED/RELOCATED text.
- [x] 9.3 Add the matching delta to `.../specs/agent-message-tool-events/spec.md` amending `openspec/specs/agent-message-tool-events/spec.md:109`, same DELIVERED/RELOCATED text.
- [x] 9.4 Confirm each delta names `src/apptest` explicitly (never a vague "a new package") and cites this change (`cachicamas-agent-layer3-handoff`) as the resolving milestone, so no shipped spec references a closed milestone that left it un-amended.

**Focused test**: N/A — OpenSpec delta files, no code.
**Runtime harness**: N/A.
**Rollback boundary**: revert the three delta files independently of code work units; they carry no code dependency.
Discharges: the falsified-requirement obligation flagged by sdd-spec (not itself a formal `R-L3H-` ID — a cross-cutting closure item).

## WU-10 — Archive

- [ ] 10.1 Promote `specs/agent-layer3-handoff/spec.md` to `openspec/specs/agent-layer3-handoff/spec.md` (new).
- [ ] 10.2 Merge the four MODIFIED/ADDED blocks (`agent-package-scaffold` R-AGP-003, `agent-run-driver` R-RUN-014, `agent-hook-taxonomy` R-HKS-010, `agent-v1-scope` R-AGS-016) into their respective promoted specs, each as a range addition — never a restated total.
- [ ] 10.3 Merge WU-9's three back-annotation deltas into `openspec/specs/agent-loop-skeleton/spec.md`, `agent-protocol-events/spec.md`, `agent-message-tool-events/spec.md`.
- [ ] 10.4 Move `openspec/changes/cachicamas-agent-layer3-handoff/` to `openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/`, per the AG-14/AG-19/AG-20/AG-21/AG-22 precedent.
- [ ] 10.5 Verify via `git diff --stat` that every promoted spec body gained a **net-positive** line count (guard against sub-agent truncation — a prior-session recurring defect class); split the promotion across disjoint per-domain writers if a single writer risks truncating a 300+ line body, and audit each writer's output by requirement/scenario ID set, not line count alone.

**Focused test**: N/A — OpenSpec structural move.
**Runtime harness**: N/A.
**Rollback boundary**: `git mv` is reversible independently of the code commits; archive move can be reverted without touching `backend/agent/`.
Discharges: the archive obligation for all five spec sets plus the three back-annotation deltas.

---

## Verification gates (run before PR, and again before merge)

- [x] G.1 `cd backend/agent && make test` — forces `go test -race -count=1 ./...`. A `(cached)` result is **not evidence**; confirm uncached wall-clock duration is reported (`src/ai/openaicompat` alone runs ~170s when uncached).
- [x] G.2 `bin/golangci-lint cache clean && cd backend/agent && make lint` — clean cache first; a stale cache artifact has previously masqueraded as a finding in this repo.
- [x] G.3 `cd backend/agent && make vuln-check` — run as its own gate; it is **not** part of `make all`.
- [x] G.4 `gofmt -l .` (backend/agent module root) — **never `make all`**, whose fmt stage rewrites committed files and manufactures failures. Confirm no newly-dirty file among this change's own additions.
- [x] G.5 Confirm every pre-existing guard stays green: import-boundary (now 5 checks), no-ambient-authority, doc-contract row count, `hksScopeFenceByteUnchangedFiles()` (17 entries incl. restored `scheduler.go`), `del024ByteUnchangedFiles()`, and `backend/agent/go.mod`/`go.sum`/`src/ai/` byte-freeze (`git diff main -- backend/agent/go.mod backend/agent/go.sum backend/agent/src/ai/` empty).
- [x] G.6 Confirm all five planted bites across WU-1/2/4 were watched FAILING for the correct reason, then reverted, with `git status --short` clean after each.

---

## Scenario coverage map

| Requirement | Scenarios | Work unit(s) |
|---|---|---|
| `R-L3H-001` | S-L3H-001…008 (consumer proof, sequencing, CheckStream close) | WU-4 |
| `R-L3H-002` | S-L3H-009…016 (guard mechanism, one guard extended not cloned, closure unwidened) | WU-2 |
| `R-L3H-003` | S-L3H-017…024 (kit importable, scripts, delegates validation, exhaustion latch) | WU-3 |
| `R-L3H-004` | S-L3H-025…030 (static + structural-modulo-identity determinism, identity difference asserted) | WU-3, WU-4 |
| `R-L3H-005` | S-L3H-031…036 (4 examples, mandatory Output, no minted IDs) | WU-5 |
| `R-L3H-006` | S-L3H-037…042 (v1 surface by capability, frozen, experimental marked, no forbidden vocabulary) | WU-6 |
| `R-L3H-007` | S-L3H-043…046 (checklist walk scoped to property, merged evidence, no row-count phrasing) | WU-6, WU-8 |
| `R-L3H-008` | S-L3H-047…050 (known-limitations register, post-v1 paths, no defect misclassification) | WU-6, WU-7 |
| `R-L3H-009` | S-L3H-051…052 (closed table of four forwarded obligations) | WU-6, WU-9 |
| `R-L3H-010` | S-L3H-053…054 (generic-client boundary, mechanical + vocabulary, non-self-tripping) | WU-4 |
| `R-L3H-011` | S-L3H-055…056 (no new event kind/outcome/cost label, go.mod/go.sum byte-identical) | WU-1, WU-7 |
| `NFR-L3H-A`, `NFR-L3H-B` | — (structural/behavioral, proven by the above) | WU-1–WU-5 |
| `R-AGP-003` (MODIFIED) | S-AGP-039…044 | WU-2 |
| `R-RUN-014` (ADDED) | S-RUN-114…117 | WU-1 |
| `R-HKS-010` (MODIFIED) | S-HKS-027, S-HKS-028 | WU-6 |
| `R-AGS-016` (MODIFIED) | S-AGS-069 | WU-6 |
| Back-annotations (agent-loop-skeleton, agent-protocol-events, agent-message-tool-events) | — | WU-9 |

No scenario in the four delta spec files or the new capability spec is left unmapped.

## Key Learnings

1. `decision.md` lives inside the OpenSpec change folder (following the `cachicamas-ai-layer2-handoff` / AI-40 precedent), so it is excluded from the counted review budget along with the rest of `openspec/**`.
2. The doc-contract row registry (`expectedLayer2ContractRows`) lives in `backend/agent/src/agent/doc.go` and `doc_contract_guard_test.go`; AG-23 adds a prose GoDoc section, never a new `L2C` row, which the row parser cannot see by construction.
3. Doc 0003's unchecked-row count was re-verified at task-planning time via `rg '^- \[ \]'` and confirmed exactly 8 (seven stale plus AG-23's own row 2181) — neither the cached design estimate nor the proposal's earlier "nine" was trusted without re-grepping.
4. `import_boundary_test.go`'s check functions live in a `_test.go` file that functions as the production guard mechanism, so its changed lines count toward the "test" category by file suffix even though the code enforces a production-import boundary.
5. `src/handoff/doc.go` (the AI-40 precedent) confirms the exact shape AG-23's `src/layer3handoff/doc.go` must follow: an intentionally empty package doc pointing readers at the change's `decision.md`.
