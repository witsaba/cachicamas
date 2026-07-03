import { createDOM } from "@builder.io/qwik/testing";
import { test, expect } from "vitest";
import Index from "./index";

// F-1 (spec §6.2) + UX-4: the landing page is the front
// door of cachicamas.  Text-first, no decorative imagery,
// a single brand mark, a value pitch, a dual CTA, and the
// three framework sections (What you can track, The
// interface) that turn the home from a wireframe into a
// landing page.

test("[routes/index]: renders a single <h1> brand mark (F-1)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  // Exactly one <h1> on the page — the brand mark.  All
  // other headings are <h2>/<h3>.
  const h1s = screen.querySelectorAll("h1");
  expect(h1s.length).toBe(1);
  const brandText = (h1s[0].textContent ?? "").trim();
  expect(brandText.length).toBeGreaterThan(0);
});

test("[routes/index]: primary CTA points to /organizations/new with 'Get started' label (F-3, this iteration)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  // The first-run flow is home → /organizations/new.  The
  // /organizations list remains reachable by direct URL.
  const cta = screen.querySelector('a[data-cta="get-started"]');
  expect(cta).not.toBeNull();
  expect((cta as HTMLAnchorElement).getAttribute("href")).toBe(
    "/organizations/new",
  );
  expect((cta as HTMLAnchorElement).textContent ?? "").toContain(
    "Get started",
  );
});

test("[routes/index]: secondary CTA anchors to the interface section", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  const cta = screen.querySelector('a[data-cta="see-interface"]');
  expect(cta).not.toBeNull();
  expect((cta as HTMLAnchorElement).getAttribute("href")).toBe("#interface");
});

test("[routes/index]: renders 4 numbered feature cards (bento grid)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  const expected = [
    "organizations",
    "projects",
    "requirements",
    "milestones",
  ];
  for (const slug of expected) {
    const card = screen.querySelector(`[data-feature="${slug}"]`);
    expect(card, `expected a feature card for ${slug}`).not.toBeNull();
  }
});

test("[routes/index]: renders a CLI/code surface in the interface section", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  const pre = screen.querySelector('pre[data-surface="cli"]');
  expect(pre).not.toBeNull();
  const text = (pre as HTMLElement).textContent ?? "";
  expect(text).toContain("cachicamas");
  expect(text).toContain("org create");
  // Monospace rendered.
  const cls = (pre as HTMLElement).className;
  expect(cls).toMatch(/font-mono/);
});

test("[routes/index]: renders 3 section labels in monospace (agentic/framework vibe)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  for (const num of ["1.0", "2.0", "3.0"]) {
    const section = screen.querySelector(`[data-section="${num}"]`);
    expect(section, `expected section [${num}]`).not.toBeNull();
    // The section labels are short monospace tags.
    const text = (section as HTMLElement).textContent ?? "";
    expect(text).toContain(`[${num}]`);
    const cls = (section as HTMLElement).className;
    expect(cls).toMatch(/font-mono/);
  }
});

test("[routes/index]: renders a footer with a text-only signature (UX-4)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  const footer = screen.querySelector("[data-footer]");
  expect(footer).not.toBeNull();
  expect((footer as HTMLElement).textContent ?? "").toContain("cachicamas");
});

test("[routes/index]: no carousel, no hero <img>, no decorative imagery (UX-4)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);
  // Per UX-4 / UX-1, the landing page must be text-first.
  const images = screen.querySelectorAll("img");
  expect(images.length).toBe(0);
  // <picture> and <svg> count too — keep the surface text-only.
  const pictures = screen.querySelectorAll("picture");
  expect(pictures.length).toBe(0);
  const svgs = screen.querySelectorAll("svg");
  expect(svgs.length).toBe(0);
});
