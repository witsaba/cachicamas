/**
 * chat-api.spec.ts — vitest coverage for the chat wire client.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-1..REQ-7.
 *
 * Strict TDD target. This file grows in three slices (T-01, T-02, T-03 of
 * openspec/changes/cachicamas-frontend-chat-layer1/tasks.md):
 *   - T-01: type-only exhaustiveness assertions against
 *           frontend/src/lib/chat-types.ts.
 *   - T-02: a recorded SSE transcript fixture is parsed and the
 *           expected ChatStreamEvent[] typed constant is asserted
 *           against the parsed fixture bytes.
 *   - T-03: FakeES-backed behavioural coverage for submitTurn,
 *           cancelTurn, and subscribeTurn (success, mid-stream
 *           error, offline path, and turn.end closing the stream
 *           exactly once).
 */
import { describe, expect, it } from "vitest";
import {
  assertNever,
  parseTranscript,
  type ChatFinishReason,
  type ChatMessage,
  type ChatSession,
  type ChatStreamError,
  type ChatStreamEvent,
  type ChatTurnError,
  type ChatTurnRequest,
  type ChatTurnResponse,
  type ChatUsage,
  type TranscriptFixture,
} from "./chat-types";

// ---------------------------------------------------------------------------
// T-02: recorded SSE transcript fixture (REQ-1, REQ-7)
//
// The fixture file lives at
//   frontend/src/components/chat/__fixtures__/single-turn.sse
// and pins the wire envelope byte-for-byte (D7). Vitest imports it
// via Vite's `?raw` query string, which returns the literal bytes
// — no transformation, no JSON parsing.
//
// The expectedEvents constant is the typed property-style assertion:
// the spec proves the recorded bytes decode into exactly this
// sequence of ChatStreamEvent values. Any wire-shape drift forces
// the fixture to change FIRST, then the typed constant, then any
// consumer that depends on a removed field.
//
// Why three deltas (not five): tasks.md T-02 says three. Smaller
// fixture = faster spec. The wire envelope (start → delta* → end →
// terminal) is invariant.
// ---------------------------------------------------------------------------

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore — vite `?raw` query is resolved at build time. The
//   fixture file MUST exist before vitest runs; if it doesn't, the
//   import resolves to a missing module and the spec goes red with
//   "Failed to load url ../components/chat/__fixtures__/single-turn.sse".
import singleTurnFixtureRaw from "../components/chat/__fixtures__/single-turn.sse?raw";

export const expectedEvents: readonly ChatStreamEvent[] = [
  { kind: "message.start", messageId: "msg-001", index: 0 },
  { kind: "message.delta", index: 0, delta: "Hello, " },
  { kind: "message.delta", index: 0, delta: "world" },
  { kind: "message.delta", index: 0, delta: "!" },
  {
    kind: "message.end",
    index: 0,
    finishReason: "stop",
  },
  {
    kind: "turn.end",
    finishReason: "stop",
    usage: { inputTokens: 12, outputTokens: 3 },
  },
] as const;

export const singleTurnFixture: TranscriptFixture = {
  name: "single-turn",
  raw: singleTurnFixtureRaw,
  expectedEvents,
  expectTerminalClose: true,
};

// ---------------------------------------------------------------------------
// T-01: type-only exhaustiveness assertions (REQ-1, REQ-4, REQ-6)
//
// The imports above MUST come from ./chat-types. If a future refactor
// moves a type back into chat-api.ts, the spec import path breaks and
// this file goes red — the contract is re-anchored.
//
// `assertNever` is the exhaustiveness probe for ChatStreamEvent. A
// future addition to the union surfaces as a `never`-typed branch
// in the switch, which is a compile-time error (TypeScript
// `Type '...' is not assignable to type 'never'`). At runtime
// the assertion is a no-op; we keep the call inside an `it(...)`
// so vitest counts the contract check.
// ---------------------------------------------------------------------------

describe("chat-types contract surface (REQ-1, REQ-4, REQ-6)", () => {
  it("asserts that every ChatStreamEvent variant is handled (exhaustive)", () => {
    const variants: ChatStreamEvent[] = [
      {
        kind: "message.start",
        messageId: "m-1",
        index: 0,
      },
      {
        kind: "message.delta",
        index: 0,
        delta: "hello",
      },
      {
        kind: "message.end",
        index: 0,
        finishReason: "stop",
      },
      {
        kind: "turn.end",
      },
      {
        kind: "error",
        error: {
          kind: "server",
          message: "boom",
        },
      },
    ];

    const labels: string[] = [];
    for (const ev of variants) {
      switch (ev.kind) {
        case "message.start":
          labels.push(`start:${ev.messageId}:${ev.index}`);
          break;
        case "message.delta":
          labels.push(`delta:${ev.index}:${ev.delta}`);
          break;
        case "message.end":
          labels.push(`end:${ev.index}:${ev.finishReason}`);
          break;
        case "turn.end":
          labels.push("turn:end");
          break;
        case "error":
          labels.push(`error:${ev.error.kind}:${ev.error.message}`);
          break;
        default:
          // assertNever returns never — if a new variant is added
          // to ChatStreamEvent without updating this switch,
          // TypeScript reports a type error on this line.
          assertNever(ev);
      }
    }

    expect(labels).toEqual([
      "start:m-1:0",
      "delta:0:hello",
      "end:0:stop",
      "turn:end",
      "error:server:boom",
    ]);
  });

  it("assertNever narrows an unknown kind to never (rejects new variants at compile time)", () => {
    // Compile-time assertion: this expression compiles, but if a
    // future maintainer adds a new ChatStreamEvent kind, the
    // assignment below becomes invalid and TypeScript fails the
    // build. The runtime expression is a no-op.
    const ev: ChatStreamEvent = {
      kind: "turn.end",
      usage: { inputTokens: 10, outputTokens: 4 },
    };
    if (ev.kind === "turn.end") {
      expect(ev.usage?.inputTokens).toBe(10);
    } else {
      assertNever(ev);
    }
  });

  it("ChatFinishReason is the AI-13.1 seven-value vocabulary", () => {
    const reasons: ChatFinishReason[] = [
      "stop",
      "length",
      "tool_calls",
      "refusal",
      "pause_turn",
      "content_filter",
      "unknown",
    ];
    // The literal set MUST be exactly seven values — adding a
    // new reason requires updating design.md §2 first.
    expect(reasons).toHaveLength(7);
  });

  it("ChatStreamError mirrors the four non-offline ApiError kinds", () => {
    const variants: ChatStreamError[] = [
      { kind: "validation", message: "v", fields: { a: "b" } },
      { kind: "conflict", message: "c" },
      { kind: "not_found", message: "n" },
      { kind: "server", message: "s" },
    ];
    expect(variants.map((v) => v.kind)).toEqual([
      "validation",
      "conflict",
      "not_found",
      "server",
    ]);
  });

  it("ChatTurnError mirrors the five ApiError kinds", () => {
    const variants: ChatTurnError[] = [
      { kind: "offline", message: "o" },
      { kind: "validation", message: "v", fields: { a: "b" } },
      { kind: "conflict", message: "c" },
      { kind: "not_found", message: "n" },
      { kind: "server", message: "s" },
    ];
    expect(variants.map((v) => v.kind)).toEqual([
      "offline",
      "validation",
      "conflict",
      "not_found",
      "server",
    ]);
  });

  it("ChatTurnRequest has id + prompt + optional systemHint", () => {
    const a: ChatTurnRequest = { id: "trn_1", prompt: "hello" };
    const b: ChatTurnRequest = {
      id: "trn_2",
      prompt: "hello",
      systemHint: "you are a chat assistant",
    };
    expect(a.id).toBe("trn_1");
    expect(b.systemHint).toBe("you are a chat assistant");
  });

  it("ChatTurnResponse has turnId + streamUrl", () => {
    const r: ChatTurnResponse = {
      turnId: "trn_1",
      streamUrl: "/api/agent/turns/trn_1/events",
    };
    expect(r.turnId).toBe("trn_1");
    expect(r.streamUrl).toContain("trn_1/events");
  });

  it("ChatUsage counts input + output tokens (cost-display deferred per design §2)", () => {
    const u: ChatUsage = { inputTokens: 12, outputTokens: 30 };
    expect(u.inputTokens).toBe(12);
    expect(u.outputTokens).toBe(30);
  });

  it("ChatMessage text is mutable (useSignal<string>) and status has 4 states", () => {
    const m: ChatMessage = {
      id: "m-1",
      role: "assistant",
      text: "draft",
      status: "pending",
    };
    // text is mutable by design (delta appends in place)
    m.text = "draft v2";
    expect(m.text).toBe("draft v2");
    // status vocabulary is locked
    const statuses: ChatMessage["status"][] = [
      "pending",
      "streaming",
      "complete",
      "error",
    ];
    expect(statuses).toHaveLength(4);
  });

  it("ChatSession has messages array, status, optional currentTurnId", () => {
    const s: ChatSession = {
      messages: [],
      status: "idle",
    };
    expect(s.messages).toEqual([]);
    expect(s.status).toBe("idle");
    // currentTurnId is optional
    const s2: ChatSession = {
      messages: [],
      status: "submitting",
      currentTurnId: "trn_x",
    };
    expect(s2.currentTurnId).toBe("trn_x");
  });

  it("TranscriptFixture pins raw bytes + expected events + terminal-close flag", () => {
    const fixture: TranscriptFixture = {
      name: "single-turn",
      raw: "event: message.start\ndata: {\"messageId\":\"m\",\"index\":0}\n\n",
      expectedEvents: [
        { kind: "message.start", messageId: "m", index: 0 },
      ],
      expectTerminalClose: true,
    };
    expect(fixture.raw).toContain("message.start");
    expect(fixture.expectedEvents).toHaveLength(1);
    expect(fixture.expectTerminalClose).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// T-02 slice — recorded SSE transcript
// ---------------------------------------------------------------------------

describe("chat recorded SSE transcript (REQ-1, REQ-7)", () => {
  it("the recorded fixture parses to at least one event", () => {
    // This is the RED step: the fixture file must exist and be
    // parseable. The test will fail with a module-resolution error
    // until the fixture is in place.
    const parsed = parseTranscript(singleTurnFixtureRaw);
    expect(parsed.length).toBeGreaterThan(0);
  });

  it("the recorded fixture parses to exactly the typed expectedEvents (REQ-7 property-style)", () => {
    const parsed = parseTranscript(singleTurnFixtureRaw);
    // Deep-assert: every variant's discriminator + payload matches
    // the typed expected list. A future wire change MUST update the
    // fixture AND expectedEvents together.
    expect(parsed).toEqual(expectedEvents);
  });

  it("the fixture's raw bytes carry `event:` names (D7 wire envelope)", () => {
    // Pin the wire shape: SSE frames MUST use the `event:` field
    // for typed dispatch (mirrors design.md §3 D1 refinement).
    expect(singleTurnFixtureRaw).toContain("event: message.start");
    expect(singleTurnFixtureRaw).toContain("event: message.delta");
    expect(singleTurnFixtureRaw).toContain("event: message.end");
    expect(singleTurnFixtureRaw).toContain("event: turn.end");
  });

  it("the fixture ends with a `turn.end` terminal event (REQ-1 S-1.a close-on-terminal)", () => {
    const parsed = parseTranscript(singleTurnFixtureRaw);
    const last = parsed[parsed.length - 1];
    expect(last?.kind).toBe("turn.end");
  });

  it("parseTranscript drops unknown event names (REQ-1 S-1.b)", () => {
    const raw =
      "event: message.start\ndata: {\"messageId\":\"m\",\"index\":0}\n\n" +
      "event: ping\ndata: {}\n\n" +
      "event: message.end\ndata: {\"index\":0,\"finishReason\":\"stop\"}\n\n";
    const parsed = parseTranscript(raw);
    expect(parsed.map((e) => e.kind)).toEqual([
      "message.start",
      "message.end",
    ]);
  });

  it("parseTranscript drops frames with malformed JSON data (defensive parity with parseSSEResponse)", () => {
    const raw =
      "event: message.start\ndata: {not json\n\n" +
      "event: message.end\ndata: {\"index\":0,\"finishReason\":\"stop\"}\n\n";
    const parsed = parseTranscript(raw);
    expect(parsed.map((e) => e.kind)).toEqual(["message.end"]);
  });
});

// ---------------------------------------------------------------------------
// T-03 slice — typed SSE client (REQ-1, REQ-2, REQ-4, REQ-5)
//
// The chat's wire client. Three responsibilities:
//   1. submitTurn() POSTs /api/agent/turns with the typed body and
//      resolves to a typed ChatTurnResponse. Network errors are
//      mapped to { kind: "offline", message: <literal> } (REQ-5).
//   2. cancelTurn() DELETEs /api/agent/turns/:id with
//      X-Requested-With: XMLHttpRequest (state-changing fetch) and
//      keepalive: true (REQ-2 — fire-and-forget cancel).
//   3. subscribeTurn() opens an EventSource on the returned stream
//      URL with withCredentials: true and dispatches typed events
//      via addEventListener('message.<x>', ...) per design §3 D1
//      refinement. turn.end closes the EventSource exactly once
//      (REQ-1 S-1.a). Mid-stream `event: error` surfaces as a
//      ChatStreamError and closes the stream (REQ-4 S-4.a).
//
// EventSource is mocked with the FakeES pattern from
// chat-api.spec.ts:394-425 inline (capturing onmessage / onerror
// / close). The earlier lib/api.sse.spec.ts:75-307 reference was
// stale — the FakeES is now inline here per explore §8.1.
// fetch is mocked via vi.stubGlobal.
// ---------------------------------------------------------------------------

import { afterEach, beforeEach, vi } from "vitest";

describe("chat-api wire client (REQ-1, REQ-2, REQ-4, REQ-5)", () => {
  let originalEventSource: typeof EventSource | undefined;
  let lastCreated: FakeES | null = null;

  // FakeES mirrors api.sse.spec.ts:79-93. Captures addEventListener
  // calls so subscribeTurn can dispatch typed events; mirrors the
  // typed `EventSource.addEventListener('message.<x>', ...)` pattern
  // that the runtime path uses (design §3 D1 refinement).
  class FakeES {
    url: string;
    withCredentials = false;
    listeners: Record<string, Array<(ev: MessageEvent<string>) => void>> = {};
    closed = false;
    constructor(url: string, opts?: { withCredentials?: boolean }) {
      this.url = url;
      this.withCredentials = opts?.withCredentials ?? false;
      lastCreated = this;
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
    // Test-only helper to dispatch a typed event the way the browser would.
    fire(name: string, data: unknown) {
      const arr = this.listeners[name];
      if (!arr) return;
      for (const cb of arr) {
        cb({ data: JSON.stringify(data) } as MessageEvent<string>);
      }
    }
  }

  beforeEach(() => {
    originalEventSource = (globalThis as { EventSource?: typeof EventSource })
      .EventSource;
    (globalThis as { EventSource?: typeof EventSource }).EventSource =
      FakeES as unknown as typeof EventSource;
    lastCreated = null;
  });

  afterEach(() => {
    if (originalEventSource === undefined) {
      delete (globalThis as { EventSource?: typeof EventSource }).EventSource;
    } else {
      (globalThis as { EventSource?: typeof EventSource }).EventSource =
        originalEventSource;
    }
    lastCreated = null;
    vi.restoreAllMocks();
  });

  it("submitTurn issues POST /api/agent/turns with JSON body + CSRF header (REQ-1, REQ-2)", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(JSON.stringify({ turnId: "trn_1", streamUrl: "/api/agent/turns/trn_1/events" }), {
        status: 202,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const { submitTurn } = await import("./chat-api");

    const result = await submitTurn({ id: "trn_1", prompt: "hello" });
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value.turnId).toBe("trn_1");
      expect(result.value.streamUrl).toBe("/api/agent/turns/trn_1/events");
    }
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toContain("/api/agent/turns");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({ id: "trn_1", prompt: "hello" });
    // stateChangingFetch wraps headers in a Headers instance, so use .get().
    const headers = init.headers as Headers;
    expect(headers.get("X-Requested-With")).toBe("XMLHttpRequest");
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("submitTurn maps network errors to kind=offline with the amended dev-honest phrase (REQ-5 S-5.a amended)", async () => {
    // The amended REQ-5 (CH-05.2) replaces the prior literal phrase
    // with a generic dev-honest phrase. The vitest spec asserts the
    // new substring so a future refactor that drops or rewrites the
    // message goes red.
    const fetchMock = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    });
    vi.stubGlobal("fetch", fetchMock);
    const { submitTurn } = await import("./chat-api");

    const result = await submitTurn({ id: "trn_1", prompt: "hello" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("offline");
      expect(result.message).toContain(
        "Couldn't reach the chat service. Is docker compose up?",
      );
    }
  });

  it("submitTurn maps a 400 validation response to kind=validation with the typed fields (REQ-4 S-4.b)", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({
          error: "validation",
          message: "Invalid request",
          fields: { prompt: "must not be empty" },
        }),
        { status: 400, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const { submitTurn } = await import("./chat-api");

    const result = await submitTurn({ id: "trn_1", prompt: "" });
    expect(result.ok).toBe(false);
    if (!result.ok && result.kind === "validation") {
      expect(result.fields.prompt).toBe("must not be empty");
    } else {
      throw new Error("expected validation result");
    }
  });

  it("submitTurn maps a 409 response to kind=conflict (REQ-4 S-4.c)", async () => {
    const fetchMock = vi.fn(async () =>
      new Response(
        JSON.stringify({ error: "conflict", message: "Turn already in flight" }),
        { status: 409, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const { submitTurn } = await import("./chat-api");

    const result = await submitTurn({ id: "trn_1", prompt: "hello" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("conflict");
      expect(result.message).toBe("Turn already in flight");
    }
  });

  it("cancelTurn issues DELETE /api/agent/turns/:id with keepalive + CSRF (REQ-2)", async () => {
    const fetchMock = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    const { cancelTurn } = await import("./chat-api");

    const result = await cancelTurn({ id: "trn_1" });
    expect(result.ok).toBe(true);
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toContain("/api/agent/turns/trn_1");
    expect(url.endsWith("/trn_1")).toBe(true);
    expect(init.method).toBe("DELETE");
    expect((init as RequestInit & { keepalive?: boolean }).keepalive).toBe(true);
    const headers = init.headers as Headers;
    expect(headers.get("X-Requested-With")).toBe("XMLHttpRequest");
  });

  it("cancelTurn maps network errors to kind=offline (defensive)", async () => {
    const fetchMock = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    });
    vi.stubGlobal("fetch", fetchMock);
    const { cancelTurn } = await import("./chat-api");

    const result = await cancelTurn({ id: "trn_1" });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("offline");
    }
  });

  it("subscribeTurn opens an EventSource on the stream URL with withCredentials (REQ-1)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const events: ChatStreamEvent[] = [];
    const unsubscribe = subscribeTurn(
      "/api/agent/turns/trn_1/events",
      (ev) => events.push(ev),
    );
    expect(lastCreated).not.toBeNull();
    expect(lastCreated!.url).toContain("/api/agent/turns/trn_1/events");
    expect(lastCreated!.withCredentials).toBe(true);
    unsubscribe();
  });

  it("subscribeTurn dispatches typed message.start/delta/end/turn.end in order (REQ-1 S-1.a)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const events: ChatStreamEvent[] = [];
    const unsubscribe = subscribeTurn(
      "/api/agent/turns/trn_1/events",
      (ev) => events.push(ev),
    );
    // Fire the canonical transcript frames.
    lastCreated!.fire("message.start", { messageId: "m-1", index: 0 });
    lastCreated!.fire("message.delta", { index: 0, delta: "Hello" });
    lastCreated!.fire("message.end", { index: 0, finishReason: "stop" });
    lastCreated!.fire("turn.end", { finishReason: "stop" });
    expect(events).toEqual([
      { kind: "message.start", messageId: "m-1", index: 0 },
      { kind: "message.delta", index: 0, delta: "Hello" },
      { kind: "message.end", index: 0, finishReason: "stop" },
      { kind: "turn.end", finishReason: "stop" },
    ]);
    // turn.end closes the EventSource exactly once.
    expect(lastCreated!.closed).toBe(true);
    unsubscribe();
  });

  it("subscribeTurn closes the EventSource exactly once on turn.end (REQ-1 S-1.a)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const events: ChatStreamEvent[] = [];
    const unsubscribe = subscribeTurn(
      "/api/agent/turns/trn_1/events",
      (ev) => events.push(ev),
    );
    lastCreated!.fire("turn.end", {});
    expect(lastCreated!.closed).toBe(true);
    // A second turn.end after close is a no-op for the consumer.
    lastCreated!.fire("turn.end", {});
    expect(events).toHaveLength(1);
    unsubscribe();
  });

  it("subscribeTurn surfaces a mid-stream error event as the typed ChatStreamError (REQ-4 S-4.a)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const events: ChatStreamEvent[] = [];
    const unsubscribe = subscribeTurn(
      "/api/agent/turns/trn_1/events",
      (ev) => events.push(ev),
    );
    lastCreated!.fire("error", {
      kind: "server",
      message: "upstream unavailable",
    });
    expect(events).toEqual([
      {
        kind: "error",
        error: { kind: "server", message: "upstream unavailable" },
      },
    ]);
    unsubscribe();
  });

  it("subscribeTurn drops frames with an unknown event name (REQ-1 S-1.b)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const events: ChatStreamEvent[] = [];
    const unsubscribe = subscribeTurn(
      "/api/agent/turns/trn_1/events",
      (ev) => events.push(ev),
    );
    lastCreated!.fire("ping", {});
    lastCreated!.fire("message.start", { messageId: "m-1", index: 0 });
    expect(events.map((e) => e.kind)).toEqual(["message.start"]);
    unsubscribe();
  });

  it("subscribeTurn treats EventSource onerror before any message as offline (REQ-5 S-5.b amended)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const errors: string[] = [];
    const unsubscribe = subscribeTurn(
      "/api/agent/turns/trn_1/events",
      () => {
        /* no events expected */
      },
      (msg) => errors.push(msg),
    );
    // Simulate the EventSource erroring without any prior message.
    (lastCreated!.listeners["error"] ?? []).forEach((cb) =>
      cb({} as MessageEvent<string>),
    );
    // The amended offline surface from the runtime path: the
    // dev-honest phrase, not the retired literal.
    expect(errors.length).toBeGreaterThan(0);
    expect(errors[0]).toContain(
      "Couldn't reach the chat service. Is docker compose up?",
    );
    unsubscribe();
  });

  it("subscribeTurn's unsubscribe closes the EventSource (REQ-1)", async () => {
    const { subscribeTurn } = await import("./chat-api");
    const unsubscribe = subscribeTurn("/api/agent/turns/trn_1/events", () => {});
    unsubscribe();
    expect(lastCreated!.closed).toBe(true);
  });
});
