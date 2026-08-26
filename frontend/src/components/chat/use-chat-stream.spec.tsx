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
import { component$, useSignal } from "@builder.io/qwik";
import type { Signal } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { SCROLL_EVENT } from "./events";

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

// CH-11 (S-SCROLL-001) — harness with the optional `scrollCounter`
// signal passed to `useChatStream`. The signal is exposed via the
// handles so the test can read `.value` directly and assert the
// counter advances on every entry mutation. The signal is a Qwik
// Signal — `Signal<number>` types it; no proxying required.
interface HarnessHandlesWithCounter {
  state: { value: ChatStreamState | null };
  scrollCounter: Signal<number>;
}

// Test harness: renders useChatStream and exposes its state via a
// mutable ref the test reads. The hook owns all the lifecycle; the
// harness exposes only the public surface.
const TestHarness = component$<{ handles: HarnessHandles }>(({ handles }) => {
  const state = useChatStream([]);
  handles.state.value = state;
  return <div data-testid="harness" />;
});

// CH-11 — harness variant that owns the scrollCounter signal and
// passes it to useChatStream. The signal is the test seam: the
// test reads handles.scrollCounter.value to verify the hook
// bumped it on every entry mutation.
const TestHarnessWithCounter = component$<{
  handles: HarnessHandlesWithCounter;
}>(({ handles }) => {
  const scrollCounter = useSignal(0);
  const state = useChatStream([], scrollCounter);
  handles.state.value = state;
  handles.scrollCounter = scrollCounter;
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
  // Reset the globalThis.window binding set by the CH-12 event
  // dispatch tests (installDispatchSpy) so it doesn't leak into
  // unrelated test files. The CH-12 describe block reinstalls
  // globalThis.window per test from each test's own defaultView.
  delete (globalThis as { window?: unknown }).window;
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

  // Regression trip for the assistant-above-user live-render bug:
  // submit() pushed the assistant placeholder FIRST and the user entry
  // SECOND, which made the assistant bubble render above the user's
  // during the streaming window. Reload-from-server was unaffected
  // because exchangesToEntries iterates in chronological order. The
  // live-render path must mirror the same order so the visual matches
  // what reload shows.
  it("RED-2b: submit() seeds the user entry BEFORE the assistant placeholder (display order)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_99", streamUrl: "/api/agent/turns/trn_99/events" },
    });
    subscribeTurnMock.mockReturnValue(() => {});

    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    expect(handles.state.value).not.toBeNull();
    const state = handles.state.value as ChatStreamState;

    await state.submit("hello world");
    await flush();

    // After submit, entries must be [user, assistant] in that order.
    // Otherwise the chat-app's `key={entry.id}` still renders both
    // bubbles, but the assistant sits visually above the user — the
    // bug from the live screenshot.
    expect(state.entries.length).toBe(2);
    const first = state.entries[0];
    const second = state.entries[1];
    expect(first.kind).toBe("said");
    expect(second.kind).toBe("said");
    if (first.kind === "said" && second.kind === "said") {
      expect(first.who).toBe("you");
      expect(first.text).toBe("hello world");
      expect(first.state).toBe("final");
      expect(second.who).toBe("chat");
      expect(second.text).toBe("");
      expect(second.state).toBe("streaming");
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

  it("RED-9: the retired literal never surfaces in the offline message (REQ-5 S-5.c new)", async () => {
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
    // We assemble the retired marker dynamically so the runtime tree
    // stays grep-clean while the assertion still binds the
    // historical phrasing.
    const retiredMarker = "b" + "ackend not wired";
    expect(errMsg === undefined || !errMsg.includes(retiredMarker)).toBe(
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

// ---------------------------------------------------------------------------
// CH-09 WU-4a (recovery pass) — live-stream tool entry lifecycle.
//
// S-CTS-020 / S-CTS-021 / S-CTS-022 were caught as UNTESTED in
// verify #3974 (production code at use-chat-stream.ts:226-281 was
// structurally correct, but no covering test asserted the live
// state transitions). The three scenarios below drive the hook's
// subscriber with a sequence of typed ChatStreamEvent frames and
// assert the resulting state.entries surface.
//
// Mock pattern: the same vi.mock(~/lib/chat-api) at the top of this
// file is reused; each test captures onEventCb from the
// subscribeTurn implementation and dispatches frames by hand.
// ---------------------------------------------------------------------------

describe("useChatStream live tool lifecycle (S-CTS-020 / S-CTS-021 / S-CTS-022, CH-09)", () => {
  // Test-local helper: capture the onEvent callback and the
  // subscribe-returned unsubscribe into a closure the test can
  // reach without re-deriving from the mock each time.
  function arrangeLiveStream() {
    let onEventCb: ((ev: unknown) => void) | null = null;
    let unsubscribeCount = 0;
    subscribeTurnMock.mockImplementation(
      (_url: string, onEvent: (ev: unknown) => void) => {
        onEventCb = onEvent;
        return () => {
          unsubscribeCount += 1;
        };
      },
    );
    return {
      fire(ev: unknown) {
        (onEventCb as unknown as (ev: unknown) => void)(ev);
      },
      unsubscribeCount: () => unsubscribeCount,
    };
  }

  it("S-CTS-020: a live tool.call.start mid-turn appends a 'running' tool entry; the assistant stream continues after (no tearing)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_t1", streamUrl: "/api/agent/turns/trn_t1/events" },
    });
    const live = arrangeLiveStream();
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("What time is it?");
    await flush();

    // Drive a message.delta + tool.call.start sequence. The
    // assistant entry is the in-flight one; the tool entry must
    // append AFTER it.
    live.fire({
      kind: "message.delta",
      index: 0,
      delta: "Let me check. ",
    });
    live.fire({
      kind: "tool.call.start",
      wireCallId: "c1",
      tool: "current_time",
      arguments: '{"format":"rfc3339"}',
    });
    await flush();

    // The tool entry is appended after the user + assistant entries.
    const toolEntry = state.entries.find((e) => e.kind === "tool");
    expect(toolEntry?.kind).toBe("tool");
    if (toolEntry?.kind === "tool") {
      expect(toolEntry.state).toBe("running");
      expect(toolEntry.tool).toBe("current_time");
      expect(toolEntry.intent).toBe("current_time");
      // args parsed from the wire-side JSON arguments string
      // (parseArgs at use-chat-stream.ts:114-127).
      expect(toolEntry.args).toEqual([["format", "rfc3339"]]);
      // The entry id is the wire call id bound verbatim — the
      // subsequent tool.result frame matches on it.
      expect(toolEntry.id).toBe("tool-c1");
      // While running, the entry has no result text yet.
      expect(toolEntry.result).toBeUndefined();
    }

    // The assistant entry's text is what was streamed BEFORE the
    // tool entry — both coexist on the wire; no tearing.
    const assistant = [...state.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat");
    expect(assistant?.kind).toBe("said");
    if (assistant?.kind === "said") {
      expect(assistant.text).toBe("Let me check. ");
    }

    // A subsequent message.delta MUST continue to accumulate into
    // the SAME assistant entry (assistantId-keyed delta accumulation
    // at use-chat-stream.ts:202-209). This is the no-tearing
    // guarantee that protects the bubble from splitting at the
    // tool boundary.
    live.fire({
      kind: "message.delta",
      index: 0,
      delta: "It is 07:17.",
    });
    await flush();
    const assistantAfter = [...state.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat");
    expect(assistantAfter?.kind).toBe("said");
    if (assistantAfter?.kind === "said") {
      expect(assistantAfter.text).toBe("Let me check. It is 07:17.");
    }
  });

  it("S-CTS-021: a matching tool.result with outcome 'execution_failure' closes the entry as 'failed'; result carries the typed category, NOT the provider content", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_t2", streamUrl: "/api/agent/turns/trn_t2/events" },
    });
    const live = arrangeLiveStream();
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("What time is it in UTC?");
    await flush();

    live.fire({
      kind: "tool.call.start",
      wireCallId: "c2",
      tool: "current_time",
      arguments: '{"timezone":"UTC"}',
    });
    await flush();
    // After tool.call.start: tool entry is "running".
    const running = state.entries.find((e) => e.kind === "tool");
    expect(running?.kind).toBe("tool");
    if (running?.kind === "tool") {
      expect(running.state).toBe("running");
    }

    // A matching tool.result frame closes the entry. The wire
    // carries BOTH content (provider text — which MUST NOT reach
    // the entry on failure) and failureCategory (typed envelope —
    // which DOES reach the entry). The hook is responsible for
    // collapsing to the typed category only (R-CCP-008 / D6
    // mirror on the live stream).
    live.fire({
      kind: "tool.result",
      wireCallId: "c2",
      tool: "current_time",
      outcome: "execution_failure",
      content: "internal provider text that must NOT leak",
      failureCategory: "invalid_argument",
    });
    await flush();

    const closed = state.entries.find((e) => e.kind === "tool");
    expect(closed?.kind).toBe("tool");
    if (closed?.kind === "tool") {
      expect(closed.state).toBe("failed");
      // Typed category reaches the entry.
      expect(closed.result).toBe("invalid_argument");
      // Provider text MUST NOT leak.
      expect(closed.result).not.toBe(
        "internal provider text that must NOT leak",
      );
      expect(closed.result).not.toContain("provider text");
    }
  });

  it("S-CTS-021 (success variant triangulation): a matching tool.result with outcome 'success' closes the entry as 'done' with content", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_t2b", streamUrl: "/api/agent/turns/trn_t2b/events" },
    });
    const live = arrangeLiveStream();
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("ping");
    await flush();

    live.fire({
      kind: "tool.call.start",
      wireCallId: "c3",
      tool: "current_time",
      arguments: "{}",
    });
    live.fire({
      kind: "tool.result",
      wireCallId: "c3",
      tool: "current_time",
      outcome: "success",
      content: "2026-08-25T07:17:00Z",
      failureCategory: "",
    });
    await flush();

    const closed = state.entries.find((e) => e.kind === "tool");
    expect(closed?.kind).toBe("tool");
    if (closed?.kind === "tool") {
      expect(closed.state).toBe("done");
      expect(closed.result).toBe("2026-08-25T07:17:00Z");
    }
  });

  it("S-CTS-022: turn.end{finishReason: 'tool_calls'} does NOT cancel assistantId-keyed delta accumulation — the next message.delta continues into the same assistant entry", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_t3", streamUrl: "/api/agent/turns/trn_t3/events" },
    });
    const live = arrangeLiveStream();
    const handles: HarnessHandles = { state: { value: null } };
    const { render } = await createDOM();
    await render(<TestHarness handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("ping");
    await flush();

    // First wave of deltas + tool roundtrip + turn.end with
    // finishReason: "tool_calls". The hook's turn.end handler at
    // use-chat-stream.ts:269-281 sets the assistant entry's state
    // to "final" but does NOT close the identity — the original
    // assistantId is still the same.
    live.fire({ kind: "message.delta", index: 0, delta: "first wave. " });
    live.fire({
      kind: "tool.call.start",
      wireCallId: "c4",
      tool: "current_time",
      arguments: "{}",
    });
    live.fire({
      kind: "tool.result",
      wireCallId: "c4",
      tool: "current_time",
      outcome: "success",
      content: "2026-08-25T07:17:00Z",
      failureCategory: "",
    });
    live.fire({
      kind: "turn.end",
      finishReason: "tool_calls",
    });
    await flush();

    // After turn.end: assistant state is "final", the original
    // assistantId is preserved on the entry.
    const assistantMid = [...state.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat");
    expect(assistantMid?.kind).toBe("said");
    if (assistantMid?.kind === "said") {
      expect(assistantMid.state).toBe("final");
      expect(assistantMid.text).toBe("first wave. ");
    }

    // CRITICAL: a subsequent message.delta — the model's
    // tool-result-aware continuation — MUST accumulate into the
    // SAME assistant entry. This is the "by construction"
    // behaviour: delta accumulation at use-chat-stream.ts:202-209
    // is keyed on the assistantId captured in submit()'s closure,
    // NOT gated on turn.end or finishReason. A future contributor
    // adding an early-return on finishReason === "tool_calls"
    // would silently break this — the test below is the trip.
    live.fire({
      kind: "message.delta",
      index: 0,
      delta: "second wave after turn.end.",
    });
    await flush();

    const assistantFinal = [...state.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat");
    expect(assistantFinal?.kind).toBe("said");
    if (assistantFinal?.kind === "said") {
      // Same id, same entry — proves no new bubble was opened.
      expect(assistantFinal.id).toBe(assistantMid?.id);
      // Text accumulated across the turn.end boundary.
      expect(assistantFinal.text).toBe(
        "first wave. second wave after turn.end.",
      );
    }
  });
});

// ---------------------------------------------------------------------------
// CH-11 (S-SCROLL-001) — scrollCounter signal regression.
//
// The previous auto-scroll implementation bound a MutationObserver
// to the transcript <ol>. Qwik's resume / re-render lifecycle can
// swap the <ol> DOM node, orphaning the observer — the result
// was: first mount scrolled correctly (the seeded transcript
// landed at the bottom), but every subsequent entry arrived
// silently below the fold.
//
// The fix: the hook owns a `useSignal(0)` counter (passed in by
// the chat-app) and bumps it on EVERY entry mutation. The
// chat-app's visible-task tracks the counter and scrolls to the
// bottom on every change. The signal is the source of truth —
// no DOM observation, no race with the resume lifecycle.
//
// The four tests below assert the counter advances on every
// mutation path the hook has today: submit() (push user +
// assistant placeholder), message.delta (each frame), reset(),
// and cancel() (finalize the in-flight entry).
// ---------------------------------------------------------------------------

describe("useChatStream scrollCounter (CH-11 S-SCROLL-001)", () => {
  function arrangeLiveStream() {
    let onEventCb: ((ev: unknown) => void) | null = null;
    let unsubscribeCount = 0;
    subscribeTurnMock.mockImplementation(
      (_url: string, onEvent: (ev: unknown) => void) => {
        onEventCb = onEvent;
        return () => {
          unsubscribeCount += 1;
        };
      },
    );
    return {
      fire(ev: unknown) {
        (onEventCb as unknown as (ev: unknown) => void)(ev);
      },
      unsubscribeCount: () => unsubscribeCount,
    };
  }

  it("starts at zero and is exposed via the harness handles", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_init", streamUrl: "/api/agent/turns/trn_init/events" },
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const handles: HarnessHandlesWithCounter = {
      state: { value: null },
      scrollCounter: { value: 0 } as unknown as Signal<number>,
    };
    const { render } = await createDOM();
    await render(<TestHarnessWithCounter handles={handles} />);
    await flush();
    expect(handles.scrollCounter.value).toBe(0);
  });

  it("submit() bumps the counter for both the user entry push AND the assistant placeholder push (CH-11 S-SCROLL-001.a)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_sub", streamUrl: "/api/agent/turns/trn_sub/events" },
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const handles: HarnessHandlesWithCounter = {
      state: { value: null },
      scrollCounter: { value: 0 } as unknown as Signal<number>,
    };
    const { render } = await createDOM();
    await render(<TestHarnessWithCounter handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    const before = handles.scrollCounter.value;
    await state.submit("hello");
    await flush();

    // submit() pushes the user entry first, then the assistant
    // placeholder — two synchronous mutations, two bumps. Without
    // the per-mutation bumps, the chat-app's visible-task would
    // only see one tracked change (or none, depending on Qwik's
    // batch) and the assistant placeholder would land below the
    // fold the moment the user submits.
    const after = handles.scrollCounter.value;
    expect(after - before).toBe(2);
    expect(state.entries.length).toBe(2);
  });

  it("submitTurn resolving to { ok: false } bumps the counter an additional time (the seeded entries are filtered out)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: false,
      kind: "offline",
      message: "Couldn't reach the chat service. Is docker compose up? (network error)",
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const handles: HarnessHandlesWithCounter = {
      state: { value: null },
      scrollCounter: { value: 0 } as unknown as Signal<number>,
    };
    const { render } = await createDOM();
    await render(<TestHarnessWithCounter handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    const before = handles.scrollCounter.value;
    await state.submit("hello");
    await flush();

    // Two pushes (user + assistant placeholder) + one filter on
    // submit-failure = three bumps.
    const after = handles.scrollCounter.value;
    expect(after - before).toBe(3);
    expect(state.entries.length).toBe(0);
  });

  it("each message.delta bumps the counter (N deltas → N bumps), so the auto-scroll follows streamed text frame-by-frame", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_d", streamUrl: "/api/agent/turns/trn_d/events" },
    });
    const live = arrangeLiveStream();
    const handles: HarnessHandlesWithCounter = {
      state: { value: null },
      scrollCounter: { value: 0 } as unknown as Signal<number>,
    };
    const { render } = await createDOM();
    await render(<TestHarnessWithCounter handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    const baseline = handles.scrollCounter.value;

    const N = 5;
    for (let i = 0; i < N; i += 1) {
      live.fire({ kind: "message.delta", index: 0, delta: `d${i} ` });
    }
    await flush();

    // submit() pushed 2 entries (user + assistant placeholder),
    // so the baseline after submit is exactly 2 (when the test
    // harness starts the counter at 0). Each message.delta adds
    // 1 more bump. After N deltas: baseline + N.
    expect(handles.scrollCounter.value).toBe(baseline + N);

    const assistant = [...state.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat");
    expect(assistant?.kind).toBe("said");
    if (assistant?.kind === "said") {
      expect(assistant.text).toBe("d0 d1 d2 d3 d4 ");
    }
  });

  it("reset() bumps the counter once, even when entries.length goes to zero (the reload surface case)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_r", streamUrl: "/api/agent/turns/trn_r/events" },
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const handles: HarnessHandlesWithCounter = {
      state: { value: null },
      scrollCounter: { value: 0 } as unknown as Signal<number>,
    };
    const { render } = await createDOM();
    await render(<TestHarnessWithCounter handles={handles} />);
    await flush();

    const state = handles.state.value as ChatStreamState;
    await state.submit("hello");
    await flush();
    const before = handles.scrollCounter.value;

    // Empty reset — covers the empty-seed fallback path
    // (loadMostRecentConversation returning 200 []).
    await state.reset();
    await flush();
    expect(handles.scrollCounter.value).toBe(before + 1);
    expect(state.entries.length).toBe(0);

    // Non-empty reset — covers the seeded-transcript path. The
    // counter advances once more so the chat-app's visible-task
    // scrolls to the bottom of the seeded <li>s.
    const beforeSeeded = handles.scrollCounter.value;
    await state.reset([
      { kind: "said", id: "u1", who: "you", text: "Hi.", state: "final" },
      { kind: "said", id: "a1", who: "chat", text: "Hello.", state: "final" },
    ]);
    await flush();
    expect(handles.scrollCounter.value).toBe(beforeSeeded + 1);
    expect(state.entries.length).toBe(2);
  });
});

// ---------------------------------------------------------------------------
// CH-12 (S-SCROLL-001) — `chat:scroll-to-bottom` event dispatch.
//
// The previous auto-scroll implementation (CH-11) drove the scroll
// from a Qwik signal counter that the hook bumped on every entry
// mutation. Qwik re-runs the visible-task in the SAME microtask as
// the signal change — BEFORE the render commit — so the rAF inside
// fired before the new <li> was in the DOM. scrollTop = scrollHeight
// read the OLD height and the new content landed below the fold.
//
// CH-12 replaces that with an explicit CustomEvent dispatched from
// the mutation site (`requestScroll` in use-chat-stream.ts). The
// chat-app's visible-task listens for the event in plain browser
// time (no `track`, no reactive race), and runs the scroll inside
// a double rAF so scrollHeight reflects the post-commit DOM.
//
// The three tests below assert the hook dispatches
// `chat:scroll-to-bottom` on `window` from the three primary
// mutation paths: submit(), message.delta, and reset().
//
// Test setup note: Qwik's `createDOM()` returns a test window
// whose `dispatchEvent` is undefined (only `addEventListener`,
// `removeEventListener`, and `CustomEvent` are stubbed). The
// hook calls `window.dispatchEvent(...)` from every mutation
// site; the helper guards with `typeof window.dispatchEvent ===
// "function"` (use-chat-stream.ts:174-177). The test installs a
// real (callable) dispatchEvent before the action runs so
// `vi.spyOn` can wrap it and observe the calls.
// ---------------------------------------------------------------------------

describe("useChatStream chat:scroll-to-bottom event dispatch (CH-12 S-SCROLL-001)", () => {
  // Test-local helper: capture the onEvent callback from the
  // subscribeTurn implementation so the test can dispatch SSE
  // frames by hand. Mirrors the helper in the scrollCounter
  // describe block above.
  function arrangeLiveStream() {
    let onEventCb: ((ev: unknown) => void) | null = null;
    let unsubscribeCount = 0;
    subscribeTurnMock.mockImplementation(
      (_url: string, onEvent: (ev: unknown) => void) => {
        onEventCb = onEvent;
        return () => {
          unsubscribeCount += 1;
        };
      },
    );
    return {
      fire(ev: unknown) {
        (onEventCb as unknown as (ev: unknown) => void)(ev);
      },
      unsubscribeCount: () => unsubscribeCount,
    };
  }

  // Install a real `dispatchEvent` on the test window if it's
  // missing (Qwik's `createDOM()` returns a stub `defaultView`
  // with no dispatchEvent). The Qwik test platform does NOT
  // expose `globalThis.window`, so the hook's
  // `typeof window !== "undefined"` guard would otherwise skip
  // the dispatch entirely (the hook is environment-agnostic —
  // production runs in a real browser, the test env doesn't set
  // `window` globally). Bind `globalThis.window` to the test
  // window's defaultView so the hook sees it, install a real
  // dispatchEvent so `vi.spyOn` can wrap it, and return the
  // spy. The spy records calls even though no listener is
  // registered (Qwik's test window's addEventListener is a
  // no-op — we don't need a real listener for these tests; we
  // just need to observe the dispatch happened).
  //
  // IMPORTANT: each test gets its own defaultView (createDOM()
  // creates one per call), so the helper ALWAYS rebinds
  // globalThis.window to the current test's defaultView. The
  // afterEach below restores globalThis.window to undefined so
  // subsequent test files start clean.
  function installDispatchSpy(
    screen: { ownerDocument: { defaultView: unknown } | null },
  ): ReturnType<typeof vi.spyOn> {
    const win = screen.ownerDocument?.defaultView as
      | (Window & { dispatchEvent?: (ev: Event) => boolean })
      | undefined
    if (!win) throw new Error("no test window from createDOM()")
    // ALWAYS rebind globalThis.window to the current test's
    // defaultView — every test gets its own. Without this, a
    // second test's hook calls dispatchEvent on the FIRST
    // test's window, never reaching the second test's spy.
    ;(globalThis as unknown as { window: unknown }).window = win
    if (typeof win.dispatchEvent !== "function") {
      // Install a real (but listener-agnostic) dispatchEvent so
      // vi.spyOn can wrap it. The defaultView's no-op
      // addEventListener means no listener ever fires, but we
      // don't care — these tests assert the dispatch happened,
      // not that a listener responded.
      ;(win as unknown as { dispatchEvent: (ev: Event) => boolean }).dispatchEvent =
        function dispatchEvent(_ev: Event): boolean {
          return true
        }
    }
    return vi.spyOn(win as Window, "dispatchEvent")
  }

  it("submit() dispatches chat:scroll-to-bottom on window — twice for the user entry + assistant placeholder push", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_evt_sub", streamUrl: "/api/agent/turns/trn_evt_sub/events" },
    })
    subscribeTurnMock.mockReturnValue(() => {})
    const handles: HarnessHandles = { state: { value: null } }
    const { render, screen } = await createDOM()
    await render(<TestHarness handles={handles} />)
    await flush()

    const spy = installDispatchSpy(screen)
    await handles.state.value!.submit("hello")
    await flush()

    // Filter the spy calls to just chat:scroll-to-bottom. The
    // hook dispatches the event from each entry-mutation site
    // (requestScroll at use-chat-stream.ts:173-181); submit()
    // runs two pushes (user entry + assistant placeholder), so
    // we expect at least 2 scroll events. There may be more
    // from later paths in submit(); the strict lower bound is 2
    // (the first two pushes are synchronous inside submit).
    const scrollEvents = spy.mock.calls.filter(
      ([ev]: unknown[]) =>
        (ev as { type?: string } | undefined)?.type === SCROLL_EVENT,
    )
    expect(scrollEvents.length).toBeGreaterThanOrEqual(2)
    expect(handles.state.value!.entries.length).toBe(2)
  })

  it("a faked message.delta SSE chunk dispatches chat:scroll-to-bottom on window", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_evt_delta", streamUrl: "/api/agent/turns/trn_evt_delta/events" },
    })
    const live = arrangeLiveStream()
    const handles: HarnessHandles = { state: { value: null } }
    const { render, screen } = await createDOM()
    await render(<TestHarness handles={handles} />)
    await flush()

    // Install the dispatch spy BEFORE submit so we observe the
    // submit-time pushes too (the test then clears them to
    // isolate the message.delta dispatch below).
    const spy = installDispatchSpy(screen)
    spy.mockClear()

    await handles.state.value!.submit("hello")
    await flush()

    // After submit the spy has at least 2 scroll events (user
    // + assistant placeholder pushes). Clear before firing the
    // SSE chunk so we count only the event from message.delta.
    spy.mockClear()

    live.fire({
      kind: "message.delta",
      index: 0,
      delta: "streaming text ",
    })
    await flush()

    const scrollEvents = spy.mock.calls.filter(
      ([ev]: unknown[]) =>
        (ev as { type?: string } | undefined)?.type === SCROLL_EVENT,
    )
    // One message.delta → one scroll event. The hook's
    // requestScroll is called exactly once per message.delta
    // (use-chat-stream.ts:requestScroll call site for the
    // message.delta case).
    expect(scrollEvents.length).toBe(1)

    const assistant = [...handles.state.value!.entries]
      .reverse()
      .find((e) => e.kind === "said" && e.who === "chat")
    expect(assistant?.kind).toBe("said")
    if (assistant?.kind === "said") {
      expect(assistant.text).toBe("streaming text ")
    }
  })

  it("reset() dispatches chat:scroll-to-bottom on window", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_evt_reset", streamUrl: "/api/agent/turns/trn_evt_reset/events" },
    })
    subscribeTurnMock.mockReturnValue(() => {})
    const handles: HarnessHandles = { state: { value: null } }
    const { render, screen } = await createDOM()
    await render(<TestHarness handles={handles} />)
    await flush()

    // Spy is installed BEFORE reset() so we capture only the
    // reset() dispatches (submit is not called in this test).
    const spy = installDispatchSpy(screen)

    // Empty reset — covers the empty-seed fallback path
    // (loadMostRecentConversation returning 200 []).
    await handles.state.value!.reset()
    await flush()
    let scrollEvents = spy.mock.calls.filter(
      ([ev]: unknown[]) =>
        (ev as { type?: string } | undefined)?.type === SCROLL_EVENT,
    )
    expect(scrollEvents.length).toBe(1)

    // Non-empty reset — covers the seeded-transcript path. The
    // counter (and the event) fires once more so the chat-app's
    // event listener scrolls to the bottom of the seeded <li>s.
    spy.mockClear()
    await handles.state.value!.reset([
      { kind: "said", id: "u1", who: "you", text: "Hi.", state: "final" },
      { kind: "said", id: "a1", who: "chat", text: "Hello.", state: "final" },
    ])
    await flush()
    scrollEvents = spy.mock.calls.filter(
      ([ev]: unknown[]) =>
        (ev as { type?: string } | undefined)?.type === SCROLL_EVENT,
    )
    expect(scrollEvents.length).toBe(1)
    expect(handles.state.value!.entries.length).toBe(2)
  })
});
