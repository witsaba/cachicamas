/**
 * `/teams` — how the company is arranged.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { TeamsBoard } from "~/components/workspace/screens/teams-board";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/teams");

  if (guard.kind === "anon") {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to see your company's teams."
          redirectTo={guard.pathname}
        />
      </div>
    );
  }

  return <TeamsBoard />;
});

export const head: DocumentHead = {
  title: "Teams — cachicamas",
  meta: [
    {
      name: "description",
      content: "Who works with whom — people and agents in the same list.",
    },
  ],
};
