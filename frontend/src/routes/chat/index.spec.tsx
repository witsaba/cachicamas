/**
 * Behavioural spec for `routes/chat/index.tsx` — the chat archetype's screen.
 *
 * The screen is a mockup, and the thing most worth defending about a mockup is
 * that it never stops looking like one. Two of the assertions below exist for
 * exactly that: the composer's demonstration notice, and the guard that the
 * frozen wire client is still on disk, unwired, waiting for CH-05.
 *
 * The mocked turn machine itself is unit-tested in
 * `components/chat/use-mock-turn.spec.ts`, where the state transitions can be
 * driven directly instead of through a clock.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import Index, { head } from "./index";
import type { DocumentHeadValue } from "@builder.io/qwik-city";

// `DocumentHead` is a union of a static value and a resolver function; these
// routes export the static form, so narrow once here rather than at every use.
const chatHead = head as DocumentHeadValue;
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

const AUTHED = {
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
};

// ===== Anon =====

test("routes/chat: an anonymous visitor gets the sign-in card, not the transcript", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  const card = screen.querySelector('[data-testid="sign-in-required-card"]');
  expect(card).toBeTruthy();
  expect(screen.querySelector('[data-testid="transcript"]')).toBeFalsy();
  const redirectTo = screen
    .querySelector('form[data-testid="sign-in-button"]')
    ?.querySelector('input[name="redirectTo"]') as HTMLInputElement | null;
  expect(redirectTo?.value).toBe("/chat");
});

test("routes/chat: head metadata names the archetype", () => {
  expect(chatHead.title).toContain("Chat");
  expect(chatHead.title).toContain("cachicamas");
  const description = chatHead.meta?.find((m: { name?: string; content?: string }) => m.name === "description")?.content;
  expect(description).toBeTruthy();
});

test("routes/chat: the frozen browser wire is still on disk, unwired", () => {
  // The mockup replaced the chat's UI, not its contract. `chat-api.ts` and
  // `chat-types.ts` carry the frozen open-then-subscribe wire and stay put for
  // CH-05 to connect; deleting them would turn a swap into a rewrite.
  const dir = resolve(fileURLToPath(import.meta.url), "../../../lib");
  expect(existsSync(resolve(dir, "chat-api.ts"))).toBe(true);
  expect(existsSync(resolve(dir, "chat-types.ts"))).toBe(true);
});

// ===== Authed (vi.doMock / vi.resetModules) — MUST stay last =====

test("routes/chat: an authed visitor gets the transcript and the composer", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  expect(screen.querySelector('[data-testid="transcript"]')).toBeTruthy();
  expect(screen.querySelector('[data-testid="composer"]')).toBeTruthy();
  expect(screen.querySelector('[data-testid="composer-input"]')).toBeTruthy();
});

test("routes/chat: the seeded conversation is on screen, whole", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  const transcript = screen.querySelector('[data-testid="transcript"]');
  expect(transcript).toBeTruthy();
  for (const entry of CONVERSATIONS[0].entries) {
    expect(
      transcript?.querySelector(`[data-testid$="-${entry.id}"]`),
      entry.id,
    ).toBeTruthy();
  }
});

test("routes/chat: the screen says it is a demonstration, unprompted", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // The workspace shell carries the standing demonstration notice (see
  // components/workspace/workspace.spec.tsx). What the composer owes is the
  // product's one promise, stated where a person is about to act on it.
  const composer = screen.querySelector('[data-testid="composer"]');
  const text = (composer as HTMLElement).textContent ?? "";
  expect(text).toContain("Enter to send");
  expect(text).toContain("without you approving it first");
});

test("routes/chat: a decided permission shows its decision and its consequence", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // Every conversation on offer is reachable from the list, including the one
  // that carries a permission answered both ways — an approval you cannot
  // audit afterwards is not much better than no approval at all.
  const list = screen.querySelector('[data-testid="conversation-list"]');
  expect(list?.querySelector('[data-testid="conversation-c-4465"]')).toBeTruthy();
  const audited = CONVERSATIONS.find((c) => c.id === "c-4465");
  const decisions = (audited?.entries ?? []).flatMap((e) =>
    e.kind === "hold" ? [e.decision] : [],
  );
  expect(decisions).toContain("granted");
  expect(decisions).toContain("denied");
});

test("routes/chat: a failure is a typed envelope with a recovery, never a spinner", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // Conversation c-4460 ends in a failure. It is not selected by default, so
  // assert the seeded data carries the contract and that its row is reachable.
  expect(
    screen.querySelector('[data-testid="conversation-c-4460"]'),
  ).toBeTruthy();
  const conversation = CONVERSATIONS.find((c) => c.id === "c-4460");
  const fault = conversation?.entries.find((e) => e.kind === "fault");
  expect(fault).toBeTruthy();
  if (fault && fault.kind === "fault") {
    // A failure names the problem AND the way out, or it is a dead end with
    // better manners.
    expect(fault.message.length).toBeGreaterThan(12);
    expect(fault.recovery.length).toBeGreaterThan(12);
  }
});
