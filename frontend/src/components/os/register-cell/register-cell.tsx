import { component$ } from "@builder.io/qwik";
import { Field } from "~/components/os/field/field";
import { Gauge } from "~/components/os/gauge/gauge";
import { StateLamp, type LampTone } from "~/components/os/lamp/lamp";
import {
  archetypeHref,
  type Archetype,
  type ArchetypeState,
} from "~/lib/mock/registry";

/**
 * RegisterCell — one archetype's line on the board.
 *
 * This is the launcher tile, in the terminal's grammar: a keyed rectangle that
 * says, in this order, what to press, what it is called, what state it is in,
 * what it is for, and what evidence backs that state. The last of those is the
 * point. A launcher that showed six identical icons would imply six working
 * specialists; this one shows that exactly one has a plan in flight, and where
 * to read it.
 */
const TONE: Record<ArchetypeState, LampTone> = {
  "on-duty": "live",
  "in-build": "build",
  planned: "ready",
  unplanned: "idle",
};

/** Which lamp an archetype's state lights. Pure, so the map is unit-tested. */
export function lampToneFor(state: ArchetypeState): LampTone {
  return TONE[state];
}

export interface RegisterCellProps {
  readonly archetype: Archetype;
}

export const RegisterCell = component$<RegisterCellProps>(
  ({ archetype: a }) => {
    const href = archetypeHref(a);
    const reachable = a.state !== "unplanned";
    return (
      <a
        href={href}
        data-testid={`register-cell-${a.slug}`}
        data-state={a.state}
        class={[
          "group border-rule bg-panel flex flex-col gap-2 border p-3 no-underline",
          "transition-[border-color,background-color] duration-150",
          reachable
            ? "hover:border-amber hover:bg-raise"
            : "hover:border-rule-strong",
        ].join(" ")}
      >
        <div class="flex items-center gap-2">
          <span class="border-rule text-legend text-fg-dim border px-1 py-px tracking-[0.12em]">
            {a.fkey}
          </span>
          <span
            class={`text-label tracking-[0.16em] uppercase ${
              reachable ? "text-amber" : "text-fg-dim"
            }`}
          >
            {a.code}
          </span>
          <span class="flex-1" />
          <StateLamp
            tone={lampToneFor(a.state)}
            word={a.stateWord}
            pulse={a.state === "on-duty"}
            testId={`register-lamp-${a.slug}`}
          />
        </div>

        <p class="font-human text-body text-fg-mid leading-snug">{a.role}</p>

        <div class="border-rule mt-auto border-t pt-2">
          <Field label="Plan">
            {a.plan ? (
              <span class="inline-flex items-center gap-2">
                <span class="text-cyan">{a.plan.label}</span>
                <Gauge done={a.plan.done} total={a.plan.total} />
              </span>
            ) : (
              <span class="text-fg-dim">None yet</span>
            )}
          </Field>
          <Field label="Owns">
            {a.system ? (
              <span class="text-fg-mid">{a.system}</span>
            ) : (
              <span class="text-fg-dim">No business system</span>
            )}
          </Field>
          <Field label="Authority">
            <span class="text-fg-mid">{a.authority}</span>
          </Field>
          {a.demoTurnsToday !== null ? (
            <Field label="Turns today">
              <span
                class="text-fg-dim"
                title="Demonstration data — no turn has ever run."
              >
                {a.demoTurnsToday} · demo
              </span>
            </Field>
          ) : null}
        </div>
      </a>
    );
  },
);
