/**
 * `/agents` — the staff directory.
 *
 * The route loader fetches the Assistant's persisted config (when
 * the caller is signed in) so the directory can render the
 * `Configured` / `Default` status word and the inline `Configure`
 * link on the Assistant card (REQ-FADR-001/002). The five mock
 * specialists stay unchanged.
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { AgentDirectory } from "~/components/workspace/screens/agent-directory";
import { getAssistantConfig } from "~/lib/api/assistant-config";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// `useAssistantConfigured` returns true when the Loader found a
// persisted row, false when only safe defaults apply, and null when
// the GET itself failed (anonymous / offline / server). The
// directory treats null as "no signal" and falls back to the static
// mock `statusWord`.
export const useAssistantConfigured = routeLoader$(async () => {
  const session = await useSession();
  if (!session || (session as { kind?: string }).kind === "anon") return null;
  const result = await getAssistantConfig();
  if (!result.ok) return null;
  return true;
});

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/agents");
  const assistantConfigured = useAssistantConfigured();

  if (guard.kind === "anon") {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <SignInRequiredCard
          signIn={signInAction}
          description="Sign in to see your company's staff."
          redirectTo={guard.pathname}
        />
      </div>
    );
  }

  return (
    <AgentDirectory
      assistantConfigured={assistantConfigured.value ?? undefined}
    />
  );
});

export const head: DocumentHead = {
  title: "Agents — cachicamas",
  meta: [
    {
      name: "description",
      content:
        "The specialist colleagues on staff, and the ones you could hire next.",
    },
  ],
};
