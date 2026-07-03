/**
 * API client for the database_administrator Go microservice.
 *
 * This client is the production bridge from the Qwik frontend to
 * the Go binary's HTTP API.  It replaces the temporary smart-stub
 * that was used while R-5 ("Qwik SSR + Go binary single-process",
 * design §10) was still locked.  Now that `docker compose up` brings
 * the Go bin and Postgres online locally, every form submission and
 * every list / detail load round-trips through this client.
 *
 * Conventions
 * -----------
 *   - Base URL defaults to `http://localhost:8080` (the Go bin's
 *     `SERVICE_PORT` from `.env`).  Override with
 *     `PUBLIC_API_BASE_URL`.  Vite inlines `PUBLIC_*` env at build
 *     time for both SSR and client bundles.
 *   - Bodies are form-encoded (`application/x-www-form-urlencoded`).
 *     Locked #3 in the API contract allows JSON or form-encoded;
 *     form-encoded avoids JSON serialisation on the client and
 *     preserves the same field names the Qwik form already
 *     collects, so the write path is mechanical.
 *   - Errors are mapped from the locked envelope (see
 *     `domain/organization_handler.go writeError`) to a single
 *     discriminated `ApiResult`.  Every non-`ok` variant carries
 *     both `kind` (a typed discriminator) and `message` (a
 *     human-readable summary).  The `validation` variant also
 *     carries the per-field `fields` map so callers can render
 *     specific inline errors.  Synthesis rules:
 *
 *       validation   → message = "<firstFieldKey>: <firstFieldMsg>"
 *                      (or "Invalid request body." if no fields)
 *       conflict     → message = body.message (slug-already-taken)
 *       not_found    → message = body.message
 *       server       → message = body.message or generic fallback
 *       offline      → message = "Couldn't reach the backend at
 *                       PUBLIC_API_BASE_URL. Is docker compose up?"
 *
 *     Callers never have to deal with raw Responses or wire-shaped
 *     bodies — they get a clean `result.kind` plus a safe
 *     `result.message` for any UI surface.
 *   - "offline" is a separate failure mode from "server 500".
 *     Offline means the fetch itself threw (DNS, refused,
 *     timeout); the user gets a friendly hint about
 *     `docker compose up`, never a stack trace.
 *
 * Env override: set `PUBLIC_API_BASE_URL` (frontend/.env.local or
 * shell).  Without it, the default kicks in.
 */

export interface OrganizationReadModel {
  id: number;
  shortname: string | null;
  full_name: string;
  identification: string;
  is_active: boolean;
  email: string | null;
  phone: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateOrganizationInput {
  fullName: string;
  identification: string;
  shortName?: string;
  email?: string;
  phone?: string;
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
      /** Human-readable summary, safe to render inline. */
      message: string;
      /** Per-field validation errors from the backend envelope. */
      fields: Record<string, string>;
    }
  | {
      ok: false;
      kind: Exclude<ApiErrorKind, "validation">;
      message: string;
    };

const DEFAULT_BASE_URL = "http://localhost:8080";

/**
 * Resolve the API base URL.  Reads `PUBLIC_API_BASE_URL` (Qwik/Vite
 * inlines PUBLIC_* at build time).  Falls back to localhost so a
 * fresh `pnpm dev` works without an env file when docker compose
 * is up on the default port.
 */
export function apiBaseUrl(): string {
  const fromEnv =
    typeof process !== "undefined"
      ? process.env.PUBLIC_API_BASE_URL
      : undefined;
  return (
    fromEnv && fromEnv.trim().length > 0 ? fromEnv : DEFAULT_BASE_URL
  ).replace(/\/+$/, "");
}

function offlineMessage(err: unknown): string {
  const detail = err instanceof Error ? err.message : "network error";
  return `Couldn't reach the backend at ${apiBaseUrl()}. Is docker compose up? (${detail})`;
}

async function envelopeToResult<T>(
  res: Response,
  parseBody: () => Promise<T>,
): Promise<ApiResult<T>> {
  if (res.ok) {
    return { ok: true as const, value: await parseBody() };
  }
  let body: Record<string, unknown> = {};
  try {
    body = (await res.json()) as Record<string, unknown>;
  } catch {
    // Body was not JSON (or empty).  Fall through with the defaults.
  }
  const err = body.error as string | undefined;
  const message = body.message as string | undefined;
  if (res.status === 400 && err === "validation") {
    const fields = (body.fields ?? {}) as Record<string, string>;
    const firstEntry = Object.entries(fields)[0];
    const synthesized = firstEntry
      ? `${firstEntry[0]}: ${firstEntry[1]}`
      : (message ?? "Invalid request body.");
    return {
      ok: false,
      kind: "validation",
      message: synthesized,
      fields,
    };
  }
  if (res.status === 409 && err === "conflict") {
    return {
      ok: false,
      kind: "conflict",
      message: message ?? "This slug is already taken. Try another.",
    };
  }
  if (res.status === 404 && err === "not_found") {
    return {
      ok: false,
      kind: "not_found",
      message: message ?? "Organization not found.",
    };
  }
  return {
    ok: false,
    kind: "server",
    message: message ?? `Server error (${res.status}).`,
  };
}

function toFormBody(input: CreateOrganizationInput): URLSearchParams {
  const params = new URLSearchParams();
  params.append("full_name", input.fullName);
  params.append("identification", input.identification);
  // Empty string is treated as "not provided" (see
  // `organization_handler.optionalFormValue`), so we only append
  // optional fields when they have real content.
  if (input.shortName) params.append("shortname", input.shortName);
  if (input.email) params.append("email", input.email);
  if (input.phone) params.append("phone", input.phone);
  return params;
}

/**
 * POST /organizations.
 *
 * Returns the newly created organization on success.  The form's
 * submit handler maps the discriminated result back to
 * `FormActionResult`:
 *   - ok              → navigate to /organizations/{id}
 *   - validation      → render per-field errors (or generic)
 *   - conflict        → render the slug-taken message
 *   - server / offline → render a top-level alert
 */
export async function createOrganization(
  input: CreateOrganizationInput,
): Promise<ApiResult<OrganizationReadModel>> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/organizations`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: toFormBody(input),
    });
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(
    res,
    async () => (await res.json()) as OrganizationReadModel,
  );
}

/** GET /organizations/{id}. */
export async function getOrganization(
  id: number,
): Promise<ApiResult<OrganizationReadModel>> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/organizations/${id}`);
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(
    res,
    async () => (await res.json()) as OrganizationReadModel,
  );
}

/**
 * GET /organizations.  Always resolves to an array (defensive:
 * a wire-level `null` would otherwise propagate to the loader
 * value and break the list render).
 */
export async function listOrganizations(): Promise<
  ApiResult<OrganizationReadModel[]>
> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/organizations`);
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as OrganizationReadModel[] | null;
    return Array.isArray(body) ? body : [];
  });
}
