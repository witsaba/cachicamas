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

/**
 * `SetupState` is the install-level "is there at least one
 * organization in the database?" boolean. Used by the
 * `requireOwnboarding` helper (frontend/src/lib/require-ownboarding.ts)
 * to decide whether the user lands on /home or /ownboarding after
 * authentication.
 *
 * Wire shape (matches backend `application.SetupState`):
 *   { hasOrganization: boolean }
 *
 * Why a dedicated interface instead of reusing `OrganizationReadModel`:
 * the setup state is structurally different (no id, no name, no
 * timestamps) and the contract is independent of any single org.
 * Keeping the types separate makes the wire shape self-documenting.
 */
export interface SetupState {
  hasOrganization: boolean;
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
 * Resolve the API base URL.
 *
 * Two paths:
 *   - **Browser**: uses `import.meta.env.PUBLIC_API_BASE_URL` (Vite-inlined
 *     at build time). A RELATIVE URL like `/api` works because the
 *     browser resolves it against the page origin. This avoids
 *     cross-origin requests from the browser to the Go bin.
 *
 *   - **Node SSR (server)**: uses `process.env.SERVER_API_BASE_URL`
 *     (set via env var in the Dockerfile, e.g.
 *     `SERVER_API_BASE_URL=http://database_administrator:8080`).
 *     Node's `fetch` REQUIRES an absolute URL — a relative one throws
 *     `TypeError: Failed to parse URL`. The server-side absolute URL
 *     uses the compose network DNS name; no CORS issue because the
 *     browser doesn't see this fetch.
 *
 * Falls back to `DEFAULT_BASE_URL` for local dev (no env vars set).
 */
export function apiBaseUrl(): string {
  // Detect Node SSR context. In Node, `process` is defined and
  // `import.meta.env` may not be populated the same way.
  const isNode =
    typeof process !== "undefined" &&
    typeof (process as { versions?: unknown }).versions !== "undefined";

  if (isNode) {
    const fromServerEnv = process.env.SERVER_API_BASE_URL;
    return (
      fromServerEnv && fromServerEnv.trim().length > 0
        ? fromServerEnv
        : DEFAULT_BASE_URL
    ).replace(/\/+$/, "");
  }

  // Browser: PUBLIC_* env vars are Vite-inlined at build time.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const fromEnv = (import.meta as any).env?.PUBLIC_API_BASE_URL as
    | string
    | undefined;
  return (
    fromEnv && fromEnv.trim().length > 0 ? fromEnv : DEFAULT_BASE_URL
  ).replace(/\/+$/, "");
}

function offlineMessage(err: unknown): string {
  const detail = err instanceof Error ? err.message : "network error";
  return `Couldn't reach the backend at ${apiBaseUrl()}. Is docker compose up? (${detail})`;
}

// ---------------------------------------------------------------------------
// SSR cookie-forwarding helpers
// ---------------------------------------------------------------------------
//
// Why this section exists (S-WS-AUTH-CHAIN-SSR-001):
//
// Qwik's `routeLoader$` runs at SSR (server-side) BEFORE the response is
// sent, and at resumption on the client. The browser auto-sends the
// `authjs.session-token` cookie with same-origin fetches, but Node's
// `fetch` (used by routeLoader$ during SSR) does NOT — the inbound
// request's cookies are NOT automatically forwarded to outgoing
// fetches. So when a routeLoader$ calls
// `${SERVER_API_BASE_URL}/workspaces`, the backend's
// IdentityFromCookie middleware sees no cookie and returns 401.
//
// The fix: the route's `onRequest` middleware (see `withSsrCookieContext`)
// captures the inbound Cookie header into AsyncLocalStorage scoped to
// the current request. Each api.ts fetch helper reads from that store
// and re-attaches the cookie to its outgoing fetch on SSR. Browser-side
// calls (after hydration) bypass this path because AsyncLocalStorage
// has nothing to read — the same fetch goes through the relative `/api`
// URL where the browser auto-attaches the cookie.
// ---------------------------------------------------------------------------

import { getSsrCookieHeader } from "./ssr-cookie-context";

function ssrBaseUrl(): string {
  const v = process.env.SERVER_API_BASE_URL;
  const base =
    v && v.trim().length > 0 ? v : DEFAULT_BASE_URL;
  return base.replace(/\/+$/, "");
}

/**
 * Attach the inbound Cookie header if running inside an SSR cookie
 * context. Browser-side calls (no context set) get no extra header
 * and the regular same-origin behaviour applies.
 */
function withSsrCookieHeader(init?: RequestInit): RequestInit {
  const cookie = getSsrCookieHeader();
  if (!cookie) return init ?? {};
  return {
    ...(init ?? {}),
    headers: {
      ...(init?.headers ?? {}),
      cookie,
    },
  };
}

async function ssrFetch(
  path: string,
  init?: RequestInit,
): Promise<Response> {
  return fetch(`${ssrBaseUrl()}${path}`, withSsrCookieHeader(init));
}

/** SSR-time GET /workspaces with the inbound request's cookies (when in SSR). */
export async function listWorkspacesSSR(): Promise<
  ApiResult<{ workspaces: WorkspaceSummary[]; truncated: boolean }>
> {
  let res: Response;
  try {
    res = await ssrFetch("/workspaces");
  } catch (err) {
    return { ok: false, kind: "offline", message: offlineMessage(err) };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as {
      workspaces: WorkspaceSummary[];
      truncated?: boolean;
    };
    return {
      workspaces: body.workspaces ?? [],
      truncated: body.truncated ?? false,
    };
  });
}

/** SSR-time GET /workspaces/:id with the inbound request's cookies (when in SSR). */
export async function getWorkspaceSSR(
  id: number,
): Promise<ApiResult<WorkspaceDetail>> {
  let res: Response;
  try {
    res = await ssrFetch(`/workspaces/${id}`);
  } catch (err) {
    return { ok: false, kind: "offline", message: offlineMessage(err) };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as WorkspaceDetail;
    return body;
  });
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
  // 400 OR 422 with envelope.error === "validation" both map to kind=validation
  // (the workspace handler returns 422 for inaccessible-repo / business-rule
  // validation; the legacy organization handler returns 400).
  if ((res.status === 400 || res.status === 422) && err === "validation") {
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

/**
 * GET /setup-state.
 *
 * Reference: openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md
 *   R-OW-005 (S-OW-040..044) — backend endpoint contract.
 *   R-OW-008 (S-OW-070..073) — failure-mode fallback semantics.
 *
 * Returns { hasOrganization: boolean } on success.
 *
 * Throws an Error on any failure path — the caller
 * (`requireOwnboarding`) is responsible for the optimistic
 * no-throw fallback. This split keeps the API client honest about
 * failures (no swallowing errors) while letting the guard decide
 * whether to redirect or render normally.
 *
 * Failure modes:
 *   - Network error / fetch throws     → throws Error("setup-state fetch failed: ...")
 *   - Non-200 HTTP status              → throws Error("setup-state failed: HTTP <code>")
 *   - Malformed JSON / wrong shape     → throws Error("setup-state: malformed response ...")
 */
export async function getSetupState(): Promise<SetupState> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/setup-state`);
  } catch (err) {
    const detail = err instanceof Error ? err.message : "unknown";
    throw new Error(`setup-state fetch failed: ${detail}`);
  }
  if (!res.ok) {
    throw new Error(`setup-state failed: HTTP ${res.status}`);
  }
  const body = (await res.json()) as Partial<SetupState>;
  if (typeof body.hasOrganization !== "boolean") {
    throw new Error(
      "setup-state: malformed response (hasOrganization not boolean)",
    );
  }
  return { hasOrganization: body.hasOrganization };
}

/**
 * GET /organization.
 *
 * Reference: openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md
 *   R-FIX-002 — header org pill use case.
 *
 * Returns the current (single) organization on success, or null when
 * 404 (no org yet). Throws on any other failure path — the caller
 * (layout `useOrgLoader`) is responsible for the optimistic no-throw
 * fallback.
 *
 * Failure modes:
 *   - Network error / fetch throws     → throws Error("organization fetch failed: ...")
 *   - HTTP 404                         → returns null (no org yet — not an error)
 *   - Other non-200 HTTP status        → throws Error("organization failed: HTTP <code>")
 *   - Malformed JSON                   → throws Error("organization: malformed response ...")
 */
export async function getCurrentOrganization(): Promise<OrganizationReadModel | null> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/organization`);
  } catch (err) {
    const detail = err instanceof Error ? err.message : "unknown";
    throw new Error(`organization fetch failed: ${detail}`);
  }
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error(`organization failed: HTTP ${res.status}`);
  }
  const body = (await res.json()) as Partial<OrganizationReadModel>;
  // Minimal wire-shape sanity check. The OrgPill only needs full_name
  // + identification, but the layout may pass the whole record to a
  // future settings page — validate just the two fields the pill
  // actually reads.
  if (typeof body.full_name !== "string" || body.full_name.length === 0) {
    throw new Error("organization: malformed response (full_name missing)");
  }
  if (
    typeof body.identification !== "string" ||
    body.identification.length === 0
  ) {
    throw new Error(
      "organization: malformed response (identification missing)",
    );
  }
  return body as OrganizationReadModel;
}

// ---------------------------------------------------------------------------
// Workspaces (2026-07-06 PR2-i)
// ---------------------------------------------------------------------------

/**
 * PrimaryRepository is the locked wire shape for the primary repo stored
 * on every workspace. Mirrors `backend/.../src/domain/workspace.go`.
 */
export interface PrimaryRepository {
  github_id: number;
  full_name: string;
  owner: string;
  name: string;
}

export interface LinkedRepository {
  id: number;
  github_id: number;
  github_full_name: string;
  github_owner: string;
  github_name: string;
  added_at: string;
}

export interface WorkspaceSummary {
  id: number;
  name: string;
  primary_repository: PrimaryRepository;
  linked_repos_count: number;
  created_at: string;
}

export interface WorkspaceDetail {
  id: number;
  name: string;
  primary_repository: PrimaryRepository;
  linked_repositories: LinkedRepository[];
  created_at: string;
  updated_at: string;
}

/** GET /workspaces (R-WS-002). */
export async function listWorkspaces(): Promise<
  ApiResult<{ workspaces: WorkspaceSummary[]; truncated: boolean }>
> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/workspaces`);
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as {
      workspaces: WorkspaceSummary[];
      truncated?: boolean;
    };
    return {
      workspaces: body.workspaces ?? [],
      truncated: body.truncated ?? false,
    };
  });
}

/** GET /workspaces/:id (R-WS-003). */
export async function getWorkspace(
  id: number,
): Promise<ApiResult<WorkspaceDetail>> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/workspaces/${id}`);
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(
    res,
    async () => (await res.json()) as WorkspaceDetail,
  );
}

/** GET /workspaces/:id/repositories (R-WS-008). */
export async function listLinkedRepos(
  workspaceID: number,
): Promise<ApiResult<{ repositories: LinkedRepository[] }>> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/workspaces/${workspaceID}/repositories`);
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as { repositories: LinkedRepository[] };
    return { repositories: body.repositories ?? [] };
  });
}

/**
 * Input for `addRepoToWorkspace` (R-WS-006).
 *
 * Backend validates the github_id against the user's accessible
 * /user/repos set on the server (T7 in design); the frontend passes
 * the user's selected repo through verbatim.
 */
export interface AddRepoInput {
  github_id: number;
  github_full_name: string;
  github_owner: string;
  github_name: string;
}

/** POST /workspaces/:id/repositories (R-WS-006). */
export async function addRepoToWorkspace(
  workspaceID: number,
  repo: AddRepoInput,
): Promise<ApiResult<LinkedRepository>> {
  let res: Response;
  try {
    res = await fetch(
      `${apiBaseUrl()}/workspaces/${workspaceID}/repositories`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          github_id: repo.github_id,
          github_full_name: repo.github_full_name,
          github_owner: repo.github_owner,
          github_name: repo.github_name,
        }),
      },
    );
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(
    res,
    async () => (await res.json()) as LinkedRepository,
  );
}

/** DELETE /workspaces/:id/repositories/:repoId (R-WS-007). */
export async function removeRepoFromWorkspace(
  workspaceID: number,
  repoID: number,
): Promise<ApiResult<null>> {
  let res: Response;
  try {
    res = await fetch(
      `${apiBaseUrl()}/workspaces/${workspaceID}/repositories/${repoID}`,
      { method: "DELETE" },
    );
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  // 204 No Content — body is empty; envelopeToResult guards against a
  // missing body, returning ok: true with null value.
  return envelopeToResult(res, async () => null);
}

/** DELETE /workspaces/:id (R-WS-005, soft delete on the backend). */
export async function deleteWorkspace(id: number): Promise<ApiResult<null>> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/workspaces/${id}`, {
      method: "DELETE",
    });
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  return envelopeToResult(res, async () => null);
}

/**
 * CreateWorkspaceInput — wire shape for POST /workspaces (R-WS-001).
 *
 * The backend validates: name 3..60 chars + primary_repository fields
 * all required (T7: server-side GitHub accessibility check). The frontend
 * Zod schema mirrors these rules for client-side early validation.
 */
export interface CreateWorkspaceInput {
  name: string;
  primaryRepository: PrimaryRepository;
}

/**
 * POST /workspaces (R-WS-001).
 *
 * Returns ApiResult<WorkspaceDetail>:
 *   - 201 → ok with the new workspace detail
 *   - 400 with fields.name → kind=validation
 *   - 422 with fields.primary_repository → kind=validation (repo not accessible)
 *   - 409 → kind=server (duplicate name)
 *   - 401 / 5xx / network → offline/server
 */
export async function createWorkspace(
  input: CreateWorkspaceInput,
): Promise<ApiResult<WorkspaceDetail>> {
  let res: Response;
  try {
    res = await fetch(`${apiBaseUrl()}/workspaces`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: input.name,
        primary_repository: {
          github_id: input.primaryRepository.github_id,
          github_full_name: input.primaryRepository.full_name,
          github_owner: input.primaryRepository.owner,
          github_name: input.primaryRepository.name,
        },
      }),
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
    async () => (await res.json()) as WorkspaceDetail,
  );
}

/**
 * GitHubRepo is the projection the GitHub proxy returns. Mirrors the
 * backend `github.Repo` shape (id, full_name, owner_login, name, private,
 * description, html_url, updated_at, stargazers_count).
 */
export interface GitHubRepo {
  id: number;
  full_name: string;
  owner_login: string;
  name: string;
  private: boolean;
  description: string;
  html_url: string;
  updated_at: string;
  stargazers_count: number;
}

/**
 * ListGitHubReposOptions — query parameters for GET /github/repos.
 */
export interface ListGitHubReposOptions {
  bustCache?: boolean;
  page?: number;
  perPage?: number;
}

/**
 * GET /github/repos (R-WS-009).
 *
 * Server-side proxy through the backend. The frontend never sees the
 * OAuth access_token; the backend loads it from identity.account and
 * calls the GitHub API. Cache is 5-min in-memory per user.
 *
 * Returns ApiResult<{ repositories, hasNext }>:
 *   - 200 → ok with the repos + hasNext
 *   - 401 github_not_connected → kind=server with reconnect message
 *   - 502 github_unauthorized / rate_limited / unreachable → kind=server
 *   - network → kind=offline
 */
export async function listGitHubRepos(
  options: ListGitHubReposOptions = {},
): Promise<ApiResult<{ repositories: GitHubRepo[]; hasNext: boolean }>> {
  const params = new URLSearchParams();
  if (options.bustCache) params.set("bust_cache", "true");
  if (options.page) params.set("page", String(options.page));
  if (options.perPage) params.set("per_page", String(options.perPage));
  const qs = params.toString();
  const url = `${apiBaseUrl()}/github/repos${qs ? `?${qs}` : ""}`;

  let res: Response;
  try {
    res = await fetch(url);
  } catch (err) {
    return {
      ok: false,
      kind: "offline",
      message: offlineMessage(err),
    };
  }
  // 401 with code=github_not_connected: surface a clear reconnect message.
  if (res.status === 401) {
    let code = "";
    try {
      const body = (await res.json()) as { error?: string };
      code = body.error ?? "";
    } catch {
      // body not JSON; fall through with empty code.
    }
    if (code === "github_not_connected") {
      return {
        ok: false,
        kind: "server",
        message: "Reconnect GitHub to list repositories.",
      };
    }
  }
  return envelopeToResult(res, async () => {
    const body = (await res.json()) as {
      repositories: GitHubRepo[];
      has_next?: boolean;
    };
    return {
      repositories: body.repositories ?? [],
      hasNext: body.has_next ?? false,
    };
  });
}
