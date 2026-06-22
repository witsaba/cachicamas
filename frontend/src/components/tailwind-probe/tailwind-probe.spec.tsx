import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import { TailwindProbe } from "./tailwind-probe";

const __dirname = dirname(fileURLToPath(import.meta.url));

test("[TailwindProbe Component]: renders the probe element with the test marker", async () => {
  const { screen, render } = await createDOM();
  await render(<TailwindProbe />);
  const el = screen.querySelector('[data-testid="tailwind-probe"]');
  expect(el).not.toBeNull();
});

test("[TailwindProbe Component]: preserves Tailwind utility classes through Qwik render", async () => {
  const { screen, render } = await createDOM();
  await render(<TailwindProbe />);
  const el = screen.querySelector('[data-testid="tailwind-probe"]') as HTMLElement;
  expect(el).not.toBeNull();
  // Each class should survive Qwik's render unchanged. If Tailwind's class
  // pass-through ever breaks, this test fails.
  expect(el.className).toContain("rounded");
  expect(el.className).toContain("bg-blue-100");
  expect(el.className).toContain("p-4");
  expect(el.className).toContain("text-red-500");
});

test("[TailwindProbe Component]: global.css wires the @import 'tailwindcss' directive", () => {
  // This is a config-level smoke test: it does NOT verify that Tailwind
  // actually processes the CSS (that needs Vite running), but it catches
  // the most common regression — someone deletes the @import line.
  const css = readFileSync(resolve(__dirname, "../../global.css"), "utf-8");
  expect(css).toContain('@import "tailwindcss"');
});
