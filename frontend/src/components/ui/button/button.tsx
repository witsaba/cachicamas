/**
 * Button primitive — the single source of button affordances for
 * `@cachicamas/frontend`.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *
 * Four intents:
 *   - `primary` (default) — the one action on a screen that commits; brand,
 *     filled.
 *   - `secondary` — anything non-committal; white, with a control border.
 *   - `destructive` — refuse, remove, cancel; the stop colour, filled.
 *   - `link` — a word in a sentence; brand, underlined, no surface.
 *
 * Press darkens rather than lifts, because a control that moves under the
 * pointer is a control you have to re-aim at. Focus is NOT per-variant —
 * `global.css` owns the single treatment the whole product shares. See
 * `classes.ts` for the table.
 *
 * Three sizes:
 *   - `sm` — a 28px control. Toolbars, table rows, inline actions.
 *   - `md` (default) — a 36px control. The product's default.
 *   - `lg` — a 44px control. A screen's committing action, and every
 *     marketing call to action.
 *
 * Two element kinds (polymorphic via `as`):
 *   - `as="button"` (default) — native `<button>`. Default `type="button"`
 *     to avoid the classic "button inside form defaults to submit" bug.
 *     `disabled` is computed from `loading || props.disabled`.
 *   - `as="a"` — renders `<a href={href}>`. The `href` prop is required.
 *     Disabled / loading do not apply (links cannot be disabled in HTML).
 *
 * `loading` is an opt-in affordance: when true, the button becomes
 * disabled, gets `aria-busy="true"`, and the consumer is expected to
 * pass alternate content via children (UX-4 — no spinner icon).
 *
 * `class` overrides: the system tokens always apply first; consumer
 * classes are APPENDED. Use this for shape overrides (circular,
 * full-width, dark-surface hover) that the variants don't anticipate.
 *
 * The className table is sourced from `./classes` so the unit tests can
 * pin the affordances without instantiating a Qwik component.
 */
import {
  Slot,
  component$,
  type QwikIntrinsicElements,
  type Signal,
} from "@builder.io/qwik";
import {
  buttonClassName,
  type ButtonSize,
  type ButtonVariant,
} from "./classes";

/**
 * Common attributes every button-like element should accept (events,
 * test ids, aria-*). The element-specific attributes (type, disabled,
 * href, target) are added by the discriminated prop union below.
 */
type CommonButtonAttrs = Omit<
  QwikIntrinsicElements["button"],
  "class" | "type" | "disabled" | "href" | "ref"
>;

/** Props when rendered as a `<button>`. */
export type ButtonAsButtonProps = CommonButtonAttrs & {
  as?: "button";
  type?: "button" | "submit" | "reset";
  disabled?: boolean;
  /** `loading` is sugar for `disabled={true}` + `aria-busy="true"`. */
  loading?: boolean;
  /**
   * Element reference (Qwik convention). Typed loosely because Qwik's
   * intrinsic `ref` is parameterized by element kind and the discriminated
   * union cannot narrow it.
   */
  ref?: Signal<Element | undefined>;
  /** Ignored when `as="button"`. */
  href?: never;
};

/** Props when rendered as an `<a>`. */
export type ButtonAsAnchorProps = CommonButtonAttrs & {
  as: "a";
  href: string;
  /** Standard anchor attributes — typed explicitly because Qwik 1.20's
   * `AnchorHTMLAttributes` interface is empty (the special attrs live
   * in a `SpecialAttrs` map that the public types don't expose). */
  target?: "_self" | "_blank" | "_parent" | "_top" | (string & {}) | undefined;
  rel?: string | undefined;
  download?: string | undefined;
  referrerPolicy?:
    | "no-referrer"
    | "no-referrer-when-downgrade"
    | "origin"
    | "origin-when-cross-origin"
    | "same-origin"
    | "strict-origin"
    | "strict-origin-when-cross-origin"
    | "unsafe-url"
    | undefined;
  /** Element reference. Loose typing — see ButtonAsButtonProps. */
  ref?: Signal<Element | undefined>;
  /** Ignored when `as="a"` — links cannot be disabled in HTML. */
  disabled?: never;
  /** Ignored when `as="a"`. */
  loading?: never;
  /** Ignored when `as="a"`. */
  type?: never;
};

/** Discriminated union of the two polymorphism cases. */
export type ButtonProps = (ButtonAsButtonProps | ButtonAsAnchorProps) & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Optional data-testid override; rendered as `data-testid`. */
  testId?: string;
  /**
   * Optional className tokens appended to the system className.
   * Use for shape overrides (circular, full-width, dark-surface hover)
   * that the variants don't anticipate. Does NOT replace the system
   * tokens — both apply.
   */
  class?: string;
};

/**
 * The Button primitive.
 *
 * Implementation note on polymorphism:
 *   We use a discriminated union on `as` rather than casting a
 *   `button` & `a` intersection. The union forces TypeScript to narrow
 *   the element-specific props (`href` / `disabled` / `type`) at the
 *   call site, which is safer than the cast approach used in earlier
 *   designs of this component.
 */
export const Button = component$<ButtonProps>((props) => {
  const variant: ButtonVariant = props.variant ?? "primary";
  const size: ButtonSize = props.size ?? "md";
  const className = buttonClassName(variant, size, props.class);

  if (props.as === "a") {
    // Anchor case: `href` is required, no disabled, no type.
    const {
      as: _as,
      variant: _v,
      size: _s,
      testId,
      class: _c,
      ...rest
    } = props;
    void _as;
    void _v;
    void _s;
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

  // Button case: default type="button" to avoid implicit form submit.
  const {
    as: _as,
    variant: _v,
    size: _s,
    testId,
    class: _c,
    type,
    disabled,
    loading,
    ref,
    ...rest
  } = props;
  void _as;
  void _v;
  void _s;
  void _c;
  const isDisabled = Boolean(loading) || Boolean(disabled);
  return (
    <button
      {...rest}
      ref={ref as unknown as Signal<HTMLButtonElement | undefined>}
      type={type ?? "button"}
      class={className}
      disabled={isDisabled}
      aria-busy={loading ? "true" : undefined}
      data-testid={testId}
    >
      <Slot />
    </button>
  );
});
