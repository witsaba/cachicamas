/**
 * Behavioural spec for `components/marketing/header/header.tsx`.
 *
 * The marketing chrome is auth-state-agnostic. The header renders the same
 * wordmark, the same three in-page destinations and the same secondary CTA
 * for every visitor — no sign-in affordance, no `/auth/` link, no
 * `signIn`/`authenticated` prop on the component. The assertions below
 * pin that contract (FRMO-1, FRMO-4).
 */
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { MarketingHeader } from "./header";

const sourcePath = fileURLToPath(new URL("./header.tsx", import.meta.url));
const source = readFileSync(sourcePath, "utf8");

async function renderHeader() {
  const { screen, render } = await createDOM();
  // Marketing-only chrome: the component takes no props.
  await render(<MarketingHeader />);
  return screen;
}

test("[marketing/header]: renders with no props and no TypeScript error", () => {
  // Source-level guard: the MarketingHeaderProps surface is empty (the
  // interface is removed entirely from the trimmed tree).
  expect(source).not.toMatch(/MarketingHeaderProps\s*\{[^}]*signIn/s);
  expect(source).not.toMatch(/MarketingHeaderProps\s*\{[^}]*authenticated/s);
});

test("[marketing/header]: renders the wordmark and the three nav destinations", async () => {
  const screen = await renderHeader();
  const nav = screen.querySelector('nav[aria-label="Site"]');
  expect(nav).toBeTruthy();
  const links = nav?.querySelectorAll("a") ?? [];
  const hrefs = Array.from(links).map((a) => a.getAttribute("href") ?? "");
  expect(hrefs).toContain("#staff");
  expect(hrefs).toContain("#how");
  expect(hrefs).toContain("#plans");

  const wordmark = screen.querySelector('a[href="/"]');
  expect(wordmark?.textContent?.trim()).toBe("cachicamas");
});

test("[marketing/header]: exposes no sign-in affordance of any shape (FRMO-1.a)", async () => {
  const screen = await renderHeader();
  const text = screen.textContent ?? "";
  expect(text).not.toMatch(/sign in with github/i);
  expect(text).not.toMatch(/open your workspace/i);
  expect(text).not.toMatch(/^sign in$/i);
});

test("[marketing/header]: exposes no link to an /auth/ path (FRMO-1.a)", async () => {
  const screen = await renderHeader();
  const authLinks = Array.from(
    screen.querySelectorAll('a[href^="/auth/"]'),
  );
  expect(authLinks).toHaveLength(0);
});

test("[marketing/header]: source carries no signIn / authenticated coupling (FRMO-1.a)", () => {
  // Strip block comments to avoid false positives (comments can mention the
  // deleted identifiers without the production code using them).
  const stripped = source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/^\s*\/\/.*$/gm, "");
  expect(stripped).not.toMatch(/\bsignIn\b/);
  expect(stripped).not.toMatch(/\bauthenticated\b/);
});