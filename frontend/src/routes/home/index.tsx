/**
 * `/home` — the canonical signed-in Home Page.
 *
 * Reference: `openspec/changes/home-page-placeholder/specs/home-page/spec.md`
 *   R-HP-001 (S-HP-001..S-HP-004) — personalised greeting for authed users.
 *   R-HP-002 (S-HP-010..S-HP-012) — single paragraph placeholder, no imagery.
 *   R-HP-003 (S-HP-020..S-HP-023) — anonymous renders SignInRequiredCard.
 *
 * Reference: `openspec/changes/home-page-placeholder/design.md` §T1, §T2, §T3
 *   T1 (inline JSX) — authed render is one <h1> + one <p>, so an extracted
 *       HomeView component is overkill at this size.
 *   T2 (routeLoader$ + requireSession) — anon visitors never receive the
 *       Home Page HTML; the loader runs in the request context.
 *   T3 (personalised greeting with fallback) — heading reads
 *       "Welcome, <name>" when the OAuth `name` claim is non-empty,
 *       otherwise "Welcome" (no trailing comma, no name token).
 *
 * Aphantasic-friendly (UX-4, archived spec, amended 2026-07-04):
 *   Text-first. No imagery. No <svg>, no <img>, no <picture>. The
 *   placeholder paragraph carries the copy.
 *
 * Why this mirrors `routes/profile/index.tsx`:
 *   Same protected-route shape. The structural test in
 *   `routes/home/route-guard.spec.ts` asserts the wiring (R-PR-003
 *   pattern). The behavioural coverage lives in
 *   `routes/home/index.spec.tsx` (mocked `useSession()`).
 */
import { component$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// 2026-07-06 native-auth-UI: anon visitors are redirected to the
// native /auth/signin page BEFORE this route's component renders.
// The redirect carries `?callbackUrl=/home` so Auth.js returns the
// visitor here after signing in. The inline SignInRequiredCard below
// is defence-in-depth — the onRequest throws first, so this branch
// never fires in production, but we keep it as a fallback for tests
// that render the component$ directly (without the HTTP middleware).
export { requireAuthRedirect as onRequest };

// 2026-07-06 ownboarding (R-OW-007 / S-OW-060..062): SSR-time
// setup-state guard. Reads GET /setup-state and throws a 302 redirect
// to /ownboarding when hasOrganization=false. On transport error or
// hasOrganization=true, the loader is a no-op (optimistic fallback).
// See `lib/require-ownboarding.ts` for the no-redirect-loop guard.
export const useSetupLoader = routeLoader$(async (event) => {
  await requireOwnboarding(event);
  return null;
});

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();
  // Guard first. The card is the anon surface; the authed branch
  // renders the personalised Home Page. We pass `useSignIn` straight
  // through to the card so the post-signin roundtrip still works
  // (the card's embedded SignInButton submits to the Auth.js action).
  const guard = requireSession(sessionSig.value, "/home");
  if (guard.kind === "anon") {
    return (
      <SignInRequiredCard
        signIn={signInAction}
        description="Sign in to view your home."
        redirectTo={guard.pathname}
      />
    );
  }
  // Personalised greeting (R-HP-001). The name claim may be null or
  // an empty string; both fall back to a generic "Welcome" heading
  // with no trailing comma (S-HP-002, S-HP-003).
  const name = guard.session?.user?.name ?? "";
  const heading = name.length > 0 ? `Welcome, ${name}` : "Welcome";
  return (
    <main class="mx-auto max-w-3xl px-4 py-16">
      <h1 class="text-3xl font-bold text-slate-900" data-testid="home-heading">
        {heading}
      </h1>
      <p class="mt-3 text-slate-700" data-testid="home-paragraph">
        This is your cachicamas home. New sections and shortcuts will appear
        here as the app grows.
      </p>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Home \u2014 Cachicamas",
  meta: [
    {
      name: "description",
      content: "Your cachicamas home, signed in via GitHub.",
    },
  ],
};
