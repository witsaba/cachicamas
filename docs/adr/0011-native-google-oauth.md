# ADR 0011 — Native Google OAuth 2.0 + HMAC session cookies (replaces Auth.js / JWE)

> Status: **Accepted** (2026-09-04). Implements the dependency admission
> `openspec/AGENTS.md` Hard Rule 5 ("New top-level dependency ⇒ ADR first") and
> `openspec/config.yaml` `apply.no-new-top-level-deps-without-an-ADR` require.
> Implements: `cachicamas-google-auth-bootstrap` change (4-PR stacked-to-main chain).
> Companion: this ADR is merged in the same change as the first production import
> that requires `github.com/oschwald/geoip2-golang` (PR-2 — already on main), and
> is the architectural record for the HMAC session-cookie pivot that retires the
> predecessor Auth.js / JWE surface (PR-1, PR-2, PR-3, and PR-4 are the four
> chained PRs that together satisfy this ADR).
> Supersedes: `openspec/specs/backend-auth-middleware/spec.md` (JWE/A256GCM
> verifier from the abandoned `cachicamas-github-login` slice) — see
> [§Superseded predecessor](#superseded-predecessor).

---

## Resolved TOC

- [Context](#context)
- [Decision](#decision)
- [Alternatives considered](#alternatives-considered)
- [Consequences](#consequences)
- [Operational notes](#operational-notes)
- [Superseded predecessor](#superseded-predecessor)
- [References](#references)

---

## Context

Cachicamas pivoted in 2026-07 from the SDLC-agentic framing to the
multiplayer-agentic framing for PYMEs (LatAm SMBs) — recorded in
[ADR 0009 § Vision](./0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md).
The product's user is a PYME owner running on Google Workspace; the
authentication surface must reach them through Google's identity provider and
must speak Spanish as a first-class language.

The prior auth slice (`cachicamas-github-login`, archived 2026-07-04 in favour
of the Google pivot) reached for GitHub OAuth through `@auth/qwik` +
`@auth/core` + a JWE/A256GCM session-cookie verifier written in Go. That
slice shipped only partially: the schema was provisioned (`identity.user` /
`identity.account`), a `lestrrat-go/jwx/v2` verifier was drafted, and the
spec was written. Three regressions surfaced during the
`gentle-ai.review` review:

1. **CVE surface from `@auth/core`.** Auth.js is an unmaintained-for-our-shape
   surface; the review surfaced an active CVE class in its token-exchange
   path that would require pinning + backporting to keep our dependency tree
   clean. We do not need any of `@auth/core`'s provider abstractions to call
   one OAuth provider ourselves.
2. **Tenancy model opacity.** `@auth/qwik`'s tenancy model is "identity-only";
   you bolt your own tenancy on top. The whole point of our MVP tenancy
   decision (Engram #4219, one Google user = one organization) is that
   *identity = tenant = organization*. Putting `@auth/qwik` in front forces
   us to bypass its callback signature to inject our own
   `users → organizations` write, which means we own the broken invariants
   without owning the clean ones.
3. **JWE for an opaque payload.** The HMAC payload we actually want to sign is
   `{user_id, organization_id, expires_at}` — nothing sensitive enough to
   require encryption. A256GCM-HS256 adds CPU, key-rotation complexity, and a
   second secret to manage, for no measurable defence-in-depth on a payload
   that's already authenticated.

Concurrently, the `gentle-ai.review` review surfaced a new top-level Go
dependency — `github.com/oschwald/geoip2-golang` for best-effort IP
geolocation enrichment of `auth.login_audits`. That dep requires this ADR
under `openspec/AGENTS.md` Hard Rule 5.

This change replaces the abandoned Auth.js + JWE surface with a native
Google OAuth 2.0 (authorization code flow) implementation in Qwik City,
HMAC-SHA256 session cookies, and a shared-secret service-to-service header
between Qwik and the Go backend. No auth library; one new top-level Go dep
(geoip2), admitted on the rationale below.

---

## Decision

### D1 — No auth library on either side of the wire

Pure Qwik City + Node 20 stdlib (`crypto.subtle`, `fetch`) on the frontend;
pure Go stdlib (`crypto/hmac`, `crypto/sha256`, `encoding/base64`) on the
backend. The previous `@auth/qwik`, `@auth/core`, `@panva/*`, and `jose`
imports are removed (PR-3 lands zero prohibited imports; PR-3 verify report
confirms via `grep -rE "@auth|@panva|jose|lucia|iron-session"` returning
0 hits).

### D2 — HMAC-SHA256 session cookies, not JWE

The session cookie envelope is:

```
cachicamas_session = base64url(JSON) + "." + base64url(HMAC-SHA256(secret, base64url(JSON)))
```

where the payload is `{user_id, organization_id, expires_at, iat}`. The
secret is `AUTH_COOKIE_SECRET` (a single env var on both Qwik and Go).
Verification is constant-time (`crypto.timingSafeEqual` on Node,
`hmac.Equal` on Go). The cookie is `HttpOnly; Secure (in production);
SameSite=Lax; Path=/; Max-Age=604800`.

There is no encryption: the payload is non-sensitive (user id +
organization id + expiry). The integrity guarantee is the entire contract.

### D3 — One user = one organization (MVP tenancy)

Locked by Engram #4219 and reaffirmed in the change's spec (R-DB-003,
R-BE-002): `auth.users` and `auth.organizations` are paired 1:1 via
`auth.organizations.owner_id REFERENCES auth.users(id)`. Email is the
main identifier (app-layer lowercased). `google_sub` is the
provider-specific secondary key and the lookup index for `/internal/auth/bootstrap`.
Multi-organization tenancy is deferred to a follow-up change.

### D4 — Native Google OAuth 2.0 (authorization code flow)

`routes/auth/google/login` generates a 32-byte CSRF `state` via
`crypto.getRandomValues`, stores it in a `cachicamas_oauth_state` cookie
(`HttpOnly; SameSite=Lax; Max-Age=600`), and 302s to
`https://accounts.google.com/o/oauth2/v2/auth` with `scope=openid email profile`,
`state`, `redirect_uri`, `response_type=code`.

`routes/auth/google/callback`:
1. verifies the state cookie ≡ query-string state (else
   `/auth/error?reason=invalid_state`),
2. exchanges the code at `https://oauth2.googleapis.com/token`
   (form-encoded POST, server-only — `AUTH_GOOGLE_SECRET` never reaches
   the browser),
3. fetches `https://openidconnect.googleapis.com/v1/userinfo` with the
   bearer access token,
4. POSTs the claims to `http://database_administrator:8080/internal/auth/bootstrap`
   with `X-Internal-Secret: ${AUTH_INTERNAL_SECRET}`,
5. inspects the bootstrap `status` (`blocked` ⇒ `/auth/error?reason=blocked`
   with no session cookie; `inactive` and `active` ⇒ continue),
6. HMAC-signs the session payload with `AUTH_COOKIE_SECRET`,
7. sets the `cachicamas_session` cookie, and
8. 302s to `/home`.

PKCE is intentionally omitted: the OAuth client is confidential and
server-side; the secret never leaves Node SSR. PKCE is a defence-in-depth
that only matters if the secret leaks, and if it leaks, the attacker can
already exchange codes. Add later if a public-client (mobile/SPA) surface
ships.

### D5 — Registration date = first successful login

`auth.users.created_at` is set on the first successful `/internal/auth/bootstrap`
and is **immutable thereafter** at the application layer
(via an explicit `UPDATE` column allowlist in the Go repository — see
PR-2's `UpdateLoginFields` helper). The audit table `auth.login_audits`
captures every attempt (success or failure) with IP, user-agent, optional
GeoIP enrichment, and a `failure_reason` taxonomy (R-BE-007 in the spec).

### D6 — Service-to-service via shared secret, not mTLS

`X-Internal-Secret` (a single `openssl rand -base64 32` value shared between
Qwik and Go) gates every `/internal/auth/*` and `/internal/me/*` request.
The middleware compares the header in constant time and returns 401 on
mismatch. There is no rate limiting in this slice (deferred to the
`cachicamas-auth-rate-limit` follow-up).

### D7 — Admit `github.com/oschwald/geoip2-golang` v1.13.0 as a top-level Go dep

This is the ADR's reason for existence under `openspec/AGENTS.md` Hard
Rule 5. The dep is pinned to `v1.13.0` (already in
`backend/database_administrator/go.mod` as `// indirect` since PR-2). The
use case is best-effort IP geolocation enrichment of
`auth.login_audits.country_code` and `auth.login_audits.city` during
`POST /internal/auth/bootstrap`.

- When `GEOIP_DB_PATH` is empty or unset → `Enrich(ip)` returns
  `("", nil)` (empty string, nil error); the audit row is written with
  NULL country and city; the bootstrap call still succeeds.
- When `GEOIP_DB_PATH` points to a readable `.mmdb` and the IP is in the
  DB → the audit row is written with populated country and city.
- When the DB file is missing or unreadable → same as the empty case
  (silent skip, no fatal error).

This is the trade-off: a new top-level dep in exchange for a small piece
of operational telemetry. Mitigated by the silent-skip posture: the
service never fails closed on GeoIP state.

### D8 — Native `Bootstrap` service replaces the predecessor `Identity` port

`application/auth_service.Bootstrap(ctx, claims, ip, ua, geo)` orchestrates
the lookup-or-create on `auth.users`, the lookup-or-create on
`auth.organizations`, and the `auth.login_audits` insert inside a single
`*sql.Tx` (atomicity locked by spec S-BE-020). The predecessor's
`domain.Identity`, `domain.IdentityRepository`,
`domain.IdentityNotFoundError`, and `interfaces/http.IdentityFromCookie`
ports are gone. Cleanup is tracked under §Follow-ups.

---

## Alternatives considered

1. **`@auth/qwik` 0.9.2 + `@auth/core` 0.41.3`** — rejected.
   * Experimental for Qwik; identity-only model that forces us to bypass
     its callback signature to inject our own `users → organizations`
     write (D3 in §Decision).
   * Active CVE class in its token-exchange path (surfaced by the
     `gentle-ai.review` review).
   * Carries `jose`, `@panva/*`, `iron-session`, and other transitive deps
     we do not need and do not want.
   * Same ADR requirement, worse fit on every axis that matters.

2. **Lucia** — rejected.
   * Heavy abstraction for an MVP. Adds a session abstraction layer that
     we would either implement ourselves anyway or replace in the
     multi-organization follow-up.
   * Renamed/re-licensed in 2024; the migration story for a future bump
     is not stable enough to bet a tenant surface on.

3. **Clerk / Auth0 / WorkOS** — rejected.
   * Third-party SaaS contradicts the project's "build our own first"
     posture (recorded in the proposal §Locked Decisions).
   * Vendor lock-in on the auth surface is exactly what we are trying
     to avoid by shipping native.
   * Per-user pricing on a SMB onboarding ramp is a non-starter for
     the MVP.

4. **`lestrrat-go/jwx/v2` JWE / A256GCM-HS256** — rejected (predecessor).
   * HMAC is simpler for an opaque payload that contains no
     confidentiality-sensitive data.
   * A256GCM-HS256 doubles the secret-management surface (encryption
     key + auth key) without doubling the security guarantee.
   * Carries a larger dep tree (`github.com/lestrrat-go/jwx/v2`,
     `github.com/lestrrat-go/jwx/v2/jwe`, `github.com/lestrrat-go/iter`,
     `github.com/decred/dcrd/dcrec/secp256k1/v4`, etc.).

5. **Native GeoIP via direct MaxMind DB parsing (no `oschwald/geoip2-golang`)** — rejected.
   * The MaxMind binary format is documented but not trivial to parse
     correctly under memory pressure (the record format includes
     variable-length lookups).
   * `oschwald/geoip2-golang` is the de-facto standard for Go consumers
     of GeoLite2; its tests cover the record-format edge cases.
   * Same ADR requirement (the stdlib shape would still need a new
     third-party format parser); worse fit.

6. **Defer GeoIP entirely** — partially rejected.
   * The MVP audit table is enriched at write time, not at read time, so
     leaving GeoIP out means the country/city columns stay NULL forever
     for historical rows. Operational telemetry is the smallest useful
     piece we can add.
   * Best-effort posture (D7) means the cost is one indirect dep that
     never fires; the operational value is real.

---

## Consequences

### Positive

- **We own the full OAuth handshake.** State CSRF, code exchange, token
  refresh, and the bootstrap transaction all live in our code. Tenancy
  rules live in our code, not in opaque library callbacks. This is the
  primary architectural benefit.
- **Smaller dependency surface.** Zero auth-library deps on either side
  of the wire. The only new top-level Go dep is `geoip2-golang` v1.13.0,
  which is best-effort and never fatal. No transitive `@panva/*`,
  `iron-session`, `jose`, `lestrrat-go/*`, or `dcrec/secp256k1` carry-over.
- **No Auth.js CVE exposure.** The active CVE class in `@auth/core`'s
  token-exchange path (surfaced by the review) is eliminated at the
  source.
- **Easy to add GitHub / Microsoft later.** The shape is:
  `routes/auth/github/login + /auth/github/callback`, plus a new
  `provider` discriminator in `auth.login_audits.provider` (already in
  the schema as `VARCHAR(32) NOT NULL DEFAULT 'google'`). Same
  callback-handler shape; same backend `/internal/auth/bootstrap`
  shape. The Google and GitHub login routes can share the cookie /
  bootstrap code via a parameterised helper.
- **One env var per concern.** `AUTH_GOOGLE_ID` /
  `AUTH_GOOGLE_SECRET` (Google Cloud Console), `AUTH_COOKIE_SECRET`
  (HMAC), `AUTH_INTERNAL_SECRET` (Qwik↔Go), `GEOIP_DB_PATH`
  (optional, empty disables). Rotation cadence is documented per var.
- **Audit trail is a database, not an opaque library log.** Every
  bootstrap call writes one `auth.login_audits` row (R-DB-004) with
  IP, user-agent, optional GeoIP, success/failure, and a stable
  failure-reason taxonomy.

### Negative / risk

- **We own OAuth correctness.** State CSRF, code-exchange error
  handling, token refresh, and redirect-URI exact-match all live in
  our code. The mitigation is the strict-TDD contract (the 116 PR-3
  tests cover the locked scenarios in spec S-FE-001..S-FE-014 + R-NFR-001).
  Future providers must be added under the same discipline.
- **Session management is custom.** Revocation today requires either
  rotating `AUTH_COOKIE_SECRET` (nuclear) or adding a `auth.sessions`
  table with a server-side session id (the MVP uses an opaque
  user-id-backed cookie and treats expiry as the only revocation
  mechanism). The DB-backed session table is recorded as the
  follow-up; the MVP accepts the limit because Google handles MFA and
  password reset, and the cookie expires in 7 days.
- **Two new top-level deps net of this change.** The one admitted in
  this ADR is `github.com/oschwald/geoip2-golang` v1.13.0 (D7).
  The other (`github.com/jackc/pgx/v5` and its `stdlib` subpackage) was
  already present via the `database_administrator` module; PR-1's
  migration runner pulls it. No new dep in `backend/agent` or in
  `frontend/`.
- **PKCE intentionally omitted.** Recorded in the OAuth callback's
  file header. Add later if a public-client surface ships (mobile / SPA).
- **JWE secret rotation is gone.** `AUTH_COOKIE_SECRET` rotation
  invalidates every existing session at rotation time (no two-secret
  overlap). Acceptable for an MVP; the migration story for an
  in-flight rotation is a follow-up.
- **Sliding session refresh only fires on `(app)` routes.** Users who
  land on the marketing site for 8+ days will need to re-login on
  next `(app)` visit. Acceptable per spec R-NFR-002 (the marketing
  site is not a protected handler).

### Neutral

- **GeoIP disabled is silent.** When `GEOIP_DB_PATH` is empty or
  unset, `Enrich(ip)` returns `("", nil)`; the service still starts,
  the bootstrap call still succeeds, and the audit row has NULL
  country and city. The service logs `geoip disabled (no DB at
  GEOIP_DB_PATH)` once at startup (operator-facing log, not user-facing).
- **GeoIP enabled is best-effort.** When the DB is loaded but the IP
  is not in the DB, the audit row gets NULL country and city. Same
  shape as the disabled case. The mitigation is the same: no fatal
  error path on GeoIP state.
- **GeoIP DB file is a host bind.** Mounted via
  `./geoip-data:/var/lib/geoip:ro` in `docker-compose.yaml` (commented
  out by default; see PR-4 commit `e00c0dc5`). The DB itself is not
  shipped in the repo (MaxMind's free tier requires an account +
  license key).

---

## Operational notes

### Required env vars

| Var | Source | Required | Rotation cadence |
|---|---|---|---|
| `AUTH_GOOGLE_ID` | Google Cloud Console OAuth client ID | yes (frontend) | per Google Cloud Console rotation |
| `AUTH_GOOGLE_SECRET` | Google Cloud Console OAuth client secret | yes (frontend) | per Google Cloud Console rotation |
| `AUTH_COOKIE_SECRET` | `openssl rand -base64 32` | yes (frontend + backend) | per session-rotation policy; rotation invalidates all sessions |
| `AUTH_INTERNAL_SECRET` | `openssl rand -base64 32` | yes (frontend + backend) | per request-rotation policy; old value overlaps with new during rollout |
| `GEOIP_DB_PATH` | host path to `GeoLite2-City.mmdb` | no (empty disables) | per MaxMind quarterly release |

### Compose wiring

PR-4 (`chore(docker): wire Google OAuth env vars to frontend service`
and `chore(docker): wire Google OAuth env vars to backend service`)
updates `docker-compose.yaml` so:

- The `frontend` service receives `AUTH_GOOGLE_ID`,
  `AUTH_GOOGLE_SECRET`, `AUTH_COOKIE_SECRET`, `AUTH_INTERNAL_SECRET`,
  `GEOIP_DB_PATH`, `PUBLIC_BASE_URL`, `PUBLIC_AUTH_REDIRECT_URI`, and
  `PUBLIC_GO_BACKEND_URL` (the last three as build args too, so Qwik
  inlines them into the client bundle).
- The `database_administrator` service receives `AUTH_INTERNAL_SECRET`
  and `GEOIP_DB_PATH`. The optional `./geoip-data:/var/lib/geoip:ro`
  volume mount is documented as a comment block (commenting out the
  whole `volumes:` block would leave YAML parsing ambiguous, so the
  mount is shown as a comment with explicit uncommenting instructions).

### Smoke test

A manual end-to-end smoke is documented in the PR-4 apply-progress
report (Engram obs #4226, PR-4 revision). The test plan covers:

- `docker compose up -d` succeeds (with `.env` populated).
- `make migrate-up` in `backend/database_administrator` applies PR-1's
  migration (already on main) so `auth.*` tables exist.
- A browser visit to `http://localhost:5173/auth/google/login`
  redirects to Google and back to `/auth/google/callback?code=…&state=…`.
- `/home` shows the authenticated user's email + name (from
  `/internal/me/:user_id`).
- `auth.users`, `auth.organizations`, and `auth.login_audits` have one
  row each (verified via `psql`).
- Logout clears the session cookie and 302s to `/`.

The smoke itself is a manual flow that requires a human + a real Google
test client; it is **not** automated in CI.

---

## Superseded predecessor

This ADR formally retires the predecessor JWE / Auth.js surface that the
abandoned `cachicamas-github-login` slice described. The retirement
landed across PR-1 (drop `identity.user` / `identity.account` and rewrite
the FKs), PR-2 (replace `Identity` port with `auth.users` /
`auth.organizations` repositories), PR-3 (replace `@auth/qwik` /
`@auth/core` imports with native Qwik City routes), and PR-4 (this ADR
+ env + docker wiring).

The predecessor spec lives at
`openspec/specs/backend-auth-middleware/spec.md` and is marked
**SUPERSEDED** in the change's spec (Engram #4222 §10
`SUPERSEDED-REFERENCE`). Formally archiving that spec is tracked as a
follow-up change (`cachicamas-jwe-spec-archive`) so future work does not
reference the JWE spec as the source of truth.

---

## Follow-ups

| ID | Change | Why deferred |
|---|---|---|
| F-1 | `cachicamas-auth-rate-limit` — IP-based throttling on `/internal/auth/*` | Out of MVP slice; design notes the closed rate-limit port as a follow-up |
| F-2 | `cachicamas-multi-organization-tenancy` — `user_organizations` join + invitations + roles | Tenancy model locked to 1:1 by Engram #4219; multi-organization is a product decision for later |
| F-3 | `cachicamas-db-session-table` — `auth.sessions` table for server-side revocation | MVP uses opaque cookie + DB-backed user; revocation requires session table |
| F-4 | `cachicamas-additional-providers` — GitHub / Microsoft / Apple login | Schema (`login_audits.provider`) is designed for it; callback shape is parameterised |
| F-5 | `cachicamas-jwe-spec-archive` — formally archive the predecessor JWE spec | Until archived, the JWE spec at `openspec/specs/backend-auth-middleware/spec.md` remains the canonical spec file (SUPERSEDED in this change's spec body) |
| F-6 | `cachicamas-identity-port-cleanup` — remove `domain.Identity`, `domain.IdentityRepository`, `domain.IdentityNotFoundError`, `interfaces/http.IdentityFromCookie` | Out of MVP slice; the JWE verifier surface is gone, the port type is no longer referenced from any production code path |
| F-7 | ESLint enforcement rule for prohibited auth imports (S-FE-060) | PR-3 verify report flagged the behaviour as correct (zero prohibited imports) but the enforcement rule as absent; add as a small follow-up |
| F-8 | PKCE for public-client OAuth surface | Deferred until a public-client (mobile / SPA) surface ships |

---

## References

- [ADR 0009](./0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md) —
  product pivot from SDLC agentic → multiplayer agentic; the context
  this ADR operates under.
- [ADR 0005 § D1](../0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) —
  Layer 3 reaches Layer 1 through ports, not vendor SDKs; the
  stdlib-shaped HMAC verifier on both sides of the wire preserves this.
- `openspec/AGENTS.md` "Hard rules" item 5 — "New top-level dependency ⇒ ADR first".
- `openspec/config.yaml` `apply.no-new-top-level-deps-without-an-ADR` — the gate
  this ADR satisfies for `github.com/oschwald/geoip2-golang` v1.13.0.
- Proposal: Engram `obs-…` (`sdd/cachicamas-google-auth-bootstrap/proposal`, observation id 4221).
- Spec: Engram `obs-1cf173b37c11df89` (`sdd/cachicamas-google-auth-bootstrap/spec`, observation id 4222).
  Includes §10 SUPERSEDED-REFERENCE that retires the predecessor JWE spec.
- Design: Engram `obs-e96ad91c117a947a` (`sdd/cachicamas-google-auth-bootstrap/design`, observation id 4223).
- Tasks: Engram `obs-c7e2e6bde85c5758` (`sdd/cachicamas-google-auth-bootstrap/tasks`, observation id 4225).
- Apply progress (PR-1..PR-4): Engram observation id 4226.
- Verify report (PR-1..PR-3): Engram observation id 4228. PR-4 verify is
  the next SDD phase after this ADR is merged.
- `gentle-ai.review` review of the predecessor slice — surfaced the CVE
  class in `@auth/core` that motivated the D1 pivot.
- `backend/database_administrator/go.mod` — `github.com/oschwald/geoip2-golang v1.13.0`
  is the dep admitted by D7. Already present as `// indirect` since PR-2.
- Predecessor JWE spec: `openspec/specs/backend-auth-middleware/spec.md` —
  **SUPERSEDED**. Authored under Engram obs `sdd/cachicamas-github-login/spec`
  (id 1729). Reviewers MUST NOT cite it as the source of truth for new work.
