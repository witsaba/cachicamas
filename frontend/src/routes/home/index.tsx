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
 *   AsyncLocalStorage. `listWorkspacesSSR` (called from useTask$)
 *   reads from that store and re-attaches the cookie to the outgoing
 *   SSR fetch to the backend. Without this forwarding the backend's
 *   IdentityFromCookie middleware (commit fbe62c0) would 401 the
 *   request and the page would render an error block.
 */
import { $, component$, useSignal, useTask$ } from "@builder.io/qwik";
import { type DocumentHead } from "@builder.io/qwik-city";
import type { RequestHandler } from "@builder.io/qwik-city";
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
import { withSsrCookieContext } from "~/lib/with-ssr-cookie";
import { useSession, useSignIn } from "~/routes/plugin@auth";

// Capture the inbound cookie first (for SSR-time api fetches), then
// run the auth + ownboarding guards.
export const onRequest: RequestHandler = (event) => {
  return withSsrCookieContext(event, () => {
    requireAuthRedirect(event);
    requireOwnboarding(event);
  });
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

  // Workspaces section state. Initial values come from the SSR-time
  // fetch (listWorkspacesSSR reads the cookie from AsyncLocalStorage
  // during SSR; from the browser it falls through to the `/api` proxy
  // and gets the cookie auto-attached).
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
    // listWorkspacesSSR reads the cookie from AsyncLocalStorage in SSR
    // (the withSsrCookieContext middleware captured it from the inbound
    // request). Browser-side calls bypass this helper.
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
