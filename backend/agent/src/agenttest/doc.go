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
//     inside ai's own tests (R-AMP-013). AI-22's stream test kit, AI-23's
//     conformance suite and every Layer 2 agent-loop test are built on it.
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
package agenttest
