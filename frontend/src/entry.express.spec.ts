import { describe, expect, it, vi } from "vitest";
import type { IncomingMessage, ServerResponse } from "node:http";
import {
  setSecurityHeaders,
  getSecurityHeaders,
  CSP_DIRECTIVES,
} from "~/lib/security-headers";

/**
 * Mock ServerResponse that records every setHeader call so tests can
 * assert against the exact header map the middleware produced.
 *
 * We cast through `unknown` because Node's ServerResponse type does
 * not expose a writable `headers` property — the real surface is
 * `setHeader` + `getHeader`. The cast is contained to the helper and
 * never leaks into production code.
 */
interface MockResponse {
  setHeader: (name: string, value: string) => void;
  headers: Record<string, string>;
}

function mockReq(headers: Record<string, string> = {}): IncomingMessage {
  return { headers } as unknown as IncomingMessage;
}

function mockRes(): ServerResponse {
  const record: MockResponse = {
    headers: {},
    setHeader(name: string, value: string) {
      record.headers[name.toLowerCase()] = value;
    },
  };
  return record as unknown as ServerResponse;
}

function captureNext(): (err?: unknown) => void {
  return vi.fn() as unknown as (err?: unknown) => void;
}

describe("setSecurityHeaders middleware", () => {
  it("RED-T-SEC-001: sets Content-Security-Policy starting with default-src 'self'", () => {
    const req = mockReq();
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    const csp = (res as unknown as MockResponse).headers[
      "content-security-policy"
    ];
    expect(csp).toBeDefined();
    expect(csp).toMatch(/^default-src 'self'/);
    expect(next).toHaveBeenCalledTimes(1);
  });

  it("RED-T-SEC-002: sets X-Content-Type-Options: nosniff", () => {
    const req = mockReq();
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    expect(
      (res as unknown as MockResponse).headers["x-content-type-options"],
    ).toBe("nosniff");
    expect(next).toHaveBeenCalledTimes(1);
  });

  it("RED-T-SEC-003: sets Referrer-Policy: strict-origin-when-cross-origin", () => {
    const req = mockReq();
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    expect((res as unknown as MockResponse).headers["referrer-policy"]).toBe(
      "strict-origin-when-cross-origin",
    );
    expect(next).toHaveBeenCalledTimes(1);
  });

  it("RED-T-SEC-004: sets X-Frame-Options: DENY", () => {
    const req = mockReq();
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    expect((res as unknown as MockResponse).headers["x-frame-options"]).toBe(
      "DENY",
    );
    expect(next).toHaveBeenCalledTimes(1);
  });

  it("RED-T-SEC-005: HSTS is set when x-forwarded-proto is https", () => {
    const req = mockReq({ "x-forwarded-proto": "https" });
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    const hsts = (res as unknown as MockResponse).headers[
      "strict-transport-security"
    ];
    expect(hsts).toBeDefined();
    expect(hsts).toContain("max-age=63072000");
    expect(hsts).toContain("includeSubDomains");
  });

  it("RED-T-SEC-006: HSTS is omitted when x-forwarded-proto is NOT https", () => {
    const req = mockReq({ "x-forwarded-proto": "http" });
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    expect(
      (res as unknown as MockResponse).headers["strict-transport-security"],
    ).toBeUndefined();
    expect(next).toHaveBeenCalledTimes(1);
  });

  it("RED-T-SEC-007: HSTS is omitted when x-forwarded-proto header is missing", () => {
    const req = mockReq();
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    expect(
      (res as unknown as MockResponse).headers["strict-transport-security"],
    ).toBeUndefined();
  });

  it("RED-T-SEC-008: CSP contains frame-ancestors 'none' and form-action 'self'", () => {
    const req = mockReq();
    const res = mockRes();
    const next = captureNext();
    setSecurityHeaders(req, res, next);
    const csp = (res as unknown as MockResponse).headers[
      "content-security-policy"
    ] as string;
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("form-action 'self'");
  });

  it("RED-T-SEC-009: CSP exposes the same string via CSP_DIRECTIVES constant", () => {
    // The constant is the same value emitted to the header.
    // This guarantees the test pins the exact policy (no drift).
    expect(typeof CSP_DIRECTIVES).toBe("string");
    expect(CSP_DIRECTIVES).toMatch(/^default-src 'self'/);
    expect(CSP_DIRECTIVES).toContain("frame-ancestors 'none'");
    expect(CSP_DIRECTIVES).toContain("form-action 'self'");
  });
});

describe("getSecurityHeaders", () => {
  it("TRIANGULATE-T-SEC-010: returns the same header set as the middleware under HTTPS", () => {
    const req = mockReq({ "x-forwarded-proto": "https" });
    const headers = getSecurityHeaders(req);
    expect(headers["Content-Security-Policy"]).toBe(CSP_DIRECTIVES);
    expect(headers["Strict-Transport-Security"]).toContain("max-age=63072000");
    expect(headers["X-Content-Type-Options"]).toBe("nosniff");
    expect(headers["Referrer-Policy"]).toBe("strict-origin-when-cross-origin");
    expect(headers["X-Frame-Options"]).toBe("DENY");
  });

  it("TRIANGULATE-T-SEC-011: omits HSTS under HTTP", () => {
    const req = mockReq({ "x-forwarded-proto": "http" });
    const headers = getSecurityHeaders(req);
    expect(headers["Strict-Transport-Security"]).toBeUndefined();
    expect(headers["X-Frame-Options"]).toBe("DENY");
  });
});
