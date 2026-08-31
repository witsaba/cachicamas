/**
 * Markdown rendering utilities.
 *
 * `renderMarkdown` is the low-level parser. It runs `marked` in sync mode
 * (suitable for Qwik SSR) and PRESERVES raw HTML embedded in the source —
 * the parser does not strip or escape `<script>`, event handlers, or
 * `javascript:` URLs. Its output is NOT safe to inject via
 * `dangerouslySetInnerHTML`.
 *
 * `renderSanitizedMarkdown` wraps `renderMarkdown` with
 * `isomorphic-dompurify`, applying a tight allowlist of tags, attributes,
 * and URI schemes. Always reach for `renderSanitizedMarkdown` when the
 * HTML reaches the DOM (prompt-body previews, etc.).
 */

import DOMPurify from "isomorphic-dompurify";
import { marked } from "marked";

interface SanitizerConfig {
  ALLOWED_TAGS: string[];
  ALLOWED_ATTR: string[];
  // Restrict URIs to safe schemes; reject javascript:, data:text/html,
  // and any obfuscated variants.
  ALLOWED_URI_REGEXP: RegExp;
}

const SANITIZER_CONFIG: SanitizerConfig = {
  ALLOWED_TAGS: [
    "p",
    "br",
    "strong",
    "em",
    "u",
    "s",
    "code",
    "pre",
    "blockquote",
    "ul",
    "ol",
    "li",
    "h1",
    "h2",
    "h3",
    "h4",
    "h5",
    "h6",
    "a",
    "img",
    "table",
    "thead",
    "tbody",
    "tr",
    "th",
    "td",
    "hr",
  ],
  ALLOWED_ATTR: ["href", "src", "alt", "title"],
  ALLOWED_URI_REGEXP: /^(?:https?:|mailto:|\/|#)/i,
};

export function renderMarkdown(md: string): string {
  return marked.parse(md, { async: false }) as string;
}

export function renderSanitizedMarkdown(md: string): string {
  const html = renderMarkdown(md);
  return DOMPurify.sanitize(html, SANITIZER_CONFIG) as string;
}
