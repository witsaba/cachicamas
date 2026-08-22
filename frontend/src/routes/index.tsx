/**
 * `/` — the public page.
 *
 * A Persuade surface: a visitor has to understand what this is, believe it can
 * be trusted near their company, and find a plan, in that order and inside a
 * couple of screens. Everything on it is about the product; nothing on it is
 * about how the product is built, because a company buying colleagues is not
 * buying an architecture.
 *
 * The proof is a real fragment of the product rather than a claim about it:
 * the hero carries a conversation in which a specialist reads a ledger, shows
 * exactly what it read, and then stops and asks before sending anything. That
 * pause is the whole argument, so it is on screen before any feature list.
 *
 * Every figure, name and conversation on this page is authored demonstration
 * material, and the pricing block says so where a visitor could be misled.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";

import { Icon } from "~/components/icon/icon";
import { MarketingFooter } from "~/components/marketing/footer/footer";
import { MarketingHeader } from "~/components/marketing/header/header";
import { HeroProof } from "~/components/marketing/hero-proof/hero-proof";
import { Pricing } from "~/components/marketing/pricing/pricing";
import { SignInButton } from "~/components/sign-in-button/sign-in-button";
import { Button } from "~/components/ui/button/button";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { AGENTS, agentBySlug } from "~/lib/mock/staff";
import { useSession, useSignIn } from "~/routes/plugin@auth";

const CARD =
  "rounded-lg border border-line bg-surface shadow-[var(--shadow-raised)]";

const QUESTIONS: readonly { q: string; a: string }[] = [
  {
    q: "Do they act on their own?",
    a: "Only inside what you have given them, and never outside the building. Anything that reaches a customer, moves money or changes your data waits for a person to say yes.",
  },
  {
    q: "What can they see?",
    a: "Exactly the tools you connect, and nothing else. Each colleague's profile lists what it may use and what it may only read, in plain words.",
  },
  {
    q: "Can we build one of our own?",
    a: "On the Workforce plan, yes — three open desks for specialists you define, trained on how your company actually works.",
  },
  {
    q: "What happens when one gets something wrong?",
    a: "It says so once and stops. Nothing here retries quietly in the background, and every action is written down with who approved it.",
  },
];

export default component$(() => {
  const session = useSession();
  const signIn = useSignIn();
  const authenticated = Boolean(session.value?.user);
  const finance = agentBySlug("finance") ?? AGENTS[0];

  return (
    <>
      <MarketingHeader signIn={signIn} authenticated={authenticated} />

      <main id="main">
        {/* ── the offer, and the proof, together ───────────────────────── */}
        <section class="border-line bg-surface border-b">
          <div class="mx-auto grid w-full max-w-6xl grid-cols-1 gap-12 px-5 py-16 sm:px-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.05fr)] lg:items-center lg:py-24">
            <div>
              <h1 class="text-ink max-w-[16ch] text-[clamp(2.25rem,5.5vw,3.5rem)] leading-[1.05] font-bold tracking-[-0.035em]">
                Hire the specialists your company is missing.
              </h1>
              <p class="text-ink-mid max-w-[52ch] pt-5 text-lg leading-relaxed">
                Finance, Support, Integrations, a Database Administrator and a
                Coding colleague — each one with a job, a set of tools it is
                allowed to use, and a line it will not cross without asking you.
                They work alongside your people, in the same place.
              </p>

              <div class="flex flex-wrap items-center gap-3 pt-8">
                {authenticated ? (
                  <Button as="a" href="/home/" size="lg">
                    Open your workspace
                  </Button>
                ) : (
                  <SignInButton
                    signIn={signIn}
                    label="Start free"
                    redirectTo="/home"
                    size="lg"
                  />
                )}
                <Button as="a" href="#staff" size="lg" variant="secondary">
                  Meet the specialists
                </Button>
              </div>
              <p class="text-ink-soft pt-4 text-sm">
                The Assistant is on every plan, including the free one.
              </p>
            </div>

            {/* the proof — the product doing the thing, not a picture of it */}
            <HeroProof agent={finance} />
          </div>
        </section>

        {/* ── the staff ────────────────────────────────────────────────── */}
        <section id="staff" class="bg-canvas scroll-mt-8">
          <div class="mx-auto w-full max-w-6xl px-5 py-20 sm:px-8">
            <div class="max-w-[54ch]">
              <h2 class="text-ink text-3xl font-bold tracking-[-0.025em]">
                Six colleagues, each with one job
              </h2>
              <p class="text-ink-mid pt-3 text-lg">
                Not one assistant with a long list of features. Six specialists
                who know where their work ends, and hand the rest to whoever it
                belongs to.
              </p>
            </div>

            {/* A roster, not a card grid.
                Six same-size cards of avatar + heading + text is the lazy
                scaffold, and it is what every page in this category ships. A
                staff list is what a company actually reads: one row per
                colleague, ruled, with the department, the job and what they do
                laid out in columns you can scan down. It is also the only
                section on this page that is not a grid of cards, which is what
                stops the whole page reading as one rhythm. */}
            <ul class="border-line mt-10 border-t">
              {AGENTS.map((agent) => (
                <li
                  key={agent.slug}
                  class="border-line grid grid-cols-1 gap-x-8 gap-y-3 border-b py-6 md:grid-cols-[minmax(0,15rem)_minmax(0,1fr)] lg:grid-cols-[minmax(0,16rem)_minmax(0,1fr)_minmax(0,17rem)]"
                >
                  <div class="flex items-start gap-3">
                    <AgentAvatar agent={agent} size="lg" />
                    <div class="min-w-0">
                      <h3 class="text-ink text-lg font-semibold tracking-[-0.01em]">
                        {agent.name}
                      </h3>
                      <p class="text-ink-soft text-xs">
                        {agent.departmentName}
                      </p>
                    </div>
                  </div>

                  <p class="text-ink-mid max-w-[52ch] text-base">
                    <span class="text-ink font-medium">{agent.tagline}</span>{" "}
                    {agent.summary}
                  </p>

                  <ul class="space-y-1">
                    {agent.skills.slice(0, 3).map((skill) => (
                      <li
                        key={skill.name}
                        class="text-ink-mid flex gap-2 text-sm"
                      >
                        <Icon
                          name="check"
                          size={14}
                          class="text-ok mt-1 shrink-0"
                        />
                        {skill.name}
                      </li>
                    ))}
                  </ul>
                </li>
              ))}
            </ul>
          </div>
        </section>

        {/* ── how it goes ──────────────────────────────────────────────── */}
        <section id="how" class="border-line bg-surface scroll-mt-8 border-t">
          <div class="mx-auto w-full max-w-6xl px-5 py-20 sm:px-8">
            <div class="max-w-[54ch]">
              <h2 class="text-ink text-3xl font-bold tracking-[-0.025em]">
                Three steps, then they are working
              </h2>
              <p class="text-ink-mid pt-3 text-lg">
                The order matters, so it is written as an order: nobody gets
                access before they are hired, and nobody acts before you have
                approved the first one.
              </p>
            </div>

            <ol class="grid grid-cols-1 gap-4 pt-10 lg:grid-cols-3">
              {/* 1 — hire */}
              <li class={`${CARD} flex flex-col p-5`}>
                <h3 class="text-ink text-lg font-semibold tracking-[-0.01em]">
                  First, hire the ones you need
                </h3>
                <p class="text-ink-mid pt-2 text-base">
                  Pick from the directory. They arrive knowing their job and
                  nothing about your company yet.
                </p>
                <ul class="border-line mt-5 space-y-px rounded-md border">
                  {AGENTS.slice(1, 4).map((agent, i) => (
                    <li
                      key={agent.slug}
                      class={[
                        "flex items-center gap-2.5 px-3 py-2",
                        i > 0 ? "border-line border-t" : "",
                      ].join(" ")}
                    >
                      <AgentAvatar agent={agent} size="sm" />
                      <span class="text-ink flex-1 truncate text-sm">
                        {agent.name}
                      </span>
                      <span
                        class={[
                          "text-2xs rounded-sm px-1.5 py-px font-medium",
                          i === 0
                            ? "bg-brand text-ink-inverse"
                            : "border-line-control text-ink-mid border",
                        ].join(" ")}
                      >
                        {i === 0 ? "Hired" : "Add"}
                      </span>
                    </li>
                  ))}
                </ul>
              </li>

              {/* 2 — access */}
              <li class={`${CARD} flex flex-col p-5`}>
                <h3 class="text-ink text-lg font-semibold tracking-[-0.01em]">
                  Then give them exactly what they need
                </h3>
                <p class="text-ink-mid pt-2 text-base">
                  Connect the tools they should use. Each one says what it may
                  do with it, in words, before it touches anything.
                </p>
                <dl class="divide-line border-line mt-5 divide-y rounded-md border">
                  {finance.tools.map((tool) => (
                    <div key={tool.name} class="px-3 py-2">
                      <dt class="text-ink flex items-center gap-2 text-sm font-medium">
                        <Icon
                          name="tool"
                          size={14}
                          class="text-ink-soft shrink-0"
                        />
                        {tool.name}
                      </dt>
                      <dd class="text-ink-soft pt-0.5 pl-[1.375rem] text-xs">
                        {tool.purpose}
                      </dd>
                    </div>
                  ))}
                </dl>
              </li>

              {/* 3 — approve */}
              <li class={`${CARD} flex flex-col p-5`}>
                <h3 class="text-ink text-lg font-semibold tracking-[-0.01em]">
                  And you keep the last word
                </h3>
                <p class="text-ink-mid pt-2 text-base">
                  They do the work and show it to you. Anything that leaves the
                  building stops here first.
                </p>
                <div class="border-line mt-5 rounded-md border p-3">
                  <p class="text-ink flex items-center gap-2 text-sm font-medium">
                    <Icon name="check" size={15} class="text-ok shrink-0" />
                    Used Ledger
                  </p>
                  <p class="text-ink-soft pt-1 pl-[1.4375rem] text-xs">
                    Read 412 entries · nothing changed
                  </p>
                  <p class="border-line text-ink mt-3 flex items-center gap-2 border-t pt-3 text-sm font-medium">
                    <Icon
                      name="shield"
                      size={15}
                      class="text-waiting shrink-0"
                    />
                    Send the chaser to Northgate?
                  </p>
                  <p class="mt-2 flex gap-2 pl-[1.4375rem]">
                    <span class="bg-brand text-2xs text-ink-inverse inline-flex h-6 items-center rounded-md px-2 font-medium">
                      Allow it
                    </span>
                    <span class="border-line-control text-2xs text-ink inline-flex h-6 items-center rounded-md border px-2 font-medium">
                      Don&rsquo;t
                    </span>
                  </p>
                </div>
              </li>
            </ol>
          </div>
        </section>

        {/* ── the thing that is actually different ─────────────────────── */}
        <section class="border-line bg-canvas border-t">
          <div class="mx-auto grid w-full max-w-6xl grid-cols-1 gap-12 px-5 py-20 sm:px-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,0.85fr)] lg:items-center">
            <div>
              <h2 class="text-ink max-w-[18ch] text-3xl font-bold tracking-[-0.025em]">
                They stop before the part you would want to check
              </h2>
              <p class="text-ink-mid max-w-[54ch] pt-4 text-lg leading-relaxed">
                Every colleague here works inside a boundary that is written
                down and visible on their profile. Finance can read your ledger
                and cannot send a supplier anything. Support can draft to a
                customer and cannot send it. Only one colleague may change how
                your data is shaped, and everyone else has to ask it.
              </p>
              <p class="text-ink-mid max-w-[54ch] pt-4 text-lg leading-relaxed">
                It is the difference between an assistant that is impressive for
                a week and a colleague you can leave alone with the company.
              </p>
            </div>

            <ul class="space-y-3">
              {AGENTS.filter((a) => a.handsOff)
                .slice(0, 4)
                .map((agent) => (
                  <li
                    key={agent.slug}
                    class="border-line bg-surface flex items-start gap-3 rounded-md border p-4"
                  >
                    <AgentAvatar agent={agent} size="md" />
                    <p class="text-ink-mid text-base">
                      <span class="text-ink font-semibold">{agent.name}</span>{" "}
                      hands {agent.handsOff!.what.toLowerCase()} to{" "}
                      <span class="text-ink font-medium">
                        {agent.handsOff!.to}
                      </span>
                      .
                    </p>
                  </li>
                ))}
            </ul>
          </div>
        </section>

        <Pricing />

        {/* ── the questions a buyer actually asks ──────────────────────── */}
        <section class="border-line bg-surface border-t">
          <div class="mx-auto grid w-full max-w-6xl grid-cols-1 gap-10 px-5 py-20 sm:px-8 lg:grid-cols-[minmax(0,0.6fr)_minmax(0,1fr)]">
            <h2 class="text-ink text-3xl font-bold tracking-[-0.025em]">
              Before you hand over anything
            </h2>
            <dl class="divide-line border-line divide-y border-t">
              {QUESTIONS.map((item) => (
                <div key={item.q} class="py-5">
                  <dt class="text-ink text-lg font-semibold tracking-[-0.01em]">
                    {item.q}
                  </dt>
                  <dd class="text-ink-mid max-w-[64ch] pt-2 text-base leading-relaxed">
                    {item.a}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        {/* ── the close ────────────────────────────────────────────────── */}
        <section class="bg-deep">
          <div class="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-8 px-5 py-16 sm:px-8">
            <div>
              <h2 class="text-ink-inverse max-w-[20ch] text-2xl font-bold tracking-[-0.025em]">
                Start with one colleague and see what a week looks like
              </h2>
              <p class="max-w-[52ch] pt-3 text-base text-[#b9c2d2]">
                The Assistant is free, forever, for up to five people. Add a
                specialist when you know which job you are short on.
              </p>
            </div>
            {authenticated ? (
              <Button as="a" href="/home/" size="lg">
                Open your workspace
              </Button>
            ) : (
              <SignInButton
                signIn={signIn}
                label="Start free"
                redirectTo="/home"
                size="lg"
              />
            )}
          </div>
        </section>
      </main>

      <MarketingFooter />
    </>
  );
});

export const head: DocumentHead = {
  title: "cachicamas — hire the specialists your company is missing",
  meta: [
    {
      name: "description",
      content:
        "Finance, Support, Integrations, Database and Coding colleagues that work alongside your people — each with a job, the tools it is allowed to use, and a line it will not cross without asking you.",
    },
  ],
};
