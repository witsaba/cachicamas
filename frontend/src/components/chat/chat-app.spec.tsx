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
import { exchangesToEntries, ChatApp } from "./chat-app";
import { AGENTS } from "~/lib/mock/staff";
import type { ToolCallDTO, ToolResultDTO } from "~/lib/chat-types";

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

  it("does not render the working wasp while the turn is idle", async () => {
    // The wasp replaces the green dot ONLY while the turn is in
    // flight (streaming | submitting | cancelling). At rest the
    // header falls back to <Status /> with the agent's idle
    // status word — the wasp would be a false-positive cue.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    expect(screen.querySelector('[data-testid="work-wasp"]')).toBeFalsy();
  });

  it("does not show the 'Working now' word anywhere while idle (the wasp is the sole cue when in flight)", async () => {
    // The wasp IS the working indicator — there is no accompanying
    // "Working now" text. This test pins that contract: an idle
    // chat must never render the word "Working" in the DOM, so a
    // future refactor that re-adds a label trips the test instead
    // of silently shipping.
    const { screen, render } = await createDOM();
    await render(<ChatApp {...WHO} />);
    expect(screen.textContent ?? "").not.toContain("Working now");
    expect(screen.textContent ?? "").not.toMatch(/\bWorking\b/);
  });
});

// ---------------------------------------------------------------------------
// CH-09 WU-4a (recovery pass) — exchangesToEntries tool entry rendering.
//
// S-CTS-019 (verify #3974 UNTESTED → COMPLIANT): exchangesToEntries
// emits one `kind: "tool"` entry appended AFTER the assistant said
// entry, walking toolCalls + toolResults in lockstep with the typed
// 4-value state union ("running" | "done" | "denied" | "failed").
// The function is exported purely as a testability seam so the
// covering test exercises it without going through the live chat
// reload path.
// ---------------------------------------------------------------------------

describe("exchangesToEntries (S-CTS-019, CH-09)", () => {
  it("renders a kind: 'tool' entry from ExchangeDTO{ToolCalls: [c1], ToolResults: [r1]} — appended AFTER the assistant said entry", () => {
    const callDTO: ToolCallDTO = {
      wireCallId: "c1",
      tool: "current_time",
      arguments: '{"format":"rfc3339"}',
    };
    const resultDTO: ToolResultDTO = {
      wireCallId: "c1",
      tool: "current_time",
      outcome: "success",
      content: "2026-08-25T07:17:00Z",
      failureCategory: "",
    };
    const out = exchangesToEntries([
      {
        promptText: "What time is it?",
        assistantText: "Let me look it up.",
        partial: false,
        toolCalls: [callDTO],
        toolResults: [resultDTO],
      },
    ]);

    // Three entries: said:you, said:chat, tool — tool comes LAST.
    expect(out).toHaveLength(3);
    const [youEntry, chatEntry, toolEntry] = out;
    expect(youEntry?.kind).toBe("said");
    if (youEntry?.kind === "said") {
      expect(youEntry.who).toBe("you");
      expect(youEntry.text).toBe("What time is it?");
      expect(youEntry.state).toBe("final");
    }
    expect(chatEntry?.kind).toBe("said");
    if (chatEntry?.kind === "said") {
      expect(chatEntry.who).toBe("chat");
      expect(chatEntry.text).toBe("Let me look it up.");
      expect(chatEntry.state).toBe("final");
    }
    expect(toolEntry?.kind).toBe("tool");
    if (toolEntry?.kind === "tool") {
      // Per S-CTS-019 acceptance: kind: "tool", tool, args parsed
      // from call.arguments, result from r1.content, state "done".
      expect(toolEntry.state).toBe("done");
      expect(toolEntry.tool).toBe("current_time");
      expect(toolEntry.intent).toBe("current_time");
      expect(toolEntry.args).toEqual([["format", "rfc3339"]]);
      expect(toolEntry.result).toBe("2026-08-25T07:17:00Z");
      // The entry id is the wire call id bound with the exchange
      // index — the assistant entry MUST appear earlier in the
      // array (asserted above); the tool entry's id is grounded
      // by the wire call id so a transcript-line can render it.
      expect(toolEntry.id).toContain("c1");
    }
  });

  it("execution_failure outcome → state: 'failed', result carries the typed failure category (R-CCP-008 / D6 mirror on the reload surface)", () => {
    const callDTO: ToolCallDTO = {
      wireCallId: "c2",
      tool: "current_time",
      arguments: '{"timezone":"UTC"}',
    };
    const resultDTO: ToolResultDTO = {
      wireCallId: "c2",
      tool: "current_time",
      outcome: "execution_failure",
      // Provider text MUST NOT reach the entry on the reload surface
      // either — only the typed category is carried (D6 mirror).
      content: "provider text that must not leak",
      failureCategory: "invalid_argument",
    };
    const out = exchangesToEntries([
      {
        promptText: "What's the time in UTC?",
        assistantText: "Calling the tool.",
        partial: false,
        toolCalls: [callDTO],
        toolResults: [resultDTO],
      },
    ]);
    const toolEntry = out[2];
    expect(toolEntry?.kind).toBe("tool");
    if (toolEntry?.kind === "tool") {
      expect(toolEntry.state).toBe("failed");
      expect(toolEntry.result).toBe("invalid_argument");
      expect(toolEntry.result).not.toBe("provider text that must not leak");
    }
  });

  it("a turn with no tool calls/omits — exchangesToEntries omits the tool entry (D-4 baseline preserved)", () => {
    const out = exchangesToEntries([
      {
        promptText: "Hi.",
        assistantText: "Hello.",
        partial: false,
        // toolCalls + toolResults deliberately omitted (the JSON
        // wire omits both keys via omitempty).
      },
    ]);
    expect(out).toHaveLength(2);
    expect(out[0]?.kind).toBe("said");
    expect(out[1]?.kind).toBe("said");
    expect(out.find((e) => e.kind === "tool")).toBeUndefined();
  });
});
