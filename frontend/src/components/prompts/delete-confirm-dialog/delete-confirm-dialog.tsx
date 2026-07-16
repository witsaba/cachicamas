/**
 * DeleteConfirmDialog — confirmation dialog for deleting a prompt.
 *
 * Props:
 *   slug       — the prompt slug being deleted
 *   onConfirm$ — called when the user confirms deletion
 *   onCancel$  — called when the user cancels
 */

import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

export interface DeleteConfirmDialogProps {
  slug: string;
  onConfirm$: QRL<() => void>;
  onCancel$: QRL<() => void>;
}

export const DeleteConfirmDialog = component$<DeleteConfirmDialogProps>(
  ({ slug, onConfirm$, onCancel$ }) => {
    return (
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-dialog-title"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
        data-testid="delete-confirm-dialog"
      >
        <div class="w-full max-w-sm rounded-lg bg-white p-6 shadow-lg">
          <h2
            id="delete-dialog-title"
            class="text-lg font-semibold text-slate-900"
          >
            Delete "{slug}"?
          </h2>
          <p class="mt-3 text-sm text-slate-700">
            This action cannot be undone. The prompt will be permanently
            removed.
          </p>
          <div class="mt-6 flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick$={onCancel$}
              testId="delete-confirm-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick$={onConfirm$}
              testId="delete-confirm-ok"
            >
              Delete
            </Button>
          </div>
        </div>
      </div>
    );
  },
);
