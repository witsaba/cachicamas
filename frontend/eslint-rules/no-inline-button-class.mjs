/**
 * ESLint rule: no-inline-button-class
 *
 * Flags inline className strings that contain the design-system variant
 * markers, in any file that imports from `~/components/ui/button` or
 * `~/components/ui/menu-item`. The rule's purpose is to prevent drift:
 * once a component consumes the design system, every button-like affordance
 * in that file must be a `<Button>` / `<MenuItem>` rather than a raw
 * `<button>` or `<a>` with the same visual tokens spelled out.
 *
 * What it flags (in files that import the primitives):
 *   - A `class="..."` attribute containing BOTH `bg-slate-900` AND `text-white`
 *     → primary intent, must use `<Button variant="primary">`.
 *   - A `class="..."` attribute containing `bg-red-700` AND `text-white`
 *     → destructive intent, must use `<Button variant="destructive">`.
 *   - A `class="..."` attribute containing `border-slate-300` AND
 *     `bg-white` AND `text-slate-900` → secondary intent, must use
 *     `<Button variant="secondary">`.
 *
 * The rule is permissive by design:
 *   - Files that do NOT import the primitives are not checked (the rule
 *     assumes such files do not need the design system — e.g. the
 *     `example` component, or third-party-style utilities).
 *   - Files in the explicit allowlist pass without check (currently:
 *     `components/sign-in-button/sign-in-button.tsx` — the brand-anchored
 *     zinc override is documented in the design system README).
 *
 * Dynamic class strings (e.g. `class={`...${variant}...`}`) are out of
 * scope for v1; the rule uses string `includes()` on the literal source.
 */

const ALLOWLIST = new Set([
  // The SignInButton's zinc dark CTA is a brand-anchored override
  // documented in the design system README. The `<Button>` wrapper
  // consumes the design system; the override is what makes it a
  // brand CTA rather than the default primary.
  "src/components/sign-in-button/sign-in-button.tsx",
]);

function importsPrimitives(text) {
  return (
    /from\s+["']~\/components\/ui\/(button|menu-item)\//.test(text) ||
    /from\s+["']~\/components\/ui\/(button|menu-item)["']/.test(text)
  );
}

function checkClassString(context, node, cls) {
  const hasBgSlate900 = /\bbg-slate-900\b/.test(cls);
  const hasBgRed700 = /\bbg-red-700\b/.test(cls);
  const hasBgRed800 = /\bbg-red-800\b/.test(cls);
  const hasTextWhite = /\btext-white\b/.test(cls);
  const hasBorderSlate300 = /\bborder-slate-300\b/.test(cls);
  const hasBgWhite = /\bbg-white\b/.test(cls);
  const hasTextSlate900 = /\btext-slate-900\b/.test(cls);

  if (hasBgSlate900 && hasTextWhite) {
    context.report({ node, messageId: "primaryDrift" });
  } else if ((hasBgRed700 || hasBgRed800) && hasTextWhite && !hasBgSlate900) {
    context.report({ node, messageId: "destructiveDrift" });
  } else if (hasBorderSlate300 && hasBgWhite && hasTextSlate900) {
    context.report({ node, messageId: "secondaryDrift" });
  }
}

const rule = {
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Flag inline button-style className strings in files that consume the Button / MenuItem primitives",
    },
    schema: [],
    messages: {
      primaryDrift:
        'Inline primary-CTA class string detected in a file that imports the Button primitive. Use `<Button variant="primary">` instead. Personalizations are allowed via the `class` prop; this rule flags the literal primary markers (bg-slate-900 + text-white).',
      destructiveDrift:
        'Inline destructive class string detected in a file that imports the Button primitive. Use `<Button variant="destructive">` instead.',
      secondaryDrift:
        'Inline secondary class string detected in a file that imports the Button primitive. Use `<Button variant="secondary">` instead.',
    },
  },

  create(context) {
    const filename = context.filename ?? context.getFilename();
    const normalized = filename.replace(/\\/g, "/");
    if (ALLOWLIST.has(normalized)) {
      return {};
    }

    const sourceCode = context.sourceCode ?? context.getSourceCode();
    const text = sourceCode.text;
    if (!importsPrimitives(text)) {
      return {};
    }

    return {
      // Only flag className attributes on actual button-shaped elements.
      // Inline chips / monograms / status badges that happen to use the
      // same color tokens (e.g. `bg-slate-900 text-white` for a "Selected"
      // pill) are NOT buttons and are out of scope for this rule.
      'JSXAttribute[name.name="class"], JSXAttribute[name.name="className"]'(
        node,
      ) {
        const parent = node.parent;
        if (!parent || parent.type !== "JSXOpeningElement") return;
        const tagName =
          parent.name && parent.name.name ? parent.name.name : null;
        if (tagName !== "button" && tagName !== "a") return;
        if (node.value && typeof node.value.value === "string") {
          checkClassString(context, node, node.value.value);
        }
      },
    };
  },
};

export default rule;
