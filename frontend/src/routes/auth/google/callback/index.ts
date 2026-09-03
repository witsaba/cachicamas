/**
 * `/auth/google/callback` — finish the OAuth round-trip.
 *
 * Steps (per design §1.1, sequence diagram):
 *   1. Read `code` + `state` from the query string and the
 *      `cachicamas_oauth_state` cookie. Mismatch ⇒ redirect to
 *      `/auth/error?reason=invalid_state`.
 *   2. Exchange `code` for tokens at `https://oauth2.googleapis.com/token`.
 *      Non-2xx ⇒ `/auth/error?reason=token_exchange_failed`.
 *   3. Fetch `/userinfo` with the access token.
 *      Non-2xx ⇒ `/auth/error?reason=userinfo_failed`.
 *   4. POST `/internal/auth/bootstrap` with `X-Internal-Secret` from
 *      server env. The backend is the source of truth for
 *      `{user_id, organization_id, status}`.
 *   5. Inspect `status`:
 *      - `blocked`     ⇒ `/auth/error?reason=blocked`, NO session cookie
 *      - `active`/`inactive`/anything-else ⇒ continue
 *   6. Sign a session payload with HMAC-SHA256, set the
 *      `cachicamas_session` cookie (HttpOnly, SameSite=Lax, 7-day TTL),
 *      and 302 to `/home`.
 *
 * Spec reference: R-FE-002 (callback), R-FE-009 (blocked rejection),
 * R-FE-010 (inactive allows login), R-BOOTSTRAP-1 (backend contract).
 *
 * The pure logic lives in `handleCallbackGet` (exported). The Qwik
 * `onGet` wrapper unpacks `RequestEvent` into `CallbackInput`. This lets
 * the test suite drive the route end-to-end with a mock `fetch` and
 * captured cookie sets, instead of standing up the Qwik server.
 */
import type { RequestHandler } from "@builder.io/qwik-city";
import {
  callBackendBootstrap,
  exchangeCode,
  fetchUserInfo,
  OAuthExchangeError,
  OAuthUserInfoError,
  BootstrapError,
  type GoogleClaims,
  type BootstrapResult,
} from "~/lib/server/oauth";
import {
  SESSION_COOKIE_NAME,
  SESSION_COOKIE_TTL_SECONDS,
  signSession,
  type SessionPayload,
  type CookieAttributes,
} from "~/lib/server/session";
import { OAUTH_STATE_COOKIE } from "~/routes/auth/google/login";

/** Locked reason codes per spec R-FE-006 / R-FE-009. */
export type CallbackErrorReason =
  | "invalid_state"
  | "token_exchange_failed"
  | "userinfo_failed"
  | "internal_error"
  | "blocked"
  | "missing_code";

export interface CallbackInput {
  /** Query params the browser sent (e.g. `code`, `state`). */
  query: URLSearchParams;
  /**
   * Full URL of the incoming request (used to derive `redirect_uri`).
   * Production wrapper passes `ev.url.toString()`.
   */
  url: string;
  /** Value of the `cachicamas_oauth_state` cookie (or null). */
  stateCookieValue: string | null;
  /**
   * Sets the session cookie. Production wrapper passes
   * `ev.cookie.set(name, value, opts)`; tests capture the call.
   */
  setSessionCookie: (
    name: string,
    value: string,
    opts: CookieAttributes,
  ) => void;
  /** Clears the OAuth state cookie (always called on a successful path). */
  clearStateCookie: () => void;
  /** 302 helper. */
  doRedirect: (url: string) => never;
  /** `error(...)` helper for hard 500s that cannot redirect (no path recovery). */
  doError: (status: number, message: string) => never;
  /** `AUTH_GOOGLE_ID` / `AUTH_GOOGLE_SECRET` / `AUTH_COOKIE_SECRET` / `AUTH_INTERNAL_SECRET`. */
  env: {
    clientId: string;
    clientSecret: string;
    cookieSecret: string;
    internalSecret: string;
    backendUrl: string;
  };
  /** `NODE_ENV === "production"` gates the `Secure` cookie attribute. */
  isProduction: boolean;
  /** Override for tests. Default: `fetch`. */
  fetchImpl?: typeof fetch;
  /** Override for tests. Default: `Date.now` (ms). */
  nowMs?: () => number;
}

export interface CallbackOutcome {
  /** Where the browser ended up. */
  redirectTo: string;
  /**
   * What the handler did. Surfaced for tests; production code just
   * throws the redirect.
   */
  reason:
    | "ok"
    | "invalid_state"
    | "missing_code"
    | "token_exchange_failed"
    | "userinfo_failed"
    | "internal_error"
    | "blocked";
  /** The cookie value the handler would have set (when reason === "ok"). */
  sessionCookieValue?: string;
}

/**
 * Pure callback handler. Performs the whole OAuth round-trip.
 *
 * Returns an outcome descriptor for tests; production callers rely on
 * `doRedirect` to throw (Qwik terminates the request) and never see
 * the return value.
 */
export async function handleCallbackGet(
  input: CallbackInput,
): Promise<CallbackOutcome> {
  // 1. Read & verify state.
  const code = input.query.get("code");
  const queryState = input.query.get("state");

  if (!code) {
    return redirectWith(
      input,
      `/auth/error?reason=${"missing_code" satisfies CallbackErrorReason}`,
      "missing_code",
    );
  }

  if (!input.stateCookieValue || !queryState || input.stateCookieValue !== queryState) {
    return redirectWith(
      input,
      `/auth/error?reason=${"invalid_state" satisfies CallbackErrorReason}`,
      "invalid_state",
    );
  }

  const now = () => input.nowMs?.() ?? Date.now();

  // 2. Exchange code → tokens.
  let accessToken: string;
  try {
    const tokenResp = await exchangeCode({
      code,
      clientId: input.env.clientId,
      clientSecret: input.env.clientSecret,
      redirectUri: new URL(input.url).origin + "/auth/google/callback",
      fetchImpl: input.fetchImpl,
    });
    accessToken = tokenResp.access_token;
  } catch (err) {
    if (err instanceof OAuthExchangeError) {
      return redirectWith(
        input,
        `/auth/error?reason=${"token_exchange_failed" satisfies CallbackErrorReason}`,
        "token_exchange_failed",
      );
    }
    return redirectWith(
      input,
      `/auth/error?reason=${"internal_error" satisfies CallbackErrorReason}`,
      "internal_error",
    );
  }

  // 3. Fetch userinfo.
  let claims: GoogleClaims;
  try {
    claims = await fetchUserInfo(accessToken, input.fetchImpl);
  } catch (err) {
    if (err instanceof OAuthUserInfoError) {
      return redirectWith(
        input,
        `/auth/error?reason=${"userinfo_failed" satisfies CallbackErrorReason}`,
        "userinfo_failed",
      );
    }
    return redirectWith(
      input,
      `/auth/error?reason=${"internal_error" satisfies CallbackErrorReason}`,
      "internal_error",
    );
  }

  // 4. Bootstrap on the backend.
  let bootstrap: BootstrapResult;
  try {
    bootstrap = await callBackendBootstrap({
      backendUrl: input.env.backendUrl,
      internalSecret: input.env.internalSecret,
      claims,
      fetchImpl: input.fetchImpl,
    });
  } catch (err) {
    if (err instanceof BootstrapError) {
      return redirectWith(
        input,
        `/auth/error?reason=${"internal_error" satisfies CallbackErrorReason}`,
        "internal_error",
      );
    }
    return redirectWith(
      input,
      `/auth/error?reason=${"internal_error" satisfies CallbackErrorReason}`,
      "internal_error",
    );
  }

  // 5. Reject blocked users. NEVER set a session for a blocked account.
  if (bootstrap.status === "blocked") {
    return redirectWith(
      input,
      `/auth/error?reason=${"blocked" satisfies CallbackErrorReason}`,
      "blocked",
    );
  }

  // 6. Sign session cookie.
  const ttlSeconds = SESSION_COOKIE_TTL_SECONDS;
  const payload: SessionPayload = {
    user_id: bootstrap.user_id,
    organization_id: bootstrap.organization_id,
    expires_at: Math.floor(now() / 1000) + ttlSeconds,
    iat: Math.floor(now() / 1000),
  };
  const sessionCookieValue = await signSession(payload, input.env.cookieSecret);

  input.setSessionCookie(SESSION_COOKIE_NAME, sessionCookieValue, {
    httpOnly: true,
    secure: input.isProduction,
    sameSite: "lax",
    path: "/",
    maxAgeSeconds: ttlSeconds,
  });
  input.clearStateCookie();
  input.doRedirect("/home");
  // Unreachable; production redirect throws.
  return {
    redirectTo: "/home",
    reason: "ok",
    sessionCookieValue,
  };
}

function redirectWith(
  input: CallbackInput,
  url: string,
  reason: CallbackOutcome["reason"],
): CallbackOutcome {
  input.doRedirect(url);
  // Unreachable; doRedirect throws.
  return { redirectTo: url, reason };
}

/**
 * Qwik City request handler. The 6-line shim that unpacks `RequestEvent`
 * into `CallbackInput` and calls the pure handler.
 */
export const onGet: RequestHandler = (ev) => {
  const url = ev.url.toString();
  void handleCallbackGet({
    query: ev.url.searchParams,
    stateCookieValue: ev.cookie.get(OAUTH_STATE_COOKIE)?.value ?? null,
    setSessionCookie: (name, value, opts) => {
      // Qwik's CookieOptions uses `maxAge` (seconds). We accept `maxAgeSeconds`.
      ev.cookie.set(name, value, {
        httpOnly: opts.httpOnly,
        secure: opts.secure,
        sameSite: opts.sameSite,
        path: opts.path,
        maxAge: opts.maxAgeSeconds,
      });
    },
    clearStateCookie: () => {
      ev.cookie.delete(OAUTH_STATE_COOKIE, { path: "/" });
    },
    doRedirect: (url) => ev.redirect(302, url),
    doError: (status, message) => ev.error(status, message),
    env: {
      clientId: ev.env.get("AUTH_GOOGLE_ID") ?? "",
      clientSecret: ev.env.get("AUTH_GOOGLE_SECRET") ?? "",
      cookieSecret: ev.env.get("AUTH_COOKIE_SECRET") ?? "",
      internalSecret: ev.env.get("AUTH_INTERNAL_SECRET") ?? "",
      backendUrl:
        ev.env.get("PUBLIC_GO_BACKEND_URL") ?? "http://localhost:8080",
    },
    isProduction: ev.env.get("NODE_ENV") === "production",
    url,
  }).then(() => {
    /* unreachable in production — doRedirect throws */
  });
};