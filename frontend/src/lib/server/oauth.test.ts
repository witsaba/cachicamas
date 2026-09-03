/**
 * Google OAuth — URL builder + state generator + token exchange + userinfo.
 *
 * Spec reference: R-OAUTH-1. Strict TDD: tests written first; production
 * authored to satisfy them; mutation discipline confirmed by flipping
 * single bytes and asserting failure.
 */
import { describe, expect, test, vi } from "vitest";
import {
  buildAuthUrl,
  callBackendBootstrap,
  DEFAULT_SCOPES,
  exchangeCode,
  fetchUserInfo,
  generateState,
  GOOGLE_AUTH_URL,
  GOOGLE_TOKEN_URL,
  GOOGLE_USERINFO_URL,
} from "./oauth";

describe("generateState", () => {
  test("returns a non-empty string", () => {
    const s = generateState();
    expect(s).toBeTruthy();
    expect(typeof s).toBe("string");
  });

  test("uses base64url alphabet only", () => {
    const s = generateState();
    expect(s).not.toContain("+");
    expect(s).not.toContain("/");
    expect(s).not.toContain("=");
  });

  test("returns distinct values across calls", () => {
    const a = generateState();
    const b = generateState();
    expect(a).not.toBe(b);
  });

  test("encodes at least 32 bytes of entropy (43 base64url chars for 32 bytes)", () => {
    const s = generateState();
    // 32 bytes → ceil(32 * 4 / 3) = 43 base64url chars (no padding).
    expect(s.length).toBeGreaterThanOrEqual(43);
  });
});

describe("buildAuthUrl", () => {
  test("points at the Google OAuth endpoint", () => {
    const url = buildAuthUrl({
      clientId: "client-123",
      redirectUri: "https://example.com/auth/google/callback",
      state: "abc",
    });
    expect(url.origin + url.pathname).toBe(GOOGLE_AUTH_URL);
  });

  test("includes client_id, redirect_uri, response_type=code", () => {
    const url = buildAuthUrl({
      clientId: "client-123",
      redirectUri: "https://example.com/auth/google/callback",
      state: "abc",
    });
    expect(url.searchParams.get("client_id")).toBe("client-123");
    expect(url.searchParams.get("redirect_uri")).toBe(
      "https://example.com/auth/google/callback",
    );
    expect(url.searchParams.get("response_type")).toBe("code");
  });

  test("defaults scope to 'openid email profile'", () => {
    const url = buildAuthUrl({
      clientId: "x",
      redirectUri: "https://e.com/cb",
      state: "s",
    });
    expect(url.searchParams.get("scope")).toBe("openid email profile");
  });

  test("honours a custom scope", () => {
    const url = buildAuthUrl({
      clientId: "x",
      redirectUri: "https://e.com/cb",
      state: "s",
      scopes: ["openid", "email"],
    });
    expect(url.searchParams.get("scope")).toBe("openid email");
  });

  test("echoes the state verbatim", () => {
    const url = buildAuthUrl({
      clientId: "x",
      redirectUri: "https://e.com/cb",
      state: "the-state-token",
    });
    expect(url.searchParams.get("state")).toBe("the-state-token");
  });

  test("OMITS client_secret from the redirect URL", () => {
    // Per R-FE-001 / S-FE-002: client_secret never appears in the URL
    // (it would leak via Referer header, server logs, browser history).
    const url = buildAuthUrl({
      clientId: "x",
      redirectUri: "https://e.com/cb",
      state: "s",
    });
    expect(url.toString()).not.toContain("client_secret");
    expect(url.toString()).not.toContain("secret");
    // Also confirm DEFAULT_SCOPES does not contain anything sensitive.
    for (const s of DEFAULT_SCOPES) {
      expect(url.searchParams.get("scope") ?? "").not.toContain("secret");
    }
  });

  test("OMITS PKCE parameters (MVP simplification)", () => {
    const url = buildAuthUrl({
      clientId: "x",
      redirectUri: "https://e.com/cb",
      state: "s",
    });
    expect(url.searchParams.has("code_challenge")).toBe(false);
    expect(url.searchParams.has("code_challenge_method")).toBe(false);
  });

  test("throws on missing clientId / redirectUri / state", () => {
    expect(() =>
      buildAuthUrl({ clientId: "", redirectUri: "r", state: "s" }),
    ).toThrow(/clientId required/);
    expect(() =>
      buildAuthUrl({ clientId: "c", redirectUri: "", state: "s" }),
    ).toThrow(/redirectUri required/);
    expect(() =>
      buildAuthUrl({ clientId: "c", redirectUri: "r", state: "" }),
    ).toThrow(/state required/);
  });
});

describe("exchangeCode", () => {
  test("POSTs to the token endpoint with form-encoded body", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ access_token: "tok", token_type: "Bearer" }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
    await exchangeCode({
      code: "the-code",
      clientId: "cid",
      clientSecret: "csecret",
      redirectUri: "https://e.com/cb",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [calledUrl, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(calledUrl).toBe(GOOGLE_TOKEN_URL);
    expect(init.method).toBe("POST");
    const headers = init.headers as Record<string, string>;
    expect(headers["content-type"]).toBe("application/x-www-form-urlencoded");
    const body = init.body as URLSearchParams;
    expect(body.get("code")).toBe("the-code");
    expect(body.get("client_id")).toBe("cid");
    expect(body.get("client_secret")).toBe("csecret");
    expect(body.get("redirect_uri")).toBe("https://e.com/cb");
    expect(body.get("grant_type")).toBe("authorization_code");
  });

  test("returns the parsed token response on 2xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          access_token: "tok",
          id_token: "id-tok",
          expires_in: 3600,
          token_type: "Bearer",
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
    const result = await exchangeCode({
      code: "c",
      clientId: "i",
      clientSecret: "s",
      redirectUri: "r",
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    expect(result.access_token).toBe("tok");
    expect(result.id_token).toBe("id-tok");
    expect(result.expires_in).toBe(3600);
    expect(result.token_type).toBe("Bearer");
  });

  test("throws OAuthExchangeError on non-2xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: "invalid_grant" }), {
        status: 400,
        headers: { "content-type": "application/json" },
      }),
    );
    await expect(
      exchangeCode({
        code: "c",
        clientId: "i",
        clientSecret: "s",
        redirectUri: "r",
        fetchImpl: fetchMock as unknown as typeof fetch,
      }),
    ).rejects.toMatchObject({ status: 400, name: "OAuthExchangeError" });
  });

  test("throws when access_token is missing from a 2xx response", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ expires_in: 0 }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    await expect(
      exchangeCode({
        code: "c",
        clientId: "i",
        clientSecret: "s",
        redirectUri: "r",
        fetchImpl: fetchMock as unknown as typeof fetch,
      }),
    ).rejects.toThrow(/missing access_token/);
  });
});

describe("fetchUserInfo", () => {
  test("GETs the userinfo endpoint with Bearer auth", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "sub-1",
          email: "founder@example.com",
          email_verified: true,
          name: "Founder",
          picture: "https://example.com/p.png",
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
    const result = await fetchUserInfo(
      "tok",
      fetchMock as unknown as typeof fetch,
    );
    const [calledUrl, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(calledUrl).toBe(GOOGLE_USERINFO_URL);
    const headers = init.headers as Record<string, string>;
    expect(headers.authorization).toBe("Bearer tok");
    expect(result).toEqual({
      google_sub: "sub-1",
      email: "founder@example.com",
      email_verified: true,
      name: "Founder",
      picture_url: "https://example.com/p.png",
    });
  });

  test("accepts sub as alternative to id (Google sometimes returns 'sub')", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          sub: "sub-2",
          email: "x@example.com",
          email_verified: false,
        }),
        { status: 200 },
      ),
    );
    const result = await fetchUserInfo(
      "tok",
      fetchMock as unknown as typeof fetch,
    );
    expect(result.google_sub).toBe("sub-2");
  });

  test("throws OAuthUserInfoError on non-2xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("server error", { status: 500 }),
    );
    await expect(
      fetchUserInfo("tok", fetchMock as unknown as typeof fetch),
    ).rejects.toMatchObject({ status: 500, name: "OAuthUserInfoError" });
  });

  test("throws when sub/id is missing", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ email: "x@example.com" }),
        { status: 200 },
      ),
    );
    await expect(
      fetchUserInfo("tok", fetchMock as unknown as typeof fetch),
    ).rejects.toThrow(/missing sub\/id/);
  });

  test("throws when email is missing", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "sub" }), { status: 200 }),
    );
    await expect(
      fetchUserInfo("tok", fetchMock as unknown as typeof fetch),
    ).rejects.toThrow(/missing email/);
  });

  test("defaults email_verified to false when absent", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ id: "sub", email: "x@example.com" }),
        { status: 200 },
      ),
    );
    const result = await fetchUserInfo(
      "tok",
      fetchMock as unknown as typeof fetch,
    );
    expect(result.email_verified).toBe(false);
  });

  test("defaults name and picture_url to empty strings when absent", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ id: "sub", email: "x@example.com" }),
        { status: 200 },
      ),
    );
    const result = await fetchUserInfo(
      "tok",
      fetchMock as unknown as typeof fetch,
    );
    expect(result.name).toBe("");
    expect(result.picture_url).toBe("");
  });
});

describe("callBackendBootstrap", () => {
  const claims = {
    google_sub: "sub-1",
    email: "founder@example.com",
    email_verified: true,
    name: "Founder",
    picture_url: "https://example.com/p.png",
  };

  test("POSTs /internal/auth/bootstrap with X-Internal-Secret", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          user_id: 1,
          organization_id: 1,
          status: "active",
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
    await callBackendBootstrap({
      backendUrl: "http://database_administrator:8080",
      internalSecret: "the-secret",
      claims,
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    const [calledUrl, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(calledUrl).toBe(
      "http://database_administrator:8080/internal/auth/bootstrap",
    );
    expect(init.method).toBe("POST");
    const headers = init.headers as Record<string, string>;
    expect(headers["x-internal-secret"]).toBe("the-secret");
    expect(headers["content-type"]).toBe("application/json");
    expect(JSON.parse(init.body as string)).toEqual(claims);
  });

  test("returns parsed bootstrap result on 2xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          user_id: 42,
          organization_id: 7,
          status: "active",
        }),
        { status: 200 },
      ),
    );
    const result = await callBackendBootstrap({
      backendUrl: "http://b:8080",
      internalSecret: "s",
      claims,
      fetchImpl: fetchMock as unknown as typeof fetch,
    });
    expect(result).toEqual({
      user_id: 42,
      organization_id: 7,
      status: "active",
    });
  });

  test("throws BootstrapError on non-2xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("unauthorized", { status: 401 }),
    );
    await expect(
      callBackendBootstrap({
        backendUrl: "http://b:8080",
        internalSecret: "s",
        claims,
        fetchImpl: fetchMock as unknown as typeof fetch,
      }),
    ).rejects.toMatchObject({ status: 401, name: "BootstrapError" });
  });

  test("throws on missing backendUrl / internalSecret", async () => {
    await expect(
      callBackendBootstrap({
        backendUrl: "",
        internalSecret: "s",
        claims,
        fetchImpl: vi.fn() as unknown as typeof fetch,
      }),
    ).rejects.toThrow(/backendUrl required/);
    await expect(
      callBackendBootstrap({
        backendUrl: "http://b:8080",
        internalSecret: "",
        claims,
        fetchImpl: vi.fn() as unknown as typeof fetch,
      }),
    ).rejects.toThrow(/internalSecret required/);
  });
});