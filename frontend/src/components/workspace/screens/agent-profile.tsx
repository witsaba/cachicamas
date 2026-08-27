/**
 * One colleague's profile — the page a manager reads before handing over work.
 *
 * It answers, in this order: who is this, are they here, how long have they
 * been here, what can they do, what are they allowed to touch, what will they
 * refuse, and who else do they work with. The last two are the ones a normal
 * staff profile would not have, and they are the ones that matter most: a
 * colleague whose limits are written down is a colleague you can trust with
 * access.
 */
import { component$ } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";
import {
  AgentAvatar,
  PersonAvatar,
} from "~/components/workspace/avatar/avatar";
import { PAGE_WELL } from "~/components/workspace/page-header/page-header";
import { count, people } from "~/lib/plural";
import { Status } from "~/components/workspace/status/status";
import { type Agent, displayStatusWord, personById, teamsForAgent } from "~/lib/mock/staff";

const CARD =
  "rounded-md border border-line bg-surface p-5 shadow-[var(--shadow-raised)]";
const CARD_TITLE = "text-base font-semibold text-ink";

export interface AgentProfileProps {
  readonly agent: Agent;
}

export const AgentProfile = component$<AgentProfileProps>(({ agent }) => {
  const teams = teamsForAgent(agent.slug);
  const hired = agent.status !== "available";
  // The Assistant's status word is API-derived (REQ-FADR-001/002);
  // when this profile renders without the API signal (offline /
  // SSR cache miss), the helper provides a static fallback.
  const statusWord = displayStatusWord(agent);

  return (
    <div class={PAGE_WELL}>
      <p class="pb-4">
        <a
          href="/agents/"
          class="text-ink-soft hover:text-ink inline-flex items-center gap-1 rounded-sm text-xs font-medium"
        >
          <Icon name="arrow-right" size={14} class="rotate-180" />
          All agents
        </a>
      </p>

      {/* who */}
      <header class="flex flex-wrap items-start gap-5 pb-7">
        <AgentAvatar agent={agent} size="xl" />
        <div class="min-w-0 flex-1">
          <h1 class="text-ink flex flex-wrap items-baseline gap-x-3 text-xl font-semibold tracking-[-0.01em]">
            {agent.name}
            <span class="border-brand/25 bg-brand-tint text-2xs text-brand rounded-sm border px-1.5 py-px font-semibold tracking-wide uppercase">
              Agent
            </span>
          </h1>
          <p class="text-ink-mid pt-1 text-base">
            {agent.departmentName}
            {agent.joined ? (
              <>
                {" · "}
                <span data-numeric>
                  On staff {agent.tenure}, since{" "}
                  {new Date(agent.joined).toLocaleDateString("en-GB", {
                    month: "long",
                    year: "numeric",
                  })}
                </span>
              </>
            ) : (
              " · Not on your staff yet"
            )}
          </p>
          <p class="pt-2">
            <Status
              status={agent.status}
              word={statusWord}
              detail={agent.statusDetail}
            />
          </p>
        </div>
        <div class="flex shrink-0 gap-2">
          {hired ? (
            <Button as="a" href={`/chat/?with=${agent.slug}`} size="lg">
              Talk to {agent.name.split(" ")[0]}
            </Button>
          ) : (
            <Button as="a" href="/settings/" size="lg">
              Add to your team
            </Button>
          )}
        </div>
      </header>

      <p class="border-line text-ink-mid max-w-[68ch] border-t pt-6 pb-8 text-lg leading-relaxed">
        {agent.summary}
      </p>

      <div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
        {/* what they can do */}
        <section class={CARD} aria-labelledby="skills">
          <h2 id="skills" class={CARD_TITLE}>
            What {agent.name.split(" ")[0]} does
          </h2>
          <ul class="divide-line divide-y pt-2">
            {agent.skills.map((s) => (
              <li key={s.name} class="flex gap-3 py-3">
                <Icon name="check" size={16} class="text-ok mt-0.5 shrink-0" />
                <span>
                  <span class="text-ink block text-base font-medium">
                    {s.name}
                  </span>
                  <span class="text-ink-soft block text-sm">{s.detail}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>

        <div class="space-y-4">
          {/* what they may touch */}
          <section class={CARD} aria-labelledby="tools">
            <h2 id="tools" class={CARD_TITLE}>
              What it may use
            </h2>
            <ul class="divide-line divide-y pt-2">
              {agent.tools.map((t) => (
                <li key={t.name} class="py-2.5">
                  <span class="text-ink flex items-center gap-2 text-base font-medium">
                    <Icon
                      name="tool"
                      size={15}
                      class="text-ink-soft shrink-0"
                    />
                    {t.name}
                  </span>
                  <span class="text-ink-soft block pt-0.5 pl-[1.4375rem] text-sm">
                    {t.purpose}
                  </span>
                </li>
              ))}
            </ul>
          </section>

          {/* what they will not do */}
          {agent.handsOff ? (
            <section class={CARD} aria-labelledby="limits">
              <h2 id="limits" class={CARD_TITLE}>
                Where it stops
              </h2>
              <p class="text-ink-mid flex gap-2.5 pt-2 text-base">
                <Icon
                  name="shield"
                  size={16}
                  class="text-waiting mt-0.5 shrink-0"
                />
                <span>
                  {agent.handsOff.what} goes to{" "}
                  <span class="text-ink font-medium">{agent.handsOff.to}</span>.
                  It will say so rather than attempt it.
                </span>
              </p>
            </section>
          ) : null}

          {/* who they work with */}
          {teams.length ? (
            <section class={CARD} aria-labelledby="teams-of">
              <h2 id="teams-of" class={CARD_TITLE}>
                Works with
              </h2>
              <ul class="space-y-3 pt-3">
                {teams.map((team) => (
                  <li key={team.slug}>
                    <a
                      href="/teams/"
                      class="text-ink rounded-sm text-base font-medium hover:underline"
                    >
                      {team.name}
                    </a>
                    <span class="flex items-center gap-1 pt-1.5">
                      {team.personIds.map((id) => {
                        const p = personById(id);
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
                        {people(team.personIds.length)} ·{" "}
                        {count(team.agentSlugs.length, "agent")}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
        </div>
      </div>
    </div>
  );
});
