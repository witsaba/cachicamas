// Package ai is Layer 1 of the cachicamas_ai model adapter stack.
//
// It is the provider-neutral boundary between cachicamas and LLM vendors:
// given a normalized request, it returns a normalized stream of events
// (text deltas, tool-call deltas, usage, finish reason, terminal error).
// The package owns no agent loop, no tool execution, no session
// persistence, and no application-level behavior — those live in Layer 2
// (cachicamas_agent) and Layer 3 (cachicamas_coding).
//
// # Architectural anchors
//
//   - Dependency direction: cachicamas_coding -> cachicamas_agent -> cachicamas_ai.
//     This package may import only the Go standard library and vendor SDKs.
//     See ADR 0004 for the full dependency rule.
//   - Vocabulary: see AI-00 § stream, § event, § provider, § tool call (Layer 1 senses).
//   - Streaming contract: see AI-01 § Cancellation contract, § Closure contract,
//     § Error delivery contract.
//   - Capability scope: see AI-02 § Capability matrix (v1 = 10 REQUIRED,
//     5 OPTIONAL, 6 OUT-OF-V1).
//
// # Strict TDD
//
// Every future milestone that adds behavior here MUST follow strict
// red-green-refactor discipline: a failing test first, the minimum
// production code second, refactor while green. The import-boundary
// test in import_boundary_test.go is the canary for Layer 1 purity.
//
// As of AI-04 the package exposes two provider-neutral types:
//
//   - Role — typed string with a 4-value canonical set (system, user,
//     assistant, tool). Zero value is invalid. See role.go.
//   - Message — the wire turn sent to the model, carrying a Role and
//     an optional stable ID. Content parts and tool calls are added
//     by AI-05 and AI-08 respectively. See message.go.
//
// Subsequent milestones (AI-05..AI-38) introduce content parts,
package ai
