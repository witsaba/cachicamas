# `@cachicamas/frontend` — the design system

Two families live under `src/components/`:

- **`ui/`** — the interaction primitives. A `<Button>` is a cell you press; a `<MenuItem>`
  is a row you pick. That is the whole family, and the split between them is deliberate.
- **`os/`** — the operating system's furniture: `Panel`, `StateLamp`, `Gauge`, `Field`,
  `ScreenTitle`, `RegisterCell`, and the shell's three bands (`StatusRail`, `CommandLine`,
  `FunctionRail`). Every screen is composed from these.

The world both families render in is defined once, in `src/global.css`. Read that file
before adding anything here; it is short, and it is the authority.

## The six material rules

They are stated in `global.css` and enforced by `classes.spec.ts` in each primitive:

1. **Nothing is rounded.** A radius anywhere is a defect, not a variation. `global.css`
   zeroes `border-radius` globally so an accidental `rounded-*` is a no-op rather than a
   visible break.
2. **Nothing glows.** No shadows, no blurs, no gradients, no backdrop filters. Separation
   comes from 1px hairlines and from the value step between `--color-void`,
   `--color-panel` and `--color-raise`.
3. **Colour is state, never decoration — and structure is neutral.** Amber is the system
   speaking about itself _now_, and its list is closed: the wordmark, prompt characters,
   the streaming caret, focus, the primary action, the demonstration marker, and the one
   archetype in build. `cyan` is navigable; `live`, `hold` and `fail` say what a run is
   doing. Panel labels, key legends, gauges, title chips and the current dock cell carry
   **no colour** — a palette spent on chrome is a palette that can no longer report
   anything, and the first build proved it by putting twenty amber marks on one screen.
4. **State works on two axes.** Colour separates the states that are _happening_; a filled
   versus hollow mark separates the ones that are not. Five of the six specialists on the
   board are inactive, so a colour-only vocabulary put the whole state read back onto the
   word.
5. **Every rule clears 3:1 against every ground it can sit on** — void, panel AND well. A
   hairline picked against the panel alone fails silently inside a raised block, which is
   where half of them are. `--color-rule-faint` is the single declared exemption: it draws
   the dotted leader, which is ornament.
6. **Two voices.** `font-system` (Spline Sans Mono) is the machine: codes, labels, states,
   tabular figures, headings. `font-human` (Spline Sans) is language: what a person typed
   and what a model answered. The switch between them is meaning, not texture.

## The four button intents

| Intent              | When to use                                            | Visual                                                        |
| ------------------- | ------------------------------------------------------ | ------------------------------------------------------------- |
| `primary` (default) | The system's own action: submit, open, confirm.        | Amber filled cell, void text. Reverses on hover and press.    |
| `secondary`         | Anything non-committal: cancel, back, refresh, "open". | Empty cell with a rule; the rule and the label warm to amber. |
| `destructive`       | Refuse, delete, cancel a run.                          | Fail-red filled cell, void text. Reverses like the others.    |
| `link`              | A word in a sentence: an inline route somewhere.       | Cyan, underlined, no cell at all.                             |

**Press feedback is reverse video**, uniformly. A filled cell swaps its fill and its text;
that is what a terminal does when you hit a key, and it is the only press affordance in
the product. Nothing travels, nothing scales, nothing lifts.

**Do not introduce a tinted variant.** The five working colours are the palette; a
`bg-emerald-600` primary is a bug, not a personalization. Adding a sixth state colour is
an ADR-level decision, because it widens the vocabulary every screen shares.

## The two sizes

| Size           | Tailwind classes         | Use                                         |
| -------------- | ------------------------ | ------------------------------------------- |
| `md` (default) | `px-3 py-1.5 text-label` | Toolbar cells, panel actions, form submits. |
| `lg`           | `px-4 py-2.5 text-body`  | A screen's single committing action.        |

There is no `sm`. Density below `md` belongs to `<MenuItem>` or to a bare element.

## Affordances (guaranteed by the primitive)

Every `<Button>` carries these regardless of variant:

- `cursor-pointer` — some OSes do not apply it to `<button>` natively.
- `disabled:cursor-not-allowed` and `disabled:opacity-40`.
- `transition-[background-color,color,border-color] duration-150` — one duration, product
  wide.
- `not-disabled:hover:*` — a disabled control does **not** respond to mouse-over.

**Focus is not a variant's business.** `global.css` gives the whole system one focus
treatment — a 1px amber outline with a 1px offset — so a focused control looks identical
on a panel, in a well, and on the void. A primitive that restyles focus is a primitive
that has broken that guarantee, and `classes.spec.ts` fails on any `ring-*` token.

## `<MenuItem>` — why it is a separate primitive

Menu items live inside dropped panels. They are full-width, left-aligned rows, and they
**tint** on hover (`bg-raise`) rather than reversing a cell. A panel that flashed amber on
every pointer move would be unreadable, which is exactly the mistake a
`<Button size="xs">` inside a panel would make.

`menu-item/classes.spec.ts` asserts the two never converge.

## Consumer personalizations (the `class` prop)

Every primitive accepts an optional `class`. System tokens apply **first**; consumer tokens
are appended. Use it for what the variants do not anticipate:

- **Shape** — `class="h-7 w-7 !p-0"` for the avatar trigger, which is size-driven by its
  square rather than by label padding.
- **Intent override** — `class="!bg-transparent !border-rule-strong hover:!border-amber"`
  for the GitHub sign-in cell, which is a route into the product rather than a page's
  committing action.
- **Width** — `class="w-full"` for a form's submit.

### The `!important` escape hatch

Tailwind 4 emits utilities in alphabetical order, and the variants use
`not-disabled:hover:*`, which compiles to `:not(:disabled):hover` — specificity (0,3,0). A
bare `hover:*` override compiles to `:hover` — (0,2,0). **The variant wins regardless of
emission order**, so any colliding override needs `!`.

One syntax gotcha that has bitten this codebase twice: for a variant utility the `!` goes
**after** the variant.

```
hover:!border-amber   ✅  compiles, wins
!hover:border-amber   ❌  Tailwind silently drops the `!`; the override loses
```

The failure is invisible in review — the class string looks right and the button renders
with the variant's colour. `avatar-dropdown.spec.tsx` and `sign-in-button.spec.tsx` both
pin the correct form for that reason.

## Legitimate carve-outs (NOT consuming the primitives)

- **The dock** (`os/function-rail`) renders anchors directly. Its cells are a key legend,
  not buttons: they carry no border, invert wholesale when current, and scroll
  horizontally as one strip.
- **The command line and the composer** render bare `<input>` / `<textarea>`. Their chrome
  is the surrounding band, not the field.
- **Form fields** use the shared constants in `os/form-classes.ts` rather than a `<Field>`
  component. There are two forms in the whole product; a component would be a wrapper
  around one class string.

## Anti-drift guardrails

- `button/classes.spec.ts` and `menu-item/classes.spec.ts` fail on any radius, shadow,
  ring, or colour outside the palette — including a returning `slate-*` or `indigo-*`.
- `os/form-classes.spec.ts` does the same for field chrome.
- `eslint-rules/` carries the project's own rules; run `pnpm lint` before pushing.

## Quick reference

```tsx
import { Button, MenuItem } from "~/components/ui";
import { Panel, StateLamp, Gauge, Field, ScreenTitle } from "~/components/os";

<Panel label="Runtime" note="3 layers">
  <Field label="Milestones">
    <Gauge done={24} total={24} />
  </Field>
  <StateLamp tone="live" word="Frozen" />
</Panel>

<Button variant="primary" size="lg">Open the register</Button>
<Button variant="destructive" onClick$={refuse}>Refuse</Button>
<Button as="a" href="/chat/" variant="secondary">Chat</Button>
<Button variant="link">Clear selection</Button>
```

## Reference

- `src/global.css` — the world: palette, voices, scale, motion. The authority.
- `PRODUCT.md` (repo root) — who this is for, and the accessibility commitment every
  primitive here inherits: no meaning carried by colour, mark or position alone.
- `DESIGN.md` (repo root) — the built system, recorded from the shipped code.
