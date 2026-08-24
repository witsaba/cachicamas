/**
 * ConversationList — the archetype's own memory.
 *
 * CH-08.2 (R-CRI-005): the rail is wire-driven. This spec passes
 * wire-shaped ConversationSummary values, NOT the legacy mock
 * Conversation type from lib/mock/chat.ts. The mock still serves
 * the demonstration surfaces (`hero-proof`, `front-desk`); this
 * component's typed prop surface is the wire (R-CRI-004).
 */
import { $ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ConversationList } from "./conversation-list";
import type { ConversationSummary } from "~/lib/chat-types";

const noop = $((_id: string) => undefined);

const wireSummaries: ConversationSummary[] = [
  {
    conversationID: "c-4471",
    lastActivityAt: "2026-08-24T17:00:00Z",
    turnCount: 4,
  },
  {
    conversationID: "c-4468",
    lastActivityAt: "2026-08-23T17:00:00Z",
    turnCount: 3,
  },
];

describe("components/chat/conversation-list (R-CRI-005 wire shape)", () => {
  it("renders one row per conversation summary", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={wireSummaries}
        selectedId={wireSummaries[0].conversationID}
        onSelect$={noop}
      />,
    );
    for (const c of wireSummaries) {
      expect(
        screen.querySelector(`[data-testid="conversation-${c.conversationID}"]`),
        c.conversationID,
      ).toBeTruthy();
    }
  });

  it("renders turnCount + relative-time age derived from lastActivityAt", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={wireSummaries}
        selectedId={wireSummaries[0].conversationID}
        onSelect$={noop}
      />,
    );
    const first = wireSummaries[0];
    const row = screen.querySelector(
      `[data-testid="conversation-${first.conversationID}"]`,
    );
    const text = row?.textContent ?? "";
    // turnCount surfaces in the row's meta line ("4 turns · 12 minutes ago")
    expect(text).toContain("4 turns");
    // The id surfaces as the row title
    expect(text).toContain(first.conversationID);
    // The relative-time age is derived from the wire's ISO timestamp;
    // the helper returns "Just now" / "X minutes ago" / etc.
    expect(text).toMatch(/Just now|minute|hour|Yesterday|days|ago/);
  });

  it("marks exactly one row current, and announces it", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={wireSummaries}
        selectedId={wireSummaries[1].conversationID}
        onSelect$={noop}
      />,
    );
    const current = screen.querySelectorAll('[aria-current="true"]');
    expect(current.length).toBe(1);
    expect(current[0].getAttribute("data-testid")).toBe(
      `conversation-${wireSummaries[1].conversationID}`,
    );
  });

  it("uses real buttons, so the list is keyboard reachable", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={wireSummaries}
        selectedId={wireSummaries[0].conversationID}
        onSelect$={noop}
      />,
    );
    for (const c of wireSummaries) {
      const row = screen.querySelector(`[data-testid="conversation-${c.conversationID}"]`);
      expect(row?.tagName.toLowerCase(), c.conversationID).toBe("button");
      expect(row?.getAttribute("type"), c.conversationID).toBe("button");
    }
  });

  it("renders an empty list as an empty list, not as a crash (S-CRI-004 / S-CCS-018)", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList conversations={[]} selectedId="" onSelect$={noop} />,
    );
    const list = screen.querySelector('[data-testid="conversation-list"]');
    expect(list).toBeTruthy();
    expect(list?.querySelectorAll("li").length).toBe(0);
  });

  it("pluralises turnCount correctly (1 turn vs N turns)", async () => {
    const { screen, render } = await createDOM();
    const summaries: ConversationSummary[] = [
      { conversationID: "single", lastActivityAt: "2026-08-24T17:00:00Z", turnCount: 1 },
      { conversationID: "many", lastActivityAt: "2026-08-24T17:00:00Z", turnCount: 5 },
    ];
    await render(
      <ConversationList
        conversations={summaries}
        selectedId={summaries[0].conversationID}
        onSelect$={noop}
      />,
    );
    const rowsText = screen.textContent ?? "";
    expect(rowsText).toContain("1 turn ");
    expect(rowsText).toContain("5 turns ");
  });
});
