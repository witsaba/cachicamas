/**
 * The front desk — where a person lands.
 *
 * It answers three questions in the order they are actually asked: who is here
 * today, what was I doing last, and who could I add. Nothing on it is a metric
 * for its own sake; a dashboard of numbers would be the wrong opening for a
 * product whose unit is a colleague rather than a chart.
 */
import { component$ } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";
import {
  AgentAvatar,
  PersonAvatar,
} from "~/components/workspace/avatar/avatar";
import {
  PAGE_WELL,
  PageHeader,
} from "~/components/workspace/page-header/page-header";
import { Status } from "~/components/workspace/status/status";
import { COMPANY } from "~/lib/mock/company";
import { CONVERSATIONS } from "~/lib/mock/chat";
import { AGENTS, PEOPLE, TEAMS, agentBySlug } from "~/lib/mock/staff";

const CARD =
  "rounded-md border border-line bg-surface shadow-[var(--shadow-raised)]";

export interface FrontDeskProps {
  readonly name: string;
}

export const FrontDesk = component$<FrontDeskProps>(({ name }) => {
  const firstName = name.trim().split(/\s+/)[0] ?? "";
  const onStaff = AGENTS.filter((a) => a.status !== "available");
  const available = AGENTS.filter((a) => a.status === "available");

  return (
    <div class={PAGE_WELL}>
      <PageHeader
        title={firstName ? `Good to see you, ${firstName}` : "Front desk"}
        lede={`${onStaff.length} colleagues on staff at ${COMPANY.name}, ${available.length} more you could hire, and ${PEOPLE.length + 1} people working alongside them.`}
      >
        <Button as="a" href="/chat/" size="md">
          <Icon name="chat" size={16} />
          Start a conversation
        </Button>
      </PageHeader>

      {/* who is here */}
      <section aria-labelledby="today">
        <h2
          id="today"
          class="text-2xs text-ink-soft pb-3 font-semibold tracking-wide uppercase"
        >
          On staff today
        </h2>
        <ul class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {onStaff.map((agent) => (
            <li key={agent.slug}>
              <a
                href={`/chat/?with=${agent.slug}`}
                data-testid={`desk-agent-${agent.slug}`}
                class={`flex items-start gap-3 p-4 transition-shadow duration-150 hover:shadow-[var(--shadow-float)] ${CARD}`}
              >
                <AgentAvatar agent={agent} size="lg" />
                <span class="min-w-0 flex-1">
                  <span class="flex flex-wrap items-baseline gap-x-2">
                    <span class="text-ink text-base font-semibold">
                      {agent.name}
                    </span>
                    <span class="text-2xs text-ink-soft font-medium">
                      Agent
                    </span>
                  </span>
                  <span class="text-ink-mid block truncate pt-0.5 text-sm">
                    {agent.tagline}
                  </span>
                  <span class="block pt-2">
                    <Status status={agent.status} word={agent.statusWord} />
                  </span>
                </span>
              </a>
            </li>
          ))}
        </ul>
      </section>

      <div class="grid gap-4 pt-9 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
        {/* what you were doing */}
        <section class={CARD} aria-labelledby="recent">
          <h2
            id="recent"
            class="border-line text-ink border-b px-4 py-3 text-base font-semibold"
          >
            Where you left off
          </h2>
          <ul class="divide-line divide-y">
            {CONVERSATIONS.map((c) => {
              const agent = agentBySlug(c.agentSlug);
              return (
                <li key={c.id}>
                  <a
                    href={`/chat/?with=${c.agentSlug}`}
                    class="hover:bg-sunken flex items-center gap-3 px-4 py-3 transition-colors duration-150"
                  >
                    {agent ? <AgentAvatar agent={agent} size="md" /> : null}
                    <span class="min-w-0 flex-1">
                      <span class="text-ink block truncate text-base font-medium">
                        {c.title}
                      </span>
                      <span class="text-ink-soft block truncate text-xs">
                        {agent?.name} · {c.age}
                      </span>
                    </span>
                    <Icon
                      name="arrow-right"
                      size={16}
                      class="text-ink-soft shrink-0"
                    />
                  </a>
                </li>
              );
            })}
          </ul>
        </section>

        <div class="space-y-4">
          {/* who you could add */}
          <section class={CARD} aria-labelledby="could-hire">
            <h2
              id="could-hire"
              class="border-line text-ink border-b px-4 py-3 text-base font-semibold"
            >
              You could also hire
            </h2>
            <ul class="divide-line divide-y">
              {available.map((agent) => (
                <li key={agent.slug}>
                  <a
                    href={`/agents/${agent.slug}/`}
                    class="hover:bg-sunken flex items-center gap-3 px-4 py-3 transition-colors duration-150"
                  >
                    <AgentAvatar agent={agent} size="md" />
                    <span class="min-w-0 flex-1">
                      <span class="text-ink block truncate text-base font-medium">
                        {agent.name}
                      </span>
                      <span class="text-ink-soft block truncate text-xs">
                        {agent.tagline}
                      </span>
                    </span>
                  </a>
                </li>
              ))}
            </ul>
            <p class="text-ink-soft px-4 py-3 text-xs">
              Both are included on the Company plan.
            </p>
          </section>

          {/* who works with whom */}
          <section class={CARD} aria-labelledby="your-teams">
            <h2
              id="your-teams"
              class="border-line text-ink border-b px-4 py-3 text-base font-semibold"
            >
              Your teams
            </h2>
            <ul class="divide-line divide-y">
              {TEAMS.map((team) => (
                <li key={team.slug} class="px-4 py-3">
                  <a
                    href="/teams/"
                    class="text-ink rounded-sm text-base font-medium hover:underline"
                  >
                    {team.name}
                  </a>
                  <span class="flex flex-wrap items-center gap-1 pt-2">
                    {team.agentSlugs.map((slug) => {
                      const a = agentBySlug(slug);
                      return a ? (
                        <AgentAvatar key={slug} agent={a} size="sm" />
                      ) : null;
                    })}
                    {team.personIds.map((id) => {
                      const p = PEOPLE.find((x) => x.id === id);
                      return p ? (
                        <PersonAvatar
                          key={id}
                          name={p.name}
                          initials={p.initials}
                          size="sm"
                        />
                      ) : null;
                    })}
                    <span class="text-ink-soft pl-1 text-xs">
                      {team.agentSlugs.length} agents · {team.personIds.length}{" "}
                      people
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
});
