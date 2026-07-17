/**
 * Frontend API client for the Skills management endpoints.
 *
 * WIRE CONTRACT — verified against the backend handler at
 *   backend/database_administrator/src/interfaces/http/skill_handler.go
 * (PR1d, PR #57, merged).
 *
 * Anti-drift guards (from obs #1959) baked into this file's contract:
 *   - parseResponse reads body.error?.message ?? body.message ?? <default>
 *     (the prompts gotcha — flat-fixture vs nested-envelope)
 *   - parseResponse reads body.error?.fields ?? body.fields ?? {}
 *   - Skill.current_revision is `number` (never undefined — backend emits it)
 *   - listSkills() URL is exactly `/skills` (NO `?deleted=` no-op param)
 *   - updateSkill sends BOTH description AND body (no silent discard)
 *   - updateSkill JSDoc says PATCH (not PUT)
 *   - 7 functions exported, no dead exports for nonexistent routes
 *   - INDEPENDENT from prompts-api.ts (separate parseResponse, separate types)
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Wire shape for a Skill returned by the backend.
 * Mirror of `backend/database_administrator/src/domain/skill.go` SkillDetail.
 *
 * Anti-drift gate (obs #1959 item 2): `current_revision` is `number`,
 * NOT optional, NOT `number | undefined`. Backend emits it via SQL JOIN
 * (ADR-SK-008). Declaring it as optional would let the `v{undefined}`
 * sidebar render bug slip back in.
 */
export interface Skill {
  /** Backend bigserial primary key. */
  id: number;
  /** URL slug + agentskills.io name. Lowercase alphanum + hyphens, 1..64 chars. */
  name: string;
  /** Human-readable description. 1..1024 chars. NOT nullable — backend always emits it. */
  description: string;
  /** Full SKILL.md content (frontmatter + markdown). 1..524288 bytes. */
  body: string;
  /** Latest revision number, ALWAYS present (ADR-SK-008). */
  current_revision: number;
  /** ISO-8601 timestamp. */
  created_at: string;
  /** ISO-8601 timestamp. */
  updated_at: string;
  /** ISO-8601 timestamp or null. Soft-deleted skills have this set. */
  deleted_at: string | null;
}

export interface SkillRevision {
  id: number;
  skill_id: number;
  revision_number: number;
  description: string;
  body: string;
  /** Free-form note (e.g. "restored from revision 2"). May be null. */
  change_note: string | null;
  created_at: string;
}

/**
 * The four error kinds the frontend distinguishes.
 *   - validation: 400 (with `fields` map)
 *   - conflict:   409 (slug already taken)
 *   - not_found:  404 + 410 (gone, treated as not_found)
 *   - server:     500
 *   - offline:    network error
 */
export type ApiErrorKind =
  | "validation"
  | "conflict"
  | "not_found"
  | "server"
  | "offline";

export type ApiResult<T> =
  | { ok: true; value: T }
  | {
      ok: false;
      kind: "validation";
      message: string;
      fields: Record<string, string>;
    }
  | {
      ok: false;
      kind: Exclude<ApiErrorKind, "validation">;
      message: string;
    };

// ---------------------------------------------------------------------------
// API Functions (7 total — no more, no less)
// ---------------------------------------------------------------------------

/**
 * List all active skills.
 * GET /skills
 * (NO `?deleted=` query — backend filters by `deleted_at IS NULL` regardless.
 *  Carrying the param invites confusion; see obs #1959 item 5.)
 */
export function listSkills(): Promise<ApiResult<Skill[]>> {
  // Implemented in Task 5.4.
  throw new Error("not implemented (will land in GREEN of Task 5.4)");
}

/**
 * Get a single skill by name.
 * GET /skills/:name
 * Returns 404 `not_found` for missing or soft-deleted skills.
 */
export function getSkill(name: string): Promise<ApiResult<Skill>> {
  // Implemented in Task 5.5.
  throw new Error("not implemented (will land in GREEN of Task 5.5)");
}

/**
 * Create a new skill.
 * POST /skills
 * Body: JSON {name, description, body}
 */
export function createSkill(input: {
  name: string;
  description: string;
  body: string;
}): Promise<ApiResult<Skill>> {
  // Implemented in Task 5.6.
  throw new Error("not implemented (will land in GREEN of Task 5.6)");
}

/**
 * Update a skill (creates a new revision).
 * PATCH /skills/:name
 * Body: JSON {description, body} — BOTH fields sent (no silent discard).
 */
export function updateSkill(
  name: string,
  input: { description: string; body: string },
): Promise<ApiResult<Skill>> {
  // Implemented in Task 5.7.
  throw new Error("not implemented (will land in GREEN of Task 5.7)");
}

/**
 * Soft-delete a skill (idempotent).
 * DELETE /skills/:name
 */
export function deleteSkill(name: string): Promise<ApiResult<void>> {
  // Implemented in Task 5.5.
  throw new Error("not implemented (will land in GREEN of Task 5.5)");
}

/**
 * List all revisions for a skill, newest-first.
 * GET /skills/:name/revisions
 */
export function listRevisions(name: string): Promise<ApiResult<SkillRevision[]>> {
  // Implemented in Task 5.9.
  throw new Error("not implemented (will land in GREEN of Task 5.9)");
}

/**
 * Restore a specific revision as a NEW latest revision.
 * POST /skills/:name/revisions/:n/restore
 */
export function restoreRevision(
  name: string,
  revisionNumber: number,
): Promise<ApiResult<Skill>> {
  // Implemented in Task 5.9.
  throw new Error("not implemented (will land in GREEN of Task 5.9)");
}
