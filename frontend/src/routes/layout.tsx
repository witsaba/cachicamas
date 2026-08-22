/**
 * Root layout — the document, and nothing else.
 *
 * The chrome a person sees depends on where they are, and there are only two
 * answers: the marketing site owns its own header and footer, and every
 * workspace screen sits inside `<Workspace>` via its section's layout. So this
 * file is deliberately thin: the skip link, the session revalidation, and a
 * slot.
 *
 * Why it takes no router context: `useLocation()` reads Qwik City's `qc-l`
 * context, which only exists inside a request handler and which the vitest
 * `createDOM()` harness does not provide. Nothing here needs to know where it
 * is, and every section layout knows statically which section it is.
 *
 * The popstate reload below predates this redesign and is kept verbatim.
 */
import { Slot, component$, useTask$ } from "@builder.io/qwik";

export default component$(() => {
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
    <>
      {/* The first focusable element of the document, visually hidden until
          it is focused. */}
      <a
        href="#main"
        data-testid="skip-to-main"
        class="focus:bg-brand focus:text-ink-inverse sr-only rounded-md focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[100] focus:px-3 focus:py-2 focus:text-base focus:font-medium"
      >
        Skip to main content
      </a>
      <Slot />
    </>
  );
});
