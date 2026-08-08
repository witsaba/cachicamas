# Design: Add the observability boundary (AI-37)

> **Change**: `cachicamas-ai-observability` · Proposal decisions **D-1…D-6 are binding**; this design specifies the *how*.
> **Version target**: otel **v1.44.0** — measured by the orchestrator under the repo-root `go.work` (`go list -m all` from `backend/agent` resolves `go.opentelemetry.io/otel v1.44.0`, `…/otel/trace v1.44.0`, `github.com/cespare/xxhash/v2 v2.3.0`, inherited from `backend/database_administrator/go.mod:10-14`; Engram #2706). All interface facts below were read from the module cache at `~/go/pkg/mod/go.opentelemetry.io/otel/trace@v1.44.0/` and `…/otel@v1.44.0/attribute/`, and independently confirmed by the orchestrator's probe.

## Technical Approach

One span per logical request. `openaicompat.Config` gains an optional injected `trace.TracerProvider` (D-1); `New` defaults it with `noop.NewTracerProvider()` (module cache `trace@v1.44.0/noop/noop.go:36-38`) so every span call site is unconditional. The span starts in `Stream` immediately before `retry.Loop` (`stream.go:215`) — *after* the R-ATS-002 validation gate at `stream.go:202-213`, whose contract says nothing is observable before validation clears (`stream.go:196-200`); a span is an observable side effect under a recording provider, so validation failures produce no span, and the spec delta states the span covers the wire request. The span ends exactly once: on `Stream`'s two post-loop error returns (`:216-218`, `:226-228`) via a pre-handover helper, or — once `go run(...)` launches at `:231` — in a new deferred finalizer inside `run`. A hand-rolled recording tracer in `backend/agent/src/agenttest/tracetest/` (D-2) proves everything; the AI-36 sweep (`agenttest/sweep/sweep.go`) proves the denylist by absence with four non-vacuity guards (D-4). The forward guard is rewritten per D-6.

## Architecture Decisions

### AD-1 — Pin v1.44.0, the workspace-resolved version. Do not fight `go.work`.

**Choice**: `backend/agent/go.mod` requires `go.opentelemetry.io/otel v1.44.0` and `go.opentelemetry.io/otel/trace v1.44.0` (+ `github.com/cespare/xxhash/v2 v2.3.0 // indirect`) — the versions `database_administrator` already pins and the workspace build list already resolves (orchestrator probe, Engram #2706).
**Alternatives**: (a) `go get …@latest` → v1.45.0 — would force the *whole workspace* (including the sibling's SDK stack) up a version inside a milestone whose charter is a boundary, and widen the `go.work.sum` delta; (b) no explicit version → whatever MVS picks, unreviewable.
**Rationale**: under `go.work` the build list is unified across three modules; requiring the version the workspace already resolves makes the `go.mod` literal, the `go list` closure, and the running build agree, and keeps the `go.work.sum` delta minimal. `R-AGM-003` (`openspec/specs/agent-module-scaffold/spec.md:64-66`) requires the resulting `go.work.sum` committed. Apply runs `go get go.opentelemetry.io/otel/trace@v1.44.0 go.opentelemetry.io/otel@v1.44.0` from `backend/agent/`.
**Guard interplay**: the workspace module list also names `go.opentelemetry.io/otel/sdk v1.44.0` and `go.opentelemetry.io/auto/sdk v1.2.1` (sibling requirements). These are **module-graph** facts, invisible to the guard, which scans the **package** closure via `go list -deps -test` (`import_boundary_test.go:189-190`). The bite proof (WU-3) must not confuse the two.

### AD-2 — Guard rewrite: exact package-set equality, version-fragile **on purpose** (D-6).

**Choice**: `TestLayer1_ModuleHasNoDependencies_ZeroRequires` (`import_boundary_test.go:147-162`) is renamed `TestLayer1_DependencySet_ExactRequiresAndClosure` and asserts three things:
1. **Exact require set** — decode `go mod edit -json` (run in the module dir; reports `go.mod` as written, immune to workspace unification) and assert requires == `{go.opentelemetry.io/otel v1.44.0 (direct), go.opentelemetry.io/otel/trace v1.44.0 (direct), github.com/cespare/xxhash/v2 v2.3.0 (indirect)}` with zero `replace`. No new dependency for parsing: `encoding/json` + `os/exec`, the guard's existing pattern.
2. **Exact non-stdlib package closure by set equality** — external entries of `listNonStdlibDeps(layer1Pattern)` (`:188-213`), sorted, equal this literal (orchestrator-measured at v1.44.0):
   `github.com/cespare/xxhash/v2`, `go.opentelemetry.io/otel/attribute`, `go.opentelemetry.io/otel/attribute/internal`, `go.opentelemetry.io/otel/attribute/internal/xxhash`, `go.opentelemetry.io/otel/codes`, `go.opentelemetry.io/otel/semconv/v1.41.0`, `go.opentelemetry.io/otel/trace`, `go.opentelemetry.io/otel/trace/embedded`, `go.opentelemetry.io/otel/trace/internal/telemetry`, `go.opentelemetry.io/otel/trace/noop`.
3. **Forbidden table unchanged** (`:88-90`).
**Version fragility is a stated property, not an accident**: a version bump rotates the vendored semconv path (v1.44.0 embeds `semconv/v1.41.0` via `trace@v1.44.0/auto.go:23`; v1.45.0 embeds `semconv/v1.43.0`) and breaks equality. That is the intended behaviour — a require bump must be a deliberate, reviewed act. The guard's source carries the **update procedure**: *on a deliberate, ADR-gated bump, re-run `go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' github.com/cachicamas/backend/agent/...`, diff against the literal, verify no `sdk`-named path appeared, and update the require-set and closure literals in the same commit.* Module-prefix equality with a package-count pin was rejected: a count survives a swap of one package for another (e.g. `trace/noop` out, `otel/metric` in nets zero) — exactly the silent drift D-6 exists to stop.
**Allowlist entries (narrower than the proposal's sketch)**: add five prefixes to `allowedNonStdlibPrefixes` (`:103-105`), each citing ADR 0005 § D3: `go.opentelemetry.io/otel/trace`, `go.opentelemetry.io/otel/attribute`, `go.opentelemetry.io/otel/codes`, `go.opentelemetry.io/otel/semconv`, `github.com/cespare/xxhash/v2`. **Deliberately NOT the root `go.opentelemetry.io/otel`**: the root package is never imported (D-1), and with these entries `otel` (root), `otel/metric`, `otel/baggage`, `otel/propagation`, and `go.opentelemetry.io/auto/sdk` all fail deny-by-default with no closure pin needed. The closure pin is *still* load-bearing, not redundant, because `isAllowed` is prefix-based (`:173-180`): the entry `…/otel/semconv` admits **every** `semconv/vX.Y.Z` directory, and `…/otel/trace` admits any future `trace/*` subpackage — set equality is what pins those. `semconv` and `xxhash` are not § D3-table paths; each entry's comment records the closure-authorisation reasoning: authorising an import path necessarily authorises its transitive closure, and neither is an OTel *module*, so § D3's "any additional OTel module requires its own ADR" clause (`docs/adr/0005-promote-agent-stack-to-own-module.md:257-261`) is not engaged.

### AD-3 — Recording tracer embeds the `noop` types (D-2, proposal trim lever (a) adopted as primary).

**Choice**: `tracetest.Provider`, `tracetest.tracer`, `tracetest.Span` embed `noop.TracerProvider`, `noop.Tracer`, `noop.Span` respectively and override every current method to record. The noop package documents this exact use: "This implementation can be embedded in other implementations… will mean the implementation defaults to no operation for methods it does not implement" (`noop/noop.go:10-12`), and each noop type already embeds its `embedded.*` marker (`noop.go:33,46,78-79`), satisfying the forward-compatibility requirement (`embedded/embedded.go:25,35,45`).
**Alternatives**: embedding `embedded.*` directly — rejected: a minor-version method addition then breaks compilation (`span.go:19-21` warns "Methods may be added to this interface in minor releases"), which is churn without a safety gain here because AD-4's method-count pin makes any silent inheritance loud.
**Rationale + the silent-no-op hazard**: an inherited (un-overridden) future method would silently *not record* — a corpus blind spot (risk 7.6). Mitigation is AD-4's pin, which fails the build of trust the moment the interface grows.

### AD-4 — Interface-surface pin: `reflect` method-count assertion in `tracetest`'s own tests.

**Choice**: a test asserts `reflect.TypeOf((*trace.Span)(nil)).Elem().NumMethod() == 11` (ten methods + unexported `span()`; for interface types reflect counts unexported methods), `trace.Tracer == 2`, `trace.TracerProvider == 2`, against the v1.44.0 surface enumerated in § Interfaces below. A version bump that adds a method fails this pin, forcing a reviewed recorder update — the same "deliberate event" posture as AD-2.

### AD-5 — Span-end topology: single owner per phase, finalizer runs before channel close.

**Choice**: exactly one of two owners ends the span, partitioned by the handover at `stream.go:231`:
- **Pre-handover** (`Stream` still owns it): the `retry.Loop` error return (`:216-218`) and the content-type refusal (`:226-228`) call `endSpanPreHandover(span, retries, err)` — sets `retry.count`, `error.type`, status, `End()`.
- **Post-handover** (`run` owns it): `run` gains a `span trace.Span` parameter and registers the span finalizer defer **after** the existing `defer close(out)` (`stream.go:607`) and drain-before-close defer (`:611`) in source order, so LIFO runs it **first**: finalizer → body drain → `close(out)`. Consequence: when a consumer observes channel close, the span has already ended — the leak test needs no synchronization. Trade-off, stated: measured span duration excludes the silent body drain; the terminal event was already emitted, so nothing user-visible is excluded.
**Alternatives**: finalizer registered first (runs last, after close) — rejected: creates a close-observed-but-span-not-ended race that every leak assertion would have to poll around.
**Exactly-once proof**: `tracetest.Span.End` increments an `endCount` (never panics); helpers `Provider.AssertAllEndedOnce(t)` assert every started span has `endCount == 1`. A table test drives every terminal path: success `[DONE]` (`stream.go:653-661`), completion-build failure (`:656`), in-band error frame (`:669-684`), malformed chunk (`:688`), applyChunk failure (`:700`), emit-race cancellation (`:704-736`), feed error (`:753`), EOF finish error (`:760`), EOF incomplete (`:766`), mid-stream ctx cancellation (`:769-792`), read error (`:793`), plus pre-handover: retry-exhausted, non-retryable, cancelled-in-loop, content-type refusal. This is the span analogue of the package's goroutine-leak discipline.
**Terminal state plumbing**: `run` keeps a local `spanOutcome{completion ai.Completion; haveCompletion bool; failure *ai.Failure; eventCount int}`; return sites fill it; the finalizer maps it to attributes + status once. `emitFailure` (`stream.go:863`) changes to **return the `*ai.Failure` it emitted** so its seven call sites can record `failure` into `spanOutcome`; the in-band (`failureFromErrorFrame`, `stream_failure.go:72`) and cancellation (`midStreamFailureFrom`, `stream_failure.go:196`) sites already hold the failure.
**`span.RecordError` is never called in production code** — `ai.Failure` causes can embed body excerpts by design (`nonStreamContentType`, `stream.go:295-299`), and "any raw provider response body" is denylisted (ADR 0005 `docs/adr/0005-…:252-255`). Status carries `codes.Error` with the **category name** as description (closed nine-member vocabulary, `provider_failure.go:134-139,155-167`) — never `err.Error()`. Success: `codes.Ok`, empty description (`SetStatus` doc: description included only for error codes, `span.go:57-61`).

### AD-6 — `retry.Loop` returns the retry count on every exit (D-3c).

**Choice**: `Loop(ctx, body, cfg, executeOnce) (*http.Response, int, error)` — the new `int` is **retries performed** = `attempt - 1` against the 1-based counter (`retry.go:78`), returned at all six return sites: success `:81` → `attempt-1`; non-retryable `:87` → `attempt-1`; budget exhausted `:90` → `attempt-1` (= MaxAttempts); ctx-cancelled `:93` and sleep-interrupted `:107` → `attempt-1`; fallback `:111` → `totalAttempts-1`. An unretried success reports `0`. `Config` (`retry.go:37-45`) and `AttemptReport` (`retry.go:50-53`) are **unchanged**.
**R-AIS-041…044 compatibility, stated precisely**: those requirements are behaviour-level (`ai-stream-lifecycle/spec.md:523-644,953-955`) — wire-request counts (`R-AIS-041`/S-1), attempt count reachable **from the error chain** on exhaustion (`R-AIS-041`/S-4, `R-AIS-043`/S-2, satisfied by the unchanged `AttemptReport`), byte-identical replay, backoff bounds. None names Go arity. The change is Go-source-incompatible (both callers — `stream.go:215` and `retry_test.go` — update in the same work unit) but **behaviour-compatible**: no delay, ordering, classification, or error-chain change; every landed scenario re-runs green unmodified.
**Rejected**: observer callback on `Config` (proposal D-3c: eight fields already, adds an ordering question for a value a return carries).

### AD-7 — `gen_ai.system` value: the dialect constant `"openai_compat"`.

**Choice**: unexported `const genAISystem = "openai_compat"` in `trace.go` — names the dialect the package speaks (D-3a), spelled in semconv value style (lowercase, underscore).
**Alternatives**: `"openai"` (the semconv well-known value) — rejected: claims the *vendor*, false for OpenRouter/gateway endpoints the same adapter serves. Apply must not re-decide this value.

### AD-8 — Span identity: name `"chat"`, kind `Client`, tracer name = package import path.

**Choice**: span name is the constant `"chat"` (GenAI operation name; cardinality 1, trivially denylist-safe); start options `trace.WithSpanKind(trace.SpanKindClient)` (`span.go:131-133`) plus `trace.WithAttributes(...)` for the request-time attributes (`tracer.go:30-32` recommends attributes at start for samplers). The tracer is derived **once** in `New`: `cfg.TracerProvider.Tracer("github.com/cachicamas/backend/agent/src/ai/openaicompat")` (instrumentation-scope naming per `provider.go:35-38`), stored as `Client.tracer trace.Tracer` beside the existing fields (`client.go:63-67`).
**Rejected**: `"chat <model>"` name — model is allowlisted metadata but a per-model span name multiplies cardinality for zero attribute value the `gen_ai.request.model` attribute doesn't already carry.

## Interfaces / Contracts (verified at v1.44.0, module cache)

Every method the recorder must provide (source: `trace@v1.44.0/span.go:22-78`, `tracer.go:17-37`, `provider.go:25-59`; orchestrator-confirmed):

```go
type TracerProvider interface { embedded.TracerProvider; Tracer(name string, options ...TracerOption) Tracer }
type Tracer         interface { embedded.Tracer; Start(ctx context.Context, spanName string, opts ...SpanStartOption) (context.Context, Span) }
type Span interface {
    embedded.Span
    End(options ...SpanEndOption)
    AddEvent(name string, options ...EventOption)
    AddLink(link Link)
    IsRecording() bool
    RecordError(err error, options ...EventOption)
    SpanContext() SpanContext
    SetStatus(code codes.Code, description string)
    SetName(name string)
    SetAttributes(kv ...attribute.KeyValue)
    TracerProvider() TracerProvider
}
```

`tracetest` contracts: `Span.IsRecording()` returns true until ended; `Span.TracerProvider()` returns the owning `*Provider` (never nil — a working provider, per `span.go:75-77`); `Tracer.Start` mirrors `noop.Tracer.Start`'s context wiring via `trace.ContextWithSpan`; `RecordError` renders `err.Error()` **at record time** into a recorded event (`span.go:47-51`: it records an exception *event* and does not change status — which is exactly why the corpus must walk events). `Provider` is mutex-guarded; `Started()`, `Ended()`, `AssertAllEndedOnce(tb)`, and `Corpus() []byte` are the read surface.

**Corpus builder walk** (risk 7.6): for every started span — name; status code + description; every attribute key and `Value.String()`; every event name, every event-attribute key and `String()`; every `RecordError` string; every `AddLink`'s link-attribute keys and `String()` values. **Never `%#v`, never `%v` over recorder structs** — the AI-41 lesson: a canary in a pointer-shaped field renders under `%#v` as a bare hex address and proves nothing (doc 0002:22). (Accessor substitution, recorded at apply time: `Value.Emit()` — this design's original citation, `attribute@v1.44.0/value.go:368` — is deprecated at v1.44.0; the shipped code calls `Value.String()` instead, the API's current accessor. Both render through the API's own accessor rather than a struct dump, so the substitution changes no property this design relies on.)

## Attribute construction table (twelve § D3 attributes, `docs/adr/0005-…:246-250`)

| # | Key (exact) | Constructor | Source (verified) | AI-37 disposition |
|---|---|---|---|---|
| 1 | `gen_ai.system` | `attribute.String` (`kv.go:62`) | new `genAISystem` const (AD-7) | **Record** at start (carrier added, WU-6) |
| 2 | `gen_ai.request.model` | `attribute.String` | `req.Model()` `request.go:362` | **Record** at start |
| 3 | `gen_ai.request.max_tokens` | `attribute.Int` (`kv.go:32`) | `req.MaxOutputTokens()` `request.go:504` | **Record** at start, only when present (presence-typed) |
| 4 | `gen_ai.response.finish_reasons` | `attribute.StringSlice` (`kv.go:67`), one element | `completion.FinishReason()` `completion.go:87` → `String()` `finish_reason.go:144` | **Record** at success terminal |
| 5 | `gen_ai.usage.input_tokens` | `attribute.Int64` (`kv.go:42`) | `completion.Usage()` `completion.go:92` → `.Input.Count()` `usage.go:63` | **Record** at success terminal, when present |
| 6 | `gen_ai.usage.output_tokens` | `attribute.Int64` | `.Output.Count()` `usage.go:63` | **Record**, when present |
| 7 | `gen_ai.usage.cache_read_tokens` | `attribute.Int64` | `.CacheRead.Count()` | **Record**, when present |
| 8 | `gen_ai.usage.cache_write_tokens` | `attribute.Int64` | `.CacheWrite.Count()` | **Record**, when present (`Reasoning` excluded — a breakdown of Output, `usage.go:124-134`) |
| 9 | `http.response.status_code` | `attribute.Int` | success: `resp.StatusCode` at `stream.go:215-226`. Failure: only `StatusClass()` survives (`provider_failure.go:484`; `failure_map.go:45-47` discards the code) | **Split (D-3b)**: record exact code on the success path; **omit** on terminal failure, justification recorded in source + spec; `error.type` covers that path |
| 10 | `retry.count` | `attribute.Int` | new second return of `retry.Loop` (AD-6) | **Record** always (carrier added) |
| 11 | `stream.event_count` | `attribute.Int` | new counter in `run` (AD-5), incremented at each confirmed send: every `emit(...) == true` (`stream.go:704`) and each bounded-wait select that takes its send branch (`:718-731`, `:779-788`) | **Record** at terminal (carrier added) |
| 12 | `error.type` | `attribute.String` | `failure.Category()` `provider_failure.go:432` → `String()` `:134-139`, closed vocabulary `:155-167` | **Record** on every failure terminal |

Keys are literal constants in `trace.go`, spelled once (AI-37.2 test 2, doc 0002:2221); no key is ever assembled at a call site.

## Data Flow

```
Stream(ctx, req)
  ├─ validate (IsZero → Translate → ctx.Err)      stream.go:202-213   ── no span yet (R-ATS-002)
  ├─ span := c.tracer.Start(ctx,"chat",kind=Client,
  │        attrs: system, model, max_tokens)      before :215
  ├─ resp, retries, err := retry.Loop(...)        :215 (AD-6 signature)
  │    └─ err ─→ endSpanPreHandover: retry.count, error.type, status=Error(category), End
  ├─ content-type refusal :226 ─→ endSpanPreHandover (same, + no status_code — code IS in hand
  │        on this path via resp.StatusCode: record it, then error.type=malformed_response)
  └─ go run(ctx, resp, out, span) :231
       run: defer close(out) :607 → defer drain :611 → defer finalize(span,&outcome)  [runs FIRST]
         finalize: retry.count, status_code (success), finish_reasons, usage, event_count,
                   error.type/status — then span.End()
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/go.mod` | Modify | +2 direct requires @v1.44.0, +1 indirect (AD-1) |
| `backend/agent/go.sum` | Create | generated |
| `go.work.sum` (repo root) | Modify/Create | generated; committed per `R-AGM-003` |
| `backend/agent/src/ai/import_boundary_test.go` | Modify | AD-2: five allowlist entries; require-set + closure-equality pin replaces zero-require pin (`:147-162`); forbidden table `:88-90` untouched |
| `backend/agent/src/agenttest/tracetest/tracetest.go` | Create | recording Provider/tracer/Span embedding noop (AD-3), Corpus builder, AssertAllEndedOnce; imports: `context`, `sync`, `otel/trace`, `otel/trace/noop`, `otel/attribute`, `otel/codes` — **not** `testing` (sweep precedent, `sweep.go:5-12`) |
| `backend/agent/src/agenttest/tracetest/tracetest_test.go` | Create | recorder behaviour + AD-4 method-count pin |
| `backend/agent/src/ai/openaicompat/trace.go` | Create | key constants, `genAISystem`, span start helper, `endSpanPreHandover`, `finalizeSpan`, attribute mapping |
| `backend/agent/src/ai/openaicompat/client.go` | Modify | `Config.TracerProvider trace.TracerProvider` (below `HTTPClient`, `client.go:57`), noop default in `New` on the `:116-118` line-shape, `tracer` field on `Client` |
| `backend/agent/src/ai/openaicompat/stream.go` | Modify | span start before `:215`; two pre-handover end sites; `run(ctx, resp, out, span)` + `spanOutcome` + finalizer defer; event counter; `emitFailure` returns `*ai.Failure` (`:863`) |
| `backend/agent/src/ai/internal/retry/retry.go` | Modify | AD-6 signature, six return sites |
| `backend/agent/src/ai/internal/retry/retry_test.go` | Modify | count asserted at every exit |
| `backend/agent/src/ai/openaicompat/ai37_*_test.go` | Create | attribute table, span-lifecycle table, denylist absence, no-op equivalence (milestone-numbered convention) |
| `openspec/changes/cachicamas-ai-observability/specs/**` | Create | per proposal § 8 (sdd-spec owns) |

## Non-vacuity plan (D-4) — canaries, corpus, overlay RED

**Six canary plant sites** (all runtime-assembled, e.g. `"CANARY-" + "PROMPT-" + nonce` — never a contiguous literal, `credential_scan_test.go:39-44`; all land in **string-valued** carriers):
1. **prompt** — the user message content string of the `ai.Request` the test builds.
2. **reasoning** — a reasoning-delta string inside a scripted SSE frame served by the stub transport.
3. **tool-call arguments** — the arguments JSON string of a scripted tool-call frame.
4. **tool-result** — the content string of a tool-result message in the request conversation.
5. **credential** — the bearer token in `Config.Credential` the client is constructed with.
6. **header + raw body** — a custom response header value set by the stub transport, and a distinct marker string inside the raw SSE body bytes.

**Four guards around the clean scan** (`sweep.Scan` returns `("", false)` on an empty corpus by construction, `sweep.go:57-59`):
1. `sweep.SelfTest(deny)` before every clean scan (`sweep.go:85-93`).
2. **Corpus-non-empty**: `len(corpus) > 0`, started-span count ≥ 1, recorded attribute count ≥ the allowlist cardinality expected for the run — the residual AI-36 recorded for AI-37 by name (doc 0002:24).
3. **Coverage**: the drained `Recording` (`agenttest/stream_kit_record.go:63`) contained text, reasoning, tool-call, completion and error events — the absence is proven "over a run that used all of them" (doc 0002:2228).
4. **Recorded overlay RED**, worktree never mutated:
   - Copy `trace.go` to the session scratchpad as `trace_leaking.go`, adding one line in the attribute mapper that copies message text into a span attribute.
   - Write `overlay.json` in the scratchpad — exact shape:
     ```json
     {"Replace": {"/abs/worktree/backend/agent/src/ai/openaicompat/trace.go": "/abs/scratchpad/trace_leaking.go"}}
     ```
   - `cd backend/agent && go test -overlay=/abs/scratchpad/overlay.json -run TestAI37_DenylistAbsence ./src/ai/openaicompat/` → **must FAIL** (the prompt canary is found); record the red output in the work-unit evidence.
   - Delete the scratchpad files; re-run without `-overlay` → green. `git status --porcelain` stays empty throughout — the overlay swaps the file at build time only.

Failure messages name vector, span, and attribute key only — never the matched bytes (AI-36 rule, `sweep.go:38`).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Guard | require set, closure equality, forbidden rows | AD-2 test + WU-3 recorded bite proof (scratch `sdk` and `exporters` imports each fail; red recorded, dropped) |
| Unit | recorder behaviour, method-count pin | `tracetest_test.go`, table-driven |
| Unit | `retry.Loop` count at all six exits | `retry_test.go` extension; existing assertions unchanged |
| Unit | per-attribute table (12 rows), exact keys and values | recording provider + stub transport; each § D3 key one row (per-attribute, not whole-set — names the regressing attribute) |
| Unit | span ends exactly once on every terminal path | AD-5 table over all 15 paths + `AssertAllEndedOnce` |
| Integration | denylist absence + 4 guards | WU-8 |
| Integration | no-tracer equivalence | same scripted stream twice — default (noop) vs recording provider — drained sequences equal; plus grep-shaped assertion that no adapter-side nil check exists (doc 0002:2234) |

## Threat Matrix

N/A — no routing, VCS/PR automation, or executable-file classification boundary. The only subprocess surface is the guard's existing `go list` exec pattern (`import_boundary_test.go:189`) extended with `go mod edit -json`: same trust domain (local Go toolchain), constant argv, no user input reaches it.

## Work-unit / commit plan (strict TDD; forecast lines = authored adds+dels, generated excluded)

| # | Commit (WU map) | RED first | Content | Forecast |
|---|---|---|---|---|
| C1 | WU-1+WU-2 — deps + guard | new guard test written first, RED against empty `go.mod` | `go get @v1.44.0`; AD-2 rewrite + allowlist | ~140 (+~30 generated) |
| C2 | WU-3 — bite proof | is itself the RED | scratch SDK + exporter imports fail; red recorded in evidence, violations dropped, suite green | ~0 |
| C3 | WU-7 — tracetest | recorder tests RED first | AD-3/AD-4 package + tests + Corpus | ~350 |
| C4 | WU-6c — retry count | exit-count assertions RED | AD-6 signature, both callers | ~140 |
| C5 | WU-4 — span seam | lifecycle table RED | Config field, noop default, start/end sites, `run` finalizer, event counter, `emitFailure` return | ~200 |
| C6 | WU-5+WU-6a/b — attributes | per-attribute table RED | `trace.go`, `genAISystem`, success status code | ~400 |
| C7 | WU-8 — denylist absence | overlay RED (recorded) | six canaries, four guards | ~280 |
| C8 | WU-9 — equivalence | equivalence test RED against a deliberately-broken tracer wiring? No — pin-style; RED via overlay variant that panics on nil provider | dual-drain equality | ~150 |
| C9 | WU-10 — spec deltas | — | per proposal § 8 | ~280 spec |

Code total ~1,660 (band 1,400–2,100 holds). `size:exception` pre-granted, auto-raising. Guard lines (re-forecast by `sdd-tasks` authoritatively): `Decision needed before apply: No` · `Chained PRs recommended: No` · `400-line budget risk: High`. Each commit leaves the module green (`make test -race`, `make lint`, `make build`); C1 is the one commit where guard and deps must land together (either alone leaves main red — proposal § 10).

## Risk register

| # | Risk | Mitigation |
|---|---|---|
| R1 | go.work version drift: `go get` lands something other than v1.44.0, or a later sibling bump moves the workspace | AD-1 pins `@v1.44.0` explicitly; AD-2's require-set pin reads `go.mod` via `go mod edit -json` (workspace-immune) while the closure pin reads the workspace build — a divergence between the two fails one of the two assertions loudly |
| R2 | Vacuous absence (the milestone's characteristic failure) | D-4's four guards; canaries only in string-valued carriers; corpus rendered via `Value.String()` only (substituted for the deprecated `Value.Emit()`; see the note above) |
| R3 | Closure pin fragility surprises a future bump | Stated intentional (AD-2) + written update procedure in guard source |
| R4 | Recorder misses a leak path (links, status description, future methods) | Corpus walks links and status description explicitly; AD-4 method-count pin turns interface growth into a build-of-trust failure |
| R5 | Span leak / double-end on an unenumerated exit path | AD-5's single-finalizer topology + 15-path table + `AssertAllEndedOnce`; finalizer-before-close ordering removes the test race |
| R6 | `retry.Loop` change regresses R-AIS-041…044 | AD-6: behaviour-neutral by construction; `AttemptReport` untouched; all landed scenarios re-run unmodified |
| R7 | `S-APC-015`/`R-APC-001` amendments regress AI-25 proofs | Full MODIFIED restatements; `S-APC-001…016` re-run green (proposal 7.4) |
| R8 | Forecast overrun (AI-35/AI-36 precedent) | exception pre-granted; remaining trim levers: fold WU-9 into WU-8 (−~70), whole-set equality (−~120, least preferred) |

## Migration / Rollout

No migration. Rollback per proposal § 10; C1 reverts as one commit.

## Open Questions

None blocking. `sdd-tasks` decides only test-file naming granularity within the `ai37_*_test.go` convention.

---

> **Post-apply reconciliation note (2026-08-08, doc-only round).** Two textual substitutions were mirrored into this design document to keep it consistent with the shipped code and `tracetest.go`'s own doc comment, which already had them right: the corpus-builder walk (§ Interfaces / Contracts) and the attribute construction table's `http.response.status_code` disposition both cite `Value.String()` — `Value.Emit()` was deprecated at v1.44.0 (the API's accessor substitution recorded in `verify-report-final.md`'s W-A/S-2 and `apply-progress.md`'s final documentation-reconciliation section). Design documents are not archive-promoted, so this note records the divergence rather than silently rewriting history; the substitution changes no property this design relied on. Two Judgment Day correction rounds landed after this design's own commit-plan table was written: `finalizeSpan` was widened to record `http.response.status_code` on every post-handover failure (not only the success path this table's row 9 states), and `emitFailure`'s failure-construction-before-send reordering was extended to the in-band error-frame branch. Both are recorded in `apply-progress.md`'s Judgment Day sections and promoted into `openspec/specs/ai-observability-boundary/spec.md`'s `R-AOB-006`.
