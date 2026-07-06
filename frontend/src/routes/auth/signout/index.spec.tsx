/**
 * Behavioural spec for `routes/auth/signout/index.tsx`.
 *
 * Reference: engram observation `cachicamas/auth/anatomy` (2026-07-06).
 *
 * What we assert (the native signout page contract):
 *   S-AUTH-SIGNOUT-01 — the page wrapper is white bg + slate-900 text
 *     (visual continuity with the rest of cachicamas; the built-in
 *     @auth/core page used a dark theme on prefers-color-scheme: dark
 *     browsers).
 *   S-AUTH-SIGNOUT-02 — the card carries brand + heading "Sign out
 *     of cachicamas?" + description + form + footnote.
 *   S-AUTH-SIGNOUT-03 — the form is a `<form data-testid="auth-signout-form">`
 *     with a hidden `redirectTo` of `/auth/signin` (the post-signout
 *     landing, matching the avatar-dropdown's hidden redirectTo so
 *     both surfaces agree).
 *   S-AUTH-SIGNOUT-04 — the submit button has the label "Sign out".
 *   S-AUTH-SIGNOUT-05 — the Cancel link is a regular `<a href="/">`,
 *     not a button — it must NOT submit the form.
 *   S-AUTH-SIGNOUT-06 — no `<img>`, no `<picture>`, no `<svg>`
 *     anywhere on the page. The sign-out surface carries no brand
 *     mark (the Octocat belongs on sign-IN, not sign-OUT).
 *
 * Mock `routes/plugin@auth.ts` so the route's `useSignOut()` call
 * doesn't require the Qwik City request context that vitest's
 * `createDOM()` harness does not set up. We do NOT mock `useSession()`
 * because the native signout page does not read it.
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect, vi } from "vitest";
import Index from "./index";

vi.mock("~/routes/plugin@auth", () => ({
  useSignOut: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
    isRunning: false,
    formData: undefined,
    value: undefined,
    submitted: false,
    status: undefined,
  }),
  useSignIn: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
  }),
  useSession: () => ({ value: null }),
  onRequest: () => Promise.resolve(),
}));

test("[routes/auth/signout]: page wrapper is white bg + slate-900 text (S-AUTH-SIGNOUT-01)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const main = screen.querySelector('[data-testid="auth-signout-page"]');
  expect(main).toBeTruthy();
  const cls = (main as HTMLElement).className;
  // Locked palette — matches /, /home, /auth/signin, /profile.
  // Regression guard against re-introducing the dark default
  // @auth/core signout page.
  expect(cls).toMatch(/\bbg-white\b/);
  expect(cls).toMatch(/\btext-slate-900\b/);
});

test("[routes/auth/signout]: card carries brand + heading + description + form + footnote (S-AUTH-SIGNOUT-02)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const card = screen.querySelector('[data-testid="auth-signout-card"]');
  expect(card).toBeTruthy();
  expect(
    screen.querySelector('[data-testid="auth-signout-brand"]'),
  ).toBeTruthy();
  const heading = screen.querySelector('[data-testid="auth-signout-heading"]');
  expect(heading).toBeTruthy();
  expect((heading as HTMLElement).textContent?.trim()).toBe(
    "Sign out of cachicamas?",
  );
  expect(
    screen.querySelector('[data-testid="auth-signout-description"]'),
  ).toBeTruthy();
  expect(
    screen.querySelector('[data-testid="auth-signout-form"]'),
  ).toBeTruthy();
  expect(
    screen.querySelector('[data-testid="auth-signout-footnote"]'),
  ).toBeTruthy();
});

test("[routes/auth/signout]: form has hidden redirectTo='/auth/signin' (S-AUTH-SIGNOUT-03)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const form = screen.querySelector('[data-testid="auth-signout-form"]');
  expect(form).toBeTruthy();
  expect(form?.tagName).toBe("FORM");
  const redirectTo = form?.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo).toBeTruthy();
  expect(redirectTo?.value).toBe("/auth/signin");
});

test("[routes/auth/signout]: submit button carries the 'Sign out' label (S-AUTH-SIGNOUT-04)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const button = screen.querySelector('[data-testid="auth-signout-submit"]');
  expect(button).toBeTruthy();
  expect(button?.tagName).toBe("BUTTON");
  expect((button as HTMLElement).textContent?.trim()).toBe("Sign out");
});

test("[routes/auth/signout]: Cancel is an anchor to '/' (S-AUTH-SIGNOUT-05)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const cancel = screen.querySelector(
    '[data-testid="auth-signout-cancel"]',
  ) as HTMLAnchorElement | null;
  expect(cancel).toBeTruthy();
  expect(cancel?.tagName).toBe("A");
  expect(cancel?.getAttribute("href")).toBe("/");
  // The Cancel MUST NOT be type=submit — clicking it must navigate,
  // not post the form. A <a> tag has no type attribute; assert it's
  // either absent or not equal to "submit".
  expect(cancel?.getAttribute("type")).not.toBe("submit");
});

test("[routes/auth/signout]: no <img>, no <picture>, no <svg> (UX-4 / S-AUTH-SIGNOUT-06)", async () => {
  // Sign-out is a deliberate action that does NOT need a brand mark.
  // The Octocat belongs on sign-IN as a recognition cue; on sign-out
  // it would be a stray visual element that violates the text-first
  // constraint. This test pins that absence.
  const { screen, render } = await createDOM();
  await render(<Index />);
  expect(screen.querySelectorAll("img").length).toBe(0);
  expect(screen.querySelectorAll("picture").length).toBe(0);
  expect(screen.querySelectorAll("svg").length).toBe(0);
});
