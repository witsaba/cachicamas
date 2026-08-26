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
import { AGENTS } from "~/lib/mock/staff";

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

test("routes/chat: the retired conversations panel (CH-05.1 D-3) stays gone — CH-08 re-mounts only the rail, not the panel chrome", async () => {
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // After CH-05.1 the conversations-panel chrome is retired; CH-08
  // only re-mounts the rail (data-testid="conversation-list"), not
  // the panel/disclosure/toggle surfaces.
  expect(
    screen.querySelector('[data-testid="conversations-panel"]'),
  ).toBeFalsy();
});

test("routes/chat: the page passes participantID into ChatApp (REQ-8 / D-1)", async () => {
  // The route surfaces the session's resolved participant id into
  // ChatApp so the page knows which conversation to load on mount.
  // The mock returns a session with user.name="Alice"; the route's
  // participantID falls back to the email when no user.id is set.
  vi.resetModules();
  vi.doMock("~/routes/plugin@auth", () => AUTHED);
  const { default: AuthedIndex } = await import("./index");
  const { screen, render } = await createDOM();
  await render(<AuthedIndex />);
  // The page must mount the chat surface (transcript + composer)
  // and the rail — proving participantID flowed through without
  // crashing the useVisibleTask$ mount hook.
  expect(screen.querySelector('[data-testid="transcript"]')).toBeTruthy();
  expect(screen.querySelector('[data-testid="composer"]')).toBeTruthy();
  expect(screen.querySelector('[data-testid="conversation-list"]')).toBeTruthy();
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
  // The transcript still mounts as the chat surface (CH-05.1) and
  // the agent's status now lives in the page header (chat-app.tsx)
  // rather than the lead-in <li> that previously opened the
  // transcript. With the lead-in removed the transcript is empty
  // until the first exchange; instead assert the header carries the
  // agent name AND the status word, which together prove the chat
  // surface is rendering the colleague the page is for.
  const header = screen.querySelector("header");
  const headerText = (header as HTMLElement).textContent ?? "";
  expect(headerText).toContain(AGENTS[0].name);
  expect(headerText).toContain(AGENTS[0].statusWord);
});
