/**
 * Every horizontal scroll container must also be its own containing block.
 *
 * The defect this exists to prevent, found on a 390px viewport: the plan
 * comparison table sits in an `overflow-x-auto` wrapper so it scrolls inside
 * itself. Its cells carry visually-hidden "Included" / "Not included" labels,
 * which `sr-only` implements with `position: absolute`. An absolutely
 * positioned element is clipped by a scroll container ONLY when that container
 * is also its containing block — and a static wrapper is not. So the labels
 * escaped to the table's full 672px width and dragged the whole document into
 * a 220px sideways scroll, while the table itself sat perfectly still.
 *
 * It cost an hour to find because nothing looked wrong: no element's box
 * overflowed, the wrapper's own clientWidth was correct, and `overflow-x: clip`
 * on the parent did not help. The fix is one word, so the guard is one grep.
 */
import { describe, it, expect } from "vitest";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));

/** Every `.tsx` under `src/`. Walked by hand: `fs.globSync` is newer than the
 *  TypeScript version this project pins, and a dependency for one glob is not
 *  worth it. */
function tsxFiles(dir: string, prefix = ""): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const rel = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.isDirectory()) out.push(...tsxFiles(`${dir}/${entry.name}`, rel));
    else if (entry.name.endsWith(".tsx")) out.push(rel);
  }
  return out;
}

describe("horizontal scrollers", () => {
  it("are always `relative`, so absolutely positioned children cannot escape", () => {
    const files = tsxFiles(root);
    // Non-vacuous: this guard is worthless if the walk finds nothing.
    expect(files.length).toBeGreaterThan(20);
    const offenders: string[] = [];
    for (const rel of files) {
      const source = readFileSync(root + rel, "utf8");
      for (const match of source.matchAll(/class="([^"]*overflow-x-auto[^"]*)"/g)) {
        const cls = match[1];
        if (!/\brelative\b/.test(cls)) {
          offenders.push(`${rel}: ${cls}`);
        }
      }
    }
    expect(offenders).toEqual([]);
  });
});
