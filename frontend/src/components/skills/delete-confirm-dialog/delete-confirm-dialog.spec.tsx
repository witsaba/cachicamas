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
 *
 * QRL closure capture pattern: spies live in module-scoped arrays
 * because Qwik's QRL serialization rejects function values captured
 * directly inside `$()` closures.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { describe, it, expect, beforeEach } from "vitest";

import { DeleteConfirmDialog } from "./delete-confirm-dialog";

let confirmCalls: number = 0;
let cancelCalls: number = 0;
let captureEnabled = false;

beforeEach(() => {
  confirmCalls = 0;
  cancelCalls = 0;
  captureEnabled = true;
});

function makeConfirmStub(): QRL<() => void> {
  return $(() => {
    if (captureEnabled) confirmCalls += 1;
  });
}

function makeCancelStub(): QRL<() => void> {
  return $(() => {
    if (captureEnabled) cancelCalls += 1;
  });
}

describe("components/skills/delete-confirm-dialog", () => {
  it("TestDeleteConfirmDialog_OpensWithName — renders the skill name in the dialog heading", async () => {
    const { screen, render } = await createDOM();
    await render(
      <DeleteConfirmDialog
        name="pdf-cleanup"
        onConfirm$={makeConfirmStub()}
        onCancel$={makeCancelStub()}
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
    const { screen, render, userEvent } = await createDOM();
    await render(
      <DeleteConfirmDialog
        name="pdf-cleanup"
        onConfirm$={makeConfirmStub()}
        onCancel$={makeCancelStub()}
      />,
    );
    const okBtn = screen.querySelector(
      '[data-testid="delete-confirm-ok"]',
    ) as HTMLElement | null;
    expect(okBtn).toBeTruthy();
    await userEvent(okBtn!, "click");
    expect(confirmCalls).toBe(1);
    expect(cancelCalls).toBe(0);
  });

  it("TestDeleteConfirmDialog_CancelDoesNotCallOnConfirm — clicking the Cancel button invokes onCancel$ and NOT onConfirm$", async () => {
    const { screen, render, userEvent } = await createDOM();
    await render(
      <DeleteConfirmDialog
        name="pdf-cleanup"
        onConfirm$={makeConfirmStub()}
        onCancel$={makeCancelStub()}
      />,
    );
    const cancelBtn = screen.querySelector(
      '[data-testid="delete-confirm-cancel"]',
    ) as HTMLElement | null;
    expect(cancelBtn).toBeTruthy();
    await userEvent(cancelBtn!, "click");
    expect(cancelCalls).toBe(1);
    expect(confirmCalls).toBe(0);
  });
});