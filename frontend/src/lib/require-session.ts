/**
 * `requireSession` — the pure guard helper for protected routes.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/specs/protected-routes/spec.md`
 *   R-PR-001 — pure function of (session, pathname) →
 *     `{ kind: "auth", session }` when the session has a user, or
 *     `{ kind: "anon", pathname }` otherwise.
 *
 * Pure: no `useTask$`, no `useSignal`, no I/O. Lives outside the Qwik City
 * request context so unit tests can call it freely under vitest.
 *
 * ADR-0008: deliberately not a Qwik hook — keeps the helper unit-testable
 *   without the createDOM() harness needing a request context.
 */

export interface RequireSessionAuth<S> {
  kind: "auth";
  session: S;
}

export interface RequireSessionAnon {
  kind: "anon";
  pathname: string;
}

export type RequireSessionGuard<S> = RequireSessionAuth<S> | RequireSessionAnon;

/**
 * Decide whether the current `useSession()` value authorizes the request.
 *
 * @param session The result of `useSession().value` (or `null` when
 *   anonymous). Shape is intentionally loose; we only read `.user`.
 * @param pathname The current path, used to populate `redirectTo` in the
 *   signed-out card so the user lands back here after signing in.
 */
export function requireSession<S extends { user?: unknown } | null>(
  session: S,
  pathname: string,
): RequireSessionGuard<NonNullable<S>> {
  if (session && typeof session === "object") {
    const user = (session as { user?: unknown }).user;
    if (user) {
      return { kind: "auth", session: session as NonNullable<S> };
    }
  }
  return { kind: "anon", pathname };
}
