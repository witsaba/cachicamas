/**
 * The icon set.
 *
 * Drawn here rather than pulled from a library, and drawn to ONE grid: a 24×24
 * box, 1.75px strokes, round caps and joins, no fills. A set assembled from
 * several sources is the tell that nobody decided; a set drawn to one grid is
 * the cheapest way a product looks made.
 *
 * Icons in this product are never the only carrier of meaning
 * (PRODUCT.md § Accessibility): every one of them ships beside a word. They are
 * therefore `aria-hidden` by default, and the label beside them is what a
 * screen reader announces.
 */
import { component$ } from "@builder.io/qwik";

export type IconName =
  | "desk"
  | "chat"
  | "staff"
  | "teams"
  | "org"
  | "settings"
  | "plus"
  | "check"
  | "arrow-right"
  | "stop"
  | "shield"
  | "send"
  | "clock"
  | "tool"
  | "chevron-down"
  | "external";

const PATHS: Record<IconName, string> = {
  // A desk seen from above: surface, and a drawer front.
  desk: "M3 10h18M4 10V7a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v3M5 10v8M19 10v8M8 14h4",
  // A conversation: one bubble with a tail.
  chat: "M20 12a7 7 0 0 1-7 7H8l-4 3v-4.4A7 7 0 0 1 6 6h7a7 7 0 0 1 7 6Z",
  // A person and a half: the directory.
  staff:
    "M10 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM3.5 20a6.5 6.5 0 0 1 13 0M17 11.5a3 3 0 0 0 0-6M18.5 20a5.5 5.5 0 0 0-2-4.3",
  // Three nodes joined: a team.
  teams:
    "M12 4.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5ZM5.5 14.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5ZM18.5 14.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5ZM10.4 9.4 7.1 14M13.6 9.4l3.3 4.6M8 17h8",
  // A hierarchy: one box over two.
  org: "M9.5 4h5v3.5h-5zM3.5 16.5h5V20h-5zM15.5 16.5h5V20h-5zM12 7.5v3.5M6 16.5V11h12v5.5",
  // A slider panel: settings that are set, not tuned.
  settings: "M4 8h10M18 8h2M4 16h4M12 16h8M15 5.5v5M9 13.5v5",
  plus: "M12 5v14M5 12h14",
  check: "M4.5 12.5 9.5 17.5 19.5 6.5",
  "arrow-right": "M4 12h15M13 6l6 6-6 6",
  stop: "M8 8h8v8H8z",
  // A shield: permission, and what an agent may not do.
  shield: "M12 3.5 5 6.2v5c0 4.4 2.9 8.1 7 9.3 4.1-1.2 7-4.9 7-9.3v-5Z",
  send: "M4.5 12 20 4.5 15.5 20l-4-6.5Z",
  clock: "M12 4.5a7.5 7.5 0 1 0 0 15 7.5 7.5 0 0 0 0-15ZM12 8v4.3l2.8 1.7",
  // A spanner head: the tools an agent is allowed to use.
  tool: "M15.5 4.5a4.5 4.5 0 0 0-5.8 5.6l-5.2 5.2a1.5 1.5 0 0 0 0 2.1l2.1 2.1a1.5 1.5 0 0 0 2.1 0l5.2-5.2a4.5 4.5 0 0 0 5.6-5.8l-2.9 2.9-2.5-.6-.6-2.5Z",
  "chevron-down": "M6 9.5 12 15.5 18 9.5",
  external:
    "M14 5h5v5M19 5l-7.5 7.5M17 14v4a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4",
};

export interface IconProps {
  readonly name: IconName;
  /** Rendered size in px. The grid is 24, so 16/18/20/24 stay crisp. */
  readonly size?: number;
  readonly class?: string;
}

export const Icon = component$<IconProps>(({ name, size = 18, ...props }) => (
  <svg
    width={size}
    height={size}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
    class={props.class}
  >
    <path d={PATHS[name]} />
  </svg>
));
