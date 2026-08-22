import { component$ } from "@builder.io/qwik";

import { Field } from "~/components/os/field/field";
import { Gauge } from "~/components/os/gauge/gauge";
import { StateLamp } from "~/components/os/lamp/lamp";
import { Panel } from "~/components/os/panel/panel";
import { ScreenTitle } from "~/components/os/screen/screen";
import { lampToneFor } from "~/components/os/register-cell/register-cell";
import { Button } from "~/components/ui/button/button";
import type { Archetype } from "~/lib/mock/registry";

/**
 * ArchetypePanel — the screen for a specialist that is not here yet.
 *
 * Lifted out of the route so the route keeps only the guard chain and the
 * loader, and the screen itself can be mounted with any archetype.
 *
 * Every launcher has this screen and most of them get it wrong: a greyed tile
 * that does nothing, or a promise with no date behind it. This one says what
 * the specialist would be responsible for, what it would own, which decision
 * record put it on the register, and — in one sentence, without euphemism —
 * why you cannot use it today. That is worth a whole screen because it is the
 * product's actual state: five of the six registered archetypes are in it.
 */
export interface ArchetypePanelProps {
  /** `null` when the slug matches nothing on the register. */
  readonly archetype: Archetype | null;
}

export const ArchetypePanel = component$<ArchetypePanelProps>((props) => {
  const a = props.archetype;

  if (!a) {
    return (
      <main id="main" class="mx-auto w-full max-w-2xl flex-1 px-4 py-16">
        <Panel label="Not on the register" testId="archetype-unknown">
          <p class="font-human text-body text-fg-mid leading-relaxed">
            There is no archetype by that name. The register holds every
            specialist this company has decided on — go back and pick one from
            the board.
          </p>
          <div class="mt-4">
            <Button as="a" href="/home/" variant="secondary">
              Back to the register
            </Button>
          </div>
        </Panel>
      </main>
    );
  }

  return (
    <main
      id="main"
      class="mx-auto w-full max-w-[1800px] flex-1 px-3 py-4 sm:px-4"
    >
      <ScreenTitle code={a.code} title={a.name} lead={a.role}>
        <StateLamp tone={lampToneFor(a.state)} word={a.stateWord} />
      </ScreenTitle>

      <div class="mt-4 grid grid-cols-1 gap-3 lg:grid-cols-3">
        <Panel
          label="Why it is not here"
          note={a.stateWord}
          testId="blocked-panel"
          class="lg:col-span-2"
        >
          <p class="font-human text-lead text-fg max-w-[68ch] leading-relaxed">
            {a.blockedBy}
          </p>
          <div class="border-rule mt-4 border-t pt-3">
            <Field label="Authority">
              <span class="text-cyan">{a.authority}</span>
            </Field>
            <Field label="Plan">
              {a.plan ? (
                <span class="inline-flex items-center gap-2">
                  <span class="text-cyan">{a.plan.label}</span>
                  <Gauge done={a.plan.done} total={a.plan.total} />
                </span>
              ) : (
                <span class="text-fg-dim">No milestone document yet</span>
              )}
            </Field>
            <Field label="Opens with">
              <span class="text-fg-dim">
                {a.fkey} · type {a.code} at the command line
              </span>
            </Field>
          </div>
        </Panel>

        <Panel label="What it would own" testId="ownership-panel">
          <Field label="Business system">
            {a.system ? (
              <span class="text-fg-mid">{a.system}</span>
            ) : (
              <span class="text-fg-dim">None</span>
            )}
          </Field>
          <p class="border-rule font-human text-data text-fg-mid mt-3 border-t pt-3 leading-relaxed">
            Every business system owns its own tables, and no archetype writes
            into another's schema. When this one needs database work it asks the
            Database Administrator, over that archetype's MCP server.
          </p>
        </Panel>

        <Panel
          label="What it would be responsible for"
          note={`${a.wouldDo.length} responsibilities`}
          testId="responsibilities-panel"
          class="lg:col-span-3"
        >
          <ul class="grid grid-cols-1 gap-x-6 gap-y-2 md:grid-cols-2">
            {a.wouldDo.map((line) => (
              // No numbering: these responsibilities are a set, not a sequence,
              // and a number that carries no order is an ornament pretending to
              // be information.
              <li key={line} class="flex gap-2.5">
                <span aria-hidden="true" class="text-rule-strong pt-0.5">
                  —
                </span>
                <span class="font-human text-body text-fg-mid leading-snug">
                  {line}
                </span>
              </li>
            ))}
          </ul>
        </Panel>
      </div>

      <div class="mt-3 flex flex-wrap gap-2">
        <Button as="a" href="/home/" variant="secondary">
          Back to the register
        </Button>
        <Button as="a" href="/chat/" variant="primary">
          Open the one that is being built
        </Button>
      </div>
    </main>
  );
});
