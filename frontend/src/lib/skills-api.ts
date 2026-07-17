/**
 * Frontend API client for the Skills management endpoints.
 *
 * WIRE CONTRACT — verified against the backend handler at
 * `backend/database_administrator/src/interfaces/http/skill_handler.go`
 * (PR1d, PR #57, merged).
 *
 * Anti-drift guards (obs #1959):
 * - parseResponse reads body.error?.message ?? body.message ?? <default>
 * - parseResponse reads body.error?.fields ?? body.fields ?? {}
 * - Skill.current_revision is `number` (backend emits it via SQL JOIN)
 * - listSkills URL is exactly `/skills` (NO `?deleted=` no-op)
 * - updateSkill sends BOTH description AND body
 * - updateSkill JSDoc says PATCH (not PUT)
 * - 7 functions exported, no dead exports
 * - INDEPENDENT from prompts-api.ts
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Wire shape for a Skill returned by the backend.
 * Mirror of `backend/database_administrator/src/domain/skill.go` SkillDetail.
 *
 * Anti-drift gate (obs #1959 item 2): `current_revision` is `number`,
 * NOT optional. Backend emits it via SQL JOIN (ADR-SK-008).
 */
export interface Skill {
  /** Backend bigserial primary key. */
  id: number;
  /** URL slug. Lowercase alphanum + hyphens, 1..64 chars. */
  name: string;
  /** Human-readable description. 1..1024 chars. NOT nullable. */
  description: string;
  /** Full SKILL.md content. 1..524288 bytes. */
  body: string;
  /** Latest revision number. ALWAYS present. */
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
// Internal helpers
// ---------------------------------------------------------------------------

/**
 * Resolve the API base URL.
 * (Mirrors prompts-api.ts so the rest of the app can stitch the same
 *  dev/prod wiring. Independent copy: prompts-api.ts does NOT export this.)
 */
function apiBaseUrl(): string {
  const isNode =
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined";

  if (isNode) {
    const fromEnv = process.env.SERVER_API_BASE_URL;
    return (
      fromEnv && fromEnv.trim().length > 0 ? fromEnv : "http://localhost:8080"
    ).replace(/\/+$/, "");
  }

  const fromEnv = (import.meta as { env?: Record<string, string> }).env
    ?.PUBLIC_API_BASE_URL as string | undefined;
  return (fromEnv ?? "http://localhost:8080").replace(/\/+$/, "");
}

/**
 * Fetch wrapper. In Node SSR resolves relative URLs against apiBaseUrl;
 * in the browser passes them through.
 *
 * Default Content-Type is application/json because every Skills endpoint
 * that accepts a body (POST, PATCH) decodes via `json.NewDecoder` on
 * the backend.
 */
async function safeFetch(
  input: string,
  init?: RequestInit,
): Promise<Response> {
  const base = apiBaseUrl();
  const isNode =
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined";
  const fullUrl = isNode
    ? input.startsWith("http")
      ? input
      : `${base}${input}`
    : input;
  return await fetch(fullUrl, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers as Record<string, string> | undefined),
    },
  });
}

/**
 * Extract the human-readable error message from a JSON response body.
 *
 * **NESTED FIRST** — backend emits `{error:{code,message,fields?}}`, so
 * we read `body.error?.message` first. Flat `body.message` is a fallback
 * only (some legacy endpoints may still emit it).
 *
 * Anti-drift gate (obs #1959 item 1): the prompts client did it the
 * other way around and silently dropped rich backend messages.
 */
function readErrorMessage(body: Record<string, unknown>, fallback: string): string {
  const err = body["error"];
  if (err && typeof err === "object" && err !== null) {
    const msg = (err as Record<string, unknown>)["message"];
    if (typeof msg === "string" && msg.length > 0) return msg;
  }
  const flat = body["message"];
  if (typeof flat === "string" && flat.length > 0) return flat;
  return fallback;
}

/**
 * Extract the field-level error map. NESTED FIRST (obs #1959 item 1).
 * Returns an empty object if no fields are present.
 */
function readErrorFields(body: Record<string, unknown>): Record<string, string> {
  const err = body["error"];
  if (err && typeof err === "object" && err !== null) {
    const fields = (err as Record<string, unknown>)["fields"];
    if (fields && typeof fields === "object" && fields !== null) {
      const out: Record<string, string> = {};
      for (const [k, v] of Object.entries(fields as Record<string, unknown>)) {
        if (typeof v === "string") out[k] = v;
      }
      return out;
    }
  }
  const flat = body["fields"];
  if (flat && typeof flat === "object" && flat !== null) {
    const out: Record<string, string> = {};
    for (const [k, v] of Object.entries(flat as Record<string, unknown>)) {
      if (typeof v === "string") out[k] = v;
    }
    return out;
  }
  return {};
}

/**
 * Map a fetch Response to ApiResult<T>.
 *
 * Status mapping (anti-drift obs #1959):
 * - 200/201 → ok:true, parsed JSON
 * - 204     → ok:true, undefined
 * - 400     → kind:validation (with fields)
 * - 409     → kind:conflict
 * - 404     → kind:not_found
 * - 410     → kind:not_found (soft-deleted UX)
 * - 500+    → kind:server
 *
 * Exported for testability.
 */
export async function parseResponse<T>(resp: Response): Promise<ApiResult<T>> {
  if (resp.status === 204) {
    return { ok: true, value: undefined as unknown as T };
  }

  if (resp.ok) {
    const data = (await resp.json()) as T;
    return { ok: true, value: data };
  }

  let body: Record<string, unknown> = {};
  try {
    body = (await resp.json()) as Record<string, unknown>;
  } catch {
    // non-JSON body; defaults below handle the fallback.
  }

  const message = readErrorMessage(body, `HTTP ${resp.status}`);
  const fields = readErrorFields(body);

  if (resp.status === 400) {
    return { ok: false, kind: "validation", message, fields };
  }
  if (resp.status === 409) {
    return { ok: false, kind: "conflict", message };
  }
  if (resp.status === 404 || resp.status === 410) {
    return { ok: false, kind: "not_found", message };
  }
  return { ok: false, kind: "server", message };
}

// ---------------------------------------------------------------------------
// API Functions (7 total — no more, no less)
// ---------------------------------------------------------------------------

/**
 * List all active skills.
 * GET /skills
 * (NO `?deleted=` query — backend filters by `deleted_at IS NULL` regardless.
 *  Carrying the param invites confusion; see obs #1959 item 5.)
 */
export async function listSkills(): Promise<ApiResult<Skill[]>> {
  try {
    const resp = await safeFetch(`${apiBaseUrl()}/skills`);
    return await parseResponse<Skill[]>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Get a single skill by name.
 * GET /skills/:name
 * Returns 404 `not_found` for missing or soft-deleted skills.
 */
export async function getSkill(name: string): Promise<ApiResult<Skill>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/skills/${encodeURIComponent(name)}`,
    );
    return await parseResponse<Skill>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Create a new skill.
 * POST /skills
 * Body: JSON {name, description, body}
 */
export async function createSkill(input: {
  name: string;
  description: string;
  body: string;
}): Promise<ApiResult<Skill>> {
  try {
    const resp = await safeFetch(`${apiBaseUrl()}/skills`, {
      method: "POST",
      body: JSON.stringify(input),
    });
    return await parseResponse<Skill>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Update a skill (creates a new revision).
 * PATCH /skills/:name
 * Body: JSON {description, body} — BOTH fields sent (no silent discard).
 */
export async function updateSkill(
  name: string,
  input: { description: string; body: string },
): Promise<ApiResult<Skill>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/skills/${encodeURIComponent(name)}`,
      {
        method: "PATCH",
        body: JSON.stringify(input),
      },
    );
    return await parseResponse<Skill>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Soft-delete a skill (idempotent).
 * DELETE /skills/:name
 */
export async function deleteSkill(name: string): Promise<ApiResult<void>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/skills/${encodeURIComponent(name)}`,
      {
        method: "DELETE",
      },
    );
    return await parseResponse<void>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * List all revisions for a skill, newest-first.
 * GET /skills/:name/revisions
 */
export async function listRevisions(
  name: string,
): Promise<ApiResult<SkillRevision[]>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/skills/${encodeURIComponent(name)}/revisions`,
    );
    return await parseResponse<SkillRevision[]>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Restore a specific revision as a NEW latest revision.
 * POST /skills/:name/revisions/:n/restore
 */
export async function restoreRevision(
  name: string,
  revisionNumber: number,
): Promise<ApiResult<Skill>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/skills/${encodeURIComponent(name)}/revisions/${revisionNumber}/restore`,
      { method: "POST" },
    );
    return await parseResponse<Skill>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}
