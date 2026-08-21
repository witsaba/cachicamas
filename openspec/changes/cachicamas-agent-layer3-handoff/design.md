# Design: AG-23 — Publish the Layer 3 readiness contract

Every citation below was re-read in this worktree (base `main@9c55eeda`). `harness.go` sits in **no** freeze list — it is absent from `hksScopeFenceByteUnchangedFiles()`'s 16 entries (`hooks_test.go:1496-1513`) and from `del024ByteUnchangedFiles()`'s 7 (`scope_fence_test.go:21-29`) — and the exported pins tolerate an internals-only fix (`Run` NumIn 4, `hooks_test.go:1638-1645`; Harness 5 methods, `:1627-1630`).

## Technical Approach

Two new sibling trees (`src/apptest/`, `src/layer3handoff/`) consume Layer 2 from genuinely third-party packages; the existing import guard grows a per-check pattern set and a fifth AST check instead of being cloned; the one production-behavior change is D-5's forwarder fix in `harness.go`, RED-first. Everything else is additive: examples, a GoDoc section, `decision.md`, the checklist walk, and the W-6 freeze restoration.

## Architecture Decisions

### AD-1 — The forwarder-race fix (D-5)

**Verified mechanism.** `Run`'s defer stack, in registration order: `defer close(sink)` (`harness.go:533`), the cancel-clear defer ending in `cancel(nil)` (`:535-540`), `defer h.queue.close()` (`:552`), `defer lane.reportOutstanding()` (`:571`). LIFO unwind: lane → queue → `cancel(nil)` → `close(sink)`. The per-attempt forwarder (`:781-817`) does a bare `for ev := range turnSink { …; sink <- ev }` and is joined at `<-forwarderDone` (`:834`) only on `Turn`'s return. The panic vector is real and unrecovered: `applyPreRequestHook` (`loop.go:971-993`) wraps errors but recovers nothing; `loop.go`'s only `recover` is `closeSink`'s own (`:891`), and `closeSink` is **not** deferred at `Turn`'s top — so a panicking `PreRequestHook` (R-AGS-016) leaves `turnSink` unclosed and unwinds `Run` with a live forwarder, and `close(sink)` executes against a sender that can be parked in `sink <- ev`.

**Choice — a dedicated abort channel plus a deferred join, NOT `runCtx.Done()`.**

```go
// registered immediately AFTER :533's defer close(sink), so LIFO runs it
// immediately BEFORE close(sink):
forwarderAbort := make(chan struct{})
var forwarderDone chan struct{} // hoisted to Run scope; reassigned per attempt
defer func() {
    close(forwarderAbort)
    if forwarderDone != nil {
        <-forwarderDone
    }
}()
```

The forwarder's receive and send both select on the abort:

```go
for {
    select {
    case ev, ok := <-turnSink:
        if !ok { return }
        …existing pure reads…
        select {
        case sink <- ev:
        case <-forwarderAbort:
            return
        }
    case <-forwarderAbort:
        return
    }
}
```

**Rejected — selecting the send on `runCtx.Done()`.** On the interrupt path `runCtx` is *already cancelled* while legitimate wind-down events still flow through the forwarder; a `select` with two ready cases chooses pseudo-randomly, so keying on the run context would drop events on a **non-panicking** path. The abort channel closes only inside `Run`'s own defer — on every non-panic path the last forwarder was already joined at `:834` before the defer runs, so both selects degenerate to today's bare receive/send and the event stream is unchanged by construction.

**The resulting unwind order, written out (LIFO):** `lane.reportOutstanding` → `h.queue.close()` → cancel-clear + `cancel(nil)` → **`close(forwarderAbort)`; `<-forwarderDone`** → `close(sink)`.

**Happens-before edge, every exit path.** `close(forwarderAbort)` → forwarder observes it and returns → its `defer close(forwarderDone)` → `Run`'s deferred receive returns → only then does `close(sink)` execute. Normal and error returns still join at `:834`; the deferred re-receive on the already-closed `forwarderDone` is a no-op. Pre-identity refusals and the `NewRunStart` early return have `forwarderDone == nil` and skip the join. The receive-side abort case is what makes the panic-path join *bounded*: `Turn` never closed `turnSink`, so without it the forwarder would block in the range receive forever.

**No hang on an abandoned consumer.** An unbuffered send case is not "ready" without a receiver, so an aborted forwarder deterministically takes the abort case — the join returns, the unwind completes, the sink still closes. No regression of AG-21 (the forwarder now provably exits on *every* path — strictly stronger) or AG-20 (the lane defer's position is untouched). On the panic path the stream may end without a run-close bracket; that path never carried a stream guarantee.

**The RED test** — `harness_forwarder_panic_test.go`, `TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink`: 100 iterations, each with a fresh `Harness` over a scripted provider and `Turn.PreRequestHook` panicking with a runtime-minted unique sentinel; the consumer drains exactly the run-open event then **abandons** the sink; the test goroutine recovers, asserts the recovered value is that exact sentinel (identity, not shape), then drains the sink to close. Why it fails for the send-on-closed reason *specifically*: the abandoned consumer guarantees the forwarder is parked in `sink <- ev` holding the turn-open event when the unwind reaches `close(sink)` — the send can never succeed (nobody receives), so pre-fix its only exit is `panic: send on closed channel`, a process crash naming the defect verbatim in the test log. It is watched failing before the production edit exists.

**Defeat in both directions** (planted at apply, watched failing, reverted, recorded): (A) revert the send to a bare `sink <- ev` → the RED crashes on iteration 1; (B) keep the abort selects but delete the deferred join → `close(forwarderAbort)` and `close(sink)` run back-to-back, a forwarder at its select then has both cases ready and Go picks uniformly at random → per-iteration crash probability ≈ ½, miss probability over 100 iterations ≤ 2⁻¹⁰⁰.

**Stream identity on non-panicking paths, proven three ways:** the mechanism argument above (abort cannot close while a forwarder lives on any non-panic path); the full existing suite — every ordered-kind assertion, `cancellation_interrupt_test.go`'s wind-down sequences — rerun green under `-race -count=1`; and AD-4's consumer proof drains an interrupted run through `agent.CheckStream` clean. **Apply obligation:** before the edit, enumerate every sender to `Turn`'s sink parameter and confirm all run on `Turn`'s own goroutine, so aborting the forwarder strands no producer.

### AD-2 — The guard extension (D-3): pattern set per check, plus an AST check 5

**Verified shapes.** Check 1 (`:239`) and check 2 (`:274`) scan `layer2Pattern` (`:136`) via `listNonStdlibDeps` with a per-invocation `len(deps)==0` fatal (`:246-249`, `:281-284`); check 3 (`:412`) is production-only because `-test` pulls `testing`→`os` (`:409-411`); check 4 (`:526-549`) is an **AST scan** — `parser.ParseFile(…, parser.ImportsOnly)` over families `{os, net, syscall, io/fs}` (`:468`) — not a regex, so comments are never parsed and the 717-char-comment failure class is structurally impossible.

**Choice.** `layer2Pattern` stays a `const`; a new `layer2TestOnlyTreePatterns = []string{modulePath + "/src/apptest/...", modulePath + "/src/layer3handoff/..."}` joins it *only where a check names it*:

| Check | Pattern set | Allowlist | Why |
|---|---|---|---|
| 1 (production closure) | `layer2Pattern` alone | `allowedProductionPrefixes` **untouched** | production provably never imports the kit or proof; this is the property D-1's sibling placement exists to keep (the `src/agent` prefix at `:188` admits any nested tree) |
| 2 (test closure) | `{layer2Pattern} ∪ layer2TestOnlyTreePatterns`, one `listNonStdlibDeps(p, true)` call **per pattern**, the `len(deps)==0` fatal **inside the loop**, naming the pattern | `allowedTestPrefixes` + the two new prefixes, same commit as the packages | the per-pattern floor: a mistyped pattern resolves zero packages and fails by name — it can never pass vacuously |
| 3 (net/fs closure) | `layer2Pattern` alone | unchanged | production-only by construction (`:409-411`) |
| 4 (source scan, production) | `src/agent` production files | unchanged | unchanged |
| **5 (new: source scan, new trees)** | every `.go` file **including `_test.go`** directly inside `src/apptest/` and `src/layer3handoff/` | families `{os, net, syscall, io/fs}` **plus `time`** | zero-I/O + no-wall-clock, by name, zero hops |

**Check 5 mechanics.** The check-4 AST mechanism verbatim (resolve each tree via `runtime.Caller(0)` + `../apptest` / `../layer3handoff`; `parser.ParseFile` `ImportsOnly`; family match), with a per-tree vacuity floor (zero files found fails naming the directory). Denying a direct `time` import removes every wall-clock reachability (`time.Now`/`Since`/`After` are unreferenceable without the import) from the trees' own code; the behavioral half of determinism is AD-3's repeat-run test. Consequence accepted deliberately: the kit's scripted tool drops `agent_test`'s wall-clock `startAt` recording (`scripted_tool_test.go:17,39`), and the proof synchronizes via gate channels, never `time.After`.

**The stdlib blind spot, stated.** Check 5 never lists dependencies, so `{{if not .Standard}}` filtering cannot blind it — it is a zero-hop scan of the trees' own sources. Transitive coverage is closed elsewhere: check 2 pins the trees' non-stdlib closure to exactly the allowlist, and every member carries its own I/O guard (checks 3/4 for `src/agent` production; Layer 1's own guards for `src/ai` and `src/agenttest`). The residual `testing`→`os` baseline is the same unscannable floor check 3's comment already records — restated, not hidden.

**Bite scenarios (spec pins):** plant a `time` import and an `os` import in each new tree, watch check 5 name file and family; plant a bogus pattern, watch the per-pattern floor fatal.

### AD-3 — The kit's exported surface (`src/apptest/`, D-2)

Package `apptest`; files `doc.go`, `permission.go`, `tool.go`, `drain.go`; imports only `context`, `sync`, `src/agent`, `src/ai`. Compile pins: `var _ agent.PermissionPolicy = (*ScriptedPermissionPolicy)(nil)`, `var _ agent.Tool = (*ScriptedTool)(nil)`.

```go
// permission.go — resolves queued verdicts strictly FIFO under a mutex.
func NewScriptedPermissionPolicy(verdicts ...agent.PermissionVerdict) *ScriptedPermissionPolicy
func (p *ScriptedPermissionPolicy) Resolve(ctx context.Context, call ai.ToolCall) agent.PermissionVerdict // interface pin: permission_protocol.go:86
func (p *ScriptedPermissionPolicy) Remember(ctx context.Context, toolName string, outcome agent.PermissionOutcome) bool // returns p.RememberReturns (zero value false)
func (p *ScriptedPermissionPolicy) Resolved() []ai.ToolCall // recorded calls, copy
func (p *ScriptedPermissionPolicy) Exhausted() bool // true iff Resolve outran the queue
// An exhausted queue returns AllowOnce (the agent_test default, :73-76) and
// latches Exhausted — the run never wedges, and the proof asserts full consumption.

// tool.go — mirrors agent_test's ScriptedTool minus the wall clock.
func NewScriptedTool(name string, effect agent.EffectClass,
    script func(ctx context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error)) *ScriptedTool
// Name() / EffectClass() / Run() satisfy agent.Tool (tool.go:182-186);
// Invocations() int and RecordedArgs() [][]byte are the mutex-guarded inspection surface.

// drain.go — the one assert helper; validation is delegated wholesale, never re-implemented.
func DrainAndCheck(sink <-chan *agent.Event) ([]agent.Event, agent.StreamReport)
// blocks until sink closes, dereferences preserving order, returns agent.CheckStream(events).
```

**Reuse vs re-implement, verified.** Reused from Layer 1: `agenttest.NewProvider(scripts ...Script)` (`fake_provider.go:64`) and its Script/Step vocabulary — importable today. Reused from Layer 2 production: `agent.CheckStream` (`stream_check.go:92`), `NewMapRegistry`, `NewSeededHistory` (`history.go:268`), `Entries` (`:407`), `Entry.Message()` (`:145`). Re-implemented one layer up: the scripted policy and tool — both verified unimportable today (`permission_protocol_test.go:57`, `scripted_tool_test.go:28`, `_test.go` declarations), and they cannot move into `agenttest`: `layer1Patterns` (`src/ai/import_boundary_test.go:58-62`) sweeps `agenttest` under Layer 1's guard, which denies `src/agent` by name, and `hooks_test.go:1591-1598` freezes the whole `src/agenttest/` tree byte-unchanged.

**Determinism, proven — and one proposal correction.** Check 5 (no `time`) is the static half; the dynamic half is a repeat-run equality test in `apptest`'s own `_test.go`. D-2's "byte-equal event sequences" is **falsifiable as written**: Run/Turn identities mint from process-global counters (`mintHarnessRunID` `harness.go:43`, `mintLoopRunID`/`mintLoopTurnID` `loop.go:228-235`), so two identical runs *cannot* be byte-equal. The test asserts **structural equality modulo minted identities**: identical event-kind sequences, identical identity-independent payload projections (outcomes, tool names, texts, verdict order), and both drains `CheckStream`-clean.

### AD-4 — The consumer proof (`src/layer3handoff/`, D-1)

`doc.go` (`package layer3handoff`, ~8-line comment: intentionally empty, the test is the deliverable — AI-40's `src/handoff` shape) + `layer3handoff_test.go` (`package layer3handoff_test`). One driver, `TestLayer3Handoff_ConsumerProof`, with **sequential** (never parallel — the transcript hand-off is ordered) `t.Run` stages over shared state:

| # | Capability | Mechanism |
|---|---|---|
| 1 | harness from injected fakes | `agenttest.NewProvider`, `agent.NewMapRegistry` over `apptest.NewScriptedTool`, `apptest.NewScriptedPermissionPolicy`, public `Harness` fields only |
| 2 | multi-turn + tool execution | `Run` #1: scripted tool round + completion; drained via `apptest.DrainAndCheck` |
| 3 | scripted permission suspension | a `Defer` verdict then its scripted resolution, asserted through the drained permission events |
| 4 | an interrupt | `Run` #2 over an `agenttest` gate (`Hold`), `Interrupt()` once the gate is reached, drain to close |
| 5 | a resumed prompt | `Run` #3 on the same harness (post-interrupt re-prompt) |
| 6 | second harness over the transcript | `first.History.Entries()` → `Entry.Message()` → `agent.NewSeededHistory(msgs)` → `Harness{History: seeded}` → one more run |
| 7 | drain + validate, zero vendor imports | every run's `StreamReport` clean; guard checks 2 and 5 green and shown to bite |

All three hand-off signatures verified this phase: `Entries() []Entry` (`history.go:407`), `Entry.Message() ai.Message` (`:145`), `NewSeededHistory([]ai.Message) (*History, error)` (`:268`).

**Generic-client discipline — the mechanical check, both halves.** Capability leaks are import-caught: any real file/shell/process reach needs an `os`/`io/fs`/`net`/`syscall` family import, denied by name by check 5. Vocabulary-only leaks (a fake tool *named* like a coding tool) are import-invisible, so a small committed guard, `generic_client_guard_test.go` in `layer3handoff_test`, scans both new trees' `.go` bytes (comments included — the discipline binds prose too) for runtime-**concatenated** word-boundary needles (`"fi"+"le"`, shell, terminal, editor, git, repository, filesystem, directory — never `path`, which would false-positive on ordinary prose); the trees' own doc comments therefore describe the rule without using the denied words. Plant-proof at apply: name one fixture tool `read_file`, watch the guard name it, revert.

### AD-5 — Examples (AG-23.2)

`backend/agent/src/agent/example_test.go`, `package agent_test` (already swept by check 2; AI-40's exact precedent one layer down). Four `Example*` functions, each with a **mandatory `// Output:` block** so they compile *and run* under `make test`: building a harness (fakes → `Harness` literal), driving a run (drain, print finish reason), consuming events (switch on kinds, print ordered kind names), handling a suspension (scripted `Defer` → resolution, print the permission outcome sequence). Hard determinism rule: outputs print kind names and outcomes only, **never minted IDs** — the global counters make IDs depend on which tests ran first in the process.

### AD-6 — The compatibility statement (D-7) and the checklist walk (D-8)

`decision.md` in this change folder, mirroring AI-40's six sections: §1 how-to-use per audience (Layer 3 application author / doc 0004 reader / reviewer); §2 the frozen v1 surface **by capability, never by Go identifier** (the `[!IMPORTANT]` box), each seam with its injection point and v1 default, experimental/deferred corners marked, under the `0003:2152` hard constraint — no file, shell, skill or terminal reference anywhere; §3 the completion-checklist walk table (row → status → closing node → evidence); §4 documented contracts + the known-limitations register with post-v1 paths (`0003:2185-2192`, restated not re-litigated) + the permanent `import_boundary_test.go` release record (D-6); §5 what Layer 3 inherits; §6 closing-checklist verification. `doc.go` gains one pointer **GoDoc section** citing the change by name (the archive path moves at archive) — **no new `L2C` row**: verified, `expectedLayer2ContractRows` (`doc_contract_guard_test.go:82-91`) parses rows and `TestDocContract_RowCountReScoped` pins `L2C-07` at index 6 (`:292-298`); a prose section is invisible to the row parser.

**Self-certification guard (§3):** every already-shipped row cites **merged** evidence — the closing milestone's archived change under `openspec/changes/archive/` and/or doc 0003's own close amendments and PR SHAs — resolved against the merge base, never against this change. The one exception is row `0003:2181` (AG-23 itself), which legitimately cites this change's own verify gate and flips last.

**D-8 count, corrected.** The proposal says nine stale rows; **there are eight unchecked rows total**, re-verified this phase: seven stale (`2161`, `2162`, `2164`, `2165`, `2166`, `2167`, `2169` — each closed by a pre-Wave-6 milestone) and one self-closing (`2181`, AG-23's own, open by design until this change lands). Tasks and apply MUST re-count at edit time (`- [ ]` within `0003:2159-2181`) and trust neither this number nor the proposal's.

### AD-7 — Guard-green plan

| Guard | Verified | How AG-23 keeps it green / what would break it |
|---|---|---|
| Import-boundary 1/3/4 | `:239,412,526` | untouched pattern + allowlists; broken by widening `allowedProductionPrefixes` — forbidden |
| Import-boundary 2 | `:274` | per-pattern loop + two `allowedTestPrefixes` entries in the same commit as the packages |
| Check 5 (new) | AD-2 | lands with the trees; bite-planted |
| No-ambient-authority | `ambient_authority_test.go` frozen (`hooks_test.go:1510`) | not edited; new trees' zero-I/O carried by check 5 |
| Doc-contract rows | `:82-91`, `:292` | GoDoc section only; a new row would fail the committed table |
| `hksScopeFenceByteUnchangedFiles()` | `:1495-1513` | 16 entries untouched; **`scheduler.go` restored (17th)** and the `:1477-1494` comment rewritten: the `import_boundary_test.go` release re-recorded as **permanent** with D-6's extension-point category-error reasoning; AG-23 must not touch `scheduler.go` and does not (D-5 is confined to `harness.go`, which is on no list) |
| Its loop.go anti-vacuity floor | `:1647-1667` | AG-23 leaves `loop.go` untouched → the floor `t.Skip`s (skip, not failure — verified); the diff-independent pins (`:1608-1645`) still run and stay green (25 kinds, 5 methods, `Turn` NumIn 6, `Run` NumIn 4 — D-5 changes internals only) |
| `src/agenttest/` freeze | `:1591-1598` | kit ships as `src/apptest`; agenttest byte-unchanged (the AG-22 tracetest filter is now vacuous and harmless) |
| `del024ByteUnchangedFiles()` | `scope_fence_test.go:21-29` | all 7 entries untouched |
| `src/ai/`, `go.mod`/`go.sum` | both fences | zero edits, zero new dependencies; `layer1Patterns` does not sweep the new siblings |

### AD-8 — Work units and counted budget

Work units per the work-unit-commits skill — tests with their behavior, each independently revertable:

| WU | Content | Prod | Test | docs |
|---|---|---|---|---|
| 1 | D-5 RED (watched failing) → fix → both defeat plants recorded | ~30 (`harness.go`) | ~85 (`harness_forwarder_panic_test.go`) | — |
| 2 | `src/apptest/` + its tests + guard checks 2/5 + bite plants | ~150 | ~175 (`apptest` tests ~120, `import_boundary_test.go` +~55) | — |
| 3 | `src/layer3handoff/` proof + generic-client guard | ~10 | ~305 | — |
| 4 | `example_test.go` (4 examples, `// Output:`) | — | ~140 | — |
| 5 | freeze restoration + `doc.go` section + doc 0003 flips + `decision.md` (openspec, uncounted) | ~10 | ~14 (`hooks_test.go`) | ~12 |
| **Total** | | **~200** | **~719** | **~12** |

**Counted ≈ 931 / 1000 — fits**, margin ≈ 69. The marginal item remains D-5 (WU-1 ≈ 115): if the abort/join shape grows past forecast, the named response is the pre-authorized extension to ~1150 with the reason recorded — never demoting the race to the limitations register.

## Data Flow

    layer3handoff_test ──imports──▶ apptest (policy/tool/drain) + agenttest (provider) + agent (public surface)
        Run ──▶ sink ──▶ DrainAndCheck ──▶ agent.CheckStream
        History.Entries → Entry.Message → NewSeededHistory ──▶ Harness #2
    Run unwind (panic): lane → queue → cancel(nil) → close(abort); <-done → close(sink)

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/harness.go` | Modify | D-5: abort channel, hoisted `forwarderDone`, deferred join, forwarder double-select |
| `backend/agent/src/agent/harness_forwarder_panic_test.go` | Create | AD-1 RED + boundedness assertions |
| `backend/agent/src/apptest/{doc,permission,tool,drain}.go` (+ tests) | Create | AD-3 kit + determinism repeat-run test |
| `backend/agent/src/layer3handoff/{doc.go,layer3handoff_test.go,generic_client_guard_test.go}` | Create | AD-4 proof + vocabulary guard |
| `backend/agent/src/agent/example_test.go` | Create | AD-5 examples |
| `backend/agent/src/agent/import_boundary_test.go` | Modify | AD-2: pattern set, per-pattern floor, check 5 |
| `backend/agent/src/agent/hooks_test.go` | Modify | AD-7: `scheduler.go` restored, permanent-release comment |
| `backend/agent/src/agent/doc.go` | Modify | AD-6 pointer section (no `L2C` row) |
| `docs/architecture/milestones/0003-…` | Modify | 7 stale flips + row 2181 + status (re-counted at edit) |
| `openspec/changes/…/decision.md` | Create | AD-6 statement (uncounted) |

## Interfaces / Contracts

AD-3's signatures above are the complete new exported surface. Nothing in `src/agent` gains or loses an exported identifier; `agent-run-driver` and `agent-v1-scope` deltas pin D-5's semantics (the panic still propagates uncontained; it can no longer crash a goroutine the caller does not own).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit | forwarder panic unwind | AD-1 RED ×100 iterations, sentinel identity, both defeat plants, `-race -count=1` |
| Unit | kit determinism | repeat-run structural equality modulo minted IDs; exhaustion latch; compile pins |
| Integration | seven-capability consumer proof | AD-4 sequential stages, every drain `CheckStream`-clean |
| Guard | zero vendor imports / zero I/O / no wall clock | checks 2 (per-pattern floor) + 5 (AST), planted bites |
| Guard | generic-client vocabulary | concatenated-needle scan, planted bite |
| Examples | compile **and run** | four `// Output:` blocks under `make test` |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary is added. Check 5 is AST-only; the pre-existing `go list` and `git diff` machinery changes only its data tables.

## Migration / Rollout

No migration. Single-PR revert removes two directories nothing imports, four additive hunks, and the checkbox flips (proposal's rollback plan).

## Open Questions

- [ ] None blocking. Apply obligations restated: enumerate `Turn`'s sink senders before the D-5 edit (AD-1); re-count the doc 0003 unchecked rows at edit time (AD-6); record every planted bite (AD-1 ×2, AD-2 ×2, AD-4 ×1) in apply-progress with watched-failing evidence.
