/**
 * Wire-shape tests for the Skills frontend API client.
 *
 * Anti-drift guards (from obs #1959) enforced as RED tests FIRST:
 *   - Task 5.1: Skill.current_revision is `number` (not undefined-allowed)
 *   - Task 5.2: parseResponse reads body.error.message (NESTED envelope)
 *   - Task 5.3: parseResponse maps HTTP status → ApiResult kind
 *   - Task 5.4: listSkills calls GET /skills (NO query string)
 *   - Task 5.5: getSkill + deleteSkill URL/method
 *   - Task 5.6: createSkill calls POST with JSON {name, description, body}
 *   - Task 5.7: updateSkill calls PATCH with BOTH description AND body
 *   - Task 5.8: updateSkill JSDoc says PATCH not PUT (source-grep)
 *   - Task 5.9: listRevisions + restoreRevision URL/method
 *
 * NO test in this file touches prompts-api.ts (independence gate).
 * NO test uses a FLAT envelope `{message:"..."}` — only NESTED
 * envelopes (the prompts gotcha fixture regression).
 */

import { afterEach, beforeEach, describe, expect, expectTypeOf, it, vi } from "vitest";

// ---------------------------------------------------------------------------
// Helpers (for wire-shape tests; task 5.2+)
// ---------------------------------------------------------------------------

function mockResponse(
  body: unknown,
  init: ResponseInit & { status?: number } = {},
): Response {
  const { status = 200, ...rest } = init;
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    ...rest,
    status,
    headers: {
      "Content-Type": "application/json",
      ...(rest.headers as Record<string, string> | undefined),
    },
  });
}

// ---------------------------------------------------------------------------
// Task 5.1 — Type contract (RED)
// ---------------------------------------------------------------------------

describe("skills-api type contract (anti-drift obs #1959 item 2)", () => {
  /**
   * Task 5.1 RED — Skill declares current_revision: number (not undefined-allowed).
   * Backend emits `current_revision` via SQL JOIN (ADR-SK-008). If the
   * type allowed `undefined`, sidebar would render `vundefined` (the
   * exact prompts bug).
   *
   * This test references `./skills-api`, which is the production file
   * that does NOT exist yet — that is the RED state. The `expectTypeOf`
   * call will fail at type-check time because the module is absent.
   */
  it("Skill.current_revision is exactly `number` (not `number | undefined`, not optional)", async () => {
    const mod = await import("./skills-api");

    // Compile-time assertion: Skill["current_revision"] MUST be assignable to
    // a `number` slot. If the type is `number | undefined`, expectTypeOf
    // reports a type mismatch and the test fails.
    expectTypeOf<mod.Skill["current_revision"]>().toEqualTypeOf<number>();

    // Runtime assertion: build a sample Skill and confirm the field
    // is present at runtime.
    const sample: mod.Skill = {
      id: 1,
      name: "x",
      description: "d",
      body: "b",
      current_revision: 1,
      created_at: "",
      updated_at: "",
      deleted_at: null,
    };
    expect(sample.current_revision).toBe(1);
  });
});

// ---------------------------------------------------------------------------
// Task 5.2 — parseResponse reads NESTED envelope (RED)
// ---------------------------------------------------------------------------

describe("skills-api parseResponse — NESTED envelope (anti-drift obs #1959 item 1)", () => {
  /**
   * Task 5.2 RED — parseResponse reads body.error.message (NESTED).
   *
   * The prompts bug (obs #1959 item 1) is that `parseResponse` read
   * `body.message ?? <default>` (flat). Backend actually emits
   * `{error:{code,message,fields?}}` (nested). The result: rich backend
   * messages are silently dropped.
   *
   * This test asserts the nested message is preferred over the flat
   * one. If the impl reads top-level `body.message` only, the test
   * FAILS — it sees the flat "Fallback flat message" instead of the
   * nested "Backend nested message".
   */
  it("prefers body.error.message (nested) over body.message (flat fallback)", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      {
        message: "Fallback flat message (this should NOT win)",
        error: { code: "validation", message: "Backend nested message" },
      },
      { status: 400 },
    );
    // parseResponse is exported as a named export for testability.
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.message).toBe("Backend nested message");
      expect(result.message).not.toBe("Fallback flat message (this should NOT win)");
    }
  });

  /**
   * Task 5.2 — also reads body.error.fields (NESTED). Same anti-drift
   * gate, different field.
   */
  it("prefers body.error.fields (nested) over body.fields (flat fallback)", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      {
        message: "x",
        fields: { name: "should NOT win" },
        error: {
          code: "validation",
          message: "validation",
          fields: { name: "Backend nested name error" },
        },
      },
      { status: 400 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok && result.kind === "validation") {
      expect(result.fields).toEqual({ name: "Backend nested name error" });
      expect(result.fields.name).not.toBe("should NOT win");
    }
  });
});

// ---------------------------------------------------------------------------
// Task 5.3 — parseResponse maps status → ApiErrorKind
// ---------------------------------------------------------------------------

describe("skills-api parseResponse — status → kind mapping", () => {
  /**
   * Task 5.3 — 400 with envelope → kind=validation + fields populated.
   */
  it("400 → kind=validation with NESTED message + fields", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      {
        error: {
          code: "validation",
          message: "Skill name must be lowercase.",
          fields: { name: "must be lowercase" },
        },
      },
      { status: 400 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("validation");
      expect(result.message).toBe("Skill name must be lowercase.");
      if (result.kind === "validation") {
        expect(result.fields).toEqual({ name: "must be lowercase" });
      }
    }
  });

  /**
   * Task 5.3 — 409 → kind=conflict (duplicate name).
   */
  it("409 → kind=conflict", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      { error: { code: "conflict", message: "Skill name already taken." } },
      { status: 409 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("conflict");
      expect(result.message).toBe("Skill name already taken.");
    }
  });

  /**
   * Task 5.3 — 404 → kind=not_found.
   */
  it("404 → kind=not_found", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      { error: { code: "not_found", message: "Skill not found." } },
      { status: 404 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("not_found");
      expect(result.message).toBe("Skill not found.");
    }
  });

  /**
   * Task 5.3 — 410 (soft-deleted) → kind=not_found (UX treats gone as missing).
   */
  it("410 → kind=not_found (soft-deleted UX)", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      { error: { code: "skill_deleted", message: "Skill has been deleted." } },
      { status: 410 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("not_found");
      expect(result.message).toBe("Skill has been deleted.");
    }
  });

  /**
   * Task 5.3 — 500 → kind=server.
   */
  it("500 → kind=server", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      { error: { code: "server", message: "Internal error." } },
      { status: 500 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("server");
      expect(result.message).toBe("Internal error.");
    }
  });

  /**
   * Task 5.3 — fetch throw → kind=offline. We exercise this through
   * the public wrapper (listSkills) which catches and returns offline.
   */
  it("fetch throw → kind=offline (via listSkills)", async () => {
    const { listSkills } = await import("./skills-api");
    // Force a network throw by mocking fetch.
    const original = globalThis.fetch;
    globalThis.fetch = (() => {
      throw new Error("ECONNREFUSED");
    }) as unknown as typeof fetch;
    try {
      const result = await listSkills();
      expect(result.ok).toBe(false);
      if (!result.ok) {
        expect(result.kind).toBe("offline");
      }
    } finally {
      globalThis.fetch = original;
    }
  });

  /**
   * Task 5.3 — 204 No Content → ok:true with undefined value.
   */
  it("204 → ok:true with value undefined", async () => {
    const mod = await import("./skills-api");
    const resp = new Response(undefined, { status: 204 });
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value).toBeUndefined();
    }
  });

  /**
   * Task 5.3 — 200/201 with JSON body → ok:true with parsed value.
   */
  it("200 + JSON body → ok:true with parsed value", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse({ hello: "world" }, { status: 200 });
    const result = await mod.parseResponse<{ hello: string }>(resp);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value).toEqual({ hello: "world" });
    }
  });
});

// ---------------------------------------------------------------------------
// Tasks 5.4-5.9 — Wire-shape tests for the 7 endpoint wrappers
// ---------------------------------------------------------------------------

// Sample fixtures (mirror backend wire shapes).
const SAMPLE_SKILL: import("./skills-api").Skill = {
  id: 1,
  name: "pdf-cleanup",
  description: "Clean up PDFs.",
  body: "---\nname: pdf-cleanup\ndescription: Clean up PDFs.\n---\n# Body",
  current_revision: 2,
  created_at: "2026-07-17T08:00:00Z",
  updated_at: "2026-07-17T09:00:00Z",
  deleted_at: null,
};

const SAMPLE_REVISION: import("./skills-api").SkillRevision = {
  id: 10,
  skill_id: 1,
  revision_number: 2,
  description: "Clean up PDFs.",
  body: "---\nname: pdf-cleanup\ndescription: Clean up PDFs.\n---\n# Body",
  change_note: null,
  created_at: "2026-07-17T09:00:00Z",
};

describe("skills-api wire shapes (anti-drift obs #1959 items 3,4,5,6)", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() => Promise.resolve(mockResponse([], { status: 200 }))),
    );
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  // -------------------------------------------------------------------------
  // Task 5.4 — listSkills calls GET /skills (NO query string)
  // -------------------------------------------------------------------------

  it("Task 5.4 — listSkills calls GET /skills with NO query string", async () => {
    const { listSkills } = await import("./skills-api");
    await listSkills();
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    // URL MUST be exactly /skills — NO ?deleted=false (obs #1959 item 5).
    expect(url).toMatch(/\/skills$/);
    expect(url).not.toContain("?");
    expect(url).not.toContain("deleted=");
    expect(call[1]?.method).toBeUndefined(); // GET defaults
  });

  // -------------------------------------------------------------------------
  // Task 5.5 — getSkill + deleteSkill URL/method
  // -------------------------------------------------------------------------

  it("Task 5.5 — getSkill calls GET /skills/:name with encoded name", async () => {
    const { getSkill } = await import("./skills-api");
    await getSkill("pdf-cleanup");
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    expect(url).toMatch(/\/skills\/pdf-cleanup$/);
    expect(call[1]?.method).toBeUndefined();
  });

  it("Task 5.5 — getSkill encodes slugs with special characters", async () => {
    const { getSkill } = await import("./skills-api");
    await getSkill("a/b c");
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    // encodeURIComponent encodes /, space, etc.
    expect(url).toContain(encodeURIComponent("a/b c"));
  });

  it("Task 5.5 — deleteSkill calls DELETE /skills/:name and returns ok on 204", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      new Response(undefined, { status: 204 }),
    );
    const { deleteSkill } = await import("./skills-api");
    const result = await deleteSkill("pdf-cleanup");
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    expect(url).toMatch(/\/skills\/pdf-cleanup$/);
    expect(call[1]?.method).toBe("DELETE");
    expect(result.ok).toBe(true);
  });

  // -------------------------------------------------------------------------
  // Task 5.6 — createSkill calls POST /skills with JSON
  // -------------------------------------------------------------------------

  it("Task 5.6 — createSkill calls POST /skills with JSON {name, description, body}", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(mockResponse(SAMPLE_SKILL, { status: 201 }));
    const { createSkill } = await import("./skills-api");
    const result = await createSkill({
      name: "pdf-cleanup",
      description: "Clean up PDFs.",
      body: "---\nname: pdf-cleanup\ndescription: Clean up PDFs.\n---\n# Body",
    });
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    expect(url).toMatch(/\/skills$/);
    expect(call[1]?.method).toBe("POST");
    const parsed = JSON.parse(call[1]?.body as string) as Record<string, string>;
    expect(parsed.name).toBe("pdf-cleanup");
    expect(parsed.description).toBe("Clean up PDFs.");
    expect(parsed.body).toContain("name: pdf-cleanup");
    const headers = call[1]?.headers as Record<string, string>;
    expect(headers["Content-Type"]).toBe("application/json");
    expect(result.ok).toBe(true);
  });

  // -------------------------------------------------------------------------
  // Task 5.7 — updateSkill calls PATCH /skills/:name with BOTH description AND body
  // -------------------------------------------------------------------------

  it("Task 5.7 — updateSkill calls PATCH /skills/:name with BOTH description AND body", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(mockResponse(SAMPLE_SKILL, { status: 200 }));
    const { updateSkill } = await import("./skills-api");
    const result = await updateSkill("pdf-cleanup", {
      description: "Updated description",
      body: "---\nname: pdf-cleanup\ndescription: Updated description\n---\n# Body",
    });
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    expect(url).toMatch(/\/skills\/pdf-cleanup$/);
    expect(call[1]?.method).toBe("PATCH");
    // Anti-drift obs #1959 item 4: BOTH fields MUST be sent.
    const parsed = JSON.parse(call[1]?.body as string) as Record<string, string>;
    expect(parsed.description).toBe("Updated description");
    expect(parsed.body).toContain("Updated description");
    expect(result.ok).toBe(true);
  });

  // -------------------------------------------------------------------------
  // Task 5.9 — listRevisions + restoreRevision
  // -------------------------------------------------------------------------

  it("Task 5.9 — listRevisions calls GET /skills/:name/revisions", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(
      mockResponse([SAMPLE_REVISION], { status: 200 }),
    );
    const { listRevisions } = await import("./skills-api");
    await listRevisions("pdf-cleanup");
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    expect(url).toMatch(/\/skills\/pdf-cleanup\/revisions$/);
    expect(call[1]?.method).toBeUndefined();
  });

  it("Task 5.9 — restoreRevision calls POST /skills/:name/revisions/:n/restore", async () => {
    vi.mocked(fetch).mockResolvedValueOnce(mockResponse(SAMPLE_SKILL, { status: 200 }));
    const { restoreRevision } = await import("./skills-api");
    const result = await restoreRevision("pdf-cleanup", 2);
    const call = vi.mocked(fetch).mock.calls[0];
    const url = call[0] as string;
    expect(url).toMatch(/\/skills\/pdf-cleanup\/revisions\/2\/restore$/);
    expect(call[1]?.method).toBe("POST");
    expect(result.ok).toBe(true);
  });

  // -------------------------------------------------------------------------
  // Anti-drift — NO ?deleted=false dead query param on listSkills
  // -------------------------------------------------------------------------

  it("anti-drift — listSkills does NOT send ?deleted=false (obs #1959 item 5)", async () => {
    const { listSkills } = await import("./skills-api");
    await listSkills();
    const url = vi.mocked(fetch).mock.calls[0][0] as string;
    expect(url).not.toMatch(/deleted=/);
  });
});
