/**
 * Design-system barrel — the interaction primitives.
 *
 * Two primitives live here, and the split between them is deliberate: a
 * `<Button>` is a control you press, a `<MenuItem>` is a row you pick.
 * Everything structural — the shell, the rail, avatars, status, page headers,
 * screens — lives in `~/components/workspace`, because those are this
 * product's furniture rather than generic UI. Icons live in
 * `~/components/icon`.
 *
 * ADDITIVE barrel: the deep-path imports throughout the app stay valid.
 */
export { Button } from "./button/button";
export { MenuItem } from "./menu-item/menu-item";
