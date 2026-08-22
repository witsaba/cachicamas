/**
 * ChatApp — the chat archetype's screen.
 *
 * The turn machine is unit-tested in `use-mock-turn.spec.ts`, where it can be
 * driven a tick at a time. What is asserted here is composition: the two
 * panels, the seeded transcript, the announcement of streaming output to
 * assistive technology, and the standing demonstration notice.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect } from "vitest";
import { ChatApp } from "./chat-app";
import { CONVERSATIONS } from "~/lib/mock/chat";

describe("components/chat/chat-app", () => {
  it("renders the memory panel and the run panel", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    expect(screen.querySelector('[data-testid="conversations-panel"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="transcript-panel"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="composer"]')).toBeTruthy();
  });

  it("opens on the most recent conversation, whole", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    const transcript = screen.querySelector('[data-testid="transcript"]');
    for (const entry of CONVERSATIONS[0].entries) {
      expect(
        transcript?.querySelector(`[data-testid$="-${entry.id}"]`),
        entry.id,
      ).toBeTruthy();
    }
  });

  it("announces the transcript politely, never assertively", async () => {
    // Streaming output that interrupts a screen reader mid-sentence is worse
    // than output it has to be asked for.
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    const transcript = screen.querySelector('[data-testid="transcript"]');
    expect(transcript?.getAttribute("aria-live")).toBe("polite");
    expect(transcript?.getAttribute("aria-label")).toBe("Conversation");
  });

  it("reports the run's status in the panel header, in words", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    const note = screen.querySelector('[data-testid="transcript-panel-note"]');
    expect(note?.textContent).toContain("Idle");
    expect(note?.textContent).toContain(`${CONVERSATIONS[0].turns} turns`);
  });

  it("names the open conversation on the panel it is in", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    const panel = screen.querySelector('[data-testid="transcript-panel"]');
    expect(panel?.querySelector("h2")?.textContent).toBe(CONVERSATIONS[0].title);
  });

  it("states the archetype's real, unshipped status in the title block", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    const title = screen.querySelector('[data-testid="screen-title"]');
    const text = title?.textContent ?? "";
    expect(text).toContain("Chat");
    expect(text).toContain("doc 0005");
    expect(text).toContain("0 of 12");
  });

  it("marks every conversation on offer as demonstration data", async () => {
    const { screen, render } = await createDOM();
    await render(<ChatApp />);
    expect(
      screen.querySelector('[data-testid="conversations-panel-note"]')?.textContent,
    ).toContain("demo");
  });
});
