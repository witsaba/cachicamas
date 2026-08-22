/**
 * The Teams section's chrome.
 *
 * Every workspace screen sits inside `<Workspace>`. The section is passed
 * statically rather than derived from the URL, so the rail's current-item
 * highlight and the address bar can never disagree, and so the shell stays
 * testable without a request context.
 *
 * When there is no session the shell is not rendered at all: an anonymous
 * visitor who reaches this route past the server guard gets the screen's own
 * sign-in card on a plain page, not a company rail belonging to nobody.
 */
import { Slot, component$ } from "@builder.io/qwik";
import { type RequestHandler } from "@builder.io/qwik-city";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { Workspace } from "~/components/workspace/workspace";
import { useSession, useSignOut } from "~/routes/plugin@auth";

/**
 * The canonical guard chain, unchanged since it was written:
 *   1. `setSsrCookieHeader` captures the inbound cookie BEFORE any guard can
 *      throw, so SSR-time fetches can re-attach it.
 *   2. `requireAuthRedirect` throws synchronously when anonymous; Qwik turns
 *      the throw into a 302 to the sign-in page.
 *   3. `requireOwnboarding` is async, so it is awaited — that is what lets its
 *      redirect propagate as a redirect rather than a fatal server error.
 */
export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

export default component$(() => {
  const session = useSession();
  const signOut = useSignOut();

  if (!session.value?.user) {
    return (
      <main id="main" class="bg-canvas min-h-screen">
        <Slot />
      </main>
    );
  }

  return (
    <Workspace section="teams" session={session.value} signOut={signOut}>
      <Slot />
    </Workspace>
  );
});
