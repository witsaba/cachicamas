/**
 * `/` — the front door.
 *
 * Persuade, in the same world as the product. The argument this page has to
 * win is not "agents are exciting" — everyone has heard that — it is "these
 * ones have jobs and boundaries, and you can stop them." So the page does not
 * describe the register: it renders the real one, with the real states, five
 * of six of them admitting they do not exist. An interface honest enough to
 * show that is the proof.
 *
 * The one interaction quoted below the register is a permission suspension,
 * rendered with the same component the chat screen uses. It is the mechanism
 * that makes a company able to let a specialist near its systems at all, and
 * quoting it is cheaper and truer than a paragraph claiming it.
 */
import { component$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";

import { TranscriptLine } from "~/components/chat/transcript-line";
import { Field } from "~/components/os/field/field";
import { Gauge } from "~/components/os/gauge/gauge";
import { StateLamp } from "~/components/os/lamp/lamp";
import { Panel } from "~/components/os/panel/panel";
import { RegisterCell } from "~/components/os/register-cell/register-cell";
import { Button } from "~/components/ui/button/button";
import { ARCHETYPES, RUNTIME } from "~/lib/mock/registry";
import { useSession } from "~/routes/plugin@auth";

/** The suspension, quoted. Same component the chat screen renders. */
const QUOTED_HOLD = {
  kind: "hold",
  id: "landing",
  tool: "dba.execute",
  intent: "Ask the Database Administrator to drop the staging schema",
  args: [
    ["system", "staging"],
    ["statement", "drop schema staging cascade"],
    ["mode", "destructive"],
  ],
  risk: "Irreversible once it runs. The run is suspended here until you decide.",
  decision: "pending",
} as const;

export default component$(() => {
  // Read so the landing sits in the same auth-aware tree as every other route;
  // the shell's status rail owns the sign-in affordance.
  const session = useSession();
  void session.value;

  return (
    <main id="main" class="flex-1">
      {/* ---- the thesis, beside the proof ------------------------------- */}
      <section class="mx-auto w-full max-w-[1800px] px-3 pt-10 pb-6 sm:px-4 sm:pt-14">
        <div class="grid grid-cols-1 items-start gap-8 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
          <div>
            <h1 class="text-board text-fg max-w-[16ch] leading-[0.95] font-semibold tracking-tight sm:text-[3.25rem]">
              A company of specialists you can talk to.
            </h1>
            <p class="font-human text-lead text-fg-mid mt-5 max-w-[58ch] leading-relaxed">
              cachicamas is a multiplayer agentic system for building and
              running a company. Database administration, finance, marketing,
              ticketing, software delivery — each one a specialist agent with
              its own job, its own systems, and a boundary it cannot cross.
              Employees talk to them.
            </p>
            <div class="mt-7 flex flex-wrap items-center gap-3">
              <Button
                as="a"
                href="/home/"
                size="lg"
                variant="primary"
                testId="get-started"
              >
                Open the register
              </Button>
              <Button
                as="a"
                href="#suspension"
                size="lg"
                variant="secondary"
                testId="see-interface"
              >
                See how it is stopped
              </Button>
            </div>
          </div>

          {/* Not an illustration of the stack — the stack, with the counts this
              repository actually reports. It sits in the first viewport
              because "0 of 6" is the most useful thing a visitor can learn
              about this product in the first ten seconds. */}
          <Panel
            label="What is actually built"
            note="Read from this repository"
            testId="landing-stack"
          >
            {RUNTIME.map((l) => (
              <div
                key={l.code}
                class="border-rule border-b py-2.5 first:pt-0 last:border-b-0 last:pb-0"
              >
                <div class="flex items-baseline gap-2">
                  <span class="text-label text-amber tracking-[0.16em] uppercase">
                    {l.code}
                  </span>
                  <span class="text-data text-fg flex-1">{l.name}</span>
                  <StateLamp
                    tone={l.state === "open" ? "build" : "live"}
                    word={l.stateWord}
                  />
                </div>
                <p class="font-human text-data text-fg-mid pt-1 leading-snug">
                  {l.detail}
                </p>
                <div class="pt-1.5">
                  <Field label="Milestones">
                    <Gauge done={l.done} total={l.total} />
                  </Field>
                </div>
              </div>
            ))}
          </Panel>
        </div>
      </section>

      {/* ---- the proof: the real board, with the real states -------------- */}
      <section class="mx-auto w-full max-w-[1800px] px-3 py-6 sm:px-4">
        <Panel
          label="The register"
          note={`${ARCHETYPES.length} decided · 0 on duty · read from this repository`}
          padded={false}
          testId="landing-register"
        >
          <div class="bg-rule grid grid-cols-1 gap-px sm:grid-cols-2 xl:grid-cols-3">
            {ARCHETYPES.map((a) => (
              <RegisterCell key={a.code} archetype={a} />
            ))}
          </div>
        </Panel>
        <p class="font-human text-data text-fg-dim mt-2 max-w-[80ch] leading-relaxed">
          Five of those six do not exist. That is the honest state of this
          system today, and it is what you would see after signing in — the
          board does not have a marketing mode.
        </p>
      </section>

      {/* ---- the mechanism ----------------------------------------------- */}
      <section
        id="suspension"
        class="mx-auto w-full max-w-[1800px] scroll-mt-4 px-3 py-6 sm:px-4"
      >
        <div class="grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1fr)_24rem]">
          <Panel
            label="How a specialist is stopped"
            padded={false}
            testId="landing-suspension"
          >
            <div class="px-3">
              <p class="font-human text-lead text-fg max-w-[70ch] py-3 leading-relaxed">
                An agent that needs permission does not ask beside the run. It
                suspends <em class="text-hold not-italic">inside</em> it, on the
                same stream as its own output, and says exactly what it is about
                to do before anyone decides. Nothing moves until you answer.
              </p>
              <ol class="divide-rule border-rule divide-y border-t">
                <TranscriptLine entry={QUOTED_HOLD} />
              </ol>
              <p class="font-human text-data text-fg-dim max-w-[70ch] py-3 leading-relaxed">
                If approval happened out of band, the event stream would stop
                being a complete description of the session — and a session you
                cannot fully replay is one you cannot audit. That is why this is
                a runtime mechanism rather than a screen this page invented.
              </p>
            </div>
          </Panel>

          <div class="flex flex-col gap-3">
            <Panel label="The rules that hold" testId="landing-rules">
              <ul class="font-human text-data text-fg-mid flex flex-col gap-2.5 leading-snug">
                <li>
                  Each business system owns its own tables. No specialist writes
                  into another's schema — it asks the one that owns it.
                </li>
                <li>
                  Each business system runs its own server, with exactly one
                  owning specialist. Integration is a boundary, not a plugin.
                </li>
                <li>
                  The runtime underneath cannot tell which specialist is
                  standing on it, which is why adding one changes nothing below.
                </li>
              </ul>
            </Panel>
          </div>
        </div>
      </section>

      <footer class="border-rule mx-auto w-full max-w-[1800px] border-t px-3 py-6 sm:px-4">
        <p class="text-legend text-fg-dim tracking-[0.16em] uppercase">
          cachicamas · a multiplayer agentic system for building and running a
          company
        </p>
      </footer>
    </main>
  );
});

export const head: DocumentHead = {
  title: "cachicamas — a company of specialists you can talk to",
  meta: [
    {
      name: "description",
      content:
        "A multiplayer agentic system for building and running a company. Database administration, finance, marketing, ticketing, software delivery — each a specialist agent with its own job, its own systems, and a boundary it cannot cross.",
    },
  ],
};
