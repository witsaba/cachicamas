/**
 * Behavioural spec for `routes/home/index.tsx`.
 *
 * Reference: `openspec/changes/home-page-placeholder/specs/home-page/spec.md`
 *   R-HP-001 (S-HP-001..S-HP-004) — personalised greeting for authed users.
 *   R-HP-002 (S-HP-010..S-HP-012) — single paragraph placeholder, no imagery.
 *   R-HP-003 (S-HP-020..S-HP-023) — anonymous renders SignInRequiredCard.
 *   R-HP-008 (S-HP-070) — behavioural coverage of the home route.
 *
 * Mock `routes/plugin@auth.ts` so the route's `useSession()` /
 * `useSignIn()` calls don't require the Qwik City request context that
 * vitest's `createDOM()` harness does not set up.
 *
 * No `vi.doMock` / `vi.resetModules` tests in this file any more at the
 * top of the file — they are placed LAST in the file because the
 * project convention (see `routes/index.spec.tsx` header) is to keep
 * `vi.doMock` tests at the end so mock-state leakage cannot break
 * earlier `vi.mock` factory assertions. If a future change adds more
 * `vi.doMock` tests, append them after the existing block.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import Index from "./index";

vi.mock("~/routes/plugin@auth", () => {
  return {
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
  };
});

// ===== Anon tests (vi.mock factory default) =====

test("[routes/home]: anonymous render shows SignInRequiredCard with redirectTo='/home' (R-HP-003 / S-HP-020, S-HP-022)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const card = screen.querySelector('[data-testid="sign-in-required-card"]');
  expect(card).toBeTruthy();
  const form = screen.querySelector('form[data-testid="sign-in-button"]');
  expect(form).toBeTruthy();
  const redirectTo = form?.querySelector(
    'input[name="redirectTo"]',
  ) as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/home");
  const providerId = form?.querySelector(
    'input[name="providerId"]',
  ) as HTMLInputElement | null;
  expect(providerId?.value).toBe("github");
});

test("[routes/home]: anonymous card description references 'home' (R-HP-003 / S-HP-021)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const card = screen.querySelector('[data-testid="sign-in-required-card"]');
  const text = (card as HTMLElement | null)?.textContent ?? "";
  expect(text).toContain("home");
  expect(text).not.toContain("profile");
  expect(text).not.toContain("organizations");
});

test("[routes/home]: anonymous render has no <img>/<picture>/<svg> (UX-4 / S-HP-012)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  expect(screen.querySelectorAll("img").length).toBe(0);
  expect(screen.querySelectorAll("picture").length).toBe(0);
  // The SignInButton's GitHub Octocat SVG is allowed per the UX-4
  // amendment (recognizable brand marks). We assert that NO other SVG
  // exists on the anonymous render — every <svg> that DOES exist
  // must be the GitHub brand mark.
  const svgNodes = Array.from(screen.querySelectorAll("svg"));
  for (const svg of svgNodes) {
    expect(svg.getAttribute("data-testid")).toBe("sign-in-button-github-mark");
  }
});

// ===== Authed tests (vi.doMock / vi.resetModules) =====
// IMPORTANT: these MUST remain LAST in this file. vi.doMock mutates the
// module registry; subsequent `vi.mock` tests would see the override.

test("[routes/home]: authed render greets 'Alice' by name (R-HP-001 / S-HP-001)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: { user: { name: "Alice" } } }),
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
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const heading = screen.querySelector('[data-testid="home-heading"]');
  expect(heading).toBeTruthy();
  expect((heading as HTMLElement).textContent).toBe("Welcome, Alice");
});

test("[routes/home]: authed render falls back to 'Welcome' when name is empty string (R-HP-001 / S-HP-002)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: { user: { name: "" } } }),
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
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const heading = screen.querySelector('[data-testid="home-heading"]');
  expect((heading as HTMLElement).textContent).toBe("Welcome");
  // No trailing comma from a missing name token.
  expect((heading as HTMLElement).textContent).not.toContain(", ");
});

test("[routes/home]: authed render falls back to 'Welcome' when name is null (R-HP-001 / S-HP-003)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: { user: { name: null } } }),
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
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const heading = screen.querySelector('[data-testid="home-heading"]');
  expect((heading as HTMLElement).textContent).toBe("Welcome");
});

test("[routes/home]: authed render preserves unicode names verbatim (R-HP-001 / S-HP-004)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: { user: { name: "María José" } } }),
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
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const heading = screen.querySelector('[data-testid="home-heading"]');
  expect((heading as HTMLElement).textContent).toBe("Welcome, María José");
});

test("[routes/home]: authed render contains exactly one <p> placeholder (R-HP-002 / S-HP-010, S-HP-011)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: { user: { name: "Alice" } } }),
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
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const paragraphs = screen.querySelectorAll("p");
  expect(paragraphs.length).toBe(1);
  const p = paragraphs[0];
  expect(p.getAttribute("data-testid")).toBe("home-paragraph");
  const text = (p.textContent ?? "").trim();
  expect(text.length).toBeGreaterThan(0);
  expect(text.length).toBeLessThanOrEqual(200);
});

test("[routes/home]: authed render has no <img>/<picture> and no <svg> (R-HP-002 / S-HP-012, UX-4)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => ({
    useSession: () => ({ value: { user: { name: "Alice" } } }),
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
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  expect(screen.querySelectorAll("img").length).toBe(0);
  expect(screen.querySelectorAll("picture").length).toBe(0);
  expect(screen.querySelectorAll("svg").length).toBe(0);
});
