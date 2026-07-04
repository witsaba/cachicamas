# Design: cachicamas-github-login

> **Change**: `cachicamas-github-login`
> **Status**: designed
> **Depends on**: `proposal.md` (Option B locked), `specs/frontend-auth/spec.md`, `specs/backend-auth-middleware/spec.md`, `specs/identity-schema/spec.md`
> **Companion artifacts**: `docs/adr/0001-accept-authjs-qwik.md`, `docs/adr/0002-promote-lestrrat-jwx-for-jwe.md`

## 1. Architecture in one paragraph

GitHub OAuth happens in **Qwik Node SSR** via `@auth/qwik@0.9.2`. On success,
Auth.js encodes a JWE (envelope `alg=dir`, `enc=A256CBC-HS512`) using
`AUTH_SECRET` (raw string), HKDF-SHA256-derived 64-byte key (salt = cookie
name, info = `"Auth.js Generated Encryption Key (cookieName)"`), and stores
it in the `authjs.session-token` cookie on the Qwik origin. The
**Go service** learns a read-only `IdentityFromCookie` Echo middleware that
mirrors Auth.js's key derivation, decrypts the JWE with
`github.com/lestrrat-go/jwx/v2/jwe.Decrypt(... jwa.DIRECT, key)`,
resolves the JWE claims into `identity.user` via a new
`IdentityRepository`, and populates `c.Set("identity", *Identity)` for
downstream handlers. Persistence uses three new goose v3 migration artifacts:
`identity.user` (CITEXT email), `identity.account` (provider records), and
`organization.owner_user_id` (nullable FK).

## 2. Sequence diagrams

### 2.1 First-time GitHub sign-in

```text
Browser                Qwik Node SSR (entry.express.tsx)         GitHub OAuth           Postgres
   │                          │                                      │                     │
   │ 1. GET /                 │                                      │                     │
   │ ───────────────────────► │ routeLoader$ uses useSession()       │                     │
   │                          │ ─ session is empty                    │                     │
   │                          │ render <SignIn button/>              │                     │
   │ ◄───── 200 (HTML form with ───────                            │                     │
   │              hidden  providerId=github)                        │                     │
   │                          │                                      │                     │
   │ 2. POST /auth/signin (body: providerId=github)                  │                     │
   │ ───────────────────────► │ QwikAuth$ useSignIn routeAction$     │                     │
   │                          │ Auth.js generates state + code_verifier/code_challenge (S256)
   │                          │ 302 → github.com/login/oauth/authorize?...                    │
   │ ◄───────────────────────                                      │                     │
   │ 3. user authorizes                                                     │                     │
   │ ────────────────────────────────────────────────────────────────────► │                     │
   │ ◄─────────────────── 302 → <ORIGIN>/auth/callback/github?code=…&state=…                     │
   │ 4. GET /auth/callback/github?code=…            │                     │                     │
   │ ───────────────────────► │ Auth.js onRequest middleware          │                     │
   │                          │ • verifies state (CSRF)               │                     │
   │                          │ • exchanges code for access_token      │                     │
   │                          │ ─────────── POST github.com/login/oauth/access_token ──────►│
   │                          │ ◄──────── access_token + scope ───────│                     │
   │                          │ • GET github.com/user (with token)    │                     │
   │                          │ ───────────►                           │                     │
   │                          │ ◄───────── { id: 12345, login:"braejan", email:"braejan@…" }
   │                          │ • signIn callback invoked              │                     │
   │                          │   looks up identity.user by email       │                     │
   │                          │   INSERT (or reuse) identity.user       │                     │
   │                          │   INSERT identity.account (provider='github', id=12345)
   │                          │ ───────────────────────────────────────────────────────────►│
   │                          │ ◄────────────────── 2 rows ─────────────────────────────────│
   │                          │ • JWE = EncryptJWT(claims).setProtectedHeader({alg:"dir", enc:"A256CBC-HS512", kid:thumbprint}).encrypt(derivedKey)
   │                          │ Set-Cookie: authjs.session-token=<JWE>; HttpOnly; SameSite=Lax; Path=/; (Secure if HTTPS)
   │                          │ 302 → /profile                        │                     │
   │ ◄─────────────────────── │                                      │                     │
   │ 5. GET /profile (Cookie: authjs.session-token=<JWE>)              │                     │
   │ ───────────────────────► │ routeLoader$ calls useSession()       │                     │
   │                          │ useSession reads cookie, decrypts (jose.jwtDecrypt in-process)
   │                          │ returns { user, expires }             │                     │
   │                          │ render <ProfileCard/>                 │                     │
   │ ◄───── 200 (HTML with name+email)               │                     │                     │
```

### 2.2 Protected `/api/*` call (Go side verifies the same cookie)

```text
Browser                Qwik Node SSR (/api/* proxy)         database_administrator (Go)          Postgres
   │                          │                                      │                                │
   │ GET /api/protected/whoami │                                      │                                │
   │   Cookie: authjs.session-token=<JWE>                             │                                │
   │ ───────────────────────► │ entry.express.tsx proxyToApi()         │                                │
   │                          │ strips /api prefix, forwards to       │                                │
   │                          │ http://database_administrator:8080/protected/whoami with same Cookie header
   │                          │ ────────────────────────────────────► │                                │
   │                          │                                      │ IdentityFromCookie middleware  │
   │                          │                                      │ 1. read authjs.session-token │
   │                          │                                      │ 2. hkdf.New(sha256, AUTH_SECRET, │
   │                          │                                      │      "authjs.session-token",  │
   │                          │                                      │      "Auth.js Generated …", 64)│
   │                          │                                      │ 3. jwe.Decrypt(jweBytes,       │
   │                          │                                      │       jwe.WithKey(jwa.DIRECT, derivedKey))│
   │                          │                                      │ 4. decode JWT claims (sub, email, name, picture, exp)│
   │                          │                                      │ 5. SELECT id, email, name, image_url │
   │                          │                                      │    FROM identity.user WHERE lower(email)=lower($1)│
   │                          │                                      │ ─────────────────────────────► │
   │                          │                                      │ ◄─────── row #42 ─────────────│
   │                          │                                      │ 6. c.Set("identity", &Identity{ID:42, Email:"braejan@…"})│
   │                          │                                      │ 7. span auth.identity_from_cookie │
   │                          │                                      │    attrs: outcome=allowed reason=ok email_hash=<sha256-12>│
   │                          │                                      │ handler returns 200 OK JSON    │
   │                          │ ◄────────────────────────────────── │                                │
   │ ◄────── 200 JSON ─────────│                                      │                                │
```

### 2.3 Sign-out

```text
Browser               Qwik Node SSR                       Browser (cookies)
   │                         │                                  │
   │ POST /auth/signout      │                                  │
   │ ───────────────────────►│ useSignOut routeAction$         │
   │                         │ Auth.js clears session store,    │
   │                         │ signs an empty JWT with iat=now  │
   │                         │ Set-Cookie: authjs.session-token=  │
   │                         │   ; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT
   │ ◄──────────────────────│                                  │
   │ next GET /profile       │                                  │
   │ ───────────────────────►│ useSession returns {}            │
   │                         │ redirect → /auth/signin?callbackUrl=%2Fprofile
   │ ◄───── 302 ─────────────│                                  │
```

## 3. The exact JWE envelope and key derivation (load-bearing)

This is the byte-level contract between the Auth.js encoder (Qwik side) and
the Go decoder. **Any drift here is a 401**

### 3.1 Auth.js v5 encoding (verified against `@auth/core@0.34.3/src/jwt.ts`)

```ts
const alg = "dir"                    // RFC 7518 §4.5 — symmetric key used directly
const enc = "A256CBC-HS512"          // RFC 7518 §5.2.5 — AES-256-CBC + HMAC-SHA-512

// Key derivation (RFC 5869 HKDF)
derivedKey = hkdf(
  "sha256",                                                   // HMAC
  AUTH_SECRET,                                                // raw string (NOT base64-decoded)
  cookieName,                                                 // salt = "authjs.session-token" or "__Secure-authjs.session-token"
  `Auth.js Generated Encryption Key (${cookieName})`,         // info
  64,                                                         // bytes = 512 bits (for A256CBC-HS512)
)

// JWE serialization
const thumbprint = jose.calculateJwkThumbprint(
  { kty: "oct", k: base64url(derivedKey) },
  "sha512",                                                   // = sha(byteLength*8) where byteLength=64
)
return new jose.EncryptJWT(token)
  .setProtectedHeader({ alg, enc, kid: thumbprint })
  .setIssuedAt()
  .setExpirationTime(now() + maxAge)                          // default 30 days
  .setJti(crypto.randomUUID())
  .encrypt(derivedKey)
```

Cookie name (per `GetTokenParams` in the same source):

- `secureCookie=true` (production HTTPS): `__Secure-authjs.session-token`
- otherwise (dev HTTP): `authjs.session-token`

The salt in HKDF is the cookie name itself, so dev and prod derive **different
keys** even if `AUTH_SECRET` matches. This means the Go verifier must agree on
which cookie name to use for the salt.

### 3.2 Go decoder (matches §3.1 exactly)

```go
// HKDF-SHA256, length=64, salt=cookieName, info="Auth.js Generated Encryption Key (cookieName)"
derivedKey := hkdf.New(sha256.New, []byte(authSecret),
    []byte(cookieName),
    []byte("Auth.js Generated Encryption Key ("+cookieName+")"),
    64).Read(make([]byte, 64))

// jwx: pass the derived key as a JWK of kty=oct.
key, err := jwk.FromRaw(derivedKey)
if err != nil { … }
_ = jwk.Key

// JWE: alg=dir, enc=A256CBC-HS512
plaintext, err := jwe.Decrypt(
    []byte(cookie),
    jwe.WithKey(jwa.DIRECT /* "dir" */, key),
)
if err != nil { … }

// plaintext is JSON: { "sub", "email", "name", "picture", "iat", "exp", "jti", ... }
```

Cross-tooling round-trip evidence will be committed as a testdata file:
`backend/database_administrator/src/interfaces/http/testdata/authjs_session_token.jwe`.
The test script `backend/database_administrator/scripts/regenerate_authjs_testdata.sh`
re-runs the Auth.js encoder against a known payload and writes the JWE +
expected plaintext JSON. The Go test reads both and asserts byte-equality.
This is the **primary** TDD evidence for `S-BAM-020`.

### 3.3 Why HKDF, not raw key use

If we passed `AUTH_SECRET` as raw 64 bytes to A256CBC-HS512, the JWE
envelope would be the same BUT any future change to `AUTH_SECRET` (rotation
or operator copy-paste) would invalidate every existing session. HKDF +
salt=cookieName means:

1. Salt binds the derived key to the environment (`dev` vs `prod` cookie
   names diverge → keys diverge even if secrets match).
2. Future per-cookie rotation can use distinct salts without changing
   `AUTH_SECRET`.

We MUST use HKDF exactly as Auth.js does. **Do not substitute** a simpler
KDF.

## 4. File layout and responsibilities

### 4.1 New files

```text
frontend/
├── src/
│   ├── routes/
│   │   ├── plugin@auth.ts                                # new — QwikAuth$ call + exports
│   │   └── profile/
│   │       ├── index.tsx                                 # new — server-known profile page
│   │       └── index.spec.tsx                            # new — vitest
│   ├── components/
│   │   └── sign-in-button/
│   │       ├── sign-in-button.tsx                        # new
│   │       └── sign-in-button.spec.tsx                   # new
│   └── lib/
│       └── sign-in-callback.ts                           # new — auto-link + identity.user/account UPSERT
└── e2e/
    ├── github-sign-in.spec.ts                            # new
    ├── sign-in-landing.spec.ts                           # new
    ├── sign-in-cookie-attrs.spec.ts                      # new
    ├── sign-in-denied.spec.ts                            # new
    └── sign-out.spec.ts                                  # new

backend/database_administrator/
├── src/
│   ├── domain/
│   │   ├── identity.go                                   # new — Identity, IdentityRepository, ErrIdentityNotFound
│   │   └── identity_test.go                              # new
│   ├── application/
│   │   └── identity_service.go                           # new (thin — only the lookup; signing is on Qwik side)
│   ├── infrastructure/
│   │   └── postgres/
│   │       └── identity_repository.go                    # new — pgx-backed IdentityRepository
│   ├── interfaces/
│   │   └── http/
│   │       ├── auth_middleware.go                        # new — IdentityFromCookie
│   │       ├── auth_middleware_test.go                   # new
│   │       └── testdata/
│   │           └── authjs_session_token.jwe              # new — fixture JWE
│   └── migration/
│       └── sql/
│           └── 20260703120000_github_login.sql           # new
└── scripts/
    └── regenerate_authjs_testdata.sh                     # new — regenerates the fixture (Node + @auth/core/jwt)

docs/adr/
├── 0001-accept-authjs-qwik.md                            # new
└── 0002-promote-lestrrat-jwx-for-jwe.md                  # new
```

### 4.2 Modified files

```text
frontend/
├── package.json                                          # + @auth/qwik@0.9.2 (pinned exact), + @auth/core
├── package.json                                          # + @panva/hkdf (transitive dep, recorded explicitly)
├── vite.config.ts                                        # + optimizeDeps.include = ["@auth/qwik", "@auth/core", "@panva/hkdf"]
├── tsconfig.json                                         # no functional change unless types surface
├── src/
│   ├── entry.express.tsx                                 # + ensure ORIGIN is read; no logic change
│   └── routes/
│       └── index.tsx                                     # + <SignInButton/> in CTA section
├── e2e/
│   └── (existing create-organization spec, untouched; runs in same compose)
├── Dockerfile                                            # + COPY --from=builder /app/node_modules/@auth
│                                                          # + COPY --from=builder /app/node_modules/@panva
│                                                          # (mirroring the existing undici pattern)
├── README.md                                             # + "Auth.js / GitHub login" section
└── test
    └── (vitest config unchanged)

backend/database_administrator/
├── go.mod                                                # + github.com/lestrrat-go/jwx/v2 v2.x.x
├── go.sum                                                # regenerated
├── src/
│   ├── cmd/server/main.go                                # + load AUTH_SECRET; env-gate dev-only /api/v1/protected/whoami
│   │                                                      #   register IdentityFromCookie on the protected group
│   └── application/
│       └── (organization_service.go untouched)
└── README.md                                             # + "Auth: JWE cookie verification" section, callback URL registration table

openspec/
└── (this design + proposal + specs authored under changes/)

.env.example                                              # + AUTH_GITHUB_ID, AUTH_GITHUB_SECRET, AUTH_SECRET, AUTH_TRUST_HOST=true, AUTH_URL
docker-compose.yaml                                       # frontend service environment block: + 5 vars
```

### 4.3 Not modified (explicit non-changes)

```text
infra/                                                    # unchanged (the identity schema is provisioned; tables come from goose)
docker-compose.vps.yaml                                   # not modified (operators add env via overrides if needed)
backend/database_administrator/src/interfaces/http/cors.go  # explicitly untouched per R-BAM-031
backend/database_administrator/src/migration/postgres/driver.go  # untouched
```

## 5. Configuration before/after diff

### 5.1 `.env.example` — new keys

```diff
 # ─── Frontend (cachicamas-frontend-dockerize) ─────────────────────────────
 # Puerto host del frontend (mapea a :80 en el container nginx).
 # Default 3015. Para VPS, mantener 3015 o ajustar al puerto público deseado.
 FRONTEND_PORT=3015
 # URL que el browser usa para hablar con el Go bin.
 # - En dev local sin compose override: http://localhost:8080 (puerto host).
 # - En VPS con nginx reverse-proxy: /api (el nginx del frontend hace el proxy).
+# ─── Auth: GitHub OAuth via @auth/qwik (cachicamas-github-login) ────────────
+# GitHub OAuth App credentials. Create the OAuth App at
+# https://github.com/settings/developers → New OAuth App. Callback URL
+# (per environment) MUST equal AUTH_URL below + "/auth/callback/github".
+AUTH_GITHUB_ID=replace-me
+AUTH_GITHUB_SECRET=replace-me
+# AUTH_SECRET: 32 random bytes base64-encoded. Generate with:
+#     openssl rand -base64 32
+# MUST be the same across Qwik and Go. MUST never be committed.
+AUTH_SECRET=replace-me-with-32-bytes-base64
+# AUTH_TRUST_HOST=true is required for non-Vercel/Netlify/Cloudflare deploys
+# (qwik.dev docs are explicit). The Qwik Node server can otherwise refuse
+# to construct correct callback URLs in compose / on the VPS.
+AUTH_TRUST_HOST=true
+# Public origin the browser sees. Used by Auth.js for callback construction.
+# Defaults to http://localhost:3015 in dev; override per env.
+AUTH_URL=http://localhost:3015
+# ─── Backend (cachicamas-github-login) ────────────────────────────────────
+# Go service uses the SAME AUTH_SECRET to verify the JWE cookie. The Go
+# service reads this from the host env (pass via docker-compose).
+# (No new var: AUTH_SECRET is duplicated in the database_administrator env block.)
```

### 5.2 `docker-compose.yaml` — frontend service env

```diff
   cachicamas-frontend:
     build: …
     environment:
       NODE_ENV: production
       HOST: 0.0.0.0
       PORT: 3000
       API_TARGET: http://database_administrator:8080
       PUBLIC_API_BASE_URL: "/api"
       SERVER_API_BASE_URL: http://database_administrator:8080
       ORIGIN: ${ORIGIN:-http://localhost:3015}
+      # cachicamas-github-login — Auth.js + GitHub OAuth
+      AUTH_GITHUB_ID: ${AUTH_GITHUB_ID:?AUTH_GITHUB_ID must be set in .env}
+      AUTH_GITHUB_SECRET: ${AUTH_GITHUB_SECRET:?AUTH_GITHUB_SECRET must be set in .env}
+      AUTH_SECRET: ${AUTH_SECRET:?AUTH_SECRET must be set in .env (openssl rand -base64 32)}
+      AUTH_TRUST_HOST: "true"
+      AUTH_URL: ${AUTH_URL:-http://localhost:3015}
```

Rationale: `${VAR:?msg}` is docker-compose's required-env-with-error syntax —
if the operator forgets to set `AUTH_SECRET`, the service refuses to start
with the message baked into the compose file. Consistent with the project's
fail-fast posture (see `db-migrations` R-DBMIG-050).

### 5.3 `docker-compose.yaml` — database_administrator service env

```diff
   database_administrator:
     build: …
     environment:
       SERVICE_ENV: ${SERVICE_ENV:-development}
       OTEL_EXPORTER_OTLP_ENDPOINT: …
       …
+      # cachicamas-github-login — JWE cookie verification (same AUTH_SECRET as the frontend)
+      AUTH_SECRET: ${AUTH_SECRET:?AUTH_SECRET must be set in .env (openssl rand -base64 32)}
```

Note: only `AUTH_SECRET` is shared; the Go service does not need
`AUTH_GITHUB_ID`/`AUTH_GITHUB_SECRET` because it does not perform OAuth —
it only verifies the post-OAuth cookie.

## 6. ADR files (committed under `docs/adr/`)

Per the proposal §7 pre-staging, two ADRs MUST be committed during
`sdd-apply` (before the run-time code change). Both have status `Accepted`
after this slice merges; the running text is below.

### 6.1 `docs/adr/0001-accept-authjs-qwik.md`

```markdown
# ADR 0001: Accept `@auth/qwik@0.9.2` as a direct frontend prod dependency

> Status: Accepted (2026-07-03, cachicamas-github-login)

## Context

The cachicamas frontend needs GitHub OAuth login. The official Auth.js
integration for Qwik is `@auth/qwik` (post-rebrand from
`@builder.io/qwik-auth`). The package is **pre-1.0** (qwik.dev docs are
explicit: "could have bugs"). Latest published version at decision time:
`0.9.2` (2025-10-26). Dependencies tracked in the npm registry: 430 files,
~87KB unpacked, ~757 weekly downloads.

## Decision drivers

- Velocity: the alternative is hand-rolling an OAuth client + CSRF + state +
  PKCE + session encryption in 600+ LOC.
- Standardization: every future OAuth provider (Google, Microsoft, generic
  OIDC) reuses the same Auth.js core. No new dep per provider.
- Security baseline: Auth.js handles double-submit CSRF cookie, the
  state parameter, and PKCE by default.

## Options considered

- `@auth/qwik` — chosen. Risk: pre-1.0.
- `@builder.io/qwik-auth` — rejected. Deprecated upstream.
- Hand-rolled OAuth client — rejected. ~600 LOC, three different CSRF
  strategies to choose from, hard to audit, no upgrade path for new
  providers.

## Decision

Accept `@auth/qwik@0.9.2` as a direct prod dependency, pinned **exactly**
(no caret). Track upstream releases; an upgrade is its own change behind a
regression test gate.

## Consequences

- We ship a pre-1.0 dependency. Pre-merge: `pnpm audit` (no high-severity
  CVEs at decision time) + one Playwright spec exercising the sign-in flow.
- When Auth.js v1 ships, the upgrade path is `npm i @auth/qwik@^1.0.0` +
  re-run the Playwright suite.
- If `@auth/qwik` is abandoned, fallback is documented in
  `docs/fallback-auth.md` (out of scope this slice).
```

### 6.2 `docs/adr/0002-promote-lestrrat-jwx-for-jwe.md`

```markdown
# ADR 0002: Promote `github.com/lestrrat-go/jwx/v2/jwe` to verify Auth.js JWE cookies on Go

> Status: Accepted (2026-07-03, cachicamas-github-login)

## Context

The cachicamas Go service (`database_administrator`) needs to verify JWE
cookies issued by Auth.js for Qwik (envelope `alg=dir`, `enc=A256CBC-HS512`,
64-byte symmetric key derived from `AUTH_SECRET` via HKDF-SHA256).
Implementing this from scratch in stdlib-only Go is ~300 LOC of fragile
crypto (AES-256-CBC + HMAC-SHA-512 + IV generation + AEAD tagging + HKDF).

## Decision drivers

- Bit-exact correctness with Auth.js's encoder. Verified against
  `@auth/core@0.34.3/src/jwt.ts`.
- Active maintenance: `lestrrat-go/jwx` v2 is the de-facto Go JOSE
  library (also used by `lestrrat-go/jwx/v2/jwk` and `/jwt`).
- Adapters to jwe.WithKey + jwk.FromRaw make the call site readable.

## Options considered

- `github.com/lestrrat-go/jwx/v2/jwe` + `golang.org/x/crypto/hkdf` —
  chosen. ~15 LOC of code on top.
- `github.com/golang-jwt/jwt/v5` — rejected. JWS-focused; JWE support
  requires extra wiring.
- Hand-rolled — rejected. Crypto is hard to get right; version skew
  between Go's `crypto/aes` and Auth.js's `jose` is a real audit risk.
- `github.com/go-jose/go-jose/v4` — considered. Larger API surface
  than needed; jwx covers our two-call footprint.

## Decision

Promote `github.com/lestrrat-go/jwx/v2` (current minor) to a direct
dependency in `backend/database_administrator/go.mod`. Use
`jwe.Decrypt(..., jwe.WithKey(jwa.DIRECT, jwk))` with the HKDF-derived
64-byte key from `golang.org/x/crypto/hkdf`.

## Consequences

- Cross-tooling byte equality is asserted by the fixture test in
  `interfaces/http/testdata/authjs_session_token.jwe` (R-BAM-020).
- The HKDF parameters (sha256, salt=cookieName, info="Auth.js Generated
  Encryption Key (cookieName)", len=64) MUST match Auth.js byte for byte.
  They live in `interfaces/http/auth_middleware.go` `deriveKey(cfg)` which
  is the single source of truth.
```

## 7. Test strategy

### 7.1 Frontend (Vitest, no DB needed)

| Spec | File | Asserts |
| --- | --- | --- |
| `plugin@auth.test.ts` | `frontend/src/routes/plugin@auth.test.ts` | `plugin@auth.ts` exports `onRequest`, `useSession`, `useSignIn`, `useSignOut`; `QwikAuth$` is called with `GitHub`. |
| `sign-in-button.test.tsx` | `frontend/src/components/sign-in-button/sign-in-button.spec.tsx` | Form has hidden `providerId=github`; visible text "Sign in with GitHub". |

### 7.2 Frontend (Playwright, requires docker compose + GitHub test account)

| Spec | File | Asserts |
| --- | --- | --- |
| `sign-in-landing.spec.ts` | `frontend/e2e/sign-in-landing.spec.ts` | Sign-in button visible when signed out; hidden when signed in. |
| `github-sign-in.spec.ts` | `frontend/e2e/github-sign-in.spec.ts` | Full OAuth roundtrip → `/profile` renders the right user; identity rows in Postgres. |
| `sign-in-cookie-attrs.spec.ts` | `frontend/e2e/sign-in-cookie-attrs.spec.ts` | Cookie attrs HttpOnly / SameSite=Lax / Path=/ ; `__Secure-` prefix when HTTPS. |
| `sign-in-denied.spec.ts` | `frontend/e2e/sign-in-denied.spec.ts` | `error=access_denied` lands on an error page (no infinite redirect). |
| `sign-out.spec.ts` | `frontend/e2e/sign-out.spec.ts` | Cookie cleared; `/profile` redirects to `/auth/signin`. |

For CI determinism without a live GitHub account:

- A **GitHub OAuth simulator** compose service (`mocks/github-oauth`) runs
  alongside the dev stack; the Playwright spec points the Qwik origin at
  `mocks/github-oauth` in CI via `AUTH_GITHUB_ID`/`AUTH_GITHUB_SECRET` env
  overrides. This is implementation detail of the test infra (out of the
  spec scope).

### 7.3 Backend (Go, run via `make test`)

| Test | File | Asserts |
| --- | --- | --- |
| `TestIdentityFromCookie_ValidCookie_PopulatesIdentity` | `interfaces/http/auth_middleware_test.go` | S-BAM-010. Uses the fixture JWE. |
| `TestIdentityFromCookie_TamperedCookie_Returns401` | same | S-BAM-011. |
| `TestIdentityFromCookie_MissingCookie_Returns401` | same | S-BAM-012. |
| `TestIdentityFromCookie_DecryptionShape` | same | S-BAM-020. Asserts the JSON shape `{sub, email, name, picture, iat, exp, jti}`. |
| `TestIdentityFromCookie_NoSetCookieOnSuccess` | same | S-BAM-030. |
| `TestIdentityFromCookie_UnknownEmail_Returns401` | same | S-BAM-041. |
| `TestIdentityFromCookie_LogLineOmitsRawEmail` | same | S-BAM-072. |
| `TestMain_RejectsEmptyAuthSecret` | `cmd/server/main_test.go` | S-BAM-080. |
| `TestMain_RejectsShortAuthSecret` | same | S-BAM-081. |
| `TestMain_RejectsCORS_CredentialsNotAdded` | `interfaces/http/cors_test.go` (new) | S-BAM-090. |
| `TestIdentityRepository_LookupByEmail_Hit` | `infrastructure/postgres/identity_repository_test.go` | (new; integration) |
| `TestIdentityRepository_LookupByEmail_Miss` | same | (new; integration) |

All Go tests run under `make test` (`go test -race -v ./...`).
Integration tests require the dev compose stack (`make test/integration`).

### 7.4 Migration (Postgres)

| Test | Where | Asserts |
| --- | --- | --- |
| `TestMigration_Applies20260703120000` | runner test | S-IS-070. |
| `TestMigration_RollsBackCleanly` | runner test | S-IS-061. |
| `TestMigration_PostgresIntegration` | integration | (full up + check constraints). |

## 8. Rollout

### 8.1 Day 0 (this slice)

1. Operator creates the GitHub OAuth App at
   <https://github.com/settings/developers> → New OAuth App.
   - Homepage URL: `http://localhost:3015` (dev)
   - Authorization callback URL: `http://localhost:3015/auth/callback/github`
2. Operator copies the Client ID + Secret into `.env`.
3. Operator generates `AUTH_SECRET` (`openssl rand -base64 32`) and writes it
   to `.env` TWICE (frontend block + database_administrator block).
4. `docker compose -f docker-compose.yaml up -d --build` — boots the stack.
5. Operator visits `http://localhost:3015`, clicks "Sign in with GitHub",
   completes the OAuth flow, lands on `/profile` with the username visible.

### 8.2 Pre-VPS cutover (deferred change)

- Register a second GitHub OAuth App for the VPS (production callback URL).
- Add the VPS `AUTH_GITHUB_ID`/`AUTH_GITHUB_SECRET` to the VPS `.env` override.

### 8.3 Out of scope (deferred)

- Add Google / Microsoft provider — follow-up change.
- Switch from JWE cookie to DB-backed sessions — follow-up change (would
  also unblock server-side revoke).
- Rotate `AUTH_SECRET` (operator docs in README appendix).
- Add `identity.audit` table.
- Add CSP / HSTS headers.

## 9. Risks specific to design

| Risk | Severity | Mitigation |
| --- | --- | --- |
| HKDF parameters drift from Auth.js source | High | Cross-tooling fixture test (S-BAM-020) is the single source of truth; regenerate script is CI-verified. |
| `@auth/qwik` pre-1.0 ships a breaking change | Med | Pinned exact version + Playwright regression guard; the upgrade is its own change. |
| Go verifier accepts `alg=dir` cookies but future Auth.js ships `alg=A256KW` (key wrapping) | Low | Spec the allowed `alg` list explicitly in the verifier (only `dir`); any other `alg` returns 401. |
| Dockerfile `COPY` for `@auth/*` dynamic imports is incomplete | Med | Inspect `frontend/server/` output of `pnpm build`; add missing modules to the `COPY --from=builder` pattern. (Identified during apply.) |
| `infra/postgres/init/01-init.sql` already installs `citext` extension; race with first boot migration | Low | Verified during apply: extension is in `01-init.sql` (idempotent `CREATE EXTENSION IF NOT EXISTS`); migration does NOT re-create. |

## 10. Open items still open (carry to `sdd-tasks`)

- Task planning around Dockerfile COPY list — apply-time discovery.
- Plan for the GitHub OAuth simulator compose service (test infrastructure;
  lives in `frontend/e2e/` or `scripts/mocks-github-oauth/`).
- Pin `golang.org/x/crypto` minor version in `go.mod` for HKDF stability.
- Pin `@panva/hkdf` minor version in `frontend/package.json` (transitive, but
  we use it explicitly when generating fixtures).
