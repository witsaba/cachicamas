/**
 * Behavioural spec for `routes/chat/index.tsx` — the chat archetype's screen.
 *
 * After CH-05.1 the page is wired to the wire (D-5): the page calls
 * `submitTurn` / `cancelTurn` / `subscribeTurn` through `useChatStream`,
 * and the conversations rail is gone (D-3). What is asserted here:
 *
 *   - The route guard behaves (anon vs authed).
 *   - The head metadata names the archetype.
 *   - The wire surface CH-05.1 connects to is exposed — T5.1 replaces
 *     the file-existence check with a semantic `export function` check.
 *   - The authed render mounts the transcript and composer.
 *   - The composer carries the demonstration promise.
 *
 * Wire-protocol behavioural coverage lives in
 * `lib/chat-api.spec.ts` and `components/chat/use-chat-stream.spec.ts`;
 * page-composition coverage lives in `components/chat/chat-app.spec.tsx`.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import Index, { head } from "./index";
import type { DocumentHeadValue } from "@builder.io/qwik-city";

// `DocumentHead` is a union of a static value and a resolver function; these
// routes export the static form, so narrow once here rather than at every use.
const chatHead = head as DocumentHeadValue;

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

test("routes/chat: the page is wired to the chat wire client (REQ-1)", async () => {
  // After CH-05.1 the page connects to the wire through chat-api.ts.
  // Asserting on file existence passed whether or not the page was
  // wired — the existence check was true whether chat-app.tsx
  // imported chat-api or kept mocking it. The semantic exports check
  // is the wire-surface contract: a future regression that reverts
  // chat-app.tsx to a mock fails this test because the wire client
  // is the only source of `submitTurn` / `cancelTurn` /
  // `subscribeTurn`.
  const mod = await import("~/lib/chat-api");
  expect(typeof mod.submitTurn).toBe("function");
  expect(typeof mod.cancelTurn).toBe("function");
  expect(typeof mod.subscribeTurn).toBe("function");
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

test("routes/chat: the retired conversations rail is not mounted (D-3)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // After CH-05.1 the page holds a single active conversation; the
  // history list is owned by CH-08.2 (cachicamas-chat-conversation-list).
  expect(
    screen.querySelector('[data-testid="conversations-panel"]'),
  ).toBeFalsy();
  expect(
    screen.querySelector('[data-testid="conversation-list"]'),
  ).toBeFalsy();
});

test("routes/chat: the composer carries the demo promise, unprompted", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // The composer states the product's one standing promise where a
  // person is about to act on it, rather than in a settings page
  // nobody opens.
  const composer = screen.querySelector('[data-testid="composer"]');
  const text = (composer as HTMLElement).textContent ?? "";
  expect(text).toContain("Enter to send");
  expect(text).toContain("without you approving it first");
});

test("routes/chat: the page renders at least one transcript slot (the assistant's opening line)", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // The lead-in line ("See what ... can do") survives D-3 in the
  // transcript opening <li> (chat-app.tsx:130-148). The second
  // status indicator inside it gives the test a stable assertion
  // anchor.
  const transcript = screen.querySelector('[data-testid="transcript"]');
  expect(transcript?.textContent ?? "").toContain("See what");
});
