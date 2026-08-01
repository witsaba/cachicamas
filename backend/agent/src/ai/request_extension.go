// AI-12.3 — the provider escape hatch: typed, namespaced, opaque.

package ai

import (
	"bytes"
	"slices"
	"strings"
)

// ProviderExtension is one namespaced, opaque, provider-specific value
// (V-REQ-28).
//
// Sealed by unexported fields and by the absence of an exported constructor:
// a consumer reads one, never builds one. It is produced only by a request,
// through [WithProviderExtension].
type ProviderExtension struct {
	namespace string
	value     []byte
}

// Namespace returns the provider namespace the value is carried under.
//
// It is caller data — the name of a provider — and never rendered by
// [ProviderExtension.String] or [ProviderExtension.GoString] (R-REX-010).
func (e ProviderExtension) Namespace() string { return e.namespace }

// Value returns the opaque payload, byte-exact and never interpreted
// (V-REQ-28, R-REX-006).
//
// The result is a fresh slice on every call, matching [Tool.Schema] and
// [ToolCall.Arguments]: a consumer that rewrites what it received must not
// be able to rewrite the request, and two consumers must not be able to
// observe each other.
func (e ProviderExtension) Value() []byte { return bytes.Clone(e.value) }

// IsZero reports whether e was never constructed through
// [WithProviderExtension] — the same "no namespace, therefore nothing"
// detector [NewTool]'s zero-value paragraph documents for [Tool].
func (e ProviderExtension) IsZero() bool { return e.namespace == "" }

// String renders the extension for a diagnostic reader, naming that it is
// one and never its namespace or its value (R-REX-010).
//
// Not the namespace: a namespace is caller-supplied data naming a provider.
// Not the value, and not its length: a length is caller-derived, and a
// prefix of a secret is still a secret.
func (e ProviderExtension) String() string { return "extension" }

// GoString renders the extension for the %#v verb.
func (e ProviderExtension) GoString() string { return e.String() }

// WithProviderExtension attaches a provider-specific value under a
// namespace (V-REQ-28).
//
// Total, never validating, last-wins per namespace: re-applying the same
// namespace replaces its value but keeps its FIRST read-back ordinal, so
// read-back order does not depend on how many times a value was revised
// (R-REX-009, design.md § 6). The value is copied on the way in, matching
// every other option that carries a slice.
//
// draft.extensions is cloned before any write, whether replacing an entry
// or appending: the seed [Request.With] copies from r.options may still
// share a backing array with a live, immutable Request, and a write through
// that shared array would silently corrupt the source (R-REX-001) exactly
// as an unguarded append could.
func WithProviderExtension(namespace string, value []byte) RequestOption {
	return func(d *requestDraft) {
		extension := ProviderExtension{namespace: namespace, value: bytes.Clone(value)}
		for i, existing := range d.extensions {
			if existing.namespace == namespace {
				d.extensions = slices.Clone(d.extensions)
				d.extensions[i] = extension
				return
			}
		}
		d.extensions = append(slices.Clone(d.extensions), extension)
	}
}

// ProviderExtension returns the value carried under namespace, and whether
// one is present (V-REQ-28).
//
// The second result is what tells "absent" from "present with a zero value
// to interpret" apart, the same distinction every other optional region in
// this package makes.
func (r Request) ProviderExtension(namespace string) (ProviderExtension, bool) {
	for _, ext := range r.options.extensions {
		if ext.namespace == namespace {
			return ext, true
		}
	}
	return ProviderExtension{}, false
}

// ProviderExtensions returns every extension the request carries, in
// first-application order (V-REQ-28, R-REX-009).
//
// The result is a fresh slice on every call, matching every other ordered
// region this package exposes.
func (r Request) ProviderExtensions() []ProviderExtension {
	return slices.Clone(r.options.extensions)
}

// providerExtensionsEqual reports whether two extensions carry the same
// namespace and byte-equal values.
//
// A free function on [toolsEqual]'s precedent (request.go), not an exported
// ProviderExtension.Equal: the region carries no presence flag of its own —
// it is a slice, so absent and empty are one fact — and [Request.Equal]'s
// own GoDoc requires every comparison to read through the exported
// accessor, which is why this reads Namespace()/Value() and not the fields.
func providerExtensionsEqual(a, b ProviderExtension) bool {
	return a.Namespace() == b.Namespace() && bytes.Equal(a.Value(), b.Value())
}

// extensionsRule is the region's own rule (design.md § 6, R-REX-008): every
// extension has a namespace, then a non-empty value, checked in ordinal
// order so the first failing extension is the one reported — the same
// per-element discipline [NewMessage]'s content rule and
// [NewSystemInstruction]'s segment rule use.
//
// A namespace is text naming a provider, so whitespace-only is folded into
// emptiness, matching the model identity and a system-instruction segment.
// A value is opaque bytes the package does not know are whitespace, so a
// whitespace-only value is legal — only a zero-length value is rejected.
// No other rule is applied to either: no format, no prefix convention, no
// length bound, no character class, and no registry of recognised
// namespaces (design.md § 6).
func (d requestDraft) extensionsRule() Rule {
	return func() *Violation {
		for i, ext := range d.extensions {
			if strings.TrimSpace(ext.namespace) == "" {
				return Invalid(ErrEmpty, AtIndex("extensions", i), At("namespace"))
			}
			if len(ext.value) == 0 {
				return Invalid(ErrEmpty, AtIndex("extensions", i), At("value"))
			}
		}
		return nil
	}
}
