# ADR 0009: Redefine cachicamas as a multiplayer agentic system

> Status: **Accepted** (2026-08-17)
> Author: cachicamas parent orchestrator
> Supersedes in part, **by reference and without editing any prior document**: the root
> `README.md` identity statement; [ADR 0008](0008-adopt-the-cachicamas-delivery-loop.md)'s
> product framing (§ D1 here); [ADR 0004](0004-adopt-tau-3-layer-agentic-architecture.md)'s
> 2026-08-10 terminology amendment and its dependency-rule "top of the stack" cell (§ D2, § D3);
> [ADR 0005 § D4 row G6](0005-promote-agent-stack-to-own-module.md#d4--v1-scope-for-cross-cutting-concerns)
> (§ D4); and two § 8 non-goals plus the § 5 layer name of
> [the v2 architecture reference](../architecture/0001-cachicamas-agent-stack-v2.md) (§ D2, § D4, § D5).
> Rides on: [ADR 0005 § D1](0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) —
> the module boundary and the network-only rule are unchanged and load-bearing.
> Companion: [ADR 0008](0008-adopt-the-cachicamas-delivery-loop.md) — the delivery loop survives
> intact as engineering machinery; only its product claim is retired.

---

## Resolved TOC

| § | Settles |
|---|---|
| [Context](#context) | Why the identity changes now |
| [D1](#d1--cachicamas-is-a-multiplayer-agentic-system-for-building-and-running-a-company) | The product identity |
| [D2](#d2--layer-3-is-the-archetype-layer) | The Layer 3 name |
| [D3](#d3--layer-3-is-not-the-top-of-the-stack) | The ceiling that is not one |
| [D4](#d4--mcp-is-the-standard-business-system-integration-pattern) | The integration pattern |
| [D5](#d5--the-database-administrator-archetype-is-planned) | The second archetype |
| [D6](#d6--each-business-system-owns-its-own-tables) | Data ownership |
| [D7](#d7--follow-ups-this-adr-creates-but-does-not-execute) | What this ADR obligates, not performs |
| [Consequences](#consequences) | What we pay |

---

## Context

Two identities currently coexist in this repository, and only one of them is being built.

The written identity — the root `README.md`, and [ADR 0008](0008-adopt-the-cachicamas-delivery-loop.md)
in four places — says cachicamas is Witsaba's Software Development Framework: an orchestrator that
wraps the `/sdd-*` pipeline, where "the framework is the product." ADR 0008's Option A rejection
argues that "cachicamas-the-product learns nothing" (~line 59), its Option B and priced-comparison
table score each option on "product capability for cachicamas" (~lines 65, 86), and its first
positive consequence declares the revived planning chain "the product's first real capability"
(~line 172). All four sentences assume the pipeline-wrapper identity.

The built identity is something else. Layer 1 — a vendor-portable model adapter — is complete at
42 of 42 milestones ([doc 0002](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md)).
Layer 2 — a portable agent runtime that can run *any* agent, pure mechanism with no judgement — is
at 14 of 24 ([doc 0003](../architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)).
Layer 3 is planned as the home of specialist agents, of which a coding agent is only the first
([doc 0004](../architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md)). Nothing in
that stack is specific to software delivery, and the parts of the old identity that were — the
PRD-orchestrator thin slice, its scope table, its schema hierarchy — are archived or rescinded.

This ADR resolves the split in favour of what is being built, and names it.

## Decision

### D1 — cachicamas is a multiplayer agentic system for building and running a company

**cachicamas is a multiplayer agentic system for building and running a company.** Everything a
company needs to operate — database administration, finance, marketing, ticketing, software
development — exists as cooperating specialist agents that employees talk and work with. It is
usable by any company; Witsaba ([witsaba.com](https://witsaba.com/)) is its **first user, not its
boundary**.

This supersedes, by reference:

- The root `README.md` identity statement ("Witsaba's Software Development Framework … The
  framework is the product"), rewritten in the same change that lands this ADR.
- ADR 0008's product definition — the four passages cited under [Context](#context) that frame
  "cachicamas-the-product" as the delivery loop and declare the SDD planning chain "the product's
  first real capability."

**What survives:** every operative decision of ADR 0008. The delivery loop, the projection
invariant, the judge protocol and the artifact deletions remain valid engineering machinery for
*building* cachicamas. What is retired is only the claim that this machinery *is* the product.
Likewise, SDD remains the engineering process this repository is built with; it is no longer the
thing being sold.

### D2 — Layer 3 is the archetype layer

**Layer 3 is named the archetype layer.** An **archetype** is the implementation of one specialist
agent: its policy, tools, resources, persistence, and frontend. The **coding archetype**
(`backend/agent/src/coding/`, planned by doc 0004 — an agent that codes, comparable to Pi or
Opencode) is the layer's **first occupant, not its definition**.

This supersedes, by reference:

- ADR 0004's terminology amendment of 2026-08-10 (its header blockquote, ~lines 16–26), which
  fixed *"the application layer"* as Layer 3's name.
- The § 5 naming of [the v2 architecture reference](../architecture/0001-cachicamas-agent-stack-v2.md#5-layer-3--the-application-layer),
  which carries that name in its heading and body.

**The substance of the superseded amendment survives entire.** Layer 3 is a *position* in the
stack, not a program; one occupant does not define it; the runtime beneath it must stay incapable
of caring which occupant is standing on it. Only the noun changes: what that document calls "an
application" is an **archetype**, and every layer-vs-occupant argument it makes transfers verbatim
under the new noun. The rename exists because "application" describes software generically, while
"archetype" says what a Layer 3 occupant actually is — the shape of one specialist agent that a
company runs.

### D3 — Layer 3 is not the top of the stack

**Higher layers may be added above Layer 3.** This supersedes, by reference, ADR 0004's
dependency-rule row (~line 253) that records `cachicamas_coding` with an empty "may NOT import"
cell annotated *"— (top of the stack)"*.

That cell was already half-retired: [ADR 0005 § D1 row 3](0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2)
gave Layer 3 a real import ceiling. What remained was the *architectural* claim that nothing sits
above Layer 3, and this ADR retires it. A multiplayer system of cooperating archetypes will need
machinery an archetype does not own — cross-archetype coordination is the obvious candidate — and
that machinery, when it is designed, will be a layer above 3, not a bigger archetype. This ADR
reserves the position; it deliberately does not design the occupant.

### D4 — MCP is the standard business-system integration pattern

**Each business system runs its own MCP server and has exactly one owning archetype.** MCP is
promoted from a deferred seam to the standard pattern by which archetypes reach business systems:

- Superseded by reference: [ADR 0005 § D4 row G6](0005-promote-agent-stack-to-own-module.md#d4--v1-scope-for-cross-cutting-concerns)
  ("MCP as a dynamic tool source — seam now", ~line 287), and the v2 architecture reference's § 8
  non-goal "Sandboxing, MCP, and subagents — seams 3, 4 and 12 exist; the implementations do not"
  (~line 746) *as it applies to MCP*. The sandboxing and subagent halves of that non-goal are
  untouched.
- A future ticket system runs its own MCP server, owned by a ticket archetype. The same holds for
  finance, marketing, and every business system after them.

**This rides on, and does not overturn, [ADR 0005 § D1 row 3](0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2):**
Layer 3 reaches other modules over the network only, never by import. MCP is precisely a
network-boundary protocol, so the promotion strengthens the existing rule's economics rather than
bending it — the pattern the boundary forced is now the pattern the product wants.

### D5 — The Database Administrator archetype is planned

**A Database Administrator archetype will front the existing `backend/database_administrator`
service through an MCP client**, making it the first business system integrated under § D4 and the
second archetype after coding.

This supersedes, by reference, the v2 architecture reference's § 8 non-goal (~lines 741–743):
*"`database_administrator` driving an agent session — permitted by ADR 0005 § D1 row 5,
unexercised, and priced there."* Two corrections to read into that supersession:

- The non-goal is retired in *direction* as well as in status: the planned wiring is not the
  service importing the agent module (D1 row 5's in-process direction, which stays unexercised and
  keeps its priced build cost), but an archetype consuming the service over MCP — the D1 row 3
  direction, which needs no Dockerfile surgery at all.
- "Planned" means planned: the archetype gets its own milestone document when its planning starts
  (§ D7c). This ADR fixes the destination and the pattern, nothing more.

### D6 — Each business system owns its own tables

**Data ownership is per system, brokered by the DBA archetype.** Each business system owns its own
tables; no archetype writes to another system's schema. When any archetype needs database work —
schema, queries, migrations, capacity — it asks the Database Administrator archetype.

The coding archetype's existing rule — *"the agent never writes back to Postgres"*
([ADR 0006 § D1](0006-resolve-skill-and-prompt-source-of-truth.md#d1--two-stores-one-resolver);
restated in doc 0004's obligations table) — **remains that archetype's answer, not a layer-wide
rule**. A coding agent has no tables to own, so read-only is its correct posture. An archetype
that *is* a business system's owner (the ticket archetype over the ticket system's tables) writes
to its own system as a matter of course; what D6 forbids is writing to somebody else's.

### D7 — Follow-ups this ADR creates but does not execute

Three obligations, each owned elsewhere:

- **(a) An openspec change on the `agent-contract-vocabulary` spec.**
  `openspec/specs/agent-contract-vocabulary/spec.md` fixes **"a Layer 3 application"** as the
  normative consumer term (the R-AGV-009 consumer-term clause, ~line 146; scenario S-AGV-028,
  ~line 153; Trap 4's corrected phrasing, ~lines 331–335). Renaming the normative term to
  **"a Layer 3 archetype"** is a change to a promoted spec and therefore requires an SDD change
  with its own delta — never a documentation edit. Until that change lands, "a Layer 3
  application" in shipped Layer 2 artifacts is read as "a Layer 3 archetype", exactly as ADR
  0004's amendment established read-with-substitution before it.
- **(b) Re-scope [PRD 0001](../prd/0001-cachicamas-delivery-loop.md).** The PRD is accepted and
  never consumed; before it generates milestone doc 0005 (prefix `DF-`), it must be re-scoped
  under the D1 identity — the loop as machinery that builds the system, not as the system.
- **(c) Future archetypes get their own milestone documents.** The DBA, ticket, finance, and
  marketing archetypes each enter the `docs/architecture/milestones/` sequence when their planning
  starts, under the ADR 0007 grammar. This ADR names them so their absence reads as sequencing,
  not omission; it schedules none of them.

## Consequences

### Positive

- The repository states one identity, and it is the one the shipped code supports. The stack's
  strongest property — a runtime that cannot tell which agent it is running — becomes the
  product's central claim instead of an implementation detail.
- "Archetype" gives the Layer 3 occupancy discussion a noun that scales past software development,
  which "application" and "coding agent" both failed to do.
- The MCP promotion converts ADR 0005's network-only boundary from a constraint into the intended
  shape of every future integration.

### Negative / risk

- **Vocabulary drift until D7(a) lands.** The promoted spec, its scenarios, and shipped Layer 2
  artifacts say "a Layer 3 application"; this ADR says "archetype". Read-with-substitution is a
  documented but real reading cost, and it persists until the SDD change closes it.
- **PRD 0001 is blocked, not advanced.** D7(b) adds a re-scope in front of milestone doc 0005 that
  did not exist yesterday.
- MCP servers per business system are an operational surface — versioning, auth, discovery — that
  no document yet designs. D4 fixes the pattern before pricing that surface.

### Neutral

- No code moves. Every decision here renames, re-scopes, or reserves; none relocates a package,
  changes an import rule, or amends a contract already shipped.
- The delivery loop (ADR 0008 D1–D5), the DAG convention (ADR 0007), the module boundary
  (ADR 0005), and the skill/prompt resolution (ADR 0006) are all unchanged.
- SDD keeps building cachicamas exactly as before. A process does not mind not being the product.

## References

- [ADR 0004 — Adopt 3-Layer Agentic Architecture](0004-adopt-tau-3-layer-agentic-architecture.md)
  — terminology amendment and dependency-rule row superseded in part (§ D2, § D3)
- [ADR 0005 — Promote the agent stack to the `backend/agent` Go module](0005-promote-agent-stack-to-own-module.md)
  — § D1 ridden on (§ D4); § D4 row G6 superseded (§ D4)
- [ADR 0006 — Resolve the skill and prompt source-of-truth split](0006-resolve-skill-and-prompt-source-of-truth.md)
  — the coding archetype's read-only posture (§ D6)
- [ADR 0008 — Adopt the cachicamas delivery loop](0008-adopt-the-cachicamas-delivery-loop.md)
  — product framing superseded (§ D1); machinery retained
- [cachicamas agent stack — hardened architecture (v2)](../architecture/0001-cachicamas-agent-stack-v2.md)
  — § 5 naming and two § 8 non-goals superseded in part (§ D2, § D4, § D5)
- [PRD 0001 — The cachicamas delivery loop](../prd/0001-cachicamas-delivery-loop.md) — re-scope
  obligation (§ D7b)
- Milestone documents: [doc 0002 (Layer 1, 42/42)](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md)
  · [doc 0003 (Layer 2, 14/24)](../architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)
  · [doc 0004 (the coding archetype, 0/25)](../architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md)
- `openspec/specs/agent-contract-vocabulary/spec.md` — the normative consumer term § D7(a) renames
