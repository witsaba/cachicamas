/**
 * API client for the prompts management endpoints.
 *
 * Mirrors the conventions from lib/api.ts:
 *   - Discriminated ApiResult<T> for errors
 *   - Form-encoded bodies (application/x-www-form-urlencoded)
 *   - serverAwareFetch() bridges browser SSR vs Node SSR contexts
 *
 * Wire shapes match the backend domain from
 * backend/database_administrator/src/domain/prompt.go.
 *
 * Error kinds:
 *   validation  → 400
 *   conflict   → 409 (slug already taken)
 *   not_found  → 404 (prompt not found) + 410 (soft-deleted, treated as not_found)
 *   server     → 500
 *   offline    → network error
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Prompt {
  id: number;
  slug: string;
  description: string | null;
  body: string;
  current_revision: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface PromptRevision {
  id: number;
  prompt_id: number;
  revision_number: number;
  body: string;
  created_at: string;
}

export interface CreatePromptInput {
  slug: string;
  description?: string;
  body: string;
}

export interface UpdatePromptInput {
  body: string;
}

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
// Helpers
// ---------------------------------------------------------------------------

/**
 * Resolve the API base URL.
 *
 * Browser: uses import.meta.env.PUBLIC_API_BASE_URL (Vite-inlined).
 * Node SSR: uses process.env.SERVER_API_BASE_URL.
 * Defaults to http://localhost:8080.
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

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const fromEnv = (import.meta as any).env?.PUBLIC_API_BASE_URL as
    | string
    | undefined;
  return (fromEnv ?? "http://localhost:8080").replace(/\/+$/, "");
}

/**
 * Check if we're running in Node (SSR) or browser.
 */
function isServerSide(): boolean {
  return (
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined"
  );
}

/**
 * Fetch wrapper that works in both Node SSR and browser contexts.
 * - Browser: uses standard fetch with relative URL
 * - Node SSR: uses absolute URL via SERVER_API_BASE_URL
 */
async function safeFetch(
  input: RequestInfo,
  init?: RequestInit,
): Promise<Response> {
  const url = typeof input === "string" ? input : input.url;
  const base = apiBaseUrl();
  const fullUrl = isServerSide()
    ? url.startsWith("http")
      ? url
      : `${base}${url}`
    : url.startsWith("http")
      ? url
      : url;

  const resp = await fetch(fullUrl, {
    ...init,
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      ...(init?.headers as Record<string, string> | undefined),
    },
  });
  return resp;
}

/**
 * Map a fetch Response to ApiResult<T>.
 * Handles the special 410 Gone case (soft-deleted prompt → treated as not_found).
 */
async function parseResponse<T>(resp: Response): Promise<ApiResult<T>> {
  if (resp.status === 410) {
    // Soft-deleted prompt — treat as not_found for UI purposes
    return {
      ok: false,
      kind: "not_found",
      message: "Prompt not found.",
    };
  }

  if (resp.ok) {
    if (resp.status === 204) {
      // DELETE success — no body
      return { ok: true, value: undefined as unknown as T };
    }
    const data = await resp.json();
    return { ok: true, value: data as T };
  }

  // Error response — parse envelope
  let body: { message?: string; fields?: Record<string, string> } = {};
  try {
    body = await resp.json();
  } catch {
    // non-JSON error body
  }

  if (resp.status === 400) {
    return {
      ok: false,
      kind: "validation",
      message: body.message ?? "Invalid request body.",
      fields: body.fields ?? {},
    };
  }

  if (resp.status === 409) {
    return {
      ok: false,
      kind: "conflict",
      message: body.message ?? "Conflict.",
    };
  }

  if (resp.status === 404) {
    return {
      ok: false,
      kind: "not_found",
      message: body.message ?? "Not found.",
    };
  }

  return {
    ok: false,
    kind: "server",
    message: body.message ?? "Server error.",
  };
}

/**
 * Encode form data as application/x-www-form-urlencoded.
 */
function formEncode(data: Record<string, string | undefined>): string {
  return Object.entries(data)
    .filter(([, v]) => v !== undefined)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v!)}`)
    .join("&");
}

// ---------------------------------------------------------------------------
// API Functions
// ---------------------------------------------------------------------------

/**
 * List all non-deleted prompts.
 * GET /prompts?deleted=false
 */
export async function listPrompts(): Promise<ApiResult<Prompt[]>> {
  try {
    const resp = await safeFetch(`${apiBaseUrl()}/prompts?deleted=false`);
    return parseResponse<Prompt[]>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Get a single prompt by slug.
 * GET /prompts/:slug
 * Returns 410 Gone (treated as not_found) for soft-deleted prompts.
 */
export async function getPrompt(slug: string): Promise<ApiResult<Prompt>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/prompts/${encodeURIComponent(slug)}`,
    );
    return parseResponse<Prompt>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Create a new prompt.
 * POST /prompts
 */
export async function createPrompt(
  input: CreatePromptInput,
): Promise<ApiResult<Prompt>> {
  try {
    const body = formEncode({
      slug: input.slug,
      description: input.description ?? "",
      body: input.body,
    });
    const resp = await safeFetch(`${apiBaseUrl()}/prompts`, {
      method: "POST",
      body,
    });
    return parseResponse<Prompt>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Update a prompt's body (creates a new revision).
 * PUT /prompts/:slug
 */
export async function updatePrompt(
  slug: string,
  body: string,
): Promise<ApiResult<Prompt>> {
  try {
    const encodedBody = formEncode({ body });
    const resp = await safeFetch(
      `${apiBaseUrl()}/prompts/${encodeURIComponent(slug)}`,
      {
        method: "PUT",
        body: encodedBody,
      },
    );
    return parseResponse<Prompt>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Soft-delete a prompt.
 * DELETE /prompts/:slug
 */
export async function deletePrompt(slug: string): Promise<ApiResult<void>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/prompts/${encodeURIComponent(slug)}`,
      {
        method: "DELETE",
      },
    );
    return parseResponse<void>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * List all revisions for a prompt.
 * GET /prompts/:slug/revisions
 */
export async function listRevisions(
  slug: string,
): Promise<ApiResult<PromptRevision[]>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/prompts/${encodeURIComponent(slug)}/revisions`,
    );
    return parseResponse<PromptRevision[]>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Get a specific revision.
 * GET /prompts/:slug/revisions/:n
 */
export async function getRevision(
  slug: string,
  revisionNumber: number,
): Promise<ApiResult<PromptRevision>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/prompts/${encodeURIComponent(slug)}/revisions/${revisionNumber}`,
    );
    return parseResponse<PromptRevision>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}

/**
 * Restore a revision as a new revision.
 * POST /prompts/:slug/revisions/:n/restore
 */
export async function restoreRevision(
  slug: string,
  revisionNumber: number,
): Promise<ApiResult<Prompt>> {
  try {
    const resp = await safeFetch(
      `${apiBaseUrl()}/prompts/${encodeURIComponent(slug)}/revisions/${revisionNumber}/restore`,
      {
        method: "POST",
      },
    );
    return parseResponse<Prompt>(resp);
  } catch {
    return {
      ok: false,
      kind: "offline",
      message: "Couldn't reach the backend. Is docker compose up?",
    };
  }
}
