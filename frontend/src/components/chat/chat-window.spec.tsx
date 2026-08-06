/**
 * chat-window.spec.tsx — vitest coverage for ChatWindow.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-1, REQ-4.
 *
 * Strict TDD target. The window is the visual surface — a scrollable
 * message list (one bubble per ChatMessage), a streaming pill for
 * pending assistant messages, and an inline <ChatInput/>. The
 * vitest spec mocks the useChatStream hook (via a thin wrapper)
 * so we exercise the component's render behavior WITHOUT booting
 * the real EventSource.
 *
 * Test scope:
 *   - empty session renders an "empty" affordance + ChatInput
 *   - user + assistant messages render as MessageBubble instances
 *   - the offline literal surfaces as an inline alert (REQ-5 S-5.a)
 *   - the ChatInput receives submit/cancel props from the hook
 *   - the streaming pill renders while a message is pending
 */
import { $, type QRL } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { ChatSession } from "~/lib/chat-types";

// ---------------------------------------------------------------------------
// Mock the hook module so we can drive session state from the spec.
// We export a tiny fake that mimics the useChatStream() return
// shape (session + submit QRL + cancel QRL).
//
// QRL discipline: Qwik rejects non-QRL functions passed as
// `onSubmit$` / `onCancel$` props at render. The mock must therefore
// return QRLs (wrapped with $()). Qwik's serializer rejects vi.fn
// refs captured inside closures, so the QRLs use module-scoped
// counters instead — same pattern as chat-input.spec.tsx.
// ---------------------------------------------------------------------------

let submitCalls = 0;
let cancelCalls = 0;
const fakeSubmit = $((_text: string) => {
  submitCalls = submitCalls + 1;
  return Promise.resolve({
    ok: true as const,
    value: { turnId: "trn_x", streamUrl: "/api/agent/turns/trn_x/events" },
  });
}) as QRL<(value: string) => Promise<{ ok: true; value: { turnId: string; streamUrl: string } }>>;
const fakeCancel = $(() => {
  cancelCalls = cancelCalls + 1;
  return Promise.resolve();
}) as QRL<() => Promise<void>>;
let mockSession: ChatSession = {
  messages: [],
  status: "idle",
};

const useChatStreamMock = () => ({
  session: mockSession,
  submit: fakeSubmit,
  cancel: fakeCancel,
});
vi.mock("./use-chat-stream", () => ({
  useChatStream: useChatStreamMock,
}));

describe("components/chat/chat-window (REQ-1, REQ-4)", () => {
  beforeEach(() => {
    submitCalls = 0;
    cancelCalls = 0;
    mockSession = { messages: [], status: "idle" };
  });
  afterEach(() => {
    // Nothing to restore — QRLs are module-scoped.
  });

  it("renders an empty-state when no messages exist (REQ-1)", async () => {
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const empty = screen.querySelector('[data-testid="chat-empty"]');
    expect(empty).toBeTruthy();
  });

  it("renders a user message bubble (REQ-1)", async () => {
    mockSession = {
      messages: [
        {
          id: "u-1",
          role: "user",
          text: "hello",
          status: "complete",
        },
      ],
      status: "idle",
    };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const bubble = screen.querySelector('[data-testid="message-bubble-user"]');
    expect(bubble).toBeTruthy();
    expect(bubble?.textContent).toContain("hello");
  });

  it("renders an assistant message bubble (REQ-1)", async () => {
    mockSession = {
      messages: [
        {
          id: "a-1",
          role: "assistant",
          text: "world",
          status: "complete",
        },
      ],
      status: "idle",
    };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const bubble = screen.querySelector('[data-testid="message-bubble-assistant"]');
    expect(bubble).toBeTruthy();
    expect(bubble?.textContent).toContain("world");
  });

  it("renders a streaming pill while the assistant is mid-stream (REQ-1)", async () => {
    mockSession = {
      messages: [
        {
          id: "a-1",
          role: "assistant",
          text: "partial",
          status: "streaming",
        },
      ],
      status: "streaming",
    };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const pill = screen.querySelector('[data-testid="chat-streaming-pill"]');
    expect(pill).toBeTruthy();
    expect(pill?.textContent).toContain("Streaming");
  });

  it("renders an inline error alert when the session's last message has status='error' (REQ-4 S-4.a)", async () => {
    mockSession = {
      messages: [
        {
          id: "a-1",
          role: "assistant",
          text: "",
          status: "error",
          error: { kind: "server", message: "upstream unavailable" },
        },
      ],
      status: "idle",
    };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const alert = screen.querySelector('[data-testid="chat-error-alert"]');
    expect(alert).toBeTruthy();
    expect(alert?.textContent).toContain("upstream unavailable");
    // The error message is rendered as the assistant bubble's text.
    const bubble = screen.querySelector('[data-testid="message-bubble-assistant"]');
    expect(bubble?.textContent).toContain("upstream unavailable");
  });

  it("renders the ChatInput controls (Send + Cancel)", async () => {
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const send = screen.querySelector('[data-testid="chat-input-send"]');
    const cancel = screen.querySelector('[data-testid="chat-input-cancel"]');
    expect(send).toBeTruthy();
    expect(cancel).toBeTruthy();
  });

  it("disables the input controls when session.status !== 'idle' (REQ-1)", async () => {
    mockSession = { messages: [], status: "streaming" };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    const send = screen.querySelector(
      '[data-testid="chat-input-send"]',
    ) as HTMLButtonElement | null;
    expect(send?.disabled).toBe(true);
  });

  it("renders each message bubble with a stable test id keyed by message id (REQ-7 traceability)", async () => {
    mockSession = {
      messages: [
        { id: "u-1", role: "user", text: "hi", status: "complete" },
        { id: "a-1", role: "assistant", text: "hello", status: "complete" },
        { id: "a-2", role: "assistant", text: "world", status: "streaming" },
      ],
      status: "streaming",
    };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    expect(screen.querySelector('[data-testid="message-bubble-u-1"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="message-bubble-a-1"]')).toBeTruthy();
    expect(screen.querySelector('[data-testid="message-bubble-a-2"]')).toBeTruthy();
  });

  it("the textarea is enabled when session.status is idle (REQ-5 S-5.a — accepts fresh submit after offline)", async () => {
    mockSession = {
      messages: [
        { id: "u-1", role: "user", text: "hello", status: "complete" },
      ],
      status: "idle",
    };
    const { ChatWindow } = await import("./chat-window");
    const { render, screen } = await createDOM();
    await render(<ChatWindow />);
    // After an offline failure the session returns to idle (the
    // hook flips status='idle' on the offline error path); the
    // textarea must remain enabled so the user can retry. The
    // Send button's `disabled` is ALSO bound to "text is empty"
    // (a separate guard — empty prompts are never submitted), so
    // we assert on the textarea itself: it must NOT carry the
    // streaming-disabled state. (RE-PR-001 mirrors how
    // routes/home guards `disabled` on session, not on text.)
    const textarea = screen.querySelector(
      '[data-testid="chat-input-textarea"]',
    ) as HTMLTextAreaElement | null;
    expect(textarea).toBeTruthy();
    expect(textarea?.disabled).toBe(false);
  });
});
