import { component$ } from "@builder.io/qwik";

import { Field } from "~/components/os/field/field";
import { Gauge } from "~/components/os/gauge/gauge";
import { StateLamp, type LampTone } from "~/components/os/lamp/lamp";
import { Panel } from "~/components/os/panel/panel";
import { RegisterCell } from "~/components/os/register-cell/register-cell";
import { ScreenTitle } from "~/components/os/screen/screen";
import { ARCHETYPES, RUNTIME } from "~/lib/mock/registry";

/**
 * DeskBoard — the register, and the runtime beneath it.
 *
 * Lifted out of `routes/home/index.tsx` so the route keeps only what a route
 * owns — the guard chain and the session — and the board itself can be mounted
 * anywhere it is useful, including without an authenticated session.
 *
 * Everything on it is one of two things: a structural fact read from this
 * repository, or a figure explicitly marked as demonstration data. There is no
 * third category, because the credibility of an interface whose subject is not
 * built yet is the only thing it has (PRODUCT.md § Product Principles 2).
 */

const RUNTIME_TONE: Record<string, LampTone> = {
  complete: "live",
  frozen: "live",
  open: "build",
};

/** The state vocabulary, said once, where someone will actually read it. */
const LEGEND: readonly { tone: LampTone; word: string; means: string }[] = [
  {
    tone: "live",
    word: "On duty",
    means: "Shipped and answering. Nothing is here yet.",
  },
  {
    tone: "build",
    word: "In build",
    means: "Has a milestone plan and work in flight.",
  },
  {
    tone: "ready",
    word: "Planned",
    means: "Named in a decision record. No plan document yet.",
  },
  {
    tone: "idle",
    word: "Unplanned",
    means: "A job this company will need. Nothing else.",
  },
];

export interface DeskBoardProps {
  /** The person's name, when it is known. The lead is impersonal without it. */
  readonly name?: string;
}

export const DeskBoard = component$<DeskBoardProps>((props) => {
  const name = props.name ?? "";

  return (
    <main
      id="main"
      class="mx-auto w-full max-w-[1800px] flex-1 px-3 py-4 sm:px-4"
    >
      <ScreenTitle
        code="DESK"
        title="The register"
        lead={
          name.length > 0
            ? `${name} — this company's specialists, and what each of them is actually doing. Press a function key, or type a code above.`
            : "This company's specialists, and what each of them is actually doing. Press a function key, or type a code above."
        }
      />

      <div class="mt-4 grid grid-cols-1 gap-3 lg:grid-cols-3 2xl:grid-cols-4">
        <div class="lg:col-span-2 2xl:col-span-3">
          <Panel
            label="Archetypes"
            note={`${ARCHETYPES.length} registered · 0 on duty`}
            padded={false}
            testId="archetype-register"
          >
            <div class="bg-rule grid grid-cols-1 gap-px md:grid-cols-2">
              {ARCHETYPES.map((a) => (
                <RegisterCell key={a.code} archetype={a} />
              ))}
            </div>
          </Panel>
        </div>

        <div class="flex flex-col gap-3">
          <Panel
            label="Runtime"
            note="Beneath every archetype"
            testId="runtime-panel"
          >
            <ul class="flex flex-col gap-3">
              {RUNTIME.map((l) => (
                <li
                  key={l.code}
                  class="border-rule border-b pb-3 last:border-b-0 last:pb-0"
                >
                  <div class="flex items-baseline gap-2">
                    <span class="text-label text-fg-mid tracking-[0.16em] uppercase">
                      {l.code}
                    </span>
                    <span class="text-data text-fg">{l.name}</span>
                    <span class="flex-1" />
                    <StateLamp
                      tone={RUNTIME_TONE[l.state]}
                      word={l.stateWord}
                    />
                  </div>
                  <p class="font-human text-data text-fg-mid pt-1 leading-snug">
                    {l.detail}
                  </p>
                  <div class="pt-1.5">
                    <Field label="Milestones">
                      <Gauge
                        done={l.done}
                        total={l.total}
                        testId={`gauge-${l.code}`}
                      />
                    </Field>
                  </div>
                </li>
              ))}
            </ul>
          </Panel>

          <Panel label="Reading this board" testId="legend-panel">
            <ul class="flex flex-col gap-2">
              {LEGEND.map((row) => (
                <li key={row.word} class="flex flex-col gap-0.5">
                  <StateLamp tone={row.tone} word={row.word} />
                  <p class="font-human text-data text-fg-mid pl-3 leading-snug">
                    {row.means}
                  </p>
                </li>
              ))}
            </ul>
            <p
              data-testid="demo-disclosure"
              class="border-rule font-human text-data text-fg-dim mt-3 border-t pt-3 leading-snug"
            >
              No archetype has ever run. Layer counts, plans and authorities on
              this board are read from the repository and are true; anything
              marked <span class="text-amber">demo</span> is invented so the
              screen has something to show.
            </p>
          </Panel>
        </div>
      </div>
    </main>
  );
});
