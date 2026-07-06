/**
 * `requireOwnboarding` — server-side guard for routes that should
 * only render when at least one organization exists in the database.
 *
 * Reference: `openspec/changes/2026-07-06-ownboarding/specs/ownboarding/spec.md`
 *   R-OW-006 (S-OW-050..054) — helper behavior.
 *   R-OW-007 (S-OW-060..062) — home loader wiring.
 *   R-OW-008 (S-OW-070..073) — failure-mode fallback (optimistic /home).
 *
 * Sibling of `requireAuthRedirect` (ADR-0011, see
 * `frontend/src/lib/require-auth-redirect.ts`). Same shape: a
 * server-side guard that throws a Qwik City redirect on the failure
 * path and is a no-op on the success path.
 *
 * Why a separate helper instead of inlining in each route:
 *   - One source of truth for the redirect destination and the
 *     failure-mode fallback logic.
 *   - Trivially unit-testable in isolation (the function just reads
 *     the setup state, decides, and either throws or no-ops).
 *   - Future routes that need the same gate (e.g. a /projects route
 *     added later) import this helper in one line.
 *
 * Failure-mode fallback (optimistic):
 *   When `getSetupState()` rejects or returns an unexpected shape,
 *   the helper logs a warning via `console.warn` and returns
 *   normally. The caller (e.g. the home loader) treats this as
 *   "probably fine, render the page" so a transient backend hiccup
 *   does not trap the user on /ownboarding. This is the optimistic
 *   branch; the opposite branch (transport failure lands on
 *   /ownboarding) would create redirect loops and is explicitly
 *   NOT taken.
 *
 * No-redirect-loop guard:
 *   When the loader runs on /ownboarding itself (e.g. during a
 *   revalidation cycle), skip the redirect outright. Otherwise the
 *   user would bounce between /home (loader throws) and /ownboarding
 *   (loader throws) indefinitely.
 *
 * What the helper does NOT do:
 *   - It does NOT clear stale cookies or sessions. Auth.js handles
 *     session refresh as part of its normal request flow.
 *   - It does NOT log the access attempt to a structured logger.
 *     A future iteration may swap `console.warn` for an injected
 *     logger; for now, plain console output keeps the helper
 *     dependency-free.
 *   - It does NOT validate the org shape beyond the boolean. Future
 *     setup-state flags (e.g. `hasFirstAdmin`) would extend the
 *     contract in `lib/api.ts`; this helper would gate on those too.
 */
import type { RequestEventCommon } from "@builder.io/qwik-city";
import { getSetupState } from "~/lib/api";

/**
 * Re-export the request event type so consumers can import it from
 * this module (keeps the import surface minimal for downstream
 * callers — they don't need to reach into `@builder.io/qwik-city`).
 */
export type { RequestEventCommon };

/**
 * The actual guard. Reads setup state from the backend; throws a 302
 * redirect to `/ownboarding` if no organization exists; returns
 * normally otherwise. On any failure path (transport error or
 * malformed response) the helper logs a warning and returns
 * normally so the caller renders the page.
 *
 * Usage in a guarded route file (e.g. `routes/home/index.tsx`):
 * ```ts
 * export const useSetupLoader = routeLoader$(async (event) => {
 *   await requireOwnboarding(event);
 *   return null;
 * });
 * ```
 */
export async function requireOwnboarding(
  event: RequestEventCommon,
): Promise<void> {
  // No-redirect-loop guard: if the loader runs on /ownboarding
  // itself (e.g. a stale request, a revalidation cycle), do nothing.
  // Otherwise the user would bounce between /home (loader throws)
  // and /ownboarding (loader throws) indefinitely.
  if (event.url.pathname === "/ownboarding") {
    return;
  }

  // Fetch setup state. Wrap in try/catch so any rejection
  // (network error, timeout, 5xx) falls through to the optimistic
  // no-throw path.
  let result: Awaited<ReturnType<typeof getSetupState>> | undefined;
  try {
    result = await getSetupState();
  } catch (error) {
    console.warn("[requireOwnboarding] setup-state fetch failed:", error);
    return;
  }

  // Defensive shape check. `getSetupState()` already validates the
  // response shape and throws on malformed input, but a defensive
  // re-check here means a future drift in `lib/api.ts` cannot leak
  // through to a redirect loop.
  if (
    result &&
    typeof result === "object" &&
    typeof (result as { hasOrganization?: unknown }).hasOrganization ===
      "boolean"
  ) {
    if (!result.hasOrganization) {
      throw event.redirect(302, "/ownboarding");
    }
    return;
  }

  // Unexpected shape — should not happen given the guard in
  // `getSetupState()`, but a defensive log keeps the failure mode
  // visible in production.
  console.warn(
    "[requireOwnboarding] unexpected setup-state shape:",
    result,
  );
  return;
}