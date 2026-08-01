// Package ai is Layer 1 of the cachicamas agent stack: the model adapter.
//
// It exposes provider-neutral contracts for what is sent to a language model and
// what comes back, so that Layer 2 can drive any supported vendor without
// importing a vendor SDK or seeing a wire type. Concretely, this package will own
// the normalized request, the content and tool vocabulary that crosses the model
// API, the streaming event contracts, the provider interface and its stream
// lifecycle, the provider and transport error taxonomy, the concrete vendor
// adapters, and the conformance tests and deterministic fakes that prove them.
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
// The module carries zero dependencies today. Two milestones may change that,
// and each needs its own ADR: one selects a transport, one adds the
// OpenTelemetry API.
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
package ai
