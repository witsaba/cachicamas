/**
 * mocks-github-oauth — minimal GitHub OAuth + /user + /user/emails
 * simulator for the cachicamas-github-login PR-2 Playwright suite.
 *
 * Reference: `openspec/changes/cachicamas-github-login/tasks.md`
 *   T2.20 [TEST-INFRA] GitHub OAuth simulator compose service.
 *
 * Endpoints (mimics the GitHub OAuth web flow):
 *   GET  /login/oauth/authorize?client_id=...&redirect_uri=...&state=...&scope=...
 *        → 302 redirect to redirect_uri?code=test_code&state=<echoed>
 *   POST /login/oauth/access_token  (Accept: application/json)
 *        → 200 { access_token, token_type, scope }
 *   GET  /user                       (Authorization: Bearer <token>)
 *        → 200 { id, login, name, email, avatar_url }
 *   GET  /user/emails                (Authorization: Bearer <token>)
 *        → 200 [{ email, primary, verified }]
 *
 * Why this is a separate compose service (not a vitest mock):
 *   The OAuth roundtrip happens in the BROWSER, not the test
 *   process. The browser fetches our redirect endpoint over HTTP,
 *   so the simulator must be a real network service. Hosting it
 *   on the same compose network as the Qwik Node SSR lets the
 *   Auth.js callback URL point at `http://mocks-github-oauth:3016`
 *   while the browser-facing redirect points at
 *   `http://localhost:3016` (mapped via the test's `E2E_BASE_URL`).
 *
 * Strict minimalism:
 *   - No state. The OAuth code is always "test_code" and the
 *     access_token is always "test_access_token_<random>". That's
 *     enough for the e2e spec to assert the roundtrip completes
 *     and the identity.user / identity.account rows land in Postgres.
 *   - No auth on /login/oauth/access_token beyond what GitHub
 *     actually requires (client_id + client_secret POST body).
 *     Auth.js sends both.
 *
 * Reuse scope:
 *   This script is INTENTIONALLY generic — it could serve any
 *   Auth.js e2e suite against any OAuth provider by editing the
 *   URLs. For cachicamas we hard-code the GitHub endpoints.
 */

import { createServer } from "node:http";
import { URL } from "node:url";

const PORT = parseInt(process.env.MOCKS_PORT ?? "3016", 10);
const HOST = process.env.MOCKS_HOST ?? "0.0.0.0";

// Test identity. Auth.js' GitHub provider reads id (numeric),
// login (string), name, email, and avatar_url from /user.
const TEST_USER = {
  id: 12345,
  login: "octocat",
  name: "Octocat",
  email: "octocat@example.com",
  avatar_url: "https://avatars.githubusercontent.com/u/583231?v=4",
};

const TEST_EMAILS = [
  { email: "octocat@example.com", primary: true, verified: true },
];

const json = (res, status, body) => {
  res.writeHead(status, {
    "Content-Type": "application/json; charset=utf-8",
    "Cache-Control": "no-store",
  });
  res.end(JSON.stringify(body));
};

const redirect = (res, location) => {
  res.writeHead(302, {
    Location: location,
    "Cache-Control": "no-store",
  });
  res.end();
};

const readBody = (req) =>
  new Promise((resolve) => {
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => resolve(Buffer.concat(chunks).toString("utf8")));
    req.on("error", () => resolve(""));
  });

const server = createServer(async (req, res) => {
  const url = new URL(req.url ?? "/", `http://${req.headers.host}`);

  // 1. /login/oauth/authorize — GitHub OAuth consent screen.
  //    The simulator short-circuits the consent by immediately
  //    redirecting back with a fixed `code=test_code` and the same
  //    `state` the client passed.
  if (url.pathname === "/login/oauth/authorize" && req.method === "GET") {
    const redirectUri = url.searchParams.get("redirect_uri");
    const state = url.searchParams.get("state");
    if (!redirectUri) {
      return json(res, 400, { error: "missing redirect_uri" });
    }
    const target = new URL(redirectUri);
    target.searchParams.set("code", "test_code");
    if (state) target.searchParams.set("state", state);
    return redirect(res, target.toString());
  }

  // 2. /login/oauth/access_token — exchanges the code for a token.
  //    Auth.js POSTs application/x-www-form-urlencoded with
  //    client_id, client_secret, code, redirect_uri, grant_type.
  if (url.pathname === "/login/oauth/access_token" && req.method === "POST") {
    const body = await readBody(req);
    const params = new URLSearchParams(body);
    if (
      !params.get("client_id") ||
      !params.get("client_secret") ||
      !params.get("code")
    ) {
      return json(res, 400, {
        error: "missing required parameter",
      });
    }
    // Auth.js calls with Accept: application/json; respect it.
    return json(res, 200, {
      access_token: `test_access_token_${Date.now()}`,
      token_type: "bearer",
      scope: params.get("scope") ?? "read:user,user:email",
    });
  }

  // 3. /user — the userinfo endpoint. Auth.js' GitHub provider calls
  //    this with `Authorization: Bearer <access_token>`. We accept
  //    any bearer (test mode) and return the canned test user.
  if (url.pathname === "/user" && req.method === "GET") {
    const auth = req.headers.authorization;
    if (!auth?.startsWith("Bearer ")) {
      return json(res, 401, { message: "Bad credentials" });
    }
    return json(res, 200, TEST_USER);
  }

  // 4. /user/emails — used by the userinfo.request override in
  //    plugin@auth.ts to fetch the primary email. GitHub hides
  //    primary email on /user unless scope=user:email is granted.
  if (url.pathname === "/user/emails" && req.method === "GET") {
    const auth = req.headers.authorization;
    if (!auth?.startsWith("Bearer ")) {
      return json(res, 401, { message: "Bad credentials" });
    }
    return json(res, 200, TEST_EMAILS);
  }

  // 5. /healthz — used by the compose healthcheck.
  if (url.pathname === "/healthz" && req.method === "GET") {
    return json(res, 200, { ok: true });
  }

  // Anything else is a 404.
  return json(res, 404, {
    error: "not found",
    path: url.pathname,
  });
});

server.listen(PORT, HOST, () => {
  // eslint-disable-next-line no-console
  console.log(
    `[mocks-github-oauth] listening on http://${HOST}:${PORT} ` +
      `(test user: ${TEST_USER.login} <${TEST_USER.email}>)`,
  );
});