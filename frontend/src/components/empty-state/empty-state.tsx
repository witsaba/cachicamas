import { component$ } from "@builder.io/qwik";

/**
 * EmptyState — reusable "nothing here yet, do this next" pattern.
 *
 * Locked from spec §5.6 / UX-4 / F-2: the markup is intentionally
 * minimal — exactly one <h1>, one explanatory <p>, one CTA <a>,
 * and ZERO <img> elements.  The component exists so the
 * `/organizations` route and any future empty surface (e.g. a
 * forthcoming `/projects` list) render the same instruction-
 * shaped affordance instead of inventing one ad-hoc.
 */
export interface EmptyStateProps {
  /** Headline rendered as the only <h1> on the page. */
  heading: string;
  /** One-sentence explanation, rendered as a <p>. */
  body: string;
  /** Href for the call-to-action <a>. */
  ctaHref: string;
  /** Accessible name and visible label of the CTA. */
  ctaLabel: string;
}

export const EmptyState = component$<EmptyStateProps>(
  ({ heading, body, ctaHref, ctaLabel }) => {
    return (
      <div class="mx-auto max-w-2xl px-4 py-8">
        <h1 class="text-3xl font-bold text-slate-900">{heading}</h1>
        <p class="mt-2 text-slate-700">{body}</p>
        <div class="mt-6">
          <a
            href={ctaHref}
            class="inline-block rounded bg-slate-900 px-4 py-2 font-semibold text-white underline"
          >
            {ctaLabel}
          </a>
        </div>
      </div>
    );
  },
);
