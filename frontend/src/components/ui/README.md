# `@cachicamas/frontend` — UI design system

This folder is the home of the **frontend UI design system**. The first slice — buttons — is implemented here. Subsequent slices (inputs, cards, modals, navigation) will follow the same shape.

## Scope

Buttons and menu items only. Anything not listed below is out of scope:

- Inputs, textareas, selects — follow-up slice.
- Cards, panels, modals — follow-up slice.
- Navigation chrome (header, breadcrumb, tabs) — follow-up slice.
- Dark mode — the project is locked to a white surface; no dark tokens yet.
- Loading animations beyond the simple `loading` flag on `<Button>`.

## The three button intents

| Intent              | When to use                                                                                   | Visual                                                                               |
| ------------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| `primary` (default) | Create, update, save — actions that **mutate the database**.                                  | `bg-slate-900` filled button with white text, indigo focus ring, slate-700 on hover. |
| `secondary`         | General-purpose actions (Cancel, secondary navigation, "open" links).                         | White button with a slate border, slate-50 on hover.                                 |
| `destructive`       | Delete, remove, sign-out-of-account. **Not** for sign-out confirmations (those stay neutral). | `bg-red-700` filled button with white text, red focus ring, red-800 on hover.        |

The color rules are the project's monochrome rule. The `primary` intent is **always** `bg-slate-900`. Do not introduce tinted variants (no `bg-indigo-600`, `bg-emerald-600`, etc.) without an ADR — every existing primary CTA is slate-900 and any drift is a bug.

## The two sizes

| Size           | Tailwind classes      | Use                                                      |
| -------------- | --------------------- | -------------------------------------------------------- |
| `md` (default) | `px-4 py-2 text-sm`   | Form submits, header CTAs, retry buttons, panel actions. |
| `lg`           | `px-5 py-3 text-base` | Hero CTAs, empty-state CTAs.                             |

There is no `sm` size in this slice. None of the current call sites use `text-xs`, so adding a third size would be speculative.

## Affordances (guaranteed by the primitive)

Every `<Button>` carries these tokens regardless of variant:

- `cursor-pointer` (cross-OS — some OSes don't apply pointer to `<button>` natively).
- `disabled:cursor-not-allowed` and `disabled:opacity-50`.
- `transition-[background-color,box-shadow,transform,border-color] duration-150` — the same hover transition everywhere.
- `focus:outline-none focus-visible:ring-2` — focus visibility is non-negotiable for keyboard users.
- `active:translate-y-px` — uniform press feedback (except `destructive` — see below).
- `not-disabled:hover:*` — disabled buttons do **not** respond visually to mouse-over. This fixes the existing project bug where disabled buttons looked interactive.

The one carve-out:

- `destructive` does **not** carry `active:translate-y-px`. A heavy destructive button should not "press" — the visual weight signals permanence, and a press feels like the page is shaking.

## `loading` state

`<Button loading={true}>` becomes disabled and gets `aria-busy="true"`. The consumer is expected to pass alternate content via children (e.g. `<Button loading>Saving…</Button>`).

There is **no** built-in spinner. The project follows UX-4 (aphantasia-friendly, text-first) — a spinner icon would be a decorative affordance that requires mental visualization.

## `as="a"` (link polymorphism)

`<Button as="a" href="/foo">` renders an `<a>` element with the same className as the equivalent button. The `disabled` and `loading` props are type-narrowed away in this case — links cannot be disabled in HTML; consumers handle that via route guards.

## `<MenuItem>` (separate primitive)

For affordances that sit inside dropdown panels (avatar dropdown, org-pill panel, pickers), use `<MenuItem>` — not `<Button>` with a small size. The two primitives have:

|           | `<Button>`                                  | `<MenuItem>`                    |
| --------- | ------------------------------------------- | ------------------------------- |
| Layout    | `inline-flex` (button shape)                | `block w-full` (full panel row) |
| Padding   | `px-4 py-2` (or `px-5 py-3` for lg)         | `px-2 py-1.5`                   |
| Hover     | surface swap (slate-700, slate-50, red-800) | panel-row tint (`bg-slate-100`) |
| Alignment | center                                      | left                            |
| Use for   | CTAs                                        | in-page actions inside panels   |

A separate primitive prevents the temptation to abuse a hypothetical `size="sm"` Button inside panels — which would produce the wrong padding and the wrong hover color.

## Carve-outs (NOT consuming the design system)

These elements have a unique shape that does not match the three intents above. They stay as their own components with their own class strings:

| Element                                                                                    | Why it is a carve-out                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `components/sign-in-button/sign-in-button.tsx`                                             | Brand-anchored CTA (zinc-900 + GitHub Octocat SVG). UX-4 amendment 2026-07-04 lets the brand mark carry the provider identification. The dark zinc surface is intentional. **Not** a destructive intent (signing in is not destroying). |
| `components/avatar-dropdown/avatar-dropdown.tsx` (avatar trigger)                          | Circular (`h-10 w-10 rounded-full`) with `active:scale-95` press feedback, unique among buttons. The `UAT-7` test pins this exact affordance.                                                                                           |
| `components/org-pill/org-pill.tsx` empty-state pill (`cursor-default`)                     | Visually-disabled passive display; not an interactive button.                                                                                                                                                                           |
| `components/org-pill/org-pill.tsx` settings-disabled row (`role="menuitem" aria-disabled`) | Future-deferred affordance; rendered as a `<span>` to stay non-interactive.                                                                                                                                                             |
| `components/example/example.tsx`                                                           | Qwik starter demo, not user-visible.                                                                                                                                                                                                    |

PR-5 introduces an ESLint rule (`no-inline-button-class`) that flags inline class strings in any file that imports the primitives. The carve-outs above have explicit allowlist entries.

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

// Menu row inside a dropdown panel
<MenuItem onClick$={openProfile}>Profile</MenuItem>

// Menu row disabled (visually disabled but still selectable for SR announcements)
<MenuItem disabled={true}>Settings (coming soon)</MenuItem>
```

## Anti-drift guardrails

1. The pure-function unit tests in `button/classes.spec.ts` and `menu-item/classes.spec.ts` pin every token. Removing `cursor-pointer` or any other affordance breaks the tests immediately.
2. PR-5 adds an ESLint rule that flags inline class strings matching the variant markers in any file that imports the primitives. See the `eslint-rules/no-inline-button-class.mjs` file (added in PR-5).
3. The `link` variant is **not** in this slice. PR-3 introduces it; do not backport it into PR-1 review unless the design is updated.

## Reference

- Proposal: `openspec/changes/cachicamas-button-design-system/proposal.md`
- Spec: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
- Design: `openspec/changes/cachicamas-button-design-system/design.md`
- Tasks: `openspec/changes/cachicamas-button-design-system/tasks.md`
