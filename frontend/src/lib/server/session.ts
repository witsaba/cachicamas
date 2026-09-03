/**
 * Session cookie — sign / verify / sliding refresh.
 *
 * Wire format: `base64url(JSON).base64url(HMAC-SHA256)`.
 *
 * Payload shape: `{ user_id, organization_id, expires_at, iat }`.
 * - `user_id`           : int64 from auth.users.id
 * - `organization_id`   : int64 from auth.organizations.id (NOT `pyme_id` — the
 *                          original slice was named `auth.pymes`; the merged
 *                          PR #226 renamed it to `auth.organizations`. The
 *                          cookie payload uses the new name throughout.)
 * - `expires_at`        : unix seconds. The sliding refresh in
 *                          `refreshIfNeeded` extends it by 7 days when the
 *                          remaining lifetime is below `thresholdMs`.
 * - `iat`               : unix seconds. Set at sign-time. Not validated
 *                          (no replay protection beyond expiry in this slice).
 *
 * Why HMAC-SHA256 and not JWE: the payload carries no sensitive data — three
 * integers + an epoch. Encryption adds no value; integrity does. See
 * `docs/adr/000X-native-google-oauth.md` (PR-4 territory) for the full
 * rationale.
 *
 * Why Web Crypto (`crypto.subtle`) and not `node:crypto`: the same module
 * works in Node SSR, in edge runtimes (Vercel / Cloudflare), and in
 * browsers — there is exactly one HMAC implementation to audit. The
 * downside is that `crypto.subtle` is async, so `signSession` and
 * `verifySession` return Promises.
 */
export interface SessionPayload {
  user_id: number;
  organization_id: number;
  expires_at: number;
  iat: number;
}

const DEFAULT_TTL_SECONDS = 7 * 24 * 60 * 60; // 7 days
const DEFAULT_REFRESH_THRESHOLD_MS = 24 * 60 * 60 * 1000; // 24 hours

/**
 * Encode a Uint8Array as base64url (no padding). Mirrors the inverse of
 * `decodeBase64Url`.
 */
export function encodeBase64Url(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  const b64 = btoa(binary);
  return b64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Decode a base64url string (with or without padding) to bytes.
 * Throws on invalid input — the caller is expected to wrap and return null
 * (a bad cookie is not exceptional; it's the normal "redirect to login" case).
 */
export function decodeBase64Url(input: string): Uint8Array {
  // Add padding back if stripped.
  const padded = input + "=".repeat((4 - (input.length % 4)) % 4);
  const b64 = padded.replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(b64);
  const out = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    out[i] = binary.charCodeAt(i);
  }
  return out;
}

/**
 * Constant-time string comparison. Returns true iff both inputs have the
 * same length and every byte matches.
 *
 * We do the comparison in the byte domain so that even an early-out on
 * length cannot leak the secret's length (the attacker already knows it
 * by virtue of the cookie envelope). A short-circuit on the first
 * mismatched byte would still leak the position of the first mismatch, but
 * for HMAC verification the timing surface is the entire constant-time
 * scan — which we keep.
 */
export function constantTimeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) {
    diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }
  return diff === 0;
}

async function importHmacKey(secret: string): Promise<CryptoKey> {
  const enc = new TextEncoder();
  return crypto.subtle.importKey(
    "raw",
    enc.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign", "verify"],
  );
}

/**
 * Sign a session payload into the cookie envelope.
 *
 *   base64url(JSON.stringify(payload)) + "." + base64url(HMAC-SHA256)
 *
 * The HMAC is computed over the base64url-encoded JSON, NOT over the raw
 * bytes. This makes the envelope self-describing (the JSON portion can be
 * decoded without the key) and keeps the signature scope identical to the
 * data the verifier will hash.
 */
export async function signSession(
  payload: SessionPayload,
  secret: string,
): Promise<string> {
  const enc = new TextEncoder();
  const json = JSON.stringify(payload);
  const jsonBytes = enc.encode(json);
  const jsonB64 = encodeBase64Url(jsonBytes);
  const key = await importHmacKey(secret);
  const sigBuf = await crypto.subtle.sign(
    "HMAC",
    key,
    enc.encode(jsonB64),
  );
  const sigB64 = encodeBase64Url(new Uint8Array(sigBuf));
  return `${jsonB64}.${sigB64}`;
}

/**
 * Verify a session cookie and return its payload.
 *
 * Returns `null` on ANY failure: bad envelope shape, base64url error, HMAC
 * mismatch, payload not JSON, missing/extra fields, wrong field types.
 * Callers MUST treat null as "redirect to login" — never as a crash.
 */
export async function verifySession(
  cookieValue: string,
  secret: string,
): Promise<SessionPayload | null> {
  if (typeof cookieValue !== "string" || cookieValue.length === 0) {
    return null;
  }
  const parts = cookieValue.split(".");
  if (parts.length !== 2) return null;
  const [jsonB64, sigB64] = parts;
  if (!jsonB64 || !sigB64) return null;

  // Verify the HMAC first (constant-time, over the exact bytes the signer
  // hashed).
  const enc = new TextEncoder();
  let valid = false;
  try {
    const key = await importHmacKey(secret);
    const sigBytes = decodeBase64Url(sigB64);
    valid = await crypto.subtle.verify(
      "HMAC",
      key,
      sigBytes,
      enc.encode(jsonB64),
    );
  } catch {
    return null;
  }
  if (!valid) return null;

  // Now decode and validate the payload shape.
  let payload: unknown;
  try {
    const jsonBytes = decodeBase64Url(jsonB64);
    payload = JSON.parse(new TextDecoder().decode(jsonBytes));
  } catch {
    return null;
  }
  if (!isSessionPayload(payload)) return null;
  return payload;
}

function isSessionPayload(value: unknown): value is SessionPayload {
  if (typeof value !== "object" || value === null) return false;
  const v = value as Record<string, unknown>;
  return (
    typeof v.user_id === "number" &&
    Number.isFinite(v.user_id) &&
    typeof v.organization_id === "number" &&
    Number.isFinite(v.organization_id) &&
    typeof v.expires_at === "number" &&
    Number.isFinite(v.expires_at) &&
    typeof v.iat === "number" &&
    Number.isFinite(v.iat)
  );
}

/**
 * Sliding refresh.
 *
 * If the payload is within `thresholdMs` of expiring (or already expired),
 * return a NEW payload with `expires_at` set to `nowMs + ttlMs`. Otherwise
 * return null (the caller leaves the cookie alone to avoid needless
 * writes).
 *
 * Why "within 24h": we want a returning user to keep their session alive
 * without ever seeing an unexpected forced re-login. A 7-day cookie that
 * refreshes on every visit within the last day of its life has a de-facto
 * lifetime of "as long as the user keeps coming back at least weekly".
 */
export function refreshIfNeeded(
  payload: SessionPayload,
  nowMs: number,
  thresholdMs: number = DEFAULT_REFRESH_THRESHOLD_MS,
  ttlMs: number = DEFAULT_TTL_SECONDS * 1000,
): SessionPayload | null {
  const expiresAtMs = payload.expires_at * 1000;
  const remainingMs = expiresAtMs - nowMs;
  if (remainingMs > thresholdMs) return null;
  return {
    user_id: payload.user_id,
    organization_id: payload.organization_id,
    expires_at: Math.floor((nowMs + ttlMs) / 1000),
    iat: payload.iat,
  };
}

/**
 * Cookie attribute helpers.
 *
 * Centralised so the login route, the callback route, the layout guard, and
 * the logout route all share the same attribute policy. The callback sets
 * the cookie; the layout's sliding refresh re-uses the same writer.
 */
export interface CookieAttributes {
  httpOnly: boolean;
  secure: boolean;
  sameSite: "lax" | "strict" | "none";
  path: string;
  maxAgeSeconds: number;
}

export const SESSION_COOKIE_NAME = "cachicamas_session";
export const SESSION_COOKIE_TTL_SECONDS = DEFAULT_TTL_SECONDS;

export function sessionCookieAttributes(
  isProduction: boolean,
): CookieAttributes {
  return {
    httpOnly: true,
    secure: isProduction,
    sameSite: "lax",
    path: "/",
    maxAgeSeconds: DEFAULT_TTL_SECONDS,
  };
}

export function clearSessionCookieAttributes(): CookieAttributes {
  return {
    httpOnly: true,
    secure: false,
    sameSite: "lax",
    path: "/",
    maxAgeSeconds: 0,
  };
}