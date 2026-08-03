# Spec — provider configuration and client construction

> **Milestone**: AI-25 — Provider configuration and client construction (doc 0002, Wave 4 "Build", the wave's **first** code milestone) · **Nodes**: AI-25.1 `[leaf]` injected construction · AI-25.2 `[guard]` no ambient authority · AI-25.3 `[leaf]` test-server viability
> **Introduced by**: `openspec/changes/cachicamas-ai-provider-client/`
> **Status**: **change-folder delta** — promoted to `openspec/specs/ai-provider-client/spec.md` at archive
> **Project**: cachicamas (witsaba) · **Target**: a dedicated OpenAI-compatible adapter package under `backend/agent/src/ai/`, distinct from `package ai`. *Its exact placement, file decomposition and type shapes are design's; this spec constrains only observable behaviour.*
> **Requirement IDs**: `R-APC-0NN` · **Scenario IDs**: `S-APC-0NN`
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable, either by a test runnable under `go test -race` or as an explicit **review obligation** — a claim discharged by reading landed prose, a diff or a document rather than by executing code. Review obligations are marked `*(review)*` on the scenario line. The distinction is operational, not cosmetic: `NFR-APC-G`'s red→green evidence rule applies to the **runnable** set only, because a prose-inspection scenario has no red phase to record.
> **Binding predecessors, cited by identifier and never modified**:
> [`ai-validation-errors`](../../../../specs/ai-validation-errors/spec.md) (AI-04) — the caller-contract failure taxonomy, its rule-class sentinels and its positional context, reused **unchanged and unextended**;
> [`ai-provider-errors`](../../../../specs/ai-provider-errors/spec.md) (AI-19) — deliberately **not** used here; construction precedes any request and any I/O, so its axes are undefined at this point;
> [`ai-model-provider`](../../../../specs/ai-model-provider/spec.md) (AI-20) — the interface this adapter must not foreclose satisfying;
> [`agent-module-scaffold`](../../../../specs/agent-module-scaffold/spec.md) (AI-00) — `S-AGM-030` the two-entry `src/` listing and `S-AGM-032` the `package ai` documentation claims, both of which MUST survive this change unchanged
> **Depends on**: AI-24 (decided: OpenAI-compatible dialect, raw `net/http`, injected opaque credential, zero module requires) · **Blocks**: AI-26 … AI-32
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-25.1 … AI-25.3, **as corrected by amendment A1** (see *Correction of record* below) · ADR 0005 (nothing below the composition root reads the environment)

---

## Purpose

AI-24 named a dialect, a transport and a credential boundary. Nothing yet *is* that adapter, and AI-26 (request translation), AI-27 (SSE decode) and AI-32 (error mapping) each need a constructed, configured value to build against — none of them authorized to invent its own configuration story.

Three failures this spec exists to prevent are not hypothetical:

1. **The whole-request timeout.** The naive construction of an HTTP client for a streaming adapter sets a whole-request cap, which per the standard library's own words includes reading the response body and will interrupt that read. Every stream longer than the cap dies mid-read and surfaces four milestones later as an unexplainable truncation. Because an equivalent cap can be reimposed internally through a per-request deadline, **field inspection cannot prove its absence.** This spec therefore requires a behavioural proof.
2. **A guard that cannot bite.** "The adapter reads no environment variable" is a claim about **call sites**. A guard that is believed but never shown to fail is worse than no guard at all.
3. **The proxy-from-environment vector.** The standard library's process-wide default transport resolves its proxy *from the environment*, and an HTTP client with no transport set falls back to that default. An adapter that merely routes through it therefore reads the environment while mutating nothing and calling nothing forbidden — invisible to `S-APC-012`'s no-mutation check and invisible to a call-site scan, whose selector resolves to the HTTP package rather than to the environment package. `R-APC-007` closes it explicitly.

## Correction of record — the guard's mechanism

Doc 0002's AI-25.2 test list describes the guard as "an AST import-and-call scan in the AI-00.3 style". **That description is defective and this spec overrides it** (living-graph amendment **A1**, recorded in AI-24's artifact and referenced here rather than duplicated). AI-00.3 is an **import-path transitive-closure** scan against a deny-by-default allowlist; it says nothing about call sites, and the HTTP transport this adapter is required to use transitively imports the operating-system package. Reusing that mechanism would either false-positive on legitimate transport use or miss a narrow environment read. AI-25.2's mechanism is a **call-site scan over the adapter package's own source files**. The scan's implementation is design's; that it is call-site-scoped, mechanical and demonstrably biting is required here.

## Delta status against existing specs

`openspec/specs/` was re-read at spec time. It carries no adapter-construction, endpoint-configuration or ambient-authority capability. `ai-provider-client` is **new**: this delta contains **ADDED requirements only** — no MODIFIED, no REMOVED and no RENAMED requirement.

Per the precedent set by `ai-fake-provider` `NFR-AFP-E` and `ai-stream-testkit` `NFR-STK-F`, the documentation obligations this change carries — including the corrections to `package ai`'s stale comments — are stated as **non-functional requirements of this new capability**, not as a delta on `agent-module-scaffold`. `S-AGM-030` and `S-AGM-032` MUST hold verbatim after this change.

## Definitions used by this spec

- **The adapter** — the OpenAI-compatible provider implementation this milestone begins. At AI-25 it is a configured value only; it acquires streaming behaviour at AI-26 and later.
- **Construction** — the act of producing that configured value from injected inputs. It happens before any request value exists and before any byte crosses a socket.
- **The adapter-built client** — the HTTP client the adapter constructs for itself when the caller injects none. The adapter owns its configuration end to end.
- **An injected client** — an HTTP client the caller supplies. The adapter uses it verbatim; its configuration is its **injector's**, decided at the composition root.
- **A configuration fault** — a caller-supplied configuration input that violates a construction rule and is decidable from the inputs alone, without contacting anything.
- **A stub transport** — a test-supplied substitute at the HTTP round-trip boundary that observes what the adapter would send without a network. It is how "the injected value was actually used" is proven, in place of reading a field.
- **The connect-phase bounds** — bounds that constrain establishing a connection and receiving response headers. They are bounded **before** the body arrives.
- **A whole-request cap** — any bound, however imposed, whose expiry terminates an in-progress response-body read.
- **Ambient authority** — reading an environment variable, touching the filesystem, or spawning a process: any capability taken from the surroundings rather than received by injection.
- **A bite proof** — a deliberate scratch violation added to real, scanned source; the guard's resulting **failure** recorded; the violation then dropped and the suite re-run green. Per doc 0002's guard-leaf grammar, a `[guard]` node closes only on a recorded bite proof.
- **A review obligation** — a scenario discharged by reading landed prose, a diff or a document. Marked `*(review)*`. It carries no red phase and is therefore outside `NFR-APC-G`'s red→green evidence rule.

---

## AI-25.1 — Injected construction `[leaf]`

### R-APC-001 — Every injected value is used, and "accepted then ignored" is a detectable failure

Construction MUST accept the endpoint, the credential and (optionally) the HTTP client from its caller. Each value the caller does supply MUST be demonstrably **used** by the constructed value. Use MUST be observable through a stub transport at the round-trip boundary — that is, by what the adapter would send — and MUST NOT be asserted by reading a stored field, because a stored-but-unused value passes a field assertion and fails in production.

#### Scenarios

- **S-APC-001** — Given construction with an endpoint, a credential and an HTTP client whose round-trip boundary is stubbed, when the adapter is driven to produce one outbound request, then the stub observes it — proving the injected client is the client that carries traffic.
- **S-APC-002** — Given that same observation, when the observed request's scheme, host and path prefix are compared against the injected endpoint, then they derive from it and from no other source.
- **S-APC-003** — Given that same observation, when the observed request's headers are inspected, then the injected credential is present on it.
- **S-APC-004** — Given two adapters constructed with two **different** stubbed clients, when each is driven once, then each stub observes exactly its own adapter's request and neither observes the other's — proving no shared or cached client is substituted.
- **S-APC-005** — Given a deliberately mutated implementation that stores an injected value without using it, when this requirement's tests run, then at least one fails — the assertions detect ignoring, not merely storing.

### R-APC-002 — A configuration fault fails at construction, typed, before any request exists

WHEN an endpoint is malformed or a credential is empty, construction MUST fail. The failure MUST be reported through the **AI-04 caller-contract taxonomy**: `errors.Is` MUST identify the violated rule class, and `errors.As` MUST yield the structural position of the offending input. A malformed endpoint MUST report the malformed-value class; an empty credential MUST report the empty-required-value class. This change MUST NOT define a new sentinel, MUST NOT define a second concrete failure type, and MUST NOT use the AI-19 provider/transport vocabulary — construction is decidable from its inputs alone, without contacting a provider, which is precisely AI-04's domain and explicitly outside AI-19's.

The configuration faults are **exactly these two**. An absent HTTP client is **not** among them; it is the documented selector for the adapter-built path (`R-APC-003`, `NFR-APC-F`).

#### Scenarios

- **S-APC-006** — Given a table of malformed endpoints (empty, whitespace-only, no scheme, an unsupported scheme, an unparseable value, a control character), when each is passed to construction, then each fails, `errors.Is` reports the malformed-value class, and `errors.As` yields a position naming the endpoint input.
- **S-APC-007** — Given an empty credential with an otherwise valid endpoint, when construction runs, then it fails, `errors.Is` reports the empty-required-value class, and the position names the credential input.
- **S-APC-008** — Given any failing construction from the two tables above and a stubbed round-trip boundary, when the failure returns, then the stub observed **zero** round trips — the fault was decided before any request existed.
- **S-APC-009** — Given the merged change, when the module's exported sentinel set is enumerated and compared against the set before the change, then it is identical — zero new sentinels.
- **S-APC-010** — Given a failed construction, when the caller inspects what it returned besides the error, then there is no usable adapter value — a partially configured adapter is never handed back.

### R-APC-003 — Defaults are safe and fixed, and no shared, global or injected client is mutated

There MUST be no default endpoint that could cause an unconfigured or partially configured adapter to reach a real provider — in particular, from a test.

**The HTTP client is optional, and which party owns the bounds follows from that.** The obligation is scoped in two halves, and conflating them is what makes the naive reading incoherent — an adapter cannot both *fix* bounds on a client and *not mutate* that client:

1. **WHEN no client is injected**, the adapter MUST construct its own. That **adapter-built** client MUST carry fixed connect-phase and idle bounds with safe values, and those bounds MUST NOT be caller-injectable: doc 0002's AI-25.1 item 3 requires that the defaults be *safe*, not that they be *overridable*, and an injectable bound permanently widens a construction surface every later milestone must live with. Doc 0002 item 4's phrase "**the constructed client** carries no whole-request timeout" presupposes that the adapter constructs one on this path.
2. **WHEN a client is injected**, it MUST be used **verbatim**. Its bounds are its **injector's** — set at the composition root, on the caller's own instruction — and the adapter MUST NOT require them, MUST NOT override them, and MUST NOT mutate the client or its transport in any way another holder of that value could observe.

Construction MUST NOT mutate any process-wide or package-level HTTP client or transport on **either** path.

#### Scenarios

- **S-APC-011** — Given construction attempted with no endpoint supplied, when it runs, then it fails per `R-APC-002` rather than substituting a vendor endpoint, and no outbound request is possible.
- **S-APC-012** — Given the observable configuration of the process-wide default HTTP client and default transport captured before construction, when construction runs on **both** the injected and the adapter-built path and the same observations are repeated, then every observed value is unchanged and the identities are the same values as before.
- **S-APC-013** — Given one HTTP client value used to construct two adapters, when the second construction completes, then the first adapter's observable outbound behaviour is unchanged and every observable field of the shared client is what it was before either construction — an injected client is used verbatim, never reconfigured.
- **S-APC-014** — Given a deliberately mutated implementation that assigns a bound onto the process-wide default transport, when `S-APC-012` runs, then it fails — the scenario detects global mutation rather than assuming its absence.
- **S-APC-015** — Given the landed construction surface, when a caller enumerates what it may supply, then it is the endpoint, the credential and an **optional** HTTP client, and **no** timeout, deadline or bound value appears among them.
- **S-APC-016** — Given construction with no client supplied, when the resulting client is compared against the process-wide default client and the process-wide default transport, then it is neither of them — the adapter built a fresh one rather than adopting a shared value it could not bound without mutating.

### R-APC-004 — *(non-negotiable)* No whole-request cap exists, proven behaviourally

The **adapter-built** client MUST NOT impose a whole-request cap on a response body read, **by any mechanism** — neither a client-level whole-request timeout nor an internally derived per-request deadline. Absence MUST be proven **behaviourally**: a response whose body is delivered over a span longer than any plausible whole-request cap MUST be read to completion. A field assertion that a timeout equals zero MUST NOT be accepted as the proof, because it does not exclude an equivalent internal deadline.

The proof MUST include a **control** that fails, so that a stream trivially short enough to pass under any cap cannot produce an accidental green.

The adapter MUST NOT derive a deadline of its own on **either** path: a caller's own context MUST remain the mechanism by which a caller bounds a call. Whether an *injected* client carries a whole-request cap is its injector's decision and is outside this requirement, which is why the behavioural proof is stated against the adapter-built client.

#### Scenarios

- **S-APC-017** — Given a local server whose handler writes and flushes several body chunks with deliberate delays totalling more than a chosen span D, when a **control** client carrying a whole-request cap of roughly D/2 reads that response, then the read fails partway through the body — proving the fixture genuinely exercises the footgun rather than being too fast to matter.
- **S-APC-018** — Given that same server, that same handler and that same timing, when the **adapter-built** client reads the response, then it reads the body to completion with every chunk intact and returns no timeout failure.
- **S-APC-019** — Given the adapter-built client and a caller context carrying no deadline, when the long response of `S-APC-018` is read, then no deadline is imposed from inside the adapter — the read is bounded only by the caller and by the server.
- **S-APC-020** — Given a deliberately mutated implementation that reimposes an equivalent cap through a per-request context deadline, when `S-APC-018` runs, then it fails — the proof catches the internal-deadline reimposition, which a field assertion would not.

### R-APC-005 — The connect-phase and idle bounds genuinely exist on the adapter-built client

The adapter-built client MUST carry a non-zero bound on establishing a connection, on completing a TLS handshake, on time-to-response-headers, and on pooled-connection idleness. This requirement exists so that "no bound anywhere" cannot pass as an accidental green for `R-APC-004`.

Unlike `R-APC-004`, this requirement is discharged by **structural** assertion — reading the bounds and asserting each is non-zero and within a stated safe range — and that asymmetry is deliberate rather than inconsistent. Proving a cap is **absent** has an escape route a field read would miss, because an equivalent cap can be reimposed internally; proving a bound **exists** has no such escape route, since a bound cannot be present-but-ineffective in the same way. The behavioural alternatives are also actively worse: a time-to-headers probe would run for the length of the header bound itself on every developer machine, and a connect probe against an unroutable address is typically refused immediately by the operating system, producing a **vacuous green** that never exercises the dial bound at all. A test that passes without exercising its subject is not evidence.

#### Scenarios

- **S-APC-021** — Given the adapter-built client, when its connect and TLS-handshake bounds are read, then each is non-zero and within the safe range the design records.
- **S-APC-022** — Given the adapter-built client, when its time-to-response-headers bound and its pooled-connection idle bound are read, then each is non-zero and within the safe range the design records, and the time-to-headers bound is documented as excluding the body read.
- **S-APC-023** — Given a deliberately mutated implementation that leaves the connect and time-to-headers bounds unset, when `S-APC-021` and `S-APC-022` run, then both fail — the bounds are asserted, not assumed.

### R-APC-006 — Endpoint joining preserves every base segment and produces no doubled separator

An endpoint carrying a path MUST join with a dialect-relative path to a request path that preserves **every** segment of the base and adds the relative segments after them. No segment MUST be dropped and no separator MUST be doubled. This requirement guards a specific, silent footgun: RFC 3986 reference merging treats a base whose path lacks a trailing slash as naming a document, and **replaces** its last segment — so a proxy base ending in a path component would silently lose that component and every request would go to the wrong place with no error anywhere.

Joining MUST be proven by a table covering at least: a base with a trailing slash; a base with a sub-path and **no** trailing slash (the footgun); a root base with no trailing slash; a base with doubled interior separators; an empty relative path; a base carrying a query string; and a relative segment that is itself shaped like an absolute URL.

#### Scenarios

- **S-APC-024** — Given a base whose path is a sub-path **without** a trailing slash, when the dialect-relative path is joined, then the resulting request path retains that sub-path in full and the relative segments follow it — nothing was replaced.
- **S-APC-025** — Given a base whose path ends with a trailing slash, when the same relative path is joined, then the result is identical to `S-APC-024`'s shape, with exactly one separator between the base's last segment and the first relative segment.
- **S-APC-026** — Given the full table of base shapes above, when each is joined, then every result matches its expected request path exactly, with no doubled separator, no dropped segment and no spurious trailing separator.
- **S-APC-027** — Given a base carrying a query string, when a relative path is joined, then the query survives on the result and is not absorbed into the path.
- **S-APC-028** — Given a relative segment shaped like an absolute URL, when it is joined, then it is treated as a literal path component of the base and does **not** redirect the request to another host.
- **S-APC-029** — Given one constructed adapter used to build two requests to two different relative paths, when both request paths are inspected, then neither accumulated the other's segments — joining is non-destructive on the stored base.

### R-APC-007 — The constructed value carries no stubbed streaming behaviour

At AI-25 the adapter is a **configured value only**. It MUST NOT ship a placeholder streaming method that returns an "unimplemented" result, a nil stream, or any other stand-in. Such a stub is a value that lies about its type: it would satisfy the AI-20 interface, admit the adapter into a conformance run, and fail in a way that reads as a contract violation rather than as work not yet done. Streaming behaviour arrives at AI-26. This milestone MUST NOT foreclose satisfying the AI-20 interface later.

#### Scenarios

- **S-APC-030** — Given the landed adapter, when a reader looks for a streaming entry point, then none exists — not even one that returns an "unimplemented" failure.
- **S-APC-031** — Given the landed adapter value, when a test asks at run time whether its type implements the AI-20 provider interface, then the answer is no — the adapter does not falsely advertise satisfaction it has not implemented.
- **S-APC-032** — Given the landed change, when the AI-20 signature guard runs, then it passes unmodified and observes the interface unchanged.

---

## AI-25.2 — No ambient authority `[guard]`

### R-APC-008 — The adapter takes no ambient authority, and this is enforced mechanically

The adapter MUST read no environment variable, MUST touch no filesystem path, and MUST spawn no process. Its credential and its configuration arrive **only** by injection. Enforcement MUST be **mechanical** — a scan that a test runs and that fails the suite — not reviewer vigilance.

The scan MUST operate on **call sites within the adapter package's own sources**, not on the transitive import closure, because the required HTTP transport transitively imports the operating-system package and an import-closure scan therefore cannot express this rule.

The forbidden set MUST be defined by **package**, deny-by-default, rather than by a curated list of function names — a curated list silently admits every function added later. It MUST cover the operating-system package, the process-execution package, the low-level system-call package and the deprecated legacy I/O package, which is broader than the charter's literal three verbs. That breadth is intended.

The scan MUST resolve a package reference through its **local identifier**, so that renaming an import does not bypass the rule.

**The guarantee is scoped, and the scope MUST be stated rather than implied.** It covers the adapter's own sources and the **adapter-built** client. It does **not** and cannot cover an injected client's transport: that value was configured at the composition root, and any environment it consults was consulted there, on the caller's instruction. Per ADR 0005, nothing below the composition root reads the environment; an injected client is not below it.

#### Scenarios

- **S-APC-033** — Given the landed adapter sources, when the guard runs as part of the suite, then it passes and reports no violation.
- **S-APC-034** — Given the guard's forbidden set, when it is enumerated, then it names the operating-system, process-execution, system-call and legacy-I/O packages by package rather than by function, and denies by default.
- **S-APC-035** — Given a guard failure, when its message is read, then it names the offending source file, the line, and the forbidden package — a reviewer can act on it without re-deriving it.

### R-APC-009 — The adapter-built client consults no environment-derived proxy

The **adapter-built** client's transport MUST leave its proxy resolution unset, and MUST NOT adopt the standard library's process-wide default transport, whose proxy resolver reads the proxy environment variables. An adapter routing through that default would read the environment while mutating nothing and calling nothing the call-site scan can see — the scan's selector resolves to the HTTP package, not to the environment package — so `R-APC-008`'s guard cannot close this vector and this requirement must close it directly.

A **structural** assertion is acceptable here specifically, and the exception is narrow and reasoned: the proxy resolver is the only environment-consulting field on the transport, so unlike `R-APC-004`'s absence proof there is no equivalent-reimposition route by which the read could return through another field.

An **injected** client's transport remains its injector's responsibility, per `R-APC-008`'s scope statement.

#### Scenarios

- **S-APC-036** — Given the adapter-built client's transport, when its proxy resolver is read, then it is unset — no environment-derived proxy is consulted.
- **S-APC-037** — Given a deliberately mutated implementation whose adapter-built client routes through the process-wide default transport, when `S-APC-016` and `S-APC-036` run, then at least one fails — the proxy vector is detected rather than merely asserted away in prose.

### R-APC-010 — *(non-negotiable)* The guard is closed only by a recorded bite proof against **four** violation shapes

Per doc 0002's guard-leaf grammar, this node MUST NOT be marked complete on a green run alone. It closes only when the guard is **shown to fail** against deliberate scratch violations added to real, scanned, non-test adapter source, with each red run recorded, and the violations then dropped and the suite re-run green.

**Four** violation shapes are required, not one, because each exercises a distinct branch and an untested branch is not a guard:

1. a plain environment read through a normally named import;
2. an environment read through an **aliased** import, proving the local-identifier resolution bites;
3. a **process spawn**, proving the forbidden set extends past environment reads;
4. an environment read through a **dot-import**, proving the independent dot-import branch bites.

The dot-import case MUST be treated as a violation **in its own right** — a dot-import of a forbidden package is forbidden regardless of whether a call to it is found, because resolving bare identifiers to their package would require full type information.

#### Scenarios

- **S-APC-038** — Given a plain environment read added to a real non-test adapter source, when the suite runs, then the guard fails and names that file and that package, and the red output is recorded.
- **S-APC-039** — Given an environment read reached through an aliased import, when the suite runs, then the guard fails — the alias did not bypass it — and the red output is recorded.
- **S-APC-040** — Given a process spawn added to a real non-test adapter source, when the suite runs, then the guard fails and names the process-execution package, and the red output is recorded.
- **S-APC-041** — Given a dot-import of a forbidden package added to a real non-test adapter source, when the suite runs, then the guard fails on the dot-import itself, and the red output is recorded.
- **S-APC-042** — Given all four scratch violations dropped, when the suite runs again, then it is green and that output is recorded alongside the four red runs.

### R-APC-011 — The scan covers non-test sources only, and the exclusion is justified in place

The scan MUST cover the adapter package's **non-test** sources and MUST exclude its test sources. The exclusion MUST be justified in the guard's own documentation, citing the in-repo precedent: this repository's sibling import guard already records that including test scope pulls in the testing package, which itself imports the operating-system package, so a guard that scanned test scope could never pass. The adapter's own tests legitimately use the standard-library HTTP test server. The guard's own source is likewise exempt from its rule as meta-tooling, exactly as the sibling AI-00.3 guard freely uses process execution to implement its check.

#### Scenarios

- **S-APC-043** *(review)* — Given the guard's documentation, when a reader looks for why test sources are excluded, then the reason is stated in place with the in-repo precedent cited, rather than left to be rediscovered.
- **S-APC-044** — Given adapter tests that use the standard-library HTTP test server, when the guard runs, then it stays green — the exclusion is real and not merely documented.
- **S-APC-045** — Given the guard's own source, when the scan's file selection is exercised, then that source is excluded, and the exemption is stated in the guard's documentation rather than left implicit.

### R-APC-012 — The scan's one honest limitation is recorded as an explicit non-requirement

Without full type information, a call-site scan can **false-positive** on a local identifier that shadows a forbidden package's name. Closing that gap would require a non-standard-library static-analysis dependency, which would spend the zero-module-requires invariant AI-24 bought. The limitation MUST therefore be **recorded, in place, as an accepted non-requirement** — stated rather than papered over — and its recording MUST be scoped to this change, so a later milestone may revisit it by writing the ADR that a new dependency would need.

#### Scenarios

- **S-APC-046** *(review)* — Given the guard's documentation, when a reader looks for its known limitations, then the local-shadow false-positive is stated explicitly, together with the dependency that closing it would cost.
- **S-APC-047** *(review)* — Given that record, when a reader asks whether the decision is permanent, then it reads as scoped to this change and reversible by a later ADR, not as a forever prohibition.

---

## AI-25.3 — Test-server viability `[leaf]`

### R-APC-013 — Against a local test server, a request arrives carrying the credential in the dialect's header shape

WHEN the adapter is constructed against a local test server and driven to issue one request, that request MUST reach the server, MUST arrive at the joined request path per `R-APC-006`, and MUST carry the injected credential in the header shape the OpenAI-compatible dialect expects. This is the milestone's only sanctioned request; it exists solely to prove that construction wired the injected values correctly end to end.

The probe MUST NOT be routed through an environment-derived proxy. This is not theoretical: the standard library's proxy resolver exempts only the literal host name `localhost`, so a local test server's loopback-address URL **is** proxy-eligible, and the probe would silently misroute on any developer machine with a proxy environment variable set. `R-APC-009`'s unset proxy resolver is what makes this probe trustworthy, which is why the probe MUST exercise the adapter-built client rather than an injected one.

#### Scenarios

- **S-APC-048** — Given the adapter constructed against a local test server on the adapter-built client path, when it issues one request, then the server's handler observes exactly one request.
- **S-APC-049** — Given that observed request, when its authorization header is read, then it carries the injected credential in the dialect's expected header shape.
- **S-APC-050** — Given that observed request, when its path is read, then it equals the path `R-APC-006` requires for that base and that dialect-relative path.
- **S-APC-051** — Given a differently valued credential injected into a second construction, when a second request is observed, then the header carries the second value — the attachment is derived from the injection, not hard-coded.
- **S-APC-052** — Given a proxy environment variable set in the test process to an address that would fail, when the probe runs, then it still reaches the local test server — the adapter-built client did not consult it.

### R-APC-014 — This node proves attachment only; it is explicitly not a wire-level secrecy guarantee

`R-APC-013` MUST NOT be read, cited or extended as a statement that the credential is protected from disclosure **on the wire or in provider-returned content**. This milestone asserts **attachment**, nothing more. The bound on what a wire error body may carry is AI-32.5's; the exhaustive sentinel sweep is AI-36.1's. The exclusion MUST be stated in the landed spec and in the probe's own documentation, so a later reviewer does not read a wire-level secrecy guarantee into a viability probe.

**A type-shape opacity assertion on the credential value is permitted and is expressly not such a guarantee.** Asserting that the credential's default and verbose rendering forms, and its serialized form, do not contain the token is a cheap, local property of the value's own type. It prevents accidental logging of the credential by a later contributor who formats a configuration value, and it is squarely inside this milestone's construction scope. It makes no claim about the wire, about provider-returned content, or about what any other component may do with a token it obtained elsewhere.

#### Scenarios

- **S-APC-053** — Given the credential value, when it is rendered through the default, string, verbose and Go-syntax formatting forms and through serialization, then no rendering contains the token.
- **S-APC-054** *(review)* — Given the landed spec and the viability probe's documentation, when a reader looks for a wire-level secrecy claim, then each states that only attachment is proven and names AI-32.5 and AI-36.1 as the owners of the wire-level bounds.
- **S-APC-055** *(review)* — Given this milestone's tests, when a reviewer enumerates their assertions, then none asserts wire-level redaction, provider-content non-disclosure or log safety of the kind AI-32.5 and AI-36.1 own — `S-APC-053`'s type-shape opacity assertion is the one permitted exception and is scoped to the credential value itself.

---

## Non-functional requirements

### NFR-APC-A — Dependency purity

This change MUST add no module dependency. The adapter and its guard MUST import only the standard library and the module's own packages. `backend/agent/go.mod` MUST still declare **zero** `require` directives, and both AI-00 import guards MUST still pass. Because no new top-level dependency is introduced, no ADR gate fires.

- **S-APC-056** — Given the change merged, when `go.mod` is read and both AI-00 import guards run, then it declares no require and both guards pass.
- **S-APC-057** *(review)* — Given the new adapter package's placement, when the AI-00 forward guard's allowlist is inspected in the diff, then it is unmodified — the package is admitted by the existing module-prefix rule with no allowlist edit.

### NFR-APC-B — Existing Layer 1 behaviour is read, never changed

This change MUST NOT alter any behaviour in `package ai`. Its only edits there are comment-level (`NFR-APC-C`). `S-AGM-030` MUST still hold — `backend/agent/src/` MUST still list exactly two entries, which the adapter's placement as a subdirectory of `ai` preserves — and every AI-00 … AI-23 test MUST pass identically with or without this change.

- **S-APC-058** *(review)* — Given the merged diff, when every edit under `src/ai/` is inspected, then each is comment-only and no declaration, signature or behaviour changed.
- **S-APC-059** — Given the merged change, when `backend/agent/src/` is listed, then it still contains exactly two entries and `S-AGM-030` still passes.
- **S-APC-060** *(review)* — Given the change reverted in isolation, when the AI-00 … AI-23 suites run, then they pass exactly as before.

### NFR-APC-C — Three stale comment claims are corrected, and `S-AGM-032`'s claims survive verbatim

Three landed comments are stale under AI-24's decision and this milestone's placement, and MUST be corrected:

1. `backend/agent/src/ai/import_boundary_test.go` names AI-24 as a milestone that will add a second allowlist entry. Under AI-24's decision, AI-24 adds no entry and never will; only the observability milestone remains a candidate.
2. `backend/agent/src/ai/doc.go` claims `package ai` will own the concrete vendor adapters. It will not: the adapter is a distinct package, precisely so `R-APC-008`'s scan has an enforceable scope.
3. `backend/agent/src/ai/doc.go` separately claims that a transport milestone will add a dependency requiring its own ADR. It does not: the transport is the standard library, and `NFR-APC-A` records that no ADR gate fires.

`S-AGM-032`'s claims — that the documentation states the Layer 1 boundary, states the import rule, and cites ADR 0005 § D1 — MUST survive the edit **verbatim**. This obligation is stated here as an NFR of the new capability, following `NFR-AFP-E` and `NFR-STK-F`, and MUST NOT be written as a delta on `agent-module-scaffold`.

- **S-APC-061** *(review)* — Given the landed `import_boundary_test.go`, when a reader looks for a future second allowlist entrant, then AI-24 is no longer named and only the remaining candidate is.
- **S-APC-062** *(review)* — Given the landed `doc.go`, when a reader looks for who owns the concrete vendor adapters, then `package ai` no longer claims to, and the ownership it does claim is the contract rather than a vendor's satisfaction of it.
- **S-APC-063** *(review)* — Given the landed `doc.go`, when a reader looks for the dependency/ADR paragraph, then it no longer claims a transport milestone will add a dependency.
- **S-APC-064** — Given the landed `doc.go`, when `S-AGM-032` is re-run, then it passes: the Layer 1 boundary, the import rule and the ADR 0005 § D1 citation are all still stated.

### NFR-APC-D — The adapter's own documentation states the rules that erode first, and the scope of its guarantees

The adapter package's documentation MUST state, in its own text:

1. that no whole-request cap may be imposed by any mechanism, and **why** — that such a cap kills every stream longer than it and surfaces as a mid-read death far from its cause;
2. that the package takes no ambient authority and receives its credential only by injection;
3. that the adapter-built client's proxy resolution is deliberately unset, and why — that the standard library's default resolver reads the environment;
4. that the no-ambient-authority guarantee is **scoped** to the adapter's own sources and its adapter-built client, and that an injected client's transport is its injector's responsibility.

All four are rules a later contributor would otherwise "fix" in good faith. Item 4 in particular prevents the guarantee being read as absolute and then found false.

- **S-APC-065** *(review)* — Given the landed package documentation, when a reader looks for the timeout posture, then it is stated with its reason, not merely implied by the absence of a setting.
- **S-APC-066** *(review)* — Given the same documentation, when a reader looks for the credential and ambient-authority posture, then injection-only is stated explicitly.
- **S-APC-067** *(review)* — Given the same documentation, when a reader looks for the proxy decision, then its deliberate absence and its environment-reading reason are both stated.
- **S-APC-068** *(review)* — Given the same documentation, when a reader asks how far the no-ambient-authority guarantee reaches, then the scope boundary at an injected client is stated explicitly rather than left to be assumed absolute.

### NFR-APC-E — Determinism under the race detector

Every behaviour this spec requires MUST hold under `go test -race`. The paired timing comparison of `R-APC-004` MUST be written to survive a loaded developer machine: it MUST use a **wide ratio** rather than tight absolute values, MUST assert the **shape** of the control's failure rather than its exact duration, and MUST NOT be run in parallel with other tests. No assertion MUST depend on elapsed wall-clock time for its correctness where a deterministic mechanism exists. In particular, no scenario in this spec MUST be discharged by a test whose runtime is the length of a configured bound, and none MUST be discharged by a probe that can pass without exercising its subject — see `R-APC-005`'s recorded reasoning.

- **S-APC-069** — Given this milestone's whole test set run repeatedly under `-race`, when the results are compared, then they are identical across runs and the race detector reports nothing.
- **S-APC-070** *(review)* — Given the timing pair of `R-APC-004`, when a reviewer inspects it, then it declares no parallelism, asserts the control's failure shape rather than a duration, and uses a ratio wide enough that a loaded machine does not flip its outcome.
- **S-APC-071** — Given this milestone's whole test set, when its wall-clock runtime is measured, then no single test's duration approaches the configured time-to-headers bound — the bounds are asserted structurally per `R-APC-005`, not waited out.

### NFR-APC-F — Totality, and the nil-client contract

No exported entry point of the adapter MUST panic for any input, including an empty endpoint, an empty credential and a whitespace-only endpoint. Each MUST instead return an AI-04 typed failure per `R-APC-002`.

**An absent HTTP client is expressly excluded from that list and MUST NOT be a fault.** It is the documented selector for the adapter-built path of `R-APC-003`: construction MUST succeed, MUST build the adapter's own bounded client, and MUST NOT return an error. Treating it as a fault would make the adapter unusable without the caller first constructing a client whose bounds the adapter is then forbidden to set — the contradiction `R-APC-003` resolves.

- **S-APC-072** — Given a table of those extreme inputs passed through every exported entry point, when each runs, then none panics and each returns a typed failure naming the offending input's position.
- **S-APC-073** — Given a valid endpoint and credential with **no** HTTP client supplied, when construction runs, then it succeeds, returns a usable adapter, and returns no error.
- **S-APC-074** *(review)* — Given the landed documentation of the construction surface, when a reader looks for what an absent client means, then it is stated as selecting the adapter's own bounded client rather than as an omission or a fault.

### NFR-APC-G — Evidence

Every **runnable** test-list item of AI-25.1 … AI-25.3 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. Scenarios marked `*(review)*` are review obligations with no red phase; each MUST instead be discharged by a recorded reviewer confirmation naming what was read, and MUST NOT be reported as a red→green item. Each slice MUST close on recorded green `make test` (`go test -race -v ./...`) from `backend/agent/` and clean `make lint`. AI-25.2's four bite proofs MUST additionally have their red output recorded in their slice's pull-request description.

- **S-APC-075** *(review)* — Given `tasks.md`, when a reviewer walks this milestone's runnable test-list items, then each carries recorded red output, recorded green output and a refactor note, and each `*(review)*` scenario instead carries a recorded confirmation naming what was read.
- **S-APC-076** *(review)* — Given AI-25.2's slice, when its pull-request description is read, then it carries all four recorded red runs and the final green run.

---

## Acceptance criteria

1. Endpoint, credential and an **optional** HTTP client are accepted, and each supplied value is observed **in use** through a stub transport rather than by a field read; a stored-but-ignored value fails at least one test (`R-APC-001`).
2. A malformed endpoint and an empty credential each fail **at construction** with an AI-04 typed failure carrying the right rule class and position, with zero round trips and **zero new sentinels**; an absent client is **not** among the faults (`R-APC-002`).
3. No default endpoint can reach a real provider; no process-wide, shared or injected client or transport is mutated; and when no client is injected the adapter builds its own with fixed, non-caller-injectable bounds (`R-APC-003`).
4. The paired comparison shows the control client dying mid-body and the adapter-built client completing the same response, with no internally derived deadline (`R-APC-004`).
5. The adapter-built client's connect, handshake, time-to-headers and idle bounds are each shown to be non-zero and in a safe range, structurally rather than by waiting them out (`R-APC-005`).
6. Every base-endpoint shape in the table joins correctly, including the sub-path base without a trailing slash that reference merging would silently truncate (`R-APC-006`).
7. The adapter ships **no** placeholder streaming method, does not falsely satisfy the AI-20 interface, and the AI-20 signature guard passes unmodified (`R-APC-007`).
8. The guard is a call-site scan over the adapter's own non-test sources, denies by default at package granularity, resolves aliased imports, names file, line and package on failure, and states its scope boundary at an injected client (`R-APC-008`, `R-APC-011`).
9. The adapter-built client's proxy resolution is unset, closing the environment-read vector the call-site scan structurally cannot see (`R-APC-009`).
10. The guard is shown to **bite** against all four violation shapes — plain, aliased, process spawn, dot-import — with each red run recorded, then dropped and re-run green (`R-APC-010`).
11. The local-shadow false-positive limitation is recorded in place as an accepted, reversible non-requirement (`R-APC-012`).
12. A request to a local test server arrives at the joined path carrying the injected credential in the dialect's header shape, unaffected by a proxy environment variable, and the node is documented as proving **attachment only**; the credential's type-shape opacity is asserted and is expressly not a wire-level secrecy claim (`R-APC-013`, `R-APC-014`).
13. `go.mod` still declares zero requires, both AI-00 import guards pass with an unmodified allowlist, `src/` still lists exactly two entries, and every edit under `src/ai/` is comment-only (`NFR-APC-A`, `NFR-APC-B`).
14. All three stale comment claims are corrected and `S-AGM-032` still passes verbatim (`NFR-APC-C`).
15. Construction with no HTTP client succeeds rather than failing (`NFR-APC-F`).
16. `make test` green under `-race` and `make lint` clean for every slice, with red/green evidence recorded per **runnable** item and a recorded confirmation per `*(review)*` scenario (`NFR-APC-E`, `NFR-APC-G`).

## Left to design, deliberately

This spec constrains observable behaviour only. Package layout, file decomposition, type and function shapes, the exact scan implementation, the concrete bound values and their safe ranges, and the mechanism by which the joined path is computed are **design's**, and are not constrained here beyond the behaviour each must produce.

Two carryovers are recorded as **not** absorbed here: a mid-stream inactivity cutoff (no standard-library field provides one — the pooled-connection idle bound of `R-APC-005` is a connection-reuse bound, not a read-idle bound; deferred and unassigned) and building a conformance-suite factory for this adapter (a later milestone — AI-25 must only not foreclose it, per `R-APC-007`).
