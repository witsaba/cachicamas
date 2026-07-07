/**
 * `withSsrCookieContext` — captures the inbound Cookie header for the
 * rest of the request, then runs `next()` which should set up the auth
 * + ownboarding guards.
 *
 * Reference (S-WS-AUTH-CHAIN-SSR-001): see `ssr-cookie-context.ts` for
 * the long version. Short version: api.ts SSR fetches go directly to
 * the backend and lose the cookie; this helper captures it once per
 * request so downstream `useTask$` api fetches can re-attach it.
 *
 * Synchronous, NOT a Promise wrapper. Why:
 *   The auth middleware (`requireAuthRedirect`) throws
 *   `event.redirect(...)` to short-circuit unauthenticated requests.
 *   Qwik City propagates synchronous throws from `onRequest` as a
 *   proper redirect — but a rejected Promise from `onRequest` is
 *   treated as a server error and the route component still tries to
 *   render, which produces `Error: Response already sent`. By calling
 *   the guards at the top level (after the sync cookie capture), the
 *   throws stay sync and Qwik handles them cleanly.
 */

import type { RequestEventCommon } from "@builder.io/qwik-city";
import { setSsrCookieHeader } from "./ssr-cookie-context";

export function withSsrCookieContext(
  event: RequestEventCommon,
  next: () => void,
): void {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  // Run the caller's next() synchronously. Guard throws propagate up
  // out of onRequest as designed.
  next();
}
