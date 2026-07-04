/**
 * Auth.js for Qwik wiring — the canonical GitHub OAuth roundtrip entry point.
 *
 * Reference: `openspec/changes/cachicamas-github-login/specs/frontend-auth/spec.md`
 *   R-FA-001 — the Qwik app exposes /auth/signin, /auth/callback/github, and
 *   /auth/signout via the four canonical exports (`onRequest`, `useSession`,
 *   `useSignIn`, `useSignOut`).
 *
 * Design contract: see `openspec/changes/cachicamas-github-login/design.md` §5.1
 * for the env bindings and §6.1 for the ADR accepting the pre-1.0 risk of
 * `@auth/qwik@0.9.2`.
 *
 * Why this lives at `src/routes/plugin@auth.ts`:
 *   The `@` suffix is the Auth.js for Qwik convention. Qwik City's
 *   build pipeline reads any `routes/plugin@*.ts` as a request-time
 *   plugin and wires its `onRequest` into the request lifecycle.
 *   Renaming this file would break the OAuth roundtrip at runtime.
 *
 * The config callback is intentionally a QRL (lazy) — env reads happen
 * per request via `ev.env.get(...)`, never at module load. This lets
 * the same Docker image be re-used across dev/staging/prod without a
 * rebuild when the operator rotates `AUTH_SECRET` or `AUTH_GITHUB_SECRET`.
 */
import { QwikAuth$ } from "@auth/qwik";
import GitHub from "@auth/qwik/providers/github";

export const { onRequest, useSession, useSignIn, useSignOut } = QwikAuth$(
  (ev) => {
    // GitHub OAuth URLs can be overridden via AUTH_GITHUB_BASE_URL
    // (browser-facing) and AUTH_GITHUB_API_BASE_URL (server-facing).
    // Production (unset) → https://github.com (canonical). Tests
    // (set) → the mocks-github-oauth compose service, e.g.
    //   browser:  http://localhost:3016 (host-published port)
    //   server:   http://cachicamas-mocks-github-oauth:3016 (compose DNS)
    // The two URLs are required because the BROWSER follows redirect
    // URLs from the host's perspective, while the Node SSR fetches
    // token / userinfo from the container network. AUTH_GITHUB_BASE_URL
    // is the public-facing one (used for authorization.url);
    // AUTH_GITHUB_API_BASE_URL is the server-facing one (used for
    // token + userinfo). If only AUTH_GITHUB_BASE_URL is set, the
    // server-facing URLs default to it for backward compatibility.
    const githubBaseUrl =
      ev.env.get("AUTH_GITHUB_BASE_URL") ?? "https://github.com";
    const githubApiBaseUrl =
      ev.env.get("AUTH_GITHUB_API_BASE_URL") ?? githubBaseUrl;
    return {
      providers: [
        GitHub({
          clientId: ev.env.get("AUTH_GITHUB_ID"),
          clientSecret: ev.env.get("AUTH_GITHUB_SECRET"),
          // Override the GitHub URLs only when AUTH_GITHUB_BASE_URL is
          // explicitly set. The default (no override) preserves the
          // canonical github.com behaviour.
          ...(githubBaseUrl !== "https://github.com"
            ? {
                authorization: {
                  url: `${githubBaseUrl}/login/oauth/authorize`,
                  params: { scope: "read:user user:email" },
                },
                token: `${githubApiBaseUrl}/login/oauth/access_token`,
                userinfo: {
                  url: `${githubApiBaseUrl}/user`,
                  async request({
                    tokens,
                    provider,
                  }: {
                    tokens: { access_token?: string };
                    provider: { userinfo?: { url?: string } };
                  }) {
                    const profile = (await fetch(
                      provider.userinfo?.url as string,
                      {
                        headers: {
                          Authorization: `Bearer ${tokens.access_token}`,
                        },
                      },
                    ).then((r) => r.json())) as {
                      id?: number;
                      login?: string;
                      name?: string;
                      email?: string | null;
                      avatar_url?: string;
                    };
                    // Augment with the verified primary email when
                    // /user/emails is reachable (mocks + github).
                    if (tokens.access_token) {
                      const emails = (await fetch(
                        `${githubApiBaseUrl}/user/emails`,
                        {
                          headers: {
                            Authorization: `Bearer ${tokens.access_token}`,
                          },
                        },
                      )
                        .then((r) => (r.ok ? r.json() : []))
                        .catch(() => [])) as Array<{
                        email?: string;
                        primary?: boolean;
                      }>;
                      const primary = Array.isArray(emails)
                        ? emails.find((e) => e.primary)
                        : null;
                      if (primary && !profile.email) {
                        profile.email = primary.email ?? null;
                      }
                    }
                    return profile;
                  },
                },
              }
            : {}),
        }),
      ],
      // Required for non-Vercel/Netlify/Cloudflare deploys (qwik.dev docs
      // are explicit). Without this, the Qwik Node server can refuse to
      // construct correct callback URLs in compose / on the VPS.
      trustHost: true,
      // The same AUTH_SECRET the Go verifier uses to decrypt the JWE
      // cookie (PR-3). Locked contract per ADR 0002 §3.1 — the byte-level
      // envelope must match `@auth/core@0.34.3` (and forwards) exactly.
      secret: ev.env.get("AUTH_SECRET"),
    };
  },
);
