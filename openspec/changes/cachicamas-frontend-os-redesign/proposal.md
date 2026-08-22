# Proposal — `cachicamas-frontend-os-redesign`

> **Status**: implemented on `feat/frontend-os-redesign`.
> **Driver**: the frontend still sold the identity [ADR 0009](../../../docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md) retired.
> **Scope**: `frontend/` only. No backend file changes.

## Why

The landing page opened with *"Four primitives, one connected graph"* over organizations,
projects, requirements and milestones; `/home` was a list of GitHub repositories to sync;
`/settings` launched Prompts and Skills. Every one of those surfaces belongs to
cachicamas-as-a-Software-Development-Framework — the identity ADR 0009 § D1 retired in
favour of **a multiplayer agentic system for building and running a company**.

Nothing in the interface said *archetype*, nothing showed the register of specialists, and
the one archetype with a plan in flight ([doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md))
was reachable only from a dropdown.

## What changes

The frontend becomes an **operating system for a company's specialists**: a persistent
shell (status rail, command line, function-key dock) with applications running inside it.

| Surface | Before | After |
| --- | --- | --- |
| `/` | Framework marketing, four primitives, a CLI transcript | The real register rendered live, with a quoted permission suspension |
| `/home` | GitHub workspaces list | **The desk** — the archetype register plus the runtime beneath it |
| `/chat` | Live SSE client against an unwired backend | The chat archetype's application, on scripted demonstration data |
| `/archetypes/[slug]` | — | **New.** A specialist that does not exist yet, and the honest reason why |
| `/settings` | Launchpad grid → Prompts, Skills | **System** — account, runtime readout, and the open decisions |
| `/workspaces/*` | GitHub repo sync | **Removed** |
| `/settings/prompts`, `/settings/skills` | Prompt and skill studios | **Removed** |

## Decisions

- **D1 — The OS metaphor stops at a dock and full-screen applications.** No window
  manager, no draggable windows. The launcher is the command line; the dock is the
  function-key legend. Chosen by the user over free-floating windows and over a tiling
  terminal.
- **D2 — The framework-era surfaces are removed, not hidden.** `/workspaces`,
  `/settings/prompts` and `/settings/skills` and their components, specs and e2e are
  deleted rather than left unadvertised. Chosen by the user over restyling them as system
  applications. Their capability specs are retired below.
- **D3 — The chat screen is rebuilt on mocked data, and the frozen wire is kept.**
  `lib/chat-api.ts` and `lib/chat-types.ts` are **byte-unchanged**: the open-then-subscribe
  wire, its typed error envelope and its transcript parser stay on disk for CH-05 to
  connect. What is replaced is the component tree above them. This costs a recorded delta
  against `frontend-chat-layer1`, which is § Spec deltas below and is the whole reason
  this record exists rather than a silent deletion of a mandated string
  ([doc 0005 inconsistency register #4](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md)).
- **D4 — Demonstration data is labelled wherever it appears.** Structural facts (layer
  counts, milestone counts, owning systems, decision records) are read from this
  repository and asserted against it in `lib/mock/registry.spec.ts`. Activity figures are
  invented, are confined to the one archetype under construction, and are marked `demo` on
  the cell, in the status rail, and on the composer.

## Spec deltas

- `frontend-chat-layer1` — **MODIFIED**. See
  [`specs/frontend-chat-layer1/spec.md`](specs/frontend-chat-layer1/spec.md).
- `workspaces`, `prompts`, `frontend-prompts`, `home-page` — **RETIRED**, in the sense
  that their UI requirements no longer have an implementation. The spec files are left in
  place rather than deleted: retiring a promoted capability is `sdd-archive`'s job and
  needs its own change, and deleting them here would erase the record of what was built
  and why. This proposal is the pointer a later retirement change starts from.
- `frontend-auth`, `frontend-runtime`, `frontend-compose-and-cors`,
  `frontend-dependency-audit`, `frontend-e2e-and-client-data` — **unaffected**. The auth
  chain, the guard order, the CSRF helper, the SSR cookie forwarding and the Playwright
  suite are untouched; only their surrounding chrome is restyled.

## Evidence

- `pnpm build.types` — clean.
- `pnpm lint` — 0 errors, 0 warnings.
- `pnpm fmt.check` — clean.
- `pnpm test:ci` — 58 files, 585 tests, all passing.
- `pnpm build` — client and SSR builds succeed; the direction contract survives into
  `server/*.js` (grep `impeccable-direction`).
