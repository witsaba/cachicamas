/**
 * Auth.js `events.signIn` callback — persists the GitHub identity to
 * the local `identity.user` + `identity.account` tables.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-010..R-FA-030 — auto-link on email match; one `identity.user`
 *   per email; multiple `identity.account` rows allowed for the same
 *   user (each tied to a different OAuth provider or provider_account_id).
 *   ON DELETE CASCADE handles account→user cleanup (forward note: see
 *   the inline 4R review R1 risk note).
 *
 * Design: see `openspec/changes/cachicamas-github-login/design.md` §2.1
 * for the first-time GitHub sign-in sequence diagram and §4.1 for the
 * file's location in the project layout.
 *
 * Why this lives in the frontend:
 *   Auth.js' `events.signIn` callback runs synchronously inside the
 *   Qwik request lifecycle on the Node SSR side, BEFORE the JWT is
 *   minted. Hitting the Go service here would add 5–15 ms latency to
 *   every sign-in. The frontend already runs Postgres-using libs in
 *   SSR mode for /api/* reverse-proxy traffic, so adding `postgres`
 *   as a peer is consistent with that posture.
 *
 * What this DOES NOT do (out of scope):
 *   - Issue tokens (Auth.js does that).
 *   - Refresh tokens (Auth.js handles via the OAuth provider).
 *   - Audit logging of the sign-in (forward slice; not required for
 *     the auto-link correctness assertion).
 *
 * Strict TDD:
 *   - T2.5 (this file is the RED): skeleton that throws "not implemented".
 *   - T2.6 (GREEN): real UPSERT logic.
 *   - T2.7 (TRIANGULATE): same-email-different-account_id negative case.
 */
export interface SignInUser {
  /** GitHub user id (opaque provider-side). */
  id?: string;
  name?: string | null;
  email?: string | null;
  image?: string | null;
}

export interface SignInAccount {
  /** OAuth provider; for the GitHub-only slice this is always "github". */
  provider: string;
  /** GitHub-side opaque numeric account id as a string. */
  providerAccountId: string;
  access_token?: string | null;
  refresh_token?: string | null;
  expires_at?: number | null;
  token_type?: string | null;
  scope?: string | null;
}

export interface SignInEvent {
  user: SignInUser;
  account: SignInAccount;
  /** True iff Auth.js considered this a first-time sign-in for the OAuth session. */
  isNewUser?: boolean;
}

/**
 * Minimal SQL interface — a tagged-template function. The runtime
 * `postgres` package (`porsager/postgres`) exposes exactly this
 * shape, so tests can use any function-callable mock without
 * pulling in the real driver.
 *
 * Methods on the SqlClient (the only ones we need):
 *   sql(strings, ...values): Promise<unknown[]>
 */
export type SqlClient = (
  strings: TemplateStringsArray,
  ...values: unknown[]
) => Promise<unknown[]>;

/**
 * GREEN implementation (T2.6).
 *
 * Flow:
 *   1. If the GitHub profile has no email (privacy-protected users,
 *      email scope denied, etc.), return `false` to deny sign-in.
 *      Auth.js redirects to `/auth/signin?error=AccessDenied`.
 *   2. SELECT identity.user by case-insensitive email. If a row
 *      exists, reuse it (T2.7 triangulation: same email, different
 *      providerAccountId). If not, INSERT a new row and use the
 *      returned id.
 *   3. INSERT INTO identity.account with the resolved user_id.
 *      ON CONFLICT (provider, provider_account_id) DO NOTHING — the
 *      identity_schema spec requires uniqueness on that pair and the
 *      `account_provider_provider_account_id_key` constraint exists
 *      (see PR-1 migration `20260703120000_github_login.sql`).
 *
 * The function does NOT issue UPDATE on identity.user — see the
 * "preserves the existing identity.user fields" test in the spec
 * suite. We never overwrite stored name/image on a re-link; the
 * application-side profile page is the only place that should mutate
 * those columns.
 */
export async function handleSignIn(
  sql: SqlClient,
  event: SignInEvent,
): Promise<boolean> {
  const email = event.user.email;
  if (!email || email.trim() === "") {
    // Privacy-protected GitHub users (no public email). Auto-link
    // requires the email column to be the canonical identifier, so
    // we deny sign-in and let Auth.js surface the error.
    return false;
  }
  const trimmedEmail = email.trim();

  // 1. SELECT identity.user by case-insensitive email.
  const userRows = (await sql`
    SELECT id, email, name, image_url
      FROM identity.user
     WHERE lower(email) = lower(${trimmedEmail})
     LIMIT 1
  `) as Array<{ id: string; email: string; name: string; image_url: string }>;

  let userId: string;
  if (userRows.length > 0) {
    userId = userRows[0].id;
  } else {
    // 2. INSERT identity.user (new row).
    const inserted = (await sql`
      INSERT INTO identity.user (email, name, image_url)
      VALUES (${trimmedEmail}, ${event.user.name ?? ""}, ${event.user.image ?? ""})
      RETURNING id
    `) as Array<{ id: string }>;
    userId = inserted[0].id;
  }

  // 3. UPSERT identity.account. ON CONFLICT DO NOTHING means a re-link
  //    with the same (provider, provider_account_id) is a no-op for
  //    the row, but we still return true (sign-in allowed).
  await sql`
    INSERT INTO identity.account (
      user_id, provider, provider_account_id,
      access_token, refresh_token, expires_at, token_type, scope
    ) VALUES (
      ${userId},
      ${event.account.provider},
      ${event.account.providerAccountId},
      ${event.account.access_token ?? null},
      ${event.account.refresh_token ?? null},
      ${event.account.expires_at ?? null},
      ${event.account.token_type ?? null},
      ${event.account.scope ?? null}
    )
    ON CONFLICT (provider, provider_account_id) DO NOTHING
  `;

  return true;
}