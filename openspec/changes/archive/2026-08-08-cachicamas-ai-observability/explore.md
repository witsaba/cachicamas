# Exploration: AI-37 — Add the observability boundary

> **Change**: `cachicamas-ai-observability` · **Milestone**: AI-37 (doc 0002 lines 2188–2235) · **Wave**: 5 — Harden, last open milestone
> **Worktree**: `cachicamas-worktrees/ai-37-observability` · base `origin/main@ff28240a` · branch `feat/ai-37-observability`
> **Provenance**: this file mirrors Engram observations **#2701** (`sdd/cachicamas-ai-observability/explore`) and **#2702** (`sdd/cachicamas-ai-observability/dependency-probe`). The `sdd-explore` phase has no write tool in this pipeline, so the mirror was materialised by `sdd-propose`. Section 10 records the corrections `sdd-propose` made to #2701 after re-running its claims.

## 1. Forward guard (AI-00.3)

`backend/agent/src/ai/import_boundary_test.go` (337 lines).

- **Mechanism** — `listNonStdlibDeps` (`:188-213`) shells out to `go list -deps -test -f "{{if not .Standard}}{{.ImportPath}}{{end}}" <pattern>`. Not AST parsing, not `go.mod` parsing. `.Standard` is the toolchain's own stdlib classification (`:30-34` explains why the "first segment contains a dot" heuristic is wrong). `-test` closes the test-import blind spot and the package comment (`:20-28`) records that choice as load-bearing.
- **Two tests**:
  - `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` (`:107`) — forbidden-prefix check first (`:66-91`; already carries `go.opentelemetry.io/otel/sdk`, `.../exporters`, `.../contrib/bridges/otelslog`, each citing "ADR 0005 § D3" by name), then the deny-by-default allowlist (`allowedNonStdlibPrefixes`, `:103-105`, exactly one entry: the module path). Its own comment (`:98-101`) names AI-37 as the milestone that adds the second entry.
  - `TestLayer1_ModuleHasNoDependencies_ZeroRequires` (`:139-162`) — a pin that fails the moment `go.mod` gains any require. Its own doc (`:143-146`) anticipates AI-37: "If a milestone with an ADR added it deliberately, update this test and `allowedNonStdlibPrefixes` together."
- **Matching is prefix-based** — `isAllowed` (`:173-180`) admits `prefix` and `prefix + "/"`, so an allowlist entry silently admits every subpackage under it.
- **Test-only imports are not a loophole** — because the scan is `-test` scoped over the whole module pattern, a test-only SDK import is caught exactly as a production import is.

## 2. `go.mod` / `go.sum` today

- `backend/agent/go.mod` is 3 lines: module path, `go 1.26.3`, zero requires. No `go.sum` exists.
- AI-24 selected raw `net/http` and added zero requires (doc 0002:83 post-amendment), so **AI-37 is the only milestone that will add a dependency**, not the second.
- Sibling module `backend/database_administrator` already depends on `go.opentelemetry.io/otel v1.44.0`, `.../trace`, `.../metric` (indirect), `go-logr/logr`, `go-logr/stdr`. Precedent evidence only — versions may differ.
- No `sdk/trace/tracetest` anywhere in the repo; no test-only-SDK precedent exists.

## 3. Span seam in the streaming lifecycle

- `Client.Stream` (`src/ai/openaicompat/stream.go:201-233`): zero-request check → `Translate` → `ctx.Err()` → `retry.Loop(ctx, body, retry.Config{}, c.executeOnce)` (`:215`) → content-type check (`:226`) → `out := make(chan ai.Event, streamCarrierBuffer)` (`:230`) → `go run(ctx, resp, out)` (`:231`).
- Natural span start: the top of `Stream`, before `retry.Loop`, so one span covers the whole logical request including retries.
- `run` (`stream.go:593`+) is the producer goroutine with one success exit (the `[DONE]` sentinel building a `Completion`) and many failure exits. The span must end in `run`'s existing `defer` chain so it fires uniformly on every exit rather than being duplicated at each `return`.
- **`retry.count` has no success-path source.** `retry.Loop` (`src/ai/internal/retry/retry.go:67-112`) returns `(*http.Response, error)`. `AttemptReport{Attempts, FinalCause}` (`:50-53`) is constructed **only** at `:90` and `:111` — the budget-exhausted exits. The success return (`:81`), the non-retryable terminal return (`:87`) and both context-cancelled returns (`:93`, `:107`) carry no count at all.

## 4. Allowlisted-attribute source-of-truth map (twelve attributes)

| # | Attribute | Source of truth | Status |
|---|---|---|---|
| 1 | `gen_ai.system` | — | **No source anywhere.** `grep -rn "gen_ai" backend/agent/` returns zero hits. No adapter carries a self-identifying dialect/vendor string. |
| 2 | `gen_ai.request.model` | `ai.Request.Model()` | Present on every request. |
| 3 | `gen_ai.request.max_tokens` | `ai.Request.MaxOutputTokens() (int, bool)` (`request.go:504-505`) | Presence-typed. |
| 4 | `gen_ai.response.finish_reasons` | `ai.Completion.FinishReason()` (`completion.go:87`) → `FinishReason.String()` (`finish_reason.go:144-149`) | Semconv wants an array; Layer 1 tracks exactly one, so a one-element array. |
| 5–8 | usage counts | `ai.Completion.Usage()` (`completion.go:92`) → `ai.Usage{Input, Output, CacheRead, CacheWrite, Reasoning}` (`usage.go:111-149`), each `TokenCount.Count() (int64, bool)` (`usage.go:63-65`) | Exact match with § D3's four names. `Reasoning` is deliberately excluded from both — it is a breakdown of `Output`, not a term beside it (`usage.go:124-134`). |
| 9 | `http.response.status_code` | success: `resp.StatusCode` at `stream.go:215-226`. terminal failure: only `Failure.StatusClass() (int, bool)` (`provider_failure.go:478-492`), the **class** 1–5 | **Partial gap.** `mapErrorResponse` (`failure_map.go:43-78`) computes `statusClass := status/100` (`:47`) and discards the 3-digit code. |
| 10 | `retry.count` | `retry.AttemptReport.Attempts` (`retry.go:51`) — budget-exhausted exit only | **Gap.** See § 3. |
| 11 | `stream.event_count` | none | Trivial: a local counter in `run`, analogous to its existing `outputPreceded` / `toolBlocksOpen` state. |
| 12 | `error.type` | `ai.FailureCategory.String()` (`provider_failure.go:134-139`), closed nine-member vocabulary (`:155-167`) | Low-cardinality, suitable. |

## 5. Denylist blast radius / AI-37.3 instrument

- AI-36 shipped `backend/agent/src/agenttest/sweep/sweep.go` (94 lines): `Scan(corpus []byte, deny []Entry) (vector string, found bool)` (`:56-69`) and `SelfTest(deny []Entry) error` (`:85-93`), the mandatory positive control. The package imports only `bytes` and `fmt` (`:30-33`), deliberately, so it is importable from non-test files (`:5-12`).
- The sweep operates on **byte corpora**, so span data must first be serialised into one.
- **`Scan` returns `("", false)` for an empty corpus by construction (`:57-59`)** — an empty corpus is an automatic false green.
- **AI-41 lesson (doc 0002:22)**: a canary planted in a pointer-shaped field renders under `%#v` as a bare hex address and proves nothing. Canaries must go in string-valued fields and value-kind causes.
- **AI-36 residual explicitly recorded for AI-37 (doc 0002:24)**: "a conformance sweep lacking a corpus-non-empty guard".

## 6. In-memory test tracer

- § D3's table (`docs/adr/0005-...:237-244`) marks `go.opentelemetry.io/otel/sdk/…` and `.../exporters/…` ❌ for L1 with no test-only column, and its closing blockquote (`:257-261`) requires "its own ADR" for any additional OTel module.
- The forward guard's `-test` scope means a test-only `sdk/trace/tracetest` import fails the existing forbidden-prefix table (`import_boundary_test.go:88`) with no change to the guard.
- No precedent exists. `database_administrator/src/otel/otel.go` — the monorepo's one SDK site — is itself forbidden to L1 by name (§ D3 row 6).
- **Recommendation**: hand-roll a recording `trace.Tracer` / `trace.Span` against the API interfaces alone. Zero new dependency, zero guard exception, zero new ADR.

## 7. Nil-safe no-op

- The OTel Trace API is a no-op API absent an installed SDK: operations on a `Tracer` or a `Span` have no side effects and never panic. (Upstream documentation; not repo-verifiable at explore time because `go.mod` has zero requires. Apply phase confirms via `go doc` after `go get`.)
- Proof pattern: `agenttest/stream_kit_record.go:63` `DrainAndRecord(tb, ch, timeout) Recording` already exists. AI-37.4's test runs the same scripted stream twice — once untraced, once with the recording tracer — and asserts the two drained event sequences are equal. Same shape AI-33/AI-34's drip-frame and leak-check equivalence proofs use.

## 8. Conformance suite impact

- `agenttest/conformance_suite.go`: `Capability` is a **closed nine-member vocabulary** fixed at AI-03 (`:34-93`, `capabilityEnd = CapRetry + 1`). It cannot exceed nine without reopening AI-03.
- AI-37's Acceptance clause (doc 0002:2196) names nothing about conformance or capability discovery, and none of AI-37.1–.4's test lists mention `RunConformance` / `Factory` / `CapabilityRecord`.
- AI-36's `R-CNF-027` (`openspec/specs/ai-provider-conformance-suite/spec.md:395-403`) is the precedent that a milestone **can** add a conformance requirement without a new capability. Open question for `sdd-propose` to settle explicitly.

## 9. Dependency probe (observation #2702, orchestrator-measured, authoritative)

Measured with throwaway scratch modules on Go 1.26.3 (`go mod tidy` + `go list -deps`), otel **v1.45.0**:

| Import set | Modules | Non-stdlib packages | Notable |
|---|---|---|---|
| `go.opentelemetry.io/otel` (the § D3 global getter) | `otel`, `otel/trace`, `otel/metric`, `go-logr/logr`, `go-logr/stdr`, `cespare/xxhash/v2`, **`go.opentelemetry.io/auto/sdk`** | **24** | pulls `auto/sdk` + `auto/sdk/internal/telemetry`, `otel/baggage`, `otel/propagation`, `otel/metric`, `otel/internal/global` |
| `otel/trace` + `/trace/noop` + `/attribute` + `/codes` | 2 direct (`go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/trace`) + 1 indirect (`github.com/cespare/xxhash/v2`) | **10** | **no `auto/sdk`, no `logr`, no `metric`, no `baggage`/`propagation`** |
| `/attribute` alone | xxhash only | — | — |
| `/codes` alone | none | — | — |

**Consequence.** The guard is deny-by-default over the *package* closure, so choosing the ergonomic global getter would force the allowlist to admit a package literally named `go.opentelemetry.io/auto/sdk` inside the milestone whose stated purpose is keeping the SDK out. That is a spec-visible contradiction of AI-37's Acceptance clause, not a cosmetic one.

**Generalisable rule**: an ADR's import table authorises a **path**; it does not describe that path's **closure**. Measure the closure before treating an allowlist entry as cheap.

## 10. Corrections `sdd-propose` made to this exploration

1. **`go.work` DOES exist** at the repository root. #2701 claimed "no `go.work` file exists in this repo — confirmed via Glob". It does (`/go.work`), and `R-AGM-003` (`openspec/specs/agent-module-scaffold/spec.md:64-66`) requires it plus a committed `go.work.sum` if the toolchain generates one. This changes apply-time mechanics: `go get` inside `backend/agent` runs under a workspace whose build list is unified across three modules, one of which already pins otel v1.44.0.
2. **`TestLayer1_ModuleHasNoDependencies_ZeroRequires` is at `:139-162`**, not `:147` (`:147` is the `func` line; `:139` starts its doc comment).
3. **`retry.count` is absent on four of five `Loop` exits**, not merely on the success path — #2701 said "success path". `AttemptReport` is built only at `retry.go:90` and `:111`; `:81`, `:87`, `:93` and `:107` all return without a count.
4. **The test-only-SDK question is settled, not open.** #2701 called it a recommendation; it is a hard block with two independent causes (the forbidden-prefix table and § D3's table), recorded as proposal decision D-2.

## 11. Sizing forecast (superseded by the proposal's § 6)

Production ~150–280, test ~550–850, total ~700–1100 — Medium-High risk against a 1000-line budget, driven by the recording-tracer harness. The proposal re-forecasts upward after resolving D-1…D-6.

## 12. Unanswered at explore time — all resolved as proposal decisions

1. `gen_ai.system`'s value source → **D-3(a)**.
2. Exact HTTP status code on the failure path → **D-3(b)**.
3. `retry.count` on a retried success → **D-3(c)**.
4. Whether AI-37 adds a conformance requirement → **D-5**.
5. `go get` never run at explore time; the actual closure was measured separately in #2702 (§ 9) and is re-measured at apply time.
