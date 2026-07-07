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
  // T-WS-2i-014 amended (S-WS-AUTH-CHAIN-SSR-001):
  //   `requireAuthRedirect` + `requireOwnboarding` are now called
  //   from the `onRequest` middleware rather than a routeLoader$.
  //   The reason is that the same middleware captures the inbound
  //   Cookie header into AsyncLocalStorage so SSR-time fetches in
  //   useTask$ can re-attach it (see ~/lib/with-ssr-cookie.ts).
it("T-WS-2i-014 amended: index.tsx wires requireAuthRedirect + requireOwnboarding in onRequest", () => {
    const src = readFileSync(`${HERE}index.tsx`, "utf8");
    expect(src).toContain("requireAuthRedirect");
    expect(src).toContain("requireOwnboarding");
    expect(src).toContain("onRequest");
    expect(src).toContain("setSsrCookieHeader(event.request.headers.get(\"cookie\") ?? \"\")");
  });
});
