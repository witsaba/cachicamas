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
 *     (Profile link, Manage organizations link, Sign out form).
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
      <div class="relative inline-block">
        <button
          ref={triggerRef}
          type="button"
          class="h-10 w-10 cursor-pointer overflow-hidden rounded-full ring-1 ring-slate-200 transition-[box-shadow,transform] duration-150 hover:shadow-md hover:ring-slate-400 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 active:scale-95"
          aria-haspopup="menu"
          aria-label={`${userName} menu`}
          aria-expanded={isOpen}
          data-testid="avatar-dropdown"
          onClick$={() => setOpen$(!open.value)}
        >
          {safeImage ? (
            <img
              src={safeImage}
              alt=""
              width={40}
              height={40}
              class="h-full w-full object-cover"
              data-testid="avatar-image"
            />
          ) : null}
        </button>

        {isOpen ? (
          <div
            ref={panelRef}
            role="menu"
            data-testid="avatar-menu-panel"
            class="absolute top-12 right-0 z-50 w-72 rounded-lg border border-slate-200 bg-white p-4 text-slate-900 shadow-lg ring-1 ring-black/5"
          >
            <p class="font-semibold" data-testid="avatar-menu-name">
              {userName}
            </p>
            <p class="text-sm text-slate-500" data-testid="avatar-menu-email">
              {userEmail}
            </p>
            <hr class="my-3 border-slate-200" />
            <ul class="space-y-1">
              <li>
                <a
                  href="/profile"
                  data-testid="avatar-menu-profile"
                  class="block rounded px-2 py-1.5 text-sm hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                >
                  Profile
                </a>
              </li>
              <li>
                <a
                  href="/organizations/new"
                  data-testid="avatar-menu-orgs"
                  class="block rounded px-2 py-1.5 text-sm hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                >
                  Manage organizations
                </a>
              </li>
            </ul>
            <hr class="my-3 border-slate-200" />
            <Form action={signOut} data-testid="avatar-menu-signout-form">
              {/*
                    R-HP-005 (S-HP-040): after sign-out the user lands on
                    the Auth.js sign-in surface (/auth/signin), not the
                    public landing. See proposal decision #5.
                  */}
              <input type="hidden" name="redirectTo" value="/auth/signin" />
              <button
                type="submit"
                data-testid="avatar-menu-signout"
                class="inline-flex w-full cursor-pointer items-center gap-2 rounded px-2 py-1.5 text-left text-sm text-slate-700 transition-[background-color,box-shadow,transform,border-color] duration-150 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 active:translate-y-px"
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
              </button>
            </Form>
          </div>
        ) : null}
      </div>
    );
  },
);
