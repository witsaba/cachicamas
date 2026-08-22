/**
 * AvatarDropdown — the authenticated identity affordance.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-003..005 — the avatar trigger (S-AS-020), the panel entries
 *   (S-AS-040), the sign-out wiring (S-AS-041), and the URL
 *   sanitization (S-AS-022).
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §3
 *   Sequence diagram + Qwik lifecycle; §7 ADR-0009 (avatar URL
 *   sanitization); §8 Tailwind class patterns.
 *
 * Composition:
 *   - Uses `useClickOutside` for the open/close state machine
 *     (toggle on click, close on outside-mousedown, close on Escape).
 *   - Renders an `<img>` for the avatar only when the sanitized URL
 *     is non-null. The `safeAvatarSrc` allowlist (ADR-0009) prevents
 *     non-https URLs (e.g. `javascript:`) from reaching the DOM.
 *   - Renders the panel as `<div role="menu">` with three entries
 *     (Profile link, Sign out form). The Manage organizations link
 *     was removed in the 2026-07-06 ownboarding change.
 *   - Aphantasic-friendly (UX-4, amended 2026-07-04) — inside the
 *     panel, the entries are text-only EXCEPT for the Sign out
 *     affordance, which renders a Lucide-style "log-out" inline
 *     <svg> (door + arrow, MIT-licensed) as a functional visual
 *     anchor. The avatar image on the trigger is identity, not
 *     decoration. See `openspec/specs/app-shell/spec.md` UX-4
 *     glossary entry for the functional-icon carve-out.
 *
 * Test escape hatch:
 *   `forceOpen` is a test-only prop. Production consumers use
 *   `useClickOutside` exclusively. The prop lets the spec render
 *   the panel unconditionally without driving QRL click handlers
 *   (which the Qwik testing harness cannot synchronously fire).
 */
import { component$ } from "@builder.io/qwik";
import { Form } from "@builder.io/qwik-city";
import type { SignInActionLike } from "~/components/sign-in-button/sign-in-button";
import { Icon } from "~/components/icon/icon";
import { MenuItem } from "~/components/ui/menu-item/menu-item";
import { PersonAvatar } from "~/components/workspace/avatar/avatar";
import { COMPANY } from "~/lib/mock/company";
import { initialsOf } from "~/lib/initials";
import { safeAvatarSrc } from "~/lib/safe-avatar-src";
import { useClickOutside } from "./use-click-outside";

export interface AvatarDropdownSession {
  user?: {
    name?: string | null;
    email?: string | null;
    image?: string | null;
  } | null;
}

export interface AvatarDropdownProps {
  session: AvatarDropdownSession;
  /**
   * Qwik City Action returned by `useSignOut()`. Same loose-alias
   * pattern as `SignInButton` so the form submits to the Auth.js
   * signout endpoint.
   */
  signOut: SignInActionLike;
  /**
   * Test-only escape hatch. When `true`, the panel renders
   * unconditionally (the click-outside hook is bypassed). The
   * production consumer never sets this; it lives so the vitest
   * spec can assert the panel's content without driving QRL clicks.
   */
  forceOpen?: boolean;
}

export const AvatarDropdown = component$<AvatarDropdownProps>(
  ({ session, signOut, forceOpen = false }) => {
    const { open, setOpen$, triggerRef, panelRef } = useClickOutside();
    const user = session?.user ?? null;
    if (!user) {
      // Authenticated state is required. Anon visitors get the
      // SignInButton at the layout level (PR-3); we render nothing
      // here as a defensive null-guard.
      return null;
    }

    const userName = user.name ?? "";
    const userEmail = user.email ?? "";
    const safeImage = safeAvatarSrc(user.image);
    const isOpen = forceOpen || open.value;

    return (
      <div class="relative">
        <button
          ref={triggerRef}
          type="button"
          data-testid="avatar-dropdown"
          aria-haspopup="menu"
          aria-label={`${userName} menu`}
          aria-expanded={isOpen}
          onClick$={() => setOpen$(!open.value)}
          class="hover:bg-sunken flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors duration-150"
        >
          <PersonAvatar
            name={userName}
            initials={initialsOf(userName, userEmail)}
            image={safeImage}
            size="md"
          />
          <span class="min-w-0 flex-1">
            <span class="text-ink block truncate text-sm font-medium">
              {userName || userEmail}
            </span>
            <span class="text-2xs text-ink-soft block truncate">
              {COMPANY.name}
            </span>
          </span>
          <Icon name="chevron-down" size={16} class="text-ink-soft shrink-0" />
        </button>

        {isOpen ? (
          <div
            ref={panelRef}
            role="menu"
            data-testid="avatar-menu-panel"
            class="border-line bg-surface text-ink absolute bottom-[calc(100%+0.375rem)] left-0 z-50 w-60 rounded-lg border p-1 shadow-[var(--shadow-float)]"
          >
            <div class="border-line border-b px-2 pt-1.5 pb-2">
              <p
                class="text-ink truncate text-base font-medium"
                data-testid="avatar-menu-name"
              >
                {userName}
              </p>
              <p
                class="text-ink-soft truncate text-xs"
                data-testid="avatar-menu-email"
              >
                {userEmail}
              </p>
            </div>
            <ul class="py-1">
              <li>
                <MenuItem as="a" href="/profile/" testId="avatar-menu-profile">
                  Profile
                </MenuItem>
              </li>
              {/*
                2026-08-06 cachicamas-frontend-chat-layer1 (REQ-3):
                The avatar menu is the live authenticated navigation
                entry point for the chat. The design named
                `components/example/example.tsx` as the sidebar but
                the worktree has no sidebar component in the app
                shell — the override is documented in
                openspec/changes/cachicamas-frontend-chat-layer1/tasks.md
                §2.1 + §3 T-09. Auth gating is inherited from the
                dropdown's anon-render-nothing branch (top of this
                component) + the /chat route's onRequest guard.
              */}
              <li>
                <MenuItem as="a" href="/chat/" testId="avatar-menu-chat">
                  Chat
                </MenuItem>
              </li>
              <li>
                <MenuItem
                  as="a"
                  href="/settings/"
                  testId="avatar-menu-settings"
                >
                  Settings
                </MenuItem>
              </li>
            </ul>
            <Form
              action={signOut}
              data-testid="avatar-menu-signout-form"
              class="border-line border-t pt-1"
            >
              {/*
                    R-HP-005 (S-HP-040): after sign-out the user lands on
                    the Auth.js sign-in surface (/auth/signin), not the
                    public landing. See proposal decision #5.
                  */}
              <input type="hidden" name="redirectTo" value="/auth/signin" />
              <MenuItem
                type="submit"
                testId="avatar-menu-signout"
                class="inline-flex w-full items-center gap-2 text-left"
              >
                {/*
                    Functional visual anchor for "log out" (UX-4
                    amendment, 2026-07-04) — Lucide's log-out icon
                    (door + arrow, MIT-licensed). aria-hidden because
                    the visible <span>Sign out</span> text label
                    already announces the affordance. stroke="currentColor"
                    so the icon inherits the button's text color in
                    both light (slate-700) and dark contexts.
                  */}
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  width="14"
                  height="14"
                  aria-hidden="true"
                  focusable="false"
                  data-testid="sign-out-icon"
                >
                  <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                  <polyline points="16 17 21 12 16 7" />
                  <line x1="21" y1="12" x2="9" y2="12" />
                </svg>
                <span>Sign out</span>
              </MenuItem>
            </Form>
          </div>
        ) : null}
      </div>
    );
  },
);
