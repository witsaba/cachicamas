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
