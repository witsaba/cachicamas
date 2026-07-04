/**
 * `/profile` — the canonical signed-in profile page.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-050 (S-FA-050..S-FA-058) — `/profile` renders the authenticated
 *   user's name (h1) + email, plus a Sign-out button. If the session
 *   is null, the page renders the SignInButton CTA.
 *
 * Design: see `openspec/changes/cachicamas-github-login/design.md` §2.3
 * for the sign-out sequence diagram.
 *
 * Why this is a thin wrapper over `ProfileView`:
 *   The Qwik City `useSession()` routeLoader runs in the request
 *   context (cookie-parse, JWT verify). That's not testable in the
 *   vitest `createDOM()` harness. By keeping all the rendering in
 *   `components/profile-view/profile-view.tsx` and just plumbing the
 *   session signal here, the unit test owns the presentation layer
 *   and the integration tests own the wiring.
 *
 * PR-2 isolation note:
 *   This slice reads `useSession()` ONLY. PR-3 will add a
 *   `routeLoader$` that hits the Go-side `/api/whoami` endpoint to
 *   enrich the profile with server-known fields (organization
 *   membership, etc.). Until then, the displayed name/email comes
 *   from the JWT cookie payload, which Auth.js populates from the
 *   GitHub OAuth userinfo endpoint.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";
import { ProfileView } from "~/components/profile-view/profile-view";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  // Guard first. The card is the anon surface; `ProfileView` is the
  // authenticated one. We pass `useSignIn` straight through to the
  // card so the post-signin roundtrip still works.
  const guard = requireSession(sessionSig.value, "/profile");
  if (guard.kind === "anon") {
    return (
      <SignInRequiredCard
        signIn={signInAction}
        description="Sign in to view your profile."
        redirectTo={guard.pathname}
      />
    );
  }
  // UAT-5 (2026-07-04): ProfileView no longer takes an `onSignOut$`
  // callback — the avatar dropdown's Sign out entry (rendered by
  // routes/layout.tsx via AvatarDropdown) is the sole sign-out
  // affordance. Removing the duplication eliminated a redundant
  // button from /profile. We deliberately do NOT call `useSignOut`
  // here any more — the dropdown owns the action wiring.
  return <ProfileView session={sessionSig.value} />;
});

export const head: DocumentHead = {
  title: "Profile \u2014 Cachicamas",
  meta: [
    {
      name: "description",
      content: "Your cachicamas profile, signed in via GitHub.",
    },
  ],
};
