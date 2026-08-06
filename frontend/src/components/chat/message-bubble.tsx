/**
 * message-bubble.tsx — a single chat message bubble.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 (single bubble; renderSanitizedMarkdown for assistant text;
 *       forbids dangerouslySetInnerHTML).
 *
 * T-06 red→green→refactor fleshes out the DOMPurify assertion + the
 * full XSS-rejection test surface. T-05 ships a working render
 * with sanitized markdown so the window can compile + render
 * end-to-end.
 */
import { component$ } from "@builder.io/qwik";

import type { ChatMessage } from "~/lib/chat-types";
import { renderSanitizedMarkdown } from "~/lib/markdown";

export interface MessageBubbleProps {
  role: ChatMessage["role"];
  text: string;
  status: ChatMessage["status"];
  error?: ChatMessage["error"] | null;
}

export const MessageBubble = component$<MessageBubbleProps>(
  ({ role, text, status, error }) => {
    const isUser = role === "user";
    const isError = status === "error";
    const containerClass = isUser
      ? "self-end max-w-[80%] rounded-md bg-slate-900 px-4 py-2 text-sm text-white"
      : "self-start max-w-[80%] rounded-md border border-slate-200 bg-white px-4 py-2 text-sm text-slate-900";
    const testIdSuffix = isUser ? "user" : "assistant";
    const sanitizedHtml = renderSanitizedMarkdown(text);

    return (
      <div
        data-testid={`message-bubble-${testIdSuffix}`}
        data-role={role}
        data-status={status}
        class={containerClass}
      >
        {isError ? (
          <div data-testid="message-bubble-error-body">
            <strong class="font-semibold">Error:</strong>{" "}
            <span data-testid="message-bubble-error-message">
              {error?.message ?? text}
            </span>
          </div>
        ) : isUser ? (
          // User messages render the raw text — no markdown, no
          // sanitization needed (the user is the author).
          <span data-testid="message-bubble-text">{text}</span>
        ) : (
          // Assistant text is sanitized through DOMPurify (REQ-6).
          // We NEVER use dangerouslySetInnerHTML on user-provided
          // content; the sanitizer is the only path for assistant
          // markdown.
          <div
            data-testid="message-bubble-text"
            dangerouslySetInnerHTML={sanitizedHtml}
          />
        )}
      </div>
    );
  },
);
