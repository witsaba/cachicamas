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
