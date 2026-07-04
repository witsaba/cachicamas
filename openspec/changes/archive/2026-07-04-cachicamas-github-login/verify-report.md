# Verify Report: cachicamas-github-login

> **Change**: `cachicamas-github-login`
> **Status**: SHIPPED (all 3 chained PRs merged)
> **Archived**: 2026-07-04
> **Predecessor**: `openspec/changes/cachicamas-github-login/` (now under archive/)

## Scope shipped

End-to-end GitHub OAuth login for cachicamas:

1. **Persistent identity schema** (PR #18, commit `69429c4`):
   `identity.user` + `identity.account` + `organization.owner_user_id`
   FK. Reversible goose migration. CITEXT for case-insensitive email
   uniqueness. Strict TDD discipline (`apply.tdd: true`); 35+ unit
   tests, all under `-race`.

2. **Frontend Auth.js UX** (PR #20, commit `7ea621a`): Qwik Node SSR
   integration with `@auth/qwik@0.9.2`. `routes/plugin@auth.ts` wires
   the GitHub provider. `lib/sign-in-callback.ts` implements
   auto-link-on-email-match on the `events.signIn` callback. Mocks
   simulator for tests. Two Playwright specs. 119 frontend unit
   tests, all under vitest.

3. **Backend JWE verifier middleware** (PR #22, commit `c4b89d4`):
   `IdentityFromCookie` Echo middleware that decrypts the same JWE
   the frontend issues, resolves the email claim to the
   `identity.user` + `identity.account` rows, and populates
   `c.Set("identity", *Identity)`. Demo endpoint
   `/api/v1/protected/whoami` behind the middleware. Committed
   `testdata/authjs_session_token.jwe` fixture proves the byte-level
   cross-tooling envelope contract holds end-to-end.

## Spec scenarios walked

| Capability | Scenarios | Status |
| --- | --- | --- |
| `identity-schema` | 15 scenarios (S-IS-010 → S-IS-080) | All PASS (PR #18, verified against live compose) |
| `frontend-auth` | 18 scenarios (S-FA-001 → S-FA-058) | All PASS at unit-test boundary (PR #20). E2E coverage deferred to follow-up PRs. |
| `backend-auth-middleware` | 17 scenarios (S-BAM-010 → S-BAM-110) | All PASS at unit-test boundary (PR #22). Live-compose integration test deferred to follow-up PRs. |

## Canonical spec promotions

| Capability | Promoted in commit | URL |
| --- | --- | --- |
| `identity-schema` | `e67dac4` (PR #19) | `openspec/specs/identity-schema/spec.md` |
| `frontend-auth` | `1f99796` (PR #21) | `openspec/specs/frontend-auth/spec.md` |
| `backend-auth-middleware` | this turn | `openspec/specs/backend-auth-middleware/spec.md` |

## Locked contracts (per ADR 0001 + ADR 0002)

- **Envelope**: `alg=dir, enc=A256CBC-HS512`, HKDF-SHA256 over raw
  `AUTH_SECRET` with `salt=cookieName`, `info="Auth.js Generated
  Encryption Key (cookieName)"`, `length=64`. Verified against
  `@auth/core@0.41.2/src/jwt.ts` byte-for-byte.
- **Cookie name**: `authjs.session-token` (dev HTTP) /
  `__Secure-authjs.session-token` (prod HTTPS). The HKDF salt is
  the cookie name itself, so dev and prod derive DIFFERENT keys
  even with the same `AUTH_SECRET`.
- **Strategy**: stateless JWE cookie. NO `identity.session` table.
  Auto-link on email match.

## Gates (all green across the 3 PRs)

| Gate | PR-1 | PR-2 | PR-3 |
| --- | --- | --- | --- |
| `make lint` | 0 issues | 0 issues | 0 issues |
| `make build` | OK | OK | OK (20 MB) |
| `make test` | All PASS (35+ tests) | All PASS (119 frontend tests) | All PASS (8 auth tests + 4 sub-tests) |
| `docker build` | OK | OK | OK |
| Compose validation | N/A | OK | N/A |

## 4R reviews (inline, by parent orchestrator)

The standard `4r-review` chain failed on a tool-binding issue across
all 3 PRs (every agent only received `intercom` + `contact_supervisor`,
no file-read tools). The parent orchestrator produced inline reviews:

- PR #18: `/Users/braejan/.pi/agent/scratch/4r-review-pr1.md`
- PR #20: `/Users/braejan/.pi/agent/scratch/4r-review-pr2.md`
- PR #22: `/Users/braejan/.pi/agent/scratch/4r-review-pr3.md`

All three returned **APPROVE**.

## Forward notes (deferred to follow-up PRs)

The change ships its core capability. The following items are
explicitly out-of-scope for the chained PR strategy and should land
in dedicated follow-up PRs:

1. **Three PR-2 e2e specs** (R-FA-060+): `sign-in-cookie-attrs`,
   `sign-in-denied`, `sign-out`. The Playwright specs are written
   but skipped in production mode; they need the dockerized stack
   for full exercise.
2. **Live-compose integration test for migration + identity_repo
   end-to-end** (S-BAM-020 verification at the integration layer).
3. **`LookupByProviderAccountID` method** on the `IdentityRepository`
   port. Currently the verifier uses `LookupByEmail` only; future
   multi-provider support needs resolution by
   `(provider, provider_account_id)`.
4. **mocks-github-oauth unit tests** for the simulator endpoints
   (currently only smoke-tested via curl).

## Rollback plan (per proposal §7)

Two-stage rollback:
- **Stage 1** (no data loss): revert PR #22 (Go verifier removed).
  Frontend continues to issue JWE cookies; the Go service loses
  the `/api/v1/protected/whoami` endpoint. Application users
  unaffected.
- **Stage 2** (data loss): revert PR #18 + drop the
  `identity.user`, `identity.account`, and
  `organization.owner_user_id` schema. All signed-in users lose
  their identity rows. The `Down` migration is committed and
  reversible; round-tripped in PR #1.

Stage 1 is the recommended rollback path for production incidents.
Stage 2 is reserved for cases where the entire feature is being
decommissioned.