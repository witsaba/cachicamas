// Package agenttest is the external-consumer proof for Layer 1.
//
// It exists so that every Layer 1 contract is exercised from outside its own
// package, on the day it lands, rather than months later by Layer 2. A type that
// validates cleanly from within package ai but cannot be constructed, read or
// asserted from another package is a defect this package is positioned to catch
// immediately. The retired Layer 1 shipped exactly that defect: content parts
// that no other package could read, which made request translation structurally
// impossible.
//
// # This directory must remain a direct sibling of src/ai
//
// This is a structural constraint, not a stylistic preference. A later milestone
// adds a signature guard over the provider interface that resolves ../ai from
// its own source file via runtime.Caller. Any other layout breaks that guard
// silently — it will not fail loudly at the moment of the move, and nothing else
// in the tree records the dependency. See ADR 0005 § D2 and Guard C.
//
// So: src/ai/ and src/agenttest/ share the parent src/. Moving, nesting or
// renaming either one is a breaking change that must update the guard in the
// same commit.
package agenttest
