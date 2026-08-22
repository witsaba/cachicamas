/**
 * `/chat` — a conversation with one colleague.
 *
 * The screen is a mockup: nothing typed here reaches a model, and the
 * workspace says so on every screen. What is not a mockup is the shape — a
 * colleague answers as they think, shows which tool they used, and stops to
 * ask before anything leaves the building.
 *
 * The live wire is untouched and still lives beside this screen in
 * `lib/chat-api.ts` and `lib/chat-types.ts`. Replacing the mock is meant to be
 * a swap of `useMockTurn` for that client with the components unchanged; if
 * anything else has to change, that is a seam worth knowing about.
 *
 * The auth chain is in `routes/chat/layout.tsx`.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { ChatApp } from "~/components/chat/chat-app";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/chat");

  if (guard.kind === "anon") {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to talk to your colleagues."
          redirectTo={guard.pathname}
        />
      </div>
    );
  }

  return (
    <ChatApp
      youName={guard.session?.user?.name ?? ""}
      youEmail={guard.session?.user?.email ?? ""}
    />
  );
});

export const head: DocumentHead = {
  title: "Chat — cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Talk to a specialist colleague. They show their work, and they stop to ask before anything leaves the building.",
    },
  ],
};
