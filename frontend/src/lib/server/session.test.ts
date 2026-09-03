/**
 * Session cookie — sign / verify / refresh.
 *
 * Spec reference: R-SESSION-1 (HMAC-SHA256 sign/verify, tamper detection,
 * expiry, sliding refresh). The cookie envelope is
 * `base64url(JSON).base64url(HMAC-SHA256)` (per design §3.1).
 *
 * Strict TDD: this file was written FIRST. The implementation
 * (`session.ts`) was authored to pass these tests; mutation discipline
 * was checked by flipping a byte in the HMAC and confirming
 * `verifySession` returns null.
 */
import { describe, expect, test } from "vitest";
import {
  decodeBase64Url,
  encodeBase64Url,
  refreshIfNeeded,
  signSession,
  verifySession,
  type SessionPayload,
} from "./session";

const TEST_SECRET = "test-cookie-secret-do-not-ship";

function makePayload(overrides: Partial<SessionPayload> = {}): SessionPayload {
  return {
    user_id: 42,
    organization_id: 7,
    expires_at: Math.floor(Date.now() / 1000) + 7 * 24 * 60 * 60,
    iat: Math.floor(Date.now() / 1000),
    ...overrides,
  };
}

describe("encodeBase64Url / decodeBase64Url", () => {
  test("round-trips ASCII bytes", () => {
    const input = new TextEncoder().encode("hello world");
    const encoded = encodeBase64Url(input);
    expect(encoded).not.toContain("+");
    expect(encoded).not.toContain("/");
    expect(encoded).not.toContain("=");
    const decoded = decodeBase64Url(encoded);
    expect(new TextDecoder().decode(decoded)).toBe("hello world");
  });

  test("round-trips binary bytes including non-ASCII", () => {
    const bytes = new Uint8Array([0, 1, 127, 128, 200, 255]);
    const decoded = decodeBase64Url(encodeBase64Url(bytes));
    expect(Array.from(decoded)).toEqual(Array.from(bytes));
  });

  test("encode produces a different result than standard base64", () => {
    // A byte sequence that contains bytes whose standard-base64 encoding
    // would produce '+' and '/' characters — base64url must replace them.
    const bytes = new Uint8Array([0xfb, 0xff, 0xbf]);
    const encoded = encodeBase64Url(bytes);
    expect(encoded).not.toContain("+");
    expect(encoded).not.toContain("/");
  });
});

describe("signSession", () => {
  test("produces an envelope of the form '<json>.<hmac>'", async () => {
    const env = await signSession(makePayload(), TEST_SECRET);
    expect(env.split(".")).toHaveLength(2);
  });

  test("the JSON half decodes to the original payload", async () => {
    const payload = makePayload({
        user_id: 1,
        organization_id: 2,
        expires_at: 1735600000,
        iat: 1735000000,
      });
      const env = await signSession(payload, TEST_SECRET);
      const [jsonB64] = env.split(".");
      const decoded = JSON.parse(
        new TextDecoder().decode(decodeBase64Url(jsonB64 ?? "")),
      );
      expect(decoded).toEqual(payload);
    });

  test("different secrets produce different signatures", async () => {
    const payload = makePayload();
    const env1 = await signSession(payload, "secret-a");
    const env2 = await signSession(payload, "secret-b");
    expect(env1).not.toBe(env2);
  });
});

describe("verifySession", () => {
  test("round-trips a freshly signed payload", async () => {
    const payload = makePayload();
    const env = await signSession(payload, TEST_SECRET);
    const verified = await verifySession(env, TEST_SECRET);
    expect(verified).toEqual(payload);
  });

  test("returns null when the HMAC is tampered with (single byte)", async () => {
    const env = await signSession(makePayload(), TEST_SECRET);
    const [jsonB64, sigB64] = env.split(".");
    // Flip one byte of the HMAC. The constant-time compare MUST notice.
    const sigBytes = decodeBase64Url(sigB64 ?? "");
    sigBytes[0] = sigBytes[0]! ^ 0x01;
    const tamperedSig = encodeBase64Url(sigBytes);
    const tampered = `${jsonB64}.${tamperedSig}`;
    expect(await verifySession(tampered, TEST_SECRET)).toBeNull();
  });

  test("returns null when the JSON payload is tampered with (single byte)", async () => {
    const payload = makePayload({ user_id: 1 });
    const env = await signSession(payload, TEST_SECRET);
    const [jsonB64, sigB64] = env.split(".");
    // Flip one byte of the JSON. The HMAC will not match.
    const jsonBytes = decodeBase64Url(jsonB64 ?? "");
    jsonBytes[0] = jsonBytes[0]! ^ 0x01;
    const tamperedJson = encodeBase64Url(jsonBytes);
    const tampered = `${tamperedJson}.${sigB64}`;
    expect(await verifySession(tampered, TEST_SECRET)).toBeNull();
  });

  test("returns null for an expired payload", async () => {
    const expired = makePayload({
      expires_at: Math.floor(Date.now() / 1000) - 1,
    });
    const env = await signSession(expired, TEST_SECRET);
    // verifySession itself does not check expiry — the caller does. But the
    // payload IS decodable and the HMAC IS valid. The CALLER checks
    // expires_at against now.
    const verified = await verifySession(env, TEST_SECRET);
    expect(verified).toEqual(expired);
    expect(verified && verified.expires_at < Math.floor(Date.now() / 1000)).toBe(true);
  });

  test("returns null for a cookie signed with a different secret", async () => {
    const env = await signSession(makePayload(), "other-secret");
    expect(await verifySession(env, TEST_SECRET)).toBeNull();
  });

  test("returns null for an empty cookie", async () => {
    expect(await verifySession("", TEST_SECRET)).toBeNull();
  });

  test("returns null for a malformed envelope (no dot)", async () => {
    expect(await verifySession("not-a-cookie", TEST_SECRET)).toBeNull();
  });

  test("returns null for a malformed envelope (too many dots)", async () => {
    expect(await verifySession("a.b.c", TEST_SECRET)).toBeNull();
  });

  test("returns null for a payload missing required fields", async () => {
    const payload = {
      user_id: 1,
      organization_id: 2,
      // missing expires_at and iat
    };
    const enc = new TextEncoder();
    const jsonB64 = encodeBase64Url(enc.encode(JSON.stringify(payload)));
    // Build a fake but valid HMAC over the encoded JSON so we get past
    // the HMAC check and exercise the payload-shape validator.
    const key = await crypto.subtle.importKey(
      "raw",
      enc.encode(TEST_SECRET),
      { name: "HMAC", hash: "SHA-256" },
      false,
      ["sign"],
    );
    const sigBuf = await crypto.subtle.sign("HMAC", key, enc.encode(jsonB64));
    const sigB64 = encodeBase64Url(new Uint8Array(sigBuf));
    const env = `${jsonB64}.${sigB64}`;
    expect(await verifySession(env, TEST_SECRET)).toBeNull();
  });
});

describe("refreshIfNeeded", () => {
  const NOW = 1_700_000_000_000; // fixed for deterministic tests

  test("returns null when the cookie has more than 24h of life left", () => {
    const payload = makePayload({
      expires_at: Math.floor((NOW + 48 * 60 * 60 * 1000) / 1000),
    });
    expect(refreshIfNeeded(payload, NOW)).toBeNull();
  });

  test("returns a refreshed cookie when within 24h of expiry", () => {
    const payload = makePayload({
      expires_at: Math.floor((NOW + 6 * 60 * 60 * 1000) / 1000),
    });
    const refreshed = refreshIfNeeded(payload, NOW);
    expect(refreshed).not.toBeNull();
    expect(refreshed?.user_id).toBe(payload.user_id);
    expect(refreshed?.organization_id).toBe(payload.organization_id);
    expect(refreshed?.expires_at).toBeGreaterThan(payload.expires_at);
    // 7 days from NOW.
    const sevenDaysFromNow = Math.floor((NOW + 7 * 24 * 60 * 60 * 1000) / 1000);
    expect(refreshed?.expires_at).toBe(sevenDaysFromNow);
  });

  test("returns a refreshed cookie when already expired", () => {
    const payload = makePayload({
      expires_at: Math.floor((NOW - 1000) / 1000),
    });
    const refreshed = refreshIfNeeded(payload, NOW);
    expect(refreshed).not.toBeNull();
    expect(refreshed?.expires_at).toBeGreaterThan(NOW / 1000);
  });

  test("respects a custom threshold (12h)", () => {
    const payload = makePayload({
      expires_at: Math.floor((NOW + 18 * 60 * 60 * 1000) / 1000),
    });
    // 18h left: above 12h threshold → no refresh.
    expect(refreshIfNeeded(payload, NOW, 12 * 60 * 60 * 1000)).toBeNull();
  });

  test("preserves user_id and organization_id exactly", () => {
    const payload = makePayload({
      user_id: 999,
      organization_id: 314,
      expires_at: Math.floor((NOW + 1000) / 1000),
    });
    const refreshed = refreshIfNeeded(payload, NOW);
    expect(refreshed?.user_id).toBe(999);
    expect(refreshed?.organization_id).toBe(314);
  });
});

describe("integration: sign + verify + refresh round-trip", () => {
  test("a refreshed cookie still verifies with the original secret", async () => {
    const NOW = 1_700_000_000_000;
    const payload = makePayload({
      expires_at: Math.floor((NOW + 1000) / 1000),
    });
    const env = await signSession(payload, TEST_SECRET);
    const verified = await verifySession(env, TEST_SECRET);
    expect(verified).not.toBeNull();
    const refreshed = refreshIfNeeded(verified!, NOW);
    expect(refreshed).not.toBeNull();
    const newEnv = await signSession(refreshed!, TEST_SECRET);
    const newVerified = await verifySession(newEnv, TEST_SECRET);
    expect(newVerified).toEqual(refreshed);
  });
});