/**
 * Test for `lib/identity-callback-client.ts` — the Qwik Node SSR client
 * that POSTs the Auth.js `events.signIn` payload to the database_administrator
 * `POST /api/v1/identity/signin-callback` endpoint with HMAC-SHA256 auth.
 *
 * Reference: `docs/adr/0003-add-identity-callback-hmac.md` (this slice).
 *
 * Test strategy:
 *   - Cross-tooling known vector for the canonical-JSON algorithm.
 *     The Go side (backend/database_administrator/src/interfaces/http/identity_handler_test.go
 *     TestCanonicalJSON_KnownVector) asserts the SAME byte sequence; if
 *     either side drifts, the other test fails.
 *   - HMAC signature: produce the signature with `node:crypto` inside
 *     the test, capture the client's request, and verify all three
 *     headers + the body.
 *   - Error paths: env unset, SSR-only guard, non-204 response, fetch
 *     failure.
 *   - Test isolation: every test that touches `process.env` uses
 *     beforeEach/afterEach to capture + restore.
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("lib/identity-callback-client — canonical JSON oracle", () => {
  // Re-import inside each test that needs a fresh module (per-test reset
  // ensures the env-var capture is clean).
  let canonicalizeJSON: (v: unknown) => string;

  beforeEach(async () => {
    const mod = await import("./identity-callback-client");
    canonicalizeJSON = mod.canonicalizeJSON;
  });

  it("canonicalizes a known vector byte-for-byte (cross-tooling oracle)", () => {
    const input = {
      user: {
        id: "12345",
        email: "octocat@example.com",
        name: "Octocat",
        image: null,
      },
      account: {
        provider: "github",
        providerAccountId: "12345",
        accessToken: "gho_test",
        refreshToken: null,
        expiresAt: null,
        tokenType: "bearer",
        scope: "read:user user:email",
      },
    };
    // Sort order: keys lexicographic (NOT lowercased); "provider" sorts
    // BEFORE "providerAccountId" because the shorter string is less.
    const want =
      '{"account":{"accessToken":"gho_test","expiresAt":null,"provider":"github","providerAccountId":"12345","refreshToken":null,"scope":"read:user user:email","tokenType":"bearer"},"user":{"email":"octocat@example.com","id":"12345","image":null,"name":"Octocat"}}';
    expect(canonicalizeJSON(input)).toBe(want);
  });

  it("canonicalizes a nested object with the same algorithm recursively", () => {
    const input = {
      z: { y: 1, a: 2 },
      a: { z: 1, a: 2 },
    };
    // top-level keys sorted: "a" < "z"
    // nested keys sorted the same way
    const want = '{"a":{"a":2,"z":1},"z":{"a":2,"y":1}}';
    expect(canonicalizeJSON(input)).toBe(want);
  });

  it("emits null for null values, not the string 'null'", () => {
    const input = { a: null, b: "x" };
    expect(canonicalizeJSON(input)).toBe('{"a":null,"b":"x"}');
  });

  it("escapes strings per RFC 8259 (quotes, backslashes, control chars)", () => {
    const input = { a: 'he said "hi"\nworld' };
    expect(canonicalizeJSON(input)).toBe('{"a":"he said \\"hi\\"\\nworld"}');
  });
});

describe("lib/identity-callback-client — postIdentityCallback", () => {
  const originalEnv = { ...process.env };
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(async () => {
    // Reset module so env-var captures happen fresh per test.
    vi.resetModules();
    process.env = { ...originalEnv };
    // Default fetch mock — returns 204. Tests override per case.
    fetchMock = vi.fn().mockResolvedValue({
      status: 204,
      ok: true,
      text: async () => "",
    });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (globalThis as any).fetch = fetchMock;
  });

  afterEach(() => {
    process.env = originalEnv;
    vi.restoreAllMocks();
  });

  it("throws when IDENTITY_CALLBACK_SECRET is unset", async () => {
    delete process.env.IDENTITY_CALLBACK_SECRET;
    delete process.env.SERVER_API_BASE_URL;
    delete process.env.ORIGIN;
    delete process.env.PUBLIC_API_BASE_URL;
    process.env.IMPORT_META_ENV_SSR = "true";

    const mod = await import("./identity-callback-client");
    await expect(
      mod.postIdentityCallback({
        user: { id: "1", email: "x@y.com", name: null, image: null },
        account: {
          provider: "github",
          providerAccountId: "1",
          access_token: null,
          refresh_token: null,
          expires_at: null,
          token_type: null,
          scope: null,
        },
      }),
    ).rejects.toThrow(/IDENTITY_CALLBACK_SECRET/);
  });

  it("throws when both SERVER_API_BASE_URL and ORIGIN/PUBLIC_API_BASE_URL are unset", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    delete process.env.SERVER_API_BASE_URL;
    delete process.env.ORIGIN;
    delete process.env.PUBLIC_API_BASE_URL;

    const mod = await import("./identity-callback-client");
    await expect(
      mod.postIdentityCallback({
        user: { id: "1", email: "x@y.com", name: null, image: null },
        account: {
          provider: "github",
          providerAccountId: "1",
          access_token: null,
          refresh_token: null,
          expires_at: null,
          token_type: null,
          scope: null,
        },
      }),
    ).rejects.toThrow(/(SERVER_API_BASE_URL|ORIGIN|PUBLIC_API_BASE_URL)/);
  });

  it("uses SERVER_API_BASE_URL when set (compose direct-call path)", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://database_administrator:8080";

    const mod = await import("./identity-callback-client");
    await mod.postIdentityCallback({
      user: { id: "1", email: "x@y.com", name: "x", image: null },
      account: {
        provider: "github",
        providerAccountId: "1",
        access_token: "t",
        refresh_token: null,
        expires_at: null,
        token_type: "bearer",
        scope: "read:user",
      },
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [calledUrl, calledInit] = fetchMock.mock.calls[0];
    expect(calledUrl).toBe(
      "http://database_administrator:8080/api/v1/identity/signin-callback",
    );
    expect(calledInit.method).toBe("POST");
    expect(calledInit.headers["Content-Type"]).toBe("application/json");
    expect(typeof calledInit.headers["X-Cachicamas-Timestamp"]).toBe("string");
    expect(typeof calledInit.headers["X-Cachicamas-Signature"]).toBe("string");
  });

  it("falls back to ORIGIN + PUBLIC_API_BASE_URL when SERVER_API_BASE_URL is unset", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    delete process.env.SERVER_API_BASE_URL;
    process.env.ORIGIN = "http://localhost:3015";
    process.env.PUBLIC_API_BASE_URL = "/api";

    const mod = await import("./identity-callback-client");
    await mod.postIdentityCallback({
      user: { id: "1", email: "x@y.com", name: "x", image: null },
      account: {
        provider: "github",
        providerAccountId: "1",
        access_token: null,
        refresh_token: null,
        expires_at: null,
        token_type: null,
        scope: null,
      },
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [calledUrl] = fetchMock.mock.calls[0];
    expect(calledUrl).toBe(
      "http://localhost:3015/api/v1/identity/signin-callback",
    );
  });

  it("computes the HMAC signature over ${ts}.${canonical_json} with node:crypto", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";

    const mod = await import("./identity-callback-client");
    await mod.postIdentityCallback({
      user: {
        id: "12345",
        email: "octocat@example.com",
        name: "Octocat",
        image: null,
      },
      account: {
        provider: "github",
        providerAccountId: "12345",
        access_token: "gho_test",
        refresh_token: null,
        expires_at: null,
        token_type: "bearer",
        scope: "read:user user:email",
      },
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [, calledInit] = fetchMock.mock.calls[0];
    const ts = calledInit.headers["X-Cachicamas-Timestamp"];
    const sig = calledInit.headers["X-Cachicamas-Signature"];

    // Reconstruct the expected signature with node:crypto and compare.
    const crypto = await import("node:crypto");
    const body = canonicalizeJSONForVerify(JSON.parse(calledInit.body));
    const expected = crypto
      .createHmac("sha256", process.env.IDENTITY_CALLBACK_SECRET!)
      .update(`${ts}.${body}`)
      .digest("base64");
    expect(sig).toBe(expected);
  });

  it("returns void on 204 No Content", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";
    fetchMock.mockResolvedValue({ status: 204, ok: true, text: async () => "" });

    const mod = await import("./identity-callback-client");
    const result = await mod.postIdentityCallback({
      user: { id: "1", email: "x@y.com", name: null, image: null },
      account: {
        provider: "github",
        providerAccountId: "1",
        access_token: null,
        refresh_token: null,
        expires_at: null,
        token_type: null,
        scope: null,
      },
    });
    expect(result).toBeUndefined();
  });

  it("throws on 401 (signature rejected)", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";
    fetchMock.mockResolvedValue({
      status: 401,
      ok: false,
      text: async () => JSON.stringify({ code: "unauthorized" }),
    });

    const mod = await import("./identity-callback-client");
    await expect(
      mod.postIdentityCallback({
        user: { id: "1", email: "x@y.com", name: null, image: null },
        account: {
          provider: "github",
          providerAccountId: "1",
          access_token: null,
          refresh_token: null,
          expires_at: null,
          token_type: null,
          scope: null,
        },
      }),
    ).rejects.toThrow(/unauthorized/);
  });

  it("throws on 422 (invalid body)", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";
    fetchMock.mockResolvedValue({
      status: 422,
      ok: false,
      text: async () => JSON.stringify({ code: "unprocessable_entity" }),
    });

    const mod = await import("./identity-callback-client");
    await expect(
      mod.postIdentityCallback({
        user: { id: "1", email: "x@y.com", name: null, image: null },
        account: {
          provider: "github",
          providerAccountId: "1",
          access_token: null,
          refresh_token: null,
          expires_at: null,
          token_type: null,
          scope: null,
        },
      }),
    ).rejects.toThrow(/unprocessable_entity/);
  });

  it("throws on 500 (server-side error)", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";
    fetchMock.mockResolvedValue({
      status: 500,
      ok: false,
      text: async () => JSON.stringify({ code: "internal_error" }),
    });

    const mod = await import("./identity-callback-client");
    await expect(
      mod.postIdentityCallback({
        user: { id: "1", email: "x@y.com", name: null, image: null },
        account: {
          provider: "github",
          providerAccountId: "1",
          access_token: null,
          refresh_token: null,
          expires_at: null,
          token_type: null,
          scope: null,
        },
      }),
    ).rejects.toThrow(/internal_error/);
  });

  it("throws when fetch itself fails (offline / DNS refused)", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";
    fetchMock.mockRejectedValue(new Error("ECONNREFUSED"));

    const mod = await import("./identity-callback-client");
    await expect(
      mod.postIdentityCallback({
        user: { id: "1", email: "x@y.com", name: null, image: null },
        account: {
          provider: "github",
          providerAccountId: "1",
          access_token: null,
          refresh_token: null,
          expires_at: null,
          token_type: null,
          scope: null,
        },
      }),
    ).rejects.toThrow(/ECONNREFUSED/);
  });

      // 2026-07-06-workspaces PR1a: the 5 OAuth token fields ARE
      // forwarded. The previous slice parsed them in the handler but
      // discarded them; this slice wires them through to the backend
      // so identity.account can persist them. These tests pin the
      // forwarding contract; if a future contributor silently drops
      // any of the 5, the workspaces feature (PR1c-i) breaks because
      // the GitHub proxy has no access_token to call /user/repos.

      it("forwards access_token / refresh_token / expires_at / token_type / scope in the wire body (PR1a)", async () => {
        process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
        process.env.SERVER_API_BASE_URL = "http://db:8080";

        const mod = await import("./identity-callback-client");
        await mod.postIdentityCallback({
          user: {
            id: "12345",
            email: "octocat@example.com",
            name: "Octocat",
            image: null,
          },
          account: {
            provider: "github",
            providerAccountId: "12345",
            access_token: "gho_pr1a_test",
            refresh_token: "ghr_pr1a_test",
            expires_at: 1735689600, // 2025-01-01 UTC
            token_type: "bearer",
            scope: "repo",
          },
        });

        expect(fetchMock).toHaveBeenCalledTimes(1);
        const [, calledInit] = fetchMock.mock.calls[0];
        const parsed = JSON.parse(calledInit.body);
        expect(parsed.account.accessToken).toBe("gho_pr1a_test");
        expect(parsed.account.refreshToken).toBe("ghr_pr1a_test");
        expect(parsed.account.expiresAt).toBe(1735689600);
        expect(parsed.account.tokenType).toBe("bearer");
        expect(parsed.account.scope).toBe("repo");
      });

      it("preserves null for token fields that the provider omits (pre-PR1a signIn events stay compatible)", async () => {
        process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
        process.env.SERVER_API_BASE_URL = "http://db:8080";

        const mod = await import("./identity-callback-client");
        await mod.postIdentityCallback({
          user: { id: "1", email: "x@y.com", name: null, image: null },
          account: {
            provider: "github",
            providerAccountId: "1",
            // All 5 token fields deliberately null. The body MUST
            // still carry them as null keys (not omit them) so the
            // backend's IdentityEvent receives 5 nil pointers and
            // persists SQL NULLs.
            access_token: null,
            refresh_token: null,
            expires_at: null,
            token_type: null,
            scope: null,
          },
        });

        expect(fetchMock).toHaveBeenCalledTimes(1);
        const [, calledInit] = fetchMock.mock.calls[0];
        const parsed = JSON.parse(calledInit.body);
        expect(Object.prototype.hasOwnProperty.call(parsed.account, "accessToken")).toBe(true);
        expect(parsed.account.accessToken).toBeNull();
        expect(parsed.account.refreshToken).toBeNull();
        expect(parsed.account.expiresAt).toBeNull();
        expect(parsed.account.tokenType).toBeNull();
        expect(parsed.account.scope).toBeNull();
      });

      it("preserves expires_at as a number (Auth.js convention: unix seconds, no string conversion)", async () => {
        process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
        process.env.SERVER_API_BASE_URL = "http://db:8080";

        const mod = await import("./identity-callback-client");
        await mod.postIdentityCallback({
          user: { id: "1", email: "x@y.com", name: null, image: null },
          account: {
            provider: "github",
            providerAccountId: "1",
            access_token: null,
            refresh_token: null,
            expires_at: 1234567890,
            token_type: null,
            scope: null,
          },
        });

        expect(fetchMock).toHaveBeenCalledTimes(1);
        const [, calledInit] = fetchMock.mock.calls[0];
        const parsed = JSON.parse(calledInit.body);
        expect(typeof parsed.account.expiresAt).toBe("number");
        expect(parsed.account.expiresAt).toBe(1234567890);
      });

      it("uses the timestamp from the moment of the call (no replay window on the client)", async () => {
    process.env.IDENTITY_CALLBACK_SECRET = "test-secret-32-bytes-or-more-please-okay";
    process.env.SERVER_API_BASE_URL = "http://db:8080";

    const mod = await import("./identity-callback-client");
    const before = Date.now();
    await mod.postIdentityCallback({
      user: { id: "1", email: "x@y.com", name: null, image: null },
      account: {
        provider: "github",
        providerAccountId: "1",
        access_token: null,
        refresh_token: null,
        expires_at: null,
        token_type: null,
        scope: null,
      },
    });
    const after = Date.now();
    const [, calledInit] = fetchMock.mock.calls[0];
    const ts = Number(calledInit.headers["X-Cachicamas-Timestamp"]);
    expect(ts).toBeGreaterThanOrEqual(before);
    expect(ts).toBeLessThanOrEqual(after);
  });
});

// canonicalizeJSONForVerify is the test-side mirror of canonicalizeJSON
// (production). It is duplicated here to keep the test file
// self-contained; if the production canonicalizer drifts, the HMAC
// verification above still uses the local copy, which is the
// EXPECTED behaviour (the test verifies the wire signature, not the
// production canonicalizer's output). The known-vector test at the
// top of this file is what pins the cross-tooling contract.
function canonicalizeJSONForVerify(v: unknown): string {
  if (v === null) return "null";
  if (typeof v === "boolean") return v ? "true" : "false";
  if (typeof v === "string") return JSON.stringify(v);
  if (typeof v === "number") {
    if (!Number.isFinite(v as number)) {
      throw new Error("non-finite number");
    }
    return JSON.stringify(v);
  }
  if (Array.isArray(v)) {
    return "[" + v.map(canonicalizeJSONForVerify).join(",") + "]";
  }
  if (typeof v === "object") {
    const obj = v as Record<string, unknown>;
    const keys = Object.keys(obj).sort();
    const pairs = keys.map(
      (k) => JSON.stringify(k) + ":" + canonicalizeJSONForVerify(obj[k]),
    );
    return "{" + pairs.join(",") + "}";
  }
  throw new Error(`unsupported type: ${typeof v}`);
}