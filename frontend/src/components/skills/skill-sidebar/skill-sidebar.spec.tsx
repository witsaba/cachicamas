/**
 * SkillSidebar spec — covers filter input, "+ New Skill" button,
 * and filter behavior (case-insensitive substring match on name).
 *
 * Note: The sidebar passes the SELECTED name through to each
 * SkillListItem so PR2c can compose the highlight state from
 * selectedName. That threading is locked in task 6.5.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { beforeEach, test, expect } from "vitest";
import { SkillSidebar } from "./skill-sidebar";
import type { Skill } from "~/lib/skills-api";

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function makeSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 1,
    name: "pdf-cleanup",
    description: "Cleans up PDF metadata",
    body: "---\nname: pdf-cleanup\ndescription: x\n---\n",
    current_revision: 1,
    created_at: "2026-07-17T00:00:00Z",
    updated_at: "2026-07-17T00:00:00Z",
    deleted_at: null,
    ...overrides,
  };
}

function makeStub(): QRL<() => void> {
  return $(() => {});
}

let selectedNames: string[] = [];
let captureEnabled = false;
beforeEach(() => {
  selectedNames = [];
  captureEnabled = true;
});

function makeSelectingOnSelect(): QRL<(name: string) => void> {
  return $((name: string) => {
    if (captureEnabled) selectedNames.push(name);
  });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test("[skill-sidebar]: renders a filter input with placeholder", async () => {
  const { screen, render } = await createDOM();
  await render(
    <SkillSidebar
      skills={[]}
      selectedName={null}
      onSelect$={makeSelectingOnSelect()}
      onNewSkill$={makeStub()}
    />,
  );
  const input = screen.querySelector(
    '[data-testid="skill-sidebar-filter"]',
  ) as HTMLInputElement;
  expect(input).toBeTruthy();
  expect(input.placeholder.toLowerCase()).toContain("filter");
});

test("[skill-sidebar]: renders a 'New Skill' button", async () => {
  const { screen, render } = await createDOM();
  await render(
    <SkillSidebar
      skills={[]}
      selectedName={null}
      onSelect$={makeSelectingOnSelect()}
      onNewSkill$={makeStub()}
    />,
  );
  const btn = screen.querySelector(
    '[data-testid="skill-sidebar-new"]',
  ) as HTMLButtonElement;
  expect(btn).toBeTruthy();
  expect(btn.textContent?.toLowerCase()).toContain("new");
});

test("[skill-sidebar]: filter input narrows the visible list by name substring (case-insensitive)", async () => {
  const skills = [
    makeSkill({ id: 1, name: "pdf-cleanup" }),
    makeSkill({ id: 2, name: "image-resize" }),
    makeSkill({ id: 3, name: "csv-export" }),
  ];

  const { screen, render, userEvent } = await createDOM();
  await render(
    <SkillSidebar
      skills={skills}
      selectedName={null}
      onSelect$={makeSelectingOnSelect()}
      onNewSkill$={makeStub()}
    />,
  );

  // Sanity: all three list items render before filtering.
  expect(screen.querySelectorAll('[data-testid="skill-list-item"]').length).toBe(3);

  // Type "PDF" — uppercase. Case-insensitive match should hide the others.
  const input = screen.querySelector(
    '[data-testid="skill-sidebar-filter"]',
  ) as HTMLInputElement;
  input.value = "PDF";
  await userEvent(input, "input", { target: input });

  const visibleNames = screen
    .querySelectorAll('[data-testid="skill-list-item-name"]')
    .map((el) => el.textContent);
  expect(visibleNames).toEqual(["pdf-cleanup"]);
});

test("[skill-sidebar]: empty skill list shows the 'No skills yet' empty hint", async () => {
  const { screen, render } = await createDOM();
  await render(
    <SkillSidebar
      skills={[]}
      selectedName={null}
      onSelect$={makeSelectingOnSelect()}
      onNewSkill$={makeStub()}
    />,
  );
  // No list items rendered.
  expect(screen.querySelectorAll('[data-testid="skill-list-item"]').length).toBe(0);
  // The empty hint is shown inside the listbox.
  const listbox = screen.querySelector(
    '[data-testid="skill-sidebar-list"]',
  ) as HTMLElement;
  expect(listbox).toBeTruthy();
  expect(listbox.textContent?.toLowerCase()).toContain("no skills");
});
