import { describe, expect, it } from "vitest";
import { renderMarkdown } from "./markdown";

describe("renderMarkdown", () => {
  it("renders a heading", () => {
    const result = renderMarkdown("# Hello World");
    expect(result).toContain("<h1");
    expect(result).toContain("Hello World");
    expect(result).toContain("</h1>");
  });

  it("renders a paragraph", () => {
    const result = renderMarkdown("This is a paragraph.");
    expect(result).toContain("<p>");
    expect(result).toContain("This is a paragraph");
    expect(result).toContain("</p>");
  });

  it("renders bold and italic", () => {
    const result = renderMarkdown("**bold** and *italic*");
    expect(result).toContain("<strong>bold</strong>");
    expect(result).toContain("<em>italic</em>");
  });

  it("renders a code block", () => {
    const result = renderMarkdown("```\nconst x = 1;\n```");
    expect(result).toContain("<code");
    expect(result).toContain("const x = 1");
  });

  it("renders an unordered list", () => {
    const result = renderMarkdown("- item 1\n- item 2");
    expect(result).toContain("<ul>");
    expect(result).toContain("<li>");
    expect(result).toContain("item 1");
    expect(result).toContain("item 2");
  });

  it("renders an ordered list", () => {
    const result = renderMarkdown("1. first\n2. second");
    expect(result).toContain("<ol>");
    expect(result).toContain("<li>");
  });

  it("renders a blockquote", () => {
    const result = renderMarkdown("> this is a quote");
    expect(result).toContain("<blockquote>");
    expect(result).toContain("this is a quote");
  });

  it("renders a horizontal rule", () => {
    const result = renderMarkdown("---\nparagraph");
    expect(result).toContain("<hr");
  });

  it("returns empty string for empty input", () => {
    const result = renderMarkdown("");
    expect(result).toBe("");
  });

  it("is synchronous — does not return a Promise", () => {
    const result = renderMarkdown("# Test");
    expect(typeof result).toBe("string");
    expect(result).not.toBeInstanceOf(Promise);
  });
});
