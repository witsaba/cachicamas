/**
 * `/agents/[slug]` — one colleague's profile.
 *
 * The slug is resolved in a `routeLoader$` so an unknown one is a real 404
 * rather than a page that renders an empty shell.
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";

import { AgentProfile } from "~/components/workspace/screens/agent-profile";
import { agentBySlug } from "~/lib/mock/staff";

export const useAgent = routeLoader$(({ params, status }) => {
  const agent = agentBySlug(params.slug ?? "");
  if (!agent) {
    status(404);
    return null;
  }
  return agent;
});

export default component$(() => {
  const agent = useAgent();
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
  return <AgentProfile agent={agent.value} />;
});

export const head: DocumentHead = ({ resolveValue }) => {
  const agent = resolveValue(useAgent);
  return {
    title: agent ? `${agent.name} — cachicamas` : "Not found — cachicamas",
    meta: agent ? [{ name: "description", content: agent.tagline }] : [],
  };
};
