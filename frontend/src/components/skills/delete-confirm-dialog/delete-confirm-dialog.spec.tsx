/**
 * Behavioural spec for the delete-confirm dialog.
 *
 * Reference: `sdd/cachicamas-skills-foundational/{spec,design}` (engram).
 *   - S-FE-5 (Delete flow): dialog confirms deletion before DELETE call
 *   - delete dialog must surface the skill NAME so the user confirms
 *     what they are deleting (no anonymous "this item" affordance)
 *
 * RED step: until `delete-confirm-dialog.tsx` exists, the import
 * fails and the suite fails — that failure IS the RED state.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $ } from "@builder.io/qwik";
import { describe, it, expect, vi } from "vitest";

import { DeleteConfirmDialog } from "./delete-confirm-dialog";

describe("components/skills/delete-confirm-dialog", () => {
  it("TestDeleteConfirmDialog_OpensWithName — renders the skill name in the dialog heading", async () => {
    const { screen, render } = await createDOM();
    await render(
      <DeleteConfirmDialog
        name="pdf-cleanup"
        onConfirm$={$(() => {})}
        onCancel$={$(() => {})}
      />,
    );
    const dialog = screen.querySelector(
      '[data-testid="delete-confirm-dialog"]',
    );
    expect(dialog).toBeTruthy();
    const text = dialog?.textContent ?? "";
    // The skill name MUST be visible in the dialog body so the user
    // knows what they are about to delete.
    expect(text).toContain("pdf-cleanup");
  });

  it("TestDeleteConfirmDialog_CallsOnConfirm — clicking the Delete button invokes onConfirm$ exactly once", async () => {
    const confirmSpy = vi.fn();
    const { screen, render } = await createDOM();
    await render(
      <DeleteConfirmDialog
        name="pdf-cleanup"
        onConfirm$={$(() => {
          confirmSpy();
        })}
        onCancel$={$(() => {})}
      />,
    );
    const okBtn = screen.querySelector(
      '[data-testid="delete-confirm-ok"]',
    ) as HTMLElement | null;
    expect(okBtn).toBeTruthy();
    okBtn?.click();
    // Event loop drain so Qwik's QRL dispatch resolves.
    await new Promise((r) => setTimeout(r, 0));
    expect(confirmSpy).toHaveBeenCalledTimes(1);
  });
});