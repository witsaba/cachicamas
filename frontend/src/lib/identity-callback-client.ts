/**
 * Identity signin-callback client — Qwik Node SSR side.
 *
 * This module is the BRIDGE between Auth.js's `events.signIn` callback
 * (frontend/src/routes/plugin@auth.ts) and the database_administrator
 * Go service's `POST /api/v1/identity/signin-callback` endpoint. It
 * replaces the previous slice's direct-from-frontend Postgres writes
 * (rejected architecture: porsager/postgres running on the Node SSR
 * shared the database role + credentials with the Go service).
 *
 * Reference: `docs/adr/0003-add-identity-callback-hmac.md` (this slice).
 *
 * Wire protocol (locked):
 *
 *   POST /api/v1/identity/signin-callback
 *   Headers:
 *     Content-Type: application/json
 *     X-Cachicamas-Timestamp: <unix_ms string>
 *     X-Cachicamas-Signature: base64(HMAC_SHA256(secret, ts + "." + canonical_json))
 *   Body: { user: {...}, account: {...} }
 *
 * Canonical JSON (locked, cross-tooling with the Go side):
 *   - map keys: sorted lexicographically (NOT lowercased)
 *   - whitespace: none
 *   - string escaping: RFC 8259 (JSON.stringify default)
 *   - numbers: NOT supported by this endpoint's schema; the Go side
 *     rejects them at canonicalization time. The TS side never
 *     produces numbers (the wire body uses strings + nulls only).
 *
 * Server-only:
 *   The client is invoked from `events.signIn`, which runs on the
 *   Node SSR side BEFORE the response is sent. It is NOT loaded from
 *   the browser bundle (no `node:crypto` available there). The
 *   module's top-level imports of `node:crypto` would crash the
 *   browser build if accidentally imported there; the production
 *   wiring in plugin@auth.ts ensures the import is server-only.
 *
 * Failure modes (locked envelope shape on the server side; the client
 * mirrors it in the thrown error message):
 *   - 204 No Content: success
 *   - 401 Unauthorized: bad / expired / missing signature
 *   - 422 Unprocessable Entity: invalid body schema
 *   - 500 Internal Server Error: service error (db down, etc.)
 *   - network failure: fetch throws (caller catches)
 *
 * Env vars:
 *   - IDENTITY_CALLBACK_SECRET (required): HMAC secret. Same value
 *     the Go service has.
 *   - SERVER_API_BASE_URL (compose): full URL the Node SSR uses to
 *     reach the Go bin directly (e.g. http://database_administrator:8080).
 *   - ORIGIN (dev): the public-facing origin used to construct the
 *     reverse-proxy URL when SERVER_API_BASE_URL is unset.
 *
 * Best-effort posture: the calling wiring in plugin@auth.ts logs the
 * thrown error and swallows it so a successful GitHub sign-in is
 * NEVER blocked by an identity persistence failure. Mirrors the
 * PR #29 posture (now superseded) that the inline 4R review approved
 * as R4-1 [LOW].
 *
 * Why `node:crypto` is dynamically imported: the file is reachable
 * from the Qwik client bundle's import graph (plugin@auth.ts imports
 * it). A static `import { createHmac } from "node:crypto"` would fail
 * the browser build with "Module 'node:crypto' has been externalized
 * for browser compatibility". The dynamic import inside the function
 * body keeps the static graph clean; Vite/Rollup see no node:crypto
 * reference and emit a tree-shakable module. The runtime call still
 * resolves the module on the Node SSR side (where the callback runs).
 */

// ---------------------------------------------------------------------------
// Public surface
// ---------------------------------------------------------------------------

/**
 * SignInEvent mirrors the Auth.js for Qwik contract for the
 * `events.signIn` callback (`@auth/core/index.d.ts:343`):
 *
 *   { user: User; account?: Account | null; profile?: Profile; isNewUser?: boolean }
 *
 * Field names use snake_case to match Auth.js's Account type (which
 * is what `@auth/qwik@0.9.2` actually passes through). The wire body
 * uses camelCase keys (see serializeWireBody); the conversion is the
 * client's responsibility.
 *
 * `account` is OPTIONAL on the wire (Auth.js allows it to be missing
 * or null for credentials-provider flows). We deny sign-in by
 * returning without dispatching when it's missing — mirrors the
 * previous slice's "no account → no persistence" posture.
 */
export interface SignInEvent {
  user: SignInUser;
  account?: SignInAccount | null;
  isNewUser?: boolean;
}

export interface SignInUser {
  id?: string;
  name?: string | null;
  email?: string | null;
  image?: string | null;
}

export interface SignInAccount {
  provider: string;
  providerAccountId: string;
  access_token?: string | null;
  refresh_token?: string | null;
  expires_at?: number | null;
  token_type?: string | null;
  scope?: string | null;
}

/**
 * postIdentityCallback sends the OAuth event to the database_administrator
 * signin-callback endpoint with an HMAC-SHA256 signature.
 *
 * Returns `void` on success (the response is 204 No Content with no body).
 * Throws on any failure (env unset, missing account, fetch failure,
 * non-204 response). The caller (plugin@auth.ts) logs + swallows so
 * the OAuth roundtrip is never blocked.
 */
export async function postIdentityCallback(event: SignInEvent): Promise<void> {
  if (!event.account) {
    // Mirrors the previous slice's "no account → no persistence" rule:
    // Auth.js's credentials-provider flow doesn't carry an account
    // row, so there's nothing to persist.
    return;
  }
  const secret = readEnv("IDENTITY_CALLBACK_SECRET");
  if (!secret) {
    throw new Error(
      "[identity-callback-client] IDENTITY_CALLBACK_SECRET must be set in the SSR environment.",
    );
  }
  const url = resolveCallbackUrl();
  const timestamp = String(Date.now());
  const body = serializeWireBody(event);
  const canonical = canonicalizeJSON(body);
  // Dynamic import keeps node:crypto out of the browser bundle's
  // static import graph (see module docstring).
  const { createHmac } = await import("node:crypto");
  const signature = createHmac("sha256", secret)
    .update(`${timestamp}.${canonical}`)
    .digest("base64");

  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Cachicamas-Timestamp": timestamp,
        "X-Cachicamas-Signature": signature,
      },
      body: JSON.stringify(body),
    });
  } catch (err) {
    const detail = err instanceof Error ? err.message : "network error";
    throw new Error(
      `[identity-callback-client] fetch to ${url} failed: ${detail}`,
    );
  }
  if (res.status !== 204) {
    let payload: unknown = null;
    try {
      const text = await res.text();
      if (text) {
        payload = JSON.parse(text);
      }
    } catch {
      // Non-JSON body; fall through with a generic message.
    }
    const code =
      typeof payload === "object" && payload !== null && "code" in payload
        ? String((payload as { code: unknown }).code)
        : `http_${res.status}`;
    throw new Error(
      `[identity-callback-client] signin-callback rejected (status=${res.status}, code=${code})`,
    );
  }
}

/**
 * resetPostIdentityCallbackForTest is a test-only escape hatch. It
 * resets any internal state on the client (currently no internal
 * state, but the slice reserves the hook for future use).
 */
export function resetPostIdentityCallbackForTest(): void {
  // No-op for now; reserved for future internal caches.
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

/**
 * canonicalizeJSON recursively serializes `v` to canonical JSON.
 *
 * The algorithm is the canonical-JSON-lite version shared with the Go
 * side (interfaces/http/identity_handler.go canonicalizeJSON):
 *
 *   - map keys: sorted lexicographically (NOT lowercased)
 *   - whitespace: none
 *   - string escaping: RFC 8259 (JSON.stringify default)
 *   - numbers: NOT supported by this endpoint's schema; throws on
 *     non-finite numbers as a guardrail.
 *
 * Exported only so the test file can assert byte-equality with the
 * Go-side known vector. Do NOT use this for any other purpose; the
 * canonicalizer's contract is locked and a separate `JSON.stringify`
 * helper exists for non-canonical use.
 */
export function canonicalizeJSON(v: unknown): string {
  if (v === null) return "null";
  if (typeof v === "boolean") return v ? "true" : "false";
  if (typeof v === "string") return JSON.stringify(v);
  if (typeof v === "number") {
    if (!Number.isFinite(v)) {
      throw new Error(
        "[canonicalizeJSON] non-finite number not allowed in canonical JSON for this endpoint",
      );
    }
    return JSON.stringify(v);
  }
  if (Array.isArray(v)) {
    return "[" + v.map(canonicalizeJSON).join(",") + "]";
  }
  if (typeof v === "object") {
    const obj = v as Record<string, unknown>;
    const keys = Object.keys(obj).sort();
    const pairs = keys.map(
      (k) => JSON.stringify(k) + ":" + canonicalizeJSON(obj[k]),
    );
    return "{" + pairs.join(",") + "}";
  }
  throw new Error(`[canonicalizeJSON] unsupported type: ${typeof v}`);
}

/**
 * serializeWireBody converts the Auth.js SignInEvent into the wire
 * payload (camelCase keys, the shape documented in ADR 0003). Only
 * the fields the Go handler validates are populated; OAuth tokens
 * travel through the body for forward compatibility but are NOT
 * persisted (see ADR 0003 §"Forward notes").
 */
function serializeWireBody(event: SignInEvent): Record<string, unknown> {
  // The non-null assertion is safe: postIdentityCallback returns
  // early when account is missing.
  const account = event.account as SignInAccount;
  return {
    user: {
      id: event.user.id ?? "",
      email: event.user.email ?? "",
      name: event.user.name ?? null,
      image: event.user.image ?? null,
    },
    account: {
      provider: account.provider,
      providerAccountId: account.providerAccountId,
      accessToken: account.access_token ?? null,
      refreshToken: account.refresh_token ?? null,
      expiresAt: account.expires_at ?? null,
      tokenType: account.token_type ?? null,
      scope: account.scope ?? null,
    },
  };
}

/**
 * resolveCallbackUrl picks the URL based on the env. Order:
 *   1. SERVER_API_BASE_URL (compose direct-call path) →
 *      `${SERVER_API_BASE_URL}/api/v1/identity/signin-callback`
 *   2. ORIGIN (dev reverse-proxy path) →
 *      `${ORIGIN}/api/v1/identity/signin-callback`
 *      (PUBLIC_API_BASE_URL is the legacy /api prefix used by other
 *      endpoints; the identity-callback endpoint already includes
 *      /api/v1/identity/signin-callback in its path, so we do NOT
 *      concatenate PUBLIC_API_BASE_URL here — that would double the
 *      /api prefix. See ADR 0003 §"Wire protocol" + entry.express.tsx
 *      proxyToApi for the corresponding /api/v1/* forwarding rule.)
 *
 * Returns the empty string + throws when neither is set; the caller
 * surfaces a clear error.
 */
function resolveCallbackUrl(): string {
  const serverBase = readEnv("SERVER_API_BASE_URL");
  if (serverBase && serverBase.trim().length > 0) {
    return (
      serverBase.replace(/\/+$/, "") + "/api/v1/identity/signin-callback"
    );
  }
  const origin = readEnv("ORIGIN");
  if (!origin) {
    throw new Error(
      "[identity-callback-client] cannot resolve callback URL: set SERVER_API_BASE_URL (compose) or ORIGIN (dev).",
    );
  }
  // Dev path: the URL is built directly as ${ORIGIN}/api/v1/identity/signin-callback.
  // PUBLIC_API_BASE_URL is ignored here because the identity callback
  // endpoint already lives at /api/v1/* in the backend, and the Qwik
  // Node SSR proxy forwards /api/v1/* to the backend with the /api
  // prefix preserved (see entry.express.tsx proxyToApi).
  return `${origin.replace(/\/+$/, "")}/api/v1/identity/signin-callback`;
}

/**
 * readEnv reads `key` from `process.env`. Always safe because the
 * module is server-only; never invoked from the browser bundle.
 */
function readEnv(key: string): string | undefined {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const proc = (globalThis as any).process;
  if (proc && typeof proc === "object" && proc.env) {
    const v = proc.env[key];
    return typeof v === "string" && v.length > 0 ? v : undefined;
  }
  return undefined;
}