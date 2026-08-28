/**
 * `/agents` — the staff directory.
 *
 * cachicamas-agent-catalog-config-reload (S2-G part 1, T5.3): the route
 * is server-authoritative. `useAssistantOverride` is dropped — the
 * assistant row (and its is_override flag) arrives inside the
 * polymorphic directory list itself. `useArchetypeList` returns
 * `readonly ArchetypeView[] | null` where the two states mean different
 * things and are rendered differently (CRL-S-010, CRL-S-011):
 *
 *   - null  → the list fetch FAILED (offline / 5xx / anonymous): an
 *     explicit error card at the route — never the static AGENTS
 *     fallback, never fabricated cards.
 *   - []    → an honest empty catalogue: an explicit empty state —
 *     still no fabricated cards.
 *   - data  → AgentDirectory renders one card per server row; the
 *     assistant card's "Configured"/"Default" word derives from the
 *     row's is_override inside AgentDirectory (C2/D-ADR-04) — the
 *     override hook is gone from this route.
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";

import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { AgentDirectory } from "~/components/workspace/screens/agent-directory";
import { listArchetypes, type ArchetypeView } from "~/lib/api/archetypes";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// `useArchetypeList` reads the polymorphic directory list WITHOUT a type
// query (the bare /api/archetypes URL — the full cross-type catalogue).
// On any failure (offline / 5xx / not authed) the loader returns null:
// the route renders the explicit error card below. An empty list is
// passed through as an honest empty state, NOT as a failure.
export const useArchetypeList = routeLoader$<readonly ArchetypeView[] | null>(
  async () => {
    const result = await listArchetypes();
    if (!result.ok) {
      return null;
    }
    return result.value;
  },
);

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  const guard = requireSession(sessionSig.value, "/agents");
  const archetypeList = useArchetypeList();

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

  const archetypes = archetypeList.value;

  // null = the fetch failed. Say so explicitly; the directory has no
  // static fallback anymore (CRL-S-010).
  if (archetypes === null) {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <div class="border-line bg-surface rounded-md border p-5 shadow-[var(--shadow-raised)]">
          <h1 class="text-ink text-xl font-semibold">
            The staff directory is unavailable
          </h1>
          <p class="text-ink-mid pt-2 text-base">
            We couldn't reach the server to load your company's staff. Check
            your connection and try again.
          </p>
          <p class="pt-3">
            <a
              href="/agents/"
              class="text-brand rounded-sm font-medium underline"
            >
              Try again
            </a>
          </p>
        </div>
      </div>
    );
  }

  // [] = an honest empty catalogue. Still no fabricated cards.
  if (archetypes.length === 0) {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <h1 class="text-ink text-xl font-semibold">Nobody on staff yet</h1>
        <p class="text-ink-mid pt-2 text-base">
          Your company's catalogue is empty. Once colleagues are added, they
          will appear here.
        </p>
      </div>
    );
  }

  return <AgentDirectory archetypes={archetypes} />;
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
