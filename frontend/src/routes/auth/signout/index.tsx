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
import { MarketingFooter } from "~/components/marketing/footer/footer";
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
      class="bg-canvas text-ink flex min-h-screen flex-col"
      data-testid="auth-signout-page"
    >
      {/* Same framing as /auth/signin, for the same reason: these are the two
          surfaces a person meets between being outside and being inside, and
          neither had been designed. */}
      <header class="border-line bg-surface border-b">
        <div class="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-5 sm:px-8">
          <a
            href="/"
            class="text-ink rounded-sm text-lg font-bold tracking-[-0.02em] no-underline"
            data-testid="auth-signout-brand"
          >
            cachicamas
          </a>
          <a
            href="/home/"
            class="text-ink-mid hover:text-ink rounded-sm text-base font-medium"
          >
            Back to your company
          </a>
        </div>
      </header>

      <div class="flex flex-1 items-center justify-center px-4 py-12">
        <section
          class="border-line bg-surface w-full max-w-md rounded-lg border p-8 shadow-[var(--shadow-float)]"
          data-testid="auth-signout-card"
        >
          <h1
            class="text-ink text-2xl font-bold tracking-[-0.025em]"
            data-testid="auth-signout-heading"
          >
            Sign out of cachicamas?
          </h1>

          <p
            class="text-ink-mid pt-3 text-base leading-relaxed"
            data-testid="auth-signout-description"
          >
            You will be returned to the sign-in page, and you can sign back in
            with the same GitHub account whenever you like.
          </p>

          <Form
            action={signOut}
            class="flex flex-col gap-3 pt-7 sm:flex-row sm:items-center"
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
            >
              Sign out
            </Button>
            {/* Cancel bails without touching the session, so it reads as
                secondary and reversible. */}
            <Button
              as="a"
              href="/home/"
              variant="secondary"
              testId="auth-signout-cancel"
            >
              Cancel
            </Button>
          </Form>

          <p
            class="border-line text-ink-soft mt-8 border-t pt-4 text-xs"
            data-testid="auth-signout-footnote"
          >
            This signs you out on this browser only. Other devices stay signed
            in until you sign out from each one.
          </p>
        </section>
      </div>

      <MarketingFooter />
    </main>
  );
});

export const head: DocumentHead = {
  title: "Sign out — cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Sign out of cachicamas on this browser. You can sign back in with the same GitHub account whenever you like.",
    },
    // Lock the surface to light regardless of OS theme (same rationale
    // as routes/auth/signin/index.tsx).
    { name: "color-scheme", content: "light" },
  ],
};
