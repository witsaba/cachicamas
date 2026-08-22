# Proposal — `cachicamas-frontend-workplace`

> **Status**: implemented on `feat/frontend-workplace`.
> **Supersedes**: `cachicamas-frontend-os-redesign` (PR #188), which was built and
> rejected on its direction. This change keeps its structural work — the retirement of the
> framework-era surfaces, the mocked conversation machine, the byte-frozen wire — and
> replaces everything a person can see.
> **Scope**: `frontend/` and `PRODUCT.md`. No backend file changes.

## Why

Two problems, and only the first was known when the previous change shipped.

**The identity was wrong.** The landing page opened with *"Four primitives, one connected
graph"* over organizations, projects, requirements and milestones; `/home` was a list of
GitHub repositories to sync; `/settings` launched Prompts and Skills. Every one of those
surfaces belongs to cachicamas-as-a-Software-Development-Framework. That was fixed by the
superseded change and stays fixed here.

**The audience was wrong.** The superseded change replaced a framework interface with an
*engineering* interface: a dealing-room board of codes, lamps, layer counts and milestone
numbers, addressed to whoever built the system. The people this product is for are
employees who need a colleague, and the people who buy it are companies deciding whether
they are short-staffed. Neither of them is served by a page that reads its own
architecture out loud.

## What changes

The frontend becomes a **workplace**: a public page that sells the product, and behind
sign-in, a company you work inside — a lateral panel, colleagues with profiles, teams, an
organisation chart, and a conversation.

| Surface | Superseded design | This change |
| --- | --- | --- |
| `/` | The register rendered live, layer counts, a quoted suspension | **A page that sells the product** — the offer, the six specialists, how it goes, plans (monthly/annual, four levels), and the questions a buyer actually asks |
| `/home` | The desk — archetype register plus runtime gauges | **The front desk** — who is on staff, where you left off, who you could hire, your teams |
| `/agents` | — | **New.** The staff directory, split into on-staff and could-hire |
| `/agents/[slug]` | `/archetypes/[slug]`, an unbuilt specialist and why | **A colleague's profile** — what they do, how long they have been here, what they may use, and where they stop |
| `/teams` | — | **New.** People and agents in one list, and the paired duo |
| `/organization` | — | **New.** Officer roles, who holds them, and which seats are open |
| `/chat` | Two panels inside an OS frame, command line above | **A conversation** with one named colleague, history beside it |
| `/settings` | System — runtime readout and open decisions | **Settings** — account, company, and what a customer may *not* switch off |
| Shell | Status rail, command line, F-key dock | **A lateral panel**, always present, with the company at the top and the colleagues in it |

## Decisions

- **D1 — The interface plays the category standard straight.** The direction round
  (`concept-seed` key `b489eace`) dealt a wayfinding-programme direction and offered three
  catalog challengers plus the standing exit. The user took **the standing exit**, and
  named **Intercom and Attio** as the craft bar. Recorded in `PRODUCT.md § Brand
  Commitments` as a standing preference, not a default. The counterweights bind: the canon
  is executed at full fidelity, with no smuggled quirk.
- **D2 — Nothing about how the product is built appears on a customer surface.** No
  architecture, no layers, no protocols, no framework, no milestone numbers. This is
  enforced by test — `routes/index.spec.tsx`, `lib/mock/chat.spec.ts`,
  `lib/mock/staff.spec.ts` and `routes/settings/index.spec.tsx` each grep the rendered
  output for the engineering vocabulary — because a rule of taste would last one PR.
- **D3 — A person is a circle; an agent is a rounded square.** The product's premise is
  that you work alongside colleagues who are not people, and a directory where the two are
  indistinguishable is dishonest in a way copy cannot fix. Because shape carries meaning
  here, it never travels alone: the literal word ships beside every avatar
  (`avatar.spec.tsx` pins both halves).
- **D4 — Pricing is a mockup, and says so where a buyer reads it.** Four levels along one
  axis, a monthly/annual switch, and one line under the cards: *preview pricing … nothing
  here is a quote*. `PRODUCT.md § Evidence on Hand` carries the replacement list. No
  customer logos, testimonials, review counts or uptime figures appear anywhere, and none
  may be added until they are real.
- **D5 — The demonstration notice is part of the shell, not a banner.** Every workspace
  screen renders inside `<Workspace>`, which carries it. A screen cannot forget it.
- **D6 — The frozen wire is still frozen.** `lib/chat-api.ts` and `lib/chat-types.ts` are
  **byte-unchanged** across both this change and the one it supersedes. Replacing the mock
  is meant to be a swap of `useMockTurn` for that client with the component tree
  unchanged; anything that does have to change is a seam worth knowing about.
- **D7 — The guard chain moved from routes to section layouts.** `setSsrCookieHeader` →
  `requireAuthRedirect` → `await requireOwnboarding` now lives once per section rather
  than once per route. The order is unchanged and is still asserted at source level — but
  the assertions now read the body of `onRequest` rather than the whole file, because the
  import block is alphabetical and the old test was passing on comment text.

## Spec deltas

- `frontend-chat-layer1` — **MODIFIED**. See
  [`specs/frontend-chat-layer1/spec.md`](specs/frontend-chat-layer1/spec.md). The delta
  is inherited from the superseded change and corrected in three places where this change
  moved the evidence.
- `app-shell` — **MODIFIED in effect, recorded here.** `S-AS-002` (a `<header>` above the
  route's content), `S-AS-010`/`S-AS-011` (the anon shell renders `SignInButton`) and
  `S-AS-020`/`S-AS-021` (the authed shell renders `AvatarDropdown`) described a chrome
  that no longer exists: there is no single shell, because the public site and the
  workspace are different places. The **properties** those scenarios protected are all
  still asserted, and moved with the code — the skip link stays first in
  `routes/layout.spec.tsx`; the avatar, its `aria-label` and the ADR-0009 URL
  sanitisation move to `components/workspace/workspace.spec.tsx`; the sign-in form and its
  GitHub mark stay in `components/sign-in-button/sign-in-button.spec.tsx` and are now the
  **only** sign-in affordance in the product, so the e2e selector
  `form[data-testid="sign-in-button"]` still resolves on `/`.
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
- `pnpm test:ci` — 53 files, 546 tests, all passing.
- `pnpm build` — client and SSR builds succeed; the direction contract survives into
  `server/*.js` (grep `impeccable-direction`).
