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
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// SSR-time auth plumbing: this loader runs in the request context and
// gives Qwik City a hook for the SSR pass. Session resolution still
// happens via `useSession()` inside the component$, so this loader is
// a no-op for now (returns null). It is exported so the route tree
// declares the SSR contract explicitly (mirrors routes/profile/index.tsx).
export const useHomeSession = routeLoader$(({ cookie, env }) => {
  void cookie;
  void env;
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
