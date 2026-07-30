# Layer 1 milestone map — `cachicamas_ai` model adapter

> **Status:** In progress — **17 of 48** milestones shipped (AI-00 … AI-16). Last merged: AI-16 (PR #87, 2026-07-30). Phase C is complete; **Phase 0 (AI-39) is the next milestone**.
> **Amended:** 2026-07-30 by ADR 0005, ADR 0006, and the adversarial architecture review of the same date (Engram `obs #2243`). See [Phase 0](#phase-0--structural-corrections-retro-inserted-2026-07-30) and [Phase H](#phase-h--contract-gaps-found-by-the-2026-07-30-review).
> **Source decisions:** [ADR 0004](../../adr/0004-adopt-tau-3-layer-agentic-architecture.md) (original) · [ADR 0005](../../adr/0005-promote-agent-stack-to-own-module.md) (module boundary, dependency rule v2, v1 scope — supersedes 0004 in part) · [ADR 0006](../../adr/0006-resolve-skill-and-prompt-source-of-truth.md) (skill/prompt sources)
> **Architecture reference:** [cachicamas agent stack v2](../0001-cachicamas-agent-stack-v2.md)
> **Target module:** `backend/agent/` (`github.com/cachicamas/backend/agent`) — the package **moves** out of `backend/database_administrator/src/tools/agent/ai/` in AI-39.
> **Target package:** `backend/agent/src/ai/`
> **Purpose:** Split Layer 1 into small, dependency-ordered changes that can each be planned and delivered through a separate future SDD cycle.
>
> **Milestone identifiers are append-only.** AI-NN ids appear in ~37 source files' GoDoc, every commit message, 15 merged PR titles, ~200 test names, and Engram topic keys. Renumbering invalidates that audit trail for no benefit. New work appends AI-39 … AI-47; logical insertion points are expressed with a `Blocks:` field, not by renumbering.

## Outcome first

Completing this map produces a provider-neutral Go package that Layer 2 can call without importing any vendor SDK or vendor wire type. The package will accept a normalized model request and emit a normalized stream containing text, reasoning, tool-call, usage, completion, and failure information.

This document is a roadmap, **not a specification**. Each milestone deliberately leaves detailed API names and wire shapes to its own SDD proposal, spec, design, tasks, implementation, and verification cycle.

## Quick navigation

- [Current project and ADR analysis](#current-project-analysis)
- [Layer boundary](#layer-boundary)
- [Rules for future SDD milestones](#rules-for-every-future-sdd-milestone)
- [Dependency map](#dependency-map)
- [**Phase 0 — Structural corrections**](#phase-0--structural-corrections-retro-inserted-2026-07-30) *(added 2026-07-30 — AI-39 … AI-42)*
- [Phases A–G](#phase-a--contract-decisions)
- [**Phase H — Contract gaps**](#phase-h--contract-gaps-found-by-the-2026-07-30-review) *(added 2026-07-30 — AI-43 … AI-47)*
- [Recommended delivery sequence](#recommended-delivery-sequence)
- [Layer 1 completion checklist](#layer-1-completion-checklist)

## Current project analysis

| Finding | Evidence | Planning consequence |
| --- | --- | --- |
| ~~Layer 1 does not exist yet.~~ **Layer 1 exists and is 17 milestones deep.** *(updated 2026-07-30)* | `…/src/tools/agent/ai/` holds ~38 files and ~10.9k lines, with mechanical import-boundary guards. | Every remaining milestone is additive or **corrective** against a shipped contract. C1–C4 (AI-40, AI-41, AI-42, AI-18) correct contracts that contradict their own GoDoc. |
| The current `tools` package is not an agent foundation. | `src/tools/tools.go` is a build-tagged dependency pin. | Do not extend that file into the model adapter. **ADR 0005 vacates the name entirely — the agent stack leaves `src/tools/`, and `src/tools/tools.go` is untouched.** |
| ~~The backend is one Go module.~~ **The backend has three Go modules.** *(updated 2026-07-30)* | `database_administrator`, `workspace_syncer`, and — from AI-39 — `agent`. | Layer boundaries between L1/L2/L3 are Go package boundaries inside `backend/agent`; the boundary to the rest of the repo is a **module** boundary, which Go enforces. |
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
- Imports are governed by [ADR 0005 § D1](../../adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) and [§ D3](../../adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary): the Go standard library, the selected provider dependency, and the OpenTelemetry **API** modules only. The OTel **SDK**, any exporter, and every package of `database_administrator` / `workspace_syncer` remain forbidden.
- Public types need stable semantics before a concrete adapter depends on them.
- A milestone may refine later names, but it must not violate ADR 0004.
- Each SDD must state what remains intentionally unsupported.

## Dependency map

```mermaid
flowchart LR
    A[Phase A: Contract decisions] --> B[Phase B: Neutral value types]
    B --> C[Phase C: Stream contract]
    C --> Z[Phase 0: Structural corrections]
    Z --> D[Phase D: Validation and test kit]
    D --> Hh[Phase H: Contract gaps]
    Hh --> E[Phase E: First provider]
    E --> F[Phase F: Operational hardening]
    F --> G[Phase G: Layer 2 handoff]

    classDef added fill:#fee2e2,stroke:#b91c1c,color:#1f2937
    class Z,Hh added
```

The critical path is sequential at the phase level. Milestones marked parallel may run concurrently only after their shared dependencies are merged.

**Phase 0 and Phase H were retro-inserted on 2026-07-30.** They sit where they do because Phase 0 corrects contracts that Phase D would otherwise freeze into a conformance suite, and Phase H closes contract gaps that become roughly three times more expensive once Phase E's first adapter depends on them. Both are numbered out of alphabetical order because milestone identifiers are append-only — see the header.

---

## Phase 0 — Structural corrections (retro-inserted 2026-07-30)

These four milestones were identified by the adversarial architecture review **after AI-16 shipped**. They are scheduled **before AI-17** so that no further milestone deepens the current package path or builds a test kit on top of a contract that contradicts itself.

Nothing here adds capability. AI-39 is a move; AI-40 through AI-42 make three shipped contracts mean what their documentation already claims.

### AI-39 — Promote the agent stack to its own module

- **SDD change name:** `cachicamas-agent-module-promotion`
- **Goal:** Move the agent stack out of `database_administrator` into the sibling module `backend/agent`, per [ADR 0005 § D2](../../adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2).
- **Deliverable:** New module (`go.mod`, `Makefile`, `.golangci.yml`, `README.md`, `.gitignore`); `src/ai/` and `src/agenttest/` relocated; import paths rewritten; the forward import guard upgraded to a module allowlist and to `go list -deps`; a **new reverse guard** in `database_administrator`; a repo-root `go.work`.
- **Acceptance:** `make test` is green in every module; both import directions are mechanically guarded; `git log --follow` traverses the move; no behavior changes.
- **Depends on:** ADR 0005 (merged, PR #81); AI-16 (merged, PR #87). **Both satisfied — AI-39 is unblocked.**
- **Blocks:** AI-17 and everything after it.
- **Out of scope:** Adding any dependency, including OpenTelemetry. `go.mod` stays empty — Layer 1 is stdlib-only today.
- **Also in scope — three lint findings confirmed on 2026-07-30** when AI-16 landed. `make lint` reports `newStreamBuffer` and `validatePreStream` in `provider.go` as **unused**, plus unreachable code in `agenttest/consumer_test.go`. The two helpers are documented as *"Adapters MUST call this helper … so the pre-stream branch is mechanically identical across implementations"* — but they are **unexported**, and adapters live in other packages, so no adapter can ever call them. The comment two lines below each concedes it (*"Adapters may inline the same two-line check"*). The module move is the natural moment to resolve this, because it is the point at which adapters get a known home: export them, delete them, or relocate them. Note `make lint` already exits non-zero on `main` for unrelated pre-existing reasons (~56 issues, mostly `errcheck` in tests), so this milestone should state whether it is also fixing the baseline or only its own three.
- **Note:** The `git mv` and the import-path rewrite are **separate commits**. Committing the move with byte-identical content makes every file a 100 % similarity rename; combining them destroys `git blame` for ~38 files. The build is broken between those two commits — say so in the PR body.

### AI-40 — Make the event sequence per-stream

- **SDD change name:** `cachicamas-ai-per-stream-sequence`
- **Goal:** Close review finding **C3**. The sequence counter is a package-global atomic, so the contract's "the first event of every stream carries sequence 1" is achievable only for the first stream in a process. The same GoDoc block admits the contradiction.
- **Deliverable:** Per-stream, 1-based, contiguous, producer-assigned sequence, with the process-global counter removed.
- **Acceptance:** Two concurrent streams each start at 1 and are independently contiguous; gap detection becomes possible.
- **Depends on:** AI-39.
- **Blocks:** AI-19, AI-20, AI-21, AI-26.
- **Note:** Contract-breaking, and **it flips a green test**: the existing concurrent-stream test documents the cross-stream gaps as expected behavior and must be rewritten. Expect a reviewer to read that test as a spec. Likely over the review budget — plan a chained PR. One design worth the SDD's consideration, though the SDD owns the decision: keep the event constructors' signatures, have them emit sequence 0, and add a producer-owned stamper — that turns a break across every constructor into a test-only change.

### AI-41 — Make content parts readable from another package

- **SDD change name:** `cachicamas-ai-contentpart-accessors`
- **Goal:** Close review finding **C2**. The text, tool-call and tool-result wrappers are unexported and expose only their discriminator, so **a provider adapter in another package cannot read content back out of a request**.
- **Deliverable:** A readable accessor for every content-part variant, and a reconciliation of the two competing strategies now in the package — the reasoning type implements the part interface directly and is therefore inspectable; the other three are wrapped and are not.
- **Acceptance:** An external-package test can extract text, tool-call arguments and tool-result content from a constructed request.
- **Depends on:** AI-39.
- **Blocks:** **AI-24 (hard blocker — request translation is structurally impossible without this)**, AI-19, AI-21.
- **Note:** The reasoning type's own GoDoc already explains why direct implementation was chosen for inspectability, in terms that apply verbatim to the three types that actually carry payload data.

### AI-42 — Close the content-part construction bypass

- **SDD change name:** `cachicamas-ai-text-seal`
- **Goal:** Close review finding **C1**. The exported text value type satisfies the content-part interface directly, so its zero value is a valid part that passes message validation and bypasses every construction rule.
- **Deliverable:** A seal that actually seals, plus correction of the package comment that currently claims this is already prevented.
- **Acceptance:** A zero-value text part cannot reach a request; the claim in the package comment is true.
- **Depends on:** AI-39.
- **Parallel with:** AI-41.
- **Blocks:** AI-24.
- **Note:** Not theoretical — a zero-value text part serializes to an empty text block, which at least one provider rejects at the API.

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

> **Amended 2026-07-30** (finding G3): add **token counting** to the explicitly-optional capability list, discovered by type assertion rather than added to the provider interface. It is the prerequisite for Layer 2 context compaction, and compaction that estimates by character count is wrong by enough to matter.

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

> **Amended 2026-07-30** (finding G12c): the enum is frozen as shipped. `refusal` and `pause_turn` arrive **additively** via AI-46; do not reopen this milestone. Separately (finding G10): the usage type already carries cache-read, cache-write and reasoning token counts, so **the Layer 1 half of cost tracking is complete** — only a Layer 2 cost event and a Layer 3 price table remain.

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

> **Amended 2026-07-30** (finding G12a): **delta events are optional.** At least one provider delivers each tool call whole, in a single chunk. A call MUST be representable as start-then-end with zero deltas, and no consumer may require at least one delta before the end event. Acceptance gains that case.

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

> **Amended 2026-07-30. This milestone owns review finding C4.** The provider interface and the terminal-kind helper both declare that a stream ends with exactly one completion event **or** one error event — but no error payload type exists and the payload interface is sealed, so **no adapter can construct the error terminal today**. AI-16 shipped the contract before AI-18 enabled it.
>
> Deliverable gains: a terminal error payload implementing the sealed payload interface, its constructor, and its accessor. Deliverable also gains a **partial-output discriminator** (finding G8), so a mid-stream disconnect after emitted events is distinguishable from a pre-stream failure — that is the most common real-world failure and the one naive retry logic excludes.
>
> Acceptance gains: after this milestone an adapter can actually emit the terminal error the provider interface declares mandatory. `Depends on:` gains AI-16 (shipped).

- **SDD change name:** `cachicamas-ai-provider-errors`
- **Goal:** Normalize authentication, authorization, rate limit, unavailable, timeout, cancellation, malformed response, unsupported capability, and unknown failures.
- **Deliverable:** Typed errors carrying safe metadata and retry hints.
- **Acceptance:** Error strings never require secrets or full response bodies; wrapped causes remain inspectable.
- **Depends on:** AI-01, AI-16, AI-17.
- **Out of scope:** Automatic retry behavior.

### AI-19 — Build a scripted fake provider

> **Amended 2026-07-30:** `Depends on:` gains **AI-40** and **AI-41**. The fake must script a terminal error (impossible before AI-18), start each scripted stream at sequence 1 (impossible before AI-40), and inspect request content to assert what it received (impossible before AI-41).

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

> **Amended 2026-07-30:** the suite gains two required cases — a **tool call delivered whole, with zero delta events** (finding G12a), and a **mid-stream disconnect that preserves partial output** (finding G8). Both are behaviours a real provider exhibits and neither is exercised by a naive happy-path suite. `Depends on:` gains AI-40, since a conformance suite written against the process-global sequence counter would freeze that bug permanently.

- **SDD change name:** `cachicamas-ai-conformance-suite`
- **Goal:** Define behavior every concrete adapter must pass.
- **Deliverable:** Reusable contract tests for text, tools, completion, errors, cancellation, stream closure, and redaction.
- **Acceptance:** A provider factory can be plugged into the suite without copying assertions.
- **Depends on:** AI-18, AI-20.
- **Out of scope:** Live API credentials.

---

## Phase E — First concrete provider

### AI-22 — Select first provider and transport

> **Amended 2026-07-30:** `Depends on:` gains **AI-43, AI-44, AI-47** — all of Phase H must be closed first. Acceptance gains four questions the decision must answer explicitly, because each is a documented cross-provider divergence ([v2 architecture § 3.3](../0001-cachicamas-agent-stack-v2.md#33-the-provider-leakage-register)): how this provider expresses cache breakpoints (or whether it caches automatically); whether tool results are a block in a user-role message, a distinct role, or a nested object; whether an explicit output-token limit is mandatory; and whether the provider assigns tool-call identifiers at all.

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

> **Amended 2026-07-30:** `Depends on:` gains **AI-41 (hard blocker)**, AI-42 and AI-43. Until AI-41 lands, this milestone is **structurally impossible** — the content-part wrappers are unexported and expose only their discriminator, so translation code in the adapter package cannot read the text, tool-call arguments or tool-result content out of the request it is translating.
>
> Acceptance gains: consecutive same-role messages are merged where the provider enforces strict alternation; a mandatory output-token limit is supplied from a documented default rather than silently truncating; and synthetic tool-call identifiers, if the provider assigns none, are minted here and recorded so they survive session reload.

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

> **Amended 2026-07-30** (finding G12b): `Depends on:` gains **AI-45**. The provider's reasoning round-trip token must be captured here and preserved byte-exact — never parsed, never reformatted. Acceptance gains a round-trip test proving the bytes are unchanged.

- **SDD change name:** `cachicamas-ai-provider-reasoning-stream`
- **Goal:** Implement the chosen optional reasoning behavior for the first provider.
- **Deliverable:** Mapping, unsupported behavior, or documented capability absence.
- **Acceptance:** Provider reasoning never leaks into text events; conformance behavior matches AI-06/14.
- **Depends on:** AI-14, AI-26.
- **Parallel with:** AI-28 where adapter internals permit.

### AI-28 — Translate tool-call stream

> **Amended 2026-07-30** (finding G5): acceptance gains that the tool-call **ordinal survives normalization**, so Layer 2 can restore results in call order regardless of completion order — several providers require tool results to correspond positionally to their calls. If the first provider needs an explicit index to make this work, the SDD promotes it to the AI-15 payload as an **additive** change. Also gains the zero-delta case from AI-15's amendment.

- **SDD change name:** `cachicamas-ai-provider-tool-stream`
- **Goal:** Map fragmented and interleaved provider tool calls into neutral events.
- **Deliverable:** Tool-call mapping and reconstruction tests.
- **Acceptance:** Multiple interleaved calls preserve identity, order, and exact argument bytes; malformed completion yields a typed failure.
- **Depends on:** AI-15, AI-26.
- **Out of scope:** JSON argument validation against the tool schema and tool execution.

### AI-29 — Translate usage and finish reasons

> **Amended 2026-07-30** (finding G12c): `Depends on:` gains **AI-46**. Refusal and pause-turn map to their own finish reasons, not to the unknown fallback. Acceptance gains that mapping.

- **SDD change name:** `cachicamas-ai-provider-completion`
- **Goal:** Complete terminal metadata mapping for the first provider.
- **Deliverable:** Usage, finish reason, unknown-value, and partial-metadata handling.
- **Acceptance:** Terminal events contain every available normalized field and never invent unavailable usage.
- **Depends on:** AI-10, AI-26.
- **Parallel with:** AI-27 and AI-28 after AI-26.

### AI-30 — Map HTTP and provider failures

> **Amended 2026-07-30** (finding G8): a mid-stream disconnect must produce a **terminal error event** carrying AI-18's partial-output discriminator, not merely a returned error. The distinction matters because the harness's retry decision depends on whether anything was already emitted.

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

> **Amended 2026-07-30** (finding G8): make explicit what the current wording only implies. The **partial-output case is never retried at Layer 1** — it is handed up as a typed error and the harness decides. Today the acceptance clause says only what must *not* happen; it must also say what *does*. This matters because a stream that dies after emitting output is the single most common real-world failure, and the naive retry predicate ("retry if nothing was emitted") is precisely the one that excludes it.

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

> **Amended 2026-07-30** (finding S5): reword to [ADR 0005 § D3](../../adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary). Layer 1 **may** import the OpenTelemetry **API** and **must not** import the OTel SDK, any exporter, or `database_administrator/src/otel`. The existing acceptance clause "Layer 1 does not import Cachicamas `otel`" stays literally true and becomes precise rather than accidental. Deliverable gains the § D3 attribute **allowlist** (GenAI semantic conventions) and its **absolute denylist**: no prompt, completion, reasoning, tool-argument or tool-result text, no header, no credential, no raw response body.

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

---

## Phase H — Contract gaps found by the 2026-07-30 review

**Every milestone in this phase must land before AI-22.** Each is a breaking change to a contract that no adapter yet depends on. Once the first adapter exists, each becomes roughly three times the work, because the contract and every adapter must migrate in lock-step. The architectural reasoning for each is in [the v2 architecture reference § 3.2](../0001-cachicamas-agent-stack-v2.md#32-what-must-change-before-a-vendor-adapter-exists); the v1 verdicts are in [ADR 0005 § D4](../../adr/0005-promote-agent-stack-to-own-module.md#d4--v1-scope-for-cross-cutting-concerns).

### AI-43 — Make cache breakpoints expressible

- **SDD change name:** `cachicamas-ai-cache-breakpoints`
- **Goal:** Close gap **G4**. The system instruction is a flat string, so there is nowhere to mark a prompt-cache boundary.
- **Deliverable:** Ordered, markable segments for the system instruction, plus breakpoint markers on tool declarations and messages. Markers are advisory — an adapter for an auto-caching provider ignores them.
- **Acceptance:** A request can express a breakpoint set that honours the vendor cap on breakpoint count and the tools → system → messages invalidation ordering, and an adapter can render or ignore it.
- **Depends on:** AI-41, AI-42.
- **Blocks:** **AI-22, AI-24**.
- **Out of scope:** Any measurement of actual cache hit rate. Also **out of scope: usage reporting** — the usage type already carries the cache-read and cache-write token counts.

### AI-44 — Add per-request options and a provider escape hatch

- **SDD change name:** `cachicamas-ai-request-extension-points`
- **Goal:** Close gap **G9**. Generation options are fixed at request construction, and there is no way to carry a provider-specific field the neutral vocabulary does not model.
- **Deliverable:** Per-request options, a typed-but-opaque provider pass-through, and copy-on-write rebuilding so a caller can derive a modified request from an existing one.
- **Acceptance:** A provider-specific field survives to its adapter without any other adapter needing to know it exists; a request can be rebuilt without mutating the original.
- **Depends on:** AI-43.
- **Blocks:** AI-22, AI-24; and Layer 2's pre-request hook.
- **Note:** The design principle is deliberate and worth restating in the SDD — the correct response to provider divergence is a typed pass-through, **not** a wider neutral vocabulary. Every field added to the neutral model for one provider becomes a field every other adapter must ignore.

### AI-45 — Carry a reasoning round-trip token

- **SDD change name:** `cachicamas-ai-reasoning-roundtrip`
- **Goal:** Close gap **G12(b)**. Reasoning content exposes its state and its text and cannot carry a provider blob that must be returned byte-identical.
- **Deliverable:** An opaque round-trip token on reasoning content, never interpreted by cachicamas, preserved exactly through normalization and session persistence.
- **Acceptance:** A reasoning block survives a full round trip byte-for-byte; nothing in Layer 1 parses or reformats it.
- **Depends on:** AI-41.
- **Blocks:** AI-27.
- **Note:** Correctness, not metadata. At least one provider signs thinking blocks cryptographically; if the signature is not returned exactly, multi-turn extended thinking with tool use fails.

### AI-46 — Add refusal and pause finish reasons

- **SDD change name:** `cachicamas-ai-finishreason-refusal-pause`
- **Goal:** Close gap **G12(c)**. Refusal and pause-turn both collapse into the unknown fallback.
- **Deliverable:** Two additional finish reasons, added additively to the frozen enum.
- **Acceptance:** Layer 2 can distinguish "the model declined", "the model paused, resume it", and "I do not recognise this provider string" — three states with three different correct responses.
- **Depends on:** —
- **Blocks:** AI-29.
- **Note:** This is a loop-termination bug, not a cosmetic gap.

### AI-47 — Close the stream-carrier decision

- **SDD change name:** `cachicamas-ai-stream-carrier-decision`
- **Goal:** Close gap **G13**. Re-evaluate the receive-only channel against a range-over-func iterator at the package boundary, **before** a concrete adapter exists.
- **Deliverable:** **A decision, no production code.**
- **Acceptance:** The decision is recorded with its rationale, and it is closed before AI-22 starts.
- **Depends on:** AI-16.
- **Blocks:** AI-22.
- **Documented default: keep channels**, and expose an iterator view from the AI-20 test kit for ergonomics. The canonical objection — a consumer who stops reading strands the producer goroutine — is already closed by the v1 contract, which selects on cancellation for every send and makes the caller's context the sole liveness signal. What remains is a caller who abandons the stream *and* never cancels, which is a contract violation rather than a design flaw. Against that, switching carriers now would invalidate AI-16's interface signature guard and its behavioural scenarios, merged days ago.

## Recommended delivery sequence

| Wave | Milestones | Exit condition |
| --- | --- | --- |
| 1 — Decide | AI-00 to AI-02 | ✅ **Shipped.** Vocabulary, lifecycle, and v1 capabilities are unambiguous. |
| 2 — Model | AI-03 to AI-10 | ✅ **Shipped.** Neutral request values compile and validate independently. |
| 3 — Stream | AI-11 to AI-16 | ✅ **Shipped** (AI-16 in PR #87). Layer 2-facing provider interface is defined without vendor leakage. |
| **3.5 — Relocate** | **AI-39** | The agent stack is its own module and **both** import directions are mechanically guarded. |
| **3.6 — Correct** | **AI-40 to AI-42** | No shipped contract contradicts its own documentation. Content is readable from another package. |
| 4 — Prove contracts | AI-17 to AI-21 | Errors — including a constructible terminal error — fake provider, test helpers, and conformance suite are reusable. |
| **4.5 — Close contract gaps** | **AI-43 to AI-47** | Cache breakpoints, per-request options, reasoning round-trip and stop reasons are expressible; the stream carrier is decided. |
| 5 — Connect vendor | AI-22 to AI-30 | First adapter streams normalized text/tools/metadata and maps failures. |
| 6 — Harden | AI-31 to AI-35 | Cancellation, pressure, retries, redaction, and observability are safe. |
| 7 — Hand off | AI-36 to AI-38 | Adapter passes conformance and Layer 2 can consume the stable v1 API. |

Waves 3.5, 3.6 and 4.5 were inserted on 2026-07-30. Their placement is not arbitrary: 3.5 and 3.6 must precede wave 4, or the test kit and conformance suite freeze the current defects into assertions; 4.5 must precede wave 5, or every Phase H change has to migrate an existing adapter in lock-step.

## Next SDD to start

Start with **AI-39 — `cachicamas-agent-module-promotion`**. Its preconditions are satisfied: ADR 0005 merged (PR #81) and AI-16 merged (PR #87). Run it on a quiet tree — it is mechanical, but it rewrites the import path in 18 files and conflicts with anything else in flight.

Then **AI-40**, before the test kit exists — a conformance suite written against the process-global sequence counter freezes that bug permanently. Then **AI-41** and **AI-42** in parallel; AI-41 is a hard blocker on AI-24, so it cannot slip.

Only then AI-17 and the original Phase D ordering.

> The original guidance, retained because it still explains *why* the ordering matters: the model adapter is a boundary. If the boundary vocabulary and stream ownership are vague, provider details harden into accidental architecture and every later layer pays for it. The 2026-07-30 review is that principle applied to what already shipped — four contracts hardened into shapes their own documentation contradicts, and they are cheapest to correct now, while no adapter depends on them.

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
- [ ] The package lives in `backend/agent` and `src/tools/` is vacated. *(added 2026-07-30)*
- [ ] **Both** import directions are mechanically guarded, not just Layer 1 purity. *(added 2026-07-30)*
- [ ] Event sequence is per-stream and provably starts at 1 for every stream. *(added 2026-07-30)*
- [ ] Every content-part variant is readable from another package. *(added 2026-07-30)*
- [ ] Cache breakpoints are expressible, and per-request provider options have an escape hatch. *(added 2026-07-30)*
- [ ] Provider round-trip tokens survive byte-exact through normalization and persistence. *(added 2026-07-30)*

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

### Named Layer 2 / Layer 3 forward requirements *(added 2026-07-30)*

These are deferred **with a reserved seam**, not merely unscheduled. Each is placed in
[the v2 architecture reference § 6](../0001-cachicamas-agent-stack-v2.md#6-the-twelve-seams-that-must-exist-now)
and dispositioned in [§ 7](../0001-cachicamas-agent-stack-v2.md#7-forward-requirements-register). None
requires Layer 1 work, which is why they appear here rather than as milestones — but all of them
shape Layer 2's design, so none should be rediscovered later as a surprise.

| ID | Deferred requirement | Owner |
| --- | --- | --- |
| G1 | Permission as a suspendable protocol on the event stream, with allow-once / allow-always / deny / modify-input | L2 protocol, L3 policy |
| G2 | Sandboxed tool execution, applied to the whole spawned process tree | L3 |
| G3 | Context compaction that protects recent turns, never orphans a call/result pair, and is recoverable | L2 |
| G5 | Parallel tool execution with call-ordered re-join | L2 |
| G6 | Dynamic, supervised tool sources | L3 |
| G7 | Subagents as a harness invoked from a tool | L2 |
| G10 | Cost as first-class events plus a price table — **the Layer 1 half is already done** | L2 emits, L3 prices |
| G11 | Hook taxonomy: pre-request, pre-compact, post-turn, session-start. Observers never synchronous on the streaming path | L2 + L3 |
