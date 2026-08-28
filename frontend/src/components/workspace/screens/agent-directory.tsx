/**
 * Agents — everyone who works here, and everyone who could.
 *
 * Split into two groups rather than sorted into one list, because "on staff"
 * and "could be on staff" are different questions and a mixed list makes a
 * person read every status word to answer either of them.
 *
 * **Server-authoritative (cachicamas-agent-catalog-config-reload, S2-G
 * T5.5, CRL-S-010/011).** The directory renders exactly the rows the
 * polymorphic `GET /api/archetypes` handler returned — one card per entry,
 * nothing fabricated. The static AGENTS literal fallback and the
 * assistantView/assistantConfigured override-hook props are gone:
 *   - a fetch failure is handled at the route (explicit error card),
 *     so the component only ever receives REAL arrays;
 *   - an empty array means "no colleagues yet" and renders an explicit
 *     empty state — never the static mock;
 *   - every card's statusWord derives from its row's `is_override`
 *     (C2/D-ADR-04: the server row wins) inside the shared projection
 *     `archetypeViewToAgent` (D-ADR-02, lib/api/archetype-view.ts).
 */

import { component$ } from "@builder.io/qwik";

import { AgentCard } from "~/components/workspace/agent-card/agent-card";
import {
  PAGE_WELL,
  PageHeader,
} from "~/components/workspace/page-header/page-header";
import { archetypeViewToAgent } from "~/lib/api/archetype-view";
import type { ArchetypeView } from "~/lib/api/archetypes";

export interface AgentDirectoryProps {
  /**
   * The directory list resolved by the `useArchetypeList` route
   * loader (cachicamas-agent-catalog-config-reload, T5.3). REQUIRED:
   * the route owns list-fetch failure (null → explicit error card
   * before this component renders), so this component only ever
   * receives real arrays. An empty array renders the explicit "No
   * colleagues yet" empty state — never a fabricated card.
   */
  readonly archetypes: readonly ArchetypeView[];
}

export const AgentDirectory = component$<AgentDirectoryProps>(
  ({ archetypes }) => {
    if (archetypes.length === 0) {
      return (
        <div class={PAGE_WELL}>
          <PageHeader
            title="Agents"
            lede="Specialist colleagues. Each one has a job, a set of things it is allowed to use, and a limit it will not cross without asking you."
          />
          <div
            data-testid="agent-directory-empty"
            class="border-line bg-surface text-ink-mid rounded-md border p-5 text-base"
          >
            No colleagues yet. Once colleagues are added, they will appear here.
          </div>
        </div>
      );
    }

    // One card per server row, projected through the shared minimal
    // display projection (D-ADR-02). No literal lookup, no fallback.
    const agents = archetypes.map(archetypeViewToAgent);

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
