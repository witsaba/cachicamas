# Spec — `ai-observability-boundary` (AI-37, Add the observability boundary)

> **Milestone**: AI-37 — Add the observability boundary (doc 0002:2188–2235) · **Wave 5 — Harden**
> **Introduced by**: `openspec/changes/archive/2026-08-08-cachicamas-ai-observability/`, landed on branch `feat/ai-37-observability`, final HEAD `66e12147`
> **Status**: **live** — the invariants below hold for the lifetime of the module, not only at the moment AI-37 merged
> **Governing ADR**: [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) — the import table, the twelve-attribute allowlist and the absolute content denylist
> **Format**: RFC 2119 + Given/When/Then, matching this repository's `R-*` / `S-*` grammar
> **Identifier convention**: requirements `R-AOB-0NN`, scenarios `S-AOB-0NN`. Append-only.
> **Sources**: [proposal.md](../../changes/archive/2026-08-08-cachicamas-ai-observability/proposal.md) § 4 D-1 … D-6 · [explore.md](../../changes/archive/2026-08-08-cachicamas-ai-observability/explore.md) § 3 … § 9

---

## Identity

| Field | Value |
| --- | --- |
| **Capability** | `ai-observability-boundary` |
| **Type** | Full spec — nine requirements `R-AOB-001` … `R-AOB-009`, scenarios `S-AOB-001` … `S-AOB-041` |
| **Numbering** | Prefix re-verified at spec time across the whole `openspec/` tree — canonical specs, active changes and archived change deltas alike. `AOB` occurred nowhere before this change's own `proposal.md`. |
| **Charter coverage** | AI-37.1 → `R-AOB-001`, `R-AOB-002` · AI-37.2 → `R-AOB-003` … `R-AOB-006` · AI-37.3 → `R-AOB-007`, `R-AOB-008` · AI-37.4 → `R-AOB-009` |

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. This capability's contract therefore lives here. The archived change folder at [`openspec/changes/archive/2026-08-08-cachicamas-ai-observability/`](../../changes/archive/2026-08-08-cachicamas-ai-observability/) is the historical record of how AI-37 was explored, proposed, designed, applied and verified — including the two recorded dependency-guard bites, the two Judgment Day correction rounds, and the doc-only reconciliation round that closed the completeness gate before archive.

A later milestone that needs to change one of these invariants amends **this file**, in the same pull request, under its own ADR gate.

## Purpose

Define what Layer 1 may observe about itself, and — more load-bearing — what it may never observe. The capability has three halves that fail in different ways: a **dependency** boundary that fails loudly at build time, an **attribute** contract that fails as a breaking change for every telemetry consumer if a key is renamed, and an **absence** claim that fails *silently* if its proof is vacuous. Each half is specified with the falsifier that makes it non-vacuous, not merely with the property it asserts.

**Register.** These requirements state behaviour. No Go identifier appears anywhere in this file. Attribute **keys** are the one deliberate exception: they are external semantic-convention names, not language identifiers, and AI-37.2 test 2 (doc 0002:2221) requires them spelled exactly.

---

## The twelve-key allowlist, quoted from ADR 0005 § D3

`gen_ai.system` · `gen_ai.request.model` · `gen_ai.request.max_tokens` · `gen_ai.response.finish_reasons` · `gen_ai.usage.input_tokens` · `gen_ai.usage.output_tokens` · `gen_ai.usage.cache_read_tokens` · `gen_ai.usage.cache_write_tokens` · `http.response.status_code` · `retry.count` · `stream.event_count` · `error.type`

## The denylist, quoted from ADR 0005 § D3, absolute

Any prompt, completion, reasoning, tool-argument or tool-result text; any HTTP header; any credential; any raw provider response body.

---

## Requirements

### R-AOB-001 — Layer 1 reaches exactly the tracing API it imports, and no telemetry state arrives ambiently

Layer 1 MUST reach the OpenTelemetry ecosystem only through the tracing **API** paths it actually imports, and that set MUST be a **strict subset** of what ADR 0005 § D3's table permits: the tracing interfaces, the API's own no-op provider, the attribute value type and the status-code type. Layer 1 MUST NOT import the ecosystem's root package — the global-getter package — even though § D3's table marks it permitted.

The narrowing is deliberate and MUST be recorded, not inferred: importing the global getter would drag a package whose path is literally the SDK's into the closure of the milestone whose Acceptance clause is "the OTel API and nothing else from that ecosystem" (doc 0002:2196).

A tracer provider MUST reach Layer 1 **only by injection**. Layer 1 MUST NOT read, write or otherwise consult any process-global telemetry registry, getter or setter. This is the tracing instance of the standing rule that the adapter's configuration arrives only by injection (`R-APC-008`); a process-global provider registry, set by whichever caller reached it first, is configuration that did not arrive by injection.

The boundary guard MUST cite ADR 0005 § D3 as the authorising ADR for every telemetry allowlist entry it carries.

#### Scenarios

- **S-AOB-001** — Given the shipped module, when every telemetry import in Layer 1 is enumerated, then the set is exactly the tracing interfaces, the API's no-op provider, the attribute value type and the status-code type, and the ecosystem's root global-getter package is **not** among them.
- **S-AOB-002** — Given the boundary guard's source, when the authorising rationale beside each telemetry allowlist entry is read, then each names ADR 0005 § D3 as the authorising ADR clause.
- **S-AOB-003** — Given the boundary guard's source, when a reader looks for why the global-getter path § D3 permits is not admitted, then a note in place records the declination as a deliberate strict subset and states its reason — that Layer 1 takes a tracer provider only by injection, never by consulting that path — rather than leaving the omission to read as an oversight. (Narrowed to the injection rationale the note currently states; the closure/`auto/sdk` rationale is a recorded follow-up, see "Out of scope" below.)
- **S-AOB-004** — Given the shipped Layer 1 sources, when they are scanned for any read or write of process-global telemetry state, then none exists, and every tracer used by a traced request is traceable to an injected value.

### R-AOB-002 — The telemetry boundary closes on a recorded bite proof, not on a green run

Per doc 0002's guard-leaf grammar, AI-37.1 MUST NOT be marked complete on a green run alone. The guard closes only when it is **shown to fail** against deliberate scratch violations added to real, scanned source, each red run recorded, the violations then dropped and the suite re-run green.

**Two** violation shapes are required, because the forbidden table names two distinct families and an untested branch is not a guard: an SDK import, and an exporter import.

#### Scenarios

- **S-AOB-005** — **(bite 1)** Given a scratch file in Layer 1 importing a package under the ecosystem's SDK path, when the suite runs, then the guard FAILS, its message names the offending import path and the rule that rejected it, and that red output is recorded.
- **S-AOB-006** — **(bite 2)** Given a scratch file in Layer 1 importing a package under the ecosystem's exporters path, when the suite runs, then the guard FAILS, its message names the offending import path and the rule that rejected it, and that red output is recorded.
- **S-AOB-007** — Given both scratch violations dropped, when the suite runs again, then it is green, that output is recorded alongside the two reds, and the merged diff contains no scratch file and no dependency entry that the two violations forced in.

### R-AOB-003 — One span covers one logical request and closes exactly once on every exit

A traced request MUST produce **exactly one** span covering the whole logical request. The span MUST start before the first transport attempt, so that every retry attempt falls inside it rather than beside it, and MUST end **exactly once** on every terminal path without exception: normal completion, every failure exit, and every mid-stream cancellation shape.

A span that ends more than once, or that is left unended on any path, MUST fail the suite. Ending at individual return sites is the failure mode this requirement exists to exclude; the closing obligation MUST be discharged uniformly for all exits.

#### Scenarios

- **S-AOB-008** — Given one traced request that completes normally, when the recording is inspected, then exactly one span was started and exactly one was ended.
- **S-AOB-009** — Given a traced request whose transport attempt is retried more than once before succeeding, when the recording is inspected, then still exactly one span exists, and the span's start is positioned immediately before the retry mechanism's first attempt in source — never beside or after it — so every retry attempt necessarily falls inside its lifetime. (Narrowed: the recording tracer captures no timestamps, so the span's "recorded duration" cannot be inspected; enclosure is established by that fixed start-before-retry source position, not by a measured interval.)
- **S-AOB-010** — Given a table of terminal paths — normal completion, a terminal provider failure, and each mid-stream cancellation shape — when each is exercised under tracing, then in every row the span's end count is exactly one and no span is left unended.
- **S-AOB-011** — Given a deliberately mutated implementation that closes the span at a single return site instead of uniformly, when `S-AOB-010` runs, then at least one row fails — the scenario detects an unclosed path rather than assuming closure.

### R-AOB-004 — A completed traced request carries only allowlisted attributes, spelled exactly, with values equal to the normalized result

The attribute key set recorded on a span MUST be a subset of the twelve keys ADR 0005 § D3 names. No thirteenth key MAY appear, under any name, on any path.

Every key MUST be spelled **exactly** as § D3 spells it — the OpenTelemetry GenAI semantic-convention names — never an ad-hoc equivalent, abbreviation or re-casing. Renaming a telemetry key later is a breaking change for every consumer of it (doc 0002:2221), so each key MUST be a fixed literal spelled in exactly one place and MUST NOT be assembled at a recording site.

Each recorded value MUST equal the corresponding value of the **normalized result** exactly:

| Key | Value contract |
| --- | --- |
| `gen_ai.system` | The **dialect** the adapter speaks, not the vendor behind the endpoint — the same adapter serves several gateways, so a vendor value would be a lie. |
| `gen_ai.request.model` | The request's model identifier, verbatim. |
| `gen_ai.request.max_tokens` | The request's maximum-output-token bound. **Presence-typed**: recorded only when the request carries one. |
| `gen_ai.response.finish_reasons` | A one-element array carrying the completion's finish reason in its normalized spelling. Layer 1 tracks exactly one reason; the array shape follows the convention. |
| `gen_ai.usage.input_tokens`, `…output_tokens`, `…cache_read_tokens`, `…cache_write_tokens` | The completion event's corresponding usage counts. **Presence-typed** individually. |
| `http.response.status_code` | The **exact** response status code, on the success path. See `R-AOB-006` for the terminal-failure path. |
| `retry.count` | The number of **retries**, so an unretried request reports `0`. A value MUST be available on **every** exit of the retry mechanism, not only on the exit that exhausts the budget. |
| `stream.event_count` | The number of events emitted on the stream carrier for this request. |
| `error.type` | The failure classification, drawn from the landed closed nine-member vocabulary. Recorded on failure paths. |

A presence-typed attribute whose underlying value is absent MUST be **omitted**, never recorded as a zero standing in for absence. The reasoning-token usage breakdown MUST NOT be recorded under any key: it is a breakdown of the output count, not a term beside it, and § D3 does not name it.

#### Scenarios

- **S-AOB-012** — Given a traced request driven to completion against the in-memory recording tracer, when the recorded attribute key set is compared against the twelve allowlisted keys, then it is a subset of them and contains no other key.
- **S-AOB-013** — Given that same recording, when each present attribute's value is compared against the corresponding field of the normalized result, then every comparison is exact, per the table above.
- **S-AOB-014** — Given the recorded keys, when each is compared against its expected literal spelling, then all match character for character; and given a deliberately mutated mapping that records one key under an ad-hoc equivalent spelling, when this scenario runs, then it fails and names the offending key.
- **S-AOB-015** — Given a table of retry outcomes — an unretried success, a request retried twice then succeeding, a non-retryable terminal failure, a budget-exhausted failure, and a cancelled request — when each is traced, then `retry.count` is present on every row and reports the number of retries, `0` on the unretried row.
- **S-AOB-016** — Given a request carrying no maximum-output-token bound and a completion carrying no cache-read count, when the span is inspected, then `gen_ai.request.max_tokens` and `gen_ai.usage.cache_read_tokens` are **absent**, not recorded as zero.
- **S-AOB-017** — Given a completion whose usage carries a reasoning-token breakdown, when the span's attribute keys are enumerated, then no key carries that breakdown.
- **S-AOB-018** — Given two adapters constructed against two different vendor endpoints, when each is traced, then both record the **same** `gen_ai.system` value — the dialect — and neither records a vendor-derived value.

### R-AOB-005 — Streaming spans close at the terminal event, and usage attributes equal the completion event's usage

A streaming span MUST NOT end before the stream's terminal event has been produced. Usage attributes MUST equal the usage carried by the **completion event** that the consumer receives — not a value read from any earlier or separate source — so that what a telemetry consumer sees and what the stream consumer sees can never disagree. `stream.event_count` MUST equal the number of events actually emitted on the carrier for that request.

#### Scenarios

- **S-AOB-019** — Given a scripted stream traced to completion, when the span's end is ordered against the terminal event's emission, then the span ended at or after the terminal event and never before it.
- **S-AOB-020** — Given that same run, when the span's four usage attributes are compared against the usage carried by the drained completion event, then each present count is equal and no count is present on the span that is absent from the event.
- **S-AOB-021** — Given that same run, when `stream.event_count` is compared against the number of events drained from the carrier, then they are equal.

### R-AOB-006 — Every allowlisted key deliberately left unrecorded carries a stated justification and a recorded follow-up

Where an allowlisted key has no exact source of truth, the milestone MUST narrow its own claim rather than record an approximate value under a precise key. This capability's terminal-failure exits do not all share one shape, and the requirement below is deliberately split rather than stated as one blanket rule for every terminal-failure path:

- On a terminal-failure path where the retry mechanism itself fails before any response value survives to the recording site — a retry-budget-exhausted or a non-retryable terminal failure — `http.response.status_code` MUST be **omitted** on that path.
- On a terminal-failure path where a response WAS obtained before the failure was recognized — the non-streaming-content-type refusal, whose exact code is in hand from that same response, **and** every post-handover (mid-stream) terminal failure, which this capability only ever begins producing after that one response has already been obtained — `http.response.status_code` MUST be **recorded**, exactly as the success path records it. Omitting a value that is genuinely in hand would itself be the falsehood this requirement exists to prevent: narrowing a claim past what is actually available is not the same discipline as narrowing it to what is available.

The omission MUST be an explicit clause of this requirement and MUST be recorded at the omission site in source. It MUST NOT be left implicit, and the class MUST NOT be recorded under the exact-code key: that key means the code, and writing a class where a consumer expects a code is a falsehood in a contract whose whole point is that a rename breaks consumers.

A follow-up MUST be recorded naming the capability that owns an exact terminal status code, so a later milestone that needs it inherits a decision rather than a silence.

#### Scenarios

- **S-AOB-022** — Given a traced request whose retry mechanism itself fails with no response value surviving to the recording site — a retry-budget-exhausted or a non-retryable terminal failure — when the span's attribute keys are enumerated, then `http.response.status_code` is absent, and a note at the omission site in source states why.
- **S-AOB-023** — Given that same span, when every recorded value is inspected, then no status *class* value appears under `http.response.status_code` or under any other allowlisted key.
- **S-AOB-024** *(review)* — Given this change's landed artefacts, when a reader looks for the owner of an exact terminal status code, then a follow-up is recorded naming the capability that owns it.
- **S-AOB-025** — Given that same terminal-failure span, when its attributes are inspected, then `error.type` is present and its value is a member of the landed closed nine-member failure vocabulary.
- **S-AOB-040** — Given a traced request that ends in a non-streaming-content-type refusal — a terminal failure for which a response WAS obtained before the refusal was recognized — when the span's attribute keys are enumerated, then `http.response.status_code` is present and equals that response's own exact status code, recorded exactly as the success path records it.
- **S-AOB-041** — Given a traced request that ends in a post-handover (mid-stream) terminal failure — a failure recognized after the stream's one response was already obtained, at any of this capability's own mid-stream terminal paths — when the span's attribute keys are enumerated, then `http.response.status_code` is present and equals that response's own exact status code, recorded exactly as the success path records it, alongside `error.type`.

### R-AOB-007 — The content denylist is absolute and is proven by absence over a run that used everything it denies

No span name, no span status description, no attribute key, no attribute value, no span event name, no span event attribute value and no recorded error string MAY carry any prompt, completion, reasoning, tool-argument or tool-result text, any HTTP header name or value, any credential, or any raw provider response body.

The denylist is **absolute at Layer 1**. No content-capture opt-in MAY exist: no configuration input, no flag, no build tag and no environment-derived switch MAY enable recording any denied category.

Absence MUST be asserted over a run that **used all of them** — a run whose request carried prompt text, whose stream carried reasoning and tool-call frames, whose conversation carried a tool result, whose client was constructed with a credential, and whose response carried a custom header and a distinctive raw-body marker.

The corpus scanned MUST be built by walking **every** field the tracing API can set, and attribute values MUST be rendered through the API's own value accessor, never through a language-level structure dump: a marker planted in a pointer-shaped field renders as a bare address under a structure dump and proves nothing (the AI-41 lesson, doc 0002:22).

Failure output MUST name the vector. It MUST NOT reprint the bytes it matched. (Naming the span and the attribute key alongside the vector is not yet true of the shipped scan; see `S-AOB-029`'s recorded follow-up.)

#### Scenarios

- **S-AOB-026** — Given a traced run that carried prompt text, reasoning text, tool-call argument text, tool-result text, a credential, a custom response header and a distinctive raw-body marker — each a distinct runtime-assembled marker — when the corpus built from every span name, status description, attribute key and value, event name and event attribute value, and recorded error is scanned, then none of the seven markers is found.
- **S-AOB-027** — Given that the tracing API's own attribute-value type is a closed set of typed constructors (string, integer, floating-point, boolean and their slice forms) with no pointer-shaped variant, when the corpus is built, then every recorded value is rendered through that type's own current string accessor — never a language-level structure dump — so the AI-41 failure mode this scenario guards against (a canary in a pointer-shaped field rendering as a bare hex address) is excluded structurally rather than demonstrated against a live pointer-shaped counter-example, because the recorded surface has none to plant one in. (Accessor substitution recorded: the API's `Value.Emit()` accessor this rendering rule was originally written against is deprecated at v1.44.0; `Value.String()` is the current accessor and is what the shipped code calls — both render through the API's own accessor rather than a struct dump, so the substitution changes no proven property.)
- **S-AOB-028** — Satisfied by construction. Given the shipped module, when every configuration input, flag, build tag and environment-derived switch reachable from Layer 1 is enumerated, then the enumeration is the empty set — Layer 1's own construction surface (`R-APC-001`, `R-APC-003`) and its deny-by-default import guard (`R-AGM-005`, `R-AGM-008`) between them admit no such switch — so there is no content-capture opt-in for any test to exercise on or off; the property holds because the surface to enable it was never built, not because a built switch was driven both ways and found inert.
- **S-AOB-029** — Given a scan that finds a planted marker, when its failure message is read, then it names the vector, and reproduces none of the matched bytes. (Narrowed: the shipped scan's result carries a vector only, no span or attribute-key location, because the corpus it scans is a flattened byte string with no positional metadata. Giving the scan a location-carrying result is a recorded follow-up, see "Out of scope" below.)

### R-AOB-008 — *(non-negotiable)* The absence proof ships four independent non-vacuity guards

An absence assertion is the single easiest test in this repository to write vacuously, and AI-36 handed this milestone the residual by name (doc 0002:24). `R-AOB-007`'s proof MUST therefore ship **all four** guards below. Three of the four are not interchangeable with the positive control: a self-test proves the *needles* bite, and proves nothing at all about whether the *corpus* was ever populated.

1. **Positive control** — the shared sweep's self-test MUST run before every clean scan and MUST fail the suite if any needle fails to bite.
2. **Corpus-non-empty guard** — the corpus length MUST be asserted greater than zero, the recorded span count at least one, and the recorded attribute count at least the expected allowlist cardinality. The scanner returns *not found* for an empty corpus by construction, so without this guard the whole absence claim passes on a run that recorded nothing.
3. **Event-coverage guard** — the drained recording MUST be asserted to have contained every event kind this adapter can structurally produce during normal operation — text, tool-call, completion and error events — so that the absence was asserted over a run that used all of them rather than over a run that used none. This adapter's own wire decoding drops any reasoning-content field before a Go value exists to carry it, so it can never construct a reasoning *event* on any real transcript; asserting event-coverage for a kind that cannot exist is not a claim this guard can make truthfully. A reasoning-content canary is still planted and still scanned by `R-AOB-007`'s absence proof — its absence from the corpus is guaranteed by the wire-decode drop, not demonstrated by this coverage guard, and it is retained for documentation value rather than as additional coverage evidence.
4. **Recorded RED** — with a deliberately leaking attribute mapping substituted at build time **without modifying the working tree**, `R-AOB-007`'s test MUST fail and name the leaking vector. The red output MUST be recorded, the substitution dropped, and the suite re-run green.

The recorder MUST additionally assert that its own captured-field count matches the tracing API surface it implements, so that a value set through a path the corpus builder does not walk fails the suite rather than escaping it.

#### Scenarios

- **S-AOB-030** — Given the denylist test, when it runs, then the sweep's self-test runs before the clean scan; and given a needle deliberately made not to bite, when the self-test runs, then the suite fails.
- **S-AOB-031** — Given a traced run that recorded nothing, when the denylist test runs against it, then the **corpus-non-empty guard** fails; and given the real traced run, when the guard runs, then corpus length is greater than zero, span count is at least one, and attribute count is at least the expected allowlist cardinality.
- **S-AOB-032** — Given the drained recording of the denylist run, when its event kinds are enumerated, then text, tool-call, completion and error events — the four kinds this adapter can structurally produce — are each present; and given a run that omitted one of those four kinds, when this guard runs, then it fails and names the missing kind. This scenario does not assert a reasoning *event*, because no run of this adapter can ever populate one; `R-AOB-007`'s own absence scan covers the reasoning-content canary independently.
- **S-AOB-033** — Given an attribute mapping deliberately altered to copy message text into a span attribute, substituted at build time without modifying the working tree, when `R-AOB-007`'s test runs, then it FAILS and names the leaking vector; the red output is recorded, the substitution is dropped, and the suite is re-run green with that output recorded alongside.
- **S-AOB-034** — Given the recording tracer, when the count of fields it captures is compared against the count of fields the tracing API allows a caller to set on a span, then they are equal — a newly settable field cannot be recorded and then skipped by the corpus builder.
- **S-AOB-035** — Given the denylist test's own source, when the sweep scans this repository's sources, then it does not flag that source — every needle is assembled at runtime and no denied marker appears as a contiguous literal.

### R-AOB-009 — With no tracer configured, behaviour is identical, nothing panics, and no nil check exists anywhere

WHEN no tracer provider is injected, streaming MUST behave **identically**: the drained event sequence of an untraced run MUST equal, element for element and value for value, the drained event sequence of the same scripted run with a tracer configured. Nothing MUST panic on any path.

The no-op default MUST be **structural** — established once at construction by substituting the tracing API's own no-op provider — so that every recording site is unconditional. Consequently **no adapter-side nil check on a tracer, a span or a provider MAY exist anywhere in Layer 1**. That clause is a requirement in its own right and MUST be separately assertable: it is what distinguishes "the API's no-op default suffices" (doc 0002:2234) from "we guarded every call site".

#### Scenarios

- **S-AOB-036** — Given one scripted stream run twice — once with no tracer configured, once with the recording tracer — when the two drained event sequences are compared, then they are equal element for element, in the same order, with the same values.
- **S-AOB-037** — Given a table of terminal paths run with **no** tracer configured — normal completion, terminal failure, and each mid-stream cancellation shape — when each is exercised, then none panics and each produces the same outcome it produces without this change.
- **S-AOB-038** — Given the shipped source files of the package implementing this capability — the scope the guard's own scan walks — when they are scanned for a nil check guarding any tracing value, then none exists in that scope, and the scan's own floor guards against a silently narrowed or empty directory listing. (Narrowed to the guard's actual scan scope; the broader claim — no such nil check exists anywhere in Layer 1 — was independently verified true this round by direct enumeration outside the guard, but widening the guard itself is a recorded follow-up, see "Out of scope" below.)
- **S-AOB-039** — Given a deliberately mutated implementation that guards its recording sites with a nil check, when `S-AOB-038` runs, then it fails — the structural claim is asserted, not assumed.

---

## Non-functional requirements

### NFR-AOB-001 — Telemetry costs nothing when unwired

Recording MUST impose no observable behavioural cost when no provider is injected: no additional allocation-visible latency on the stream carrier, no additional goroutine, and no change in event ordering or timing that a consumer can observe. This is the property that makes the tracing API safe below a composition root at all (ADR 0005 § D3), and `R-AOB-009`'s equality proof is its behavioural evidence.

### NFR-AOB-002 — Key literacy

Each attribute key MUST be spelled in exactly one place and referenced from every recording site, so a rename is a single, reviewable edit rather than a search. A recording site MUST NOT assemble a key from parts.

---

## Out of scope, with the owner of each

| Item | Owner |
| --- | --- |
| Dashboards, exporters and application tracing setup | The composition root — charter Out-of-scope verbatim (doc 0002:2198), per § D3 |
| Metrics of any kind, and any metric instrument | Non-goal — § D3 permits the path, but the charter's Deliverable is "minimal spans and attributes" and no charter test names a metric |
| Any OpenTelemetry module beyond § D3's table — baggage, propagation, contrib, SDK | Each requires **its own ADR** (§ D3's closing blockquote) |
| Trace-context propagation and header injection | Needs a module § D3's measured subset excludes; no charter test names it |
| An exact status code on the terminal-failure path | `ai-provider-errors`, per `R-AOB-006`'s recorded follow-up |
| Any conformance-suite change or new capability member | Not this milestone — the charter is silent and the capability vocabulary is closed at nine |
| Tracing in Layer 2, Layer 3 or the composition root | Docs 0003/0004 — those directories do not exist |
| The construction surface's shape, defaulting and injection proof | `ai-provider-client` — `R-APC-001`, `R-APC-003`, `R-APC-016` |
| The dependency require set, the guard allowlist and the package-closure pin | `agent-module-scaffold` — `R-AGM-001`, `R-AGM-005`, `R-AGM-008` |
| Every name, receiver, package and file layout implied above | **Design phase** — this spec pins behaviour only |
| Widening the closure/`auto/sdk` declination reason into the boundary guard's own comment (`S-AOB-003`'s narrowed clause) | A later round that edits `import_boundary_test.go` — deferred at archive |
| Widening the nil-check-absence guard's scan from this capability's implementing package to all of Layer 1 (`S-AOB-038`'s narrowed clause) | A later round that edits `ai37_noop_equivalence_test.go` — deferred at archive |
| Giving the denylist scan a location-carrying result so a failure can name the span and the attribute key, not only the vector (`R-AOB-007`'s and `S-AOB-029`'s narrowed clauses) | A later round that edits `ai37_denylist_test.go` and the shared sweep — deferred at archive |

---

## Acceptance criteria

The capability holds when every scenario `S-AOB-001` … `S-AOB-041` has recorded evidence, the two bites of `R-AOB-002` and the overlay RED of `R-AOB-008` item 4 are recorded red then green, and `make test` under race detection, `make lint` and `make build` are green in the module with a clean working tree.

Verified at archive time (2026-08-08): `verify-report-final.md` recorded 0 CRITICAL, 0 blockers, 72/80 scenarios COMPLIANT and 8 PARTIAL (each PARTIAL narrowed to what its shipped instrument actually proves, per the scenario texts above); Judgment Day reached terminal state **APPROVED** after two correction rounds. The eight narrowed scenarios and their three recorded follow-ups (`S-AOB-003`, `S-AOB-038`, `S-AOB-029`/`R-AOB-007`) are the honest, final state of this capability's proof — not a gap awaiting a third round.
