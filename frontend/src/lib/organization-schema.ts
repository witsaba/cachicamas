/**
 * Shared Zod schema for the organizations form.
 *
 * This is the SINGLE SOURCE OF TRUTH for client-side validation
 * of an organization payload.  The Go server-side rules in
 * `backend/database_administrator/src/domain/organization.go`
 * MUST match this schema byte-for-byte (regex, lengths, error
 * messages).  The parity test in `organization-schema.spec.ts`
 * cross-asserts both sides cannot drift.
 *
 * Locked decisions honoured here:
 *
 *   - Spec §2.3: full_name 3–120 chars after trim; identification
 *     matches `^[a-z0-9][a-z0-9-]{1,58}[a-z0-9]$`; shortname ≤40;
 *     email RFC 5322; phone E.164 `^\+[1-9]\d{1,14}$`.
 *   - Spec §5.1: every error message string below is the exact
 *     one rendered inline in the form.  The backend
 *     `*ValidationError.Fields` map uses the same strings.
 *   - Locked #4 (proposal): optional fields are sent as empty
 *     string by the Qwik form (Qwik's `useStore` cannot store
 *     `undefined` for `<input>`s reliably).  We accept the empty
 *     string variant of every optional field via `.or(z.literal(""))`.
 */
import { z } from "zod";

/**
 * Locked error messages.  Centralised so a typo in one place
 * cannot diverge from the spec.  Mirrored in
 * `backend/database_administrator/src/domain/organization.go`.
 */
export const organizationErrorMessages = {
  nameRequired: "Name is required.",
  nameLength: "Name must be 3–120 characters.",
  slugRequired: "Slug is required.",
  slugInvalid:
    "Slug must be 3–60 characters, lowercase letters, digits, and hyphens; must start and end with a letter or digit.",
  shortNameLength: "Short name must be 40 characters or fewer.",
  emailInvalid: "Email is not a valid email address.",
  phoneInvalid: "Phone must be in E.164 format (e.g. +14155552671).",
} as const;

/**
 * Slug regex from spec §2.3 — 3–60 chars, alphanumeric start/end,
 * hyphens allowed only in the middle.  Mirrored in Go
 * `var slugRegex = regexp.MustCompile(...)`.
 */
export const SLUG_REGEX = /^[a-z0-9][a-z0-9-]{1,58}[a-z0-9]$/;

/** E.164 phone regex from spec §2.3. */
export const PHONE_REGEX = /^\+[1-9]\d{1,14}$/;

/**
 * Required shape used by the create form.  Optional fields
 * accept the empty string as a valid value (the form submits
 * `""` instead of `undefined` so server-side Zod can parse
 * form-encoded bodies uniformly).
 */
export const organizationInputSchema = z.object({
  fullName: z
    .string()
    .trim()
    .min(1, organizationErrorMessages.nameRequired)
    .min(3, organizationErrorMessages.nameLength)
    .max(120, organizationErrorMessages.nameLength),
  identification: z
    .string()
    .min(1, organizationErrorMessages.slugRequired)
    .regex(SLUG_REGEX, organizationErrorMessages.slugInvalid),
  shortName: z
    .string()
    .max(40, organizationErrorMessages.shortNameLength)
    .optional()
    .or(z.literal("")),
  email: z
    .string()
    .email(organizationErrorMessages.emailInvalid)
    .optional()
    .or(z.literal("")),
  phone: z
    .string()
    .regex(PHONE_REGEX, organizationErrorMessages.phoneInvalid)
    .optional()
    .or(z.literal("")),
});

/** Inferred TypeScript shape of the form's `useStore`. */
export type OrganizationInput = z.infer<typeof organizationInputSchema>;
