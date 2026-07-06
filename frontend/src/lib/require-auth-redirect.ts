/**
 * `requireAuthRedirect` — server-side guard for protected routes.
 *
 * Reference: engram observation `cachicamas/auth/anatomy` (2026-07-06)
 * and the native-auth-UI decision (2026-07-06): anonymous visitors are
 * redirected to the native `/auth/signin` page instead of seeing an
 * inline `SignInRequiredCard` on the protected route. The OAuth
 * roundtrip then brings them back to the original URL via the
 * `callbackUrl` query parameter.
 *
 * Why a server-side `onRequest` instead of a client-side `useTask$` /
 * `useNavigate`:
 *   - SSR-time redirect: the browser receives a 302 with no body, no
 *     protected-route HTML ever rendered. No SSR flash of the inline
 *     card, no client-side hydration gap, no double-render.
 *   - Runs BEFORE `routeLoader$` and the component$, so DB calls and
 *     other expensive side effects never fire on the anon path.
 *     This is the same defence-in-depth argument the inline-card
 *     pattern made (R-PR-003), now enforced at the HTTP boundary.
 *
 * Why a separate helper instead of inlining in each route:
 *   - One source of truth for the redirect URL shape.
 *   - Trivially testable in isolation (the function is just a RequestHandler
 *     that either throws a redirect or no-ops).
 *   - Adding a new protected route is one line:
 *       export { requireAuthRedirect as onRequest } from "~/lib/require-auth-redirect";
 *
 * Where the session comes from:
 *   `@auth/qwik`'s `onRequest` populates `event.sharedMap.get("session")`
 *   with the JWT-decoded Auth.js session object (or `null`). The session
 *   shape is `{ user?: { ... }, expires?: string }`. We only need to
 *   check `.user`; if it's truthy, the visitor is authenticated.
 *
 * What the helper does NOT do:
 *   - It does NOT clear stale cookies. Auth.js's session loader
 *     handles cookie refresh as part of its normal request flow.
 *   - It does NOT validate CSRF tokens. The Auth.js callback flow has
 *     its own CSRF check.
 *   - It does NOT log the access attempt. If we later need an audit
 *     trail of unauthenticated access to protected routes, add a
 *     structured log line here.
 */
import type { RequestEventCommon, RequestHandler } from "@builder.io/qwik-city";

/**
 * The Auth.js session shape we read from sharedMap. We only look at
 * `.user`, so the rest is loose `unknown`. The cast to `unknown` (via
 * `as { user?: unknown } | null`) is intentional: sharedMap stores
 * arbitrary values, and the runtime check is what matters.
 */
type AuthSession = { user?: unknown } | null | undefined;

/**
 * Build the canonical signin URL with a `callbackUrl` query param.
 *
 * Pure function (no side effects, no I/O) so unit tests can assert the
 * exact URL shape without spinning up a Qwik request event.
 *
 * Why include the search string in `callbackUrl`:
 *   Some protected routes carry meaningful query params (e.g.
 *   `/organizations/[id]?tab=members`). Stripping them would silently
 *   lose context on the round trip. We keep the full
 *   `pathname + search` so the post-signin redirect lands the visitor
 *   on the same view they tried to access.
 *
 * Why strip trailing slashes from the callbackUrl:
 *   Qwik City auto-canonicalises paths to trailing-slash form
 *   BEFORE middleware runs (so `/home` arrives as `/home/`). If we
 *   forward that verbatim, the post-signin redirect lands on the
 *   trailing-slash form, which works but is ugly. Normalising here
 *   keeps the round-trip URL clean.
 *
 * Why URL-encode the callbackUrl:
 *   The callback URL is itself a URL with possible `?` and `&`
 *   characters (e.g. `/foo?bar=baz`). Encoding it once makes it a
 *   safe value for the OUTER `?callbackUrl=...` query parameter.
 */
export function buildSigninRedirectUrl(rawCallbackUrl: string): string {
  const normalised = rawCallbackUrl.replace(/\/+$/, "") || "/";
  const encoded = encodeURIComponent(normalised);
  return `/auth/signin?callbackUrl=${encoded}`;
}

/**
 * The actual `onRequest` handler. Reads the session from sharedMap
 * (populated by `@auth/qwik`'s `onRequest` middleware), checks for
 * `.user`, and throws a 302 redirect to the native signin page if
 * absent.
 *
 * Usage in a protected route file:
 * ```ts
 * export { requireAuthRedirect as onRequest } from "~/lib/require-auth-redirect";
 * ```
 *
 * The `export ... as onRequest` re-export is the canonical Qwik City
 * pattern for declaring a route-level onRequest. The build pipeline
 * reads the export name, so the alias must be `onRequest` exactly.
 */
export const requireAuthRedirect: RequestHandler = (event) => {
  const session = event.sharedMap.get("session") as AuthSession;
  const user = session?.user;
  if (!user) {
    const callbackUrl = event.url.pathname + event.url.search;
    throw event.redirect(302, buildSigninRedirectUrl(callbackUrl));
  }
};

/**
 * Re-export the `RequestEventCommon` type so consumers (e.g. tests)
 * can import the request shape alongside the handler without reaching
 * into `@builder.io/qwik-city` directly. Keeps the import surface
 * minimal for downstream callers.
 */
export type { RequestEventCommon };
