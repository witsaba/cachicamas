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
 *   - displays `Configured` if the persisted row exists, `Default`
 *     if only safe defaults apply (auto-seeded by the backend's
 *     Loader on absent row).
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
import type { Agent } from "~/lib/mock/staff";
import { AGENTS } from "~/lib/mock/staff";

export interface AgentDirectoryProps {
  /**
   * `true` when the persisted `archetype_configurations` row exists
   * for the Assistant (the Loader returned `found: true`). When
   * `false` or `undefined`, the Assistant card shows "Default".
   * When the GET failed entirely (anonymous, offline, server), the
   * route loader passes `undefined` and the card falls back to the
   * static mock `statusWord` — better than hiding the signal
   * completely.
   */
  readonly assistantConfigured?: boolean;
}

export const AgentDirectory = component$<AgentDirectoryProps>(
  ({ assistantConfigured }) => {
    // REQ-FADR-002: statusWord is derived from the API response, not
    // from the static mock. The five mock cards keep their existing
    // statusWord untouched.
    const agents: Agent[] = AGENTS.map((agent) => {
      if (agent.slug !== "assistant") return agent;
      if (assistantConfigured === undefined) return agent;
      return {
        ...agent,
        statusWord: assistantConfigured ? "Configured" : "Default",
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
