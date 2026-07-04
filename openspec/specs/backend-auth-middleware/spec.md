# backend-auth-middleware Specification

> **Domain**: backend-auth-middleware
> **Change**: cachicamas-github-login
> **Type**: New capability (full spec — no existing auth on the Go side)
> **Created**: 2026-07-03
> **Persistence**: hybrid (this file + Engram `sdd/cachicamas-github-login/spec/backend-auth-middleware`)

## Purpose

Defines the Go-side authentication contract. The `database_administrator`
service MUST be able to verify a JWE session cookie that Auth.js for Qwik
issued on the same origin, so future protected `/api/*` endpoints have a
tested, audited verifier from day one. The slice introduces one Echo
middleware (`IdentityFromCookie`) and the smallest possible domain surface
(`Identity` value + `IdentityRepository` port). It MUST NOT introduce a
session storage abstraction (the slice is stateless per the proposal).

## Glossary

| Term | Meaning |
| ------ | --------- |
| **JWE cookie** | The Auth.js JWT-strategy session, serialized as a signed/encrypted JWE, carried in the `authjs.session-token` cookie. |
| **AUTH_SECRET** | The 32-byte base64 secret shared between the Qwik Node server (Auth.js encoder) and the Go service (verifier). MUST be the same value across both. |
| **KEK (Key Encryption Key)** | The 32-byte symmetric key derived from `AUTH_SECRET` that wraps the CEK in the JWE envelope. |
| **CEK (Content Encryption Key)** | The per-request content encryption key. |
| **A256GCM-HS256** | The Auth.js default JWE envelope: HS256-derived KEK + A256GCM content encryption. The Go verifier MUST support this exact envelope. |
| **Echo middleware** | The Echo v5 `echo.MiddlewareFunc` shape. The new middleware follows the same shape as `cors.go`. |
| **`Identity`** | The Go-side value object the middleware populates on `c.Set("identity", *Identity)` for downstream handlers. |
| **`IdentityRepository`** | The domain port the middleware uses to resolve an `Identity` from the JWE claims. |

---

## Capability: JWE verification on the Go side

### Requirement: R-BAM-001 The Go service exposes an `IdentityFromCookie` Echo middleware

The `backend/database_administrator/src/interfaces/http/auth_middleware.go`
file MUST export an `IdentityFromCookie(cfg IdentityMiddlewareConfig) echo.MiddlewareFunc`
function with the same shape as `cors.go` (a `Config` struct + a factory
returning the middleware function). The middleware MUST:

1. Read the `authjs.session-token` cookie from the request.
2. Verify the JWE envelope using the configured `AUTH_SECRET`.
3. Decode the JWT claims (`sub`, `email`, `name`, `picture`, `provider`, `provider_account_id`).
4. Resolve the `Identity` value via `cfg.IdentityRepository.LookupByEmail` (or `LookupByProviderAccountID` if the cookie claims carry that).
5. Populate `c.Set("identity", &identity)`.
6. On any failure, return HTTP 401 with a locked error envelope
   (same shape as the existing `MsgNotFound` envelope but with code `unauthorized`).

#### Scenario: S-BAM-010 Valid cookie populates `c.Get("identity")`

- GIVEN a request whose `Cookie:` header carries a JWE that was signed by the
  Auth.js encoder using the project's `AUTH_SECRET`
- AND the JWE payload's `email` matches a row in `identity.user`
- WHEN the request hits a handler protected by `IdentityFromCookie`
- THEN the middleware SHALL call `next(c)` without writing any response
- AND the downstream handler SHALL observe `c.Get("identity")` returns a non-nil `*Identity`
- AND that `Identity.Email` SHALL equal the JWE payload's `email`

Verification: `interfaces/http/auth_middleware_test.go` `TestIdentityFromCookie_ValidCookie_PopulatesIdentity`.

#### Scenario: S-BAM-011 Tampered cookie returns 401

- GIVEN a request whose `Cookie:` header carries a JWE that was issued with `AUTH_SECRET="AAAAAAAA..."` but the request is presented to a server configured with `AUTH_SECRET="BBBBBBBB..."`
- WHEN the request hits a handler protected by `IdentityFromCookie`
- THEN the middleware SHALL write an HTTP 401 response
- AND the response body's `code` field SHALL equal `"unauthorized"`
- AND `next(c)` SHALL NOT be invoked

Verification: `TestIdentityFromCookie_TamperedCookie_Returns401`.

#### Scenario: S-BAM-012 Missing cookie returns 401

- GIVEN a request that does NOT carry an `authjs.session-token` cookie
- WHEN the request hits a handler protected by `IdentityFromCookie`
- THEN the middleware SHALL write an HTTP 401 response with `code=unauthorized`

Verification: `TestIdentityFromCookie_MissingCookie_Returns401`.

### Requirement: R-BAM-002 The middleware uses lestrrat-go/jwx/v2 for JWE decryption

The implementation MUST use `github.com/lestrrat-go/jwx/v2/jwe` to perform
the JWE decryption with the A256GCM-HS256 envelope (KEK derivation matching
Auth.js's HKDF-SHA256 over `AUTH_SECRET`, content encryption A256GCM). The
choice of `jwx` MUST be recorded in the ADR `adr-jwx-for-jwe` (filed during
`sdd-design`).

#### Scenario: S-BAM-020 Decryption produces a JSON payload

- GIVEN a JWE produced by `@auth/core/jwt`'s encoder
- WHEN the Go verifier decrypts it
- THEN the resulting plaintext SHALL be valid UTF-8 JSON
- AND the JSON SHALL parse into the `AuthJSCookiePayload` struct
  (`{"sub": ..., "email": ..., "name": ..., "picture": ..., "exp": ..., "iat": ...}`)

Verification: `TestIdentityFromCookie_DecryptionShape` uses a fixture JWE (committed under `testdata/authjs_session_token.jwe` and regenerated by a helper script) and asserts on the decoded payload.

### Requirement: R-BAM-003 The middleware does not introduce any cross-origin cookie behavior

The middleware MUST NOT set any `Set-Cookie` header of its own. The Auth.js
encoder on the Qwik side is the sole issuer of the `authjs.session-token`
cookie. The middleware is **read-only** with respect to cookies.

#### Scenario: S-BAM-030 Middleware never sets a Set-Cookie header

- GIVEN a successful authenticated request
- WHEN the response is returned
- THEN the response SHALL NOT include a `Set-Cookie` header
- AND the response SHALL NOT include any `authjs.*` cookie update

Verification: `TestIdentityFromCookie_NoSetCookieOnSuccess`.

---

## Capability: Domain surface for identity

### Requirement: R-BAM-010 The domain defines `Identity`, `IdentityRepository`, and the lookup errors

The `domain/identity.go` file MUST define:

- `type Identity struct { ID int64; Email string; Name string; ImageURL string; Provider string; ProviderAccountID string }`
- `type IdentityRepository interface { LookupByEmail(ctx context.Context, email string) (*Identity, error) }`
- A `ErrIdentityNotFound error` (or `*NotFoundError` analog) for the "no row" case
- The `Identity` MUST be considered `nil` + `ErrIdentityNotFound` returned only when no `identity.user` row matches the email

#### Scenario: S-BAM-040 Identity struct fields match DB columns

- GIVEN the change is applied
- WHEN the orchestrator reads `domain/identity.go`
- THEN the struct SHALL have the locked field list (ID, Email, Name, ImageURL, Provider, ProviderAccountID)
- AND the `db` and `json` tags SHALL match the schema in the `identity-schema` spec

Verification: `go test ./domain/... -run TestIdentityStructFields`.

#### Scenario: S-BAM-041 Repository returns ErrIdentityNotFound on miss

- GIVEN `identity.user` has no row for `ghost@example.com`
- WHEN `IdentityRepository.LookupByEmail(ctx, "ghost@example.com")` is called
- THEN the function SHALL return `(nil, ErrIdentityNotFound)`

Verification: `interfaces/http/auth_middleware_test.go` `TestIdentityFromCookie_UnknownEmail_Returns401` (cookie decrypts OK but the email has no row → 401).

### Requirement: R-BAM-011 The Postgres `IdentityRepository` implementation is hexagonal-clean

The `infrastructure/postgres/identity_repository.go` file MUST be the ONLY
file outside `domain/` and `interfaces/` that imports `github.com/jackc/pgx`.
The `domain/identity.go` MUST NOT import `pgx` or any infrastructure package
(consistent with the project's hexagonal rule, see `db-migrations` R-DBMIG-030
/ R-DBMIG-031).

#### Scenario: S-BAM-050 Hexagonal rule respected

- GIVEN the change is applied
- WHEN the orchestrator greps for `jackc/pgx` imports under `backend/database_administrator/src/`
- THEN only `infrastructure/postgres/identity_repository.go` (and the
      pre-existing `migration/postgres/driver.go` and `migration/runner.go`)
      SHALL match
- AND `domain/identity.go` SHALL NOT match
- AND `application/identity_service.go` SHALL NOT match

Verification: `grep -rE "jackc/pgx" backend/database_administrator/src/domain/ backend/database_administrator/src/application/` returns no Go-import lines.

---

## Capability: Hexagonal wiring and observability

### Requirement: R-BAM-020 The middleware is registered AFTER CORS and BEFORE the protected routes

The composition root (`src/cmd/server/main.go`) MUST register
`IdentityFromCookie` on the route group(s) that need it. CORS MUST be
registered before any auth middleware so preflight `OPTIONS` requests
continue to short-circuit (consistent with the existing CORS behavior in
`cors.go`). Health and organization-list endpoints MUST NOT require auth in
this slice (they remain public); at least one demo protected endpoint MAY be
added to prove the middleware is wired.

#### Scenario: S-BAM-060 Health endpoint remains public

- GIVEN the middleware is registered on `/api/v1/protected/*` but NOT on `/health`
- WHEN `curl -fsS http://localhost:8080/health` is executed
- THEN the response SHALL be HTTP 200 (the endpoint is public)
- AND no `Set-Cookie` or 401 SHALL appear

Verification: integration smoke test in `e2e/` (Go-level) or the
existing curl-based smoke in `Makefile`'s `test/smoke` target.

#### Scenario: S-BAM-061 Demo protected endpoint rejects an unauthenticated request

- GIVEN a route `/api/v1/protected/whoami` is registered behind `IdentityFromCookie`
- WHEN `curl -fsS http://localhost:8080/api/v1/protected/whoami` is executed
- THEN the response SHALL be HTTP 401 with `code=unauthorized`

Verification: same integration smoke. The route is exercised only in dev compose and is removed (or guarded by an env flag) before production hardening.

### Requirement: R-BAM-021 The middleware emits an OTel span and a slog line per request

Consistent with the project's observability rule, `IdentityFromCookie` MUST
emit an OTel span named `auth.identity_from_cookie` carrying the attributes
`auth.outcome` (`allowed` / `rejected`), `auth.reason` (`missing_cookie` /
`jwe_verify_failed` / `unknown_email` / `ok`), and `auth.email_hash`
(`sha256(email)` truncated to 12 hex chars for PII safety). The matching
`slog` line MUST carry the same fields.

#### Scenario: S-BAM-070 Span and log line on success

- GIVEN a request that the middleware allows
- WHEN the OTel collector receives the batch
- THEN a span named `auth.identity_from_cookie` SHALL be present with
      `auth.outcome=allowed`, `auth.reason=ok`
- AND the matching `slog.Info` line SHALL appear in `docker compose logs database_administrator`

Verification: open Jaeger UI at the agreed service name, locate the span, confirm the attributes; grep the container log.

#### Scenario: S-BAM-071 Span and log line on rejection

- GIVEN a request with a missing cookie
- WHEN the OTel collector receives the batch
- THEN the span SHALL have `auth.outcome=rejected`, `auth.reason=missing_cookie`
- AND the matching `slog.Warn` line SHALL appear

Verification: same as S-BAM-070 with different attribute values.

#### Scenario: S-BAM-072 PII is hashed, not logged raw

- GIVEN a request whose cookie payload's `email = "braejan@example.com"`
- WHEN the span and log line are emitted
- THEN the email SHALL appear ONLY as `auth.email_hash = "<sha256 first 12 hex chars>"`
- AND the literal email SHALL NOT appear anywhere in the span attributes,
      log message, or structured log fields

Verification: `grep -c "braejan@example.com"` against the Jaeger span and
against the structured log line returns 0.

---

## Capability: Configuration and secrets

### Requirement: R-BAM-030 The middleware reads `AUTH_SECRET` from the environment

The composition root MUST read `AUTH_SECRET` (32 bytes base64) from the
environment and pass it as `cfg.AuthSecret` to `IdentityFromCookie`. If
`AUTH_SECRET` is empty or shorter than 32 bytes, the composition root MUST
fail to start with a clear `slog.Error` line and exit 1 (consistent with the
fail-fast rule from R-DBMIG-050 / R-DBMIG-051 in the existing
`db-migrations` spec).

#### Scenario: S-BAM-080 Missing AUTH_SECRET crashes startup

- GIVEN `AUTH_SECRET` is empty
- WHEN `database_administrator` starts
- THEN the process SHALL emit a `slog.Error` line naming `AUTH_SECRET` and
      stating it MUST be set to a 32-byte base64 value
- AND the process SHALL `os.Exit(1)` BEFORE Echo binds the listener

Verification: `docker compose up database_administrator` with `AUTH_SECRET=""`; observe exit 1 and the error log line.

#### Scenario: S-BAM-081 Too-short AUTH_SECRET crashes startup

- GIVEN `AUTH_SECRET="Zm9v"` (3 bytes after decode)
- WHEN `database_administrator` starts
- THEN the process SHALL exit 1 with a clear error message

Verification: same as S-BAM-080 with the short value.

### Requirement: R-BAM-031 The Go service does NOT add CORS `Access-Control-Allow-Credentials`

This slice MUST NOT change `cors.go`. The cookie is set by the Qwik Node
server on its own origin and travels to the Go service via the same-origin
reverse proxy. The Go CORS config continues to forbid credentials. (This is
a non-change requirement, recorded so future maintainers don't accidentally
loosen CORS.)

#### Scenario: S-BAM-090 cors.go is unchanged

- GIVEN the change is applied
- WHEN the orchestrator reads `src/interfaces/http/cors.go`
- THEN the file SHALL NOT contain `Access-Control-Allow-Credentials`
- AND the file SHALL NOT have any diff against the pre-slice version

Verification: `git diff main -- backend/database_administrator/src/interfaces/http/cors.go` returns empty.

---

## Capability: Test coverage contract

### Requirement: R-BAM-100 TDD coverage of the middleware

`interfaces/http/auth_middleware_test.go` MUST contain at minimum:

- `TestIdentityFromCookie_ValidCookie_PopulatesIdentity` (S-BAM-010)
- `TestIdentityFromCookie_TamperedCookie_Returns401` (S-BAM-011)
- `TestIdentityFromCookie_MissingCookie_Returns401` (S-BAM-012)
- `TestIdentityFromCookie_DecryptionShape` (S-BAM-020)
- `TestIdentityFromCookie_NoSetCookieOnSuccess` (S-BAM-030)
- `TestIdentityFromCookie_UnknownEmail_Returns401` (S-BAM-041)
- `TestIdentityFromCookie_LogLineOmitsRawEmail` (S-BAM-072)

The test file MUST use the project's Makefile convention
(`make test` runs `-race -v`) and MUST NOT require a live Postgres
connection (the repository is faked via a struct implementing
`IdentityRepository`).

#### Scenario: S-BAM-110 All middleware tests pass under -race

- GIVEN the change is applied
- WHEN `cd backend/database_administrator && make test` runs
- THEN all 7+ new tests SHALL pass under the `-race -v` flags
- AND no test SHALL panic or leak goroutines
- AND the total test count SHALL increase by at least 7 vs. the pre-slice baseline

Verification: CI job `backend/test` reports the new tests and a race-clean run.

---

## Review checklist

- [ ] reviewer can confirm the spec describes WHAT (capabilities, requirements, scenarios) and not HOW (no jwx API names appear inside requirement bodies beyond "uses lestrrat-go/jwx/v2" in R-BAM-002's name and the ADR reference)
- [ ] reviewer can confirm every scenario uses `GIVEN/WHEN/THEN` and is independently verifiable with a `go test` name or a curl-based integration smoke
- [ ] reviewer can confirm the hexagonal rule is asserted in R-BAM-011 (consistent with the existing `db-migrations` rule)
- [ ] reviewer can confirm the PII hashing requirement is explicit (R-BAM-021, S-BAM-072)
- [ ] reviewer can confirm CORS is explicitly NOT changed (R-BAM-031)
- [ ] reviewer can confirm the middleware is read-only with respect to cookies (R-BAM-003)
- [ ] reviewer can confirm ADR `adr-jwx-for-jwe` is referenced (R-BAM-002) and must be filed at design time
- [ ] reviewer can confirm no `identity.session` table is introduced (intentional per proposal)
