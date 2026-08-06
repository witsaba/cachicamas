/**
 * message-bubble.spec.tsx — vitest coverage for MessageBubble.
 *
 * Reference: openspec/specs/frontend-chat-layer1/spec.md REQ-6, REQ-7.
 *
 * Strict TDD target. MessageBubble is the single chat message bubble.
 * The spec covers REQ-6 (DOMPurify sanitization) and REQ-7 (per-file
 * spec discipline). It is structural: no EventSource, no EventSource
 * mock, no SSE subscriber — the bubble is a pure render component
 * that takes a `role` + `text` + `status` + `error` and emits HTML.
 *
 * Test scope:
 *   - REQ-6 S-6.a — markdown structure is preserved by the
 *     allowlist (bold, code, list)
 *   - REQ-6 S-6.b — `<script>` and `onerror=` are stripped by
 *     DOMPurify; no executable DOM is emitted
 *   - role-based affordances (user: solid bubble; assistant: bordered)
 *   - error status renders the typed error body inline
 *   - user text is rendered as literal text (no markdown, no
 *     sanitization needed — the user is the author)
 *   - source does NOT use `dangerouslySetInnerHTML` directly
 *     (the sanitizer is the only path for assistant markdown)
 */
import { createDOM } from "@builder.io/qwik/testing";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

import { MessageBubble } from "./message-bubble";

const here = fileURLToPath(import.meta.url);
const bubblePath = here.replace(/\/message-bubble\.spec\.tsx$/, "/message-bubble.tsx");

describe("components/chat/message-bubble (REQ-6, REQ-7)", () => {
  it("renders a user bubble with literal text (no sanitization needed — the user is the author)", async () => {
    const { render, screen } = await createDOM();
    await render(<MessageBubble role="user" text="hello world" status="complete" />);
    const bubble = screen.querySelector('[data-testid="message-bubble-user"]');
    expect(bubble).toBeTruthy();
    expect(bubble?.textContent).toContain("hello world");
    // User bubbles use the literal-text span, NOT the sanitized div.
    const text = screen.querySelector('[data-testid="message-bubble-text"]');
    expect(text).toBeTruthy();
    expect(text?.textContent).toBe("hello world");
  });

  it("renders an assistant bubble with sanitized markdown (REQ-6 S-6.a)", async () => {
    const { render, screen } = await createDOM();
    // Use a template literal (NOT a JSX attribute string) so the
    // `\n\n` is a real newline pair — JSX attribute strings do NOT
    // interpret backslash escapes. Markdown lists require a blank
    // line before them; the literal `\n` characters would render
    // as text inside the <p> instead.
    const md = `**bold** and \`code\`\n\n- item`;
    await render(
      <MessageBubble role="assistant" text={md} status="complete" />,
    );
    const bubble = screen.querySelector('[data-testid="message-bubble-assistant"]');
    expect(bubble).toBeTruthy();
    // REQ-6 S-6.a — the allowlist preserves <strong>, <code>, and
    // <ul><li> from the markdown. We assert the rendered HTML
    // (innerHTML) contains those tags and no other tag kinds.
    const text = screen.querySelector('[data-testid="message-bubble-text"]');
    expect(text).toBeTruthy();
    const html = text?.innerHTML ?? "";
    expect(html).toMatch(/<strong>/);
    expect(html).toMatch(/<code>/);
    expect(html).toMatch(/<ul>/);
    expect(html).toMatch(/<li>/);
    // No disallowed tags leaked (sanity check).
    expect(html).not.toContain("<script");
    expect(html).not.toContain("<iframe");
    expect(html).not.toContain("onerror=");
  });

  it("strips <script> from assistant text (REQ-6 S-6.b — XSS rejection)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <MessageBubble
        role="assistant"
        text='<script>alert("xss")</script>safe'
        status="complete"
      />,
    );
    const bubble = screen.querySelector('[data-testid="message-bubble-assistant"]');
    expect(bubble).toBeTruthy();
    const html = bubble?.innerHTML ?? "";
    // REQ-6 S-6.b — NO <script> element reaches the DOM. The
    // allowlist rejects the tag outright; only the literal text
    // survives (escaped as text content if anything). Either way,
    // there is no <script> tag and no executable payload.
    expect(html.toLowerCase()).not.toContain("<script");
    expect(html.toLowerCase()).not.toContain("alert(");
    // The "safe" suffix should survive (markup is stripped, text
    // content is preserved where the allowlist permits).
    expect(bubble?.textContent).toContain("safe");
  });

  it("strips onerror= event handlers from assistant text (REQ-6 S-6.b — XSS rejection)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <MessageBubble
        role="assistant"
        text='<img src="x" onerror="alert(1)" />safe'
        status="complete"
      />,
    );
    const bubble = screen.querySelector('[data-testid="message-bubble-assistant"]');
    expect(bubble).toBeTruthy();
    const html = bubble?.innerHTML ?? "";
    // REQ-6 S-6.b — onerror= attribute must not reach the DOM. The
    // allowlist's ALLOWED_ATTR list (lib/markdown.ts:57) explicitly
    // excludes event handlers; only href/src/alt/title survive.
    expect(html.toLowerCase()).not.toContain("onerror");
    expect(html.toLowerCase()).not.toContain("alert(");
  });

  it("renders an error body when status='error' (REQ-4 S-4.a)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <MessageBubble
        role="assistant"
        text=""
        status="error"
        error={{ kind: "server", message: "upstream unavailable" }}
      />,
    );
    const bubble = screen.querySelector('[data-testid="message-bubble-assistant"]');
    expect(bubble).toBeTruthy();
    const errorBody = screen.querySelector(
      '[data-testid="message-bubble-error-body"]',
    );
    expect(errorBody).toBeTruthy();
    const errorMessage = screen.querySelector(
      '[data-testid="message-bubble-error-message"]',
    );
    expect(errorMessage?.textContent).toBe("upstream unavailable");
  });

  it("uses the right container class for user vs assistant (REQ-1 visual contract)", async () => {
    const { render, screen } = await createDOM();
    await render(
      <MessageBubble role="user" text="hi" status="complete" />,
    );
    const userBubble = screen.querySelector('[data-testid="message-bubble-user"]');
    expect(userBubble?.className).toContain("self-end");
    expect(userBubble?.className).toContain("bg-slate-900");

    const { render: render2, screen: screen2 } = await createDOM();
    await render2(
      <MessageBubble role="assistant" text="hello" status="complete" />,
    );
    const assistantBubble = screen2.querySelector(
      '[data-testid="message-bubble-assistant"]',
    );
    expect(assistantBubble?.className).toContain("self-start");
    expect(assistantBubble?.className).toContain("bg-white");
  });

  it("source does NOT use dangerouslySetInnerHTML directly on user text (REQ-6 S-6.b contract)", async () => {
    // The contract is: assistant text may use dangerouslySetInnerHTML
    // (the sanitizer is the only path for assistant markdown), but
    // user text MUST be rendered as plain text. The source must NOT
    // apply dangerouslySetInnerHTML to a `role: "user"` path. We
    // assert this structurally so a future contributor cannot widen
    // the XSS surface by accident.
    const source = readFileSync(bubblePath, "utf8");
    // The source has at most ONE dangerouslySetInnerHTML, and it is
    // on the assistant path (after the `isUser ? ... : ...` ternary).
    // Easiest structural guard: the bubble's user-text path must
    // wrap text in a plain span with no innerHTML.
    expect(source).toContain("message-bubble-text");
    expect(source).toContain("dangerouslySetInnerHTML");
    // User path: a <span data-testid="message-bubble-text">{text}</span>
    // with NO dangerouslySetInnerHTML inside the user ternary. Assert
    // by substring scan: find the user branch and confirm no
    // dangerouslySetInnerHTML appears between `isUser ?` and `: (`.
    const userBranch = source.match(/isUser\s*\?\s*\([\s\S]*?\)\s*:\s*\(/);
    expect(userBranch).toBeTruthy();
    expect(userBranch?.[0]).not.toContain("dangerouslySetInnerHTML");
  });
});