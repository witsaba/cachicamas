// Package main is a compile-negative fixture (design.md D4 piece 3): it
// lives under testdata/ so `./...` never builds or tests it, and it
// imports the live-smoke package from OUTSIDE the
// .../openaicompat/openrouter/ subtree the smoke package's own directory
// placement restricts importers to. The reachability guard
// (internal/smoke/reachability_guard_test.go) runs `go build` on this
// file and asserts the build fails, naming Go's own import-visibility
// rule as the reason — proof that the compiler, not only the guard,
// refuses this import (R-LSM-005, S-LSM-017).
//
// No credential-shaped literal appears anywhere in this file (AI-36
// recursive credential scan surface, R-ART-022): it only imports a
// package and never constructs a Credential, a request, or any string
// resembling a secret.
package main

import (
	_ "github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke"
)

func main() {}
