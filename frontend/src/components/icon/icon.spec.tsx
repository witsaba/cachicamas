/**
 * The icon set.
 *
 * Two things are worth pinning. One: every icon is drawn to the same grid, at
 * the same stroke weight — a set assembled from several sources is the tell
 * that nobody decided. Two: no icon in this product is ever the only carrier
 * of meaning, so every one of them is hidden from assistive technology and
 * ships beside a word.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { Icon, type IconName } from "./icon";

const NAMES: readonly IconName[] = [
  "desk",
  "chat",
  "staff",
  "teams",
  "org",
  "settings",
  "plus",
  "check",
  "arrow-right",
  "stop",
  "shield",
  "send",
  "clock",
  "tool",
  "chevron-down",
  "external",
];

describe("components/icon", () => {
  it("draws every icon on one 24px grid at one stroke weight", async () => {
    for (const name of NAMES) {
      const { screen, render } = await createDOM();
      await render(<Icon name={name} />);
      const svg = screen.querySelector("svg");
      expect(svg, name).toBeTruthy();
      expect(svg?.getAttribute("viewBox"), name).toBe("0 0 24 24");
      expect(svg?.getAttribute("stroke-width"), name).toBe("1.75");
      expect(svg?.getAttribute("stroke-linecap"), name).toBe("round");
      expect(svg?.getAttribute("fill"), name).toBe("none");
      // Stroked, not filled: a mixed set reads as two sets.
      expect(svg?.getAttribute("stroke"), name).toBe("currentColor");
      const path = svg?.querySelector("path");
      expect(path?.getAttribute("d")?.length ?? 0, name).toBeGreaterThan(10);
    }
  });

  it("is never announced — the word beside it is", async () => {
    const { screen, render } = await createDOM();
    await render(<Icon name="shield" />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("aria-hidden")).toBe("true");
    expect(svg?.getAttribute("focusable")).toBe("false");
  });

  it("renders at the size it is asked for", async () => {
    const { screen, render } = await createDOM();
    await render(<Icon name="check" size={14} />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("width")).toBe("14");
    expect(svg?.getAttribute("height")).toBe("14");
  });
});
