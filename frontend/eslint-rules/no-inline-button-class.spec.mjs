/**
 * Tests for the `no-inline-button-class` ESLint rule.
 *
 * Uses ESLint's RuleTester to exercise the rule against synthetic source
 * snippets. Verifies:
 *   - The rule is a no-op for files that do not import the primitives.
 *   - The rule flags primary / secondary / destructive drift on actual
 *     <button> / <a> JSX elements.
 *   - The rule does NOT flag inline chips / spans / divs (only button-shaped
 *     elements trigger it).
 *   - The rule respects the allowlist for documented brand overrides.
 *
 * Run with: node --test eslint-rules/no-inline-button-class.spec.mjs
 */
import { RuleTester } from "eslint";
import rule from "./no-inline-button-class.mjs";

const ruleTester = new RuleTester({
  languageOptions: {
    parserOptions: {
      ecmaVersion: 2022,
      sourceType: "module",
      ecmaFeatures: { jsx: true },
    },
  },
});

// Synthetic source that imports the primitives so the rule is enabled.
const IMPORT_LINE = 'import { Button } from "~/components/ui/button/button";\n';

ruleTester.run("no-inline-button-class", rule, {
  valid: [
    // Files without the primitive import are not checked.
    {
      code: '<button class="bg-slate-900 text-white">x</button>',
      filename: "/app/src/components/no-import.tsx",
    },
    // Inline chip (NOT a button) — should not be flagged.
    {
      code:
        IMPORT_LINE +
        '<span class="bg-slate-900 text-white px-2 py-0.5">Selected</span>',
      filename: "/app/src/components/chip.tsx",
    },
    // Inline monogram (NOT a button) — should not be flagged.
    {
      code:
        IMPORT_LINE +
        '<span class="inline-flex h-6 w-6 rounded bg-slate-900 text-white">M</span>',
      filename: "/app/src/components/monogram.tsx",
    },
    // The Button primitive itself is the source of truth — its own
    // classes are flagged for review but excluded by tag check (the
    // primitive is <button>, so this rule DOES see it — we test the
    // exact className below to ensure no false positive on the
    // legitimate system tokens).
    {
      code: IMPORT_LINE + '<Button variant="primary">Save</Button>',
      filename: "/app/src/components/clean.tsx",
    },
    // Documented `!important` overrides (e.g. the avatar trigger's
    // `!rounded-full` to override the variant's `rounded-md`) are
    // intentionally excluded by the rule — the `!` prefix signals an
    // intentional override of a system token. Without stripping `!`,
    // this case would false-positive.
    {
      code:
        IMPORT_LINE +
        '<button class="!rounded-full h-10 w-10 overflow-hidden">x</button>',
      filename: "/app/src/components/avatar.tsx",
    },
    // Allowlisted file (SignInButton zinc override) passes.
    {
      code:
        IMPORT_LINE +
        '<button class="inline-flex cursor-pointer items-center gap-2 rounded-md border border-zinc-700 bg-zinc-900 px-4 py-2 text-sm font-medium text-zinc-100 shadow-sm transition-[background-color,box-shadow,transform,border-color] duration-150 hover:border-zinc-600 hover:bg-zinc-800 hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-zinc-500 active:translate-y-px">Sign in</button>',
      filename: "/app/src/components/sign-in-button/sign-in-button.tsx",
    },
  ],
  invalid: [
    // Primary drift on a <button> — flagged.
    {
      code: IMPORT_LINE + '<button class="bg-slate-900 text-white">x</button>',
      filename: "/app/src/components/drift.tsx",
      errors: [{ messageId: "primaryDrift" }],
    },
    // Primary drift on an <a> — flagged.
    {
      code: IMPORT_LINE + '<a href="/x" class="bg-slate-900 text-white">x</a>',
      filename: "/app/src/components/drift.tsx",
      errors: [{ messageId: "primaryDrift" }],
    },
    // Destructive drift on a <button> — flagged.
    {
      code:
        IMPORT_LINE + '<button class="bg-red-700 text-white">Delete</button>',
      filename: "/app/src/components/drift.tsx",
      errors: [{ messageId: "destructiveDrift" }],
    },
    // Secondary drift on a <button> — flagged.
    {
      code:
        IMPORT_LINE +
        '<button class="bg-white border border-slate-300 text-slate-900">Cancel</button>',
      filename: "/app/src/components/drift.tsx",
      errors: [{ messageId: "secondaryDrift" }],
    },
    // Hover-bg variant of destructive — also flagged (hover state on the
    // raw button class is the same drift pattern).
    {
      code:
        IMPORT_LINE + '<button class="hover:bg-red-800 text-white">x</button>',
      filename: "/app/src/components/drift.tsx",
      errors: [{ messageId: "destructiveDrift" }],
    },
  ],
});
