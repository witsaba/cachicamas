/**
 * MenuItem primitive — the row affordance inside dropdown panels
 * (avatar menu, org-pill panel, pickers).
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
 * Polymorphism via `as`:
 *   - `as="button"` (default) — native `<button>`. Default `type="button"`.
 *   - `as="a"` — renders `<a href={href}>` for menu items that navigate
 *     to another route (e.g. avatar menu: Profile / Workspaces).
 *
 * `class` overrides are appended to the system tokens, identical to
 * the Button contract.
 */
import { Slot, component$, type QwikIntrinsicElements } from "@builder.io/qwik";
import { MENU_ITEM_BASE } from "./classes";

/** Common attrs shared by the two polymorphism cases. */
type CommonMenuItemAttrs = Omit<
  QwikIntrinsicElements["button"],
  "class" | "type" | "disabled" | "ref"
>;

export type MenuItemAsButtonProps = CommonMenuItemAttrs & {
  as?: "button";
  type?: "button" | "submit" | "reset";
  /** When true, the button is non-interactive and styled as disabled. */
  disabled?: boolean;
  /** Ignored when `as="button"`. */
  href?: never;
};

export type MenuItemAsAnchorProps = CommonMenuItemAttrs & {
  as: "a";
  href: string;
  /** Standard anchor attrs (Qwik 1.20's AnchorHTMLAttributes is empty). */
  target?: "_self" | "_blank" | "_parent" | "_top" | (string & {}) | undefined;
  rel?: string | undefined;
  /** Links cannot be disabled in HTML. */
  disabled?: never;
  /** Ignored when `as="a"`. */
  type?: never;
};

export type MenuItemProps = (MenuItemAsButtonProps | MenuItemAsAnchorProps) & {
  /** Optional data-testid override. */
  testId?: string;
  /**
   * Optional className tokens appended to the system className.
   * Use for color overrides on dark surfaces (e.g. `hover:bg-slate-700`
   * on a slate-900 picker chip).
   */
  class?: string;
};

export const MenuItem = component$<MenuItemProps>((props) => {
  const className = props.class
    ? [MENU_ITEM_BASE, props.class].join(" ")
    : MENU_ITEM_BASE;

  if (props.as === "a") {
    const { as: _as, testId, class: _c, ...rest } = props;
    void _as;
    void _c;
    const anchorProps = rest as unknown as QwikIntrinsicElements["a"];
    return (
      <a
        {...anchorProps}
        href={props.href}
        class={className}
        data-testid={testId}
      >
        <Slot />
      </a>
    );
  }

  // Button case
  const { as: _as, testId, class: _c, type, disabled, ...rest } = props;
  void _as;
  void _c;
  const isDisabled = disabled === true;
  return (
    <button
      {...rest}
      type={type ?? "button"}
      class={className}
      disabled={isDisabled}
      data-testid={testId}
    >
      <Slot />
    </button>
  );
});
