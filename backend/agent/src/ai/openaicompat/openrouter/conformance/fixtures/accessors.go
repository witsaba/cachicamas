// AI-38 — fixture accessor functions.
//
// This file exposes every fixture var-block declared in the
// sibling fixture files as a no-argument function returning the
// recorded bytes / map / count. Each accessor is the single
// consumer-side door onto the recorded canonical shape — the
// conformance bridge's tests iterate through these functions, never
// through the underlying var-blocks directly, so a future change to
// the fixture storage shape (e.g., generating bytes at runtime
// instead of storing a string literal) is local to the fixtures
// package.
//
// The accessor names use the "Openrouter" prefix (matching the
// sub-package's package name) rather than the bare fixture names
// — the sub-package import boundary gives no overlap risk today,
// but a future merger with a sibling fixtures package would
// otherwise collide. The exported shape is deliberately
// function-shaped (returning the value) rather than a plain var
// export so the assertions in fixtures_test.go can iterate
// without leaking the fixture bytes through a panic or a copy
// hazard.

package fixtures

// OpenrouterTextStream returns the recorded OpenRouter-shaped text
// stream (ResponseStart + 3 TextDelta + Completion + [DONE]) — see
// text_stream.go.
func OpenrouterTextStream() string {
	return openrouterTextStream
}

// OpenrouterToolCallStream returns the recorded OpenRouter-shaped
// tool-call stream (ToolCallStart + 2 ToolCallDelta +
// bridge-synthesized Completion + [DONE]) — see tool_call.go.
func OpenrouterToolCallStream() string {
	return openrouterToolCallStream
}

// OpenrouterTextStreamWithUsage returns the recorded
// OpenRouter-shaped text stream with a C8 usage object on the
// terminal chunk — see with_usage.go.
func OpenrouterTextStreamWithUsage() string {
	return openrouterTextStreamWithUsage
}

// OpenrouterAttributionObjectDiscriminatorValue returns the value
// ("chat.completion.chunk") the conformance bridge asserts every
// rendered chunk's object discriminator must equal — see
// bridgeAttributionRoundTripper's own constants. The accessor is
// not strictly necessary (the value is a constant) but it gives the
// fixtures_test.go assertions a single source of truth they
// iterate through, matching every other fixture's accessor shape.
func OpenrouterAttributionObjectDiscriminatorValue() string {
	return "chat.completion.chunk"
}

// OpenrouterAttributionHeaderNames returns the fixed list of HTTP
// header names the conformance bridge injects on every outbound
// request (R-OR-02 sub-scenario 1) — see
// with_attribution_headers.go.
func OpenrouterAttributionHeaderNames() []string {
	return openrouterAttributionHeaderNames
}

// OpenrouterAttributionPresentValues returns the expected outbound
// header map a request built with all non-empty attribution strings
// produces (R-OR-02 sub-scenario 1) — see with_attribution_headers.go.
func OpenrouterAttributionPresentValues() map[string]string {
	return openrouterAttributionPresentValues
}

// OpenrouterAttributionAbsentCount returns the expected count of
// attribution-named headers in the outbound header when every
// attribution field is the zero value (R-OR-02 sub-scenario 2) —
// see with_attribution_headers.go.
func OpenrouterAttributionAbsentCount() int {
	return openrouterAttributionAbsentCount
}

// OpenrouterAttributionPresentCount returns the expected count of
// attribution-named headers in the outbound header when every
// attribution field is non-empty (R-OR-02 sub-scenario 1) — see
// with_attribution_headers.go.
func OpenrouterAttributionPresentCount() int {
	return openrouterAttributionPresentCount
}