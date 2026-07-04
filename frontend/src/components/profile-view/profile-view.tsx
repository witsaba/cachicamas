/**
 * ProfileView — presentational view of the authenticated user's profile.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/profile-home/spec.md`
 *   R-PH-001 — renders the user's name as <h1> (the brand mark).
 *   R-PH-002 — renders email and avatar (when present).
 *   R-PH-003 — renders a github.com anchor when github_login is set.
 *   R-PH-004 — renders the SignOutButton when onSignOut$ is provided.
 *   R-PH-005 — renders a "Manage organizations" link to /organizations/new.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-022 / ADR-0009 — non-https avatar URLs are rejected via
 *   `safeAvatarSrc()` (PR-2a #34). The avatar <img> is omitted
 *   entirely when the URL is unsafe.
 *
 * Why this is a pure component (no useSession here):
 *   Qwik City's `routeLoader$` (which `useSession` is) needs a
 *   request context that's not always available in the vitest
 *   `createDOM()` harness. By taking the session value as a prop,
 *   we let the caller (`routes/profile/index.tsx`) call
 *   `useSession()` in its own Qwik City request context, and let
 *   tests inject any session shape they want. The same pattern as
 *   SignInButton.
 *
 * Aphantasic-friendly (UX-4, amended 2026-07-04):
 *   Text-first. The user's name is the visual hero; the email is a
 *   secondary line; the avatar <img> is the user's own identity
 *   (per UX-4, identity imagery is the one allowed <img>). The
 *   Sign out button renders a Lucide-style "log-out" inline <svg>
 *   (door + arrow, MIT-licensed) alongside the label as a
 *   functional visual anchor — same UX-4 carve-out as the
 *   SignInButton brand mark. No decorative iconography elsewhere.
 *   decorative icons.
 *
 * PR-4 history:
 *   T2.10/T2.11 — initial implementation (name, email, avatar,
 *                  sign-out button).
 *   T4.1 — added `github_login` field + `profile-github-login`
 *          anchor (R-PH-003).
 *   T4.2 — added `profile-manage-orgs` link (R-PH-005) and reused
 *          `safeAvatarSrc()` from PR-2a (#34) for the avatar URL.
 *   T4.3 — dropped the signed-out branch (T4.3); /profile now
 *          uses `SignInRequiredCard` (PR-1b #33) for anon, so the
 *          component's signed-out branch was unreachable. ProfileView
 *          now treats `session.user === null/empty` as "render
 *          nothing" — the parent route owns the anon surface.
 */
import { component$, type QRL } from "@builder.io/qwik";
import { safeAvatarSrc } from "~/lib/safe-avatar-src";

/**
 * Mirror of the minimal Auth.js Session shape we depend on. Defined
 * locally (not imported from `@auth/core`) so the test doesn't have
 * to resolve the auth stack just to type a fixture.
 */
export interface ProfileUser {
  name?: string | null;
  email?: string | null;
  image?: string | null;
  /**
   * The user's GitHub login, populated by the Auth.js GitHub
   * provider. Distinct from `name` (the user's display name) —
   * "octocat" vs "The Octocat". When set, we render an external
   * anchor that opens https://github.com/{login}.
   */
  github_login?: string | null;
}

export interface ProfileSession {
  user?: ProfileUser | null;
}

export interface ProfileViewProps {
  session: ProfileSession | null;
  /** Optional QRL fired when the user clicks "Sign out". */
  onSignOut$?: QRL<() => unknown>;
}

/**
 * Renders the authenticated profile, or nothing if there is no
 * authenticated session. The parent route handles the anon case
 * via `SignInRequiredCard` (PR-1b), so this component no longer
 * renders its own signed-out CTA (T4.3 — the branch was
 * unreachable post PR-1b).
 */
export const ProfileView = component$<ProfileViewProps>(
  ({ session, onSignOut$ }) => {
    // T4.3: no signed-out branch. If session.user is missing, the
    // route has already rendered SignInRequiredCard, so we just
    // produce an empty root. Returning null from a component is
    // the idiomatic Qwik pattern for "no content here".
    if (!session?.user) {
      return null;
    }

    const user = session.user;
    const userName = user.name ?? "";
    const userEmail = user.email ?? "";
    const userGithubLogin = user.github_login ?? "";
    // Reuse the safeAvatarSrc allowlist (PR-2a #34). The same
    // helper backs the AvatarDropdown's avatar; keeping the helper
    // in one place means a future tightening of the allowlist
    // (e.g., add a domain blocklist) applies at both surfaces at
    // once.
    const safeImage = safeAvatarSrc(user.image);

    return (
      <main class="mx-auto max-w-xl px-6 py-12" data-testid="profile-signed-in">
        <h1 class="mb-2 text-2xl font-semibold" data-testid="profile-name">
          {userName}
        </h1>
        <p class="mb-1 text-sm text-zinc-400" data-testid="profile-email">
          {userEmail}
        </p>
        {userGithubLogin ? (
          <p class="mb-6 text-sm">
            <a
              href={`https://github.com/${encodeURIComponent(userGithubLogin)}`}
              target="_blank"
              rel="noopener noreferrer"
              class="font-mono text-sm text-zinc-300 underline hover:text-zinc-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500"
              data-testid="profile-github-login"
            >
              @{userGithubLogin}
            </a>
          </p>
        ) : null}
        {safeImage ? (
          <img
            src={safeImage}
            alt=""
            width={64}
            height={64}
            class="mb-6 h-16 w-16 rounded-full"
            data-testid="profile-image"
          />
        ) : null}
        {onSignOut$ ? (
          <button
            type="button"
            onClick$={onSignOut$}
            class="mb-6 inline-flex cursor-pointer items-center gap-2 rounded-md border border-zinc-700 bg-zinc-900 px-4 py-2 text-sm font-medium text-zinc-100 shadow-sm transition-[background-color,box-shadow,transform,border-color] duration-150 hover:border-zinc-600 hover:bg-zinc-800 hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500 active:translate-y-px"
            data-testid="profile-sign-out"
          >
            {/*
                  Functional visual anchor for "log out" (UX-4
                  amendment, 2026-07-04) — Lucide's log-out icon
                  (door + arrow, MIT-licensed). aria-hidden because
                  the visible <span>Sign out</span> text label
                  already announces the affordance. fill="none" +
                  stroke="currentColor" matches the menu-item style
                  from AvatarDropdown.
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
        ) : null}
        <hr class="my-4 border-zinc-800" />
        <a
          href="/organizations/new"
          class="inline-block rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium text-zinc-100 underline hover:bg-zinc-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500"
          data-testid="profile-manage-orgs"
        >
          Manage organizations
        </a>
      </main>
    );
  },
);
