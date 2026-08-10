// AI-00.2 — the external-consumer compile proof.
//
// This file is deliberately a compile proof and nothing more. When it was
// written package ai exported no declaration, so there was no behavior to
// assert; the package has long since grown its full surface, but this file's
// job is unchanged: prove the Layer 1 package is importable, by its full
// module path, from a package that is not itself part of it.
//
// That is not a tautology. It fails if the module path in go.mod disagrees with
// the directory layout, if src/agenttest stops being a direct sibling of src/ai,
// or if package ai acquires a compile error. Each of those breaks Layer 2 in a
// way whose first symptom would otherwise appear far from its cause.
//
// The external readability of a real Layer 1 type — a type that can be
// constructed here, read back, and type-asserted — is proven by AI-06.2, on the
// first content part that exists. This file is its placeholder, not its
// substitute.
package agenttest_test

import (
	"testing"

	// Imported for its side effect on the build graph: this import is the
	// proof. It is blank because package ai exports nothing to reference yet.
	_ "github.com/cachicamas/backend/agent/src/ai"
)

func TestLayer1Package_IsImportableFromAnExternalPackage_Compiles(t *testing.T) {
	t.Parallel()

	// Reaching this line means the blank import above resolved and the test
	// binary linked. There is no runtime assertion to make: the compiler and
	// the linker already made it.
}
