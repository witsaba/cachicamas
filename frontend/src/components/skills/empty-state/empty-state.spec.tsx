/**
 * Behavioural spec for the SkillStudio empty state.
 *
 * Reference: `sdd/cachicamas-skills-foundational/{spec,design}` (engram).
 *   - S-FE-2: /settings/skills renders empty state when list empty
 *   - the empty state MUST surface a CTA so the user has a path
 *     forward from "no skills yet"
 *
 * RED step: until `empty-state.tsx` exists, the import fails and
 * the suite fails — that failure IS the RED state.
 *
 * API note: the skills EmptyState is intentionally IDENTICAL in
 * shape to the prompts EmptyState (just a `onCreate$` handler). It
 * could be reused, but a copy keeps the per-feature vocabulary
 * ("Create your first skill") without coupling the two routes.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { describe, it, expect, beforeEach } from "vitest";

import { EmptyState } from "./empty-state";

let createCalls: number = 0;
let captureEnabled = false;

beforeEach(() => {
  createCalls = 0;
  captureEnabled = true;
});

function makeCreateStub(): QRL<() => void> {
  return $(() => {
    if (captureEnabled) createCalls += 1;
  });
}

describe("components/skills/empty-state", () => {
  it("TestEmptyState_RendersNoSkillsMessage — shows the 'no skills yet' headline + descriptive subtext", async () => {
    const { screen, render } = await createDOM();
    await render(<EmptyState onCreate$={makeCreateStub()} />);
    const text = document.body.textContent ?? "";
    expect(text.toLowerCase()).toContain("no skills");
  });

  it("TestEmptyState_HasCreateCTA — clicking the 'Create your first skill' button invokes onCreate$", async () => {
    const { screen, render, userEvent } = await createDOM();
    await render(<EmptyState onCreate$={makeCreateStub()} />);
    const cta = screen.querySelector(
      '[data-testid="empty-state-create"]',
    ) as HTMLElement | null;
    expect(cta).toBeTruthy();
    await userEvent(cta!, "click");
    expect(createCalls).toBe(1);
  });
});