/**
 * XSS regression tests for `renderSanitizedMarkdown`.
 *
 * Per spec `markdown-xss-prevention` REQ-04:
 * the sanitizer SHALL neutralize scripts, javascript URLs, event handlers,
 * iframe srcdoc, object data, and data:text/html URLs.
 */

import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { renderSanitizedMarkdown } from "./markdown";

describe("renderSanitizedMarkdown — XSS payload table (REQ-04)", () => {
  const payloadTable: ReadonlyArray<{
    readonly name: string;
    readonly input: string;
    readonly forbidden: ReadonlyArray<string>;
  }> = [
    {
      name: "script tag",
      input: "<script>alert(1)</script>",
      forbidden: ["<script"],
    },
    {
      name: "img with onerror handler",
      input: '<img src="x" onerror="alert(1)">',
      forbidden: ["onerror"],
    },
    {
      name: "javascript: URL",
      input: '<a href="javascript:alert(1)">click</a>',
      forbidden: ["javascript:"],
    },
    {
      name: "iframe srcdoc",
      input: '<iframe srcdoc="<script>alert(1)</script>"></iframe>',
      forbidden: ["<iframe"],
    },
    {
      name: "data:text/html URL",
      input: '<a href="data:text/html;base64,PHNjcmlwdD4=">click</a>',
      forbidden: ["data:text/html"],
    },
    {
      name: "obfuscated JaVaScRiPt: URL (case variant)",
      input: '<a href="JaVaScRiPt:alert(1)">click</a>',
      forbidden: [":alert(1)"],
    },
  ];

  for (const { name, input, forbidden } of payloadTable) {
    it(`strips ${name}`, () => {
      const html = renderSanitizedMarkdown(input);
      const lowered = html.toLowerCase();
      for (const fragment of forbidden) {
        expect(
          lowered.includes(fragment.toLowerCase()),
          `payload "${name}" leaked "${fragment}" into sanitizer output: ${html}`,
        ).toBe(false);
      }
    });
  }
});

describe("renderSanitizedMarkdown — happy path (REQ-01)", () => {
  it("preserves headings, lists, and code blocks from legitimate Markdown", () => {
    const md = "# Header\n\n- one\n- two\n\n```js\nconst x = 1;\n```";
    const html = renderSanitizedMarkdown(md);
    expect(html).toContain("<h1");
    expect(html).toContain("Header");
    expect(html).toContain("<ul");
    expect(html).toContain("<li");
    expect(html).toContain("<code");
  });

  it("returns an empty string for empty input", () => {
    expect(renderSanitizedMarkdown("")).toBe("");
  });
});

describe("markdown module — contract (REQ-03)", () => {
  it("does not contain the misleading 'no sanitization needed' claim", () => {
    const source = readFileSync(
      new URL("./markdown.ts", import.meta.url),
      "utf8",
    );
    expect(source).not.toMatch(/no sanitization library is needed/i);
    // And the file MUST expose a sanitizer function used by previews.
    expect(source).toMatch(/export function renderSanitizedMarkdown/);
  });
});
