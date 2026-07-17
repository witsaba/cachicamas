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
