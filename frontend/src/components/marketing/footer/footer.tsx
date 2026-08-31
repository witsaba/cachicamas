/**
 * The public site's footer.
 *
 * Deliberately short. A four-column footer of links that do not exist yet is
 * the oldest way a young product signals that it is pretending to be older
 * than it is.
 */
import { component$ } from "@builder.io/qwik";

export const MarketingFooter = component$(() => (
  <footer class="border-line bg-surface border-t">
    <div class="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-4 px-5 py-8 sm:px-8">
      <p class="text-ink text-base font-bold tracking-[-0.02em]">cachicamas</p>
      <nav aria-label="Footer">
        <ul class="flex flex-wrap items-center gap-x-6 gap-y-2">
          <li>
            <a
              href="#staff"
              class="text-ink-soft hover:text-ink rounded-sm text-sm"
            >
              The specialists
            </a>
          </li>
          <li>
            <a
              href="#plans"
              class="text-ink-soft hover:text-ink rounded-sm text-sm"
            >
              Plans
            </a>
          </li>
        </ul>
      </nav>
      <p class="text-ink-soft text-sm">
        A place to work with colleagues who are not people.
      </p>
    </div>
  </footer>
));
