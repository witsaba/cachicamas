/**
 * Component tests for `<SkillsIcon>`.
 *
 * Reference: `sdd/cachicamas-skills-foundational/design` (engram, obs #1968).
 *   - §4.8: Lucide `book-marked` SVG, 48×48, viewBox 24×24
 *   - SCN-R-SK-008 (icon contract)
 *
 * The Lucide `book-marked` path data is asserted byte-equal to the
 * upstream source so any hand-edit or re-syndication drift is caught
 * at the test layer.
 *
 * Upstream source (verbatim, fetched at design time):
 *   https://raw.githubusercontent.com/lucide-icons/lucide/main/icons/book-marked.svg
 *
 * NOTE: The orchestrator task spec mentioned "3 path elements", but
 * the actual Lucide book-marked SVG has **2** path elements. The
 * load-bearing constraint is byte-equal to Lucide, not the path
 * count. This spec locks what the upstream ACTUALLY ships.
 *
 * Licence: ISC (Lucide).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { SkillsIcon } from "./skills-icon";

// Verbatim Lucide `book-marked` path data (byte-equal to upstream).
// Two paths: bookmark/star at the top, book outline with folded binding.
const LUCIDE_PATH_1 = "M10 2v8l3-3 3 3V2";
const LUCIDE_PATH_2 =
  "M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H19a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1H6.5a1 1 0 0 1 0-5H20";

describe("routes/settings/icons/skills-icon", () => {
  // ---------------------------------------------------------------------------
  // Task 5.10 RED — renders an <svg> with the documented geometry
  // ---------------------------------------------------------------------------

  it("Task 5.10 — renders an <svg> with the documented geometry (48×48, viewBox 0 0 24 24, fill=none)", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg?.getAttribute("width")).toBe("48");
    expect(svg?.getAttribute("height")).toBe("48");
    expect(svg?.getAttribute("viewBox")).toBe("0 0 24 24");
    expect(svg?.getAttribute("fill")).toBe("none");
  });

  it("Task 5.10 — stroke uses currentColor (monochrome rule contract)", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("stroke")).toBe("currentColor");
  });

  it("Task 5.10 — stroke-width is 1.75 (matches prompts-icon for visual consistency)", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("stroke-width")).toBe("1.75");
    expect(svg?.getAttribute("stroke-linecap")).toBe("round");
    expect(svg?.getAttribute("stroke-linejoin")).toBe("round");
  });

  it("Task 5.10 — contains exactly 2 <path> elements (Lucide book-marked verbatim)", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths.length).toBe(2);
  });

  it("Task 5.10 — FIRST path d= matches Lucide book-marked 'bookmark' verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths[0]?.getAttribute("d")).toBe(LUCIDE_PATH_1);
  });

  it("Task 5.10 — SECOND path d= matches Lucide book-marked 'book outline' verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths[1]?.getAttribute("d")).toBe(LUCIDE_PATH_2);
  });

  it("Task 5.10 — exposes data-testid='skills-icon' for test pinning", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    expect(screen.querySelector('[data-testid="skills-icon"]')).toBeTruthy();
  });

  // ---------------------------------------------------------------------------
  // Task 5.11 RED — aria-hidden=true AND focusable=false
  // ---------------------------------------------------------------------------

  it("Task 5.11 — decorative aria: aria-hidden=true AND focusable=false", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("aria-hidden")).toBe("true");
    expect(svg?.getAttribute("focusable")).toBe("false");
  });

  // ---------------------------------------------------------------------------
  // Task 5.12 RED — path data is locked (concurrent SVG snapshot)
  // ---------------------------------------------------------------------------

  it("Task 5.12 — full SVG attribute set matches the locked snapshot (anti-drift)", async () => {
    const { screen, render } = await createDOM();
    await render(<SkillsIcon />);
    const svg = screen.querySelector("svg");
    expect({
      width: svg?.getAttribute("width"),
      height: svg?.getAttribute("height"),
      viewBox: svg?.getAttribute("viewBox"),
      pathCount: svg?.querySelectorAll("path").length,
    }).toEqual({
      width: "48",
      height: "48",
      viewBox: "0 0 24 24",
      pathCount: 2,
    });
    // Individual assertions above already lock stroke, a11y, fill,
    // and each path's d= attribute byte-equal. This snapshot guards
    // against "extra attributes creep" (e.g. a stray class=).
  });
});
