// Package openrouter is the first concrete vendor wrapper on top of the
// shipped, vendor-agnostic openaicompat adapter (design D1, memory #2571):
// OpenRouter as a thin composition of openaicompat.Config plus a
// wrapper-owned http.RoundTripper that injects the three attribution
// headers OpenRouter expects on every outbound request.
//
// # Composes, does not re-implement
//
// The wrapper builds an *openaicompat.Client through the shipped
// openaicompat.New(Config{...}) constructor. Every wire-format choice —
// the streamed delta shape, the streaming media type, the
// stream_options.include_usage field, the credential's Bearer rendering —
// is openaicompat's own, untouched by this package. The wrapper's only
// observable effects are:
//
//   - Three attribution headers (HTTP-Referer, X-OpenRouter-Title, and
//     X-Title as alias, X-OpenRouter-Categories) attached by a
//     wrapper-owned http.RoundTripper wrapping openaicompat's transport
//     (R-OR-02).
//   - A deliberate-model field on Config that overrides the ai.Request's
//     model identifier on the wire, defaulting to "openai/gpt-4o"
//     (R-OR-03, R-OR-05).
//
// # Injection-only, no ambient authority (R-OR-01, AI-25.2 invariant)
//
// This package reads no environment variable, touches no filesystem path
// and spawns no process. The credential, the endpoint, the HTTP client,
// the attribution strings and the model identifier all arrive by injection
// through Config. ambient_authority_test.go (R-OR-01's mechanical guard)
// scans this package's own non-test sources for any os / os/exec /
// syscall / io/ioutil call site and fails on a single one.
//
// # No Endpoint field on Config (D1, D3)
//
// The design deliberately does NOT expose an Endpoint field: OpenRouter's
// endpoint is fixed by this wrapper's existence. A caller who needs a
// different OpenAI-compatible endpoint uses the shipped openaicompat
// package directly — there is no second wrapper to discover here.
//
// # No whole-request cap, no global default transport (AI-24, R-APC-009)
//
// When Config.HTTPClient is nil, NewProvider builds an *openaicompat.Client
// through openaicompat.New, which constructs its own bounded transport
// (openaicompat.Client's newDefaultHTTPClient, R-APC-009) — never
// http.DefaultTransport, never http.ProxyFromEnvironment, never a
// whole-request timeout. A Config.HTTPClient a caller injects is used
// verbatim by openaicompat.New (R-APC-003) and is the injector's decision
// to bound.
//
// # Zero new go.mod require (R-OR-09, AI-00.3 invariant)
//
// This package's own imports are net/http (stdlib) plus package ai and
// package openaicompat (both in this module). No new top-level dependency
// is added; go.mod remains three lines (module + go directive + blank
// line) and the AI-00.3 forward-guard passes unmodified.
package openrouter