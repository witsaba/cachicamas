/**
 * Behavioural spec for `routes/settings/index.tsx`.
 *
 * Reference: `sdd/settings-app-grid/{spec,design}.md` (engram).
 *   - REQ-10 (`/settings` index route uses canonical guard chain)
 *   - SCN-10.1 / SCN-10.2 (auth gate + grid render)
 *   - REQ-11 (headerless — no `<h1>`)
 *   - SCN-11.1
 *
 * RED step: until `./index` exists, the import fails and the suite
 * is reported as failing by vitest. That failure IS the RED state.
 *
 * Mock strategy:
 *   The three guard helpers (`setSsrCookieHeader`,
 *   `requireAuthRedirect`, `requireOwnboarding`) are mocked with
 *   `vi.mock` factories so the test can:
 *     - call the exported `onRequest` and verify the call ORDER
 *       matches the canonical guard chain (design §7)
 *     - render the component without spinning up the full
 *       Qwik City request context (the `<main>` + `<div>` DOM
 *       needs no auth, just the three mocked guards)
 *
 * Module-level coverage:
 *   `head.title === "Settings — Cachicamas"` is asserted by reading
 *   the named `head` export and asserting its `title` property.
 *   This catches a regression where the head accidentally drops the
 *   em-dash or swaps the brand suffix.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect, vi } from "vitest";

// Mock the three guard helpers in the canonical order. The
// `vi.hoisted` shim is required so the mock factory closes over
// the spy variables when vitest hoists `vi.mock` calls to the top
// of the file (Vitest's mock hoisting contract).
const { setSsrSpy, requireAuthSpy, requireOwnboardingSpy } = vi.hoisted(
  () => ({
    setSsrSpy: vi.fn(),
    requireAuthSpy: vi.fn(),
    requireOwnboardingSpy: vi.fn(),
  }),
);

vi.mock("~/lib/ssr-cookie-context", () => ({
  setSsrCookieHeader: setSsrSpy,
}));

vi.mock("~/lib/require-auth-redirect", () => ({
  requireAuthRedirect: requireAuthSpy,
}));

vi.mock("~/lib/require-ownboarding", () => ({
  requireOwnboarding: requireOwnboardingSpy,
}));

import Index, { head, onRequest } from "./index";

describe("routes/settings/index — canonical guard chain + grid render", () => {
  it("REQ-10 / SCN-10.1 — onRequest runs setSsrCookieHeader → requireAuthRedirect → requireOwnboarding (in that order)", async () => {
    setSsrSpy.mockClear();
    requireAuthSpy.mockClear();
    requireOwnboardingSpy.mockClear();

    // Synthesise the minimum RequestEvent shape the onRequest body
    // needs. The route only reads `event.request.headers.get("cookie")`
    // and passes `event` through to the guards (which are mocked —
    // they don't read any fields).
    const fakeEvent = {
      request: { headers: { get: () => "session=abc123" } },
    } as unknown as Parameters<typeof onRequest>[0];

    await onRequest(fakeEvent);

    // 1. All three guards were invoked.
    expect(setSsrSpy).toHaveBeenCalledTimes(1);
    expect(requireAuthSpy).toHaveBeenCalledTimes(1);
    expect(requireOwnboardingSpy).toHaveBeenCalledTimes(1);

    // 2. setSsrCookieHeader received the inbound cookie string.
    expect(setSsrSpy).toHaveBeenCalledWith("session=abc123");

    // 3. setSsrCookieHeader was called BEFORE requireAuthRedirect,
    //    which was called BEFORE requireOwnboarding. The canonical
    //    guard chain (design §7) MUST hold — otherwise the SSR
    //    cookie context misses the auth check, and the requireOwnboarding
    //    redirect happens before the cookie is captured.
    const setSsrOrder = setSsrSpy.mock.invocationCallOrder[0];
    const requireAuthOrder = requireAuthSpy.mock.invocationCallOrder[0];
    const requireOwnboardingOrder =
      requireOwnboardingSpy.mock.invocationCallOrder[0];
    expect(setSsrOrder).toBeLessThan(requireAuthOrder);
    expect(requireAuthOrder).toBeLessThan(requireOwnboardingOrder);
  });

  it("SCN-10.2 — renders <main> wrapping a grid with data-testid='settings-grid'", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const grid = screen.querySelector('[data-testid="settings-grid"]');
    expect(grid).toBeTruthy();
    // The grid sits inside a <main> (semantic landmark; SCN-10.2).
    const main = grid?.closest("main");
    expect(main).toBeTruthy();
  });

  it("grid class is 2-col on mobile + 3-col on sm + 4-col on md (design §7.1)", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const grid = screen.querySelector(
      '[data-testid="settings-grid"]',
    ) as HTMLElement | null;
    expect(grid).toBeTruthy();
    expect(grid?.className).toContain("grid");
    expect(grid?.className).toContain("grid-cols-2");
    expect(grid?.className).toContain("sm:grid-cols-3");
    expect(grid?.className).toContain("md:grid-cols-4");
  });

  it("renders exactly ONE <SettingCard> (Prompts only in v1)", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const cards = screen.querySelectorAll(
      '[data-testid="settings-card-prompts"]',
    );
    expect(cards.length).toBe(1);
  });

  it("the Prompts tile carries href='/settings/prompts' + visible label 'Prompts'", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const card = screen.querySelector(
      '[data-testid="settings-card-prompts"]',
    ) as HTMLElement | null;
    expect(card).toBeTruthy();
    expect(card?.getAttribute("href")).toBe("/settings/prompts");
    expect((card?.textContent ?? "").trim()).toContain("Prompts");
  });

  it("the Prompts tile embeds the PromptsIcon (data-testid='prompts-icon') inside the icon container", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    const card = screen.querySelector(
      '[data-testid="settings-card-prompts"]',
    );
    expect(card).toBeTruthy();
    const icon = card?.querySelector('[data-testid="prompts-icon"]');
    expect(icon).toBeTruthy();
  });

  it("exports DocumentHead with title 'Settings — Cachicamas'", () => {
    expect(head.title).toBe("Settings — Cachicamas");
  });

  it("REQ-11 / SCN-11.1 — renders NO <h1> element (headerless page)", async () => {
    const { screen, render } = await createDOM();
    await render(<Index />);
    expect(screen.querySelectorAll("h1").length).toBe(0);
  });
});