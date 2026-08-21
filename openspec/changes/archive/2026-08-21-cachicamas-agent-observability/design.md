# Design: AG-22 — Add the observability boundary

Every citation below was re-read in this worktree (base `main@7dac9ec9`). Layer 1 (`src/ai/`), `go.mod` and `go.sum` are untouchable: `hooks_test.go:2386` (`TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode`) asserts both diffs empty against the resolved base ref. All changes live in `src/agent/`, `src/agenttest/`, `docs/` and `openspec/`.

## Technical Approach

Mirror AI-37's Layer 1 shape at four Layer 2 seams: one constants-and-finalizers file, spans started and ended co-located with the existing Start/End event constructors, failure recorded only as `Failure.Category().String()` (`failure.go:44`), one deferred finalizer per span family so every exit path — error, panic unwind, cancellation, detached — ends the span exactly once (AG-21's leak discipline, telemetry form). Span context threads the existing `ctx` chain (`Harness.Run` → `Turn` → `Schedule` → `executeCall` → `tool.Run`); no signature changes.

## Architecture Decisions

### D-A — Tracer acquisition: injected `Harness.TracerProvider`, never the global getter

**Choice**: One new exported optional field, `Harness.TracerProvider trace.TracerProvider` (the Layer 1 precedent: `openaicompat/client.go:62-71`, R-APC-016). Nil defaults lazily to `noop.NewTracerProvider()` (Harness is zero-value-usable; there is no constructor to normalize in — `harness.go:48-117`). `Harness.Run`/`Harness.Compact` read the field; `Turn`, the scheduler and `runCompaction` acquire the tracer ambiently via `trace.SpanFromContext(ctx).TracerProvider().Tracer(scope)` — `tracetest.Span.TracerProvider()` already deliberately returns its owning Provider (`tracetest.go:324-329`), and the noop chain never panics. Zero signature changes; a bare `Turn`/`Schedule` call records only when the caller's ctx already carries a recording span.

**Rejected**: the § D3-granted global getter `otel.Tracer(...)`. Its cost, measured, not hand-waved: (1) the root `go.opentelemetry.io/otel` package's own measured closure contains `go.opentelemetry.io/auto/sdk` (recorded in `src/ai/import_boundary_test.go:136-144`), and `S-AGP-024` (`agent-package-scaffold/spec.md:84`) forbids any allowlist entry naming an SDK path — the getter is structurally inadmissible under the existing guard family, not merely inelegant; (2) tests would need `otel.SetTracerProvider`, a process-global mutation hostile to this package's pervasive `t.Parallel()` under `-race`; (3) Layer 1's own D-1 already declined it for both reasons.

**Cost of the winning option**: the exported surface gains a telemetry seam, so the spec phase MUST ship a MODIFIED `agent-v1-scope` delta amending `S-AGS-066` (`spec.md:343` — "declares … no telemetry sink"). The shipped enforcing test (`hooks_test.go:2353-2379`) checks only `hooks.go` wall clocks and zero-value `Hooks` inertness and stays green; only the spec sentence is amended: the surface declares no telemetry *sink* — spans reach an injected OTel **API** `TracerProvider` defaulting to no-op; the SDK/exporter sink remains composition-root-only (§ D3). Zero-value `Harness` stays inert (nil → noop, byte-identical behavior).

### D-B — The Layer 2 attribute vocabulary (AG-22.1, recorded as an ADR 0005 § D3 EXTENSION)

Appended to ADR 0005 as a subsection immediately after § D3's Layer 1 allowlist (`0005:246-250`) — an extension of its table per the milestone's own wording (`0003:2067`), not a sibling ADR. The § D3 denylist (`0005:252-255`) is restated verbatim, absolute, never weakened. Span names follow the OTel GenAI semantic conventions where they exist (`invoke_agent`, `execute_tool {name}`, `gen_ai.tool.name`, `gen_ai.tool.call.id`); turn and compaction have no convention equivalent, so their names/keys are minimal inventions under the `cachicamas.` namespace — the milestone's own carve-out exercised, not violated.

| Span (name) | Attribute | Go type | Mirrors (exact event field) |
|---|---|---|---|
| run (`invoke_agent`) | `gen_ai.operation.name` = `"invoke_agent"` | string | convention-required constant on agent spans |
| run | `cachicamas.run.id` | string | `Event.Run()` envelope RunID (`event.go:481`) |
| run | `cachicamas.run.parent_id` (iff delegated) | string | `Event.Parent()` (`event.go:492`); minted by `NewDelegatedRunStart` (`run_events.go:68`) |
| run | `cachicamas.run.outcome` | string | `RunEnd.Outcome().String()` (`run_events.go:202,130`) |
| run | `error.type` (iff outcome=failed) | string | `RunEnd.Failure()` → `Failure.Category().String()` (`run_events.go:206`, `failure.go:44`) |
| turn (`turn`) | `cachicamas.run.id`, `cachicamas.turn.id` | string | envelope `Event.Run()` / `Event.Turn()` (`event.go:481,486`) |
| turn | `cachicamas.turn.outcome` | string | `TurnEnd.Outcome().String()` (`turn_events.go:202,124`) — eight-member vocabulary |
| turn | `error.type` (iff outcome=aborted) | string | `TurnEnd.Failure()` → `.Category().String()` (`turn_events.go:206`; failure present iff Aborted, `:159-173`) |
| tool (`execute_tool {name}`) | `gen_ai.tool.name` | string | `ToolStart.Name()` (`tool_event.go:128`) |
| tool | `gen_ai.tool.call.id` | string | `ToolStart.CallID()` (`tool_event.go:121`) |
| tool | `cachicamas.tool.ordinal` | int64 | `ToolStart.Ordinal()` (`tool_event.go:125`) |
| tool | `cachicamas.tool.outcome` | string | end event's `Outcome() ToolOutcome` (`tool_event.go:331,406,491`) |
| tool | `cachicamas.tool.detached` (iff true) | bool | the detached arm's typed failure (`scheduler.go:502-506`, `typedDetachedCallFailure` `:1132`) |
| tool | `error.type` (iff execution failure) | string | `ToolEndExecutionFailure.Failure()` → `.Category().String()` (`tool_event.go:488`) |
| compaction (`compact`) | `cachicamas.run.id`, `cachicamas.turn.id` | string | envelope; the compaction bracket's own minted turn (`compaction.go:259`) |
| compaction | `cachicamas.compaction.id` | string | `CompactionStarted.CompactionID()` (`compaction_events.go:121`) |
| compaction | `cachicamas.compaction.summary_id` (iff finished) | string | `CompactionFinished.SummaryID()` (`compaction_events.go:203`) |
| compaction | `cachicamas.turn.outcome` | string | closing `TurnEnd.Outcome().String()` (`compaction.go:384`, fail arms via `emitCompactionFailedArm`) |
| compaction | `error.type` (iff aborted) | string | `CompactionFailed.Failure()` → `.Category().String()` (`compaction_events.go:268`) |

Span status: `codes.Error` iff run=Failed / turn=Aborted / tool=execution_failure (incl. detached) / compaction fail-arm; description = category name only; `codes.Ok` otherwise. The compaction bracket gets exactly ONE span (family compaction), not a turn span plus a child — one bracket, one span.

**Layer 1 overlap — decided: defer, do not re-record.** `gen_ai.system`, `gen_ai.request.model`, `gen_ai.request.max_tokens`, `gen_ai.response.finish_reasons`, `gen_ai.usage.*` (×4), `http.response.status_code`, `retry.count`, `stream.event_count` are owned by Layer 1's request span (AI-37, `openaicompat/trace.go`) and are NOT re-recorded at Layer 2 — stated in the vocabulary doc so a future contributor does not re-add them. `error.type` is the one shared KEY (the standard error key reused on different spans, per convention — not a re-record). Layer 1's `chat` span nests under the turn span automatically: `Turn` passes its span-carrying ctx to `provider.Stream`.

### D-C — Deliberately NOT recorded (each with its reason)

1. **Per-delta events** (`message_text.go`, `message_reasoning.go` deltas; `ToolProgress` payloads, `tool_event.go:193`) — raw model/tool output: denylisted content AND unbounded cardinality.
2. **Hook timings** — a concrete telemetry hook is Layer 3's (`agent-hook-taxonomy/spec.md:328`); timing hooks would also require a wall clock `hooks.go` is forbidden to carry (`hooks_test.go:2365-2373`).
3. **Permission argument content** (`permission_events.go`) — tool-argument text, denylisted. Additionally: NO permission span and no permission attributes at all; the gate runs before the tool span opens (see D-D placement).
4. **Usage/cost** (`cost_events.go` CostTurn/CostSession) — owned by Layer 1's `gen_ai.usage.*` at the request span (AI-37); re-recording would duplicate. Deferred, stated on the record.
5. **Compaction instruction and summary content** — the instruction is a prompt, the summary is a completion; both denylisted. Only the opaque `summary_id` is recorded.
6. **`Failure` projections beyond `Category().String()`** — `Unwrap()` error text is denylisted; `Delivery()`/`Retryable()`/`PartialOutput()` are safe but gratuitous.
7. **`SubagentID` and `CompactionSpan` start/end turn IDs** — opaque and safe, but derivable from the event stream by envelope join; minimality (nesting is proven structurally, not by attribute).

### D-D — Detached tool-call span lifecycle

**Choice**: the tool-execution span ends at the scheduler's own wind-down bound, on the `detached==true` arm (`scheduler.go:502-506`), exactly where `typedDetachedCallFailure` + `emitExecutionFailure` already run — status `Error`, `error.type` = the typed cancellation category, `cachicamas.tool.detached=true`. The still-running goroutine owns nothing: it holds no span handle (`runToolWithWindDown`'s inner goroutine "never writes to `results` or `emissions`", `scheduler.go:576-586`), and any spans code it hosts opens later (e.g., a nested child `Harness.Run`) are that lifecycle's own, ended by its own deferred finalizers on its own exits — no orphan by construction. This is AG-21's discipline exactly: every scheduler-visible bound ends its own resource; tests that drive a detached fixture MUST release and join the detached goroutine before `AssertAllEndedOnce()`. **Rejected**: handing the span to the detached goroutine to end "when it really finishes" — that recreates the unbounded-lifetime shape AG-21 removed and makes exactly-once unassertable.

**Span placement rule (all four seams, from the verified exits)**: a span opens iff its bracket's Start event is emitted, via a deferred finalizer registered at open, with the terminal outcome set where each End event is constructed — so panic unwinding and early error returns still end it exactly once. Concretely: `Harness.Run` opens after the pre-identity refusals (`harness.go:444-461` emit nothing → no span); `Turn` opens at `emitStamped(turnStart)` (`loop.go:314`; pre-emission refusals `:274-313` get none, run span additionally on the nil-continuation path `:300-307`); `executeCall` opens after the permission gate proceeds — a gated-out call (`:433-440`) never reaches `ToolStart` and gets NO span; all post-gate exits (start-constructor failure `:456-463`, panic re-raise `:499-501`, detached `:502-506`, `runErr` `:514-518`, four outcome arms `:524-557`) converge on the one finalizer; `runCompaction` opens at the `CompactionStarted` emission (`compaction.go:265-267`) — every one of its twelve returns routes through `emitCompactionFailedArm` or the success tail (`:381-387`); `Harness.Compact` refusals (`:415-422`) emit nothing → no span.

### D-E — Per-field denylist enforcement and the absence proof

Accessors whose values must NEVER reach any attribute, event or status (enumerated, per-field): `ToolStart.Arguments()` (`tool_event.go:131`), `ToolEndSuccess.Result()` (`:328`), `ToolEndResultFailure.Result()` (`:403`), `Failure.Unwrap()` (`failure.go:87`) and any `Failure` projection beyond `.Category().String()`, `CompactionRequest.Instruction`, the compaction summary's content, all message/reasoning delta payloads, all permission argument bytes. `Span.RecordError` is never called in production (AI-37's rule).

**Absence proof** (`observability_denylist_test.go`, mirroring AI-37's 282-line `ai37_denylist_test.go` precedent): mint runtime-CONCATENATED sentinels — prompt, reasoning, tool-arguments, tool-result, permission-argument, compaction-instruction, summary, credential-shaped, header-shaped — drive one full-featured run (tools + delegation fixture + permission + compaction + a failure arm) through one `tracetest.Provider`, then scan `Provider.Corpus()` with `sweep.Scan`. Mandatory guards, in order: (1) `sweep.SelfTest(deny)` positive control — every needle bites its own synthetic corpus BEFORE any clean scan; (2) `AssertAllEndedOnce()`; (3) corpus non-empty AND attribute-count floor ≥ the decided vocabulary's cardinality; (4) event-kind coverage — the drained stream actually contains every covered kind; (5) every needle non-empty. Bite proof: a scratch defeat (temporarily record `ToolStart.Arguments()` as an attribute), watch the scan FAIL naming the vector, revert — recorded in progress notes, S-AGP-027-style.

### D-F — Nesting proof mechanism: extend `tracetest` with parent recording

Verified gap: `tracetest.tracer.Start` (`tracetest.go:182-200`) never reads the parent from ctx, and `Span.SpanContext()` returns `trace.SpanContext{}` (`:272-275`) — parent linkage is NOT recorded today. **Choice**: extend `tracetest` (it lives in `src/agenttest`, already in `allowedTestPrefixes` via prefix `import_boundary_test.go:137` — no guard change, no `src/ai` touch): in `Start`, `parent, _ := trace.SpanFromContext(ctx).(*Span)` recorded on the new Span; add accessor `(*Span).Parent() *Span` (nil = root). Works across Tracer instances and across parent/child harness providers because parenting reads ctx, not the tracer. The AD-4 method-count pin is unaffected (`Parent` is a test-support struct accessor, not a `trace.Span` method). Delegation nesting then proves: child-run span's parent == the delegating tool's span — honestly against AG-19's test fixtures (`delegating_tool_test.go`, `nested_run_test.go`, `siblings_test.go`; no production delegating tool exists), with the fixture wiring the same Provider into the child `Harness.TracerProvider`.

### D-G — Test architecture (strict TDD)

All new tests in `package agent_test` (external — the package's own convention: `import_boundary_test.go:64`, `hooks_test.go`), all `t.Parallel()`-safe because D-A gives each test its own `tracetest.Provider` (no process-global). Files: `observability_nesting_test.go` (scenario 1: parent chain run→turn→tool→child-run, attribute values byte-equal to drained events'), `observability_denylist_test.go` (scenario 2, per D-E), `observability_parity_test.go` (scenario 3: same scripted run traced vs `TracerProvider` nil — event sequences identical, nothing panics, PLUS `provider.Started() > 0` on the traced arm as the non-vacuity floor), `observability_lifecycle_test.go` (table-driven exactly-once over every exit family incl. the detached fixture — release and join before asserting; run with `-count=1`, never trust a cached run).

**RED-first bite list** (each watched failing before its GREEN): (1) tracetest `Parent()` test fails before the tracetest extension; (2) nesting test fails on zero spans before instrumentation; (3) denylist test's corpus-non-empty/attribute-floor guards fail before instrumentation; (4) parity test's `Started() > 0` fails before instrumentation; (5) `TestLayer2_ProductionClosure_...` fails on the first production OTel import until the allowlist entries land — amendment ships in the SAME commit (C4), plus a scratch SDK-import bite recorded then removed (S-AGP-024/S-AGP-027 pattern).

## Import selection (C4 — exact paths, each with its § D3 citation)

`allowedProductionPrefixes` (`import_boundary_test.go:127-130`) gains exactly five entries, mirroring Layer 1's own measured set (`src/ai/import_boundary_test.go:158-173`):

1. `go.opentelemetry.io/otel/trace` — ADR 0005 § D3 table row (`0005:240`, ✅ L2); prefix covers the forced subpackages `trace/noop`, `trace/embedded`, `trace/internal/telemetry`.
2. `go.opentelemetry.io/otel/attribute` — § D3 (`0005:241`).
3. `go.opentelemetry.io/otel/codes` — § D3 (`0005:241`).
4. `go.opentelemetry.io/otel/semconv` — not a § D3 table row: forced transitive closure of `otel/trace` at v1.44.0 (`trace@v1.44.0/auto.go`; recorded-reason entry per R-AGP-003's forced-closure clause; not an OTel module of its own).
5. `github.com/cespare/xxhash/v2` — forced by `otel/attribute`'s hashing (`attribute/internal/xxhash`); same recorded-reason clause.

Deliberately NOT admitted: root `go.opentelemetry.io/otel` (unused under D-A; its closure contains `go.opentelemetry.io/auto/sdk` — S-AGP-024-inadmissible) and `otel/metric` (§ D3 permits it, design does not select it — AG-22 records no metrics). **Closure evidence**: Layer 1's set-equality pin `wantExternalClosure` (`src/ai/import_boundary_test.go:236-247`) is the fresh measurement of exactly these imports at v1.44.0 under the same `go.work`; `tracetest.go:57-60` compiles these paths in this module's test closure today with the current `go.mod` — so no new `require` line and no `go mod tidy` drift. This design could not itself shell out; **apply MUST re-run `go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/agent/...` before and after, diff it, and admit nothing beyond the measured set** — an ADR authorises a path, not its closure.

## Data Flow

    Harness.Run ──tracer.Start──▶ run span ──ctx──▶ Turn ──▶ turn span ──ctx──▶ Scheduler.executeCall
        │ (h.TracerProvider,          │                │ (SpanFromContext         │ post-gate: tool span
        │  nil → noop)                │                │  .TracerProvider())      │ ──ctx──▶ tool.Run
        ▼                            ▼                ▼                          ▼ (delegating fixture:
    RunStart/RunEnd events      TurnStart/End    L1 chat span (AI-37)        child Harness.Run → child
    (span mirrors bracket)      (co-located)     nests via same ctx          run span, parent = tool span)

## File Changes

| File | Action | Description |
|---|---|---|
| `docs/adr/0005-promote-agent-stack-to-own-module.md` | Modify | § D3 extension: Layer 2 span/attribute table (D-B), not-recorded list (D-C), denylist restated verbatim |
| `backend/agent/src/agent/observability.go` | Create | Scope name, attribute-key constants (literal strings, AI-37 pattern), tracer-acquisition helpers, one finalizer per family |
| `backend/agent/src/agent/harness.go` | Modify | `TracerProvider` field; run span in `Run` (post-refusal open, deferred finalizer) |
| `backend/agent/src/agent/loop.go` | Modify | Turn span (+ run span on nil-continuation path) |
| `backend/agent/src/agent/scheduler.go` | Modify | Tool span in `executeCall` (post-gate), detached arm marker |
| `backend/agent/src/agent/compaction.go` | Modify | Compaction span in `runCompaction`; run span in `Compact` |
| `backend/agent/src/agent/import_boundary_test.go` | Modify | Five § D3-annotated allowlist entries (above) |
| `backend/agent/src/agenttest/tracetest/tracetest.go` (+ test) | Modify | Parent recording + `Parent()` accessor (D-F) |
| `backend/agent/src/agent/observability_{nesting,denylist,parity,lifecycle}_test.go` | Create | D-G |
| `docs/architecture/milestones/0003-…` | Modify | AG-22 back-annotation |

Spec deltas (spec phase): NEW `agent-observability-boundary` (prefix `AGO`), MODIFIED `agent-package-scaffold` (guard entries + bite scenarios), MODIFIED `agent-v1-scope` (S-AGS-066 amendment, per D-A).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit | tracetest parent recording | RED-first accessor test in `tracetest_test.go` |
| Unit | Per-seam exactly-once span ending | Table-driven over every exit family, `AssertAllEndedOnce`, `-count=1`, `-race` |
| Integration | Nesting + attribute equality | Full run w/ delegation fixture; parent-chain + per-key equality vs drained events |
| Integration | Denylist absence | Sentinel corpus scan w/ `sweep.SelfTest` positive control + non-vacuity guards |
| Integration | No-tracer parity | Traced vs nil provider, identical event sequence, `Started() > 0` floor |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. (The import guard already shells to the Go toolchain; AG-22 changes only its data tables.)

## Migration / Rollout

No migration. Purely additive; single-PR revert restores the two-entry allowlist and removes every production OTel import (proposal's rollback plan).

## Open Questions

- [ ] None blocking. Apply-phase obligations restated: fresh `go list` closure measurement (this phase had no shell); detached-fixture tests must join before asserting; sdd-tasks re-forecasts size (estimate ≈ 800–1050 non-`openspec/` lines vs the granted 1000 — the extension is pre-authorised if genuinely needed).

## Addendum — closure measured by command (orchestrator, pre-spec)

The design phase flagged that its allowlist rested on Layer 1's set-equality pin rather than a fresh
measurement, because that executor had no shell. The measurement has now been run directly in this
worktree (`backend/agent`, current `go.mod`, resolved through `go.work`). It **confirms the
five-entry selection exactly** and requires no change to it.

```
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' go.opentelemetry.io/otel/trace/noop
```

returns exactly ten packages — `trace/noop` is the widest path D-A imports, so this is the binding
measurement:

| Package | Covered by allowlist entry |
| --- | --- |
| `go.opentelemetry.io/otel/trace` | 1 |
| `go.opentelemetry.io/otel/trace/noop` | 1 (prefix) |
| `go.opentelemetry.io/otel/trace/embedded` | 1 (prefix) |
| `go.opentelemetry.io/otel/trace/internal/telemetry` | 1 (prefix) |
| `go.opentelemetry.io/otel/attribute` | 2 |
| `go.opentelemetry.io/otel/attribute/internal` | 2 (prefix) |
| `go.opentelemetry.io/otel/attribute/internal/xxhash` | 2 (prefix) |
| `go.opentelemetry.io/otel/codes` | 3 |
| `go.opentelemetry.io/otel/semconv/v1.41.0` | 4 (prefix) |
| `github.com/cespare/xxhash/v2` | 5 |

Nothing falls outside the five prefixes, and `go.opentelemetry.io/auto/sdk` does **not** appear.

Two corrections to the design's stated reasoning, neither of which changes the selection:

1. `trace/noop` is **not** in the bare closure of `otel/trace` — it is pulled only because D-A imports
   it directly for the nil-provider default. Entry 1's prefix still covers it. The forced subpackages
   of `otel/trace` proper are `trace/embedded` and `trace/internal/telemetry`.
2. The concrete semconv path is `go.opentelemetry.io/otel/semconv/v1.41.0`, not a bare `semconv`.
   Entry 4 is a prefix and covers it; the version is pinned by `otel/trace@v1.44.0`, so a future OTel
   bump can move it and the entry must stay a prefix, never an exact path.

The rejection of the root global getter is likewise confirmed by measurement rather than inference —
its closure is 25 packages and does contain `go.opentelemetry.io/auto/sdk`, alongside `otel/metric`,
`otel/baggage`, `otel/propagation`, `go-logr/logr` and `go-logr/stdr`.

`git status --short backend/agent/go.mod backend/agent/go.sum` is empty: every selected path already
resolves under the committed module graph, so C2's freeze guard
(`TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode`) is satisfied.

Apply still re-runs the measurement over `./src/agent/...` before and after its own edits and diffs
the two, per the design's standing instruction. This addendum discharges the *pre-implementation*
half of that obligation only.
