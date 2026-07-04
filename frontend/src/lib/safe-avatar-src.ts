/**
 * `safeAvatarSrc` — sanitize the avatar URL before it reaches the DOM.
 *
 * Reference: `openspec/chachicamas-login-ux/design.md` §7
 *   ADR-0009 — avatar URL sanitization uses a scheme allowlist.
 *
 * Context:
 *   `session.user.image` comes from GitHub's `avatar_url` field.
 *   GitHub's value is always `https://avatars.githubusercontent.com/…`,
 *   never a `javascript:` or `data:` URL. We MUST still defend the
 *   rendered `<img>` because the session JWE is decryptable by anyone
 *   with `AUTH_SECRET`. If a future change adds a custom OAuth
 *   provider, a malicious or buggy provider could supply a non-HTTP
 *   URL — and the avatar image would execute JavaScript the moment
 *   it lands in an `<img src>`.
 *
 * Behavior:
 *   - null / undefined / "" → null (caller renders no <img>).
 *   - URL whose scheme is `https:` → returned verbatim.
 *   - URL with no scheme (a relative path like `/uploads/avatar.png`) → returned verbatim.
 *   - Anything else (http:, javascript:, data:, vbscript:, file:, …) → null.
 *
 * Implementation note: we DO NOT use `URL` here because we want to
 * permit relative paths (no scheme). The allowlist is regex-based on
 * the raw string, after a `.trim()`. This is intentionally simple —
 * the failure mode is "we reject something we could have shown",
 * never "we show something dangerous".
 */

/** Allowed schemes for an avatar URL. */
const ALLOWED_SCHEMES = ["https:"] as const;

/**
 * Trim and lowercase the URL prefix for the scheme check. Relative
 * paths (no scheme) are returned as-is.
 *
 * @returns the original URL when it passes the allowlist, or null.
 */
export function safeAvatarSrc(url: string | null | undefined): string | null {
  if (!url) return null;
  const trimmed = url.trim();
  if (!trimmed) return null;
  // Find the scheme: "<letters>:" prefix.
  // A relative path (no colon before the first slash) has no scheme.
  const schemeMatch = /^([a-zA-Z][a-zA-Z0-9+.\-]*):/.exec(trimmed);
  if (!schemeMatch) {
    // No scheme — treat as a same-origin relative path. Return as-is.
    return trimmed;
  }
  const scheme = schemeMatch[1]!.toLowerCase() + ":";
  if ((ALLOWED_SCHEMES as readonly string[]).includes(scheme)) {
    return trimmed;
  }
  return null;
}