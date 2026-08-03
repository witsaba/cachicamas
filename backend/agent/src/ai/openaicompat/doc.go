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
//
// # Wire-shape provenance (AI-26.1, R-ART-001)
//
// Translate begins at this milestone (translation.go, body.go). Four
// wire-shape claims its later slices depend on were uncited or only
// partially cited at proposal time. R-ART-001's earlier-of-charter-and-
// dependency rule pulls all four into this slice regardless of which
// node each nominally gates, so all four are settled here, against the
// vendor's own published documentation, before any node whose behaviour
// depends on one specifies that behaviour as fact. Retrieved 2026-08-03.
//
//  1. System instruction placement and role. OpenAI's own OpenAPI
//     specification (github.com/openai/openai-openapi, an
//     organization-owned repository described as "OpenAPI specification
//     for the OpenAI API"; main branch, commit d4fb706e6e05d4cc9f1b33ca5
//     9b6e4f3e8edd439, openapi.yaml) defines schemas
//     ChatCompletionRequestSystemMessage and
//     ChatCompletionRequestDeveloperMessage, both members of the
//     ChatCompletionRequestMessage discriminated union that IS the
//     `messages` array's element type: a system instruction is therefore
//     a `messages` entry, never a top-level field. Two roles are
//     documented, both legal: `system` ("Developer-provided instructions
//     that the model should follow, regardless of messages sent by the
//     user") and `developer`, which the same specification states
//     explicitly replaces `system` "with o1 models and newer". This
//     package holds no model catalog (mirroring package ai's own V-OUT-14
//     posture), so which of the two a later slice renders is that
//     slice's decision, not decided here; both are cited.
//
//  2. Tool-call-argument encoding, request side specifically. The same
//     specification's schema ChatCompletionRequestAssistantMessage —
//     the request-side shape a replayed assistant turn's history uses —
//     carries `tool_calls`, typed as `$ref:
//     ChatCompletionMessageToolCalls`, whose element
//     ChatCompletionMessageToolCall.function.arguments is `type: string`
//     ("The arguments to call the function with, as generated by the
//     model in JSON format"). This is the identical schema object AI-24
//     §7 already read on the response side (the model's own reply);
//     the specification reuses it verbatim for request-side replay, which
//     is what settles the request-side symmetry this claim needed and
//     AI-24 §7 alone did not establish.
//
//  3. tools[].function.parameters carries the neutral schema verbatim.
//     The same specification's ChatCompletionTool.function resolves to
//     FunctionObject, whose `parameters` field is FunctionParameters:
//     `type: object`, "The parameters the functions accepts, described as
//     a JSON Schema object ... Omitting `parameters` defines a function
//     with an empty parameter list", `additionalProperties: true`. The
//     field accepts an arbitrary JSON Schema object whole; there is no
//     vendor-owned narrowing to marshal a caller's schema through, which
//     is exactly why body.go never re-marshals it.
//
//  4. No strict role alternation is enforced. Two sources, neither alone
//     dispositive, together close what AI-24 §6's lone `role: "tool"` row
//     left open (that row shows a third role exists, not that the SAME
//     role may repeat consecutively):
//     - the same specification's `messages` array and its
//     ChatCompletionRequestMessage discriminator impose no ordering
//     or alternation constraint on role sequence at all — checked
//     exhaustively: no normative use of "alternat*" appears anywhere
//     in the ~84,000-line specification;
//     - OpenAI's own Function calling guide
//     (developers.openai.com/api/docs/guides/function-calling,
//     archived snapshot dated 2026-06-01 via web.archive.org)
//     documents "Parallel function calling" as a supported feature,
//     and its own canonical worked example appends one
//     `{"role": "tool", "tool_call_id": ..., "content": ...}` message
//     per call, inside a loop over the model's returned
//     `message.tool_calls`, onto the very same `messages` list,
//     before the next request. When a turn produces more than one
//     tool call — the guide's own documented, supported case — this
//     is the vendor's own intended mechanism for the wire request to
//     carry two or more CONSECUTIVE `tool`-role messages. That is
//     materially stronger evidence than a single distinct role
//     existing: it is a working, vendor-endorsed multi-message
//     same-role sequence.
//
// None of the four is contradicted by anything read during this
// research. All four are treated as cited for every node whose behaviour
// depends on them, per R-ART-001.
//
// # System role: system, not developer (AI-26.2, R-ART-005)
//
// Claim 1 above cites both `system` and `developer` as legal roles for a
// system-instruction message and defers which one an adapter renders to
// "a later slice" — this one. The ruling: this adapter always renders
// the wire role `system`. `ChatCompletionRequestDeveloperMessage` exists
// in the cited OpenAPI specification (claim 1, same commit) and is
// deliberately never used by this package.
//
// The reason is the vendor selection itself, not a coin flip between two
// cited options. AI-24 chose the OpenAI-compatible dialect explicitly
// for its widest reach across every vendor sharing that dialect —
// OpenRouter, vLLM, llama.cpp, Ollama, LocalAI and similar — not for
// first-party OpenAI fidelity. `developer` is a recent OpenAI-proprietary
// addition that replaces `system` "with o1 models and newer" (claim 1's
// own quote); it is precisely the part of the schema that is NOT shared
// across that wider dialect. Emitting it would narrow this adapter to
// recent first-party OpenAI models and defeat the reach rationale that
// selected the dialect in the first place. Between the two cited,
// legal, accepted roles, the one shared by the whole target population
// wins over the one proprietary to a single vendor's newest models.
//
// This package holds no model catalog (NFR-ART-B; the same V-OUT-14
// posture noted above), so per-model detection was never available as a
// third option: the choice had to be one constant for every model this
// adapter is pointed at. `system` is that constant.
//
// # Living-graph reopen trigger — not a hedge
//
// This ruling is scoped to what AI-24 chose this dialect for: breadth
// across shared-dialect vendors. If this adapter is ever pointed at a
// first-party OpenAI endpoint whose target models require `developer`
// (o1-and-newer, per claim 1), that is a real, concrete future condition
// this package cannot detect on its own — and this decision reopens when
// it occurs. It is written down as a reopen trigger, not folded silently
// into "always system", so a later reader who hits that condition knows
// this is the place the decision was made and why, rather than
// rediscovering the same research from nothing.
//
// # Refusal taxonomy: AI-25 vs AI-26 (NFR-ART-E)
//
// AI-25's construction faults and AI-26's capability refusals are
// different failure classes and MUST NOT be read as drift, even though
// both eventually route through this same package:
//
//	|                        | AI-25 construction fault             | AI-26 refusal                       |
//	|------------------------|---------------------------------------|--------------------------------------|
//	| Example                | malformed endpoint, empty credential | a request carries a reasoning part   |
//	| Is the request valid?  | no request exists yet                | yes — neutrally valid                |
//	| What failed            | the caller's contract                | the provider's expressiveness        |
//	| Class                  | validation fault                     | capability failure                   |
//	| Taxonomy               | ai.Violation (AI-04)                 | ai.PreStreamFailure + ai.ErrUnsupportedCapability (AI-19) |
//
// AI-03 §10.4 settles the classification: unsupported capability is a
// request-time failure and is not an absent optional capability. AI-26
// adds no new sentinel: every refusal this package will construct
// (AI-26.6's reasoning refusal and AI-26.8.2's exhaustive-walk refusals,
// both later slices) is
// ai.PreStreamFailure(ai.FailureReport{Category:
// ai.FailureCategoryUnsupportedCapability, Cause: <error naming the
// unsupported feature>}), reachable via errors.Is(err,
// ai.ErrUnsupportedCapability) through ai.Failure.Is — the same uniform
// door for both later slices, never a per-site construction.
//
// # Cache-boundary markers vanish whole (AI-26.2, R-ART-006)
//
// This adapter targets a vendor that caches automatically and exposes no
// client-supplied cache-boundary annotation, so every marker a request
// carries — on a system-instruction segment, a tool declaration or a
// message — is dropped whole: it renders as no wire field, no hint, no
// comment and no ordering change. This exercises, and does not modify,
// AI-11.3's own advisory contract (openspec/specs/ai-cache-breakpoints/
// spec.md, R-ACB-008): "markers are advisory: a translator may ignore
// every one of them", conditioned on reading a region's full substantive
// content and simply never consulting the marker on it — R-ACB-008's own
// text names the control (S-ACB-034) that keeps the identity from being
// vacuous, and S-ACB-035 requires the region read to be complete, not
// merely present, for the proof to count.
//
// That conditioning is why this slice's twin-comparison proof
// (cache_marker_test.go) walks every region ai.CacheRegions() enumerates
// but runs only two of three to completion, non-vacuously:
// appendMessageObject (body.go, AI-26.1) and appendSystemMessageObject
// (system.go, AI-26.2) are the only per-carrier renderers that already
// emit a carrier's full substantive content — role and text, and segment
// text in the system role, respectively — onto the wire. The remaining
// cache region is recorded SKIP, with a named reason, rather than a
// fabricated PASS over content that is not rendered at all:
//
//   - Tools: tool.go does not exist before AI-26.4 (slice 3). A
//     marker-drop proof over declarations Translate does not render at
//     all would be vacuous under S-ACB-035, not a proof that the marker
//     specifically is what is dropped.
//
// That gap is structural and self-closing: cache_marker_test.go's walk
// is driven by ai.CacheRegions(), so once tool.go exists, the same
// unedited test starts proving R-ART-006 for that region too, with no
// edit needed here.
//
// The system sub-test's own twin comparison was green on first write,
// for the same structural reason the messages sub-test was
// (appendSystemMessageObject never reads Segment.IsCacheBoundary()):
// falsifiability was proven by a staged, reverted mutation — temporarily
// rendering a marked segment's marker as a scratch wire field, observing
// the sub-test fail, then reverting — mirroring the messages region's
// own staged-mutation proof exactly (S-ART-023).
//
// # Output-token limit: optional for this vendor, mandatory-default branch dead (AI-26.7, R-ART-018)
//
// doc 0002's own amended AI-26.7 charter (line 1569, dated 2026-08-03,
// AI-24) already records this vendor's output-token limit as optional,
// not mandatory: "AI-24.1 recorded that this vendor's output-token limit
// is optional, not mandatory. Item 2's IF-guard therefore never fires for
// this adapter, recorded per this node's own instruction rather than left
// unmentioned. The text stays in force for a future adapter whose vendor
// does mandate a limit." R-ART-018 restates the same fact as this node's
// own obligation: omitting the neutral max-output-tokens option MUST
// leave the wire body's max_tokens field explicitly absent, never a
// supplied default value.
//
// appendGenerationOptionFields (option.go) renders max_tokens only when
// Request.MaxOutputTokens' own presence flag is true; when it is false,
// no code path constructs a default, a fallback or a placeholder — the
// field is simply never appended. This is the honest pin for a dead
// branch: absence is asserted directly
// (TestOption_MaxOutputTokensOmitted_FieldExplicitlyAbsent,
// option_test.go), paired with the registered "max output tokens option
// present with caller value" case proving the field renders when the
// caller does set it — the field is asserted present when set and
// asserted absent when it is not, rather than merely missing from an
// expectation literal that could simply have forgotten to mention it.
//
// # Escape hatch: reserved namespace not yet defined upstream (AI-26.7, R-ART-019)
//
// R-ART-019 requires this adapter's own reserved provider-extension
// namespace value to be read from AI-25's landed artifact, never
// re-invented here. This slice searched AI-25's landed artifact
// exhaustively before writing any escape-hatch code: every AI-25 non-test
// source this package owns (client.go, credential.go, endpoint.go,
// request.go), this file's own AI-25-era sections above, AI-25's own SDD
// artifact (openspec/changes/cachicamas-ai-provider-client/{proposal,
// spec,design,tasks}.md), AI-24's merged decision
// (cachicamas-ai-first-provider-decision/decision.md), and doc 0002's own
// AI-25 charter (§§ AI-25 … AI-25.3). None of these defines a reserved
// namespace value anywhere. The generic escape-hatch mechanism itself
// (request_extension.go, AI-12.3) takes an arbitrary caller-supplied
// namespace and reserves none of its own, by design — R-REX-006 states
// "Layer 1 MUST NOT maintain any list of recognised namespaces", so
// Layer 1 was never going to be the source of this value either.
//
// Because this node's own instruction forbids inventing the value here,
// R-ART-019's escape-hatch merge (tasks.md 4.7-4.11: own-namespace merge,
// foreign-namespace twin comparison) is NOT implemented in this slice.
// appendBody's own splice-point comment (body.go) marks exactly where it
// would land once the value exists. This is a disclosed gap, not a
// silent omission: tasks.md's Phase 4 evidence log records the search
// performed and the requirement blocked, the same posture R-ART-001
// takes toward an unobtainable citation (S-ART-004) — halted and
// recorded, not invented locally.
package openaicompat
