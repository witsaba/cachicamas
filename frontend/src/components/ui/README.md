# `@cachicamas/frontend` — UI design system

This folder is the home of the **frontend UI design system**. The first slice — buttons — is implemented here. Subsequent slices (inputs, cards, modals, navigation) will follow the same shape.

## Scope

Buttons and menu items only. Anything not listed below is out of scope:

- Inputs, textareas, selects — follow-up slice.
- Cards, panels, modals — follow-up slice.
- Navigation chrome (header, breadcrumb, tabs) — follow-up slice.
- Dark mode — the project is locked to a white surface; no dark tokens yet.
- Loading animations beyond the simple `loading` flag on `<Button>`.

## The four button intents

| Intent              | When to use                                                           | Visual                                                                               |
| ------------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `primary` (default) | Create, update, save — actions that **mutate the database**.          | `bg-slate-900` filled button with white text, indigo focus ring, slate-700 on hover. |
| `secondary`         | General-purpose actions (Cancel, secondary navigation, "open" links). | White button with a slate border, slate-50 on hover.                                 |
| `destructive`       | Delete, remove, sign-out-of-account.                                  | `bg-red-700` filled button with white text, red focus ring, red-800 on hover.        |
| `link`              | Bare-underline text button (clear selection, refresh, etc.).          | No surface; text color change on hover.                                              |

The color rules are the project's monochrome rule. The `primary` intent is **always** `bg-slate-900`. Do not introduce tinted variants (no `bg-indigo-600`, `bg-emerald-600`, etc.) without an ADR — every existing primary CTA is slate-900 and any drift is a bug.

## The two sizes

| Size           | Tailwind classes      | Use                                                      |
| -------------- | --------------------- | -------------------------------------------------------- |
| `md` (default) | `px-4 py-2 text-sm`   | Form submits, header CTAs, retry buttons, panel actions. |
| `lg`           | `px-5 py-3 text-base` | Hero CTAs, empty-state CTAs.                             |

There is no `sm` size in this slice.

## Affordances (guaranteed by the primitive)

Every `<Button>` carries these tokens regardless of variant:

- `cursor-pointer` (cross-OS — some OSes don't apply pointer to `<button>` natively).
- `disabled:cursor-not-allowed` and `disabled:opacity-50`.
- `transition-[background-color,box-shadow,transform,border-color] duration-150` — the same hover transition everywhere.
- `focus:outline-none focus-visible:ring-2` — focus visibility is non-negotiable for keyboard users.
- `active:translate-y-px` — uniform press feedback (except `destructive` — see below).
- `not-disabled:hover:*` — disabled buttons do **not** respond visually to mouse-over.

The one carve-out:

- `destructive` does **not** carry `active:translate-y-px`. A heavy destructive button should not "press" — the visual weight signals permanence.

The `link` variant uses `transition-colors` (a lighter transition than the filled variants) because the hover state is a text color change, not a surface swap.

## Consumer personalizations (the `class` prop)

Every `<Button>` and `<MenuItem>` accepts an optional `class` prop. The system tokens always apply first; consumer tokens are **appended**. Use this for personalizations the variants don't anticipate:

- **Shape**: `class="h-10 w-10 rounded-full"` for circular triggers (avatar dropdown).
- **Color override**: `class="border border-zinc-700 bg-zinc-900 ..."` for the brand-anchored zinc dark CTA (SignInButton, SignOut confirmation).
- **Dark-surface hover**: `class="hover:bg-slate-700"` for chips on a slate-900 background (the picker "Clear" button).
- **Padding override**: `class="px-3 py-1"` for tighter button sizes inside dense layouts.
- **Outline destructive**: `class="border border-red-300 bg-transparent text-red-700 hover:bg-red-50"` for the destructive-outline variant (one call site today; the solid variant is the default).

The design system does not encode these as named variants because each is a one-off consumer concern. Promoting them to variants would inflate the API for marginal gain.

### The `!important` escape hatch

Tailwind 4 emits utilities in alphabetical order in the generated CSS. The class attribute order does **not** affect specificity — only the CSS emission order does. When a consumer override class **conflicts** with a variant token (same property name, different value), the order of the two class strings in the HTML attribute does not decide which wins; the order in the generated CSS does.

For most conflicts (e.g. `bg-zinc-900` vs `bg-slate-900`), the alphabetical emission order happens to favor the override because `bg-zinc-900` > `bg-slate-900`. But for **three specific cases**, the override silently loses without help:

1. **`rounded-md` (variant) vs `rounded-full` (override)**: `'f' < 'm'` alphabetically → `rounded-md` emitted later → `rounded-md` wins → square corner instead of circle. **Affected**: the avatar trigger (`<Button variant="primary" class="h-10 w-10 …">`).
2. **`not-disabled:hover:bg-slate-700` (variant) vs `hover:bg-zinc-800` (override)**: the variant compiles to `:not(:disabled):hover`, which has CSS specificity `(0, 3, 0)` (class + :not(:disabled) pseudo + :hover pseudo). The bare override compiles to `:hover` only, specificity `(0, 2, 0)`. **Higher specificity always wins** regardless of emission order. **Affected**: any override on a primary/secondary/destructive button that wants to change the hover color (the SignInButton zinc CTA, the destructive-outline delete button, the SignOut confirm).
3. **`px-4 py-2` (from `BUTTON_SIZE_MD`) vs a shape-driven override like `h-10 w-10`**: Tailwind's default `box-sizing` is `border-box`, so `h-10 w-10 px-4 py-2` is a 40×40 box with the content area shrunk to `40 − 32 = 8` wide and `40 − 16 = 24` tall. A child with `h-full w-full` becomes an 8×24 vertical strip inside the 40×40 circular clip. **Affected**: the avatar trigger (a shape-driven element, not a size-driven one). Fix: `!p-0` neutralizes the size padding.

The escape hatch is the `!important` prefix — Tailwind 4's standard pattern for "this utility MUST win over the cascade":

```tsx
// Avatar trigger — `!rounded-full` wins over `rounded-md` from BUTTON_BASE.
// `!p-0` neutralizes the size's `px-4 py-2` so the content area
// fills the full 40×40 box (otherwise the image renders as an 8×24 strip).
// `active:!scale-95` wins over `active:translate-y-px` from the variant.
// Tailwind 4 syntax: `!` goes AFTER the variant (`active:!scale-95`),
// not before (`!active:scale-95` — silently ignored by Tailwind).
<Button variant="primary" class="!rounded-full !bg-transparent !p-0 h-10 w-10 overflow-hidden ring-1 ring-slate-200 hover:shadow-md hover:ring-slate-400 active:!scale-95">
  <img src={avatar} alt="" />
</Button>

// Brand-anchored zinc CTA — every conflicting token needs `!`.
<Button variant="primary" class="!bg-zinc-900 !text-zinc-100 border border-zinc-700 shadow-sm !hover:border-zinc-600 !hover:bg-zinc-800 hover:shadow-md !focus-visible:ring-zinc-500">
  Sign in
</Button>

// Outline destructive — solid red overridden with transparent + red border.
<Button variant="destructive" class="!bg-transparent !text-red-700 border border-red-300 !hover:bg-red-50 !focus-visible:ring-red-500">
  Delete workspace
</Button>
```

When **not** to use `!important`:

- Pure additions (a new utility not present in the variant): `class="px-3 py-1"` — no conflict, no need.
- Same property, same value: `class="text-sm"` (already in the variant's size) — no-op, harmless.
- Different property entirely: `class="shadow-sm"` — no conflict.

The lint rule `no-inline-button-class` strips `!` before matching, so documented `!important` overrides don't trigger drift warnings. The presence of `!` is itself the signal that the consumer knows they are overriding a system token.

## `loading` state

`<Button loading={true}>` becomes disabled and gets `aria-busy="true"`. The consumer is expected to pass alternate content via children (e.g. `<Button loading>Saving…</Button>`).

There is **no** built-in spinner. The project follows UX-4 (aphantasia-friendly, text-first) — a spinner icon would be a decorative affordance that requires mental visualization.

## `as="a"` (link polymorphism)

`<Button as="a" href="/foo">` renders an `<a>` element with the same className as the equivalent button. Standard anchor attrs (`target`, `rel`, `download`, `referrerPolicy`) are typed explicitly. The `disabled` and `loading` props are type-narrowed away in this case — links cannot be disabled in HTML.

## `<MenuItem>` (separate primitive)

For affordances that sit inside dropdown panels (avatar dropdown, org-pill panel, pickers), use `<MenuItem>` — not `<Button>` with a small size. The two primitives have:

|           | `<Button>`                                  | `<MenuItem>`                    |
| --------- | ------------------------------------------- | ------------------------------- |
| Layout    | `inline-flex` (button shape)                | `block w-full` (full panel row) |
| Padding   | `px-4 py-2` (or `px-5 py-3` for lg)         | `px-2 py-1.5`                   |
| Hover     | surface swap (slate-700, slate-50, red-800) | panel-row tint (`bg-slate-100`) |
| Alignment | center                                      | left                            |
| Use for   | CTAs                                        | in-page actions inside panels   |

`<MenuItem>` also supports `as="a"` for menu items that navigate to another route (e.g. avatar menu: Profile / Workspaces).

## `<SettingCard>` (separate primitive — settings surface tile)

For affordances that are settings surface entries — tiles in the `/settings` Launchpad-style grid, where each tile is the entry point to one settings surface (Prompts, Profile, Billing, ...). The macOS Launchpad metaphor: a centered icon-in-rounded-square with a label below, hover lift, and the same focus-visible chrome the other DS primitives use.

| SettingCard                                  | `<Button>`                          | `<MenuItem>`                  |
| -------------------------------------------- | ----------------------------------- | ----------------------------- |
| `flex-col` (vertical tile)                   | `inline-flex` (button shape)        | `block w-full` (panel row)    |
| `p-3` + `rounded-lg` (tile)                  | `px-4 py-2` + `rounded-md`          | `px-2 py-1.5`                 |
| `bg-slate-100` icon well + label below       | `bg-slate-900` / `bg-white` surface | `bg-slate-100` row tint       |
| Hover: `bg-slate-50` + group-hover text flip | surface swap                        | `bg-slate-100` row tint       |
| Use for: settings surface tiles (Launchpad)  | CTAs                                | in-page actions inside panels |

`<SettingCard>` is polymorphic via `as="a" | "button"` (default `as="a"`, matching the Launchpad's navigation semantics). The icon is passed via the `icon: JSXNode` prop and rendered inside a 64×64 slate-100 rounded-xl icon well; the label is a sibling span.

The icon container's `text-slate-700 group-hover:text-slate-900` tokens drive the icon's stroke — the consumer-provided SVG MUST use `stroke="currentColor"` (or `fill="currentColor"` for filled icons). The currentColor contract is the monochrome rule: do not hard-code colors in consumer-provided icons.

Currently one consumer (`/settings` index route → Prompts tile). Designed to grow without layout change — Profile, Billing, Notifications, etc. slot in as additional `<SettingCard>` children.

## Migration coverage

Every interactive button-like element in `@cachicamas/frontend` consumes the design system:

| File                                                             | Element                              | Primitive                                                                   |
| ---------------------------------------------------------------- | ------------------------------------ | --------------------------------------------------------------------------- |
| `routes/index.tsx`                                               | "Get started" CTA                    | `<Button variant="primary" size="lg" as="a">`                               |
| `routes/index.tsx`                                               | "See the interface" CTA              | `<Button variant="secondary" size="lg" as="a">`                             |
| `routes/workspaces/index.tsx`                                    | Create workspace CTAs (×2)           | `<Button variant="primary" as="a">`                                         |
| `routes/workspaces/index.tsx`                                    | Retry                                | `<Button variant="primary">`                                                |
| `routes/workspaces/[id]/index.tsx`                               | Delete workspace (outline)           | `<Button variant="destructive" class="…outline override…">`                 |
| `routes/workspaces/[id]/index.tsx`                               | Delete confirm (solid)               | `<Button variant="destructive">`                                            |
| `routes/workspaces/[id]/index.tsx`                               | Cancel                               | `<Button variant="secondary">`                                              |
| `routes/home/index.tsx`                                          | (uses HomeWorkspacesSection)         | —                                                                           |
| `components/home-workspaces-section/home-workspaces-section.tsx` | Create workspace CTAs (×2)           | `<Button variant="primary" as="a">`                                         |
| `components/home-workspaces-section/home-workspaces-section.tsx` | Retry                                | `<Button variant="primary">`                                                |
| `components/workspace-card/workspace-card.tsx`                   | Open link                            | `<Button variant="primary" as="a">`                                         |
| `components/profile-view/profile-view.tsx`                       | github.com/{login}                   | `<Button variant="secondary" size="lg" as="a">`                             |
| `components/organization-form/organization-form.tsx`             | Submit (create organization)         | `<Button variant="primary" type="submit">`                                  |
| `components/organization-form/organization-form.tsx`             | Add optional details                 | `<Button variant="secondary">`                                              |
| `components/ownboarding-form/ownboarding-form.tsx`               | Submit (create organization)         | `<Button variant="primary" type="submit">`                                  |
| `components/workspace-form/workspace-form.tsx`                   | Submit (create workspace)            | `<Button variant="primary" type="submit">`                                  |
| `components/workspace-form/workspace-form.tsx`                   | Clear selection                      | `<Button variant="link">`                                                   |
| `components/github-repo-picker/github-repo-picker.tsx`           | Clear (chip on dark surface)         | `<MenuItem class="hover:bg-slate-700">`                                     |
| `components/github-repo-picker/github-repo-picker.tsx`           | Refresh repos                        | `<Button variant="link">`                                                   |
| `components/github-repo-picker/github-repo-picker.tsx`           | Picker option row                    | `<MenuItem>`                                                                |
| `components/github-repo-picker/github-repo-picker.tsx`           | Load more                            | `<Button variant="secondary">`                                              |
| `components/avatar-dropdown/avatar-dropdown.tsx`                 | Avatar trigger (circular)            | `<Button variant="primary" class="h-10 w-10 rounded-full active:scale-95">` |
| `components/avatar-dropdown/avatar-dropdown.tsx`                 | Profile link (menu row)              | `<MenuItem as="a">`                                                         |
| `components/avatar-dropdown/avatar-dropdown.tsx`                 | Workspaces link (menu row)           | `<MenuItem as="a">`                                                         |
| `components/avatar-dropdown/avatar-dropdown.tsx`                 | Sign out (menu row submit)           | `<MenuItem type="submit">`                                                  |
| `components/org-pill/org-pill.tsx`                               | Interactive trigger                  | `<Button variant="secondary">`                                              |
| `components/sign-in-button/sign-in-button.tsx`                   | Sign in (zinc dark + GitHub Octocat) | `<Button variant="primary" class="…zinc override…">`                        |
| `routes/auth/signout/index.tsx`                                  | Sign out confirm (zinc dark)         | `<Button variant="primary" class="…zinc override…">`                        |
| `routes/auth/signout/index.tsx`                                  | Cancel                               | `<Button variant="secondary" as="a">`                                       |

## Legitimate carve-outs (NOT consuming the primitives)

These are NOT buttons or button-like affordances. They are passive visual elements that happen to look button-shaped:

| Element                                                                                                                      | Why it is a carve-out                                                                                                                                 |
| ---------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `components/org-pill/org-pill.tsx` loading `<div>`                                                                           | Status display while fetching the current org. Not interactive.                                                                                       |
| `components/org-pill/org-pill.tsx` empty-state `<div>` ("No organization yet")                                               | Status display when the user has no org yet. Not interactive.                                                                                         |
| `components/org-pill/org-pill.tsx` settings-disabled `<span role="menuitem" aria-disabled>`                                  | Placeholder for a future settings link. `aria-disabled` semantics require a non-button element.                                                       |
| `components/example/example.tsx`                                                                                             | Qwik starter demo, not user-visible.                                                                                                                  |
| Inline chips / monograms with `bg-slate-900 text-white` (e.g. the "Selected" pill in `github-repo-picker`, the org monogram) | Not button-shaped — they sit inside other elements and have no click affordance. The lint rule skips them by checking only `<button>` and `<a>` tags. |

The `sign-in-button/sign-in-button.tsx` file is in the lint rule's allowlist because the zinc-anchored dark CTA is a documented brand override (the SignInButton's GitHub Octocat SVG is the UX-4 visual anchor; the dark zinc surface is intentional). The `<Button>` wrapper still consumes the design system — the override is what makes it a brand CTA rather than the default primary.

## Anti-drift guardrail (ESLint rule)

`frontend/eslint-rules/no-inline-button-class.mjs` is a custom ESLint rule that runs as part of `pnpm lint`. It flags inline className strings on `<button>` or `<a>` JSX elements that contain the design-system variant markers (primary/secondary/destructive), in any file that imports from `~/components/ui/button` or `~/components/ui/menu-item`. The rule is set to `"warn"` so a regression surfaces in CI without breaking the build.

Files in the allowlist (`sign-in-button.tsx`) are not checked — their override is documented above. Dynamic class strings (``class={`...${variant}...`}``) are out of scope for v1; the rule uses string `includes()` on the literal source.

## Quick reference

```tsx
import { Button, MenuItem } from "~/components/ui";

// Primary CTA — "Save", "Create", "Update"
<Button onClick$={save} loading={saving}>Save</Button>

// Primary CTA as a link to another route (no disabled, no loading)
<Button as="a" href="/workspaces/new" size="lg">Create workspace</Button>

// Secondary outline — "Cancel"
<Button as="a" href="/" variant="secondary">Cancel</Button>

// Destructive solid — "Delete workspace" in a confirmation dialog
<Button variant="destructive" onClick$={onConfirmDelete}>Delete</Button>

// Destructive outline — "Delete workspace" trigger
<Button variant="destructive" onClick$={onShowConfirm}
  class="border border-red-300 bg-transparent text-red-700 hover:bg-red-50">
  Delete workspace
</Button>

// Bare-underline text button — "Clear selection", "Refresh repos"
<Button variant="link" onClick$={onClear$}>Clear selection</Button>

// Circular avatar trigger (custom shape)
<Button variant="primary" onClick$={toggleMenu}
  class="h-10 w-10 overflow-hidden rounded-full active:scale-95">
  <img src={avatar} alt="" />
</Button>

// Brand-anchored zinc dark CTA (SignInButton, SignOut confirm)
<Button variant="primary" type="submit"
  class="border border-zinc-700 bg-zinc-900 text-zinc-100 shadow-sm hover:border-zinc-600 hover:bg-zinc-800 hover:shadow-md focus-visible:ring-zinc-500">
  Sign out
</Button>

// Menu row inside a dropdown panel
<MenuItem onClick$={openProfile}>Profile</MenuItem>

// Menu row as a link
<MenuItem as="a" href="/workspaces">Workspaces</MenuItem>

// Menu row disabled (visually disabled but still selectable for SR announcements)
<MenuItem disabled={true}>Settings (coming soon)</MenuItem>
```

## Reference

- Proposal: `openspec/changes/cachicamas-button-design-system/proposal.md`
- Spec: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
- Design: `openspec/changes/cachicamas-button-design-system/design.md`
- Tasks: `openspec/changes/cachicamas-button-design-system/tasks.md`
- Explore: `openspec/changes/cachicamas-button-design-system/explore.md`
