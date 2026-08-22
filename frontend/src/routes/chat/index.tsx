/**
 * `/chat` — the chat archetype's application.
 *
 * The screen is a mockup. `cachicamas_chat` is planned at 0 of 12 (doc 0005),
 * so nothing here reaches a model; the transcript is scripted and says so on
 * the composer, in the status rail, and in the register that links here.
 *
 * The frozen browser wire is untouched and still lives beside this screen in
 * `lib/chat-api.ts` and `lib/chat-types.ts`, waiting for CH-05 to connect it.
 * Replacing the mock is meant to be a swap of `useMockTurn` for that client,
 * with the transcript components unchanged — and anything that does have to
 * change is a Layer 2 seam that was missing, which is the same value delivered
 * as a defect report (doc 0005 § Outcome first).
 *
 * The auth chain is in `routes/chat/layout.tsx`, unchanged.
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
      <main id="main" class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to open the chat archetype."
          redirectTo={guard.pathname}
        />
      </main>
    );
  }

  return <ChatApp />;
});

export const head: DocumentHead = {
  title: "Chat — cachicamas",
  meta: [
    {
      name: "description",
      content:
        "The chat archetype: one conversation, one model, and a hand-off to whichever specialist owns the work.",
    },
  ],
};
