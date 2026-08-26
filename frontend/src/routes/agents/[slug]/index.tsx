/**
 * `/agents/[slug]` — one colleague's profile.
 *
 * The slug is resolved in a `routeLoader$` so an unknown one is a real 404
 * rather than a page that renders an empty shell.
 *
 * For `slug === "assistant"`, the route loader also fetches the live
 * AssistantConfig (auto-seeded defaults on first read) so the
 * Configure section on the profile can render with the persisted
 * state. Other slugs skip the fetch — the static mock fields are
 * enough for the staff-directory surface.
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
// renders without the ConfigureSection in either case.
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
