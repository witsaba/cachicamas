# Proposal: Add the observability boundary (AI-37)

> **Change**: `cachicamas-ai-observability` · **Milestone**: AI-37 (doc 0002 lines 2188–2235) · **Wave**: 5 — Harden, the **last open milestone** of doc 0002
> **Module**: `backend/agent/` (layered per ADR 0005 § D1) · **Governing ADR**: [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) (`docs/adr/0005-promote-agent-stack-to-own-module.md:230-261`)
> **Strategy**: `single-pr`, `size:exception` pre-granted and auto-raising · strict TDD on · branch `feat/ai-37-observability`, base `origin/main@ff28240a`
> **Unblocked by**: AI-36 (redaction — doc 0002:24 records AI-37 as unblocked) · **Blocks**: AI-38

## 1. Intent

AI-37 is the only milestone in doc 0002 still permitted to add a dependency, and it is the one whose Acceptance clause is graded on *not* adding too much: "Layer 1 imports the OTel API and nothing else from that ecosystem; model content and secrets are never recorded; streaming behaves identically with no tracer configured" (doc 0002:2196).

Three things make this more than wiring a span:

- **The ADR authorises a path; it does not describe that path's closure.** § D3's table blesses `go.opentelemetry.io/otel` — the global getter — with a ✅ for Layer 1 (`docs/adr/0005-...:239`). Measured at otel v1.45.0, that one import pulls **24 non-stdlib packages including `go.opentelemetry.io/auto/sdk`** (explore.md § 9). The AI-00.3 guard is deny-by-default over the *package* closure (`import_boundary_test.go:107-137, 188-213`), so taking the blessed ergonomic path would force this milestone to allowlist a package literally named `…/auto/sdk` inside the milestone whose purpose is keeping the SDK out.
- **Three of the twelve allowlisted attributes have no source of truth in the tree.** `gen_ai.system` has zero hits anywhere; `http.response.status_code` survives only as a status *class* on the failure path (`provider_failure.go:478-492`); `retry.count` exists on exactly one of `retry.Loop`'s five exits (`retry.go:90, 111` vs `:81, 87, 93, 107`).
- **The milestone's headline test is an absence assertion**, the single easiest test in this repo to write vacuously — and AI-36 handed AI-37 the exact vacuity residual by name: "a conformance sweep lacking a corpus-non-empty guard" (doc 0002:24). `sweep.Scan` returns `("", false)` for an empty corpus by construction (`agenttest/sweep/sweep.go:57-59`).

## 2. Scope (in)

| # | Work unit | Charter leaf | Shape |
|---|---|---|---|
| WU-1 | The dependency: `go.mod` gains the § D3 API modules; `go.sum` is created; `go.work.sum` reconciled | AI-37.1 | Config |
| WU-2 | Forward guard extended — § D3 API paths allowlisted with the ADR cited in source; the zero-require pin **replaced** by an exact require-set **and** exact package-closure pin (D-6) | AI-37.1 t1 | Test-support |
| WU-3 | Recorded bite proof: a scratch SDK import and a scratch exporter import each fail the guard; red output recorded, violations dropped, suite re-run green | AI-37.1 t2 | Recorded evidence |
| WU-4 | Span seam: one span per logical request, started at `Stream`'s top before the retry loop, ended in `run`'s defer chain so every exit path closes it exactly once | AI-37.2 t3 | Prod + tests |
| WU-5 | Attribute mapping for the § D3 allowlist, keys spelled exactly as § D3 spells them, values equal to the normalized result | AI-37.2 t1/t2 | Prod + tests |
| WU-6 | The three missing carriers: a dialect constant, an exact status code on the success path, an attempt count out of `retry.Loop` (D-3) | AI-37.2 | Prod + tests |
| WU-7 | Hand-rolled recording tracer/span in `agenttest/tracetest/` implementing the API interfaces (D-2) | AI-37.2 t1 | Test-support |
| WU-8 | Denylist proven by absence over a run that used prompt, reasoning, tool-call, tool-result, credential, header and raw body — with four non-vacuity guards (D-4) | AI-37.3 t1 | Tests |
| WU-9 | Nil-safe no-op: drained event sequences with and without a tracer are equal; no adapter-side nil check exists | AI-37.4 t1 | Tests |
| WU-10 | Spec deltas (§ 8) | — | Spec |

## 3. Non-goals (out)

- **Dashboards, exporters and application tracing setup** — charter Out-of-scope, verbatim (doc 0002:2198); § D3 assigns them to the composition root.
- **Any OTel module beyond § D3's table** — no `metric`, no `baggage`, no `propagation`, no `contrib`, no `sdk`. § D3's closing blockquote (`docs/adr/0005-...:257-261`) requires a separate ADR for each. This proposal takes a **strict subset** of what § D3 permits (D-1), which needs no ADR amendment.
- **Metrics of any kind.** § D3 lists `/metric` as ✅, but the charter's Deliverable is "minimal spans and attributes". A metric instrument is a new lifecycle concern with no charter test behind it.
- **The exact HTTP status code on the terminal-failure path** (D-3b) — a recorded omission plus a follow-up, not a widening of `ai.Failure`.
- **Any conformance-suite change** (D-5) — the charter is silent and the `Capability` enum is closed at nine.
- **Tracing in Layer 2/3 or the composition root** — those directories do not exist yet (`import_boundary_test.go:76-82`).
- **Context propagation / trace-context header injection.** That needs `otel/propagation`, which the closure measurement excludes and which no charter test names.
- **Re-litigating AI-36's residuals** (interior non-tail credential prefixes; per-file plant allowlist; the by-value structural guard's composite shapes — doc 0002:24). AI-37 discharges exactly **one** of the four informational items AI-36 recorded for it: the corpus-non-empty guard (D-4).

## 4. Decisions taken

### D-1 — Injected `TracerProvider`, not the § D3 global getter. Layer 1 takes a **strict subset** of what the ADR permits.

**Decision.** Layer 1 imports exactly `go.opentelemetry.io/otel/trace`, `go.opentelemetry.io/otel/trace/noop`, `go.opentelemetry.io/otel/attribute` and `go.opentelemetry.io/otel/codes`. It does **not** import the root package `go.opentelemetry.io/otel`, and therefore never calls `otel.Tracer(...)` or `otel.GetTracerProvider()`.

**Entry point.** A new optional field on the existing construction surface: `openaicompat.Config` (`client.go:43-58`) gains a `TracerProvider` field alongside `HTTPClient`, mirroring it exactly — optional, nil-defaulted, used verbatim, never mutated. `New` (`client.go:106-125`) substitutes `noop.NewTracerProvider()` when the field is nil, on the same line-shape as `newDefaultHTTPClient()` at `:116-118`. The tracer is derived once at construction and stored on `Client`; `Stream` never reaches for global state.

**How the no-op default is established.** By the API's own no-op provider, not by an adapter-side nil check. `noop.NewTracerProvider()` returns a real, non-nil `trace.TracerProvider` whose spans are genuine no-ops, so every `span.SetAttributes` / `span.End` call site is unconditional. This is precisely what AI-37.4 asks for — "the API's no-op default suffices without adapter-side nil checks" (doc 0002:2234) — and it makes AI-37.4's test a *behavioural* equivalence proof rather than a nil-guard inspection.

**Why not the global getter, which § D3 blesses.**

1. **Closure.** Measured (explore.md § 9): the global getter's package closure contains `go.opentelemetry.io/auto/sdk`, `go-logr/logr`, `go-logr/stdr`, `otel/metric`, `otel/baggage`, `otel/propagation` — 24 non-stdlib packages. The injected set contains 10 and no `sdk`-named path at all. Allowlisting a package named `auto/sdk` would make AI-37's Acceptance clause — "the OTel API and nothing else from that ecosystem" — literally false in the guard that is supposed to enforce it.
2. **Ambient authority.** `R-APC-008` is landed and normative: "Its credential and its configuration arrive **only** by injection" (`openspec/specs/ai-provider-client/spec.md:161`). A process-global tracer provider registry, set by whoever called `otel.SetTracerProvider` first, is configuration that does not arrive by injection. R-APC-008's mechanical guard scans for os/exec/syscall (`:163-165`) and would not catch it — which makes this a rule the milestone must honour deliberately, not one the tooling will enforce for it.

**Why this needs no ADR amendment.** § D3 pre-authorises "exactly the paths in the table above and nothing else". Importing fewer of those paths is inside the authorisation; importing more is not. `trace/noop` and `trace/embedded` are subpackages of `go.opentelemetry.io/otel/trace`, which the table names, and the guard's matcher is prefix-based (`import_boundary_test.go:173-180`), so both are covered without a table change. **The declined row is recorded in the guard's source and in the spec delta**, so a future reader sees a deliberate narrowing rather than an oversight.

**One consequence, stated up front.** The module `go.opentelemetry.io/otel` still appears as a `require` in `go.mod`, because `attribute` and `codes` live in it — a *module* require without a *package* import. D-6's pin distinguishes the two.

**Alternative considered and rejected — context-derived provider.** `trace.SpanFromContext(ctx).TracerProvider()` would give injection with zero construction-surface change and zero spec delta. Rejected: it silently produces no span whenever the caller has not already opened a parent span, so a caller who configured a provider correctly still gets nothing, with no error anywhere. Implicit is worse than explicit here. (It also depends on the `Span` interface exposing `TracerProvider()`, which is unverified in-repo — `go.mod` has zero requires, so no vendored source exists to cite. Design phase confirms via `go doc` after `go get`.)

**Spec cost, paid openly.** Injection contradicts two landed scenarios, and both must be amended rather than quietly outgrown:

- **`S-APC-015`** (`ai-provider-client/spec.md:100`) enumerates the construction surface exhaustively: "it is the endpoint, the credential and an **optional** HTTP client". A tracer provider is a fourth injectable. Note its qualifier — "and **no** timeout, deadline or bound value appears among them" — a tracer provider is none of those, so the *rationale* under S-APC-015 survives intact; only the enumeration changes.
- **`R-APC-001`** (`:57-59`) requires every injected value to be proven **used**, "observable through a stub transport at the round-trip boundary … and MUST NOT be asserted by reading a stored field". A tracer provider is never observable in what the adapter sends. The requirement needs a scenario shaped for a non-transport injectable: two adapters built with two different recording providers, each recording exactly its own span and neither the other's — the direct analogue of `S-APC-004` (`:66`).

### D-2 — Hand-rolled recording tracer in `agenttest/tracetest/`. A test-only SDK dependency is **genuinely blocked**, not merely inconvenient.

**Verified independently of the exploration.** Two independent blocks, either of which alone is fatal:

1. **The guard.** `forbiddenPrefixes` carries `{"go.opentelemetry.io/otel/sdk", "ADR 0005 § D3: the OTel SDK belongs to a composition root"}` (`import_boundary_test.go:88`), checked **before** the allowlist (`:119-125`), over a package set produced by `go list -deps -test` (`:189-190`) scoped to the whole module pattern (`:51`). A `_test.go` file importing `sdk/trace/tracetest` lands in that set exactly as a production import does. Admitting it would require **deleting a forbidden-prefix row** — reverting a guard to unblock a dependency, which `agent-module-scaffold/spec.md:20` names as "the exact failure ADR 0005 § Guard A was written against".
2. **The ADR.** § D3's table marks `go.opentelemetry.io/otel/sdk/…` ❌ for L1 with no test-only column (`docs/adr/0005-...:242`), and its closing blockquote requires "its own ADR" for any additional OTel module (`:257-261`). `go.opentelemetry.io/otel/sdk` is a separate module.

So the answer is: blocked, and unblocking it costs a guard weakening **and** a new ADR — for a test fake this milestone can write in a few hundred lines.

**Decision.** Hand-roll a recording `trace.TracerProvider` / `trace.Tracer` / `trace.Span` against the API interfaces, in a new package **`backend/agent/src/agenttest/tracetest/`**.

**Why that path.** It follows the `agenttest/sweep/` precedent exactly, and for the same recorded reason: `agenttest` imports `testing` package-wide, so a subpackage that imports neither `testing` nor anything else is what makes the helper importable from any package in the module (`agenttest/sweep/sweep.go:5-12`). `src/ai/internal/…` (the `retry` precedent) is rejected here: Go's internal rule would make it importable only under `src/ai/`, excluding `agenttest` itself, which ADR 0005 § D2 designates as the external-consumer-proof location (`docs/adr/0005-...:205, 223-226`) and which AI-38 will plausibly want.

**Forward-compatibility constraint.** `trace.Span` embeds `embedded.Span`, and `trace.Tracer` / `trace.TracerProvider` embed their `embedded` counterparts — the OTel API's mechanism for adding interface methods without a breaking change. Any hand-rolled implementation must satisfy them. `go.opentelemetry.io/otel/trace/embedded` is under the § D3-named `/trace` prefix, so it is allowlisted by prefix and costs no ADR. The design phase may either embed the `embedded` types directly or embed `noop.Span` / `noop.Tracer` to inherit the required method set; both are inside the same import budget.

### D-3 — The three source-of-truth gaps: **add two carriers, omit one with a recorded justification.**

The charter's Out-of-scope line is "dashboards, exporters and application tracing setup" (doc 0002:2198). None of the carriers below is any of those; each is the minimum source of truth for an attribute § D3 names and AI-37.2 test 1 enumerates. But "minimum" is load-bearing — where a carrier would re-open a landed decision, the milestone narrows its own claim instead, per the AI-41 precedent (doc 0002:22).

**(a) `gen_ai.system` — ADD a carrier. Cost: one unexported constant.**
A package-level `const` in `openaicompat` naming **the dialect the package speaks, not the vendor behind the endpoint** — the honest semantic, since the same adapter serves OpenAI, OpenRouter and any compatible gateway. No exported surface, no vendor-detection mechanism, no per-adapter override. Omitting it instead would contradict AI-37.2 test 1's own enumeration, which names "system" first (doc 0002:2220). If a later adapter needs a distinct value, that is a follow-up with an exported seam behind it — not this milestone's invention.

**(b) `http.response.status_code` — SPLIT: record it where the exact code exists, OMIT it where only the class does.**
- *Success path*: `resp.StatusCode` is in hand at `stream.go:215-226` before the carrier is built. Recorded, exactly. Zero production carrier needed.
- *Terminal-failure path*: only `Failure.StatusClass() (int, bool)` survives (`provider_failure.go:478-492`); `mapErrorResponse` computes `status/100` and discards the code (`failure_map.go:43-78`). **Omitted, with the omission recorded in the spec and in source.**
- **Why not add an exact-code carrier to `ai.Failure`.** It would re-open `R-AIP-009`'s landed "bounded safe metadata" decision on a spec'd public type, for telemetry convenience, inside a milestone whose Deliverable is a boundary. That is the shape AI-41 refused (doc 0002:22).
- **Why not record the class under the semconv key.** `http.response.status_code` means the code. Writing `4` where a consumer expects `429` is a lie in a telemetry contract that "renaming later is a breaking change for every consumer" (AI-37.2 test 2, doc 0002:2221) exists to protect.
- **What covers the gap meanwhile.** `error.type` carries the failure classification on exactly that path, from a closed nine-member vocabulary (`provider_failure.go:134-139, 155-167`).
- **Follow-up recorded**: an exact terminal status code is a change to `ai-provider-errors`, owned by whichever milestone needs it.

**(c) `retry.count` — ADD a carrier. An internal signature change with one production call site.**
`retry.Loop` returns `(*http.Response, error)` (`retry.go:67-72`) and builds `AttemptReport` only at `:90` and `:111`. Its other three exits — success `:81`, non-retryable terminal `:87`, context-cancelled `:93`/`:107` — carry no count. So the attribute is unavailable on **four of five** exits, not merely on success.
**Decision**: `Loop` returns the retry count alongside the response, valid on every exit. `retry` is `src/ai/internal/retry` — an internal package with exactly one production caller (`stream.go:215`) plus its own tests. Zero public blast radius, zero behaviour change.
**Semantics pinned now**: the attribute is the number of **retries**, i.e. `attempt - 1` against `Loop`'s 1-based counter (`:78`), so an unretried success reports `0`.
**Rejected alternative**: an observer callback on `retry.Config`. `Config` already carries eight fields; a callback adds a surface and an ordering question for a value a return can carry.

### D-4 — Non-vacuity of AI-37.3: four guards, and a canary discipline this repo has already been burned by.

**Instrument.** Reuse `agenttest/sweep` (`sweep.go:56-69, 85-93`) — one scanner in the tree, per AI-36's own charter. The corpus is a deterministic serialization built by walking every recorded span: span name; status description; **every attribute key and value**; every event name and every event-attribute value; and every recorded error string.

**Rendering rule (the AI-41 lesson, doc 0002:22).** Attribute values are rendered through the API's own value accessor (`attribute.Value.Emit()` / `AsString()`), **never** by `%#v` over the recorder's structs. A canary planted in a pointer-shaped field renders under `%#v` as a bare hex address and proves nothing — that is how AI-41's first canary went green for the wrong reason.

**Where the canaries go — string-valued and value-kind, six vectors**, each a runtime-assembled needle (never a contiguous literal, or the scan flags its own source — `credential_scan_test.go:39-44`):

| Vector | Plant site |
|---|---|
| prompt text | a user message's content string |
| reasoning text | a reasoning delta in the scripted stream |
| tool-call arguments | the arguments JSON text of a tool-call frame |
| tool-result text | a tool message's result string |
| credential | the bearer token the client is constructed with |
| header + raw body | a custom response header value, and a distinctive marker inside the SSE frames |

**Four non-vacuity guards, because `SelfTest` alone proves only that the needles bite:**

1. **`sweep.SelfTest(deny)` before every clean scan** — the mandatory positive control (`sweep.go:85-93`).
2. **Corpus-non-empty guard** — assert corpus length > 0, recorded span count ≥ 1, and recorded attribute count ≥ the expected allowlist cardinality. `Scan` returns `("", false)` for an empty corpus by construction (`sweep.go:57-59`), so without this the whole absence claim passes on a run that recorded nothing. **This is the exact residual AI-36 recorded for AI-37 by name** (doc 0002:24).
3. **Coverage guard** — assert the drained `Recording` (`agenttest/stream_kit_record.go:63`) actually contained text, reasoning, tool-call, completion and error events, so the absence is asserted "over a run that used all of them" (AI-37.3's own wording, doc 0002:2228) rather than over a run that used none.
4. **Recorded RED proof by overlay** — `go test -overlay` against a variant attribute mapper that copies message text into a span attribute must make the absence test **fail**; the red output is recorded, the overlay dropped, the suite re-run green. This is the repo's standard for proving a RED without mutating the worktree, and it matches the "recorded, dropped" grammar `R-APC-010` already uses for a guard leaf (`ai-provider-client/spec.md:190-209`).

**Failure-message rule inherited from AI-36**: no sweep reprints the bytes it matched — it names the vector, the span, and the attribute key only.

### D-5 — The conformance suite: **AI-37 does not touch it.** Silence is the reason.

**Decision.** No new `Capability`, no new `R-CNF-*` requirement, no change to `agenttest/conformance_suite.go`.

**Why.**
1. The charter's Acceptance clause (doc 0002:2196) names nothing about conformance or capability discovery, and none of AI-37.1–.4's four test lists mentions the suite. AI-36's parallel decision was the opposite because its charter's own test list demanded a reusable cross-adapter helper; AI-37's does not.
2. `Capability` is a **closed nine-member vocabulary** fixed at AI-03 (`agenttest/conformance_suite.go:34-93`, `capabilityEnd = CapRetry + 1`). A tenth reopens AI-03's decision — a cost with no charter benefit.
3. **AI-36's `R-CNF-027` precedent does not transfer.** It works because redaction is a property *every* conformant adapter must have. Tracing is not: the fake provider in `agenttest` is a conformant `ModelProvider` with no HTTP transport, no retry loop and no status code, so a generic tracing requirement would be either vacuous for it or an obligation to trace that no charter states.
4. **Identifier hazard.** `R-CNF-020`…`R-CNF-026` are consumed by two archived amendments that were never promoted, and `R-CNF-023`/`R-CNF-024` are cited as binding from two other specs; AI-36 took `R-CNF-027` and deliberately left the gap as a tracked repository defect (doc 0002:24). Taking another R-CNF identifier for an unrequested requirement grows a known defect.

**Follow-up recorded**: if AI-38 makes tracing a cross-adapter obligation, AI-38 owns the conformance requirement.

### D-6 — Replace the zero-require pin with an **exact require-set plus exact package-closure** pin. Delete nothing.

`TestLayer1_ModuleHasNoDependencies_ZeroRequires` (`import_boundary_test.go:139-162`) fails the instant `go.mod` gains a require. Its own doc says what to do: "If a milestone with an ADR added it deliberately, update this test and `allowedNonStdlibPrefixes` together" (`:143-146`).

**Replacement contract** — the test is renamed and re-expressed, not removed, and it asserts **three** things:

1. **Exact require set.** Parse `backend/agent/go.mod` and assert its `require` set equals a literal in-test table of module paths **and versions**, including the `// indirect` entry, with **zero** `replace` directives. Currently expected (re-measured at apply time): `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/trace`, `github.com/cespare/xxhash/v2 // indirect`.
2. **Exact non-stdlib package closure, by set equality — not prefix subset.** Assert `listNonStdlibDeps(layer1Pattern)` (`:188-213`) equals an enumerated set: the module's own packages plus the measured OTel/xxhash packages. **This is the load-bearing half.** The allowlist matcher is prefix-based (`:173-180`), so `go.opentelemetry.io/otel/metric`, `otel/baggage` and `otel/propagation` would all be admitted *silently* by the entry `go.opentelemetry.io/otel` the moment a future import or version bump pulled them in. Set equality is what keeps § D3's "and nothing else" honest at package granularity, and it is what makes a version bump that introduces `auto/sdk` a build failure rather than a quiet drift.
3. **Forbidden table unchanged.** The `sdk` / `exporters` / `otelslog` rows (`:88-90`) stay exactly as written. AI-37 adds no forbidden row and removes none.

**The xxhash question, answered explicitly.** `github.com/cespare/xxhash/v2` enters the closure transitively through `go.opentelemetry.io/otel/attribute`, and § D3's table does not name it. It is allowlisted, with the reasoning recorded in the guard's source: **authorising an import path necessarily authorises its closure — you cannot import the path otherwise** — and xxhash is not an OTel module, so § D3's "any additional OTel module requires its own ADR" clause is not engaged. The closure pin in (2) is what stops that reasoning from becoming a blank cheque.

**Spec impact, pre-authorised.** `S-AGM-001` asserts "zero `require` directives AND zero `replace` directives" (`agent-module-scaffold/spec.md:42`). The scaffold spec itself names AI-37 as the milestone that amends it: "A later milestone that needs to change one of these invariants — most concretely AI-24 adding a transport allowlist entry, or **AI-37 adding the OpenTelemetry API modules** — amends **this file**, in the same pull request, under its own ADR gate" (`:20`, restated at `:26`). Amending it here is the documented path, not an exception.

## 5. Approach

Strict TDD throughout (`openspec/AGENTS.md` rule 3): RED asserting for the right reason → GREEN with the minimum code → REFACTOR.

Work-unit order follows the charter's dependency edges (`AD1 → AD2 → {AD3, AD4}`, doc 0002:2201-2208):

**WU-1 → WU-2 → WU-3** (the dependency and its guard land first, so no span code is ever written under a stale guard) **→ WU-7** (the recording tracer, since every later test needs it) **→ WU-4 → WU-6 → WU-5** (carriers before the mapping that consumes them) **→ {WU-8, WU-9} → WU-10.**

Three disciplines are non-negotiable for this milestone specifically:

1. **Every absence assertion ships its positive control and its corpus guard.** § D-4. An absence assertion that cannot fail trains reviewers to trust it.
2. **Attribute keys are literal constants, spelled once.** AI-37.2 test 2 makes a rename a breaking change for every consumer (doc 0002:2221); a key must never be assembled at a call site.
3. **The span never sees a value that has not already passed AI-36's redaction posture.** AI-37 depends on AI-36 precisely so that nothing is recorded before the sweep that proves it safe exists (doc 0002:2215).

## 6. Size forecast and delivery plan

**Delivery**: one PR on `feat/ai-37-observability`, base `origin/main@ff28240a`. `size:exception` is pre-granted for this milestone and the budget auto-raises, so the forecast below is stated honestly rather than trimmed to fit 1000 lines. No chained/stacked split is proposed: the PR covers AI-37 alone, and WU-1/WU-2 are not independently shippable — a `go.mod` require without the guard update leaves `main` red.

| Work unit | Production | Test / test-support | Subtotal |
|---|---|---|---|
| WU-1 dependency (`go.mod` +~6, `go.sum` ~+15 generated, `go.work.sum` ~+15 generated) | ~6 | — | ~6 (+~30 generated) |
| WU-2 guard: allowlist + require-set + closure pin | — | ~130 (−~24 replaced) | ~130 |
| WU-3 bite proof | — | 0 (recorded, dropped) | 0 |
| WU-4 span seam (`stream.go` start/end + defer) | ~45 | ~140 | ~185 |
| WU-5 attribute mapping (new `openaicompat/trace.go`) | ~140 | ~260 | ~400 |
| WU-6 three carriers (dialect const ~5; status code ~8; `retry.Loop` signature ~12 + call site ~4) | ~30 | ~110 | ~140 |
| WU-7 recording tracer `agenttest/tracetest/` | ~230 (test-support) | ~180 | ~410 |
| WU-8 denylist absence + 4 non-vacuity guards | — | ~280 | ~280 |
| WU-9 nil-safe no-op equivalence | — | ~150 | ~150 |
| **Code total** | **~450** | **~1,250** | **~1,700** |
| WU-10 spec deltas | — | — | ~280 |

**Band**: 1,400–2,100 code lines. **Central estimate ~1,700.**

**Production-vs-test ratio** ≈ 2.8×, *lower* than this repo's recent band (AI-34 +1241/−2 for +15 production; AI-35 +1301/−62; AI-41 +257/−12 for +27 production; AI-36 +4997/−130) because AI-37 is the first Wave-5 milestone with a substantial production surface of its own. The forecast is a **central estimate, not a ceiling**: AI-35 overran its accepted exception by 168 lines and AI-36 overran its ~1,310 forecast by roughly 3.8×, both on test-evidence density.

**Guard lines** (per `sdd-phase-common.md` § E; `sdd-tasks` re-forecasts authoritatively):

```
Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: High
```

`Decision needed before apply: No` because the maintainer pre-granted `size:exception` with automatic raising for this milestone specifically, and `Chained PRs recommended: No` because the PR covers AI-37 alone and its first two work units are not independently shippable.

**Named scope-trim levers, ranked**, so apply can shed lines without reopening a decision: (a) let the recording tracer embed `noop.Span`/`noop.Tracer` instead of hand-writing the full `embedded`-satisfying method set (−~90, no loss of coverage); (b) fold WU-9's equivalence test into WU-8's existing traced run rather than running a second scripted stream (−~70, slightly weaker isolation); (c) drop the per-attribute table test in favour of one whole-attribute-set equality assertion (−~120, least preferred — it stops naming which attribute regressed).

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 7.1 | The absence assertion (WU-8) passes vacuously — the milestone's characteristic failure mode, and the one AI-36 handed forward by name. | **High** | D-4's four guards: SelfTest, corpus-non-empty, event-coverage, and a recorded overlay RED. |
| 7.2 | `go.work` exists at the repository root, so `go get` inside `backend/agent` resolves under a **unified workspace build list** across three modules, one of which already pins otel v1.44.0. The version landing in `backend/agent/go.mod` may differ from what `go list` reports, and a `go.work.sum` change will appear in the PR. | **High** | D-6 pins the require set from `backend/agent/go.mod` **and** the closure from `go list` separately, so a divergence fails the build instead of hiding. `R-AGM-003` (`agent-module-scaffold/spec.md:64-66`) already requires `go.work.sum` to be committed. Design phase re-measures both under the workspace. |
| 7.3 | A version bump silently grows the closure (e.g. `auto/sdk`) under a prefix the allowlist already admits. | Med | D-6's **set-equality** closure pin, deliberately not a prefix subset. |
| 7.4 | Amending `S-APC-015` / `R-APC-001` regresses AI-25's landed proofs. | Med | Both are `MODIFIED` deltas restating the **entire** requirement including every unchanged scenario, per `openspec-convention.md`. `S-APC-001`…`S-APC-016` all re-run green as an explicit success criterion. |
| 7.5 | The hand-rolled tracer drifts from the API's interface set on a later otel version (`embedded.Span` exists precisely to allow method additions). | Med | Trim lever (a): embed `noop.Span`/`noop.Tracer` so new methods are inherited rather than hand-written. D-6's version pin makes any bump a deliberate, reviewed event. |
| 7.6 | The span leaks a value through a path the corpus builder does not walk (e.g. a span *link*, or a status description set by a helper). | Med | The corpus builder walks name, status description, attributes, events and errors; the recording tracer records **everything the API can set**, and WU-8 asserts the recorder's captured-field count matches the API surface it implements. |
| 7.7 | `retry.Loop`'s signature change breaks `retry_test.go`'s existing assertions. | Low | Internal package, one production caller (`stream.go:215`); the change is additive in meaning (behaviour-neutral) and the tests are in the same work unit. |
| 7.8 | The size forecast overruns, as AI-35's and AI-36's both did. | **High** | `size:exception` pre-granted and auto-raising; three ranked trim levers pre-approved in § 6. |
| 7.9 | Spec-identifier collision if another change lands first. | Low | Maxima verified at propose time (§ 8). `sdd-spec` re-verifies before writing. |

## 8. Capabilities (contract with `sdd-spec`)

### New capabilities

| Capability | Suggested prefix | What it covers |
|---|---|---|
| `ai-observability-boundary` | `R-AOB-` (verified free against every spec under `openspec/specs/`) | The whole of AI-37's behaviour: which telemetry paths Layer 1 may reach at all; that a tracer arrives only by injection; that one span covers one logical request and closes exactly once on every exit; the twelve-key allowlist with exact semconv spellings and their value sources; the absolute content denylist proven by absence with its non-vacuity guards; and behavioural identity with no tracer configured. |

### Modified capabilities

| Capability | Next free ID | What changes |
|---|---|---|
| `agent-module-scaffold` | `R-AGM-008` | **`R-AGM-001` MODIFIED** — the module's dependency invariant becomes an exact, pinned require set rather than "zero requires"; `S-AGM-001` restated. **`R-AGM-005` MODIFIED** — the forward guard's allowlist admits the § D3 API paths and their transitively-required packages, and the guard additionally pins the non-stdlib package closure by **set equality**. A new `R-AGM-008` may carry the closure pin as its own requirement if `sdd-spec` prefers it separable. Pre-authorised in place by `agent-module-scaffold/spec.md:20, 26`. |
| `ai-provider-client` | `R-APC-016` | **`R-APC-001` MODIFIED** — a fourth injectable exists whose use cannot be proven at the round-trip boundary; its use is proven instead by two adapters with two different recording providers each observing only their own span (the `S-APC-004` analogue). **`R-APC-003` MODIFIED** — `S-APC-015`'s enumeration of the construction surface gains the optional tracer provider; its "no timeout, deadline or bound value" qualifier is unchanged and still true. `R-APC-016` states the new value's own defaulting rule: absent injection, the API's no-op provider is used, never a process-global one. |

> Spec files honour the **no-Go-identifier** rule: behaviour-level wording only. Every identifier in this proposal is for design/apply consumption, not for spec prose. Attribute **keys** are the one deliberate exception — `gen_ai.system`, `http.response.status_code` and their nine siblings are external semantic-convention names, not Go identifiers, and AI-37.2 test 2 requires them spelled exactly.

## 9. Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/go.mod` | **Modified** | +2 direct requires, +1 indirect. First requires this module has ever carried. |
| `backend/agent/go.sum` | **New** | Created for the first time. Generated. |
| `go.work.sum` (repo root) | Modified/**New** | Workspace consequence of WU-1; `R-AGM-003` requires it committed. Generated. |
| `backend/agent/src/ai/import_boundary_test.go` | Modified | Allowlist entries + the require-set/closure pin replacing the zero-require pin (D-6). Forbidden table unchanged. |
| `backend/agent/src/ai/openaicompat/trace.go` | **New** | Attribute keys, span start/end, attribute mapping. ~140 lines. |
| `backend/agent/src/ai/openaicompat/client.go` | Modified | `Config.TracerProvider` field, `noop` default in `New`, tracer stored on `Client`. ~25 lines. |
| `backend/agent/src/ai/openaicompat/stream.go` | Modified | Span start above `retry.Loop` (`:215`), span end in `run`'s defer chain, event counter, exact status code on the success path. ~30 lines. |
| `backend/agent/src/ai/internal/retry/retry.go` | Modified | `Loop` returns the retry count on every exit (D-3c). ~12 lines. |
| `backend/agent/src/agenttest/tracetest/` | **New** | Hand-rolled recording `TracerProvider`/`Tracer`/`Span`. Dependency-light subpackage, per the `agenttest/sweep/` precedent. ~230 lines. |
| `backend/agent/src/ai/openaicompat/a_i-37_*_test.go` | **New** | Attribute tables, denylist absence, no-op equivalence. Milestone-numbered convention, per AI-33/34/35/36 in this package. |
| `backend/agent/src/ai/internal/retry/retry_test.go` | Modified | Signature update; assertions on the count at each of the five exits. |
| `openspec/specs/ai-observability-boundary/spec.md` | **New** | Via the change folder's delta. |
| `openspec/changes/cachicamas-ai-observability/specs/**` | **New** | Delta specs per § 8. |
| `backend/agent/src/agenttest/conformance_suite.go` | **Untouched** | D-5. |
| `.gitignore` (root and module) | Verify | Must not exclude `go.sum` or `go.work.sum` (`R-AGM-003`). |

## 10. Rollback plan

Revert the PR. Every unit is independently revertible, and the revert order is the inverse of § 5's:

- **WU-8/WU-9** — tests only; removal is behaviour-neutral.
- **WU-5/WU-4/WU-6** — deleting `trace.go`, the two `stream.go` seams and the `Config` field restores today's exact behaviour. `retry.Loop`'s extra return value is dropped with its one call site; because the count was never consulted for control flow, no assertion's verdict changes.
- **WU-7** — deleting `agenttest/tracetest/` removes a package with no production importer.
- **WU-2/WU-1** — restoring the zero-require pin and emptying `go.mod` is mechanical **only if reverted together**. Reverting WU-1 alone leaves the guard pinning a require set that no longer exists; reverting WU-2 alone leaves `main` red. They revert as one commit pair.

If the boundary proves wrong in substance — a value later found on a span that the denylist declared absent — the correct move is **not** to narrow the sweep. Record the leak, re-open the affected requirement, and amend the spec append-only. Leaving a requirement reading *proven* while its proof was quietly weakened is the one rollback error that reproduces the failure mode this milestone exists to stop (AI-36's own rollback rule, restated).

## 11. Success criteria

- [ ] `backend/agent/go.mod` requires **exactly** the § D3 API modules plus their transitive indirect, with zero `replace`; the non-stdlib package closure equals an enumerated set containing **no** `sdk`-named path.
- [ ] Layer 1 imports `otel/trace`, `otel/trace/noop`, `otel/attribute` and `otel/codes` — and **not** `go.opentelemetry.io/otel`. The declined global getter is recorded in source as a deliberate strict subset of § D3.
- [ ] The forward guard cites § D3 as the authorising ADR for every OTel allowlist entry, and a scratch SDK import and a scratch exporter import each **fail** it — red output recorded, violations dropped, suite re-run green.
- [ ] A traced request carries **only** § D3-allowlisted attribute keys, spelled exactly as § D3 spells them, with values equal to the normalized result; every omitted allowlisted key carries a recorded justification in spec and in source.
- [ ] The span starts before the retry loop and closes **exactly once** on every terminal path — `[DONE]`, every failure exit, and both mid-stream cancellation paths.
- [ ] A tracer provider arrives **only** by injection; no process-global telemetry state is read anywhere in Layer 1.
- [ ] Six planted canaries — prompt, reasoning, tool-call arguments, tool-result, credential, header/raw body — are absent from every span name, attribute value, event, status description and recorded error, over a run proven to have exercised all of them.
- [ ] That absence proof ships **all four** non-vacuity guards, and the overlay RED is recorded: with the leaking mapper overlaid, the test fails.
- [ ] With no tracer configured, drained event sequences are byte-equal to the traced run's, nothing panics, and **no adapter-side nil check exists** — the API's no-op default carries it.
- [ ] `cd backend/agent && make test` green under `-race`; `make lint` 0 issues; `make build` exit 0; `git status --porcelain` empty.
- [ ] `S-APC-001`…`S-APC-016` and every `S-AGM-*` scenario re-run green under their amended requirements.
- [ ] Spec deltas authored per § 8 in behaviour-level wording, with zero Go identifiers (semconv attribute keys excepted).

## 12. Recommended next step

`sdd-spec` and `sdd-design` (parallel).

- **`sdd-spec`** authors the new `ai-observability-boundary` capability and the two `MODIFIED` deltas (§ 8), restating `R-APC-001`, `R-APC-003`, `R-AGM-001` and `R-AGM-005` in full including every unchanged scenario.
- **`sdd-design`** owns five deliverables this proposal deliberately leaves open: (1) the **measured** require set, versions and package closure after `go get` under the workspace, which D-6's pin literals depend on; (2) the recording tracer's exact interface-satisfaction strategy — `embedded` types directly, or `noop` embedding; (3) the precise `defer` site in `run` where the span ends, proven against every terminal exit; (4) the corpus-builder's field walk and the recorder's captured-field-count assertion (risk 7.6); (5) the overlay shape for D-4's recorded RED.

## References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2188-2235`; Acceptance `:2196`; Out-of-scope `:2198`; § D3 note `:2199`; subnode test lists `:2210-2235`.
- **AI-41 canary lesson and the "narrow the claim" precedent** — doc 0002:22.
- **AI-36 close, the four items recorded for AI-37, and the R-CNF identifier hazard** — doc 0002:24.
- **Governing ADR** — `docs/adr/0005-promote-agent-stack-to-own-module.md:230-261`; import table `:237-244`; attribute allowlist `:246-250`; absolute denylist `:252-255`; the "nothing else" blockquote `:257-261`; `agenttest` sibling constraint `:205, 223-226`.
- **Forward guard** — `backend/agent/src/ai/import_boundary_test.go`: package rationale `:1-34`; module pattern `:44-51`; forbidden table `:66-91`; allowlist `:93-105`; deny-by-default test `:107-137`; zero-require pin `:139-162`; prefix matcher `:164-180`; `go list -deps -test` `:188-213`.
- **Construction surface** — `backend/agent/src/ai/openaicompat/client.go:43-58` (`Config`), `:52-57` (optional `HTTPClient`), `:63-67` (`Client`), `:106-125` (`New`), `:134-147` (adapter-built default).
- **Stream seam** — `backend/agent/src/ai/openaicompat/stream.go:201-233`; `retry.Loop` call `:215`; content-type check `:226`; carrier + goroutine `:230-232`; producer goroutine `run` `:593`+.
- **Retry helper** — `backend/agent/src/ai/internal/retry/retry.go:50-53` (`AttemptReport`), `:67-72` (`Loop` signature), `:78` (1-based counter), `:81, 87, 90, 93, 107, 111` (the five exits).
- **Status class, not code** — `backend/agent/src/ai/provider_failure.go:478-492`; `error.type` source `:134-139, 155-167`.
- **Shared sweep to reuse** — `backend/agent/src/agenttest/sweep/sweep.go:5-12` (subpackage rationale), `:39-42` (`Entry`), `:44-69` (`Scan`, empty-corpus behaviour at `:57-59`), `:71-93` (`SelfTest`).
- **Drain helper** — `backend/agent/src/agenttest/stream_kit_record.go:63`.
- **Landed client spec** — `openspec/specs/ai-provider-client/spec.md:57-59` (`R-APC-001`), `:66` (`S-APC-004`), `:83-101` (`R-APC-003`), `:100` (`S-APC-015`), `:159-169` (`R-APC-008`, "only by injection" at `:161`), `:190-209` (`R-APC-010`, the recorded-bite-proof grammar).
- **Landed scaffold spec** — `openspec/specs/agent-module-scaffold/spec.md:8` (dependency convention), `:20, 26` (AI-37 pre-authorised to amend), `:36-42` (`R-AGM-001`/`S-AGM-001`), `:64-66` (`R-AGM-003`, `go.work.sum`), `:91-95` (`R-AGM-005`).
- **Conformance capability closure** — `backend/agent/src/agenttest/conformance_suite.go:34-93`; AI-36's precedent `openspec/specs/ai-provider-conformance-suite/spec.md:395-403`.
- **Explore artefact** — `openspec/changes/cachicamas-ai-observability/explore.md`; Engram `#2701` (exploration) and `#2702` (dependency probe).
- **House-style precedent for this artefact** — `openspec/changes/archive/2026-08-08-cachicamas-ai-redaction/proposal.md`.
