/**
 * Behavioural spec for the restore-confirm dialog.
 *
 * Reference: `sdd/cachicamas-skills-foundational/{spec,design}` (engram).
 *   - S-FE-6 (Restore flow): dialog confirms restore before POST call
 *   - dialog must surface the REVISION NUMBER so the user knows
 *     what they are restoring (no anonymous "this version" affordance)
 *
 * RED step: until `restore-confirm-dialog.tsx` exists, the import
 * fails and the suite fails — that failure IS the RED state.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { describe, it, expect, beforeEach } from "vitest";

import { RestoreConfirmDialog } from "./restore-confirm-dialog";

let confirmCalls: number = 0;
let cancelCalls: number = 0;
let restoreCalls: number[] = [];
let captureEnabled = false;

beforeEach(() => {
  confirmCalls = 0;
  cancelCalls = 0;
  restoreCalls = [];
  captureEnabled = true;
});

function makeRestoreStub(): QRL<(n: number) => void> {
  return $((n: number) => {
    if (captureEnabled) restoreCalls.push(n);
  });
}

function makeCancelStub(): QRL<() => void> {
  return $(() => {
    if (captureEnabled) cancelCalls += 1;
  });
}

describe("components/skills/restore-confirm-dialog", () => {
  it("TestRestoreConfirmDialog_OpensWithRevisionNumber — renders the revision number in the dialog heading", async () => {
    const { screen, render } = await createDOM();
    await render(
      <RestoreConfirmDialog
        revisionNumber={3}
        onConfirm$={makeRestoreStub()}
        onCancel$={makeCancelStub()}
      />,
    );
    const dialog = screen.querySelector(
      '[data-testid="restore-confirm-dialog"]',
    );
    expect(dialog).toBeTruthy();
    const text = dialog?.textContent ?? "";
    // The revision number MUST be visible in the dialog body so the
    // user knows what they are about to restore.
    expect(text).toContain("3");
  });

  it("TestRestoreConfirmDialog_CallsOnConfirm — clicking the Restore button invokes onConfirm$ with the revision number", async () => {
    const { screen, render, userEvent } = await createDOM();
    // We use a confirm-counter stub because the RestoreConfirmDialog
    // contract is just "click Restore → invoke onConfirm$". The
    // revision number is rendered (test above); the dialog itself
    // does not call onConfirm$ with a payload — the parent route
    // already knows the revision from its state.
    void confirmCalls;
    await render(
      <RestoreConfirmDialog
        revisionNumber={3}
        onConfirm$={makeRestoreStub()}
        onCancel$={makeCancelStub()}
      />,
    );
    const okBtn = screen.querySelector(
      '[data-testid="restore-confirm-ok"]',
    ) as HTMLElement | null;
    expect(okBtn).toBeTruthy();
    await userEvent(okBtn!, "click");
    expect(restoreCalls).toEqual([3]);
    expect(cancelCalls).toBe(0);
  });
});