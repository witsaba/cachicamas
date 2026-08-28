/**
 * `/agents/[slug]` — one colleague's profile.
 *
 * cachicamas-agent-catalog-config-reload (S2-G part 1, T5.4): the route
 * no longer gates rendering on the static AGENTS literal (`agentBySlug`).
 * A pure `resolveAgentProfile(slug)` performs the per-slug server loads
 * and classifies the result into the three-state `AgentProfileResolution`
 * per D-ADR-03:
 *
 *   - profile GET /api/archetypes/{slug} → 200
 *       → { kind: "ok", view, config }   (config is null when the config
 *         GET failed — the profile renders WITHOUT the ConfigureSection,
 *         no synthesized config — CRL-S-014)
 *   - profile GET → 404 (slug not registered on the server)
 *       → { kind: "unknown" } — the page maps this to status(404) + the
 *         "No such colleague" state (CRL-S-013)
 *   - profile GET → transient failure (5xx / offline)
 *       → { kind: "unavailable", message } — an explicit error card, NOT
 *         a 404, and never a profile fabricated from AGENTS (CRL-S-015)
 *
 * A slug absent from AGENTS but known to the server opens the profile
 * (CRL-S-012). The ArchetypeView projects into the profile page via the
 * pure helpers `agentProfileProps` (minimal display props, CRL-S-016)
 * and `archetypeViewToAgent` (full display Agent, D-ADR-02).
 */
import { component$ } from "@builder.io/qwik";
import {
  routeLoader$,
  useLocation,
  type DocumentHead,
} from "@builder.io/qwik-city";

import { ConfigureSection } from "~/components/assistant-configure-section/assistant-configure-section";
import { AgentProfile } from "~/components/workspace/screens/agent-profile";
import { archetypeViewToAgent } from "~/lib/api/archetype-view";
import { getArchetype, type ArchetypeView } from "~/lib/api/archetypes";
import {
  getArchetypeConfig,
  type ArchetypeConfig,
} from "~/lib/api/assistant-config";

export type AgentProfileResolution =
  | { kind: "ok"; view: ArchetypeView; config: ArchetypeConfig | null }
  | { kind: "unknown" }
  | { kind: "unavailable"; message: string };

/**
 * loadArchetypeConfigForSlug fetches the flat `ArchetypeConfig` for the
 * supplied slug via the legacy adapter. On any failure (offline / 5xx /
 * not authed) it returns null — the profile renders WITHOUT the
 * ConfigureSection and no config object is synthesized from the failure
 * (CRL-S-014). It no longer gates on a static AGENTS entry.
 */
export async function loadArchetypeConfigForSlug(
  slug: string,
): Promise<ArchetypeConfig | null> {
  const result = await getArchetypeConfig(slug);
  if (!result.ok) {
    return null;
  }
  return result.value;
}

/**
 * resolveAgentProfile is the pure three-state loader (D-ADR-03) that the
 * `useAgentProfile` route loader delegates to. Exported so the spec at
 * `routes/agents/[slug]/index.test.tsx` can drive it directly with a
 * mocked `globalThis.fetch`, without the Qwik City request context.
 *
 *   - profile GET not_found → { kind: "unknown" } (honest 404)
 *   - profile GET any other failure → { kind: "unavailable", message }
 *   - profile GET ok → fetch the per-org config; a config failure
 *     degrades to config:null, never to a fabricated config.
 */
export async function resolveAgentProfile(
  slug: string,
): Promise<AgentProfileResolution> {
  const profile = await getArchetype(slug);
  if (!profile.ok) {
    if (profile.kind === "not_found") {
      return { kind: "unknown" };
    }
    return { kind: "unavailable", message: profile.message };
  }
  const config = await loadArchetypeConfigForSlug(slug);
  return { kind: "ok", view: profile.value, config };
}

/** T5.4 task-artifact name; the S2-R spec pins `resolveAgentProfile`. */
export const loadAgentProfileForSlug = resolveAgentProfile;

/**
 * agentProfileProps projects the ArchetypeView minimally into the
 * AgentProfile display props (CRL-S-016, D-ADR-02): name=display_name,
 * tagline, slug — no extra transformation.
 */
export function agentProfileProps(view: ArchetypeView): {
  name: string;
  tagline: string;
  slug: string;
} {
  return { name: view.display_name, tagline: view.tagline, slug: view.slug };
}

// The route loader is the thin Qwik City wrapper: it delegates to the
// pure `resolveAgentProfile` and maps only kind:"unknown" to a real
// status(404) — a transient failure must never masquerade as a 404
// (CRL-S-015).
export const useAgentProfile = routeLoader$(async ({ params, status }) => {
  const slug = params.slug ?? "";
  const resolution = await resolveAgentProfile(slug);
  if (resolution.kind === "unknown") {
    status(404);
  }
  return resolution;
});

export default component$(() => {
  const loc = useLocation();
  const resolution = useAgentProfile();

  if (resolution.value.kind === "unknown") {
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

  // Transient failure (5xx / offline): an explicit error card — never
  // the 404 state, never a profile fabricated from the static literal.
  if (resolution.value.kind === "unavailable") {
    return (
      <div class="mx-auto w-full max-w-2xl px-4 py-16">
        <div class="border-line bg-surface rounded-md border p-5 shadow-[var(--shadow-raised)]">
          <h1 class="text-ink text-xl font-semibold">
            This profile is unavailable
          </h1>
          <p class="text-ink-mid pt-2 text-base">
            We couldn't reach the server to load this colleague's profile.
            {resolution.value.message ? ` (${resolution.value.message})` : ""}
          </p>
          <p class="pt-3">
            <a
              href={loc.url.pathname}
              class="text-brand rounded-sm font-medium underline"
            >
              Try again
            </a>
            {" · "}
            <a
              href="/agents/"
              class="text-brand rounded-sm font-medium underline"
            >
              Back to the staff directory
            </a>
          </p>
        </div>
      </div>
    );
  }

  const { view, config } = resolution.value;
  const props = agentProfileProps(view);
  return (
    <>
      <AgentProfile agent={archetypeViewToAgent(view)} />
      {config ? (
        <div class="mx-auto w-full max-w-3xl px-4 pb-16">
          <ConfigureSection slug={props.slug} initial={config} />
        </div>
      ) : null}
    </>
  );
});

export const head: DocumentHead = ({ resolveValue }) => {
  const resolution = resolveValue(useAgentProfile);
  if (resolution.kind !== "ok") {
    return { title: "Not found — cachicamas", meta: [] };
  }
  const props = agentProfileProps(resolution.view);
  return {
    title: `${props.name} — cachicamas`,
    meta: [{ name: "description", content: props.tagline }],
  };
};
