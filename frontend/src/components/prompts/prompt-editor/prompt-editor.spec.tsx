/**
 * PromptEditor spec — covers Save button reactivity + debounce serialization.
 *
 * What this guards against:
 *   1. Save button reactivity — the Save button must enable when the
 *      user types non-whitespace into the body. In Qwik, plain consts
 *      derived from signals ARE reactive (component$ re-runs on signal
 *      change), so this test passes whether canSave is a `const` or a
 *      `useComputed$`. The migration to useComputed$ is a perf
 *      improvement (surgical updates vs full template re-render — see
 *      Qwik best practices, "Moving signal reads to useTask$ or
 *      useComputed$"). NOT a reactivity bug fix.
 *   2. Debounce serialization — the `debounceTimer` signal previously
 *      held a Timeout object via plain useSignal, which Qwik cannot
 *      serialize (unhandled rejection on every keystroke). The fix
 *      wraps the timer in `noSerialize()` so it survives Qwik's
 *      resumability model.
 *
 * Scenarios:
 *   1. Save button is disabled when body is empty (sanity).
 *   2. Save button becomes enabled after user types non-whitespace
 *      into body (reactive behavior — passes with const or useComputed$).
 *   3. Clicking Save invokes `onSave$` with the typed body (end-to-end).
 *   4. Save button is disabled again while `saving={true}`, even when
 *      the body has been filled (covers the `&& !saving` clause).
 *
 * Why module-level captures instead of vi.fn():
 *   Qwik's `$()` requires captured variables to be serializable.
 *   `vi.fn()` returns a Mock that fails Qwik's verifier ("Captured
 *   variable in the closure can not be serialized"). The project
 *   convention (mirrored from `ownboarding-form.spec.tsx`) is to push
 *   captured calls into module-level arrays and assert on the array
 *   shape directly.
 *
 * Why userEvent("input", { target, value }) instead of plain value:
 *   `MarkdownTextarea`'s `onInput$` reads `(e.target as HTMLTextAreaElement).value`,
 *   not `e.value` directly. Qwik's `userEvent` creates a synthetic Event
 *   whose `target` is `null` by default — `Object.assign(event, payload)`
 *   only copies payload keys, and `Event.target` is a non-writable
 *   internal getter until dispatch. Passing `{ target: textarea }` in
 *   the eventPayload assigns the element directly to `event.target`
 *   via Object.assign, which shadows the internal getter and lets the
 *   handler read the pre-set `textarea.value`.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { beforeEach, test, expect } from "vitest";
import { PromptEditor } from "./prompt-editor";
import type { Prompt, PromptRevision } from "~/lib/prompts-api";

type SaveInput = { slug?: string; description?: string; body: string };

let capturedCalls: SaveInput[] = [];
let captureEnabled = false;

beforeEach(() => {
  capturedCalls = [];
  captureEnabled = true;
});

function makeQrlStub(): QRL<() => void> {
  return $(() => {});
}

function makeRestoreQrlStub(): QRL<(n: number) => void> {
  return $((_n: number) => {});
}

function makeRecordingOnSave(): QRL<(input: SaveInput) => void> {
  return $((input: SaveInput) => {
    if (!captureEnabled) return;
    capturedCalls.push(input);
  });
}

interface RenderResult {
  screen: Awaited<ReturnType<typeof createDOM>>["screen"];
  userEvent: Awaited<ReturnType<typeof createDOM>>["userEvent"];
}

async function renderEditor(args: {
  mode: "create" | "edit";
  prompt: Prompt | null;
  saving: boolean;
}): Promise<RenderResult> {
  const { screen, render, userEvent } = await createDOM();
  await render(
    <PromptEditor
      prompt={args.prompt}
      revisions={[]}
      mode={args.mode}
      saving={args.saving}
      error={null}
      onSave$={makeRecordingOnSave()}
      onCancel$={makeQrlStub()}
      onDelete$={makeQrlStub()}
      onRestore$={makeRestoreQrlStub()}
    />,
  );
  return { screen, userEvent };
}

function getSaveButton(
  screen: Awaited<ReturnType<typeof createDOM>>["screen"],
): HTMLButtonElement {
  const btn = screen.querySelector(
    '[data-testid="prompt-editor-save"]',
  ) as HTMLButtonElement | null;
  if (!btn) throw new Error("expected prompt-editor-save button to be rendered");
  return btn;
}

function getBodyTextarea(
  screen: Awaited<ReturnType<typeof createDOM>>["screen"],
): HTMLTextAreaElement {
  const ta = screen.querySelector(
    '[data-testid="prompt-editor-body"]',
  ) as HTMLTextAreaElement | null;
  if (!ta) throw new Error("expected prompt-editor-body textarea to be rendered");
  return ta;
}

test("[prompt-editor]: Save button is disabled when body is empty (create mode)", async () => {
  const { screen } = await renderEditor({
    mode: "create",
    prompt: null,
    saving: false,
  });
  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});

test("[prompt-editor]: Save button becomes enabled after user types non-whitespace into body (perf boundary: useComputed$ surgical update)", async () => {
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    prompt: null,
    saving: false,
  });
  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);

  const textarea = getBodyTextarea(screen);
  textarea.value = "Hello world";
  await userEvent(textarea, "input", { target: textarea });

  expect(saveBtn.disabled).toBe(false);
});

test("[prompt-editor]: clicking Save invokes onSave$ with the typed body (end-to-end)", async () => {
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    prompt: null,
    saving: false,
  });

  const textarea = getBodyTextarea(screen);
  textarea.value = "First prompt body";
  await userEvent(textarea, "input", { target: textarea });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(false);

  await userEvent(saveBtn, "click");

  expect(capturedCalls.length).toBe(1);
  expect(capturedCalls[0]).toEqual(
    expect.objectContaining({ body: "First prompt body" }),
  );
});

test("[prompt-editor]: Save button is disabled while saving={true}, even with body filled", async () => {
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    prompt: null,
    saving: true,
  });

  const textarea = getBodyTextarea(screen);
  textarea.value = "Some content";
  await userEvent(textarea, "input", { target: textarea });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});
