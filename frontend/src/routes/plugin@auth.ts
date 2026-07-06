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
 *
 * `events.signIn` persistence (cachicamas-identity-signin-callback):
 *   The previous slice (PR #29, now superseded) wired this callback
 *   to a direct Postgres client (`porsager/postgres`). That
 *   architecture is wrong — the frontend and the backend shared the
 *   same DB role + credentials, and the Vite bundler needed a stub
 *   plugin to keep `postgres` out of the client bundle.
 *
 *   This slice (see ADR 0003) replaces it with an HMAC-signed POST
 *   to the database_administrator Go service. The frontend's
 *   identity-callback-client.ts handles canonicalization + signing
 *   + dispatch. This callback just forwards the Auth.js event.
 */
import { QwikAuth$ } from "@auth/qwik";
import GitHub from "@auth/qwik/providers/github";
import { isServer } from "@builder.io/qwik/build";
import {
  postIdentityCallback,
  type SignInEvent,
} from "~/lib/identity-callback-client";

// 2026-07-06 native-auth-UI carve-out: we deliberately do NOT
// destructure `onRequest` from the library's return value. Instead we
// re-export the library's hooks (`useSession`, `useSignIn`,
// `useSignOut`) and write our own `onRequest` below. The custom
// `onRequest` skips @auth/core interception for the bare page-render
// GETs on `/auth/signin` and `/auth/signout` (no provider segment)
// so Qwik City routes them to our native pages. For every other path
// we delegate to the library's `onRequest`, which still handles the
// OAuth protocol (`/auth/callback/{provider}`, the internal POST
// from `useSignIn`/`useSignOut`, `/auth/error`, `/auth/csrf`,
// `/auth/session`).
//
// Why this design (vs. trying to disable interception inside the
// library): the library's `onRequest` is a closed function; we cannot
// pass it a "skip these paths" option. The cleanest override is to
// own the plugin's `onRequest` slot ourselves and call the library's
// `onRequest` only for paths we want it to handle.
//
// Cost: on the native pages, `useSession()` returns `null` because we
// do not populate `sharedMap.session` for those paths (the library's
// `getSessionData()` is internal). For now this is acceptable — the
// native signin/signout pages do not yet need session-aware
// branching. If that becomes a requirement, replicate `getSessionData`
// by importing `Auth` from `@auth/core` directly.
const auth = QwikAuth$((ev) => {
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
    // Custom page overrides (2026-07-06, native auth UI): the
    // signin and signout pages now render as native Qwik routes at
    // `routes/auth/signin/index.tsx` and `routes/auth/signout/index.tsx`
    // instead of the @auth/core built-in HTML (which uses a
    // prefers-color-scheme dark theme and looks off-brand against
    // the rest of cachicamas). The custom `onRequest` below (NOT the
    // library's) skips interception for `GET /auth/signin` and
    // `GET /auth/signout` so Qwik City routes them natively; this
    // `pages` block is the defensive twin — it tells Auth.js where
    // to redirect when it needs to send the user to a signin or
    // signout surface (e.g. for protected resources or error pages).
    pages: {
      signIn: "/auth/signin",
      signOut: "/auth/signout",
    },
    // The same AUTH_SECRET the Go verifier uses to decrypt the JWE
    // cookie (PR-3). Locked contract per ADR 0002 §3.1 — the byte-level
    // envelope must match `@auth/core@0.34.3` (and forwards) exactly.
    secret: ev.env.get("AUTH_SECRET"),
    // Identity persistence: forward each successful GitHub sign-in
    // to the database_administrator Go service so the
    // identity.user + identity.account rows are persisted there
    // (no longer from the Node SSR — see ADR 0003). The forwarding
    // is best-effort: a service error is logged + swallowed so a
    // successful OAuth roundtrip is never blocked by an identity
    // persistence failure. The same posture PR #29 took (now
    // superseded) — kept here because it is strictly better than
    // refusing a successful GitHub sign-in.
    events: {
      signIn: async (event: SignInEvent) => {
        try {
          await postIdentityCallback(event);
        } catch (err) {
          console.error(
            "[plugin@auth] identity signin-callback failed (best-effort; OAuth session is still valid):",
            err instanceof Error ? err.message : String(err),
          );
        }
      },
    },
  };
});

// Re-export the library's three hooks unchanged — they call
// `globalAction$`/`routeLoader$` and have no onRequest side effect.
export const { useSession, useSignIn, useSignOut } = auth;

// Custom onRequest — see the long comment block above the `QwikAuth$`
// call for the full rationale. Summary:
//   1. If the request is `GET /auth/signin` or `GET /auth/signout` with
//      no provider segment, skip @auth/core entirely so Qwik City
//      renders our native page instead of the dark default HTML.
//   2. Otherwise, delegate to the library's onRequest, which still
//      handles every Auth.js protocol path (OAuth callbacks, the
//      internal POSTs from useSignIn/useSignOut, /auth/error, /auth/csrf,
//      /auth/session).
export const onRequest = async (
  req: Parameters<typeof auth.onRequest>[0],
): Promise<void> => {
  if (!isServer) return;
  // 2026-07-06 bugfix: Qwik City auto-redirects `/auth/signin` to
  // `/auth/signin/` (trailing-slash canonicalisation) BEFORE middleware
  // runs, so the path the interceptor sees almost always has the
  // trailing slash. Comparing `path === "/auth/signin"` therefore misses
  // both shapes and falls through to @auth/core, which redirects back
  // to `/auth/signin?callbackUrl=...` — an infinite loop in the
  // browser. Normalise by stripping trailing slashes before comparing.
  // `req.url.pathname` excludes the query string, so the callbackUrl
  // param is not affected by the strip.
  const normalizedPath = req.url.pathname.replace(/\/+$/, "") || "/";
  const isGet = req.request.method === "GET";
  const isNativePageRender =
    isGet &&
    (normalizedPath === "/auth/signin" || normalizedPath === "/auth/signout");
  if (isNativePageRender) {
    // No-op: Qwik City's router will match `routes/auth/signin/index.tsx`
    // or `routes/auth/signout/index.tsx` and render our native page.
    // Trade-off documented above: useSession() returns null on these
    // pages until we wire session loading manually.
    return;
  }
  await auth.onRequest(req);
};
