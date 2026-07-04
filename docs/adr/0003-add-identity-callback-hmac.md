# ADR 0003: HMAC-based identity signin-callback protocol

> Status: **Accepted** (2026-07-04, change `cachicamas-identity-signin-callback`)
> Author: cachicamas SDD pipeline (parent orchestrator + worker)
> Supersedes: PR #29's `IDENTITY_DATABASE_URL` + `porsager/postgres` +
> `cachicamas-stub-server-only-deps` Vite plugin (rejected architecture)

## Context

`cachicamas-github-login` PR-2 (merged on origin/main) shipped the
frontend's `events.signIn` callback that persisted GitHub OAuth
identity rows directly to Postgres from the Qwik Node SSR. The
implementation used the `porsager/postgres` driver behind a Vite
plugin that stubbed `postgres` for the browser bundle (otherwise
Rollup walked `import { performance } from "perf_hooks"` and crashed
the client build with `"performance" is not exported by "__vite-browser-external"`).

PR #29 (`feat/cachicamas-github-login-events-signin`, draft only;
never merged) extended PR-2 with the same pattern. The 4R inline
review of PR #29 surfaced two architectural concerns:

- **R1-1 [LOW] credential sprawl**: the frontend and the
  `database_administrator` service shared `QUEEN_PASSWORD` via the
  `IDENTITY_DATABASE_URL` env var. Two writers (`events.signIn` on
  the Node SSR + the Go service on the same DB) and one role /
  password to rotate.
- **R2-1 [LOW] Vite plugin inlined in `vite.config.ts`**: a fragile
  bundling workaround that depends on Rollup's static-analysis path
  for `postgres`. Future dep upgrades (or sibling deps that pull
  `node:perf_hooks`) can break the build with no upstream signal.

This ADR closes those two findings by replacing the architecture
with an HMAC-signed HTTP endpoint on the backend service. The
frontend calls the endpoint; the backend writes to Postgres.

## Decision

### Wire protocol: HMAC-SHA256 over `${timestamp}.${canonical_json}`

The Qwik Node SSR (`events.signIn` callback) POSTs to
`POST /api/v1/identity/signin-callback` on the `database_administrator`
service with three headers:

```
Content-Type: application/json
X-Cachicamas-Timestamp: <unix_ms string>            e.g. "1720000000000"
X-Cachicamas-Signature: base64(HMAC_SHA256(secret, ts + "." + canonical_json))
```

Body shape:

```json
{
  "user":    { "id": "12345", "email": "...", "name": "...", "image": null },
  "account": { "provider": "github", "providerAccountId": "12345",
               "accessToken": "gho_...", "refreshToken": null,
               "expiresAt": null, "tokenType": "bearer",
               "scope": "read:user user:email" }
}
```

Canonical JSON is locked at: keys sorted lexicographically
(NOT lowercased); no whitespace; no padding. The implementation is
shared between Go (`backend/.../interfaces/http/identity_handler.go`)
and TypeScript (`frontend/src/lib/identity-callback-client.ts`); a
known input → known output vector test pins the contract on both
sides.

### Endpoint contract

| Status | When |
| --- | --- |
| `204 No Content` | Signature valid + body valid + UPSERT succeeds |
| `401 Unauthorized` | Missing / malformed / expired / wrong signature |
| `422 Unprocessable Entity` | JSON parse error OR missing required fields |
| `500 Internal Server Error` | Service error (DB down); logged with `auth.outcome=error` |

### Anti-replay

- The backend rejects if `|now - timestamp| > 5 min` (configurable
  via `antiReplayWindow`; clock-skew window in each direction).
- The backend uses `crypto/subtle.ConstantTimeCompare` for the
  signature comparison itself (defense against timing side-channels).

### Env var

A new env var, `IDENTITY_CALLBACK_SECRET`, lives in BOTH the
`frontend` and `database_administrator` compose env blocks. The
fail-fast posture (`${IDENTITY_CALLBACK_SECRET:?IDENTITY_CALLBACK_SECRET must be set in .env}`)
matches the existing `${AUTH_SECRET:?...}` convention.

The value MUST be DIFFERENT from `AUTH_SECRET`: the JWE cookie
encryption key has a different purpose (stateless session
envelope) and rotation cadence (rotate on user logout / session
invalidation) than the HMAC callback secret (rotate on suspected
secret leakage).

Generate with `openssl rand -base64 32`.

## Alternatives considered

### JWT (JWS)

Pros: signature format is standardized (RFC 7519 / RFC 7515),
cross-tooling libraries on every platform.

Cons: requires JWS library on both sides (Go: `lestrrat-go/jwx/v2/jws`,
which we already use for JWE; TS: `jose` or `@panva/jws`). Heavier
than HMAC for the same security property at this trust level.
Not chosen.

### mTLS (mutual TLS)

Pros: strongest defense against network adversaries (no shared
secret; identity is bound to the cert).

Cons: requires PKI (cert authority, cert rotation, mutual trust
store updates in both containers). The trust boundary in this
slice is the compose internal network — a private bridge network
that only the two containers can reach. The cost / benefit is
unfavorable for a development-stage stack.

Deferred to the "hostile network" forward note: if the stack ever
moves to multi-tenant / public-facing, switch to mTLS or SPIFFE.

### SPIFFE / workload identity

Pros: workload-identity-based authentication without shared
secrets; aligns with cloud-native zero-trust posture.

Cons: requires a SPIRE control plane + workload-agent sidecars in
each container. Same forward note as mTLS — defer until the
network trust boundary changes.

### Long-lived bearer token

Pros: trivial to implement; same library on both sides.

Cons: no rotation, no replay defense, no integrity protection
over the body (the token authenticates the sender but not the
specific request). Rejected.

## Threat model

### In scope (HMAC + timestamp is sufficient)

- **Insider adversary on the compose network**: an attacker who
  has compromised one container can attempt to replay old signed
  requests. The 5-minute window + the operational expectation
  that the IdP rotates GitHub account ids on a longer cadence
  make replay irrelevant at the persistence level (the UPSERT is
  idempotent on `(provider, provider_account_id)`).
- **Accidental leakage of `IDENTITY_CALLBACK_SECRET`**: the secret
  is committed only in `.env` (gitignored) and rendered into
  compose. Rotation requires updating the secret + restarting both
  containers; no other consumer.

### Out of scope (forward notes)

- **Multi-tenant or public-facing deploy**: if the stack ever
  moves off the compose internal bridge network, switch to mTLS or
  SPIFFE. HMAC + timestamp does not protect against an adversary
  who can MitM the wire.
- **Replay at the application level**: the backend's idempotent
  UPSERT means a re-signed request from the same client within
  the window is a no-op for the row, but it does inflate audit
  logs. Forward note: add a nonce store at the backend (in-memory
  ring buffer keyed by HMAC) for stricter replay defense.
- **PII in logs**: the handler logs `auth.email_hash` (sha256 first
  12 hex chars) on success, NOT the raw email. Forward note:
  audit the OTel exporter for any field that might leak raw PII.

## Consequences

### Positive

- The frontend and backend no longer share a DB role + password.
- The frontend's `package.json` no longer needs the `postgres`
  npm package as a runtime dep (kept for now to avoid an
  unrelated dep churn; cleanup is a future slice).
- The Vite `cachicamas-stub-server-only-deps` plugin is no longer
  needed. `optimizeDeps.exclude: ["postgres"]` is reverted.
- The backend's `events.signIn` storage path now goes through the
  same observability stack as every other handler (`auth.upsert`
  OTel span, `auth.outcome` slog attribute).
- The identity schema (R-IS-001 / R-IS-020 / R-IS-030) is
  unchanged; the UPSERT SQL is added as a CTE in the existing
  `identity_repository.go` adapter.

### Negative

- One more moving part (the new endpoint) to monitor + alert on.
- One more secret (`IDENTITY_CALLBACK_SECRET`) to rotate on
  suspected leakage.
- The frontend's `events.signIn` callback now has a network
  dependency that wasn't there before. The best-effort posture
  (log + swallow) protects the OAuth roundtrip from cascading
  failures, but a chronic backend outage will show up as
  re-sign-ins creating duplicate `identity.account` rows
  (mitigated by the `ON CONFLICT DO NOTHING` UNIQUE constraint).

### Neutral

- OAuth tokens in the wire body are accepted but NOT persisted.
  The `identity.account` schema does not have token columns in
  this slice; adding them is a separate migration + ADR.
- The `/api/*` reverse proxy in `frontend/src/entry.express.tsx`
  now preserves the `/api` prefix for `/api/v1/*` paths (the
  identity callback is at `/api/v1/identity/signin-callback` on
  the backend). Legacy `/api/*` routes (`/api/organizations`
  etc.) continue to be stripped as before.

## Cross-references

- ADR 0001: `@auth/qwik@0.9.2` acceptance (pre-1.0 mitigation)
- ADR 0002: `lestrrat-go/jwx/v2` for JWE verifier middleware
- 4R inline review of PR #29:
  `/Users/braejan/.pi/agent/scratch/4r-review-events-signin.md`
- Backend tests: `backend/database_administrator/src/interfaces/http/identity_handler_test.go`
  (12 tests including cross-tooling canonical oracle + HMAC +
  anti-replay + schema validation)
- Frontend tests: `frontend/src/lib/identity-callback-client.test.ts`
  (15 tests including cross-tooling canonical oracle + HMAC +
  URL resolution + 204/401/422/500 mapping)
- Backend service: `backend/database_administrator/src/application/identity_service.go`
  (`UpsertFromOAuth` — OTel span `identity.upsert`)
- Backend adapter: `backend/database_administrator/src/infrastructure/postgres/identity_repository.go`
  (`Upsert` — CTE + ON CONFLICT DO NOTHING)
- Frontend client: `frontend/src/lib/identity-callback-client.ts`
  (canonicalizeJSON + postIdentityCallback)
- Frontend wiring: `frontend/src/routes/plugin@auth.ts`
  (`events.signIn` callback — best-effort forward)