/**
 * `/auth/signout` — native sign-out confirmation page (replaces the
 * @auth/core built-in HTML).
 *
 * Reference: engram observation `cachicamas/auth/anatomy` (2026-07-06).
 *
 * Why this exists:
 *   Same motivation as `routes/auth/signin/index.tsx`. The default
 *   `@auth/core` signout page renders a dark `<body
 *   class="__next-auth-theme-auto">` HTML page on browsers with
 *   prefers-color-scheme: dark. We replace it with a native Qwik page
 *   that matches the rest of cachicamas's white surface.
 *
 * Confirmation UX:
 *   A direct visit to /auth/signout is a deliberate user action (typed
 *   URL, bookmark, or redirect from an auth error). We render a
 *   confirmation card instead of signing the visitor out immediately,
 *   because:
 *     1. CSRF-safety: an immediate sign-out would let any link to
 *        /auth/signout (including a hostile one) terminate the session.
 *        The confirmation Form's `redirectTo` plus the useSignOut
 *        action together ensure the request is intentional.
 *     2. Discoverability: a visitor who mistyped the URL gets a
 *        chance to cancel and stay signed in.
 *
 * Aphantasic-friendly (UX-4):
 *   - No `<img>` elements.
 *   - No `<svg>` (the sign-out action does not need a brand mark; the
 *     GitHub Octocat belongs on sign-IN, not sign-OUT).
 *   - Plain language.
 *
 * Trade-off: same as `routes/auth/signin/index.tsx` — `useSession()`
 * returns `null` on this page because the custom `onRequest` does not
 * populate `sharedMap.session` for native page renders. The page is
 * therefore stateless: anyone landing here sees the confirmation card.
 * If we later want to redirect already-signed-out visitors to `/`, we
 * must replicate `getSessionData()` (see plugin@auth.ts trade-off
 * comment).
 */
import { component$ } from "@builder.io/qwik";
import { Form, type DocumentHead } from "@builder.io/qwik-city";
import { Button } from "~/components/ui/button/button";
import { useSignOut } from "~/routes/plugin@auth";

// Where to send the user after a successful sign-out. Matches the
// avatar-dropdown's hidden redirectTo (`/auth/signin`) — the visitor
// lands on the sign-in surface, ready to re-authenticate if they want.
const POST_SIGNOUT_REDIRECT_TO = "/auth/signin";

export default component$(() => {
  const signOut = useSignOut();

  return (
    <main
      class="min-h-screen bg-white text-slate-900"
      data-testid="auth-signout-page"
    >
      {/* Subtle gradient accent line — same idiom as /auth/signin
          and the landing page. Visual continuity, not decoration. */}
      <div
        class="h-px w-full bg-gradient-to-r from-slate-200 via-indigo-500 to-slate-200"
        aria-hidden="true"
      />

      <section
        class="mx-auto flex max-w-md flex-col items-stretch px-6 py-16 sm:py-24"
        data-testid="auth-signout-card"
      >
        {/* Brand mark — same chrome as the app header and the
            /auth/signin page. Lets the visitor bail to the landing
            without using the browser back button. */}
        <a
          href="/"
          class="font-mono text-sm font-bold tracking-tight"
          data-testid="auth-signout-brand"
        >
          cachicamas
        </a>

        <h1
          class="mt-10 text-2xl leading-tight font-semibold text-slate-900 sm:text-3xl"
          data-testid="auth-signout-heading"
        >
          Sign out of cachicamas?
        </h1>

        <p
          class="mt-3 text-sm leading-relaxed text-slate-600"
          data-testid="auth-signout-description"
        >
          Your session cookie will be cleared and you'll be returned to the
          sign-in surface. You can sign back in at any time with the same GitHub
          account.
        </p>

        <Form
          action={signOut}
          class="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center"
          data-testid="auth-signout-form"
        >
          <input
            type="hidden"
            name="redirectTo"
            value={POST_SIGNOUT_REDIRECT_TO}
          />
          <Button
            type="submit"
            variant="primary"
            testId="auth-signout-submit"
            class="border border-zinc-700 bg-zinc-900 text-zinc-100 shadow-sm hover:border-zinc-600 hover:bg-zinc-800 hover:shadow-md focus-visible:ring-zinc-500"
          >
            Sign out
          </Button>
          {/* Cancel — bails to the landing without touching the session.
              A regular link (not a button) so the affordance reads as
              "secondary, reversible". */}
          <Button
            as="a"
            href="/"
            variant="secondary"
            testId="auth-signout-cancel"
            class="focus-visible:ring-slate-400"
          >
            Cancel
          </Button>
        </Form>

        <p
          class="mt-10 text-xs text-slate-500"
          data-testid="auth-signout-footnote"
        >
          Sign-out is final for this browser. Other devices and browsers remain
          signed in until you sign out from each one.
        </p>
      </section>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Sign out — Cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Confirm sign-out from cachicamas. Clears the session cookie and returns you to the sign-in surface.",
    },
    // Lock the surface to light regardless of OS theme (same rationale
    // as routes/auth/signin/index.tsx).
    { name: "color-scheme", content: "light" },
  ],
};
