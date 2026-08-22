/**
 * `/auth/signin` — native sign-in page (replaces the @auth/core built-in HTML).
 *
 * Reference: engram observation `cachicamas/auth/anatomy` (2026-07-06).
 *
 * Why this exists:
 *   The default `@auth/core` signin page rendered a dark HTML body
 *   (`class="__next-auth-theme-auto"`) whose CSS flipped to a GitHub-dark
 *   palette (`#0d1117` background, `#161b22` card, white text) on any
 *   browser with `prefers-color-scheme: dark` enabled. Every other page
 *   in cachicamas is a white `bg-white text-slate-900` surface, so the
 *   signin page looked off-brand the moment the visitor's OS was in
 *   dark mode.
 *
 *   The fix is to render a native Qwik page at this canonical URL and
 *   skip `@auth/core`'s interception for it. The `plugin@auth.ts`
 *   custom `onRequest` (NOT the library's) handles the routing
 *   decision; see the long comment block above the `QwikAuth$` call.
 *
 * What this page does:
 *   Renders a centered card with the cachicamas brand mark, a heading,
 *   a one-line description, and the same `SignInButton` used in the
 *   app header (consistency = no UX wart). The `<Form action={signIn}>`
 *   inside the SignInButton calls our `useSignIn` action, which
 *   internally POSTs to `/auth/signin/github` (handled by
 *   `useSignIn`, NOT by `onRequest`) and starts the OAuth roundtrip.
 *
 * Aphantasic-friendly (UX-4):
 *   - No `<img>` elements anywhere.
 *   - The single `<svg>` is the GitHub Octocat brand mark inside the
 *     SignInButton — a recognizable visual anchor, aphantasia-friendly
 *     per the UX-4 amendment (2026-07-04).
 *   - Plain language, no marketing copy, no metaphor.
 *
 * Why we deliberately ignore `prefers-color-scheme` here:
 *   Every other page in cachicamas is locked to white bg / slate-900
 *   text. A sign-in page that flips to dark mid-funnel breaks the
 *   visual continuity and looks like the visitor left the product.
 *   Consistency wins over OS preference on auth surfaces — see the
 *   engram UX research observation `cachicamas/auth/ux-research`.
 *
 * Trade-off: `useSession()` returns `null` on this page because the
 * custom `onRequest` does not populate `sharedMap.session` for native
 * page renders (the library's `getSessionData()` is internal). If we
 * later want to redirect already-signed-in visitors to `/home`, we
 * must replicate `getSessionData()` here. For now, the page is
 * stateless: anyone landing here sees the sign-in card.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead, useLocation } from "@builder.io/qwik-city";
import { SignInButton } from "~/components/sign-in-button/sign-in-button";
import { useSignIn } from "~/routes/plugin@auth";

// Default post-signin landing for direct visits to /auth/signin (no
// `?callbackUrl=...` query param). The header SignInButton in the
// root layout still uses /home via redirectTo (see layout.tsx), but
// if someone navigates directly to /auth/signin — e.g. from a
// bookmark, a deep link from an OAuth error redirect, or a typed
// URL — the round-trip should land them somewhere sensible.
const DEFAULT_REDIRECT_TO = "/home";

/**
 * Resolve the post-signin landing from the request URL.
 *
 * Auth.js's canonical flow is: protected route's `onRequest`
 * redirects anon visitors to `/auth/signin?callbackUrl=<original>`.
 * After a successful GitHub OAuth roundtrip, the Auth.js callback
 * handler reads `callbackUrl` from the original sign-in request and
 * redirects the user there. To preserve that behaviour, we forward
 * the `callbackUrl` query param verbatim into the SignInButton's
 * hidden `redirectTo` field. Auth.js then sees `redirectTo` and
 * appends it to the OAuth state, so the post-callback redirect
 * lands on the originally-requested URL.
 *
 * Defence against open-redirect:
 *   We MUST validate that `callbackUrl` is a relative path starting
 *   with `/` (and NOT `//` or `http://`). Otherwise an attacker
 *   could craft `/auth/signin?callbackUrl=https://evil.example.com`
 *   and steal the user's OAuth code via the redirect. Auth.js's
 *   redirect callback handler validates this internally too, but
 *   failing fast here gives a cleaner error path.
 */
function resolveRedirectTo(rawCallbackUrl: string | null): string {
  if (!rawCallbackUrl) return DEFAULT_REDIRECT_TO;
  // Must start with a single `/` (relative path). Reject `//foo`
  // (protocol-relative URL) and `https://foo` (absolute URL).
  if (!rawCallbackUrl.startsWith("/") || rawCallbackUrl.startsWith("//")) {
    return DEFAULT_REDIRECT_TO;
  }
  return rawCallbackUrl;
}

export default component$(() => {
  const signIn = useSignIn();
  // `useLocation()` reads the Qwik City location context (qc-l). It
  // is available on both SSR and client; for SSR it reflects the
  // inbound request URL. We use it to pull `?callbackUrl=...` so the
  // page works for both:
  //   1. Direct visits (`/auth/signin`) → DEFAULT_REDIRECT_TO (/home)
  //   2. Redirects from protected routes
  //      (`/auth/signin?callbackUrl=/organizations/42`) → /organizations/42
  const loc = useLocation();
  const redirectTo = resolveRedirectTo(loc.url.searchParams.get("callbackUrl"));

  return (
    <main class="bg-void text-fg flex-1" data-testid="auth-signin-page">
      {/* Subtle gradient accent line — the same visual idiom as the
          landing page (`routes/index.tsx`). Nods to Linear/Cursor
          without breaking the text-first UX-4 constraint. */}
      <div class="bg-amber h-px w-full" aria-hidden="true" />

      <section
        class="mx-auto flex w-full max-w-md flex-col items-stretch px-4 py-16"
        data-testid="auth-signin-card"
      >
        {/* Brand mark — single monospace token, same chrome as the
            app header. Identical to routes/layout.tsx:65-71. */}
        <a
          href="/"
          class="text-label text-amber font-semibold tracking-[0.22em] uppercase no-underline"
          data-testid="auth-signin-brand"
        >
          cachicamas
        </a>

        <h1
          class="text-screen text-fg mt-8 leading-none font-semibold tracking-tight"
          data-testid="auth-signin-heading"
        >
          Sign in to cachicamas
        </h1>

        <p
          class="font-human text-body text-fg-mid mt-3 leading-relaxed"
          data-testid="auth-signin-description"
        >
          You'll be redirected to GitHub to authorise the sign-in. No password
          is stored by cachicamas — your session lives in a signed cookie that
          expires when you sign out.
        </p>

        <div class="mt-8" data-testid="auth-signin-action">
          <SignInButton signIn={signIn} redirectTo={redirectTo} />
        </div>

        <p
          class="text-legend text-fg-dim mt-10 tracking-[0.14em] uppercase"
          data-testid="auth-signin-footnote"
        >
          By signing in you agree to cachicamas acting on your behalf with the
          GitHub scopes you authorise (read your public profile and primary
          email).
        </p>
      </section>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Sign in — Cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Sign in to cachicamas via GitHub. No password stored — your session lives in a signed cookie.",
    },
    // The built-in @auth/core page had a dark <body>. We lock the
    // signin surface to light regardless of OS theme for visual
    // continuity with the rest of the product.
    { name: "color-scheme", content: "light" },
  ],
};
