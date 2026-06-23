import { describe, expect, test } from "vitest";
import { organizationErrorMessages, organizationInputSchema } from "./organization-schema";

// Parity test: the Zod schema's error messages MUST match the Go
// *ValidationError.Fields strings from backend/.../domain/organization.go
// byte-for-byte. Both sides are the source of truth for v1; the
// frontend schema is the mirror (locked decision #4 in spec §5.1).
// If a developer ever relaxes a Go rule, this test must be updated in
// the same commit so the two sides cannot drift.

const invalidFullNameCases: ReadonlyArray<readonly [string, string, string]> = [
  ["empty fullName", "", organizationErrorMessages.nameRequired],
  ["2-char fullName", "  ab  ", organizationErrorMessages.nameLength],
  ["121-char fullName", "x".repeat(121), organizationErrorMessages.nameLength],
];

const invalidIdentificationCases: ReadonlyArray<readonly [string, string]> = [
  ["empty identification", ""],
  ["uppercase slug", "Acme-Industrial"],
  ["2-char slug", "ab"],
  ["61-char slug", "a".repeat(61)],
  ["leading hyphen slug", "-acme"],
  ["trailing hyphen slug", "acme-"],
  ["space inside slug", "acme industrial"],
];

describe("[organization-schema] byte-for-byte parity with Go validation rules", () => {
  describe("fullName", () => {
    test.each(invalidFullNameCases)(
      "rejects %s with the locked message",
      (label, value, expected) => {
        const result = organizationInputSchema.safeParse({
          fullName: value,
          identification: "valid-slug",
        });
        expect(result.success).toBe(false);
        if (!result.success) {
          const messages = result.error.issues.map((i) => i.message);
          expect(messages).toContain(expected);
        }
        // Reference label so the test name shows the case in failure output.
        expect(label.length).toBeGreaterThan(0);
      },
    );
  });

  describe("identification", () => {
    test.each(invalidIdentificationCases)("rejects %s with the locked slug message", (label, value) => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme Industrial S.A.",
        identification: value,
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(organizationErrorMessages.slugInvalid);
      }
      expect(label.length).toBeGreaterThan(0);
    });
  });

  describe("shortName", () => {
    test("rejects 41-char shortName with the locked short-name message", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme Industrial S.A.",
        identification: "acme",
        shortName: "x".repeat(41),
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(organizationErrorMessages.shortNameLength);
      }
    });

    test("accepts empty shortName (optional)", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme Industrial S.A.",
        identification: "acme",
        shortName: "",
      });
      expect(result.success).toBe(true);
    });
  });

  describe("email", () => {
    test("rejects malformed email with the locked email message", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme",
        identification: "acme",
        email: "not-an-email",
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(organizationErrorMessages.emailInvalid);
      }
    });

    test("accepts empty email (optional)", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme",
        identification: "acme",
        email: "",
      });
      expect(result.success).toBe(true);
    });
  });

  describe("phone", () => {
    test("rejects non-E.164 phone with the locked phone message", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme",
        identification: "acme",
        phone: "4155552671",
      });
      expect(result.success).toBe(false);
      if (!result.success) {
        const messages = result.error.issues.map((i) => i.message);
        expect(messages).toContain(organizationErrorMessages.phoneInvalid);
      }
    });

    test("accepts empty phone (optional)", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme",
        identification: "acme",
        phone: "",
      });
      expect(result.success).toBe(true);
    });
  });

  describe("happy path", () => {
    test("accepts a complete, valid payload", () => {
      const result = organizationInputSchema.safeParse({
        fullName: "Acme Industrial S.A.",
        identification: "acme-industrial",
        shortName: "Acme",
        email: "hello@acme.example",
        phone: "+14155552671",
      });
      expect(result.success).toBe(true);
    });
  });
});
