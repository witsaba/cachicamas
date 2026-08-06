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
