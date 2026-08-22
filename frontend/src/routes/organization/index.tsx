/**
 * `/organization` — the officer roles, and who holds them.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { OrganizationPanel } from "~/components/workspace/screens/organization-panel";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/organization");

  if (guard.kind === "anon") {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to set up your organisation."
          redirectTo={guard.pathname}
        />
      </div>
    );
  }

  return (
    <OrganizationPanel
      name={guard.session?.user?.name ?? ""}
      email={guard.session?.user?.email ?? ""}
    />
  );
});

export const head: DocumentHead = {
  title: "Organisation — cachicamas",
  meta: [
    {
      name: "description",
      content: "Who answers for what, and which seats are still empty.",
    },
  ],
};
