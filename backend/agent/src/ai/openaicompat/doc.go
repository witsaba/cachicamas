// Package openaicompat is the OpenAI-compatible dialect adapter Layer 1
// begins at AI-25. At this milestone it is a configured value only:
// construction turns an injected endpoint, an injected credential and an
// optional injected HTTP client into a value later milestones build
// against. Streaming behaviour arrives at AI-26; this package does not ship
// a placeholder for it (see provider_boundary_test.go).
//
// It lives in its own subpackage rather than inside package ai because
// AI-25.2's ambient-authority guard must scan "the adapter's own source
// files", a scope that is unenforceable inside ai's much larger, shared
// package.
//
// # No whole-request cap, by any mechanism
//
// The client this package builds for itself never sets http.Client.Timeout
// and never derives an equivalent deadline internally on any code path.
// The standard library's own documentation says that field "includes ...
// reading the response body" and "will interrupt reading of the
// Response.Body" — so a whole-request cap kills every stream longer than
// itself, and the failure surfaces as an unexplained mid-read truncation
// far from here, which owns the actual cause. A caller's own context
// remains the only way to bound a call. Connect-phase bounds and a
// pooled-connection idle bound exist instead (see the default* constants in
// client.go): each bounds a phase before any response body arrives, never
// the body read itself.
//
// # Injection only, no ambient authority
//
// This package reads no environment variable, touches no filesystem path
// and spawns no process. Its credential and its endpoint arrive only by
// injection through Config. See credential.go for why the credential value
// itself cannot be formatted or serialized into the open, and
// ambient_authority_test.go (AI-25.2) for the mechanical, call-site-scanned
// enforcement of this rule.
//
// # No environment-derived proxy on the client this package builds
//
// When no HTTP client is injected, this package builds its own, and that
// client's transport deliberately leaves its proxy resolver unset. It never
// adopts http.DefaultTransport and never falls back to
// http.ProxyFromEnvironment, both of which resolve a proxy by reading
// HTTP_PROXY, HTTPS_PROXY and NO_PROXY from the environment. Routing
// through either would be an environment read this package's own guard
// cannot see, because a call-site scan's selector resolves to the http
// package, not to the environment package — the vector R-APC-009 exists to
// close directly.
//
// # The no-ambient-authority guarantee is scoped, not absolute
//
// The guarantee above covers this package's own sources and the client it
// builds for itself when the caller injects none. It does not, and cannot,
// cover an injected client's transport: that value was configured at the
// composition root, on the caller's own instruction, and any environment
// variable it consults was consulted there — not here. Per ADR 0005,
// nothing below the composition root reads the environment, and an
// injected client is not below it. Rejecting or silently re-wrapping an
// injected client to "fix" this would also break a legitimate
// corporate-proxy deployment the caller configured on purpose.
//
// # An absent HTTP client is not a fault
//
// Config.HTTPClient is optional. Its absence selects this package's own
// bounded client; it is never treated as an omission, and construction
// never fails because of it (see New in client.go).
package openaicompat
