import { describe, expect, it, vi } from "vitest";
import type { IncomingMessage, ServerResponse } from "node:http";
import {
  setSecurityHeaders,
  getSecurityHeaders,
  CSP_DIRECTIVES,
} from "~/lib/security-headers";
import { routeApiRequest } from "~/lib/api-router";

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

describe("routeApiRequest", () => {
  // The chat binary registers its config endpoint under `/api/chat/*` by
  // design (see backend/agent/.../chat.RegisterAssistantConfigRoutes), so
  // both `/api/agent/*` and `/api/chat/*` must proxy to the chat binary.
  // Order matters: the more specific prefixes MUST be checked before the
  // generic `/api/*` fall-through, otherwise `/api/chat/assistant/config`
  // would be routed to database_administrator and die with a 404.
      it("/api/agent/turns routes to the agent_chat binary (agent)", () => {
        expect(routeApiRequest("/api/agent/turns")).toBe("agent");
      });

      it("/api/chat/assistant/config routes to the agent_chat binary (chat)", () => {
        expect(routeApiRequest("/api/chat/assistant/config")).toBe("chat");
      });

      it("/api/chat/anything/else also routes to the agent_chat binary (chat)", () => {
        expect(routeApiRequest("/api/chat/anything/else")).toBe("chat");
      });

      it("/api/organizations routes to the database_administrator binary (api)", () => {
        expect(routeApiRequest("/api/organizations")).toBe("api");
      });

      it("/api/v1/identity/signin-callback routes to the database_administrator binary (api)", () => {
        expect(routeApiRequest("/api/v1/identity/signin-callback")).toBe("api");
      });

      it("/some/non/api/path is NOT an API route (null)", () => {
        expect(routeApiRequest("/some/non/api/path")).toBeNull();
      });

      it("an empty URL is NOT an API route (defensive)", () => {
        expect(routeApiRequest("")).toBeNull();
      });

      // T-24 of cachicamas-archetype-system-foundation (PR-2): the
      // polymorphic /api/archetypes/* tree is hosted on the chat
      // binary. The new arm MUST be checked BEFORE the `/api/*`
      // fall-through, otherwise the catch-all would silently route
      // archetype reads/writes to database_administrator and die
      // with a 404.
      it("Test_routeApiRequest_ArchetypesArm_BeforeFallback: /api/archetypes/assistant/config resolves to 'chat'", () => {
        expect(routeApiRequest("/api/archetypes/assistant/config")).toBe(
          "chat",
        );
      });

      it("Test_routeApiRequest_ArchetypesArm_BeforeFallback: /api/archetypes?type=system resolves to 'chat'", () => {
        // The directory-list endpoint shares the polymorphic surface.
        expect(routeApiRequest("/api/archetypes?type=system")).toBe("chat");
      });

      it("Test_routeApiRequest_ArchetypesArm_AfterChat: /api/chat/... still resolves to 'chat'", () => {
        // Lock the legacy /api/chat/* arm so the new arm can't
        // accidentally absorb it.
        expect(routeApiRequest("/api/chat/assistant/config")).toBe("chat");
        expect(routeApiRequest("/api/chat/anything/else")).toBe("chat");
      });

      it("Test_routeApiRequest_ArchetypesArm_AfterChat: /api/archetypes/... does not lose ordering", () => {
        // The new arm sits AFTER the /api/chat/* arm in the dispatcher
        // (matches the comment in api-router.ts). Both must continue
        // to route to the chat binary. This guards against an
        // accidental re-ordering where /api/chat/* falls into the
        // /api/archetypes/* branch on a prefix match.
        expect(routeApiRequest("/api/chat/something/specific")).toBe("chat");
        expect(routeApiRequest("/api/archetypes/something/specific")).toBe(
          "chat",
        );
      });
    });
