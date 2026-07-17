# Layer 1 milestone map — `cachicamas_ai` model adapter

> **Status:** Planning document; no implementation has started.
> **Source decision:** [ADR 0004 — Adopt 3-Layer Agentic Architecture](../../adr/0004-adopt-tau-3-layer-agentic-architecture.md)
> **Target package:** `backend/database_administrator/src/tools/agent/ai/`
> **Purpose:** Split Layer 1 into small, dependency-ordered changes that can each be planned and delivered through a separate future SDD cycle.

## Outcome first

Completing this map produces a provider-neutral Go package that Layer 2 can call without importing any vendor SDK or vendor wire type. The package will accept a normalized model request and emit a normalized stream containing text, reasoning, tool-call, usage, completion, and failure information.

This document is a roadmap, **not a specification**. Each milestone deliberately leaves detailed API names and wire shapes to its own SDD proposal, spec, design, tasks, implementation, and verification cycle.

## Quick navigation

- [Current project and ADR analysis](#current-project-analysis)
- [Layer boundary](#layer-boundary)
- [Rules for future SDD milestones](#rules-for-every-future-sdd-milestone)
- [Dependency map](#dependency-map)
- [Phases A–G](#phase-a--contract-decisions)
- [Recommended delivery sequence](#recommended-delivery-sequence)
- [Layer 1 completion checklist](#layer-1-completion-checklist)

## Current project analysis

| Finding | Evidence | Planning consequence |
| --- | --- | --- |
| Layer 1 does not exist yet. | `backend/database_administrator/src/tools/agent/` is absent. | Start with contracts and package boundaries; do not begin with a vendor SDK. |
| The current `tools` package is not an agent foundation. | `src/tools/tools.go` is a build-tagged dependency pin. | Do not extend that file into the model adapter. Create the ADR-defined package. |
| The backend is one Go module. | `backend/database_administrator/go.mod` declares `github.com/cachicamas/backend/database_administrator`, Go 1.26.3. | Layer boundaries are Go package boundaries inside the existing module. |
| No LLM SDK is currently declared. | `go.mod` has HTTP, database, auth, OTel, retry, and YAML dependencies, but no model-provider SDK. | Provider and transport selection must be an explicit milestone, not an accidental dependency addition. |
| Existing adapters favor explicit transport code. | `src/infrastructure/github/client.go` uses `context.Context`, `net/http`, typed projections, injected clients/base URLs, and typed errors. | Layer 1 should preserve those testability and security conventions unless an SDD records a reason not to. |
| Tests are colocated with Go packages. | Existing `*_test.go` files use unit and `httptest` patterns. | Every behavior milestone should land with tests in the same SDD change. |
| OpenSpec is already initialized. | `openspec/` exists. | Later milestones can each become a small named SDD change without modifying this roadmap. |

## ADR 0004 analysis

### Decisions already settled

The following are architectural constraints, not questions for later milestones:

1. Dependencies flow only `cachicamas_coding -> cachicamas_agent -> cachicamas_ai`; the arrows represent imports, and the reverse direction is forbidden.
2. Layer 1 owns provider-specific request/stream translation.
3. Layer 1 exposes provider-neutral contracts to Layer 2.
4. Layer 1 may import only the Go standard library and provider-specific dependencies.
5. Layer 1 must not import Layer 2, Layer 3, or application/domain/infrastructure packages.
6. Streaming uses Go primitives rather than copying Python async-iterator APIs.
7. Frontends and session persistence are outside Layer 1.

### Gaps that must be resolved before Layer 2 can depend on Layer 1

ADR 0004 correctly sets the dependency direction, but it intentionally does not define the complete Go contract. The milestones below must resolve these gaps:

- Which normalized message, content-part, tool-declaration, tool-call, usage, and finish-reason types Layer 1 owns.
- Whether a stream is a receive-only channel, an iterator-like object, or a channel plus a terminal result.
- How stream errors differ from provider-emitted error events.
- Who creates and closes channels, and what happens on cancellation or an unread consumer.
- Buffer size and backpressure behavior.
- Whether reasoning/thinking content is first-class, optional metadata, or unsupported initially.
- How fragmented tool-call arguments are reconstructed without exposing vendor chunks.
- Which errors are retryable and when retrying a partially emitted stream is unsafe.
- Which provider is implemented first and whether it uses `net/http` or an SDK.
- How conformance is tested so every future provider emits equivalent normalized semantics.

### Two wording traps to avoid

1. **“Layer 1 does not know what a tool is” is too broad.** Layer 1 must understand the provider-neutral *transport representation* of tool declarations and tool calls because those cross the model API. It must not execute tools, resolve tool names, or own application behavior.
2. **“Provider swap is a config change” applies only after adapters exist.** Switching between already-supported providers can be configuration-only. Adding a new vendor still requires a new adapter unless it is genuinely compatible with an existing transport.

## Layer boundary

### Layer 1 owns

- Provider-neutral model request contracts.
- Provider-neutral message and content-part contracts used at the model boundary.
- Provider-neutral tool declaration and emitted tool-call contracts.
- Provider-neutral streaming event contracts.
- Provider interface and stream lifecycle contract.
- Provider/transport error taxonomy.
- Concrete provider adapters.
- Adapter conformance tests and deterministic fakes.

### Layer 1 must not own

- Agent turns, loop termination, or transcript mutation.
- Tool execution or tool-result scheduling.
- Session history persistence.
- Provider/model catalog files or user preference storage.
- Environment-variable loading, login flows, or secret persistence.
- CLI, TUI, HTTP handlers, slash commands, skills, prompts, or project instructions.
- Application-specific retries that could duplicate an agent turn.

## Rules for every future SDD milestone

Each milestone below should become its **own change** unless its SDD exploration proves it is too small to review independently.

- One primary contract or behavior per milestone.
- Tests travel with the behavior they prove.
- Prefer less than 250 changed lines; stop and reassess before 400.
- No production secrets or live-network dependency in normal tests.
- No imports from outside `stdlib`, `tools/agent/ai`, and the selected provider dependency.
- Public types need stable semantics before a concrete adapter depends on them.
- A milestone may refine later names, but it must not violate ADR 0004.
- Each SDD must state what remains intentionally unsupported.

## Dependency map

```mermaid
flowchart LR
    A[Phase A: Contract decisions] --> B[Phase B: Neutral value types]
    B --> C[Phase C: Stream contract]
    C --> D[Phase D: Validation and test kit]
    D --> E[Phase E: First provider]
    E --> F[Phase F: Operational hardening]
    F --> G[Phase G: Layer 2 handoff]
```

The critical path is sequential at the phase level. Milestones marked parallel may run concurrently only after their shared dependencies are merged.

---

## Phase A — Contract decisions

### AI-00 — Record the Layer 1 contract vocabulary

- **SDD change name:** `cachicamas-ai-contract-vocabulary`
- **Goal:** Define the words used by every later spec: provider, model, request, message, content part, event, stream, tool declaration, tool call, usage, finish reason, and provider error.
- **Deliverable:** A short contract note or delta spec; no production Go code.
- **Acceptance:** Every term has one meaning, ownership is assigned to a layer, and tool transport is clearly separated from tool execution.
- **Depends on:** ADR 0004.
- **Out of scope:** Go field names, provider choice, adapter code.

### AI-01 — Decide stream lifecycle and ownership

- **SDD change name:** `cachicamas-ai-stream-lifecycle`
- **Goal:** Lock the Go streaming model before defining events.
- **Deliverable:** Decision covering producer/consumer ownership, channel direction, channel closure, terminal error delivery, cancellation, buffering, and slow/abandoned consumers.
- **Acceptance:** The design answers who closes what, exactly once; no goroutine is allowed to block forever after `ctx.Done()`.
- **Depends on:** AI-00.
- **Out of scope:** Concrete event structs and HTTP streaming.

### AI-02 — Decide the minimum first-provider capability set

- **SDD change name:** `cachicamas-ai-minimum-capabilities`
- **Goal:** Prevent the neutral contract from becoming either OpenAI-shaped or an abstract superset nobody can implement.
- **Deliverable:** Required v1 capabilities and explicitly optional capabilities.
- **Acceptance:** Text streaming, tool calls, completion reason, usage, cancellation, and errors have a clear required/optional status; images, audio, reasoning, structured output, and multimodality are explicitly classified.
- **Depends on:** AI-00.
- **Out of scope:** Selecting the provider.

---

## Phase B — Provider-neutral value types

### AI-03 — Introduce package scaffold and import boundary

- **SDD change name:** `cachicamas-ai-package-scaffold`
- **Goal:** Create `src/tools/agent/ai` with package documentation and an enforceable dependency boundary.
- **Deliverable:** Minimal compiling package plus an import-boundary test or equivalent guard.
- **Acceptance:** `go test ./src/tools/agent/ai/...` passes; the package imports no Cachicamas package outside itself.
- **Depends on:** AI-00.
- **Out of scope:** Provider interface, events, SDKs.

### AI-04 — Define roles and message identity

- **SDD change name:** `cachicamas-ai-message-roles`
- **Goal:** Represent provider-neutral message roles and stable message identity.
- **Deliverable:** Role and message-shell types with validation tests.
- **Acceptance:** Supported roles and invalid/unknown role behavior are explicit; zero values cannot silently become valid wire requests.
- **Depends on:** AI-02, AI-03.
- **Out of scope:** Content parts and tool calls.

### AI-05 — Define text content parts

- **SDD change name:** `cachicamas-ai-text-content`
- **Goal:** Add the smallest content representation needed for text input and output.
- **Deliverable:** Text content type(s), construction/validation rules, JSON behavior only if part of the public contract.
- **Acceptance:** Empty, whitespace-only, and multi-part text behavior is specified and tested.
- **Depends on:** AI-04.
- **Out of scope:** Images, audio, files, reasoning.

### AI-06 — Define optional reasoning content

- **SDD change name:** `cachicamas-ai-reasoning-content`
- **Goal:** Model reasoning/thinking without pretending every provider supports or exposes it.
- **Deliverable:** Optional neutral representation and capability rule, or an explicit v1 deferral.
- **Acceptance:** Layer 2 can distinguish absent, redacted, and streamed reasoning without inspecting vendor metadata.
- **Depends on:** AI-02, AI-05.
- **Out of scope:** Rendering or persistence policy.

### AI-07 — Define tool declarations

- **SDD change name:** `cachicamas-ai-tool-declarations`
- **Goal:** Represent a tool name, description, and input JSON Schema at the model boundary.
- **Deliverable:** Tool declaration types, the v1 schema-validation scope, and validation tests. Any new third-party JSON Schema dependency requires its own ADR before it enters `go.mod`.
- **Acceptance:** Empty/duplicate names, malformed schema JSON, and unsupported schema features have deterministic behavior; no execution callback enters Layer 1.
- **Depends on:** AI-03.
- **Out of scope:** Full JSON Schema evaluation unless separately approved, tool execution, and built-in coding tools.

### AI-08 — Define tool calls and tool results

- **SDD change name:** `cachicamas-ai-tool-messages`
- **Goal:** Represent assistant tool requests and model-visible tool results in normalized messages.
- **Deliverable:** Tool-call identity, argument payload, result correlation, and error-result representation.
- **Acceptance:** Multiple calls per response and call/result correlation are testable without vendor IDs leaking upward.
- **Depends on:** AI-04, AI-07.
- **Out of scope:** Executing or authorizing a call.

### AI-09 — Define normalized model request

- **SDD change name:** `cachicamas-ai-model-request`
- **Goal:** Bundle model, system instruction, messages, tools, and supported generation options into one request value.
- **Deliverable:** Request type, validation rules, and immutable/copying policy.
- **Acceptance:** Missing model, invalid history, duplicate tools, and unsupported option combinations fail before network I/O.
- **Depends on:** AI-05, AI-07, AI-08.
- **Out of scope:** Provider credentials and catalog lookup.

### AI-10 — Define finish reasons and usage

- **SDD change name:** `cachicamas-ai-completion-metadata`
- **Goal:** Normalize why generation ended and how usage is reported.
- **Deliverable:** Finish-reason and usage types with unknown-provider-value behavior.
- **Acceptance:** Stop, length/limit, tool-call, content filter/refusal, cancellation, and unknown are distinguishable where providers expose them.
- **Depends on:** AI-02, AI-03.
- **Out of scope:** Billing calculations and quotas.

---

## Phase C — Stream contract

### AI-11 — Define event envelope and ordering invariants

- **SDD change name:** `cachicamas-ai-event-envelope`
- **Goal:** Create the common event envelope without yet adding every event kind.
- **Deliverable:** Event identity, sequence/order metadata if needed, event-kind discrimination, and ordering rules.
- **Acceptance:** Consumers can exhaustively distinguish supported event kinds; illegal orderings are documented.
- **Depends on:** AI-01, AI-03.
- **Out of scope:** HTTP chunks and agent-level events.

### AI-12 — Add response lifecycle events

- **SDD change name:** `cachicamas-ai-response-events`
- **Goal:** Represent response start and terminal completion.
- **Deliverable:** Start/completed event payloads carrying normalized identifiers and completion metadata.
- **Acceptance:** Exactly one start and at most one successful terminal event are permitted per stream.
- **Depends on:** AI-10, AI-11.
- **Out of scope:** Text and tool deltas.

### AI-13 — Add text delta events

- **SDD change name:** `cachicamas-ai-text-events`
- **Goal:** Stream normalized text increments without exposing vendor chunk structures.
- **Deliverable:** Text start/delta/end semantics or a documented simpler equivalent.
- **Acceptance:** Concatenating ordered deltas reconstructs the final text exactly, including Unicode boundaries.
- **Depends on:** AI-05, AI-11, AI-12.
- **Out of scope:** TUI rendering and final transcript mutation.

### AI-14 — Add reasoning delta events

- **SDD change name:** `cachicamas-ai-reasoning-events`
- **Goal:** Stream optional reasoning under the policy chosen in AI-06.
- **Deliverable:** Reasoning event semantics or an explicit no-op/unsupported contract.
- **Acceptance:** Reasoning can never be mistaken for user-visible answer text.
- **Depends on:** AI-06, AI-11, AI-12.
- **Parallel with:** AI-13 after shared dependencies.

### AI-15 — Add tool-call delta events

- **SDD change name:** `cachicamas-ai-tool-call-events`
- **Goal:** Normalize tool-call start, fragmented arguments, and completion.
- **Deliverable:** Tool-call event payloads and reconstruction rules.
- **Acceptance:** Interleaved multiple calls reconstruct independently; malformed/incomplete arguments terminate predictably.
- **Depends on:** AI-08, AI-11, AI-12.
- **Parallel with:** AI-13 and AI-14 after shared dependencies.

### AI-16 — Define provider interface

- **SDD change name:** `cachicamas-ai-model-provider`
- **Goal:** Expose the smallest interface Layer 2 needs to request a stream.
- **Deliverable:** `ModelProvider`-equivalent interface and compile-time consumer example/test.
- **Acceptance:** It accepts `context.Context` and the normalized request, exposes only normalized stream semantics, and has no vendor types.
- **Depends on:** AI-09, AI-12, AI-13, AI-15.
- **Out of scope:** Provider registry, catalog, and model selection UI.

---

## Phase D — Validation, errors, and deterministic test kit

### AI-17 — Define validation error taxonomy

- **SDD change name:** `cachicamas-ai-validation-errors`
- **Goal:** Separate caller-contract failures from network/provider failures.
- **Deliverable:** Typed validation errors with stable `errors.Is`/`errors.As` behavior.
- **Acceptance:** Invalid requests fail before a stream goroutine or HTTP request starts.
- **Depends on:** AI-09.
- **Out of scope:** Provider status-code mapping.

### AI-18 — Define provider error taxonomy

- **SDD change name:** `cachicamas-ai-provider-errors`
- **Goal:** Normalize authentication, authorization, rate limit, unavailable, timeout, cancellation, malformed response, unsupported capability, and unknown failures.
- **Deliverable:** Typed errors carrying safe metadata and retry hints.
- **Acceptance:** Error strings never require secrets or full response bodies; wrapped causes remain inspectable.
- **Depends on:** AI-01, AI-16, AI-17.
- **Out of scope:** Automatic retry behavior.

### AI-19 — Build a scripted fake provider

- **SDD change name:** `cachicamas-ai-fake-provider`
- **Goal:** Let Layer 1 and future Layer 2 tests script events, delays, failures, and cancellation without network access.
- **Deliverable:** Test-only or exported testing package implementing the provider interface.
- **Acceptance:** Tests can script a text response, tool call, terminal error, blocked stream, and context cancellation deterministically.
- **Depends on:** AI-16, AI-18.
- **Out of scope:** Mocking a particular vendor wire format.

### AI-20 — Build stream recording and assertion helpers

- **SDD change name:** `cachicamas-ai-stream-testkit`
- **Goal:** Make event-order and goroutine-lifecycle assertions concise and repeatable.
- **Deliverable:** Package-importable drain/record helpers, timeout-safe assertions, and leak-sensitive test patterns; unlike AI-19, this milestone provides assertions rather than a provider implementation.
- **Acceptance:** A broken producer cannot hang the test suite indefinitely; event differences are readable.
- **Depends on:** AI-19.
- **Out of scope:** General-purpose testing framework.

### AI-21 — Create provider conformance suite

- **SDD change name:** `cachicamas-ai-conformance-suite`
- **Goal:** Define behavior every concrete adapter must pass.
- **Deliverable:** Reusable contract tests for text, tools, completion, errors, cancellation, stream closure, and redaction.
- **Acceptance:** A provider factory can be plugged into the suite without copying assertions.
- **Depends on:** AI-18, AI-20.
- **Out of scope:** Live API credentials.

---

## Phase E — First concrete provider

### AI-22 — Select first provider and transport

- **SDD change name:** `cachicamas-ai-first-provider-decision`
- **Goal:** Choose the first vendor/protocol and `net/http` versus SDK using evidence.
- **Deliverable:** Decision covering capability fit, streaming quality, testability, dependency weight, endpoint configurability, maintenance, and credential handling boundary.
- **Acceptance:** The decision names one first adapter and explains rejected alternatives. If the transport adds a top-level `go.mod` dependency, its artifact must include or be promoted to the ADR required by `openspec/AGENTS.md` before AI-23 adds that dependency.
- **Depends on:** AI-02, AI-21.
- **Out of scope:** Adapter implementation.

### AI-23 — Add provider configuration value and client construction

- **SDD change name:** `cachicamas-ai-provider-client`
- **Goal:** Construct the adapter from injected endpoint, credential/token source, HTTP client, and safe defaults.
- **Deliverable:** Adapter shell with testable constructor.
- **Acceptance:** Tests inject `httptest.Server`; invalid endpoint/config fails early; the adapter does not read environment variables itself.
- **Depends on:** AI-22.
- **Out of scope:** Catalog files, login, persistence, request sending.

### AI-24 — Translate normalized requests to provider wire requests

- **SDD change name:** `cachicamas-ai-request-translation`
- **Goal:** Map system instructions, messages, text, tools, and options into the selected provider request.
- **Deliverable:** Pure translation code and golden/table tests.
- **Acceptance:** No credential appears in serialized test fixtures; unsupported normalized features fail explicitly rather than being dropped silently.
- **Depends on:** AI-09, AI-23.
- **Out of scope:** Network execution and stream parsing.

### AI-25 — Implement streaming frame decoder

- **SDD change name:** `cachicamas-ai-stream-decoder`
- **Goal:** Decode the selected transport framing independently from semantic event mapping.
- **Deliverable:** Incremental decoder for split frames, multiple frames per read, keep-alives, EOF, and malformed data.
- **Acceptance:** Arbitrary read boundaries do not change decoded frames; malformed/truncated frames return typed failures.
- **Depends on:** AI-18, AI-23.
- **Parallel with:** AI-24 after AI-23.

### AI-26 — Translate response lifecycle and text

- **SDD change name:** `cachicamas-ai-provider-text-stream`
- **Goal:** Map provider response-start, text deltas, and completion into neutral events.
- **Deliverable:** End-to-end `httptest` stream for a text-only response.
- **Acceptance:** Event order satisfies AI-11/12/13 and the conformance text cases pass.
- **Depends on:** AI-24, AI-25.
- **Out of scope:** Tool calls and reasoning.

### AI-27 — Translate reasoning stream

- **SDD change name:** `cachicamas-ai-provider-reasoning-stream`
- **Goal:** Implement the chosen optional reasoning behavior for the first provider.
- **Deliverable:** Mapping, unsupported behavior, or documented capability absence.
- **Acceptance:** Provider reasoning never leaks into text events; conformance behavior matches AI-06/14.
- **Depends on:** AI-14, AI-26.
- **Parallel with:** AI-28 where adapter internals permit.

### AI-28 — Translate tool-call stream

- **SDD change name:** `cachicamas-ai-provider-tool-stream`
- **Goal:** Map fragmented and interleaved provider tool calls into neutral events.
- **Deliverable:** Tool-call mapping and reconstruction tests.
- **Acceptance:** Multiple interleaved calls preserve identity, order, and exact argument bytes; malformed completion yields a typed failure.
- **Depends on:** AI-15, AI-26.
- **Out of scope:** JSON argument validation against the tool schema and tool execution.

### AI-29 — Translate usage and finish reasons

- **SDD change name:** `cachicamas-ai-provider-completion`
- **Goal:** Complete terminal metadata mapping for the first provider.
- **Deliverable:** Usage, finish reason, unknown-value, and partial-metadata handling.
- **Acceptance:** Terminal events contain every available normalized field and never invent unavailable usage.
- **Depends on:** AI-10, AI-26.
- **Parallel with:** AI-27 and AI-28 after AI-26.

### AI-30 — Map HTTP and provider failures

- **SDD change name:** `cachicamas-ai-provider-error-mapping`
- **Goal:** Convert transport/status/body failures into the AI-18 taxonomy.
- **Deliverable:** Tests for auth, permission, rate limit, unavailable, timeout, malformed body, unexpected status, and mid-stream disconnect.
- **Acceptance:** Response bodies are size-limited and sanitized; retry hints and status metadata are preserved safely.
- **Depends on:** AI-18, AI-25.
- **Parallel with:** AI-26 after shared decoder dependencies.

---

## Phase F — Operational hardening

### AI-31 — Prove cancellation and goroutine cleanup

- **SDD change name:** `cachicamas-ai-cancellation`
- **Goal:** Ensure cancellation works before headers, between chunks, during a blocked send, and after completion.
- **Deliverable:** Cancellation-safe producer logic and deterministic tests.
- **Acceptance:** Requests stop, channels close once, no send occurs forever without observing context, and no goroutine leak is detected.
- **Depends on:** AI-26, AI-28, AI-30.
- **Out of scope:** Agent-level stop UX.

### AI-32 — Lock backpressure and buffer behavior

- **SDD change name:** `cachicamas-ai-backpressure`
- **Goal:** Implement the AI-01 decision with measurements rather than arbitrary channel sizes.
- **Deliverable:** Buffer constant/configuration, slow-consumer tests, and rationale.
- **Acceptance:** Ordering is stable, memory is bounded, and cancellation unblocks a saturated producer.
- **Depends on:** AI-31.
- **Out of scope:** Dropping text or tool-call events; those must remain lossless.

### AI-33 — Define retry and idempotency policy

- **SDD change name:** `cachicamas-ai-retry-policy`
- **Goal:** Decide and implement only retries that cannot duplicate a partially observed response.
- **Deliverable:** Explicit pre-stream retry conditions, backoff bounds, `Retry-After` handling, or a documented no-auto-retry v1 policy.
- **Acceptance:** No automatic retry occurs after any semantic event has been emitted unless the protocol provides a proven resume mechanism.
- **Depends on:** AI-30, AI-31.
- **Out of scope:** Agent-turn retries and fallback to another model.

### AI-34 — Enforce secret redaction and safe diagnostics

- **SDD change name:** `cachicamas-ai-redaction`
- **Goal:** Prevent credentials, authorization headers, sensitive prompt bodies, and unbounded provider errors from entering logs/errors.
- **Deliverable:** Safe diagnostic metadata and adversarial redaction tests.
- **Acceptance:** Sentinel secrets do not appear in errors, logs, fixtures, event metadata, or test failure output.
- **Depends on:** AI-23, AI-30.
- **Parallel with:** AI-31 through AI-33 after error mapping.

### AI-35 — Add adapter observability boundary

- **SDD change name:** `cachicamas-ai-observability`
- **Goal:** Expose enough safe timing/request metadata for callers without coupling Layer 1 to application telemetry policy.
- **Deliverable:** Decision and minimal hooks/attributes, preferably through context and safe result metadata.
- **Acceptance:** Layer 1 does not import Cachicamas `otel`; model content and secrets are not recorded by default.
- **Depends on:** AI-29, AI-30, AI-34.
- **Out of scope:** Dashboards, exporters, and application tracing setup.

---

## Phase G — Integration proof and Layer 2 handoff

### AI-36 — Run full deterministic adapter conformance

- **SDD change name:** `cachicamas-ai-adapter-conformance`
- **Goal:** Run the reusable AI-21 suite against the first concrete adapter through `httptest`.
- **Deliverable:** One deterministic end-to-end test matrix covering text, reasoning policy, tools, usage, all terminal paths, errors, cancellation, and closure.
- **Acceptance:** The first adapter passes every required capability; optional capability results are recorded explicitly.
- **Depends on:** AI-27 through AI-35.
- **Out of scope:** Real vendor network calls.

### AI-37 — Add opt-in live smoke test

- **SDD change name:** `cachicamas-ai-live-smoke`
- **Goal:** Prove the real provider still matches recorded assumptions without making CI depend on credentials.
- **Deliverable:** Explicitly gated test infrastructure, such as an internal smoke-test package unreachable from application `cmd/` entry points, with safe setup instructions.
- **Acceptance:** It skips cleanly without credentials, uses a bounded request, has a timeout, and never prints the credential or full sensitive prompt.
- **Depends on:** AI-36.
- **Out of scope:** A user-facing CLI, production deployment, and scheduled billing-consuming CI.

### AI-38 — Publish Layer 2 readiness contract

- **SDD change name:** `cachicamas-ai-layer2-handoff`
- **Goal:** Freeze the v1 surface that `cachicamas_agent` may consume.
- **Deliverable:** Package examples, compatibility statement, supported-capability matrix, and a fake-provider example for future `AgentLoop` tests.
- **Acceptance:** A tiny external-package test can construct a request, invoke a fake provider, drain events, handle cancellation/errors, and compile without vendor imports.
- **Depends on:** AI-36; AI-37 may remain optional.
- **Out of scope:** Implementing `AgentLoop` or `AgentHarness`.

## Recommended delivery sequence

| Wave | Milestones | Exit condition |
| --- | --- | --- |
| 1 — Decide | AI-00 to AI-02 | Vocabulary, lifecycle, and v1 capabilities are unambiguous. |
| 2 — Model | AI-03 to AI-10 | Neutral request values compile and validate independently. |
| 3 — Stream | AI-11 to AI-16 | Layer 2-facing provider interface is defined without vendor leakage. |
| 4 — Prove contracts | AI-17 to AI-21 | Errors, fake provider, test helpers, and conformance suite are reusable. |
| 5 — Connect vendor | AI-22 to AI-30 | First adapter streams normalized text/tools/metadata and maps failures. |
| 6 — Harden | AI-31 to AI-35 | Cancellation, pressure, retries, redaction, and observability are safe. |
| 7 — Hand off | AI-36 to AI-38 | Adapter passes conformance and Layer 2 can consume the stable v1 API. |

## First SDD to start later

Start with **AI-00 — `cachicamas-ai-contract-vocabulary`**, not with SDK installation or HTTP streaming. Then run AI-01 and AI-02 before creating public Go types.

That ordering matters: the model adapter is a boundary. If the boundary vocabulary and stream ownership are vague, provider details will harden into accidental architecture, and every later layer will pay for it.

## Layer 1 completion checklist

- [ ] Package exists at the ADR-defined location.
- [ ] Import direction is mechanically guarded.
- [ ] Neutral request/message/content/tool contracts are documented and tested.
- [ ] Event order and stream ownership are explicit.
- [ ] Cancellation cannot leak goroutines.
- [ ] Backpressure is bounded and lossless.
- [ ] Provider interface exposes no vendor type.
- [ ] Error taxonomy is typed, safe, and inspectable.
- [ ] Fake provider supports deterministic Layer 2 tests.
- [ ] Conformance suite can be reused for every future adapter.
- [ ] First concrete adapter passes deterministic conformance tests.
- [ ] Secrets and sensitive bodies are absent from diagnostics by default.
- [ ] Live test is optional and bounded.
- [ ] Layer 2 handoff example compiles without vendor dependencies.

## Explicitly deferred until after Layer 1

- `AgentLoop` and `AgentHarness`.
- Agent-level events such as `AgentStart`, `TurnStart`, and tool-execution events.
- Tool execution.
- `CodingSession` and JSONL persistence.
- Provider catalog and model-selection UI.
- Skills, project instructions, slash commands, CLI, TUI, and print mode.
- Multi-provider fallback/routing.
- Cost policy, quota management, and organization billing.
- Production rollout.
