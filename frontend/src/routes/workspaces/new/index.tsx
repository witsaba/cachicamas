/**
 * /workspaces/new — workspace creation form route (R-WS-012).
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-012 (S-WS-110..113) — workspace create form contract.
 *
 * Mirrors the structure of routes/ownboarding/index.tsx:
 *   - requireAuthRedirect as onRequest (anon → /auth/signin?callbackUrl=/workspaces/new).
 *   - routeLoader$ + requireOwnboarding (no org → /ownboarding).
 *   - Authed branch renders the WorkspaceForm.
 *   - Submit success navigates to /workspaces/:id (R-WS-001 S-WS-001).
 */
import { $, component$ } from "@builder.io/qwik";
import {
  routeLoader$,
  type DocumentHead,
  useNavigate,
} from "@builder.io/qwik-city";
import {
  WorkspaceForm,
  type WorkspaceFormAction,
  type WorkspaceFormActionResult,
} from "~/components/workspace-form/workspace-form";
import { createWorkspace, listGitHubRepos } from "~/lib/api";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { useSession, useSignIn } from "~/routes/plugin@auth";

export { requireAuthRedirect as onRequest };

export const useSetupLoader = routeLoader$(async (event) => {
  await requireOwnboarding(event);
  return null;
});

/**
 * Submit action wrapping createWorkspace. Maps the ApiResult envelope to
 * the WorkspaceFormActionResult discriminated union.
 */
const submitAction: WorkspaceFormAction = $(
  async (data: FormData): Promise<WorkspaceFormActionResult> => {
    const name = String(data.get("name") ?? "");
    const primaryRepo = {
      github_id: Number(data.get("primary_repository_id") ?? "0"),
      full_name: String(data.get("primary_repository_full_name") ?? ""),
      owner: String(data.get("primary_repository_owner") ?? ""),
      name: String(data.get("primary_repository_name") ?? ""),
    };
    if (
      !primaryRepo.full_name ||
      !primaryRepo.owner ||
      !primaryRepo.name ||
      primaryRepo.github_id <= 0
    ) {
      return {
        ok: false,
        field: "primary_repository",
        message: "Pick a primary repository.",
      };
    }
    const result = await createWorkspace({
      name,
      primaryRepository: primaryRepo,
    });
    if (result.ok) {
      return { ok: true, id: result.value.id };
    }
    if (result.kind === "validation") {
      const nameMessage = result.fields.name;
      if (nameMessage) {
        return {
          ok: false,
          field: "name",
          message: nameMessage,
        };
      }
      const primaryRepoMessage = result.fields.primary_repository;
      if (primaryRepoMessage) {
        return {
          ok: false,
          field: "primary_repository",
          message: primaryRepoMessage,
        };
      }
      const firstField = Object.entries(result.fields)[0];
      return {
        ok: false,
        field: "form",
        message: firstField
          ? `${firstField[0]}: ${firstField[1]}`
          : "Invalid form data.",
      };
    }
    if (result.kind === "conflict") {
      return {
        ok: false,
        field: "name",
        message: result.message,
      };
    }
    return {
      ok: false,
      field: "form",
      message: result.message,
    };
  },
);

/**
 * Repo fetcher QRL for the WorkspaceForm. Wraps listGitHubRepos so the
 * component doesn't need to import the api module directly.
 */
const repoFetcher = $(
  async (opts: { page: number; perPage: number; bustCache?: boolean }) => {
    const result = await listGitHubRepos({
      page: opts.page,
      perPage: opts.perPage,
      bustCache: opts.bustCache,
    });
    if (result.ok) {
      return {
        repositories: result.value.repositories,
        has_next: result.value.hasNext,
      };
    }
    return { repositories: [], has_next: false };
  },
);

export default component$(() => {
  const nav = useNavigate();
  const session = useSession();
  const signIn = useSignIn();
  void signIn;
  if (session.value === null) {
    return null;
  }
  return (
    <main class="mx-auto max-w-3xl px-4 py-16">
      <h1 class="text-3xl font-bold text-slate-900">Create a workspace</h1>
      <p class="mt-3 text-slate-700">
        Pick a name and a primary GitHub repository. You can connect more
        repositories to the workspace after it's created.
      </p>
      <div class="mt-8">
        <WorkspaceForm
          action={submitAction}
          repoFetcher={repoFetcher}
          onSuccess$={$(async (id: number) => {
            await nav(`/workspaces/${id}`);
          })}
        />
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Create a workspace \u2014 Cachicamas",
  meta: [
    {
      name: "description",
      content: "Create a new workspace and bind it to a GitHub repository.",
    },
  ],
};
