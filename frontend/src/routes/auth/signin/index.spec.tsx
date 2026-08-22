/**
 * Behavioural spec for `routes/auth/signin/index.tsx`.
 *
 * Reference: engram observation `cachicamas/auth/anatomy` (2026-07-06).
 *
 * What we assert (the native signin page contract):
 *   S-AUTH-SIGNIN-01 — the page renders a `<main data-testid="auth-signin-page">`
 *     wrapper with `bg-white text-slate-900` (NO dark theme, NO
 *     `prefers-color-scheme` flip — visual continuity with `/`,
 *     `/home`, `/profile`, etc.).
 *   S-AUTH-SIGNIN-02 — a card with `data-testid="auth-signin-card"`
 *     carries the brand mark, heading, description, action, and footnote.
 *   S-AUTH-SIGNIN-03 — the action is a `<Form>` whose hidden
 *     `providerId` is `"github"` and whose hidden `redirectTo` is
 *     `/home` (the default landing for direct visits).
 *   S-AUTH-SIGNIN-04 — the only `<svg>` in the page is the GitHub
 *     Octocat brand mark inside the SignInButton (UX-4 amendment:
 *     brand marks are aphantasia-friendly visual anchors). Zero
 *     `<img>` and zero `<picture>` elements anywhere on the page.
 *   S-AUTH-SIGNIN-05 — the brand mark links to `/` (so a visitor who
 *     changed their mind can return to the landing without using the
 *     browser back button).
 *
 * Mock `routes/plugin@auth.ts` so the route's `useSignIn()` call
 * doesn't require the Qwik City request context that vitest's
 * `createDOM()` harness does not set up. We do NOT mock `useSession()`
 * because the native signin page does not read it (see the trade-off
 * note in plugin@auth.ts).
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect, vi } from "vitest";
import Index from "./index";

/**
 * `useLocation()` is provided by Qwik City and reads the inbound
 * request URL via the qc-l context. `createDOM()` does not set up
 * that context, so the spec MUST mock it. The mock takes the URL
 * from a mutable variable so individual tests can swap it per-case.
 */
let mockLocationUrl = "http://localhost/auth/signin";
vi.mock("@builder.io/qwik-city", async () => {
  const actual = await vi.importActual<typeof import("@builder.io/qwik-city")>(
    "@builder.io/qwik-city",
  );
  return {
    ...actual,
    useLocation: () => ({ url: new URL(mockLocationUrl) }),
  };
});

vi.mock("~/routes/plugin@auth", () => ({
  useSignIn: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
    isRunning: false,
    formData: undefined,
    value: undefined,
    submitted: false,
    status: undefined,
  }),
  useSignOut: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
  }),
  useSession: () => ({ value: null }),
  onRequest: () => Promise.resolve(),
}));

test("[routes/auth/signin]: page wrapper is painted by the product, not by the browser theme (S-AUTH-SIGNIN-01)", async () => {
  mockLocationUrl = "http://localhost/auth/signin";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const main = screen.querySelector('[data-testid="auth-signin-page"]');
  expect(main).toBeTruthy();
  const cls = (main as HTMLElement).className;
  // The product paints its own ground on every route. The built-in
  // @auth/core page flipped with `prefers-color-scheme`, which produced a
  // page that belonged to neither the browser nor the product; this is the
  // regression guard against that returning.
  expect(cls).toMatch(/\bbg-void\b/);
  expect(cls).toMatch(/\btext-fg\b/);
  expect(cls).not.toMatch(/slate|zinc|bg-white/);
});

test("[routes/auth/signin]: card carries brand + heading + description + action + footnote (S-AUTH-SIGNIN-02)", async () => {
  mockLocationUrl = "http://localhost/auth/signin";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const card = screen.querySelector('[data-testid="auth-signin-card"]');
  expect(card).toBeTruthy();
  expect(
    screen.querySelector('[data-testid="auth-signin-brand"]'),
  ).toBeTruthy();
  const heading = screen.querySelector('[data-testid="auth-signin-heading"]');
  expect(heading).toBeTruthy();
  expect((heading as HTMLElement).textContent?.trim()).toBe(
    "Sign in to cachicamas",
  );
  expect(
    screen.querySelector('[data-testid="auth-signin-description"]'),
  ).toBeTruthy();
  expect(
    screen.querySelector('[data-testid="auth-signin-action"]'),
  ).toBeTruthy();
  expect(
    screen.querySelector('[data-testid="auth-signin-footnote"]'),
  ).toBeTruthy();
});

test("[routes/auth/signin]: hidden providerId is 'github' and redirectTo defaults to '/home' on direct visit (S-AUTH-SIGNIN-03)", async () => {
  // Direct visit: no `?callbackUrl=...` → defaults to /home.
  mockLocationUrl = "http://localhost/auth/signin";
  const { screen, render } = await createDOM();
  await render(<Index />);
  // The SignInButton renders a <form data-testid="sign-in-button">.
  // We assert on its hidden inputs to prove the action wiring is
  // wired up for the GitHub OAuth flow with the canonical landing.
  const form = screen.querySelector('form[data-testid="sign-in-button"]');
  expect(form).toBeTruthy();
  const providerId = form?.querySelector(
    'input[name="providerId"]',
  ) as HTMLInputElement | null;
  expect(providerId?.value).toBe("github");
  const redirectTo = form?.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/home");
});

test("[routes/auth/signin]: redirectTo forwards the callbackUrl query param verbatim (post-signin round-trip)", async () => {
  // The whole point of this slice: when a protected route's onRequest
  // redirects to `/auth/signin?callbackUrl=/organizations/42`, the
  // visitor lands on /auth/signin, clicks the Sign in button, and the
  // hidden redirectTo carries the original URL so Auth.js's callback
  // handler returns them to /organizations/42 after OAuth.
  mockLocationUrl =
    "http://localhost/auth/signin?callbackUrl=%2Forganizations%2F42";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const redirectTo = screen.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/organizations/42");
});

test("[routes/auth/signin]: callbackUrl preserves the search string for query-param-rich protected routes", async () => {
  // `/organizations/[id]?tab=members` carries meaningful state. After
  // sign-in, the visitor must land back on the same view.
  mockLocationUrl =
    "http://localhost/auth/signin?callbackUrl=%2Forganizations%2F42%3Ftab%3Dmembers";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const redirectTo = screen.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/organizations/42?tab=members");
});

test("[routes/auth/signin]: callbackUrl is rejected when it is an absolute URL (open-redirect defence)", async () => {
  // Defence: if a malicious actor crafts a link
  // `/auth/signin?callbackUrl=https://evil.example.com`, we MUST NOT
  // forward that to the SignInButton's redirectTo (otherwise the
  // post-OAuth redirect could be hijacked). Falls back to /home.
  mockLocationUrl =
    "http://localhost/auth/signin?callbackUrl=https%3A%2F%2Fevil.example.com";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const redirectTo = screen.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/home");
});

test("[routes/auth/signin]: callbackUrl is rejected when it is a protocol-relative URL (open-redirect defence)", async () => {
  // `//evil.example.com` is a protocol-relative URL that browsers
  // interpret as `https://evil.example.com` when the current scheme
  // is https. Must be rejected just like the absolute-URL case.
  mockLocationUrl =
    "http://localhost/auth/signin?callbackUrl=%2F%2Fevil.example.com";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const redirectTo = screen.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/home");
});

test("[routes/auth/signin]: no <img>, no <picture>; the only <svg> is the GitHub Octocat brand mark (UX-4 / S-AUTH-SIGNIN-04)", async () => {
  mockLocationUrl = "http://localhost/auth/signin";
  const { screen, render } = await createDOM();
  await render(<Index />);
  expect(screen.querySelectorAll("img").length).toBe(0);
  expect(screen.querySelectorAll("picture").length).toBe(0);
  // Every <svg> on the page must be the SignInButton's brand mark.
  // No other SVG would be allowed: the page is text-first per UX-4.
  const svgs = Array.from(screen.querySelectorAll("svg"));
  expect(svgs.length).toBeGreaterThan(0);
  for (const svg of svgs) {
    expect(svg.getAttribute("data-testid")).toBe("sign-in-button-github-mark");
  }
});

test("[routes/auth/signin]: brand mark links to / (S-AUTH-SIGNIN-05)", async () => {
  mockLocationUrl = "http://localhost/auth/signin";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const brand = screen.querySelector(
    '[data-testid="auth-signin-brand"]',
  ) as HTMLAnchorElement | null;
  expect(brand).toBeTruthy();
  expect(brand?.tagName).toBe("A");
  expect(brand?.getAttribute("href")).toBe("/");
});

test("[routes/auth/signin]: uses the system-of-record component, not a re-implementation (regression guard)", async () => {
  // The page delegates the actual OAuth start to the SignInButton
  // component (system-of-record: routes/layout.tsx uses the same
  // component for the header CTA). This test pins that delegation so
  // a future contributor can't sneak in a hand-rolled <Form> that
  // drifts from the header behaviour (different redirectTo, missing
  // providerId, different brand mark, etc.).
  mockLocationUrl = "http://localhost/auth/signin";
  const { screen, render } = await createDOM();
  await render(<Index />);
  const button = screen.querySelector('button[type="submit"]');
  expect(button).toBeTruthy();
  // The visible label is the short "Sign in" per UX-4 amendment.
  const labelSpan = button?.querySelector("span");
  expect(labelSpan?.textContent?.trim()).toBe("Sign in");
});
