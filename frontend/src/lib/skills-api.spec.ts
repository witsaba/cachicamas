/**
 * Wire-shape tests for the Skills frontend API client.
 *
 * Anti-drift guards (from obs #1959) each enforced as a failing-then-passing test:
 *   - 5.1: Skill.current_revision is `number` (not undefined-allowed)
 *   - 5.2: parseResponse reads body.error.message (NESTED envelope)
 *   - 5.3: parseResponse maps HTTP status → ApiResult kind
 *   - 5.4: listSkills calls GET /skills (NO query string)
 *   - 5.5: getSkill + deleteSkill URL/method
 *   - 5.6: createSkill calls POST with JSON {name, description, body}
 *   - 5.7: updateSkill calls PATCH with BOTH description AND body
 *   - 5.8: updateSkill JSDoc says PATCH not PUT (source-grep)
 *   - 5.9: listRevisions + restoreRevision URL/method
 *
 * NO test imports prompts-api.ts (independence gate).
 * NO test uses FLAT error envelopes — only NESTED.
 */

import { afterEach, beforeEach, describe, expect, expectTypeOf, it, vi } from "vitest";
import type { Skill } from "./skills-api";

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
  // Task 5.1 — Skill.current_revision is `number`, not undefined.
  // If the type allowed `undefined`, sidebar would render `v{undefined}`
  // (the exact prompts bug). Backend emits via SQL JOIN (ADR-SK-008).
  it("Skill.current_revision is exactly `number` (not `number | undefined`)", async () => {
    // Compile-time + runtime assertion.
    expectTypeOf<Skill["current_revision"]>().toEqualTypeOf<number>();
    const sample: Skill = {
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
  // Task 5.2 — parseResponse reads body.error.message (NESTED first).
  // The prompts bug read body.message (flat) first; backend emits
  // `{error:{code,message,fields?}}` (nested). Rich backend messages
  // were silently dropped.
  it("prefers body.error.message (nested) over body.message (flat fallback)", async () => {
    const mod = await import("./skills-api");
    const resp = mockResponse(
      {
        message: "Fallback flat message (this should NOT win)",
        error: { code: "validation", message: "Backend nested message" },
      },
      { status: 400 },
    );
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.message).toBe("Backend nested message");
      expect(result.message).not.toBe("Fallback flat message (this should NOT win)");
    }
  });

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
  // Task 5.3 — status → kind mapping with NESTED envelopes.
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

  it("fetch throw → kind=offline (via listSkills)", async () => {
    const { listSkills } = await import("./skills-api");
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

  it("204 → ok:true with value undefined", async () => {
    const mod = await import("./skills-api");
    const resp = new Response(undefined, { status: 204 });
    const result = await mod.parseResponse<unknown>(resp);
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value).toBeUndefined();
    }
  });

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

// ---------------------------------------------------------------------------
// Task 5.8 — updateSkill JSDoc MUST say "PATCH" (not "PUT")
// ---------------------------------------------------------------------------

describe("skills-api JSDoc — PATCH not PUT (anti-drift obs #1959 item 4)", () => {
  // Task 5.8 — updateSkill JSDoc MUST mention PATCH. The prompts bug
  // (#1959 item 4) had JSDoc saying PUT while impl used PATCH — this
  // source-grep pins the comment.
  it("updateSkill JSDoc explicitly mentions PATCH", async () => {
    const fs = await import("node:fs/promises");
    const path = await import("node:path");
    const source = await fs.readFile(
      path.resolve(__dirname, "./skills-api.ts"),
      "utf8",
    );
    const idx = source.indexOf("export async function updateSkill");
    expect(idx).toBeGreaterThan(-1);
    const before = source.slice(0, idx);
    const jsDocStart = before.lastIndexOf("/**");
    expect(jsDocStart).toBeGreaterThan(-1);
    const jsDoc = source.slice(jsDocStart, idx);
    expect(jsDoc).toContain("PATCH");
  });
});

// ---------------------------------------------------------------------------
// Anti-drift — skills-api does NOT import prompts-api (independence gate)
// ---------------------------------------------------------------------------

describe("skills-api — independence from prompts-api (anti-drift design §4.4)", () => {
  it("skills-api.ts does NOT import from prompts-api (separate parseResponse)", async () => {
    const fs = await import("node:fs/promises");
    const path = await import("node:path");
    const source = await fs.readFile(
      path.resolve(__dirname, "./skills-api.ts"),
      "utf8",
    );
    expect(source).not.toContain('from "./prompts-api"');
    expect(source).not.toContain("from \"./prompts-api\"");
  });
});
