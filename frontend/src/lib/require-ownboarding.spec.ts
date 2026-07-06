/**
 * `requireOwnboarding` unit tests.
 *
 * Reference: `openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md`
 *   R-OW-006 (S-OW-050..054) — helper behavior.
 *   R-OW-008 (S-OW-070..073) — failure-mode fallback.
 *
 * The tests stub the `getSetupState` import (the only external
 * dependency the helper has) and assert on the helper's three
 * outcomes:
 *   - hasOrganization=true  → no-op
 *   - hasOrganization=false → throw redirect to /ownboarding
 *   - transport error or malformed shape → log warning + no-op
 *
 * The no-redirect-loop case (S-OW-054) is exercised by passing a
 * mock RequestEvent with pathname === "/ownboarding" and asserting
 * that the redirect is NOT thrown.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { requireOwnboarding } from "./require-ownboarding";

// Mock the API client so the tests don't touch a real backend.
vi.mock("./api", () => ({
  getSetupState: vi.fn(),
}));

// Import the mocked function after the mock declaration so the
// test sees the mocked version.
import { getSetupState } from "./api";
const mockedGetSetupState = vi.mocked(getSetupState);

/**
 * Build a minimal RequestEvent stub that satisfies the helper's
 * surface (url + redirect). The redirect method captures the call
 * and rethrows an Error so the test can assert the exact redirect.
 *
 * Cast through `unknown` because `RequestEventCommon` is a wide
 * interface (status, locale, rewrite, error, ...) and the helper
 * only reads `url` + calls `redirect`. The test is structurally
 * isolated; it never depends on the other surface.
 */
function makeEvent(pathname: string) {
  const url = new URL(`http://localhost${pathname}`);
  return {
    url,
    redirect: ((status: number, location: string) => {
      throw new Error(`REDIRECT ${status} ${location}`);
    }) as never,
  } as unknown as Parameters<typeof requireOwnboarding>[0];
}

describe("requireOwnboarding", () => {
  let warnSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    mockedGetSetupState.mockReset();
    warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    warnSpy.mockRestore();
  });

  // S-OW-050
  it("is a no-op when hasOrganization=true", async () => {
    mockedGetSetupState.mockResolvedValue({ hasOrganization: true });
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).not.toHaveBeenCalled();
  });

  // S-OW-051
  it("throws a redirect to /ownboarding when hasOrganization=false", async () => {
    mockedGetSetupState.mockResolvedValue({ hasOrganization: false });
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).rejects.toThrow(
      "REDIRECT 302 /ownboarding",
    );
    expect(warnSpy).not.toHaveBeenCalled();
  });

  // S-OW-052 — failure-mode fallback: transport error
  it("logs a warning and is a no-op when getSetupState rejects", async () => {
    mockedGetSetupState.mockRejectedValue(new Error("network down"));
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).toHaveBeenCalledTimes(1);
    expect(warnSpy.mock.calls[0][0]).toContain("setup-state fetch failed");
  });

  // S-OW-053 — defensive fallback: malformed shape
  it("logs a warning and is a no-op when getSetupState resolves with unexpected shape", async () => {
    // Cast through `unknown` to simulate a backend drift that bypasses
    // the api.ts guard.
    mockedGetSetupState.mockResolvedValue(
      undefined as unknown as Awaited<ReturnType<typeof getSetupState>>,
    );
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).toHaveBeenCalledTimes(1);
    expect(warnSpy.mock.calls[0][0]).toContain("unexpected setup-state shape");
  });

  // S-OW-054 — no-redirect-loop guard
  it("is a no-op when currentPath is /ownboarding, even with hasOrganization=false", async () => {
    mockedGetSetupState.mockResolvedValue({ hasOrganization: false });
    const event = makeEvent("/ownboarding");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).not.toHaveBeenCalled();
  });

  // S-OW-070 — explicit failure-mode test
  it("treats a thrown Error from getSetupState as the optimistic no-op path", async () => {
    mockedGetSetupState.mockImplementation(() => {
      throw new Error("boom");
    });
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).toHaveBeenCalledTimes(1);
  });

  // S-OW-071 — explicit redirect path
  it("uses event.redirect(302, /ownboarding) on hasOrganization=false", async () => {
    mockedGetSetupState.mockResolvedValue({ hasOrganization: false });
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).rejects.toThrow(
      "REDIRECT 302 /ownboarding",
    );
  });

  // S-OW-072 — hasOrganization=true path
  it("renders normally when hasOrganization=true and pathname is not /ownboarding", async () => {
    mockedGetSetupState.mockResolvedValue({ hasOrganization: true });
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
  });

  // S-OW-073 — defensive fallback: null shape (drift test)
  it("treats null as a malformed response and falls back to no-op", async () => {
    mockedGetSetupState.mockResolvedValue(
      null as unknown as Awaited<ReturnType<typeof getSetupState>>,
    );
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).toHaveBeenCalled();
  });

  // Defensive fallback: hasOrganization is a string (schema drift)
  it("treats hasOrganization=\"true\" (string) as malformed and falls back to no-op", async () => {
    mockedGetSetupState.mockResolvedValue(
      { hasOrganization: "true" } as unknown as Awaited<
        ReturnType<typeof getSetupState>
      >,
    );
    const event = makeEvent("/home");
    await expect(requireOwnboarding(event)).resolves.toBeUndefined();
    expect(warnSpy).toHaveBeenCalled();
  });
});