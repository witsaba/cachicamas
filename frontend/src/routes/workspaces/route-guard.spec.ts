/**
 * /workspaces route-guard structural test.
 *
 * Asserts that the route's source contains the two structural primitives
 * required by the locked chain: requireAuthRedirect (anon redirect) +
 * requireOwnboarding (no-org redirect). This is the canonical
 * route-guard test pattern from the ownboarding PR.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const HERE = fileURLToPath(new URL(".", import.meta.url));

describe("/workspaces route-guard wiring (PR2-i)", () => {
  it("T-WS-2i-014: index.tsx wires requireAuthRedirect + requireOwnboarding", () => {
    const src = readFileSync(`${HERE}index.tsx`, "utf8");
    expect(src).toContain("requireAuthRedirect");
    expect(src).toContain("requireOwnboarding");
    expect(src).toContain("routeLoader$");
  });
});
