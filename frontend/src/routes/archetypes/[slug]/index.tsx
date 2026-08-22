/**
 * `/archetypes/[slug]` — a specialist that is not here yet.
 *
 * The route owns the guard chain and the loader; the screen is
 * `<ArchetypePanel>`, which takes an archetype (or `null`) and knows about
 * neither.
 */
import { component$ } from "@builder.io/qwik";
import {
  type DocumentHead,
  type RequestHandler,
  routeLoader$,
} from "@builder.io/qwik-city";

import { ArchetypePanel } from "~/components/os/archetype-panel/archetype-panel";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { archetypeBySlug, type Archetype } from "~/lib/mock/registry";

export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

export const useArchetype = routeLoader$<Archetype | null>(({ params }) => {
  return archetypeBySlug(params.slug ?? "") ?? null;
});

export default component$(() => {
  const archetype = useArchetype();
  return <ArchetypePanel archetype={archetype.value} />;
});

export const head: DocumentHead = ({ resolveValue }) => {
  const a = resolveValue(useArchetype);
  return {
    title: a ? `${a.name} — cachicamas` : "Not on the register — cachicamas",
    meta: [
      {
        name: "description",
        content: a
          ? `${a.role}. ${a.stateWord.toLowerCase()}.`
          : "No archetype by that name.",
      },
    ],
  };
};
