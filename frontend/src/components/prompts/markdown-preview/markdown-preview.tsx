/**
 * MarkdownPreview — the right pane of the split editor.
 *
 * Renders the prompt body as formatted HTML using the `prose` class from
 * @tailwindcss/typography. The HTML reaches the DOM through
 * `dangerouslySetInnerHTML`, so it MUST be sanitized first via
 * `renderSanitizedMarkdown` (allowlist-based DOMPurify pass). See
 * spec `markdown-xss-prevention` (REQ-01).
 *
 * Props:
 *   body — the raw markdown string to render
 */

import { component$ } from "@builder.io/qwik";
import { renderSanitizedMarkdown } from "~/lib/markdown";

export interface MarkdownPreviewProps {
  body: string;
  testId?: string;
}

export const MarkdownPreview = component$<MarkdownPreviewProps>(
  ({ body, testId }) => {
    const html = renderSanitizedMarkdown(body);

    return (
      <div
        class="prose prose-sm max-w-none overflow-y-auto border border-slate-200 bg-slate-50 p-3"
        dangerouslySetInnerHTML={html}
        data-testid={testId ?? "markdown-preview"}
        aria-label="Prompt body preview"
      />
    );
  },
);
