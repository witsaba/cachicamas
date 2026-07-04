/**
 * Unit tests for the mocks-github-oauth simulator endpoints.
 *
 * Reference: `openspec/changes/cachicamas-github-login/verify-report.md`
 *   Forward notes — "mocks-github-oauth unit tests for the simulator
 *   endpoints (currently only smoke-tested via curl)".
 *
 * Why this is in `e2e/`:
 *   The simulator is a Node HTTP server. Vitest can spawn it
 *   directly on an ephemeral port and drive the four endpoints
 *   via fetch(). The spec does NOT use Playwright's browser APIs.
 *
 * Scope:
 *   - /login/oauth/authorize redirects with code=test_code + echoed state.
 *   - /login/oauth/access_token returns the test token when client_id +
 *     client_secret + code are all present.
 *   - /user returns the test user when Bearer auth is provided.
 *   - /user/emails returns the test emails when Bearer auth is provided.
 *   - /healthz returns {ok:true}.
 *   - All endpoints return 404 for unknown paths.
 *   - /user and /user/emails return 401 without Bearer auth.
 */
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { spawn, type ChildProcess } from "node:child_process";
import { setTimeout as wait } from "node:timers/promises";

const TEST_PORT = 3017; // ephemeral; mocks listens on 3016 in compose
const BASE = `http://127.0.0.1:${TEST_PORT}`;

let server: ChildProcess;

async function waitForServer(url: string, attempts = 30): Promise<void> {
  for (let i = 0; i < attempts; i += 1) {
    try {
      const r = await fetch(url);
      if (r.ok || r.status === 404) return; // server is responding
    } catch {
      // not yet listening
    }
    await wait(100);
  }
  throw new Error(`server did not start at ${url} after ${attempts} attempts`);
}

beforeAll(async () => {
  server = spawn(process.execPath, ["server.mjs"], {
    cwd: new URL("../../../scripts/mocks-github-oauth/", import.meta.url)
      .pathname,
    env: {
      ...process.env,
      MOCKS_PORT: String(TEST_PORT),
      MOCKS_HOST: "127.0.0.1",
    },
    stdio: ["ignore", "pipe", "pipe"],
  });
  await waitForServer(`${BASE}/healthz`);
});

afterAll(async () => {
  if (server && !server.killed) {
    server.kill("SIGTERM");
    // Give it a moment to clean up.
    await wait(100);
  }
});

describe("mocks-github-oauth simulator", () => {
  it("GET /healthz returns {ok:true}", async () => {
    const r = await fetch(`${BASE}/healthz`);
    expect(r.status).toBe(200);
    const body = await r.json();
    expect(body).toEqual({ ok: true });
  });

  it("GET /login/oauth/authorize redirects with code=test_code + echoed state", async () => {
    const r = await fetch(
      `${BASE}/login/oauth/authorize?client_id=foo&redirect_uri=${BASE}/cb&state=abc123`,
      {
        redirect: "manual",
      },
    );
    expect(r.status).toBe(302);
    const location = r.headers.get("location");
    expect(location).toBeTruthy();
    const u = new URL(location!);
    expect(u.searchParams.get("code")).toBe("test_code");
    expect(u.searchParams.get("state")).toBe("abc123");
  });

  it("GET /login/oauth/authorize without redirect_uri returns 400", async () => {
    const r = await fetch(`${BASE}/login/oauth/authorize`);
    expect(r.status).toBe(400);
    const body = await r.json();
    expect(body.error).toMatch(/redirect_uri/);
  });

  it("POST /login/oauth/access_token returns a token with all required fields", async () => {
    const r = await fetch(`${BASE}/login/oauth/access_token`, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        Accept: "application/json",
      },
      body: new URLSearchParams({
        client_id: "foo",
        client_secret: "bar",
        code: "test_code",
        redirect_uri: `${BASE}/cb`,
        grant_type: "authorization_code",
      }),
    });
    expect(r.status).toBe(200);
    const body = await r.json();
    expect(body.access_token).toMatch(/^test_access_token_/);
    expect(body.token_type).toBe("bearer");
    expect(typeof body.scope).toBe("string");
  });

  it("POST /login/oauth/access_token without code returns 400", async () => {
    const r = await fetch(`${BASE}/login/oauth/access_token`, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
        Accept: "application/json",
      },
      body: new URLSearchParams({
        client_id: "foo",
        client_secret: "bar",
      }),
    });
    expect(r.status).toBe(400);
    const body = await r.json();
    expect(body.error).toMatch(/missing/);
  });

  it("GET /user with Bearer returns the test user", async () => {
    const r = await fetch(`${BASE}/user`, {
      headers: { Authorization: "Bearer any-token" },
    });
    expect(r.status).toBe(200);
    const body = await r.json();
    expect(body.login).toBe("octocat");
    expect(body.email).toBe("octocat@example.com");
    expect(body.id).toBe(12345);
  });

  it("GET /user without Bearer returns 401", async () => {
    const r = await fetch(`${BASE}/user`);
    expect(r.status).toBe(401);
  });

  it("GET /user/emails with Bearer returns the primary email", async () => {
    const r = await fetch(`${BASE}/user/emails`, {
      headers: { Authorization: "Bearer any-token" },
    });
    expect(r.status).toBe(200);
    const body = await r.json();
    expect(Array.isArray(body)).toBe(true);
    const primary = body.find((e: { primary?: boolean }) => e.primary);
    expect(primary.email).toBe("octocat@example.com");
    expect(primary.verified).toBe(true);
  });

  it("GET /user/emails without Bearer returns 401", async () => {
    const r = await fetch(`${BASE}/user/emails`);
    expect(r.status).toBe(401);
  });

  it("GET /unknown returns 404", async () => {
    const r = await fetch(`${BASE}/some/nonexistent/path`);
    expect(r.status).toBe(404);
  });
});
