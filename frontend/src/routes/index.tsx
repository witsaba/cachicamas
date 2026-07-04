import { component$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";
import { useSession, useSignIn } from "~/routes/plugin@auth";
import { SignInButton } from "~/components/sign-in-button/sign-in-button";

/**
 * Landing page — the front door of cachicamas.
 *
 * Aphantasic-friendly (UX-4, spec §6.2): text-first, no
 * decorative imagery, no icons that carry meaning, no
 * hero illustration.  The visual language is built from
 * typographic hierarchy, monospace section numbers (the
 * "agentic" / framework feel — Linear/Cursor-style), and
 * a subtle gradient accent line at the top of the page.
 *
 * Sections:
 *   [1.0]  The framework   — hero, headline, dual CTA
 *   [2.0]  What you track  — 4-item bento grid
 *   [3.0]  The interface   — CLI/code block
 *   [—]    Footer          — text-only, one line
 *
 * Locked decisions honoured:
 *   - F-1 / UX-1: brand mark is the <h1> (unique on the page).
 *   - UX-4: zero <img> elements.
 *   - F-3 (locked this iteration): primary CTA points to
 *     /organizations/new so the first-run experience is
 *     home → create.  The /organizations list is reachable
 *     by direct URL but not by the landing CTA.
 */

const FEATURES = [
  {
    num: "01",
    title: "Organizations",
    body: "The tenant that owns the work. Each org is an independent root for projects, requirements, and milestones.",
  },
  {
    num: "02",
    title: "Projects",
    body: "Scoped efforts under an organization, each with its own timeline, owner, and set of requirements.",
  },
  {
    num: "03",
    title: "Requirements",
    body: "What the project must deliver, written as plain text you can grep, link, and reason about.",
  },
  {
    num: "04",
    title: "Milestones",
    body: "Measurable checkpoints that make the timeline concrete and the work shippable.",
  },
] as const;

export default component$(() => {
  const signIn = useSignIn();
  const session = useSession();
  const isAuthenticated =
    session.value?.user !== null && session.value?.user !== undefined;
  return (
    <main class="min-h-screen bg-white text-slate-900">
      {/* Subtle gradient accent — the only decorative
          element on the page.  A 1px line that nods to
          Linear/Cursor without breaking the text-first
          constraint. */}
      <div
        class="h-px w-full bg-gradient-to-r from-slate-200 via-indigo-500 to-slate-200"
        aria-hidden="true"
      />

      {/* ===== [1.0] HERO ===== */}
      <section class="mx-auto max-w-5xl px-4 py-16 sm:py-24">
        <p
          class="font-mono text-xs tracking-widest text-slate-500 uppercase"
          data-section="1.0"
        >
          [1.0] The framework
        </p>
        <h1 class="mt-3 text-4xl leading-tight font-bold tracking-tight sm:text-5xl">
          The agentic framework for tracking work.
        </h1>
        <p class="mt-4 max-w-2xl text-lg text-slate-700">
          Track the organizations, projects, requirements, and milestones that
          move your software forward — with agents in the loop. Text-first,
          built for clarity, designed for the AI era.
        </p>
        <div class="mt-8 flex flex-wrap items-center gap-3">
          <a
            href="/organizations/new"
            data-cta="get-started"
            class="inline-block rounded bg-slate-900 px-5 py-2.5 font-semibold text-white underline"
          >
            Get started
          </a>
          <a
            href="#interface"
            data-cta="see-interface"
            class="inline-block rounded border border-slate-300 px-5 py-2.5 font-semibold text-slate-900 underline"
          >
            See the interface
          </a>
          {/* cachicamas-github-login (R-FA-040): GitHub OAuth sign-in CTA.
                      Renders a <Form action={signIn}> with hidden providerId="github".
                      Landing shows this for anonymous visitors — once the
                      visitor has signed in, the shell's AvatarDropdown
                      (introduced in PR-3 of cachicamas-login-ux) replaces
                      this affordance. Wrapping in if(!session) avoids the
                      stale sign-in CTA competing with the avatar dropdown. */}
          {!isAuthenticated ? (
            <SignInButton
              signIn={signIn}
              label="Sign in with GitHub"
              redirectTo="/profile"
            />
          ) : null}
        </div>
      </section>

      {/* ===== [2.0] WHAT YOU CAN TRACK ===== */}
      <section class="mx-auto max-w-5xl px-4 py-12 sm:py-16">
        <p
          class="font-mono text-xs tracking-widest text-slate-500 uppercase"
          data-section="2.0"
        >
          [2.0] What you can track
        </p>
        <h2 class="mt-3 text-2xl font-bold sm:text-3xl">
          Four primitives, one connected graph.
        </h2>
        <p class="mt-3 max-w-2xl text-slate-700">
          cachicamas models your work as four text-first records. Each one is
          reachable from the web, the CLI, and the API.
        </p>
        <div class="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {FEATURES.map((f) => (
            <article
              key={f.num}
              data-feature={f.title.toLowerCase()}
              class="rounded-lg border border-slate-200 bg-slate-50 p-5"
            >
              <p class="font-mono text-xs text-slate-500">{f.num}</p>
              <h3 class="mt-2 text-lg font-semibold text-slate-900">
                {f.title}
              </h3>
              <p class="mt-2 text-sm text-slate-700">{f.body}</p>
            </article>
          ))}
        </div>
      </section>

      {/* ===== [3.0] THE INTERFACE ===== */}
      <section id="interface" class="mx-auto max-w-5xl px-4 py-12 sm:py-16">
        <p
          class="font-mono text-xs tracking-widest text-slate-500 uppercase"
          data-section="3.0"
        >
          [3.0] The interface
        </p>
        <h2 class="mt-3 text-2xl font-bold sm:text-3xl">
          A surface your agents can drive.
        </h2>
        <p class="mt-3 max-w-2xl text-slate-700">
          Every entity in cachicamas is reachable from the command line, so an
          agent can read state, draft changes, and ship them without a human in
          the loop.
        </p>
        <pre
          data-surface="cli"
          class="mt-6 overflow-x-auto rounded-lg bg-slate-900 px-4 py-4 font-mono text-sm text-slate-100"
        >
          {`$ cachicamas org create \\
      --name "Acme Industrial" \\
      --slug acme
✓ created organization #1 (acme)

$ cachicamas project list --org acme
(no projects yet)

$ cachicamas requirement add \\
      --org acme \\
      --project "Q3 launch" \\
      --text "Track first-run onboarding events end-to-end"
✓ requirement #7 attached to project "Q3 launch"`}
        </pre>
      </section>

      {/* ===== FOOTER ===== */}
      <footer class="mx-auto max-w-5xl px-4 py-12">
        <p class="font-mono text-xs text-slate-500" data-footer>
          cachicamas · witsaba framework · text-first · aphantasia-friendly
        </p>
      </footer>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Cachicamas — the agentic framework for tracking work",
  meta: [
    {
      name: "description",
      content:
        "Track organizations, projects, requirements, and milestones — text-first, agent-friendly, aphantasia-friendly.",
    },
  ],
};
