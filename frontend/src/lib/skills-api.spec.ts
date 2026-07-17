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

import { describe, expect, expectTypeOf, it } from "vitest";

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
