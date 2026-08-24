/**
 * use-chat-stream.spec.ts — RED-GREEN for CH-05.1.
 *
 * Reference: openspec/changes/cachicamas-chat-frontend-wire/design.md §4
 * (RED cases RED-1 .. RED-10). The 10 cases below are the
 * acceptance criteria for `useChatStream`. Test names carry the
 * REQ-N identifier per frontend-chat-layer1/spec.md:91.
 *
 * Conventions:
 *   - FakeES is copied verbatim from chat-api.spec.ts:394-425 (D-5,
 *     explore §8.1). REQ-7 S-7.b catches any drift.
 *   - The chat-api module is mocked wholesale (submitTurn /
 *     cancelTurn / subscribeTurn) so the tests can drive the
 *     hook's lifecycle from the test without a real EventSource
 *     and without a running backend.
 *   - The harness renders the hook and exposes its state via a
 *     mutable ref the test reads after awaiting each action.
 */

import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { component$ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";

// Mocked wire-client surface. The factory must be hoisted before
// the import of the hook resolves, so the module-level `vi.mock`
// uses the `vi.fn()` placeholders below; per-test overrides wire
// them via the exported references (which the factory closes over).
const submitTurnMock: Mock = vi.fn();
const cancelTurnMock: Mock = vi.fn();
const subscribeTurnMock: Mock = vi.fn();

vi.mock("~/lib/chat-api", async () => {
  const actual = await vi.importActual<typeof import("~/lib/chat-api")>("~/lib/chat-api");
  return {
    ...actual,
    submitTurn: (...args: unknown[]) => submitTurnMock(...args),
    cancelTurn: (...args: unknown[]) => cancelTurnMock(...args),
    subscribeTurn: (...args: unknown[]) => subscribeTurnMock(...args),
  };
});

// Importing after vi.mock so the test sees the mocked module.
const { useChatStream } = await import("./use-chat-stream");
import type { ChatStreamState as _ChatStreamState } from "./use-chat-stream";
type ChatStreamState = _ChatStreamState;

interface HarnessHandles {
  state: { value: ChatStreamState | null };
}

// Test harness: renders useChatStream and exposes its state via a
// mutable ref the test reads. The hook owns all the lifecycle; the
// harness exposes only the public surface.
const TestHarness = component$<{ handles: HarnessHandles }>(({ handles }) => {
  const state = useChatStream([]);
  handles.state.value = state;
  return <div data-testid="harness" />;
});

// Drain microtasks + macrotasks so async action QRLs finish their
// first run before the test reads `handles.state.value`.
async function flush(): Promise<void> {
  // Three cycles of microtask + macrotask drains reliably move
  // Qwik's QRL resolution + the mock resolution forward.
  for (let i = 0; i < 5; i += 1) {
    await new Promise((r) => setTimeout(r, 10));
  }
}

beforeEach(() => {
  submitTurnMock.mockReset();
  cancelTurnMock.mockReset();
  subscribeTurnMock.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useChatStream (REQ-1, REQ-2, REQ-4, REQ-5)", () => {
  it("RED-1: useChatStream is exported as a function (REQ-1 module surface)", () => {
    expect(typeof useChatStream).toBe("function");
  });

  it("RED-2: submit invokes submitTurn with the typed payload and opens subscribeTurn on the returned streamUrl (REQ-1 S-1.a)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_42", streamUrl: "/api/agent/turns/trn_42/events" },
    });
    subscribeTurnMock.mockReturnValue(() => {
      /* unsubscribe */
    });
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    expect(handles.state.value).not.toBeNull();
    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();

    expect(submitTurnMock).toHaveBeenCalledTimes(1);
    const [req] = submitTurnMock.mock.calls[0] as [{ id: string; prompt: string }];
    expect(req).toMatchObject({ prompt: "hello" });
    expect(typeof req.id).toBe("string");
    expect(req.id.length).toBeGreaterThan(0);

    expect(subscribeTurnMock).toHaveBeenCalledTimes(1);
    const [url] = subscribeTurnMock.mock.calls[0] as [string];
    expect(url).toBe("/api/agent/turns/trn_42/events");

    expect(state.status).toBe("streaming");
    // The cancel key sent to DELETE is the hook's client-minted id, NOT
    // the backend's trn_42. The hook only knows its own id.
    expect(state.currentTurnId).toBe(req.id);
    const assistant = [...state.entries].reverse().find(
      (e) => e.kind === "said" && e.who === "chat",
    );
    expect(assistant?.kind).toBe("said");
    if (assistant && assistant.kind === "said") {
      expect(assistant.state).toBe("streaming");
    }
  });

  it("RED-3: message.delta accumulates into the in-flight assistant entry (REQ-1 S-1.a)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_1", streamUrl: "/api/agent/turns/trn_1/events" },
    });
    let onEventCb: ((ev: unknown) => void) | null = null;
    subscribeTurnMock.mockImplementation((_url: string, onEvent: (ev: unknown) => void) => {
      onEventCb = onEvent;
      return () => {};
    });
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    (onEventCb as unknown as (ev: unknown) => void)({
      kind: "message.delta",
      index: 0,
      delta: "Hello, ",
    });
    (onEventCb as unknown as (ev: unknown) => void)({
      kind: "message.delta",
      index: 0,
      delta: "world!",
    });
    await flush();

    const assistant = [...state.entries].reverse().find(
      (e) => e.kind === "said" && e.who === "chat",
    );
    expect(assistant?.kind).toBe("said");
    if (assistant && assistant.kind === "said") {
      expect(assistant.text).toBe("Hello, world!");
    }
  });

  it("RED-4: turn.end finalises the in-flight entry and unsubscribe fires exactly once (REQ-1 S-1.a, REQ-2 S-2.c)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_1", streamUrl: "/api/agent/turns/trn_1/events" },
    });
    let onEventCb: ((ev: unknown) => void) | null = null;
    let unsubscribeCount = 0;
    subscribeTurnMock.mockImplementation((_url: string, onEvent: (ev: unknown) => void) => {
      onEventCb = onEvent;
      return () => {
        unsubscribeCount += 1;
      };
    });
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    (onEventCb as unknown as (ev: unknown) => void)({
      kind: "message.delta",
      index: 0,
      delta: "done",
    });
    (onEventCb as unknown as (ev: unknown) => void)({ kind: "turn.end", finishReason: "stop" });
    await flush();

    expect(state.status).toBe("idle");
    const assistant = [...state.entries].reverse().find(
      (e) => e.kind === "said" && e.who === "chat",
    );
    expect(assistant?.kind).toBe("said");
    if (assistant && assistant.kind === "said") {
      expect(assistant.state).toBe("final");
      expect(assistant.text).toBe("done");
    }
    // turn.end closes the stream exactly once (REQ-1 S-1.a, REQ-2 S-2.c).
    // The hook invokes unsubscribe exactly once when the stream terminates
    // naturally — no DELETE.
    expect(cancelTurnMock).not.toHaveBeenCalled();
    expect(unsubscribeCount).toBe(1);
  });

  it("RED-5: stop invokes cancelTurn with the current turnId and closes the EventSource (REQ-2 S-2.b)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_77", streamUrl: "/api/agent/turns/trn_77/events" },
    });
    let unsubscribeCount = 0;
    subscribeTurnMock.mockImplementation((_url: string, _onEvent: (ev: unknown) => void) => {
      return () => {
        unsubscribeCount += 1;
      };
    });
    cancelTurnMock.mockResolvedValue({ ok: true, value: null });
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    // The cancel key is the hook's client-minted id (the same id passed
    // to submitTurn), NOT the backend's trn_77. Snapshot it before cancel.
    const submitCallId = (submitTurnMock.mock.calls[0] as [{ id: string }])[0].id;
    await state.cancel();
    await flush();

    expect(cancelTurnMock).toHaveBeenCalledTimes(1);
    const [req] = cancelTurnMock.mock.calls[0] as [{ id: string }];
    expect(req).toEqual({ id: submitCallId });
    // The unsubscribe returned by subscribeTurn is invoked by cancel.
    expect(unsubscribeCount).toBe(1);
  });

  it("RED-6: cancel invokes cancelTurn and unsubscribe (REQ-2 S-2.a) — both fire exactly once", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_99", streamUrl: "/api/agent/turns/trn_99/events" },
    });
    let unsubscribeCount = 0;
    subscribeTurnMock.mockImplementation((_url: string, _onEvent: (ev: unknown) => void) => {
      return () => {
        unsubscribeCount += 1;
      };
    });
    cancelTurnMock.mockResolvedValue({ ok: true, value: null });
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    await state.cancel();
    await flush();

    expect(cancelTurnMock).toHaveBeenCalledTimes(1);
    expect(unsubscribeCount).toBe(1);
  });

  it("RED-7: event: error with kind:'server' surfaces state.error and re-enables the input (REQ-4 S-4.a)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_err", streamUrl: "/api/agent/turns/trn_err/events" },
    });
    let onEventCb: ((ev: unknown) => void) | null = null;
    subscribeTurnMock.mockImplementation((_url: string, onEvent: (ev: unknown) => void) => {
      onEventCb = onEvent;
      return () => {};
    });
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    (onEventCb as unknown as (ev: unknown) => void)({
      kind: "error",
      error: { kind: "server", message: "upstream unavailable" },
    });
    await flush();

    expect(state.error).toBeDefined();
    expect(state.error?.kind).toBe("server");
    if (state.error && state.error.kind === "server") {
      expect(state.error.message).toBe("upstream unavailable");
    }
    expect(state.status).toBe("idle");
  });

  it("RED-8: submitTurn resolving to kind:'offline' sets state.error.message to the amended phrase (REQ-5 S-5.a amended)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: false,
      kind: "offline",
      message: "Couldn't reach the chat service. Is docker compose up? (network error)",
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();

    expect(state.error).toBeDefined();
    expect(state.error?.kind).toBe("offline");
    expect(state.error?.message).toContain(
      "Couldn't reach the chat service. Is docker compose up?",
    );
    expect(state.status).toBe("idle");
    expect(subscribeTurnMock).not.toHaveBeenCalled();
  });

  it("RED-9: the retired literal 'backend not wired — see PR for backend wire' never surfaces in the offline message (REQ-5 S-5.c new)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: false,
      kind: "offline",
      message: "Couldn't reach the chat service. Is docker compose up? (network error)",
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();

    const errMsg = state.error?.message;
    expect(errMsg === undefined || !errMsg.includes("backend not wired")).toBe(
      true,
    );
  });

  it("RED-10: the offline kind is preserved (D-1) — ChatStreamError union still includes { kind: 'offline' }", () => {
    const variants: ChatStreamState["error"][] = [
      { kind: "offline", message: "o" },
      { kind: "validation", message: "v", fields: {} },
      { kind: "conflict", message: "c" },
      { kind: "not_found", message: "n" },
      { kind: "server", message: "s" },
    ];
    const kinds = variants.map((v) => v && v.kind);
    expect(kinds).toEqual([
      "offline",
      "validation",
      "conflict",
      "not_found",
      "server",
    ]);
  });
});
