# Exploration — `cachicamas-agent-pre-request-hook` (AG-08)

> Milestone AG-08 (Layer 2 Wave 2, milestone 8 of 24; doc 0003, lines 833–900) of doc 0003. SDD change slug: `cachicamas-agent-pre-request-hook`. Artifact store: HYBRID (Engram + OpenSpec). Engram topic key: `sdd/cachicamas-agent-pre-request-hook/explore`. Branch `feat/agent-layer2-wave2-ag08` based at `93077c07` (post-AG-07 PR #167 merge).

## Identity

- **Slug**: `cachicamas-agent-pre-request-hook`
- **Milestone**: AG-08 (Layer 2 Wave 2; doc 0003 § AG-08, lines 833–900)
- **Branch**: `feat/agent-agent-layer2-wave2-ag08`
- **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag08`
- **Store**: HYBRID (Engram topic key `sdd/cachicamas-agent-pre-request-hook/explore` + filesystem `openspec/changes/cachicamas-agent-pre-request-hook/explore.md`)
- **Mode**: automatic (A2 — gatekeeper between phases; no user interruption)
- **Strict TDD**: ACTIVE (carried from AG-04/05/06/07 cycle; openspec/AGENTS.md `## Strict TDD is on`)
- **Review budget**: 1000 lines, `size:exception` pre-authorized for the hook-seam scope (braejan's standing rule from AG-04 / AG-05 / AG-07)
- **Closes**: R-12 (G4's Layer 2 half — prompt-cache prefix stability); seam 1 of v2 § 6 (the only point where the outgoing request still exists as data)
- **Depends on**: AG-07 (walking skeleton + per-call `Turn`). AI-12 (`Request.With(...)` copy-on-write rebuild, already shipped in PR #165)
- **Parallel with**: AG-09 (tool execution contract + scheduler — also wraps `Turn`)
- **Blocks**: AG-20 (the four-hook taxonomy registration surface); CO-24.1 (Layer 3 breakpoint placement consumes the seam)
- **Out of scope**: any concrete hook (cache-breakpoint placement is Layer 3 wiring — doc 0004 CO-24); the other three hook points (AG-20); tool execution / permission / retry / cost (other milestones)

## Context — the AG-07 walking skeleton AG-08 wraps

AG-07 landed the stateless one-turn function `agent.Turn(...)` at `backend/agent/src/agent/loop.go:103-179`. AG-08 wraps the seam AG-07 left open: the place in `Turn` between `buildLoopRequest` (`loop.go:210-223`) and `provider.Stream(...)` (`loop.go:140`).

```text
Turn(ctx, provider, system, transcript, opts, sink)
  │
  ├─► mint runID, turnID, LaneStamper (fresh)
  ├─► emit run_start → sink
  ├─► emit turn_start → sink
  ├─► req, err := buildLoopRequest(opts, system, transcript)        ← loop.go:132
  ├─► *PRE-REQUEST HOOK SEAM* (AG-08 lands here)                     ← AG-08's only new insertion point
  ├─► pCh, pErr := provider.Stream(ctx, req)                        ← loop.go:140
  ├─► for ev in pCh: translate(ev)
  ├─► emit turn_end / run_end → sink
  └─► return (msg, finish, err)
```

The exact lines AG-08 will modify inside `Turn` (lines 132 and 140) and the new field AG-08 will add to `TurnOptions` (line 46). Everything else stays byte-identical to AG-07 — the 5th consecutive "substrate untouched" milestone.

## Substrate AG-08 consumes (Layer 1's copy-on-write rebuild, AI-12)

The mutation mechanism AG-08 will use is AI-12's `Request.With(opts ...RequestOption) (Request, error)` at `backend/agent/src/ai/request.go:325-336`. `Request` is a value type (not a pointer); every region unsupplied by opts carries forward unchanged, and the seed (`r.options`) is a struct of scalars + one slice header whose backing arrays are never mutated in place anywhere in this package. This is the property that makes the "hook cannot mutate the loop's input in place" scenario (AG-08.1 #4) reachable: a hook that calls `req.With(...)` receives a derived value; the caller's original `req` is left observably unmodified (R-REX-001).

Available `Request.With(...)` options AG-08's first hook (per AG-08.1 #1, "adds a system segment") will compose:

| Option | File:line | What it replaces |
|---|---|---|
| `ai.WithSystemInstruction(SystemInstruction)` | `system_instruction.go:148` | the whole system region |
| `ai.WithMessages(ms ...Message)` | `request.go:357` | the whole message sequence |
| `ai.WithTools(ToolSet)` | `request.go:463` | the tool set |
| `ai.WithToolChoice(ToolChoice)` | `request.go:473` | the tool-choice |
| `ai.WithModel(string)` | `request.go:346` | the model identity |
| `ai.WithMaxOutputTokens(int)` | `request.go:439` | the max-tokens budget |
| `ai.WithTemperature`, `WithTopP`, `WithStopSequences` | `request.go:442-455` | sampling options |
| `WithExtension` (AI-12.3) | `request_extension.go` | the per-request escape-hatch region |

For AG-08.1 #1 ("adds a system segment"), the canonical write is:

```go
seg, _ := ai.NewSystemSegment("injected-context")
cur, _ := req.SystemInstruction()
instr, _ := ai.NewSystemInstruction(append(cur.Segments(), seg)...)
next, err := req.With(ai.WithSystemInstruction(instr))
```

— exercising two region rebuilders (`NewSystemSegment` + `WithSystemInstruction`), one rule list (the same `draft.rules()` `request.go:214-272` that `NewRequest` uses, R-REX-005 "derive-time validation is construction's validation"), and `Request.With` returning a fresh value on success or zero on failure.

## Test substrate AG-08 stands on (AI-21, AI-22)

- `agenttest.Provider.Requests() []ai.Request` at `backend/agent/src/agenttest/fake_provider.go:157-161` — the captured-request history AG-08.1 will inspect to prove "the hook's return value is what the provider received" (AG-08.1 #1) and "the captured request is byte-identical to the skeleton's" (AG-08.1 #2). `Requests()[i]` corresponds to script `i` (R-AFP-014).
- `ai.Request.Equal(other) bool` at `request.go:555-627` — region-wise equality (excludes `MessageID` identity). The exact comparison primitive AG-08.2's "tools → system → messages cascade" scenario will use to assert byte-stability across turns (excluding the message region, which grows by append).
- `ai.NewSystemSegment`, `NewSystemInstruction` — for AG-08.1 #1's hook fixture.
- `agenttest.Script` / `Emit` / `Hold` — for scripting the provider's response.
- No new `agenttest` helpers required for AG-08 (carried-forward AG-07 precedent).

## Charter (AG-08, doc 0003:833–900)

| Node | Type | Charter scenarios | Closes |
|---|---|---|---|
| **AG-08.1** — The hook seam `[leaf]` | hook sees + shapes outgoing request; identity default changes nothing; failing hook aborts before I/O; hook cannot mutate input in place | 4 (0003:856-876) | R-12 (the seam's existence); v2 § 6 seam 1 |
| **AG-08.2** — Prefix stability `[leaf]` | byte-stable tool/system regions across turns; message region grows by append; hook deterministic for identical inputs | 2 (0003:886-897) | R-12 (G4 Layer 2 half) |

**Total**: 6 charter Gherkin scenarios in 2 leaves. Forecast ~8–12 spec scenarios after per-scenario expansion (AG-07 precedent: 5 charter → 7 spec + 2 bites = 9 total).

## Key decisions surfaced — D1–D5

| # | Decision | Cited evidence | Recommendation (surface, not pre-pick) |
|---|---|---|---|
| **D1** | **Hook surface shape.** A `func(req ai.Request) (ai.Request, error)` field on `TurnOptions` (callable form, NOT a value type) vs an `interface { OnPreRequest(...) }` (interface) vs `[]Hook` (a chain of hooks) | AG-07 D1 chose function-form surface for `Turn` itself; AG-07 walking skeleton is the thinnest end-to-end path; AG-08's charter is "the hook" (singular). AG-20 is the surface that registers the chain (per doc 0001 G11 / v2 § 6 seam 1 row). A single-hook seam keeps AG-08 narrow; AG-20 widens to the chain | **Surface D1a** — a single callable on `TurnOptions`, type `func(req ai.Request) (ai.Request, error)`. The nil-value is the identity default (returns req unchanged with a nil error). AG-20 chains this; the chain composition is AG-20's, not AG-08's |
| **D2** | **Where the seam wraps `Turn`.** Between `buildLoopRequest` and `provider.Stream` (lines 132-140), with a typed-error return path that closes `sink` and returns `loop, ai.FinishReason(0), typedErr` | `Turn`'s existing pre-stream-failure path (loop.go:140-147) is the precedent: closes `sink`, returns `ai.Message{}, 0, streamErr`. The hook's typed error must take exactly the same path so a hook failure looks identical to a provider pre-stream failure from the caller's side | **Surface D2a** — wrap between line 136 and line 140; the failure path mirrors the pre-stream-failure path. Two-line change inside `Turn` |
| **D3** | **Hook invocation signature.** `func(req ai.Request) (ai.Request, error)` vs `func(ctx context.Context, req ai.Request) (ai.Request, error)` | `ctx` already flows to `provider.Stream(ctx, req)` (D5a, AG-07). The hook may want `ctx` for cancellation, deadlines, or to plumb tracing. Carrying it is cheap; not carrying it forecloses any of those | **Surface D3a** — `func(ctx context.Context, req ai.Request) (ai.Request, error)`. The loop passes its own `ctx`. AG-20's chain composition will inherit the parameter |
| **D4** | **Identity default behaviour.** Nil hook = no-op (returns `req, nil`); zero-value `TurnOptions` = byte-identical output to AG-07's skeleton | AG-07's `func Turn(...)` is byte-stable on identical inputs (R-LSK-002; proven by `TestTurn_TwoSequentialTurnsShareNothing`). AG-08 must preserve this when no hook is installed — this is AG-08.1 #2's "byte-identical to the skeleton's" claim | **Surface D4a** — if the hook field is nil, `Turn` skips the seam and proceeds to `provider.Stream(ctx, req)` unchanged. AG-08.1 #2 asserts this against AG-07's pre-AG-08 captured request (re-running the walking-skeleton script on a pre-AG-08 commit baseline via `git worktree add` is the cleanest proof; alternatively, AG-07's existing tests cover the path implicitly and AG-08.1 #2's assertion is `provider.GetRequests()[0].Equal(buildLoopRequest(...))`) |
| **D5** | **Spec prefix.** `R-LSK-` (loop skeleton — already used by AG-07) vs `R-PRH-` (pre-request hook) vs `R-HK-` (hook) | AG-04/05/06 used two-letter prefixes tied to slug (`R-AEV-`/`R-AMT-`/`R-APE-`). AG-07 used `R-LSK-` (loop-skeleton). Two options: stay under `R-LSK-` (extend AG-07's prefix; AG-08 is the second leaf in the same skeleton batch) or open a new prefix (`R-PRH-` matches the slug cleanly). The doc 0003 charter header says "Closes: R-12; seam 1 of v2 § 6" — R-12 has no prefix-locked home yet | **Surface D5a** — open a new prefix `R-PRH-` (pre-request hook). Distinct from `R-LSK-` because AG-08 is a leaf that closes a different requirement (R-12), and AG-07's R-LSK-001..005 stay a closed quintet. AG-09 / AG-10 / AG-11 will open their own prefixes; AG-20 (the four-hook taxonomy) gets `R-HK-` or `R-FHT-`. Scenarios `S-PRH-NNN` |

**Note for orchestrator**: D1, D2, D3 are load-bearing. Surface to braejan before proposing. D4 inherits directly from AG-07 R-LSK-002. D5 is the smallest decision; if braejan prefers staying under `R-LSK-`, AG-08 becomes the 6th–8th requirement in that file.

## Substrate inventory (line counts)

Total `backend/agent/src/agent/`: **11,006 lines** (post-AG-07 merge at `93077c07`; `loop.go` is 457 lines, `loop_test.go` is 1359 lines).

- AG-08 will MODIFY: `loop.go` only (one new field on `TurnOptions`, one new function `applyPreRequestHook`, one new branch in `Turn`). Forecast +20–40 lines in `loop.go`.
- AG-08 will likely CREATE: `loop_hook_test.go` (or append to `loop_test.go`). Forecast ~300–500 lines for the 6 scenarios + bites + the AG-07 W1 unbuffered-sink carry-forward.
- AG-08 will NOT touch: `event.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `agent_test_helpers_test.go`, `envelope_test.go`, `invariant_pin_test.go`, `protocol_events_test.go`, `message_text_test.go`. **NO** modifications to envelope/descriptor/validator. **NO** register row additions. **NO** envelope variant extension. **NO** new `agenttest` helpers. **5th** consecutive "substrate untouched" milestone.
- AG-08 will NOT change: `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, AG-03 boundary guards.

**Forecast**: 350–600 lines added (smaller than AG-07's 1816). The 1000-line budget with `size:exception` pre-authorized is sufficient with margin.

## Carry-forward list (AG-04/05/06/07 warnings applied to AG-08)

| Source | Finding | AG-08 mitigation |
|---|---|---|
| **AG-07 W1** | "S-LSK-002's drain clause is only indirectly evidenced" (verify-report.md:175). Every AG-07 test uses a **buffered** sink (`make(chan *agent.Event, 16)`) drained *after* `Turn` returns. The concurrent-consumer / back-pressure path is unproven | **MUST add at least one unbuffered-sink test for the hook path** (the AG-08.1 hook + an unbuffered sink + a concurrent consumer goroutine counting events) — closes AG-07 W1. The hook path is the natural place to prove back-pressure because the hook's `With(...)` call is a synchronous gating point before the stream starts, and a goroutine-count assertion via `runtime.NumGoroutine()` before / after proves no stranded producer |
| **AG-07 W6** | "NFR: tests are `package agent_test` external; carry forward as new NFR" (archive-report.md:28) | AG-08 inherits: every AG-08 test lives in `package agent_test` (or another external package). Re-state as NFR-PRH-001 in the AG-08 spec |
| **AG-07 SUGGESTION 4** | "`translate()` could become a method on a `providerEventTranslator` interface to make translation unit-testable in isolation. AG-08's hook seam may introduce this naturally" (verify-report.md:187) | AG-08's hook-wrap point is **not** the translation path; it's between `buildLoopRequest` and `provider.Stream`. Translation is out of AG-08's scope. SUGGESTION 4 stays parked — AG-13's `Harness` may re-introduce it as part of `value-form` |
| **AG-07 W4** | "`mintLoopMessageID` discards two errors... if either ever returned an error, `placeholderMsg` would be the zero value and `ID()` would hand back a zero identity to every bracket" (verify-report.md:178) | AG-08 does not touch `mintLoopMessageID`. Latent, not live. Carries to AG-23 (typed minting bridge already planned) |
| **AG-05 W1** | Vacuous reconstruction helper | AG-08.1 #1 ("hook shapes outgoing request") is the load-bearing property test for the hook — apply the AG-05 W1 bite pattern (mutate-captured-request-attack bites): a hook that ADDS nothing would pass the captured-request byte-equality assertion vacuously. Two bites: (a) hook-doesn't-add-the-segment (the captured request lacks the segment), (b) hook-returns-the-same-request (the captured request is byte-equal but the system region should differ) — both must RED before GREEN |
| **AG-04 W9** | Scenario count drift | State identically in proposal, tasks, apply-progress, verify-report: **6 charter → ~8–12 spec** |
| **Project-wide** | `test-driven-development` skill missing at `~/.claude/skills/` | Cite `openspec/AGENTS.md` `## Strict TDD is on` block directly |
| **External gate** | `TestTurn_CoverageGate` is a `t.Skip` marker; the real gate is `make test/cover` (AG-07 W2) | AG-08 re-states: the gate is enforced by `make test/cover`. AG-08's forecast: loop.go coverage stays ≥ 80% (modest +20–40 lines; AG-07 hit 85.89%) |
| **Substrate ref** | `TestTurn_SubstrateUntouched` hard-codes `8420b2c4` (AG-07 W3) — ref goes stale on merge | AG-08's substrate-untouched test uses the AG-08 BASE_REF env var fallback (already shipped in AG-07 PR #167 W3 fix), with the dynamic merge-base as the default. Adapt the existing test, do not duplicate it |

## Risks & unknowns

### Quantified risks

1. **R1 — Hook signature in `TurnOptions` adds a field (low)**: `TurnOptions` is exported (`loop.go:46`); adding a `PreRequestHook` field is a non-breaking change (zero value = identity default per D4a). No external caller of `Turn` exists yet (AG-08 is the first user of `TurnOptions`'s new field). **Mitigation**: surface D1/D2/D3 in the proposal; the field is the only public-surface change.
2. **R2 — Hook return path must mirror `Turn`'s pre-stream-failure path exactly (low→medium)**: a hook failure must close `sink`, return `ai.Message{}, 0, typedErr` — exactly the lines 140-147 pattern. A naive `return ..., err` would leave `sink` open. **Mitigation**: write the failing-hook test FIRST (AG-08.1 #3) — the test asserts `len(provider.GetRequests()) == 0` AND that draining `sink` unblocks (channel closed). RED before GREEN, per AG-05 W1 carry.
3. **R3 — R-REX-001 / `Request.With` derivation chain (low)**: a hook that calls `req.With(...)` returns a derived value; the loop's `req` value is unchanged (R-REX-001). This is the substrate's promise. **Mitigation**: AG-08.1 #4 ("hook cannot mutate input in place") is the load-bearing test — write a hook that takes `req`, mutates its returned-from-accessors slice (e.g. `msgs := req.Messages(); msgs[0] = ...`), and assert the captured request is byte-equal to the skeleton's. If the substrate's copy-on-clone holds, the assertion passes. RED bites first (mutate the request and assert the loop sends the original).
4. **R4 — Prefix stability across turns needs an append-only assumption (low→medium)**: AG-08.2 #1 says "the message region grows strictly by append". This is true at AG-07 walking-skeleton scope (the loop mints fresh `runID` and `turnID` per turn; the transcript is passed in fresh from the caller) — but the "successive turns with unchanged system material, tools, and hook" precondition is the CALLER's responsibility, not the loop's. **Mitigation**: surface in the proposal that AG-08.2 #1's claim holds *assuming the caller passes the same transcript up to N and then the (N+1)-th with N+1 messages*; the loop itself does not own the message-region append discipline (that is AG-12.1's, in scope for the append-only-history leaf). AG-08.2 #1's test writes the message-region growth externally.
5. **R5 — Hook determinism (AG-08.2 #2) (low)**: a non-deterministic hook (e.g. reads `time.Now()`, reads a random source) would violate the prefix-stability guarantee. **Mitigation**: AG-08.2 #2's test asserts the hook is called with byte-equal input across two calls and returns byte-equal output; a hook function that calls `time.Now()` would still pass this test *if the rest of the request is byte-equal* (the hook's added segment would just contain a different timestamp, which AG-08.2 #1's cascade comparison would catch — the timestamp-bearing system region would differ between turns). The combination of AG-08.2 #1 + #2 closes the determinism property.
6. **R6 — `Request.Equal` excludes `MessageID` identity (low→medium)**: `request.go:555-627` excludes `MessageID` on purpose ("two messages built from identical inputs are two distinct messages — but they are still the same *value*"). For AG-08.2 #1's tools/system cascade byte-equality, `Equal` is correct. For AG-08.2 #1's message region, `Equal` cannot be used directly — the messages carry different identities even when their content is identical. **Mitigation**: the test reads the messages region via `req.Messages()` and compares part-by-part using `Message.Equal` (which excludes identity). For the cascade byte-stability claim, the test asserts (a) tools and system regions `Equal` byte-equal across turns, (b) message regions differ only in count (the new turn's messages region has N+1 messages vs the previous N), with the first N messages content-equal under `Message.Equal`.
7. **R7 — Review budget 1000 lines: low at 350–600 forecast (low)**: AG-08's footprint is smaller than AG-07's. `size:exception` pre-authorized carries forward. No chained PRs needed.
8. **R8 — External-package test posture (low)**: AG-07 W6 — AG-08 inherits. Every AG-08 test in `package agent_test`. NFR-PRH-001.

### Unknowns to verify before design

- **U1**: Whether the AG-08 spec opens under a new prefix (`R-PRH-`, D5a) or stays under `R-LSK-` (extend AG-07). Confirmed by D5a (open new). Surface in proposal.
- **U2**: Whether AG-08 introduces a typed-error type (e.g. `agent.PreRequestHookError`) or reuses the `ai.PreStreamFailure` / `ai.MidStreamFailure` types. The substrate's `ai.PreStreamFailure` (`provider_failure.go:34-93`) wraps a `FailureReport{Category, Cause}`. A hook failure pre-I/O is a pre-stream failure semantically — it is a typed failure of the loop, not the provider. **Mitigation**: surface in the proposal. Two paths: (a) reuse `ai.PreStreamFailure` with `Category: ai.FailureCategory(loop-specific hook name)`; (b) introduce a thin `agent.PreRequestHookError` value type. Path (a) keeps AG-08 within the substrate's error vocabulary; path (b) is more expressive but adds a new exported type. Recommend (a) — the substrate has the typed-error machinery, AG-08 reuses it.
- **U3**: Whether AG-08.1 #1's "adds a system segment" hook operates at the segment level (one or more `Segment` values via `ai.NewSystemSegment`) or at the `SystemInstruction` level (replacing the whole instruction). **Mitigation**: the doc 0003 charter scenario phrasing ("a hook that adds a system segment") suggests segment-level injection — the hook appends one or more `Segment`s. AG-08's first test fixture hook implements the segment-level path. AG-20 widens to instruction-level hooks if needed.
- **U4**: Whether AG-08.2 #1's "tools → system → messages cascade" comparison reuses `ai.Request.CacheBoundaries()` (which already returns the cascade order, `cache_boundary.go:118-120`) for byte-stability proof, or uses `ai.Request.Equal` directly. **Mitigation**: use `Equal` for the byte-stability claim and `CacheBoundaries()` for the cascade-order pin (the cascade order is R-ACB-007's contract). The two together prove both stability AND cascade ordering across turns.

## Recommendation for next phase

**Ready for Proposal: YES.** The substrate is well-understood (AG-07 walking skeleton stable, AI-12 `Request.With` copy-on-write rebuild stable, AI-21 captured-request inspection stable). AG-07's warnings are explicit carry-forwards; AG-08's specific risks are quantified; the open decisions (D1–D5) are listed with cited evidence.

The orchestrator should launch `sdd-propose` next, with the following pre-commit confirmations it should surface to braejan before proposing:

1. **D1 (hook surface shape)** — `func(req ai.Request) (ai.Request, error)` field on `TurnOptions`, NOT an interface, NOT a `[]Hook` chain. AG-20 widens to the chain composition.
2. **D2 (wrap point)** — between `buildLoopRequest` and `provider.Stream` (loop.go:132-140). Failure path mirrors the existing pre-stream-failure path (loop.go:140-147) — close `sink`, return zero.
3. **D3 (hook signature)** — `func(ctx context.Context, req ai.Request) (ai.Request, error)` (with `ctx`).
4. **D5 (spec prefix)** — open a new prefix `R-PRH-` (pre-request hook), distinct from AG-07's `R-LSK-`. Scenarios `S-PRH-NNN`. Alternative: stay under `R-LSK-` and extend the AG-07 quintet.

## Key Learnings

1. AG-08 wraps a precise 4-line insertion point in `Turn`: between `buildLoopRequest` (loop.go:132) and `provider.Stream` (loop.go:140). The 5th consecutive "substrate untouched" milestone — AG-08 modifies only `loop.go` and `loop_test.go` / a new `loop_hook_test.go`.
2. AI-12's `Request.With(...)` (`request.go:325-336`) is the mutation mechanism — copy-on-write rebuild, not in-place mutation. The "hook cannot mutate the loop's input in place" scenario (AG-08.1 #4) is a direct read of R-REX-001.
3. The `agenttest.Provider.Requests() []ai.Request` accessor (`fake_provider.go:157-161`) is the test substrate for both AG-08.1 #1 (hook shapes the request) and AG-08.2 #1 (byte-stable prefix across turns). No new `agenttest` helpers required.
4. The "tools → system → messages cascade" mentioned in AG-08.2 #1 is exactly the order in `Request.CacheBoundaries()` (`cache_boundary.go:130-153`) and the contract of R-ACB-007 (`specs/ai-cache-breakpoints/spec.md:158-178`). AG-08.2's test uses `CacheBoundaries()` for the cascade-order pin and `Request.Equal` for the byte-stability claim.
5. AG-07's W1 (back-pressure path unproven) carries forward as a MUST for AG-08: at least one AG-08 test uses an **unbuffered** sink + a concurrent consumer goroutine, asserting `runtime.NumGoroutine()` returns to baseline. The hook path is the natural place to prove it.
6. AG-07's W6 (external-package test posture) carries forward as NFR-PRH-001: every AG-08 test in `package agent_test`.
7. `Request.Equal` (`request.go:555-627`) excludes `MessageID` identity on purpose — two messages with identical content are NOT `Equal` because their identities differ. AG-08.2 #1's tools/system cascade uses `Equal`; the message-region growth check uses `Message.Equal` + a length diff.
8. Hook-failure typed error: AG-08 likely reuses `ai.PreStreamFailure` (the substrate's typed-error machinery) rather than introducing a new exported type. This keeps AG-08 within the substrate's vocabulary.
9. The hook surface is a single callable on `TurnOptions`, NOT a chain. AG-20 (the four-hook taxonomy registration surface) widens to the chain composition — out of AG-08's scope.
10. The 1000-line review budget with `size:exception` pre-authorized carries forward from AG-04 / AG-05 / AG-07 standing rule. AG-08's 350–600 forecast lands well under the limit even with the strict TDD bites + the AG-07 W1 unbuffered-sink carry-forward.

## Evidence

Every claim cites file:line or doc section:

- **Charter**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:833-900` (AG-08, lines 833-900; 6 charter Gherkin scenarios in 2 leaves, lines 856-897)
- **v2 architecture § 6 seam 1**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:653` (seam 1: pre-request hook; "the only point where the outgoing request still exists as data")
- **v2 architecture § 3.2 (cache cascade)**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:406` (tools → system → messages invalidation cascade, hard cap on breakpoint count)
- **v2 architecture § 7 G4 row**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:698` (G4: Layer 1 places breakpoints, Layer 2 stabilizes — AG-08 closes the L2 half)
- **AG-07 walking skeleton (the seam AG-08 wraps)**: `backend/agent/src/agent/loop.go:103-179` (`Turn`), `:210-223` (`buildLoopRequest`), `:140-147` (pre-stream failure path)
- **AG-07 `TurnOptions` (where the hook field lands)**: `backend/agent/src/agent/loop.go:46-51`
- **Layer 1 — AI-12 `Request.With` (the mutation mechanism)**: `backend/agent/src/ai/request.go:325-336` (`With` derives from `r.options`); `:189-200` (`NewRequest` — same rule list via `draft.rules()`); `:214-272` (`rules()` — R-REX-005 "derive-time validation is construction's validation")
- **Layer 1 — AI-11 cascade order**: `backend/agent/src/ai/cache_boundary.go:118-120` (`Request.CacheBoundaries`); `:130-153` (the cascade walk: tools → system → messages)
- **Layer 1 — R-ACB-007 cascade contract**: `openspec/specs/ai-cache-breakpoints/spec.md:158-178` (markers readable in tools → system → messages order; region vocabulary is closed)
- **Layer 1 — AI-21 captured-request inspection**: `backend/agent/src/agenttest/fake_provider.go:157-161` (`Requests() []ai.Request`); `:138-150` (`consume` captures req exactly when a script is popped — R-AFP-014)
- **Layer 1 — typed pre-stream failure (for hook failure path)**: `backend/agent/src/ai/provider_failure.go:34-93` (`PreStreamFailure(FailureReport)` wraps `Category` + `Cause`)
- **AG-07 substrate preservation (precedent)**: `backend/agent/src/agent/loop.go:19-26` (substrate preservation comment); `openspec/changes/archive/2026-08-13-cachicamas-agent-loop-skeleton/design.md:84-94` (the 21 untouched files table)
- **AG-07 carry-forwards**: `openspec/changes/archive/2026-08-13-cachicamas-agent-loop-skeleton/archive-report.md:22-28` (W1 → AG-08 back-pressure, W6 → AG-08 NFR external posture); `verify-report.md:175` (W1 detail: "every test uses a buffered sink... back-pressure path... is never exercised")
- **CO-24 (Layer 3 consumer of the seam)**: `docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md:680-710` (CO-24.1 breakpoint placement — explicitly out of AG-08's scope; CO-24 *consumes* the seam)
- **AG-07 spec format (precedent for AG-08)**: `openspec/specs/agent-loop-skeleton/spec.md` — `R-LSK-001..005`, `S-LSK-001..007 + 2 bites = 9 total`, Gherkin + RFC 2119
- **AG-07 spec prefix convention**: `R-AEV-` (AG-04), `R-AMT-` (AG-05), `R-APE-` (AG-06), `R-LSK-` (AG-07); AG-08 opens `R-PRH-` per D5a
- **Project AGENTS.md**: `openspec/AGENTS.md` (stack, strict TDD, sub-agent launch contract, hard rules)