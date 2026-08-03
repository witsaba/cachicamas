import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";
import { stateChangingFetch } from "~/lib/csrf";

/**
 * TDD coverage for the CSRF defense-in-depth header.
 *
 * Reference: sdd/security-vulnerability-remediation/spec/csrf-origin-validation
 *   REQ-02 — state-changing requests SHALL include X-Requested-With.
 *   REQ-03 — safe methods (GET/HEAD/OPTIONS) SHALL NOT be gated.
 *
 * The helper is a thin wrapper around `fetch`; the tests focus on
 * the contract: which methods carry the header, which do not, and
 * whether user-supplied headers survive the merge.
 */
describe("stateChangingFetch — CSRF defense-in-depth header", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    globalThis.fetch = vi.fn().mockResolvedValue(new Response(null, { status: 200 }));
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  function captureCallHeaders(): Record<string, string> {
    const mock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const [, init] = mock.mock.calls[0] as [string, RequestInit];
    const headers: Record<string, string> = {};
    if (init?.headers) {
      const h = new Headers(init.headers as HeadersInit);
      h.forEach((value, key) => {
        headers[key.toLowerCase()] = value;
      });
    }
    return headers;
  }

  it("RED-T-CSRF-001: POST fetch includes X-Requested-With: XMLHttpRequest", async () => {
    await stateChangingFetch("/organizations", { method: "POST" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBe("XMLHttpRequest");
  });

  it("RED-T-CSRF-002: PUT fetch includes X-Requested-With: XMLHttpRequest", async () => {
    await stateChangingFetch("/workspaces/1", { method: "PUT" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBe("XMLHttpRequest");
  });

  it("RED-T-CSRF-003: PATCH fetch includes X-Requested-With: XMLHttpRequest", async () => {
    await stateChangingFetch("/workspaces/1", { method: "PATCH" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBe("XMLHttpRequest");
  });

  it("RED-T-CSRF-004: DELETE fetch includes X-Requested-With: XMLHttpRequest", async () => {
    await stateChangingFetch("/workspaces/1", { method: "DELETE" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBe("XMLHttpRequest");
  });

  it("RED-T-CSRF-005: GET fetch does NOT add X-Requested-With", async () => {
    await stateChangingFetch("/workspaces");
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBeUndefined();
  });

  it("RED-T-CSRF-006: HEAD fetch does NOT add X-Requested-With", async () => {
    await stateChangingFetch("/workspaces/1", { method: "HEAD" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBeUndefined();
  });

  it("RED-T-CSRF-007: OPTIONS fetch does NOT add X-Requested-With", async () => {
    await stateChangingFetch("/workspaces/1", { method: "OPTIONS" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBeUndefined();
  });

  it("RED-T-CSRF-008: default method (no init) is treated as GET — no header added", async () => {
    await stateChangingFetch("/workspaces");
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBeUndefined();
  });

  it("RED-T-CSRF-009: caller-supplied headers survive on a POST", async () => {
    await stateChangingFetch("/organizations", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
    });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBe("XMLHttpRequest");
    expect(headers["content-type"]).toBe(
      "application/x-www-form-urlencoded",
    );
  });

  it("RED-T-CSRF-010: lowercase method 'post' is treated as POST — header added", async () => {
    await stateChangingFetch("/organizations", { method: "post" });
    const headers = captureCallHeaders();
    expect(headers["x-requested-with"]).toBe("XMLHttpRequest");
  });
});
