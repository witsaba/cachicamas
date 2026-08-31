/**
 * The public site's header.
 *
 * A wordmark, three destinations and a single secondary CTA a visitor can
 * take. It stays at the top of the document rather than following the scroll:
 * a sticky bar earns its keep on a long documentation page, not on a page
 * whose primary action is repeated in every second section.
 *
 * Marketing-only chrome: no sign-in affordance, no session prop, no
 * `/auth/` link. Every visitor sees the same shape.
 */
import { component$ } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

const LINKS: readonly { href: string; label: string }[] = [
  { href: "#staff", label: "The specialists" },
  { href: "#how", label: "How it works" },
  { href: "#plans", label: "Plans" },
];

export const MarketingHeader = component$(() => (
  <header class="border-line bg-surface border-b">
    <div class="mx-auto flex h-16 w-full max-w-6xl items-center gap-6 px-5 sm:px-8">
      <a
        href="/"
        class="text-ink rounded-sm text-lg font-bold tracking-[-0.02em]"
      >
        cachicamas
      </a>

      <nav aria-label="Site" class="hidden flex-1 md:block">
        <ul class="flex items-center gap-6">
          {LINKS.map((l) => (
            <li key={l.href}>
              <a
                href={l.href}
                class="text-ink-mid hover:text-ink rounded-sm text-base font-medium transition-colors duration-150"
              >
                {l.label}
              </a>
            </li>
          ))}
        </ul>
      </nav>

      <div class="ml-auto flex items-center gap-2 md:ml-0">
        {/* The hiding lives on a wrapper, not on the control. A `hidden`
            passed through to the button collides with the variant's own
            `inline-flex`, and Tailwind 4 emits utilities in a fixed order
            in which the variant wins — so the button would stay visible
            on a phone and crowd the wordmark. */}
        <span class="hidden sm:block">
          <Button as="a" href="#plans" size="md" variant="secondary">
            See plans
          </Button>
        </span>
      </div>
    </div>
  </header>
));