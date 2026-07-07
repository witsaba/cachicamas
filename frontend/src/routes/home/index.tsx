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
 */
import { $, component$, useSignal, useTask$ } from "@builder.io/qwik";
import { routeLoader$, type DocumentHead } from "@builder.io/qwik-city";
import { HomeWorkspacesSection } from "~/components/home-workspaces-section/home-workspaces-section";
import { SignInRequiredCard } from "~/components/sign-in-required-card/sign-in-required-card";
import { listWorkspaces, type WorkspaceSummary } from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { requireSession } from "~/lib/require-session";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export { requireAuthRedirect as onRequest };

export const useSetupLoader = routeLoader$(async (event) => {
  await requireOwnboarding(event);
  return null;
});

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

  // Workspaces section state.
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);
  const workspaces = useSignal<WorkspaceSummary[]>([]);
  const truncated = useSignal(false);

  // Module-level QRL so the optimizer transforms it correctly.
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
    await reloadWorkspaces();
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
