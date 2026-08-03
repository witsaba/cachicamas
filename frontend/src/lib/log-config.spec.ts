import { describe, expect, it, afterEach, beforeEach } from "vitest";
import { logInternalTarget } from "~/lib/log-config";

/**
 * TDD coverage for the internal-target log gate.
 *
 * Reference: sdd/security-vulnerability-remediation/spec/security-response-headers
 *   REQ-03 — production MUST be silent; `DEBUG=1` is the only toggle.
 *
 * The gate is a pure function over `process.env.DEBUG`, so the test
 * sets and clears the env var around each case. This mirrors the
 * production runtime: the value is read on every invocation, not
 * captured at module load (which would defeat the point of the gate).
 */
describe("logInternalTarget", () => {
  const originalDebug = process.env.DEBUG;

  beforeEach(() => {
    delete process.env.DEBUG;
  });

  afterEach(() => {
    if (originalDebug === undefined) {
      delete process.env.DEBUG;
    } else {
      process.env.DEBUG = originalDebug;
    }
  });

  it("RED-T-SEC-LOG-001: returns false when DEBUG is unset", () => {
    delete process.env.DEBUG;
    expect(logInternalTarget()).toBe(false);
  });

  it("RED-T-SEC-LOG-002: returns false when DEBUG is anything other than '1'", () => {
    for (const v of ["0", "true", "yes", "on", "True", "TRUE", ""]) {
      process.env.DEBUG = v;
      expect(logInternalTarget(), `DEBUG=${v} should be false`).toBe(false);
    }
  });

  it("RED-T-SEC-LOG-003: returns true ONLY when DEBUG is exactly '1'", () => {
    process.env.DEBUG = "1";
    expect(logInternalTarget()).toBe(true);
  });
});
