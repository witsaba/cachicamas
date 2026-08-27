/**
 * `/agents` — the staff directory.
 *
 * T-23 of cachicamas-archetype-system-foundation: the directory's
 * `useAssistantOverride` loader replaces the legacy ad-hoc
 * `/api/chat/assistant/config` round-trip with a call to the
 * polymorphic `/api/archetypes/assistant/` endpoint. The loader now
 * returns the full ArchetypeView so the AgentDirectory can read
 * `display_name` + `tagline` + `is_override` from one source.
 *
 * "Configured" still means the org has a per-org row that shadows the
 * system default (`is_override=true`). "Default" means the org is on
 * the seeded default — the `__default__` row is treated as the system
 * default and surfaces as `is_override=false`. On a fetch failure
 * (offline / anon / server) `resolveSystemArchetype` falls back to
 * the AGENTS-derived synthetic view, so the loader always returns a
 * view.
 *
 * Anonymous callers fall through to the `not_found` branch of the
 * polymorphic envelope, and the AGENTS fallback kicks in: the
 * directory always has data to render.
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { AgentDirectory } from "~/components/workspace/screens/agent-directory";
import type { ArchetypeView } from "~/lib/api/archetypes";
import { resolveSystemArchetype } from "~/lib/hooks/use-system-archetype";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// `useAssistantOverride` returns the polymorphic ArchetypeView for the
// assistant. On any failure (offline, anon, server error), the
// resolver falls back to the AGENTS-derived synthetic view, so the
// loader's return type is non-null in practice — but the type stays
// nullable to keep the unknown-slug escape hatch honest.
export const useAssistantOverride = routeLoader$<ArchetypeView | null>(
  async () => resolveSystemArchetype("assistant"),
);

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

  const view = assistantOverride.value;
  const assistantConfigured = view ? view.is_override : undefined;

  return (
    <AgentDirectory
      assistantConfigured={assistantConfigured}
      assistantView={view ?? undefined}
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
