/**
 * SkillListItem spec — covers name + version rendering, testid, click.
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

// Module-level capture (Qwik `$` forbids vi.fn in closures — see
// prompt-editor.spec.tsx comment for the pattern).
let capturedClicks = 0;
beforeEach(() => {
  capturedClicks = 0;
});

function makeCountingOnClick(): QRL<() => void> {
  return $(() => {
    capturedClicks++;
  });
}

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

test("[skill-list-item]: exposes a data-testid that distinguishes selected vs unselected", async () => {
  // The unselected item has testid "skill-list-item".
  const { screen: s1, render: r1 } = await createDOM();
  await r1(
    <ul>
      <SkillListItem
        skill={makeSkill({ name: "one" })}
        selected={false}
        onClick$={makeOnClickStub()}
      />
    </ul>,
  );
  const unselected = s1.querySelector('[data-testid="skill-list-item"]');
  expect(unselected).toBeTruthy();

  // The selected item has testid "skill-list-item-selected" so PR2c
  // can query both the generic and the highlighted variants.
  const { screen: s2, render: r2 } = await createDOM();
  await r2(
    <ul>
      <SkillListItem
        skill={makeSkill({ name: "two" })}
        selected={true}
        onClick$={makeOnClickStub()}
      />
    </ul>,
  );
  const selected = s2.querySelector('[data-testid="skill-list-item-selected"]');
  expect(selected).toBeTruthy();
});

test("[skill-list-item]: clicking the item invokes onClick$", async () => {
  // Verifies the button wires the click handler (the parent
  // SkillSidebar passes a QRL that selects the skill in its own
  // state — this test pins the wiring contract).
  const { screen, render, userEvent } = await createDOM();
  await render(
    <ul>
      <SkillListItem
        skill={makeSkill({ name: "clickable" })}
        selected={false}
        onClick$={makeCountingOnClick()}
      />
    </ul>,
  );

  const btn = screen.querySelector(
    '[data-testid="skill-list-item"]',
  ) as HTMLButtonElement;
  expect(btn).toBeTruthy();

  await userEvent(btn, "click");

  expect(capturedClicks).toBe(1);
});
