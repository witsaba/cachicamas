/**
 * `/agents/[slug]` — one colleague's profile.
 *
 * T-23 of cachicamas-archetype-system-foundation: the route loader
 * for `slug === "assistant"` now reads the polymorphic
 * `/api/archetypes/assistant/config` endpoint (via the legacy
 * `assistant-config.ts` adapter) instead of the retired
 * `/api/chat/assistant/config`.
 *
 * feat/archetype-list-endpoint (slice 6 — GREEN): the loader now
 * resolves a config for ANY known slug (the
 * `params.slug === "assistant"` gate is dropped). For unknown slugs
 * the loader short-circuits and returns null; the page renders the
 * 404 state via the `agentBySlug` check above. The loader logic is
 * extracted into the pure `loadArchetypeConfigForSlug(slug, agent)`
 * function so the spec at `routes/agents/[slug]/index.test.tsx`
 * can drive it directly without the Qwik City request context
 * that `createDOM()` does not set up.
 *
 * The ConfigureSection continues to consume the flat
 * `ArchetypeConfig` shape via `getAssistantConfig` so its component
 * migration is decoupled from this route's wire-shape switch.
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";

import { ConfigureSection } from "~/components/assistant-configure-section/assistant-configure-section";
import { AgentProfile } from "~/components/workspace/screens/agent-profile";
import {
  getArchetypeConfig,
  type ArchetypeConfig,
} from "~/lib/api/assistant-config";
import { agentBySlug, type Agent } from "~/lib/mock/staff";

export const useAgent = routeLoader$(({ params, status }) => {
  const agent = agentBySlug(params.slug ?? "");
  if (!agent) {
    status(404);
    return null;
  }
  return agent;
});

/**
 * loadArchetypeConfigForSlug is the pure loader logic that
 * `useAssistantConfig` delegates to. Exported so the spec at
 * `routes/agents/[slug]/index.test.tsx` can drive it directly
 * without a Qwik City request context.
 *
 * Contract (feat/archetype-list-endpoint):
 *   - `agent` is null → return null. The page renders the 404
 *     state via `useAgent`; the loader has nothing to fetch.
 *   - `agent` is set → call `getArchetypeConfig(slug)` (the
 *     polymorphic wrapper). On success, return the flat
 *     `ArchetypeConfig` shape. On any failure (offline / 5xx /
 *     not authed), return null so the page renders the profile
 *     WITHOUT the ConfigureSection.
 *
 * The function does NOT gate on `slug === "assistant"` — any
 * archetype in the directory list resolves its per-org config
 * through the polymorphic surface.
 */
export async function loadArchetypeConfigForSlug(
  slug: string,
  agent: Agent | null,
): Promise<ArchetypeConfig | null> {
  if (!agent) {
    return null;
  }
  const result = await getArchetypeConfig(slug);
  if (!result.ok) {
    return null;
  }
  return result.value as ArchetypeConfig;
}

// `useAssistantConfig` delegates to the pure
// `loadArchetypeConfigForSlug`. The route loader is kept as a
// thin Qwik City wrapper so the page can call `useAssistantConfig()`
// inside the component (the pure function is the test surface; the
// loader is the render surface).
export const useAssistantConfig = routeLoader$(async ({ params }) => {
  const slug = params.slug ?? "";
  const agent = agentBySlug(slug) ?? null;
  return loadArchetypeConfigForSlug(slug, agent);
});

export default component$(() => {
  const agent = useAgent();
  const assistantConfig = useAssistantConfig();
  if (!agent.value) {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <h1 class="text-ink text-xl font-semibold">No such colleague</h1>
        <p class="text-ink-mid pt-2 text-base">
          Nobody by that name works here.{" "}
          <a
            href="/agents/"
            class="text-brand rounded-sm font-medium underline"
          >
            See who does
          </a>
          .
        </p>
      </div>
    );
  }
  return (
    <>
      <AgentProfile agent={agent.value} />
      {assistantConfig.value ? (
        <div class="mx-auto w-full max-w-3xl px-4 pb-16">
          <ConfigureSection
            slug={agent.value.slug}
            initial={assistantConfig.value}
          />
        </div>
      ) : null}
    </>
  );
});

export const head: DocumentHead = ({ resolveValue }) => {
  const agent = resolveValue(useAgent);
  return {
    title: agent ? `${agent.name} — cachicamas` : "Not found — cachicamas",
    meta: agent ? [{ name: "description", content: agent.tagline }] : [],
  };
};
