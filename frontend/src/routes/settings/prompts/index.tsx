/**
 * /settings/prompts — Prompt Studio.
 *
 * Reference: openspec/changes/2026-07-16-prompts-frontend/specs/frontend-prompts/spec.md
 *
 * Renders: PromptStudio — split-panel layout with sidebar + editor.
 *
 * States:
 *   loading   → <p>Loading…</p>
 *   error     → error alert with retry
 *   empty     → EmptyState (no prompts)
 *   loaded    → PromptSidebar + PromptEditor
 *
 * Navigation:
 *   - The `← Back` affordance uses `window.history.back()` with a
 *     `/settings` fallback for deep-link entries. NOT a hardcoded
 *     `<Link href="...">` — semantic actions survive navigation
 *     flow changes (deep links, bookmarks, cross-navigation).
 */

import { $, component$, useSignal } from "@builder.io/qwik";
import {
  routeLoader$,
  useNavigate,
  type DocumentHead,
  type RequestHandler,
} from "@builder.io/qwik-city";
import {
  listPrompts,
  getPrompt,
  createPrompt,
  updatePrompt,
  deletePrompt,
  listRevisions,
  restoreRevision,
  type Prompt,
  type PromptRevision,
  type ApiResult,
} from "~/lib/prompts-api";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { Button } from "~/components/ui/button/button";
import { PromptSidebar } from "~/components/prompts/prompt-sidebar/prompt-sidebar";
import { PromptEditor } from "~/components/prompts/prompt-editor/prompt-editor";
import { EmptyState } from "~/components/prompts/empty-state/empty-state";

// ---------------------------------------------------------------------------
// Request guard — cookie capture + auth + ownboarding
// ---------------------------------------------------------------------------

export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

// ---------------------------------------------------------------------------
// Route loader — SSR initial prompt list
// ---------------------------------------------------------------------------

export const usePromptsLoader = routeLoader$(async (event) => {
  await requireOwnboarding(event);
  const result = await listPrompts();
  if (result.ok) return { ok: true as const, prompts: result.value };
  return { ok: false as const, message: result.message };
});

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default component$(() => {
  const promptsLoader = usePromptsLoader();

  // Prompts list (from SSR loader, updated client-side)
  const prompts = useSignal<Prompt[]>(
    promptsLoader.value.ok ? promptsLoader.value.prompts : [],
  );
  const loaderError = useSignal<string | null>(
    promptsLoader.value.ok ? null : promptsLoader.value.message,
  );
  const loading = useSignal(false);

  // Editor state
  const mode = useSignal<"list" | "edit" | "create">("list");
  const selectedSlug = useSignal<string | null>(null);
  const currentPrompt = useSignal<Prompt | null>(null);
  const currentRevisions = useSignal<PromptRevision[]>([]);
  const editorSaving = useSignal(false);
  const editorError = useSignal<string | null>(null);

  // Real-history back: navigate to wherever the user came from.
  // Falls back to /settings (the URL hierarchy parent) for
  // deep-link / new-tab entries that have no history.
  const nav = useNavigate();
  const handleBack = $(() => {
    if (typeof window === "undefined") return; // SSR guard
    if (window.history.length > 1) {
      window.history.back();
    } else {
      nav("/settings");
    }
  });

  // Load prompt detail + revisions
  const loadPromptDetail = $(async (slug: string) => {
    loading.value = true;
    editorError.value = null;

    const [promptResult, revResult] = await Promise.all([
      getPrompt(slug),
      listRevisions(slug),
    ]);

    if (promptResult.ok) {
      currentPrompt.value = promptResult.value;
      currentRevisions.value = revResult.ok ? revResult.value : [];
      mode.value = "edit";
      selectedSlug.value = slug;
    } else {
      editorError.value = promptResult.message;
    }

    loading.value = false;
  });

  // Select a prompt from the sidebar
  const handleSelectPrompt = $(async (slug: string) => {
    await loadPromptDetail(slug);
  });

  // Switch to create mode
  const handleNewPrompt = $(() => {
    currentPrompt.value = null;
    currentRevisions.value = [];
    mode.value = "create";
    selectedSlug.value = null;
    editorError.value = null;
  });

  // Cancel editing
  const handleCancel = $(() => {
    mode.value = "list";
    currentPrompt.value = null;
    currentRevisions.value = [];
    editorError.value = null;
  });

  // Save (create or update)
  const handleSave = $(
    async (input: { slug?: string; description?: string; body: string }) => {
      editorSaving.value = true;
      editorError.value = null;

      let result: ApiResult<Prompt>;

      if (mode.value === "create") {
        result = await createPrompt({
          slug: input.slug ?? "",
          description: input.description,
          body: input.body,
        });
      } else if (selectedSlug.value) {
        result = await updatePrompt(selectedSlug.value, input.body);
      } else {
        editorSaving.value = false;
        return;
      }

      if (result.ok) {
        // Refresh the prompts list
        const listResult = await listPrompts();
        if (listResult.ok) {
          prompts.value = listResult.value;
        }

        if (mode.value === "create") {
          // Switch to edit mode for the new prompt
          selectedSlug.value = result.value.slug;
          currentPrompt.value = result.value;
          mode.value = "edit";
          // Load revisions
          const revResult = await listRevisions(result.value.slug);
          currentRevisions.value = revResult.ok ? revResult.value : [];
        } else {
          // Update current prompt in memory
          currentPrompt.value = result.value;
          const revResult = await listRevisions(result.value.slug);
          currentRevisions.value = revResult.ok ? revResult.value : [];
        }
      } else {
        editorError.value = result.message;
      }

      editorSaving.value = false;
    },
  );

  // Delete
  const handleDelete = $(async () => {
    if (!selectedSlug.value) return;
    editorSaving.value = true;

    const result = await deletePrompt(selectedSlug.value);

    if (result.ok) {
      // Remove from list
      prompts.value = prompts.value.filter(
        (p) => p.slug !== selectedSlug.value,
      );
      mode.value = "list";
      currentPrompt.value = null;
      currentRevisions.value = [];
      selectedSlug.value = null;
    } else {
      editorError.value = result.message;
    }

    editorSaving.value = false;
  });

  // Restore
  const handleRestore = $(async (revisionNumber: number) => {
    if (!selectedSlug.value) return;
    editorSaving.value = true;
    editorError.value = null;

    const result = await restoreRevision(selectedSlug.value, revisionNumber);

    if (result.ok) {
      currentPrompt.value = result.value;
      const revResult = await listRevisions(selectedSlug.value);
      currentRevisions.value = revResult.ok ? revResult.value : [];
    } else {
      editorError.value = result.message;
    }

    editorSaving.value = false;
  });

  // -----------------------------------------------------------------------
  // Render
  // -----------------------------------------------------------------------

  if (!promptsLoader.value.ok && loaderError.value === null) {
    loaderError.value = promptsLoader.value.message;
  }

  if (loading.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <p data-testid="prompt-studio-loading">Loading…</p>
      </main>
    );
  }

  if (loaderError.value && prompts.value.length === 0) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <div
          role="alert"
          class="rounded border border-red-300 bg-red-50 px-4 py-3 text-red-800"
          data-testid="prompt-studio-error"
        >
          {loaderError.value}
          <Button
            type="button"
            variant="secondary"
            class="mt-2"
            onClick$={async () => {
              const result = await listPrompts();
              if (result.ok) prompts.value = result.value;
              else loaderError.value = result.message;
            }}
          >
            Retry
          </Button>
        </div>
      </main>
    );
  }

  const hasPrompts = prompts.value.length > 0;

  return (
    <main class="mx-auto flex max-w-5xl flex-col px-4 py-8">
      {/* Page header */}
      <div class="mb-6 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <button
            type="button"
            onClick$={handleBack}
            class="text-sm text-slate-500 hover:text-slate-700"
            data-testid="prompt-studio-back"
          >
            &larr; Back
          </button>
          <h1 class="text-2xl font-bold text-slate-900">Prompts</h1>
        </div>
        {hasPrompts && mode.value !== "create" && (
          <Button
            type="button"
            variant="primary"
            onClick$={handleNewPrompt}
            testId="prompt-studio-new"
          >
            + New Prompt
          </Button>
        )}
      </div>

      {/* Prompt Studio layout */}
      <div class="flex flex-1 gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
        {hasPrompts ? (
          <>
            {/* Sidebar */}
            <PromptSidebar
              prompts={prompts.value}
              selectedSlug={selectedSlug.value}
              onSelect$={handleSelectPrompt}
              onNewPrompt$={handleNewPrompt}
            />

            {/* Editor panel */}
            <div class="flex flex-1 flex-col overflow-y-auto">
              {mode.value === "list" ? (
                <div class="flex flex-1 flex-col items-center justify-center p-8 text-center">
                  <p class="text-sm text-slate-500">
                    Select a prompt from the sidebar to edit it,
                    <br />
                    or create a new one.
                  </p>
                  <Button
                    type="button"
                    variant="primary"
                    class="mt-4"
                    onClick$={handleNewPrompt}
                    testId="prompt-studio-new-from-empty"
                  >
                    Create your first prompt
                  </Button>
                </div>
              ) : (
                <PromptEditor
                  prompt={currentPrompt.value}
                  revisions={currentRevisions.value}
                  mode={mode.value === "create" ? "create" : "edit"}
                  saving={editorSaving.value}
                  error={editorError.value}
                  onSave$={handleSave}
                  onCancel$={handleCancel}
                  onDelete$={handleDelete}
                  onRestore$={handleRestore}
                />
              )}
            </div>
          </>
        ) : (
          /* Empty state — no prompts at all */
          <div class="flex flex-1 flex-col">
            <EmptyState onCreate$={handleNewPrompt} />
            {mode.value === "create" && (
              <div class="flex flex-1">
                <PromptEditor
                  prompt={null}
                  revisions={[]}
                  mode="create"
                  saving={editorSaving.value}
                  error={editorError.value}
                  onSave$={handleSave}
                  onCancel$={handleCancel}
                  onDelete$={handleDelete}
                  onRestore$={handleRestore}
                />
              </div>
            )}
          </div>
        )}
      </div>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Prompts — Settings — Cachicamas",
};
