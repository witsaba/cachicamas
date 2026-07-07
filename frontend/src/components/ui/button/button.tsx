/**
 * Button primitive — the single source of button affordances for
 * `@cachicamas/frontend`.
 *
 * Reference: `openspec/changes/cachicamas-button-design-system/specs/frontend-ui-button/spec.md`
 *
 * Three intents:
 *   - `primary` (default) — create / update actions; `bg-slate-900`.
 *   - `secondary` — general-purpose outline; `bg-white` + `border-slate-300`.
 *   - `destructive` — delete / remove actions; `bg-red-700`.
 *
 * Two sizes:
 *   - `md` (default) — `text-sm px-4 py-2`. Used by forms, headers, retry buttons.
 *   - `lg` — `text-base px-5 py-3`. Used by hero / empty-state CTAs.
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
 * The className table is sourced from `./classes` so the unit tests can
 * pin the affordances without instantiating a Qwik component.
 */
import { Slot, component$, type QwikIntrinsicElements } from "@builder.io/qwik";
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
  /** Ignored when `as="button"`. */
  href?: never;
};

/** Props when rendered as an `<a>`. */
export type ButtonAsAnchorProps = CommonButtonAttrs & {
  as: "a";
  href: string;
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
  const variant = props.variant ?? "primary";
  const size = props.size ?? "md";
  const className = buttonClassName(variant, size);

  if (props.as === "a") {
    // Anchor case: `href` is required, no disabled, no type.
    // The `rest` cast is safe because TypeScript discriminated the
    // union on `as`; the consumer cannot have passed button-only props
    // (disabled, type, loading). Qwik's intrinsic element types share
    // most attributes but narrow event handlers and refs by element
    // kind, so the union of button+anchor intrinsic types is not
    // assignable to either alone. The runtime guard above is the
    // safety net; the cast here bridges the gap.
    const { as: _as, variant: _v, size: _s, testId, ...rest } = props;
    void _as;
    void _v;
    void _s;
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
    type,
    disabled,
    loading,
    ...rest
  } = props;
  void _as;
  void _v;
  void _s;
  const isDisabled = Boolean(loading) || Boolean(disabled);
  return (
    <button
      {...rest}
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
