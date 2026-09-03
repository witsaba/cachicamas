/**
 * `(app)/layout.tsx` — auth guard.
 *
 * Spec reference: R-LAYOUT-1. The `(app)` route group covers every
 * authenticated surface (`/home`, and any future workspace pages).
 * The layout:
 *   1. reads `cachicamas_session` from the request cookie;
 *   2. verifies the HMAC via `verifySession`;
 *   3. on failure/absence/expired ⇒ 302 to `/auth/google/login`;
 *   4. on success, optionally re-signs the cookie when within 24h of
 *      expiry (sliding refresh per R-NFR-002) and exposes
 *      `{ user_id, organization_id }` to child routes via `useContext`
 *      (currently we just stash the values in `sharedMap` so the
 *      `/home` page can read them via `routeLoader$`).
 *
 * The pure logic lives in `guardLayout` (exported). The Qwik
 * `onRequest` wrapper unpacks `RequestEvent` into `GuardInput`.
 *
 * The Qwik (app) layout is a group layout (parens in the path), so its
 * default export is a `component$` that renders `<Slot />`. The guard
 * runs in `onRequest` (BEFORE the page's `routeLoader$`), so a redirect
 * thrown there prevents the page from rendering.
 */
import { Slot, component$ } from "@builder.io/qwik";
import type { RequestHandler } from "@builder.io/qwik-city";
import {
  refreshIfNeeded,
  SESSION_COOKIE_NAME,
  signSession,
  verifySession,
  type SessionPayload,
} from "~/lib/server/session";

export const SHARED_MAP_SESSION_KEY = "session";
export const LOGIN_ROUTE = "/auth/google/login";

export interface GuardInput {
  /** Raw cookie value (e.g. `cachicamas_session`'s `cookie.get(...)?.value`). */
  sessionCookieValue: string | null;
  /** `verifySession` (injected so tests can stub it). */
  verifySession: (
    cookie: string,
    secret: string,
  ) => Promise<SessionPayload | null>;
  /** `signSession` (injected so tests can stub it). */
  signSession: (
    payload: SessionPayload,
    secret: string,
  ) => Promise<string>;
  /** `refreshIfNeeded` (injected so tests can stub it). */
  refreshIfNeeded: typeof refreshIfNeeded;
  /** `AUTH_COOKIE_SECRET`. */
  cookieSecret: string;
  /** `NODE_ENV === "production"`. */
  isProduction: boolean;
  /** Qwik cookie writer for the sliding refresh (only called when refresh needed). */
  setSessionCookie?: (
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
  /** Shared map for child loaders (Qwik provides this on RequestEvent). */
  sharedMap?: Map<string, unknown>;
  /** 302 helper. */
  doRedirect: (url: string) => never;
  /** Current time in ms (test-injectable). */
  nowMs?: () => number;
}

export interface GuardOutcome {
  /** Where the user ended up. `"continue"` means the child page renders. */
  decision: "continue" | "redirect";
  /** The redirect URL when `decision === "redirect"`. */
  redirectTo?: string;
  /** The validated session (when `decision === "continue"`). */
  session?: SessionPayload;
  /** Whether the cookie was re-issued (sliding refresh). */
  refreshed?: boolean;
}

/**
 * Pure guard. Returns an outcome descriptor; the Qwik `onRequest`
 * wrapper applies the redirect (or stores the session in `sharedMap`).
 */
export async function guardLayout(input: GuardInput): Promise<GuardOutcome> {
  const raw = input.sessionCookieValue;
  if (!raw) {
    return redirectToLogin(input);
  }
  const session = await input.verifySession(raw, input.cookieSecret);
  if (!session) {
    return redirectToLogin(input);
  }
  if (session.expires_at * 1000 <= (input.nowMs?.() ?? Date.now())) {
    return redirectToLogin(input);
  }

  // Sliding refresh: if the cookie is within 24h of expiry, re-sign
  // with a fresh 7-day expiry. The handler decides whether to actually
  // issue the Set-Cookie header (the production wrapper wires it up).
  const refreshed = input.refreshIfNeeded(
    session,
    input.nowMs?.() ?? Date.now(),
  );
  if (refreshed && input.setSessionCookie) {
    const newValue = await input.signSession(refreshed, input.cookieSecret);
    input.setSessionCookie(SESSION_COOKIE_NAME, newValue, {
      httpOnly: true,
      secure: input.isProduction,
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 24 * 7, // 7 days
    });
    if (input.sharedMap) {
      input.sharedMap.set(SHARED_MAP_SESSION_KEY, refreshed);
    }
    return { decision: "continue", session: refreshed, refreshed: true };
  }

  if (input.sharedMap) {
    input.sharedMap.set(SHARED_MAP_SESSION_KEY, session);
  }
  return { decision: "continue", session, refreshed: false };
}

function redirectToLogin(input: GuardInput): GuardOutcome {
  input.doRedirect(LOGIN_ROUTE);
  return { decision: "redirect", redirectTo: LOGIN_ROUTE };
}

/**
 * Qwik `onRequest` for the `(app)` route group. Runs BEFORE the page's
 * `routeLoader$` and `component$`. If it throws a redirect (via
 * `ev.redirect`), the child page never renders.
 */
export const onRequest: RequestHandler = async (ev) => {
  const cookieValue = ev.cookie.get(SESSION_COOKIE_NAME)?.value ?? null;
  const outcome = await guardLayout({
    sessionCookieValue: cookieValue,
    verifySession,
    signSession,
    refreshIfNeeded,
    cookieSecret: ev.env.get("AUTH_COOKIE_SECRET") ?? "",
    isProduction: ev.env.get("NODE_ENV") === "production",
    setSessionCookie: (name, value, opts) => {
      ev.cookie.set(name, value, {
        httpOnly: opts.httpOnly,
        secure: opts.secure,
        sameSite: opts.sameSite,
        path: opts.path,
        maxAge: opts.maxAge,
      });
    },
    sharedMap: ev.sharedMap,
    doRedirect: (url) => ev.redirect(302, url),
  });
  if (outcome.decision === "redirect") {
    // doRedirect threw; this line is unreachable in production.
    return;
  }
  // Stash for child routeLoaders that need the validated session.
  ev.sharedMap.set(SHARED_MAP_SESSION_KEY, outcome.session);
};

/**
 * The visible layout. Renders `<Slot />` so child routes (`/home`,
 * future workspace pages) mount inside the chrome.
 *
 * The (app) group has no shared chrome yet — the workspace will add
 * one in a follow-up. For PR-3, the layout is intentionally invisible
 * (just a Slot).
 */
export default component$(() => {
  return <Slot />;
});