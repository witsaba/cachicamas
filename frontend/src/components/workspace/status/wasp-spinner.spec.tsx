/**
 * WaspSpinner — the cachicama wasp that replaces the green "Working
 * now" dot.
 *
 * Two contracts worth pinning:
 *   1. The SVG renders on the 32×32 grid (consistent with the icon
 *      family at 24×24; the wasp needs the extra room for leg + wing
 *      detail).
 *   2. Both wing paths exist with their `wasp-wing-left` /
 *      `wasp-wing-right` classes intact — the CSS keyframes in
 *      global.css rotate around those pivots. Renaming the classes
 *      here without touching the CSS would silently stop the flap.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { WaspSpinner } from "./wasp-spinner";

describe("components/workspace/status/wasp-spinner", () => {
  it("renders an inline svg with the 32×32 grid and an aria-label", async () => {
    const { screen, render } = await createDOM();
    await render(<WaspSpinner />);
    const svg = screen.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg?.getAttribute("viewBox")).toBe("0 0 32 32");
    expect(svg?.getAttribute("aria-label")).toBe("Assistant is working");
    expect(svg?.getAttribute("role")).toBe("img");
  });

  it("carries both wing paths with their animation pivot classes", async () => {
    const { screen, render } = await createDOM();
    await render(<WaspSpinner />);
    const left = screen.querySelector(".wasp-wing-left");
    const right = screen.querySelector(".wasp-wing-right");
    expect(left).toBeTruthy();
    expect(right).toBeTruthy();
    expect(left?.tagName.toLowerCase()).toBe("path");
    expect(right?.tagName.toLowerCase()).toBe("path");
  });

  it("draws the body, head, abdomen, thorax and stinger", async () => {
    const { screen, render } = await createDOM();
    await render(<WaspSpinner />);
    const svg = screen.querySelector("svg");
    // Two ellipses for eyes + head + thorax = 4 ellipses minimum.
    const ellipses = svg?.querySelectorAll("ellipse") ?? [];
    expect(ellipses.length).toBeGreaterThanOrEqual(4);
    // One small filled triangle for the stinger, distinct from the
    // dark red of the rest of the body.
    const paths = Array.from(svg?.querySelectorAll("path") ?? []);
    const hasStinger = paths.some(
      (p) => p.getAttribute("fill") === "#991b1b",
    );
    expect(hasStinger).toBe(true);
  });

  it("the compound eyes carry the project's green-dot colour (the eyes ARE the indicator)", async () => {
    // The wasp IS the working cue, no accompanying word. The eyes are
    // the visual focal point — same colour the previous `bg-ok` dot
    // carried (var(--color-ok) = #0e7c5a), so the wasp reads as a
    // richer, themed version of the same affordance.
    const { screen, render } = await createDOM();
    await render(<WaspSpinner />);
    const svg = screen.querySelector("svg");
    const ellipses = Array.from(svg?.querySelectorAll("ellipse") ?? []);
    const greenEyes = ellipses.filter(
      (e) => e.getAttribute("fill") === "#0e7c5a",
    );
    expect(greenEyes.length).toBeGreaterThanOrEqual(2);
  });

  it("forwards class to the root svg (consumer sizing)", async () => {
    const { screen, render } = await createDOM();
    await render(<WaspSpinner class="h-3.5 w-3.5 text-ok" />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("class")).toContain("h-3.5");
    expect(svg?.getAttribute("class")).toContain("wasp-spinner");
  });
});
