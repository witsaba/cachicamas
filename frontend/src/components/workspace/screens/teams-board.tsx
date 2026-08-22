/**
 * Teams — who works with whom.
 *
 * A team here is deliberately mixed: people and agents in one list, told apart
 * by shape and by a word, never sorted into separate columns. Segregating them
 * would say the agents are tooling that the humans use; putting them in one
 * list says what the product actually claims, which is that they are on the
 * team.
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
import { COMPANY } from "~/lib/mock/company";
import { agentBySlug, personById, TEAMS } from "~/lib/mock/staff";

const CARD =
  "rounded-md border border-line bg-surface shadow-[var(--shadow-raised)]";

export const TeamsBoard = component$(() => (
  <div class={PAGE_WELL}>
    <PageHeader
      title="Teams"
      lede={`How ${COMPANY.name} is arranged. People and agents sit in the same list, because they do the same work.`}
    >
      <Button size="md" variant="secondary" disabled>
        <Icon name="plus" size={16} />
        New team
      </Button>
    </PageHeader>

    <div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-2">
      {TEAMS.map((team) => (
        <section
          key={team.slug}
          class={CARD}
          aria-labelledby={`team-${team.slug}`}
          data-testid={`team-${team.slug}`}
        >
          <header class="border-line border-b px-5 py-4">
            <h2
              id={`team-${team.slug}`}
              class="text-ink text-lg font-semibold tracking-[-0.01em]"
            >
              {team.name}
            </h2>
            <p class="text-ink-mid pt-0.5 text-base">{team.purpose}</p>
          </header>

          <ul class="divide-line divide-y">
            {team.agentSlugs.map((slug) => {
              const agent = agentBySlug(slug);
              if (!agent) return null;
              return (
                <li key={slug} class="flex items-center gap-3 px-5 py-3">
                  <AgentAvatar agent={agent} size="md" />
                  <span class="min-w-0 flex-1">
                    <a
                      href={`/agents/${agent.slug}/`}
                      class="text-ink block truncate rounded-sm text-base font-medium hover:underline"
                    >
                      {agent.name}
                    </a>
                    <span class="text-ink-soft block truncate text-xs">
                      {agent.tagline}
                    </span>
                  </span>
                  <span class="border-brand/25 bg-brand-tint text-2xs text-brand shrink-0 rounded-sm border px-1.5 py-px font-semibold tracking-wide uppercase">
                    Agent
                  </span>
                </li>
              );
            })}
            {team.personIds.map((id) => {
              const person = personById(id);
              if (!person) return null;
              return (
                <li key={id} class="flex items-center gap-3 px-5 py-3">
                  <PersonAvatar
                    name={person.name}
                    initials={person.initials}
                    size="md"
                  />
                  <span class="min-w-0 flex-1">
                    <span class="text-ink block truncate text-base font-medium">
                      {person.name}
                    </span>
                    <span class="text-ink-soft block truncate text-xs">
                      {person.title ?? "No officer role"}
                    </span>
                  </span>
                  <span class="border-line bg-sunken text-2xs text-ink-soft shrink-0 rounded-sm border px-1.5 py-px font-semibold tracking-wide uppercase">
                    Person
                  </span>
                </li>
              );
            })}
          </ul>

          {team.pair ? (
            <p class="border-line bg-sunken text-ink-mid flex gap-2.5 border-t px-5 py-3 text-sm">
              <Icon name="teams" size={16} class="text-brand mt-px shrink-0" />
              <span>
                <span class="text-ink font-medium">Working as a pair.</span>{" "}
                {agentBySlug(team.pair[0])?.name} and{" "}
                {agentBySlug(team.pair[1])?.name} hand work to each other
                directly, without waiting for a person in the middle.
              </span>
            </p>
          ) : null}
        </section>
      ))}
    </div>

    {/* The one thing this plan does not have, said once and quietly.
        This is a person's own workspace, not a shop: a dashed panel with its
        own heading and a call to action was the second-loudest thing on the
        page, competing with the teams it was describing. */}
    <p class="border-line text-ink-mid mt-8 flex flex-wrap items-baseline gap-x-2 gap-y-1 border-t pt-5 text-sm">
      <Icon name="teams" size={16} class="text-ink-soft relative top-0.5" />
      <span class="max-w-[70ch]">
        On the Workforce plan a company gets three open desks — specialists you
        define yourself — and can put any two agents together as a permanent
        pair on one team.{" "}
        <a
          href="/#plans"
          class="text-brand rounded-sm font-medium hover:underline"
        >
          See what Workforce includes
        </a>
      </span>
    </p>
  </div>
));
