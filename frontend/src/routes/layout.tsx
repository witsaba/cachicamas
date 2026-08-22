/**
 * Root layout — the operating system's chrome.
 *
 * ┌─ StatusRail ────────── whose board · which org · demo · clock · identity ─┐
 * │  CommandLine ───────── type a code, press Enter                           │
 * │  <Slot/> ───────────── the running application, and the only thing that   │
 * │                       scrolls                                             │
 * └─ FunctionRail ──────── F1..F8, the dock                                   ┘
 *
 * Two shells, not one:
 *   - Signed out, only the rail renders. There is nothing to launch and no
 *     board to look at, so the landing page gets the whole canvas.
 *   - Signed in, the command line and the dock appear and stay. Every screen
 *     under them is an application in the same frame.
 *
 * Why the layout takes no router context: `useLocation()` reads Qwik City's
 * `qc-l` context, which only exists inside a request handler and which the
 * vitest `createDOM()` harness does not provide. The chrome therefore never
 * asks where it is; the dock reads `window.location` itself on the client,
 * and every navigation from the chrome is a full document load, which is also
 * what an OS launching an application should feel like.
 *
 * The popstate reload below predates this redesign and is kept verbatim.
 */
import { component$, Slot, useTask$ } from "@builder.io/qwik";
import { AvatarDropdown } from "~/components/avatar-dropdown/avatar-dropdown";
import { CommandLine } from "~/components/os/command-line/command-line";
import { FunctionRail } from "~/components/os/function-rail/function-rail";
import { StatusRail } from "~/components/os/status-rail/status-rail";
import { SignInButton } from "~/components/sign-in-button/sign-in-button";
import { useSession, useSignIn, useSignOut } from "~/routes/plugin@auth";

export default component$(() => {
  const session = useSession();
  const signIn = useSignIn();
  const signOut = useSignOut();
  const isAuthenticated =
    session.value?.user !== null && session.value?.user !== undefined;

  // UAT-8 revision 4 (2026-07-04): re-validate session on browser
  // back/forward navigation. Without this, Qwik City's SPA router serves the
  // CACHED component$ render from browser history when the user navigates back
  // to a route they visited before signing out — their name/email/avatar still
  // visible as if signed in, until they manually refresh.
  //
  // Strategy: register a `popstate` listener on `window` IN THE CAPTURE PHASE
  // via `useTask$` + raw `window.addEventListener`. This bypasses Qwik's
  // `useOnWindow` (bubble phase, races with Qwik's own popstate listener) and
  // runs BEFORE it, so the reload happens before Qwik can serve its cached
  // render. `e.stopImmediatePropagation()` in capture phase also prevents
  // Qwik from running its handler.
  //
  // Revision history (so a future contributor doesn't regress):
  //   r1: useOnDocument("popstate", reload) — popstate fires on window.
  //   r2: useOnWindow("popstate", reload) — bubble phase, races with Qwik.
  //   r3: useNavigate + forceReload — needs qc-l context createDOM lacks.
  //   r4 (current): useTask$ + raw addEventListener({ capture: true }).
  useTask$(({ cleanup }) => {
    if (typeof window === "undefined") return;
    const handler = (e: PopStateEvent) => {
      e.stopImmediatePropagation();
      window.location.reload();
    };
    window.addEventListener("popstate", handler, { capture: true });
    cleanup(() =>
      window.removeEventListener("popstate", handler, { capture: true }),
    );
  });

  return (
    <div class="bg-void flex min-h-screen flex-col">
      {/* The skip link is the very first focusable element of the document,
          visually hidden until focused. */}
      <a
        href="#main"
        data-testid="skip-to-main"
        class="focus:border-amber focus:bg-void focus:text-label focus:text-amber sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-50 focus:border focus:px-3 focus:py-2 focus:tracking-[0.12em] focus:uppercase"
      >
        Skip to main content
      </a>

      <StatusRail
        authenticated={isAuthenticated}
        demo={isAuthenticated}
        brandHref={isAuthenticated ? "/home/" : "/"}
      >
        <div data-testid="app-shell-identity">
          {isAuthenticated && session.value ? (
            <AvatarDropdown session={session.value} signOut={signOut} />
          ) : (
            <SignInButton signIn={signIn} redirectTo="/home" />
          )}
        </div>
      </StatusRail>

      {isAuthenticated ? <CommandLine /> : null}

      {/* The one scrolling region in the product. An operating system's chrome
          does not scroll away, so the rail, the command line and the dock sit
          outside this box and the running application scrolls inside it. */}
      <div class="flex min-h-0 flex-1 flex-col overflow-y-auto">
        <Slot />
      </div>

      {isAuthenticated ? <FunctionRail /> : null}
    </div>
  );
});
