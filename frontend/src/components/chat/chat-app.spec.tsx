/**
 * ChatApp — a conversation with one colleague.
 *
 * The turn machine is unit-tested in `use-mock-turn.spec.ts`, where it can be
 * driven a tick at a time. What is asserted here is composition: who you are
 * talking to, the seeded conversation, how the history survives a phone, and
 * the announcement of arriving output to assistive technology.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ChatApp } from "./chat-app";
import { CONVERSATIONS } from "~/lib/mock/chat";
import { agentBySlug } from "~/lib/mock/staff";

const WHO = { youName: "Ana Rivas", youEmail: "ana@example.com" } as const;

describe("components/chat/chat-app", () => {
  it("renders the history, the conversation and the composer", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    expect(
      screen.querySelector('[data-testid="conversations-panel"]'),
    ).toBeTruthy();
    expect(screen.querySelector('[data-testid="transcript"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="composer"]')).toBeTruthy();
  });

  it("opens on the most recent conversation, whole", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const transcript = screen.querySelector('[data-testid="transcript"]');
    for (const entry of CONVERSATIONS[0].entries) {
      expect(
        transcript?.querySelector(`[data-testid$="-${entry.id}"]`),
        entry.id,
      ).toBeTruthy();
    }
  });

  it("names the colleague you are talking to, and calls them an agent", async () => {
    // The shape rule (a person is a circle, an agent is a rounded square) is
    // never the only carrier: the word is always beside it.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const agent = agentBySlug(CONVERSATIONS[0].agentSlug);
    const header = screen.querySelector("header");
    expect(header?.textContent).toContain(agent?.name);
    expect(header?.textContent).toContain("Agent");
  });

  it("keeps multi-conversation history reachable on a phone", async () => {
    // The standing panel is hidden below `lg`. Hiding it with nothing in its
    // place would take a real capability off phones, which is a capability
    // disappearing rather than a layout adapting — so a labelled control
    // brings it back.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const toggle = screen.querySelector('[data-testid="history-toggle"]');
    expect(toggle).toBeTruthy();
    expect(toggle?.className ?? "").toContain("lg:hidden");
    expect(toggle?.textContent).toContain("History");
    expect(
      screen.querySelector('[data-testid="conversations-panel"]')?.className ??
        "",
    ).toContain("lg:flex");
  });

  it("lists every conversation, so no route to one can silently drift", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const list = screen.querySelector('[data-testid="conversation-list"]');
    for (const c of CONVERSATIONS) {
      expect(
        list?.querySelector(`[data-testid="conversation-${c.id}"]`),
        c.id,
      ).toBeTruthy();
    }
  });

  it("announces the conversation politely, never assertively", async () => {
    // Output that interrupts a screen reader mid-sentence is worse than output
    // it has to be asked for.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    const transcript = screen.querySelector('[data-testid="transcript"]');
    expect(transcript?.getAttribute("aria-live")).toBe("polite");
    expect(transcript?.getAttribute("aria-label")).toContain("Conversation with");
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
});
