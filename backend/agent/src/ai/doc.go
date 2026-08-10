// Package ai is Layer 1 of the cachicamas agent stack: the model adapter.
//
// It exposes provider-neutral contracts for what is sent to a language model and
// what comes back, so that Layer 2 can drive any supported vendor without
// importing a vendor SDK or seeing a wire type. Concretely, this package will own
// the normalized request, the content and tool vocabulary that crosses the model
// API, the streaming event contracts, the provider interface and its stream
// lifecycle, and the provider and transport error taxonomy. Concrete vendor
// adapters live in their own subpackages instead — openaicompat is the first —
// and this package owns the conformance tests and deterministic fakes that prove
// them against its contracts.
//
// It is empty today by design. The contract text grows one milestone at a time,
// and each milestone's documentation paragraph is guarded where it makes a
// checkable claim. See the Layer 1 task graph in
// docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md.
//
// # What this package must not own
//
// Agent turns, loop termination and transcript mutation; tool execution and
// tool-result scheduling; session history persistence; provider or model catalog
// files and user preference storage; environment-variable loading, login flows
// and secret persistence; any CLI, TUI, HTTP handler, slash command, skill,
// prompt or project instruction; and application-specific retries that could
// duplicate an agent turn. Those belong to Layers 2 and 3 and to the composition
// root.
//
// Two wording traps are worth stating, because each has already caused one wrong
// decision. First, "Layer 1 does not know what a tool is" is too broad: this
// package must understand the provider-neutral transport representation of tool
// declarations and tool calls, because those cross the model API. What it must
// not do is execute tools, resolve tool names, or own application behavior.
// Second, "a provider swap is a config change" applies only after adapters
// exist: switching between already-supported providers can be configuration
// only, but adding a new vendor still requires a new adapter unless that vendor
// is genuinely compatible with an existing transport.
//
// # The import rule
//
// This package may import the Go standard library, net/http, vendor model SDKs,
// and the OpenTelemetry API. It must not import the sibling layers
// (src/agent, src/coding, src/cmd), any package of another backend module, or
// the OpenTelemetry SDK, its exporters, or otelslog. The rule is
// ADR 0005 § D1 row 1 and § D3; it is enforced mechanically by the deny-by-default
// guard in import_boundary_test.go, so a dependency nobody thought to forbid by
// name still fails.
//
// The module carries zero dependencies today. One milestone may change that,
// and it needs its own ADR: AI-37 adds the OpenTelemetry API.
//
// # What comes back
//
// AI-14 adds the first contract for what a stream carries the other way: an
// [Event], its kind derived from a sealed payload the same way [Part] derives
// its own, and a per-stream [Sequence] with no process-global state — see
// sequence.go for the rule governing what a sequence means, and does not
// mean, across two different streams. This milestone registers no concrete
// event kind; AI-15 … AI-20 add those without editing this package's
// AI-14-owned contracts.
//
// # The provider boundary
//
// AI-20 closes the loop: [ModelProvider] is the one call every adapter
// offers — a cancellable context and a normalized [Request] in, a
// receive-only [Event] stream out — with the ownership, cancellation,
// buffering and failure-delivery rules ai-stream-lifecycle decided
// restated on its own GoDoc, and a pre-stream contract that validates req
// exactly once before consulting the context at all. [TokenCounter] is the
// one optional capability this package asks a provider value to advertise
// in v1, discovered only by asserting the value against it — never a
// widening of [ModelProvider] itself, mechanically pinned by the signature
// guard in src/agenttest. Concrete vendor adapters implementing
// [ModelProvider] arrive from AI-25 onward; this package owns the
// contract, never a vendor's satisfaction of it.
//
// # The first adapter's capability record
//
// AI-38.2 runs TestOpenRouterAdapter_FullConformance
// (src/ai/openaicompat/openrouter/conformance/run_for_test.go) — the
// AI-23 conformance suite's whole registered case table, unscoped,
// against a real *openaicompat.Client speaking real HTTP to a local
// httptest.Server — and asserts the generated nine-entry capability
// record, entry-for-entry, against the committed expectation in the same
// package's capability_record_test.go (expectedOpenRouterRecord). The
// table below is transcribed from that committed expectation — never
// regenerated, re-decided, or amended by this comment — and is protected
// against silent drift by a read-only test in the same conformance
// package (doc_matrix_guard_test.go).
//
//	CAP-R-01(streaming_text)      required  satisfied — text arrives as delimited blocks, reconstructible exactly
//	CAP-R-02(tool_calls)          required  satisfied — a tool call arrives with identity, name, exact argument bytes and an observable ordinal
//	CAP-R-03(completion_metadata) required  satisfied — a normally-finished stream ends carrying a finish reason and a usage record
//	CAP-R-04(cancellation)        required  satisfied — the caller-owned signal ends the stream within bounded time
//	CAP-R-05(typed_failures)      required  satisfied — every failure is classifiable through one vocabulary, on both delivery paths, and states whether content preceded it
//	CAP-O-01(reasoning_content)   optional  absent    — struck verdict (AI-29's decision.md); reopens only on R-OR-05/R-ACR-004's named triggers, with a new ADR required — never read as a permanent property of this adapter
//	CAP-O-02(token_counting)      optional  absent    — the discoverable optional token-counting contract is not offered by this adapter
//	CAP-O-03(cache_boundary)      optional  absent    — cache-boundary markers are not honored by this adapter
//	CAP-O-04(retry)               optional  satisfied — retryable pre-stream failures are auto-retried with bounded attempts
//
// Two publication duties delegated to this node (AI-29's decision.md
// § 11) are stated below, beside each other — neither in place of the
// other.
//
// Completion-checklist item 6's wire clause is not exercisable in v1:
// AI-26.6 landed as a refusal and AI-29.2 is struck by AI-29, so there is
// no reasoning on this wire to round-trip and no v1 node can close the
// wire half. The stream half of item 6, already closed by AI-17
// (R-ARE-009/R-ARE-010), is unaffected and is not reopened.
//
// Layer 2 MUST strip OpenRouter's reasoning_details field on the wire —
// a recorded absence (AI-29's decision.md), not an oversight.
//
// # The v1 surface freeze
//
// The v1 surface this package and its siblings expose is enumerated and
// declared frozen as of the cachicamas-ai-layer2-handoff change (doc
// 0002 § AI-40); see that change's decision.md for the full by-capability
// enumeration, the eighteen-item completion-checklist walk, and the
// never-cancelled abandoned-consumer posture.
package ai
