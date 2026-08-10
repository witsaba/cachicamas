// Package agenttest is Layer 1's external-consumer proof, and — as of
// AI-21 — an importable testing library built on top of it. It carries two
// roles, deliberately in the one package rather than two:
//
//  1. The proof role (AI-00 … AI-20): every Layer 1 contract is exercised
//     from outside its own package, on the day it lands, rather than months
//     later by Layer 2. A type that validates cleanly from within package ai
//     but cannot be constructed, read or asserted from another package is a
//     defect this package is positioned to catch immediately. The retired
//     Layer 1 shipped exactly that defect: content parts that no other
//     package could read, which made request translation structurally
//     impossible.
//  2. The library role (AI-21 onward): [Provider] is an exported, scriptable
//     [ai.ModelProvider] — the one producer every consumer above Layer 1
//     needs and, before this milestone, the only one existed unexported
//     inside ai's own tests (R-AMP-013). AI-22 builds the stream test kit
//     on top of it, in this package's own stream_kit_*.go files:
//     [DrainAndRecord] and [Recording] (timeout-safe drain, one recording
//     backing many assertions), [RequireSameEvents] (readable, bounded
//     diffs), [RequireValidStream] and [CheckContiguity] (ordering,
//     delegated to [ai.CheckStream], plus new sequence contiguity),
//     [RequireNoGoroutineLeak] (opt-in, serial-only leak detection), and
//     [Iter] (a carrier view over a stream the caller already holds).
//     AI-23 builds the provider conformance suite on top of both, in this
//     package's own conformance_*.go files: [RunConformance] (the one
//     exported entry point — a caller-supplied [Factory] in, AI-03 §10's
//     [CapabilityRecord] out, with zero copied assertions), [Capability]
//     and [Capabilities] (AI-03 §5/§6's closed nine-member list), and
//     [CapabilityRecord.Verdict] (the pass/fail/inconclusive summary AI-03
//     §10's rule computes mechanically). Every case delegates its
//     mechanics to the fake and the kit — none reimplements a drain, a
//     diff, an ordering check, a contiguity check or a leak detector
//     (NFR-CNF-D) — and every Layer 2 agent-loop test is built on all
//     three: the fake, the kit and the suite. AI-35 extends the suite
//     with the CapRetry capability (CAP-O-04) and an httptest-driven
//     conformance case body that targets the *openaicompat.Client
//     directly; the case body's openaicompat import is recorded here
//     because the case body is adapter-specific (the suite's other
//     cases stay adapter-agnostic against the scripted fake).
//
// # Dependency-free (R-STK-009)
//
// This package imports only the standard library and this module's own
// src/ai — [Provider] and the AI-22 stream test kit alike. AI-22.4's
// goroutine leak helper ([RequireNoGoroutineLeak]) deliberately rejected a
// third-party detector (go.uber.org/goleak) for exactly this reason:
// adopting one would need its own ADR (openspec/AGENTS.md rule 5) and would
// break this pin before AI-24 first selects a transport dependency.
// backend/agent/go.mod carries zero requires; both AI-00 import guards
// enforce that as a build failure, not a review convention.
//
// # Contract-faithful, not convenient
//
// The fake in this package MUST reproduce AI-20's stream physics exactly —
// one producer goroutine per stream, one closing site, every send
// (including the terminal one) selecting on cancellation, and the one
// sanctioned loss path (a saturated buffer racing cancellation drops the
// remaining events and closes bare, with no terminal event forced through).
// It MUST NOT be simplified for a later contributor's convenience: a fake
// that closes cleanly where the real contract drops events, or that forces
// a terminal event through where AI-20 forbids one, teaches Layer 2 the
// wrong physics permanently, and does so silently. Where this package's own
// tests found a scripting need without precedent in scriptProvider — the
// Gate synchronization primitive, the typed ErrScriptsExhausted queue
// failure — the choice is recorded in cachicamas-ai-fake-provider/design.md,
// not improvised inline.
//
// # This directory must remain a direct sibling of src/ai
//
// This is a structural constraint, not a stylistic preference. AI-20 adds a
// signature guard over the provider interface that resolves ../ai from its
// own source file via runtime.Caller. Any other layout breaks that guard
// silently — it will not fail loudly at the moment of the move, and nothing
// else in the tree records the dependency. See ADR 0005 § D2 and Guard C.
//
// So: src/ai/ and src/agenttest/ share the parent src/. Moving, nesting or
// renaming either one is a breaking change that must update the guard in the
// same commit.
//
// # The v1 surface freeze
//
// AI-40.1's consumer proof (src/handoff) and AI-40.2's four runnable
// examples (src/ai/example_test.go) are the compiled, run evidence for the
// v1 surface this package and src/ai expose. See
// cachicamas-ai-layer2-handoff's decision.md (doc 0002 § AI-40) for the
// full by-capability enumeration, frozen as of that change.
package agenttest
