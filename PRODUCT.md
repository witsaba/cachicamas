# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

**Inside the product:** employees of a company, working in a browser during their normal
working day. They are not the people who built the agents; they are the people who *work
with them* — asking a question, handing over a task, reading what came back, and deciding
whether the thing an agent proposes to do actually happens.

Witsaba (witsaba.com) is the first company to use it, not the boundary. The product is
built for any company.

**On the public page:** whoever decides what their company buys. An operations lead, a
founder, a head of finance. They arrive knowing they are short-staffed in a specific way
and leave knowing whether this fills it.

## Product Purpose

cachicamas is where a company hires specialist colleagues who are not people.

A company signs up and gets an Assistant. When it knows which job it is short on, it hires
a specialist — Finance, Support, Integrations, a Database Administrator, a Coding
colleague — and that specialist works alongside the staff, in the same place, with a job,
a set of tools it is allowed to use, and a limit it will not cross without asking.

Success is an employee opening the product, finding the colleague they need, handing over
work in plain language, and being able to see and stop what happens next.

## Positioning

**Colleagues with boundaries, not one assistant with a feature list.**

Three things a neighbouring product cannot truthfully copy without rebuilding around them:

- **Nothing leaves the building unapproved.** An email, a payment, a message to a customer
  — anything with an outside consequence stops and asks a person, showing exactly what it
  is about to do. This is not a setting.
- **One colleague owns the shape of the company's data**, and everyone else asks it. The
  boundaries between specialists are real, written on each profile, and visible to the
  customer.
- **A colleague that fails says so once, and stops.** Nothing retries quietly in the
  background, because the expensive failure is the one nobody was told about.

## Operating Context

- Signed in with GitHub. One company per install; a first-run step names it before
  anything else is reachable.
- Work with a colleague is a conversation. The answer arrives as it is written. Anything
  the colleague does is shown in the conversation — which tool, what for, what came back —
  rather than summarised afterwards.
- A colleague asking permission is a **pause inside the conversation**, not a notification
  somewhere else. Nothing moves until a person answers.
- An employee will talk to several colleagues in a day and will move between them.

## Capabilities and Constraints

What exists today: identity, sign-in, the company record and its first-run setup, and the
whole interface — the public page, the workspace shell, the staff directory, profiles,
teams, the organisation chart, and the conversation.

What does not exist yet: **no specialist has started work.** Every colleague, conversation,
figure and tenure the interface shows is authored demonstration material.

That constrains the interface rather than the copy:

- The workspace carries a **standing demonstration strip** on every screen. It is part of
  the shell, not a banner a screen can forget to render.
- The public page's prices are **preview pricing**, and the pricing section says so in
  words directly beneath the plans. They are a placeholder on the replacement list below,
  not an offer.
- No customer logos, testimonials, review counts, uptime figures or usage statistics
  appear anywhere, and none may be added until they are real.

Terminology, in the product's own words:

- **agent** or **colleague** — one specialist. Never "bot", "assistant" (except the
  Assistant, which is one specific colleague), "plugin" or "app".
- **on staff**, **in training**, **available** — the three things a colleague can be.
- **company**, **team**, **officer role** — how the organisation is described.
- No biological metaphors: never "brain", "mind", "neural".
- The name is lowercase: **cachicamas**.
- **Nothing about how the product is built appears on any surface a customer sees.** Not
  the architecture, not the layers, not the protocols, not the framework, not the
  milestone plan. This is enforced by test, not by taste.

## Brand Commitments

- The wordmark is `cachicamas`, always lowercase, never title-cased.
- Product voice is plain, literal and unhyped. It states what a thing is and what it will
  do. It sells on the public page and stops selling the moment you sign in.
- **The interface plays the category standard straight** (recorded 2026-08-22, at the
  user's explicit direction). This is a standing preference, not a default: the design
  round dealt a wayfinding-programme direction and the user took the canon instead. The
  craft bar is **Intercom and Attio** — products that ship a marketing site and an
  application in one identity, at one level. Nothing here is quirky, and a category-fluent
  person should be able to use it without pausing once.
- **A person is a circle. An agent is a rounded square.** The one rule in the system that
  carries meaning through form — and therefore the one that always ships a literal word
  beside it.

## Evidence on Hand

Absent, and not to be fabricated: customers, testimonials, benchmarks, uptime claims,
review counts, and screenshots of a specialist doing real work.

**The replacement list** — what a real launch must swap out:

1. Every price in `frontend/src/lib/mock/plans.ts`.
2. Every colleague, tenure and workload figure in `frontend/src/lib/mock/staff.ts`.
3. Every conversation in `frontend/src/lib/mock/chat.ts`.
4. The company in `frontend/src/lib/mock/company.ts`, which stands in for the real
   organisation record until the workspace reads it directly.

## Product Principles

1. **The colleague is the unit.** The interface is organised by *who*, not by feature.
   Everything a person does happens with one of them.
2. **Never overstate readiness.** A colleague who has not started work does not have a
   tenure. Demonstration data says it is demonstration data. Credibility is the only thing
   this product currently has.
3. **The work is visible and interruptible.** Whatever a colleague is doing, a person can
   see it and can stop it. Permission is asked in the flow of the conversation.
4. **Boundaries are shown, not hidden.** What a colleague may touch, and what it will hand
   to someone else, is on its profile. That is what makes access grantable.
5. **Literal over evocative.** Plain nouns, real numbers, honest states.
6. **The customer never reads the engineering.** If a surface mentions how it is built,
   that is a defect.

## Accessibility & Inclusion

- **Aphantasia-friendly (UX-4).** No meaning may be carried by an image, icon, glyph,
  colour or shape alone — every avatar ships with a name, every status dot with its word,
  every department hue with its department, and every icon beside a label. This constrains
  what may *carry* meaning; it does not forbid colour, depth, material, or an icon system
  used alongside words.
- Contrast is measured against **every ground a token can land on** — the white surface,
  the page, and the sunken well — never against white alone. Body text clears 4.5:1
  everywhere; a control's border clears 3:1, because on this surface the border is how the
  control is found.
- Keyboard reachable throughout, with one visible focus ring on every interactive element,
  and a skip link as the first focusable element of the document.
- Arriving output is announced politely to assistive technology, never assertively.
- Model output is rendered sanitised; raw model HTML is never injected.
