/**
 * assistant-config.ts — typed client for the assistant config endpoints.
 *
 * Two endpoints, both under `/api/chat/assistant/config`:
 *   - GET  → returns the persisted config for the caller's org
 *            (auto-seeded defaults if no row yet, per the chat
 *            archetype's safe-default contract)
 *   - PUT  → validates + persists a new config; returns the new
 *            config with the bumped version
 *
 * Wire shape mirrors the Go ArchetypeConfig (see
 * backend/agent/src/archetype/config.go). The `kind` field is
 * always `chat` for the chat archetype; future archetypes would
 * surface their own endpoints.
 *
 * Error envelope parity with chat-api.ts (REQ-4, REQ-5):
 *   - HTTP 400 → ApiResult with `kind: "validation"` and per-field
 *     errors from the backend's response envelope.
 *   - HTTP 403 → ApiResult with `kind: "not_found"` (the chat
 *     handler refuses anonymous callers with a generic envelope;
 *     treating as not_found is the closest mapping for the UI).
 *   - Network errors → `{ ok: false, kind: "offline", message }`.
 */

import type { ApiResult } from "~/lib/api";

/**
 * ArchetypeConfig is the wire shape for `/api/chat/assistant/config`.
 *
 * Mirrors `archetype.ArchetypeConfig` from
 * `backend/agent/src/archetype/config.go` — keep the field names in
 * lock-step when extending either side. The frontend never reads the
 * `kind` field directly (the URL namespace already disambiguates
 * the chat archetype); it's included for round-trip completeness.
 */
export interface ArchetypeConfig {
  kind: "chat";
  org_id: string;
  system_prompt: string;
  tool_allowlist: string[];
  defer_tool_names: string[];
  /** Informational; the actual model is selected by the env-driven
   * runtime, not the persisted config. May be absent. */
  model?: string | null;
  version: number;
  updated_by?: string;
  updated_at?: string;
  /** `true` when the config came from a per-org row that shadows the
   * system default. `false` when the config came from the seeded
   * `__default__` row OR the in-memory fallback. The directory uses
   * this to label the Assistant card "Configured" (per-org override)
   * vs "Default" (system default). */
  is_override: boolean;
}

/**
 * AssistantConfigUpdate is the PUT body shape (REQ-CACAPI-002).
 * The backend rejects empty prompts, oversized prompts, HTML
 * patterns, unknown tool names, and defer names not in the
 * allowlist; see the backend sentinels
 * (`archetype.Err*`).
 */
export interface AssistantConfigUpdate {
  system_prompt: string;
  tool_allowlist: string[];
  defer_tool_names: string[];
  model?: string | null;
}

const ASSISTANT_CONFIG_ENDPOINT = "/api/chat/assistant/config";

/**
 * getAssistantConfig fetches the persisted config for the caller's
 * org. Auto-seeded defaults are returned when no row exists yet
 * (the chat handler falls back to safe defaults per design AD-2).
 *
 * Returns ApiResult so the caller can branch on offline / not_found
 * / server errors without parsing the envelope shape inline.
 */
export async function getAssistantConfig(): Promise<ApiResult<ArchetypeConfig>> {
  return getJson<ArchetypeConfig>(ASSISTANT_CONFIG_ENDPOINT);
}

/**
 * putAssistantConfig validates + persists a new config. The backend
 * returns the new row (with bumped version + server-set updated_at).
 *
 * On 400 the backend's error envelope is decoded into the ApiResult's
 * `fields` map so the Configure UI can highlight per-field errors.
 */
export async function putAssistantConfig(
  update: AssistantConfigUpdate,
): Promise<ApiResult<ArchetypeConfig>> {
  return sendJson<ArchetypeConfig>(ASSISTANT_CONFIG_ENDPOINT, "PUT", update);
}

/* ---------- internal helpers (mirrors chat-api.ts conventions) ---------- */

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
      message: envelope.message ?? "Sign in to manage the Assistant.",
    };
  }
  return {
    ok: false,
    kind: "server",
    message: envelope.message ?? "The Assistant could not be reached.",
  };
}

function offlineResult<T>(error: unknown): ApiResult<T> {
  return {
    ok: false,
    kind: "offline",
    message: error instanceof Error ? error.message : "Network unreachable.",
  };
}
