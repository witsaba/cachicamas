/**
 * `withSsrCookieContext` — request-handler wrapper that makes the inbound
 * Cookie header available to api.ts fetches in `useTask$` during SSR.
 *
 * Reference (S-WS-AUTH-CHAIN-SSR-001): see `ssr-cookie-context.ts` for
 * the long version. Short version: api.ts SSR fetches go directly to
 * the backend and lose the cookie; this wrapper captures it once per
 * request and stashes it in AsyncLocalStorage so downstream code can
 * read it without explicit threading.
 *
 * Usage in a route file:
 *
 * ```ts
 * export const onRequest: RequestHandler = (event) => {
 *   return withSsrCookieContext(event, () => {
 *     requireAuthRedirect(event); // optional
 *     requireOwnboarding(event);  // optional
 *   });
 * };
 * ```
 *
 * Per-request only — the AsyncLocalStorage store is keyed by the inbound
 * request, never shared.
 */

import type { RequestEventCommon } from "@builder.io/qwik-city";
import { runWithSsrCookie } from "./ssr-cookie-context";

export function withSsrCookieContext(
  event: RequestEventCommon,
  next: () => unknown | Promise<unknown>,
): Promise<void> {
  const cookieHeader = event.request.headers.get("cookie") ?? "";
  return runWithSsrCookie(cookieHeader, next).then(() => undefined);
}
