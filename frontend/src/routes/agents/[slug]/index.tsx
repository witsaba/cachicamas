/**
 * `/agents/[slug]` — one colleague's profile.
 *
 * T-23 of cachicamas-archetype-system-foundation: the route loader
 * for `slug === "assistant"` now reads the polymorphic
 * `/api/archetypes/assistant/config` endpoint (via the legacy
 * `assistant-config.ts` adapter) instead of the retired
 * `/api/chat/assistant/config`. Other slugs skip the fetch — the
 * static mock fields are enough for the staff-directory surface, and
 * the route still returns a real 404 when the slug is unknown.
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
  getAssistantConfig,
  type ArchetypeConfig,
} from "~/lib/api/assistant-config";
import { agentBySlug } from "~/lib/mock/staff";

export const useAgent = routeLoader$(({ params, status }) => {
  const agent = agentBySlug(params.slug ?? "");
  if (!agent) {
    status(404);
    return null;
  }
  return agent;
});

// `useAssistantConfig` returns null when the slug is not "assistant"
// (the only real archetype today), when the GET fails (anonymous /
// offline / server), or when the slug doesn't resolve. The page
// renders without the ConfigureSection in either case. After T-22,
// the GET goes through the polymorphic client; the legacy flat
// `ArchetypeConfig` shape is restored by `assistant-config.ts` for
// downstream consumers.
export const useAssistantConfig = routeLoader$(async ({ params }) => {
  if (params.slug !== "assistant") {
    return null;
  }
  const result = await getAssistantConfig();
  if (!result.ok) {
    return null;
  }
  return result.value as ArchetypeConfig;
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
          <ConfigureSection initial={assistantConfig.value} />
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
