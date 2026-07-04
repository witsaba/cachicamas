/**
 * Test for `lib/safe-avatar-src.ts`.
 *
 * Reference: `openspec/changes/cachicamas-login-ux/design.md` §7
 *   ADR-0009 — sanitize the avatar URL with a scheme allowlist that
 *   permits only `https:` and the same-origin relative paths.
 *
 * Behavioral contract:
 *   - null / undefined / "" → null (caller renders no <img>).
 *   - URL with `https:` scheme → returned as-is.
 *   - URL with any other scheme (`javascript:`, `data:`, `vbscript:`, etc.) → null.
 *   - URL with no scheme (relative path, e.g. "/uploads/avatar.png") → returned as-is.
 */
import { describe, it, expect } from "vitest";
import { safeAvatarSrc } from "./safe-avatar-src";

describe("lib/safe-avatar-src", () => {
  it("returns null for null", () => {
    expect(safeAvatarSrc(null)).toBeNull();
  });

  it("returns null for undefined", () => {
    expect(safeAvatarSrc(undefined)).toBeNull();
  });

  it("returns null for an empty string", () => {
    expect(safeAvatarSrc("")).toBeNull();
  });

  it("returns the value as-is for an https URL (GitHub avatar)", () => {
    const url =
      "https://avatars.githubusercontent.com/u/12345?v=4";
    expect(safeAvatarSrc(url)).toBe(url);
  });

  it("returns the value as-is for a same-origin relative path", () => {
    expect(safeAvatarSrc("/uploads/avatar.png")).toBe("/uploads/avatar.png");
  });

  it("returns null for a javascript: URL (R-AS-022)", () => {
    expect(safeAvatarSrc("javascript:alert(1)")).toBeNull();
  });

  it("returns null for a data: URL", () => {
    expect(safeAvatarSrc("data:text/html,<script>alert(1)</script>")).toBeNull();
  });

  it("returns null for a vbscript: URL", () => {
    expect(safeAvatarSrc("vbscript:msgbox(1)")).toBeNull();
  });

  it("returns null for a file: URL", () => {
    expect(safeAvatarSrc("file:///etc/passwd")).toBeNull();
  });

  it("returns null for an http: URL (plain HTTP, not HTTPS)", () => {
    expect(safeAvatarSrc("http://insecure.example.com/avatar.png")).toBeNull();
  });

  it("is case-insensitive on the scheme (R-AS-022)", () => {
    expect(safeAvatarSrc("JavaScript:alert(1)")).toBeNull();
    expect(safeAvatarSrc("JAVASCRIPT:alert(1)")).toBeNull();
    expect(safeAvatarSrc("HtTp://insecure.example.com/avatar.png")).toBeNull();
  });
});
