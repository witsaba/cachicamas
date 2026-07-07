/**
 * WorkspaceForm — the create-workspace form (R-WS-012 / S-WS-110..113).
 *
 * Mirrors the OwnboardingForm structure (zod-style client validation +
 * submit-action discriminated union result), but with two fields:
 *   - `name` (3..60 chars)
 *   - `repository` (selected via the GitHubRepoPicker component)
 *
 * The submit action is injected by the route (so the route controls the
 * navigation side-effect). The repo picker fetcher is also injected (so
 * the form is testable in isolation).
 */
import { $, component$, useSignal, type QRL } from "@builder.io/qwik";
import {
  GitHubRepoPicker,
  type GitHubRepoFetcher,
} from "~/components/github-repo-picker/github-repo-picker";
import type { Repository } from "~/lib/api";

/**
 * Submit-action discriminated-union result. Mirrors the OwnboardingForm
 * shape for consistency.
 */
export type WorkspaceFormActionResult =
  | { ok: true; id: number }
  | {
      ok: false;
      field: "name" | "repository" | "form";
      message: string;
    };

export type WorkspaceFormAction = QRL<
  (data: FormData) => Promise<WorkspaceFormActionResult>
>;

export type WorkspaceFormOnSuccess = QRL<(id: number) => void>;

export interface WorkspaceFormProps {
  action: WorkspaceFormAction;
  repoFetcher: GitHubRepoFetcher;
  onSuccess$?: WorkspaceFormOnSuccess;
}

const MAX_NAME = 60;

export const WorkspaceForm = component$<WorkspaceFormProps>(
  ({ action, repoFetcher, onSuccess$ }) => {
    const name = useSignal("");
    const nameError = useSignal<string | null>(null);
    const topError = useSignal<string | null>(null);
    const selectedRepo = useSignal<Repository | null>(null);
    const repoError = useSignal<string | null>(null);
    const submitting = useSignal(false);
    const pickerKey = useSignal(0);

    const onNameInput = $((_event: InputEvent, el: HTMLInputElement) => {
      name.value = el.value;
      nameError.value = null;
      topError.value = null;
    });

    const onRepoChange$ = $((repo: Repository | null) => {
      selectedRepo.value = repo;
      repoError.value = null;
      topError.value = null;
    });

    const onClearRepo$ = $(() => {
      selectedRepo.value = null;
      repoError.value = null;
      // Re-mount picker to clear internal state
      pickerKey.value += 1;
    });

    const onSubmit = $(async () => {
      // Client-side validation (server is the source of truth).
      const trimmedName = name.value.trim();
      if (trimmedName.length === 0) {
        nameError.value = "Name is required.";
        return;
      }
      if (trimmedName.length < 3 || trimmedName.length > MAX_NAME) {
        nameError.value = "Name must be 3–60 characters.";
        return;
      }
      if (!selectedRepo.value) {
        repoError.value = "Pick a the GitHub repository.";
        return;
      }
      submitting.value = true;
      topError.value = null;
      const fd = new FormData();
      fd.append("name", trimmedName);
      fd.append("repository_id", String(selectedRepo.value.github_id));
      fd.append("repository_full_name", selectedRepo.value.full_name);
      fd.append("repository_owner", selectedRepo.value.owner);
      fd.append("repository_name", selectedRepo.value.name);
      const result = await action(fd);
      submitting.value = false;
      if (result.ok) {
        if (onSuccess$) await onSuccess$(result.id);
        return;
      }
      if (result.field === "name") {
        nameError.value = result.message;
      } else if (result.field === "repository") {
        repoError.value = result.message;
      } else {
        topError.value = result.message;
      }
    });

    return (
      <form
        preventdefault:submit
        onSubmit$={onSubmit}
        data-testid="workspace-form"
        class="mx-auto max-w-xl space-y-4"
      >
        {topError.value ? (
          <div
            role="alert"
            data-testid="workspace-form-top-error"
            class="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800"
          >
            {topError.value}
          </div>
        ) : null}

        <div>
          <label for="ws-name" class="block text-sm font-medium text-slate-700">
            Workspace name
          </label>
          <input
            id="ws-name"
            name="name"
            type="text"
            required
            maxLength={MAX_NAME}
            value={name.value}
            onInput$={onNameInput}
            data-testid="workspace-form-name"
            class="mt-1 block w-full rounded border border-slate-300 px-3 py-2 text-slate-900"
            aria-describedby={nameError.value ? "ws-name-error" : undefined}
          />
          {nameError.value ? (
            <p
              id="ws-name-error"
              data-testid="workspace-form-name-error"
              class="mt-1 text-sm text-red-700"
            >
              {nameError.value}
            </p>
          ) : null}
        </div>

        <div>
          <span class="block text-sm font-medium text-slate-700">
            Primary repository
          </span>
          <div class="mt-1" data-testid="workspace-form-repo-picker">
            <GitHubRepoPicker
              key={pickerKey.value}
              fetcher={repoFetcher}
              value={selectedRepo.value}
              onChange$={onRepoChange$}
              skipInitialFetch
            />
          </div>
          {repoError.value ? (
            <p
              data-testid="workspace-form-repo-error"
              class="mt-1 text-sm text-red-700"
            >
              {repoError.value}
            </p>
          ) : null}
          {selectedRepo.value ? (
            <button
              type="button"
              data-testid="workspace-form-clear-repo"
              onClick$={onClearRepo$}
              class="mt-2 text-sm text-slate-600 underline"
            >
              Clear selection
            </button>
          ) : null}
        </div>

        <button
          type="submit"
          disabled={submitting.value}
          data-testid="workspace-form-submit"
          class="rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
        >
          {submitting.value ? "Creating..." : "Create workspace"}
        </button>
      </form>
    );
  },
);
