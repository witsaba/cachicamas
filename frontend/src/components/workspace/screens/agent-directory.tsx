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
}

export const AgentDirectory = component$<AgentDirectoryProps>(
  ({ assistantConfigured, assistantView }) => {
    // REQ-FADR-002: statusWord is derived from the API response, not
    // from the static mock. After T-23 the assistant's display name
    // + tagline also come from the API (the polymorphic view), and
    // AGENTS[0] is just a fallback for offline / SSR-cache-miss.
    // The five mock cards keep their existing data untouched.
    const agents: Agent[] = AGENTS.map((agent) => {
      if (agent.slug !== "assistant") return agent;
      const word =
        assistantConfigured === undefined
          ? undefined // let displayStatusWord provide the static fallback
          : assistantConfigured
            ? "Configured"
            : "Default";
      // The polymorphic view's display_name + tagline take precedence
      // over the AGENTS literal — the server is authoritative.
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
