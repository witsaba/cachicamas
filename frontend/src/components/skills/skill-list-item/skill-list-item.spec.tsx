/**
 * SkillListItem spec — covers name + version rendering.
 *
 * Anti-drift gate (obs #1959 item 2):
 *   The component MUST render `v{N}` using `skill.current_revision`,
 *   never `v{undefined}`. Backend emits the field (ADR-SK-008) so
 *   the type says `number`, but the component MUST defend against
 *   runtime `undefined` (e.g. legacy fixtures, optimistic updates)
 *   to avoid the prompts `vundefined` regression.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { beforeEach, test, expect } from "vitest";
import { SkillListItem } from "./skill-list-item";
import type { Skill } from "~/lib/skills-api";

// ---------------------------------------------------------------------------
// Test fixtures
// ---------------------------------------------------------------------------

function makeSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 1,
    name: "pdf-cleanup",
    description: "Cleans up PDF metadata",
    body: "---\nname: pdf-cleanup\ndescription: x\n---\n",
    current_revision: 3,
    created_at: "2026-07-17T00:00:00Z",
    updated_at: "2026-07-17T00:00:00Z",
    deleted_at: null,
    ...overrides,
  };
}

function makeOnClickStub(): QRL<() => void> {
  return $(() => {});
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

beforeEach(() => {
  // No module-level capture needed — these tests don't bind to a handler.
});

test("[skill-list-item]: renders the skill name", async () => {
  const { screen, render } = await createDOM();
  await render(
    <SkillListItem
      skill={makeSkill({ name: "pdf-cleanup" })}
      selected={false}
      onClick$={makeOnClickStub()}
    />,
  );

  const el = screen.querySelector('[data-testid="skill-list-item-name"]');
  expect(el).toBeTruthy();
  expect(el?.textContent).toBe("pdf-cleanup");
});

test("[skill-list-item]: renders the current revision as v{N}", async () => {
  const { screen, render } = await createDOM();
  await render(
    <SkillListItem
      skill={makeSkill({ current_revision: 5 })}
      selected={false}
      onClick$={makeOnClickStub()}
    />,
  );

  const meta = screen.querySelector('[data-testid="skill-list-item-meta"]');
  expect(meta).toBeTruthy();
  // Must contain "v5" — anti-drift gate.
  expect(meta?.textContent).toContain("v5");
  // Must NOT contain "undefined".
  expect(meta?.textContent).not.toContain("undefined");
});

test("[skill-list-item]: renders v prefix even when revision is missing", async () => {
  // Defensive: if current_revision is somehow undefined (legacy fixture,
  // optimistic update, type cast), the component must render without
  // crashing and must NOT produce "vundefined".
  const skill = makeSkill();
  // @ts-expect-error — simulate runtime missing field (anti-drift defense)
  skill.current_revision = undefined;

  const { screen, render } = await createDOM();
  await render(
    <SkillListItem
      skill={skill}
      selected={false}
      onClick$={makeOnClickStub()}
    />,
  );

  const meta = screen.querySelector('[data-testid="skill-list-item-meta"]');
  expect(meta).toBeTruthy();
  // Must NOT contain "undefined" (the prompts regression).
  expect(meta?.textContent).not.toContain("undefined");
  // Should still contain a "v" prefix.
  expect(meta?.textContent).toContain("v");
});
