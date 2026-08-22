/**
 * `/home` — the desk. The board every signed-in employee lands on.
 *
 * The route owns exactly two things: the guard chain, and the session. The
 * board itself is `<DeskBoard>`, which knows about neither — which is what lets
 * it be looked at, and tested, without standing up an authenticated session.
 *
 * Guard chain (canonical, unchanged from the retired surfaces):
 *   1. `setSsrCookieHeader` — capture the inbound cookie before any guard
 *      throws, so SSR-time fetches can re-attach it.
 *   2. `requireAuthRedirect` — 302 to /auth/signin when anonymous.
 *   3. `requireOwnboarding` — 302 to /ownboarding when no organization exists.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead, type RequestHandler } from "@builder.io/qwik-city";

import { DeskBoard } from "~/components/os/desk-board/desk-board";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { requireSession } from "~/lib/require-session";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();

  const guard = requireSession(sessionSig.value, "/home");
  if (guard.kind === "anon") {
    return (
      <main id="main" class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to see the register."
          redirectTo={guard.pathname}
        />
      </main>
    );
  }

  return <DeskBoard name={guard.session?.user?.name ?? ""} />;
});

export const head: DocumentHead = {
  title: "Desk — cachicamas",
  meta: [
    {
      name: "description",
      content:
        "The register of this company's specialist agents, and what each of them is doing.",
    },
  ],
};
