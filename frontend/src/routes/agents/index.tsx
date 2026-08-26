/**
 * `/agents` — the staff directory.
 *
 * The route loader fetches the Assistant's persisted config (when
 * the caller is signed in) so the directory can render the
 * `Configured` / `Default` status word and the inline `Configure`
 * link on the Assistant card (REQ-FADR-001/002). The five mock
 * specialists stay unchanged.
 *
 * "Configured" means the org has a per-org row that shadows the
 * system default (`is_override=true` in the API response). The
 * `__default__` seed row is treated as the system default and
 * surfaces as `assistantConfigured: false` — the user sees "Default".
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { AgentDirectory } from "~/components/workspace/screens/agent-directory";
import { getAssistantConfig } from "~/lib/api/assistant-config";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// `useAssistantOverride` returns true when the caller's org has
// persisted a per-org row that shadows the system default. False
// means the org is running on the seeded default (or the GET
// failed entirely — anonymous / offline / server). The directory
// labels the Assistant card "Default" in either false case.
//
// Anonymous callers fall through to the `not_found` branch of the
// GET envelope (the chat handler refuses them with 403), so we
// don't need a separate session check here.
export const useAssistantOverride = routeLoader$(async () => {
  const result = await getAssistantConfig();
  if (!result.ok) return null;
  return result.value.is_override;
});

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/agents");
  const assistantOverride = useAssistantOverride();

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

  // `null` from the loader (offline / anon / server) is treated as
  // "no signal" — the directory falls back to the static mock
  // statusWord. Otherwise, when the value is defined, we pass it
  // through: `true` means "Configured" (per-org override),
  // `false` means "Default" (system default).
  const assistantConfigured =
    assistantOverride.value === null ? undefined : assistantOverride.value;

  return <AgentDirectory assistantConfigured={assistantConfigured} />;
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
