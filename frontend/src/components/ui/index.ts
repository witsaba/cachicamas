/**
 * Design-system barrel — the interaction primitives.
 *
 * Two primitives live here, and the split between them is deliberate: a
 * `<Button>` is a cell you press, a `<MenuItem>` is a row you pick. Everything
 * structural — panels, lamps, gauges, fields, title blocks — lives in
 * `~/components/os`, because those are the operating system's furniture rather
 * than generic UI.
 *
 * ADDITIVE barrel: the deep-path imports throughout the app stay valid.
 */
export { Button } from "./button/button";
export { MenuItem } from "./menu-item/menu-item";
