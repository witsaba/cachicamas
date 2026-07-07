/**
 * /workspaces/:id — workspace detail page.
 *
 * Reference: openspec/changes/2026-07-08-workspaces-simplify/specs/workspaces/spec.md
 *   R-WS-003 — detail page contract (post-1:1)
 *
 * 2026-07-08-workspaces-simplify changelog:
 *   - Dropped the Linked repositories section + Add repository button +
 *     Disconnect button + Disconnect confirmation dialog + the
 *     pendingDisconnect / showAddPanel state signals. In the 1:1
 *     model the workspace IS its repo — there is nothing else to
 *     connect.
 *   - Renamed the "Primary repository:" header to "Repository:"
 *     (no contrast with a secondary exists).
 *   - Renamed the `primaryRepo` local to `repo` and switched all
 *     field reads from `primary_repository` to `repository`
 *     (matches the new WorkspaceDetail wire shape from lib/api.ts).
 *
 * Render states:
 *   - loading → <p>Loading…</p>
 *   - not found (HTTP 404) → "Workspace not found." with back link
 *   - loaded → name + repository + delete button
 *   - confirm dialog for delete
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
  deleteWorkspace,
  getWorkspaceSSR,
  type WorkspaceDetail,
} from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";

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
// Why this matters: getWorkspaceSSR(id) runs from useTask$ during
// SSR. It calls serverAwareFetch() which dispatches to ssrFetch()
// on the Node runtime. ssrFetch() attaches currentRequestCookie to
// the outgoing request to the backend. If we never called
// setSsrCookieHeader on this route, currentRequestCookie is empty,
// the backend's IdentityFromCookie middleware rejects with 401
// "authentication required", and the page renders the error alert
// instead of the workspace detail.
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
  const showDeleteConfirm = useSignal(false);

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
  const repo = ws.repository;

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
          <p data-testid="workspace-detail-repo" class="text-sm text-slate-600">
            Repository: <span class="font-mono">{repo.full_name}</span>
          </p>
        </header>

        <div class="mb-8 flex items-center justify-end">
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
      </section>

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
