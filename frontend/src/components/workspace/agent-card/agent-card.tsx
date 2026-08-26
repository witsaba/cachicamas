/**
 * A colleague, as a card.
 *
 * The five things a person wants before deciding whether to talk to someone:
 * who they are, what they do, whether they are actually available, how long
 * they have been here, and one way to start. In that order, because that is
 * the order the questions arrive in.
 *
 * The card is not a container for a heading and an icon — every field on it is
 * a fact about a specific colleague, and two cards side by side read
 * differently. That is the test for whether a card earned its place.
 */
import { component$ } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { Status } from "~/components/workspace/status/status";
import type { Agent } from "~/lib/mock/staff";

export interface AgentCardProps {
  readonly agent: Agent;
  /**
   * When set, the card renders an inline "Configure" link anchored
   * to this URL alongside the status. Used by the AgentDirectory
   * to distinguish the real Assistant from the static mocks
   * (REQ-FADR-001).
   */
  readonly configureHref?: string;
}

export const AgentCard = component$<AgentCardProps>(
  ({ agent, configureHref }) => {
    const hired = agent.status !== "available";
    return (
      <article
        data-testid={`agent-card-${agent.slug}`}
        class="border-line bg-surface flex flex-col rounded-md border p-4 shadow-[var(--shadow-raised)] transition-shadow duration-150 hover:shadow-[var(--shadow-float)]"
      >
        <div class="flex items-start gap-3">
          <AgentAvatar agent={agent} size="lg" />
          <div class="min-w-0 flex-1">
            <h3 class="flex flex-wrap items-baseline gap-x-2">
              <a
                href={`/agents/${agent.slug}/`}
                class="text-ink rounded-sm text-lg font-semibold tracking-[-0.01em] hover:underline"
              >
                {agent.name}
              </a>
              <span class="text-2xs text-ink-soft font-medium">Agent</span>
            </h3>
            <p class="text-ink-soft text-xs">
              {agent.departmentName}
              {agent.tenure ? <> · on staff {agent.tenure}</> : null}
            </p>
          </div>
        </div>

        <p class="text-ink-mid mt-3 text-base">{agent.tagline}</p>

        <ul class="mt-3 space-y-1">
          {agent.skills.slice(0, 3).map((s) => (
            <li key={s.name} class="text-ink-mid flex gap-2 text-sm">
              <Icon name="check" size={14} class="text-ok mt-1 shrink-0" />
              {s.name}
            </li>
          ))}
        </ul>

        <div class="border-line mt-4 flex items-center justify-between gap-3 border-t pt-3">
          <div class="flex items-center gap-2">
            <Status status={agent.status} word={agent.statusWord} />
            {configureHref ? (
              <a
                href={configureHref}
                class="text-ink-soft hover:text-ink rounded-sm text-xs font-medium underline"
                data-testid={`agent-card-${agent.slug}-configure`}
              >
                Configure
              </a>
            ) : null}
          </div>
          {/* Secondary, on both branches. Six brand-filled buttons down one grid
              is six primary actions, which is none; the colleague's own profile
              is where a single committing action belongs. */}
          <Button
            as="a"
            href={hired ? `/chat/?with=${agent.slug}` : `/agents/${agent.slug}/`}
            size="sm"
            variant="secondary"
          >
            {hired ? `Talk to ${agent.name.split(" ")[0]}` : "Add to your team"}
          </Button>
        </div>
      </article>
    );
  },
);
