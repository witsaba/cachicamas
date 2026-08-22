/**
 * The lateral panel — the one piece of chrome that is always on screen.
 *
 * Three bands, in the order a person needs them:
 *
 *   1. The company. Whose workspace this is, and which plan it is on. It is at
 *      the top because the product's whole claim is that you are inside your
 *      own company, not inside a tool.
 *   2. Where you can go. Five destinations, no more — the moment a work tool's
 *      rail passes about seven items it stops being scannable and starts being
 *      a menu you read.
 *   3. Who you can talk to. The colleagues currently on staff, each one a
 *      direct link into a conversation, because starting a conversation is the
 *      single most common thing anyone does here.
 *
 * The rail takes the current section as a PROP rather than reading the router.
 * Each section's layout knows statically which section it is, which keeps this
 * component pure, testable without a request context, and free of the class of
 * bug where a highlight and a URL disagree.
 */
import { component$, type QRL } from "@builder.io/qwik";
import {
  AvatarDropdown,
  type AvatarDropdownSession,
} from "~/components/avatar-dropdown/avatar-dropdown";
import { Icon, type IconName } from "~/components/icon/icon";
import type { SignInActionLike } from "~/components/sign-in-button/sign-in-button";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { COMPANY } from "~/lib/mock/company";
import { AGENTS } from "~/lib/mock/staff";

export type WorkspaceSection =
  | "home"
  | "chat"
  | "agents"
  | "teams"
  | "organization"
  | "settings"
  | "profile";

interface NavEntry {
  readonly section: WorkspaceSection;
  readonly label: string;
  readonly href: string;
  readonly icon: IconName;
}

const NAV: readonly NavEntry[] = [
  { section: "home", label: "Front desk", href: "/home/", icon: "desk" },
  { section: "chat", label: "Chat", href: "/chat/", icon: "chat" },
  { section: "agents", label: "Agents", href: "/agents/", icon: "staff" },
  { section: "teams", label: "Teams", href: "/teams/", icon: "teams" },
  {
    section: "organization",
    label: "Organisation",
    href: "/organization/",
    icon: "org",
  },
];

const ROW =
  "flex items-center gap-2.5 rounded-md px-2 py-1.5 text-base transition-colors duration-150";
const ROW_REST = "text-ink-mid hover:bg-sunken hover:text-ink";
/** The current destination: the brand's tint, and the brand's ink. Selection
 *  is the only thing in the rail allowed to use the accent. */
const ROW_CURRENT = "bg-brand-tint font-medium text-brand";

export interface SidebarProps {
  readonly section: WorkspaceSection;
  readonly session: AvatarDropdownSession;
  readonly signOut: SignInActionLike;
  /** Closes the mobile drawer after a navigation. Absent on desktop. */
  readonly onNavigate$?: QRL<() => void>;
}

export const Sidebar = component$<SidebarProps>(
  ({ section, session, signOut, onNavigate$ }) => {
    // The colleagues you can actually open a conversation with, plus the ones
    // still being set up — because a colleague who is arriving is news, and
    // hiding them until they are ready would make the rail feel static.
    const onStaff = AGENTS.filter((a) => a.status !== "available");

    return (
      <div class="bg-surface flex h-full flex-col">
        {/* 1 — the company */}
        <div class="px-3 pt-3 pb-2">
          <a
            href="/home/"
            onClick$={onNavigate$}
            class="hover:bg-sunken flex items-center gap-2.5 rounded-md px-2 py-1.5 transition-colors duration-150"
            data-testid="sidebar-company"
          >
            <span
              aria-hidden="true"
              class="bg-ink text-ink-inverse inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-sm font-bold"
            >
              {COMPANY.initials}
            </span>
            <span class="min-w-0">
              <span class="text-ink block truncate text-base font-semibold">
                {COMPANY.name}
              </span>
              <span class="text-2xs text-ink-soft block truncate">
                {COMPANY.plan} plan · {COMPANY.people} people
              </span>
            </span>
          </a>
        </div>

        {/* 2 — where you can go */}
        <nav aria-label="Workspace" class="px-3">
          <ul class="space-y-0.5">
            {NAV.map((entry) => {
              const current = entry.section === section;
              return (
                <li key={entry.section}>
                  <a
                    href={entry.href}
                    onClick$={onNavigate$}
                    aria-current={current ? "page" : undefined}
                    data-testid={`nav-${entry.section}`}
                    class={[ROW, current ? ROW_CURRENT : ROW_REST].join(" ")}
                  >
                    <Icon name={entry.icon} size={18} class="shrink-0" />
                    {entry.label}
                  </a>
                </li>
              );
            })}
          </ul>
        </nav>

        {/* 3 — who you can talk to */}
        <div class="mt-6 min-h-0 flex-1 overflow-y-auto px-3 pb-3">
          <h2 class="text-2xs text-ink-soft px-2 pb-1.5 font-semibold tracking-wide uppercase">
            Your colleagues
          </h2>
          <ul class="space-y-0.5">
            {onStaff.map((agent) => (
              <li key={agent.slug}>
                <a
                  href={`/chat/?with=${agent.slug}`}
                  onClick$={onNavigate$}
                  data-testid={`sidebar-agent-${agent.slug}`}
                  class={[ROW, ROW_REST].join(" ")}
                >
                  <AgentAvatar agent={agent} size="sm" />
                  <span class="min-w-0 flex-1 truncate">{agent.name}</span>
                  {agent.status === "training" ? (
                    <span class="text-2xs text-waiting shrink-0">Training</span>
                  ) : null}
                </a>
              </li>
            ))}
          </ul>

          <a
            href="/agents/"
            onClick$={onNavigate$}
            class={[ROW, ROW_REST, "mt-1"].join(" ")}
            data-testid="sidebar-hire"
          >
            <span
              aria-hidden="true"
              class="border-line-control text-ink-soft inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-[0.3125rem] border border-dashed"
            >
              <Icon name="plus" size={14} />
            </span>
            Add a colleague
          </a>
        </div>

        {/* the person using it */}
        <div class="border-line border-t p-2">
          <AvatarDropdown session={session} signOut={signOut} />
        </div>
      </div>
    );
  },
);
