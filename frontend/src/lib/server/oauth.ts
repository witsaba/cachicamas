/**
 * Google OAuth — URL builder + state generator + token exchange + userinfo.
 *
 * Server-only module. The browser NEVER imports this file (it's pinned
 * under `lib/server/`, which is excluded from the client bundle by the
 * Vite `routeLoader$` boundary — Qwik never tree-shakes `lib/server/`
 * into a client chunk).
 *
 * Spec reference: R-OAUTH-1 (`/auth/google/login` → 302 Google;
 * `/callback` state verify; bootstrap call; cookie set).
 *
 * MVP simplification: PKCE (code_challenge / code_verifier) is intentionally
 * OMITTED. The OAuth client is confidential and server-side; the
 * client_secret never leaves Node SSR. PKCE is a defence-in-depth that
 * only matters if the secret leaks — if it leaks, the attacker can already
 * exchange codes. Add later if a public-client (mobile/SPA) surface ships.
 */
export const GOOGLE_AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth";
export const GOOGLE_TOKEN_URL = "https://oauth2.googleapis.com/token";
export const GOOGLE_USERINFO_URL =
  "https://www.googleapis.com/oauth2/v2/userinfo";

export const DEFAULT_SCOPES: readonly string[] = ["openid", "email", "profile"];

export interface BuildAuthUrlOptions {
  clientId: string;
  redirectUri: string;
  state: string;
  scopes?: readonly string[];
}

export interface GoogleClaims {
  google_sub: string;
  email: string;
  email_verified: boolean;
  name: string;
  picture_url: string;
}

export interface TokenResponse {
  access_token: string;
  id_token?: string;
  expires_in?: number;
  token_type?: string;
}

/**
 * Generate a 32-byte cryptographically random state string, base64url
 * encoded. The OAuth `state` is the CSRF defence: it round-trips
 * Google → callback and the callback compares the cookie to the query
 * string value.
 *
 * Rejection sampling is not needed — `crypto.getRandomValues` over
 * 32 bytes is fine.
 */
export function generateState(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return encodeBase64Url(bytes);
}

function encodeBase64Url(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  const b64 = btoa(binary);
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Build the Google OAuth authorization URL. The caller is responsible for
 * persisting `state` in a short-lived cookie before the redirect fires.
 *
 * Deterministic: same inputs produce same output (modulo the caller
 * passing a fresh `state`). Easy to test.
 */
export function buildAuthUrl(opts: BuildAuthUrlOptions): URL {
  if (!opts.clientId) throw new Error("buildAuthUrl: clientId required");
  if (!opts.redirectUri)
    throw new Error("buildAuthUrl: redirectUri required");
  if (!opts.state) throw new Error("buildAuthUrl: state required");
  const url = new URL(GOOGLE_AUTH_URL);
  url.searchParams.set("client_id", opts.clientId);
  url.searchParams.set("redirect_uri", opts.redirectUri);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", (opts.scopes ?? DEFAULT_SCOPES).join(" "));
  url.searchParams.set("state", opts.state);
  // PKCE intentionally omitted for MVP — see file header.
  return url;
}

/**
 * Exchange the authorization code for an access token. Uses native fetch
 * so it works in Node SSR, edge runtimes, and browsers (the latter is
 * never reached for this function — see file header).
 *
 * Throws on non-2xx (the caller maps to `/auth/error?reason=...`).
 */
export async function exchangeCode(opts: {
  code: string;
  clientId: string;
  clientSecret: string;
  redirectUri: string;
  fetchImpl?: typeof fetch;
}): Promise<TokenResponse> {
  const fetchImpl = opts.fetchImpl ?? fetch;
  const body = new URLSearchParams({
    code: opts.code,
    client_id: opts.clientId,
    client_secret: opts.clientSecret,
    redirect_uri: opts.redirectUri,
    grant_type: "authorization_code",
  });
  const res = await fetchImpl(GOOGLE_TOKEN_URL, {
    method: "POST",
    headers: { "content-type": "application/x-www-form-urlencoded" },
    body,
  });
  if (!res.ok) {
    throw new OAuthExchangeError(
      `token endpoint returned ${res.status}`,
      res.status,
    );
  }
  const data = (await res.json()) as Record<string, unknown>;
  if (typeof data.access_token !== "string") {
    throw new OAuthExchangeError(
      "token response missing access_token",
      res.status,
    );
  }
  return {
    access_token: data.access_token,
    id_token: typeof data.id_token === "string" ? data.id_token : undefined,
    expires_in: typeof data.expires_in === "number" ? data.expires_in : undefined,
    token_type: typeof data.token_type === "string" ? data.token_type : undefined,
  };
}

/**
 * Fetch the user's profile from Google's `/userinfo` endpoint using the
 * access token from the token endpoint. Maps the Google response to our
 * `GoogleClaims` shape (which the callback then forwards to the Go
 * backend's bootstrap endpoint).
 *
 * Throws on non-2xx (the caller maps to `/auth/error?reason=...`).
 */
export async function fetchUserInfo(
  accessToken: string,
  fetchImpl: typeof fetch = fetch,
): Promise<GoogleClaims> {
  const res = await fetchImpl(GOOGLE_USERINFO_URL, {
    headers: { authorization: `Bearer ${accessToken}` },
  });
  if (!res.ok) {
    throw new OAuthUserInfoError(
      `userinfo returned ${res.status}`,
      res.status,
    );
  }
  const data = (await res.json()) as Record<string, unknown>;
  const sub = data.id ?? data.sub;
  const email = data.email;
  if (typeof sub !== "string" || sub.length === 0) {
    throw new OAuthUserInfoError("userinfo missing sub/id", res.status);
  }
  if (typeof email !== "string" || email.length === 0) {
    throw new OAuthUserInfoError("userinfo missing email", res.status);
  }
  return {
    google_sub: sub,
    email,
    email_verified: data.email_verified === true,
    name: typeof data.name === "string" ? data.name : "",
    picture_url: typeof data.picture === "string" ? data.picture : "",
  };
}

/**
 * POST the Google claims to the backend bootstrap endpoint. The backend
 * is the source of truth for `{ user_id, organization_id, status }`
 * (see PR-2 `auth_handler.go::Bootstrap`).
 *
 * Requires `X-Internal-Secret` from the env (set per PR-4 in
 * `.env.example` / `docker-compose.yaml`).
 *
 * Throws on non-2xx or on a missing `secret` / `backendUrl`.
 */
export async function callBackendBootstrap(opts: {
  backendUrl: string;
  internalSecret: string;
  claims: GoogleClaims;
  fetchImpl?: typeof fetch;
}): Promise<BootstrapResult> {
  if (!opts.backendUrl) throw new Error("callBackendBootstrap: backendUrl required");
  if (!opts.internalSecret)
    throw new Error("callBackendBootstrap: internalSecret required");
  const fetchImpl = opts.fetchImpl ?? fetch;
  // The backend lives at /internal/auth/bootstrap (mounted under /api/* by
  // the Node proxy in entry.express.tsx). The proxy strips /api — see
  // `lib/api-router.ts`. From the SSR runtime we call the backend
  // DIRECTLY (not via the proxy) on the compose network. The path here is
  // the canonical backend path, NOT the proxy-prefixed path.
  const url = new URL("/internal/auth/bootstrap", opts.backendUrl).toString();
  const res = await fetchImpl(url, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      "x-internal-secret": opts.internalSecret,
    },
    body: JSON.stringify({
      google_sub: opts.claims.google_sub,
      email: opts.claims.email,
      email_verified: opts.claims.email_verified,
      name: opts.claims.name,
      picture_url: opts.claims.picture_url,
    }),
  });
  if (!res.ok) {
    throw new BootstrapError(
      `backend bootstrap returned ${res.status}`,
      res.status,
    );
  }
  const data = (await res.json()) as Record<string, unknown>;
  if (
    typeof data.user_id !== "number" ||
    typeof data.organization_id !== "number" ||
    typeof data.status !== "string"
  ) {
    throw new BootstrapError(
      "backend bootstrap response missing required fields",
      res.status,
    );
  }
  return {
    user_id: data.user_id,
    organization_id: data.organization_id,
    status: data.status,
  };
}

export interface BootstrapResult {
  user_id: number;
  organization_id: number;
  status: string;
}

export class OAuthExchangeError extends Error {
  readonly status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
    this.name = "OAuthExchangeError";
  }
}

export class OAuthUserInfoError extends Error {
  readonly status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
    this.name = "OAuthUserInfoError";
  }
}

export class BootstrapError extends Error {
  readonly status: number;
  constructor(message: string, status: number) {
    super(message);
    this.status = status;
    this.name = "BootstrapError";
  }
}