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
import { component$, Slot } from "@builder.io/qwik";
import { AvatarDropdown } from "~/components/avatar-dropdown/avatar-dropdown";
import { SignInButton } from "~/components/sign-in-button/sign-in-button";
import { useSession, useSignIn, useSignOut } from "~/routes/plugin@auth";

export default component$(() => {
  const session = useSession();
  const signIn = useSignIn();
  const signOut = useSignOut();
  const isAuthenticated =
    session.value?.user !== null && session.value?.user !== undefined;

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
          href="/"
          class="font-mono text-sm font-bold tracking-tight"
          data-testid="app-shell-brand"
        >
          cachicamas
        </a>
        <div data-testid="app-shell-identity">
          {isAuthenticated && session.value ? (
            <AvatarDropdown session={session.value} signOut={signOut} />
          ) : (
            <SignInButton signIn={signIn} label="Sign in with GitHub" />
          )}
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
