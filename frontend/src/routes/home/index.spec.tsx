/**
 * Behavioural spec for `routes/home/index.tsx` — the front desk.
 *
 * The claims worth defending here are honesty claims and shape claims: every
 * colleague on the screen carries a literal status word beside its dot, the
 * word "Agent" beside its avatar, and the screen never advertises someone the
 * company has not hired as though they were already working.
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
import { AGENTS, TEAMS } from "~/lib/mock/staff";
import { CONVERSATIONS } from "~/lib/mock/chat";

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
  const card = screen.querySelector('[data-testid="sign-in-required-card"]');
  expect(card).toBeTruthy();
  const redirect = screen.querySelector('input[name="redirectTo"]');
  expect(redirect?.getAttribute("value")).toBe("/home");
});

// ===== Authenticated (vi.doMock + dynamic import) =====

async function renderAuthed(name: string | null) {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED(name));
  const mod = await import("./index");
  const Authed = mod.default;
  const { screen, render } = await createDOM();
  await render(<Authed />);
  return screen;
}

test("[routes/home]: greets the person by their first name", async () => {
  const screen = await renderAuthed("Ana Rivas");
  expect(screen.querySelector("h1")?.textContent).toContain("Ana");
});

test("[routes/home]: falls back to the desk's own name when the session has none", async () => {
  // A greeting that renders "Good to see you, " with nothing after it is worse
  // than no greeting.
  const screen = await renderAuthed(null);
  const h1 = screen.querySelector("h1")?.textContent ?? "";
  expect(h1).toBe("Front desk");
});

test("[routes/home]: shows every colleague on staff, and none that are not", async () => {
  const screen = await renderAuthed("Ana Rivas");
  for (const agent of AGENTS) {
    const card = screen.querySelector(`[data-testid="desk-agent-${agent.slug}"]`);
    if (agent.status === "available") {
      expect(card, agent.slug).toBeFalsy();
    } else {
      expect(card, agent.slug).toBeTruthy();
      // The status word, never the dot alone.
      expect(card?.textContent, agent.slug).toContain(agent.statusWord);
      // And the species, in a word.
      expect(card?.textContent, agent.slug).toContain("Agent");
    }
  }
});

test("[routes/home]: mentions the ones you could hire, below the work and quietly", async () => {
  // A person's own front desk should not read as a store: the hire prompt
  // closes the page as one line, after what they were actually doing.
  const screen = await renderAuthed("Ana Rivas");
  const text = screen.textContent ?? "";
  expect(text).toContain("Your plan also includes");
  expect(text).toContain("Neither has started work");
  for (const agent of AGENTS.filter((a) => a.status === "available")) {
    expect(text, agent.slug).toContain(agent.name);
  }
});

test("[routes/home]: picks up the conversations you actually had", async () => {
  const screen = await renderAuthed("Ana Rivas");
  const text = screen.textContent ?? "";
  for (const c of CONVERSATIONS) {
    expect(text, c.id).toContain(c.title);
  }
});

test("[routes/home]: shows the company's teams", async () => {
  const screen = await renderAuthed("Ana Rivas");
  const text = screen.textContent ?? "";
  for (const team of TEAMS) {
    expect(text, team.slug).toContain(team.name);
  }
});

test("[routes/home]: says nothing about how any of it is built", async () => {
  const screen = await renderAuthed("Ana Rivas");
  expect(screen.textContent ?? "").not.toMatch(
    /\b(archetype|runtime|MCP|schema|Layer [123]|SSE|endpoint|milestone|ADR)\b/i,
  );
});
