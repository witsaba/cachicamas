/**
 * ProfileView — presentational view of the authenticated user's profile.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/profile-home/spec.md`
 *   R-PH-001 — renders the user's name as <h1> (the brand mark).
 *   R-PH-002 — renders email and avatar (when present).
 *   R-PH-003 — renders a github.com anchor when github_login is set.
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
 *   (per UX-4, identity imagery is the one allowed <img>). No
 *   decorative iconography elsewhere.
 *
 * UAT-5 (2026-07-04): this component previously rendered a Sign out
 * button alongside the avatar block. That affordance is REDUNDANT
 * with the avatar dropdown's Sign out entry (R-AS-005 + S-AS-042 in
 * `app-shell/spec.md`), which is reachable from every signed-in
 * surface via the shell's identity affordance. Keeping two sign-out
 * affordances was duplicative chrome. The profile's Sign out button
 * has been REMOVED; the avatar dropdown is the sole sign-out entry
 * point. The `onSignOut$` prop + the Lucide log-out icon + the
 * `data-testid="profile-sign-out"` were all retired with it.
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
import { component$ } from "@builder.io/qwik";
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
}

/**
 * Renders the authenticated profile, or nothing if there is no
 * authenticated session. The parent route handles the anon case
 * via `SignInRequiredCard` (PR-1b), so this component no longer
 * renders its own signed-out CTA (T4.3 — the branch was
 * unreachable post PR-1b).
 */
export const ProfileView = component$<ProfileViewProps>(({ session }) => {
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
      {/*
          UAT-5 (2026-07-04): the profile no longer renders its own
          Sign out button — the avatar dropdown's Sign out entry
          (data-testid="avatar-menu-signout" in the shell's identity
          affordance) is the sole sign-out entry point. Keeping the
          affordance only in the dropdown ensures there's one
          discoverable sign-out path from any signed-in surface.
          The `data-testid="profile-sign-out"` affordance was
          retired with this change.
        */}
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
});
