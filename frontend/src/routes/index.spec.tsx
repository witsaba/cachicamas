import { createDOM } from "@builder.io/qwik/testing";
import { $, type QRL } from "@builder.io/qwik";
import { test, expect, vi } from "vitest";
import Index from "./index";

// No `vi.doMock` / `vi.resetModules` tests in this file any more.
// The PR-3 "SignInButton hidden when authenticated" scenario was
// removed together with the body SignInButton in UAT-1: there is
// no per-session variant to assert on the landing body, since the
// header (routes/layout.tsx) is the sole sign-in affordance now.
// If a future change re-introduces body-level sign-in UI, place
// any vi.doMock test LAST and expect mock-state leakage to
// following tests.


/**
 * Mock `routes/plugin@auth.ts` so the route's `useSignIn()` call doesn't
 * require the Qwik City request context (useContext(qc-l)) that the
 * vitest `createDOM()` harness does not set up. The SignInButton unit
 * spec covers the form shape in detail; this mock just unblocks the
 * landing-page rendering.
 */
vi.mock("~/routes/plugin@auth", () => {
  return {
    useSession: () => ({ value: null }),
    useSignIn: () => ({
      submit: $((_fd: FormData) => Promise.resolve()) as QRL<
        (formData: FormData) => unknown
      >,
      actionPath: "/auth/signin",
    }),
    useSignOut: () => ({
      submit: $((_fd: FormData) => Promise.resolve()) as QRL<
        (formData: FormData) => unknown
      >,
      actionPath: "/auth/signout",
    }),
    onRequest: () => Promise.resolve(),
  };
});

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

test("[routes/index]: primary CTA points to /ownboarding with 'Get started' label (F-3, 2026-07-06 ownboarding)", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  // The first-run flow is landing → /ownboarding (which redirects
  // to /home on success). The /organizations surface was removed
  // in the 2026-07-06 ownboarding change.
  const cta = screen.querySelector('a[data-testid="get-started"]');
  expect(cta).not.toBeNull();
  expect((cta as HTMLAnchorElement).getAttribute("href")).toBe(
    "/ownboarding",
  );
  expect((cta as HTMLAnchorElement).textContent ?? "").toContain(
    "Get started",
  );
});

test("[routes/index]: secondary CTA anchors to the interface section", async () => {
  const { screen, render } = await createDOM();
  await render(<Index />);

  const cta = screen.querySelector('a[data-testid="see-interface"]');
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

test("[routes/index]: UAT-1 — landing body has NO SignInButton (header is the sole sign-in CTA surface)", async () => {
  // UAT feedback (2026-07-04) flagged the body SignInButton as a
  // chrome duplicate of the header's SignInButton (rendered by
  // routes/layout.tsx for anonymous visitors). The fix removes
  // the body CTA entirely so the header is the single identity
  // affordance on the landing. Layout's SignInButton coverage
  // lives in routes/layout.spec.tsx under S-AS-010.
  const { screen, render } = await createDOM();
  await render(<Index />);
  const signInForm = screen.querySelector(
    'form[data-testid="sign-in-button"]',
  );
  expect(signInForm).toBeFalsy();
// The "Get started" primary CTA is still present — it points
  // at /organizations/new, not the sign-in flow.
  const cta = screen.querySelector('a[data-testid="get-started"]');
  expect(cta).toBeTruthy();
});
