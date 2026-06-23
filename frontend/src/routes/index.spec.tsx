import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import Index from "./index";

// F-1 (spec §6.2): the landing route renders a brand mark, a
// tagline, and a single CTA anchor pointing to /organizations
// with an accessible name containing "Get started".

test("[routes/index]: renders a brand mark and tagline", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  // Brand mark — at least one <h1> with the product name.
  const h1s = screen.querySelectorAll("h1");
  expect(h1s.length).toBe(1);
  const brandText = h1s[0].textContent ?? "";
  expect(brandText.length).toBeGreaterThan(0);

  // Tagline — at least one <p> describing the product.
  const ps = screen.querySelectorAll("p");
  expect(ps.length).toBeGreaterThan(0);
  const tagline = ps[0].textContent ?? "";
  expect(tagline.length).toBeGreaterThan(0);
});

test("[routes/index]: renders a CTA anchor with href /organizations and accessible name containing 'Get started'", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  const cta = screen.querySelector('a[href="/organizations"]');
  expect(cta).not.toBeNull();
  const text = (cta as HTMLAnchorElement).textContent ?? "";
  expect(text).toContain("Get started");
});

test("[routes/index]: renders no carousel, no hero <img>", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  // Per UX-4 / UX-1, the landing page must be text-first.
  const images = screen.querySelectorAll("img");
  expect(images.length).toBe(0);
});
