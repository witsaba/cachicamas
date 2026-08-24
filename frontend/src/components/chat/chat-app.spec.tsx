/**
 * ChatApp — a conversation with one colleague (CH-05.1 + CH-08).
 *
 * CH-05.1 wired the page to the SSE stream (D-5): `useChatStream`
 * drives submit / cancel / subscribe via `chat-api.ts`. CH-08
 * re-mounts the conversation rail against the wire shape
 * (R-CRI-005, REQ-8); the page's `useVisibleTask$` fires both
 * resume GETs in parallel on first paint (REQ-8), seeds
 * `useChatStream.reset(entries)`, and populates the rail.
 *
 * The D-3 test ("the conversations rail is gone") is now wrong —
 * CH-08 unmounted that drop. The replacement below proves the rail
 * mounts against the wire shape with its `data-testid`. The
 * transcript + composer data-testids persist.
 *
 * Behavioural coverage of the wire client lives in
 * `use-chat-stream.spec.ts`; `chat-app.spec.tsx` defends the page
 * composition only.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ChatApp } from "./chat-app";
import { AGENTS } from "~/lib/mock/staff";

const WHO = {
  youName: "Ana Rivas",
  youEmail: "ana@example.com",
  participantID: "alice",
} as const;

describe("components/chat/chat-app (CH-05.1 + CH-08)", () => {
  it("renders the transcript and the composer", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    expect(screen.querySelector('[data-testid="transcript"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="composer"]')).toBeTruthy();
  });

  it("mounts the conversation rail (R-CRI-005) — CH-08 undoes the CH-05.1 D-3 drop", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    // The rail's container is the data-testid="conversation-list" <ul>.
    // An empty rail is still a mounted rail (S-CRI-004: empty list is
    // success, not a crash).
    expect(screen.querySelector('[data-testid="conversation-list"]')).toBeTruthy();
  });

  it("does not render the retired conversations panel (CH-05.1 D-3 retired surfaces)", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    // CH-05.1 retired the panel + disclosure + history-toggle testids;
    // CH-08 only re-mounts the rail itself, never re-introduces the
    // panel/disclosure/toggle chrome. Verify those stay gone.
    expect(screen.querySelector('[data-testid="conversations-panel"]')).toBeFalsy();
    expect(screen.querySelector('[data-testid="conversations-disclosure"]')).toBeFalsy();
    expect(screen.querySelector('[data-testid="history-toggle"]')).toBeFalsy();
  });

  it("names the colleague you are talking to, and calls them an agent", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const header = screen.querySelector("header");
    const firstAgent = AGENTS[0];
    expect(header?.textContent).toContain(firstAgent.name);
    expect(header?.textContent).toContain("Agent");
  });

  it("falls back to the first agent from staff.ts when no ?with= is set (D-6)", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const header = screen.querySelector("header");
    expect(header?.textContent).toContain(AGENTS[0].name);
  });

  it("announces the conversation politely, never assertively", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const transcript = screen.querySelector('[data-testid="transcript"]');
    expect(transcript?.getAttribute("aria-live")).toBe("polite");
    expect(transcript?.getAttribute("aria-label")).toContain("Conversation with");
  });

  it("says nothing about how any of it is built", async () => {
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
