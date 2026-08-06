// AI-26.7 — the escape hatch: this adapter's own reserved namespace
// merges into the wire body; every foreign namespace is ignored whole
// (R-ART-019, AI-12.3's property, now proven at the wire).
//
// The reserved namespace value (openaicompat.Namespace) was NOT read
// from AI-25's landed artifact, despite tasks.md's original instruction
// to do so: an exhaustive search (recorded in doc.go and tasks.md's
// evidence log) found AI-25 was never given a task to define one. The
// coordinator ruled this milestone defines it instead — see doc.go's
// "Escape hatch: reserved namespace" section for the full rationale.
package openaicompat_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

func init() {
	registerExpectations(
		expectationCase{
			name: "own-namespace extension merges its members into the wire body",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}, ai.WithProviderExtension(openaicompat.Namespace, []byte(`"top_k":40,"repetition_penalty":1.1`))))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true},"top_k":40,"repetition_penalty":1.1}`,
		},
	)
}

// TestExtension_OwnNamespaceMergesDiffersFromExtensionFreeTwin proves
// S-ART-065: a request carrying an extension under this adapter's own
// reserved namespace translates to a body that (a) differs from its
// extension-free twin and (b) matches the registered expectation above
// with the extension's bytes merged in, verbatim, at the end.
func TestExtension_OwnNamespaceMergesDiffersFromExtensionFreeTwin(t *testing.T) {
	message := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!"))))

	withExtension := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message},
		ai.WithProviderExtension(openaicompat.Namespace, []byte(`"top_k":40,"repetition_penalty":1.1`))))
	withoutExtension := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message}))

	withBytes, err := openaicompat.Translate(withExtension)
	if err != nil {
		t.Fatalf("Translate(with extension): unexpected error: %v", err)
	}
	withoutBytes, err := openaicompat.Translate(withoutExtension)
	if err != nil {
		t.Fatalf("Translate(without extension): unexpected error: %v", err)
	}

	if string(withBytes) == string(withoutBytes) {
		t.Fatalf("with-extension and without-extension bodies were identical = %s, want the extension's bytes merged in", withBytes)
	}

	wantWith := `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true},"top_k":40,"repetition_penalty":1.1}`
	if string(withBytes) != wantWith {
		t.Fatalf("Translate(with extension) =\n%s\nwant\n%s", withBytes, wantWith)
	}
	wantWithout := `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`
	if string(withoutBytes) != wantWithout {
		t.Fatalf("Translate(without extension) =\n%s\nwant\n%s", withoutBytes, wantWithout)
	}
}

// foreignExtensionExpectation is the hard-coded anchor both
// foreign-namespace tests below compare against directly, not merely
// against each other. Anchoring at least one side to an
// independently-known-correct literal is what keeps this proof from
// being vacuous in either of this wave's catalogued shapes: (1) a
// fixture that cannot distinguish "implemented" from "not implemented"
// — before appendExtensionFields existed, a twin-only comparison would
// ALSO have passed, since nothing rendered any extension either way; (2)
// a test comparing an implementation only against itself — two runtime
// outputs that happen to match each other could still both be wrong in
// the same way (e.g. an unrelated bug that corrupted every body
// identically). This literal is the plain, unmarked "one user text
// message" body, pinned independently of the extension-handling code
// path under test.
const foreignExtensionExpectation = `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`

// TestExtension_ForeignNamespaceIgnoredWhole proves S-ART-066: a request
// carrying a foreign-namespace extension (not this adapter's own
// reserved Namespace) translates byte-identically to its extension-free
// twin, and both match the hard-coded anchor above directly.
func TestExtension_ForeignNamespaceIgnoredWhole(t *testing.T) {
	message := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!"))))

	withForeign := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message},
		ai.WithProviderExtension("anthropic", []byte(`"anthropic_only_field":"should never appear"`))))
	withoutAny := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message}))

	foreignBytes, err := openaicompat.Translate(withForeign)
	if err != nil {
		t.Fatalf("Translate(foreign namespace): unexpected error: %v", err)
	}
	plainBytes, err := openaicompat.Translate(withoutAny)
	if err != nil {
		t.Fatalf("Translate(no extension): unexpected error: %v", err)
	}

	if string(foreignBytes) != string(plainBytes) {
		t.Fatalf("Translate(foreign namespace) = %s, want byte-identical to the extension-free twin %s", foreignBytes, plainBytes)
	}
	if string(foreignBytes) != foreignExtensionExpectation {
		t.Fatalf("Translate(foreign namespace) =\n%s\nwant the hard-coded anchor\n%s", foreignBytes, foreignExtensionExpectation)
	}
	if string(plainBytes) != foreignExtensionExpectation {
		t.Fatalf("Translate(no extension) =\n%s\nwant the hard-coded anchor\n%s", plainBytes, foreignExtensionExpectation)
	}
}

// TestExtension_SeveralForeignNamespacesAtOnceStillIgnoredWhole proves
// S-ART-067: the rule is per-namespace and total, not first-match —
// three simultaneous foreign namespaces still leave the body
// byte-identical to the same hard-coded anchor.
func TestExtension_SeveralForeignNamespacesAtOnceStillIgnoredWhole(t *testing.T) {
	message := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!"))))

	withThreeForeign := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message},
		ai.WithProviderExtension("anthropic", []byte(`"anthropic_field":"a"`)),
		ai.WithProviderExtension("gemini", []byte(`"gemini_field":"b"`)),
		ai.WithProviderExtension("cohere", []byte(`"cohere_field":"c"`)),
	))

	got, err := openaicompat.Translate(withThreeForeign)
	if err != nil {
		t.Fatalf("Translate: unexpected error: %v", err)
	}
	if string(got) != foreignExtensionExpectation {
		t.Fatalf("Translate(3 foreign namespaces) =\n%s\nwant the hard-coded anchor (unchanged)\n%s", got, foreignExtensionExpectation)
	}
}
