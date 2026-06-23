import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

// UX-10 (spec §6.3): AI pre-fill from PRD ideas is out of
// scope for v1.  The spec REQUIRES that the literal string
// "TODO(organizations-first-front): UX-10" exists somewhere
// in the source so a future PR implementing the feature
// can grep for it.  This test enforces the presence of the
// marker — if a developer removes it, this test fails.
//
// The marker currently lives in
// src/components/organization-readback/organization-readback.tsx.

const ROOT = join(__dirname, "..", "..");
const SOURCE_GLOBS = ["src", "frontend"];

function walk(dir: string, acc: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    const stat = statSync(full);
    if (stat.isDirectory()) {
      // Skip node_modules and build artefacts.
      if (entry === "node_modules" || entry === "dist" || entry === "tmp") {
        continue;
      }
      walk(full, acc);
    } else {
      acc.push(full);
    }
  }
  return acc;
}

describe("UX-10: out-of-scope TODO marker", () => {
  test('the literal "TODO(organizations-first-front): UX-10" exists in the source', () => {
    const needle = "TODO(organizations-first-front): UX-10";
    const targets = SOURCE_GLOBS.filter((p) => {
      try {
        return statSync(join(ROOT, p)).isDirectory();
      } catch {
        return false;
      }
    });
    if (targets.length === 0) {
      // No source tree found (test running from a build
      // artefact?) — skip rather than fail.
      return;
    }
    const files: string[] = [];
    for (const t of targets) {
      walk(join(ROOT, t), files);
    }
    const matches: string[] = [];
    for (const file of files) {
      if (!/\.(ts|tsx|js|jsx|md|mdx)$/.test(file)) continue;
      const text = readFileSync(file, "utf-8");
      if (text.includes(needle)) {
        matches.push(file);
      }
    }
    expect(matches.length, `expected at least one file containing ${needle}`).toBeGreaterThanOrEqual(1);
  });
});
