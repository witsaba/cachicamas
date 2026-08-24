/**
 * ChatApp — a conversation with one colleague (CH-05.1).
 *
 * After CH-05.1 the page is wired to the wire (D-5). The
 * conversations rail is gone (D-3): the active conversation is
 * single, the entry list is owned by `useChatStream` against the
 * real SSE stream, and the testid rail (`conversations-panel`,
 * `history-toggle`, `conversation-list`, `conversation-c-4460`,
 * `c-4465`) is retired. What remains: the transcript +
 * composer data-testids, the chat window announces output to
 * assistive technology via `aria-live="polite"`, and the active
 * agent is the first in `staff.ts` by default (overridable via
 * `?with=<slug>`).
 *
 * Behavioural coverage of the wire client lives in
 * `use-chat-stream.spec.ts`; `chat-app.spec.tsx` defends the page
 * composition only.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ChatApp } from "./chat-app";
import { AGENTS } from "~/lib/mock/staff";

const WHO = { youName: "Ana Rivas", youEmail: "ana@example.com" } as const;

describe("components/chat/chat-app", () => {
  it("renders the transcript and the composer", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    expect(screen.querySelector('[data-testid="transcript"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="composer"]')).toBeTruthy();
  });

  it("does not render the retired conversations rail (D-3)", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    expect(
      screen.querySelector('[data-testid="conversations-panel"]'),
    ).toBeFalsy();
    expect(
      screen.querySelector('[data-testid="conversations-disclosure"]'),
    ).toBeFalsy();
    expect(screen.querySelector('[data-testid="history-toggle"]')).toBeFalsy();
    expect(screen.querySelector('[data-testid="conversation-list"]')).toBeFalsy();
  });

  it("names the colleague you are talking to, and calls them an agent", async () => {
    // The shape rule (a person is a circle, an agent is a rounded square) is
    // never the only carrier: the word is always beside it.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const header = screen.querySelector("header");
    const firstAgent = AGENTS[0];
    expect(header?.textContent).toContain(firstAgent.name);
    expect(header?.textContent).toContain("Agent");
  });

  it("falls back to the first agent from staff.ts when no ?with= is set (D-6)", async () => {
    // The deep-link useVisibleTask$ reads window.location.href in a browser;
    // under createDOM there is no window, so activeSlug stays at AGENTS[0]
    // (the default). Verifies the default-vs-deep-link contract.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const header = screen.querySelector("header");
    expect(header?.textContent).toContain(AGENTS[0].name);
  });

  it("announces the conversation politely, never assertively", async () => {
    // Output that interrupts a screen reader mid-sentence is worse than output
    // it has to be asked for.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const transcript = screen.querySelector('[data-testid="transcript"]');
    expect(transcript?.getAttribute("aria-live")).toBe("polite");
    expect(transcript?.getAttribute("aria-label")).toContain(
      "Conversation with",
    );
  });

  it("says nothing about how any of it is built", async () => {
    // The people using this hire colleagues; they do not run a runtime.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const text = screen.textContent ?? "";
    expect(text).not.toMatch(
      /\b(archetype|runtime|MCP|schema|SSE|token|endpoint|milestone|doc 000)\b/i,
    );
  });

  it("the composer is interactable — idle status binds the input (D-5)", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const input = screen.querySelector(
      '[data-testid="composer-input"]',
    ) as HTMLTextAreaElement;
    expect(input).toBeTruthy();
    expect(input.hasAttribute("disabled")).toBe(false);
  });
});
