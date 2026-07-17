/**
 * DeleteConfirmDialog — confirmation dialog for deleting a skill.
 *
 * Props:
 *   name       — the skill name being deleted (shown in the heading)
 *   onConfirm$ — called when the user confirms deletion
 *   onCancel$  — called when the user cancels
 *
 * Anti-drift gate (obs #1959):
 *   The dialog MUST surface the skill name so the user confirms
 *   what they are about to delete. A bare "this item" affordance
 *   hides context from the user.
 */

import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

export interface DeleteConfirmDialogProps {
  name: string;
  onConfirm$: QRL<() => void>;
  onCancel$: QRL<() => void>;
}

export const DeleteConfirmDialog = component$<DeleteConfirmDialogProps>(
  ({ name, onConfirm$, onCancel$ }) => {
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
            Delete &ldquo;{name}&rdquo;?
          </h2>
          <p class="mt-3 text-sm text-slate-700">
            This action cannot be undone. The skill will be permanently
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