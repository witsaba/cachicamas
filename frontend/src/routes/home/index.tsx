/**
 * `/home` — the canonical signed-in Home Page.
 *
 * Reference: openspec/changes/home-page-placeholder/specs/home-page/spec.md
 *   R-HP-001 (S-HP-001..S-HP-004) — personalised greeting for authed users.
 *   R-HP-003 (S-HP-020..S-HP-023) — anonymous renders SignInRequiredCard.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-015 (S-WS-140..143) — /home workspaces section replaces the
 *     single-paragraph placeholder for the authed branch.
 *
 * Aphantasic-friendly: text-first, no imagery.
 *
 * SSR cookie forwarding (S-WS-AUTH-CHAIN-SSR-001):
 *   `onRequest` runs first and captures the inbound Cookie header into
 *   the module-level ssrCookie header (sync, no Promise). The auth +
 *   ownboarding guards run AFTER that capture and throw synchronously
 *   to short-circuit anonymous requests. `listWorkspacesSSR` (called
 *   from useTask$) reads the captured header and re-attaches the cookie
 *   to the outgoing SSR fetch. Without this forwarding the backend's
 *   IdentityFromCookie middleware (commit fbe62c0) would 401 the
 *   request.
 */
import { $, component$, useSignal, useTask$ } from "@builder.io/qwik";
import { type DocumentHead, type RequestHandler } from "@builder.io/qwik-city";
import { HomeWorkspacesSection } from "~/components/home-workspaces-section/home-workspaces-section";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import {
  listWorkspaces,
  listWorkspacesSSR,
  type WorkspaceSummary,
} from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { requireSession } from "~/lib/require-session";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export const onRequest: RequestHandler = async (event) => {
  // Capture the inbound cookie BEFORE the guards run so SSR-time
  // api fetches in useTask$ can re-attach it.
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  // requireAuthRedirect throws synchronously when anonymous — Qwik
  // catches the sync throw and short-circuits with the redirect.
  requireAuthRedirect(event);
  // requireOwnboarding is ASYNC (it awaits a setup-state fetch to the
  // backend). Awaiting it makes its `event.redirect(...)` rejection
  // propagate through onRequest's returned Promise, which Qwik
  // treats as a redirect rather than a fatal server error.
  await requireOwnboarding(event);
};

// Module-level QRLs so the Qwik optimizer can transform them (inline
// `$(...)` inside JSX is rejected at runtime).
const noOpNavigate = $(() => undefined);
const noOpCreate = $(() => undefined);

export default component$(() => {
  const sessionSig = useSession();
  const signInAction = useSignIn();

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
  const name = guard.session?.user?.name ?? "";
  const heading = name.length > 0 ? `Welcome, ${name}` : "Welcome";

  // Workspaces section state. Initial values come from useTask$ which
  // calls listWorkspacesSSR — that helper reads the cookie from the
  // module-level ssrCookie header (set in onRequest above).
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const workspaces = useSignal<WorkspaceSummary[]>([]);
  const truncated = useSignal(false);

  const reloadWorkspaces = $(async () => {
    loading.value = true;
    error.value = null;
    const result = await listWorkspaces();
    if (result.ok) {
      workspaces.value = result.value.workspaces;
      truncated.value = result.value.truncated;
    } else {
      error.value = result.message;
    }
    loading.value = false;
  });

  useTask$(async () => {
    loading.value = true;
    error.value = null;
    // listWorkspacesSSR reads the cookie from the module-level
    // header (set during this request's onRequest). Browser-side
    // calls bypass this helper — useTask$ only runs on the SSR /
    // hydration side, and at hydration time the cookie has already
    // been used by the server path.
    const result = await listWorkspacesSSR();
    if (result.ok) {
      workspaces.value = result.value.workspaces;
      truncated.value = result.value.truncated;
    } else {
      error.value = result.message;
    }
    loading.value = false;
  });

  return (
    <main class="mx-auto max-w-3xl px-4 py-16">
      <h1 class="text-3xl font-bold text-slate-900" data-testid="home-heading">
        {heading}
      </h1>
      <p class="mt-3 text-slate-700" data-testid="home-paragraph">
        Your workspaces connect GitHub repositories to your organization. Pick
        one to start, or create a new one for a different repo.
      </p>
      <HomeWorkspacesSection
        loading={loading.value}
        error={error.value}
        workspaces={workspaces.value}
        truncated={truncated.value}
        onRetry={reloadWorkspaces}
        onNavigate={noOpNavigate}
        onCreateWorkspace={noOpCreate}
      />
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
