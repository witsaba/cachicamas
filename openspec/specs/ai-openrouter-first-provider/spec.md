# Spec — OpenRouter as the first concrete AI provider for Layer 1

> **Introduced by**: `add-openrouter-first-provider` (AI-24…AI-32, Waves 4–5; archived 2026-08-06) · **Modified by**: `cachicamas-ai-adapter-conformance` (AI-38) · `cachicamas-ai-live-smoke` (AI-39, Wave 6)
> **Status**: live
> **Capability**: `ai-openrouter-first-provider` (new)
> **Requirement IDs**: `R-OR-0NN` · **Scenario IDs**: `S-OR-0NN`
> **Composes (read-only, NOT modified)**: `ai-provider-client` (AI-25) · `ai-provider-conformance-suite` (AI-23/38) · `ai-stream-testkit` (AI-22) · `ai-model-provider` (AI-20)
> **Archived version**: `openspec/changes/archive/2026-08-06-add-openrouter-first-provider/` · **Current modification**: `openspec/changes/archive/2026-08-09-cachicamas-ai-live-smoke/`

---

## Purpose

ADR 0004 (Engram #1997) established the three-layer agentic architecture; AI-20 (Engram #2235) froze the `ModelProvider` interface every concrete adapter must satisfy; AI-24 (Engram #2432) pre-decided vendor = OpenAI-compatible Chat Completions streaming dialect with raw `net/http` transport and zero `go.mod` requires. The shipped `backend/agent/src/ai/openaicompat/` package is vendor-agnostic but has **no first concrete vendor**, so AI-38 (full deterministic adapter conformance) and AI-39 (opt-in live smoke) have no subject, and AI-40 (Layer 2 readiness handoff) cannot publish. This capability concretizes **OpenRouter** as that first vendor — composing (not re-implementing) the shipped `openaicompat` package, `agenttest.RunConformance`, and the `ai-stream-testkit` helpers. It ships as three chained PRs under a no-merge tracker: wrapper → conformance bridge → live smoke.

## Status

This specification is **live** as of 2026-08-09. It was introduced by `add-openrouter-first-provider` (AI-24…AI-32) and modified by `cachicamas-ai-adapter-conformance` (AI-38) and `cachicamas-ai-live-smoke` (AI-39). Requirements R-OR-07 and R-OR-08 were modified in AI-39 to establish the live smoke test's two-stage opt-in gate, internal package placement, setup instructions, and capture-sink sweep binding. It is canonical at `/openspec/specs/ai-openrouter-first-provider/spec.md`.

---

## ADDED Requirements

### R-OR-01 — Wrapper construction is injection-only

The wrapper SHALL accept the OpenRouter base URL, an opaque bearer credential, a model identifier, and an optional HTTP client — all by injection. The wrapper SHALL NOT read any environment variable, touch the filesystem, or spawn a process (AI-25.2 invariant). The wrapper SHALL construct an `openaicompat.Client` via the shipped `openaicompat.New(Config{...})` constructor.

#### Scenario: Construction with injected values

- GIVEN a wrapper built with the OpenRouter URL, opaque bearer, model `openai/gpt-4o`, no client
- WHEN the wrapper's non-test sources are scanned for ambient authority
- THEN no `os.Getenv`, `os/exec`, or filesystem call is present
- AND the underlying `openaicompat.Client` is the one the injected values configured.

#### Scenario: Construction rejects invalid configuration

- GIVEN a wrapper built with an empty (zero-value) bearer credential
- WHEN construction runs
- THEN it fails with an AI-04 typed failure naming the credential input
- AND no outbound request is made.

### R-OR-02 — Attribution headers are wrapper-injected, never openaicompat-injected

The wrapper SHALL inject `HTTP-Referer`, `X-OpenRouter-Title` (alias `X-Title`), and `X-OpenRouter-Categories` on every outbound request via a wrapper-owned `http.RoundTripper`. Empty strings SHALL suppress a header. The shipped `openaicompat` package SHALL remain unaware of these headers; its "`Authorization` and `Content-Type` are the only headers it sets" rule SHALL stay intact.

#### Scenario: All three headers observed when non-empty

- GIVEN a stub-transport probe with all three attribution strings non-empty
- WHEN the stub reads the outbound request
- THEN `HTTP-Referer`, `X-OpenRouter-Title`/`X-Title`, and `X-OpenRouter-Categories` are each present with the injected values.

#### Scenario: Empty strings suppress the headers

- GIVEN a stub-transport probe with all three attribution strings empty
- WHEN the stub reads the outbound request
- THEN none of the three headers is present.

#### Scenario: openaicompat's header surface is unmodified

- GIVEN the merged wrapper
- WHEN `openaicompat/request.go` is read
- THEN it does not name any of the three attribution headers and still sets only `Authorization` and `Content-Type`.

### R-OR-03 — Default model and deliberate-model field

The conformance bridge SHALL target `openai/gpt-4o` (non-reasoning, paid) by default. The wrapper SHALL expose a deliberate-model field on its `Config` so model swaps do not require a code change at construction sites.

#### Scenario: Bridge uses the documented default

- GIVEN a fresh conformance-bridge factory
- WHEN the factory declares the model
- THEN the declared model equals `openai/gpt-4o`.

#### Scenario: Config carries a deliberate-model field

- GIVEN the wrapper's `Config` shape
- WHEN a reader enumerates the fields
- THEN a model-identifier field is present and defaults to `openai/gpt-4o`.

### R-OR-04 — `stream_options.include_usage` stays set

The wrapper SHALL always emit `stream_options.include_usage = true` in the wire body. OpenRouter's deprecation of the field SHALL be documented but not acted on; dropping the field would diverge the wire body from other openaicompat-target vendors (vLLM, Ollama, LocalAI).

#### Scenario: Field is present in the outbound body

- GIVEN a stub-transport probe
- WHEN the stub reads the request body
- THEN `stream_options.include_usage` equals `true`.

### R-OR-05 — Capability-record assertion: the record is generated, and `CAP-O-01` reasoning is absent under the default model

The capability record asserted for the OpenRouter bridge SHALL be the record a real **unscoped** conformance run emits, not the factory's declared capability pointers. It SHALL record `CAP-O-01` (reasoning) as `absent` under the default model `openai/gpt-4o`. Switching the default to a reasoning-capable model SHALL require an explicit ADR AND a spec amendment that reopens AI-29's struck verdict under trigger #1 — never a silent default swap. A **generated** `CAP-O-01 = satisfied` SHALL block the change and escalate as that trigger, and SHALL NOT be absorbed by amending the committed expectation.
(Previously: the requirement did not state that the asserted record must be generated by a real run, which permitted a declared-pointer check to stand in for it.)

> **Amended 2026-08-09 (AI-38)** by `cachicamas-ai-adapter-conformance`: the requirement now binds the record to a real unscoped run rather than a declared-pointer check, and states the reasoning-`satisfied` escalation as a hard block rather than leaving it implicit. `verify-report.md`'s WARN-B is closed at this promotion: the "Default-model swap" scenario below is corrected to name the actual enforcement mechanism — the capability-record test itself is model-insensitive (`factory.Reasoning` is hard-coded `false`), so the prior text's claim that it fails on a model swap was inaccurate.

#### Scenario: Default-model record equals `absent`

- GIVEN the conformance bridge running the unscoped conformance suite with the default factory
- WHEN the generated record is read
- THEN `CAP-O-01` carries the outcome `absent`
- AND the record is the run's emitted record, not a read of the factory's declared pointers
- AND the capability-record test passes.

#### Scenario: Default-model swap does not happen silently

- GIVEN the wrapper's `Config`
- WHEN the default model is changed without reopening AI-29
- THEN `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` fails, pinned against the `openrouterDefaultModel` constant, until the spec amendment lands
- AND the capability-record test itself is unaffected by the swap, because `CAP-O-01`'s asserted outcome is model-insensitive (`factory.Reasoning` is hard-coded `false`).

#### Scenario: A generated reasoning `satisfied` blocks rather than updating the expectation

- GIVEN a run that emits `CAP-O-01` as `satisfied`
- WHEN the record is asserted against the committed expectation
- THEN the assertion fails naming the AI-29 reopen trigger
- AND the committed expectation is not amended to accommodate the observed outcome.

### R-OR-06 — The conformance bridge runs the unscoped suite against the OpenRouter wrapper over regenerable transcripts

The conformance bridge SHALL run the **unscoped** conformance entry point end-to-end against the OpenRouter wrapper using OpenRouter-shaped transcripts. All five required capabilities SHALL pass (`CapStreamingText`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTerminal`), with no skip and no waiver. The generated record SHALL be asserted entry by entry against a committed **nine-entry** expectation in which `CAP-O-01`, `CAP-O-02` and `CAP-O-03` are `absent` and `CAP-O-04` (retry) is `satisfied`. Every transcript the bridge replays that has a neutral-event preimage SHALL be regenerable by the recording helper from captured wire bytes and SHALL be byte-identical to that helper's output; a transcript hand-authoring a vendor wire-extension field with no `ai.Event` equivalent (OpenRouter's `reasoning` / `reasoning_details`) has no such preimage and is explicitly excluded from the round-trip, named as such in the recorder test rather than silently omitted. The reasoning-extension scenario SHALL be cited as covering OpenRouter's `reasoning` / `reasoning_details` extension fields.
(Previously: "at least the five required capabilities", the scoped-vs-unscoped distinction left implicit, the record stated as AI-24 §8's `absent × 3`, transcripts unconstrained in origin, and "every transcript" regenerable with no exception for a hand-authored vendor-extension fixture with no `ai.Event` preimage — verify-report.md's own SUG-1 named this gap.)

> **Amended 2026-08-09 (AI-38)** by `cachicamas-ai-adapter-conformance`: the run is now bound to the unscoped entry point, the expectation grows to nine entries with `CAP-O-04` satisfied (superseding the stale "absent × 3" phrasing, which predates AI-35's `CAP-O-04`), transcripts are bound to the recording helper's drift guard, and the vendor-extension exclusion is named explicitly.

#### Scenario: Required capabilities all pass under an unscoped run

- GIVEN the OpenRouter bridge factory
- WHEN the unscoped conformance suite runs against it
- THEN every required capability case executes and passes
- AND no case is skipped or waived
- AND the run's verdict is a pass.

#### Scenario: The nine-entry record matches the committed expectation

- GIVEN the record emitted by that run
- WHEN it is compared entry by entry with the committed expectation
- THEN all nine entries match
- AND `CAP-O-01`, `CAP-O-02` and `CAP-O-03` are `absent`
- AND `CAP-O-04` is `satisfied`.

#### Scenario: Transcripts are regenerable and drift-guarded

- GIVEN each transcript the bridge replays that has a neutral-event preimage, and the recording helper's output for the same captured wire bytes
- WHEN the drift guard compares them
- THEN they are byte-identical
- AND a hand-edited transcript fails the guard naming the transcript
- AND a transcript with no neutral-event preimage (a hand-authored vendor wire-extension fixture) is explicitly excluded from this comparison, named as an exclusion rather than silently omitted.

#### Scenario: Reasoning extension field is dropped, not leaked, not failed

- GIVEN an OpenRouter transcript carrying a `delta.reasoning_details` extension field
- WHEN the conformance bridge decodes it
- THEN no reasoning byte appears in any text event
- AND the stream does not fail
- AND the reasoning-extension test passes.

### R-OR-07 — Live smoke is opt-in via environment variables only, with no CI workflow file

The live smoke (`TestOpenRouterAdapter_LiveSmoke`) SHALL be `t.Skip`-gated unless BOTH
`OPENROUTER_API_KEY` is present in the test process AND `RUN_LIVE_OPENROUTER_SMOKE` equals the exact
literal `1`; the credential alone SHALL NOT be sufficient consent. The skip reason SHALL attribute
which stage closed the gate. `make test` in `backend/agent/` SHALL NOT depend on OpenRouter or on any
network credential. **No CI workflow file is required or created by this spec**: the repository's
established posture is that no `.github/workflows/` directory exists (ADR 0005 § Enforcement; doc 0002
"No CI exists"), so the live smoke is opt-in for human, local runs only. The smoke package SHALL live
under a path containing an `internal` segment beneath `.../openaicompat/openrouter/`, and
credential-safe setup instructions SHALL ship in that package directory. The full live-smoke contract
is stated in the `ai-live-smoke` capability (`R-LSM-001`…`R-LSM-008`); this requirement is its
vendor-side anchor.
(Previously: the gate was stated as `OPENROUTER_API_KEY` alone, and the requirement mandated a
`.github/workflows/agent-openrouter-smoke.yml` `workflow_dispatch`-only file. That file does not
exist, cannot exist under ADR 0005's no-CI posture, and was never created; its scenario is therefore
retired with the clause that required it rather than left as an unsatisfiable obligation. The
`internal` placement and the shipped setup instructions are new, from AI-39.1 items 4 and 5.)

> **Amended 2026-08-09 (AI-39)** by `cachicamas-ai-live-smoke`: the gate is restated as the shipped
> two-stage environment-variable opt-in; the CI-workflow obligation is removed as superseded by facts
> on disk, adopting the text the orphan `add-openrouter-first-provider` change folder already carried;
> `internal` placement and shipped setup instructions are added. The removed
> "Workflow file is dispatch-only" scenario is retired deliberately, not lost — it asserted properties
> of a file this repository will never create.

#### Scenario: Skip path exercised without the env vars

- GIVEN `OPENROUTER_API_KEY` OR `RUN_LIVE_OPENROUTER_SMOKE=1` absent from the test process
- WHEN `make test` runs
- THEN the live smoke is skipped with a reason naming the missing stage
- AND no outbound request is made.

#### Scenario: The credential alone does not open the gate

- GIVEN `OPENROUTER_API_KEY` present and `RUN_LIVE_OPENROUTER_SMOKE` set to any value other than the
  exact literal `1`
- WHEN the gate is evaluated
- THEN it reports closed and the test skips
- AND no outbound request is made.

#### Scenario: No CI workflow file is introduced

- GIVEN the merged repository
- WHEN it is searched for a `.github/workflows/` file triggering the live smoke
- THEN none exists
- AND no scheduled, push, or pull-request trigger of a billing-consuming run exists.

#### Scenario: The package is internal and ships its own setup instructions

- GIVEN the merged smoke package
- WHEN its path and directory contents are read
- THEN the path contains an `internal` segment beneath `.../openaicompat/openrouter/`
- AND setup instructions naming both env vars, shell `export` only, the exact post-move invocation,
  the timeout bound and the per-run cost are present in that directory
- AND no step in them writes the credential to a file inside the repository.

### R-OR-08 — Credential redaction extends to the live smoke's own captured output, proven by a positive control

The live smoke SHALL NOT log the credential, the secret value, or the full prompt, on the success path
or on any failure path. It SHALL route every diagnostic it produces through a single capture sink and
SHALL run the shared sentinel sweep over that sink before any of those bytes reaches the test log. The
deny list SHALL include the literal `OPENROUTER_API_KEY` env-var name, the secret value's 4-character
prefix, and the planted prompt marker. The sweep's positive control (`sweep.SelfTest`) SHALL pass over
that deny list before a clean sweep result is trusted; a clean result obtained without the positive
control SHALL count as no result, per `R-CNF-027` clause 5. A sentinel match SHALL fail the test,
naming only the vector and stating that output was withheld, and SHALL NOT reproduce the matched
bytes. The smoke package's scan SHALL delegate to the module's single sweep implementation and SHALL
be proven equivalent to it, so `S-CNF-080`'s "both reach that one implementation" holds after the
package is relocated.
(Previously: the sweep was required to exist and to catch a planted leak, but nothing bound it to the
live test's own captured output, the failure paths, or the mandatory positive control — so the "even
on failure" clause was an untested intention and a clean result could pass vacuously.)

> **Amended 2026-08-09 (AI-39)** by `cachicamas-ai-live-smoke`: the sweep is now bound to the live
> run's own capture sink on every path, the `sweep.SelfTest` positive control becomes mandatory before
> a clean result is trusted, and the single-implementation convergence obligation is restated so it
> survives the move to `.../openrouter/internal/smoke`.

#### Scenario: Sentinel sweep catches a deliberate leak mutation

- GIVEN a mutated smoke test that calls `t.Logf(key)` with the credential
- WHEN the sentinel sweep runs
- THEN the test fails and names the leak vector without reprinting the credential.

#### Scenario: `openaicompat.Credential` redaction carries through

- GIVEN the wrapper's credential value
- WHEN it is rendered through `String()`, `GoString()`, `MarshalJSON`, or default formatting
- THEN no rendering contains the token.

#### Scenario: The sweep runs over the live run's captured output on every path

- GIVEN a credentialled live run that ends by success, by request failure, by drain failure, or by a
  failed stream-shape invariant
- WHEN the run finishes
- THEN the sweep has run over the captured diagnostics for that path before any of them reached the
  test log
- AND a clean sweep releases the captured diagnostics in full.

#### Scenario: The positive control gates the clean result

- GIVEN the deny list built for a live run
- WHEN `sweep.SelfTest` is forced to fail over it
- THEN the test fails rather than reporting a clean sweep
- AND no clean-sweep claim is made without the control having passed.

#### Scenario: One sweep implementation, reached from both sides after the move

- GIVEN the merged module with the smoke package under `.../openrouter/internal/smoke`
- WHEN the sweep implementations are enumerated and both the smoke-side scan and the suite-side scan
  are run over the same corpus and planted needle
- THEN exactly one implementation exists
- AND both are proven equivalent to it, with every previously published name still published.

### R-OR-09 — AI-00.3 forward guard stays green

The wrapper SHALL add no new top-level Go dependency. `backend/agent/go.mod` SHALL declare zero `require` lines after the change. `TestLayer1_ModuleHasNoDependencies_ZeroRequires` SHALL pass unmodified. `allowedNonStdlibPrefixes` SHALL remain at one entry (the module itself).

#### Scenario: Module has no new requires

- GIVEN the change merged
- WHEN `backend/agent/go.mod` is read
- THEN it declares zero `require` lines
- AND both AI-00 import guards pass.

### R-OR-10 — Out-of-scope fence (negative SHALLs)

The wrapper SHALL NOT support an Anthropic native adapter, SHALL NOT widen AI-32 to map OpenRouter's `error_type` discriminator, SHALL NOT emit a `reasoning_content` (or `reasoning` / `reasoning_details`) extension field, and SHALL NOT introduce any new `go.mod` require.

#### Scenario: Negative fences are mechanical

- GIVEN the merged wrapper sources and `go.mod`
- WHEN a reviewer looks for an Anthropic-only type, an `error_type` map, a `reasoning_content` render, or any new `require`
- THEN none is present.

## Decisions

| ID | Decision | Basis |
|---|---|---|
| **C1** | Default model = `openai/gpt-4o` (non-reasoning, paid) | Preserves AI-29's struck verdict; explore §2.7 |
| **C2** | Conformance + smoke use a paid model (no `:free` suffix) | `:free` rate-limit flakiness; explore §2.10 |
| **C3** | Attribution headers wrapper-injected, NOT openaicompat-injected | Keeps openaicompat's header surface narrow; explore §2.3, Engram #2571 |
| **C4** | `stream_options.include_usage` stays set | Cross-vendor wire-body uniformity; explore §2.5 |
| **Q1** | Conformance sweep = `openai/gpt-4o` only | Sweeping Anthropic passthrough reopens AI-29 under trigger #1 |
| **Q2** | Layer 3 reads env, passes opaque bearer into `Config.Credential` | AI-25.2 invariant; wrapper reads nothing (memory #2432 §3) |
| **Q3** | Wrapper placement = sibling sub-package `openaicompat/openrouter/` | AI-25.2 call-site scan scope stays clean; Engram #2571 |
| **Q4** | Conformance bridge ships in PR #2 of this change | AI-38 first-concrete-adapter charter; doc 0002 lines 2241–2277 |
| **Q5** | ~~Live smoke opt-in via `workflow_dispatch` + repo secret `OPENROUTER_API_KEY`~~ — superseded at AI-39: env-var-only two-stage gate, no CI workflow (see R-OR-07) | AI-39.1 charter; doc 0002 lines 2279–2299 |
| **Q6** | Three chained PRs under no-merge tracker | 800-line PR budget per `sdd-phase-common.md §E`; natural work-unit split |

## Traceability

- **Proposal**: `openspec/changes/archive/2026-08-06-add-openrouter-first-provider/proposal.md` · Engram observation **#2570** (`sdd/add-openrouter-first-provider/proposal`)
- **Explore**: `openspec/changes/archive/2026-08-06-add-openrouter-first-provider/explore.md` · Engram topic `sdd/add-openrouter-first-provider/explore` (obs #2568)
- **Wrapper-placement decision**: Engram observation **#2571** (`decision/openrouter-wrapper-placement`)
- **Composed shipped specs** (read-only, NOT modified):
  - [`openspec/specs/ai-provider-client/spec.md`](../ai-provider-client/spec.md) (AI-25) — provider client construction
  - [`openspec/specs/ai-provider-conformance-suite/spec.md`](../ai-provider-conformance-suite/spec.md) (AI-23/38) — conformance runner + capability record
  - [`openspec/specs/ai-stream-testkit/spec.md`](../ai-stream-testkit/spec.md) (AI-22) — drain, ordering, leak, redaction helpers
  - [`openspec/specs/ai-model-provider/spec.md`](../ai-model-provider/spec.md) (AI-20) — `ModelProvider` interface + signature guard
- **Milestone charters** (doc 0002 — Layer 1 task graph): AI-25 lines 1447–1491 · AI-38 lines 2241–2277 · AI-39 lines 2279–2299
- **AI-24 pre-decision** (vendor/transport, OpenAI-compatible Chat Completions + raw `net/http` + zero `go.mod` requires): Engram observation **#2432**
- **AI-29 struck verdict** (2026-08-04, reasoning stream absent): `openspec/changes/archive/2026-08-04-cachicamas-ai-provider-reasoning-stream/decision.md` §5 row 4, §7, §9 (triggers)
- **ADR 0004** (3-layer agentic architecture): Engram observation **#1997** + `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`
- **Backend test runner**: Engram observation **#2055** (`cd backend/agent && make test` = `go test -race -v ./...`)
- **OpenSpec rules**: `openspec/config.yaml` `rules.specs` — Given/When/Then, RFC 2119 keywords, independently verifiable scenarios

## Per-PR coverage map

| PR | Branch chain | Milestone slice | Requirements covered |
|---|---|---|---|
| **PR #1 — wrapper** | `feat/openrouter-wrapper` → `tracker/add-openrouter-first-provider` | AI-25.1 + AI-25.3 | R-OR-01, R-OR-02, R-OR-03, R-OR-04, R-OR-09, R-OR-10 |
| **PR #2 — conformance bridge** | `feat/openrouter-conformance-bridge` → PR #1's branch | AI-38.1 + AI-38.2 + AI-38.3 | R-OR-05, R-OR-06 |
| **PR #3 — live smoke** | `feat/openrouter-live-smoke` → PR #2's branch | AI-39.1 | R-OR-07, R-OR-08 |