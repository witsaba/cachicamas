/**
 * Contrast, measured against every ground a token can land on.
 *
 * This exists because of a specific failure: on the previous build a hairline
 * was reported at 3.1:1 and shipped at 2.75:1, because it was measured against
 * a panel colour that was then lightened in the same batch. A value picked
 * against one surface quietly fails on another, and nothing in a screenshot
 * shows it.
 *
 * So the floors are computed from `global.css` itself, on every ground, on
 * every run. Change a token and this fails before a person has to notice.
 *
 * The floors:
 *   - body and small text ≥ 4.5:1 (WCAG 1.4.3)
 *   - a control's own boundary ≥ 3:1 (WCAG 1.4.11) — on this surface the
 *     border is how the control is found, so it is not decorative
 *   - text on a filled surface ≥ 4.5:1, including every department field
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const css = readFileSync(
  fileURLToPath(new URL("../global.css", import.meta.url)),
  "utf8",
);

const TOKENS: Record<string, string> = Object.fromEntries(
  [...css.matchAll(/--color-([a-z-]+):\s*(#[0-9a-fA-F]{6});/g)].map((m) => [
    m[1],
    m[2],
  ]),
);

const channel = (c: number) =>
  c / 255 <= 0.04045 ? c / 255 / 12.92 : ((c / 255 + 0.055) / 1.055) ** 2.4;

function luminance(hex: string): number {
  const n = hex.replace("#", "");
  return (
    0.2126 * channel(parseInt(n.slice(0, 2), 16)) +
    0.7152 * channel(parseInt(n.slice(2, 4), 16)) +
    0.0722 * channel(parseInt(n.slice(4, 6), 16))
  );
}

function ratio(a: string, b: string): number {
  const [la, lb] = [luminance(a), luminance(b)];
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

/** Every surface a token can sit on. Measuring against `surface` alone is the
 *  mistake this file exists to prevent. */
const GROUNDS = ["surface", "canvas", "sunken"] as const;

const DEPARTMENTS = [
  "dept-assistant",
  "dept-finance",
  "dept-support",
  "dept-integrations",
  "dept-data",
  "dept-engineering",
];

describe("contrast", () => {
  it("parses the whole palette out of global.css", () => {
    // Non-vacuous: if the parse breaks, every assertion below passes silently.
    expect(Object.keys(TOKENS).length).toBeGreaterThan(20);
    for (const g of GROUNDS) expect(TOKENS[g], g).toMatch(/^#[0-9a-f]{6}$/i);
  });

  it("clears 4.5:1 for every text colour on every ground", () => {
    const text = [
      "ink",
      "ink-mid",
      "ink-soft",
      "brand",
      "brand-press",
      "ok",
      "waiting",
      "stop",
      "idle",
      ...DEPARTMENTS,
    ];
    const failures: string[] = [];
    for (const name of text) {
      for (const ground of GROUNDS) {
        const r = ratio(TOKENS[name], TOKENS[ground]);
        if (r < 4.5) failures.push(`${name} on ${ground}: ${r.toFixed(2)}`);
      }
    }
    expect(failures).toEqual([]);
  });

  it("clears 3:1 for a control's own boundary on every ground", () => {
    const failures: string[] = [];
    for (const ground of GROUNDS) {
      const r = ratio(TOKENS["line-control"], TOKENS[ground]);
      if (r < 3) failures.push(`line-control on ${ground}: ${r.toFixed(2)}`);
    }
    expect(failures).toEqual([]);
  });

  it("clears 4.5:1 for text on every filled surface", () => {
    const filled = [
      "brand",
      "brand-press",
      "stop",
      "ok",
      "waiting",
      "deep",
      "ink",
      ...DEPARTMENTS,
    ];
    const failures: string[] = [];
    for (const name of filled) {
      const r = ratio(TOKENS["ink-inverse"], TOKENS[name]);
      if (r < 4.5) failures.push(`ink-inverse on ${name}: ${r.toFixed(2)}`);
    }
    // The selected-row tint carries brand ink, not white.
    const tint = ratio(TOKENS["brand"], TOKENS["brand-tint"]);
    if (tint < 4.5) failures.push(`brand on brand-tint: ${tint.toFixed(2)}`);
    expect(failures).toEqual([]);
  });

  it("gives every department a distinct hue", () => {
    const values = DEPARTMENTS.map((d) => TOKENS[d]);
    expect(new Set(values).size).toBe(values.length);
  });
});
