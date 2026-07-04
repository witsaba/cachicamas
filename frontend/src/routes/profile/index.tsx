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
import { $, component$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";
import { useSession, useSignOut } from "~/routes/plugin@auth";
import { ProfileView } from "~/components/profile-view/profile-view";

export default component$(() => {
  const sessionSig = useSession();
  const signOutAction = useSignOut();
  // Wrap the Qwik Action's `.submit()` so the click handler has the
  // FormData shape Auth.js expects. Auth.js looks at the form fields
  // for the optional `redirectTo` (post-signout destination); we
  // default to `/` (the landing page).
  const onSignOut$ = $(() => {
    const fd = new FormData();
    fd.set("redirectTo", "/");
    return signOutAction.submit(fd);
  });
  return <ProfileView session={sessionSig.value} onSignOut$={onSignOut$} />;
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
