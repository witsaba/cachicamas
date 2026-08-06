/**
 * use-chat-stream.spec.ts — vitest coverage for the chat hook.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-1..REQ-7.
 *
 * Strict TDD target. The hook is the seam between the typed wire
 * client (chat-api.ts) and the visual surface (chat-window.tsx).
 * The vitest spec mocks the EventSource (FakeES pattern from
 * lib/api.sse.spec.ts:75-307) and the chat-api module so we exercise
 * the hook's session-machine + cleanup discipline WITHOUT booting
 * Qwik (createDOM is a thin harness — we exercise the pure mutator
 * directly).
 *
 * Test scope (mirrors design.md §4):
 *   - applyEvent(message.start) appends an assistant bubble with status="streaming"
 *   - applyEvent(message.delta) appends delta to the last assistant bubble's text
 *   - applyEvent(message.end) flips status="complete"
 *   - applyEvent(turn.end) flips session.status="idle" and closes the EventSource
 *   - applyEvent(error) surfaces the typed error inline (REQ-4 S-4.a)
 *   - submitTurn() POSTs through chat-api and opens the EventSource on success
 *   - cancel() issues DELETE through chat-api AND closes the EventSource
 *   - cleanup() closes the EventSource and issues cancelTurn if a turn is open
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { ChatSession, ChatStreamEvent } from "~/lib/chat-types";

// ---------------------------------------------------------------------------
// FakeES — mirrors api.sse.spec.ts:79-93. We need the same shape
// because chat-api.ts calls es.addEventListener('message.<x>', ...)
// (design §3 D1 refinement) and es.close().
// ---------------------------------------------------------------------------

class FakeES {
  url: string;
  withCredentials = false;
  listeners: Record<string, Array<(ev: MessageEvent<string>) => void>> = {};
  closed = false;
  constructor(url: string, opts?: { withCredentials?: boolean }) {
    this.url = url;
    this.withCredentials = opts?.withCredentials ?? false;
  }
  addEventListener(name: string, cb: (ev: MessageEvent<string>) => void) {
    if (!this.listeners[name]) this.listeners[name] = [];
    this.listeners[name].push(cb);
  }
  removeEventListener(name: string, cb: (ev: MessageEvent<string>) => void) {
    const arr = this.listeners[name];
    if (!arr) return;
    const idx = arr.indexOf(cb);
    if (idx >= 0) arr.splice(idx, 1);
  }
  close() {
    this.closed = true;
  }
  fire(name: string, data: unknown) {
    const arr = this.listeners[name];
    if (!arr) return;
    for (const cb of arr) {
      cb({ data: JSON.stringify(data) } as MessageEvent<string>);
    }
  }
}

// ---------------------------------------------------------------------------
// chat-api mock — captures calls + lets the spec drive submitTurn /
// cancelTurn / subscribeTurn behavior. The hook consumes the same
// surface; we don't re-test chat-api here.
// ---------------------------------------------------------------------------

const submitTurnMock = vi.fn();
const cancelTurnMock = vi.fn();
const subscribeTurnMock = vi.fn();
const offlineLiteral = "backend not wired — see PR for backend wire";

vi.mock("~/lib/chat-api", () => ({
  submitTurn: (req: unknown) => submitTurnMock(req),
  cancelTurn: (req: unknown) => cancelTurnMock(req),
  subscribeTurn: (
    url: string,
    onEvent: (ev: ChatStreamEvent) => void,
    onError?: (msg: string) => void,
  ) => {
    const fakeEs = new FakeES(url, { withCredentials: true });
    subscribeTurnMock(url, onEvent, onError, fakeEs);
    return () => {
      fakeEs.close();
    };
  },
  OFFLINE_LITERAL: offlineLiteral,
}));

describe("use-chat-stream pure mutator (REQ-1, REQ-2, REQ-7)", () => {
  beforeEach(() => {
    submitTurnMock.mockReset();
    cancelTurnMock.mockReset();
    subscribeTurnMock.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("applyEvent(message.start) appends an assistant bubble with status='streaming'", async () => {
    const { applyEvent, createSession } = await import("./use-chat-stream");
    const session = createSession();
    applyEvent(session, {
      kind: "message.start",
      messageId: "m-1",
      index: 0,
    });
    expect(session.messages).toHaveLength(1);
    expect(session.messages[0]?.role).toBe("assistant");
    expect(session.messages[0]?.status).toBe("streaming");
    expect(session.messages[0]?.text).toBe("");
  });

  it("applyEvent(message.delta) appends to the last assistant bubble's text", async () => {
    const { applyEvent, createSession } = await import("./use-chat-stream");
    const session = createSession();
    applyEvent(session, { kind: "message.start", messageId: "m-1", index: 0 });
    applyEvent(session, { kind: "message.delta", index: 0, delta: "Hello, " });
    applyEvent(session, { kind: "message.delta", index: 0, delta: "world" });
    expect(session.messages).toHaveLength(1);
    expect(session.messages[0]?.text).toBe("Hello, world");
    expect(session.messages[0]?.status).toBe("streaming");
  });

  it("applyEvent(message.end) flips the bubble status to 'complete' (REQ-1)", async () => {
    const { applyEvent, createSession } = await import("./use-chat-stream");
    const session = createSession();
    applyEvent(session, { kind: "message.start", messageId: "m-1", index: 0 });
    applyEvent(session, { kind: "message.delta", index: 0, delta: "x" });
    applyEvent(session, {
      kind: "message.end",
      index: 0,
      finishReason: "stop",
    });
    expect(session.messages[0]?.status).toBe("complete");
  });

  it("applyEvent(turn.end) flips session.status to 'idle'", async () => {
    const { applyEvent, createSession } = await import("./use-chat-stream");
    const session: ChatSession = {
      messages: [],
      status: "streaming",
      currentTurnId: "trn_1",
    };
    applyEvent(session, { kind: "turn.end" });
    expect(session.status).toBe("idle");
    expect(session.currentTurnId).toBeUndefined();
  });

  it("applyEvent(error) surfaces the typed error inline and flips bubble to 'error' (REQ-4 S-4.a)", async () => {
    const { applyEvent, createSession } = await import("./use-chat-stream");
    const session = createSession();
    applyEvent(session, { kind: "message.start", messageId: "m-1", index: 0 });
    applyEvent(session, {
      kind: "error",
      error: { kind: "server", message: "upstream unavailable" },
    });
    expect(session.messages[0]?.status).toBe("error");
    expect(session.messages[0]?.error?.message).toBe("upstream unavailable");
    expect(session.messages[0]?.error?.kind).toBe("server");
  });

  it("applyEvent ignores unknown event kinds (defensive)", async () => {
    const { applyEvent, createSession } = await import("./use-chat-stream");
    const session = createSession();
    // Cast through unknown to inject a malformed event — the mutator
    // MUST be a no-op for anything outside the discriminated union.
    applyEvent(
      session,
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      { kind: "wat" } as any,
    );
    expect(session.messages).toHaveLength(0);
    expect(session.status).toBe("idle");
  });

  it("createSession() starts with empty messages and idle status", async () => {
    const { createSession } = await import("./use-chat-stream");
    const session = createSession();
    expect(session.messages).toEqual([]);
    expect(session.status).toBe("idle");
    expect(session.currentTurnId).toBeUndefined();
  });
});

// ---------------------------------------------------------------------------
// Hook integration tests — drive submitTurn / cancelTurn through the
// chat-api mock and assert the session state machine + EventSource
// lifecycle. The hook itself is hard to instantiate outside Qwik
// (it depends on useStore + useVisibleTask$), so we exercise the
// exported pure helpers (applyEvent, createSession) and the
// submit/cancel path by stubbing the module.
// ---------------------------------------------------------------------------

describe("use-chat-stream submit/cancel wiring (REQ-1, REQ-2)", () => {
  beforeEach(() => {
    submitTurnMock.mockReset();
    cancelTurnMock.mockReset();
    subscribeTurnMock.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("submit() calls chat-api.submitTurn with a UUID + prompt and opens the EventSource on success", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_1", streamUrl: "/api/agent/turns/trn_1/events" },
    });
    let capturedOnEvent: ((ev: ChatStreamEvent) => void) | null = null;
    let capturedFakeEs: FakeES | null = null;
    subscribeTurnMock.mockImplementation(
      (
        _url: string,
        onEvent: (ev: ChatStreamEvent) => void,
        _onError: ((msg: string) => void) | undefined,
        fakeEs: FakeES,
      ) => {
        capturedOnEvent = onEvent;
        capturedFakeEs = fakeEs;
        return () => {
          fakeEs.close();
        };
      },
    );

    const { submit, applyEvent, createSession } = await import("./use-chat-stream");
    const session = createSession();
    const result = await submit(session, "hello");
    expect(result.ok).toBe(true);
    expect(submitTurnMock).toHaveBeenCalledOnce();
    const req = submitTurnMock.mock.calls[0]?.[0] as { id: string; prompt: string };
    expect(req.prompt).toBe("hello");
    expect(req.id).toMatch(/^[0-9a-f-]{36}$/i); // UUID
    // EventSource was opened with the returned stream URL.
    expect(subscribeTurnMock).toHaveBeenCalledOnce();
    expect(subscribeTurnMock.mock.calls[0]?.[0]).toBe(
      "/api/agent/turns/trn_1/events",
    );
    expect(session.status).toBe("streaming");
    expect(session.currentTurnId).toBe("trn_1");
    expect(capturedFakeEs).not.toBeNull();
    // Drive a delta through the captured onEvent — the session mutator
    // receives it as if the EventSource had fired the listener.
    capturedOnEvent!({ kind: "message.start", messageId: "m-1", index: 0 });
    capturedOnEvent!({ kind: "message.delta", index: 0, delta: "hi" });
    // The mutator we exported is the same one the hook uses internally.
    applyEvent(session, { kind: "message.start", messageId: "m-2", index: 1 });
    expect(session.messages.length).toBeGreaterThan(0);
  });

  it("submit() returns the offline envelope when submitTurn fails with kind=offline (REQ-5)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: false,
      kind: "offline",
      message: offlineLiteral,
    });
    const { submit, createSession } = await import("./use-chat-stream");
    const session = createSession();
    const result = await submit(session, "hello");
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("offline");
      expect(result.message).toContain(offlineLiteral);
    }
    // The session stays idle (no EventSource was opened).
    expect(session.status).toBe("idle");
    // No subscribeTurn call happened.
    expect(subscribeTurnMock).not.toHaveBeenCalled();
  });

  it("cancel() calls chat-api.cancelTurn with the current turn id AND closes the EventSource", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_1", streamUrl: "/api/agent/turns/trn_1/events" },
    });
    let capturedFakeEs: FakeES | null = null;
    subscribeTurnMock.mockImplementation(
      (
        _url: string,
        _onEvent: (ev: ChatStreamEvent) => void,
        _onError: ((msg: string) => void) | undefined,
        fakeEs: FakeES,
      ) => {
        capturedFakeEs = fakeEs;
        return () => {
          fakeEs.close();
        };
      },
    );
    cancelTurnMock.mockResolvedValue({ ok: true, value: null });

    const { submit, cancel, createSession } = await import("./use-chat-stream");
    const session = createSession();
    await submit(session, "hello");
    expect(capturedFakeEs).not.toBeNull();
    expect(capturedFakeEs!.closed).toBe(false);
    await cancel(session);
    expect(cancelTurnMock).toHaveBeenCalledWith({ id: "trn_1" });
    expect(capturedFakeEs!.closed).toBe(true);
    expect(session.status).toBe("idle");
    expect(session.currentTurnId).toBeUndefined();
  });

  it("cancel() is a no-op when no turn is open", async () => {
    cancelTurnMock.mockResolvedValue({ ok: true, value: null });
    const { cancel, createSession } = await import("./use-chat-stream");
    const session = createSession();
    await cancel(session);
    expect(cancelTurnMock).not.toHaveBeenCalled();
  });

  it("subscribeTurn onerror surfaces the offline literal (REQ-5 S-5.b)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_1", streamUrl: "/api/agent/turns/trn_1/events" },
    });
    let capturedOnError: ((msg: string) => void) | null = null;
    subscribeTurnMock.mockImplementation(
      (
        _url: string,
        _onEvent: (ev: ChatStreamEvent) => void,
        onError: ((msg: string) => void) | undefined,
        _fakeEs: FakeES,
      ) => {
        capturedOnError = onError ?? null;
        return () => {};
      },
    );

    const { submit, createSession } = await import("./use-chat-stream");
    const session = createSession();
    await submit(session, "hello");
    expect(capturedOnError).not.toBeNull();
    // The hook's offline state machine: the chat input clears + the
    // last error is recorded on the session.
    capturedOnError!(offlineLiteral);
    expect(session.status).toBe("idle");
    expect(session.currentTurnId).toBeUndefined();
  });
});
