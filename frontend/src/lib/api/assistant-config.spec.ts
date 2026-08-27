/**
 * Tests for `assistant-config.ts` — backwards-compatible wrapper around
 * the polymorphic /api/archetypes/{slug}/config/ client
 * (frontend/src/lib/api/archetypes.ts, T-21 PR-2 of
 * cachicamas-archetype-system-foundation).
 *
 * After T-22, `getAssistantConfig` and `putAssistantConfig` route through
 * the polymorphic surface (`/api/archetypes/assistant/...`) and adapt
 * the nested `ArchetypeView` shape back to the flat `ArchetypeConfig`
 * shape ConfigureSection + the agents/* routes consume. The fetch mock
 * therefore returns the polymorphic shape (with an `override` block when
 * a per-org row exists, or without one when the org is on the system
 * default); the test asserts on the flattened legacy shape.
 *
 * Reference: REQ-CACAPI-001 (GET), REQ-CACAPI-002/003 (PUT) of the
 * cachicamas-assistant-configuration-ui change.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { archetypeURL } from "./archetypes";
import type { ArchetypeView } from "./archetypes";
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

// -----------------------------------------------------------------------------
// Polymorphic ArchetypeView fixtures
// -----------------------------------------------------------------------------

function overrideView(overrides: Partial<ArchetypeView> = {}): ArchetypeView {
  // The "per-org override exists" case: the polymorphic view carries
  // the per-org row inside the nested `override` block, and
  // `is_override` is the derived true-on-present boolean.
  return {
    slug: "assistant",
    type: "system",
    display_name: "Assistant",
    tagline: "The colleague everyone talks to first.",
    status: "active",
    archived_at: null,
    created_at: "2026-08-26T15:00:00Z",
    created_by: "seed",
    bundle_version: "v1",
    is_critical: true,
    is_override: true,
    override: {
      system_prompt: "you are a helpful assistant",
      tool_allowlist: ["current_time", "summarize_conversation"],
      defer_tool_names: ["summarize_conversation"],
      model: null,
      version: 3,
      updated_at: "2026-08-26T15:00:00Z",
      updated_by: "user_alice",
    },
    ...overrides,
  };
}

function defaultView(): ArchetypeView {
  // The "system default" case: no per-org override block; the adapter
  // synthesises a flat ArchetypeConfig with is_override=false.
  return {
    slug: "assistant",
    type: "system",
    display_name: "Assistant",
    tagline: "The colleague everyone talks to first.",
    status: "active",
    archived_at: null,
    created_at: "2026-08-26T15:00:00Z",
    created_by: "seed",
    bundle_version: "v1",
    is_critical: true,
    is_override: false,
  };
}

describe("getAssistantConfig", () => {
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns the flattened config from a per-org override view", async () => {
    mockResponseOnce(
      new Response(JSON.stringify(overrideView({ override: { ...overrideView().override!, version: 3 } })), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const result = await getAssistantConfig();
    expect(result.ok).toBe(true);
    if (result.ok) {
      // Flattened contract: prompt/allowlist/version bubble up from the
      // nested override block; is_override is the derived true.
      expect(result.value.version).toBe(3);
      expect(result.value.system_prompt).toBe("you are a helpful assistant");
      expect(result.value.tool_allowlist).toContain("current_time");
      expect(result.value.tool_allowlist).toContain("summarize_conversation");
      expect(result.value.defer_tool_names).toEqual(["summarize_conversation"]);
      expect(result.value.is_override).toBe(true);
      expect(result.value.updated_by).toBe("user_alice");
    }
  });

  it("calls the polymorphic per-slug URL after T-22", async () => {
    mockResponseOnce(
      new Response(JSON.stringify(overrideView()), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await getAssistantConfig();
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypeURL("assistant"),
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
  });

  it("returns the system-default config when no override block is present", async () => {
    mockResponseOnce(
      new Response(JSON.stringify(defaultView()), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const result = await getAssistantConfig();
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value.is_override).toBe(false);
      expect(result.value.system_prompt).toBe("");
      expect(result.value.tool_allowlist).toEqual([]);
      expect(result.value.defer_tool_names).toEqual([]);
      expect(result.value.version).toBe(1);
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

  it("returns the flattened server response (with the bumped version) on 200", async () => {
    const serverView = overrideView({
      override: {
        system_prompt: "you are a helpful assistant",
        tool_allowlist: ["current_time", "summarize_conversation"],
        defer_tool_names: ["summarize_conversation"],
        model: null,
        version: 4, // server bumped
        updated_at: "2026-08-26T16:00:00Z",
        updated_by: "user_alice",
      },
      is_override: true,
    });
    mockResponseOnce(
      new Response(JSON.stringify(serverView), {
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
      expect(result.value.is_override).toBe(true);
    }
  });

  it("calls the polymorphic per-slug config URL with PUT", async () => {
    mockResponseOnce(
      new Response(JSON.stringify(overrideView()), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    await putAssistantConfig({
      system_prompt: "ok",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/archetypes/assistant/config/",
      expect.objectContaining({
        method: "PUT",
        credentials: "include",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
      }),
    );
  });

  it("returns a validation result on 400 with per-field errors", async () => {
    mockResponseOnce(
      new Response(
        JSON.stringify({
          kind: "validation",
          message: "prompt rejected",
          fields: { code: "ERR_PROMPT_TOO_LONG" },
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
        expect(result.fields.code).toBe("ERR_PROMPT_TOO_LONG");
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
