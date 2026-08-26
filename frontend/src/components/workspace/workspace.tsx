/**
 * The workspace shell.
 *
 * A persistent lateral panel and a white content column, which is the layout
 * every modern work tool uses because it is the one that survives both a
 * five-item rail and a fifty-message conversation. Nothing here is invented.
 *
 * Two shapes, one component:
 *   - From `lg` up, the rail is always present and never moves.
 *   - Below `lg` it becomes a drawer behind a labelled button, and the content
 *     column takes the whole width. A rail that shrinks to icons is a rail
 *     nobody can read; a rail that disappears is one you can get back.
 *
 */
import { Slot, component$, useSignal, useStyles$ } from "@builder.io/qwik";
import type { AvatarDropdownSession } from "~/components/avatar-dropdown/avatar-dropdown";
import type { SignInActionLike } from "~/components/sign-in-button/sign-in-button";
import {
  Sidebar,
  type WorkspaceSection,
} from "~/components/workspace/sidebar/sidebar";
import { COMPANY } from "~/lib/mock/company";

export interface WorkspaceProps {
  readonly section: WorkspaceSection;
  readonly session: AvatarDropdownSession;
  readonly signOut: SignInActionLike;
  /**
   * Screens that own their own scrolling — the conversation — opt out of the
   * shell's scroll container so the composer can stay pinned to the bottom.
   */
  readonly fills?: boolean;
}

export const Workspace = component$<WorkspaceProps>(
  ({ section, session, signOut, fills = false }) => {
    const drawerOpen = useSignal(false);

    // Defence in depth. The section layouts already branch on the session, but
    // a company rail belonging to nobody is worse than no rail at all: it
    // implies a signed-in state that is not there. If this component is ever
    // reached without a person, it renders the page and no chrome.
    const person = session?.user ?? null;

    // The drawer's scrim fades; the panel slides. 180ms, because a person
    // opening a navigation drawer is mid-task and waiting for it.
    useStyles$(`
      @keyframes ws-scrim { from { opacity: 0 } to { opacity: 1 } }
      @keyframes ws-panel { from { transform: translateX(-100%) } to { transform: none } }
    `);

    if (!person) {
      return (
        <main id="main" class="bg-canvas min-h-screen">
          <Slot />
        </main>
      );
    }

    return (
      <div class="bg-canvas flex h-screen w-full overflow-hidden">
        {/* the rail, from lg up */}
        <aside
          class="border-line hidden w-[16rem] shrink-0 border-r lg:block"
          data-testid="workspace-rail"
        >
          <Sidebar section={section} session={session} signOut={signOut} />
        </aside>

        {/* the rail, below lg */}
        {drawerOpen.value ? (
          <div
            class="fixed inset-0 z-50 lg:hidden"
            data-testid="workspace-drawer"
          >
            <button
              type="button"
              aria-label="Close navigation"
              onClick$={() => (drawerOpen.value = false)}
              class="bg-ink/35 absolute inset-0 h-full w-full cursor-default"
              style="animation: ws-scrim 180ms var(--ease-work) both"
            />
            <div
              class="border-line absolute inset-y-0 left-0 w-[17rem] border-r shadow-[var(--shadow-float)]"
              style="animation: ws-panel 180ms var(--ease-work) both"
            >
              <Sidebar
                section={section}
                session={session}
                signOut={signOut}
                onNavigate$={() => (drawerOpen.value = false)}
              />
            </div>
          </div>
        ) : null}

        <div class="flex min-w-0 flex-1 flex-col">
          {/* the mobile bar */}
          <div class="border-line bg-surface flex items-center gap-2 border-b px-3 py-2 lg:hidden">
            <button
              type="button"
              onClick$={() => (drawerOpen.value = true)}
              data-testid="workspace-menu"
              class="border-line-control bg-surface text-ink inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-sm font-medium"
            >
              <span aria-hidden="true" class="flex flex-col gap-[3px]">
                <span class="block h-px w-3.5 bg-current" />
                <span class="block h-px w-3.5 bg-current" />
                <span class="block h-px w-3.5 bg-current" />
              </span>
              Menu
            </button>
            <span class="text-ink truncate text-base font-semibold">
              {COMPANY.name}
            </span>
          </div>

          <main
            id="main"
            class={
              fills
                ? "flex min-h-0 flex-1 flex-col overflow-hidden"
                : "min-h-0 flex-1 overflow-y-auto"
            }
          >
            <Slot />
          </main>
        </div>
      </div>
    );
  },
);
