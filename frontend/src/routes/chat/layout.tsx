/**
 * routes/chat/layout.tsx — auth + onboarding guard chain for /chat.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-3.
 *   "Unauthenticated access to /chat SHALL redirect via the existing
 *    setSsrCookieHeader → requireAuthRedirect → requireOwnboarding
 *    chain, identical to routes/home/."
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 row 11 (routes/chat/layout.tsx) — "Auth gate (same chain as
 *   routes/home/index.tsx:39-51). REQ-3."
 *
 * The chain mirrors `frontend/src/routes/home/index.tsx:39-51`:
 *
 *   1. setSsrCookieHeader captures the inbound cookie BEFORE the
 *      guards run so any SSR-time api fetches in useTask$ can
 *      re-attach it.
 *   2. requireAuthRedirect throws synchronously when anonymous —
 *      Qwik catches the sync throw and short-circuits with the
 *      redirect (302 → /auth/signin?callbackUrl=%2Fchat).
 *   3. requireOwnboarding is ASYNC and awaits a setup-state fetch
 *      to the backend. Awaiting it makes its `event.redirect(...)`
 *      rejection propagate through onRequest's returned Promise,
 *      which Qwik treats as a redirect rather than a fatal server
 *      error.
 *
 * Aphantasic-friendly: no imagery, text-first. SSR cookie forwarding
 * is invisible to the rendered HTML.
 */
import { type RequestHandler } from "@builder.io/qwik-city";

import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";

export const onRequest: RequestHandler = async (event) => {
  // Capture the inbound cookie BEFORE the guards run so SSR-time
  // api fetches in useTask$ can re-attach it. The chat window does
  // not currently issue SSR fetches (the SSE stream is browser-only
  // per REQ-1 S-1.a), but future additions (history, list-restore)
  // would need this and it costs nothing to set up now.
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  // REQ-3 S-3.a — requireAuthRedirect throws synchronously when
  // anonymous; Qwik converts the throw into a 302.
  requireAuthRedirect(event);
  // REQ-3 S-3.b — requireOwnboarding is async; awaiting it lets
  // its `event.redirect(...)` rejection propagate as a redirect.
  await requireOwnboarding(event);
};
