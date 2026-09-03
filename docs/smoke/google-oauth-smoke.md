# Google OAuth end-to-end smoke test (dev compose)

> Author: `cachicamas-google-auth-bootstrap` PR-4 (`docs(adr): add ADR 0011-native-google-oauth`).
> Status: **manual smoke, not automated**. Recorded as the acceptance smoke
> for [ADR 0011](../adr/0011-native-google-oauth.md) and the 4-PR chain
> (PR-1 schema → PR-2 backend → PR-3 frontend → PR-4 config + ADR + this smoke).
>
> **CI limitation**: this smoke requires a real Google OAuth client and a
> human-driven browser round-trip. CI cannot exercise the OAuth consent
> screen or the `state` cookie round-trip. The commands below are written
> for a developer running `docker compose up` on a workstation. A CI gate
> is out of scope for this slice.

---

## Pre-requisites

1. **Dev compose stack running**:
   ```bash
   docker compose up -d --build
   # Wait for postgres + database_administrator + frontend to be healthy
   docker compose ps
   ```

2. **Database migrations applied**:
   ```bash
   cd backend/database_administrator
   make migrate-up   # PR-1's 20260903120000_google_auth.sql lands auth.users / auth.organizations / auth.login_audits
   ```

3. **Google Cloud Console OAuth client configured**:
   - Create a project at <https://console.cloud.google.com/>.
   - Enable the **Google Identity** (or legacy **Google+ API**) service.
   - Create an OAuth client ID of type **Web application**.
   - Add `http://localhost:5173/auth/google/callback` to **Authorized
     redirect URIs**.
   - Copy the client ID and client secret into `.env` (see step 4).

4. **`.env` populated** (use `.env.example` as a template):
   ```bash
   cp .env.example .env
   # Set the existing required vars (POSTGRES_PASSWORD, AUTH_SECRET,
   # IDENTITY_CALLBACK_SECRET, CACHICAMAS_CHAT_PROVIDER_API_KEY) per the
   # existing comment blocks. Then add the PR-4 additions:
   AUTH_GOOGLE_ID=<your-client-id>
   AUTH_GOOGLE_SECRET=<your-client-secret>
   AUTH_COOKIE_SECRET=$(openssl rand -base64 32)
   AUTH_INTERNAL_SECRET=$(openssl rand -base64 32)
   GEOIP_DB_PATH=                        # leave empty to disable GeoIP silently
   PUBLIC_BASE_URL=http://localhost:5173
   ```

5. **Optional GeoIP** (skip if `GEOIP_DB_PATH` is empty):
   ```bash
   mkdir -p ./geoip-data
   # Download GeoLite2-City.mmdb from https://www.maxmind.com/en/geolite2/thank-you
   # (free account required) and drop it into ./geoip-data/.
   # Then uncomment the volume mount block in docker-compose.yaml under
   # the database_administrator service (see PR-4 commit e00c0dc5) and
   # restart:
   docker compose up -d database_administrator
   ```

---

## Manual flow

1. **Open the landing** at <http://localhost:5173>. The marketing
   landing renders without any auth gate (regression-locked by
   `routes/index.spec.tsx`; no `href="/auth/"` in the rendered HTML).

2. **Click "Sign in with Google"** (or visit
   <http://localhost:5173/auth/google/login> directly). Expect a
   `302` to `https://accounts.google.com/o/oauth2/v2/auth?…&state=<32B
   base64url>` with the `cachicamas_oauth_state` cookie set
   (`HttpOnly; SameSite=Lax; Path=/; Max-Age=600`). The state query
   parameter SHALL equal the cookie value (CSRF defence, locked by
   spec R-FE-001 / S-FE-001).

3. **Complete Google OAuth consent**. Pick a Google account whose email
   you control. Grant the requested scopes (`openid email profile`).

4. **Expect a 302 to `/auth/google/callback?code=…&state=…`**, then a
   302 to `/home` with the `cachicamas_session` cookie set
   (`HttpOnly; Secure (in production); SameSite=Lax; Path=/;
   Max-Age=604800`).

5. **`/home` shows the user email + name**. The page renders
   `Hola, {name}` (data-testid=`home-greeting`), the email
   (data-testid=`home-email`), the organization name, an "En
   construcción" notice, and a "Cerrar sesión" form
   (data-testid=`home-logout`).

6. **Click logout** (POST to `/auth/logout`). Expect a
   `Set-Cookie: cachicamas_session=; Max-Age=0` and a `302` to `/`.

7. **Visit `/home` again**. The `(app)` layout guard (R-FE-004) sees
   no valid session cookie and `302`s to `/auth/google/login`.
   This is the locked R-FE-004 / S-FE-030 invariant.

---

## Verification commands

After the flow above, run these `psql` queries against the dev
Postgres to confirm the database state matches the spec:

```bash
# 1. The authenticated user exists.
psql "postgres://queen:$QUEEN_PASSWORD@localhost:5432/cachicamas_pg?sslmode=disable" \
  -c "SELECT id, email, status, created_at, last_login_at \
        FROM auth.users \
       WHERE email = '<your-email>'"
# Expect: 1 row, status='active', created_at == last_login_at,
# google_sub non-null.

# 2. The user's organization exists (1:1 with the user per D3).
psql "postgres://queen:$QUEEN_PASSWORD@localhost:5432/cachicamas_pg?sslmode=disable" \
  -c "SELECT id, owner_id, name, slug \
        FROM auth.organizations \
       WHERE owner_id = (SELECT id FROM auth.users WHERE email = '<your-email>')"
# Expect: 1 row.

# 3. The login was audited.
psql "postgres://queen:$QUEEN_PASSWORD@localhost:5432/cachicamas_pg?sslmode=disable" \
  -c "SELECT id, email_attempted, provider, provider_subject, success, \
             ip_address, country_code, city, login_at \
        FROM auth.login_audits \
       ORDER BY login_at DESC LIMIT 5"
# Expect: at least 1 successful row (success=true, provider='google',
# provider_subject non-null, login_at recent). country_code and city
# are populated when GEOIP_DB_PATH is set and the IP is in the DB;
# NULL otherwise (best-effort).
```

---

## Optional: GeoIP-enabled variant

If `GEOIP_DB_PATH` is set (see pre-requisites step 5):

```bash
# Confirm the country/city columns are populated when the IP is in
# the MaxMind DB:
psql "postgres://queen:$QUEEN_PASSWORD@localhost:5432/cachicamas_pg?sslmode=disable" \
  -c "SELECT country_code, city, login_at \
        FROM auth.login_audits \
       WHERE success = true \
       ORDER BY login_at DESC LIMIT 1"
# Expect: country_code='US' (or whatever the dev IP resolves to),
# city populated. If columns are NULL, the DB was loaded but the IP
# was not in the DB (best-effort; still success).

# Confirm the service logged "geoip enabled" once at startup
# (vs "geoip disabled (no DB at GEOIP_DB_PATH)" when the path is empty):
docker compose logs database_administrator 2>&1 | grep -i geoip
```

---

## Failure-mode checklist

If the smoke fails at any step, work the list below:

| Symptom | Likely cause | Remediation |
|---|---|---|
| `/auth/google/login` 500s immediately | `AUTH_GOOGLE_ID` or `AUTH_GOOGLE_SECRET` unset on the frontend | Check `docker compose logs frontend`; set both vars in `.env` and `docker compose up -d frontend` |
| Google returns `redirect_uri_mismatch` | Google Cloud Console Authorized redirect URIs does not include exactly `http://localhost:5173/auth/google/callback` (the value of `${PUBLIC_AUTH_REDIRECT_URI:-${PUBLIC_BASE_URL:-http://localhost:5173}/auth/google/callback}` with `PUBLIC_BASE_URL` unset) | Add the exact URI to the OAuth client's authorized redirect URIs |
| Callback redirects to `/auth/error?reason=invalid_state` | The `cachicamas_oauth_state` cookie was not preserved across the Google redirect (Lax is correct; check that the browser is not blocking third-party cookies) | Verify `SameSite=Lax` on the cookie; if behind a corporate proxy that strips cookies, set `Secure=true` and use HTTPS |
| Callback redirects to `/auth/error?reason=token_exchange_failed` | `AUTH_GOOGLE_SECRET` mismatch with the Google Cloud Console value, OR Google returned a non-2xx from `/token` for another reason (rate limit, revoked client) | Confirm the secret matches the Google Cloud Console value; check `docker compose logs frontend` for the upstream error |
| Callback redirects to `/auth/error?reason=userinfo_failed` | Google's `/userinfo` returned non-2xx | Check `docker compose logs frontend`; verify the access token is sent as `Authorization: Bearer …` |
| Callback redirects to `/auth/error?reason=internal_error` | The backend's `/internal/auth/bootstrap` returned 5xx (DB write failed) OR `AUTH_INTERNAL_SECRET` mismatch between frontend and backend | Confirm `AUTH_INTERNAL_SECRET` matches in both `.env` values; check `docker compose logs database_administrator` for the DB-side error |
| `/home` 302s back to `/auth/google/login` immediately after success | The session cookie was not preserved (SameSite=Lax is correct; Secure=true required for cross-site) OR `AUTH_COOKIE_SECRET` mismatch between frontend (sign) and backend (verify, if you ever verify in Go) | Confirm the browser shows the cookie; confirm `AUTH_COOKIE_SECRET` matches on both sides |
| `/home` shows `user_not_found` or `fetch_failed` | The backend's `/internal/me/:user_id` returned non-2xx (DB issue or stale user id) | Check `docker compose logs database_administrator`; verify the user row exists with `psql` (verification command 1 above) |
| `auth.login_audits` row has `country_code=NULL, city=NULL` when GeoIP was expected | `GEOIP_DB_PATH` empty OR DB not loaded OR IP not in the DB | Check `docker compose logs database_administrator` for the GeoIP status line; verify the DB file exists at the mount path |

---

## CI status

**The smoke is NOT executed in CI.** Reasons:

- Requires a real Google OAuth client (secrets + cloud configuration).
- Requires a human-driven browser round-trip (the consent screen is
  not scriptable in CI without violating Google's automation policy).
- Requires the `database_administrator` + `frontend` services running
  with a Postgres instance, which CI's ephemeral runners do not
  provision for this slice.

The smoke is documented here for a developer to run on a workstation
after `docker compose up` succeeds. CI confidence comes from:

- **PR-1**: 12/12 migration tests PASS (auth.* schema present,
  identity.* dropped, FK rewritten).
- **PR-2**: 11/11 auth integration tests PASS, 43/43 application /
  domain tests PASS (the wire format from `vi.fn()` fetch mocks in
  PR-3's tests is the same wire format this smoke exercises).
- **PR-3**: 296/296 frontend tests PASS, 0 lint errors, 0 type errors
  (the `vi.fn()` mocks fully exercise the wire format that this smoke
  exercises end-to-end).
- **PR-4**: `docker compose config` resolves cleanly with the PR-4 env
  vars populated (see `chore(docker): wire Google OAuth env vars to
  frontend service` and `…to backend service` commits).

The gap between CI confidence and this smoke is the OAuth round-trip
itself (browser → Google consent → callback) and the real DB row
writes (which CI's pgx integration tests already cover in PR-2).

---

## See also

- [ADR 0011 — Native Google OAuth 2.0 + HMAC session cookies](../adr/0011-native-google-oauth.md)
- [Spec (Engram obs 4222, §4–§8)](../../openspec/changes/cachicamas-google-auth-bootstrap/spec.md) — scenarios S-FE-001..S-FE-014, S-NFR-001..S-NFR-010
- [Design (Engram obs 4223, §1.1 OAuth round-trip sequence diagram)](../../openspec/changes/cachicamas-google-auth-bootstrap/design.md)
- [PR-1 migration runner test (`TestRunner_Up_AuthTablesExist`)](../../backend/database_administrator/migration/runner_test.go) — proves the schema is in place
- [PR-2 backend handler test (`TestAuthHandler_Bootstrap_*`)](../../backend/database_administrator/interfaces/http/auth_handler_test.go) — proves the `/internal/auth/bootstrap` wire format
- [PR-3 frontend test (`routes/auth/google/callback/index.test.ts`)](../../frontend/src/routes/auth/google/callback/index.test.ts) — proves the callback handler wire format
