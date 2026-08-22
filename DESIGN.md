---
name: cachicamas
description: A dealing-room terminal for a company of specialist agents — neutral structure, colour spent only on what is happening.
colors:
  void: "#070a10"
  panel: "#161e2b"
  raise: "#202a3b"
  rule: "#55637a"
  rule-strong: "#6f7f99"
  rule-faint: "#2b3547"
  amber: "#ffb020"
  amber-dim: "#7a5714"
  cyan: "#56c8f0"
  live: "#5fd97f"
  hold: "#b18cff"
  fail: "#ff5f7a"
  fg: "#e6ecf5"
  fg-mid: "#9fb0c7"
  fg-dim: "#7f8ea8"
typography:
  board:
    fontFamily: "Spline Sans Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace"
    fontSize: "2.5rem"
    fontWeight: 600
    lineHeight: 0.95
    letterSpacing: "-0.015em"
  screen:
    fontFamily: "Spline Sans Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace"
    fontSize: "1.75rem"
    fontWeight: 600
    lineHeight: 1
    letterSpacing: "-0.015em"
  lead:
    fontFamily: "Spline Sans, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.625
    letterSpacing: "normal"
  body:
    fontFamily: "Spline Sans, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.625
    letterSpacing: "normal"
  data:
    fontFamily: "Spline Sans Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.375
    letterSpacing: "normal"
  label:
    fontFamily: "Spline Sans Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.25
    letterSpacing: "0.14em"
  legend:
    fontFamily: "Spline Sans Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace"
    fontSize: "0.6875rem"
    fontWeight: 400
    lineHeight: 1.25
    letterSpacing: "0.14em"
rounded:
  none: "0"
spacing:
  hairline: "1px"
  tight: "6px"
  snug: "8px"
  cell: "12px"
  screen: "16px"
  section: "32px"
components:
  button-primary:
    backgroundColor: "{colors.amber}"
    textColor: "{colors.void}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "6px 12px"
  button-primary-hover:
    backgroundColor: "{colors.void}"
    textColor: "{colors.amber}"
  button-primary-lg:
    backgroundColor: "{colors.amber}"
    textColor: "{colors.void}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
    padding: "10px 16px"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.fg}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "6px 12px"
  button-secondary-hover:
    backgroundColor: "transparent"
    textColor: "{colors.amber}"
  button-destructive:
    backgroundColor: "{colors.fail}"
    textColor: "{colors.void}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "6px 12px"
  button-destructive-hover:
    backgroundColor: "{colors.void}"
    textColor: "{colors.fail}"
  button-link:
    backgroundColor: "transparent"
    textColor: "{colors.cyan}"
    rounded: "{rounded.none}"
    padding: "0"
  button-link-hover:
    textColor: "{colors.fg}"
  menu-item:
    backgroundColor: "transparent"
    textColor: "{colors.fg-mid}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "8px 12px"
    width: "100%"
  menu-item-hover:
    backgroundColor: "{colors.raise}"
    textColor: "{colors.amber}"
  panel:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.fg}"
    rounded: "{rounded.none}"
  panel-header:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.fg}"
    typography: "{typography.label}"
    padding: "6px 12px"
  panel-header-note:
    textColor: "{colors.fg-dim}"
    typography: "{typography.legend}"
  panel-body:
    backgroundColor: "{colors.panel}"
    padding: "12px"
  form-input:
    backgroundColor: "{colors.raise}"
    textColor: "{colors.fg}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
    padding: "8px 10px"
    width: "100%"
  form-label:
    textColor: "{colors.fg-dim}"
    typography: "{typography.legend}"
  state-lamp-mark:
    rounded: "{rounded.none}"
    height: "6px"
    width: "6px"
  gauge-segment-complete:
    backgroundColor: "{colors.fg}"
    rounded: "{rounded.none}"
    height: "8px"
    width: "6px"
  gauge-segment-partial:
    backgroundColor: "{colors.fg-mid}"
    rounded: "{rounded.none}"
    height: "8px"
    width: "6px"
  gauge-segment-empty:
    backgroundColor: "{colors.rule}"
    rounded: "{rounded.none}"
    height: "8px"
    width: "6px"
  register-cell:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.fg-mid}"
    rounded: "{rounded.none}"
    padding: "12px"
  register-cell-hover:
    backgroundColor: "{colors.raise}"
  screen-title-chip:
    backgroundColor: "transparent"
    textColor: "{colors.fg-dim}"
    typography: "{typography.legend}"
    rounded: "{rounded.none}"
    padding: "1px 6px"
  dock-cell:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.fg-mid}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "6px 12px"
  dock-cell-current:
    backgroundColor: "{colors.fg}"
    textColor: "{colors.void}"
  command-line:
    backgroundColor: "{colors.void}"
    textColor: "{colors.fg}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
    padding: "6px 12px"
---

# Design System: cachicamas

## Overview

**Creative North Star: "The Exchange Terminal"**

This is a dealing-room board, not an app. A company's specialist agents are services on a
board — each with a code, a state and a number, all readable at once — and the interface's
whole job is to let one person hold the state of the whole company in a single glance.
Deep ink-navy panels sit on a void ground, separated by 1px hairlines and by a value step
rather than by shadow. Nothing is rounded, nothing glows, nothing floats. Density is high
and deliberate: a screen is composed by tiling panels the way a trading board is composed
by tiling readouts.

The system refuses the launcher grid of identical icons, and the refusal is load-bearing.
Six specialists are on the register; five of them do not exist. A grid of six identical
tiles would say the opposite. So every cell on the board carries the evidence for its own
claim — a lamp, a status word, a plan reference, a gauge with the literal `done/total`
beside it — and a specialist that has never run says so in words on its own screen.

The hardest rule here is that **structure is neutral**. Chrome — panel labels, key legends,
gauges, hairlines, title chips, the current dock cell — carries no colour at all. Colour is
reserved for what is happening: a run in flight, a suspension waiting on a person, a
failure, a link you can follow, the system speaking about itself right now. An earlier
build gave panel headings, dock keys, gauges and the active dock cell the accent colour,
and the result was a near-black surface with one neon hue everywhere — the exact category
default this world exists to refuse. A palette spent on structure is a palette that can no
longer report anything.

Type follows the same discipline. There are two faces and the switch between them is
meaning: Spline Sans Mono is the machine — codes, labels, states, tabular figures,
headings. Spline Sans is language — what a person typed and what a model answered.

**Key Characteristics:**

- Hard rectangles everywhere; `border-radius` is globally zeroed with `!important`
- Flat by construction: no shadow, no blur, no gradient, no backdrop filter
- Neutral structure: chrome is greyscale, colour is reserved for activity
- A three-step value ladder (`void` → `panel` → `raise`) plus contrast-floored hairlines
- Four activity colours (amber, green, violet, red) plus cyan for "you can go here"; the
  two inactive lamp states are neutral
- Two type faces, one fixed non-fluid seven-step scale, tabular figures by default
- Every lamp is followed by its literal status word — no meaning by colour or mark alone
- One focus treatment for the whole system; one transition duration (150ms)
- Almost no motion: a blinking caret, a pulsing lamp, and colour transitions

## Colors

A near-black void, two ink-navy surfaces above it, two contrast-floored hairlines, three
text greys, and five saturated colours that are spent only on activity and navigation.
Every value is measured against `--color-panel`.

### Primary

- **Terminal Amber** (`{colors.amber}`, 9.3:1 on panel): the system speaking about itself
  **right now**. Its permitted list is short and closed — the wordmark, the command-line
  and composer prompt characters, the streaming caret, the focus ring and the focused-field
  border, the primary action, the demonstration marker, and the single `build` lamp for the
  one archetype in flight. It is explicitly **not** used for structure.
- **Amber Dim** (`{colors.amber-dim}`): the hairline companion to amber. Its one use is the
  border of the status rail's "Demo data" marker.

### Secondary

- **Navigable Cyan** (`{colors.cyan}`, 8.8:1 on panel): "you can go here", and nothing else.
  Plan references, the authority reference, the `link` button variant, and anchors inside
  sanitised model output. A plan reference is cyan because it opens, not because of what it
  says — cyan is never a state.

### Tertiary

The activity colours. They describe what a run is doing.

- **Running Green** (`{colors.live}`, 10.4:1): a run in progress, a connection open, an
  archetype on duty, a tool call executing. The only colour permitted to pulse.
- **Suspended Violet** (`{colors.hold}`, 7.2:1): a run suspended waiting on a person's
  decision. It borders the permission block, names it, and states the risk.
- **Refused Red** (`{colors.fail}`, 6.4:1): a typed error, a denied permission, a cancelled
  turn, a destructive action. The `destructive` button, the command line's refusal, the
  fault block.

### Neutral

- **Void** (`{colors.void}`): the dark of the room the board hangs in. The document ground,
  the command-line band, and the text inside any filled cell.
- **Panel** (`{colors.panel}`): a panel's own surface, one value step up from the void. Also
  the status rail and the dock.
- **Raise** (`{colors.raise}`): a well inside a panel — an input, a selected row, a code
  block, a tool block, a hovered menu item or dock cell.
- **Rule** (`{colors.rule}`, 3.1:1 on panel): every hairline that carries structure — panel
  borders, header dividers, cell separators, an unlit gauge segment. Never thicker than 1px,
  and never fainter than this.
- **Rule Strong** (`{colors.rule-strong}`, 4.6:1 on panel): a focused, active or interactive
  edge — a dropped panel's border, a hovered cell, the `secondary` button's border, the
  `ScreenTitle` code chip, the scrollbar thumb.
- **Rule Faint** (`{colors.rule-faint}`): **ornament only**, and deliberately below the
  boundary floor. Its one use is the dotted leader in `Field`, which guides the eye across a
  row and carries no structure, so it must not compete with the rules that do.
- **Foreground** (`{colors.fg}`): primary data, what a person typed, a panel's own label, a
  complete gauge, and the current dock cell's fill.
- **Foreground Mid** (`{colors.fg-mid}`): what a model answered, secondary data, a partial
  gauge, and the `ready` lamp.
- **Foreground Dim** (`{colors.fg-dim}`, 5.65:1): chrome labels, units, legends, key legends,
  the `idle` lamp, and anything explicitly unavailable. This is the contrast floor for text
  in this system.

### The state vocabulary

Six lamp tones, shared by every screen, defined once in `StateLamp`. Only a state that is
**happening** gets a colour; "planned" and "unplanned" describe an absence of activity, so
they are two brightnesses of neutral and do not compete with the three states a person
actually has to notice.

| Tone    | Colour           | Means                                     |
| ------- | ---------------- | ----------------------------------------- |
| `live`  | green            | Running now, connected, on duty           |
| `build` | amber            | Has a plan and work in flight             |
| `hold`  | violet           | Suspended, waiting on a person's decision |
| `fail`  | red              | Errored, denied, cancelled                |
| `ready` | neutral (fg-mid) | Planned, but nothing is happening         |
| `idle`  | neutral (fg-dim) | Nothing here yet                          |

### Named Rules

**The One Job Rule.** Each working colour has exactly one job: amber = the system speaking
about itself now, cyan = you can go here, green = running, violet = suspended, red = failed.
A colour used for mood is a colour that has stopped carrying information. Adding a sixth is a
decision-record-level change, not a personalization.

**The Neutral Structure Rule.** Chrome carries no colour. Panel labels, key legends, gauges,
hairlines, title chips and the current dock cell are greyscale. If a mark is describing the
interface rather than reporting an event, it is neutral. This rule is what keeps the four
activity colours legible on a dense board.

**The Word Beside the Lamp Rule.** No meaning is ever carried by colour, mark or position
alone (PRODUCT.md § Accessibility, UX-4). `StateLamp` requires its `word` prop — the status
word is not a caption, it is half the component. A gauge draws segments *and* prints
`done/total`. The dock is words, not icons. Demonstration data is marked with the word
"demo".

**The Boundary Floor Rule.** A hairline that carries structure clears 3.1:1 against the
surface behind it (WCAG 1.4.11). In a board built entirely out of hairlines that floor is not
a detail, it is the whole structure. `rule-faint` is the single exemption, and only because
it is ornament: it guides the eye and carries nothing.

## Typography

**Machine Font:** Spline Sans Mono (ui-monospace, SFMono-Regular, Menlo, Consolas, Liberation Mono, monospace)
**Human Font:** Spline Sans (ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica Neue, Arial, sans-serif)

Both are loaded from Google Fonts at weights 400/500/600, preconnected in `src/root.tsx`.
The document's default face is the machine font at `data` size with
`font-variant-numeric: tabular-nums` set on `body`, so a column of counts never shifts as it
updates.

**Character:** A terminal that has learned to write prose. The mono face gives the board its
alignment, its uppercase tracked labels and its unshifting figures; the sans face appears only
when something is being *said* — a role description, a model's answer, a person's message, an
explanatory note.

### Hierarchy

- **Board** (600, 2.5rem/40px, line-height 0.95, tight tracking): the landing headline. One
  per document, and the only place this step appears.
- **Screen** (600, 1.75rem/28px, line-height 1, tight tracking): the `ScreenTitle` heading at
  the head of an application.
- **Lead** (400, 1rem/16px, relaxed): a panel's lead paragraph and the sentence that carries a
  screen's argument. Human face. Measure capped around 68–74ch.
- **Body** (400, 0.875rem/14px, relaxed or snug): prose, role descriptions, transcript
  language, inputs. Human face. Measure capped at 58–80ch where it runs long.
- **Data** (400, 0.8125rem/13px, snug): table cells, log lines, field values, secondary notes.
  Machine face by default; human face when the content is language.
- **Label** (400, 0.75rem/12px, tracking 0.08–0.16em, uppercase): panel labels, archetype
  codes, status words on chips, button labels, menu items, dock names. Machine face.
- **Legend** (400, 0.6875rem/11px, tracking 0.10–0.22em, uppercase): F-key legends, units,
  column heads, lamp words, gutter speaker labels, the clock, disclosure lines, the wordmark.
  Machine face.

All seven declared steps are used; there is no unused type token.

### Named Rules

**The Two Voices Rule.** `font-system` is the machine — codes, labels, states, tabular
figures, headings. `font-human` is language — what a person typed and what a model answered. A
third face, or a face used for contrast rather than for role, breaks the only typographic
signal this system has.

**The Fixed Scale Rule.** The ramp is fixed, never fluid. No `clamp()`, no viewport units in
type. A screen that needs a size the ramp does not have is a screen that has invented a
hierarchy level. The one exception in the build is the landing headline, which steps up to
`3.25rem` at the `sm` breakpoint; treat that as the documented exception, not as licence.

**The Tracked Uppercase Rule.** Machine voice is uppercase and letter-spaced; human voice is
sentence case and untracked. Uppercase without tracking, or tracked human prose, reads as a
mistake in this world.

**The Chip Earns Its Place Rule.** `ScreenTitle` suppresses its code chip when the code and
the title are the same word, case-insensitively. A chip that repeats the heading beneath it is
an eyebrow with no payload.

## Layout

The product is an operating-system shell with exactly one scrolling region. Four bands, in
fixed order, top to bottom:

1. **StatusRail** — wordmark, organization, demo marker, live clock, identity. Always present,
   identical on every screen.
2. **CommandLine** — the launcher. Rendered only when signed in.
3. **The application** — a `<Slot/>` inside `flex min-h-0 flex-1 flex-col overflow-y-auto`.
   This is the only box that scrolls; the chrome never scrolls away.
4. **FunctionRail** — the dock, `sticky bottom-0`, one cell per destination, F-keys live.
   Rendered only when signed in.

Applications set their own container width: `max-w-[1800px]` for board-like screens (the desk,
the chat app, the landing), `max-w-[1100px]` for reading screens (an archetype panel, the
system panel), `max-w-2xl` for a single message. Horizontal padding is 12px stepping to 16px at
`sm`; vertical padding is 16px.

**Spacing rhythm.** Bands use `px-3 py-1.5` (12px/6px). Panel bodies use `p-3` (12px). Grid
gutters between panels are `gap-3` (12px). Wells inside a panel use `px-2.5 py-2` (10px/8px).
The landing's thesis block is the only place a 32px gap appears.

**Grids are hairline grids.** Tiled cells sit on a `bg-rule` grid with `gap-px`, so the 1px
background shows through as the rule between cells. There is no border-collapse problem and no
double rule. The landing register uses this at 1 / 2 / 3 columns; the desk board's register
uses 1 / 2 inside a 3–4 column page grid.

**Responsive behavior** is by disclosure, not by reflow tricks. Below `sm` the status rail
drops the organization, the demo marker and the clock. Below `lg` the chat app drops the
conversation list panel entirely. The dock scrolls horizontally as one strip rather than
wrapping. Nothing collapses into a hamburger, because the command line already is the
non-pointing route to everything.

## Elevation & Depth

**There is no elevation.** No `box-shadow`, no `filter: blur`, no gradient, no
`backdrop-filter` exists anywhere in the built system, and the primitives' unit tests fail on a
`shadow-*` or `ring-*` token. Depth is a value ladder and a hairline: `void` is the room,
`panel` is a board hanging in it, `raise` is a well cut into a board. A dropped panel (the
identity menu, the command-line suggestions) is separated from what it covers only by
`z-index` and by a `rule-strong` border, which is enough because the value step is already
doing the work.

Alpha is used sparingly and only inside a signal block: `border-hold/40` and `border-fail/40`
divide the header of a permission or fault block from its body without introducing a second
border colour. `disabled:opacity-40` dims an unavailable control. The lamp pulse animates
opacity to 0.35 and back. That is the whole use of transparency.

### Named Rules

**The Flat Rule.** Surfaces are flat at rest and flat under interaction. Hover changes colour,
never geometry — nothing travels, nothing scales, nothing lifts.

**The Hairline Rule.** Every rule is exactly 1px, in `rule` or `rule-strong`, never in a
working colour used decoratively. A 2px border or a double rule is a defect.

## Shapes

**Nothing is rounded.** `global.css` sets `border-radius: 0 !important` on `*`, `*::before` and
`*::after` in the base layer, so an accidental `rounded-*` utility is a no-op rather than a
visible break in the world. No Tailwind radius utility is used anywhere in the codebase.

The system's whole geometry is the rectangle and the 1px line:

- **The cell** — a bordered rectangle with an uppercase machine label. Buttons, register cells,
  code chips, F-key chips, the `kbd` hint.
- **The panel** — a rectangle with a 1px rule and a header band divided by another 1px rule.
- **The well** — a `raise` rectangle inside a panel: inputs, tool blocks, selected rows.
- **The lamp** — a 6px filled square. Never a circle, never a dot.
- **The gauge** — eight 6×8px segments with a 1px gap, filled left to right.
- **The leader** — a `border-b border-dotted` in `rule-faint`, running between a `Field`'s
  label and its value so the eye can cross a wide panel.

### Named Rules

**The No Radius Rule.** A radius anywhere is a defect, not a variation. This includes avatars,
which are square.

## Components

### Buttons

A button is a keyed cell on a board, not a pill. Uppercase machine label, `tracking-[0.08em]`,
1px border, hard corners. Buttons are one of the few places colour is still permitted on a
control, because a committing action *is* the system offering to act.

- **Shape:** hard rectangle (0 radius), `border` 1px, `inline-flex` with `gap-2`.
- **Sizes:** `md` (default) at `px-3 py-1.5` + label size — toolbar cells, panel actions, form
  submits. `lg` at `px-4 py-2.5` + body size — a screen's single committing action. There is
  no `sm`; density below `md` belongs to `MenuItem` or to a bare element.
- **Primary:** amber fill, void text, amber border. The system's own action — submit, open,
  confirm.
- **Secondary:** transparent fill, `rule-strong` border, `fg` text. Anything non-committal.
  Hover warms the border and the label to amber.
- **Destructive:** fail fill, void text, fail border. Refuse, delete, cancel a run.
- **Link:** cyan, underlined, `underline-offset-2`, no cell, no padding, no uppercase. A word
  in a sentence. Hover goes to `fg`.
- **Hover / Active:** filled variants **reverse video** — fill and text swap. Secondary darkens
  to `raise` on press. All under `not-disabled:`, so a disabled control does not respond to
  mouse-over. Transition is `transition-[background-color,color,border-color] duration-150`.
- **Disabled:** `opacity-40` and `cursor-not-allowed`. `loading` is sugar for disabled plus
  `aria-busy="true"`; there is no spinner glyph — the consumer changes the label.
- **Consumer overrides** append to the system tokens and need `!` to win, because the variants
  compile to `:not(:disabled):hover` at specificity (0,3,0). For a variant utility the `!` goes
  *after* the variant: `hover:!border-amber`, never `!hover:border-amber`.

### Menu Items

Rows inside a dropped panel, deliberately not a small button.

- **Style:** full-width, left-aligned, `px-3 py-2`, label size, uppercase, tracked, `fg-mid`
  text.
- **Hover:** tints the row (`raise` background, amber text) rather than reversing a cell. A
  dropped panel that flashed amber on every pointer move would be unreadable.

### Panel

The only container in the system. There is no card, no tile, no second container shape.

- **Corner Style:** hard rectangle, 0 radius.
- **Background:** `panel`, with a 1px `rule` border.
- **Header:** a band divided by a bottom 1px rule, `px-3 py-1.5`, carrying an uppercase
  **neutral** (`fg`) label on the left and an optional `fg-dim` `legend`-size note on the
  right. The note is where a count, a state or a reference goes — **never a control**.
- **Internal Padding:** `p-3`, removable via `padded={false}` when the body supplies its own
  (tables, lists, transcripts).
- **Heading level:** `h2` or `h3`, so a screen's panels form a real document outline.

### Inputs / Fields

Form chrome lives in shared constants (`os/form-classes.ts`), not a component — there are two
forms in the whole product.

- **Style:** `raise` well, 1px `rule` border, `px-2.5 py-2`, hard corners, human face at body
  size. The label above is `legend` size, uppercase, `tracking-[0.14em]`, `fg-dim`.
- **Hover:** border steps to `rule-strong`.
- **Focus:** border goes amber (`focus:border-amber`) — the system acknowledging you — *in
  addition to* the global focus outline.
- **Error:** `aria-invalid:border-fail`, plus a message in `fail` beneath. Helper text is
  human-face `data` size in `fg-dim`.
- **Bare fields:** the command line and the composer render an unstyled `<input>` /
  `<textarea>` with `border-none bg-transparent outline-none`. Their chrome is the surrounding
  band, not the field.

### Navigation

- **StatusRail:** `panel` background, bottom rule, `px-3 py-1.5`. Wordmark in amber at label
  size with `tracking-[0.22em]`, uppercase — one of amber's closed list of permitted uses.
  Organization, demo marker and a live `tabular-nums` clock follow, all `hidden sm:inline`.
  Identity sits at the right end.
- **FunctionRail (the dock):** `sticky bottom-0`, `panel` background, top rule, `gap-px`,
  horizontally scrollable as one strip. Each cell is an anchor showing its F-key in `fg-dim`
  `legend` and its short name in uppercase `label` at `fg-mid`. The current cell inverts to
  **`fg` on void** — neutral, not amber — and carries `aria-current="page"`. Unplanned
  destinations render at `fg-dim`. The dock renders anchors directly rather than consuming
  `<Button>`: its cells are a key legend, not buttons.
- **Skip link:** the first focusable element in the document, `sr-only` until focused, then a
  fixed amber-bordered cell at top-left.

### Focus

- **Treatment:** `outline: 1px solid var(--color-amber)` with `outline-offset: 1px`, applied to
  `:focus-visible` globally — a 1px amber frame with a 1px void gap, so it reads on a panel, in
  a well and on the void alike.

### Named Rules

**The Reverse Video Rule.** Press feedback is reverse video, uniformly. A filled cell swaps its
fill and its text; that is what a terminal does when you hit a key, and it is the only press
affordance in the product.

**The One Focus Rule.** Focus is not a variant's business. The system has exactly one focus
treatment, defined once globally. A primitive that restyles focus has broken the guarantee that
a focused control looks identical everywhere; the primitives' tests fail on any `ring-*` token.

### StateLamp (signature)

A state, said twice: a 6px filled square plus the state's own word, always together, in an
`inline-flex` with `gap-1.5`. The word is required by the component's own type. `pulse` is
permitted only for a state genuinely in motion (`term-lamp`, 1.6s), and in practice only
`on-duty` and a running tool call use it. `data-tone` is emitted for testing.

### Gauge (signature)

Eight segments plus the literal figure, and **no state colour at all**. Complete fills `fg` and
prints its figure in `fg`; partial fills `fg-mid` against `rule`. Complete and in-progress
differ by brightness, not by hue — a gauge reports a quantity, and a quantity is not a state;
the lamp beside it already says whether the thing is running. A partial is never rounded up to
full (41 of 42 must not read as complete), and zero of N is a legitimate, common reading that
must look like a real zero rather than a broken component.

### Field (signature)

One labelled reading: a dim uppercase `legend` label, a dotted 1px `rule-faint` leader that
flexes to fill, and the value right-aligned in `fg`. This is the atom every dense readout is
built from. A screen that needs a spec table uses a stack of these rather than inventing a
table.

### ScreenTitle (signature)

An engineering drawing's title block: a neutral code chip bordered in `rule-strong` (the
command you would have typed to get here), the screen title at `screen` size, a slot at the
right for a lamp, and an optional human-face lead beneath at `lead` size capped to 74ch. The
whole block sits above a 1px bottom rule. The chip is suppressed when it would merely repeat
the title.

### RegisterCell (signature)

One archetype's line on the board, as an anchor: F-key chip, code in `fg` (or `fg-mid` when
unreachable), lamp with its word, role in the human face, then a stack of `Field`s — Plan (cyan
reference plus a gauge), Owns, Authority, and demo turns where they exist. Hover moves the
border to `rule-strong` and, when reachable, the fill to `raise`. This is the launcher tile in
the terminal's grammar, and the evidence rows are the point: a grid of identical icons would
imply six working specialists.

### CommandLine (signature)

The launcher. An amber `>` prompt, a bare field on the `void` band, and a `kbd` chip showing
⌘K. Focuses from anywhere on ⌘K / Ctrl-K; Escape hands focus back. Suggestions drop as a
`rule-strong`-bordered `panel` list beneath, each row showing code / label / hint in three
columns, the code in neutral `fg`. An unknown code prints a refusal in `fail` beneath the field
— never a silent no-op.

### TranscriptLine (signature)

There are no chat bubbles, and their absence is a decision. A bubble column implies two peers
exchanging messages; what is happening is a run being narrated. So the transcript is a
speaker-labelled log with a fixed 56px left gutter (`legend` size, uppercase, tracked) and five
line kinds sharing it:

- **note** — a tracked uppercase label, a 1px rule flexing to fill, an optional detail.
- **said** — the gutter reads "You" in `fg` and "Chat" in amber: the person is neutral, because
  the person is not the system speaking; the archetype is amber, because it is. A person's text
  renders in `fg`; a model's renders through `renderSanitizedMarkdown` into `.prose-terminal`
  in `fg-mid`. A streaming line carries the amber `term-caret`.
- **tool** — a `raise` well with a header band naming the tool and a lamp: `live`/Running,
  `ready`/Returned, `fail`/Refused or Failed. Arguments render as a two-column `<dl>`.
- **hold** — a `hold`-bordered well: "Permission required", the intent, the exact call and its
  arguments, the risk in violet, then Allow once (primary) and Refuse (destructive).
- **fault** — a `fail`-bordered well: the error code, the message, and the recovery, with "No
  automatic retry" stated in the header.

### Sanitised model output (`.prose-terminal`)

Model prose is rendered from an allowlist and styled in `global.css` rather than by a typography
plugin. `strong` goes to `fg` at 600; links are cyan and underlined at offset 2px; `code` and
`pre` are `raise` wells with a 1px `rule`; unordered lists use `list-style: square`; blockquotes
take a `rule-strong` left rule and drop to `fg-dim`; tables collapse with 1px `rule` cells at
`data` size; and headings are flattened to uppercase tracked `label` size in the machine face
regardless of level, so a model cannot introduce a hierarchy the screen did not ask for.

## Motion

One duration and two keyframes. This is the smallest motion vocabulary the product has had, and
it shrank on purpose.

- **State change:** 150ms, on `background-color`, `color` and `border-color` only. Nothing else
  transitions.
- **`term-caret`** (1s, `steps(1, end)`, infinite): a 0.5em block caret in `currentColor`,
  blinking. Only on a line that is still arriving, and the command line's own idiom.
- **`term-lamp`** (1.6s, `cubic-bezier(0.2, 0, 0, 1)`, infinite): opacity 1 → 0.35 → 1. Only on
  a lamp whose state is genuinely in motion.
- **No entrance animation.** The staggered register paint was removed rather than tuned: this
  interface's first-viewport promise is that you can see the whole company at once, and an
  entrance that reveals it over half a second is a promise the animation itself eats. The board
  arrives painted.
- **`prefers-reduced-motion: reduce`** collapses animation duration, iteration count and
  transition duration to near zero **and pins `animation-delay: 0s`**. The delay guard matters
  as much as the duration: a staggered entrance with fill mode `both` would otherwise leave each
  element at opacity 0 for its own delay, so the surface would arrive *invisible* for the people
  who asked not to see it move.

### Named Rules

**The Only Moving Thing Rule.** The one thing moving on any screen is the one thing actually
happening. There is no entrance animation, no parallax, no scroll-linked effect, and no easing
variety.

## Do's and Don'ts

### Do:

- **Do** keep structure neutral. Panel labels, key legends, gauges, hairlines, title chips and
  the current dock cell are greyscale; colour is for what is happening.
- **Do** compose every screen from `Panel`. It is the only container; tile panels rather than
  inventing a card, a tile or a second surface shape.
- **Do** pair every lamp with its literal status word, every gauge with its `done/total` figure,
  and every dock cell with a word. `StateLamp` enforces this in its type — keep it that way.
- **Do** keep amber to its closed list: the wordmark, the prompt characters, the streaming
  caret, focus, the primary action, the demo marker, and the `build` lamp.
- **Do** use cyan only for something that opens. A reference is cyan because it is navigable,
  never because of what it says.
- **Do** clear 3.1:1 against the surface for any hairline that carries structure; use
  `rule-faint` only for ornament that carries nothing.
- **Do** switch to `font-human` the moment content is language — a role description, a model's
  answer, a person's message — and back to `font-system` for everything the machine says about
  itself.
- **Do** build dense readouts from stacked `Field`s with their dotted leaders rather than
  authoring a table.
- **Do** tile cells on a `bg-rule` grid with `gap-px` so the background shows through as the
  rule.
- **Do** mark demonstration data with the word "demo" in the copy, and say plainly which figures
  are read from the repository and which are invented.
- **Do** append consumer overrides with `!` *after* the variant (`hover:!border-amber`) when
  they must beat a `not-disabled:hover:*` variant token.
- **Do** refuse out loud. An unrecognised command prints what it does accept; a denied tool call
  says nothing ran.

### Don't:

- **Don't** put a working colour on chrome. A coloured panel label, key legend, gauge or title
  chip is the specific regression this palette was corrected to remove.
- **Don't** colour a state that is not happening. `ready` and `idle` are neutral; giving
  "planned" a hue makes it compete with the three states a person must actually notice.
- **Don't** introduce a radius. `border-radius: 0 !important` is global; a `rounded-*` utility
  is dead code and a rounded custom element is a defect.
- **Don't** add a shadow, a glow, a blur, a gradient or a `backdrop-filter`. The primitives'
  tests fail on `shadow-*` and `ring-*`; the world has no light source.
- **Don't** add a sixth working colour, or a tinted button variant. `bg-emerald-600` as a
  primary is a bug, not a personalization; a returning `slate-*` or `indigo-*` is a regression
  the class tests catch.
- **Don't** use `rule-faint` for anything that separates two regions. It is below the boundary
  floor by design.
- **Don't** restyle focus per component. There is one focus treatment, defined globally.
- **Don't** move, scale or lift anything on hover or press. Reverse video and colour change are
  the entire interaction vocabulary.
- **Don't** add an entrance animation, or any staged reveal of the board.
- **Don't** use a fluid or `clamp()` type size, or invent a step outside the seven-step ramp.
- **Don't** put a control in a `Panel`'s header note. It is a readout — a count, a state, a
  reference.
- **Don't** render a chat bubble, an avatar circle, or a glyph icon standing in for a word.
  Every destination in this product is already a word.
- **Don't** reach for `<Button size="xs">` inside a dropped panel. That is what `MenuItem`
  exists for, and the two primitives' tests assert they never converge.
- **Don't** inject model output as raw HTML. It goes through `renderSanitizedMarkdown` into
  `.prose-terminal`, which is the only thing standing between a model and the DOM.

## Divergences recorded from the build

Where the direction contract, the in-repo prose and the shipped code disagree, the code is what
is recorded above. The open disagreements, for the record:

- **The direction contract in `src/root.tsx` is now behind the palette.** Its OWN-WORLD block
  still reads "five working colours, each with one job: amber is the machine speaking, cyan is
  navigable, green running, violet suspended, red failed." The built system narrowed amber to
  *the system speaking about itself right now, never structure*, and moved two of the six lamp
  tones (`ready`, `idle`) off colour entirely. The contract text was not updated with the fix.
- **The contract's FIRST VIEWPORT describes the signed-in desk**, not the built `/`. The landing
  renders the status rail, a headline and a "What is actually built" panel, then the register,
  then the suspension. The command line and the dock are gated on `isAuthenticated` in
  `routes/layout.tsx` and do not render for a signed-out visitor.
- **`src/components/ui/README.md` rule 3 is stale.** It still states the pre-fix rule ("Colour
  is state, never decoration… amber (the machine speaking)") and does not carry the
  structure-is-neutral clause that `global.css` rule 3 now states. `global.css` is the
  authority; the README lags it.
- **Structural amber survives at five sites** the neutralization did not reach, all outside the
  `os/` primitives. They are recorded as residual defects, not as design-system rules:
  `routes/index.tsx` (the landing's layer-code chips), `os/form-classes.ts` `FORM_LEGEND`
  (fieldset legends), `sign-in-required-card.tsx` (a panel-style heading, where `Panel`'s own
  label is now neutral), `routes/auth/signin/index.tsx` and `routes/auth/signout/index.tsx` (a
  full-width `bg-amber h-px` rule at the top of the page). A sixth, borderline: the conversation
  list's active-row marker is amber, which is selection state rather than the system speaking.
- **A stale comment on the auth pages** calls that 1px amber rule a "subtle gradient accent
  line". There is no gradient in the build — it is a flat 1px fill — so the comment misdescribes
  its own code rather than violating the no-glow rule.

Resolved since the previous recording, and no longer divergences: `PRODUCT.md` exists at the
repository root and is the source of the accessibility commitment recorded above;
`button.tsx`'s header comment has been rewritten to the terminal vocabulary and now matches
`classes.ts`; and `--text-panel` has been deleted, so every declared type step is used.
