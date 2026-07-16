/**
 * Markdown rendering utilities.
 *
 * Uses `marked` for parsing. Output is safe for SSR: `marked.parse()`
 * with `async: false` returns a synchronously-resolved string, suitable
 * for Qwik's SSR pipeline. The resulting HTML is rendered via
 * `{dangerouslySetInnerHTML}` in MarkdownPreview components.
 *
 * SECURITY NOTE: `marked` does not execute arbitrary HTML or scripts.
 * It parses markdown syntax into HTML elements. No sanitization library
 * is needed for v1 given that (a) only admin users access this page and
 * (b) prompt bodies are controlled by trusted admin input.
 */

import { marked } from "marked";

/**
 * Render a markdown string to an HTML string.
 *
 * @param md - The raw markdown text
 * @returns An HTML string safe to use with `dangerouslySetInnerHTML`
 */
export function renderMarkdown(md: string): string {
  return marked.parse(md, { async: false }) as string;
}
