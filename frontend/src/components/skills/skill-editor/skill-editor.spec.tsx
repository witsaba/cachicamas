/**
 * SkillEditor spec — covers form scaffold, canSave logic, save emission,
 * and reset-on-currentSkill-change.
 *
 * Anti-drift gates (locked in obs #1959):
 *   1. canSave is FALSE initially (no description, no body, no changes).
 *   2. canSave is FALSE after loading a Skill (no changes yet).
 *   3. canSave becomes TRUE only when (a) hasChanges, (b) description
 *      non-empty, (c) body non-empty.
 *   4. onSave$ payload MUST include BOTH description AND body (no
 *      silent discard like prompts — obs #1959 item 4).
 *   5. Reset signals when currentSkill transitions (including null).
 *
 * Layers: integration test using Qwik's createDOM (SSR-shaped, no
 * browser). Same pattern as prompt-editor.spec.tsx.
 */
import { $, component$, useSignal, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { beforeEach, test, expect } from "vitest";
import { SkillEditor } from "./skill-editor";
import type { Skill, SkillRevision } from "~/lib/skills-api";

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

type SaveInput = { description: string; body: string };

function makeSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 1,
    name: "pdf-cleanup",
    description: "Cleans up PDF metadata",
    body: "---\nname: pdf-cleanup\ndescription: Cleans up PDF metadata\n---\nDo the cleanup.",
    current_revision: 1,
    created_at: "2026-07-17T00:00:00Z",
    updated_at: "2026-07-17T00:00:00Z",
    deleted_at: null,
    ...overrides,
  };
}

function makeQrlStub(): QRL<() => void> {
  return $(() => {});
}

function makeRestoreQrlStub(): QRL<(n: number) => void> {
  return $((_n: number) => {});
}

let capturedCalls: SaveInput[] = [];
let captureEnabled = false;
beforeEach(() => {
  capturedCalls = [];
  captureEnabled = true;
});

function makeRecordingOnSave(): QRL<(input: SaveInput) => void> {
  return $((input: SaveInput) => {
    if (captureEnabled) capturedCalls.push(input);
  });
}

interface RenderResult {
  screen: Awaited<ReturnType<typeof createDOM>>["screen"];
  userEvent: Awaited<ReturnType<typeof createDOM>>["userEvent"];
}

async function renderEditor(args: {
  mode: "edit" | "create";
  skill: Skill | null;
  revisions?: SkillRevision[];
  saving?: boolean;
}): Promise<RenderResult> {
  const { screen, render, userEvent } = await createDOM();
  await render(
    <SkillEditor
      skill={args.skill}
      revisions={args.revisions ?? []}
      mode={args.mode}
      saving={args.saving ?? false}
      error={null}
      onSave$={makeRecordingOnSave()}
      onCancel$={makeQrlStub()}
      onDelete$={makeQrlStub()}
      onRestore$={makeRestoreQrlStub()}
    />,
  );
  return { screen, userEvent };
}

// ---------------------------------------------------------------------------
// Task 6.6 — form scaffold
// ---------------------------------------------------------------------------

test("[skill-editor]: renders a description input with the current value", async () => {
  const { screen } = await renderEditor({
    mode: "edit",
    skill: makeSkill({ description: "An existing description" }),
  });

  const input = screen.querySelector(
    '[data-testid="skill-editor-description"]',
  ) as HTMLInputElement;
  expect(input).toBeTruthy();
  expect(input.value).toBe("An existing description");
});

test("[skill-editor]: renders a body textarea with the current value", async () => {
  const { screen } = await renderEditor({
    mode: "edit",
    skill: makeSkill({ body: "Original body content here" }),
  });

  const ta = screen.querySelector(
    '[data-testid="skill-editor-body"]',
  ) as HTMLTextAreaElement;
  expect(ta).toBeTruthy();
  expect(ta.value).toBe("Original body content here");
});

test("[skill-editor]: description + body inputs are both empty in create mode with no skill", async () => {
  // Spec S-FE-3 (create flow): empty form when starting a new skill.
  const { screen } = await renderEditor({ mode: "create", skill: null });

  const input = screen.querySelector(
    '[data-testid="skill-editor-description"]',
  ) as HTMLInputElement;
  const ta = screen.querySelector(
    '[data-testid="skill-editor-body"]',
  ) as HTMLTextAreaElement;
  expect(input.value).toBe("");
  expect(ta.value).toBe("");
});

// ---------------------------------------------------------------------------
// Task 6.7 — canSave logic (anti-drift gate from obs #1959)
// ---------------------------------------------------------------------------

function getSaveButton(
  screen: Awaited<ReturnType<typeof createDOM>>["screen"],
): HTMLButtonElement {
  const btn = screen.querySelector(
    '[data-testid="skill-editor-save"]',
  ) as HTMLButtonElement | null;
  if (!btn) throw new Error("expected skill-editor-save button to be rendered");
  return btn;
}

function getDescriptionInput(
  screen: Awaited<ReturnType<typeof createDOM>>["screen"],
): HTMLInputElement {
  return screen.querySelector(
    '[data-testid="skill-editor-description"]',
  ) as HTMLInputElement;
}

function getBodyTextarea(
  screen: Awaited<ReturnType<typeof createDOM>>["screen"],
): HTMLTextAreaElement {
  return screen.querySelector(
    '[data-testid="skill-editor-body"]',
  ) as HTMLTextAreaElement;
}

test("[skill-editor]: Save button is DISABLED in create mode with empty fields (no description, no body)", async () => {
  // Anti-drift gate: canSave MUST be false initially. Creating an
  // empty skill would 400 on the backend, so the editor must not
  // let the user submit an empty form.
  const { screen } = await renderEditor({ mode: "create", skill: null });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});

test("[skill-editor]: Save button is DISABLED in edit mode right after picking up a skill (no changes yet)", async () => {
  // Anti-drift gate: canSave must consult hasChanges in edit mode.
  // Without this, clicking Save after loading a skill would fire a
  // no-op PATCH that creates an empty revision — exactly the prompts
  // silent-update regression.
  const { screen } = await renderEditor({
    mode: "edit",
    skill: makeSkill({
      name: "existing",
      description: "Existing description",
      body: "Existing body content",
    }),
  });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});

test("[skill-editor]: Save button becomes ENABLED after the user edits both description AND body (non-empty)", async () => {
  // Anti-drift gate: canSave becomes true only when ALL THREE
  // conditions hold: (a) hasChanges, (b) description non-empty,
  // (c) body non-empty.
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    skill: null,
  });

  const desc = getDescriptionInput(screen);
  desc.value = "A meaningful description";
  await userEvent(desc, "input", { target: desc });

  const body = getBodyTextarea(screen);
  body.value = "---\nname: x\ndescription: y\n---\nBody content";
  await userEvent(body, "input", { target: body });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(false);
});

test("[skill-editor]: Save button stays DISABLED if description is empty even after body edit", async () => {
  // Triangulation: missing one of the three required conditions
  // must keep the button disabled. Without this, the test from the
  // previous case would pass trivially because the body change alone
  // flipped hasChanges.
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    skill: null,
  });

  const body = getBodyTextarea(screen);
  body.value = "Some body content";
  await userEvent(body, "input", { target: body });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});

test("[skill-editor]: Save button stays DISABLED if body is whitespace-only even with description filled", async () => {
  // Triangulation: whitespace-only body must NOT enable save.
  // Backend ValidateBody rejects empty / whitespace bodies.
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    skill: null,
  });

  const desc = getDescriptionInput(screen);
  desc.value = "A real description";
  await userEvent(desc, "input", { target: desc });

  const body = getBodyTextarea(screen);
  body.value = "   \n  \t  ";
  await userEvent(body, "input", { target: body });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});

test("[skill-editor]: Save button is DISABLED while saving={true}, even with all fields filled", async () => {
  // The saving-state branch: while a save is in flight, the button
  // must stay disabled to prevent double-submits.
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    skill: null,
    saving: true,
  });

  const desc = getDescriptionInput(screen);
  desc.value = "Description";
  await userEvent(desc, "input", { target: desc });

  const body = getBodyTextarea(screen);
  body.value = "Body";
  await userEvent(body, "input", { target: body });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);
});

// ---------------------------------------------------------------------------
// Task 6.8 — onSave$ payload (anti-drift gate: BOTH description AND body)
// ---------------------------------------------------------------------------

test("[skill-editor]: clicking Save invokes onSave$ with BOTH description AND body (no silent discard)", async () => {
  // Anti-drift gate (obs #1959 item 4):
  //   onSave$ payload MUST include both description AND body, even
  //   in EDIT mode where the user only changed the body. The prompts
  //   feature silently discarded description on save; this test pins
  //   the Skills behavior to "always send both".
  //
  // Setup: load an existing skill, then change ONLY the body.
  // Expected: onSave$ receives { description: ORIGINAL, body: NEW }.
  const { screen, userEvent } = await renderEditor({
    mode: "edit",
    skill: makeSkill({
      name: "edit-me",
      description: "Original description",
      body: "Original body",
    }),
  });

  // Edit the body only.
  const body = getBodyTextarea(screen);
  body.value = "Updated body content";
  await userEvent(body, "input", { target: body });

  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(false);

  await userEvent(saveBtn, "click");

  // Capture must have one call with BOTH fields populated.
  expect(capturedCalls.length).toBe(1);
  expect(capturedCalls[0].description).toBe("Original description");
  expect(capturedCalls[0].body).toBe("Updated body content");
});

test("[skill-editor]: clicking Save in create mode emits the typed description and body verbatim", async () => {
  // Counterpart for create mode: the new skill's description and body
  // both flow into onSave$. This catches the symmetric bug where
  // create-mode wires only one of the two fields.
  const { screen, userEvent } = await renderEditor({
    mode: "create",
    skill: null,
  });

  const desc = getDescriptionInput(screen);
  desc.value = "Brand new description";
  await userEvent(desc, "input", { target: desc });

  const body = getBodyTextarea(screen);
  body.value = "---\nname: brand-new\ndescription: Brand new description\n---\nBrand new body";
  await userEvent(body, "input", { target: body });

  await userEvent(getSaveButton(screen), "click");

  expect(capturedCalls.length).toBe(1);
  expect(capturedCalls[0].description).toBe("Brand new description");
  expect(capturedCalls[0].body).toBe(
    "---\nname: brand-new\ndescription: Brand new description\n---\nBrand new body",
  );
});

test("[skill-editor]: clicking Save when canSave is FALSE does NOT invoke onSave$", async () => {
  // Regression: clicking the (disabled) Save button must not fire
  // onSave$. The button is HTML-disabled, so the browser blocks the
  // event, but we lock the contract here in case the disable logic
  // regresses and the button is incorrectly enabled.
  const { screen, userEvent } = await renderEditor({
    mode: "edit",
    skill: makeSkill(),
  });

  // Save is disabled right after load (no changes). Forcing a click
  // should be a no-op.
  const saveBtn = getSaveButton(screen);
  expect(saveBtn.disabled).toBe(true);

  await userEvent(saveBtn, "click");

  expect(capturedCalls.length).toBe(0);
});
