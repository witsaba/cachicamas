/**
 * CSRF defense-in-depth on the frontend.
 *
 * Reference: sdd/security-vulnerability-remediation/spec/csrf-origin-validation
 *   REQ-02 — every state-changing request MUST carry
 *            `X-Requested-With: XMLHttpRequest`. The backend's
 *            csrf_middleware rejects form-encoded POSTs that lack
 *            it, so the frontend must add it on every mutating call.
 *   REQ-03 — safe methods (GET/HEAD/OPTIONS) MUST NOT be gated.
 *
 * The server-side check in dbadmin's `interfaces/http/csrf_middleware.go`
 * is the authoritative validation. This client-side helper is a
 * belt-and-suspenders: it (1) prevents accidental misses at the
 * call site, and (2) carries the `X-Requested-With` header that
 * most cross-origin attackers cannot forge from a `<form>` element.
 */
const STATE_CHANGING_METHODS: ReadonlySet<string> = new Set([
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
]);

/**
 * Fetch wrapper that adds the CSRF defense-in-depth header on
 * state-changing methods. Safe methods are passed through unchanged.
 */
export function stateChangingFetch(
  url: string,
  init?: RequestInit,
): Promise<Response> {
  const method = (init?.method ?? "GET").toUpperCase();
  if (STATE_CHANGING_METHODS.has(method)) {
    const headers = new Headers(init?.headers);
    headers.set("X-Requested-With", "XMLHttpRequest");
    return fetch(url, { ...init, headers });
  }
  return fetch(url, init);
}
