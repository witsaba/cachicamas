import { component$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";

/**
 * Landing route — F-1 (spec §6.2).
 *
 * A text-first brand surface for aphantasic users: an <h1> with
 * the product name, a one-sentence tagline, and a single CTA
 * pointing to /organizations.  No hero image, no carousel, no
 * decorative icon.  The CTA is the only navigable affordance.
 */
export default component$(() => {
  return (
    <main class="mx-auto max-w-2xl px-4 py-8">
      <h1 class="text-3xl font-bold text-slate-900">Cachicamas</h1>
      <p class="mt-2 text-slate-700">
        Track the organizations, projects, requirements, and milestones that
        move your work forward.
      </p>
      <div class="mt-6">
        <a
          href="/organizations"
          class="inline-block rounded bg-slate-900 px-4 py-2 font-semibold text-white underline"
        >
          Get started
        </a>
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Track organizations, projects, requirements, and milestones — text-first, aphantasia-friendly.",
    },
  ],
};
