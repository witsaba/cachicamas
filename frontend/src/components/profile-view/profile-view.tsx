/**
 * ProfileView — presentational view of the authenticated user's profile.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-050 (S-FA-050..S-FA-058) — the `/profile` route renders the
 *   authenticated user's name and email; if no session is active, the
 *   route redirects to `/auth/signin` (or shows a sign-in CTA in the
 *   SSR-rendered HTML so search engines see a meaningful page).
 *
 * Why this is a pure component (no useSession here):
 *   Qwik City's `routeLoader$` (which `useSession` is) needs a request
 *   context that's not always available in the vitest `createDOM()`
 *   harness. By taking the session value as a prop, we let the
 *   caller (routes/profile/index.tsx) call `useSession()` in its own
 *   Qwik City request context, and let tests inject any session shape
 *   they want. The same pattern as SignInButton.
 *
 * Aphantasic-friendly (UX-4):
 *   Text-first. The user's name is the visual hero; the email is a
 *   secondary line; the avatar is shown ONLY when present and is
 *   meaningful (the user's own picture). No decorative icons.
 */
import { component$, type QRL } from "@builder.io/qwik";
import {
  SignInButton,
  type SignInActionLike,
} from "~/components/sign-in-button/sign-in-button";

/**
 * Mirror of the minimal Auth.js Session shape we depend on. Defined
 * locally (not imported from @auth/core) so the test doesn't have to
 * resolve the auth stack just to type a fixture.
 */
export interface ProfileUser {
  name?: string | null;
  email?: string | null;
  image?: string | null;
}

export interface ProfileSession {
  user?: ProfileUser | null;
}

export interface ProfileViewProps {
  session: ProfileSession | null;
  /**
   * Qwik City Action returned by `useSignIn()`. We pass it down so
   * the unauthenticated state can render the SignInButton without
   * re-importing the action.
   */
  signIn?: SignInActionLike;
  /** Optional QRL fired when the user clicks "Sign out". */
  onSignOut$?: QRL<() => unknown>;
}

/**
 * Renders either the authenticated profile or a "you're signed out"
 * CTA. Pure presentational — no side effects, no useSession, no
 * `useTask$` lifecycle.
 *
 * Strict TDD:
 *   T2.10 (RED) — this file did not exist; the test failed.
 *   T2.11 (GREEN) — this implementation; the test asserts:
 *     - The user's name and email render when session is non-null.
 *     - The SignInButton is shown when session is null.
 *     - A "Sign out" button is wired to the onSignOut$ callback.
 *     - The `<h1>` element carries the user's name as the brand-mark
 *       (F-1 / R-FA-058).
 */
export const ProfileView = component$<ProfileViewProps>(
  ({ session, signIn, onSignOut$ }) => {
    if (!session?.user) {
      // Signed-out state. Render a CTA + sign-in button so the page
      // still has a primary action.
      return (
        <main
          class="mx-auto max-w-xl px-6 py-12"
          data-testid="profile-signed-out"
        >
          <h1 class="mb-2 text-2xl font-semibold">You're signed out</h1>
          <p class="mb-6 text-sm text-zinc-400">
            Sign in to view your profile.
          </p>
          {signIn ? (
            <SignInButton signIn={signIn} label="Sign in to continue" />
          ) : null}
        </main>
      );
    }

    const userName = session.user.name ?? "";
    const userEmail = session.user.email ?? "";
    const userImage = session.user.image ?? "";

    return (
      <main class="mx-auto max-w-xl px-6 py-12" data-testid="profile-signed-in">
        <h1 class="mb-2 text-2xl font-semibold" data-testid="profile-name">
          {userName}
        </h1>
        <p class="mb-6 text-sm text-zinc-400" data-testid="profile-email">
          {userEmail}
        </p>
        {userImage ? (
          <img
            src={userImage}
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
            class="rounded-md border border-zinc-700 bg-zinc-900 px-4 py-2 text-sm font-medium text-zinc-100 hover:bg-zinc-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500"
            data-testid="profile-sign-out"
          >
            Sign out
          </button>
        ) : null}
      </main>
    );
  },
);
