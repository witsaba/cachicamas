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
// As of AI-05 the package also exposes content-part types: Text
// (a string-typed value validated by NewText; see text.go) and
// ContentPart (a discriminated union with a Kind enum that reserves
// slots for reasoning, image, audio, tool declarations, tool calls,
// and tool results; see content.go). Message.Content is a
// []ContentPart slice so multi-part text and future variants are
// additive; subsequent milestones (AI-06..AI-38) introduce reasoning
// content, tool declarations, tool calls, the model request shape,
// finish reasons, usage, the event envelope, the stream shape, and
// concrete provider adapters — each in its own change.
//
// As of AI-06 the package exposes an optional Reasoning content-part
// type with three variants — ReasoningAbsent, ReasoningRedacted, and
// ReasoningStreamed. ContentPartFromReasoning is the sanctioned
// constructor that validates state first and payload per variant:
// Absent ignores its payload, Redacted caps at MaxReasoningSummaryLength,
// and Streamed requires non-empty, non-whitespace input capped at
// MaxReasoningStreamedLength. v1 Layer 1 adapters only emit
// ReasoningAbsent (see AI-02 § Reasoning policy); Redacted and Streamed
// are reserved for v1.1+ but kept in the type system for forward
// compatibility.
//
// As of AI-07 the package also exposes the provider-neutral tool
// declaration shape used at the model boundary: ToolDeclaration
// (name + description + input JSON Schema, structurally validated);
// NewToolDeclaration (sanctioned constructor); ValidateTools
// (collection helper for duplicate-name detection); and the
// MaxToolNameLength (64 bytes) and MaxToolDescriptionLength
// (1024 bytes) max-length constants; See AI-07 § Capability matrix
// for the structural-only validation scope.
package ai
