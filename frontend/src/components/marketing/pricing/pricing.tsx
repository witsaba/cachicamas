/**
 * Plans.
 *
 * Four levels along one axis — how many specialists you have on staff — plus
 * one thing only the top level gets. The billing switch is a real control
 * rather than two sets of numbers, because the question a buyer is answering
 * is "which of these", not "which of these, twice".
 *
 * The prices are a mockup and the section says so once, quietly, under the
 * cards. That line is not decoration: nobody should be able to read this page
 * and believe they have been quoted.
 */
import { component$, useSignal } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";
import {
  ANNUAL_SAVING,
  COMPARISON,
  PLANS,
  priceFor,
  type Billing,
} from "~/lib/mock/plans";

export const Pricing = component$(() => {
  const billing = useSignal<Billing>("annual");

  return (
    <section id="plans" class="border-line bg-canvas scroll-mt-8 border-t">
      <div class="mx-auto w-full max-w-6xl px-5 py-20 sm:px-8">
        <div class="max-w-[54ch]">
          <h2 class="text-ink text-3xl font-bold tracking-[-0.025em]">
            Pay for the colleagues you actually have
          </h2>
          <p class="text-ink-mid pt-3 text-lg">
            Every plan includes the Assistant and unlimited conversations. What
            changes is how many specialists work for you — and, at the top,
            whether you can build one nobody else sells.
          </p>
        </div>

        {/* the billing switch */}
        <div class="flex flex-wrap items-center gap-3 pt-8 pb-8">
          <div
            role="group"
            aria-label="Billing period"
            class="border-line-control bg-surface inline-flex rounded-md border p-0.5"
          >
            {(["monthly", "annual"] as const).map((option) => {
              const current = billing.value === option;
              return (
                <button
                  key={option}
                  type="button"
                  aria-pressed={current}
                  data-testid={`billing-${option}`}
                  onClick$={() => (billing.value = option)}
                  class={[
                    "cursor-pointer rounded-sm px-3.5 py-1.5 text-base font-medium capitalize transition-colors duration-150",
                    current
                      ? "bg-brand text-ink-inverse"
                      : "text-ink-mid hover:text-ink",
                  ].join(" ")}
                >
                  {option}
                </button>
              );
            })}
          </div>
          <p class="text-ink-soft text-sm">
            {billing.value === "annual"
              ? `Billed yearly — ${ANNUAL_SAVING}.`
              : `Billed monthly. Switch to yearly for ${ANNUAL_SAVING}.`}
          </p>
        </div>

        {/* the levels */}
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-4">
          {PLANS.map((plan) => {
            const price = priceFor(plan, billing.value);
            return (
              <article
                key={plan.slug}
                data-testid={`plan-${plan.slug}`}
                class={[
                  "bg-surface flex flex-col rounded-lg border p-6",
                  plan.recommended
                    ? "border-brand shadow-[var(--shadow-float)]"
                    : "border-line shadow-[var(--shadow-raised)]",
                ].join(" ")}
              >
                <div class="flex items-center justify-between gap-2">
                  <h3 class="text-ink text-lg font-semibold tracking-[-0.01em]">
                    {plan.name}
                  </h3>
                  {plan.recommended ? (
                    <span class="bg-brand-tint text-2xs text-brand rounded-sm px-2 py-0.5 font-semibold tracking-wide uppercase">
                      Most companies
                    </span>
                  ) : null}
                </div>
                {/* The row-alignment floor is for the 4-up grid only. Left unscoped it
                    opens a ~48px hole above the price on a stacked phone. */}
                <p class="text-ink-soft pt-1.5 text-sm lg:min-h-[3rem]">
                  {plan.forWhom}
                </p>

                <p class="flex items-baseline gap-1.5 pt-5" data-numeric>
                  <span class="text-ink text-3xl font-bold tracking-[-0.03em]">
                    {price === 0 ? "Free" : `$${price}`}
                  </span>
                  {price === 0 ? null : (
                    <span class="text-ink-soft text-sm">
                      per person / month
                    </span>
                  )}
                </p>
                <p class="text-ink-soft pt-1 text-xs">
                  {price === 0
                    ? "No card needed"
                    : billing.value === "annual"
                      ? "Billed yearly"
                      : "Billed monthly"}
                </p>

                <p class="bg-sunken text-ink mt-5 rounded-md px-3 py-2 text-sm font-medium">
                  {plan.staffing}
                </p>

                <ul class="flex-1 space-y-2 pt-5">
                  {plan.includes.map((line) => (
                    <li key={line} class="text-ink-mid flex gap-2 text-sm">
                      <Icon
                        name="check"
                        size={15}
                        class="text-ok mt-0.5 shrink-0"
                      />
                      {line}
                    </li>
                  ))}
                </ul>

                <p class="pt-6">
                  <Button
                    as="a"
                    href="/auth/signin/"
                    size="md"
                    variant={plan.recommended ? "primary" : "secondary"}
                    class="w-full"
                  >
                    {plan.cta}
                  </Button>
                </p>
              </article>
            );
          })}
        </div>

        <p class="text-ink-soft pt-5 text-sm" data-testid="pricing-disclaimer">
          Preview pricing. These figures are an illustration while the first
          specialists finish training — nothing here is a quote.
        </p>

        {/* the comparison, kept to the rows that actually differ */}
        {/*
          `relative` is load-bearing, not decoration. The visually-hidden
          "Included" / "Not included" labels inside the cells are
          `position: absolute`, and an absolutely positioned element is only
          clipped by a scroll container that is also its containing block. With
          a static wrapper they escape, land at the table's full 672px width,
          and drag the whole document into a 220px horizontal scroll on a phone
          — the page scrolling sideways while the table sits still.
        */}
        <div class="border-line bg-surface relative mt-12 overflow-x-auto rounded-md border">
          <table class="w-full min-w-[42rem] border-collapse text-left">
            <caption class="sr-only">What each plan includes</caption>
            <thead>
              <tr class="border-line border-b">
                <th
                  scope="col"
                  class="text-2xs text-ink-soft px-5 py-3 font-semibold tracking-wide uppercase"
                >
                  &nbsp;
                </th>
                {/* The recommendation does not stop at the card. A table that
                    drops the emphasis makes a reader re-find the column they
                    were just looking at. */}
                {PLANS.map((p) => (
                  <th
                    key={p.slug}
                    scope="col"
                    class={[
                      "px-5 py-3 text-base font-semibold",
                      p.recommended ? "bg-brand-tint text-brand" : "text-ink",
                    ].join(" ")}
                  >
                    {p.name}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody class="divide-line divide-y">
              {COMPARISON.map((row) => (
                <tr key={row.label}>
                  <th
                    scope="row"
                    class="text-ink-mid px-5 py-3 text-base font-normal"
                  >
                    {row.label}
                  </th>
                  {PLANS.map((p) => {
                    const value = row.values[p.slug];
                    return (
                      <td
                        key={p.slug}
                        class={[
                          "text-ink px-5 py-3 text-base",
                          p.recommended ? "bg-brand-tint/45" : "",
                        ].join(" ")}
                      >
                        {value === true ? (
                          <>
                            <Icon
                              name="check"
                              size={16}
                              class="text-ok inline"
                            />
                            <span class="sr-only">Included</span>
                          </>
                        ) : value === false ? (
                          <>
                            <span aria-hidden="true" class="text-ink-soft">
                              —
                            </span>
                            <span class="sr-only">Not included</span>
                          </>
                        ) : (
                          value
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
});
