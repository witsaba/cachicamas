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
  (ev) => ({
    providers: [
      GitHub({
        clientId: ev.env.get("AUTH_GITHUB_ID"),
        clientSecret: ev.env.get("AUTH_GITHUB_SECRET"),
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
  }),
);
