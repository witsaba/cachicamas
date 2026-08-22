---
name: cachicamas
description: A workplace where a company's specialist colleagues are staff — met, read, and talked to.
colors:
  canvas: "#f7f8fa"
  surface: "#ffffff"
  sunken: "#f1f3f7"
  deep: "#101724"
  line: "#e4e7ec"
  line-firm: "#d6dae1"
  line-control: "#828b9c"
  ink: "#101724"
  ink-mid: "#3d4757"
  ink-soft: "#5f6a78"
  ink-inverse: "#ffffff"
  brand: "#2b4be8"
  brand-press: "#1e39c4"
  brand-tint: "#eef1fe"
  ok: "#0e7c5a"
  waiting: "#b4530a"
  stop: "#c02434"
  idle: "#5f6a78"
  dept-assistant: "#2b4be8"
  dept-finance: "#0e7c5a"
  dept-support: "#7a2fd6"
  dept-integrations: "#b4530a"
  dept-data: "#0e6fa8"
  dept-engineering: "#c2185b"
typography:
  display:
    fontFamily: "Onest, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif"
    fontSize: "clamp(2.25rem, 5.5vw, 3.5rem)"
    fontWeight: 700
    lineHeight: 1.05
    letterSpacing: "-0.035em"
  headline:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "2.375rem"
    fontWeight: 700
    lineHeight: 1.15
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.3125rem"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.01em"
  card-title:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 600
    lineHeight: 1.35
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  prose:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.6
    letterSpacing: "normal"
  dense:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "normal"
  meta:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.45
    letterSpacing: "normal"
  label:
    fontFamily: "Onest, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.6875rem"
    fontWeight: 600
    lineHeight: 1.4
    letterSpacing: "0.025em"
rounded:
  sm: "6px"
  md: "8px"
  lg: "12px"
  full: "9999px"
spacing:
  tight: "4px"
  snug: "8px"
  base: "12px"
  card: "16px"
  panel: "20px"
  well: "28px"
  band: "80px"
components:
  button-primary:
    backgroundColor: "{colors.brand}"
    textColor: "{colors.ink-inverse}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0 0.875rem"
    height: "2.25rem"
  button-primary-hover:
    backgroundColor: "{colors.brand-press}"
    textColor: "{colors.ink-inverse}"
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0 0.875rem"
    height: "2.25rem"
  button-secondary-hover:
    backgroundColor: "{colors.sunken}"
    textColor: "{colors.ink}"
  button-destructive:
    backgroundColor: "{colors.stop}"
    textColor: "{colors.ink-inverse}"
    rounded: "{rounded.md}"
    padding: "0 0.875rem"
    height: "2.25rem"
  button-link:
    textColor: "{colors.brand}"
    typography: "{typography.body}"
  button-sm:
    typography: "{typography.meta}"
    padding: "0 0.625rem"
    height: "1.75rem"
  button-lg:
    typography: "{typography.body}"
    padding: "0 1.25rem"
    height: "2.75rem"
  form-input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.5rem 0.75rem"
  form-input-disabled:
    backgroundColor: "{colors.sunken}"
    textColor: "{colors.ink}"
  menu-item:
    textColor: "{colors.ink-mid}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "0.375rem 0.5rem"
  menu-item-hover:
    backgroundColor: "{colors.sunken}"
    textColor: "{colors.ink}"
  nav-row:
    textColor: "{colors.ink-mid}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "0.375rem 0.5rem"
  nav-row-current:
    backgroundColor: "{colors.brand-tint}"
    textColor: "{colors.brand}"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.md}"
    padding: "16px"
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    padding: "16px"
  species-label-agent:
    backgroundColor: "{colors.brand-tint}"
    textColor: "{colors.brand}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "1px 6px"
  species-label-person:
    backgroundColor: "{colors.sunken}"
    textColor: "{colors.ink-soft}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "1px 6px"
  avatar-agent:
    rounded: "6px"
    size: "2rem"
  avatar-person:
    backgroundColor: "{colors.sunken}"
    textColor: "{colors.ink-mid}"
    rounded: "{rounded.full}"
    size: "2rem"
---

# Design System: cachicamas

## Overview

**Creative North Star: "The Workplace"**

This is a building, not a dashboard. The product's claim is that a company's
specialists are staff — you meet them, read what they do, see how long they
have been here, and start a conversation — so the interface is organised by
*who*, never by feature. A colleague has a face, a department, a tenure, a
status and a way to be reached, and those five facts are what a card, a row and
a rail entry each carry. Two cards side by side read differently, because every
field on them is a fact about a specific colleague. That is the test for
whether a card earned its place.

The build takes the standing exit: **the category standard, played straight**,
with the craft bar set at Intercom and Attio — products that ship a marketing
site and an application in one identity, at one level. Nothing here is invented
for novelty and nothing here is quirky. A person who has used a modern work
tool should be able to use this one without pausing once. The material is white
cards on a cool-neutral page, 1px lines, 6–12px radii, one brand blue, and six
department hues. Onest sets every word on both sides of the sign-in.

**One token set serves two modes.** The public page is *Persuade*: full-width
bands, 38–56px headlines, a dark closing band, a live proof playing. Everything
behind sign-in is *Operate*: a fixed 14px body size, a persistent 16rem rail,
13px table cells, and no fluid type anywhere. Same colours, same family, same
radii, same shadows. The mode changes the density and the scale in use — never
the vocabulary.

The system's hard floor is that **nothing may mean anything on its own**. No
image, icon, glyph, colour or shape carries meaning without a word beside it.
Every avatar ships a name, every status dot ships its word, every department hue
ships its department, every icon ships its label. This is why the one rule that
does carry meaning through form — a person is a circle, an agent is a rounded
square — is also the one rule that always prints the literal word "Agent" or
"Person" next to itself.

**Key Characteristics:**

- White cards on a cool-neutral page; separation is a value step plus a 1px line, never a heavy shadow.
- One family (Onest), one accent (brand blue), six department hues that identify and never rank.
- Fixed type scale inside the product; exactly one `clamp()` in the whole build, on the marketing h1.
- Elevation is reserved for things that genuinely float: menus, dialogs, the drawer, a card under the pointer.
- Exactly one authored motion moment; everything else is still at rest.
- Every colour value is measured against all three grounds by test, not asserted.
- No meaning rides on colour, shape or icon alone — ever.

## Colors

A near-achromatic product surface with one saturated accent and a six-hue
identity band, all of it measured rather than eyeballed.

### Primary

- **Brand Blue** (`{colors.brand}`): The one accent. It carries the action that commits, the current selection in the rail (`{colors.brand-tint}` ground with brand ink), links, and the focus ring. It never decorates and it never appears twice as a fill on one screen.
- **Brand Press** (`{colors.brand-press}`): The pressed and hovered state of a brand fill. Press darkens; it never lifts.
- **Brand Tint** (`{colors.brand-tint}`): The selected row, the "Agent" chip, and the standing demonstration strip. It is the only pre-mixed tint token in the system.

### Secondary — status, each with exactly one job

- **Working Green** (`{colors.ok}`): Working, available, connected; also the check glyph in a skills list.
- **Waiting Amber** (`{colors.waiting}`): Waiting on a person to decide. The only state that gets its own panel treatment (see Components → Transcript).
- **Stop Red** (`{colors.stop}`): Refused, failed, stopped. Destructive controls.
- **Idle Grey** (`{colors.idle}`): Not started. Deliberately the same value as `{colors.ink-soft}` — nothing is wrong, so nothing is coloured.

### Tertiary — the six departments

- **Assistant** (`{colors.dept-assistant}`), **Finance** (`{colors.dept-finance}`), **Support** (`{colors.dept-support}`), **Integrations** (`{colors.dept-integrations}`), **Data** (`{colors.dept-data}`), **Engineering** (`{colors.dept-engineering}`).

No two are adjacent on the wheel. Each is used as an avatar fill (with white
initials) and as ink for the department's own name. Assistant and Finance share
values with brand and ok respectively; that is intentional economy, not a
collision — a department hue is never read as a status because a status always
prints its word.

### Neutral

- **Surface White** (`{colors.surface}`): Cards, panels, the content column, the rail, the marketing header, alternating page bands.
- **Canvas** (`{colors.canvas}`): The page itself. One value step below a card; that step, plus a hairline, is how a card is separated.
- **Sunken** (`{colors.sunken}`): A well — an input at rest, a hovered row, a code block, a table head, a quiet band.
- **Deep Ink** (`{colors.deep}`): The one dark surface in the product, used exactly once: the public page's closing band. Never inside the workspace.
- **Ink** (`{colors.ink}`), **Ink Mid** (`{colors.ink-mid}`), **Ink Soft** (`{colors.ink-soft}`): Headings and values / body copy / labels, timestamps, placeholders and meta.
- **Line** (`{colors.line}`), **Line Firm** (`{colors.line-firm}`), **Line Control** (`{colors.line-control}`): Decorative rule / a genuine region divide / a control's own findable boundary.

*Ground-truth note: the header comment in `global.css` calls the page
"warm-neutral" in one line. The shipped value is cool-neutral (`#f7f8fa`), which
is what the direction contract and every other comment say. The value is
correct; the one word is stale.*

### Named Rules

**The One Accent Rule.** Brand blue means *commit*, *current*, or *link*. A
brand fill that decorates is a defect. One brand-filled button per screen — six
brand buttons down a grid is six primary actions, which is none, so a card's
call to action is `secondary` on both of its branches.

**The Identify-Never-Rank Rule.** A department hue names a department. It never
encodes status, priority, quality or order, and it never travels without the
department's name somewhere on the same screen.

**The Three Grounds Rule.** Every colour is measured against `surface`, `canvas`
*and* `sunken`, never against white alone — a token picked on white quietly
fails inside a hovered row. `src/__tests__/contrast.spec.ts` parses the palette
straight out of `global.css` and enforces it on every run: text ≥ 4.5:1 on all
three grounds, a control boundary ≥ 3:1 on all three, and white ≥ 4.5:1 on every
filled surface including all six departments. The tightest pairing in the whole
system is `dept-integrations` on `sunken` at **4.52** — a darker well or a
lighter hue breaks that one first.

**The Derived Tint Rule.** The only pre-mixed tint token is `brand-tint`. Every
other tinted panel is composed from an existing token at low alpha — a pending
permission is `border-waiting/35` over `bg-waiting/[0.06]`, a fault is
`border-stop/30` over `bg-stop/[0.05]`, the drawer scrim is `bg-ink/35`. Never
introduce a new hex to make a tint.

**The Decorative-Line Rule.** `line` and `line-firm` sit *below* the 3:1
boundary floor on purpose: a 3:1 grid rule at this density reads as a wireframe.
They may only ever separate. The moment a border is how a person finds a
control, it must be `line-control`.

## Typography

**Display Font:** Onest — same as body. There is no second face.
**Body Font:** Onest (400 / 500 / 600 / 700), with `ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif` behind it.
**Mono:** `ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace` — reachable only inside `.prose-work code` and `.prose-work pre`, i.e. only in sanitised model output. Nothing in the chrome is monospaced.

**Character:** A neutral grotesque with just enough character to stay ours at
11px and at 56px. It is preconnected in `root.tsx` so no surface repaints its
own type after first byte. Headings tighten as they grow (`-0.01em` at 17–21px,
`-0.025em` at 38px, `-0.035em` on the hero); body never tracks.

### Hierarchy

- **Display** (700, `clamp(2.25rem, 5.5vw, 3.5rem)`, 1.05, `-0.035em`): The marketing h1 and nothing else. The single fluid size in the build.
- **Headline** (700, 38px / `--text-3xl`, `-0.025em`): Public-page section heads.
- **Title** (600, 21px / `--text-xl`, `-0.01em`): The one `h1` of every workspace screen, always via `<PageHeader>`.
- **Card Title** (600, 17px / `--text-lg`, `-0.01em`): A colleague's name on a card or a staff row.
- **Body** (400/500, 14px / `--text-base`, 1.5): The product's body size. Buttons, nav rows, form fields, ledes, card copy. On the public page it steps up to `text-lg` (17px) for section ledes.
- **Prose** (400, 15px / `--text-md`, 1.6): Conversation text only — a person's message and, through `.prose-work`, a colleague's sanitised reply.
- **Dense** (400, 13px / `--text-sm`): Table cells, skill lists, help and error text under a field.
- **Meta** (400/500, 12px / `--text-xs`): Timestamps, department lines, tenure, captions, the demonstration strip.
- **Label** (600, 11px / `--text-2xs`, `0.025em`, uppercase): Section overlines ("On staff today", "Your colleagues"), the Agent/Person chip, table heads, badges.

### Named Rules

**The One Family Rule.** Onest sets every word on both sides of the sign-in. A
second typeface — including a display face for the marketing page — is a
defect, not a variation.

**The Fixed-Scale Rule.** Operate never goes fluid. The nine scale steps
(`--text-2xs` … `--text-3xl`, ratio ≈1.15) are the only sizes; the product side
never uses `clamp()`, `vw` units or a per-screen size. Exactly one arbitrary
size exists in the build, on the marketing h1, and it is deliberate.

**The Tabular Rule.** `table` and any `[data-numeric]` element gets
`font-variant-numeric: tabular-nums` from base styles, so counts, tenures,
prices and tool results line up down a column without a per-site class.

**The Lowercase Wordmark Rule.** The name is `cachicamas`, always lowercase,
never title-cased — in copy, in a `<title>`, in a route head, anywhere.
`src/__tests__/wordmark.spec.ts` fails the build on `Cachicamas` in any shipped
source file (the `X-Cachicamas-*` HTTP headers are the one carve-out; they are a
wire contract, not the wordmark).

## Layout

**The workspace shell** (`components/workspace/workspace.tsx`) is a persistent
left rail plus a white content column. The rail is `16rem` (256px), always
present from `lg` up, `border-r border-line`, and it never moves. Below `lg` it
becomes a `17rem` drawer behind a labelled **Menu** button — a rail that shrinks
to icons is a rail nobody can read; a rail that disappears is one you can get
back. The shell owns the scroll (`overflow-y-auto`) except for screens that pass
`fills`, which is only the conversation, so its composer can stay pinned.

Above the content, in the shell rather than in any screen, sits the standing
demonstration strip: a `brand-tint` band with a shield icon and one sentence.
It is part of the shell precisely so no screen can forget to render it.

**Screen wells.** Every non-conversation screen sits in `PAGE_WELL`:
`mx-auto w-full max-w-5xl px-4 py-7 sm:px-6 lg:px-8`. Every one of them opens
with `<PageHeader>` — one `h1`, one optional sentence capped at `62ch`, and the
screen-level action on the right. Same place, same size, every time; surprise is
what a person in a task least wants.

**Marketing bands.** `mx-auto w-full max-w-6xl px-5 sm:px-8`, `py-16`–`py-20`
(`lg:py-24` on the hero), alternating `bg-surface` and `bg-canvas` with a
`border-line` between them, and one `bg-deep` band to close. Section anchors
carry `scroll-mt-8`.

**Rhythm.** Tailwind's 4px scale, concentrated on `gap-2` (8px) between controls,
`gap-3` (12px) between an avatar and its facts, `p-3`/`p-4` (12/16px) inside a
card, `p-5` (20px) inside a fieldset, `space-y-0.5` (2px) between rail rows.

**Breakpoints.** Only three are in real use: `sm` (640px, 21 uses), `md` (768px,
5 uses) and `lg` (1024px, 24 uses — the shell's rail/drawer switch and every
multi-column split). `xl` appears twice.

**Measure.** Prose is capped in characters, not pixels: `16ch`–`20ch` on
headlines, `52ch`–`54ch` on marketing body and section ledes, `62ch` on a page
lede, `64ch`–`70ch` on long-form answers. The conversation column is `max-w-2xl`.

### Named Rules

**The Base Column Rule.** Every grid declares `grid-cols-1` before any
responsive columns. A bare `grid` in Tailwind is one *implicit* column sized to
`max-content`, so a grid whose columns only appear at `sm:` and up cannot shrink
below its widest child and will push a phone layout sideways. Twelve `grid-cols-1`
declarations in the build exist for this reason.

**The Contained Scroller Rule.** A horizontal scroll container must also be
`relative`. An absolutely positioned child — `sr-only` implements visually
hidden text with `position: absolute` — is only clipped by a scroller that is
*also its containing block*, and a static wrapper is not. The pricing table's
hidden "Included" / "Not included" cell labels escaped to the table's full 672px
and dragged the whole document into a 220px sideways scroll on a 390px viewport
while the table itself sat perfectly still. Nothing looked wrong; no box
overflowed; `overflow-x: clip` on the parent did not help. The fix is one word,
so the guard is one grep: `src/__tests__/no-escaping-scrollers.spec.ts` fails any
`overflow-x-auto` class list without `relative` in it.

**The Rail-of-Five Rule.** The workspace rail carries exactly five destinations
— Front desk, Chat, Agents, Teams, Organisation. Past about seven, a rail stops
being scannable and starts being a menu you read.

## Elevation & Depth

The system is **near-flat with two reserved lifts**. Separation at rest is a
value step (`canvas` → `surface`, or `surface` → `sunken`) plus a 1px `line` —
never a shadow. `--shadow-raised` is a hairline that keeps a white card from
dissolving into a white band; it is not depth. Real elevation is spent only on
things that genuinely float.

### Shadow Vocabulary

- **Raised** (`box-shadow: 0 1px 2px rgba(16, 23, 36, 0.06)`): The resting lift on a card, a button, an input, a fieldset, a transcript panel. 11 sites. It should read as a seam, not a shadow.
- **Float** (`box-shadow: 0 8px 24px -6px rgba(16, 23, 36, 0.12), 0 2px 6px -2px rgba(16, 23, 36, 0.06)`): Menus, dialogs, the mobile navigation drawer, and a colleague card under the pointer (`hover:shadow-[var(--shadow-float)]`, `transition-shadow duration-150`). 7 sites.

### Named Rules

**The Offset Rule.** Every shadow in the system carries a vertical offset and a
soft blur. A zero-offset halo is decoration; a hard offset shadow belongs to a
different world entirely. Neither exists in this build and neither may be added.

**The Floats-Only Rule.** Elevation answers "is this thing above the page?" —
menus, dialogs, drawers, and the card you are pointing at. It never answers
"is this thing important?" That question is answered by position, size and ink.

## Shapes

Three radii, and a corner scale that grows with the box.

- **6px** (`{rounded.sm}`): Menu rows, chips and badges, the focus ring, the small dashed "Add a colleague" tile, and any inline anchor that needs a focusable box.
- **8px** (`{rounded.md}`): The default. Buttons, inputs, cards, nav rows, avatars, the company tile, transcript panels, fieldsets.
- **12px** (`{rounded.lg}`): Things with a lot inside them — the hero proof card, the composer field, marketing cards, dialogs.
- **Full** (`{rounded.full}`): Person avatars, status dots, the scrollbar thumb. Nothing else is a pill; there are no pill-shaped buttons in this system.

**Lines.** Every border in the product is 1px. The only wider stroke is the
2px `blockquote` rule inside sanitised model output. Icons are the exception to
the "no strokes" quiet: 1.75px, by their own grid.

**Icons** (`components/icon/icon.tsx`). Sixteen paths, drawn here rather than
pulled from a library, all to **one grid**: a 24×24 box, 1.75px strokes, round
caps and joins, no fills, rendered at 14/16/18/20/24px (18 by default). A set
assembled from several sources is the tell that nobody decided. Every icon is
`aria-hidden` with `focusable="false"`, because every icon ships beside a word
and the word is what is announced.

### Named Rules

**The Species Rule.** *A person is a circle. An agent is a rounded square.* This
is the only rule in the system that carries meaning through form, which is
exactly why it is the one rule that always ships the literal word beside it —
`<SpeciesLabel>` prints "Agent" or "Person", and a card prints "Agent" under the
name. The agent's radius scales with the box so the silhouette stays constant:
5px at 24px, 6px at 32px, 7px at 40px, 10px at 56px. `avatar.spec.tsx` pins both
halves — the shapes must differ, and the word must be available. A circle for an
agent, or a rounded square without its word, are both defects.

**The One Grid Rule.** A new icon is drawn into the existing 24×24 / 1.75px /
round-cap grid or it does not ship. No second icon set, no icon font, no glyph
standing in for a word.

## Components

### Buttons

`components/ui/button/classes.ts` — a pure className table so the affordances
can be unit-tested with no DOM, and so Tailwind's scanner sees literal strings.
The consumer's class is *appended* to the system tokens; it never replaces them.

- **Shape:** 8px radius (`{rounded.md}`), 1px border on every variant (transparent where there is a fill), `font-medium`, `whitespace-nowrap`, `gap-1.5` to an icon.
- **Sizes:** `sm` 28px tall / 10px side padding / 12px text (toolbars, table rows, inline actions); `md` 36px / 14px / 14px (the default); `lg` 44px / 20px / 14px (a screen's single committing action, and every marketing CTA).
- **Primary:** brand fill, white ink, `raised`. Hover *and* active both go to `brand-press`.
- **Secondary:** white fill, `line-control` border, ink text, `raised`. Hover and active both go to `sunken`. This is the default for anything non-committal, including a card's own action.
- **Destructive:** stop fill, white ink. The label always names the destruction in words.
- **Link:** brand, underlined at `2px` offset, no border, no height, no padding — a word in a sentence, not a control. `hover:text-brand-press`.
- **Disabled:** `opacity-45` and `cursor-not-allowed`, uniform across variants.
- **Transition:** `background-color, color, border-color, box-shadow` over 150ms on `--ease-work`.

**The Press-Darkens Rule.** Press darkens; nothing moves. No translate, no
scale, no lift, on any control in the system. A control that shifts under the
pointer is a control you have to re-aim at.

**The One Focus Rule.** Focus belongs to `global.css` and to nothing else: a
2px `brand` outline at 2px offset with a 6px radius, applied via `:focus-visible`
for the entire product. No component restyles it, so a focused control looks
identical on a white card, in a tinted row, and on a coloured fill.

### Menu Items

A separate primitive from `<Button>` on purpose. A menu row is full width,
left-aligned, `6px` radius, `8px×6px` padding, `ink-mid` text, and it *tints its
row* on hover (`bg-sunken`, `text-ink`) rather than filling with brand — a
dropped panel that flashed blue under every pointer move would be unreadable.
Reaching for `<Button size="sm">` inside a panel brings the wrong padding, the
wrong alignment and the wrong hover.

### Inputs / Fields

`components/ui/form/classes.ts` — one table, so every form in the product looks
like the same form.

- **Style:** white fill, **`line-control` border**, 8px radius, `12px×8px` padding, 14px ink, `raised`.
- **Hover:** border steps to `ink-soft`. **Focus:** the system ring; nothing restyled per field.
- **Disabled:** `sunken` fill, `opacity-70`, `cursor-not-allowed`.
- **Label:** 14px `font-medium` `ink`, block. **Help:** 13px `ink-soft` at `mt-1`. **Error:** 13px `font-medium` `stop` at `mt-1`.
- **Fieldset:** `line` border (decorative here — the fields inside carry their own findable borders), 8px radius, `p-5`, `raised`.

**The Findable Border Rule.** Inputs, selects, checkboxes and secondary buttons
use `line-control`, which clears 3:1 on all three grounds, because on this
surface *the border is the control* (WCAG 1.4.11). The softer `line` is for
decorative separation only. Getting this backwards is the most likely way to
break the system without anything looking wrong.

### Cards / Containers

- **Corner:** 8px in the product (`rounded-md`), 12px on marketing and on the hero proof.
- **Background:** `surface`, on a `canvas` page. **Border:** 1px `line`. **Shadow:** `raised` at rest; `float` on hover only where the card is itself a link to a colleague.
- **Padding:** 16px (`p-4`) for a colleague card, 12px (`p-3`) for a transcript panel.
- **Internal rule:** a card's footer is separated by `border-t border-line` with `pt-3`, and carries the status on the left and one `secondary` action on the right.

**The Facts-Not-Furniture Rule.** A card is not a container for a heading and an
icon. Every field on it is a fact about a specific thing, and two cards side by
side must read differently. A colleague card carries exactly the five things a
person wants before deciding to talk to someone, in the order the questions
arrive: who they are, what they do, whether they are available, how long they
have been here, and one way to start.

### Navigation

- **Workspace rail** (`components/workspace/sidebar/sidebar.tsx`): three bands — the company (initials tile in `ink`, name, plan and headcount), five destinations, then the colleagues currently on staff with a direct link into a conversation, closing with the signed-in person's own dropdown above a `border-t`.
- **Row:** `gap-2.5`, 8px radius, `8px×6px` padding, 14px. At rest `ink-mid` with `hover:bg-sunken hover:text-ink`. Current is `bg-brand-tint font-medium text-brand` plus `aria-current="page"` — selection is the only thing in the rail allowed to use the accent.
- **Current section is a prop, not a router read**, so a highlight and a URL cannot disagree.
- **Marketing header:** not sticky. `h-16`, white on a `border-b`, wordmark left, three anchors, actions right. Responsive hiding lives on a *wrapper* span, never on the `<Button>` itself — a `hidden` passed through collides with the variant's own `inline-flex` and Tailwind 4 emits them in a fixed order in which the variant wins.

### Status

A dot and its word, never a dot alone. An 8px `rounded-full` dot at
`top-px`, `gap-1.5`, then a 12px `font-medium` word in the matching ink, with an
optional ` · detail` in `ink-soft`. Three states, three tokens: working (`ok`,
and the only element in the product that pulses), training (`waiting`),
available (`idle` — deliberately the neutral value, because nothing is wrong).

### Transcript (signature component)

`components/chat/transcript-line.tsx` — five kinds, and the differences between
them are **structural rather than decorative**, because the point of showing a
colleague's work is that a person can tell at a glance which part was *speech*
and which was an *action*.

- **said** — avatar, name, words. A person's text is rendered plain at `prose`; a colleague's goes through the sanitiser into `.prose-work`. A line still being written carries `work-caret`.
- **tool** — inset by `pl-11` (aligning under the speaker's words, not their face) into a `raised` white panel: which tool, why, a `dt/dd` grid of arguments, and what came back above a `border-t`.
- **hold** — the same panel in the waiting treatment (`border-waiting/35`, `bg-waiting/[0.06]`), the risk in words, and two real buttons: `primary` "Allow it", `secondary` "Don't". Once answered it reverts to the plain panel and states the outcome in words.
- **fault** — `border-stop/30`, `bg-stop/[0.05]`, the code, the message, the recovery, and the literal line "No automatic retry."
- **note** — the conversation narrating itself: a 1px rule, a short label, a 1px rule.

Every state prints a word, because a colour alone never tells anyone whether
something happened. Argument values are `relative overflow-x-auto` — see The
Contained Scroller Rule.

### Composer

A `border-line-control` field at 12px radius inside a `border-t` bar, capped at
`max-w-2xl`, growing to the message up to 160px, Enter to send and Shift+Enter
for a newline. While a colleague is working the field disables and the button
becomes a `destructive` **Stop** — a colleague you cannot interrupt is the
failure the product exists to avoid, so Stop is never more than one key away.
The hint line under it is 12px `ink-soft` and states what will not happen.

### Hero Proof (signature component)

The public page's proof, playing. It runs the *same* state machine
(`useMockTurn`) and the *same* `TranscriptLine` as the workspace conversation,
so the marketing page cannot drift into showing something the product does not
do. A 900ms beat lets the headline be read first, then the answer arrives as it
is written and the work halts on a permission and waits. The card has a
`min-h-[19rem]` floor so it does not grow under the reader, an `aria-live="polite"`
list, and a footer line saying it is an example.

Under `prefers-reduced-motion: reduce` it **settles** — the finished exchange is
present on first paint with no clock — rather than animating at zero duration.

## Do's and Don'ts

### Do:

- **Do** give every carrier of meaning a word: an avatar its name, a status dot its state, a department hue its department, an icon its label. This is the system's hard floor (PRODUCT.md UX-4).
- **Do** print "Agent" or "Person" wherever an avatar appears outside its own full profile. The shape rule and the word ship together or not at all.
- **Do** use `line-control` for anything whose border is how a person finds it, and `line` only to separate.
- **Do** declare `grid-cols-1` on every grid before any responsive columns.
- **Do** put `relative` on every `overflow-x-auto` container.
- **Do** reach for the token scale: nine type steps, three radii, two shadows, one easing. Anything outside them needs a reason that survives a test.
- **Do** keep one committing (`primary`) action per screen; everything else is `secondary`.
- **Do** open every workspace screen with `<PageHeader>` inside `PAGE_WELL`.
- **Do** derive a shared count from one function. Two places independently counting the company's people produced three separate regressions in one review batch; `rosterFor` / `holdersFor` in `lib/mock/staff.ts` is now the single answer, and the rail, the front desk and the Organisation panel all read it.
- **Do** state demonstration material as demonstration material, in the shell, where no screen can forget it.

### Don't:

- **Don't** let colour, shape, an icon or a glyph be the only carrier of anything.
- **Don't** put anything about how the product is built on a customer-facing surface — no architecture, no layers, no protocols, no framework, no runtime figures, no codes, no terminal. This is enforced by test, not by taste.
- **Don't** title-case the wordmark. It is `cachicamas`.
- **Don't** add a second typeface, a gradient, a hard offset shadow, a zero-offset halo, or a glow. None exist in this build.
- **Don't** use a department hue to mean a status, a rank, a priority or an order.
- **Don't** use `--color-line` on a control's boundary, or `--color-line-control` as a decorative rule.
- **Don't** animate anything new. The build has exactly one authored moment (the hero proof), plus a caret on a line still being written, a pulse on a dot for an agent actually working, and a 180ms drawer. Nothing else on any surface moves on its own — no page-load choreography, no staged reveals, no scroll-triggered anything. Product loads into a task.
- **Don't** move a control on press. Darken it.
- **Don't** restyle focus per component.
- **Don't** reach for `<Button size="sm">` inside a floating panel; that is what `<MenuItem>` is for.
- **Don't** use `bg-deep` anywhere but the public page's closing band. There is one dark surface in this product.
- **Don't** measure a new colour against white and ship it. Add it to `contrast.spec.ts` and let all three grounds decide.
- **Don't** add customer logos, testimonials, review counts, uptime figures or usage statistics. None are real yet, and credibility is the only thing this product currently has.

## Known divergences

Recorded rather than filed, because a documentation pass that finds a defect
and files it instead of naming it has chosen the cheaper half of its job.

**Onest loads from a third party, and the fallback is silent.** `src/root.tsx`
links Onest from `fonts.googleapis.com` rather than self-hosting it. A tightened
CSP, a blocked host or an offline client drops the entire product onto
`ui-sans-serif` — the platform face — with nothing failing loudly, and the one
family that carries every word on both sides of the sign-in becomes whatever the
OS supplies. This is an accepted risk for this build rather than an oversight;
self-hosting is the fix when it stops being acceptable.

**The workspace's data is a mockup, and the interface says so rather than
hiding it.** `lib/mock/{staff,plans,chat,company}.ts` are demonstration
material; `PRODUCT.md § Evidence on Hand` carries the list of what a real launch
must replace. The design consequence is deliberate: the demonstration strip
lives in `<Workspace>` rather than on any screen, so no screen can forget it,
and the pricing section states in words that its figures are a preview. Neither
may be removed before the data behind them is real.
