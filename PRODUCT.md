# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Employees of a company, signed in with GitHub, working in a browser during their normal
working day. They are not the people who built the agents; they are the people who *talk
and work with them* to get a job done — asking a question, handing over a task, watching
it run, approving or refusing what it proposes to do.

Witsaba (witsaba.com) is the first user, not the boundary (ADR 0009 § D1). The product is
built to be used by any company.

Secondary audience, on the public landing surface only: someone evaluating whether this
is a thing their company could run.

## Product Purpose

cachicamas is a **multiplayer agentic system for building and running a company**.
Everything a company needs to operate — database administration, finance, marketing,
ticketing, software development — exists as cooperating specialist agents that employees
talk and work with.

Success is an employee opening the product, finding the specialist they need, handing it
work in plain language, and being able to see and control what it does.

## Positioning

The unit of the product is the **archetype**: one specialist agent implemented whole — its
policy, its tools, its resources, its persistence, and its own frontend (ADR 0009 § D2).
Archetypes stand on a shared, vendor-portable agent runtime that is deliberately incapable
of caring which archetype is running on it.

Two consequences a neighboring product could not truthfully copy:

- **Every business system gets its own MCP server and exactly one owning archetype**
  (ADR 0009 § D4). Integration is a boundary, not a plugin.
- **Each business system owns its own tables; no archetype writes another's schema**
  (ADR 0009 § D6). When an archetype needs database work it asks the Database
  Administrator archetype.

This is a *company* of agents with jobs and boundaries, not one assistant with a tool list.

## Operating Context

- Authenticated by GitHub OAuth (Auth.js). Single organization per install; a first-run
  "ownboarding" step names it before any other surface is reachable.
- Work with an archetype is a **conversation that runs**: a turn is opened by request, its
  events arrive on a subscribed stream, and it is cancelled by a discrete signal. Output
  arrives token by token. Stop cancels. Failures arrive as a typed envelope inline, never
  as a spinner that never ends, and the client never auto-retries.
- An agent asking permission to act is a **suspension inside the run**, surfaced on the
  same stream as its output — not a side channel and not an out-of-band approval.
- Employees will run several archetypes and will move between them during a day.

## Capabilities and Constraints

Built and frozen today:

- Layer 1 — vendor-portable model adapter. Complete, 42 of 42.
- Layer 2 — portable agent runtime, pure mechanism, no judgement. Complete and frozen at
  24 of 24; the browser wire and the error envelope are frozen with it.
- Identity, session, organization ownboarding, and the route guard chain.

Not built:

- **No archetype exists on disk yet.** The chat archetype is planned at 0 of 12
  (`docs/architecture/milestones/0005-…`); the coding archetype and the Database
  Administrator archetype are planned and unstarted.
- Therefore **every archetype the interface shows is mocked**, and must read as mocked to
  the person looking at it. Status is product truth here: `chat` is the one being built,
  the rest are planned. Overstating readiness is the one lie this interface can tell.

Terminology that must survive:

- **archetype** — one specialist agent, implemented whole. Never "app", "bot", "plugin",
  or "assistant" in product copy.
- **turn**, **stream**, **cancel**, **permission** — the vocabulary of a run.
- **organization** — the single tenant that owns the work.
- No biological metaphors: never "brain", "mind", "neural".
- The name is lowercase: **cachicamas**.

Explicitly undecided: pricing, deployment model, multi-organization support, and any
cross-archetype coordination surface (ADR 0009 § D3 reserves the position and deliberately
does not design the occupant).

## Brand Commitments

- The wordmark is `cachicamas`, always lowercase, never title-cased.
- Product voice is plain, literal and unhyped. It states what a thing is and what it will
  do. It does not sell in the app.

## Evidence on Hand

Real, in this repository:

- `docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md` — the identity.
- `docs/architecture/milestones/0002-…`, `0003-…`, `0004-…`, `0005-…` — the shipped and
  planned layer plans, with real completion counts.
- `openspec/specs/frontend-chat-layer1/spec.md` — the frozen browser wire.

Absent, and not to be fabricated: customers, pricing, benchmarks, testimonials, uptime
claims, screenshots of archetypes doing real work. Any conversation, tool call, cost
figure or archetype status shown in the interface is **demonstration data** and must be
labeled as such where a viewer could mistake it for a live system.

## Product Principles

1. **The archetype is the unit.** The interface is organized by *which specialist*, not by
   which feature. Everything a person does happens inside one archetype's surface.
2. **Never overstate readiness.** A planned archetype looks planned. Mocked data looks
   mocked. The product's credibility is the only thing it currently has.
3. **A run is visible and interruptible.** Whatever an agent is doing, the person can see
   it and can stop it. Permission is asked in the flow of the run, not around it.
4. **Boundaries are shown, not hidden.** Which archetype owns what, and what it may not
   touch, is information the employee benefits from seeing.
5. **Literal over evocative.** Plain nouns, real numbers, honest states.

## Accessibility & Inclusion

- **Aphantasia-friendly (UX-4, carried forward from the shipped specs).** No meaning may
  be carried by an image, icon, glyph or spatial metaphor alone — every affordance and
  every status carries a literal text label beside it. No decorative photography of people
  or places. This constrains what may carry meaning; it does not forbid color, depth,
  material, or an icon system used *alongside* labels.
- Keyboard reachable throughout, with a visible focus ring on every interactive element,
  and a skip link as the first focusable element of the document.
- Streaming output is announced politely to assistive technology, not aggressively.
- Model output is rendered sanitized; raw model HTML is never injected.
