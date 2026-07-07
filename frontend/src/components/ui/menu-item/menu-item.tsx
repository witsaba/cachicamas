/**
 * MenuItem primitive — the row affordance inside dropdown panels
 * (avatar menu, org-pill panel, future pickers).
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *   R-UB-006 — MenuItem primitive renders panel-row affordances.
 *
 * Why not `<Button variant="...">` with a small size:
 *   - Menu items are block, not flex (full panel width).
 *   - Hover is a panel-row tint (bg-slate-100), not a button surface swap.
 *   - Padding is `px-2 py-1.5`, not `px-4 py-2`.
 *   - Text is left-aligned, not centered.
 *
 * Promoting this to its own primitive keeps the Button mental model
 * clean ("this is a CTA, not a menu row") and prevents the temptation
 * to abuse a hypothetical `size="sm"` Button inside panels.
 *
 * The component always renders a `<button>` (never an `<a>`) because
 * menu items are in-page actions (sign out, switch view, etc.). Links
 * to other routes are not rendered as menu items.
 */
import { Slot, component$, type QwikIntrinsicElements } from "@builder.io/qwik";
import { MENU_ITEM_BASE } from "./classes";

export type MenuItemProps = {
  /** When true, the button is non-interactive and styled as disabled. */
  disabled?: boolean;
  /** Default "button"; can be overridden to "submit" or "reset". */
  type?: "button" | "submit" | "reset";
  /** Optional data-testid override. */
  testId?: string;
} & Omit<
  QwikIntrinsicElements["button"],
  "class" | "type" | "disabled" | "ref"
>;

export const MenuItem = component$<MenuItemProps>((props) => {
  const { testId, type, ...rest } = props;
  // The `disabled` field is owned by the primitive (typed via the prop
  // union above). We forward it as the native attribute.
  const disabled = props.disabled === true;
  return (
    <button
      {...rest}
      type={type ?? "button"}
      class={MENU_ITEM_BASE}
      disabled={disabled}
      data-testid={testId}
    >
      <Slot />
    </button>
  );
});
