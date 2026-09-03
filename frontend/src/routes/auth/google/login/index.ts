/**
 * `/auth/google/login` — issue an OAuth state, store it in a short-lived
 * cookie, and 302 to Google.
 *
 * Spec reference: R-FE-001 / R-OAUTH-1. The browser-side flow is:
 *   1. visitor hits `/auth/google/login` (or is redirected here by the
 *      `(app)/layout.tsx` guard when their session cookie is missing);
 *   2. THIS route handler generates a 32-byte base64url state, persists it
 *      in `cachicamas_oauth_state` (HttpOnly, Secure when in prod,
 *      SameSite=Lax, 10-minute TTL), and 302-redirects to Google's OAuth
 *      endpoint with `state`, `scope`, and `redirect_uri`;
 *   3. after Google consent, the browser lands at `/auth/google/callback`
 *      with `code` + `state` query params.
 *
 * The state cookie is the CSRF defence: the callback compares the cookie
 * value against the query value, and a mismatch ⇒
 * `/auth/error?reason=invalid_state`.
 *
 * The pure logic lives in `handleLoginGet` (exported alongside the Qwik
 * handler) so the test suite can call it with a plain mock event instead
 * of booting the Qwik server runtime. The Qwik `onGet` wrapper unpacks
 * `RequestEvent` into the simpler `LoginInput` shape.
 */
import type { RequestHandler } from "@builder.io/qwik-city";
import { buildAuthUrl, generateState } from "~/lib/server/oauth";

export const OAUTH_STATE_COOKIE = "cachicamas_oauth_state";
const OAUTH_STATE_TTL_SECONDS = 10 * 60; // 10 minutes

export interface LoginInput {
  /** Sets the state cookie. Production wrapper passes Qwik's `cookie`. */
  setStateCookie: (
    name: string,
    value: string,
    opts: {
      httpOnly: boolean;
      secure: boolean;
      sameSite: "lax" | "strict" | "none";
      path: string;
      maxAge: number;
    },
  ) => void;
  /**
   * The redirect the handler should throw. Production wrapper passes
   * Qwik's `redirect(302, url)`; tests pass a function that captures the
   * url and throws a sentinel.
   */
  doRedirect: (url: string) => never;
  /** `AUTH_GOOGLE_ID` — set in `.env.example` / docker-compose (PR-4). */
  clientId: string;
  /** `PUBLIC_AUTH_REDIRECT_URI` — must match the Google Cloud console. */
  redirectUri: string;
  /** `NODE_ENV === "production"` gates the `Secure` cookie attribute. */
  isProduction: boolean;
  /**
   * State generator. Default: `generateState`. Override only in tests
   * that want a deterministic state value.
   */
  generateStateFn?: () => string;
}

/**
 * Pure login handler. Returns the Google OAuth URL the browser would be
 * redirected to (also throws via `doRedirect` so the caller can rely on
 * Qwik's redirect semantics).
 *
 * Sets `cachicamas_oauth_state` cookie to the generated state, scoped to
 * `/`, with HttpOnly + SameSite=Lax always, and `Secure` only when
 * `isProduction` (dev compose over plain HTTP must NOT set Secure or
 * the browser refuses to round-trip the cookie).
 */
export function handleLoginGet(input: LoginInput): never {
  if (!input.clientId) {
    throw new Error("AUTH_GOOGLE_ID env is required");
  }
  if (!input.redirectUri) {
    throw new Error("PUBLIC_AUTH_REDIRECT_URI env is required");
  }
  const state = (input.generateStateFn ?? generateState)();
  input.setStateCookie(OAUTH_STATE_COOKIE, state, {
    httpOnly: true,
    secure: input.isProduction,
    sameSite: "lax",
    path: "/",
    maxAge: OAUTH_STATE_TTL_SECONDS,
  });
  const url = buildAuthUrl({
    clientId: input.clientId,
    redirectUri: input.redirectUri,
    state,
  });
  input.doRedirect(url.toString());
  // Unreachable — `doRedirect` throws. The return type is `never` for
  // callers that statically know this is a terminal handler.
  throw new Error("unreachable: doRedirect must throw");
}

/**
 * Qwik City request handler. Reads env via `ev.env.get(...)`, sets the
 * state cookie on the response, and 302s to Google.
 */
export const onGet: RequestHandler = (ev) => {
  handleLoginGet({
    setStateCookie: (name, value, opts) => {
      // CookieOptions on Qwik's Cookie uses `maxAge` in seconds.
      ev.cookie.set(name, value, opts);
    },
    doRedirect: (url) => ev.redirect(302, url),
    clientId: ev.env.get("AUTH_GOOGLE_ID") ?? "",
    redirectUri:
      ev.env.get("PUBLIC_AUTH_REDIRECT_URI") ??
      "http://localhost:5173/auth/google/callback",
    isProduction: ev.env.get("NODE_ENV") === "production",
  });
};