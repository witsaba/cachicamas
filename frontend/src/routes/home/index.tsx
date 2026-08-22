/**
 * `/home` — the front desk.
 *
 * The route owns the session; the screen is `<FrontDesk>`, which takes a name
 * and knows nothing about authentication. The guard chain lives in this
 * section's layout.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { FrontDesk } from "~/components/workspace/screens/front-desk";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/home");

  if (guard.kind === "anon") {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to open your company."
          redirectTo={guard.pathname}
        />
      </div>
    );
  }

  return <FrontDesk name={guard.session?.user?.name ?? ""} />;
});

export const head: DocumentHead = {
  title: "Front desk — cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Everyone who works here, what they are for, and where you left off.",
    },
  ],
};
