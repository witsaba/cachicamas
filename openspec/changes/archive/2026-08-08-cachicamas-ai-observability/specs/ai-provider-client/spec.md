# Delta for `ai-provider-client` — AI-37 the tracer-provider injectable

> **Change**: `cachicamas-ai-observability` · **Milestone**: AI-37 (doc 0002:2188–2235) · **Nodes**: AI-37.1 (injection), AI-37.4 (no-op default) · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-provider-client/spec.md`](../../../../specs/ai-provider-client/spec.md): two `MODIFIED` requirements and one `ADDED` requirement
> **Format**: RFC 2119 + Given/When/Then, matching the canonical spec's `R-APC-*` / `S-APC-*` grammar
> **Sources**: [proposal.md](../../proposal.md) § 4 D-1 · § 7 risk 7.4

---

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-provider-client` |
| **Type** | Delta — `MODIFIED` `R-APC-001` and `R-APC-003`; `ADDED` `R-APC-016` |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical, active and archived alike: maxima `R-APC-015` / `S-APC-080`. `R-APC-016` and `S-APC-081` onward are free. |
| **New scenarios** | `S-APC-081` (under `R-APC-001`), `S-APC-082` … `S-APC-085` (under `R-APC-016`) |
| **Amended scenario** | `S-APC-015` — the construction-surface enumeration only |

### The spec cost, paid openly

Injecting a tracer provider contradicts two landed statements, and both are amended here rather than quietly outgrown.

1. **`R-APC-001`** requires every injected value's use to be observable **through a stub transport at the round-trip boundary**, and forbids proving it by reading a stored field. A tracer provider is never observable in what the adapter sends. Rather than weaken the rule for everything, the requirement gains a narrow, explicitly bounded second proof shape for injectables that carry no traffic — and that shape is still behavioural, still a stub, still not a field read. It is the direct analogue of the landed `S-APC-004`.
2. **`S-APC-015`** enumerates the construction surface exhaustively. A tracer provider is a fourth injectable, so the enumeration changes. Its qualifier — that **no** timeout, deadline or bound value appears among them — survives intact and is restated verbatim, because a tracer provider is none of those. The rationale under `S-APC-015` is therefore unchanged; only its list grew.

Both blocks below restate their **entire** requirement, including every unchanged scenario, so that `S-APC-001` … `S-APC-016` survive the archive step exactly as landed.

---

## MODIFIED Requirements

### R-APC-001 — Every injected value is used, and "accepted then ignored" is a detectable failure

Construction MUST accept the endpoint, the credential, (optionally) the HTTP client and (optionally) a tracer provider from its caller. Each value the caller does supply MUST be demonstrably **used** by the constructed value.

For every injectable whose effect **reaches the wire**, use MUST be observable through a stub transport at the round-trip boundary — that is, by what the adapter would send — and MUST NOT be asserted by reading a stored field, because a stored-but-unused value passes a field assertion and fails in production.

For an injectable whose effect **does not reach the wire**, that proof shape is unavailable, and substituting a field read would reintroduce exactly the failure this requirement exists to detect. Such an injectable MUST instead be proven used by **discriminating observation at its own boundary**: two adapters constructed with two different recording substitutes, each driven once, each substitute observing exactly its own adapter's effect and neither observing the other's. This is the same discrimination `S-APC-004` performs at the transport boundary, moved to the boundary where the value's effect is actually visible. A field read remains forbidden on both shapes. (Previously: the requirement admitted only the round-trip-boundary proof shape, because every injectable then in the construction surface reached the wire.)

#### Scenarios

- **S-APC-001** — Given construction with an endpoint, a credential and an HTTP client whose round-trip boundary is stubbed, when the adapter is driven to produce one outbound request, then the stub observes it — proving the injected client is the client that carries traffic.
- **S-APC-002** — Given that same observation, when the observed request's scheme, host and path prefix are compared against the injected endpoint, then they derive from it and from no other source.
- **S-APC-003** — Given that same observation, when the observed request's headers are inspected, then the injected credential is present on it.
- **S-APC-004** — Given two adapters constructed with two **different** stubbed clients, when each is driven once, then each stub observes exactly its own adapter's request and neither observes the other's — proving no shared or cached client is substituted.
- **S-APC-005** — Given a deliberately mutated implementation that stores an injected value without using it, when this requirement's tests run, then at least one fails — the assertions detect ignoring, not merely storing.
- **S-APC-081** — Given two adapters constructed with two **different** recording tracer providers, when each is driven once, then each provider records exactly its own adapter's span and neither records the other's; and given a deliberately mutated implementation that stores the injected provider but derives its tracer from another source, when this scenario runs, then it fails — the injected provider's use is proven by discrimination at its own boundary, never by reading a stored field.

### R-APC-003 — Defaults are safe and fixed, and no shared, global or injected client is mutated

There MUST be no default endpoint that could cause an unconfigured or partially configured adapter to reach a real provider — in particular, from a test.

**The HTTP client is optional, and which party owns the bounds follows from that.** The obligation is scoped in two halves, and conflating them is what makes the naive reading incoherent — an adapter cannot both *fix* bounds on a client and *not mutate* that client:

1. **WHEN no client is injected**, the adapter MUST construct its own. That **adapter-built** client MUST carry fixed connect-phase and idle bounds with safe values, and those bounds MUST NOT be caller-injectable: doc 0002's AI-25.1 item 3 requires that the defaults be *safe*, not that they be *overridable*, and an injectable bound permanently widens a construction surface every later milestone must live with. Doc 0002 item 4's phrase "**the constructed client** carries no whole-request timeout" presupposes that the adapter constructs one on this path.
2. **WHEN a client is injected**, it MUST be used **verbatim**. Its bounds are its **injector's** — set at the composition root, on the caller's own instruction — and the adapter MUST NOT require them, MUST NOT override them, and MUST NOT mutate the client or its transport in any way another holder of that value could observe.

Construction MUST NOT mutate any process-wide or package-level HTTP client or transport on **either** path.

The construction surface admits a **fourth** injectable, a tracer provider, which is likewise optional and likewise used verbatim; its own defaulting rule is `R-APC-016`. It carries no timeout, no deadline and no bound, so the qualifier restated in `S-APC-015` below is unaffected by its addition. (Previously: the construction surface was enumerated as exactly three inputs — the endpoint, the credential and an optional HTTP client.)

#### Scenarios

- **S-APC-011** — Given construction attempted with no endpoint supplied, when it runs, then it fails per `R-APC-002` rather than substituting a vendor endpoint, and no outbound request is possible.
- **S-APC-012** — Given the observable configuration of the process-wide default HTTP client and default transport captured before construction, when construction runs on **both** the injected and the adapter-built path and the same observations are repeated, then every observed value is unchanged and the identities are the same values as before.
- **S-APC-013** — Given one HTTP client value used to construct two adapters, when the second construction completes, then the first adapter's observable outbound behaviour is unchanged and every observable field of the shared client is what it was before either construction — an injected client is used verbatim, never reconfigured.
- **S-APC-014** — Given a deliberately mutated implementation that assigns a bound onto the process-wide default transport, when `S-APC-012` runs, then it fails — the scenario detects global mutation rather than assuming its absence.
- **S-APC-015** — Given the landed construction surface, when a caller enumerates what it may supply, then it is the endpoint, the credential, an **optional** HTTP client and an **optional** tracer provider, and **no** timeout, deadline or bound value appears among them.
- **S-APC-016** — Given construction with no client supplied, when the resulting client is compared against the process-wide default client and the process-wide default transport, then it is neither of them — the adapter built a fresh one rather than adopting a shared value it could not bound without mutating.

---

## ADDED Requirements

### R-APC-016 — An absent tracer provider defaults to the tracing API's own no-op, never to a process-global one

WHEN no tracer provider is injected, construction MUST substitute the tracing API's **own no-op provider**, established once at construction, and MUST NOT consult any process-global telemetry registry, getter or setter for a substitute. A process-global registry is configuration set by whichever caller reached it first; adopting it would be configuration that did not arrive by injection, which `R-APC-008` forbids in terms and which its call-site scan cannot detect.

The substitution MUST be **structural**: because the API's no-op provider is a real, non-nil value whose spans are genuine no-ops, every recording site downstream is unconditional and **no nil check on a tracing value may exist**. That structural clause is separately assertable and is the difference between satisfying doc 0002:2234 and merely appearing to.

WHEN a tracer provider **is** injected, it MUST be used **verbatim**: never wrapped in a way an observer could detect, never mutated, and never replaced by a global value.

Any concrete adapter this module ships that composes `openaicompat.New` internally — the wrapper pattern this module uses — MUST expose the identical optional tracer-provider door on its own construction surface and thread it to `openaicompat.New` verbatim. A composing wrapper's construction surface MUST NOT be narrower than the door it wraps: closing that door at the wrapper boundary would make this requirement's own injection guarantee unreachable for that adapter's own callers, leaving the only shipped concrete provider permanently untraceable.

#### Scenarios

- **S-APC-082** — Given construction with no tracer provider supplied, when the adapter is driven through a complete request, then no span reaches any provider registered in any process-global telemetry state, and the request completes normally — the tracing API's own no-op provider was substituted at construction.
- **S-APC-083** — Satisfied by construction. `R-AOB-001`'s import boundary and `R-AGM-008`'s package-closure pin together prove that no path to the ecosystem's process-global telemetry registry exists anywhere in this module — the root global-getter package is absent from both the require set and the package closure. This scenario's own Given clause — a recording provider installed into process-global telemetry state, observed from inside this module — therefore cannot be constructed: there is no import through which a test could install one to observe from. The property (the adapter never consults process-global state) follows by entailment from those two already-proven guards holding, not from an executable pair that installs a provider and drives the adapter to confirm it goes unconsulted.
- **S-APC-084** — Given construction with a recording tracer provider injected, when the adapter is driven, then that exact provider records the request's span, its identity is unchanged, and no observable property of it differs from before construction — it was used verbatim.
- **S-APC-085** — Given the shipped OpenRouter wrapper — the only concrete adapter this module ships that composes `openaicompat.New` — when a caller constructs it with an injected tracer provider on the wrapper's own construction surface, then that provider records the resulting span exactly as it would through `openaicompat.New` directly, proving the wrapper threads the value verbatim rather than closing the door this requirement otherwise guarantees.

---

## Pins / regressions

| Behaviour leaf | Contract pin | Regression assertion |
| --- | --- | --- |
| A non-transport injectable is proven used | Landed `S-APC-004` discrimination shape | Two adapters, two recording providers, neither observes the other's |
| Field reads remain forbidden | Landed `R-APC-001` rule | Control with a stored-but-unused provider must fail |
| Construction surface enumeration | Landed `S-APC-015` | Four injectables; the no-bound qualifier restated verbatim |
| No ambient telemetry authority | Landed `R-APC-008` "only by injection" | Globally installed provider records nothing |
| No-op default is structural | doc 0002:2234 | No nil check on any tracing value exists |
| Every landed scenario still green | `S-APC-001` … `S-APC-016` | Re-run under the amended requirements as an explicit success criterion |

## Out of scope

| Item | Owner |
| --- | --- |
| Which attributes a span carries and which it must never carry | `ai-observability-boundary` — `R-AOB-004`, `R-AOB-007` |
| The behavioural equivalence of a traced and an untraced run | `ai-observability-boundary` — `R-AOB-009` |
| The dependency require set and the guard allowlist | `agent-module-scaffold` — `R-AGM-001`, `R-AGM-005`, `R-AGM-008` |
| The field name, its position on the construction surface, and every other identifier | **Design phase** — this spec pins behaviour only |
| Any bound, deadline or timeout becoming injectable | Non-goal — `S-APC-015`'s qualifier is unchanged and still true |
