/**
 * ConversationList — the archetype's own memory.
 */
import { $ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ConversationList } from "./conversation-list";
import { CONVERSATIONS } from "~/lib/mock/chat";

const noop = $((_id: string) => undefined);

describe("components/chat/conversation-list", () => {
  it("renders one row per conversation", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={CONVERSATIONS}
        selectedId={CONVERSATIONS[0].id}
        onSelect$={noop}
      />,
    );
    for (const c of CONVERSATIONS) {
      expect(
        screen.querySelector(`[data-testid="conversation-${c.id}"]`),
        c.id,
      ).toBeTruthy();
    }
  });

  it("shows what a person actually scans by: title, turns, age", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={CONVERSATIONS}
        selectedId={CONVERSATIONS[0].id}
        onSelect$={noop}
      />,
    );
    const first = CONVERSATIONS[0];
    const row = screen.querySelector(
      `[data-testid="conversation-${first.id}"]`,
    );
    const text = row?.textContent ?? "";
    expect(text).toContain(first.title);
    expect(text).toContain(`${first.turns} turns`);
    expect(text).toContain(first.age);
  });

  it("marks exactly one row current, and announces it", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={CONVERSATIONS}
        selectedId={CONVERSATIONS[1].id}
        onSelect$={noop}
      />,
    );
    const current = screen.querySelectorAll('[aria-current="true"]');
    expect(current.length).toBe(1);
    expect(current[0].getAttribute("data-testid")).toBe(
      `conversation-${CONVERSATIONS[1].id}`,
    );
  });

  it("uses real buttons, so the list is keyboard reachable", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList
        conversations={CONVERSATIONS}
        selectedId={CONVERSATIONS[0].id}
        onSelect$={noop}
      />,
    );
    for (const c of CONVERSATIONS) {
      const row = screen.querySelector(`[data-testid="conversation-${c.id}"]`);
      expect(row?.tagName.toLowerCase(), c.id).toBe("button");
      expect(row?.getAttribute("type"), c.id).toBe("button");
    }
  });

  it("renders an empty list as an empty list, not as a crash", async () => {
    const { screen, render } = await createDOM();
    await render(
      <ConversationList conversations={[]} selectedId="" onSelect$={noop} />,
    );
    const list = screen.querySelector('[data-testid="conversation-list"]');
    expect(list).toBeTruthy();
    expect(list?.querySelectorAll("li").length).toBe(0);
  });
});
