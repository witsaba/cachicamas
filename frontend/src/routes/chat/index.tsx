/**
 * `/chat` — a conversation with one colleague.
 *
 * After CH-05.1 the page is wired to the wire (D-5). After CH-08
 * the page's useVisibleTask$ mount fetches the participant's
 * recorded transcript and conversation-summary list, seeds
 * `useChatStream.reset(entries)`, and re-mounts the rail
 * (R-CRI-001, R-CRI-002, R-CRI-005, REQ-8 / REQ-9).
 *
 * The auth chain is in `routes/chat/layout.tsx`; `requireSession`
 * here surfaces the authenticated participant id and the session
 * user object. The participant id is what the page hands to
 * `ChatApp` (D-1: `conversationID == participantID`).
 *
 * What is *not* a mockup is the shape — a colleague answers as
 * they think, shows which tool they used, and stops to ask before
 * anything leaves the building.
 *
 * The live wire is intact and runs beside this screen in
 * `lib/chat-api.ts` and `lib/chat-types.ts`. Replacing the mock is
 * meant to be a swap of `useMockTurn` for that client with the
 * components unchanged; if anything else has to change, that is a
 * seam worth knowing about.
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

  // CH-08 (REQ-8): the participant id is surfaced from the session
  // (requireSession's resolved value) and passed to ChatApp so the
  // page knows which conversation to load on mount. We never trust
  // a URL parameter for this — the wire's cross-participant guard
  // (R-CHS-004.b) would refuse a tampered id, but the page passes
  // it in defensively.
  const session = guard.session;
  const participantID =
    (session?.user as { id?: string } | undefined)?.id ??
    (session?.user?.email ?? "anonymous");

  return (
    <ChatApp
      youName={guard.session?.user?.name ?? ""}
      youEmail={guard.session?.user?.email ?? ""}
      participantID={participantID}
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
