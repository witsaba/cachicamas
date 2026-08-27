/**
 * Agents — everyone who works here, and everyone who could.
 *
 * Split into two groups rather than sorted into one list, because "on staff"
 * and "could be on staff" are different questions and a mixed list makes a
 * person read every status word to answer either of them.
 *
 * **Real vs mock signal (REQ-FADR-001/002).** Only the Assistant is real
 * today — the other five are placeholders until their archetypes ship.
 * The directory must distinguish them so the user doesn't try to
 * configure a card that's read-only. The Assistant card:
 *   - displays `display_name` and `tagline` from the polymorphic
 *     `/api/archetypes/assistant/` response (T-23 of
 *     cachicamas-archetype-system-foundation). On the AGENTS fallback
 *     path the literal's `name`/`tagline` are used.
 *   - displays `Configured` if the persisted per-org row exists
 *     (`is_override=true`), `Default` otherwise
 *     (`is_override=false`). The AGENTS fallback carries
 *     `is_override=false` so a stale client surfaces "Default" — the
 *     honest answer when the API was unreachable.
 *   - renders an inline `Configure` link anchored to
 *     `/agents/assistant/#configure` (lands on the ConfigureSection
 *     in the profile page).
 *
 * The five mock cards keep their existing `statusWord` verbatim —
 * they're fictional placeholders, not user-editable.
 */

import { component$ } from "@builder.io/qwik";

import { AgentCard } from "~/components/workspace/agent-card/agent-card";
import {
  PAGE_WELL,
  PageHeader,
} from "~/components/workspace/page-header/page-header";
import type { ArchetypeView } from "~/lib/api/archetypes";
import type { Agent } from "~/lib/mock/staff";
import { AGENTS } from "~/lib/mock/staff";

export interface AgentDirectoryProps {
  /**
   * `true` when the persisted `archetype_configurations` row exists
   * for the Assistant (`is_override=true` on the polymorphic view).
   * `false` when the org is on the system default. When the GET
   * failed entirely AND the AGENTS fallback also produced no signal,
   * the route loader passes `undefined` and the card falls back to
   * the static mock — better than hiding the signal completely.
   */
  readonly assistantConfigured?: boolean;
  /**
   * The polymorphic view for the Assistant (T-23 of
   * cachicamas-archetype-system-foundation, PR-2). When present,
   * the Assistant card uses its `display_name` + `tagline` instead of
   * the AGENTS literal. Optional so this component stays callable
   * from tests that mock only the override flag.
   */
  readonly assistantView?: ArchetypeView;
  /**
   * The directory list resolved by the `useArchetypeList` route
   * loader (feat/archetype-list-endpoint, slice 6). When the loader
   * resolves (200 from the polymorphic GET /api/archetypes handler),
   * the component renders one card per entry — the directory is the
   * list. When the loader is absent (offline / 5xx / not authed) or
   * resolves to an empty list, the component falls back to the
   * static AGENTS literal so the user still sees the assistant card
   * + the existing overlay path. The `assistantView` overlay
   * continues to take precedence over the loader's assistant row
   * (the per-org customisation surface is authoritative for the
   * assistant card).
   */
  readonly archetypes?: readonly ArchetypeView[];
}

/**
 * viewToAgent projects a polymorphic ArchetypeView into the Agent
 * shape the AgentCard consumes. The status word is derived from
 * the view's `override` block (REQ-FADR-001/002): present means
 * the org has a per-org row → "Configured"; absent means the org
 * is on the system default → "Default". Skills, tools, tenure and
 * hands-off come from the static AGENTS literal for the assistant
 * (the polymorphic view does not carry those fields — they belong
 * to the profile page, not the directory card).
 */
function viewToAgent(view: ArchetypeView): Agent {
  // Find the matching AGENTS literal to inherit the static
  // fields (skills, tools, etc.) that the view does not carry.
  // Fall back to a minimal Agent built from the view itself.
  const literal = AGENTS.find((a) => a.slug === view.slug);
  if (!literal) {
    // No static entry — synthesise a minimal Agent from the
    // view. The directory card renders the agent's name,
    // tagline, status word, and a "Configure" link. Other
    // fields (skills, tools, tenure) are blank on the card
    // for these new kinds.
    return {
      slug: view.slug,
      initials: view.slug.slice(0, 2).toUpperCase(),
      name: view.display_name,
      department: "assistant",
      departmentName: "Front desk",
      tagline: view.tagline,
      summary: "",
      status: view.status === "active" ? "working" : "training",
      statusWord: view.override !== undefined ? "Configured" : "Default",
      statusDetail:
        view.status === "active"
          ? "On staff and answering now."
          : "Hired, still being set up.",
      joined: view.created_at,
      tenure: null,
      skills: [],
      tools: [],
      handsOff: null,
      conversationsThisWeek: null,
    };
  }
  // Known slug (e.g. assistant): inherit the static fields,
  // overlay the view's display_name + tagline + status word.
  return {
    ...literal,
    name: view.display_name,
    tagline: view.tagline,
    statusWord: view.override !== undefined ? "Configured" : "Default",
  };
}

export const AgentDirectory = component$<AgentDirectoryProps>(
  ({ assistantConfigured, assistantView, archetypes }) => {
    // The directory source: the loader's list when present and
    // non-empty, otherwise the static AGENTS literal. An empty
    // loader result is treated as "fall back to the static mock"
    // — better than an empty directory.
    const hasList = archetypes !== undefined && archetypes.length > 0;
    const baseAgents: readonly Agent[] = hasList
      ? // Map the polymorphic views into Agent shape so the
        // AgentCard can render them uniformly. The five mock
        // specialists (when they ship) carry their own static
        // statusWord verbatim; the loader entries get the
        // API-derived status word from the view's override
        // block.
        (archetypes as readonly ArchetypeView[]).map(viewToAgent)
      : AGENTS;
    // REQ-FADR-002: statusWord is derived from the API response, not
    // from the static mock. After T-23 the assistant's display name
    // + tagline also come from the API (the polymorphic view), and
    // AGENTS[0] is just a fallback for offline / SSR-cache-miss.
    // The five mock cards keep their existing data untouched.
    const agents: readonly Agent[] = baseAgents.map((agent) => {
      if (agent.slug !== "assistant") return agent;
      const word =
        assistantConfigured === undefined
          ? agent.statusWord // use the API-derived value from the view, or the literal fallback
          : assistantConfigured
            ? "Configured"
            : "Default";
      // The polymorphic view's display_name + tagline take precedence
      // over the AGENTS literal — the server is authoritative. The
      // `assistantView` overlay (passed in addition to the loader
      // list) takes precedence over the loader's assistant row —
      // the per-org customisation surface is the authoritative
      // signal for the assistant card.
      const name = assistantView?.display_name ?? agent.name;
      const tagline = assistantView?.tagline ?? agent.tagline;
      return {
        ...agent,
        name,
        tagline,
        statusWord: word,
      };
    });

    const onStaff = agents.filter((a) => a.status !== "available");
    const available = agents.filter((a) => a.status === "available");

    return (
      <div class={PAGE_WELL}>
        <PageHeader
          title="Agents"
          lede="Specialist colleagues. Each one has a job, a set of things it is allowed to use, and a limit it will not cross without asking you."
        />

        <section aria-labelledby="on-staff">
          <h2
            id="on-staff"
            class="text-2xs text-ink-soft pb-3 font-semibold tracking-wide uppercase"
          >
            On staff · {onStaff.length}
          </h2>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {onStaff.map((agent) => (
              <AgentCard
                key={agent.slug}
                agent={agent}
                configureHref={
                  agent.slug === "assistant"
                    ? "/agents/assistant/#configure"
                    : undefined
                }
              />
            ))}
          </div>
        </section>

        <section aria-labelledby="available" class="pt-9">
          <h2
            id="available"
            class="text-2xs text-ink-soft font-semibold tracking-wide uppercase"
          >
            You could also hire · {available.length}
          </h2>
          <p class="text-ink-mid max-w-[62ch] pt-1 pb-3 text-base">
            Included on the Company plan. Nobody starts work until you say so.
          </p>
          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {available.map((agent) => (
              <AgentCard key={agent.slug} agent={agent} />
            ))}
          </div>
        </section>
      </div>
    );
  },
);
