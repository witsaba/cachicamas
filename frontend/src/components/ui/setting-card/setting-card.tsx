/**
 * SettingCard primitive — a vertical tile for settings surface entries.
 *
 * Reference: `sdd/settings-app-grid/{proposal,spec,design}.md` (engram).
 *
 * The macOS Launchpad metaphor: a centered icon-in-rounded-square with
 * a label below, hover lift, and the same focus-visible chrome the
 * other DS primitives use. Each tile is the entry point to one
 * settings surface (Prompts, Profile, Billing, ...).
 *
 * Two element kinds (polymorphic via `as`):
 *   - `as="a"` (default) — renders `<a href={href}>`. The Launchpad
 *     is a navigation surface, so the default is a link. `target` /
 *     `rel` are typed explicitly because Qwik 1.20's
 *     `AnchorHTMLAttributes` interface is empty (the special attrs
 *     live in a `SpecialAttrs` map that the public types do not
 *     expose).
 *   - `as="button"` — native `<button>`. Default `type="button"` to
 *     avoid the classic "button inside form defaults to submit" bug.
 *     `disabled` is forwarded.
 *
 * Icon API: `icon: JSXOutput` — the consumer passes any renderable JSX
 * (typically a Qwik `<svg>` component using `stroke="currentColor"`).
 * The icon container's `text-slate-700 group-hover:text-slate-900`
 * tokens drive the icon's stroke.
 *
 * Label: `label: string` — required. Rendered as a sibling `<span>`
 * with the `LABEL` chrome (mt-3, text-sm, font-medium, ...).
 *
 * `class` override: the system tokens always apply first; consumer
 * tokens are APPENDED. Use this for personalizations the variant
 * does not anticipate. See `classes.ts` for the className table and
 * the README's `!important` escape-hatch guidance.
 */
import {
  component$,
  type JSXOutput,
  type QwikIntrinsicElements,
} from "@builder.io/qwik";
import {
  ICON_CONTAINER,
  LABEL,
  type SettingCardVariant,
  settingCardClassName,
} from "./classes";

/** Common attributes shared by the two polymorphism cases. */
type CommonSettingCardAttrs = Omit<
  QwikIntrinsicElements["button"],
  "class" | "type" | "disabled" | "ref"
>;

/** Props when rendered as an `<a>` (the default). */
export type SettingCardAsAnchorProps = CommonSettingCardAttrs & {
  as?: "a";
  /** Destination route. Required when rendered as an anchor. */
  href: string;
  /** Qwik 1.20's AnchorHTMLAttributes is empty; type anchor attrs explicitly. */
  target?: "_self" | "_blank" | "_parent" | "_top" | (string & {}) | undefined;
  rel?: string | undefined;
  /** Ignored when `as="a"` — links cannot be disabled in HTML. */
  disabled?: never;
  /** Ignored when `as="a"`. */
  type?: never;
};

/** Props when rendered as a `<button>`. */
export type SettingCardAsButtonProps = CommonSettingCardAttrs & {
  as: "button";
  type?: "button" | "submit" | "reset";
  disabled?: boolean;
  /** Ignored when `as="button"`. */
  href?: never;
};

/** Public SettingCard API. */
export type SettingCardProps = (
  | SettingCardAsAnchorProps
  | SettingCardAsButtonProps
) & {
  /**
   * Consumer-provided Qwik JSX rendered inside the icon container.
   * Expects an `<svg>` (the container's text color drives the
   * stroke via `currentColor`).
   *
   * Typed as `JSXOutput` rather than `JSXNode` because Qwik function
   * components (declared via `component$`) return `JSXOutput`, which
   * is the union `JSXNode | string | number | boolean | null |
   * undefined | JSXOutput[]` — wider than `JSXNode`. The icon prop
   * needs to accept any renderable JSX, so `JSXOutput` is the
   * right contract.
   */
  icon: JSXOutput;
  /** Visible label below the icon. Always required. */
  label: string;
  /** Optional data-testid override; rendered as `data-testid`. */
  testId?: string;
  /**
   * Optional className tokens APPENDED to the system tokens. System
   * tokens always apply FIRST. Tailwind 4 alphabetical emission order
   * — use the `!important` prefix (`!bg-slate-900`, `!p-0`) when an
   * override conflicts with a system variant. See
   * `frontend/src/components/ui/README.md`.
   */
  class?: string;
};

export const SettingCard = component$<SettingCardProps>((props) => {
  const variant: SettingCardVariant = props.as === "button" ? "button" : "a";
  const className = settingCardClassName(variant, props.class);

  if (props.as === "button") {
    // Button case: default `type="button"`, forward `disabled`.
    const {
      as: _as,
      icon,
      label,
      testId,
      class: _c,
      type,
      disabled,
      ...rest
    } = props;
    void _as;
    void _c;
    return (
      <button
        {...rest}
        type={type ?? "button"}
        class={className}
        disabled={disabled === true}
        data-testid={testId}
      >
        <span class={ICON_CONTAINER}>{icon}</span>
        <span class={LABEL}>{label}</span>
      </button>
    );
  }

  // Anchor case: `href` is required, no `disabled`, no `type`.
  // TypeScript narrows `props` to the anchor branch automatically
  // because the `as === "button"` branch has returned above; no
  // explicit cast (which would strip the intersection).
  const { as: _as, icon, label, testId, class: _c, href, ...rest } = props;
  void _as;
  void _c;
  const anchorProps = rest as unknown as QwikIntrinsicElements["a"];
  return (
    <a {...anchorProps} href={href} class={className} data-testid={testId}>
      <span class={ICON_CONTAINER}>{icon}</span>
      <span class={LABEL}>{label}</span>
    </a>
  );
});
