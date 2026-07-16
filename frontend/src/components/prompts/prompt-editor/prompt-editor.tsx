/**
 * PromptEditor — the main editing panel for prompts.
 *
 * Modes:
 *   edit    — editing an existing prompt
 *   create  — creating a new prompt
 *
 * Features:
 *   - Split pane: MarkdownTextarea + MarkdownPreview (debounced)
 *   - Description field (edit mode only)
 *   - Save/Cancel buttons
 *   - Delete button (edit mode only)
 *   - History panel (collapsed by default)
 *   - Activity feed
 */

import { component$, useSignal, useTask$, $, type QRL } from "@builder.io/qwik";
import type { Prompt, PromptRevision } from "~/lib/prompts-api";
import { MarkdownTextarea } from "~/components/prompts/markdown-textarea/markdown-textarea";
import { MarkdownPreview } from "~/components/prompts/markdown-preview/markdown-preview";
import { DiffViewer } from "~/components/prompts/diff-viewer/diff-viewer";
import { ActivityFeed } from "~/components/prompts/activity-feed/activity-feed";
import { DeleteConfirmDialog } from "~/components/prompts/delete-confirm-dialog/delete-confirm-dialog";
import { RestoreConfirmDialog } from "~/components/prompts/restore-confirm-dialog/restore-confirm-dialog";
import { Button } from "~/components/ui/button/button";

export interface PromptEditorProps {
  prompt: Prompt | null; // null in create mode
  revisions: PromptRevision[];
  mode: "edit" | "create";
  saving: boolean;
  error: string | null;
  onSave$: QRL<
    (input: { slug?: string; description?: string; body: string }) => void
  >;
  onCancel$: QRL<() => void>;
  onDelete$: QRL<() => void>;
  onRestore$: QRL<(revisionNumber: number) => void>;
}

const DEBOUNCE_MS = 300;

export const PromptEditor = component$<PromptEditorProps>(
  ({
    prompt,
    revisions,
    mode,
    saving,
    error,
    onSave$,
    onCancel$,
    onDelete$,
    onRestore$,
  }) => {
    // Local editing state
    const slug = useSignal(prompt?.slug ?? "");
    const description = useSignal(prompt?.description ?? "");
    const body = useSignal(prompt?.body ?? "");
    const previewBody = useSignal(prompt?.body ?? "");

    // Debounce timer for preview update
    const debounceTimer = useSignal<ReturnType<typeof setTimeout> | null>(null);

    // Dialog state
    const showDeleteConfirm = useSignal(false);
    const showRestoreConfirm = useSignal<number | null>(null);

    // Reset local state when prompt changes
    useTask$(({ track }) => {
      track(() => prompt?.slug);
      if (prompt) {
        slug.value = prompt.slug;
        description.value = prompt.description ?? "";
        body.value = prompt.body;
        previewBody.value = prompt.body;
      }
    });

    // Sync body to preview with debounce
    const handleBodyChange = $((newBody: string) => {
      body.value = newBody;

      if (debounceTimer.value) {
        clearTimeout(debounceTimer.value);
      }
      debounceTimer.value = setTimeout(() => {
        previewBody.value = newBody;
      }, DEBOUNCE_MS);
    });

    const hasChanges =
      mode === "create"
        ? body.value.trim().length > 0
        : body.value !== (prompt?.body ?? "");

    const canSave = body.value.trim().length > 0 && !saving;

    const handleSave = $(() => {
      if (!canSave) return;
      onSave$({
        slug: mode === "create" ? slug.value : undefined,
        description: mode === "create" ? description.value : undefined,
        body: body.value,
      });
    });

    // Restore confirm
    const handleRestoreConfirm = $(() => {
      if (showRestoreConfirm.value !== null) {
        onRestore$(showRestoreConfirm.value);
        showRestoreConfirm.value = null;
      }
    });

    return (
      <div class="flex flex-1 flex-col">
        {/* Header: slug + description */}
        <div class="space-y-2 border-b border-slate-200 px-4 py-3">
          <div class="flex items-center gap-2">
            <label class="text-xs font-medium text-slate-500" for="prompt-slug">
              Slug
            </label>
            <input
              id="prompt-slug"
              type="text"
              value={slug.value}
              onInput$={(e) => {
                slug.value = (e.target as HTMLInputElement).value;
              }}
              disabled={mode === "edit"}
              placeholder="my-prompt"
              class={`flex-1 rounded border border-slate-200 px-2 py-1 font-mono text-sm text-slate-900 ${mode === "edit" ? "bg-slate-50" : ""} focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none`}
              data-testid="prompt-editor-slug"
            />
          </div>
          <div class="flex items-center gap-2">
            <label
              class="text-xs font-medium text-slate-500"
              for="prompt-description"
            >
              Description
            </label>
            <input
              id="prompt-description"
              type="text"
              value={description.value}
              onInput$={(e) => {
                description.value = (e.target as HTMLInputElement).value;
              }}
              placeholder="Short description..."
              class="flex-1 rounded border border-slate-200 px-2 py-1 text-sm text-slate-900 placeholder-slate-400 focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none"
              data-testid="prompt-editor-description"
            />
          </div>
        </div>

        {/* Split editor: textarea + preview */}
        <div class="flex flex-1 gap-4 p-4">
          <div class="flex-1">
            <MarkdownTextarea
              value={body.value}
              onInput$={handleBodyChange}
              placeholder="Write your prompt body in markdown..."
              testId="prompt-editor-body"
            />
          </div>
          <div class="flex-1">
            <MarkdownPreview
              body={previewBody.value}
              testId="prompt-editor-preview"
            />
          </div>
        </div>

        {/* Error alert */}
        {error && (
          <div
            role="alert"
            class="mx-4 mt-2 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800"
            data-testid="prompt-editor-error"
          >
            {error}
          </div>
        )}

        {/* History panel (edit mode only) */}
        {mode === "edit" && revisions.length > 0 && prompt && (
          <div class="px-4">
            <DiffViewer
              revisions={revisions}
              currentRevision={prompt.current_revision}
              promptSlug={prompt.slug}
              onRestore$={(_, rev) => {
                showRestoreConfirm.value = rev;
              }}
            />
          </div>
        )}

        {/* Activity feed (edit mode only) */}
        {mode === "edit" && prompt && revisions.length > 0 && (
          <div class="px-4 pb-2">
            <ActivityFeed prompt={prompt} revisions={revisions} />
          </div>
        )}

        {/* Footer: actions */}
        <div class="flex items-center justify-between border-t border-slate-200 px-4 py-3">
          <div>
            {mode === "edit" && (
              <Button
                type="button"
                variant="destructive"
                class="border border-red-300 !bg-transparent !text-red-700 hover:!bg-red-50"
                onClick$={() => {
                  showDeleteConfirm.value = true;
                }}
                testId="prompt-editor-delete"
              >
                Delete prompt
              </Button>
            )}
          </div>
          <div class="flex items-center gap-2">
            {!hasChanges && (
              <span class="text-xs text-slate-400">No changes to save</span>
            )}
            <Button
              type="button"
              variant="secondary"
              onClick$={onCancel$}
              testId="prompt-editor-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              disabled={!canSave}
              loading={saving}
              onClick$={handleSave}
              testId="prompt-editor-save"
            >
              {saving ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>

        {/* Delete confirmation */}
        {showDeleteConfirm.value && (
          <DeleteConfirmDialog
            slug={prompt?.slug ?? ""}
            onConfirm$={() => {
              showDeleteConfirm.value = false;
              onDelete$();
            }}
            onCancel$={() => {
              showDeleteConfirm.value = false;
            }}
          />
        )}

        {/* Restore confirmation */}
        {showRestoreConfirm.value !== null && (
          <RestoreConfirmDialog
            revisionNumber={showRestoreConfirm.value}
            onConfirm$={handleRestoreConfirm}
            onCancel$={() => {
              showRestoreConfirm.value = null;
            }}
          />
        )}
      </div>
    );
  },
);
