// This program MUST NOT compile. That is the assertion, and the compiler makes
// it — see TestPart_HandRolledFromAnotherPackage_DoesNotCompile in ../../.
//
// It is written from a package that is not ai, exactly as a determined consumer
// would write it, and it attempts every bypass decision.md § 3.3 assigns to the
// compiler rather than to validation. Each attempt is a separate declaration so
// that one refusal cannot mask another, and each carries the bypass number it
// answers.
//
// The two bypasses that are NOT here are the two that compile: the zero Part
// (bypass 1) and a Part promoted out of an embedding type (bypass 5). Those are
// legal Go and are rejected at the message boundary instead, by
// TestMessage_UnconstructedContentElement_IsRejected. Putting them here would
// make the file's claim false.
//
// Bypass 8 — copying a valid part and mutating it — is also absent, and for a
// third reason: there is nothing to mutate. A Part is a value, a textPayload is a
// value struct holding a string, and a string is immutable.
//
// This directory is testdata, which the go tool and golangci-lint both exclude
// from package patterns, so nothing here is part of `go build ./...`,
// `make test` or `make lint`.
package main

import "github.com/cachicamas/backend/agent/src/ai"

// myPayload is the best guess a consumer can make at the payload contract: it
// implements both of partPayload's methods with the right names and the right
// signatures. It still does not implement it, because the methods are unexported
// and an unexported method belongs to the package that declared it — see
// nameTheContract below for the diagnostic.
type myPayload struct{ text string }

func (myPayload) kind() ai.PartKind                { return ai.PartKindText }
func (myPayload) validate(_ ai.Path) *ai.Violation { return nil }

// embedsPart is the AI-05 bypass: the shape that satisfied the open Content
// interface by promoting its method set. Against a concrete []Part it is simply
// a different type.
type embedsPart struct{ ai.Part }

// Bypass 2 — naming the field in a composite literal.
func namedLiteral() ai.Part { return ai.Part{payload: myPayload{text: "smuggled"}} }

// Bypass 3 — the positional form of the same literal.
func positionalLiteral() ai.Part { return ai.Part{myPayload{text: "smuggled"}} }

// Bypass 4 — offering the embedding type where message content is taken.
func offerEmbedded() (ai.Message, error) { return ai.NewMessage(ai.RoleUser, embedsPart{}) }

// Bypass 6 — naming the contract, and implementing it.
func nameTheContract() { var _ ai.partPayload = myPayload{} }

// Bypass 7 — assigning the field on a value that was legally obtained.
func writeTheField() {
	var p ai.Part
	p.payload = myPayload{text: "smuggled"}
}

func main() {
	_ = namedLiteral()
	_ = positionalLiteral()
	_, _ = offerEmbedded()
	nameTheContract()
	writeTheField()
}
