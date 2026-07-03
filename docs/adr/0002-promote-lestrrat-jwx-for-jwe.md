# ADR 0002: Promote `github.com/lestrrat-go/jwx/v2/jwe` to verify Auth.js JWE cookies on Go

> Status: **Accepted** (2026-07-03, change `cachicamas-github-login`)
> Author: cachicamas SDD pipeline (parent orchestrator)
> Companion to: ADR 0001

## Context

The cachicamas Go service (`database_administrator`) needs to verify JWE
session cookies issued by `@auth/qwik` when a user signs in via GitHub.
The full envelope contract (verified against `@auth/core@0.34.3/src/jwt.ts`):

- `alg = "dir"` — RFC 7518 §4.5 — symmetric key used directly.
- `enc = "A256CBC-HS512"` — RFC 7518 §5.2.5 — AES-256-CBC +
  HMAC-SHA-512.
- **Key derivation**: HKDF-SHA256 (`@panva/hkdf`) over `AUTH_SECRET`
  (raw UTF-8 string from env, NOT base64-decoded), `salt = cookieName`,
  `info = "Auth.js Generated Encryption Key (cookieName)"`, `length = 64`.
- **Cookie name**: `authjs.session-token` (dev, HTTP) or
  `__Secure-authjs.session-token` (production, HTTPS) — drives the salt.

Implementing this from scratch in stdlib-only Go would be:

- HKDF (RFC 5869) — stdlib `crypto/sha256` + an HKDF constructor;
  ~30 LOC.
- A256CBC-HS512 — `crypto/aes` + `crypto/hmac` + padding + IV
  generation + AEAD tagging.
- Compact-serialization JWE parser/decoder.
- Total: ~300 LOC of fragile crypto.

The cross-tooling correctness bar is byte equality with the
`@auth/core/jwt` encoder's output. A drift by one byte (e.g., using the
`base64` env value of `AUTH_SECRET` instead of the raw string) would
yield 401s on every request.

## Decision drivers

- **Cross-tooling correctness**: the verifier MUST match Auth.js's
  encoder byte for byte. Verified envelope = confirmed via the
  `@auth/core@0.34.3/src/jwt.ts` source.
- **Active maintenance**: `lestrrat-go/jwx` v2 is the de-facto Go JOSE
  library; widely used; ~700+ stars; maintained by a long-standing
  maintainer.
- **Readability**: the call site is
  `jwe.Decrypt(cookie, jwe.WithKey(jwa.DIRECT, key))` with the
  HKDF-derived key constructed externally. The intent reads at a glance.
- **No triple-bookkeeping**: `jwx` covers both the JWE parsing and the
  JWK construction; no need to find and pin two libraries.

## Options considered

1. **`github.com/lestrrat-go/jwx/v2/jwe` + `golang.org/x/crypto/hkdf`**
   — chosen. Two direct deps; ~15 LOC on top. Confirmed `jwe.WithKey`
   accepts `jwa.DIRECT` (the `"dir"` alg constant in jwx v2).
2. **`github.com/golang-jwt/jwt/v5`** — rejected. The library is
   JWS-focused; JWE support is labelled "experimental" by the upstream
   maintainers and the envelope we'd write would be more code than
   using jwx directly.
3. **Hand-rolled AEAD + HKDF** — rejected. Crypto is hard to get
   right; version skew between Go's `crypto/aes` and Auth.js's
   `jose` implementations is a real audit risk.
4. **`github.com/go-jose/go-jose/v4`** — considered. Larger API
   surface; the `jwx` library covers our two-call footprint more
   cleanly. Go-jose would also work; the deciding factor was the
   sharper maintenance signal from `lestrrat-go/jwx` v2's release
   cadence.

## Decision

Promote `github.com/lestrrat-go/jwx/v2` (current minor at acceptance
time) to a direct dependency in
`backend/database_administrator/go.mod`. Implement the verifier using:

```go
// 1. Derive the 64-byte symmetric key with HKDF-SHA256.
dk := hkdf.New(sha256.New,
    []byte(cfg.AuthSecret),                              // raw env string
    []byte(cfg.CookieName),                              // salt
    []byte("Auth.js Generated Encryption Key ("+cfg.CookieName+")"),
    64).Read(make([]byte, 64))

// 2. Build an OCT JWK and decrypt the cookie.
key, _ := jwk.FromRaw(dk)
plaintext, err := jwe.Decrypt(
    []byte(cookie),
    jwe.WithKey(jwa.DIRECT /* "dir" */, key),
)
```

The HKDF parameters (sha256, salt=cookieName, info literal, length=64)
are the **single source of truth** in
`backend/database_administrator/src/interfaces/http/auth_middleware.go`'s
`deriveKey(cfg)` function. Any change to those parameters is a
breaking change to this ADR.

## Consequences

### Positive

- ~15 LOC of code on top of the library; no crypto implemented by hand.
- Maintained upstream with no API churn anticipated.
- The fixture test in
  `backend/database_administrator/src/interfaces/http/testdata/authjs_session_token.jwe`
  asserts byte-equality between Auth.js's encoder and the Go decoder.
  This is the primary TDD evidence for R-BAM-020.

### Negative / risk

- **HKDF parameters drift** between Auth.js encoder and the Go decoder
  is the single biggest foot-gun. Mitigation: the fixture test is the
  gate; CI gates the byte equality.
- **`AUTH_SECRET` handling**: Auth.js does NOT base64-decode the secret;
  it uses the raw env string as HKDF input. The Go verifier must do
  the same. Mitigation: the unit test asserts the byte equality
  end-to-end and a documentation comment in `auth_middleware.go`
  links this ADR.
- **`jwe.WithKey(jwa.DIRECT, ...)`** — `jwa.DIRECT` is the jwx v2
  constant for the `"dir"` alg. If a future Auth.js release adds an
  alternative alg (e.g., `A256KW`), the verifier must spec the
  allowed list explicitly and reject other algs with 401.

### Future-proofing

- If Auth.js ever changes the JWE envelope (e.g., switches to A256GCM),
  the Go verifier MUST be re-tested first; the fixture test in
  `testdata/authjs_session_token.jwe` must be regenerated via
  `backend/database_administrator/scripts/regenerate_authjs_testdata.sh`
  and the diff committed.
- HKDF can rotate the salt without rotating `AUTH_SECRET` by changing
  the cookie name in a future change; this ADR does not assume
  permanence of the salt value.

## References

- `openspec/changes/cachicamas-github-login/design.md` §3 — The exact
  JWE envelope and key derivation contract.
- `openspec/changes/cachicamas-github-login/specs/backend-auth-middleware/spec.md`
  — R-BAM-010..R-BAM-100.
- `backend/database_administrator/src/interfaces/http/auth_middleware.go`
  — implementation (when PR-3 lands).
- `backend/database_administrator/src/interfaces/http/testdata/authjs_session_token.jwe`
  — fixture (when PR-3 lands).
- `backend/database_administrator/scripts/regenerate_authjs_testdata.sh`
  — fixture regenerator (when PR-3 lands).
