/**
 * Avatars — and the one rule in this product that carries meaning through form.
 *
 * A PERSON IS A CIRCLE. AN AGENT IS A ROUNDED SQUARE.
 *
 * That distinction exists because the whole premise of the product is that you
 * work alongside colleagues who are not people, and a directory where the two
 * are indistinguishable would be dishonest in a way no amount of copy fixes.
 * It is also the reason the rule never travels alone: shape is a signal, not a
 * statement, so every agent avatar in the product ships with the literal word
 * "Agent" beside it (see `<SpeciesLabel>` below and PRODUCT.md § Accessibility).
 *
 * Department colour identifies; it never ranks and it never means a status.
 * The department name is always written next to it somewhere on the same
 * screen.
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

export interface PersonAvatarProps {
  readonly name: string;
  readonly initials: string;
  /** An already-validated image URL, or null. */
  readonly image?: string | null;
  readonly size?: AvatarSize;
}

/** The circle. Never used for an agent. */
export const PersonAvatar = component$<PersonAvatarProps>(
  ({ name, initials, image = null, size = "md" }) =>
    image ? (
      <img
        src={image}
        alt=""
        width={56}
        height={56}
        data-species="person"
        aria-hidden="true"
        class={[
          "border-line inline-block shrink-0 rounded-full border object-cover",
          BOX[size],
        ].join(" ")}
      />
    ) : (
      <span
        data-species="person"
        data-name={name}
        aria-hidden="true"
        class={[
          "bg-sunken text-ink-mid ring-line inline-flex shrink-0 items-center justify-center rounded-full font-semibold ring-1",
          BOX[size],
        ].join(" ")}
      >
        {initials}
      </span>
    ),
);

/**
 * The word that keeps the shape rule honest. Rendered wherever an avatar
 * appears without its full profile around it.
 */
export const SpeciesLabel = component$<{ species: "agent" | "person" }>(
  ({ species }) => (
    <span
      class={[
        "text-2xs inline-flex items-center rounded-sm border px-1.5 py-px font-semibold tracking-wide uppercase",
        species === "agent"
          ? "border-brand/25 bg-brand-tint text-brand"
          : "border-line bg-sunken text-ink-soft",
      ].join(" ")}
    >
      {species === "agent" ? "Agent" : "Person"}
    </span>
  ),
);
