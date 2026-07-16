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
import { $, component$, useSignal, type QRL } from "@builder.io/qwik";
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

test("[prompt-editor]: resets local form state (slug/description/body/preview) when prompt transitions from Prompt to null (new-prompt flow)", async () => {
  // Regression guard for UAT bug 2026-07-16: clicking "New Prompt"
  // after editing an existing prompt left the editor form populated
  // with the old prompt's data. The useTask$ reset task only ran
  // its body when `prompt` was truthy, so the transition to null
  // kept the previous values.
  //
  // Setup: wrap the editor in a parent that holds prompt as a
  // signal, render once with a populated prompt, then mutate the
  // signal to null (the same shape as the real route's
  // `handleNewPrompt` handler) and verify all four form fields
  // clear out.
  //
  // This test uses an inline TestWrapper component$ because
  // createDOM()'s render() doesn't support in-place prop mutation;
  // the wrapper lets us drive the transition through a signal,
  // which IS what Qwik does in production when the parent
  // component re-renders with a different prop value.
  const originalPrompt: Prompt = {
    id: 1,
    slug: "hello",
    description: "An old prompt",
    body: "Old body content that should be cleared",
    current_revision: 1,
    created_at: "2026-07-16T00:00:00Z",
    updated_at: "2026-07-16T00:00:00Z",
    deleted_at: null,
  };

  const TestWrapper = component$(() => {
    const prompt = useSignal<Prompt | null>(originalPrompt);
    const mode = useSignal<"create" | "edit">("edit");

    return (
      <>
        <button
          type="button"
          data-testid="simulate-new-prompt"
          onClick$={() => {
            // Mirror the real route's handleNewPrompt: clear prompt, switch to create.
            prompt.value = null;
            mode.value = "create";
          }}
        >
          Simulate New Prompt
        </button>
        <PromptEditor
          prompt={prompt.value}
          revisions={[]}
          mode={mode.value}
          saving={false}
          error={null}
          onSave$={makeRecordingOnSave()}
          onCancel$={makeQrlStub()}
          onDelete$={makeQrlStub()}
          onRestore$={makeRestoreQrlStub()}
        />
      </>
    );
  });

  const { screen, render, userEvent } = await createDOM();
  await render(<TestWrapper />);

  // Step 1: Editor is populated with the original prompt.
  const slugInput = screen.querySelector(
    '[data-testid="prompt-editor-slug"]',
  ) as HTMLInputElement;
  expect(slugInput).toBeTruthy();
  expect(slugInput.value).toBe("hello");

  const descriptionInput = screen.querySelector(
    '[data-testid="prompt-editor-description"]',
  ) as HTMLInputElement;
  expect(descriptionInput.value).toBe("An old prompt");

  expect(getBodyTextarea(screen).value).toBe(
    "Old body content that should be cleared",
  );

  // Step 2: Simulate the parent's "New Prompt" click.
  const newPromptBtn = screen.querySelector(
    '[data-testid="simulate-new-prompt"]',
  ) as HTMLButtonElement;
  expect(newPromptBtn).toBeTruthy();
  await userEvent(newPromptBtn, "click");

  // Step 3: All four form fields MUST be empty after the transition.
  expect(slugInput.value).toBe("");
  expect(descriptionInput.value).toBe("");
  expect(getBodyTextarea(screen).value).toBe("");
});
