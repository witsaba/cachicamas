/**
 * ProfileView — presentational view of the authenticated user's
 * `/profile` page. Renders a welcoming "home base" for the user
 * instead of dumping the same identity data (name, email, avatar)
 * that the shell's AvatarDropdown already shows in its panel
 * header.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/profile-home/spec.md`
 *   R-PH-008 — the page opens with a personalized welcome heading.
 *   R-PH-003 — renders a github.com anchor when github_login is set.
 *   R-PH-005 — REMOVED 2026-07-06 ownboarding. The /organizations
 *     surface was deleted (single-tenant model). Users reach the
 *     setup page via /ownboarding on first sign-in only.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/app-shell/spec.md`
 *   R-AS-005 — the shell's AvatarDropdown already shows the user's
 *     name + email in its panel header. ProfileView does NOT
 *     duplicate that identity rendering; the welcome heading uses
 *     the first name only as a personalization cue.
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
 *   Text-first, no decorative iconography. The welcome heading is
 *   the visual hero; the action links are clearly labeled. No
 *   <img>, no <svg>. The avatar <img> lives in the shell's
 *   AvatarDropdown trigger (one allowed <img> per UX-4 — identity,
 *   not decoration).
 *
 * PR-4 history:
 *   T2.10/T2.11 — initial implementation (name, email, avatar,
 *                  sign-out button).
 *   T4.1 — added `github_login` field + `profile-github-login`
 *          anchor (R-PH-003).
 *   T4.2 — REMOVED 2026-07-06 ownboarding. The `profile-manage-orgs`
 *     link was part of the /profile enrichment. After the ownboarding
 *     change, the /organizations surface is gone; the link is replaced
 *     by an implicit redirect to /home (which itself gates on
 *     setup-state).
 *   T4.3 — dropped the signed-out branch (T4.3); /profile uses
 *          `SignInRequiredCard` (PR-1b #33) for anon.
 *
 * PR-4 enrichment completion (2026-07-04, UAT-9):
 *   The original PR-4 left /profile as a thin identity dump:
 *   h1=name, p=email, p=github_login, img=avatar. But the avatar
 *   dropdown (rendered by the shell on every signed-in page)
 *   already shows name + email in its panel header — so the
 *   /profile rendering was duplicative chrome. The redesign
 *   replaces the data dump with a proper "welcome home" UX:
 *     - H1 personalized welcome ("Welcome back, {first-name}!")
 *       uses the first name only as a personalization cue, not a
 *       redundant identity display
 *     - one-sentence body explaining what cachicamas is and what
 *       the user can do from here
 *     - two action links: "Manage organizations →" (primary,
 *       internal) and "github.com/{login} ↗" (secondary, external
 *       via github_login if present)
 *   The avatar <img>, email <p>, and name <p> (separate from the
 *   welcome h1) are all removed. `safeAvatarSrc` import is no
 *   longer needed and was dropped. The `data-testid="profile-image"`,
 *   `data-testid="profile-name"`, `data-testid="profile-email"`
 *   markers are retired; e2e assertions that referenced them were
 *   updated in the same PR.
 */
import { component$ } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

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
 * Extracts the first name from a full display name. The full name
 * may be a single token ("octocat"), two tokens ("Braejan Arias"),
 * or more. We split on whitespace and take the first non-empty
 * token. If the name is missing or empty, returns "" so the
 * welcome heading falls back to a generic "Welcome back!" without
 * a dangling space.
 */
function firstName(fullName: string | null | undefined): string {
  if (!fullName) return "";
  const trimmed = fullName.trim();
  if (!trimmed) return "";
  const first = trimmed.split(/\s+/)[0];
  return first ?? "";
}

export const ProfileView = component$<ProfileViewProps>(({ session }) => {
  // T4.3: no signed-out branch. If session.user is missing, the
  // route has already rendered SignInRequiredCard, so we just
  // produce an empty root. Returning null from a component is
  // the idiomatic Qwik pattern for "no content here".
  if (!session?.user) {
    return null;
  }

  const user = session.user;
  const userGithubLogin = user.github_login ?? "";
  const greetingName = firstName(user.name);

  return (
    <main class="mx-auto max-w-xl px-6 py-12" data-testid="profile-signed-in">
      {/*
        Welcome heading (R-PH-008). Personalized to the user's first
        name (a personalization cue, not a redundant identity display
        — the avatar dropdown's panel header already shows the full
        name + email). Falls back to a generic "Welcome back!" when
        the name is missing.
      */}
      <h1
        class="text-3xl font-bold tracking-tight text-slate-900"
        data-testid="profile-welcome"
      >
        {greetingName ? `Welcome back, ${greetingName}!` : "Welcome back!"}
      </h1>

      <p
        class="mt-4 max-w-lg text-base text-slate-700"
        data-testid="profile-body"
      >
        cachicamas is your home for tracking your organization's projects,
        requirements, and milestones. From here you can open your GitHub
        profile.
      </p>

      <div class="mt-8 flex flex-wrap gap-3">
        {/* Secondary action: external link to the user's GitHub
            profile. Renders only when github_login is set (the
            Auth.js GitHub provider populates it from the OAuth
            userinfo payload). */}
        {userGithubLogin ? (
          <Button
            as="a"
            href={`https://github.com/${encodeURIComponent(userGithubLogin)}`}
            target="_blank"
            rel="noopener noreferrer"
            size="lg"
            variant="secondary"
            testId="profile-github-login"
          >
            github.com/{userGithubLogin}
            <span aria-hidden="true">↗</span>
          </Button>
        ) : null}
      </div>
    </main>
  );
});
