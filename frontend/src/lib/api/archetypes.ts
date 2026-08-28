/**
 * archetypes.ts — typed client for the polymorphic archetype surface.
 *
 * Three endpoints, all under the polymorphic /api/archetypes tree:
 *   - GET  /api/archetypes/{slug}             (per-profile overlay)
 *   - GET  /api/archetypes?type={type}        (directory list)
 *   - GET  /api/archetypes/{slug}/config/     (per-org config; this is
 *     what the chat-package assistant-config.ts evolves into)
 *   - PUT  /api/archetypes/{slug}/config/     (per-org write)
 *
 * The wire shape mirrors the Go backend's ArchetypeView from
 * `backend/agent/src/archetype/catalog.go` — keep the field names in
 * lock-step when extending either side. The `override` field is
 * nested (the chat-package's flat `ArchetypeConfig` is gone; the
 * polymorphic surface returns a nested `override` block, with the
 * per-org customisation living alongside the system defaults).
 *
 * The `is_override` field on the TS interface is a derived boolean
 * (true iff `override` is present and non-null) so the frontend can
 * branch on it without `?.`-chaining the override field. The backend
 * does NOT emit `is_override`; the client synthesises it.
 *
 * Error envelope parity with chat-api.ts (REQ-4, REQ-5):
 *   - HTTP 400 → ApiResult with `kind: "validation"` and the
 *     `code` field from the backend envelope's fields map.
 *   - HTTP 403 → ApiResult with `kind: "not_found"` (the handler
 *     refuses anonymous callers with a generic envelope; treating as
 *     not_found is the closest mapping for the UI).
 *   - HTTP 404 → ApiResult with `kind: "not_found"`.
 *   - Network errors → `{ ok: false, kind: "offline", message }`.
 */

import type { ApiResult } from "~/lib/api";

// -----------------------------------------------------------------------------
// Wire types
// -----------------------------------------------------------------------------

export type ArchetypeType = "system" | "general" | "owned";

export type ArchetypeStatus = "active" | "draft" | "archived";

/**
 * ArchetypeOverride is the per-org customisation row. Present iff the
 * caller's org has a row in archetype_configurations(archetype_slug, org_id);
 * absent (or null) when the org is on the system default.
 */
export interface ArchetypeOverride {
  readonly system_prompt: string;
  readonly tool_allowlist: readonly string[];
  readonly defer_tool_names: readonly string[];
  /** Informational; the actual model is env-driven, not the persisted config. */
  readonly model: string | null;
  readonly version: number;
  readonly updated_at: string;
  readonly updated_by: string;
}

/**
 * ArchetypeView is the polymorphic read result. Parent columns are
 * always populated when the row exists. Child columns (bundle_version,
 * is_critical) are populated only for type='system'. The optional
 * `override` field carries the per-org customisation when one exists.
 *
 * The derived `is_override` boolean is `true` iff `override` is
 * present; the frontend can branch on it without optional chaining.
 */
export interface ArchetypeView {
  readonly slug: string;
  readonly type: ArchetypeType;
  readonly display_name: string;
  readonly tagline: string;
  readonly status: ArchetypeStatus;
  readonly archived_at: string | null;
  readonly created_at: string;
  readonly created_by: string;

  // System-only child columns. Absent for type='general'|'owned'.
  readonly bundle_version?: string;
  readonly is_critical?: boolean;

  // Per-org override. Absent when the org is on the system default.
  readonly override?: ArchetypeOverride;

  // Derived: true iff override is present.
  readonly is_override: boolean;
}

/**
 * ArchetypeUpdate is the PUT body shape. The slug comes from the
 * path; orgID is server-derived; the backend returns the new persisted
 * view (with the bumped version + server-set updated_at/updated_by).
 */
export interface ArchetypeUpdate {
  readonly system_prompt: string;
  readonly tool_allowlist: readonly string[];
  readonly defer_tool_names: readonly string[];
  readonly model?: string | null;
}

// -----------------------------------------------------------------------------
// URL helpers
// -----------------------------------------------------------------------------

/** URL for the per-slug config GET/PUT. */
export function archetypeConfigURL(slug: string): string {
  return `/api/archetypes/${encodeURIComponent(slug)}/config/`;
}

/** URL for the per-slug read (no override block; profile overlay). */
export function archetypeURL(slug: string): string {
  return `/api/archetypes/${encodeURIComponent(slug)}`;
}

/**
 * URL for the directory list. Without `type` this is the BARE
 * /api/archetypes URL with NO type query (the unfiltered cross-type
 * directory, CRL-S-010); with `type` it narrows to one catalogue.
 * `type` MUST be one of the locked values when supplied.
 */
export function archetypesListURL(type?: ArchetypeType): string {
  return type === undefined
    ? "/api/archetypes"
    : `/api/archetypes?type=${encodeURIComponent(type)}`;
}

// -----------------------------------------------------------------------------
// Public client
// -----------------------------------------------------------------------------

/**
 * getArchetype(slug) reads the polymorphic ArchetypeView for the
 * supplied slug. Returns the per-org config block in the `override`
 * field; `is_override` is true iff an override exists.
 */
export async function getArchetype(
  slug: string,
): Promise<ApiResult<ArchetypeView>> {
  return getJson<ArchetypeView>(archetypeURL(slug));
}

/**
 * putArchetypeConfig(slug, update) validates + persists a new per-org
 * config. Returns the server's response (the new persisted row, with
 * bumped version + server-set updated_at/updated_by), NOT the request
 * body — see Test_putArchetypeConfig_ReturnsServerResponseNotRequestBody
 * in archetypes.spec.ts.
 */
export async function putArchetypeConfig(
  slug: string,
  update: ArchetypeUpdate,
): Promise<ApiResult<ArchetypeView>> {
  return sendJson<ArchetypeView>(archetypeConfigURL(slug), "PUT", update);
}

/**
 * getArchetypeConfigPolymorphic(slug) reads the polymorphic
 * per-org config block for the supplied slug. This is the
 * /config/ counterpart of getArchetype(slug): where getArchetype
 * hits /api/archetypes/{slug} for the parent overlay (no override),
 * this hits /api/archetypes/{slug}/config/ for the per-org override
 * block. The legacy adapter at assistant-config.ts flattens the
 * response to the legacy ArchetypeConfig shape; new code should
 * prefer this helper over the legacy getAssistantConfig() alias.
 *
 * @see REQ-ACAR-1, REQ-ACAR-5
 */
export async function getArchetypeConfigPolymorphic(
  slug: string,
): Promise<ApiResult<ArchetypeView>> {
  return getJson<ArchetypeView>(archetypeConfigURL(slug));
}

/**
 * listArchetypes(type?) returns the directory list. Called WITHOUT a
 * type argument it requests the bare /api/archetypes URL with no
 * `type` query (CRL-S-010) — the full catalogue the staff directory
 * renders, assistant row included. With a type argument it requests
 * /api/archetypes?type={type} for a narrowed list. The backend's
 * handler returns 400 for an unknown type; the client surfaces that
 * as a typed ApiResult.kind="validation".
 */
export async function listArchetypes(
  type?: ArchetypeType,
): Promise<ApiResult<readonly ArchetypeView[]>> {
  return getJson<readonly ArchetypeView[]>(archetypesListURL(type));
}

// -----------------------------------------------------------------------------
// Internal helpers (mirrors assistant-config.ts conventions)
// -----------------------------------------------------------------------------

interface ErrorEnvelope {
  kind?: "validation" | "conflict" | "not_found" | "server";
  message?: string;
  fields?: Record<string, string>;
}

async function getJson<T>(path: string): Promise<ApiResult<T>> {
  try {
    const response = await fetch(path, {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" },
    });
    return parseResponse<T>(response);
  } catch (error) {
    return offlineResult<T>(error);
  }
}

async function sendJson<T>(
  path: string,
  method: "PUT" | "POST",
  body: unknown,
): Promise<ApiResult<T>> {
  try {
    const response = await fetch(path, {
      method,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify(body),
    });
    return parseResponse<T>(response);
  } catch (error) {
    return offlineResult<T>(error);
  }
}

async function parseResponse<T>(response: Response): Promise<ApiResult<T>> {
  if (response.ok) {
    const value = (await response.json()) as T;
    return { ok: true, value };
  }
  let envelope: ErrorEnvelope = {};
  try {
    envelope = (await response.json()) as ErrorEnvelope;
  } catch {
    // body wasn't JSON — fall through with empty envelope
  }
  if (response.status === 400 || envelope.kind === "validation") {
    return {
      ok: false,
      kind: "validation",
      message: envelope.message ?? "The configuration was rejected.",
      fields: envelope.fields ?? {},
    };
  }
  if (response.status === 403) {
    return {
      ok: false,
      kind: "not_found",
      message: envelope.message ?? "Sign in to manage the archetype.",
    };
  }
  if (response.status === 404) {
    return {
      ok: false,
      kind: "not_found",
      message: envelope.message ?? "Archetype slug is not registered.",
    };
  }
  return {
    ok: false,
    kind: "server",
    message: envelope.message ?? "The archetype service could not be reached.",
  };
}

function offlineResult<T>(error: unknown): ApiResult<T> {
  return {
    ok: false,
    kind: "offline",
    message: error instanceof Error ? error.message : "Network unreachable.",
  };
}
