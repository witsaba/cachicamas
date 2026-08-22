/**
 * FunctionRail — the dock.
 *
 * Two properties matter. First, every cell is a word plus its key, never a
 * picture: that is what makes the dock usable without recognising an icon set
 * (PRODUCT.md § Accessibility). Second, the dock and the register must agree —
 * a destination in one and not the other is how a launcher starts lying.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { FunctionRail } from "./function-rail";
import { ARCHETYPES, archetypeHref } from "~/lib/mock/registry";

describe("components/os/function-rail", () => {
  it("is a labelled navigation landmark", async () => {
    const { screen, render } = await createDOM();
    await render(<FunctionRail />);
    const nav = screen.querySelector('[data-testid="function-rail"]');
    expect(nav?.tagName.toLowerCase()).toBe("nav");
    expect(nav?.getAttribute("aria-label")).toBe("Applications");
  });

  it("carries one cell per archetype, plus System", async () => {
    const { screen, render } = await createDOM();
    await render(<FunctionRail />);
    for (const a of ARCHETYPES) {
      const cell = screen.querySelector(`[data-testid="dock-${a.fkey}"]`);
      expect(cell, a.fkey).toBeTruthy();
      expect(cell?.getAttribute("href"), a.fkey).toBe(archetypeHref(a));
    }
    const system = screen.querySelector('[data-testid="dock-F8"]');
    expect(system?.getAttribute("href")).toBe("/settings/");
  });

  it("names every cell in words, never in a glyph alone", async () => {
    const { screen, render } = await createDOM();
    await render(<FunctionRail />);
    for (const a of ARCHETYPES) {
      const cell = screen.querySelector(`[data-testid="dock-${a.fkey}"]`);
      const text = cell?.textContent ?? "";
      expect(text, a.fkey).toContain(a.fkey);
      // The dock's short name, not the display name: a cell that read
      // "Database Administrator" would push the rest off a laptop screen.
      expect(text, a.fkey).toContain(a.dockName);
    }
    // No images anywhere in the dock — the whole rail is type.
    expect(
      screen
        .querySelector('[data-testid="function-rail"]')
        ?.querySelector("img"),
    ).toBeFalsy();
  });

  it("assigns each function key exactly once", async () => {
    const { screen, render } = await createDOM();
    await render(<FunctionRail />);
    const cells = Array.from(
      screen.querySelectorAll('[data-testid^="dock-"]'),
    ).map((c) => c.getAttribute("data-testid"));
    expect(new Set(cells).size).toBe(cells.length);
  });

  it("marks no cell current until the browser has said where it is", async () => {
    // Server-side there is no location to read; guessing would light the wrong
    // cell on first paint.
    const { screen, render } = await createDOM();
    await render(<FunctionRail />);
    expect(screen.querySelector('[aria-current="page"]')).toBeFalsy();
  });

  it("dims the specialists that do not exist without hiding them", async () => {
    const { screen, render } = await createDOM();
    await render(<FunctionRail />);
    const unplanned = ARCHETYPES.filter((a) => a.state === "unplanned");
    expect(unplanned.length).toBeGreaterThan(0);
    for (const a of unplanned) {
      const cell = screen.querySelector(`[data-testid="dock-${a.fkey}"]`);
      expect(cell, a.fkey).toBeTruthy();
      expect(cell?.className, a.fkey).toContain("text-fg-dim");
    }
  });
});
