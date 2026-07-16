/**
 * Component tests for `<PromptsIcon>`.
 *
 * Reference: `sdd/settings-app-grid/{spec,design}.md` (engram).
 *   - REQ-3 (`stroke="currentColor"` SVG contract)
 *   - SCN-3.2 (first consumer obeys the contract)
 *
 * This file is the RED step for the PromptsIcon implementation.
 * Until `prompts-icon.tsx` exists, the import at the top of this
 * file will fail to resolve and the test file will be reported as
 * failing by vitest. That failure IS the RED state.
 *
 * The Lucide `file-pen` path data is asserted byte-equal to the
 * upstream source so any hand-edit or re-syndication drift is
 * caught at the test layer.
 *
 * Upstream source (verbatim):
 *   https://raw.githubusercontent.com/lucide-icons/lucide/main/icons/file-pen.svg
 * Licence: ISC (Lucide).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { PromptsIcon } from "./prompts-icon";

// Verbatim Lucide `file-pen` path data (byte-equal to upstream).
// Three paths: document outline, folded corner, pencil.
const LUCIDE_PATH_1 =
  "M12.659 22H18a2 2 0 0 0 2-2V8a2.4 2.4 0 0 0-.706-1.706l-3.588-3.588A2.4 2.4 0 0 0 14 2H6a2 2 0 0 0-2 2v9.34";
const LUCIDE_PATH_2 = "M14 2v5a1 1 0 0 0 1 1h5";
const LUCIDE_PATH_3 =
  "M10.378 12.622a1 1 0 0 1 3 3.003L8.36 20.637a2 2 0 0 1-.854.506l-2.867.837a.5.5 0 0 1-.62-.62l.836-2.869a2 2 0 0 1 .506-.853z";

describe("routes/settings/icons/prompts-icon", () => {
  it("REQ-3 / SCN-3.2 — renders an <svg> with the documented geometry (48×48, viewBox 0 0 24 24, fill=none)", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg?.getAttribute("width")).toBe("48");
    expect(svg?.getAttribute("height")).toBe("48");
    expect(svg?.getAttribute("viewBox")).toBe("0 0 24 24");
    expect(svg?.getAttribute("fill")).toBe("none");
  });

  it("REQ-3 — stroke uses currentColor (monochrome rule contract)", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("stroke")).toBe("currentColor");
  });

  it("REQ-3 — stroke-width is 1.75 (between empty-state's 1.5 and sign-out's 2)", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("stroke-width")).toBe("1.75");
    expect(svg?.getAttribute("stroke-linecap")).toBe("round");
    expect(svg?.getAttribute("stroke-linejoin")).toBe("round");
  });

  it("REQ-3 — decorative aria: aria-hidden=true AND focusable=false", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const svg = screen.querySelector("svg");
    expect(svg?.getAttribute("aria-hidden")).toBe("true");
    expect(svg?.getAttribute("focusable")).toBe("false");
  });

  it("contains exactly 3 <path> elements (document outline + folded corner + pencil)", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths.length).toBe(3);
  });

  it("FIRST path d= matches Lucide file-pen 'document outline' verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths[0]?.getAttribute("d")).toBe(LUCIDE_PATH_1);
  });

  it("SECOND path d= matches Lucide file-pen 'folded corner' verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths[1]?.getAttribute("d")).toBe(LUCIDE_PATH_2);
  });

  it("THIRD path d= matches Lucide file-pen 'pencil' verbatim", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const paths = screen.querySelectorAll("path");
    expect(paths[2]?.getAttribute("d")).toBe(LUCIDE_PATH_3);
  });

  it("renders no text content (decorative — visible label 'Prompts' is the affordance)", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    const svg = screen.querySelector("svg");
    expect((svg?.textContent ?? "").trim()).toBe("");
  });

  it("exposes data-testid='prompts-icon' for test pinning", async () => {
    const { screen, render } = await createDOM();
    await render(<PromptsIcon />);
    expect(screen.querySelector('[data-testid="prompts-icon"]')).toBeTruthy();
  });
});