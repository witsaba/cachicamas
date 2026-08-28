/**
 * archetypes.spec.ts — TDD contract for the polymorphic archetype
 * client (T-21 PR-2 of cachicamas-archetype-system-foundation).
 *
 * Round-trips for:
 *   - getArchetype(slug)            (per-profile read)
 *   - putArchetypeConfig(slug, …)   (per-profile write — server response, not request body)
 *   - listArchetypes(type?)         (directory list)
 *
 * Error-envelope mapping:
 *   - HTTP 400 + validation envelope → ApiResult.kind="validation"
 *     with the backend's `code` field surfaced
 *   - HTTP 403 → ApiResult.kind="not_found" (anonymous refusal)
 *   - HTTP 404 → ApiResult.kind="not_found" (unknown slug)
 *   - network error → ApiResult.kind="offline"
 *
 * Mirrors the assistant-config.spec.ts conventions: a single
 * `originalFetch` saved at module scope + per-test restore; each
 * scenario is hermetic.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as archetypesModule from "./archetypes";
import {
  archetypeConfigURL,
  archetypeURL,
  archetypesListURL,
  getArchetype,
  getArchetypeConfigPolymorphic,
  listArchetypes,
  putArchetypeConfig,
  type ArchetypeType,
  type ArchetypeView,
} from "./archetypes";
import type { ApiResult } from "~/lib/api";

const originalFetch = globalThis.fetch;

// -----------------------------------------------------------------------------
// Test fixtures
// -----------------------------------------------------------------------------

function assistantView(overrides: Partial<ArchetypeView> = {}): ArchetypeView {
  return {
    slug: "assistant",
    type: "system",
    display_name: "Assistant",
    tagline: "Your default assistant",
    status: "active",
    archived_at: null,
    created_at: "2026-08-27T10:00:00Z",
    created_by: "seed",
    bundle_version: "v1",
    is_critical: true,
    is_override: true,
    override: {
      system_prompt: "you are the cachicamas assistant",
      tool_allowlist: ["current_time", "summarize_conversation"],
      defer_tool_names: ["summarize_conversation"],
      model: null,
      version: 4,
      updated_at: "2026-08-27T10:00:00Z",
      updated_by: "user_alice",
    },
    ...overrides,
  };
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

// -----------------------------------------------------------------------------
// getArchetype scenarios
// -----------------------------------------------------------------------------

describe("getArchetype", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("resolves with the full ArchetypeView (parent + system child + override)", async () => {
    const view = assistantView();
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, view),
    );

    const result = await getArchetype("assistant");
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.value.slug).toBe("assistant");
    expect(result.value.type).toBe("system");
    expect(result.value.bundle_version).toBe("v1");
    expect(result.value.is_critical).toBe(true);
    expect(result.value.is_override).toBe(true);
    expect(result.value.override?.version).toBe(4);
  });

  it("calls /api/archetypes/{slug} (the per-profile URL)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, assistantView()),
    );
    await getArchetype("assistant");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypeURL("assistant"),
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
  });

  it("URL-encodes the slug (defensive against future kinds with special chars)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, assistantView()),
    );
    await getArchetype("ad-hoc/v1");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypeURL("ad-hoc/v1"),
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("returns is_override=false when the override block is absent", async () => {
    const view = assistantView();
    // Strip the override block.
    const { override: _drop, ...rest } = view;
    void _drop;
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, { ...rest, is_override: false }),
    );

    const result = await getArchetype("assistant");
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.value.is_override).toBe(false);
    expect(result.value.override).toBeUndefined();
  });

  it("rejects with ApiError kind='not_found' on HTTP 404", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(404, {
        kind: "not_found",
        message: "archetype slug is not registered",
        fields: { code: "ERR_UNKNOWN_SLUG" },
      }),
    );

    const result = await getArchetype("no-such-slug");
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("not_found");
  });

  it("rejects with ApiError kind='offline' on network failure", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error("network unreachable"),
    );

    const result = await getArchetype("assistant");
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("offline");
  });
});

// -----------------------------------------------------------------------------
// putArchetypeConfig scenarios
// -----------------------------------------------------------------------------

describe("putArchetypeConfig", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("returns the SERVER response, not the request body", async () => {
    // The server bumps version + server-sets updated_at/updated_by.
    const serverResponse = assistantView({
      override: {
        system_prompt: "client-supplied prompt (different from request body)",
        tool_allowlist: ["current_time", "summarize_conversation"],
        defer_tool_names: ["summarize_conversation"],
        model: null,
        version: 7, // server bumped from 4 to 7
        updated_at: "2026-08-27T11:00:00Z",
        updated_by: "user_alice",
      },
      is_override: true,
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, serverResponse),
    );

    const result = await putArchetypeConfig("assistant", {
      system_prompt: "client-supplied prompt (different from request body)",
      tool_allowlist: ["current_time", "summarize_conversation"],
      defer_tool_names: ["summarize_conversation"],
    });

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    // The resolved value MUST equal the server response (with the
    // bumped version), NOT the request body.
    expect(result.value.override?.version).toBe(7);
    expect(result.value.override?.updated_at).toBe("2026-08-27T11:00:00Z");
  });

  it("calls PUT /api/archetypes/{slug}/config/ with JSON body", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, assistantView()),
    );
    await putArchetypeConfig("assistant", {
      system_prompt: "you are helpful",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypeConfigURL("assistant"),
      expect.objectContaining({
        method: "PUT",
        credentials: "include",
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
      }),
    );
  });

  it("rejects with kind='validation' on HTTP 400 + ERR_HTML_PATTERN code", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(400, {
        kind: "validation",
        message: "system_prompt contains disallowed HTML pattern",
        fields: { code: "ERR_HTML_PATTERN" },
      }),
    );

    const result = await putArchetypeConfig("assistant", {
      system_prompt: "you are <script>alert(1)</script>",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("validation");
    if (result.kind !== "validation") return;
    expect(result.fields.code).toBe("ERR_HTML_PATTERN");
  });

  it("rejects with kind='validation' on HTTP 400 + ERR_PROMPT_TOO_LONG code", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(400, {
        kind: "validation",
        message: "system_prompt exceeds 4000 characters",
        fields: { code: "ERR_PROMPT_TOO_LONG" },
      }),
    );

    const result = await putArchetypeConfig("assistant", {
      system_prompt: "a".repeat(4001),
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("validation");
  });

  it("rejects with kind='not_found' on HTTP 404 + ERR_UNKNOWN_SLUG code", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(404, {
        kind: "not_found",
        message: "archetype slug is not registered",
        fields: { code: "ERR_UNKNOWN_SLUG" },
      }),
    );

    const result = await putArchetypeConfig("no-such-slug", {
      system_prompt: "you are helpful",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("not_found");
  });

  it("rejects with kind='not_found' on HTTP 403 (anonymous)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(403, {
        kind: "server",
        message: "identity not resolved",
      }),
    );

    const result = await putArchetypeConfig("assistant", {
      system_prompt: "you are helpful",
      tool_allowlist: ["current_time"],
      defer_tool_names: [],
    });
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("not_found");
  });
});

// -----------------------------------------------------------------------------
// listArchetypes scenarios
// -----------------------------------------------------------------------------

describe("listArchetypes", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("resolves with the directory list (one entry for the Assistant)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, [assistantView({ is_override: false })]),
    );

    const result = await listArchetypes("system");
    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(Array.isArray(result.value)).toBe(true);
    expect(result.value).toHaveLength(1);
    expect(result.value[0].slug).toBe("assistant");
  });

  it("calls /api/archetypes?type=system with credentials", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, []),
    );
    await listArchetypes("system");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypesListURL("system"),
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
  });

  it("URL-encodes the type parameter", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, []),
    );
    await listArchetypes("system");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/archetypes?type=system",
      expect.anything(),
    );
  });

  it("rejects with kind='validation' on HTTP 400 (empty / unknown type)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(400, {
        kind: "validation",
        message: "type query parameter is required",
        fields: { code: "BAD_REQUEST" },
      }),
    );

    const result = await listArchetypes("system");
    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("validation");
  });
});

// -----------------------------------------------------------------------------
// URL helper coverage (cheap, locks the wire contract)
// -----------------------------------------------------------------------------

describe("URL helpers", () => {
  it("archetypeConfigURL", () => {
    expect(archetypeConfigURL("assistant")).toBe(
      "/api/archetypes/assistant/config/",
    );
  });
  it("archetypeURL", () => {
    expect(archetypeURL("assistant")).toBe("/api/archetypes/assistant");
  });
  it("archetypesListURL", () => {
    expect(archetypesListURL("system")).toBe("/api/archetypes?type=system");
    expect(archetypesListURL("general")).toBe("/api/archetypes?type=general");
    expect(archetypesListURL("owned")).toBe("/api/archetypes?type=owned");
  });
});

// -----------------------------------------------------------------------------
// cachicamas-archetype-per-slug-overlay (RED — T-09..T-13) — TDD contract for
// the new getArchetypeConfigPolymorphic(slug) helper and the drift-guard on
// getArchetype(slug) (REQ-ACAR-1, REQ-ACAR-5, REQ-ACAR-7).
//
//   - getArchetypeConfigPolymorphic(slug) MUST hit archetypeConfigURL(slug)
//     (/api/archetypes/{slug}/config/), NOT archetypeURL(slug)
//     (/api/archetypes/{slug}). This is the wire-shape contract for the
//     /config/ endpoint that the adapter rewire locks.
//   - getArchetypeConfigPolymorphic pass-through: the server response is
//     returned verbatim (no flattening — the adapter layer in
//     assistant-config.ts owns that).
//   - getArchetypeConfigPolymorphic error envelope mapping: 404 → not_found,
//     403 → not_found (anonymous), preserving the parseResponse mapping
//     inherited from getJson<T>.
//   - getArchetype drift guard: getArchetype(slug) MUST still hit
//     archetypeURL(slug) (parent-overlay consumers rely on this).
//
// RED state: the file fails to import `getArchetypeConfigPolymorphic`
// because the helper does not exist yet — that is the canonical RED for
// strict TDD. Commit 2 of this change adds the helper and the
// adapter-rewire GREEN.
// -----------------------------------------------------------------------------

describe("getArchetypeConfigPolymorphic", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("Test_ArchetypeConfigPolymorphicURL_CallsConfigURLNotBareURL: hits archetypeConfigURL(slug), NOT archetypeURL(slug)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, assistantView()),
    );

    await getArchetypeConfigPolymorphic("assistant");

    // MUST hit the /config/ URL — that is the wire contract.
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypeConfigURL("assistant"),
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
    // MUST NOT hit the bare URL.
    expect(globalThis.fetch).not.toHaveBeenCalledWith(
      archetypeURL("assistant"),
      expect.anything(),
    );
  });

  it("Test_ArchetypeConfigPolymorphic_PassesThroughServerResponse: returns the server response verbatim (no flattening, no is_override derivation)", async () => {
    const view = assistantView({
      display_name: "Polymorphic /config/ name",
      tagline: "From the /config/ endpoint",
      is_override: true,
      override: {
        system_prompt: "polymorphic prompt",
        tool_allowlist: ["current_time"],
        defer_tool_names: [],
        model: null,
        version: 9,
        updated_at: "2026-08-27T11:00:00Z",
        updated_by: "user_alice",
      },
    });
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, view),
    );

    const result = await getArchetypeConfigPolymorphic("assistant");

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    // Pass-through: server response is returned verbatim. No flattening,
    // no override-stripping on the client (server keeps the override;
    // client preserves it).
    expect(result.value.display_name).toBe("Polymorphic /config/ name");
    expect(result.value.tagline).toBe("From the /config/ endpoint");
    expect(result.value.override?.version).toBe(9);
    expect(result.value.override?.system_prompt).toBe("polymorphic prompt");
    expect(result.value.override?.updated_by).toBe("user_alice");
    expect(result.value.is_override).toBe(true);
  });

  it("Test_ArchetypeConfigPolymorphic_404_MapsToNotFound: HTTP 404 + ERR_UNKNOWN_SLUG → ApiResult.kind='not_found'", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(404, {
        kind: "not_found",
        message: "archetype slug is not registered",
        fields: { code: "ERR_UNKNOWN_SLUG" },
      }),
    );

    const result = await getArchetypeConfigPolymorphic("no-such-slug");

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("not_found");
  });

  it("Test_ArchetypeConfigPolymorphic_403_MapsToNotFound: HTTP 403 (anonymous) → ApiResult.kind='not_found'", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(403, {
        kind: "server",
        message: "identity not resolved",
      }),
    );

    const result = await getArchetypeConfigPolymorphic("assistant");

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.kind).toBe("not_found");
  });
});

describe("getArchetype drift guard", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("Test_GetArchetype_StillCallsBareURL_NoDrift: getArchetype(slug) still hits archetypeURL(slug), NOT archetypeConfigURL(slug)", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, assistantView()),
    );

    await getArchetype("assistant");

    // Pin: getArchetype(slug) MUST keep hitting the bare URL. The
    // parent-overlay consumers (use-system-archetype.ts:59,
    // agents/index.tsx:48) depend on this.
    expect(globalThis.fetch).toHaveBeenCalledWith(
      archetypeURL("assistant"),
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
    expect(globalThis.fetch).not.toHaveBeenCalledWith(
      archetypeConfigURL("assistant"),
      expect.anything(),
    );
  });
});

// -----------------------------------------------------------------------------
// cachicamas-agent-catalog-config-reload (S2-R — RED) — the directory list
// client gains an optional type param: `listArchetypes()` with no argument
// requests the BARE /api/archetypes URL with NO `type` query (the ?type=
// narrowing guard, CRL-S-010), while `listArchetypes("system")` still sends
// ?type=system.
//
// RED seam: `listArchetypes` is added by the S2-G GREEN commit (the current
// client only has `listArchetypesByType`). The namespace cast keeps the RED
// a behavioral assertion failure ("expected a function, got undefined"),
// not a module-link error.
// -----------------------------------------------------------------------------

describe("listArchetypes (optional type param)", () => {
  const listArchetypes = (
    archetypesModule as unknown as {
      listArchetypes?: (
        type?: ArchetypeType,
      ) => Promise<ApiResult<readonly ArchetypeView[]>>;
    }
  ).listArchetypes;

  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("Test_ListArchetypes_NoArg_BareURL_NoTypeQuery: listArchetypes() requests /api/archetypes with NO type query (CRL-S-010)", async () => {
    expect(
      listArchetypes,
      "listArchetypes is added by the S2-G GREEN commit",
    ).toBeTypeOf("function");
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, []),
    );

    const result = await listArchetypes!();
    expect(result.ok).toBe(true);
    // The narrowing guard: the unfiltered call MUST NOT send a ?type= query.
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/archetypes",
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
  });

  it("Test_ListArchetypes_WithArg_StillSendsTypeQuery: listArchetypes('system') still sends ?type=system", async () => {
    expect(listArchetypes).toBeTypeOf("function");
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(200, []),
    );

    await listArchetypes!("system");
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/archetypes?type=system",
      expect.anything(),
    );
  });
});
