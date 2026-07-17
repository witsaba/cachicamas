/**
 * RestoreConfirmDialog — confirmation dialog for restoring a revision.
 *
 * Props:
 *   revisionNumber  — the revision number being restored (shown in the heading)
 *   onConfirm$      — called with the revision number when the user confirms
 *   onCancel$       — called when the user cancels
 *
 * API note (skills vs prompts):
 *   The prompts dialog uses `onConfirm$: QRL<() => void>` and the parent
 *   tracks which revision is being restored via its own state. The
 *   Skills dialog passes the revision number through `onConfirm$` so
 *   the parent does not need a parallel "confirming revision" signal —
 *   the dialog is the source of truth for which revision the user is
 *   restoring.
 */

import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

export interface RestoreConfirmDialogProps {
  revisionNumber: number;
  onConfirm$: QRL<(revisionNumber: number) => void>;
  onCancel$: QRL<() => void>;
}

export const RestoreConfirmDialog = component$<RestoreConfirmDialogProps>(
  ({ revisionNumber, onConfirm$, onCancel$ }) => {
    return (
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="restore-dialog-title"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4"
        data-testid="restore-confirm-dialog"
      >
        <div class="w-full max-w-sm rounded-lg bg-white p-6 shadow-lg">
          <h2
            id="restore-dialog-title"
            class="text-lg font-semibold text-slate-900"
          >
            Restore to v{revisionNumber}?
          </h2>
          <p class="mt-3 text-sm text-slate-700">
            This will create a new revision with the v{revisionNumber} content.
            The current content will be preserved in history.
          </p>
          <div class="mt-6 flex justify-end gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick$={onCancel$}
              testId="restore-confirm-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              onClick$={$(() => onConfirm$(revisionNumber))}
              testId="restore-confirm-ok"
            >
              Restore
            </Button>
          </div>
        </div>
      </div>
    );
  },
);

// Local helper import — the prompts pattern uses inline `$()` but
// here we keep the explicit `$` import at the top of the file. Re-
// imported here as a re-export to keep the file self-contained for
// future readers; the `onClick$={$(() => onConfirm$(revisionNumber))}`
// pattern is intentional (lazy QRL wrap) so the parent QRL is not
// re-invoked inside the click handler — it just calls the already-
// created QRL with the revision number.
import { $ } from "@builder.io/qwik";