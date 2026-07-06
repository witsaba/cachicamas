/**
 * Root layout — wraps every route with the app shell.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-001 (S-AS-001, S-AS-002) — a single root layout that
 *   renders the chrome (header + skip link) above the route's
 *   main content.
 *   R-AS-002 (S-AS-010, S-AS-011) — anonymous visitors see
 *   `SignInButton` (with `providerId="github"`), not the
 *   `AvatarDropdown`.
 *   R-AS-003 (S-AS-020, S-AS-021, S-AS-022) — authenticated
 *   visitors see `AvatarDropdown` with a sanitized avatar URL,
 *   not `SignInButton`.
 *   R-AS-006 (S-AS-050) — a "Skip to main content" link is the
 *   first focusable element of the document, visually hidden
 *   until focused (the standard :focus pattern).
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §8
 *   The Tailwind class pattern for the header + skip link.
 *
 * Why the layout reads `useSession()` directly instead of taking
 * it as a prop: `routes/layout.tsx` is wired by Qwik City at
 * build time, so it can call `useSession()` from the Auth.js
 * plugin (`plugin@auth.ts`) in its own Qwik context. The vitest
 * spec stubs that plugin via `vi.mock("~/routes/plugin@auth")`
 * the same way `routes/index.spec.tsx` does, so the layout is
 * testable without a Qwik City request context.
 *
 * Why we do NOT call `useLocation()` here: `useLocation()` from
 * `@builder.io/qwik-city` reads the Qwik City location context
 * (`qc-l`), which only exists inside a Qwik City request handler.
 * The vitest `createDOM()` harness does not set up the qc-l
 * context — the existing route specs explicitly avoid it for the
 * same reason. The SignInButton's default `redirectTo="/profile"`
 * is good enough for the shell context; the per-route origins
 * stay on the page itself.
 */
import { component$, Slot, useTask$ } from "@builder.io/qwik";
import { AvatarDropdown } from "~/components/avatar-dropdown/avatar-dropdown";
import { OrgPill } from "~/components/org-pill/org-pill";
import { SignInButton } from "~/components/sign-in-button/sign-in-button";
import { useSession, useSignIn, useSignOut } from "~/routes/plugin@auth";

export default component$(() => {
  const session = useSession();
  const signIn = useSignIn();
  const signOut = useSignOut();
  const isAuthenticated =
    session.value?.user !== null && session.value?.user !== undefined;

  // UAT-8 revision 4 (2026-07-04): re-validate session on browser
  // back/forward navigation. Without this, Qwik City's SPA router
  // serves the CACHED component$ render from browser history when
  // the user navigates back to a route they visited before signing
  // out — their name/email/avatar still visible as if signed in,
  // until they manually refresh (which re-runs the server-side
  // session check correctly).
  //
  // Strategy: register a `popstate` listener on `window` IN THE
  // CAPTURE PHASE via `useTask$` + raw `window.addEventListener`.
  // This bypasses Qwik's `useOnWindow` (which uses bubble phase
  // and races with Qwik's own popstate listener for SPA navigation)
  // and runs BEFORE Qwik's listener, so the reload happens before
  // Qwik can serve its cached component$ render. Calling
  // `e.stopImmediatePropagation()` in capture phase also prevents
  // Qwik from running its handler.
  //
  // Revision history (so a future contributor doesn't regress):
  //   r1: useOnDocument("popstate", reload) — broken: popstate
  //       fires on window, not document; sibling roots, no bubble.
  //   r2: useOnWindow("popstate", reload) — registered in bubble
  //       phase, races with Qwik's own popstate handler. Qwik
  //       often serves the cached component$ before the reload
  //       can take effect.
  //   r3: useNavigate + forceReload — broken: useNavigate needs
  //       qc-l context that createDOM() doesn't provide, fails
  //       the layout spec.
  //   r4 (current): useTask$ + raw addEventListener({ capture:
  //       true }) + e.stopImmediatePropagation(). Runs before
  //       Qwik's bubble-phase listener, blocks it, hard reloads.
  useTask$(({ cleanup }) => {
    if (typeof window === "undefined") return;
    const handler = (e: PopStateEvent) => {
      console.log("[UAT-8 r4] popstate fired in capture phase, reloading");
      // Prevent Qwik's own popstate listener from running and
      // serving the cached component$ render.
      e.stopImmediatePropagation();
      window.location.reload();
    };
    // { capture: true } ensures we run BEFORE Qwik's bubble-phase
    // popstate listener for SPA navigation.
    window.addEventListener("popstate", handler, { capture: true });
    cleanup(() =>
      window.removeEventListener("popstate", handler, { capture: true }),
    );
  });

  // S-AS-050: the skip link is the very first focusable element
  // of the document. It is visually hidden until focused (the
  // standard :focus style moves it into view).
  return (
    <>
      <a
        href="#main"
        data-testid="skip-to-main"
        class="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:rounded focus:bg-slate-900 focus:px-3 focus:py-2 focus:text-sm focus:font-semibold focus:text-white"
      >
        Skip to main content
      </a>

      <header
        data-testid="app-shell-header"
        class="mx-auto flex max-w-5xl items-center justify-between px-4 py-4"
      >
        <a
          href={isAuthenticated ? "/home/" : "/"}
          class="font-mono text-sm font-bold tracking-tight"
          data-testid="app-shell-brand"
        >
          cachicamas
        </a>
        <div class="flex items-center gap-3" data-testid="app-shell-right">
          {/*
                R-FIX-002 (2026-07-06): the org pill surfaces the
                current organization context. Mirrors the
                Slack/Linear/PatternFly context-selector pattern.
                Single-tenant: NOT a switcher, just a passive display
                of full_name + a first-letter monogram. Empty state
                ("No organization yet") is rendered by the pill
                itself when the backend returns 404. The pill does
                its own SSR + client-side fetch via useResource$ —
                the layout does not need a routeLoader$ for this.
              */}
          <OrgPill />
          <div data-testid="app-shell-identity">
            {isAuthenticated && session.value ? (
              <AvatarDropdown session={session.value} signOut={signOut} />
            ) : (
              // UX-4 amendment (2026-07-04): the default SignInButton
              // label is "Sign in" + the GitHub Octocat brand mark
              // (rendered by the component itself). No explicit label
              // override needed.
              // R-HP-004 (S-HP-030): after a successful GitHub OAuth
              // roundtrip, the anonymous visitor lands on the new
              // authed-only Home Page (/home) instead of /profile.
              <SignInButton signIn={signIn} redirectTo="/home" />
            )}
          </div>
        </div>
      </header>

      {/* The route's main content. The child route typically
          already provides its own <main>; the layout slots it
          in here. The default `<main id="main">` target of
          the skip link matches the route that does. */}
      <Slot />
    </>
  );
});
