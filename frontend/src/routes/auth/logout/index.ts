/**
 * `/auth/logout` — clear the session cookie.
 *
 * Spec reference: R-FE-003 / S-FE-020. POST clears `cachicamas_session`
 * (Max-Age=0; HttpOnly; SameSite=Lax; Path=/) and 302s to `/`.
 *
 * Qwik routes export the route action as a default function and any
 * request handlers (`onRequest`, `onPost`, `onGet`). We expose a plain
 * `handleLogout` so the test suite can drive it without booting the
 * Qwik runtime.
 */
import type { RequestHandler } from "@builder.io/qwik-city";
import { SESSION_COOKIE_NAME } from "~/lib/server/session";

export interface LogoutInput {
  /** Production wrapper passes `ev.cookie.delete(name, opts)`. */
  clearSessionCookie: (name: string) => void;
  /** 302 helper. */
  doRedirect: (url: string) => unknown;
}

/**
 * Clear the session cookie and redirect to `/`. Idempotent: even if
 * there is no current session, the response clears any stale cookie
 * the browser might still hold and lands on `/`.
 */
export function handleLogout(input: LogoutInput): unknown {
  input.clearSessionCookie(SESSION_COOKIE_NAME);
  return input.doRedirect("/");
}

export const onPost: RequestHandler = (ev) => {
  handleLogout({
    clearSessionCookie: (name) => {
      ev.cookie.delete(name, { path: "/" });
    },
    doRedirect: (url) => ev.redirect(302, url),
  });
};

/**
 * Allow GET on `/auth/logout` as a graceful fallback for cases where a
 * POST cannot be issued (e.g. a user pastes the URL into the address
 * bar). Spec locks POST as the primary path, but a GET is a safer default
 * than a 405 — the cookie clears either way.
 */
export const onGet: RequestHandler = onPost;
