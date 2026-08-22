/**
 * Behavioural spec for `routes/home/index.tsx` — the desk.
 *
 * The claims worth defending here are all honesty claims. The board must not
 * be able to drift into implying that something works: every archetype cell
 * carries a literal status word, the demo disclosure is present, and the
 * runtime gauges show the real counts rather than round numbers someone typed.
 *
 * `routes/plugin@auth.ts` is mocked so the route's `useSession()` /
 * `useSignIn()` calls do not need the Qwik City request context that
 * `createDOM()` does not set up. The `vi.doMock` tests are LAST in the file,
 * by project convention, so mock-state leakage cannot break the `vi.mock`
 * factory assertions above them.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import Index from "./index";
import { ARCHETYPES, RUNTIME } from "~/lib/mock/registry";

vi.mock("~/routes/plugin@auth", () => ({
  useSession: () => ({ value: null }),
  useSignIn: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
  }),
  useSignOut: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
  }),
  onRequest: () => Promise.resolve(),
}));

const AUTHED = (name: string | null) => ({
  useSession: () => ({ value: { user: { name } } }),
  useSignIn: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signin",
  }),
  useSignOut: () => ({
    submit: $((_fd: FormData) => Promise.resolve()) as QRL<
      (formData: FormData) => unknown
    >,
    actionPath: "/auth/signout",
  }),
  onRequest: () => Promise.resolve(),
});

// ===== Anon (vi.mock factory default) =====

test("[routes/home]: anonymous render shows the sign-in card pointed back at /home", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  expect(
    screen.querySelector('[data-testid="sign-in-required-card"]'),
  ).toBeTruthy();
  const form = screen.querySelector('form[data-testid="sign-in-button"]');
  expect(form).toBeTruthy();
  const redirectTo = form?.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/home");
});

test("[routes/home]: anonymous render never leaks the board", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  expect(screen.querySelector('[data-testid="archetype-register"]')).toBeFalsy();
  expect(screen.querySelector('[data-testid="runtime-panel"]')).toBeFalsy();
});

// ===== Authed (vi.doMock / vi.resetModules) — MUST stay last =====

test("[routes/home]: the board carries one cell per registered archetype", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED("Alice"));
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  for (const a of ARCHETYPES) {
    const cell = screen.querySelector(`[data-testid="register-cell-${a.slug}"]`);
    expect(cell, a.code).toBeTruthy();
  }
  expect(
    screen.querySelectorAll('[data-testid^="register-cell-"]').length,
  ).toBe(ARCHETYPES.length);
});

test("[routes/home]: every archetype states its status in words, not only in colour", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED("Alice"));
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  for (const a of ARCHETYPES) {
    const lamp = screen.querySelector(
      `[data-testid="register-lamp-${a.slug}"]`,
    );
    expect(lamp, a.code).toBeTruthy();
    // The literal word is half the component. A lamp on its own would put the
    // whole state vocabulary behind colour perception.
    expect((lamp as HTMLElement).textContent ?? "").toContain(a.stateWord);
  }
});

test("[routes/home]: the runtime panel shows the real milestone counts", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED("Alice"));
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const panel = screen.querySelector('[data-testid="runtime-panel"]');
  expect(panel).toBeTruthy();
  const text = (panel as HTMLElement).textContent ?? "";
  for (const layer of RUNTIME) {
    expect(text, layer.code).toContain(`${layer.done}/${layer.total}`);
  }
  // Layer 1 and Layer 2 are genuinely finished; Layer 3 genuinely is not.
  expect(text).toContain("42/42");
  expect(text).toContain("24/24");
  expect(text).toContain(`0/${RUNTIME[2].total}`);
});

test("[routes/home]: the board discloses that its activity figures are invented", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED("Alice"));
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const disclosure = screen.querySelector('[data-testid="demo-disclosure"]');
  expect(disclosure).toBeTruthy();
  const text = (disclosure as HTMLElement).textContent ?? "";
  expect(text).toContain("No archetype has ever run");
  expect(text.toLowerCase()).toContain("demo");
});

test("[routes/home]: the screen names the person when their name is known", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED("Alice"));
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const title = screen.querySelector('[data-testid="screen-title"]');
  expect((title as HTMLElement).textContent ?? "").toContain("Alice");
});

test("[routes/home]: a missing name degrades to the impersonal lead, not to 'undefined'", async () => {
  for (const name of ["", null]) {
    vi.resetModules();
    vi.doMock("~/routes/plugin@auth", () => AUTHED(name));
    const { default: AuthedIndex } = await import("./index");
    const { screen, render } = await createDOM();
    await render(<AuthedIndex />);
    const title = screen.querySelector('[data-testid="screen-title"]');
    const text = (title as HTMLElement).textContent ?? "";
    expect(text).toContain("This company's specialists");
    expect(text).not.toContain("undefined");
    expect(text).not.toContain("null");
  }
});
