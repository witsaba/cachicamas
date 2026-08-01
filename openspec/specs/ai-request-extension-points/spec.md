# Spec — request extension points

> **Milestone**: AI-12 — Add per-request options, the escape hatch, and rebuild · **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Introduced by**: `openspec/changes/archive/2026-08-01-cachicamas-ai-request-extension-points/`, in PR #101 (open for review at archive time, base `main` @ `efdedc4`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Closes**: **G9** — per-request options and the provider escape hatch. It is also the mechanism doc 0002 records as meeting **G11**, the pre-request hook (AI-12.1's rebuild). Wire rendering is AI-26.7 (Wave 2)
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-REX-0NN` · **Scenario IDs**: `S-REX-0NN`
> **Binding vocabulary**: [Layer 1 contract vocabulary](../ai-contract-vocabulary/spec.md) — `V-REQ-19` … `V-REQ-29`, `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`; every Layer 1 noun below is one of its rows, cited by identifier
> **Binding predecessor**: [the normalized request core](../ai-model-request/spec.md) — `R-AMR-001` … `R-AMR-017`; this file adds a capability and restates none of them
> **Binding predecessor**: [cache-boundary markers](../ai-cache-breakpoints/spec.md) — `R-REX-002`'s region set includes AI-11's marker region and must keep it reachable
> **Binding predecessor**: [reasoning content and its round-trip token](../ai-reasoning-content/spec.md) — `R-REX-003` extends AI-07.3's byte-exactness across the rebuild path
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-12--add-per-request-options-the-escape-hatch-and-rebuild) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)

---

## Purpose

Constrain the runtime behavior of three register terms on the normalized request: `V-REQ-29` **request rebuild**, `V-REQ-27` **per-request option override**, and `V-REQ-28` **provider escape hatch** — plus the read-back determinism those three surfaces must jointly hold.

This capability **adds** to the request. It restates no `R-AMR-*` requirement; where it constrains a surface AI-10 owns, it cites AI-10's requirement and states the additional obligation.

Requirement count: **10**.

> **Re-verified 2026-08-01** against finished Wave 1 head `1c4171e`. No requirement was removed or weakened. Three edits, all additive: `R-REX-002`'s region set is now enumerated one region per entry rather than collapsing four generation options into one; `R-REX-007` states that participation in the documented equality is necessary and not sufficient; and two scenarios are **appended** — `S-REX-053` and `S-REX-054`. The archived `design.md` § 13 carries the resolved register and `explore.md` § 6 the evidence.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The rebuild path, the per-request override rule, the escape hatch and their read-back determinism therefore live here, in their own text, and not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-01-cachicamas-ai-request-extension-points/`](../../changes/archive/2026-08-01-cachicamas-ai-request-extension-points/) is the historical record of how AI-12 was explored, proposed, designed, applied and verified.

`R-REX-002`'s totality obligation binds every later milestone that adds a region to the request: a region reachable only at construction must not exist, and the region set is enumerated so an addition without a rebuild path fails rather than passes unnoticed. `R-REX-007`'s closing paragraph records why equality participation alone is not enough — the round-trip walk `R-AMR-015` pins does not route through the documented equality.

> **Known limitation, owed to Wave 2.** The region-exhaustiveness guard catches a deleted table row but not a field added to the request draft — the failure that recurred at AI-10.6, AI-11.1 and AI-12.3. Doc 0002's traceability spine carries this as a Wave 2 obligation.

## Requirement ownership by leaf

| Leaf | Requirements |
| --- | --- |
| AI-12.1 copy-on-write rebuild | `R-REX-001`, `R-REX-002`, `R-REX-003` |
| AI-12.2 per-request options | `R-REX-004`, `R-REX-005` |
| AI-12.3 typed-but-opaque pass-through | `R-REX-006`, `R-REX-007`, `R-REX-008`, `R-REX-010` |
| AI-12.4 read-back determinism | `R-REX-009` |

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A request rebuild** — `V-REQ-29`. Deriving a modified request from an existing one such that the original is observably unmodified and the derived request validates independently.
- **A per-request option override** — `V-REQ-27`. A generation option supplied or replaced for one call without rebuilding the caller's own defaults.
- **A provider escape hatch** — `V-REQ-28`. A typed-but-opaque, namespaced pass-through carrying a provider-specific value the neutral vocabulary deliberately does not model.
- **A region** — AI-10's term for one addressable part of a request. This spec's region list is AI-10's six — model identity, system instruction, messages, tool set, tool choice, generation options — **plus** the cache-boundary markers `V-REQ-23` attaches, and **plus** the escape-hatch region this capability adds.
- **Observably unmodified** — every value the request's exported accessors return is equal to what it returned before, under AI-10's documented equality (`R-AMR-016`).
- **A caller-contract failure**, **a validation sentinel**, **a position**, **the rule order** — `V-FAIL-01` … `V-FAIL-04`, as landed by AI-04.

---

## R-REX-001 — A rebuild derives a new request and leaves the original observably unmodified *(AI-12.1)*

Layer 1 MUST provide a way to derive a request from a constructed request by supplying zero or more option values, without naming any region the caller does not wish to change.

WHEN a rebuild succeeds, THEN the derived request MUST carry the supplied changes, MUST carry every unsupplied region unchanged from the source, and the **source request MUST be observably unmodified** under `R-AMR-016`'s equality, compared both before and after the derivation.

WHEN a rebuild fails a rule, THEN it MUST return the zero request together with the failure, and the source request MUST still be observably unmodified. A caller that ignored the error MUST NOT be able to mistake the result for a derived request.

The derived request MUST be **independently valid**: it satisfies every rule a request constructed from the same regions would satisfy, and no rule is skipped because the source already passed it.

Deriving with **no** options MUST succeed and MUST produce a request equal to the source under `R-AMR-016`'s equality.

### Scenarios

- **S-REX-001** — Given a constructed request, when a caller derives a request that changes one generation option, then the derived request reports the new value and the source request reports its original value.
- **S-REX-002** — Given a constructed request captured region-by-region before the derivation, when a derivation has completed, then every region of the source compares equal to its captured value.
- **S-REX-003** — Given a constructed request, when a caller derives a request supplying no options, then the derivation succeeds and the result is equal to the source.
- **S-REX-004** — Given a constructed request, when a caller derives a request supplying an option value that violates a bound, then the derivation fails, the returned request is the zero request, and the source request is unchanged.
- **S-REX-005** — Given a source request whose messages region a rebuild replaces, when the caller mutates the message slice it passed to the rebuild, then the derived request is unchanged, because the rebuild copies on the way in exactly as construction does.
- **S-REX-006** — Given a derived request, when the caller mutates any slice a reader returned from it, then the derived request is unchanged on re-read.

---

## R-REX-002 — The rebuild path is total over every region *(AI-12.1)*

Every region a request can carry MUST be reachable by the rebuild path. The region set, enumerated one region per entry against the landed request surface, is:

1. **model identity** — `Model()`
2. **messages** — `Messages()`
3. **system instruction segments** — `SystemInstruction()`
4. **tool set** — `Tools()`
5. **tool choice** — `ToolChoice()`
6. **maximum output tokens** — `MaxOutputTokens()`
7. **temperature** — `Temperature()`
8. **top-p** — `TopP()`
9. **stop sequences** — `StopSequences()`
10. **cache-boundary markers** (`V-REQ-23`) — AI-11's surface
11. **provider extensions** — this capability's

The four generation options are enumerated **individually** and MUST NOT be collapsed into a single "generation options" entry. A totality check that treats them as one entry is a check a fifth generation option can be added past without failing, which defeats the purpose of the requirement.

A region reachable only at construction MUST NOT exist. The reason is recorded rather than assumed: a region the pre-request hook cannot reach is a region a cache breakpoint or an injected context can never be applied to (doc 0001 § 6 seam 1).

Cache-boundary markers (`V-REQ-23`) MUST be reachable. Whether they are reached **transitively** — because a marker is carried on a value that some region-level option already replaces — or through an option of their own is AI-11's to determine; either satisfies this requirement, and neither may be left unproven.

Totality MUST be asserted by **enumerating the region set**, so that a region added by a later milestone without a rebuild path fails this requirement rather than passing unnoticed.

### Scenarios

- **S-REX-007** — Given a constructed request, when a caller derives a request replacing the model identity, then the derived request reports the new identity and the source reports the old one.
- **S-REX-008** — Given a constructed request, when a caller derives a request replacing the messages, then the derived request carries the new messages in the supplied order and the source carries its own.
- **S-REX-009** — Given a constructed request, when a caller derives a request replacing the system instruction, the tool set, the tool choice and each generation option in turn, then each replacement is observable on the derived request and absent from the source.
- **S-REX-010** — Given a request whose system instruction, tool set or messages carry cache-boundary markers, when a caller derives a request through the region-level path, then the markers on the derived request are those the caller supplied and the source's markers are unchanged.
- **S-REX-011** — Given the enumerated region set, when the rebuild path is checked against it, then every region has a reachable path, and a region without one fails this scenario.
- **S-REX-053** *(appended by Phase 0 re-verification)* — Given a request whose system instruction is replaced through the rebuild path, when the derived request's system instruction is read back through its accessor, then it is the supplied one. This is asserted through the **accessor** and not through internal state, because the region is stored in two places on a constructed request and a rebuild that refreshed only one of them would revert it silently (`design.md` § 2.2.1).

---

## R-REX-003 — Opaque payloads survive a rebuild byte-identically *(AI-12.1, pin)*

WHEN a request carrying reasoning round-trip tokens (`V-REQ-11`) is rebuilt, THEN every token on the derived request MUST be **byte-identical** to the corresponding token on the source.

This extends AI-07.3's property across the path session persistence will later travel. Nothing on the rebuild path may re-encode, re-marshal, trim, normalize or reconstruct an opaque payload; it is copied or it is carried.

The same guarantee MUST hold for tool-call argument bytes (`V-REQ-17`) and for provider-extension values (`R-REX-006`), for the same reason and by the same mechanism.

### Scenarios

- **S-REX-012** — *(pin)* Given a request whose assistant message carries reasoning content with a round-trip token containing non-UTF-8 bytes, when the request is rebuilt with an unrelated change, then the derived request's token compares byte-equal to the source's.
- **S-REX-013** — *(pin)* Given a request carrying tool-call argument bytes with insignificant whitespace and a non-canonical key order, when it is rebuilt, then the derived request's argument bytes compare byte-equal.
- **S-REX-014** — *(pin)* Given a request carrying a provider extension whose value is arbitrary bytes, when it is rebuilt with an unrelated change, then the derived request's extension value compares byte-equal.

---

## R-REX-004 — A per-request override replaces the construction-time value; absence falls through *(AI-12.2)*

WHEN a rebuild supplies a generation option the source request already carries, THEN the derived request's effective value for that option MUST be the supplied one, observable through the option's readback accessor.

WHEN a rebuild does **not** supply an option the source carries, THEN the derived request MUST report the source's value **and** the source's presence flag, unchanged.

WHEN a rebuild supplies an option the source does **not** carry, THEN the derived request MUST report it as present, and the source MUST still report it as absent.

An override MUST NOT require the caller to restate any other region. Overriding is the same option value applied again — not a second mechanism — so the last application wins, as the option contract already documents.

### Scenarios

- **S-REX-015** — Given a request constructed with temperature `0.2`, when a caller derives a request supplying temperature `0.9`, then the derived request reports `0.9` and set, and the source reports `0.2` and set.
- **S-REX-016** — Given a request constructed with maximum output tokens set and temperature unset, when a caller derives a request supplying only temperature, then the derived request reports the source's maximum output tokens unchanged and reports temperature as set.
- **S-REX-017** — Given a request with no stop sequences, when a caller derives a request supplying stop sequences, then the derived request reports them as set and the source still reports them as unset.
- **S-REX-018** — Given a request constructed with temperature `0.7`, when a caller derives a request supplying temperature `0` , then the derived request reports `0` **and** set — distinguishable from an unset temperature.
- **S-REX-019** — Given a single derivation applying the same option twice with different values, when the derived request is read back, then it reports the value applied last.

---

## R-REX-005 — Derive-time validation is construction's validation *(AI-12.2)*

Rebuild validation MUST run **the same rules, in the same documented order, reporting the same rule classes at the same positions** as construction. There MUST NOT be a second validation path, a weaker one, or one that skips rules the source already satisfied.

WHEN a rebuild produces a request that violates a rule, THEN the reported failure MUST be indistinguishable — by `errors.Is` class and by rendered position — from the failure the same regions would produce through construction.

WHEN a rebuild replaces a region in a way that invalidates a **cross-region** rule that the source satisfied, THEN the rebuild MUST fail. A cross-region rule that only ran at construction would be a rule the derive path silently drops.

Bounds and rules that arrive in later milestones MUST apply to the derive path with no edit at the derive site. This is a structural requirement on the implementation: one rule set, two callers.

Equivalence is asserted **between the two doors** — the same regions, constructed versus derived, yield the same class at the same rendered position. It MUST NOT be asserted against an absolute ordinal in the rule order, because the order is shared with milestones that append to it and an absolute ordinal would make this requirement fail on an append that changed nothing about its behavior.

### Scenarios

- **S-REX-020** — Given a valid request, when a caller derives a request supplying a maximum output tokens value of `0`, then it fails, `errors.Is` reports `ErrOutOfRange`, and the position renders `maxOutputTokens` — the same class and position construction reports.
- **S-REX-021** — Given a valid request, when a caller derives a request supplying an empty model identity, then it fails with `ErrEmpty` at `model`.
- **S-REX-022** — Given a valid request, when a caller derives a request supplying an empty message sequence, then it fails with `ErrEmpty` at `messages`.
- **S-REX-023** — Given a request carrying a tool set and a specific tool choice naming a declared tool, when a caller derives a request replacing the tool set with one that does not declare that tool, then the derivation fails with `ErrUnresolvedReference` at the tool choice's position, proving the cross-region rule ran on the derive path.
- **S-REX-024** — Given a request whose messages a rebuild replaces with a sequence violating a content rule, when the derivation runs, then it fails with the class and composed position construction reports for the same content — for example `messages[1].content[0]`.
- **S-REX-025** — Given a request violating one rule from each of two regions through a single derivation, when the derivation runs many times, then the identical failure — same class, same position — is reported every time, and it is the one construction would report first.

---

## R-REX-006 — A provider-specific value is carried namespaced, typed, opaque and byte-exact *(AI-12.3)*

A request MUST be able to carry provider-specific values the neutral vocabulary deliberately does not model, each under a **namespace** naming the provider that claims it.

The carried value MUST be **opaque**: Layer 1 MUST NOT parse, interpret, re-encode, re-marshal, trim, normalize, case-fold or reorder it, and MUST NOT require any encoding of it. The only property Layer 1 decides about a value is whether it is empty.

The carrier MUST be **typed**: a sealed value type this package alone can produce, not an untyped payload and not a bare byte slice. No vendor wire type may cross the Layer 1 method boundary through it.

A value MUST read back **byte-exact** from another package. A reader MUST be able to ask for one namespace and MUST be able to read the whole set in a defined order.

Applying the same namespace twice MUST be last-wins: the later value replaces the earlier one, and the namespace MUST keep the read-back ordinal of its **first** application, so read-back order does not depend on revision history.

The escape hatch MUST NOT be a way to use a provider that has no adapter (register trap 2), and Layer 1 MUST NOT maintain any list of recognised namespaces.

### Scenarios

- **S-REX-026** — Given a request carrying an extension under a namespace with an arbitrary byte value, when an external package reads that namespace, then it obtains the value byte-equal to what was supplied.
- **S-REX-027** — Given a request carrying an extension whose value is not valid UTF-8, when it is read back, then it is byte-equal, because the package carries bytes rather than text.
- **S-REX-028** — Given a request carrying no extension for a namespace, when a reader asks for that namespace, then the reader is told it is absent rather than handed a zero value it must interpret.
- **S-REX-029** — Given a request whose extension value a reader received, when the reader mutates the returned bytes and re-reads, then the request's value is unchanged.
- **S-REX-030** — Given a request built by applying namespaces `a`, `b`, then `a` again with a different value, when the extension set is read back, then it holds two extensions, `a` first with the second value and `b` second.
- **S-REX-031** — Given a namespace naming a provider for which no adapter exists, when a request carrying it is constructed, then construction succeeds, because Layer 1 recognises no namespace and rejects none.

---

## R-REX-007 — A foreign namespace is invisible to the adapter that does not claim it, and inert in validation *(AI-12.3)*

WHEN a consumer reads a request scoped to the namespace it claims, THEN a value stored under any other namespace MUST NOT be reachable through that read, and MUST NOT change the consumer's output.

The pass-through MUST be **inert in validation**: no rule other than the extension region's own construction rules may consult an extension, and no rule may behave differently because a namespace is present. Two requests differing only in a **third** provider's namespace MUST validate identically — same outcome, and on failure the same class at the same position.

The pass-through MUST **participate in equality**, compared structurally — namespace by string equality, value by byte equality, in read-back order — and MUST NOT be interpreted or special-cased per namespace when compared. "Inert" constrains interpretation, not participation: an equality that ignored extensions would let a rebuild drop every extension and still be reported equal.

**Every walk over the request's regions MUST account for this one.** Participation in the documented equality is necessary and **not sufficient**: `R-AMR-015`'s round-trip pin is realised by a rebuild-from-readback and a region-by-region comparison that do **not** route through the documented equality, so a region absent from *those* walks is dropped and unnoticed there too (`design.md` § 7.2). Any walk that reconstructs a request from its readable surface MUST re-apply this region, and any walk that compares two requests region by region MUST compare it.

### Scenarios

- **S-REX-032** — Given a request carrying an extension under namespace `alpha`, when a consumer that claims namespace `beta` reads the request, then it finds no `beta` extension.
- **S-REX-033** — Given two requests identical except that one carries an extension under namespace `alpha`, when a translator that claims `beta` renders each, then the two outputs are byte-identical.
- **S-REX-034** — Given a translator that claims `alpha`, when it renders the request carrying the `alpha` extension, then its output contains the extension value byte-exactly.
- **S-REX-035** — Given two requests differing only in a third provider's namespace, when both are constructed, then both succeed.
- **S-REX-036** — Given two requests differing only in a third provider's namespace and both violating the same rule in another region, when both are constructed, then both fail with the same class at the same position.
- **S-REX-037** — Given two requests identical except for one extension value, when they are compared under the documented equality, then they are **not** equal.
- **S-REX-038** — Given a request carrying extensions, when it is rebuilt with an unrelated change and the result is compared to the source under the documented equality, then the extension region is present on the derived request and the two differ only where the change was made.
- **S-REX-054** *(appended by Phase 0 re-verification)* — Given a request carrying extensions, when it is reconstructed from its readable surface alone — the round-trip walk `R-AMR-015` pins, which does not use the documented equality — and the reconstruction is compared region by region against the source, then the extension region survives with byte-equal values, and a reconstruction that omitted it fails this scenario.

---

## R-REX-008 — The extension region has its own rules, reported through AI-04's vocabulary *(AI-12.3)*

A namespace MUST be present. WHEN a namespace is empty, or consists only of whitespace, THEN validation MUST fail with `ErrEmpty` at `extensions[i].namespace`, where `i` is the extension's ordinal.

A value MUST carry at least one byte. WHEN a value is empty, THEN validation MUST fail with `ErrEmpty` at `extensions[i].value`.

The value MUST NOT be trimmed, folded or normalized. Emptiness is a rejection criterion, never a normalization — a value of whitespace bytes is a legal value, because the bytes are opaque and the package does not know they are whitespace.

No rule beyond these two MUST be applied to a namespace or a value. In particular, no format, prefix convention, length bound or character-class rule is imposed.

These rules MUST run in the request's documented rule order and MUST report through AI-04's existing rule classes. This capability MUST NOT append a rule class.

### Scenarios

- **S-REX-039** — Given an extension applied with an empty namespace, when the request is constructed, then it fails, `errors.Is` reports `ErrEmpty`, and the position renders `extensions[0].namespace`.
- **S-REX-040** — Given an extension applied with a whitespace-only namespace, when the request is constructed, then it fails identically, because a name made of spaces names nothing.
- **S-REX-041** — Given an extension applied with an empty value, when the request is constructed, then it fails with `ErrEmpty` at `extensions[0].value`.
- **S-REX-042** — Given an extension whose value is a single space byte, when the request is constructed and read back, then construction succeeds and the value is byte-equal, because the bytes are opaque.
- **S-REX-043** — Given a request whose second extension carries an empty namespace, when it is constructed, then the position renders `extensions[1].namespace`, identifying the extension by ordinal and never by namespace.
- **S-REX-044** — Given a namespace containing punctuation, dots and non-ASCII bytes, when the request is constructed, then construction succeeds, because no format rule is imposed.
- **S-REX-045** — Given the same rules applied through the rebuild path, when a derivation supplies an extension with an empty namespace, then it fails with the same class at the same position construction reports.

---

## R-REX-009 — Reading the option set and the extension set is deterministic *(AI-12.4)*

WHEN a caller reads or iterates a request's generation options and its provider extensions twice, THEN both reads MUST yield **identical order and identical content**.

The order MUST be a property of the request rather than of the run: the same request read in a different process, or after any number of intervening reads, yields the same order. No map iteration may reach any of these surfaces.

The extension order MUST be first-application order, per `R-REX-006`. The option order MUST be the documented option order.

This requirement is about the **neutral surface only**. It guarantees that a future serializer cannot inherit nondeterminism from the request. Wire-byte determinism is out of scope and belongs to AI-26.1 and AI-26.4, where wire bytes first exist.

### Scenarios

- **S-REX-046** — Given a request carrying five extensions applied in a known order, when the extension set is read back a hundred times, then every read yields the same five namespaces in the same order with byte-equal values.
- **S-REX-047** — Given two requests built from identical inputs in the same order, when each has its extension set read back, then the two sequences are identical.
- **S-REX-048** — Given a request carrying all four generation options, when the applied-option set is read back repeatedly, then the order is identical on every read and matches the documented option order.
- **S-REX-049** — Given a request carrying options and extensions, when it is rendered through its string form repeatedly, then the rendering is byte-identical every time.

---

## R-REX-010 — The extension region renders no payload through any formatting verb *(AI-12.3)*

A namespace is caller-supplied data and a value is a provider payload; neither is a structural fact about the request.

The extension carrier type MUST define both a string rendering and a Go-syntax rendering, and the request's own renderings MUST account for the region. Every one of them MUST name **structural facts only**: that the region is present, and how many extensions it holds. None MUST reproduce a namespace or any byte of a value.

WHEN a request carrying a secret namespace and a secret value is formatted through any verb, THEN neither secret MUST appear in the output.

The Go-syntax rendering is required explicitly and not by implication: without it, that verb falls back to reflection and prints the unexported fields (`V-FAIL-13`, and `R-AMR-017`'s reasoning applied to the new region).

### Scenarios

- **S-REX-050** — Given a request whose extension namespace and value each carry a distinct secret, when the request is formatted through the default, string, extended and Go-syntax verbs, then neither secret appears in any of the four outputs.
- **S-REX-051** — Given the same request, when it is formatted, then the rendering names the extension region and the number of extensions, so a diagnostic reader still learns the shape.
- **S-REX-052** — Given an extension value formatted directly through the same four verbs, when the outputs are inspected, then none reproduces the namespace or any value byte.
