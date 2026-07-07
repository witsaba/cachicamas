/**
 * /workspaces — list page.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-011 (S-WS-100..104) — list page contract.
 *   R-WS-016 (S-WS-150..152) — header "Workspaces" link is auth-aware.
 *
 * Auth chain (mirrors the /ownboarding pattern):
 *   requireAuthRedirect on anon → /auth/signin?callbackUrl=/workspaces
 *   requireOwnboarding when no org → /ownboarding
 *
 * Render states:
 *   - loading → <p>Loading…</p>
 *   - error → alert with "Retry" button
 *   - empty → "No workspaces yet" + CTA to /workspaces/new
 *   - populated → list of WorkspaceCard
 *
 * SSR cookie forwarding (S-WS-AUTH-CHAIN-SSR-001):
 *   `onRequest` runs first and captures the inbound Cookie header into
 *   the module-level ssrCookie header (sync, no Promise). The auth +
 *   ownboarding guards run AFTER that capture and throw synchronously
 *   to short-circuit anonymous requests. `listWorkspacesSSR` (called
 *   from useTask$) reads the captured header and re-attaches the
 *   cookie to the outgoing SSR fetch.
 */
import { $, component$, useSignal, useTask$ } from "@builder.io/qwik";
import { type DocumentHead, type RequestHandler } from "@builder.io/qwik-city";
import {
  listWorkspaces,
  listWorkspacesSSR,
  type WorkspaceSummary,
} from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { WorkspaceCard } from "~/components/workspace-card/workspace-card";

export const onRequest: RequestHandler = async (event) => {
  // See routes/home/index.tsx for the rationale on async + cookie
  // capture + guards.
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

export default component$(() => {
  const workspaces = useSignal<WorkspaceSummary[]>([]);
  const loading = useSignal(true);
  const error = useSignal<string | null>(null);

  const load = $(async () => {
    loading.value = true;
    error.value = null;
    const result = await listWorkspaces();
    if (result.ok) {
      workspaces.value = result.value.workspaces;
    } else {
      error.value = result.message;
    }
    loading.value = false;
  });

  useTask$(async () => {
    loading.value = true;
    error.value = null;
    const result = await listWorkspacesSSR();
    if (result.ok) {
      workspaces.value = result.value.workspaces;
    } else {
      error.value = result.message;
    }
    loading.value = false;
  });

  if (loading.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <p data-testid="workspaces-loading">Loading…</p>
      </main>
    );
  }

  if (error.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <div
          data-testid="workspaces-error"
          role="alert"
          class="rounded border border-red-300 bg-red-50 px-4 py-3 text-red-800"
        >
          <p>{error.value}</p>
          <button
            type="button"
            data-testid="workspaces-retry"
            class="mt-3 inline-flex rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white"
            onClick$={load}
          >
            Retry
          </button>
        </div>
      </main>
    );
  }

  if (workspaces.value.length === 0) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <section data-testid="workspaces-empty">
          <h1 class="text-3xl font-bold text-slate-900">No workspaces yet</h1>
          <p class="mt-3 text-slate-700">
            Create your first one to connect a GitHub repository.
          </p>
          <a
            href="/workspaces/new"
            data-testid="create-workspace-cta"
            class="mt-6 inline-flex rounded bg-slate-900 px-5 py-3 text-base font-medium text-white hover:bg-slate-700"
          >
            Create workspace
          </a>
        </section>
      </main>
    );
  }

  return (
    <main class="mx-auto max-w-3xl px-4 py-16">
      <section data-testid="workspaces-list">
        <header class="mb-6 flex items-center justify-between">
          <h1 class="text-3xl font-bold text-slate-900">Workspaces</h1>
          <a
            href="/workspaces/new"
            data-testid="create-workspace-cta"
            class="inline-flex rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
          >
            Create workspace
          </a>
        </header>
        <ul class="space-y-3">
          {workspaces.value.map((w) => (
            <li key={w.id}>
              <WorkspaceCard workspace={w} />
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Workspaces — Cachicamas",
  meta: [
    {
      name: "description",
      content:
        "Manage the GitHub repositories your Cachicamas install is connected to.",
    },
  ],
};
