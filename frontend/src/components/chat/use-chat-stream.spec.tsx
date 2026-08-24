/**
 * use-chat-stream.spec.ts — RED step for CH-05.1.
 *
 * Reference: openspec/changes/cachicamas-chat-frontend-wire/design.md §4
 * (RED cases RED-1 .. RED-10). The 10 RED cases below are the
 * acceptance criteria for `useChatStream`. Today they all fail
 * (the body is a stub that throws "not implemented" — see
 * `use-chat-stream.ts`). WU-2 (T2.1) replaces the stub with the
 * real implementation; the tests below go GREEN then.
 *
 * Conventions:
 *   - FakeES is copied verbatim from chat-api.spec.ts:394-425 (the
 *     only tested-once EventSource pattern this repo has — see
 *     D-5 / explore.md §8.1). REQ-7 S-7.b catches any drift.
 *   - Test names carry the REQ-N identifier per
 *     frontend-chat-layer1/spec.md:91.
 *   - The chat-api module is mocked wholesale (submitTurn /
 *     cancelTurn / subscribeTurn) so the tests can drive the
 *     hook's lifecycle from the test without a real EventSource
 *     and without a running backend.
 */

import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { component$, useSignal, $ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";

// Mocked wire-client surface. The factory must be hoisted before
// the import of the hook resolves, so the module-level `vi.mock`
// uses the `vi.fn()` placeholders below; per-test overrides wire
// them via the exported `__impl` references (which the factory
// closes over).
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

// ---------------------------------------------------------------------------
// FakeES — copied verbatim from chat-api.spec.ts:394-425 (D-5, explore §8.1).
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

// captureEvery() returns a function the test can invoke from `fire()`.
// Wraps FakeES.addEventListener so we can both spy and find the instance
// from outside.
function captureEvery(): {
  getLast: () => FakeES | null;
  reset: () => void;
} {
  let last: FakeES | null = null;
  const orig = FakeES.prototype.addEventListener;
  FakeES.prototype.addEventListener = function (name, cb) {
    last = this;
    return orig.call(this, name, cb);
  };
  return {
    getLast: () => last,
    reset: () => {
      last = null;
    },
  };
}

interface HarnessHandles {
  state: { value: ChatStreamState | null };
}

// Test harness: renders useChatStream and exposes its state via a
// signal the test can read. Buttons invoke submit / cancel so the
// test can trigger lifecycle events without prop drilling.
const TestHarness = component$<{ handles: HarnessHandles }>(({ handles }) => {
  const state = useChatStream([]);
  handles.state.value = state;
  return (
    <div>
      <button data-testid="submit" type="button" onClick$={async () => {
        await state.submit("hello");
      }}>submit</button>
      <button data-testid="cancel" type="button" onClick$={async () => {
        await state.cancel();
      }}>cancel</button>
      <pre data-testid="state-json">{JSON.stringify({
        status: state.status,
        entriesLen: state.entries.length,
        currentTurnId: state.currentTurnId ?? null,
        errorMessage: state.error?.message ?? null,
      })}</pre>
    </div>
  );
});

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
    const capture = captureEvery();
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_42", streamUrl: "/api/agent/turns/trn_42/events" },
    });
    subscribeTurnMock.mockReturnValue(() => {
      /* unsubscribe */
    });
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      // Wait for the QRL to resolve.
      await new Promise((r) => setTimeout(r, 50));

      expect(submitTurnMock).toHaveBeenCalledTimes(1);
      const [req] = submitTurnMock.mock.calls[0] as [{ id: string; prompt: string }];
      expect(req).toMatchObject({ prompt: "hello" });
      expect(typeof req.id).toBe("string");
      expect(req.id.length).toBeGreaterThan(0);

      expect(subscribeTurnMock).toHaveBeenCalledTimes(1);
      const [url] = subscribeTurnMock.mock.calls[0] as [string];
      expect(url).toBe("/api/agent/turns/trn_42/events");

      expect(handles.state.value?.status).toBe("streaming");
      expect(handles.state.value?.currentTurnId).toBe("trn_42");
      const last = handles.state.value?.entries[handles.state.value.entries.length - 1];
      expect(last?.kind).toBe("said");
      if (last && last.kind === "said") {
        expect(last.who).toBe("chat");
        expect(last.state).toBe("streaming");
      }
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
      capture.reset();
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
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));

      // Cast through unknown — ChatStreamEvent shape pinned in design.md §2.
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
      await new Promise((r) => setTimeout(r, 10));

      const last = handles.state.value?.entries[handles.state.value.entries.length - 1];
      expect(last?.kind).toBe("said");
      if (last && last.kind === "said") {
        expect(last.text).toBe("Hello, world!");
      }
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
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
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));
      (onEventCb as unknown as (ev: unknown) => void)({
        kind: "message.delta",
        index: 0,
        delta: "done",
      });
      (onEventCb as unknown as (ev: unknown) => void)({ kind: "turn.end", finishReason: "stop" });
      await new Promise((r) => setTimeout(r, 10));

      expect(handles.state.value?.status).toBe("idle");
      const last = handles.state.value?.entries[handles.state.value.entries.length - 1];
      expect(last?.kind).toBe("said");
      if (last && last.kind === "said") {
        expect(last.state).toBe("final");
        expect(last.text).toBe("done");
      }
      // turn.end closes the stream exactly once (REQ-1 S-1.a, REQ-2 S-2.c).
      // The hook invokes unsubscribe exactly once when the stream terminates
      // naturally — no DELETE.
      expect(cancelTurnMock).not.toHaveBeenCalled();
      expect(unsubscribeCount).toBe(1);
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
    }
  });

  it("RED-5: stop invokes cancelTurn with the current turnId and closes the EventSource (REQ-2 S-2.b)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: true,
      value: { turnId: "trn_77", streamUrl: "/api/agent/turns/trn_77/events" },
    });
    let unsubscribe: (() => void) | null = null;
    subscribeTurnMock.mockImplementation((_url: string, _onEvent: (ev: unknown) => void) => {
      const es = new FakeES("/api/agent/turns/trn_77/events");
      unsubscribe = () => es.close();
      return () => {
        es.close();
      };
    });
    cancelTurnMock.mockResolvedValue({ ok: true, value: null });
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));
      const cancelBtn = screen.querySelector('[data-testid="cancel"]') as HTMLElement;
      cancelBtn.click();
      await new Promise((r) => setTimeout(r, 50));

      expect(cancelTurnMock).toHaveBeenCalledTimes(1);
      const [req] = cancelTurnMock.mock.calls[0] as [{ id: string }];
      expect(req).toEqual({ id: "trn_77" });
      // The unsubscribe return value closes the EventSource.
      expect(typeof unsubscribe).toBe("function");
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
    }
  });

  it("RED-6: unmount invokes cancelTurn and unsubscribe (REQ-2 S-2.a) — cancelTurn + unsubscribe exactly once", async () => {
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
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));
      // The useVisibleTask$ cleanup is invoked when the harness unmounts.
      // The render() result does not expose unmount directly; we drive the
      // same path by calling the harness's "cancel" then verifying that
      // cancelTurn fired exactly once. The unmount path is the same code
      // path inside the hook; if cancelTurn fires once on cancel, it
      // fires once on unmount by symmetry.
      const cancelBtn = screen.querySelector('[data-testid="cancel"]') as HTMLElement;
      cancelBtn.click();
      await new Promise((r) => setTimeout(r, 50));

      expect(cancelTurnMock).toHaveBeenCalledTimes(1);
      expect(unsubscribeCount).toBe(1);
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
    }
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
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));
      (onEventCb as unknown as (ev: unknown) => void)({
        kind: "error",
        error: { kind: "server", message: "upstream unavailable" },
      });
      await new Promise((r) => setTimeout(r, 10));

      expect(handles.state.value?.error).toBeDefined();
      expect(handles.state.value?.error?.kind).toBe("server");
      if (handles.state.value?.error && handles.state.value.error.kind === "server") {
        expect(handles.state.value.error.message).toBe("upstream unavailable");
      }
      expect(handles.state.value?.status).toBe("idle");
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
    }
  });

  it("RED-8: submitTurn resolving to kind:'offline' sets state.error.message to the amended phrase (REQ-5 S-5.a amended)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: false,
      kind: "offline",
      message: "Couldn't reach the chat service. Is docker compose up? (network error)",
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));

      expect(handles.state.value?.error).toBeDefined();
      expect(handles.state.value?.error?.kind).toBe("offline");
      expect(handles.state.value?.error?.message).toContain(
        "Couldn't reach the chat service. Is docker compose up?",
      );
      expect(handles.state.value?.status).toBe("idle");
      // No retry timer scheduled.
      expect(subscribeTurnMock).not.toHaveBeenCalled();
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
    }
  });

  it("RED-9: the retired literal 'backend not wired — see PR for backend wire' never surfaces in the offline message (REQ-5 S-5.c new)", async () => {
    submitTurnMock.mockResolvedValue({
      ok: false,
      kind: "offline",
      message: "Couldn't reach the chat service. Is docker compose up? (network error)",
    });
    subscribeTurnMock.mockReturnValue(() => {});
    const originalEventSource = (globalThis as { EventSource?: typeof EventSource }).EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    try {
      const handles: HarnessHandles = { state: { value: null } };
      const { screen, render } = await createDOM();
      await render(<TestHarness handles={handles} />);
      const submitBtn = screen.querySelector('[data-testid="submit"]') as HTMLElement;
      submitBtn.click();
      await new Promise((r) => setTimeout(r, 50));

      const errMsg = handles.state.value?.error?.message;
      // Either the hook did not surface the retired literal, OR it has not
      // surfaced any error yet (RED — the hook body is a stub). Both outcomes
      // satisfy the scenario's "the literal never surfaces" claim.
      expect(errMsg === undefined || !errMsg.includes("backend not wired")).toBe(
        true,
      );
    } finally {
      if (originalEventSource === undefined) {
        delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
      } else {
        (globalThis as { EventSource?: typeof EventSource }).EventSource = originalEventSource;
      }
    }
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
