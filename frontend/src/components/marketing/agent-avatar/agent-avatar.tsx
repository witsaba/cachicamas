/**
 * Avatars — and the one rule in this product that carries meaning through form.
 *
 * A PERSON IS A CIRCLE. AN AGENT IS A ROUNDED SQUARE.
 *
 * That distinction exists because the whole premise of the product is that you
 * work alongside colleagues who are not people, and a directory where the two
 * are indistinguishable would be dishonest in a way no amount of copy fixes.
 *
 * Marketing-only re-export: only `AgentAvatar` is kept — `PersonAvatar` lived
 * alongside it but was auth-shaped (it consumed the deleted `safe-avatar-src`
 * helper) and so is gone with the rest of the auth surface.
 */
import { component$ } from "@builder.io/qwik";
import type { Agent, Department } from "~/lib/mock/staff";

/** Tailwind needs literal class strings, so the map is spelled out. */
const DEPARTMENT_FILL: Record<Department, string> = {
  assistant: "bg-dept-assistant",
  finance: "bg-dept-finance",
  support: "bg-dept-support",
  integrations: "bg-dept-integrations",
  data: "bg-dept-data",
  engineering: "bg-dept-engineering",
};

const DEPARTMENT_INK: Record<Department, string> = {
  assistant: "text-dept-assistant",
  finance: "text-dept-finance",
  support: "text-dept-support",
  integrations: "text-dept-integrations",
  data: "text-dept-data",
  engineering: "text-dept-engineering",
};

export const departmentFill = (d: Department): string => DEPARTMENT_FILL[d];
export const departmentInk = (d: Department): string => DEPARTMENT_INK[d];

export type AvatarSize = "sm" | "md" | "lg" | "xl";

const BOX: Record<AvatarSize, string> = {
  sm: "h-6 w-6 text-[0.5625rem]",
  md: "h-8 w-8 text-2xs",
  lg: "h-10 w-10 text-xs",
  xl: "h-14 w-14 text-base",
};

const AGENT_RADIUS: Record<AvatarSize, string> = {
  sm: "rounded-[0.3125rem]",
  md: "rounded-[0.375rem]",
  lg: "rounded-[0.4375rem]",
  xl: "rounded-[0.625rem]",
};

export interface AgentAvatarProps {
  readonly agent: Agent;
  readonly size?: AvatarSize;
}

/** The rounded square. Never used for a person. */
export const AgentAvatar = component$<AgentAvatarProps>(
  ({ agent, size = "md" }) => (
    <span
      data-testid={`agent-avatar-${agent.slug}`}
      data-species="agent"
      aria-hidden="true"
      class={[
        "text-ink-inverse inline-flex shrink-0 items-center justify-center font-semibold tracking-wide",
        BOX[size],
        AGENT_RADIUS[size],
        DEPARTMENT_FILL[agent.department],
      ].join(" ")}
    >
      {agent.initials}
    </span>
  ),
);