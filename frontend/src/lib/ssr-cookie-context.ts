/**
 * SSR cookie context — captures the inbound request's Cookie header so the
 * api.ts fetch helpers can forward it to the backend during SSR.
 *
 * Why this exists (S-WS-AUTH-CHAIN-SSR-001):
 *
 * In the browser, `fetch("/api/workspaces")` is a same-origin request and the
 * browser auto-sends the `authjs.session-token` cookie. The Qwik Node server
 * proxies `/api/*` to the backend with all headers including the cookie, so
 * the backend's IdentityFromCookie middleware (introduced in commit fbe62c0)
 * sees the session.
 *
 * During SSR (Node side), `useTask$` calls `fetch("${SERVER_API_BASE_URL}/...")`
 * which hits the backend DIRECTLY, bypassing the proxy. Node's fetch does
 * NOT auto-forward the inbound request's cookies. So the backend would see
 * no cookie and 401 the request — even though the user is signed in.
 *
 * Fix: a module-level variable holds the current request's Cookie header.
 * The route's `onRequest` middleware calls `setSsrCookieHeader(...)` once
 * per request, BEFORE the auth middleware that throws on anonymous
 * requests. api.ts functions consult `getSsrCookieHeader()` and re-attach
 * the cookie to their outgoing SSR fetch.
 *
 * Why module-level state and not AsyncLocalStorage:
 *   The earlier prototype wrapped the cookie capture in an
 *   AsyncLocalStorage `.run(store, fn)`. That returned a Promise, which
 *   turned the guards' `event.redirect(...)` throws into rejected
 *   Promises. Qwik City propagates synchronous throws differently from
 *   rejected Promises from `onRequest` — the rejected-Promise path
 *   didn't short-circuit the route render, so the route component
 *   tried to write the response AFTER the redirect had been sent,
 *   crashing the Node SSR process with `Error: Response already sent`.
 *   A synchronous setter sidesteps the issue: the cookie is captured
 *   before any guard throws, and the throws themselves stay sync so
 *   Qwik handles them as designed.
 *
 * Multi-tenant caveat:
 *   The module variable is process-global; concurrent requests on the
 *   same Node instance can briefly observe each other's cookies. This
 *   is safe for cachicamas's single-tenant dev install. Multi-tenant
 *   production would replace this with `AsyncLocalStorage.enterWith`
 *   OR per-request resolver pattern.
 */

let currentRequestCookie: string | undefined;

export function setSsrCookieHeader(cookie: string | undefined): void {
  currentRequestCookie = cookie;
}

export function getSsrCookieHeader(): string | undefined {
  return currentRequestCookie;
}

/** Reset between tests; production callers don't need to invoke this. */
export function clearSsrCookieHeader(): void {
  currentRequestCookie = undefined;
}
