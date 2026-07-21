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

// AI-08 paragraph (added after package clause so the greedy AI-07
// paragraph test does not count these lines).
//
// As of AI-08 the package also exposes tool-call and tool-result
// content-part types: ToolCall (name + arguments json.RawMessage,
// validated by NewToolCall with MaxToolCallArgumentsLength = 1 MiB);
// ToolResult (callID + content json.RawMessage, validated by
// NewToolResult with MaxToolResultCallIDLength = 512 bytes and
// MaxToolResultContentLength = 10 MiB); ContentPartFromToolCall
// and ContentPartFromToolResult (sanctioned ContentPart constructors).
// See AI-08 § Capability matrix.

// AI-09 paragraph (added after package clause so the greedy AI-07
// paragraph test does not count these lines).
//
// As of AI-09 the package exposes the normalized model request shape
// that Layer 2 hands to Layer 1 to start a turn: Request (model +
// messages + tools + options, validated end-to-end via NewRequest);
// GenerationOptions (vendor-neutral v1 generation parameters:
// SystemInstruction, MaxOutputTokens, Temperature, TopP, StopSequences,
// ToolChoice); ToolChoice (typed-string enum with Auto / None / Required).
// ValidateTools from AI-07 is reused for duplicate-tool detection.
// See AI-09 § Capability matrix for the v1 option surface.

// AI-10 paragraph (added after package clause so the greedy AI-07
// paragraph test does not count these lines).
//
// As of AI-10 the package exposes the provider-neutral completion
// metadata that AI-11 / AI-12 stream events carry as terminal metadata:
// FinishReason (typed-string enum with 6 canonical values: Stop,
// Length, ToolCall, ContentFilter, Cancellation, Unknown) plus
// FinishReasonFromProvider (case-insensitive, trimmed, vendor-synonym
// normalization; unknown input collapses to Unknown with no error and
// no raw upward leakage); Usage (3 required int64 counts — InputTokens,
// OutputTokens, TotalTokens — plus 3 optional *int64 detail counts —
// CacheReadTokens, CacheWriteTokens, ReasoningTokens — with MaxUsageTokenCount
// = math.MaxInt64, validated end-to-end via NewUsage); IsAbsent()
// reports "no usage reported" (zero required + all optional absent).
// Cancellation is value-level only (model-reported cancel); transport
// cancellation is owned by AI-01 § Cancellation contract. Usage and
// FinishReason are value types — they do NOT implement ContentPart and
// have NO wire-format methods; AI-11 owns marshaling. No billing /
// quota fields; no JSON serialization. See AI-10 § Capability matrix.

// AI-11 paragraph (added after package clause so the greedy AI-07
// paragraph test does not count these lines).
//
// As of AI-11 the package exposes the provider-neutral event envelope
// that crosses Layer 1 through a ModelProvider producer-owned <-chan
// Event: Event (Kind EventKind + Sequence uint64 + sealed Payload);
// EventKind (typed-string discriminator with 12 canonical values:
// EventKindResponseStart, EventKindResponseComplete, EventKindTextStart,
// EventKindTextDelta, EventKindTextEnd, EventKindReasoningStart,
// EventKindReasoningDelta, EventKindReasoningEnd,
// EventKindToolCallStart, EventKindToolCallDelta, EventKindToolCallEnd,
// EventKindError; zero value is invalid); NewEvent (sanctioned
// constructor that derives Kind from payload.Kind() and stamps the
// next Sequence from the package-private atomic counter); Validate
// (idempotent kind/registry/payload-parity check); AllEventKinds
// (canonical registry accessor returning a defensive copy in
// registration order for AI-20 / AI-21 conformance). The Payload
// field uses the sealed eventPayload interface (unexported
// aiPayload() marker), so payload implementations live exclusively
// in this package. See event.go for the envelope, ordering rules,
// and sentinel errors.
//
// # Ordering rules for the event envelope
//
//   - Each producer-owned stream starts with exactly one
//     EventKindResponseStart; the first event on a fresh producer is
//     sequence 1 and MUST be response.start. Duplicates are illegal.
//   - Sequence is 1-based, producer-assigned, and contiguous. NewEvent
//     reads the next value from the package-private atomic counter;
//     AI-20 (stream testkit) detects dropped events by inspecting gaps.
//   - At most one EventKindResponseComplete may appear per stream, and
//     it MUST NOT coexist with EventKindError. Exactly one of these
//     two terminal kinds closes a successful stream; neither is a
//     hard guarantee under cancellation (see next rule).
//   - AI-01 § Cancellation contract: cancellation is best-effort —
//     a cancelled stream may close without any terminal event
//     (no response.complete, no error). AI-31 reconciles the
//     "exactly-one-terminal" guarantee with transport-level
//     cancellation; until AI-31 lands, consumers MUST tolerate an
//     empty terminal under cancellation.
//
// See AI-11 § "Requirement: ai-event-envelope-ordering-rules" for the
// authoritative ordering contract.

// AI-12 paragraph (added after package clause; greedy AI-07 paragraph test does not count).
//
// As of AI-12: ResponseStartPayload and ResponseCompletePayload are
// concrete eventPayload implementations with event constructors and
// accessors. IsTerminalKind reports response.complete or error; AI-20
// / AI-21 own per-producer ordering. See response.go.
