/**
 * Status — a dot and its word, never a dot alone.
 *
 * Three states, and each one answers a different question a manager actually
 * asks: is this colleague answering right now, is it still being set up, or is
 * it something we could add. The colour is a fast second read; the word is the
 * one that carries.
 */
import { component$ } from "@builder.io/qwik";
import type { AgentStatus } from "~/lib/mock/staff";

const DOT: Record<AgentStatus, string> = {
  working: "bg-ok",
  training: "bg-waiting",
  available: "bg-idle",
};

const INK: Record<AgentStatus, string> = {
  working: "text-ok",
  training: "text-waiting",
  available: "text-ink-soft",
};

export interface StatusProps {
  readonly status: AgentStatus;
  readonly word: string;
  /** Adds the explaining sentence under the word. Used on profiles. */
  readonly detail?: string;
}

export const Status = component$<StatusProps>(({ status, word, detail }) => (
  <span class="inline-flex items-baseline gap-1.5">
    <span
      aria-hidden="true"
      class={[
        "relative top-px inline-block h-2 w-2 shrink-0 rounded-full",
        DOT[status],
        status === "working" ? "work-pulse" : "",
      ].join(" ")}
    />
    <span class="text-xs font-medium">
      <span class={INK[status]}>{word}</span>
      {detail ? <span class="text-ink-soft"> · {detail}</span> : null}
    </span>
  </span>
));
