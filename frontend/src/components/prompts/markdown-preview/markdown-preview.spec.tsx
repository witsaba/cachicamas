/**
 * Component-level XSS regression tests for `MarkdownPreview`.
 *
 * Per spec `markdown-xss-prevention` REQ-01 — all preview Markdown SHALL be
 * sanitized before HTML insertion (dangerouslySetInnerHTML). The component
 * must therefore call `renderSanitizedMarkdown`, not `renderMarkdown`.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, expect, it } from "vitest";
import { MarkdownPreview } from "./markdown-preview";

describe("components/prompts/markdown-preview — sanitization", () => {
  it("does not execute an inline <script> tag from the prompt body", async () => {
    const { screen, render } = await createDOM();
    await render(<MarkdownPreview body='<script>alert(1)</script>' />);

    const scriptCount = screen.querySelectorAll("script").length;
    expect(scriptCount).toBe(0);

    const root = screen.querySelector('[data-testid="markdown-preview"]');
    expect(root).toBeTruthy();
    // Empty payload — the script tag must not leak text back either.
    expect(root?.innerHTML.toLowerCase()).not.toContain("<script");
  });

  it("strips the onerror handler from an injected <img> tag", async () => {
    const { screen, render } = await createDOM();
    await render(
      <MarkdownPreview body='<img src="x" onerror="alert(1)">' />,
    );

    const root = screen.querySelector('[data-testid="markdown-preview"]');
    expect(root).toBeTruthy();
    const html = root?.innerHTML ?? "";
    expect(html.toLowerCase()).not.toContain("onerror");
  });

  it("rejects a javascript: URL inside an <a> tag", async () => {
    const { screen, render } = await createDOM();
    await render(
      <MarkdownPreview body='<a href="javascript:alert(1)">click</a>' />,
    );

    const root = screen.querySelector('[data-testid="markdown-preview"]');
    const html = root?.innerHTML ?? "";
    expect(html.toLowerCase()).not.toContain("javascript:");
  });

  it("renders a heading from legitimate markdown", async () => {
    const { screen, render } = await createDOM();
    await render(<MarkdownPreview body="# My Header" />);

    const h1 = screen.querySelector("h1");
    expect(h1?.textContent?.trim()).toBe("My Header");
  });

  it("renders an empty body without throwing", async () => {
    const { screen, render } = await createDOM();
    await render(<MarkdownPreview body="" />);

    const root = screen.querySelector('[data-testid="markdown-preview"]');
    expect(root).toBeTruthy();
  });
});
