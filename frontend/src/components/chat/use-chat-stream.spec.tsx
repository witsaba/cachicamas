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
