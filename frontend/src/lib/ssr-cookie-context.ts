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
 * Fix: a middleware (mounted as `onRequest` in the workspace route files)
 * captures the cookie header from `event.request.headers` and stashes it in
 * an AsyncLocalStorage store scoped to the current request. api.ts
 * functions, when running in SSR, read from that store and re-attach the
 * cookie header to their outgoing fetch.
 *
 * Why globalThis + dynamic import instead of a top-level
 * `import { AsyncLocalStorage } from "node:async_hooks"`:
 *   api.ts is reachable from both the SSR server bundle and the
 *   browser client bundle. A static `node:` import would fail the
 *   client build with "Module 'node:async_hooks' has been externalised
 *   for browser compatibility" (mirrors identity-callback-client.ts).
 *   The AsyncLocalStorage instance lives on `globalThis` and is
 *   created lazily on first use on the server; the client never calls
 *   the SSR helper path so the import never resolves there.
 */

interface SsrCookieContext {
  cookieHeader: string;
}

interface GlobalShape {
  /** Lazily-initialised on first use on the server. */
  __cachicamasSsrCookieStore?: {
    run<T>(value: SsrCookieContext, fn: () => T): T;
    getStore(): SsrCookieContext | undefined;
  };
}

// Store is created lazily because the AsyncLocalStorage class is
// resolved via dynamic import on the server (Node only).
async function ensureStore(): Promise<NonNullable<GlobalShape["__cachicamasSsrCookieStore"]>> {
  const g = globalThis as unknown as GlobalShape;
  if (g.__cachicamasSsrCookieStore) return g.__cachicamasSsrCookieStore;
  // Dynamic import keeps `node:async_hooks` out of the browser bundle.
  const { AsyncLocalStorage } = await import("node:async_hooks");
  const store = new AsyncLocalStorage<SsrCookieContext>();
  g.__cachicamasSsrCookieStore = store;
  return store;
}

/**
 * Reads the inbound-cookie context established by `withSsrCookieContext`.
 * Returns `undefined` when called from the browser (no store set) so the
 * api helper can skip the cookie header attachment in that path.
 *
 * Synchronous read is required because api.ts fetch helpers run inside
 * `await fetch(...)` callers that don't tolerate extra `await`s here.
 * The store is set up once during startup via `initSsrCookieContext`
 * (called from the onRequest middleware), and a sync read of the
 * globalThis pointer is enough.
 */
export function getSsrCookieHeader(): string | undefined {
  const g = globalThis as unknown as GlobalShape;
  return g.__cachicamasSsrCookieStore?.getStore()?.cookieHeader;
}

/**
 * Runs the given function with the inbound cookie header available to
 * `getSsrCookieHeader()`. Called from the route's `onRequest`
 * middleware so all downstream `useTask$` fetches see the cookie.
 */
export async function runWithSsrCookie<T>(
  cookieHeader: string,
  fn: () => T | Promise<T>,
): Promise<T> {
  const store = await ensureStore();
  return store.run({ cookieHeader }, () => fn());
}
