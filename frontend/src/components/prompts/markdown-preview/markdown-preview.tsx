/**
 * MarkdownPreview — the right pane of the split editor.
 *
 * Renders the prompt body as formatted HTML using the `prose` class from
 * @tailwindcss/typography. Uses `dangerouslySetInnerHTML` because the
 * output is trusted (admin-only input, no user-generated content).
 *
 * Props:
 *   body — the raw markdown string to render
 */

import { component$ } from "@builder.io/qwik";
import { renderMarkdown } from "~/lib/markdown";

export interface MarkdownPreviewProps {
  body: string;
  testId?: string;
}

export const MarkdownPreview = component$<MarkdownPreviewProps>(
  ({ body, testId }) => {
    const html = renderMarkdown(body);

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
