/**
 * Root layout.
 *
 * The layout used to be the app shell. It is not any more: the marketing site
 * owns its own header and footer, and every workspace screen sits inside
 * `<Workspace>` via its section's layout (see
 * `components/workspace/workspace.spec.tsx`, which carries the identity
 * assertions that used to live here).
 *
 * What is left is the part that belongs to the document rather than to any
 * surface, and each of these is a regression guard for a real defect:
 *
 *   - the skip link is the FIRST focusable element (S-AS-050);
 *   - the layout takes no router context, because `useLocation()` reads Qwik
 *     City's `qc-l`, which only exists inside a request handler;
 *   - back/forward re-validates the session instead of serving Qwik's cached
 *     render of a signed-in page to someone who has signed out (UAT-8 r4).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import Layout from "./layout";

const layoutPath = resolve(
  new URL(".", import.meta.url).pathname,
  "layout.tsx",
);

async function renderLayout() {
  const { screen, render } = await createDOM();
  await render(
    <Layout>
      <main id="main" data-testid="child">
        child content
      </main>
    </Layout>,
  );
  return screen;
}

test("[routes/layout]: exists and exports a default component$", () => {
  expect(existsSync(layoutPath)).toBe(true);
  const source = readFileSync(layoutPath, "utf8");
  expect(source).toMatch(/export default component\$/);
});

test("[routes/layout]: renders its child untouched", async () => {
  const screen = await renderLayout();
  const child = screen.querySelector('[data-testid="child"]');
  expect(child).toBeTruthy();
  expect(child?.textContent).toContain("child content");
});

test("[routes/layout]: the skip link is the first focusable element (S-AS-050)", async () => {
  const screen = await renderLayout();
  const focusables = screen.querySelectorAll(
    'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])',
  );
  expect(focusables.length).toBeGreaterThan(0);
  const first = focusables[0] as HTMLElement;
  expect(first.getAttribute("data-testid")).toBe("skip-to-main");
  expect(first.getAttribute("href")).toBe("#main");
});

test("[routes/layout]: the skip link is hidden until it is focused", async () => {
  const screen = await renderLayout();
  const link = screen.querySelector(
    '[data-testid="skip-to-main"]',
  ) as HTMLElement | null;
  const cls = link?.className ?? "";
  expect(cls).toContain("sr-only");
  expect(cls).toContain("focus:not-sr-only");
});

test("[routes/layout]: renders no chrome of its own", () => {
  // A header here would be a second header on every marketing page and a
  // stray one above every workspace rail. The shell moved; this guards the
  // move.
  const source = readFileSync(layoutPath, "utf8");
  expect(source).not.toContain("<header");
  expect(source).not.toContain("AvatarDropdown");
  expect(source).not.toContain("SignInButton");
});

test("[routes/layout]: asks for no router context (createDOM has no qc-l)", () => {
  const source = readFileSync(layoutPath, "utf8");
  expect(source).not.toMatch(/^import .*qwik-city/m);
});

test("[routes/layout]: re-validates the session on back/forward (UAT-8 r4)", () => {
  // Without this, Qwik City's SPA router serves the CACHED render from browser
  // history when someone navigates back to a page they saw while signed in —
  // their name and avatar still on screen after signing out.
  const source = readFileSync(layoutPath, "utf8");
  expect(source).toContain("popstate");
  expect(source).toContain("capture: true");
  expect(source).toContain("stopImmediatePropagation");
  expect(source).toContain("window.location.reload()");
});
