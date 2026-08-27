/**
 * Decide which internal proxy target handles a given request URL.
 *
 * - `/api/agent/*` → agent_chat binary
 * - `/api/chat/*` → agent_chat binary (the chat binary's config endpoints
 *   live here, by design — see backend/agent/.../chat.RegisterAssistantConfigRoutes)
 * - `/api/archetypes/*` → agent_chat binary (T-24 of
 *   cachicamas-archetype-system-foundation: the polymorphic
 *   /api/archetypes/{slug}, /api/archetypes?type=, and
 *   /api/archetypes/{slug}/config/ handlers are hosted on the chat
 *   binary today; the router arm shares the 'chat' target rather than
 *   introducing a fourth proxy hop).
 * - `/api/*` → database_administrator binary
 * - other → handled by Qwik/static/404 (return `null`)
 *
 * Order matters: the more specific prefixes MUST be checked before the
 * generic `/api/*` fall-through, otherwise `/api/chat/assistant/config`
 * would be routed to database_administrator and die with a 404.
 *
 * Lives in its own module so unit tests can import the pure decision
 * without dragging in Qwik City's SSR middleware (which has module-level
 * side-effects that cannot run in a vitest context).
 */
export type ApiRouteTarget = "agent" | "chat" | "api" | null;

export function routeApiRequest(url: string): ApiRouteTarget {
  if (typeof url !== "string" || url.length === 0) return null;
  if (url.startsWith("/api/agent/")) return "agent";
  if (url.startsWith("/api/chat/")) return "chat";
  // T-24: the polymorphic /api/archetypes/* tree is hosted on the chat
  // binary (it serves HandleGetArchetype and HandlePutArchetypeConfig
  // after PR-2 T-19). Returns 'chat' so entry.express.tsx's existing
  // `case "chat"` arm proxies to the same upstream as /api/chat/*.
  // Declared BEFORE the `/api/*` fall-through so a future spec change
  // that narrows the catch-all (e.g. to /api/v1/*) does not strand
  // the polymorphic surface. Matches BOTH the bare `/api/archetypes`
  // (used by the directory list at `/api/archetypes?type=system`) and
  // the tree under `/api/archetypes/...`.
  if (
    url.startsWith("/api/archetypes/") ||
    url === "/api/archetypes" ||
    url.startsWith("/api/archetypes?")
  ) {
    return "chat";
  }
  if (url.startsWith("/api/")) return "api";
  return null;
}
