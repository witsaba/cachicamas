/**
 * Tests for `assistant-config.ts` — typed client for the
 * /api/chat/assistant/config endpoint.
 *
 * Reference: REQ-CACAPI-001 (GET), REQ-CACAPI-002/003 (PUT).
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  getAssistantConfig,
  putAssistantConfig,
} from "./assistant-config";

const originalFetch = globalThis.fetch;

function mockResponseOnce(response: Response): void {
  const fetchMock = vi.fn(async () => response);
  globalThis.fetch = fetchMock as unknown as typeof fetch;
}

function mockFetchError(message: string): void {
  const fetchMock = vi.fn(async () => {
    throw new Error(message);
  });
  globalThis.fetch = fetchMock as unknown as typeof fetch;
}

const sampleConfig = {
  kind: "chat",
  org_id: "user_alice",
  system_prompt: "you are a helpful assistant",
  tool_allowlist: ["current_time", "summarize_conversation"],
  defer_tool_names: ["summarize_conversation"],
  model: null,
  version: 3,
  updated_by: "user_alice",
  updated_at: "2026-08-26T15:00:00Z",
};

describe("getAssistantConfig", () => {
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns the persisted config on 200", async () => {
    mockResponseOnce(
      new Response(JSON.stringify(sampleConfig), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const result = await getAssistantConfig();
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value.version).toBe(3);
      expect(result.value.system_prompt).toBe("you are a helpful assistant");
      expect(result.value.tool_allowlist).toContain("current_time");
    }
  });

  it("returns a not_found result on 403 (anonymous)", async () => {
    mockResponseOnce(
      new Response(
        JSON.stringify({ kind: "server", message: "identity not resolved" }),
        { status: 403, headers: { "Content-Type": "application/json" } },
      ),
    );
    const result = await getAssistantConfig();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("not_found");
    }
  });

  it("returns an offline result on network failure", async () => {
    mockFetchError("connection refused");
    const result = await getAssistantConfig();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("offline");
    }
  });
});

describe("putAssistantConfig", () => {
  beforeEach(() => {
    globalThis.fetch = originalFetch;
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns the persisted config on 200", async () => {
    mockResponseOnce(
      new Response(JSON.stringify({ ...sampleConfig, version: 4 }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const result = await putAssistantConfig({
      system_prompt: "updated prompt",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value.version).toBe(4);
    }
  });

  it("returns a validation result on 400 with per-field errors", async () => {
    mockResponseOnce(
      new Response(
        JSON.stringify({
          kind: "validation",
          message: "prompt rejected",
          fields: { system_prompt: "exceeds maximum length" },
        }),
        { status: 400, headers: { "Content-Type": "application/json" } },
      ),
    );

    const result = await putAssistantConfig({
      system_prompt: "x".repeat(5000),
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("validation");
      if (result.kind === "validation") {
        expect(result.fields.system_prompt).toBe("exceeds maximum length");
      }
    }
  });

  it("returns an offline result on network failure", async () => {
    mockFetchError("ETIMEDOUT");
    const result = await putAssistantConfig({
      system_prompt: "ok",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("offline");
    }
  });
});
