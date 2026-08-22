# The interaction primitives

Two components live here. Everything else that looks like a component in this
app is either furniture (`~/components/workspace`), an icon
(`~/components/icon`), or a screen.

- **`<Button>`** — a control you press. Four intents (primary, secondary,
  destructive, link), three sizes (sm, md, lg), and two element kinds (`button`
  and `a`). Its className table lives in `button/classes.ts` so the affordances
  can be tested as a pure function.
- **`<MenuItem>`** — a row you pick, inside a floating panel. Separate from
  `<Button>` on purpose: reaching for `<Button size="sm">` inside a menu brings
  the wrong padding, the wrong alignment and the wrong hover.

## The material rules

Every surface in this product obeys these. They are asserted in
`button/classes.spec.ts` and `menu-item/classes.spec.ts`, so breaking one fails
a test rather than a screenshot.

1. **One family, one accent.** Onest sets every word. Brand blue carries
   primary actions, the current selection, and links — never decoration.
2. **White on a cool-neutral page.** Separation is a value step plus a 1px
   line. Elevation is reserved for things that genuinely float: menus, dialogs,
   the composer.
3. **A control's border must be findable.** Inputs, selects and secondary
   buttons use `--color-line-control`, which clears 3:1 on every ground in the
   product, because on this surface the border IS the control (WCAG 1.4.11).
   The softer `--color-line` is for decorative separation only.
4. **Press darkens; nothing moves.** No translate, no scale, no lift. A control
   that shifts under the pointer is a control you have to re-aim at.
5. **Focus belongs to `global.css`.** One treatment for the whole product — a
   2px brand ring at 2px offset. No component restyles it.
6. **Contrast is measured on every ground.** A token picked against white
   quietly fails inside a hovered row. Body text clears 4.5:1 on the surface,
   the page and the sunken well alike.
7. **Department colour identifies; it never ranks and never means a status.**
   And it never travels alone — the department name is always on the same
   screen.
8. **A person is a circle. An agent is a rounded square.** The only rule in the
   system that carries meaning through form, which is why it always ships the
   literal word beside it (`avatar.spec.tsx` pins both halves).
