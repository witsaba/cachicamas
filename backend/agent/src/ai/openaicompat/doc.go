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
// adds no new sentinel: every refusal this package constructs (AI-26.6's
// reasoning refusal, landed below, and AI-26.8.2's exhaustive-walk
// refusals, a later slice) is
// ai.PreStreamFailure(ai.FailureReport{Category:
// ai.FailureCategoryUnsupportedCapability, Cause: <error naming the
// unsupported feature>}), reachable via errors.Is(err,
// ai.ErrUnsupportedCapability) through ai.Failure.Is — the same uniform
// door for both, never a per-site construction. policy.go's refuse is
// that one door; reasoning_refusal_test.go's
// TestPolicy_NoNewSentinelsExported keeps the "zero new sentinels" claim
// mechanical, not only prose, for every future change to this package.
//
// # Reasoning replay refuses; Layer 2 must strip first (AI-26.6, R-ART-015)
//
// A neutral request whose assistant message carries a reasoning content
// part — [ai.ReasoningStateText], [ai.ReasoningStateRedacted] or
// [ai.ReasoningStateTokenOnly], at any position, in any message —
// fails [Translate]. It does not drop the part, does not render it as
// text (no accessor yields reasoning content as text — AI-07.1 item 2,
// and mechanically impossible for the redacted and token-only states
// anyway, which have no text to render at all), and does not smuggle it
// through this package's own reserved provider-extension namespace
// ("Escape hatch: reserved namespace", above): that namespace is
// caller-owned by contract (request_extension.go), not an
// adapter-internal channel this package may repurpose for its own
// unexpressed content.
//
// This is stated plainly here so a later reader who hits it never mistakes
// it for an unfixed defect:
//
//   - A reasoning-bearing transcript fails hard, here, every time, for as
//     long as this vendor's wire schema carries no field for it. Every
//     reasoning state refuses identically; position — first, last or
//     mid-message, in the first message or a later one — never changes
//     the outcome (S-ART-051, S-ART-052), because policy.go's
//     refuseReasoning walks the whole request before Translate calls
//     appendBody at all, never discovering a reasoning part mid-render.
//   - The remedy is consumer-owned, not this package's: Layer 2 MUST strip
//     every reasoning content part from a message before handing that
//     message's history to this adapter for replay. AI-03's standing
//     rule 4 anticipates exactly this shape of gap — "Layer 1 never
//     substitutes for an absent capability... Layer 1 states the
//     absence, the consumer owns the fallback" — so an unstripped replay
//     failing here is this package doing its one documented job, not a
//     bug report waiting to happen.
//   - The duty is written down, and it is routed to AI-40 ("Publish the
//     Layer 2 readiness contract", doc 0002 — Layer 1's own exit
//     milestone): AI-40.2's capability matrix and AI-40.3's compatibility
//     statement are where "Layer 2 must strip reasoning before replay"
//     becomes part of the frozen v1 surface a consumer reads, not a fact
//     only this file states. AI-26 needs no node of its own to publish
//     that — AI-40 already owns the job.
//
// The classification is a refusal (AI-19's [ai.PreStreamFailure] +
// [ai.ErrUnsupportedCapability]), not a validation fault (AI-04's
// [ai.Violation], which AI-25 correctly uses for a construction-time
// defect): the request is neutrally valid — AI-10 and AI-12 both accept
// it — and what fails is this vendor's own expressiveness, never the
// caller's contract. See "Refusal taxonomy: AI-25 vs AI-26 (NFR-ART-E)",
// above, for the full comparison table and its AI-03 §10.4 citation.
//
// If a future vendor update adds a wire field this adapter can express
// reasoning content through, AI-26.6 is revisited from refusal back to
// rendering — the same living-graph reopen-trigger discipline "System
// role: system, not developer", above, already applies to this package's
// other vendor-schema-shaped decision.
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
// That conditioning is why this proof (cache_marker_test.go) walks every
// region ai.CacheRegions() enumerates: appendMessageObject (message.go,
// AI-26.3; originally body.go, AI-26.1), appendSystemMessageObject
// (system.go, AI-26.2) and appendToolObject (tool.go, AI-26.4) are each a
// per-carrier renderer that emits a carrier's full substantive content —
// role/text, segment text in the system role, and a tool's
// name/description/schema bytes, respectively — onto the wire, so all
// three sub-tests now run to completion, non-vacuously, under S-ACB-035.
// None is recorded SKIP.
//
// At AI-26.2's own slice-open, tool.go did not yet exist (it landed at
// AI-26.4, slice 3), and the tools sub-test recorded a disclosed t.Skip
// rather than a fabricated PASS over declarations Translate did not yet
// render at all — cache_marker_test.go's own loop/switch structure needed
// no edit to pick that region up once tool.go landed, exactly as this
// section originally predicted. This paragraph is a belated correction:
// AI-26.4's own tasks.md evidence log claimed this section was "updated
// to reflect completion" at the time, but the update itself was not
// actually made until AI-26.3 found the stale two-of-three text while
// editing this file for its own, unrelated reason and corrected it here
// — recorded per this milestone's own practice of disclosing a correction
// in place rather than silently overwriting an inaccurate prior claim.
//
// The messages and system sub-tests' own twin comparisons were green on
// first write, for the same structural reason (neither
// appendMessageObject nor appendSystemMessageObject ever reads a
// carrier's IsCacheBoundary()): falsifiability for each was proven by a
// staged, reverted mutation — temporarily rendering a marked carrier's
// marker as a scratch wire field, observing the sub-test fail, then
// reverting (S-ART-023). The tools region's own analogous staged
// mutation, gated on Tool.IsCacheBoundary(), is recorded in tasks.md's
// Phase 3 evidence log.
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
// # Escape hatch: reserved namespace (AI-26.7, R-ART-019)
//
// R-ART-019 requires this adapter's own reserved provider-extension
// namespace value to be read from AI-25's landed artifact, never
// re-invented here. Before writing any escape-hatch code, this slice
// searched AI-25's landed artifact exhaustively: every AI-25 non-test
// source this package owns (client.go, credential.go, endpoint.go,
// request.go), this file's own AI-25-era sections above, AI-25's own SDD
// artifact (openspec/changes/cachicamas-ai-provider-client/{proposal,
// spec,design,tasks}.md), AI-24's merged decision
// (cachicamas-ai-first-provider-decision/decision.md), and doc 0002's own
// AI-25 charter (§§ AI-25 … AI-25.3). None of these defines a reserved
// namespace value anywhere. The generic escape-hatch mechanism itself
// (request_extension.go, AI-12.3) takes an arbitrary caller-supplied
// namespace and reserves none of its own, by design — R-REX-006 states
// "Layer 1 MUST NOT maintain any list of recognised namespaces".
//
// # Coordinator ruling: the value is "openaicompat", and this milestone defines it
//
// The instruction to read the value from AI-25 was itself the defect,
// not this package's search: AI-25 was never given a task to define one.
// Reserving a namespace was always this milestone's own job, not AI-25's
// (construction) or AI-12's (the neutral escape-hatch mechanism, which
// deliberately reserves nothing of its own per R-REX-006 above) — this
// milestone is the sole owner of the wire body a reserved value would
// merge into.
//
// The value is the exported constant [Namespace] (option.go), equal to
// the string "openaicompat":
//
//   - It is this adapter package's own identity, so a caller writing
//     ai.WithProviderExtension(openaicompat.Namespace, value) is
//     unambiguous about which adapter the value targets.
//   - It is dialect-named, not vendor-branded — the same reasoning that
//     named this package openaicompat rather than something
//     OpenAI-specific: OpenRouter, vLLM, Ollama, LocalAI and other
//     OpenAI-compatible servers are all in this adapter's scope, not
//     first-party OpenAI alone.
//   - Namespace is exported specifically so a caller references it by
//     name rather than hand-typing the string: a caller who mistypes a
//     hand-written namespace silently gets "foreign namespace, ignored
//     whole" (below), with no error anywhere — a miserable failure mode
//     an exported constant is the cheapest guard against.
//
// # Own namespace merges; every other namespace is ignored whole
//
// appendExtensionFields (option.go) reads req.ProviderExtension(Namespace)
// and, when present, splices its Value() bytes raw into the wire body,
// last, merged as additional top-level members — the same raw-splice
// reasoning tool.go's schema splice already establishes (R-ART-010): a
// caller-supplied JSON fragment must survive byte-exact, never
// decoded/re-encoded. Every namespace other than Namespace is never
// separately read or rendered by this function at all: "ignored whole"
// is what NOT reading any other namespace produces structurally, the
// same reasoning appendMessageObject/appendSystemMessageObject never
// reading IsCacheBoundary() already established for cache-boundary
// markers (above).
//
// "Ignored whole" is proven by the same twin-comparison shape R-ART-006
// established: a request carrying a foreign-namespace extension and its
// extension-free twin translate byte-identically (extension_test.go),
// anchored against a hard-coded expectation literal on both sides — not
// merely against each other — so the proof cannot pass vacuously either
// because nothing was implemented yet, or because both sides shared one
// (potentially buggy) code path. Falsifiability is a staged, reverted
// mutation that rendered every namespace rather than only the reserved
// one, breaking both the twin comparison and the hard-coded anchor for a
// single foreign namespace and for three attached at once.
//
// # No merging of consecutive same-role messages (AI-26.3, R-ART-009)
//
// appendMessageObject (message.go) renders exactly one wire message
// object per ai.Message, unconditionally: it carries no memory of a
// previously rendered message's role, and appendMessagesField's own loop
// (body.go) never groups, folds or skips a message because an adjacent
// one shares its role. A run of two, three, or any longer sequence of
// consecutive same-role ai.Message values therefore always produces that
// many distinct wire message objects, never fewer — structurally, by
// construction, not by a check that could be bypassed.
//
// This is a deliberate reading of claim 4 (this file's own "Wire-shape
// provenance" section), not an oversight this adapter happened not to
// hit. The cited specification imposes no role-alternation constraint on
// the messages array at all, and the vendor's own documented
// parallel-function-calling flow appends multiple CONSECUTIVE
// role:"tool" messages onto one messages list as its supported, canonical
// shape for a single turn's several tool results — evidence for a role
// repeating consecutively being an accepted, working pattern, not merely
// for a distinct role existing once. Merging would therefore not be a
// defensible simplification: it would silently discard caller-authored
// message boundaries the vendor is documented to accept as given, with no
// error the caller could observe anywhere on the response side — exactly
// the failure mode S-ART-031 names.
//
// message_test.go's "two/three consecutive same-role messages render as
// ... distinct wire objects" registered cases exercise this directly,
// each anchored against a hard-coded expectation. Falsifiability was
// proven by a staged, reverted mutation that collapsed each run of
// consecutive same-role messages down to one representative wire object
// (S-ART-033) — dropping the rest of the run is the mutation's concrete
// shape, chosen because appendBody's byte-append architecture has no
// in-place buffer to concatenate two already-written wire objects'
// content arrays without a larger refactor; dropping still violates
// R-ART-009's "not merged" contract exactly as a content-concatenating
// merge would, and is caught by the same registered cases either mutation
// shape would break. See tasks.md's Phase 5 evidence log for the exact
// staged change and its observed, then reverted, failures.
//
// # Tool calls render in their own tool_calls array; tool results render as their own distinct role:"tool" messages (AI-26.5, R-ART-012 … R-ART-014)
//
// message_test.go's own hand-off note (partKindDispositions,
// PartKindToolCall/PartKindToolResult, recorded at AI-26.3) already
// stated the shape this slice discharges: a tool call renders as an
// element of a SEPARATE top-level tool_calls array on the assistant
// message object, never as a content-part array element; a tool result
// renders as its own distinct role:"tool" wire message, never a block
// nested inside a user-role message and never a nested object on another
// message (R-ART-012, AI-24 §6's given shape, not re-derived here).
//
// splitToolCalls (message.go) is where the two paths separate: it
// partitions a non-tool-role message's content, once, into its
// non-tool-call remainder (rendered as "content") and its tool-call
// parts (rendered as "tool_calls"), so appendMessageContent and
// appendContentPartObject never need to know a tool-call part can occur
// in their own input at all — PartKindToolCall reaches neither. A
// RoleTool message is intercepted whole, before any of that, by
// appendMessageObject's own dispatch (appendToolResultMessages) — request.go's
// own rolePermittedKinds table already restricts such a message to
// PartKindToolResult alone, enforced by ai.NewRequest before Translate is
// ever reached, so PartKindToolResult reaches neither path either.
//
// # Tool-call identifier: pass-through only, the minting branch has no subject (R-ART-013)
//
// Doc 0002's amended AI-26.5 charter (2026-08-03) records that this
// vendor assigns tool-call identifiers on the call's own opening delta,
// so the requirement's synthetic-minting branch is marked NOT-APPLICABLE
// for this adapter, not silently skipped — the text stays in force for a
// future adapter whose vendor assigns none. What this adapter proves
// instead is the pass-through half: appendToolCallObject's "id" and
// appendToolResultObject's "tool_call_id" both splice ai.ToolCall.ID()/
// ai.ToolResult.CallID() through appendJSONString unchanged — no
// generation, derivation, normalization or truncation on any path.
// Result-to-call correlation (R-ART-014) is a direct consequence, not a
// separate algorithm: appendToolResultObject performs no lookup, no
// matching and no positional reasoning of its own — the wire
// tool_call_id is literally the value the caller supplied when it built
// the ai.ToolResult, so correlation survives translation because nothing
// on this path has an opportunity to substitute anything else for it.
// tool_result_test.go's own interleaved multi-call case and its staged,
// reverted position-based mutation (S-ART-048/049) both exist because a
// NON-interleaved fixture cannot distinguish this direct pass-through
// from a plausible positional bug that happens to produce the same bytes
// when calls and results are declared in matching order.
//
// # Assistant "content" is omitted, never null, when a message carries only tool calls
//
// Claim 1.0.5 (this file's own wire-shape provenance section) settled
// content's shape as oneOf[string, array] — it was never settled as
// oneOf[string, array, null]. An assistant message whose content is
// entirely tool calls (ai.rolePermittedKinds' own RoleAssistant row
// permits Text, Reasoning and ToolCall in any combination, including
// zero Text parts) therefore has nothing citable to put in "content" at
// all: appendMessageObject omits the field outright rather than
// fabricating a "content":null this citation gate never licensed. This
// is not a new convention invented for this case — it is the same
// presence-flag, omit-when-absent shape every other optional field in
// this package already uses (option.go's generation options and
// tool_choice, tool.go's own "tools" field, body.go's extension
// members): "the caller said nothing here" renders as no field, never a
// placeholder value standing in for absence.
//
// # No distinct wire field for a failed tool result (S-ART-044)
//
// AI-24 §6's given shape for a tool-role message carries no field
// recording whether the tool failed, and this package invents none:
// appendToolResultObject renders result.Content() unconditionally,
// identically whether result.Failed() is true or false, never branching
// on it. tool_result.go's own doc comment states the reason at the
// source: a failing tool's answer is content the model reasons about,
// not a fault this package's taxonomy names ("a taxonomy here would be
// Layer 2's policy wearing a Layer 1 type"). "Not silently rendered as
// success" (S-ART-044) is exactly what this produces: there is no
// separate "success" code path a failed result could fall into, because
// there is only one code path. tool_result_test.go's
// TestToolResult_FailedAndSucceededRenderIdentically proves this beyond
// one hand-picked case: two results differing only in identifier,
// content and Failed() translate to bodies differing only in identifier
// and content — substituting one identifier for the other in the
// succeeded body's own bytes reproduces the failed body exactly.
package openaicompat
