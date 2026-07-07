/**
 * /workspaces/:id — workspace detail page.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-013 (S-WS-120..124) — detail page contract.
 *   R-WS-014 (S-WS-130..134) — GitHub repo picker used here.
 *
 * Auth chain (mirrors /workspaces):
 *   requireAuthRedirect on anon → /auth/signin?callbackUrl=/workspaces/:id
 *   requireOwnboarding when no org → /ownboarding
 *
 * Render states:
 *   - loading → <p>Loading…</p>
 *   - not found (HTTP 404) → "Workspace not found." with back link
 *   - loaded → name + primary repo + linked repos + actions
 *   - add-repo panel → picker visible inline (toggled via ?add=1)
 *   - confirm dialogs for disconnect + delete
 *
 * Aphantasic-friendly (UX-4): text-first. Buttons use bg-slate-900 per
 * the locked design system rule.
 */
import { $, component$, useSignal, useTask$ } from "@builder.io/qwik";
import {
  Link,
  routeLoader$,
  useLocation,
  useNavigate,
  type DocumentHead,
  type RequestHandler,
} from "@builder.io/qwik-city";
import {
  addRepoToWorkspace,
  deleteWorkspace,
  getWorkspaceSSR,
  removeRepoFromWorkspace,
  type LinkedRepository,
  type WorkspaceDetail,
} from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { GitHubRepoPicker } from "~/components/github-repo-picker/github-repo-picker";
import type { PrimaryRepository } from "~/lib/api";

// SSR cookie forwarding (S-WS-AUTH-CHAIN-SSR-001). The pattern is the
// same as routes/home/index.tsx and routes/workspaces/index.tsx:
//   1. Capture the inbound Cookie header into the module-level
//      ssrCookieContext (sync, no Promise). This MUST run BEFORE the
//      auth/ownboarding guards because those guards can short-circuit
//      via throw for anonymous users; if the throw fires first, the
//      subsequent ssrFetch calls during the same render cycle never
//      see a captured cookie.
//   2. requireAuthRedirect — throws for anonymous → /auth/signin?cb=...
//   3. requireOwnboarding — throws for no-org → /ownboarding
//
// Why this matters: getWorkspace(id) runs from useTask$ during SSR.
// It calls serverAwareFetch() which dispatches to ssrFetch() on the
// Node runtime. ssrFetch() attaches currentRequestCookie to the
// outgoing request to the backend. If we never called
// setSsrCookieHeader on this route, currentRequestCookie is empty,
// the backend's IdentityFromCookie middleware rejects with 401
// "authentication required", and the page renders the error alert
// instead of the workspace detail. Pre-fix behaviour reproduced
// 2026-07-07 (UAT, see commit message).
export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

export const useSetupLoader = routeLoader$(async (event) => {
  await requireOwnboarding(event);
  return null;
});

export default component$(() => {
  const loc = useLocation();
  const nav = useNavigate();
  const id = Number(loc.params.id);

  const workspace = useSignal<WorkspaceDetail | null>(null);
  const loading = useSignal(true);
  const notFound = useSignal(false);
  const error = useSignal<string | null>(null);
  const showAddPanel = useSignal(loc.url.searchParams.get("add") === "1");
  const pendingDisconnect = useSignal<LinkedRepository | null>(null);
  const showDeleteConfirm = useSignal(false);

  /**
   * listGitHubRepos fetcher injected into the picker. Stubbed in tests
   * via the same QRL signature; production wires it to a real fetch.
   */
  const listGitHubRepos = $(
    async (opts: { page: number; perPage: number; bustCache?: boolean }) => {
      const qs = new URLSearchParams();
      qs.set("page", String(opts.page));
      qs.set("per_page", String(opts.perPage));
      if (opts.bustCache) qs.set("bust_cache", "true");
      const res = await fetch(`${apiBaseUrl()}/github/repos?${qs.toString()}`);
      if (res.status === 401) {
        const body = (await res.json().catch(() => ({}))) as {
          error?: string;
        };
        if (body.error === "github_not_connected") {
          showReconnectBanner();
        }
        return { repositories: [], has_next: false };
      }
      if (!res.ok) return { repositories: [], has_next: false };
      const body = (await res.json()) as {
        repositories: Array<{
          id: number;
          full_name: string;
          owner_login: string;
          name: string;
        }>;
        has_next: boolean;
      };
      return {
        repositories: body.repositories.map((r) => ({
          github_id: r.id,
          full_name: r.full_name,
          owner: r.owner_login,
          name: r.name,
        })),
        has_next: body.has_next,
      };
    },
  );

  const showReconnectBanner = $(() => {
    /* banner state is rendered inline via the picker component */
  });

  const onSelectRepoForAdd = $(async (repo: PrimaryRepository | null) => {
    if (!repo) return;
    const result = await addRepoToWorkspace(id, {
      github_id: repo.github_id,
      full_name: repo.full_name,
      owner: repo.owner,
      name: repo.name,
    });
    if (result.ok && workspace.value) {
      workspace.value = {
        ...workspace.value,
        linked_repositories: [
          ...workspace.value.linked_repositories,
          result.value,
        ],
      };
      showAddPanel.value = false;
    } else if (!result.ok) {
      error.value = result.message;
    }
  });

  const onConfirmDisconnect = $(async () => {
    if (!pendingDisconnect.value || !workspace.value) return;
    const target = pendingDisconnect.value;
    const result = await removeRepoFromWorkspace(id, target.id);
    if (result.ok) {
      workspace.value = {
        ...workspace.value,
        linked_repositories: workspace.value.linked_repositories.filter(
          (r) => r.id !== target.id,
        ),
      };
    } else if (!result.ok) {
      error.value = result.message;
    }
    pendingDisconnect.value = null;
  });

  const onConfirmDelete = $(async () => {
    const result = await deleteWorkspace(id);
    if (result.ok) {
      await nav("/workspaces");
    } else if (!result.ok) {
      error.value = result.message;
      showDeleteConfirm.value = false;
    }
  });

  useTask$(async ({ track }) => {
    track(() => id);
    loading.value = true;
    notFound.value = false;
    error.value = null;
    const result = await getWorkspaceSSR(id);
    if (result.ok) {
      workspace.value = result.value;
    } else if (
      !result.ok &&
      result.message.toLowerCase().includes("not found")
    ) {
      notFound.value = true;
    } else {
      error.value = result.message;
    }
    loading.value = false;
  });

  // -- Render ---------------------------------------------------------------

  if (loading.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <p data-testid="workspace-detail-loading">Loading…</p>
      </main>
    );
  }

  if (notFound.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <section data-testid="workspace-not-found">
          <h1 class="text-3xl font-bold text-slate-900">
            Workspace not found.
          </h1>
          <p class="mt-3 text-slate-700">
            The workspace you're looking for doesn't exist or was deleted.
          </p>
          <Link
            href="/workspaces"
            class="mt-6 inline-flex rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white"
          >
            Back to workspaces
          </Link>
        </section>
      </main>
    );
  }

  if (!workspace.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <div
          data-testid="workspace-detail-error"
          role="alert"
          class="rounded border border-red-300 bg-red-50 px-4 py-3 text-red-800"
        >
          {error.value ?? "Couldn't load workspace."}
        </div>
      </main>
    );
  }

  const ws = workspace.value;
  const primaryRepo = ws.primary_repository;

  return (
    <main class="mx-auto max-w-3xl px-4 py-16">
      <section data-testid="workspace-detail">
        <header class="mb-6 space-y-2">
          <h1
            data-testid="workspace-detail-name"
            class="text-3xl font-bold text-slate-900"
          >
            {ws.name}
          </h1>
          <p
            data-testid="workspace-detail-primary-repo"
            class="text-sm text-slate-600"
          >
            Primary repository:{" "}
            <span class="font-mono">{primaryRepo.full_name}</span>
          </p>
        </header>

        <div class="mb-8 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-slate-900">
            Linked repositories
          </h2>
          <div class="flex gap-2">
            <button
              type="button"
              data-testid="workspace-detail-toggle-add"
              class="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
              onClick$={() => {
                showAddPanel.value = !showAddPanel.value;
              }}
            >
              {showAddPanel.value ? "Cancel" : "Add repository"}
            </button>
            <button
              type="button"
              data-testid="workspace-detail-delete"
              class="rounded border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
              onClick$={() => {
                showDeleteConfirm.value = true;
              }}
            >
              Delete workspace
            </button>
          </div>
        </div>

        {showAddPanel.value ? (
          <div data-testid="workspace-detail-add-panel" class="mb-6">
            <GitHubRepoPicker
              fetcher={listGitHubRepos}
              value={null}
              onChange$={onSelectRepoForAdd}
              skipInitialFetch
            />
          </div>
        ) : null}

        {ws.linked_repositories.length === 0 ? (
          <p
            data-testid="workspace-detail-no-linked"
            class="rounded border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-600"
          >
            No linked repositories yet. Click "Add repository" above.
          </p>
        ) : (
          <ul
            data-testid="workspace-detail-linked-list"
            class="divide-y divide-slate-200 rounded border border-slate-200"
          >
            {ws.linked_repositories.map((r) => (
              <li
                key={r.id}
                data-testid="workspace-detail-linked-row"
                data-linked-id={r.id}
                class="flex items-center justify-between gap-3 px-4 py-3"
              >
                <span class="truncate font-mono text-sm">{r.full_name}</span>
                <button
                  type="button"
                  data-testid="workspace-detail-disconnect"
                  class="rounded border border-slate-300 px-3 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50"
                  onClick$={() => {
                    pendingDisconnect.value = r;
                  }}
                >
                  Disconnect
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      {pendingDisconnect.value ? (
        <div
          data-testid="workspace-detail-disconnect-confirm"
          role="dialog"
          aria-modal="true"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
        >
          <div class="w-full max-w-sm rounded-lg bg-white p-6 shadow-lg">
            <h2 class="text-lg font-semibold text-slate-900">
              Disconnect repository?
            </h2>
            <p class="mt-3 text-sm text-slate-700">
              This will remove{" "}
              <span class="font-mono">{pendingDisconnect.value.full_name}</span>{" "}
              from this workspace.
            </p>
            <div class="mt-6 flex justify-end gap-2">
              <button
                type="button"
                data-testid="workspace-detail-disconnect-cancel"
                class="rounded border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700"
                onClick$={() => {
                  pendingDisconnect.value = null;
                }}
              >
                Cancel
              </button>
              <button
                type="button"
                data-testid="workspace-detail-disconnect-confirm-button"
                class="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white"
                onClick$={onConfirmDisconnect}
              >
                Disconnect
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {showDeleteConfirm.value ? (
        <div
          data-testid="workspace-detail-delete-confirm"
          role="dialog"
          aria-modal="true"
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
        >
          <div class="w-full max-w-sm rounded-lg bg-white p-6 shadow-lg">
            <h2 class="text-lg font-semibold text-slate-900">
              Delete workspace?
            </h2>
            <p class="mt-3 text-sm text-slate-700">
              This will soft-delete <strong>{ws.name}</strong>. You can no
              longer access it from this page.
            </p>
            <div class="mt-6 flex justify-end gap-2">
              <button
                type="button"
                data-testid="workspace-detail-delete-cancel"
                class="rounded border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700"
                onClick$={() => {
                  showDeleteConfirm.value = false;
                }}
              >
                Cancel
              </button>
              <button
                type="button"
                data-testid="workspace-detail-delete-confirm-button"
                class="rounded bg-red-700 px-4 py-2 text-sm font-medium text-white hover:bg-red-800"
                onClick$={onConfirmDelete}
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </main>
  );
});

export const head: DocumentHead = {
  title: "Workspace — Cachicamas",
};

// apiBaseUrl is injected inline by the workspace detail page rather than
// imported from ~/lib/api because the picker fetcher is a small QRL
// captured at component scope.
function apiBaseUrl(): string {
  const isNode =
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined";
  if (isNode) {
    const fromServerEnv = process.env.SERVER_API_BASE_URL;
    return (
      fromServerEnv && fromServerEnv.trim().length > 0
        ? fromServerEnv
        : "http://localhost:8080"
    ).replace(/\/+$/, "");
  }
  const fromEnv = (import.meta as { env?: { PUBLIC_API_BASE_URL?: string } })
    .env?.PUBLIC_API_BASE_URL;
  return (
    fromEnv && fromEnv.trim().length > 0 ? fromEnv : "http://localhost:8080"
  ).replace(/\/+$/, "");
}
