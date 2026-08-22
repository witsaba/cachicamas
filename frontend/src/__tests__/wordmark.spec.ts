/**
 * The name is lowercase. Always.
 *
 * `PRODUCT.md § Brand Commitments` states it, and it had already drifted: one
 * route head read "Profile — Cachicamas" and one read "Sign out — Cachicamas"
 * while the other ten were lowercase. Title-casing a wordmark is the kind of
 * thing nobody notices in review and everybody notices in a browser tab.
 */
import { describe, it, expect } from "vitest";
import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));

function sourceFiles(dir: string, prefix = ""): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const rel = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.isDirectory()) out.push(...sourceFiles(`${dir}/${entry.name}`, rel));
    else if (/\.tsx?$/.test(entry.name) && !/\.spec\.tsx?$/.test(entry.name)) {
      out.push(rel);
    }
  }
  return out;
}

describe("the wordmark", () => {
  it("is never title-cased in shipped source", () => {
    const files = sourceFiles(root);
    // Non-vacuous: a walk that finds nothing proves nothing.
    expect(files.length).toBeGreaterThan(40);
    const offenders: string[] = [];
    for (const rel of files) {
      const source = readFileSync(root + rel, "utf8");
      source.split("\n").forEach((line, i) => {
        // `X-Cachicamas-Timestamp` and `X-Cachicamas-Signature` are HTTP
        // header names on a wire contract the backend also implements. They
        // are not the wordmark and renaming them would break the callback.
        const copy = line.replace(/X-Cachicamas-[A-Za-z]+/g, "");
        if (/\bCachicamas\b/.test(copy)) offenders.push(`${rel}:${i + 1}`);
      });
    }
    expect(offenders).toEqual([]);
  });
});
