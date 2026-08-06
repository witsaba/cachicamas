/**
 * routes/chat/index.tsx — the /chat page shell.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-3,
 *   REQ-5, REQ-6, REQ-7.
 *
 * The page:
 *   1. Reads useSession() and decides authed-vs-anon via
 *      requireSession — REQ-3 S-3.c.
 *   2. Authed users render <ChatWindow /> which owns the streaming
 *      lifecycle (useChatStream$) + the typed-error surface
 *      (REQ-4, REQ-5, REQ-6).
 *   3. Anon visitors render <SignInRequiredCard /> with the
 *      callbackUrl=/chat so the post-signin redirect lands here.
 *      (Defensive — the SSR onRequest in layout.tsx already
 *      short-circuits anon requests with a 302, but the client-side
 *      navigation fallback mirrors routes/home/.)
 *
 * Aphantasic-friendly: text-first; the chat is monochrome (slate).
 * Head metadata makes the page discoverable to crawlers + the
 * browser tab strip.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { ChatWindow } from "~/components/chat/chat-window";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  // REQ-3 S-3.c — authed session AND onboarding-complete lets the
  // ChatWindow mount; otherwise the SignInRequiredCard carries the
  // signin affordance. We do NOT call listWorkspacesSSR here — the
  // chat doesn't read workspaces — so no cookie-forwarding inside
  // useTask$ is needed.
  const guard = requireSession(sessionSig.value, "/chat");

  if (guard.kind === "anon") {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to use the chat."
          redirectTo={guard.pathname}
        />
      </main>
    );
  }

  // Authed + onboarded. Mount the chat window — it owns its own
  // session store via useChatStream$() (REQ-1 S-1.a, REQ-7 S-7.a).
  // No useTask$ here: the chat's EventSource opens client-side via
  // useVisibleTask$ (browser-only global, SSR would crash).
  return (
    <main>
      <ChatWindow />
    </main>
  );
});

export const head: DocumentHead = {
  title: "Chat \u2014 Cachicamas",
  meta: [
    {
      name: "description",
      content: "Chat with the cachicamas agent.",
    },
  ],
};