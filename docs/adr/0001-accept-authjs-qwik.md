# ADR 0001: Accept `@auth/qwik@0.9.2` as a direct frontend prod dependency

> Status: **Accepted** (2026-07-03, change `cachicamas-github-login`)
> Author: cachicamas SDD pipeline (parent orchestrator)
> Replaces: n/a

## Context

The cachicamas frontend needs GitHub OAuth login. The OAuth roundtrip has to
happen in the Qwik app (the only browser-facing surface; there is no
separate API frontend, and the Go service is reached over the same-origin
`/api/*` reverse proxy embedded in `frontend/src/entry.express.tsx`).

The pre-existing alternative for Qwik was `@builder.io/qwik-auth`, which
qwik.dev now lists as deprecated in favor of the upstream `Auth.js`
rebrand. There are no other OAuth-ecosystem-native Qwik integrations of
maturity; rolling our own would be ~600 LOC of OAuth client + CSRF + state

+ PKCE + JWE/JWT session encoding.

What Auth.js brings out of the box:

+ OIDC/OAuth 2.0 roundtrip with state validation (CSRF defense).
+ PKCE parameter generation when the provider accepts it.
+ Cookie-based JWE session storage; stateless by default.
+ `useSession` / `useSignIn` / `useSignOut` as Qwik primitives that hook
  into the existing `routes/plugin@auth.ts` lifecycle.
+ A single config file that becomes the place to add Google, Microsoft,
  or generic OIDC providers later.

What Auth.js costs:

+ The Qwik adapter is pre-1.0 (`@auth/qwik` is at `0.9.2` as of
  2025-10-26; qwik.dev is explicit: "could have bugs"). The package is
  small (~87KB unpacked, 430 files, ~757 weekly npm downloads) but
  version-skews faster than a v1+ library.
+ Cross-tooling portability means the JWE envelope (alg=dir, enc=A256CBC-HS512)
  becomes a load-bearing contract between the Qwik Node SSR encoder and
  any other server that wants to verify the cookie (handled in ADR 0002).

## Decision drivers

+ **Velocity**: shipping the auth UX is on the critical path; we cannot
  rebuild OAuth-client machinery in Qwik per provider.
+ **Forward compatibility**: every future OAuth provider reuses the same
  Auth.js core; no new dep per provider.
+ **Security baseline**: Auth.js handles double-submit CSRF cookie, the
  state parameter, and PKCE by default. Hand-rolled alternatives need
  three different CSRF strategies evaluated.
+ **Audit cost**: the alternative (hand-rolled) would be ~600 LOC of
  under-instrumented crypto, easy to drift; Auth.js is well-tested
  upstream.

## Options considered

1. **`@auth/qwik`** — chosen. Risk: pre-1.0.
2. **`@builder.io/qwik-auth`** — rejected. Deprecated upstream.
3. **Hand-rolled OAuth client** — rejected. ~600 LOC; three different
   CSRF strategies to choose from; hard to audit; no upgrade path for
   new providers; no migration story if we later want SSO.
4. **Pass-through to Go for OAuth** — rejected at the proposal gate.
   The Proposal lock (Option B) chose to keep the roundtrip in Qwik for
   UX consistency and accept a Go JWE verifier instead. This ADR
   records that the FRONTEND portion of Option B uses `@auth/qwik`
   specifically.

## Decision

Accept `@auth/qwik@0.9.2` as a direct production dependency. Pin the
version **exactly** (no caret) in `frontend/package.json`. The exact pin
is `0.9.2` until Auth.js v1 ships; the upgrade path is documented below.

The installation runs `pnpm add @auth/qwik@0.9.2` (or the equivalent
`pnpm run qwik add auth`); both achieve the same end state.

## Consequences

### Positive

+ One canonical OAuth client instead of N hand-rolled providers.
+ Stateless cookie sessions (no `identity.session` table) by default —
  matches the proposal's "no DB-backed sessions this slice" decision.
+ Qwik primitives (`useSession`, etc.) integrate cleanly with
  `routeLoader$` and `routeAction$` without bespoke wrappers.

### Negative / risk

+ **Pre-1.0 dependency**: a breaking change could land in a minor
  release. Mitigation: pin exact version + Playwright regression
  guard (`e2e/github-sign-in.spec.ts`) covers the round-trip.
+ **No upstream SLO**: `@auth/qwik` has 757 weekly downloads; not a
  bus-factor-of-one but small enough that we should track the upstream.
  Mitigation: subscribe to the `@auth/qwik` GitHub release feed; the
  ADR re-review trigger is "any new 0.9.x release within the next
  month".
+ **JWE envelope contract**: the format (`alg=dir`, `enc=A256CBC-HS512`,
  HKDF-SHA256 with salt=cookieName, derivedKey length=64) is the
  load-bearing interop point with the Go verifier. Cross-tooling
  byte-equality is asserted by the `authjs_session_token.jwe` fixture
  in `backend/database_administrator/src/interfaces/http/testdata/`
  (R-BAM-020, enforced by ADR 0002's companion contract).

### Upgrade path (when Auth.js v1 ships)

1. `pnpm add @auth/qwik@^1.0.0`
2. Re-run `pnpm test:ci` and `pnpm test:e2e` against the dev compose +
   `mocks/github-oauth` simulator service.
3. Re-run the cross-tooling fixture roundtrip; confirm the Go verifier
   still parses v1 cookies (the JWE envelope is stable across major
   versions in <jose@5.x> — the version Auth.js depends on).
4. Update the pinned version in `frontend/package.json`.

If Auth.js v1 changes the JWE envelope, the Go verifier must be
re-tested first (and ADR 0002 amended).

## Addendum (2026-08-01, `frontend-vuln-check`)

`@auth/core` — the peer dependency `@auth/qwik@0.9.2` pins as a regular
(non-peer) dependency — was overridden from `0.41.2` to exact `0.41.3`
via `overrides` in `frontend/pnpm-workspace.yaml`, closing
GHSA-7rqj-j65f-68wh (a homoglyph email-normalization auth bypass) on
the login path. This addendum does **not** change the decision above:
the `@auth/qwik@0.9.2` pin itself is untouched, `0.41.2 → 0.41.3` is a
patch-only security fix, and `pnpm why @auth/core` confirms a single
tree-wide resolution (no second, vulnerable copy remains under
`@auth/qwik`). See `frontend/README.md` § "Vulnerability scanning" →
"Remediation history" for verification detail.
