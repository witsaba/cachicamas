/**
 * Gate for diagnostic startup logs that would otherwise expose internal
 * topology (e.g. the database_administrator service hostname).
 *
 * Reference: sdd/security-vulnerability-remediation/spec/security-response-headers
 *   REQ-03 — silence the internal API target log in production.
 *
 * Production MUST be silent. The single toggle is `DEBUG=1` so we
 * keep the surface tiny (no log levels, no namespaces).
 */
export function logInternalTarget(): boolean {
  return process.env.DEBUG === "1";
}
